---
status: in-progress
date: 2026-10-30
associated-madr: "0066-MADR-wizard-v1-24-debugging-pass-findings.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Remediation of the Wizard's v1.24 Debugging-Pass Findings

Associated MADR: [0066-MADR-wizard-v1-24-debugging-pass-findings.md](0066-MADR-wizard-v1-24-debugging-pass-findings.md)

## Goal

The MADR's D1, under the owner's answers of 2026-10-30, (a) to each of Q1 to
Q6, and its amendments "facts found while planning" and "issue #10": each of
the 16 findings *(17, with V17 added 2026-10-31)*
becomes a test that fails on `3fad786`, or a planted break that a new check
catches, and then passes. In particular:

* no byte a server sends acts on the user's terminal, and no listed id
  holding a control character reaches a menu or the Result (V1);
* a Chat Completions server, keyed or not, is configured through the wizard
  to a Result that builds its provider, with the menu and the facts
  `catalog` builds for it (V2, V3, V4), and the servers running on this
  machine are offered under `Options.LocalServers` (V5);
* the returned catalog's `Err` says why a listing failed, on every path, and
  `ResolveCredential`'s doc says how to tell an unchecked credential (V6); a
  refused kept session leads to the sign-in menu (V7);
* no notice shows a service's text or a URL's query (V8, V9); an offline
  listing is called offline (V10); *a failed listing beside a static
  catalog is told (V17, added 2026-10-31)*;
* an unknown `Options.Providers` id is named (V11), the guide is right
  (V12), the wizard's tests run behind the `localonly` guard (V13), and the
  two untested paths are tested (V14);
