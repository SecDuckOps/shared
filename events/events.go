package events

import "time"

// RawInputReceived represents the initial input from the user (CLI/UI)
type RawInputReceived struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

// CommandAccepted is issued by the Prompt Engine when a command is parsed and safe
type CommandAccepted struct {
	ID         string            `json:"id"`
	Intent     string            `json:"intent"`
	Target     string            `json:"target"`
	Parameters map[string]string `json:"parameters"`
	Timestamp  time.Time         `json:"timestamp"`
}

// ScanTask is the command sent to the Agent for execution
type ScanTask struct {
	ID        string                 `json:"id"`
	Tool      string                 `json:"tool"`
	Args      map[string]interface{} `json:"args"`
	Timestamp time.Time              `json:"timestamp"`
}

// ScanResult is the final output from the Agent
type ScanResult struct {
	ScanID          string      `json:"scan_id"`
	Status          string      `json:"status"` // completed, failed
	Vulnerabilities interface{} `json:"vulnerabilities,omitempty"`
	Logs            []string    `json:"logs,omitempty"`
	StartedAt       time.Time   `json:"started_at"`
	FinishedAt      time.Time   `json:"finished_at"`
	Error           string      `json:"error,omitempty"`
}
