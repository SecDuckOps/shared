package domain

import (
	"time"
)

// ScannerType identifies the category of security scanner
type ScannerType string

const (
	ScannerTypeSAST       ScannerType = "SAST"
	ScannerTypeDAST       ScannerType = "DAST"
	ScannerTypeSecrets    ScannerType = "SECRETS"
	ScannerTypeContainer  ScannerType = "CONTAINER"
	ScannerTypeDependency ScannerType = "DEPENDENCY"
	ScannerTypeIaC        ScannerType = "IAC"
	ScannerTypeCustom     ScannerType = "CUSTOM"
)

// Finding represents a single identified vulnerability or security issue.
type Finding struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Severity    Severity    `json:"severity"`
	Scanner     string      `json:"scanner"`       // e.g., "trivy", "semgrep"
	Type        ScannerType `json:"type"`          // e.g., SAST, IAC
	File        string      `json:"file,omitempty"`
	Line        int         `json:"line,omitempty"`
	Match       string      `json:"match,omitempty"`     // Raw matched code/text
	Remediation string      `json:"remediation,omitempty"` // Suggested fix
	CVE         string      `json:"cve,omitempty"`       // Common Vulnerabilities and Exposures ID if applicable
}

// ScanStats holds aggregated metric counts for a scan
type ScanStats struct {
	TotalFinds int            `json:"total_findings"`
	BySeverity map[Severity]int `json:"by_severity"`
}

// ScanResult contains the full parsed output from a single scanner run
type ScanResult struct {
	ScanID      string    `json:"scan_id"`
	ScannerName string    `json:"scanner_name"`
	Target      string    `json:"target"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    string    `json:"duration"`
	Findings    []Finding `json:"findings"`
	Stats       ScanStats `json:"stats"`
	Error       string    `json:"error,omitempty"`      // populated if scan failed
	RawOutput   string    `json:"raw_output,omitempty"` // populated if parsing fails or explicitly requested
}

// ScanResultRecord is the Database entity representing a completed overall scan
type ScanResultRecord struct {
	ID          string       `json:"id"`
	Target      string       `json:"target"`
	CreatedAt   time.Time    `json:"created_at"`
	TotalStats  ScanStats    `json:"total_stats"`
}

// ScanFilter defines querying parameters for the metadata storage port
type ScanFilter struct {
	Target string
	Since  time.Time
	Limit  int
}
