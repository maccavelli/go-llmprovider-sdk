package wizard

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// CredentialKind identifies the credential represented by a Result. It is a
// string, safe for concurrent use.
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
	loginChatGPTPlan  = auth.LoginChatGPTPlan
	kiloProfile       = auth.KiloProfile
)

type resolvedCredential struct {
	kind         CredentialKind
	apiKey       string
	keyFrom      keySource
	session      *auth.OAuthSession
	source       llmprovider.TokenSource
	vendorPath   string
	organization string
	// saveTo is the store a new session is due to: credentialStep saves it
	// there once the listing has not refused it (0059-MADR W2).
	saveTo auth.TokenStore
	// kept is the saved session keepExistingOAuth kept: a listing that
	// refuses it leads to the sign-in menu (0066-MADR V7, Q4 (a)).
	kept bool
}

// keySource is where resolveAPIKey found an API key: a service that refuses
// a typed key is asked about another, and a refused key from anywhere else is
// an error naming it (0028-MADR D-H10).
type keySource int

const (
	keyOther       keySource = iota // not resolveAPIKey's: a pasted credential, a session
	keyTyped                        // the Secret prompt
	keyEnvironment                  // the descriptor's environment variable
	keySaved                        // Options.Existing
	keyDevice                       // Kilo's device sign-in, under Options.KiloDeviceKey
	keyAbsent                       // an optional key not given: a refusal asks for one (0066-MADR V2)
)

// offeredAuthMethods returns the methods the caller can keep. Without a
// TokenStore, a method that saves a session is left out (0020-MADR F51),
// unless Options.KiloDeviceKey makes Kilo's device sign-in save none;
// without the consumer's app name, so is the ChatGPT plan sign-in
// (0043-MADR D7). Options.SkipMethods are left out last (0056-MADR D4).
func offeredAuthMethods(d llmprovider.Descriptor, o Options) []llmprovider.AuthMethod {
	haveStore, haveName := o.TokenStore != nil, strings.TrimSpace(o.AgentName) != ""
	var keep []llmprovider.AuthMethod
	for _, m := range d.AuthMethods {
		if slices.Contains(o.SkipMethods, m.ID) {
			continue
		}
		stored := needsStore(d.ID, m.ID) && !deviceKey(d.ID, m.ID, o)
		if (haveStore || !stored) && (haveName || m.ID != llmprovider.AuthChatGPTPlan) {
			keep = append(keep, m)
		}
	}
	return keep
}

// deviceKey reports that method is Kilo's device sign-in taken as a key
// (Options.KiloDeviceKey), which saves no session (0056-MADR D2).
func deviceKey(provider llmprovider.ProviderID, method llmprovider.AuthMethodID, o Options) bool {
	return o.KiloDeviceKey && provider == llmprovider.ProviderKilo && method == llmprovider.AuthDeviceCode
}

// skippedEvery reports that Options.SkipMethods, and nothing else, left d no
// method: a menu it emptied, or the API key of a provider whose descriptor
// lists no methods, the key being implicit (0056-MADR D4).
func skippedEvery(d llmprovider.Descriptor, o Options, methods []llmprovider.AuthMethod) bool {
	if len(methods) > 0 || len(o.SkipMethods) == 0 {
		return false
	}
	if len(d.AuthMethods) == 0 {
		return d.RequiresAPIKey && slices.Contains(o.SkipMethods, llmprovider.AuthAPIKey)
	}
	unskipped := o
	unskipped.SkipMethods = nil
	return len(offeredAuthMethods(d, unskipped)) > 0
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
		// An OpenAI session under a client other than Codex's is a plan
		// session, whose client OpenAI issued the app (0043-MADR D7).
		if d.ID == llmprovider.ProviderOpenAI && o.Existing.ClientID != "" && o.Existing.ClientID != auth.DefaultOpenAIClientID {
			want = []llmprovider.AuthMethodID{llmprovider.AuthChatGPTPlan}
		}
	}
	for _, id := range want {
		if i := slices.IndexFunc(methods, func(m llmprovider.AuthMethod) bool { return m.ID == id }); i >= 0 {
			return i
		}
	}
	return 0
}

