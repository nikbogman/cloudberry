// Package edge is the public entry point: it hosts the UI and forwards
// to the Waker API, the Sleeper API and Caddy-fronted workloads over the
// tailnet, so the browser only ever talks to one origin.
//
// NO AUTHENTICATION: anyone with the URL can wake, suspend and reach every
// workload. Deliberate and temporary -- auth will be added later.
package edge

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/nikbogman/homelab/waker-service/internal/tailnet"
)

type Config struct {
	WakerAPIURL   *url.URL
	SleeperAPIURL *url.URL
	// Caddy serves each stack as <stack>.<TailnetDomain>.
	TailnetDomain string
	UIAssets      fs.FS
}

// A stack name picks a tailnet host from public input, so it's held to a
// single DNS label.
var stackName = regexp.MustCompile(`^[a-z0-9-]+$`)

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

	stackProxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(&url.URL{Scheme: "https", Host: pr.In.PathValue("stack") + "." + cfg.TailnetDomain})
			pr.Out.URL.Path = "/" + pr.In.PathValue("rest")
			pr.Out.URL.RawPath = ""
			pr.SetXForwarded()
			pr.Out.Header.Del(tailnet.IdentityHeader)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("POST /api/waker/wake", http.StripPrefix("/api/waker", proxyTo(cfg.WakerAPIURL)))
	sleeper := http.StripPrefix("/api/sleeper", proxyTo(cfg.SleeperAPIURL))
	mux.Handle("GET /api/sleeper/health", sleeper)
	mux.Handle("POST /api/sleeper/suspend", sleeper)
	mux.HandleFunc("/proxy/{stack}/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		if !stackName.MatchString(r.PathValue("stack")) {
			http.Error(w, "invalid stack name", http.StatusBadRequest)
			return
		}
		stackProxy.ServeHTTP(w, r)
	})
	mux.Handle("/ui/", http.StripPrefix("/ui", http.FileServer(http.FS(cfg.UIAssets))))
	mux.Handle("GET /{$}", http.RedirectHandler("/ui/", http.StatusFound))
	return mux
}
