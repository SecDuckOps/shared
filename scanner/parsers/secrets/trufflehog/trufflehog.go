package trufflehog

import (
	"encoding/json"
	"bufio"
	"bytes"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// TrufflehogParser parses JSON output from Trufflehog (Secrets)
// Trufflehog actually outputs JSON Lines (NDJSON), not a single JSON array
type TrufflehogParser struct{}

func NewTrufflehogParser() ports.ResultParserPort {
	return &TrufflehogParser{}
}

func (p *TrufflehogParser) ScannerName() string {
	return "trufflehog"
}

func (p *TrufflehogParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *TrufflehogParser) GetScanCommand(target string) []string {
	// Trufflehog filesystem scan with JSON output
	return []string{"filesystem", "--directory", ".", "--json"}
}

type trufflehogOutput struct {
	SourceMetadata struct {
		Data struct {
			Git struct {
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"Git"`
			Filesystem struct {
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"Filesystem"`
		} `json:"Data"`
	} `json:"SourceMetadata"`
	DetectorName string `json:"DetectorName"`
	Raw          string `json:"Raw"`
	Redacted     string `json:"Redacted"`
}

func (p *TrufflehogParser) Parse(raw []byte) ([]domain.Finding, error) {
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

		var parsed trufflehogOutput
		if err := json.Unmarshal(lineText, &parsed); err != nil {
			// Skip lines that aren't valid JSON (e.g. startup logs)
			continue
		}

		// It might be a git repository scan or a filesystem directory scan.
		file := parsed.SourceMetadata.Data.Git.File
		line := parsed.SourceMetadata.Data.Git.Line
		
		if file == "" {
			file = parsed.SourceMetadata.Data.Filesystem.File
			line = parsed.SourceMetadata.Data.Filesystem.Line
		}
		
		// If both are still empty, and DetectorName is empty, it might be an invalid finding block
		if file == "" && parsed.DetectorName == "" {
			continue
		}

		finding := domain.Finding{
			ID:          parsed.DetectorName,
			Title:       parsed.DetectorName + " Secret Discovered",
			Description: "Hardcoded secret discovered.",
			Severity:    domain.SeverityCritical,
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeSecrets,
			File:        file,
			Line:        line,
			Match:       parsed.Redacted, // Keep the redacted version for safety in storage
		}
		
		findings = append(findings, finding)
	}

	return findings, nil
}
