package data

import (
	"context"
	"fmt"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// SeparatorForTheView "|-|" - это сепаратор для значений(столбцов), во view_search
const SeparatorForTheView = "|-|"

// SearchM Функция поиска
func (s *Store) SearchM(ctx context.Context,
	searchQuery string, notLike bool, projectID int32, scenarioID int32, startWith bool, endWith bool) (
	mScripts models.ScriptSlice,
	mSimpleScripts models.SimpleScriptSlice,
	mScenarios models.ScenarioSlice,
	mProjects models.ProjectSlice,
	err error) {
	if len(searchQuery) > 200 {
		return mScripts, mSimpleScripts, mScenarios, mProjects, fmt.Errorf("the search word is too long")
	} else if len(searchQuery) == 0 {
		if projectID == 0 && scenarioID == 0 {
			return mScripts, mSimpleScripts, mScenarios, mProjects, fmt.Errorf("the search word is too short")
		}
	}

	queryMods := s.modsForRequest(projectID, scenarioID, notLike, searchQuery, startWith, endWith)

	//ctx = boil.WithDebug(ctx, true)
	result, err := models.ViewSearches(
		queryMods...,
	).
		All(ctx, s.db)

	logger.Infof(ctx, "Search result count: %v", len(result))

	if err != nil {
		logger.Errorf(ctx, "Search error: %v", err.Error())
		return mScripts, mSimpleScripts, mScenarios, mProjects, err
	}

	mScriptsIDs, mSimpleScriptsIDs, mScenariosIDs, mProjectsIDs := s.extractIDsFromResult(ctx, result)
	logger.Infof(ctx, "Count result:{ mScriptsIDs:%v; mSimpleScriptsIDs:%v; mProjectsIDs:%v; mScenariosIDs:%v }",
		len(mScriptsIDs), len(mSimpleScriptsIDs), len(mProjectsIDs), len(mScenariosIDs))

	if len(mScriptsIDs) > 0 {
		mScripts, err = models.Scripts(
			models.ScriptWhere.ScriptID.IN(mScriptsIDs),
		).All(ctx, s.db)
		if err != nil {
			logger.Errorf(ctx, "Get Scripts error: %v", err.Error())
		}
	}

	if len(mSimpleScriptsIDs) > 0 {
		mSimpleScripts, err = models.SimpleScripts(
			models.SimpleScriptWhere.ScriptID.IN(mSimpleScriptsIDs),
		).All(ctx, s.db)
		if err != nil {
			logger.Errorf(ctx, "Get SimpleScripts error: %v", err.Error())
		}
	}

	if len(mProjectsIDs) > 0 {
		mProjects, err = models.Projects(
			models.ProjectWhere.ID.IN(mProjectsIDs),
		).All(ctx, s.db)
		if err != nil {
			logger.Errorf(ctx, "Get Projects error: %v", err.Error())
		}
	}

	if len(mScenariosIDs) > 0 {
		mScenarios, err = models.Scenarios(
			models.ScenarioWhere.ScenarioID.IN(mScenariosIDs),
		).All(ctx, s.db)
		if err != nil {
			logger.Errorf(ctx, "Get Scenarios error: %v", err.Error())
		}
	}

	return mScripts, mSimpleScripts, mScenarios, mProjects, err
}

func (s *Store) extractIDsFromResult(ctx context.Context, result models.ViewSearchSlice) (
	mScriptsIDs []int32, mSimpleScriptsIDs []int32, mScenariosIDs []int32, mProjectsIDs []int32) {
	for _, viewEntry := range result {
		switch viewEntry.Type.String {
		case "script":
			mScriptsIDs = append(mScriptsIDs, viewEntry.ID.Int32)
		case "simple_script":
			mSimpleScriptsIDs = append(mSimpleScriptsIDs, viewEntry.ID.Int32)
		case "project":
			mProjectsIDs = append(mProjectsIDs, viewEntry.ID.Int32)
		case "scenario":
			mScenariosIDs = append(mScenariosIDs, viewEntry.ID.Int32)
		default:
			logger.Errorf(ctx, "incorrect type to search: %v", viewEntry.Type.String)
		}
	}
	return mScriptsIDs, mSimpleScriptsIDs, mScenariosIDs, mProjectsIDs
}

func (s *Store) modsForRequest(
	projectID int32, scenarioID int32,
	notLike bool,
	searchQuery string,
	startWith bool, endWith bool) []qm.QueryMod {
	if startWith {
		searchQuery = fmt.Sprintf("%%%s%s%%", SeparatorForTheView, searchQuery)
	} else if endWith {
		searchQuery = fmt.Sprintf("%%%s%s%%", searchQuery, SeparatorForTheView)
	} else {
		searchQuery = fmt.Sprintf("%%%v%%", searchQuery)
	}
	condition := models.ViewSearchWhere.Data.LIKE(null.StringFrom(searchQuery))
	if notLike {
		condition = models.ViewSearchWhere.Data.NLIKE(null.StringFrom(searchQuery))
	}

	queryMods := make([]qm.QueryMod, 0) //qm.Where(condition, searchQuery),

	if projectID != 0 {
		queryMods = append(queryMods,
			models.ViewSearchWhere.ProjectID.EQ(null.Int32From(projectID)))
	} else if scenarioID != 0 {
		queryMods = append(queryMods,
			models.ViewSearchWhere.ScenarioID.EQ(null.Int32From(scenarioID)))
	}

	queryMods = append(queryMods, condition)
	return queryMods
}

func (s *Store) PageNumber(ctx context.Context, typeEntry pb.PageNumRequest_TypesEntry, entryID int32, limit int32) (
	pageNum int32, err error) {
	if entryID < 1 {
		return 0, fmt.Errorf("incorrect entryID")
	}
	if limit < 1 {
		return 0, fmt.Errorf("incorrect limit")
	}
	var idColumnsName, titleColumnsName, tableName, title string

	switch typeEntry {
	case pb.PageNumRequest_TYPES_ENTRY_SCENARIOS:
		idColumnsName = models.ScenarioColumns.ScenarioID
		titleColumnsName = models.ScenarioColumns.Title
		tableName = "scenarios"
	case pb.PageNumRequest_TYPES_ENTRY_PROJECTS:
		idColumnsName = models.ProjectColumns.ID
		titleColumnsName = models.ProjectColumns.Title
		tableName = "projects"
	default:
		return 0, nil
	}

	query := fmt.Sprintf(`
WITH RankedEntries AS (
    SELECT %s, ROW_NUMBER() OVER (ORDER BY %s) AS row_num, %s
    FROM %s
)
SELECT CEIL(row_num / $1)+1 AS page_number, %s
FROM RankedEntries
WHERE %s = $2;
`, idColumnsName, titleColumnsName, titleColumnsName,
		tableName,
		titleColumnsName,
		idColumnsName,
	)

	//ctx = boil.WithDebug(ctx, true)//
	err = s.db.QueryRowContext(ctx, query,
		limit,
		entryID).Scan(&pageNum, &title)
	if err != nil {
		return pageNum, err
	}

	return pageNum, nil
}
