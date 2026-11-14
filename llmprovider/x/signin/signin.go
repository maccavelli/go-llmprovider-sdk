package signin

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Options configures Start. Its zero value is a sign-in with this module's
// own identity and no change to any default. It is safe for concurrent use
// while no goroutine changes it.
type Options struct {
	// HTTPClient sends the sign-in's requests; nil is the default client.
	HTTPClient *http.Client
	// ClientID is OpenAI's or Grok's OAuth client; empty is the default.
	// Kilo and the plan ignore it.
	ClientID string
	// Issuer is OpenAI's or Grok's issuer, or Kilo's origin; empty is the
	// default. The plan ignores it.
	Issuer string
	// Identity names the application on the sign-in's requests; the zero
	// value names this module (0076-MADR D3).
	Identity auth.Identity
	// AgentName is the app's own name, which chatgpt_plan requires and shows
	// the person; the other methods ignore it.
	AgentName string
	// Previous is a plan session to sign in to again; chatgpt_plan only. A
	// first sign-in, with none, takes the method's FirstPaste.
	Previous *auth.OAuthSession
	// HostID is the app's host-lifetime id for the chatgpt_plan sign-in, as
	// OpenAI's guide asks one kept for the host's life; the app persists it.
	// Without it the id is Previous's, else the stored client record's, else
	// a new one the client record then keeps (0075-MADR D6).
	HostID string
	// ClientStore, when set, keeps the chatgpt_plan sign-in's issued client
	// and host id apart from the session, so that a sign-out or a switch to
	// another OpenAI session reuses the client instead of registering another
	// (0075-MADR D7). Before a sign-in with no Previous it loads the record,
	// and once the sign-in returns it saves the client and host id, whatever
	// the caller then does with the session; a record that cannot be saved
	// does not fail the sign-in. Nil keeps none.
	ClientStore auth.ClientStore
	// ConsentText maps a consent id to the app's own wording for it, which
	// the sign-in's confirm step shows in place of the method's default
	// text; a consent with no entry uses the default (0075-MADR D4).
	ConsentText map[string]string
	// NoPaste leaves out the paste step: the loopback alone completes a
	// browser sign-in.
	NoPaste bool
	// NoLoopback binds no listener: a paste alone completes a browser
	// sign-in, which it passes to auth's NoLoopback fields (0074-MADR D4).
	// A device sign-in ignores it, and NoLoopback with NoPaste is an error
	// matching llmprovider.ErrInvalidRequest.
	NoLoopback bool
	// LookupEnv reads the environment for import_vendor_cli's paths, such as
	// os.Getenv; nil finds none (0015-MADR D9).
	LookupEnv func(string) string
}

// drive is how Start runs one (provider, method) pair: the login to call and
// the method's facts, which the engine's steps follow.
type drive struct {
	mode   signinflow.Mode
	method llmprovider.AuthMethod
}

type pair struct {
	provider llmprovider.ProviderID
	method   llmprovider.AuthMethodID
}

// The facts below repeat the built-in descriptors' for the pairs Start
// drives, because this package may import neither the providers nor a
// Registry (R2): a test holds each row equal to its descriptor's.
const (
	factOAuthSession = string(authcontract.YieldsOAuthSession)
	factVendor       = string(authcontract.ClientVendorBorrowed)
	factNone         = string(authcontract.ClientNone)
	factURLOrCode    = string(authcontract.PasteURLOrCode)
	factBrowser      = string(authcontract.KindBrowser)
	factPaste        = string(authcontract.CompletionBrowserPaste)
	factDeviceKind   = string(authcontract.KindDeviceCode)
	factDevice       = string(authcontract.CompletionDevice)
	factImportKind   = string(authcontract.KindVendorImport)
	factLocalFile    = string(authcontract.CompletionLocalFile)
	factReadThrough  = string(authcontract.YieldsReadThrough)
	factConsent      = "chatgpt_plan.background_use"
	factConsentURL   = "https://chatgpt.com/settings/usage"
	factConsentText  = "You're using your ChatGPT plan: this app's requests count toward its usage, " +
		"which you can manage on the usage page. " +
		"Allow this app to also use your plan for background requests it starts on its own?"
)

// browserFacts, deviceFacts and importFacts build the rows below.
func browserFacts(label, detail, slot string) llmprovider.AuthMethod {
	return llmprovider.AuthMethod{ID: llmprovider.AuthBrowserOAuth, Label: label, Detail: detail,
		Kind: factBrowser, Completion: factPaste, Paste: factURLOrCode, Yields: factOAuthSession,
		ClientIdentity: factVendor, Slot: slot, Refreshable: true, Revocable: true}
}

