package data

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
)

// var logger = zap.S()

func (s *Store) GetAllMAgents(ctx context.Context) (mAgents []*models.Agent, err error) {
	agents, err := models.Agents(
		qm.OrderBy(models.AgentColumns.Tags),
		qm.OrderBy("host_name ASC"),
	).All(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Get all mAgents error")
		logger.Warnf(ctx, "GetAllAgents: error:'%v'", err)
	}

	return agents, err
}

func (s *Store) GetAllEnabledMAgents(ctx context.Context) (mAgents []*models.Agent, err error) {
	mAgents, errSelect := models.Agents(
		models.AgentWhere.Enabled.EQ(true),
		qm.OrderBy(models.AgentColumns.Tags),
	).All(ctx, s.db)
	if errSelect != nil {
		err = errors.Wrap(errSelect, "Get all mAgents errorSelect")
		logger.Warnf(ctx, "GetAllMAgents: errorSelect:'%v'", errSelect)
	}

	return mAgents, err
}

// SetMAgent Сохранение нового агента в БД
func (s *Store) SetMAgent(ctx context.Context, mAgent *models.Agent) (agentID int32, errInsert error) {
	errInsert = mAgent.Insert(ctx, s.db, boil.Blacklist(
		models.AgentColumns.DeletedAt,
		models.AgentColumns.AgentID,
	))
	if errInsert != nil {
		errInsert = errors.Wrap(errInsert, "Set MAgent mAgents error insert")
		logger.Warnf(ctx, "SetMAgent: errInsert: '%v'", errInsert)

		return -1, errInsert
	}

	logger.Infof(ctx, "SetMAgent mAgent: '%+v'", mAgent)

	return mAgent.AgentID, errInsert
}

func (s *Store) GetMAgent(ctx context.Context, agentID int32) (mAgent *models.Agent, err error) {
	mAgent, err = models.Agents(
		models.AgentWhere.AgentID.EQ(agentID),
	).One(ctx, s.db)

	return mAgent, err
}

// GetAllTags Возвращает список уникальных тегов со всех активных агентов.
func (s *Store) GetAllTags(ctx context.Context) (tags []string, err error) {
	if config.Get(ctx).ENV != config.EnvInfra {
		defer undecided.WarnTimer(ctx, "GetAllTags")()
	}

	agents, err := s.GetAllEnabledMAgents(ctx)
	if err != nil {
		logger.Warnf(ctx, "GetAllTags: error:'%v'", err)
	}

	for _, agent := range agents {
		if len(agent.Tags) > 0 {
			for _, tegAgent := range agent.Tags {
				var exists bool

				for _, tag := range tags {
					if tegAgent == tag {
						exists = true
						break
					}
				}

				if exists {
					continue
				}

				tags = append(tags, tegAgent)
			}
		} else {
			logger.Warnf(ctx, "There are no active tags")
		}
	}

	logger.Infof(ctx, "AllTags raw: '%+v'", tags)
	sort.Strings(tags)
	logger.Infof(ctx, "AllTags sort: '%+v'", tags)
	return tags, err
}

func (s *Store) GetAllMAgentsByTag(ctx context.Context, tag string) (mAgents []*models.Agent, err error) {
	mAgents, err = models.Agents(
		qm.Where("agent.tags && ARRAY[?]", tag),
		qm.OrderBy(models.AgentColumns.HostName),
	).All(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Get mAgents by tags error")
		logger.Warnf(ctx, "GetAllMAgentsByTag: error:'%v'", err)
	}

	return mAgents, err
}

func (s *Store) GetAllEnabledMAgentsByTag(ctx context.Context, tag string) (mAgents []*models.Agent, err error) {
	mAgents, errSelect := models.Agents(
		qm.Where("agent.tags && ARRAY[?]", tag),
		models.AgentWhere.Enabled.EQ(true),
		qm.OrderBy(models.AgentColumns.TotalLoading),
	).All(ctx, s.db)
	if errSelect != nil {
		err = errors.Wrap(errSelect, "Get all mAgents by tag errorSelect")
		logger.Warnf(ctx, "GetAllEnabledMAgentsByTag: errorSelect:'%v'", errSelect)
	}

	return mAgents, err
}

// UpdateMAgent Обновление свойств агента в БД
func (s *Store) UpdateMAgent(ctx context.Context, mAgent *models.Agent) (returnMAgent *models.Agent, err error) {
	if mAgent != nil && !mAgent.Enabled {
		logger.Debug(ctx, "Agent is disabled", debug.Stack())
	}

	if mAgent != nil {
		_, err = mAgent.Update(ctx, s.db, boil.Blacklist(
			models.AgentColumns.CreatedAt,
			models.AgentColumns.DeletedAt,
		))
		if err != nil {
			message := fmt.Sprintf("Error update mAgent, update to db: '%v'", err.Error())
			logger.Error(ctx, "Errorr: ", message, err)
		}
	} else {
		logger.Errorf(ctx, "UpdateMAgent -> mAgent: '%v'", mAgent)
	}

	return mAgent, err
}

func (s *Store) DeleteAgent(ctx context.Context, mAgent *models.Agent) error {
	_, err := mAgent.Delete(ctx, s.db)
	return err
}

func (s *Store) GetExistEnabledMAgentsByTag(ctx context.Context, tag string) (exist bool, err error) {
	exist, err = models.Agents(
		qm.Where("agent.tags && ARRAY[?]", tag),
		models.AgentWhere.Enabled.EQ(true),
		//qm.OrderBy(models.AgentColumns.TotalLoading),
	).Exists(ctx, s.db)
	if err != nil {
		err = errors.Wrap(err, "Get exist enabled mAgents by tag errorSelect")
		logger.Warnf(ctx, "GetExistEnabledMAgentsByTag: errorSelect:'%v'", err)
	}

	return exist, err
}
