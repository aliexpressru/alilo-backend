package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetAllMRunningFromDB(
	ctx context.Context,
	projectID int32,
	scenarioID int32,
	limit int32,
	offset int32,
) (
	mRuns models.RunSlice, totalPages int64, err error) {
	var numberLines int64

	var qms = append([]qm.QueryMod{},
		qm.OrderBy(fmt.Sprint(models.RunColumns.UpdatedAt, " desc")),
		qm.Limit(int(limit)),
		qm.Offset(int(offset)),
	)

	if projectID != 0 {
		numberLines, err = models.Runs(
			models.RunWhere.ProjectID.EQ(projectID),
		).Count(ctx, s.db)

		if numberLines > 0 {
			mRuns, err = models.Runs(
				append(qms, models.RunWhere.ProjectID.EQ(projectID))...,
			).All(ctx, s.db)
		}
	} else if scenarioID != 0 {
		numberLines, err = models.Runs(
			models.RunWhere.ScenarioID.EQ(scenarioID),
		).Count(ctx, s.db)

		if numberLines > 0 {
			mRuns, err = models.Runs(
				append(qms, models.RunWhere.ScenarioID.EQ(scenarioID))...,
			).All(ctx, s.db)
		}
	} else {
		numberLines, err = models.Runs().Count(ctx, s.db)
		if numberLines > 0 {
			mRuns, err = models.Runs(
				qms...,
			).All(ctx, s.db)
		}
	}

	totalPages = CalculateTotalPages(numberLines, limit)

	return mRuns, totalPages, err
}

// GetCountActiveMRunning Функция подсчета запущенных или запускаемых ранов по условиям:
// если задан projectID - ищем по projectID, иначе
// если задан scenarioID - ищем по scenarioID, иначе
// ищем общее кол-во запущенных ранов
func (s *Store) GetCountActiveMRunning(ctx context.Context, projectID int32, scenarioID int32) (
	countMRuns int64, err error) {
	if projectID != 0 {
		countMRunning, errRunning := models.Runs(
			models.RunWhere.ProjectID.EQ(projectID),
			models.RunWhere.Status.EQ(models.EstatusSTATUS_RUNNING),
		).Count(ctx, s.db)
		countMPrepared, errPrepared := models.Runs(
			models.RunWhere.ProjectID.EQ(projectID),
			models.RunWhere.Status.EQ(models.EstatusSTATUS_PREPARED),
		).Count(ctx, s.db)

		countMRuns = countMRunning + countMPrepared

		if errRunning != nil || errPrepared != nil {
			err = errors.Wrapf(errRunning, "%+v", errPrepared)
		}
	} else if scenarioID != 0 {
		countMRunning, errRunning := models.Runs(
			models.RunWhere.ScenarioID.EQ(scenarioID),
			models.RunWhere.Status.EQ(models.EstatusSTATUS_RUNNING),
		).Count(ctx, s.db)
		countMPrepared, errPrepared := models.Runs(
			models.RunWhere.ScenarioID.EQ(scenarioID),
			models.RunWhere.Status.EQ(models.EstatusSTATUS_PREPARED),
		).Count(ctx, s.db)

		countMRuns = countMRunning + countMPrepared

		if errRunning != nil || errPrepared != nil {
			err = errors.Wrapf(errRunning, "%+v", errPrepared)
		}
	} else {
		countMRunning, errRunning := models.Runs(
			models.RunWhere.Status.EQ(models.EstatusSTATUS_RUNNING),
		).Count(ctx, s.db)
		countMPrepared, errPrepared := models.Runs(
			models.RunWhere.Status.EQ(models.EstatusSTATUS_PREPARED),
		).Count(ctx, s.db)

		countMRuns = countMRunning + countMPrepared

		if errRunning != nil || errPrepared != nil {
			err = errors.Wrapf(errRunning, "%+v", errPrepared)
		}
	}

	return countMRuns, err
}

func (s *Store) UpdateMRunningInTheDB(ctx context.Context, mRun *models.Run) (
	returnMRun *models.Run, err error) {
	if mRun != nil {
		if mRun.Info != "" { //fixme: не делать логики в слое БД!!!!
			percent := 0.2
			length := len(mRun.Info)
			twentyPercentLength := int(float64(length) * percent)
			mRun.Info = mRun.Info[length-twentyPercentLength:]
		}
		_, err = mRun.Update(ctx, s.db, boil.Blacklist(
			models.RunColumns.CreatedAt,
			models.RunColumns.DeletedAt,
		))
		if err != nil {
			message := fmt.Sprintf("Error update run, update to db: '%v'", err.Error())
			logger.Errorf(ctx, "Errorr: %v", message, err)
		}
	} else {
		logger.Errorf(ctx, "UpdateMRunningInTheDB -> mRun: '%v'", mRun)
	}

	return mRun, err
}

