package conv

import (
	"context"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
)

func ModelToPBCmd(ctx context.Context, mCmd *models.Command) (pbCmd *pb.Command, err error) {
	pbCmd = &pb.Command{}

	err = ModelToPb(ctx, mCmd, pbCmd)
	if err != nil {
		err = errors.Wrapf(err, "Command model to pb error")
		logger.Error(ctx, "Errrr: ", err)

		return pbCmd, err
	}

	logger.Debugf(ctx, "-1--ModelToPBCmd mCmd  '%+v'", mCmd)
	logger.Debugf(ctx, "-1--ModelToPBCmd pbCmd '%+v'", pbCmd)
	pbCmd.Status = pb.Command_Status(pb.Command_Status_value[mCmd.Status])
	pbCmd.ErrorDescription = mCmd.ErrorDescription
	pbCmd.PercentageOfTarget = mCmd.PercentageOfTarget.Int32
	logger.Debugf(ctx, "-2--ModelToPBCmd mCmd  '%+v'", mCmd)
	logger.Debugf(ctx, "-2--ModelToPBCmd pbCmd '%+v'", pbCmd)

	return pbCmd, nil
}