func deviceFacts(label string, slot string) llmprovider.AuthMethod {
	return llmprovider.AuthMethod{ID: llmprovider.AuthDeviceCode, Label: label,
		Kind: factDeviceKind, Completion: factDevice, Yields: factOAuthSession,
		ClientIdentity: factVendor, Slot: slot, Refreshable: true, Revocable: true}
}

func importFacts(label string) llmprovider.AuthMethod {
	return llmprovider.AuthMethod{ID: llmprovider.AuthImportVendorCLI, Label: label,
		Kind: factImportKind, Completion: factLocalFile, Yields: factReadThrough, ClientIdentity: factNone}
}

// driven is every pair Start runs (0074-MADR D3).
var driven = map[pair]drive{
	{llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth}: {signinflow.ModeBrowser,
		browserFacts("Sign in with ChatGPT", "Plus/Pro/Business/Edu/Enterprise plan", "openai")},
	{llmprovider.ProviderOpenAI, llmprovider.AuthDeviceCode}: {signinflow.ModeDevice,
		deviceFacts("Sign in with ChatGPT (device code)", "openai")},
	{llmprovider.ProviderOpenAI, llmprovider.AuthImportVendorCLI}: {signinflow.ModeImport,
		importFacts("Use the Codex CLI login (~/.codex/auth.json)")},
	{llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan}: {signinflow.ModePlan, llmprovider.AuthMethod{
		ID: llmprovider.AuthChatGPTPlan, Label: "Use your ChatGPT plan with this app",
		Detail: "Sign in with ChatGPT for agents; usage counts toward your plan",
		Kind:   factBrowser, Completion: factPaste, Paste: factURLOrCode,
		FirstPaste: string(authcontract.PasteURLOnly), Yields: factOAuthSession,
		ClientIdentity: string(authcontract.ClientAppRegistered), Slot: "openai", Refreshable: true, Revocable: true,
		Consent: factConsent, ConsentText: factConsentText, ConsentURL: factConsentURL}},
	{llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth}: {signinflow.ModeBrowser,
		browserFacts("Sign in with xAI", "", "grok")},
	{llmprovider.ProviderGrok, llmprovider.AuthDeviceCode}: {signinflow.ModeDevice,
		deviceFacts("Sign in with xAI (device code)", "grok")},
	{llmprovider.ProviderGrok, llmprovider.AuthImportVendorCLI}: {signinflow.ModeImport,
		importFacts("Use the Grok CLI login (~/.grok/auth.json)")},
	{llmprovider.ProviderKilo, llmprovider.AuthDeviceCode}: {signinflow.ModeDevice, llmprovider.AuthMethod{
		ID: llmprovider.AuthDeviceCode, Label: "Sign in with Kilo (device code)",
		Kind: factDeviceKind, Completion: factDevice, Yields: string(authcontract.YieldsDeviceKey),
		ClientIdentity: factNone, Slot: "kilo"}},
}

// methodFor returns the contract's description of a pair Start drives.
func methodFor(p llmprovider.ProviderID, m llmprovider.AuthMethodID) (authcontract.Method, bool) {
	d, ok := driven[pair{p, m}]
	if !ok {
		return authcontract.Method{}, false
	}
	return methodOf(d.method), true
}

// SignIn is a sign-in in progress. Read Steps, answer the steps that ask
// with Answer, and call Wait for the credential, or Cancel to end it. It is
// safe for concurrent use: Answer, Cancel and Wait may run at once.
type SignIn struct {
	run    *signinflow.Run
	method authcontract.Method
}

// Start begins the sign-in of method m for provider p. It drives
// device_code (OpenAI, Grok and Kilo), browser_oauth (OpenAI and Grok),
// chatgpt_plan, and import_vendor_cli (OpenAI and Grok). A value method
// (api_key, token_stdin) and any other pair give an error matching
// llmprovider.ErrUnsupported: the caller reads the value itself.
//
// With o.NoLoopback, a browser method that takes no paste gives an error
// matching llmprovider.ErrUnsupported, and o.NoPaste as well one matching
// llmprovider.ErrInvalidRequest; chatgpt_plan without o.AgentName gives one
// matching llmprovider.ErrInvalidRequest too. Each is returned before any
// request. Every later failure is the sign-in's: its last step is failed
// and Wait returns its error, which ErrorKindOf names. ctx bounds the
// sign-in; Cancel ends it sooner.
func Start(ctx context.Context, p llmprovider.ProviderID, m llmprovider.AuthMethodID, o Options) (*SignIn, error) {
	return startWith(ctx, p, m, o, signinflow.Logins{})
}

