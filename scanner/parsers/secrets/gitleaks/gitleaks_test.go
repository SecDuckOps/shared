package gitleaks

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitleaksParser_Parse(t *testing.T) {
	parser := NewGitleaksParser()

	assert.Equal(t, "gitleaks", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid gitleaks json", func(t *testing.T) {
		raw := []byte(`[
			{
				"Description": "AWS Access Key",
				"StartLine": 12,
				"Match": "AKIAIOSFODNN7EXAMPLE",
				"Secret": "AKIAIOSFODNN7EXAMPLE",
				"File": "config/credentials.yml",
				"Commit": "a1b2c3d4",
				"RuleID": "aws-access-token"
			}
		]`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "aws-access-token", f.ID)
		assert.Equal(t, "AWS Access Key", f.Title)
		assert.Equal(t, "config/credentials.yml", f.File)
		assert.Equal(t, 12, f.Line)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeSecrets, f.Type)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", f.Match)
		assert.Equal(t, "gitleaks", f.Scanner)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "not": "array" }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse gitleaks json")
	})
}
