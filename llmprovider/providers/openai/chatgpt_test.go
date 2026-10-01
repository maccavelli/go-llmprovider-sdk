package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's openai_chatgpt_test.go, chatgpt_stream_test.go,
// chatgpt_fedramp_test.go, vendor_session_test.go, responses_store_test.go,
// command_token_test.go and transport_defaults_test.go (0015-PLAN S7).

func sessionProvider(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	return build(t, append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
}

func TestOpenAI_ChatGPTSessionDoesNotHitPlatformHost(t *testing.T) {
	t.Parallel()
	var captured *http.Request
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		captured = request
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	session := &llmprovider.OAuthSession{Issuer: llmprovider.DefaultOpenAIIssuer, Access: "sess", Refresh: "refresh",
		Expiry: time.Now().Add(time.Hour)}
	p := sessionProvider(t, session, "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if captured == nil {
		t.Fatal("Generate() made no request")
	}
	if captured.URL.Host != "chatgpt.com" || captured.URL.Path != "/backend-api/codex/responses" {
		t.Fatalf("request URL = %s, want chatgpt.com/backend-api/codex/responses", captured.URL.Redacted())
	}
}

func TestOpenAI_StaticKeyDoesNotHitChatGPTHost(t *testing.T) {
	t.Parallel()
	var captured *http.Request
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		captured = request
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	p := sessionProvider(t, llmprovider.NewStaticToken("sk-test"), "gpt-4.1-mini", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if captured == nil || captured.URL.Host != "api.openai.com" || captured.URL.Path != "/v1/responses" {
		t.Fatalf("request URL = %v, want api.openai.com/v1/responses", captured)
	}
}

// TestNew_APIKeyStillPlatform was TestNewOpenAI_APIKeyStillPlatform: the key
// shorthand goes to the platform.
func TestNew_APIKeyStillPlatform(t *testing.T) {
	t.Parallel()
	var captured *http.Request
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		captured = request
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	p := build(t, llmprovider.WithAPIKey("sk-test"), llmprovider.WithModel("gpt-4.1-mini"), llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if captured == nil || captured.URL.Host != "api.openai.com" || captured.URL.Path != "/v1/responses" {
		t.Fatalf("request URL = %v, want api.openai.com/v1/responses", captured)
	}
}

func TestOpenAI_ChatGPTSetsAccountHeader(t *testing.T) {
	t.Parallel()
	var accountHeader string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		accountHeader = request.Header.Get("ChatGPT-Account-Id")
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	session := &llmprovider.OAuthSession{Issuer: llmprovider.DefaultOpenAIIssuer, Access: "sess", Refresh: "refresh",
		Expiry: time.Now().Add(time.Hour), AccountID: "acct_1"}
	p := sessionProvider(t, session, "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if accountHeader != "acct_1" {
		t.Fatalf("ChatGPT-Account-Id = %q, want acct_1", accountHeader)
	}
}

func TestOpenAI_ChatGPTSetsOriginatorHeader(t *testing.T) {
	t.Parallel()
	var originator string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		originator = request.Header.Get(llmprovider.ChatGPTOriginatorHeader)
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	p := sessionProvider(t, chatGPTSession(), "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if originator != llmprovider.ChatGPTOriginatorValue {
		t.Fatalf("originator = %q, want %q", originator, llmprovider.ChatGPTOriginatorValue)
	}
}

// TestOpenAI_ChatGPTOmitsMaxOutputTokens answers MADR 0008 open question 1:
// the ChatGPT backend rejects max_output_tokens (400 "Unsupported parameter",
// gate G-C 2026-09-27), so a ChatGPT session never sends it.
func TestOpenAI_ChatGPTOmitsMaxOutputTokens(t *testing.T) {
	t.Parallel()
	var body map[string]any
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		return httpResponse(request, http.StatusOK, okResponse), nil
	})}
	p := sessionProvider(t, chatGPTSession(), "gpt-5.4-mini", llmprovider.WithHTTPClient(client), llmprovider.WithMaxTokens(321))
	req := text("hello")
	req.MaxOutputTokens = 99 // the new API's per-request limit is not sent either
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got, ok := body["max_output_tokens"]; ok {
		t.Fatalf("max_output_tokens = %v, want absent", got)
	}
}

