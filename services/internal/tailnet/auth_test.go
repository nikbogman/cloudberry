package tailnet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func writeIdentityJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"identity": GetCallerIdentity(r)})
}

func protectedHandler() http.Handler {
	return RequireTailnetIdentity(http.HandlerFunc(writeIdentityJSON))
}

func loopbackProtectedHandler() http.Handler {
	return RequireTailnetIdentityOrLoopback(http.HandlerFunc(writeIdentityJSON))
}

func TestRejectsRequestWithNoIdentityHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	protectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestRejectsRequestWithBlankIdentityHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(IdentityHeader, "   ")
	rec := httptest.NewRecorder()

	protectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestAllowsRequestWithValidIdentityHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(IdentityHeader, "nicola@example.com")
	rec := httptest.NewRecorder()

	protectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestGetCallerIdentityReturnsHeaderValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(IdentityHeader, "nicola@example.com")
	rec := httptest.NewRecorder()

	protectedHandler().ServeHTTP(rec, req)

	want := `{"identity":"nicola@example.com"}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}

func TestLoopbackAllowsARequestFrom127001WithNoIdentityHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/loopback-protected", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	loopbackProtectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
}

func TestLoopbackRejectsARequestFromElsewhereWithNoIdentityHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/loopback-protected", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()

	loopbackProtectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestLoopbackStillHonorsAValidIdentityHeaderFromElsewhere(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/loopback-protected", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set(IdentityHeader, "nicola@example.com")
	rec := httptest.NewRecorder()

	loopbackProtectedHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	want := `{"identity":"nicola@example.com"}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}
