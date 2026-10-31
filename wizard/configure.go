package wizard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// defaultDiscoverLimit bounds a live model listing so a slow or unreachable
// provider cannot stall a wizard indefinitely. It equals the lister's own
// 10 s bound, which no caller deadline extends (MADR 0013 Q4).
const defaultDiscoverLimit = 10 * time.Second

// Result is what ConfigureLLM produces. It is deliberately data, not config:
// each consumer persists it in its own schema. Unifying configuration storage
// across the three wizards is a separate decision (MADR 0005, Out of scope).
//
// A CredOAuth Result holds no token: the session is in Options.TokenStore,
// its only copy, and the consumer loads it from there (0016-MADR D11, A7).
// Its String, GoString and LogValue never show APIKey; json.Marshal keeps it,
// for the consumer to persist (0016-MADR D5, A9). It is safe for
// concurrent use only while no goroutine changes it.
type Result struct {
	Provider llmprovider.ProviderID
	Kind     CredentialKind
	APIKey   string
	// TokenExpiry, Issuer, ClientID, AccountID and FedRAMP describe a
	// CredOAuth session without its tokens.
	TokenExpiry time.Time
	Issuer      string
	ClientID    string
	AccountID   string
	FedRAMP     bool
	Model       string
	BaseURL     string
	Fallbacks   []string
	// VendorAuthPath is the vendor CLI auth file a CredVendorCLI result reads
	// through; consumers persist it and build auth.VendorCLISession.
	VendorAuthPath string
	// Organization is the Kilo organization chosen after a Kilo device login;
	// pass it to the provider with kilo.WithOrganization, and to a listing
	// with catalog.WithKiloOrganization. Empty is the personal account
	// (0017-MADR D2).
	Organization string
	// Workspace is the Anthropic workspace a Claude key not scoped to one
	// runs in; pass it to the provider with claude.WithWorkspace, and to a
	// listing with catalog.WithClaudeWorkspace. Empty is a key scoped to its
	// workspace, and every other provider (0064-MADR D4).
	Workspace string
}

