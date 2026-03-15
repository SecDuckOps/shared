package njsscan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// NjsscanParser parses JSON output from njsscan (NodeJS SAST)
type NjsscanParser struct{}

func NewNjsscanParser() ports.ResultParserPort {
	return &NjsscanParser{}
}

func (p *NjsscanParser) ScannerName() string {
	return "njsscan"
}

func (p *NjsscanParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *NjsscanParser) GetScanCommand(target string) []string {
	// njsscan scan command with JSON output
	return []string{"--json", "."}
}

type njsscanOutput struct {
	NodeJS map[string]struct {
		Files []struct {
			FilePath    string `json:"file_path"`
			MatchLines  []int  `json:"match_lines"`
			MatchString string `json:"match_string"`
		} `json:"files"`
		Metadata struct {
			Description string `json:"description"`
			Severity    string `json:"severity"` // ERROR, WARNING
		} `json:"metadata"`
	} `json:"nodejs"`
}

func (p *NjsscanParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed njsscanOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse njsscan json: %w", err)
	}

	var findings []domain.Finding
	for ruleID, ruleDetails := range parsed.NodeJS {
		mappedSeverity := domain.SeverityHigh
		if ruleDetails.Metadata.Severity == "ERROR" {
			mappedSeverity = domain.SeverityCritical
		} else if ruleDetails.Metadata.Severity == "WARNING" {
			mappedSeverity = domain.SeverityHigh
		} else {
			mappedSeverity = domain.NormalizeSeverity(ruleDetails.Metadata.Severity)
		}

		for _, fileFound := range ruleDetails.Files {
			startLine := 0
			if len(fileFound.MatchLines) > 0 {
				startLine = fileFound.MatchLines[0]
			}

			finding := domain.Finding{
				ID:          ruleID,
				Title:       ruleDetails.Metadata.Description,
				Description: ruleDetails.Metadata.Description,
				Severity:    mappedSeverity,
				Scanner:     p.ScannerName(),
				Type:        domain.ScannerTypeSAST,
				File:        fileFound.FilePath,
				Line:        startLine,
				Match:       strings.TrimSpace(fileFound.MatchString),
			}
			if finding.Title == "" {
				finding.Title = ruleID
			}

			findings = append(findings, finding)
		}
	}

	return findings, nil
}
