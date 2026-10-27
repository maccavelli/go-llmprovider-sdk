package llmprovider_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
)

// workspaceLiveModel is the model 0064-PLAN Phase 6 generates with: the
// cheapest Claude model Phase 1's listing named on 2026-10-27.
const workspaceLiveModel = "claude-haiku-5-5"

// workspaceLiveClasses are the classes a problem names.
var workspaceLiveClasses = []struct {
	name string
	err  error
}{
	{"ErrWorkspaceRequired", llmprovider.ErrWorkspaceRequired},
	{"ErrWorkspaceNotFound", llmprovider.ErrWorkspaceNotFound},
	{"ErrInvalidRequest", llmprovider.ErrInvalidRequest},
	{"ErrAuthFailure", llmprovider.ErrAuthFailure},
	{"ErrRateLimited", llmprovider.ErrRateLimited},
	{"ErrProviderUnavailable", llmprovider.ErrProviderUnavailable},
}

// workspaceKind names err by its classes and status, never its text, which
// can hold a workspace ID or a request ID.
func workspaceKind(err error) string {
	if err == nil {
		return "no error"
	}
	var names []string
	for _, c := range workspaceLiveClasses {
		if errors.Is(err, c.err) {
			names = append(names, c.name)
		}
	}
	status := 0
	if apiErr, ok := errors.AsType[*llmprovider.APIError](err); ok {
		status = apiErr.Status
	}
	return fmt.Sprintf("[%s] HTTP %d", strings.Join(names, ","), status)
}

// checkWorkspaceLive is 0064-PLAN Phase 6 step 1's checks, for a key not
// scoped to a workspace and a workspace it may use: with no workspace,
// generation and the listing are ErrWorkspaceRequired; with a malformed
// one, ErrWorkspaceNotFound; with the workspace, Generate and Stream answer
// and the listing is live. opts go to every provider and listing, after the
// check's own. It returns each problem, worded without the key, the
// workspace or the service's text; none is a pass.
func checkWorkspaceLive(ctx context.Context, key, workspace string, opts ...llmprovider.Option) []string {
	var problems []string
	fail := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	own := func(extra ...llmprovider.Option) []llmprovider.Option {
		return slices.Concat([]llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(workspaceLiveModel),
			llmprovider.WithMaxTokens(256), llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(),
			catalog.WithCache(new(catalog.Cache))}, opts, extra)
	}
	ask := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say ok."}}}
	list := func(workspace string) (catalog.Catalog, error) {
		cat, err := catalog.List(ctx, llmprovider.ProviderClaude, llmprovider.NewStaticToken(key),
			own(catalog.WithClaudeWorkspace(workspace))...)
		if err == nil && !cat.Live {
			err = cat.Err
		}
		return cat, err
	}
	for _, c := range []struct {
		name, workspace string
		want            error
	}{
		{"no workspace", "", llmprovider.ErrWorkspaceRequired},
		{"a malformed workspace", "not-a-workspace", llmprovider.ErrWorkspaceNotFound},
	} {
		p, err := claude.New(own(claude.WithWorkspace(c.workspace))...)
		if err != nil {
			fail("%s: New: %s", c.name, workspaceKind(err))
			continue
		}
		if _, err := p.Generate(ctx, ask); !errors.Is(err, c.want) || !errors.Is(err, llmprovider.ErrInvalidRequest) {
			fail("%s: Generate: %s; want %v, an invalid request", c.name, workspaceKind(err), c.want)
		}
		if _, err := list(c.workspace); !errors.Is(err, c.want) {
			fail("%s: List: %s; want %v", c.name, workspaceKind(err), c.want)
		}
	}
	p, err := claude.New(own(claude.WithWorkspace(workspace))...)
	if err != nil {
		fail("the workspace: New: %s", workspaceKind(err))
		return problems
	}
	if resp, err := p.Generate(ctx, ask); err != nil || resp.OutputText() == "" {
		fail("the workspace: Generate: %s; want an answer", workspaceKind(err))
	}
	var streamed strings.Builder
	var streamErr error
	for ev, err := range p.(llmprovider.Streamer).Stream(ctx, ask) {
		if err != nil {
			streamErr = err
			break
		}
		if ev.Type == llmprovider.EventTextDelta {
			streamed.WriteString(ev.Text)
		}
	}
	if streamErr != nil || streamed.Len() == 0 {
		fail("the workspace: Stream: %s; want text", workspaceKind(streamErr))
	}
	if cat, err := list(workspace); err != nil || !cat.Live || len(cat.Usable) == 0 {
		fail("the workspace: List: %s, live %v, %d usable; want a live listing", workspaceKind(err), cat.Live, len(cat.Usable))
	}
	return problems
}

