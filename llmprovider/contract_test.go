package llmprovider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// stubProvider is a Provider with no network, for the contract's own tests.
type stubProvider struct {
	id   ProviderID
	caps Capabilities
	gen  func(context.Context, *Request) (*Response, error)
}

func (s *stubProvider) ID() ProviderID { return s.id }

func (s *stubProvider) Capabilities() Capabilities { return s.caps }

func (s *stubProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	return s.gen(ctx, req)
}

func textInput(text string) []Item {
	return []Item{MessageItem{Role: RoleUser, Text: text}}
}

func TestCapabilitiesCheck_RefusesUnsupportedNeeds(t *testing.T) {
	tools := []Tool{{Name: "t"}}
	for _, c := range []struct {
		name string
		caps Capabilities
		req  *Request
	}{
		{"tools", Capabilities{}, &Request{Tools: tools}},
		{"a forced tool", Capabilities{Tools: Supported}, &Request{Tools: tools, ToolChoice: ForceTool("t")}},
		{"a required tool", Capabilities{Tools: Supported}, &Request{Tools: tools, ToolChoice: ToolChoiceRequired}},
		{"reasoning", Capabilities{}, &Request{Reasoning: &Reasoning{}}},
		{"continuation", Capabilities{}, &Request{PreviousResponseID: "r1"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.caps.Check(c.req)
			if !errors.Is(err, ErrUnsupported) || !errors.Is(err, errors.ErrUnsupported) {
				t.Fatalf("Check = %v, want an error matching ErrUnsupported and errors.ErrUnsupported", err)
			}
		})
	}
}

func TestCapabilitiesCheck_AllowsBestEffortAndSupported(t *testing.T) {
	req := &Request{
		Input:              textInput("hi"),
		Tools:              []Tool{{Name: "t"}},
		ToolChoice:         ForceTool("t"),
		Reasoning:          &Reasoning{Effort: EffortMedium, Budget: 1024},
		PreviousResponseID: "r1",
	}
	caps := Capabilities{Tools: Supported, ForcedToolChoice: BestEffort, Reasoning: BestEffort, Continuation: Supported}
	if err := caps.Check(req); err != nil {
		t.Fatalf("Check = %v, want nil", err)
	}
	if err := (Capabilities{}).Check(&Request{Input: textInput("hi"), ToolChoice: ToolChoiceNone}); err != nil {
		t.Fatalf("a plain request on a provider with no capabilities: Check = %v, want nil", err)
	}
}

func TestCapabilitiesCheck_RefusesInvalidValues(t *testing.T) {
	all := Capabilities{Tools: Supported, ForcedToolChoice: Supported, Reasoning: Supported, Continuation: Supported}
	for _, c := range []struct {
		name string
		req  *Request
	}{
		{"a nil request", nil},
		{"a negative output limit", &Request{MaxOutputTokens: -1}},
		{"a tool with no name", &Request{Tools: []Tool{{}}}},
		{"an unknown tool choice", &Request{ToolChoice: "sometimes"}},
		{"a required tool with no tools", &Request{ToolChoice: ToolChoiceRequired}},
		{"a forced tool that is not offered", &Request{Tools: []Tool{{Name: "a"}}, ToolChoice: ForceTool("b")}},
		{"an unknown effort", &Request{Reasoning: &Reasoning{Effort: "extreme"}}},
		{"a negative budget", &Request{Reasoning: &Reasoning{Budget: -1}}},
		{"an unknown role", &Request{Input: []Item{MessageItem{Role: RoleUser, Text: "hi"}, MessageItem{Role: "model", Text: "x"}}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := all.Check(c.req); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("Check = %v, want an error matching ErrInvalidRequest", err)
			}
		})
	}
}

// TestCapabilitiesCheck_AcceptsTheRoles: the three roles pass, and so does an
// empty one, which each wire sends as the user's (0015-MADR D6).
func TestCapabilitiesCheck_AcceptsTheRoles(t *testing.T) {
	for _, role := range []Role{"", RoleUser, RoleAssistant, RoleSystem} {
		if err := (Capabilities{}).Check(&Request{Input: []Item{MessageItem{Role: role, Text: "hi"}}}); err != nil {
			t.Errorf("role %q: Check = %v, want nil", role, err)
		}
	}
	err := (Capabilities{}).Check(&Request{Input: []Item{MessageItem{Text: "a"}, MessageItem{Role: "developer", Text: "b"}}})
	if err == nil || !strings.Contains(err.Error(), `input 1 has unknown role "developer"`) {
		t.Errorf("Check = %v, want the item and its role named", err)
	}
}

