package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────
// Core Event envelope
// ─────────────────────────────────────────────

// EventType defines the topic or channel the event belongs to.
type EventType string

// Event is the canonical envelope for all cross-boundary messages.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent creates a properly structured event.
func NewEvent(source string, eventType EventType, payload interface{}) (Event, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Source:    source,
		Payload:   b,
	}, nil
}

// ─────────────────────────────────────────────
// System / Agent lifecycle events
// ─────────────────────────────────────────────

const (
	AgentStarted   EventType = "agent.started"
	AgentStopped   EventType = "agent.stopped"
	SessionCreated EventType = "session.created"
	SessionEnded   EventType = "session.ended"
)

// ─────────────────────────────────────────────
// Tool execution events
// ─────────────────────────────────────────────

const (
	ToolStarted   EventType = "tool.started"
	ToolCompleted EventType = "tool.completed"
	ToolFailed    EventType = "tool.failed"
)

// ─────────────────────────────────────────────
// Scan events
// ─────────────────────────────────────────────

const (
	ScanStarted   EventType = "scan.started"
	ScanCompleted EventType = "scan.completed"
	ScanFailed    EventType = "scan.failed"
)

// ─────────────────────────────────────────────
// Subagent session events  (SubagentEventType)
// Replaces domain/subagent.EventType in the agent repo.
// ─────────────────────────────────────────────

// SubagentEventType classifies events emitted during a subagent loop.
type SubagentEventType string

const (
	// Existing events — kept for backward compatibility
	SubagentEventLog      SubagentEventType = "log"
	SubagentEventToolCall SubagentEventType = "tool_call"
	SubagentEventResult   SubagentEventType = "result"
	SubagentEventError    SubagentEventType = "error"
	SubagentEventStatus   SubagentEventType = "status_change"
	SubagentEventRetry    SubagentEventType = "retry"
	SubagentEventPaused   SubagentEventType = "paused"
	SubagentEventResumed  SubagentEventType = "resumed"
	SubagentEventThought  SubagentEventType = "thought"

	// ── Typed lifecycle events (duckops AgentEvent parity) ──────────────
	// TurnStarted is emitted at the beginning of each LLM inference turn.
	// Consumers: TUI spinner, rate monitor, telemetry.
	SubagentEventTurnStarted SubagentEventType = "turn_started"

	// TurnCompleted is emitted when a turn finishes (text or tool cycle).
	SubagentEventTurnCompleted SubagentEventType = "turn_completed"

	// RunCompleted is emitted when the session loop exits cleanly.
	// Data: RunCompletedData
	SubagentEventRunCompleted SubagentEventType = "run_completed"

	// RunError is emitted when the session loop exits with an error.
	// Data: RunErrorData
	SubagentEventRunError SubagentEventType = "run_error"

	// ToolExecutionStarted is emitted just before a tool is dispatched.
	// Data: ToolEventData
	SubagentEventToolExecutionStarted SubagentEventType = "tool_execution_started"

	// ToolExecutionCompleted is emitted after a tool returns.
	// Data: ToolCompletedData
	SubagentEventToolExecutionCompleted SubagentEventType = "tool_execution_completed"

	// UsageReport is emitted after each LLM call with token counts.
	// Data: UsageData
	SubagentEventUsageReport SubagentEventType = "usage_report"
)

// ── Typed event payload structs ──────────────────────────────────────────────

// TurnStartedData is the payload for SubagentEventTurnStarted.
type TurnStartedData struct {
	Turn    int    `json:"turn"`
	MaxTurn int    `json:"max_turn"`
	RunID   string `json:"run_id,omitempty"`
}

// TurnCompletedData is the payload for SubagentEventTurnCompleted.
type TurnCompletedData struct {
	Turn         int    `json:"turn"`
	FinishReason string `json:"finish_reason"` // "stop" | "tool_calls" | "max_tokens" | "cancelled"
	RunID        string `json:"run_id,omitempty"`
}

// RunCompletedData is the payload for SubagentEventRunCompleted.
type RunCompletedData struct {
	TotalTurns int    `json:"total_turns"`
	StopReason string `json:"stop_reason"` // "completed" | "cancelled" | "max_turns" | "error"
	RunID      string `json:"run_id,omitempty"`
}

// RunErrorData is the payload for SubagentEventRunError.
type RunErrorData struct {
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
	RunID     string `json:"run_id,omitempty"`
}

// ToolEventData is the payload for SubagentEventToolExecutionStarted.
type ToolEventData struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	RunID      string `json:"run_id,omitempty"`
}

