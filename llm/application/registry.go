package application

import (
	"context"
	"sync"

	"github.com/SecDuckOps/shared/llm/domain"
	"github.com/SecDuckOps/shared/llm/infrastructure"
)

// RegistryAdapter implements the domain.LLMRegistry interface.
type RegistryAdapter struct {
	llms            map[string]domain.LLM
	defaultProvider string
	mu              sync.RWMutex
}

// NewLLMRegistry creates a new thread-safe LLM registry with a fallback provider.
func NewLLMRegistry(cfg domain.Config) (*RegistryAdapter, error) {
	defaultProvider := cfg.Default
	if defaultProvider == "" {
		defaultProvider = "default"
	}
	r := &RegistryAdapter{
		llms:            make(map[string]domain.LLM),
		defaultProvider: defaultProvider,
	}
	r.RegisterFromConfig(cfg.Providers)
	return r, nil
}

// Register adds a new LLM provider to the registry.
func (r *RegistryAdapter) Register(llm domain.LLM) {
	if llm == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llms[llm.Name()] = llm
}

// Lookup returns the registered LLM provider by exact name without fallback.
func (r *RegistryAdapter) Lookup(name string) (domain.LLM, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.llms[name]
	return provider, exists
}

// Get returns the registered LLM provider by name, with O(1) fallback capability.
func (r *RegistryAdapter) Get(name string) domain.LLM {
	if provider, exists := r.Lookup(name); exists {
		return provider
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.llms[r.defaultProvider]
}

// List returns all registered LLM provider names efficiently.
func (r *RegistryAdapter) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.llms))
	for k := range r.llms {
		names = append(names, k)
	}
	return names
}

// MustGet returns the registered LLM provider or panics if not found.
func (r *RegistryAdapter) MustGet(name string) domain.LLM {
	llm := r.Get(name)
	if llm == nil {
		panic("LLM provider not found: " + name)
	}
	return llm
}

// Default returns the default LLM provider.
func (r *RegistryAdapter) Default() domain.LLM {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.llms[r.defaultProvider]
}

func (r *RegistryAdapter) RegisterFromConfig(cfgs map[string]domain.ProviderConfig) {
	for name, cfg := range cfgs {
		if cfg.APIKey == "" && cfg.BaseURL == "" && name != "lmstudio" && name != "ollama" {
			continue // Skip if no API key provided (except for local providers)
		}

		// Handle known providers specifically OR via generic OpenAI-compatible adapter
		switch name {
		case "openai":
			r.Register(infrastructure.NewOpenAIAdapter(cfg.APIKey, cfg.Model))
		case "openrouter":
			r.Register(infrastructure.NewOpenRouterAdapter(cfg.APIKey, cfg.Model))
		case "lmstudio":
			r.Register(infrastructure.NewLMStudioAdapter(cfg.APIKey, cfg.Model, cfg.BaseURL))
		case "google", "gemini":
			adapter, err := infrastructure.NewGeminiAdapter(context.Background(), cfg.APIKey, cfg.Model)
			if err == nil {
				r.Register(adapter)
				if name != adapter.Name() {
					r.mu.Lock()
					r.llms[name] = adapter
					r.mu.Unlock()
				}
			}
		default:
			// Treat everything else with a BaseURL as a custom compatible provider
			if cfg.BaseURL != "" {
				r.Register(infrastructure.NewOpenAICompatibleAdapter(name, cfg.APIKey, cfg.Model, cfg.BaseURL))
			}
		}
	}
}
