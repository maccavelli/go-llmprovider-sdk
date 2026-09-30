package wiretest

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestUserAgent(t *testing.T) {
	for in, want := range map[string]string{
		"go-llmprovider-sdk/(devel) (darwin; arm64) go-llmprovider-sdk/(devel)": "go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>",
		"app/1.2.3 (linux; amd64) go-llmprovider-sdk/v1.0.0-rc.1":               "app/<version> (<os>; <arch>) go-llmprovider-sdk/<version>",
		"Go-http-client/1.1": "Go-http-client/<version>",
	} {
		if got := UserAgent(in); got != want {
			t.Errorf("UserAgent(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestServer_RecordsNormalisedRequests: the method, path, query, headers
// (without Content-Length, with User-Agent normalised) and the decoded body
// are recorded, and the canned reply is served.
func TestServer_RecordsNormalisedRequests(t *testing.T) {
	s := NewServer(t, func(r *http.Request) Reply {
		if r.URL.Path == "/missing" {
			return Reply{Status: http.StatusNotFound, Body: "no"}
		}
		return Reply{Header: map[string]string{"X-Reply": "1"}, Body: `{"ok":true}`}
	})
	req, err := http.NewRequest(http.MethodPost, s.URL+"/v1/chat?b=2&a=1",
		strings.NewReader(`{"model":"m","max_tokens":8192,"n":1.50}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "app/1.0 (linux; amd64) go-llmprovider-sdk/v1.0.0")
	req.Header.Set("Authorization", "Bearer k")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != `{"ok":true}` || resp.Header.Get("X-Reply") != "1" {
		t.Fatalf("reply = %d %q %v", resp.StatusCode, body, resp.Header)
	}
	resp, err = http.Get(s.URL + "/missing")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	got := s.Requests()
	if len(got) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(got))
	}
	get, post := got[0], got[1]
	if get.Method != http.MethodGet || get.Path != "/missing" || get.Body != nil || get.Query != nil {
		t.Errorf("GET recorded as %+v", get)
	}
	if post.Path != "/v1/chat" || !reflect.DeepEqual(post.Query, map[string][]string{"a": {"1"}, "b": {"2"}}) {
		t.Errorf("POST path/query = %q %v", post.Path, post.Query)
	}
	if _, ok := post.Header["Content-Length"]; ok {
		t.Error("Content-Length recorded")
	}
	if post.Header["User-Agent"] != "app/<version> (<os>; <arch>) go-llmprovider-sdk/<version>" || post.Header["Authorization"] != "Bearer k" {
		t.Errorf("headers = %v", post.Header)
	}
	enc, err := Encode(post.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"max_tokens\": 8192,\n  \"model\": \"m\",\n  \"n\": 1.50\n}\n"; string(enc) != want {
		t.Errorf("body encodes as %q, want %q (numbers exact, keys sorted)", enc, want)
	}
}

// TestServer_RequestsStableOrder: requests sent concurrently come back in one
// order however they arrived.
func TestServer_RequestsStableOrder(t *testing.T) {
	s := NewServer(t, func(*http.Request) Reply { return Reply{} })
	var wg sync.WaitGroup
	for _, path := range []string{"/c", "/a", "/b"} {
		wg.Go(func() {
			resp, err := http.Get(s.URL + path)
			if err != nil {
				t.Error(err)
				return
			}
			_ = resp.Body.Close()
		})
	}
	wg.Wait()
	var paths []string
	for _, r := range s.Requests() {
		paths = append(paths, r.Path)
	}
	if strings.Join(paths, " ") != "/a /b /c" {
		t.Errorf("order = %v, want /a /b /c", paths)
	}
}

func TestDecodeBody(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want any
	}{
		{"", nil},
		{"  \n", nil},
		{"grant_type=refresh_token", "grant_type=refresh_token"},
		{`{"a":1} {"b":2}`, `{"a":1} {"b":2}`},
		{`["x"]`, []any{"x"}},
	} {
		if got := decodeBody([]byte(tc.in)); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("decodeBody(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

// TestCompare covers the golden lifecycle: missing, written by update,
// matched, and changed, with each difference named by its path.
func TestCompare(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wire", "p", "text.json")
	rec := Record{Requests: []Request{{Method: "POST", Path: "/responses",
		Header: map[string]string{"Authorization": "Bearer k"},
		Body:   map[string]any{"store": false, "input": []any{"a", "b"}}}}, Result: "hello"}

	if _, err := Compare(path, rec, false); err == nil || !strings.Contains(err.Error(), "-update") {
		t.Fatalf("missing golden: err = %v, want one naming -update", err)
	}
	if diffs, err := Compare(path, rec, true); err != nil || diffs != nil {
		t.Fatalf("update: %v %v", diffs, err)
	}
	if diffs, err := Compare(path, rec, false); err != nil || diffs != nil {
		t.Fatalf("unchanged: %v %v", diffs, err)
	}

	rec.Requests[0].Body = map[string]any{"store": true, "input": []any{"a"}, "stream": true}
	rec.Result = nil
	rec.Error = "boom"
	diffs, err := Compare(path, rec, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`error: want (absent), got "boom"`,
		"requests[0].body.input[1]: want \"b\", got (absent)",
		"requests[0].body.store: want false, got true",
		"requests[0].body.stream: want (absent), got true",
		`result: want "hello", got (absent)`,
	}
	if !reflect.DeepEqual(diffs, want) {
		t.Errorf("diffs =\n  %s\nwant\n  %s", strings.Join(diffs, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestCompare_Errors(t *testing.T) {
	dir := t.TempDir()
	if _, err := Compare(filepath.Join(dir, "x.json"), func() {}, false); err == nil {
		t.Error("an unencodable value: want an error")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Compare(bad, map[string]any{}, false); err == nil || !strings.Contains(err.Error(), "golden file") {
		t.Errorf("corrupt golden: err = %v", err)
	}
	if _, err := Compare(dir, map[string]any{}, false); err == nil {
		t.Error("a directory as golden: want an error")
	}
	crlf := filepath.Join(dir, "crlf.json")
	if err := os.WriteFile(crlf, []byte("{\r\n  \"a\": 1\r\n}\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if diffs, err := Compare(crlf, map[string]any{"a": 1}, false); err != nil || diffs != nil {
		t.Errorf("CRLF golden: %v %v, want a match", diffs, err)
	}
	hand := filepath.Join(dir, "hand.json")
	if err := os.WriteFile(hand, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	diffs, err := Compare(hand, map[string]any{"a": 1}, false)
	if err != nil || len(diffs) != 1 || !strings.Contains(diffs[0], "edited by hand") {
		t.Errorf("reformatted golden: %v %v", diffs, err)
	}
}

func TestDiff_Shapes(t *testing.T) {
	for _, tc := range []struct {
		want, got any
		diffs     []string
	}{
		{"a", "a", nil},
		{"a", "b", []string{`(root): want "a", got "b"`}},
		{map[string]any{"k": "v"}, []any{"v"}, []string{`(root): want {"k":"v"}, got ["v"]`}},
		{[]any{"x"}, []any{"x", "y"}, []string{`[1]: want (absent), got "y"`}},
		{[]any{"<b>"}, []any{"<i>"}, []string{`[0]: want "<b>", got "<i>"`}},
	} {
		if got := Diff(tc.want, tc.got); !reflect.DeepEqual(got, tc.diffs) {
			t.Errorf("Diff(%v, %v) = %q, want %q", tc.want, tc.got, got, tc.diffs)
		}
	}
	if got := show(func() {}); !strings.HasPrefix(got, "0x") {
		t.Errorf("show(func) = %q, want the fmt form", got)
	}
}

func TestCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "g.json")
	Check(t, path, Record{Result: "ok"}, true)
	Check(t, path, Record{Result: "ok"}, false)
}
