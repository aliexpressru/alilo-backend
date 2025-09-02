package processing

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	dataPb "github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	string2 "github.com/aliexpressru/alilo-backend/pkg/util/string"
	"github.com/pkg/errors"
)

func GetAllScripts(
	ctx context.Context,
	scenarioID int32,
	db *dataPb.Store,
) (returnScripts []*pb.Script, message string) {
	logger.Infof(ctx, "Get all scripts. Send data, scenarioID: '%+v'", scenarioID)

	returnScripts, err := db.GetAllPbScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Error fetch pbScripts: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnScripts, message
	}

	if len(returnScripts) > 0 {
		logger.Debugf(ctx, "Get all scripts. Prepared slice: '%+v'", returnScripts)

		for _, script := range returnScripts {
			logger.Debugf(ctx, "ScenarioID: '%+v'; Script: '%+v'", scenarioID, script)
		}
	} else {
		logger.Warnf(ctx, "mScriptsList is empty in scenario:'%v'", scenarioID)
		return returnScripts, message
	}

	return returnScripts, message
}

// GetAllEnabledScripts Возвращает массив включенных Script-ов
func GetAllEnabledScripts(
	ctx context.Context,
	scenarioID int32,
	db *dataPb.Store,
) (returnScripts []*pb.Script, err error) {
	logger.Infof(ctx, "Get all Active scripts. Send data, scenarioID: '%+v'", scenarioID)

	returnScripts, err = db.GetAllEnabledScripts(ctx, scenarioID)
	if err != nil {
		err = errors.Wrapf(err, "error fetch active mScriptsList in scenario:'%v'", scenarioID)
		logger.Warnf(ctx, "GetAllEnabledScripts error: %v", err)

		return returnScripts, err
	}

	if len(returnScripts) > 0 {
		logger.Infof(ctx, "Get all Active scripts. Prepared model len: '%+v'", len(returnScripts))

		for _, script := range returnScripts {
			logger.Debugf(ctx, "ScenarioID: '%+v'; Script: '%+v'", scenarioID, script)
		}
	} else {
		logger.Warnf(ctx, "Active mScriptsList is empty in scenario:'%v'", scenarioID)
	}

	return returnScripts, err
}

