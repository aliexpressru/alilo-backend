package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	agentapi "github.com/aliexpressru/alilo-backend/pkg/clients/pb/qa/loadtesting/alilo/agent-v2/agent/api/qa/loadtesting/alilo/agent/v1"
	agentapi2 "github.com/aliexpressru/alilo-backend/pkg/model/agentapi"
	"github.com/aliexpressru/alilo-backend/pkg/util/httputil"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	math2 "github.com/aliexpressru/alilo-backend/pkg/util/math"
	"github.com/aliexpressru/alilo-backend/pkg/util/promutil"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
)

const (
	keyEnv    string = "-e"
	keyOutput string = "--out"
	keyDebug  string = "--http-debug=full"

	markerStop  string = "Stop"
	markerStart string = "Start"

	optionSteps            optionVal = "STEPS="
	optionRps              optionVal = "RPS="
	optionDuration         optionVal = "DURATION="
	optionAPI              optionVal = "API_URL="
	optionAmmo             optionVal = "AMMO_URL="
	optionPrometheusURL    optionVal = "K6_PROMETHEUS_REMOTE_URL="
	optionPrometheus       optionVal = "prometheus=port="
	optionOutputPrometheus optionVal = "output-prometheus-remote"

	executionStatusInterrupted = "Interrupted"
	executionStatusEnded       = "Ended"
)

type optionVal string

// decreaseScriptRun Структура для функции понижения нагрузки
type decreaseScriptRun = struct {
	ScriptRunID int32
	CurRps      int32
}

func (p *ProcessorPool) prepareTaskToRun(ctx context.Context, scriptRun *pb.ScriptRun, scenarioTitle string) (
	task *agentapi.Task) {
	logger.Debugf(ctx, "Script to run: '%v'", scriptRun)

	switch scriptRun.TypeScriptRun {
	case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
		//params := []string{
		//	keyEnv, p.optionToString(optionSteps, scriptRun.Script.GetOptions().GetSteps()),
		//	keyEnv, p.optionToString(optionRps, scriptRun.Script.GetOptions().GetRps()),
		//	keyEnv, p.optionToString(optionDuration, scriptRun.Script.GetOptions().GetDuration()),
		//	keyEnv, p.optionToString(optionAPI, scriptRun.Script.GetBaseUrl()),
		//	keyEnv, p.optionToString(optionAmmo, scriptRun.Script.GetAmmoId()),
		//}
		//
		//logger.Infof(ctx, "Params: '%+v'", params)
		task = &agentapi.Task{
			ScriptUrl:     scriptRun.GetScript().ScriptFile,
			ScenarioTitle: scenarioTitle,
			//Params:        params,
			ScriptTitle: scriptRun.Script.Name,
			ProjectId:   int64(scriptRun.GetScript().ProjectId),
			ScenarioId:  int64(scriptRun.GetScript().ScenarioId),
			ScriptId:    int64(scriptRun.GetScript().ScriptId),
			RunId:       int64(scriptRun.RunId),
			ScriptRunId: int64(scriptRun.RunScriptId),
			Envs: &agentapi.Envs{
				Steps:         scriptRun.Script.GetOptions().GetSteps(),
				Rps:           scriptRun.Script.GetOptions().GetRps(),
				Duration:      scriptRun.Script.GetOptions().GetDuration(),
				ApiUrl:        scriptRun.Script.GetBaseUrl(),
				AmmoUrl:       scriptRun.Script.GetAmmoId(),
				AdditionalEnv: scriptRun.Script.GetAdditionalEnv(),
			},
		}
	case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
		task = &agentapi.Task{
			ScriptUrl:     scriptRun.GetSimpleScript().GetScriptFileUrl(),
			ScenarioTitle: scenarioTitle,
			//Params:        params,
			ScriptTitle: scriptRun.GetSimpleScript().Name,
			ProjectId:   int64(scriptRun.GetSimpleScript().ProjectId),
			ScenarioId:  int64(scriptRun.GetSimpleScript().ScenarioId),
			ScriptId:    int64(scriptRun.GetSimpleScript().ScriptId),
			RunId:       int64(scriptRun.RunId),
			ScriptRunId: int64(scriptRun.RunScriptId),
			Envs: &agentapi.Envs{
				Steps:    scriptRun.GetSimpleScript().GetSteps(),
				Rps:      scriptRun.GetSimpleScript().GetRps(),
				Duration: scriptRun.GetSimpleScript().GetDuration(),
				ApiUrl: fmt.Sprintf(
					"%v://%v",
					scriptRun.GetSimpleScript().GetScheme(),
					scriptRun.GetSimpleScript().GetPath(),
				),
				AmmoUrl:       scriptRun.GetSimpleScript().GetAmmoUrl(),
				AdditionalEnv: scriptRun.GetSimpleScript().GetAdditionalEnv(),
			},
		}
	}

	if config.CheckValueSendMetricsToPromet(
		ctx,
	) { // проверка на необходимость отправки метрики скриптов в прометеус через плагин
		task.Envs.Out = []string{string(optionPrometheus)}
	}
	if p.checkSmoke(scenarioTitle) {
		//task.Params = append(task.Params, keyDebug)
		task.Envs.HttpDebug = keyDebug
	}

	logger.Infof(ctx, "Task: '%+v'", task)

	return task
}

func (p *ProcessorPool) checkSmoke(scenarioTitle string) bool {
	return strings.Contains(scenarioTitle, config.SmokeMarker)
}

func (p *ProcessorPool) checkProd(pbRun *pb.Run) bool {
	for _, scriptRun := range pbRun.ScriptRuns {
		for _, tag := range scriptRun.Agent.Tags {
			if strings.Contains(tag, "prod") {
				return true
			}
		}
	}
	return false
}

func (p *ProcessorPool) observingForStatusRun(ctx context.Context, cmd *models.Command, pbRun *pb.Run) bool {
	if pbRun.GetStatus() == pb.Run_STATUS_RUNNING {
		logger.Infof(ctx, "Run is running, continue to monitor...")

		_, err := p.db.NewMCmdUpdate(ctx, pbRun.GetRunId())
		if err != nil {
			logger.Errorf(ctx, "Error created new command: '%v'", err)
			cmd.ErrorDescription = err.Error()

			return false
		}
	} else {
		logger.Errorf(ctx, "failed observing run the scenario: %v -> %v -> %v",
			pbRun.GetStatus(), pbRun.Title, cmd.ErrorDescription)

		return false
	}

	return true
}

func (p *ProcessorPool) updateStatusScriptRun(ctx context.Context,
	cmdUpdating *models.Command, scriptRun *pb.ScriptRun, testName string, waitGroup *sync.WaitGroup) *pb.ScriptRun {
	defer func(waitGroup *sync.WaitGroup) {
		if err := recover(); err != nil {
			logger.Errorf(
				ctx,
				"updateStatusScriptRun failed: '%+v' testName:'%v', scriptRun.RunScriptId:'%v', scriptRun.TypeScriptRun:'%v', Stack: %s",
				err,
				testName,
				scriptRun.RunScriptId,
				scriptRun.TypeScriptRun,
				string(debug.Stack()),
			)
			mess := fmt.Sprintf("%s: %s", "scriptRun status is", scriptRun.Status)
			if !strings.Contains(scriptRun.Info, mess) {
				scriptRun.Info = fmt.Sprintf("%s recover{%s};", scriptRun.Info, mess)
			}
		}

		waitGroup.Done()
	}(waitGroup)

	defer undecided.InfoTimer(
		ctx,
		fmt.Sprintf("updateStatusScriptRun %v/%v", cmdUpdating.RunID, scriptRun.RunScriptId),
	)()

	if scriptRun.Status != pb.ScriptRun_STATUS_RUNNING {
		logger.Warnf(ctx, "update RunScriptId:'%v'; RunScriptPid:'%v'; RunScript status:'%v';",
			scriptRun.RunScriptId, scriptRun.Pid, scriptRun.Status)
		mess := fmt.Sprintf("%s{%s}", "scriptRun status is ", scriptRun.Status)
		if !strings.Contains(scriptRun.Info, mess) {
			scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, mess)
		}

		return scriptRun
	}

	agentHost := scriptRun.GetAgent()
	// runScriptStatusRq := p.prepareRequest(ctx, scriptRun)

	logger.Infof(ctx, "GetClient(%s:%s)", agentHost.HostName, agentHost.Port)

	agent, err := p.agentManager.GetClientPB(ctx, agentHost)
	if err != nil {
		logger.Errorf(ctx, "execute GetClientPB error: '%v' ", err)
		err = errors.Wrapf(err, "execute GetClient: '%v':'%v'", scriptRun.GetRunId(), agentHost.HostName)

		errorDescription := strings.Builder{}
		errorDescription.WriteString(cmdUpdating.ErrorDescription)
		errorDescription.WriteString(" RunScriptId:")
		errorDescription.WriteString(strconv.FormatInt(int64(scriptRun.GetRunScriptId()), 10))
		errorDescription.WriteString(" ERROR:")
		errorDescription.WriteString(err.Error())
		errorDescription.WriteString("; ")
		p.db.UpdateStatusMCommand(ctx, cmdUpdating, cmdUpdating.Status, errorDescription.String())

		mess := "mistake getting AgentClient"
		if !strings.Contains(scriptRun.Info, mess) {
			scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, mess)
		}

		return scriptRun
	}

	//rs, err := agent.GetStatus(ctx, &agentapi.GetStatusRequest{Pid: scriptRun.Pid})
	bytes, err := httputil.Post(
		ctx,
		fmt.Sprintf("http://%s:%s/api/v1/getStatus", scriptRun.Agent.HostName, scriptRun.Agent.Port),
		"application/json",
		map[string]string{},
		&agentapi2.GetStatusRequest{Pid: scriptRun.Pid})
	if err != nil {
		logger.Errorf(ctx, "HTTP Start request failed: %v", err)
		return nil
	}

	rs := &agentapi2.ResponseGetStatus{}

	err = json.Unmarshal(bytes, rs)
	if err != nil {
		logger.Errorf(ctx, "HTTP get status request failed: %v", err)

		return nil
	}
	if err != nil && !strings.Contains(err.Error(), "no such test run") {
		logger.Errorf(ctx, "executeRequest GetStatus ToAgentAndReturnResponse error: '%v' ", err)
		err = errors.Wrapf(
			err,
			"executeRequestToAgentAndReturnResponse GetStatus: '%v':'%v'",
			scriptRun.GetRunId(),
			agentHost.HostName,
		)

		errorDescription := strings.Builder{}
		errorDescription.WriteString(cmdUpdating.ErrorDescription)
		errorDescription.WriteString(" RunScriptId:")
		errorDescription.WriteString(strconv.FormatInt(int64(scriptRun.GetRunScriptId()), 10))
		errorDescription.WriteString(" ERROR:")
		errorDescription.WriteString(err.Error())
		errorDescription.WriteString("; ")
		p.db.UpdateStatusMCommand(ctx, cmdUpdating, cmdUpdating.Status, errorDescription.String())
		mess := "mistake getting ScriptRun status"
		if !strings.Contains(scriptRun.Info, mess) {
			scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, mess)
			scriptRun.Metrics.ExecutionStatus = executionStatusInterrupted
		}

		if strings.Contains(err.Error(), "could not initialize") {
			scriptRun.Status = pb.ScriptRun_STATUS_FAILED
		}

		return scriptRun
	}

	logger.Infof(ctx, "GetStatus: Agent{%v}; RunScriptId{%v}",
		agentHost.HostName, scriptRun.RunScriptId)
	logger.Debugf(ctx, "GetStatus{Agent{%v}; RunScriptId{%v} Response{%+v}",
		agentHost.HostName, scriptRun.RunScriptId, rs)
	if err != nil || rs == nil {
		if strings.Contains(err.Error(), "no such test run") {
			scriptRun.Status = pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED
			if scriptRun.Metrics.ExecutionStatus != executionStatusInterrupted {
				scriptRun.Metrics.ExecutionStatus = executionStatusEnded
			}
		} else {
			logger.Errorf(ctx, "error getting status scriptRun %v/%v", cmdUpdating.RunID, scriptRun.RunScriptId)
			if !strings.Contains(scriptRun.Info, err.Error()) {
				scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, err.Error())
			}
			scriptRun.Status = pb.ScriptRun_STATUS_FAILED
		}
		return scriptRun
	}
	scriptRun.Status = pb.ScriptRun_STATUS_RUNNING

	scriptName := ""
	scriptProjectID := int32(-1)
	scriptScenarioID := int32(-1)

	switch scriptRun.TypeScriptRun {
	case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
		scriptName = scriptRun.GetScript().Name
		scriptProjectID = scriptRun.GetScript().ProjectId
		scriptScenarioID = scriptRun.GetScript().ScenarioId
	case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
		scriptName = scriptRun.GetSimpleScript().Name
		scriptProjectID = scriptRun.GetSimpleScript().ProjectId
		scriptScenarioID = scriptRun.GetSimpleScript().ScenarioId
	default:
		logger.Warnf(ctx, "Unknown script type: %+v", rs)
	}

	logger.Debugf(ctx, "---Metrics: Run '%v'('%v') scriptRun'%v'('%v') curData: '%+v'",
		testName, scriptRun.GetRunId(), scriptName, scriptRun.GetRunScriptId(), rs.Metrics)

	failedValue, err := strconv.ParseInt(rs.Metrics.Failed, 10, 64)
	if err != nil {
		logger.Errorf(ctx, "failed to parse failed value: %v", err)
	}

	if rs.Metrics != nil {
		scriptRun.Metrics = &pb.Metrics{
			Rps:               rs.Metrics.Rps,
			Rt90P:             rs.Metrics.Rt90P,
			Rt95P:             rs.Metrics.Rt95P,
			RtMax:             rs.Metrics.RtMax,
			Rt99P:             rs.Metrics.Rt99P,
			Failed:            failedValue,
			Vus:               rs.Metrics.Vus,
			Sent:              rs.Metrics.Sent,
			Received:          rs.Metrics.Received,
			VarietyTs:         0,
			Checks:            0,
			ProgressBar:       "",
			FailedRate:        "",
			ActiveVusCount:    "",
			DroppedIterations: "",
			CurrentTestRunDuration: &durationpb.Duration{
				Seconds: 0,
				Nanos:   0,
			},
			HasStarted:         true,
			HasEnded:           true,
			FullIterationCount: 0,
			ExecutionStatus:    "",
		}

		maksURLsInOneScriptRun := config.Get(ctx).MaksURLsInOneScriptRun
		logger.Infof(ctx, "CountURLsInRun{name:%v, ID: %v, VarietyTs: %v/%v}",
			scriptName, scriptRun.RunScriptId, scriptRun.Metrics.VarietyTs, maksURLsInOneScriptRun)

		// Не должно быть больше 'MaksURLsInOneScriptRun' URL в метриках у одного скрипт-рана
		if scriptRun.Metrics.VarietyTs > maksURLsInOneScriptRun {
			logger.Warnf(ctx, "Script stops{%v > %v}", scriptRun.Metrics.VarietyTs, maksURLsInOneScriptRun)
			mess := ""
			logger.Warnf(ctx, "Stop scriptRun{scriptName:%v, RunScriptId:%v, Pid:%v}",
				scriptName, scriptRun.RunScriptId, scriptRun.Pid)
			_, stopErr := agent.Stop(ctx, &agentapi.StopRequest{Pid: scriptRun.Pid})
			if stopErr != nil {
				stopMes := fmt.Sprintf("error script{%v/%v} Stop {%+v}",
					scriptRun.RunId, scriptRun.RunScriptId, stopErr)
				logger.Errorf(ctx, stopMes)
				p.db.UpdateStatusMCommand(ctx, cmdUpdating, cmdUpdating.Status, stopMes)
				if !strings.Contains(scriptRun.Info, stopMes) {
					scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, stopMes)
				}
				scriptRun.Metrics.ExecutionStatus = executionStatusInterrupted
			}

			if scriptRun.Metrics.ExecutionStatus != executionStatusInterrupted {
				scriptRun.Metrics.ExecutionStatus = executionStatusEnded
			}

			mess = "there are too many TS metrics, please redefine the 'url' and 'name' fields in the script"
			if !strings.Contains(scriptRun.Info, mess) {
				scriptRun.Info = fmt.Sprintf("%s %s; ", scriptRun.Info, mess)
				logger.Errorf(ctx, "%v:%v -> %v Need to fix the script(too many TS metrics)",
					scriptProjectID, scriptScenarioID, scriptName)
			}
		}
	}

	return scriptRun
}

