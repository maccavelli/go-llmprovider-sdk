package providers_test

// 0085-MADR D9 (Phase 1b): a key that is empty once trimmed, given through
// WithAPIKey or NewStaticToken, is no key: every constructor meets it as ""
// and does what it does with "" today. Every request is answered in process by
// a RoundTripper; every key is made up or blank; no outside host (0065-MADR).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// blanks0085 are the eight values the wizard's tests use: empty once trimmed.
var blanks0085 = []struct{ name, value string }{
	{"empty", ""},
	{"spaces", "   "},
	{"tab", "\t"},
	{"crlf", "\r\n"},
	{"nbsp", "\u00a0"},
	{"mixed", " \t  \n"},
	{"cr", "\r"},
	{"lf", "\n"},
}

// compat0085 stands for the Chat Completions provider, built directly.
const compat0085 llmprovider.ProviderID = "lmstudio-0085"

// recorder0085 answers every request with a 401 and keeps its headers.
type recorder0085 struct {
	mu   sync.Mutex
	reqs []http.Header
}

func (r *recorder0085) client() *http.Client {
	return &http.Client{Transport: roundTrip0085(func(req *http.Request) (*http.Response, error) {
		r.mu.Lock()
		r.reqs = append(r.reqs, req.Header.Clone())
		r.mu.Unlock()
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
	})}
}

type roundTrip0085 func(*http.Request) (*http.Response, error)

func (f roundTrip0085) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (r *recorder0085) all() []http.Header {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]http.Header(nil), r.reqs...)
}

// leaks0085 names a header that carries a blank value: an empty or
// whitespace-only value, or a bare "Bearer".
func leaks0085(reqs []http.Header) []string {
	var out []string
	for _, h := range reqs {
		for name, values := range h {
			for _, v := range values {
				if t := strings.TrimSpace(v); t == "" || t == "Bearer" {
					out = append(out, fmt.Sprintf("%s=%q", name, v))
				}
			}
		}
	}
	return out
}

func model0085(id llmprovider.ProviderID) string {
	if d, ok := providers.Default().Descriptor(id); ok && len(d.StaticModels) > 0 {
		return d.StaticModels[0]
	}
	return "made-up-model"
}

// keyOption0085 gives key to a provider: through WithAPIKey, or through a
// NewStaticToken whose Header the caller set.
func keyOption0085(key, header string) llmprovider.Option {
	if header == "" {
		return llmprovider.WithAPIKey(key)
	}
	s := llmprovider.NewStaticToken(key)
	s.Header = header
	return llmprovider.WithTokenSource(s)
}

// outcome0085 builds id with key and, when it builds, sends one request. It is
// what a caller observes: the error text, or the headers of every request.
type outcome0085 struct {
	err     error
	headers []http.Header
}

func run0085(t *testing.T, id llmprovider.ProviderID, key llmprovider.Option) outcome0085 {
	t.Helper()
	rec := &recorder0085{}
	common := []llmprovider.Option{key, llmprovider.WithHTTPClient(rec.client()), llmprovider.WithModel(model0085(id)),
		extractDoc()}
	var p llmprovider.Provider
	var err error
	if id == compat0085 {
		p, err = chatcompat.New(append(common, chatcompat.WithID(compat0085), llmprovider.WithBaseURL("http://127.0.0.1:1/v1"))...)
	} else {
		p, err = providers.New(id, common...)
	}
	if err != nil {
		return outcome0085{err: err}
	}
	_, _ = llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}})
	return outcome0085{headers: rec.all()}
}

func allIDs0085() []llmprovider.ProviderID {
	var ids []llmprovider.ProviderID
	for _, d := range providers.Default().Descriptors() {
		ids = append(ids, d.ID)
	}
	return append(ids, compat0085)
}

