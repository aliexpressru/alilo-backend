package datapb

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/conv"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func (s *Store) GetPBSimpleScript(
	ctx context.Context,
	simpleScriptID int32,
) (simpleScript *pb.SimpleScript, err error) {
	mSimpleScript, err := s.db.GetMSimpleScript(ctx, simpleScriptID)
	if err != nil {
		logger.Errorf(ctx, "Error get '%v' mSimpleScript: '%v'", simpleScriptID, err.Error())
		return simpleScript, err
	}

	simpleScript, mes := conv.ModelToPBSimpleScript(ctx, mSimpleScript)
	if mes != "" {
		mes = fmt.Sprintf("Error simpleScriptModel To PB: '%v'", mes)
		err = errors.New(mes)
		logger.Errorf(ctx, err.Error())

		return simpleScript, err
	}

	return simpleScript, err
}

func (s *Store) GetAllEnabledPBSimpleScripts(ctx context.Context, scenarioID int32) (
	sliceSimpleScript []*pb.SimpleScript, err error) {
	simpleScriptSlice, err := s.db.GetAllEnabledMSimpleScripts(ctx, scenarioID)
	if err != nil {
		logger.Errorf(ctx, "Error get mSimpleScript: '%v'", err.Error())
		return sliceSimpleScript, err
	}

	logger.Infof(
		ctx,
		"Get all Active simpleScript. Send data, scenarioID: '%+v'; simpleScriptSlice len: '%+v'",
		scenarioID,
		len(simpleScriptSlice),
	)

	for _, script := range simpleScriptSlice {
		simpleScript, message := conv.ModelToPBSimpleScript(ctx, script)
		if message != "" {
			mess := fmt.Sprintf("Error conversion Model To PB SimpleScript: '%v'", message)
			logger.Warnf(ctx, mess)
			err = errors.New(mess)
		}

		sliceSimpleScript = append(sliceSimpleScript, simpleScript)
	}

	return sliceSimpleScript, err
}

func (s *Store) GetAllPBSimpleScripts(
	ctx context.Context,
	scenarioID int32,
) (simpleScripts []*pb.SimpleScript, err error) {
	mSimpleScripts, err := s.db.GetAllMSimpleScripts(ctx, scenarioID)
	for _, script := range mSimpleScripts {
		simpleScript, message := conv.ModelToPBSimpleScript(ctx, script)
		if message != "" {
			mess := fmt.Sprintf("Error conversion Model To PB SimpleScript: '%v'", message)
			logger.Warnf(ctx, mess)
			err = errors.New(mess)
		}

		simpleScripts = append(simpleScripts, simpleScript)
	}

	return simpleScripts, err
}

func (s *Store) CreatePBSimpleScript(ctx context.Context, simpleScript *pb.SimpleScript) (scriptID int32, err error) {
	mSimpleScript, mess := conv.PbToSimpleScriptModel(ctx, simpleScript)
	if mess != "" {
		err = errors.New(mess)
		logger.Warnf(ctx, "Error UpdatePbSimpleScript '%v'", mess)

		return -1, err
	}

	simpleScriptID, err := s.db.CreateMSimpleScript(ctx, mSimpleScript)
	if err != nil {
		return simpleScriptID, err
	}

	return simpleScriptID, err
}

func (s *Store) UpdatePbSimpleScript(ctx context.Context, simpleScript *pb.SimpleScript) error {
	mSimpleScript, mess := conv.PbToSimpleScriptModel(ctx, simpleScript)
	if mess != "" {
		err := errors.New(mess)
		logger.Warnf(ctx, "Error converted to PbSimpleScript '%v'", mess)

		return err
	}
	prevScript, err := s.db.GetMSimpleScript(ctx, simpleScript.ScriptId)
	if err != nil {
		logger.Warnf(ctx, "Failed to get simple script with id %d %v", simpleScript.ScriptId, err)
		return err
	}
	if mSimpleScript.Enabled == prevScript.Enabled {
		err = util.SetModalHeader(ctx)
		if err != nil {
			logger.Warnf(ctx, "Failed to set header %v", err)
			return err
		}
	}
	err = s.db.UpdateMSimpleScript(ctx, mSimpleScript)
	if err != nil {
		err = errors.Wrap(err, "Fail UpdatePbSimpleScript in DB")
		logger.Warn(ctx, err.Error())

		return err
	}

	return err
}
