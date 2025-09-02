package agent

import (
	"context"
	"fmt"

	v1 "github.com/aliexpressru/alilo-backend/pkg/clients/pb/qa/loadtesting/alilo/agent-v2/agent/api/qa/loadtesting/alilo/agent/v1"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/math"
)

const geometricDistributionParam = 0.3

// GetAFreeAgent Функция получения pb.Agent для запуска скрипта.
func (a *Manager) GetAFreeAgent(ctx context.Context, tag string) (agentHost *pb.Agent, err error) {
	logger.Infof(ctx, "Get A Free Agent.")

	tag, err = util.CheckingTagForPresenceInDB(ctx, tag, a.db)
	if err != nil {
		logger.Error(ctx, "GetAFreeAgent -->", err)

		return nil, err
	}

	allAgents, message := a.GetAllEnabledAndCheckedAgents(ctx, tag)
	if message != nil {
		return nil, message
	}

	var randAgent *models.Agent

	l := len(allAgents)

	if l <= 0 {
		return nil, fmt.Errorf("0 free agents (tag: %s) (all: %v)", tag, l)
	}

	var i int
	if l != 1 {
		i = math.GeometricRandomValue(ctx, geometricDistributionParam, l)
	}

	randAgent = allAgents[i]

	rAgent, err := conv.ModelsToPbAgent(ctx, randAgent)
	if err != nil {
		logger.Errorf(ctx, "ModelsToPbAgent err: %v", err)

		return nil, err
	}

	logger.Debugf(ctx, "Get A Free Agent success. db: %#v pb: %#v", rAgent, randAgent)

	return rAgent, nil
}

func (a *Manager) GetAllEnabledAndCheckedAgents(ctx context.Context, tag string) (agents []*models.Agent, err error) {
	var errSelect error

	if tag == "" {
		agents, errSelect = a.db.GetAllEnabledMAgents(ctx)
	} else {
		agents, errSelect = a.db.GetAllEnabledMAgentsByTag(ctx, tag)
	}

	if errSelect != nil {
		err = fmt.Errorf("get all mAgents errorSelect: '%w'", errSelect)
		logger.Warnf(ctx, "GetAllAgents: message:'%v'; errorSelect:'%v'", err, errSelect)

		return nil, err
	} else if len(agents) == 0 {
		err = fmt.Errorf("get all mAgents Select: There are no available agents")
		logger.Warnf(ctx, "GetAllAgents: message:'%v'; errorSelect:'%v'", err, errSelect)

		return nil, err
	}

	//return a.CheckingAgentsAvailability(ctx, agents), nil
	return agents, nil
}

// CheckingAgentsAvailability Функция проверяет доступность переданных агентов
func (a *Manager) CheckingAgentsAvailability(ctx context.Context, notCheckedAgents []*models.Agent) (
	checkedAgents []*models.Agent) {
	for _, mAgent := range notCheckedAgents {
		ac, err := a.GetClientM(ctx, mAgent)
		if err != nil {
			continue
		}

		_, err = ac.GetAllTasks(ctx, &v1.GetAllTasksRequest{})
		if err != nil {
			logger.Warnf(ctx, "Checking pbAgent, err: '%v'", err)

			mAgent.Enabled = false

			_, err = a.db.UpdateMAgent(ctx, mAgent)
			if err != nil {
				logger.Warnf(ctx, "Checking pbAgent, mAgent.Update errorUpdate: '%v'", err)
			}

			continue
		}

		checkedAgents = append(checkedAgents, mAgent)
	}

	return checkedAgents
}
