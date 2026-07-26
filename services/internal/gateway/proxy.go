package gateway

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/nikbogman/homelab/services/internal/httpresponse"
	"github.com/nikbogman/homelab/services/internal/tailnet"
)

// Package vars, not consts, so tests can shrink them.
var (
	proxyRetryWindow   = 60 * time.Second
	proxyRetryInterval = 1 * time.Second
	autoWakeThrottle   = 90 * time.Second
)

// There is exactly one compute upstream in this service, so
// autoWakeThrottler needs only a single timestamp -- no per-target keying.
type autoWakeThrottler struct {
	doWake func(identity string) error
	now    func() time.Time

	mu      sync.Mutex
	lastRun time.Time
}

// trigger's own doWake error is intentionally ignored: the caller (a
// proxied request retry loop) must never fail because of a wake attempt's
// outcome -- it only cares that a wake was attempted at all before it
// starts retrying.
func (t *autoWakeThrottler) trigger() {
	t.mu.Lock()
	now := t.now()
	if !t.lastRun.IsZero() && now.Sub(t.lastRun) < autoWakeThrottle {
		t.mu.Unlock()
		return
	}
	t.lastRun = now
	t.mu.Unlock()

	_ = t.doWake(tailnet.LoopbackIdentity)
}

// newComputeProxy: a Go-level transport failure (dial refused, timeout)
// triggers the throttled auto-wake and retries the request until it
// succeeds or proxyRetryWindow elapses; a genuine HTTP response from a
// live backend (including a real 502) is passed straight through
// untouched.
func newComputeProxy(computeHost string, computeProxyPort int, doWake func(identity string) error) http.Handler {
	target := &url.URL{Scheme: "http", Host: net.JoinHostPort(computeHost, strconv.Itoa(computeProxyPort))}
	proxy := httputil.NewSingleHostReverseProxy(target)

	throttle := &autoWakeThrottler{doWake: doWake, now: time.Now}

	proxy.ErrorHandler = func(w http.ResponseWriter, outreq *http.Request, err error) {
		throttle.trigger()
		retryUntilReachable(w, outreq, proxy.Transport)
	}

	return http.StripPrefix("/server", proxy)
}

// retryUntilReachable re-issues outreq (already director-rewritten to the
// compute host by the failed ReverseProxy attempt) every proxyRetryInterval
// for up to proxyRetryWindow, writing the first successful response
// straight through to w. Buffers outreq's body once up front so each retry
// attempt can replay it.
func retryUntilReachable(w http.ResponseWriter, outreq *http.Request, transport http.RoundTripper) {
	if transport == nil {
		transport = http.DefaultTransport
	}

	var body []byte
	if outreq.Body != nil {
		body, _ = io.ReadAll(outreq.Body)
		outreq.Body.Close()
	}

	deadline := time.Now().Add(proxyRetryWindow)
	for {
		attempt := outreq.Clone(outreq.Context())
		if body != nil {
			attempt.Body = io.NopCloser(bytes.NewReader(body))
			attempt.ContentLength = int64(len(body))
		}

		res, err := transport.RoundTrip(attempt)
		if err == nil {
			httpresponse.Copy(w, res)
			return
		}

		if time.Now().After(deadline) {
			http.Error(w, "compute host unreachable", http.StatusBadGateway)
			return
		}
		time.Sleep(proxyRetryInterval)
	}
}
