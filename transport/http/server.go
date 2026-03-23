package transport

import (
	"net/http"

	"github.com/SecDuckOps/shared/logger"
	"golang.org/x/time/rate"
)

// AuthMiddleware allows the authentication strategy to be swapped
// without touching routing code. Injectable via interface.
type AuthMiddleware interface {
	Handle(next http.Handler) http.Handler
}

// NoOpAuth passes requests through unchanged (used when no auth is provided).
// This guarantees the Auth layer is strictly nil-safe!
type NoOpAuth struct{}

func (n NoOpAuth) Handle(next http.Handler) http.Handler { return next }

// Server encapsulates the HTTP multiplexer and middleware dependencies.
type Server struct {
	mux            *http.ServeMux
	auth           AuthMiddleware
	rateLimiter    *IPRateLimiter
	logger         *logger.Logger
	sessionHandler *SessionHandler
	sseHandler     *SSEHandler
}

// NewServer bootstraps a fresh Server, its routes, and its shared state.
func NewServer(
	auth AuthMiddleware, 
	log *logger.Logger, 
	sessionHandler *SessionHandler,
	sseHandler *SSEHandler,
) *Server {
	if sessionHandler == nil {
		panic("sessionHandler cannot be nil")
	}
	if sseHandler == nil {
		panic("sseHandler cannot be nil")
	}

	if auth == nil {
		auth = NoOpAuth{}
	}

	s := &Server{
		mux:            http.NewServeMux(),
		auth:           auth,
		rateLimiter:    NewIPRateLimiter(rate.Limit(10), 20), // 10 req/s, burst 20
		logger:         log,
		sessionHandler: sessionHandler,
		sseHandler:     sseHandler,
	}

	s.routes()
	return s
}

// Close cleanly releases all acquired server resources (like background goroutines).
func (s *Server) Close() error {
	s.rateLimiter.Close()
	return nil
}

// Handler returns the fully equipped router wrapped in its middleware chain.
// Order: Logger -> CORS -> RateLimit -> Auth -> Router
func (s *Server) Handler() http.Handler {
	return s.Logger(s.CORS(s.RateLimit(s.auth.Handle(s.mux))))
}

// routes registers exactly 9 endpoints using native Go 1.22+ method patterns.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	s.mux.HandleFunc("POST /v1/sessions", s.sessionHandler.handleCreateSession)
	s.mux.HandleFunc("GET /v1/sessions", s.sessionHandler.handleListSessions)
	s.mux.HandleFunc("GET /v1/sessions/{id}", s.sessionHandler.handleGetSession)
	s.mux.HandleFunc("DELETE /v1/sessions/{id}", s.sessionHandler.handleDeleteSession)

	s.mux.HandleFunc("GET /v1/sessions/{id}/messages", s.sessionHandler.handleListMessages)
	s.mux.HandleFunc("POST /v1/sessions/{id}/messages", s.sessionHandler.handleCreateMessage)

	s.mux.HandleFunc("GET /v1/sessions/{id}/checkpoints", s.sessionHandler.handleListCheckpoints)
	s.mux.HandleFunc("GET /v1/sessions/{id}/events", s.sseHandler.handleSubscribeEvents)
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
