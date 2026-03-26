package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SecDuckOps/shared/llm/domain"
)

func TestOpenAICompatibleAdapterGenerateUsesConservativeDefaultMaxTokens(t *testing.T) {
	var gotMaxTokens int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var req struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotMaxTokens = req.MaxTokens
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1,
			"model":   "test-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "ok",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		})
	}))
	defer server.Close()

	llm := NewOpenAICompatibleAdapter("profile:default", "test-key", "test-model", server.URL)
	_, err := llm.Generate(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}
	if gotMaxTokens != defaultCompletionMaxTokens {
		t.Fatalf("expected default max_tokens to be %d, got %d", defaultCompletionMaxTokens, gotMaxTokens)
	}
}

func TestOpenAICompatibleAdapterGenerateRespectsMaxTokensOverride(t *testing.T) {
	var gotMaxTokens int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotMaxTokens = req.MaxTokens
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1,
			"model":   "test-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "ok",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		})
	}))
	defer server.Close()

	llm := NewOpenAICompatibleAdapter("profile:default", "test-key", "test-model", server.URL)
	_, err := llm.Generate(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "hello"},
	}, &domain.GenerateOptions{MaxTokens: 1200})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}
	if gotMaxTokens != 1200 {
		t.Fatalf("expected override max_tokens to be 1200, got %d", gotMaxTokens)
	}
}

func TestOpenAICompatibleAdapterGenerateRetriesWithAffordableTokenBudget(t *testing.T) {
	var requested []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requested = append(requested, req.MaxTokens)
		if len(requested) == 1 {
			w.WriteHeader(http.StatusPaymentRequired)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "This request requires more credits, or fewer max_tokens. You requested up to 2048 tokens, but can only afford 1930.",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1,
			"model":   "test-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "ok",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     1,
				"completion_tokens": 1,
				"total_tokens":      2,
			},
		})
	}))
	defer server.Close()

	llm := NewOpenAICompatibleAdapter("profile:default", "test-key", "test-model", server.URL)
	result, err := llm.Generate(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "hello"},
	}, &domain.GenerateOptions{MaxTokens: 2048})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}
	if result.Content != "ok" {
		t.Fatalf("expected content %q, got %q", "ok", result.Content)
	}
	if len(requested) != 2 {
		t.Fatalf("expected two attempts, got %v", requested)
	}
	if requested[1] >= requested[0] {
		t.Fatalf("expected retry max_tokens to be lower, got %v", requested)
	}
}
