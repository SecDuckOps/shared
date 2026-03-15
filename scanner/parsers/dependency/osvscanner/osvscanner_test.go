package osvscanner

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsvScannerParser_Parse(t *testing.T) {
	parser := NewOsvScannerParser()

	assert.Equal(t, "osvscanner", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid osv-scanner json", func(t *testing.T) {
		raw := []byte(`{
			"results": [
				{
					"source": {
						"type": "lockfile",
						"path": "go.mod"
					},
					"packages": [
						{
							"package": {
								"name": "github.com/gin-gonic/gin",
								"version": "1.6.3"
							},
							"vulnerabilities": [
								{
									"id": "GHSA-h395-qjr6-8jq8",
									"aliases": [
										"CVE-2020-28483"
									],
									"details": "Gin-gonic vulnerability"
								}
							]
						}
					]
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "GHSA-h395-qjr6-8jq8", f.ID)
		assert.Equal(t, "GHSA-h395-qjr6-8jq8 in github.com/gin-gonic/gin", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeDependency, f.Type)
		assert.Equal(t, "go.mod", f.File)
		assert.Equal(t, 0, f.Line)
		assert.Equal(t, "osvscanner", f.Scanner)
		assert.Equal(t, "Package: github.com/gin-gonic/gin, Version: 1.6.3", f.Match)
		assert.Equal(t, "CVE-2020-28483", f.CVE)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse osv-scanner json")
	})
}
