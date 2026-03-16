package tflint

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// TflintParser parses JSON output from TFLint (IaC)
type TflintParser struct{}

func NewTflintParser() ports.ResultParserPort {
	return &TflintParser{}
}

func (p *TflintParser) ScannerName() string {
	return "tflint"
}

func (p *TflintParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *TflintParser) GetScanCommand(target string) []string {
	// TFLint scan command with JSON output
	return []string{"--format", "json"}
}

type tflintOutput struct {
	Issues []struct {
		Rule struct {
			Name     string `json:"name"`
			Severity string `json:"severity"` // error, warning
		} `json:"rule"`
		Message string `json:"message"`
		Range   struct {
			Filename string `json:"filename"`
			Start    struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"range"`
	} `json:"issues"`
}

func (p *TflintParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed tflintOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse tflint json: %w", err)
	}

	var findings []domain.Finding
	for _, issue := range parsed.Issues {
		sev := domain.SeverityMedium
		if issue.Rule.Severity == "error" {
			sev = domain.SeverityHigh
		} else if issue.Rule.Severity == "warning" {
			sev = domain.SeverityMedium
		}

		finding := domain.Finding{
			ID:          issue.Rule.Name,
			Title:       issue.Rule.Name,
			Description: issue.Message,
			Severity:    sev,
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeIaC,
			File:        issue.Range.Filename,
			Line:        issue.Range.Start.Line,
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
