package trivy

import (
	"testing"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrivyParser_Parse(t *testing.T) {
	parser := NewTrivyParser()

	assert.Equal(t, "trivy", parser.ScannerName())
	assert.Contains(t, parser.SupportedFormats(), "json")

	t.Run("empty input", func(t *testing.T) {
		findings, err := parser.Parse([]byte{})
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("valid trivy json vulnerabilities and misconfigs", func(t *testing.T) {
		raw := []byte(`{
			"Results": [
				{
					"Target": "alpine:3.10",
					"Class": "os-pkgs",
					"Type": "alpine",
					"Vulnerabilities": [
						{
							"VulnerabilityID": "CVE-2019-5021",
							"PkgName": "alpine-baselayout",
							"InstalledVersion": "3.1.2-r0",
							"FixedVersion": "3.1.2-r1",
							"Title": "alpine-baselayout: missing password for root user",
							"Description": "In Alpine Linux before 3.10... ",
							"Severity": "CRITICAL"
						}
					]
				},
				{
					"Target": "Dockerfile",
					"Class": "config",
					"Misconfigurations": [
						{
							"ID": "DS002",
							"Title": "Image user should not be 'root'",
							"Description": "Running containers with 'root' user can lead to a container escape situation.",
							"Message": "Specify at least 1 USER command in Dockerfile with non-root user as argument",
							"Severity": "HIGH"
						}
					]
				}
			]
		}`)

		findings, err := parser.Parse(raw)
		require.NoError(t, err)
		require.Len(t, findings, 2)

		f1 := findings[0]
		assert.Equal(t, "CVE-2019-5021", f1.ID)
		assert.Equal(t, "alpine-baselayout: missing password for root user", f1.Title)
		assert.Equal(t, domain.SeverityCritical, f1.Severity)
		assert.Equal(t, domain.ScannerTypeContainer, f1.Type)
		assert.Equal(t, "alpine:3.10", f1.File)
		assert.Equal(t, "Package: alpine-baselayout, Installed: 3.1.2-r0, Fixed: 3.1.2-r1", f1.Match)
		assert.Equal(t, "CVE-2019-5021", f1.CVE)

		f2 := findings[1]
		assert.Equal(t, "DS002", f2.ID)
		assert.Equal(t, "Image user should not be 'root'", f2.Title)
		assert.Equal(t, domain.SeverityHigh, f2.Severity)
		assert.Equal(t, domain.ScannerTypeIaC, f2.Type)
		assert.Equal(t, "Dockerfile", f2.File)
		assert.Equal(t, "Specify at least 1 USER command in Dockerfile with non-root user as argument", f2.Match)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parser.Parse([]byte(`{ "invalid": }`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse trivy json")
	})
}
