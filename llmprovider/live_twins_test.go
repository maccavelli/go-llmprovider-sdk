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
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
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

// TestLiveTwin_OpenAIDevice_0074 is TestLive_OpenAIDeviceLogin's checks, on a
// device login's ChatGPT session whose token endpoint is the fake's. Each
// marked line is one of 0074-PLAN Phase 7's plants.
func TestLiveTwin_OpenAIDevice_0074(t *testing.T) {
	f := newTwinFake(t)
	f.echo = true        // twin 0074 device: the fake echoes the word
	f.revokeKills = true // twin 0074 device: a refresh after revocation is refused
	deviceRefresh := "twin-device-refresh"
	deviceIssuer := auth.DefaultOpenAIIssuer // twin 0074 device: a ChatGPT session's issuer
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: deviceIssuer, TokenURL: f.srv.URL + "/oauth/token",
		Access: "twin-access", Refresh: deviceRefresh, Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultOpenAIClientID,
		HTTPClient: f.srv.Client()}
	checkDeviceChatGPT(twinContext(t), t, session, "gpt-6-astra", noSkip, f.opts("")...)
}

// scriptRun is a sign-in that has already emitted its steps, as the engine's
// channel does, and keeps the answers it is given.
type scriptRun struct {
	steps chan authcontract.Step

	mu      sync.Mutex
	answers []authcontract.Answer
}

func newScriptRun(steps ...authcontract.Step) *scriptRun {
	r := &scriptRun{steps: make(chan authcontract.Step, len(steps))}
	for _, s := range steps {
		r.steps <- s
	}
	close(r.steps)
	return r
}

func (r *scriptRun) Steps() <-chan authcontract.Step { return r.steps }

func (r *scriptRun) Answer(_ context.Context, a authcontract.Answer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.answers = append(r.answers, a)
	return nil
}

// twinAuthorizeURL is an authorize URL of the plan's shape whose redirect_uri
// names port.
func twinAuthorizeURL(port int) string {
	query := url.Values{"redirect_uri": {fmt.Sprintf("http://127.0.0.1:%d/auth/callback", port)}, "state": {pasteTwinState}}
	return "https://auth.twin.test/oauth/authorize?" + query.Encode()
}

// TestLiveTwin_PasteSteps_0074 is the checks of TestLive_0074PasteBrowser and
// TestLive_0074PastePlan: the step driver, the findings, the redirect port,
// the held port, the refresh and the client reuse, each offline.
func TestLiveTwin_PasteSteps_0074(t *testing.T) {
	t.Run("Drive", twinDriveSteps)
	t.Run("DriveNoPaste", twinDriveNoPaste)
	t.Run("SignInFinding", twinSignInFindings)
	t.Run("LoopbackFinding", twinLoopbackFindings)
	t.Run("RedirectPort", twinRedirectPort)
	t.Run("HoldPort", twinHoldPort)
	t.Run("SameClient", twinSameClient)
	t.Run("RefreshOnce", twinRefreshOnce)
}

