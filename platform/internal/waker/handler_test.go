package waker

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikbogman/homelab/platform/internal/tailnet"
)

const (
	testMACAddress = "AA:BB:CC:DD:EE:FF"
	testIdentity   = "nicola@example.com"
)

type fakeSender struct {
	calls []string
	err   error
}

func (f *fakeSender) Send(macAddress string) error {
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

func mustNewHandler(t *testing.T, sender Sender, logger EventLogger) http.Handler {
	t.Helper()
	h, err := NewHandler(Config{MACAddress: testMACAddress, BindHost: "127.0.0.1"}, sender, logger)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	return h
}

func TestWakeRequiresIdentityHeader(t *testing.T) {
	h := mustNewHandler(t, &fakeSender{}, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestWakeAllowsALoopbackCallerWithNoIdentityHeader(t *testing.T) {
	sender := &fakeSender{}
	h := mustNewHandler(t, sender, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if len(sender.calls) != 1 || sender.calls[0] != testMACAddress {
		t.Fatalf("got sender calls %v, want [%q]", sender.calls, testMACAddress)
	}
}

func TestWakeStillHonorsIdentityHeaderFromANonLoopbackCaller(t *testing.T) {
	sender := &fakeSender{}
	h := mustNewHandler(t, sender, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if len(sender.calls) != 1 || sender.calls[0] != testMACAddress {
		t.Fatalf("got sender calls %v, want [%q]", sender.calls, testMACAddress)
	}
}

func TestWakeSendsMagicPacketToConfiguredMac(t *testing.T) {
	sender := &fakeSender{}
	h := mustNewHandler(t, sender, &fakeLogger{})
	req := httptest.NewRequest(http.MethodPost, "/wake", nil)
	req.Header.Set(tailnet.IdentityHeader, testIdentity)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if len(sender.calls) != 1 || sender.calls[0] != testMACAddress {
		t.Fatalf("got sender calls %v, want [%q]", sender.calls, testMACAddress)
	}
}

func TestWakeReturns200OnSuccess(t *testing.T) {
	h := mustNewHandler(t, &fakeSender{}, &fakeLogger{})
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
	h := mustNewHandler(t, &fakeSender{}, logger)
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
	sender := &fakeSender{err: errors.New("network is unreachable")}
	logger := &fakeLogger{}
	h := mustNewHandler(t, sender, logger)
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
	sender := &fakeSender{}
	h := mustNewHandler(t, sender, &fakeLogger{})
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/wake", nil)
		r.Header.Set(tailnet.IdentityHeader, testIdentity)
		return r
	}

	h.ServeHTTP(httptest.NewRecorder(), req())
	h.ServeHTTP(httptest.NewRecorder(), req())

	if len(sender.calls) != 2 {
		t.Fatalf("got %d sender calls, want 2", len(sender.calls))
	}
}

func TestNewHandlerRefusesOffTailnetBindHost(t *testing.T) {
	cfg := Config{MACAddress: testMACAddress, BindHost: "0.0.0.0"}
	_, err := NewHandler(cfg, &fakeSender{}, &fakeLogger{})
	var bindErr *tailnet.BindOffTailnetError
	if !errors.As(err, &bindErr) {
		t.Fatalf("got err %v, want *tailnet.BindOffTailnetError", err)
	}
}

func TestNewHandlerRejectsAMalformedMacAddressAtStartup(t *testing.T) {
	cfg := Config{MACAddress: "not-a-mac", BindHost: "127.0.0.1"}
	_, err := NewHandler(cfg, &fakeSender{}, &fakeLogger{})
	if err == nil {
		t.Fatal("NewHandler succeeded, want error")
	}
}