// Options controls the flow. The zero value runs a full interactive
// configuration over every known provider with no discovery. It is safe for
// concurrent use only while no goroutine changes it.
type Options struct {
	// Registry is where the menu's providers come from. Nil uses
	// providers.Default(), every built-in provider (0015-PLAN S8).
	Registry *llmprovider.Registry
	// Providers restricts the menu to these ids. Empty offers every
	// descriptor, which is the behaviour that keeps a wizard current when
	// this module adds a provider.
	Providers []llmprovider.ProviderID
	// Existing pre-fills the flow, enabling "keep existing key?" and
	// defaulting the model selection.
	Existing Result
	// AllowEnv offers a credential found in the provider's environment
	// variable.
	AllowEnv bool
	// Discover queries the provider's live model listing. When false, or when
	// the listing fails or is empty, the static catalog is used. A ChatGPT
	// session has no static catalog, so the user is asked for a model id.
	Discover bool
	// DiscoverLimit bounds the listing call. Zero uses defaultDiscoverLimit.
	// The lister caps every listing at 10 s, so a larger value has no effect.
	DiscoverLimit time.Duration
	// NeedFallbacks collects additional models after the primary.
	NeedFallbacks bool
	// Profile selects how the open catalogs (Kilo, OpenCode Zen and Go,
	// Hugging Face, Together) rank the recommended models. The zero value,
	// catalog.ProfileUtility, suits short frequent tasks such as commit
	// messages; catalog.ProfileCapable suits reasoning-heavy tiers.
	Profile catalog.Profile
	// ModelLimit is the most models the menu recommends, on every listing
	// path and the built-in catalog alike. Zero recommends the whole curated
	// list, with the best six first; catalog.MaxListed keeps the menu to
	// those six. Search still covers the whole listing. The wizard passes it
	// to catalog.List as catalog.WithLimit: pass it here, not in
	// ProviderOptions, which a third-party provider would refuse (0038-MADR
	// D5, A2).
	ModelLimit int
	// LookupEnv reads an environment variable, for AllowEnv, the vendor CLI
	// homes and the home directory under which a vendor CLI's auth file is
	// found by default (HOME, or USERPROFILE on Windows). Nil reads nothing,
	// so that default is not found: pass os.Getenv for the process
	// environment, or a map for a deterministic test (0015-MADR D9;
	// 0020-MADR F25).
	LookupEnv func(string) string
	// ProviderOptions are added, after the wizard's own, to every model
	// listing and to the provider it builds for a ChatGPT session. Pass
	// catalog.OptionsFromEnv() here, or llmprovider.WithoutModelMetadata()
	// to keep a listing off the network.
	ProviderOptions []llmprovider.Option
	// TokenStore persists sessions created by the browser, device-code and
	// token-paste flows. Supplying it opts in to every sign-in method; when it
	// is nil, only those that save no session are offered: the API key, the
	// vendor CLI read-through, and a pasted Grok key.
	TokenStore auth.TokenStore
	// HTTPClient is used for live model listing, the Ollama endpoint check,
	// the sign-in flows and the Kilo profile; a session a sign-in creates
	// keeps it. Nil uses the package default.
	HTTPClient *http.Client
	// OpenURL opens an OAuth authorization URL. Nil reports the URL through Prompter.
	OpenURL func(string) error
	// AgentName is the consumer's own app name, which OpenAI's Sign in with
	// ChatGPT for agents shows the user and registers the app under
	// (auth.ChatGPTPlanOptions). Keep it the same across installations.
	// Without it, that method (llmprovider.AuthChatGPTPlan) is not offered:
	// OpenAI's terms ask an app to sign in under its own name, and this
	// module's is not the consumer's (0043-MADR D7).
	AgentName string
	// KiloDeviceKey takes Kilo's device sign-in as a key, for a consumer that
	// stores keys rather than sessions: the sign-in returns CredAPIKey with the
	// device token in APIKey, saves no session and reads no profile, so no
	// organization is asked and Organization is empty. The method is then
	// offered without a TokenStore. The token has no refresh and no expiry,
	// as a key (0056-MADR D2; 0017-MADR D2).
	KiloDeviceKey bool
	// ShowDeviceCode, when set, is given each device sign-in's code, in place
	// of the notice "Open <URL> and enter code <code>": Grok's, OpenAI's and
	// Kilo's, Kilo's as a key included. It is called once, on the flow's
	// goroutine, before the wait for approval. An error from it ends the
	// sign-in before any poll and is returned as it is. The code's Expiry is
	// never zero; for OpenAI it is this module's own wait (0057-MADR D1).
	ShowDeviceCode func(DeviceCode) error
	// SkipMethods leaves these sign-in methods out of every provider's menu,
	// after TokenStore and AgentName filter it. A provider it leaves with no
	// method, its API key included, is an error (0056-MADR D4).
	SkipMethods []llmprovider.AuthMethodID
	// NoEndpointPrompt asks no endpoint: the base URL is Existing's for that
	// provider, else the descriptor's default. A remote URL is checked as a
	// typed one is, a local one for reachability, and a failure is returned,
	// not asked again (0056-MADR D4). A plain-http URL to a remote host is
	// used without a question, since the caller holds it, with a warning
	// notice (0059-MADR W9).
	NoEndpointPrompt bool
	// NoPastePrompt starts no paste prompt in a browser sign-in, for a caller
	// with no terminal: no paste instruction is shown and the Prompter is not
	// read. The authorize URL is still shown and given to OpenURL, and the
	// browser's return to the loopback alone completes the sign-in, the
	// ChatGPT plan's included (0057-MADR D2).
	NoPastePrompt bool
}

// lookupEnv is the run's environment reader: the caller's, else one that
// finds nothing. The wizard reads no environment variable unless the caller
// passes one (0015-MADR D9, amendment "no ambient state, in detail").
func (o Options) lookupEnv() func(string) string {
	if o.LookupEnv != nil {
		return o.LookupEnv
	}
	return func(string) string { return "" }
}

