// Package edge is the public entry point: it hosts the UI and forwards
// to gateway and hostd over the tailnet, so the browser only ever talks to
// one origin.
//
// NO AUTHENTICATION: anyone with the URL can wake and suspend blackberry.
// Deliberate and temporary -- auth will be added later.
package edge

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/nikbogman/cloudberry/internal/tailnet"
)

type Config struct {
	GatewayURL *url.URL
	HostdURL   *url.URL
	UIAssets   fs.FS
}

// NewHandler proxies through transport, which in production dials over the
// tailnet.
func NewHandler(cfg Config, transport http.RoundTripper) http.Handler {
	proxyTo := func(target *url.URL) http.Handler {
		return &httputil.ReverseProxy{
			Transport: transport,
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.SetXForwarded()
				// tailnet.IdentityHeader is set by `tailscale serve` on the far
				// side from Edge's own node -- never trust a caller's copy.
				pr.Out.Header.Del(tailnet.IdentityHeader)
			},
		}
	}

	mux := http.NewServeMux()
	mux.Handle("POST /api/gateway/wake", http.StripPrefix("/api/gateway", proxyTo(cfg.GatewayURL)))
	hostd := http.StripPrefix("/api/hostd", proxyTo(cfg.HostdURL))
	mux.Handle("GET /api/hostd/health", hostd)
	mux.Handle("POST /api/hostd/suspend", hostd)
	mux.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.FS(cfg.UIAssets))))
	mux.Handle("GET /{$}", http.RedirectHandler("/ui/", http.StatusFound))
	return mux
}
