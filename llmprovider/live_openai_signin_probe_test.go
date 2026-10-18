//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// The OpenAI "Sign in with ChatGPT" API flow with dynamic client registration
// (0017-MADR D4; 0017-PLAN U4; 0017-REPORT P3), as pi implements it. Only this
// probe uses it: it changes no default and adds no API.
const (
	signinDynamicClientID = "dynamic_agent_client"
	signinAgentNameHint   = "go-llmprovider-sdk"
	signinAuthorizeURL    = "https://auth.openai.com/api/accounts/authorize"
	signinTokenURL        = "https://auth.openai.com/api/accounts/oauth/token"
	signinResource        = "https://api.openai.com/v1"
	signinListenAddr      = "127.0.0.1:1455"
	signinRedirectURI     = "http://127.0.0.1:1455/auth/callback"
	signinDirectScope     = "chatgpt.tokens.use.direct"
	signinScope           = "openid profile email offline_access resource.invoke " + signinDirectScope
	signinDiscoveryURL    = "https://auth.openai.com/.well-known/openid-configuration"
	signinModel           = "gpt-6-luna"
)

// TestLive_OpenAISigninProbe is 0017-PLAN U4's probe. It REQUIRES
// LLMPROVIDER_LIVE_OPENAI_SIGNIN=1 and a person to sign in at the URL it logs
// within five minutes. Each run registers a new client at OpenAI under this
// module's agent name.
//
// It records, without logging any token or the issued client id:
//   - whether registration succeeds, and the issued client id's form;
//   - the granted scopes;
//   - the model listing's shape; the rest uses its first listed model
//     (0017-PLAN U5);
//   - two raw streamed requests, OpenAI's documented minimum and the same
//     with max_output_tokens (U5);
//   - one text and one tool call on api.openai.com/v1/responses, and which
//     request options the token is refused;
//   - a streamed text and a streamed tool call through openai.New with
//     WithStore(false) (U5);
//   - a refresh with the issued client id, a revocation at the discovery
//     document's revocation_endpoint, and a refresh after it (U5).
//
// The flow failing fails the test. A refused generation, refresh or
// revocation is a finding, logged as REFUSED, not a failure.
func TestLive_OpenAISigninProbe(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_OPENAI_SIGNIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_OPENAI_SIGNIN unset: this registers a client at OpenAI and needs a person to sign in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	verifier, state, nonce := probeRandom(t), probeRandom(t), probeRandom(t)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	hostID := "urn:uuid:" + probeUUID(t)

	type callback struct {
		code, clientID string
		err            error
	}
	got := make(chan callback, 1)
	listener, err := net.Listen("tcp", signinListenAddr)
	if err != nil {
		t.Fatalf("listen on %s: %v", signinListenAddr, err)
	}
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/callback" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		var cb callback
		switch {
		case q.Get("error") != "":
			cb.err = fmt.Errorf("authorization failed: %s: %s", q.Get("error"), redact.String(q.Get("error_description")))
		case q.Get("state") != state:
			cb.err = errors.New("callback state mismatch")
		case q.Get("code") == "":
			cb.err = errors.New("callback has no code")
		case strings.TrimSpace(q.Get("client_id")) == "":
			cb.err = errors.New("callback has no issued client_id: registration did not happen")
		default:
			cb.code, cb.clientID = q.Get("code"), strings.TrimSpace(q.Get("client_id"))
		}
		_, _ = io.WriteString(w, "go-llmprovider-sdk probe: you can close this window.")
		select {
		case got <- cb:
		default:
		}
	})}
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Close() })

	authorize, _ := url.Parse(signinAuthorizeURL)
	authorize.RawQuery = url.Values{
		"client_id":             {signinDynamicClientID},
		"agent_name_hint":       {signinAgentNameHint},
		"ext_agent_host_id":     {hostID},
		"response_type":         {"code"},
		"redirect_uri":          {signinRedirectURI},
		"resource":              {signinResource},
		"scope":                 {signinScope},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"nonce":                 {nonce},
	}.Encode()
	t.Logf("open this URL and sign in: %s", authorize)

	var cb callback
	select {
	case cb = <-got:
	case <-ctx.Done():
		t.Fatalf("no callback within the time limit: %v", ctx.Err())
	}
	if cb.err != nil {
		t.Fatalf("registration: %v", cb.err)
	}
	t.Logf("FINDING registration: succeeded; issued client id %s", probeShape(cb.clientID))

	token := probeExchange(ctx, t, cb.code, verifier, cb.clientID)
	scopes := strings.Fields(token.Scope)
	t.Logf("FINDING scopes granted: %q; direct scope %s: %t", scopes, signinDirectScope, slices.Contains(scopes, signinDirectScope))
	t.Logf("FINDING token response: access %s, refresh present %t, id_token present %t, expires_in %d s",
		probeShape(token.AccessToken), token.RefreshToken != "", token.IDToken != "", token.ExpiresIn)
	if token.AccessToken == "" {
		t.Fatal("the token response has no access_token")
	}
	// The rest gets its own time, however long the sign-in took.
	ctx, stop := context.WithTimeout(context.Background(), 3*time.Minute)
	defer stop()

	model := probeModel(ctx, t, token.AccessToken)
	for _, tc := range []struct {
		name  string
		extra map[string]any
	}{
		{"raw streamed, the documented minimum", nil},
		{"raw streamed, with max_output_tokens", map[string]any{"max_output_tokens": 64}},
	} {
		body := map[string]any{
			"model":  model,
			"input":  []map[string]any{{"role": "user", "content": "Reply with only the word ALPHA"}},
			"store":  false,
			"stream": true,
		}
		maps.Copy(body, tc.extra)
		status, reply, err := probeDo(ctx, http.MethodPost, signinResource+"/responses", token.AccessToken, body)
		if err != nil {
			t.Logf("FINDING %s: REFUSED: %v", tc.name, err)
			continue
		}
		t.Logf("FINDING %s: %s", tc.name, probeStreamOutcome(status, reply))
	}

	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		tool bool
	}{
		{"text", nil, false},
		{"tool call", nil, true},
		{"text with WithStore(false)", []llmprovider.Option{openai.WithStore(false)}, false},
		{"text with reasoning effort low", []llmprovider.Option{llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortLow})}, false},
	} {
		p, err := openai.New(append([]llmprovider.Option{llmprovider.WithAPIKey(token.AccessToken), llmprovider.WithModel(model)}, tc.opts...)...)
		if err != nil {
			t.Fatalf("%s: openai.New: %v", tc.name, err)
		}
		req := userText("Reply with only the word ALPHA")
		if tc.tool {
			req = userText("What is the weather in Paris?")
			req.Tools = []llmprovider.Tool{tool}
			call, err := llmprovider.GenerateToolCall(ctx, p, req)
			probeOutcome(t, tc.name, fmt.Sprintf("call %s(%s)", call.Name, call.Arguments), err)
			continue
		}
		out, err := llmprovider.GenerateText(ctx, p, req)
		probeOutcome(t, tc.name, fmt.Sprintf("text %q", out), err)
	}

	for _, tc := range []struct {
		name string
		tool bool
	}{
		{"streamed text with WithStore(false)", false},
		{"streamed tool call with WithStore(false)", true},
	} {
		p, err := openai.New(llmprovider.WithAPIKey(token.AccessToken), llmprovider.WithModel(model), openai.WithStore(false))
		if err != nil {
			t.Fatalf("%s: openai.New: %v", tc.name, err)
		}
		req := userText("Reply with only the word ALPHA")
		if tc.tool {
			req = userText("What is the weather in Paris?")
			req.Tools = []llmprovider.Tool{tool}
			req.ToolChoice = llmprovider.ForceTool(tool.Name)
		}
		resp, deltas, err := probeStream(ctx, p, req)
		if err != nil {
			probeOutcome(t, tc.name, "", err)
			continue
		}
		ok := fmt.Sprintf("text %q after %d text deltas", resp.OutputText(), deltas)
		if tc.tool {
			ok = "no call"
			for _, item := range resp.Output {
				if call, isCall := item.(llmprovider.FunctionCallItem); isCall {
					ok = fmt.Sprintf("call %s(%s)", call.Name, call.Arguments)
				}
			}
		}
		probeOutcome(t, tc.name, ok, nil)
	}

	refresh := token.RefreshToken
	status, reply, err := probeForm(ctx, signinTokenURL, probeRefreshForm(cb.clientID, refresh))
	switch {
	case err != nil:
		t.Logf("FINDING refresh with the issued client id: REFUSED: %v", err)
	case status != http.StatusOK:
		t.Logf("FINDING refresh with the issued client id: %s", probeRefusal(status, reply))
	default:
		var renewed probeToken
		if err := json.Unmarshal(reply, &renewed); err != nil {
			t.Logf("FINDING refresh with the issued client id: HTTP 200, a body that is not a token response: %v", err)
			break
		}
		t.Logf("FINDING refresh with the issued client id: accepted; access %s, refresh present %t, rotated %t, scopes %q, expires_in %d s",
			probeShape(renewed.AccessToken), renewed.RefreshToken != "", renewed.RefreshToken != "" && renewed.RefreshToken != refresh,
			strings.Fields(renewed.Scope), renewed.ExpiresIn)
		if renewed.RefreshToken != "" {
			refresh = renewed.RefreshToken
		}
	}

	status, reply, err = probeDo(ctx, http.MethodGet, signinDiscoveryURL, "", nil)
	if err != nil || status != http.StatusOK {
		t.Logf("FINDING revocation: no discovery document: HTTP %d, %v", status, err)
		return
	}
	endpoint, err := probeRevocationEndpoint(reply)
	if err != nil {
		t.Logf("FINDING revocation: %v", err)
		return
	}
	status, reply, err = probeForm(ctx, endpoint, url.Values{
		"token":           {refresh},
		"token_type_hint": {"refresh_token"},
		"client_id":       {cb.clientID},
	})
	switch {
	case err != nil:
		t.Logf("FINDING revocation at %s: REFUSED: %v", endpoint, err)
	case status != http.StatusOK:
		t.Logf("FINDING revocation at %s: %s", endpoint, probeRefusal(status, reply))
	default:
		t.Logf("FINDING revocation at %s: HTTP 200, a body of %d bytes", endpoint, len(reply))
	}
	status, reply, err = probeForm(ctx, signinTokenURL, probeRefreshForm(cb.clientID, refresh))
	switch {
	case err != nil:
		t.Logf("FINDING refresh after revocation: REFUSED: %v", err)
	case status == http.StatusOK:
		t.Logf("FINDING refresh after revocation: ACCEPTED, so the revocation did not hold")
	default:
		t.Logf("FINDING refresh after revocation: %s", probeRefusal(status, reply))
	}
}

