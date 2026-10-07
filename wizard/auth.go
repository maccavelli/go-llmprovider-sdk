package wizard

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// CredentialKind identifies the credential represented by a Result.
type CredentialKind string

const (
	// CredNone means the provider does not require a credential.
	CredNone CredentialKind = ""
	// CredAPIKey means APIKey contains a static provider key.
	CredAPIKey CredentialKind = "api_key"
	// CredOAuth means the OAuth fields contain a refreshable or access-only session.
	CredOAuth CredentialKind = "oauth"
	// CredVendorCLI means VendorAuthPath names a vendor CLI's auth file, read on
	// every request through auth.VendorCLISession; the CLI keeps it
	// fresh and this module holds no token (MADR 0012 §5.1).
	CredVendorCLI CredentialKind = "vendor_cli"
)

var (
	loginBrowserOAuth = auth.LoginBrowserOAuth
	loginDeviceOAuth  = auth.LoginDeviceOAuth
	kiloProfile       = auth.KiloProfile
)

type resolvedCredential struct {
	kind         CredentialKind
	apiKey       string
	session      *auth.OAuthSession
	source       llmprovider.TokenSource
	vendorPath   string
	organization string
}

// offeredAuthMethods returns the methods the caller can keep. Without a
// TokenStore, a method that saves a session is left out (0020-MADR F51).
func offeredAuthMethods(d llmprovider.Descriptor, haveStore bool) []llmprovider.AuthMethod {
	if haveStore {
		return d.AuthMethods
	}
	var keep []llmprovider.AuthMethod
	for _, m := range d.AuthMethods {
		if !needsStore(d.ID, m.ID) {
			keep = append(keep, m)
		}
	}
	return keep
}

// needsStore reports whether method saves a session to the TokenStore. The
// API key and the vendor CLI read-through save nothing, nor does a pasted
// Grok key; a pasted OpenAI credential may be a ChatGPT access token, which
// is saved.
func needsStore(provider llmprovider.ProviderID, method llmprovider.AuthMethodID) bool {
	switch method {
	case llmprovider.AuthAPIKey, llmprovider.AuthImportVendorCLI:
		return false
	case llmprovider.AuthTokenStdin:
		return provider == llmprovider.ProviderOpenAI
	default:
		return true
	}
}

// existingMethod is the method menu's default: the method that made the
// credential Existing names for d, else the first (0020-MADR F19).
func existingMethod(methods []llmprovider.AuthMethod, d llmprovider.Descriptor, o Options) int {
	if o.Existing.Provider != d.ID {
		return 0
	}
	var want []llmprovider.AuthMethodID
	switch o.Existing.Kind {
	case CredAPIKey:
		want = []llmprovider.AuthMethodID{llmprovider.AuthAPIKey}
	case CredVendorCLI:
		want = []llmprovider.AuthMethodID{llmprovider.AuthImportVendorCLI}
	case CredOAuth:
		want = []llmprovider.AuthMethodID{llmprovider.AuthBrowserOAuth, llmprovider.AuthDeviceCode}
	}
	for _, id := range want {
		if i := slices.IndexFunc(methods, func(m llmprovider.AuthMethod) bool { return m.ID == id }); i >= 0 {
			return i
		}
	}
	return 0
}

