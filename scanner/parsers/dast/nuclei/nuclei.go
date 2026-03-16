package nuclei

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// NucleiParser parses JSON Lines output from ProjectDiscovery Nuclei (DAST)
type NucleiParser struct{}

func NewNucleiParser() ports.ResultParserPort {
	return &NucleiParser{}
}

func (p *NucleiParser) ScannerName() string {
	return "nuclei"
}

func (p *NucleiParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *NucleiParser) GetScanCommand(target string) []string {
	// Nuclei scan command with JSON output. Target can be URL or CIDR/IP, but we assume URL for DAST.
	return []string{"-u", target, "-json"}
}

type nucleiOutput struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name        string   `json:"name"`
		Severity    string   `json:"severity"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"info"`
	Type             string   `json:"type"`
	Host             string   `json:"host"`
	MatchedAt        string   `json:"matched-at"`
	ExtractedResults []string `json:"extracted-results"`
}

func (p *NucleiParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var findings []domain.Finding
	scanner := bufio.NewScanner(bytes.NewReader(raw))

	for scanner.Scan() {
		lineText := scanner.Bytes()
		if len(lineText) == 0 {
			continue
		}

		var parsed nucleiOutput
		if err := json.Unmarshal(lineText, &parsed); err != nil {
			// Skip lines that aren't valid JSON
			continue
		}

		// Ensure we don't pick up completely empty lines masquerading as valid
		if parsed.TemplateID == "" {
			continue
		}

		extracted := ""
		if len(parsed.ExtractedResults) > 0 {
			extracted = "Extracted: " + strings.Join(parsed.ExtractedResults, ", ")
		}

		finding := domain.Finding{
			ID:          parsed.TemplateID,
			Title:       parsed.Info.Name,
			Description: parsed.Info.Description,
			Severity:    domain.NormalizeSeverity(parsed.Info.Severity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeDAST,
			File:        parsed.Host,
			Match:       fmt.Sprintf("[%s] %s | %s", parsed.Type, parsed.MatchedAt, extracted),
		}

		// Try setting CVE if we see cve in tags or template ID
		if strings.HasPrefix(strings.ToLower(parsed.TemplateID), "cve-") {
			finding.CVE = strings.ToUpper(parsed.TemplateID)
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
