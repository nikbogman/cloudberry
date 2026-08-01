// Package tailnet holds tailnet identity-header auth and bind-safety
// helpers.
//
// Both apps sit behind `tailscale serve`, which injects the
// Tailscale-User-Login header for any tailnet-authenticated caller. These
// helpers only check the header is present, not which identity it names.
package tailnet

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const (
	IdentityHeader = "Tailscale-User-Login"
	// LoopbackIdentity is the synthetic identity assigned to a caller on
	// 127.0.0.1 with no identity header (ADR-0004, ADR-0012) -- the
	// Gateway API's own same-device auto-wake trigger. Kept as the literal
	// value "gateway" for continuity with existing Grafana Cloud event
	// history (ADR-0016), even though the separate Caddy-based Gateway
	// proxy process that name originally referred to is gone.
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

// RequireTailnetIdentityOrLoopback is like RequireTailnetIdentity, but also
// accepts a caller on 127.0.0.1 with no identity header (ADR-0004,
// ADR-0012) -- for the Gateway API's own same-device auto-wake trigger. Opt
// in per route; doesn't change RequireTailnetIdentity itself.
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
