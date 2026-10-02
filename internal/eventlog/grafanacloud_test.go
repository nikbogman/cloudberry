package eventlog

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	testLokiUser   = "123456"
	testLokiAPIKey = "glc_faketoken"
)

func TestSendEventPostsALokiPushPayloadWithBasicAuth(t *testing.T) {
	var gotRequest *http.Request
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := NewGrafanaCloudLogger(server.URL, testLokiUser, testLokiAPIKey, "waker-api")
	identity := "nicola@example.com"
	logger.SendEvent("wake_requested", "success", &identity, nil)

	if gotRequest == nil {
		t.Fatal("expected a request to be sent")
	}
	if user, pass, ok := gotRequest.BasicAuth(); !ok || user != testLokiUser || pass != testLokiAPIKey {
		t.Fatalf("got basic auth (%q, %q, %v), want (%q, %q, true)", user, pass, ok, testLokiUser, testLokiAPIKey)
	}

	var payload lokiPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if len(payload.Streams) != 1 {
		t.Fatalf("got %d streams, want 1", len(payload.Streams))
	}
	stream := payload.Streams[0]
	if stream.Stream != (lokiStreamLabels{EventType: "wake_requested", Outcome: "success", App: "waker-api"}) {
		t.Fatalf("got stream labels %+v, want event_type=wake_requested outcome=success app=waker-api", stream.Stream)
	}
	if len(stream.Values) != 1 {
		t.Fatalf("got %d values, want 1", len(stream.Values))
	}
	timestamp, line := stream.Values[0][0], stream.Values[0][1]
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		t.Fatalf("timestamp %q is not a digit string: %v", timestamp, err)
	}
	var gotLine lokiLine
	if err := json.Unmarshal([]byte(line), &gotLine); err != nil {
		t.Fatalf("failed to unmarshal line: %v", err)
	}
	if gotLine.Identity == nil || *gotLine.Identity != identity {
		t.Fatalf("got identity %v, want %q", gotLine.Identity, identity)
	}
	if len(gotLine.Extra) != 0 {
		t.Fatalf("got extra %v, want empty", gotLine.Extra)
	}
}

func TestSendEventIncludesExtraFieldsInTheLine(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := NewGrafanaCloudLogger(server.URL, testLokiUser, testLokiAPIKey, "waker-api")
	logger.SendEvent("reachability_changed", "reachable", nil, map[string]any{"previous_state": "unreachable"})

	var payload lokiPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	line := payload.Streams[0].Values[0][1]
	var gotLine lokiLine
	if err := json.Unmarshal([]byte(line), &gotLine); err != nil {
		t.Fatalf("failed to unmarshal line: %v", err)
	}
	if gotLine.Identity != nil {
		t.Fatalf("got identity %v, want nil", gotLine.Identity)
	}
	if gotLine.Extra["previous_state"] != "unreachable" {
		t.Fatalf("got extra %v, want previous_state=unreachable", gotLine.Extra)
	}
}

func TestSendEventLogsUnexpectedStatusCode(t *testing.T) {
	// A wrong Loki URL/user/API key still gets a response, not a transport
	// error -- this must be surfaced too, not just connection failures.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid credentials"))
	}))
	defer server.Close()

	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	logger := NewGrafanaCloudLogger(server.URL, testLokiUser, testLokiAPIKey, "waker-api")
	logger.SendEvent("wake_requested", "success", nil, nil)

	got := logs.String()
	if !strings.Contains(got, "401") || !strings.Contains(got, "invalid credentials") {
		t.Fatalf("got log output %q, want it to mention status 401 and the response body", got)
	}
}

func TestSendEventDoesNotPanicWhenGrafanaCloudIsUnreachable(t *testing.T) {
	// A logging failure must never take down the caller's request path.
	logger := NewGrafanaCloudLogger("http://127.0.0.1:1", testLokiUser, testLokiAPIKey, "waker-api")
	identity := "nicola@example.com"
	logger.SendEvent("wake_requested", "success", &identity, nil)
}
