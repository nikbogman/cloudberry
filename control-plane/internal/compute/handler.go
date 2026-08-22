// Package compute runs on the compute host behind its own tailscale serve
// instance, a distinct origin from the UI -- hence the CORS allow-list.
package compute

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikbogman/homelab/control-plane/internal/httpresponse"
	"github.com/nikbogman/homelab/control-plane/internal/tailnet"
)

type Suspender interface {
	Suspend() error
}

type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

// RoutableContainer is a workload container discovered by its
// homelab.route label -- Path is the label value (e.g. "/immich"), Addr
// is where the proxy dials to reach it.
type RoutableContainer struct {
	Path string
	Addr string
}

// ContainerRuntime abstracts Docker daemon access: the current set of
// routable containers, sufficient to resolve a path to a proxy target, plus
// the idle watcher's container-activity signal.
type ContainerRuntime interface {
	RoutableContainers() ([]RoutableContainer, error)

	// ActivityAboveBaseline reports whether any workload container's
	// CPU/network usage is currently above the idle-noise baseline.
	// Container running/stopped state alone is never this signal --
	// workload containers run continuously via restart:unless-stopped
	// regardless of actual use.
	ActivityAboveBaseline() (bool, error)
}

// lastProxied tracks when a request was last successfully proxied to a
// workload container -- the activity signal the idle watcher will read.
// atomic.Int64 of UnixNano, not a mutex-guarded time.Time: net/http serves
// concurrently and this is written on every proxied request.
type lastProxied struct {
	unixNano atomic.Int64
}

func (l *lastProxied) record(t time.Time) {
	l.unixNano.Store(t.UnixNano())
}

// At returns the zero time if no request has ever been proxied.
func (l *lastProxied) At() time.Time {
	nano := l.unixNano.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// sync.Once, not a bool: net/http serves concurrently. Also the only
// reachability edge this app can observe -- it's asleep whenever Compute
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
	uiOrigin     string
	suspender    Suspender
	runtime      ContainerRuntime
	logger       EventLogger
	reachability reachabilityTracker
	lastProxied  lastProxied
	hold         hold
	serve        http.Handler
}

// NewHandler refuses to start if bindHost would expose the app
// off-tailnet. It returns *Handler rather than plain http.Handler so
// callers -- e.g. the future idle watcher -- can also reach methods like
// LastProxiedAt beyond just serving requests.
func NewHandler(uiOrigin string, suspender Suspender, runtime ContainerRuntime, logger EventLogger, bindHost string) (*Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(bindHost); err != nil {
		return nil, err
	}

	h := &Handler{uiOrigin: uiOrigin, suspender: suspender, runtime: runtime, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("GET /health", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleHealth)))
	mux.Handle("POST /suspend", tailnet.RequireTailnetIdentity(http.HandlerFunc(h.handleSuspend)))
	// Loopback exception: an unattended backup/restore script running
	// locally on Compute has no tailnet identity header of its own.
	mux.Handle("POST /hold", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleHoldAcquire)))
	mux.Handle("DELETE /hold", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleHoldRelease)))
	// Catch-all: routes /{prefix}/... to whichever container carries a
	// matching homelab.route label. Not identity-gated, mirroring the
	// Gateway proxy's /server/ route -- this carries workload traffic,
	// not a control-plane action.
	mux.Handle("/", http.HandlerFunc(h.handleProxy))
	h.serve = h.withCORS(mux)
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.serve.ServeHTTP(w, r)
}

// withCORS allows only h.uiOrigin. A mismatched Origin gets no CORS
// headers -- the browser blocks it, not the server. OPTIONS preflight is
// answered directly since /suspend triggers it.
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

func (h *Handler) handleHoldAcquire(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	h.hold.acquire(h.logger, identity)
	httpresponse.WriteJSON(w, http.StatusOK, map[string]string{"hold": "acquired"})
}

func (h *Handler) handleHoldRelease(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	h.hold.release(h.logger, identity)
	httpresponse.WriteJSON(w, http.StatusOK, map[string]string{"hold": "released"})
}

// LastProxiedAt is when a request was last successfully proxied to a
// workload container -- one of the idle watcher's activity signals. The
// zero time means no request has been proxied yet.
func (h *Handler) LastProxiedAt() time.Time {
	return h.lastProxied.At()
}

// HoldActive reports whether a Hold is currently in effect -- one of the
// idle watcher's activity signals.
func (h *Handler) HoldActive() bool {
	return h.hold.isActive()
}
