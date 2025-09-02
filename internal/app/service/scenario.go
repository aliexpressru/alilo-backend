package service

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) GetAllScenarios(
	ctx context.Context,
	request *pb.GetAllScenariosRequest,
) (*pb.GetAllScenariosResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_scenarios")

	logger.Infof(ctx, "Successful request GetAllScenarios: '%v'", request.String())

	scenarios, totalPages, message := processing.GetAllScenarios(
		ctx,
		request.GetProjectId(),
		request.GetLimit(),
		request.GetPageNumber(),
		s.store,
	)
	rs := &pb.GetAllScenariosResponse{
		Status:     message == "",
		Scenarios:  scenarios,
		Message:    message,
		TotalPages: &totalPages,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) GetScenario(ctx context.Context, request *pb.GetScenarioRequest) (*pb.GetScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_scenario")

	logger.Infof(ctx, "Successful request GetScenario: '%v'", request.String())

	// todo: project_id - нужно ли это поле в request
	pbScenario, message := s.store.GetScenario(ctx, request.GetScenarioId())
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetScenarioResponse{
		Status:   message == "",
		Scenario: pbScenario,
		Message:  message,
	}, nil
}

func (s *Service) CreateScenario(
	ctx context.Context,
	request *pb.CreateScenarioRequest,
) (*pb.CreateScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "create_scenario")

	logger.Infof(ctx, "Successful request CreateScenario: '%v'", request.String())
	status, message, id := processing.CreateScenario(ctx, request.GetScenario(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CreateScenarioResponse{
		Status:     status,
		ScenarioId: id,
		Message:    message,
	}, nil
}

func (s *Service) UpdateScenario(
	ctx context.Context,
	request *pb.UpdateScenarioRequest,
) (*pb.UpdateScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "update_scenario")

	logger.Infof(ctx, "Successful request UpdateScenario: '%v'", request.String())
	status, message := processing.UpdateScenario(ctx, request.GetScenario(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.UpdateScenarioResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) DeleteScenario(
	ctx context.Context,
	request *pb.DeleteScenarioRequest,
) (*pb.DeleteScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_scenario")

	logger.Infof(ctx, "Successful request DeleteScenario: '%v'", request.String())
	status, message := processing.DeleteScenario(ctx, request.GetScenarioId(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.DeleteScenarioResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) SetDurationScenario(ctx context.Context, request *pb.SetDurationScenarioRequest) (
	*pb.SetDurationScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "set_duration_scenario")

	logger.Infof(ctx, "Successful request SetDurationScenario: '%v'", request.String())
	status, message := processing.SetDuration(ctx,
		request.GetProjectId(), request.GetScenarioId(), request.GetDuration(), s.data, s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.SetDurationScenarioResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) GetScenarioLastRunStatus(
	ctx context.Context,
	request *pb.GetScenarioLastRunStatusRequest,
) (*pb.GetScenarioLastRunStatusResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_scenario_last_run_status")

	logger.Infof(ctx, "Successful request GetScenarioLastRunStatus: '%v'", request.String())
	lastRunStatus, lastRunID, message := s.store.GetScenarioLastRunStatus(ctx, request.GetScenarioId())

	return &pb.GetScenarioLastRunStatusResponse{
		Status:        message == "",
		Message:       message,
		LastRunStatus: lastRunStatus,
		LastRunId:     lastRunID,
	}, nil
}

func (s *Service) CopyScenario(ctx context.Context, request *pb.CopyScenarioRequest) (*pb.CopyScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "copy_scenario")

	logger.Infof(ctx, "Successful request CopyScenario: '%v'", request.String())
	_ = util.SetModalHeader(ctx)
	targetScenarioID := request.GetTargetScenarioId()
	logger.Infof(ctx, "Copy scenario: %v -> %v", request.GetSourceScenarioId(), targetScenarioID)
	if targetScenarioID == 0 {
		if request.GetTargetScenarioData().ProjectId == 0 {
			logger.Errorf(ctx, "ProjectId is empty")

			return &pb.CopyScenarioResponse{
				Status:     false,
				ScenarioId: -1,
				Message:    "ProjectId is empty",
			}, nil
		} else if request.GetTargetScenarioData().Title == "" {
			logger.Errorf(ctx, "Title is empty")

			return &pb.CopyScenarioResponse{
				Status:     false,
				ScenarioId: -1,
				Message:    "Title is empty",
			}, nil
		}

		logger.Infof(ctx, "Target scenarioID == 0. Do create new scenario")
		targetScenario := request.GetTargetScenarioData()
		targetScenario.ScenarioId = 0
		targetScenario.Title = fmt.Sprint("Copied_", targetScenario.Title)

		status, message, scenarioID := processing.CreateScenario(ctx, targetScenario, s.data)
		logger.Infof(ctx, "New ScenarioGroup for copying scripts{%v}: %+v", status, targetScenario)
		if !status {
			return &pb.CopyScenarioResponse{
				Status:     status,
				Message:    message,
				ScenarioId: scenarioID,
			}, nil
		} else {
			logger.Infof(ctx, "Create new scenario success")
		}

		targetScenarioID = scenarioID
	}

	logger.Infof(ctx, "Do copy scenario: %v -> %v", request.GetSourceScenarioId(), targetScenarioID)

	message := processing.CopyScenario(ctx, request.GetSourceScenarioId(), targetScenarioID, s.data)

	return &pb.CopyScenarioResponse{
		Status:  message == "",
		Message: message,
	}, nil
}

func (s *Service) SetStepsScenario(
	ctx context.Context,
	request *pb.SetStepsScenarioRequest,
) (*pb.SetStepsScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "set_steps_scenario")

	logger.Infof(ctx, "Successful request SetStepsScenario: '%v'", request.String())
	status, message := processing.SetSteps(ctx,
		request.GetProjectId(), request.GetScenarioId(), request.GetSteps(), s.data, s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.SetStepsScenarioResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) AmountRPS(ctx context.Context, request *pb.AmountRPSRequest) (*pb.AmountRPSResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "amount_rps")

	logger.Infof(ctx, "Successful request AmountRPS: '%v'", request.String())
	rps, err := processing.AmountRPS(ctx, request.GetScenarioId(), s.data)
	logger.Infof(ctx, "AmountRPS: %+v", rps)
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}
	rs := &pb.AmountRPSResponse{
		Status: err == nil,
		Rps:    rps,
	}

	return rs, err
}

func (s *Service) SetTegScenario(
	ctx context.Context,
	request *pb.SetTegScenarioRequest,
) (*pb.SetTegScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "set_teg_scenario")

	logger.Infof(ctx, "Successful request SetStepsScenario: '%v'", request.String())
	status, err := processing.SetTeg(ctx, request.GetScenarioId(), request.GetTag(), s.data)
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.SetTegScenarioResponse{
		Status: status,
	}, err
}

func (s *Service) GetGrafanaStructure(
	ctx context.Context,
	request *pb.GetGrafanaStructureRequest,
) (*pb.GetGrafanaStructureResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_grafana_structure")

	logger.Infof(ctx, "Successful request to GetGrafanaStructure: '%v'", request.String())

	message, err := processing.GenerateGrafanaStructure(ctx, request.GetScenarioId(), s.data)
	if err != nil {
		message = err.Error()
	}

	return &pb.GetGrafanaStructureResponse{
		Status:  err == nil,
		Message: message,
	}, nil
}

func (s *Service) GetGrafanaStructureByIds(
	ctx context.Context,
	request *pb.GetGrafanaStructureByIdsRequest,
) (*pb.GetGrafanaStructureResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_grafana_structure_by_ids")

	logger.Infof(ctx, "Successful request to GetGrafanaStructureByIds: '%v'", request.String())

	message, err := processing.GenerateGrafanaStructureByIds(ctx, request.GetScenarioIds(), request.GetTitle(), s.data)
	if err != nil {
		message = err.Error()
	}

	return &pb.GetGrafanaStructureResponse{
		Status:  err == nil,
		Message: message,
	}, nil
}