// ConfigureLLM runs the canonical provider configuration flow: choose a
// provider, resolve a base URL, resolve a credential, choose a model, and
// optionally choose fallbacks. ResolveCredential runs the steps before the
// model alone.
//
// It renders nothing itself — every interaction goes through p — so a consumer
// keeps its own look and feel, and the flow is testable with a scripted
// Prompter and no TTY.
//
// It never writes configuration and never logs a credential: an API key
// appears only in the returned Result, a session only in Options.TokenStore,
// and anything shown to the user is masked with redact.MaskSecret. A
// service's error is shown as its class and HTTP status, never its text; the
// error returned keeps its text, for errors.Is (0056-MADR D3).
//
// A Prompter's error ends the flow and is returned, wrapped or as it is, so
// errors.Is matches it; so is a failed sign-in's, browser, device-code or
// ChatGPT plan, as the login gave it, and Options.ShowDeviceCode's, as it is.
// An ended ctx gives an error matching
// ctx's, before any question when it has ended already (0059-MADR W8), and a
// credential the provider refuses while its models are listed, one matching
// llmprovider.ErrAuthFailure, and a Claude workspace refused after the third
// one entered, one matching llmprovider.ErrWorkspaceRequired or
// llmprovider.ErrWorkspaceNotFound. Options.Providers naming no provider
// of the Registry, a Prompter's choice out of range, and no model chosen are
// plain errors, as are Options.SkipMethods leaving the provider no sign-in
// method and, under Options.NoEndpointPrompt, an endpoint refused or not
// reached. A listing that fails otherwise is not an error: the flow
// warns and offers the provider's static catalog.
func ConfigureLLM(ctx context.Context, p Prompter, o Options) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("wizard: %w", err)
	}
	d, err := chooseProvider(p, o)
	if err != nil {
		return Result{}, err
	}
	res, cat, err := credentialStep(ctx, p, d, o)
	if err != nil {
		return Result{}, err
	}
	if len(cat.Recommended) == 0 && len(cat.Usable) == 0 {
		// Ollama with nothing installed, or a provider whose listing failed
		// and which has no static catalog. Let the user type an id rather
		// than dead-ending the wizard.
		manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))
		if inputErr != nil {
			return Result{}, fmt.Errorf("enter model: %w", inputErr)
		}
		manual = strings.TrimSpace(manual)
		if manual == "" {
			// Returning Result{Model: ""} would hand the caller a
			// configuration that cannot generate anything.
			return Result{}, fmt.Errorf("wizard: no model available for %s and none entered", d.Label)
		}
		res.Model = manual
		return res, nil
	}

	if res.Model, err = selectModel(ctx, p, d, cat, o); err != nil {
		return Result{}, err
	}
	if o.NeedFallbacks {
		if res.Fallbacks, err = selectFallbacks(ctx, p, d, cat, res.Model, existingFallbacks(o, d.ID)); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// ResolveCredential runs ConfigureLLM's credential step alone, for a consumer
// with its own model picker: the provider (asked when id is empty, as
// ConfigureLLM asks it), its endpoint, the sign-in method and credential,
// every method included, the ChatGPT plan sign-in among them, and the listing
// that checks it, which asks again for a typed key it refuses (0056-MADR D1).
// It asks no model question, no fallback question and no model id, and ignores
// Options.Existing.Model and Options.NeedFallbacks.
//
// It returns the Result, with no Model and no Fallbacks, and the catalog the
// check listed, so the consumer need not list again. The credential is
// checked only with Options.Discover; without it, the catalog is the
// provider's static one, and may be empty.
//
// A Prompter's error is returned, wrapped or as it is, so errors.Is matches
// it; so is a failed sign-in's, and Options.ShowDeviceCode's, as it is. An
// ended ctx gives an error matching ctx's,
// and a credential the provider refuses, one matching
// llmprovider.ErrAuthFailure; a Claude workspace refused after the third one
// entered, one matching llmprovider.ErrWorkspaceRequired or
// llmprovider.ErrWorkspaceNotFound. An id the Registry lacks, or that
// Options.Providers leaves out, gives an error matching
// llmprovider.ErrInvalidProvider. A Prompter's choice out of range is a plain
// error, as are Options.SkipMethods leaving the provider no sign-in method
// and, under Options.NoEndpointPrompt, an endpoint refused or not reached.
func ResolveCredential(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) (Result, catalog.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, catalog.Catalog{}, fmt.Errorf("wizard: %w", err)
	}
	d, err := credentialProvider(p, o, id)
	if err != nil {
		return Result{}, catalog.Catalog{}, err
	}
	res, cat, err := credentialStep(ctx, p, d, o)
	if err != nil {
		return Result{}, catalog.Catalog{}, err
	}
	// A listing ctx ended is not a refusal: it falls back to the static
	// catalog, which must not pass for a checked credential.
	if err := ctx.Err(); err != nil {
		return Result{}, catalog.Catalog{}, fmt.Errorf("wizard: %w", err)
	}
	return res, cat, nil
}

