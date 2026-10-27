package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0064-PLAN Phase 4: the wizard asks a Claude key not scoped to a workspace
// for one (0064-MADR D4, amendment "D4's pre-fill").

// The keys anthropic0064 knows: one scoped to its workspace, which lists
// whatever header it is sent, and one that needs a workspace.
const (
	scopedKey0064   = "sk-ant-scoped-0064"
	unscopedKey0064 = "sk-ant-unscoped-0064"
)

// workspaceID0064 is a well-formed workspace ID, which anthropic0064 answers
// with its 404 unless it is one the unscoped key may use.
var workspaceID0064 = regexp.MustCompile(`^wrkspc_[A-Za-z0-9]+$`)

// anthropic0064 is Anthropic's listing as measured on 2026-10-27, and, for an
// unknown workspace, as its documentation says (0064-MADR, amendment of
// 2026-10-27). usable are the workspaces the unscoped key may use. It records
// each listing's workspace, "-" for none.
type anthropic0064 struct {
	usable []string
	mu     sync.Mutex
	sent   []string
}

func (a *anthropic0064) client() *http.Client {
	reply := func(status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
	}
	refuse := func(status int, kind, message string) *http.Response {
		return reply(status, `{"type":"error","error":{"type":"`+kind+`","message":"`+message+`"},"request_id":"req_REDACTED"}`)
	}
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/v1/models") {
			return reply(http.StatusNotFound, `{}`), nil
		}
		workspace, named := r.Header.Get("anthropic-workspace-id"), r.Header.Values("anthropic-workspace-id") != nil
		a.mu.Lock()
		if named {
			a.sent = append(a.sent, workspace)
		} else {
			a.sent = append(a.sent, "-")
		}
		a.mu.Unlock()
		listed := reply(http.StatusOK, `{"data":[{"id":"claude-haiku-4-5"}],"has_more":false}`)
		switch key := r.Header.Get("x-api-key"); {
		case key == scopedKey0064:
			return listed, nil
		case key != unscopedKey0064:
			return refuse(http.StatusUnauthorized, "authentication_error", "invalid x-api-key"), nil
		case named && !workspaceID0064.MatchString(workspace):
			return refuse(http.StatusBadRequest, "invalid_request_error", "anthropic-workspace-id header must be a valid workspace ID."), nil
		case !named:
			return refuse(http.StatusBadRequest, "invalid_request_error", "This API key is not scoped to a workspace, so this "+
				"request must include the anthropic-workspace-id header with the ID of the workspace to use. Add the header, "+
				"or use an API key that is scoped to a workspace."), nil
		case !slices.Contains(a.usable, workspace):
			return refuse(http.StatusNotFound, "not_found_error", "Workspace `"+workspace+"` not found."), nil
		}
		return listed, nil
	})}
}

// listings is every listing's workspace so far, joined.
func (a *anthropic0064) listings() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return strings.Join(a.sent, ",")
}

// options0064 lists through a with a cache of t's own.
func options0064(t *testing.T, a *anthropic0064) Options {
	return Options{Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: a.client(),
		ProviderOptions: []llmprovider.Option{testCache(t)}}
}

// assertNoWorkspaceText fails t if anything shown holds Anthropic's
// messages, which a notice words by class and status instead (0056-MADR D3).
func assertNoWorkspaceText(t *testing.T, f *fakePrompter) {
	t.Helper()
	for _, s := range f.allText {
		for _, message := range []string{"not scoped to a workspace", "must be a valid workspace ID", "` not found", "req_"} {
			if strings.Contains(s, message) {
				t.Errorf("shown %q; want none of the service's message", s)
			}
		}
	}
}

