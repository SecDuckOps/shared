package brakeman

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// BrakemanParser parses JSON output from brakeman (Ruby on Rails SAST)
type BrakemanParser struct{}

func NewBrakemanParser() ports.ResultParserPort {
	return &BrakemanParser{}
}

func (p *BrakemanParser) ScannerName() string {
	return "brakeman"
}

func (p *BrakemanParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeSAST
}

func (p *BrakemanParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *BrakemanParser) GetScanCommand(target string) []string {
	// Brakeman scan command with JSON output
	return []string{"-f", "json", "-p", "."}
}

type brakemanOutput struct {
	Warnings []struct {
		WarningType string `json:"warning_type"`
		CheckName   string `json:"check_name"`
		Message     string `json:"message"`
		File        string `json:"file"`
		Line        int    `json:"line"`
		Code        string `json:"code"`
		Confidence  string `json:"confidence"` // High, Medium, Weak
	} `json:"warnings"`
	Errors []struct {
		Error string `json:"error"`
	} `json:"errors"`
}

func (p *BrakemanParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed brakemanOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse brakeman json: %w", err)
	}

	var findings []domain.Finding
	for _, warn := range parsed.Warnings {
		// Map Brakeman's "Confidence" to Severity (High confidence -> High severity usually)
		sev := domain.SeverityMedium
		if warn.Confidence == "High" {
			sev = domain.SeverityHigh
		} else if warn.Confidence == "Weak" {
			sev = domain.SeverityLow
		}

		finding := domain.Finding{
			ID:          warn.CheckName,
			Title:       warn.WarningType,
			Description: warn.Message,
			Severity:    sev,
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSAST,
			File:        warn.File,
			Line:        warn.Line,
			Match:       strings.TrimSpace(warn.Code),
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
