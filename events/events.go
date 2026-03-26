package events

import (
	"encoding/json"
	"time"

	scanner_domain "github.com/SecDuckOps/shared/scanner/domain"
	"github.com/google/uuid"
)

// EventType defines the topic or channel the event belongs to.
type EventType string

const (
	// System Events
	AgentStarted EventType = "agent.started"
	AgentStopped EventType = "agent.stopped"

	// Tool Execution Events
	ToolStarted   EventType = "tool.started"
	ToolCompleted EventType = "tool.completed"
	ToolFailed    EventType = "tool.failed"

	// Scan Specific Events
	ScanStarted   EventType = "scan.started"
	ScanCompleted EventType = "scan.completed"
)

// Event is the canonical structure for all cross-boundary messages.
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent is a helper for creating properly structured events.
func NewEvent(source string, eventType EventType, payload interface{}) (Event, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}

	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Source:    source,
		Payload:   bytes,
	}, nil
}

// Scanner and finding aliases shared across agent/server boundaries.
type ScannerType = scanner_domain.ScannerType
type Severity = scanner_domain.Severity
type Vulnerability = scanner_domain.Finding

const (
	ScannerTypeSAST       = scanner_domain.ScannerTypeSAST
	ScannerTypeDAST       = scanner_domain.ScannerTypeDAST
	ScannerTypeSecrets    = scanner_domain.ScannerTypeSecrets
	ScannerTypeContainer  = scanner_domain.ScannerTypeContainer
	ScannerTypeDependency = scanner_domain.ScannerTypeDependency
	ScannerTypeIaC        = scanner_domain.ScannerTypeIaC
	ScannerTypeCustom     = scanner_domain.ScannerTypeCustom

	SeverityCritical = scanner_domain.SeverityCritical
	SeverityHigh     = scanner_domain.SeverityHigh
	SeverityMedium   = scanner_domain.SeverityMedium
	SeverityLow      = scanner_domain.SeverityLow
	SeverityInfo     = scanner_domain.SeverityInfo
	SeverityUnknown  = scanner_domain.SeverityUnknown
)

// ScanStatus tracks lifecycle state for a scan request.
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
)

// ScanRequestEvent is emitted when a scan is requested.
type ScanRequestEvent struct {
	ScanID      string      `json:"scan_id,omitempty"`
	SessionID   string      `json:"session_id,omitempty"`
	Target      string      `json:"target"`
	Scanner     string      `json:"scanner,omitempty"`
	ScannerType ScannerType `json:"scanner_type,omitempty"`
	RequestedAt time.Time   `json:"requested_at,omitempty"`
}

// ScanSummary captures aggregate counts for a scan.
type ScanSummary struct {
	TotalFindings int              `json:"total_findings"`
	BySeverity    map[Severity]int `json:"by_severity,omitempty"`
	ScannerCount  int              `json:"scanner_count,omitempty"`
}

// ScanResultEvent is emitted when a scan completes.
type ScanResultEvent struct {
	ScanID      string          `json:"scan_id,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	Scanner     string          `json:"scanner,omitempty"`
	ScannerType ScannerType     `json:"scanner_type,omitempty"`
	Status      ScanStatus      `json:"status"`
	Findings    []Vulnerability `json:"findings,omitempty"`
	Summary     ScanSummary     `json:"summary,omitempty"`
	Error       string          `json:"error,omitempty"`
	CompletedAt time.Time       `json:"completed_at,omitempty"`
}

// StreamEventType identifies a UI/terminal stream chunk.
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

// StreamEvent is a lightweight streaming payload for UI updates.
type StreamEvent struct {
	Type      StreamEventType `json:"type"`
	Content   string          `json:"content,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Data      interface{}     `json:"data,omitempty"`
}

// NewStreamEvent creates a stream event with the current UTC time.
func NewStreamEvent(eventType StreamEventType, content string, data ...interface{}) StreamEvent {
	evt := StreamEvent{
		Type:      eventType,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}
	if len(data) > 0 {
		evt.Data = data[0]
	}
	return evt
}

// SubagentEventType tracks delegated-agent lifecycle events.
type SubagentEventType string

const (
	SubagentEventLog                    SubagentEventType = "log"
	SubagentEventToolCall               SubagentEventType = "tool_call"
	SubagentEventResult                 SubagentEventType = "result"
	SubagentEventError                  SubagentEventType = "error"
	SubagentEventStatus                 SubagentEventType = "status"
	SubagentEventRetry                  SubagentEventType = "retry"
	SubagentEventPaused                 SubagentEventType = "paused"
	SubagentEventResumed                SubagentEventType = "resumed"
	SubagentEventThought                SubagentEventType = "thought"
	SubagentEventStreamToken            SubagentEventType = "stream_token"
	SubagentEventTurnStarted            SubagentEventType = "turn_started"
	SubagentEventTurnCompleted          SubagentEventType = "turn_completed"
	SubagentEventRunCompleted           SubagentEventType = "run_completed"
	SubagentEventRunError               SubagentEventType = "run_error"
	SubagentEventToolExecutionStarted   SubagentEventType = "tool_execution_started"
	SubagentEventToolExecutionCompleted SubagentEventType = "tool_execution_completed"
	SubagentEventUsageReport            SubagentEventType = "usage_report"
)

// SubagentEvent is the canonical event shape for delegated sessions.
type SubagentEvent struct {
	Type      SubagentEventType `json:"type"`
	Message   string            `json:"message,omitempty"`
	Data      interface{}       `json:"data,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	SessionID string            `json:"session_id,omitempty"`
	RunID     string            `json:"run_id,omitempty"`
}

// Typed payloads for structured subagent lifecycle events.
type TurnStartedData struct {
	Turn    int    `json:"turn"`
	MaxTurn int    `json:"max_turn"`
	RunID   string `json:"run_id,omitempty"`
}

type TurnCompletedData struct {
	Turn         int    `json:"turn"`
	FinishReason string `json:"finish_reason,omitempty"`
	RunID        string `json:"run_id,omitempty"`
}

type RunCompletedData struct {
	TotalTurns int    `json:"total_turns"`
	StopReason string `json:"stop_reason,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

type RunErrorData struct {
	Error     string `json:"error"`
	Retryable bool   `json:"retryable,omitempty"`
	RunID     string `json:"run_id,omitempty"`
}

type ToolEventData struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	RunID      string `json:"run_id,omitempty"`
}

type ToolCompletedData struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	IsError    bool   `json:"is_error,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

type UsageData struct {
	Turn             int `json:"turn"`
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}