// twinDriveSteps drives the plan's steps: a confirm, the authorize URL, a
// paste, and done. The confirm is answered yes only when the hook says yes;
// each answer names its own step.
func twinDriveSteps(t *testing.T) {
	steps := []authcontract.Step{{Seq: 1, Kind: authcontract.StepConfirm, ConfirmID: "twin", Text: "consent text"},
		{Seq: 2, Kind: authcontract.StepOpenURL, URL: twinAuthorizeURL(1455)},
		{Seq: 3, Kind: authcontract.StepPaste}, {Seq: 4, Kind: authcontract.StepDone}}
	pasted := "http://127.0.0.1:1455/auth/callback?code=" + pasteTwinCode + "&state=" + pasteTwinState
	for _, tc := range []struct {
		name    string
		confirm func(authcontract.Step) bool
		want    bool
	}{
		{"yes", func(authcontract.Step) bool { return true }, true},
		{"no", func(authcontract.Step) bool { return false }, false},
		{"no hook", nil, false},
	} {
		run := newScriptRun(steps...)
		var shown []string
		kinds, err := driveSteps(twinContext(t), run, stepHooks{
			confirm: tc.confirm,
			openURL: func(step authcontract.Step) { shown = append(shown, step.URL) },
			paste:   func(context.Context) (string, error) { return pasted, nil },
		})
		if err != nil || stepKinds(kinds) != "confirm, open_url, paste, done" {
			t.Fatalf("%s: driveSteps = %q, %v", tc.name, stepKinds(kinds), err)
		}
		if len(shown) != 1 || shown[0] != twinAuthorizeURL(1455) {
			t.Errorf("%s: the URL was shown as %q; want it once", tc.name, shown)
		}
		if len(run.answers) != 2 {
			t.Fatalf("%s: %d answers; want a confirm and a paste", tc.name, len(run.answers))
		}
		if c := run.answers[0]; c.Seq != 1 || c.Confirm == nil || *c.Confirm != tc.want || c.Paste != "" {
			t.Errorf("%s: confirm answer = %+v; want step 1, confirm %t", tc.name, c, tc.want)
		}
		if p := run.answers[1]; p.Seq != 3 || p.Paste != pasted || p.Confirm != nil {
			t.Errorf("%s: paste answer for step %d; want step 3 with the pasted line", tc.name, p.Seq)
		}
	}
}

// twinDriveNoPaste: a paste step the run does not take is an error, and no
// answer is given to it.
func twinDriveNoPaste(t *testing.T) {
	run := newScriptRun(authcontract.Step{Seq: 1, Kind: authcontract.StepOpenURL, URL: twinAuthorizeURL(1457)},
		authcontract.Step{Seq: 2, Kind: authcontract.StepPaste})
	kinds, err := driveSteps(twinContext(t), run, stepHooks{})
	if err == nil || stepKinds(kinds) != "open_url, paste" || len(run.answers) != 0 {
		t.Errorf("driveSteps = %q, %v, %d answers; want an error at the paste step and no answer", stepKinds(kinds), err, len(run.answers))
	}
}

// twinSignInFindings is signInFinding over every outcome.
func twinSignInFindings(t *testing.T) {
	kinds := []authcontract.StepKind{authcontract.StepOpenURL, authcontract.StepPaste, authcontract.StepDone}
	const steps = "; steps: open_url, paste, done"
	shape := pasteShape("http://127.0.0.1:1455/auth/callback?code=" + pasteTwinCode + "&state=" + pasteTwinState)
	paste := &pasteRecord{returned: true, shape: shape}
	for _, tc := range []struct {
		name   string
		rec    *pasteRecord
		err    error
		want   string
		passed bool
	}{
		{"paste", paste, nil, "FINDING sign-in: OpenAI sign-in completed by a paste of " + shape + steps, true},
		{"no paste", &pasteRecord{}, nil,
			"FINDING sign-in: OpenAI sign-in completed with no paste, which a sign-in with no listener cannot do" + steps, false},
		{"refused", paste, errors.New("oauth: token exchange: status 400"),
			"FINDING sign-in: OpenAI did not complete by the pasted " + shape + ": oauth: token exchange: status 400" + steps, false},
		{"deadline", &pasteRecord{}, fmt.Errorf("sign-in: %w", context.DeadlineExceeded),
			"FINDING sign-in: nothing was pasted before the deadline" + steps, false},
		{"ended", &pasteRecord{}, errors.New("oauth: discovery: status 503"),
			"FINDING sign-in: the sign-in ended before any paste: oauth: discovery: status 503" + steps, false},
	} {
		line, passed := signInFinding("OpenAI", tc.rec, kinds, tc.err)
		if line != tc.want || passed != tc.passed {
			t.Errorf("%s: signInFinding = %q, %t; want %q, %t", tc.name, line, passed, tc.want, tc.passed)
		}
		if strings.Contains(line, pasteTwinCode) || strings.Contains(line, pasteTwinState) {
			t.Errorf("%s: the line holds the pasted value: %q", tc.name, line)
		}
	}
}

