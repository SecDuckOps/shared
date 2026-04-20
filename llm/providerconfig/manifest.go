package providerconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const ConfigFileEnv = "DUCKOPS_CONFIG_FILE"

var productionConfigPath = "/etc/duckops/config.toml"
var dockerSecretsPath = "/run/secrets"

type Manifest struct {
	Settings Settings           `toml:"settings"`
	Profiles map[string]Profile `toml:"profiles"`
}

type Settings struct {
	DefaultProfile string `toml:"default_profile,omitempty"`
}

type Profile struct {
	Provider  string              `toml:"provider,omitempty"`
	Model     string              `toml:"model,omitempty"`
	Providers map[string]Provider `toml:"providers,omitempty"`
}

type Provider struct {
	Type    string        `toml:"type,omitempty"`
	APIKey  string        `toml:"api_key,omitempty"`
	Model   string        `toml:"model,omitempty"`
	BaseURL string        `toml:"base_url,omitempty"`
	Auth    *ProviderAuth `toml:"auth,omitempty"`
}

type ProviderAuth struct {
	Type string `toml:"type,omitempty"`
	Key  string `toml:"key,omitempty"`
}

type ResolvedProvider struct {
	ProfileName  string
	ProviderName string
	Model        string
	BaseURL      string
	APIKey       string
	AuthMode     string
	AuthRef      string
	Warnings     []string
}

type credentialCandidate struct {
	envKey     string
	secretName string
	warning    string
}

func ResolveConfigPath(explicitPath string, fallbackPath string) string {
	if path := strings.TrimSpace(explicitPath); path != "" {
		return path
	}
	if path := strings.TrimSpace(os.Getenv(ConfigFileEnv)); path != "" {
		return path
	}
	if _, err := os.Stat(productionConfigPath); err == nil {
		return productionConfigPath
	}
	return strings.TrimSpace(fallbackPath)
}

func LoadManifest(path string) (*Manifest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("config path is not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var manifest Manifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if manifest.Profiles == nil {
		manifest.Profiles = map[string]Profile{}
	}
	return &manifest, nil
}

func ResolveActiveProvider(manifest *Manifest) (ResolvedProvider, error) {
	if manifest == nil {
		return ResolvedProvider{}, fmt.Errorf("provider manifest is nil")
	}

	profileName := strings.TrimSpace(manifest.Settings.DefaultProfile)
	if profileName == "" {
		profileName = "default"
	}

	profile, ok := manifest.Profiles[profileName]
	if !ok {
		return ResolvedProvider{}, fmt.Errorf("default_profile %q does not exist", profileName)
	}

	resolved, err := ResolveProvider(profileName, profile.Provider, profile.Model, profile.Providers)
	if err != nil {
		return ResolvedProvider{}, err
	}
	return resolved, nil
}

func ResolveProvider(profileName string, providerName string, profileModel string, providers map[string]Provider) (ResolvedProvider, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "default"
	}

	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		return ResolvedProvider{}, fmt.Errorf("profile %q does not configure a provider", profileName)
	}

	providerCfg, ok := providers[providerName]
	if !ok {
		return ResolvedProvider{}, fmt.Errorf("profile %q references provider %q but no [profiles.%s.providers.%s] entry exists", profileName, providerName, profileName, providerName)
	}

	resolved := ResolvedProvider{
		ProfileName:  profileName,
		ProviderName: providerName,
		Model:        strings.TrimSpace(profileModel),
		BaseURL:      strings.TrimSpace(providerCfg.BaseURL),
	}
	if resolved.Model == "" {
		resolved.Model = strings.TrimSpace(providerCfg.Model)
	}
	if resolved.Model == "" {
		return ResolvedProvider{}, fmt.Errorf("profile %q provider %q does not configure a model", profileName, providerName)
	}

	creds, err := ResolveCredentials(providerName, providerCfg)
	if err != nil {
		return ResolvedProvider{}, err
	}
	resolved.APIKey = creds.APIKey
	resolved.AuthMode = creds.AuthMode
	resolved.AuthRef = creds.AuthRef
	resolved.Warnings = append(resolved.Warnings, creds.Warnings...)
	return resolved, nil
}

type CredentialResolution struct {
	APIKey   string
	AuthMode string
	AuthRef  string
	Warnings []string
}

