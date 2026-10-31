---
status: accepted
date: 2026-10-30
decision-makers: repository owner
consulted: "the owner's request of 2026-10-30; a read of every file of wizard/ at 3fad786 (v1.24.1 and the docs gates after it); probes A to L on a scratch clone of 3fad786 (method below); the wizard's statement coverage; providers/chatcompat, catalog and auth at the same revision; gobble-cli at 61f7115, the wizard's caller; 0059-MADR-wizard-debugging-pass-findings.md, 0061-MADR-consumer-gaps-from-gobble-issue-9.md, 0064-MADR-anthropic-workspace-for-unscoped-keys.md, 0065-MADR-tests-reach-no-outside-host.md, 0056-MADR-wizard-credential-only-entry-point.md and 0021-MADR-harden-and-tune-after-the-v1-1-review.md"
informed: gobble-cli, and every consumer of the wizard
---

<!-- markdownlint-disable MD013 -->

# Remediate the Wizard's v1.24 Debugging-Pass Findings, Listed Ids and Chat Completions Servers First

## Context and Problem Statement

On 2026-10-30 the owner asked for a pass over the wizard: "give the wizard a
debugging pass. find bugs. find gaps. find missing or incomplete wiring or
functionality. report findings as a detailed, complete, madr for review."

The last pass,
[0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md),
read the wizard at `2d6e7f3` and is complete. Since then three records
changed what the wizard calls or what it can be given:

