package conv

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aarondl/null/v8"
	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func ModelToPBScript(ctx context.Context, mScript *models.Script) (scr *pb.Script, err error) {
	scr = &pb.Script{}

	if mScript != nil {
		err = ModelToPb(ctx, *mScript, scr)
		if err != nil {
			err = errors.Wrapf(err, "Script model to pb")
			logger.Errorf(ctx, err.Error())

			return nil, err
		}

		var opt pb.Options

		opt.Duration = mScript.OptionsDuration.String
		opt.Rps = strconv.Itoa(mScript.OptionsRPS.Int)
		opt.Steps = strconv.Itoa(mScript.OptionsSteps.Int)

		scr.Options = &opt

		var selectors pb.Selectors

		selectors.ExprRps = mScript.ExprRPS
		selectors.SourceRps = mScript.SourceRPS
		selectors.CmtRps = mScript.CMTRPS

		selectors.ExprRt = mScript.ExprRT
		selectors.SourceRt = mScript.SourceRT
		selectors.CmtRt = mScript.CMTRT

		selectors.ExprErr = mScript.ExprErr
		selectors.SourceErr = mScript.SourceErr
		selectors.CmtErr = mScript.CMTErr

		scr.Selectors = &selectors
		scr.Title = mScript.Title

		scr.AdditionalEnv, err = modelToPbHeaders(ctx, mScript.AdditionalEnv)
		if err != nil {
			logger.Errorf(ctx, "modelToPb AdditionalEnv error: %+v -> %+v", err, mScript.AdditionalEnv)
			scr.AdditionalEnv = map[string]string{}
		}
	} else {
		err = errors.New("mScript is nil")
	}

	return scr, err
}

func PbToModelScript(ctx context.Context, protoScript *pb.Script) (mScript *models.Script, message string) {
	mScript = &models.Script{}

	message = PbToModel(ctx, mScript, protoScript)
	if message != "" {
		message = fmt.Sprintf("Script pb to model: '%v'", message)
		logger.Errorf(ctx, message)

		return nil, message
	}

	mScript.ProjectID = protoScript.ProjectId
	mScript.ScenarioID = protoScript.ScenarioId
	mScript.Enabled = protoScript.Enabled

	if protoScript.Options != nil {
		mScript.OptionsDuration = null.StringFrom(protoScript.Options.Duration)

		rps, err := strconv.Atoi(protoScript.GetOptions().GetRps())
		if err != nil {
			message = fmt.Sprintf("Cannot convert RPS proto model to int: '%v'", err.Error())
			logger.Errorf(ctx, message, err)

			return nil, message
		}

		mScript.OptionsRPS = null.IntFrom(rps)

		steps, err := strconv.Atoi(protoScript.GetOptions().GetSteps())
		if err != nil {
			message = fmt.Sprintf("Cannot convert STEPS proto model to int: '%v'", err.Error())
			logger.Errorf(ctx, message, err)

			return nil, message
		}

		mScript.OptionsSteps = null.IntFrom(steps)
	} else {
		message = "Error fetch options"
		logger.Errorf(ctx, message)

		return nil, message
	}

	if protoScript.Selectors != nil {
		mScript.ExprRPS = protoScript.Selectors.ExprRps
		mScript.SourceRPS = protoScript.Selectors.SourceRps
		mScript.CMTRPS = protoScript.Selectors.CmtRps

		mScript.ExprRT = protoScript.Selectors.ExprRt
		mScript.SourceRT = protoScript.Selectors.SourceRt
		mScript.CMTRT = protoScript.Selectors.CmtRt

		mScript.ExprErr = protoScript.Selectors.ExprErr
		mScript.SourceErr = protoScript.Selectors.SourceErr
		mScript.CMTErr = protoScript.Selectors.CmtErr
	} else {
		logger.Errorf(ctx, "Error fetch selectors(extendedScript): %v", protoScript.Name)

		return mScript, message
	}

	tmpAdditionalEnvBytes, err := json.Marshal(protoScript.AdditionalEnv)
	if err != nil {
		logger.Errorf(ctx, "mScript failed to marshal in AdditionalEnv: %v", err)
		return mScript, err.Error()
	}

	mScript.AdditionalEnv = string(tmpAdditionalEnvBytes)

	return mScript, message
}
