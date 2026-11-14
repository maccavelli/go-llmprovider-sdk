package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// countingClient answers every request as OpenAI would, and counts them.
func countingClient() (*http.Client, *atomic.Int32) {
	var sent atomic.Int32
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sent.Add(1)
		return httpResponse(r, http.StatusOK, okResponse), nil
	})}, &sent
}

// storedPlanSession saves a plan session holding consents in a FileTokenStore
// and returns it as loaded, so that its Store is set, with a second handle on
// the same directory, as another process has.
func storedPlanSession(t *testing.T, consents string) (*auth.OAuthSession, *auth.FileTokenStore) {
	t.Helper()
	dir := t.TempDir()
	store, err := auth.NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := planSession()
	s.Consents = consents
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || loaded == nil {
		t.Fatalf("Load = %v, %v", loaded, err)
	}
	other, err := auth.NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, other
}

// requireRefused fails unless Generate and Stream, on a marked context, give
// an error matching ErrNotPermitted that names the consent.
func requireRefused(t *testing.T, stage string, p llmprovider.Provider) {
	t.Helper()
	ctx := llmprovider.WithBackground(context.Background())
	check := func(how string, err error) {
		t.Helper()
		if !errors.Is(err, llmprovider.ErrNotPermitted) || err == nil || !strings.Contains(err.Error(), ConsentBackgroundUse) {
			t.Errorf("%s: %s err = %v; want ErrNotPermitted naming %s", stage, how, err, ConsentBackgroundUse)
		}
	}
	_, err := p.Generate(ctx, text("hi"))
	check("Generate", err)
	for _, err := range llmprovider.Stream(ctx, p, text("hi")) {
		check("Stream", err)
	}
}

