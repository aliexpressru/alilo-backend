package processing

import (
	"context"
	"fmt"

	"github.com/aliexpressru/alilo-backend/internal/app/data"
	dataPb "github.com/aliexpressru/alilo-backend/internal/app/datapb"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func Search(ctx context.Context,
	searchQuery string, notLike bool, projectID int32, scenarioID int32, startWith bool, endWith bool,
	pbStore *dataPb.Store) (
	scripts []*pb.Script,
	simpleScripts []*pb.SimpleScript,
	scenarios []*pb.Scenario,
	projects []*pb.Project,
	message string) {
	logger.Infof(ctx, "Send data, search word: '%.200v'", searchQuery)

	defer undecided.WarnTimer(ctx, "Search")()
	var err error
	scripts, simpleScripts, scenarios, projects, err = pbStore.SearchPB(ctx,
		searchQuery, notLike, projectID, scenarioID, startWith, endWith)
	if err != nil {
		message = fmt.Sprintf("pbSearch error: '%v'", err.Error())
		logger.Errorf(ctx, message)

		return scripts, simpleScripts, scenarios, projects, message
	}

	if len(scripts) < 1 && len(simpleScripts) < 1 {
		logger.Info(ctx, "The search did not yield results")
	}

	return scripts, simpleScripts, scenarios, projects, message
}

func PageNum(ctx context.Context,
	typeEntry pb.PageNumRequest_TypesEntry, entryID int32, limit int32, store *data.Store) (
	pageNumber int32, err error) {
	defer undecided.WarnTimer(ctx, "PageNum")()

	pageNumber, err = store.PageNumber(ctx, typeEntry, entryID, limit)
	if err != nil {
		return pageNumber, err
	}

	return pageNumber, nil
}
