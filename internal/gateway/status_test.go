package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikbogman/cloudberry/internal/routes"
)

func TestStatusTracksWakesAndSuspends(t *testing.T) {
	g := NewRouter("http://unused", "00:11:22:33:44:55", &fakeBlackberry{}, noopLogger{})
	file := filepath.Join(t.TempDir(), "state.json")
	if err := g.LoadState(file); err != nil {
		t.Fatal(err)
	}
	g.RouteHandler(routes.Route{Name: "immich", Upstream: "http://10.0.0.2:2283", Wake: true})
	g.SetURLs("raspberry", "tail1.ts.net")

	t0 := time.Now()
	g.observe(false, t0) // first probe: state known, no event
	g.WakeRequested("nik@github")
	g.observe(true, time.Now().Add(8*time.Second))
	g.observe(true, time.Now().Add(9*time.Second)) // no change, no event
	g.observe(false, time.Now().Add(time.Hour))

	rec := httptest.NewRecorder()
	g.StatusHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/status", nil))
	var st struct {
		Awake           *bool   `json:"awake"`
		LastWakeSeconds float64 `json:"last_wake_seconds"`
		Events          []Event `json:"events"`
		Routes          []routeStatus
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err, rec.Body.String())
	}
	if st.Awake == nil || *st.Awake || len(st.Events) != 2 || st.Events[0].Type != "woke" || st.Events[1].Type != "suspended" {
		t.Fatalf("got %s", rec.Body.String())
	}
	if st.LastWakeSeconds < 7.5 || st.LastWakeSeconds > 8.5 {
		t.Fatalf("last wake %vs, want ~8s", st.LastWakeSeconds)
	}
	if len(st.Routes) != 1 || st.Routes[0].URL != "https://immich.tail1.ts.net/" || st.Routes[0].LastRequest != nil {
		t.Fatalf("routes %+v", st.Routes)
	}

	// A restart keeps the history, and a first probe that agrees adds nothing.
	g2 := NewRouter("http://unused", "00:11:22:33:44:55", &fakeBlackberry{}, noopLogger{})
	if err := g2.LoadState(file); err != nil {
		t.Fatal(err)
	}
	g2.observe(false, time.Now())
	if len(g2.host.Events) != 2 || g2.host.LastWakeSeconds != st.LastWakeSeconds {
		t.Fatalf("after restart: %+v", g2.host)
	}

	// Events older than eventsKept drop off.
	g2.observe(true, time.Now().Add(eventsKept+2*time.Hour))
	if len(g2.host.Events) != 1 || g2.host.WokenBy != "" {
		t.Fatalf("old events kept or stale woken_by: %+v", g2.host)
	}
}
