---
status: accepted
date: 2026-10-21
decision-makers: repository owner
consulted: "GitHub issue 5 (maccavelli/go-llmprovider-sdk), and gobble-cli at d76e3d2 as it cites; 0016-MADR-provider-auth-and-support-baseline.md (D11, A7); 0017-MADR-together-provider-and-auth-extensions.md (D2); 0043-MADR-openai-chatgpt-plan-sign-in.md; the standards guide (R19, R39, R40, R48, R50 to R52, R55); this repository at 5c60593"
informed: gobble-cli, and every consumer of the wizard
---

<!-- markdownlint-disable MD013 -->

# Give the Wizard a Credential-Only Entry Point, Kilo's Device Sign-In as a Key, and Refusals Without the Provider's Text

## Context and Problem Statement

[Issue 5](https://github.com/maccavelli/go-llmprovider-sdk/issues/5),
opened 2026-10-21 03:05 UTC by the owner for gobble-cli, asks for three
things and two optional ones. `wizard.ConfigureLLM` always ends with a model
menu, so an application with its own model picker cannot use the wizard's
sign-in without also taking the menu. Kilo's device sign-in always ends as
a stored session with an organization. A refusal is shown with the
provider's own error text.

The wizard is unchanged from `v1.17.0` to `HEAD`: `git diff --stat v1.17.0
HEAD -- wizard/` is empty.

### Facts (observed 2026-10-20 at `5c60593`)

* **F1. The model step cannot be skipped.** `ConfigureLLM` (C1) chooses a
  provider, resolves a base URL and a credential, lists models, then always
  runs `selectModel` (C3). With an empty listing it asks for a model id
  instead (C5). No `Options` field ends it earlier. The credential steps,
  `resolveCredential` (C8) and `resolveChatGPTPlan` (C11), are unexported.
* **F2. What the credential step holds,** each worth reusing, as the issue
  lists them:
  * the method menu, filtered by `TokenStore` and `AgentName` (C6);
  * the environment and existing-key offers, masked;
  * the key-shape and foreign-vendor warnings (0028-MADR H10b);
  * the refusal loop, up to three typed keys (`maxTypedKeys`), with
    `refusalError` (C2, C14);
  * a saved session offered first (0020-MADR F19);
  * the ChatGPT plan sign-in (C11): OpenAI's one-time notice, the previous
    plan session's client reused, the pasted callback URL on a first
    sign-in (0043-MADR A3).
* **F3. The listing is the check.** A key is checked only when
  `Options.Discover` is set (C4): the listing's refusal is what the loop
  reads.
* **F4. Kilo's device sign-in is a session.**
  * `needsStore` (C7) is true for it, so it is offered only with a
    `TokenStore`.
  * It saves the session (C9), and with organizations asks "Use Kilo as:"
    (C10), returning the choice in `Result.Organization`.
  * This is 0016-MADR A7's choice, whose rejected option reads "Keeping
    Kilo's token as `APIKey`: two copies, which D11 exists to prevent" (C19).
    0017-MADR D2 says the token "has no refresh and no known expiry. It is
    stored in the `TokenStore` and applied like an API key."
* **F5. Kilo sends a key and a device token alike.** The provider applies
  either as `Authorization: Bearer` (C17), and its own listing passes a
  session's token as `WithAPIKey` (C18). So the device token works as Kilo's
  API key **[read]**.
* **F6. A refusal shows the provider's text.** The refusal notice (C12)
  and the three listing warnings (C13, C15, C16) print the error with `%v`.
  An `*APIError`'s text ends with its `Message` (C21), "the service's
  message, redacted and bounded to 512 bytes" (C20).
* **F7. Redaction leaves a key the provider has already masked.** On a
  scratch clone, `redact.Redact` returned these unchanged **[probed]**:
  * "Incorrect API key provided: sk-proj-***…***abcd. You can find your API
    key at `https://platform.openai.com/account/api-keys`.";
  * "Incorrect API key provided: sk-abc12***…***wxyz.".

  Its generic key pattern needs 16 characters from `[A-Za-z0-9_-]` after
  `sk-` (C22), and the asterisks end the match. The strings are written in
  the shape gobble's records report, not taken from a provider's reply.
* **F8. One existing test reads a warning's `%v`.**
  `wizard/model_select_test.go:267` expects "live model listing for
  OpenCode Zen is unavailable (llmprovider: provider unavailable: opencode
  HTTP 500); …" (C23). The other two that read these warnings match their
  prefix only.
* **F9. The methods and the endpoint come from the descriptor.** Changing
  either means registering other descriptors, and `Registry.Register`
  needs a `Factory` (C24), which the registry does not expose.
* **F10. A precedent for the shape.** `wizard.Logout(ctx, p, o, id)` (C25)
  already takes the provider as an argument.
* **F11. gobble, at `d76e3d2`,** is as the issue says **[read]**:
  * `HasLogin` is Grok's alone (`llm/provider/login.go:71`);
  * `KeyLogin` returns the device session's `Access` as Kilo's key;
  * `internal/cli/auth.go:198` refuses OpenAI's login until this module
    offers the ChatGPT plan sign-in to it.

  It is the call site R55 asks for.

## Decision Drivers

* A consumer should be able to reuse the wizard's credential logic without
  re-implementing it. The ChatGPT plan sign-in matters most: its notice is
  OpenAI's terms (0043-MADR D7).
* `ConfigureLLM`'s behaviour and its tests stay as they are.
* Nothing shown through the `Prompter` should carry a credential, even one
  the provider masked (`ConfigureLLM`'s doc: "anything shown to the user
  is masked").
* One copy of a credential (0016-MADR D11).
* Only additions to the API (R48); each new name has a call site (R55).

## Considered Options

The entry point (request 1):

* **E1.** A new function, `wizard.ResolveCredential(ctx, p, o, id)
  (Result, catalog.Catalog, error)`.
* **E2.** An `Options` field, such as `CredentialOnly bool`, that ends
  `ConfigureLLM` after the check.

Kilo's device sign-in (request 2):

* **K1.** An `Options` field, `KiloDeviceKey bool`.
* **K2.** A general field for "a device sign-in as a key".
* **K3.** No change; the consumer runs `auth.LoginDeviceOAuth` itself.

The refusal wording (request 3):

* **R1.** By default: the class and HTTP status, never the message.
* **R2.** The same, under an `Options` field; the default unchanged.

The optional fields (request 4):

* **O1.** Both: `Options.SkipMethods []llmprovider.AuthMethodID` and
  `Options.NoEndpointPrompt bool`.
* **O2.** Neither, for now.

## Decision Outcome

Chosen options: **E1, K1, R1 and O1**, the owner's answers of 2026-10-20
to Q1 to Q4 below, each the recommended one. The record stays `proposed`
until its PLAN's Phase 0 marks it `accepted`.

### D1. `wizard.ResolveCredential`

```go
func ResolveCredential(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) (Result, catalog.Catalog, error)
```

* **The provider.** With `id` empty it asks, as `ConfigureLLM` does, over
  `o.Providers`. With `id` set it asks nothing. An `id` the run's Registry
  lacks, or that `o.Providers` leaves out, is an error matching
  `llmprovider.ErrInvalidProvider`, as `Registry.New` gives one.
* **The steps,** `ConfigureLLM`'s own, unchanged: the endpoint, then the
  credential (every method, the ChatGPT plan included), then the listing
  that checks it, with the refusal loop, `maxTypedKeys` and
  `refusalError`.
* **The listing** follows `o.Discover`, as in `ConfigureLLM` (F3): without
  it, nothing is checked and the catalog is the static one. The doc says
  so.
* **The result.** The `Result` has no `Model` and no `Fallbacks`, and the
  catalog is the one the check listed, so the consumer need not list again.
  An empty catalog is returned as it is, with no question: the "enter a
  model id" step is `ConfigureLLM`'s. `o.Existing.Model` and
  `o.NeedFallbacks` are ignored.
* **Its errors**, named in its doc (R52), each shown by a test: a
  `Prompter`'s, as it is; a sign-in's; `ctx`'s; a refused credential
  matching `llmprovider.ErrAuthFailure`; `ErrInvalidProvider`; a choice out
  of range, a plain error.
* **`ConfigureLLM`** becomes the provider menu, then the same unexported
  credential step, then the model step. Its tests pass unchanged, except
  F8's under R1.
* It gets an example (R50), offline, over a scripted `Prompter`.

### D2. `Options.KiloDeviceKey`

When true:

* Kilo's device method is offered without a `TokenStore`;
* the sign-in returns `Kind: CredAPIKey`, with the device token in
  `APIKey`;
* it saves no session, reads no profile and asks no organization question,
  so `Organization` is empty;
* a token the listing refuses ends the run with "Kilo refused the device
  sign-in; sign in again", matching `ErrAuthFailure`;
* the next run offers it as the existing key (`Options.Existing`), as any
  API key.

When false, Kilo's sign-in is as today.

This amends 0016-MADR A7 for this mode only. A7 rejected the token as
`APIKey` because the store would also hold it: two copies. Here the store
holds none, so there is one copy, which D11 asks for. 0017-MADR D2's "it is
stored in the `TokenStore`" gains the same exception. Both records get a
dated amendment pointing here.

The field names Kilo because no other device sign-in qualifies: Grok's
session refreshes and expires, so its token is not a key (K2 rejected).

### D3. Refusals worded from the error's class

The refusal notice and the three listing warnings (C12, C13, C15, C16) show
a description built from the error, never its text. So do the two other
notices that show a service's error, which the issue does not list:

* a failed revocation in `Logout` (C28);
* Kilo's unreadable profile (C29).

Two notices stay as they are, since neither carries a service's text:

* the endpoint's validation (`configure.go:301`), which is the module's own
  message;
* a local endpoint that cannot be reached (`configure.go:310`), the user's
  own Ollama, which takes no key.

The description:

* **the class:** the most specific of the module's kind sentinels the
  error matches, a child before its parent: `ErrQuotaExhausted`,
  `ErrRateLimited`, `ErrNotPermitted`, `ErrAuthFailure`,
  `ErrContextOverflow`, `ErrIncomplete`, `ErrInvalidRequest`,
  `ErrProviderUnavailable`, `ErrUnsupported`. It is shown as the
  sentinel's text without its "llmprovider:" prefix, such as
  "authentication failed" (C27);
* **the status:** " (HTTP 401)" when an `*APIError` with a status is in the
  chain;
* **a context's end:** "timed out" or "cancelled";
* **anything else:** "the request failed".

So "Acme refused this key (authentication failed, HTTP 401); enter
another". `refusalError` keeps `%w`, so `errors.Is` still matches, and its
doc says that its text, unlike the notices, can hold the provider's
message: a consumer that prints a returned error words it itself. A new
unexported `describe` in `wizard` builds the description.

`APIError.Code`, such as `invalid_api_key`, is left out as well. It is a
diagnostic, not a secret, but the issue asks for the class and status only,
and the owner chose that (Q1 (a)).

### D4. `Options.SkipMethods` and `Options.NoEndpointPrompt`

* **`SkipMethods`** leaves those methods out of the menu, after
  `TokenStore` and `AgentName` filter it. gobble's case: the Codex-client
  methods and the vendor-CLI imports. If no method is left, the run is a
  plain error naming the provider.
* **`NoEndpointPrompt`** asks no endpoint: the base URL is
  `o.Existing.BaseURL` for that provider, else the descriptor's default.
  For a local provider (Ollama) the reachability check still runs, and a
  failure ends the run with that error, since there is no prompt to ask
  again.

Each is one field and a filter, and saves a consumer from rebuilding the
Registry around wrapped factories (F9).

### D5. Tests, docs, release

* **The issue's acceptance list, as tests,** each first seen failing on a
  planted break:
  * the entry point with `id` set: no provider question, no model question,
    the typed key and the listed catalog, no `Model`;
  * a refused typed key asked for again, and after three the refusal error,
    with nothing stored;
  * OpenAI with `AgentName` and no saved plan session: the notice, then the
    session saved;
  * Kilo in key mode: `CredAPIKey` with the token, nothing saved, no
    organization question;
  * a refusal whose message holds `sk-` text: the class and status shown,
    nothing of the message;
  * `ConfigureLLM`'s existing tests.
* **Plus:** an unknown `id`; `SkipMethods` emptying the menu;
  `NoEndpointPrompt` for Ollama, reachable and not.
* **Docs:** `docs/architecture.md`'s wizard section, and `CHANGELOG.md`'s
  Added and Changed.
* **Release:** `v1.21.0`, a minor release, since it adds exported names.
  It follows
  [0055-MADR-go-1-27-3-and-the-v1-20-0-release.md](0055-MADR-go-1-27-3-and-the-v1-20-0-release.md)'s
  `v1.20.1` (Q4).
* **The new names**, for R55, each with gobble's `/login` as the caller:
  `ResolveCredential`, `Options.KiloDeviceKey`, `Options.SkipMethods`,
  `Options.NoEndpointPrompt`.

### Consequences

* Good, because a consumer gets the whole credential step, the ChatGPT plan
  sign-in included, with OpenAI's notice in one place.
* Good, because a consumer that stores keys can take Kilo's sign-in as a
  key, which gobble already does by hand.
* Good, because no provider message reaches the screen through the wizard.
* Neutral, because `ConfigureLLM` changes only its notices' wording (R1),
  and one test's expected text (F8).
* Bad, because a user loses the provider's own explanation in the notice,
  which sometimes helps: "quota exceeded", "model not found". The class
  carries most of it, and the code is left out too (Q1).
* Bad, because an organization-only Kilo account in key mode sends no
  organization. Whether Kilo then accepts the personal-account requests is
  not measured **[unknown]**. A consumer that needs organizations leaves the
  field off.
* Bad, because four names are added to the API, and stay (R48).

### Confirmation

* Each test of D5, seen failing first, and `make api-check` listing exactly
  the four additions.
* `make docs-check`: `ResolveCredential`'s doc names its error kinds (R52),
  and its example runs (R50).
* The issue's acceptance list, item by item, in the PLAN's record.

## Pros and Cons of the Options

### E1. `ResolveCredential`

* Good, because it returns the catalog, which a `Result` cannot hold
  (it is data the consumer persists).
* Good, because it matches `Logout`'s shape, with `id` as an argument
  (F10).
* Bad, because it is a second entry point to keep in step with
  `ConfigureLLM`; one shared step does that.

### E2. `Options.CredentialOnly`

* Good, because it is one field.
* Bad, because it returns no catalog, so the consumer lists again.
* Bad, because `ConfigureLLM`'s name and doc promise a model.

### K1. `Options.KiloDeviceKey`

* Good, because it says exactly which sign-in changes.
* Bad, because it is a provider's name in a general struct.

### K2. A general device-as-key field

* Bad, because no other device token is a key: Grok's expires and
  refreshes.

### K3. No change

* Bad, because the consumer re-implements the device flow, as gobble does,
  and loses the wizard's method menu around it.

### R1. By default

* Good, because the wizard's promise holds for every consumer, and F7 shows
  that redaction does not cover a provider's masked echo.
* Bad, because one test's text changes (F8), and a user sees less.

### R2. Under an `Options` field

* Good, because the default is unchanged.
* Bad, because every consumer that does not set it keeps the provider's
  text on screen.

### O1. Both optional fields

* Good, because each removes a Registry rebuild that F9 makes awkward.
* Bad, because two more names.

### O2. Neither

* Good, because the API grows less.
* Bad, because gobble wraps factories to filter methods.

## Owner questions

* **Q1 (D3). The wording.**
  * **(a)** By default, the class and status only (recommended).
  * **(b)** By default, the class, status and `APIError.Code`.
  * **(c)** Under an `Options` field, the default unchanged.

  **Answered 2026-10-20: (a).**
* **Q2 (D4). The optional fields.**
  * **(a)** Both (recommended).
  * **(b)** `SkipMethods` only.
  * **(c)** Neither.

  **Answered 2026-10-20: (a).**
* **Q3 (D1). The entry point.**
  * **(a)** `ResolveCredential` (recommended).
  * **(b)** `Options.CredentialOnly`.

  **Answered 2026-10-20: (a).**
* **Q4 (D5). The release.**
  * **(a)** `v1.21.0`, after 0055's `v1.20.1` (recommended).
  * **(b)** Together with 0055, as `v1.21.0`, with no `v1.20.1`.

  **Answered 2026-10-20: (a).**

## More Information

* The issue:
  [maccavelli/go-llmprovider-sdk#5](https://github.com/maccavelli/go-llmprovider-sdk/issues/5).
* The redaction probe ran on a scratch clone of `5c60593`; its test is kept
  in the session's scratch directory, not committed.
* Related: [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
  (D11, A7), [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md)
  (D2), [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)
  (D7, A3), [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  (D-H10, H10b).

### Code claims

<!-- claims: self=5c60593 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The one entry point | `wizard/configure.go:158` | `func ConfigureLLM(ctx context.Context, p Prompter, o Options) (Result, error) {` |
| C2 | The refusal loop's limit | `wizard/configure.go:323` | `const maxTypedKeys = 3` |
| C3 | The model step, always run | `wizard/configure.go:232` | `if res.Model, err = selectModel(ctx, p, d, cat, o); err != nil {` |
| C4 | No listing without Discover | `wizard/configure.go:424` | `if !o.Discover {` |
| C5 | The manual model id | `wizard/configure.go:218` | `manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))` |
| C6 | The method filter | `wizard/auth.go:76` | `!needsStore(d.ID, m.ID)) && (haveName` |
| C7 | A device sign-in needs a store | `wizard/auth.go:94` | `return true` |
| C8 | The credential step, unexported | `wizard/auth.go:126` | `func resolveCredential(` |
| C9 | Kilo's session saved | `wizard/auth.go:307` | `if err := o.TokenStore.Save(ctx, d.ID, session); err != nil {` |
| C10 | Kilo's organization question | `wizard/auth.go:343` | `idx, err := choose(p, "Use Kilo as:", choices, defaultIdx)` |
| C11 | The ChatGPT plan sign-in | `wizard/auth.go:218` | `func resolveChatGPTPlan(ctx context.Context, p Prompter, o Options) (resolvedCredential, error) {` |
| C12 | The refusal notice | `wizard/configure.go:204` | `p.Notify(LevelWarn, "%s refused this key (%v); enter another", d.Label, listErr)` |
| C13 | A listing warning | `wizard/configure.go:469` | `p.Notify(LevelWarn, "could not list models for %s (%v)", d.Label, err)` |
| C14 | The refusal error keeps the chain | `wizard/configure.go:337` | `return fmt.Errorf("wizard: %s refused %d keys: %w", d.Label, keys, err)` |
| C15 | A listing warning | `wizard/configure.go:472` | `p.Notify(LevelWarn, "could not list models for %s (%v); using the built-in catalog", d.Label, err)` |
| C16 | A listing warning | `wizard/configure.go:483` | `p.Notify(LevelWarn, "live model listing for %s is unavailable (%v); search covers the built-in catalog only",` |
| C17 | Kilo sends its credential as Bearer | `llmprovider/providers/kilo/kilo.go:302` | `token.Apply(r, headerAuthorization, "Bearer")` |
| C18 | Kilo lists a session's token as a key | `llmprovider/providers/kilo/kilo.go:318` | `slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)` |
| C19 | 0016-MADR A7's rejected option | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:737` | `Keeping Kilo's token as` |
| C20 | An APIError's message | `llmprovider/api_error.go:86` | `// Message is the service's message, redacted and bounded to 512 bytes.` |
| C21 | Its text ends with the message | `llmprovider/api_error.go:126` | `for _, s := range []string{e.Reason, e.Message} {` |
| C22 | The generic key pattern | `internal/redact/redact.go:121` | `reTokenFallback = regexp.MustCompile(` |
| C23 | The test that reads a warning's text | `wizard/model_select_test.go:267` | `notice := "live model listing for OpenCode Zen is unavailable (llmprovider: provider unavailable: opencode HTTP 500); " +` |
| C24 | Register needs a factory | `llmprovider/registry.go:66` | `func (r *Registry) Register(d Descriptor, f Factory) error {` |
| C25 | Logout takes the id | `wizard/logout.go:22` | `func Logout(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) error {` |
| C26 | The unknown-id sentinel | `llmprovider/api_error.go:54` | `ErrInvalidProvider = errors.New("llmprovider: invalid provider id")` |
| C28 | A revocation's notice | `wizard/logout.go:48` | `p.Notify(LevelWarn, "could not revoke the %s session (%v); deleting the local copy", label, err)` |
| C29 | Kilo's profile notice | `wizard/auth.go:321` | `p.Notify(LevelWarn, "could not read the Kilo account's organizations (%v); using the personal account", err)` |
| C27 | A sentinel's text has the prefix | `llmprovider/provider.go:63` | `ErrAuthFailure         = errors.New("llmprovider: authentication failed")` |
