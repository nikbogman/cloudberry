package gateway

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type noopLogger struct{}

func (noopLogger) SendEvent(string, string, *string, map[string]any) {}

// Fake hostd that answers /health only once the fake WoL sender has fired.
type fakeBlackberry struct {
	up    atomic.Bool
	sends atomic.Int32
}

func (f *fakeBlackberry) Send(string) error {
	f.sends.Add(1)
	go func() { time.Sleep(50 * time.Millisecond); f.up.Store(true) }()
	return nil
}

func TestConcurrentRequestsShareOneWake(t *testing.T) {
	bb := &fakeBlackberry{}
	hostd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !bb.up.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer hostd.Close()
	stack := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer stack.Close()

	g := NewRouter(hostd.URL, "00:11:22:33:44:55", bb, noopLogger{})
	g.pollEvery = 10 * time.Millisecond
	h, err := g.RouteHandler(Route{Name: "t", Upstream: stack.URL, Public: true, Wake: true})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			if rec.Code != 200 || rec.Body.String() != "ok" {
				t.Errorf("got %d %q", rec.Code, rec.Body.String())
			}
		}()
	}
	wg.Wait()
	if n := bb.sends.Load(); n != 1 {
		t.Fatalf("WoL sent %d times, want 1", n)
	}
	if !g.autoWoken.Load() {
		t.Fatal("router should remember it woke blackberry")
	}
}

func TestPrivateRouteNeedsIdentity(t *testing.T) {
	g := NewRouter("http://unused", "00:11:22:33:44:55", &fakeBlackberry{}, noopLogger{})
	h, _ := g.RouteHandler(Route{Name: "t", Upstream: "http://unused"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}
