package gateway

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/nikbogman/cloudberry/internal/httpresponse"
	"github.com/nikbogman/cloudberry/internal/routes"
)

// eventsKept is how far back GET /status reports wakes and suspends: the UI
// shows today, in the browser's time zone, so a day either side of the Pi's.
const eventsKept = 48 * time.Hour

// Event is a change in blackberry's state, as gateway's probe saw it.
type Event struct {
	At   time.Time `json:"at"`
	Type string    `json:"type"` // "woke" or "suspended"
}

// hostState is what gateway knows about blackberry, saved with the switches
// so that restarts keep today's history. It's gateway's own view: a probe of
// the first route's upstream every probeEvery, so it sees a suspend from the
// UI, gateway or the power button alike.
type hostState struct {
	Awake           *bool     `json:"awake,omitempty"` // nil until the first probe
	Since           time.Time `json:"since,omitempty"`
	WokenBy         string    `json:"woken_by,omitempty"` // a route name or a tailnet login; empty if gateway didn't send the wake
	LastWakeSeconds float64   `json:"last_wake_seconds,omitempty"`
	Events          []Event   `json:"events,omitempty"`
}

type pendingWake struct {
	by string
	at time.Time
}

type routeStat struct {
	route routes.Route
	url   string
	last  atomic.Int64 // unix nanos of the latest request, 0 for none
}

// WakeRequested records that a magic packet just went out on by's behalf (a
// route name, or the caller's login for the UI's Wake), and probes fast until
// blackberry answers, so the wake's duration is measured to the probe interval.
func (g *Router) WakeRequested(by string) {
	g.statusMu.Lock()
	g.pending = pendingWake{by: by, at: time.Now()}
	g.statusMu.Unlock()
	select {
	case g.kick <- struct{}{}:
	default:
	}
}

// RunProbe keeps hostState current by dialing addr (a stack's upstream on
// blackberry's LAN IP) every probeEvery, or every pollEvery while a wake is
// pending. ponytail: "awake" means that stack answers; a host up with the
// stack down reads as asleep.
func (g *Router) RunProbe(ctx context.Context, addr string, probeEvery time.Duration) {
	for {
		g.observe(g.up(ctx, addr), time.Now())
		wait := probeEvery
		g.statusMu.Lock()
		if !g.pending.at.IsZero() && time.Since(g.pending.at) < g.wakeTimeout {
			wait = g.pollEvery
		}
		g.statusMu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-g.kick:
		case <-time.After(wait):
		}
	}
}

func (g *Router) observe(up bool, now time.Time) {
	g.statusMu.Lock()
	h := &g.host
	if h.Awake != nil && *h.Awake == up {
		g.statusMu.Unlock()
		return
	}
	first := h.Awake == nil
	h.Awake = &up
	h.Since = now
	if first {
		// No saved state: when it changed is unknown, so no event.
		g.statusMu.Unlock()
		g.save()
		return
	}
	if up {
		h.Events = append(h.Events, Event{At: now, Type: "woke"})
		h.WokenBy = ""
		if p := g.pending; !p.at.IsZero() && now.Sub(p.at) < g.wakeTimeout {
			h.WokenBy = p.by
			h.LastWakeSeconds = now.Sub(p.at).Round(100 * time.Millisecond).Seconds()
		}
	} else {
		h.Events = append(h.Events, Event{At: now, Type: "suspended"})
		h.WokenBy = ""
	}
	g.pending = pendingWake{}
	for len(h.Events) > 0 && now.Sub(h.Events[0].At) > eventsKept {
		h.Events = h.Events[1:]
	}
	g.statusMu.Unlock()
	g.save()
}

// SetURLs fills each route's public URL once gateway's node knows its tailnet
// domain: its own hostname for a hostname route, under self's for a path one.
func (g *Router) SetURLs(self, domain string) {
	g.statusMu.Lock()
	defer g.statusMu.Unlock()
	for _, s := range g.routeStats {
		if s.route.Path == "" {
			s.url = "https://" + s.route.Name + "." + domain + "/"
		} else {
			s.url = "https://" + self + "." + domain + s.route.Path + "/"
		}
	}
}

type routeStatus struct {
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Wake        bool       `json:"wake"`
	LastRequest *time.Time `json:"last_request"`
}

type status struct {
	hostState
	Events []Event `json:"events"`
	// SleepsAt is when gateway will suspend blackberry if nothing uses it:
	// set only when gateway did the waking and auto-suspend is on.
	SleepsAt          *time.Time    `json:"sleeps_at"`
	IdleWindowSeconds float64       `json:"idle_window_seconds"`
	WakePending       bool          `json:"wake_pending"`
	AutoWake          bool          `json:"wake"`
	AutoSuspend       bool          `json:"suspend"`
	Routes            []routeStatus `json:"routes"`
}

// StatusHandler serves GET /status, everything the UI shows but Wake and
// Suspend themselves.
func (g *Router) StatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		g.statusMu.Lock()
		st := status{
			hostState:   g.host,
			Events:      append([]Event{}, g.host.Events...),
			WakePending: !g.pending.at.IsZero() && time.Since(g.pending.at) < g.wakeTimeout,
			AutoWake:    g.autoWake.Load(),
			AutoSuspend: g.autoSuspend.Load(),
			Routes:      []routeStatus{},
		}
		for _, s := range g.routeStats {
			rs := routeStatus{Name: s.route.Name, URL: s.url, Wake: s.route.Wake}
			if n := s.last.Load(); n != 0 {
				t := time.Unix(0, n)
				rs.LastRequest = &t
			}
			st.Routes = append(st.Routes, rs)
		}
		g.statusMu.Unlock()
		if st.Awake != nil && *st.Awake && g.autoWoken.Load() && st.AutoSuspend {
			t := time.Unix(0, g.idleAt.Load())
			st.SleepsAt = &t
			st.IdleWindowSeconds = time.Duration(g.idleWindow.Load()).Seconds()
		}
		httpresponse.WriteJSON(w, http.StatusOK, st)
	})
}
