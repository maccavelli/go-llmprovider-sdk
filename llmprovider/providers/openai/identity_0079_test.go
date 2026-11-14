package openai

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// A provider hands WithClientInfo to the session it uses (0079-MADR D4;
// 0079-PLAN Phase 3).

// storedChatGPTSession0079 is a ChatGPT session saved by a FileTokenStore and
// loaded again, so it has no identity and no client, as a consumer's session
// from a store has not.
func storedChatGPTSession0079(t *testing.T) *auth.OAuthSession {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	saved := &auth.OAuthSession{
		Provider: llmprovider.ProviderOpenAI,
		Issuer:   auth.DefaultOpenAIIssuer,
		ClientID: auth.DefaultOpenAIClientID,
		Access:   "old-access",
		Refresh:  "old-refresh",
		Expiry:   time.Now().Add(time.Hour),
		TokenURL: "https://auth.test/oauth/token",
	}
	if err := store.Save(t.Context(), llmprovider.ProviderOpenAI, saved); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(t.Context(), llmprovider.ProviderOpenAI)
	if err != nil || loaded == nil {
		t.Fatalf("Load() = %v, %v", loaded, err)
	}
	return loaded
}

// refreshAgent0079 generates once through a provider built with opts over
// session, whose first answer is a 401 that forces a refresh, and returns the
// User-Agent the refresh sent.
func refreshAgent0079(t *testing.T, session *auth.OAuthSession, opts ...llmprovider.Option) string {
	t.Helper()
	var agents []string
	var generateCalls int
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "chatgpt.com":
			generateCalls++
			if generateCalls == 1 {
				return httpResponse(request, http.StatusUnauthorized, ""), nil
			}
			return httpResponse(request, http.StatusOK, okResponse), nil
		case "auth.test":
			agents = append(agents, request.Header.Get("User-Agent"))
			return httpResponse(request, http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`), nil
		}
		t.Errorf("unexpected request to %s", request.URL.Host)
		return httpResponse(request, http.StatusInternalServerError, ""), nil
	})}
	p := sessionProvider(t, session, "gpt-5.4-mini", append([]llmprovider.Option{llmprovider.WithHTTPClient(client)}, opts...)...)
	if _, err := llmprovider.GenerateText(context.Background(), p, text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("the refresh was sent %d times, want once", len(agents))
	}
	return agents[0]
}

func TestProviderNamesSession_0079(t *testing.T) {
	t.Run("named", func(t *testing.T) {
		got := refreshAgent0079(t, storedChatGPTSession0079(t), llmprovider.WithClientInfo("probe-app", "1.2.3"))
		if !strings.HasPrefix(got, "probe-app/1.2.3 (") || !strings.Contains(got, "go-llmprovider-sdk/") {
			t.Errorf("refresh User-Agent = %q, want it to begin %q and name this module", got, "probe-app/1.2.3 (")
		}
	})
	t.Run("unnamed", func(t *testing.T) {
		got := refreshAgent0079(t, storedChatGPTSession0079(t))
		if !strings.HasPrefix(got, "go-llmprovider-sdk/") || strings.Contains(got, "probe-app") {
			t.Errorf("refresh User-Agent = %q, want this module's", got)
		}
	})
	t.Run("kept", func(t *testing.T) {
		session := storedChatGPTSession0079(t)
		session.UseIdentity(auth.Identity{Name: "first-app", Version: "9.0.0"})
		got := refreshAgent0079(t, session, llmprovider.WithClientInfo("probe-app", "1.2.3"))
		if !strings.HasPrefix(got, "first-app/9.0.0 (") {
			t.Errorf("refresh User-Agent = %q, want the session's own first-app/9.0.0", got)
		}
	})
	t.Run("version", func(t *testing.T) {
		got := refreshAgent0079(t, storedChatGPTSession0079(t), llmprovider.WithClientInfo("probe-app", ""))
		st, err := llmprovider.ResolveOptions(llmprovider.ProviderOpenAI, []llmprovider.Option{llmprovider.WithClientInfo("probe-app", "")})
		if err != nil {
			t.Fatal(err)
		}
		want := "probe-app/" + st.ClientVersion() + " ("
		if !strings.HasPrefix(got, want) {
			t.Errorf("refresh User-Agent = %q, want it to begin %q, the version the provider's requests name", got, want)
		}
	})
}
