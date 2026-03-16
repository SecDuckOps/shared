package semgrep

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// SemgrepParser parses JSON output from Semgrep
type SemgrepParser struct{}

func NewSemgrepParser() ports.ResultParserPort {
	return &SemgrepParser{}
}

func (p *SemgrepParser) ScannerName() string {
	return "semgrep"
}

func (p *SemgrepParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *SemgrepParser) GetScanCommand(target string) []string {
	// Standard semgrep scan with JSON output
	return []string{"semgrep", "scan", "--json", "."}
}

type semgrepOutput struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
		} `json:"start"`
		Extra struct {
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Metadata struct {
				CVE []string `json:"cve"`
				CWE []string `json:"cwe"` // Semgrep often uses CWE rather than CVE for SAST
			} `json:"metadata"`
			Lines string `json:"lines"`
		} `json:"extra"`
	} `json:"results"`
}

func (p *SemgrepParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed semgrepOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep json: %w", err)
	}

	var findings []domain.Finding
	for _, res := range parsed.Results {
		severity := domain.NormalizeSeverity(res.Extra.Severity)
		if res.Extra.Severity == "ERROR" {
			severity = domain.SeverityCritical
		} else if res.Extra.Severity == "WARNING" {
			severity = domain.SeverityHigh // Or Medium depending on config, Semgrep uses ERROR/WARNING/INFO
		}

		cve := ""
		if len(res.Extra.Metadata.CVE) > 0 {
			cve = res.Extra.Metadata.CVE[0]
		} else if len(res.Extra.Metadata.CWE) > 0 {
			cve = res.Extra.Metadata.CWE[0] // Fallback to CWE if CVE not present
		}

		finding := domain.Finding{
			ID:          res.CheckID,
			Title:       res.CheckID,
			Description: res.Extra.Message,
			Severity:    severity,
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSAST,
			File:        res.Path,
			Line:        res.Start.Line,
			Match:       strings.TrimSpace(res.Extra.Lines),
			CVE:         cve,
		}
		findings = append(findings, finding)
	}

	return findings, nil
}
