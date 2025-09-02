package mw

import (
	"context"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// LoggingUnaryServerInterceptor logging unary server interceptor.
func LoggingUnaryServerInterceptor(opts ...grpc_zap.Option) grpc.UnaryServerInterceptor {
	return grpc_zap.UnaryServerInterceptor(
		zap.L(),
		append([]grpc_zap.Option{
			grpc_zap.WithMessageProducer(DefaultMessageProducer),
		}, opts...)...,
	)
}

// LoggingStreamServerInterceptor logging stream server interceptor.
func LoggingStreamServerInterceptor(opts ...grpc_zap.Option) grpc.StreamServerInterceptor {
	return grpc_zap.StreamServerInterceptor(
		zap.L(),
		append([]grpc_zap.Option{
			grpc_zap.WithMessageProducer(DefaultMessageProducer),
		}, opts...)...,
	)
}

// DefaultMessageProducer writes the default message
func DefaultMessageProducer(
	ctx context.Context,
	msg string,
	level zapcore.Level,
	code codes.Code,
	err error,
	duration zapcore.Field,
) {
	// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
	extractLogger := ctxzap.Extract(ctx)
	extractLogger.Check(level, msg).Write(
		zap.Error(err),
		zap.String("grpc.code", code.String()),
		duration,
	)
}
