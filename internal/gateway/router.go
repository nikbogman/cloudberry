package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// Router fronts the stacks in the routes file: it wakes blackberry on request
// for routes that ask for it, holds the request until blackberry answers, then
// reverse-proxies. It also suspends blackberry again once it has been idle,
// but only if the Router woke it.
type Router struct {
	hostdURL     string
	mac          string
	sender       Sender
	logger       EventLogger
	client       *http.Client
	wakeTimeout  time.Duration
	pollEvery    time.Duration
	probeTimeout time.Duration

	inflight  atomic.Int64
	lastUse   atomic.Int64 // unix nanos of the last request start or finish
	autoWoken atomic.Bool  // only an auto-wake is ever auto-suspended

	mu     sync.Mutex
	waking *wakeAttempt
}

type wakeAttempt struct {
	done chan struct{}
	err  error
}

// NewRouter's hostdURL is hostd's tailnet URL, used only for /suspend;
// raspberry's own tailscale identity is what hostd sees.
func NewRouter(hostdURL, mac string, sender Sender, logger EventLogger) *Router {
	g := &Router{
		hostdURL: hostdURL, mac: mac, sender: sender, logger: logger,
		client:       &http.Client{},
		wakeTimeout:  90 * time.Second,
		pollEvery:    250 * time.Millisecond,
		probeTimeout: 300 * time.Millisecond,
	}
	g.touch()
	return g
}

func (g *Router) touch() { g.lastUse.Store(time.Now().UnixNano()) }

// RouteHandler holds a Wake route's request until blackberry is reachable.
func (g *Router) RouteHandler(r Route) (http.Handler, error) {
	u, err := url.Parse(r.Upstream)
	if err != nil {
		return nil, err
	}
	target := upstreamAddr(u)
	proxy := httputil.NewSingleHostReverseProxy(u)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = u.Host
		req.Header.Del(tailnet.IdentityHeader) // upstreams never trust it
	}

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		g.inflight.Add(1)
		defer func() { g.touch(); g.inflight.Add(-1) }()
		g.touch()
		if r.Wake {
			if err := g.awake(req.Context(), target); err != nil {
				w.Header().Set("Retry-After", "30")
				http.Error(w, "blackberry is waking, retry shortly", http.StatusServiceUnavailable)
				return
			}
		}
		proxy.ServeHTTP(w, req)
	})
	// Public routes sit on Funnel; the rest are only ever reached through
	// `tailscale serve`, which injects the identity header.
	if !r.Public {
		h = tailnet.RequireTailnetIdentity(h)
	}
	return h, nil
}

// upstreamAddr is the host:port a route proxies to, defaulting the port by
// scheme.
func upstreamAddr(u *url.URL) string {
	if u.Port() != "" {
		return u.Host
	}
	if u.Scheme == "https" {
		return net.JoinHostPort(u.Hostname(), "443")
	}
	return net.JoinHostPort(u.Hostname(), "80")
}

// up reports whether the stack accepts TCP connections. Dialing the upstream
// on the LAN, not hostd over the tailnet, avoids waiting for TLS on the Pi and
// for tailscaled to reconnect after resume, and proves the stack itself is up.
func (g *Router) up(ctx context.Context, addr string) bool {
	ctx, cancel := context.WithTimeout(ctx, g.probeTimeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// awake returns once addr answers, sending WoL if it doesn't. Concurrent
// callers share one attempt, so a burst sends one packet. Every upstream is
// on blackberry, so the attempt polls whichever addr started it.
func (g *Router) awake(ctx context.Context, addr string) error {
	if g.up(ctx, addr) {
		return nil
	}
	g.mu.Lock()
	a := g.waking
	if a == nil {
		a = &wakeAttempt{done: make(chan struct{})}
		g.waking = a
		go g.wake(a, addr)
	}
	g.mu.Unlock()

	select {
	case <-a.done:
		return a.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// wake sends one packet. ponytail: no resend, so a lost packet costs a full
// timeout; resend every ~10s if that ever shows up in the logs.
//
// Events ship in the background: SendEvent is a synchronous HTTPS call to
// Grafana, which on the Pi Zero would delay the packet and the held requests.
func (g *Router) wake(a *wakeAttempt, addr string) {
	defer func() {
		g.mu.Lock()
		g.waking = nil
		g.mu.Unlock()
		close(a.done)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), g.wakeTimeout)
	defer cancel()

	go g.logger.SendEvent("autowake_requested", "requested", nil, nil)
	if a.err = g.sender.Send(g.mac); a.err != nil {
		go g.logger.SendEvent("autowake_failed", "failed", nil, nil)
		return
	}
	for !g.up(ctx, addr) {
		select {
		case <-ctx.Done():
			a.err = ctx.Err()
			go g.logger.SendEvent("autowake_failed", "timeout", nil, nil)
			return
		case <-time.After(g.pollEvery):
		}
	}
	g.autoWoken.Store(true)
	go g.logger.SendEvent("autowake_succeeded", "succeeded", nil, nil)
}

// RunIdleSuspend suspends blackberry once no route has had traffic for idle,
// if the Router woke it. A manual Wake is never undone.
func (g *Router) RunIdleSuspend(ctx context.Context, idle time.Duration) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			g.suspendIfIdle(ctx, idle)
		}
	}
}

func (g *Router) suspendIfIdle(ctx context.Context, idle time.Duration) {
	if !g.autoWoken.Load() || g.inflight.Load() > 0 ||
		time.Since(time.Unix(0, g.lastUse.Load())) < idle {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.hostdURL+"/suspend", nil)
	if err != nil {
		return
	}
	resp, err := g.client.Do(req)
	if err != nil {
		g.logger.SendEvent("autosuspend_failed", "failed", nil, nil)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		g.logger.SendEvent("autosuspend_failed", "failed", nil, nil)
		return
	}
	g.autoWoken.Store(false)
	g.logger.SendEvent("autosuspend_succeeded", "succeeded", nil, nil)
}
