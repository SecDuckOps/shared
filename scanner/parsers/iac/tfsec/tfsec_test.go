package tfsec

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTfsecParser_Parse(t *testing.T) {
	parser := NewTfsecParser()

	assert.Equal(t, "tfsec", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid tfsec json", func(t *testing.T) {
		raw := []byte(`{
			"results": [
				{
					"rule_id": "aws-s3-enable-bucket-encryption",
					"description": "Bucket does not have encryption enabled",
					"severity": "HIGH",
					"location": {
						"filename": "main.tf",
						"start_line": 15
					}
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "aws-s3-enable-bucket-encryption", f.ID)
		assert.Equal(t, "aws-s3-enable-bucket-encryption", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f.Type)
		assert.Equal(t, "main.tf", f.File)
		assert.Equal(t, 15, f.Line)
		assert.Equal(t, "tfsec", f.Scanner)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tfsec json")
	})
}