// TestWorkspace_AskedAndListed_0064: a key needing a workspace is told so by
// class and status, asked for one, lists with it, and the Result keeps it;
// a wrong one, or a malformed one, is asked again, and the answer is trimmed.
func TestWorkspace_AskedAndListed_0064(t *testing.T) {
	for _, tc := range []struct {
		name     string
		inputs   []string
		listings string
		notices  []string
	}{
		{"the first answer", []string{"wrkspc_Good1"}, "-,wrkspc_Good1",
			[]string{"Claude (Anthropic) needs a workspace for this key (workspace required, HTTP 400)"}},
		{"asked again", []string{"not-a-workspace", "wrkspc_Wrong1", "  wrkspc_Good1\n"},
			"-,not-a-workspace,wrkspc_Wrong1,wrkspc_Good1",
			[]string{"(workspace required, HTTP 400)", "(workspace not found, HTTP 400)", "(workspace not found, HTTP 404)"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &anthropic0064{usable: []string{"wrkspc_Good1"}}
			f := newFake(t, fakePrompter{secrets: []string{unscopedKey0064}, inputs: tc.inputs})
			res, cat, err := ResolveCredential(context.Background(), f, options0064(t, a), llmprovider.ProviderClaude)
			if err != nil {
				t.Fatal(err)
			}
			if res.Workspace != "wrkspc_Good1" || res.APIKey != unscopedKey0064 || !cat.Live {
				t.Errorf("Result %v, live %v; want the workspace wrkspc_Good1 and the key, listed live", res, cat.Live)
			}
			if got := a.listings(); got != tc.listings {
				t.Errorf("listings %q; want %q", got, tc.listings)
			}
			if len(f.seenInput) != len(tc.inputs) || f.seenInput[0] != workspacePrompt {
				t.Errorf("inputs %q; want the workspace prompt %d times", f.seenInput, len(tc.inputs))
			}
			for i, want := range tc.notices {
				if i >= len(f.seenNotify) || !strings.Contains(f.seenNotify[i], want) {
					t.Errorf("notices %q; want notice %d to hold %q", f.seenNotify, i, want)
				}
			}
			assertNoWorkspaceText(t, f)
		})
	}
}

// TestWorkspace_Bound_0064: three refused workspaces end the run with an
// error matching the last refusal's class, and no Result; a blank answer is
// sent as no workspace.
func TestWorkspace_Bound_0064(t *testing.T) {
	for _, tc := range []struct {
		name     string
		inputs   []string
		want     error
		listings string
	}{
		{"three wrong", []string{"wrkspc_Wrong1", "wrkspc_Wrong2", "wrkspc_Wrong3"}, llmprovider.ErrWorkspaceNotFound,
			"-,wrkspc_Wrong1,wrkspc_Wrong2,wrkspc_Wrong3"},
		{"three blank", []string{"", " ", ""}, llmprovider.ErrWorkspaceRequired, "-,-,-,-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &anthropic0064{usable: []string{"wrkspc_Good1"}}
			f := newFake(t, fakePrompter{secrets: []string{unscopedKey0064}, inputs: tc.inputs})
			res, _, err := ResolveCredential(context.Background(), f, options0064(t, a), llmprovider.ProviderClaude)
			if !errors.Is(err, tc.want) || !strings.Contains(err.Error(), "refused 3 workspaces entered") {
				t.Errorf("error %v; want %v, after 3 workspaces entered", err, tc.want)
			}
			if res.Provider != "" || res.APIKey != "" || res.Workspace != "" {
				t.Errorf("Result %v; want none", res)
			}
			if got := a.listings(); got != tc.listings {
				t.Errorf("listings %q; want %q", got, tc.listings)
			}
			assertNoWorkspaceText(t, f)
		})
	}
}

// TestWorkspace_ScopedKeyNeverAsked_0064: a key scoped to its workspace
// lists at once, and is never asked for one.
func TestWorkspace_ScopedKeyNeverAsked_0064(t *testing.T) {
	a := &anthropic0064{}
	f := newFake(t, fakePrompter{secrets: []string{scopedKey0064}})
	res, _, err := ResolveCredential(context.Background(), f, options0064(t, a), llmprovider.ProviderClaude)
	if err != nil || res.Workspace != "" || len(f.seenInput) != 0 || a.listings() != "-" {
		t.Errorf("Result %v, %v, inputs %q, listings %q; want no workspace, no prompt, one listing", res, err,
			f.seenInput, a.listings())
	}
}

