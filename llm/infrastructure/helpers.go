package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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

func normalizeOpenAICompatibleBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return ensureV1BaseURL(trimmed)
	}

	segments := pathSegments(parsed.Path)
	for len(segments) > 0 {
		switch {
		case len(segments) >= 2 && strings.EqualFold(segments[len(segments)-2], "chat") && strings.EqualFold(segments[len(segments)-1], "completions"):
			segments = segments[:len(segments)-2]
		case strings.EqualFold(segments[len(segments)-1], "chat"):
			segments = segments[:len(segments)-1]
		default:
			goto normalized
		}
	}

normalized:
	parsed.Path = ensureV1Path(segments)
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/")
}

func pathSegments(path string) []string {
	if path == "" || path == "/" {
		return nil
	}
	rawSegments := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

func ensureV1Path(segments []string) string {
	if len(segments) == 0 {
		return "/v1"
	}
	if strings.EqualFold(segments[len(segments)-1], "v1") {
		return "/" + strings.Join(segments, "/")
	}
	return "/" + strings.Join(append(segments, "v1"), "/")
}

func ensureV1BaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(baseURL), "/v1") {
		return baseURL
	}
	return baseURL + "/v1"
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
