package processing

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func GetAllProjects(ctx context.Context, limit int32, number int32, store *datapb.Store) (
	serverProjects []*pb.Project, pages int64, message string) {
	countProjects, _ := store.GetDataStore().GetCountProjects(ctx)
	if limit < 1 {
		//nolint:gosec
		limit = int32(countProjects)
	}
	return store.GetAllProjects(ctx, limit, number)
}

func CreateProject(
	ctx context.Context,
	project *pb.Project,
	dataStore *data.Store,
) (status bool, message string, projectID int32) {
	var mProject models.Project

	message = conv.PbToModel(ctx, &mProject, project)
	if message != "" {
		message = fmt.Sprintf("Error create project,  convert pbProject to model: '%v'", message)
		logger.Errorf(ctx, message)

		return false, message, 0
	}

	mProject, err := dataStore.CreateMProject(ctx, mProject)
	if err != nil {
		message = fmt.Sprintf("Error create project, insert to db: '%+v'", err.Error())
		logger.Errorf(ctx, message)

		return false, message, 0
	}

	return true, message, mProject.ID
}

func UpdateProject(ctx context.Context, project *pb.Project, dataStore *data.Store) (status bool, message string) {
	logger.Infof(ctx, "UpdateProject: '%v' '%v' ", project.Id, project)

	mProject, message := conv.PBProjectToModel(ctx, project)
	if message != "" {
		message = fmt.Sprintf("Error update project: '%v'", message)
		logger.Errorf(ctx, message)

		return status, message
	}

	status, err := dataStore.UpdateMProject(ctx, mProject)
	if err != nil {
		message = fmt.Sprintf("Error update project: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return status, message
	}

	return status, message
}

func DeleteProject(ctx context.Context, projectID int32, dataStore *data.Store) (status bool, message string) {
	status, err := dataStore.DeleteMProject(ctx, projectID)
	if err != nil {
		message = fmt.Sprintf("Error delete project, error get project: '%v'", err.Error())
		return status, message
	}

	return status, message
}
