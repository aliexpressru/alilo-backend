package datapb

import (
	"context"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
)

// GetWorkingScriptRunsByScriptID функция возвращает только все ScriptRun`ы из Run`а соответствующие конкретному scriptID
func (s *Store) GetWorkingScriptRunsByScriptID(ctx context.Context, runID int32, scriptID int32) (
	returnActiveScriptsRun []*pb.ScriptRun, message string) {
	scriptsRuns, message := s.GetScriptRunsByScriptID(ctx, runID, scriptID)

	for _, scriptRun := range scriptsRuns {
		if scriptRun.Status == pb.ScriptRun_STATUS_RUNNING {
			returnActiveScriptsRun = append(returnActiveScriptsRun, scriptRun)
		}
	}

	return returnActiveScriptsRun, message
}

// GetScriptRunsByScriptID Функция возвращает все ScriptRun`ы из Run`а только соответствующие конкретному scriptID
func (s *Store) GetScriptRunsByScriptID(ctx context.Context, runID int32, scriptID int32) (
	returnScriptsRun []*pb.ScriptRun, message string) {
	pbRun, message := s.GetRunning(ctx, runID)
	if message == "" {
		for _, scrRun := range pbRun.GetScriptRuns() {
			var curScriptID int32

			switch scrRun.TypeScriptRun {
			case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
				curScriptID = scrRun.GetScript().GetScriptId()
			case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
				curScriptID = scrRun.GetSimpleScript().GetScriptId()
			}

			if curScriptID == scriptID {
				returnScriptsRun = append(returnScriptsRun, scrRun)
				logger.Infof(ctx,
					"GetScriptRunsByScriptID ----> GetScriptRuns() '%v' == '%v'", scrRun.RunScriptId, scriptID)
			}

			logger.Infof(ctx,
				"GetScriptRunsByScriptID -хx-> GetScriptRuns() '%v' != '%v'", scrRun.RunScriptId, scriptID)
		}
	}

	logger.Infof(ctx, "'%v' ScriptRun found", len(returnScriptsRun))

	return returnScriptsRun, message
}

// UpdatePbScriptRunInDB Обновляет или добавляет скриптРан в Run
// fixme: это очень долгая и дорогая функция, обновлять скрипт ран по отдельности, нужно переделать
// Deprecated: todo
func (s *Store) UpdatePbScriptRunInDB(ctx context.Context, runID int32, updatedScriptRun *pb.ScriptRun) (
	message string) {
	defer undecided.WarnTimer(
		ctx,
		fmt.Sprintf("UpdatePbScriptRunInDB{RunId: %v, RunScriptId: %v}", runID, updatedScriptRun.RunScriptId),
	)()
	run, message := s.GetRunning(ctx, runID)
	if message != "" {
		return message
	}

	needToAdd := true

	for i, scriptRun := range run.GetScriptRuns() {
		if scriptRun.GetRunScriptId() == updatedScriptRun.GetRunScriptId() {
			run.GetScriptRuns()[i] = updatedScriptRun
			logger.Debugf(ctx, "UpdatedScriptRun: '%+v'", run.GetScriptRuns()[i])

			needToAdd = false

			break
		}
	}

	if needToAdd {
		run.ScriptRuns = append(run.GetScriptRuns(), updatedScriptRun)
		logger.Infof(ctx, "UpdatePbScriptRunInDB Add ScriptRun '%v' to '%v'", updatedScriptRun.RunScriptId, run.RunId)
	}

	logger.Infof(ctx, "ScriptRuns len: '%v'", len(run.GetScriptRuns()))

	_, mes := s.UpdatePbRunningInTheDB(ctx, run)
	if mes != "" {
		message = fmt.Sprintf("Error ScriptStop, UpdateMRunningInTheDB(Error:'%v' Message:'%v')", mes, message)
		logger.Error(ctx, message)

		return message
	}

	return message
}