* [0061-MADR-consumer-gaps-from-gobble-issue-9.md](0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  added `providers/chatcompat`, any Chat Completions server under an id the
  caller picks, with `Descriptor` for a caller's `Registry` and `Discover`
  for a configure step. It also rebuilt the wizard's model search on
  `Catalog.Search`, and added `ErrOffline` and `ErrEmptyAnswer`.
* [0064-MADR-anthropic-workspace-for-unscoped-keys.md](0064-MADR-anthropic-workspace-for-unscoped-keys.md)
  added the workspace prompt.
* [0065-MADR-tests-reach-no-outside-host.md](0065-MADR-tests-reach-no-outside-host.md)
  added the `localonly` guard to six test packages.

This pass read every file of `wizard/` at `3fad786` (about 3,250 lines of
source, 93.5 % of statements covered), traced each path into `catalog`,
`auth` and `providers/chatcompat`, and ran probes A to L on a scratch clone.
It found 14 findings:

* 2 of high severity;
* 7 medium;
* 5 low: a corner, documentation, or a gap in the tests.

Four of them, V2 to V5, come from wiring a `chatcompat` provider into the
wizard, which 0061 named as a call site and nothing built. This record lists each finding
with its evidence. It decides how to remediate them, and asks the owner the
design questions the fixes depend on.

### Method

* **Read.** Every file of `wizard/`, against the doc of each exported name,
  and each package doc and guide that describes the wizard: `README.md`'s
  "The configuration wizard", `docs/architecture.md`'s "The wizard", and
  `docs/guides/adding-a-provider.md`.
* **Probed.** A to L are scratch tests in package `wizard`, in a clone of
  `3fad786` outside the tree. Each runs the wizard through its own seams: the
  package's `fakePrompter`, or a `TextPrompter` over a scripted reader; an
  `httptest` server, or an `HTTPClient` whose transport answers in place of
  the service. Each records what the wizard does today; none asserts a fix.
  None reads or prints a real key: the keys are made-up strings.
* **Measured.** `go test -coverprofile ./wizard/` at `3fad786`, and the
  uncovered statements.
* **Checked against.** The caller: gobble-cli at `61f7115` calls
  `ResolveCredential` with `Discover: true` (G1) and `NoEndpointPrompt:
  true` (G2). Its `Prompter`'s `Input` answers `errNoTerminal` when there is
  no terminal (G3).
* **One request left this machine.** The guard's planted leak (V13), run
  once without the guard to show the difference, reached
  `api.anthropic.com` and `models.opencode.ai` from this host, carrying the
  made-up key `sk-ant-api03-planted`. No other probe sent a request off the
  machine; each that lists serves its own replies and turns models.dev off.

### Findings

Severity follows 0059-MADR, with one case added to high, for V1: **high**
misleads the user about a credential, loses one, or lets a remote party act
on the user's terminal; **medium** is
wrong behaviour on a path a user takes; **low** is a corner, a gap in the
tests, or documentation. Each finding says how it is known: **[probed]** A to
L, **[read]** the code at the claim, **[measured]** coverage.

#### High

* **V1. A listed model id carries escape sequences to the terminal and into
  the Result** **[probed]** (J; C18, C19, C28).
  * 0021-MADR Z3 stopped escape sequences in an error body: `Message` and
    `Code` are stripped, and `TextPrompter.Notify` strips what it prints
    (C19). A menu row is not: `renderChoices` prints each `Choice`'s
    `Label` and `Detail` as they are (C18).
  * A model id comes from a server: any listing, and a Chat Completions
    server's above all, at an address the user typed. `catalog` checks no
    listed id for control characters; a search of it finds only the
    workspace's check.
  * J: a server listed `evil\u001b]0;owned\u0007\u001b[2Jmodel`.
    `TextPrompter` wrote the menu row `1) evil\x1b]0;owned\a\x1b[2Jmodel`,
    two ESC bytes and one BEL: the terminal's title is set and its screen is
    cleared. The user picked row 1, and the Result's `Model` held the same
    bytes, for the consumer to persist and print again.
  * The same row reaches a consumer's own `Prompter`. A Kilo organization's
    name, from Kilo's profile, is a menu label in the same way (C28)
    **[read]**.
* **V2. A Chat Completions server that wants a key cannot be configured,
  and the error blames a pasted credential** **[probed]** (B; C5, C6, C7,
  C31).
  * `chatcompat`'s key is optional (C7): a local server takes none, and a
    remote one, or a local one started with a key, answers 401.
    `chatcompat.Descriptor` sets no `RequiresAPIKey` and no `AuthMethods`
    (C3).
  * The wizard asks for a key only when `RequiresAPIKey` is set (C5). With
    none, the key is empty and its source `keyOther`, "a pasted credential,
    a session" (C31).
  * B: a server that answers 401. No `Secret` was asked; the listing sent no
    `Authorization` header and was refused once. The run ended with "wizard:
    LM Studio refused the pasted credential: … HTTP 401", from
    `refusalError`'s last case (C6). Nothing was pasted, and the user has no
    way, in the wizard, to give the key the server wants.

#### Medium

* **V3. A blank endpoint is accepted for a provider with no default, and
  the Result cannot build its provider** **[probed]** (A; C1, C2, C4).
  * A blank URL "keeps the provider's default" (C2). `chatcompat.Descriptor`
    has none, so the Result's `BaseURL` is empty.
  * A: Enter at "LM Studio endpoint", then a model id typed at "No models
    found". The run returned no error and `Result{Provider:lmstudio
    Model:local-model-00}`, after the notice "could not list models for LM
    Studio (invalid request)". Building the provider from that Result failed:
    "chatcompat "lmstudio" needs WithBaseURL" (C4).
  * Under `NoEndpointPrompt`, with no `Existing.BaseURL`, the same: no
    error, an empty `BaseURL`.
* **V4. A Chat Completions provider is listed through `ListModels`, which
  drops its menu, its facts and its `Source`** **[probed]** (C; C8, C9,
  C24).
  * The wizard calls `catalog.List` with the provider's id (C8). For an id
    that is not a built-in one, `catalog` lists it only when the options
    carry `chatcompat.WithID` (C9); the wizard's do not. So it gets
    `ErrUnsupported` and lists through the provider's `ListModels`
    (`providerCatalog`), which returns ids only (C24).
  * C, a server listing 30 models: through the wizard, `Source=""`,
    `Menu=0`, 30 recommended and 30 usable; through `catalog.List` with
    `WithID`, `Source="live"` and `Menu=30`.
  * So the search runs on ids alone: no facts (`vision`, `tools`, a context
    size), and none of the dialect's grouping (0061-MADR E2, E4), which
    `catalog` built for this provider. The ids still match: "local-model-2"
    found 10.
* **V5. `chatcompat.Discover` is not wired into the wizard, though 0061
  names it as a call site** **[read]** (C11).
  * 0061-MADR D10, the table of names each with the call site that needs it
    (R55), gives `Discover` and `Server` two: "a caller's configure step:
    gobble-cli's, the wizard's" (C11). D11 says why: "A configure step,
    gobble-cli's or the wizard's, offers what is running without asking for
    a URL."
  * No file of `wizard/` names `chatcompat` or calls `Discover`; a search
    finds none. 0061-PLAN records no step for the wizard's call, nor a
    decision to leave it out.
  * The wizard offers a Chat Completions server only when the caller has
    registered its descriptor, and then asks for its URL (V3).
* **V6. `ResolveCredential`'s caller cannot tell that the listing failed,
  and the credential went unchecked** **[probed]** (L, D; C20, C29, C36).
  * `ResolveCredential` returns the catalog "the check listed", and says
    "The credential is checked only with Options.Discover". A listing that
    fails for any reason but a refusal or the end of `ctx` is not an error:
    the wizard warns and returns the static catalog.
  * On `catalog.List`'s path the catalog's `Err` keeps the failure: D,
    offline, `cat.Err` matched `ErrOffline`. On a provider's own
    `ListModels`, and on the chatcompat path, it does not: the fallback is
    built with no `Err` (C36) and returned beside the error (C20, C29).
  * L: a provider whose listing failed with HTTP 500 returned `err=nil`,
    `Live=false`, `cat.Err=nil`; so did a chatcompat server answering 502.
    A run without `Discover` returned the same three. The warning went to
    the `Prompter` only.
  * gobble-cli calls `ResolveCredential` with `Discover: true` (G1) to check
    a sign-in. On those paths it cannot tell a checked credential from an
    unchecked one.
* **V7. A kept session the listing refuses ends the run; the sign-in menu
  is not offered** **[probed]** (G; C21).
  * 0059-MADR W6 made an unusable saved session not offered: a notice says
    why, and the sign-in menu follows. A saved session that passes that
    check, is kept, and is then refused by the listing still ends the run:
    "refused the sign-in; sign in again" (C21).
  * G: a Grok session in the store, kept; the listing answered 401, and so
    did the refresh it tried. The error was "wizard: Grok (xAI) refused the
    sign-in; sign in again: … oauth: refresh failed: 401 Unauthorized"; the
    method menu was not shown. The user must start the wizard again.
  * A refused key from the environment or the saved configuration also ends
    the run, as decided (0028-MADR D-H10; 0059-MADR Q6 (a)). A session was
    not part of that question; Q4 asks.
* **V8. The local endpoint check shows the server's text in a notice**
  **[probed]** (E; C14, C15).
  * `ConfigureLLM`'s doc: "A service's error is shown as its class and HTTP
    status, never its text" (C15; 0056-MADR D3). Every listing notice uses
    `describe`. The Ollama endpoint check does not: it prints the error with
    `%v` (C14).
  * E: an endpoint answering 500 with a body naming `SERVER-TEXT-0066` and
    a masked key. The notice read `cannot reach http://127.0.0.1:…:
    llmprovider: provider unavailable: ollama HTTP 500: SERVER-TEXT-0066
    echoed sk-ant-api03-••••abcd`. 0056-MADR D3 is about exactly that echo.
  * "cannot reach" is also wrong for a server that answered.
* **V9. A refused base URL is echoed in full, its query included, where a
  key can be** **[probed]** (F; C16, C17).
  * `Prompter.Notify` "must not be used to display a credential; callers
    mask first" (C17). A URL refused for its query is quoted whole in the
    error (C16), and the wizard shows that error as a notice.
  * F: `https://proxy.example/v1?key=SECRET-QUERY-KEY-0066` gave the notice
    `wizard: base URL refused: https://proxy.example/v1?key=SECRET-QUERY-KEY-0066
    has a query or a fragment`. A key in the query, as some gateways take
    one, reaches the terminal, and any log a consumer's `Notify` writes.
  * A URL refused as "not a URL" or for its scheme is quoted the same way.
    One with a user name and password is not: that message names neither.

#### Low

