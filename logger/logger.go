package logger

import (
	"context"
	"fmt"

	"github.com/SecDuckOps/shared/ports"
	"github.com/SecDuckOps/shared/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger and provides specialized context-aware logging.
type Logger struct {
	zap     *zap.Logger
	service string
}

// New creates a new production-ready structured logger.
func New(service string, level string) (*Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	return &Logger{
		zap:     l.With(zap.String("service", service)),
		service: service,
	}, nil
}

// Debug logs a debug message with context fields.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...ports.Field) {
	l.zap.Debug(msg, l.withContextFields(ctx, toZapFields(fields))...)
}

// Info logs an info message with context fields.
func (l *Logger) Info(ctx context.Context, msg string, fields ...ports.Field) {
	l.zap.Info(msg, l.withContextFields(ctx, toZapFields(fields))...)
}

// ErrorErr logs an error with automatic level mapping and context extraction.
func (l *Logger) ErrorErr(ctx context.Context, err error, msg string, fields ...ports.Field) {
	if err == nil {
		l.zap.Error(msg, l.withContextFields(ctx, toZapFields(fields))...)
		return
	}

	zapFields := toZapFields(fields)
	level := zapcore.ErrorLevel

	if appErr, ok := err.(*types.AppError); ok {
		level = mapLevel(appErr.Code)
		zapFields = append(zapFields,
			zap.String("error_code", string(appErr.Code)),
			zap.String("error_message", appErr.Message),
			zap.Reflect("error_context", appErr.Context),
			zap.Time("error_timestamp", appErr.Timestamp),
		)
		if appErr.Cause != nil {
			zapFields = append(zapFields, zap.String("cause", appErr.Cause.Error()))
		}
	} else {
		zapFields = append(zapFields,
			zap.String("error_code", string(types.ErrCodeInternal)),
			zap.Error(err),
		)
	}

	l.zap.Log(level, msg, l.withContextFields(ctx, zapFields)...)
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Helpers

func toZapFields(fields []ports.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}

func (l *Logger) withContextFields(ctx context.Context, fields []zap.Field) []zap.Field {
	if ctx == nil {
		return fields
	}
	if cid, ok := ctx.Value("correlation_id").(string); ok {
		return append(fields, zap.String("correlation_id", cid))
	}
	return fields
}

func mapLevel(code types.ErrorCode) zapcore.Level {
	switch code {
	case types.ErrCodeInvalidInput, types.ErrCodeNotFound:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}
