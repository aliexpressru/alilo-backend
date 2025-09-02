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
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
)

func ModelToPBScenario(ctx context.Context, mScenario *models.Scenario) (pbScenario *pb.Scenario, message string) {
	pbScenario = &pb.Scenario{}

	err := ModelToPb(ctx, *mScenario, pbScenario)
	if err != nil {
		logger.Errorf(ctx, "Scenario model to pb: '%v'", err)
	}

	pbScenario.Selectors, err = modelToPbScenarioSelector(ctx, mScenario.Selectors)
	if err != nil {
		logger.Errorf(ctx, "ScenarioSelector model to pb: '%v'", err)

		return pbScenario, ""
	}
	return pbScenario, message
}

func modelToPbScenarioSelector(ctx context.Context, mSelectors string) (
	pbScenarioSelectors []*pb.Scenario_Selector, err error) {
	if mSelectors != "" {
		if json.Valid([]byte(mSelectors)) {
			jsonDecoder := json.NewDecoder(strings.NewReader(mSelectors))

			_, err = jsonDecoder.Token()
			if err != nil {
				logger.Errorf(ctx, "mSelector Err jsonDecoder: %v", err)
				return nil, err
			}

			for jsonDecoder.More() {
				protoMessage := &pb.Scenario_Selector{}

				err = jsonpb.UnmarshalNext(jsonDecoder, protoMessage)
				if err != nil {
					logger.Errorf(ctx, "mSelector UnmarshalNext error: %v", err)
					return nil, err
				}

				pbScenarioSelectors = append(pbScenarioSelectors, protoMessage)
			}
		} else {
			logger.Errorf(ctx, "mSelector Not Valid json: '%v'", mSelectors)

			return nil, errors.New("invalidArgument mSelectors")
		}
	} else {
		logger.Warnf(ctx, "mScriptRuns is empty: '%v'", mSelectors)

		return pbScenarioSelectors, nil
	}

	return pbScenarioSelectors, nil
}

func PBToModelScenario(ctx context.Context, scenario *pb.Scenario) (mScenario *models.Scenario, message string) {
	mScenario = &models.Scenario{}

	mess := PbToModel(ctx, mScenario, scenario)
	if mess != "" {
		message = fmt.Sprintf("Scenario model to pb: '%v'", mess)
		logger.Errorf(ctx, "error conv: %v", message)
	}
	tmpSBytes, err := json.MarshalIndent(scenario.Selectors, "", " ")
	if err != nil {
		logger.Errorf(ctx, "failed to marshal in tmpQ{}: %v", scenario.Selectors, err)
		return mScenario, err.Error()
	}

	mScenario.Selectors = string(tmpSBytes)

	return mScenario, message
}
