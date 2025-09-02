package conv

import (
	"context"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

func ModelsToPbTraces(ctx context.Context, mTraces []*models.Trace) (pbTraces []*pb.Trace, err error) {
	pbTraces = []*pb.Trace{}

	logger.Infof(ctx, "len(mTraces):  %v", len(mTraces))

	for _, mTrace := range mTraces {
		var pbTrace *pb.Trace

		pbTrace, err = ModelsToPbTrace(ctx, mTrace)
		if err != nil {
			logger.Warnf(ctx, "modelsToPbTrace error:%v", err)
			continue
		}

		pbTraces = append(pbTraces, pbTrace)
	}

	logger.Infof(ctx, "len(ReturnPBTraces): %v", len(pbTraces))

	return pbTraces, err
}

func ModelsToPbTrace(ctx context.Context, mTrace *models.Trace) (*pb.Trace, error) {
	ag := &pb.Trace{}

	err := ModelToPb(ctx, mTrace, ag)
	if err != nil {
		logger.Warnf(ctx, "ModelsToPbTrace -> ModelToPb ERROR: '%v'", err)
		return nil, err
	}
	logger.Debug(ctx, "ModelsToPbTrace -> ModelToPb Success")

	return ag, err
}
