package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// workspaceMalformed is the malformed workspace 0064-PLAN Phase 1 sends.
const workspaceMalformed = "not-a-workspace"

// workspaceCase is one of 0064-PLAN Phase 1's requests: GET /v1/models with
// an anthropic-workspace-id, or none. sent names what was sent, never the
// value.
type workspaceCase struct {
	name, sent, header string
}

// workspaceCases is Phase 1's requests: no workspace, a malformed one, a
// foreign one when foreign is set, and the key's own.
func workspaceCases(workspace, foreign string) []workspaceCase {
	cases := []workspaceCase{{"no-workspace", "none", ""}, {"malformed", "malformed", workspaceMalformed}}
	if foreign != "" {
		cases = append(cases, workspaceCase{"foreign", "foreign", foreign})
	}
	return append(cases, workspaceCase{"workspace", "workspace", workspace})
}

// workspaceRecord is what a case records (0064-PLAN Phase 1): the status,
// the body and the service's type and message, with every workspace ID and
// request ID redacted; whether the reply named a workspace; and the classes
// ClassifyHTTPError gives it today. Never the key.
type workspaceRecord struct {
	Case           string   `json:"case"`
	Sent           string   `json:"sent"`
	Status         int      `json:"status"`
	Type           string   `json:"type,omitempty"`
	Message        string   `json:"message,omitempty"`
	Body           string   `json:"body"`
	ReplyWorkspace string   `json:"reply_workspace_header"`
	Classes        []string `json:"classes,omitempty"`
}

// The identifiers a record redacts: workspace IDs (AGENTS.md's "Recorded
// fixtures"; 0064-PLAN's rules) and request IDs.
var (
	workspaceID = regexp.MustCompile(`wrkspc_[A-Za-z0-9]+`)
	requestID   = regexp.MustCompile(`req_[A-Za-z0-9]+`)
)

// redactWorkspace is s with each of secrets, and every workspace and request
// ID, replaced by its redaction.
func redactWorkspace(s string, secrets ...string) string {
	for _, v := range secrets {
		if v != "" {
			s = strings.ReplaceAll(s, v, "REDACTED")
		}
	}
	s = workspaceID.ReplaceAllString(s, "wrkspc_REDACTED")
	return requestID.ReplaceAllString(s, "req_REDACTED")
}

// workspaceClasses are the sentinels a reply's error is checked against.
var workspaceClasses = []struct {
	name string
	err  error
}{
	{"ErrInvalidRequest", llmprovider.ErrInvalidRequest},
	{"ErrAuthFailure", llmprovider.ErrAuthFailure},
	{"ErrNotPermitted", llmprovider.ErrNotPermitted},
	{"ErrProviderUnavailable", llmprovider.ErrProviderUnavailable},
	{"ErrRateLimited", llmprovider.ErrRateLimited},
}

// recordWorkspace sends c to base's GET /v1/models?limit=1 with key, as the
// Claude listing sends it, and records the reply. secrets are redacted from
// the record: the key and the workspace values sent.
func recordWorkspace(ctx context.Context, client *http.Client, base, key string, c workspaceCase,
	secrets ...string) (workspaceRecord, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models?limit=1", nil)
	if err != nil {
		return workspaceRecord{}, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.header != "" {
		req.Header.Set("anthropic-workspace-id", c.header)
	}
	resp, err := client.Do(req)
	if err != nil {
		return workspaceRecord{}, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return workspaceRecord{}, fmt.Errorf("%s: %w", c.name, err)
	}
	all := append([]string{key}, secrets...)
	r := workspaceRecord{Case: c.name, Sent: c.sent, Status: resp.StatusCode, Body: redactWorkspace(string(raw), all...),
		ReplyWorkspace: "absent"}
	if resp.Header.Get("anthropic-workspace-id") != "" {
		r.ReplyWorkspace = "present"
	}
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	if cerr := llmprovider.ClassifyHTTPError("claude", resp); cerr != nil {
		var apiErr *llmprovider.APIError
		if errors.As(cerr, &apiErr) {
			r.Type, r.Message = apiErr.Code, redactWorkspace(apiErr.Message, all...)
		}
		for _, class := range workspaceClasses {
			if errors.Is(cerr, class.err) {
				r.Classes = append(r.Classes, class.name)
			}
		}
	}
	return r, nil
}

// writeWorkspaceRecord writes r to dir as <case>.json.
func writeWorkspaceRecord(dir string, r workspaceRecord) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, r.Case+".json"), append(raw, '\n'), 0o600)
}

