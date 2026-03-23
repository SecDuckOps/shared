package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/SecDuckOps/shared/core/session"
	"github.com/SecDuckOps/shared/infra/bus"
	"github.com/SecDuckOps/shared/logger"
	"github.com/SecDuckOps/shared/ports"
	"github.com/SecDuckOps/shared/types"
)

// SessionHandler handles all HTTP transport logic for session operations.
type SessionHandler struct {
	sessions    session.SessionRepository
	messages    session.MessageRepository
	checkpoints session.CheckpointRepository
	bus         bus.EventPublisher
	logger      *logger.Logger
}

// NewSessionHandler creates a SessionHandler with pure domain dependencies.
func NewSessionHandler(
	sessions session.SessionRepository,
	messages session.MessageRepository,
	checkpoints session.CheckpointRepository,
	eventBus bus.EventPublisher,
	log *logger.Logger,
) *SessionHandler {
	return &SessionHandler{
		sessions:    sessions,
		messages:    messages,
		checkpoints: checkpoints,
		bus:         eventBus,
		logger:      log,
	}
}

// --- Endpoints ---

type createSessionReq struct {
	Name     string `json:"name"`
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
}

// POST /v1/sessions
func (h *SessionHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, types.New(types.ErrCodeInvalidInput, "invalid JSON payload: "+err.Error()))
		return
	}

	s, err := session.NewSession(req.Name, req.UserID, req.DeviceID)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.sessions.Create(r.Context(), s); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvent(r.Context(), string(session.EventSessionCreated), map[string]string{
		"session_id": s.ID,
		"user_id":    s.UserID,
	})

	writeJSON(w, http.StatusCreated, s)
}

// GET /v1/sessions
func (h *SessionHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, types.New(types.ErrCodeInvalidInput, "missing user_id query parameter"))
		return
	}

	offset, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	list, err := h.sessions.List(r.Context(), userID, offset, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// GET /v1/sessions/{id}
func (h *SessionHandler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s, err := h.sessions.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, s)
}

// DELETE /v1/sessions/{id}
func (h *SessionHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.sessions.SoftDelete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvent(r.Context(), string(session.EventSessionDeleted), map[string]string{
		"session_id": id,
	})

	writeJSON(w, http.StatusNoContent, nil) // Ensures no body is written
}

type createMessageReq struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// POST /v1/sessions/{id}/messages
func (h *SessionHandler) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req createMessageReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, types.New(types.ErrCodeInvalidInput, "invalid JSON payload: "+err.Error()))
		return
	}

	msg, err := session.NewMessage(id, req.Content, session.Role(req.Role))
	if err != nil {
		writeError(w, err)
		return
	}

	// 1. Primary Operation: Create the message
	if err := h.messages.Create(r.Context(), msg); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvent(r.Context(), string(session.EventMessageAdded), map[string]string{
		"session_id": id,
		"message_id": msg.ID,
	})

	// 2. Best-Effort Operation: Auto-checkpoint logic
	count, err := h.messages.CountBySessionID(r.Context(), id)
	if err != nil {
		h.logger.ErrorErr(r.Context(), err, "failed to get message count; skipping auto-checkpoint")
		writeJSON(w, http.StatusCreated, msg)
		return
	}

	cp, err := session.NewCheckpoint(id, count, "Auto-checkpoint")
	if err != nil {
		h.logger.ErrorErr(r.Context(), err, "failed to build checkpoint structure; skipping auto-checkpoint")
		writeJSON(w, http.StatusCreated, msg)
		return
	}

	if err := h.checkpoints.Create(r.Context(), cp); err != nil {
		h.logger.ErrorErr(r.Context(), err, "failed to persist auto-checkpoint to repository")
	} else {
		h.publishEvent(r.Context(), string(session.EventCheckpointSaved), map[string]string{
			"session_id":    id,
			"checkpoint_id": cp.ID,
		})
	}

	// Always return the created message even if checkpointing failed
	writeJSON(w, http.StatusCreated, msg)
}

// GET /v1/sessions/{id}/messages
func (h *SessionHandler) handleListMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	offset, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	list, err := h.messages.ListBySessionID(r.Context(), id, offset, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// GET /v1/sessions/{id}/checkpoints
func (h *SessionHandler) handleListCheckpoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// The repository contract handles deterministic ordering of checkpoints.
	list, err := h.checkpoints.ListBySessionID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// publishEvent wraps bus payload encoding, embeds the event type if the payload is a map,
// and guarantees fire-and-forget handling by catching errors at the logger level.
// It explicitly enables SSE streams by mirrored publishing to both a global topic and a session-specific topic.
func (h *SessionHandler) publishEvent(ctx context.Context, topic string, payload any) {
	var b []byte
	var err error
	var sessionID string

	if m, ok := payload.(map[string]string); ok {
		m["type"] = topic // Embed event type
		sessionID = m["session_id"]
		b, err = json.Marshal(m)
	} else {
		b, err = json.Marshal(payload)
	}

	if err != nil {
		h.logger.ErrorErr(ctx, err, "Failed to marshal JSON for event bus payload", ports.Field{Key: "topic", Value: topic})
		return
	}

	// 1. Always publish to the standard global topic for backend subscribers
	if err := h.bus.Publish(ctx, bus.Message{Topic: topic, Payload: b}); err != nil {
		h.logger.ErrorErr(ctx, err, "Failed to publish global event to bus", ports.Field{Key: "topic", Value: topic})
	}

	// 2. Mirror strictly to session:{id} so the SSE endpoint receives it seamlessly!
	if sessionID != "" {
		topicMirrored := "session:" + sessionID
		if err := h.bus.Publish(ctx, bus.Message{Topic: topicMirrored, Payload: b}); err != nil {
			h.logger.ErrorErr(ctx, err, "Failed to mirror session event to bus", ports.Field{Key: "topic", Value: topicMirrored})
		}
	}
}