* a caller that adds request headers keeps the SDK's bounded client,
  through `llmprovider.NewHTTPClient`, and the way to do it is documented
  (V15, V16; issue #10);
* `v1.25.0` is released, a minor release for the three exported names.

## Scope

### In scope

* **`wizard/`:** `configure.go`, `auth.go`, `base_url.go`, `describe.go`
  and `text_prompter.go`; a new `localonly_test.go`; new tests named
  `*_0066_test.go`. `model_select.go` does not change: V14's test of
  `capMatches` covers it as it is.
* **`llmprovider/catalog/`:** a new file for `Lister`; the control-character
  filter in `discovery.go` (`fetchListing` and `ollamaCatalog`); new tests
  named `*_0066_test.go`.
* **`llmprovider/providers/chatcompat/`:** `ListCatalog` on its provider;
  `discover.go`'s `Server.Models` filtered as `catalog` filters a listing;
  the package doc; new tests named `*_0066_test.go`.
* **`llmprovider/`:** `NewHTTPClient` in `options.go`, beside
  `WithHTTPClient`; `WithHTTPClient`'s doc and the package doc in
  `provider.go`; `ExampleNewHTTPClient`; new tests named
  `*_0066_test.go` (Phase 7).
* **One existing test gains rows, its rows unchanged:**
  `TestRefusalError_NamesTheSource` (`wizard/configure_0028_test.go`), a
  row for a refused request that carried no key (Phase 2).
* *Added 2026-10-31, the deviation in Phase 2's record:*
  `TestResolveBaseURL_Validates` (`wizard/configure_test.go`), whose
  descriptor gains the real Kilo's `DefaultBaseURL`, its rows unchanged.
* *Added 2026-10-31, the deviation in Phase 8's record:*
  `TestConfigureLLM_ListingTokenFailureUsesStaticCatalog`
  (`wizard/model_select_edge_test.go`), which gains an `HTTPClient` that
  refuses every request, its own cache and no models.dev document, its
  assertions unchanged.
* **Docs:** each changed function's and type's doc (R41, R51, R52);
  `README.md`'s "The configuration wizard" and Status;
  `docs/architecture.md`'s package table row for `wizard`, which gains
  `providers/chatcompat` (R43), its "The wizard", and its
  `internal/localonly` row, which gains `wizard`;
  `docs/guides/adding-a-provider.md`'s "The models `wizard` offers";
  `AGENTS.md`'s list of the packages behind the guard; `CHANGELOG.md`.
* **Records:** this pair, the MADR's status, and their index rows.

### Out of scope

* `ErrEmptyAnswer` in `describe`: no listing gives it; the MADR names
  `ErrOffline` only (D1, phase 5).
* A probe of Hugging Face's listing with a wrong key: the MADR's amendment
  records it as not measured. A live test of it would be a record of its
  own.
* Offering a found server in a consumer's own configuration, such as
  gobble-cli's `models.json`: D5 of gobble-cli's
  `0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md` decides that, in
  that repository.
* The workspace prompt under a caller with no terminal: checked in the
  MADR, not a finding.
* gobble-cli, which takes `v1.25.0` in its own repository, and changes its
  `withHeaders` to wrap `NewHTTPClient()` there, by its own record.

### Exported names (R55)

| Name | Package | The call site that needs it |
| :--- | :--- | :--- |
| `Lister`, an interface, with its one method `ListCatalog` | `llmprovider/catalog` | the wizard's `providerCatalog`, which lists through it before `ListModels`; `chatcompat`'s provider implements it (MADR Q1 (a); amendment "Q1 (a)'s name and shape") |
| `Options.LocalServers`, a field | `wizard` | a consumer's configure step; the wizard's provider menu and `credentialProvider` read it (MADR Q2 (a)) |
| `NewHTTPClient`, a function | `llmprovider` | gobble-cli's `withHeaders` (`llm/provider/provider.go:150` at `61f7115`), which wraps `http.DefaultTransport` today; the docs of `WithHTTPClient` and the wizard's `Options.HTTPClient` point to it (MADR amendment "issue #10", D2) |

`chatcompat`'s `ListCatalog` is a method of an unexported type, so it is
not an exported name; the package doc names it. No signature changes.
~~`make api-check` and `tools/apidiff.py v1.24.1` list exactly the three
additions.~~ *`make api-check` shows no incompatible change, and
`tools/apidiff.py v1.24.1` lists exactly the three additions; `make
api-check` lists none (the deviation of 2026-10-31 in Phase 7's record).*

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `make api-check` and `make docs-check` (`sdkaudit -check`, R50 to
    R52), in each phase that adds or documents an exported name;
  * `tools/claims.py` on the MADR;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **Red first.** Every new test is seen to fail on `3fad786`'s code, or on
  a planted break, by `tools/plant.py` on a scratch clone, never in the
  tree. The phase's record holds each failure line. A probe of the MADR (A
  to L) becomes its finding's first test.
* **The drift check.** After each phase's code and before its new tests,
  `go test -race -count=1 -shuffle=on ./wizard/ ./llmprovider/catalog/
  ./llmprovider/providers/chatcompat/` passes with every existing test
  unchanged, the one named in Scope aside.
* **No network.** Every new test serves its own replies, through `httptest`
  or an `HTTPClient` whose transport answers, and turns models.dev off
  (`llmprovider.WithoutModelMetadata()`), as `AGENTS.md` requires. From
  Phase 6, the wizard's run is behind the guard; before it, each phase runs
  `go test ./wizard/` once behind a scratch `TestMain` on a scratch clone,
  as the MADR's probe did, and records "passed".
* **No key in a test, a log or a record.** Keys are made-up strings.
* **Coverage does not fall** (R47): `wizard` stays at or above its floor,
  86.4 %, and `catalog` and `chatcompat` at theirs.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR, or of its amendments, no longer holds;
  * an existing test other than the one in Scope would change;
  * an exported name other than the three above would be added, or a
    signature would change;
  * a doc would state a behaviour no test shows.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.

### Phase 1: control characters, in both layers (V1; Q3 (a))

1. **`catalog` drops a listed id holding a control character.** An
   unexported `hasControl(s string) bool` in `catalog`, the rule of
   `internal/redact`'s `isControl`: a C0 control but tab, DEL, or a C1
   control (U+0080 to U+009F). `catalog` may not import `internal/redact`
   (R2), so the rule is written again, and a test pins both ends of each
   range.
   * In `fetchListing` (`discovery.go:617`), every lister's result passes
     one filter before `buildListing`: `usable` loses each such id, Kilo's
     `kilo` entries and a chatcompat server's `created` the same ids.
   * In `ollamaCatalog`, after `fetchOllamaTags` (`discovery.go:910`), a tag
     whose name holds one is skipped.
   * Each drop is logged once per listing at Debug, on the listing's logger,
     with the provider and the count, never the id.
2. **`chatcompat.Discover`'s `Server.Models`** drop the same ids. The
   package has its own copy of the rule, for the same reason.
3. **The wizard's own listings.** `providerCatalog` and `chatGPTCatalog`,
   which read a provider's `ListModels`, drop an id that
   `redact.StripControl` would change. A Kilo organization's `Name` and
   `Role`, which `resolveKiloDevice` puts in a `Choice` (`auth.go:380`),
   pass through `redact.StripControl`; the id the user picks is the
   organization's `ID`, kept as it came.
4. **`TextPrompter` prints no control character from its arguments.**
   `renderChoices` strips the title, each label and each detail;
   `Select`'s and `MultiSelect`'s prompt lines carry only numbers;
   `Confirm`'s question, `Input`'s prompt and default, and `Secret`'s and
   `readMasked`'s prompt are stripped as `Notify`'s text is. The value
   `Input` returns is not changed: an accepted default is returned as the
   caller gave it. The format strings' own `\n`, `\r` and `\b` stay.
5. **Tests, red first:**
   * `TestTextPrompter_ListedIDEscapes_0066`: probe J as a test. No ESC or
     BEL byte reaches `Out`, and the Result's `Model` is `plain-model` when
     row 1 is picked. On `3fad786`: 2 ESC bytes, 1 BEL, and the evil id as
     `Model`.
   * `TestTextPrompter_StripsEveryArgument_0066`: a title, label, detail,
     question, prompt and default each holding `\x1b[2J` and `\x9b`; none
     reaches `Out`.
   * `TestList_ControlIDsDropped_0066` (`catalog`): a chatcompat listing
     (the package's test helper `withCompatID`, since `catalog` cannot
     import `chatcompat`, and a base URL on an `httptest` server) and an
     Ollama `/api/tags` reply, each with one such id among clean ones; the
     catalog's `Usable`, `Recommended` and `Menu` hold the clean ones only.
   * `TestHasControl_0066` (`catalog`, `chatcompat`): each end of C0, DEL
     and C1, tab and U+00A0.
   * `TestDiscover_ControlIDsDropped_0066` (`chatcompat`), over a transport
     that answers by port.
   * `TestKiloOrg_LabelStripped_0066` and
     `TestProviderCatalog_ControlIDsDropped_0066` (`wizard`).
6. **Docs:** `Catalog.Usable`'s doc ("an id holding a control character is
   dropped"), `chatcompat.Server.Models`', `TextPrompter`'s, and
   `CHANGELOG.md`'s `[Unreleased]` Fixed.

### Phase 2: a Chat Completions provider through the wizard (V2, V3, V4; Q1 (a))

1. **An endpoint is required where there is no default (V3).** In
   `resolveBaseURL` (`configure.go:445`), for a provider with
   `SupportsBaseURL`, no `DefaultBaseURL` and no `IsLocal`:
   * a blank answer is not taken: a notice, "<Label> needs an endpoint",
     and the prompt again;
   * under `NoEndpointPrompt` with no `Existing.BaseURL` for it, a plain
     error, "wizard: <Label> needs an endpoint: pass Options.Existing.BaseURL",
     with nothing asked.

   No built-in provider is affected: each with `SupportsBaseURL` has a
   default (their descriptors at `3fad786`).
2. **A key, when the server wants one (V2; the amendment).** For a
   provider with no `RequiresAPIKey` and no `IsLocal`:
   * `resolveAPIKey` (`configure.go:573`) offers `Existing.APIKey` for it,
     "Keep the existing key (…)?", as for a required key; a kept key is
     `keySaved`, and its refusal ends the run, naming it, as 0059-MADR Q6
     (a) keeps for a saved key;
   * otherwise the first listing runs with no key. When the listing refuses
     it, `credentialStep` notifies "<Label> refused the request without a
     key (<describe>); enter its key", asks with `promptAPIKey`, and goes on
     as for a typed key: up to `maxTypedKeys` typed keys, the first being
     the first typed. The Result's `Kind` is `CredAPIKey` and its `APIKey`
     the accepted key.
   * `refusalError` gains a case for a credential of kind `CredNone`, which
     only a local provider such as Ollama now reaches: "wizard: <Label>
     refused a request that carried no key: …". The pasted-credential case
     stays for a pasted key.
3. **`catalog.Lister` (V4; Q1 (a); the amendment's name and shape).** In
   `llmprovider/catalog`, a new file:

   ```go
   // Lister is a provider that lists its models as a Catalog, its menu,
   // facts and Source with them, where llmprovider.ModelLister gives ids
   // only (0066-MADR Q1). A caller type-asserts a Provider to it, as to
   // llmprovider.ModelLister. An implementation must be safe for
   // concurrent use, as the Provider that has it is.
   type Lister interface {
   	// ListCatalog lists the provider's models as List does, with List's
   	// options, ~~WithProfile and WithLimit among them,~~ after the
   	// provider's own. It returns List's errors.
   	ListCatalog(ctx context.Context, opts ...llmprovider.Option) (Catalog, error)
   }
   ```

   The final wording is checked by `make docs-check`; the doc names each
   kind `List` returns (R52). *`WithProfile` and `WithLimit` are scoped to
   the built-in ids, so `List` refuses them for a chatcompat id: the
   deviation of 2026-10-30 in Phase 2's record.*
4. **`chatcompat`'s provider implements it:** `catalog.List(ctx, p.id,
   p.src, append(slices.Clone(p.listing), opts...)...)`, with no probe
   (`ListModels` keeps its probe). A compile-time assertion,
   `var _ catalog.Lister = (*provider)(nil)`, and the package doc name it.
5. **The wizard lists through it.** `listerFor` (`configure.go:777`)
   returns the built `llmprovider.Provider`. `providerCatalog` asserts
   `catalog.Lister` first and calls `ListCatalog` ~~with
   `catalog.WithProfile(o.Profile)` and `catalog.WithLimit(o.ModelLimit)`~~
   *with no catalog option, and cuts `Recommended` to `ModelLimit` itself,
   as it cuts `ListModels`' ids (the deviation of 2026-10-30)*;
   without it, `ListModels` as today. `chatGPTCatalog` asserts
   `llmprovider.ModelLister` as today. In `discoverModels`, a catalog that
   is not live, holds an `Err` and is not a refusal, with no static
   catalog, gives the notice "could not list models for <Label>
   (<describe>)", as the `ListModels` path gives today.
6. **Tests, red first:**
   * `TestCompat_BlankEndpointAskedAgain_0066` and
     `TestCompat_NoEndpointPromptNeedsURL_0066`: probe A. On `3fad786`:
     no error, an empty `BaseURL`.
   * `TestCompat_KeyAskedAfterRefusal_0066`: probe B, with a server that
     answers 401 without `Authorization: Bearer lm-key` and lists with it.
     The `Secret` is asked once; the second listing request carries the
     key; the Result is `CredAPIKey`. On `3fad786`: no `Secret`, and "the
     pasted credential".
   * `TestCompat_ThreeKeysRefused_0066`: three refused keys end the run with
     "refused 3 keys", matching `ErrAuthFailure`.
   * `TestCompat_SavedKeyOffered_0066`: `Existing.APIKey` is offered and
     kept.
   * `TestOllama_RefusedNoKeyAsked_0066`: Ollama behind a server that
     answers 401: no `Secret`, the new wording, `ErrAuthFailure`.
   * `TestRefusalError_NamesTheSource`: its new row.
   * `TestCompat_ListedThroughLister_0066`: probe C. `Source` is `live`,
     `Menu` holds 30, `Recommended` 6 with `ModelLimit: 6`, `Usable` 30.
     On `3fad786`: `Source` empty and `Menu` 0.
   * `TestListCatalog_0066` (`chatcompat`): equals `catalog.List` with
     `WithID` and the base URL; ~~`WithLimit(3)` cuts `Recommended` and
     `Menu`~~ *`WithLimit(3)` is refused, an error matching
     `ErrUnsupported`, as `List` refuses it (the deviation of 2026-10-30)*;
     a 401 is `Live` false with an `Err` matching `ErrAuthFailure`.
7. **Docs:** `ConfigureLLM`'s and `ResolveCredential`'s docs (the endpoint
   error under `NoEndpointPrompt`; the key asked); `Options.NoEndpointPrompt`'s;
   the `chatcompat` package doc; `docs/guides/adding-a-provider.md`, a
   bullet: a provider that implements `catalog.Lister` gives the wizard its
   menu and facts; `docs/architecture.md`'s "The wizard"; `README.md`'s
   wizard section; `CHANGELOG.md` Added (`catalog.Lister`) and Changed.

### Phase 3: the servers running on this machine (V5; Q2 (a); the amendment's details)

1. **`Options.LocalServers bool`.** Its doc says what follows, and that a
   Result whose `Provider` the caller's `Registry` lacks is a found server,
   built with `chatcompat.New(chatcompat.WithID(res.Provider),
   llmprovider.WithBaseURL(res.BaseURL))`, with `llmprovider.WithAPIKey`
   when `res.Kind` is `CredAPIKey`.
2. **The run's registry.** When it is set, and the menu is shown (no id),
   or `ResolveCredential`'s id is not in the `Registry`, the wizard calls
   `chatcompat.Discover(ctx, llmprovider.WithHTTPClient(o.HTTPClient))`
   (the client only when set) once, before the menu. It builds a run-local
   `Registry`: each of the caller's descriptors, with a factory that calls
   the caller's `Registry.New`, then each found server:
   * id `local-<port>`; a label of the dialect's name and its address, such
     as "Ollama (127.0.0.1:11434)", or "Chat Completions server
     (127.0.0.1:1234)"; notes "running on this machine", with its model
     count or "wants a key";
   * `chatcompat.Descriptor(id, label)` with `DefaultBaseURL` the server's
     `BaseURL`, and no `IsLocal`, so its endpoint is checked as a remote
     one's, not with Ollama's `/api/version`;
   * a factory: `chatcompat.New(append(opts, chatcompat.WithID(id))...)`;
   * skipped when its port is that of a descriptor with `IsLocal` whose
     `DefaultBaseURL` names a loopback host (Ollama's `localhost:11434`).

   When `Options.Providers` is not empty, the found ids are added to it for
   the run. The caller's `Options` are not changed.
3. **The flow after the menu is Phase 2's:** the endpoint default is the
   server's `BaseURL`; a server that `NeedsKey` is refused without a key,
   and the key is asked; the catalog comes through `catalog.Lister`.
4. **Errors (R52):** `ctx` ending during `Discover` gives its error,
   wrapped; `ResolveCredential` with an id the `Registry` lacks and no
   server found under it still matches `llmprovider.ErrInvalidProvider`.
5. **Tests, red first,** each over an `HTTPClient` whose transport answers
   `GET http://127.0.0.1:<port>/v1/models` by port, so no port of the host
   is used:
   * `TestLocalServers_Offered_0066`: a server on `1234` listing two
     models is a menu row after the registry's; picked, its endpoint
     default is its `BaseURL`, and the Result is `local-1234` with that
     `BaseURL`, which the documented `chatcompat.New` call builds. On
     `3fad786` the option does not exist; the test is planted against a
     scratch clone with the field and no menu change, and fails.
   * `TestLocalServers_OllamaPortSkipped_0066`, `…_NeedsKeyAsksKey_0066`,
     `…_OffAsksNothing_0066` (no request to any port),
     `…_ResolveByID_0066`, `…_NotRunningIsInvalidProvider_0066`,
     `…_EndedContext_0066`, and `…_ProvidersAddsFound_0066`.
6. **Docs:** `Options.LocalServers`, `ConfigureLLM`'s and
   `ResolveCredential`'s errors; `docs/architecture.md`'s package table,
   `wizard`'s imports gaining `providers/chatcompat` (R43), and "The
   wizard"; the `chatcompat` package doc ("the wizard's
   `Options.LocalServers` offers them"); `README.md`; `CHANGELOG.md`
   Added.

### Phase 4: the unchecked credential and the refused kept session (V6, V7; Q5 (a), Q4 (a))

1. **`Err` on every path (V6).** In `discoverModels`, the fallback returned
   for a failed listing carries the failure: `fallback.Err = err` before
   the returns at `configure.go:692` and `:695`, and `fallback.Err =
   cat.Err` at `:700`. `ResolveCredential`'s doc says: ~~`Live` false is a
   credential no live listing checked, and `Err` holds why when a listing
   ran and failed; `Live` true is a listing the provider answered with this
   credential, now or within the listing cache's 24 hours,~~ *`Live` true
   is a listing the provider answered with this credential, now or within
   the listing cache's 10 minutes, or, after a passing failure, its last
   good listing from within 24 hours, `Source` `ListStale` and `Err` the
   failure; `Live` false is the static catalog: `Discover` off, a listing
   that failed, `Err` saying why, or one that listed nothing usable, `Err`
   nil (the deviation of 2026-10-31),* and for a
   listing that needs none, such as Hugging Face's, says nothing of the
   key. `Options.Discover`'s doc points to it.
2. **A refused kept session leads to the sign-in menu (V7).** A credential
   kept by `keepExistingOAuth` is marked kept. When the listing refuses it,
   `credentialStep` notifies "the saved <Label> session was refused
   (<describe>); sign in again", and runs `signIn` again, once, without
   offering the kept session (an unexported parameter of `signIn`). The
   store is not touched until the new session is saved (0059-MADR W2). A
   refusal of the new one ends the run as today.
3. **Tests, red first:**
   * `TestResolve_FailedListingErr_0066`: probe L's three rows and probe
     D's, each with `Live` false and an `Err` matching the failure; without
     `Discover`, `Err` nil. On `3fad786`: two rows have a nil `Err`.
   * *`TestResolve_LiveMeans_0066`: a listing answered with no usable id
     is `Live` false with a nil `Err`; one read again at once, the server
     now failing, is `Live` true with no request (the deviation of
     2026-10-31).*
   * `TestKept_RefusedLeadsToSignIn_0066`: probe G, then a stubbed browser
     sign-in whose session the listing accepts: the method menu is shown,
     the new session is saved, and the Result names it. On `3fad786`: the
     run ends, no menu.
   * `TestKept_RefusedThenNewRefused_0066`: the second refusal ends the run
     with "refused the sign-in", and the store still holds the kept
     session.
   * `TestKept_KiloRefusedLeadsToSignIn_0066`: the same for a Kilo device
     session.
4. **Docs:** the two docs above; `README.md`'s wizard section;
   `CHANGELOG.md` Changed.

### Phase 5: the notices (V8, V9, V10; *V17, added 2026-10-31*)

1. **The endpoint check (V8).** At `configure.go:489`, an error that is an
   `*llmprovider.APIError` gives "<URL> answered: <describe>", and any other
   "cannot reach <URL> (<describe>)". The error returned under
   `NoEndpointPrompt` keeps its text, as 0056-MADR D3 allows.
2. **A refused URL (V9).** `validateBaseURL` shows a URL without its user
   information, query or fragment: a helper gives
   `scheme://host/path`, then `?…` for a query and `#…` for a fragment. A
   value that is not a URL, or lacks a scheme or host, is not echoed: "the
   endpoint is not a URL", "an endpoint needs http:// or https:// and a
   host". The plain-http notice and question use the same helper and keep
   their words, "plain http to a remote host", which
   `edges_0059_test.go` reads.
3. **Offline (V10).** `errorClasses` gains `llmprovider.ErrOffline` before
   `llmprovider.ErrProviderUnavailable`; `describe` then says "offline".
4. **Tests, red first:**
   * `TestEndpointCheck_NoServerText_0066`: probe E. The notice holds
     "provider unavailable, HTTP 500" and not `SERVER-TEXT-0066`.
   * `TestBaseURL_QueryNotShown_0066`: probe F, and rows for a fragment, a
     user and password, no scheme, and a non-URL; no notice holds the
     planted value.
   * `TestDescribe_Offline_0066`: probe D's `describe`, and its notice
     "could not list models … (offline)" or "unavailable (offline)".
5. **Docs:** `validateBaseURL`'s and `describe`'s; `CHANGELOG.md` Fixed.
6. *Added 2026-10-31, on the owner's word (the MADR's amendment "V17"):*
   **A silent fallback (V17).** In `discoverModels`, a catalog with nothing
   recommended that carries a failure gives "could not list models for
   <Label> (<describe>); using the built-in catalog" before the fallback
   is returned. A listing that answered with no usable id stays as it is.
   Its test, red first: `TestListing_StaticBesideNoticed_0066`, a Chat
   Completions server whose descriptor has a static catalog, answering
   502: the notice once, `Err` matching `ErrProviderUnavailable`, the
   static catalog. On `5c11035`: no notice. `CHANGELOG.md` Fixed.

