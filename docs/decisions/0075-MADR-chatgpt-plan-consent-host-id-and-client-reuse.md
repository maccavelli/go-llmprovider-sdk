---
status: accepted
date: 2026-11-12
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b), its §6, findings F20, F22 to F24 and F29, ledger S7 (the host id), S8 and S9, and the owner's decisions of 2026-11-10 on its Q2 to Q5 and Q12; 0017-REPORT-openai-sign-in-for-agents.md (OpenAI's flow and terms); this repository at 773b84c; re-checked 2026-11-11 against gobble-cli at 17d1605 (its origin/main; the 0031-MADR amendment committed at 86ad4a7), magic-cli-remote at eefb7c25 (its origin/master, the 0245 report unchanged since 3e2f784b) and its proposed, untracked 0051 and 0052 pairs, and magic-cli-remote's proposed, untracked 0246 and 0247 pairs"
informed: gobble-cli, which classifies and gates its background requests and keeps the wording; magic-cli-remote, which relays the confirm step; every caller of LoginChatGPTPlan and the wizard's plan sign-in
---

<!-- markdownlint-disable MD013 MD024 -->

# Carry the ChatGPT Plan's Background-Use Consent on the Session, Ask It Before the Browser, and Keep the Issued Client and the Host Id Apart from the Session

## Context and Problem Statement

OpenAI's terms for Sign in with ChatGPT ask an app to "Obtain express consent
before background use" (C1). This SDK's plan sign-in
([0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md))
left that to the consumer: D8 says the SDK cannot meet it, "and no option
claims to" (C2). The package doc says the consumer "obtains the user's
express consent before any background use" (C3). The 0017 report foresaw
the gap: "An SDK can carry the consent as an option and expose the usage URL;
it cannot meet them alone" (C4).

The result, in the 0245 report's words (F29 to F32): every plan consumer
re-implements consent, no client can learn in advance that one is needed,
and gobble's hand-rolled version reaches no one remotely. The wizard shows a
second, separate notice (C5).

Two more plan defects belong to the same sign-in:

* **The host id belongs to a sign-in, not to the host** (F24). With no
  `HostID` and no `Previous`, the sign-in mints a new random one (C6, C7).
  OpenAI's guide says the id is "persisted before the first sign-in and kept
  for the host's life" (C8). The wizard passes none (C9).
* **The issued client is lost with the session** (F22, F23). A re-sign-in
  reuses the issued client only through `Previous`, a plan session (C10,
  C11). The wizard's logout deletes the session (C12), and a plan sign-in
  replaces any other OpenAI session in the store (0043-MADR D3, C13), so
  after either the next plan sign-in registers yet another client: "Each run
  registers a new client under the account, listed in its ChatGPT settings"
  (C14).

