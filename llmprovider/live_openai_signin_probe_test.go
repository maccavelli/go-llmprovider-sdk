//go:build live_gateways

package llmprovider_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
//   - one text and one tool call on api.openai.com/v1/responses, and which
//     request options the token is refused.
//
// The flow failing fails the test. A refused generation is a finding, logged
// as REFUSED, not a failure.
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
		p, err := openai.New(append([]llmprovider.Option{llmprovider.WithAPIKey(token.AccessToken), llmprovider.WithModel(signinModel)}, tc.opts...)...)
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
	t.Log("NOTE no revocation endpoint is known for this client, so the session is left to expire")
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