### Phase 6: the menu, the guide and the tests (V11, V12, V13, V14; Q6 (a))

1. **An unknown `Providers` id (V11).** `selectableDescriptors` returns the
   ids the registry lacks; `chooseProvider` notifies once, "no provider
   <id> here; it is left out of the menu", for each. When none exists the
   error is as today, which `configure_errors_0048_test.go` reads.
2. **The guide (V12).** `docs/guides/adding-a-provider.md`: "With
   `Options.ModelLimit` at zero, every listed id is recommended, in your
   order; a positive `ModelLimit` recommends that many. Search covers the
   whole listing."
3. **The guard (V13).** `wizard/localonly_test.go`, built without
   `live_gateways`, with `TestMain` calling `localonly.Main`, as the six
   packages of 0065-MADR D2 have.
4. **The two paths (V14).**
   * `TestFallbacks_CapMatches_0066`: a fallback search with 25 matches
     shows 20, with the notice "showing 20 of 25 matches".
   * `TestSave_StoreFails_0066`: a `TokenStore` whose `Save` fails, for a
     Grok sign-in and a Kilo device sign-in: "wizard: save OAuth session"
     and "wizard: save Kilo login", each wrapping the store's error.
5. **Red first:** probe H as `TestProviders_UnknownNamed_0066`, failing on
   `3fad786` (no notice). For V13, `tools/plant.py` adds the MADR's planted
   leak to a scratch clone: the guarded run fails naming
   `api.anthropic.com:443`. V14's tests cover code that is right today, so
   each is seen to fail on a planted break: `capMatches` cutting at 21, and
   `save` dropping the store's error.
6. **Docs:** `Options.Providers`'; the packages the guard runs in, which
   gain `wizard`: `docs/architecture.md`'s `internal/localonly` row
   (`docs/architecture.md:109` at `3fad786`) and `AGENTS.md`'s paragraph
   that lists them; `CHANGELOG.md` Fixed.
7. **Coverage:** `capMatches` 100 %; `save` 90 % or more.

### Phase 7: a bounded client that headers can wrap (V15, V16; issue #10; Q7 (a))

The MADR's amendment "issue #10", D2. This phase runs once Q7 is answered;
an answer other than (a) is a deviation, and this phase is rewritten before
it starts.

1. **`NewHTTPClient`.** In `llmprovider/options.go`, beside
   `WithHTTPClient`: `func NewHTTPClient() *http.Client`, returning
   `transport.DefaultClient()`. Its doc, as D2 words it: a new client per
   call, over the transport the process shares; the bounds (30 s to
   connect, 10 s for the TLS handshake, 300 s to the first byte, idle
   connections closed after 90 s, 16 kept per host, no total timeout); the
   client a provider builds when given none; its `Transport` a
   `RoundTripper` to wrap, not an `*http.Transport` to configure; safe for
   concurrent use, as an `http.Client` is (R51); no error (R52).
2. **The documentation 0061-MADR D6 promised (V16).**
   * `WithHTTPClient`'s doc (`options.go:44`): a given client serves the
     provider's requests, its listings and a session's refresh (R32); to add
     headers, wrap `NewHTTPClient()`'s transport in a `RoundTripper` and give
     that client; what the transport needs per request, such as the
     request's model, goes in the request's context.
   * The `llmprovider` package doc (`provider.go`): the same, in one
     sentence, as D6 said.
   * The wizard's `Options.HTTPClient` doc: a client with headers is built
     the same way, and every listing, sign-in and session of the run uses
     it.
   * `README.md`: a short paragraph, "Adding request headers", with the
     wrapping shown in four lines.
3. **Tests, red first:**
   * `TestNewHTTPClient_Bounds_0066` (`llmprovider`): its `Transport`,
     asserted to `interface{ Base() *http.Transport }` as
     `options_test.go:22` does, is the very transport a provider built with
     no client uses, with `ResponseHeaderTimeout` 300 s,
     `TLSHandshakeTimeout` 10 s, `IdleConnTimeout` 90 s and
     `MaxIdleConnsPerHost` 16. Seen to fail on a scratch clone where
     `NewHTTPClient` returns `&http.Client{}`.
   * `TestNewHTTPClient_NewEachCall_0066`: two calls give two clients over
     one transport; setting a field of one leaves the other as it was.
   * `TestWrappedClient_Headers_0066` (`providers/chatcompat`, which can
     build a provider over an `httptest` server): a provider given
     `WithHTTPClient` of a `NewHTTPClient` whose transport adds `X-Test`
     from the request's context. A generation and a listing each carry the
     header; a transport that refuses to resolve it ends the request, with
     no request reaching the server. Seen to fail on a planted break: the
     wrapper not installed.
   * `TestOptions_HTTPClientHeaders_0066` (`wizard`): the same client as
     `Options.HTTPClient`; the listing's request carries the header.
   * `ExampleNewHTTPClient` (`llmprovider_test`, R50): the wrapping, a
     request to an `httptest` server, and `// Output:` printing the header
     the server saw.
