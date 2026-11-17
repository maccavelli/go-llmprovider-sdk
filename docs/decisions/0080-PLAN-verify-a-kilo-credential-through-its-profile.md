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