// signIn resolves d's credential by the method the user chooses: a key, a
// saved session, a browser or device sign-in, the ChatGPT plan, a pasted
// token or a vendor CLI's login. (Named so beside ResolveCredential, 0056-PLAN
// Phase 1's deviation.) offerKept is false after the listing refused the
// kept session, which is then not offered again (0066-MADR V7).
func signIn(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	baseURL string,
	o Options,
	offerKept bool,
) (resolvedCredential, error) {
	methods := offeredAuthMethods(d, o)
	if skippedEvery(d, o, methods) {
		return resolvedCredential{}, fmt.Errorf("wizard: Options.SkipMethods leaves no sign-in method for %s", d.Label)
	}
	if len(methods) == 0 || (len(methods) == 1 && methods[0].ID == llmprovider.AuthAPIKey) {
		key, from, err := resolveAPIKey(p, d, o)
		if err != nil {
			return resolvedCredential{}, err
		}
		kind := CredNone
		if d.RequiresAPIKey || key != "" {
			kind = CredAPIKey
		}
		cred := staticCredential(kind, key)
		cred.keyFrom = from
		return cred, nil
	}

	// A saved session is offered first, whatever method made it (0020-MADR
	// F19).
	if offerKept {
		session, keep, keepErr := keepExistingOAuth(ctx, p, d, o)
		if keepErr != nil {
			return resolvedCredential{}, keepErr
		}
		if keep {
			cred := oauthCredential(session)
			cred.kept = true
			if d.ID == llmprovider.ProviderKilo {
				cred.organization = o.Existing.Organization
			}
			return cred, nil
		}
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
		key, from, keyErr := resolveAPIKey(p, d, o)
		if keyErr != nil {
			return resolvedCredential{}, keyErr
		}
		cred := staticCredential(CredAPIKey, key)
		cred.keyFrom = from
		return cred, nil
	}
	if method == llmprovider.AuthTokenStdin {
		return resolveTokenStdin(ctx, p, d, o)
	}
	if method == llmprovider.AuthImportVendorCLI {
		return resolveVendorCLI(ctx, p, d, o)
	}
	if method == llmprovider.AuthChatGPTPlan {
		return resolveChatGPTPlan(ctx, p, o)
	}
	switch method {
	case llmprovider.AuthBrowserOAuth:
		flow, drain := browserFlowOptions(p, o)
		session, loginErr := loginBrowserOAuth(ctx, d.ID, flow)
		drain()
		if loginErr != nil {
			return resolvedCredential{}, loginErr
		}
		return pendingOAuthCredential(o.TokenStore, d.ID, session)
	case llmprovider.AuthDeviceCode:
		if d.ID == llmprovider.ProviderKilo {
			if o.KiloDeviceKey {
				return resolveKiloDeviceKey(ctx, p, d, o)
			}
			return resolveKiloDevice(ctx, p, d, baseURL, o)
		}
		session, loginErr := deviceLogin(ctx, d.ID, oauthFlowOptions(o), showDeviceCode(p, o))
		if loginErr != nil {
			return resolvedCredential{}, loginErr
		}
		return pendingOAuthCredential(o.TokenStore, d.ID, session)
	default:
		return resolvedCredential{}, fmt.Errorf("wizard: unsupported authentication method %q", method)
	}
}