// TestWorkspace_Defaults_0064: an existing Claude workspace is listed with
// first and kept, with no prompt; refused, it is the first prompt's default,
// and a later prompt has none. Without one, ANTHROPIC_WORKSPACE_ID is the
// default under AllowEnv, and nothing is without it. Another provider's
// existing workspace is neither (0064-MADR D4.3, amendment "D4's pre-fill").
func TestWorkspace_Defaults_0064(t *testing.T) {
	existing := func(workspace string) Result {
		return Result{Provider: llmprovider.ProviderClaude, APIKey: unscopedKey0064, Workspace: workspace}
	}
	env := func(k string) string {
		if k == envWorkspace {
			return " wrkspc_Good1 "
		}
		return ""
	}
	for _, tc := range []struct {
		name     string
		edit     func(*Options)
		script   fakePrompter
		listings string
	}{
		{"an existing workspace that lists",
			func(o *Options) { o.Existing = existing("wrkspc_Good1") },
			fakePrompter{confirms: []bool{true}}, "wrkspc_Good1"},
		{"an existing workspace refused",
			func(o *Options) { o.Existing = existing("wrkspc_Gone1") },
			fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault, acceptDefault, "wrkspc_Good1"}},
			"wrkspc_Gone1,wrkspc_Gone1,-,wrkspc_Good1"},
		{"the variable under AllowEnv",
			func(o *Options) { o.AllowEnv, o.LookupEnv = true, env },
			fakePrompter{secrets: []string{unscopedKey0064}, inputs: []string{acceptDefault}}, "-,wrkspc_Good1"},
		{"the variable without AllowEnv",
			func(o *Options) { o.LookupEnv = env },
			fakePrompter{secrets: []string{unscopedKey0064}, inputs: []string{acceptDefault, "wrkspc_Good1"}},
			"-,-,wrkspc_Good1"},
		{"another provider's existing workspace",
			func(o *Options) {
				o.Existing = Result{Provider: llmprovider.ProviderKilo, APIKey: "kilo-key", Workspace: "wrkspc_Good1"}
			},
			fakePrompter{secrets: []string{unscopedKey0064}, inputs: []string{acceptDefault, "wrkspc_Good1"}},
			"-,-,wrkspc_Good1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &anthropic0064{usable: []string{"wrkspc_Good1"}}
			o := options0064(t, a)
			tc.edit(&o)
			res, _, err := ResolveCredential(context.Background(), newFake(t, tc.script), o, llmprovider.ProviderClaude)
			if err != nil || res.Workspace != "wrkspc_Good1" {
				t.Errorf("Result %v, %v; want the workspace wrkspc_Good1", res, err)
			}
			if got := a.listings(); got != tc.listings {
				t.Errorf("listings %q; want %q", got, tc.listings)
			}
		})
	}
}

// TestWorkspace_ControlCharacter_0064: an answer holding a control character
// is not sent, and is asked again.
func TestWorkspace_ControlCharacter_0064(t *testing.T) {
	a := &anthropic0064{usable: []string{"wrkspc_Good1"}}
	f := newFake(t, fakePrompter{secrets: []string{unscopedKey0064}, inputs: []string{"wrkspc_\tGood1", "wrkspc_Good1"}})
	res, _, err := ResolveCredential(context.Background(), f, options0064(t, a), llmprovider.ProviderClaude)
	if err != nil || res.Workspace != "wrkspc_Good1" || a.listings() != "-,wrkspc_Good1" ||
		countContaining(f.seenNotify, "a workspace ID holds no control character") != 1 {
		t.Errorf("Result %v, %v, listings %q, notices %q; want the second answer, the first not sent", res, err,
			a.listings(), f.seenNotify)
	}
}

// TestWorkspace_NoDiscover_0064: with Discover off nothing is listed, and an
// existing Claude workspace is kept; another provider's Result has none.
func TestWorkspace_NoDiscover_0064(t *testing.T) {
	a := &anthropic0064{}
	o := options0064(t, a)
	o.Discover = false
	o.Existing = Result{Provider: llmprovider.ProviderClaude, APIKey: unscopedKey0064, Workspace: "wrkspc_Good1"}
	res, _, err := ResolveCredential(context.Background(), newFake(t, fakePrompter{confirms: []bool{true}}), o,
		llmprovider.ProviderClaude)
	if err != nil || res.Workspace != "wrkspc_Good1" || a.listings() != "" {
		t.Errorf("Result %v, %v, listings %q; want the existing workspace and no listing", res, err, a.listings())
	}
	if !strings.Contains(res.String(), "Workspace:wrkspc_Good1") {
		t.Errorf("String() = %q; want the workspace shown", res.String())
	}
	o = Options{Registry: registry0056(t, thirdPartyStatic, nil),
		Existing: Result{Provider: thirdPartyID, APIKey: "acme-key", Workspace: "wrkspc_Good1"}}
	res, _, err = ResolveCredential(context.Background(), newFake(t, fakePrompter{confirms: []bool{true}}), o, thirdPartyID)
	if err != nil || res.Workspace != "" {
		t.Errorf("a third party's Result %v, %v; want no workspace", res, err)
	}
}