func TestToolChoice_ForceTool(t *testing.T) {
	if name, ok := ForceTool("lookup").Tool(); !ok || name != "lookup" {
		t.Fatalf("ForceTool(lookup).Tool() = %q, %v", name, ok)
	}
	for _, c := range []ToolChoice{"", ToolChoiceAuto, ToolChoiceNone, ToolChoiceRequired} {
		if _, ok := c.Tool(); ok {
			t.Errorf("%q.Tool() reports a named tool", c)
		}
	}
}

func TestSupport_String(t *testing.T) {
	for s, want := range map[Support]string{Unsupported: "Unsupported", BestEffort: "BestEffort", Supported: "Supported", Support(9): "Support(invalid)"} {
		if got := s.String(); got != want {
			t.Errorf("Support(%d).String() = %q, want %q", int(s), got, want)
		}
	}
}

func TestGenerateText_ReturnsTheOutputText(t *testing.T) {
	p := &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) {
		return &Response{Output: []Item{ReasoningItem{Text: "hmm"}, MessageItem{Text: "a"}, MessageItem{Text: "b"}}}, nil
	}}
	if got, err := GenerateText(context.Background(), p, &Request{}); err != nil || got != "ab" {
		t.Fatalf("GenerateText = %q, %v; want ab", got, err)
	}
	boom := errors.New("boom")
	p.gen = func(context.Context, *Request) (*Response, error) { return nil, boom }
	if _, err := GenerateText(context.Background(), p, &Request{}); !errors.Is(err, boom) {
		t.Fatalf("GenerateText err = %v, want boom", err)
	}
	p.gen = func(context.Context, *Request) (*Response, error) { return nil, nil }
	if _, err := GenerateText(context.Background(), p, &Request{}); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("a nil response: err = %v, want ErrIncomplete", err)
	}
}

func TestGenerateToolCall_ForcesTheOneTool(t *testing.T) {
	var sent *Request
	p := &stubProvider{id: "stub", gen: func(_ context.Context, req *Request) (*Response, error) {
		sent = req
		return &Response{Output: []Item{
			FunctionCallItem{Name: "other", Arguments: "{}"},
			FunctionCallItem{CallID: "c1", Name: "lookup", Arguments: `{"q":1}`},
		}}, nil
	}}
	req := &Request{Input: textInput("hi"), Tools: []Tool{{Name: "lookup"}}}
	call, err := GenerateToolCall(context.Background(), p, req)
	if err != nil || call.CallID != "c1" {
		t.Fatalf("GenerateToolCall = %+v, %v; want call c1", call, err)
	}
	if sent.ToolChoice != ForceTool("lookup") {
		t.Fatalf("sent ToolChoice %q, want ForceTool(lookup)", sent.ToolChoice)
	}
	if req.ToolChoice != "" {
		t.Fatalf("the caller's request was changed: ToolChoice %q", req.ToolChoice)
	}
}

func TestGenerateToolCall_Refusals(t *testing.T) {
	noCall := &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) {
		return &Response{Output: []Item{MessageItem{Text: "no"}}}, nil
	}}
	one := []Tool{{Name: "t"}}
	for _, c := range []struct {
		name string
		p    Provider
		req  *Request
		want error
	}{
		{"a nil request", noCall, nil, ErrInvalidRequest},
		{"no tools", noCall, &Request{}, ErrInvalidRequest},
		{"two tools", noCall, &Request{Tools: []Tool{{Name: "a"}, {Name: "b"}}}, ErrInvalidRequest},
		{"no call in the response", noCall, &Request{Tools: one}, ErrIncomplete},
		{"a failed generation", &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) {
			return nil, ErrAuthFailure
		}}, &Request{Tools: one}, ErrAuthFailure},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := GenerateToolCall(context.Background(), c.p, c.req); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}