// resolveChatGPTPlan signs the user in with their ChatGPT plan, under the
// consumer's own app name (auth.LoginChatGPTPlan; 0043-MADR D7). A plan
// session the store already holds is signed in to again, so that the client
// OpenAI issued the app is reused. With none, the user first confirms
// OpenAI's one-time notice, "You're using your ChatGPT plan", with the page
// where the plan's usage is managed. The new session replaces whatever
// OpenAI session the store held (D3).
func resolveChatGPTPlan(ctx context.Context, p Prompter, o Options) (resolvedCredential, error) {
	previous, err := o.TokenStore.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: load the saved OpenAI session: %w", err)
	}
	if previous != nil && (!previous.PlanUsage() || previous.ClientID == "") {
		previous = nil
	}
	instruction, prompt := pasteCodeInstruction, pasteCodePrompt
	if previous == nil {
		use, err := p.Confirm(fmt.Sprintf("You're using your ChatGPT plan: this app's requests count toward its usage, "+
			"which you can manage at %s. Continue?", openai.ChatGPTUsageURL), true)
		if err != nil {
			return resolvedCredential{}, err
		}
		if !use {
			return resolvedCredential{}, errors.New("wizard: ChatGPT plan sign-in declined")
		}
		// A first sign-in learns the app's client from the callback URL, so
		// a bare code cannot complete it (0043-MADR A3).
		instruction, prompt = pasteURLInstruction, pasteURLPrompt
	}
	flow, drain := browserFlowOptionsFor(p, o, instruction, prompt)
	session, loginErr := loginChatGPTPlan(ctx, auth.ChatGPTPlanOptions{AgentName: o.AgentName, Previous: previous,
		HTTPClient: o.HTTPClient, OpenURL: flow.OpenURL, InputCode: flow.InputCode})
	drain()
	if loginErr != nil {
		return resolvedCredential{}, loginErr
	}
	return pendingOAuthCredential(o.TokenStore, llmprovider.ProviderOpenAI, session)
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
// refresh and no expiry: it is due to the TokenStore, its only copy, which
// credentialStep saves it to once the listing has not refused it (0059-MADR
// W2), and the session is the Kilo credential (0016-MADR D11, A7). When the
// account belongs to organizations, the user chooses one, or the personal
// account; the one Existing names is the default. The profile is read from
// the endpoint the user entered, with the caller's client (0020-MADR F19,
// F20).
func resolveKiloDevice(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	baseURL string,
	o Options,
) (resolvedCredential, error) {
	session, err := deviceLogin(ctx, d.ID, oauthFlowOptions(o), showDeviceCode(p, o))
	if err != nil {
		return resolvedCredential{}, err
	}
	if session == nil || session.Access == "" {
		return resolvedCredential{}, errors.New("wizard: Kilo login returned no token")
	}
	session.Provider = d.ID
	cred := oauthCredential(session)
	cred.saveTo = o.TokenStore
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
		p.Notify(LevelWarn, "could not read the Kilo account's organizations (%s); using the personal account", describe(err))
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
		// The profile is the service's text: a menu shows it without the
		// characters a terminal acts on (0066-MADR V1).
		choices = append(choices, Choice{Label: redact.StripControl(org.Name), Detail: redact.StripControl(org.Role)})
		ids = append(ids, org.ID)
	}
	idx, err := choose(p, "Use Kilo as:", choices, defaultIdx)
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("select Kilo organization: %w", err)
	}
	cred.organization = ids[idx]
	return cred, nil
}

// resolveKiloDeviceKey runs Kilo's device login and takes its token as the
// key, under Options.KiloDeviceKey: nothing is saved and no profile is read,
// so no organization is chosen. One copy of the token, in the Result
// (0056-MADR D2; 0016-MADR D11).
func resolveKiloDeviceKey(ctx context.Context, p Prompter, d llmprovider.Descriptor, o Options) (resolvedCredential, error) {
	session, err := deviceLogin(ctx, d.ID, oauthFlowOptions(o), showDeviceCode(p, o))
	if err != nil {
		return resolvedCredential{}, err
	}
	if session == nil || session.Access == "" {
		return resolvedCredential{}, errors.New("wizard: Kilo login returned no token")
	}
	cred := staticCredential(CredAPIKey, session.Access)
	cred.keyFrom = keyDevice
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
// there, the user signs in again. The session is checked before it is
// offered: one that cannot be used is not, a notice says why, and the method
// menu follows (0059-MADR W6).
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
	session.Store = o.TokenStore
	// The kept session refreshes through the caller's client, as the
	// listing and a new sign-in do (0016-MADR D8; 0026-MADR F10). A nil
	// client changes nothing.
	session.UseHTTPClient(o.HTTPClient)
	if err := validateKept(d.ID, session); err != nil {
		p.Notify(LevelWarn, "the saved %s session cannot be used (%v); sign in again", d.Label, err)
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
	_ context.Context,
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
			return pendingAccessOnlyOpenAI(o, value)
		}
	}
	return staticCredential(CredAPIKey, value), nil
}

// foreignKeyPrefixes are the vendors' key prefixes, checked in order: sk-ant-
// and sk-or- before OpenAI's own sk- (0021-MADR Z10; 0028-PLAN D3). Each
// names the providers whose key it is, so that a key is named only when it
// is another provider's (0059-MADR S1).
var foreignKeyPrefixes = []struct {
	prefix, vendor string
	owners         []llmprovider.ProviderID
}{
	{"sk-ant-", "an Anthropic", []llmprovider.ProviderID{llmprovider.ProviderClaude}},
	{"sk-or-", "an OpenRouter", nil},
	{"xai-", "an xAI", []llmprovider.ProviderID{llmprovider.ProviderGrok}},
	{"tgp_", "a Together", []llmprovider.ProviderID{llmprovider.ProviderTogether}},
	{"hf_", "a Hugging Face", []llmprovider.ProviderID{llmprovider.ProviderHuggingFace}},
	{"api_org_", "a Hugging Face", []llmprovider.ProviderID{llmprovider.ProviderHuggingFace}},
	{"AIza", "a Google", []llmprovider.ProviderID{llmprovider.ProviderGemini}},
	{"AQ.", "a Google Gemini", []llmprovider.ProviderID{llmprovider.ProviderGemini}},
}

