package logger

import (
	"context"

	"github.com/SecDuckOps/shared/ports"
	"github.com/SecDuckOps/shared/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger and provides specialized context-aware logging.
type Logger struct {
	zap         *zap.Logger
	auditZap    *zap.Logger
	securityZap *zap.Logger
	service     string
}

// New creates a new production-ready structured logger dumping to the logs directory.
func New(service string, level string) (*Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "msg" // Use 'msg' consistently for Elasticsearch

	logDir := "logs"
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// App and Error logs rotators
	generalRotator, err := createRotator(logDir, "app.log")
	if err != nil {
		return nil, err
	}

	errorRotator, err := createRotator(logDir, "error.log")
	if err != nil {
		return nil, err
	}

	// Audit & Security rotators
	auditRotator, err := createRotator(logDir, "audit.log")
	if err != nil {
		return nil, err
	}

	securityRotator, err := createRotator(logDir, "security.log")
	if err != nil {
		return nil, err
	}

	// Info vs Error routing for app logs
	infoLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapLevel && lvl < zapcore.ErrorLevel
	})
	errorLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	core := zapcore.NewTee(
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(generalRotator), infoLevelEnabler),
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(errorRotator), errorLevelEnabler),
	)

	return &Logger{
		zap:         zap.New(core).With(zap.String("service", service)),
		auditZap:    zap.New(zapcore.NewCore(jsonEncoder, zapcore.AddSync(auditRotator), zapcore.DebugLevel)).With(zap.String("service", service)),
		securityZap: zap.New(zapcore.NewCore(jsonEncoder, zapcore.AddSync(securityRotator), zapcore.DebugLevel)).With(zap.String("service", service)),
		service:     service,
	}, nil
}

// Debug logs a debug message with context fields.
func (l *Logger) Debug(ctx context.Context, event string, msg string, fields ...ports.Field) {
	zapFields := toZapFields(fields)
	zapFields = append(zapFields, zap.String("event", event))
	l.zap.Debug(msg, l.withContextFields(ctx, zapFields)...)
}

// Info logs an info message with context fields.
func (l *Logger) Info(ctx context.Context, event string, msg string, fields ...ports.Field) {
	zapFields := toZapFields(fields)
	zapFields = append(zapFields, zap.String("event", event))
	l.zap.Info(msg, l.withContextFields(ctx, zapFields)...)
}

// ErrorErr logs an error with automatic level mapping and context extraction.
func (l *Logger) ErrorErr(ctx context.Context, event string, err error, msg string, fields ...ports.Field) {
	zapFields := toZapFields(fields)
	zapFields = append(zapFields, zap.String("event", event))

	if err == nil {
		l.zap.Error(msg, l.withContextFields(ctx, zapFields)...)
		return
	}

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
	_ = l.auditZap.Sync()
	_ = l.securityZap.Sync()
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
