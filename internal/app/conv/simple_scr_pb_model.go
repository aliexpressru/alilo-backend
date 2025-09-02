package conv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"

	//nolint
	"github.com/golang/protobuf/jsonpb"
)

func ModelToPBSimpleScript(ctx context.Context, mSimpleScript *models.SimpleScript) (
	protoSimpleScript *pb.SimpleScript, message string) {
	protoSimpleScript = &pb.SimpleScript{}
	if mSimpleScript != nil {
		err := ModelToPb(ctx, *mSimpleScript, protoSimpleScript)
		if err != nil {
			message = fmt.Sprintf("SimpleScript model to pb: '%v'", err)
			logger.Errorf(ctx, message)

			return nil, message
		}

		protoSimpleScript.QueryParams, err = modelToPbQueryParams(ctx, mSimpleScript.QueryParams)
		if err != nil {
			return protoSimpleScript, err.Error()
		}

		protoSimpleScript.Headers, err = modelToPbHeaders(ctx, mSimpleScript.Headers)
		if err != nil {
			return protoSimpleScript, err.Error()
		}

		var selectors pb.Selectors
		selectors.ExprRps = mSimpleScript.ExprRPS
		selectors.SourceRps = mSimpleScript.SourceRPS
		selectors.CmtRps = mSimpleScript.CMTRPS

		selectors.ExprRt = mSimpleScript.ExprRT
		selectors.SourceRt = mSimpleScript.SourceRT
		selectors.CmtRt = mSimpleScript.CMTRT

		selectors.ExprErr = mSimpleScript.ExprErr
		selectors.SourceErr = mSimpleScript.SourceErr
		selectors.CmtErr = mSimpleScript.CMTErr

		protoSimpleScript.Selectors = &selectors
		protoSimpleScript.Title = mSimpleScript.Title

		protoSimpleScript.AdditionalEnv, err = modelToPbHeaders(ctx, mSimpleScript.AdditionalEnv)
		if err != nil {
			logger.Errorf(ctx, "modelToPb AdditionalEnv error: %+v -> %+v", err, mSimpleScript.AdditionalEnv)
			protoSimpleScript.AdditionalEnv = map[string]string{}
		}
	} else {
		message = "mSimpleScript is nil"
	}

	return protoSimpleScript, message
}

func modelToPbQueryParams(ctx context.Context, mQueryParams string) (pbQueryParams []*pb.QueryParams, err error) {
	if mQueryParams != "" {
		if json.Valid([]byte(mQueryParams)) {
			jsonDecoder := json.NewDecoder(strings.NewReader(mQueryParams))

			_, err = jsonDecoder.Token()
			if err != nil {
				logger.Errorf(ctx, "mQueryParam Err jsonDecoder: %v", err)
				return nil, err
			}

			for jsonDecoder.More() {
				protoMessage := &pb.QueryParams{}

				err = jsonpb.UnmarshalNext(jsonDecoder, protoMessage)
				if err != nil {
					logger.Errorf(ctx, "mQueryParam UnmarshalNext error: %v", err)
					return nil, err
				}

				pbQueryParams = append(pbQueryParams, protoMessage)
			}
		} else {
			logger.Errorf(ctx, "mQueryParam Not Valid json: '%v'", mQueryParams)
			return nil, errors.New("invalidArgument mQueryParams")
		}
	} else {
		logger.Warnf(ctx, "mScriptRuns is empty: '%v'", mQueryParams)
		return pbQueryParams, nil
	}

	return pbQueryParams, nil
}

func modelToPbHeaders(ctx context.Context, mHeaders string) (pbHeaders map[string]string, err error) {
	if mHeaders != "" {
		if json.Valid([]byte(mHeaders)) {
			pbHeaders = make(map[string]string)

			err = json.Unmarshal([]byte(mHeaders), &pbHeaders)
			if err != nil {
				logger.Errorf(ctx, "Not Valid json: '%v'", mHeaders)
			}
		} else {
			logger.Errorf(ctx, "Not Valid json: '%v'", mHeaders)
			return nil, errors.New("invalidArgument mHeaders")
		}
	} else {
		logger.Warnf(ctx, "mScriptRuns is empty: '%v'", mHeaders)
		return nil, errors.New("invalidArgument mHeaders")
	}

	return pbHeaders, nil
}

func PbToSimpleScriptModel(ctx context.Context,
	protoSimpleScript *pb.SimpleScript) (mSimpleScript *models.SimpleScript, message string) {
	mSimpleScript = &models.SimpleScript{}

	message = PbToModel(ctx, mSimpleScript, protoSimpleScript)
	if message != "" {
		message = fmt.Sprintf("Error fetch mSimpleScript: '%v'", message)
		logger.Errorf(ctx, message)

		return mSimpleScript, message
	}

	tmpQBytes, err := json.Marshal(protoSimpleScript.QueryParams)
	if err != nil {
		logger.Errorf(ctx, "failed to marshal in tmpQ: %v", err)
		return mSimpleScript, err.Error()
	}

	mSimpleScript.QueryParams = string(tmpQBytes)

	tmpHBytes, err := json.Marshal(protoSimpleScript.Headers)
	if err != nil {
		logger.Errorf(ctx, "failed to marshal in tmpH: %v", err)
		return mSimpleScript, err.Error()
	}

	mSimpleScript.Headers = string(tmpHBytes)

	if protoSimpleScript.Selectors != nil {
		mSimpleScript.ExprRPS = protoSimpleScript.Selectors.ExprRps
		mSimpleScript.SourceRPS = protoSimpleScript.Selectors.SourceRps
		mSimpleScript.CMTRPS = protoSimpleScript.Selectors.CmtRps

		mSimpleScript.ExprRT = protoSimpleScript.Selectors.ExprRt
		mSimpleScript.SourceRT = protoSimpleScript.Selectors.SourceRt
		mSimpleScript.CMTRT = protoSimpleScript.Selectors.CmtRt

		mSimpleScript.ExprErr = protoSimpleScript.Selectors.ExprErr
		mSimpleScript.SourceErr = protoSimpleScript.Selectors.SourceErr
		mSimpleScript.CMTErr = protoSimpleScript.Selectors.CmtErr
	} else {
		logger.Errorf(ctx, "Error fetch selectors(simpleScript): %v", protoSimpleScript.Name)
	}

	tmpAdditionalEnvBytes, err := json.Marshal(protoSimpleScript.AdditionalEnv)
	if err != nil {
		logger.Errorf(ctx, "failed to marshal in AdditionalEnv: %v", err)
		return mSimpleScript, err.Error()
	}

	mSimpleScript.AdditionalEnv = string(tmpAdditionalEnvBytes)

	return mSimpleScript, message
}
