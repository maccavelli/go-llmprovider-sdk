package llmprovider_test

// The offline twins of the person-gated live tests (0046-MADR D3): each runs
// its live test's checks (live_checks_test.go) against twinFake, so a planted
// break of a check is caught with no sign-in. Each twinFake setting is one of
// 0046-PLAN Phase 2's plants.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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

// identityFake is the services of TestLive_0076Identity, by path: Grok's
// issuer (its discovery, device form, refresh and revocation), OpenAI's
// refresh and revocation under /oauth, the ChatGPT backend's listing under
// /chatgpt and xAI's under /xai. Each setting is one of 0076-PLAN Phase 4's
// plants.
type identityFake struct {
	srv *httptest.Server

	originator  string // the originator of authorizeURL
	listed      bool   // each listing holds a model, else none
	revokeKills bool   // after a revocation, a refresh is refused invalid_grant

	mu      sync.Mutex
	revoked bool
}

func newIdentityFake(t *testing.T) *identityFake {
	t.Helper()
	f := &identityFake{originator: probeIdentityName, listed: true, revokeKills: true}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// opts point a listing at the fake, under path.
func (f *identityFake) opts(path string) []llmprovider.Option {
	return []llmprovider.Option{llmprovider.WithBaseURL(f.srv.URL + path), llmprovider.WithHTTPClient(f.srv.Client())}
}

// authorizeURL is an authorize URL of OpenAI's shape, at the fake, with the
// fake's originator. LoginBrowserOAuth cannot give it here: OpenAI's
// sign-in listens on the registered ports 1455 and 1457, which no offline
// test of this package takes; auth's TestIdentity_Originator_0076 pins what
// the SDK puts there.
func (f *identityFake) authorizeURL() string {
	query := url.Values{"client_id": {auth.DefaultOpenAIClientID}, "response_type": {"code"},
		"originator": {f.originator}}
	return f.srv.URL + "/oauth/authorize?" + query.Encode()
}

func (f *identityFake) serve(w http.ResponseWriter, r *http.Request) {
	u := f.srv.URL
	switch path := r.URL.Path; {
	case r.Method == http.MethodGet && path == "/.well-known/openid-configuration":
		writeJSON(w, http.StatusOK, map[string]any{"issuer": u, "authorization_endpoint": u + "/authorize",
			"token_endpoint": u + "/token", "device_authorization_endpoint": u + "/device",
			"revocation_endpoint": u + "/revoke", "jwks_uri": u + "/jwks"})
	case r.Method == http.MethodPost && path == "/device":
		writeJSON(w, http.StatusOK, map[string]any{"device_code": "twin-device-0076", "user_code": "TWIN-0076",
			"verification_uri": u + "/verify", "expires_in": 600, "interval": 5})
	case r.Method == http.MethodGet && path == "/chatgpt/models":
		models := []any{}
		if f.listed {
			models = append(models, map[string]any{"slug": "gpt-twin-0076", "visibility": "list"})
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": models})
	case r.Method == http.MethodGet && path == "/xai/models":
		data := []any{}
		if f.listed {
			data = append(data, map[string]any{"id": "grok-twin-0076", "object": "model"})
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
	case r.Method == http.MethodPost && (path == "/token" || path == "/oauth/token"):
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.revoked && f.revokeKills {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "twin-access-1", "refresh_token": "twin-refresh-1",
			"expires_in": 3600})
	case r.Method == http.MethodPost && (path == "/revoke" || path == "/oauth/revoke"):
		f.mu.Lock()
		f.revoked = true
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

// TestLiveTwin_Identity_0076 is TestLive_0076Identity's checks against
// identityFake. Its OpenAI half checks the fake's authorize URL, then uses
// and revokes a ChatGPT session whose token endpoint is the fake's. Its Grok
// half starts a device sign-in at the fake through identityRecorder, as the
// live half does, and checks the form it posted, then uses and revokes a
// Grok session whose issuer is the fake.
func TestLiveTwin_Identity_0076(t *testing.T) {
	t.Run("OpenAIBrowser", func(t *testing.T) {
		f := newIdentityFake(t)
		ctx := twinContext(t)
		if !checkAuthorizeIdentity(t, f.authorizeURL(), "originator", probeIdentityName) {
			t.FailNow()
		}
		session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
			TokenURL: f.srv.URL + "/oauth/token", Access: "twin-access", Refresh: "twin-refresh",
			Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultOpenAIClientID, HTTPClient: f.srv.Client()}
		checkIdentitySession(ctx, t, session, f.opts("/chatgpt")...)
		checkIdentityRevoked(ctx, t, session)
	})
	t.Run("GrokDevice", func(t *testing.T) {
		f := newIdentityFake(t)
		ctx := twinContext(t)
		rec := &identityRecorder{base: f.srv.Client().Transport}
		client := &http.Client{Transport: rec}
		login, err := auth.StartDeviceOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
			Issuer: f.srv.URL, HTTPClient: client, Identity: auth.Identity{Name: probeIdentityName}})
		if err != nil {
			t.Fatalf("StartDeviceOAuth: %v", err)
		}
		login.Cancel()
		checkDeviceReferrer(t, rec, probeIdentityName)
		session := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: f.srv.URL, TokenURL: f.srv.URL + "/token",
			Access: "twin-access", Refresh: "twin-refresh", Expiry: time.Now().Add(time.Hour),
			ClientID: auth.DefaultGrokOAuthClientID, HTTPClient: client}
		checkIdentitySession(ctx, t, session, f.opts("/xai")...)
		checkIdentityRevoked(ctx, t, session)
	})
}

