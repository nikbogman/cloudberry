package edge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type event struct {
	eventType, outcome string
	extra              map[string]any
}

type chanLogger chan event

func (c chanLogger) SendEvent(eventType, outcome string, _ *string, extra map[string]any) {
	c <- event{eventType, outcome, extra}
}

func TestWithAccessLogShipsOneEventPerRequest(t *testing.T) {
	events := make(chanLogger, 1)
	h := WithAccessLog(http.NotFoundHandler(), events)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	h.ServeHTTP(httptest.NewRecorder(), req)

	e := <-events
	if e.eventType != "request" || e.outcome != "4xx" {
		t.Fatalf("got %s/%s, want request/4xx", e.eventType, e.outcome)
	}
	if e.extra["status"] != 404 || e.extra["path"] != "/nope" || e.extra["client"] != "203.0.113.7" {
		t.Fatalf("unexpected extra: %v", e.extra)
	}
}
