package session

import (
	"context"
	"time"

	"github.com/SecDuckOps/shared/types"
)

// Standard domain errors used by persistence implementations.
var (
	// Record does not exist.
	ErrNotFound = types.New(types.ErrCodeNotFound, "record not found")

	// Duplicate key / unique constraint violation.
	ErrAlreadyExists = types.New(types.ErrCodeAlreadyExists, "record already exists")

	// Optimistic locking conflict.
	ErrVersionConflict = types.New(types.ErrCodeVersionConflict, "version conflict — record was modified concurrently")
)

// Persistence contract for session aggregates.
type SessionRepository interface {

	// Persist a new session.
	Create(ctx context.Context, s *Session) error

	// Retrieve a session by its identifier.
	GetByID(ctx context.Context, id string) (*Session, error)

	// Return sessions for a user with pagination (newest first).
	List(ctx context.Context, userID string, offset, limit int) ([]*Session, error)

	// Persist modifications to an existing session.
	Update(ctx context.Context, s *Session) error

	// Mark a session as deleted without removing stored data.
	SoftDelete(ctx context.Context, id string) error
}

// Persistence contract for conversation messages.
type MessageRepository interface {

	// Append a message to a session conversation.
	Create(ctx context.Context, m *Message) error

	// Retrieve messages for a session with pagination.
	ListBySessionID(ctx context.Context, sessionID string, offset, limit int) ([]*Message, error)

	// Return the total number of messages in a session.
	CountBySessionID(ctx context.Context, sessionID string) (int, error)
}

// Persistence contract for conversation checkpoints.
type CheckpointRepository interface {

	// Persist a checkpoint snapshot.
	Create(ctx context.Context, cp *Checkpoint) error

	// Retrieve checkpoints for a session ordered by message index.
	ListBySessionID(ctx context.Context, sessionID string) ([]*Checkpoint, error)

	// Retrieve the most recent checkpoint.
	GetLatestBySessionID(ctx context.Context, sessionID string) (*Checkpoint, error)
}

// Pending synchronization operation for offline-first workflows.
type SyncItem struct {
	ID        string
	TableName string
	RecordID  string
	Operation string
	Attempts  int
	CreatedAt time.Time
	LastError string
}

// Contract for managing the synchronization queue.
type SyncQueue interface {

	// Add an item to the queue.
	Enqueue(ctx context.Context, item SyncItem) error

	// Retrieve pending items (oldest first).
	Dequeue(ctx context.Context, limit int) ([]SyncItem, error)

	// Mark an item as successfully synchronized.
	MarkSynced(ctx context.Context, id string) error

	// Record a failed attempt.
	MarkFailed(ctx context.Context, id string, reason string) error

	// Return the number of pending items.
	PendingCount(ctx context.Context) (int, error)
}