// workspaceTwinStream is a Messages event stream answering "ok".
const workspaceTwinStream = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-haiku-5-5\"," +
	"\"usage\":{\"input_tokens\":3,\"output_tokens\":1}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// workspaceTwin is Anthropic for the twin: "anthropic" answers as measured on
// 2026-10-27 (0064-MADR, amendment of that date), "ignores" answers every
// request, "refuses" refuses every request for want of a workspace, and
// "unknown" refuses every workspace named with the documented 404, which
// echoes the ID (E5).
func workspaceTwin(t *testing.T, mode, workspace string) *httptest.Server {
	t.Helper()
	refuse := func(w http.ResponseWriter, status int, kind, message string) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"`+kind+`","message":"`+message+`"},"request_id":"req_REDACTED"}`)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, named := r.Header.Get("anthropic-workspace-id"), r.Header.Values("anthropic-workspace-id") != nil
		switch {
		case mode == "refuses", mode == "anthropic" && !named:
			refuse(w, http.StatusBadRequest, "invalid_request_error", "This API key is not scoped to a workspace, so this "+
				"request must include the anthropic-workspace-id header with the ID of the workspace to use.")
			return
		case mode == "anthropic" && got != workspace:
			refuse(w, http.StatusBadRequest, "invalid_request_error", "anthropic-workspace-id header must be a valid workspace ID.")
			return
		case mode == "unknown" && named:
			refuse(w, http.StatusNotFound, "not_found_error", "Workspace `"+got+"` not found.")
			return
		case mode == "unknown":
			refuse(w, http.StatusBadRequest, "invalid_request_error", "This API key is not scoped to a workspace.")
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/models"):
			_, _ = io.WriteString(w, `{"data":[{"id":"claude-haiku-5-5","type":"model"}],"has_more":false}`)
		case r.Header.Get("Accept") == "text/event-stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, workspaceTwinStream)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"model":"claude-haiku-5-5","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestWorkspaceLive_Twin_0064 (0064-PLAN Phase 6): checkWorkspaceLive passes
// against Anthropic as measured, and names each check that fails against a
// service that ignores the workspace, refuses every request, or knows no
// workspace, never the key or the workspace.
func TestWorkspaceLive_Twin_0064(t *testing.T) {
	const key, workspace = "sk-ant-twin-0064", "wrkspc_Twin0064"
	for _, tc := range []struct {
		mode string
		want []string
	}{
		{"anthropic", nil},
		{"ignores", []string{"no workspace: Generate: no error", "no workspace: List: no error",
			"a malformed workspace: Generate: no error", "a malformed workspace: List: no error"}},
		{"refuses", []string{"a malformed workspace: Generate: [ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400",
			"a malformed workspace: List: [ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400",
			"the workspace: Generate: [ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400",
			"the workspace: Stream: [ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400",
			"the workspace: List: [ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400"}},
		{"unknown", []string{"the workspace: Generate: [ErrWorkspaceNotFound,ErrInvalidRequest] HTTP 404",
			"the workspace: Stream: [ErrWorkspaceNotFound,ErrInvalidRequest] HTTP 404",
			"the workspace: List: [ErrWorkspaceNotFound,ErrInvalidRequest] HTTP 404"}},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			srv := workspaceTwin(t, tc.mode, workspace)
			problems := checkWorkspaceLive(context.Background(), key, workspace, llmprovider.WithBaseURL(srv.URL))
			if len(problems) != len(tc.want) {
				t.Fatalf("problems %q; want %d: %q", problems, len(tc.want), tc.want)
			}
			for i, want := range tc.want {
				if !strings.HasPrefix(problems[i], want) {
					t.Errorf("problem %d = %q; want it to begin %q", i, problems[i], want)
				}
			}
			for _, problem := range problems {
				if strings.Contains(problem, key) || strings.Contains(problem, workspace) || strings.Contains(problem, "req_") {
					t.Errorf("problem %q names the key, the workspace or a request", problem)
				}
			}
		})
	}
}
