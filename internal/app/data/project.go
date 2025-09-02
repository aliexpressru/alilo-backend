package data

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetAllMProjects(ctx context.Context, limit int32, offset int32) (
	serverProjects []*models.Project, totalPages int64, err error) {
	numberLines, err := s.GetCountProjects(ctx)
	if err != nil {
		err = errors.Wrapf(err, "Error counting projects: '%v'", err)
		logger.Error(ctx, "Error select count Projects: ", err)
	}

	var qMods []qm.QueryMod
	qMods = append(qMods,
		qm.Limit(int(limit)),
		qm.Offset(int(offset)),
		qm.OrderBy("title ASC"),
	)

	mProjectsList, err := models.Projects(qMods...).All(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Error fetch projects")
		logger.Errorf(ctx, "Error select projects: %v", err)
	}

	totalPages = CalculateTotalPages(numberLines, limit)

	return mProjectsList, totalPages, err
}

func (s *Store) GetCountProjects(ctx context.Context) (countProjects int64, err error) {
	numberLines, err := models.Projects().Count(ctx, s.db)

	return numberLines, err
}

func (s *Store) GetMProject(ctx context.Context, projectID int32) (project *models.Project, err error) {
	serverProject, err := models.Projects(
		models.ProjectWhere.ID.EQ(projectID),
	).One(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Error get mProject")
		logger.Warnf(ctx, err.Error())

		return nil, err
	}

	logger.Infof(ctx, "Get mProject. Prepared model: '%+v'", serverProject)

	return serverProject, err
}

func (s *Store) DeleteMProject(ctx context.Context, projectID int32) (status bool, err error) {
	// TODO: переделать на софт-удаление с обновлением deleted_at
	mProject, err := models.FindProject(ctx, s.db, projectID)
	if err != nil {
		err = errors.Wrap(err, "Error delete project, error get project")
		return false, err
	}

	if _, er := mProject.Delete(ctx, s.db); er != nil {
		err = errors.Wrap(err, "Error delete project, insert to db")
		return false, err
	}

	return true, err
}

func (s *Store) UpdateMProject(ctx context.Context, project *models.Project) (status bool, err error) {
	if _, err = project.Update(ctx, s.db, boil.Blacklist(
		models.ProjectColumns.CreatedAt,
		models.ProjectColumns.DeletedAt,
	)); err != nil {
		err = errors.Wrap(err, "Error insert to db project")
		logger.Errorf(ctx, "Errorr: %v", err)

		return false, err
	}

	return true, err
}

func (s *Store) CreateMProject(ctx context.Context, mProject models.Project) (models.Project, error) {
	err := mProject.Insert(ctx, s.db, boil.Blacklist(
		models.ProjectColumns.DeletedAt,
		models.ProjectColumns.ID,
	))

	return mProject, err
}
