package protocol

import "time"

// Topics/Queues
const (
	QueueRawInput        = "raw_inputs"
	QueueCommandAccepted = "commands.accepted"
	QueueCommandRejected = "commands.rejected"
	QueueCommandApproval = "commands.approval_needed"
	QueueAgentTasks      = "agent_tasks"
	QueueTaskResults     = "tasks.results"
	QueueResultProcessed = "results.processed"

	// Subagent events (for distributed subagent coordination)
	QueueSubagentSpawned   = "subagent.spawned"
	QueueSubagentCompleted = "subagent.completed"
	QueueSubagentFailed    = "subagent.failed"
)

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

// ProcessedResult is published by ResultProcessor after enrichment and storage
type ProcessedResult struct {
	ScanID             string    `json:"scan_id"`
	Status             string    `json:"status"`
	VulnerabilityCount int       `json:"vulnerability_count"`
	ProcessedAt        time.Time `json:"processed_at"`
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
