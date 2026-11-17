---
status: proposed
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
