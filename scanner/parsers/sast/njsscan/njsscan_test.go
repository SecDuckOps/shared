package njsscan

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNjsscanParser_Parse(t *testing.T) {
	parser := NewNjsscanParser()

	assert.Equal(t, "njsscan", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid njsscan json", func(t *testing.T) {
		raw := []byte(`{
			"nodejs": {
				"express_xss": {
					"files": [
						{
							"file_path": "/app/routes.js",
							"match_lines": [10, 11],
							"match_string": "  res.send(req.query.user)\n"
						}
					],
					"metadata": {
						"description": "Untrusted User Input in Response (XSS)",
						"severity": "ERROR"
					}
				}
			}
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "express_xss", f.ID)
		assert.Equal(t, "Untrusted User Input in Response (XSS)", f.Title)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeSAST, f.Type)
		assert.Equal(t, "/app/routes.js", f.File)
		assert.Equal(t, 10, f.Line)
		assert.Equal(t, "njsscan", f.Scanner)
		assert.Equal(t, "res.send(req.query.user)", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse njsscan json")
	})
}
