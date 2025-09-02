package data

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetMScenario(ctx context.Context, scenarioID int32) (scenario *models.Scenario, err error) {
	logger.Infof(ctx, "Get mScenario. Come data: '%+v'", scenarioID)

	serverScenario, err := models.FindScenario(ctx, s.db, scenarioID)
	if err != nil {
		err = errors.Wrapf(err, "Error get mScenario '%v'", scenarioID)
		logger.Warn(ctx, err)

		return nil, err
	}

	return serverScenario, err
}

func (s *Store) GetAllMScenarios(ctx context.Context, projectID int32) (scens []*models.Scenario, err error) {
	logger.Infof(ctx, "Get all scenarios. Send data: '%+v'", projectID)

	mScenariosList, err := models.Scenarios(
		models.ScenarioWhere.ProjectID.EQ(projectID),
	).All(ctx, s.db)
	logger.Debugf(ctx, "Get all scenarios. Prepared model: '%+v'", mScenariosList)

	if err != nil {
		err = errors.Wrap(err, "Error fetch scenarios")
		logger.Errorf(ctx, err.Error())

		return scens, err
	}

	return mScenariosList, err
}

func (s *Store) GetMScenariosPaging(ctx context.Context, projectID int32, limit int32, offset int32) (
	scens []*models.Scenario, totalPages int64, err error) {
	logger.Infof(ctx, "Get all scenarios. Send data: '%+v'", projectID)
	numberLines, err := s.GetCountScenarios(ctx, projectID)
	if err != nil {
		err = errors.Wrapf(err, "Error counting projects: '%v'", err)
		logger.Error(ctx, "Errorr: ", err)

		return scens, totalPages, err
	}

	mScenariosList, err := models.Scenarios(
		models.ScenarioWhere.ProjectID.EQ(projectID),
		qm.Limit(int(limit)),
		qm.Offset(int(offset)),
		qm.OrderBy("title ASC"),
	).All(ctx, s.db)
	logger.Debugf(ctx, "Get all scenarios. Prepared model: '%+v'", mScenariosList)

	if err != nil {
		err = errors.Wrap(err, "Error fetch scenarios")
		logger.Errorf(ctx, err.Error())

		return scens, totalPages, err
	}
	totalPages = CalculateTotalPages(numberLines, limit)

	return mScenariosList, totalPages, err
}

func (s *Store) GetCountScenarios(ctx context.Context, projectID int32) (int64, error) {
	numberLines, err := models.Scenarios(
		models.ScenarioWhere.ProjectID.EQ(projectID),
	).Count(ctx, s.db)

	return numberLines, err
}

func (s *Store) GetScenariosByIds(ctx context.Context, scenarioIDs []int32) (models.ScenarioSlice, error) {
	return models.Scenarios(
		models.ScenarioWhere.ScenarioID.IN(scenarioIDs),
	).All(ctx, s.db)
}

func (s *Store) CreateMScenario(ctx context.Context, mScenario *models.Scenario) (*models.Scenario, error) {
	err := mScenario.Insert(ctx, s.db, boil.Blacklist(
		models.ScenarioColumns.DeletedAt,
		models.ScenarioColumns.ScenarioID,
	))

	return mScenario, err
}

func (s *Store) DeleteMScenario(ctx context.Context, mScenario *models.Scenario) error {
	_, err := mScenario.Delete(ctx, s.db)
	return err
}

func (s *Store) UpdateMScenario(ctx context.Context, mScenario *models.Scenario) error {
	if _, err := mScenario.Update(ctx, s.db, boil.Blacklist(
		models.ProjectColumns.CreatedAt,
		models.ProjectColumns.DeletedAt,
	)); err != nil {
		err = errors.Wrap(err, "Error insert to db mScenario")
		logger.Errorf(ctx, "UpdateMScenario Er: %v", err)

		return err
	}

	return nil
}
