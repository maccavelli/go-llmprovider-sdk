package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// planSession is a ChatGPT plan session, as auth.LoginChatGPTPlan returns
// one; its account id must never reach a request (0043-MADR A3).
func planSession() *auth.OAuthSession {
	return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Access: "plan-access", Refresh: "plan-refresh", Expiry: time.Now().Add(time.Hour), ClientID: "oaiapp_test",
		AccountID: "acct-plan", Resource: "https://api.openai.com/v1", TokenURL: auth.DefaultOpenAIIssuer + "/api/accounts/oauth/token"}
}

// planServer records every request and answers by path: the listing, or
// the event stream.
type planServer struct {
	mu       sync.Mutex
	requests []*http.Request
	listing  string
	stream   string
}

func (s *planServer) client() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		s.mu.Lock()
		s.requests = append(s.requests, r)
		s.mu.Unlock()
		body := s.stream
		if r.Method == http.MethodGet {
			body = s.listing
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
}

func (s *planServer) sent() []*http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*http.Request(nil), s.requests...)
}

// TestPlan_Request_0043: a plan session's request goes to the platform,
// streamed, with store false, no output limit, the cache key only when
// asked, and none of the Codex backend's headers; Generate reads the stream
// (0043-MADR D5).
func TestPlan_Request_0043(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	var url string
	inner := client.Transport
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		url = r.URL.String()
		return inner.RoundTrip(r)
	})
	p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(client),
		llmprovider.WithMaxTokens(2048), WithStore(true))
	req := text("hi")
	req.MaxOutputTokens = 512
	got, err := llmprovider.GenerateText(context.Background(), p, req)
	if err != nil || got != "ok" {
		t.Fatalf("GenerateText = %q, %v; want the stream's text", got, err)
	}
	if url != PlatformBaseURL+"/responses" {
		t.Errorf("URL %s; want the platform's", url)
	}
	if c.body["stream"] != true || c.body["store"] != false {
		t.Errorf("stream %v, store %v; want true, false", c.body["stream"], c.body["store"])
	}
	for _, absent := range []string{"max_output_tokens", "prompt_cache_key", "previous_response_id"} {
		if _, ok := c.body[absent]; ok {
			t.Errorf("the body sends %s: %v", absent, c.body[absent])
		}
	}
	if include, _ := c.body["include"].([]any); len(include) != 1 || include[0] != includeEncryptedReasoning {
		t.Errorf("include %v; want the encrypted reasoning", c.body["include"])
	}
	if c.header.Get("Authorization") != "Bearer plan-access" || c.header.Get("Accept") != "text/event-stream" {
		t.Errorf("Authorization %q, Accept %q", c.header.Get("Authorization"), c.header.Get("Accept"))
	}
	for _, h := range []string{headerChatGPTAccount, headerOriginator, headerFedRAMP, headerSession, headerResidency} {
		if v := c.header.Get(h); v != "" {
			t.Errorf("header %s = %q; a plan session sends none of the Codex backend's", h, v)
		}
	}

	req = text("hi")
	req.Cache = llmprovider.CacheShort
	if _, err := llmprovider.GenerateText(context.Background(), p, req); err != nil {
		t.Fatal(err)
	}
	if c.body["prompt_cache_key"] == nil {
		t.Error("a cached request sends no prompt_cache_key")
	}
}

// TestPlan_Refusals_0043: a continuation and a system message in the input
// are refused before the network; instructions are sent (0043-MADR D5, A4).
func TestPlan_Refusals_0043(t *testing.T) {
	s := &planServer{stream: fixture(t, "chatgpt-text.sse")}
	p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(s.client()))
	if p.Capabilities().Continuation != llmprovider.Unsupported {
		t.Errorf("Continuation %v; want Unsupported", p.Capabilities().Continuation)
	}
	continued := text("hi")
	continued.PreviousResponseID = "resp_1"
	system := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "be brief"},
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	for name, req := range map[string]*llmprovider.Request{"a continuation": continued, "a system message": system} {
		if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) {
			t.Errorf("%s: Generate err = %v; want ErrUnsupported", name, err)
		}
		for _, err := range llmprovider.Stream(context.Background(), p, req) {
			if !errors.Is(err, llmprovider.ErrUnsupported) {
				t.Errorf("%s: Stream err = %v; want ErrUnsupported", name, err)
			}
		}
	}
	if _, err := p.Generate(context.Background(), system); err == nil || !strings.Contains(err.Error(), "Request.Instructions") {
		t.Errorf("err = %v; want it to name Request.Instructions", err)
	}
	if n := len(s.sent()); n != 0 {
		t.Errorf("%d requests sent; want none", n)
	}
	instructed := text("hi")
	instructed.Instructions = "be brief"
	if _, err := p.Generate(context.Background(), instructed); err != nil {
		t.Errorf("instructions: %v", err)
	}
}

// TestPlan_SystemMessageElsewhere_0043: a key and a ChatGPT session send a
// system message as before.
func TestPlan_SystemMessageElsewhere_0043(t *testing.T) {
	system := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "be brief"},
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), system); err != nil || c.calls != 1 {
		t.Errorf("a ChatGPT session: err %v, %d calls; want it sent", err, c.calls)
	}
}