// probeModel reads the token's model listing, logs its shape, and returns
// the first model a picker shows, or signinModel when there is none.
func probeModel(ctx context.Context, t *testing.T, token string) string {
	t.Helper()
	status, body, err := probeDo(ctx, http.MethodGet, signinResource+"/models", token, nil)
	switch {
	case err != nil:
		t.Logf("FINDING listing: REFUSED: %v; using %s", err, signinModel)
		return signinModel
	case status != http.StatusOK:
		t.Logf("FINDING listing: %s; using %s", probeRefusal(status, body), signinModel)
		return signinModel
	}
	s, err := probeListing(body)
	if err != nil {
		t.Logf("FINDING listing: HTTP 200, top-level keys %q: %v; using %s", s.Keys, err, signinModel)
		return signinModel
	}
	t.Logf("FINDING listing: HTTP 200, top-level keys %q, %d models, %d with visibility \"list\", the first %q",
		s.Keys, s.Count, s.Listed, s.First)
	if s.First == "" {
		t.Logf("NOTE no listed model; using %s", signinModel)
		return signinModel
	}
	return s.First
}

// probeStream streams req from p, and returns the response and the number of
// text deltas before it.
func probeStream(ctx context.Context, p llmprovider.Provider, req *llmprovider.Request) (*llmprovider.Response, int, error) {
	deltas := 0
	for ev, err := range llmprovider.Stream(ctx, p, req) {
		if err != nil {
			return nil, deltas, err
		}
		switch ev.Type {
		case llmprovider.EventTextDelta:
			deltas++
		case llmprovider.EventDone:
			return ev.Response, deltas, nil
		}
	}
	return nil, deltas, errors.New("the stream ended with no EventDone")
}