func resolveCredential(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	baseURL string,
	o Options,
) (resolvedCredential, error) {
	methods := offeredAuthMethods(d, o.TokenStore != nil)
	if len(methods) == 0 || (len(methods) == 1 && methods[0].ID == llmprovider.AuthAPIKey) {
		key, err := resolveAPIKey(p, d, o)
		if err != nil {
			return resolvedCredential{}, err
		}
		kind := CredNone
		if d.RequiresAPIKey {
			kind = CredAPIKey
		}
		return staticCredential(kind, key), nil
	}

	// A saved session is offered first, whatever method made it (0020-MADR
	// F19).
	if session, keep, keepErr := keepExistingOAuth(ctx, p, d, o); keepErr != nil {
		return resolvedCredential{}, keepErr
	} else if keep {
		cred := oauthCredential(session)
		if d.ID == llmprovider.ProviderKilo {
			cred.organization = o.Existing.Organization
		}
		return cred, nil
	}

	choices := make([]Choice, 0, len(methods))
	for _, method := range methods {
		choices = append(choices, Choice{Label: method.Label, Detail: method.Detail})
	}
	idx, err := choose(p, fmt.Sprintf("Choose how to authenticate with %s:", d.Label), choices,
		existingMethod(methods, d, o))
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("select authentication method: %w", err)
	}
	method := methods[idx].ID
	if method == llmprovider.AuthAPIKey {
		key, keyErr := resolveAPIKey(p, d, o)
		if keyErr != nil {
			return resolvedCredential{}, keyErr
		}
		return staticCredential(CredAPIKey, key), nil
	}
	if method == llmprovider.AuthTokenStdin {
		return resolveTokenStdin(ctx, p, d, o)
	}
	if method == llmprovider.AuthImportVendorCLI {
		return resolveVendorCLI(ctx, p, d, o)
	}
	switch method {
	case llmprovider.AuthBrowserOAuth:
		flow, drain := browserFlowOptions(p, o)
		session, loginErr := loginBrowserOAuth(ctx, d.ID, flow)
		drain()
		if loginErr != nil {
			return resolvedCredential{}, loginErr
		}
		return saveOAuthCredential(ctx, o.TokenStore, d.ID, session)
	case llmprovider.AuthDeviceCode:
		if d.ID == llmprovider.ProviderKilo {
			return resolveKiloDevice(ctx, p, d, baseURL, o)
		}
		session, loginErr := loginDeviceOAuth(ctx, d.ID, oauthFlowOptions(p, o))
		if loginErr != nil {
			return resolvedCredential{}, loginErr
		}
		return saveOAuthCredential(ctx, o.TokenStore, d.ID, session)
	default:
		return resolvedCredential{}, fmt.Errorf("wizard: unsupported authentication method %q", method)
	}
}

// resolveVendorCLI uses a vendor CLI's login read-through: it checks the
// file holds a live token, asks, and returns the path, never the tokens
// (MADR 0012 §5.1). No TokenStore is needed. The path Existing names for d
// comes first (0020-MADR F19).
func resolveVendorCLI(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (resolvedCredential, error) {
	path := ""
	if o.Existing.Provider == d.ID && o.Existing.Kind == CredVendorCLI {
		path = o.Existing.VendorAuthPath
	}
	if path == "" {
		var err error
		if path, err = vendorAuthPath(d.ID, o); err != nil {
			return resolvedCredential{}, err
		}
	}
	session := &auth.VendorCLISession{Provider: d.ID, Path: path}
	token, err := session.Token(ctx)
	if err != nil {
		return resolvedCredential{}, err
	}
	use, err := p.Confirm(fmt.Sprintf("Use the %s CLI login in %s (%s)?", d.Label, path,
		redact.MaskSecret(token.Value)), true)
	if err != nil {
		return resolvedCredential{}, err
	}
	if !use {
		return resolvedCredential{}, errors.New("wizard: vendor CLI login declined")
	}
	return resolvedCredential{kind: CredVendorCLI, source: session, vendorPath: path}, nil
}

// resolveKiloDevice runs Kilo's device login (0017-MADR D2). The token has no
// refresh and no expiry: it is saved to the TokenStore, its only copy, and the
// session is the Kilo credential (0016-MADR D11, A7). When the account
// belongs to organizations, the user chooses one, or the personal account;
// the one Existing names is the default. The profile is read from the
// endpoint the user entered, with the caller's client (0020-MADR F19, F20).
func resolveKiloDevice(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	baseURL string,
	o Options,
) (resolvedCredential, error) {
	session, err := loginDeviceOAuth(ctx, d.ID, oauthFlowOptions(p, o))
	if err != nil {
		return resolvedCredential{}, err
	}
	if session == nil || session.Access == "" {
		return resolvedCredential{}, errors.New("wizard: Kilo login returned no token")
	}
	session.Provider, session.Store = d.ID, o.TokenStore
	if err := o.TokenStore.Save(ctx, d.ID, session); err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: save Kilo login: %w", err)
	}
	cred := oauthCredential(session)
	var opts []llmprovider.Option
	if baseURL != "" {
		opts = append(opts, llmprovider.WithBaseURL(baseURL))
	}
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	opts = append(opts, o.ProviderOptions...)
	account, err := kiloProfile(ctx, session.Access, opts...)
	if err != nil {
		p.Notify(LevelWarn, "could not read the Kilo account's organizations (%v); using the personal account", err)
		return cred, nil
	}
	if len(account.Organizations) == 0 {
		return cred, nil
	}
	var choices []Choice
	var ids []string
	if account.HasPersonalAccount {
		choices, ids = append(choices, Choice{Label: "Personal account"}), append(ids, "")
	}
	want := account.SelectedOrganizationID
	if o.Existing.Provider == d.ID && o.Existing.Organization != "" {
		want = o.Existing.Organization
	}
	defaultIdx := 0
	for _, org := range account.Organizations {
		if org.ID == want {
			defaultIdx = len(ids)
		}
		choices, ids = append(choices, Choice{Label: org.Name, Detail: org.Role}), append(ids, org.ID)
	}
	idx, err := choose(p, "Use Kilo as:", choices, defaultIdx)
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("select Kilo organization: %w", err)
	}
	cred.organization = ids[idx]
	return cred, nil
}

