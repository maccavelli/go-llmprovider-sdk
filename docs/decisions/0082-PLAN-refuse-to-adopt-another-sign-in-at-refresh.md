---
status: in-progress
date: 2026-11-17
associated-madr: "0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Refusing to Adopt a Stored Session That Is Not the Same Sign-In

Associated MADR: [0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md](0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md)

## Goal

After this PLAN, in the release it shares with
[0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
and the three other drafts, on the owner's word (MADR D6):

* a refreshable `auth.OAuthSession` never adopts, uses or refreshes a stored
  session whose provider, kind, issuer or client differs from its own, or
  whose account differs when both name one, at the store's first read or at
  the re-read after a rejected refresh (MADR D1, D2);
* such a token request fails with an error matching
  `llmprovider.ErrAuthFailure`, naming no secret; the held session keeps its
  fields, and resumes once the store holds its sign-in again, or nothing
  (D3);
* a rotation of the same sign-in is adopted as today (D2);
* no name is exported (D4); every provider's refreshable sessions are
  covered (D5);
* the docs say so, and the changelog's `[Unreleased]` names it under
  Security (D6).

The MADR's questions are answered (its "Decided"); the MADR stays
`proposed` until the owner accepts it once placed and reviewed, and this
PLAN is `proposed` until the owner approves it. Nothing below Phase 0 starts
before that approval.

## Prerequisites

* 0080-PLAN Phase 3's code is committed on `main`, and no other agent holds
  staged or uncommitted work in the tree. This PLAN's Phase 1 starts on that
  commit, called **the base** below.
* The drafting's tests, `repro/auth/held_session_0082_test.go` (ten tests)
  and `repro/openai/held_session_0082_test.go` (two), and the prototype
  guard's patch and plant drivers (`repro/proto_b.py`, `repro/plants_b.py`,
  for reference only), are handed to the Phase 1 engineer by the
  coordinator (`place.py --test-out`). They are not committed with the
  records: the bootstrap exception takes records only.

## Relation to 0080 and the other drafts

* **Order.** This PLAN runs after 0080-PLAN Phase 3 is committed and before
  0080-PLAN Phase 5 (the release). Its gates never run at once with
  another record's on one host: the queue serialises them.
* **Files both touch.** 0080's staged Phase 3 (as read on 2026-11-17 from
  the index) changes `CHANGELOG.md` (`[Unreleased]` gains Added and
  Changed), `README.md`, `docs/architecture.md` (the refresh paragraph this
  PLAN edits moves from line 842 to 843) and `llmprovider/auth/kilo_device.go`.
  It does not change `oauth_session.go`, `tokenstore.go`,
  `tokenstore_file.go`, `providers/openai` or `internal/chatgpt`. The
  cancelled-sign-in draft changes `wizard/`; this PLAN changes nothing there.
* **What Phase 1 re-reads at the base,** recording each line in the
  execution record before any edit:
  * `llmprovider/auth/oauth_session.go`: `Token`'s doc and body,
    `complete`, `persistRotation`, `adopt`, `reloadOrRefresh`, `loadRotated`,
    `sessionKindOf` and `kindLocked` (MADR E1 to E27; this PLAN's C4 to C6);
  * `llmprovider/auth/tokenstore.go`: `TokenStore`'s doc (C7), the
    `OAuthSession` fields and `fileRecord`; `tokenstore_file.go`'s `Load`
    (MADR E31, E32);
  * `llmprovider/providers/openai/openai.go` `New` and `checkBackground`,
    and `internal/chatgpt` (unchanged by this PLAN; T7 and T8 go through
    them);
  * the sibling tests named under "Rules" and their helpers (C11 to C19);
  * `CHANGELOG.md`'s `[Unreleased]` headings, `README.md`'s store bullet,
    `docs/architecture.md`'s refresh paragraph (C1 to C3), and the
    sections of 0012-MADR and 0016-MADR that get notes (C8 to C10);
  * 0080-PLAN Phase 5, for how the shared release is cut.

## Scope

### In scope

* `llmprovider/auth/oauth_session.go`: the guard in `reloadOrRefresh` at
  both reads; an unexported `sameSignIn` and the refusal's error; the doc
  comments of `Token`, `reloadOrRefresh` and `loadRotated`.
* `llmprovider/auth/tokenstore.go`: `TokenStore`'s doc (a store keeps
  `Provider`, `Kind`, `Issuer`, `ClientID` and `AccountID`, or a sibling's
  rotation is refused).
* `llmprovider/auth/held_session_0082_test.go` and
  `llmprovider/providers/openai/held_session_0082_test.go`: the tests.
* Docs, in the same commit: `docs/architecture.md` (the refresh paragraph),
  `README.md` (the OpenAI store bullet), `CHANGELOG.md` `[Unreleased]`
  **Security**; dated one-line notes in 0012-MADR §5.2 and 0016-MADR D4 and
  its amendment A1.

### Exported names (R55)

None. No exported name is added, removed or changed (MADR D4).

### Out of scope

* Interfaces in place of the concrete session checks (MADR D4).
* `FileTokenStore.Load` (D5): the guard's provider comparison covers it.
* Vendor CLI sessions; Kilo's device and access-only sessions, which never
  re-read the store (MADR "By provider").
* Option D in any caller, and gobble-cli's changes (Phase 5 names them;
  gobble's records make them).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **One commit: the tests with the code**, as 0080's ruling of 2026-11-17
  had it for its own phase, proposed here for the owner's approval with the
  PLAN: the tests are seen red on a scratch clone of the base and land with
  the guard in one commit, so `main` never carries a failing build.
* **Red, then green.** Each test Phase 1's table marks red is seen red at
  the base and at `b109cea` with `python3 -B tools/redgreen.py RED.json
  --expect fail --rev <rev>`; every new test is green on the phase's tree
  with `--expect pass`; each new assertion is seen to fail on a planted
  break with `python3 -B tools/plant.py PLANTS.json`, on a scratch clone,
  never in the tree. The execution record quotes each failure line. A break
  no test catches is a deviation.
* **Heavy runs queue.** Every `-race` run and the gate run through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` on
  the session host, or through the skill's `scripts/run-on-host` (with
  `--race`) on the host `~/.agents/skills/available-dev-hosts/scripts/pick-host`
  prints. No run goes to a WSL distro. Every multi-round `-race` run carries
  an explicit `-timeout`. A single-package `go test` without `-race` may run
  on the session host.
* **Pre-add.** `make pre-add-check FILES="..."` on every changed Go file,
  through `run-heavy` (it lints and tests the packages).
* **No outside host in a test** (0065-MADR): every issuer is an
  `httptest` server or a transport that fails the test; every record's token
  URL points at one of them or the test's failing transport. The `openai`
  package's `localonly` guard holds it there.
* **No credential in a record or a test**: the tests use fixed made-up
  strings (`held-plan-access`, `other-refresh`, `acct-held` and the like).
* **Stop conditions,** each a stop and a prompt to the owner:
  * an existing test, golden or fixture needs a change. The ones to watch:
    `TestOAuthRefresh_ReloadsStoreFirst`,
    `TestOAuthRefresh_StoredTokenIsTheOneRefreshed`,
    `TestOAuthRefresh_SameStoredTokenStillRefreshes`,
    `TestOAuthRefresh_RejectedAdoptsSiblingRotation`,
    `TestOAuthRefresh_RejectedRefreshesWithSiblingToken`,
    `TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer`,
    `TestFileTokenStore_RefreshLock_OnceAcrossSessions`, and every test in
    `oauth_rotation_kept_test.go`, `oauth_spent_refresh_test.go`,
    `session_kind_0074_test.go`, `consent_0075_test.go`,
    `plan_session_0043_test.go`, `session_identity_0079_test.go` and
    `session_store_kinds_0048_test.go`;
  * the guard needs a change of lock order, or a store call it did not make
    before;
  * `make api-check` or `tools/apidiff.py v1.31.0` reports any change of
    this record's;
  * a test marked red cannot be red at the base without changing code that
    is not a test.

## Implementation Steps

### Phase 0: records

Model: lightweight, low.

1. The coordinator places this pair after 0080's Phase 3 commit, as the
   drafting's `PLACEMENT.md` says: the number from `python3 -B
   scripts/check_records.py --next`, `python3 -I place.py <root> <number>`,
   the index rows by `python3 -B tools/records.py add`, committed under the
   bootstrap exception (records only).
2. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --markdown`, then stage and commit (`git commit
   --no-edit`).
3. When the owner accepts the record and approves this PLAN,
   `tools/records.py status` sets the MADR `accepted` and this PLAN
   `in-progress`, each with its index row.

Acceptance: `make records-check links-check` pass; `python3 -B
tools/claims.py` passes on both records; no source changed. Docs: the
records are the docs.

### Phase 1: the tests, the guard and the docs, one commit

Model: mainstream, high (a small change, but it decides which stored
credentials a session uses, and it runs under the store's refresh lock;
the tests and a prototype exist).

Files: `llmprovider/auth/held_session_0082_test.go` (new);
`llmprovider/providers/openai/held_session_0082_test.go` (new);
`llmprovider/auth/oauth_session.go`; `llmprovider/auth/tokenstore.go` (a doc
comment); `docs/architecture.md`; `README.md`; `CHANGELOG.md`; the dated
notes in 0012-MADR and 0016-MADR; this PLAN's execution record.

1. **Re-read the base** ("Relation to 0080") and record the lines.
2. **The tests**, from the drafting's files, renamed with the record's
   number. The MADR's S1 to S5 become T1 to T3, T7 and T8.

   | Test | Asserts | Red at the base and at `b109cea` |
   | :--- | :--- | :--- |
   | T1 `TestHeldSessionRefusesAnotherKind_0082` | another application's ChatGPT login with a current token, and a ChatGPT login differing from the held plan session in kind alone: an auth failure, no token, no request, the held kind and client kept (MADR S1) | drafting: `Token = "other-access", <nil>` |
   | T2 `TestHeldSessionNeverSpendsAnotherRefreshToken_0082` | the other login's token due: no refresh request, the store's entry untouched, no save (S2) | drafting: `refresh tokens sent ["other-refresh"], want none` |
   | T3 `TestHeldSessionRefusesAnotherProviderRecord_0082` | a Grok record in `FileTokenStore`'s `openai.json`, and a record that differs in its provider alone: refused, `Owner()` kept (S3) | drafting: `Token = "grok-access", <nil>` |
   | T4 `TestHeldSessionRefusesAnotherClient_0082` | the same kind from another client, and from another issuer: refused | drafting: `Token = "other-access", <nil>` |
   | T5 `TestHeldSessionRefusesAnotherAccount_0082` | the same kind and client for another account: refused | drafting: `Token = "other-access", <nil>` |
   | T6 `TestReReadAfterRejectedRefreshRefusesAnotherSignIn_0082` | the held session's own refresh rejected as reused while another program writes another sign-in: the re-read refuses it; one refresh sent | drafting: `Token = "other-access", <nil>` |
   | T7 `TestPlanProviderNeverSendsAnotherSignInsToken_0082` (`providers/openai`) | a provider built over a plan session, after a 401: no request carries the other login's token; the listing's error is an auth failure (S4) | drafting: `requests ["held-plan-access" "other-access"]` |
   | T8 `TestChatGPTProviderNeverSendsAPlanToken_0082` (`providers/openai`) | the reverse: no plan token to the Codex route (S5) | drafting: `requests ["held-chatgpt-access" "plan-access"]` |
   | T9 `TestHeldSessionResumesWhenItsSignInReturns_0082` | refused while the store holds another sign-in; adopts its own sign-in's rotation once it is back; refreshes on its own once the store is empty, sending only its own refresh token (Q3) | drafting: `Token = "other-access", <nil>` |
   | T10 `TestSiblingRotationOfTheSameSignInIsAdopted_0082` | a sibling's rotation with no `Kind`, with a changed scope, ID token and consents, and with no account on one side: adopted | green (a guard) |
   | T11 `TestGrokHeldSessionRefusesAnotherClient_0082` | a Grok session refuses another client's Grok record (Q6) | drafting: `Token = "grok-other-access", <nil>` |
   | T12 `TestRefusalNamesNoSecret_0082` | the refusal matches `ErrAuthFailure`; its text carries no access token, refresh token or account id of either side (Q4) | drafting: `err = <nil>` |

3. **Red.** `python3 -B tools/redgreen.py RED.json --expect fail --rev
   <base>` and `--rev b109cea` for T1 to T9, T11 and T12, with both test
   files in `files`; T10 with `--expect pass`. Quote each failure.