func TestOpenAI_ChatGPTSetsResidencyHeader(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		claims map[string]any
		want   string
	}{
		{"namespaced", map[string]any{"https://api.openai.com/auth": map[string]any{"chatgpt_compute_residency": "eu"}}, "eu"},
		{"root_fallback", map[string]any{"chatgpt_compute_residency": "us"}, "us"},
		{"namespaced_no_constraint_wins", map[string]any{
			"chatgpt_compute_residency":   "eu",
			"https://api.openai.com/auth": map[string]any{"chatgpt_compute_residency": "no_constraint"},
		}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var residencyHeader string
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				residencyHeader = request.Header.Get("x-openai-internal-codex-residency")
				return httpResponse(request, http.StatusOK, okResponse), nil
			})}
			session := &llmprovider.OAuthSession{Issuer: llmprovider.DefaultOpenAIIssuer, Access: testJWT(t, test.claims),
				Refresh: "refresh", Expiry: time.Now().Add(time.Hour)}
			p := sessionProvider(t, session, "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
			if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if residencyHeader != test.want {
				t.Fatalf("residency header = %q, want %q", residencyHeader, test.want)
			}
		})
	}
}

func TestResidency_UnreadableTokens(t *testing.T) {
	for _, token := range []string{"not-a-jwt", "a.!!!.c", "a." + "bm90IGpzb24" + ".c"} {
		if got := residency(token); got != "" {
			t.Errorf("residency(%q) = %q, want empty", token, got)
		}
	}
}

func TestOpenAI_OAuth401RetriesOnceAfterRefresh(t *testing.T) {
	t.Parallel()
	var generateCalls, refreshCalls int
	client := &http.Client{}
	client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "chatgpt.com":
			generateCalls++
			if generateCalls == 1 {
				if got := request.Header.Get("Authorization"); got != "Bearer old-access" {
					t.Errorf("first Authorization = %q", got)
				}
				return httpResponse(request, http.StatusUnauthorized, ""), nil
			}
			if got := request.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Errorf("retry Authorization = %q", got)
			}
			return httpResponse(request, http.StatusOK, okResponse), nil
		case "auth.test":
			refreshCalls++
			var got map[string]string
			if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
				t.Errorf("decode refresh body: %v", err)
			}
			want := map[string]string{
				"client_id":     llmprovider.DefaultOpenAIClientID,
				"grant_type":    "refresh_token",
				"refresh_token": "old-refresh",
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("refresh body = %v, want the JSON grant %v (MADR 0012 §5.2)", got, want)
			}
			return httpResponse(request, http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`), nil
		default:
			return nil, errors.New("unexpected request host")
		}
	})
	session := &llmprovider.OAuthSession{
		Provider:   llmprovider.ProviderOpenAI,
		Issuer:     llmprovider.DefaultOpenAIIssuer,
		ClientID:   llmprovider.DefaultOpenAIClientID,
		Access:     "old-access",
		Refresh:    "old-refresh",
		Expiry:     time.Now().Add(time.Hour),
		TokenURL:   "https://auth.test/oauth/token",
		HTTPClient: client,
	}
	p := sessionProvider(t, session, "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generateCalls != 2 || refreshCalls != 1 {
		t.Fatalf("generate/refresh calls = %d/%d, want 2/1", generateCalls, refreshCalls)
	}
}

// rotatingSource is an InvalidatingSource that hands out key-1, then key-2
// after Invalidate, as a CommandToken does when its command prints a new key.
type rotatingSource struct {
	mu          sync.Mutex
	key         int
	invalidated int
}

func (s *rotatingSource) Token(context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return llmprovider.Token{Value: fmt.Sprintf("key-%d", s.key+1), Type: llmprovider.TokenAPIKey}, nil
}

func (s *rotatingSource) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key++
	s.invalidated++
}

// TestOpenAI_RetriesOnceAfterInvalidate is the provider half of
// llmprovider's old TestCommandToken_RerunAfter401: a 401 invalidates the
// source and retries once, with its fresh token. llmprovider keeps the source
// half, TestCommandToken_RerunsAfterInvalidate.
func TestOpenAI_RetriesOnceAfterInvalidate(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer key-2" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"expired key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"r","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`))
	}))
	t.Cleanup(srv.Close)
	src := &rotatingSource{}
	p := sessionProvider(t, src, "gpt-5.5", llmprovider.WithBaseURL(srv.URL))
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil || out != "hello" || src.invalidated != 1 || len(seen) != 2 {
		t.Fatalf("Generate = %q, %v after %d invalidations (sent %v); want one retry after the 401", out, err, src.invalidated, seen)
	}
}

