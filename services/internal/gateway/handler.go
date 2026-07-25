// Package gateway implements the Gateway API: sends a Wake-on-LAN packet to
// the compute host on request.
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

// Handler serves the Gateway API's /wake action.
type Handler struct {
	macAddress string
	waker      Waker
	logger     EventLogger
}

// NewHandler builds the Gateway API's HTTP handler. It refuses to start if
// bindHost would expose the app off-tailnet, and fails fast on a malformed
// macAddress at construction rather than mid-request.
func NewHandler(macAddress string, waker Waker, logger EventLogger, bindHost string) (http.Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(bindHost); err != nil {
		return nil, err
	}
	if _, err := BuildMagicPacket(macAddress); err != nil {
		return nil, err
	}

	h := &Handler{macAddress: macAddress, waker: waker, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("POST /wake", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleWake)))
	return mux, nil
}

func (h *Handler) handleWake(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	h.logger.SendEvent("wake_requested", "requested", &identity, nil)

	if err := h.waker.Send(h.macAddress); err != nil {
		h.logger.SendEvent("wake_failed", "failed", &identity, nil)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"wake": "failed"})
		return
	}

	h.logger.SendEvent("wake_succeeded", "succeeded", &identity, nil)
	writeJSON(w, http.StatusOK, map[string]string{"wake": "succeeded"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
