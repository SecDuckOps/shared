package providerconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathPrecedence(t *testing.T) {
	t.Setenv(ConfigFileEnv, "")

	originalProductionPath := productionConfigPath
	productionConfigPath = filepath.Join(t.TempDir(), "etc-config.toml")
	t.Cleanup(func() {
		productionConfigPath = originalProductionPath
	})

	fallback := filepath.Join(t.TempDir(), "home-config.toml")
	if got := ResolveConfigPath("/tmp/explicit.toml", fallback); got != "/tmp/explicit.toml" {
		t.Fatalf("expected explicit path, got %q", got)
	}

	t.Setenv(ConfigFileEnv, "/tmp/from-env.toml")
	if got := ResolveConfigPath("", fallback); got != "/tmp/from-env.toml" {
		t.Fatalf("expected env path, got %q", got)
	}

	t.Setenv(ConfigFileEnv, "")
	if err := os.WriteFile(productionConfigPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write production config: %v", err)
	}
	if got := ResolveConfigPath("", fallback); got != productionConfigPath {
		t.Fatalf("expected production path, got %q", got)
	}

	if err := os.Remove(productionConfigPath); err != nil {
		t.Fatalf("remove production config: %v", err)
	}
	if got := ResolveConfigPath("", fallback); got != fallback {
		t.Fatalf("expected fallback path, got %q", got)
	}
}

func TestResolveActiveProviderUsesLegacyAliasWarning(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("LLM_API_KEY", "legacy-openrouter-key")

	manifest := &Manifest{
		Settings: Settings{DefaultProfile: "production"},
		Profiles: map[string]Profile{
			"production": {
				Provider: "openrouter",
				Model:    "openrouter/auto",
				Providers: map[string]Provider{
					"openrouter": {
						Type:    "custom",
						BaseURL: "https://openrouter.ai/api/v1",
						Auth:    &ProviderAuth{Type: "env", Key: "OPENROUTER_API_KEY"},
					},
				},
			},
		},
	}

	resolved, err := ResolveActiveProvider(manifest)
	if err != nil {
		t.Fatalf("ResolveActiveProvider() failed: %v", err)
	}
	if resolved.APIKey != "legacy-openrouter-key" {
		t.Fatalf("expected legacy alias API key, got %q", resolved.APIKey)
	}
	if len(resolved.Warnings) == 0 || !strings.Contains(resolved.Warnings[0], "legacy auth reference") {
		t.Fatalf("expected legacy warning, got %#v", resolved.Warnings)
	}
}

func TestResolveCredentialsSupportsGeminiLegacyGoogleEnv(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "google-key")

	resolved, err := ResolveCredentials("gemini", Provider{
		Auth: &ProviderAuth{Type: "env", Key: "GEMINI_API_KEY"},
	})
	if err != nil {
		t.Fatalf("ResolveCredentials() failed: %v", err)
	}
	if resolved.APIKey != "google-key" {
		t.Fatalf("expected GOOGLE_API_KEY fallback, got %q", resolved.APIKey)
	}
	if len(resolved.Warnings) == 0 {
		t.Fatalf("expected warning for GOOGLE_API_KEY fallback")
	}
}
