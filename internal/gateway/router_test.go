package gateway

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type noopLogger struct{}

func (noopLogger) SendEvent(string, string, *string, map[string]any) {}

// Fake blackberry: the stack only starts listening once the fake WoL sender
// has fired, like a suspended host whose port refuses connections.
type fakeBlackberry struct {
	sends atomic.Int32
	start func()
}

func (f *fakeBlackberry) Send(string) error {
	f.sends.Add(1)
	go func() { time.Sleep(50 * time.Millisecond); f.start() }()
	return nil
}

func TestConcurrentRequestsShareOneWake(t *testing.T) {
	stack := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer stack.Close()
	upstream := "http://" + stack.Listener.Addr().String()
	// Close the listener so the port refuses until "resume"; reopen on start.
	addr := stack.Listener.Addr().String()
	stack.Listener.Close()
	bb := &fakeBlackberry{start: func() {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			t.Error(err)
			return
		}
		stack.Listener = l
		stack.Start()
	}}

	g := NewRouter("http://unused", "00:11:22:33:44:55", bb, noopLogger{})
	g.pollEvery = 10 * time.Millisecond
	h, err := g.RouteHandler(Route{Name: "t", Upstream: upstream, Public: true, Wake: true})
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

func TestUpstreamAddrDefaultsPort(t *testing.T) {
	for in, want := range map[string]string{
		"http://192.168.0.196:2283": "192.168.0.196:2283",
		"http://blackberry":         "blackberry:80",
		"https://blackberry":        "blackberry:443",
	} {
		u, _ := url.Parse(in)
		if got := upstreamAddr(u); got != want {
			t.Errorf("%s: got %s, want %s", in, got, want)
		}
	}
}
