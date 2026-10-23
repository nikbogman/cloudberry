package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const uiOrigin = "https://cloudberry.pages.dev"

func serve(origin, method string) *httptest.ResponseRecorder {
	h := Allow(uiOrigin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	req := httptest.NewRequest(method, "/wake", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAllowedOriginSeesTheResponse(t *testing.T) {
	rec := serve(uiOrigin, http.MethodPost)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Access-Control-Allow-Origin") != uiOrigin {
		t.Fatalf("got %d ACAO=%q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestPreflightFromAllowedOriginSkipsTheHandler(t *testing.T) {
	rec := serve(uiOrigin, http.MethodOptions)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("got %d methods=%q", rec.Code, rec.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestOtherOriginsGetNoCORSHeaders(t *testing.T) {
	for _, origin := range []string{"https://evil.example", ""} {
		rec := serve(origin, http.MethodOptions)
		if rec.Code != http.StatusUnauthorized || rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("origin %q: got %d ACAO=%q", origin, rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}
