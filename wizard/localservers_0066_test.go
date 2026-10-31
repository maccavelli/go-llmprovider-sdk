package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// 0066-PLAN Phase 3 (V5; Q2 (a)): under Options.LocalServers the provider
// menu offers each Chat Completions server chatcompat.Discover finds.

// portReply0066 answers one request to a port: a status and a body.
type portReply0066 func(r *http.Request) (int, string)

// ports0066 is a transport that answers http://127.0.0.1:<port> by port, so
// no port of the host is used; a port it does not hold refuses the
// connection. It records each request as "port path auth".
type ports0066 struct {
	mu    sync.Mutex
	by    map[string]portReply0066
	seen  []string
	onReq func(*http.Request)
}

func (p *ports0066) RoundTrip(r *http.Request) (*http.Response, error) {
	p.mu.Lock()
	p.seen = append(p.seen, r.URL.Port()+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
	reply, ok := p.by[r.URL.Port()]
	on := p.onReq
	p.mu.Unlock()
	if on != nil {
		on(r)
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if r.URL.Hostname() != "127.0.0.1" || !ok {
		return nil, errors.New("connection refused")
	}
	status, body := reply(r)
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func (p *ports0066) requests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.seen)
}

// listing0066 lists ids at /v1/models; with a key, it answers 401 to a
// request without "Bearer <key>".
func listing0066(key string, ids ...string) portReply0066 {
	return func(r *http.Request) (int, string) {
		if r.URL.Path != "/v1/models" {
			return http.StatusNotFound, `{}`
		}
		if key != "" && r.Header.Get("Authorization") != "Bearer "+key {
			return http.StatusUnauthorized, `{"error":{"message":"no key"}}`
		}
		data := make([]string, len(ids))
		for i, id := range ids {
			data[i] = `{"id":"` + id + `","object":"model"}`
		}
		return http.StatusOK, `{"object":"list","data":[` + strings.Join(data, ",") + `]}`
	}
}

// localOptions0066 is a LocalServers run over t's transport, off models.dev,
// in t's own cache.
func localOptions0066(t *testing.T, p *ports0066) Options {
	return Options{LocalServers: true, Discover: true, HTTPClient: &http.Client{Transport: p},
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}
}

// menuLabels0066 is the first menu's labels.
func menuLabels0066(f *fakePrompter) []string {
	if len(f.seenSelectItems) == 0 {
		return nil
	}
	var out []string
	for _, c := range f.seenSelectItems[0] {
		out = append(out, c.Label)
	}
	return out
}

// TestLocalServers_Offered_0066: a server on 1234 is a menu row after the
// registry's; picked, its endpoint default is its BaseURL, and the Result
// names local-1234 and that BaseURL, which the documented chatcompat.New call
// builds.
func TestLocalServers_Offered_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("", "model-a", "model-b")}}
	builtins := len(providers.Default().Descriptors())
	f := newFake(t, fakePrompter{selects: []int{builtins, 0}, inputs: []string{acceptDefault}, blankSearches: true})
	res, err := ConfigureLLM(context.Background(), f, localOptions0066(t, p))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seenSelectItems) == 0 || len(f.seenSelectItems[0]) != builtins+1 {
		t.Fatalf("menu %q; want the %d built-ins, then the server", menuLabels0066(f), builtins)
	}
	row := f.seenSelectItems[0][builtins]
	if row.Label != "Chat Completions server (127.0.0.1:1234)" || row.Detail != "running on this machine, 2 models" {
		t.Errorf("row %q / %q; want the server, running on this machine, 2 models", row.Label, row.Detail)
	}
	const base = "http://127.0.0.1:1234/v1"
	if res.Provider != "local-1234" || res.BaseURL != base || res.Model != "model-a" || res.Kind != CredNone {
		t.Errorf("Result %s %q %q %q; want local-1234, %q, model-a, no credential", res.Provider, res.BaseURL, res.Model, res.Kind, base)
	}
	built, err := chatcompat.New(chatcompat.WithID(res.Provider), llmprovider.WithBaseURL(res.BaseURL))
	if err != nil || built.ID() != res.Provider {
		t.Errorf("chatcompat.New from the Result: %v", err)
	}
}

// TestLocalServers_OllamaPortSkipped_0066: a server on the port of a local
// provider the Registry holds, Ollama's 11434, is that provider, offered
// already.
func TestLocalServers_OllamaPortSkipped_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{
		"11434": listing0066("", "llama3.2:latest"),
		"1234":  listing0066("", "model-a"),
	}}
	// The run ends at the menu, which is what is checked.
	f := newFake(t, fakePrompter{})
	_, _ = ConfigureLLM(context.Background(), f, localOptions0066(t, p))
	labels := menuLabels0066(f)
	if slices.ContainsFunc(labels, func(l string) bool { return strings.Contains(l, ":11434") }) ||
		!slices.Contains(labels, "Chat Completions server (127.0.0.1:1234)") {
		t.Errorf("menu %q; want the server on 1234 and none on 11434", labels)
	}
}

