package trufflehog

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrufflehogParser_Parse(t *testing.T) {
	parser := NewTrufflehogParser()

	assert.Equal(t, "trufflehog", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid trufflehog json lines", func(t *testing.T) {
		raw := []byte(`{"SourceMetadata":{"Data":{"Filesystem":{"file":"config.yaml","line":15}}},"DetectorName":"AWS","Raw":"AKIAIOSFODNN7EXAMPLE","Redacted":"AKIAIOSFODN*********"}
{"SourceMetadata":{"Data":{"Git":{"file":"src/main.js","line":42}}},"DetectorName":"Slack","Raw":"xoxb-123456789","Redacted":"xoxb-123********"}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 2)

		f1 := findings[0]
		assert.Equal(t, "AWS", f1.ID)
		assert.Equal(t, "AWS Secret Discovered", f1.Title)
		assert.Equal(t, domain.SeverityCritical, f1.Severity)
		assert.Equal(t, domain.ScannerTypeSecrets, f1.Type)
		assert.Equal(t, "config.yaml", f1.File)
		assert.Equal(t, 15, f1.Line)
		assert.Equal(t, "AKIAIOSFODN*********", f1.Match)

		f2 := findings[1]
		assert.Equal(t, "Slack", f2.ID)
		assert.Equal(t, "src/main.js", f2.File)
		assert.Equal(t, 42, f2.Line)
	})

	t.Run("ignore non json parts", func(t *testing.T) {
		raw := []byte(`Starting scan...
{"SourceMetadata":{"Data":{"Filesystem":{"file":"config.yaml","line":15}}},"DetectorName":"AWS","Raw":"AKIAIOSFODNN7EXAMPLE","Redacted":"AKIAIOSFODN*********"}
Scan finished.`)
		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 1)
	})
}
