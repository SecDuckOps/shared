package gosec

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// GosecParser parses JSON output from gosec (Go SAST)
type GosecParser struct{}

func NewGosecParser() ports.ResultParserPort {
	return &GosecParser{}
}

func (p *GosecParser) ScannerName() string {
	return "gosec"
}

func (p *GosecParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeSAST
}

func (p *GosecParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *GosecParser) GetScanCommand(target string) []string {
	// Gosec scan command with JSON output
	return []string{"-fmt", "json", "-out", "/dev/stdout", "./..."}
}

type gosecOutput struct {
	Issues []struct {
		Severity   string `json:"severity"`
		Confidence string `json:"confidence"`
		RuleID     string `json:"rule_id"`
		Details    string `json:"details"`
		File       string `json:"file"`
		Code       string `json:"code"`
		Line       string `json:"line"`
	} `json:"Issues"`
}

func (p *GosecParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed gosecOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse gosec json: %w", err)
	}

	var findings []domain.Finding
	for _, issue := range parsed.Issues {
		// Convert line which might be a range "15-18" or "15"
		linePart := strings.Split(issue.Line, "-")[0]
		lineNum, _ := strconv.Atoi(linePart)

		finding := domain.Finding{
			ID:          issue.RuleID,
			Title:       fmt.Sprintf("%s: %s", issue.RuleID, issue.Details),
			Description: issue.Details,
			Severity:    domain.NormalizeSeverity(issue.Severity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSAST,
			File:        issue.File,
			Line:        lineNum,
			Match:       strings.TrimSpace(issue.Code),
		}
		findings = append(findings, finding)
	}

	return findings, nil
}
