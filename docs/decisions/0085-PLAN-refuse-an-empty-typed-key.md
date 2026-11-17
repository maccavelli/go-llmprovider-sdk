---
status: in-progress
date: 2026-11-17
associated-madr: "0085-MADR-refuse-an-empty-typed-key.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Refusing a Typed API Key That Is Empty After Trimming

Associated MADR: [0085-MADR-refuse-an-empty-typed-key.md](0085-MADR-refuse-an-empty-typed-key.md)

## Goal

After this PLAN, in the one release the owner ruled for 0080 and the other
four drafted records (MADR D6, on the owner's word; if this PLAN is not
ready when the word is given, in the release after):

* a typed key, or a pasted credential, that is empty after trimming is not
  returned by `ConfigureLLM` or `ResolveCredential`: the person is asked
  again, up to three asks for each prompt, and the third blank ends the run
  with an error (MADR D2, D3);
* that holds for every provider whose sign-in reaches those prompts, the
  nine built-in providers that require a key and a caller's provider asked
  for a key after a refusal, under any `Prompter`, and a scripted one that
  answers blank for ever ends the run;
* an environment value and a saved key that are empty after trimming are
  absent, as an empty one is (MADR D5);
* the docs say so, and no exported name changes (MADR D8).

The MADR is `proposed`, and so is this PLAN. Nothing below Phase 0 starts
before the owner approves this PLAN. The steps follow the owner's rulings of
2026-11-17 on Q1 to Q6 (the MADR's "Decided"); no question is open. "Ordinary
whitespace" is what `strings.TrimSpace` removes, as the question put to the
owner described it (the MADR's Q1 entry). The phases are unchanged by the
rulings (see "Decisions of 2026-11-17" below).

## Decisions of 2026-11-17

The owner answered the MADR's six questions the same day; the labels are in
the MADR's "Decided". What each does to this PLAN:

* **Q1, Q2, Q3, Q4:** each took the recommended option the steps already
  followed. Every "under Q* (b) or (c)" branch that this PLAN carried has
  been removed from the steps; nothing else changed.
* **Q5:** the release is the one the owner ruled for 0080 and the other four
  drafted records; if this PLAN is not ready when the owner gives the word, it
  moves to the release after. The only change is Phase 3 (below).
* **Q6:** the owner chose a later record that adds an option letting an empty
  key through for Kilo, OpenCode Zen and OpenCode Go, not the recommended
  option of no such path. This PLAN builds nothing for it; its only effect is
  Phase 1 step 4's rule that the refusal stays in `secretKey` and that the doc
  sentences are worded so the later record can amend them.
* **Phases:** unchanged, with their prices.

## Prerequisites

* 0080-PLAN Phase 3's code is committed on `main`, and no other agent holds
  staged or uncommitted work in the tree. This PLAN's Phase 1 starts on that
  commit, called **the base** below.
* The drafting's reproduction, five files named `empty_key_*_0085_test.go`
  (written and run against scratch copies while drafting), is handed to the
  Phase 1 engineer by the coordinator, with the prototype's two scripts. They
  are not committed with the records: the bootstrap exception takes records
  only.

## Relation to 0080 and to the save-on-cancel record

* **Base.** The base is the commit that carries 0080 Phase 3's code, not
  `b109cea`, where the MADR's evidence was taken. 0080's staged work, as read
  on 2026-11-17, does not touch the lines this PLAN changes (`promptAPIKey`,
  `resolveAPIKey`, `keepExistingKey`, `resolveTokenStdin`); it changes
  `credentialStep`'s checks, `wizard/prompter.go`'s package comment, and the
  doc comments of `Options`, `ConfigureLLM` and `ResolveCredential`.
* **What Phase 1 re-reads at the base,** recording each line in the
  execution record before any edit: MADR C1 to C11 and C14 to C17 (the two
  prompts and the two routes); `credentialStep`'s re-ask (C6) and the loop
  around it (C27, C28), which 0080 changes to call `checkCredential`; the
  doc comments of `ConfigureLLM` and `ResolveCredential`; `Prompter.Secret`'s
  doc; the README, architecture and guide places in the Code claims; and
  0080's `verifyKilo`, whose skip of an empty token this PLAN leaves as it is.
* **The save-on-cancel record.** The drafted record "a cancelled sign-in saves
  nothing" changes `credentialStep`'s save, `ResolveCredential`'s last
  context check and the doc comments of `ConfigureLLM`, `ResolveCredential`,
  `credentialStep` and `Options.TokenStore`. The code edits do not overlap.
  The two PLANs both add a sentence to the `ConfigureLLM` and
  `ResolveCredential` docs, a bullet to the README's wizard section, a
  paragraph to the architecture's wizard section and a line under
  `[Unreleased]`. Neither depends on the other; the second to land re-reads
  those places at its own base and merges its sentence. Their gates never run
  at once on one host.
* **The release.** 0080's release also carries the typed "tool schema
  refused" error's record (0080-MADR D11), unless another comes first. The
  owner ruled this record into that one release with 0080 and the other
  four drafted records (MADR D6); the release itself, the tag and the GitHub
  Release are each the owner's word.

## Scope

### In scope

* `wizard/configure.go`: `secretKey`, the one helper that asks for a key;
  `promptAPIKey` calls it; `resolveAPIKey`'s environment test and
  `keepExistingKey`'s saved-key test trim (D5); the doc comments of
  `promptAPIKey`, `resolveAPIKey`, `keepExistingKey`, `ConfigureLLM` and
  `ResolveCredential`.
* `wizard/auth.go`: `resolveTokenStdin` calls `secretKey`; its doc comment.
* `wizard/prompter.go`: the doc comment of `Prompter.Secret` (an empty value
  is not a key; an implementation signals a cancel with an error).
* `wizard/empty_key_0085_test.go` and its companions: the tests.
* Docs: `README.md` (the wizard section, and the line that says Kilo and
  OpenCode accept no key), `docs/architecture.md` ("A refused key", "An
  optional key"), `docs/guides/adding-a-provider.md` (the key step),
  `CHANGELOG.md` `[Unreleased]` **Fixed** (D6).

### Exported names (R55)

None. No exported name is added, removed or changed; `Prompter`'s method set
is the same, as the owner's ruling on the error (a plain wizard error, no
new name) requires. The later record for keyless use may add an export; R55
then asks it be listed with the call site that needs it, and that record
does it.

### Out of scope

* `TextPrompter` (it refuses a blank entry already; MADR R2).
* Any check of a key's shape, inner whitespace or invisible characters
  (MADR D4; the owner kept the refusal to a blank value).
* The option that lets an empty key through for Kilo, OpenCode Zen and
  OpenCode Go: a later record, ruled by the owner (MADR D7). Nothing is built
  for it here: no field, no descriptor fact, no menu entry.
* 0080's checks and `llmprovider/x/signin`.
* gobble-cli (Phase 4 names what it sees).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **One commit: the tests with the code**, as 0080's ruling of 2026-11-17
  made it for 0080 ("One commit: the tests with phase 3's code"), proposed
  here for the owner's approval with the PLAN: the tests are seen red on a
  scratch clone of the base and land with the change in one commit, so
  `main` never carries a failing build.
* **Red, then green.** Each new test the Phase 1 table marks red is seen red
  at the base, and at `b109cea`, with `python3 -B tools/redgreen.py RED.json
  --expect fail --rev <rev>`; every new test, the guards included, is green
  on the phase's tree with `--expect pass`; each is seen to fail on a planted
  break with `python3 -B tools/plant.py PLANTS.json`, on a scratch clone,
  never in the tree. The execution record quotes each failure line. A break
  no test catches is a deviation.
* **Heavy runs queue.** Every `-race` run and the gate run through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` on
  the session host, or through the skill's `scripts/run-on-host` on the host
  `~/.agents/skills/available-dev-hosts/scripts/pick-host` prints. No `-race`
  run goes to a WSL distro. One heavy run per host at a time. A single-package
  `go test` without `-race` may run on the session host.
* **Pre-add.** `make pre-add-check FILES="..."` on every changed Go file.
* **No outside host in a test** (0065-MADR): the `Prompter` is scripted, the
  listing is a `RoundTripper`, the optional-key server is a loopback
  `httptest` server.
* **No credential in a record or a test**: a test's own fixed made-up key
  (`made-up-key-1234`) and the package's existing placeholders only.
* **Stop conditions,** each a stop and a prompt to the owner:
  * an existing test, golden or fixture needs a change (the ones to watch:
    `TestSecret_EmptyReprompts`, the `compat_0066`, `credential_0066`,
    `configure_0028`, `anthropic_key` and `auth` tests that script
    `secrets`);
  * an error's text changes other than as MADR D8 names;
  * `make api-check` or `tools/apidiff.py v1.31.0` reports any change of
    this record's;
  * a new test cannot be red at the base without changing code that is not
    a test;
  * a statement in an earlier record becomes untrue (a dated note, then a
    deviation entry).

## Implementation Steps

### Phase 0: records

Model: lightweight, low.

1. The coordinator places this pair, as the drafting's `PLACEMENT.md` says:
   the number from `python3 -B scripts/check_records.py --next`, the index
   rows by `python3 -B tools/records.py add`, committed under the bootstrap
   exception (records only), after 0080 Phase 3's commit and when no other
   agent holds staged work in the tree.
2. The owner's rulings on Q1 to Q6 are already in the MADR's "Decided", and
   no question is open. `tools/records.py status` sets the MADR `accepted`
   and this PLAN `in-progress`, each with its index row, when the owner says
   so.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --markdown`, then stage and commit (`git commit --no-edit`).

Acceptance: `make records-check links-check` pass; `python3 -B
tools/claims.py` passes on both records; no source changed. Docs: the
records are the docs.

### Phase 1: the tests, the helper and the docs, one commit

Model: mainstream, low. Lightweight was considered and not chosen: the doc
comments must say exactly what the code does, and a test that is red for the
wrong reason needs a judgment, not a rerun. The reproduction and the
prototype exist, so effort stays low; it rises to medium if the base differs
from `b109cea` in the lines Phase 1 re-reads.

Files: `wizard/empty_key_0085_test.go` and its companions (new);
`wizard/configure.go`; `wizard/auth.go`; `wizard/prompter.go`; `README.md`;
`docs/architecture.md`; `docs/guides/adding-a-provider.md`; `CHANGELOG.md`;
this PLAN's execution record.

1. **Re-read the base** (Relation to 0080) and record the lines.
2. **The tests.** Start from the drafting's reproduction, renamed with the
   record's number, keeping the assertion tests and dropping the logging
   ones (their output is in the MADR's evidence). Each lists through
   `testCache(t)` (0058-MADR) and sets `NoEndpointPrompt`. Every test below
   was written and run while drafting, at `b109cea` and on the prototype;
   Phase 1 renames, reviews and runs them again at the base. The words of the
   notice the first test counts are the proposal's (MADR D3) and change with
   the owner's answer.

   | Test | Asserts | At the base and `b109cea` |
   | :--- | :--- | :--- |
   | `TestTypedBlankKeyIsRefused_0085` | for each of the nine providers that require a key and each of six blank values (empty, spaces, a tab, CRLF, a non-breaking space, a mix), a `Prompter` that returns the blank, then a key, is asked twice and the Result carries the key; one notice says no key was entered (words per Q-answered D3) | drafting: red, 54 of 54 cases, "err <nil>, key \"\", Secret asked 1 time(s)" |
   | `TestConfigureBlankKeyIsRefused_0085` | the same through `ConfigureLLM` (Together) | drafting: red |
   | `TestPastedBlankKeyIsRefused_0085` | Grok's pasted key: a blank paste, then a key, is asked twice | drafting: red |
   | `TestForeverBlankPrompterEnds_0085` | a `Prompter` that answers blank for ever is asked `maxTypedKeys` times, and the run ends with an error and no Result (MADR D3) | drafting: red, "asks=1 key=\"\" kind=\"api_key\" err=<nil>" |
   | `TestBlankEnvAndSavedKeyAreAbsent_0085` | a blank `LookupEnv` value and a blank `Existing.APIKey` (four blank values each) are not offered (no `Confirm`) and the typed key is used (MADR D5) | drafting: red |
   | `TestCompatBlankKeyIsNotListed_0085` | an optional-key server that wants "lm-key": a blank answer is never listed with; the server sees `""`, then `Bearer lm-key` | drafting: red, it saw `""`, `""`, `Bearer lm-key` |
   | `TestTextPrompterRefusesBlank_0085` | `TextPrompter`, over a pipe and in raw mode, asks again for spaces, a tab and a non-breaking space (three warnings) and returns the key | green at the base (a guard; MADR R2) |
   | `TestCompatNoKeyNeeded_0085` | a server that takes no key is never asked for one, kind none | green at the base (a guard) |
   | `TestBlankKeyNeverReachesTheListing_0085` | Claude with `Discover` and `claudeRefusals(0, &n)`: a blank answer, then a key, make one listing request, not two, and no "refused this key" notice | drafting: red at `b109cea` |
   | `TestKeyWhitespaceKinds_0085` | a padded key, a non-breaking-space-padded key and a key with a trailing newline are trimmed; a key with inner spaces and a lone zero-width space are returned unchanged (MADR D4) | green at the base (a guard) |
   | `TestKeyPrompterErrorIsReturned_0085` | a `Prompter` whose `Secret` returns a sentinel error ends the run at once, and `errors.Is` matches it, for the typed and the pasted prompt | green at the base (a guard) |

3. **Red.** `python3 -B tools/redgreen.py RED.json --expect fail --rev
   <base>` and `--rev b109cea`, for every test marked red, with the test
   files in `files`; the guards with `--expect pass`. Quote each failure.
4. **The helper** (`wizard/configure.go`), the prototype's shape, 15 lines:

   ```go
   func secretKey(p Prompter, d llmprovider.Descriptor, prompt, wrap string) (string, error) {
   	for asked := 1; ; asked++ {
   		value, err := p.Secret(prompt)
   		if err != nil {
   			return "", fmt.Errorf("%s: %w", wrap, err)
   		}
   		if value = strings.TrimSpace(value); value != "" {
   			return value, nil
   		}
   		if asked == maxTypedKeys {
   			return "", fmt.Errorf("wizard: no %s key entered after %d tries", d.Label, asked)
   		}
   		p.Notify(LevelWarn, "no key was entered; %s needs one", d.Label)
   	}
   }
   ```

   `promptAPIKey` calls it with "enter API key" and keeps `warnForeignKey`;
   `resolveTokenStdin` calls it with "enter credential" and drops its own
   trim. The notice and the error use the words the owner approves with
   this PLAN (MADR D3); the owner's rulings fix their kind, a warning and a
   plain error with no new name, not their words. **Keep the seam** (MADR
   D2, D7): the refusal is the one `value != ""` test in `secretKey`, no
   other code refuses a blank key, `secretKey` is called by exactly
   `promptAPIKey` and `resolveTokenStdin`, and both keep the descriptor and
   `Options` in scope. Add no field, no descriptor fact and no parameter for
   the later record's option, and no code that names Kilo, OpenCode Zen or
   OpenCode Go: nothing is built for it, and a test asserts that all nine
   providers are refused alike.
5. **The two routes** (MADR D5): `resolveAPIKey`'s `envVal != ""` and
   `keepExistingKey`'s `APIKey == ""` test the trimmed value (C8, C10); a
   value that is not blank is offered and used untrimmed, as now.
6. **Docs**, in the same commit:
   * doc comments: `secretKey` (what a blank is, the bound); `promptAPIKey`;
     `resolveAPIKey` and `keepExistingKey` (a blank value is absent);
     `resolveTokenStdin`; `Prompter.Secret` (a value empty once trimmed is not
     a key: the wizard asks again, up to three times, then ends the run; an
     implementation signals a cancel with an error); `ConfigureLLM` and
     `ResolveCredential` (one sentence each: a typed or pasted key that is
     empty once trimmed is asked for again, and the third gives an error);
   * `README.md`: a bullet in the wizard section after the sign-in bullet
     ("A typed or pasted key that is empty once trimmed is asked for again,
     up to three times, then the run ends with an error; an environment value
     or an `Existing` key that is empty once trimmed is not offered"), and in
     the line that says Kilo and OpenCode accept no key, that the wizard asks
     for one and takes no empty one;
   * `docs/architecture.md`: the same facts in "A refused key" and in "An
     optional key, and no default endpoint";
   * `docs/guides/adding-a-provider.md`: the key step ("otherwise `Secret`")
     says an empty answer is asked again;
   * `CHANGELOG.md` `[Unreleased]`: **Fixed** (the owner's ruling on Q5):
     "The wizard returned a typed or pasted key that was empty once trimmed
     as a successful `CredAPIKey` with no key, for every provider that
     requires one, when a caller's `Prompter` returned it. It now asks again,
     up to three times, and then returns an error; an environment value or
     an `Existing` key that is empty once trimmed is no longer offered. The
     package's own `TextPrompter` already did (0085-MADR)";
   * no earlier record's statement becomes untrue by this change (MADR "Prior
     records"); if Phase 1 finds one, a dated note in that record and a
     deviation entry here.
7. **Before staging,** on a scratch clone of the phase's tree: `go test
   -count=1 ./wizard/` (session host, no `-race`); every existing test
   passes unchanged, or the phase stops.

Verification: the tests green; `~/.agents/skills/available-dev-hosts/scripts/run-heavy --
make pre-add-check FILES="wizard/configure.go wizard/auth.go wizard/prompter.go
wizard/empty_key_0085_test.go ..."` (it lints and tests the packages, so it
queues); `make api-check`; `python3 -B tools/apidiff.py v1.31.0`; `make
docs-check tables-check changelog-check links-check records-check
markdownlint`.

Acceptance: every test as its row says; no existing test changed; no API
change; the docs say what the code does, including what stays (inner
whitespace, a lone invisible character).

### Phase 2: planted breaks and the gates

Model: lightweight, medium.

1. `python3 -B tools/plant.py PLANTS.json` on a scratch clone, each plant
   caught, each failure quoted. The drafting ran all eight with its own
   script on copies of the prototype, and each was caught; this phase runs
   them with the repository's tool, on the real change:
   * the trim removed from `secretKey` (caught by the six-value test and by
     `TestKeyWhitespaceKinds_0085`);
   * the bound removed (caught by `TestForeverBlankPrompterEnds_0085`);
   * a blank returned after the bound (the same test);
   * `resolveTokenStdin` calling `Secret` directly, unchecked (caught by
     `TestPastedBlankKeyIsRefused_0085`);
   * the environment test untrimmed (caught by
     `TestBlankEnvAndSavedKeyAreAbsent_0085`);
   * the saved-key test untrimmed (the same test);
   * the notice removed (caught by the notice assertion in
     `TestTypedBlankKeyIsRefused_0085`);
   * `%w` made `%v` in `secretKey`'s wrap (caught by
     `TestKeyPrompterErrorIsReturned_0085`).
2. `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and
   `--expect pass` on the tree, quoted.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race
   -count=1 ./wizard/...`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --timeout 5400`
5. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
6. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
7. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/denyscan.py --changed`; `python3 -B
   tools/recordlint.py --changed`.

Acceptance: every plant caught; every command exit 0, each status taken
before any filter. Docs: none beyond Phase 1's; a gate that finds a docs gap
is a deviation.

### Phase 3: the release (the owner's word)

Model: mainstream, low. No step runs under a standing approval: the tag and
the GitHub Release each wait for the owner's word in the same turn.

1. The owner ruled this record into the one release of 0080 and the other
   four drafted records (MADR D6). On the owner's word, 0080-PLAN Phase 5
   cuts it; this PLAN adds nothing but its changelog entry, already under
   `[Unreleased]`. If Phases 1 and 2 are not done when the word is given, this
   PLAN's entry is not in that release: Phase 3 then waits for the release
   after, by the same steps, and the owner says which version.
2. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`, quoted.
3. `tools/records.py status` sets this PLAN `complete` once the tag's CI and
   step 2 are recorded.

Acceptance: the tag's CI passes; the section and the Release name this fix.

### Phase 4: gobble-cli, after the release (gobble-cli's records)

Not executed under this PLAN; recorded so the hand-off is exact. At
gobble-cli `cecc892`, by a gobble-cli record: `go.mod`'s SDK pin moves to the
release (with 0080's bump, if they share it). Its `Secret` already turns an
empty answer into a cancel (MADR G1), so its terminal path sees no change; a
blank still never reaches the wizard from it. Whether gobble keeps its own
check once the SDK's holds is gobble's choice.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`.

* `go test -count=1 ./wizard/`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` (and
  `--rev b109cea`), then `--expect pass`; `python3 -B tools/plant.py
  PLANTS.json`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race
  -count=1 ./wizard/...`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
  tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN

## Rollout and Rollback

* **Order.** Phase 0 when placed; Phase 1 on approval, on the base; Phase 2
  after it; Phase 3 with 0080's release or on its own, on the owner's word;
  Phase 4 in gobble-cli after the tag.
* **Rollout.** A caller whose `Prompter` never returns a blank key sees
  nothing new. One whose `Prompter` returned blank is asked again, then gets
  an error where it got a `CredAPIKey` with no key; a caller that relied on a
  blank key to get Kilo, OpenCode Zen or OpenCode Go anonymously through its
  own `Prompter` loses that until the later record the owner ruled exists
  (MADR D7, Consequences); a caller wanting anonymous use builds the provider
  without the wizard's key step. `TextPrompter` users see nothing.
* **Rollback before the tag:** revert Phase 1's commit; the records stay,
  with a dated note.
* **Rollback after the tag:** a patch release that restores the two prompts'
  old bodies, by a dated amendment of the MADR. Nothing persistent changes
  shape, so nothing stored needs migrating either way.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, at `b109cea`;
Phase 1 re-reads each at the base.

<!-- claims: self=b109cea -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | `resolveAPIKey` | `wizard/configure.go:739` | `func resolveAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, keySource, error) {` |
| P2 | `keepExistingKey` | `wizard/configure.go:778` | `func keepExistingKey(p Prompter, d llmprovider.Descriptor, o Options) (string, bool, error) {` |
| P3 | `resolveTokenStdin` | `wizard/auth.go:520` | `func resolveTokenStdin(` |
| P4 | `warnForeignKey`, which `promptAPIKey` keeps | `wizard/auth.go:638` | `func warnForeignKey(p Prompter, d llmprovider.Descriptor, key string, o Options) {` |
| P5 | `ConfigureLLM`'s doc ends at its func | `wizard/configure.go:275` | `func ConfigureLLM(ctx context.Context, p Prompter, o Options) (Result, error) {` |
| P6 | `ResolveCredential`'s doc ends at its func | `wizard/configure.go:358` | `func ResolveCredential(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) (Result, catalog.Catalog, error) {` |
| P7 | The test seam: the scripted `Prompter` | `wizard/fake_prompter_test.go:39` | `type fakePrompter struct {` |
| P8 | its constructor | `wizard/fake_prompter_test.go:74` | `func newFake(t *testing.T, f fakePrompter) *fakePrompter {` |
| P9 | the per-test cache | `wizard/fake_prompter_test.go:20` | `func testCache(t *testing.T) llmprovider.Option {` |
| P10 | the raw-mode seam | `wizard/edges_0059_test.go:59` | `func rawSeam0059(t *testing.T, rawErr, restoreErr error) (made, restored *int) {` |
| P11 | a `TextPrompter` over a pipe | `wizard/edges_0059_test.go:81` | `func pipe0059(t *testing.T, typed string) (*TextPrompter, *bytes.Buffer) {` |
| P12 | the optional-key server | `wizard/compat_0066_test.go:34` | `func newCompatServer0066(t *testing.T, n int, key string) *compatServer0066 {` |
| P13 | its options | `wizard/compat_0066_test.go:69` | `func compatOptions0066(t *testing.T) Options {` |
| P14 | Claude's refusing listing | `wizard/configure_0028_test.go:27` | `func claudeRefusals(refusals int32, requests *atomic.Int32) *http.Client {` |
| P15 | a token store | `wizard/auth_test.go:262` | `func newMemoryTokenStore() *memoryTokenStore {` |
| P16 | a notice counter | `wizard/model_select_test.go:69` | `func countContaining(msgs []string, sub string) int {` |
| P17 | The README's wizard bullet | `README.md:388` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| P18 | The README's line on keyless Kilo and OpenCode | `README.md:84` | `Ollama needs no key. Kilo and OpenCode accept none: they then send the` |
| P19 | The architecture's "A refused key" | `docs/architecture.md:1130` | `- **A refused key.** When the listing refuses a typed key, the wizard asks` |
| P20 | and "An optional key" | `docs/architecture.md:1104` | `- **An optional key, and no default endpoint.** A remote provider with no` |
| P21 | The guide's key step | `docs/guides/adding-a-provider.md:282` | `` otherwise `Secret`. `` |
| P22 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| P23 | `Prompter.Secret`'s doc | `wizard/prompter.go:88` | `// Secret reads a credential without echoing it in full. Implementations` |

## Execution record

### Drafting (2026-11-17)

* **Provenance.** Drafted by a mainstream-tier model at high effort, on the
  coordinator's brief, from read-only investigation; nothing implemented,
  nothing written to the repository's tree or index. The MADR and this PLAN
  are `proposed`, with `0085` for the number, which the placing step takes
  from `scripts/check_records.py --next`.
* **The ruling,** the owner's of 2026-11-17, relayed by the coordinator: its
  label is in the MADR's "The owner's ruling of 2026-11-17", with what it
  settles and what it leaves.
* **Verified, and how.**
  * Read at `b109cea` (`git archive` into a scratch folder): the wizard's
    key prompts, environment and saved-key routes, `TextPrompter`, the
    descriptors of every built-in provider, the providers' constructors, the
    README, the architecture, the guides and the records 0020, 0028, 0065,
    0066 and 0080. Read on the index of the tree as it stood
    (`git show :<path>`), not pinned: 0080's `wizard/kilo_verify.go`,
    `internal/signinflow/kilo.go`, `llmprovider/x/signin/verify.go` and its
    tests, and the diffs of `wizard/configure.go`, `wizard/auth.go` and
    `wizard/prompter.go`. Read at gobble-cli `cecc892`: `internal/cli/prompt.go`.
    Read, in the read-only checkouts: codex at `57e6468`, opencode at
    `4ce137b`, kilocode at `58072bd`.
  * Re-run, on scratch copies only: the reproduction's tests at `b109cea`.
    Red: `TestTypedBlankKeyIsRefused_0085` (54 of 54 cases),
    `TestForeverBlankPrompterEnds_0085`, `TestConfigureBlankKeyIsRefused_0085`,
    `TestPastedBlankKeyIsRefused_0085`, `TestCompatBlankKeyIsNotListed_0085`,
    `TestBlankEnvAndSavedKeyAreAbsent_0085` and
    `TestBlankKeyNeverReachesTheListing_0085`. Green (guards):
    `TestTextPrompterRefusesBlank_0085` (both modes),
    `TestCompatNoKeyNeeded_0085`, `TestKeyWhitespaceKinds_0085` and
    `TestKeyPrompterErrorIsReturned_0085`. The logging tests were run and
    read.
  * The prototype, on a copy only: the helper and the two trims. All the
    reproduction's tests pass, and `go test -count=1 ./wizard/` passes whole
    ("ok", about 8 s), the package's existing tests unchanged. Eight planted
    breaks on copies of it were each caught.
  * The line citations of the MADR's index and this PLAN's code claims were
    taken from the cited revisions and checked with `tools/claims.py` from a
    scratch clone (the MADR's result and the PLAN's are in the drafting's
    `PLACEMENT.md`).
* **Not verified.** The ones in the MADR's "Not verified"; any `-race` run;
  any gate; any host but the session's; the tests on 0080's committed code
  (the base does not exist yet; the drafting ran at `b109cea` only, and the
  prototype's edits are to lines 0080's staged diff does not touch).

### Amendment of the drafts (2026-11-17): the owner's rulings

* **What changed.** The owner's rulings on the MADR's Q1 to Q6 are recorded
  (the MADR's "Decided"); the steps carry only the chosen option, and the
  "under Q* (b)" branches are gone. Phases, prices and the test list are
  unchanged: each ruling took the option the steps already followed, except
  Q5 (Phase 3's release wording) and Q6 (a later record, with Phase 1
  step 4's "keep the seam"). A reading of Q1's label, drafted as a seventh
  question, was settled by the question as put to the owner (relayed by the
  coordinator, in the MADR's Q1 entry); no question is open.
* **Re-run after the amendment, on scratch copies only.** The test files'
  invisible zero-width characters were replaced by the `\u200b` escape, which
  has the same value: at `b109cea` seven tests are still red and the four
  guards green; on the prototype the whole `./wizard/` package still passes.
  The records were placed in a fresh scratch clone and checked again
  (`PLACEMENT.md`).
* **Not done.** Nothing for the later keyless option was built or
  prototyped; the MADR's "Not verified" says so.

### Approval (2026-11-17)

* **Approved 2026-11-17.** This PLAN was approved through its live step or
  release by the owner's ruling of 2026-11-17, relayed by the coordinator:
  "Accept all five and run each plan up to its live step or release
  (Recommended)". Its phases are committed and pushed by the coordinator as
  each passes. Tags and releases need the owner's word.
