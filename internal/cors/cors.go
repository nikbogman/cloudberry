// Package cors lets the UI, served from its own origin on Cloudflare Pages,
// call gateway and hostd directly from the browser.
package cors

import "net/http"

// Allow answers preflights and adds CORS headers for requests from origin,
// before auth runs, so the UI can read a 401 too. Other origins get no CORS
// headers and the browser blocks the response. An empty origin disables it.
func Allow(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin == "" || r.Header.Get("Origin") != origin {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			// The backends' *.ts.net names resolve to tailnet (100.x)
			// addresses, which Chrome may treat as a private network.
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
