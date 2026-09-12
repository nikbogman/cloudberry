// Package tailnet holds tailnet identity-header auth and bind-safety
// helpers. `tailscale serve` injects Tailscale-User-Login for any
// tailnet-authenticated caller; these helpers only check it's present.
package tailnet

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const (
	IdentityHeader = "Tailscale-User-Login"
	// LoopbackIdentity is assigned to a caller on 127.0.0.1 with no
	// identity header -- a script or shell on the device itself. Kept as
	// "gateway" for continuity with existing event history.
	LoopbackIdentity = "gateway"
)

type contextKey int

const identityContextKey contextKey = iota

// RequireTailnetIdentity rejects any request missing the Tailscale identity
// header with a 401, and otherwise makes it readable via GetCallerIdentity.
func RequireTailnetIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := strings.TrimSpace(r.Header.Get(IdentityHeader))
		if identity == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityContextKey, identity)))
	})
}

// RequireTailnetIdentityOrLoopback also accepts a caller on 127.0.0.1 with
// no identity header, for local scripts with no browser session to carry
// an identity.
func RequireTailnetIdentityOrLoopback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := strings.TrimSpace(r.Header.Get(IdentityHeader))
		if identity != "" {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityContextKey, identity)))
			return
		}
		if remoteHost(r) == "127.0.0.1" {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityContextKey, LoopbackIdentity)))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

// GetCallerIdentity returns the caller's identity captured by
// RequireTailnetIdentity or RequireTailnetIdentityOrLoopback.
func GetCallerIdentity(r *http.Request) string {
	identity, _ := r.Context().Value(identityContextKey).(string)
	return identity
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
