---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: this module at 150bcc2 and its v1.15.0 tag (b1bf9bb); gobble-cli at ea509b6 and prepare-commit-msg at a04b617, read beside it; issues 2 and 3; CI runs 39181623652 and 39182359773
informed: gobble-cli and prepare-commit-msg; anyone who writes a live test or changes a package's imports
---

<!-- markdownlint-disable MD013 -->

# Close v1.15.0's Follow-ups: Answer the Issues, Tell the Consumers, Prove the Person-Gated Live Tests, and Keep the Docs Tables True

## Context and Problem Statement

`v1.15.0` is released. It is tagged on `b1bf9bb`, the module proxy serves
it, and CI passed on `main` and on the tag:

* 0043-MADR-openai-chatgpt-plan-sign-in.md (the ChatGPT plan sign-in);
* 0044-MADR-observe-retries.md (`WithRetryNotify`);
* 0045-MADR-tests-never-reuse-a-freed-port.md (the test port races).

Their PLANs are `complete` (`150bcc2`). Four things were left over, and
nothing yet decides any of them:

1. **The issues.** Issues 2 and 3 asked for what shipped, and are still
   open. The comment posted on issue 2 before the design changed carries
   two statements that are no longer true.
2. **The consumers.** gobble-cli filed issue 3 and planned around both
   features; prepare-commit-msg uses the wizard. Neither knows that
   `v1.15.0` is out, nor what it would take to use it.
3. **The live tests a person must sign in to.** The owner's rule is that a
   test is unused until it has been seen to fail on a planted break.
   0043-PLAN says its live test was not, and the four older login live
   tests never were. Each try costs a person signing in.
4. **The docs tables.** R2 in `docs/guides/api-standards.md` says what each
   package may import; `docs/architecture.md` says what each depends on;
   `AGENTS.md` lists the live-test variables. All three have drifted from
   the code, and nothing checks them. One drift came from 0043-PLAN Phase
   4.

What has to be decided is how each is closed, in a way that stays closed.

### What was measured (2026-10-19)

Every fact below was measured from the trees named in `consulted`, and is
repeatable with the command given.

| # | Fact | How it was measured | Result |
| :--- | :--- | :--- | :--- |
| F1 | Issues 2 and 3 are the only open issues | the GitHub CLI, issue list in the open state | #2 and #3 |
| F2 | The owner's comment on issue 2 says the session gains `Scopes`, and that the 0043 pair is proposed | the GitHub CLI, issue 2's comments, the owner's last one, `#issuecomment-6250094676` | its line 26 names `Scopes`; its line 3 says "both are proposed". The field shipped as `Scope` (0043-MADR A1), and the pair is accepted and complete |
| F3 | `v1.15.0` adds 12 exported names and changes none | `tools/apidiff.py v1.14.0 v1.15.0`, run in this tree | exit 0; 12 lines, each `added` |
| F4 | The release passed every after-tag check | 0043-PLAN Phase 5 step 5's table | CI on `main` and on the tag, the proxy, `consumer.py`, `offline.py`: all passed |
| F5 | The record commit is not pushed | git, `origin/main..HEAD` after a fetch | one commit, `150bcc2` |
| F6 | Neither consumer uses any `v1.15.0` name | a grep for the 12 names over each consumer's non-test Go files | no match in either |
| F7 | The docs tables disagree with the imports in 13 places | a script that compares `go list` imports (host and `GOOS=windows`) with each table's cells; it is to become `scripts/check_tables.py` (D4) | 9 in `docs/architecture.md`, 4 in R2; the table under D4 lists them |
| F8 | `AGENTS.md` omits ~~three~~ eighteen live variables that a test reads (A1), and says one variable runs Grok's device login only | a grep for `LLMPROVIDER_LIVE_` over the live tests, against `AGENTS.md`'s list | `LLMPROVIDER_LIVE_GEMINI_429`, `…_429_MODEL` and `…_429_BURST` are missing; `LLMPROVIDER_LIVE_DEVICE_LOGIN` also runs Kilo's |
| F9 | Five live tests need a person to sign in, in three suites | a grep for the variable gates in the live tests | `BROWSER_LOGIN` (ChatGPT, Grok), `DEVICE_LOGIN` (Grok, Kilo), `CHATGPT_PLAN` |
| F10 | Live tests with a key have been planted live; no person-gated one has | a search of the records for live plants | key-gated: 0033-, 0038-, 0039-, 0040-, 0042-PLAN; person-gated: none, and 0043-PLAN says so |

