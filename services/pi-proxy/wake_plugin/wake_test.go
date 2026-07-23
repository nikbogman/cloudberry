package wakeplugin

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCall_PostsToTheGivenURLAndReportsSuccess(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if r.Method != http.MethodPost {
			t.Errorf("got method %s, want POST", r.Method)
		}
	}))
	defer server.Close()

	caller := newWakeCaller()
	got := caller.Call(server.URL)

	if requests := atomic.LoadInt32(&requests); requests != 1 {
		t.Errorf("got %d requests, want 1", requests)
	}
	if got != outcomeSucceeded {
		t.Errorf("got outcome %v, want outcomeSucceeded", got)
	}
}

func TestCall_ThrottlesRepeatCallsToTheSameURLWithinTheWindow(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
	}))
	defer server.Close()

	now := time.Now()
	caller := newWakeCaller()
	caller.now = func() time.Time { return now }

	caller.Call(server.URL)
	now = now.Add(throttleWindow - time.Second)
	got := caller.Call(server.URL)

	if requests := atomic.LoadInt32(&requests); requests != 1 {
		t.Errorf("got %d requests, want 1 (second call should have been throttled)", requests)
	}
	if got != outcomeThrottled {
		t.Errorf("got outcome %v, want outcomeThrottled", got)
	}
}

func TestCall_DoesNotThrottleOnceTheWindowHasElapsed(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
	}))
	defer server.Close()

	now := time.Now()
	caller := newWakeCaller()
	caller.now = func() time.Time { return now }

	caller.Call(server.URL)
	now = now.Add(throttleWindow + time.Second)
	caller.Call(server.URL)

	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("got %d requests, want 2 (throttle window had elapsed)", got)
	}
}

func TestCall_DoesNotThrottleDifferentURLs(t *testing.T) {
	var requestsA, requestsB int32
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestsA, 1)
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestsB, 1)
	}))
	defer serverB.Close()

	caller := newWakeCaller()
	caller.Call(serverA.URL)
	caller.Call(serverB.URL)

	if got := atomic.LoadInt32(&requestsA); got != 1 {
		t.Errorf("server A got %d requests, want 1", got)
	}
	if got := atomic.LoadInt32(&requestsB); got != 1 {
		t.Errorf("server B got %d requests, want 1", got)
	}
}

func TestCall_ReportsFailedOnAnErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	got := callWithTimeout(t, newWakeCaller(), server.URL)

	if got != outcomeFailed {
		t.Errorf("got outcome %v, want outcomeFailed", got)
	}
}

func TestCall_ReportsTimedOutOnATimeout(t *testing.T) {
	unblock := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
	}))
	defer server.Close()
	defer close(unblock)

	caller := newWakeCaller()
	caller.client.Timeout = 50 * time.Millisecond

	got := callWithTimeout(t, caller, server.URL)

	if got != outcomeTimedOut {
		t.Errorf("got outcome %v, want outcomeTimedOut", got)
	}
}

func TestCall_DoesNotRecordAThrottledCallAsSentOnFailure(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	caller := newWakeCaller()
	caller.Call(server.URL)
	caller.Call(server.URL)

	// A failed call still counts against the throttle window (it was
	// attempted) -- this documents that choice rather than a silent retry
	// storm against a Pi API that's already returning errors.
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("got %d requests, want 1 (throttle applies even after a failed call)", got)
	}
}

// callWithTimeout runs Call in a goroutine and fails the test if it
// doesn't return promptly -- Call must never block the caller regardless
// of outcome (ADR-0012's fail-open guarantee).
func callWithTimeout(t *testing.T, c *wakeCaller, url string) outcome {
	t.Helper()

	result := make(chan outcome, 1)
	go func() { result <- c.Call(url) }()

	select {
	case got := <-result:
		return got
	case <-time.After(time.Second):
		t.Fatal("Call did not return in time")
		return outcomeFailed
	}
}
