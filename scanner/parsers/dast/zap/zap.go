package zap

import (
	"encoding/json"
	"fmt"

	"github.com/SecDuckOps/shared/scanner/domain"
	"github.com/SecDuckOps/shared/scanner/ports"
)

// ZapParser parses JSON output from OWASP ZAP (DAST)
type ZapParser struct{}

func NewZapParser() ports.ResultParserPort {
	return &ZapParser{}
}

func (p *ZapParser) ScannerName() string {
	return "zap"
}

func (p *ZapParser) SupportedFormats() []string {
	return []string{"json"}
}

func (p *ZapParser) GetScanCommand(target string) []string {
	// ZAP baseline scan for a target URI. 
	// Note: target for ZAP is often a URL, but we default to baseline scan for now.
	return []string{"zap-baseline.py", "-t", target, "-J", "report.json"}
}

type zapOutput struct {
	Site []struct {
		Name   string `json:"@name"`
		Alerts []struct {
			PluginID  string `json:"pluginid"`
			Name      string `json:"alert"`
			RiskDesc  string `json:"riskdesc"`
			Desc      string `json:"desc"`
			Instances []struct {
				URI      string `json:"uri"`
				Method   string `json:"method"`
				Evidence string `json:"evidence"`
			} `json:"instances"`
		} `json:"alerts"`
	} `json:"site"`
}

func (p *ZapParser) Parse(raw []byte) ([]domain.Finding, error) {
	if len(raw) == 0 {
		return []domain.Finding{}, nil
	}

	var parsed zapOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse zap json: %w", err)
	}

	var findings []domain.Finding
	for _, site := range parsed.Site {
		for _, alert := range site.Alerts {
			// e.g "High (Medium)" -> High
			riskLevel := alert.RiskDesc
			if len(riskLevel) > 4 && riskLevel[:4] == "High" {
				riskLevel = "HIGH"
			} else if len(riskLevel) > 6 && riskLevel[:6] == "Medium" {
				riskLevel = "MEDIUM"
			} else if len(riskLevel) > 3 && riskLevel[:3] == "Low" {
				riskLevel = "LOW"
			} else {
				riskLevel = "INFO"
			}

			severity := domain.NormalizeSeverity(riskLevel)

			for _, instance := range alert.Instances {
				finding := domain.Finding{
					ID:          alert.PluginID,
					Title:       alert.Name,
					Description: alert.Desc,
					Severity:    severity,
					Scanner:     p.ScannerName(),
					Type:        domain.ScannerTypeDAST,
					File:        instance.URI,
					Match:       fmt.Sprintf("%s %s (Evidence: %s)", instance.Method, instance.URI, instance.Evidence),
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings, nil
}
