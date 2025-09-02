package datapb

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) NewPbCmdUpdate(ctx context.Context, runID int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdUpdate(ctx, runID)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdUpdate getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdUpdate conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdStopScenario(ctx context.Context, runID int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdStopScenario(ctx, runID)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdStopScenario getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdStopScenario conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdStopScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdStopScript(ctx, runID, scriptIDs)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdStopScript getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdStopScript conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdRunScenarioWithRpsAdjustment(ctx context.Context, runID int32, percentageOfTarget int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdRunScenarioWithRpsAdjustment(ctx, runID, percentageOfTarget)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScenarioWithRpsAdjustment getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScenarioWithRpsAdjustment conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdRunScriptWithRpsAdjustment(ctx context.Context,
	runID int32, scriptIDs []int64, percentageOfTarget int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdRunScriptWithRpsAdjustment(ctx, runID, scriptIDs, percentageOfTarget)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScriptWithRpsAdjustment getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScriptWithRpsAdjustment conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdRunScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdRunScript(ctx, runID, scriptIDs)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScript getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunScript conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdAdjustment(ctx context.Context, runID int32, percentageOfTarget int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdAdjustment(ctx, runID, percentageOfTarget)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdAdjustment getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdAdjustment conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}

func (s *Store) NewPbCmdRunSimpleScriptWithRpsAdjustment(ctx context.Context,
	runID int32, scriptIDs []int64, percentageOfTarget int32) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdRunSimpleScriptWithRpsAdjustment(ctx, runID, scriptIDs, percentageOfTarget)
	if err != nil {
		err = errors.Wrap(err, "the creation of a command to run a simple script on the rps broke down")
		return nil, err
	}

	cmdAdjustmentSimpleScript, err := conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunSimpleScriptWithRpsAdjustment conversion error: '%+v'", err)
		return pbCommand, err
	}

	return cmdAdjustmentSimpleScript, err
}

func (s *Store) NewPbCmdRunSimpleScript(ctx context.Context, runID int32, scriptIDs []int64) (
	pbCommand *pb.Command, err error) {
	mCmd, err := s.db.NewMCmdRunSimpleScript(ctx, runID, scriptIDs)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunSimpleScript getting error: '%+v'", err)
		return pbCommand, err
	}

	pbCommand, err = conv.ModelToPBCmd(ctx, mCmd)
	if err != nil {
		logger.Warnf(ctx, "NewPbCmdRunSimpleScript conversion error: '%+v'", err)
		return pbCommand, err
	}

	return pbCommand, err
}
