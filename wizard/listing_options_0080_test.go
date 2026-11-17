package wizard

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// 0080-MADR D4: Options.listingOptions is the one place the model listing's
// options are built, and Kilo's sign-in check is given the same, so their
// order is the listing's order, for every provider. Before the refactor
// nothing outside Kilo's tests held it: a caller's own client moved after
// ProviderOptions in listingOptions, or all of ProviderOptions moved to the
// front, passed the whole wizard package although the caller's client, endpoint,
// limit and Claude workspace then stopped winning. This is a gap that existed at
// b109cea and that the refactor made matter. The test holds it through the
// listing's actual request, for Kilo, Claude, OpenAI and a chatcompat
// descriptor: ProviderOptions come last, so the caller's options win over the
// wizard's own.

// listingRecorder0080 answers every listing with three models and records, for
// each request that is not Kilo's profile, which client sent it, to which
// host, and its Claude workspace header.
type listingRecorder0080 struct {
	tag  string
	mu   *sync.Mutex
	seen *[]listingSeen0080
}

type listingSeen0080 struct{ client, host, workspace string }

func (r listingRecorder0080) RoundTrip(q *http.Request) (*http.Response, error) {
	body := `{"object":"list","data":[{"id":"gpt-5","object":"model","type":"model"},` +
		`{"id":"gpt-5-mini","object":"model","type":"model"},{"id":"o4-mini","object":"model","type":"model"}],"has_more":false}`
	if strings.HasSuffix(q.URL.Path, "/api/profile") {
		body = `{"organizations":[],"hasPersonalAccount":true}`
	} else {
		r.mu.Lock()
		*r.seen = append(*r.seen, listingSeen0080{r.tag, q.URL.Host, q.Header.Get("Anthropic-Workspace-Id")})
		r.mu.Unlock()
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
}

// defaultPrompter0080 accepts every default, chooses the API key method, gives
// a chatcompat server the endpoint it needs and types a fixed fake key, so that
// one flow serves each provider.
type defaultPrompter0080 struct{}

func (defaultPrompter0080) Select(_ string, choices []Choice, def int) (int, error) {
	for i, c := range choices {
		if strings.Contains(c.Label, "API key") {
			return i, nil
		}
	}
	return def, nil
}

func (defaultPrompter0080) MultiSelect(string, []Choice, []int) ([]int, error) { return nil, nil }
func (defaultPrompter0080) Confirm(_ string, def bool) (bool, error)           { return def, nil }
func (defaultPrompter0080) Secret(string) (string, error)                      { return "fake-key-0080", nil }
func (defaultPrompter0080) Notify(Level, string, ...any)                       {}

func (defaultPrompter0080) Input(_, def string) (string, error) {
	if def == "" {
		return "https://compat.example.test/v1", nil
	}
	return def, nil
}

func TestListingOptions_CallerOptionsComeLast_0080(t *testing.T) {
	const compatID llmprovider.ProviderID = "lmstudio"
	const override = "override.example.test"
	reg := providers.Default()
	if err := reg.Register(chatcompat.Descriptor(compatID, "LM Studio"), func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return chatcompat.New(append(opts, chatcompat.WithID(compatID))...)
	}); err != nil {
		t.Fatal(err)
	}
	type row struct {
		name  string
		opts  func(caller *http.Client) []llmprovider.Option
		check func(t *testing.T, id llmprovider.ProviderID, seen []listingSeen0080, recommended int)
	}
	onlyClient := func(want string) func(*testing.T, llmprovider.ProviderID, []listingSeen0080, int) {
		return func(t *testing.T, id llmprovider.ProviderID, seen []listingSeen0080, _ int) {
			t.Helper()
			for _, s := range seen {
				if s.client != want {
					t.Errorf("%s: a listing request went through the %s client; want the %s one: %+v", id, s.client, want, seen)
				}
			}
		}
	}
	rows := []row{
		{"the wizard's own, with no caller option", func(*http.Client) []llmprovider.Option { return nil },
			func(t *testing.T, id llmprovider.ProviderID, seen []listingSeen0080, recommended int) {
				t.Helper()
				onlyClient("wizard")(t, id, seen, recommended)
				for _, s := range seen {
					if s.host == override || s.workspace != "" {
						t.Errorf("%s: request %+v; want the wizard's endpoint and no workspace", id, s)
					}
				}
				if recommended != 2 {
					t.Errorf("%s: %d recommended; want the wizard's ModelLimit, 2", id, recommended)
				}
			}},
		{"the caller's client", func(c *http.Client) []llmprovider.Option { return []llmprovider.Option{llmprovider.WithHTTPClient(c)} },
			onlyClient("caller")},
		{"the caller's endpoint", func(*http.Client) []llmprovider.Option {
			return []llmprovider.Option{llmprovider.WithBaseURL("https://" + override + "/api/gateway")}
		}, func(t *testing.T, id llmprovider.ProviderID, seen []listingSeen0080, _ int) {
			t.Helper()
			for _, s := range seen {
				if s.host != override {
					t.Errorf("%s: a listing request went to %s; want the caller's endpoint, %s", id, s.host, override)
				}
			}
		}},
		{"the caller's limit", func(*http.Client) []llmprovider.Option { return []llmprovider.Option{catalog.WithLimit(1)} },
			func(t *testing.T, id llmprovider.ProviderID, _ []listingSeen0080, recommended int) {
				t.Helper()
				if recommended != 1 {
					t.Errorf("%s: %d recommended; want the caller's limit, 1, not the wizard's 2", id, recommended)
				}
			}},
		{"the caller's Claude workspace in a For", func(*http.Client) []llmprovider.Option {
			return []llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, catalog.WithClaudeWorkspace("ws-x-0080"))}
		}, func(t *testing.T, id llmprovider.ProviderID, seen []listingSeen0080, _ int) {
			t.Helper()
			for _, s := range seen {
				want := ""
				if id == llmprovider.ProviderClaude {
					want = "ws-x-0080"
				}
				if s.workspace != want {
					t.Errorf("%s: workspace header %q; want %q: %+v", id, s.workspace, want, seen)
				}
			}
		}},
	}
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderKilo, llmprovider.ProviderClaude, llmprovider.ProviderOpenAI, compatID} {
		for _, r := range rows {
			// A third party's provider refuses catalog.WithLimit in
			// ProviderOptions (0038-MADR D5, A2;
			// TestWizard_ProviderOptionsLimitRefused), so that row does not
			// apply to it.
			if id == compatID && r.name == "the caller's limit" {
				continue
			}
			t.Run(string(id)+", "+r.name, func(t *testing.T) {
				var mu sync.Mutex
				var seen []listingSeen0080
				wizard := &http.Client{Transport: listingRecorder0080{"wizard", &mu, &seen}}
				caller := &http.Client{Transport: listingRecorder0080{"caller", &mu, &seen}}
				opts := append([]llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}, r.opts(caller)...)
				o := Options{Registry: reg, Discover: true, HTTPClient: wizard, ProviderOptions: opts,
					Profile: catalog.ProfileCapable, ModelLimit: 2}
				_, cat, err := ResolveCredential(context.Background(), defaultPrompter0080{}, o, id)
				if err != nil {
					t.Fatalf("ResolveCredential: %v", err)
				}
				mu.Lock()
				got := append([]listingSeen0080(nil), seen...)
				mu.Unlock()
				if len(got) == 0 {
					t.Fatalf("no listing request was sent")
				}
				r.check(t, id, got, len(cat.Recommended))
			})
		}
	}
}