The owner decided all five questions on 2026-11-10 (the report's §11): Q2
option C, the split; Q3 consent on the session; Q4 asked before the browser,
recorded on save; Q5 an opt-in enforcement helper; Q12 one slot per provider,
with the issued client id and host id kept apart from the session. This
record decides how.

### What the code does at `773b84c`

* **The session record** has `Resource`, `Scope`, `IDToken` and `HostID`
  for a plan (C15), and nothing for consent.
* **A refresh does not re-read the store when the stored refresh token is
  its own.** `loadRotated` returns nothing for a record with the same
  refresh token (C16), or with the token this session's unsaved rotation
  spent (C40), and the refreshed session is then saved from the
  session's own fields (C17); an unsaved rotation is re-saved the same way
  (C18). So a field another process changed in the store, without rotating
  the token, is overwritten by the next refresh. The 0245 report's §6.4 says
  that under consent-on-the-session "'from the next request' holds, because
  the session is re-read per request". The code does not do that (C16, C17):
  this record must, for consent.
* **The openai provider** knows a plan session (C19) and refuses, before the
  network, what a plan cannot take (C20). It has no notion of a background
  request.
* **The notice.** The wizard asks "You're using your ChatGPT plan: this app's
  requests count toward its usage, which you can manage at %s. Continue?",
  default yes; "no" declines the sign-in (C5, C21). The usage page is the Go
  constant `openai.ChatGPTUsageURL` (C22).

### Evidence index

<!-- claims: self=773b84c gobble-cli=17d1605 magic-cli-remote=eefb7c25 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The terms ask for consent | `docs/reports/0017-REPORT-openai-sign-in-for-agents.md:164` | `Obtain express consent before background use.` |
| C2 | 0043-MADR D8 | `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md:262` | `pooling, and no charge (terms §2 and §4). No option claims to.` |
| C3 | The package doc | `llmprovider/auth/doc.go:23` | `express consent before any background use.` |
| C4 | The 0017 report's open item | `docs/reports/0017-REPORT-openai-sign-in-for-agents.md:291` | `can carry the consent as an option and expose the usage URL; it cannot` |
| C5 | The wizard's notice | `wizard/auth.go:278` | `use, err := p.Confirm(fmt.Sprintf("You're using your ChatGPT plan: this app's requests count toward its usage, "+` |
| C6 | A host id is minted | `llmprovider/auth/oauth_plan.go:224` | `func planHostID(opts ChatGPTPlanOptions) (string, error) {` |
| C7 | as a random urn:uuid | `llmprovider/auth/oauth_plan.go:237` | `return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil` |
| C8 | OpenAI's rule for the host id | `docs/reports/0017-REPORT-openai-sign-in-for-agents.md:68` | `**The host identifier** is persisted before the first sign-in and kept for` |
| C9 | The wizard passes none | `wizard/auth.go:291` | `session, loginErr := loginChatGPTPlan(ctx, auth.ChatGPTPlanOptions{AgentName: o.AgentName, Previous: previous,` |
| C10 | A re-sign-in reuses `Previous`'s client | `llmprovider/auth/oauth_plan.go:259` | `query.Set(oauthParamClientID, previous.ClientID)` |
| C11 | `Previous` must be a plan session | `llmprovider/auth/oauth_plan.go:197` | `if opts.Previous != nil && (!opts.Previous.PlanUsage() \|\| opts.Previous.ClientID == "") {` |
| C12 | Logout deletes the session | `wizard/logout.go:53` | `if err := o.TokenStore.Delete(ctx, id); err != nil {` |
| C13 | 0043-MADR D3 | `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md:154` | `` under `openai` replaces a Codex-id session there, and the reverse; a `` |
| C14 | Each run registers a client | `docs/reports/0017-REPORT-openai-sign-in-for-agents.md:293` | `**The probe's runs.** Each run registers a new client under the account,` |
| C15 | The record's plan fields | `llmprovider/auth/tokenstore.go:205` | `` HostID   string `json:"host_id,omitempty"` `` |
| C16 | A record with the same token is not read | `llmprovider/auth/oauth_session.go:876` | `if err != nil \|\| stored == nil \|\| stored.Refresh == "" \|\| stored.Refresh == state.refresh \|\|` |
| C17 | The refreshed session is saved from its own fields | `llmprovider/auth/oauth_session.go:862` | `out.saveErr = state.store.Save(ctx, state.provider, next)` |
| C18 | and an unsaved rotation too | `llmprovider/auth/oauth_session.go:331` | `return state.store.Save(ctx, state.provider, next)` |
| C19 | The provider knows a plan session | `llmprovider/providers/openai/openai.go:181` | `plan      bool` |
| C20 | and refuses before the network | `llmprovider/providers/openai/openai.go:351` | `if p.plan && slices.ContainsFunc(req.Input, func(item llmprovider.Item) bool {` |
| C21 | "No" declines the sign-in | `wizard/auth.go:284` | `return resolvedCredential{}, errors.New("wizard: ChatGPT plan sign-in declined")` |
| C22 | The usage page is a constant | `llmprovider/providers/openai/openai.go:162` | `const ChatGPTUsageURL = "https://chatgpt.com/settings/usage"` |
| C23 | 0043-MADR D7's notice | `docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md:251` | `Before the first plan sign-in, the wizard shows OpenAI's one-time notice,` |
| C24 | A later sign-in reuses the client with a hint | `docs/reports/0017-REPORT-openai-sign-in-for-agents.md:65` | `id_token_hint` |
| C25 | `id_token_hint` only with `Previous`'s token | `llmprovider/auth/oauth_plan.go:261` | `query.Set("id_token_hint", previous.IDToken)` |
| C26 | The provider's entry point | `llmprovider/providers/openai/openai.go:303` | `func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {` |
| C27 | A store is one session per provider | `llmprovider/auth/tokenstore.go:167` | `// TokenStore persists OAuth sessions, one per provider. Load returns nil and no` |
| C28 | The plan's options take a host id | `llmprovider/auth/oauth_plan.go:60` | `HostID string` |
| C29 | The wizard holds one token store | `wizard/configure.go:121` | `TokenStore auth.TokenStore` |
| C30 | `auth` finds a `RefreshLocker` by a type assertion | `llmprovider/auth/oauth_session.go:318` | `if locker, ok := state.store.(RefreshLocker); ok {` |
| C31 | A wizard test of the first plan sign-in answers a `Confirm` | `wizard/plan_0043_test.go:108` | `confirms: []bool{true},` |
| C32 | and one of the notice declined | `wizard/plan_0043_test.go:155` | `confirms: []bool{false},` |
| C33 | and one that replaces a Codex session | `wizard/plan_0043_test.go:207` | `confirms: []bool{true},` |
| C34 | The re-sign-in test stores a plan session with no answer | `wizard/plan_0043_test.go:177` | `store.sessions[llmprovider.ProviderOpenAI] = planSessionFor("oaiapp_prev")` |
| C35 | `ResolveCredential`'s plan test answers a `Confirm` | `wizard/resolve_0056_test.go:76` | `f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t)}, confirms: []bool{true}})` |
| C36 | and so does the no-paste plan test | `wizard/paste_0057_test.go:65` | `f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t)}, confirms: []bool{true}})` |
| C37 | and the bare-error plan test | `wizard/signin_errors_0049_test.go:46` | `confirms: []bool{true},` |
| C38 | The fake refuses an unscripted `Select` | `wizard/fake_prompter_test.go:106` | `return 0, fmt.Errorf("fakePrompter: unexpected Select(%q)", title)` |
| C39 | and reports a scripted answer left unused | `wizard/fake_prompter_test.go:86` | `t.Errorf("fakePrompter: %d scripted %s left unused", left[kind], kind)` |
| C40 | nor a record holding the token an unsaved rotation spent | `llmprovider/auth/oauth_session.go:877` | `(state.stored != "" && stored.Refresh == state.stored) {` |
| C41 | `FileTokenStore`'s lock wait | `llmprovider/auth/tokenstore_file.go:26` | `const lockWait = 60 * time.Second` |
| C42 | A ctx that ends during the wait gives its own error | `llmprovider/auth/tokenstore_file.go:289` | `return nil, ctx.Err()` |
| C43 | The openai provider holds the session as a `TokenSource` | `llmprovider/providers/openai/openai.go:177` | `src     llmprovider.TokenSource` |
| C44 | `adopt` rewrites the session's `Store` under its lock | `llmprovider/auth/oauth_session.go:588` | `s.Store = next.Store` |
| C45 | `Store` is an exported field, beside the locked ones | `llmprovider/auth/tokenstore.go:45` | `Store      TokenStore` |
| C46 | The report's §6.4 | `magic-cli-remote:docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md:866` | `because the session is re-read per request.` |
| C47 | The report's §9.2 names the downgrade loss | `magic-cli-remote:docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md:1141` | `The downgrade loss of new fields is the` |
| C48 | gobble's 0031-MADR, superseded in part | `gobble-cli:docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md:406` | `*Superseded in part 2026-11-10 by the owner's answers of that date to magic-cli-remote's` |
| C49 | its consent setting superseded | `gobble-cli:docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md:421` | `"Consent is asked once at the plan sign-in and saved as a setting"` |
| C50 | its notice before the browser kept | `gobble-cli:docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md:435` | `("a notice at sign-in, before the browser opens") stands; the notice is now the` |
| C51 | gobble's 0025-MADR D6 | `gobble-cli:docs/decisions/0025-MADR-provider-choice-and-auth.md:224` | `` a stable `HostID` in the state directory `` |

## Decision Drivers

* **The owner's decisions of 2026-11-10:**
  * **Q2, option C.** "The SDK holds the consent fact, a `confirm` step with
    the stable id `chatgpt_plan.background_use`, a storage contract and an
    optional enforcement helper. gobble holds the wording, the background
    classification, the gating and the setting UI. Magic relays."
  * **Q3, on the session.** "`OAuthSession` consents persisted by the token
    store, changed through an SDK `SetConsent` helper, and cleared with the
    session at sign-out."
  * **Q4, before the browser.** "Before the browser opens, with the notice,
    as one `confirm` step. It is recorded only if the sign-in then saves."
  * **Q5, opt-in enforcement.** "A context marker for background requests.
    The openai provider refuses a marked request on a plan session with no
    recorded consent (`ErrNotPermitted`)."
  * **Q12, one slot, keep the ids.** "The plan's issued client id and host id
    are kept apart from the session, so a sign-out or switch never forces a
    new client registration."
  * **Q11:** a plan sign-in from the phone is allowed, disclosed by the
    phone's `confirm` step.
* **The standing constraints.** No gobble baggage in the SDK: the stable id
  is generic (`chatgpt_plan.background_use`), never gobble's setting name;
  the wording, the classification of background requests and the setting
  belong to gobble.
* **This repository's rules:** `OAuthSession` stays comparable (0043-MADR
  A1), so consents are a string; R48 for every name outside `llmprovider/x/`;
  the refresh code is not rewritten
  ([0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
  D5), and a change to it is the narrowest one that keeps a consent alive.
* **"From the next request."** A user who withdraws consent in one process
  must see background use stop at the next background request in every
  process that holds the session.

## Considered Options

The owner chose among the report's options for each question. The ones
weighed here are the report's §6.2 and §6.4:

* **A. All in gobble** (the report's option A, gobble's work in progress).
* **B. All in the SDK** (option B): wording, classification and gating too.
* **C. Split** (option C): the fact, the step, the storage and an opt-in
  helper in the SDK; wording, classification, gating and setting in gobble.

For where the answer is kept (the report's §6.4): (a) on the session; (b) in
the app's settings; (c) both, the setting writing through the helper.

## Decision Outcome

Chosen option: "C. Split", with the answer kept on the session ((a)),
because it gives every plan consumer the requirement, the question and a
compliant store, while the app keeps what only it can know: which of its
requests are background, and how to word the question. The owner chose these
options on 2026-11-10 (Q2 to Q5 and Q12). This record is `proposed`: it
awaits the owner's review of this draft.

Two points this draft raised were answered by the owner the same day, each as
recommended:

* **"No" at the plan's confirm step declines background use only.** The
  sign-in goes on, and a third choice of the wizard's `Select` declines the
  sign-in (D4).
* **If OpenAI refuses the issued client without `id_token_hint`,** the release
  is gated on the PLAN's owner-run probe. The fallback is the one D7 states:
  stop passing `ClientID` and keep only the host id.

On 2026-11-11 the owner answered the question that day's review had left
open in D5, choosing option (a) of the PLAN's Phase 3: the session gains an
exported reader, `StoredConsent`, through which the openai provider reads
the stored answer, and a failed read is refused as not granted. The record
stays `proposed`: the owner has answered a question, not accepted it.

The owner accepted this record on 2026-11-12, in the coordinator session,
and approved its PLAN the same day. Its code follows 0074's and waits, as
0074's does, for `ci.yml` to pass on `main` after
[0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md](0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md).

### D1. The consent is a fact of the method

* `llmprovider.AuthMethod` gains three comparable string fields, beside
  0074-MADR D1's: `Consent` (the id), `ConsentText` (the default wording) and
  `ConsentURL` (a page the wording links).
* The openai descriptor's `chatgpt_plan` sets:
  * `Consent`: `openai.ConsentBackgroundUse`, a new constant,
    `"chatgpt_plan.background_use"`;
  * `ConsentText`: the notice and the question in one: "You're using your
    ChatGPT plan: this app's requests count toward its usage, which you can
    manage on the usage page. Allow this app to also use your plan for
    background requests it starts on its own?";
  * `ConsentURL`: `openai.ChatGPTUsageURL` (C22).
* `signin.Methods` gives it as `Consents: [{ID, Required: true, Text, Links:
  [{Rel: "usage", URL}]}]`, the usage link in `Links` too, and `consent` in
  `Features`. A client learns before it starts that the method asks.

### D2. The answer is kept on the session

* **`OAuthSession.Consents string`:** space-separated `<id>=granted` or
  `<id>=declined` entries, sorted by id. A string keeps the struct
  comparable, as `Scope` does (0043-MADR A1). An id with no entry is
  unanswered.
* **`(*OAuthSession).Consent(id string) (granted, answered bool)`** reads it
  under the session's lock.
* **Persisted** by `FileTokenStore` as `consents`, `omitempty`; shown by
  `String`, `GoString`, `MarshalJSON` and `LogValue`, since it is not a
  secret.
* **Carried** by the five copies of 0074-MADR D5, and held by its field-copy
  guard.
* **Cleared at sign-out** with the session: `wizard.Logout` deletes the
  session (C12), and the consent with it. A consumer's own sign-out does the
  same by deleting its session.

### D3. `auth.SetConsent`, and a refresh that never loses an answer

* **`auth.SetConsent(ctx, store TokenStore, provider llmprovider.ProviderID, id string, granted bool) error`**
  changes the stored session's answer. When `store` is a `RefreshLocker`, it
  holds the provider's refresh lock throughout, so it never races a refresh.
  It loads the session, sets the entry, and saves. Errors: no session, or an
  empty id, match `llmprovider.ErrInvalidRequest`; the store's own errors
  are returned as they are (`ErrInvalidProvider` for a bad id, a plain error
  otherwise); a lock not taken within the store's wait (60 s for
  `FileTokenStore`, C41) matches `ErrProviderUnavailable`, as the refresh's
  does, and a `ctx` that ends first gives an error matching `ctx`'s (C42).
  A store that is not a `RefreshLocker` is loaded and saved unlocked, and a
  refresh in another process can then still race the save.
* **The store's answer wins over a refresh** (the gap C16 and C17 leave).
  Under the refresh lock, the refresh already loads the stored record (C16).
  When that record holds the session's own refresh token, or the token its
  unsaved rotation spent (C40), the two cases `loadRotated` passes over, the
  session saved next (C17) takes the record's `Consents`;
  so does an unsaved rotation re-saved over a record the store still holds
  as this session's (C18). `adopt` carries them into the live session. When
  the record holds another token, today's code adopts the whole record,
  consents included. Only `SetConsent` and a sign-in's first save write
  consents. This is the one change to the refresh path; its 66 tests
  (0074-PLAN) stay unchanged.

### D4. Asked before the browser, as one step with the notice

* **When.** `x/signin`'s plan sign-in starts with a `confirm` step when there
  is no plan session to sign in to again, or when that session has no answer
  for the method's consent. A re-sign-in whose session has an answer carries
  it to the new session and shows no confirm, as the notice is shown only
  before a first sign-in today (C23).
* **The step.** `confirm_id` is the consent's id; `text` is the app's wording
  when it gives one (`signin.Options.ConsentText` and
  `wizard.Options.ConsentText`, maps from a consent id to the app's text),
  else `ConsentText`; `links` holds the usage page; `default` is false, since
  the consent must be express.
* **The answer.** `true` records `granted`, `false` records `declined`, and
  the sign-in goes on either way: background use is a separate permission
  from the sign-in. Cancelling the sign-in is `Cancel`, or a `Driver`'s
  error.
* **Recorded only on save** (Q4). The answer is put on the session the
  sign-in returns. Nothing is written before the caller saves it, so a
  sign-in that fails or is discarded records nothing.
* **The wizard's rendering.** Its default renderer shows a consent `confirm`
  as a `Select` of three choices, so that a terminal user can still decline
  the sign-in at the notice, as today (C21): "Allow background use",
  "Do not allow background use" (the default) and "Cancel the sign-in". Its
  title is the step's `text` followed by the usage page's URL from `links`, so
  a terminal user sees the page, as today's notice shows it (C5). The last
  choice returns today's error, "wizard: ChatGPT plan sign-in declined". A
  `Driver` receives the plain step.

### D5. The opt-in enforcement helper

* **`llmprovider.WithBackground(ctx) context.Context`** marks the requests a
  caller makes on its own, not for the user's direct activity, and
  **`llmprovider.IsBackground(ctx) bool`** reads the mark. Unmarked requests
  are unaffected: the app decides what is background.
* **The openai provider** refuses, before the network, a marked request
  (`Generate` and `Stream`, C26) on a plan session whose consent is not
  granted, with an error matching `llmprovider.ErrNotPermitted` that names
  the consent id. It reads the answer from the session's `Store` when it has
  one, so a `SetConsent` in any process applies from the next background
  request; with no store, from the session itself. It reads through
  `StoredConsent`, below. Other sessions, keys and providers ignore the mark.
* **How the provider reaches the store: `StoredConsent`** *(the owner's
  ruling of 2026-11-11, option (a) of the PLAN's Phase 3)*. The provider
  holds the session as a `TokenSource` (C43) in another package. `Store` is
  an exported field (C45) that `adopt` rewrites under the session's lock
  (C44), so reading it unlocked from the provider is a data race, and D2's
  `Consent` reads only the session's own field. So the session gains an
  exported reader,
  **`(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err error)`**,
  which takes `Store` under the session's lock and loads from it. With no
  `Store` it gives the session's own answer, as `Consent` does, since the
  provider cannot tell the two cases apart without the same race. The PLAN's
  R55 table lists it, with the provider's check as its call site.
* **A failed read is refused** *(the owner's ruling of 2026-11-11)*. When
  `StoredConsent` returns an error, the provider refuses the marked request
  as not granted, before the network: an error matching
  `llmprovider.ErrNotPermitted` that names the consent id. The terms ask for
  express consent (C1), and an answer that cannot be read is not one.

  *(2026-11-11: these two bullets were one open question, "how the provider
  reaches the store". It said that no name in the PLAN's R55 table read the
  stored answer, and that the record did not say what a marked request gets
  when that read fails, and left both to the owner.)*
* **What stays the app's:** which requests to mark (gobble's titles and
  compaction), the setting's name and its UI, and its own gating if it
  wants more than this.

### D6. A host id for the host's life

* **`wizard.Options.HostID`** and **`signin.Options.HostID`** pass a
  consumer's host-lifetime id to `ChatGPTPlanOptions.HostID` (C28). gobble
  persists one in its state directory (gobble-cli
  `docs/decisions/0025-MADR-provider-choice-and-auth.md` D6, built by gobble-cli
  `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`
  D4).
* **Without one,** the id is `Previous`'s when there is a plan session to
  sign in to again (C6), else the stored client record's (D7), which is read
  only when there is none, else a new random one (C6), which D7 then keeps.
  `Previous`'s comes first because its client was registered with it.

### D7. The issued client and the host id, apart from the session

* **`auth.ClientRecord{ClientID, HostID string}`,** comparable, and an
  optional interface a `TokenStore` may implement:

  ```go
  type ClientStore interface {
      LoadClient(ctx context.Context, provider llmprovider.ProviderID) (ClientRecord, error) // zero when none
      SaveClient(ctx context.Context, provider llmprovider.ProviderID, c ClientRecord) error // zero removes it
  }
  ```

  An optional interface, as `RefreshLocker` is, because a method added to
  `TokenStore` would break every caller's store (R48).
* **`FileTokenStore` implements it,** in `<provider>.client` beside the
  session, with the session file's durable, owner-only write. Deleting the
  session leaves it.
* **`ChatGPTPlanOptions.ClientID`:** an issued client to sign in to again
  without a `Previous` session. The authorize request names it, sends no
  `agent_name_hint`, and, having no ID token, no `id_token_hint`. A callback
  naming another client is refused, as with `Previous` (0043-MADR D4, A3).
* **Where each finds the store.** `x/signin` takes it as
  `signin.Options.ClientStore auth.ClientStore`; nil means none. The wizard
  adds no option: it uses its `Options.TokenStore` (C29) when that store also
  implements `ClientStore`, found by a type assertion, as `auth` finds a
  `RefreshLocker` (C30).
  A consumer's own store implements it where it likes: gobble's, over
  `<state>/clients.json` (gobble-cli
  `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`
  D5).
* **Use.** The wizard and `x/signin`, given a `ClientStore`, load the record
  before a plan sign-in that has no plan session to reuse, and pass its
  client and host id. After a plan sign-in returns, they save the issued
  client and the host id used, at once: the registration happened at OpenAI
  whatever the caller then does with the session.
* **So** a sign-out, or a switch to a Codex-id session and back (0043-MADR
  D3, kept), reuses the issued client instead of registering another.
* **Unverified, and gated.** OpenAI's guide describes a later sign-in that
  reuses the client "with `id_token_hint`" (C24); this module sends the hint
  only from `Previous` (C25). Whether OpenAI accepts the issued client with no
  hint, after the last session was revoked, is not known. The PLAN's
  owner-run probe settles it before release. If OpenAI refuses, the wizard
  and `x/signin` stop passing `ClientID`, keep only the host id, and this
  decision is amended; registration churn then remains for a sign-out, and is
  recorded as such.

### D8. What this amends in 0043-MADR

* **D8, "What stays the consumer's"** (C2): amended. The requirement of
  consent, the question and its storage move into the SDK (D1 to D4), with
  an opt-in enforcement (D5). The classification of background requests, the
  wording and the gating stay the consumer's. The package doc (C3) and the
  README's login section say so.
* **D7, the wizard's notice** (C23): amended. The notice becomes the
  method's default `confirm` text (D1, D4), with the usage page as data; a
  "no" declines background use, and a third choice declines the sign-in.
* **D3, one OpenAI login per store** (C13): kept (Q12 (a)), and refined: the
  issued client and the host id outlive the session (D7).
* **D4's options:** `HostID` and `Previous` stay; `ClientID` is added (D7).

### Consequences

* Good, because every plan consumer gets the consent requirement, a
  compliant question before the browser, and a store for the answer, and a
  remote client can learn of it before it starts.
* Good, because a withdrawal applies from the next background request, in
  every process, and no refresh can lose an answer.
* Good, because one notice replaces two, and a phone sees it as an ordinary
  `confirm` step.
* Good, because a sign-out or a switch stops registering new clients, if
  OpenAI accepts the reuse (D7).
* Neutral, because enforcement is opt-in: an app that never marks a request
  gets no refusal, and is as compliant as its own gating.
* Neutral, because each marked request on a plan session reads the store
  once: a small file, for requests the app starts on its own.
* Bad, because a record from this version, refreshed and re-saved by an older
  version, loses its consents (the older record type has no field for them),
  and the next sign-in asks again. This is 0043-MADR D3's downgrade cost,
  again.
* Bad, because seven existing wizard tests reach a plan sign-in through the
  `stubPlanLogin` seam (C31 to C37), and each changes how it answers the
  notice or what it stores, though not what it asserts:
  * six script the notice as a `Confirm` and meet D4's `Select` instead, which
    the fake prompter refuses when no `Select` is scripted and whose unused
    `Confirm` it reports (C38, C39): `TestPlan_FirstSignIn_0043`,
    `TestPlan_NoticeDeclined_0043`, `TestPlan_ReplacesCodexSession_0043`,
    `TestResolveCredential_ChatGPTPlan_0056`, `TestNoPastePrompt_PlanSignIn_0057`
    and `TestConfigureLLM_PlanLoginErrorBare_0049`. Each scripts one more
    `Select` where it scripted the notice's `Confirm` (the third choice where
    it declined), and, where it reads the notice's text, reads it from the
    `Select`'s title;
  * `TestPlan_ReSignIn_0043` stores a plan session with no answer (C34), so
    D4 now asks it. Its stored session gains a recorded answer, and it keeps
    asserting that a re-sign-in with an answer shows no notice; a new test
    covers the re-sign-in without one.

  What each asserts is kept: the notice is shown with the usage page,
  declining ends the sign-in before any login, a re-sign-in reuses the
  previous session, and an error stays bare.
* Bad, because `OAuthSession` grows another field, `AuthMethod` three, and
  `auth` an interface, a record type and a function, each under the v1
  promise. *(2026-11-11: and `OAuthSession` a second reader, `StoredConsent`,
  by the owner's ruling.)*

### Confirmation

* The PLAN's tests pass, each seen to fail on its planted break.
* The refresh tests of 0074-PLAN pass unchanged, and the consent survives a
  refresh after another process's `SetConsent`.
* `make api-check` lists exactly the PLAN's table.
* `tools/claims.py` passes on this record.
* The owner-run probe of D7 is recorded before the release, and D7 amended if
  it fails.

## Pros and Cons of the Options

### A. All in gobble

* Good, because it matches 0043-MADR D8 and needs no SDK API.
* Bad, because it is terminal-only and cannot be relayed (F32), and its
  hand-rolled version has defects: asked at every sign-in, a "no" keeping an
  earlier "yes", and missing from three of gobble's paths (F30, F31).
* Bad, because no client learns in advance that consent is needed, and every
  other plan consumer re-implements it.

### B. All in the SDK

* Good, because one implementation is compliant by default.
* Bad, because the SDK cannot know which of an app's requests are background,
  nor its settings or wording: gobble baggage in the SDK, against the
  owner's constraint.

### C. Split

* Good, because each part sits where it can be done: the requirement, the
  question and the store in the SDK; classification and wording in the app.
* Good, because the remote relay is an ordinary `confirm` step.
* Neutral, because enforcement depends on the app marking its requests.
* Bad, because it amends 0043-MADR D8 and adds v1 API.

### (a) On the session, (b) in settings, (c) both

* (a) Good, because the answer is tied to the connection, as the terms tie
  consent, is cleared with it, and is the enforcement's one source. Bad,
  because a change needs the SDK's helper, and a refresh must carry it (D3).
* (b) Good, because it is easy to toggle. Bad, because it outlives the
  sign-in and can disagree with it, and a "no" must be written explicitly
  (F31).
* (c) Good, because it keeps (b)'s toggle. Bad, because two names hold one
  fact, kept equal by code. A consumer may still build (c) over (a), its
  setting writing through `SetConsent`.

## More Information

### Records this decision amends, applied on acceptance

Applied by the PLAN's Phase 0, step 3, each as a dated amendment quoting the
text it amends:

* **`0043-MADR-openai-chatgpt-plan-sign-in.md`:**
  * D8 (`:259-262`): "The package doc of `auth`, and the README's login
    section, say plainly that the SDK cannot meet these for the consumer:
    requests from the user's own runtime, consent before background use,
    "connected application only", no pooling, and no charge (terms §2 and
    §4). No option claims to." Amended by D8 above.
  * D7 (`:251-252`): "Before the first plan sign-in, the wizard shows
    OpenAI's one-time notice, "You're using your ChatGPT plan", and the usage
    page." Amended by D4.
  * D3 (`:153-155`): "`TokenStore` still keeps one session per provider. A
    plan sign-in saved under `openai` replaces a Codex-id session there, and
    the reverse; a person signs in with one of the two." Kept, refined by D7.
  * D4's struct: `ClientID` added (D7).
* **`0017-REPORT-openai-sign-in-for-agents.md`,** its open item "The terms in
  a consumer" (`:289-292`): a note that this record answers its consent half.

### Records in other repositories this bears on

None is changed by this repository.

* gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`
  (proposed, 2026-11-10, and not yet committed there; its PLAN's Deviation 1
  of 2026-11-11 archived the uncommitted P4b-2 work on a local branch) adopts
  D1 to D7: gobble's wording as `ConsentText`
  and its gate on the session's answer (its D2); the `chatgptPlanBackground`
  setting retired and `gobble auth consent` changing the answer through
  `SetConsent` (its D3, the owner's QF (a)); a host id in `<state>/host-id`
  (its D4); the issued client in `<state>/clients.json` (its D5, QG (a)); and a
  one-time migration that carries a stored `chatgptPlanBackground: true` with
  `SetConsent` (its D9; the report's G13, G16). gobble's
  `docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md`, as
  committed at gobble-cli 86ad4a7, records itself superseded in part (C48):
  its consent "saved as a setting" (C49) and its §2.4 consent setting by
  0052-MADR D2 and D3, while its §2.3 notice before the browser stands, now
  as the `confirm` step's text (C50). 0052-MADR builds
  `docs/decisions/0025-MADR-provider-choice-and-auth.md` D6 (a stable host
  id in the state directory, C51; consent asked once).
* gobble-cli `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  (proposed, not yet committed there) has its ACP sign-in set
  `signin.Options.HostID`, `ClientStore`, `ConsentText` and `Previous` (its
  D3), and lists `chatgpt_plan.background_use` among its honoured consents
  once 0052-PLAN P3 adds it (its D2).
* magic-cli-remote `docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  (proposed, not yet committed there) renders the plan's consent `confirm`
  with three outcomes: allow, decline background use (`confirm: false`, the
  sign-in goes on), and cancel. That is D4's answer, relayed. Its release
  order puts the minor carrying this record and 0074 after
  `authcontract/v0.1.0`. magic-cli-remote's
  `docs/decisions/0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md`
  (proposed) does not bear on this record: its table leaves gobble's plan
  sign-in to gobble's own paste, the `paste` step answered by ACP
  elicitation, and replays nothing for it.
* magic-cli-remote `docs/decisions/0074-MADR-remote-provider-auth-from-phone.md`
  D2 ("no vault"): unaffected. Magic relays the `confirm` step and stores
  nothing.

### Where the report differs from the code

* **§6.4's "the session is re-read per request"** (C46) is not so: a
  session reads the store only when it refreshes, and not even then for a
  record whose refresh token is its own, or the one its unsaved rotation
  spent (C16, C40). D3 and D5 make "from the next request" true.
* ~~**§9.2's downgrade note** says an older SDK still routes a plan session
  correctly, which holds; it does not say that an older SDK's re-save drops
  a newer field, which this record records as a cost.~~ Withdrawn
  2026-11-11: §9.2 does say it, "The downgrade loss of new fields is the one
  0043-MADR D3 already accepted" (C47). The report and this record agree.

### Revisions of 2026-11-11

A review against the code at `773b84c` and the sibling records as they stood
on 2026-11-11. The decisions are unchanged except where a bullet says so.

* **Context, the refresh; D3.** `loadRotated` passes over two records, not
  one: the session's own refresh token, and the token its unsaved rotation
  spent (C40). D3's carry-over now names both; a consent set while a
  rotation is unsaved was otherwise lost at the next refresh.
* **D3, `SetConsent`'s errors.** The lock's wait is the store's (60 s for
  `FileTokenStore`, C41), and a `ctx` that ends first gives its own error
  (C42); "in time" did not say which. A store that is not a `RefreshLocker`
  is not serialised, which D3 now says.
* **D5.** Open question (2026-11-11), added: the provider has no race-free
  way to read the stored answer with the names this record exports (C43 to
  C45), and the behaviour on a failed read is not decided.
* **D6, the host id's order.** It read "the stored client record's, else
  `Previous`'s", but D7 reads the record only when there is no plan session
  to sign in to again, so the record could never come before `Previous`.
  Now `Previous`'s, else the record's, else a new one, as D7 already
  implied. This changes the PLAN's `TestHostIDOrder_0075`.
* **D7.** The refusal of a callback naming another client is 0043-MADR D4's,
  which A3 refines; the citation now names both.
* **More Information.** gobble-cli's 0052 pair is still uncommitted there,
  and its PLAN's Deviation 1 (2026-11-11) archived the P4b-2 work; gobble's
  0031-MADR amendment is committed at 86ad4a7 and keeps the notice before
  the browser, so "the notice placement" was not superseded (C48 to C50).
  gobble-cli's 0051 and magic-cli-remote's 0246 and 0247 are now named.
* **"Where the report differs".** The §9.2 bullet is withdrawn: the report
  does name the downgrade loss (C47).
* **The claims marker.** gobble-cli is read at `17d1605` and
  magic-cli-remote at `eefb7c25`, the pins 0073-MADR and 0076-PLAN use, so
  the series reads each sibling at one revision. Every row passes at both
  these and the earlier `86ad4a7` and `3e2f784b`.

The owner's ruling of 2026-11-11, written in after the review above.

* **D5, the open question answered.** Option (a): the exported reader
  `(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err
  error)`, which takes `Store` under the session's lock and loads from it,
  with the openai provider's check as its call site in the PLAN's R55
  table. A failed read is refused as not granted, an error matching
  `llmprovider.ErrNotPermitted` naming the consent id. D5's open question is
  replaced by the two bullets that say so. The PLAN's Phase 3 loses its
  "stop before this step", keeps `TestOpenAIBackground_ReadsStore_0075` and
  the `-race` run, and gains the failed read's assertion with a planted
  break.
* **The PLAN's order.**
  [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md),
  which raises the Go floor to 1.27.4, is its first step, before 0076's
  `v1.30.0`.

### Not verified

* Whether OpenAI accepts a re-sign-in with the issued client and no
  `id_token_hint` after revocation (D7; the PLAN's probe).
* Whether OpenAI regards a plan sign-in driven from the owner's phone, on the
  owner's own host, as compliant (the report's U12). The owner's stance is
  "own host only, disclosed" (Q11); this record supplies the disclosure step.

### Related

* PLAN: [0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md).
* The step engine and the method facts:
  [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md).
* The contract's `confirm` step and `Consent` type:
  [0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md).
