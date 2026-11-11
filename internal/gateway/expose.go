package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tailscale.com/tsnet"

	"github.com/nikbogman/cloudberry/internal/routes"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// Expose publishes path routes under 443 with `tailscale serve`, forwarding to
// each one's loopback port, and unpublishes any other path on 443 but /
// (gateway's own /wake), left by a route that's gone. Hostname routes use
// ServeHostname instead.
func Expose(rs []routes.Route) error {
	want := map[string]bool{}
	for _, r := range rs {
		if r.Path == "" {
			continue
		}
		want[r.Path] = true
		args := []string{"serve", "--bg", "--https=443", "--set-path=" + r.Path, "localhost:" + strconv.Itoa(r.Port)}
		if out, err := exec.Command("tailscale", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("tailscale serve for route %q: %w: %s", r.Name, err, out)
		}
	}
	out, err := exec.Command("tailscale", "serve", "status", "--json").Output()
	if err != nil {
		return fmt.Errorf("tailscale serve status: %w", err)
	}
	for _, p := range stalePaths(out, want) {
		if out, err := exec.Command("tailscale", "serve", "--https=443", "--set-path="+p, "off").CombinedOutput(); err != nil {
			return fmt.Errorf("tailscale serve off for stale path %s: %w: %s", p, err, out)
		}
	}
	return nil
}

// stalePaths lists the paths published on 443 in `tailscale serve status
// --json` that aren't / or in want.
func stalePaths(status []byte, want map[string]bool) []string {
	var st struct {
		Web map[string]struct {
			Handlers map[string]json.RawMessage
		}
	}
	json.Unmarshal(status, &st) // no serve config yet: nothing is stale
	var stale []string
	for hostPort, web := range st.Web {
		if !strings.HasSuffix(hostPort, ":443") {
			continue
		}
		for p := range web.Handlers {
			if p != "/" && !want[p] {
				stale = append(stale, p)
			}
		}
	}
	return stale
}

// ServeHostname blocks, serving next at https://<route name>.<tailnet>.ts.net
// on 443 from an in-process tsnet node. Public routes use Funnel. The rest set
// the identity header from a WhoIs lookup, since no `tailscale serve` is in
// front. State lives in stateDir/<name>; the first start needs TS_AUTHKEY.
// ponytail: one tsnet node (tens of MB) per route; move to Tailscale Services
// if the Pi Zero's RAM ever runs short.
func ServeHostname(r routes.Route, stateDir string, next http.Handler) error {
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
