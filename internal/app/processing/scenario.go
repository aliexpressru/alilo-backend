package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/aarondl/null/v8"
	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	strUtils "github.com/aliexpressru/alilo-backend/pkg/util/string"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
)

const invalidNameMessage = "The name is not valid"

func GetAllScenarios(ctx context.Context, projectID int32, limit int32, pageNumber int32, store *datapb.Store) (
	scenarios []*pb.Scenario, pages int64, message string) {
	countProjects, _ := store.GetDataStore().GetCountScenarios(ctx, projectID)
	if limit < 1 {
		//nolint:gosec
		limit = int32(countProjects)
	}

	return store.GetAllScenarios(ctx, projectID, limit, pageNumber)
}

func CreateScenario(ctx context.Context, scenario *pb.Scenario, dataStore *data.Store) (
	status bool, message string, scenarioID int32) {
	logger.Infof(ctx, "Create scenario. Send data: '%+v'", scenario)
	// T.к. имя сценария передается в скрипт для запуска и используется в структуре JSON скрипта,
	// которая не позволяет цифровой префикс без кавычек:
	if CheckingScenarioTitle(ctx, scenario.GetTitle()) {
		return false, invalidNameMessage, -1
	}

	if scenario.Title == "" {
		message = fmt.Sprintf("Error create scenario: '%v'", "Title is empty")
		logger.Error(ctx, message)

		return false, message, scenarioID
	} else if scenario.ProjectId == 0 {
		message = fmt.Sprintf("Error create scenario: '%v'", "ProjectId is empty")
		logger.Error(ctx, message)

		return false, message, scenarioID
	}

	scenario.Title = strUtils.ReplaceAllUnnecessarySymbols(scenario.GetTitle())

	mScenario, message := conv.PBToModelScenario(ctx, scenario)
	if message != "" {
		message = fmt.Sprintf("Error create scenario,  convert pbScenario to model: '%v'", message)
		logger.Errorf(ctx, message)

		return false, message, scenarioID
	}

	logger.Infof(ctx, "Create scenario. Prepared model: '%+v'", mScenario)

	mScenario, err := dataStore.CreateMScenario(ctx, mScenario)
	if err != nil {
		message = fmt.Sprintf("Error create scenario, insert to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message, mScenario.ScenarioID
	}

	return true, message, mScenario.ScenarioID
}

func UpdateScenario(ctx context.Context, scenario *pb.Scenario, pbStore *datapb.Store) (status bool, message string) {
	logger.Infof(ctx, "Update scenario. Send data: '%+v'", scenario)
	// т.к. имя сценария передается в скрипт для запуска и используется в структуре JSON скрипта, которая не позволяет цифровой префикс
	if CheckingScenarioTitle(ctx, scenario.GetTitle()) {
		return false, invalidNameMessage
	}

	scenario.Title = strUtils.ReplaceAllUnnecessarySymbols(scenario.GetTitle())

	err := pbStore.UpdateScenario(ctx, scenario)
	if err != nil {
		message = fmt.Sprintf("Error update scenario, insert to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}
	message = fmt.Sprintf("Scenario %s successfully updated", scenario.GetTitle())
	return true, message
}

func DeleteScenario(ctx context.Context, scenarioID int32, dataStore *data.Store) (status bool, message string) {
	logger.Infof(ctx, "Delete scenario. Send data: '%+v'", scenarioID)

	mScenario, err := dataStore.GetMScenario(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Error delete scenario, get to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	logger.Infof(ctx, "Delete scenario. Prepared model: '%+v'", mScenario)

	if er := dataStore.DeleteMScenario(ctx, mScenario); er != nil {
		message = fmt.Sprintf("Error delete scenario, delete to db: '%v'", er.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	return true, message
}

func CopyScenario(
	ctx context.Context,
	sourceScenarioID int32,
	targetScenarioID int32,
	dataStore *data.Store,
) (message string) {
	logger.Infof(ctx, "Copy scenario. Send data: '%+v : '%+v'", sourceScenarioID, targetScenarioID)

	targetMScenario, err := dataStore.GetMScenario(ctx, targetScenarioID)
	if err != nil {
		message = fmt.Sprintf("Error get target MScenario: %v", err.Error())
		logger.Warnf(ctx, message)

		return message
	}

	mScripts, err := dataStore.GetAllMScripts(ctx, sourceScenarioID)
	if err != nil {
		message = fmt.Sprintf("Error select mScripts: %v", err.Error())
		logger.Warnf(ctx, message)

		return message
	}

	logger.Infof(ctx, "len scripts to copy %v", len(mScripts))

	for _, mScript := range mScripts {
		mScript.ScenarioID = targetMScenario.ScenarioID
		mScript.ProjectID = targetMScenario.ProjectID
		mScript.Name = fmt.Sprint("Copied_", mScript.Name)

		_, err = dataStore.CreateMScript(ctx, mScript)
		if err != nil {
			message = fmt.Sprintf("Error create script, insert to db: '%v'", err.Error())
			logger.Errorf(ctx, message)

			return message
		}
	}

	mSimpleScripts, err := dataStore.GetAllMSimpleScripts(ctx, sourceScenarioID)
	if err != nil {
		message = fmt.Sprintf("Error select mSimpleScripts: %v", err.Error())
		logger.Warnf(ctx, message)

		return message
	}

	logger.Infof(ctx, "len mSimpleScripts to copy", len(mScripts))

	for _, mSimpleScript := range mSimpleScripts {
		mSimpleScript.ScenarioID = targetMScenario.ScenarioID
		mSimpleScript.ProjectID = targetMScenario.ProjectID
		mSimpleScript.Name = fmt.Sprint("Copied_", mSimpleScript.Name)

		_, err = dataStore.CreateMSimpleScript(ctx, mSimpleScript)
		if err != nil {
			message = fmt.Sprintf("Error create mSimpleScript, insert to db: '%v'", err.Error())
			logger.Errorf(ctx, message)

			return message
		}
	}

	return message
}

func CheckingScenarioTitle(ctx context.Context, title string) bool {
	prefix := title[0:1]
	logger.Infof(ctx, "Title prefix: v '%v'", prefix)

	if _, err := strconv.Atoi(prefix); err == nil {
		return true
	}

	return false
}

func SetDuration(ctx context.Context,
	projectID int32, scenarioID int32, duration string, dataStore *data.Store, store *datapb.Store) (
	status bool, message string) {
	logger.Infof(ctx, "Set duration scenario. Send data{projectID:'%v'; scenarioID:'%v'}", projectID, scenarioID)

	status = true

	if projectID != 0 {
		scenarios, err := dataStore.GetAllMScenarios(ctx, projectID)
		if err != nil {
			logger.Errorf(ctx, "SetDuration: error getting mScenarios: %v", err)

			return false, err.Error()
		}

		logger.Infof(ctx, "Count scenarios to set duration: %v", len(scenarios))

		for _, scenario := range scenarios {
			stat, mes := SetDurationScenario(ctx, scenario.ScenarioID, duration, dataStore, store)
			if !stat {
				status = stat
				message = fmt.Sprintf("%vScenarioID:{%v}, Message:{%v}; ", message, scenario.ScenarioID, mes)
			}
		}

		return status, message
	}

	return SetDurationScenario(ctx, scenarioID, duration, dataStore, store)
}

func SetDurationScenario(ctx context.Context,
	scenarioID int32, duration string, dataStore *data.Store, pbStore *datapb.Store) (status bool, message string) {
	logger.Infof(ctx, "Set duration scenario. Send data: '%+v'", scenarioID)

	scriptSlice, err := dataStore.GetAllMScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Set duration scenario. Error select scriptSlice: %v", err.Error())
		logger.Warnf(ctx, message)

		return false, message
	}

	for _, mScript := range scriptSlice {
		mScript.OptionsDuration = null.StringFrom(duration)
		err = dataStore.UpdateMScript(ctx, mScript)
		if err != nil {
			message = fmt.Sprintf("Error Set duration mScript, Update to db: '%v'", err.Error())
			logger.Errorf(ctx, message, err)

			return false, message
		}
	}

	pbSimpleScriptSlice, err := pbStore.GetAllPBSimpleScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Set duration scenario. Error select pbSimpleScriptSlice: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	var wg sync.WaitGroup // что бы не ждать последовательно каждую генерацию, делаем параллельно

	for _, pbSimpleScript := range pbSimpleScriptSlice {
		pbSimpleScript.Duration = duration
		go func(pbSimScript *pb.SimpleScript, wg *sync.WaitGroup) {
			defer func(wg *sync.WaitGroup) {
				logger.Warnf(ctx, "WaitGroup -1!")
				wg.Done()
			}(wg)
			wg.Add(1)

			er := SimpleScriptGenerate(ctx, pbSimScript, dataStore)
			if er != nil {
				logger.Errorf(ctx, "Error generate pbSimpleScript by set duration scenario: '%v'", er.Error())
			}
		}(pbSimpleScript, &wg)

		err = pbStore.UpdatePbSimpleScript(ctx, pbSimpleScript)
		if err != nil {
			message = fmt.Sprintf("Error Set duration pbSimpleScript, Update to db: '%v'", err.Error())
			logger.Errorf(ctx, message, err)

			return false, message
		}
	}

	wg.Wait()
	logger.Warnf(ctx, "Wait group 'SetDuration' is done. scenarioID:{%v}", scenarioID)

	return true, message
}

func SetSteps(ctx context.Context,
	projectID int32, scenarioID int32, steps string, dataStore *data.Store, pbStore *datapb.Store) (
	status bool, message string) {
	logger.Infof(ctx, "Set steps. {projectID: %v; scenarioID: '%v'; steps: '%v'}", projectID, scenarioID, steps)

	status = true

	if projectID != 0 {
		scenarios, err := dataStore.GetAllMScenarios(ctx, projectID)
		if err != nil {
			logger.Errorf(ctx, "SetSteps error getting mScenarios: %v", err)

			return false, err.Error()
		}

		logger.Infof(ctx, "Count scenarios to set steps: %v", len(scenarios))

		for _, scenario := range scenarios {
			stat, mes := SetStepsScenario(ctx, scenario.ScenarioID, steps, dataStore, pbStore)
			if !stat {
				status = false
				message = fmt.Sprintf(
					"SetSteps:{Message:{%v} ScenarioID:{%v}, Message:{%v}} ",
					message,
					scenario.ScenarioID,
					mes,
				)
			}
		}

		return status, message
	}

	return SetStepsScenario(ctx, scenarioID, steps, dataStore, pbStore)
}

func SetStepsScenario(ctx context.Context,
	scenarioID int32, steps string, dataStore *data.Store, pbStore *datapb.Store) (status bool, message string) {
	logger.Infof(ctx, "Set steps scenario. scenarioID: '%+v' steps: '%v'", scenarioID, steps)

	scriptSlice, err := dataStore.GetAllMScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Set steps scenario. Error select scriptSlice: %v", err.Error())
		logger.Warnf(ctx, message)

		return false, message
	}

	for _, mScript := range scriptSlice {
		toI, err2 := strconv.Atoi(steps)
		if err2 != nil {
			message = fmt.Sprintf("Error conversion steps{%v} mScript:{%v}", steps, err2.Error())
			logger.Errorf(ctx, message, err2)

			return false, message
		}

		mScript.OptionsSteps = null.IntFrom(toI)

		err2 = dataStore.UpdateMScript(ctx, mScript)
		if err2 != nil {
			message = fmt.Sprintf("Error Set steps mScript, Update to db: '%v'", err2.Error())
			logger.Errorf(ctx, message, err2)

			return false, message
		}
	}

	pbSimpleScriptSlice, err := pbStore.GetAllPBSimpleScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Set steps scenario. Error select pbSimpleScriptSlice: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	var wg sync.WaitGroup // что бы не ждать последовательно каждую генерацию для симплСкрипта, делаем параллельно

	for _, pbSimpleScript := range pbSimpleScriptSlice {
		pbSimpleScript.Steps = steps
		go func(pbSimScript *pb.SimpleScript, wg *sync.WaitGroup) {
			defer func(wg *sync.WaitGroup) {
				logger.Warnf(ctx, "WaitGroup -1!")
				wg.Done()
			}(wg)
			wg.Add(1)

			er := SimpleScriptGenerate(ctx, pbSimScript, dataStore)
			if er != nil {
				logger.Errorf(ctx, "Error generate pbSimpleScript by set steps scenario: '%v'", er.Error())
			}
		}(pbSimpleScript, &wg)

		err = pbStore.UpdatePbSimpleScript(ctx, pbSimpleScript)
		if err != nil {
			message = fmt.Sprintf("Error Set steps pbSimpleScript, Update to db: '%v'", err.Error())
			logger.Errorf(ctx, message, err)

			return false, message
		}
	}

	wg.Wait()
	logger.Warnf(ctx, "Wait group 'Set steps{%v}' is done. scenarioID:{%v}", steps, scenarioID)

	return true, message
}

func AmountRPS(ctx context.Context, runID int32, store *data.Store) (int64, error) {
	mScripts, err := store.GetAllMScripts(ctx, runID)
	if err != nil {
		logger.Warnf(ctx, "The error of getting a Run: %+v", err)
		return -1, err
	}
	mSiScripts, err := store.GetAllMSimpleScripts(ctx, runID)
	if err != nil {
		logger.Warnf(ctx, "The error of getting a Run: %+v", err)
		return -1, err
	}
	rps := int64(0)
	for _, script := range mScripts {
		if !script.Enabled {
			continue
		}
		rps = rps + int64(script.OptionsRPS.Int)
	}
	for _, script := range mSiScripts {
		if !script.Enabled {
			continue
		}
		var i int64
		i, err = strconv.ParseInt(script.RPS, 10, 64)
		if err != nil {
			continue
		}
		rps = rps + i
	}
	return rps, nil
}

func SetTeg(ctx context.Context, scenarioID int32, teg string, dataStore *data.Store) (status bool, err error) {
	exist, err := dataStore.GetExistEnabledMAgentsByTag(ctx, teg)
	if !exist || err != nil {
		return false, fmt.Errorf("no active agents were found for the %v tag", teg)
	}
	mScripts, err := dataStore.GetAllMScripts(ctx, scenarioID)
	if err != nil {
		return false, err
	}
	for _, mScript := range mScripts {
		mScript.Tag = teg
		if mScript.AdditionalEnv == "" {
			mScript.AdditionalEnv = "{}"
		}
		err2 := dataStore.UpdateMScript(ctx, mScript)
		if err2 != nil {
			return false, err2
		}
	}

	mSScripts, err := dataStore.GetAllMSimpleScripts(ctx, scenarioID)
	if err != nil {
		return false, err
	}
	for _, mSScript := range mSScripts {
		mSScript.Tag = teg
		err = dataStore.UpdateMSimpleScript(ctx, mSScript)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func GenerateGrafanaStructure(ctx context.Context, scenarioID int32, dataStore *data.Store) (string, error) {
	scenario, err := dataStore.GetMScenario(ctx, scenarioID)
	if err != nil {
		return "", fmt.Errorf("failed to get scenario when generate grafana structure: %v", err)
	}

	project, err := dataStore.GetMProject(ctx, scenario.ProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project when generate grafana structure: %v", err)
	}

	builder, err := util.NewDashBoardBuilder(scenario.Title, scenario.Descrip.String)
	if err != nil {
		return "", fmt.Errorf("failed to create a new dashboard when generate grafana structure: %v", err)
	}

	scripts, err := dataStore.GetAllMScriptsWithExpr(ctx, scenarioID)
	if err != nil {
		return "", fmt.Errorf("failed to get scripts when generate grafana structure %v", err)
	}

	// todo rework via generic
	yPos := uint32(0) // Start with Y position 1
	for _, script := range scripts {
		builder.WithRow(dashboard.NewRowBuilder(fmt.Sprintf("%s / %s", project.Title, script.Name)))
		xPos := uint32(0)
		builder.ScriptTimeSeriesRpsPanel(script, xPos, yPos)

		xPos += 8
		builder.ScriptTimeSeriesRTPanel(script, xPos, yPos)

		xPos += 8
		builder.ScriptTimeSeriesErrPanel(script, xPos, yPos)

		xPos = 0                   // reset counter
		yPos += util.DefaultHeight // Move to the next row after adding all panels for this script
	}
	yPos += util.DefaultHeight // Move to the next row after adding all panels for this script

	sScripts, err := dataStore.GetAllMSimpleScriptsWithExpr(ctx, scenarioID)
	if err != nil {
		return "", fmt.Errorf("failed to get simple scripts when generate grafana structure %v", err)
	}
	for _, script := range sScripts {
		builder.WithRow(dashboard.NewRowBuilder(fmt.Sprintf("%s / %s", project.Title, script.Name)))

		xPos := uint32(0)
		builder.SimpleScriptTimeSeriesRpsPanel(script, xPos, yPos)

		xPos += 8
		builder.SimpleScriptTimeSeriesRTPanel(script, xPos, yPos)

		xPos += 8
		builder.SimpleScriptTimeSeriesErrPanel(script, xPos, yPos)

		xPos = 0                   // reset counter
		yPos += util.DefaultHeight // Move to the next row after adding all panels for this script
	}

	sampleDashboard, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build grafana dashboard when generate grafana structure %v", err)
	}
	boardJSON, err := json.MarshalIndent(sampleDashboard, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal board to JSON: %v", err)
	}

	return string(boardJSON), nil
}

func GenerateGrafanaStructureByIds(
	ctx context.Context,
	scenarioIds []int32,
	title string,
	dataStore *data.Store,
) (string, error) {
	scenarios, err := dataStore.GetScenariosByIds(ctx, scenarioIds)
	if err != nil {
		return "", fmt.Errorf("failed to get scenarios when generate grafana structure %v", err)
	}

	// todo название - project title
	yPos := uint32(0)
	builder, err := util.NewDashBoardBuilder(title, "")
	if err != nil {
		return "", fmt.Errorf("failed to create dashboard builder: %v", err)
	}
	for _, scenario := range scenarios {
		// todo сделать одним запросом
		// todo тянуть simple scripts тоже
		scripts, sErr := dataStore.GetAllMScriptsWithExpr(ctx, scenario.ScenarioID)
		if sErr != nil {
			return "", fmt.Errorf("failed to get scripts for scenario %d: %v", scenario.ScenarioID, err)
		}

		builder.WithRow(dashboard.NewRowBuilder(scenario.Title))
		for _, script := range scripts {
			xPos := uint32(0)
			builder.ScriptTimeSeriesRpsPanel(script, xPos, yPos)
			xPos += 8

			builder.ScriptTimeSeriesRTPanel(script, xPos, yPos)
			xPos += 8

			builder.ScriptTimeSeriesErrPanel(script, xPos, yPos)

			xPos = 0 // reset counter
			yPos += util.DefaultHeight
		}

		sScripts, sErr := dataStore.GetAllMSimpleScriptsWithExpr(ctx, scenario.ScenarioID)
		if sErr != nil {
			return "", fmt.Errorf("failed to get simple scripts for scenario %d: %v", scenario.ScenarioID, err)
		}
		for _, script := range sScripts {
			yPos += util.DefaultHeight // Move to the next row after adding all panels for this script
			xPos := uint32(0)
			builder.SimpleScriptTimeSeriesRpsPanel(script, xPos, yPos)
			xPos += 8

			builder.SimpleScriptTimeSeriesRTPanel(script, xPos, yPos)
			xPos += 8

			builder.SimpleScriptTimeSeriesErrPanel(script, xPos, yPos)

			xPos = 0 // reset counter
		}
	}

	sampleDashboard, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build grafana dashboard when generate grafana structure %v", err)
	}
	boardJSON, err := json.MarshalIndent(sampleDashboard, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal board to JSON: %v", err)
	}

	return string(boardJSON), nil
}