// TestLocalServers_NeedsKeyAsksKey_0066: a server that wants a key says so
// in its row, and the key is asked when its listing refuses the request
// without one.
func TestLocalServers_NeedsKeyAsksKey_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("lm-key", "model-a")}}
	builtins := len(providers.Default().Descriptors())
	f := newFake(t, fakePrompter{selects: []int{builtins, 0}, inputs: []string{acceptDefault}, secrets: []string{"lm-key"},
		blankSearches: true})
	res, err := ConfigureLLM(context.Background(), f, localOptions0066(t, p))
	if err != nil {
		t.Fatal(err)
	}
	if row := f.seenSelectItems[0][builtins]; row.Detail != "running on this machine, wants a key" {
		t.Errorf("row detail %q; want \"running on this machine, wants a key\"", row.Detail)
	}
	if res.Provider != "local-1234" || res.Kind != CredAPIKey || res.APIKey != "lm-key" || res.Model != "model-a" {
		t.Errorf("Result %s %q key set %v %q; want local-1234, api_key, true, model-a", res.Provider, res.Kind,
			res.APIKey == "lm-key", res.Model)
	}
}

// TestLocalServers_OffAsksNothing_0066: without the option no port is
// asked, and the menu is the registry's.
func TestLocalServers_OffAsksNothing_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("", "model-a")}}
	o := localOptions0066(t, p)
	o.LocalServers = false
	f := newFake(t, fakePrompter{})
	_, _ = ConfigureLLM(context.Background(), f, o)
	if got := p.requests(); len(got) != 0 {
		t.Errorf("requests %q; want none", got)
	}
	if n, want := len(menuLabels0066(f)), len(providers.Default().Descriptors()); n != want {
		t.Errorf("menu of %d; want the registry's %d", n, want)
	}
}

// TestLocalServers_ResolveByID_0066: ResolveCredential with a found
// server's id, which the Registry lacks, asks no menu and lists it.
func TestLocalServers_ResolveByID_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("", "model-a", "model-b")}}
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}})
	res, cat, err := ResolveCredential(context.Background(), f, localOptions0066(t, p), "local-1234")
	if err != nil {
		t.Fatal(err)
	}
	if res.Provider != "local-1234" || res.BaseURL != "http://127.0.0.1:1234/v1" || !cat.Live || len(cat.Usable) != 2 {
		t.Errorf("Result %s %q, Live %v, Usable %q; want local-1234, its BaseURL, live, 2", res.Provider, res.BaseURL,
			cat.Live, cat.Usable)
	}
	if len(f.seenSelect) != 0 {
		t.Errorf("selects %q; want none", f.seenSelect)
	}
}

// TestLocalServers_NotRunningIsInvalidProvider_0066: an id the Registry
// lacks and no server found under it is ErrInvalidProvider, as without the
// option.
func TestLocalServers_NotRunningIsInvalidProvider_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{}}
	f := newFake(t, fakePrompter{})
	_, _, err := ResolveCredential(context.Background(), f, localOptions0066(t, p), "local-1234")
	if !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("err = %v; want ErrInvalidProvider", err)
	}
	if len(p.requests()) == 0 {
		t.Error("no port was asked; want the servers looked for")
	}
}

// TestLocalServers_EndedContext_0066: ctx ending while the servers are
// looked for ends the run with ctx's error, before the menu.
func TestLocalServers_EndedContext_0066(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("", "model-a")}, onReq: func(*http.Request) { cancel() }}
	f := newFake(t, fakePrompter{})
	_, err := ConfigureLLM(ctx, f, localOptions0066(t, p))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want context.Canceled", err)
	}
	if len(f.seenSelect) != 0 {
		t.Errorf("selects %q; want none", f.seenSelect)
	}
}

// TestLocalServers_ProvidersAddsFound_0066: Options.Providers restricts the
// registry's rows, and the found servers are added to it for the run; the
// caller's slice is not written.
func TestLocalServers_ProvidersAddsFound_0066(t *testing.T) {
	p := &ports0066{by: map[string]portReply0066{"1234": listing0066("", "model-a")}}
	o := localOptions0066(t, p)
	o.Providers = make([]llmprovider.ProviderID, 1, 4)
	o.Providers[0] = llmprovider.ProviderClaude
	f := newFake(t, fakePrompter{})
	_, _ = ConfigureLLM(context.Background(), f, o)
	if got, want := menuLabels0066(f), []string{"Claude (Anthropic)", "Chat Completions server (127.0.0.1:1234)"}; !slices.Equal(got, want) {
		t.Errorf("menu %q; want %q", got, want)
	}
	if spare := o.Providers[:cap(o.Providers)]; len(o.Providers) != 1 || spare[1] != "" {
		t.Errorf("the caller's Providers became %q, its spare capacity %q", o.Providers, spare)
	}
}
