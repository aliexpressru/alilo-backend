// Пакет работы с БД из прото сущностей
package datapb

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// var logger = zap.S()

func (s *Store) GetAgent(ctx context.Context, agentID int32) (agent *pb.Agent, message string) {
	mAgent, err := s.db.GetMAgent(ctx, agentID)
	if err != nil {
		message = fmt.Sprintf("Select agent error: '%+v'", err)
		logger.Warnf(ctx, "GetAgent select: message:'%v'; error:'%+v'", message)
	}

	agent, err = conv.ModelsToPbAgent(ctx, mAgent)
	if err != nil {
		logger.Warnf(ctx, "GetAgent -> modelsToPbAgent error: %+v", err)
	}

	if agent.GetAgentId() == 0 {
		logger.Warnf(ctx, "The required agent is missing! Agent(id-> '%v'; val-> '%+v';)", agentID, mAgent)
		message = fmt.Sprint("Agent ID '", agentID, "' not found!")
	} else {
		logger.Infof(ctx, "Found agent(id-> '%v'; agent-> '%+v';)", agentID, agent)
	}

	return agent, message
}

func (s *Store) GetAllAgents(ctx context.Context) (ags []*pb.Agent, message string) {
	agents, err := s.db.GetAllMAgents(ctx)
	if err != nil {
		message = fmt.Sprintf("Get all agents error: '%+v'", err)
		logger.Warnf(ctx, "GetAllAgents error, message:'%v';", message)
	}

	ags, err = conv.ModelsToPbAgents(ctx, agents)
	if err != nil {
		message = fmt.Sprintf("error when converting from a model to a proto structure: %+v", err)
		logger.Error(ctx, message)
	}

	return ags, message
}

func (s *Store) SetAgent(ctx context.Context, pbAgent *pb.Agent) (agentID int32, message string) {
	logger.Infof(ctx, "SetAgent pbAgent: '%+v'", pbAgent)

	mAgent, err := conv.PbToModelsAgent(ctx, pbAgent)
	if err != nil {
		logger.Warnf(ctx, "Error convert Pb to Model Agent", err.Error())

		return 0, err.Error()
	}

	agentID, err = s.db.SetMAgent(ctx, mAgent)
	if err != nil {
		return agentID, err.Error()
	}

	return agentID, message
}

// GetAllEnabledAgents метод не проверяет фактическую доступность агентов,
// если требуются проверенно доступные агенты вызывать метод processing.GetAllEnabledAndCheckedAgents
func (s *Store) GetAllEnabledAgents(ctx context.Context) (agents []*pb.Agent, err error) {
	mAgents, errSelect := s.db.GetAllEnabledMAgents(ctx)
	if errSelect != nil {
		message := fmt.Sprintf("Get all mAgents errorSelect: '%+v'", errSelect)
		logger.Warnf(ctx, "GetAllAgents: message:'%v'; errorSelect:'%v'", message, errSelect)
	}

	for _, agent := range mAgents {
		pbAgent, errConvert := conv.ModelsToPbAgent(ctx, agent)
		if errConvert != nil {
			logger.Warnf(ctx, "modelsToPbAgent errorConvert:", errConvert)
		}

		agents = append(agents, pbAgent)
	}

	return agents, errSelect
}

func (s *Store) GetAllTags(ctx context.Context) (tags []string, message error) {
	return s.db.GetAllTags(ctx)
}

func (s *Store) GetAllAgentsByTag(ctx context.Context, tag string) (ags []*pb.Agent, message string) {
	agents, err := s.db.GetAllMAgentsByTag(ctx, tag)
	if err != nil {
		message = fmt.Sprintf("Get all agents by teg error: '%+v'", err)
		logger.Warnf(ctx, "GetAllAgentsByTag: message:'%v'; error:'%v'", message)
	}

	ags, err = conv.ModelsToPbAgents(ctx, agents)
	if err != nil {
		message = fmt.Sprintf("error when converting from a model to a proto structure: %+v", err)
		logger.Error(ctx, message)
	}

	return ags, message
}

//	GetAllEnabledAgentsByTag Метод не проверяет фактическую доступность агентов,
//
// Если требуются проверенно доступные агенты вызывать метод GetAllEnabledAndCheckedAgents
func (s *Store) GetAllEnabledAgentsByTag(ctx context.Context, tag string) (agents []*pb.Agent, err error) {
	mAgents, errSelect := s.db.GetAllEnabledMAgentsByTag(ctx, tag)
	if errSelect != nil {
		message := fmt.Sprintf("Get all enabled mAgents by tag errorSelect: '%+v'", errSelect)
		logger.Warnf(ctx, "GetAllEnabledMAgentsByTag: message:'%v'; errorSelect:'%v'", message, errSelect)
	}

	for _, agent := range mAgents {
		pbAgent, errConvert := conv.ModelsToPbAgent(ctx, agent)
		if errConvert != nil {
			logger.Warnf(ctx, "modelsToPbAgent errorConvert:", errConvert)
		}

		agents = append(agents, pbAgent)
	}

	return agents, errSelect
}

func (s *Store) UpdatePbAgent(ctx context.Context, pbAgent *pb.Agent) (
	returnPbAgent *pb.Agent, err error) {
	mAgent, err := conv.PbToModelsAgent(ctx, pbAgent)
	if err != nil {
		logger.Warnf(ctx, "UpdateAgent -> PbToModelsAgent fail: '%v'", err.Error())

		return nil, err
	}

	mAgent, err = s.db.UpdateMAgent(ctx, mAgent)
	if err != nil {
		logger.Warnf(ctx, "UpdateAgent -> UpdateMAgent fail: '%v'", err.Error())

		return nil, err
	}

	pbAgent, err = conv.ModelsToPbAgent(ctx, mAgent)
	if err != nil {
		logger.Warnf(ctx, "UpdateAgent -> ModelsToPbAgent fail: '%v'", err.Error())

		return nil, err
	}

	return pbAgent, err
}