func staticCredential(kind CredentialKind, key string) resolvedCredential {
	return resolvedCredential{
		kind:   kind,
		apiKey: key,
		source: llmprovider.NewStaticToken(key),
	}
}

func oauthCredential(session *auth.OAuthSession) resolvedCredential {
	return resolvedCredential{kind: CredOAuth, session: session, source: session}
}

// keepExistingOAuth offers the session the store holds for d when Existing
// names d as an OAuth credential, before the method menu, whatever method
// made it. The store is the session's only copy (0016-MADR A10); with none
// there, the user signs in again.
func keepExistingOAuth(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (*auth.OAuthSession, bool, error) {
	if o.Existing.Kind != CredOAuth || o.Existing.Provider != d.ID || o.TokenStore == nil {
		return nil, false, nil
	}
	session, err := o.TokenStore.Load(ctx, d.ID)
	if err != nil {
		return nil, false, fmt.Errorf("wizard: load the saved %s session: %w", d.Label, err)
	}
	if session == nil || session.Access == "" {
		return nil, false, nil
	}
	keep, err := p.Confirm(
		fmt.Sprintf("Keep the existing session (%s)?", redact.MaskSecret(session.Access)), true)
	if err != nil {
		return nil, false, err
	}
	if !keep {
		return nil, false, nil
	}
	session.Store = o.TokenStore
	// The kept session refreshes through the caller's client, as the
	// listing and a new sign-in do (0016-MADR D8; 0026-MADR F10). A nil
	// client changes nothing.
	session.UseHTTPClient(o.HTTPClient)
	if err := validateKept(d.ID, session); err != nil {
		return nil, false, fmt.Errorf("wizard: the saved %s session cannot be kept (%w); sign in again", d.Label, err)
	}
	return session, true, nil
}

// validateKept checks that a saved session can be used again. A Kilo session
// has no refresh token and never expires, so Kilo's rule is a token for Kilo
// (0020-MADR F19); every other session must be refreshable.
func validateKept(provider llmprovider.ProviderID, session *auth.OAuthSession) error {
	if provider != llmprovider.ProviderKilo {
		return auth.ValidateOAuthSession(session)
	}
	if session.Provider != llmprovider.ProviderKilo {
		return fmt.Errorf("it is a %q session", session.Provider)
	}
	return nil
}

func resolveTokenStdin(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (resolvedCredential, error) {
	// CODEX_ACCESS_TOKEN is not offered: in Codex it holds a personal access
	// token or an agent-identity JWT, not a ChatGPT OAuth bearer (MADR 0012 §5.4).
	value, err := p.Secret(fmt.Sprintf("Paste your %s credential", d.Label))
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("enter credential: %w", err)
	}
	// A paste's spaces would make an API key look like an access token
	// (0020-MADR F18).
	value = strings.TrimSpace(value)
	if d.ID == llmprovider.ProviderOpenAI {
		if vendor := foreignKeyVendor(value); vendor != "" {
			return resolvedCredential{}, fmt.Errorf("wizard: this looks like %s key, not an OpenAI credential", vendor)
		}
		if !strings.HasPrefix(value, "sk-") {
			if err := checkAccessToken(value, time.Now()); err != nil {
				return resolvedCredential{}, err
			}
			return saveAccessOnlyOpenAI(ctx, o, value)
		}
	}
	return staticCredential(CredAPIKey, value), nil
}

// foreignKeyPrefixes are other vendors' key prefixes, checked in order:
// sk-ant- before OpenAI's own sk- (0021-MADR Z10).
var foreignKeyPrefixes = []struct{ prefix, vendor string }{
	{"sk-ant-", "an Anthropic"}, {"xai-", "an xAI"}, {"tgp_", "a Together"},
	{"hf_", "a Hugging Face"}, {"AIza", "a Google"},
}

// foreignKeyVendor names the vendor whose key value looks like, with its
// article, or "".
func foreignKeyVendor(value string) string {
	for _, k := range foreignKeyPrefixes {
		if strings.HasPrefix(value, k.prefix) {
			return k.vendor
		}
	}
	return ""
}

// checkAccessToken refuses a pasted value that is not a ChatGPT access token:
// a JWT of three base64url segments whose header is JSON with an alg, and
// whose exp, when it has one, is after now (0021-MADR Z10). The session is
// still saved with no Expiry, so ValidateOAuthSession's rule for an
// access-only session is unchanged.
func checkAccessToken(value string, now time.Time) error {
	notJWT := errors.New("wizard: this is not a ChatGPT access token (a JWT) or an OpenAI API key (sk-…)")
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return notJWT
	}
	segment := func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "=")) }
	header, err := segment(parts[0])
	if err != nil {
		return notJWT
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(header, &h) != nil || h.Alg == "" {
		return notJWT
	}
	payload, err := segment(parts[1])
	if err != nil {
		return notJWT
	}
	// The signature is opaque: only its alphabet is checked.
	if strings.Trim(parts[2], "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_") != "" {
		return notJWT
	}
	var claims struct {
		Exp *float64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return notJWT
	}
	if claims.Exp != nil && !time.Unix(int64(*claims.Exp), 0).After(now) {
		return errors.New("wizard: this ChatGPT access token has expired; sign in again for a fresh one")
	}
	return nil
}