4. **Docs and gates:** `CHANGELOG.md` Added (`llmprovider.NewHTTPClient`)
   and Fixed (D6's missing sentence); ~~`make api-check` lists
   `NewHTTPClient`~~ *`tools/apidiff.py v1.24.1` lists `NewHTTPClient`, and
   `make api-check` shows no incompatible change (the deviation of
   2026-10-31)*; `make docs-check` passes for it.

### Phase 8: release `v1.25.0`

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.25.0] - <date>`, dated
   by the commit the tag will name, with Added (`catalog.Lister`,
   `wizard.Options.LocalServers`, `llmprovider.NewHTTPClient`), Changed and
   Fixed from Phases 1 to 7, and
   the docs-gates entry already under `[Unreleased]`. An empty
   `[Unreleased]` goes above it, and the comparison links are updated. The
   README's Status names `v1.25.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.24.1`, listing the
   three additions and nothing else; the notes from
   `scripts/check_changelog.py --section v1.25.0` on a local clone with a
   local tag, scanned; the disclosure guard.
3. **The tag,** on the owner's word: `v1.25.0`, annotated, pushed. Every
   `validate` job passes, and the `release` job's body equals step 2's
   notes.
4. **After the tag:** `gh release delete v1.22.1 --yes`, the Release that
   falls to sixth (R53; the five newest are `v1.24.1`, `v1.24.0`,
   `v1.23.0`, `v1.22.2` and `v1.22.1` on 2026-10-30). `tools/consumer.py
   v1.25.0` and `tools/offline.py v1.25.0`.
5. **Issue #10,** on the owner's word: a comment naming `v1.25.0` and
   `NewHTTPClient`, and the issue closed.
6. **Close:** the MADR's Confirmation, item by item; this PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| 1 | No control character from a server reaches the terminal, a menu or the Result | 1 | `TestTextPrompter_ListedIDEscapes_0066`, `…_StripsEveryArgument_0066`, `TestList_ControlIDsDropped_0066`, `TestDiscover_ControlIDsDropped_0066`, `TestKiloOrg_LabelStripped_0066`, `TestProviderCatalog_ControlIDsDropped_0066` |
| 2 | A provider with no default endpoint is given one | 2 | `TestCompat_BlankEndpointAskedAgain_0066`, `TestCompat_NoEndpointPromptNeedsURL_0066` |
| 3 | A keyed server gets its key; a local provider is not asked one | 2 | `TestCompat_KeyAskedAfterRefusal_0066`, `…_ThreeKeysRefused_0066`, `…_SavedKeyOffered_0066`, `TestOllama_RefusedNoKeyAsked_0066`, `TestRefusalError_NamesTheSource` |
| 4 | A Chat Completions catalog has its `Source`, `Menu` and limit | 2 | `TestCompat_ListedThroughLister_0066`, `TestListCatalog_0066` |
| 5 | Found servers are offered and build | 3 | `TestLocalServers_*_0066` |
| 6 | `Err` on every failed path; the doc names the signal | 4 | `TestResolve_FailedListingErr_0066` |
| 7 | A refused kept session leads to sign-in once | 4 | `TestKept_RefusedLeadsToSignIn_0066`, `…_RefusedThenNewRefused_0066`, `…_KiloRefusedLeadsToSignIn_0066` |
| 8 | No service text, no URL query, offline named | 5 | `TestEndpointCheck_NoServerText_0066`, `TestBaseURL_QueryNotShown_0066`, `TestDescribe_Offline_0066` |
| 9 | Unknown ids named; guide right; guard on; two paths tested | 6 | `TestProviders_UnknownNamed_0066`, the guide's text, `wizard/localonly_test.go` with its plant, `TestFallbacks_CapMatches_0066`, `TestSave_StoreFails_0066` |
| 10 | Exactly three exported names added; no signature changed | 2, 3, 7, 8 | `make api-check`, `tools/apidiff.py v1.24.1` |
| 11 | Docs carry R41, R51, R52 for the new names | 2, 3, 7 | `make docs-check` |
| 12 | Every new test failed first | 1–7 | the execution record |
| 13 | Coverage at or above each floor | 1–7 | `make coverage-check` |
| 14 | A client with headers keeps the SDK's bounds and shared transport | 7 | `TestNewHTTPClient_Bounds_0066`, `…_NewEachCall_0066` |
| 15 | Headers from a wrapped `NewHTTPClient` reach requests, listings and the wizard; the way is documented | 7 | `TestWrappedClient_Headers_0066`, `TestOptions_HTTPClientHeaders_0066`, `ExampleNewHTTPClient`, the docs of `WithHTTPClient`, the package and `Options.HTTPClient` |

## Rollout and Rollback

* **Rollout.** Nine commits on `main`, staged by the agent and committed by
  the owner: the records, one per phase, and the release commit. Then the
  owner's tag. gobble-cli takes `v1.25.0` by its own record; nothing it
  calls today changes its result, except notices' wording and, on its
  `ListModels` paths, a catalog `Err` that was nil.
* **Rollback.**
  * Before the tag, each phase reverts on its own, newest first; Phase 3
    needs Phase 2. Phase 7 stands alone.
  * After the tag, an exported name stays (R48): a fault in Phase 2 or 3
    is fixed forward in a patch, and `LocalServers` can be documented as
    doing nothing, never removed, before `v2`.
  * Reverting Phase 1 brings back escape sequences in menus; Phase 2, the
    blank endpoint and the unanswerable keyed server.

## Execution record

Approved by the owner's "committed, proceed", 2026-10-30, after the records
landed as `dafea26`. Phase 0 rides with Phase 1's commit, as the eight-commit
rollout counts it.

### Phase 0: records (2026-10-30)

`tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`;
each prints "records: written; records-check: 0 problem(s)", and the index's
two rows follow.

### Phase 1: control characters, in both layers (2026-10-30)

* **Red first, on a scratch clone of `dafea26`** (the code of `3fad786`),
  with the new tests copied in and the two `hasControl` tests left out,
  their function not existing there. Each package exits 1:
  * `catalog`: `TestList_ControlIDsDropped_0066/chatcompat`, "Usable =
    ["clean-a" "evil\x1b]0;owned\a\x1b[2Jmodel" "clean-b" "c1\u009bx"];
    want ["clean-a" "clean-b"]", and the same of `Recommended` and `Menu`;
    `…/ollama`, "Usable = ["clean-a" "evil\x1b[2J" "clean-b" "del\x7f"]".
  * `chatcompat`: `TestDiscover_ControlIDsDropped_0066`, "Models =
    ["clean-a" "evil\x1b]0;owned\a\x1b[2Jmodel" "clean-b" "c1\u009bx"]; want
    the clean ids only".
  * `wizard`: `TestTextPrompter_ListedIDEscapes_0066`, "2 ESC and 1 BEL
    bytes reached the terminal; want none" and "Model =
    "evil\x1b]0;owned\a\x1b[2Jmodel"; want plain-model";
    `TestTextPrompter_StripsEveryArgument_0066`, "Out holds a control
    character: "\ntitle\x1b[2J\u009b\n  1) label\x1b[2J\u009b — …";
    `TestKiloOrg_LabelStripped_0066`, "menu row "Ev\x1b]0;x\ail" /
    "ad\u009bmin" holds a control character";
    `TestProviderCatalog_ControlIDsDropped_0066`, "Usable ["acme-a"
    "ev\x1b[2Jil" "acme-b" "c1\u009bx"] … want ["acme-a" "acme-b"]".
* **The `hasControl` tests, on planted breaks** (`tools/plant.py`, scratch
  clone of the working tree): the C1 range removed from each copy, "all 2
  plant(s) caught", each with `hasControl("c1\u0080") = false; want true`,
  and the same for `\u009b` and `\u009f`.
* **The code:**
  * `catalog`: `control.go`, with `hasControl`,
    `fetchedListing.withoutControl` and `logDropped`; `fetchListing` now
    calls `fetchByProvider`, the former body, and filters its result before
    the listing cache keeps it; `ollamaCatalog` drops such a tag after
    `fetchOllamaTags`. `Catalog.Usable`'s doc says so.
  * `chatcompat`: `hasControl`, used where `askPort` collects
    `Server.Models`, whose doc says so.
  * `wizard`: `withoutControl` on the ids of a provider's own `ListModels`,
    in `providerCatalog` and `chatGPTCatalog`; a Kilo organization's name
    and role through `redact.StripControl` (its id kept); `TextPrompter`
    strips a menu's title, labels and details, `Confirm`'s question,
    `Input`'s prompt and shown default, and `Secret`'s prompt, and its type
    doc says so. An accepted default is returned as given.
  * `CHANGELOG.md`: `[Unreleased]` Fixed.
* **Green:** `go test -count=1 -run _0066` on the three packages: `ok` each.
* **The drift check:** `go test -race -count=1 -shuffle=on ./wizard/
  ./llmprovider/catalog/ ./llmprovider/providers/chatcompat/`: exit 0, every
  existing test unchanged.
* **No network:** `./wizard/` behind a scratch `TestMain` calling
  `localonly.Main`, on a scratch clone of the working tree: `ok`, exit 0.
* **Pre-add:** `make pre-add-check` on the phase's nine Go files: "9 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)", exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while other sessions held the load
  near 86 on ten CPUs, started at 9.5, and printed "gate: 29 of 29 passed",
  exit 0: `pre-add-check`, `vet` on darwin, linux and windows with and
  without `live_gateways`, `test-race-cover`, `test-shuffle`,
  `test-repeat`, `tidy-diff`, `lint`, and every `make` check CI runs, with
  `deny-scan` and `tools-test`.
* **Notes, not deviations** (no behaviour, decision or test outside Scope
  changes):
  * `TestDiscover_ControlIDsDropped_0066` uses `chatcompat`'s own
    `withPorts` and a loopback stub server, as its other `Discover` tests
    do, where step 5 said "over a transport that answers by port". Discover
    asks `127.0.0.1` only, so the stub is on the machine.
  * Step 2 said `chatcompat` keeps its own copy of the rule "for the same
    reason" as `catalog`. R2 names `internal/redact` for `wizard` and the
    wire packages, and gives a provider package "internal packages" without
    naming it. No provider package imports it today. The copy stands, as
    step 2 says, and its comment states that fact, not an R2 rule.
  * `[Unreleased]`'s earlier Changed entry, on the documentation gates,
    ends "The module's code is unchanged", which was true of that change
    alone. Phase 8 words it so when the section becomes `v1.25.0`.

### Phase 2: a Chat Completions provider through the wizard (2026-10-30)

* **Deviation 2026-10-30: the Lister's options.** Found before Phase 2's
  code, in writing and then running its tests red. Two questions were put
  to the owner, the first on a fact found incomplete.
  * **The first question, 2026-10-30.** Step 6 says that with
    `ModelLimit: 6` the catalog's `Menu` holds 30; `List` cuts `Menu` with
    `WithLimit` (`llmprovider/catalog/menu.go:225`, `cutMenu`), and
    `Catalog.Search` finds a model cut from `Menu` "by its id and label
    alone" (`llmprovider/catalog/search.go:199`). The owner chose "correct
    the records": pass `WithLimit` to `ListCatalog`, as the wizard passes it
    to `List` for a built-in, and assert `Menu` 6.
  * **What the red run showed.** `catalog.List` with `chatcompat.WithID`
    and `catalog.WithLimit(6)` returned "model listing: unsupported
    provider "lmstudio"". `WithProfile` and `WithLimit` are scoped to the
    built-in ids (`llmprovider/catalog/config.go:112`, `:124`), and `List`
    refuses a chatcompat id whose options hold one
    (`llmprovider/catalog/config.go:241`). So step 5's call would fail every
    Chat Completions listing, and step 3's Lister doc, "WithProfile and
    WithLimit among them", is false for chatcompat. The first answer could
    not be built as it was put.
  * **The second question, 2026-10-31,** with that fact, and three
    resolutions: the wizard cuts `Recommended` itself (recommended); widen
    the two options' scope, a change to two exported options needing its
    own amendment; or stop Phase 2. The owner chose "the wizard cuts it".
  * **What changes.** `providerCatalog` calls `ListCatalog` with no catalog
    option and cuts `Recommended` to `ModelLimit`, as it cuts `ListModels`'
    ids; `Menu` and `Usable` stay whole, so search keeps every model's
    facts. Step 6's numbers hold: `Menu` 30, `Recommended` 6, `Usable` 30.
    `TestListCatalog_0066` asserts that `WithLimit(3)` is refused, matching
    `ErrUnsupported`, in place of a cut. The Lister doc names its options
    as those `List` takes for the provider's id. Steps 3, 5 and 6 are
    annotated, and the MADR has the amendment "the Lister's options". No
    file is added to the phase's scope.
* **Deviation 2026-10-31: an existing test with no default endpoint.**
  The drift check (`go test -race -count=1 -shuffle=on`, after the code)
  failed `TestResolveBaseURL_Validates` (`wizard/configure_test.go:391`),
  its row "blank keeps the default": "resolveBaseURL = "", enter base URL:
  fakePrompter: unexpected Input("Kilo endpoint")". The test builds its own
  descriptor, id `kilo`, label "Kilo", `SupportsBaseURL` and no
  `DefaultBaseURL`: the case V3 now refuses. The real Kilo descriptor has a
  default (`llmprovider/providers/kilo/descriptor.go:12`), and the row came
  with `3f18dd7`, before V3. Put to the owner with three resolutions; the
  owner chose "give it a default": the test's descriptor gains
  `DefaultBaseURL: "https://api.kilo.ai/api/gateway"`, the real Kilo's, and
  every row and assertion stays, so "blank keeps the default" tests what it
  names. The test is added to Scope's "One existing test gains rows" item.
* **Red first, on a scratch clone of `2ab63be`**, Phase 1's commit, whose
  code on these paths is `3fad786`'s, with the phase's wizard tests copied
  in. `./wizard/` exits 1:
  * `TestCompat_BlankEndpointAskedAgain_0066`: "BaseURL "", Live false",
    "inputs ["LM Studio endpoint"]", "notices ["could not list models for
    LM Studio (invalid request)"]; want "LM Studio needs an endpoint"".
  * `TestCompat_NoEndpointPromptNeedsURL_0066`: "err = <nil>; want
    "wizard: LM Studio needs an endpoint: pass Options.Existing.BaseURL"".
  * `TestCompat_KeyAskedAfterRefusal_0066`: "wizard: LM Studio refused the
    pasted credential: llmprovider: authentication failed: lmstudio HTTP
    401 …", a select and the secret left unused.
  * `TestCompat_ThreeKeysRefused_0066`: the same error, "1 listing
    requests; want 4", three secrets unused.
  * `TestCompat_SavedKeyOffered_0066`: `kept` and `refused` each "confirms
    []; want the keep question", "listing Authorization headers [""]".
  * `TestOllama_RefusedNoKeyAsked_0066`: "wizard: Ollama (local) refused
    the pasted credential: … ollama HTTP 401: unauthorized".
  * `TestCompat_ListedThroughLister_0066`: "Source "", Menu 0, Recommended
    6, Usable 30; want live, 30, 6, 30", probe C's numbers.
  * `TestRefusalError_NamesTheSource`, its new row: "wizard: Claude refused
    the pasted credential: …; want … "refused a request that carried no
    key"".
  * `TestCompat_ListingFailureNoticed_0066`, added for step 5's notice,
    passes there: the old path gave the same notice through `ListModels`.
    Its plant below is its proof. `TestResolveBaseURL_Validates`, with its
    default, passes there too.
* **Planted breaks** (`tools/plant.py`, scratch clone of the working tree),
  "all 4 plant(s) caught":
  * `ListCatalog` dropping the caller's options:
    `TestListCatalog_0066`, "WithLimit(3): err <nil>; want ErrUnsupported,
    as List refuses it".
  * The wizard not cutting a Lister's `Recommended`:
    `TestCompat_ListedThroughLister_0066`, "Source "live", Menu 30,
    Recommended 30".
  * The wizard skipping the Lister: the same test, "Source "", Menu 0".
  * No notice for a failure a catalog carries:
    `TestCompat_ListingFailureNoticed_0066`, "notices []; want "could not
    list models for LM Studio (provider unavailable, HTTP 502)"".
* **The code:**
  * `catalog`: `lister.go`, `Lister` with `ListCatalog`; its doc names
    `ErrUnsupported` for an option `List` refuses for the id (R52) and says
    an implementation is safe for concurrent use (R51).
  * `chatcompat`: `ListCatalog`, `catalog.List` with the provider's
    options and the caller's after them, no probe; `var _ catalog.Lister =
    (*provider)(nil)`; the package doc.
  * `wizard`: `resolveBaseURL` refuses a blank endpoint for a remote
    provider with no default, a notice and the prompt again, or under
    `NoEndpointPrompt` the error "wizard: <Label> needs an endpoint: pass
    Options.Existing.BaseURL". `resolveAPIKey` offers such a provider
    `Existing`'s key (`keepExistingKey`, the required key's question
    factored out), else returns `keyAbsent`, a new `keySource`; `signIn`
    makes a given key `CredAPIKey`. `credentialStep` answers a refusal of
    `keyAbsent` with the notice "<Label> refused the request without a key
    (<describe>); enter its key" and `promptAPIKey`, the first key asked
    being the first typed, and sets the `Result`'s `Kind`. `refusalError`
    has the `CredNone` case. `listerFor` returns the built provider;
    `modelLister` asserts `llmprovider.ModelLister`; `providerCatalog`
    lists through `catalog.Lister` first, `Recommended` cut to
    `ModelLimit`; `discoverModels` gives the notice for a catalog that is
    not live, holds an `Err`, and has no static catalog.
  * Docs: `ConfigureLLM`'s, `ResolveCredential`'s and
    `Options.NoEndpointPrompt`'s; `docs/guides/adding-a-provider.md`
    (`catalog.Lister`, an optional key, no default endpoint);
    `docs/architecture.md`'s "The wizard"; `README.md`'s wizard section;
    `CHANGELOG.md` `[Unreleased]` Added and Changed.
* **Green:** each new test `--- PASS`, by name, in `go test -v`.
* **The drift check:** `go test -race -count=1 -shuffle=on ./wizard/
  ./llmprovider/catalog/ ./llmprovider/providers/chatcompat/`: exit 0, after
  the second deviation's one line; before it, the failure that deviation
  records.
* **No network:** `./wizard/` behind a scratch `TestMain` calling
  `localonly.Main`, on a scratch clone of the working tree: `ok`, exit 0.
* **The API:** `tools/apidiff.py v1.24.1`: "./llmprovider/catalog.Lister:
  added", the only change; `make api-check`: "0 incompatible change(s)";
  `make docs-check`: "0 violation(s) of R50,R51,R52", after `Lister`'s doc
  was rewrapped so that "concurrent use" sits on one line, which R51's
  check reads.
* **Pre-add:** `make pre-add-check` on the phase's eight Go files: "8
  file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)",
  exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while other sessions held the load
  near 51 on ten CPUs, started at 9.9, and printed "gate: 29 of 29 passed",
  exit 0, `coverage-check` among them.

### Phase 3: the servers running on this machine (2026-10-31)

Approved by the owner's "committed, proceed", after Phase 2 landed as
`ecd3668`. No deviation.

* **The tests,** in `wizard/localservers_0066_test.go`, each over an
  `HTTPClient` whose transport answers `http://127.0.0.1:<port>` by port
  and refuses any other, so no port of the host is used: the eight step 5
  names. `TestLocalServers_OllamaPortSkipped_0066`, `…_OffAsksNothing_0066`
  and `…_ProvidersAddsFound_0066` check the first menu, where the run ends,
  their prompter holding no answer.
