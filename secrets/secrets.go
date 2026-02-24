package secrets

import (
	"os"
	"strings"
)

var defaultSecretsPath = "/run/secrets/"

// GetSecret resolves a secret value using a priority chain:
//  1. Environment variable (envKey)
//  2. Docker secret file (/run/secrets/{secretName})
//  3. Fallback value
//
// This allows seamless operation in both local dev (env vars / .env)
// and Docker production (mounted secret files).
func GetSecret(envKey, secretName, fallback string) string {
	// Priority 1: Environment variable
	if val, ok := os.LookupEnv(envKey); ok && val != "" {
		return val
	}

	// Priority 2: Docker secret file
	if secretName != "" {
		if val, err := readSecretFile(secretName); err == nil {
			return val
		}
	}

	// Priority 3: Fallback
	return fallback
}

// MustGetSecret resolves a secret and panics if no value is found.
// Use this for required credentials that must be present at startup.
func MustGetSecret(envKey, secretName string) string {
	val := GetSecret(envKey, secretName, "")
	if val == "" {
		panic("required secret not found: env=" + envKey + " secret=" + secretName)
	}
	return val
}

// readSecretFile reads a Docker secret from /run/secrets/{name}.
func readSecretFile(name string) (string, error) {
	path := defaultSecretsPath + name
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
