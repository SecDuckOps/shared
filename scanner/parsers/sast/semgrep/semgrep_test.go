package semgrep

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemgrepParser_Parse(t *testing.T) {
	parser := NewSemgrepParser()

	assert.Equal(t, "semgrep", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid semgrep json", func(t *testing.T) {
		raw := []byte(`{
			"results": [
				{
					"check_id": "go.lang.security.audit.xss.import-text-template.use-text-template",
					"path": "test/main.go",
					"start": { "line": 42 },
					"extra": {
						"message": "Use of text/template can lead to XSS. Use html/template instead.",
						"severity": "WARNING",
						"lines": "import \"text/template\"\n",
						"metadata": {
							"cwe": ["CWE-79: Improper Neutralization of Input During Web Page Generation ('Cross-site Scripting')"]
						}
					}
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "go.lang.security.audit.xss.import-text-template.use-text-template", f.ID)
		assert.Equal(t, "test/main.go", f.File)
		assert.Equal(t, 42, f.Line)
		assert.Equal(t, domain.SeverityHigh, f.Severity) // WARNING parses to High typically, or HIGH if normlized directly
		assert.Equal(t, "semgrep", f.Scanner)
		assert.Equal(t, domain.ScannerTypeSAST, f.Type)
		assert.Equal(t, "import \"text/template\"", f.Match)
		assert.Equal(t, "CWE-79: Improper Neutralization of Input During Web Page Generation ('Cross-site Scripting')", f.CVE)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": json }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse semgrep json")
	})
}
