package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/aliexpressru/alilo-backend/pkg/clients/pb/qa/loadtesting/alilo/agent-v2/agent/api/qa/loadtesting/alilo/agent/v1"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

const (
	agentTrackingContextKey = "_agent_tracker"
	weightSpu               = 0.35
	weightMem               = 0.6
	weightPorts             = 0.05
)

var d time.Duration

func AgentsTracker(ctx context.Context, p *ProcessorPool) {
	ctx = undecided.NewContextWithMarker(ctx, agentTrackingContextKey, "")
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "agentsTracker failed: '%+v'", err)
		}
	}()

	cfg := config.Get(ctx)
	d = cfg.JobAgentTrackingFrequency
	for range time.Tick(d) {
		logger.Info(ctx, "Start AgentsTracker")
		p.agentsTracking(ctx)
	}
}

func (p *ProcessorPool) agentsTracking(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "agentsTracking failed: '%+v'", err)
		}
	}()

	defer undecided.InfoTimer(ctx, "AgentsTracking")()

	trackingWaitGroup := &sync.WaitGroup{}

	agents, err := p.db.GetAllEnabledMAgents(ctx)
	if err != nil {
		logger.Errorf(ctx, "AgentTracking error Getting all agents: '%+v'", err)
	}
	logger.Infof(ctx, "agents count: %v", len(agents))
	for i, agent := range agents {
		trackingWaitGroup.Add(1)
		logger.Infof(ctx, "Trekking (%v) %+v", i, agent)

		execPool.Go(func() {
			p.updateStatusMAgent(ctx, agent, trackingWaitGroup)
		})
	}

	trackingWaitGroup.Wait()
}

func (p *ProcessorPool) updateStatusMAgent(ctx context.Context, agent *models.Agent, group *sync.WaitGroup) {
	defer func(group *sync.WaitGroup) {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "agentsTracking failed: '%+v'", err)
		}
		group.Done()
	}(group)

	if !agent.Enabled {
		return
	}

	defer undecided.InfoTimer(ctx, fmt.Sprintf("UpdateStatusAgent: '%v'", agent.HostName))()

	// Initial status check
	agent = p.getStatusMAgent(ctx, agent)

	// Retry configuration
	maxRetries := 3
	baseDelay := time.Second * 2

	for i := 0; i <= maxRetries; i++ {
		getMAgent, err := p.db.GetMAgent(ctx, agent.AgentID)
		if err != nil {
			logger.Errorf(
				ctx,
				"GetMAgent to get the current status (attempt %d/%d). error: '%+v'",
				i+1,
				maxRetries,
				err,
			)
			if i == maxRetries {
				return
			}
			time.Sleep(baseDelay * time.Duration(i+1))
			continue
		}

		if getMAgent == nil {
			logger.Errorf(ctx, "GetMAgent to get the current status (attempt %d/%d). Agent is nil", i+1, maxRetries)
			if i == maxRetries {
				return
			}
			time.Sleep(baseDelay * time.Duration(i+1))
			continue
		}

		// If agent is still enabled, no need for retry
		if getMAgent.Enabled {
			agent.Enabled = getMAgent.Enabled
			_, err = p.db.UpdateMAgent(ctx, agent)
			if err != nil {
				logger.Errorf(ctx, "updateStatusMAgent update MAgent error: '%+v'", err)
			}
			return
		}

		// If we get here, the agent is disabled
		if i < maxRetries {
			logger.Infof(ctx, "Agent %s is disabled (was enabled), retrying in %v (attempt %d/%d)",
				agent.HostName, baseDelay*time.Duration(i+1), i+1, maxRetries)
			time.Sleep(baseDelay * time.Duration(i+1))
			continue
		}

		// Final attempt and still disabled
		logger.Infof(ctx, "Agent %s is now disabled after %d retries", agent.HostName, maxRetries)
		agent.Enabled = false
		_, err = p.db.UpdateMAgent(ctx, agent)
		if err != nil {
			logger.Errorf(ctx, "updateStatusMAgent update MAgent error: '%+v'", err)
		}
		return
	}
}

func (p *ProcessorPool) getStatusMAgent(ctx context.Context, agent *models.Agent) *models.Agent {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "getStatusMAgent failed: '%+v'", err)
		}
	}()

	defer undecided.InfoTimer(ctx, fmt.Sprintf("GetStatusAgent: '%v'", agent.HostName))()

	c, err := p.agentManager.GetClientM(ctx, agent)
	if err != nil {
		logger.Errorf(ctx, "get Client agent error: '%+v'", err)

		return agent
	}

	rsGetMetrics, err := c.Metrics(ctx, &v1.MetricsRequest{})
	if err != nil {
		logger.Errorf(ctx, "get metrics agent{%v} execute request error: '%+v'", agent.HostName, err)

		return agent
	}

	utilization := rsGetMetrics.GetAgentUtilization()
	if utilization == nil {
		logger.Errorf(ctx, "get status agent execute request fail. Utilization: '%+v'", utilization)

		return agent
	}

	logger.Infof(ctx, "get status agent response: '%+v'", utilization)
	agent.CPUUsed = int32(utilization.GetCpu())
	agent.MemUsed = int32(utilization.GetMem())
	//nolint:gosec
	agent.PortsUsed = int32(utilization.GetPercentAvailablePorts())
	agent.TotalLoading = totalLoading(agent)
	logger.Infof(ctx, "Total agent{%v} utilization: '%+v'", agent.HostName, agent.TotalLoading)

	return agent
}

func totalLoading(agent *models.Agent) (value int16) {
	result := (float64(agent.CPUUsed) * weightSpu) + (float64(agent.MemUsed) * weightMem) + (float64(agent.PortsUsed) * weightPorts)

	return int16(result)
}