// pasteTwinCode is a code no line may hold: it stands for the secret a
// person pastes.
const pasteTwinCode = "PLANTEDCODE-7f3a91"

// pasteTwinState is a state value no line may hold either.
const pasteTwinState = "PLANTEDSTATE-5c2e"

// TestLiveTwin_GrokPaste_0074 is TestLive_0074GrokPaste's checks: the file
// channel, the shape of a paste and the finding line, each offline.
func TestLiveTwin_GrokPaste_0074(t *testing.T) {
	old := pasteInterval
	pasteInterval = 10 * time.Millisecond
	t.Cleanup(func() { pasteInterval = old })

	t.Run("FileInTwoWrites", twinPasteTwoWrites)
	t.Run("ContextEndsFirst", twinPasteContextEnds)
	t.Run("NoNewlineWithinLimit", twinPasteOversized)
	t.Run("Shape", twinPasteShapes)
	t.Run("Finding", twinPasteFindings)
}

type pasteResult struct {
	line string
	err  error
}

// startPaste runs pasteFromFile on its own goroutine.
func startPaste(ctx context.Context, path string, rec *pasteRecord) chan pasteResult {
	done := make(chan pasteResult, 1)
	go func() {
		line, err := pasteFromFile(path, rec)(ctx)
		done <- pasteResult{line, err}
	}()
	return done
}

// twinPasteTwoWrites writes the paste in two writes, the first without its
// newline: nothing may be taken before the newline, and the file must be
// gone after the read.
func twinPasteTwoWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paste")
	rec := &pasteRecord{}
	done := startPaste(twinContext(t), path, rec)
	if err := os.WriteFile(path, []byte("first-half"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		t.Fatalf("pasteFromFile returned %q, %v before the newline was written", r.line, r.err)
	case <-time.After(150 * time.Millisecond):
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("-second-half\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		if r.err != nil || r.line != "first-half-second-half" {
			t.Fatalf("pasteFromFile = %q, %v; want the whole line and no error", r.line, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pasteFromFile did not return after the newline was written")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the paste file is still there after the read: %v", err)
	}
	if returned, shape := rec.get(); !returned || shape != "a bare code" {
		t.Errorf("record = %t, %q; want a bare code returned", returned, shape)
	}
}

// twinPasteContextEnds ends ctx before anything is written.
func twinPasteContextEnds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paste")
	rec := &pasteRecord{}
	ctx, cancel := context.WithCancel(context.Background())
	done := startPaste(ctx, path, rec)
	cancel()
	select {
	case r := <-done:
		if !errors.Is(r.err, context.Canceled) || r.line != "" {
			t.Errorf("pasteFromFile = %q, %v; want context.Canceled", r.line, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pasteFromFile did not return when ctx ended")
	}
	if returned, _ := rec.get(); returned {
		t.Error("the record says a value was returned")
	}
}

// twinPasteOversized gives the file more than its limit and no newline.
func twinPasteOversized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paste")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", pasteFileLimit+100)), 0o600); err != nil {
		t.Fatal(err)
	}
	r := <-startPaste(twinContext(t), path, &pasteRecord{})
	if r.err == nil || r.line != "" {
		t.Errorf("pasteFromFile returned a line of %d bytes, %v; want an error and no line", len(r.line), r.err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the oversized paste file is still there: %v", err)
	}
}

