package datapb

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	v1 "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func (s *Store) GetAllProjects(ctx context.Context, limit int32, pageNumber int32) (
	serverProjects []*v1.Project, pages int64, message string) {
	offset, limit := data.OffsetCalculation(limit, pageNumber)
	logger.Infof(ctx, "Query params(projects): (limit:'%v'; offset:'%v')", limit, offset)
	mProjectsList, totalPages, err := s.db.GetAllMProjects(ctx, limit, offset)
	if err != nil {
		message = fmt.Sprintf("Error getting all mProjects: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return serverProjects, totalPages, err.Error()
	}

	serverProjects = make([]*v1.Project, 0, len(mProjectsList))

	for _, serverProject := range mProjectsList {
		pbProject, mes := conv.ModelToPBProject(ctx, serverProject)
		message = fmt.Sprint(message, mes)

		if mes == "" {
			serverProjects = append(serverProjects, pbProject)
		} else {
			logger.Errorf(ctx, "projectModelToPB: '%v' '%v'", pbProject, mes)
		}
	}

	logger.Infof(ctx, "len(mProjects): '%v'", len(mProjectsList))
	logger.Infof(ctx, "len(ReturnProjects): '%v'", len(serverProjects))
	logger.Infof(ctx, "Total Pages(Projects): '%v'", totalPages)
	logger.Debugf(ctx, "ReturnProjects: '%v'", serverProjects)

	return serverProjects, totalPages, message
}

func (s *Store) GetProject(ctx context.Context, projectID int32) (project *v1.Project, message string) {
	serverProject, err := s.db.GetMProject(ctx, projectID)
	if err != nil {
		message = fmt.Sprintf("Error get project'%v': '%v'", projectID, err.Error())
		logger.Warnf(ctx, message)

		return nil, message
	}

	pbProject, mes := conv.ModelToPBProject(ctx, serverProject)
	message = fmt.Sprint(message, mes)

	logger.Infof(ctx, "Get project. Prepared model: '%+v'", pbProject)

	return pbProject, message
}
