package infrastructure

import (
	"context"
	"net/http"

	"github.com/SecDuckOps/Shared/llm/domain"
	"github.com/SecDuckOps/Shared/types"
	"github.com/sashabaranov/go-openai"
)

// OpenAICompatibleAdapter implements the domain.LLM interface for any provider
// that supports the OpenAI API specification (e.g., Ollama, vLLM, Groq, etc.).
type OpenAICompatibleAdapter struct {
	providerName string
	client       *openai.Client
	model        string
}

// NewOpenAICompatibleAdapter initializes a generic OpenAI-compatible client.
// It accepts a custom base URL to point to any compatible endpoint.
func NewOpenAICompatibleAdapter(name, apiKey, model, baseURL string) domain.LLM {
	// 1. Configure the OpenAI client with a custom BaseURL
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	// Add generic caching headers
	config.HTTPClient = &http.Client{
		Transport: newHeaderTransport(map[string]string{
			"anthropic-beta": "prompt-caching-2024-07-31",
		}, nil),
	}

	// 2. Return the adapter which satisfies domain.LLM
	return &OpenAICompatibleAdapter{
		providerName: name,
		client:       openai.NewClientWithConfig(config),
		model:        model,
	}
}

// Name returns the identifier for this provider.
func (a *OpenAICompatibleAdapter) Name() string {
	return a.providerName
}

// Generate implements the domain.LLM interface.
func (a *OpenAICompatibleAdapter) Generate(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions) (string, error) {
	reqMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		reqMessages[i] = openai.ChatCompletionMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	model := a.model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    reqMessages,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			req.Temperature = opts.Temperature
		}
	}

	resp, err := a.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", types.Wrapf(err, types.ErrCodeAgentFailed, "%s provider error", a.Name())
	}

	if len(resp.Choices) == 0 {
		return "", types.Newf(types.ErrCodeAgentFailed, "received empty response from %s", a.Name())
	}

	return resp.Choices[0].Message.Content, nil
}

// Stream implements the domain.LLM interface.
func (a *OpenAICompatibleAdapter) Stream(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions) (<-chan domain.ChatChunk, error) {
	ch := make(chan domain.ChatChunk)

	reqMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		reqMessages[i] = openai.ChatCompletionMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	model := a.model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: reqMessages,
		Stream:   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.MaxTokens = opts.MaxTokens
		}
	}

	stream, err := a.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, types.Wrapf(err, types.ErrCodeAgentFailed, "%s streaming error", a.Name())
	}

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					return
				}
				ch <- domain.ChatChunk{Error: err}
				return
			}

			if len(response.Choices) > 0 {
				content := response.Choices[0].Delta.Content
				if content != "" {
					ch <- domain.ChatChunk{Content: content}
				}
			}
		}
	}()

	return ch, nil
}

// HealthCheck verifies connectivity to the provider.
func (a *OpenAICompatibleAdapter) HealthCheck(ctx context.Context) error {
	// Simple check by listing models or just a ping if supported
	_, err := a.client.GetModel(ctx, a.model)
	return err
}

// GenerateJSON implements structured output enforcement.
func (a *OpenAICompatibleAdapter) GenerateJSON(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions, target interface{}) error {
	return generateJSON(ctx, a, messages, opts, target)
}
