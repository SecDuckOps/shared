package session

import (
	"fmt"
	"time"

	"github.com/SecDuckOps/shared/types"
	"github.com/SecDuckOps/shared/validation"
	"github.com/google/uuid"
)

// Constants
// ============================================================================

// Domain constraints — input length limits
const (
	MaxSessionNameLen   = 1_000
	MaxIdentifierLen    = 256
	MaxMessageLen       = 500_000
	MaxCheckpointLen    = 50_000
	MaxMetadataKeyLen   = 128
	MaxMetadataValueLen = 10_000
	MaxMetadataEntries  = 64
)

// Types & Enums
// ============================================================================

// synchronization state Offline first
type SyncStatus string

const (
	SyncStatusLocalOnly SyncStatus = "local_only"
	SyncStatusPending   SyncStatus = "pending_sync"
	SyncStatusSynced    SyncStatus = "synced"
	SyncStatusConflict  SyncStatus = "conflict"
)

// Session lifecycle state
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusClosed  SessionStatus = "closed"
	SessionStatusDeleted SessionStatus = "deleted"
)

// Message author identity
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Domain event identifiers <Event Driven Architecture>
type EventType string

const (
	EventSessionCreated  EventType = "session.created"
	EventSessionUpdated  EventType = "session.updated"
	EventSessionClosed   EventType = "session.closed"
	EventSessionDeleted  EventType = "session.deleted"
	EventMessageAdded    EventType = "message.added"
	EventCheckpointSaved EventType = "checkpoint.saved"
)

// Structs
// ============================================================================

// Session — aggregate root for a conversation session
type Session struct {
	ID         string
	UserID     string
	DeviceID   string
	Name       string
	Status     SessionStatus
	Metadata   map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	Version    int
	SyncStatus SyncStatus
}

// single conversation turn within a session
type Message struct {
	ID         string
	SessionID  string
	Role       Role
	Content    string
	CreatedAt  time.Time
	SyncStatus SyncStatus
}

// conversation snapshot at a specific message index
type Checkpoint struct {
	ID           string
	SessionID    string
	MessageIndex int
	Summary      string
	CreatedAt    time.Time
	SyncStatus   SyncStatus
}

// Event — domain event record
type Event struct {
	ID         string
	SessionID  string
	Type       EventType
	Payload    map[string]interface{}
	OccurredAt time.Time
}

// Internal helpers
// ============================================================================

// allowed sync state machine
func syncTransitions(from SyncStatus) []SyncStatus {
	switch from {
	case SyncStatusLocalOnly:
		return []SyncStatus{SyncStatusPending}
	case SyncStatusPending:
		return []SyncStatus{SyncStatusSynced, SyncStatusConflict}
	case SyncStatusSynced:
		return []SyncStatus{SyncStatusPending}
	case SyncStatusConflict:
		return []SyncStatus{SyncStatusPending}
	default:
		return nil
	}
}

//  Factory & Methods
// ============================================================================

// creates a validated Session with sensible defaults
func NewSession(name, userID, deviceID string) (*Session, error) {
	var err error

	name, err = validation.RequireMaxLen(name, "session name", MaxSessionNameLen)
	if err != nil {
		return nil, err
	}

	userID, err = validation.RequireIdentifier(userID, "user ID")
	if err != nil {
		return nil, err
	}

	deviceID, err = validation.RequireIdentifier(deviceID, "device ID")
	if err != nil {
		return nil, err
	}

	now := time.Now()

	return &Session{
		ID:         uuid.NewString(), //Unique UUID for the session
		UserID:     userID,
		DeviceID:   deviceID,
		Name:       name,
		Status:     SessionStatusActive,
		Metadata:   make(map[string]string),
		CreatedAt:  now,
		UpdatedAt:  now,
		DeletedAt:  nil,
		Version:    1,
		SyncStatus: SyncStatusLocalOnly,
	}, nil
}

// transitions an active session to closed state
func (s *Session) Close() error {
	if s.Status != SessionStatusActive {
		return types.Newf(types.ErrCodeInvalidTransition,
			"cannot close session in '%s' state, must be '%s'", s.Status, SessionStatusActive)
	}
	s.Status = SessionStatusClosed
	s.UpdatedAt = time.Now()
	s.Version++
	return nil
}

