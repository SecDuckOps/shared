package gitleaks

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// GitleaksParser parses JSON output from Gitleaks
type GitleaksParser struct{}

func NewGitleaksParser() ports.ResultParserPort {
	return &GitleaksParser{}
}

func (p *GitleaksParser) ScannerName() string {
	return "gitleaks"
}

func (p *GitleaksParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *GitleaksParser) GetScanCommand(target string) []string {
	// Gitleaks detect command with JSON report
	return []string{"detect", "--source", ".", "--report-path", "/dev/stdout", "--no-git", "--exit-code", "0"}
}

type gitleaksFinding struct {
	Description string `json:"Description"`
	StartLine   int    `json:"StartLine"`
	Match       string `json:"Match"`
	Secret      string `json:"Secret"`
	File        string `json:"File"`
	RuleID      string `json:"RuleID"`
}

func (p *GitleaksParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed []gitleaksFinding
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse gitleaks json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed {
		// We should redact or handle the secret carefully.
		// For the domain.Match, we can safely store the matched line but maybe mask the exact secret if needed.
		// However, typical platforms store it to show the user. We will store it in Match.
		
		finding := domain.Finding{
			ID:          res.RuleID,
			Title:       res.Description,
			Description: fmt.Sprintf("Hardcoded secret discovered: %s", res.Description),
			Severity:    domain.SeverityCritical, // Secrets are usually critical
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSecrets,
			File:        res.File,
			Line:        res.StartLine,
			Match:       res.Match,
		}
		
		findings = append(findings, finding)
	}

	return findings, nil
}