### Code claims

`tools/claims.py` checks each row below: its text is on its line, at the
revision the marker names. gobble-cli and prepare-commit-msg are read
beside this repository with `git show`, never from their working trees.

<!-- claims: self=150bcc2 gobble-cli=ea509b6 prepare-commit-msg=a04b617 -->

| # | What | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | `wizard` imports `providers/openai` (0043-PLAN Phase 4) | `wizard/auth.go:19` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"` |
| C2 | `wizard` imports `internal/redact` | `wizard/auth.go:16` | `"github.com/maccavelli/go-llmprovider-sdk/internal/redact"` |
| C3 | the architecture row for `wizard` does not name `providers/openai` | `docs/architecture.md:91` | `` \| `llmprovider`, `auth`, `catalog`, `providers`, `internal/redact`, `golang.org/x/term` \| `` |
| C4 | R2's row for `wizard` names "the above", which lists `internal/redact` below it | `docs/guides/api-standards.md:48` | `` \| `wizard` \| the configuration flow, over a `Registry` \| the above, `golang.org/x/term` \| `` |
| C5 | the Responses wire imports `wire/deferral` | `llmprovider/internal/wire/responses/responses.go:16` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"` |
| C6 | `wire/deferral` imports `wire/messages` | `llmprovider/internal/wire/deferral/deferral.go:19` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"` |
| C7 | R2's row for the wire formats allows neither | `docs/guides/api-standards.md:46` | `` \| `llmprovider/internal/wire/<format>` \| one wire format \| `llmprovider`, `internal/wire`, `internal/transport`, `internal/redact` \| `` |
| C8 | the catalogs command imports `internal/transport` | `llmprovider/internal/cmd/catalogs/main.go:45` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"` |
| C9 | R2's row for it does not allow that | `docs/guides/api-standards.md:39` | `` \| `llmprovider`, `catalog`, `internal/catalogtool` \| `` |
| C10 | the architecture row for the command does not name it either | `docs/architecture.md:122` | `` \| `llmprovider`, `catalog`, `internal/catalogtool` \| `` |
| C11 | `kilo` imports `wire/messages` | `llmprovider/providers/kilo/kilo.go:88` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"` |
| C12 | the architecture row for `kilo` does not name it | `docs/architecture.md:102` | `` `internal/wire/chatcompletions`, `internal/kiloendpoint`, `internal/wire`, `internal/wire/deferral` \| `` |
| C13 | the architecture row for `openai` names `internal/transport`, which it does not import (one of six such rows) | `docs/architecture.md:97` | `` `internal/wire/responses`, `internal/transport`, `internal/wire`, `internal/wire/deferral` \| `` |
| C14 | `AGENTS.md` gives `DEVICE_LOGIN` to Grok only | `AGENTS.md:180` | `` `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok device-code `` |
| C15 | Kilo's device login live test reads the same variable | `llmprovider/live_oauth_login_test.go:136` | `if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {` |
| C16 | a live variable `AGENTS.md` does not list | `llmprovider/live_gemini_429_test.go:64` | `if os.Getenv("LLMPROVIDER_LIVE_GEMINI_429") == "" {` |
| C17 | the ChatGPT browser login live test's gate | `llmprovider/live_oauth_login_test.go:62` | `if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {` |
| C18 | the Grok browser login live test's gate | `llmprovider/live_oauth_login_test.go:85` | `if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {` |
| C19 | the Grok device login live test's gate | `llmprovider/live_oauth_login_test.go:109` | `if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {` |
| C20 | the plan live test's gate | `llmprovider/live_chatgpt_plan_test.go:33` | `if os.Getenv("LLMPROVIDER_LIVE_CHATGPT_PLAN") != "1" {` |
| C21 | what a login live test checks after the sign-in: one generation | `llmprovider/live_oauth_login_test.go:74` | `out, err := llmprovider.GenerateText(ctx, liveOpenAI(t, session, "gpt-6-astra"), userText("Reply with only the word ALPHA"))` |
| C22 | Kilo's reads the profile at the default endpoint, with no option | `llmprovider/live_oauth_login_test.go:153` | `account, err := auth.KiloProfile(ctx, session.Access)` |
| C23 | 0043-PLAN's own statement of the gap | `docs/decisions/0043-PLAN-openai-chatgpt-plan-sign-in.md:788` | `* **Not done:** the live test was not seen to fail on a planted break,` |
| G1 | gobble-cli pins `v1.13.0` | `gobble-cli:go.mod:16` | `github.com/maccavelli/go-llmprovider-sdk v1.13.0` |
| G2 | it wraps every provider in `WithRetry` | `gobble-cli:llm/provider/provider.go:197` | `w = llmprovider.WithRetry(p, policyOf(r))` |
| G3 | its default policy: 4 tries, 2 s, at most 60 s | `gobble-cli:llm/llm.go:42` | `var DefaultRetry = Retry{MaxRetries: 3, BaseDelay: 2 * time.Second, MaxDelay: 60 * time.Second, OfflineWait: 5 * time.Minute}` |
| G4 | its retry causes today are its own two | `gobble-cli:agent/event.go:86` | `RetryOffline    RetryCause = "offline"` |
| G5 | its terminal view already words any cause | `gobble-cli:internal/cli/view.go:185` | `attempts[argCause] = strings.ReplaceAll(r.Cause, "_", " ")` |
| G6 | it sends each retry to an ACP client | `gobble-cli:acpserver/extensions.go:54` | `const methodRetry = "_gobble/retry"` |
| G7 | its plan names cause `provider` for the SDK's retries, once issue 3 ships | `gobble-cli:docs/decisions/0005-PLAN-v1-feature-scope.md:5698` | `` The SDK's own retries get the same notification, `cause` `provider`, once the SDK offers a callback (issue 3), in a `` |
| G8 | it refuses an OpenAI login until the plan sign-in ships | `gobble-cli:internal/cli/auth.go:198` | `return usageErrorf("openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT for agents (0005-PLAN F4-3d); use --api-key -")` |
| G9 | its OAuth login is Grok's only | `gobble-cli:llm/provider/login.go:71` | `func HasLogin(id string) bool { return id == string(llmprovider.ProviderGrok) }` |
| G10 | its stored login has no field for a plan session's resource, scope, ID token or host id | `gobble-cli:llm/provider/login.go:20` | `Issuer, ClientID, AccountID, TokenURL string` |
| P1 | prepare-commit-msg pins `v1.12.0` | `prepare-commit-msg:go.mod:6` | `github.com/maccavelli/go-llmprovider-sdk v1.12.0` |
| P2 | it retries through `WithRetry`, with no `MaxDelay` | `prepare-commit-msg:main.go:392` | `policy := llmprovider.RetryPolicy{MaxAttempts: max(retries, 0) + 1, BaseDelay: delay}` |
| P3 | it runs the wizard | `prepare-commit-msg:internal/ui/setup.go:265` | `res, err := wizard.ConfigureLLM(ctx, prompter, wizard.Options{` |
| P4 | with a `TokenStore` | `prepare-commit-msg:internal/ui/setup.go:275` | `TokenStore:      store,` |
| P5 | it passes a saved session's client id back as `Existing` | `prepare-commit-msg:internal/ui/setup.go:245` | `existing.ClientID = session.ClientID` |
| P6 | its README says a retry is not shown | `prepare-commit-msg:README.md:463` | `is no longer logged.` |

## Decision Drivers

* **Every open loop of the release is closed,** and stays closed: a drift
  or an unproven test that can come back is checked, not just fixed once.
* **The owner's red-first rule holds for every test,** including those a
  person must sign in to, without a sign-in per planted break.
* **Each repository keeps its own records.** This module drafts what a
  consumer needs to know. It does not change a consumer's code, which
  needs that repository's own MADR.
* **No production behaviour, exported API or wire golden changes.** These
  are records, docs, tests, a check and two issue comments.
* **Anything published is first scanned** with `tools/denyscan.py`, and
  waits for the owner's word (`AGENTS.md`, Identifiers).

## Considered Options

* **A. One record, four parts:** each follow-up with its own decision,
  and the options it was chosen from.
* **B. A record for each follow-up.**
* **C. Answer the issues now, and leave the rest.**

## Decision Outcome

Chosen option: "A. One record, four parts", because the four come from one
release and share its evidence (F1–F10, C1–P6). One PLAN can then land
them in an order that publishes nothing before the owner asks. C would
leave the drift and the gap to come back.

### D1. The issues: one comment each, then close

* **Issue 2.** One comment, then close, on the owner's ask. It says:
  * that it shipped in `v1.15.0`, with the names (F3) and the wizard's
    `Options.AgentName`;
  * that the earlier comment (F2) had two errors: the field is `Scope`
    (0043-MADR A1), and the pair is accepted and complete;
  * what 0043-MADR A4 added after that comment: the three other
    `subscription_sharing_*` codes classified in a stream, and a system
    message in the input refused before the network;
  * what stays the consumer's (0043-MADR D8).
* **Issue 3.** One comment, then close, on the owner's ask. It says:
  * that `WithRetryNotify` and `RetryNotice` shipped in `v1.15.0`, with
    the six fields;
  * why it is a function and not the issue's `OnRetry` field: the field
    makes `RetryPolicy` incomparable, which `api-check` refuses
    (0044-MADR E1), and that would need a `v2`;
  * when notify is and is not called (0044-MADR D2).
* Each text is in the PLAN, scanned with `tools/denyscan.py`, and posted
  only on the owner's ask in that turn.
* **Rejected:**
  * closing with a link only, which leaves F2's errors standing in the
    issue;
  * keeping the issues open until the consumers adopt the release, which
    ties this repository's issues to other repositories' schedules.

### D2. The consumers: one adoption issue each, drafted here

The PLAN drafts one issue for each consumer, opened in that repository on
the owner's ask, after `tools/denyscan.py`.

* **gobble-cli** (G1–G10), from `v1.13.0`:
  * **Retry notices.** `WithRetryNotify` fills the slot its 0005-PLAN
    planned (G7): a notify that emits its `Retry` event with cause
    `provider` reaches its ACP client (G6) and its terminal view (G5)
    unchanged. Its policy's waits (G3) are what the user sees.
  * **The ChatGPT plan sign-in (F4-3d).** It needs code of gobble's own:
    its stored login cannot hold a plan session's `Resource`, `Scope`,
    `IDToken` or `HostID` (G10), its OAuth login is Grok's only (G9), and
    it refuses an OpenAI login (G8). The issue names these. The design is
    gobble's.
* **prepare-commit-msg** (P1–P6), from `v1.12.0`:
  * The plan sign-in needs one field: `AgentName` in the `wizard.Options`
    it already builds with a `TokenStore` (P3, P4). The saved client id it
    passes back (P5) makes the wizard default to the plan method on the
    next run.
  * `WithRetryNotify` would let it print a retry to stderr (P2, P6), if
    it wants to.
  * Its jump from `v1.12.0` crosses `v1.13.0` and `v1.14.0` too; the issue
    points at their notes.
* **Rejected:**
  * changing the consumers from here, which skips their own records;
  * telling no one, which leaves gobble-cli refusing a login that now
    exists (G8).

### D3. Person-gated live tests: their checks run offline too

A live test that needs a person keeps the steps after the sign-in in a
function, untagged, in package `llmprovider_test`. The live test calls it
with the real session. An offline test, its twin, calls it with a session
and an endpoint from a fake. A planted break is caught by the twin with no
sign-in. The live run still proves the service.

* **The five tests** (F9; C17–C21):
  * **ChatGPT browser login:** one generation on the Codex backend; the
    twin uses an SSE fake.
  * **Grok browser and device logins:** one generation each on the xAI
    Responses wire.
  * **Kilo's device login:** the token's shape (an access token only), the
    profile, and one generation. `KiloProfile` takes options, so the twin
    passes its fake's base URL (C22).
  * **The ChatGPT plan:** the session's kind, the store round trip, the
    listing with probes asked for, a generation, a stream, a forced tool,
    a forced refresh checked against the store, a revocation, and a
    refused refresh. The twin builds a plan session whose token and
    revocation endpoints are the fake's.
* **What a twin cannot cover:** the sign-in itself, which the offline
  tests of `LoginBrowserOAuth`, `LoginDeviceOAuth` and `LoginChatGPTPlan`
  in `auth` already prove against fake issuers (0043-PLAN Phase 2). It
  also cannot cover the service's own behaviour, which only the live run
  shows.
* **The rule** goes in `AGENTS.md`'s live-test section: a live test that
  needs a person keeps its checks in a function that an offline twin runs,
  and its planted breaks run on the twin.
* **Rejected:**
  * a planted break run live, with a person signing in for each: five
    tests times several breaks;
  * an exemption that names each test's offline counterparts, which
    leaves the checks in the live test unproven;
  * a session saved between runs so that one sign-in serves every
    planted break, which keeps a live refresh token on disk for the
    length of the run.

### D4. The docs tables: fixed, and checked

* **Fixed once,** in one change. The 13 places of F7, and F8:

  | Table | Package or entry | Drift | Fix |
  | :--- | :--- | :--- | :--- |
  | architecture | `wizard` | `providers/openai` missing (C1, C3) | add it |
  | architecture | `openai`, `claude`, `gemini`, `grok`, `ollama`, `internal/wire/responses` | `internal/transport` listed, not imported (C13) | remove it |
  | architecture | `kilo` | `internal/wire/messages` missing (C11, C12) | add it |
  | architecture | `internal/cmd/catalogs` | `internal/transport` missing (C8, C10) | add it |
  | R2 | `wizard` | imports `internal/redact`, below "the above" (C2, C4) | name it |
  | R2 | `internal/wire/<format>` | `responses` imports `wire/deferral` (C5, C7) | allow `internal/wire/deferral` |
  | R2 | `internal/wire/deferral` | under the wire-format row, imports `wire/messages` (C6, C7) | a row of its own: `llmprovider`, `internal/wire/messages` |
  | R2 | `internal/cmd/catalogs` | imports `internal/transport` (C8, C9) | allow it |
  | `AGENTS.md` | the live variables | `GEMINI_429`, `…_MODEL`, `…_BURST` missing; `DEVICE_LOGIN` "Grok" only (C14–C16); *(A1: and 15 record measurement switches)* | list the three; say Grok and Kilo; *(A1: state the family rule for the switches)* |

  R2's rows change by this record, which they will cite. Every import is
  legitimate, so the rule moves to the code rather than the reverse.
* **Checked from then on.** `scripts/check_tables.py`, run by CI, by
  `make tables-check` and by `tools/gate.py`, fails when:
  * an architecture row's "Depends on" differs from the package's imports;
  * a package imports what its R2 row does not allow;
  * a package has no row in either table;
  * `AGENTS.md`'s live variables differ from those the live tests read.
    *(A1: a record's measurement switch is accepted when that record
    names it.)*

  It reads `go list` for the host and for `GOOS=windows`, as
  `scripts/check_deps.py` does. `scripts/test_gates.py` proves it with a
  planted drift of each kind.
* **Rejected:**
  * fixing the tables by hand with no check, which is how F7 happened;
  * generating the columns from `go list`, which loses the tables' wording
    ("the above", "internal packages") and R2's role as a rule rather than
    a description.

### Consequences

* Good, because both issues end with an accurate account of what shipped,
  and F2's errors are corrected where they were made.
* Good, because each consumer learns, from its own issue, the exact lines
  that `v1.15.0` changes for it.
* Good, because every assertion of the five person-gated live tests can
  be proven on a planted break without a sign-in.
* Good, because a drift of the three tables fails CI from now on, at the
  change that causes it.
* Neutral, because the person-gated live tests still need a person to
  show the service accepts the sign-in; the twins prove the checks, not
  the service.
* Bad, because CI gains a check, and the gate goes from 23 checks to 24.
* Bad, because the five live tests are restructured, and each gains an
  offline twin and a fake to maintain.

### Confirmation

* **D1:** both comments posted and both issues closed, each on the
  owner's ask, after `tools/denyscan.py`; the PLAN records each comment's
  URL.
* **D2:** both drafts in the PLAN, checked against G1–G10 and P1–P6 with
  `tools/claims.py`; each opened on the owner's ask, with its URL
  recorded.
* **D3:**
  * the five live tests call their check functions;
  * five twins pass offline;
  * every assertion in a check function fails on a planted break, run on
    its twin with `tools/plant.py`;
  * one live run of each suite, with the owner signing in, passes on the
    restructured code;
  * `AGENTS.md` states the rule.
* **D4:**
  * `scripts/check_tables.py` exits 1 at `150bcc2` with the 13 places and
    F8, and 0 after the fix;
  * `scripts/test_gates.py` plants each kind of drift and sees it caught;
  * `make tables-check` and CI run it;
  * `tools/gate.py` passes 24 of 24.
* **The record itself:** `tools/claims.py` passes every row of the claims
  table at the revisions its marker names.

## Pros and Cons of the Options

### A. One record, four parts

* Good, because one set of evidence serves four decisions, and one PLAN
  orders them.
* Bad, because the record is longer than one decision's.

### B. A record for each follow-up

* Good, because each record holds one choice.
* Bad, because the shared evidence is split four ways, and the order
  between them (the docs check before the consumer issues cite the docs)
  sits in no one record.

### C. Answer the issues now, and leave the rest

* Good, because it is the least work.
* Bad, because the drift and the unproven live tests stay, and come back
  with the next change.

### D1's options

* **A comment, then close (chosen).** Good, because F2's errors are
  corrected in the issue that holds them.
* **Close with a link.** Bad, because F2's errors stay as the last word.
* **Keep open until the consumers adopt.** Bad, because this repository's
  issues wait on other repositories.

### D2's options

* **Drafted issues (chosen).** Good, because each repository decides its
  own adoption. Neutral, because nothing changes there until it does.
* **Changes made from here.** Bad, because they skip the consumers'
  records.
* **No notice.** Bad, because gobble-cli keeps refusing a login that now
  ships (G8).

### D3's options

* **Offline twins (chosen).** Good, because every check is proven with no
  sign-in. Bad, because five fakes are added.
* **Plants run live.** Bad, because each planted break costs a person
  signing in.
* **An exemption.** Bad, because the checks in the live tests are never
  seen to fail.
* **A session saved across runs.** Bad, because a refresh token stays on
  disk for the run.

### D4's options

* **Fixed and checked (chosen).** Good, because drift fails at its cause.
  Bad, because CI gains a check.
* **Fixed by hand.** Bad, because nothing stops the next drift.
* **Generated columns.** Bad, because R2 is a rule with wording a
  generator cannot keep.

## More Information

* The release: 0043-PLAN-openai-chatgpt-plan-sign-in.md, Phase 5 (its
  notes and after-tag checks); 0044-PLAN-observe-retries.md;
  0045-PLAN-tests-never-reuse-a-freed-port.md.
* The issues: [issue 2](https://github.com/maccavelli/go-llmprovider-sdk/issues/2)
  and [issue 3](https://github.com/maccavelli/go-llmprovider-sdk/issues/3);
  the comment F2 corrects:
  [`#issuecomment-6250094676`](https://github.com/maccavelli/go-llmprovider-sdk/issues/2#issuecomment-6250094676).
* The consumers' records cited: gobble-cli's
  `docs/decisions/0005-PLAN-v1-feature-scope.md` (F4-3d; issue 3), and
  prepare-commit-msg's
  `docs/decisions/0014-MADR-adopt-go-llmprovider-sdk-v1-11-0-and-go-selfupdate-lib-v1-15-2.md`
  (its last adoption, with an amendment for `v1.12.0`).
* **How this record was proven (2026-10-19),** before it was staged:
  * `tools/claims.py` passed all 39 rows of the claims table.
  * Five false claims, planted in scratch copies of this record, were each
    caught: a wrong line (C1, P4), a wrong row text (C13), a changed line
    in gobble-cli (G2), and a missing revision in the marker.
  * F7's measurement responds in both directions on a scratch clone: 13
    problems, then 12 with the `wizard` row fixed, then 13 again with a
    stale entry planted.
  * F6 was measured again with `git grep` at each consumer's `HEAD`: no
    match in either.
* The PLAN is written once this record is accepted:
  `0046-PLAN-v1-15-0-follow-ups.md`.

## Amendments

### A1. F8 undercounted the live variables (2026-10-19)

* **Found** writing 0046-PLAN, before any change. F8's grep used the
  pattern `LLMPROVIDER_LIVE_[A-Z_]*`, which stops at a digit. With digits
  (`LLMPROVIDER_LIVE_[A-Z0-9_]*[A-Z0-9]`), the live tests read 24
  variables, and `AGENTS.md` names 6 of them. The other 18 are:
  * `LLMPROVIDER_LIVE_GEMINI_429`, `…_429_MODEL` and `…_429_BURST`, as F8
    said;
  * 15 record measurement switches: `LLMPROVIDER_LIVE_0028`, `_0031`,
    `_0031B`, `_0032`, `_0032_HF_MODEL`, `_0033`, `_0033_REFRESH`, `_0034`,
    `_0036`, `_0037`, `_0039`, `_0040`, `_0040_FOLLOWUP`, `_0040_PROBE` and
    `_0041`.

  Each switch is named, as a whole word, in a record of its own number
  (`grep -w`). The claims C14–C16 stand. The question put to the owner
  counted 16 switches; the list holds 15.
* **Decided** by the owner: name the family.
  * `AGENTS.md` lists the suites by name, the three `GEMINI_429` variables
    among them, and states the rule: a record's own measurement test is
    switched on by `LLMPROVIDER_LIVE_<NNNN>`, with suffixes for its
    options, as that record says.
  * `scripts/check_tables.py` accepts a variable of that form only when a
    record numbered `NNNN` names it as a whole word, and fails on any other
    variable `AGENTS.md` does not name.
* **Rejected:** listing all 24 in `AGENTS.md`, a list that every new
  measurement record would have to edit.