// openCodeOwners are the providers an OpenCode key is for.
var openCodeOwners = []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo}

// keyVendor names the vendor whose key value looks like, with its article,
// and the providers that key is for; "" when it looks like no listed
// vendor's. An OpenCode key is sk- like OpenAI's, and is told apart by its
// shape: 64 letters and digits, with no T3BlbkFJ.
func keyVendor(value string) (string, []llmprovider.ProviderID) {
	for _, k := range foreignKeyPrefixes {
		if strings.HasPrefix(value, k.prefix) {
			return k.vendor, k.owners
		}
	}
	if reOpenCodeKey.MatchString(value) {
		return "an OpenCode", openCodeOwners
	}
	return "", nil
}

// foreignKeyVendor names the vendor whose key value looks like, with its
// article, or "".
func foreignKeyVendor(value string) string {
	vendor, _ := keyVendor(value)
	return vendor
}

// foreignKeyFor names the vendor whose key value looks like when that key is
// not provider's own, with its article, or "" (0059-MADR S1).
func foreignKeyFor(provider llmprovider.ProviderID, value string) string {
	vendor, owners := keyVendor(value)
	if vendor == "" || slices.Contains(owners, provider) {
		return ""
	}
	return vendor
}

// reOpenCodeKey is an OpenCode key: sk- and 64 letters and digits (opencode
// packages/console/core/src/key.ts; 0028-PLAN D3).
var reOpenCodeKey = regexp.MustCompile(`^sk-[A-Za-z0-9]{64}$`)

// keyShapes are the confirmed shapes of each provider's API key, from the
// vendors' code and documentation and the secret scanners' rules, checked
// against the owner's keys by shape only (0028-MADR H10b; 0028-PLAN D3).
// xAI's length is not confirmed, so only its prefix is checked; nor are
// Anthropic's tag and length, which working keys show to vary (api03, usr),
// so only its prefix and a tag are (0059-MADR W14). Kilo's key is
// a JWT and Ollama's has no published shape, so neither is checked. No vendor
// publishes a full format, and a key the provider accepts may not have its
// shape, so a shape is named only once the provider has refused a typed key
// (0059-MADR S1).
var keyShapes = map[llmprovider.ProviderID]struct {
	re    *regexp.Regexp
	shape string
}{
	llmprovider.ProviderOpenAI: {regexp.MustCompile(
		`^sk-(?:(?:proj|svcacct|admin|None)-[A-Za-z0-9_-]*T3BlbkFJ[A-Za-z0-9_-]+|[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20})$`),
		"sk-proj-, sk-svcacct- or sk-admin-, holding T3BlbkFJ"},
	llmprovider.ProviderClaude: {regexp.MustCompile(`^sk-ant-[A-Za-z0-9]+-`),
		"sk-ant- and a tag, such as sk-ant-api03- or sk-ant-usr-"},
	llmprovider.ProviderGemini: {regexp.MustCompile(`^(?:AIza[0-9A-Za-z_-]{35}|AQ\.[0-9A-Za-z_.-]+)$`),
		"AIza and 35 characters, or an auth key beginning AQ."},
	llmprovider.ProviderTogether:    {regexp.MustCompile(`^tgp_v1_[A-Za-z0-9_-]{43}$`), "tgp_v1_ and 43 characters"},
	llmprovider.ProviderHuggingFace: {regexp.MustCompile(`^hf_[A-Za-z0-9]{34}$`), "hf_ and 34 letters and digits"},
	llmprovider.ProviderOpencodeZen: {reOpenCodeKey, "sk- and 64 letters and digits"},
	llmprovider.ProviderOpencodeGo:  {reOpenCodeKey, "sk- and 64 letters and digits"},
	llmprovider.ProviderGrok:        {regexp.MustCompile(`^xai-`), "xai- and letters and digits"},
}

