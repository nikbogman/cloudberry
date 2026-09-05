package compute

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func backendAddr(t *testing.T, backend *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("failed to parse backend URL %q: %v", backend.URL, err)
	}
	return u.Host
}

func TestProxy_RoutesToTheContainerLabeledForThePath(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("hello from immich"))
	}))
	defer backend.Close()

	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/immich", Addr: backendAddr(t, backend)},
	}}
	h := mustNewHandlerWithRuntime(t, &fakeSuspender{}, runtime, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/immich/photos/1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "hello from immich" {
		t.Fatalf("got body %q, want %q", rec.Body.String(), "hello from immich")
	}
	if gotPath != "/photos/1" {
		t.Fatalf("got backend path %q, want %q (route prefix stripped)", gotPath, "/photos/1")
	}
}

func TestProxy_NoConfigChangeNeededToRouteANewlyDiscoveredContainer(t *testing.T) {
	// The fake stands in for Docker label discovery -- no compute-api
	// code change or redeploy, just a container appearing in the list.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/newworkload", Addr: backendAddr(t, backend)},
	}}
	h := mustNewHandlerWithRuntime(t, &fakeSuspender{}, runtime, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/newworkload/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestProxy_NeverRoutesToAContainerWithNoRouteLabel(t *testing.T) {
	// fakeContainerRuntime.RoutableContainers only ever returns labeled
	// containers, so a request for any other path has nothing to match.
	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/immich", Addr: "127.0.0.1:1"},
	}}
	h := mustNewHandlerWithRuntime(t, &fakeSuspender{}, runtime, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unlabeled-service/", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", rec.Code)
	}
}

func TestProxy_ARealHTTPErrorFromALiveBackendPassesThroughUntouched(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream says no"))
	}))
	defer backend.Close()

	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/immich", Addr: backendAddr(t, backend)},
	}}
	h := mustNewHandlerWithRuntime(t, &fakeSuspender{}, runtime, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/immich/broken", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if rec.Body.String() != "upstream says no" {
		t.Fatalf("got body %q, want it passed through untouched", rec.Body.String())
	}
}

func TestProxy_DiscoveryFailureReturns502(t *testing.T) {
	runtime := &fakeContainerRuntime{err: errors.New("docker daemon unreachable")}
	h := mustNewHandlerWithRuntime(t, &fakeSuspender{}, runtime, &fakeLogger{})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/immich/", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestProxy_RecordsLastProxiedTimeOnASuccessfulRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/immich", Addr: backendAddr(t, backend)},
	}}
	signals := NewActivitySignals(&fakeLogger{}, time.Hour, false)
	h, err := NewHandler(testUIOrigin, &fakeSuspender{}, runtime, &fakeLogger{}, "127.0.0.1", signals)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}

	if got := signals.LastProxiedAt(); !got.IsZero() {
		t.Fatalf("got LastProxiedAt() %v before any request, want zero time", got)
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/immich/photos/1", nil))

	if got := signals.LastProxiedAt(); got.IsZero() {
		t.Fatal("got zero LastProxiedAt() after a successful proxy, want it recorded")
	}
}

func TestProxy_DoesNotRecordLastProxiedTimeWhenTheBackendIsUnreachable(t *testing.T) {
	runtime := &fakeContainerRuntime{containers: []RoutableContainer{
		{Path: "/immich", Addr: "127.0.0.1:1"}, // nothing listens on port 1
	}}
	signals := NewActivitySignals(&fakeLogger{}, time.Hour, false)
	h, err := NewHandler(testUIOrigin, &fakeSuspender{}, runtime, &fakeLogger{}, "127.0.0.1", signals)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/immich/photos/1", nil))

	if got := signals.LastProxiedAt(); !got.IsZero() {
		t.Fatalf("got LastProxiedAt() %v after an unreachable backend, want zero time", got)
	}
}
