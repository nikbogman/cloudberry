// Package sleeper runs on the sleeper host behind its own tailscale serve
// instance. Only Edge calls it, server-side, so it needs no CORS.
package sleeper

import (
	"net/http"
	"sync"

	"github.com/nikbogman/homelab/platform/internal/httpresponse"
	"github.com/nikbogman/homelab/platform/internal/tailnet"
)

type Suspender interface {
	Suspend() error
}

type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

// sync.Once, not a bool: net/http serves concurrently. Also the only
// reachability edge this app can observe -- it's asleep whenever Sleeper
// goes unreachable, so it can never log that transition.
type reachabilityTracker struct {
	once sync.Once
}

func (t *reachabilityTracker) noteReachable(logger EventLogger) {
	t.once.Do(func() {
		logger.SendEvent("reachability_changed", "reachable", nil, nil)
	})
}

type Handler struct {
	suspender    Suspender
	logger       EventLogger
	reachability reachabilityTracker
	mux          *http.ServeMux
}

// NewHandler refuses to start if bindHost would expose the app off-tailnet.
func NewHandler(suspender Suspender, logger EventLogger, bindHost string) (*Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(bindHost); err != nil {
		return nil, err
	}

	h := &Handler{suspender: suspender, logger: logger, mux: http.NewServeMux()}

	h.mux.Handle("GET /health", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleHealth)))
	h.mux.Handle("POST /suspend", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleSuspend)))
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.reachability.noteReachable(h.logger)
	httpresponse.WriteJSON(w, http.StatusOK, map[string]bool{"reachable": true})
}

func (h *Handler) handleSuspend(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	h.logger.SendEvent("suspend_requested", "requested", &identity, nil)

	if err := h.suspender.Suspend(); err != nil {
		h.logger.SendEvent("suspend_failed", "failed", &identity, nil)
		httpresponse.WriteJSON(w, http.StatusInternalServerError, map[string]string{"suspend": "failed"})
		return
	}

	h.logger.SendEvent("suspend_succeeded", "succeeded", &identity, nil)
	httpresponse.WriteJSON(w, http.StatusOK, map[string]string{"suspend": "succeeded"})
}
