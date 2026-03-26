package bandit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// BanditParser parses JSON output from bandit (Python SAST)
type BanditParser struct{}

func NewBanditParser() ports.ResultParserPort {
	return &BanditParser{}
}

func (p *BanditParser) ScannerName() string {
	return "bandit"
}

func (p *BanditParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeSAST
}

func (p *BanditParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *BanditParser) GetScanCommand(target string) []string {
	// Bandit scan command with JSON output
	return []string{"-f", "json", "-r", "."}
}

type banditOutput struct {
	Results []struct {
		TestID        string `json:"test_id"`
		TestName      string `json:"test_name"`
		IssueSeverity string `json:"issue_severity"`
		IssueText     string `json:"issue_text"`
		Filename      string `json:"filename"`
		LineNumber    int    `json:"line_number"`
		Code          string `json:"code"`
	} `json:"results"`
}

func (p *BanditParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed banditOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse bandit json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed.Results {
		finding := domain.Finding{
			ID:          res.TestID,
			Title:       res.TestName,
			Description: res.IssueText,
			Severity:    domain.NormalizeSeverity(res.IssueSeverity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSAST,
			File:        res.Filename,
			Line:        res.LineNumber,
			Match:       strings.TrimSpace(res.Code),
		}

		if finding.Title == "" {
			finding.Title = res.TestID
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