// ToolCompletedData is the payload for SubagentEventToolExecutionCompleted.
type ToolCompletedData struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	IsError    bool   `json:"is_error"`
	RunID      string `json:"run_id,omitempty"`
}

// UsageData is the payload for SubagentEventUsageReport.
type UsageData struct {
	Turn             int `json:"turn"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SubagentEvent is a single event emitted from inside a subagent session loop.
type SubagentEvent struct {
	SessionID string            `json:"session_id"`
	RunID     string            `json:"run_id,omitempty"`
	Type      SubagentEventType `json:"type"`
	Message   string            `json:"message,omitempty"`
	Data      any               `json:"data,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ─────────────────────────────────────────────
// Stream events (TUI progressive rendering)
// Replaces domain/stream.go in the agent repo.
// ─────────────────────────────────────────────

// StreamEventType classifies structured output events for the TUI.
type StreamEventType string

const (
	StreamEventStdout      StreamEventType = "stdout"
	StreamEventStderr      StreamEventType = "stderr"
	StreamEventThought     StreamEventType = "thought"
	StreamEventStatus      StreamEventType = "status"
	StreamEventWardenAlert StreamEventType = "warden_alert"
	StreamEventProgress    StreamEventType = "progress"
	StreamEventComplete    StreamEventType = "complete"
	StreamEventError       StreamEventType = "error"
)

// StreamEvent is a structured output event for progressive TUI rendering.
type StreamEvent struct {
	TaskID    string            `json:"task_id"`
	EventType StreamEventType   `json:"event_type"`
	Payload   []byte            `json:"payload"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewStreamEvent creates a new stream event.
func NewStreamEvent(taskID string, eventType StreamEventType, payload []byte) StreamEvent {
	return StreamEvent{
		TaskID:    taskID,
		EventType: eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// ─────────────────────────────────────────────
// Scan domain events (bus messages)
// Replaces parts of agent/internal/domain/events.go
// ─────────────────────────────────────────────

// ScannerType identifies the category of security scanner.
type ScannerType string

const (
	ScannerTypeSAST       ScannerType = "SAST"
	ScannerTypeDAST       ScannerType = "DAST"
	ScannerTypeSecrets    ScannerType = "SECRETS"
	ScannerTypeContainer  ScannerType = "CONTAINER"
	ScannerTypeDependency ScannerType = "DEPENDENCY"
	ScannerTypeIaC        ScannerType = "IAC"
)

// ScanStatus represents the lifecycle state of a scan job.
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "PENDING"
	ScanStatusRunning   ScanStatus = "RUNNING"
	ScanStatusCompleted ScanStatus = "COMPLETED"
	ScanStatusFailed    ScanStatus = "FAILED"
)

// Severity represents the impact level of a vulnerability.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// Vulnerability is a single security finding emitted on the bus.
type Vulnerability struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    Severity  `json:"severity"`
	Location    string    `json:"location"`
	Line        int       `json:"line,omitempty"`
	CVE         string    `json:"cve,omitempty"`
	CVSS        float64   `json:"cvss,omitempty"`
	Remediation string    `json:"remediation,omitempty"`
	References  []string  `json:"references,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ScanRequestEvent is published to the bus to trigger a scan worker.
type ScanRequestEvent struct {
	ID          string            `json:"id"`
	Target      string            `json:"target"`
	ScannerType ScannerType       `json:"scanner_type"`
	Priority    int               `json:"priority,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RequestedAt time.Time         `json:"requested_at"`
	RequestedBy string            `json:"requested_by,omitempty"`
}

// ScanResultEvent is published back by a scanner worker after completing a scan.
type ScanResultEvent struct {
	ScanID          string          `json:"scan_id"`
	ScannerType     ScannerType     `json:"scanner_type"`
	Status          ScanStatus      `json:"status"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Logs            []string        `json:"logs"`
	Summary         ScanSummary     `json:"summary"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     time.Time       `json:"completed_at"`
	Error           string          `json:"error,omitempty"`
}

// ScanSummary provides a quick count of findings by severity.
type ScanSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// ComputeSummary populates ScanSummary from the Vulnerabilities slice.
func (r *ScanResultEvent) ComputeSummary() {
	r.Summary = ScanSummary{}
	for _, v := range r.Vulnerabilities {
		r.Summary.Total++
		switch v.Severity {
		case SeverityCritical:
			r.Summary.Critical++
		case SeverityHigh:
			r.Summary.High++
		case SeverityMedium:
			r.Summary.Medium++
		case SeverityLow:
			r.Summary.Low++
		case SeverityInfo:
			r.Summary.Info++
		}
	}
}
