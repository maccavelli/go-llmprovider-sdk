package llmprovider_test

// The offline twins of the person-gated live tests (0046-MADR D3): each runs
// its live test's checks (live_checks_test.go) against twinFake, so a planted
// break of a check is caught with no sign-in. Each twinFake setting is one of
// 0046-PLAN Phase 2's plants.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// twinFake answers the requests the checks send, by path suffix.
type twinFake struct {
	srv *httptest.Server

	echo          bool     // reply with the word the prompt asks for, else OMEGA
	chatWord      bool     // a Chat Completions reply carries the word, else ""
	deltas        bool     // an event stream carries the text as deltas
	tool          bool     // a request with tools is answered with get_weather(Paris)
	rotate        bool     // a refresh returns new tokens
	revokeKills   bool     // after a revocation, a refresh is refused invalid_grant
	profileStatus int      // the Kilo profile's status
	models        []string // the plan listing's entries, each listed

	mu        sync.Mutex
	revoked   bool
	refreshes int
}

func newTwinFake(t *testing.T) *twinFake {
	t.Helper()
	f := &twinFake{echo: true, chatWord: true, deltas: true, tool: true, rotate: true, revokeKills: true,
		profileStatus: http.StatusOK, models: []string{"gpt-6.1-sol", "gpt-6-luna"}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// opts point a provider, a listing or a profile at the fake.
func (f *twinFake) opts(path string) []llmprovider.Option {
	return []llmprovider.Option{llmprovider.WithBaseURL(f.srv.URL + path), llmprovider.WithHTTPClient(f.srv.Client())}
}

var twinPrompt = regexp.MustCompile(`Reply with only the word ([A-Z]+)`)

// twinRequest is what the fake reads of a generation request.
type twinRequest struct {
	word  string
	tools bool
}

func (f *twinFake) read(r *http.Request) twinRequest {
	body, _ := io.ReadAll(r.Body)
	got := twinRequest{word: "hello", tools: strings.Contains(string(body), `"tools"`)}
	if m := twinPrompt.FindSubmatch(body); m != nil {
		got.word = string(m[1])
	}
	if !f.echo {
		got.word = "OMEGA"
	}
	return got
}

func (f *twinFake) serve(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/responses"):
		f.responses(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/chat/completions"):
		word := f.read(r).word
		if !f.chatWord {
			word = ""
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": "chat_twin", "choices": []any{map[string]any{"index": 0,
			"finish_reason": "stop", "message": map[string]any{"role": "assistant", "content": word}}}})
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/models"):
		var models []any
		for _, m := range f.models {
			models = append(models, map[string]any{"slug": m, "visibility": "list"})
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": models})
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/profile"):
		writeJSON(w, f.profileStatus, map[string]any{"organizations": []any{}, "hasPersonalAccount": true})
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/.well-known/openid-configuration"):
		u := f.srv.URL
		writeJSON(w, http.StatusOK, map[string]any{"issuer": u, "authorization_endpoint": u + "/authorize",
			"token_endpoint": u + "/token", "revocation_endpoint": u + "/revoke", "jwks_uri": u + "/jwks"})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/token"):
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.revoked && f.revokeKills {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
			return
		}
		if f.rotate {
			f.refreshes++
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": fmt.Sprintf("plan-access-%d", f.refreshes),
			"refresh_token": fmt.Sprintf("plan-refresh-%d", f.refreshes), "expires_in": 3600})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/revoke"):
		f.mu.Lock()
		f.revoked = true
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

// responses answers a Responses request: an event stream when it asks for
// one, as the ChatGPT backend and a plan session's requests do, else one JSON
// reply.
func (f *twinFake) responses(w http.ResponseWriter, r *http.Request) {
	req := f.read(r)
	message := map[string]any{"type": "message", "role": "assistant",
		"content": []any{map[string]any{"type": "output_text", "text": req.word}}}
	items := []any{message}
	if req.tools && f.tool {
		items = append(items, map[string]any{"type": "function_call", "call_id": "call_twin", "name": "get_weather",
			"arguments": `{"city":"Paris"}`})
	}
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		writeJSON(w, http.StatusOK, map[string]any{"id": "resp_twin", "status": "completed", "output": items})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	event := func(v map[string]any) {
		raw, _ := json.Marshal(v)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", v["type"], raw)
	}
	event(map[string]any{"type": "response.created", "response": map[string]any{"id": "resp_twin"}})
	if f.deltas {
		event(map[string]any{"type": "response.output_text.delta", "delta": req.word})
	}
	for _, item := range items {
		event(map[string]any{"type": "response.output_item.done", "item": item})
	}
	event(map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp_twin"}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func twinContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestLiveTwin_ChatGPT_0046 is TestLive_ChatGPTBrowserLogin's checks, on a
// ChatGPT session against the fake.
func TestLiveTwin_ChatGPT_0046(t *testing.T) {
	f := newTwinFake(t)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Access: "twin-access", Refresh: "twin-refresh", Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultOpenAIClientID}
	checkWord(twinContext(t), t, liveOpenAI(t, session, "gpt-6-astra", f.opts("")...), "ALPHA", noSkip)
}

// TestLiveTwin_Grok_0046 is the checks of TestLive_GrokBrowserLogin and
// TestLive_GrokDeviceLogin, which share them, on an xAI session.
func TestLiveTwin_Grok_0046(t *testing.T) {
	f := newTwinFake(t)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: auth.DefaultGrokOAuthIssuer,
		Access: "twin-access", Refresh: "twin-refresh", Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultGrokOAuthClientID}
	checkWord(twinContext(t), t, liveGrok(t, session, "grok-4.6", f.opts("")...), "ALPHA", noSkip)
}

// TestLiveTwin_Kilo_0046 is TestLive_KiloDeviceLogin's checks, on a device
// login's session: an access token alone.
func TestLiveTwin_Kilo_0046(t *testing.T) {
	f := newTwinFake(t)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: "twin-kilo"}
	checkKiloSession(twinContext(t), t, session, "twin-free-model", noSkip, f.opts("/api/gateway")...)
}

// TestLiveTwin_ChatGPTPlan_0046 is TestLive_ChatGPTPlan's checks after the
// sign-in, on a plan session whose issuer is the fake.
func TestLiveTwin_ChatGPTPlan_0046(t *testing.T) {
	f := newTwinFake(t)
	ctx := twinContext(t)
	signedIn := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: f.srv.URL, TokenURL: f.srv.URL + "/token",
		Access: "plan-access-0", Refresh: "plan-refresh-0", Expiry: time.Now().Add(time.Hour), ClientID: "oaia_twin",
		Resource: openai.PlatformBaseURL, Scope: "openid offline_access chatgpt.tokens.use.direct",
		HostID: "urn:uuid:00000000-0000-4000-8000-000000000046", IDToken: "twin-id-token", HTTPClient: f.srv.Client()}
	checkPlanKind(t, signedIn)
	session, store := planStoreRoundTrip(ctx, t, signedIn)
	planRequests(ctx, t, session, noSkip, f.opts("")...)
	planRefresh(ctx, t, session, store)
	planRevoke(ctx, t, session)
}