func (p *ProcessorPool) updatingRun(ctx context.Context, cmdUpdating *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "UpdatingRun failed: '%+v'", err)
		}
	}()

	logger.Infof(ctx, "CommandID:'%v', CommandStatus: '%v', RunID: '%v'",
		cmdUpdating.CommandID, cmdUpdating.Status, cmdUpdating.RunID)

	runToUpdate, mess := p.dbPB.GetRunning(ctx, cmdUpdating.RunID)
	if mess != "" {
		mess = fmt.Sprintf("Pb.GetRunning('%v' error:'%v')", runToUpdate.RunId, mess)
		p.db.UpdateStatusMCommand(ctx, cmdUpdating, models.CmdstatusSTATUS_FAILED, mess)

		return false
	}

	logger.Debugf(ctx, "ProjectID: '%v'; ScenarioID: '%v'; RunID: '%v'; RunToUpdate: '%+v';",
		runToUpdate.GetProjectId(), runToUpdate.GetScenarioId(), runToUpdate.GetRunId(), runToUpdate)

	var countRunningScriptsUpdate = 0

	var wg sync.WaitGroup
	for i := range runToUpdate.ScriptRuns {
		wg.Add(1)

		go func(curPbScriptRun *pb.ScriptRun, waitGroup *sync.WaitGroup) {
			ctxWithTimeout, canc := context.WithTimeout(ctx, 11*time.Second)
			defer canc()
			p.updateStatusScriptRun(ctxWithTimeout, cmdUpdating, curPbScriptRun, runToUpdate.GetTitle(), waitGroup)
		}(runToUpdate.ScriptRuns[i], &wg)
	}

	wg.Wait()

	for i := range runToUpdate.ScriptRuns {
		statusScriptAfterUpdate := runToUpdate.GetScriptRuns()[i].Status
		if statusScriptAfterUpdate == pb.ScriptRun_STATUS_RUNNING {
			countRunningScriptsUpdate++
			logger.Infof(ctx, "ScriptRun '%+v', running script: %v",
				statusScriptAfterUpdate, countRunningScriptsUpdate)
		}
	}

	logger.Infof(ctx, "CountRunningScripts '%v' into Run '%v' '%v'",
		countRunningScriptsUpdate, runToUpdate.GetTitle(), runToUpdate.GetRunId())

	if countRunningScriptsUpdate < 1 {
		//todo: Проверять наличие других команд по данному RunID прежде чем останавливать Ран
		logger.Warnf(ctx, "There are no running scripts in the Run %v:'%v'!",
			runToUpdate.GetTitle(), runToUpdate.GetRunId())

		runToUpdate.Status = pb.Run_STATUS_STOPPING
		logger.Infof(ctx, "Creating a new command to stop Running:", runToUpdate.GetRunId())

		_, err := p.dbPB.NewPbCmdStopScenario(ctx, runToUpdate.RunId)
		if err != nil {
			errorMessage := fmt.Sprintf("Error created new command('%v':'%v'): '%v'",
				runToUpdate.GetTitle(), runToUpdate.GetRunId(), err.Error())
			p.db.UpdateStatusMCommand(ctx, cmdUpdating, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}
	}

	runToUpdate, message := p.dbPB.UpdatePbRunningInTheDB(ctx, runToUpdate)
	if message != "" {
		logger.Warnf(ctx, "Scenario update message: '%v'", message)
		cmdUpdating.ErrorDescription = message
		p.db.UpdateStatusMCommand(ctx, cmdUpdating, models.CmdstatusSTATUS_FAILED, message)

		return false
	}

	if !p.observingForStatusRun(ctx, cmdUpdating, runToUpdate) &&
		runToUpdate.GetStatus() != pb.Run_STATUS_STOPPING {
		errorMessage := "Error when activating the update"
		p.db.UpdateStatusMCommand(ctx, cmdUpdating, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	p.db.UpdateStatusMCommand(ctx, cmdUpdating, models.CmdstatusSTATUS_COMPLETED, "")

	return p.db.DeleteMCmd(ctx, cmdUpdating)
}

func (p *ProcessorPool) stoppingScript(ctx context.Context, cmdStoppingScript *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "stoppingScript failed: '%+v'", err)
		}
	}()
	p.db.UpdateStatusMCommand(ctx, cmdStoppingScript, models.CmdstatusSTATUS_PROCESSED, "")
	logger.Infof(ctx, "Cmd.Type:'%v' ScriptIDLen:'%v' ScriptIDs:'%v'",
		cmdStoppingScript.Type, len(cmdStoppingScript.ScriptIds), cmdStoppingScript.ScriptIds)

	pbRun, message := p.dbPB.GetRunning(ctx, cmdStoppingScript.RunID)
	if message != "" {
		message = fmt.Sprintf("scriptStop GetScript errorMessage: '%v'", message)
		p.db.UpdateStatusMCommand(ctx, cmdStoppingScript, models.CmdstatusSTATUS_FAILED, message)

		return false
	}

	for _, scriptID := range cmdStoppingScript.ScriptIds {
		for _, scriptRunToStop := range pbRun.ScriptRuns {
			if int64(scriptRunToStop.GetRunScriptId()) != scriptID {
				logger.Infof(ctx, "ScriptRun search GetScriptId: '%v'",
					scriptRunToStop.GetScript().GetScriptId())

				continue
			}

			_, message = p.scriptStop(ctx, scriptRunToStop.GetRunId(), scriptRunToStop.GetRunScriptId())
			if message != "" {
				if strings.Contains(message, "no such test run") {
					logger.Warnf(ctx, "StopScript: Script stop message: '%v'", message)
				} else {
					logger.Errorf(ctx, "StopScript: Script stop message: '%v'", message)
				}
			}
		}
	}

	//fixme: проверить, нужно ли тут обновлять статус Рана
	var pbRunningFresh *pb.Run
	pbRunningFresh, message = p.dbPB.GetRunning(ctx, cmdStoppingScript.RunID)
	if message != "" {
		logger.Errorf(ctx, "StopScript: Get Run message: '%v'", message)
	}

	message = p.SetTheRunScenarioStatusStopped(ctx, pbRunningFresh)
	if message != "" {
		logger.Errorf(ctx, "Scenario stopped message: '%v'", message)
	}
	p.db.UpdateStatusMCommand(ctx, cmdStoppingScript, models.CmdstatusSTATUS_COMPLETED, message)

	return p.db.DeleteMCmd(ctx, cmdStoppingScript)
}