// TestOpenAIBackground_Refused_0075 (0075-MADR D5): a marked request on a
// plan session, unanswered or declined, is ErrNotPermitted, through Generate
// and Stream, and nothing is sent.
func TestOpenAIBackground_Refused_0075(t *testing.T) {
	for name, consents := range map[string]string{"unanswered": "", "declined": ConsentBackgroundUse + "=declined"} {
		t.Run(name, func(t *testing.T) {
			session := planSession()
			session.Consents = consents
			client, sent := countingClient()
			p := sessionProvider(t, session, "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
			requireRefused(t, name, p)
			if got := sent.Load(); got != 0 {
				t.Errorf("%d request(s) sent; want none before the refusal", got)
			}
		})
	}
}

// TestOpenAIBackground_Allowed_0075 (0075-MADR D5): granted, or unmarked, the
// request is sent; a Codex session and a key, marked, are sent too.
func TestOpenAIBackground_Allowed_0075(t *testing.T) {
	marked := llmprovider.WithBackground(context.Background())
	granted := planSession()
	granted.Consents = ConsentBackgroundUse + "=granted"
	for _, c := range []struct {
		name string
		src  llmprovider.TokenSource
		ctx  context.Context
	}{
		{"a granted plan session, marked", granted, marked},
		{"an unanswered plan session, unmarked", planSession(), context.Background()},
		{"a declined plan session, unmarked", func() llmprovider.TokenSource {
			s := planSession()
			s.Consents = ConsentBackgroundUse + "=declined"
			return s
		}(), context.Background()},
		{"a Codex session, marked", chatGPTSession(), marked},
	} {
		t.Run(c.name, func(t *testing.T) {
			client, sent := countingClient()
			p := sessionProvider(t, c.src, "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
			if _, err := p.Generate(c.ctx, text("hi")); errors.Is(err, llmprovider.ErrNotPermitted) {
				t.Errorf("Generate err = %v; want the request sent", err)
			}
			if got := sent.Load(); got != 1 {
				t.Errorf("%d request(s) sent; want 1", got)
			}
		})
	}
	t.Run("a key, marked", func(t *testing.T) {
		client, sent := countingClient()
		p := build(t, llmprovider.WithAPIKey("sk-0075"), llmprovider.WithModel("gpt-6.1-sol"), llmprovider.WithHTTPClient(client))
		if _, err := p.Generate(marked, text("hi")); errors.Is(err, llmprovider.ErrNotPermitted) {
			t.Errorf("Generate err = %v; want the request sent", err)
		}
		if got := sent.Load(); got != 1 {
			t.Errorf("%d request(s) sent; want 1", got)
		}
	})
}

// refreshIssuer is an issuer that answers every refresh.
func refreshIssuer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "plan-access-next", "refresh_token": "plan-refresh-next", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestOpenAIBackground_ReadsStore_0075 (0075-MADR D5, C44): a SetConsent made
// through a second FileTokenStore handle, withdrawing the consent, refuses the
// very next marked request of a provider built before it; and a marked
// request beside a refresh of the same session, which rewrites its Store, is
// never refused while the consent holds, and reports no race under -race.
func TestOpenAIBackground_ReadsStore_0075(t *testing.T) {
	ctx := llmprovider.WithBackground(context.Background())
	session, other := storedPlanSession(t, ConsentBackgroundUse+"=granted")
	client, sent := countingClient()
	p := sessionProvider(t, session, "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(ctx, text("hi")); err != nil {
		t.Fatalf("Generate with the consent granted = %v", err)
	}
	if err := auth.SetConsent(context.Background(), other, llmprovider.ProviderOpenAI, ConsentBackgroundUse, false); err != nil {
		t.Fatal(err)
	}
	before := sent.Load()
	requireRefused(t, "after the withdrawal", p)
	if got := sent.Load(); got != before {
		t.Errorf("%d request(s) sent after the withdrawal; want none", got-before)
	}

	t.Run("beside a refresh", func(t *testing.T) {
		session, _ := storedPlanSession(t, ConsentBackgroundUse+"=granted")
		issuer := refreshIssuer(t)
		session.TokenURL, session.HTTPClient = issuer.URL, issuer.Client()
		client, sent := countingClient()
		p := sessionProvider(t, session, "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				session.Invalidate()
				if _, err := session.Token(context.Background()); err != nil {
					t.Errorf("refresh: %v", err)
					return
				}
				if err := session.Wait(context.Background()); err != nil {
					t.Errorf("Wait: %v", err)
					return
				}
			}
		}()
		const requests = 20
		for range requests {
			if _, err := p.Generate(ctx, text("hi")); errors.Is(err, llmprovider.ErrNotPermitted) {
				t.Errorf("Generate beside a refresh = %v; want the granted consent read", err)
			}
		}
		wg.Wait()
		if got := sent.Load(); got != requests {
			t.Errorf("%d requests sent; want %d", got, requests)
		}
	})
}

// failingLoadStore is a TokenStore whose Load fails.
type failingLoadStore struct{ err error }

func (s failingLoadStore) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	return nil, s.err
}

func (s failingLoadStore) Save(context.Context, llmprovider.ProviderID, *auth.OAuthSession) error {
	return nil
}

func (s failingLoadStore) Delete(context.Context, llmprovider.ProviderID) error { return nil }

// TestOpenAIBackground_ReadFails_0075 (0075-MADR D5, the owner's ruling of
// 2026-11-11): a marked request on a plan session whose own answer is
// granted, and whose store's Load fails, is refused as not granted and sends
// nothing.
func TestOpenAIBackground_ReadFails_0075(t *testing.T) {
	session := planSession()
	session.Consents = ConsentBackgroundUse + "=granted"
	session.Store = failingLoadStore{err: errors.New("planted load failure")}
	client, sent := countingClient()
	p := sessionProvider(t, session, "gpt-6.1-sol", llmprovider.WithHTTPClient(client))
	requireRefused(t, "a failed read", p)
	if got := sent.Load(); got != 0 {
		t.Errorf("%d request(s) sent; want none", got)
	}
	_, err := p.Generate(llmprovider.WithBackground(context.Background()), text("hi"))
	if err == nil || !strings.Contains(err.Error(), "planted load failure") {
		t.Errorf("Generate err = %v; want the read's error named", err)
	}
}
