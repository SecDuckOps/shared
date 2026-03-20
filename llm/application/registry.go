package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	
	// Automatically wrap all registered LLMs with our token logger
	wrapped := NewLoggingLLM(llm)
	r.llms[llm.Name()] = wrapped
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

// ── Token / Interaction Logger Middleware ────────────────────────────────────

type LoggingLLM struct {
	inner   domain.LLM
	logFile string
}

func NewLoggingLLM(inner domain.LLM) domain.LLM {
	if inner == nil {
		return nil
	}
	home, _ := os.UserHomeDir()
	duckopsDir := filepath.Join(home, ".duckops")
	os.MkdirAll(duckopsDir, 0755)
	return &LoggingLLM{
		inner:   inner,
		logFile: filepath.Join(duckopsDir, "llm_tokens.log"),
	}
}

func (l *LoggingLLM) logTransaction(messages []domain.Message, output interface{}) {
	f, err := os.OpenFile(l.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString("========== " + time.Now().Format(time.RFC3339) + " [" + l.inner.Model() + "] ==========\n")
		f.WriteString("INPUT (PROMPT):\n")
		inBytes, _ := json.MarshalIndent(messages, "", "  ")
		f.Write(inBytes)
		f.WriteString("\n\nOUTPUT (COMPLETION):\n")
		if strOut, ok := output.(string); ok {
			f.WriteString(strOut)
		} else {
			outBytes, _ := json.MarshalIndent(output, "", "  ")
			f.Write(outBytes)
		}
		f.WriteString("\n=======================================================\n\n")
	}
}

func (l *LoggingLLM) Name() string { return l.inner.Name() }
func (l *LoggingLLM) Model() string { return l.inner.Model() }
func (l *LoggingLLM) HealthCheck(ctx context.Context) error { return l.inner.HealthCheck(ctx) }

func (l *LoggingLLM) Generate(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions) (domain.GenerationResult, error) {
	res, err := l.inner.Generate(ctx, messages, opts)
	if err != nil {
		l.logTransaction(messages, "ERROR: "+err.Error())
	} else {
		l.logTransaction(messages, res.Content)
	}
	return res, err
}

func (l *LoggingLLM) Stream(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions) (<-chan domain.ChatChunk, error) {
	ch, err := l.inner.Stream(ctx, messages, opts)
	if err != nil {
		l.logTransaction(messages, "STREAM ERROR: "+err.Error())
		return ch, err
	}
	
	outCh := make(chan domain.ChatChunk)
	go func() {
		defer close(outCh)
		var fullOutput string
		for chunk := range ch {
			fullOutput += chunk.Content
			outCh <- chunk
		}
		l.logTransaction(messages, fullOutput)
	}()
	return outCh, nil
}

func (l *LoggingLLM) GenerateJSON(ctx context.Context, messages []domain.Message, opts *domain.GenerateOptions, target interface{}) error {
	err := l.inner.GenerateJSON(ctx, messages, opts, target)
	if err != nil {
		l.logTransaction(messages, "JSON GEN ERROR: "+err.Error())
	} else {
		l.logTransaction(messages, target)
	}
	return err
}

