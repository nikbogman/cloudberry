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

	"github.com/nikbogman/homelab/control-plane/internal/httpresponse"
	"github.com/nikbogman/homelab/control-plane/internal/tailnet"
)

// Package vars, not consts, so tests can shrink them.
var (
	proxyRetryWindow   = 60 * time.Second
	proxyRetryInterval = 1 * time.Second
	autoWakeThrottle   = 90 * time.Second
)

// Single timestamp: exactly one compute upstream exists.
type autoWakeThrottler struct {
	doWake func(identity string) error
	now    func() time.Time

	mu      sync.Mutex
	lastRun time.Time
}

// Error discarded -- callers only need a wake attempted, not confirmed.
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

// newComputeProxy reverse-proxies to the compute host. A transport failure
// triggers throttled auto-wake and retries until proxyRetryWindow elapses;
// a real HTTP response (even a 502) passes through untouched.
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

// retryUntilReachable retries outreq every proxyRetryInterval up to
// proxyRetryWindow, writing the first success through to w. The body is
// buffered once so each retry can replay it.
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
