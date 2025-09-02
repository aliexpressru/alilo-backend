package service

import (
	"context"
	"errors"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/util"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) GetAllRunning(ctx context.Context, request *pb.GetAllRunningRequest) (
	*pb.GetAllRunningResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_running")

	logger.Infof(ctx, "Successful request run-get-all: '%v'", request.String())
	runs, totalPages, message := s.store.GetAllRunning(ctx,
		request.GetProjectId(), request.GetScenarioId(), request.GetLimit(), request.GetPageNumber())

	rs := &pb.GetAllRunningResponse{
		Status:     message == "",
		Message:    message,
		Runs:       runs,
		TotalPages: &totalPages,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}
	return rs, nil
}

func (s *Service) GetRunning(ctx context.Context, request *pb.GetRunningRequest) (*pb.GetRunningResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_running")

	logger.Infof(ctx, "Successful request GetRunning: '%v'", request.String())
	scriptRun, message := s.store.GetRunning(ctx, request.GetRunId())

	rs := &pb.GetRunningResponse{
		Status:  message == "",
		Message: message,
		Run:     scriptRun,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}
	return rs, nil
}

func (s *Service) StopScript(ctx context.Context, req *pb.StopScriptRequest) (*pb.StopScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "stop_script")

	logger.Infof(ctx, "Successful request StopScript: '%v'", req.String())
	scriptRun, message := processing.ScriptToStop(ctx, req.GetRunId(), req.GetRunScriptId(), s.store)
	// проверяем наличие запущенных скриптов в родительском ране
	if message != "" {
		_ = util.SetModalHeader(ctx)

		return nil, errors.New(message) // TODO
	}

	var pbRunning *pb.Run

	pbRunning, message = s.store.GetRunning(ctx, scriptRun.GetRunId())
	if message != "" {
		_ = util.SetModalHeader(ctx)

		return nil, errors.New(message) // TODO
	}

	// fixme: это косяк, нужно передумать где хранить эту функцию
	message = s.commandProcessor.SetTheRunScenarioStatusStopped(ctx, pbRunning)
	if message != "" {
		_ = util.SetModalHeader(ctx)
		logger.Warnf(ctx, "ScenarioGroup stopped message: '%v'", message)
	}

	rs := &pb.StopScriptResponse{
		Status:  message == "",
		Message: message,
		Run:     scriptRun,
	}

	return rs, nil
}

func (s *Service) RunScenario(ctx context.Context, req *pb.RunScenarioRequest) (*pb.RunScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "run_scenario")

	logger.Infof(ctx, "Successful request RunScenario: '%v'", req.String())
	running, message := processing.ScenarioToRunning(ctx,
		req.GetScenarioId(),
		req.GetPercentageOfTarget(),
		req.UserName,
		req.PreferredUserName,
		s.store,
	)

	rs := &pb.RunScenarioResponse{
		Status:  message == "",
		Message: message,
		Run:     running,
	}

	err := util.SetModalHeader(ctx)
	if err != nil {
		_ = util.SetModalHeader(ctx)
		logger.Warnf(ctx, "SetModalHeader error: %+v", err)
		return rs, nil
	}

	return rs, nil
}

func (s *Service) StopScenario(ctx context.Context, request *pb.StopScenarioRequest) (*pb.StopScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "stop_scenario")

	logger.Infof(ctx, "Successful request StopScenario: '%v'", request)
	pBRun, message := processing.ScenarioToStop(ctx, request.GetRunId(), s.data, s.store)
	rs := &pb.StopScenarioResponse{
		Status:  message == "",
		Message: message,
		Run:     pBRun,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) GetRunsByStatus(
	ctx context.Context,
	request *pb.GetRunsByStatusRequest,
) (*pb.GetRunsByStatusResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_runs_by_status")

	logger.Infof(ctx, "Successful request GetRunsByStatus: '%v'", request.String())
	runs, totalPages, message := s.store.GetRunsByStatus(ctx,
		request.GetRunStatus(), request.GetLimit(), request.GetPageNumber())

	// fixme status должен проставляться не хардкодом true
	rs := &pb.GetRunsByStatusResponse{
		Status:     true,
		Message:    message,
		Runs:       runs,
		TotalPages: &totalPages,
	}

	return rs, nil
}

func (s *Service) AdjustmentScenario(
	ctx context.Context,
	request *pb.AdjustmentScenarioRequest,
) (*pb.AdjustmentScenarioResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "adjustment_scenario")

	logger.Infof(ctx, "Successful request AdjustmentScenario: '%v'", request.String())
	message := processing.ScenarioToAdjustment(ctx, request.GetRunId(), request.AdjustmentOnPercent, s.data, s.store)

	rs := &pb.AdjustmentScenarioResponse{
		Status:  message == "",
		Message: message,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) IncreaseScriptByRPS(
	ctx context.Context,
	request *pb.IncreaseScriptByRPSRequest,
) (*pb.IncreaseScriptByRPSResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "increase_script_prs")

	logger.Infof(ctx, "Successful request IncreaseScriptByRPS: '%v'", request.String())
	message := processing.ScriptToIncreaseByRPS(ctx,
		request.GetRunId(), request.GetScriptId(), request.GetIncreaseByRps(), s.data, s.store)

	rs := &pb.IncreaseScriptByRPSResponse{
		Status:  message == "",
		Message: message,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) RunScript(ctx context.Context, request *pb.RunScriptRequest) (*pb.RunScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "smoke_test_run_script")

	logger.Infof(ctx, "Successful request Smoke RunScript: '%v'", request.String())
	run, message := processing.ScriptToTestRunning(ctx, request.GetScriptId(), s.data, s.store)
	rs := &pb.RunScriptResponse{
		Status:  message == "",
		Message: message,
		Run:     run,
	}

	err := util.SetModalHeader(ctx)
	if err != nil {
		logger.Warnf(ctx, "SetModalHeader error: %+v", err)
		return rs, nil
	}

	return rs, nil
}