func (s *Store) UpdatePbRunningInTheDB(ctx context.Context, pbRun *pb.Run) (returnRun *pb.Run, message string) {
	var mRun *models.Run

	logger.Debugf(ctx, "UpdatePbRunningInTheDB pbRun: '%+v'", pbRun)
	mRun, message = conv.PbToModelRun(ctx, pbRun)
	logger.Debugf(ctx, "UpdatePbRunningInTheDB   mRun: '%+v'", mRun)

	if message == "" {
		var err error

		mRun, err = s.db.UpdateMRunningInTheDB(ctx, mRun)
		if err != nil {
			message = fmt.Sprintf("Error UpdatePbRunningInTheDB to db: '%v'", message)
			logger.Errorf(ctx, message, err)
		}

		returnRun, err = conv.ModelToPBRun(ctx, mRun)
		if err != nil {
			message = fmt.Sprintf("Error runModel to db: '%v'", err.Error())
			logger.Error(ctx, message, err)
		}

		logger.Debugf(ctx, "UpdatePbRunningInTheDB returnRun: '%+v'", returnRun)
	} else {
		logger.Errorf(ctx, "pbToModelRun message: '%v'", message)
	}

	return returnRun, message
}

// GetScriptRunning функция возвращает скрипт из Run`а по runScriptID
//
//	fixme: если в сценарии большое кол-во ScriptRuns поиск будет долгим
//	к примеру 1000, то циклом искать в массиве 1 не оптимально, переделать с быстрым поиском
func (s *Store) GetScriptRunning(ctx context.Context, runID int32, runScriptID int32) (
	returnScriptRun *pb.ScriptRun, message string) {
	run, message := s.GetRunning(ctx, runID)
	if message == "" {
		for _, scrRun := range run.GetScriptRuns() {
			if scrRun.RunScriptId == runScriptID {
				returnScriptRun = scrRun
				logger.Infof(ctx,
					"GetScriptRunning ----> GetScriptRuns() '%v' == '%v'", scrRun.RunScriptId, runScriptID)

				break
			}

			logger.Debugf(ctx,
				"GetScriptRunning -х--> GetScriptRuns() '%v' != '%v'", scrRun.RunScriptId, runScriptID)
		}
	}

	return returnScriptRun, message
}

func (s *Store) GetRunning(ctx context.Context, runID int32) (returnRun *pb.Run, errorMessage string) {
	if runID != 0 {
		mRun, err := s.db.GetMRunning(ctx, runID)
		if err != nil {
			err = errors.Wrapf(err, "Error GetMRunning: '%v'", runID)
			logger.Error(ctx, err)

			return returnRun, err.Error()
		}

		if mRun != nil {
			logger.Debugf(ctx, "Get run. Prepared model: '%+v';", mRun)
			logger.Debugf(ctx, "mProjectID: '%v'; mScenarioID: '%v'; mRunID: '%v'; mRun: '%+v';",
				mRun.ProjectID, mRun.ScenarioID, mRun.RunID, mRun)

			returnRun, err = conv.ModelToPBRun(ctx, mRun)
			if err != nil {
				logger.Errorf(ctx, "ModelToPBRun: '%v' '%v'", returnRun, err)
			}
		} else {
			errorMessage = "GetRunning. mRun is empty"
			logger.Warnf(ctx, errorMessage)
		}
	} else {
		errorMessage = fmt.Sprint("RunID is empty: ", runID)
	}

	logger.Debugf(ctx, "ReturnRunning: errorMessage:%v Run:%v", errorMessage, returnRun)

	return returnRun, errorMessage
}