// warnForeignKey warns when a typed key is plainly another provider's,
// naming the vendor, never the key; the run goes on with it. Only a run that
// lists its models says the listing will check it (0059-MADR S1, W3).
func warnForeignKey(p Prompter, d llmprovider.Descriptor, key string, o Options) {
	vendor := foreignKeyFor(d.ID, key)
	if vendor == "" {
		return
	}
	check := ""
	if o.Discover {
		check = "; the model listing will check it"
	}
	p.Notify(LevelWarn, "this looks like %s key, not one for %s%s", vendor, d.Label, check)
}

// notifyShape tells the user, once the provider has refused a typed key, that
// the key does not have d's confirmed shape, naming the shape, never the key.
// A key with the shape, or a provider with none, gets nothing (0059-MADR S1).
func notifyShape(p Prompter, d llmprovider.Descriptor, key string) {
	s, ok := keyShapes[d.ID]
	if !ok || key == "" || s.re.MatchString(key) {
		return
	}
	p.Notify(LevelWarn, "%s API keys look like %s; this one does not", d.Label, s.shape)
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

// pendingAccessOnlyOpenAI is the session of a pasted ChatGPT access token.
// It keeps the caller's client, as a sign-in's session does (Options.HTTPClient;
// 0059-MADR W11).
func pendingAccessOnlyOpenAI(o Options, access string) (resolvedCredential, error) {
	session := accessOnlyOpenAISession(access)
	session.UseHTTPClient(o.HTTPClient)
	return pendingOAuthCredential(o.TokenStore, llmprovider.ProviderOpenAI, session)
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

// oauthFlowOptions is a sign-in's options: the caller's client (0020-MADR
// F20). It sets no NotifyDevice: a device sign-in shows its code through
// showDeviceCode (0057-MADR D1).
func oauthFlowOptions(o Options) auth.OAuthFlowOptions {
	return auth.OAuthFlowOptions{HTTPClient: o.HTTPClient}
}

// The paste instruction and prompt, and a ChatGPT plan's first sign-in's,
// which needs the URL (0043-MADR A3).
const (
	pasteCodeInstruction = "If the browser does not return here, paste the redirected URL or the authorization code"
	pasteCodePrompt      = "Paste the redirected URL or authorization code if the browser does not return"
	pasteURLInstruction  = "If the browser does not return here, paste the redirected URL"
	pasteURLPrompt       = "Paste the redirected URL if the browser does not return"
)

// browserFlowOptions adds MADR 0008 D3's paste-code race to oauthFlowOptions.
// The authorize URL and the paste instruction are shown even when the consumer
// opens the browser itself. The returned drain must run once the login
// returns: it finishes a paste prompt the loopback overtook, so no later
// prompt reads alongside it. Under Options.NoPastePrompt there is no race:
// the URL is shown, and the loopback alone completes the sign-in.
func browserFlowOptions(p Prompter, o Options) (auth.OAuthFlowOptions, func()) {
	return browserFlowOptionsFor(p, o, pasteCodeInstruction, pasteCodePrompt)
}

// browserFlowOptionsFor is browserFlowOptions with the paste instruction and
// prompt given.
func browserFlowOptionsFor(p Prompter, o Options, instruction, prompt string) (auth.OAuthFlowOptions, func()) {
	flow := oauthFlowOptions(o)
	if o.NoPastePrompt {
		// No InputCode: nothing reads the Prompter, so nothing is drained
		// (0057-MADR D2).
		flow.OpenURL = func(authorizeURL string) error {
			p.Notify(LevelInfo, "Open %s in your browser", authorizeURL)
			if o.OpenURL == nil {
				return nil
			}
			return o.OpenURL(authorizeURL)
		}
		return flow, func() {}
	}
	paste := &pastePrompt{p: p, shown: make(chan struct{}), instruction: instruction, prompt: prompt}
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
	// instruction is shown with the URL; prompt asks for the paste.
	instruction, prompt string

	mu      sync.Mutex
	done    bool               // drain ran: nothing further may use the prompter
	pending chan struct{}      // closed when the started read returns
	cancel  context.CancelFunc // ends the started read's context
}

// drainGrace is how long drain waits, silently, for an InputerContext read
// to give up before it asks for the Enter (0057-MADR D3).
var drainGrace = 2 * time.Second

// pasteQuick is how soon an empty answer comes when it is the prompter's, not
// a person's: one that answers at once with nothing would otherwise be asked
// without end. pasteQuickEmpties of them in a row stop the asking (0059-MADR
// W5).
const (
	pasteQuick        = 50 * time.Millisecond
	pasteQuickEmpties = 3
)

func (s *pastePrompt) show(authorizeURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return
	}
	s.p.Notify(LevelInfo, "Open %s in your browser", authorizeURL)
	s.p.Notify(LevelInfo, "%s", s.instruction)
	close(s.shown)
}

// input reads the paste, asking again while the answer is empty: an Enter
// alone is no paste, and the loopback may still return (0057-MADR D4). Once
// drain has run, no read is started, so the Enter drain asks for ends it.
// After pasteQuickEmpties empty answers, each within pasteQuick, the prompter
// is not asked again: input says so and waits for ctx, which the loopback or
// the login's limit ends. A slower empty answer is a person's Enter, and
// starts the count again (0059-MADR W5).
func (s *pastePrompt) input(ctx context.Context) (string, error) {
	select {
	case <-s.shown:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	quick := 0
	for {
		start := time.Now()
		answer, err := s.read(ctx)
		if err != nil || strings.TrimSpace(answer) != "" {
			return answer, err
		}
		if time.Since(start) >= pasteQuick {
			quick = 0
			continue
		}
		quick++
		if quick < pasteQuickEmpties {
			continue
		}
		s.mu.Lock()
		if !s.done {
			s.p.Notify(LevelWarn, "the prompter answers at once; waiting for the browser")
		}
		s.mu.Unlock()
		<-ctx.Done()
		return "", ctx.Err()
	}
}

// read is one read of the paste, unless drain has run or ctx has ended. A
// Prompter that can give up a read is given a context of the wizard's own,
// which drain ends, so the wizard does not rely on the login's own cancel
// (0057-MADR D3).
func (s *pastePrompt) read(ctx context.Context) (string, error) {
	s.mu.Lock()
	if s.done || ctx.Err() != nil {
		s.mu.Unlock()
		return "", context.Canceled
	}
	readCtx, cancel := context.WithCancel(ctx)
	pending := make(chan struct{})
	s.pending, s.cancel = pending, cancel
	s.mu.Unlock()
	defer close(pending)
	defer cancel()
	if ic, ok := s.p.(InputerContext); ok {
		return ic.InputContext(readCtx, s.prompt, "")
	}
	return s.p.Input(s.prompt, "")
}

// drain finishes a read the loopback overtook, before any other prompt. An
// InputerContext's read is ended, and waited for in silence; one that has not
// returned within drainGrace, like any other Prompter's read, is finished by
// the Enter the notice asks for (0008-MADR's amendment).
func (s *pastePrompt) drain() {
	s.mu.Lock()
	s.done = true
	pending, cancel := s.pending, s.cancel
	s.mu.Unlock()
	if pending == nil {
		return
	}
	cancel()
	if _, ok := s.p.(InputerContext); ok {
		select {
		case <-pending:
			return
		case <-time.After(drainGrace):
		}
	}
	select {
	case <-pending:
	default:
		s.p.Notify(LevelInfo, "Browser sign-in finished; press Enter to continue")
		<-pending
	}
}

// pendingOAuthCredential checks a new session and names it provider's. It is
// not saved here: credentialStep saves it to store once the listing has not
// refused it (0059-MADR W2).
func pendingOAuthCredential(
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
	cred := oauthCredential(session)
	cred.saveTo = store
	return cred, nil
}

// save writes a new session to the store it is due to; a kept session, or a
// credential that is not a session, has none (0059-MADR W2). It first waits
// for a refresh the listing left running, past its limit, so that the store
// gets the rotated tokens, not the spent ones. A session the issuer refuses
// is not saved.
func (c resolvedCredential) save(ctx context.Context, d llmprovider.Descriptor) error {
	if c.saveTo == nil || c.session == nil {
		return nil
	}
	if _, err := c.session.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure) || ctx.Err() != nil) {
		return fmt.Errorf("wizard: the new %s session was not saved: %w", d.Label, err)
	}
	c.session.Store = c.saveTo
	if err := c.saveTo.Save(ctx, d.ID, c.session); err != nil {
		if d.ID == llmprovider.ProviderKilo {
			return fmt.Errorf("wizard: save Kilo login: %w", err)
		}
		return fmt.Errorf("wizard: save OAuth session: %w", err)
	}
	return nil
}
