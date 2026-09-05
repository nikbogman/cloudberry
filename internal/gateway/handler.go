// Package gateway is path-routed under the same tailscale serve app as
// the UI, so no CORS entry is needed.
package gateway

import (
	"io/fs"
	"net/http"

	"github.com/nikbogman/homelab/internal/httpresponse"
	"github.com/nikbogman/homelab/internal/tailnet"
)

type Waker interface {
	Send(macAddress string) error
}

type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

type Config struct {
	MACAddress string
	BindHost   string
	// Served at "/": the real binary passes an embed.FS via fs.Sub, tests
	// pass os.DirFS.
	UIAssets         fs.FS
	ComputeHost      string
	ComputeProxyPort int
}

type Handler struct {
	macAddress string
	waker      Waker
	logger     EventLogger
}

func NewHandler(cfg Config, waker Waker, logger EventLogger) (http.Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(cfg.BindHost); err != nil {
		return nil, err
	}
	// Fail fast at startup rather than on the first /wake request.
	if _, err := BuildMagicPacket(cfg.MACAddress); err != nil {
		return nil, err
	}

	h := &Handler{macAddress: cfg.MACAddress, waker: waker, logger: logger}

	mux := http.NewServeMux()
	// Lets the compute proxy trigger auto-wake in-process without a
	// tailnet identity header.
	mux.Handle("POST /wake", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleWake)))
	mux.Handle("/server/", newComputeProxy(cfg.ComputeHost, cfg.ComputeProxyPort, h.doWake))
	mux.Handle("/", http.FileServer(http.FS(cfg.UIAssets)))
	return mux, nil
}

func (h *Handler) handleWake(w http.ResponseWriter, r *http.Request) {
	identity := tailnet.GetCallerIdentity(r)
	if err := h.doWake(identity); err != nil {
		httpresponse.WriteJSON(w, http.StatusInternalServerError, map[string]string{"wake": "failed"})
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, map[string]string{"wake": "succeeded"})
}

// doWake is shared by the HTTP-facing /wake handler and the proxy's
// in-process auto-wake trigger.
func (h *Handler) doWake(identity string) error {
	h.logger.SendEvent("wake_requested", "requested", &identity, nil)

	if err := h.waker.Send(h.macAddress); err != nil {
		h.logger.SendEvent("wake_failed", "failed", &identity, nil)
		return err
	}

	h.logger.SendEvent("wake_succeeded", "succeeded", &identity, nil)
	return nil
}
