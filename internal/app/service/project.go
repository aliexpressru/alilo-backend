package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) GetAllProjects(
	ctx context.Context,
	request *pb.GetAllProjectsRequest,
) (*pb.GetAllProjectsResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_projects")

	logger.Infof(ctx, "Successful request GetAllProjects: '%v'", request.String())

	projects, totalPages, message := processing.GetAllProjects(
		ctx,
		request.GetLimit(),
		request.GetPageNumber(),
		s.store,
	)
	rs := &pb.GetAllProjectsResponse{
		Status:     message == "",
		Projects:   projects,
		Message:    message,
		TotalPages: &totalPages,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) GetProject(ctx context.Context, request *pb.GetProjectRequest) (*pb.GetProjectResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_project")

	logger.Infof(ctx, "Successful request GetProject: '%v'", request.String())

	proj, message := s.store.GetProject(ctx, request.GetId())
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetProjectResponse{
		Status:  message == "",
		Project: proj,
		Message: message,
	}, nil
}

func (s *Service) CreateProject(
	ctx context.Context,
	request *pb.CreateProjectRequest,
) (*pb.CreateProjectResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "create_project")

	logger.Infof(ctx, "Successful request CreateProject: '%v'", request.String())

	status, message, id := processing.CreateProject(ctx, request.GetProject(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CreateProjectResponse{
		Status:  status,
		Id:      id,
		Message: message,
	}, nil
}

func (s *Service) UpdateProject(
	ctx context.Context,
	request *pb.UpdateProjectRequest,
) (*pb.UpdateProjectResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "update_project")

	logger.Infof(ctx, "Successful request UpdateProject: '%v'", request.String())

	status, message := processing.UpdateProject(ctx, request.GetProject(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.UpdateProjectResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) DeleteProject(
	ctx context.Context,
	request *pb.DeleteProjectRequest,
) (*pb.DeleteProjectResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_project")

	logger.Infof(ctx, "Successful request DeleteProject: '%v'", request.String())

	status, message := processing.DeleteProject(ctx, request.GetId(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.DeleteProjectResponse{
		Status:  status,
		Message: message,
	}, nil
}