// startWith is Start with the login functions a test replaces.
func startWith(ctx context.Context, p llmprovider.ProviderID, m llmprovider.AuthMethodID, o Options, logins signinflow.Logins) (*SignIn, error) {
	d, ok := driven[pair{p, m}]
	if !ok {
		return nil, fmt.Errorf("%w: signin: %s's %q is not a sign-in this package drives; read the value itself",
			llmprovider.ErrUnsupported, p, m)
	}
	method := methodOf(d.method)
	spec := signinflow.Spec{
		Provider:    p,
		Method:      method,
		Mode:        d.mode,
		First:       o.Previous == nil,
		NoPaste:     o.NoPaste,
		NoLoopback:  o.NoLoopback,
		ConsentText: o.ConsentText,
		Logins:      logins,
		Flow: auth.OAuthFlowOptions{HTTPClient: o.HTTPClient, ClientID: o.ClientID,
			Issuer: o.Issuer, Identity: o.Identity},
		ClientStore: o.ClientStore,
		Plan: auth.ChatGPTPlanOptions{AgentName: o.AgentName, Previous: o.Previous, HostID: o.HostID,
			HTTPClient: o.HTTPClient, Identity: o.Identity},
	}
	if d.mode == signinflow.ModePlan && strings.TrimSpace(o.AgentName) == "" {
		return nil, fmt.Errorf("%w: signin: chatgpt_plan needs the app's own name (Options.AgentName)",
			llmprovider.ErrInvalidRequest)
	}
	if d.mode == signinflow.ModeImport {
		spec.Import = func(ctx context.Context) (*auth.VendorCLISession, error) {
			return importVendor(ctx, p, o.LookupEnv)
		}
	}
	run, err := signinflow.Start(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &SignIn{run: run, method: method}, nil
}

// importVendor checks the vendor CLI's auth file holds a live token and
// returns the session that reads it.
func importVendor(ctx context.Context, p llmprovider.ProviderID, lookupEnv func(string) string) (*auth.VendorCLISession, error) {
	path, err := auth.VendorCLIPath(p, lookupEnv)
	if err != nil {
		return nil, err
	}
	session := &auth.VendorCLISession{Provider: p, Path: path}
	if _, err := session.Token(ctx); err != nil {
		return nil, err
	}
	return session, nil
}

// Method returns the method's facts, as Methods describes the built-in
// descriptor's.
func (s *SignIn) Method() authcontract.Method { return s.method }

// Steps returns the sign-in's steps, in order, numbered from 1 by Seq. The
// channel is buffered for the whole sign-in, so a slow reader never blocks
// it, and is closed after the done or failed step.
func (s *SignIn) Steps() <-chan authcontract.Step { return s.run.Steps() }

// Answer gives the answer to the step whose Seq it names. Only a paste,
// confirm or form step asks; an answer to any other step, to a step already
// answered, to another Seq, or after the end gives an error matching
// llmprovider.ErrInvalidRequest. The sign-in sets the answer's Secret from
// its own step, whatever the caller sent. A paste that is not usable brings
// a new paste step, and a third bad one ends the sign-in (0074-MADR D4).
func (s *SignIn) Answer(ctx context.Context, a authcontract.Answer) error {
	return s.run.Answer(ctx, a)
}

// Wait returns the credential the sign-in yields, or its error, which
// ErrorKindOf names; a Cancel gives an error matching context.Canceled. It
// saves nothing: the caller saves. It returns ctx's error when ctx ends first.
func (s *SignIn) Wait(ctx context.Context) (Credential, error) {
	result, err := s.run.Wait(ctx)
	if err != nil {
		return nil, err
	}
	if s.method.Yields == authcontract.YieldsReadThrough {
		return ReadThrough{Source: result.Import}, nil
	}
	if result.Session == nil || result.Session.Access == "" {
		return nil, fmt.Errorf("signin: the %s sign-in returned no token", s.method.ID)
	}
	switch s.method.Yields {
	case authcontract.YieldsDeviceKey:
		return DeviceKey{Key: result.Session.Access}, nil
	case authcontract.YieldsAccessOnly:
		return AccessOnly{OAuth: result.Session}, nil
	default:
		return Session{OAuth: result.Session}, nil
	}
}

// Cancel ends the sign-in: its last step is failed with kind canceled, and
// Wait's error matches context.Canceled. It does nothing after the end.
func (s *SignIn) Cancel() { s.run.Cancel() }
