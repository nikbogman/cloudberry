// Package compute implements the Compute API: exposes a Reachable health
// check and a Suspend action.
//
// Runs on the compute host behind its own tailscale serve instance, a
// distinct origin from the UI — hence the CORS allow-list.
package compute

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/nikbogman/homelab/services/internal/tailnet"
)

// Suspender suspends the host to RAM.
type Suspender interface {
	Suspend() error
}

// EventLogger ships a structured audit event.
type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

// reachabilityTracker logs a reachability_changed event the first time this
// process observes itself as reachable. sync.Once (rather than a plain
// bool) matters here: net/http serves requests concurrently by default,
// unlike Flask's dev server, so "log exactly once" needs real
// synchronization. It's the only "changed" edge this app can ever witness:
// it can't log going *un*reachable, since it's asleep while that's true.
type reachabilityTracker struct {
	once sync.Once
}

func (t *reachabilityTracker) noteReachable(logger EventLogger) {
	t.once.Do(func() {
		logger.SendEvent("reachability_changed", "reachable", nil, nil)
	})
}

// Handler serves the Compute API's /health and /suspend actions.
type Handler struct {
	uiOrigin     string
	suspender    Suspender
	logger       EventLogger
	reachability reachabilityTracker
}

// NewHandler builds the Compute API's HTTP handler. It refuses to start if
// bindHost would expose the app off-tailnet.
func NewHandler(uiOrigin string, suspender Suspender, logger EventLogger, bindHost string) (http.Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(bindHost); err != nil {
		return nil, err
	}

	h := &Handler{uiOrigin: uiOrigin, suspender: suspender, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("GET /health", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleHealth)))
	mux.Handle("POST /suspend", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleSuspend)))
	return h.withCORS(mux), nil
}

// withCORS allows only h.uiOrigin. A mismatched Origin gets no CORS headers
// at all — the request still proceeds server-side; the browser is what
// blocks it. OPTIONS preflight requests are answered directly, since
// real cross-origin POSTs (e.g. /suspend) trigger them.
func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && origin == h.uiOrigin
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if r.Method == http.MethodOptions {
			if allowed {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
				w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.reachability.noteReachable(h.logger)
	writeJSON(w, http.StatusOK, map[string]bool{"reachable": true})
}

func (h *Handler) handleSuspend(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	h.logger.SendEvent("suspend_requested", "requested", &identity, nil)

	if err := h.suspender.Suspend(); err != nil {
		h.logger.SendEvent("suspend_failed", "failed", &identity, nil)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"suspend": "failed"})
		return
	}

	h.logger.SendEvent("suspend_succeeded", "succeeded", &identity, nil)
	writeJSON(w, http.StatusOK, map[string]string{"suspend": "succeeded"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
