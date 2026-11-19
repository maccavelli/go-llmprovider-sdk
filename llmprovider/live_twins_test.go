package llmprovider_test

// The offline twins of the person-gated live tests (0046-MADR D3): each runs
// its live test's checks (live_checks_test.go) against twinFake, so a planted
// break of a check is caught with no sign-in. Each twinFake setting is one of
// 0046-PLAN Phase 2's plants.

import (
	"bytes"
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
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
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
	t.Run("FileHeldOpenThenClosed", twinPasteHeldOpen)
	t.Run("OversizedHeldOpenThenClosed", twinPasteOversizedHeldOpen)
	t.Run("HeldOpenUntilContextEnds", twinPasteHeldOpenContextEnds)
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

// holdPasteFile creates path holding content and keeps the handle open, as a
// command that writes a paste and has not yet closed its file does. Go opens
// it without FILE_SHARE_DELETE, so on Windows the file cannot be removed
// until the handle is closed; elsewhere it can, and the hold changes nothing.
func holdPasteFile(t *testing.T, path, content string) *os.File {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	return f
}

// pasteHeldOpen runs pasteFromFile on a file whose writer holds it open,
// and closes it once the helper has had many looks at it. On Windows the
// helper must not return while the handle is open (its removal is refused);
// off Windows the file is removed at once, so only the result is checked.
func pasteHeldOpen(t *testing.T, content string) pasteResult {
	t.Helper()
	path := filepath.Join(t.TempDir(), "paste")
	f := holdPasteFile(t, path, content)
	done := startPaste(twinContext(t), path, &pasteRecord{})
	select {
	case r := <-done:
		if runtime.GOOS == "windows" {
			t.Fatalf("pasteFromFile returned %d bytes, %v while the file was held open", len(r.line), r.err)
		}
		_ = f.Close()
		return r
	case <-time.After(300 * time.Millisecond):
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var r pasteResult
	select {
	case r = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pasteFromFile did not return after the writer closed the file")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the paste file is still there after the read: %v", err)
	}
	return r
}

// twinPasteHeldOpen is the file a writer wrote whole but has not closed:
// the helper returns the line once the writer closes it, and the file is
// gone. This is the deterministic form of the flake of 0074-PLAN Phase 8.
// On Windows it fails without the retry; off Windows it proves the line and
// the removal only.
func twinPasteHeldOpen(t *testing.T) {
	r := pasteHeldOpen(t, "held-line\r\n")
	if r.err != nil || r.line != "held-line" {
		t.Fatalf("pasteFromFile = %q, %v; want the line and no error", r.line, r.err)
	}
}

// twinPasteOversizedHeldOpen is the same for the over-limit branch.
func twinPasteOversizedHeldOpen(t *testing.T) {
	r := pasteHeldOpen(t, strings.Repeat("x", pasteFileLimit+100))
	if r.err == nil || r.line != "" || !strings.Contains(r.err.Error(), "no newline") {
		t.Errorf("pasteFromFile = %d bytes, %v; want the no-newline error and no line", len(r.line), r.err)
	}
}

// twinPasteHeldOpenContextEnds is a file that stays held open past ctx: the
// helper returns ctx's error, with no line, and leaves the file. Windows only,
// since elsewhere the removal succeeds and nothing holds it.
func twinPasteHeldOpenContextEnds(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open file can be removed off Windows; nothing holds the file here")
	}
	path := filepath.Join(t.TempDir(), "paste")
	holdPasteFile(t, path, "held-line\r\n")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	rec := &pasteRecord{}
	r := <-startPaste(ctx, path, rec)
	if !errors.Is(r.err, context.DeadlineExceeded) || r.line != "" {
		t.Errorf("pasteFromFile = %q, %v; want the deadline error and no line", r.line, r.err)
	}
	if returned, _ := rec.get(); returned {
		t.Error("the record says a value was returned")
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

// noNetworkClient is a client whose every request fails, so that the plan's
// discovery fails and its built-in endpoints are used, which is how its
// authorize URL is built offline.
func noNetworkClient() *http.Client {
	return &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("twin: no network")
	})}
}

// capturedAuthorizeURL starts a sign-in with signin.Start, answers its
// confirms yes, takes the URL its open_url step shows, and cancels it: the
// URL is the shipped builder's own.
func capturedAuthorizeURL(t *testing.T, p llmprovider.ProviderID, m llmprovider.AuthMethodID, o signin.Options) string {
	t.Helper()
	ctx := twinContext(t)
	run, err := signin.Start(ctx, p, m, o)
	if err != nil {
		t.Fatalf("signin.Start: %v", err)
	}
	defer run.Cancel()
	var shown string
	_, _ = driveSteps(ctx, run, stepHooks{
		confirm: func(authcontract.Step) bool { return true },
		openURL: func(step authcontract.Step) {
			shown = step.URL
			run.Cancel()
		},
	})
	if shown == "" {
		t.Fatal("the sign-in showed no authorize URL")
	}
	return shown
}

// twinPlanSession is a plan session a sign-in can be made again to.
func twinPlanSession() *auth.OAuthSession {
	return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "twin-access", Refresh: "twin-refresh",
		Expiry: time.Now().Add(time.Hour), ClientID: "oaia_twin_authorize", Kind: auth.SessionChatGPTPlan,
		Resource: openai.PlatformBaseURL, Scope: "openid offline_access chatgpt.tokens.use.direct",
		HostID: "urn:uuid:00000000-0000-4000-8000-000000000074", IDToken: "twin-id-token-authorize"}
}

// TestLiveTwin_AuthorizeURLs_0074 runs every authorize-URL assertion of the
// live probes against the URL the shipped builders make, through
// signin.Start or auth.LoginBrowserOAuth with a fake issuer, so that a probe
// asserting a parameter its method never sends fails here, not in a live run.
func TestLiveTwin_AuthorizeURLs_0074(t *testing.T) {
	t.Run("PlanFirst", func(t *testing.T) {
		raw := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
			signin.Options{AgentName: pasteProbeAgentNameTwin, NoLoopback: true, HTTPClient: noNetworkClient()})
		if err := authorizeError(raw, planFirstAuthorizeWant(pasteProbeAgentNameTwin)); err != nil {
			t.Errorf("the plan's first sign-in URL: %v", err)
		}
		if port, err := redirectPort(raw); err != nil || port != 1455 {
			t.Errorf("redirectPort of the plan's NoLoopback URL = %d, %v; want 1455", port, err)
		}
	})
	t.Run("PlanAgain", func(t *testing.T) {
		previous := twinPlanSession()
		raw := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
			signin.Options{AgentName: pasteProbeAgentNameTwin, Previous: previous, NoLoopback: true, HTTPClient: noNetworkClient()})
		if err := authorizeError(raw, planAgainAuthorizeWant(previous)); err != nil {
			t.Errorf("the plan's sign-in again URL: %v", err)
		}
	})
	t.Run("PlanReuse", func(t *testing.T) {
		store, err := auth.NewFileTokenStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SaveClient(twinContext(t), llmprovider.ProviderOpenAI, twinReuseRecord); err != nil {
			t.Fatal(err)
		}
		raw := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
			signin.Options{AgentName: pasteProbeAgentNameTwin, ClientStore: store, NoLoopback: true, HTTPClient: noNetworkClient()})
		if err := reuseAuthorizeError(raw, twinReuseRecord); err != nil {
			t.Errorf("the plan's reuse URL: %v", err)
		}
	})
	t.Run("OpenAIBrowser", func(t *testing.T) {
		f := newNameFake(t)
		raw := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth,
			signin.Options{Issuer: f.srv.URL, NoLoopback: true, HTTPClient: f.srv.Client()})
		if err := authorizeError(raw, browserAuthorizeWant("originator", twinModuleName)); err != nil {
			t.Errorf("OpenAI's browser URL: %v", err)
		}
		named := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth,
			signin.Options{Issuer: f.srv.URL, NoLoopback: true, HTTPClient: f.srv.Client(), Identity: auth.Identity{Name: "gobble"}})
		if err := authorizeError(named, browserAuthorizeWant("originator", "gobble")); err != nil {
			t.Errorf("OpenAI's browser URL under an identity: %v", err)
		}
	})
	t.Run("GrokBrowser", func(t *testing.T) {
		f := newNameFake(t)
		ctx, cancel := context.WithCancel(twinContext(t))
		defer cancel()
		var shown string
		_, _ = auth.LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
			Issuer: f.srv.URL, HTTPClient: f.srv.Client(),
			OpenURL:   func(raw string) error { shown = raw; cancel(); return nil },
			InputCode: func(ctx context.Context) (string, error) { <-ctx.Done(); return "", ctx.Err() },
		})
		if shown == "" {
			t.Fatal("the sign-in showed no authorize URL")
		}
		if err := authorizeError(shown, browserAuthorizeWant("referrer", twinModuleName)); err != nil {
			t.Errorf("Grok's browser URL: %v", err)
		}
	})
	t.Run("OldPlanAssertion", func(t *testing.T) {
		// What TestLive_0074PastePlan asserted before the owner's live run:
		// an originator the plan never sends. The shipped URL must fail it.
		raw := capturedAuthorizeURL(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
			signin.Options{AgentName: pasteProbeAgentNameTwin, NoLoopback: true, HTTPClient: noNetworkClient()})
		if err := authorizeError(raw, browserAuthorizeWant("originator", twinModuleName)); err == nil {
			t.Error("the plan's URL carries an originator; the probe's old assertion would hold")
		} else if strings.Contains(err.Error(), "%!") {
			t.Errorf("the error holds a formatting fault: %q", err)
		}
		for name, want := range map[string]authorizeWant{
			"an absent parameter that is sent":   {absent: []string{"agent_name_hint"}},
			"a present parameter that is absent": {present: []string{"originator"}},
		} {
			if err := authorizeError(raw, want); err == nil {
				t.Errorf("authorizeError accepted %s", name)
			}
		}
	})
}

