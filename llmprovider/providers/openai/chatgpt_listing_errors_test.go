package openai

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// chatGPTListingSession is a ChatGPT session whose refresh goes to auth.test.
func chatGPTListingSession(client *http.Client) *auth.OAuthSession {
	return &auth.OAuthSession{
		Provider:   llmprovider.ProviderOpenAI,
		Issuer:     auth.DefaultOpenAIIssuer,
		ClientID:   auth.DefaultOpenAIClientID,
		Access:     "old-access",
		Refresh:    "old-refresh",
		Expiry:     time.Now().Add(time.Hour),
		TokenURL:   "https://auth.test/oauth/token",
		HTTPClient: client,
	}
}

// TestListModels_ChatGPT401RefreshesOnce (0020-MADR F38): a ChatGPT listing's
// 401 refreshes the session and lists once more, as Generate does (F2).
func TestListModels_ChatGPT401RefreshesOnce(t *testing.T) {
	var listCalls, refreshCalls int
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host {
		case "chatgpt.com":
			listCalls++
			if listCalls == 1 {
				return httpResponse(r, http.StatusUnauthorized, `{"detail":"token expired"}`), nil
			}
			if got := r.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Errorf("second listing Authorization = %q", got)
			}
			return httpResponse(r, http.StatusOK, `{"models":[{"slug":"gpt-6-astra","visibility":"list","priority":1}]}`), nil
		case "auth.test":
			refreshCalls++
			return httpResponse(r, http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`), nil
		default:
			return nil, errors.New("unexpected request host")
		}
	})}
	p := sessionProvider(t, chatGPTListingSession(client), "gpt-6-astra", llmprovider.WithHTTPClient(client))
	models, err := list(t, p)
	if err != nil || !slices.Equal(models, []string{"gpt-6-astra"}) {
		t.Fatalf("ListModels = %v, %v; want the listing after one refresh", models, err)
	}
	if listCalls != 2 || refreshCalls != 1 {
		t.Errorf("listing/refresh calls = %d/%d, want 2/1", listCalls, refreshCalls)
	}
}

// TestListModels_ChatGPTFailureIsAnAPIError (0020-MADR F38): a listing that
// is not 200 is classified like any other answer. A 403 refuses something the
// session may not do, not the session, so it is neither refreshed nor sent
// again (0028-MADR D-H2): the ChatGPT backend refuses a dead session with 401
// (0028-PLAN D5), which TestListModels_ChatGPT401RefreshesOnce covers.
func TestListModels_ChatGPTFailureIsAnAPIError(t *testing.T) {
	var listCalls, refreshCalls int
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "auth.test" {
			refreshCalls++
			return httpResponse(r, http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`), nil
		}
		listCalls++
		return httpResponse(r, http.StatusForbidden, `{"detail":"not allowed"}`), nil
	})}
	p := sessionProvider(t, chatGPTListingSession(client), "gpt-6-astra", llmprovider.WithHTTPClient(client))
	_, err := list(t, p)
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || apiErr.Status != http.StatusForbidden {
		t.Errorf("ListModels = %v, want an *APIError with status 403", err)
	}
	if listCalls != 1 || refreshCalls != 0 {
		t.Errorf("listing/refresh calls = %d/%d, want 1/0", listCalls, refreshCalls)
	}
}
