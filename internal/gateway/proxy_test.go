package gateway

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

var errTestWaker = errors.New("network is unreachable")

func withShortRetryWindow(t *testing.T, window, interval time.Duration) {
	t.Helper()
	prevWindow, prevInterval := proxyRetryWindow, proxyRetryInterval
	proxyRetryWindow, proxyRetryInterval = window, interval
	t.Cleanup(func() { proxyRetryWindow, proxyRetryInterval = prevWindow, prevInterval })
}

func withShortAutoWakeThrottle(t *testing.T, window time.Duration) {
	t.Helper()
	prev := autoWakeThrottle
	autoWakeThrottle = window
	t.Cleanup(func() { autoWakeThrottle = prev })
}

// closedPortTarget returns a host:port that refuses connections outright
// (nothing listening), splitting the result into host/port for Config.
func closedPortTarget(t *testing.T) (host string, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	if err := ln.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}
	return addr.IP.String(), addr.Port
}

func splitHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", rawURL, err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("failed to split host/port from %q: %v", u.Host, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port %q: %v", portStr, err)
	}
	return host, port
}

func TestProxy_AutoWakeThrottlesRepeatCallsWithinTheWindow(t *testing.T) {
	withShortAutoWakeThrottle(t, 90*time.Second)
	withShortRetryWindow(t, 50*time.Millisecond, 10*time.Millisecond)

	computeHost, computeProxyPort := closedPortTarget(t)
	waker := &fakeWaker{}
	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(t.TempDir()),
		ComputeHost: computeHost, ComputeProxyPort: computeProxyPort,
	}, waker, &fakeLogger{})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/server/foo", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/server/bar", nil))

	if len(waker.calls) != 1 {
		t.Fatalf("got %d wake calls, want 1 (second request should have been throttled)", len(waker.calls))
	}
}

func TestProxy_AutoWakeDoesNotThrottleOnceTheWindowHasElapsed(t *testing.T) {
	withShortAutoWakeThrottle(t, 10*time.Millisecond)
	withShortRetryWindow(t, 50*time.Millisecond, 10*time.Millisecond)

	computeHost, computeProxyPort := closedPortTarget(t)
	waker := &fakeWaker{}
	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(t.TempDir()),
		ComputeHost: computeHost, ComputeProxyPort: computeProxyPort,
	}, waker, &fakeLogger{})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/server/foo", nil))
	time.Sleep(20 * time.Millisecond)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/server/foo", nil))

	if len(waker.calls) != 2 {
		t.Fatalf("got %d wake calls, want 2 (throttle window had elapsed)", len(waker.calls))
	}
}

func TestProxy_FailsOpenWhenTheWakerErrors(t *testing.T) {
	withShortRetryWindow(t, 30*time.Millisecond, 10*time.Millisecond)

	computeHost, computeProxyPort := closedPortTarget(t)
	waker := &fakeWaker{err: errTestWaker}
	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(t.TempDir()),
		ComputeHost: computeHost, ComputeProxyPort: computeProxyPort,
	}, waker, &fakeLogger{})

	done := make(chan struct{})
	rec := httptest.NewRecorder()
	go func() {
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/server/foo", nil))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ServeHTTP did not return in time; a failing waker must not block the retry loop")
	}

	if len(waker.calls) != 1 {
		t.Fatalf("got %d wake calls, want 1", len(waker.calls))
	}
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughWithoutWaking(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream says no"))
	}))
	defer backend.Close()
	computeHost, computeProxyPort := splitHostPort(t, backend.URL)

	waker := &fakeWaker{}
	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(t.TempDir()),
		ComputeHost: computeHost, ComputeProxyPort: computeProxyPort,
	}, waker, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/server/foo", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if rec.Body.String() != "upstream says no" {
		t.Fatalf("got body %q, want it passed through untouched", rec.Body.String())
	}
	if len(waker.calls) != 0 {
		t.Fatalf("got %d wake calls, want 0 -- a real HTTP response must not trigger auto-wake", len(waker.calls))
	}
}

func TestProxy_RetriesUntilTheBackendStartsRespondingWithinTheWindow(t *testing.T) {
	withShortRetryWindow(t, time.Second, 20*time.Millisecond)

	computeHost, computeProxyPort := closedPortTarget(t)
	var backend *httptest.Server
	backend = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	waker := &fakeWaker{}
	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(t.TempDir()),
		ComputeHost: computeHost, ComputeProxyPort: computeProxyPort,
	}, waker, &fakeLogger{})

	// Start the real backend on the same address shortly after the first
	// attempt fails, simulating the compute host waking up mid-retry-loop.
	go func() {
		time.Sleep(60 * time.Millisecond)
		ln, listenErr := net.Listen("tcp", net.JoinHostPort(computeHost, strconv.Itoa(computeProxyPort)))
		if listenErr != nil {
			return
		}
		backend.Listener = ln
		backend.Start()
	}()
	defer backend.Close()

	done := make(chan *httptest.ResponseRecorder)
	go func() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/server/foo", nil))
		done <- rec
	}()

	select {
	case rec := <-done:
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", rec.Code)
		}
		if rec.Body.String() != "ok" {
			t.Fatalf("got body %q, want %q", rec.Body.String(), "ok")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete once the backend came up")
	}

	if len(waker.calls) != 1 {
		t.Fatalf("got %d wake calls, want exactly 1", len(waker.calls))
	}
}

func TestProxy_ServesStaticFilesFromUIAssets(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html>hi</html>")

	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(dir),
		ComputeHost: "127.0.0.1", ComputeProxyPort: 1,
	}, &fakeWaker{}, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "<html>hi</html>" {
		t.Fatalf("got body %q, want the file's contents", rec.Body.String())
	}
}

func TestProxy_DoesNotFallBackToIndexHTMLForAnUnknownPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html>hi</html>")

	h := mustNewHandlerWithConfig(t, Config{
		MACAddress: testMACAddress, BindHost: "127.0.0.1", UIAssets: os.DirFS(dir),
		ComputeHost: "127.0.0.1", ComputeProxyPort: 1,
	}, &fakeWaker{}, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/some/spa/route", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404 -- no SPA/try_files fallback", rec.Code)
	}
}

func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.WriteFile(dir+"/"+name, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