// pasteProbeAgentNameTwin is the plan probe's app name, which the untagged
// twin cannot read from the tagged live file.
const pasteProbeAgentNameTwin = probeIdentityName

// twinModuleName is this module's own name, which the sign-ins send when no
// Identity names another application (the tagged live files' identityValue).
const twinModuleName = "go-llmprovider-sdk"

// The twin of TestLive_EarlierGrant_0081 (0081-PLAN Phase 1a, step 7): the
// probe's own checking and reporting code, earlierGrant.run, against an
// in-process issuer and Kilo stand-in. Every answer carries secret-shaped
// values planted for the purpose, each also echoed in a refused answer's body
// and in the error the SDK returns; the test asserts that the captured output
// holds each verdict line, and none of the planted values. It also asserts
// that the code reached the stand-in terminal and the URLs the stand-in
// browser, and neither the output.

const (
	grantPlantedAccess  = "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJwbGFudGVkLTAwODEifQ.cGxhbnRlZC1zaWctMDA4MQ"
	grantPlantedCode    = "authcode-planted-0081-Zq83"
	grantPlantedState   = "state-planted-0081-Yk17"
	grantPlantedAccount = "acct-planted-0081-account"
	grantPlantedClient  = "app_planted0081clientid"
	grantPlantedHost    = "urn:uuid:00000000-0000-4000-8000-000000000081"
	grantPlantedUser    = "PLNT-0081"
	grantPlantedKilo    = "kilo-planted-0081-token"
	grantPlantedVerify  = "https://kilo.invalid/device-planted-0081"
	grantPlantedAuthURL = "https://issuer.invalid/authorize?client_id=" + grantPlantedClient + "&code=" + grantPlantedCode + "&state=" + grantPlantedState
	grantPlantedOther   = "made_up_0081_code"
)

// grantCase is one run of the twin: the provider, how the fake answers (mode),
// the shape of its refusals (shape: "" flat, nested, code or upper) and the
// code of an ended grant (code, invalid_grant when empty), what to change in
// the run before it starts, and how long the run may take (30 s when zero).
type grantCase struct {
	provider string
	mode     string
	shape    string
	code     string
	// revoke is how the fake answers a revocation: "" with an empty 200,
	// refused with a 400 whose body carries the planted values, serverError
	// with a 500 that does, slow not before the request's context ends.
	revoke      string
	revokeLimit time.Duration
	limit       time.Duration
	edit        func(tw *grantTwin, g *earlierGrant)
}

// grantFake is the issuer and the Kilo stand-in. Its mode says how it answers
// the first sign-in's session once a second sign-in has been issued, and a
// step before: kept answers every use; ended refuses the first session's
// (with the case's code, or Kilo's 401); reused refuses it as
// refresh_token_reused; unlisted with a code outside the allow-list; plain401
// with a 401 and no OAuth body; controlFails refuses the second session's
// with a 500; baselineFails refuses the first session's before the second
// sign-in with a 500; flaky answers the first refresh once with a 503; lazy answers the
// first session's first use after the second sign-in and refuses its second
// (with the case's code, or Kilo's 401).
type grantFake struct {
	srv  *httptest.Server
	c    grantCase
	code string

	mu          sync.Mutex
	issued      int
	counter     int
	flaked      bool
	discoveries int
	lazyUses    int
	revoked     []int          // the sign-in that held each token a revocation named, in order; 0 for none
	owner       map[string]int // a refresh or Kilo token -> the sign-in that holds it
	used        map[string]bool
	reusedSent  bool
	secrets     []string // every token handed out
	first       *auth.OAuthSession
}

func newGrantFake(t *testing.T, c grantCase) *grantFake {
	t.Helper()
	f := &grantFake{c: c, code: c.code, owner: map[string]int{}, used: map[string]bool{}}
	if f.code == "" {
		f.code = "invalid_grant"
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// nextRefresh makes a 40-character refresh token held by sign-in n; the
// caller holds f.mu.
func (f *grantFake) nextRefresh(n int) string {
	f.counter++
	rt := fmt.Sprintf("rt0081planted%027d", f.counter)
	f.owner[rt] = n
	f.secrets = append(f.secrets, rt)
	return rt
}

// issue is a sign-in: a new session, which a refresh or a profile read of
// the fake answers.
func (f *grantFake) issue() *auth.OAuthSession {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issued++
	n := f.issued
	var s *auth.OAuthSession
	switch f.c.provider {
	case grantKilo:
		tok := fmt.Sprintf("%s-%d", grantPlantedKilo, n)
		f.owner[tok] = n
		f.secrets = append(f.secrets, tok)
		s = &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: tok}
	default:
		s = &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: f.srv.URL, TokenURL: f.srv.URL + "/token",
			Access: grantPlantedAccess, Refresh: f.nextRefresh(n), Expiry: time.Now().Add(time.Hour),
			ClientID: grantPlantedClient, AccountID: grantPlantedAccount}
		switch f.c.provider {
		case grantGrok:
			s.Provider = llmprovider.ProviderGrok
		case grantPlan:
			s.Kind, s.HostID = auth.SessionChatGPTPlan, grantPlantedHost
			s.Resource, s.Scope, s.IDToken = openai.PlatformBaseURL, "openid offline_access chatgpt.tokens.use.direct", grantPlantedAccess
		}
	}
	if n == 1 {
		f.first = s
	}
	return s
}

// refusal is how the fake answers a use by sign-in n now: a status and an
// OAuth code, or no status for an answer.
func (f *grantFake) refusal(n int) (status int, code string) {
	later := f.issued >= 2 && n == 1
	switch {
	case f.c.mode == "ended" && later && f.c.provider == grantKilo:
		return http.StatusUnauthorized, "unauthorized"
	case f.c.mode == "ended" && later:
		return http.StatusBadRequest, f.code
	case f.c.mode == "reused" && later:
		return http.StatusBadRequest, grantReusedCode
	case f.c.mode == "unlisted" && later:
		return http.StatusBadRequest, grantPlantedOther
	case f.c.mode == "forbidden" && later && f.c.provider == grantKilo:
		return http.StatusForbidden, "forbidden"
	case f.c.mode == "forbidden" && later:
		return http.StatusForbidden, f.code
	case f.c.mode == "plain401" && later:
		return http.StatusUnauthorized, ""
	case f.c.mode == "lazy" && later:
		f.lazyUses++
		switch {
		case f.lazyUses < 2:
			return 0, ""
		case f.c.provider == grantKilo:
			return http.StatusUnauthorized, "unauthorized"
		}
		return http.StatusBadRequest, f.code
	case f.c.mode == "controlFails" && n == 2:
		return http.StatusInternalServerError, "server_error"
	case f.c.mode == "baselineFails" && f.issued == 1 && n == 1:
		return http.StatusInternalServerError, "server_error"
	case f.c.mode == "flaky" && n == 1 && !f.flaked:
		f.flaked = true
		return http.StatusServiceUnavailable, "server_error"
	}
	return 0, ""
}

// echo is every planted value in one string, for a refused answer to carry.
func (f *grantFake) echo(token string) string {
	return strings.Join([]string{token, grantPlantedAccess, grantPlantedClient, grantPlantedAccount, grantPlantedHost,
		grantPlantedCode, grantPlantedState, grantPlantedUser}, " ")
}

func (f *grantFake) refuse(w http.ResponseWriter, status int, code, token string) {
	echo := f.echo(token)
	switch {
	case code == "":
		w.WriteHeader(status)
		_, _ = io.WriteString(w, "unauthorized: "+echo)
	case f.c.provider == grantKilo:
		writeJSON(w, status, map[string]any{"error": echo})
	case f.c.shape == "nested":
		writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": echo}})
	case f.c.shape == "code":
		writeJSON(w, status, map[string]any{"code": code, "message": echo})
	case f.c.shape == "upper":
		writeJSON(w, status, map[string]any{"error": strings.ToUpper(code), "error_description": echo})
	default:
		writeJSON(w, status, map[string]any{"error": code, "error_description": echo})
	}
}

func (f *grantFake) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration"):
		f.discoveries++
		u := f.srv.URL
		writeJSON(w, http.StatusOK, map[string]any{"issuer": u, "authorization_endpoint": u + "/authorize",
			"token_endpoint": u + "/token", "revocation_endpoint": u + "/revoke", "jwks_uri": u + "/jwks"})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/revoke"):
		f.revokeAnswer(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/token"):
		if r.ParseForm() != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		rt := r.PostForm.Get("refresh_token")
		n := f.owner[rt]
		if f.used[rt] {
			f.reusedSent = true
		}
		f.used[rt] = true
		if status, code := f.refusal(n); status != 0 {
			f.refuse(w, status, code, rt)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": grantPlantedAccess, "refresh_token": f.nextRefresh(n),
			"expires_in": 3600})
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/profile"):
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if status, code := f.refusal(f.owner[token]); status != 0 {
			f.refuse(w, status, code, token)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"organizations": []any{}, "hasPersonalAccount": true})
	default:
		http.NotFound(w, r)
	}
}

