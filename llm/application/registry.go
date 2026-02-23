package application

import (
	"sync"

	"github.com/SecDuckOps/Shared/llm/domain"
	"github.com/SecDuckOps/Shared/llm/infrastructure"
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

// Get returns the registered LLM provider by name, with O(1) fallback capability.
func (r *RegistryAdapter) Get(name string) domain.LLM {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Direct match
	if provider, exists := r.llms[name]; exists {
		return provider
	}

	// 2. Fallback to default avoiding double locking
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
		if cfg.APIKey == "" && name != "lmstudio" {
			continue // Skip if no API key provided (except for local LMStudio)
		}

		switch name {
		case "openai":
			r.Register(infrastructure.NewOpenAIAdapter(cfg.APIKey, cfg.Model))
		case "openrouter":
			r.Register(infrastructure.NewOpenRouterAdapter(cfg.APIKey, cfg.Model))
		case "lmstudio":
			r.Register(infrastructure.NewLMStudioAdapter(cfg.APIKey, cfg.Model, cfg.BaseURL))
		case "gemini":
			// Gemini is handled separately in InitApp/App due to context requirement
			continue
		default:
			// Treat everything else with a BaseURL as a custom compatible provider
			if cfg.BaseURL != "" {
				r.Register(infrastructure.NewOpenAICompatibleAdapter(name, cfg.APIKey, cfg.Model, cfg.BaseURL))
			}
		}
	}
}
