package domain

import "strings"

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
	SeverityUnknown  Severity = "UNKNOWN"
)

// NormalizeSeverity maps various scanner specific severity strings to our standard Severity enum.
func NormalizeSeverity(s string) Severity {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL", "CRIT":
		return SeverityCritical
	case "HIGH":
		return SeverityHigh
	case "MEDIUM", "MED":
		return SeverityMedium
	case "LOW":
		return SeverityLow
	case "INFO", "INFORMATIONAL", "NOTE":
		return SeverityInfo
	default:
		return SeverityUnknown
	}
}
