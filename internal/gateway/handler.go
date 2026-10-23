// Package gateway is raspberry's one service. It sends Wake-on-LAN, either for
// the UI's Wake (edge forwards POST /wake over the tailnet) or on a request to
// a stack route that asks for it, and reverse-proxies those routes. It's on
// raspberry because the magic packet is a LAN broadcast.
package gateway

import (
	"net/http"

	"github.com/nikbogman/cloudberry/internal/httpresponse"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

type Sender interface {
	Send(macAddress string) error
}

type EventLogger interface {
	SendEvent(eventType, outcome string, identity *string, extra map[string]any)
}

type Config struct {
	MACAddress string
	BindHost   string
}

type Handler struct {
	macAddress string
	sender     Sender
	logger     EventLogger
}

func NewHandler(cfg Config, sender Sender, logger EventLogger) (http.Handler, error) {
	if err := tailnet.AssertTailnetOnlyBind(cfg.BindHost); err != nil {
		return nil, err
	}
	// Fail fast at startup rather than on the first /wake request.
	if _, err := BuildMagicPacket(cfg.MACAddress); err != nil {
		return nil, err
	}

	h := &Handler{macAddress: cfg.MACAddress, sender: sender, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("POST /wake", tailnet.RequireTailnetIdentityOrLoopback(http.HandlerFunc(h.handleWake)))
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

func (h *Handler) doWake(identity string) error {
	// Packet first: SendEvent is a synchronous call to Grafana, which on the
	// Pi Zero would delay the wake.
	err := h.sender.Send(h.macAddress)
	h.logger.SendEvent("wake_requested", "requested", &identity, nil)
	if err != nil {
		h.logger.SendEvent("wake_failed", "failed", &identity, nil)
		return err
	}

	h.logger.SendEvent("wake_succeeded", "succeeded", &identity, nil)
	return nil
}