func (s *Service) RunSimpleScript(ctx context.Context, request *pb.RunSimpleScriptRequest) (
	*pb.RunSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "smoke_test_run_simple_script")

	logger.Infof(ctx, "Successful request Smoke RunSimpleScript: '%v'", request.String())
	run, message := processing.SimpleScriptToTestRunning(ctx, request.GetSimpleScriptId(), s.data, s.store)
	rs := &pb.RunSimpleScriptResponse{
		Status:  message == "",
		Message: message,
		Run:     run,
	}

	err := util.SetModalHeader(ctx)
	if err != nil {
		logger.Warnf(ctx, "SetModalHeader error: %+v", err)
		return rs, nil
	}

	return rs, nil
}

func (s *Service) IncreaseSimpleScriptByRPS(
	ctx context.Context,
	request *pb.IncreaseSimpleScriptByRPSRequest,
) (*pb.IncreaseSimpleScriptByRPSResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "increase_simple_script_prs")

	logger.Infof(ctx, "Successful request IncreaseSimpleScriptByRPS: '%v'", request.String())
	message := processing.SimpleScriptToIncreaseByRPS(ctx,
		request.GetRunId(), request.GetSimpleScriptId(), request.GetIncreaseByRps(), s.data, s.store)

	rs := &pb.IncreaseSimpleScriptByRPSResponse{
		Status:  message == "",
		Message: message,
	}
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) TimeRange(ctx context.Context, request *pb.TimeRangeRequest) (*pb.TimeRangeResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "time_range")

	logger.Infof(ctx, "Successful request TimeRange: '%v'", request.String())
	timeRange, err := processing.TimeRange(ctx, request.GetRunId(), s.store)
	logger.Infof(ctx, "TimeRange: %+v", timeRange)

	rs := &pb.TimeRangeResponse{
		Status: err != nil,
		FromTs: timeRange.From.AsTime().Unix(),
		ToTs:   timeRange.To.AsTime().Unix(),
		FromF:  timeRange.From.AsTime().Format(time.DateTime),
		ToF:    timeRange.To.AsTime().Format(time.DateTime),
	}
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}

	return rs, err
}

func (s *Service) CurrentRPS(ctx context.Context, request *pb.CurrentRPSRequest) (*pb.CurrentRPSResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "current_rps")

	logger.Infof(ctx, "Successful request CurrentRPS: '%v'", request.String())
	rps, err := processing.CurrentRPS(ctx, request.GetRunId(), s.store)
	logger.Infof(ctx, "CurrentRPS: %+v", rps)
	rs := &pb.CurrentRPSResponse{
		Status: err == nil,
		Rps:    rps,
	}
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}

	return rs, err
}

func (s *Service) ScriptRunsByScriptName(
	ctx context.Context,
	request *pb.ScriptRunsByScriptNameRequest,
) (*pb.ScriptRunsByScriptNameResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_script_runs")

	logger.Infof(ctx, "Successful request ScriptRunsByScriptName: '%v'", request.String())
	scriptRuns, err := processing.ScriptRunsByScriptName(ctx, request.GetRunId(), request.GetScriptName(), s.store)
	logger.Infof(ctx, "ScriptRunsByScrip len: %v", len(scriptRuns))
	rs := &pb.ScriptRunsByScriptNameResponse{
		ScriptRuns: scriptRuns,
	}
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}

	return rs, err
}

func (s *Service) ScriptRunsByScriptRunID(
	ctx context.Context,
	request *pb.ScriptRunsByScriptRunIDRequest,
) (*pb.ScriptRunsByScriptRunIDResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_script_run_by_id")

	logger.Infof(ctx, "Successful request ScriptRunsByScriptRunID: '%v'", request.String())
	scriptRuns, err := processing.ScriptRunsByScriptRunID(ctx, request.GetRunId(), request.GetScriptRunId(), s.store)
	logger.Infof(ctx, "ScriptRunsByID: %v", scriptRuns)
	rs := &pb.ScriptRunsByScriptRunIDResponse{
		ScriptRuns: scriptRuns,
	}
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}

	return rs, err
}

func (s *Service) IsChangeLoadLevel(
	ctx context.Context,
	request *pb.IsChangeLoadLevelRequest,
) (*pb.IsChangeLoadLevelResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "is_change_load_level")

	isChangeLoadLevel, err := processing.IsChangeLoadLevel(ctx, request.RunId, s.store)
	if err != nil {
		_ = util.SetModalHeader(ctx)
	}
	return isChangeLoadLevel, err
}
