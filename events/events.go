package events

import (
	"encoding/json"
	"time"
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
