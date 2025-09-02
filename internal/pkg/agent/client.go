package agent

import (
	"context"
	"sync"
	"time"

	agentclient "github.com/aliexpressru/alilo-backend/pkg/clients/pb/qa/loadtesting/alilo/agent-v2/agent/api/qa/loadtesting/alilo/agent/v1"

	"github.com/aliexpressru/alilo-backend/internal/app/data"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	v1 "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Manager struct {
	mu sync.Mutex
	am map[string]agentclient.AgentServiceClient

	db *data.Store
}

func NewAgentManager(ctx context.Context, db *data.Store, agentHosts ...string) (*Manager, error) {
	a := &Manager{
		mu: sync.Mutex{},
		am: make(map[string]agentclient.AgentServiceClient),

		db: db,
	}

	for _, agent := range agentHosts {
		ac, err := a.newClient(ctx, agent)
		if err != nil {
			return nil, err
		}

		a.am[agent] = ac
	}

	return a, nil
}

func (a *Manager) GetClientPB(ctx context.Context, agent *v1.Agent) (agentclient.AgentServiceClient, error) {
	return a.getClient(ctx, undecided.GetPBHost(agent))
}

func (a *Manager) GetClientM(ctx context.Context, agent *models.Agent) (agentclient.AgentServiceClient, error) {
	return a.getClient(ctx, undecided.GetMHost(agent))
}

func (a *Manager) getClient(ctx context.Context, agentHost string) (agentclient.AgentServiceClient, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	logger.Debugf(ctx, "GetAgentClient(%s)", agentHost)

	if c, ok := a.am[agentHost]; ok {
		return c, nil
	}

	newAgentClient, err := a.newClient(ctx, agentHost)
	if err != nil {
		logger.Errorf(ctx, "Error getting newClient agent{%v}: %+v", agentHost, err)
		return nil, err
	}

	a.am[agentHost] = newAgentClient

	return newAgentClient, nil
}

func (a *Manager) newClient(ctx context.Context, agentHost string) (agentclient.AgentServiceClient, error) {
	ctx, canc := context.WithTimeout(ctx, 3*time.Second)
	defer canc()

	cc, err := grpc.NewClient(
		agentHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpczap.UnaryClientInterceptor(zap.L()),
		),
	)
	if err != nil {
		logger.Error(ctx, "agent conn{%v} err: %+v", err, agentHost)

		return nil, err
	}

	return agentclient.NewAgentServiceClient(cc), nil
}