// TODO make it 40 lines of code tops
func (p *ProcessorPool) startingScript(ctx context.Context, cmdStartScript *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "startingScript failed: '%+v'", err)
		}
	}()
	p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_PROCESSED, "")
	logger.Infof(ctx, "Cmd.Type:'%v' ScriptIDLen:'%v' ScriptIDs:'%v' PercentageOfTarget:'%v'",
		cmdStartScript.Type, len(cmdStartScript.ScriptIds), cmdStartScript.ScriptIds, cmdStartScript.PercentageOfTarget)

	for _, scriptID := range cmdStartScript.ScriptIds {
		//nolint:gosec // scriptID < MaxInt32
		scriptToRun, err := p.dbPB.GetPbScript(ctx, int32(scriptID))
		if err != nil {
			logger.Errorf(ctx, "script.GetScript error message: '%v'", err.Error())

			if scriptToRun == nil {
				logger.Warnf(ctx, "____ScriptRun not found scriptId: '%v'", scriptID)

				errorMessage := fmt.Sprintf("ScriptRun GetPBScript message: '%v'", err.Error())
				p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

				continue
			}
		}
		scriptToRun.Tag, err = util.CheckingTagForPresenceInDB(ctx, scriptToRun.Tag, p.db)
		if err != nil {
			errorMessage := fmt.Sprintf("ScriptRun startingScript 'CheckingTag' message: '%v'", err.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)
			continue
		}
		freeAgent, err := p.agentManager.GetAFreeAgent(ctx, scriptToRun.Tag)
		if err != nil {
			errorMessage := fmt.Sprintf("ScriptRun startingScript 'GetAFreeAgent' message: '%v'", err.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}

		targetToRunningScript, err := strconv.ParseInt(scriptToRun.Options.Rps, 10, 64)
		if err != nil {
			errorMessage := fmt.Sprintf("ScriptRun convert target RPS error message: '%v'", err.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}

		var pbScriptRunToRunning = &pb.ScriptRun{
			RunId:       cmdStartScript.RunID,
			RunScriptId: math2.GetRandomID32(ctx),
			Script:      scriptToRun,
			Status:      pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED,
			//nolint:gosec // target < MaxInt32
			Target: int32(targetToRunningScript),
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
			Agent:         freeAgent,
		}

		p.loadValidation(ctx, cmdStartScript.PercentageOfTarget.Int32, pbScriptRunToRunning)

		getScenario, mes := p.dbPB.GetScenario(ctx, pbScriptRunToRunning.GetScript().GetScenarioId())
		if mes != "" {
			errorMessage := fmt.Sprintf("ScriptRun GetScenario message: '%v'", mes)
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}

		scriptRunTask := p.prepareTaskToRun(ctx, pbScriptRunToRunning, getScenario.Title)

		runTask := func() (*agentapi.StartResponse, error) {
			rs := &agentapi.StartResponse{}
			er := httputil.ProxyCall(ctx, httputil.ProxyCallConfig{
				URL: fmt.Sprintf(
					"http://%s:%s",
					pbScriptRunToRunning.Agent.HostName,
					pbScriptRunToRunning.Agent.Port,
				),
				Path:   "api/v1/start",
				Method: http.MethodPost,
				RequestBody: &agentapi.StartRequest{
					ScenarioTitle: getScenario.Title,
					ScriptId:      scriptID,
					ScriptTitle:   scriptToRun.Title,
					ScriptURL:     scriptToRun.BaseUrl,
					Params:        nil,
					ScriptRunId:   scriptRunTask.ScriptRunId,
				},
				ResponseBody: rs,
			})
			if er != nil {
				logger.Errorf(ctx, "HTTP Start request failed: %v", er)
				return nil, er
			}
			return rs, nil
		}

		rs, err := runTask()
		if err != nil {
			errorMessage := fmt.Sprintf(
				"ScriptRun Execute request Run error '%+v' scriptID:'%v'", err, scriptID)
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			mes = p.dbPB.UpdatePbScriptRunInDB(ctx, pbScriptRunToRunning.RunId, pbScriptRunToRunning)
			if mes != "" {
				logger.Errorf(ctx, "errorMessage:'%v', mes'%v'", errorMessage, mes)
			}

			continue
		}

		scriptRunMessage := p.processingTheResponseFromTheAgent(ctx, rs, pbScriptRunToRunning)
		if scriptRunMessage != "" {
			errorMessage := fmt.Sprintf(
				"ScriptRun, processingTheResponseFromTheAgent: '%v'", scriptRunMessage)
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}

		statusMessage := p.setTheScriptRunStatusRunning(ctx, scriptRunMessage, -1, pbScriptRunToRunning)
		if statusMessage != "" {
			errorMessage := fmt.Sprintf("Setting the status message: '%v'", statusMessage)
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}

		message := p.dbPB.UpdatePbScriptRunInDB(ctx, cmdStartScript.RunID, pbScriptRunToRunning)
		if message != "" {
			errorMessage := fmt.Sprintf("Error update ScriptRunning in DB '%v'", message)
			p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}
	}

	p.db.UpdateStatusMCommand(ctx, cmdStartScript, models.CmdstatusSTATUS_COMPLETED, "")

	return p.db.DeleteMCmd(ctx, cmdStartScript)
}

// TODO remove, because identical to startingScript in essence
func (p *ProcessorPool) startingRunning(ctx context.Context, cmdRunning *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "startingRunning failed: '%+v'", err)
		}
	}()

	defer undecided.InfoTimer(ctx, fmt.Sprint("Starting Running - ", cmdRunning.RunID))()
	if !p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_PROCESSED, "") {
		return false
	}

	logger.Infof(ctx, "startingRunning CommandID:'%v', CommandStatus: '%v', RunID: '%v', PercentageOfTarget: '%v'",
		cmdRunning.CommandID, cmdRunning.Status, cmdRunning.RunID, cmdRunning.PercentageOfTarget)

	pbRun, err := p.dbPB.GetRunning(ctx, cmdRunning.RunID)
	logger.Debugf(ctx, "mRun received ID:'%v'; Run:'%+v';", pbRun.GetRunId(), pbRun)
	p.markerForTheAnnotation(ctx, markerStart, pbRun.GetTitle(), pbRun.GetRunId())
	logger.Debugf(ctx, "RunId:'%v'; pbRun:'%+v';", pbRun.RunId, pbRun)

	if err != "" {
		errorMessage := fmt.Sprintf("ModelToPBRun error: '%v'", err)
		p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_FAILED, errorMessage)
		p.markerForTheAnnotation(ctx, markerStop, pbRun.GetTitle(), pbRun.GetRunId())

		return false
	}

	if len(pbRun.ScriptRuns) > 0 {
		var wg sync.WaitGroup
		for i, scriptRun := range pbRun.ScriptRuns {
			wg.Add(1)
			logger.Debugf(ctx, "ScriptRun: %+v", scriptRun)

			go func(curScriptRun *pb.ScriptRun, i int, wg *sync.WaitGroup) {
				defer func(wg *sync.WaitGroup) {
					if er := recover(); er != nil {
						logger.Errorf(ctx, "CommandProcessor startingRunning failed: '%+v'", er)
					}

					wg.Done()
				}(wg)
				logger.Debugf(ctx, "ScriptRun before load validation: %+v", curScriptRun)
				p.loadValidation(ctx, cmdRunning.PercentageOfTarget.Int32, curScriptRun)
				logger.Debugf(ctx, "ScriptRun after load validation: %+v", curScriptRun)
				scenarioRunTask := p.prepareTaskToRun(ctx, curScriptRun, pbRun.Title)

				runTask := func() (*agentapi.StartResponse, error) {
					var er error

					var tag string
					var scriptName string
					switch curScriptRun.TypeScriptRun {
					case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
						tag = curScriptRun.Script.Tag
						scriptName = curScriptRun.Script.Name
					case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
						tag = curScriptRun.SimpleScript.Tag
						scriptName = curScriptRun.SimpleScript.Name
					}
					tag, er = util.CheckingTagForPresenceInDB(ctx, tag, p.db)
					if er != nil {
						curScriptRun.Info = er.Error()
						logger.Errorf(ctx, "the tag{%v} failed verification. Script{%v}", tag, scriptName)
						return nil, er
					}

					agentHost, er := p.agentManager.GetAFreeAgent(ctx, tag)
					if er != nil {
						curScriptRun.Info = er.Error()
						logger.Errorf(ctx, "no agents were received by tag{%v}. Script{%v}", tag, scriptName)

						return nil, er
					}

					//agent, er := p.agentManager.GetClientPB(ctx, agentHost)
					//if er != nil {
					//	curScriptRun.Info = er.Error()
					//	logger.Errorf(ctx, "client{%v:%v} receipt error{%v}. Script{%v}", tag, agentHost.HostName, scriptName)
					//
					//	return nil, er
					//}

					curScriptRun.Agent = agentHost
					// todo params builder
					startRequest := agentapi2.AgentStartRequest{
						ScenarioTitle: scenarioRunTask.ScenarioTitle,
						ScriptTitle:   scriptName,
						ScriptURL:     scenarioRunTask.ScriptUrl,
						AmmoURL:       scenarioRunTask.Envs.AmmoUrl,
						Params: []string{"-e", fmt.Sprintf("RPS=%s", scenarioRunTask.Envs.Rps),
							"-e", fmt.Sprintf("DURATION=%s", scenarioRunTask.Envs.Duration),
							"-e", fmt.Sprintf("STEPS=%s", scenarioRunTask.Envs.Steps),
						},
					}
					logger.Infof(ctx, "Start Script{%s} Request %+v", scenarioRunTask.ScriptTitle, startRequest)
					// todo apiURl имеет странный префикс: ://
					//return agent.Start(ctx, startRequest)

					bytes, err := httputil.Post(
						ctx,
						fmt.Sprintf("http://%s:%s/api/v1/start",
							agentHost.HostName,
							agentHost.Port),
						"application/json",
						map[string]string{}, startRequest)
					if err != nil {
						logger.Errorf(ctx, "HTTP Start request failed: %v", er)
						return nil, er
					}

					rr := &agentapi.StartResponse{}
					err = json.Unmarshal(bytes, rr)
					if err != nil {
						logger.Errorf(ctx, "Report{%v} Unmarshal err: %+v", rr, err)

						return nil, err
					}
					logger.Infof(ctx, "pid processed: %d", rr.Pid)

					return rr, nil
				}

				rs, er := runTask()
				if er != nil {
					message := fmt.Sprintf(
						"Execute request Run error scriptID:'%v' Error: '%v'; ",
						curScriptRun.GetScript().GetScriptId(), er.Error())
					logger.Error(ctx, message, er)

					curScriptRun.Info = fmt.Sprint(curScriptRun.Info, message)
					//if message = p.dbPB.UpdatePbScriptRunInDB(ctx, curScriptRun.GetRunId(), curScriptRun); message != "" {
					//	logger.Error(ctx, message, er)
					//}

					return
				}

				agentMessage := p.processingTheResponseFromTheAgent(ctx, rs, curScriptRun)

				statusMessage := p.setTheScriptRunStatusRunning(ctx, agentMessage, i, curScriptRun)
				if statusMessage != "" {
					logger.Errorf(ctx, "setTheScriptRunStatusRunning '%v'", statusMessage)
				}

				logger.Infof(ctx, "Save scriptRunToDB '%v'", curScriptRun.GetRunScriptId())
				p.dbPB.UpdatePbRunningInTheDB(ctx, pbRun)
			}(scriptRun, i, &wg)
		}

		wg.Wait()
		p.setTheRunScenarioStatusRunning(ctx, pbRun)
		logger.Infof(ctx, "Save RunToDB '%v':'%v'", pbRun.GetRunId(), pbRun.GetStatus())
		// начинаем наблюдение за запуском
		if !p.observingForStatusRun(ctx, cmdRunning, pbRun) {
			p.db.UpdateStatusMCommand(
				ctx,
				cmdRunning,
				models.CmdstatusSTATUS_FAILED,
				"Error observingForStatusRun: false.",
			)

			return false
		}

		p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_COMPLETED, "")
		p.dbPB.UpdatePbRunningInTheDB(ctx, pbRun)

		if !p.db.DeleteMCmd(ctx, cmdRunning) {
			p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_FAILED, "Error DeleteMCmd: false.")

			return false
		}

		logger.Infof(ctx, "The scenario is running '%v' '%v'", pbRun.GetScenarioId(), pbRun.Status)

		// первое обновление делаем в рамках запуска сценария
		mRunning, er := p.db.GetMRunning(ctx, pbRun.RunId)
		if er != nil {
			p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_FAILED, "Error GetMRunning: false.")

			return false
		}

		cmdUpdate, er2 := p.db.GetRunCommand(ctx, mRunning.RunID)
		if er2 != nil {
			logger.Errorf(ctx, "Error fetching cmdUpdate for the first observing the Run", cmdRunning.Type)

			return false
		}

		ctxUpdate := undecided.NewContextWithMarker(
			ctx,
			cMDProcessorContextKey,
			fmt.Sprintf("%v__%v", cmdRunning.Type, cmdUpdate.Type),
		)
		logger.Infof(ctx, "UpdatingRun:'%v', CommandID:'%v', Status:'%v', ErrorDescription:'%v'",
			p.updatingRun(ctxUpdate, cmdUpdate), cmdUpdate.CommandID, cmdUpdate.Status, cmdUpdate.ErrorDescription)
	} else {
		errorMessage := fmt.Sprintf("There are no scripts to run. Title:'%v' RunId:'%v'",
			pbRun.Title, pbRun.RunId)
		if !p.db.UpdateStatusMCommand(ctx, cmdRunning, models.CmdstatusSTATUS_FAILED, errorMessage) {
			logger.Warnf(ctx, "Error update command '%v' status in DB", cmdRunning.Type)

			return false
		}

		pbRun.Info = fmt.Sprint(pbRun.Info, errorMessage)
		pbRun.Status = pb.Run_STATUS_STOPPED_UNSPECIFIED
		pbRun.PercentageOfTarget = cmdRunning.PercentageOfTarget.Int32

		if _, err = p.dbPB.UpdatePbRunningInTheDB(ctx, pbRun); err != "" {
			logger.Errorf(ctx, "UpdatePbRunningInTheDB error message: %v", err)
		}

		logger.Infof(ctx, "Command update '%v' - '%v'; errorMessage '%v'",
			cmdRunning.CommandID, cmdRunning.Status, errorMessage)
		p.markerForTheAnnotation(ctx, markerStop, pbRun.GetTitle(), pbRun.GetRunId())
	}

	logger.Debugf(ctx, "pbRun '%v'; error: '%v'", pbRun, err)

	execPool.Go(func() {
		logger.Infof(ctx, "Sending notifications to chats")
		// fixme refactor please
		if p.failureToComplyWithTheConditionsForNotification(ctx, pbRun) {
			logger.Warnf(ctx, "Trash notification")
			return
		}

		//message := initNfMessage(ctx, pbRun)
		//p.sendDingNotification(ctx, message.toString(false))
		//p.sendMMNotification(ctx, message.toString(false), config.Get(ctx).MmChannelID)
	})

	return true
}

