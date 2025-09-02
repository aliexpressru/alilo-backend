// package service представляет собой реализацию API
package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

// var logger = zap.S()

func (s *Service) GetAllAgents(ctx context.Context, request *pb.GetAllAgentsRequest) (*pb.GetAllAgentsResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_agents")

	logger.Infof(ctx, "Successful request GetAllAgents: '%v'", request.String())

	agents, message := s.store.GetAllAgents(ctx)
	rs := &pb.GetAllAgentsResponse{
		Status:  message == "",
		Agents:  agents,
		Message: message,
	}

	return rs, nil
}

func (s *Service) GetAgent(ctx context.Context, request *pb.GetAgentRequest) (*pb.GetAgentResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_agent")

	logger.Infof(ctx, "Successful request GetAgent: '%v'", request.String())

	agentHost, message := s.store.GetAgent(ctx, request.GetAgentId())

	return &pb.GetAgentResponse{
		Status:  message == "",
		Agent:   agentHost,
		Message: message,
	}, nil
}

func (s *Service) SetAgent(ctx context.Context, request *pb.SetAgentRequest) (*pb.SetAgentResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "set_agent")

	logger.Infof(ctx, "Successful request CreateAgent: '%v'", request.String())
	if len(request.GetAgent().Tags) == 0 {
		request.GetAgent().Tags = []string{""}
	}
	agentID, message := s.store.SetAgent(ctx, request.GetAgent())

	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.SetAgentResponse{
		Status:  message == "",
		AgentId: agentID,
		Message: message,
	}, nil
}

func (s *Service) UpdateAgent(ctx context.Context, request *pb.UpdateAgentRequest) (*pb.UpdateAgentResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "update_agent")

	logger.Infof(ctx, "Successful request UpdateAgent: '%v'", request.String())
	status, message := processing.UpdateAgent(ctx, request.GetAgent(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.UpdateAgentResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) DeleteAgent(ctx context.Context, request *pb.DeleteAgentRequest) (*pb.DeleteAgentResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_agent")

	logger.Infof(ctx, "Successful request DeleteAgent: '%v'", request.String())
	status, message := processing.DeleteAgent(ctx, request.GetAgentId(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.DeleteAgentResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) GetAllTags(ctx context.Context, request *pb.GetAllTagsRequest) (*pb.GetAllTagsResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_tags")

	logger.Infof(ctx, "Successful request GetAllTags: '%v'", request.String())
	var rs *pb.GetAllTagsResponse
	tags, err := s.data.GetAllTags(ctx)
	if err != nil {
		rs = &pb.GetAllTagsResponse{
			Status:  false,
			Message: err.Error(),
			Tags:    tags,
		}

		_ = util.SetModalHeader(ctx)
		return rs, err
	}
	rs = &pb.GetAllTagsResponse{
		Status:  true,
		Message: "",
		Tags:    tags,
	}

	return rs, nil
}

func (s *Service) GetAllAgentsByTag(ctx context.Context, request *pb.GetAllAgentsByTagRequest) (
	*pb.GetAllAgentsByTagResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_agents_by_tag")

	logger.Infof(ctx, "Successful request GetAllAgentsByTag: '%v'", request.String())

	agents, message := s.store.GetAllAgentsByTag(ctx, request.GetTag())
	rs := &pb.GetAllAgentsByTagResponse{
		Status:  message == "",
		Message: message,
		Agents:  agents,
	}

	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) RemoveLogs(ctx context.Context, request *pb.RemoveLogsRequest) (*pb.RemoveLogsResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "remove_logs")
	logger.Infof(ctx, "Successful request RemoveLogs: '%v'", request.String())

	message := processing.RemoveLogs(ctx, request.MoreThanDays, request.MoreThanMb, s.data, s.agentManager)

	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.RemoveLogsResponse{
		Status:  message == "",
		Message: message,
	}, nil
}
