package edge

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// echoBackend replies with its name, the request line and the identity
// header it received.
func echoBackend(t *testing.T, name string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, name+" "+r.Method+" "+r.URL.Path+" identity="+r.Header.Get(tailnet.IdentityHeader))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// newTestHandler points waker and hostd at local echo backends.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	waker := echoBackend(t, "waker")
	hostd := echoBackend(t, "hostd")
	return NewHandler(Config{
		WakerURL: mustParse(t, waker.URL),
		HostdURL: mustParse(t, hostd.URL),
		UIAssets: fstest.MapFS{"index.html": {Data: []byte("<html>ui</html>")}},
	}, http.DefaultTransport)
}

func serve(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set(tailnet.IdentityHeader, "spoofed@example.com")
	h.ServeHTTP(rec, req)
	return rec
}

func TestAPIRoutesForwardWithThePrefixStrippedAndNoCallerIdentity(t *testing.T) {
	h := newTestHandler(t)
	cases := []struct{ method, path, want string }{
		{http.MethodPost, "/api/waker/wake", "waker POST /wake identity="},
		{http.MethodGet, "/api/hostd/health", "hostd GET /health identity="},
		{http.MethodPost, "/api/hostd/suspend", "hostd POST /suspend identity="},
	}
	for _, c := range cases {
		rec := serve(h, c.method, c.path)
		if rec.Code != http.StatusOK || rec.Body.String() != c.want {
			t.Errorf("%s %s: got %d %q, want 200 %q", c.method, c.path, rec.Code, rec.Body.String(), c.want)
		}
	}
}

func TestOnlyTheKnownAPIRoutesAreForwarded(t *testing.T) {
	h := newTestHandler(t)
	for _, path := range []string{"/api/waker/other", "/api/hostd/other"} {
		if rec := serve(h, http.MethodPost, path); rec.Code != http.StatusNotFound {
			t.Errorf("POST %s: got %d, want 404", path, rec.Code)
		}
	}
	if rec := serve(h, http.MethodGet, "/api/hostd/suspend"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/hostd/suspend: got %d, want 405", rec.Code)
	}
}

func TestUIIsServedUnderSlashUI(t *testing.T) {
	h := newTestHandler(t)
	rec := serve(h, http.MethodGet, "/ui/")
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>ui</html>" {
		t.Fatalf("got %d %q, want 200 with index.html", rec.Code, rec.Body.String())
	}
}

func TestRootRedirectsToTheUI(t *testing.T) {
	h := newTestHandler(t)
	rec := serve(h, http.MethodGet, "/")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/ui/" {
		t.Fatalf("got %d Location=%q, want 302 /ui/", rec.Code, rec.Header().Get("Location"))
	}
}
