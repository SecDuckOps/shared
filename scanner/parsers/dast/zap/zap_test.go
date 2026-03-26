package zap

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZapParser_Parse(t *testing.T) {
	parser := NewZapParser()

	assert.Equal(t, "zap", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid zap json", func(t *testing.T) {
		raw := []byte(`{
			"site": [
				{
					"@name": "http://localhost:8080",
					"alerts": [
						{
							"pluginid": "10021",
							"alert": "X-Content-Type-Options Header Missing",
							"riskdesc": "Low (Medium)",
							"desc": "The Anti-MIME-Sniffing header X-Content-Type-Options was not set to 'nosniff'.",
							"instances": [
								{
									"uri": "http://localhost:8080/login",
									"method": "GET",
									"evidence": ""
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
		assert.Equal(t, "10021", f.ID)
		assert.Equal(t, "X-Content-Type-Options Header Missing", f.Title)
		assert.Equal(t, domain.SeverityLow, f.Severity)
		assert.Equal(t, domain.ScannerTypeDAST, f.Type)
		assert.Equal(t, "http://localhost:8080/login", f.File)
		assert.Equal(t, "GET http://localhost:8080/login (Evidence: )", f.Match)
		assert.Equal(t, "zap", f.Scanner)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse zap json")
	})
}
