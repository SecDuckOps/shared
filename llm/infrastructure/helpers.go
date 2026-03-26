package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/SecDuckOps/shared/llm/domain"
	"github.com/SecDuckOps/shared/types"
)

const defaultCompletionMaxTokens = 1024

var affordableTokenBudgetPattern = regexp.MustCompile(`(?i)can only afford\s+(\d+)`)

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

func resolveMaxTokens(opts *domain.GenerateOptions) int {
	if opts != nil && opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return defaultCompletionMaxTokens
}

func affordableRetryMaxTokens(err error, requested int) (int, bool) {
	if err == nil || requested <= 1 {
		return 0, false
	}

	match := affordableTokenBudgetPattern.FindStringSubmatch(err.Error())
	if len(match) != 2 {
		return 0, false
	}

	affordable, parseErr := strconv.Atoi(match[1])
	if parseErr != nil || affordable <= 0 {
		return 0, false
	}
	if affordable >= requested {
		return 0, false
	}

	retryMaxTokens := affordable
	if affordable > 128 {
		retryMaxTokens = affordable - 32
	}
	if retryMaxTokens <= 0 {
		return 0, false
	}
	return retryMaxTokens, true
}

// generateJSON handles structured output enforcement by stripping markdown and unmarshaling.
func generateJSON(ctx context.Context, llm domain.LLM, messages []domain.Message, opts *domain.GenerateOptions, target interface{}) error {
	result, err := llm.Generate(ctx, messages, opts)
	if err != nil {
		return err
	}

	content := result.Content

	// Clean up potential markdown formatting block
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		// Also handle generic code blocks if they contain JSON
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	if err := json.Unmarshal([]byte(content), target); err != nil {
		return types.Wrap(err, types.ErrCodeInvalidInput, "invalid llm json response").
			WithContext("raw_response", content)
	}

	return nil
}
