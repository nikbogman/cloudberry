package edge

import (
	"net/http"
	"strconv"
	"time"
)

type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

// WithAccessLog ships one "request" event per request, outcome being the
// status class (2xx, 4xx, ...). Sent off the request path so a slow Loki
// never delays a response.
// ponytail: one goroutine + POST per request, batch if traffic grows.
func WithAccessLog(next http.Handler, logger EventLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Railway's proxy sets X-Forwarded-For; RemoteAddr is the proxy.
		client := r.Header.Get("X-Forwarded-For")
		if client == "" {
			client = r.RemoteAddr
		}
		go logger.SendEvent("request", strconv.Itoa(rec.status/100)+"xx", nil, map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      rec.status,
			"duration_ms": time.Since(start).Milliseconds(),
			"client":      client,
		})
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController (used by ReverseProxy to flush)
// reach the real writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
