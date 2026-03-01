package logger

import (
	"context"

	"github.com/SecDuckOps/shared/ports"
	"go.uber.org/zap"
)

// LogSecurity logs security-related events to security.log.
func (l *Logger) LogSecurity(ctx context.Context, event string, ip string, reason string, fields ...ports.Field) {
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("event", event),
		zap.String("ip", ip),
		zap.String("reason", reason),
	)
	l.securityZap.Warn("Security Event", l.withContextFields(ctx, zapFields)...)
}
