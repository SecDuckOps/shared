package tfsec

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// TfsecParser parses JSON output from Tfsec (IaC scanner)
type TfsecParser struct{}

func NewTfsecParser() ports.ResultParserPort {
	return &TfsecParser{}
}

func (p *TfsecParser) ScannerName() string {
	return "tfsec"
}

func (p *TfsecParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeIaC
}

func (p *TfsecParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *TfsecParser) GetScanCommand(target string) []string {
	// TFSec scan command with JSON output
	return []string{".", "--format", "json", "-q"}
}

type tfsecOutput struct {
	Results []struct {
		RuleID      string `json:"rule_id"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
		Location    struct {
			Filename  string `json:"filename"`
			StartLine int    `json:"start_line"`
		} `json:"location"`
	} `json:"results"`
}

func (p *TfsecParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed tfsecOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse tfsec json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed.Results {
		finding := domain.Finding{
			ID:          res.RuleID,
			Title:       res.RuleID,
			Description: res.Description,
			Severity:    domain.NormalizeSeverity(res.Severity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeIaC,
			File:        res.Location.Filename,
			Line:        res.Location.StartLine,
		}
		findings = append(findings, finding)
	}

	return findings, nil
}
