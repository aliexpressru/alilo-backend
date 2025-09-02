/*
Package conv Пакет представляет функционал преобразования структур БД к Pb, Pb к БД
*/
package conv

import (
	"context"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

// var logger = zap.S()

func ModelsToPbAgents(ctx context.Context, agents models.AgentSlice) (ags []*pb.Agent, err error) {
	ags = []*pb.Agent{}

	logger.Infof(ctx, "len(allAgents):  %v", len(agents))

	for _, agent := range agents {
		var pbAgent *pb.Agent

		pbAgent, err = ModelsToPbAgent(ctx, agent)
		if err != nil {
			logger.Warnf(ctx, "modelsToPbAgent error:%v", err)
			return ags, err
		}

		ags = append(ags, pbAgent)
	}

	logger.Infof(ctx, "len(ReturnAgents): %v", len(ags))

	return ags, err
}

func ModelsToPbAgent(ctx context.Context, mAgent *models.Agent) (*pb.Agent, error) {
	ag := &pb.Agent{}

	err := ModelToPb(ctx, mAgent, ag)
	if err != nil {
		logger.Warnf(ctx, "ModelsToPbAgents -> ModelToPb ERROR: '%v'", err)
		return nil, err
	}
	logger.Info(ctx, "ModelsToPbAgents -> ModelToPb Success")

	return ag, err
}

func PbToModelsAgent(ctx context.Context, agent *pb.Agent) (mAgent *models.Agent, err error) {
	mAgent = &models.Agent{}

	mes := PbToModel(ctx, mAgent, agent)
	if mes != "" {
		logger.Warnf(ctx, "ModelsToPbAgents -> ModelToPb ERROR: '%v'", err)
		return nil, errors.New(mes)
	}

	return mAgent, err
}