// TestOpenAIChatGPT_SendsCodexRequest: the backend rejects a non-streaming
// request, one without store:false and one with max_output_tokens (gate G-C);
// Codex also sends include, prompt_cache_key and session-id.
func TestOpenAIChatGPT_SendsCodexRequest(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, chatGPTSession(), "gpt-6-astra",
		llmprovider.WithHTTPClient(client), llmprovider.WithMaxTokens(321), llmprovider.WithSessionID("sess-42"))
	_, genErr := llmprovider.GenerateText(context.Background(), p, text("hello"))
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body["stream"] != true || c.body["store"] != false || c.body["prompt_cache_key"] != "sess-42" {
		t.Errorf("stream/store/prompt_cache_key = %v/%v/%v, want true/false/sess-42",
			c.body["stream"], c.body["store"], c.body["prompt_cache_key"])
	}
	if include, _ := c.body["include"].([]any); !slices.Equal(include, []any{"reasoning.encrypted_content"}) {
		t.Errorf("include = %v", c.body["include"])
	}
	if v, ok := c.body["max_output_tokens"]; ok {
		t.Errorf("max_output_tokens = %v, want absent", v)
	}
	if c.header.Get("Accept") != "text/event-stream" || c.header.Get("session-id") != "sess-42" {
		t.Errorf("Accept = %q, session-id = %q", c.header.Get("Accept"), c.header.Get("session-id"))
	}
	if genErr != nil {
		t.Errorf("Generate: %v", genErr)
	}
}

// TestOpenAIChatGPT_DecodesRecordedStreams replays gate G-C's recordings
// (2026-09-27, gpt-6-astra; account identifiers redacted).
func TestOpenAIChatGPT_DecodesRecordedStreams(t *testing.T) {
	for _, tc := range []struct {
		fixture, id, text string
		call              *llmprovider.FunctionCallItem
	}{
		{"chatgpt-text.sse", "resp_01ae07221376cd90016ab96e72ca3c87d1a3c12bb5aaf6db92", "ok", nil},
		{"chatgpt-reasoning.sse", "resp_047ea2fee4cf7ab6016ab96e77a3f087d1ad70bca0c3b0cd8d", "391", nil},
		{"chatgpt-tool.sse", "resp_059e69aebb680141016ab96e74543487d1af68b2af975de82f", "",
			&llmprovider.FunctionCallItem{CallID: "call_SAcT3cGrAgagMgcNJEcgK62C", Name: "record", Arguments: `{"subject":"fix: probe"}`}},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			client, _ := chatGPTClient(t, fixture(t, tc.fixture))
			p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
			res, err := p.Generate(context.Background(), text("hi"))
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if res.ID != tc.id || res.OutputText() != tc.text {
				t.Errorf("ID = %q, text = %q; want %q, %q", res.ID, res.OutputText(), tc.id, tc.text)
			}
			var calls []llmprovider.FunctionCallItem
			for _, it := range res.Output {
				if fc, ok := it.(llmprovider.FunctionCallItem); ok {
					calls = append(calls, fc)
				}
			}
			if (tc.call == nil) != (len(calls) == 0) || (tc.call != nil && (len(calls) != 1 || calls[0] != *tc.call)) {
				t.Errorf("function calls = %+v, want %+v", calls, tc.call)
			}
		})
	}
}

