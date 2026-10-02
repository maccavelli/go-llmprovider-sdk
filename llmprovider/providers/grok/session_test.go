package grok

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Ported from llmprovider's grok_oauth_test.go (0015-PLAN S7).
// newGrokWithSource is New with WithTokenSource.

const sessionResponse = `{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`

func grokSession() *auth.OAuthSession {
	return &auth.OAuthSession{
		Provider: llmprovider.ProviderGrok,
		Issuer:   auth.DefaultGrokOAuthIssuer,
		Access:   "session-access",
		Refresh:  "session-refresh",
		Expiry:   time.Now().Add(time.Hour),
	}
}

func TestGrok_SessionUsesAPIXAIHost(t *testing.T) {
	t.Parallel()

	var captured *http.Request
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		captured = request
		return httpResponse(request, http.StatusOK, sessionResponse), nil
	})}
	p := build(t, llmprovider.WithTokenSource(grokSession()), llmprovider.WithModel("grok-4.6"), llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if captured == nil {
		t.Fatal("Generate() made no request")
	}
	if captured.URL.Host != "api.x.ai" || captured.URL.Path != "/v1/responses" ||
		strings.Contains(captured.URL.String(), "cli-chat-proxy") {
		t.Fatalf("request URL = %s, want api.x.ai/v1/responses", captured.URL.Redacted())
	}
}

func TestGrok_SessionOmitsCLITokenAuthHeader(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		for header := range request.Header {
			if strings.EqualFold(header, "X-XAI-Token-Auth") {
				t.Errorf("request unexpectedly contains %s", header)
			}
		}
		return httpResponse(request, http.StatusOK, sessionResponse), nil
	})}
	p := build(t, llmprovider.WithTokenSource(grokSession()), llmprovider.WithModel("grok-4.6"), llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

// TestGrok_EmptyStaticKeyStillRejected: NewGrok("") and an empty StaticToken
// were refused; New refuses both forms, now with ErrInvalidRequest.
func TestGrok_EmptyStaticKeyStillRejected(t *testing.T) {
	t.Parallel()

	if _, err := New(llmprovider.WithAPIKey(""), llmprovider.WithModel("grok-4.6")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("New(WithAPIKey(\"\")) error = %v, want ErrInvalidRequest", err)
	}
	if _, err := New(llmprovider.WithTokenSource(llmprovider.NewStaticToken("")), llmprovider.WithModel("grok-4.6")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("New(empty StaticToken) error = %v, want ErrInvalidRequest", err)
	}
	if _, err := New(llmprovider.WithModel("grok-4.6")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("New() error = %v, want ErrInvalidRequest", err)
	}
}

func TestGrok_TokenSourceCalledPerRequest(t *testing.T) {
	t.Parallel()

	source := &sequenceTokenSource{values: []string{"first-token", "second-token"}}
	var authorizations []string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		authorizations = append(authorizations, request.Header.Get("Authorization"))
		return httpResponse(request, http.StatusOK, sessionResponse), nil
	})}
	p := build(t, llmprovider.WithTokenSource(source), llmprovider.WithModel("grok-4.6"), llmprovider.WithHTTPClient(client))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hello")); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
	}
	want := []string{"Bearer first-token", "Bearer second-token"}
	if !reflect.DeepEqual(authorizations, want) || source.calls != 2 {
		t.Fatalf("authorizations/calls = %v/%d, want %v/2", authorizations, source.calls, want)
	}
}

func TestGrok_OAuth401RetriesOnceAfterRefresh(t *testing.T) {
	t.Parallel()

	var generateCalls, refreshCalls int
	client := &http.Client{}
	client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Host {
		case "api.x.ai":
			generateCalls++
			if generateCalls == 1 {
				if got := request.Header.Get("Authorization"); got != "Bearer old-access" {
					t.Errorf("first Authorization = %q", got)
				}
				return httpResponse(request, http.StatusUnauthorized, ""), nil
			}
			if got := request.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Errorf("retry Authorization = %q", got)
			}
			return httpResponse(request, http.StatusOK, sessionResponse), nil
		case "auth.test":
			refreshCalls++
			if err := request.ParseForm(); err != nil {
				t.Errorf("parse refresh form: %v", err)
			}
			want := url.Values{
				"client_id":     {auth.DefaultGrokOAuthClientID},
				"grant_type":    {"refresh_token"},
				"refresh_token": {"old-refresh"},
			}
			if !reflect.DeepEqual(request.PostForm, want) {
				t.Errorf("refresh form = %v, want %v", request.PostForm, want)
			}
			return httpResponse(request, http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`), nil
		default:
			return nil, errors.New("unexpected request host")
		}
	})
	session := &auth.OAuthSession{
		Provider:   llmprovider.ProviderGrok,
		Issuer:     auth.DefaultGrokOAuthIssuer,
		ClientID:   auth.DefaultGrokOAuthClientID,
		Access:     "old-access",
		Refresh:    "old-refresh",
		Expiry:     time.Now().Add(time.Hour),
		TokenURL:   "https://auth.test/oauth/token",
		HTTPClient: client,
	}
	p := build(t, llmprovider.WithTokenSource(session), llmprovider.WithModel("grok-4.6"), llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generateCalls != 2 || refreshCalls != 1 {
		t.Fatalf("generate/refresh calls = %d/%d, want 2/1", generateCalls, refreshCalls)
	}
}

func TestGrok_Static401DoesNotRetry(t *testing.T) {
	t.Parallel()

	var calls int
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return httpResponse(request, http.StatusUnauthorized, ""), nil
	})}
	p := build(t, llmprovider.WithAPIKey("static-key"), llmprovider.WithModel("grok-4.6"), llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), text("hello")); !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("Generate() error = %v, want ErrAuthFailure", err)
	}
	if calls != 1 {
		t.Fatalf("generate calls = %d, want 1", calls)
	}
}

type sequenceTokenSource struct {
	values []string
	calls  int
}

func (source *sequenceTokenSource) Token(context.Context) (llmprovider.Token, error) {
	value := source.values[source.calls]
	source.calls++
	return llmprovider.Token{Value: value, Type: llmprovider.TokenBearer}, nil
}
