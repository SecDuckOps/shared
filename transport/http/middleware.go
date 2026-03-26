package transport

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SecDuckOps/shared/ports"
	"github.com/SecDuckOps/shared/types"
	"golang.org/x/time/rate"
)

// --- Logger Middleware ---

// responseWriter wraps http.ResponseWriter to capture the HTTP status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK) // explicitly default to 200 OK
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Logger logs the incoming request and its duration/status code using the shared logger.
func (s *Server) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		s.logger.Info(r.Context(), "HTTP Request",
			ports.Field{Key: "method", Value: r.Method},
			ports.Field{Key: "path", Value: r.URL.Path},
			ports.Field{Key: "status", Value: rw.status},
			ports.Field{Key: "duration_ms", Value: duration.Milliseconds()},
			ports.Field{Key: "ip", Value: getClientIP(r)},
		)
	})
}

// --- CORS Middleware ---

// CORS applies basic Cross-Origin Resource Sharing headers.
func (s *Server) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := localOrigin(r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func localOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	host := strings.Trim(strings.TrimSpace(parsed.Hostname()), "[]")
	if host == "" || host == "localhost" {
		return origin
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return origin
	}
	return ""
}

// --- Rate Limiter Middleware ---

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips       map[string]*visitor
	mu        sync.Mutex
	r         rate.Limit
	b         int
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewIPRateLimiter creates a stateful rate limiter and immediately starts EXACTLY ONE cleanup goroutine.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:    make(map[string]*visitor),
		r:      r,
		b:      b,
		stopCh: make(chan struct{}),
	}

	go limiter.cleanupLoop()

	return limiter
}

// Close gracefully stops the background cleanup goroutine safely.
func (i *IPRateLimiter) Close() {
	i.closeOnce.Do(func() {
		close(i.stopCh)
	})
}

// cleanupLoop removes stale IP entries to prevent memory leaks from map growth over time.
func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-i.stopCh:
			return
		case <-ticker.C:
			i.mu.Lock()
			for ip, v := range i.ips {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(i.ips, ip)
				}
			}
			i.mu.Unlock()
		}
	}
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	v, exists := i.ips[ip]
	if !exists {
		limiter := rate.NewLimiter(i.r, i.b)
		i.ips[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// RateLimit uses the server's IPRateLimiter state to restrict excessive requests per IP.
func (s *Server) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		limiter := s.rateLimiter.getLimiter(ip)

		if !limiter.Allow() {
			appErr := types.New(types.ErrCodeTooManyRequests, "Rate limit exceeded")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(appErr.ToMap())

			// Log explicitly since it blocked the request early
			s.logger.ErrorErr(r.Context(), appErr, "Rate limit exceeded for IP", ports.Field{Key: "ip", Value: ip})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts IP using proper split to handle IPv6 and explicit parts safely.
func getClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Fallback if no port is present or parsing fails
		return r.RemoteAddr
	}
	return ip
}
