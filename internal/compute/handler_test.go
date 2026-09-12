package compute

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikbogman/homelab/internal/tailnet"
)

const (
	testUIOrigin = "https://control.example.ts.net"
	testIdentity = "nicola@example.com"
)

type loggedEvent struct {
	eventType string
	outcome   string
	identity  *string
	extra     map[string]any
}

type fakeLogger struct {
	events []loggedEvent
}

func (f *fakeLogger) SendEvent(eventType, outcome string, identity *string, extra map[string]any) {
	f.events = append(f.events, loggedEvent{eventType: eventType, outcome: outcome, identity: identity, extra: extra})
}

type fakeSuspender struct {
	calls int
	err   error
}

func (f *fakeSuspender) Suspend() error {
	f.calls++
	return f.err
}

func mustNewHandler(t *testing.T, suspender Suspender, logger EventLogger) *Handler {
	t.Helper()
	h, err := NewHandler(testUIOrigin, suspender, logger, "127.0.0.1")
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	return h
}

func authedRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set(tailnet.IdentityHeader, testIdentity)
	return r
}

func TestHealthCheckRequiresIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestHealthCheckReportsReachableWithValidIdentity(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodGet, "/health"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	want := `{"reachable":true}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}

func TestHealthCheckRequiresNoExtraCredentialsBeyondIdentityHeader(t *testing.T) {
	// No allow-list of specific identities -- tailnet membership alone is sufficient.
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(tailnet.IdentityHeader, "anyone-on-the-tailnet@example.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestFirstHealthCheckLogsReachabilityChanged(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodGet, "/health"))

	if len(logger.events) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(logger.events), logger.events)
	}
	got := logger.events[0]
	if got.eventType != "reachability_changed" || got.outcome != "reachable" || got.identity != nil {
		t.Fatalf("got event %+v, want reachability_changed/reachable/nil", got)
	}
}

func TestSubsequentHealthChecksDoNotRelogReachability(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodGet, "/health"))
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodGet, "/health"))
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodGet, "/health"))

	if len(logger.events) != 1 {
		t.Fatalf("got %d events, want 1", len(logger.events))
	}
}

func TestCorsAllowsUiOrigin(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	req := authedRequest(http.MethodGet, "/health")
	req.Header.Set("Origin", testUIOrigin)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testUIOrigin {
		t.Fatalf("got Access-Control-Allow-Origin %q, want %q", got, testUIOrigin)
	}
}

func TestCorsRejectsOtherOrigins(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	req := authedRequest(http.MethodGet, "/health")
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "https://evil.example.com" {
		t.Fatalf("got Access-Control-Allow-Origin %q, want anything else", got)
	}
}

func TestNewHandlerRefusesOffTailnetBindHost(t *testing.T) {
	_, err := NewHandler(testUIOrigin, &fakeSuspender{}, &fakeLogger{}, "0.0.0.0")
	var bindErr *tailnet.BindOffTailnetError
	if !errors.As(err, &bindErr) {
		t.Fatalf("got err %v, want *tailnet.BindOffTailnetError", err)
	}
}

func TestSuspendRequiresIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/suspend", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestSuspendRunsTheConfiguredSystemSuspender(t *testing.T) {
	suspender := &fakeSuspender{}
	h := mustNewHandler(t, suspender, &fakeLogger{})

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/suspend"))

	if suspender.calls != 1 {
		t.Fatalf("got %d suspend calls, want 1", suspender.calls)
	}
}

func TestSuspendReturns200OnSuccess(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodPost, "/suspend"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	want := `{"suspend":"succeeded"}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}

func TestSuspendLogsRequestedThenSucceededWithCallerIdentity(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/suspend"))

	if len(logger.events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(logger.events), logger.events)
	}
	if logger.events[0].eventType != "suspend_requested" || logger.events[0].outcome != "requested" ||
		logger.events[0].identity == nil || *logger.events[0].identity != testIdentity {
		t.Fatalf("got first event %+v, want suspend_requested/requested/%s", logger.events[0], testIdentity)
	}
	if logger.events[1].eventType != "suspend_succeeded" || logger.events[1].outcome != "succeeded" ||
		logger.events[1].identity == nil || *logger.events[1].identity != testIdentity {
		t.Fatalf("got second event %+v, want suspend_succeeded/succeeded/%s", logger.events[1], testIdentity)
	}
}

func TestSuspendReturns500AndLogsFailedWhenSuspenderFails(t *testing.T) {
	logger := &fakeLogger{}
	suspender := &fakeSuspender{err: errors.New("systemctl suspend: exit status 1")}
	h := mustNewHandler(t, suspender, logger)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodPost, "/suspend"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want 500", rec.Code)
	}
	last := logger.events[len(logger.events)-1]
	if last.eventType != "suspend_failed" || last.outcome != "failed" || last.identity == nil || *last.identity != testIdentity {
		t.Fatalf("got last event %+v, want suspend_failed/failed/%s", last, testIdentity)
	}
}

func TestSuspendDoesNotGuardAgainstRepeatedRequests(t *testing.T) {
	// The API always attempts the action -- the UI's disabled state is the only guard.
	suspender := &fakeSuspender{}
	h := mustNewHandler(t, suspender, &fakeLogger{})

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/suspend"))
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/suspend"))

	if suspender.calls != 2 {
		t.Fatalf("got %d suspend calls, want 2", suspender.calls)
	}
}
