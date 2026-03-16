package dependencycheck

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// DependencyCheckParser parses JSON output from OWASP Dependency-Check
type DependencyCheckParser struct{}

func NewDependencyCheckParser() ports.ResultParserPort {
	return &DependencyCheckParser{}
}

func (p *DependencyCheckParser) ScannerName() string {
	return "dependencycheck"
}

func (p *DependencyCheckParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *DependencyCheckParser) GetScanCommand(target string) []string {
	// Dependency-Check scan command with JSON output
	return []string{"--project", "DuckOps-Scan", "--scan", ".", "--format", "JSON", "--out", "/dev/stdout"}
}

type dcOutput struct {
	Dependencies []struct {
		FilePath        string `json:"filePath"`
		Vulnerabilities []struct {
			Name        string `json:"name"`
			Severity    string `json:"severity"`
			Description string `json:"description"`
		} `json:"vulnerabilities"`
	} `json:"dependencies"`
}

func (p *DependencyCheckParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed dcOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse dependencycheck json: %w", err)
	}

	var findings []domain.Finding
	for _, dep := range parsed.Dependencies {
		for _, vuln := range dep.Vulnerabilities {
			finding := domain.Finding{
				ID:          vuln.Name,
				Title:       vuln.Name,
				Description: vuln.Description,
				Severity:    domain.NormalizeSeverity(vuln.Severity),
				Scanner:     p.ScannerName(),
				Type:        domain.ScannerTypeDependency,
				File:        dep.FilePath,
				CVE:         vuln.Name,
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}
