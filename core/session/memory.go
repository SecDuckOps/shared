package session

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is shared backing state for in-memory session repositories.
type MemoryStore struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	messages    map[string][]*Message
	checkpoints map[string][]*Checkpoint
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions:    make(map[string]*Session),
		messages:    make(map[string][]*Message),
		checkpoints: make(map[string][]*Checkpoint),
	}
}

func (s *MemoryStore) SessionRepository() SessionRepository {
	return &MemorySessionRepository{store: s}
}

func (s *MemoryStore) MessageRepository() MessageRepository {
	return &MemoryMessageRepository{store: s}
}

func (s *MemoryStore) CheckpointRepository() CheckpointRepository {
	return &MemoryCheckpointRepository{store: s}
}

type MemorySessionRepository struct {
	store *MemoryStore
}

func (r *MemorySessionRepository) Create(_ context.Context, session *Session) error {
	if session == nil {
		return ErrNotFound
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, exists := r.store.sessions[session.ID]; exists {
		return ErrAlreadyExists
	}
	r.store.sessions[session.ID] = cloneSession(session)
	return nil
}

func (r *MemorySessionRepository) GetByID(_ context.Context, id string) (*Session, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	session, exists := r.store.sessions[id]
	if !exists {
		return nil, ErrNotFound
	}
	return cloneSession(session), nil
}

func (r *MemorySessionRepository) List(_ context.Context, userID string, offset, limit int) ([]*Session, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	var list []*Session
	for _, session := range r.store.sessions {
		if userID == "" || session.UserID == userID {
			list = append(list, cloneSession(session))
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return paginate(list, offset, limit), nil
}

func (r *MemorySessionRepository) Update(_ context.Context, session *Session) error {
	if session == nil {
		return ErrNotFound
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, exists := r.store.sessions[session.ID]; !exists {
		return ErrNotFound
	}
	r.store.sessions[session.ID] = cloneSession(session)
	return nil
}

func (r *MemorySessionRepository) SoftDelete(_ context.Context, id string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	session, exists := r.store.sessions[id]
	if !exists {
		return ErrNotFound
	}

	clone := cloneSession(session)
	if err := clone.SoftDelete(); err != nil {
		return err
	}
	r.store.sessions[id] = clone
	return nil
}

type MemoryMessageRepository struct {
	store *MemoryStore
}

func (r *MemoryMessageRepository) Create(_ context.Context, message *Message) error {
	if message == nil {
		return ErrNotFound
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	r.store.messages[message.SessionID] = append(r.store.messages[message.SessionID], cloneMessage(message))
	return nil
}

func (r *MemoryMessageRepository) ListBySessionID(_ context.Context, sessionID string, offset, limit int) ([]*Message, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	items := cloneMessages(r.store.messages[sessionID])
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return paginate(items, offset, limit), nil
}

func (r *MemoryMessageRepository) CountBySessionID(_ context.Context, sessionID string) (int, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	return len(r.store.messages[sessionID]), nil
}

type MemoryCheckpointRepository struct {
	store *MemoryStore
}

func (r *MemoryCheckpointRepository) Create(_ context.Context, checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return ErrNotFound
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	r.store.checkpoints[checkpoint.SessionID] = append(
		r.store.checkpoints[checkpoint.SessionID],
		cloneCheckpoint(checkpoint),
	)
	sort.Slice(r.store.checkpoints[checkpoint.SessionID], func(i, j int) bool {
		left := r.store.checkpoints[checkpoint.SessionID][i]
		right := r.store.checkpoints[checkpoint.SessionID][j]
		if left.MessageIndex == right.MessageIndex {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.MessageIndex < right.MessageIndex
	})
	return nil
}

func (r *MemoryCheckpointRepository) ListBySessionID(_ context.Context, sessionID string) ([]*Checkpoint, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	return cloneCheckpoints(r.store.checkpoints[sessionID]), nil
}

func (r *MemoryCheckpointRepository) GetLatestBySessionID(_ context.Context, sessionID string) (*Checkpoint, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	items := r.store.checkpoints[sessionID]
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return cloneCheckpoint(items[len(items)-1]), nil
}

func cloneSession(src *Session) *Session {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Metadata != nil {
		dst.Metadata = make(map[string]string, len(src.Metadata))
		for k, v := range src.Metadata {
			dst.Metadata[k] = v
		}
	}
	return &dst
}

func cloneMessage(src *Message) *Message {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func cloneCheckpoint(src *Checkpoint) *Checkpoint {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func cloneMessages(src []*Message) []*Message {
	items := make([]*Message, 0, len(src))
	for _, item := range src {
		items = append(items, cloneMessage(item))
	}
	return items
}

func cloneCheckpoints(src []*Checkpoint) []*Checkpoint {
	items := make([]*Checkpoint, 0, len(src))
	for _, item := range src {
		items = append(items, cloneCheckpoint(item))
	}
	return items
}

func paginate[T any](items []T, offset, limit int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	if limit <= 0 || offset+limit > len(items) {
		limit = len(items) - offset
	}
	return items[offset : offset+limit]
}

var _ SessionRepository = (*MemorySessionRepository)(nil)
var _ MessageRepository = (*MemoryMessageRepository)(nil)
var _ CheckpointRepository = (*MemoryCheckpointRepository)(nil)
