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

func (s *Service) GetAllScripts(
	ctx context.Context,
	request *pb.GetAllScriptsRequest,
) (*pb.GetAllScriptsResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_all_scripts")

	logger.Infof(ctx, "Successful request GetAllScripts: '%v'", request.String())

	scripts, sMessage := processing.GetAllScripts(ctx, request.GetScenarioId(), s.store)
	simpleScripts, sSMessage := processing.GetAllSimpleScript(ctx, request.GetScenarioId(), s.store)
	status := processing.CheckStatusGetAllScripts(ctx, sMessage, sSMessage)
	message := processing.MessageProcessing(ctx, sMessage, sSMessage)
	rs := &pb.GetAllScriptsResponse{
		Status:        status,
		Message:       message,
		Scripts:       scripts,
		SimpleScripts: simpleScripts,
	}
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return rs, nil
}

func (s *Service) GetScript(ctx context.Context, request *pb.GetScriptRequest) (*pb.GetScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_script")

	logger.Infof(ctx, "Successful request GetScript: '%v'", request.String())

	scr, message := processing.GetScript(ctx, request.GetScriptId(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetScriptResponse{
		Status:  message == "",
		Script:  scr,
		Message: message,
	}, nil
}

func (s *Service) CreateScript(ctx context.Context, request *pb.CreateScriptRequest) (*pb.CreateScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "create_script")

	logger.Infof(ctx, "Successful request CreateScript: '%v'", request.String())
	status, message, id := processing.CreateScript(ctx, request.GetScript(), s.store)
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CreateScriptResponse{
		Status:   status,
		ScriptId: id,
		Message:  message,
	}, nil
}

func (s *Service) UpdateScript(ctx context.Context, request *pb.UpdateScriptRequest) (*pb.UpdateScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "update_script")

	logger.Infof(ctx, "Successful request UpdateScript: '%v'", request.String())
	status, message := processing.UpdateScript(ctx, request.GetScript(), s.data)
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.UpdateScriptResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) DeleteScript(ctx context.Context, request *pb.DeleteScriptRequest) (*pb.DeleteScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_script")
	logger.Infof(ctx, "Successful request DeleteScript: '%v'", request.String())
	status, message := processing.DeleteScript(ctx, request.GetScriptId(), s.data)

	err := util.SetModalHeader(ctx)
	if err != nil {
		message = fmt.Sprintf("Failed to set header %v", err)
		status = false
	}
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.DeleteScriptResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) MoveScript(ctx context.Context, request *pb.MoveScriptRequest) (*pb.MoveScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "move_script")

	logger.Infof(ctx, "Successful request MoveScript: '%v'", request.String())
	status, message := processing.MoveScript(
		ctx,
		request.GetScriptId(),
		request.GetScenarioId(),
		request.GetProjectId(),
		s.data,
	)
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.MoveScriptResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) CopyScript(ctx context.Context, request *pb.CopyScriptRequest) (*pb.CopyScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "copy_script")

	logger.Infof(ctx, "Successful request CopyScript: '%v'", request.String())
	message := processing.CopyScript(ctx, request.GetSourceScriptId(), request.GetTargetScenarioId(), s.data)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CopyScriptResponse{
		Status:  message == "",
		Message: message,
	}, nil
}

func (s *Service) GetSimpleScript(
	ctx context.Context,
	request *pb.GetSimpleScriptRequest,
) (*pb.GetSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "get_simple_script")

	logger.Infof(ctx, "Successful request GetSimpleScript: '%v'", request.String())

	simpleScript, message := processing.GetSimpleScript(ctx, request.GetSimpleScriptId(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.GetSimpleScriptResponse{
		Status:       message == "",
		SimpleScript: simpleScript,
		Message:      message,
	}, nil
}

func (s *Service) CreateSimpleScript(
	ctx context.Context,
	request *pb.CreateSimpleScriptRequest,
) (*pb.CreateSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "create_simple_script")

	logger.Infof(ctx, "Successful request CreateSimpleScript: '%v'", request.String())
	message, simpleScriptID := processing.CreateSimpleScript(ctx, request.GetSimpleScript(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CreateSimpleScriptResponse{
		Status:         message == "",
		SimpleScriptId: simpleScriptID,
		Message:        message,
	}, nil
}

func (s *Service) UpdateSimpleScript(
	ctx context.Context,
	request *pb.UpdateSimpleScriptRequest,
) (*pb.UpdateSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "update_simple_script")

	logger.Infof(ctx, "Successful request UpdateSimpleScript: '%v'", request.String())
	script := request.GetSimpleScript()
	err := processing.UpdateSimpleScript(ctx, script, s.store)
	message := fmt.Sprintf("The script %s was updated!", script.Name)

	if err != nil {
		message = err.Error()
		_ = util.SetModalHeader(ctx)
	}

	return &pb.UpdateSimpleScriptResponse{
		Status:  err == nil,
		Message: message,
	}, nil
}

func (s *Service) DeleteSimpleScript(
	ctx context.Context,
	request *pb.DeleteSimpleScriptRequest,
) (*pb.DeleteSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "delete_simple_script")

	logger.Infof(ctx, "Successful request DeleteSimpleScript: '%v'", request.String())
	err := processing.DeleteSimpleScript(ctx, request.GetSimpleScriptId(), s.store)
	if err != nil {
		logger.Warnf(ctx, "Failed to delete simple script with id %d %v", request.SimpleScriptId, err.Error())
		_ = util.SetModalHeader(ctx)

		return &pb.DeleteSimpleScriptResponse{
			Status:  false,
			Message: "Failed to delete script",
		}, err
	}

	message := "Script was deleted!"
	return &pb.DeleteSimpleScriptResponse{
		Status:  err == nil,
		Message: message,
	}, err
}

func (s *Service) MoveSimpleScript(
	ctx context.Context,
	request *pb.MoveSimpleScriptRequest,
) (*pb.MoveSimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "move_simple_script")

	logger.Infof(ctx, "Successful request MoveSimpleScript: '%v'", request.String())
	status, message := processing.MoveSimpleScript(ctx,
		request.GetSimpleScriptId(), request.GetTargetScenarioId(), s.store)
	if !status {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.MoveSimpleScriptResponse{
		Status:  status,
		Message: message,
	}, nil
}

func (s *Service) CopySimpleScript(
	ctx context.Context,
	request *pb.CopySimpleScriptRequest,
) (*pb.CopySimpleScriptResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "copy_simple_script")

	logger.Infof(ctx, "Successful request CopySimpleScript: '%v'", request.String())
	message, SimpleScriptID := processing.CopySimpleScript(ctx,
		request.GetSourceSimpleScriptId(), request.GetTargetScenarioId(), s.store)
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.CopySimpleScriptResponse{
		Status:         message == "",
		Message:        message,
		SimpleScriptId: SimpleScriptID,
	}, nil
}