// credentialProvider is ResolveCredential's provider: the one the user
// chooses when id is empty, else id's, which the run must be able to offer.
func credentialProvider(p Prompter, o Options, id llmprovider.ProviderID) (llmprovider.Descriptor, error) {
	if id == "" {
		return chooseProvider(p, o)
	}
	d, ok := o.registry().Descriptor(id)
	if !ok {
		return llmprovider.Descriptor{}, fmt.Errorf("wizard: no provider %q in the registry: %w", id, llmprovider.ErrInvalidProvider)
	}
	if len(o.Providers) > 0 && !slices.Contains(o.Providers, id) {
		return llmprovider.Descriptor{}, fmt.Errorf("wizard: provider %q is not in Options.Providers: %w", id, llmprovider.ErrInvalidProvider)
	}
	return d, nil
}

// chooseProvider asks which provider to configure, from the run's Registry as
// Options.Providers restricts it; Existing's provider is the default.
func chooseProvider(p Prompter, o Options) (llmprovider.Descriptor, error) {
	descriptors, err := selectableDescriptors(o.registry(), o.Providers)
	if err != nil {
		return llmprovider.Descriptor{}, err
	}

	choices := make([]Choice, 0, len(descriptors))
	defaultIdx := 0
	for i, d := range descriptors {
		choices = append(choices, Choice{Label: d.Label, Detail: d.Notes})
		if d.ID == o.Existing.Provider {
			defaultIdx = i
		}
	}

	idx, err := choose(p, "Choose an LLM provider:", choices, defaultIdx)
	if err != nil {
		return llmprovider.Descriptor{}, fmt.Errorf("select provider: %w", err)
	}
	return descriptors[idx], nil
}

// credentialStep resolves d's endpoint and credential, then lists its models,
// which checks the credential: a typed key the listing refuses is asked for
// again, up to maxTypedKeys (0028-MADR D-H10). Each refusal of a typed key
// without the provider's confirmed shape names the shape (0059-MADR S1). A new
// session is saved once the listing has not refused it, so a refused sign-in
// leaves the store as it was (0059-MADR W2). A Claude key's existing
// workspace is listed with first; a workspace the listing refuses is asked
// for, up to maxTypedKeys, and a refusal after that saves nothing
// (0064-MADR D4, amendment "D4's pre-fill"). The Result names no model;
// the catalog is the listing, or the static one (0056-MADR D1).
func credentialStep(ctx context.Context, p Prompter, d llmprovider.Descriptor, o Options) (Result, catalog.Catalog, error) {
	res := Result{Provider: d.ID}
	var err error
	if res.BaseURL, err = resolveBaseURL(ctx, p, d, o); err != nil {
		return Result{}, catalog.Catalog{}, err
	}
	credential, err := signIn(ctx, p, d, res.BaseURL, o)
	if err != nil {
		return Result{}, catalog.Catalog{}, err
	}
	res.Kind = credential.kind
	res.APIKey = credential.apiKey
	res.VendorAuthPath = credential.vendorPath
	res.Organization = credential.organization
	if credential.session != nil {
		res.TokenExpiry = credential.session.Expiry
		res.Issuer = credential.session.Issuer
		res.ClientID = credential.session.ClientID
		res.AccountID = credential.session.AccountID
		res.FedRAMP = credential.session.FedRAMP
	}

	if d.ID == llmprovider.ProviderClaude && o.Existing.Provider == d.ID {
		res.Workspace = o.Existing.Workspace
	}

	cat, listErr := discoverModels(ctx, p, d, res, credential.source, o)
	for keys, workspaces := 1, 0; stepRefusal(listErr); {
		if refusedWorkspace(listErr) {
			if workspaces == maxTypedKeys {
				return Result{}, catalog.Catalog{}, fmt.Errorf("wizard: %s refused %d workspaces entered: %w",
					d.Label, workspaces, listErr)
			}
			p.Notify(LevelWarn, "%s needs a workspace for this key (%s)", d.Label, describe(listErr))
			workspace, wsErr := p.Input(workspacePrompt, workspaceDefault(o, d, workspaces))
			if wsErr != nil {
				return Result{}, catalog.Catalog{}, fmt.Errorf("enter workspace: %w", wsErr)
			}
			workspaces++
			if workspace = strings.TrimSpace(workspace); strings.ContainsFunc(workspace, unicode.IsControl) {
				p.Notify(LevelWarn, "a workspace ID holds no control character")
				continue
			}
			res.Workspace = workspace
			cat, listErr = discoverModels(ctx, p, d, res, credential.source, o)
			continue
		}
		typed := credential.keyFrom == keyTyped
		if !typed || keys == maxTypedKeys {
			if typed {
				notifyShape(p, d, credential.apiKey)
			}
			return Result{}, catalog.Catalog{}, refusalError(d, credential, keys, listErr)
		}
		p.Notify(LevelWarn, "%s refused this key (%s); enter another", d.Label, describe(listErr))
		notifyShape(p, d, credential.apiKey)
		key, keyErr := promptAPIKey(p, d, o)
		if keyErr != nil {
			return Result{}, catalog.Catalog{}, keyErr
		}
		credential = staticCredential(CredAPIKey, key)
		credential.keyFrom = keyTyped
		res.APIKey = key
		keys++
		cat, listErr = discoverModels(ctx, p, d, res, credential.source, o)
	}
	if err := credential.save(ctx, d); err != nil {
		return Result{}, catalog.Catalog{}, err
	}
	return res, cat, nil
}

