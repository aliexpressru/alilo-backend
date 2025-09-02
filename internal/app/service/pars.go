package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/internal/app/processing"
	"github.com/aliexpressru/alilo-backend/internal/app/util"
	pb "github.com/aliexpressru/alilo-backend/pkg/pb/qa/loadtesting/alilo/backend/v1"
	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
)

func (s *Service) ParseCUrl(ctx context.Context, request *pb.ParseCUrlRequest) (*pb.ParseCUrlResponse, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "parse_curl")

	logger.Infof(ctx, "Successful request ParseCUrl: '%v'", request.String())
	json, message := processing.ParseCUrl(ctx, request.GetCurl())
	if message != "" {
		_ = util.SetModalHeader(ctx)
	}

	return &pb.ParseCUrlResponse{
		Status:  message == "",
		Message: message,
		Json:    json,
	}, nil
}
