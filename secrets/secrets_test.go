package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSecret_EnvVarPriority(t *testing.T) {
	// Env var should take priority over everything
	os.Setenv("TEST_SECRET_KEY", "from-env")
	defer os.Unsetenv("TEST_SECRET_KEY")

	val := GetSecret("TEST_SECRET_KEY", "nonexistent", "fallback")
	if val != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", val)
	}
}

func TestGetSecret_FilePriority(t *testing.T) {
	// Create a temp secret file
	tmpDir := t.TempDir()
	origPath := defaultSecretsPath
	defaultSecretsPath = tmpDir + "/"
	defer func() { defaultSecretsPath = origPath }()

	secretFile := filepath.Join(tmpDir, "db_password")
	os.WriteFile(secretFile, []byte("from-file\n"), 0600)

	// No env var set — should read from file
	os.Unsetenv("TEST_DB_PASS")
	val := GetSecret("TEST_DB_PASS", "db_password", "fallback")
	if val != "from-file" {
		t.Errorf("expected 'from-file', got '%s'", val)
	}
}

func TestGetSecret_Fallback(t *testing.T) {
	os.Unsetenv("TEST_MISSING_KEY")

	val := GetSecret("TEST_MISSING_KEY", "nonexistent_file", "my-fallback")
	if val != "my-fallback" {
		t.Errorf("expected 'my-fallback', got '%s'", val)
	}
}

func TestGetSecret_EnvOverridesFile(t *testing.T) {
	// Create a secret file
	tmpDir := t.TempDir()
	origPath := defaultSecretsPath
	defaultSecretsPath = tmpDir + "/"
	defer func() { defaultSecretsPath = origPath }()

	secretFile := filepath.Join(tmpDir, "db_password")
	os.WriteFile(secretFile, []byte("from-file"), 0600)

	// Set env var — should take priority over file
	os.Setenv("TEST_DB_PASS", "from-env")
	defer os.Unsetenv("TEST_DB_PASS")

	val := GetSecret("TEST_DB_PASS", "db_password", "fallback")
	if val != "from-env" {
		t.Errorf("expected 'from-env', got '%s'", val)
	}
}

func TestMustGetSecret_Panic(t *testing.T) {
	os.Unsetenv("TEST_REQUIRED_MISSING")

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic, got none")
		}
	}()

	MustGetSecret("TEST_REQUIRED_MISSING", "nonexistent")
}

func TestMustGetSecret_Success(t *testing.T) {
	os.Setenv("TEST_REQUIRED_KEY", "found-it")
	defer os.Unsetenv("TEST_REQUIRED_KEY")

	val := MustGetSecret("TEST_REQUIRED_KEY", "some_secret")
	if val != "found-it" {
		t.Errorf("expected 'found-it', got '%s'", val)
	}
}

func TestGetSecret_TrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := defaultSecretsPath
	defaultSecretsPath = tmpDir + "/"
	defer func() { defaultSecretsPath = origPath }()

	secretFile := filepath.Join(tmpDir, "trimtest")
	os.WriteFile(secretFile, []byte("  secret-value  \n"), 0600)

	os.Unsetenv("TEST_TRIM_KEY")
	val := GetSecret("TEST_TRIM_KEY", "trimtest", "")
	if val != "secret-value" {
		t.Errorf("expected 'secret-value', got '%s'", val)
	}
}
