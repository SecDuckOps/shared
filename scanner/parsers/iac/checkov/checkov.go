package checkov

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// CheckovParser parses JSON output from Checkov
type CheckovParser struct{}

func NewCheckovParser() ports.ResultParserPort {
	return &CheckovParser{}
}

func (p *CheckovParser) ScannerName() string {
	return "checkov"
}

func (p *CheckovParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeIaC
}

func (p *CheckovParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *CheckovParser) GetScanCommand(target string) []string {
	// Checkov scan command with JSON output
	return []string{"-d", ".", "-o", "json", "--quiet", "--no-guide"}
}

type checkovOutput struct {
	Results struct {
		FailedChecks []struct {
			CheckID       string `json:"check_id"`
			CheckName     string `json:"check_name"`
			FilePath      string `json:"file_path"`
			FileLineRange []int  `json:"file_line_range"`
			Resource      string `json:"resource"`
			Severity      string `json:"severity"` // Checkov severity can be HIGH, LOW etc or absent depending on policies
		} `json:"failed_checks"`
	} `json:"results"`
}

func (p *CheckovParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed checkovOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Output can sometimes be an array of outputs if scanning multiple path types
		// A full parser would handle either obj or array of obj. For this scope we handle obj.
		var parsedArr []checkovOutput
		if errArr := json.Unmarshal(raw, &parsedArr); errArr == nil && len(parsedArr) > 0 {
			parsed = parsedArr[0] // Taking the first one for simplicity, real app would merge them
		} else {
			return nil, fmt.Errorf("failed to parse checkov json: %w", err)
		}
	}

	var findings []domain.Finding
	for _, check := range parsed.Results.FailedChecks {
		line := 0
		if len(check.FileLineRange) > 0 {
			line = check.FileLineRange[0]
		}

		sev := check.Severity
		if sev == "" {
			sev = "HIGH" // Default assumption for failed IaC policy if no severity given
		}

		finding := domain.Finding{
			ID:          check.CheckID,
			Title:       check.CheckName,
			Description: fmt.Sprintf("Resource: %s\n%s", check.Resource, check.CheckName),
			Severity:    domain.NormalizeSeverity(sev),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeIaC,
			File:        check.FilePath,
			Line:        line,
			Match:       check.Resource,
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
