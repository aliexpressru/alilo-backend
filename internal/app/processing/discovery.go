package processing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aliexpressru/alilo-backend/internal/app/data"
	"github.com/aliexpressru/alilo-backend/internal/app/datapb"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func GetEndpoints(ctx context.Context, dataStore *data.Store, pbStore *datapb.Store) ([]*pb.Endpoint, string) {
	limit := int32(100)

	var endpoints []*pb.Endpoint

	countRuns, err := dataStore.GetCountRunsByStatus(ctx, pb.Run_STATUS_RUNNING)
	if err != nil {
		logger.Errorf(ctx, "Getting count endpoints error: '%+v'", err)
		return []*pb.Endpoint{}, err.Error()
	}

	logger.Infof(ctx, "Endpoints count: '%v'", countRuns)
	pages := data.CalculateTotalPages(countRuns, limit)
	logger.Infof(ctx, "Pages count: '%v'", countRuns)

	for i := int64(1); i <= pages; i++ {
		//nolint:gosec
		runs, _, message := pbStore.GetRunsByStatus(ctx, pb.Run_STATUS_RUNNING, limit, int32(i))
		if message != "" {
			logger.Errorf(ctx, "Getting Endpoints message: '%v'", message)
			return []*pb.Endpoint{}, message
		}

		for _, run := range runs {
			runURL := undecided.RunLink(ctx, run.GetRunId())

			for _, scriptRun := range run.GetScriptRuns() {
				if scriptRun.GetStatus() != pb.ScriptRun_STATUS_RUNNING {
					continue
				}

				scriptTitle := ""

				agent := scriptRun.GetAgent()
				if agent.GetHostName() == "" && scriptRun.GetPortPrometheus() == "" {
					continue
				}

				switch scriptRun.TypeScriptRun {
				case pb.ScriptRun_TYPE_SCRIPT_RUN_EXTENDED_UNSPECIFIED:
					scriptTitle = scriptRun.GetScript().GetName()
				case pb.ScriptRun_TYPE_SCRIPT_RUN_SIMPLE:
					scriptTitle = scriptRun.GetSimpleScript().GetName()
				default:
					logger.Warnf(ctx, "Unknown TypeScriptRun '%v'", scriptRun.TypeScriptRun)
					continue
				}

				endpoints = append(endpoints, &pb.Endpoint{
					Targets: []string{
						fmt.Sprintf("%v:%v", agent.HostName, scriptRun.GetPortPrometheus()),
					},
					Labels: &pb.Labels{
						RunId:         strconv.FormatInt(int64(run.GetRunId()), 10),
						ScriptTitle:   scriptTitle,
						ScenarioTitle: run.GetTitle(),
						RanByUser:     "",
						ScriptRunId:   strconv.FormatInt(int64(scriptRun.GetRunScriptId()), 10),
						ScriptType:    scriptRun.TypeScriptRun.String(),
						RunUrl:        runURL,
					},
				})
			}
		}
	}

	return endpoints, ""
}
