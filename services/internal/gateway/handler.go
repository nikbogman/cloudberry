// Package gateway implements the Gateway: serves the UI's static files,
// sends a Wake-on-LAN packet to the compute host on request, and reverse
// proxies to the compute host, auto-waking it on a transport failure.
//
// Runs on the Pi Zero, path-routed under the same tailscale serve app as
// the UI — same-origin, so no CORS entry is needed.
package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/nikbogman/homelab/services/internal/tailnet"
)

// Waker sends a Wake-on-LAN packet to macAddress.
type Waker interface {
	Send(macAddress string) error
}

// EventLogger ships a structured audit event.
type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

// Config configures a Handler.
type Config struct {
	MACAddress string
	BindHost   string
	// UIRoot is the directory served at "/". Passed in rather than a
	// package-level const so tests can point it at a t.TempDir().
	UIRoot           string
	ComputeHost      string
	ComputeProxyPort int
}

// Handler serves the Gateway's /wake action, the /server* proxy to the
// compute host, and the UI's static files.
type Handler struct {
	macAddress string
	waker      Waker
	logger     EventLogger
}

// NewHandler builds the Gateway's HTTP handler. It refuses to start if
// cfg.BindHost would expose the app off-tailnet, and fails fast on a
// malformed MAC address at construction rather than mid-request.
func NewHandler(cfg Config, waker Waker, logger EventLogger) (http.Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(cfg.BindHost); err != nil {
		return nil, err
	}
	if _, err := BuildMagicPacket(cfg.MACAddress); err != nil {
		return nil, err
	}

	h := &Handler{macAddress: cfg.MACAddress, waker: waker, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("POST /wake", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleWake)))
	mux.Handle("/server/", newComputeProxy(cfg.ComputeHost, cfg.ComputeProxyPort, h))
	mux.Handle("/", http.FileServer(http.Dir(cfg.UIRoot)))
	return mux, nil
}

func (h *Handler) handleWake(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	if err := h.doWake(identity); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"wake": "failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"wake": "succeeded"})
}

// doWake sends a Wake-on-LAN packet and logs the requested/succeeded/failed
// event sequence, attributing it to identity. Shared by the HTTP-facing
// /wake handler and the proxy's in-process auto-wake trigger.
func (h *Handler) doWake(identity string) error {
	h.logger.SendEvent("wake_requested", "requested", &identity, nil)

	if err := h.waker.Send(h.macAddress); err != nil {
		h.logger.SendEvent("wake_failed", "failed", &identity, nil)
		return err
	}

	h.logger.SendEvent("wake_succeeded", "succeeded", &identity, nil)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
