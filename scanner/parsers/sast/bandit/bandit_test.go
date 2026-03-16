package bandit

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBanditParser_Parse(t *testing.T) {
	parser := NewBanditParser()

	assert.Equal(t, "bandit", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid bandit json", func(t *testing.T) {
		raw := []byte(`{
			"results": [
				{
					"test_id": "B101",
					"test_name": "assert_used",
					"issue_severity": "LOW",
					"issue_text": "Use of assert detected.",
					"filename": "test.py",
					"line_number": 10,
					"code": "  assert True\n"
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "B101", f.ID)
		assert.Equal(t, "assert_used", f.Title)
		assert.Equal(t, domain.SeverityLow, f.Severity)
		assert.Equal(t, domain.ScannerTypeSAST, f.Type)
		assert.Equal(t, "test.py", f.File)
		assert.Equal(t, 10, f.Line)
		assert.Equal(t, "bandit", f.Scanner)
		assert.Equal(t, "assert True", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse bandit json")
	})
}