* **Red, by planted breaks** (`tools/plant.py`, scratch clone of the
  working tree), "all 9 plant(s) caught". The option does not exist on
  `3fad786`, so the first plant is step 5's: the field, and no menu
  change.
  * The field and no menu change: `TestLocalServers_Offered_0066` and
    `…_NeedsKeyAsksKey_0066`, "select provider: fakePrompter: scripted
    index 10 out of range"; `…_OllamaPortSkipped_0066`, the ten built-ins
    alone; `…_ResolveByID_0066`, `…_ProvidersAddsFound_0066`.
  * Ollama's port not skipped: "… "Ollama (local)" "Chat Completions server
    (127.0.0.1:11434)" …".
  * Found ids not added to `Providers`: "menu ["Claude (Anthropic)"]".
  * The caller's `Providers` grown in place: "the caller's Providers
    became ["claude"], its spare capacity ["claude" "local-1234" "" ""]".
  * An ended `ctx` not returned: "err = select provider: …; want
    context.Canceled".
  * `ResolveCredential` looking only for an empty id: "wizard: no provider
    "local-1234" in the registry", and "no port was asked".
  * A keyed server's row not told: "row detail "running on this machine"".
  * The endpoint default not the server's: "unexpected Input("Chat
    Completions server (127.0.0.1:1234) endpoint")".
  * `Discover` not given the caller's client: the menu without the server.
    Under this plant alone, `Discover` used its default client and asked
    this machine's loopback ports, in the scratch clone, for 0.12 s; it
    found none, and no request left the machine.