// TestWorkspaceRecord_Twin_0064 (0064-PLAN Phase 1): the recorder, against a
// local server answering as Anthropic did on 2026-10-25 (the MADR's evidence)
// and as its documentation says of a foreign workspace (E5): each case sends
// the header it names, or none; each record has its status, type, message and
// classes; the reply's workspace header is seen; and no record holds the key,
// a workspace ID or a request ID.
func TestWorkspaceRecord_Twin_0064(t *testing.T) {
	const (
		key       = "sk-ant-test-KEY0064"
		workspace = "wrkspc_01Good456"
		foreign   = "wrkspc_01Foreign789"
	)
	var (
		mu   sync.Mutex
		sent []string
	)
	refuse := func(w http.ResponseWriter, status int, kind, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "request_id": "req_011Twin",
			"error": map[string]string{"type": kind, "message": message}})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("anthropic-workspace-id")
		mu.Lock()
		sent = append(sent, got)
		mu.Unlock()
		switch got {
		case "":
			refuse(w, 400, "invalid_request_error", "This API key is not scoped to a workspace, so this request must "+
				"include the anthropic-workspace-id header with the ID of the workspace to use.")
		case workspaceMalformed:
			refuse(w, 400, "invalid_request_error", "anthropic-workspace-id header must be a valid workspace ID.")
		case foreign:
			refuse(w, 404, "not_found_error", "Workspace "+foreign+" not found.")
		default:
			w.Header().Set("anthropic-workspace-id", got)
			// A workspace ID the request did not send: only the pattern
			// redacts it.
			_, _ = io.WriteString(w, `{"data":[{"id":"claude-haiku-5-5","type":"model"}],"has_more":false,`+
				`"note":"wrkspc_01Other000"}`)
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	want := map[string]struct {
		status  int
		kind    string
		classes string
		reply   string
	}{
		"no-workspace": {400, "invalid_request_error", "ErrInvalidRequest", "absent"},
		"malformed":    {400, "invalid_request_error", "ErrInvalidRequest", "absent"},
		"foreign":      {404, "not_found_error", "ErrInvalidRequest", "absent"},
		"workspace":    {200, "", "", "present"},
	}
	for _, c := range workspaceCases(workspace, foreign) {
		r, err := recordWorkspace(context.Background(), srv.Client(), srv.URL, key, c, workspace, foreign)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeWorkspaceRecord(dir, r); err != nil {
			t.Fatal(err)
		}
		w := want[c.name]
		if r.Status != w.status || r.Type != w.kind || strings.Join(r.Classes, ",") != w.classes || r.ReplyWorkspace != w.reply {
			t.Errorf("%s: %+v; want status %d, type %q, classes %q, reply header %s", c.name, r, w.status, w.kind, w.classes, w.reply)
		}
	}
	if got := strings.Join(sent, ","); got != ","+workspaceMalformed+","+foreign+","+workspace {
		t.Errorf("headers sent = %q; want none, the malformed, the foreign and the workspace", got)
	}
	if n := len(workspaceCases(workspace, "")); n != 3 {
		t.Errorf("with no foreign workspace: %d cases; want 3", n)
	}
	// A secret outside the pattern, such as the key, is redacted by value.
	if got := redactWorkspace("echoed "+key+" and odd-workspace-9", key, "odd-workspace-9"); got != "echoed REDACTED and REDACTED" {
		t.Errorf("redactWorkspace = %q; want each secret redacted by value", got)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) != 4 {
		t.Fatalf("records = %v, %v; want 4", files, err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(filepath.Clean(f))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{key, workspace, foreign, "req_011Twin", "wrkspc_01Other000"} {
			if bytes.Contains(raw, []byte(secret)) {
				t.Errorf("%s holds %q", filepath.Base(f), secret)
			}
		}
	}
}