// twinPasteShapes is pasteShape over every shape, with a planted code and
// state that no result may hold.
func twinPasteShapes(t *testing.T) {
	cb := "http://127.0.0.1:50321/callback"
	for _, tc := range []struct{ raw, want string }{
		{"  \t", "an empty line"},
		{pasteTwinCode, "a bare code"},
		{cb + "?code=" + pasteTwinCode + "&state=" + pasteTwinState,
			"a callback URL with code and state (scheme http, a loopback host, path /callback)"},
		{cb + "?code=" + pasteTwinCode,
			"a URL with a code and no state (scheme http, a loopback host, path /callback)"},
		{cb + "?state=" + pasteTwinState,
			"a URL with a state and no code (scheme http, a loopback host, path /callback)"},
		{cb, "a URL with no code and no state (scheme http, a loopback host, path /callback)"},
		{cb + "?error=access_denied&state=" + pasteTwinState,
			"a URL with an error (scheme http, a loopback host, path /callback)"},
		{"https://accounts.x.ai/" + pasteTwinCode + "?code=" + pasteTwinCode + "&state=" + pasteTwinState,
			"a callback URL with code and state (scheme https, another host, another path)"},
		{pasteTwinCode + "://h/?code=" + pasteTwinCode + "&state=" + pasteTwinState,
			"a callback URL with code and state (scheme another scheme, another host, another path)"},
	} {
		got := pasteShape(tc.raw)
		if got != tc.want {
			t.Errorf("pasteShape = %q, want %q", got, tc.want)
		}
		if strings.Contains(got, pasteTwinCode) || strings.Contains(got, pasteTwinState) {
			t.Errorf("pasteShape holds the pasted value: %q", got)
		}
	}
}

// twinPasteFindings is pasteFinding over every outcome of the live test.
func twinPasteFindings(t *testing.T) {
	urlShape := pasteShape("http://127.0.0.1:50321/callback?code=" + pasteTwinCode + "&state=" + pasteTwinState)
	bare := pasteShape(pasteTwinCode)
	paste := func(shape string) *pasteRecord { return &pasteRecord{returned: true, shape: shape} }
	for _, tc := range []struct {
		name   string
		rec    *pasteRecord
		err    error
		want   string
		passed bool
	}{
		{"paste of a URL", paste(urlShape), nil, "FINDING sign-in: completed by a paste of " + urlShape, true},
		{"paste of a code", paste(bare), nil, "FINDING sign-in: completed by a paste of a bare code", true},
		{"loopback won", &pasteRecord{}, nil,
			"FINDING sign-in: completed by the loopback, not a paste; the browser reached the host, so this run decides nothing", false},
		{"xAI refused", paste(bare), errors.New("oauth: token exchange: status 400"),
			"FINDING sign-in: xAI refused the pasted a bare code: oauth: token exchange: status 400", false},
		{"state mismatch", paste(urlShape), errors.New("oauth: callback state mismatch"),
			"FINDING sign-in: the paste was refused before any request: " + urlShape + ": oauth: callback state mismatch", false},
		{"no code", paste(urlShape), errors.New("oauth: callback URL missing authorization code"),
			"FINDING sign-in: the paste was refused before any request: " + urlShape +
				": oauth: callback URL missing authorization code", false},
		{"empty", paste("an empty line"), errors.New("oauth: empty authorization code"),
			"FINDING sign-in: the paste was refused before any request: an empty line: oauth: empty authorization code", false},
		{"denied", paste(urlShape), fmt.Errorf("oauth: authorization failed: access_denied: %w", auth.ErrSignInDenied),
			"FINDING sign-in: the paste was refused before any request: " + urlShape +
				": oauth: authorization failed: access_denied: " + auth.ErrSignInDenied.Error(), false},
		{"nothing pasted", &pasteRecord{}, fmt.Errorf("oauth: browser login timed out: %w", context.DeadlineExceeded),
			"FINDING sign-in: nothing was pasted before the deadline", false},
		{"ended before a paste", &pasteRecord{}, errors.New("oauth: callback server: closed"),
			"FINDING sign-in: the sign-in ended before any paste: oauth: callback server: closed", false},
	} {
		line, passed := pasteFinding(tc.rec, tc.err)
		if line != tc.want || passed != tc.passed {
			t.Errorf("%s: pasteFinding = %q, %t; want %q, %t", tc.name, line, passed, tc.want, tc.passed)
		}
		if strings.Contains(line, pasteTwinCode) || strings.Contains(line, pasteTwinState) {
			t.Errorf("%s: the line holds the pasted value: %q", tc.name, line)
		}
	}
}