* **V10. `describe` has no `ErrOffline`: an offline listing is worded
  "provider unavailable"** **[probed]** (D; C12, C13).
  * `describe` names the most specific of `errorClasses`, "each child
    before its parent" (C12). `ErrOffline`, added in `v1.23.0` as a child of
    `ErrProviderUnavailable` (C13), is not among them.
  * D: `describe` of an error matching `ErrOffline` is "provider
    unavailable". A Claude listing whose dial was refused gave "live model
    listing for Claude (Anthropic) is unavailable (provider unavailable);
    search covers the built-in catalog only". The user is not told that the
    machine is offline, which is what they can fix.
* **V11. An id in `Options.Providers` that the registry lacks is dropped
  without a word** **[probed]** (H; C22, C35).
  * `Providers` "restricts the menu to these ids" (C35).
    `selectableDescriptors` errs only when none exists (C22).
  * H: `Providers: [claude, claud]` showed a menu of one, with no notice and
    no error. `ResolveCredential` with an unknown id does refuse it, with
    `ErrInvalidProvider`.
* **V12. The provider guide says the first six listed ids are recommended;
  every one is** **[probed]** (C; C23, C24).
  * `docs/guides/adding-a-provider.md`: "The first six listed ids are
    recommended, in your order" (C23). `providerCatalog` cuts them to
    `Options.ModelLimit` (C24), and zero, the default, cuts nothing, as
    `Options.ModelLimit`'s doc and `docs/architecture.md` say.
  * C: 30 listed, 30 recommended.
* **V13. The wizard's tests run without the `localonly` guard**
  **[probed]** (C25, C38).
  * 0065-MADR D2 put the guard in six packages, and "a package added later
    opts in the same way" (C38); `AGENTS.md` says the same of "a package
    whose tests list models" (C25). The wizard's tests list models, and it
    has no `TestMain`.
  * With a scratch `TestMain` calling `localonly.Main`, today's tests pass:
    none leaves the machine. A planted test that lists Claude with no client
    of its own failed the guarded run, naming `api.anthropic.com:443 (1)`
    and `models.opencode.ai:443 (1)`, and passed the unguarded one. So a
    later test that leaks is not caught in this package today.
* **V14. Two paths have no test** **[measured]** (C26, C27).
  * `capMatches`' cut, past 20 fallback matches, with its notice, is 50 %
    covered: no test searches fallbacks with more than 20 matches.
  * `save`'s failed `Save`, for Kilo and for any other session, runs in no
    test (C27). Its refusal path is tested.
  * The other uncovered statements are error returns, as in 0059-MADR's
    measure.

#### Checked, not findings

* `README.md`'s glob claim holds on both search paths: `local-model-0*` and
  `*model-1*` each matched 10 of 30, through `Catalog.Search` and through
  `catalog.Search` (probe K).
* The fixes of 0059-MADR still hold: `go test ./wizard/` passes at
  `3fad786`, their tests among them.
* `Logout` gives its session the caller's client, and revokes only OpenAI's
  and Grok's sessions, the plan's among them (`auth.RevokeOAuthSession`).
* A workspace refusal under a caller with no terminal ends the run. That is
  gobble-cli's `Prompter` answering `errNoTerminal` (G3), and the variable's
  value is a prompt's default by 0064-MADR D4.3 and Q2, as decided.
* A saved model whose case differs from the listed id is offered as
  "current" beside the listed one. A provider's ids can differ by case, so
  the saved id is kept as it was saved.

## Decision Drivers

* Nothing a server sends may act on the user's terminal, or reach the
  consumer's configuration as a model id it cannot be (V1).
* A Result the wizard returns without an error must build its provider
  (V3), and a provider whose server wants a key must be able to get one
  (V2).
* What a record names as a call site is built, or the record says it is
  not (V5).
* A caller that checks a credential can tell when it was not checked (V6).
* A path a user takes must not end where it could go on (V7).
* Notices keep 0056-MADR D3 and the `Prompter`'s contract: no service text,
  no credential (V8, V9).
* A behaviour that changes is documented in the same change (R52); an
  exported name added lists its call site (R55).
* Each fix is a test seen to fail first, on today's code or on a planted
  break.

## Considered Options

* **R1.** Remediate in a 0066 PLAN, by severity, after the design questions
  below.
* **R2.** Fix V1 and V2 now, and leave the rest to a later pass.
* **R3.** Fold each finding into the record that owns its area: 0021 for
  V1, 0061 for V2 to V5, 0056 for V6, V8 and V9, 0059 for V7.

## Decision Outcome

Chosen option: **R1** (proposed), because V2 to V5 are one piece of work,
making the wizard serve a Chat Completions server, and R3 would reopen four
records at once. R1 keeps one record for one
pass, as 0059-MADR did, and V1, which any listing can trigger, is its first
phase.

### D1. Phases for the PLAN, once this record is accepted

1. **V1:** control characters, by Q3's answer.
2. **V2, V3, V4:** a Chat Completions provider through the wizard: its
   endpoint required, a key asked when its server refuses the listing, and
   its catalog by Q1's answer.
3. **V5:** `Discover`, by Q2's answer.
4. **V6, V7:** the unchecked credential, by Q5's answer; a refused kept
   session, by Q4's answer.
5. **V8, V9, V10:** notices: the endpoint check through `describe`; a
   refused URL shown without its query, fragment or user information;
   `ErrOffline` in `errorClasses`, with a notice that says the machine is
   offline.
6. **V11, V12, V13, V14:** unknown `Providers` ids, by Q6's answer; the
   guide; a `TestMain` with the guard; tests for `capMatches`' cut and
   `save`'s failed `Save`.

Each phase is tests first: A to L become tests that fail on `3fad786`, or on
a planted break where a probe recorded no defect (V13). Each phase updates
the docs of what it changes, and `CHANGELOG.md`. The release is minor if Q1
or Q2 adds an exported name, else a patch.

### Consequences

* Good, because no server's bytes reach the terminal or the Result as a
  model id (V1).
* Good, because a Chat Completions server, keyed or not, is configured
  through the wizard to a Result that builds (V2, V3, V5), and, under Q1
  (a) or (b), with the menu and the facts `catalog` already builds for it
  (V4).
* Good, because a caller can tell a checked credential from an unchecked one
  (V6), and a refused kept session leads to the sign-in menu (V7).