// kind0085 is what "" gives each provider (0020-MADR F52): refused at New,
// the service's anonymous or public key, or no Authorization.
func kind0085(id llmprovider.ProviderID) string {
	switch id {
	case llmprovider.ProviderKilo, llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		return "anonymous"
	case llmprovider.ProviderOllama, compat0085:
		return "none"
	}
	return "refused"
}

// authorization0085 is, for each request, the credential it carries in the
// header its route uses: Authorization, else x-api-key (OpenCode's Anthropic
// route), else x-goog-api-key (its Google route); "" when it carries none.
func authorization0085(reqs []http.Header) []string {
	var out []string
	for _, h := range reqs {
		v := h.Get("Authorization")
		for _, name := range []string{"X-Api-Key", "X-Goog-Api-Key"} {
			if v == "" {
				v = h.Get(name)
			}
		}
		out = append(out, v)
	}
	return out
}

// TestConstructorsEmptyKeyUnchanged_0085 (ST71): "" gives each of the eleven
// constructors the outcome 0020-MADR F52 decided: six refuse it at New with
// ErrInvalidRequest, "<id> api key is required"; Kilo and the OpenCode
// gateways send their anonymous or public key, keeping a caller's Header;
// Ollama and a Chat Completions server send no Authorization.
func TestConstructorsEmptyKeyUnchanged_0085(t *testing.T) {
	for _, id := range allIDs0085() {
		t.Run(string(id), func(t *testing.T) {
			o := run0085(t, id, llmprovider.WithAPIKey(""))
			switch kind0085(id) {
			case "refused":
				if !errors.Is(o.err, llmprovider.ErrInvalidRequest) || !strings.Contains(o.err.Error(), "api key is required") {
					t.Errorf("err %v; want ErrInvalidRequest, \"api key is required\"", o.err)
				}
			case "anonymous":
				auth := authorization0085(o.headers)
				if o.err != nil || len(auth) == 0 || strings.TrimSpace(strings.TrimPrefix(auth[0], "Bearer")) == "" {
					t.Errorf("err %v, credential %q; want the anonymous or public key", o.err, auth)
				}
				custom := run0085(t, id, keyOption0085("", "X-Made-Up-Key"))
				if len(custom.headers) == 0 || strings.TrimSpace(custom.headers[0].Get("X-Made-Up-Key")) == "" {
					t.Errorf("err %v, headers %v; want the anonymous key in the caller's header", custom.err, custom.headers)
				}
			default:
				if o.err != nil || len(o.headers) == 0 || slicesAny0085(authorization0085(o.headers)) {
					t.Errorf("err %v, Authorization %q; want a request with no Authorization", o.err, authorization0085(o.headers))
				}
			}
		})
	}
}

// stable0085 is h without the headers that differ for every provider built:
// Kilo's task id and the OpenCode gateways' session id.
func stable0085(h http.Header) http.Header {
	h = h.Clone()
	h.Del("X-Opencode-Session")
	h.Del("X-Kilocode-Taskid")
	return h
}

func slicesAny0085(values []string) bool {
	for _, v := range values {
		if v != "" {
			return true
		}
	}
	return false
}

// TestConstructorsBlankKeyIsNoKey_0085 (ST72): a blank key given through
// WithAPIKey is never sent as a credential by any of the eleven constructors:
// no request carries it in any header (a provider that refuses it sends none).
func TestConstructorsBlankKeyIsNoKey_0085(t *testing.T) {
	for _, id := range allIDs0085() {
		for _, b := range blanks0085 {
			t.Run(string(id)+"/"+b.name, func(t *testing.T) {
				o := run0085(t, id, llmprovider.WithAPIKey(b.value))
				if leak := leaks0085(o.headers); len(leak) != 0 {
					t.Errorf("a request carries a blank value: %v", leak)
				}
			})
		}
	}
}

