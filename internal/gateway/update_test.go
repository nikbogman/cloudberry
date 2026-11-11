package gateway

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikbogman/cloudberry/internal/routes"
)

func TestRoutesChanged(t *testing.T) {
	body := `{"idle_minutes":30,"routes":[{"name":"immich","upstream":"http://10.0.0.2:2283","public":false,"wake":true}]}`
	hostd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/routes" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	defer hostd.Close()

	file := filepath.Join(t.TempDir(), "routes.json")
	u := &Updates{HostdURL: hostd.URL, RoutesFile: file, Logger: noopLogger{}, Client: hostd.Client()}

	u.Current, _ = routes.Parse([]byte(`{"idle_minutes":30,"routes":[]}`), "old")
	if !u.routesChanged(t.Context()) {
		t.Fatal("new route not picked up")
	}
	if got, _ := os.ReadFile(file); string(got) != body {
		t.Fatalf("routes file = %s", got)
	}

	u.Current, _ = routes.Load(file)
	if u.routesChanged(t.Context()) {
		t.Fatal("unchanged routes should not restart gateway")
	}

	body = `{"idle_minutes":0,"routes":[]}` // invalid: kept out
	if u.routesChanged(t.Context()) {
		t.Fatal("invalid routes accepted")
	}

	hostd.Close() // asleep
	if u.routesChanged(t.Context()) {
		t.Fatal("unreachable hostd should change nothing")
	}
}

func TestStatePersists(t *testing.T) {
	file := filepath.Join(t.TempDir(), "state.json")
	g := NewRouter("http://unused", "00:11:22:33:44:55", &fakeBlackberry{}, noopLogger{})
	if err := g.LoadState(file); err != nil {
		t.Fatal(err)
	}
	g.autoWoken.Store(true)
	g.autoSuspend.Store(false)
	g.save()

	g2 := NewRouter("http://unused", "00:11:22:33:44:55", &fakeBlackberry{}, noopLogger{})
	if err := g2.LoadState(file); err != nil {
		t.Fatal(err)
	}
	if !g2.autoWoken.Load() || g2.autoSuspend.Load() || !g2.autoWake.Load() {
		t.Fatalf("restored woken=%v suspend=%v wake=%v", g2.autoWoken.Load(), g2.autoSuspend.Load(), g2.autoWake.Load())
	}
}