// registry is the run's Registry: the caller's, else every built-in provider.
func (o Options) registry() *llmprovider.Registry {
	if o.Registry != nil {
		return o.Registry
	}
	return providers.Default()
}

// selectableDescriptors returns the descriptors a run may offer, preserving
// the registry's menu order.
func selectableDescriptors(reg *llmprovider.Registry, allow []llmprovider.ProviderID) ([]llmprovider.Descriptor, error) {
	all := reg.Descriptors()
	if len(allow) == 0 {
		return all, nil
	}
	wanted := make(map[llmprovider.ProviderID]struct{}, len(allow))
	for _, id := range allow {
		wanted[id] = struct{}{}
	}
	var out []llmprovider.Descriptor
	for _, d := range all {
		if _, ok := wanted[d.ID]; ok {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("wizard: none of the requested providers exist: %v", allow)
	}
	return out, nil
}

// resolveBaseURL prompts for an endpoint when the provider supports one. A
// local provider is validated for reachability, with the caller's client
// and identity, and re-prompted on failure until ctx ends (0020-MADR F6).
func resolveBaseURL(ctx context.Context, p Prompter, d llmprovider.Descriptor, o Options) (string, error) {
	if !d.SupportsBaseURL {
		return "", nil
	}
	def := d.DefaultBaseURL
	if o.Existing.Provider == d.ID && o.Existing.BaseURL != "" {
		def = o.Existing.BaseURL
	}
	var check []llmprovider.Option
	if o.HTTPClient != nil {
		check = append(check, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	check = append(check, o.ProviderOptions...)
	if o.NoEndpointPrompt {
		// Nothing to ask again with: a refusal or an unreachable endpoint is
		// the caller's to fix (0056-MADR D4).
		if !d.IsLocal {
			return validateBaseURL(p, def, false)
		}
		if vErr := catalog.ValidateOllamaURLWith(ctx, def, check...); vErr != nil {
			return "", fmt.Errorf("wizard: cannot reach %s: %w", def, vErr)
		}
		return def, nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("enter base URL: %w", err)
		}
		url, err := p.Input(fmt.Sprintf("%s endpoint", d.Label), def)
		if err != nil {
			return "", fmt.Errorf("enter base URL: %w", err)
		}
		if !d.IsLocal {
			checked, vErr := validateBaseURL(p, url, true)
			if errors.Is(vErr, errBaseURLRefused) {
				p.Notify(LevelWarn, "%v", vErr)
				continue
			}
			return checked, vErr
		}
		vErr := catalog.ValidateOllamaURLWith(ctx, url, check...)
		if vErr == nil {
			return url, nil
		}
		p.Notify(LevelWarn, "cannot reach %s: %v", url, vErr)
		again, err := p.Confirm("Try a different endpoint?", true)
		if err != nil {
			return "", err
		}
		if !again {
			return url, nil
		}
	}
}

// maxTypedKeys is how many typed keys a run tries before it stops on a
// refusal (0028-MADR D-H10).
const maxTypedKeys = 3

// refusedCredential reports whether the listing's failure is the service
// refusing the credential: an authentication failure that is not a refusal
// to permit something (0028-MADR D-H2).
func refusedCredential(err error) bool {
	return errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted)
}

// refusedWorkspace reports whether the listing's failure is Anthropic
// refusing the workspace a key runs in, or asking for one (0064-MADR D3,
// D4).
func refusedWorkspace(err error) bool {
	return errors.Is(err, llmprovider.ErrWorkspaceRequired) || errors.Is(err, llmprovider.ErrWorkspaceNotFound)
}

// stepRefusal reports a listing failure credentialStep answers itself, by
// asking again or by returning it: a refused credential or workspace.
func stepRefusal(err error) bool { return refusedCredential(err) || refusedWorkspace(err) }

// workspacePrompt asks for a Claude key's workspace (0064-MADR D4.1), and
// envWorkspace names the variable whose value AllowEnv offers as its
// default (D4.3, Q2).
const (
	workspacePrompt = "Enter the workspace ID (wrkspc_…) from Console → Settings → Workspaces"
	envWorkspace    = "ANTHROPIC_WORKSPACE_ID"
)

// workspaceDefault is the default of the workspace prompt after asked
// earlier ones: for the first, the existing Claude workspace, else
// envWorkspace's value under AllowEnv; for a later one, none (0064-MADR
// D4.3, amendment "D4's pre-fill").
func workspaceDefault(o Options, d llmprovider.Descriptor, asked int) string {
	switch {
	case asked > 0:
		return ""
	case o.Existing.Provider == d.ID && o.Existing.Workspace != "":
		return o.Existing.Workspace
	case o.AllowEnv:
		return o.lookupEnv()(envWorkspace)
	}
	return ""
}

// refusalError names the refused credential's source, so the user knows what
// to replace; keys is how many typed keys were tried. It wraps err, whose
// text can hold the service's message: a notice shows describe(err) instead
// (0056-MADR D3).
func refusalError(d llmprovider.Descriptor, c resolvedCredential, keys int, err error) error {
	switch {
	case c.keyFrom == keyTyped:
		return fmt.Errorf("wizard: %s refused %d keys: %w", d.Label, keys, err)
	case c.keyFrom == keyEnvironment:
		return fmt.Errorf("wizard: %s refused the key in %s: %w", d.Label, d.EnvVar, err)
	case c.keyFrom == keySaved:
		return fmt.Errorf("wizard: %s refused the saved key: %w", d.Label, err)
	case c.keyFrom == keyDevice:
		return fmt.Errorf("wizard: %s refused the device sign-in; sign in again: %w", d.Label, err)
	case c.kind == CredVendorCLI:
		return fmt.Errorf("wizard: %s refused the %s CLI login in %s; sign in there again: %w",
			d.Label, d.Label, c.vendorPath, err)
	case c.kind == CredOAuth:
		return fmt.Errorf("wizard: %s refused the sign-in; sign in again: %w", d.Label, err)
	default:
		return fmt.Errorf("wizard: %s refused the pasted credential: %w", d.Label, err)
	}
}

// resolveAPIKey applies the precedence environment → existing → prompt, and
// says which gave the key. Any key shown to the user is masked; the raw value
// only ever reaches Result.
func resolveAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, keySource, error) {
	if !d.RequiresAPIKey {
		return "", keyOther, nil
	}

	if o.AllowEnv && d.EnvVar != "" {
		if envVal := o.lookupEnv()(d.EnvVar); envVal != "" {
			use, err := p.Confirm(
				fmt.Sprintf("Use %s from the environment (%s)?", d.EnvVar, redact.MaskSecret(envVal)), true)
			if err != nil {
				return "", keyOther, err
			}
			if use {
				return envVal, keyEnvironment, nil
			}
		}
	}

	if o.Existing.Provider == d.ID && o.Existing.APIKey != "" {
		keep, err := p.Confirm(
			fmt.Sprintf("Keep the existing key (%s)?", redact.MaskSecret(o.Existing.APIKey)), true)
		if err != nil {
			return "", keyOther, err
		}
		if keep {
			return o.Existing.APIKey, keySaved, nil
		}
	}

	key, err := promptAPIKey(p, d, o)
	return key, keyTyped, err
}