// TestBlankKeyIsEmptyKeyPerProvider_0085 (ST73a to ST73d): each blank value
// gives exactly the outcome "" gives, provider by provider: the six refuse at
// New with ErrInvalidRequest and the text of ""; Kilo and OpenCode send the
// same anonymous or public key, a caller's Header kept; Ollama and a Chat
// Completions server send no Authorization; a caller-registered constructor
// receives a *StaticToken whose Value is "".
func TestBlankKeyIsEmptyKeyPerProvider_0085(t *testing.T) {
	for _, id := range allIDs0085() {
		for _, header := range []string{"", "X-Made-Up-Key"} {
			want := run0085(t, id, keyOption0085("", header))
			for _, b := range blanks0085 {
				t.Run(fmt.Sprintf("%s/header=%q/%s", id, header, b.name), func(t *testing.T) {
					got := run0085(t, id, keyOption0085(b.value, header))
					if (got.err == nil) != (want.err == nil) || (got.err != nil && (got.err.Error() != want.err.Error() ||
						errors.Is(got.err, llmprovider.ErrInvalidRequest) != errors.Is(want.err, llmprovider.ErrInvalidRequest))) {
						t.Fatalf("err %v; want the outcome of an empty key, %v", got.err, want.err)
					}
					if len(got.headers) != len(want.headers) {
						t.Fatalf("%d request(s); want %d, as for an empty key", len(got.headers), len(want.headers))
					}
					for i := range got.headers {
						if !maps.EqualFunc(stable0085(got.headers[i]), stable0085(want.headers[i]), func(a, b []string) bool { return strings.Join(a, "\x00") == strings.Join(b, "\x00") }) {
							t.Errorf("request %d headers %v; want those of an empty key, %v", i, got.headers[i], want.headers[i])
						}
					}
				})
			}
		}
	}

	t.Run("caller-registered constructor", func(t *testing.T) {
		const id llmprovider.ProviderID = "acme-0085"
		for _, b := range blanks0085 {
			var received *llmprovider.StaticToken
			reg := llmprovider.NewRegistry()
			if err := reg.Register(llmprovider.Descriptor{ID: id, Label: "Acme", RequiresAPIKey: true},
				func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
					st, err := llmprovider.ResolveOptions(id, opts)
					if err != nil {
						return nil, err
					}
					received, _ = st.TokenSource().(*llmprovider.StaticToken)
					return nil, errors.New("not built by this test")
				}); err != nil {
				t.Fatal(err)
			}
			_, _ = reg.New(id, llmprovider.WithAPIKey(b.value))
			if received == nil || received.Value != "" {
				t.Errorf("%s: the constructor received %+v; want a *StaticToken whose Value is empty", b.name, received)
			}
		}
	})
}