func (s *Store) GetAllRunning(ctx context.Context, projectID int32, scenarioID int32, limit int32, pageNumber int32) (
	returnRuns []*pb.Run, totalPages int64, message string) {
	offset, limit := data.OffsetCalculation(limit, pageNumber)
	logger.Infof(ctx, "Query params: (limit:'%v'; offset:'%v')", limit, offset)

	mRunsList, totalPages, err := s.db.GetAllMRunningFromDB(ctx, projectID, scenarioID, limit, offset)
	if err != nil {
		message = fmt.Sprintf("Error fetch runs: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return returnRuns, totalPages, message
	}

	returnRuns, message = conv.ModelToPBRuns(ctx, mRunsList)
	logger.Infof(ctx, "len(mRunsList): '%v'", len(mRunsList))
	logger.Infof(ctx, "len(ReturnRunning): '%v'", len(returnRuns))
	logger.Infof(ctx, "Total Pages: '%v'", totalPages)

	return returnRuns, totalPages, message
}

// GetRunsByStatus возвращает список Run-ов, но без тежолых данных, таких как StaticAmmo
func (s *Store) GetRunsByStatus(ctx context.Context, status pb.Run_Status, limit int32, pageNumber int32) (
	returnRuns []*pb.Run, totalPages int64, message string) {
	offset, limit := data.OffsetCalculation(limit, pageNumber)
	logger.Infof(ctx, "Query params(runs): (limit:'%v'; offset:'%v')", limit, offset)

	mRuns, totalPages, err := s.db.GetMRunsByStatus(ctx, status, limit, offset)
	if err != nil {
		err = errors.Wrapf(err, "Error GetRunsByStatus: '%v'", status)
		logger.Error(ctx, err)

		return returnRuns, totalPages, err.Error()
	}

	if mRuns != nil {
		logger.Infof(ctx, "Get runs by status. Prepared len runs: '%+v';", len(mRuns))

		for i, mRun := range mRuns {
			logger.Debugf(ctx, "NumRun:'%v' mProjectID: '%v'; mScenarioID: '%v'; mRunID: '%v'; mRun: '%+v';", i,
				mRun.ProjectID, mRun.ScenarioID, mRun.RunID, mRun)
		}

		returnRuns, message = conv.ModelToPBRuns(ctx, mRuns)
		if message != "" {
			logger.Errorf(ctx, "ModelToPBRunsByStatus: error message:'%v' lenRuns'%v'", message, len(returnRuns))
		}
	} else {
		message = "mRuns is empty"
		logger.Warnf(ctx, message)
		return nil, 0, message
	}

	logger.Infof(ctx, "len(mRuns): '%v'", len(mRuns))
	logger.Infof(ctx, "len(ReturnRunning): '%v'", len(returnRuns))
	logger.Infof(ctx, "Total Pages(Runs): '%v'", totalPages)
	logger.Debugf(ctx, "ReturnRuns(Runs): '%v'", returnRuns)

	for _, run := range returnRuns {
		for _, scriptRun := range run.ScriptRuns {
			scriptRun.Info = ""
			switch scriptRun.TypeScriptRun {
			case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
				scriptRun.SimpleScript.StaticAmmo = ""
			}
		}
	}

	return returnRuns, totalPages, message
}

func (s *Store) GetLastRunByScenarioID(ctx context.Context, scenarioID int32) (
	returnRun *pb.Run, err error) {
	logger.Infof(ctx, "Get Scenario Last Run Status. Send data: '%+v'", scenarioID)

	mRun, err := s.db.GetLastMRunning(ctx, scenarioID)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result set") {
			err = errors.Wrapf(err, "error getting last. Scenario{%v} Last Run Status: '%v'", scenarioID, err.Error())
		}
		logger.Warnf(ctx, "error get lastRun: %+v", err.Error())

		return returnRun, err
	}
	if mRun == nil {
		err = fmt.Errorf("error getting last. MRunning == nil: scenarioID{%v}", scenarioID)
		logger.Error(ctx, err.Error())

		return returnRun, err
	}

	logger.Debugf(ctx, "getting last. Get run. Prepared model: '%+v';", mRun)
	logger.Infof(ctx, "getting last. mProjectID: '%v'; mScenarioID: '%v'; mRunID: '%v';",
		mRun.ProjectID, mRun.ScenarioID, mRun.RunID)

	returnRun, err = conv.ModelToPBRun(ctx, mRun)
	if err != nil {
		logger.Errorf(ctx, "error getting last. ModelToPBRun: '%v' '%v'", returnRun, err)
	}

	logger.Debugf(ctx, "getting last. Get pbRun.: '%+v';", returnRun)

	return returnRun, err
}
