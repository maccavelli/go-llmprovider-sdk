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

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
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
	// saveTo is the store a new session is due to, at the run's end, not the
	// listing's: commitCredential saves it there, right after the check for
	// ResolveCredential and as ConfigureLLM's last step, once the check has not
	// refused it and the run's context has not ended (0059-MADR W2; 0080-MADR
	// D3; 0081-MADR D5).
	saveTo auth.TokenStore
	// kept is the saved session keepExistingOAuth kept: a listing that
	// refuses it leads to the sign-in menu (0066-MADR V7, Q4 (a)).
	kept bool
	// profileRead is set when resolveKiloDevice read Kilo's profile under
	// Options.Discover: that read is the credential's check, and profileErr
	// is its failure, nil when the profile answered (0080-MADR D3).
	profileRead bool
	profileErr  error
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
// Phase 1's deviation.) offerKept is false after the check refused the
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
		return resolveChatGPTPlan(ctx, p, d, o)
	}
	switch method {
	case llmprovider.AuthBrowserOAuth:
		session, loginErr := browserSignIn(ctx, p, d.ID, o, signinflow.Spec{
			Method: authcontract.Method{ID: string(method), Paste: authcontract.PasteURLOrCode,
				Yields: authcontract.YieldsOAuthSession},
			Mode: signinflow.ModeBrowser,
		}, pasteCodeInstruction, pasteCodePrompt)
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
		session, loginErr := deviceSignIn(ctx, p, d.ID, o)
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
// the consent that the plan's sign-in asks, as one question with three
// choices, its text the notice "You're using your ChatGPT plan" and the page
// where the plan's usage is managed (0075-MADR D4). A plan session that holds
// an answer is signed in to again without it. The new session replaces
// whatever OpenAI session the store held (D3).
func resolveChatGPTPlan(ctx context.Context, p Prompter, d llmprovider.Descriptor, o Options) (resolvedCredential, error) {
	previous, err := o.TokenStore.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		return resolvedCredential{}, fmt.Errorf("wizard: load the saved OpenAI session: %w", err)
	}
	// A TokenStore that also keeps the issued client apart from the session
	// has it reused after a sign-out or a switch (0075-MADR D7).
	var clients auth.ClientStore
	var issued auth.ClientRecord
	var read *signinflow.ClientRead
	var unreadable bool
	if store, ok := o.TokenStore.(auth.ClientStore); ok {
		clients = store
		// The record is read once, here, and the read is given to the engine.
		// Only a record that exists but cannot be decoded is no record, and the
		// person is told, once: the sign-in goes on, and the client it registers
		// replaces the record. With no record to say whose a stored plan session
		// is, none is handed over as Previous: the person registers a new client,
		// which is the ruled behaviour, not the reuse of 0075-MADR D7. The
		// record's content, and the read error's text, which quotes some of it,
		// are never shown. Any other read error, a permission refused, an input
		// or output error, fails the sign-in at once, naming the store's error
		// and not the file's content, with no notice and no registration: the
		// record is as it was (0082-MADR, third amendment of 2026-11-17, R-9; the
		// second's R-6; the fourth of 2026-11-18, R-12).
		var readErr error
		if issued, readErr = store.LoadClient(ctx, llmprovider.ProviderOpenAI); readErr != nil {
			if !signinflow.Undecodable(readErr) {
				return resolvedCredential{}, fmt.Errorf("wizard: read the saved %s client record: %w", d.Label, readErr)
			}
			issued, unreadable = auth.ClientRecord{}, true
			if previous != nil && previous.PlanUsage() {
				p.Notify(LevelWarn, "the saved %s client record could not be read; the saved plan session is not signed in to again, and a new client is registered", d.Label)
			} else {
				p.Notify(LevelWarn, "the saved %s client record could not be read; this sign-in replaces it", d.Label)
			}
		}
		read = &signinflow.ClientRead{Record: issued, Err: readErr}
	}
	if previous != nil && (unreadable || !signsInAgainTo(previous, issued)) {
		previous = nil
	}
	instruction, prompt := pasteCodeInstruction, pasteCodePrompt
	if previous == nil {
		// A first sign-in learns the app's client from the callback URL, so
		// a bare code cannot complete it (0043-MADR A3).
		instruction, prompt = pasteURLInstruction, pasteURLPrompt
	}
	session, loginErr := browserSignIn(ctx, p, llmprovider.ProviderOpenAI, o, signinflow.Spec{
		Method: planMethod(d),
		Mode:   signinflow.ModePlan,
		Plan: auth.ChatGPTPlanOptions{AgentName: o.AgentName, Previous: previous, HTTPClient: o.HTTPClient,
			Identity: o.Identity, HostID: o.HostID},
		First:       previous == nil,
		ConsentText: o.ConsentText,
		ClientStore: clients,
		Client:      read,
	}, instruction, prompt)
	if loginErr != nil {
		return resolvedCredential{}, loginErr
	}
	return pendingOAuthCredential(o.TokenStore, llmprovider.ProviderOpenAI, session)
}

