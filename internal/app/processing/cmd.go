package processing

import (
	"context"
	"fmt"
	"math"

	dataPb "github.com/aliexpressru/alilo-backend/internal/app/datapb"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"github.com/pkg/errors"
)

func CommandNew(ctx context.Context,
	commandType string, runID int32, scriptIDs []int64, percentageOfTarget int32, increaseRPS int32, db *dataPb.Store) (
	cmd *pb.Command, err error) {
	if commandType != pb.Type_TYPE_UPDATE.String() &&
		commandType != pb.Type_TYPE_STOP_SCENARIO.String() &&
		commandType != pb.Type_TYPE_RUN_SCRIPT.String() {
		if len(scriptIDs) < 1 {
			message := fmt.Sprintf("No ID to process %v", scriptIDs)
			logger.Errorf(ctx, "CommandNew error: %v", message)

			return nil, errors.New(message)
		}

		// пересчитываем РПСы в процент, fixme: есть проблема, процент проставляется по первому айдишнику из массива скриптов
		if increaseRPS > 0 && percentageOfTarget == 0 {
			if math.MaxInt32 < scriptIDs[0] {
				message := fmt.Sprintf("variable overflow error, scriptID %v > %v", scriptIDs[0], math.MaxInt32)
				logger.Errorf(ctx, "CommandNew error: %v ", message)

				return nil, errors.New(message)
			}
			//nolint:gosec // scriptID всегда < MaxInt32
			scriptsRuns, message := db.GetScriptRunsByScriptID(ctx, runID, int32(scriptIDs[0]))
			if message != "" && len(scriptsRuns) > 0 {
				logger.Errorf(ctx, "CommandNew GetScriptRunsByScriptID error: %v", message)
				return nil, errors.New(message)
			}

			percentageOfTarget = int32(
				undecided.WhatPercentageRoundedToWhole(float64(increaseRPS), float64(scriptsRuns[0].GetTarget())),
			)
		}
	}

	switch commandType {
	case pb.Type_TYPE_RUN_SCENARIO_UNSPECIFIED.String():
		cmd, err = db.NewPbCmdRunScenarioWithRpsAdjustment(ctx, runID, percentageOfTarget)
	case pb.Type_TYPE_RUN_SCRIPT.String():
		cmd, err = db.NewPbCmdRunScriptWithRpsAdjustment(ctx, runID, scriptIDs, percentageOfTarget)
	case pb.Type_TYPE_RUN_SIMPLE_SCRIPT.String():
		cmd, err = db.NewPbCmdRunSimpleScriptWithRpsAdjustment(ctx, runID, scriptIDs, percentageOfTarget)
	case pb.Type_TYPE_ADJUSTMENT.String():
		cmd, err = db.NewPbCmdAdjustment(ctx, runID, percentageOfTarget)
	case pb.Type_TYPE_STOP_SCENARIO.String():
		cmd, err = db.NewPbCmdStopScenario(ctx, runID)
	case pb.Type_TYPE_STOP_SCRIPT.String():
		cmd, err = db.NewPbCmdStopScript(ctx, runID, scriptIDs)
	case pb.Type_TYPE_UPDATE.String():
		cmd, err = db.NewPbCmdUpdate(ctx, runID)
	default:
		message := fmt.Sprintf("the case %v is not implemented! ", commandType)
		logger.Errorf(ctx, "CommandNew default: %v", message)

		return nil, errors.New(message)
	}

	return cmd, err
}
