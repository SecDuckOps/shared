package osvscanner

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// OsvScannerParser parses JSON output from Google OSV-Scanner (Dependency)
type OsvScannerParser struct{}

func NewOsvScannerParser() ports.ResultParserPort {
	return &OsvScannerParser{}
}

func (p *OsvScannerParser) ScannerName() string {
	return "osvscanner"
}

func (p *OsvScannerParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *OsvScannerParser) GetScanCommand(target string) []string {
	// OSV-Scanner scan command with JSON output
	return []string{"-r", "--format", "json", "."}
}

type osvOutput struct {
	Results []struct {
		Source struct {
			Type string `json:"type"`
			Path string `json:"path"`
		} `json:"source"`
		Packages []struct {
			Package struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"package"`
			Vulnerabilities []struct {
				ID      string   `json:"id"`
				Aliases []string `json:"aliases"`
				Details string   `json:"details"`
			} `json:"vulnerabilities"`
		} `json:"packages"`
	} `json:"results"`
}

func (p *OsvScannerParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed osvOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse osv-scanner json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed.Results {
		for _, pkg := range res.Packages {
			for _, vuln := range pkg.Vulnerabilities {

				finding := domain.Finding{
					ID:          vuln.ID,
					Title:       fmt.Sprintf("%s in %s", vuln.ID, pkg.Package.Name),
					Description: vuln.Details,
					Severity:    domain.SeverityHigh, // OSV-Scanner JSON doesn't always contain severity directly, assume HIGH for known CVEs
					Scanner:     p.ScannerName(),
					Type:        domain.ScannerTypeDependency,
					File:        res.Source.Path,
					Match:       fmt.Sprintf("Package: %s, Version: %s", pkg.Package.Name, pkg.Package.Version),
					CVE:         vuln.ID,
				}

				// If aliases contains a CVE, use that as the primary CVE field
				for _, alias := range vuln.Aliases {
					if len(alias) > 3 && alias[:3] == "CVE" {
						finding.CVE = alias
						break
					}
				}

				findings = append(findings, finding)
			}
		}
	}

	return findings, nil
}
