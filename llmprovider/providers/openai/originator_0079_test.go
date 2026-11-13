package openai

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
)

// originatorBackend records the headers of each request a ChatGPT session
// sends: the Codex listing (GET) and the generation (POST).
type originatorBackend struct {
	mu   sync.Mutex
	gets []http.Header
	post []http.Header
}

func (b *originatorBackend) client(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		b.mu.Lock()
		defer b.mu.Unlock()
		if r.Method == http.MethodGet {
			b.gets = append(b.gets, r.Header.Clone())
			return httpResponse(r, http.StatusOK, `{"models":[{"slug":"gpt-5.4-mini","visibility":"list","priority":1}]}`), nil
		}
		b.post = append(b.post, r.Header.Clone())
		return httpResponse(r, http.StatusOK, okResponse), nil
	})}
}

// exercise sends one generation and one listing with a ChatGPT session.
func (b *originatorBackend) exercise(t *testing.T, opts ...llmprovider.Option) {
	t.Helper()
	p := sessionProvider(t, chatGPTSession(), "gpt-5.4-mini",
		append([]llmprovider.Option{llmprovider.WithHTTPClient(b.client(t))}, opts...)...)
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(b.gets) != 1 || len(b.post) != 1 {
		t.Fatalf("requests: %d listings, %d generations; want one each", len(b.gets), len(b.post))
	}
}

// TestChatGPTOriginator_ClientInfo_0079: WithClientInfo's name is the
// originator of a generation and of the listing, and leads their User-Agent
// (0079-MADR D1).
func TestChatGPTOriginator_ClientInfo_0079(t *testing.T) {
	b := &originatorBackend{}
	b.exercise(t, llmprovider.WithClientInfo("probe-app", "1.2.3"))
	for what, h := range map[string]http.Header{"listing": b.gets[0], "generation": b.post[0]} {
		if got := h.Get(headerOriginator); got != "probe-app" {
			t.Errorf("%s originator = %q; want probe-app", what, got)
		}
		if got := h.Get("User-Agent"); !strings.HasPrefix(got, "probe-app/1.2.3 (") {
			t.Errorf("%s User-Agent = %q; want it to begin probe-app/1.2.3 (", what, got)
		}
	}
}

// TestChatGPTOriginator_Default_0079: with no name, or an empty one, the
// originator is this module's and the User-Agent is what it was.
func TestChatGPTOriginator_Default_0079(t *testing.T) {
	checkEmptyNameDefaults(t)
	for name, opts := range map[string][]llmprovider.Option{
		"no option":   nil,
		"empty name":  {llmprovider.WithClientInfo("", "1.2.3")},
		"empty, bare": {llmprovider.WithClientInfo("", "")},
	} {
		b := &originatorBackend{}
		b.exercise(t, opts...)
		for what, h := range map[string]http.Header{"listing": b.gets[0], "generation": b.post[0]} {
			if got := h.Get(headerOriginator); got != originatorValue {
				t.Errorf("%s, %s: originator = %q; want %q", name, what, got, originatorValue)
			}
			if got := h.Get("User-Agent"); !strings.HasPrefix(got, "go-llmprovider-sdk/") ||
				!strings.Contains(got, ") go-llmprovider-sdk/") {
				t.Errorf("%s, %s: User-Agent = %q; want this module's, as before", name, what, got)
			}
		}
	}
}

// checkEmptyNameDefaults (part of TestChatGPTOriginator_Default_0079): the shared helpers default an empty
// name themselves, whatever calls them (0079-MADR D1).
func checkEmptyNameDefaults(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://example.test/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	chatgpt.SetHeadersAs(req, nil, "")
	if got := req.Header.Get(headerOriginator); got != originatorValue {
		t.Errorf("SetHeadersAs(\"\") originator = %q; want %q", got, originatorValue)
	}
	req, err = chatgpt.ModelsRequestAs(context.Background(), "http://example.test/codex", "agent/1", "",
		llmprovider.Token{Value: "t"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get(headerOriginator); got != originatorValue {
		t.Errorf("ModelsRequestAs(\"\") originator = %q; want %q", got, originatorValue)
	}
}

// TestPlanSendsNoOriginator_0079: a plan session's generation and listing
// send no originator, whatever WithClientInfo says (0043-MADR D5).
func TestPlanSendsNoOriginator_0079(t *testing.T) {
	s := &planServer{listing: `{"models":[{"slug":"gpt-6.1-sol","visibility":"list"}]}`, stream: sseEvents(sseCreated)}
	p := sessionProvider(t, planSession(), "gpt-6.1-sol", llmprovider.WithHTTPClient(s.client()),
		llmprovider.WithClientInfo("probe-app", "1.2.3"), llmprovider.WithModelProbes(false))
	if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	_, _ = llmprovider.GenerateText(context.Background(), p, text("hi")) // the stream may fail; its request was sent
	sent := s.sent()
	if len(sent) != 2 {
		t.Fatalf("requests = %d; want the listing and the generation", len(sent))
	}
	for _, r := range sent {
		if got := r.Header.Get(headerOriginator); got != "" {
			t.Errorf("%s %s sent originator %q; want none", r.Method, r.URL.Path, got)
		}
	}
}
