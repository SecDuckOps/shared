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
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	ErrorErr(ctx context.Context, err error, msg string, fields ...Field)
}