// failureToComplyWithTheConditionsForNotification its test env, or not
func (p *ProcessorPool) failureToComplyWithTheConditionsForNotification(ctx context.Context, pbRun *pb.Run) bool {
	checkSmoke := p.checkSmoke(pbRun.Title)
	checkProd := p.checkProd(pbRun)
	env := config.Get(ctx).ENV

	if env != config.EnvInfra {
		logger.Warnf(ctx, "Skipped notification: by env: {%v: %v}", env, pbRun.Title)

		return true
	}
	if !checkProd {
		logger.Warnf(ctx, "Skipped notification: by checkProd: {%v: %v}", checkProd, pbRun.Title)

		return true
	}
	if checkSmoke {
		logger.Warnf(ctx, "Skipped notification: by checkSmoke: {%v: %v}", checkSmoke, pbRun.Title)

		return true
	}

	return false
}

var client = http.Client{}

// Post fixed:
// Deprecated: т.к. нужно пользоваться универсальными функциями
func Post(ctx context.Context, host string, contentType string, rqBody interface{}) (
	result []byte, err error) {
	defer func() {
		if er := recover(); er != nil {
			logger.Errorf(ctx, "Post failed: '%+v'", er)
		}
	}()

	defer undecided.WarnTimer(ctx, fmt.Sprintf("Post Request %v", host))()
	bytesMarshal, err := json.Marshal(rqBody)
	if err != nil {
		logger.Errorf(ctx, "Post rq marshal error: %v", err)

		return result, err
	}
	buffer := bytes.NewBuffer(bytesMarshal)
	logger.Infof(ctx, "Execution of POST request AR %v: %v", host, buffer.String())
	resp, err := client.Post(host, contentType, buffer)
	if err != nil {
		logger.Errorf(ctx, "Post Execution error: %v", err)

		return result, err
	}

	defer func(resp *http.Response) {
		err2 := resp.Body.Close()
		if err2 != nil {
			logger.Errorf(ctx, "resp.Body.Close Error: %v", err2.Error())
		}
	}(resp)
	logger.Debugf(ctx, "Post '%v' Request: '%+v'", host, resp.Request)
	logger.Debugf(ctx, "Post '%v' Headers: '%+v'", host, resp.Header)
	logger.Infof(ctx, "Post '%v' Status: '%+v'", host, resp.Status)

	result, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf(ctx, "Post ReadAll error: %v", err)

		return result, err
	}
	logger.Infof(ctx, "Post '%v' Body: '%+v'", host, string(result))
	logger.Debugf(ctx, "Post resp.Body: '%s'", string(result))

	return result, err
}