// signsInAgainTo reports whether the stored session previous is the plan
// sign-in the wizard is about to repeat, so that its issued client, host id and
// ID token are reused (0075-MADR D3, D7) and nothing of another application's
// is: an OpenAI plan session of OpenAI's own issuer with an issued client, and,
// when the store keeps the issued client apart (issued, the zero record when it
// has none), that very client. Before the new sign-in exists its provider,
// kind and issuer are the plan method's, and its client is the one the app was
// issued; a store with no client record gives no other evidence, and its plan
// session is taken as the app's own, as before. The host id is not compared: it
// follows from the client (0082-MADR D2).
func signsInAgainTo(previous *auth.OAuthSession, issued auth.ClientRecord) bool {
	return previous.Owner() == llmprovider.ProviderOpenAI && previous.PlanUsage() &&
		strings.TrimRight(previous.Issuer, "/") == auth.DefaultOpenAIIssuer && previous.ClientID != "" &&
		(issued.ClientID == "" || previous.ClientID == issued.ClientID)
}

// planMethod is the plan method's facts for the engine: the descriptor's
// consent, or the built-in descriptor's when d lists none (0075-MADR D1).
func planMethod(d llmprovider.Descriptor) authcontract.Method {
	method := authcontract.Method{ID: string(llmprovider.AuthChatGPTPlan), Paste: authcontract.PasteURLOrCode,
		FirstPaste: authcontract.PasteURLOnly, Yields: authcontract.YieldsOAuthSession}
	var facts llmprovider.AuthMethod
	for _, descriptor := range []llmprovider.Descriptor{d, openai.Descriptor()} {
		for _, m := range descriptor.AuthMethods {
			if m.ID == llmprovider.AuthChatGPTPlan && m.Consent != "" && facts.Consent == "" {
				facts = m
			}
		}
	}
	if facts.Consent != "" {
		consent := authcontract.Consent{ID: facts.Consent, Required: true, Text: facts.ConsentText}
		if facts.ConsentURL != "" {
			consent.Links = []authcontract.Link{{Rel: authcontract.LinkUsage, URL: facts.ConsentURL}}
		}
		method.Consents = []authcontract.Consent{consent}
	}
	return method
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
// commitCredential saves it to once the profile and the listing have not
// refused it (0059-MADR W2; 0080-MADR D3; 0081-MADR D5), and the session is the Kilo
// credential (0016-MADR D11, A7). When the account belongs to organizations,
// the user chooses one, or the personal account; the one Existing names is the
// default. The profile is read from the endpoint the user entered, with the
// caller's client (0020-MADR F19, F20).
//
// That read is also the credential's check, and the only one: under
// Options.Discover it is bound by Options.DiscoverLimit and its result goes
// on the credential, for unsavedStep, which refuses a 401 as it refuses a
// listing's and does not read the profile again. Any other failure, a 403, a
// 5xx, a body that does not decode, a refused connection or a stall, is two
// notices, in this order: the warning below, that the organizations could not
// be read and the personal account is used, then the check's, as every Kilo
// credential's is (0080-MADR D6: "could not verify the Kilo key"); the
// session then goes on to be saved if the run ends well. A ctx that ended, by
// cancel or by its deadline, is told by neither, with Discover or without, and
// the run returns its error from the next check (0081-MADR S73). Without
// Discover nothing is checked, and a failure of any kind, a 401 included, is
// the warning below alone (0080-MADR D3).
func resolveKiloDevice(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	baseURL string,
	o Options,
) (resolvedCredential, error) {
	session, err := deviceSignIn(ctx, p, d.ID, o)
	if err != nil {
		return resolvedCredential{}, err
	}
	if session == nil || session.Access == "" {
		return resolvedCredential{}, errors.New("wizard: Kilo login returned no token")
	}
	session.Provider = d.ID
	cred := oauthCredential(session)
	cred.saveTo = o.TokenStore
	readCtx := ctx
	if o.Discover {
		var cancel context.CancelFunc
		readCtx, cancel = context.WithTimeout(ctx, o.discoverLimit())
		defer cancel()
	}
	account, err := kiloProfile(readCtx, session.Access, o.kiloProfileOptions(baseURL)...)
	if o.Discover {
		cred.profileRead, cred.profileErr = true, err
	}
	if err != nil {
		if ctx.Err() != nil || (o.Discover && signinflow.Refused(err)) {
			// Under Discover the read is the credential's check. A refusal is
			// told by no warning: it ends the sign-in and saves nothing
			// (0080-MADR D3). A ctx that ended is told by none either, with
			// Discover or without: the run returns its error from the next
			// check and nothing is saved (0080-MADR D6; 0081-MADR S73).
			return cred, nil
		}
		// Anything else, and every failure without Discover, is told here, as
		// it was before: the organizations could not be read, so the personal
		// account is used. Under Discover the check then tells D6's own warning
		// after this one, so a person put on the personal account is told so
		// and that the key was not verified (0080-MADR D6, amended 2026-11-17).
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
// key, under Options.KiloDeviceKey: nothing is saved and no organization is
// chosen. It reads no profile itself: under Options.Discover unsavedStep's
// check reads it once, to verify the key as any Kilo key is (0080-MADR D2).
// One copy of the token, in the Result (0056-MADR D2; 0016-MADR D11).
func resolveKiloDeviceKey(ctx context.Context, p Prompter, d llmprovider.Descriptor, o Options) (resolvedCredential, error) {
	session, err := deviceSignIn(ctx, p, d.ID, o)
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
		Kind:      auth.SessionChatGPT,
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
// F20) and identity (0076-MADR D4), and neither a client id nor an issuer.
// It sets no NotifyDevice: a device sign-in shows its code through
// showDeviceCode (0057-MADR D1).
func oauthFlowOptions(o Options) auth.OAuthFlowOptions {
	return auth.OAuthFlowOptions{HTTPClient: o.HTTPClient, Identity: o.Identity}
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

// browserSignIn runs a browser or plan sign-in on the step engine, given the
// wizard's own seams (loginBrowserOAuth, loginChatGPTPlan), and renders its
// steps: through Options.Driver, or with the notices and prompts of
// browserFlowOptionsFor. The paste step exists unless Options.NoPastePrompt
// is set. The login's options are the caller's; the engine sets OpenURL,
// InputCode and NoLoopback itself (0074-MADR D6).
func browserSignIn(
	ctx context.Context,
	p Prompter,
	provider llmprovider.ProviderID,
	o Options,
	spec signinflow.Spec,
	instruction, prompt string,
) (*auth.OAuthSession, error) {
	flow, drain := browserFlowOptionsFor(p, o, instruction, prompt)
	spec.Provider = provider
	spec.Flow = oauthFlowOptions(o)
	spec.NoPaste = o.NoPastePrompt
	spec.Logins = signinflow.Logins{Browser: loginBrowserOAuth, Plan: loginChatGPTPlan}
	r := &stepRenderer{p: p, o: o, provider: provider, open: flow.OpenURL, input: flow.InputCode, drain: drain}
	result, err := r.run(ctx, spec)
	if err != nil {
		return nil, err
	}
	if result.ClientSaveErr != nil {
		p.Notify(LevelWarn, "could not keep the issued client (%s); the next sign-in registers another",
			redact.StripControl(redact.String(result.ClientSaveErr.Error())))
	}
	return result.Session, nil
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
// not saved here: commitCredential saves it to store once the listing has not
// refused it (0059-MADR W2; 0081-MADR D5).
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
//
// A store that offers the refresh lock is saved to under it, as SetConsent
// does: a refresh of the session the store holds, in this process or another,
// holds the lock across its request and its own save, so this save waits for
// it and lands after it, and a refresh that starts after it meets the new
// sign-in (0082-MADR, second amendment of 2026-11-17, R-1, R-7). The wizard
// tries the lock once, with a context already cancelled. The lock is held
// only when that try's error wraps context.Canceled, as RefreshLocker
// documents: then it says it is waiting, on its own goroutine, and waits. The
// wait ends with ctx, saving nothing, and at the store's own bound (a minute
// for FileTokenStore) with the store's error, whatever it is, matching
// llmprovider.ErrProviderUnavailable for FileTokenStore and not for gobble's:
// the completed sign-in is then kept in memory and p is asked, in words that
// say another process is still refreshing, whether to try the save again,
// which tries and waits again. Any failed wait but the sign-in's own
// context's ending leads to that question. A try that fails for another
// reason while the sign-in's own context is alive, a lock file that cannot be
// opened, is not the lock being held: there is no notice, no wait and no
// claim of another process; p is asked in neutral words, showing the error
// with its control characters removed, whether to try again, and yes repeats
// the try (0082-MADR, fourth amendment of 2026-11-18, R-13). The error shown
// passes through redact.String, so a token-shaped string is masked; its
// newlines become spaces and its control characters go, but what remains of an
// escape sequence ("[31m") stays as harmless text, and a long path is shown
// whole; the error returned on a refusal is the store's own, whole (fifth
// amendment of 2026-11-18, N-b). A save that
// fails under a lock the wizard holds is returned as it is. A
// refusal, an answer that cannot be given, or no Prompter returns the error and
// saves nothing; the loop is the person's, with no bound of its own
// (0082-MADR, third amendment, R-10). A store that is not a RefreshLocker is
// saved to as before. The store is given to the session only under the lock, just before
// the save: a refresh of the new session in the window before would adopt the
// store's old record of the same labels, and the save would write its tokens
// (N-3). The context is checked after the lock is held, before anything is
// written: a lock that outlasts the sign-in saves nothing. This is the check
// that covers the wait (0082-MADR N-6); the caller, commitCredential, has
// already checked the context once, before the wait, for every credential
// (0081-MADR D5).
func (c resolvedCredential) save(ctx context.Context, p Prompter, d llmprovider.Descriptor) error {
	if c.saveTo == nil || c.session == nil {
		return nil
	}
	if _, err := c.session.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure) || ctx.Err() != nil) {
		return fmt.Errorf("wizard: the new %s session was not saved: %w", d.Label, err)
	}
	locker, locked := c.saveTo.(auth.RefreshLocker)
	if !locked {
		return c.write(ctx, d)
	}
	for {
		held, contended, err := c.writeUnderLock(ctx, p, locker, d)
		if held {
			return err
		}
		if ctx.Err() != nil {
			return fmt.Errorf("wizard: the new %s session was not saved: %w", d.Label, ctx.Err())
		}
		failed := fmt.Errorf("wizard: the new %s session was not saved: %w", d.Label, err)
		if p == nil {
			return failed
		}
		question := fmt.Sprintf("The lock on the saved %s session could not be taken (%s). Try saving the new sign-in again?",
			d.Label, redact.StripControl(redact.String(err.Error())))
		if contended {
			question = fmt.Sprintf("Another process is still refreshing the saved %s session. Try saving the new sign-in again?", d.Label)
		}
		again, askErr := p.Confirm(question, true)
		if askErr != nil || !again {
			return failed
		}
	}
}

// writeUnderLock takes the store's refresh lock, writes with it held and
// releases it. held reports that the lock was taken, and err is then the
// write's; with held false, err is why it was not taken, and contended
// reports that the lock was seen held by another process.
func (c resolvedCredential) writeUnderLock(ctx context.Context, p Prompter, locker auth.RefreshLocker, d llmprovider.Descriptor) (held, contended bool, err error) {
	unlock, contended, err := lockToSave(ctx, p, locker, d)
	if err != nil {
		return false, contended, err
	}
	defer unlock()
	return true, contended, c.write(ctx, d)
}

// lockToSave takes the store's refresh lock for a save: one try, with a
// context already done, as persistRotation tries. The lock is held by another
// process only when that try's error wraps context.Canceled, which is what
// auth.RefreshLocker documents for a held lock and what FileTokenStore and
// gobble's locker return; then a notice, and a wait that ends with ctx or at the
// store's bound, and contended is true. A try that fails with the sign-in's own
// context ended is that context's error. Any other failure of the try, with
// that context alive, is not the lock being held: it is returned at once,
// with contended false, for the caller to ask about in neutral words (0082-MADR,
// fourth amendment, R-13).
func lockToSave(ctx context.Context, p Prompter, locker auth.RefreshLocker, d llmprovider.Descriptor) (unlock func(), contended bool, err error) {
	once, cancel := context.WithCancel(ctx)
	cancel()
	unlock, err = locker.LockRefresh(once, d.ID)
	if err == nil {
		return unlock, false, nil
	}
	if ctx.Err() != nil || !errors.Is(err, context.Canceled) {
		return nil, false, err
	}
	if p != nil {
		p.Notify(LevelInfo, "waiting for another process to finish refreshing the saved %s session; the new sign-in is saved after it", d.Label)
	}
	unlock, err = locker.LockRefresh(ctx, d.ID)
	return unlock, true, err
}

// write gives the new session its store and saves it. The caller holds the
// store's refresh lock when the store has one.
func (c resolvedCredential) write(ctx context.Context, d llmprovider.Descriptor) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("wizard: the new %s session was not saved: %w", d.Label, err)
	}
	c.session.UseStore(c.saveTo)
	if err := c.saveTo.Save(ctx, d.ID, c.session); err != nil {
		if d.ID == llmprovider.ProviderKilo {
			return fmt.Errorf("wizard: save Kilo login: %w", err)
		}
		return fmt.Errorf("wizard: save OAuth session: %w", err)
	}
	return nil
}
