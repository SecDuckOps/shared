package brakeman

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrakemanParser_Parse(t *testing.T) {
	parser := NewBrakemanParser()

	assert.Equal(t, "brakeman", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid brakeman json", func(t *testing.T) {
		raw := []byte(`{
			"warnings": [
				{
					"warning_type": "Command Injection",
					"check_name": "Execute",
					"message": "Possible command injection",
					"file": "app/controllers/users_controller.rb",
					"line": 10,
					"code": "system(params[:id])",
					"confidence": "High"
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "Execute", f.ID)
		assert.Equal(t, "Command Injection", f.Title)
		assert.Equal(t, domain.SeverityHigh, f.Severity)
		assert.Equal(t, domain.ScannerTypeSAST, f.Type)
		assert.Equal(t, "app/controllers/users_controller.rb", f.File)
		assert.Equal(t, 10, f.Line)
		assert.Equal(t, "brakeman", f.Scanner)
		assert.Equal(t, "system(params[:id])", f.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse brakeman json")
	})
}
