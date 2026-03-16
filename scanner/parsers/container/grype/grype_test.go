package grype

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrypeParser_Parse(t *testing.T) {
	parser := NewGrypeParser()

	assert.Equal(t, "grype", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid grype json", func(t *testing.T) {
		raw := []byte(`{
			"matches": [
				{
					"vulnerability": {
						"id": "CVE-2021-44228",
						"severity": "Critical",
						"description": "Log4j vulnerability"
					},
					"artifact": {
						"name": "log4j-core",
						"version": "2.14.1",
						"locations": [
							{
								"path": "/app/lib/log4j-core.jar"
							}
						]
					}
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "CVE-2021-44228", f.ID)
		assert.Equal(t, "CVE-2021-44228 in log4j-core", f.Title)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeContainer, f.Type)
		assert.Equal(t, "/app/lib/log4j-core.jar", f.File)
		assert.Equal(t, "grype", f.Scanner)
		assert.Equal(t, "Package: log4j-core, Version: 2.14.1", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse grype json")
	})
}
