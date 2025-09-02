package processing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/math"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var exePool = pool.New()

func ScriptToStop(ctx context.Context, runID int32, scriptRunID int32, pbStore *datapb.Store) (
	returnScriptRun *pb.ScriptRun, message string) {
	logger.Warnf(ctx, "ScriptRun to stopping(RunID:'%v'; ScriptsRun int64:'%v';)", runID, scriptRunID)

	scriptRunning, message := pbStore.GetScriptRunning(ctx, runID, scriptRunID)
	if message != "" {
		logger.Errorf(ctx, "Error get ScriptRunning: '%v'", message)

		return scriptRunning, message
	}

	scriptRunning.Status = pb.ScriptRun_STATUS_STOPPING

	message = pbStore.UpdatePbScriptRunInDB(ctx, runID, scriptRunning)
	if message != "" {
		logger.Errorf(ctx, "Error Update In DB ScriptRunning: '%v'", message)

		return scriptRunning, message
	}

	logger.Infof(ctx, "Creating a new command '%v'", models.CmdtypeTYPE_STOP_SCRIPT)

	cmdNewStopScr, err := pbStore.NewPbCmdStopScript(ctx,
		runID, undecided.ConvInt32toArrInt64(scriptRunID))
	if err != nil {
		message = fmt.Sprintf("Error created new command: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return scriptRunning, message
	}

	logger.Infof(ctx, "Created new command:'%v'", cmdNewStopScr)
	logger.Infof(ctx, "Script to stop message: '%v';", message)

	return scriptRunning, message
}

func ScenarioToRunning(ctx context.Context,
	scenarioID int32, percentageOfTarget int32, userName string, preferredUserName string, pbStore *datapb.Store) (
	returnRun *pb.Run, message string) {
	if percentageOfTarget == 0 {
		percentageOfTarget = 100
	}

	scenarioToRun, scenarioMessage := pbStore.GetScenario(ctx, scenarioID)

	scriptsToRun, er := GetAllEnabledScripts(ctx, scenarioID, pbStore)
	allScriptsMessage := ""
	if er != nil {
		allScriptsMessage = er.Error()
	}

	simpleScriptsToRun, er := pbStore.GetAllEnabledPBSimpleScripts(ctx, scenarioID)
	allSimpleScriptsMessage := ""
	if er != nil {
		logger.Errorf(ctx, "Error GetAllEnabledPBSimpleScripts: %v", er.Error())
		allSimpleScriptsMessage = er.Error()
	}

	exists := validationOfLaunchTags(ctx, scriptsToRun, simpleScriptsToRun, pbStore)

	logger.Infof(ctx, "Scenario run message: '%v'; All scripts message: '%v'; percentageOfTarget: '%v'",
		scenarioMessage, allScriptsMessage, percentageOfTarget)

	condition, conditionMessage := allConditionsAreMetToPrepareRunScenario(ctx, scenarioToRun, exists,
		scenarioMessage, allScriptsMessage, allSimpleScriptsMessage,
		len(scriptsToRun), len(simpleScriptsToRun), pbStore.GetDataStore())
	if !condition {
		logger.Errorf(ctx, "Conditions message: %v", conditionMessage)

		return returnRun, conditionMessage
	}

	mRun, err := pbStore.GetDataStore().CreateNewMRun(
		ctx,
		scenarioToRun.GetProjectId(),
		scenarioToRun.GetScenarioId(),
		percentageOfTarget,
		scenarioToRun.GetTitle(),
		userName,
		preferredUserName,
	)

	if err != nil {
		logger.Errorf(ctx, "Error create NewMRun: %v", err.Error())

		return nil, err.Error()
	}

	logger.Debugf(ctx, "mRun after Insert:'%+v'", mRun)

	returnRun, err = PreparingTheRunEntity(ctx,
		mRun,
		scriptsToRun,
		simpleScriptsToRun,
		pbStore.GetDataStore(),
	)
	if err != nil {
		message = fmt.Sprintf("Error preparing the Run entity: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Debugf(
		ctx,
		"Scenario Run ScenarioId:'%+v'; runID:'%+v'; Scripts.len:'%+v';  SimpleScriptsToRun.len:'%+v'; ReturnRun:'%+v';",
		scenarioToRun.GetScenarioId(),
		returnRun.GetRunId(),
		len(scriptsToRun),
		len(simpleScriptsToRun),
		returnRun,
	)

	logger.Infof(ctx, "Creating a new command")

	cmdNewRunningScenario, err := pbStore.NewPbCmdRunScenarioWithRpsAdjustment(ctx,
		returnRun.RunId, percentageOfTarget)
	if err != nil {
		message = fmt.Sprintf("Error created new command: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Infof(ctx, "Created new command: CommandType:'%v', CommandId:'%v', Status:'%v'",
		cmdNewRunningScenario.Type, cmdNewRunningScenario.CommandId, cmdNewRunningScenario.Status)

	return returnRun, conditionMessage
}

// validationOfLaunchTags проверка, у всех тегов есть доступные агенты
func validationOfLaunchTags(ctx context.Context,
	scripts []*pb.Script, simpleScripts []*pb.SimpleScript, pbStore *datapb.Store) bool {
	dataStore := pbStore.GetDataStore()
	for _, script := range scripts {
		exist, err := dataStore.GetExistEnabledMAgentsByTag(ctx, script.Tag)
		if err != nil {
			logger.Errorf(ctx, "validationOfLaunchTags check failed {%v -> %v}: %+v", script.Name, script.Tag, err)
			return false
		}
		if !exist {
			logger.Warnf(ctx, "validation tag {%v -> %v}: %+v", script.Tag, exist)
			return false
		}
	}

	for _, script := range simpleScripts {
		exist, err := dataStore.GetExistEnabledMAgentsByTag(ctx, script.Tag)
		if err != nil {
			logger.Errorf(ctx, "validationOfLaunchTags check failed {%v -> %v}: %+v", script.Name, script.Tag, err)
			return false
		}
		if !exist {
			logger.Warnf(ctx, "validation tag {%v -> %v}: %+v", script.Name, script.Tag, exist)
			return false
		}
	}

	return true
}

func ScenarioToStop(ctx context.Context, runID int32, dataStore *data.Store, pbStore *datapb.Store) (
	returnRun *pb.Run, message string) {
	logger.Infof(ctx, "Process of stopping the Run '%v'", runID)

	mRun, err := dataStore.GetMRunning(ctx, runID)
	if err == nil {
		// если он уже остановлен не изменяем статус и не создаем команду на остановку, просто возвращаем как есть
		if mRun.Status != models.EstatusSTATUS_STOPPED_UNSPECIFIED {
			err = setTheRunScenarioStatusStopping(ctx, mRun, dataStore)
			if err != nil {
				err = errors.Wrapf(err, "ScenarioToStop -> Scenario set status stopped error")
				logger.Errorf(ctx, err.Error())

				return returnRun, err.Error()
			}
			// Удаление лишней команды для обновления Рана
			dataStore.StopObservingForStatusRun(ctx, runID)
			logger.Infof(ctx, "Creating a new command to stop Running")

			_, err = pbStore.NewPbCmdStopScenario(ctx, runID)
			if err != nil {
				err = errors.Wrapf(err, "ScenarioToStop -> Error created new command")
				logger.Errorf(ctx, err.Error())

				return returnRun, err.Error()
			}
		} else {
			logger.Warnf(ctx, "Running '%v' is already stopped!", runID)
		}

		returnRun, err = conv.ModelToPBRun(ctx, mRun)
		if err != nil {
			err = errors.Wrapf(err, "ScenarioToStop -> ModelToPBRun error")
			logger.Errorf(ctx, err.Error())

			return returnRun, err.Error()
		}
	} else {
		logger.Debugf(ctx, "ScenarioToStop -> ScenarioStop ReturnRun: '%v'; Error: '%v'", mRun, err.Error())

		return nil, err.Error()
	}

	return returnRun, message
}

// ScenarioToAdjustment функция создания задачи для корректировки нагрузки запущенного сценария(runID) до указанного процента adjustmentOnPercent
func ScenarioToAdjustment(ctx context.Context,
	runID int32, adjustmentOnPercent int32, dataStore *data.Store, pbStore *datapb.Store) (message string) {
	logger.Warnf(ctx, "Adjustment RunScenario(RunID:'%v'; AdjustmentOnPercent:'%v';)",
		runID, adjustmentOnPercent)
	// не позволять запускать больше 10к процентов
	if adjustmentOnPercent > 10000 {
		adjustmentOnPercent = int32(10000)
		message = "There was an adjustment of the target percentage of the goal. because it is impossible to run more than 10,000 %"
	}

	mRun, err := dataStore.GetMRunning(ctx, runID)
	if err != nil {
		message = fmt.Sprintf("Error getting Running '%v' -> %v", runID, err.Error())
		logger.Warn(ctx, message)

		return message
	} else if mRun.Status != models.EstatusSTATUS_RUNNING {
		message = fmt.Sprintf("Interruption due to incorrect status '%v' RunID '%v'", mRun.Status, runID)
		logger.Warn(ctx, message)

		return message
	} else if mRun.PercentageOfTarget == adjustmentOnPercent {
		message = fmt.Sprintf(
			"Required level of '%v%%' for Run N'%v' has already been reached. The current percentage is '%v%%'",
			adjustmentOnPercent, mRun.RunID, mRun.PercentageOfTarget)
		logger.Warn(ctx, message)

		return message
	} else if adjustmentOnPercent == 0 {
		message = fmt.Sprintf(
			"Required level of '%v%%' for Run №'%v' requires stopping the run.",
			adjustmentOnPercent, mRun.RunID)
		logger.Warn(ctx, message)
		logger.Infof(ctx, "Creating a new command stop scenario")

		cmdNew, er := pbStore.NewPbCmdStopScenario(ctx, runID)
		if er != nil {
			message = fmt.Sprintf("Error created new command stop scenario: '%v'", er.Error())
			logger.Errorf(ctx, message)

			return message
		}

		logger.Infof(ctx, "Created new adjustment command:'%v'", cmdNew)

		return message
	} else if strings.Contains(mRun.Title, config.SmokeMarker) {
		message = "The adjustment does not support test runs"
		logger.Warn(ctx, message)

		return message
	}

	logger.Infof(ctx, "Creating a new command adjustment")

	cmdNew, err := pbStore.NewPbCmdAdjustment(ctx, runID, adjustmentOnPercent)
	if err != nil {
		message = fmt.Sprintf("Error created new command adjustment: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return message
	}

	logger.Infof(ctx, "Created new adjustment command:'%v'", cmdNew)

	return message
}

// setTheRunScenarioStatusStopping Выставление статуса "остановлен" сценарию только в случае, когда все скрипты в статусе "остановлен"
func setTheRunScenarioStatusStopping(ctx context.Context, mRun *models.Run, dataStore *data.Store) (err error) {
	mRun.Status = models.EstatusSTATUS_STOPPING
	logger.Infof(ctx, "Run stopping '%v'", mRun.RunID)
	_, err = dataStore.UpdateMRunningInTheDB(ctx, mRun)

	return err
}

// PbScriptRunArraySafe структура для ускорения подготовки запусков
type PbScriptRunArraySafe struct {
	mu    sync.RWMutex
	array []*pb.ScriptRun
}

func (sa *PbScriptRunArraySafe) Append(ctx context.Context, pbScriptRun *pb.ScriptRun) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	logger.Debugf(ctx, "---Append ScriptRun: %+v", pbScriptRun)
	sa.array = append(sa.array, pbScriptRun)
}

func (sa *PbScriptRunArraySafe) GetArray() []*pb.ScriptRun {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	return sa.array
}

func PreparingTheRunEntity(ctx context.Context, mRun *models.Run,
	scripts []*pb.Script, simpleScripts []*pb.SimpleScript, dataStore *data.Store) (
	returnRun *pb.Run, err error) {
	logger.Infof(ctx, "Preparing ScenarioID:'%+v'", mRun.ScenarioID)
	defer undecided.WarnTimer(ctx, fmt.Sprint("Preparing the Run Entity - ", mRun.ScenarioID))()

	var (
		pbScriptRunArr = &PbScriptRunArraySafe{array: make([]*pb.ScriptRun, 0)}
		errorMessage   = ""
		wg             = new(sync.WaitGroup)
		//freeAgent          *pb.Agent
	)

	for _, pbScript := range scripts {
		wg.Add(1)
		exePool.Go(func() {
			func(ctx context.Context, pbScript *pb.Script, wg *sync.WaitGroup, pbScriptRunArraySync *PbScriptRunArraySafe) {
				defer func(ctx context.Context, wg *sync.WaitGroup) {
					logger.Warnf(ctx, "scripts WaitGroup -1!")
					wg.Done()
				}(ctx, wg)

				if pbScript.Enabled { // добавляем только для выбранных(включенных) скриптов
					scriptTarget, er := strconv.ParseInt(pbScript.GetOptions().GetRps(), 10, 64)
					if er != nil {
						errorMessage = fmt.Sprintf("%v startingScript{Name:{%v}, RPS conversion error:{%v}}; ",
							errorMessage, pbScript.GetName(), er.Error())

						return
					}

					extendedScriptRun := &pb.ScriptRun{
						Status:      pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED,
						RunScriptId: math.GetRandomID32(ctx),
						Script:      pbScript,
						RunId:       mRun.RunID,
						Pid:         -1,
						//nolint:gosec
						Target: int32(scriptTarget),
						Metrics: &pb.Metrics{
							Rps:                    "0",
							RtMax:                  "0",
							Rt90P:                  "0",
							Rt95P:                  "0",
							Rt99P:                  "0",
							Failed:                 0,
							Vus:                    "0",
							Sent:                   "0",
							Received:               "0",
							VarietyTs:              0,
							ProgressBar:            "",
							Checks:                 0,
							FailedRate:             "0",
							ActiveVusCount:         "0",
							DroppedIterations:      "0",
							CurrentTestRunDuration: nil,
							HasStarted:             false,
							HasEnded:               false,
							FullIterationCount:     0,
							ExecutionStatus:        "Created",
						},
						TypeScriptRun: pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED,
					}

					logger.Debugf(
						ctx,
						"Added a new ExtendedScriptRun:{RunScriptId:{%v}, TypeScriptRun:{%v}, Script:{%+v}, Target:{%+v} }",
						extendedScriptRun.GetRunScriptId(),
						extendedScriptRun.GetTypeScriptRun(),
						extendedScriptRun.GetScript(),
						extendedScriptRun.GetTarget(),
					)

					pbScriptRunArraySync.Append(ctx, extendedScriptRun)
				} else {
					logger.Infof(ctx, "Disable pbScript:'%v'", pbScript.GetScriptId())
				}
			}(
				ctx,
				pbScript,
				wg,
				pbScriptRunArr,
			)
		})
	}

	for _, pbSimpleScript := range simpleScripts {
		wg.Add(1)
		exePool.Go(func() {
			func(ctx context.Context,
				pbSimpleScript *pb.SimpleScript, wg *sync.WaitGroup, pbScriptRunArraySync *PbScriptRunArraySafe) {
				defer func(ctx context.Context, wg *sync.WaitGroup) {
					logger.Warnf(ctx, "simpleScripts WaitGroup -1!")
					wg.Done()
				}(ctx, wg)

				if pbSimpleScript.Enabled { // добавляем только для включенных Simple скриптов
					simpleScriptTarget, er := strconv.ParseInt(pbSimpleScript.GetRps(), 10, 64)
					if er != nil {
						errorMessage = fmt.Sprintf("%v {startingScript{Name:{%v}, RPS conversion error:{%v}}}; ",
							errorMessage, pbSimpleScript.GetName(), er.Error())

						return
					}

					simpleScriptRun := &pb.ScriptRun{
						Status:       pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED,
						RunScriptId:  math.GetRandomID32(ctx),
						SimpleScript: pbSimpleScript,
						RunId:        mRun.RunID,
						Pid:          -1,
						//nolint:gosec
						Target: int32(simpleScriptTarget),
						Metrics: &pb.Metrics{
							Rps:                    "0",
							RtMax:                  "0",
							Rt90P:                  "0",
							Rt95P:                  "0",
							Rt99P:                  "0",
							Failed:                 0,
							Vus:                    "0",
							Sent:                   "0",
							Received:               "0",
							VarietyTs:              0,
							ProgressBar:            "",
							Checks:                 0,
							FailedRate:             "0",
							ActiveVusCount:         "0",
							DroppedIterations:      "0",
							CurrentTestRunDuration: nil,
							HasStarted:             false,
							HasEnded:               false,
							FullIterationCount:     0,
							ExecutionStatus:        "Created",
						},
						TypeScriptRun: pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE,
					}
					logger.Debugf(
						ctx,
						"Added a new simpleScriptRun:{RunScriptId:{%v}, TypeScriptRun:{%v}, SimpleScript:{%+v}, Target:{%+v} }",
						simpleScriptRun.GetRunScriptId(),
						simpleScriptRun.GetTypeScriptRun(),
						simpleScriptRun.GetSimpleScript(),
						simpleScriptRun.GetTarget(),
					)

					pbScriptRunArraySync.Append(ctx, simpleScriptRun)
				} else {
					logger.Infof(ctx, "Disable pbSimpleScript('%v':'%v')", pbSimpleScript.GetName(), pbSimpleScript.GetScriptId())
				}
			}(ctx, pbSimpleScript, wg, pbScriptRunArr)
		})
	}

	wg.Wait()

	scriptRuns := pbScriptRunArr.GetArray()
	logger.Infof(ctx, "pbScriptRuns count: %+v", len(scriptRuns))

	if len(scriptRuns) > 0 {
		for _, scriptRun := range scriptRuns {
			logger.Infof(ctx, "pbScriptRuns: %+v", scriptRun)
		}
	} else {
		mess := "error PreparingTheRunEntity: list scriptRuns is empty"
		logger.Errorf(ctx, mess)
		return nil, fmt.Errorf("%v", mess)
	}

	mRun.ScriptRuns = conv.PbToModelScriptRuns(ctx, scriptRuns)
	mRun.Status = models.EstatusSTATUS_PREPARED
	mRun.Info = fmt.Sprint(mRun.Info, errorMessage) // закидываем сообщение о том, с какими скриптами были проблемы

	mRun, err = dataStore.UpdateMRunningInTheDB(ctx, mRun)
	if err != nil {
		err = errors.Wrapf(err, "Error preparing the Run entity, UpdateMRunningInTheDB")
		logger.Warn(ctx, err)

		return returnRun, err
	}

	logger.Debugf(ctx, "mRun after Update:'%+v'", mRun)

	returnRun, err = conv.ModelToPBRun(ctx, mRun)
	if err != nil {
		message := fmt.Sprintf("Error runModel to db: '%v'", err.Error())
		logger.Error(ctx, message, err)
	}

	logger.Debugf(ctx, "Prepared Run:'%+v'", returnRun)

	return returnRun, err
}

// allConditionsAreMetToPrepareRunScenario проверяем все ли готово для запуска
func allConditionsAreMetToPrepareRunScenario(ctx context.Context, pbScenario *pb.Scenario, existsTags bool,
	getScenarioMessage string, getAllScriptsMessage string, getAllSimpleScriptsMessage string,
	scriptsLen int, simpleScriptsLen int,
	dataStore *data.Store) (
	yes bool, message string) {
	countAllReadyRunning, err := dataStore.GetCountActiveMRunning(ctx, 0, pbScenario.GetScenarioId())
	if err != nil {
		message = fmt.Sprintf("error getting EnabledMRunning '%+v'", err)
		logger.Errorf(ctx, message)

		return false, message
	}

	yes = pbScenario != nil &&
		getScenarioMessage == "" &&
		getAllScriptsMessage == "" &&
		getAllSimpleScriptsMessage == "" &&
		scriptsLen+simpleScriptsLen > 0 &&
		countAllReadyRunning == 0 &&
		existsTags
	if !yes {
		if scriptsLen+simpleScriptsLen < 1 {
			message = fmt.Sprintf(
				"There are no scripts to run. scriptsLen(%v), simpleScriptsLen(%v)", scriptsLen, simpleScriptsLen)
			logger.Warn(ctx, message)
		} else if getScenarioMessage != "" {
			message = fmt.Sprintf("GetScenario message: '%v'", getScenarioMessage)
			logger.Warn(ctx, message)
		} else if getAllScriptsMessage != "" {
			message = fmt.Sprintf("GetAllScripts message: '%v'", getAllScriptsMessage)
			logger.Warn(ctx, message)
		} else if pbScenario == nil {
			message = "pbScenario message: 'Scenario not found'"
			logger.Warn(ctx, message)
		} else if countAllReadyRunning > 0 {
			message = "Count Runs in scenario message: 'Scenario already in running'"
			logger.Warn(ctx, message)
		} else if !existsTags { // todo: Серега просил возвращать конкретный тег и скрипт, но нужно будет вытаскивать проверку отсюда, лучше реализовать отдельной ручкой
			message = "Count Runs in scenario message: There are no launch agents matching the selected tags"
			logger.Warn(ctx, message)
		}
	}

	logger.Infof(ctx, "Conditions: '%v'", yes)
	logger.Infof(
		ctx,
		"Conditions(Scenario not nil: '%v'; GetScenarioMessage: '%v'; GetAllScriptsMessage: '%v'; getAllSimpleScriptsMessage: '%v'; ScriptsLen: '%v'; simpleScriptsLen: '%v'; CountAllReadyRunning: '%v'; existsTags: '%v';)",
		pbScenario != nil,
		getScenarioMessage,
		getAllScriptsMessage,
		getAllSimpleScriptsMessage,
		scriptsLen,
		simpleScriptsLen,
		countAllReadyRunning,
		existsTags,
	)

	return yes, message
}

// ScriptToIncreaseByRPS Функция создания задачи cmd_processor-у для корректировки нагрузки одного запущенного скрипта в сценарии
//
// RPS конвертируется в процент от таргет-RPS скрипта
func ScriptToIncreaseByRPS(ctx context.Context,
	runID int32, scriptID int32, increaseByRPS int32, dataStore *data.Store, pbStore *datapb.Store) (message string) {
	logger.Warnf(ctx, "Script To Increase By RPS(RunID:'%v'; ScriptID:'%v'; IncreaseByRPS:'%v';)",
		runID, scriptID, increaseByRPS)
	// не позволять запускать больше 200к RPS
	if increaseByRPS > 200000 {
		increaseByRPS = int32(200000)
	}

	scriptsRun, message := pbStore.GetScriptRunsByScriptID(ctx, runID, scriptID)
	if scriptsRun == nil {
		message = fmt.Sprintf("Error getting Run'%v' ScriptRun'%v' -> %v", runID, scriptID, message)
		logger.Warnf(ctx, message)

		return message
	}
	// increasePercent := job.CalculateTheIncreaseOnPercent(ctx, increaseByRPS, scriptsRun[0].GetTarget())
	increasePercent := undecided.WhatPercentageRoundedToWhole(
		float64(increaseByRPS),
		float64(scriptsRun[0].GetTarget()),
	)

	mRun, err := dataStore.GetMRunning(ctx, runID)
	if err != nil {
		message = fmt.Sprintf("Error getting Running '%v' -> %v", runID, err.Error())
		logger.Warnf(ctx, message)

		return message
	} else if mRun.Status != models.EstatusSTATUS_RUNNING {
		message = fmt.Sprintf("Interruption due to incorrect status '%v' RunID '%v'", mRun.Status, runID)
		logger.Warnf(ctx, message)

		return message
	} else if strings.EqualFold(mRun.Title, "TestRun") {
		message = fmt.Sprintf("Interruption due to incorrect Run type. Test Run cannot be adjusted. RunID '%v'", runID)
		logger.Warnf(ctx, message)

		return message
	}

	logger.Infof(ctx, "Creating a new command script increase by RPS")
	// cmdNew, err := datapb.NewMCmdIncreaseScriptByRPS(ctx, runID, []int32{scriptID}, increaseByRPS, db)
	cmdNew, err := pbStore.NewPbCmdRunScriptWithRpsAdjustment(ctx,
		runID, undecided.ConvInt32toArrInt64(scriptID), int32(increasePercent))
	if err != nil {
		message = fmt.Sprintf("Error created new command increaseByRPS: '%v'", err.Error())
		logger.Errorf(ctx, message, err)

		return message
	}

	logger.Infof(ctx, "Created new increase command:'%v'", cmdNew)

	return message
}

//	SimpleScriptToIncreaseByRPS Функция создания задачи для корректировки нагрузки одного запущенного Simple скрипта в сценарии
//
// до указанного процента RPS-а
func SimpleScriptToIncreaseByRPS(ctx context.Context,
	runID int32, scriptID int32, increaseByRPS int32, dataStore *data.Store, pbStore *datapb.Store) (message string) {
	logger.Warnf(ctx, "SimpleScript To Increase By RPS(RunID:'%v'; SimpleScriptID:'%v'; IncreaseByRPS:'%v';)",
		runID, scriptID, increaseByRPS)
	// не позволять запускать больше 200к RPS
	if increaseByRPS > 200000 {
		increaseByRPS = int32(200000)
	}

	scriptsRun, message := pbStore.GetScriptRunsByScriptID(ctx, runID, scriptID)
	if scriptsRun == nil {
		message = fmt.Sprintf("Error getting Run'%v' ScriptRun'%v' -> %v", runID, scriptID, message)
		logger.Warnf(ctx, message)

		return message
	}

	increasePercent := undecided.WhatPercentageRoundedToWhole(
		float64(increaseByRPS),
		float64(scriptsRun[0].GetTarget()),
	)

	mRun, err := dataStore.GetMRunning(ctx, runID)
	if err != nil {
		message = fmt.Sprintf("Error getting Running '%v' -> %v", runID, err.Error())
		logger.Warnf(ctx, message)

		return message
	} else if mRun.Status != models.EstatusSTATUS_RUNNING {
		message = fmt.Sprintf("Interruption due to incorrect status '%v' RunID '%v'", mRun.Status, runID)
		logger.Warnf(ctx, message)

		return message
	} else if strings.EqualFold(mRun.Title, "TestRun") {
		message = fmt.Sprintf("Interruption due to incorrect Run type. Test Run cannot be adjusted. RunID '%v'", runID)
		logger.Warnf(ctx, message)

		return message
	}

	logger.Infof(ctx, "Creating a new command SimpleScript increase by RPS")

	cmdNew, err := pbStore.NewPbCmdRunSimpleScriptWithRpsAdjustment(ctx,
		runID, undecided.ConvInt32toArrInt64(scriptID), int32(increasePercent))
	if err != nil {
		message = fmt.Sprintf("Error created new command SimpleIncreaseByRPS: '%v'", err.Error())
		logger.Errorf(ctx, message, err)

		return message
	}

	logger.Infof(ctx, "Created new increase command:'%v'", cmdNew)

	return message
}

// ScriptToTestRunning Функция тестового запуска одного скрипта из сценария
func ScriptToTestRunning(ctx context.Context,
	scriptID int32, dataStore *data.Store, pbStore *datapb.Store) (
	returnRun *pb.Run, message string) {
	pbScript, err := pbStore.GetPbScript(ctx, scriptID)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch mScript")
		logger.Errorf(ctx, err.Error())

		return nil, err.Error()
	}
	countAllReadyRunning, err := dataStore.GetCountActiveMRunning(ctx, 0, pbScript.GetScenarioId())
	if err != nil {
		logger.Errorf(ctx, "pbScript run error getting EnabledMRunning '%+v'", err)
	}

	if countAllReadyRunning > 0 {
		message = fmt.Sprintf("the Scenario{%v} is already running", pbScript.GetScenarioId())
		logger.Errorf(ctx, message)

		return nil, message
	}

	// 100 % указывается для того что бы запустить скрипт только на всю указанную нагрузку в скриптране
	//(а в самом скриптране выставляется меньше)
	mRun, err := dataStore.CreateNewMRun(ctx,
		pbScript.GetProjectId(), pbScript.GetScenarioId(), 100,
		fmt.Sprintf("%v_%v", config.SmokeMarker, pbScript.Name), "System User", "")
	if err != nil {
		logger.Errorf(ctx, "Error create NewMRun: %v", err.Error())

		return nil, err.Error()
	}

	pbScript.GetOptions().Rps = "2"
	pbScript.GetOptions().Duration = config.SmokeDuration
	pbScript.GetOptions().Steps = "1"
	pbScript.Enabled = true
	logger.Debugf(ctx, "mRun after Insert:'%+v'", mRun)

	returnRun, er := PreparingTheRunEntity(ctx, mRun,
		[]*pb.Script{pbScript},
		[]*pb.SimpleScript{},
		dataStore)
	if er != nil {
		message = fmt.Sprintf("Error preparing the Run entity: '%v'", er.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Debugf(ctx, "ScriptTestRun{ScenarioId:'%+v'; runID:'%+v'; Scripts.len:'%+v'; ReturnRun:'%+v';}",
		pbScript.GetScenarioId(), returnRun.GetRunId(), 1, returnRun)

	logger.Infof(ctx, "Creating a new command")

	cmdNewSmokeSRunningScenario, err := pbStore.NewPbCmdRunScenarioWithRpsAdjustment(ctx,
		returnRun.RunId, 100)
	if err != nil {
		message = fmt.Sprintf("Error created new command: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Infof(ctx, "Created new command: CommandType:'%v', CommandId:'%v', Status:'%v'",
		cmdNewSmokeSRunningScenario.Type, cmdNewSmokeSRunningScenario.CommandId, cmdNewSmokeSRunningScenario.Status)

	return returnRun, ""
}

// SimpleScriptToTestRunning Функция тестового запуска одного simple скрипта из сценария
func SimpleScriptToTestRunning(ctx context.Context,
	simpleScriptID int32, dataStore *data.Store, pbStore *datapb.Store) (
	returnRun *pb.Run, message string) {
	pbSScript, err := pbStore.GetPBSimpleScript(ctx, simpleScriptID)
	if err != nil {
		err = errors.Wrapf(err, "Error getting pbSScript")
		logger.Errorf(ctx, err.Error())

		return nil, err.Error()
	}
	countAllReadyRunning, err := dataStore.GetCountActiveMRunning(ctx, 0, pbSScript.GetScenarioId())
	if err != nil {
		logger.Errorf(ctx, "pbSScript run error getting EnabledMRunning '%+v'", err)
	}

	if countAllReadyRunning > 0 {
		message = fmt.Sprintf("the Scenario{%v} is already running", pbSScript.GetScenarioId())
		logger.Errorf(ctx, message)

		return nil, message
	}
	mRun, err := dataStore.CreateNewMRun(ctx,
		pbSScript.GetProjectId(), pbSScript.GetScenarioId(), 100,
		fmt.Sprintf("%v_%v", config.SmokeMarker, pbSScript.Name), "System User", "")
	if err != nil {
		logger.Errorf(ctx, "Error create NewMRun: %v", err.Error())
		return nil, err.Error()
	}

	pbSScript.Rps = "2"
	pbSScript.Duration = config.SmokeDuration
	pbSScript.Steps = "1"
	pbSScript.Enabled = true

	logger.Debugf(ctx, "mRun after Insert:'%+v'", mRun)

	returnRun, er := PreparingTheRunEntity(ctx, mRun,
		[]*pb.Script{},
		[]*pb.SimpleScript{pbSScript},
		dataStore)
	if er != nil {
		message = fmt.Sprintf("Error preparing the Run entity: '%v'", er.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Debugf(ctx, "ScriptTestRun{ScenarioId:'%+v'; runID:'%+v'; Scripts.len:'%+v'; ReturnRun:'%+v';}",
		pbSScript.GetScenarioId(), returnRun.GetRunId(), 1, returnRun)

	logger.Infof(ctx, "Creating a new command")

	cmdNewSmokeSSRunningScenario, err := pbStore.NewPbCmdRunScenarioWithRpsAdjustment(ctx,
		returnRun.RunId, 100)
	if err != nil {
		message = fmt.Sprintf("Error created new command: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnRun, message
	}

	logger.Infof(ctx, "Created new command: CommandType:'%v', CommandId:'%v', Status:'%v'",
		cmdNewSmokeSSRunningScenario.Type, cmdNewSmokeSSRunningScenario.CommandId, cmdNewSmokeSSRunningScenario.Status)

	return returnRun, message
}

// TimeRange диапазон времени выполнения Run
func TimeRange(ctx context.Context, runID int32, store *datapb.Store) (*pb.DataRange, error) {
	running, mess := store.GetRunning(ctx, runID)
	if mess != "" {
		logger.Warnf(ctx, "The error of getting a Run:", mess)
		return nil, fmt.Errorf("%v", mess)
	}
	dr := &pb.DataRange{
		From: running.CreatedAt,
		To:   running.UpdatedAt,
	}
	return dr, nil
}

func CurrentRPS(ctx context.Context, runID int32, store *datapb.Store) (int64, error) {
	running, mess := store.GetRunning(ctx, runID)
	if mess != "" {
		logger.Warnf(ctx, "The error of getting a Run:", mess)
		return -1, fmt.Errorf("%v", mess)
	}
	rps := int64(0)
	if running.Status != pb.Run_STATUS_RUNNING {
		return rps, nil
	}
	for _, scriptRun := range running.ScriptRuns {
		if scriptRun.Status == pb.ScriptRun_STATUS_RUNNING {
			i, err := strconv.ParseInt(scriptRun.Metrics.Rps, 10, 64)
			if err != nil {
				continue
			}
			rps = rps + i
		}
	}
	return rps, nil
}

func ScriptRunsByScriptName(
	ctx context.Context,
	runID int32,
	scriptName string,
	store *datapb.Store,
) ([]*pb.ScriptRun, error) {
	running, mess := store.GetRunning(ctx, runID)
	if mess != "" {
		logger.Warnf(ctx, "Error of getting a Run:", mess)
		return nil, fmt.Errorf("%v", mess)
	}
	scriptRuns := make([]*pb.ScriptRun, 0)
	for _, scriptRun := range running.ScriptRuns {
		scriptRunName := ""
		switch scriptRun.TypeScriptRun {
		case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
			scriptRunName = scriptRun.GetScript().Name
		case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
			scriptRunName = scriptRun.GetSimpleScript().Name
		default:
			logger.Warnf(ctx, "Unknown script type: %+v", scriptRun.TypeScriptRun)
			continue
		}
		if scriptRunName == scriptName {
			scriptRuns = append(scriptRuns, scriptRun)
		}
	}
	return scriptRuns, nil
}

func ScriptRunsByScriptRunID(
	ctx context.Context,
	runID int32,
	scriptRunID int32,
	store *datapb.Store,
) (*pb.ScriptRun, error) {
	running, mess := store.GetRunning(ctx, runID)
	if mess != "" {
		logger.Warnf(ctx, "Error of getting a Run:", mess)
		return nil, fmt.Errorf("%v", mess)
	}

	for _, scriptRun := range running.ScriptRuns {
		if scriptRun.RunScriptId == scriptRunID {
			return scriptRun, nil
		}
	}
	return nil, fmt.Errorf("no script with id %v was found", scriptRunID)
}

func IsChangeLoadLevel(ctx context.Context, runID int32, store *datapb.Store) (*pb.IsChangeLoadLevelResponse, error) {
	pbRun, mess := store.GetRunning(ctx, runID)
	if mess != "" {
		logger.Warnf(ctx, "Error getting mRun: %+v", mess)

		return nil, status.Errorf(codes.InvalidArgument, "%v", mess)
	}
	if pbRun == nil {
		mess = fmt.Sprintf("Error getting pbRun == %+v", pbRun)
		logger.Warnf(ctx, mess)

		return nil, status.Errorf(codes.DataLoss, "%v", mess)
	}
	resp := &pb.IsChangeLoadLevelResponse{
		CurrentLevel: pbRun.PercentageOfTarget,
		NextLevel:    int32(0),
		CanChange:    true,
		RunStatus:    pbRun.Status,
	}

	command, err := store.GetDataStore().GetRunCommand(ctx, runID)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result set") {
			logger.Warnf(ctx, "GetRunCommand error: %+v", err)
			return resp, status.Errorf(codes.Internal, "%v", mess)
		}
	}

	switch pbRun.Status {
	case pb.Run_STATUS_PREPARED:
		resp.CanChange = false
		resp.NextLevel = pbRun.PercentageOfTarget
	case pb.Run_STATUS_STOPPING, pb.Run_STATUS_STOPPED_UNSPECIFIED:
		resp.CanChange = false
	case pb.Run_STATUS_RUNNING:
		if command == nil {
			mess = fmt.Sprintf("Error getting RunCommand == %+v", command)
			logger.Warnf(ctx, mess)

			return resp, status.Errorf(codes.DataLoss, "%v", mess)
		}
		switch command.Type {
		case models.CmdtypeTYPE_UPDATE, models.CmdtypeTYPE_INCREASE:
			logger.Debugf(ctx, "cmd type update and increase scip")
		default:
			switch command.Status {
			case models.CmdstatusSTATUS_CREATED_UNSPECIFIED, models.CmdstatusSTATUS_PROCESSED:
				resp.CanChange = false
				resp.NextLevel = command.PercentageOfTarget.Int32
			default:
				logger.Warnf(ctx, "default command.Status!")
			}
		}

	default:
		logger.Warnf(ctx, "default Run.Status!")
	}

	return resp, nil
}

func checkExistActiveRun(ctx context.Context, store *datapb.Store, scenarioID int32) int64 {
	count, getCountErr := store.GetDataStore().GetCountActiveMRunning(ctx, 0, scenarioID)
	if getCountErr != nil {
		logger.Errorf(ctx, "error getting count active runs by scenarioID: %+v", getCountErr)
		return 0
	}
	return count
}
