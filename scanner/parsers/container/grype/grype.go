package grype

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// GrypeParser parses JSON output from Anchore Grype
type GrypeParser struct{}

func NewGrypeParser() ports.ResultParserPort {
	return &GrypeParser{}
}

func (p *GrypeParser) ScannerName() string {
	return "grype"
}

func (p *GrypeParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeContainer
}

func (p *GrypeParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *GrypeParser) GetScanCommand(target string) []string {
	// Grype scan command with JSON output
	return []string{"dir:.", "-o", "json"}
}

type grypeOutput struct {
	Matches []struct {
		Vulnerability struct {
			ID          string `json:"id"`
			Severity    string `json:"severity"`
			Description string `json:"description"`
		} `json:"vulnerability"`
		Artifact struct {
			Name      string `json:"name"`
			Version   string `json:"version"`
			Locations []struct {
				Path string `json:"path"`
			} `json:"locations"`
		} `json:"artifact"`
	} `json:"matches"`
}

func (p *GrypeParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed grypeOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse grype json: %w", err)
	}

	var findings []domain.Finding
	for _, match := range parsed.Matches {
		path := ""
		if len(match.Artifact.Locations) > 0 {
			path = match.Artifact.Locations[0].Path
		}

		finding := domain.Finding{
			ID:          match.Vulnerability.ID,
			Title:       fmt.Sprintf("%s in %s", match.Vulnerability.ID, match.Artifact.Name),
			Description: match.Vulnerability.Description,
			Severity:    domain.NormalizeSeverity(match.Vulnerability.Severity),
			Scanner:     p.ScannerName(),
			Type:        domain.ScannerTypeContainer, // Grype primarily scans containers, though it can scan dirs
			File:        path,
			Match:       fmt.Sprintf("Package: %s, Version: %s", match.Artifact.Name, match.Artifact.Version),
			CVE:         match.Vulnerability.ID,
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