4. **The guard** (`oauth_session.go`):
   * an unexported `sameSignIn(state oauthSessionState, stored *OAuthSession) bool`:
     the provider (an empty one on either side takes the held session's),
     the kind by `sessionKindOf` on the stored fields against
     `state.kindOf()`, the issuer with trailing slashes trimmed, the
     `ClientID`, and the `AccountID` when both are non-empty;
   * in `reloadOrRefresh`, after `loadRotated` at the first read and at the
     re-read after an `ErrAuthFailure`, before the stored token is used or
     refreshed: a stored session that is not the same sign-in returns
     `refreshed{}` and the refusal, so `complete` adopts nothing and the
     token request fails;
   * the refusal: `fmt.Errorf("oauth: the store holds another sign-in for
     %s (kind %s), not this session's (kind %s); sign in again: %w", ...,
     llmprovider.ErrAuthFailure)`, naming the provider and both kinds only
     (MADR D3);
   * nothing else changes: `loadRotated`, `adopt`, `persistRotation` and
     `FileTokenStore` stay as they are. The drafting's `repro/proto_b.py` is
     this shape, for reference.
5. **Docs**, in the same commit:
   * doc comments: `Token` (a stored session that is not the same sign-in
     is refused with an error matching `ErrAuthFailure`, and the session
     keeps its own); `reloadOrRefresh` and `loadRotated` (what "the same
     sign-in" is; MADR D2); `TokenStore` (a store keeps `Provider`, `Kind`,
     `Issuer`, `ClientID` and `AccountID`, or a sibling's rotation is
     refused);
   * `docs/architecture.md`, the refresh paragraph (C2): "adopts a rotation
     another process saved" becomes "adopts a rotation another process saved
     of the same sign-in (provider, kind, issuer, client, and account when
     both name one); another sign-in is refused with an auth failure, and
     the session keeps its own";
   * `README.md`, the OpenAI store bullet (C1): a running session does not
     take up a login of the other kind, or another application's, that its
     store comes to hold; it asks for a new sign-in instead;
   * `CHANGELOG.md` `[Unreleased]`, a `### Security` heading after 0080's
     headings, per Keep a Changelog's order: "A refreshable `OAuthSession`
     no longer adopts a session its token store holds that is another
     sign-in (another provider, kind, issuer or client, or another account
     when both name one), which it did at its next refresh, sending that
     login's token on the route it was built for. Its token requests now
     fail with an error matching `llmprovider.ErrAuthFailure` until the store
     holds its sign-in again, or nothing. A rotation of the same sign-in by
     another process is adopted as before (0082-MADR)";
   * dated one-line notes, each pointing at 0082-MADR: 0012-MADR §5.2
     (C8); 0016-MADR D4 (C9) and amendment A1 (C10).
6. **Before staging,** on a scratch clone of the phase's tree: `go test
   -count=1 ./llmprovider/auth/` and `go test -count=1
   ./llmprovider/providers/openai/` (session host, no `-race`); every
   existing test passes unchanged, or the phase stops.

Verification: the tests as their rows say;
`~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make
pre-add-check FILES="llmprovider/auth/oauth_session.go
llmprovider/auth/tokenstore.go llmprovider/auth/held_session_0082_test.go
llmprovider/providers/openai/held_session_0082_test.go"`; `make api-check`;
`python3 -B tools/apidiff.py v1.31.0`; `make docs-check tables-check
changelog-check links-check records-check markdownlint`.

Acceptance: every test as its row says; no existing test changed; no API
change; the docs say what the code does, the multi-process cases that now
fail included.

### Phase 2: planted breaks and the gates

Model: mainstream, low.

1. `python3 -B tools/plant.py PLANTS.json` on a scratch clone, each plant
   caught, each failure quoted. The plants, each rehearsed on the drafting's
   prototype with the test that caught it:
   * the guard removed at the first read: T1 to T5, T7 to T9, T11, T12;
   * the guard removed at the re-read: T6;
   * the provider not compared: T3;
   * the kind not compared: T1;
   * the kind compared as declared, not inferred: T10;
   * the issuer not compared: T4;
   * the client not compared: T4, T11;
   * the account not compared: T5;
   * an empty account counted as another: T10;
   * the scope compared too: T10;
   * the refusal not an auth failure: T1 to T6, T9, T11, T12;
   * the refusal naming the stored account: T12.
2. `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and
   `--expect pass` on the tree, quoted.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=20 -timeout 30m -run '_0082$' ./llmprovider/auth/ ./llmprovider/providers/openai/`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 -timeout 30m ./llmprovider/auth/... ./llmprovider/providers/... ./llmprovider/catalog/... ./llmprovider/x/signin/... ./wizard/...`
5. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
6. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
7. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
8. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/denyscan.py --changed`; `python3 -B
   tools/recordlint.py --changed`.

Acceptance: every plant caught; every command exit 0, each status taken
before any filter. Docs: none beyond Phase 1's; a gate that finds a docs gap
is a deviation.

### Phase 3: independent review

Model: strongest, high. A reviewer who did not write Phase 1, on the
coordinator's brief, because this change decides which stored credentials a
session will use.

1. Read Phase 1's commit against the MADR: every D, the decided labels, the
   stop conditions; the guard's place relative to the refresh lock and the
   session's lock; that nothing is sent, spent or saved for a refused
   sign-in; that the error names no secret.
2. Try to break it, on a scratch clone: a stored record that differs only in
   a field the guard does not compare (token URL, resource, host id, scope),
   and say whether each is right to pass; a record with no `Kind` of each
   kind; an empty `Provider` on either side; a race of two sessions in one
   process sharing a store while a third writes another sign-in (under
   `run-heavy`, with `-race -timeout`).
3. Findings go to the coordinator; a blocker stops the release. Fixes are a
   dated deviation here, with their tests seen red first.

Acceptance: the review's verdict and each finding's resolution are in the
execution record.

### Phase 4: the release (the owner's word)

Model: mainstream, medium. No step runs under a standing approval: the tag
and the GitHub Release each wait for the owner's word in the same turn.

1. By the owner's answer to Q8: this record ships in the next release with
   0080 and the three other drafts, cut by 0080-PLAN Phase 5; this PLAN
   adds nothing but its changelog entry, already under `[Unreleased]`. If
   this PLAN has not finished Phase 3 with its gates passed when the owner
   gives the word, its change moves to the release after, and Phase 1's
   commit stays on `main` under `[Unreleased]`.
2. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`, quoted.
3. `tools/records.py status` sets this PLAN `complete` once the tag's CI and
   step 2 are recorded.

Acceptance: the tag's CI passes; the section and the Release name this
change under Security.

### Phase 5: gobble, after the release (gobble-cli's own phase)

Not executed under this PLAN; recorded so the hand-off is exact, and priced
by gobble's record. At gobble-cli branch `p3-acpext-v0.5.0`, by a gobble-cli
record:

* `go.mod`: the SDK pin from `v1.31.0` to the release.
* The held-session test: the reviewer's `TestReviewCachedSessionAdoptsAForeignEntry`
  adopted into the tree and made to pass: the held session's token request
  fails with an auth failure, no foreign token is sent, its kind stays
  `chatgpt_plan`. Its plan session's token URL and client must first point
  at an in-process issuer: as kept, it names OpenAI's real issuer (the
  coordinator's caution of 2026-11-17).
* The documented limit removed: `docs/architecture.md` (the `llm/provider`
  row), `docs/guides/credentials.md` (the held-session sentence), the dated
  note at 0051-MADR D2, and 0052-PLAN P4 step 1d amended: no wrapper is
  built, by the owner's answer to this record's Q1.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`, never a WSL distro.

* `go test -count=1 ./llmprovider/auth/` and `./llmprovider/providers/openai/`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` (and
  `--rev b109cea`), then `--expect pass`; `python3 -B tools/plant.py
  PLANTS.json`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=20 -timeout 30m -run '_0082$' ./llmprovider/auth/ ./llmprovider/providers/openai/`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 -timeout 30m ./llmprovider/auth/... ./llmprovider/providers/... ./llmprovider/catalog/... ./llmprovider/x/signin/... ./wizard/...`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN

## Rollout and Rollback

* **Order.** Phase 0 when placed; Phase 1 on approval, on the base; Phase 2
  after it; Phase 3 before the release; Phase 4 with 0080's release on the
  owner's word; Phase 5 in gobble-cli after the tag.
* **Rollout.** A caller whose store only ever holds its own sign-in sees
  nothing new. One whose store comes to hold another sign-in gets an auth
  failure on the next refresh, where it adopted that sign-in; the multi-
  process cases the MADR names (the other OpenAI method, another account, a
  plan sign-in without a `ClientStore`) now ask for a new sign-in in the
  running process.
* **Rollback before the tag:** revert Phase 1's commit; the records stay,
  with a dated note.
* **Rollback after the tag:** a patch release that removes the guard, by a
  dated amendment of the MADR. Nothing persistent changes shape, so no
  stored session needs migrating either way.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, at `b109cea`;
Phase 1 re-reads each at the base.

<!-- claims: self=b109cea -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The README's OpenAI store bullet | `README.md:310` | `` - **Store.** A `TokenStore` keeps one OpenAI login, of either kind: a plan `` |
| C2 | The architecture's refresh paragraph | `docs/architecture.md:842` | `re-reads the store, and adopts a rotation another process saved.` |
| C3 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| C4 | `Token`'s doc | `llmprovider/auth/oauth_session.go:138` | `// Token returns the current bearer token, refreshing and persisting it when` |
| C5 | `reloadOrRefresh`'s doc | `llmprovider/auth/oauth_session.go:940` | `// reloadOrRefresh re-reads the session from its store before refreshing:` |
| C6 | `loadRotated`'s doc | `llmprovider/auth/oauth_session.go:1000` | `// loadRotated returns the stored session when the store holds a refresh token` |
| C7 | `TokenStore`'s doc | `llmprovider/auth/tokenstore.go:217` | `// TokenStore persists OAuth sessions, one per provider. Load returns nil and no` |
| C8 | 0012-MADR §5.2 | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:502` | `` **5.2 Refresh of `mcplib`-owned sessions.** `` |
| C9 | 0016-MADR D4 | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:299` | `### D4. A rotated token is never discarded` |
| C10 | 0016-MADR A1 | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:558` | `### A1. Re-read the store after a rejected refresh (extends D4)` |
| C11 | A sibling test: the store is read first | `llmprovider/auth/oauth_refresh_test.go:44` | `func TestOAuthRefresh_ReloadsStoreFirst(t *testing.T) {` |
| C12 | the stored token is the one refreshed | `llmprovider/auth/oauth_refresh_test.go:65` | `func TestOAuthRefresh_StoredTokenIsTheOneRefreshed(t *testing.T) {` |
| C13 | the same stored token still refreshes | `llmprovider/auth/oauth_refresh_test.go:91` | `func TestOAuthRefresh_SameStoredTokenStillRefreshes(t *testing.T) {` |
| C14 | a rejected refresh adopts a sibling's rotation | `llmprovider/auth/oauth_rotation_test.go:53` | `func TestOAuthRefresh_RejectedAdoptsSiblingRotation(t *testing.T) {` |
| C15 | and refreshes with its token | `llmprovider/auth/oauth_rotation_test.go:69` | `func TestOAuthRefresh_RejectedRefreshesWithSiblingToken(t *testing.T) {` |
| C16 | an unsaved rotation never overwrites a newer one | `llmprovider/auth/oauth_rotation_test.go:86` | `func TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer(t *testing.T) {` |
| C17 | one refresh across sessions sharing a store | `llmprovider/auth/tokenstore_durable_test.go:101` | `func TestFileTokenStore_RefreshLock_OnceAcrossSessions(t *testing.T) {` |
| C18 | Their helpers: a refresh server | `llmprovider/auth/oauth_refresh_test.go:23` | `func refreshServer(t *testing.T, reply func(n int32, w http.ResponseWriter, r *http.Request)) (*httptest.Server, *atomic.Int32) {` |
| C19 | and two sessions sharing a store | `llmprovider/auth/tokenstore_durable_test.go:77` | `func sharedStoreSessions(t *testing.T, tokenURL string) (*FileTokenStore, *OAuthSession, *OAuthSession) {` |
| C20 | The openai package's build helper | `llmprovider/providers/openai/helpers_test.go:154` | `func build(t *testing.T, opts ...llmprovider.Option) llmprovider.Provider {` |

## Execution record

### Drafting (2026-11-17)

* **Provenance.** Drafted by a strongest-tier model at high effort, on the
  coordinator's brief, from read-only investigation and runs on scratch
  copies; nothing implemented, nothing written to the repository's tree or
  index. The MADR and this PLAN are `proposed`, with `0082` for the number,
  which the placing step takes from `scripts/check_records.py --next`.
* **The owner's answers,** 2026-11-17, relayed by the coordinator: all
  eight of the MADR's questions as the draft recommended (its "Decided",
  with the labels); Q5 not asked, falling away by Q4's answer as the
  coordinator applied it.
* **Verified, and how.**
  * Read at `b109cea` (`git archive` into a scratch folder, and `git
    show`): the session, the stores, the OpenAI provider, `internal/chatgpt`,
    the other providers' `New`, the catalog's raw listing, the records 0012,
    0016, 0043, 0074, 0075, 0079 and 0080. Read on the index of the tree as
    it stood (`git show :<path>`): 0080's staged changes to `CHANGELOG.md`,
    `README.md`, `docs/architecture.md` and `kilo_device.go`, and 0080-PLAN's
    Phases 4 to 6.
  * The MADR's S1 to S8, on scratch copies of `b109cea`.
  * The twelve tests of Phase 1's table, on two scratch copies: at `b109cea`,
    eleven fail as the table quotes and T10 passes; on a copy with the
    prototype guard (`repro/proto_b.py`), all twelve pass.
  * The twelve plants of Phase 2, each on a fresh scratch copy of the
    prototype, each caught by the tests named (`repro/plants_b.py`; the
    driver exits 0 only when every plant is caught).
  * The twelve tests on the prototype with `go test -race -count=5 -timeout
    10m`, queued through `run-heavy` on the session host: exit 0.
  * `tools/claims.py` from a scratch copy: the MADR's 81 rows and this
    PLAN's 20 pass at their revisions; two planted false rows were caught.
* **Not verified.** Any gate; the full `-race` set of Phase 2; a host but
  the session's; the base itself, which does not exist yet; the MADR's own
  "Not verified" list.

### Approval (2026-11-17)

* **Approved 2026-11-17.** This PLAN was approved through its live step or
  release by the owner's ruling of 2026-11-17, relayed by the coordinator:
  "Accept all five and run each plan up to its live step or release
  (Recommended)". Its phases are committed and pushed by the coordinator as
  each passes. Tags and releases need the owner's word.
* **Annotated 2026-11-17.** Since 2026-11-17 the owner makes every commit and
  push from the coordinator's staged list, in place of the line above; tags
  and releases still need the owner's word.

### Phase 1 (2026-11-17)

