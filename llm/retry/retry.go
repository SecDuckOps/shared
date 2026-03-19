// Package retry provides a transparent retry wrapper around any LLM implementation.
// Retries on transient errors with exponential backoff + jitter.
// Honours provider Retry-After headers (mirrors duckops agent-core/src/retry.rs).
package retry

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SecDuckOps/shared/llm/domain"
	"github.com/SecDuckOps/shared/types"
)

// Config controls retry behaviour.
type Config struct {
	MaxAttempts int           // total attempts (default 3)
	BaseDelay   time.Duration // initial backoff (default 500ms)
	MaxDelay    time.Duration // cap on backoff (default 30s)
	Multiplier  float64       // backoff multiplier (default 2.0)
	JitterFrac  float64       // jitter fraction 0.0–1.0 (default 0.25)
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterFrac:  0.25,
	}
}

// RetryLLM wraps an LLM with transparent retry logic.
type RetryLLM struct {
	inner  domain.LLM
	config Config
}

func Wrap(llm domain.LLM, cfg Config) *RetryLLM { return &RetryLLM{inner: llm, config: cfg} }
func WrapDefault(llm domain.LLM) *RetryLLM      { return Wrap(llm, DefaultConfig()) }

func (r *RetryLLM) Name() string                          { return r.inner.Name() }
func (r *RetryLLM) Model() string                         { return r.inner.Model() }
func (r *RetryLLM) HealthCheck(ctx context.Context) error { return r.inner.HealthCheck(ctx) }

func (r *RetryLLM) Generate(ctx context.Context, msgs []domain.Message, opts *domain.GenerateOptions) (domain.GenerationResult, error) {
	var result domain.GenerationResult
	var err error
	r.withRetry(ctx, func() error {
		result, err = r.inner.Generate(ctx, msgs, opts)
		return err
	})
	return result, err
}

// Stream is not retried — stateful streaming cannot be safely replayed.
func (r *RetryLLM) Stream(ctx context.Context, msgs []domain.Message, opts *domain.GenerateOptions) (<-chan domain.ChatChunk, error) {
	return r.inner.Stream(ctx, msgs, opts)
}

func (r *RetryLLM) GenerateJSON(ctx context.Context, msgs []domain.Message, opts *domain.GenerateOptions, target interface{}) error {
	var err error
	r.withRetry(ctx, func() error {
		err = r.inner.GenerateJSON(ctx, msgs, opts, target)
		return err
	})
	return err
}

// withRetry executes fn up to MaxAttempts times.
// On each retryable failure it computes the sleep duration by:
//  1. Parsing the Retry-After or Retry-After-Ms header from the error (if present)
//  2. Falling back to exponential backoff with jitter
//
// This mirrors duckops agent-core/src/retry.rs resolve_retry_delay_ms().
func (r *RetryLLM) withRetry(ctx context.Context, fn func() error) {
	baseDelay := r.config.BaseDelay

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return
		}
		if !isRetryable(err) {
			return
		}
		if attempt == r.config.MaxAttempts-1 {
			return
		}

		// ── Compute sleep: prefer provider header, fall back to backoff ──
		sleep := parseRetryAfter(err)
		if sleep == 0 {
			sleep = exponentialBackoff(baseDelay, r.config.MaxDelay, r.config.Multiplier, r.config.JitterFrac, attempt)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		}
	}
}

// parseRetryAfter extracts a wait duration from error messages that embed
// Retry-After or Retry-After-Ms HTTP header values.
//
// Providers typically surface these as:
//
//	"rate_limit_error: retry after 30s"
//	"429 Too Many Requests, retry-after-ms: 1500"
func parseRetryAfter(err error) time.Duration {
	if err == nil {
		return 0
	}
	msg := strings.ToLower(err.Error())

	// retry-after-ms: <milliseconds>  (highest priority — matches duckops)
	if idx := strings.Index(msg, "retry-after-ms:"); idx != -1 {
		rest := strings.TrimSpace(msg[idx+len("retry-after-ms:"):])
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			if ms, parseErr := strconv.ParseInt(fields[0], 10, 64); parseErr == nil && ms > 0 {
				return time.Duration(ms) * time.Millisecond
			}
		}
	}

	// retry-after: <seconds>
	if idx := strings.Index(msg, "retry-after:"); idx != -1 {
		rest := strings.TrimSpace(msg[idx+len("retry-after:"):])
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			if secs, parseErr := strconv.ParseInt(fields[0], 10, 64); parseErr == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}

	// "retry after Ns" / "retry after Ns." pattern (Anthropic SDK messages)
	if idx := strings.Index(msg, "retry after "); idx != -1 {
		rest := strings.TrimSpace(msg[idx+len("retry after "):])
		rest = strings.TrimSuffix(rest, ".")
		rest = strings.TrimSuffix(rest, "s")
		if secs, parseErr := strconv.ParseInt(rest, 10, 64); parseErr == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}

	return 0
}

// exponentialBackoff returns BaseDelay * Multiplier^attempt + jitter, capped at MaxDelay.
func exponentialBackoff(base, max time.Duration, multiplier, jitterFrac float64, attempt int) time.Duration {
	delay := float64(base)
	for i := 0; i < attempt; i++ {
		delay *= multiplier
	}
	if delay > float64(max) {
		delay = float64(max)
	}
	jitter := delay * jitterFrac * rand.Float64()
	total := time.Duration(delay + jitter)
	if total > max {
		total = max
	}
	return total
}

// isRetryable returns true for transient errors worth retrying.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	msg := strings.ToLower(err.Error())

	// Rate limit signals
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "retry after") {
		return true
	}
	// Server errors
	if strings.Contains(msg, "500") || strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") || strings.Contains(msg, "504") {
		return true
	}
	// Network transients
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "eof") ||
		strings.Contains(msg, "timeout") || strings.Contains(msg, "temporary") {
		return true
	}
	// AppError codes
	var appErr *types.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case types.ErrCodeInternal, types.ErrCodeExecutionFailed:
			return true
		}
	}
	// HTTP status embedded in error
	var httpErr interface{ StatusCode() int }
	if errors.As(err, &httpErr) {
		code := httpErr.StatusCode()
		return code == http.StatusTooManyRequests || code >= 500
	}
	return false
}