// promptAPIKey asks for d's API key, and warns when it is plainly another
// provider's (0059-MADR S1). A key's shape is named only once the provider
// refuses it, in credentialStep.
func promptAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, error) {
	key, err := p.Secret(fmt.Sprintf("Enter your %s API key", d.Label))
	if err != nil {
		return "", fmt.Errorf("enter API key: %w", err)
	}
	key = strings.TrimSpace(key) // a paste's spaces (0020-MADR F18)
	warnForeignKey(p, d, key, o)
	return key, nil
}

// discoverModels returns the catalog to offer: the live listing when requested
// and available, otherwise the descriptor's static catalog in both views
// (MADR 0007 §4). It also returns the listing's failure, whether the listing
// returned it or the catalog carries it, so the caller can tell a refused
// credential, which it reports itself, from an outage, reported here
// (0028-MADR D-H10).
func discoverModels(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	res Result,
	source llmprovider.TokenSource,
	o Options,
) (catalog.Catalog, error) {
	chatGPT := (res.Kind == CredOAuth || res.Kind == CredVendorCLI) && d.ID == llmprovider.ProviderOpenAI
	// A ChatGPT session lists only from the Codex backend (MADR 0008 D11):
	// the Platform catalog is not available to it, so there is no fallback.
	static := d.StaticModels
	if chatGPT {
		static = nil
	}
	// The static catalog is cut in both views, which stay equal (0038-MADR
	// D3).
	fallback := catalog.Catalog{Recommended: cutModels(static, o.ModelLimit), Usable: cutModels(static, o.ModelLimit)}
	if !o.Discover {
		return fallback, nil
	}
	limit := o.DiscoverLimit
	if limit <= 0 {
		limit = defaultDiscoverLimit
	}
	dCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()

	// One list whichever provider was chosen: Kilo's organization and
	// Claude's workspace are in their overlays (0015-MADR D5; 0064-MADR D4;
	// an empty one is the personal account, or a key scoped to its
	// workspace).
	opts := []llmprovider.Option{
		catalog.WithProfile(o.Profile),
		catalog.WithLimit(o.ModelLimit),
		llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization(res.Organization)),
		llmprovider.For(llmprovider.ProviderClaude, catalog.WithClaudeWorkspace(res.Workspace)),
	}
	if res.BaseURL != "" {
		opts = append(opts, llmprovider.WithBaseURL(res.BaseURL))
	}
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	opts = append(opts, o.ProviderOptions...)
	var cat catalog.Catalog
	var err error
	if chatGPT {
		cat, err = chatGPTCatalog(dCtx, o, d, res, source)
	} else {
		cat, err = catalog.List(dCtx, d.ID, source, opts...)
		if errors.Is(err, llmprovider.ErrUnsupported) {
			// An id catalog does not list, such as a third party's: ask
			// the provider (0015-MADR, amendment "the wizard lists a
			// provider through its own ListModels").
			cat, err = providerCatalog(dCtx, o, d, res, source)
		}
	}
	if err == nil && !cat.Live && stepRefusal(cat.Err) {
		err = cat.Err
	}
	if stepRefusal(err) {
		return fallback, err
	}
	if err != nil {
		if len(static) == 0 {
			p.Notify(LevelWarn, "could not list models for %s (%s)", d.Label, describe(err))
			return fallback, err
		}
		p.Notify(LevelWarn, "could not list models for %s (%s); using the built-in catalog", d.Label, describe(err))
		return fallback, err
	}
	// A live listing with nothing recommended is kept for search (0021-MADR
	// C13).
	if len(cat.Recommended) == 0 && (!cat.Live || len(cat.Usable) == 0) {
		return fallback, cat.Err
	}
	switch {
	case cat.Live || chatGPT:
	case cat.Err != nil:
		p.Notify(LevelWarn, "live model listing for %s is unavailable (%s); search covers the built-in catalog only",
			d.Label, describe(cat.Err))
	default:
		p.Notify(LevelInfo, "live model listing for %s is unavailable; search covers the built-in catalog only", d.Label)
	}
	return cat, cat.Err
}