// twinLoopbackFindings is loopbackFinding over both outcomes.
func twinLoopbackFindings(t *testing.T) {
	kinds := []authcontract.StepKind{authcontract.StepOpenURL, authcontract.StepDone}
	line, passed := loopbackFinding("OpenAI plan", 1457, kinds, nil)
	if want := "FINDING sign-in: the OpenAI plan sign-in completed by the loopback, which redirected to port 1457; steps: open_url, done"; !passed || line != want {
		t.Errorf("loopbackFinding = %q, %t; want %q, true", line, passed, want)
	}
	line, passed = loopbackFinding("OpenAI plan", 1457, kinds, errors.New("callback server: closed"))
	if want := "FINDING sign-in: the OpenAI plan sign-in by the loopback on port 1457 failed: callback server: closed; steps: open_url, done"; passed || line != want {
		t.Errorf("loopbackFinding = %q, %t; want %q, false", line, passed, want)
	}
}

// twinRedirectPort is redirectPort over the two registered ports and two
// URLs that name none.
func twinRedirectPort(t *testing.T) {
	for _, port := range []int{1455, 1457} {
		if got, err := redirectPort(twinAuthorizeURL(port)); err != nil || got != port {
			t.Errorf("redirectPort(%d) = %d, %v", port, got, err)
		}
	}
	for _, raw := range []string{"https://auth.twin.test/authorize?state=x", "https://auth.twin.test/authorize?redirect_uri=http%3A%2F%2F127.0.0.1%2Fcb"} {
		got, err := redirectPort(raw)
		if err == nil {
			t.Errorf("redirectPort(%q) = %d, nil; want an error", raw, got)
		} else if strings.Contains(err.Error(), "%!") {
			t.Errorf("redirectPort(%q) error %q holds a formatting fault", raw, err)
		}
	}
	// An unparsable redirect_uri wraps its cause.
	_, err := redirectPort("https://auth.twin.test/authorize?redirect_uri=http%3A%2F%2F%5B%3A%3A1")
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Errorf("redirectPort of an unparsable redirect_uri = %v; want it to wrap a *url.Error", err)
	} else if strings.Contains(err.Error(), "%!") {
		t.Errorf("redirectPort error %q holds a formatting fault", err)
	}
}

// twinHoldPort holds a free port, and checks that no one else can take it.
func twinHoldPort(t *testing.T) {
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	holdLoopbackPort(t, port)
	if l, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))); err == nil {
		_ = l.Close()
		t.Errorf("port %d was free after holdLoopbackPort", port)
	}
}

// twinSameClient is sameClient over a reuse, another client and another host.
func twinSameClient(t *testing.T) {
	a := &auth.OAuthSession{ClientID: "oaia_twin_a", HostID: "urn:uuid:twin-a"}
	for _, tc := range []struct {
		name string
		b    *auth.OAuthSession
		ok   bool
	}{
		{"reuse", &auth.OAuthSession{ClientID: "oaia_twin_a", HostID: "urn:uuid:twin-a"}, true},
		{"another client", &auth.OAuthSession{ClientID: "oaia_twin_b", HostID: "urn:uuid:twin-a"}, false},
		{"another host", &auth.OAuthSession{ClientID: "oaia_twin_a", HostID: "urn:uuid:twin-b"}, false},
		{"no client", &auth.OAuthSession{HostID: "urn:uuid:twin-a"}, false},
	} {
		if err := sameClient(a, tc.b); (err == nil) != tc.ok {
			t.Errorf("%s: sameClient = %v; want ok %t", tc.name, err, tc.ok)
		}
	}
	if err := sameClient(&auth.OAuthSession{HostID: "urn:uuid:twin-a"}, &auth.OAuthSession{HostID: "urn:uuid:twin-a"}); err == nil {
		t.Error("two sessions with no client are the same client")
	}
}

// twinRefreshOnce is checkRefreshOnce on a session whose token endpoint is
// the fake's: its refresh rotates the token.
func twinRefreshOnce(t *testing.T) {
	f := newTwinFake(t)
	f.rotate = true // twin 0074 refresh: a refresh returns new tokens
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		TokenURL: f.srv.URL + "/oauth/token", Access: "plan-access-0", Refresh: "plan-refresh-0",
		Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultOpenAIClientID, HTTPClient: f.srv.Client()}
	checkRefreshOnce(twinContext(t), t, session)
}

