package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) Search(ctx context.Context, request *pb.SearchRequest) (*pb.SearchResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "search")

	logger.Infof(ctx, "Successful request search: '%v'", request.String())

	scripts, simpleScripts, scenarios, projects, mess := processing.Search(ctx,
		request.GetSearchQuery(),
		request.GetNotLike(),
		request.GetProjectId(),
		request.GetScenarioId(),
		request.GetStartWith(),
		request.GetEndWith(),
		s.store)
	if mess != "" {
		_ = util.SetModalHeader(ctx)
	}
	rs := &pb.SearchResponse{
		Status:        mess == "",
		Message:       mess,
		Scripts:       scripts,
		SimpleScripts: simpleScripts,
		Scenarios:     scenarios,
		Projects:      projects,
	}

	return rs, nil
}

func (s *Service) PageNum(ctx context.Context, request *pb.PageNumRequest) (*pb.PageNumResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "page_num")

	logger.Infof(ctx, "Successful request page_num: '%v'", request.String())

	pageNum, err := processing.PageNum(ctx,
		request.GetTypeEntry(),
		request.GetId(),
		request.GetLimit(),
		s.data)
	var status = true
	var mes = ""
	if err != nil {
		_ = util.SetModalHeader(ctx)
		return nil, err
	}
	rs := &pb.PageNumResponse{
		PageNum: pageNum,
		Status:  status,
		Message: mes,
	}

	return rs, nil
}
