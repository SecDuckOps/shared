package gosec

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGosecParser_Parse(t *testing.T) {
	parser := NewGosecParser()

	assert.Equal(t, "gosec", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid gosec json", func(t *testing.T) {
		raw := []byte(`{
			"Issues": [
				{
					"severity": "HIGH",
					"confidence": "HIGH",
					"rule_id": "G104",
					"details": "Errors unhandled.",
					"file": "main.go",
					"code": "func main() { \n fmt.Println(\"test\") \n }",
					"line": "15"
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "G104", f.ID)
		assert.Equal(t, "G104: Errors unhandled.", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeSAST, f.Type)
		assert.Equal(t, "main.go", f.File)
		assert.Equal(t, 15, f.Line)
		assert.Equal(t, "gosec", f.Scanner)
		assert.Equal(t, "func main() { \n fmt.Println(\"test\") \n }", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse gosec json")
	})
}