// TestLiveTwin_PlanReuse_0075 is TestLive_0075ClientReuse's checks: the
// authorize URL of a reuse, the reused client, the sign-out that keeps the
// client record, and the consent end to end, each offline. Each marked line
// is one of 0075-PLAN Phase 6's plants.
func TestLiveTwin_PlanReuse_0075(t *testing.T) {
	t.Run("AuthorizeURL", twinReuseAuthorizeURL)
	t.Run("ReusedClient", twinReusedClient)
	t.Run("LogoutKeepsClient", twinLogoutKeepsClient)
	t.Run("Consent", twinPlanConsent)
}

// twinReuseRecord is the client record the twins reuse.
var twinReuseRecord = auth.ClientRecord{ClientID: "oaia_twin_0075", HostID: "urn:uuid:00000000-0000-4000-8000-000000000075"}

// twinReuseURL is a reuse's authorize URL with set applied to its query.
func twinReuseURL(set func(url.Values)) string {
	query := url.Values{"client_id": {twinReuseRecord.ClientID}, "ext_agent_host_id": {twinReuseRecord.HostID},
		"redirect_uri": {"http://127.0.0.1:1455/auth/callback"}, "state": {pasteTwinState}}
	if set != nil {
		set(query)
	}
	return "https://auth.twin.test/oauth/authorize?" + query.Encode()
}

func twinReuseAuthorizeURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		ok   bool
	}{
		{"a reuse", twinReuseURL(nil), true},
		{"the agent name hint", twinReuseURL(func(q url.Values) { q.Set("agent_name_hint", "twin") }), false},
		{"the id token hint", twinReuseURL(func(q url.Values) { q.Set("id_token_hint", "twin-token") }), false},
		{"another client", twinReuseURL(func(q url.Values) { q.Set("client_id", "oaia_other") }), false},
		{"no client", twinReuseURL(func(q url.Values) { q.Del("client_id") }), false},
		{"another host id", twinReuseURL(func(q url.Values) { q.Set("ext_agent_host_id", "urn:uuid:other") }), false},
		{"unparsable", "https://auth.twin.test/%zz", false},
	} {
		err := reuseAuthorizeError(tc.raw, twinReuseRecord)
		if (err == nil) != tc.ok {
			t.Errorf("%s: reuseAuthorizeError = %v; want ok %t", tc.name, err, tc.ok)
		}
		if err != nil && strings.Contains(err.Error(), "%!") {
			t.Errorf("%s: the error holds a formatting fault: %q", tc.name, err)
		}
	}
}

func twinReusedClient(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    *auth.OAuthSession
		rec  auth.ClientRecord
		ok   bool
	}{
		{"reused", &auth.OAuthSession{ClientID: twinReuseRecord.ClientID, HostID: twinReuseRecord.HostID}, twinReuseRecord, true},
		{"another client", &auth.OAuthSession{ClientID: "oaia_other", HostID: twinReuseRecord.HostID}, twinReuseRecord, false},
		{"another host id", &auth.OAuthSession{ClientID: twinReuseRecord.ClientID, HostID: "urn:uuid:other"}, twinReuseRecord, false},
		{"no record", &auth.OAuthSession{}, auth.ClientRecord{}, false},
	} {
		err := reusedClientError(tc.s, tc.rec)
		if (err == nil) != tc.ok {
			t.Errorf("%s: reusedClientError = %v; want ok %t", tc.name, err, tc.ok)
		}
		if err != nil && strings.Contains(err.Error(), "%!") {
			t.Errorf("%s: the error holds a formatting fault: %q", tc.name, err)
		}
	}
}

// wipingStore is a FileTokenStore whose Delete also removes the client
// record: what 0075-MADR D7 forbids.
type wipingStore struct{ *auth.FileTokenStore }

func (w wipingStore) Delete(ctx context.Context, provider llmprovider.ProviderID) error {
	if err := w.FileTokenStore.Delete(ctx, provider); err != nil {
		return err
	}
	return w.SaveClient(ctx, provider, auth.ClientRecord{})
}