* Neutral, because V8 to V14 are small, and each is a phase's step.
* Bad, because Q1 and Q2 may add exported names: a minor release, and one
  more surface to keep.
* Bad, because the pass is 14 findings in 6 phases.

### Confirmation

* A to L, made tests, each failing on `3fad786` and passing after its phase.
* For V1: a listed id holding C0 or C1 control characters is not offered,
  and no menu row a `TextPrompter` prints holds one.
* For V2 to V4: through a registered `chatcompat` descriptor and an
  `httptest` server, a keyed server is configured with a typed key; a blank
  endpoint is asked again; under Q1 (a) or (b), the catalog has `Source`
  and `Menu`.
* `make pre-add-check`, `python3 -B tools/gate.py`, `make api-check` for any
  name Q1 or Q2 adds, and CI on the three platforms.

## Pros and Cons of the Options

### R1. A 0066 PLAN, by severity

* Good, because one record holds the pass, its evidence and its fixes.
* Good, because V2 to V5 change the same lines (`credentialStep`,
  `discoverModels`, `resolveBaseURL`) and land together.
* Bad, because V1 waits for Q3. It is Phase 1, so it is first.

### R2. V1 and V2 now

* Good, because the two high findings are fixed soonest.
* Bad, because V3 to V5 change the same lines as V2 (`credentialStep`,
  `discoverModels`, `resolveBaseURL`), so a second pass reopens them.

### R3. Fold into the owning records

* Good, because each fix sits with its decision.
* Bad, because four records reopen at once (0021, 0056, 0059, 0061), and
  the pass's evidence is split among them.

## Owner questions

* **Q1 (V4; V2 and V3 the same under each).** V2 and V3 need no exported
  name, and are fixed the same way whatever the answer: a provider with
  `SupportsBaseURL` and no `DefaultBaseURL` must be given an endpoint (V3);
  one with no `RequiresAPIKey` that the listing refuses is asked for a key,
  up to three, as a typed key is, and the error names the server, not a
  paste (V2). The wizard cannot tell a `chatcompat` provider from another
  by building it, since its type is not exported. Q1 is how the wizard gets
  such a provider's whole catalog:
  * **(a)** An interface in `catalog`, such as `CatalogLister` with
    `ListCatalog(ctx) (Catalog, error)`, which a provider implements to give
    its catalog, menu and facts included. `chatcompat`'s provider implements
    it, and `providerCatalog` uses it before `ListModels`. One exported
    name; a third party's provider can give its facts the same way
    (recommended).
  * **(b)** A field on `llmprovider.Descriptor` that marks a Chat
    Completions provider, set by `chatcompat.Descriptor`. The wizard lists
    such a provider with `chatcompat.WithID`, and asks for its optional key
    before the listing, Enter for none. One exported name; `chatcompat`
    only.
  * **(c)** Document it: a caller whose menu holds one Chat Completions
    provider passes `chatcompat.WithID` in `ProviderOptions`, as probe C
    did. No exported name; V4 stays for a caller that does not.

  **Answered 2026-10-30: (a).**
* **Q2 (V5). `Discover` in the wizard.**
  * **(a)** An option, such as `Options.LocalServers`: the provider menu
    adds each server `Discover` finds, as a `chatcompat` provider with its
    URL filled in, and a server that `NeedsKey` asks for one. The Result
    names its id and `BaseURL`; the consumer registers it as the package
    doc shows. An exported name; a minor release (recommended: 0061-MADR
    D10 decided it).
  * **(b)** Leave `Discover` to consumers, and amend 0061-MADR D10 and D11
    to name gobble-cli's configure step only.
  * **(c)** A helper that returns the found servers as descriptors, for the
    caller to register before it runs the wizard.

  **Answered 2026-10-30: (a).**
* **Q3 (V1). Control characters in a listed id.**
  * **(a)** Both layers: `catalog` drops a listed id holding a C0 or C1
    control character, with a debug log, and `TextPrompter` strips them
    from every row, title and default it prints, as `Notify` does
    (recommended).
  * **(b)** `TextPrompter` only. A consumer's `Prompter` and the Result
    still get the bytes.
  * **(c)** `catalog` only. A Kilo organization's name still reaches the
    menu as it came.

  **Answered 2026-10-30: (a).**
* **Q4 (V7). A kept session the listing refuses.**
  * **(a)** A notice that says so, then the sign-in menu, as 0059-MADR W6's
    answer does for an unusable one; the store keeps the session until a
    new one is saved (recommended).
  * **(b)** Keep ending the run, as a refused saved key does (0059-MADR
    Q6).
  * **(c)** Ask whether to sign in again, and end the run on no.

  **Answered 2026-10-30: (a).**
* **Q5 (V6). An unchecked credential.**
  * **(a)** The returned catalog's `Err` carries the listing's failure on
    every path, and `ResolveCredential`'s doc says so: `Live` false and
    `Err` set is a credential the listing did not check. No exported name
    (recommended).
  * **(b)** `ResolveCredential` returns an error matching the listing's when
    `Discover` is set and the listing fails. A change of behaviour a
    consumer sees.
  * **(c)** Document today's behaviour only.

  **Answered 2026-10-30: (a).**
* **Q6 (V11). An id in `Options.Providers` the registry lacks.**
  * **(a)** A notice naming each such id; the menu offers the rest
    (recommended).
  * **(b)** An error matching `ErrInvalidProvider`, as `ResolveCredential`
    gives for an unknown id.
  * **(c)** Leave it, and say so in `Providers`' doc.

  **Answered 2026-10-30: (a).**

## More Information

### Probes

Each probe is a scratch test in package `wizard`, in a clone of `3fad786`
outside the tree, run with `go test -count=1 -run TestProbe0066 -v
./wizard/`. The Chat Completions provider is registered as
`providers/chatcompat`'s package doc shows, under the id `lmstudio`, label
"LM Studio".

* **A.** `ConfigureLLM`, `Discover`, the endpoint answered with Enter, then
  `local-model-00` typed:

  ```text
  err=<nil> res=Result{Provider:lmstudio Model:local-model-00} BaseURL=""
  inputs=["LM Studio endpoint" "No models found; enter a model id"] notices=["could not list models for LM Studio (invalid request)"]
  registry.New with the Result: err=llmprovider: invalid request: chatcompat "lmstudio" needs WithBaseURL
  NoEndpointPrompt: err=<nil> BaseURL="" notices=["could not list models for LM Studio (invalid request)"]
  ```

