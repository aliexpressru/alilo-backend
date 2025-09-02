package conv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"

	//nolint
	"github.com/aarondl/null/v8"
	//nolint
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	jsonMarshaler = jsonpb.Marshaler{EnumsAsInts: false, Indent: "	", EmitDefaults: true}
	// compPid       = regexp.MustCompile(`^"(.*)"$`)
)

func PbToModelRun(ctx context.Context, pbRun *pb.Run) (*models.Run, string) {
	mRun := models.Run{} //fixme: mRun
	logger.Debugf(ctx, "pbToModelRun modelRun: '%+v'", mRun)

	message := PbToModel(ctx, &mRun, pbRun)
	if message != "" {
		message = fmt.Sprintf("Error fetch mRun: '%v'", message)
		logger.Errorf(ctx, message)

		return nil, message
	}

	logger.Debugf(ctx, "pbToModelRun modelRun: '%+v'", mRun)
	mRun.Info = pbRun.Info
	mRun.CreatedAt = pbRun.CreatedAt.AsTime()
	mRun.UpdatedAt = pbRun.UpdatedAt.AsTime()
	mRun.ScriptRuns = PbToModelScriptRuns(ctx, pbRun.ScriptRuns)
	mRun.ReportLink = null.StringFrom(pbRun.ReportLink)

	logger.Debugf(ctx, "pbToModelRun modelRun: '%+v'", mRun)

	return &mRun, message
}

// PbToModelScriptRuns Use the "google.golang.org/protobuf/encoding/protojson" package instead. (staticcheck)
func PbToModelScriptRuns(ctx context.Context, pbScriptRuns []*pb.ScriptRun) string {
	jsonScriptRuns := strings.Builder{}
	jsonScriptRuns.WriteString("[")

	for i, scriptRun := range pbScriptRuns {
		mScriptRun, err := jsonMarshaler.MarshalToString(
			scriptRun,
		) //	todo: перевести, github.com/golang/protobuf/jsonpb is deprecated:
		if err != nil {
			logger.Errorf(ctx, "Error unmarshal pbScriptRun to modelScriptRun: %s", err.Error())
		}

		if i > 0 {
			jsonScriptRuns.WriteString(",\n")
			jsonScriptRuns.WriteString(mScriptRun)
		} else {
			jsonScriptRuns.WriteString("\n")
			jsonScriptRuns.WriteString(mScriptRun)
		}
	}

	jsonScriptRuns.WriteString("]")
	logger.Debugf(ctx, "PbToModelScriptRuns pbScriptRuns: '%s'", pbScriptRuns)

	result := jsonScriptRuns.String()
	logger.Debugf(ctx, "PbToModelScriptRuns jsonScriptRuns: '%s'", result)

	return result
}

func ModelToPBRun(ctx context.Context, mRun *models.Run) (run *pb.Run, err error) {
	run = &pb.Run{}

	err = ModelToPb(ctx, *mRun, run)
	if err != nil {
		err = errors.Wrapf(err, "Run model to pb error")
		logger.Error(ctx, err)

		return run, err
	}

	logger.Debugf(ctx, "-1--ModelToPBRun mRun  '%+v'", mRun)
	logger.Debugf(ctx, "-1--ModelToPBRun pbRun '%+v'", run)
	run.ScriptRuns = ModelToPBScriptRuns(ctx, mRun.ScriptRuns)
	run.Info = mRun.Info
	run.CreatedAt = timestamppb.New(mRun.CreatedAt)
	run.UpdatedAt = timestamppb.New(mRun.UpdatedAt)
	run.PercentageOfTarget = mRun.PercentageOfTarget
	run.UserName = mRun.UserName
	run.PreferredUserName = mRun.PreferredUserName
	run.ReportLink = mRun.ReportLink.String
	logger.Debugf(ctx, "-2--ModelToPBRun mRun  '%+v'", mRun)
	logger.Debugf(ctx, "-2--ModelToPBRun pbRun '%+v'", run)

	return run, nil
}

func ModelToPBScriptRuns(ctx context.Context, mScriptRuns string) (scriptRuns []*pb.ScriptRun) {
	if mScriptRuns != "" {
		if json.Valid([]byte(mScriptRuns)) {
			jsonDecoder := json.NewDecoder(strings.NewReader(mScriptRuns))

			_, err := jsonDecoder.Token()
			if err != nil {
				logger.Error(ctx, "mScriptRun Err jsonDecoder: ", err)
			}

			for jsonDecoder.More() {
				protoMessage := &pb.ScriptRun{}

				err = jsonpb.UnmarshalNext(jsonDecoder, protoMessage)
				if err != nil {
					logger.Errorf(ctx, "mScriptRun UnmarshalNext error: %v", err)
				}

				scriptRuns = append(scriptRuns, protoMessage)
			}
		} else {
			logger.Errorf(ctx, "mScriptRun Not Valid json: '%v'", mScriptRuns)
			return []*pb.ScriptRun{}
		}
	} else {
		logger.Warnf(ctx, "mScriptRuns is empty: '%v'", mScriptRuns)
		return []*pb.ScriptRun{}
	}

	return scriptRuns
}

func ModelToPBRuns(ctx context.Context, mRuns models.RunSlice) (
	returnRuns []*pb.Run, message string) {
	if len(mRuns) > 0 {
		logger.Infof(ctx, "ModelToPBRuns. Prepared len: '%+v'", len(mRuns))
		returnRuns = make([]*pb.Run, 0, len(mRuns))

		for _, mRun := range mRuns {
			pbRun, err := ModelToPBRun(ctx, mRun)
			if err == nil {
				returnRuns = append(returnRuns, pbRun)
			} else {
				err = errors.Wrapf(err, "ModelToPBRuns ERROR RunID: '%v'", mRun.RunID)
				logger.Errorf(ctx, "ModelToPBRun: '%v' '%v'", mRun, err)
				message = fmt.Sprint(message, err.Error())
			}
		}
	} else {
		logger.Warnf(ctx, "mRuns is empty")
	}

	logger.Infof(ctx, "ModelToPBRuns. Return len: '%+v'", len(returnRuns))

	return returnRuns, message
}