// sseEvents renders data-only events, as the backend frames them.
func sseEvents(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: " + e + "\n\n")
	}
	return b.String()
}

const sseCreated = `{"type":"response.created","response":{"id":"resp_1","status":"in_progress"}}`

func sseFailed(code string) string {
	return `{"type":"response.failed","response":{"id":"resp_1","status":"failed","error":{"code":"` + code +
		`","message":"` + code + ` happened"}}}`
}

// TestOpenAIChatGPT_StreamFailures: response.failed codes map onto MADR 0012
// §1.1 as Codex's parse_failed_response classifies them, response.incomplete
// onto MADR 0012 §1.5, and a stream that ends before response.completed is
// retryable. "terminal" is read through Retryable, which replaced Terminal
// (0015-MADR D7); for these kinds the two say the same.
func TestOpenAIChatGPT_StreamFailures(t *testing.T) {
	for _, tc := range []struct {
		name, stream string
		want         error
		terminal     bool
	}{
		{"usage_not_included", sseEvents(sseCreated, sseFailed("usage_not_included")), llmprovider.ErrNotPermitted, true},
		{"usage_limit_reached", sseEvents(sseCreated, sseFailed("usage_limit_reached")), llmprovider.ErrQuotaExhausted, true},
		{"context_length_exceeded", sseEvents(sseCreated, sseFailed("context_length_exceeded")), llmprovider.ErrInvalidRequest, true},
		{"rate_limit_exceeded", sseEvents(sseCreated, sseFailed("rate_limit_exceeded")), llmprovider.ErrRateLimited, false},
		{"server_error", sseEvents(sseCreated, sseFailed("server_error")), llmprovider.ErrProviderUnavailable, false},
		{"truncated", sseEvents(sseCreated), llmprovider.ErrProviderUnavailable, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := chatGPTClient(t, tc.stream)
			p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
			_, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if !errors.Is(tc.want, llmprovider.ErrInvalidRequest) && errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("err = %v also matches ErrInvalidRequest", err)
			}
			var apiErr *llmprovider.APIError
			isAPI := errors.As(err, &apiErr)
			if tc.name == "rate_limit_exceeded" && !isAPI {
				// A stream rate limit is an *APIError too (0015-MADR D7).
				t.Fatalf("err = %#v, want an *APIError", err)
			}
			if isAPI && apiErr.Retryable() == tc.terminal {
				t.Errorf("Retryable() = %t, want %t", apiErr.Retryable(), !tc.terminal)
			}
		})
	}
	t.Run("incomplete", func(t *testing.T) {
		client, _ := chatGPTClient(t, sseEvents(sseCreated,
			`{"type":"response.incomplete","response":{"id":"resp_1","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`))
		p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
		_, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
		var inc *llmprovider.APIError
		if !errors.As(err, &inc) || !errors.Is(inc.Kind, llmprovider.ErrIncomplete) || inc.Reason != "max_output_tokens" {
			t.Fatalf("err = %v, want an *APIError of kind ErrIncomplete, reason max_output_tokens", err)
		}
	})
}

// TestOpenAIChatGPT_ContinueIsUnsupported was TestOpenAIChatGPT_ContinueIsInvalid:
// with store:false there is no response to chain from, so a continuation
// fails without a request (MADR 0012 §4.2). The kind is ErrUnsupported, not
// ErrInvalidRequest: a need a provider does not support is refused with it
// before the network (0015-MADR D4).
func TestOpenAIChatGPT_ContinueIsUnsupported(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
	if p.Capabilities().Continuation != llmprovider.Unsupported {
		t.Fatalf("Continuation = %v, want Unsupported", p.Capabilities().Continuation)
	}
	req := text("more")
	req.PreviousResponseID = "resp_prev"
	_, err := p.Generate(context.Background(), req)
	c.mu.Lock()
	defer c.mu.Unlock()
	if !errors.Is(err, llmprovider.ErrUnsupported) || c.calls != 0 {
		t.Fatalf("err = %v after %d requests, want ErrUnsupported and none", err, c.calls)
	}
}

