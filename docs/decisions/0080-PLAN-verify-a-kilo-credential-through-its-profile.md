---
status: in-progress
date: 2026-11-16
associated-madr: "0080-MADR-verify-a-kilo-credential-through-its-profile.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Verifying a Kilo Credential Against Kilo's Profile at Sign-In When No Organization Is Set

Associated MADR: [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)

## Goal

After this PLAN, in the release it shares with the typed "tool schema
refused" error (MADR D11):

* the wizard's credential step, under `Options.Discover`, reads Kilo's
  profile for a Kilo credential used with no organization, and treats its
  401 as the listing's refusal (MADR D1, D3);
* an outage of the profile warns and saves (D6); an organization's listing
  stays the check where one is set (D4);
* a caller gets the same rule from one function (D10), and gobble adopts it
  by its own records (D12).

~~The MADR is `proposed`, and so is this PLAN. Nothing below Phase 0 starts
before the owner answers the MADR's Q1 to Q7 and approves this PLAN. The
steps are written to the MADR's recommendations; a different answer changes
the steps it names, by a dated note here before the phase starts.~~
*(2026-11-16: the owner answered Q1 to Q7, each as recommended, so the
steps stand as written, with their alternatives removed below; the MADR is
`accepted` and this PLAN `in-progress`. The approval, the owner's ruling, relayed by the coordinator: "Approve and
run through phase 4"; see "Rules for every phase" and the Execution
record's "Phase 0".)*

## Prerequisites

* For Phase 5: the typed "tool schema refused" error's record is approved
  and its code is committed, or the owner rules that this one ships alone.
* For every phase after Phase 1: Phase 1's evidence is recorded and does not
  stop the PLAN (its acceptance).

## Scope

### In scope

* `internal/signinflow`: the Kilo check (D10), one file and its test.
* `wizard/configure.go` and `wizard/auth.go`: the check in the credential
  step (D2, D3, D6), and their doc comments (`ConfigureLLM`,
  `ResolveCredential`, `credentialStep`, `resolveKiloDevice`,
  `resolveKiloDeviceKey`, `Options.KiloDeviceKey`).
* `llmprovider/x/signin`: `VerifyKilo` (D10; Q2's answer), its doc and an
  example.
* `llmprovider/auth/live_kilo_profile_0080_test.go`: Phase 1's measurement
  test, live-tagged.
* Docs: `README.md`, `docs/architecture.md`, `docs/guides/api-standards.md`
  (R2's `x/signin` row), `CHANGELOG.md`; dated notes in 0056-MADR D2,
  0017-MADR D2, 0059-MADR W2 and 0028-MADR D-H10 (MADR, "Records this
  decision amends").

### Exported names (R55)

Inside `llmprovider/x/`, outside the v1 promise (R4, R48):

| Name | Call site that needs it |
| :--- | :--- |
| `signin.VerifyKilo` | gobble-cli's ACP check, `checkListing` (`llm/provider/signinrun.go:209`, MADR G1), by a gobble-cli record; this module's example `ExampleVerifyKilo` |

None outside `x/` (Q2's answer, "An experimental `signin.VerifyKilo`").
*(2026-11-16: the alternatives this paragraph named for Q2 (b) and (c) are
removed; neither was chosen.)*

### Out of scope

* The typed "tool schema refused" error: its own record, sharing only the
  release.
* Changing the listing (`catalog`), the classification table, or any other
  provider's check (MADR V5, D5).
* gobble's changes (Phase 6 names them; gobble's records make them).
* `kiloendpoint`'s stale line citations of Kilo's source (MADR, "Found
  beyond the question asked").

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
  *(2026-11-16, the owner's ruling, relayed by the coordinator: "Approve and run through phase 4". Phase 0 is
  committed under the standing approval. Phase 1's test is written and
  committed, then the owner runs its live steps; after that, Phases 2, 3
  and 4 run without stopping for approval, each reviewed and committed by
  the coordinator. Phase 5, the release, waits for the owner's word. A
  stop condition below is a deviation, not a phase boundary: it still
  stops the run and goes to the owner.)*
  *(Added 2026-11-17, the owner's ruling, relayed by the coordinator: "One
  commit: the tests with phase 3's code". As put to the owner: the tests go
  into the tree when Phase 3's implementation makes them pass, and the two
  land together as one commit; "red" stays proven on a scratch clone, as
  this section describes, and is in the record; `main` never carries a
  failing build; a dated deviation note merges the Phase 2 and Phase 3
  commits (the Execution record's "Phase 3 (with Phase 2's tests)").
  Options not chosen: the tests behind a temporary build tag; commit them
  red on `main`.)*
* **Red, then green.** A new test that calls only existing API is seen red at
  `ceb489e` with `python3 -B tools/redgreen.py SPEC.json --expect fail --rev
  ceb489e`, and green on the phase's tree with `--expect pass`. A test that
  calls a new name, and a test whose red run cannot show each assertion, is
  seen to fail on each planted break with `python3 -B tools/plant.py
  SPEC.json`, on its scratch clone, never in the tree. The execution record
  quotes each failure line. A break no test catches is a deviation.
* **Heavy runs queue.** Every `-race` run and the gate run through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` on
  the session host, or through the skill's `scripts/run-on-host` on the host
  `~/.agents/skills/available-dev-hosts/scripts/pick-host` prints. No `-race`
  run goes to a WSL distro.
* **Pre-add.** `make pre-add-check FILES="..."` on every changed Go file.
* **No outside host in a test** (0065-MADR): every new offline test serves
  its own profile and listing, by `httptest` or a `RoundTripper`.
* **Nothing shows a key.** No test, log, record or commit message holds a
  real key; a record holds no made-up one either, and an offline test only
  its own fixed placeholder; a live run prints status and counts only.
* **Stop conditions,** each a stop and a prompt to the owner:
  * Phase 1 finds a valid key kind the profile refuses;
  * an existing test, golden or fixture needs a change; the six to watch are
    `TestKept_KiloRefusedLeadsToSignIn_0066`,
    `TestListingWarnings_NoProviderText_0056`,
    `TestWizardOwnRequestsNamed_0079`,
    `TestConfigureLLM_KiloListsTheChosenOrganization`,
    `TestKiloDeviceKey_Refused_0056` and
    `TestSession_KiloRefusedNotSaved_0059` (P8 to P13);
  * an error's text changes, other than the new warning;
  * `tools/apidiff.py v1.31.0` lists an addition outside `llmprovider/x/`;
  * a Kilo request is sent where the MADR says none is: with an
    organization, without Discover, or for another provider.

## Implementation Steps

### Phase 0: records

Model: lightweight, low. *(2026-11-16: executed by the strongest-tier model
that drafted the pair, at high effort, because it held the context; the
coordinator's choice. See the Execution record's "Phase 0".)*

1. This PLAN, the MADR and their index rows (`tools/records.py add`, done
   with the draft) are committed ~~on the owner's word~~ by the coordinator,
   under the standing approval (the approval's ruling).
2. The owner's answers to Q1 to Q7 go into the MADR, each as "Answered
   2026-MM-DD: (x)", and any step below they change gets a dated note.
   `tools/records.py status` sets the MADR `accepted` and this PLAN
   `in-progress`, each with its index row, when the owner says so.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --markdown`, then stage.

Acceptance: `make records-check links-check` pass;
`python3 -B tools/claims.py` passes on both records; no source changed.

### Phase 1: which key kinds the profile accepts (ruled first step)

Model: mainstream, low, for the test and the record; the owner runs the
live steps.

**What made-up tokens can and cannot show.** Done in the MADR's probes
(L1 to L25), and repeated here with the script below: the profile refuses no
credential, a made-up one and a JWT-shaped one, with 401, and tells the first
from the others only by its body; an organization's listing refuses them at
all three URLs; the public listing refuses nothing. They cannot show what a
valid key gets: that needs the owner's own keys.

1. **The made-up probes again**, from a scratch script outside the tree
   (Python, the standard library, a fresh made-up token per run, bodies
   hashed and deleted): the MADR's L1 to L18. Record the table here. A
   change from the MADR's table is a deviation.
2. **The measurement test.** `llmprovider/auth/live_kilo_profile_0080_test.go`,
   build tag `live_gateways`, `TestLive_KiloProfile_0080`, skipped unless
   `LLMPROVIDER_LIVE_0080=1` (AGENTS.md's rule for a record's variable, P7;
   `tables-check` accepts it because this record names it, P18). Subtests,
   each skipped without its variable:
   * `DashboardKey`: the key in `KILO_API_KEY` (a key from "Your Profile" on
     the personal account, MADR K14); `auth.KiloProfile` must return no
     error. It logs "accepted", the number of organizations and
     `HasPersonalAccount`, nothing else.
   * `OrganizationToken`: the token in `LLMPROVIDER_LIVE_0080_ORG_KEY` and
     the organization id in `LLMPROVIDER_LIVE_0080_ORG`; it logs the
     profile's status class and the organization listing's
     (`catalog.List` with `catalog.WithKiloOrganization`), each "accepted"
     or the error's class and status.
     *(Note, 2026-11-16: the test calls `catalog.ListRaw`, not `catalog.List`.
     `List` degrades a refused listing to a fallback catalog and returns no
     error: on a stand-in, a 401 listing read "accepted" through it. A
     deviation recorded by the executing agent and confirmed by the
     coordinator on 2026-11-16 (the coordinator's, not the owner's).)*
   * `RevokedKey`: a key the owner has revoked or rotated, in
     `LLMPROVIDER_LIVE_0080_REVOKED_KEY`; the profile must refuse it with
     `ErrAuthFailure`; it logs the class and status.
     *(Note, 2026-11-16: a 403 reads "ErrNotPermitted", not "ErrAuthFailure",
     and fails this subtest, since a 403 is not a refusal (Q4); see the
     execution record's "Phase 1, the agent's part".)*
   On a failure it prints the error's sentinel and, for an `*APIError`, its
   status, never the error's text and never the key.
3. **The owner's run.** The owner runs it, never an agent, at the commit that
   adds the test, in a new terminal window that is closed afterwards. Before
   the owner runs it, the agent that wrote the test confirms by reading it
   that no `t.Log`, `t.Logf`, `t.Error`, `t.Errorf`, `t.Fatal`, `t.Fatalf` or
   `t.Skip` call is given a key, a variable holding one, or an error's text.
   *(Rewritten 2026-11-16 in Phase 0, at the coordinator's request, so that
   the commands are exact; the first draft said only "typed at a hidden
   prompt, such as `read -rs KILO_API_KEY`".)*

   Each `read -rs NAME` line waits, showing nothing, for one value: the owner
   types or pastes it and presses Enter. Nothing is echoed. The value lives
   only in that shell's environment and in the `go test` process the shell
   starts; it is never written to a file. The lines typed hold no key, so
   the shell's history holds none. The commands work as written in zsh and
   in bash. Nothing else is typed in that window: no `echo`, `printenv`,
   `env`, `set` or `set -x`.

   1. Open the checkout and confirm the commit:

      ```sh
      cd <the go-llmprovider-sdk checkout>
      git log -1 --format=%h
      ```

   2. The dashboard key (required): a key copied from "Your Profile" on the
      personal account at app.kilo.ai (MADR K14).

      ```sh
      read -rs KILO_API_KEY
      export KILO_API_KEY
      ```

   3. Only for a kind the owner has, before step 4 (each optional):

      ```sh
      read -rs LLMPROVIDER_LIVE_0080_ORG_KEY
      export LLMPROVIDER_LIVE_0080_ORG_KEY
      read -rs LLMPROVIDER_LIVE_0080_ORG
      export LLMPROVIDER_LIVE_0080_ORG
      read -rs LLMPROVIDER_LIVE_0080_REVOKED_KEY
      export LLMPROVIDER_LIVE_0080_REVOKED_KEY
      ```

      The first pair is an organization token and its organization's id,
      together or not at all; the last is a key the owner has revoked or
      rotated.

   4. The profile test:

      ```sh
      LLMPROVIDER_LIVE_0080=1 go test -tags live_gateways ./llmprovider/auth -run '^TestLive_KiloProfile_0080$' -count=1 -v
      ```

   5. Remove the values from the shell:

      ```sh
      unset KILO_API_KEY LLMPROVIDER_LIVE_0080_ORG_KEY LLMPROVIDER_LIVE_0080_ORG LLMPROVIDER_LIVE_0080_REVOKED_KEY
      ```

   6. The device sign-in (required; P6, P14). It logs a URL and a code; the
      owner opens the URL, enters the code and approves within five minutes.
      It then reads the profile with the new token (MADR C34), logging
      counts only, and generates once on a free model. It is the
      device-session and device-key case: the same token.

      ```sh
      LLMPROVIDER_LIVE_DEVICE_LOGIN=1 go test -tags live_gateways ./llmprovider -run '^TestLive_KiloDeviceLogin$' -count=1 -v
      ```

   7. Close the terminal window.

   **What the owner sends back,** to the coordinator, for this record: from
   each run, every `--- PASS:`, `--- FAIL:` and `--- SKIP:` line, with its
   test or subtest name; each line the tests log with a result
   ("accepted", the organization count and personal account, or a sentinel
   and an HTTP status); and the last line (`ok` or `FAIL`). Not the device
   code line. Never a key: if any line shows anything that looks like one,
   the owner sends nothing, says so, and the PLAN stops.

   **A kind with no key** *(proposed 2026-11-16 in Phase 0; the owner's
   answers do not settle it, so it is Q8 below)*: its subtest skips; this
   record writes "not run: no such key" for it, and the MADR's "Not
   verified" keeps it, named. The dashboard key and the device sign-in are
   required: without either, the PLAN does not go past Phase 1, since they
   are the kinds the wizard's paths P1 to P7 receive. Without an organization token
   or a revoked key, the PLAN proceeds.

   *(Answered 2026-11-16, the owner's ruling, relayed by the coordinator:
   "The dashboard key and the device sign-in" (Q8). Those two are the kinds
   the wizard's paths actually receive; the organization token and the
   revoked key run only if the owner has them to hand, otherwise each is
   recorded "not run: no such key" and stays named under "Not verified".
   Not chosen: whatever the owner can run, none required; all four.)*

   * **Q8. Which Phase 1 kinds must be run before Phase 2 starts?**
     * **(a)** The dashboard key and the device sign-in; the organization
       token and the revoked key only if the owner has them, recorded "not
       run: no such key" otherwise (recommended: the two required kinds are
       the ones the wizard's paths P1 to P7 receive; an organization token
       expires in 15 minutes (MADR K12), so it is not a key a person keeps or
       types, and its risk stays named in the MADR's "Not verified").
     * **(b)** Whatever the owner can run, none required; a kind not run is
       recorded and the PLAN proceeds.
     * **(c)** All four; the PLAN waits until the owner has each kind.

     *(Answered 2026-11-16: (a), "The dashboard key and the device sign-in";
     see the paragraph above.)*

4. **The record.** The execution record gets each kind's result; the
   MADR's "Not verified" loses each kind settled; a kind not run (for want
   of an organization token, say) stays there, named.

Files: `llmprovider/auth/live_kilo_profile_0080_test.go`; this PLAN; the
MADR.

Verification: `go vet -tags live_gateways ./llmprovider/auth/`; `make
pre-add-check FILES="llmprovider/auth/live_kilo_profile_0080_test.go"`;
`make tables-check`; the owner's two runs, quoted.

Acceptance: the dashboard key and the device token are each "accepted"; a
revoked key, if run, is refused with 401; an organization token, if run, is
accepted by the profile and by its organization's listing. **If any valid
kind is refused, the PLAN stops**: the MADR's D1 would refuse a working key,
and the owner decides (narrow the check to the kinds accepted, change the
check, or stop). A kind with no key follows the rule in step 3.
Docs: none (a test only); the variables are named in this record.

### Phase 2: the tests, red

Model: mainstream, medium.

Files: `wizard/kilo_verify_0080_test.go`;
`internal/signinflow/kilo_0080_test.go`;
`llmprovider/x/signin/verify_0080_test.go`. The wizard's tests stub the
profile through its existing seam, `kiloProfile` (P15), or serve
`/api/profile` from the fake client; each lists through a cache of its own
(`testCache(t)`, 0058-MADR).

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestKiloVerify_TypedKeyRefused_0080` | `ResolveCredential` with Discover, Kilo's API key typed, the public listing 200, the profile 401 for the first key and 200 for the second: one "refused this key; enter another" warning by class and status, a second key asked, the second returned | red at `ceb489e`: the first key is returned |
| `TestKiloVerify_EnvironmentAndSavedKeyRefused_0080` | `KILO_API_KEY` under `AllowEnv`, then `Existing`'s key: the profile's 401 ends the run naming the variable, then "the saved key"; each matches `ErrAuthFailure`; nothing is asked again | red at `ceb489e` |
| `TestKiloVerify_DeviceKeyRefused_0080` | `KiloDeviceKey`, Discover, the profile 401: "refused the device sign-in; sign in again" (MADR C20), `ErrAuthFailure`; one profile read; no organization question | red at `ceb489e` |
| `TestKiloVerify_DeviceSessionRefused_0080` | the device session, Discover, the profile 401: the store is as it was (W2), the error is "refused the sign-in; sign in again", one profile read, not two | red at `ceb489e` (today: a warning and a save) |
| `TestKiloVerify_KeptSessionRefused_0080` | a kept Kilo session, no organization, Discover, the profile 401: the notice and the sign-in menu, once (0066-MADR V7) | red at `ceb489e` |
| `TestKiloVerify_OutagesWarn_0080` | the profile 403, 404, 500, 503, a body that does not decode, a refused connection (`internal/nettest`) and a stall past `DiscoverLimit`: one warning each, by class and status, no service text; the key returned or the session saved | red at `ceb489e` for the key paths (no read, no warning); planted: the 403 treated as a refusal; the stall bounded by 30 s instead of the limit |
| `TestKiloVerify_OrganizationReadsNoProfile_0080` | a kept session with `Existing.Organization`: no profile read; the organization listing's 401 still refuses | green at `ceb489e`; planted: the check ignores the organization |
| `TestKiloVerify_NoDiscoverNoRead_0080` | without Discover, each of the typed key, the device key and the device session: no profile read beyond the session's menu read, and that one's 401 still a warning | green at `ceb489e`; planted: the check runs without Discover |
| `TestKiloVerify_OtherProvidersReadNoProfile_0080` | OpenCode Zen and Hugging Face keys under Discover: no request to `/api/profile` | green at `ceb489e`; planted: the check runs for every provider |
| `TestKiloVerify_RequestNamed_0080` | the profile request carries `Options.Identity`'s User-Agent and the endpoint in use (0079-MADR D5; 0020-MADR F20) | red at `ceb489e` for a typed key (no request); planted: the identity left off |
| `TestVerifyKilo_Kinds_0080` (`internal/signinflow`) | 200 nil; 401 an `*APIError` matching `ErrAuthFailure`; 403, 5xx, offline and a decode failure not refused, with the cause; an organization: no request | new name: planted, each case's branch flipped |
| `TestVerifyKilo_0080` (`x/signin`) | the exported function's kinds, as its doc names them (R52): nil, `ErrAuthFailure` with `ErrorKindOf` `auth`, ctx's error | new name: planted, the 401 returned as nil |

Acceptance: each test is seen as the table says, quoted in the execution
record. Docs: none.

*(Added 2026-11-17, the owner's ruling, relayed by the coordinator: "One
commit: the tests with phase 3's code". The three test files are not
committed on their own: they go into the tree with Phase 3's implementation,
which makes them pass, in one commit. Their red runs stay as the execution
record's "Phase 2, the tests" has them, on a scratch clone at `ceb489e`.
Options not chosen: the tests behind a temporary build tag; commit them red
on `main`. See Phase 3's note and the "Phase 3 (with Phase 2's tests)"
entry.)*

### Phase 3: the check, and the docs

Model: mainstream, high (the credential loop is subtle: kept sessions, the
typed-key count, the device session's existing read).

1. `internal/signinflow`: `VerifyKilo(ctx, read, token, organization,
   opts...) (refused, unverified error)`: no request when `organization` is
   not empty; `read` (the wizard passes `kiloProfile`, P15; `x/signin`
   passes `auth.KiloProfile`) under ctx; `refusedCredential`'s rule decides
   which of the two returns the error (MADR D1, D5, D9).
2. `wizard/configure.go`: in the credential step's check, under Discover and
   within `DiscoverLimit`, a Kilo credential with no organization is
   verified before the listing; a refusal is returned as the listing's is,
   so the loop handles it unchanged; an unverified result gives the D6
   warning once. The request takes the options `resolveKiloDevice` builds
   (identity, base URL, client, `ProviderOptions`).
3. `wizard/auth.go`: `resolveKiloDevice` keeps its profile read and carries
   its result on the credential, so the check reuses it (D3); with Discover
   its 401 is a refusal, without it today's warning. `resolveKiloDeviceKey`
   reads nothing itself; the step's check reads the profile once.
4. `llmprovider/x/signin`: `VerifyKilo(ctx, key, organization string, opts
   ...llmprovider.Option) error`, as MADR D10; `ExampleVerifyKilo`, offline,
   over `httptest`, with `// Output:` (R50).
5. **Docs**, in the same commit:
   * doc comments: `ConfigureLLM`, `ResolveCredential` (C19's sentence names
     Kilo's check), `credentialStep`, `Options.KiloDeviceKey` (one profile
     read under Discover), `resolveKiloDevice`, `resolveKiloDeviceKey`,
     `auth.KiloProfile` (that the wizard and `signin.VerifyKilo` use it to
     verify);
   * `README.md`: the wizard's listing check and the `x/signin` section
     (P5);
   * `docs/architecture.md`: the wizard's Kilo paragraph (P3) and the Kilo
     device login (P4);
   * `docs/guides/api-standards.md`: R2's `x/signin` row names `VerifyKilo`
     (P1);
   * `CHANGELOG.md` `[Unreleased]` (P16): **Changed** "the wizard verifies a
     Kilo credential against Kilo's profile when no organization is set, and
     refuses it on a 401"; **Added** "`signin.VerifyKilo` (experimental)";
   * dated notes in 0056-MADR D2, 0017-MADR D2, 0059-MADR W2 and 0028-MADR
     D-H10, each one line pointing at 0080-MADR.
6. Before staging: every existing wizard test, the six the stop condition
   names (P8 to P13) among them, on a scratch clone of the phase's tree;
   any that fails or needs a change stops the phase.

Verification: Phase 2's tests green; `make pre-add-check FILES="..."`;
`go test -count=1 ./internal/signinflow/ ./llmprovider/x/signin/ ./wizard/`
(no `-race`, on the session host); `make api-check`; `python3 -B
tools/apidiff.py v1.31.0`; `make docs-check tables-check changelog-check
links-check records-check`.

Acceptance: every test of Phase 2 passes; no existing test changed; the API
check reports no change under the v1 promise; the docs say what the code
does.

*(Added 2026-11-17, the owner's ruling, relayed by the coordinator: "One
commit: the tests with phase 3's code". This phase's commit also holds
Phase 2's three test files, placed unchanged unless the real implementation
shows a test to be wrong; Phases 2 and 3 land as one commit, and `main`
never carries a failing build. Options not chosen: the tests behind a
temporary build tag; commit them red on `main`.)*

*(Added 2026-11-17, after an independent review; the owner's two rulings,
relayed by the coordinator, each quoted by its label. "Its own 10 s bound; a
stall is "cannot say"": `signin.VerifyKilo` bounds its read by the wizard's
default discovery limit itself and returns nil when that passes, and ctx's
error only when the caller's own ctx ended (MADR D10, Q5). "Treat it as
having an organisation: no profile read": a credential whose listing is an
organization's, by the explicit organization, a token's embedded URL or the
base URL's path, is checked by that listing and the profile is not read
(MADR D4); the helper is `internal/kiloendpoint` `Organization`, and that
package moved from `llmprovider/internal` to the module's own `internal`, so
that `internal/signinflow` may import it. Step 1's `VerifyKilo` is as written
and gains `CheckKiloOptions` and `DefaultCheckLimit` beside it. Neither ruling
adds an exported name. The "Phase 3 (with Phase 2's tests)" entry's "Review and
fixes" has what was built.)*

### Phase 4: planted breaks and the gates

Model: mainstream, low.

1. `python3 -B tools/plant.py PLANTS.json` with each break Phase 2's table
   names, on a scratch clone; each must fail its test. Quote each failure.
2. `python3 -B tools/redgreen.py RED.json --expect fail --rev ceb489e` for
   the tests marked red, and `--expect pass` on the tree.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./internal/signinflow/... ./llmprovider/x/signin/... ./llmprovider/auth/... ./wizard/...`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
5. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
6. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
7. `python3 -B tools/claims.py` on the MADR and this PLAN, and
   `python3 -B tools/denyscan.py --changed`.

Acceptance: every plant caught; every command exit 0, each status captured
before any filter. Docs: none beyond Phase 3's; a gate that finds a docs gap
is a deviation.

### Phase 5: the release (the owner's word)

Model: mainstream, medium. No step here runs under a standing approval: the
tag and the GitHub Release each wait for the owner's word in the same turn.

1. With the "tool schema refused" record's code committed (Prerequisites):
   `CHANGELOG.md`'s `[Unreleased]` becomes the next minor version's section
   (`v1.32.0` unless another release comes first), dated by its commit;
   `make changelog-check`.
2. `python3 -B tools/apidiff.py v1.31.0` lists the other record's additions
   and nothing of this one's outside `x/`; the full gate (Phase 4 step 4)
   on the release commit.
3. On the owner's word: the tag, pushed; CI on the tag in full (0052-MADR).
4. On the owner's word: the GitHub Release with the section's notes, and
   the R53 step for the oldest of seven Releases, as `v1.31.0`'s was.
5. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`.
6. `tools/records.py status` sets this PLAN `complete` once the tag's CI
   and step 5 are recorded.

Acceptance: the tag's CI passes; the section and the Release name this
change.

### Phase 6: gobble, after the release (gobble-cli's records)

Not executed under this PLAN; recorded so the hand-off is exact. At gobble-cli
`1305a61` (branch `p3-acpext-v0.5.0`), by a gobble-cli record:

* `go.mod`: the SDK pin from `v1.31.0` (MADR G8) to the release.
* Terminal path: nothing to call; the wizard verifies (MADR G4, G5).
  `internal/cli/login.go`'s notice that a typed key "was not checked"
  (G7) is no longer true for Kilo: gobble narrows it to OpenCode Zen, Go and
  Hugging Face.
* ACP path: `checkListing` (G1) calls `signin.VerifyKilo` for Kilo with the
  key and its organization, and returns its refusal; nothing is removed,
  since `checkListing`, `refusedCredential` (G3) and `publicListings` (G6)
  serve the other providers. *(2026-11-16: the alternatives this item named
  for Q1 (b) and Q2 (c) are removed; neither was chosen.)*
* *(2026-11-16: at gobble-cli `cecc892` on the same branch, the notice of
  G7 is at `internal/cli/login.go:131`, not `:117`; the hand-off reads the
  branch as it stands when gobble's record is written.)*

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`.

* Phase 1's two owner-run commands, quoted.
* `go test -count=1 ./internal/signinflow/ ./llmprovider/x/signin/ ./wizard/`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./internal/signinflow/... ./llmprovider/x/signin/... ./llmprovider/auth/... ./wizard/...`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev ceb489e`, then
  `--expect pass`; `python3 -B tools/plant.py PLANTS.json`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN

## Rollout and Rollback

* **Order.** Phase 0 now; Phase 1 on approval, and nothing after it until
  its evidence is recorded; Phases 2 to 4 in order; Phase 5 with the other
  record; Phase 6 in gobble-cli after the tag.
* **Rollout.** A caller of the wizard with Discover sees one more request
  for a Kilo credential with no organization, a refusal where Kilo refuses
  the key, and a warning where the profile cannot say. A caller without
  Discover, with an organization, or of another provider sees nothing new.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** a patch release that turns the refusal back
  into a warning, by a dated amendment of the MADR (for instance, if a key
  kind Phase 1 could not reach is refused). `signin.VerifyKilo` may change or
  go in any release (R4).

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index.

<!-- claims: self=ceb489e -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | R2's `x/signin` row, which Phase 3 extends | `docs/guides/api-standards.md:64` | `` \| `llmprovider/x/signin` \| experimental: one sign-in handle `` |
| P2 | `internal/signinflow` already imports `auth` | `internal/signinflow/signinflow.go:24` | `"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"` |
| P3 | The architecture's wizard paragraph on Kilo | `docs/architecture.md:1050` | `- **Kilo.** A Kilo device login saves its token the same way, offers the` |
| P4 | and its Kilo device login | `docs/architecture.md:864` | `` `KiloProfile` lists the account's organizations. `` |
| P5 | The README's `x/signin` example | `README.md:366` | `run, err := signin.Start(ctx, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, signin.Options{})` |
| P6 | The device-login variable covers Kilo | `AGENTS.md:260` | `` `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok, Kilo and `` |
| P7 | A record's own variable | `AGENTS.md:266` | `` switched on by `LLMPROVIDER_LIVE_<NNNN>`, with suffixes for its options, as `` |
| P8 | A watched test | `wizard/credential_0066_test.go:247` | `func TestKept_KiloRefusedLeadsToSignIn_0066(` |
| P9 | A watched test | `wizard/describe_0056_test.go:108` | `func TestListingWarnings_NoProviderText_0056(` |
| P10 | A watched test | `wizard/identity_0079_test.go:54` | `func TestWizardOwnRequestsNamed_0079(` |
| P11 | A watched test | `wizard/kilo_device_test.go:19` | `func TestConfigureLLM_KiloListsTheChosenOrganization(` |
| P12 | A watched test | `wizard/options_0056_test.go:108` | `func TestKiloDeviceKey_Refused_0056(` |
| P13 | A watched test | `wizard/session_0059_test.go:190` | `func TestSession_KiloRefusedNotSaved_0059(` |
| P14 | The live device login Phase 1 runs | `llmprovider/live_oauth_login_test.go:148` | `func TestLive_KiloDeviceLogin(t *testing.T) {` |
| P15 | The wizard's profile seam | `wizard/auth.go:44` | `kiloProfile       = auth.KiloProfile` |
| P16 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| P17 | `TestKiloDeviceKey_ReturnsTheKey_0056` runs without Discover | `wizard/options_0056_test.go:69` | `res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, KiloDeviceKey: true},` |
| P18 | `tables-check` accepts a record's variable | `scripts/check_tables.py:13` | `variable that AGENTS.md does not name, unless it is LLMPROVIDER_LIVE_<NNNN>` |

## Execution record

### Drafting (2026-11-16)

* **Provenance.** Drafted by a strongest-tier model at high effort, on the
  coordinator's brief, from read-only investigation; nothing implemented.
  The MADR and this PLAN are `proposed`.
* **The ruling,** relayed by the coordinator, label verbatim: "In the SDK,
  with its next record". What it settles and what it leaves: the MADR, "The
  owner's ruling of 2026-11-16".
* **Verified, and how:**
  * every path a Kilo credential takes (MADR P1 to P8), read at `ceb489e`;
  * `KiloProfile`'s URL, client, bound and classification, read (MADR C5 to
    C11);
  * Kilo's client and documentation, read in the `kilocode` checkout at
    `58072bd696` (MADR K1 to K14);
  * gobble's two paths, read at gobble-cli `1305a61` with `git show`
    (MADR G1 to G8);
  * 25 live requests with made-up tokens only (MADR L1 to L25); bodies
    hashed, then deleted;
  * every line the two records cite, by `python3 -B tools/claims.py` at
    the revisions their markers name.
* **Not verified:** what any real key gets (Phase 1); whether Kilo answers
  a bad credential with 403; the profile's rate limits on a 200; whether
  the six watched tests stay green under Phase 3 (Phase 3, step 6); the
  "tool schema refused" record, which does not exist yet.

### Phase 0 (2026-11-16): records

* **Provenance.** Executed by the strongest-tier model that drafted the pair,
  at high effort. Phase 0 is priced at lightweight, low; the coordinator gave
  it to this model because it held the drafting context. Records only:
  nothing committed, pushed or tagged here; the coordinator commits.
* **The approval,** the owner's ruling, relayed by the coordinator:
  "Approve and run through phase 4". As put: the answers are recorded, the
  pair is set to accepted and committed under the standing approval; after
  the owner has run Phase 1's live steps, the tests, the implementation and
  the gates run without stopping, each reviewed and committed by the
  coordinator; the release still waits for the owner's word. Not the option
  the coordinator recommended (approve and run Phase 0 only, then stop).
  Not chosen: approve and run Phase 0 only, then stop; record the answers
  and do not approve yet.
* **The answers,** each the owner's ruling, relayed by the coordinator, each
  the draft's recommendation, recorded in full under its question in the
  MADR's "Open questions for the owner":
  * Q1: "All of them".
  * Q2: "An experimental `signin.VerifyKilo`".
  * Q3: "The existing auth-failure error".
  * Q4: "No: warn and continue".
  * Q5: "Warn within 10 s and save".
  * Q6: "The origin in use, as `KiloProfile` does".
  * Q7: "None beyond `Discover` false".
* **What changed.**
  * The MADR: `status: accepted`; a section "The owner's answers and
    approval of 2026-11-16"; "Decision Outcome" decided; each D heading
    answered; D5's clause that was Q4's option (c) struck with a dated note;
    the answers under Q1 to Q7; the gobble-cli 0051 notes cited at
    `cecc892`, where they are now committed.
  * This PLAN: `status: in-progress`; the approval under "Rules for every
    phase"; the alternatives for unchosen answers removed with dated notes;
    Phase 1 step 3 rewritten with exact commands for the owner; Q8 put to
    the owner; G7's line at `cecc892` noted in Phase 6.
  * `docs/README.md`: both rows' statuses, by `tools/records.py status`.
* **Open:** Q8 (Phase 1 step 3), which the answers do not settle. Phase 1
  can be written and committed before it is answered; Phase 2 cannot start
  before it is.

### Phase 1, the agent's part (2026-11-16)

* **Provenance.** A mainstream model at low effort, as the phase prices it,
  on the coordinator's brief. Nothing committed, pushed or tagged here; the
  coordinator commits. Base commit `41160af`.
* **The approval,** the owner's ruling, relayed by the coordinator: "Approve
  and run through phase 4". Phase 1's live steps are the owner's own.
* **The made-up probes, repeated** (a fresh made-up token per request; "no
  credential", "a made-up bearer token", "a JWT-shaped made-up token"; bodies
  hashed, never stored). No result differs from the MADR's table:

  | MADR ids | Request | No credential | Made-up | JWT-shaped |
  | :--- | :--- | :--- | :--- | :--- |
  | L1 to L3 | `GET api.kilo.ai/api/gateway/models` | 200 | 200 | 200, bodies identical |
  | L4 to L6 | `GET ai-gateway.kilo.ai/api/v1/models` | 200 | 200 | 200, identical to L1 |
  | L7, L8 | `GET api.kilo.ai/api/organizations/{made-up id}/models`, organization header | not sent | 401 | 401 |
  | L9, L10 | `GET api.kilo.ai/api/gateway/organizations/{made-up id}/models` | not sent | 401 | 401 |
  | L11, L12 | `GET ai-gateway.kilo.ai/api/v1/organizations/{made-up id}/models` | not sent | 401 | 401 |
  | L13 to L15 | `GET api.kilo.ai/api/profile` | 401, "Unauthorized" | 401, "Invalid token (id)" | 401, "Invalid token (id)" |
  | L16 | `GET api.kilo.ai/api/profile`, organization header | not sent | 401, "Invalid token (id)" | 401, "Invalid token (id)" |
  | L17, L18 | `GET api.kilo.ai/api/profile/balance` | 401, "Unauthorized" | 401, "Invalid token (id)" | 401, "Invalid token (id)" |

  The three organization URLs answered 401 "User not authenticated - no user
  to set on context" to both token shapes.
* **The test.** `llmprovider/auth/live_kilo_profile_0080_test.go`, package
  `auth_test` (an external test package, because `catalog` imports `auth`),
  build tag `live_gateways`, `TestLive_KiloProfile_0080`, skipped unless
  `LLMPROVIDER_LIVE_0080=1`. Subtests `DashboardKey` (`KILO_API_KEY`;
  `auth.KiloProfile`), `OrganizationToken`
  (`LLMPROVIDER_LIVE_0080_ORG_KEY` and `LLMPROVIDER_LIVE_0080_ORG`; the
  profile, then the organization's listing) and `RevokedKey`
  (`LLMPROVIDER_LIVE_0080_REVOKED_KEY`; must match `ErrAuthFailure` and not
  `ErrNotPermitted`), each
  skipping with a fixed message when its variable is unset. It logs a fixed
  word ("accepted", `ErrAuthFailure`, `ErrNotPermitted` or "other error"), the
  HTTP status, the number of organizations and whether a personal account is
  present.
  * **Deviation from the PLAN's wording, for the owner's notice.** The PLAN
    names `catalog.List` for the organization listing. `List` degrades a
    refused listing to a fallback catalog and returns no error (seen on the
    stand-in: a 401 listing read "accepted"), so it could not show a
    refusal. The test calls `catalog.ListRaw`, whose errors are returned.
    Noted at the step too; confirmed by the coordinator on 2026-11-16 (the
    coordinator's, not the owner's).
  * **A 403 (revised 2026-11-16 at the coordinator's review).** A Kilo 403
    is `ErrNotPermitted` by the classification table
    (`llmprovider/api_error.go:552` to `:554`), and `APIError.Unwrap`
    (`llmprovider/api_error.go:179` to `:187`) adds the legacy status
    sentinel, which for 403 is `ErrAuthFailure` (`:225`): `errors.Is` is
    true for both. The first version tested `ErrAuthFailure` first, and on
    a stand-in 403 it logged "ErrAuthFailure, HTTP 403" and `RevokedKey`
    passed. `classifyKiloError` now tests `ErrNotPermitted` first: a 403
    logs "ErrNotPermitted, HTTP 403" and `RevokedKey` fails, its message
    saying in fixed words that a 403 is not a refusal (Q4: "No: warn and
    continue"). Re-run: the same eight stand-in cases, fake-key count 0
    in each, and a 403 on `RevokedKey`, which fails with "ErrNotPermitted,
    HTTP 403"; the key-logging break counted 1, the error-text break 0
    keys and 4 lines of error wording, as before.
* **Secret safety, proved on stand-ins.** The logic under proof is the live
  test's own functions (`kiloProfileVerdict`, `kiloListingVerdict`,
  `classifyKiloError`, `logProfileVerdict`) and its three subtests, reached
  through `kiloTestOpts`, a package variable that is nil in a live run and
  that a scratch test set to an `httptest` server. The proof is scratch, not
  committed: a test of a test would be dead weight in the tree. A scratch
  clone held the file and a scratch test planting one fixed fake key value
  (`FAKE-KEY-0080-PLANTED-xyzzy`, in all four variables). For each case the
  whole `go test -v` output (stdout and stderr) was grepped for the fake key:

  | Case | Stand-in answers | Fake-key count |
  | :--- | :--- | :--- |
  | 200 | a profile and a model list | 0 |
  | 200, echo | the Authorization header echoed in the profile's email field | 0 |
  | 401 | refusal | 0 |
  | 403 | refusal | 0 |
  | 500 | outage | 0 |
  | 401, echo | the header echoed in the body and a response header | 0 |
  | 401, echo of the bare token | the key in the message field | 0 |
  | connection refused | an error whose text carries the URL | 0 |

  Planted breaks, on the scratch copy only: the test logging the key
  (`t.Logf("key %s", key)`) gave a count of 1, so the grep can fail. The test
  printing the error's text (`err.Error()`) gave fake-key count 0 (the SDK
  redacts an echoed token and drops the body's message from `APIError`'s
  text), so the key grep alone does not catch that break; a grep for the
  error's wording (`authentication failed`, `kilo HTTP`) found 4 lines in the
  broken run and 0 in every unbroken one.
* **The read-through** of every log, error, skip and print call in the file
  (none holds a key, a variable that holds one, or an error's text;
  `classifyKiloError` never calls `Error()`; there is no `fmt`, `log`,
  `os.Stdout`, `os.Stderr` or `panic`):
  * line 33 `t.Skip("LLMPROVIDER_LIVE_0080 unset")`: fixed text;
  * line 38 `t.Skip("KILO_API_KEY unset")`: fixed text;
  * line 45 `t.Errorf(..., v.word, v.status)`: a fixed word and an int;
  * line 52 `t.Skip("LLMPROVIDER_LIVE_0080_ORG_KEY or LLMPROVIDER_LIVE_0080_ORG unset")`: fixed text;
  * line 59 `t.Logf("organization listing: %s, HTTP %d", l.word, l.status)`: a fixed word and an int;
  * lines 61 and 64 `t.Errorf(..., word, status)`: a fixed word and an int;
  * line 70 `t.Skip("LLMPROVIDER_LIVE_0080_REVOKED_KEY unset")`: fixed text;
  * line 77 `t.Errorf(..., v.word, v.status)`: a fixed word and an int;
  * line 150 `t.Logf("profile: %s, organizations %d, personal account %t", v.word, v.orgs, v.personal)`: a fixed word, a count and a bool;
  * line 153 `t.Logf("profile: %s, HTTP %d", v.word, v.status)`: a fixed word and an int.

  The `kiloVerdict` struct holds a word, a status, a count and a bool, and no
  error, key, body or account field. The test writes no file and sets no
  environment variable; its requests go to the origin the SDK uses.
* **Gates,** each exit status 0: `make pre-add-check
  FILES="llmprovider/auth/live_kilo_profile_0080_test.go"` (through
  `run-heavy`); `go vet -tags live_gateways ./llmprovider/auth/`;
  `golangci-lint run -c .golangci.yml --build-tags live_gateways
  ./llmprovider/auth/` ("0 issues"); `go test ./llmprovider/auth/ -count=1`.
  The live test was compiled under its tag and not run.
* **Q8, answered 2026-11-16,** the owner's ruling, relayed by the
  coordinator: "The dashboard key and the device sign-in". As put: those two
  are the kinds the wizard's paths actually receive; the organization token
  and the revoked key run only if the owner has them to hand, otherwise each
  is recorded "not run: no such key" and stays named under "Not verified".
  Not chosen: whatever the owner can run (nothing required); all four.
* **The owner's run:** not yet run. The owner runs it at the commit that
  adds this test, named in the entry for the owner's run.

### Phase 1, the owner's run (2026-11-17)

* **Provenance.** The owner ran both required steps in their own terminal at
  commit `288d329` (the commit that adds the live test) and sent back the
  lines below, relayed by the coordinator. They hold no key; they are
  recorded as sent. Recorded by a mainstream model at medium effort, on the
  coordinator's brief, together with Phase 2's entry. The owner's words with
  them, relayed by the coordinator: "so both passed/success".
* **Run 1,** `LLMPROVIDER_LIVE_0080=1 go test -tags live_gateways
  ./llmprovider/auth -run '^TestLive_KiloProfile_0080$' -count=1 -v`, with
  `KILO_API_KEY` set to a dashboard key, as the owner's output relayed by the
  coordinator:

  ```text
  --- PASS: TestLive_KiloProfile_0080 (0.37s)
      --- PASS: TestLive_KiloProfile_0080/DashboardKey (0.37s)
      --- SKIP: TestLive_KiloProfile_0080/OrganizationToken (0.00s)
      --- SKIP: TestLive_KiloProfile_0080/RevokedKey (0.00s)
  PASS
  ok      github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth       1.021s
  ```

* **Run 2,** `LLMPROVIDER_LIVE_DEVICE_LOGIN=1 go test -tags live_gateways
  ./llmprovider -run '^TestLive_KiloDeviceLogin$' -count=1 -v`, the device
  code approved by the owner, as the owner's output relayed by the
  coordinator:

  ```text
      live_oauth_login_test.go:163: picked kilo-auto/free from [kilo-auto/free nvidia/nemotron-3.5-lightning:free poolside/laguna-s-2.1:free]
      live_oauth_login_test.go:163: profile: 0 organizations, personal account true
  --- PASS: TestLive_KiloDeviceLogin (9.57s)
  PASS
  ok      github.com/maccavelli/go-llmprovider-sdk/llmprovider    10.293s
  ```

* **What the runs show.**
  * **A dashboard API key is accepted by the profile endpoint.** The
    `DashboardKey` subtest passes only when `auth.KiloProfile` returns no
    error: `kiloProfileVerdict` returns the word "accepted" only on a nil
    error (`llmprovider/auth/live_kilo_profile_0080_test.go:110` to `:116`),
    and the subtest fails unless the word is "accepted" (`:42` to `:46`).
    The owner's paste did not include that subtest's `profile:` log line, so
    its organization count and personal-account flag are "not sent"; none
    is recorded here.
  * **A device token is accepted by the profile.** The second run logged
    "profile: 0 organizations, personal account true", which
    `checkKiloSession` prints only after `auth.KiloProfile` returned no
    error (`llmprovider/live_checks_test.go:88` to `:92`); the test called it
    through `llmprovider/live_oauth_login_test.go:163`. One generation on a
    free model then succeeded: the test passed, and `checkKiloSession`
    fails on an empty or failed generation
    (`llmprovider/live_checks_test.go:98` to `:101`). This also settles, as
    of 2026-11-17, that a device token still reads the profile (last seen
    2026-10-19, MADR C35).
  * **The organization token: not run: no such key,** by the owner's answer
    to Q8 ("The dashboard key and the device sign-in", relayed by the
    coordinator). It stays named under "Not verified" in the MADR. The
    revoked-key subtest was skipped in run 1; the owner ran it later (run 3,
    below).
  * **For the design (revised by run 3, below):** after runs 1 and 2 a
    REFUSED key had not been seen live, and the 401 for a bad key rested on
    the made-up-token probes. Run 3 supplies the live observation.
* **Run 3 (a continuation, added 2026-11-17 at the coordinator's relay).**
  After the two required runs the owner ran one more on 2026-11-17, from a
  clean copy of commit `288d329` (a `git archive` of that commit in a
  temporary directory, so no in-progress file was in the build), with an
  invalid key in `LLMPROVIDER_LIVE_0080_REVOKED_KEY`: `LLMPROVIDER_LIVE_0080=1
  go test -tags live_gateways ./llmprovider/auth
  -run '^TestLive_KiloProfile_0080$/^RevokedKey$' -count=1 -v`. The owner's
  output, relayed by the coordinator (no key in it):

  ```text
  === RUN   TestLive_KiloProfile_0080
  === RUN   TestLive_KiloProfile_0080/RevokedKey
      live_kilo_profile_0080_test.go:75: profile: ErrAuthFailure, HTTP 401
  --- PASS: TestLive_KiloProfile_0080 (0.52s)
      --- PASS: TestLive_KiloProfile_0080/RevokedKey (0.52s)
  PASS
  ok      github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth       1.159s
  ```

  * **What it shows.** A refused key HAS now been observed live: Kilo's
    profile endpoint answered HTTP 401 to a key the owner supplied as
    invalid, and the SDK classified it `ErrAuthFailure`, not
    `ErrNotPermitted`, which the refusal rule relies on (Q3, "The existing
    auth-failure error"; Q4, "No: warn and continue"). The 401 for a bad key
    was observed with a made-up token by the agents (MADR L13 to L16, and the
    repeat in "Phase 1, the agent's part") and with an invalid key by the
    owner.
  * **The invalid key's kind,** in the owner's words, relayed by the
    coordinator: "it was a mistyped real key, changed the last char". So a
    real dashboard key with its last character changed was refused by Kilo's
    profile endpoint with HTTP 401, classified `ErrAuthFailure`: the
    mistyped-key case the decision exists for, observed live.
  * **What it does not show.** The subtest is named `RevokedKey`, but no
    revoked key was tested. Still not verified: a key revoked after having
    been valid; an organization token ("not run: no such key").
* **Phase 1's stop condition** (a valid key kind the profile refuses) **did
  not arise for the two required kinds:** both were accepted. Phase 1's
  acceptance is met for the dashboard key and the device token; the
  organization token was not run, which step 3's rule and Q8 allow, and the
  invalid-key run (run 3) was refused with 401, `ErrAuthFailure`. Phase 2
  may start.
* **Step 4's other asks:** the execution record gets each kind's result
  (above); the MADR's "Not verified" loses each kind settled and keeps each
  kind not run, named. The MADR's list is changed by a dated addition and a
  struck bullet, 2026-11-17 (its "Not verified").

### Phase 2, the tests (2026-11-17), stopped at what "red" means

* **Provenance.** A mainstream model at medium effort, as the phase prices
  it, on the coordinator's brief. Base commit `288d329`, whose Go code is
  identical to `ceb489e` (`git diff ceb489e 288d329` touches only records
  and the live test). Nothing committed, pushed or tagged.
* **The approval,** the owner's ruling, relayed by the coordinator: "Approve
  and run through phase 4".
* **Stop on one point, for the owner.** "Rules for every phase" says a new
  test "is seen red at `ceb489e`" on a scratch clone, and that "A phase
  starts on the commit of the one before it, on `main`." It does not say
  whether Phase 2's tests are committed while red, kept out of the default
  build until Phase 3, or committed with Phase 3. Committed alone, two of the
  three files do not compile (`signinflow.VerifyKilo`, `signin.VerifyKilo`
  are new names) and the wizard file has seven red tests, so `main`, which
  runs CI on every push, would fail `go vet` and `go test` for three
  packages. The three test files are therefore NOT in the tree and NOT
  staged; they are in the scratch folder (copied to
  `~/.cache/claude-session-5a7234fa/scratch/sdk-0080-p2/`). The options put
  to the owner are in the coordinator's report.
* **The twelve tests** (files as they stand in the scratch copy; "red" is
  at `ceb489e`, by `tools/redgreen.py --expect fail --rev ceb489e`, 7 of 7;
  "green" is `--expect pass --rev ceb489e`, 3 of 3):
  * `wizard/kilo_verify_0080_test.go:220` `TestKiloVerify_TypedKeyRefused_0080`:
    red. `notices ["live model listing for Kilo Gateway is unavailable; ..."];
    want "Kilo Gateway refused this key (authentication failed, HTTP 401);
    enter another" once` (the first key is returned; one secret unused).
  * `:248` `TestKiloVerify_EnvironmentAndSavedKeyRefused_0080`: red. `error
    nil; want a refusal saying "Kilo Gateway refused the key in
    KILO_API_KEY"`, and the same for "the saved key".
  * `:284` `TestKiloVerify_DeviceKeyRefused_0080`: red. `error nil; want a
    refusal saying "refused the device sign-in; sign in again"`; `0 profile
    reads; want one`.
  * `:305` `TestKiloVerify_DeviceSessionRefused_0080`: red. `error nil; want a
    refusal saying "refused the sign-in; sign in again"`, and the session was
    saved.
  * `:326` `TestKiloVerify_KeptSessionRefused_0080`: red. The kept-refused
    notice and the menu are absent; `profile read with 0 tokens`.
  * `:354` `TestKiloVerify_OutagesWarn_0080`: red for the eight key paths
    (403, 404, 500, 503, a body that does not decode, a refused connection, a
    stall past `DiscoverLimit`, the device key at 503): `want "could not
    verify the Kilo key (<class>)" once`. The two device-session subcases
    pass at `ceb489e`, since today's menu warning already covers them.
  * `:478` `TestKiloVerify_OrganizationReadsNoProfile_0080`: green at
    `ceb489e`.
  * `:520` `TestKiloVerify_NoDiscoverNoRead_0080`: green at `ceb489e`.
  * `:579` `TestKiloVerify_OtherProvidersReadNoProfile_0080`: green at
    `ceb489e`.
  * `:606` `TestKiloVerify_RequestNamed_0080`: red. `0 profile requests; want
    one`.
  * `internal/signinflow/kilo_0080_test.go:37` `TestVerifyKilo_Kinds_0080`:
    does not compile until Phase 3: `undefined: VerifyKilo`.
  * `llmprovider/x/signin/verify_0080_test.go:60` `TestVerifyKilo_0080`: does
    not compile until Phase 3: `undefined: signin.VerifyKilo`.
* **The fakes** follow the MADR's rows: the public listing answers 200 to any
  credential (L1 to L6); an organization's listing 401 to a credential it does
  not know (L7 to L12); `/api/profile` 401 "Unauthorized" with none and 401
  "Invalid token" with a bad one (L13 to L16), 200 with a profile for a good
  one (the owner's runs, "Phase 1, the owner's run"); each failure body holds
  a fixed marker, which no prompt may show. Every credential is a fixed fake
  value; no test logs a key or an error's text. The wording of each outage
  class was checked against `describe` on the real `auth.KiloProfile`.
* **Seen to fail on a planted break,** on a throwaway prototype of Phase 3 in
  a scratch clone (never in the tree, and not Phase 3's code): `tools/plant.py`,
  11 of 11 caught: the 403 treated as a refusal, the stall bounded by 30 s,
  the check ignoring the organization, running without Discover, running for
  every provider, the identity left off, the 401 or the cause or a 200 or
  ctx's error mishandled in the two `VerifyKilo` functions. With the
  prototype every test of the three files, and the whole `wizard` package,
  passed. Phase 4 repeats the plants on the real code.
* **Existing tests:** the six named and `TestKiloDeviceKey_ReturnsTheKey_0056`
  pass at `ceb489e` with the wizard file present; the rest of the `wizard`
  package passes with it. No existing test changed.
* **Not verified:** the prototype is not Phase 3; the signinflow and
  x/signin tests were never run red (they do not compile); no push or CI.

### Phase 3 (with Phase 2's tests) (2026-11-17)

This entry has two parts: the first staging, as the executing agent wrote it
(its statements corrected in place after the reviews where they were false, and
its line references re-derived from the final staging), and "Review and fixes"
below it. The staged tree is the first staging with the reviews' fixes.

* **Provenance.** A mainstream model at high effort, as the phase prices it,
  on the coordinator's brief. Base commit `b109cea`, level with `origin/main`.
  Staged, not committed; the coordinator commits. Nothing pushed or tagged.
* **The approval and the ruling,** the owner's, relayed by the coordinator.
  The approval: "Approve and run through phase 4". The ruling of 2026-11-17:
  "One commit: the tests with phase 3's code". As put to the owner: the tests
  go into the tree when Phase 3's implementation makes them pass, and the two
  land together; "red" stays proven on a scratch clone, as the plan
  describes, and is in the record ("Phase 2, the tests"); `main` never
  carries a failing build; a dated deviation note merges the two commits.
  Options not chosen: the tests behind a temporary build tag; commit them red
  on `main`. This entry is that deviation note: Phase 2 and Phase 3 are one
  commit. The ruling is recorded under "Rules for every phase", at Phase 2
  and at Phase 3, each as a dated addition; the committed text stays.
* **Phase 2's three test files** were placed byte for byte as in the scratch
  copy (`cmp` identical): `wizard/kilo_verify_0080_test.go`,
  `internal/signinflow/kilo_0080_test.go`,
  `llmprovider/x/signin/verify_0080_test.go`. No test was changed to fit the
  code as first staged; the reviews then changed `wizard/kilo_verify_0080_test.go`
  in three places, each so that the test says what the MADR or the owner's
  ruling says (listed under "Review and fixes"). The implementation is written
  from the MADR and this PLAN, not from the Phase 2 agent's prototype, which
  was not read.
* **The steps,** each as the PLAN's Phase 3 writes it, at the staged tree:
  1. The rule once, in `internal/signinflow/kilo.go`: `VerifyKilo` (`:107`),
     with `Refused` (`:29`), the one rule (`ErrAuthFailure` and not
     `ErrNotPermitted`, 0028-MADR D-H2), `KiloVerdict` (`:36`),
     `KiloProfileReader` (`:22`), `CheckKiloOptions` (`:75`) and
     `DefaultCheckLimit` (`:18`). No request when the credential has an
     organization (`:113`); none after ctx has ended (`:116`). The wizard's
     `refusedCredential` returns `signinflow.Refused`
     (`wizard/configure.go:708`), so the listing's rule and the profile's are
     one.
  2. The wizard's check: `wizard/kilo_verify.go`, `checkCredential` (`:52`)
     and `verifyKilo` (`:82`), called by `credentialStep` at
     `wizard/configure.go:491`, `:509`, `:521` and `:548` in place of
     `discoverModels`, which they call after the profile. Under `Discover`,
     for Kilo, one read with the credential's token (`kiloToken`, `:131`: a
     key's, or a session's access token), through the `kiloProfile` seam,
     bound by `Options.DiscoverLimit` (`discoverLimit`, `:17`; `:110`), with
     the options `kiloProfileOptions` builds (`:29`: identity, endpoint in
     use, client, `ProviderOptions`, as `resolveKiloDevice` built them). A 401
     is returned as the listing's refusal is, so the loop is unchanged.
     Anything else is one warning, "could not verify the Kilo key (<class>[,
     HTTP <status>]); it is saved unchecked" (D6's wording, `:128`), and the
     sign-in goes on.
  3. `wizard/auth.go`: `resolveKiloDevice` keeps its one read (`:416`), bound
     by `DiscoverLimit` under `Discover` (`:413`), and carries its result on
     the credential (`profileRead`, `profileErr`, `:64` and `:65`; set at
     `:418`), which `verifyKilo` reuses (`wizard/kilo_verify.go:102` to
     `:109`): no second request. Under `Discover` a 401 and an ended ctx are
     told by no menu warning (`wizard/auth.go:421`): the check refuses the 401
     and returns ctx's error. Any other failure is two notices (R-A below).
     Without `Discover` today's warning stands. `resolveKiloDeviceKey` reads
     nothing; the check reads the profile once.
  4. `llmprovider/x/signin/verify.go:74`: `VerifyKilo(ctx, key, organization
     string, opts ...llmprovider.Option) error`, with `ExampleVerifyKilo`
     (`llmprovider/x/signin/example_test.go:59`, offline, `// Output:`).
  5. Docs and the dated notes, listed under "Docs rows" below.
* **The twelve tests on the real code,** `go test -count=1 -v`, all `PASS`:
  `TestKiloVerify_TypedKeyRefused_0080`,
  `TestKiloVerify_EnvironmentAndSavedKeyRefused_0080`,
  `TestKiloVerify_DeviceKeyRefused_0080`,
  `TestKiloVerify_DeviceSessionRefused_0080`,
  `TestKiloVerify_KeptSessionRefused_0080`, `TestKiloVerify_OutagesWarn_0080`
  (eight key paths and two session cases; the reviews added a stall case),
  `TestKiloVerify_OrganizationReadsNoProfile_0080`,
  `TestKiloVerify_NoDiscoverNoRead_0080`,
  `TestKiloVerify_OtherProvidersReadNoProfile_0080`,
  `TestKiloVerify_RequestNamed_0080`, `TestVerifyKilo_Kinds_0080` and
  `TestVerifyKilo_0080`. The first run of `TestVerifyKilo_0080` failed on its
  `ctx ended` case: the stand-in there answers 200 whatever the context, so a
  `VerifyKilo` that only passed ctx on returned nil. The test is right: the
  MADR's D10 says "ctx's error when ctx ends". The implementation changed (the
  rule checks ctx before the request), the test did not.
* **Two tests added,** in `wizard/kilo_verify_ctx_0080_test.go`, not in
  Phase 2's table, for what the brief's points 2 and 3 ask to be shown:
  `TestKiloVerify_EndedContext_0080` (`:19`: ctx ends during the profile read
  of a typed key, and of a device session: ctx's error, no warning, no
  refusal, nothing saved) and `TestKiloVerify_RefusalShowsNoKey_0080` (`:76`: a
  401 whose body and headers echo the Authorization header and the key in the
  shapes the SDK's redaction knows: nothing of the key in a notice or in the
  error's text). *(Corrected after the first review: as first staged the
  device-session case of the first test did not look at notices, and the code
  showed "could not read the Kilo account's organizations (cancelled)" for it,
  so "no warning" was false for a device session. Fixed in code and in the
  test; B-1 below.)* Seen to fail on planted breaks, on a scratch clone: the
  ended-context return removed gave "error <nil>; want ctx's error" for both
  cases; the ended ctx warned instead gave "notice "could not verify the Kilo
  key (cancelled); it is saved unchecked"; want neither a warning nor a
  refusal"; the key put in the returned error gave "the error's text shows the
  key". Phase 4's plant list does not name them. The coordinator keeps both
  tests on the reviewer's evidence (each is the only test that catches one
  break); the coordinator's decision, not the owner's.
* **Existing tests.** `go test -count=1 -v ./wizard/` on a scratch clone of
  the first staging, before staging (PLAN step 6): 301 `--- PASS`, 0 `--- FAIL`,
  `ok ... wizard 7.762s`; the six stop-condition tests and
  `TestKiloDeviceKey_ReturnsTheKey_0056` each `--- PASS`. No existing test,
  golden or fixture was changed by the first staging;
  `llmprovider/x/signin/example_test.go` gained `ExampleVerifyKilo` and an
  import, nothing else. The reviews' fixes move `llmprovider/internal/kiloendpoint`
  to `internal/kiloendpoint` (below) and add one test to its test file; the
  whole module passes again (gates below).
* **Point 2, secrets.** The 401 the check returns is the existing
  `*APIError` (kind auth failure), its text as the listing's: on a stand-in
  that echoed the key in the body, in `X-Echo` and in the Authorization
  header, `ResolveCredential`'s error read `wizard: Kilo Gateway refused 3
  keys: llmprovider: authentication failed: kilo HTTP 401: Invalid
  [REDACTED]` and its notices `Kilo Gateway refused this key
  (authentication failed, HTTP 401); enter another`. The key is not there.
  The text does keep a fragment of the service's own message ("Invalid"), as
  a listing's refusal's does (0056-MADR D3: the error keeps its text, a
  notice shows class and status). A warning is `describe(err)`: class and
  status. Redaction is by pattern, not by the key's value: an opaque key a
  service echoes may remain in the error's text (the review's N-4; MADR
  Consequences).
* **Point 3, bounds and cancellation.** The profile read is bound by
  `Options.DiscoverLimit` (`wizard/kilo_verify.go:17`, `:110`; for the device
  session `wizard/auth.go:413`), default 10 s, not auth's 30 s: the stall
  cases of `TestKiloVerify_OutagesWarn_0080` (limit 300 ms; a typed key and,
  since the first review, a device session, `wizard/kilo_verify_0080_test.go:466`)
  return with the "timed out" warning. The check and the listing each have the
  limit, so a check may take up to twice it. Today's listing with ctx ended
  is `wizard/configure.go:928`, `:944` (a "could not list models" warning,
  the catalog falls back) and `ResolveCredential`'s own test
  (`wizard/configure.go:410`: "A listing ctx ended is not a refusal ... must
  not pass for a checked credential"), which runs after `credentialStep` has
  saved. The check does the stricter thing: a profile read that failed while
  ctx is done is neither refusal nor warning, and `credentialStep` returns
  ctx's error before `save` (`wizard/configure.go:552`). The coordinator keeps
  this, on the reviewer's evidence: at `b109cea` a sign-in whose context ended
  inside the check window already returned ctx's error, so nothing that
  succeeded before now fails; the difference is that nothing is saved. The
  coordinator's decision, not the owner's.
* **Found, not changed by 0080** (three items; none is built here).
  * **A new session is saved after ctx ended, on the listing path,** for a
    session that is current. A scratch test on a clone: a stand-in whose
    listing cancels ctx and fails gave `err=<nil> saves=1` for a Kilo device
    session and the notice "live model listing for Kilo Gateway is unavailable
    (cancelled)". `resolvedCredential.save` (`wizard/auth.go:1026`) refuses
    only when `Token(ctx)` fails, which a current session does not.
    Pre-existing at `ceb489e`, and for every provider's new session sign-in.
    By the owner's ruling of 2026-11-17, relayed by the coordinator, "A record
    of its own, drafted now": it goes to a record of its own, which has been
    drafted outside this tree and is placed after this phase commits; 0080
    does not change it.
  * **A key that a caller's own `Prompter` returns empty, or of only spaces,
    is not checked,** for every provider, with no error. By the owner's ruling
    of 2026-11-17, relayed by the coordinator, "A record of its own: refuse an
    empty key at the prompt, every provider (Recommended)" (R-C below): 0080
    does not change it. The facts, as the third review corrected them: the
    SDK's own `TextPrompter` never returns a blank key (`Secret` returns a value
    only when it is non-empty, `wizard/text_prompter.go:402`; trims the entry on a terminal,
    `:496`, and, when it is not reading from a terminal or cannot make it raw,
    reads the line through `readLine`, `:420` and `:430`, whose trim is `:156`;
    ends with an error at the end of input, `:408`; and asks again, "empty
    value; please try again", `:410`). The gap needs a caller's own
    `Prompter` that returns an empty or all-space key: `promptAPIKey` trims it
    to empty and the wizard returns it as a key, with no check and no error, at
    `b109cea` and on this staging alike, for every provider. A scratch test on
    this staging, with such a `Prompter`, agrees: Kilo, `Discover`, `"   "`:
    `err=<nil> key="" profile reads=0`, and OpenCode Zen the same;
    `signin.VerifyKilo("")` refuses it (`TestVerifyKilo_0080/no_credential`).
    The first account of it, in the first review's second round and in this
    entry's earlier text, said a key "typed as only spaces" was returned
    unchecked; the third review corrected it, and this is the corrected
    statement. The wizard's check skips an empty token
    (`wizard/kilo_verify.go:94`), with a comment saying why. The docs say so
    (README, CHANGELOG, architecture).
    *(Added 2026-11-20: this gap is closed by
    [0085-MADR-refuse-an-empty-typed-key.md](0085-MADR-refuse-an-empty-typed-key.md),
    whose prompts ask again for a blank key. By the owner's ruling of that
    day, relayed by the coordinator, "Reword the comment in 0085 and note it
    in 0080's plan (Recommended)", 0085's Phase 1 rewords the comment at
    `wizard/kilo_verify.go:95` and the three docs sentences; the skip of an
    empty token stays. See 0085-PLAN's entry of 2026-11-20, second pass.)*
  * **A flaky test of another package,** hit once by the fourth review's race
    run: `TestStart_NoLoopback_0074/grok` failed with "something listens at
    127.0.0.1:53531; want no listener under NoLoopback"
    (`llmprovider/x/signin/start_browser_0074_test.go:277`). The test picks a
    random port between 49152 and 65535 and dials it (`:314` to `:331`), so
    another package's loopback listener in the same `go test` can hold it. The
    file is byte-identical to `b109cea`, and a rerun of the package passed ten
    rounds. It is a pre-existing flaky test, not 0080's, and it is going to the
    owner; 0080 does not change it, and this phase did not touch the test.
    *(Added 2026-11-17: by the owner's ruling of that day, relayed by the
    coordinator, "yes, fold that into the plan", its repair is a step of
    [0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md](0083-PLAN-let-the-wizard-sign-in-without-a-callback-listener.md),
    Phase 1, under its "Amendment 2026-11-17: the no-listener test that another program's listener can fail".)*
* **Point 4, one request, the right origin.** The stand-in counts: one profile
  read where the MADR says one (typed, environment, saved, device key: the
  key tests assert `1` or the token list); none with an organization
  (`TestKiloVerify_OrganizationReadsNoProfile_0080`, both cases, and, since
  the first review, an organization in a token's embedded URL or in the
  endpoint's path), none with `Discover` false
  (`TestKiloVerify_NoDiscoverNoRead_0080`: 0 requests for a typed key and a
  device key, 1 for a device session, its menu's), none for another provider
  (`TestKiloVerify_OtherProvidersReadNoProfile_0080`); the device session
  reads once, not twice (`TestKiloVerify_DeviceSessionRefused_0080`,
  `TestKiloVerify_OutagesWarn_0080`). Origin: a custom endpoint,
  `kilo.example.test` (`TestKiloVerify_RequestNamed_0080`: host, path
  `/api/profile`, the application's User-Agent, the key as bearer); and a
  scratch test of a token that embeds a backend URL,
  `https://backend.example.test:<secret>`, sent the profile to
  `backend.example.test/api/profile` and the listing to the same host, as
  `KiloProfile` does. Nothing is sent to an origin the caller did not choose:
  the options are the caller's, as `resolveKiloDevice` built them. The edge
  this item first named, a token whose embedded URL names an organization, is
  the first review's S-4, resolved by ruling 2 below.
* **Point 5, API surface.** Exactly one new exported name,
  `signin.VerifyKilo`, in `llmprovider/x/signin` (`verify.go:74`), with a doc
  comment that names each kind (R52). `internal/signinflow`'s and
  `internal/kiloendpoint`'s names are internal. See "Review and fixes" for the
  API check's lines on the final tree.
* **Gates of the first staging,** after its last change to any Go file, each
  exit status taken before any filter, all `exit=0` (superseded by the final
  gates below): the full suite on the Windows laptop by `run-on-host` (42
  packages `ok`), `make pre-add-check` on the 13 Go files by `run-heavy`
  (after a 409 s wait in the queue), `go vet` with and without `live_gateways`
  and for Windows, `GOOS=windows go build`, `make api-check docs-check
  tables-check changelog-check dep-check parity-check` and
  `tools/apidiff.py v1.31.0`, `gofmt -l`.
* **Docs rows.** Fixed: `wizard/configure.go` (Options.Discover,
  Options.DiscoverLimit, Options.ProviderOptions, Options.KiloDeviceKey,
  ConfigureLLM, ResolveCredential, credentialStep); `wizard/auth.go`
  (`resolveKiloDevice`, `resolveKiloDeviceKey`, two field comments);
  `wizard/prompter.go` (the package doc); `llmprovider/auth/kilo_device.go`
  (`KiloProfile`); `llmprovider/x/signin/doc.go`; `README.md` (the `x/signin`
  section and the wizard's bullets); `docs/architecture.md` (the package
  tables, the Kilo device login, "The wizard", "The handle", "Kilo", "A
  refused key"); `docs/guides/api-standards.md` (R2's `x/signin` and
  `internal/signinflow` rows); `CHANGELOG.md` `[Unreleased]` (Added and
  Changed); dated notes of 2026-11-17 in 0056-MADR D2, 0017-MADR D2,
  0059-MADR W2 and 0028-MADR D-H10, each pointing at 0080-MADR. No
  `docs/README.md` row changed: no record's title or status changed. The
  reviews' docs changes are listed below.

#### Review and fixes (2026-11-17)

* **Provenance.** Four independent strongest-tier reviews at high effort, each
  on the coordinator's brief, of the staging as it then stood. The first
  reviewed the first staging at `b109cea` (22 files; the first report said
  21). Its verdict, relayed by the coordinator: the behaviour is right for
  every credential's refusal and save (the whole module and ten race rounds
  pass, 17 of its 21 own breaks were caught, the API check is clean), and not
  committable as staged: two false statements, two things in `VerifyKilo`,
  two test gaps and two owner rulings. The reviewer first reported 19 of its
  21 breaks caught and corrected its own count in its second round: its first
  round log shows 17, since B2b and B2c survived because the organization
  guard was duplicated, and B5b and B13 were real test gaps. The second review
  found the code to do what the decision and the two rulings say in every case
  it ran (31 breaks, 29 caught; race rounds and the whole module clean; the
  package move judged necessary and the smaller risk), and two survivors, R1d
  and R1e, closed below. The third review (43 of its breaks caught; race rounds
  and the whole module clean; the listing's request unchanged against
  `b109cea`) found everything in the fourth staging's delta right except one
  defect in R-D, which let the wizard accept a wrong key with no check (R-D
  below), and some false statements, corrected in place below. The fourth
  review passed the fifth staging ("this can be committed as staged"): the
  blocker fixed by construction; the listing's actual requests identical to
  `b109cea` in all 48 rows of its comparison (four providers by twelve
  caller-option sets); 51 of 52 breaks caught, the one survivor a double entry
  of the same organization, which changes nothing; the one flake above; and two
  should-fix items and three notes, closed in this entry: the listing's option
  order pinned for every provider, the "three files" statement corrected, the
  `kiloProfileOptions` comment narrowed, the `TextPrompter` citation completed
  and the flake recorded. The fixes are by the same executing agent, a
  mainstream model at high effort, on the coordinator's second, third, fourth,
  fifth and sixth briefs. Two coordinator's decisions, recorded as such and not the owner's:
  the two added tests stay (each is the only test that catches one break), and
  the stricter handling of a done context stays (Point 3 above).
* **B-1, a device session whose ctx ends showed a warning.** Under `Discover`
  a device session whose ctx ended shows no menu warning
  (`wizard/auth.go:421`); the check returns ctx's error
  (`wizard/kilo_verify.go:126`, `signinflow.VerifyKilo`'s ctx check,
  `internal/signinflow/kilo.go:116`).
  `TestKiloVerify_EndedContext_0080`'s device-session case runs the same
  assertion as the typed key's (`assertEnded`, which also forbids "could not
  read"). Plant (the ended-ctx condition dropped): `--- FAIL:
  TestKiloVerify_EndedContext_0080`. The two sentences of this entry that
  said so are corrected in place above.
* **R-A, the device session's notices under `Discover`, the owner's ruling of
  2026-11-17, relayed by the coordinator: "Two notices".** As put: the staged
  code showed only D6's warning, so a person whose account has organizations
  was put on the personal account without being told, which affects whose
  usage and billing apply; the record asks for one warning but does not
  settle this; the option chosen reads "The decision's warning and the old
  organisation-menu warning both show. Says everything, but contradicts the
  decision's "one warning"." This replaces the first review's B-2 as built
  (D6's warning alone, which was the coordinator's reading, superseded). Built:
  under `Discover`, a device session whose profile read ends in anything that
  is not a refusal and not the caller's ctx ending shows two notices, each
  once, the session saved: first "could not read the Kilo account's
  organizations (<class>); using the personal account"
  (`wizard/auth.go:433`), then "could not verify the Kilo key (<class>); it is
  saved unchecked" (`wizard/kilo_verify.go:128`). Unchanged: a 401 is a
  refusal with no menu warning (D3); an ended ctx shows nothing and saves
  nothing; without `Discover` the menu's warning alone. The order: the menu's
  first, because it reports what happened first (the organizations could not
  be read, the personal account is used) and the check's then reports the
  conclusion for the key; it is also the order the code runs in. Tests:
  `assertSessionOutage0080` (`wizard/kilo_verify_0080_test.go:359`) requires
  each notice exactly once and the menu's first; the device-session cases of
  `TestKiloVerify_OutagesWarn_0080` (403, 503, and the stall, `:466`) use it,
  which tightens them from "either wording" and then from "D6's alone" to the
  ruling (a test change that makes the test say what the owner ruled);
  `TestKiloVerify_DeviceSessionRefused_0080` asserts no menu warning beside a
  401. Plants: the menu's warning missing under `Discover`: `--- FAIL:
  TestKiloVerify_OutagesWarn_0080`; D6's warning missing for the device
  session: `--- FAIL: TestKiloVerify_OutagesWarn_0080`; the menu's warning
  returning on a 401: `--- FAIL: TestKiloVerify_DeviceSessionRefused_0080`;
  the menu's warning shown for an ended ctx: `--- FAIL:
  TestKiloVerify_EndedContext_0080`; the menu's warning lost without
  `Discover`: `--- FAIL: TestKiloVerify_NoDiscoverNoRead_0080`. The MADR has a
  dated amendment at D6 saying it departs from "one warning" for this case,
  and its earlier staged addition at D6, the coordinator's reading, is
  corrected in place to say it was superseded.
* **Ruling 1 (the first review's S-1), the owner's ruling, relayed by the
  coordinator: "Its own 10 s bound; a stall is "cannot say"".** As put:
  `VerifyKilo` bounds its profile read by 10 s itself and returns nil when
  that passes, as for any outage; it returns the context's error only when the
  caller's own context ended; one rule for the wizard and for callers such as
  gobble, matching Q5. Options not chosen: document it and let the caller
  handle its deadline; a bound the caller can set, defaulting to 10 s. Built:
  the 10 s is `signinflow.DefaultCheckLimit` (`internal/signinflow/kilo.go:18`),
  which the wizard's default discovery limit is too
  (`wizard/configure.go:26`): one constant. `signin.VerifyKilo` bounds its read
  with it (`llmprovider/x/signin/verify.go:15`, `:80`) and tells the two ends
  apart by whose context ended: after an unverified result it returns an error
  only if the caller's own `ctx.Err()` is non-nil (`:87`); its own bound ends
  only the derived context, so that case is nil. When both ended at once the
  caller's ctx wins. The doc comment says exactly this; the README example
  switches on `errors.Is(err, llmprovider.ErrAuthFailure)` and returns any
  other error, which is ctx's or a mistake in `opts`, so it no longer fails a
  sign-in on a slow network. Tests (`llmprovider/x/signin/verify_review_0080_test.go`,
  `TestVerifyKilo_Stall_0080`): a stall with no caller deadline returns nil
  after about the bound, shortened to 150 ms by `signin.SetVerifyLimit`
  (`llmprovider/x/signin/verify_export_0080_test.go:13`), a seam in a test
  file, so no exported name; a caller deadline that ends first returns its
  error within about 150 ms of a 5 s bound. Each carries a 3 s caller deadline
  as a guard, so a missing bound fails the test. Plants: no bound of its own:
  `--- FAIL: TestVerifyKilo_Stall_0080/the_function's_own_bound_passes`; the
  caller's end read as "cannot say": `--- FAIL: TestVerifyKilo_Stall_0080` and
  `TestVerifyKilo_0080/ctx_ended`; the bound's own end returned as an error:
  `--- FAIL: TestVerifyKilo_Stall_0080`. The dated additions are at D10 and Q5
  in the MADR.
  * **The 10 s is pinned (the second review's R1d and R1e survived).** A
    test holds `DefaultCheckLimit == 10*time.Second`
    (`internal/signinflow/kilo_org_0080_test.go:78`, `TestDefaultCheckLimit_0080`),
    one that `signin`'s limit starts equal to it and to 10 s
    (`llmprovider/x/signin/verify_export_0080_test.go:24`,
    `TestVerifyLimitStartsAtTheDefault_0080`, in the package, through the
    existing export file, no new exported name), and one that the wizard's
    default limit is it (`wizard/kilo_verify_ctx_0080_test.go:97`,
    `TestDiscoverLimit_0080`). Plants: `verifyLimit` set to 30 s (R1d): `---
    FAIL: TestVerifyLimitStartsAtTheDefault_0080`; `DefaultCheckLimit` set to
    30 s (R1e): `--- FAIL: TestDefaultCheckLimit_0080` and
    `TestVerifyLimitStartsAtTheDefault_0080`; the wizard's default constant set
    to 30 s: `--- FAIL: TestDiscoverLimit_0080`.
* **Ruling 2 (the first review's S-4), the owner's ruling, relayed by the
  coordinator: "Treat it as having an organisation: no profile read".** As
  put: a token whose embedded URL names an organisation is checked by that
  organisation's listing alone, as D4 intends, so a valid organisation token
  cannot be refused by an unverified check; needs a small shared helper,
  because of Go's internal-package rule. Options not chosen: accept it as
  built and name the risk; skip the profile check for any token that embeds a
  URL. Built: the organization the SDK resolves for the listing is
  `internal/kiloendpoint` `Organization` (`internal/kiloendpoint/kiloendpoint.go:95`):
  the explicit one, else the one a token's embedded URL names (`Resolve`),
  else the one the base URL's own path names. The package could not stay in
  `llmprovider/internal`, which `internal/signinflow` and `wizard` may not
  import, so it moved, by `git mv`, to the module's own `internal/kiloendpoint`,
  with its three importers (`catalog`, `providers/kilo`, `auth`) and two
  live-tagged tests re-pointed and the docs' two table rows renamed
  (`make tables-check`, `dep-check`: 0 problems). No exported name changed;
  the second review judged the move necessary and the smaller risk, and it
  stays. `signinflow.VerifyKilo` decides, once, for the wizard and for
  `signin.VerifyKilo`: `internal/signinflow/kilo.go:113`. Tests:
  `TestOrganization_0080` (`internal/kiloendpoint/kiloendpoint_test.go:52`:
  its eight rows, which are no organization, an explicit one, a token's
  embedded URL with and without an explicit one, a token's URL with a prefix
  and no organization, a base URL whose path names one with and without a
  trailing slash, and a plain gateway base, give the organization each should,
  and in each row the result is non-empty exactly when the listing URL
  `Resolve` builds contains "/organizations/"; that is all it shows);
  `TestVerifyKilo_OrganizationFromTheCredential_0080` (`internal/signinflow`,
  and `llmprovider/x/signin`);
  `TestKiloVerify_EmbeddedOrganizationReadsNoProfile_0080` and
  `TestKiloVerify_BaseURLOrganizationReadsNoProfile_0080` (`wizard`: no
  profile request, the organization's listing alone refuses the first key and
  accepts the second). Plants: the token's organization ignored, the base
  URL's path ignored, `signinflow` reading no base URL from the options, each
  `--- FAIL` in those tests. The dated addition is at D4 in the MADR.
* **R-B, a base URL whose path names an organization, the owner's ruling of
  2026-11-17, relayed by the coordinator: "Count it; record the unprobed
  listing as not verified (Recommended)".** As built: such a base URL counts as
  having an organization and the profile is not read. This is not part of the
  earlier ruling "Treat it as having an organisation: no profile read", which
  is the token's embedded URL; the MADR's D4 says so in two dated additions.
  Not verified, recorded in the MADR ("Not verified") and here: that the
  organization listing sent WITHOUT the `X-KILOCODE-ORGANIZATIONID` header
  refuses a wrong key. `llmprovider/catalog/discovery.go:1413` to `:1424`
  (confirmed): `Resolve` builds the listing's URL, and the header is set only
  when `endpoints.Org` is non-empty, which a base URL's path does not make so.
  The live probes L7 and L8 sent the header; L9 to L12 are not recorded as sent
  with it and used a made-up organization and token, so their 401 does not
  show it; the test stand-in checks the bearer token only
  (`wizard/kilo_verify_0080_test.go:117` to `:122`). The changelog and the
  README name it among their limit lines, now four.
* **R-C, a key that a caller's own `Prompter` returns empty, the owner's
  ruling of 2026-11-17, relayed by the coordinator: "A record of its own:
  refuse an empty key at the prompt, every provider (Recommended)".** 0080 does
  not change it. It is listed above under "Found, not changed by 0080", with
  the facts as the third review corrected them: the gap is a caller's own
  `Prompter` returning an empty or all-space key, which `TextPrompter` never
  does. The wizard's check leaves an empty token unchecked
  (`wizard/kilo_verify.go:94`), as every provider's check does, and the docs
  say so; `signin.VerifyKilo("")` refuses it.
* **S-2, a caller's mistake read as "cannot say".** `signin.VerifyKilo` now
  resolves its options first (`signinflow.CheckKiloOptions`,
  `internal/signinflow/kilo.go:75`; called at `llmprovider/x/signin/verify.go:75`)
  and returns the error, matching `ErrInvalidRequest`, with no request, for an
  option scoped to another provider or a base URL that does not parse.
  `TestVerifyKilo_Mistakes_0080` and `TestCheckKiloOptions_0080`
  (`internal/signinflow/kilo_org_0080_test.go:55`); plants (no check; the check
  reports nothing): `--- FAIL`. A first version of the check also parsed the
  resolved gateway URL; its plant survived because `ResolveOptions` already
  rejects an unparsable base URL, so the extra parse was dead and was removed.
* **S-3, two breaks survived.** (a) `TestKiloVerify_OutagesWarn_0080` has a
  device-session stall case (`wizard/kilo_verify_0080_test.go:466`); plants
  (the session's read bounded by 30 s, or not at all): `--- FAIL`. (b)
  `TestKiloVerify_DeviceSessionRefused_0080` asserts that the menu's warning is
  absent beside the refusal (D3); plant (the 401 warned about as before): `---
  FAIL`. Both additions, and R-A's change to the device-session cases of
  `TestKiloVerify_OutagesWarn_0080`, edit Phase 2's
  `wizard/kilo_verify_0080_test.go`, each to make a test say what the MADR or
  the owner's ruling says; nothing was loosened.
* **N-6, the duplicated organization guard.** There is one now: the wizard's
  own `res.Organization != ""` guard was removed, and
  `signinflow.VerifyKilo`'s rule (`internal/signinflow/kilo.go:113`) decides for
  both. The device session's carried read goes through the same function by a
  `read` that returns the carried result (`wizard/kilo_verify.go:102` to
  `:109`), so the rule applies to it too. Plants (the rule off, the wizard
  passing none, `signinflow` ignoring the explicit organization): `--- FAIL`.
* **Docs lines.** `CHANGELOG.md` and `README.md` name the four limits (N-1: a
  key valid only for an organization, typed with none entered, is read at the
  profile, not verified live; N-5: with `https://ai-gateway.kilo.ai/api/v1` as
  the endpoint the profile answers 404, so every sign-in warns and a wrong key
  passes with that warning, by D7; R-B's unprobed listing; R-C's empty key
  from a caller's own `Prompter`), and say two notices for a device session and that the personal
  account is used. `VerifyKilo`'s doc says plainly that nil is verified or
  cannot say, and where a caller reads the profile itself (N-3). The N-7
  slips: `internal/signinflow/kilo.go`'s comment on bounds now says who
  bounds (the caller, by ctx: the wizard by `DiscoverLimit`,
  `signin.VerifyKilo` by `DefaultCheckLimit`); the changelog's "and `ctx`'s
  error" is completed; "21 files" was 22. N-4: `VerifyKilo`'s doc no longer
  says "Its text holds no key", and says the redaction is by pattern; the MADR
  has a dated addition under Consequences, with N-1 and N-5. The architecture
  and the R2 tables name `internal/kiloendpoint` and the new `signinflow`
  names. The sentences that said an organization was "given" state
  the listing's real order, once, as R-D below built it, and the README,
  CHANGELOG, architecture, the MADR and the doc comments say the same thing;
  the earlier wording, "in that order of precedence" with the Result's
  organization first, was false and is corrected in place.
* **R-D, an organization given through the caller's provider options, the
  owner's ruling of 2026-11-17, relayed by the coordinator: "Share the type
  internally and fix it now (Recommended)".** The second review's item 3
  asked whether the profile is read when the organization is passed as
  `ProviderOptions: For(Kilo, catalog.WithKiloOrganization(x))`. A scratch test
  said yes (the listing was the organization's and the profile was read once
  as well), and the fix needed more than the one place the brief allowed, so
  it stopped, and the owner ruled. As put: the rule is that a Kilo credential
  with an organisation reads no profile; a test showed the profile is still
  read when the calling program supplies the organisation through the wizard's
  provider options; the fix needs one internal type shared across three
  packages. The option chosen read:
  "Three files, no exported name, the listing behaves as before. The rule then holds for every way an organisation can be given, with a test and a planted break. One more short pass and re-check before the commit."
  Built: the option's value is
  `kiloendpoint.OrganizationOption` (`internal/kiloendpoint/kiloendpoint.go:33`),
  which `catalog.WithKiloOrganization` carries (`llmprovider/catalog/config.go:167`)
  and the listing's configuration reads (`:212`), replacing catalog's own
  unexported type; `signinflow.VerifyKilo` reads it from the resolved options at
  the one place the organization is decided (`internal/signinflow/kilo.go:56`,
  `kiloInputs`, called at `:113`, through `kiloendpoint.OptionOrganization`,
  `internal/kiloendpoint/kiloendpoint.go:44`). Five non-test files changed for
  it, not three: `internal/kiloendpoint/kiloendpoint.go`,
  `llmprovider/catalog/config.go` and `internal/signinflow/kilo.go`, the three of the
  option the owner chose, and `wizard/configure.go` and
  `wizard/kilo_verify.go`, for the one place the options are built (below); no
  exported name was added or changed. **The coordinator's decision, not the
  owner's ruling:** the option the owner chose read "Three files, no exported
  name, the listing behaves as before..."; the third review's blocker could only
  be closed by also building the options in one place in the wizard; the
  coordinator judged that inside the ruling's purpose (the check decides by the
  listing's own rule), on the evidence that no exported name changed and that
  the fourth review found the listing's requests identical to `b109cea` in 48
  of 48 rows (four providers by twelve caller-option sets); the owner is told. The listing's request is what it was: the existing
  `catalog` and `providers/kilo` tests, among them the one that asserts the
  organization's URL and header for `WithKiloOrganization`
  (`llmprovider/providers/kilo/listing_test.go`,
  `TestListModels_ListsTheOrganization`), pass unchanged, and the catalog test
  below asserts the request's URL and header for every source.
  * **The defect the third review found, and was right about.** A wrong typed
    Kilo key was accepted with no check when `Options.ProviderOptions` held
    `catalog.WithKiloOrganization("org-x")` directly, outside any `For`, and
    the Result had no organization: the reviewer's probe gave
    `accepted=true err=<nil> profileReads=0 requests=["api.kilo.ai/api/gateway/models hdr="]`,
    the public listing, which authenticates nothing. Each step of its reading
    was verified here, on a scratch copy of the fourth staging: (1) `kiloInputs`
    resolved the check's own options, which were the identity, the endpoint,
    the client and `Options.ProviderOptions`, and found `org-x`, so it decided
    the credential had an organization and read no profile; (2)
    `llmprovider.ResolveOptions` applies every option outside a `For` before
    any `For` (`llmprovider/settings.go:191` to `:213`; its doc says so); (3)
    the listing's options, built by `discoverModels`, also carried the wizard's
    own `For(Kilo, WithKiloOrganization(res.Organization))`, which is a `For`,
    so the Result's empty organization was applied after the direct `org-x`
    and the listing went to the public URL; (4) the probe's first row, on the
    fourth staging, gave the result above. The check and the listing
    disagreed, and neither checked the key. My table of the fourth pass used
    only the `For` form, and the catalog test resolved one option set for both
    sides, so neither could see it.
  * **The fix, by construction:** the check is given the listing's options, and
    they are built in one place, `Options.listingOptions`
    (`wizard/configure.go:853`), the wizard's own `For` among them at its
    actual position. Its two callers: `discoverModels`
    (`wizard/configure.go:899`) and `verifyKilo`
    (`wizard/kilo_verify.go:117`, which passes them to `signinflow.VerifyKilo`
    with no organization argument, since the Result's is in them). A change of
    the listing's options changes the check's. No exported name, and no file
    beyond the wizard, `signinflow` and `kiloendpoint` already touched.
    For `signin.VerifyKilo`, whose `organization` argument has no wizard
    around it, the argument, when not empty, is one more option put where the
    wizard puts the Result's: `catalog.WithKiloOrganization(organization)`
    inside a `For` for Kilo, before every `For` of the caller's and after every
    option outside a `For` (`internal/signinflow/kilo.go:56`); an empty one adds
    nothing, so an organization option of the caller's stands. Its doc says
    that, and what a caller who then lists or builds a provider with the same
    arguments must do to get the listing the decision assumed
    (`llmprovider/x/signin/verify.go:36` to `:50`).
  * **The precedence, stated once, as the listing's real order.** The
    explicit organization is the last `catalog.WithKiloOrganization` applied, an
    empty one included, replacing an earlier one, where `ResolveOptions`
    applies the options outside a `For` first, in order, then those inside
    `For(Kilo, ...)`, in order; the wizard's own `For`, carrying the Result's
    organization (a kept session's, or a device session's choice), is before
    any `For` of `Options.ProviderOptions`: so an option the caller gives inside
    a `For` replaces the Result's, and one given outside a `For` is replaced by
    it, even when the Result's is empty. With no explicit organization, a
    token's embedded URL names one; failing that, the base URL's own path does.
    The listing for the first two sends the organization header, for the third
    none (R-B). The same sentence is in `kiloendpoint.OptionOrganization`'s doc
    (`internal/kiloendpoint/kiloendpoint.go:36` to `:43`), the MADR's D4
    (corrected in place, with a dated line saying the third review found the
    earlier wording false), the README, the CHANGELOG, the architecture and
    `ResolveCredential`'s doc. The earlier wording, which put the Result's
    organization first and said the options replace it, was false.
  * **Tests.** `TestKiloVerify_OptionForms_0080`
    (`wizard/kilo_verify_forms_0080_test.go:25`, 29 subtests): the reviewer's ten
    rows as typed-key rows, each run twice, once with a key the stand-in accepts,
    asserting what the check did (profile requests) and where the listing went
    (URL and organization header), and in every row that either the profile was
    read or the listing was an organization's, never neither; and once with a
    key the service refuses, asserting the refusal; then rows with the Result's
    own organization against each option form (a kept session's: five rows; a
    device session's choice: four rows). Seen failing on the fourth staging
    first (a scratch copy of it with this file and the reviewer's probe files added): 3 of the 29
    subtests failed, "a direct option" with profile requests 0 and want 1 and
    "no profile request and the listing "/api/gateway/models" is not an
    organization's: a wrong key would be accepted", its wrong-key twin with
    "error nil; want a refusal saying "refused 3 keys"", and "a kept session's
    organization and a direct empty option" with profile requests 1 and want 0;
    the other 26 passed there, because the wizard's own `For` reached the check
    only in those forms; all 29 pass on the final staging.
    `TestKiloListing_OrganizationPrecedence_0080`
    (`llmprovider/catalog/kilo_org_0080_test.go:32`, 14 rows): each
    combination of sources, direct and `For` options in each order, a token's
    URL and a base path, the listing's actual request, URL and header, against
    a server, and the organization the check decides from the same resolved
    options, which must agree. `TestVerifyKilo_OrganizationPrecedence_0080`
    (`internal/signinflow/kilo_org_0080_test.go:95`, 17 rows) and
    `TestVerifyKilo_OptionForms_0080`
    (`llmprovider/x/signin/verify_review_0080_test.go:120`, 9 rows): the
    check's decision for the argument against direct and `For` options in each
    order, an empty one, a token's URL, a base path, and a `For` for another
    provider. `TestOptionOrganization_0080`
    (`internal/kiloendpoint/kiloendpoint_test.go:76`): the last option wins, an
    empty one replaces.
    `TestKiloVerify_ProviderOptionOrganizationReadsNoProfile_0080`
    (`wizard/kilo_verify_org_0080_test.go:85`) and
    `TestVerifyKilo_OrganizationFromTheCredential_0080`
    (`llmprovider/x/signin/verify_review_0080_test.go:89`) stay.
  * **Plants** (`--- FAIL` lines from the breaks log): the check given the
    caller's options without the wizard's own (the fourth staging's
    construction): --- FAIL: TestKiloVerify_OptionForms_0080; the two built in
    different orders (the check's reversed): --- FAIL: TestKiloVerify_OptionForms_0080; the
    argument entering after the caller's options: --- FAIL: TestVerifyKilo_OrganizationPrecedence_0080; the
    listing built with one more option than the check: --- FAIL: TestConfigureLLM_KiloListsTheChosenOrganization; the option
    ignored at the decision place: --- FAIL: TestKept_KiloRefusedLeadsToSignIn_0066; the first option
    winning: --- FAIL: TestKiloVerify_OptionForms_0080; an empty option not replacing: --- FAIL: TestKiloVerify_OptionForms_0080; the
    listing ignoring the shared type: --- FAIL: TestConfigureLLM_KiloListsTheChosenOrganization.
* **The listing's option order, pinned for every provider (the fourth
  review's D10 and D9).** Nothing held it outside Kilo: with the caller's
  `WithHTTPClient` moved after `ProviderOptions` in `Options.listingOptions`,
  the whole `wizard` package still passed although a caller's own client then
  stopped winning, and a full move of `ProviderOptions` to the front, which
  changes Claude's and OpenAI's limit, client and workspace, was caught only
  through two Kilo tests. This is a gap that existed at `b109cea` (the
  wizard's listing options were built inline in `discoverModels` with
  `ProviderOptions` last, and no test said so) and that the refactor made
  matter, since the sign-in check is now given the same options. Test, no
  production code changed for it: `TestListingOptions_CallerOptionsComeLast_0080`
  (`wizard/listing_options_0080_test.go:80`, 19 subtests, from the reviewer's
  probe): for Kilo, Claude, OpenAI and a `chatcompat` descriptor, through the
  listing's actual request, a row with no caller option (the wizard's own
  client, endpoint and `ModelLimit` of 2 apply, and no Claude workspace), and
  rows where the caller's own `ProviderOptions` win: its client (every listing
  request goes through it), its endpoint (every request goes to its host), its
  `catalog.WithLimit(1)` (one model recommended, not the wizard's two; not for
  the `chatcompat` descriptor, which refuses `WithLimit` in `ProviderOptions`,
  0038-MADR D5), and a Claude workspace in `For(Claude, ...)` (the
  `Anthropic-Workspace-Id` header on Claude's requests, and on no other
  provider's). Plants, on a scratch copy, whole `wizard` package: the wizard's
  client after the caller's options (D10): `--- FAIL:
  TestListingOptions_CallerOptionsComeLast_0080/kilo,_the_caller's_client`, and
  the same for Claude and OpenAI, and no other test of the package failed, so
  this test alone holds it; `ProviderOptions` moved to the front (the reviewer's
  corrected D9): `--- FAIL: TestListingOptions_CallerOptionsComeLast_0080/kilo,_the_caller's_endpoint`,
  its client, limit and workspace rows for the providers they apply to, and
  the Kilo option-form tests as well.
* **The breaks re-run** on the final tree, on a scratch copy
  (`breaks4.py`): the third reviewer's 43, loaded from its own `breaks3.py`,
  each as it wrote it except where its old string is gone from the final code,
  and then re-formed and named as such in the log (B2b, N5 and N9), plus four
  of this phase's own (F1 to F4, above), all `CAUGHT` by a test failure and
  none by a build failure; the table is in the report to the coordinator.
* **Gates, after the last change to any Go file** (this pass changed one comment of production code and added one test; the fifth pass's race run of six packages, its breaks and its gates stand for the rest, and the whole module was run again), each exit status taken before any filter (`gates.log` of the scratch folder):
  * `vet`: start=13:07:14 end=13:07:16 exit=0
  * `vet-windows`: start=13:07:16 end=13:07:18 exit=0
  * `vet-live`: start=13:07:18 end=13:07:19 exit=0
  * `build-windows`: start=13:07:19 end=13:07:20 exit=0
  * `api-check`: start=13:07:20 end=13:07:26 exit=0 "api-check: against v1.31.0, 0 incompatible change(s) outside llmprovider/x/".
  * `apidiff`: start=13:07:26 end=13:07:31 exit=0 `tools/apidiff.py v1.31.0`: "Compatible changes:" and "- ./llmprovider/x/signin.VerifyKilo: added", nothing else.
  * `docs-check`: start=13:07:31 end=13:07:32 exit=0 "sdkaudit: 0 violation(s) of R50,R51,R52".
  * `tables-check`: start=13:07:32 end=13:07:33 exit=0 "check-tables: 47 packages, 0 problem(s)".
  * `changelog-check`: start=13:07:33 end=13:07:34 exit=0
  * `dep-check`: start=13:07:34 end=13:07:37 exit=0
  * `parity-check`: start=13:07:37 end=13:07:39 exit=0
  * `gofmt`: start=13:07:40 end=13:07:40 exit=0
  * `preadd`: start=13:07:14 end=13:07:40 exit=0 `make pre-add-check` on the 27 changed Go files, through `run-heavy`: "go-precheck: 27 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)".
  * `fullsuite`: start=13:07:14 end=13:08:50 exit=0 `go test -count=1 ./...` through `run-on-host --need any`, which `pick-host` placed on the Windows laptop: 42 packages `ok`, no `FAIL`.
  * `race-wizard`: start=13:07:14 end=13:09:01 exit=0 `go test -race -count=10 -timeout 20m ./wizard/` through `run-heavy` on the session host: `ok`.
  * After the last edit of any record, run again and appended to `gates.log`, each exit 0: `make records-check`, `links-check`, `disclosure-check`, `markdownlint`, `changelog-check`, `tables-check`, `docs-check`; `tools/claims.py` on the four records that hold evidence rows (0080-PLAN 18 of 18, 0080-MADR 67 of 67, 0056-MADR 29 of 29, 0059-MADR 23 of 23; 0017-MADR and 0028-MADR hold none); `tools/denyscan.py --changed`; `tools/recordlint.py --changed` (no rule rose).
* **Not verified:** a live run (none; the owner's runs are Phase 1's); the
  planted breaks in `tools/plant.py`'s form, and the CI set and the full gate,
  which are Phase 4's; that a real Kilo's profile accepts a key valid only for
  an organization; that the organization listing sent without the header
  refuses a wrong key (R-B; the MADR's "Not verified"); the real 10 s of
  `signin.VerifyKilo` (tested at 150 ms, and the constant pinned at 10 s);
  the Linux host for the whole module.
* **Staged, not committed; the coordinator commits.**

### Phase 4 (2026-11-17)

* **Provenance.** A mainstream model at low effort, as the phase prices it, on
  the coordinator's brief. Base commit `bf8e502` (Phase 3's code and tests are
  `00757ed`), level with `origin/main`, tree clean. No source, test or docs
  file changed; this entry is the only write. Staged, not committed.
* **Step 1, `python3 -B tools/plant.py PLANTS.json`** (a scratch clone, 13:57
  to 13:58; exit 1: one plant not caught under the table's own test filter).
  Spec: packages `./wizard/`, `./internal/signinflow/`,
  `./llmprovider/x/signin/`, `-run '_0080$'`. Phase 3 moved the rule to
  `internal/signinflow/kilo.go` and the listing's options to
  `Options.listingOptions`, so the table's breaks are re-formed where marked.
  14 plants (the table's 11, with 2 split in two, and one extra):

  | Break | Failing line | Result |
  | :--- | :--- | :--- |
  | 1 the 403 treated as a refusal (`signinflow.Refused`) | `kilo_0080_test.go:91: refused ... not permitted: kilo HTTP 403 ...; want no refusal, only a cause`; `kilo_verify_0080_test.go:508` | caught |
  | 2a the stall bounded by 100 x `DiscoverLimit`, typed key (re-formed from "30 s": file is `wizard/kilo_verify.go`) | `kilo_verify_0080_test.go:448: took 30.00189625s; want the check bound by DiscoverLimit (300ms), not auth's 30 s` | caught |
  | 2b the same, device session (`wizard/auth.go`, extra) | `kilo_verify_0080_test.go:483: took 30.001403416s; want the session's read bound by DiscoverLimit (300ms)` | caught |
  | 3 the check ignores the organization (`kiloendpoint.Organization(...) != "" && false`) | `kilo_verify_0080_test.go:540: 1 profile reads, organization "org-kept-0080"; want none`; `kilo_verify_forms_0080_test.go:137` | caught |
  | 4 the check runs without Discover | `kilo_verify_0080_test.go:577`, `:588`, `:600` (NoDiscoverNoRead, three cases) | caught |
  | 5 the check runs for every provider | `kilo_verify_0080_test.go:645: request to "opencode.ai/zen/v1/api/profile"; want no Kilo profile read for opencode-zen` | caught |
  | 6a identity left off, re-formed: `Options.listingOptions` builds what the check is given | `kilo_verify_0080_test.go:669: profile request: User-Agent "go-llmprovider-sdk/(devel) ..." ... want the application's name` | caught |
  | 6b identity left off the device session's own profile read (`kiloProfileOptions`, extra) | none under `_0080$` (exit 0). Caught by the existing 0079 test under `-run '_0079$\|_0080$'`: `identity_0079_test.go:149: the Kilo profile: User-Agent "go-llmprovider-sdk/(devel) ..." want it to begin "probe-app/1.2.3 ("` | SURVIVED the table's filter; caught by an existing test outside it |
  | 7 the 401 returned as nil (`KiloVerdict`) | `kilo_verify_0080_test.go:230: ... want "Kilo Gateway refused this key ... enter another" once`; `:262`, `:273` | caught |
  | 8 the cause dropped (`KiloVerdict`) | `kilo_verify_0080_test.go:421: notices ["live model listing ... unavailable ..."]; want "could not verify the Kilo key (not permitted, HTTP 403)" once` | caught |
  | 9 a 200 reported unverified (`KiloVerdict`) | `kilo_0080_test.go:101: refused <nil>, unverified x; want neither` | caught |
  | 10 the 401 returned as nil (`signin.VerifyKilo`) | `verify_0080_test.go:75: error <nil>; want the profile's 401 *APIError matching ErrAuthFailure` | caught |
  | 11a ctx's error dropped (`signin.VerifyKilo`) | `verify_0080_test.go:109: error <nil>; want one matching context.Canceled`; `verify_review_0080_test.go:57` | caught |
  | 11b ctx check before the request removed (`signinflow.VerifyKilo`, extra) | `verify_0080_test.go:109: error <nil>; want one matching context.Canceled` | caught |

  Counts: 14 planted; 13 caught by the `_0080$` tests, 1 (6b, an extra the
  table did not name) not caught by them and caught by
  `TestWizardOwnRequestsNamed_0079`. All 11 of the table's own breaks are
  caught. Whether 6b is a gap for the coordinator: no 0080 test pins the
  device session's profile-read identity; an existing 0079 test does.
* **Step 2, `redgreen.py RED.json --expect fail --rev ceb489e`** (13:59 to
  14:00): exit 0, `7 of 7 fail as expected`. On the tree, `--expect pass`
  refused with `"files" and "drop" need --rev` (exit 2), so it was run with
  the same spec minus `files`: exit 0, `7 of 7 pass as expected`.
* **Step 3** (queued, 14:00 to 14:23, ran 12 s): `run-heavy -- go test -race
  -count=1 -timeout 40m ./internal/signinflow/... ./internal/kiloendpoint/...
  ./llmprovider/x/signin/... ./llmprovider/auth/... ./wizard/...`, exit 0, five
  `ok`. `./internal/kiloendpoint/...` was added by the agent since Phase 3
  moved that package.
* **Step 4** (queued, 14:25 to 15:03, ran 602 s): `run-heavy -- python3 -B
  tools/gate.py --timeout 5400`, exit 0, `gate: 30 of 30 passed`.
* **Step 5** (14:03): `make lint` exit 0 (0 issues); `make vuln` exit 0 (no
  vulnerabilities); `go vet -tags live_gateways ./...` exit 0. The live tests
  were compiled, never run.
* **Step 6** (14:03 to 14:10): `make parity-check dep-check coverage-check
  api-check contract-check generate-check records-check tables-check
  fixtures-check changelog-check docs-check markdownlint links-check
  disclosure-check gate-selftest`, each run on its own, all exit 0;
  `make tools-test` exit 0 (`Ran 107 tests ... OK`).
* **Step 7** (14:06): `claims.py` on the MADR and PLAN exit 0 (67 of 67, 18 of
  18); `denyscan.py --changed` exit 0 (0 hits).
* **Order.** Steps 5 to 7 ran while step 3 waited in the queue; they do not
  depend on it.
* **Not done / not verified.** No live run, no outside request. The plan's
  `--expect pass` on the tree needed a spec without `files`. 6b above is for
  the coordinator's ruling. Phase 5 is not started; this PLAN stays as it was.
* **Staged, not committed.**
