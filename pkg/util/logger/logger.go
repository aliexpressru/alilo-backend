package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

type ctxKey struct{}

func init() {
	// Initialize with a default level (e.g., InfoLevel)
	if err := Init(zapcore.InfoLevel); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}

// Init initializes the logger
func Init(level zapcore.Level) error {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)

	var err error
	Log, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// Logger returns the global logger instance
func Logger() *zap.Logger {
	if Log == nil {
		panic("logger not initialized - call logger.Init() first")
	}
	return Log
}

// Sync flushes any buffered log entries
func Sync() error {
	return Log.Sync()
}

// WithContext returns a logger with context fields
func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return Log
	}

	if ctxFields, ok := ctx.Value(ctxKey{}).([]zap.Field); ok {
		return Log.With(ctxFields...)
	}

	return Log
}

// Info logs at info level with context
func Info(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Info(args...)
}

// Infof logs formatted message at info level with context
func Infof(ctx context.Context, template string, args ...interface{}) {
	WithContext(ctx).Sugar().Infof(template, args...)
}

// Warn logs at warn level with context
func Warn(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Warn(args...)
}

// Warnf logs formatted message at warn level with context
func Warnf(ctx context.Context, template string, args ...interface{}) {
	WithContext(ctx).Sugar().Warnf(template, args...)
}

// Error logs at error level with context
func Error(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Error(args...)
}

// Errorf logs formatted message at error level with context
func Errorf(ctx context.Context, template string, args ...interface{}) {
	WithContext(ctx).Sugar().Errorf(template, args...)
}

func Fatalf(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Fatal(args...)
}

func Debug(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Debug(args...)
}

func Debugf(ctx context.Context, args ...interface{}) {
	WithContext(ctx).Sugar().Debug(args...)
}

// ToContext adds a logger with specific fields to the context
func ToContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// WithFields adds fields to context for logging
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if existing, ok := ctx.Value(ctxKey{}).([]zap.Field); ok {
		return context.WithValue(ctx, ctxKey{}, append(existing, fields...))
	}

	return context.WithValue(ctx, ctxKey{}, fields)
}
