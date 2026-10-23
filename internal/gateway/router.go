package gateway

import (
	"context"
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
	hostdURL    string
	mac         string
	sender      Sender
	logger      EventLogger
	client      *http.Client
	wakeTimeout time.Duration
	pollEvery   time.Duration

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

// NewRouter's hostdURL is hostd's tailnet URL; raspberry's own tailscale
// identity is what hostd sees on /health and /suspend.
func NewRouter(hostdURL, mac string, sender Sender, logger EventLogger) *Router {
	g := &Router{
		hostdURL: hostdURL, mac: mac, sender: sender, logger: logger,
		client:      &http.Client{},
		wakeTimeout: 90 * time.Second,
		pollEvery:   2 * time.Second,
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
			if err := g.awake(req.Context()); err != nil {
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

func (g *Router) reachable(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.hostdURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// awake returns once blackberry is reachable, sending WoL if it isn't.
// Concurrent callers share one attempt, so a burst sends one packet.
func (g *Router) awake(ctx context.Context) error {
	if g.reachable(ctx) {
		return nil
	}
	g.mu.Lock()
	a := g.waking
	if a == nil {
		a = &wakeAttempt{done: make(chan struct{})}
		g.waking = a
		go g.wake(a)
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
func (g *Router) wake(a *wakeAttempt) {
	defer func() {
		g.mu.Lock()
		g.waking = nil
		g.mu.Unlock()
		close(a.done)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), g.wakeTimeout)
	defer cancel()

	g.logger.SendEvent("autowake_requested", "requested", nil, nil)
	if a.err = g.sender.Send(g.mac); a.err != nil {
		g.logger.SendEvent("autowake_failed", "failed", nil, nil)
		return
	}
	for !g.reachable(ctx) {
		select {
		case <-ctx.Done():
			a.err = ctx.Err()
			g.logger.SendEvent("autowake_failed", "timeout", nil, nil)
			return
		case <-time.After(g.pollEvery):
		}
	}
	g.autoWoken.Store(true)
	g.logger.SendEvent("autowake_succeeded", "succeeded", nil, nil)
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
