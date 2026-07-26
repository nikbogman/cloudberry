// Package httpresponse provides small response-writing helpers.
package httpresponse

import (
	"encoding/json"
	"io"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Copy closes res.Body after copying it to w.
func Copy(w http.ResponseWriter, res *http.Response) {
	defer res.Body.Close()
	for key, values := range res.Header {
		w.Header()[key] = values
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}
