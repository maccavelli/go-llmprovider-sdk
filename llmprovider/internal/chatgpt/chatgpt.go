// Package chatgpt is what a ChatGPT session's requests share between
// providers/openai and catalog's ListRaw (0041-MADR D9): the session check,
// the headers Codex sends, and the listing request. Moved from
// providers/openai in 0041-PLAN Phase 8, unchanged. A ChatGPT plan session's
// check and listing request are here too (0043-MADR D5).
package chatgpt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// BaseURL is the ChatGPT subscription endpoint, the Codex backend.
const BaseURL = "https://chatgpt.com/backend-api/codex"

// ChatGPT session headers, as Codex sends them.
const (
	// HeaderAccount carries a ChatGPT session's account id.
	HeaderAccount = "ChatGPT-Account-Id"
	// HeaderOriginator names the client to the ChatGPT backend.
	HeaderOriginator = "originator"
	// OriginatorValue is this module's originator (0002-MADR §5).
	OriginatorValue = "go-llmprovider-sdk"
	// HeaderFedRAMP marks a FedRAMP account's requests, as Codex's bearer auth
	// does (model-provider/src/bearer_auth_provider.rs:43-45).
	HeaderFedRAMP = "X-OpenAI-Fedramp"
)

// accountSource is a session that knows its ChatGPT account:
// *auth.OAuthSession and *auth.VendorCLISession.
type accountSource interface {
	Account() (id string, fedRAMP bool)
}

// IsSession reports whether src is a ChatGPT login: an OpenAI OAuth session,
// or the Codex CLI's.
func IsSession(src llmprovider.TokenSource) bool {
	if vendor, ok := src.(*auth.VendorCLISession); ok {
		return vendor.Provider == llmprovider.ProviderOpenAI
	}
	session, ok := src.(*auth.OAuthSession)
	return ok && session.ChatGPT()
}

// IsPlanSession reports whether src is a Sign in with ChatGPT plan session
// (auth.OAuthSession.PlanUsage): its requests go to the platform's Responses
// API, never to this backend (0043-MADR D1, D5).
func IsPlanSession(src llmprovider.TokenSource) bool {
	session, ok := src.(*auth.OAuthSession)
	return ok && session.PlanUsage()
}

// PlanModelsRequest is a plan session's listing request: GET baseURL/models
// with the user agent and the bearer token, and none of this backend's
// headers or its client_version (0043-MADR D5, M6).
func PlanModelsRequest(ctx context.Context, baseURL, userAgent string, token llmprovider.Token) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("model listing: create plan models request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	token.Apply(req, "Authorization", "Bearer")
	return req, nil
}

// SetHeaders sets the originator, and the account id and FedRAMP flag of src
// when it knows them.
func SetHeaders(req *http.Request, src llmprovider.TokenSource) {
	req.Header.Set(HeaderOriginator, OriginatorValue)
	account, ok := src.(accountSource)
	if !ok {
		return
	}
	id, fedRAMP := account.Account()
	if id != "" {
		req.Header.Set(HeaderAccount, id)
	}
	if fedRAMP {
		req.Header.Set(HeaderFedRAMP, "true")
	}
}

// ModelsClientVersion is the client_version a build with no release version
// sends. The ChatGPT backend's /models endpoint requires the parameter; 0.0.0
// is accepted but hides models whose minimal_client_version is higher
// (gpt-6-sol and gpt-6-luna on 2026-09-27).
const ModelsClientVersion = "0.0.0"

// versionRE reads X.Y.Z from a module version: the only form the backend
// accepts ("v1.5.0", "1.5" and "(devel)" answer 400, as does anything over 32
// characters; measured 2026-09-27). A pseudo-version names the release it
// precedes.
var versionRE = regexp.MustCompile(`^v?(\d{1,9}\.\d{1,9}\.\d{1,9})(?:[-+].*)?$`)

// ClientVersion is the client_version for this module at version v: its own
// release, never a Codex version string (MADR 0012 §4.3), else
// ModelsClientVersion.
func ClientVersion(v string) string {
	if m := versionRE.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return ModelsClientVersion
}

// ModelsRequest is a ChatGPT session's listing request: GET
// baseURL/models?client_version=…, with the user agent, the bearer token and
// the session's headers.
func ModelsRequest(ctx context.Context, baseURL, userAgent string, token llmprovider.Token,
	src llmprovider.TokenSource) (*http.Request, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/models")
	if err != nil {
		return nil, fmt.Errorf("model listing: parse chatgpt models URL: %w", err)
	}
	query := endpoint.Query()
	sdk, _ := transport.BuildVersions()
	query.Set("client_version", ClientVersion(sdk))
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("model listing: create chatgpt models request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	token.Apply(req, "Authorization", "Bearer")
	SetHeaders(req, src)
	return req, nil
}
