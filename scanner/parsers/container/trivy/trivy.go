package trivy

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// TrivyParser parses JSON output from Trivy
type TrivyParser struct{}

func NewTrivyParser() ports.ResultParserPort {
	return &TrivyParser{}
}

func (p *TrivyParser) ScannerName() string {
	return "trivy"
}

func (p *TrivyParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeContainer
}

func (p *TrivyParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *TrivyParser) GetScanCommand(target string) []string {
	// target in container is always /scan/workspace (mounted read-only)
	return []string{"filesystem", "--format", "json", "."}
}

type trivyOutput struct {
	Results []struct {
		Target          string `json:"Target"`
		Class           string `json:"Class"`
		Type            string `json:"Type"`
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Title            string `json:"Title"`
			Description      string `json:"Description"`
			Severity         string `json:"Severity"`
		} `json:"Vulnerabilities"`
		Misconfigurations []struct {
			ID          string `json:"ID"`
			Title       string `json:"Title"`
			Description string `json:"Description"`
			Message     string `json:"Message"`
			Severity    string `json:"Severity"`
		} `json:"Misconfigurations"`
	} `json:"Results"`
}

func (p *TrivyParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed trivyOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse trivy json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed.Results {
		// Determine scanner type based on class or type
		var scannerType domain.ScannerType
		if res.Class == "os-pkgs" || res.Class == "lang-pkgs" {
			scannerType = domain.ScannerTypeDependency
			if res.Type != "" { // e.g. "alpine", "npm" etc
				// Optional: mapping could be Container if it's OS packages, but Dependency is fine or Container
				if res.Class == "os-pkgs" {
					scannerType = domain.ScannerTypeContainer
				} else {
					scannerType = domain.ScannerTypeDependency
				}
			}
		} else if res.Class == "config" {
			scannerType = domain.ScannerTypeIaC
		} else {
			scannerType = domain.ScannerTypeContainer
		}

		for _, vuln := range res.Vulnerabilities {
			matchDesc := fmt.Sprintf("Package: %s, Installed: %s", vuln.PkgName, vuln.InstalledVersion)
			if vuln.FixedVersion != "" {
				matchDesc += fmt.Sprintf(", Fixed: %s", vuln.FixedVersion)
			}

			finding := domain.Finding{
				ID:          vuln.VulnerabilityID,
				Title:       vuln.Title,
				Description: vuln.Description,
				Severity:    domain.NormalizeSeverity(vuln.Severity),
				Scanner:     p.ScannerName(),
				Type:        scannerType,
				File:        res.Target,
				Match:       matchDesc,
				CVE:         vuln.VulnerabilityID,
			}
			if finding.Title == "" {
				finding.Title = vuln.VulnerabilityID
			}
			findings = append(findings, finding)
		}

		for _, misconf := range res.Misconfigurations {
			finding := domain.Finding{
				ID:          misconf.ID,
				Title:       misconf.Title,
				Description: misconf.Description,
				Severity:    domain.NormalizeSeverity(misconf.Severity),
				Scanner:     p.ScannerName(),
				Type:        domain.ScannerTypeIaC, // Or container depending on what it is
				File:        res.Target,
				Match:       misconf.Message,
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}