func ResolveCredentials(providerName string, provider Provider) (CredentialResolution, error) {
	providerName = strings.TrimSpace(providerName)
	inlineKey := strings.TrimSpace(provider.APIKey)
	if inlineKey != "" {
		return CredentialResolution{
			APIKey:   inlineKey,
			AuthMode: "inline",
			AuthRef:  "stored",
		}, nil
	}

	if isLocalProvider(providerName) {
		return CredentialResolution{
			AuthMode: "none",
		}, nil
	}

	if provider.Auth == nil {
		return CredentialResolution{}, fmt.Errorf("provider %q requires auth.type=env or an inline api_key", providerName)
	}

	authType := strings.TrimSpace(strings.ToLower(provider.Auth.Type))
	authKey := strings.TrimSpace(provider.Auth.Key)
	switch authType {
	case "":
		return CredentialResolution{}, fmt.Errorf("provider %q auth.type is not configured", providerName)
	case "env":
		if authKey == "" {
			return CredentialResolution{}, fmt.Errorf("provider %q auth.type=env is configured without auth.key", providerName)
		}

		candidates := credentialCandidates(providerName, authKey)
		for i, candidate := range candidates {
			if candidate.envKey == "" {
				continue
			}
			value, authMode, authRef := resolveCandidateValue(candidate)
			if strings.TrimSpace(value) == "" {
				continue
			}

			resolution := CredentialResolution{
				APIKey:   strings.TrimSpace(value),
				AuthMode: authMode,
				AuthRef:  authRef,
			}
			if i > 0 && candidate.warning != "" {
				resolution.Warnings = append(resolution.Warnings, candidate.warning)
			}
			return resolution, nil
		}

		return CredentialResolution{}, fmt.Errorf("%s is not set", authKey)
	default:
		return CredentialResolution{}, fmt.Errorf("provider %q auth.type %q is not supported", providerName, authType)
	}
}

func credentialCandidates(providerName string, authKey string) []credentialCandidate {
	authKey = strings.TrimSpace(authKey)
	canonical := canonicalEnvKey(providerName)
	seen := map[string]struct{}{}
	candidates := make([]credentialCandidate, 0, 4)
	appendCandidate := func(envKey string, warning string) {
		envKey = strings.TrimSpace(envKey)
		if envKey == "" {
			return
		}
		if _, exists := seen[envKey]; exists {
			return
		}
		seen[envKey] = struct{}{}
		candidates = append(candidates, credentialCandidate{
			envKey:     envKey,
			secretName: envKeyToSecretName(envKey),
			warning:    warning,
		})
	}

	appendCandidate(authKey, "")
	appendCandidate(canonical, legacyCredentialWarning(providerName, canonical))
	for _, alias := range legacyEnvAliases(providerName) {
		appendCandidate(alias, legacyCredentialWarning(providerName, alias))
	}
	return candidates
}

func canonicalEnvKey(providerName string) string {
	switch normalizeProviderName(providerName) {
	case "openai":
		return "OPENAI_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "gemini", "google":
		return "GEMINI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	default:
		return ""
	}
}

func legacyEnvAliases(providerName string) []string {
	switch normalizeProviderName(providerName) {
	case "openrouter":
		return []string{"LLM_API_KEY"}
	case "gemini", "google":
		return []string{"GOOGLE_API_KEY"}
	default:
		return nil
	}
}

func legacyCredentialWarning(providerName string, envKey string) string {
	envKey = strings.TrimSpace(envKey)
	if envKey == "" {
		return ""
	}
	canonical := canonicalEnvKey(providerName)
	if envKey == canonical {
		return ""
	}
	return fmt.Sprintf("provider %q resolved credentials from legacy auth reference %s; migrate auth.key to %s", providerName, envKey, canonical)
}

func envKeyToSecretName(envKey string) string {
	envKey = strings.TrimSpace(strings.ToLower(envKey))
	envKey = strings.ReplaceAll(envKey, "-", "_")
	return filepath.Base(envKey)
}

func resolveCandidateValue(candidate credentialCandidate) (value string, authMode string, authRef string) {
	if candidate.envKey != "" {
		if envValue, ok := os.LookupEnv(candidate.envKey); ok && strings.TrimSpace(envValue) != "" {
			return strings.TrimSpace(envValue), "env", candidate.envKey
		}
	}
	if candidate.secretName != "" {
		secretPath := filepath.Join(dockerSecretsPath, candidate.secretName)
		if secretValue, err := os.ReadFile(secretPath); err == nil && strings.TrimSpace(string(secretValue)) != "" {
			return strings.TrimSpace(string(secretValue)), "secret", candidate.secretName
		}
	}
	return "", "", ""
}

func isLocalProvider(providerName string) bool {
	switch normalizeProviderName(providerName) {
	case "ollama", "lmstudio":
		return true
	default:
		return false
	}
}

func normalizeProviderName(providerName string) string {
	return strings.TrimSpace(strings.ToLower(providerName))
}