func (p *ProcessorPool) stoppingRunning(ctx context.Context, cmdStopping *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "stoppingRunning failed: '%+v'", err)
		}
	}()

	if !p.db.UpdateStatusMCommand(ctx, cmdStopping, models.CmdstatusSTATUS_PROCESSED, "") {
		return false
	}
	defer undecided.InfoTimer(ctx, fmt.Sprint("Stopping Running - ", cmdStopping.RunID))()
	logger.Infof(ctx, "CommandID:'%v', CommandStatus: '%v', RunID: '%v'",
		cmdStopping.CommandID, cmdStopping.Status, cmdStopping.RunID)

	pbRun, message := p.scenarioStop(ctx, cmdStopping.RunID)
	if message != "" {
		errorMessage := fmt.Sprintf("ScenarioRun stopped message: '%v'", message)
		p.db.UpdateStatusMCommand(ctx, cmdStopping, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	p.db.UpdateStatusMCommand(ctx, cmdStopping, models.CmdstatusSTATUS_COMPLETED, "")

	// Удаление лишней команды для обновления Рана
	p.db.StopObservingForStatusRun(ctx, pbRun.RunId)

	if !p.db.DeleteMCmd(ctx, cmdStopping) {
		return false
	}

	logger.Infof(ctx, "The scenario is stopped '%v'", pbRun.GetScenarioId())

	return true
}

// Функция adjustment изменение текущего процента от таргета.
//
// Опускает и поднимает скрипты с нужной нагрузкой
func (p *ProcessorPool) adjustment(ctx context.Context, cmdAdjustment *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "Adjustment failed: '%+v'", err)
		}
	}()

	logger.Infof(ctx, "Adjustment(CommandID:'%v', CommandStatus: '%v', RunID: '%v', PercentageOfTarget: '%v')",
		cmdAdjustment.CommandID, cmdAdjustment.Status, cmdAdjustment.RunID, cmdAdjustment.PercentageOfTarget.Int32)

	mRunToAdjustment, err := p.db.GetMRunning(ctx, cmdAdjustment.RunID)
	if err != nil {
		errorMessage := fmt.Sprintf("Error getting mRunning '%v' -> %v", cmdAdjustment.RunID, err.Error())
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	} else if mRunToAdjustment.Status != models.EstatusSTATUS_RUNNING {
		errorMessage := fmt.Sprintf("Interruption due to incorrect status '%v' RunID '%v'",
			mRunToAdjustment.Status, cmdAdjustment.RunID)
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	currentPercentageOfTarget := mRunToAdjustment.PercentageOfTarget
	// если требуемый процент нагрузки равен текущему
	if cmdAdjustment.PercentageOfTarget.Int32 == currentPercentageOfTarget {
		errorMessage := fmt.Sprintf(
			"The required level of '%v%%' for Run N'%v' has already been reached. The current percentage is '%v%%'",
			cmdAdjustment.PercentageOfTarget.Int32, mRunToAdjustment.RunID, currentPercentageOfTarget)
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	} else
	// если требуемый процент нагрузки больше текущего
	if cmdAdjustment.PercentageOfTarget.Int32 > currentPercentageOfTarget {
		differencePercentage := cmdAdjustment.PercentageOfTarget.Int32 - currentPercentageOfTarget
		logger.Infof(ctx,
			"The required percentage(%v) of load is greater than the current percentage(%v), we increase the missing load(%v)",
			cmdAdjustment.PercentageOfTarget.Int32, currentPercentageOfTarget, differencePercentage)

		// нужно только добавить запуск скриптов/simple скриптов с недостающей нагрузкой в текущий запуск:
		// запускаем скрипты:
		arrayScriptID, mess := util.GetArrayActiveScriptIDs(ctx, mRunToAdjustment.ScenarioID, p.db)
		if mess != "" {
			errorMessage := fmt.Sprintf("Error getting arrayScriptID(runID:'%v'; ScenarioID:'%v'; Error:%v)",
				cmdAdjustment.RunID, mRunToAdjustment.ScenarioID, mess)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		cmdRunScripts, er := p.db.NewMCmdRunScriptWithRpsAdjustment(ctx,
			cmdAdjustment.RunID, undecided.ConvertArrayInt32toArrayInt64(arrayScriptID), differencePercentage)
		if er != nil {
			errorMessage := fmt.Sprintf(
				"Error create cmd to running adjustment(script) '%v%%'(runID:'%v'; ScenarioID:'%v'; Error:%v)",
				differencePercentage, cmdAdjustment.RunID, mRunToAdjustment.ScenarioID, er.Error())
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		statusRunningScripts := p.startingScript(ctx, cmdRunScripts)
		if !statusRunningScripts {
			message := fmt.Sprintf("Running Script fali{status:{%v}; arrayScriptID:{%+v}}",
				statusRunningScripts, arrayScriptID)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)
		}

		logger.Infof(ctx, "RunningScript to :'%v', CommandID:'%v', Status:'%v'",
			statusRunningScripts, cmdRunScripts.CommandID, cmdRunScripts.Status)

		// запускаем Simple скрипты:
		arraySimpleScriptID, mess := util.GetArrayActiveSimpleScriptIDs(ctx, mRunToAdjustment.ScenarioID, p.db)
		if mess != "" {
			errorMessage := fmt.Sprintf("Error getting arraySimpleScriptID(runID:'%v'; ScenarioID:'%v'; Error:%v)",
				cmdAdjustment.RunID, mRunToAdjustment.ScenarioID, mess)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		cmdRunSimpleScripts, er2 := p.db.NewMCmdRunSimpleScriptWithRpsAdjustment(ctx,
			cmdAdjustment.RunID, undecided.ConvertArrayInt32toArrayInt64(arraySimpleScriptID), differencePercentage)
		if er2 != nil {
			errorMessage := fmt.Sprintf(
				"Error create cmd to running adjustment(simpleScript) '%v%%'(runID:'%v'; ScenarioID:'%v'; Error:%v)",
				differencePercentage, cmdAdjustment.RunID, mRunToAdjustment.ScenarioID, er2.Error())
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		statusRunningSimpleScripts := p.startingSimpleScript(ctx, cmdRunSimpleScripts)
		if !statusRunningScripts {
			message := fmt.Sprintf("Running SimpleScript fali{status:{%v}; arrayScriptID:{%+v}}",
				statusRunningSimpleScripts, arraySimpleScriptID)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)
		}

		logger.Infof(ctx, "RunningSimpleScript:'%v', CommandID:'%v', Status:'%v'",
			statusRunningSimpleScripts, cmdRunSimpleScripts.CommandID, cmdRunSimpleScripts.Status)
	} else
	// если требуемый процент нагрузки меньше текущего
	if cmdAdjustment.PercentageOfTarget.Int32 < currentPercentageOfTarget {
		// получаем разницу которую нужно запускать
		differencePercent := currentPercentageOfTarget - cmdAdjustment.PercentageOfTarget.Int32
		logger.Infof(ctx,
			"The required percentage of load (%v) is less than the current percentage(%v), the difference when decreasing(%v)",
			cmdAdjustment.PercentageOfTarget.Int32, currentPercentageOfTarget, differencePercent)
		// получаем активные скрипты в сценарии(не запущенные, а активные)
		pbEnabledScriptsToRun, er := p.dbPB.GetAllEnabledScripts(ctx, mRunToAdjustment.ScenarioID)
		if er != nil {
			errorMessage := fmt.Sprintf("Error get All Enabled Scripts: '%v'", er.Error())
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		logger.Infof(ctx, "Count of active scripts to adjustment(%v) in Scenario(%v) ScenarioID(%v)",
			len(pbEnabledScriptsToRun), mRunToAdjustment.Title, mRunToAdjustment.ScenarioID)

		if !p.processingTheLoadReductionInRunningScenarioByScripts(ctx, cmdAdjustment, differencePercent, pbEnabledScriptsToRun) {
			return false
		}

		// получаем активные Simple скрипты в сценарии
		pbEnabledSimpleScriptsToRun, er := p.dbPB.GetAllEnabledPBSimpleScripts(ctx, mRunToAdjustment.ScenarioID)
		if er != nil {
			errorMessage := fmt.Sprintf("Error get All Enabled Scripts: '%v'", er.Error())
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

			return false
		}

		logger.Infof(ctx, "Count of active scripts to adjustment(%v) in Scenario(%v) ScenarioID(%v)",
			len(pbEnabledScriptsToRun), mRunToAdjustment.Title, mRunToAdjustment.ScenarioID)

		if !p.processingTheLoadReductionInRunningScenarioBySimpleScripts(ctx, cmdAdjustment, differencePercent, pbEnabledSimpleScriptsToRun) {
			return false
		}
	}

	mRunToAdjustment, err = p.db.GetMRunning(ctx, mRunToAdjustment.RunID)
	if err != nil {
		errorMessage := fmt.Sprintf("Error when getting the mRunning after adjustment '%v' -> %v",
			cmdAdjustment.RunID, err.Error())
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	mRunToAdjustment.PercentageOfTarget = cmdAdjustment.PercentageOfTarget.Int32

	_, err = p.db.UpdateMRunningInTheDB(ctx, mRunToAdjustment)
	if err != nil {
		errorMessage := fmt.Sprintf("Error updating mRunning in DB '%v' -> %v", cmdAdjustment.RunID, err.Error())
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_COMPLETED, "")

	return p.db.DeleteMCmd(ctx, cmdAdjustment)
}

// Процесс снижения нагрузки Сценария по скриптам
func (p *ProcessorPool) processingTheLoadReductionInRunningScenarioByScripts(ctx context.Context,
	cmdAdjustment *models.Command, differencePercent int32, pbEnabledScriptsToRun []*pb.Script) bool {
	var scriptsRunIDsToStop []int32

	for _, pbEnabledScript := range pbEnabledScriptsToRun {
		// получили все СкриптРаны по этому скрипту
		pbWorkingScriptRuns, mess := p.dbPB.GetWorkingScriptRunsByScriptID(ctx,
			cmdAdjustment.RunID, pbEnabledScript.GetScriptId())
		if mess != "" {
			errorMessage := fmt.Sprintf("Error get ScriptRunning: '%v'", mess)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		if len(pbWorkingScriptRuns) < 1 {
			errorMessage := fmt.Sprintf("Error Decrease ScriptId(%v); Active ScriptRunning to decreace (%+v)",
				pbEnabledScript.GetScriptId(), pbWorkingScriptRuns)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		logger.Infof(ctx, "Number of active scriptsRun(ScriptId:%v) to decrease: '%v'",
			pbEnabledScript.GetScriptId(), len(pbWorkingScriptRuns))
		// получаем какое кол-во РПС нужно остановить
		decreaseRPS := p.calculateTheDecreaseOnRPS(ctx, differencePercent, pbWorkingScriptRuns[0].GetTarget())
		logger.Infof(ctx, "DecreaseRPS scriptsRun(ScriptId:%v; ScriptName:%v): '%v'",
			pbEnabledScript.GetScriptId(), pbEnabledScript.GetName(), decreaseRPS)

		arrayDecreaseScriptRun, err := p.preparingDecreaseRunScriptElements(ctx, cmdAdjustment, pbWorkingScriptRuns)
		if err != nil {
			errorMessage := fmt.Sprintf(
				"Error Preparing DecreaseRunScriptElements{ScriptId(%v); DecreaseRunScriptElements(%+v); Error(%+v)}",
				pbEnabledScript.GetScriptId(), arrayDecreaseScriptRun, err)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		if len(arrayDecreaseScriptRun) < 1 {
			errorMessage := fmt.Sprintf(
				"Error Preparing DecreaseRunScriptElements{ScriptId(%v); DecreaseRunScriptElements(%+v)}",
				pbEnabledScript.GetScriptId(), arrayDecreaseScriptRun)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}
		//  получили скрипты для остановки и кол-во РПС что бы сбалансировать РПС до цели
		scriptRunIDsToStop, increaseRPS := p.decreaseLoadAlgorithm(ctx, arrayDecreaseScriptRun, decreaseRPS)
		logger.Infof(ctx, "ScriptRunIDs To Stop(ScriptId: '%v'; IDs to stop: '%v'; increaseRPS: '%v')",
			pbEnabledScript.GetScriptId(), scriptRunIDsToStop, increaseRPS)
		// fixme: нужно обрабатывать ситуацию понижения нагрузки при целевой нагрузке = 1 РПСу
		// if pbWorkingScriptRuns[0].GetTarget() == 1 && cmdAdjustment.PercentageOfTarget.Int32 > 0 && cmdAdjustment.PercentageOfTarget.Int32 < 100 {
		//	break
		//}

		// при необходимости сохранить требуемую нагрузку поднимаем 1 скрипт с корректировкой RPS(increasePercent)
		if increaseRPS > 0 {
			logger.Infof(ctx, "Creating a new command to increase script '%v'", models.CmdtypeTYPE_RUN_SCRIPT)

			increasePercent := p.calculateTheIncreaseOnPercent(ctx, increaseRPS, pbWorkingScriptRuns[0].GetTarget())

			runningScriptMCmd, er := p.db.NewMCmdRunScriptWithRpsAdjustment(ctx, cmdAdjustment.RunID,
				undecided.ConvInt32toArrInt64(pbEnabledScript.GetScriptId()), increasePercent)
			if er != nil {
				errorMessage := fmt.Sprintf("Error created new command to increase runScripts: '%v'", er.Error())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

				continue
			}

			logger.Infof(ctx, "Created new command to increase runScripts:'%+v'", runningScriptMCmd)
			// запускаем скрипты, проверяем статус
			statusRunningScripts := p.startingScript(ctx, runningScriptMCmd)
			if !statusRunningScripts {
				message := fmt.Sprintf("Running Script fali{status:{%v}; ScriptID:{%+v}}",
					statusRunningScripts, pbEnabledScript.GetScriptId())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)

				continue
			}

			logger.Infof(ctx, "RunningScript:'%v', CommandID:'%v', Status:'%v'",
				statusRunningScripts, cmdAdjustment.CommandID, cmdAdjustment.Status)
		}

		// останавливаем скрипты с лишней нагрузкой
		if len(scriptRunIDsToStop) > 0 {
			// проставляем статусы скриптам которые нужно остановить
			for _, scriptRunToStop := range pbWorkingScriptRuns {
				for _, runScriptIDToStop := range scriptRunIDsToStop {
					if scriptRunToStop.GetRunScriptId() == runScriptIDToStop {
						scriptRunToStop.Status = pb.ScriptRun_STATUS_STOPPING

						message := p.dbPB.UpdatePbScriptRunInDB(ctx, cmdAdjustment.RunID, scriptRunToStop)
						if message != "" {
							errorMessage := fmt.Sprintf("Error Update In DB ScriptRunningToStop: '%+v' '%v'",
								runScriptIDToStop, message)
							if !p.db.UpdateStatusMCommand(
								ctx,
								cmdAdjustment,
								models.CmdstatusSTATUS_FAILED,
								errorMessage,
							) {
								logger.Warnf(ctx, "UpdateStatusMCommand error message", errorMessage)

								continue
							}
						}
					}
				}
			}

			scriptsRunIDsToStop = append(scriptsRunIDsToStop, scriptRunIDsToStop...)
		} else {
			logger.Warnf(ctx, "There are no scripts to stop '%v'", len(scriptRunIDsToStop))
		}
	}

	logger.Infof(ctx, "Creating a new command to stop runScripts '%v'",
		models.CmdtypeTYPE_STOP_SCRIPT)

	mCmd, er := p.db.NewMCmdStopScript(ctx,
		cmdAdjustment.RunID, undecided.ConvertArrayInt32toArrayInt64(scriptsRunIDsToStop))
	if er != nil {
		errorMessage := fmt.Sprintf("Error created new command to stop runScripts: '%v'", er.Error())
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	logger.Infof(ctx, "Created new command to stop runScripts:'%v'", mCmd)
	// останавливаем скрипты, проверяем статус
	statusStoppingScripts := p.stoppingScript(ctx, mCmd)
	if !statusStoppingScripts {
		message := fmt.Sprintf("Stopping Scripts fali{status:{%v}; Scripts:{%+v}}",
			statusStoppingScripts, pbEnabledScriptsToRun)
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)

		return false
	}

	return true
}

// Процесс снижения нагрузки Сценария по Simple скриптам
func (p *ProcessorPool) processingTheLoadReductionInRunningScenarioBySimpleScripts(ctx context.Context,
	cmdAdjustment *models.Command, differencePercent int32, pbEnabledSimpleScriptsToRun []*pb.SimpleScript) bool {
	var simpleScriptsRunIDsToStop []int32

	for _, pbEnabledSimpleScript := range pbEnabledSimpleScriptsToRun {
		// получили все СкриптРаны по этому скрипту
		pbWorkingSimpleScriptRuns, mess := p.dbPB.GetWorkingScriptRunsByScriptID(ctx,
			cmdAdjustment.RunID, pbEnabledSimpleScript.GetScriptId())
		if mess != "" {
			errorMessage := fmt.Sprintf("Error get ScriptRunning: '%v'", mess)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		if pbWorkingSimpleScriptRuns == nil {
			errorMessage := fmt.Sprintf("Error Decrease SimpleScriptId(%v); Active ScriptRunning to decreace (%+v)",
				pbEnabledSimpleScript.GetScriptId(), pbWorkingSimpleScriptRuns)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		logger.Infof(ctx, "Number of active scriptsRun(ScriptId:%v) to adjustment: '%v'",
			pbEnabledSimpleScript.GetScriptId(), len(pbWorkingSimpleScriptRuns))
		// считаем какое кол-во РПС нужно остановить
		decreaseRPS := p.calculateTheDecreaseOnRPS(ctx, differencePercent, pbWorkingSimpleScriptRuns[0].GetTarget())
		logger.Infof(ctx, "DecreaseRPS scriptsRun(ScriptId:%v; ScriptName:%v): '%v'",
			pbEnabledSimpleScript.GetScriptId(), pbEnabledSimpleScript.GetName(), decreaseRPS)

		arrayDecreaseScriptRuns, err := p.preparingDecreaseRunScriptElements(
			ctx,
			cmdAdjustment,
			pbWorkingSimpleScriptRuns,
		)
		if err != nil {
			errorMessage := fmt.Sprintf(
				"Error Preparing DecreaseRunScriptElements{ScriptId(%v); DecreaseRunScriptElements(%+v); Error(%+v)}",
				pbEnabledSimpleScript.GetScriptId(), arrayDecreaseScriptRuns, err)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		if arrayDecreaseScriptRuns == nil {
			errorMessage := fmt.Sprintf(
				"Error Preparing DecreaseRunScriptElements{ScriptId(%v); DecreaseRunScriptElements(%+v)}",
				pbEnabledSimpleScript.GetScriptId(), arrayDecreaseScriptRuns)
			logger.Warnf(ctx, errorMessage)
			p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}
		//  получили скрипты для остановки и кол-во РПС что бы сбалансировать РПС до цели
		simpleScriptRunIDsToStop, increaseRPS := p.decreaseLoadAlgorithm(ctx, arrayDecreaseScriptRuns, decreaseRPS)
		logger.Infof(ctx, "ScriptRunIDs To Stop(ScriptId: '%v'; IDs to stop: '%+v'; increaseRPS: '%v')",
			pbEnabledSimpleScript.GetScriptId(), simpleScriptRunIDsToStop, increaseRPS)
		// fixme: нужно обрабатывать ситуацию понижения нагрузки при целевой нагрузке = 1 РПСу
		// if pbWorkingSimpleScriptRuns[0].GetTarget() == 1 && cmdAdjustment.PercentageOfTarget.Int32 > 0 && cmdAdjustment.PercentageOfTarget.Int32 < 100 {
		//	break
		//}

		// при необходимости сохранить требуемую нагрузку поднимаем 1 Simple скрипт с корректировкой RPS(increasePercent)
		if increaseRPS > 0 {
			logger.Infof(ctx, "Creating a new command to increase Simplescript '%v : %v'",
				models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT, pbEnabledSimpleScript.GetName())

			increasePercent := p.calculateTheIncreaseOnPercent(
				ctx,
				increaseRPS,
				pbWorkingSimpleScriptRuns[0].GetTarget(),
			)

			runningScriptMCmd, er := p.db.NewMCmdRunSimpleScriptWithRpsAdjustment(ctx, cmdAdjustment.RunID,
				undecided.ConvInt32toArrInt64(pbEnabledSimpleScript.GetScriptId()), increasePercent)
			if er != nil {
				errorMessage := fmt.Sprintf("Error created new command to increase runScripts: '%v'", er.Error())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

				continue
			}

			logger.Infof(ctx, "Created new command to increase runSimpleScripts:'%+v'", runningScriptMCmd)
			// запускаем скрипты, проверяем статус
			statusRunningSimpleScripts := p.startingSimpleScript(ctx, runningScriptMCmd)
			if !statusRunningSimpleScripts {
				message := fmt.Sprintf("Running SimpleScript fali{status:{%v}; ScriptID:{%+v}}",
					statusRunningSimpleScripts, pbEnabledSimpleScript.GetScriptId())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)

				continue
			}

			logger.Infof(ctx, "RunningSimpleScript:'%v', CommandID:'%v', Status:'%v'",
				statusRunningSimpleScripts, cmdAdjustment.CommandID, cmdAdjustment.Status)
		}

		// останавливаем скрипты с лишней нагрузкой
		if len(simpleScriptRunIDsToStop) > 0 {
			// проставляем статусы скриптам которые нужно остановить
			for _, scriptRunToStop := range pbWorkingSimpleScriptRuns {
				for _, runScriptIDToStop := range simpleScriptRunIDsToStop {
					if scriptRunToStop.GetRunScriptId() == runScriptIDToStop {
						scriptRunToStop.Status = pb.ScriptRun_STATUS_STOPPING

						message := p.dbPB.UpdatePbScriptRunInDB(ctx, cmdAdjustment.RunID, scriptRunToStop)
						if message != "" {
							errorMessage := fmt.Sprintf("Error Update In DB ScriptRunningToStop: '%+v' '%v'",
								runScriptIDToStop, message)
							if !p.db.UpdateStatusMCommand(
								ctx,
								cmdAdjustment,
								models.CmdstatusSTATUS_FAILED,
								errorMessage,
							) {
								logger.Warnf(ctx, "UpdateStatusMCommand error message", errorMessage)

								continue
							}
						}
					}
				}
			}

			simpleScriptsRunIDsToStop = append(simpleScriptsRunIDsToStop, simpleScriptRunIDsToStop...)
		} else {
			logger.Warnf(ctx, "There are no scripts to stop '%v'", len(simpleScriptRunIDsToStop))
		}
	}

	logger.Infof(ctx, "Creating a new command to stop runScripts '%v'",
		models.CmdtypeTYPE_STOP_SCRIPT)

	mCmd, er := p.db.NewMCmdStopScript(ctx,
		cmdAdjustment.RunID, undecided.ConvertArrayInt32toArrayInt64(simpleScriptsRunIDsToStop))
	if er != nil {
		errorMessage := fmt.Sprintf("Error created new command to stop runScripts: '%v'", er.Error())
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	logger.Infof(ctx, "Created new command to stop runScripts:'%v'", mCmd)
	// останавливаем скрипты, проверяем статус
	statusStoppingScripts := p.stoppingScript(ctx, mCmd)
	if !statusStoppingScripts {
		message := fmt.Sprintf("Stopping Scripts fali{status:{%v}; Scripts:{%+v}}",
			statusStoppingScripts, pbEnabledSimpleScriptsToRun)
		p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, message)

		return false
	}

	return true
}

// Функция собирает массив объектов decreaseScriptRun для алгоритма функции понижения нагрузки
func (p *ProcessorPool) preparingDecreaseRunScriptElements(ctx context.Context,
	cmdAdjustment *models.Command, pbActiveScriptsRun []*pb.ScriptRun) (
	decreaseRunScripts []decreaseScriptRun, err error) {
	decreaseRunScripts = []decreaseScriptRun{}

	for _, pbScriptRun := range pbActiveScriptsRun {
		var rps int64

		switch pbScriptRun.TypeScriptRun {
		case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
			rps, err = strconv.ParseInt(pbScriptRun.GetScript().GetOptions().GetRps(), 10, 64)
			if err != nil {
				errorMessage := fmt.Sprintf("Adjustment script(%v). RPS conversion error: '%v'",
					pbScriptRun.GetSimpleScript().GetScriptId(), err.Error())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

				return nil, err
			}
		case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
			rps, err = strconv.ParseInt(pbScriptRun.GetSimpleScript().GetRps(), 10, 64)
			if err != nil {
				errorMessage := fmt.Sprintf("Adjustment script(%v). RPS conversion error: '%v'",
					pbScriptRun.GetSimpleScript().GetScriptId(), err.Error())
				p.db.UpdateStatusMCommand(ctx, cmdAdjustment, models.CmdstatusSTATUS_FAILED, errorMessage)

				return nil, err
			}
		}

		elem := decreaseScriptRun{
			ScriptRunID: pbScriptRun.GetRunScriptId(),
			//nolint:gosec // rps < MaxInt32
			CurRps: int32(rps),
		}
		decreaseRunScripts = append(decreaseRunScripts, elem)
	}

	return decreaseRunScripts, nil
}

func (p *ProcessorPool) calculateTheRunningLoad(
	ctx context.Context,
	percentageOfTarget int32,
	target int32,
) (rps string) {
	fRps := undecided.PercentageRoundedToAWhole(float64(percentageOfTarget), float64(target))
	logger.Infof(ctx, "Calculate the running load: ('%v'/100)*'%v'='%v' rps", target, percentageOfTarget, fRps)

	return fmt.Sprint(fRps)
}

// Если в команде указано запустить с другим процентом от целевой нагрузки(RPS`ом)
//
// функция корректирует опцию RPS у ScriptRun для запуска с нужным РПСом
// Если PercentageOfTarget == 0, то не меняем исходное значение
func (p *ProcessorPool) loadValidation(
	ctx context.Context,
	percentageOfTargetFromCommand int32,
	pbScriptRunToRunning *pb.ScriptRun,
) {
	if percentageOfTargetFromCommand != 0 {
		switch pbScriptRunToRunning.TypeScriptRun {
		case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
			logger.Infof(ctx,
				"Adjusting the RPS value in the script options. ScriptID: '%v', PercentageOfTarget: '%v'",
				pbScriptRunToRunning.GetScript().GetScriptId(), percentageOfTargetFromCommand)

			pbScriptRunToRunning.GetScript().Options.Rps = p.calculateTheRunningLoad(ctx,
				percentageOfTargetFromCommand, pbScriptRunToRunning.GetTarget())

			logger.Infof(ctx, "Adjusting the RPS value in the script options. ScriptID: '%v', new Rps: '%v'",
				pbScriptRunToRunning.GetScript().GetScriptId(), pbScriptRunToRunning.GetScript().Options.Rps)
		case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
			logger.Infof(ctx,
				"Adjusting the RPS value in the simpleScript options. SimpleScriptID: '%v', PercentageOfTarget: '%v'",
				pbScriptRunToRunning.GetSimpleScript().GetScriptId(), percentageOfTargetFromCommand)

			pbScriptRunToRunning.GetSimpleScript().Rps = p.calculateTheRunningLoad(ctx,
				percentageOfTargetFromCommand, pbScriptRunToRunning.GetTarget())

			logger.Infof(
				ctx,
				"Adjusting the RPS value in the simpleScript options. SimpleScriptID: '%v', new Rps: '%v'",
				pbScriptRunToRunning.GetSimpleScript().GetScriptId(),
				pbScriptRunToRunning.GetSimpleScript().Rps,
			)
		}
	}

	if pbScriptRunToRunning.TypeScriptRun == pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE {
		err := processing.SimpleScriptGenerate(ctx, pbScriptRunToRunning.SimpleScript, p.db)
		if err != nil {
			logger.Errorf(ctx, "Generate SimpleScript error from loadValidation: %v", err.Error())
		}

		time.Sleep(time.Second * 3)
	}
}

// Функция вычисляет какие запущенные экземпляры скрипта требуется погасить
// и на сколько РПС поднять если нагрузки будет недостаточно
func (p *ProcessorPool) decreaseLoadAlgorithm(
	ctx context.Context,
	inputRunScripts []decreaseScriptRun,
	decreaseOnRPS int32,
) (
	runScriptIDsToStop []int32, resumeRPS int32) {
	logger.Infof(ctx, "Input scripts to decrease on '%v': '%+v'", decreaseOnRPS, inputRunScripts)
	sort.SliceStable(inputRunScripts, func(i, j int) bool {
		return inputRunScripts[i].CurRps > inputRunScripts[j].CurRps
	})
	logger.Infof(ctx, "Sorted scripts to decrease on '%v': '%+v'", decreaseOnRPS, inputRunScripts)

	runScriptIDsToStop = []int32{}
	resumeRPS = decreaseOnRPS

	for _, dataRunningScript := range inputRunScripts {
		runScriptIDsToStop = append(runScriptIDsToStop, dataRunningScript.ScriptRunID)
		resumeRPS = resumeRPS - dataRunningScript.CurRps
		logger.Infof(ctx, "ResumeRPS '%v'", resumeRPS)

		if resumeRPS <= 0 {
			break
		}
	}

	logger.Infof(ctx, "To stop %v %v", runScriptIDsToStop, resumeRPS)
	// resumeRPS на данный момент должен быть либо отрицательным,
	// либо равным 0. т.к. совокупная нагрузка со всех скриптов должна быть больше чем корректировка.
	// А если больше 0 - ищем ошибку в алгоритме работы функции adjustment!!!!
	if resumeRPS > 0 {
		logger.Warnf(ctx,
			"Error algorithm, incorrect load decrease(decreaseOnRPS:'-%v'; resumeRPS: '+%v'; inputRunScripts: '%+v')",
			decreaseOnRPS, resumeRPS, inputRunScripts)

		resumeRPS = 0
	} else {
		//nolint:gosec
		resumeRPS = int32(math.Abs(float64(resumeRPS)))
	}

	return runScriptIDsToStop, resumeRPS
}

// Функция рассчитывает число являющееся процентом от target. percentageOfTarget - процент от target
func (p *ProcessorPool) calculateTheDecreaseOnRPS(
	ctx context.Context,
	percentageOfTarget int32,
	target int32,
) (rps int32) {
	fRps := undecided.PercentageRoundedToAWhole(float64(percentageOfTarget), float64(target))
	logger.Infof(ctx, "Calculate the decrease RPS: ('%v'/100)*'%v'='%v' rps", target, percentageOfTarget, fRps)

	//nolint:gosec
	return int32(fRps)
}

// Функция перерасчета РПС в проценты по отношению к целевой нагрузке
func (p *ProcessorPool) calculateTheIncreaseOnPercent(ctx context.Context, rps int32, target int32) (percentage int32) {
	fPercent := undecided.WhatPercentageRoundedToWhole(float64(rps), float64(target))
	logger.Infof(ctx, "Calculate the increase percent: ('%v'/'%v') * 100='%v'%c ", rps, target, fPercent, '%')

	//nolint:gosec
	return int32(fPercent)
}

// Процесс обработки ответного сообщения от агента
func (p *ProcessorPool) processingTheResponseFromTheAgent(ctx context.Context,
	responseFromAgent *agentapi.StartResponse, scriptRun *pb.ScriptRun) (message string) {
	scriptRun.Pid = responseFromAgent.Pid

	if responseFromAgent.Task != nil {
		scriptRun.LogFileName = responseFromAgent.Task.LogFileName
		scriptRun.K6ApiPort = responseFromAgent.Task.K6ApiPort
		scriptRun.PortPrometheus = responseFromAgent.Task.PortPrometheus
	} else {
		message = fmt.Sprintf("Task == nil '%v'", responseFromAgent)
		scriptRun.Info = fmt.Sprintf("%s; %+v", scriptRun.Info, responseFromAgent)
		logger.Error(ctx, "Error responseFromAgent. Task is nil", message)
	}

	logger.Debugf(ctx, "Response processing message: '%v'; ScriptRun: '%v'", message, scriptRun)

	return message
}

func (p *ProcessorPool) stoppingScriptRunAndSetTheStatusStopped(ctx context.Context, scrRun *pb.ScriptRun) (
	updatedScriptRun *pb.ScriptRun, message string) {
	defer undecided.InfoTimer(
		ctx,
		fmt.Sprintf("Stopping ScriptRun{RunId: %v, RunScriptId: %v}", scrRun.RunId, scrRun.RunScriptId),
	)()
	scrRun, message = p.scriptStop(ctx, scrRun.RunId, scrRun.RunScriptId)
	if message == "" {
		logger.Warnf(ctx, "ScriptRun.Status: '%v'", scrRun.Status)

		if scrRun.Status == pb.ScriptRun_STATUS_FAILED {
			scrRun.Info = message
			message = fmt.Sprintf("RunScriptId:'%v' is status FAILED ", scrRun.RunScriptId)
			logger.Warnf(ctx, "scriptStop message: ", message)
		} else {
			scrRun.Status = pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED
			logger.Infof(ctx, "Script stopped RunScriptId:'%v' Status: '%v'", scrRun.GetRunScriptId(), scrRun.Status)
		}

		scrRun.Metrics.ExecutionStatus = "Teardown"
		logger.Debugf(ctx, "--Set the RunScript status stopped: '%+v'", scrRun)

		message = p.dbPB.UpdatePbScriptRunInDB(ctx, scrRun.RunId, scrRun)
		if message != "" {
			logger.Errorf(ctx, "UpdatePbScriptRunInDB error: '%v'", message)
		}
	} else if !strings.Contains(message, "no such test run") {
		logger.Errorf(ctx, "scenarioStop scriptStop error : '%v'", message)
	} else {
		logger.Warnf(ctx, "scenarioStop scriptStop error: '%v'", message)
	}

	return scrRun, message
}

func (p *ProcessorPool) scriptStop(ctx context.Context, runID int32, scriptRunID int32) (
	returnScriptRun *pb.ScriptRun, message string) {
	returnScriptRun, message = p.dbPB.GetScriptRunning(ctx, runID, scriptRunID)
	defer undecided.InfoTimer(ctx, fmt.Sprintf("script stop{RunId: %v, RunScriptId: %v}", runID, scriptRunID))()
	if message == "" {
		if returnScriptRun == nil {
			message = fmt.Sprintf("scriptStop No ScriptRun found '%v' in '%v'", scriptRunID, runID)

			return returnScriptRun, message
		}

		stopTask := func() (*agentapi.StopResponse, error) {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf(ctx, "stopTask failed: '%+v'", err)
				}
			}()
			agent, err := p.agentManager.GetClientPB(ctx, returnScriptRun.Agent)
			if err != nil {
				logger.Errorf(ctx, "stopTask GetClientPB failed: '%+v'", err)
				return nil, err
			}
			if agent == nil {
				err = fmt.Errorf("invalid getting agentClientPB")
				logger.Errorf(ctx, "stopTask GetClientPB{%v -> %v} failed: '%+v'",
					scriptRunID, returnScriptRun.Agent, err)

				return nil, err
			}

			ctxWithTimeout, canc := context.WithTimeout(ctx, 11*time.Second)
			defer canc()
			return agent.Stop(ctxWithTimeout, &agentapi.StopRequest{
				Pid: returnScriptRun.Pid,
			})
		}

		_, err := stopTask()
		if err != nil {
			logger.Debugf(ctx, "stopTask err: %v", err)
			message = fmt.Sprintf("Execute request Stop error '%v'", err.Error())
			if strings.Contains(err.Error(), "no such test run") {
				logger.Warnf(ctx, "'%v'", message)
			} else {
				returnScriptRun.Metrics.ExecutionStatus = executionStatusInterrupted
				logger.Errorf(ctx, "'%v'", message)
			}
			if returnScriptRun.Metrics.ExecutionStatus != executionStatusInterrupted {
				returnScriptRun.Metrics.ExecutionStatus = executionStatusEnded
			}
			returnScriptRun.Status = pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED
			logger.Warnf(ctx, "scriptStop: Set RunScriptId '%v' status:'%+v'; '%v'",
				returnScriptRun.GetRunScriptId(), returnScriptRun.Status, err)

			returnScriptRun.Info = fmt.Sprint(returnScriptRun.Info, err)
		}

		returnScriptRun.Status = pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED
		logger.Infof(ctx, "Set RunScriptId '%v' status:'%+v'",
			returnScriptRun.GetRunScriptId(), returnScriptRun.Status)
	}

	if returnScriptRun.Metrics.ExecutionStatus != executionStatusInterrupted {
		returnScriptRun.Metrics.ExecutionStatus = executionStatusEnded
	}
	logger.Infof(ctx, "scriptStop(runID:'%v'; ScriptRunStatus:'%v'; Info:'%v';)",
		runID, returnScriptRun.Status, returnScriptRun.Info)

	mes := p.dbPB.UpdatePbScriptRunInDB(ctx, runID, returnScriptRun)
	if mes != "" {
		message = fmt.Sprintf("scriptStop: Error Set ScriptRunning to DB '%v'", mes)
		logger.Errorf(ctx, message)

		return returnScriptRun, message
	}

	logger.Debugf(ctx, "ReturnStopScriptRun(Message:'%v', ScriptRun:'%+v')", message, returnScriptRun)

	return returnScriptRun, message
}

func (p *ProcessorPool) scenarioStop(ctx context.Context, runID int32) (returnRun *pb.Run, message string) {
	defer undecided.InfoTimer(ctx, fmt.Sprint("Stopping scenario - ", runID))()
	returnRun, message = p.dbPB.GetRunning(ctx, runID)

	defer p.markerForTheAnnotation(ctx, markerStop, returnRun.GetTitle(), returnRun.GetRunId())

	if message != "" {
		logger.Debugf(ctx, "scenarioStop ReturnRun: '%v'; Message: '%v'", returnRun, message)

		return returnRun, message
	}

	if returnRun.GetStatus() != pb.Run_STATUS_STOPPED_UNSPECIFIED { //fixme: не обязательно проверять на статус, что бы остановить скрипты
		countRunning := 0

		for i, scrRun := range returnRun.GetScriptRuns() { //fixme: переделать на параллельную остановку всех скриптРанов
			var mes string
			returnRun.GetScriptRuns()[i], mes = p.stoppingScriptRunAndSetTheStatusStopped(ctx, scrRun)

			if returnRun.GetScriptRuns()[i].GetStatus() == pb.ScriptRun_STATUS_RUNNING {
				countRunning++
			}

			logger.Warnf(ctx,
				"stoppingScriptRunAndSetTheStatusStopped ScriptRun:'%v' Status:'%+v' message: '%v'",
				returnRun.GetScriptRuns()[i].GetRunScriptId(), returnRun.GetScriptRuns()[i].GetStatus(), mes)
		}

		if countRunning < 1 {
			message = p.SetTheRunScenarioStatusStopped(ctx, returnRun)
			if message != "" {
				logger.Warnf(ctx, "Scenario stopped message: '%v'", message)
			}

			returnRun, message = p.dbPB.UpdatePbRunningInTheDB(ctx, returnRun)
			if message != "" {
				logger.Warnf(ctx, "Scenario stopped -> UpdatePbRunningInTheDB message: '%v'", message)

				return returnRun, message
			}
		} else {
			message = fmt.Sprintf(
				"The script is not stopped, number of runned:'%v'; number of scriptRuns:'%v';",
				countRunning, len(returnRun.GetScriptRuns()))
			returnRun.Info = message
			logger.Warnf(ctx, "scenarioStop: ", message)
		}
	}

	return returnRun, message
}

//	Функция отправки маркеров запуска и остановки нагрузочных тестов
//
// для использования в grafanaAnnotations по запросу qa_alilo_load_testing_running{}>0.
// Список актуальных labels: "load_testing"(title), "runID", "linc", "user", "type"
func (p *ProcessorPool) markerForTheAnnotation(ctx context.Context, marker string, title string, runID int32) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "markerForTheAnnotation failed: '%+v'", err)
		}
	}()
	logger.Warnf(ctx, "MarkerForTheAnnotation: %v -> %v", marker, title)

	link := undecided.RunLink(ctx, runID)
	logger.Warnf(ctx, "LoadTesting %v '%v' link: %v", marker, title, link)

	gauge, status := promutil.GetGaugeAnnotation(ctx)
	logger.Infof(ctx, "GetGauge Marker{%s}'%v':%v {%+v}", marker, title, runID, gauge)
	if !status {
		logger.Warnf(ctx, "Get GaugeAnnotation{%s} miss{%v:%+v} %s", marker, status, gauge, title)

		return
	}

	switch marker {
	case markerStart:
		gauge.WithLabelValues(title, strconv.Itoa(int(runID)), link, "--", "--").Inc()
	case markerStop:
		logger.Infof(ctx, "Marker stop '%v':%v", title, runID)
		gauge.WithLabelValues(title, strconv.Itoa(int(runID)), link, "--", "--").Dec()
	}

	logger.Infof(ctx, "Marker %v completed '%v:%v'", marker, title, runID)
}

// Выставление статуса "STATUS_RUNNING" сценарию только в случае, когда все скрипты в статусе "STATUS_RUNNING"
func (p *ProcessorPool) setTheRunScenarioStatusRunning(ctx context.Context, running *pb.Run) {
	logger.Infof(ctx, "Preparation RunScenario: '%v', Status: '%v', ", running.RunId, running.Status)

	inProgress := false

	for _, runScr := range running.GetScriptRuns() {
		if runScr.Status == pb.ScriptRun_STATUS_RUNNING { // Если хоть один запущен, считаем что сценарий запущен
			inProgress = true
		} else {
			logger.Infof(ctx, "Status preparation(RunScriptId:'%v'; Status:'%v')",
				runScr.RunScriptId, runScr.Status)
		}
	}

	if inProgress {
		running.Status = pb.Run_STATUS_RUNNING
		logger.Infof(ctx, "The whole Run is running correctly: '%v'", running.RunId)
	} else {
		running.Status = pb.Run_STATUS_STOPPED_UNSPECIFIED
		running.Info = fmt.Sprint(running.Info, "No scripts running ")
		logger.Infof(ctx, "Run not running '%v'", running.GetRunId())
	}
}

func (p *ProcessorPool) setTheScriptRunStatusRunning(ctx context.Context,
	scriptRunMessage string, scriptNumber int, scriptRun *pb.ScriptRun) (message string) {
	if scriptRunMessage != "" && strings.Contains(scriptRunMessage, "level=error") {
		// message = "ScriptNumber: '№" + strconv.Itoa(scriptNumber) + "' Script Run ID:'" +//fixme:
		//	fmt.Sprint(scriptRun.RunScriptId) +
		//	"' '" + scriptRunMessage + "'"
		messageBuilder := strings.Builder{}
		messageBuilder.WriteString("ScriptNumber: '№")
		messageBuilder.WriteString(strconv.Itoa(scriptNumber))
		messageBuilder.WriteString("' Script Run ID:'")
		messageBuilder.WriteString(strconv.Itoa(int(scriptRun.RunScriptId)))
		messageBuilder.WriteString("' '")
		messageBuilder.WriteString(scriptRunMessage)
		messageBuilder.WriteString("'")
		logger.Warn(ctx, scriptRunMessage)

		message = messageBuilder.String()
		scriptRun.Status = pb.ScriptRun_STATUS_FAILED
		scriptRun.Info = fmt.Sprint(scriptRunMessage, "setting STATUS_FAILED ") //fixme: может дублироваться
	} else {
		logger.Debugf(ctx, "Status preparation RUNNING ScriptRun: '%v'; RunScriptId: '%v'; ScriptRun message: '%v';",
			scriptRun, scriptRun.RunScriptId, scriptRunMessage)

		scriptRun.Status = pb.ScriptRun_STATUS_RUNNING
		scriptRun.Info = fmt.Sprint(scriptRun.Info, scriptRunMessage)
	}

	logger.Infof(ctx, "Status ScriptRun('%v'): '%v'", scriptRun.GetRunScriptId(), scriptRun.GetStatus())

	return message
}

// SetTheRunScenarioStatusStopped Выставление статуса "остановлен" сценарию только в случае, когда все скрипты в статусе "остановлен"
func (p *ProcessorPool) SetTheRunScenarioStatusStopped(ctx context.Context, pbRun *pb.Run) (message string) {
	var countRunsScript = 0

	for _, scrRun := range pbRun.GetScriptRuns() {
		if scrRun.GetStatus() == pb.ScriptRun_STATUS_RUNNING {
			logger.Infof(ctx, "There is still a running script RunScriptId:'%v'; Pid:'%v'",
				scrRun.RunScriptId, scrRun.Pid)

			countRunsScript++
		}
	}
	// и выставляем статус Run_STOPPED для всего запуска
	if countRunsScript == 0 {
		pbRun.Status = pb.Run_STATUS_STOPPED_UNSPECIFIED
		logger.Infof(ctx, "Run stopped '%v'", pbRun.GetRunId())
	} else {
		logger.Warnf(ctx, "There is still a running script '%v'", countRunsScript)
		pbRun.Info = fmt.Sprint("'CountRunsScript: ", countRunsScript, "' ")
	}

	_, message = p.dbPB.UpdatePbRunningInTheDB(ctx, pbRun)

	return message
}

func (p *ProcessorPool) startingSimpleScript(ctx context.Context, cmdStartSimpleScript *models.Command) bool {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "startingSimpleScript failed: '%+v'", err)
		}
	}()
	p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, "")
	logger.Infof(ctx, "Cmd.Type:'%v' SimpleScriptIDs count:'%v' SimpleScriptIDs:'%v'",
		cmdStartSimpleScript.Type, len(cmdStartSimpleScript.ScriptIds), cmdStartSimpleScript.ScriptIds)

	mRun, err := p.db.GetMRunning(ctx, cmdStartSimpleScript.RunID)
	if err != nil {
		errorMessage := fmt.Sprintf("SimpleScriptRun GetMRunning message: '%v'", err.Error())
		p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	getScenario, err := p.db.GetMScenario(ctx, mRun.ScenarioID)
	if err != nil {
		errorMessage := fmt.Sprintf("SimpleScriptRun GetMScenario message: '%v'", err.Error())
		p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_FAILED, errorMessage)

		return false
	}

	for _, simpleScriptID := range cmdStartSimpleScript.ScriptIds {
		//nolint:gosec // rps < MaxInt32
		simpleScriptToRun, err2 := p.dbPB.GetPBSimpleScript(ctx, int32(simpleScriptID))
		if err2 != nil {
			logger.Errorf(ctx, "p.dbPB.GetPBSimpleScript error message: '%v'", err2.Error())

			if simpleScriptToRun == nil {
				logger.Warnf(ctx, "____SimpleScriptRun not found SimpleScriptID: '%v'", simpleScriptID)
			}

			errorMessage := fmt.Sprintf("SimpleScriptRun GetPBSimpleScript message: '%v'", err2.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		simpleScriptToRun.Tag, err = util.CheckingTagForPresenceInDB(ctx, simpleScriptToRun.Tag, p.db)
		if err != nil {
			errorMessage := fmt.Sprintf("SimpleScriptRun checkingTag message: '%v'", err.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)
			continue
		}

		freeAgent, err2 := p.agentManager.GetAFreeAgent(ctx, simpleScriptToRun.Tag)
		if err2 != nil {
			errorMessage := fmt.Sprintf("SimpleScriptRun GetAFreeAgent message: '%v'", err2.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		SimpleScriptTarget, err2 := strconv.ParseInt(simpleScriptToRun.Rps, 10, 64)
		if err2 != nil {
			errorMessage := fmt.Sprintf("startingSimpleScript script(%v). RPS conversion error: '%v'",
				simpleScriptToRun.GetScriptId(), err2.Error())
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		var pbScriptRunToRunning = &pb.ScriptRun{
			RunId:        cmdStartSimpleScript.RunID,
			RunScriptId:  math2.GetRandomID32(ctx),
			SimpleScript: simpleScriptToRun,
			Status:       pb.ScriptRun_STATUS_STOPPED_UNSPECIFIED,
			//nolint:gosec
			Target: int32(SimpleScriptTarget),
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
			Agent:         freeAgent,
		}

		p.loadValidation(ctx, cmdStartSimpleScript.PercentageOfTarget.Int32, pbScriptRunToRunning)
		simpleScriptRunTask := p.prepareTaskToRun(ctx, pbScriptRunToRunning, getScenario.Title)
		runTask := func() (*agentapi.StartResponse, error) {
			agent, err2 := p.agentManager.GetClientPB(ctx, pbScriptRunToRunning.Agent)
			if err2 != nil {
				logger.Errorf(ctx, "error GetClientPB %+v", err2)
				return nil, err2
			}

			return agent.Start(ctx, &agentapi.StartRequest{
				ScenarioTitle: simpleScriptRunTask.ScenarioTitle,
				ScriptTitle:   simpleScriptRunTask.ScriptTitle,
				ScriptURL:     simpleScriptRunTask.ScriptUrl,
				//Params:        simpleScriptRunTask.Params,

				ProjectId:   simpleScriptRunTask.ProjectId,
				ScenarioId:  simpleScriptRunTask.ScenarioId,
				ScriptId:    simpleScriptRunTask.ScriptId,
				RunId:       simpleScriptRunTask.RunId,
				ScriptRunId: simpleScriptRunTask.ScriptRunId,

				Envs: simpleScriptRunTask.Envs,
			})
		}

		rs, err2 := runTask()
		if err2 != nil {
			errorMessage := fmt.Sprintf(
				"SimpleScriptRun Execute request Run error '%v' SimpleScriptID:'%v'", err2, simpleScriptID)
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			mes := p.dbPB.UpdatePbScriptRunInDB(ctx, pbScriptRunToRunning.RunId, pbScriptRunToRunning)
			if mes != "" {
				logger.Errorf(ctx, "errorMessage:'%v', mes'%v'", errorMessage, mes)
			}

			continue
		}

		scriptRunMessage := p.processingTheResponseFromTheAgent(ctx, rs, pbScriptRunToRunning)
		if scriptRunMessage != "" {
			errorMessage := fmt.Sprintf(
				"SimpleScriptRun, processingTheResponseFromTheAgent: '%v'", scriptRunMessage)
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		statusMessage := p.setTheScriptRunStatusRunning(ctx, scriptRunMessage, -1, pbScriptRunToRunning)
		if statusMessage != "" {
			errorMessage := fmt.Sprintf("Setting the status message: '%v'", statusMessage)
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_PROCESSED, errorMessage)

			continue
		}

		message := p.dbPB.UpdatePbScriptRunInDB(ctx, cmdStartSimpleScript.RunID, pbScriptRunToRunning)
		if message != "" {
			errorMessage := fmt.Sprintf("Error update ScriptRunning in DB '%v'", message)
			p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_FAILED, errorMessage)

			continue
		}
	}

	p.db.UpdateStatusMCommand(ctx, cmdStartSimpleScript, models.CmdstatusSTATUS_COMPLETED, "")

	return p.db.DeleteMCmd(ctx, cmdStartSimpleScript)
}
