package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SecDuckOps/shared/llm/domain"
	"github.com/SecDuckOps/shared/types"
)

// headerTransport is an http.RoundTripper that adds custom headers to every request.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

// newHeaderTransport creates an http.RoundTripper that injects the given headers.
func newHeaderTransport(headers map[string]string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &headerTransport{
		base:    base,
		headers: headers,
	}
}

// generateJSON handles structured output enforcement by stripping markdown and unmarshaling.
func generateJSON(ctx context.Context, llm domain.LLM, messages []domain.Message, opts *domain.GenerateOptions, target interface{}) error {
	resp, err := llm.Generate(ctx, messages, opts)
	if err != nil {
		return err
	}

	// Clean up potential markdown formatting block
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```json") {
		resp = strings.TrimPrefix(resp, "```json")
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	} else if strings.HasPrefix(resp, "```") {
		// Also handle generic code blocks if they contain JSON
		resp = strings.TrimPrefix(resp, "```")
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	}

	if err := json.Unmarshal([]byte(resp), target); err != nil {
		return types.Wrap(err, types.ErrCodeInvalidInput, "invalid llm json response").
			WithContext("raw_response", resp)
	}

	return nil
}
