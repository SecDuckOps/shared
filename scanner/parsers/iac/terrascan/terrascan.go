package terrascan

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// TerrascanParser parses JSON output from Tenable Terrascan (IaC)
type TerrascanParser struct{}

func NewTerrascanParser() ports.ResultParserPort {
	return &TerrascanParser{}
}

func (p *TerrascanParser) ScannerName() string {
	return "terrascan"
}

func (p *TerrascanParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeIaC
}

func (p *TerrascanParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *TerrascanParser) GetScanCommand(target string) []string {
	// Terrascan scan command with JSON output
	return []string{"scan", "-p", ".", "-o", "json"}
}

type terrascanOutput struct {
	Results struct {
		Violations []struct {
			RuleName    string `json:"rule_name"`
			Description string `json:"description"`
			RuleID      string `json:"rule_id"`
			Severity    string `json:"severity"`
			Category    string `json:"category"`
			File        string `json:"file"`
			Line        int    `json:"line"`
		} `json:"violations"`
	} `json:"results"`
}

func (p *TerrascanParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed terrascanOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse terrascan json: %w", err)
	}

	var findings []domain.Finding
	for _, v := range parsed.Results.Violations {
		finding := domain.Finding{
			ID:          v.RuleID,
			Title:       v.RuleName,
			Description: v.Description,
			Severity:    domain.NormalizeSeverity(v.Severity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeIaC,
			File:        v.File,
			Line:        v.Line,
			Match:       fmt.Sprintf("Category: %s", v.Category),
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
