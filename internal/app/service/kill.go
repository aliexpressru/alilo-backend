package service

import (
	"context"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/aliexpressru/alilo-backend/pkg/util/ssh"
	"github.com/aliexpressru/alilo-backend/pkg/util/undecided"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) KillProcess(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	ctx = undecided.NewContextWithMarker(ctx, LoggerContextKey, "kill_process")

	if md, b := metadata.FromIncomingContext(ctx); b {
		user := md.Get("x-identification")
		logger.Warnf(ctx, "X-user: %v", len(user))
	}

	logger.Infof(ctx, "Successful request KillProcessRequest: '%v'", request.String())
	ssh.SendCommandForAll(ctx, "kill")

	return new(emptypb.Empty), nil
}
