// Package wiretest records the HTTP requests a provider sends, normalised so a
// recording is the same on every run and platform, and compares recordings
// with golden files. It is the recorder behind G-wire
// (docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md D12), and
// only tests import it.
package wiretest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

// Request is one recorded HTTP request. Header holds every header the server
// received except those in droppedHeaders, with User-Agent normalised by
// UserAgent. Body is the decoded JSON body, the raw text when it is not JSON,
// or nil when it is empty.
type Request struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Query  map[string][]string `json:"query,omitempty"`
	Header map[string]string   `json:"header"`
	Body   any                 `json:"body,omitempty"`
}

// Reply is a canned response. A zero Status is 200.
type Reply struct {
	Status int
	Header map[string]string
	Body   string
}

// Record is the content of one golden file: the requests a scenario sent,
// and what the provider returned from the canned replies.
type Record struct {
	Requests []Request `json:"requests"`
	Result   any       `json:"result,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// droppedHeaders vary with the body's encoding, not with what was asked.
var droppedHeaders = map[string]bool{"Content-Length": true}

var (
	// userAgentPlatform is the "(goos; goarch)" part of a User-Agent.
	userAgentPlatform = regexp.MustCompile(`\([^;()]+; [^;()]+\)`)
	// userAgentVersion is the version after a product name, including the
	// "(devel)" of an untagged build.
	userAgentVersion = regexp.MustCompile(`/(?:\([^\s()]*\)|[^\s()]+)`)
)

// UserAgent replaces the version text and the platform in a User-Agent with
// placeholders, keeping its structure: "app/1.2 (linux; amd64) sdk/v1.0.0"
// becomes "app/<version> (<os>; <arch>) sdk/<version>".
func UserAgent(ua string) string {
	ua = userAgentPlatform.ReplaceAllString(ua, "(<os>; <arch>)")
	return userAgentVersion.ReplaceAllString(ua, "/<version>")
}

// Normalize records r, whose body has already been read into body.
func Normalize(r *http.Request, body []byte) Request {
	header := make(map[string]string, len(r.Header))
	for name, values := range r.Header {
		if droppedHeaders[name] {
			continue
		}
		value := strings.Join(values, ", ")
		if name == "User-Agent" {
			value = UserAgent(value)
		}
		header[name] = value
	}
	var query map[string][]string
	if q := r.URL.Query(); len(q) > 0 {
		query = q
	}
	return Request{Method: r.Method, Path: r.URL.Path, Query: query, Header: header, Body: decodeBody(body)}
}

// decodeBody decodes one JSON value, keeping numbers exact. Anything else is
// kept as text.
func decodeBody(body []byte) any {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil || dec.More() {
		return string(body)
	}
	return v
}

// Server is an httptest.Server that records every request and answers it with
// the Reply its function chooses.
type Server struct {
	// URL is the server's base URL.
	URL string

	mu       sync.Mutex
	requests []Request
}

// NewServer starts a Server that answers with reply. It is closed when t ends.
func NewServer(t testing.TB, reply func(r *http.Request) Reply) *Server {
	t.Helper()
	s := &Server{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("wiretest: read request body: %v", err)
		}
		s.mu.Lock()
		s.requests = append(s.requests, Normalize(r, body))
		s.mu.Unlock()
		answer := reply(r)
		for name, value := range answer.Header {
			w.Header().Set(name, value)
		}
		if answer.Status == 0 {
			answer.Status = http.StatusOK
		}
		w.WriteHeader(answer.Status)
		if _, err := io.WriteString(w, answer.Body); err != nil {
			t.Errorf("wiretest: write reply: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

// Requests returns the recorded requests in a stable order, sorted by their
// encoding: concurrent requests, such as a listing's probes, arrive in any
// order.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	type keyed struct {
		key string
		req Request
	}
	sorted := make([]keyed, len(s.requests))
	for i, req := range s.requests {
		key, err := json.Marshal(req)
		if err != nil {
			// A body decoded from JSON always encodes; fall back to a stable text anyway.
			key = []byte(fmt.Sprintf("%+v", req))
		}
		sorted[i] = keyed{string(key), req}
	}
	slices.SortStableFunc(sorted, func(a, b keyed) int { return strings.Compare(a.key, b.key) })
	out := make([]Request, len(sorted))
	for i, k := range sorted {
		out[i] = k.req
	}
	return out
}

// Encode renders v as golden text: indented JSON with sorted keys, HTML left
// unescaped, ending in a newline.
func Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Compare returns the differences between got and the golden file at path,
// one "field: want X, got Y" line each. With update it writes got to path
// instead, and returns none.
func Compare(path string, got any, update bool) ([]string, error) {
	enc, err := Encode(got)
	if err != nil {
		return nil, fmt.Errorf("wiretest: encode %s: %w", path, err)
	}
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return nil, err
		}
		return nil, os.WriteFile(path, enc, 0o600)
	}
	want, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("wiretest: no golden file %s; record one with -update", path)
	}
	if err != nil {
		return nil, err
	}
	// A Windows checkout may have converted the file's line endings.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if bytes.Equal(want, enc) {
		return nil, nil
	}
	wantValue, err := decodeJSON(want)
	if err != nil {
		return nil, fmt.Errorf("wiretest: golden file %s: %w", path, err)
	}
	gotValue, err := decodeJSON(enc)
	if err != nil {
		return nil, err
	}
	if diffs := Diff(wantValue, gotValue); len(diffs) > 0 {
		return diffs, nil
	}
	return []string{"(root): same value, different text; the golden file was edited by hand"}, nil
}

// Check is Compare, reported through t.
func Check(t testing.TB, path string, got any, update bool) {
	t.Helper()
	diffs, err := Compare(path, got, update)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) > 0 {
		t.Errorf("%s: %d difference(s) from the golden file; regenerate it only for a difference a record explains:\n  %s",
			path, len(diffs), strings.Join(diffs, "\n  "))
	}
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

// Diff lists where two decoded JSON values differ, by path: "requests[0].body.store:
// want false, got true". A key present on one side only is reported as
// "(absent)" on the other.
func Diff(want, got any) []string {
	var out []string
	diffAt("", want, got, &out)
	return out
}

func diffAt(path string, want, got any, out *[]string) {
	switch w := want.(type) {
	case map[string]any:
		if g, ok := got.(map[string]any); ok {
			keys := make([]string, 0, len(w)+len(g))
			for k := range w {
				keys = append(keys, k)
			}
			for k := range g {
				if _, seen := w[k]; !seen {
					keys = append(keys, k)
				}
			}
			slices.Sort(keys)
			for _, k := range keys {
				wv, wok := w[k]
				gv, gok := g[k]
				switch {
				case !wok:
					*out = append(*out, fmt.Sprintf("%s: want (absent), got %s", join(path, k), show(gv)))
				case !gok:
					*out = append(*out, fmt.Sprintf("%s: want %s, got (absent)", join(path, k), show(wv)))
				default:
					diffAt(join(path, k), wv, gv, out)
				}
			}
			return
		}
	case []any:
		if g, ok := got.([]any); ok {
			for i := range max(len(w), len(g)) {
				at := fmt.Sprintf("%s[%d]", path, i)
				switch {
				case i >= len(w):
					*out = append(*out, fmt.Sprintf("%s: want (absent), got %s", at, show(g[i])))
				case i >= len(g):
					*out = append(*out, fmt.Sprintf("%s: want %s, got (absent)", at, show(w[i])))
				default:
					diffAt(at, w[i], g[i], out)
				}
			}
			return
		}
	}
	if show(want) != show(got) {
		if path == "" {
			path = "(root)"
		}
		*out = append(*out, fmt.Sprintf("%s: want %s, got %s", path, show(want), show(got)))
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// show renders one value compactly for a difference line.
func show(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprint(v)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