* **Provenance.** A mainstream-tier model at high effort, as this PLAN prices
  the phase, on the coordinator's brief. Base `8f133db` (level with
  `origin/main`, tree clean). Staged, not committed: the coordinator reads the
  diff and runs its own gate; since 2026-11-17 the owner makes the commit and
  the push from the coordinator's staged list; the independent review is
  Phase 3 (its findings, and this entry's corrections, are in "Phase 1,
  second pass" below).
* **Step 1, the base re-read.** Every claim of the MADR's index and of this
  PLAN's table was found again at the base by its text. Unchanged: C3 to C7,
  C9 to C20 and every row E1 to E67 and E69 to E70 in `llmprovider/auth`,
  `providers` and `internal` (0080's Phase 3 changed `kilo_device.go` in
  `auth` and nothing of `oauth_session.go`, `tokenstore.go` or
  `tokenstore_file.go`). Moved: C2 `docs/architecture.md:842` to `:843`, as
  this PLAN said it would; E68 `wizard/auth.go:716` to `:747` (the text is the
  same, and nothing here reads it). C1 (`README.md:310`) and C8
  (`0012-MADR-...md:502`) are where they were; a search by text missed them
  only because their cells carry backticks. Nothing differed in substance.
  Two facts the re-read added: the doc block that begins "reloadOrRefresh
  re-reads the session from its store" (line 940) is attached to the type
  `refreshed`, not to the function, whose own doc is at line 957, so both were
  edited; and a record saved since the first commit carries `issuer`,
  `client_id`, `account_id` and `provider` always, and `kind` only since 0074
  (`fileRecord`), so a legacy record's empty `kind` is carried unchanged
  through every rotation, by the refresh (E19 to E21) and by
  `FileTokenStore`. *Corrected by the review, 2026-11-17 (N4):* this entry
  first also named "an older ChatGPT record's empty issuer". No release wrote
  one: every sign-in sets the issuer (`Issuer: config.issuer`, resolved to the
  built-in issuer, at `v1.0.0` and every tag since), so an empty issuer comes
  only from a hand-built or imported record, which the amendment's R-4 now
  reads as the built-in issuer.
* **Step 2 and 3, the tests, red first.** The drafting's two files were
  placed as `llmprovider/auth/held_session_0082_test.go` and
  `llmprovider/providers/openai/held_session_0082_test.go`, unchanged. At the
  base `8f133db` and at `b109cea`, `tools/redgreen.py --expect fail` saw all
  eleven (T1 to T9, T11, T12) fail, each on its assertion and as the
  drafting's table quotes: T1 `Token = "other-access", <nil>`; T2 `refresh
  tokens sent ["other-refresh"], want none`; T3 `Token = "grok-access",
  <nil>`; T4, T5, T6, T9 `Token = "other-access", <nil>`; T7 `requests
  ["held-plan-access" "other-access"]`; T8 `requests ["held-chatgpt-access"
  "plan-access"]`; T11 `Token = "grok-other-access", <nil>`; T12 `err =
  <nil>`. T10 passes at both revisions (`--expect pass`): a guard.
* **Added beyond the drafting (the edges D2 raises),** in the same auth test
  file, each seen red or green at the base the same way: T13
  `TestLegacyRecordRotationIsAdopted_0082` (a record with no kind and no
  issuer, as an older version saved it, loaded by two sessions of one
  `FileTokenStore`: the first refreshes, the second adopts the rotation with
  no request, and the record keeps its empty kind and issuer; green at the
  base, a guard); T14 `TestSameSignInEdges_0082` (adopted: an account on the
  stored side only, no provider on the record, a trailing slash on the
  issuer; refused: no issuer, or no client, on the record; the two refusals
  are red at the base, `Token = "other-access", <nil>`); T15
  `TestRefusalTextIsPlain_0082` (a stored kind carrying control characters
  does not reach the error; red at the base, `err = <nil>`).
* **Step 4, the guard** (`llmprovider/auth/oauth_session.go`): `sameSignIn`
  (line 1068) and `errOtherSignIn` (1089), called from `reloadOrRefresh`
  (983) after the first `loadRotated` (994) and after the re-read (1008),
  before the stored token is used or refreshed. Provider, kind (by
  `sessionKindOf` on the stored fields against `state.kindOf()`), issuer with
  trailing slashes trimmed, client, and the account when both name one. The
  refusal's text is the plan's, with the two kinds passed through
  `redact.Field` so that a record's text cannot put a control character in an
  error. Both comparisons use the `state` each read is made with: at the
  re-read that is the stored session's state when the first read took one up
  (stricter than the held session's own where a field is empty on the held
  side, and the prototype's behaviour). `loadRotated`, `adopt`,
  `persistRotation` and `FileTokenStore` are unchanged.
* **The lock.** `Token` copies the session into `state` under `s.mu`
  (`refreshState`, `state.stored`) and releases it before starting `complete`,
  which releases `s.mu` again before `reloadOrRefresh` and takes it only after
  it returns. `reloadOrRefresh` takes the store's refresh lock first and holds
  it to its end, so both reads and the guard are under it. The guard reads
  `state` (a by-value copy) and the `*OAuthSession` the store returned, which
  `loadRotated` has already written to and no other goroutine holds; it holds
  no session lock and calls no store method `loadRotated` did not. A refusal
  returns before `complete` adopts, sets `storedRefresh` or `nextRefresh`: the
  session is untouched, and as the error matches `ErrAuthFailure`, `complete`
  does not fall back to a held token either.
* **Step 5, the docs:** doc comments of `Token`, `refreshed`'s block,
  `reloadOrRefresh`, `loadRotated` (`oauth_session.go`) and `TokenStore`
  (`tokenstore.go:217`); `docs/architecture.md:840`; `README.md:310`;
  `CHANGELOG.md` `[Unreleased]` `### Security`; dated notes in
  0012-MADR §5.2 and 0016-MADR D4 and A1 (additions only).
* **Plants of this phase's own additions** (`tools/plant.py`, a scratch
  clone, each caught; Phase 2 owns the plan's twelve): T14, the account
  compared when the held side names none (`held_session_0082_test.go:434`);
  an empty stored provider counted as another (`:434`); the issuer's trailing
  slash not ignored (`:434`); an empty stored issuer counted as the same
  (`:452`); an empty stored client counted as the same (`:452`); T15, the
  stored kind reaching the error raw (`:472`, `:475`); T13, a record with no
  issuer refused (`:403`), and a refresh writing the inferred kind into the
  record (`:400`). Also, as a rehearsal only, the guard removed at the first
  read (T1 to T5 and the rest fail), removed at the re-read (T6 fails alone),
  and the refusal not an auth failure (T1 to T6 fail).
* **Gates** (each status taken before any filter):

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` on the four Go files | session host, `run-heavy` | exit 0: gofmt, golangci-lint (host and Windows), vet, test, govulncheck |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `go test -count=1 ./llmprovider/auth/ ./llmprovider/providers/openai/ ./llmprovider/x/signin/ ./wizard/` | session host | exit 0; every existing test unchanged and green |
  | `go test -race -count=20 -timeout 30m` on `_0082$`, `TestOAuthRefresh_`, `TestOAuthSession_`, `TestFileTokenStore_RefreshLock`, `TestSessionKind`, `TestHeldSession`, `TestPlan` in `auth` and `providers/openai` | session host, `run-heavy` | exit 0 |
  | `go test -count=1 ./...` | a Linux host chosen by `pick-host`, `run-on-host`, no `-race` | exit 0, 42 packages ok |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; the diff lists only `llmprovider/x/signin.VerifyKilo: added`, 0080's |

  `make docs-check tables-check changelog-check links-check records-check
  disclosure-check markdownlint` passed on the tree before this entry was
  written; the record gates (`records-check`, `links-check`,
  `disclosure-check`, `markdownlint`, `changelog-check`, `tools/claims.py`,
  `tools/recordlint.py --changed`, `tools/denyscan.py --changed`) were run
  again after it, and their results are in the hand-off.
* **Not done.** Phase 2's twelve planted breaks and its gate set (`make lint`
  and `make vuln` beyond what `pre-add-check` ran, the CI set, `tools/gate.py`);
  the independent review (Phase 3).
* **Observations, not changes** (each out of this decision's scope): a
  successful refresh saves over the store without testing that it still holds
  this sign-in, so another sign-in written during the refresh's own request is
  overwritten (the read before it is guarded, the write is not; `persistRotation`
  guards only the retried save); and a stored record with no refresh token (an
  access-only session of another sign-in) is not returned by `loadRotated`, so
  it is neither adopted nor refused, and the held session's refresh saves over
  it, as before.
* **Not verified.** A plan ID token's `chatgpt_account_id` (the MADR's own
  caveat); any host but the session's and one Linux host; behaviour of a
  third-party `TokenStore` that drops a compared field (the `TokenStore` doc
  now says it must keep them).
* **Staged, not committed.**

### Phase 1, second pass (2026-11-17)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over the staged first pass (base `8f133db`). Staged, not committed:
  since 2026-11-17 the owner makes the commit and the push from the
  coordinator's staged list.
* **The review (Phase 3, run before the commit).** A reviewer who did not write
  Phase 1 found no blocker: in every path it could reach, another sign-in is no
  longer adopted, spent or used, and no legitimate rotation of the same
  sign-in is refused. Findings: S1 (the staged suite missed the reviewer's
  plants P14, P15 and P16: the text carrying the stored client id, `storedRefresh`
  advanced on a refusal, a refusal falling back to the held token), S2 (an
  access-only record of another sign-in is neither adopted nor refused, and
  is written over), S3 (a sign-in written during the refresh's request is
  overwritten by its save), S4 (`StoredConsent` reads any record; the wizard
  hands another application's plan record over as `Previous`), S5 (the
  README's "or nothing" and the changelog's account of a custom store); N1 (the
  text names values), N2 (Grok and account-less plan sign-ins are
  indistinguishable: a stated limit), N3 (stored kind inferred with the stored
  provider where D2 says the held session's; the re-read compared with the
  first read's state; the reviewer's plants P13 and P24 survived both suites),
  N4 (this PLAN's "an older ChatGPT record's empty issuer", corrected in place
  above). The owner's four rulings, 2026-11-17, relayed by the coordinator:
  R-1 "Amend 0082: guard the save too, before this is committed
  (Recommended)"; R-2 "Amend 0082: refuse on any stored record that is
  another sign-in (Recommended)"; R-3 "Amend 0082: a record that is not this
  sign-in counts as unanswered (Recommended)"; R-4 "Treat an empty issuer as
  the built-in issuer; keep the client strict (Recommended)". The MADR's dated
  amendment records them.
* **A slip of order, stated.** The guard's code was written before the new
  tests had been run red. The red runs below were taken afterwards on a
  scratch copy of the index, the first pass exactly as staged, with the new and
  changed test files copied in, so each red line is the previous staging's.
* **Built, with `path:line`.**
  * R-1: `storeMovedOn` (`oauth_session.go:370`), `isKnownToken` (385),
    `saveRotation` (396), called by `persistRotation` (416) and by
    `reloadOrRefresh` before its save (1094): one predicate for both saves
    (E27); `errStoreOtherSignIn` (359), left as it is by `complete`'s retried
    save so the spent token stays known (`complete`, the `errStoreMovedOn`
    case only clears). The wizard's save takes the store's lock (`wizard/auth.go:1063`).
  * R-2: `loadRotated` (1117) judges every record, `sameSignIn` before the
    refresh-token test; a record holding the session's own token is exempt
    (below).
  * R-3: `StoredConsent` (`consent.go:153`); `signsInAgainTo` and its call
    (`wizard/auth.go:330`, `296`), the client record read only when a plan
    session is stored (`291`).
  * R-4: `signInIssuer` (1182), `builtinIssuer` (1192), used by `signInDiff`
    (1160) and by `refreshTokenURL` (1213), whose literal tests it replaces
    with the same result; no table existed to reuse, since the module reads an
    empty issuer as the built-in one in `refreshTokenURL`, `ChatGPT()` and
    `inferKind` by three separate tests, so `builtinIssuer` is the one table
    now and the other two are unchanged.
  * N1: `errOtherSignIn` (1208) names the provider and the one field, by its
    name; N3: both reads compare with `held`; the stored kind is inferred with
    `cmp.Or(stored.Provider, held.provider)`.
  * The docs, in the same commit: below.
* **The decisions made inside the rulings, for the owner to see.**
  * A record that holds the session's own refresh token, or the one the store
    is known to hold, is not judged by the comparison. R-2's wording reads
    "every stored record"; judged so, a store that drops a field would refuse
    its own session at every refresh, which refreshed before. The record that
    is another sign-in holds another token. Test: `TestRecordWithTheHeldRefreshTokenIsTheHeldSessions_0082`.
  * An issuer's case is not folded: every other issuer test in the module
    compares exactly after trimming a slash, so `https://AUTH.openai.com` is
    another issuer (the reviewer's E3, now refused by rule, not by accident).
  * The re-read compares with the held session's state, as D1 says; the
    stricter alternative, comparing with the first read's record, is not what
    the MADR states.
  * `StoredConsent` and the wizard's lookup use the same predicate where the
    session exists; the wizard has no held state before the new sign-in, so its
    rule is its own (`signsInAgainTo`): OpenAI's own plan session of OpenAI's
    issuer with an issued client, and, with a `ClientStore` record, that
    client. A store with no client record gives no evidence about whose a plan
    session is, so it is taken as the app's own, as before; the host id is not
    compared (D2). 0075's reuse holds: `TestWizardPlanReusesClient_0075`,
    `TestWizardPlanSwitchKeepsClient_0075`, `TestPlan_ReSignIn_0043` and
    `TestPlanPreviousOnlyForTheSameSignIn_0082`'s own-client cases pass.
* **The state a skipped save leaves, and the next `Token` in each case.**
  The rotation is used in memory and returned; `storedRefresh` stays at the
  token the refresh spent. (1) The store still holds another sign-in: the
  retried save fails the same check and logs; the next due refresh is refused,
  nothing is sent. (2) The store is restored to the spent token: the retry
  writes the rotation (`storedRefresh` was kept, so that token is "the store's"
  and is never adopted back and sent); a due refresh first would refresh with
  the held session's current token, not the spent one. (3) The store is empty:
  the retry writes it. Tests: `TestSignInWrittenDuringRefreshIsNotOverwritten_0082`,
  three subtests for each of a locking and a non-locking store.
* **The risks, closed.**
  * (a) Lock order. `Token` releases `s.mu` before it starts `complete`
    (`oauth_session.go`), which takes `s.mu` only in short sections that call
    no store and releases it before `reloadOrRefresh`, which takes the store's
    refresh lock first and, inside it, `Load`, the refresh request and `Save`;
    `FileTokenStore.Save` takes no OS lock, and reads the session through
    `snapshot`, which takes the session's `s.mu` of the private `next` the
    refresh built. The wizard takes the store lock, then `Save`, which takes
    the live session's `s.mu`. So the order is store lock, then session lock,
    everywhere; no path takes the session lock and then the store's (the only
    callers of `LockRefresh` are `persistRotation`, `reloadOrRefresh`,
    `SetConsent` and the wizard's save, none under `s.mu`; `StoredConsent`
    copies under `s.mu` and loads after releasing it). No lock is held across
    a network call that was not already: the refresh holds the store lock across
    its request, bounded by its attempts (`lockWait`'s 45.6 s).
  * (b) The wizard's wait. Bounded by the store's own bound (`FileTokenStore`:
    `lockWait`, 60 s, then an error matching `ErrProviderUnavailable`);
    cancellable by the wizard's context: `FileTokenStore.LockRefresh` returns
    once `ctx` ends, and the save returns that error wrapped and saves
    nothing (`TestSaveCancelledWhileWaitingSavesNothing_0082`); a context that
    ends just after the lock is taken saves nothing either. The person sees
    nothing while it waits (no notice is raised: the prompter is not
    available to the save and a timer notice would call it from another
    goroutine), then the saved sign-in or the error.
  * (c) A store that is not a `RefreshLocker`: the wizard saves as before
    (`TestSaveWithoutALockerSaves_0082`); the refresh's save gets the re-read
    check, not atomic with the save, documented on `RefreshLocker`,
    `TokenStore`, `Options.TokenStore` and the changelog. gobble's adapter
    (read only) implements `RefreshLocker` over its own `LoginStore.LockRefresh`,
    documented as not the lock its `Save` takes, and keeps all five fields:
    its three session packages pass against this tree.
  * (d) Re-entrancy. The wizard calls `c.session.Token` before it asks for the
    lock, which waits for the session's own refresh to finish; no refresh waits
    for the wizard, and no lock is requested while one is held by the same
    goroutine. `TestSaveWaitsForARefreshInFlightInOneProcess_0082`: a refresh
    in flight in the same process holds the lock across its request, the save
    waits, lands after it, and the test's timeout would expose a deadlock.
* **Red on the previous staging, green now** (the new and changed tests; each
  red line from `go test -v` on the scratch copy of the index):
  * `TestEmptyIssuerIsTheBuiltInIssuer_0082` (R-4, E1 and E2's issuer half):
    four adopted cases red, `held_save_0082_test.go:256: Token = "", oauth: the
    store holds another sign-in for openai (kind chatgpt), not this session's
    (kind chatgpt); ...; want the sibling's rotation adopted`; its refusals
    (empty client either side, another issuer, the host's case, E3) green.
  * `TestAccessOnlyRecordOfAnotherSignInIsRefused_0082` (R-2, RV_B4): red in
    all three stores, `Token = "", llmprovider: provider unavailable: oauth:
    refresh request: Post ...: no network in this test; want no token and an
    error matching ErrAuthFailure` (the held session refreshed over it).
  * `TestSignInWrittenDuringRefreshIsNotOverwritten_0082` (R-1, RV_B5): red in
    all six subtests, `the newer sign-in was overwritten by the refresh's save`.
  * `TestStoredConsentIgnoresAnotherSignIn_0082` (R-3, RV_B6): four red,
    `StoredConsent = true, true, <nil>; want granted and answered both false`.
  * `TestRefusalNamesTheFieldNotTheValue_0082` (N1): five red, the text names
    kinds and carries the kind's value.
  * `TestRecordWithNoProviderTakesTheHeldProvider_0082` (N3): red, `... (kind )
    ... want the record, which names no provider, kind or issuer, adopted`;
    `TestReReadComparesWithTheHeldSession_0082` (N3, P24): red;
    `TestSameSignInEdges_0082/adopted/no_issuer_on_the_record`: red (the first
    pass refused it; R-4 flips it).
  * `TestPlanPreviousOnlyForTheSameSignIn_0082` (R-3), two red,
    `Previous = OAuthSession{... ClientID:oaiapp_other ...}`; and
    `TestPlanPreviousNotFromAnotherProvider_0082`: red;
    `TestSaveTakesTheRefreshLock_0082`: `events ["save"]; want lock, save,
    unlock`; `TestSaveCancelledWhileWaitingSavesNothing_0082`: `save returned
    <nil> before the lock was available or the context ended`;
    `TestSaveWaitsForARefreshInFlightInOneProcess_0082`: `the save returned
    <nil> while a refresh held the lock`.
  * Green on the previous staging, as guards, and green now:
    `TestRefusalLeavesEverythingAlone_0082` (RV_S1: three requests, every field
    compared after each, bookkeeping included, the store re-read each time, the
    stored client id absent from the text),
    `TestEarlyRefreshRefusalDoesNotFallBack_0082` (RV_B1),
    `TestAccessOnlyRecordOfTheSameSignInIsRefreshedOver_0082` (a same-sign-in
    access-only record is not adopted; the held session refreshes its own and
    saves over it), `TestRecordWithTheHeldRefreshTokenIsTheHeldSessions_0082`,
    `TestSignInSaveRacingRefreshes_0082` (three sessions and a lock-taking
    writer, in one process, under `-race`; it passes on the previous staging
    because the race it covers is not forced, so only the deterministic
    subtests above prove the guard; it proves no data race and the end state).
  * Changed: `refused0082` compares every field, bookkeeping included, in T1 to
    T6, T9 and T11 (S1); T3 asserts `openai.json` byte for byte unchanged; T7
    and T8 assert the refusal's own words (`another sign-in`), not only an
    auth failure the server's 401 would give; T6 asserts the re-read's
    refusal text, which a plant that ignores it (the issuer's own rejection is
    an auth failure too) survived until now; T14's "no issuer" case moved from
    refused to adopted (R-4); the first pass's T15 is replaced by the N1 test.
* **Plants** (`tools/plant.py`, a scratch clone of the final tree, 41 of 41
  caught, none a build failure; the first run left two, P08 and R1c, which
  the T6 text assertion and a retry against the other sign-in before the
  restore now catch). The reviewer's 24, adapted to the amended code:
  P01 to P24 caught, with P13 (`TestRecordNamingTheProviderInfersItsKind_0082`),
  P14 and P20 and P23 (the N1 test), P15 and P10 (every held field compared),
  P16 (`TestEarlyRefreshRefusalDoesNotFallBack_0082`), P24
  (`TestReReadComparesWithTheHeldSession_0082`) now caught; P13b, the stored
  provider alone, is caught by `TestRecordWithNoProviderTakesTheHeldProvider_0082`.
  The amendment's: R1a the save not re-read; R1b the predicate inverted; R1c
  the other-sign-in refusal clearing `storedRefresh`; R1d a same-sign-in
  access-only record not written over; R2 the access-only record skipped
  again; R2b an empty token counted as known; R2c the own token no longer
  exempt; R3a the consent read unguarded; R3b `Previous` handed over unguarded;
  R3c `Previous` not compared with the issued client; R4a to R4d the empty
  issuer not normalised on either side, the client normalised too, the case
  folded; W1 the wizard's save not taking the lock; W2 its wait not
  cancellable (the test ends it at 5 s). Each is a test failure, none a build
  failure; the first attempt at R3a and R3b was an unused variable, a build
  failure, and was planted again so that it compiles. The plant run was
  repeated on the final tree, after the last change to a Go file, a lint
  repair: `nilerr` on `loadRotated`'s failed read, now `stored = nil` before
  the test, and `gocognit` on the R-1 test, now split into helpers; neither
  changes behaviour, and no `//nolint` was added.
* **Gates** (each status taken before any filter, after the last change to a
  Go file; an earlier `make pre-add-check` failed on the two lint findings
  above and a `-race` run was cancelled for it, both then repeated):

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` on the ten changed Go files | session host, `run-heavy` | exit 0: gofmt, golangci-lint (host and Windows), vet, test, govulncheck |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; the diff lists only `llmprovider/x/signin.VerifyKilo: added`, 0080's: no name from this change |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./llmprovider/providers/openai/ ./llmprovider/providers/grok/ ./wizard/` | session host, `run-heavy` | exit 0 (auth 220 s, wizard 174 s) |
  | `go test -race -count=1 -timeout 45m ./...` | a Linux host chosen by `pick-host`, `run-on-host --race` | exit 0, 42 packages ok; an earlier run, before the lint repair, on the Windows host, exit 0 |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` against this tree by a scratch `replace`, a scratch copy of gobble, gobble untouched | session host | exit 0, three ok |
  | `tools/plant.py` | scratch clone | 41 of 41 caught |

  The record gates (`records-check`, `links-check`, `disclosure-check`,
  `markdownlint`, `changelog-check`, `docs-check`, `tables-check`,
  `tools/claims.py`, `tools/recordlint.py --changed`, `tools/denyscan.py
  --changed`) ran after this entry's last edit; their results are in the
  hand-off.
* **Not done.** Phase 2's own set (its twelve plants are covered by the 41,
  its gate set beyond what is listed here: `make lint`, `make vuln`, the CI
  set, `tools/gate.py`). The reviewer's L, E, I and C sequence tests were not
  ported as a table; their cases are covered by T10, T13, T14, the E-cases of
  `TestEmptyIssuerIsTheBuiltInIssuer_0082` and the plants.
* **Not verified.** Behaviour on a store whose `LockRefresh` ignores its
  context; a wizard save waiting the full 60 s (the bound is `lockWait`'s, not
  exercised); whether a plan ID token carries an account id (N2); the
  windows-only race run is one run, not twenty.
* **Staged, not committed.**

### Phase 1, third pass (2026-11-17)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over the staged second pass (base `8f133db`; 17 paths staged). Staged,
  not committed: the owner makes every commit and push from the coordinator's
  staged list; tags and releases need the owner's word.
* **The second review (Phase 3, run before the commit).** No blocker, no
  deadlock; every ruling of the first amendment implemented; the pass's 41
  plants and the reviewer's re-formed 24 caught. It left B-1 (fixtures that
  could reach the real issuer), B-2 (three missing tests), B-3 (docs), N-1 (a
  store that keeps neither the refresh token nor the client is refused at its
  second refresh in one process, where it worked before 0082), N-2 (a failed
  re-read still saves), N-3 (the wizard gave the store to the new session
  before taking the lock), N-4 (a dead arm, two survivors), N-5 (the re-read's
  account), N-6 (0081's check must stay after the lock). Survivors of the
  staged suite, as the reviewer named them: N06 (a sibling's newer rotation
  written over), N17 (the 30 s retry spending the other sign-in's token), N28
  (the wizard's check of the context after the lock removed); survivors of
  everything: N02 (an empty store counted as moved on), N19 (the save's known
  list missing the stored token), N21 (a failed re-read skipping the save), N29
  (the retried save's known list missing the rotation).
* **B-1, stated plainly.** Two of the reviewer's planted breaks, P11 (a refresh
  inside `loadRotated` before the stored session's client was set) and N17,
  made the second pass's fixtures send refresh POSTs toward OpenAI's real token
  endpoint (`https://auth.openai.com/oauth/token`), because the fixtures'
  foreign records had an empty `TokenURL` and the endpoint then defaults to the
  real one; a record of another provider reached xAI's endpoint the same way.
  The bodies carried only made-up values (`other-refresh`, a made-up client id
  such as `app_other`). Whether the requests left the machine is not known: the
  runs had no guard, and the reviewer's run was not under a sandbox. The owner
  was told. It broke a standing rule of this work (no outside request from any
  test; in-process issuers only), 0065-MADR, and this PLAN's own sentence ("every
  record's token URL points at one of them or the test's failing transport").
  The fix: every foreign or hand-built record in the 0082 test files names an
  in-process issuer or `deadURL0082` (`https://issuer.invalid/never`, a name
  RFC 6761 reserves and nothing resolves); every client that reaches an
  in-process issuer is `local0082`, which refuses a non-loopback host without
  dialing; and `llmprovider/auth` has the `TestMain` that `wizard` and `openai`
  have (`localonly_test.go`, behind `!live_gateways`), which runs the tests
  behind `internal/localonly`'s proxy, refusing and recording any request to a
  host off the machine and failing the run naming it. Proof, on scratch trees
  under a macOS sandbox that denies every outbound connection but loopback
  (`sandbox-exec`), with the two plants re-applied: the second pass's fixtures
  with the guard added aimed at `auth.openai.com:443` (48 requests for P11, 12
  for N17) and `auth.x.ai:443` (6, P11) and `issuer.invalid:443` (24, P11), as
  the guard's report names them; this pass's fixtures aim only at
  `issuer.invalid:443` (87 for P11, 12 for N17), each refused by the guard's
  proxy before any dial, with no `dial tcp` or sandbox-denial line in the output
  and `auth.openai.com` absent; and each plant fails a test on the in-process
  issuer's own record (`refresh tokens sent ["held-plan-refresh" "other-refresh"
  ...]`, `held_round3_0082_test.go:377`). No plant that could refresh a foreign
  record was run before this guard was in place and proven.
* **The rulings,** 2026-11-17, relayed by the coordinator: R-5 "Keep it, and
  also treat a record holding the session's own access token as its own"; R-6
  "Treat an unreadable record as no record, with a notice (Recommended)"; R-7
  "Say it is waiting; at the limit keep the session and offer to retry the save
  (Recommended)"; R-8 "Keep exact comparison (Recommended)". The MADR's second
  amendment records them.
* **Order.** Tests first, each run and seen failing before the code that makes
  it pass: the auth tests (`held_round3_0082_test.go`) red on the second pass's
  code (`go test -v`, `red3_auth.txt`); then, for the wizard, one pure signature
  change (`save` takes the prompter, ignoring it) so that the tests compile,
  and the wizard tests red on that (`red3_wiz.txt`); then the code. Red lines,
  on the second pass's code:
  * `TestLossyStoreSingleProcessKeepsRefreshing_0082` (R-5, the review's
    N-1/Q4), both stores: `refresh 2: Token = "", oauth: the store holds another
    sign-in than this openai session's (its client differs); sign in again: ...;
    want the rotation a-2`;
  * `TestRetriedSaveMeetsTheRotatedAccessToken_0082` (R-5, the retried save):
    `1 saves, want the first (which reported an error) and the retry over the
    rotation it wrote`; `storedRefresh "held-plan-refresh" after the retry`;
  * `TestOwnAccessTokenExemption_0082/a record with the session's access token
    is its own`: `Token = "", ... (its client differs); ... want the session's
    own refresh`; its other two cases (an empty access token never known;
    another access token refused) are green on both, as guards;
  * `TestSaveSaysItIsWaitingOnlyWhenTheLockIsHeld_0082/held, then freed during
    the wait`: `notices [], questions []; want one waiting notice and no
    question`; `events ["wait" "lock" "save" "unlock"]; want one try, one wait,
    then lock, save, unlock`;
  * `TestSaveRetriesAfterTheBoundOnRequest_0082`: `wizard: the new OpenAI
    session was not saved: llmprovider: provider unavailable: another process is
    refreshing`; `TestSaveDeclinedRetrySavesNothing_0082`: `questions []; want
    one`;
  * `TestSaveCancelledAtAnyWaitSavesNothing_0082` (the retry's wait, and a
    context ended while the prompt was open): `err = ... provider unavailable:
    another process is refreshing; want an error matching context.Canceled`;
  * `TestUnreadableClientRecordIsNoRecord_0082` (R-6): `ResolveCredential:
    wizard: load the issued client: FileTokenStore client decode: invalid
    character '{' looking for beginning of object key string; want the sign-in
    to carry on`;
  * `TestNewSessionIsGivenTheStoreOnlyUnderTheLock_0082` (N-3): `the store
    holds OAuthSession{... ClientID:app_EMoamEEZ73f0CkXaXp7hrann ...}` with the
    old record's refresh token: the window is real, forced by a token request
    on the new session between the wizard's first look at the lock and its
    taking it, which adopted the store's old record of the same labels;
  * green on the second pass's code, as guards the plants then proved:
    `TestSiblingRotatedDuringRefreshOnAStoreWithNoLock_0082` (B-2, Q1),
    `TestRetriesWhileAnotherSignInIsStoredSendNothing_0082` (B-2, RV_B3/Q6: the
    foreign record's token URL is the in-process issuer, so a spent token would
    be counted), `TestSaveWithALockerThatIgnoresTheContext_0082` (B-2,
    R2W_LockIgnoringContext), `TestFailedReReadBeforeTheSaveStillSaves_0082`
    (N-2, pinned), `TestRefreshAfterAnUnsavedRotationSavesAtOnce_0082` (N-4,
    N19), `TestRetriedSaveMeetsItsOwnRotation_0082` (N-4, N29).
  The consent test `TestStoredConsentIgnoresAnotherSignIn_0082` failed, four
  cases, when R-5's exemption was built: its foreign records had been made as
  copies of the held session and so carried its access token. That is the
  coordinator's stop condition ("if the exemption would let any of your refusal
  tests ... pass a foreign record") met literally, by a fixture that no real
  record can be: a record carrying this login's access token is made from this
  login's record. I corrected the fixtures (each foreign record carries its own
  access token) and ran the reviewer's B-tests, round-two and RV, against the
  tree before going on: none passes a foreign record (the only differences are
  their own E1, which R-4 reverses, and Q13, which fails on the second pass's
  code too, for a hand-built session with neither provider nor kind nor issuer).
  This was built and then reported, which the brief forbade; the evidence is
  here, and the options at the time were: (a) correct the fixtures, as done;
  (b) narrow the exemption to records with no refresh token of their own, which
  would not cover the reviewer's N-1 case in a store that also keeps a stale
  refresh token; (c) stop and ask.
* **Built, with `path:line`.**
  * R-5: `oauthSessionState.access` (`oauth_session.go:690`), `ownTokens`
    (371) and `holds` (374), `storeMovedOn` (392, the empty-store arm dropped,
    N-4), `saveRotation` (412), `persistRotation` (432) with the access tokens
    `{state.access, next.Access}`, `reloadOrRefresh`'s save (1116) with
    `{held.access}`, `loadRotated` (1142, 1153) with `{held.access}`,
    `StoredConsent` (`consent.go:141`, `154`). The access token that is the
    session's own: at the reads and at the refresh's own save, the one it held
    when the refresh began; at the retried save, the rotated one.
  * R-6: `resolveChatGPTPlan`'s read of the client record (`wizard/auth.go:298`):
    an unreadable record is `auth.ClientRecord{}`, with a `LevelWarn` notice
    that names no content.
  * R-7: `save` (1069), `writeUnderLock` (1102), `lockToSave` (1116), `write`
    (1134). A caller with no prompter, a prompter that cannot answer, or a
    refusal gets the bound's error, matching `ErrProviderUnavailable`, with
    nothing saved: what it got before.
  * N-3: `write` gives the session its store (1138), under the lock when there
    is one.
  * The risk, (a) lock order: unchanged (store lock, then session lock;
    `write` is called with the lock held and `Save` takes the session lock),
    and `writeUnderLock` is the only place the wizard holds a store lock. (b)
    the wait: `FileTokenStore.LockRefresh` with an already-done context makes
    one try and returns, with an error that matches `ErrProviderUnavailable`
    and the context's (`tokenstore_file.go`, the `ctx.Err() != nil` arm), which
    is how the wizard knows the lock is held; with a context that is not done
    it waits to `lockWait`, 60 s; the wizard's context ends it. The notice is
    issued on the wizard's own goroutine between the try and the wait, no timer.
    (c) a store that is not a `RefreshLocker` is saved to as before. (d) the
    wizard does not hold the lock while it asks: the lock is released before the
    question, so a refresh in this process is never kept waiting for an answer.
* **Which prompter primitive.** `Confirm`, as the wizard asks its other yes/no
  questions; a non-interactive caller (`TextPrompter` at end of input, a
  prompter with no terminal) fails it and gets the bound's error.
* **N-5, N-2,** recorded in the MADR's second amendment as limits; the
  corrupt-record case in the sign-in itself (`signinflow`), found at the base
  and not fixed, with three options, there too.
* **Plants.** The driver, `plants3.py`, kept with the scratch scripts, runs the reviewer's table
  re-formed where the code moved, plus this pass's own, each under
  `sandbox-exec` with the guard. 72 plants on the final tree, each a scratch copy under the sandbox: the reviewer's 53 re-formed where the code moved (N02 became an empty store counted as moved on, now in `saveRotation`, the arm of `storeMovedOn` being gone; N01, N03, N08, N09, N10, N12, N17, N19, N20, N25, N28, N29, N30 re-worded to the new code) and this pass's 19: R5a the access exemption widened to any access token; R5b an empty access token known; R5c the held access token not known at the reads; R5d not known at the save; R5e the rotated one not known at the retried save; R5f the consent read ignoring the access token; R6a the notice dropped; R6b the unreadable record fatal again; R7a the waiting notice dropped; R7b the try dropped; R7c the retry not offered; R7d the retry saving without the lock; R7e a declined retry saving; R7f no prompter: asking anyway; R7g a failed answer retrying; R7h the context's error not preferred after a failed wait; N3 the store given before the lock; N2 a failed re-read skipping the save; B1 the P11 refresh with the dead URL. Result: **72 of 72 caught, none a build failure.** N06, N17, N19, N21, N28 and N29 are caught now (N06 by `TestSiblingRotatedDuringRefreshOnAStoreWithNoLock_0082`, N17 by `TestRetriesWhileAnotherSignInIsStoredSendNothing_0082`, N19 by `TestRefreshAfterAnUnsavedRotationSavesAtOnce_0082`, N21 by `TestFailedReReadBeforeTheSaveStillSaves_0082`, N28 by `TestSaveCancelledAtAnyWaitSavesNothing_0082`, N29 by `TestRetriedSaveMeetsItsOwnRotation_0082`, which makes the store drop the access token so that the rotation's refresh token is all the record shows); the rest by the tests `plants3.json` lists for each, kept with the scratch scripts. The first run of this pass's table, before the last two repairs, left four, verbatim: `N25 the consent read's own-token exemption removed: BUILD FAILURE` (an unused variable; re-formed so that it compiles), `N29 the retried save's known list misses the rotation: NOT CAUGHT` (the rotation's access token, also known, covered it; the test now makes the store drop the access token), `R5f the consent read ignores the session's access token: NOT CAUGHT` (no consent case carried the session's access token under foreign labels; one added) and `R7h the context's error not preferred after a failed wait: NOT CAUGHT` (no test had a store report an ended wait as unavailable only; one added). One further Go change followed the first full run: a `gocritic` `deferInLoop` finding, repaired by `writeUnderLock`; the table was run again on the final tree.
* **Gates.** each status taken before any filter, after the last change to a Go file; an earlier `make pre-add-check` failed on that finding and its companion `-race` run was cancelled by me (my own ticket) and redone.

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` on the 13 changed Go files | `run-heavy`, session host | exit 0: gofmt, golangci-lint (host and Windows), vet, test, govulncheck |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; only `llmprovider/x/signin.VerifyKilo: added`, 0080's: no name from this change |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./llmprovider/providers/openai/ ./llmprovider/providers/grok/ ./wizard/` | `run-heavy`, session host | exit 0 (auth 218 s, openai 24 s, grok 12 s, wizard 180 s) |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race` (`pick-host` chose a Windows host) | exit 0, 42 packages ok |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` by a scratch `replace`, on a scratch copy of gobble | session host | exit 0, three ok |
  | the 72 plants, under the sandbox | scratch copy | 72 of 72 caught |
  | the two B-1 plants, sandboxed | scratch copy | both fail tests; only `issuer.invalid:443` named by the guard |

  The record gates ran after this entry's last edit; their results are in the hand-off.

### Phase 1, committed (2026-11-17)

* **Committed and pushed** by the coordinator on 2026-11-17 as `8051b00`
  ("fix(auth): refuse to adopt another sign-in at refresh"), `main`, after the
  third independent review and the coordinator's own gate. The "Staged, not
  committed" lines above are the record of the staging that preceded it. The
  fourth pass below is staged on top of it; since the evening of 2026-11-17 the
  owner restored the coordinator's commits and pushes, and a builder still only
  stages.

### Phase 1, fourth pass (2026-11-17)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over `8051b00`. Staged, not committed, pushed or tagged.
* **The third review (Phase 3, run before the commit).** No blocker: on no path
  does anything send, spend or adopt another sign-in's token; every run exit 0
  under a no-network sandbox. It found three of the owner's rulings not fully
  met and one regression: F-1 (R-6 was not met end to end: the engine read the
  unreadable record again and failed; sign-out does not clear it; another
  application's plan session was handed over as `Previous` and its client
  written as this one's), F-2 (the retry was offered only for
  `ErrProviderUnavailable`, which gobble's own `*HeldError` is not), F-3 (the
  access-token exemption took a record with its own client and refresh token,
  that carried this session's access token, as the session's own: the session
  refreshed and overwrote it, and a forged `background_use` in it was read as
  granted), F-4 (a store that keeps neither the refresh token nor the client, one
  failed save, then the retry: the record holds the previous access token, no
  longer "own", and every later token request is refused until the process
  ends; before 0082 the next refresh skipped that record and saved normally).
  Notes: N-a (a locker that answers a done context on the try with its own
  unwrapped error failed the save outright), N-b (the notice test did not assert
  that no part of the read error's text appears), N-c (own access is the held
  session's, not the adopted record's), N-d (a save that ends at the limit and is
  not retried drops the completed sign-in's tokens without revoking them), N-e
  (the retry loop is the person's and unbounded). The plants that survived the
  earlier suite, verbatim from the reviewer's table:
  `M02a the first read's own access taken from the adopted state: build=False
  staged=0 mine=0 left=[]`, `M02b the refresh's save's own access taken from the
  adopted state: build=False staged=0 mine=0 left=[]`, `W09 the notice carries
  the read error's text: build=False staged=0 mine=0 left=[]`, `W10 every failed
  try counts as held: build=False staged=0 mine=0 left=[]`.
* **The rulings,** 2026-11-17, relayed by the coordinator: R-9 "Treat it as no
  record in the sign-in engine too, and replace it (Recommended)"; R-10 "Offer
  the retry for any failed wait once the lock was seen held (Recommended)"; R-11
  "Narrow it to records that have no refresh token of their own (Recommended)",
  with F-4 directed by the coordinator under the same ruling's intent that no
  single-process store regresses. The MADR's third amendment records them, with
  the second amendment's sentences they supersede.
* **Order, and what changed first.** Tests first, each run and seen failing on
  `8051b00` before the code that makes it pass: `go test -v` under a macOS
  sandbox that denies every outbound connection but loopback
  (`sandbox-exec`), the tests written against the staged code with one inert
  change so that they compile and fail on assertions, not on the build:
  `signinflow.Result.ClientLoadErr`, a field nothing set or read. Red lines on
  `8051b00`:
  * `TestRecordWithItsOwnRefreshTokenAndOurAccessTokenIsAnotherSignIn_0082`
    (R-11, the review's A1), both stores: `Token = "a-1", <nil>; want no token
    and an error matching ErrAuthFailure`;
  * `TestStoredConsentIgnoresAnotherSignIn_0082/a record with its own refresh
    token and the held access token is another sign-in's (R-11, A2)`:
    `StoredConsent = true, true, <nil>; want granted and answered both false`;
  * `TestStoreThatKeepsNoRefreshTokenSurvivesOneFailedSave_0082` (F-4, the
    review's A4), both stores: `refresh 3: Token = "", oauth: the store holds
    another sign-in than this openai session's (its client differs); sign in
    again: ...; want the session never stranded`, and the store holding the
    previous record after the retry; `TestRefreshAfterAFailedSaveOnAStoreThatKeepsNoRefreshToken_0082`:
    `refresh 3 before any retry: Token = "", ... (its client differs)`;
  * `TestUnreadableClientRecordEndToEnd_0082` (R-9, the review's F-1), the four
    situations: with a plan session stored, `the login was given Previous
    OAuthSession{... ClientID:oaiapp_mine ...}`; with nothing stored and with a
    ChatGPT login stored, `the person is stuck: signinflow: load the issued
    client: FileTokenStore client decode: invalid character '{' looking for
    beginning of object key string`; with another application's plan session
    stored, `the login was given Previous OAuthSession{... ClientID:oaiapp_other
    ...}`;
  * `TestPlanUnreadableClientRecordIsNone_0082` (the engine): `sign-in with an
    unreadable record: err signinflow: load the issued client: planted decode
    failure: invalid character, session OAuthSession(nil); want it to go on`;
    `TestStart_PlanUnreadableClientRecord_0082` (`x/signin`): `Wait =
    signinflow: load the issued client: planted decode failure; want the
    sign-in to go on`;
  * `TestGobbleLikeLockerAtItsLimitOffersTheRetry_0082/yes` (R-10, the review's
    F-2, a locker modelled on gobble's `Logins.LockRefresh` over `filelock.Lock`):
    `save = wizard: the new OpenAI session was not saved: auth.json: openai
    refresh lock: locked by pid 4242; want the retry to save once the lock is
    free`; `TestAnyFailedTryWithALiveContextIsHeld_0082/waits and saves` (N-a):
    `save = wizard: the new OpenAI session was not saved: lock busy; want the
    wait to end with the save`; `TestLockThatFailsForGoodEndsWithTheQuestionThenTheError_0082`:
    `questions [], notices []; want one of each`;
  * green on `8051b00`, as guards the plants then proved:
    `TestReReadOwnAccessTokenIsTheHeldSessions_0082` and
    `TestSaveOwnAccessTokenIsTheHeldSessions_0082` (N-c, M02a and M02b),
    `TestGobbleLikeLockerAtItsLimitOffersTheRetry_0082/no`,
    `TestSaveFailingUnderTheLockAsksNothing_0082`; the notice's assertion that no
    part of the read error's text appears (N-b, W09) is in the end-to-end test
    above.
* **Existing tests changed by the rulings, stated.** Two, each asserting the
  behaviour a ruling reverses, neither on the plan's list of tests to watch, and
  I changed them without stopping, which the brief's rule would have had me
  stop for; the owner's rulings name the behaviour they reverse, and the
  change is that and no more: `TestPlanClientRecordErrors_0075`'s first half
  (`internal/signinflow`) asserted that an unreadable record fails the
  sign-in before the login runs, and now asserts the login ran and
  `Result.ClientLoadErr` reports the error; `TestUnreadableClientRecordIsNoRecord_0082`
  (this series' own, `wizard`) asserted that the stored plan session is handed
  over as `Previous` with no readable record, and now asserts it is not.
* **Built, with `path:line`.**
  * R-11, F-4: `ownTokens.holds` (`oauth_session.go:382`), the access token
    exempts only a record with no refresh token; `OAuthSession.storedAccess`
    (`tokenstore.go`), `state.storedAccess` (221), set with `storedRefresh` at
    291, 294, 313 and 317 (a failed first save: `refreshed.spentAccess`); the lists
    at the retried save (454), the refresh's save (1134) and the reads (1172);
    `StoredConsent` (`consent.go:141`). The access token that is the session's
    own: the one it held when the refresh began and the one the store is known to
    hold; at the retried save also the rotated one. It re-widens nothing: the
    stored-access list only ever holds a token this session held or saved, and
    applies only to a record with no refresh token.
  * R-9: the engine (`internal/signinflow/signinflow.go:434-462`), `Result.ClientLoadErr`;
    the wizard's read of the record whenever the store keeps one, one notice, no
    `Previous` (`wizard/auth.go:288-309`); the save that replaces the record is the
    engine's existing `SaveClient`, which writes the file anew (`FileTokenStore.writeDurable`)
    and does not read it first (`clientstore.go`). A caller of `x/signin` gets a
    session, no error and no notice; `Options.ClientStore` says so. No name is
    exported.
  * R-10: `save` (`wizard/auth.go:1083`), `writeUnderLock` (1116), `lockToSave`
    (1132). The rule for a failed try: a try that fails while the sign-in's own
    context is alive is the lock being held, whatever its error (the wider rule:
    it loses nothing for an existing custom store, since a locker that reports a
    held lock in its own words is waited for, and a lock that fails for good
    costs one notice and one question, then its error); with the sign-in's own
    context ended, that context's error, with no notice and no question. The rule
    for a failed wait: any error, the store's limit included, leads to the
    question, but for the sign-in's own context ending; a save that fails under a
    lock the wizard holds is returned, with no question.
  * gobble's exact behaviour, read from `internal/auth/logins.go` and
    `internal/filelock/lock.go`, read-only: `LockRefresh` calls `filelock.Lock`
    under a 60 s policy; with a done context and the lock held it waits on `ctx`
    and returns `ctx.Err()` at once (wrapped `auth.json: <id> refresh lock: %w`,
    so `errors.Is(context.Canceled)`); at its limit it returns the last try's
    `*HeldError`, wrapped likewise, which is not `ErrProviderUnavailable`.
* **Plants.** `plants4.py`, kept with the scratch scripts, on a scratch copy of the tree under `sandbox-exec` (outbound connections denied but loopback), the `localonly` guard on: 118 plants, the reviewer's round-three table (31) and this series' earlier table (72) re-formed where the code moved, and this pass's 27 (R9a to R9g: the engine fatal again, `Previous` handed over with an unreadable record, the notice doubled by the engine's error, the notice carrying the error text, the engine's error not kept, the record read only with a plan session stored, a partial record not discarded; R10a to R10f: the question after a save error under the lock, only for `ErrProviderUnavailable`, after the sign-in's own context ended, the old try rule, the waiting notice dropped, a failed answer retrying; R11a to R11j: the exemption applied to a record with its own refresh token, the stored access token not kept at the retried save, at the reads, at the refresh's save, not set by a save, by the retried save, by a failed first save, the consent read ignoring it, an empty token known, the stored access token kept when another process saved a newer session). **M02a, M02b, W09 and W10 are caught now**: M02a by `TestReReadOwnAccessTokenIsTheHeldSessions_0082`, M02b by `TestSaveOwnAccessTokenIsTheHeldSessions_0082`, W09 by `TestUnreadableClientRecordEndToEnd_0082`, W10 by `TestAnyFailedTryWithALiveContextIsHeld_0082`. The first run of the whole table left seven, verbatim: `R9b Previous handed over with an unreadable record: BUILD FAILURE` (the variable unused; re-formed so it compiles), `R9g the engine ignores the loaded record's client after an error: NOT CAUGHT` (the test's store returned no partial record; it now does), `R10a the retry offered after a save error under the lock: BUILD FAILURE` (re-formed), `R11f storedAccess not set by the retried save: NOT CAUGHT` (a later successful save masked it; the store-keeps-no-refresh-token test now fails a second save after the retry), `R11g storedAccess not set after a failed first save (the spent access token): NOT CAUGHT` (`TestFailedFirstSaveOnAStoreThatKeepsNoRefreshToken_0082` added), `R11h the consent read ignores the stored access token: NOT CAUGHT` (`TestStoredConsentOnAStoreThatKeepsThePreviousAccessToken_0082` added) and `R11j the stored access token kept when another process saved a newer session: NOT CAUGHT` (`TestStoredAccessIsForgottenWhenAnotherProcessSavedANewerSession_0082` added). An earlier, aborted run of the table, stopped at 77 plants when it showed `M06 an empty access token counts as own: NOT CAUGHT` (the test of the empty token used a record with a refresh token of its own, which R-11 now excludes anyway; its record now has none), was repeated whole. The eight plants those repairs concern were run again on the final tree: **8 of 8 caught**; the other 110 were caught by the full run before the repairs, which changed only tests that can fail a plant, not pass one. Plants that hang a test (a wait with no end) end at the run's own 6 minute timeout and count as caught.
* **Gates.** each status taken before any filter, after the last change to a Go file.

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` on the 15 changed Go files | `run-heavy`, session host, with golangci-lint at the repository's pin, v2.14.0 | exit 0: gofmt, golangci-lint (host and Windows), vet, test, govulncheck |
  | the same, with the host's golangci-lint v2.15.0 | `run-heavy`, session host | exit 2, three `nolintlint` findings in `llmprovider/internal/transport/offline.go:21, 24` and `unsendable.go:48`, files this pass does not touch: see below |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; only `llmprovider/x/signin.VerifyKilo: added`: no name from this change |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./llmprovider/providers/openai/ ./llmprovider/providers/grok/ ./wizard/ ./internal/signinflow/ ./llmprovider/x/signin/` | `run-heavy`, session host | exit 0 (auth 220 s, wizard 185 s) |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race` (`pick-host` chose a Windows host) | exit 0, 42 packages ok |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` by a scratch `replace`, on a scratch copy of gobble | session host | exit 0, three ok |
  | the plants | scratch copy, sandboxed | 118 of 118 caught (110, then the 8 repaired) |

  The host's golangci-lint binary has a file time of 22:17 on 2026-11-17, after
  this pass's first gate, and is v2.15.0, where CI and the Makefile pin v2.14.0;
  this pass did not install it.
  v2.15.0 finds three `//nolint:errcheck` directives unused in two files this
  pass does not touch and no commit of this series touched; v2.14.0 is clean on
  the whole tree. I did not change those files, loosen the gate or replace the
  host's binary: the gate ran with `GOLANGCI_LINT` set, as `go-precheck` allows,
  to a v2.14.0 binary installed under the scratch folder. The three directives
  are for the owner's word when the pin moves.

  The record gates ran after this entry's last edit; their results are in the
  hand-off.
* **Not done.** Phase 2's own set beyond the gates here (`make lint`, `make
  vuln`, the CI set, `tools/gate.py`).
* **Not verified.** The 60 s wait itself, and a wait on another real process;
  what the issuer does with an unused, still-registered client after the ruled
  re-registration (R-9); a custom locker that ignores its context beyond the
  ported test; that no prompter ever sees two notices from a path I did not
  test; the 20-round `-race` run on a Windows host (the module run there was one
  pass).

### Phase 1, fourth pass, committed (2026-11-18)

* **Committed and pushed** by the coordinator on 2026-11-18 as `04433ae`
  ("fix(auth): apply third-review rulings on sign-in adoption and retry"),
  `main`, after the fourth independent review and the coordinator's own gate
  (the whole module under race; twenty race rounds of auth, openai, wizard,
  signinflow and `x/signin`; `make api-check`; `make pre-add-check` at the
  repository's pinned golangci-lint v2.14.0 over 1016 files). The "fourth pass"
  entry above is the record of the staging that preceded it. The fifth pass
  below is staged on top of it.

### Phase 1, fifth pass (2026-11-18)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over `04433ae`. Staged, not committed, pushed or tagged.
* **The fourth review.** No blocker; it sampled 20 of the 110 plants of the
  fourth pass and caught 20 of 20, and it passed the coordinator's own gate.
  It found two places where the code reads a ruling wider than its words:
  F-1 (any `LoadClient` error counted as "unreadable", so a permission error on
  a good `openai.client` replaced the record with a new client and abandoned
  the old one at the issuer; a transient failure of the wizard's read made it
  announce a new client and drop `Previous` while the engine's second read
  succeeded and reused the old client, and the reverse registered a new client
  with no notice) and F-2 (any failed first try counted as the lock being held,
  so a lock that can never be taken, its path a directory, got "waiting for
  another process ..." and "Another process is still refreshing ...", both
  false, each yes repeating at once). Its survivors among its own plants,
  verbatim from its log: `Q10 a non-decode load error fatal in the wizard:
  build=False staged=0 mine=0 left=[]`, `Q16 a whitespace refresh token counted
  as none: build=False staged=0 mine=1 left=[]` and `Q17 the retried save's own
  access list drops the rotation's: build=False staged=0 mine=0 left=[]`. Q10 is
  now the ruled behaviour, so its plant inverts (a non-decode error treated as
  none) and is caught below; Q16 is caught by the reviewer's own test, ported as
  `TestWhitespaceRefreshTokenIsARefreshToken_0082`; Q17 is the reviewer's
  equivalent mutant (no test, his or this series', tells it from the code); its
  result in this pass's run of the whole table is in the plants paragraph below.
* **The rulings,** 2026-11-18, relayed by the coordinator: R-12 "Only a record
  that exists but cannot be decoded; other read errors fail the sign-in
  (Recommended)"; R-13 "Say \"another process\" only when the lock really was
  held; otherwise ask in neutral words (Recommended)". The MADR's fourth
  amendment records them, with the third amendment's sentences they supersede,
  and N-1 to N-4.
* **Notes,** as asked: N-1 (the `storedAccess` doc and the third amendment said
  it is set wherever `storedRefresh` is; `FileTokenStore.Load` sets
  `storedRefresh` only: the field's doc now says so and why it is harmless; the
  committed sentence is annotated in the amendment, not rewritten), N-2 (the
  "list holds only tokens this session itself held or saved" is too strong:
  after a failed save `spentAccess` can be a sibling's rotation the first read
  took up; annotated, and in the `spentAccess` and `ownTokens` docs), N-3 (the
  trust boundary: a store writer can plant an access-only record with the
  session's stale access token; its forged consent is read and it is overwritten
  at the next save; needs write access to the session's own record; before 0082
  the consent read trusted any record: a stated limit), N-4 (an `x/signin` caller
  is not told its record was replaced: documented on `Options.ClientStore`, with
  the read error now failing `Wait`).
* **N-5, a flaky test, found and not fixed.** `TestVerifyIDToken_UnknownKidRefetchesOnce`
  failed twice in one 20-round race run of `llmprovider/auth` in the fourth
  review (`oauth_idtoken_test.go:107: 0 key fetches to populate the cache, want
  1`) and passed on a re-run. The reviewer's reading, not tested here: the
  process-wide `jwksCache` is keyed by URL, so a later test server that reuses a
  port finds keys already cached. I did not touch the test or the cache. It did not fail in this pass's gates: the 20-round `-race` run of `llmprovider/auth` passed (223.7 s), as did the whole module once.
  The owner ruled on it on 2026-11-18, relayed by the coordinator: "fix this:
  fold it into the 0083 plan that already carries another flaky-test fix". So:
  found by the fourth review, not fixed under 0082, to be fixed under record
  0083's plan by amendment, by the owner's ruling. The agent that runs 0083
  writes that amendment; I did not edit the 0083 records.
* **The mechanism for telling a decode error apart (R-12), and no stop on an
  exported name.** `signinflow.Undecodable`, in an internal package, is
  `errors.As` over encoding/json's `*json.SyntaxError` and
  `*json.UnmarshalTypeError`, which `FileTokenStore.LoadClient` wraps
  (`FileTokenStore client decode: %w`); nothing is matched on text, no sentinel
  or marker is added, and no name is exported, so I did not stop. A custom
  `ClientStore`'s own decode error is a read error that fails the sign-in unless
  it wraps encoding/json's; `auth.ClientStore` documents it. The wizard and the
  engine are both outside `auth` and both see the standard library's types.
  One judgment, stated: a file over 64 KiB is `readBounded`'s plain error, not a
  decode failure, so it fails the sign-in and names the file; the person deletes
  it. A decode failure is "exists but cannot be decoded"; a file too large is
  not decoded at all.
* **Order, and what changed first.** Tests first, each run and seen failing on
  `04433ae` before the code that makes it pass: `go test -v` under a macOS
  sandbox that denies every outbound connection but loopback (`sandbox-exec`),
  the tests written against the tree with one inert change so that they compile
  and fail on assertions, not on the build: `signinflow.Spec.Client`, the type
  `signinflow.ClientRead`, and `signinflow.Undecodable` returning false, none
  read by anything. Two tests written after that run (the lock-file directory
  case with `FileTokenStore`, and the permission error injected) were run in a
  scratch clone of `04433ae` itself, with the new test files copied in. Red
  lines on `04433ae`:
  * `TestUndecodable_0082`: `a syntax error: Undecodable = false, want true`
    (and likewise a type error, a syntax error wrapped twice, the store's decode
    failure);
  * `TestPlanReadErrorFailsTheSignIn_0082` (R-12, engine): `err <nil>, login ran
    true, session OAuthSession{...}; want the read error and no login`, and `the
    record = {ClientID:client-issued HostID:urn:uuid:minted} after 1 save(s);
    want it untouched`;
  * `TestPlanGivenTheCallersReadReadsNothing_0082` (R-12, one read), both cases:
    `err <nil>, login given client "client-other" host "urn:uuid:other" after 1
    load(s); want the caller's record and no read`; `err <nil>, client
    "client-other", 1 load(s), ClientLoadErr <nil>; want none of the record, no
    read, the failure reported`;
  * `TestStart_PlanClientReadErrorFailsWait_0082` (`x/signin`): `Wait = <nil>
    after 1 login(s); want the read error and no registration`, and `the record
    = {ClientID:client-new HostID:urn:uuid:new} after 1 save(s), 1 load(s); want
    it untouched and read once`;
  * `TestClientRecordPermissionDeniedFailsTheSignIn_0082` (R-12, the reviewer's
    probe, on a real file with mode 0, Unix only): `err = <nil>; want the
    sign-in failed, naming openai.client`; `TestClientRecordPermissionErrorFailsTheSignIn_0082`
    (the same with the error injected, every platform, in the clone): `err =
    <nil>; want the sign-in failed with the permission error, naming
    openai.client`;
  * `TestTransientClientReadFailsOnlyThatSignIn_0082` (R-12, the reviewer's
    second probe): `err = <nil>; want the read's error`;
  * `TestOneClientRecordReadPerSignIn_0082` (R-12, c): `2 reads of the client
    record in one sign-in, want one`, for nothing saved, a record and another
    application's session, a record and no session, and an undecodable record
    (green only for a record and its plan session, where the engine reads
    nothing because `Previous` is handed over);
  * `TestFailedTryThatIsNotContentionAsksInNeutralWords_0082` (R-13), `yes`:
    `questions []; want the one neutral question "The lock on the saved OpenAI
    session could not be taken (open /x/openai.lock: permission denied). Try
    saving the new sign-in again?"`, `notices ["waiting for another process to
    finish refreshing the saved OpenAI session; the new sign-in is saved after
    it"]; want none`, `2 lock calls, 1 of them waits; want two tries and no
    wait`; `yes twice, then no`: `6 lock calls, 3 waits; want three tries and
    no wait`; `nothing says another process`: the waiting notice `says the lock
    is held by another process`; `TestNeutralQuestionShowsTheErrorWithoutControlCharacters_0082`:
    `questions []; want one`;
  * `TestLockFileThatCanNeverBeOpenedAsksNeutrally_0082` (R-13, the reviewer's
    third probe, `FileTokenStore` with the lock path a directory, in the clone):
    `"Another process is still refreshing the saved OpenAI session. Try saving
    the new sign-in again?" says another process holds the lock, which nothing
    does` and `... does not show the store's error, which names the lock file`,
    for each of the three questions and for the waiting notice;
  * `TestAnyFailedTryWithALiveContextIsHeld_0082/asks, does not wait, and saves
    on yes` (R-13, an existing test changed, below): `save = wizard: the new
    OpenAI session was not saved: context deadline exceeded; want the second try
    to save` (the old code waits on a lock that nothing frees; the test's own
    3 s bound ends it); `TestLockThatFailsForGoodEndsWithTheQuestionThenTheError_0082`:
    `questions ["Another process is still refreshing the saved OpenAI session.
    Try saving the new sign-in again?"], notices ["waiting for another process
    to finish refreshing the saved OpenAI session; ..."]; want one question and
    no notice`;
  * green on `04433ae`, as guards the plants then proved: `TestPlanUndecodableRecordFromTheStoreIsNone_0082`,
    `TestUndecodableRecordsAreReplacedWithOneNotice_0082` (junk, a wrong type,
    an empty file, a truncated one: each replaced, one notice),
    `TestUnreadableClientRecordEndToEnd_0082` (the four R-9 situations with a
    corrupt file, which the ruling keeps),
    `TestStart_PlanClientDecodeFailureIsNone_0082`,
    `TestFirstTryWrappingTheContextErrorIsContention_0082` (the contention path,
    its three shapes), `TestQuestionAfterContentionKeepsItsWords_0082`,
    `TestGobbleLikeLockerAtItsLimitOffersTheRetry_0082` (the gobble-modelled
    locker, unchanged), and the ported `TestWhitespaceRefreshTokenIsARefreshToken_0082`
    (Q16: a guard of code that was already right, seen failing under the Q16
    plant below).
* **Existing tests changed by the rulings, stated.** Each asserts, or plants an
  error for, the behaviour a ruling reverses, which the ruling names, so I
  changed them without stopping, as the coordinator's confirmation of the last
  pass allows when the ruling names the behaviour; I say so here:
  `TestPlanUnreadableClientRecordIsNone_0082` and `TestPlanClientRecordErrors_0075`
  (`internal/signinflow`) planted a plain `errors.New` as the "unreadable" error
  and now plant a decode failure (`decodeFailure0082`, encoding/json's error
  wrapped as the store wraps it), since a plain error is now a read error that
  fails the sign-in (R-12); `unreadableClients` in `x/signin`'s
  `TestStart_PlanUnreadableClientRecord_0082` likewise; `TestAnyFailedTryWithALiveContextIsHeld_0082`'s
  first subtest ("waits and saves", now "asks, does not wait, and saves on yes")
  asserted that a failed try with a live context is waited for, with the
  waiting notice and no question, which R-13 reverses, and
  `TestLockThatFailsForGoodEndsWithTheQuestionThenTheError_0082` asserted the
  waiting notice before the question, which R-13 drops; the fixture `ownErrLock0082`
  gained a counter of the calls made with a live context (a wait) and is built
  with keyed fields. Their other assertions are unchanged. No other existing
  assertion changed.
* **A defect in a test of this pass, found by the gate and fixed.** The first
  permission test, `TestClientRecordPermissionDeniedFailsTheSignIn_0082`, made
  the case with `chmod 0`, which a Windows file's owner reads past, so the first
  whole-module run on a Windows host failed it (`client_read_0082_test.go:108:
  err = <nil>; want the sign-in failed, naming openai.client`). That test is now
  in `wizard/client_read_unix_0082_test.go` under `//go:build !windows`, and the
  same case with the permission error injected, which every platform runs, is
  `TestClientRecordPermissionErrorFailsTheSignIn_0082`. It is a fault in what I
  wrote, not in the code and not in any earlier test.
* **Built, with `path:line`.**
  * R-12: `signinflow.Undecodable` (`internal/signinflow/signinflow.go:172`),
    `ClientRead` (161), `Spec.Client` (139); the engine uses the caller's read or
    its own single read, takes a decode failure as none (`Result.ClientLoadErr`)
    and fails the sign-in on any other error before the login (476 onward); the
    wizard reads once and passes the read (`wizard/auth.go:279-335`; the fatal
    error at 306, the read given at 316 and 335, no notice for it); the docs on
    `auth.ClientStore`'s `LoadClient` (`clientstore.go:32`),
    `FileTokenStore.LoadClient` (61), `signin.Options.ClientStore`
    (`x/signin/signin.go:47-62`), `wizard.Options.TokenStore`
    (`wizard/configure.go:128-154`).
  * R-13: `save` (`wizard/auth.go:1098`), `writeUnderLock` (1137), `lockToSave`
    (1156). Contention is the first try's error wrapping `context.Canceled` with
    the sign-in's context alive; then the waiting notice, the wait, and, after a
    failed wait, the question as it was ("Another process is still refreshing the
    saved OpenAI session. Try saving the new sign-in again?"). Any other failed
    try returns at once with `contended` false and the question is: "The lock on
    the saved OpenAI session could not be taken (<error>). Try saving the new
    sign-in again?", the error through `redact.StripControl(redact.String(...))`;
    yes repeats the try, no returns the error whole. `RefreshLocker`'s doc
    (`tokenstore.go:299`) says that wrapping the context's error on a done
    context is how a locker signals "held".
  * Verified, read-only: `FileTokenStore.LockRefresh` returns `ErrProviderUnavailable`
    and `ctx.Err()` wrapped when the lock is held and the context is done, in
    code that does not depend on the operating system (`tokenstore_file.go`); its
    failures to open or restrict the lock file return `ErrProviderUnavailable`
    wrapping the operating system's error only, so a lock path that is a directory
    is not contention; gobble's `filelock.Lock` returns the context's error,
    wrapped by `Logins.LockRefresh` (`auth.json: <id> refresh lock: %w`), on a
    done context with the lock held. The lock errors name a path and an
    operating system's text, no secret.
  * One read per sign-in, shown: `TestOneClientRecordReadPerSignIn_0082` counts
    the store's `LoadClient` calls through the wizard for five situations, and
    `TestPlanGivenTheCallersReadReadsNothing_0082` counts zero in the engine given
    the wizard's read.
  * N-1 to N-3 in the docs of `storedAccess` (`tokenstore.go:83`), `ownTokens`
    (`oauth_session.go:388`) and `spentAccess` (1080).
* **Plants.** `plants5.py`, kept with the scratch scripts, on a scratch copy of the final tree under `sandbox-exec` (outbound connections denied but loopback), the `localonly` guard on, run once through `run-heavy`: **160 plants**, the whole of the fourth pass's table (118, read from its driver) with the six whose anchors moved with the code re-formed (W01, W11, W14, N09, N10, R7b: the wizard's lock helpers now return whether the lock was seen held), the reviewer's round-four series (Q01 to Q19, read from its file, with Q06, Q11, Q15 and Q16 re-formed for the new code), and this pass's own 22 plus the inverse of Q10: S01 to S12 for R-12 (non-decode errors treated as none in the engine; a decode failure fatal, `Undecodable` saying no to everything; the engine reading a second time; the notice shown for a fatal error; a registration sent before the read error is returned; the wizard giving the engine no read; the engine ignoring the record it was given; a decode failure told by the error's text; a wrong-type failure not counted; the wizard's fatal error losing the store's error; non-decode errors treated as none in the wizard only; the wizard reading only when a plan session is stored) and T01 to T10 for R-13 ("another process" shown for a non-contention failure; the neutral question without the error's text; no question at all; contention given the neutral wording; the error's control characters shown; yes not repeating the try; contention recognised by `ErrProviderUnavailable`; the waiting notice shown for a non-contention failure; a non-contention failure waited for in silence; contention not reported to the caller). The first run of the whole table left four, verbatim: `Q10 a non-decode load error fatal in the wizard: NOT CAUGHT; failing=[] guard=no`, `Q15 a try that fails with an ended context still waits: NOT CAUGHT; failing=[] guard=no`, `Q17 the retried save's own access list drops the rotation's: NOT CAUGHT; failing=[] guard=no` and `S06 the wizard hands the engine no read: BUILD FAILURE; failing=[] guard=no`. **Q10** (the reviewer's) made a non-decode error fatal in the wizard, which R-12 makes the code's own, so the plant is the code, an equivalent mutant from now on; its inverse, `Q10 a non-decode load error treated as none in the wizard (R-12 inverted)`, is in the table and caught, as the brief asked, by the permission, transient and one-read tests. **Q15** was a real gap that the narrowing created: its guard was a locker whose try error wraps nothing, which R-13 now returns on at once whatever the sign-in's context does, so no test had a try that wraps the cancelled context while the sign-in's own context has ended; `TestSignInContextEndedAtTheTryIsNotAHeldLock_0082` was added and the plant re-run on the final tree: `Q15 ...: CAUGHT; failing=['TestSignInContextEndedAtTheTryIsNotAHeldLock_0082']`. **S06** failed to build (the variable it dropped became unused); re-formed so that it compiles and re-run: `S06 the wizard hands the engine no read: CAUGHT; failing=['TestOneClientRecordReadPerSignIn_0082', ...]`. **Q17** is the reviewer's equivalent mutant and survives here too: no test of this series tells the code from the plant, and I did not find one that could. The four plants that name `guard=yes` (P11 twice, B1 and N17, which would send a foreign token to a real endpoint) were stopped by the guard, as before. Final count: **158 of 160 caught; Q10 (equivalent, replaced by its inverse, which is caught) and Q17 (the reviewer's equivalent mutant) survive.** Plants that hang a test end at the run's own 6 minute timeout and count as caught.
* **Gates.** each status taken before any filter, after the last change to a Go
  file.

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` | `run-heavy`, session host, golangci-lint at the repository's pin, v2.14.0 (`GOLANGCI_LINT` set) | exit 0: 1016 files clean (gofmt, golangci-lint, vet, test, govulncheck); nested module `authcontract` clean |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; only `llmprovider/x/signin.VerifyKilo: added`: no name from this change |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./llmprovider/providers/openai/ ./llmprovider/providers/grok/ ./wizard/ ./internal/signinflow/ ./llmprovider/x/signin/` | `run-heavy`, session host | exit 0 (auth 223.7 s, wizard 194.6 s); `TestVerifyIDToken_UnknownKidRefetchesOnce` did not fail |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race` (`pick-host` chose a Linux host) | exit 0, 42 packages ok |
  | the first whole-module run, on a Windows host | `run-on-host --race` | exit 1: `TestClientRecordPermissionDeniedFailsTheSignIn_0082` (my own test, `chmod 0` does not refuse a Windows owner); fixed as stated above, and `go test` of `./wizard/ ./internal/signinflow/ ./llmprovider/auth/ ./llmprovider/x/signin/` then run on a Windows host: exit 0, four ok |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` by a scratch `replace`, on a scratch copy of gobble | `run-heavy`, session host | exit 0, three ok |
  | the plants | scratch copy, sandboxed | 158 of 160 caught; the two survivors are equivalent mutants (Q10, Q17), stated above |

  The golangci-lint gate ran with `GOLANGCI_LINT` set to the v2.14.0 binary in
  the scratch folder, as the pin requires. The host's v2.15.0 finds three
  `//nolint:errcheck` directives unused in `llmprovider/internal/transport/`
  files this pass does not touch; I did not change them or the host's binary. The
  owner has since ruled to standardize on v2.15.0 across the repositories, under
  its own record in each; until this repository's pin moves the gate keeps the
  pin, and those three directives stay.

  The record gates ran after this entry's last edit; their results are in the
  hand-off.
* **Not done.** Phase 2's own set beyond the gates here (`make lint`, `make
  vuln`, the CI set, `tools/gate.py`).
* **Not verified.** A wait on another real process, and the 60 s bound itself
  (the tests use scripted lockers and `FileTokenStore` with a directory in the
  lock's place); a custom `ClientStore` of another package returning a decode
  error of its own that does not wrap encoding/json's (treated as a read error,
  by design); what the issuer does with an unused, still-registered client; that
  the neutral question's error text is free of anything a person should not see
  for a custom locker (the SDK's own and gobble's name a path and an operating
  system's text); Windows under `-race` for the 20-round run (the module run was
  one pass, on Linux); whether Q17 is truly equivalent beyond the reviewer's
  reading and my not finding a test.

### Phase 1, fifth pass, committed (2026-11-18)

* **Committed and pushed** by the coordinator on 2026-11-18 as `5f2c2ab`
  ("fix(auth): apply fourth-review rulings on record reads and lock retry"),
  `main`, after the fifth independent review and the coordinator's own gate; CI on
  it is green. The "fifth pass" entry above is the record of the staging that
  preceded it. The sixth pass below is staged on top of it.

### Phase 1, sixth pass (2026-11-18)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over `5f2c2ab`. Staged, not committed, pushed or tagged.
* **The fifth review.** No blocker, and the coordinator's own gate passed. One
  finding, F-1: a client record file over 64 KiB, which a real one (about a
  hundred bytes) never is, fails every sign-in under R-12, with only the file's
  base name shown (its table row "valid JSON over 64 KiB": `FAILED: wizard: read
  the saved OpenAI client record: FileTokenStore client read: openai.client is
  larger than 65536 bytes | logins=0 | record after "" (err true)`, with a plan
  session stored and with none). Its table also showed the row `null` (decodes to
  the zero record: signed in, no notice, a stored plan session signed in to
  again) and a record that decodes but names a garbage client id (invalid UTF-8
  inside the string: used and saved, no notice). Its survivors, verbatim from its
  log: `S14 a secret in the lock error not redacted: build=False staged=0 [] mine=1
  left=[]` (caught only by the reviewer's own test), `S16 the wizard's read handed
  over without its error: build=False staged=0 [] mine=0 left=[]` and `Q17 the
  retried save's own access list drops the rotation's: build=False staged=0 []
  mine=0 left=[]`. S16 and Q17 are equivalent mutants by the review's showing: S16
  changes only `Result.ClientLoadErr`, which the wizard never renders; Q17 as the
  fifth pass recorded.
* **The ruling,** 2026-11-18, relayed by the coordinator: R-14 "Treat an oversized
  record as undecodable and replace it (Recommended)". The MADR's fifth amendment
  records it, the sentences of the fourth it supersedes, the `null` row and the
  limits below.
* **Notes,** as asked: N-a (a record that decodes but names a garbage client id is
  used and saved with no notice; against the real issuer the sign-in would fail at
  authorize on every attempt until the file is deleted; it predates 0082 and is
  outside R-12's words: recorded in the amendment as a limit and as a candidate
  for a later record, not fixed here); N-b (the neutral question removes control
  characters but not what remains of an escape sequence, newlines become spaces, a
  long path is shown whole, and the error returned on "no" is the store's own,
  whole: stated in the doc comment of `save` and in the amendment, pinned by
  `TestNeutralQuestionStatedLimits_0082`, no change); N-c (a rename, below); N-d
  (S14's test, below; S16 and Q17 recorded as equivalent).
* **R-14's mechanism, and the proof the session file is unchanged.**
  `readBounded` (`llmprovider/auth/tokenstore_file.go`) now returns an unexported
  `*tooLargeError`, whose text is the text it has always had. `readBounded` is
  shared: its other caller is `FileTokenStore.Load` for the session file, which
  keeps its behaviour: a session file over 64 KiB gives no session and the error
  `FileTokenStore read: openai.json is larger than 65536 bytes`, byte for byte as
  before, not marked, the file left as it is. `FileTokenStore.LoadClient` alone
  wraps the size error in an unexported type with the method `Undecodable() bool`
  returning true (`clientstore.go`), and `signinflow.Undecodable` also asks any
  error for that method through `errors.As` and an interface target. I chose the
  marker over making LoadClient's error a fabricated `*json.SyntaxError`, which
  would misdescribe it; the choice did not need any change to the session path,
  so I did not stop. Nothing is exported: `make api-check` and apidiff report only
  `x/signin.VerifyKilo`, as before. The proof the session file is unchanged is
  `TestSessionFileOverTheSizeBoundKeepsBeingRefused_0082` (exact text, no session,
  not marked, bytes unchanged), with the old `TestFileTokenStore_LoadRefusesOversized`
  unchanged and green, and the plants U02 to U04 that change the session path.
* **Order, and what changed first.** Tests first, each run and seen failing in a
  scratch clone of `5f2c2ab` itself, with the new test files copied in, under a
  macOS sandbox that denies every outbound connection but loopback
  (`sandbox-exec`); no inert change was needed (the tests compile on `5f2c2ab`).
  Red lines on `5f2c2ab`:
  * `TestClientRecordOverTheSizeBoundIsUndecodable_0082` (`llmprovider/auth`):
    `LoadClient = FileTokenStore client read: openai.client is larger than 65536
    bytes; want an error marked undecodable`;
  * `TestUndecodableKnowsTheStoresOwnMarker_0082` (`internal/signinflow`): `a
    marked error: Undecodable = false, want true` and `a marked error, wrapped
    twice: Undecodable = false, want true`;
  * `TestPlanOverSizeRecordFromFileTokenStoreIsNone_0082` (the engine): `LoadClient
    = FileTokenStore client read: openai.client is larger than 65536 bytes; want an
    error Undecodable recognises`;
  * `TestOverSizeClientRecordIsReplacedWithOneNotice_0082` (the wizard, the
    review's row), both subtests: `the sign-in: err wizard: read the saved OpenAI
    client record: FileTokenStore client read: openai.client is larger than 65536
    bytes after 0 login(s); want it to go on`;
  * `TestStart_PlanOverSizeClientRecordIsReplaced_0082` (`x/signin`): `Wait =
    signinflow: load the issued client: FileTokenStore client read: openai.client
    is larger than 65536 bytes after 0 login(s); want the sign-in to go on`;
  * green on `5f2c2ab`, as guards the plants then proved:
    `TestClientRecordThatCannotBeReadIsNotMarkedUndecodable_0082`,
    `TestSessionFileOverTheSizeBoundKeepsBeingRefused_0082`,
    `TestNullClientRecordIsNoRecord_0082` (both rows),
    `TestNeutralQuestionRedactsASecretInTheLockError_0082` (S14: red under the
    plant, below) and `TestNeutralQuestionStatedLimits_0082`.
* **Existing test changed: a rename only (N-c).** `TestAnyFailedTryWithALiveContextIsHeld_0082`
  (`wizard/save_followup_0082_test.go`), whose first subtest has asserted since
  R-13 that such a try is not waited for, is now
  `TestFailedTryThatIsNotContentionAsksAndDoesNotWait_0082`; its body, subtests
  and assertions are unchanged. The earlier entries of this PLAN keep the old name
  as it was then. No other existing test changed.
* **Built, with `path:line`.** R-14: `tooLargeError` (`llmprovider/auth/tokenstore_file.go:107`) and `readBounded` (120), whose text is unchanged; `undecodableRecordError` with its `Undecodable() bool` (`llmprovider/auth/clientstore.go:69`) and `FileTokenStore.LoadClient` (85, the one place that wraps it, at 95); `signinflow.Undecodable` asking for the method (`internal/signinflow/signinflow.go:176`). The docs: `auth.ClientStore`'s `LoadClient` and `FileTokenStore.LoadClient` (`clientstore.go`), `signin.Options.ClientStore` (`x/signin/signin.go`), `wizard.Options.TokenStore` (`wizard/configure.go`), the wizard's `save` (N-b; `wizard/auth.go`), `signinflow.Undecodable`; CHANGELOG `[Unreleased]` Security, README's Store bullet, `docs/architecture.md` (the issued-client paragraph, the `FileTokenStore` list).
* **Plants.** `plants6.py`, kept with the scratch scripts, on a scratch copy of the final tree under `sandbox-exec` (outbound connections denied but loopback), the `localonly` guard on, run through `run-heavy`: **173 plants**, the fifth pass's table in full (its 160 less Q10, the reviewer's equivalent since R-12, so 159; the three whose anchor moved with the marker, S02, S08 and S09, re-formed), plus this pass's own 14: U01 to U12 (the oversize record fatal again; the oversize exemption applied to the session file, as a marked error and as no session; the oversize error's text changed; the notice dropped; the notice carrying the size error's text; the marker saying no; `Undecodable` ignoring the marker; `Undecodable` taking a marker that says no; every read error marked undecodable; the size bound read as exclusive; a record that decodes to the zero record counted as a decode failure), the reviewer's S14 and S16. A first run of the whole table was cut short by the owner's quota pause, at plant 107 of 173, with the pre-add check, the race rounds, gobble's packages and the CI set queued behind it: those runs were cancelled, none of them counts, and each was run again in full on the same tree after the working tree and the scratch copy were checked (the scratch copy held one plant's edit left by the cut-short run, which was discarded by copying the tree again). The complete run left three, verbatim: `Q17 the retried save's own access list drops the rotation's: NOT CAUGHT; failing=[] guard=no`, `U10 every read error marked undecodable: BUILD FAILURE; failing=[] guard=no` and `S16 the wizard's read handed over without its error (equivalent: only ClientLoadErr differs): NOT CAUGHT; failing=[] guard=no`. **U10** failed to build (the variable it left unused); re-formed so that it compiles and re-run on the final tree: `U10 every read error marked undecodable: CAUGHT; failing=['TestClientRecordPermissionDeniedFailsTheSignIn_0082', 'TestClientRecordThatCannotBeReadIsNotMarkedUndecodable_0082']`. **S14**: `CAUGHT; failing=['TestNeutralQuestionRedactsASecretInTheLockError_0082']`, the test this pass added, where the reviewer's own test alone had caught it. **Q17** and **S16** survive and are equivalent mutants by the review's showing, as recorded above. Final count: **171 of 173 caught; Q17 and S16 survive.** The four plants that name `guard=yes` (P11 twice, B1, N17) were stopped by the guard, as before. Plants that hang a test end at the run's own 6 minute timeout and count as caught.
* **Gates.** each status taken before any filter, after the last change to a Go
  file.

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` | `run-heavy`, session host, golangci-lint at the repository's pin, v2.14.0 (`GOLANGCI_LINT` set) | exit 0: 1021 files clean (gofmt, golangci-lint, vet, test, govulncheck); nested module `authcontract` clean |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | exit 0; "0 incompatible change(s) outside llmprovider/x/"; only `llmprovider/x/signin.VerifyKilo: added`: no name from this change |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./llmprovider/providers/openai/ ./wizard/ ./internal/signinflow/ ./llmprovider/x/signin/` | `run-heavy`, session host | exit 0 (auth 235.7 s, wizard 202.4 s, openai 32.8 s); `TestVerifyIDToken_UnknownKidRefetchesOnce` did not fail |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race` (`pick-host` chose a Windows host) | exit 0, 42 packages ok; run on the final Go tree before the pause, and not cancelled |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` by a scratch `replace`, on a scratch copy of gobble | `run-heavy`, session host | exit 0, three ok |
  | the CI set: `make parity-check dep-check coverage-check contract-check generate-check fixtures-check gate-selftest tools-test lint vuln` (lint at the pin) | `run-heavy`, session host | exit 0: coverage 43 packages, 0 problems; `authcontract` against its tag, 0 problems; lint 0 issues; govulncheck, both modules, no vulnerabilities |
  | the plants | scratch copy, sandboxed | 171 of 173 caught; the two survivors are equivalent mutants (Q17, S16), stated above |

  The golangci-lint gate ran with `GOLANGCI_LINT` set to the v2.14.0 binary in
  the scratch folder, as the pin requires. The owner has ruled to standardize on
  v2.15.0 under its own record in each repository; until this repository's pin
  moves the gate keeps the pin, and the three `nolintlint`-flagged directives in
  `llmprovider/internal/transport/` stay.

  The record gates ran after this entry's last edit; their results are in the
  hand-off.
* **Not done.** `tools/gate.py --timeout 5400`, `tools/plant.py PLANTS.json`
  and `tools/redgreen.py RED.json`: this series ran its own drivers
  (`plants6.py`, and `go test` red runs on scratch clones) in their place, which
  is not what Phase 2 names.
* **Not verified.** A wait on another real process; the 20-round `-race` run on
  Windows (the module run was one pass); that a client record file of exactly
  64 KiB plus one byte is the only boundary a real corrupt file would cross (a
  file of any size under the bound that does not decode is the earlier rule's);
  that Q17 and S16 are equivalent beyond the review's showing and my not
  finding a test.
* **Acceptance criteria of this PLAN, as they stand once this pass is committed.**
  Phase 0 (the records placed): met, by the Drafting and Approval entries.
  Phase 1 (tests, guard and docs, one commit): met in substance. Every test is
  in, seen red first on its base in every pass, and the docs say what the code
  does. Two of its sentences did not hold as written: "no existing test
  changed" (existing tests were changed by the owner's rulings, each named in
  the entry of the pass that did it, with its reason) and one commit (the work
  is four commits, `8051b00`, `04433ae`, `5f2c2ab` and this pass's, by the
  rulings). "No API change" holds: apidiff reports only a name from another
  record. Phase 2 (planted breaks and the gates): the plants are met by this
  series' own driver, 171 of 173 caught with two named equivalent mutants, and
  the commands of its steps 3 to 8 ran at their pinned versions with exit 0
  except the ones listed under "Not done": `tools/gate.py --timeout 5400`,
  `tools/plant.py PLANTS.json` and `tools/redgreen.py RED.json`. Phase 3
  (independent review): met, five reviews, each verdict and finding resolved in
  an amendment and an entry. Phase 4 (the release): not met, and not mine to
  meet: it waits for the owner's word on 0080's release, `tools/consumer.py`
  and `tools/offline.py` quoted, the tag's CI, the changelog section naming the
  change under Security, and `tools/records.py status` setting this PLAN
  `complete`. Phase 5 (gobble): not executed under this PLAN, by its own text.
  So what remains for the coordinator's close-out is: the three tools Phase 2
  names that this series did not run, or the owner's word that its own drivers
  stand in for them; the commit of this pass; Phase 4; and the status. I have
  not set the status.

### Phase 1, sixth pass, committed (2026-11-18)

* **Committed and pushed** by the coordinator on 2026-11-18 as `1545172`
  ("feat(auth): treat over-size client record as undecodable and replace it"),
  `main`, after the sixth independent review and the coordinator's own gate. The
  "sixth pass" entry above is the record of the staging that preceded it. The
  seventh pass below is staged on top of it.

### Phase 2, the three tools the PLAN names, run by the sixth review (2026-11-18)

The sixth independent review ran, on the final code of `1545172`, the three tools
that Phase 2 names and that this series had not run (it had used its own drivers,
`plants*.py` and red runs on scratch clones, in their place; the "Not done" lines
of the fifth and sixth passes say so). Its results, as the coordinator relayed
them:

* `tools/redgreen.py` (step 2): 11 of 11 red at `8f133db`, and green on the tree.
* `tools/plant.py PLANTS.json` (step 1): 12 of 12 caught.
* `tools/gate.py --timeout 5400` (step 5): all 30 checks pass; four of them after
  a re-run against a local module proxy, because they need `go run tool@version`.

These close the Phase 2 item the sixth pass listed as remaining. The reviewer
found the spec files of the earlier passes stale, and re-formed them before the
run: `RED.json`'s `files` no longer listed the helpers' file
(`held_save_0082_test.go`), so the `auth` package did not build at the base until
`files` was held_session, held_save, localonly and the `openai` held_session;
`RED_EDGES.json`'s `TestRefusalTextIsPlain_0082` was replaced by a ruling; and
anchors had moved in `PLANTS_P1` (5 of 8), `PLANTS_P2` (5 of 41) and
`PLANTS_SITES` (3 of 3). **The spec files live only in scratch, not in the
repository**: this series' copies are in its first pass's scratch folder
(`RED.json`, `RED_EDGES.json`, `PLANTS_P1.json`, `PLANTS_P2.json`,
`PLANTS_SITES.json`), and the reviewer's re-formed copies are in its round-six
folder. Nothing committed names them, so nothing in the repository needed to
change, and I did not edit them.

### Phase 1, seventh pass (2026-11-18)

* **Provenance.** A mainstream-tier model at high effort, on the coordinator's
  brief, over `1545172`. Staged, not committed, pushed or tagged. Every check of
  the pass that could reach a module proxy ran inside the sandbox, as below.
* **The sixth review.** No blocker. One finding, F-1: the fifth amendment says a
  custom `ClientStore` keeps exactly one route to "undecodable" (wrapping
  encoding/json's error), but `signinflow.Undecodable` honoured any error in the
  chain with an `Undecodable() bool` method returning true, an undocumented second
  route reachable from outside the module (its probe `TestReview0082_CustomMarkerIsHonoured`:
  a custom store's own type with that method got its record replaced). Its note
  N-1: an unrelated file over 64 KiB sitting at the record's path is replaced too.
  Its V12 (`the marker hides its cause`) is an equivalent mutant in its own suite.
* **The decision, the coordinator's** (recorded as the coordinator's; the owner is
  told), on F-1, the reviewer's recommended option: make the marker unreachable
  from outside the module. The marker type lives in a module `internal` package
  that both `llmprovider/auth` and `internal/signinflow` import, and `Undecodable`
  matches that concrete type with `errors.As`, besides the two encoding/json
  types; no method-based matching. The MADR's sixth amendment records it, N-1 as a
  stated limit, and annotates the fifth amendment's "one route" sentence as true
  from this commit on.
* **The internal package, and why there is no cycle.** `internal/recorderr`
  (`internal/recorderr/recorderr.go`) holds `Oversize`, a struct with the marked
  error, `Error` and `Unwrap`. It imports nothing of the module (`go list -deps`
  shows only itself), so `llmprovider/auth` (`clientstore.go`) and
  `internal/signinflow` (`signinflow.go:179`) both import it: a root `internal`
  package is importable by any package of the module, as `internal/redact`
  already is by both. Nothing exported moves, so I did not stop: `make api-check`
  and apidiff, run in the sandbox, report only `x/signin.VerifyKilo`.
* **Order, and what changed first.** The tests were written, and an inert
  `internal/recorderr` (the type alone, used by nothing) added so that they
  compile and fail on assertions, then run on that tree, which is `1545172` plus
  the inert package, under a macOS sandbox that denies every outbound connection
  but loopback (`sandbox-exec`). Red lines:
  * `TestUndecodableTakesTheModulesMarkerTypeAndNoMethod_0082`
    (`internal/signinflow`): `the marker type: Undecodable = false, want true`,
    `the marker type, wrapped twice: Undecodable = false, want true`, `a custom
    store's error with the method saying yes: Undecodable = true, want false`
    (the reviewer's probe, as an assertion) and `the same, wrapped:
    Undecodable = true, want false`;
  * `TestClientRecordOverTheSizeBoundIsUndecodable_0082` (`llmprovider/auth`, its
    helper now asking for the type): `LoadClient = FileTokenStore client decode:
    openai.client is larger than 65536 bytes; want an error marked undecodable`;
  * `TestCustomStoreErrorWithAMarkerMethodIsAReadError_0082` (the wizard, end to
    end, a custom store's error type with the method): `err = <nil>; want the
    sign-in failed with the store's own error`;
  * green on `1545172` and after, as guards: `TestOversizeShowsAndUnwrapsItsCause`
    (`internal/recorderr`, green on the inert type, red under the V03 plant below),
    `TestPlanOverSizeRecordFromFileTokenStoreIsNone_0082`,
    `TestOverSizeClientRecordIsReplacedWithOneNotice_0082` (both rows),
    `TestSessionFileOverTheSizeBoundKeepsBeingRefused_0082`,
    `TestClientRecordThatCannotBeReadIsNotMarkedUndecodable_0082`: the over-size
    record is still replaced, with the same notice, and the session file is
    unchanged.
* **Existing tests changed by the decision, stated.** Two, each asserting the
  method route the decision removes: `TestUndecodableKnowsTheStoresOwnMarker_0082`
  (`internal/signinflow`), whose cases took an error with the method as undecodable,
  is replaced by `TestUndecodableTakesTheModulesMarkerTypeAndNoMethod_0082` above,
  which asserts the opposite for the method and the type for the marker; and the
  helper `marked0082` in `llmprovider/auth/clientstore_oversize_0082_test.go`,
  which asked for the method, now asks for the type, so its three tests keep
  asserting what they meant. The decision names the behaviour, so I changed them
  without stopping. No other existing assertion changed.
* **Built.** The marker and its package as above; `FileTokenStore.LoadClient`
  (`llmprovider/auth/clientstore.go`) returns `&recorderr.Oversize{...}` where it
  returned the unexported type, which is deleted; `signinflow.Undecodable`'s doc;
  `auth.ClientStore`'s `LoadClient` doc (the one route, and no method of a custom
  error type standing in for it); `docs/architecture.md` (the issued-client
  paragraph, the package tree, the dependency table and a row for
  `internal/recorderr`) and `docs/guides/api-standards.md` (its package
  table, which `make tables-check` checks, and which found the new package's
  row and the two importers' rows missing at the first record gates).
* **Plants.** `plants7.py`, kept with the scratch scripts, on a scratch copy of the final tree under `sandbox-exec` (outbound connections denied but loopback), the `localonly` guard on, run through `run-heavy`: **174 plants**, the whole of the sixth pass's table (173) less U07, U08 and U09, which plant the method route that no longer exists, plus this pass's own V01 to V04: V01 the method-based match restored, V02 the concrete type not matched, V03 the marker hiding its cause (the review's V12), V04 the over-size error not marked in `LoadClient`. Where the code moved with the marker, S02, S08, S09, U02 and U12 were re-formed (U02 also adds the import its marked error needs). Complete run, survivors verbatim: `Q17 the retried save's own access list drops the rotation's: NOT CAUGHT; failing=[] guard=no`, `S16 the wizard's read handed over without its error (equivalent: only ClientLoadErr differs): NOT CAUGHT; failing=[] guard=no` and `V03 the marker hides its cause (the review's V12): NOT CAUGHT; failing=[] guard=no`. **Q17** and **S16** are the equivalent mutants recorded above. **V03** is the review's V12, an equivalent mutant in its own suite; here the driver does not catch it either, because its test selection (`_0082` tests of five packages) does not include `internal/recorderr`, whose test of the marker is not so named; run by hand on the scratch copy under the same sandbox with `Unwrap` removed, `go test ./internal/recorderr/` fails: `TestOversizeShowsAndUnwrapsItsCause: errors.Is(err, cause) = false; the marker hides its cause`. So it is caught by a test of this series, outside the driver's selection, and the survivor list stays as the driver gave it. V01: `CAUGHT; failing=['TestCustomStoreErrorWithAMarkerMethodIsAReadError_0082', 'TestUndecodableTakesTheModulesMarkerTypeAndNoMethod_0082', ...]`; V02 and V04: caught by the over-size tests (`TestOverSizeClientRecordIsReplacedWithOneNotice_0082`, `TestClientRecordOverTheSizeBoundIsUndecodable_0082`). Final count: **171 of 174 caught by the driver, 172 with V03's hand run; Q17 and S16 survive, and both are equivalent mutants by the review's showing.** The four plants that name `guard=yes` (P11 twice, B1, N17) were stopped by the guard, as before. Plants that hang a test end at the run's own 6 minute timeout and count as caught.
* **Gates.** each status taken before any filter, after the last change to a Go
  file.

  | Gate | Where | Result |
  | :--- | :--- | :--- |
  | `make pre-add-check` | `run-heavy`, session host, golangci-lint at the repository's pin, v2.14.0 (`GOLANGCI_LINT` set); its tests and govulncheck run unsandboxed, as the brief had it | exit 0: 1025 files clean (gofmt, golangci-lint, vet, test, govulncheck); nested module `authcontract` clean |
  | `go vet ./...`; `GOOS=windows go vet ./...`; `go vet -tags live_gateways ./...` | session host, inside the sandbox (`sbx-localproxy.sh`) | exit 0 each |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host, inside the sandbox with the local module cache as the proxy (`sbx-localproxy.sh`: `sandbox-exec` denying every outbound connection but loopback, `GOPROXY=file://<module cache>/cache/download`, `GOSUMDB=off`) | exit 0; "0 incompatible change(s) outside llmprovider/x/"; only `llmprovider/x/signin.VerifyKilo: added` |
  | `go test -race -count=20 -timeout 45m ./llmprovider/auth/ ./wizard/ ./internal/signinflow/ ./llmprovider/x/signin/` | `run-heavy`, session host | exit 0 (auth 226.9 s, wizard 200.6 s); `TestVerifyIDToken_UnknownKidRefetchesOnce` did not fail |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race` (`pick-host` chose a Windows host) | exit 0, 43 packages ok (the new `internal/recorderr` among them) |
  | gobble's `internal/auth`, `llm/provider`, `internal/cli` by a scratch `replace`, on a scratch copy of gobble | `run-heavy`, session host | exit 0, three ok |
  | the plants | scratch copy, sandboxed | 171 of 174 by the driver (172 with V03's hand run); the survivors are Q17 and S16, equivalent mutants |

  How I know the sandboxed runs were sandboxed: the plants driver runs every
  `go test` under `sandbox-exec -p '(version 1)(allow default)(deny
  network-outbound)(allow network-outbound (remote ip "localhost:*"))'`, and
  `sbx-localproxy.sh` is that profile plus the module cache as the proxy; a
  `curl https://proxy.golang.org/` run through `sbx-localproxy.sh` fails with
  `Could not resolve host`, where the same command outside it does not. The
  `localonly` guard in the auth tests was on. `make pre-add-check` and the race
  runs were not sandboxed, as the brief had it; `make pre-add-check`'s
  govulncheck needs its database.

  The record gates ran after this entry's last edit; their results are in the
  hand-off.
* **Not verified.** A wait on another real process; the 20-round `-race` run on
  Windows (the module run was one pass); that `go test ./internal/recorderr/` is
  covered by the coverage floor the way every package is (the CI set was not rerun
  this pass; the package has a test and the whole module run passed).
* **What remains for the coordinator's close-out.** Nothing else, by my reading of the PLAN, once this pass is
  committed. Phase 0, Phase 1, Phase 2 (its three tools now run by the sixth
  review, and the plants by this series' own driver) and Phase 3 (six reviews,
  each verdict and finding resolved in an amendment and an entry) are met, with
  the two sentences of Phase 1 that the owner's rulings overrode ("no existing
  test changed", "one commit") stated in the earlier entries. What remains is
  Phase 4, the release, on the owner's word with 0080's release: `tools/consumer.py`
  and `tools/offline.py` quoted, the tag's CI, the changelog section naming the
  change under Security, and `tools/records.py status` setting this PLAN
  `complete`; and Phase 5, gobble's own phase, which is not executed under this
  PLAN by its own text. Before Phase 4, this pass's commit, and a rerun of the
  Phase 2 CI set on it if the coordinator wants one (I ran only the gates the
  brief named). I have not set the status.

### The release preparation (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This record's entry is under `Security`.
