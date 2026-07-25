package gateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikbogman/homelab/services/internal/tailnet"
)

const (
	testMACAddress = "AA:BB:CC:DD:EE:FF"
	testIdentity  = "nicola@example.com"
)

type fakeWaker struct {
	calls []string
	err   error
}

func (f *fakeWaker) Send(macAddress string) error {
	f.calls = append(f.calls, macAddress)
	return f.err
}

type loggedEvent struct {
	eventType string
	outcome   string
	identity  *string
}

type fakeLogger struct {
	events []loggedEvent
}

func (f *fakeLogger) SendEvent(eventType, outcome string, identity *string, extra map[string]any) {
	f.events = append(f.events, loggedEvent{eventType: eventType, outcome: outcome, identity: identity})
}

func mustNewHandler(t *testing.T, waker Waker, logger EventLogger) http.Handler {
	t.Helper()
	h, err := NewHandler(testMACAddress, waker, logger, "127.0.0.1")
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	return h
}

func TestWakeRequiresIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeWaker{}, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestWakeAllowsALoopbackCallerWithNoIdentityHeader(t *testing.T) {
	waker := &fakeWaker{}
	h := mustNewHandler(t, waker, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if len(waker.calls) != 1 || waker.calls[0] != testMACAddress {
		t.Fatalf("got waker calls %v, want [%q]", waker.calls, testMACAddress)
	}
}

func TestWakeStillHonorsIdentityHeaderFromANonLoopbackCaller(t *testing.T) {
	waker := &fakeWaker{}
	h := mustNewHandler(t, waker, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if len(waker.calls) != 1 || waker.calls[0] != testMACAddress {
		t.Fatalf("got waker calls %v, want [%q]", waker.calls, testMACAddress)
	}
}

func TestWakeSendsMagicPacketToConfiguredMac(t *testing.T) {
	waker := &fakeWaker{}
	h := mustNewHandler(t, waker, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if len(waker.calls) != 1 || waker.calls[0] != testMACAddress {
		t.Fatalf("got waker calls %v, want [%q]", waker.calls, testMACAddress)
	}
}

func TestWakeReturns200OnSuccess(t *testing.T) {
	h := mustNewHandler(t, &fakeWaker{}, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	want := `{"wake":"succeeded"}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}

func TestWakeLogsRequestedThenSucceededWithCallerIdentity(t *testing.T) {
	logger := &fakeLogger{}
	h := mustNewHandler(t, &fakeWaker{}, logger)
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if len(logger.events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(logger.events), logger.events)
	}
	if logger.events[0].eventType != "wake_requested" || logger.events[0].outcome != "requested" ||
		logger.events[0].identity == nil || *logger.events[0].identity != testIdentity {
		t.Fatalf("got first event %+v, want wake_requested/requested/%s", logger.events[0], testIdentity)
	}
	if logger.events[1].eventType != "wake_succeeded" || logger.events[1].outcome != "succeeded" ||
		logger.events[1].identity == nil || *logger.events[1].identity != testIdentity {
		t.Fatalf("got second event %+v, want wake_succeeded/succeeded/%s", logger.events[1], testIdentity)
	}
}

func TestWakeReturns500AndLogsFailedWhenSenderFails(t *testing.T) {
	waker := &fakeWaker{err: errors.New("network is unreachable")}
	logger := &fakeLogger{}
	h := mustNewHandler(t, waker, logger)
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want 500", rec.Code)
	}
	last := logger.events[len(logger.events)-1]
	if last.eventType != "wake_failed" || last.outcome != "failed" || last.identity == nil || *last.identity != testIdentity {
		t.Fatalf("got last event %+v, want wake_failed/failed/%s", last, testIdentity)
	}
}

func TestWakeDoesNotGuardAgainstRepeatedRequests(t *testing.T) {
	waker := &fakeWaker{}
	h := mustNewHandler(t, waker, &fakeLogger{})
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/wake", nil)
		r.Header.Set(tailnet.IdentityHeader, testIdentity)
		return r
	}

	h.ServeHTTP(httptest.NewRecorder(), req())
	h.ServeHTTP(httptest.NewRecorder(), req())

	if len(waker.calls) != 2 {
		t.Fatalf("got %d waker calls, want 2", len(waker.calls))
	}
}

func TestNewHandlerRefusesOffTailnetBindHost(t *testing.T) {
	_, err := NewHandler(testMACAddress, &fakeWaker{}, &fakeLogger{}, "0.0.0.0")
	var bindErr *tailnet.BindOffTailnetError
	if !errors.As(err, &bindErr) {
		t.Fatalf("got err %v, want *tailnet.BindOffTailnetError", err)
	}
}

func TestNewHandlerRejectsAMalformedMacAddressAtStartup(t *testing.T) {
	_, err := NewHandler("not-a-mac", &fakeWaker{}, &fakeLogger{}, "127.0.0.1")
	if err == nil {
		t.Fatal("NewHandler succeeded, want error")
	}
}
