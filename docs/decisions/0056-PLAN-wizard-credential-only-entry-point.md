---
status: proposed
date: 2026-10-20
associated-madr: "0056-MADR-wizard-credential-only-entry-point.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Wizard's Credential-Only Entry Point, Kilo's Device Sign-In as a Key, and Refusals Without the Provider's Text

Associated MADR: [0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)

## Goal

The MADR's D1 to D5, as the owner answered its questions on 2026-10-20
(Q1 (a), Q2 (a), Q3 (a), Q4 (a)), so that each item of
[issue 5](https://github.com/maccavelli/go-llmprovider-sdk/issues/5)'s
acceptance list is a passing test:

* `wizard.ResolveCredential` runs the credential step alone and returns the
  checked catalog (D1);
* `Options.KiloDeviceKey` gives Kilo's device sign-in as a key (D2);
* every notice that showed a service's error shows its class and status
  instead (D3);
* `Options.SkipMethods` and `Options.NoEndpointPrompt` (D4);
* `v1.21.0` is released, after
  [0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md](0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md)'s
  `v1.20.1` (D5).

## Scope

### In scope

* **`wizard/`:** `configure.go`, `auth.go`, `logout.go`, a new
  `describe.go`, `example_test.go`, and new tests named `*_0056_test.go`.
* **One existing test, changed by D3,** with its reason:
  `wizard/model_select_test.go:267` (MADR F8).
* **Docs:** `docs/architecture.md` ("The wizard", and its layout entry if it
  names the files), `CHANGELOG.md`, the README's Status.
* **Records:** this pair; dated amendments to
  `0016-MADR-provider-auth-and-support-baseline.md` (A7) and
  `0017-MADR-together-provider-and-auth-extensions.md` (D2).
* **The issue:** a reply and its closing, on the owner's word.

### Out of scope

* Any exported name other than these four:

  | New name | Kind | Call site (R55) |
  | :--- | :--- | :--- |
  | `wizard.ResolveCredential` | function | gobble's `/login [provider]`, first-run offer and `gobble setup` |
  | `wizard.Options.KiloDeviceKey` | field | gobble's `/login kilo`, which stores the device token as Kilo's key |
  | `wizard.Options.SkipMethods` | field | gobble, leaving out the Codex-client methods and the vendor-CLI imports |
  | `wizard.Options.NoEndpointPrompt` | field | gobble, which keeps each provider's endpoint in its own configuration |

* `ConfigureLLM`'s steps and its defaults, other than D3's wording.
* The text of a returned error: `refusalError` keeps `%w` and its words.
* `configure.go:301` and `:310`'s notices (MADR D3).
* Any other provider's device sign-in as a key (MADR K2).
* gobble's own change, which its repository makes.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **Order with 0055.** Phase 0 may land at any time. Phases 1 to 3 start
  once 0055's Phase 1 is on `main`, so that CI is green under them, and
  Phase 4 once `v1.20.1` is released. Every `go` command sets
  `GOTOOLCHAIN=go1.27.3` until the owner's environment names it.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on both records;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **Red first.** Every new test is seen to fail, on today's code or on a
  planted break, by `tools/plant.py` on a scratch clone, never in the tree.
  The phase's record holds each plant and its failure line.
* **The drift check.** After each phase's code and before its new tests,
  `go test -race -count=1 ./wizard/` passes with every existing test
  unchanged. The one exception is Phase 3's named test.
* **The docs.** A function whose behaviour changes has its doc changed in
  the same phase, naming only what a test shows (R52). `make docs-check`
  stays at 0 violations.
* **The API.** `tools/apidiff.py v1.20.1` (or `v1.20.0`, until that is
  tagged) lists only the names a phase adds, from the table above.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * a name outside the table would be added;
  * an existing test would change for a reason no step names;
  * a doc would state a behaviour no test shows.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`
   (this PLAN is indexed by `tools/records.py add` first).
2. **0016-MADR,** at its end: "## Amendment 2026-10-20: Kilo's device
   sign-in as a key, under `Options.KiloDeviceKey`". A7 stands. With the
   field set, the device token is returned as `APIKey`, and the store holds
   none: one copy, which D11 asks for (0056-MADR D2).
3. **0017-MADR,** at its end: the same exception to D2's "it is stored in
   the `TokenStore`", pointing to 0056-MADR D2.

### Phase 1: `ResolveCredential` (D1)

1. **Extract the credential step.** In `configure.go`, unexported:
   * `chooseProvider(p Prompter, o Options) (llmprovider.Descriptor,
     error)`: today's lines 159 to 177;
   * `credentialStep(ctx, p, d, o) (Result, catalog.Catalog, error)`: today's
     lines 178 to 213 (the Result's fields, `resolveBaseURL`,
     `resolveCredential`, `discoverModels` and the refusal loop).

   `ConfigureLLM` becomes `chooseProvider`, then `credentialStep`, then its
   model steps (lines 214 to 240), as they are. **The drift check runs
   here**, before any other change.
2. **`ResolveCredential(ctx, p, o, id)`**, after `ConfigureLLM` in
   `configure.go`:
   * `id` empty: `chooseProvider`;
   * `id` set: `o.registry().Descriptor(id)`. If it is missing:
     `fmt.Errorf("wizard: no provider %q in the registry: %w", id,
     llmprovider.ErrInvalidProvider)`. If `o.Providers` is non-empty and
     lacks it: `fmt.Errorf("wizard: provider %q is not in
     Options.Providers: %w", id, llmprovider.ErrInvalidProvider)`;
   * then `credentialStep`, and its `Result` and catalog returned as they
     are.
3. **Its doc** says:
   * what it runs and what it does not: no model question, no fallback
     question, no manual model id;
   * that the credential is checked only with `Options.Discover`;
   * that `Existing.Model` and `NeedFallbacks` are ignored;
   * the error kinds (R52): a `Prompter`'s, as it is; a sign-in's; `ctx`'s;
     one matching `ErrAuthFailure` for a refused credential; one matching
     `ErrInvalidProvider` for an `id` it cannot offer; a plain error for a
     choice out of range.

   `ConfigureLLM`'s doc gains one sentence naming `ResolveCredential`.
4. **The example,** `ExampleResolveCredential` in `example_test.go`: a
   `TextPrompter` answering "your-api-key\n", `id` Claude, no `Discover`. It
   prints `res.Provider`, `res.Kind`, `res.Model == ""` and
   `len(cat.Recommended) > 0`. `// Output: claude api_key true true`.
5. **The tests,** `wizard/resolve_0056_test.go`, over `fakePrompter`
   (`newFake`, which fails on an unscripted prompt or a leftover answer) and
   `claudeRefusals`:
   * `TestResolveCredential_AsksNoProviderNoModel_0056`: `id` Claude,
     `Discover`, one key, the listing accepting it. No `Select` and no
     `Input` was shown. `APIKey` is the key, `Model` is empty, `Fallbacks`
     is nil, and the catalog is live and holds the listed model.
   * `TestResolveCredential_RefusedThreeKeys_0056`: three refusals. Three
     `Secret` prompts, an error matching `ErrAuthFailure` and naming 3
     keys, and the `TokenStore` saved nothing.
   * `TestResolveCredential_ChatGPTPlan_0056`: OpenAI, `AgentName`, an
     empty store, `stubPlanLogin`. The notice was confirmed with OpenAI's
     usage page, the plan session was saved, and `Kind` is `CredOAuth`. No
     model id was asked (`ConfigureLLM`'s plan test asks one).
   * `TestResolveCredential_EmptyIDAsksProvider_0056`: the provider menu is
     shown, as `ConfigureLLM` shows it.
   * `TestResolveCredential_UnknownID_0056`: an id the Registry lacks, and
     one `Options.Providers` leaves out, each match `ErrInvalidProvider`,
     with no prompt shown.
   * `TestResolveCredential_EmptyCatalog_0056`: Ollama with nothing
     installed, over `httptest`. No "enter a model id" question, and an
     empty catalog with no error.
   * `TestResolveCredential_ErrorKinds_0056`: a `Prompter` error matches as
     it is; a cancelled `ctx` gives an error matching `context.Canceled`;
     a choice out of range is a plain error.
6. **Plants** (`tools/plant.py`), each expected to fail a test above:
   * `ResolveCredential` running the model step (a call to `selectModel`
     added);
   * the refusal loop dropped from `credentialStep`;
   * the `o.Providers` check dropped;
   * the static fallback returned in place of the listed catalog;
   * `ResolveCredential` asking the provider when `id` is set.
7. **Docs:** `docs/architecture.md`'s "The wizard" names
   `ResolveCredential`; `CHANGELOG.md`'s `[Unreleased]` Added.

### Phase 2: `KiloDeviceKey`, `SkipMethods`, `NoEndpointPrompt` (D2, D4)

1. **`Options.KiloDeviceKey bool`,** documented after `AgentName`:
   * `offeredAuthMethods`: Kilo's `AuthDeviceCode` needs no store when it is
     set;
   * `resolveCredential`: Kilo's device method with the field set goes to a
     new `resolveKiloDeviceKey`. It runs `loginDeviceOAuth`, refuses an
     empty token as today, and returns `staticCredential(CredAPIKey,
     session.Access)` with `keyFrom` a new `keyDevice`. It saves nothing
     and reads no profile;
   * `refusalError`: `keyFrom == keyDevice` gives `"wizard: %s refused the
     device sign-in; sign in again: %w"`;
   * the refusal loop ends on it, as on every key that was not typed.
2. **`Options.SkipMethods []llmprovider.AuthMethodID`:** `offeredAuthMethods`
   leaves them out, after the store and name filters. When the descriptor
   has methods and none is left, `resolveCredential` returns
   `fmt.Errorf("wizard: Options.SkipMethods leaves no sign-in method for
   %s", d.Label)`. A descriptor with no methods (a provider that needs no
   key) is unchanged.
3. **`Options.NoEndpointPrompt bool`:** `resolveBaseURL` asks nothing. The
   URL is `o.Existing.BaseURL` for that provider, else
   `d.DefaultBaseURL`.
   * A remote URL goes through `validateBaseURL` as a typed one does; a
     refusal is returned, not re-asked.
   * A local one through `catalog.ValidateOllamaURLWith`; a failure is
     returned as `fmt.Errorf("wizard: cannot reach %s: %w", url, err)`.
4. **The docs of `ConfigureLLM` and `ResolveCredential`** name the two new
   errors (R52).
5. **The tests,** `wizard/options_0056_test.go`, with `stubKilo` counting
   its profile reads:
   * `TestKiloDeviceKey_ReturnsTheKey_0056`: with a store. `Kind` is
     `CredAPIKey`, `APIKey` is the token, the store saved nothing, no
     profile was read, no "Use Kilo as:" was shown, and `Organization` is
     empty.
   * `TestKiloDeviceKey_NoStoreNeeded_0056`: with no store, the device
     method is in the menu.
   * `TestKiloDeviceKey_Refused_0056`: a 401 listing. The error matches
     `ErrAuthFailure` and says "refused the device sign-in".
   * `TestKiloDeviceKey_KeptNextRun_0056`: `Existing` is that key. The run
     offers "Keep the existing key", and keeps it.
   * `TestSkipMethods_LeavesThemOut_0056`: OpenAI with the Codex methods
     skipped. The menu's labels lack them.
   * `TestSkipMethods_NoneLeft_0056`: Claude with `AuthAPIKey` skipped. The
     plain error, with no prompt shown.
   * `TestNoEndpointPrompt_0056`: Kilo, default and `Existing`. No `Input`,
     and each `BaseURL`. Ollama over `httptest`, reachable: its URL.
     Ollama, unreachable: an error saying "cannot reach", with no prompt.
   * The existing Kilo tests (`kilo_device_test.go`) pass unchanged: the
     field's zero value is today's behaviour.
6. **Plants:** the device token saved to the store in key mode; the profile
   read in key mode; `SkipMethods` ignored; an empty menu falling into the
   API-key path; the endpoint asked despite `NoEndpointPrompt`; an
   unreachable Ollama endpoint returned as success.
7. **Docs:** "The wizard" in `docs/architecture.md`, its Kilo entry naming
   key mode; `CHANGELOG.md` Added.

### Phase 3: refusals without the provider's text (D3)

1. **`describe(err error) string`,** in a new `wizard/describe.go`:
   * the context's end first: `context.DeadlineExceeded` gives "timed
     out", and `context.Canceled` "cancelled";
   * else the first sentinel the error matches, in the MADR's order (a
     child before its parent), as its text without "llmprovider: ";
   * then ", HTTP <status>", when `errors.As` finds an `*APIError` with a
     non-zero `Status`;
   * else "the request failed", with the status if there is one.
2. **The six notices** use it in place of `%v`:
   * `configure.go:204`, `:469`, `:472` and `:483`;
   * `auth.go:321`;
   * `logout.go:48`.

   For example: "Acme refused this key (authentication failed, HTTP 401);
   enter another".
3. **`ConfigureLLM`'s doc:** "A service's error is shown as its class and
   status, never its text; a returned error keeps its text, for
   errors.Is." `refusalError`'s comment says the same.
4. **The changed test.** `model_select_test.go:267`'s expected notice
   becomes "live model listing for OpenCode Zen is unavailable (provider
   unavailable, HTTP 500); …". The reason: D3, by the owner's Q1 answer.
   Nothing else in it changes.
5. **The tests,** `wizard/describe_0056_test.go`:
   * `TestDescribe_0056`, a table:
     * `*APIError` 401 gives "authentication failed, HTTP 401";
     * 403 not permitted gives "not permitted, HTTP 403";
     * 429 with `ErrQuotaExhausted` gives "quota exhausted, HTTP 429", not
       "rate limited";
     * 500 gives "provider unavailable, HTTP 500";
     * a deadline gives "timed out";
     * a plain `errors.New("dial tcp …")` gives "the request failed".
   * `TestRefusalNotice_NoProviderText_0056`: Claude's listing answers 401
     with the message "Incorrect API key provided: sk-proj-***…***abcd".
     Every string in `allText` lacks "sk-" and "Incorrect", and the notice
     reads "authentication failed, HTTP 401". The returned error, after
     three keys, still matches `ErrAuthFailure`.
   * `TestListingWarnings_NoProviderText_0056`: a 500 whose message holds
     the same text, through each of the three listing warnings, and Kilo's
     profile.
   * `TestLogoutNotice_NoProviderText_0056`: a revocation answered 400 with
     that text.
6. **Plants:** one notice back to `%v`, once for each of the six; the
   sentinel order with `ErrRateLimited` before `ErrQuotaExhausted`; the
   status left out.
7. **Docs:** "A refused key" in `docs/architecture.md`; `CHANGELOG.md`
   Changed.

### Phase 4: release `v1.21.0` (D5)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.21.0] - <date>`, dated
   by the commit the tag will name. It holds Added (the four names) and
   Changed (D3). An empty `[Unreleased]` goes above it, and the comparison
   links are updated. The README's Status names `v1.21.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.20.1`, listing
   exactly the table's four additions; the notes from `--section v1.21.0
   --repo maccavelli/go-llmprovider-sdk` on a local clone with a local tag,
   scanned; the disclosure guard.
3. **The tag,** on the owner's word: `v1.21.0`, annotated, pushed. Every
   `validate` job passes, and the `release` job's body equals step 2's
   notes.
4. **After the tag:** `tools/consumer.py v1.21.0` and `tools/offline.py
   v1.21.0`.
5. **The issue,** on the owner's word: a reply naming `v1.21.0`, each
   acceptance item with its test, and gobble's change (its `/login` over
   `ResolveCredential`, `KiloDeviceKey` for Kilo). It is written to a
   scratch file, scanned and shown first. Then issue 5 is closed.
6. **Close:** the MADR's Confirmation, item by item; this PLAN
   `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | With `id` set: no provider or model question; the key, the listed catalog, no `Model` | 1 | `TestResolveCredential_AsksNoProviderNoModel_0056` |
| V2 | A refused typed key is asked again; after three, the refusal, and nothing stored | 1 | `TestResolveCredential_RefusedThreeKeys_0056` |
| V3 | The ChatGPT plan sign-in shows the notice and saves the session | 1 | `TestResolveCredential_ChatGPTPlan_0056` |
| V4 | Kilo's key mode: `CredAPIKey` with the token, nothing saved, no organization question | 2 | `TestKiloDeviceKey_ReturnsTheKey_0056` |
| V5 | A refusal holding `sk-` text shows the class and status, none of the message | 3 | `TestRefusalNotice_NoProviderText_0056` |
| V6 | `ConfigureLLM`'s existing tests pass, one named test's text aside | 1–3 | the drift checks |
| V7 | `SkipMethods` and `NoEndpointPrompt` behave as D4 says | 2 | their tests |
| V8 | Only the four names are added | 1–4 | `tools/apidiff.py` |
| V9 | `ResolveCredential`'s doc names its error kinds, and its example runs | 1 | `make docs-check`; `go test` |
| V10 | `v1.21.0`'s Release is its section | 4 | `gh release view` |
| V11 | Every new test failed first | 1–3 | the execution record |

V1 to V5 are the issue's acceptance list, in its order.

## Rollout and Rollback

* **Rollout.** Five commits on `main`, staged by the agent and committed by
  the owner: the records, one commit per phase, then the release commit.
  Then the owner's tag.
* **Rollback.**
  * Each phase reverts on its own, newest first.
  * Reverting Phase 3 restores the `%v` notices and the old test text.
  * Reverting Phase 1 or 2 removes names. That is allowed only before
    `v1.21.0` is tagged (R48). After it, a fix ships forward.
