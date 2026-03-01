package logger

import (
	"context"

	"github.com/SecDuckOps/shared/ports"
	"go.uber.org/zap"
)

// LogAudit logs an important system action to the audit log (audit.log).
func (l *Logger) LogAudit(ctx context.Context, event string, actor string, action string, resource string, fields ...ports.Field) {
	zapFields := toZapFields(fields)
	zapFields = append(zapFields,
		zap.String("event", event),
		zap.String("actor", actor),
		zap.String("action", action),
		zap.String("resource", resource),
	)
	l.auditZap.Info("Audit Event", l.withContextFields(ctx, zapFields)...)
}