func (s *Session) SoftDelete() error {
	if s.Status == SessionStatusDeleted {
		return types.New(types.ErrCodeInvalidTransition, "session is already deleted")
	}
	now := time.Now()
	s.Status = SessionStatusDeleted
	s.DeletedAt = &now
	s.UpdatedAt = now
	s.Version++
	return nil
}

func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive
}

func (s *Session) IsDeleted() bool {
	return s.Status == SessionStatusDeleted
}

// adds or updates a metadata entry
func (s *Session) SetMetadata(key, value string) error {
	var err error

	key, err = validation.RequireMaxLen(key, "metadata key", MaxMetadataKeyLen)
	if err != nil {
		return err
	}
	if len(value) > MaxMetadataValueLen {
		return types.Newf(types.ErrCodeInvalidInput, "metadata value exceeds max length of %d", MaxMetadataValueLen)
	}

	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}

	// allow update of existing keys without counting against the limit
	if _, exists := s.Metadata[key]; !exists && len(s.Metadata) >= MaxMetadataEntries {
		return types.Newf(types.ErrCodeInvalidInput, "metadata entries exceed max count of %d", MaxMetadataEntries)
	}

	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
	return nil
}

// transitions the session's SyncStatus
func (s *Session) TransitionSync(target SyncStatus) error {
	allowed := syncTransitions(s.SyncStatus)
	if allowed == nil {
		return types.Newf(types.ErrCodeInvalidTransition, "unknown current sync status '%s'", s.SyncStatus)
	}
	for _, valid := range allowed {
		if target == valid {
			s.SyncStatus = target
			s.UpdatedAt = time.Now()
			return nil
		}
	}
	return types.Newf(types.ErrCodeInvalidTransition,
		"sync transition from '%s' to '%s' is not allowed", s.SyncStatus, target)
}

// transitions sync status to pending_sync
func (s *Session) MarkPending() error { return s.TransitionSync(SyncStatusPending) }

// transitions sync status to synced
func (s *Session) MarkSynced() error { return s.TransitionSync(SyncStatusSynced) }

// transitions sync status to conflict
func (s *Session) MarkConflict() error { return s.TransitionSync(SyncStatusConflict) }

// Message Factory
// ============================================================================

// creates a validated Message belonging to a session
func NewMessage(sessionID, content string, role Role) (*Message, error) {
	var err error

	sessionID, err = validation.RequireIdentifier(sessionID, "session ID")
	if err != nil {
		return nil, err
	}

	content, err = validation.RequireMaxLen(content, "message content", MaxMessageLen)
	if err != nil {
		return nil, err
	}

	if role != RoleUser && role != RoleAssistant && role != RoleSystem {
		return nil, types.New(types.ErrCodeInvalidInput, "invalid message role")
	}

	return &Message{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		Role:       role,
		Content:    content,
		CreatedAt:  time.Now(),
		SyncStatus: SyncStatusLocalOnly,
	}, nil
}

// Checkpoint Factory
// ============================================================================

// creates a Checkpoint for resuming long conversations
func NewCheckpoint(sessionID string, messageIndex int, summary string) (*Checkpoint, error) {
	var err error

	sessionID, err = validation.RequireIdentifier(sessionID, "session ID")
	if err != nil {
		return nil, err
	}

	if messageIndex < 0 {
		return nil, types.New(types.ErrCodeInvalidInput, "message index must be non-negative")
	}

	summary, err = validation.RequireMaxLen(summary, "checkpoint summary", MaxCheckpointLen)
	if err != nil {
		return nil, err
	}

	return &Checkpoint{
		ID:           uuid.NewString(),
		SessionID:    sessionID,
		MessageIndex: messageIndex,
		Summary:      summary,
		CreatedAt:    time.Now(),
		SyncStatus:   SyncStatusLocalOnly,
	}, nil
}

// Event -  Factory & Methods
// ============================================================================

// creates a domain event with validation
func NewEvent(sessionID string, eventType EventType, payload map[string]interface{}) (*Event, error) {
	var err error

	sessionID, err = validation.RequireNonEmpty(sessionID, "event session ID")
	if err != nil {
		return nil, err
	}
	if eventType == "" {
		return nil, types.New(types.ErrCodeInvalidInput, "event type is required")
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}

	return &Event{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		Type:       eventType,
		Payload:    payload,
		OccurredAt: time.Now(),
	}, nil
}

// returns a human-readable representation of the event
func (e *Event) String() string {
	return fmt.Sprintf("[%s] %s @ %s", e.Type, e.SessionID, e.OccurredAt.Format(time.RFC3339))
}
