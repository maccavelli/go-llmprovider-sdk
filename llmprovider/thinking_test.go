package llmprovider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureServer returns an httptest server that records the decoded JSON request body of
// the last call and replies with the given canned response body.
func captureServer(t *testing.T, lastBody *map[string]any, respBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		m := map[string]any{}
		_ = json.Unmarshal(raw, &m)
		*lastBody = m
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}