* **B.** As A, the endpoint an `httptest` server answering 401 to every
  request, one `Secret` answer scripted:

  ```text
  err=wizard: LM Studio refused the pasted credential: llmprovider: authentication failed: lmstudio HTTP 401: no key kind=""
  secrets asked=[] selects=["Choose an LLM provider:"] listing requests=1 auth headers=[""]
  errors.Is ErrAuthFailure=true
  ```

* **C.** `ResolveCredential` against a server listing 30 models, and
  `catalog.List` on the same server, with and without `chatcompat.WithID`:

  ```text
  wizard: err=<nil> Live=true Source="" Recommended=30 Usable=30 Menu=0 notices=[]
  catalog.List with WithID: err=<nil> Live=true Source="live" Recommended=30 Usable=30 Menu=30
  catalog.List without WithID: err=model listing: unsupported provider "lmstudio": llmprovider: unsupported ErrUnsupported=true
  search 'model-2' through the wizard's catalog: 10 matches
  ```

* **D.** `describe` of each new sentinel; then `ResolveCredential` for
  Claude with an `HTTPClient` whose dial is refused (`ECONNREFUSED`):

  ```text
  describe(ErrOffline)="provider unavailable"
  describe(ErrEmptyAnswer)="incomplete response"
  err=<nil> cat.Err offline=true notices=["live model listing for Claude (Anthropic) is unavailable (provider unavailable); search covers the built-in catalog only"]
  ```

* **E.** `ResolveCredential` for Ollama, its endpoint a server answering 500
  with a body naming `SERVER-TEXT-0066`, then "Try a different endpoint?"
  answered no:

  ```text
  notice: "cannot reach http://127.0.0.1:52346: llmprovider: provider unavailable: ollama HTTP 500: SERVER-TEXT-0066 echoed sk-ant-api03-••••abcd" (server text shown: true)
  ```

* **F.** `ResolveCredential` for Together, the endpoint first typed with a
  query:

  ```text
  notice: "wizard: base URL refused: https://proxy.example/v1?key=SECRET-QUERY-KEY-0066 has a query or a fragment" (query value shown: true)
  ```

* **G.** `ConfigureLLM` for Grok, a session in the store, `Existing` naming
  it, every request answered 401, the keep question answered yes:

  ```text
  err=wizard: Grok (xAI) refused the sign-in; sign in again: model listing: acquire token: llmprovider: authentication failed: oauth: refresh failed: 401 Unauthorized: {"error":"invalid_grant"}
  confirms=["Keep the existing session (••••••••abcd)?"] selects=["Choose an LLM provider:"] (method menu shown: false)
  ```

* **H.** `ConfigureLLM` with `Providers: [claude, claud]`: the first menu
  had 1 choice, and no notice or error came of `claud`.
* **J.** `ConfigureLLM` over a `TextPrompter` with a scripted reader, the
  server listing `evil\u001b]0;owned\u0007\u001b[2Jmodel` and
  `plain-model`:

  ```text
  err=<nil> model="evil\x1b]0;owned\a\x1b[2Jmodel"
  ESC bytes written to Out: 2; BEL bytes: 1
  line: "  1) evil\x1b]0;owned\a\x1b[2Jmodel"
  ```

* **K.** The README's globs on a 30-model listing, through `Catalog.Search`
  and `catalog.Search`: `local-model-0*` 10 and 10, `*model-1*` 10 and 10,
  `local-model-07` 1 and 1.
* **L.** `ResolveCredential` with `Discover` for a provider whose
  `ListModels` fails with HTTP 500, for a Chat Completions server answering
  502, and without `Discover`:

  ```text
  provider ListModels path: err=<nil> kind="api_key" Live=false cat.Err=<nil> Recommended=2 notices=["could not list models for Acme (provider unavailable, HTTP 500); using the built-in catalog"]
  chatcompat path: err=<nil> Live=false cat.Err=<nil> notices=["could not list models for LM Studio (provider unavailable, HTTP 502)"]
  no Discover: err=<nil> Live=false cat.Err=<nil>
  ```

* **The guard (V13).** A scratch `wizard/localonly_test.go` with
  `TestMain(m) { os.Exit(localonly.Main(m)) }`: `go test -count=1 ./wizard/`
  passed. A planted test, `ResolveCredential` for Claude with `Discover` and
  no `HTTPClient`, failed the guarded run with "localonly: the tests sent
  requests off the machine, to api.anthropic.com:443 (1),
  models.opencode.ai:443 (1)", exit 1, and passed without the guard, exit 0.

### Coverage

`go test -coverprofile ./wizard/` at `3fad786`: 93.5 %. Below 85 %:
`skippedEvery` 57.1 %, `capMatches` 50.0 %, `resolveVendorCLI` 76.5 %,
`resolveKiloDeviceKey` 75.0 %, `keepExistingOAuth` 83.3 %, `validateKept`
80.0 %, `checkAccessToken` 78.3 %, `accessTokenAccount` 70.0 %, `save`
70.0 %, `homeDir` 75.0 %, `providerLabel` 66.7 %, `secretText` 66.7 %, and
`TextPrompter`'s `out` and `in`, 66.7 % each. Apart from V14's two paths,
each uncovered statement is an error return or a nil default.

### Related

* [0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md):
  the last pass; W6 and Q6 beside V7.
* [0061-MADR-consumer-gaps-from-gobble-issue-9.md](0061-MADR-consumer-gaps-from-gobble-issue-9.md):
  `chatcompat`, `Discover` and its call sites (D10, D11), the search (S1,
  S4), `ErrOffline` (D2).
* [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md):
  Z3, control characters in an error body.
* [0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md):
  `ResolveCredential`, and D3, no service text in a notice.
* [0064-MADR-anthropic-workspace-for-unscoped-keys.md](0064-MADR-anthropic-workspace-for-unscoped-keys.md):
  the workspace prompt.
* [0065-MADR-tests-reach-no-outside-host.md](0065-MADR-tests-reach-no-outside-host.md):
  the guard (D2).