// TestStaticTokenBlankIsEmpty_0085 (ST74a, ST74c): NewStaticToken stores a
// blank value as ""; WithAPIKey resolves to a *StaticToken with Value "";
// catalog.List given NewStaticToken("   ") sends no Authorization; and the
// signin package's APIKey and DeviceKey sources yield "".
func TestStaticTokenBlankIsEmpty_0085(t *testing.T) {
	for _, b := range blanks0085 {
		t.Run(b.name, func(t *testing.T) {
			if v := llmprovider.NewStaticToken(b.value).Value; v != "" {
				t.Errorf("NewStaticToken(%q).Value = %q; want empty", b.value, v)
			}
			st, err := llmprovider.ResolveOptions(llmprovider.ProviderTogether, []llmprovider.Option{llmprovider.WithAPIKey(b.value)})
			if err != nil {
				t.Fatal(err)
			}
			if s, ok := st.TokenSource().(*llmprovider.StaticToken); !ok || s.Value != "" {
				t.Errorf("WithAPIKey(%q) resolves to %#v; want a *StaticToken with an empty Value", b.value, st.TokenSource())
			}
			for name, src := range map[string]llmprovider.TokenSource{
				"APIKey": signin.APIKey{Key: b.value}.TokenSource(), "DeviceKey": signin.DeviceKey{Key: b.value}.TokenSource()} {
				tok, _ := src.Token(context.Background())
				if tok.Value != "" {
					t.Errorf("signin.%s{Key: %q}.TokenSource() yields %q; want empty", name, b.value, tok.Value)
				}
			}
		})
	}
	// catalog.List: a Chat Completions server's listing sends no Authorization
	// for "", so none for a blank key given through NewStaticToken; so is
	// Together's (0085-MADR ST82).
	list := func(id llmprovider.ProviderID, key string) []http.Header {
		rec := &recorder0085{}
		opts := []llmprovider.Option{llmprovider.WithHTTPClient(rec.client()), catalog.WithCache(new(catalog.Cache))}
		if id == compat0085 {
			opts = append(opts, chatcompat.WithID(compat0085), llmprovider.WithBaseURL("http://127.0.0.1:1/v1"))
		}
		_, _ = catalog.List(context.Background(), id, llmprovider.NewStaticToken(key), opts...)
		return rec.all()
	}
	t.Run("catalog.List, Chat Completions server", func(t *testing.T) {
		reqs := list(compat0085, "   ")
		if len(reqs) == 0 || slicesAny0085(authorization0085(reqs)) {
			t.Errorf("%d listing request(s), Authorization %q; want at least one, none with an Authorization", len(reqs), authorization0085(reqs))
		}
	})
	t.Run("catalog.List, Together", func(t *testing.T) {
		for _, key := range []string{"   ", ""} {
			reqs := list(llmprovider.ProviderTogether, key)
			if len(reqs) == 0 || slicesAny0085(authorization0085(reqs)) {
				t.Errorf("key %q: %d listing request(s), Authorization %q; want at least one, none with an Authorization",
					key, len(reqs), authorization0085(reqs))
			}
		}
	})
}

// TestConstructorsPaddedKeySentAsGiven_0085 (ST75, a guard): a key that is not
// blank is stored and sent as given, padding and all.
func TestConstructorsPaddedKeySentAsGiven_0085(t *testing.T) {
	for _, v := range []string{" made-up-key ", "made-up-key\n"} {
		if got := llmprovider.NewStaticToken(v).Value; got != v {
			t.Errorf("NewStaticToken(%q).Value = %q; want it as given", v, got)
		}
	}
	o := run0085(t, llmprovider.ProviderTogether, llmprovider.WithAPIKey(" made-up-key "))
	if auth := authorization0085(o.headers); o.err != nil || len(auth) == 0 || auth[0] != "Bearer  made-up-key " {
		t.Errorf("err %v, Authorization %q; want \"Bearer  made-up-key \"", o.err, auth)
	}
}

// TestBlankKeyNoFallback_0085 (ST76): with the provider's environment variable
// set by the test, a blank key is no key and nothing else: Claude is refused at
// New and Kilo sends its anonymous key; neither uses the variable.
func TestBlankKeyNoFallback_0085(t *testing.T) {
	const envKey = "made-up-env-key-0085"
	t.Setenv("ANTHROPIC_API_KEY", envKey)
	t.Setenv("KILO_API_KEY", envKey)
	claude := run0085(t, llmprovider.ProviderClaude, llmprovider.WithAPIKey("   "))
	if !errors.Is(claude.err, llmprovider.ErrInvalidRequest) || len(claude.headers) != 0 {
		t.Errorf("Claude: err %v, %d request(s); want it refused at New, no request", claude.err, len(claude.headers))
	}
	kilo := run0085(t, llmprovider.ProviderKilo, llmprovider.WithAPIKey("   "))
	auth := authorization0085(kilo.headers)
	if kilo.err != nil || len(auth) == 0 || strings.Contains(strings.Join(auth, " "), envKey) ||
		strings.TrimSpace(strings.TrimPrefix(auth[0], "Bearer")) == "" {
		t.Errorf("Kilo: err %v, Authorization %q; want its anonymous key, not the variable's", kilo.err, auth)
	}
}
