package llmprovider

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// workspaceFixture is a reply in testdata/anthropic-workspace, measured or,
// for not-found.json, written from Anthropic's documentation (0064-PLAN
// Phase 1).
type workspaceFixture struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

func loadWorkspaceFixture(t *testing.T, name string) workspaceFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "anthropic-workspace", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var f workspaceFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return f
}

// TestWorkspaceClasses_0064: Anthropic's workspace refusals classify to their
// sentinel, still match ErrInvalidRequest, and are terminal; the replies that
// listed are not errors (0064-MADR D3).
func TestWorkspaceClasses_0064(t *testing.T) {
	for _, c := range []struct {
		fixture     string
		want, other error
	}{
		{"no-workspace", ErrWorkspaceRequired, ErrWorkspaceNotFound},
		{"malformed", ErrWorkspaceNotFound, ErrWorkspaceRequired},
		{"not-found", ErrWorkspaceNotFound, ErrWorkspaceRequired},
	} {
		f := loadWorkspaceFixture(t, c.fixture)
		err := classifyFixture(string(ProviderClaude), f.Status, f.Body, nil)
		if !errors.Is(err, c.want) || !errors.Is(err, ErrInvalidRequest) || errors.Is(err, c.other) {
			t.Errorf("%s: %v; want %v, and so ErrInvalidRequest, and not %v", c.fixture, err, c.want, c.other)
		}
		if apiErr, ok := errors.AsType[*APIError](err); !ok || apiErr.Retryable() || apiErr.Status != f.Status {
			t.Errorf("%s: %#v; want a terminal *APIError with status %d", c.fixture, err, f.Status)
		}
	}
	// The sentinels themselves, apart from a reply's status: an APIError's
	// Unwrap also gives a 400's or 404's legacy ErrInvalidRequest.
	for _, s := range []error{ErrWorkspaceRequired, ErrWorkspaceNotFound} {
		if !errors.Is(s, ErrInvalidRequest) || (&APIError{Kind: s}).Retryable() {
			t.Errorf("%v: want a child of ErrInvalidRequest, never retried", s)
		}
	}
	for _, name := range []string{"workspace", "unassociated"} {
		f := loadWorkspaceFixture(t, name)
		if err := classifyFixture(string(ProviderClaude), f.Status, f.Body, nil); err != nil {
			t.Errorf("%s: %v; want nil for HTTP %d", name, err, f.Status)
		}
	}
}

// TestWorkspaceClasses_OnlyClaude_0064: the same replies from another service
// keep their status's class (0064-MADR D3).
func TestWorkspaceClasses_OnlyClaude_0064(t *testing.T) {
	for _, name := range []string{"no-workspace", "malformed", "not-found"} {
		f := loadWorkspaceFixture(t, name)
		for _, provider := range []string{string(ProviderOpenAI), string(ProviderKilo), "opencode/claude"} {
			err := classifyFixture(provider, f.Status, f.Body, nil)
			if errors.Is(err, ErrWorkspaceRequired) || errors.Is(err, ErrWorkspaceNotFound) || !errors.Is(err, ErrInvalidRequest) {
				t.Errorf("%s from %s: %v; want ErrInvalidRequest and no workspace class", name, provider, err)
			}
		}
	}
}

// TestWorkspaceClasses_OtherRefusals_0064: a Claude reply without both the
// type and the phrase of a workspace refusal keeps the class it has today.
func TestWorkspaceClasses_OtherRefusals_0064(t *testing.T) {
	body := func(kind, message string) string {
		raw, err := json.Marshal(map[string]any{"type": "error", "request_id": "req_REDACTED",
			"error": map[string]string{"type": kind, "message": message}})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	const notScoped = "This API key is not scoped to a workspace, so this request must include the anthropic-workspace-id header."
	for _, c := range []struct {
		name, kind, message string
		status              int
		want                error
	}{
		{"another 400", "invalid_request_error", "max_tokens: Field required", 400, ErrInvalidRequest},
		{"a model not found", "not_found_error", "model: claude-nope", 404, ErrInvalidRequest},
		{"another object not found", "not_found_error", "File file_011REDACTED not found.", 404, ErrInvalidRequest},
		{"a workspace, not a not-found", "not_found_error", "Workspace wrkspc_REDACTED is archived.", 404, ErrInvalidRequest},
		{"the not-found phrase, another type", "invalid_request_error", "Workspace wrkspc_REDACTED not found.", 404, ErrInvalidRequest},
		{"the not-scoped phrase, another type", "authentication_error", notScoped, 401, ErrAuthFailure},
		{"the malformed phrase, another type", "authentication_error", "anthropic-workspace-id header must be a valid workspace ID.", 401, ErrAuthFailure},
	} {
		err := classifyFixture(string(ProviderClaude), c.status, body(c.kind, c.message), nil)
		if errors.Is(err, ErrWorkspaceRequired) || errors.Is(err, ErrWorkspaceNotFound) || !errors.Is(err, c.want) {
			t.Errorf("%s: %v; want %v and no workspace class", c.name, err, c.want)
		}
	}
}
