package terrascan

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerrascanParser_Parse(t *testing.T) {
	parser := NewTerrascanParser()

	assert.Equal(t, "terrascan", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid terrascan json", func(t *testing.T) {
		raw := []byte(`{
			"results": {
				"violations": [
					{
						"rule_name": "s3EnforceUserACL",
						"description": "S3 bucket Access is allowed to all AWS Account Users.",
						"rule_id": "AC_AWS_0497",
						"severity": "HIGH",
						"category": "S3",
						"file": "main.tf",
						"line": 10
					}
				]
			}
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "AC_AWS_0497", f.ID)
		assert.Equal(t, "s3EnforceUserACL", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f.Type)
		assert.Equal(t, "main.tf", f.File)
		assert.Equal(t, 10, f.Line)
		assert.Equal(t, "terrascan", f.Scanner)
		assert.Equal(t, "Category: S3", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse terrascan json")
	})
}
