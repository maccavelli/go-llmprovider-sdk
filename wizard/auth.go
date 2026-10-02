package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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
	// every request through llmprovider.VendorCLISession; the CLI keeps it
	// fresh and this module holds no token (MADR 0012 §5.1).
	CredVendorCLI CredentialKind = "vendor_cli"
)

var (
	loginBrowserOAuth = llmprovider.LoginBrowserOAuth
	loginDeviceOAuth  = llmprovider.LoginDeviceOAuth
	kiloProfile       = llmprovider.KiloProfile
)

type resolvedCredential struct {
	kind         CredentialKind
	apiKey       string
	session      *llmprovider.OAuthSession
	source       llmprovider.TokenSource
	vendorPath   string
	organization string
}

// offeredAuthMethods returns the methods the caller can keep. Every method
// needs somewhere to store its session except the API key, so without a
// TokenStore only the API key is offered.
func offeredAuthMethods(all []llmprovider.AuthMethod, haveStore bool) []llmprovider.AuthMethod {
	if haveStore {
		return all
	}
	var keep []llmprovider.AuthMethod
	for _, m := range all {
		if m.ID == llmprovider.AuthAPIKey {
			keep = append(keep, m)
		}
	}
	return keep
}

func resolveCredential(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (resolvedCredential, error) {
	methods := offeredAuthMethods(d.AuthMethods, o.TokenStore != nil)
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

	choices := make([]Choice, 0, len(methods))
	for _, method := range methods {
		choices = append(choices, Choice{Label: method.Label, Detail: method.Detail})
	}
	idx, err := p.Select(fmt.Sprintf("Choose how to authenticate with %s:", d.Label), choices, 0)
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
		if session, keep, keepErr := keepExistingOAuth(ctx, p, d, o); keepErr != nil {
			return resolvedCredential{}, keepErr
		} else if keep {
			return oauthCredential(session), nil
		}
		flow, drain := browserFlowOptions(p, o)
		session, loginErr := loginBrowserOAuth(ctx, d.ID, flow)
		drain()
		if loginErr != nil {
			return resolvedCredential{}, loginErr
		}
		return saveOAuthCredential(ctx, o.TokenStore, d.ID, session)
	case llmprovider.AuthDeviceCode:
		if d.ID == llmprovider.ProviderKilo {
			return resolveKiloDevice(ctx, p, d, o)
		}
		session, loginErr := loginDeviceOAuth(ctx, d.ID, oauthFlowOptions(p))
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
// (MADR 0012 §5.1). No TokenStore is needed.
func resolveVendorCLI(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (resolvedCredential, error) {
	path, err := vendorAuthPath(d.ID, o)
	if err != nil {
		return resolvedCredential{}, err
	}
	session := &llmprovider.VendorCLISession{Provider: d.ID, Path: path}
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
// belongs to organizations, the user chooses one, or the personal account.
func resolveKiloDevice(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (resolvedCredential, error) {
	session, err := loginDeviceOAuth(ctx, d.ID, oauthFlowOptions(p))
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
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
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
	defaultIdx := 0
	for _, org := range account.Organizations {
		if org.ID == account.SelectedOrganizationID {
			defaultIdx = len(ids)
		}
		choices, ids = append(choices, Choice{Label: org.Name, Detail: org.Role}), append(ids, org.ID)
	}
	idx, err := p.Select("Use Kilo as:", choices, defaultIdx)
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

func oauthCredential(session *llmprovider.OAuthSession) resolvedCredential {
	return resolvedCredential{kind: CredOAuth, session: session, source: session}
}

// keepExistingOAuth offers the session the store holds for d when Existing
// names d as an OAuth credential. The store is the session's only copy
// (0016-MADR A10); with none there, the user signs in again.
func keepExistingOAuth(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	o Options,
) (*llmprovider.OAuthSession, bool, error) {
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
	if err := llmprovider.ValidateOAuthSession(session); err != nil {
		return nil, false, fmt.Errorf("wizard: the saved %s session cannot be kept (%w); sign in again", d.Label, err)
	}
	return session, true, nil
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
	if d.ID == llmprovider.ProviderOpenAI && !strings.HasPrefix(value, "sk-") {
		return saveAccessOnlyOpenAI(ctx, o, value)
	}
	return staticCredential(CredAPIKey, value), nil
}

func saveAccessOnlyOpenAI(ctx context.Context, o Options, access string) (resolvedCredential, error) {
	return saveOAuthCredential(
		ctx,
		o.TokenStore,
		llmprovider.ProviderOpenAI,
		accessOnlyOpenAISession(access),
	)
}

func accessOnlyOpenAISession(access string) *llmprovider.OAuthSession {
	return &llmprovider.OAuthSession{
		Provider: llmprovider.ProviderOpenAI,
		Access:   access,
		Issuer:   llmprovider.DefaultOpenAIIssuer,
		ClientID: llmprovider.DefaultOpenAIClientID,
	}
}

func oauthFlowOptions(p Prompter) llmprovider.OAuthFlowOptions {
	return llmprovider.OAuthFlowOptions{
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
func browserFlowOptions(p Prompter, o Options) (llmprovider.OAuthFlowOptions, func()) {
	paste := &pastePrompt{p: p, shown: make(chan struct{})}
	flow := oauthFlowOptions(p)
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
	store llmprovider.TokenStore,
	provider llmprovider.ProviderID,
	session *llmprovider.OAuthSession,
) (resolvedCredential, error) {
	if session == nil {
		return resolvedCredential{}, errors.New("wizard: OAuth login returned no session")
	}
	if err := llmprovider.ValidateOAuthSession(session); err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: refusing to save the OAuth session: %w", err)
	}
	session.Provider = provider
	session.Store = store
	if err := store.Save(ctx, provider, session); err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: save OAuth session: %w", err)
	}
	return oauthCredential(session), nil
}
