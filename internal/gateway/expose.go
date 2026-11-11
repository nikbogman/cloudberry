package gateway

import (
	"fmt"
	"net/http"
	"path/filepath"

	"tailscale.com/tsnet"

	"github.com/nikbogman/cloudberry/internal/routes"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// Node is a tsnet node named name, with its state in stateDir/name. The first
// start needs TS_AUTHKEY.
// ponytail: one tsnet node (tens of MB) per route plus gateway's own; move to
// Tailscale Services if the Pi Zero's RAM ever runs short.
func Node(name, stateDir string) *tsnet.Server {
	return &tsnet.Server{Hostname: name, Dir: filepath.Join(stateDir, name)}
}

// ServeHostname blocks, serving a route at https://<route name>.<tailnet>.ts.net
// on 443 from its own tsnet node.
func ServeHostname(r routes.Route, stateDir string, next http.Handler) error {
	if err := ServeNode(Node(r.Name, stateDir), r.Public, next); err != nil {
		return fmt.Errorf("route %q: %w", r.Name, err)
	}
	return nil
}

// ServeNode blocks, serving next on s's port 443. Public nodes use Funnel and
// get no identity. The rest set the identity header from a WhoIs lookup, and
// reject a caller with no user profile: this is what `tailscale serve` did
// when it fronted gateway.
func ServeNode(s *tsnet.Server, public bool, next http.Handler) error {
	if public {
		ln, err := s.ListenFunnel("tcp", ":443")
		if err != nil {
			return err
		}
		return http.Serve(ln, next)
	}
	ln, err := s.ListenTLS("tcp", ":443")
	if err != nil {
		return err
	}
	lc, err := s.LocalClient()
	if err != nil {
		return err
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
