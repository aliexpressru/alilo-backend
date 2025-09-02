package job

import (
	"context"
	"time"

	"github.com/aliexpressru/alilo-backend/internal/app/config"
	"github.com/aliexpressru/alilo-backend/internal/app/data"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

const cMDProviderContextKey = "_command_provider"

var (
	duration time.Duration
)

func CommandProvider(ctx context.Context, db *data.Store) {
	ctx = undecided.NewContextWithMarker(ctx, cMDProviderContextKey, "")

	defer func(ctx context.Context) {
		if err := recover(); err != nil {
			logger.Errorf(ctx, "! ! ! ! CommandProcessor failed: '%+v'", err)
		}
	}(ctx)

	cfg := config.Get(ctx)
	duration = cfg.JobCmdProcessorFrequency
	logger.Infof(ctx, "CommandProvider Started: '%v'", duration)

	hostname := cfg.Hostname
	logger.Infof(ctx, "CommandProvider Hostname: '%v'", hostname)

	for {
		logger.Info(ctx, "Start Processor")

		cmd, ifReceived := db.GetMCommandToProcess(ctx, hostname)
		if !ifReceived {
			time.Sleep(duration)

			continue
		}

		logger.Infof(ctx, "Command received(Provider) '%v:%v'{%+v}", cmd.CommandID, cmd.Type, cmd)
		cfg.CommandChan <- cmd

		// Если команда UPDATE, и других UPDATE нет, немного поспать
		if cmd.Type == models.CmdtypeTYPE_UPDATE {
			countNewCmd, er := db.CountNewCmd(ctx, hostname)
			if er != nil {
				logger.Errorf(ctx, "Error get Count cmdUpdate: %v", er.Error())
			} else if countNewCmd == 0 {
				time.Sleep(duration)
			}
		}

		logger.Infof(ctx, "CommandProcessor go to start")
	}
}
