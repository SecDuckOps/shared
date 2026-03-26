package tflint

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTflintParser_Parse(t *testing.T) {
	parser := NewTflintParser()

	assert.Equal(t, "tflint", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid tflint json", func(t *testing.T) {
		raw := []byte(`{
			"issues": [
				{
					"rule": {
						"name": "aws_instance_invalid_type",
						"severity": "error"
					},
					"message": "invalid instance type",
					"range": {
						"filename": "main.tf",
						"start": {
							"line": 10
						}
					}
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "aws_instance_invalid_type", f.ID)
		assert.Equal(t, "aws_instance_invalid_type", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f.Type)
		assert.Equal(t, "main.tf", f.File)
		assert.Equal(t, 10, f.Line)
		assert.Equal(t, "tflint", f.Scanner)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tflint json")
	})
}
