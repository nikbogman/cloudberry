package gateway

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"

	"tailscale.com/tsnet"

	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// Expose publishes path routes under 443 with `tailscale serve`, forwarding to
// each one's loopback port. Hostname routes use ServeHostname instead.
// ponytail: only adds; a path route removed from the file stays published until
// `tailscale serve reset` on raspberry.
func Expose(routes []Route) error {
	for _, r := range routes {
		if r.Path == "" {
			continue
		}
		args := []string{"serve", "--bg", "--https=443", "--set-path=" + r.Path, "localhost:" + strconv.Itoa(r.Port)}
		if out, err := exec.Command("tailscale", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("tailscale serve for route %q: %w: %s", r.Name, err, out)
		}
	}
	return nil
}

// ServeHostname blocks, serving next at https://<route name>.<tailnet>.ts.net
// on 443 from an in-process tsnet node. Public routes use Funnel. The rest set
// the identity header from a WhoIs lookup, since no `tailscale serve` is in
// front. State lives in stateDir/<name>; the first start needs TS_AUTHKEY.
// ponytail: one tsnet node (tens of MB) per route; move to Tailscale Services
// if the Pi Zero's RAM ever runs short.
func ServeHostname(r Route, stateDir string, next http.Handler) error {
	s := &tsnet.Server{Hostname: r.Name, Dir: filepath.Join(stateDir, r.Name)}
	if r.Public {
		ln, err := s.ListenFunnel("tcp", ":443")
		if err != nil {
			return fmt.Errorf("route %q: %w", r.Name, err)
		}
		return http.Serve(ln, next)
	}
	ln, err := s.ListenTLS("tcp", ":443")
	if err != nil {
		return fmt.Errorf("route %q: %w", r.Name, err)
	}
	lc, err := s.LocalClient()
	if err != nil {
		return fmt.Errorf("route %q: %w", r.Name, err)
	}
	return http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Header.Del(tailnet.IdentityHeader) // never trust a client-supplied one
		who, err := lc.WhoIs(req.Context(), req.RemoteAddr)
		if err != nil || who.UserProfile == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		req.Header.Set(tailnet.IdentityHeader, who.UserProfile.LoginName)
		next.ServeHTTP(w, req)
	}))
}