// TestOpenAIPlatform_KeepsJSONRequest: an API key keeps the platform shape:
// max_output_tokens, no stream or store, a JSON response.
func TestOpenAIPlatform_KeepsJSONRequest(t *testing.T) {
	client, c := chatGPTClient(t, `{"id":"resp_p","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	p := build(t, llmprovider.WithAPIKey("sk-test"), llmprovider.WithModel("gpt-6-luna"),
		llmprovider.WithHTTPClient(client), llmprovider.WithMaxTokens(321))
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil || out != "ok" {
		t.Fatalf("Generate = %q, %v", out, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if got, _ := c.body["max_output_tokens"].(float64); got != 321 {
		t.Errorf("max_output_tokens = %v, want 321", c.body["max_output_tokens"])
	}
	for _, k := range []string{"stream", "store", "include", "prompt_cache_key"} {
		if v, ok := c.body[k]; ok {
			t.Errorf("%s = %v, want absent", k, v)
		}
	}
}

// fedrampHeader returns X-OpenAI-Fedramp as one ChatGPT call sent it.
func fedrampHeader(t *testing.T, session *llmprovider.OAuthSession) (string, bool) {
	t.Helper()
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, session, "gpt-6-astra", llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.header["X-Openai-Fedramp"]
	if !ok {
		return "", false
	}
	return v[0], true
}

// TestOpenAIChatGPT_FedRAMPHeader: a FedRAMP session sends X-OpenAI-Fedramp:
// true, as Codex's bearer auth does (model-provider/src/bearer_auth_provider.rs:43-45).
// The old test built the session through the login, from an id token's
// chatgpt_account_is_fedramp; llmprovider keeps that half,
// TestChatGPTLogin_FedRAMPClaim.
func TestOpenAIChatGPT_FedRAMPHeader(t *testing.T) {
	session := chatGPTSession()
	session.AccountID, session.FedRAMP = "acct", true
	if v, ok := fedrampHeader(t, session); !ok || v != "true" {
		t.Fatalf("X-OpenAI-Fedramp = %q (present %t), want true", v, ok)
	}
}

// TestOpenAIChatGPT_NoFedRAMPHeader: without the flag, no header.
func TestOpenAIChatGPT_NoFedRAMPHeader(t *testing.T) {
	session := chatGPTSession()
	session.AccountID = "acct"
	if v, ok := fedrampHeader(t, session); ok {
		t.Fatalf("X-OpenAI-Fedramp = %q, want absent", v)
	}
}

// TestVendorCLISession_OpenAIIsChatGPT: a Codex CLI login puts OpenAI in
// ChatGPT mode, with the file's account id.
func TestVendorCLISession_OpenAIIsChatGPT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	access := testJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()})
	content := `{"tokens":{"access_token":"` + access + `","refresh_token":"rt-cli","account_id":"acct_cli"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, &llmprovider.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: path}, "gpt-6-astra",
		llmprovider.WithHTTPClient(client))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body["stream"] != true || c.header.Get(llmprovider.ChatGPTAccountHeader) != "acct_cli" {
		t.Fatalf("stream = %v, account = %q; want ChatGPT mode for acct_cli", c.body["stream"], c.header.Get(llmprovider.ChatGPTAccountHeader))
	}
}

// TestWithStore_ChatGPTIgnoresTrue: a ChatGPT session always sends store
// false.
func TestWithStore_ChatGPTIgnoresTrue(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client), WithStore(true))
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hi")); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body["store"] != false {
		t.Errorf("store = %v, want false", c.body["store"])
	}
}