func CreateScript(
	ctx context.Context,
	script *pb.Script,
	db *dataPb.Store,
) (status bool, message string, scriptID int32) {
	util.TrimSpaceInScript(script)
	script.Name = string2.ReplaceAllUnnecessarySymbols(script.Name)
	tag, err2 := util.CheckingTagForPresenceInDB(ctx, script.GetTag(), db.GetDataStore())

	if err2 != nil {
		logger.Warnf(ctx, "Invalid tag specified: %v", script.GetTag())
		script.Tag = tag
	}

	script.GetOptions().Duration = util.CheckingNegativeValue(script.GetOptions().Duration)
	script.GetOptions().Steps = util.CheckingNegativeValue(script.GetOptions().Steps)
	script.GetOptions().Rps = util.CheckingNegativeValue(script.GetOptions().Rps)
	if !util.IsTimeUnit(string2.GetLastRune(script.Options.Duration, 1)) {
		return false, fmt.Sprintf("%v: %+v", "duration must contain a time unit", script.Options.Duration), -1
	}

	if script.AdditionalEnv == nil {
		script.AdditionalEnv = make(map[string]string)
	}
	scriptID, err := db.CreatePbScript(ctx, script)
	if err != nil {
		message = fmt.Sprintf("Error create script, CreatePbScript: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message, 0
	}

	return true, message, scriptID
}

func GetScript(ctx context.Context, scriptID int32, db *dataPb.Store) (script *pb.Script, message string) {
	pbScript, err := db.GetPbScript(ctx, scriptID)
	if err != nil {
		message = fmt.Sprintf("Error get pbScript: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return nil, message
	}

	logger.Debugf(ctx, "Get pbScript. Prepared model: '%+v'", pbScript)

	return pbScript, message
}

func UpdateScript(ctx context.Context, script *pb.Script, db *data.Store) (status bool, message string) {
	tag, err2 := util.CheckingTagForPresenceInDB(ctx, script.GetTag(), db)
	script.Name = string2.ReplaceAllUnnecessarySymbols(script.Name)

	if err2 != nil {
		logger.Warnf(ctx, "Invalid tag specified: %v", script.GetTag())
		script.Tag = tag
	}

	logger.Debugf(ctx, "Update script. Prepared model: '%+v'", script)

	util.TrimSpaceInScript(script)

	script.GetOptions().Duration = util.CheckingNegativeValue(script.GetOptions().Duration)
	script.GetOptions().Steps = util.CheckingNegativeValue(script.GetOptions().Steps)
	script.GetOptions().Rps = util.CheckingNegativeValue(script.GetOptions().Rps)

	if !util.IsTimeUnit(string2.GetLastRune(script.Options.Duration, 1)) {
		_ = util.SetModalHeader(ctx)

		return false, fmt.Sprintf("%v: %+v", "duration must contain a time unit", script.Options.Duration)
	}
	if script.AdditionalEnv == nil {
		script.AdditionalEnv = make(map[string]string)
	}

	serverScript, message := conv.PbToModelScript(ctx, script)
	if message != "" {
		message = fmt.Sprintf("Error converting PBScript to ModelScript: '%v'", message)
		logger.Errorf(ctx, message)
		_ = util.SetModalHeader(ctx)

		return false, message
	}
	prevScript, err := db.GetMScript(ctx, serverScript.ScriptID)
	if err != nil {
		_ = util.SetModalHeader(ctx)

		return false, fmt.Sprintf("Failed to get script with id: %d", serverScript.ScriptID)
	}
	if prevScript.Enabled == serverScript.Enabled {
		err = util.SetModalHeader(ctx)
		if err != nil {
			return false, err.Error()
		}
	}

	err = db.UpdateMScript(ctx, serverScript)
	if err != nil {
		message = fmt.Sprintf("Error update script, update to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	message = fmt.Sprintf("The script %s was updated!", script.Name)
	return true, message
}

func DeleteScript(ctx context.Context, scriptID int32, db *data.Store) (status bool, message string) {
	logger.Infof(ctx, "DeleteScript: %v", scriptID)

	err := db.DeleteMScript(ctx, scriptID)
	if err != nil {
		err = errors.Wrapf(err, "DeleteMSimpleScript error:")
		logger.Errorf(ctx, "Error deleting the script{%v}", err.Error())

		return false, err.Error()
	}

	message = "Script was deleted!"
	return true, message
}

func MoveScript(
	ctx context.Context,
	scriptID int32,
	scenarioID int32,
	projectID int32,
	db *data.Store,
) (status bool, message string) {
	logger.Infof(ctx, "MoveScript: ", scriptID)

	mScript, err := db.FindScript(ctx, scriptID)
	if err != nil {
		message = fmt.Sprintf("Error find script for move: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	logger.Debugf(ctx, "Move script. Prepared model: '%+v'", mScript)

	if scenarioID != 0 {
		mScript.ScenarioID = scenarioID
	}

	if projectID != 0 {
		mScript.ProjectID = projectID
	}

	err = db.UpdateMScript(ctx, mScript)
	if err != nil {
		message = fmt.Sprintf("Error move script, update to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	return true, message
}

func CopyScript(ctx context.Context, sourceScriptID int32, targetScenarioID int32, db *data.Store) (message string) {
	logger.Infof(ctx, "Copy script. Send data: '%+v : '%+v'", sourceScriptID, targetScenarioID)

	targetMScenario, err := db.GetMScenario(ctx, targetScenarioID)
	if err != nil {
		message = fmt.Sprintf("Error select target MScenario: %v", err.Error())
		logger.Warnf(ctx, message)

		return message
	}

	mScript, err := db.GetMScript(ctx, sourceScriptID)
	if err != nil {
		message = fmt.Sprintf("Error select mScript: %v", err.Error())
		logger.Warnf(ctx, message)

		return message
	}

	logger.Infof(ctx, "Scripts to copy: ", mScript.ProjectID, mScript.ScenarioID, mScript.Name)

	mScript.ScenarioID = targetMScenario.ScenarioID
	mScript.ProjectID = targetMScenario.ProjectID
	mScript.Name = fmt.Sprint("Copied_", mScript.Name)

	_, err = db.CreateMScript(ctx, mScript)
	if err != nil {
		message = fmt.Sprintf("Error create script. Insert to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return message
	}

	return message
}

// MessageProcessing Собирается читабельное сообщения для возвращения пользователю при запросе списка скриптов всех типов
func MessageProcessing(ctx context.Context, sMessage string, sSMessage string) (message string) {
	if sMessage == "" && sSMessage == "" {
		return message
	}
	message = fmt.Sprintf("scrMessage(%v) sScrMessage(%v)", sMessage, sSMessage)

	logger.Infof(ctx, "Return message: '%v'", message)

	return message
}

// CheckStatusGetAllScripts Анализ сообщений для проставления валидного статуса при запросе списка скриптов
func CheckStatusGetAllScripts(ctx context.Context, scriptMessage string, simpleScriptMessage string) (status bool) {
	status = true
	if scriptMessage == "" || simpleScriptMessage == "" {
		return status
	} else if !strings.Contains(scriptMessage, "is empty in scenario:'") {
		status = false
	} else if !strings.Contains(simpleScriptMessage, "is empty in scenario:'") {
		status = false
	}

	logger.Infof(ctx, "Еhe absence of scripts is not considered an error")

	return status
}