// TestPlan_DeferredToolsAppended_0043: a plan session sends no tool_search;
// a loaded tool is appended, with WarningDeferralEmulated (0043-MADR D5).
func TestPlan_DeferredToolsAppended_0043(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
	schema := map[string]any{"type": "object"}
	req := &llmprovider.Request{
		Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "stock?"},
			llmprovider.FunctionCallItem{CallID: "s1", Name: "tool_search", Arguments: `{"query":"stock"}`},
			llmprovider.FunctionCallOutputItem{CallID: "s1", Output: "Loaded lookup_stock."},
			llmprovider.ToolLoadItem{CallID: "s1", Name: "lookup_stock"},
		},
		Tools: []llmprovider.Tool{
			{Name: "tool_search", Schema: schema, Search: true},
			{Name: "lookup_stock", Schema: schema, Deferred: true},
			{Name: "send_email", Schema: schema, Deferred: true},
		},
	}
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	tools, _ := c.body["tools"].([]any)
	for _, tool := range tools {
		m, _ := tool.(map[string]any)
		if m["type"] == "tool_search" {
			t.Errorf("a tool_search entry was sent: %v", m)
		}
		name, _ := m["name"].(string)
		names = append(names, name)
	}
	if strings.Join(names, ",") != "tool_search,lookup_stock" {
		t.Errorf("tools %q; want the search tool, then the loaded tool only", names)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0].Kind != llmprovider.WarningDeferralEmulated {
		t.Errorf("warnings %+v; want WarningDeferralEmulated", resp.Warnings)
	}
}

// TestPlan_Listing_0043: the account's "models" array, visibility "list"
// only, in order, once each; never a probe; a key's "data" body or an empty
// catalog is an error (0043-MADR D5).
func TestPlan_Listing_0043(t *testing.T) {
	s := &planServer{listing: `{"models":[{"slug":"gpt-6.1-sol","visibility":"list"},{"slug":"gpt-6-hidden","visibility":"hide"},` +
		`{"slug":" gpt-6-luna ","visibility":"LIST"},{"slug":"gpt-6.1-sol","visibility":"list"},{"slug":"","visibility":"list"},` +
		`{"slug":"gpt-6-none"}]}`}
	p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(s.client()), llmprovider.WithModelProbes(true))
	got, err := p.(llmprovider.ModelLister).ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "gpt-6.1-sol,gpt-6-luna" {
		t.Errorf("ListModels = %q; want gpt-6.1-sol, gpt-6-luna", got)
	}
	sent := s.sent()
	if len(sent) != 1 || sent[0].Method != http.MethodGet || sent[0].URL.String() != PlatformBaseURL+"/models" ||
		sent[0].Header.Get(headerOriginator) != "" || sent[0].Header.Get(headerChatGPTAccount) != "" {
		t.Errorf("requests %v; want one GET of the platform's /models, no probe and no Codex headers", sent)
	}

	for body, want := range map[string]string{
		`{"object":"list","data":[{"id":"gpt-5.5"}]}`:        `no "models" array`,
		`{"models":[{"slug":"x","visibility":"hide"}]}`:      "listed no models",
		`{"models":[{"slug":"x","visibility":"list"}` + "\n": "decode plan catalog",
	} {
		s := &planServer{listing: body}
		p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(s.client()))
		if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); !errors.Is(err, llmprovider.ErrIncomplete) ||
			!strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v; want ErrIncomplete naming %q", body, err, want)
		}
	}
}

// TestPlan_StreamFailures_0043: each subscription_sharing_* code inside a
// plan session's stream has D6's kind, and is not retried (A4).
func TestPlan_StreamFailures_0043(t *testing.T) {
	for code, want := range map[string]error{
		"subscription_sharing_usage_limit_exceeded":   llmprovider.ErrQuotaExhausted,
		"subscription_sharing_user_not_eligible":      llmprovider.ErrNotPermitted,
		"subscription_sharing_route_not_supported":    llmprovider.ErrNotPermitted,
		"subscription_sharing_unsupported_capability": llmprovider.ErrInvalidRequest,
	} {
		client, _ := chatGPTClient(t, sseEvents(sseCreated, sseFailed(code)))
		p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
		_, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
		if !errors.Is(err, want) {
			t.Errorf("%s: err = %v; want %v", code, err, want)
		}
		if apiErr, ok := errors.AsType[*llmprovider.APIError](err); !ok || apiErr.Retryable() {
			t.Errorf("%s: %#v; want a terminal *APIError", code, err)
		}
	}
}

// TestChatGPTUsageURL_0043 pins the "Manage usage" page (0043-MADR D6).
func TestChatGPTUsageURL_0043(t *testing.T) {
	if ChatGPTUsageURL != "https://chatgpt.com/settings/usage" {
		t.Errorf("ChatGPTUsageURL = %q", ChatGPTUsageURL)
	}
}
