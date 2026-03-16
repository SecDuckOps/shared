package detectsecrets

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSecretsParser_Parse(t *testing.T) {
	parser := NewDetectSecretsParser()

	assert.Equal(t, "detectsecrets", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid detect-secrets json", func(t *testing.T) {
		raw := []byte(`{
			"results": {
				"app.py": [
					{
						"type": "Basic Auth Credentials",
						"line_number": 42,
						"hashed_secret": "abcdef123456"
					}
				]
			}
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "Basic Auth Credentials", f.ID)
		assert.Equal(t, "Secret: Basic Auth Credentials", f.Title)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeSecrets, f.Type)
		assert.Equal(t, "app.py", f.File)
		assert.Equal(t, 42, f.Line)
		assert.Equal(t, "detectsecrets", f.Scanner)
		assert.Equal(t, "Hash: abcdef123456", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse detect-secrets json")
	})
}
