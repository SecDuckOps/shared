package ports

import (
	"context"
)

// Field represents a structured log field, decoupling the port from specific logger implementations.
type Field struct {
	Key   string
	Value interface{}
}

// Logger defines the interface for the centralized logging system.
// It is Zap-independent and context-aware to support correlation IDs.
type Logger interface {
	Debug(ctx context.Context, event string, msg string, fields ...Field)
	Info(ctx context.Context, event string, msg string, fields ...Field)
	ErrorErr(ctx context.Context, event string, err error, msg string, fields ...Field)
	LogAudit(ctx context.Context, event string, actor string, action string, resource string, fields ...Field)
	LogSecurity(ctx context.Context, event string, ip string, reason string, fields ...Field)
}
