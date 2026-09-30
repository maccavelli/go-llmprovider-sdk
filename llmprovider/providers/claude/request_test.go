package claude

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// New in the new API: credentials, and the Request fields the old methods
// could not express.

// TestNew_RefusesAnOAuthSession carries llmprovider's old
// TestNewProviderWithSource_RejectsClaude: Claude takes no session. A
// source of a kind the service does not accept is refused by New (R16).
func TestNew_RefusesAnOAuthSession(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"oauth": &llmprovider.OAuthSession{Issuer: llmprovider.DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)},
		"cli":   &llmprovider.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "/nonexistent"},
	} {
		if _, err := New(llmprovider.WithTokenSource(src)); !errors.Is(err, llmprovider.ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
	if _, err := New(); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("no credential: err = %v, want ErrInvalidRequest", err)
	}
	if _, err := New(llmprovider.WithAPIKey("k"), llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", "o")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a Kilo option: err = %v, want ErrInvalidRequest", err)
	}
}

// keySource hands out a new key per call, as a CommandToken may.
type keySource struct{ n int }

func (s *keySource) Token(context.Context) (llmprovider.Token, error) {
	s.n++
	return llmprovider.Token{Value: "key-" + string(rune('0'+s.n)), Type: llmprovider.TokenAPIKey}, nil
}

// TestGenerate_ReadsTheKeyPerRequest: the key comes from the source on each
// request, so a rotating source is honoured.
func TestGenerate_ReadsTheKeyPerRequest(t *testing.T) {
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("x-api-key"))
		_, _ = w.Write([]byte(okReply))
	}))
	t.Cleanup(srv.Close)
	p := build(t, llmprovider.WithTokenSource(&keySource{}), llmprovider.WithBaseURL(srv.URL))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatal(err)
		}
	}
	if len(keys) != 2 || keys[0] != "key-1" || keys[1] != "key-2" {
		t.Fatalf("x-api-key sent %v, want [key-1 key-2]", keys)
	}
}

func TestGenerate_RequestFields(t *testing.T) {
	weather := llmprovider.Tool{Name: "get_weather", Schema: map[string]any{"type": "object"}}
	for _, c := range []struct {
		name  string
		req   func() *llmprovider.Request
		opts  []llmprovider.Option
		check func(t *testing.T, body map[string]any)
	}{
		{"model override, and instructions before system items", func() *llmprovider.Request {
			r := text("hi")
			r.Model, r.Instructions = "claude-override", "Be brief."
			r.Input = append([]llmprovider.Item{llmprovider.MessageItem{Role: "system", Text: "Answer in French."}}, r.Input...)
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if body["model"] != "claude-override" || body["system"] != "Be brief.\n\nAnswer in French." {
				t.Errorf("model = %v, system = %q", body["model"], body["system"])
			}
		}},
		{"MaxOutputTokens overrides WithMaxTokens", func() *llmprovider.Request {
			r := text("hi")
			r.MaxOutputTokens = 77
			return r
		}, []llmprovider.Option{llmprovider.WithMaxTokens(500)}, func(t *testing.T, body map[string]any) {
			if body["max_tokens"] != float64(77) {
				t.Errorf("max_tokens = %v, want 77", body["max_tokens"])
			}
		}},
		{"auto sends no tool_choice", func() *llmprovider.Request {
			r := text("hi")
			r.Tools = []llmprovider.Tool{weather}
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if _, ok := body["tool_choice"]; ok {
				t.Errorf("tool_choice = %v, want absent", body["tool_choice"])
			}
		}},
		{"required is any", func() *llmprovider.Request {
			r := text("hi")
			r.Tools, r.ToolChoice = []llmprovider.Tool{weather}, llmprovider.ToolChoiceRequired
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if tc, _ := body["tool_choice"].(map[string]any); tc["type"] != "any" {
				t.Errorf("tool_choice = %v, want any", body["tool_choice"])
			}
		}},
		{"none", func() *llmprovider.Request {
			r := text("hi")
			r.Tools, r.ToolChoice = []llmprovider.Tool{weather}, llmprovider.ToolChoiceNone
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if tc, _ := body["tool_choice"].(map[string]any); tc["type"] != "none" {
				t.Errorf("tool_choice = %v, want none", body["tool_choice"])
			}
		}},
		{"required while thinking is auto", func() *llmprovider.Request {
			r := thinking(text("hi"))
			r.Tools, r.ToolChoice = []llmprovider.Tool{weather}, llmprovider.ToolChoiceRequired
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if tc, _ := body["tool_choice"].(map[string]any); tc["type"] != "auto" {
				t.Errorf("tool_choice = %v, want auto", body["tool_choice"])
			}
		}},
		{"the default reasoning applies to a request with none", func() *llmprovider.Request { return text("hi") },
			[]llmprovider.Option{llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 3000})},
			func(t *testing.T, body map[string]any) {
				if th, _ := body["thinking"].(map[string]any); th["budget_tokens"] != float64(3000) {
					t.Errorf("thinking = %v, want budget 3000", body["thinking"])
				}
			}},
		{"a request's budget wins over the default", func() *llmprovider.Request {
			r := text("hi")
			r.Reasoning = &llmprovider.Reasoning{Budget: 1500}
			return r
		}, []llmprovider.Option{llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 3000})},
			func(t *testing.T, body map[string]any) {
				if th, _ := body["thinking"].(map[string]any); th["budget_tokens"] != float64(1500) {
					t.Errorf("thinking = %v, want budget 1500", body["thinking"])
				}
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, okReply)
			p := build(t, apiKey(srv.URL, "claude-haiku-4-5", c.opts...)...)
			if _, err := p.Generate(context.Background(), c.req()); err != nil {
				t.Fatal(err)
			}
			c.check(t, body)
		})
	}
}

// TestGenerate_ContinuationRefusedBeforeTheNetwork: the Messages API is
// stateless, so a PreviousResponseID is refused with no request sent.
func TestGenerate_ContinuationRefusedBeforeTheNetwork(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okReply)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	req := text("hi")
	req.PreviousResponseID = "msg_1"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	if body != nil {
		t.Fatalf("a request was sent: %v", body)
	}
}
