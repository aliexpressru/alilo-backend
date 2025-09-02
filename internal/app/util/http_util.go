package util

import (
	"context"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	XShowNotificationHeader = "SHOW-NOTIFICATION"
)

// SetModalHeader Устанавливает header для показа модалки на front /**
func SetModalHeader(ctx context.Context) error {
	err := grpc.SetHeader(ctx, metadata.Pairs(XShowNotificationHeader, "true"))
	if err != nil {
		logger.Warnf(ctx, "Failed to set header %v", err)
		return err
	}

	return nil
}