func twinLogoutKeepsClient(t *testing.T) {
	ctx := twinContext(t)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "twin-access", Refresh: "twin-refresh",
		Expiry: time.Now().Add(time.Hour), ClientID: twinReuseRecord.ClientID, HostID: twinReuseRecord.HostID}
	good, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := good.SaveClient(ctx, llmprovider.ProviderOpenAI, twinReuseRecord); err != nil {
		t.Fatal(err)
	}
	if err := logoutKeepsClientError(ctx, good, session); err != nil {
		t.Errorf("logoutKeepsClientError on a store that keeps the record = %v", err)
	}
	bad, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := bad.SaveClient(ctx, llmprovider.ProviderOpenAI, twinReuseRecord); err != nil {
		t.Fatal(err)
	}
	if err := logoutKeepsClientError(ctx, wipingStore{bad}, session); err == nil {
		t.Error("logoutKeepsClientError on a store whose Delete removes the record = nil; want an error")
	}
	empty, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := logoutKeepsClientError(ctx, empty, session); err == nil {
		t.Error("logoutKeepsClientError with no record = nil; want an error")
	}
}

// twinPlanConsent is checkPlanConsent on a plan session whose provider is the
// fake's.
func twinPlanConsent(t *testing.T) {
	f := newTwinFake(t)
	f.echo = true // twin 0075 consent: the fake echoes the word
	ctx := twinContext(t)
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: f.srv.URL, TokenURL: f.srv.URL + "/token",
		Access: "plan-access-0", Refresh: "plan-refresh-0", Expiry: time.Now().Add(time.Hour), ClientID: "oaia_twin",
		Resource: openai.PlatformBaseURL, Scope: "openid offline_access chatgpt.tokens.use.direct",
		HostID: "urn:uuid:00000000-0000-4000-8000-000000000075", IDToken: "twin-id-token", HTTPClient: f.srv.Client()}
	checkPlanConsent(ctx, t, session, store, "gpt-6.1-sol", f.srv.Client(), f.opts("")...)
}

// nameFake serves what TestLive_0079Name's two halves use, by path suffix: a
// Codex backend's and xAI's /models (one JSON holding both shapes),
// /responses, a token endpoint, discovery and a revocation endpoint. Each
// marked setting is one of 0079-PLAN Phase 5's plants.
type nameFake struct {
	srv  *httptest.Server
	body *twinFake // answers /responses: only its echo, deltas and tool are read

	refuseRefresh bool // a refresh is refused invalid_grant
	revokeKills   bool // after a revocation, a refresh is refused invalid_grant

	mu        sync.Mutex
	revoked   bool
	refreshes int
}