* **The code:** `Options.LocalServers`, its doc with the `chatcompat.New`
  call that builds a found server; `wizard/local_servers.go`:
  `withLocalServers`, a run-local `Registry` of the caller's descriptors,
  each built by the caller's `Registry.New`, then each found server, id
  `local-<port>`, label `serverLabel` ("Chat Completions server
  (127.0.0.1:1234)", or the dialect's name and the address), notes
  `serverNotes` ("running on this machine, 2 models", or "…, wants a
  key"), `DefaultBaseURL` its `BaseURL`, skipped on `localPort`'s port; a
  found id the caller registered itself stays the caller's. `Providers` is
  cloned before it grows. `ConfigureLLM` calls it before the menu;
  `ResolveCredential` when its id is empty or not in the `Registry`. An
  ended `ctx` is "wizard: look for the servers on this machine: …",
  wrapping ctx's error.
* **Docs:** `ResolveCredential`'s errors (an id neither the `Registry`'s
  nor a found server's); `ConfigureLLM`'s already say that an ended `ctx`
  gives ctx's error, which covers `Discover`. The `chatcompat` package doc;
  `docs/architecture.md`'s package table, `wizard` gaining
  `providers/chatcompat`, and "The wizard"; `README.md`; `CHANGELOG.md`
  Added.
* **Green:** the eight `--- PASS`, by name.
* **The drift check:** `go test -race -count=1 -shuffle=on ./wizard/
  ./llmprovider/catalog/ ./llmprovider/providers/chatcompat/`: exit 0, no
  existing test changed.
* **No network:** `./wizard/` behind a scratch `TestMain` calling
  `localonly.Main`, on a scratch clone of the working tree: `ok`, exit 0.
* **The API:** `tools/apidiff.py v1.24.1`: "./wizard.Options.LocalServers:
  added" and "./llmprovider/catalog.Lister: added", nothing else; `make
  api-check`, `docs-check` and `tables-check`: 0 problems.
* **Pre-add:** `make pre-add-check` on the phase's four Go files: "4
  file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)",
  exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while other sessions held the load
  near 34 on ten CPUs, started at 9.9, and printed "gate: 29 of 29 passed",
  exit 0, `coverage-check` and `tables-check` among them.

### Phase 4: the unchecked credential and the refused kept session (2026-10-31)

Approved by the owner's "committed, proceed", after Phase 3 landed as
`df0a9b9`.

* **Deviation 2026-10-31: what `Live` says.** Found after the phase's code,
  before its docs. Step 1's doc sentence does not hold in two parts, each
  shown by a scratch probe through `ResolveCredential` on a copy of the
  working tree (`TestProbe0066_LiveMeans`, Grok, a key, an `HTTPClient`
  whose transport answers):
  * "`Live` false is a credential no live listing checked": Grok answering
    200 with `{"data":[]}` gave "Live=false Source="static" Err=<nil>
    Recommended=5", and the notice "live model listing for Grok (xAI) is
    unavailable; search covers the built-in catalog only". The credential
    was accepted; a listing with no usable id is replaced by the static
    catalog (`llmprovider/catalog/discovery.go:528`).
  * "within the listing cache's 24 hours": a listing is served again for
    10 minutes (`listingMetadataTTL`,
    `llmprovider/catalog/listing_cache.go:20`, `:120`). The second run, the
    server now answering 503, sent no request ("requests=1" after both) and
    gave "Live=true Source="live" Err=<nil>". 24 hours is `lastGoodWindow`
    (`:56`), served only after a passing failure, a 5xx, 408, rate limit,
    timeout or network error, with `Source` `ListStale` and `Err` the
    failure (`llmprovider/catalog/discovery.go:355`;
    `TestList_StaleOn503_0041`), or the listing `catalog.WithLastGood`
    hands in.

  Both held before this phase; only step 1's sentence was wrong. Put to the
  owner with three resolutions: the doc as the code behaves (recommended);
  a narrower doc that points to `catalog.Catalog`; or the code now and the
  doc in Phase 6. The owner chose "the doc as the code behaves". Step 1's
  sentence is struck through and replaced, step 3 gains
  `TestResolve_LiveMeans_0066`, and the MADR has the amendment "what `Live`
  says". No file is added to the phase's scope, and no behaviour changes.
* **Notes, not deviations.**
  * Step 1's lines are `3fad786`'s. On `df0a9b9` the fallback's returns are
    the two for a listing that returned its failure, the one Phase 2 added
    for a failure a catalog carries with no static catalog, and the one for
    a catalog with nothing recommended. `fallback.Err = err` precedes the
    first two, and one `fallback.Err = cat.Err` precedes the other two.
  * `TestResolve_FailedListingErr_0066` has a fifth row, step 3 names four:
    a Chat Completions server whose descriptor has a static catalog,
    answering 502. It reaches the return for a catalog with nothing
    recommended, which no other row does.
* **Red first, on a scratch clone of `df0a9b9`**, Phase 3's commit, with the
  phase's test file copied in. `./wizard/` exits 1:
  * `TestResolve_FailedListingErr_0066`: its rows "a provider's
    ListModels, HTTP 500", "a Chat Completions server, HTTP 502" and "… beside
    a static catalog, HTTP 502", each "Err = <nil>; want one matching
    llmprovider: provider unavailable". "Claude offline" (probe D) and "no
    Discover" pass: they held before.
  * `TestKept_RefusedLeadsToSignIn_0066`: "wizard: Grok (xAI) refused the
    sign-in; sign in again: model listing: acquire token: llmprovider:
    authentication failed: oauth: refresh failed: Unauthorized:
    {"error":"invalid_grant"}", probe G's.
  * `TestKept_RefusedThenNewRefused_0066`: "notices []; want one that the
    saved session was refused, by class", "selects []; want the method
    menu once".
  * `TestKept_KiloRefusedLeadsToSignIn_0066`: "wizard: Kilo Gateway refused
    the sign-in; sign in again: … oauth: no refresh token: llmprovider:
    authentication failed".
  * `TestResolve_LiveMeans_0066` passes: it pins the catalog's behaviour,
    which held before; its plants below fail it.
* **Planted breaks** (`tools/plant.py`, scratch clone of the working tree):
  "all 12 plant(s) caught".
  * The fallback without the error a listing returned: the ListModels row,
    "Err = <nil>".
  * Without the error the catalog carries: both Chat Completions rows.
  * A kept session not marked kept, the plan's red: the three kept tests,
    as on `df0a9b9`.
  * The kept session offered again: "fakePrompter: unexpected
    Confirm("Keep the existing session (••••••••0066)?")", in each.
  * The `Result` not given the new credential: "Result{… TokenExpiry: …
    Model:grok-4.5}; want the new session's", and the Kilo test's
    "Organization "org-kept"".
  * The kept session's organization outliving it: "Organization
    "org-kept"; want oauth, the personal account".
  * The new session's facts not in the `Result`: "Result{Provider:grok
    Kind:oauth Model:grok-4.5}".
  * The notice not saying "refused": each kept test, "notices ["the saved
    Grok (xAI) session failed (authentication failed); sign in again"]".
  * The store emptied before a new session is saved: "saves [], the store
    holds OAuthSession(nil); want none, and the kept session".
  * The sign-in menu more than once: "unexpected Select("Choose how to
    authenticate with Grok (xAI):"); want the new sign-in refused".
  * In `catalog`, a listing never served again from the cache: "again: Live
    true, Err llmprovider: provider unavailable: grok HTTP 503, 2 requests;
    want true, nil, still 1", the stale listing the deviation names.
  * In `catalog`, a listing with no usable id not the static catalog: "Live
    false, Err <nil>, Source ""; want false, nil, "static"".
* **The code:** `discoverModels` gives the fallback the listing's error on
  every path but a refusal, which is returned. `resolvedCredential.kept`,
  set by `signIn` for the session `keepExistingOAuth` kept; `signIn`'s
  unexported `offerKept`. In `credentialStep`'s refusal loop, after the
  workspace case, a kept credential gives the notice "the saved <Label>
  session was refused (<describe>); sign in again", and `signIn` runs
  again with `offerKept` false. The store is not written: the new session
  is saved after the listing, as before (0059-MADR W2). `credentialResult`
  fills the `Result` from a credential, in place of any earlier one's, so
  the kept session's expiry and Kilo organization do not outlive it.
* **Docs:** `ResolveCredential`'s, as the deviation words it;
  `Options.Discover`'s points to it; `ConfigureLLM`'s, the kept session's
  refusal; `docs/architecture.md`, under `ResolveCredential` and "Keeping a
  saved session"; `README.md`'s wizard section; `CHANGELOG.md` Changed.
* **Green:** `TestResolve_FailedListingErr_0066` (five rows),
  `TestResolve_LiveMeans_0066` (two), `TestKept_RefusedLeadsToSignIn_0066`,
  `TestKept_RefusedThenNewRefused_0066` and
  `TestKept_KiloRefusedLeadsToSignIn_0066`: `--- PASS`, each. The notice,
  from a scratch probe of the first kept test: "the saved Grok (xAI)
  session was refused (authentication failed); sign in again".
* **The drift check:** `go test -race -count=1 -shuffle=on ./wizard/
  ./llmprovider/catalog/ ./llmprovider/providers/chatcompat/`, on a copy of
  the working tree without the phase's test file: `ok` three times, exit 0;
  no existing test changed.
* **No network:** `./wizard/` behind a scratch `TestMain` calling
  `localonly.Main`, on a scratch clone of the working tree, the new tests
  in it: `ok`, exit 0.
* **The API:** `tools/apidiff.py v1.24.1`: "./wizard.Options.LocalServers:
  added" and "./llmprovider/catalog.Lister: added", nothing else; `make
  api-check`, `docs-check` and `tables-check`: 0 problems.
* **The claims:** `tools/claims.py` on the MADR: "60 of 60 pass";
  `--plants` of C55 to C60, each made false: "all 6 plant(s) caught".
* **Pre-add:** `make pre-add-check` on `wizard/auth.go`,
  `wizard/configure.go` and `wizard/credential_0066_test.go`: "3 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)", exit 0. Its
  first run failed gofmt on the test file, a struct literal's alignment,
  after the plants; `gofmt -w` aligned it.
* **Found, outside the scope.** A listing that fails beside a static
  catalog, where the listing's own catalog recommends nothing, falls back
  with no notice: the fifth row's server answering 502 gave "Live=false
  Err=llmprovider: provider unavailable: lmstudio HTTP 502: down
  Recommended=["static-a" "static-b"] notices=[]", in a scratch probe.
  The path predates this phase; Phase 4 gives it its `Err`. It is put to
  the owner, not fixed here.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while other sessions held the load
  near 34 on ten CPUs, started at 9.9, and printed "gate: 29 of 29 passed",
  exit 0, `coverage-check` and `tables-check` among them.

### Phase 5: the notices (2026-10-31)

Approved by the owner's "committed, proceed", after Phase 4 landed as
`5c11035`, with the word to add Phase 4's out-of-scope finding to this
phase, as recommended. No deviation.

* **V17 added first.** Before the phase's code, the MADR gained the
  amendment "V17, a silent fallback", with C61 to C63, and this PLAN the
  Goal's, the heading's and step 6's annotations. `tools/claims.py`: "63 of
  63 pass"; `--plants` of C61 to C63, each made false: "all 3 plant(s)
  caught". C62 claims C29's line, `return fallback, cat.Err`, for V17.
* **Notes, not deviations.**
  * The tests have rows step 4 does not name: `TestEndpointCheck_…` a
    "not reached" row, an endpoint whose dial is refused;
    `TestBaseURL_QueryNotShown_0066` "both", a query and a fragment, and
    "through ResolveCredential", probe F itself.
  * Step 1's `<URL>` is the endpoint as typed, as before: an Ollama
    endpoint is not checked for a query or user information, so one typed
    with a key in its query is shown with it. Step 1 did not ask for more;
    this is told to the owner, not changed.
  * The first pre-add run failed `errcheck` on `errors.AsType`'s discarded
    `*APIError` in the V8 notice; `errors.As` replaced it, as `catalog`
    uses it. The plants, the drift check and the guard run were run again
    after it; the lines below are those.
* **Red first, on a scratch clone of `5c11035`**, Phase 4's commit, with the
  phase's test file copied in. `./wizard/` exits 1:
  * `TestEndpointCheck_NoServerText_0066`: "notices ["cannot reach
    http://127.0.0.1:…: llmprovider: provider unavailable: ollama HTTP 500:
    SERVER-TEXT-0066 echoed sk-ant-api03-••••abcd"]", and "a prompt shows
    SERVER-TEXT-0066"; the "not reached" row, "notices ["cannot reach
    http://127.0.0.1:1: llmprovider: provider unavailable: could not reach
    Ollama at http://127.0.0.1:1: llmprovider: offline: Get …: dial tcp:
    connect: connection refused"]".
  * `TestBaseURL_QueryNotShown_0066`: `err = wizard: base URL refused:
    https://proxy.example/v1?key=SECRET-0066 has a query or a fragment`, and
    as much for the fragment, both, no scheme ("proxy.example/v1?key=… needs
    http:// or https:// and a host") and not a URL ("http://[SECRET-0066 is
    not a URL"); "through ResolveCredential", "a prompt shows SECRET-0066".
    "a user and password" passes: that message never named the URL.
  * `TestDescribe_Offline_0066`: "describe(ErrOffline) = "provider
    unavailable"; want offline", and the notice "… is unavailable (provider
    unavailable); …".
  * `TestListing_StaticBesideNoticed_0066`: "notices []; want ["could not
    list models for LM Studio (provider unavailable, HTTP 502); using the
    built-in catalog"]".
* **Planted breaks** (`tools/plant.py`, scratch clone of the working tree):
  "all 11 plant(s) caught". `ErrOffline` not a class, and after its parent:
  `cannot reach http://127.0.0.1:1 (provider unavailable)`. A server that
  answered called unreachable: `cannot reach http://127.0.0.1:… (provider
  unavailable, HTTP 500)`. The answered notice with the error's text:
  "… answered: llmprovider: provider unavailable: ollama HTTP 500:
  SERVER-TEXT-0066 …"; the unreached one: `cannot reach http://127.0.0.1:1
  (llmprovider: provider unavailable: could not reach Ollama at …`. The
  query shown, the fragment shown, a non-URL echoed, a value with no scheme
  echoed, and the refused URL echoed whole, the plan's red: each row's
  "err = …" with `SECRET-0066`. A failure beside a static catalog not told:
  "notices []".
* **The code:** `errorClasses` gains `llmprovider.ErrOffline` before
  `ErrProviderUnavailable`. The Ollama endpoint check's notice is
  "<URL> answered: <describe>" for an `*llmprovider.APIError`, else "cannot
  reach <URL> (<describe>)"; the error under `NoEndpointPrompt` keeps its
  text. `shownURL` shows a URL's scheme, host and escaped path, then `?…`
  and `#…`; `validateBaseURL` uses it for a query or fragment refused, and
  for the plain-http notice and question, whose words stay; "the endpoint
  is not a URL" and "an endpoint needs http:// or https:// and a host" show
  no value. `discoverModels` tells a failure a catalog with nothing
  recommended carries, as V17's step says.
* **Docs:** `validateBaseURL`'s and `describe`'s; `CHANGELOG.md` Fixed.
  `ConfigureLLM`'s already says a service's error is shown by class and
  status.
* **Green:** `TestEndpointCheck_NoServerText_0066` (two rows),
  `TestBaseURL_QueryNotShown_0066` (seven), `TestDescribe_Offline_0066` and
  `TestListing_StaticBesideNoticed_0066`: `--- PASS`, each.
* **The drift check:** `go test -race -count=1 -shuffle=on ./wizard/
  ./llmprovider/catalog/ ./llmprovider/providers/chatcompat/`, on a copy of
  the working tree without the phase's test file: `ok` three times, exit 0;
  no existing test changed, `TestBaseURL_NoPromptPlainHTTP_0059` among
  them.
* **No network:** `./wizard/` behind a scratch `TestMain` calling
  `localonly.Main`, on a scratch clone of the working tree, the new tests
  in it: `ok`, exit 0.
* **The API:** `tools/apidiff.py v1.24.1`: "./wizard.Options.LocalServers:
  added" and "./llmprovider/catalog.Lister: added", nothing else; `make
  api-check`, `docs-check` and `tables-check`: 0 problems; `make
  markdownlint` and `changelog-check`: 0.
* **Pre-add:** `make pre-add-check` on `wizard/base_url.go`,
  `wizard/configure.go`, `wizard/describe.go` and
  `wizard/notices_0066_test.go`: "4 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)", exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while the load was 14.7 on ten
  CPUs, started at 8.5, and printed "gate: 29 of 29 passed", exit 0,
  `coverage-check` and `tables-check` among them.

### Phase 6: the menu, the guide and the tests (2026-10-31)

Approved by the owner's "committed, proceed", after Phase 5 landed as
`3e48e8b`. No deviation.

* **Notes, not deviations.**
  * The guide's new sentence had no test of its zero case: the existing
    `TestConfigureLLM_ThirdPartyListsThroughItsLister` sets `ModelLimit`.
    So that the guide states nothing a test does not show,
    `TestGuide_EveryListedRecommended_0066` is added: ten listed through a
    provider's `ListModels`, `ModelLimit` zero, all ten recommended in
    order. Step 4 does not name it.
  * `TestProviders_UnknownNamed_0066` names a repeated unknown id once, and
    checks that a `Providers` the registry lacks wholly gives the error as
    before, with no notice.
  * The unguarded run of the planted leak, which the MADR's probe made, was
    not repeated: it sends requests off the machine.
* **The drift check, after the code and before the tests:** `go test -race
  -count=1 -shuffle=on ./wizard/ ./llmprovider/catalog/
  ./llmprovider/providers/chatcompat/`, on a copy of the working tree
  without `wizard/localonly_test.go`: `ok` three times, exit 0;
  `TestConfigureLLM_UnknownProvidersIsPlain_0048` among them, unchanged.
  Then the whole of `./wizard/`, behind its guard in the tree, with `-race
  -shuffle=on`: `ok`, exit 0.
* **Red first, on a scratch clone of `3e48e8b`**, Phase 5's commit, with the
  phase's test file copied in. `./wizard/` exits 1:
  `TestProviders_UnknownNamed_0066`, "notices []; want ["no provider
  \"claud\" here; it is left out of the menu" "no provider
  \"nonexistent\" here; it is left out of the menu"]". The other three
  pass there: they cover code that was right.
* **Planted breaks** (`tools/plant.py`, scratch clone of the working tree):
  "all 7 plant(s) caught".
  * An unknown id not named, the plan's red: "notices []".
  * An unknown id named twice: three notices, "claud" twice.
  * `capMatches` cutting at 21, the plan's: a menu of 21.
  * `save` dropping the store's error, the plan's, for a session and for
    Kilo: "err = <nil>; want the save named, wrapping the store's error",
    each.
  * A zero `ModelLimit` cutting every id: "Recommended []; want every
    listed id, in order".
  * The MADR's planted leak, a test listing Claude with no client of its
    own: the run fails. On a scratch copy, the same test alone: "PASS", then
    "localonly: the tests sent requests off the machine, to
    api.anthropic.com:443 (1), models.opencode.ai:443 (1); …", "FAIL", exit
    1.
* **The code:** `selectableDescriptors` also returns each id of `allow` the
  registry lacks, once, in `allow`'s order; `chooseProvider` notifies "no
  provider "<id>" here; it is left out of the menu" for each. When none
  exists, the error is as before. `wizard/localonly_test.go`, built
  without `live_gateways`, with `TestMain` calling `localonly.Main`.
* **Docs:** `Options.Providers`'; `docs/guides/adding-a-provider.md`'s
  sentence, as step 2 words it; `AGENTS.md`'s list of the packages behind
  the guard and `docs/architecture.md`'s `internal/localonly` row, each
  gaining `wizard`; `CHANGELOG.md` Fixed.
* **Green:** `TestProviders_UnknownNamed_0066`,
  `TestGuide_EveryListedRecommended_0066`, `TestFallbacks_CapMatches_0066`
  and `TestSave_StoreFails_0066` (Grok, Kilo): `--- PASS`, each, behind the
  guard.
* **Coverage:** `go test -coverprofile ./wizard/`: 93.8 % of statements;
  `capMatches` 100.0 %, `save` 100.0 %, `selectableDescriptors` and
  `chooseProvider` 100.0 %.
* **The API:** `tools/apidiff.py v1.24.1`: "./wizard.Options.LocalServers:
  added" and "./llmprovider/catalog.Lister: added", nothing else; `make
  api-check`, `docs-check`, `tables-check`, `markdownlint` and
  `changelog-check`: 0 problems.
* **Pre-add:** `make pre-add-check` on `wizard/configure.go`,
  `wizard/localonly_test.go` and `wizard/menu_0066_test.go`: "3 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)", exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It waited while other sessions held the load
  near 36 on ten CPUs, started at 9.0, and printed "gate: 29 of 29
  passed", exit 0, `coverage-check` and `tables-check` among them.

### Phase 7: a bounded client that headers can wrap (2026-10-31)

Approved by the owner's "committed, proceed", after Phase 6 landed as
`5f90002`. Q7 was answered (a) on 2026-10-30, so the phase ran as written.

* **Deviation 2026-10-31: what `make api-check` prints.** Found after the
  phase's code and tests, in its gates. Step 4 says "`make api-check`
  lists `NewHTTPClient`", and the section "Exported names (R55)" that it
  and `tools/apidiff.py` "list exactly the three additions".
  `scripts/check_api.py` takes no option, and its whole output was
  "api-check: against v1.24.1, 0 incompatible change(s) outside
  llmprovider/x/": it reports incompatible changes only. The additions are
  listed by `tools/apidiff.py v1.24.1`. This held before the phase; Phases
  2, 3 and 4 recorded `api-check` as "0 problems" without saying so. Put to
  the owner with three resolutions: correct the records (recommended);
  make `check_api.py` list additions, a CI gate outside the scope, under
  an amendment; or stop the phase. The owner chose "correct the records".
  The two sentences are struck through and replaced; the Verification
  table's row 10 already gives each tool its part. No code, gate or MADR
  fact changes, and no file is added to the phase's scope.
* **Notes, not deviations.**
  * The example uses an `openai` provider given the wrapped client, so it
    shows `WithHTTPClient` in use, where step 3 says "a request to an
    `httptest` server".
  * The doc's "a connection within 30 s" is the dialer's, which
    `llmprovider/internal/transport/client_test.go:32` asserts at 30 s; the
    new test asserts that the shared transport has it, and the other four
    bounds by value.
  * The wizard's `Options.HTTPClient` doc says a client with headers adds
    them to the requests that doc already names, not to "every request of
    the run", which no test shows.
* **Red, by planted breaks** (`tools/plant.py`, scratch clone of the
  working tree; `NewHTTPClient` does not exist on `5f90002`, so no test
  compiles there): "all 7 plant(s) caught".
  * `NewHTTPClient` returning `&http.Client{}`, the plan's red:
    "Transport <nil> has no Base" in both `TestNewHTTPClient_…` tests, and
    `ExampleNewHTTPClient` and `TestWrappedClient_Headers_0066` panicking
    on a nil base.
  * `NewHTTPClient` returning one shared client: "clients 0x… and 0x…;
    want two, over one transport".
  * A given client not used (`settings.go` always the default):
    `TestWrappedClient_Headers_0066`, "request "/chat/completions X-Test=";
    want X-Test from the context", the same for "/models", "err = <nil>;
    want the wrapper's refusal" and "1 requests reached the server after
    the refusal; want none"; `TestOptions_HTTPClientHeaders_0066`,
    "listings' X-Test [""]"; the example's output.
  * The wrapper not installed, the plan's, in `chatcompat`'s test, the
    wizard's and the example: each fails as above.
  * The wizard's provider listing not given `Options.HTTPClient`
    (`listerFor`): "Live true, listings' X-Test [""]; want a live listing,
    each request carrying the header".
* **The code:** `llmprovider.NewHTTPClient`, in `options.go` beside
  `WithHTTPClient`, returning `transport.DefaultClient()`, with D2's doc.
* **Docs:** `WithHTTPClient`'s (the requests it serves, R32, and how to
  add headers); the `llmprovider` package doc's sentence; the wizard's
  `Options.HTTPClient`; `README.md`'s "Adding request headers", the
  wrapping in four lines; `docs/architecture.md`'s `llmprovider` row
  (R43); `CHANGELOG.md` Added (`llmprovider.NewHTTPClient`) and Fixed (D6's
  sentence).
* **Green:** `TestNewHTTPClient_Bounds_0066`,
  `TestNewHTTPClient_NewEachCall_0066`, `ExampleNewHTTPClient` (its
  `// Output:` "Hello" and "the server saw X-Example: from-the-wrapper"),
  `TestWrappedClient_Headers_0066` and `TestOptions_HTTPClientHeaders_0066`:
  `--- PASS`, each. With the new tests, `go test -race -count=1
  -shuffle=on ./llmprovider/ ./wizard/
  ./llmprovider/providers/chatcompat/`: `ok` three times, the wizard
  behind its guard.
* **The drift check:** the same command with `./llmprovider/catalog/`, on a
  copy of the working tree without the three new test files and with
  `llmprovider/example_test.go` as `5f90002` has it: `ok` four times, exit
  0; no existing test changed.
* **The API:** `tools/apidiff.py v1.24.1`: "./wizard.Options.LocalServers:
  added", "./llmprovider.NewHTTPClient: added" and
  "./llmprovider/catalog.Lister: added", nothing else; `make api-check`:
  "0 incompatible change(s)"; `make docs-check`: "sdkaudit: 0
  violation(s) of R50,R51,R52"; `tables-check`, `markdownlint`,
  `changelog-check` and `links-check`: 0 problems.
* **Pre-add:** `make pre-add-check` on `llmprovider/options.go`,
  `llmprovider/provider.go`, `llmprovider/httpclient_0066_test.go`,
  `llmprovider/example_test.go`,
  `llmprovider/providers/chatcompat/headers_0066_test.go`,
  `wizard/configure.go` and `wizard/headers_0066_test.go`: "7 file(s)
  clean (gofmt, golangci-lint, go vet, go test, govulncheck)", exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 7200`, on a scratch
  clone of the working tree. It started at once, the load at 3.1 on ten
  CPUs, and printed "gate: 29 of 29 passed", exit 0, `coverage-check` and
  `tables-check` among them.

### Phase 8: release `v1.25.0` (2026-10-31)

Approved by the owner's "committed proceed", after Phase 7 landed as
`e31fb56`. Steps 1 and 2 are this commit's; steps 3 to 6 follow on the
owner's word, and are recorded when done. No deviation.

* **Step 1.** `CHANGELOG.md`: `[Unreleased]`, now empty, above `[v1.25.0] -
  2026-10-31`, dated by the release commit, which the tag names; Added
  (`catalog.Lister`, `wizard.Options.LocalServers`,
  `llmprovider.NewHTTPClient`), Changed and Fixed from Phases 1 to 7, and
  the docs-gates entry, its last sentence reworded from "The module's code
  is unchanged" to "That change is to CI alone, not to the module's code",
  which a release with code changes needs. A blank line that split the
  Changed list since Phase 2 (`ecd3668`) is gone. The comparison links:
  `[Unreleased]` from `v1.25.0`, and `[v1.25.0]` from `v1.24.1`.
  `README.md`'s Status names `v1.25.0`. If the commit lands on another
  day, the section's date changes with it: `changelog-check` holds it to
  the tag's commit date.
* **Step 2, before the push.**
  * The notes: on a scratch clone of `e31fb56` with this `CHANGELOG.md`
    copied in and a local tag `v1.25.0`, `scripts/check_changelog.py
    --section v1.25.0 --repo maccavelli/go-llmprovider-sdk`: exit 0, 99
    lines, Added, Changed and Fixed; each record link absolute, at
    `blob/v1.25.0/`, none relative. Read in full. `tools/denyscan.py` on
    them: "0 hit(s) in 1 file(s)". `changelog-check` in that clone, the
    tag present: "37 release section(s), 0 problem(s)".
  * `tools/apidiff.py v1.24.1`: "./llmprovider/catalog.Lister: added",
    "./llmprovider.NewHTTPClient: added" and
    "./wizard.Options.LocalServers: added", nothing else.
  * The disclosure guard over the eight commits `origin/main..e31fb56`:
    exit 0, nothing printed. It runs again over the release commit before
    the push.
  * `make changelog-check`, `markdownlint` and `links-check` in the tree:
    0 problems.
  * The gate: `python3 -B tools/gate.py --timeout 7200`, on a scratch
    clone of the working tree. It started at once, the load at 3.9, and
    printed "gate: 29 of 29 passed", exit 0, `changelog-check`,
    `coverage-check` and `tables-check` among them.
* **Deviation 2026-10-31: a leak on Windows, found by CI on the release
  commit.** The owner committed and pushed the release commit, `3d0fc16`.
  Its CI run failed on `windows-2025` alone; `ubuntu-24.04`, `macos-15`
  and the Docs run passed, and the `release` job was skipped. Every test
  passed; the wizard's run failed the guard Phase 6 added (V13):
  "localonly: the tests sent requests off the machine, to auth.x.ai:443
  (1)". No tag was made.
  * **Reproduced** on the Windows dev host, go1.27.3 windows/amd64, on a
    clone of `3d0fc16`: `go test -count=1 ./wizard/` with CI's seed,
    1793465033152313900, three runs, and three random orders: each failed
    the guard, `auth.x.ai:443` (1) or (3). On this machine, the same seed,
    three runs: `ok`.
  * **Traced,** by running the 270 tests in chunks of 25 behind the guard
    on that host, then leaving each of the leaking chunk's tests out in
    turn: the leak needs `TestConfigureLLM_ListingTokenFailureUsesStaticCatalog`
    (`wizard/model_select_edge_test.go:222`, from `cf70dbe`, 2026-09-27),
    and fails to show when it runs alone. The test keeps an expired Grok
    session with no `HTTPClient` and a `DiscoverLimit` of 1 ns. The
    catalog's fetch is detached from its caller (0053-MADR D4), so after
    the listing has timed out it goes on to refresh the session at
    `auth.x.ai`, through the default client. The request is sent only if
    the process is still running then, which a later test makes it: with
    the test and sixteen others, one run of two leaked; with the sixteen
    alone, neither did.
  * Put to the owner with three resolutions: fix the test, then tag
    (recommended); stop a detached fetch's refresh in `catalog`, a change
    to 0053-MADR D4 needing its own decision; or stop the release. The
    owner chose "fix the test, tag later". The test gains an `HTTPClient`
    whose transport refuses every request, and `testCache(t)` and
    `llmprovider.WithoutModelMetadata()`, as `AGENTS.md` asks of a test
    that lists; its assertions are unchanged. Its comment, which said
    "no request leaves the host", says why the client is there. Scope
    gains the test, and
    the MADR has the amendment "a leak V13's guard found on Windows".
    `CHANGELOG.md` is unchanged: no code of the module changes. The tag
    names the commit that holds the fix.
  * **The fix's checks.** Red: the six guarded runs of the package on the
    Windows host above, each failing, are the break the fix is seen
    against. With the test's first form, a refusing client alone, on that
    host: six random orders and CI's seed, each `ok`. With the committed
    form, its cache and no models.dev document added: `go test
    -shuffle=on ./...`, as CI's Windows job runs it, exit 0, 40 packages
    `ok`; then `./wizard/` five more times, each `ok`. Here: the test
    `--- PASS`; `make pre-add-check` on the file: "1 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)", exit 0, run again after
    the comment's correction, which changes no code. CI's Linux-only steps
    passed on `3d0fc16`, and see no change but this test.
  * The gate, with the fix: `python3 -B tools/gate.py --timeout 7200`, on
    a scratch clone of the working tree. It waited while other sessions
    held the load near 32, started at 8.6, and printed "gate: 29 of 29
    passed", exit 0, `changelog-check` and `coverage-check` among them.
