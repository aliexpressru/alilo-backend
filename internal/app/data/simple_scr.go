package data

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetMSimpleScript(ctx context.Context, simpleScriptID int32) (*models.SimpleScript, error) {
	serverScript, err := models.SimpleScripts(
		models.SimpleScriptWhere.ScriptID.EQ(simpleScriptID),
	).One(ctx, s.db)

	return serverScript, err
}

func (s *Store) GetAllEnabledMSimpleScripts(ctx context.Context, scenarioID int32) (models.SimpleScriptSlice, error) {
	return models.SimpleScripts(
		models.SimpleScriptWhere.ScenarioID.EQ(scenarioID),
		models.SimpleScriptWhere.Enabled.EQ(true),
	).All(ctx, s.db)
}

func (s *Store) GetAllMSimpleScripts(ctx context.Context, scenarioID int32) (models.SimpleScriptSlice, error) {
	return models.SimpleScripts(
		models.SimpleScriptWhere.ScenarioID.EQ(scenarioID),
		qm.OrderBy("name ASC"),
	).All(ctx, s.db)
}

func (s *Store) CreateMSimpleScript(ctx context.Context, mSimpleScript *models.SimpleScript) (int32, error) {
	//ctx = boil.WithDebug(ctx, true)
	err := mSimpleScript.Insert(ctx, s.db, boil.Blacklist(
		models.SimpleScriptColumns.DeletedAt,
		models.SimpleScriptColumns.ScriptID,
	))

	return mSimpleScript.ScriptID, err
}

func (s *Store) UpdateMSimpleScript(ctx context.Context, mSimpleScript *models.SimpleScript) error {
	//ctx = boil.WithDebug(ctx, true)
	_, err := mSimpleScript.Update(ctx, s.db, boil.Blacklist(
		models.SimpleScriptColumns.DeletedAt,
		models.SimpleScriptColumns.CreatedAt,
	))

	return err
}

func (s *Store) DeleteMSimpleScript(ctx context.Context, simpleScriptID int32) error {
	mSimpleScript, err := models.FindSimpleScript(ctx, s.db, simpleScriptID)
	if err != nil {
		err = errors.Wrapf(err, "Error find SimpleScript")
		logger.Errorf(ctx, err.Error())

		return err
	}

	logger.Debugf(ctx, "Delete SimpleScript. Prepared model: '%+v'", mSimpleScript)
	// TODO: переделать на софт-удаление с обновлением deleted_at
	if _, er := mSimpleScript.Delete(ctx, s.db); er != nil {
		err = errors.Wrapf(err, "Error delete SimpleScript")
		logger.Errorf(ctx, err.Error())
	}

	return err
}

func (s *Store) GetAllMSimpleScriptsWithExpr(ctx context.Context, scenarioID int32) (models.SimpleScriptSlice, error) {
	return models.SimpleScripts(
		qm.Where("scenario_id = ?", scenarioID),
		qm.And("(expr_rt <> '' OR expr_err <> '' OR expr_rps <> '')"),
	).All(ctx, s.db)
}
