package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nikbogman/cloudberry/internal/httpresponse"
	"github.com/nikbogman/cloudberry/internal/routes"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// Router fronts the stacks in the routes file: it wakes blackberry on request
// for routes that ask for it, holds the request until blackberry answers, then
// reverse-proxies. It also suspends blackberry again once it has been idle,
// but only if the Router woke it. Both are switchable at runtime (SettingsHandler).
type Router struct {
	hostdURL     string
	mac          string
	sender       Sender
	logger       EventLogger
	client       *http.Client
	wakeTimeout  time.Duration
	pollEvery    time.Duration
	probeTimeout time.Duration

	inflight   atomic.Int64
	idleAt     atomic.Int64 // unix nanos: latest last-use + that route's idle time
	idleWindow atomic.Int64 // the idle time that set idleAt, for the UI's countdown
	autoWoken  atomic.Bool  // only an auto-wake is ever auto-suspended

	autoWake    atomic.Bool
	autoSuspend atomic.Bool

	// stateFile keeps autoWoken and the two switches across restarts, which
	// self-updates make routine: without it, a restart after an auto-wake
	// would leave blackberry up for good. Empty: in memory only.
	stateFile string
	saveMu    sync.Mutex

	mu     sync.Mutex
	waking *wakeAttempt

	// GET /status (status.go).
	statusMu   sync.Mutex
	host       hostState
	pending    pendingWake
	routeStats []*routeStat
	kick       chan struct{}
}

type routerState struct {
	AutoWoken   bool      `json:"auto_woken"`
	AutoWake    bool      `json:"wake"`
	AutoSuspend bool      `json:"suspend"`
	Host        hostState `json:"host"`
}

// LoadState restores the switches and autoWoken from path, and saves every
// later change there. A missing file keeps the defaults.
func (g *Router) LoadState(path string) error {
	g.stateFile = path
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	st := routerState{AutoWake: true, AutoSuspend: true}
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	g.autoWoken.Store(st.AutoWoken)
	g.autoWake.Store(st.AutoWake)
	g.autoSuspend.Store(st.AutoSuspend)
	g.statusMu.Lock()
	g.host = st.Host
	g.statusMu.Unlock()
	return nil
}

func (g *Router) save() {
	if g.stateFile == "" {
		return
	}
	g.saveMu.Lock()
	defer g.saveMu.Unlock()
	g.statusMu.Lock()
	host := g.host
	g.statusMu.Unlock()
	data, _ := json.Marshal(routerState{AutoWoken: g.autoWoken.Load(), AutoWake: g.autoWake.Load(), AutoSuspend: g.autoSuspend.Load(), Host: host})
	tmp := g.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err == nil {
		err = os.Rename(tmp, g.stateFile)
	}
}

// Idle reports whether no request is held or proxied and no wake is running,
// so a restart drops nothing.
func (g *Router) Idle() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.inflight.Load() == 0 && g.waking == nil
}

type wakeAttempt struct {
	done chan struct{}
	err  error
}

// NewRouter's hostdURL is hostd's tailnet URL, for /suspend. SetHTTPClient
// points the Router at gateway's own tsnet node, whose identity is what hostd sees.
func NewRouter(hostdURL, mac string, sender Sender, logger EventLogger) *Router {
	g := &Router{
		hostdURL: hostdURL, mac: mac, sender: sender, logger: logger,
		client:       &http.Client{},
		wakeTimeout:  90 * time.Second,
		pollEvery:    250 * time.Millisecond,
		probeTimeout: 300 * time.Millisecond,
		kick:         make(chan struct{}, 1),
	}
	g.idleAt.Store(time.Now().UnixNano())
	g.autoWake.Store(true)
	g.autoSuspend.Store(true)
	return g
}

// SetHTTPClient sets the client for hostd calls: gateway's tsnet node's,
// since the Pi runs no tailscaled to route 100.x addresses.
func (g *Router) SetHTTPClient(c *http.Client) { g.client = c }

// touch pushes the idle deadline to now+idle, never pulls it in: a short-idle
// route used after a long-idle one doesn't cut the long one's time short.
func (g *Router) touch(idle time.Duration) {
	until := time.Now().Add(idle).UnixNano()
	for {
		cur := g.idleAt.Load()
		if cur >= until {
			return
		}
		if g.idleAt.CompareAndSwap(cur, until) {
			g.idleWindow.Store(int64(idle))
			return
		}
	}
}

// RouteHandler holds a Wake route's request until blackberry is reachable.
func (g *Router) RouteHandler(r routes.Route) (http.Handler, error) {
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

	stat := &routeStat{route: r}
	g.statusMu.Lock()
	g.routeStats = append(g.routeStats, stat)
	g.statusMu.Unlock()

	idle := time.Duration(r.IdleMinutes) * time.Minute
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		g.inflight.Add(1)
		defer func() { g.touch(idle); g.inflight.Add(-1) }()
		g.touch(idle)
		stat.last.Store(time.Now().UnixNano())
		if r.Wake && g.autoWake.Load() {
			if err := g.awake(req.Context(), target, r.Name); err != nil {
				w.Header().Set("Retry-After", "30")
				http.Error(w, "blackberry is waking, retry shortly", http.StatusServiceUnavailable)
				return
			}
		}
		proxy.ServeHTTP(w, req)
	})
	// Public routes sit on Funnel; the rest are only ever reached through a
	// tsnet node that injects the identity header (ServeNode).
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
func (g *Router) awake(ctx context.Context, addr, by string) error {
	if g.up(ctx, addr) {
		return nil
	}
	g.mu.Lock()
	a := g.waking
	if a == nil {
		a = &wakeAttempt{done: make(chan struct{})}
		g.waking = a
		go g.wake(a, addr, by)
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
func (g *Router) wake(a *wakeAttempt, addr, by string) {
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
	g.WakeRequested(by)
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
	g.save()
	go g.logger.SendEvent("autowake_succeeded", "succeeded", nil, nil)
}

// RunIdleSuspend suspends blackberry once every route has been idle for its
// idle_minutes, if the Router woke it. A manual Wake is never undone.
func (g *Router) RunIdleSuspend(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			g.suspendIfIdle(ctx)
		}
	}
}

func (g *Router) suspendIfIdle(ctx context.Context) {
	if !g.autoSuspend.Load() || !g.autoWoken.Load() || g.inflight.Load() > 0 ||
		time.Now().UnixNano() < g.idleAt.Load() {
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
	g.save()
	g.logger.SendEvent("autosuspend_succeeded", "succeeded", nil, nil)
}

// SettingsHandler reads (GET) and sets (POST, form fields wake/suspend =
// true|false) the auto-wake and auto-suspend switches. Turning auto-suspend
// off and on again keeps an auto-wake pending, so it still gets suspended.
func (g *Router) SettingsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			for name, sw := range map[string]*atomic.Bool{"wake": &g.autoWake, "suspend": &g.autoSuspend} {
				v := req.FormValue(name)
				if v == "" {
					continue
				}
				on, err := strconv.ParseBool(v)
				if err != nil {
					http.Error(w, name+" must be true or false", http.StatusBadRequest)
					return
				}
				sw.Store(on)
			}
			g.save()
			identity := tailnet.GetCallerIdentity(req)
			go g.logger.SendEvent("settings_changed", "succeeded", &identity,
				map[string]any{"wake": g.autoWake.Load(), "suspend": g.autoSuspend.Load()})
		}
		httpresponse.WriteJSON(w, http.StatusOK, map[string]bool{"wake": g.autoWake.Load(), "suspend": g.autoSuspend.Load()})
	})
}
