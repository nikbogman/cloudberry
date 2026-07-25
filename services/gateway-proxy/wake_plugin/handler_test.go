package wakeplugin

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestHandler_ProvisionRequiresAWakeURL(t *testing.T) {
	h := &Handler{}

	if err := h.Provision(caddy.Context{}); err == nil {
		t.Error("got nil error, want an error for a missing WakeURL")
	}
}

func TestHandler_ServeHTTPCallsTheWakeURLAndAlwaysContinues(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusInternalServerError) // failure: next must still run
	}))
	defer server.Close()

	h := &Handler{WakeURL: server.URL}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	var nextCalled bool
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/server/photo", nil)
	if err := h.ServeHTTP(httptest.NewRecorder(), req, next); err != nil {
		t.Fatalf("ServeHTTP returned an error: %v", err)
	}

	if requests := atomic.LoadInt32(&requests); requests != 1 {
		t.Errorf("got %d requests to the wake URL, want 1", requests)
	}
	if !nextCalled {
		t.Error("next was not called despite the wake call failing (fail-open, ADR-0012)")
	}
}