### Code claims

<!-- claims: self=3fad786 gobble-cli=61f7115 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The endpoint step | `wizard/configure.go:445` | `func resolveBaseURL(` |
| C2 | A blank URL keeps the default | `wizard/base_url.go:24` | `if raw == "" {` |
| C3 | chatcompat's descriptor: a base URL, nothing else | `llmprovider/providers/chatcompat/descriptor.go:13` | `SupportsBaseURL: true,` |
| C4 | chatcompat needs a base URL | `llmprovider/providers/chatcompat/chatcompat.go:187` | `needs WithBaseURL` |
| C5 | A key only when the descriptor requires one | `wizard/configure.go:574` | `if !d.RequiresAPIKey {` |
| C6 | The refusal's last case | `wizard/configure.go:566` | `refused the pasted credential` |
| C7 | chatcompat's key is optional | `llmprovider/providers/chatcompat/chatcompat.go:13` | `// The key is optional, unlike a built-in's` |
| C8 | The wizard's listing call | `wizard/configure.go:675` | `cat, err = catalog.List(dCtx, d.ID, source, opts...)` |
| C9 | catalog lists a non-built-in id only as chatcompat | `llmprovider/catalog/config.go:241` | `if err != nil \|\| !cfg.compat {` |
| C10 | then the provider's own ListModels | `wizard/configure.go:680` | `cat, err = providerCatalog(dCtx, o, d, res, source)` |
| C11 | Discover's call sites | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:373` | `a caller's configure step: gobble-cli's, the wizard's` |
| C12 | describe's classes, no ErrOffline | `wizard/describe.go:25` | `llmprovider.ErrProviderUnavailable,` |
| C13 | ErrOffline, a child of unavailable | `llmprovider/api_error.go:71` | `ErrOffline error = &sentinelError{msg: "llmprovider: offline", parent: ErrProviderUnavailable}` |
| C14 | The endpoint check's notice | `wizard/configure.go:489` | `p.Notify(LevelWarn, "cannot reach %s: %v", url, vErr)` |
| C15 | No service text in a notice | `wizard/configure.go:187` | `service's error is shown as its class and HTTP status, never its text` |
| C16 | A refused URL quoted whole | `wizard/base_url.go:36` | `has a query or a fragment", errBaseURLRefused, raw)` |
| C17 | Notify shows no credential | `wizard/prompter.go:92` | `Notify reports progress or a problem. It must not be used to display a` |
| C18 | A menu row printed as it is | `wizard/text_prompter.go:208` | `c.Label, fill, c.Detail, mark)` |
| C19 | Notify strips control characters | `wizard/text_prompter.go:181` | `redact.StripControl(fmt.Sprintf(format, args...))` |
| C20 | A failed listing: the fallback and the error | `wizard/configure.go:694` | `using the built-in catalog", d.Label, describe(err))` |
| C21 | A refused session ends the run | `wizard/configure.go:564` | `refused the sign-in; sign in again: %w` |
| C22 | Unknown Providers ids dropped | `wizard/configure.go:421` | `func selectableDescriptors(` |
| C23 | The guide's six | `docs/guides/adding-a-provider.md:225` | `The first six listed ids are recommended, in` |
| C24 | Cut to ModelLimit, zero cutting nothing | `wizard/configure.go:762` | `recommended := cutModels(models, o.ModelLimit)` |
| C25 | A package whose tests list opts in | `AGENTS.md:220` | `a package whose tests list models opts in the same way.` |
| C26 | capMatches | `wizard/model_select.go:370` | `func capMatches(` |
| C27 | save's failed Save | `wizard/auth.go:920` | `if err := c.saveTo.Save(ctx, d.ID, c.session); err != nil {` |
| C28 | A Kilo organization's name as a label | `wizard/auth.go:380` | `Choice{Label: org.Name, Detail: org.Role}` |
| C29 | An empty listing: the fallback, with cat.Err | `wizard/configure.go:700` | `return fallback, cat.Err` |
| C31 | keyOther, "a pasted credential" | `wizard/auth.go:64` | `keyOther       keySource = iota // not resolveAPIKey's: a pasted credential, a session` |
| C35 | Providers' doc | `wizard/configure.go:70` | `// Providers restricts the menu to these ids.` |
| C36 | The fallback has no Err | `wizard/configure.go:642` | `fallback := catalog.Catalog{Recommended: cutModels(static, o.ModelLimit), Usable: cutModels(static, o.ModelLimit)}` |
| C38 | A package added later opts in | `docs/decisions/0065-MADR-tests-reach-no-outside-host.md:126` | `opts in the same way.` |
| G1 | gobble lists | `gobble-cli:llm/provider/signin.go:179` | `Discover:         true,` |
| G2 | gobble asks no endpoint | `gobble-cli:llm/provider/signin.go:187` | `NoEndpointPrompt: true,` |
| G3 | gobble's Input with no terminal | `gobble-cli:internal/cli/prompt.go:89` | `return "", errNoTerminal }` |

## Amendment 2026-10-30: the owner's answers

* **The answers.** The owner answered Q1 to Q6 on 2026-10-30, each from
  three options put to them, with the recommended one first: (a) each.
  * Q1 (a): an interface in `catalog` through which a provider gives its
    whole catalog; `chatcompat`'s provider implements it, and the wizard's
    `providerCatalog` uses it before `ListModels`.
  * Q2 (a): an opt-in option that adds the servers `chatcompat.Discover`
    finds to the provider menu.
  * Q3 (a): both layers, `catalog` and `TextPrompter`.
  * Q4 (a): a refused kept session gives a notice, then the sign-in menu.
  * Q5 (a): the returned catalog's `Err` carries the listing's failure on
    every path.
  * Q6 (a): a notice names each unknown `Options.Providers` id.
* **What follows.** Q1 and Q2 each add an exported name, so the release is
  a minor one (D1). The PLAN lists each name with its call site (R55): the
  interface's is the wizard's `providerCatalog`, the option's the wizard's
  provider menu. R1 stands. This record stays `proposed` until the owner
  accepts it.

## Amendment 2026-10-30: facts found while planning (proposed)

Drafting
[0066-PLAN-wizard-v1-24-debugging-pass-findings.md](0066-PLAN-wizard-v1-24-debugging-pass-findings.md)
against the code found four facts this record did not weigh. Each is
proposed here, and approved or not with the PLAN; the earlier text stands as
it was written.

* **V6 corrected, and lowered to low.** `Catalog.Live` "reports whether
  Usable came from the provider's listing rather than the static catalog"
  (C39), and it is false on every path where the listing did not run or
  failed: probe L shows `Live=false` in each. So a caller can tell an
  unchecked credential today, by `Live`, and V6's "cannot tell a checked
  credential from an unchecked one" overstates it. What holds: on a
  provider's own `ListModels` path, and when a listing yields nothing,
  `Catalog.Err` does not carry the failure, which its doc says it is (C40),
  and `ResolveCredential`'s doc does not say that `Live` is the signal. Q5
  (a) is the fix still. One limit of the signal is recorded, not measured:
  Hugging Face's listing needs no credential (C41), so a live listing there
  says nothing of the key. Whether it refuses a wrong one was not probed.
  The count becomes 2 high, 6 medium and 6 low.
* **V2's rule leaves a local provider out, and offers a saved key.** Q1's
  preamble asks a key of "one with no `RequiresAPIKey` that the listing
  refuses". Ollama is such a provider, and its provider sends no key:
  Ollama's is "required but ignored", and no `Authorization` header is sent
  with `WithAPIKey` (C42). Asking would refuse the same request three times.
  So the rule is for a provider with no `RequiresAPIKey` and no `IsLocal`
  (C43), which `chatcompat.Descriptor` is (C3). Such a provider's saved key,
  `Existing.APIKey`, is offered to keep, as a required key's is, so a keyed
  server is not typed again on every run. A refused request that carried no
  key, Ollama's, is worded as that, not as "the pasted credential".
* **Q1 (a)'s name and shape.** The interface is `catalog.Lister`, not
  `CatalogLister`: `revive`'s `exported` rule, on in `.golangci.yml` (C44),
  refuses a name that repeats its package. Its method is
  `ListCatalog(ctx context.Context, opts ...llmprovider.Option) (Catalog,
  error)`: `opts` carry `WithProfile` and `WithLimit`, which `chatcompat.New`
  refuses as options scoped to the built-ins, so the catalog is cut by
  `ListCatalog`'s caller as `List` cuts it, and a model past the limit keeps
  its facts for search (0061-MADR, amendment "S1 amended in execution").
* **Q2 (a)'s details.**
  * A found server's id is `local-<port>`, such as `local-1234`: stable
    across runs, never a built-in's, which `chatcompat.New` refuses.
  * A server on the port of a local provider the `Registry` holds, Ollama's
    `11434`, is not added: it is Ollama, offered already.
  * `Discover` is given the caller's `HTTPClient` only. `ProviderOptions`
    may hold an option it refuses (C45).
  * The Result needs no new field. A Result whose `Provider` the caller's
    `Registry` lacks is a found server, which `LocalServers`' doc says, with
    the `chatcompat.New` call that builds it.

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C39 | Live: the provider's listing, not the static | `llmprovider/catalog/discovery.go:139` | `// Live reports whether Usable came from the provider's listing rather` |
| C40 | Err: the failure that made Live false | `llmprovider/catalog/discovery.go:143` | `// Err is the listing failure: the one that made Live false, or, on a` |
| C41 | Hugging Face's listing is public | `llmprovider/catalog/discovery.go:1156` | `The endpoint is PUBLIC (200 with` |
| C42 | Ollama sends no key | `llmprovider/providers/ollama/ollama.go:6` | `Authorization header is sent, with or without WithAPIKey.` |
| C43 | IsLocal | `llmprovider/registry.go:28` | `// IsLocal reports whether the provider runs on the user's machine.` |
| C44 | revive's exported rule | `.golangci.yml:83` | `- name: exported` |
| C45 | Discover refuses an option ResolveOptions refuses | `llmprovider/providers/chatcompat/discover.go:61` | `identity. It returns an error for an option ResolveOptions refuses, and` |

## Amendment 2026-10-30: issue #10, a bounded client that headers can wrap (proposed)

On 2026-10-30 the owner opened
[issue #10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10), "a way
to add request headers that keeps the SDK's bounded HTTP client. Headers from
models.json drop its 300 s timeout today", with "We need to figure out the
best way to implement this", and asked that it be assessed, specified, and
added to this pair. It is not the wizard's, but it rides the same minor
release, and the wizard's `Options.HTTPClient` is one of its paths.

### The facts

* **The bounded client.** `transport.DefaultClient` (C46) is every
  provider's client when the caller gives none (C48): a new `http.Client`
  per call, over one transport the process shares, so instances share
  connections. Its bounds are in that transport: a dial within 30 s, a TLS
  handshake within 10 s, the first byte of a reply within 300 s
  (`ResponseHeaderTimeout`, C47), idle connections closed after 90 s, and 16
  kept per host. It has no total timeout: a streamed reply runs while it
  sends.
* **What `WithHTTPClient` replaces.** A client given with it is used as it
  is (C48), for requests, listings and a session's refresh (R32). Its doc
  says only "a custom HTTP client for connection pooling" (C49).
* **0061-MADR D6** made a caller's `RoundTripper`, given through
  `WithHTTPClient`, the supported way to send headers, "No new API", and
  said "The package doc of llmprovider says so in a sentence under
  WithHTTPClient". That sentence was never written: no Go file outside the
  tests, nor `README.md`, `docs/architecture.md` or a guide, names a
  `RoundTripper` for headers, and 0061-PLAN has no step for it.
* **gobble-cli,** at `61f7115`, resolves each request's headers from
  `models.json` for that request and its model, and fails the request unsent
  when they cannot be resolved (G4). With no client of its own it wraps
  `http.DefaultTransport` (G5). `net/http`'s `DefaultTransport` has the same
  30 s dial, 10 s handshake and 90 s idle bounds, and no
  `ResponseHeaderTimeout`. So its requests lose the 300 s bound to the
  first byte: an endpoint that accepts the connection and never answers
  holds the request until its context ends. They also leave the SDK's
  shared connections, and keep two idle ones per host, not 16. Rebuilding
  the SDK's transport in gobble-cli would copy numbers that the SDK owns
  and may change.
* **Other Go SDKs.** openai-go has `option.WithHeader` and
  `option.WithMiddleware`, and its `WithHTTPClient` doc recommends an
  `*http.Client` with a custom transport. HashiCorp's `go-cleanhttp` exports
  `DefaultPooledClient`, a new client over a shared, bounded, pooled
  transport, for a caller to use or wrap.

### The findings

* **V15 (medium). A caller that adds headers loses the SDK's bounded
  client** **[read]** (C46 to C49; G4, G5). The only supported way to add a
  header is a caller's client, and the SDK gives no way to start from its
  own. A caller that wraps `http.DefaultTransport`, as gobble-cli does, has
  no bound to the first byte.
* **V16 (low). 0061-MADR D6's sentence was never written** **[read]**. The
  supported way is documented nowhere a caller reads.

The count becomes 2 high, 7 medium and 7 low, with the amendment "facts
found while planning".

### The options

* **H1. `llmprovider.NewHTTPClient() *http.Client`.** A new client with the
  module's bounds, over the process's shared transport: the client a
  provider builds when given none. A caller wraps its `Transport` in its
  own `RoundTripper` and gives the client with `WithHTTPClient`, or as the
  wizard's `Options.HTTPClient`. One exported function; no provider
  changes.
* **H2. A common option, `WithMiddleware(func(http.RoundTripper)
  http.RoundTripper)`,** that wraps whichever transport the provider uses,
  as openai-go's `WithMiddleware` does.
* **H3. `WithHeader(key, value)`, or 0061's declined `WithHeaderFunc`.**
* **H4. Documentation only:** a caller clones `http.DefaultTransport` and
  sets `ResponseHeaderTimeout` to 300 s.

### Pros and cons

* **H1:**
  * Good, because it is one name, and the idiom is `net/http`'s own: a
    client whose transport the caller wraps (`go-cleanhttp`'s
    `DefaultPooledClient`).
  * Good, because 0061-MADR D6's mechanism stands, and R28 and R32 are
    unchanged: no provider, `catalog` or `auth` code changes. A client given
    with `WithHTTPClient` already serves requests, listings and refreshes,
    and the wizard passes it on.
  * Good, because the bounds stay the SDK's, and a caller's requests share
    its connections, from one `NewHTTPClient` or many.
  * Neutral, because a caller that wants a request's model in its header
    still puts it in the context itself, as gobble-cli does (0061-MADR D6).
  * Bad, because the client's transport is a `RoundTripper` to wrap, not an
    `*http.Transport` to configure. A caller that wants other bounds builds
    its own client, as today.
* **H2:**
  * Good, because no client is handled by the caller.
  * Bad, because every provider, `catalog`'s listings, `auth`'s flows and
    the sessions' refreshes must each apply it, or it is a header on some
    requests and not others. R28 and R32 would change. Twelve packages to
    change and test (the ten under `llmprovider/providers/`, `catalog` and
    `auth`) for what H1 does with none.
* **H3:**
  * Good, because the simplest case, a fixed header, is one call.
  * Bad, because a fixed header is not gobble-cli's: its headers are
    resolved per request and model and can fail (G4). `WithHeaderFunc` is
    the hook 0061 declined, and H3 has H2's reach problem.
* **H4:**
  * Good, because no API is added.
  * Bad, because the caller copies the SDK's bounds, which drift, and its
    requests do not share the SDK's connections.

### The proposed decision, D2

**H1** (proposed), because it closes V15 with one function and no change to
how any provider sends a request, by the idiom `net/http` and
`go-cleanhttp` set, and it writes the documentation V16 lacks. It keeps
0061-MADR D6's mechanism, a caller's `RoundTripper`; it replaces D6's "No
new API" with this one name. 0061's records are not changed; this record is
the later one.

* **Signature:** `func NewHTTPClient() *http.Client`, in `llmprovider`
  beside `WithHTTPClient` (R17). It returns `transport.DefaultClient()`.
* **Its doc:** each call returns a new client over the transport the
  process shares, with the bounds above; it is the client a provider
  builds when given none. Its `Transport` is a `RoundTripper` to wrap, not
  an `*http.Transport`. It is safe for concurrent use, as an `http.Client`
  is (R51); it returns no error (R52).
* **The documentation V16 lacks:** `WithHTTPClient`'s doc, and the
  `llmprovider` package doc, say how to add headers: wrap
  `NewHTTPClient()`'s transport, and put what the transport needs, such as a
  request's model, in the request's context. The wizard's
  `Options.HTTPClient` doc and `README.md` say the same.
* **An example (R50):** `ExampleNewHTTPClient`, a header added by a wrapped
  transport, sent to an `httptest` server, with its `// Output:`.
* **gobble-cli** changes `withHeaders` to wrap `NewHTTPClient()`'s
  transport in place of `http.DefaultTransport`, in its own repository and
  record.

* **Q7 (V15, V16). Issue #10's way.**
  * **(a)** H1, `llmprovider.NewHTTPClient` (recommended).
  * **(b)** H2, a `WithMiddleware` common option.
  * **(c)** H3, `WithHeader` and `WithHeaderFunc`.

  **Answered 2026-10-30: (a).** *(Q7: answered by the owner from three options,
  the recommended one first. D2 stands as proposed.)*

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C46 | The bounded default client | `llmprovider/internal/transport/transport.go:51` | `func DefaultClient() *http.Client {` |
| C47 | 300 s to the first byte | `llmprovider/internal/transport/transport.go:72` | `ResponseHeaderTimeout: 300 * time.Second,` |
| C48 | Used only when no client is given | `llmprovider/settings.go:227` | `s.cfg.HTTPClient = transport.DefaultClient()` |
| C49 | WithHTTPClient's doc | `llmprovider/options.go:44` | `// WithHTTPClient sets a custom HTTP client for connection pooling. Nil is the` |
| G4 | gobble's per-request headers | `gobble-cli:llm/provider/provider.go:66` | `Headers func(ctx context.Context, model string) (map[string]string, error)` |
| G5 | gobble wraps net/http's transport | `gobble-cli:llm/provider/provider.go:152` | `base := http.DefaultTransport` |
