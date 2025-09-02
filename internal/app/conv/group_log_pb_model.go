package conv

import (
	"context"
	"fmt"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func ModelGroupLogToPB(ctx context.Context, mRunLog *models.GroupLog) (runLog *pb.GroupLog, err error) {
	runLog = &pb.GroupLog{}
	if mRunLog != nil {
		er := ModelToPb(ctx, *mRunLog, runLog)
		if er != nil {
			err = fmt.Errorf("RunLog model to pb: '%+v'", er)
			logger.Errorf(ctx, err.Error())

			return nil, err
		}
	} else {
		return nil, fmt.Errorf("model RunLog is nil")
	}

	return runLog, err
}

func PBGroupLogToModel(ctx context.Context, pbRunLog *pb.GroupLog) (mRunLog *models.GroupLog, err error) {
	mRunLog = &models.GroupLog{}
	if pbRunLog != nil {
		message := PbToModel(ctx, mRunLog, pbRunLog)
		if message != "" {
			err = fmt.Errorf("RunLog pb to model: '%v'", message)
			logger.Errorf(ctx, message)

			return nil, err
		}
	} else {
		return nil, fmt.Errorf("pb RunLog is nil")
	}

	return mRunLog, err
}