// revokeAnswer answers a revocation: OpenAI's JSON body or an RFC 7009 form,
// either naming the token; the caller holds f.mu.
func (f *grantFake) revokeAnswer(w http.ResponseWriter, r *http.Request) {
	token := ""
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Token string `json:"token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		token = body.Token
	} else if r.ParseForm() == nil {
		token = r.PostForm.Get("token")
	}
	f.revoked = append(f.revoked, f.owner[token])
	switch f.c.revoke {
	case "refused":
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_request", "error_description": f.echo(token)})
	case "serverError":
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "revocation failed: "+f.echo(token))
	case "slow":
		f.mu.Unlock()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
		f.mu.Lock()
	}
}

// grantNopCloser is the stand-in terminal device.
type grantNopCloser struct{ io.Writer }

func (grantNopCloser) Close() error { return nil }

type grantFailingWriter struct{}

func (grantFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed: " + grantPlantedUser)
}

// grantUse is the text of a use step's line after its provider and name.
func grantUse(status int, kind, code string, requests int) string {
	return fmt.Sprintf("status %d, kind %s, code %s, requests %d", status, kind, code, requests)
}

// grantLines is the lines of a run through every step: the sign-ins, the
// uses (last is "" when the first session's second use was not made), the
// verdict, and the revocations that follow it.
func grantLines(provider string, baseline, first, control, last, verdict string, revokes ...string) []string {
	out := []string{
		provider + " sign-in-1: ok",
		provider + " baseline: " + baseline,
		provider + " sign-in-2: ok",
		provider + " first-after: " + first,
		provider + " control: " + control,
	}
	if last != "" {
		out = append(out, provider+" last-after: "+last)
	}
	out = append(out, provider+": "+verdict)
	for i, revoke := range revokes {
		out = append(out, fmt.Sprintf("%s revoke-%d: %s", provider, i+1, revoke))
	}
	return out
}

// grantRevoked is the revocation line's text of both grants when nothing goes
// wrong: the provider answers 200, or, Kilo's, there is no revocation to send,
// or, chatgpt's and grok's, the probe does not revoke.
func grantRevoked(provider string) []string {
	switch provider {
	case grantKilo:
		return []string{"status 0, kind unsupported", "status 0, kind unsupported"}
	case grantChatGPT, grantGrok:
		return []string{"status 0, kind by_hand", "status 0, kind by_hand"}
	}
	return []string{"status 200, kind ok", "status 200, kind ok"}
}

var (
	grantOK      = grantUse(200, "ok", "-", 1)
	grantRefused = grantUse(400, "auth", "invalid_grant", 1)
)

// grantTwin is one run against the fake, and what it produced.
type grantTwin struct {
	fake      *grantFake
	out       []string
	opened    []string
	terminal  bytes.Buffer
	previous  []*auth.OAuthSession
	signIns   int
	rawErrors []string // the SDK's errors for the uses
	// revokeErrors are the SDK's errors for the revocations.
	revokeErrors []string
	verdict      string
	stopped      string
	elapsed      time.Duration
}

// newGrantTwin builds the run of a case, without starting it.
func newGrantTwin(t *testing.T, c grantCase) (*grantTwin, *earlierGrant) {
	t.Helper()
	tw := &grantTwin{fake: newGrantFake(t, c)}
	rec := &grantRecorder{next: tw.fake.srv.Client().Transport}
	use := grantUseRefresh
	if c.provider == grantKilo {
		use = grantUseKiloProfile(llmprovider.WithBaseURL(tw.fake.srv.URL+"/api/gateway"), llmprovider.WithHTTPClient(rec.client()))
	}
	revokeLimit := c.revokeLimit
	if revokeLimit == 0 {
		revokeLimit = 5 * time.Second
	}
	g := &earlierGrant{
		provider:    c.provider,
		logf:        func(format string, args ...any) { tw.out = append(tw.out, fmt.Sprintf(format, args...)) },
		rec:         rec,
		needsCode:   c.provider == grantKilo,
		wait:        30 * time.Second,
		revokeLimit: revokeLimit,
		revoke: func(ctx context.Context, session *auth.OAuthSession) error {
			err := auth.RevokeOAuthSession(ctx, session)
			if err != nil {
				tw.revokeErrors = append(tw.revokeErrors, err.Error())
			}
			return err
		},
		io: grantIO{
			canOpen: func() bool { return true },
			open:    func(raw string) error { tw.opened = append(tw.opened, raw); return nil },
			tty:     func() (io.WriteCloser, error) { return grantNopCloser{&tw.terminal}, nil },
		},
		signIn: func(_ context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error) {
			tw.signIns++
			tw.previous = append(tw.previous, previous)
			shown := grantPlantedAuthURL
			if c.provider == grantKilo {
				shown = grantPlantedVerify
			}
			if err := show.URL(shown); err != nil {
				return nil, err
			}
			if c.provider == grantKilo {
				show.Code(grantPlantedUser)
			}
			return tw.fake.issue(), nil
		},
		use: func(ctx context.Context, session *auth.OAuthSession) error {
			err := use(ctx, session)
			if err != nil {
				tw.rawErrors = append(tw.rawErrors, err.Error())
			}
			return err
		},
	}
	if c.edit != nil {
		c.edit(tw, g)
	}
	return tw, g
}

func runGrantTwin(t *testing.T, c grantCase) *grantTwin {
	t.Helper()
	tw, g := newGrantTwin(t, c)
	limit := c.limit
	if limit == 0 {
		limit = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	start := time.Now()
	tw.verdict, tw.stopped = g.run(ctx)
	tw.elapsed = time.Since(start)
	return tw
}

// planted is every secret-shaped value of the run: the constants, and every
// token the fake handed out.
func (tw *grantTwin) planted() []string {
	all := []string{grantPlantedAccess, grantPlantedCode, grantPlantedState, grantPlantedAccount, grantPlantedClient,
		grantPlantedHost, grantPlantedUser, grantPlantedKilo, grantPlantedVerify, grantPlantedAuthURL, grantPlantedOther,
		tw.fake.srv.URL, strings.TrimPrefix(tw.fake.srv.URL, "http://")}
	tw.fake.mu.Lock()
	defer tw.fake.mu.Unlock()
	return append(all, tw.fake.secrets...)
}

// checkOutput fails unless the output holds exactly want, and none of the
// planted values, none of the URLs shown and none of the error texts the SDK
// returned.
func (tw *grantTwin) checkOutput(t *testing.T, want []string) {
	t.Helper()
	if !slices.Equal(tw.out, want) {
		t.Errorf("output:\n%s\nwant:\n%s", strings.Join(tw.out, "\n"), strings.Join(want, "\n"))
	}
	joined := strings.Join(tw.out, "\n")
	for _, secret := range append(tw.planted(), tw.opened...) {
		if strings.Contains(joined, secret) {
			t.Errorf("the output holds a planted value (%d characters, starting %q)", len(secret), secret[:4])
		}
	}
	for _, raw := range append(append([]string(nil), tw.rawErrors...), tw.revokeErrors...) {
		if strings.Contains(joined, raw) {
			t.Errorf("the output holds an error's text (%d characters)", len(raw))
		}
	}
}

// checkShown fails unless each sign-in's page reached the stand-in browser
// and, for Kilo, its user code the stand-in terminal device and nothing
// else, and the plan's second sign-in was made as the first's again.
func (tw *grantTwin) checkShown(t *testing.T, provider string) {
	t.Helper()
	page := grantPlantedAuthURL
	if provider == grantKilo {
		page = grantPlantedVerify
	}
	if !slices.Equal(tw.opened, []string{page, page}) {
		t.Errorf("the browser was handed %d URLs, want the sign-in's page twice", len(tw.opened))
	}
	wantTerminal := ""
	if provider == grantKilo {
		wantTerminal = "Kilo code: " + grantPlantedUser + "\nKilo code: " + grantPlantedUser + "\n"
	}
	if tw.terminal.String() != wantTerminal {
		t.Errorf("the terminal device was written %q, want %q", tw.terminal.String(), wantTerminal)
	}
	if len(tw.previous) != 2 || tw.previous[0] != nil {
		t.Fatalf("the sign-ins were given %d previous sessions, the first %v", len(tw.previous), tw.previous)
	}
	if provider == grantPlan && (tw.previous[1] == nil || tw.previous[1] != tw.fake.first) {
		t.Error("the plan's second sign-in was not made with the first session as Previous")
	}
	if provider != grantPlan && tw.previous[1] != nil {
		t.Error("a sign-in other than the plan's was given a Previous session")
	}
	tw.fake.mu.Lock()
	defer tw.fake.mu.Unlock()
	if tw.fake.reusedSent {
		t.Error("a refresh token was sent twice: the probe's own defect")
	}
}

// TestEarlierGrantReport_0081 is the redaction proof and the verdicts of
// TestLive_EarlierGrant_0081's checks (0081-PLAN Phase 1a, step 7).
func TestEarlierGrantReport_0081(t *testing.T) {
	refused := func(provider string) string {
		if provider == grantKilo {
			return grantUse(401, "auth", "other", 1)
		}
		return grantRefused
	}
	// A 5xx is sent again by the SDK's refresh, three times in all; Kilo's
	// profile read is not.
	failed := func(provider string) string {
		if provider == grantKilo {
			return grantUse(500, "unavailable", "other", 1)
		}
		return grantUse(500, "unavailable", "other", 3)
	}
	for _, provider := range grantProviders {
		t.Run(provider+"/kept", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, grantOK, grantOK, grantKept, grantRevoked(provider)...))
			tw.checkShown(t, provider)
			tw.checkRevoked(t, provider, 1, 2)
		})
		t.Run(provider+"/ended", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "ended"})
			tw.checkOutput(t, grantLines(provider, grantOK, refused(provider), grantOK, "", grantEnded, grantRevoked(provider)...))
			tw.checkShown(t, provider)
			if len(tw.rawErrors) != 1 || !strings.Contains(tw.rawErrors[0], grantPlantedClient) && !strings.Contains(tw.rawErrors[0], "rt0081planted") {
				t.Errorf("the SDK's error carries none of the planted echo, so the output check proves nothing: %d errors", len(tw.rawErrors))
			}
		})
		t.Run(provider+"/lazy", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "lazy"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, grantOK, refused(provider), grantEnded, grantRevoked(provider)...))
		})
		// A 403 is classified an auth failure by the SDK, but it is not a dead
		// grant: Kilo's profile that cannot say (0080-MADR D1, D5), and, for the
		// others, a refusal at a status the SDK's terminal codes do not come at.
		t.Run(provider+"/forbidden", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "forbidden"})
			code := "invalid_grant"
			if provider == grantKilo {
				code = "other"
			}
			tw.checkOutput(t, grantLines(provider, grantOK, grantUse(403, "auth", code, 1), grantOK, "", grantIndeterminate, grantRevoked(provider)...))
		})
		t.Run(provider+"/control fails", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "controlFails"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, failed(provider), grantOK, grantIndeterminate, grantRevoked(provider)...))
		})
		t.Run(provider+"/baseline fails", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "baselineFails"})
			revoked := grantRevoked(provider)[0]
			tw.checkOutput(t, []string{provider + " sign-in-1: ok", provider + " baseline: " + failed(provider),
				provider + ": " + grantIndeterminate, provider + " revoke-1: " + revoked})
			if tw.signIns != 1 {
				t.Errorf("%d sign-ins, want 1: a failed baseline ends the run", tw.signIns)
			}
		})
		if provider == grantKilo {
			continue
		}
		t.Run(provider+"/reused", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "reused"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantUse(400, "auth", grantReusedCode, 1), grantOK, "", grantIndeterminate, grantRevoked(provider)...))
		})
		t.Run(provider+"/lazy reused", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "lazy", code: grantReusedCode})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, grantOK, grantUse(400, "auth", grantReusedCode, 1), grantIndeterminate, grantRevoked(provider)...))
		})
	}
	t.Run("a code outside the allow-list is kept as other", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantChatGPT, mode: "unlisted"})
		tw.checkOutput(t, grantLines(grantChatGPT, grantOK, grantUse(400, "invalid_request", "other", 1), grantOK, "", grantIndeterminate, grantRevoked(grantChatGPT)...))
	})
	t.Run("a late refusal with a code outside the allow-list is unclear", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantChatGPT, mode: "lazy", code: grantPlantedOther})
		tw.checkOutput(t, grantLines(grantChatGPT, grantOK, grantOK, grantOK, grantUse(400, "invalid_request", "other", 1), grantIndeterminate, grantRevoked(grantChatGPT)...))
	})
	t.Run("a 401 with no OAuth body is not an ended grant", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantGrok, mode: "plain401"})
		tw.checkOutput(t, grantLines(grantGrok, grantOK, grantUse(401, "auth", "-", 1), grantOK, "", grantIndeterminate, grantRevoked(grantGrok)...))
	})
	t.Run("kilo's 401 with an unreadable answer is an ended grant", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantKilo, mode: "plain401"})
		tw.checkOutput(t, grantLines(grantKilo, grantOK, grantUse(401, "auth", "-", 1), grantOK, "", grantEnded, grantRevoked(grantKilo)...))
	})
	// The shapes in which an issuer sends its code, as the SDK reads them
	// (0081-PLAN Phase 1a, review fix F1).
	for _, shape := range []struct{ name, shape, code, want, verdict string }{
		{"nested invalid_grant", "nested", "invalid_grant", "invalid_grant", grantEnded},
		{"nested refresh_token_invalidated", "nested", "refresh_token_invalidated", "refresh_token_invalidated", grantEnded},
		{"nested refresh_token_expired", "nested", "refresh_token_expired", "refresh_token_expired", grantEnded},
		{"nested refresh_token_reused", "nested", grantReusedCode, grantReusedCode, grantIndeterminate},
		{"a bare code", "code", "invalid_grant", "invalid_grant", grantEnded},
		{"a bare reused code", "code", grantReusedCode, grantReusedCode, grantIndeterminate},
		{"an upper-case code", "upper", "invalid_grant", "invalid_grant", grantEnded},
		{"an upper-case reused code", "upper", grantReusedCode, grantReusedCode, grantIndeterminate},
	} {
		t.Run("the shape: "+shape.name, func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: grantChatGPT, mode: "ended", shape: shape.shape, code: shape.code})
			tw.checkOutput(t, grantLines(grantChatGPT, grantOK, grantUse(400, "auth", shape.want, 1), grantOK, "", shape.verdict, grantRevoked(grantChatGPT)...))
		})
	}
	// The count of requests a step made (review fix F7), and the owner's
	// ruling that a step that made more than one makes the run indeterminate.
	// The owner's ruling "Stop there: end the run as `indeterminate` before the
	// second sign-in (Recommended)": a baseline that succeeded but needed more
	// than one request ends the run, as a failed one does, with its first grant
	// revoked where the provider's revocation is on.
	for _, provider := range []string{grantChatGPT, grantPlan, grantGrok} {
		t.Run(provider+"/a baseline that needed two requests ends the run before the second sign-in", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "flaky"})
			tw.checkOutput(t, []string{provider + " sign-in-1: ok", provider + " baseline: " + grantUse(200, "ok", "-", 2),
				provider + ": " + grantIndeterminate, provider + " revoke-1: " + grantRevoked(provider)[0]})
			if tw.signIns != 1 {
				t.Errorf("%d sign-ins, want 1: a baseline of two requests ends the run", tw.signIns)
			}
		})
	}
}

// checkRevoked fails unless the fake was asked to revoke the grants of the
// sign-ins in want, in that order, and, for Kilo, which has no revocation,
// was asked nothing.
func (tw *grantTwin) checkRevoked(t *testing.T, provider string, want ...int) {
	t.Helper()
	tw.fake.mu.Lock()
	defer tw.fake.mu.Unlock()
	if provider == grantKilo || provider == grantChatGPT || provider == grantGrok {
		want = nil
	}
	if provider == grantChatGPT || provider == grantGrok {
		if tw.fake.discoveries != 0 {
			t.Errorf("%d discovery requests were made for a revocation that is not made", tw.fake.discoveries)
		}
	}
	if !slices.Equal(tw.fake.revoked, want) {
		t.Errorf("the revocations named the grants of sign-ins %v, want %v", tw.fake.revoked, want)
	}
}

// TestEarlierGrantRevocation_0081 (the owner's ruling "Revoke the probe's own
// two grants at the end, after the verdict is printed"): each grant obtained
// is revoked after the verdict line, on every path, on a context of its own
// and within its own bound; a revocation's failure shows in its line, a fixed
// shape that carries no body, code or error text and no count, and never
// changes the verdict or fails the run.
func TestEarlierGrantRevocation_0081(t *testing.T) {
	t.Run("a refused revocation is told, and changes nothing", func(t *testing.T) {
		for _, provider := range []string{grantPlan} {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept", revoke: "refused"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, grantOK, grantOK, grantKept,
				"status 400, kind other", "status 400, kind other"))
			if tw.verdict != grantKept || tw.stopped != "" {
				t.Errorf("%s: verdict %q, stopped %q after a refused revocation; want kept", provider, tw.verdict, tw.stopped)
			}
			if len(tw.revokeErrors) != 2 || !strings.Contains(tw.revokeErrors[0]+tw.revokeErrors[1], "rt0081planted") && !strings.Contains(tw.revokeErrors[0]+tw.revokeErrors[1], grantPlantedClient) {
				t.Errorf("%s: the SDK's revocation errors carry none of the planted echo, so the output check proves nothing: %d", provider, len(tw.revokeErrors))
			}
		}
	})
	t.Run("a server error is told", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", revoke: "serverError"})
		tw.checkOutput(t, grantLines(grantPlan, grantOK, grantOK, grantOK, grantOK, grantKept,
			"status 500, kind other", "status 500, kind other"))
	})
	t.Run("a revocation that outlasts its bound is cut", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", revoke: "slow", revokeLimit: 150 * time.Millisecond})
		tw.checkOutput(t, grantLines(grantPlan, grantOK, grantOK, grantOK, grantOK, grantKept,
			"status 0, kind canceled", "status 0, kind canceled"))
		if tw.elapsed > 3*time.Second {
			t.Errorf("the run took %v: a revocation was not bound", tw.elapsed)
		}
	})
	t.Run("the run's own bound having passed does not stop the revocation", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", limit: 1500 * time.Millisecond, edit: func(tw *grantTwin, g *earlierGrant) {
			inner := g.signIn
			g.signIn = func(ctx context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error) {
				if tw.signIns >= 1 {
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return inner(ctx, show, previous)
			}
		}})
		tw.checkOutput(t, []string{"plan sign-in-1: ok", "plan baseline: " + grantOK, "plan sign-in-2: canceled",
			"plan: " + grantIndeterminate, "plan revoke-1: status 200, kind ok"})
		tw.checkRevoked(t, grantPlan, 1)
	})
	t.Run("a sign-in that failed has no grant to revoke", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.signIn = func(context.Context, *grantShow, *auth.OAuthSession) (*auth.OAuthSession, error) {
				return nil, errors.New("sign-in failed: " + grantPlantedCode)
			}
		}})
		tw.checkOutput(t, []string{"plan sign-in-1: other", "plan: " + grantIndeterminate})
		tw.checkRevoked(t, grantPlan)
	})
	t.Run("a stop after the first sign-in still revokes it, after no verdict", func(t *testing.T) {
		opens := 0
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.io.open = func(string) error {
				opens++
				if opens > 1 {
					return errors.New("open failed: " + grantPlantedCode)
				}
				return nil
			}
		}})
		if want := "plan: stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
			t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
		}
		tw.checkOutput(t, []string{"plan sign-in-1: ok", "plan baseline: " + grantOK, "plan revoke-1: status 200, kind ok"})
		tw.checkRevoked(t, grantPlan, 1)
	})
	t.Run("a grant obtained as the run is stopped is still revoked", func(t *testing.T) {
		opens := 0
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", edit: func(tw *grantTwin, g *earlierGrant) {
			g.io.open = func(string) error {
				opens++
				if opens > 1 {
					return errors.New("open failed: " + grantPlantedCode)
				}
				return nil
			}
			// a sign-in that finishes though its opener failed
			g.signIn = func(_ context.Context, show *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
				tw.signIns++
				_ = show.URL(grantPlantedAuthURL)
				return tw.fake.issue(), nil
			}
		}})
		if want := "plan: stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
			t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
		}
		tw.checkOutput(t, []string{"plan sign-in-1: ok", "plan baseline: " + grantOK,
			"plan revoke-1: status 200, kind ok", "plan revoke-2: status 200, kind ok"})
		tw.checkRevoked(t, grantPlan, 1, 2)
	})
	t.Run("a first sign-in that stops the run has nothing to revoke", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantPlan, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.io.open = func(string) error { return errors.New("open failed") }
		}})
		tw.checkOutput(t, nil)
		tw.checkRevoked(t, grantPlan)
	})
}

// TestEarlierGrantNoRevocation_0081 (the owner's rulings that chatgpt's and
// grok's grants are not revoked): no request to a revocation endpoint, nor a
// discovery for one, on the verdict path, the stop path, the indeterminate
// path or when the run's bound has passed, and a fixed word in each line.
func TestEarlierGrantNoRevocation_0081(t *testing.T) {
	// Each case below: no request to a revocation endpoint, nor a discovery
	// for one, on the verdict
	// path, the stop path, the indeterminate path or when the run's bound has
	// passed, and a fixed word in each line.
	for _, provider := range []string{grantChatGPT, grantGrok} {
		byHand := func(n int) string { return fmt.Sprintf("%s revoke-%d: status 0, kind by_hand", provider, n) }
		t.Run(provider+"'s grants are not revoked, on the verdict path", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept"})
			tw.checkOutput(t, grantLines(provider, grantOK, grantOK, grantOK, grantOK, grantKept, "status 0, kind by_hand", "status 0, kind by_hand"))
			tw.checkRevoked(t, provider)
		})
		t.Run(provider+"'s grants are not revoked, on the indeterminate path", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept", edit: func(tw *grantTwin, g *earlierGrant) {
				inner := g.signIn
				g.signIn = func(ctx context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error) {
					if tw.signIns >= 1 {
						tw.signIns++
						return nil, errors.New("sign-in failed: " + grantPlantedCode)
					}
					return inner(ctx, show, previous)
				}
			}})
			tw.checkOutput(t, []string{provider + " sign-in-1: ok", provider + " baseline: " + grantOK, provider + " sign-in-2: other",
				provider + ": " + grantIndeterminate, byHand(1)})
			tw.checkRevoked(t, provider)
		})
		t.Run(provider+"'s grants are not revoked, on the stop path", func(t *testing.T) {
			opens := 0
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept", edit: func(tw *grantTwin, g *earlierGrant) {
				g.io.open = func(string) error {
					opens++
					if opens > 1 {
						return errors.New("open failed: " + grantPlantedCode)
					}
					return nil
				}
				g.signIn = func(_ context.Context, show *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
					tw.signIns++
					_ = show.URL(grantPlantedAuthURL)
					return tw.fake.issue(), nil
				}
			}})
			if want := provider + ": stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
				t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
			}
			tw.checkOutput(t, []string{provider + " sign-in-1: ok", provider + " baseline: " + grantOK, byHand(1), byHand(2)})
			tw.checkRevoked(t, provider)
		})
		t.Run(provider+"'s grants are not revoked, when the run's bound has passed", func(t *testing.T) {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept", limit: 1500 * time.Millisecond, edit: func(tw *grantTwin, g *earlierGrant) {
				inner := g.signIn
				g.signIn = func(ctx context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error) {
					if tw.signIns >= 1 {
						<-ctx.Done()
						return nil, ctx.Err()
					}
					return inner(ctx, show, previous)
				}
			}})
			tw.checkOutput(t, []string{provider + " sign-in-1: ok", provider + " baseline: " + grantOK, provider + " sign-in-2: canceled",
				provider + ": " + grantIndeterminate, byHand(1)})
			tw.checkRevoked(t, provider)
		})
	}
	t.Run("only plan's and kilo's grants go through the revocation", func(t *testing.T) {
		for provider, want := range map[string]bool{grantChatGPT: false, grantGrok: false, grantPlan: true, grantKilo: true} {
			if got := grantRevokes(provider); got != want {
				t.Errorf("grantRevokes(%q) = %t, want %t", provider, got, want)
			}
		}
	})
}

// TestEarlierGrantStops_0081: when the browser or the terminal device cannot
// be used, the run stops with one plain line holding neither the URL, the
// code nor the error's text, and no verdict; the stand-in's own failure
// carries the planted values, and nothing is written to the output. The
// browser sign-ins call the opener from a goroutine and ignore what it
// returns, so the stand-ins that block until their context ends are the ones
// that show the stop is the opener's own.
func TestEarlierGrantStops_0081(t *testing.T) {
	failing := func(raw string) error {
		return fmt.Errorf("open %s: %s %s failed", raw, grantPlantedCode, grantPlantedUser)
	}
	blocksUntilCancelled := func(tw *grantTwin, g *earlierGrant) {
		g.wait = 4 * time.Second
		g.signIn = func(ctx context.Context, show *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
			tw.signIns++
			go func() { _ = show.URL(grantPlantedAuthURL) }() // as runBrowserLogin does
			<-ctx.Done()
			return nil, ctx.Err()
		}
	}
	t.Run("the browser fails", func(t *testing.T) {
		for _, provider := range []string{grantChatGPT, grantKilo} {
			tw := runGrantTwin(t, grantCase{provider: provider, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) { g.io.open = failing }})
			if want := provider + ": stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
				t.Errorf("%s: stopped %q, verdict %q; want %q and no verdict", provider, tw.stopped, tw.verdict, want)
			}
			tw.checkOutput(t, nil)
			if tw.stopped != "" && strings.ContainsAny(tw.stopped, "/?=") {
				t.Errorf("the stop line carries a URL or an error: %q", tw.stopped)
			}
		}
	})
	t.Run("a browser sign-in that ignores the opener's error is stopped by the opener", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantGrok, mode: "kept", limit: 10 * time.Second, edit: func(tw *grantTwin, g *earlierGrant) {
			blocksUntilCancelled(tw, g)
			g.io.open = failing
		}})
		if want := "grok: stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
			t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
		}
		if tw.elapsed > 2*time.Second {
			t.Errorf("the run took %v: the failed opener did not end the sign-in", tw.elapsed)
		}
		tw.checkOutput(t, nil)
	})
	t.Run("the shipped browser login is stopped by a failing opener", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantGrok, mode: "kept", limit: 10 * time.Second, edit: func(tw *grantTwin, g *earlierGrant) {
			g.wait = 4 * time.Second
			g.io.open = failing
			g.signIn = func(ctx context.Context, show *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
				tw.signIns++
				return auth.LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
					Issuer: tw.fake.srv.URL, HTTPClient: g.rec.client(), OpenURL: show.URL})
			}
		}})
		if want := "grok: stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" {
			t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
		}
		if tw.elapsed > 2*time.Second {
			t.Errorf("the run took %v: the failed opener did not end the shipped sign-in", tw.elapsed)
		}
		tw.checkOutput(t, nil)
	})
	t.Run("there is no browser opener", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantGrok, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.io.canOpen = func() bool { return false }
		}})
		if want := "grok: stopped: the browser could not be opened"; tw.stopped != want || tw.verdict != "" || tw.signIns != 0 {
			t.Errorf("stopped %q, verdict %q, %d sign-ins; want %q, none, 0", tw.stopped, tw.verdict, tw.signIns, want)
		}
		tw.checkOutput(t, nil)
	})
	t.Run("the terminal device cannot be opened", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantKilo, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.io.tty = func() (io.WriteCloser, error) { return nil, failing("/dev/tty") }
		}})
		if want := "kilo: stopped: the terminal device could not be opened"; tw.stopped != want || tw.verdict != "" || tw.signIns != 0 {
			t.Errorf("stopped %q, verdict %q, %d sign-ins; want %q, none, 0", tw.stopped, tw.verdict, tw.signIns, want)
		}
		tw.checkOutput(t, nil)
		if len(tw.opened) != 0 {
			t.Errorf("the browser was handed %d URLs before the terminal device was known to work", len(tw.opened))
		}
	})
	t.Run("the terminal device fails to take the code", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantKilo, mode: "kept", limit: 10 * time.Second, edit: func(tw *grantTwin, g *earlierGrant) {
			g.wait = 4 * time.Second
			g.io.tty = func() (io.WriteCloser, error) { return grantNopCloser{grantFailingWriter{}}, nil }
			g.signIn = func(ctx context.Context, show *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
				tw.signIns++
				show.Code(grantPlantedUser)
				<-ctx.Done()
				return nil, ctx.Err()
			}
		}})
		if want := "kilo: stopped: the terminal device could not be opened"; tw.stopped != want || tw.verdict != "" {
			t.Errorf("stopped %q, verdict %q; want %q and no verdict", tw.stopped, tw.verdict, want)
		}
		if tw.elapsed > 2*time.Second {
			t.Errorf("the run took %v: the failed write did not end the sign-in", tw.elapsed)
		}
		tw.checkOutput(t, nil)
	})
	t.Run("the wait for the person is bounded", func(t *testing.T) {
		tw := runGrantTwin(t, grantCase{provider: grantChatGPT, mode: "kept", limit: 8 * time.Second, edit: func(tw *grantTwin, g *earlierGrant) {
			g.wait = 300 * time.Millisecond
			g.signIn = func(ctx context.Context, _ *grantShow, _ *auth.OAuthSession) (*auth.OAuthSession, error) {
				tw.signIns++
				<-ctx.Done()
				return nil, ctx.Err()
			}
		}})
		if tw.stopped != "" || tw.verdict != grantIndeterminate || tw.elapsed > 3*time.Second {
			t.Errorf("stopped %q, verdict %q after %v; want none, indeterminate, within the sign-in's wait", tw.stopped, tw.verdict, tw.elapsed)
		}
		tw.checkOutput(t, []string{"chatgpt sign-in-1: canceled", "chatgpt: indeterminate"})
	})
}

// TestEarlierGrantRunTest_0081: the body of the live test, with its switch and
// its provider read through a stand-in environment: it skips with the switch
// off before anything is built, fails at once, with a message that does not
// repeat the value, on a provider that is not one of the four, and fails only
// when the verdict is not kept.
func TestEarlierGrantRunTest_0081(t *testing.T) {
	env := func(pairs ...string) func(string) string {
		return func(key string) string {
			for i := 0; i+1 < len(pairs); i += 2 {
				if pairs[i] == key {
					return pairs[i+1]
				}
			}
			return ""
		}
	}
	never := func(string) (*earlierGrant, error) {
		t.Error("the probe was built before the switch and the provider were right")
		return nil, errors.New("built")
	}
	ctx := twinContext(t)
	t.Run("the switch is off", func(t *testing.T) {
		for _, getenv := range []func(string) string{env(), env("LLMPROVIDER_LIVE_0081", "0"), env("LLMPROVIDER_LIVE_0081", "true", "LLMPROVIDER_LIVE_0081_PROVIDER", "kilo")} {
			if out := grantRunTest(ctx, getenv, never); out.skip == "" || out.fatal != "" || out.fail {
				t.Errorf("outcome %+v, want a skip", out)
			}
		}
	})
	t.Run("the provider is none of the four", func(t *testing.T) {
		const odd = "secret-shaped-provider-0081"
		for _, getenv := range []func(string) string{env("LLMPROVIDER_LIVE_0081", "1"), env("LLMPROVIDER_LIVE_0081", "1", "LLMPROVIDER_LIVE_0081_PROVIDER", odd)} {
			out := grantRunTest(ctx, getenv, never)
			if out.fatal == "" || out.skip != "" || out.fail || strings.Contains(out.fatal, odd) {
				t.Errorf("outcome %+v, want a fatal message that does not repeat the value", out)
			}
		}
	})
	t.Run("the probe cannot be set up", func(t *testing.T) {
		out := grantRunTest(ctx, env("LLMPROVIDER_LIVE_0081", "1", "LLMPROVIDER_LIVE_0081_PROVIDER", "plan"),
			func(string) (*earlierGrant, error) { return nil, errors.New("host id: " + grantPlantedHost) })
		if out.fatal == "" || strings.Contains(out.fatal, grantPlantedHost) {
			t.Errorf("outcome %+v, want a fixed fatal message", out)
		}
	})
	for _, c := range []struct {
		mode string
		fail bool
	}{{"kept", false}, {"ended", true}, {"controlFails", true}} {
		t.Run("the verdict of the "+c.mode+" run", func(t *testing.T) {
			_, g := newGrantTwin(t, grantCase{provider: grantGrok, mode: c.mode})
			out := grantRunTest(ctx, env("LLMPROVIDER_LIVE_0081", "1", "LLMPROVIDER_LIVE_0081_PROVIDER", "grok"),
				func(string) (*earlierGrant, error) { return g, nil })
			if out.skip != "" || out.fatal != "" || out.fail != c.fail {
				t.Errorf("outcome %+v, want fail %t only", out, c.fail)
			}
		})
	}
	t.Run("a stopped run is fatal with its line", func(t *testing.T) {
		_, g := newGrantTwin(t, grantCase{provider: grantGrok, mode: "kept", edit: func(_ *grantTwin, g *earlierGrant) {
			g.io.canOpen = func() bool { return false }
		}})
		out := grantRunTest(ctx, env("LLMPROVIDER_LIVE_0081", "1", "LLMPROVIDER_LIVE_0081_PROVIDER", "grok"),
			func(string) (*earlierGrant, error) { return g, nil })
		if out.fatal != "grok: stopped: the browser could not be opened" || out.skip != "" {
			t.Errorf("outcome %+v, want the stop line", out)
		}
	})
}

// grantRecorderT is a stand-in testing.T that records what an outcome does to
// it.
type grantRecorderT struct {
	skipped, fatal []any
	failed         bool
}

func (r *grantRecorderT) Skip(args ...any)  { r.skipped = args }
func (r *grantRecorderT) Fatal(args ...any) { r.fatal = args }
func (r *grantRecorderT) Fail()             { r.failed = true }

// TestGrantApply_0081: an outcome skips, fails at once with its message, or
// fails at the end, and does nothing when it is clean.
func TestGrantApply_0081(t *testing.T) {
	for _, c := range []struct {
		name string
		out  grantOutcome
		want grantRecorderT
	}{
		{"clean", grantOutcome{}, grantRecorderT{}},
		{"skip", grantOutcome{skip: "off"}, grantRecorderT{skipped: []any{"off"}}},
		{"fatal", grantOutcome{fatal: "stopped"}, grantRecorderT{fatal: []any{"stopped"}}},
		{"fail", grantOutcome{fail: true}, grantRecorderT{failed: true}},
	} {
		var got grantRecorderT
		grantApply(&got, c.out)
		if fmt.Sprint(got.skipped, got.fatal, got.failed) != fmt.Sprint(c.want.skipped, c.want.fatal, c.want.failed) {
			t.Errorf("%s: skipped %v, fatal %v, failed %t; want %v, %v, %t", c.name, got.skipped, got.fatal, got.failed,
				c.want.skipped, c.want.fatal, c.want.failed)
		}
	}
}

// TestGrantPasses_0081: only kept passes (0081-PLAN Phase 1a, the reading
// taken; the owner may rule otherwise).
func TestGrantPasses_0081(t *testing.T) {
	for verdict, want := range map[string]bool{grantKept: true, grantEnded: false, grantIndeterminate: false, "": false} {
		if got := grantPasses(verdict); got != want {
			t.Errorf("grantPasses(%q) = %t, want %t", verdict, got, want)
		}
	}
}

// TestGrantWebURL_0081: the browser opener is given an http or https URL with
// a host, nothing else.
func TestGrantWebURL_0081(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://issuer.invalid/authorize?a=b": true,
		"http://127.0.0.1:1455/x":              true,
		"file:///etc/passwd":                   false,
		"javascript:alert(1)":                  false,
		"ssh://host.invalid/x":                 false,
		"https:///nohost":                      false,
		"//issuer.invalid/x":                   false,
		"issuer.invalid/x":                     false,
		"":                                     false,
		"https://[::1":                         false,
	} {
		if got := grantWebURL(raw); got != want {
			t.Errorf("grantWebURL(%q) = %t, want %t", raw, got, want)
		}
	}
}

// TestGrantKind_0081: an error's kind by errors.Is, a context's end before
// every other kind (an auth failure that wraps a cancel is a cancel).
func TestGrantKind_0081(t *testing.T) {
	wrap := func(sentinel error, inner error) error { return fmt.Errorf("%w: %w", sentinel, inner) }
	for _, c := range []struct {
		name string
		err  error
		want string
	}{
		{"none", nil, "ok"},
		{"canceled", context.Canceled, "canceled"},
		{"deadline", context.DeadlineExceeded, "canceled"},
		{"auth", llmprovider.ErrAuthFailure, "auth"},
		{"rate limited", llmprovider.ErrRateLimited, "rate_limited"},
		{"unavailable", llmprovider.ErrProviderUnavailable, "unavailable"},
		{"invalid request", llmprovider.ErrInvalidRequest, "invalid_request"},
		{"unsupported", llmprovider.ErrUnsupported, "unsupported"},
		{"anything else", errors.New("other"), "other"},
		{"auth wrapping a cancel", wrap(llmprovider.ErrAuthFailure, context.Canceled), "canceled"},
		{"auth wrapping a deadline", wrap(llmprovider.ErrAuthFailure, context.DeadlineExceeded), "canceled"},
		{"unavailable wrapping a cancel", wrap(llmprovider.ErrProviderUnavailable, context.Canceled), "canceled"},
	} {
		if got := grantKind(c.err); got != c.want {
			t.Errorf("%s: grantKind = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestGrantVerdict_0081: the verdict by step, every row of the table on
// grantVerdict, and the count rule (0081-PLAN Phase 1a, review fixes and the
// owner's rulings of 2026-11-18).
func TestGrantVerdict_0081(t *testing.T) {
	ok := grantStep{status: 200, kind: "ok", requests: 1}
	refused := func(status int, kind, code string) grantStep {
		return grantStep{status: status, kind: kind, code: code, requests: 1}
	}
	dead := refused(400, "auth", "invalid_grant")
	reused := refused(400, "auth", grantReusedCode)
	unclear := refused(500, "unavailable", "")
	skipped := grantStep{}
	twice := func(s grantStep) grantStep { s.requests = 2; return s }
	for _, c := range []struct {
		name                           string
		provider                       string
		baseline, first, last, control grantStep
		want                           string
	}{
		{"both later uses succeed", grantGrok, ok, ok, ok, ok, grantKept},
		{"the lazy case: the first later use succeeds, the second is refused as dead", grantGrok, ok, ok, dead, ok, grantEnded},
		{"the second later use is a reuse refusal", grantGrok, ok, ok, reused, ok, grantIndeterminate},
		{"the second later use is unclear", grantGrok, ok, ok, unclear, ok, grantIndeterminate},
		{"the second later use is a refusal with another code", grantGrok, ok, ok, refused(400, "auth", "other"), ok, grantIndeterminate},
		{"the first later use is refused as dead", grantGrok, ok, dead, skipped, ok, grantEnded},
		{"the first later use is refused as dead by 401", grantChatGPT, ok, refused(401, "auth", "invalid_client"), skipped, ok, grantEnded},
		{"the first later use is a reuse refusal", grantGrok, ok, reused, skipped, ok, grantIndeterminate},
		{"the first later use is unclear", grantGrok, ok, unclear, skipped, ok, grantIndeterminate},
		{"the second later use was not made though the first succeeded", grantGrok, ok, ok, skipped, ok, grantIndeterminate},
		{"the baseline fails", grantGrok, unclear, ok, ok, ok, grantIndeterminate},
		{"the baseline is refused as dead", grantGrok, dead, dead, skipped, ok, grantIndeterminate},
		{"the control fails", grantGrok, ok, ok, ok, unclear, grantIndeterminate},
		{"the control fails and the first later use is dead", grantGrok, ok, dead, skipped, unclear, grantIndeterminate},
		{"a dead grant at 500", grantGrok, ok, refused(500, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a dead grant at 403", grantGrok, ok, refused(403, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a dead grant at 200", grantGrok, ok, refused(200, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"no code", grantGrok, ok, refused(401, "auth", ""), skipped, ok, grantIndeterminate},
		{"not an auth failure", grantGrok, ok, refused(400, "invalid_request", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a 200 that carried an error", grantGrok, ok, grantStep{status: 200, kind: "other", requests: 1}, skipped, ok, grantIndeterminate},
		{"a 200 that carried an auth failure", grantGrok, ok, grantStep{status: 200, kind: "auth", requests: 1}, skipped, ok, grantIndeterminate},
		{"a late 200 that carried an error", grantGrok, ok, ok, grantStep{status: 200, kind: "other", requests: 1}, ok, grantIndeterminate},
		{"a 200 that carried an error, then a later use succeeded", grantGrok, ok, grantStep{status: 200, kind: "other", requests: 1}, ok, ok, grantIndeterminate},
		{"a 200 that carried an auth failure, then a later use succeeded", grantGrok, ok, grantStep{status: 200, kind: "auth", requests: 1}, ok, ok, grantIndeterminate},
		{"the baseline needed two requests", grantGrok, twice(ok), ok, ok, ok, grantIndeterminate},
		{"the baseline needed two requests and the rest is clean", grantPlan, twice(ok), skipped, skipped, skipped, grantIndeterminate},
		{"the first later use needed two requests", grantGrok, ok, twice(ok), ok, ok, grantIndeterminate},
		{"the second later use needed two requests", grantGrok, ok, ok, twice(ok), ok, grantIndeterminate},
		{"the control needed two requests", grantGrok, ok, ok, ok, twice(ok), grantIndeterminate},
		{"a dead grant that needed two requests", grantGrok, ok, twice(dead), skipped, ok, grantIndeterminate},
		{"a lazy ending that needed two requests", grantGrok, ok, ok, twice(dead), ok, grantIndeterminate},
		{"kilo kept", grantKilo, ok, ok, ok, ok, grantKept},
		{"kilo's 401", grantKilo, ok, refused(401, "auth", ""), skipped, ok, grantEnded},
		{"kilo's late 401", grantKilo, ok, ok, refused(401, "auth", "other"), ok, grantEnded},
		{"kilo's 400", grantKilo, ok, refused(400, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"kilo's 403", grantKilo, ok, refused(403, "other", ""), skipped, ok, grantIndeterminate},
		{"kilo's 403 as the SDK classifies it, an auth failure", grantKilo, ok, refused(403, "auth", ""), skipped, ok, grantIndeterminate},
		{"kilo's 403 auth failure with a code", grantKilo, ok, refused(403, "auth", "other"), skipped, ok, grantIndeterminate},
		{"kilo's late 403 auth failure", grantKilo, ok, ok, refused(403, "auth", ""), ok, grantIndeterminate},
		{"kilo's 404", grantKilo, ok, refused(404, "invalid_request", ""), skipped, ok, grantIndeterminate},
		{"kilo's 500", grantKilo, ok, refused(500, "unavailable", ""), skipped, ok, grantIndeterminate},
		{"kilo's 200 that carried an auth failure", grantKilo, ok, refused(200, "auth", ""), skipped, ok, grantIndeterminate},
		{"a terminal code at 403 is not dead", grantGrok, ok, refused(403, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a terminal code at 404 is not dead", grantChatGPT, ok, refused(404, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a terminal code at 422 is not dead", grantPlan, ok, refused(422, "auth", "invalid_client"), skipped, ok, grantIndeterminate},
		{"a terminal code at 429 is not dead", grantGrok, ok, refused(429, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a terminal code at 502 is not dead", grantChatGPT, ok, refused(502, "auth", "invalid_grant"), skipped, ok, grantIndeterminate},
		{"a late terminal code at 403 is not dead", grantPlan, ok, ok, refused(403, "auth", "invalid_grant"), ok, grantIndeterminate},
	} {
		if got := grantVerdict(c.provider, c.baseline, c.first, c.last, c.control); got != c.want {
			t.Errorf("%s: verdict %q, want %q", c.name, got, c.want)
		}
	}
}

// TestEarlierGrantSay_0081: every line of the output goes through the
// repository's redaction, as a last guard behind the fixed words.
func TestEarlierGrantSay_0081(t *testing.T) {
	var out []string
	g := &earlierGrant{provider: grantChatGPT, logf: func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }}
	g.say("%s", "Authorization: Bearer "+grantPlantedAccess)
	g.say("access_token=%s", grantPlantedAccess)
	if joined := strings.Join(out, "\n"); len(out) != 2 || strings.Contains(joined, grantPlantedAccess) {
		t.Errorf("the lines were not redacted: %d lines, one holds a planted token (%t)", len(out), strings.Contains(strings.Join(out, "\n"), grantPlantedAccess))
	}
	out = nil
	g.say("%s %s: status %d, kind %s, code %s, requests %d", grantChatGPT, "baseline", 200, "ok", "-", 1)
	if want := "chatgpt baseline: status 200, kind ok, code -, requests 1"; len(out) != 1 || out[0] != want {
		t.Errorf("redaction changed a step line: %q, want %q", out, want)
	}
}

// grantSeq is a transport that answers a sequence: a status and a body, or an
// error.
type grantSeq struct {
	status int
	body   string
	err    error
}

func grantSequence(t *testing.T, steps ...grantSeq) http.RoundTripper {
	t.Helper()
	i := 0
	return roundTripperFunc(func(*http.Request) (*http.Response, error) {
		if i >= len(steps) {
			t.Fatalf("a request beyond the %d the test planned", len(steps))
		}
		s := steps[i]
		i++
		if s.err != nil {
			return nil, s.err
		}
		return &http.Response{StatusCode: s.status, Body: io.NopCloser(strings.NewReader(s.body)), Header: http.Header{}}, nil
	})
}

// TestGrantRecorder_0081: the recorder reads no request, and no successful
// response's body, puts back the body of every refusal it reads, keeps a
// code only from the allow-list, tells the last response and the round trips
// a step made, and forgets a step's answers at the next.
func TestGrantRecorder_0081(t *testing.T) {
	// run sends one request, whose body the recorder must not read, through
	// the recorder, which is answered with status and body; it returns the
	// recorder, what the caller then reads of the answer, and whether the
	// recorder read the answer before the caller did.
	run := func(t *testing.T, status int, body string) (rec *grantRecorder, got string, readEarly bool) {
		t.Helper()
		request := &bodyProbe{Reader: strings.NewReader(grantPlantedAccess)}
		answer := &bodyProbe{Reader: strings.NewReader(body)}
		rec = &grantRecorder{next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: answer, Header: http.Header{}}, nil
		})}
		req, err := http.NewRequest(http.MethodPost, "https://issuer.invalid/token", request)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := rec.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if request.readBeforeCaller {
			t.Error("the recorder read the request's body")
		}
		readEarly = answer.readBeforeCaller
		answer.callerReading = true
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return rec, string(data), readEarly
	}
	t.Run("a refusal's code is kept and its body put back", func(t *testing.T) {
		for _, status := range []int{400, 401, 403, 429, 500, 503} {
			body := `{"error":"invalid_grant","error_description":"` + grantPlantedCode + `"}`
			rec, got, _ := run(t, status, body)
			if s, code, n := rec.last(); s != status || code != "invalid_grant" || n != 1 {
				t.Errorf("%d: recorded %d %q after %d round trips, want %d invalid_grant after 1", status, s, code, n, status)
			}
			if got != body {
				t.Errorf("%d: the body was not put back: %d bytes of %d", status, len(got), len(body))
			}
		}
	})
	t.Run("a long refusal is put back whole, and gives no code past what the SDK reads", func(t *testing.T) {
		for _, size := range []int{grantCodeLimit, grantCodeLimit + 1, 70000, 200000} {
			body := grantPadded(size)
			rec, got, _ := run(t, 400, body)
			_, code, _ := rec.last()
			want := ""
			if size == grantCodeLimit {
				want = "invalid_grant"
			}
			if got != body || code != want {
				t.Errorf("%d bytes: the caller got %d bytes, code %q; want the whole body and %q", size, len(got), code, want)
			}
		}
	})
	t.Run("a code outside the allow-list is other", func(t *testing.T) {
		rec, _, _ := run(t, 400, `{"error":"`+grantPlantedOther+`"}`)
		if _, code, _ := rec.last(); code != "other" {
			t.Errorf("recorded %q, want other", code)
		}
	})
	t.Run("a refusal with no OAuth error field has no code", func(t *testing.T) {
		for _, body := range []string{"not json " + grantPlantedCode, `{"message":"` + grantPlantedCode + `"}`} {
			rec, _, _ := run(t, 401, body)
			if status, code, _ := rec.last(); status != 401 || code != "" {
				t.Errorf("recorded %d %q, want 401 and no code", status, code)
			}
		}
	})
	t.Run("a success's body is not read", func(t *testing.T) {
		rec, got, readEarly := run(t, 200, `{"access_token":"`+grantPlantedAccess+`"}`)
		if status, code, _ := rec.last(); status != 200 || code != "" {
			t.Errorf("recorded %d %q, want 200 and no code", status, code)
		}
		if readEarly || got == "" {
			t.Errorf("the recorder read a successful response's body (%t), or the caller got none (%d bytes)", readEarly, len(got))
		}
	})
}

// TestGrantRecorderSteps_0081: the recorder tells the last response of a step
// and the round trips the step made, forgets the status of a response when a
// transport error follows it, and begins each step with nothing answered.
func TestGrantRecorderSteps_0081(t *testing.T) {
	send := func(rec *grantRecorder) {
		req, err := http.NewRequest(http.MethodGet, "https://issuer.invalid/x", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := rec.RoundTrip(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
	t.Run("the last response of a step is the one told, and each round trip is counted", func(t *testing.T) {
		rec := &grantRecorder{next: grantSequence(t, grantSeq{status: 503, body: `{"error":"x"}`}, grantSeq{status: 200, body: "{}"})}
		rec.begin()
		send(rec)
		send(rec)
		if status, code, n := rec.last(); status != 200 || code != "" || n != 2 {
			t.Errorf("recorded %d %q after %d round trips, want 200, no code, 2", status, code, n)
		}
	})
	t.Run("a transport error forgets the status of the response before it", func(t *testing.T) {
		rec := &grantRecorder{next: grantSequence(t, grantSeq{status: 401, body: `{"error":"invalid_grant"}`}, grantSeq{err: errors.New("connection refused")})}
		rec.begin()
		send(rec)
		send(rec)
		if status, code, n := rec.last(); status != 0 || code != "" || n != 2 {
			t.Errorf("recorded %d %q after %d round trips, want 0, no code, 2", status, code, n)
		}
	})
	t.Run("a step begins with nothing answered", func(t *testing.T) {
		rec := &grantRecorder{next: grantSequence(t, grantSeq{status: 500, body: `{"error":"invalid_grant"}`})}
		rec.begin()
		send(rec)
		rec.begin()
		if status, code, n := rec.last(); status != 0 || code != "" || n != 0 {
			t.Errorf("recorded %d %q after %d round trips, want nothing", status, code, n)
		}
	})
}

// grantPadded is a JSON body of exactly total bytes that holds the code
// invalid_grant in its first bytes and a padding after it.
func grantPadded(total int) string {
	const head = `{"error":"invalid_grant","pad":"`
	const tail = `"}`
	return head + strings.Repeat("x", total-len(head)-len(tail)) + tail
}

// TestGrantCodeAgreesWithTheSDK_0081: grantCodeOf is a copy of the SDK's own
// reader of a refresh refusal's code (refreshErrorCode in auth, unexported, so
// a test of this package cannot call it, and export_test.go reaches only
// auth's own tests). What an outside test can see of that reader is the
// SDK's verdict on a 400: it is an auth failure, a dead refresh token, exactly
// when the code it read is terminal. The test sends each body to the SDK's
// real refresh, as a 400, and fails when the SDK's verdict and the recorder's
// code disagree, and when the recorder does not give the expected code.
func TestGrantCodeAgreesWithTheSDK_0081(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"flat invalid_grant", `{"error":"invalid_grant"}`, "invalid_grant"},
		{"flat invalid_client", `{"error":"invalid_client"}`, "invalid_client"},
		{"flat refresh_token_expired", `{"error":"refresh_token_expired"}`, "refresh_token_expired"},
		{"flat refresh_token_reused", `{"error":"refresh_token_reused"}`, "refresh_token_reused"},
		{"flat refresh_token_invalidated", `{"error":"refresh_token_invalidated"}`, "refresh_token_invalidated"},
		{"nested invalid_grant", `{"error":{"code":"invalid_grant","message":"m"}}`, "invalid_grant"},
		{"nested refresh_token_expired", `{"error":{"code":"refresh_token_expired"}}`, "refresh_token_expired"},
		{"nested refresh_token_reused", `{"error":{"code":"refresh_token_reused"}}`, "refresh_token_reused"},
		{"nested refresh_token_invalidated", `{"error":{"code":"refresh_token_invalidated"}}`, "refresh_token_invalidated"},
		{"bare invalid_grant", `{"code":"invalid_grant"}`, "invalid_grant"},
		{"bare refresh_token_reused", `{"code":"refresh_token_reused","message":"m"}`, "refresh_token_reused"},
		{"upper flat", `{"error":"INVALID_GRANT"}`, "invalid_grant"},
		{"mixed nested", `{"error":{"code":"Refresh_Token_Expired"}}`, "refresh_token_expired"},
		{"upper bare", `{"code":"INVALID_CLIENT"}`, "invalid_client"},
		{"an empty error string, then a bare code", `{"error":"","code":"invalid_grant"}`, "invalid_grant"},
		{"a flat error beats a bare code", `{"error":"x","code":"invalid_grant"}`, "other"},
		{"an error object with no code, then a bare code", `{"error":{"message":"m"},"code":"invalid_grant"}`, "invalid_grant"},
		{"an unlisted flat code", `{"error":"server_error"}`, "other"},
		{"an unlisted nested code", `{"error":{"code":"server_error"}}`, "other"},
		{"an unlisted bare code", `{"code":"slow_down"}`, "other"},
		{"the code in the message only", `{"error":{"message":"invalid_grant"}}`, ""},
		{"a null error", `{"error":null}`, ""},
		{"a numeric nested code", `{"error":{"code":5}}`, ""},
		{"a numeric bare code", `{"code":5}`, ""},
		{"an array", `[]`, ""},
		{"an empty body", ``, ""},
		{"text", `invalid_grant`, ""},
		// The SDK reads the first 2048 bytes of a refusal: a body that is valid
		// JSON in them gives a code, a longer one gives none.
		{"a body of exactly 2048 bytes", grantPadded(grantCodeLimit), "invalid_grant"},
		{"a body of 2049 bytes", grantPadded(grantCodeLimit + 1), ""},
		{"a body of about 3 KB holding the code", grantPadded(3000), ""},
		{"a body of 70000 bytes", grantPadded(70000), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, c.body)
			}))
			defer srv.Close()
			rec := &grantRecorder{next: srv.Client().Transport}
			session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: srv.URL, TokenURL: srv.URL + "/token",
				Access: "a", Refresh: "r-made-up", Expiry: time.Now().Add(-time.Hour), ClientID: "c-made-up", HTTPClient: rec.client()}
			rec.begin()
			_, err := session.Token(twinContext(t))
			if err == nil {
				t.Fatal("the refresh of a 400 succeeded")
			}
			_, code, _ := rec.last()
			if sdkDead, probeDead := errors.Is(err, llmprovider.ErrAuthFailure), slices.Contains(grantTerminalCodes, code); sdkDead != probeDead {
				t.Errorf("the SDK calls the refresh token dead: %t; the recorder's code %q is terminal: %t", sdkDead, code, probeDead)
			}
			if code != c.want {
				t.Errorf("the recorder's code is %q, want %q", code, c.want)
			}
			if got := grantCodeOf([]byte(c.body[:min(len(c.body), grantCodeLimit)])); got != c.want {
				t.Errorf("grantCodeOf = %q, want %q", got, c.want)
			}
		})
	}
}

// bodyProbe is a body that says whether it was read before the caller's own
// read of it, which the caller marks by setting callerReading.
type bodyProbe struct {
	io.Reader
	readBeforeCaller bool
	callerReading    bool
}

func (b *bodyProbe) Read(p []byte) (int, error) {
	if !b.callerReading {
		b.readBeforeCaller = true
	}
	return b.Reader.Read(p)
}

func (b *bodyProbe) Close() error { return nil }
