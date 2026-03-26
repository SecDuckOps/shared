package dependencycheck

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyCheckParser_Parse(t *testing.T) {
	parser := NewDependencyCheckParser()

	assert.Equal(t, "dependencycheck", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid dependencycheck json", func(t *testing.T) {
		raw := []byte(`{
			"dependencies": [
				{
					"filePath": "/workspace/pom.xml",
					"vulnerabilities": [
						{
							"name": "CVE-2021-44228",
							"severity": "CRITICAL",
							"description": "Apache Log4j2 vulnerability"
						}
					]
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "CVE-2021-44228", f.ID)
		assert.Equal(t, "CVE-2021-44228", f.Title)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeDependency, f.Type)
		assert.Equal(t, "/workspace/pom.xml", f.File)
		assert.Equal(t, "dependencycheck", f.Scanner)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse dependencycheck json")
	})
}
