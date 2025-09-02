package processing

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	string2 "github.com/aliexpressru/alilo-backend/pkg/util/string"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetAllSimpleScript(ctx context.Context, scenarioID int32, db *datapb.Store) (
	returnSimpleScripts []*pb.SimpleScript, message string) {
	logger.Infof(ctx, "Get all simpleScripts. Send data, scenarioID: '%+v'", scenarioID)

	returnSimpleScripts, err := db.GetAllPBSimpleScripts(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Error fetch All simpleScripts: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnSimpleScripts, message
	}

	if len(returnSimpleScripts) == 0 {
		message = fmt.Sprintf("returnSimpleScripts is empty in scenario:'%v'", scenarioID)
		logger.Warnf(ctx, message)
	}

	return returnSimpleScripts, message
}

func CreateSimpleScript(ctx context.Context, sScript *pb.SimpleScript, db *datapb.Store) (
	message string, simpleScriptID int32) {
	if sScript == nil {
		return "incorrect value SimpleScript", -1
	}
	util.TrimSpaceInSimpleScript(sScript)
	sScript.Name = string2.ReplaceAllUnnecessarySymbols(sScript.Name)
	tag, err2 := util.CheckingTagForPresenceInDB(ctx, sScript.GetTag(), db.GetDataStore())

	if err2 != nil {
		logger.Warnf(ctx, "Invalid tag specified: %v", sScript.GetTag())
		sScript.Tag = tag
	}

	if sScript.IsStaticAmmo && !util.CheckingStaticAmmoLength(ctx, len(sScript.StaticAmmo)) {
		mess := fmt.Sprintf("Static AMMO are too long '%v'", len(sScript.StaticAmmo))
		logger.Warnf(ctx, "CheckingStaticAmmoLength: %v", mess)

		return mess, -2
	}
	sScript.Steps = util.CheckingNegativeValue(sScript.Steps)
	sScript.Duration = util.CheckingNegativeValue(sScript.Duration)
	sScript.Rps = util.CheckingNegativeValue(sScript.Rps)
	sScript.MaxVUs = util.CheckingNegativeValue(sScript.MaxVUs)
	if !util.IsTimeUnit(string2.GetLastRune(sScript.Duration, 1)) {
		return fmt.Sprintf("%v: %+v", "duration must contain a time unit", sScript.Duration), -1
	}
	if sScript.AdditionalEnv == nil {
		sScript.AdditionalEnv = make(map[string]string)
	}
	err := SimpleScriptGenerate(ctx, sScript, db.GetDataStore())
	if err != nil {
		message = fmt.Sprintf("Error generate SimpleScript by create: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return message, -1
	}

	ScriptID, err := db.CreatePBSimpleScript(ctx, sScript)
	if err != nil {
		message = fmt.Sprintf("%v Error create SimpleScript: '%v'", message, err.Error())
		logger.Errorf(ctx, message)
	}

	return message, ScriptID
}

func GetSimpleScript(ctx context.Context, simpleScriptID int32, db *datapb.Store) (
	simpleScript *pb.SimpleScript, message string) {
	simpleScript, err := db.GetPBSimpleScript(ctx, simpleScriptID)
	if err != nil {
		logger.Warnf(ctx, "%v", err)
		return simpleScript, err.Error()
	}

	logger.Debugf(ctx, "Get simpleScript. Prepared model: '%+v'", simpleScript)

	return simpleScript, message
}

func UpdateSimpleScript(ctx context.Context, simpleScript *pb.SimpleScript, db *datapb.Store) (err error) {
	util.TrimSpaceInSimpleScript(simpleScript)
	simpleScript.Name = string2.ReplaceAllUnnecessarySymbols(simpleScript.Name)

	tag, err2 := util.CheckingTagForPresenceInDB(ctx, simpleScript.GetTag(), db.GetDataStore())
	if err2 != nil {
		logger.Warnf(ctx, "Invalid tag specified: %v", simpleScript.GetTag())
		simpleScript.Tag = tag
	}

	if simpleScript.IsStaticAmmo && !util.CheckingStaticAmmoLength(ctx, len(simpleScript.StaticAmmo)) {
		return fmt.Errorf("static AMMO are too long")
	}
	simpleScript.Steps = util.CheckingNegativeValue(simpleScript.Steps)
	simpleScript.Duration = util.CheckingNegativeValue(simpleScript.Duration)
	simpleScript.Rps = util.CheckingNegativeValue(simpleScript.Rps)
	simpleScript.MaxVUs = util.CheckingNegativeValue(simpleScript.MaxVUs)
	if !util.IsTimeUnit(string2.GetLastRune(simpleScript.Duration, 1)) {
		return status.Errorf(
			codes.InvalidArgument,
			"%v: %+v",
			"duration must contain a time unit",
			simpleScript.Duration,
		)
	}
	if !simpleScript.IsStaticAmmo {
		simpleScript.StaticAmmo = ""
	}
	if simpleScript.AdditionalEnv == nil {
		simpleScript.AdditionalEnv = make(map[string]string)
	}

	err = SimpleScriptGenerate(ctx, simpleScript, db.GetDataStore())
	if err != nil {
		err = errors.Wrap(err, "Error generate SimpleScript by update")
		logger.Errorf(ctx, err.Error())

		return err
	}

	err = db.UpdatePbSimpleScript(ctx, simpleScript)
	if err != nil {
		err = errors.Wrapf(err, "Error update to db simpleScript")
		logger.Errorf(ctx, err.Error())

		return err
	}

	return err
}

func DeleteSimpleScript(ctx context.Context, simpleScriptID int32, db *datapb.Store) error {
	logger.Infof(ctx, "processing deleteSimpleScript: %v", simpleScriptID)
	err := db.GetDataStore().DeleteMSimpleScript(ctx, simpleScriptID)
	if err != nil {
		return err
	}

	err = util.SetModalHeader(ctx)

	return err
}

func MoveSimpleScript(ctx context.Context, simpleScriptID int32, scenarioID int32, db *datapb.Store) (
	status bool, message string) {
	logger.Infof(ctx, "MoveSimpleScript: ", simpleScriptID)

	if scenarioID == 0 {
		return false, "The scenario ID cannot be 0"
	}

	mScenario, err := db.GetDataStore().GetMScenario(ctx, scenarioID)
	if err != nil {
		message = fmt.Sprintf("Error find mScenario for move SimpleScript: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	pbSimpleScript, err := db.GetPBSimpleScript(ctx, simpleScriptID)
	if err != nil {
		message = fmt.Sprintf("Error find SimpleScript for move: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	logger.Debugf(ctx, "Move SimpleScript. Prepared model SimpleScript: '%+v'", pbSimpleScript)

	pbSimpleScript.ProjectId = mScenario.ProjectID
	pbSimpleScript.ScenarioId = mScenario.ScenarioID

	err = SimpleScriptGenerate(ctx, pbSimpleScript, db.GetDataStore())
	if err != nil {
		err = errors.Wrap(err, "Error generate SimpleScript")
		logger.Errorf(ctx, err.Error())

		return false, err.Error()
	}

	err = db.UpdatePbSimpleScript(ctx, pbSimpleScript)
	if err != nil {
		message = fmt.Sprintf("Error move SimpleScript, update to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message
	}

	return true, message
}

func CopySimpleScript(ctx context.Context, sourceSimpleScriptID int32, targetScenarioID int32, db *datapb.Store) (
	message string, simpleScriptID int32) {
	logger.Infof(ctx, "Copy SimpleScript. Send data: '%+v': '%+v'", sourceSimpleScriptID, targetScenarioID)

	targetMScenario, err := db.GetDataStore().GetMScenario(ctx, targetScenarioID)
	if err != nil {
		message = fmt.Sprintf("Error select target MScenario: %v", err.Error())
		logger.Warnf(ctx, message)

		return message, -1
	}

	mSimpleScript, err := db.GetDataStore().GetMSimpleScript(ctx, sourceSimpleScriptID)
	if err != nil {
		message = fmt.Sprintf("Error select mSimpleScript(%v): %v", sourceSimpleScriptID, err.Error())
		logger.Warnf(ctx, message)

		return message, -1
	}

	logger.Infof(ctx, "Scripts to copy: ProjectID(%+v) ScenarioID(%+v) Name(%+v) ScriptID(%+v)",
		mSimpleScript.ProjectID, mSimpleScript.ScenarioID, mSimpleScript.Name, mSimpleScript.ScriptID)

	mSimpleScript.ScenarioID = targetMScenario.ScenarioID
	mSimpleScript.ProjectID = targetMScenario.ProjectID
	mSimpleScript.Name = fmt.Sprint("Copied_", mSimpleScript.Name)

	scriptID, err := db.GetDataStore().CreateMSimpleScript(ctx, mSimpleScript)
	if err != nil {
		message = fmt.Sprintf("Error create SimpleScript. Insert to db: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return message, scriptID
	}

	logger.Infof(ctx, "Script coped: ProjectID(%+v) ScenarioID(%+v) Name(%+v) ScriptID(%+v)",
		mSimpleScript.ProjectID, mSimpleScript.ScenarioID, mSimpleScript.Name, scriptID)

	return message, scriptID
}