func newNameFake(t *testing.T) *nameFake {
	t.Helper()
	f := &nameFake{body: &twinFake{echo: true, deltas: true}, refuseRefresh: false, revokeKills: true}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// opts point a provider or a listing at the fake.
func (f *nameFake) opts() []llmprovider.Option {
	return []llmprovider.Option{llmprovider.WithBaseURL(f.srv.URL)}
}

func (f *nameFake) serve(w http.ResponseWriter, r *http.Request) {
	u, path := f.srv.URL, r.URL.Path
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/models"):
		writeJSON(w, http.StatusOK, map[string]any{"object": "list",
			"models": []any{map[string]any{"slug": "gpt-twin-0079", "visibility": "list"}},
			"data":   []any{map[string]any{"id": "grok-twin-0079", "object": "model"}}})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/responses"):
		f.body.responses(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/.well-known/openid-configuration"):
		writeJSON(w, http.StatusOK, map[string]any{"issuer": u, "authorization_endpoint": u + "/authorize",
			"token_endpoint": u + "/token", "revocation_endpoint": u + "/revoke", "jwks_uri": u + "/jwks"})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/token"):
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.refuseRefresh || (f.revoked && f.revokeKills) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
			return
		}
		f.refreshes++
		writeJSON(w, http.StatusOK, map[string]any{"access_token": fmt.Sprintf("name-access-%d", f.refreshes),
			"refresh_token": fmt.Sprintf("name-refresh-%d", f.refreshes), "expires_in": 3600})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/revoke"):
		f.mu.Lock()
		f.revoked = true
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

// newNameProbe is a probe of name against the fake, with a scratch store.
func newNameProbe(t *testing.T, f *nameFake) nameProbe {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := &nameRecorder{}
	return nameProbe{name: "gobble", rec: rec, client: &http.Client{Transport: rec.wrap(f.srv.Client().Transport)},
		opts: f.opts(), store: store}
}

// TestLiveTwin_Name_0079 is TestLive_0079Name's two halves after their
// sign-ins, run against nameFake with the shipped SDK code: a session loaded
// from a store has no identity, and the providers and the listing that are
// given the name give it to the session, so that its refresh and its
// revocation lead with the name, and the Codex backend's originator is the
// name. The sign-ins themselves are the live test's.
func TestLiveTwin_Name_0079(t *testing.T) {
	t.Run("ChatGPT", func(t *testing.T) {
		f := newNameFake(t)
		n := newNameProbe(t, f)
		signed := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
			TokenURL: f.srv.URL + "/oauth/token", Access: "name-access-0", Refresh: "name-refresh-0",
			Expiry: time.Now().Add(time.Hour), ClientID: auth.DefaultOpenAIClientID}
		runChatGPTName(twinContext(t), t, n, signed)
	})
	t.Run("Grok", func(t *testing.T) {
		f := newNameFake(t)
		n := newNameProbe(t, f)
		signed := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: f.srv.URL, TokenURL: f.srv.URL + "/token",
			Access: "name-access-0", Refresh: "name-refresh-0", Expiry: time.Now().Add(time.Hour),
			ClientID: auth.DefaultGrokOAuthClientID}
		runGrokName(twinContext(t), t, n, signed)
	})
	t.Run("Control", twinNameControl)
}

// twinNameControl is probeWithControl over its three outcomes.
func twinNameControl(t *testing.T) {
	named := []llmprovider.Option{llmprovider.WithClientInfo("gobble", "")}
	var control []llmprovider.Option
	boom := errors.New("backend: status 403")
	for _, tc := range []struct {
		name    string
		failsAs func(named bool) error // the call's error, named or not
		want    nameVerdict
		wantErr bool
	}{
		{"passes", func(bool) error { return nil }, namePassed, false},
		{"refused", func(isNamed bool) error {
			if isNamed {
				return boom
			}
			return nil
		}, nameRefused, true},
		{"fails both", func(bool) error { return boom }, nameFailed, true},
	} {
		got, err := probeWithControl(func(opts []llmprovider.Option) error { return tc.failsAs(len(opts) > 0) }, named, control)
		if got != tc.want || (err != nil) != tc.wantErr {
			t.Errorf("%s: probeWithControl = %q, %v; want %q, error %t", tc.name, got, err, tc.want, tc.wantErr)
		}
	}
	recs := []nameRecord{{product: "gobble", hasOriginator: true, originator: "gobble"}, {product: "gobble"}}
	for _, tc := range []struct {
		name string
		recs []nameRecord
		min  int
		want bool
		ok   bool
	}{
		{"named", recs, 2, true, true},
		{"too few", recs[:1], 2, true, false},
		{"another product", append(append([]nameRecord(nil), recs...), nameRecord{product: "go-llmprovider-sdk"}), 2, true, false},
		{"another originator", []nameRecord{{product: "gobble", hasOriginator: true, originator: "go-llmprovider-sdk"}}, 1, true, false},
		{"no originator when wanted", []nameRecord{{product: "gobble"}}, 1, true, false},
		{"no originator, not wanted", []nameRecord{{product: "gobble"}}, 1, false, true},
	} {
		err := namedRequestsError(tc.recs, "gobble", tc.min, tc.want)
		if (err == nil) != tc.ok {
			t.Errorf("%s: namedRequestsError = %v; want ok %t", tc.name, err, tc.ok)
		}
		if err != nil && strings.Contains(err.Error(), "%!") {
			t.Errorf("%s: the error holds a formatting fault: %q", tc.name, err)
		}
	}
	if got := firstProduct("gobble/1.2.3 (darwin; arm64) go-llmprovider-sdk/1.31.0"); got != "gobble" {
		t.Errorf("firstProduct = %q, want gobble", got)
	}
}