// chatGPTCatalog lists a ChatGPT session through the openai provider's
// ListModels, built from the run's Registry: the catalog does not list ChatGPT
// sessions (0015-MADR amendment "the ChatGPT helpers leave llmprovider"). The
// listing is live; there is no static one to fall back to.
func chatGPTCatalog(
	ctx context.Context,
	o Options,
	d llmprovider.Descriptor,
	res Result,
	source llmprovider.TokenSource,
) (catalog.Catalog, error) {
	lister, err := listerFor(o, d, res, source)
	if err != nil {
		return catalog.Catalog{}, err
	}
	models, err := lister.ListModels(ctx)
	if err != nil {
		return catalog.Catalog{}, err
	}
	models = withoutControl(models)
	// At most ModelLimit are recommended, as providerCatalog cuts them; search
	// covers them all (0026-MADR F47; 0038-MADR A2).
	recommended := cutModels(models, o.ModelLimit)
	return catalog.Catalog{Recommended: recommended, Usable: slices.Clone(models), Live: true}, nil
}

// providerCatalog lists a provider catalog does not know, such as a third
// party's, through its own ListModels (0015-MADR, amendment "the wizard lists
// a provider through its own ListModels"). Its ids are recommended in the
// provider's order, cut to Options.ModelLimit, and search covers them all. A
// provider that cannot list, or lists nothing, is an error, so the caller
// warns and offers the static catalog.
func providerCatalog(
	ctx context.Context,
	o Options,
	d llmprovider.Descriptor,
	res Result,
	source llmprovider.TokenSource,
) (catalog.Catalog, error) {
	lister, err := listerFor(o, d, res, source)
	if err != nil {
		return catalog.Catalog{}, err
	}
	models, err := lister.ListModels(ctx)
	if err != nil {
		return catalog.Catalog{}, err
	}
	if models = withoutControl(models); len(models) == 0 {
		return catalog.Catalog{}, fmt.Errorf("wizard: %s listed no models", d.Label)
	}
	recommended := cutModels(models, o.ModelLimit)
	return catalog.Catalog{Recommended: recommended, Usable: slices.Clone(models), Live: true}, nil
}

