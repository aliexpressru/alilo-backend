// Package processing пакет содержит основную логику обработки сущностей
package processing

import (
	"context"
	"fmt"
	"sync"

	v1 "github.com/aliexpressru/alilo-backend/pkg/clients/pb/qa/loadtesting/alilo/agent-v2/agent/api/qa/loadtesting/alilo/agent/v1"

	"github.com/aliexpressru/alilo-backend/internal/app/data"
	dataPb "github.com/aliexpressru/alilo-backend/internal/app/datapb"
	"github.com/aliexpressru/alilo-backend/internal/pkg/agent"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// var logger = zap.S()

func UpdateAgent(ctx context.Context, agent *pb.Agent, db *dataPb.Store) (status bool, message string) {
	logger.Infof(ctx, "Update agent(AgentId:'%v'; Agent:'%+v')", agent.AgentId, agent)
	if len(agent.Tags) == 0 {
		agent.Tags = []string{""}
	}
	agent, err := db.UpdatePbAgent(ctx, agent)
	if err != nil {
		message = err.Error()
		logger.Warnf(ctx, "pbAgent UpdateAgent error: '%v'", message)

		return false, message
	}

	logger.Infof(ctx, "UpdatePbAgent(pbAgentId:'%v'; pbAgent:'%+v')", agent.GetAgentId(), agent)

	return true, message
}

// TODO MOVE TO data PACKAGE!!!!
func DeleteAgent(ctx context.Context, agentID int32, db *data.Store) (status bool, message string) {
	logger.Infof(ctx, "DeleteAgent: ", agentID)

	mAgent, err := db.GetMAgent(ctx, agentID)
	if err != nil {
		message = fmt.Sprintf("Select agent error: '%+v'", err)
		logger.Warnf(ctx, "DeleteAgent select: message:'%v'; error:'%+v'", message, err)
	}

	logger.Infof(ctx, "DeleteAgent(mAgentId:'%v'; mAgent:'%+v')", mAgent.AgentID, mAgent)

	err = db.DeleteAgent(ctx, mAgent)
	if err != nil {
		message = err.Error()
		logger.Warnf(ctx, "mAgent.Delete error: '%v'", message)

		return false, message
	}

	return true, message
}

func RemoveLogs(
	ctx context.Context,
	moreThanDays int64,
	moreThanMb int64,
	store *data.Store,
	am *agent.Manager,
) string {
	agents, err := store.GetAllEnabledMAgents(ctx)
	if err != nil {
		logger.Errorf(ctx, "Error getting all agents: %+v", err)

		return err.Error()
	}

	var mess = struct {
		m string
		sync.RWMutex
	}{}
	var wg = &sync.WaitGroup{}
	for _, a := range agents {
		exePool.Go(func() {
			wg.Add(1)
			defer wg.Done()
			cliAgent, errR := am.GetClientM(ctx, a)
			if errR != nil {
				logger.Errorf(ctx, "Error getting agent client{%v}: %v", a.HostName, errR)
				mess.Lock()
				defer mess.Unlock()
				mess.m = fmt.Sprintf(mess.m, errR.Error())

				return
			}

			request := &v1.RemoveLogsRequest{MoreThanMb: moreThanMb, MoreThanDays: moreThanDays}
			rs, errR := cliAgent.RemoveLogs(ctx, request)
			if errR != nil {
				logger.Errorf(ctx, "Error exec remove rq{%v}: %v", a.HostName, errR)
				mess.Lock()
				defer mess.Unlock()
				mess.m = fmt.Sprintf(mess.m, errR.Error())

				return
			}

			logger.Infof(ctx, "RemoveLogs %v: %v", a.HostName, rs)
		})
	}

	logger.Infof(ctx, "Wait RemoveLogs")
	wg.Wait()
	logger.Infof(ctx, "Wait id done. Message: '%v'", mess.m)

	return mess.m
}