func (s *Store) GetMRunning(ctx context.Context, runID int32) (returnRun *models.Run, err error) {
	mRun, err := models.Runs(

		models.RunWhere.RunID.EQ(runID),
	).One(ctx, s.db)
	if err != nil {
		err = errors.Wrapf(err, "Error fetch runs: '%v'", runID)
		logger.Errorf(ctx, "Errorr: '%v'", err)
	}

	// logger.Debugf("GetModelsRunning: '%+v'", mRun)

	return mRun, err
}

func (s *Store) GetCountRunsByStatus(ctx context.Context, status pb.Run_Status) (countRuns int64, err error) {
	numberLines, err := models.Runs(
		models.RunWhere.Status.EQ(status.String()),
	).Count(ctx, s.db)

	return numberLines, err
}

func (s *Store) GetMRunsByStatus(ctx context.Context, status pb.Run_Status, limit int32, offset int32) (
	returnRun []*models.Run, totalPages int64, err error) {
	numberLines, err := s.GetCountRunsByStatus(ctx, status)
	if err != nil {
		err = errors.Wrapf(err, "Error counting runs{%s}: '%v'", status, err)
		logger.Error(ctx, "Errorr: ", err)

		return returnRun, totalPages, err
	}

	mods := []qm.QueryMod{
		models.RunWhere.Status.EQ(status.String()), // Filter by status
		qm.Limit(int(limit)),                       // Limit the number of results
		qm.Offset(int(offset)),                     // Offset for pagination
	}
	if status == pb.Run_STATUS_STOPPING {
		mods = append(mods, qm.OrderBy(fmt.Sprintf("%s desc", models.RunColumns.UpdatedAt)))
	} else {
		mods = append(mods, qm.OrderBy(fmt.Sprintf("%s desc", models.RunColumns.CreatedAt)))
	}
	if numberLines > 0 {
		returnRun, err = models.Runs(
			mods...,
		).All(ctx, s.db)
		if err != nil {
			err = errors.Wrapf(err, "Error fetch runs: '%v'", status)
			logger.Error(ctx, "Errorr: ", err)
		}
	}

	// logger.Infof("Get ModelsRuns By Status: '%+v'", len(returnRun))

	totalPages = CalculateTotalPages(numberLines, limit)

	return returnRun, totalPages, err
}

func (s *Store) CreateNewMRun(
	ctx context.Context,
	projectID int32,
	scenarioID int32,
	percentageOfTarget int32,
	title string,
	userName string,
	preferredUserName string,
) (*models.Run, error) {
	mRun := &models.Run{
		Status:             models.EstatusSTATUS_PREPARED,
		ProjectID:          projectID,
		ScenarioID:         scenarioID,
		Title:              title,
		PercentageOfTarget: percentageOfTarget,
		UserName:           userName,
		PreferredUserName:  preferredUserName,
	}

	err := mRun.Insert(ctx, s.db, boil.Blacklist(
		models.RunColumns.DeletedAt,
		models.RunColumns.RunID,
	))
	if err != nil {
		err = errors.Wrapf(err, "Error create run, insert to db")
		logger.Error(ctx, err)

		return mRun, err
	}

	return mRun, nil
}

func (s *Store) GetLastMRunning(ctx context.Context, scenarioID int32) (returnMRun *models.Run, err error) {
	returnMRun, err = models.Runs(
		models.RunWhere.ScenarioID.EQ(scenarioID),
		qm.OrderBy(fmt.Sprint(models.RunColumns.RunID, " desc")),
	).One(ctx, s.db)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return returnMRun, nil
		}
		err = errors.Wrapf(err, "error fetch last runs")
	}

	logger.Debugf(ctx, "GetLastMRunning: '%+v'", returnMRun)

	return returnMRun, err
}

// UpdateReportURLMRunning функция сохранения ссылки в репорт.
// Не обновляет поле UpdatedAt
func (s *Store) UpdateReportURLMRunning(ctx context.Context, runID int32, reportURL string) (err error) {
	mRun, err := s.GetMRunning(ctx, runID)
	if err != nil {
		return err
	}
	mRun.ReportLink = null.StringFrom(reportURL)
	_, err = mRun.Update(ctx, s.db, boil.Blacklist(
		models.RunColumns.CreatedAt,
		models.RunColumns.DeletedAt,
		models.RunColumns.UpdatedAt,
	))
	if err != nil {
		logger.Errorf(ctx, "mRun.Update reportURL err: %v", err)
	}
	return err
}
