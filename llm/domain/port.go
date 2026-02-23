package domain

import "context"

type LLM interface {
	// Provider name (openai, ollama, openrouter, etc)
	Name() string

	// Generate full response (blocking)
	Generate(
		ctx context.Context,
		messages []Message,
		opts *GenerateOptions,
	) (string, error)

	// Stream response (non-blocking, low latency)
	Stream(
		ctx context.Context,
		messages []Message,
		opts *GenerateOptions,
	) (<-chan ChatChunk, error)

	// Optional but very useful
	HealthCheck(ctx context.Context) error

	// GenerateJSON handles structured output enforcement
	GenerateJSON(ctx context.Context, messages []Message, opts *GenerateOptions, target interface{}) error
}

type LLMRegistry interface {
	Register(llm LLM)
	Get(name string) LLM
	MustGet(name string) LLM
	List() []string
	Default() LLM
}
