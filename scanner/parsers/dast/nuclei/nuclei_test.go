package nuclei

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNucleiParser_Parse(t *testing.T) {
	parser := NewNucleiParser()

	assert.Equal(t, "nuclei", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid nuclei json fields", func(t *testing.T) {
		raw := []byte(`{"template-id":"cve-2021-44228","info":{"name":"Log4j JNDI RCE","severity":"critical","description":"Apache Log4j2 vulnerability"},"type":"http","host":"https://example.com","matched-at":"https://example.com/?q=...","extracted-results":["jndi:ldap"]}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)

		f := findings[0]
		assert.Equal(t, "cve-2021-44228", f.ID)
		assert.Equal(t, "Log4j JNDI RCE", f.Title)
		assert.Equal(t, domain.SeverityCritical, f.Severity)
		assert.Equal(t, domain.ScannerTypeDAST, f.Type)
		assert.Equal(t, "https://example.com", f.File)
		assert.Equal(t, "nuclei", f.Scanner)
		assert.Equal(t, "[http] https://example.com/?q=... | Extracted: jndi:ldap", f.Match)
		assert.Equal(t, "CVE-2021-44228", f.CVE)
	})

	t.Run("ignore non json parts", func(t *testing.T) {
		raw := []byte(`Starting scan...
{"template-id":"test","info":{"severity":"info"},"type":"dns","host":"example.com"}
Scan finished.`)
		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)
	})
}
