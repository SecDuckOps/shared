package kics

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// KicsParser parses JSON output from Checkmarx KICS (IaC)
type KicsParser struct{}

func NewKicsParser() ports.ResultParserPort {
	return &KicsParser{}
}

func (p *KicsParser) ScannerName() string {
	return "kics"
}

func (p *KicsParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeIaC
}

func (p *KicsParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *KicsParser) GetScanCommand(target string) []string {
	// KICS scan command with JSON output
	return []string{"scan", "-p", ".", "--report-formats", "json", "--output-path", "/dev/stdout", "--output-name", "kics-result"}
}

type kicsOutput struct {
	Queries []struct {
		QueryName   string `json:"query_name"`
		QueryID     string `json:"query_id"`
		Severity    string `json:"severity"` // HIGH, MEDIUM, LOW, INFO
		Description string `json:"description"`
		Files       []struct {
			FileName      string `json:"file_name"`
			Line          int    `json:"line"`
			IssueType     string `json:"issue_type"`
			ExpectedValue string `json:"expected_value"`
			ActualValue   string `json:"actual_value"`
		} `json:"files"`
	} `json:"queries"`
}

func (p *KicsParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed kicsOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse kics json: %w", err)
	}

	var findings []domain.Finding
	for _, q := range parsed.Queries {
		for _, f := range q.Files {
			finding := domain.Finding{
				ID:          q.QueryID,
				Title:       q.QueryName,
				Description: q.Description,
				Severity:    domain.NormalizeSeverity(q.Severity),
				Scanner:     p.ScannerName(),
				Type:        domain.ScannerTypeIaC,
				File:        f.FileName,
				Line:        f.Line,
				Match:       fmt.Sprintf("Expected: %s | Actual: %s", f.ExpectedValue, f.ActualValue),
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}
