package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) CommandNew(ctx context.Context, request *pb.CommandNewRequest) (*pb.CommandNewResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "command_new")

	logger.Infof(ctx, "Successful request CommandNew: '%v'", request.String())

	message := ""
	cmd, err := processing.CommandNew(
		ctx,
		request.GetCommandType().
			String(), // TODO разделить данные в проекте на 3 модели (Доменную, БД, Прото), использовать только доменую модель во всех процессах
		request.GetRunId(),              // TODO разделить данные в проекте на 3 модели (Доменную, БД, Прото), использовать только доменую модель во всех процессах
		request.GetScriptIds(),          // TODO разделить данные в проекте на 3 модели (Доменную, БД, Прото), использовать только доменую модель во всех процессах
		request.GetPercentageOfTarget(), // TODO разделить данные в проекте на 3 модели (Доменную, БД, Прото), использовать только доменую модель во всех процессах
		request.GetIncreaseRps(),        // TODO разделить данные в проекте на 3 модели (Доменную, БД, Прото), использовать только доменую модель во всех процессах
		s.store,
	)

	if err != nil {
		message = err.Error()
	}

	rs := &pb.CommandNewResponse{
		Status:  err == nil,
		Message: message,
		Command: cmd,
	}

	return rs, nil
}