// withoutControl is a copy of models less each id holding a character a
// terminal acts on, which catalog drops from its own listings; a provider's
// ListModels is not catalog's (0066-MADR V1, Q3 (a)).
func withoutControl(models []string) []string {
	return slices.DeleteFunc(slices.Clone(models), func(m string) bool { return redact.StripControl(m) != m })
}

// cutModels is a copy of models, cut to its first limit when limit is
// positive (Options.ModelLimit; 0038-MADR A2).
func cutModels(models []string, limit int) []string {
	if limit > 0 && limit < len(models) {
		models = models[:limit]
	}
	return slices.Clone(models)
}

// listerFor builds d's provider from the run's Registry, with the run's
// credential, base URL, HTTP client and ProviderOptions, for its own listing.
func listerFor(
	o Options,
	d llmprovider.Descriptor,
	res Result,
	source llmprovider.TokenSource,
) (llmprovider.ModelLister, error) {
	opts := []llmprovider.Option{llmprovider.WithTokenSource(source)}
	if res.BaseURL != "" {
		opts = append(opts, llmprovider.WithBaseURL(res.BaseURL))
	}
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	opts = append(opts, o.ProviderOptions...)
	built, err := o.registry().New(d.ID, opts...)
	if err != nil {
		return nil, err
	}
	lister, ok := built.(llmprovider.ModelLister)
	if !ok {
		return nil, fmt.Errorf("wizard: %s cannot list models", d.Label)
	}
	return lister, nil
}

// choose runs p.Select and checks its answer, so a Prompter that returns an
// index outside the menu is an error, not a panic (0020-MADR F49).
func choose(p Prompter, title string, choices []Choice, defaultIdx int) (int, error) {
	idx, err := p.Select(title, choices, defaultIdx)
	if err != nil {
		return 0, err
	}
	if idx < 0 || idx >= len(choices) {
		return 0, fmt.Errorf("wizard: %q: the prompter chose %d of %d choices", title, idx, len(choices))
	}
	return idx, nil
}

func modelChoices(provider llmprovider.ProviderID, models []string) []Choice {
	out := make([]Choice, 0, len(models))
	for _, m := range models {
		label := catalog.Label(provider, m)
		if label == m {
			out = append(out, Choice{Label: m})
			continue
		}
		out = append(out, Choice{Label: label})
	}
	return out
}