// probeRefreshForm is a refresh with the issued client id, without scope,
// which keeps the grant (OpenAI's profiles-and-sessions page).
func probeRefreshForm(clientID, refresh string) url.Values {
	return url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refresh},
		"resource":      {signinResource},
	}
}

// probeDo sends a request with an optional bearer token and JSON body, and
// returns the status and up to 1 MiB of the reply.
func probeDo(ctx context.Context, method, target, token string, body map[string]any) (int, []byte, error) {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return probeSend(req)
}

// probeForm posts a form, and returns the status and up to 1 MiB of the
// reply.
func probeForm(ctx context.Context, target string, form url.Values) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return probeSend(req)
}

func probeSend(req *http.Request) (int, []byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, body, err
}

type probeToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

// probeExchange trades the code for tokens with the issued client id.
func probeExchange(ctx context.Context, t *testing.T, code, verifier, clientID string) probeToken {
	t.Helper()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {signinRedirectURI},
		"resource":      {signinResource},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signinTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatalf("token response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token request: HTTP %d: %s", resp.StatusCode, redact.String(string(body)))
	}
	var tok probeToken
	if err := json.Unmarshal(body, &tok); err != nil {
		t.Fatalf("token response is not JSON: %v", err)
	}
	return tok
}

// probeOutcome logs a generation's result, or its refusal with the kind and
// the service's redacted message.
func probeOutcome(t *testing.T, name, ok string, err error) {
	t.Helper()
	if err == nil {
		t.Logf("FINDING %s: accepted, %s", name, ok)
		return
	}
	if apiErr, isAPI := errors.AsType[*llmprovider.APIError](err); isAPI {
		t.Logf("FINDING %s: REFUSED, HTTP %d, code %q: %s", name, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	t.Logf("FINDING %s: REFUSED: %v", name, err)
}

// probeShape describes a credential without revealing it: its length, and
// its prefix up to the first separator, at most four characters.
func probeShape(s string) string {
	prefix := s
	if i := strings.IndexAny(s, "_-."); i >= 0 {
		prefix = s[:i]
	}
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}
	return fmt.Sprintf("of %d characters, starting %q", len(s), prefix)
}

func probeRandom(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// probeUUID is a random version 4 UUID, the per-run agent host id.
func probeUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
