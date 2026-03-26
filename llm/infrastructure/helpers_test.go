package infrastructure

import (
	"errors"
	"testing"
)

func TestAffordableRetryMaxTokensParsesProviderBudget(t *testing.T) {
	retryMaxTokens, ok := affordableRetryMaxTokens(errors.New("status code: 402, message: You requested up to 2048 tokens, but can only afford 1930."), 2048)
	if !ok {
		t.Fatal("expected affordable retry budget to be detected")
	}
	if retryMaxTokens != 1898 {
		t.Fatalf("expected retry max_tokens 1898, got %d", retryMaxTokens)
	}
}

func TestResolveMaxTokensUsesConservativeDefault(t *testing.T) {
	if got := resolveMaxTokens(nil); got != defaultCompletionMaxTokens {
		t.Fatalf("expected default max_tokens %d, got %d", defaultCompletionMaxTokens, got)
	}
}
