package wakeplugin

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const directiveName = "call_wake_api"

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective(directiveName, parseCaddyfile)
}

// Handler is the call_wake_api Caddyfile directive: on invocation it
// triggers the Gateway API's wake action and always continues the
// handler chain, regardless of that call's outcome (ADR-0012).
type Handler struct {
	WakeURL string `json:"wake_url,omitempty"`

	caller *wakeCaller
}

func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers." + directiveName,
		New: func() caddy.Module { return new(Handler) },
	}
}

func (h *Handler) Provision(_ caddy.Context) error {
	if h.WakeURL == "" {
		return fmt.Errorf("%s: wake URL is required", directiveName)
	}
	h.caller = newWakeCaller()
	return nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	h.caller.Call(h.WakeURL)
	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile parses: call_wake_api <url>
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	if !d.NextArg() {
		return d.ArgErr()
	}
	h.WakeURL = d.Val()
	if d.NextArg() {
		return d.ArgErr()
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Handler
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

var (
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)
