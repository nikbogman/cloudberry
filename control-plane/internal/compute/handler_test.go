package compute

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/nikbogman/homelab/control-plane/internal/tailnet"
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

// mu guards events: the hold's expiry timer calls SendEvent from its own
// goroutine, concurrently with whatever the test is doing on the main one.
type fakeLogger struct {
	mu     sync.Mutex
	events []loggedEvent
}

// snapshot is the race-safe way to read events; tests with a live expiry
// timer (a shrunk holdDuration) must use it instead of the field directly.
func (f *fakeLogger) snapshot() []loggedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]loggedEvent(nil), f.events...)
}

func (f *fakeLogger) SendEvent(eventType, outcome string, identity *string, extra map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

type fakeContainerRuntime struct {
	containers []RoutableContainer
	err        error

	active   bool
	statsErr error
}

func (f *fakeContainerRuntime) RoutableContainers() ([]RoutableContainer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.containers, nil
}

func (f *fakeContainerRuntime) ActivityAboveBaseline() (bool, error) {
	if f.statsErr != nil {
		return false, f.statsErr
	}
	return f.active, nil
}

// mustNewHandler is for tests unconcerned with proxy routing; it wires an
// empty fakeContainerRuntime. Proxy tests use mustNewHandlerWithRuntime.
func mustNewHandler(t *testing.T, suspender Suspender, logger EventLogger) *Handler {
	t.Helper()
	return mustNewHandlerWithRuntime(t, suspender, &fakeContainerRuntime{}, logger)
}

func mustNewHandlerWithRuntime(t *testing.T, suspender Suspender, runtime ContainerRuntime, logger EventLogger) *Handler {
	t.Helper()
	h, err := NewHandler(testUIOrigin, suspender, runtime, logger, "127.0.0.1")
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
	_, err := NewHandler(testUIOrigin, &fakeSuspender{}, &fakeContainerRuntime{}, &fakeLogger{}, "0.0.0.0")
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

func loopbackRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.RemoteAddr = "127.0.0.1:12345"
	return r
}

func TestHoldAcquireRequiresIdentityHeaderOrLoopback(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/hold", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestHoldAcquireAcceptsLoopbackCallerWithNoIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, loopbackRequest(http.MethodPost, "/hold"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestHoldAcquireLogsHoldAcquiredWithCallerIdentity(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))

	if len(logger.events) != 1 {
		t.Fatalf("got %d events, want 1: %+v", len(logger.events), logger.events)
	}
	got := logger.events[0]
	if got.eventType != "hold_acquired" || got.outcome != "acquired" || got.identity == nil || *got.identity != testIdentity {
		t.Fatalf("got event %+v, want hold_acquired/acquired/%s", got, testIdentity)
	}
}

func TestHoldAcquireAgainBeforeExpiryRenewsRatherThanStacking(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))

	// A single release must clear the hold entirely -- if renewal had
	// stacked a second hold underneath, one release wouldn't be enough
	// and a second release would still find something active to log.
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodDelete, "/hold"))
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodDelete, "/hold"))

	released := 0
	for _, e := range logger.events {
		if e.eventType == "hold_released" {
			released++
		}
	}
	if released != 1 {
		t.Fatalf("got %d hold_released events across repeated deletes, want 1 (no stacking): %+v", released, logger.events)
	}
}

func TestHoldReleaseEndsAnActiveHoldAndLogsHoldReleased(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodDelete, "/hold"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	last := logger.events[len(logger.events)-1]
	if last.eventType != "hold_released" || last.outcome != "released" || last.identity == nil || *last.identity != testIdentity {
		t.Fatalf("got last event %+v, want hold_released/released/%s", last, testIdentity)
	}
}

func TestHoldReleaseWithNoActiveHoldLogsNothing(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodDelete, "/hold"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if len(logger.events) != 0 {
		t.Fatalf("got %d events, want 0: %+v", len(logger.events), logger.events)
	}
}

func withShortHoldDuration(t *testing.T, d time.Duration) {
	t.Helper()
	prev := holdDuration
	holdDuration = d
	t.Cleanup(func() { holdDuration = prev })
}

func TestHoldExpiresOnItsOwnAndLogsHoldExpiredWithNoIdentity(t *testing.T) {
	withShortHoldDuration(t, 10*time.Millisecond)
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range logger.snapshot() {
			if e.eventType == "hold_expired" {
				if e.outcome != "expired" || e.identity != nil {
					t.Fatalf("got expire event %+v, want hold_expired/expired/nil", e)
				}
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("hold_expired was never logged: %+v", logger.snapshot())
}

func TestHoldRenewalPreventsTheOriginalDeadlineFromLoggingAnExpiry(t *testing.T) {
	withShortHoldDuration(t, 100*time.Millisecond)
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeSuspender{}, logger)

	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold"))
	time.Sleep(60 * time.Millisecond)
	h.ServeHTTP(httptest.NewRecorder(), authedRequest(http.MethodPost, "/hold")) // renew, pushing deadline out again

	// Past the original deadline (100ms from the first acquire) but well
	// before the renewed one (100ms from the second, ~60ms in).
	time.Sleep(60 * time.Millisecond)
	for _, e := range logger.snapshot() {
		if e.eventType == "hold_expired" {
			t.Fatalf("got hold_expired from the stale timer after renewal, want none yet: %+v", logger.snapshot())
		}
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

type statusResponse struct {
	IdleSeconds          float64 `json:"idleSeconds"`
	HoldActive           bool    `json:"holdActive"`
	HoldRemainingSeconds float64 `json:"holdRemainingSeconds"`
	DryRun               bool    `json:"dryRun"`
}

func TestStatusRequiresIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	h.SetIdleStatus(func() (time.Duration, bool) { return 0, false })
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestStatusRejectsLoopbackWithNoIdentityHeader(t *testing.T) {
	// Unlike /hold, /status is a read diagnostic, not an automation
	// target -- no loopback exception.
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	h.SetIdleStatus(func() (time.Duration, bool) { return 0, false })
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, loopbackRequest(http.MethodGet, "/status"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestStatusReportsIdleDurationHoldStateAndDryRunFlag(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	h.SetIdleStatus(func() (time.Duration, bool) { return 90 * time.Second, true })
	h.hold.acquire(&fakeLogger{}, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodGet, "/status"))

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	var got statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.IdleSeconds != 90 {
		t.Fatalf("got idleSeconds %v, want 90", got.IdleSeconds)
	}
	if !got.HoldActive {
		t.Fatalf("got holdActive false, want true")
	}
	if got.HoldRemainingSeconds <= 0 || got.HoldRemainingSeconds > holdDuration.Seconds() {
		t.Fatalf("got holdRemainingSeconds %v, want in (0, %v]", got.HoldRemainingSeconds, holdDuration.Seconds())
	}
	if !got.DryRun {
		t.Fatalf("got dryRun false, want true")
	}
}

func TestStatusReportsNoHoldWhenNoneActive(t *testing.T) {
	h := mustNewHandler(t, &fakeSuspender{}, &fakeLogger{})
	h.SetIdleStatus(func() (time.Duration, bool) { return 0, false })
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, authedRequest(http.MethodGet, "/status"))

	var got statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.HoldActive || got.HoldRemainingSeconds != 0 {
		t.Fatalf("got holdActive=%v holdRemainingSeconds=%v, want false/0", got.HoldActive, got.HoldRemainingSeconds)
	}
}
