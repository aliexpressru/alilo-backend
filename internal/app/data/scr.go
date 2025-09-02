package data

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
)

func (s *Store) GetMScript(ctx context.Context, scriptID int32) (*models.Script, error) {
	serverScript, err := models.Scripts(
		models.ScriptWhere.ScriptID.EQ(scriptID),
	).One(ctx, s.db)

	return serverScript, err
}

func (s *Store) GetAllEnabledMScripts(ctx context.Context, scenarioID int32) (models.ScriptSlice, error) {
	return models.Scripts(
		models.ScriptWhere.ScenarioID.EQ(scenarioID),
		models.ScriptWhere.Enabled.EQ(true),
	).All(ctx, s.db)
}

func (s *Store) GetAllMScripts(ctx context.Context, scenarioID int32) (models.ScriptSlice, error) {
	return models.Scripts(
		models.ScriptWhere.ScenarioID.EQ(scenarioID),
		qm.OrderBy("name ASC"),
	).All(ctx, s.db)
}

func (s *Store) CreateMScript(ctx context.Context, mScript *models.Script) (int32, error) {
	err := mScript.Insert(ctx, s.db, boil.Blacklist(
		models.ScriptColumns.DeletedAt,
		models.ScriptColumns.ScriptID,
	))

	return mScript.ScriptID, err
}

func (s *Store) DeleteMScript(ctx context.Context, scriptID int32) error {
	script, err := models.FindScript(ctx, s.db, scriptID)
	if err != nil {
		return err
	}

	_, err = script.Delete(ctx, s.db)
	if err != nil {
		return err
	}

	return err
}

func (s *Store) UpdateMScript(ctx context.Context, mScript *models.Script) error {
	if mScript.AdditionalEnv == "" {
		mScript.AdditionalEnv = "{}"
	}
	_, err := mScript.Update(ctx, s.db, boil.Blacklist(
		models.ProjectColumns.CreatedAt,
		models.ProjectColumns.DeletedAt,
	))

	return err
}

func (s *Store) FindScript(ctx context.Context, scriptID int32) (*models.Script, error) {
	return models.FindScript(ctx, s.db, scriptID)
}

func (s *Store) GetAllMScriptsWithExpr(ctx context.Context, scenarioID int32) (models.ScriptSlice, error) {
	return models.Scripts(
		qm.Where("scenario_id = ?", scenarioID),
		qm.And("(expr_rt <> '' OR expr_err <> '' OR expr_rps <> '')"),
	).All(ctx, s.db)
}
