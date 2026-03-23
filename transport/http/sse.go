package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/SecDuckOps/shared/core/session"
	"github.com/SecDuckOps/shared/infra/bus"
	"github.com/SecDuckOps/shared/logger"
	"github.com/SecDuckOps/shared/types"
)

// SSEHandler orchestrates real-time Server-Sent Events broadcasting.
type SSEHandler struct {
	bus      bus.EventSubscriber
	logger   *logger.Logger
	sessions session.SessionRepository
}

// NewSSEHandler leverages the domain SessionRepository to correctly 404 fake connections.
// It mandates exact dependencies fail-fast panicking on nil inputs to prevent silent crashes later.
func NewSSEHandler(eventBus bus.EventSubscriber, log *logger.Logger, sessions session.SessionRepository) *SSEHandler {
	if eventBus == nil {
		panic("eventBus EventSubscriber cannot be nil")
	}
	if log == nil {
		panic("logger cannot be nil")
	}
	if sessions == nil {
		panic("sessions SessionRepository cannot be nil")
	}

	return &SSEHandler{
		bus:      eventBus,
		logger:   log,
		sessions: sessions,
	}
}

// handleSubscribeEvents opens a unidirectional EventSource stream for browser or Agent clients.
func (h *SSEHandler) handleSubscribeEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// 1. Validate Session Existence pre-subscription (guarantees real events routing).
	if _, err := h.sessions.GetByID(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	// 2. Validate SSE capabilities
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, types.New(types.ErrCodeInvalidInput, "streaming unsupported by client connection"))
		return
	}

	// 3. Setup Bus Subscription strictly against the matched Session Topic routing
	topic := "session:" + id
	ch, cleanup, err := h.bus.Subscribe(r.Context(), topic)
	if err != nil {
		h.logger.ErrorErr(r.Context(), err, "failed to subscribe to bus for SSE stream")
		writeError(w, types.New(types.ErrCodeInternal, "failed to establish event subscription"))
		return
	}
	if ch == nil || cleanup == nil {
		h.logger.ErrorErr(r.Context(), nil, "bus returned nil channel or cleanup func")
		if cleanup != nil {
			cleanup()
		}
		writeError(w, types.New(types.ErrCodeInternal, "invalid subscription state"))
		return
	}
	// Crucial: Guarantees goroutine/connection cleanup on any exit path!
	defer cleanup()

	// 4. Secure SSE Headers + flush explicit 200 OK before iterating
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// 5. Select Loop Lifecycle with Heartbeat
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client Disconnected OR proxy timeout. Normal termination behavior.
			return

		case <-ticker.C:
			// SSE Keep-Alive Heartbeat prevents AWS/Nginx from silently murdering idle connections!
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case msg, ok := <-ch:
			if !ok {
				// Channel gracefully closed upstream
				return
			}

			// Pre-parse the event type natively pushed inside the augmented JSON
			// (Written beautifully by session_handler.publishEvent). 
			var peek struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(msg.Payload, &peek)

			// Safely decompose multiline JSON bodies guaranteeing standard SSE spec rules
			lines := bytes.Split(msg.Payload, []byte("\n"))

			if peek.Type != "" {
				if _, err := fmt.Fprintf(w, "event: %s\n", peek.Type); err != nil {
					return // Dead connection write failure. Break to run defer cleanup()
				}
			}

			for _, line := range lines {
				if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
					return
				}
			}

			if _, err := fmt.Fprintf(w, "\n\n"); err != nil {
				return
			}

			flusher.Flush()
		}
	}
}
