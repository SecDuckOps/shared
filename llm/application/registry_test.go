package application

import (
	"testing"

	"github.com/SecDuckOps/shared/llm/domain"
)

func TestLookupDoesNotFallbackToDefault(t *testing.T) {
	registry, err := NewLLMRegistry(domain.Config{
		Default: "openai",
		Providers: map[string]domain.ProviderConfig{
			"openai": {
				APIKey: "test-openai-key",
				Model:  "gpt-4o-mini",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewLLMRegistry() failed: %v", err)
	}

	if _, ok := registry.Lookup("missing"); ok {
		t.Fatal("expected exact lookup to fail for missing provider")
	}
	if got := registry.Get("missing"); got == nil || got.Name() != "openai" {
		t.Fatalf("expected Get() fallback to default openai, got %#v", got)
	}
}

func TestRegisterFromConfigRegistersGoogleAlias(t *testing.T) {
	registry, err := NewLLMRegistry(domain.Config{
		Default: "google",
		Providers: map[string]domain.ProviderConfig{
			"google": {
				APIKey: "google-test-key",
				Model:  "gemini-1.5-flash",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewLLMRegistry() failed: %v", err)
	}

	if _, ok := registry.Lookup("google"); !ok {
		t.Fatal("expected google alias to be registered")
	}
	if _, ok := registry.Lookup("gemini"); !ok {
		t.Fatal("expected gemini canonical provider to be registered")
	}
}