func saveAccessOnlyOpenAI(ctx context.Context, o Options, access string) (resolvedCredential, error) {
	return saveOAuthCredential(
		ctx,
		o.TokenStore,
		llmprovider.ProviderOpenAI,
		accessOnlyOpenAISession(access),
	)
}

func accessOnlyOpenAISession(access string) *auth.OAuthSession {
	account, fedramp := accessTokenAccount(access)
	return &auth.OAuthSession{
		Provider:  llmprovider.ProviderOpenAI,
		Access:    access,
		Issuer:    auth.DefaultOpenAIIssuer,
		ClientID:  auth.DefaultOpenAIClientID,
		AccountID: account,
		FedRAMP:   fedramp,
	}
}

// accessTokenAccount reads a ChatGPT access token's account id and FedRAMP
// flag, from the claims a login reads in its id_token: chatgpt_account_id,
// top level or under "https://api.openai.com/auth", and
// chatgpt_account_is_fedramp. A token without them gives "" and false
// (0026-MADR F48).
func accessTokenAccount(access string) (string, bool) {
	parts := strings.Split(access, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return "", false
	}
	var claims struct {
		AccountID string `json:"chatgpt_account_id"`
		Auth      struct {
			AccountID string `json:"chatgpt_account_id"`
			FedRAMP   bool   `json:"chatgpt_account_is_fedramp"`
		} `json:"https://api.openai.com/auth"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return "", false
	}
	return cmp.Or(claims.AccountID, claims.Auth.AccountID), claims.Auth.FedRAMP
}

// oauthFlowOptions is a sign-in's options: the caller's client, and the
// device code shown through p (0020-MADR F20).
func oauthFlowOptions(p Prompter, o Options) auth.OAuthFlowOptions {
	return auth.OAuthFlowOptions{
		HTTPClient: o.HTTPClient,
		NotifyDevice: func(verificationURL, userCode string) {
			p.Notify(LevelInfo, "Open %s and enter code %s", verificationURL, userCode)
		},
	}
}

const pasteCodePrompt = "Paste the redirected URL or authorization code if the browser does not return"

// browserFlowOptions adds MADR 0008 D3's paste-code race to oauthFlowOptions.
// The authorize URL and the paste instruction are shown even when the consumer
// opens the browser itself. The returned drain must run once the login
// returns: it finishes a paste prompt the loopback overtook, so no later
// prompt reads alongside it.
func browserFlowOptions(p Prompter, o Options) (auth.OAuthFlowOptions, func()) {
	paste := &pastePrompt{p: p, shown: make(chan struct{})}
	flow := oauthFlowOptions(p, o)
	flow.OpenURL = func(authorizeURL string) error {
		paste.show(authorizeURL)
		if o.OpenURL == nil {
			return nil
		}
		return o.OpenURL(authorizeURL)
	}
	flow.InputCode = paste.input
	return flow, paste.drain
}

// pastePrompt is the paste-code prompt that races the loopback. The login
// calls show and input on their own goroutines, so the prompter is used by
// at most one of them at a time: input waits for show, and at most one read
// is ever pending.
type pastePrompt struct {
	p     Prompter
	shown chan struct{} // closed once the URL and instruction are shown

	mu      sync.Mutex
	done    bool          // drain ran: nothing further may use the prompter
	pending chan struct{} // closed when the started read returns
}

func (s *pastePrompt) show(authorizeURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return
	}
	s.p.Notify(LevelInfo, "Open %s in your browser", authorizeURL)
	s.p.Notify(LevelInfo, "If the browser does not return here, paste the redirected URL or the authorization code")
	close(s.shown)
}

func (s *pastePrompt) input(ctx context.Context) (string, error) {
	select {
	case <-s.shown:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	s.mu.Lock()
	if s.done || ctx.Err() != nil {
		s.mu.Unlock()
		return "", context.Canceled
	}
	pending := make(chan struct{})
	s.pending = pending
	s.mu.Unlock()
	defer close(pending)
	return s.p.Input(pasteCodePrompt, "")
}

func (s *pastePrompt) drain() {
	s.mu.Lock()
	s.done = true
	pending := s.pending
	s.mu.Unlock()
	if pending == nil {
		return
	}
	select {
	case <-pending:
	default:
		s.p.Notify(LevelInfo, "Browser sign-in finished; press Enter to continue")
		<-pending
	}
}

func saveOAuthCredential(
	ctx context.Context,
	store auth.TokenStore,
	provider llmprovider.ProviderID,
	session *auth.OAuthSession,
) (resolvedCredential, error) {
	if session == nil {
		return resolvedCredential{}, errors.New("wizard: OAuth login returned no session")
	}
	if err := auth.ValidateOAuthSession(session); err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: refusing to save the OAuth session: %w", err)
	}
	session.Provider = provider
	session.Store = store
	if err := store.Save(ctx, provider, session); err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: save OAuth session: %w", err)
	}
	return oauthCredential(session), nil
}
