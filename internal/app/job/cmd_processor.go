// Jobs пакет содержит обработчики разных задач запускаемые параллельно серверу
package job

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/internal/pkg/agent"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

const cMDProcessorContextKey = "_command_processor"

////var logger = zap.S().With(zap.Any("_job", ""))

type ProcessorPool struct {
	db   *data.Store
	dbPB *datapb.Store

	agentManager *agent.Manager
}

func NewProcessorPool(dataStore *data.Store, pbStore *datapb.Store, aManager *agent.Manager) *ProcessorPool {
	return &ProcessorPool{
		db:           dataStore,
		dbPB:         pbStore,
		agentManager: aManager,
	}
}

func (p *ProcessorPool) StartProcessor(ctx context.Context) {
	ctx = undecided.NewContextWithMarker(ctx, cMDProcessorContextKey, "")
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "CommandProcessor failed: '%+v'", err)
		}
	}()

	cfg := config.Get(ctx)
	hostname := cfg.Hostname
	logger.Infof(ctx, "Processor started. Hostname: '%v'", hostname)

	for cmd := range cfg.CommandChan {
		logger.Warn(ctx, "Start Processor")

		logger.Infof(ctx, "Command received '%v', '%+v'", cmd.CommandID, cmd)

		var switchResult bool

		childCmdCtx := undecided.NewContextWithMarker(ctx, cMDProcessorContextKey, cmd.Type)
		logger.Warnf(ctx, cmd.Type)

		switch cmd.Type {
		case models.CmdtypeTYPE_RUN_SCENARIO_UNSPECIFIED:
			switchResult = p.startingRunning(childCmdCtx, cmd)

		case models.CmdtypeTYPE_STOP_SCENARIO:
			switchResult = p.stoppingRunning(childCmdCtx, cmd)

		case models.CmdtypeTYPE_STOP_SCRIPT:
			switchResult = p.stoppingScript(childCmdCtx, cmd)

		case models.CmdtypeTYPE_ADJUSTMENT:
			switchResult = p.adjustment(childCmdCtx, cmd)

		case models.CmdtypeTYPE_RUN_SCRIPT:
			switchResult = p.startingScript(childCmdCtx, cmd)

		case models.CmdtypeTYPE_RUN_SIMPLE_SCRIPT:
			switchResult = p.startingSimpleScript(childCmdCtx, cmd)

		case models.CmdtypeTYPE_UPDATE:
			switchResult = p.updatingRun(childCmdCtx, cmd)

		default:
			message := fmt.Sprintf("_-_-_-_-_- The case %v is not implemented! -_-_-_-_-_", cmd.Type)
			p.db.UpdateStatusMCommand(ctx, cmd, models.CmdstatusSTATUS_FAILED, message)
			continue
		}
		if !switchResult {
			logger.Warnf(ctx, "Cmd ")
		}
		logger.Infof(
			ctx,
			"%v:'%v', CommandID:'%v', CommandStatus:'%v'",
			cmd.Type,
			switchResult,
			cmd.CommandID,
			cmd.Status,
		)
		logger.Infof(ctx, "CommandProcessor go to start")
	}
}
