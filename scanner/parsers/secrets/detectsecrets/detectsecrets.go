package detectsecrets

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// DetectSecretsParser parses JSON output from Yelp's detect-secrets
type DetectSecretsParser struct{}

func NewDetectSecretsParser() ports.ResultParserPort {
	return &DetectSecretsParser{}
}

func (p *DetectSecretsParser) ScannerName() string {
	return "detectsecrets"
}

func (p *DetectSecretsParser) ScannerType() domain.ScannerType {
	return domain.ScannerTypeSecrets
}

func (p *DetectSecretsParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *DetectSecretsParser) GetScanCommand(target string) []string {
	// detect-secrets scan command
	return []string{"scan", "."}
}

type detectSecretsOutput struct {
	Results map[string][]struct {
		Type         string `json:"type"`
		LineNumber   int    `json:"line_number"`
		HashedSecret string `json:"hashed_secret"`
	} `json:"results"`
}

func (p *DetectSecretsParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed detectSecretsOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse detect-secrets json: %w", err)
	}

	var findings []domain.Finding
	for filename, secretsFound := range parsed.Results {
		for _, secret := range secretsFound {

			finding := domain.Finding{
				ID:          secret.Type,
				Title:       fmt.Sprintf("Secret: %s", secret.Type),
				Description: "Hardcoded secret discovered.",
				Severity:    domain.SeverityCritical, // Secrets generally handled as CRITICAL
				Scanner:     p.ScannerName(),
				Type:        domain.ScannerTypeSecrets,
				File:        filename,
				Line:        secret.LineNumber,
				Match:       fmt.Sprintf("Hash: %s", secret.HashedSecret),
			}

			findings = append(findings, finding)
		}
	}

	return findings, nil
}
