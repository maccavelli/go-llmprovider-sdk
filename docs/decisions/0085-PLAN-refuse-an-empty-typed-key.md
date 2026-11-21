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
* *(Annotated 2026-11-20.)* The base is no longer the commit that carries
  0080 Phase 3's code (`00757ed`). By the coordinator's build order of
  2026-11-20, 0081, then this record, then 0083, it is the tip of `main` once
  0081-PLAN's code phases (its Phases 2 and 3) are recorded, and 0081's code
  waits on the owner's live run (0081-PLAN Phase 1b). The execution record's
  entry of 2026-11-20 names each file 0081 changes before this PLAN's
  Phase 1 starts. Phase 1 also waits for the owner's answers to the five
  open rows of the MADR's state table (its amendment of 2026-11-20).

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
* *(Added 2026-11-20, second pass.)* `wizard/kilo_verify.go`: the one comment
  at `:95` to `:98` (MADR ST62, the owner's ruling "Reword the comment in 0085
  and note it in 0080's plan (Recommended)"), in Phase 1; the three
  sentences that state the gap today, `README.md:538`, `CHANGELOG.md:57`
  and `docs/architecture.md:1169` (ST70), in Phase 1.
* *(Added 2026-11-20, second pass.)* Phase 1b, the constructors (MADR D9,
  the owner's ruling "Pull it into 0085"): `llmprovider/settings.go` and
  `llmprovider/token.go` (the rule, in `NewStaticToken`, by the owner's
  ruling on ST74); the doc comments of `WithAPIKey`, `StaticToken`,
  `NewStaticToken` and each constructor (no constructor's code); `llmprovider/provider.go`'s
  package example; `README.md`, `docs/architecture.md`, `CHANGELOG.md`
  `[Unreleased]` **Changed**; the tests of Phase 1b.

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
* *(Annotated 2026-11-20, fourth pass.)* Of `llmprovider/x/signin`, only
  `signin.VerifyKilo` stays out of scope (MADR ST74d): its `APIKey` and
  `DeviceKey` sources use `NewStaticToken` and are decided and tested in
  Phase 1b (MADR ST74c).
* gobble-cli (Phase 4 names what it sees).
* *(Annotated 2026-11-20, second pass.)* A provider's constructor given a
  whitespace-only key is no longer out of scope: MADR D9 brings it in
  (Phase 1b). `TextPrompter`'s late line feed (MADR ST69) stays out, found,
  with a follow-up for the owner.

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

   *(Added 2026-11-20.)* This is the phase's first step and is done at the
   phase's own start commit, the base, even though the readiness pass of
   2026-11-20 re-grounded every citation at `24756a6`: 0081's code changes
   `wizard/configure.go`, `wizard/auth.go`, `README.md`,
   `docs/architecture.md` and `CHANGELOG.md` between the two, and that pass's
   lines are not to be reused. Run `python3 -B tools/claims.py` on a scratch
   copy of the two re-pointed tables (the MADR's "The evidence index
   re-pointed at `24756a6`" and this PLAN's "Code claims re-pointed at
   `24756a6`", with plain ids and `self=<base>`), record the result here,
   and re-read 0081's unsaved step and `commitCredential`, where the re-ask
   (C6) and the refusal loop (C27 to C29) now sit, and the docs of
   `ConfigureLLM` and `ResolveCredential` as 0081 leaves them. A row that no
   longer holds is a dated deviation entry before any edit.
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

   *(Added 2026-11-20, second pass.)* Also, in the same commit:
   * the **Fixed** entry and the README bullet say that a key the provider
     refused, followed by three blanks, now ends with the no-key error, which
     does not match `llmprovider.ErrAuthFailure` where the run's error did
     (MADR ST67, the owner's ruling "Confirm D3; add the row and test; note
     it for gobble (Recommended)"); Phase 4's adoption note says it too;
   * the three sentences that state the gap today, `README.md:538`,
     `CHANGELOG.md:57` (0080's entry, still under `[Unreleased]`) and
     `docs/architecture.md:1169`, say instead that a blank key is asked
     again (ST70);
   * `wizard/kilo_verify.go:95` to `:98`: the comment says that the wizard's
     prompts no longer return a blank key (0085), so an empty token reaches
     the check only from a credential with no key at all (ST62);
   * `resolveAPIKey`, `promptAPIKey` and `secretKey` take the run's
     context, and `secretKey` reads it before the second and third ask
     (MADR D10, ST53); their doc comments say so.
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

*(Added 2026-11-20.)* Exit criterion: no blocker, every row tested, plants
caught. "Every row" is each decided row of the MADR's state table (its
amendment of 2026-11-20) that names a Phase 1 test, with the tests the
execution record's entry of 2026-11-20 proposes; the phase does not start
while a row is open.

*(Added 2026-11-20, second pass.)* The owner answered the first pass's five
open rows the same day (MADR, second amendment), so Phase 1 is no longer
held by them; ST73 to ST75, answered in the third pass, concern Phase 1b
only. Step 1's
re-grounding also re-reads `wizard/kilo_verify.go` as 0081 leaves it: by the
owner's ruling on 0081, 0081 changes its notice at `:128` ("could not verify
the Kilo key ... it is saved unchecked") and when warnings are shown, and
this phase's comment edit at `:95` to `:98` is made on that text.

### Phase 1b: a provider's constructor and a whitespace-only key (added 2026-11-20, second pass; outcomes decided in the third pass)

Added by the owner's ruling on MADR ST48, "Pull it into 0085" (MADR D9);
its outcomes are the owner's rulings on ST73 to ST75 (MADR, third
amendment). Model: mainstream, medium.

**What it does.** `llmprovider.NewStaticToken` stores a value that is empty
after `strings.TrimSpace` as `""`, and any other value as given.
`llmprovider.WithAPIKey` already builds its source with `NewStaticToken`
(`llmprovider/settings.go:109`), so the two places the owner chose are one
code change and two doc comments. Every constructor then meets a blank key as
`""` and does what it does with `""` today: Claude, Gemini, Grok, Hugging
Face, OpenAI and Together refuse it at `New` with `ErrInvalidRequest`
("<id> api key is required"); Kilo sends its anonymous key and OpenCode Zen
and Go their public key, keeping a caller's `Header`; Ollama and a Chat
Completions server send no `Authorization`; a caller-registered provider's
constructor receives `""`. No constructor's code changes. A key that is not
blank is sent as given. A `StaticToken` built as a struct literal is not
changed, and its doc comment says so.

**A phase of its own, and when.** It is its own commit, after Phase 1's,
not part of Phase 1's commit. It changes other packages (`llmprovider`, the
doc comments in `llmprovider/providers/...`) and public constructor
behaviour, with a **Changed** entry and the API checks of its own; as its
own commit it can be reverted without the wizard's fix; and its red, green
and plants then run on one change. Nothing in it depends on 0081's code or
on Phase 1's, so it can be built in the same build session right after
Phase 1's commit; it is not folded into Phase 1. It ships in the same
release (MADR D6).

**Files.** `llmprovider/token.go` (`NewStaticToken`'s code and doc, at
`:110` to `:113`; `StaticToken`'s doc, at `:100` to `:102`);
`llmprovider/settings.go` (`WithAPIKey`'s doc, at `:106` to `:107`); the doc
comments above each constructor, and only those, at `24756a6`:
`claude/claude.go:198` to `:202` (`New`), `gemini/gemini.go:142`,
`grok/grok.go:158`, `huggingface/huggingface.go:132`, `kilo/kilo.go:191`,
`openai/openai.go:209`, `together/together.go:128`, `ollama/ollama.go:138`,
`chatcompat/chatcompat.go:168`, `opencode/opencode.go:216` to `:220`
(`NewZen`) and `:225` (`NewGo`); the package example at
`llmprovider/provider.go:17`; `README.md` (`:42`, `:68`, `:84`);
`docs/architecture.md`; `CHANGELOG.md` `[Unreleased]` **Changed**; the test
file `llmprovider/providers/blank_key_0085_test.go` (new).

**The overlap with 0084.** 0084-PLAN's "In scope" (at `24756a6`) changes the
`claude` and `opencode` packages: each send path, which passes its sent tool
list to a new helper (`claude.go`'s `wire.Send` call near `:283`;
`opencode.go`'s near `:322` and its Google declarations near `:765`), and
each package doc, at the head of `claude.go` and `opencode.go`. This phase
edits only the doc comments above `New`, `NewZen` and `NewGo` in those two
files. The two touch the same files and no common line; whichever lands
second re-reads both files at its start and merges (MADR ST80).

1. **Re-ground** at the phase's start commit, and record it: the lines
   above; each constructor's key test (at `24756a6`: `claude.go:213`,
   `gemini.go:157`, `grok.go:173`, `huggingface.go:146`, `kilo.go:205`,
   `openai.go:225`, `opencode.go:240`, `together.go:142`, `ollama.go:149`,
   `chatcompat.go:195` and `:335`); the catalog's `token.Value != ""` tests
   (`catalog/dialect.go:79`, `catalog/discovery.go:1185`, `:1222`,
   `:1419`); `x/signin/credential.go:39` and `:96`; and `claude.go` and
   `opencode.go` as 0084 leaves them if 0084 has landed.
2. **The tests**, one for each of MADR rows ST71 to ST77; ST78 to ST80 are
   checks and steps, named with them:

   | Row | Test or check | Asserts | At the base |
   | :--- | :--- | :--- | :--- |
   | ST71 | `TestConstructorsEmptyKeyUnchanged_0085` | `""` gives each provider's outcome above | green (a guard) |
   | ST72 | `TestConstructorsBlankKeyIsNoKey_0085` | each of the eleven constructors, the ten built-ins through `providers.New` and `chatcompat.New` directly, with each of eight blank values (the wizard tests' six, a lone CR, a lone LF) and an in-process `RoundTripper` that records headers: no request carries the blank value in any header | red for the providers that send a key today |
   | ST73 | `TestBlankKeyIsEmptyKeyPerProvider_0085` | for each provider, each blank value gives exactly the outcome `""` gives: the six refuse at `New` with `ErrInvalidRequest` and the text of `""`; Kilo's anonymous and OpenCode's public key, with a caller's `Header` kept; no `Authorization` from Ollama and a Chat Completions server; a caller-registered constructor in a `Registry` receives `Value` `""` | red for all but Ollama |
   | ST74 | `TestStaticTokenBlankIsEmpty_0085` | `NewStaticToken(v).Value` is `""` for the eight blank values; `WithAPIKey(v)` resolves to a `*StaticToken` whose `Value` is `""`; `catalog.List` given `NewStaticToken("   ")` sends no `Authorization`; `signin.APIKey{Key: "   "}.TokenSource()` yields `""` | red |
   | ST75 | `TestConstructorsPaddedKeySentAsGiven_0085` | `NewStaticToken(" made-up-key ").Value` is unchanged, and Together sends it as given | green (a guard) |
   | ST76 | `TestBlankKeyNoFallback_0085` | with the provider's environment variable set by the test, Claude given `WithAPIKey("   ")` is refused at `New` and Kilo sends its anonymous key: neither uses the variable | red: at the base `NewStaticToken("   ")` keeps the spaces and `claude.go:213` and `kilo.go:205` test only `== ""`, so Claude builds and Kilo sends the spaces |
   | ST77 | `TestWizardAndConstructorAgreeOnBlank_0085` | an environment value of spaces is not offered by `ResolveCredential` under `AllowEnv`, and the same value given to `WithAPIKey` gives the outcome of `""` | red (both halves, before Phases 1 and 1b) |
   | ST78 | `make api-check`; `python3 -B tools/apidiff.py v1.31.0`; `make docs-check` | no exported change; the doc comments, the example, README, architecture and **Changed** entry say what a blank key does, and `StaticToken`'s doc says a struct literal is not changed | — |
   | ST79 | Phase 4's adoption note and the **Changed** entry | what gobble-cli and other callers observe | — |
   | ST80 | step 1 | the lines 0084 and this phase share, re-read | — |

3. **The code:** in `NewStaticToken`, a value that is empty after
   `strings.TrimSpace` is stored as `""`. No other code changes; no exported
   name is added.
4. **Docs**, in the same commit (MADR ST78): the files above; the
   **Changed** entry says that a whitespace-only key given to
   `WithAPIKey` or `NewStaticToken` is no key, with each provider's outcome,
   and that a key that is not blank is sent as given.
5. **Before staging:** `go test -count=1` of each changed package (session
   host, no `-race`); `make api-check`; `python3 -B tools/apidiff.py
   v1.31.0`, which must list no change of this phase's.

Plants, for Phase 2 step 1, each on a scratch clone: **P-k1**, Kilo's
empty-key branch removed (catches ST71's test); **P-k2**, `NewStaticToken`
stores every value as given (catches the tests of ST72, ST73 but Ollama,
ST74 and ST77); **P-k3**, Claude's constructor reads its environment
variable when the key is empty (catches ST76's test); **P-k6**, Ollama puts
its token in `Authorization` (catches ST73's Ollama case); **P-k7**,
`NewStaticToken` trims every value (catches ST75's test); **P-k8**,
`WithAPIKey` builds `&StaticToken{Value: key}` instead of calling
`NewStaticToken` (catches ST74's `WithAPIKey` case).

Acceptance: the tests as above; `TestKeyedProviders_RefuseAnEmptyKey` and
every existing test unchanged; no API change; the docs say what each
constructor does with a blank key. Exit criterion: no blocker, every row
tested, plants caught.

### Phase 2: planted breaks and the gates

Model: lightweight, medium.

*(Added 2026-11-20.)* First step, before step 1: re-ground the cited lines
at this phase's start commit (Phase 1's commit), as Phase 1 step 1 does,
and record the result. Step 1's plants include the sixteen of the execution
record's entry of 2026-11-20.

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

*(Added 2026-11-20.)* Exit criterion: no blocker, every row tested, plants
caught.

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

*(Added 2026-11-20.)* First step: re-ground the changelog entry and the
cited lines at the release commit. Exit criterion: no blocker, every row
tested, plants caught (ST66's entry present; Phases 1 and 2 recorded).

### Phase 4: gobble-cli, after the release (gobble-cli's records)

Not executed under this PLAN; recorded so the hand-off is exact. At
gobble-cli `cecc892`, by a gobble-cli record: `go.mod`'s SDK pin moves to the
release (with 0080's bump, if they share it). Its `Secret` already turns an
empty answer into a cancel (MADR G1), so its terminal path sees no change; a
blank still never reaches the wizard from it. Whether gobble keeps its own
check once the SDK's holds is gobble's choice.

*(Added 2026-11-20, second pass.)* The adoption note also says, by the
owner's ruling "Confirm D3; add the row and test; note it for gobble
(Recommended)": a key the provider refused, followed by three blank
answers, now ends with the wizard's no-key error, which does not match
`llmprovider.ErrAuthFailure`, so gobble's report of the keys asked for
(`llm/provider/signin.go:372` to `:374` at gobble-cli `b1994a0`) is not
filled for that run (MADR ST67). And, from Phase 1b: a blank environment
variable that gobble passes to `WithAPIKey` (`llm/provider/provider.go:218`
to `:219`; its `ambient.go:31` tests only `!= ""`) is no longer sent as a
key: for Claude, Gemini, Grok, Hugging Face, OpenAI and Together, `New`
refuses it with `ErrInvalidRequest`; for Kilo and OpenCode, the anonymous
or public key is sent (MADR ST73, ST79).

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

### Readiness pass (2026-11-20): the test map, the proposed tests and the re-grounding

A records pass before any code, at `24756a6`, the tip of `main`. Nothing
above is rewritten; the steps it touches carry dated annotations. The MADR's
amendment of 2026-11-20 holds the state table (rows ST1 to ST66) this entry
maps.

#### Where the PLAN stands

* **Phase 0 is done; no later phase has started.** The records were placed
  and accepted in `a3002aa` and `8f133db`, which this record did not log
  as a Phase 0 entry. No commit after them names 0085, no `wizard/*0085*`
  file exists, and no `secretKey` exists at `24756a6`. Phases 1 to 3 are
  unbuilt; Phase 4 is gobble-cli's and is not run under this PLAN.
* **The base and the order** are in "Prerequisites", annotated 2026-11-20:
  0081's code first, then this PLAN's Phase 1.
* **Phase 1 waits for the owner** on the state table's five open rows
  (ST12, ST13, ST15, ST53, ST62). An answer that adds a test adds it to
  this map by a dated entry before the phase starts.
* **Pass numbering.** Each run of a phase is recorded here as "Phase N, pass
  M", from pass 1, with its own verification output; a third pass of one
  phase is a deviation and stops for the owner.

#### What 0081 will have changed before Phase 1 starts

From 0081-PLAN's "In scope" and Phase 2 "Files" at `24756a6`; each is
re-read at the base by Phase 1 step 1, not taken from this list:

* `wizard/configure.go`: `credentialStep` becomes an unsaved step followed
  by `commitCredential` (one context check, then `save`), so the refusal
  loop and the re-ask call of `promptAPIKey` (MADR C6, C7, C27 to C29) move
  into the unsaved step; `ResolveCredential` loses its own context check
  after the step (C33's second site); `ConfigureLLM` calls the unsaved step,
  checks the context once after it, and commits before each of its two
  successful returns (C25's neighbourhood); the doc comments of
  `credentialStep`, `ConfigureLLM`, `ResolveCredential` and
  `Options.TokenStore`.
* `wizard/auth.go`: the doc comments of `resolvedCredential.save` and of
  `saveTo`. `resolveTokenStdin`, `signIn`'s key branches and
  `warnForeignKey` are not in 0081's scope.
* `wizard/cancel_save_0081_test.go` and `wizard/save_at_end_0081_test.go`
  (new).
* `llmprovider/x/signin/signin.go`: one sentence in `Wait`'s doc.
* `README.md` (the wizard's sign-in bullet, which this PLAN's new bullet
  follows), `docs/architecture.md` ("When a session is saved" and the
  wizard's paragraph on the save), `CHANGELOG.md` `[Unreleased]` (a
  **Fixed** and a **Changed** entry beside this PLAN's **Fixed**).
* Dated notes in 0059-MADR, 0056-MADR, 0080-MADR and 0080-PLAN.
* Not in 0081's scope, though the program brief lists it among the shared
  files: `wizard/kilo_verify.go` (0081 does not change it; this PLAN
  touches it only if the owner answers ST62 so).

#### The test map

Every decided row of the state table and its test. "PLAN" is a test of
Phase 1 step 2's table; "proposed" is added below.

| Rows | Test | Phase | Status |
| :--- | :--- | :--- | :--- |
| ST1, ST2, ST4, ST26, ST29, ST52, ST63, ST64, ST65 | `TestTypedBlankKeyIsRefused_0085` | 1 | PLAN |
| ST3, ST23 | `TestTypedBlankKeyIsRefused_0085`, values `"\r"` and `"\n"` | 1 | proposed |
| ST27 | `TestTypedBlankKeyIsRefused_0085`, the menu providers named | 1 | proposed |
| ST26, ST58 | `TestConfigureBlankKeyIsRefused_0085` | 1 | PLAN |
| ST32, ST64 | `TestPastedBlankKeyIsRefused_0085` | 1 | PLAN |
| ST33, ST64 | `TestPastedBlankKeyIsRefused_0085`, an OpenAI case | 1 | proposed |
| ST11, ST19, ST65 | `TestForeverBlankPrompterEnds_0085` | 1 | PLAN |
| ST14 | `TestForeverBlankPrompterEnds_0085`, four `errors.Is` assertions | 1 | proposed |
| ST37, ST41, ST64 | `TestBlankEnvAndSavedKeyAreAbsent_0085` | 1 | PLAN |
| ST38, ST42 | `TestBlankEnvAndSavedKeyAreAbsent_0085`, `""` and an optional-key case | 1 | proposed |
| ST39, ST43 | `TestNonBlankEnvAndSavedKeyUntrimmed_0085` | 1 | proposed |
| ST44, ST45, ST46, ST59 | `TestBlankAfterDecliningSavedKey_0085` | 1 | proposed |
| ST28, ST30 | `TestCompatBlankKeyIsNotListed_0085` | 1 | PLAN |
| ST31 | `TestCompatNoKeyNeeded_0085`; the existing `TestOllama_RefusedNoKeyAsked_0066` | 1 | PLAN; existing |
| ST21 | `TestTextPrompterRefusesBlank_0085` | 1 | PLAN |
| ST20 | `TestTextPrompterRefusesBlank_0085`, an end-of-input case | 1 | proposed |
| ST51 | `TestBlankKeyNeverReachesTheListing_0085` | 1 | PLAN |
| ST51 | `TestBlankKeyNeverReachesTheListing_0085`, a Kilo case | 1 | proposed |
| ST5, ST7, ST8 | `TestKeyWhitespaceKinds_0085` | 1 | PLAN |
| ST5, ST6, ST7, ST9, ST10 | `TestKeyWhitespaceKinds_0085`, ten values added | 1 | proposed |
| ST16, ST22, ST24 | `TestKeyPrompterErrorIsReturned_0085` | 1 | PLAN |
| ST17, ST18 | `TestKeyPrompterErrorIsReturned_0085`, two cases | 1 | proposed |
| ST57 | every Phase 1 test, in CI's three-system matrix | 2 | PLAN (CI) |
| ST58 | Phase 1 step 1, at 0081's commit | 1 | PLAN, annotated |
| ST66 | `make changelog-check`; Phase 1's review of the entry | 1 | PLAN |

**PLAN tests that map to no row:** none. Each of the eleven tests of
Phase 1 step 2 maps to at least one decided row above.

**Rows with no test until the owner answers:** ST12, ST13 (the words are
asserted by `TestTypedBlankKeyIsRefused_0085` and
`TestForeverBlankPrompterEnds_0085` once fixed), ST15, ST53, ST62.

#### The proposed tests (dated addition to Phase 1, 2026-11-20)

Each is written with Phase 1's tests, under the same rules (in-process
stand-ins, `testCache(t)`, `NoEndpointPrompt`, a made-up key). Each has a
**plant**: the single production change that must turn it red, run in
Phase 2 step 1 with `tools/plant.py` on a scratch clone, never in the tree.
Where the test is red at the base without a plant, `tools/redgreen.py
--expect fail` shows that too; a guard green at the base is held by its
plant alone.

1. **`TestTypedBlankKeyIsRefused_0085`, two values** (ST3, ST23): `"\r"` and
   `"\n"` join the six, each asked again. Plant P-a: `secretKey` trims with
   `strings.Trim(value, " \t ")` in place of `strings.TrimSpace`.
2. **`TestTypedBlankKeyIsRefused_0085`, the menu named** (ST27): the test
   asserts, for Grok, Kilo and OpenAI, that the method menu was asked and the
   API key chosen, and for the six others that it was not. Plant P-b:
   `signIn`'s menu branch (C13) asks `p.Secret` itself in place of
   `resolveAPIKey`.
3. **`TestKeyWhitespaceKinds_0085`, ten values** (ST5, ST6, ST7, ST9,
   ST10): `"made-up-key\r\n"` gives `"made-up-key"`; `"\"made-up-key\""`,
   `"'made-up-key'"` and `"\"\""` are returned unchanged; `"made-up\nkey"`,
   `"made-up\x00key"` and `"made-up\x1bkey"` unchanged; one key of 1 MiB of
   letters unchanged; `"\xff"` and `"\xa0"` unchanged. Plants: P-c,
   `strings.Trim(value, " \n")` in place of `TrimSpace` (the CR+LF value);
   P-d, quotes stripped after the trim; P-e, a value holding a control
   character treated as blank; P-f, a value over 4096 bytes treated as
   blank; P-g, `strings.ToValidUTF8(value, "")` before the blank test.
4. **`TestForeverBlankPrompterEnds_0085`, four assertions** (ST14):
   `errors.Is` is false for `llmprovider.ErrAuthFailure`,
   `llmprovider.ErrInvalidRequest`, `context.Canceled` and `io.EOF`. Plant
   P-h: the no-key error wraps `llmprovider.ErrInvalidRequest` with `%w`.
5. **`TestKeyPrompterErrorIsReturned_0085`, two cases** (ST17, ST18): a blank
   then a sentinel error gives one notice and the error, matched by
   `errors.Is`, after two asks; a value returned with the sentinel gives the
   error and no Result. Plants: P-i, `secretKey` treats a `Prompter` error
   as a blank while asks remain; P-j, `secretKey` tests the value before the
   error.
6. **`TestTextPrompterRefusesBlank_0085`, end of input** (ST20): through
   `ResolveCredential`, a pipe holding `"   \n"` and then nothing gives one
   `TextPrompter` warning and an error matching `io.EOF`, after one wizard
   ask. Plant: P-i (the error then reaches the bound as blanks, and the
   no-key error does not match `io.EOF`).
7. **`TestPastedBlankKeyIsRefused_0085`, OpenAI** (ST33): a blank paste,
   then a made-up `sk-` key, gives that key after two asks and no
   access-token error. Plant P-k: `resolveTokenStdin` asks `p.Secret`
   itself for OpenAI.
8. **`TestBlankEnvAndSavedKeyAreAbsent_0085`, two cases** (ST38, ST42): an
   environment value `""` (if its four values leave it out) is not offered;
   a Chat Completions server with a blank `Existing.APIKey` asks no
   `Confirm` and is listed with no key. Plants: P-l, the environment's
   emptiness test removed (every value offered); the drafting's own
   "saved-key test untrimmed" (Phase 2 step 1) for the second case.
9. **`TestNonBlankEnvAndSavedKeyUntrimmed_0085`** (new; ST39, ST43): an
   environment value `"made-up-key\n"` that is kept is the Result's
   `APIKey` with its line feed; an `Existing.APIKey` of `" made-up-key "`
   that is kept is returned as it is. Plants: P-m, the environment value
   trimmed on return; P-n, the saved key trimmed on return.
10. **`TestBlankAfterDecliningSavedKey_0085`** (new; ST44 to ST46, ST59): (a)
    a valid `Existing.APIKey`, declined, then three blanks: the no-key
    error, no Result, three `Secret` calls; (b) declined, a blank, then a
    new key: the new key; (c) a valid saved session in a memory
    `TokenStore`, declined, the API key method, three blanks: the no-key
    error, no save, and the store's session loads as it was. Plant P-o: on
    the no-key error, `promptAPIKey` returns `o.Existing.APIKey` when one is
    set.
11. **`TestBlankKeyNeverReachesTheListing_0085`, Kilo** (ST51): under
    `Discover`, a blank then a key: Kilo's profile stub sees one read, with
    the key, and none without it. Plant P-p: `secretKey` returns its value
    whatever it is (the refusal removed).

Sixteen plants, P-a to P-p, beside the drafting's eight of Phase 2 step 1.

#### Code claims re-pointed at `24756a6`

The "Code claims" above stay at `b109cea`. This table is beside them, one
row per row, ids ending in "now" so that `tools/claims.py` does not read
them against `b109cea`. Checked by `tools/claims.py` on a scratch copy with
plain ids and `self=24756a6`: 23 of 23 pass. Tally: **13 unchanged**, **9
moved**, **1 moved with changed text** (P19: still true in substance).
P5 and P6 are the functions 0081 changes; Phase 1 step 1 re-reads them at
the base.

| Id | Claim | Now | Location at `24756a6` | Text |
| :--- | :--- | :--- | :--- | :--- |
| P1now | `resolveAPIKey` | moved | `wizard/configure.go:801` | `func resolveAPIKey(p Prompter, d llmprovider.Descriptor, o Options) (string, keySource, error) {` |
| P2now | `keepExistingKey` | moved | `wizard/configure.go:840` | `func keepExistingKey(p Prompter, d llmprovider.Descriptor, o Options) (string, bool, error) {` |
| P3now | `resolveTokenStdin` | moved | `wizard/auth.go:596` | `func resolveTokenStdin(` |
| P4now | `warnForeignKey`, which `promptAPIKey` keeps | moved | `wizard/auth.go:714` | `func warnForeignKey(p Prompter, d llmprovider.Descriptor, key string, o Options) {` |
| P5now | `ConfigureLLM`'s doc ends at its func | moved | `wizard/configure.go:318` | `func ConfigureLLM(ctx context.Context, p Prompter, o Options) (Result, error) {` |
| P6now | `ResolveCredential`'s doc ends at its func | moved | `wizard/configure.go:415` | `func ResolveCredential(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) (Result, catalog.Catalog, error) {` |
| P7now | The test seam: the scripted `Prompter` | unchanged | `wizard/fake_prompter_test.go:39` | `type fakePrompter struct {` |
| P8now | its constructor | unchanged | `wizard/fake_prompter_test.go:74` | `func newFake(t *testing.T, f fakePrompter) *fakePrompter {` |
| P9now | the per-test cache | unchanged | `wizard/fake_prompter_test.go:20` | `func testCache(t *testing.T) llmprovider.Option {` |
| P10now | the raw-mode seam | unchanged | `wizard/edges_0059_test.go:59` | `func rawSeam0059(t *testing.T, rawErr, restoreErr error) (made, restored *int) {` |
| P11now | a `TextPrompter` over a pipe | unchanged | `wizard/edges_0059_test.go:81` | `func pipe0059(t *testing.T, typed string) (*TextPrompter, *bytes.Buffer) {` |
| P12now | the optional-key server | unchanged | `wizard/compat_0066_test.go:34` | `func newCompatServer0066(t *testing.T, n int, key string) *compatServer0066 {` |
| P13now | its options | unchanged | `wizard/compat_0066_test.go:69` | `func compatOptions0066(t *testing.T) Options {` |
| P14now | Claude's refusing listing | unchanged | `wizard/configure_0028_test.go:27` | `func claudeRefusals(refusals int32, requests *atomic.Int32) *http.Client {` |
| P15now | a token store | unchanged | `wizard/auth_test.go:262` | `func newMemoryTokenStore() *memoryTokenStore {` |
| P16now | a notice counter | unchanged | `wizard/model_select_test.go:69` | `func countContaining(msgs []string, sub string) int {` |
| P17now | The README's wizard bullet | moved | `README.md:434` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| P18now | The README's line on keyless Kilo and OpenCode | unchanged | `README.md:84` | `Ollama needs no key. Kilo and OpenCode accept none: they then send the` |
| P19now | The architecture's "A refused key" | moved, text changed | `docs/architecture.md:1249` | `- **A refused key.** When the listing, or for a Kilo credential used with` |
| P20now | and "An optional key" | moved | `docs/architecture.md:1223` | `- **An optional key, and no default endpoint.** A remote provider with no` |
| P21now | The guide's key step | unchanged | `docs/guides/adding-a-provider.md:282` | `` otherwise `Secret`. `` |
| P22now | The changelog's open section | unchanged | `CHANGELOG.md:12` | `## [Unreleased]` |
| P23now | `Prompter.Secret`'s doc | moved | `wizard/prompter.go:94` | `// Secret reads a credential without echoing it in full. Implementations` |

### Readiness pass, second (2026-11-20): the owner's rulings, the review's findings, and Phase 1b

A records pass the same day, still before any code, at `24756a6`. It applies
an independent review of the first pass and the owner's rulings on its open
rows, relayed by the coordinator; the MADR's "Amendment of 2026-11-20, second
pass" records both, with each label. Nothing above is rewritten.

#### What changed in this PLAN

* **Phase 1 is no longer held** by open rows: the owner answered ST12, ST13,
  ST15, ST53 and ST62. Its scope gains `wizard/kilo_verify.go`'s one comment
  (ST62), the three docs sentences of ST70, the changelog and README words of
  ST67, and the context threaded through `resolveAPIKey`, `promptAPIKey` and
  `secretKey` (MADR D10), each by a dated annotation in "Scope" and step 6.
* **Phase 1b is added**, for MADR D9 (the constructors), with its reason for
  being a phase of its own. It waits on ST73 to ST75.
* **Phase 1's re-grounding** names `wizard/kilo_verify.go` as 0081 leaves it
  (its notice at `:128` and when warnings are shown, by the owner's ruling on
  0081).
* **Phase 4's adoption note** names what gobble-cli observes (ST67, ST79).
* **0080-PLAN** carries one dated note under "Found, not changed by 0080",
  pointing here (ST62's ruling); it is the only edit outside this pair.

#### The test map, corrected

Rows the first pass mapped to a test that could not pin them, each now with
its test and the plant that turns it red. "P-" names are the plants run in
Phase 2 step 1, on a scratch clone, never in the tree.

| Row | Test | Change | Plant |
| :--- | :--- | :--- | :--- |
| ST11, ST19 | `TestForeverBlankPrompterEnds_0085` | asserts exactly `maxTypedKeys` asks and two notices, not "at most" | P-r: the bound shortened to two asks; P-s: the notice also given on the third blank |
| ST12, ST13 | `TestForeverBlankPrompterEnds_0085`; `TestTypedBlankKeyIsRefused_0085` | asserts the error's whole text and the notice's | P-txt: the error's words changed; the drafting's "notice removed" |
| ST65 | `TestForeverBlankPrompterEnds_0085` | runs for all nine providers that require a key | P-t: at the bound, a blank returned for Kilo, OpenCode Zen and OpenCode Go |
| ST27 | `TestTypedBlankKeyIsRefused_0085` | Kilo run twice, without a `TokenStore` (no menu) and with a memory one (menu); the test asserts which runs asked the method menu | P-b: `signIn`'s menu branch asks `p.Secret` itself |
| ST22 | `TestKeyPrompterErrorIsReturned_0085` | a raw-mode case through `rawSeam0059` and a Ctrl-C byte, matched with `context.Canceled` | P-u: `secretKey` treats an error matching `context.Canceled` as a blank |
| ST21 | `TestTextPrompterRefusesBlank_0085` | asserts three `TextPrompter` warnings and no wizard notice | P-v: `TextPrompter.Secret` returns an empty value instead of asking again |
| ST31 | `TestCompatNoKeyNeeded_0085`; the existing `TestOllama_RefusedNoKeyAsked_0066` | unchanged | P-w: `resolveAPIKey`'s optional-key branch asks the typed prompt; P-x: its local branch does |
| ST44 | `TestBlankAfterDecliningSavedKey_0085` | case (d) added: a valid environment value under `AllowEnv`, declined, three blanks | P-o (case a, `Existing`); P-o2: on the no-key error, `resolveAPIKey` returns the environment value it offered |
| ST46, ST59 | `TestBlankAfterDecliningSavedKey_0085` | case (c) asserts the error and that the memory store saw no `Save` and no `Delete` | P-q: a blank returned after the bound (the drafting's plant); P-y: `keepExistingOAuth` deletes a session the person declines |

#### The new tests and plants (dated addition to Phase 1, 2026-11-20, second pass)

1. **`TestRefusedKeyThenBlanks_0085`** (MADR ST67, ST15). (i) Claude under
   `Discover` with `claudeRefusals(1, &n)`: a typed key refused, then three
   blanks: the no-key error, `errors.Is(err, llmprovider.ErrAuthFailure)`
   false, one listing request, four `Secret` calls. (ii) An optional-key
   Chat Completions server that refuses no key, then three blanks: the
   same error, not `ErrAuthFailure`. (iii) A kept OpenAI session that the
   listing refuses, the API key method, three blanks: the same. (iv) The
   count: key one refused, two blanks, key two refused, two blanks, key
   three refused: seven `Secret` calls and `refusalError`, which matches
   `ErrAuthFailure`. Plants: **P-auth**, `credentialStep` returns
   `refusalError(d, credential, keys, listErr)` when the prompt after a
   refusal fails (catches i and ii); **P-auth2**, after a refused kept
   session, `credentialStep` wraps the listing's error into the failed
   sign-in's (catches iii); **P-cnt**, one blank count kept across
   calls of the typed prompt (catches iv).
2. **`TestCancelBeforeRepeatAsk_0085`** (MADR D10, ST53). A `Prompter`
   whose first `Secret` ends the run's context and returns a blank: one
   `Secret` call, an error matching `context.Canceled`, for the typed and the
   paste prompt; the same with a deadline that passes, matching
   `context.DeadlineExceeded`. Plant **P-ctx**: the check removed.
3. **`TestTypedBlankKeyIsRefused_0085`, a caller's descriptor** (ST68): a
   `Registry` holding a descriptor with `RequiresAPIKey`, an id of the
   test's own and a stand-in constructor: a blank, then a key, is asked
   twice. Plant **P-reg**: the refusal applies only to an id the default
   registry holds.
4. Phase 1b's three tests and plants P-k1 to P-k3, in Phase 1b.

With the first pass's sixteen (P-a to P-p) and the drafting's eight, Phase 2
step 1 runs every plant named here: P-auth, P-auth2, P-b, P-cnt, P-ctx, P-o2, P-q,
P-r, P-reg, P-s, P-t, P-txt, P-u, P-v, P-w, P-x, P-y and, after Phase 1b,
P-k1 to P-k3.

**PLAN tests that map to no row:** none. The rows still without a test are
the open ones, ST73 to ST75, and ST69 (out of scope).

#### The review's findings, one line each

* Case 1 (refused key, then blanks): held; applied as ST67 and test 1, by
  the owner's ruling.
* Case 2 (constructors and a key of spaces): held; applied as MADR D9,
  Phase 1b and ST71 to ST80, by the owner's ruling.
* Case 3 (a caller's descriptor): held; applied as ST68 and test 3.
* Case 4 (`TextPrompter`'s late line feed): held; out of scope, ST69, with a
  follow-up for the owner.
* Question 3, seven test gaps: each held against the drafting's tests as
  handed over; each applied in the corrected map above.

#### Gaps found by this pass, beyond the review

* **The docs state the gap as it is today** (`README.md:538`,
  `CHANGELOG.md:57`, `docs/architecture.md:1169`), and would be untrue after
  Phase 1; the first pass's docs step did not name them. Applied as ST70.

### Readiness pass, third (2026-11-20): ST73 to ST75 answered

The owner answered the three rows the second pass left open, relayed by the
coordinator: "Same as an empty key today, provider by provider
(Recommended)", "In `WithAPIKey` and `NewStaticToken` (Recommended)" and
"Sent as given (Recommended)" (MADR, third amendment). No row is open.

* **Phase 1b** now states its outcomes, files, tests (one for each of rows
  ST71 to ST77, with checks for ST78 to ST80) and plants. Its section was
  this session's own unstaged text and was rewritten in place; the second
  pass's "waits on ST73 to ST75" no longer holds.
* **Plants added:** P-k6 (Ollama's case), P-k7 (ST75), P-k8 (`WithAPIKey`
  bypassing `NewStaticToken`). P-k2 is now "`NewStaticToken` stores every
  value as given".
* **Build order.** Phase 1 follows 0081's code ("Prerequisites", annotated).
  Phase 1b is its own commit after Phase 1's; it depends on neither 0081's
  code nor Phase 1's, so it may be built in the same session right after
  Phase 1's commit, but not in Phase 1's commit.
* **The overlap with 0084.** 0084-PLAN changes the `claude` and `opencode`
  send paths and package docs; Phase 1b changes only the doc comments above
  `New` (`claude.go:198` to `:202`), `NewZen` (`opencode.go:216` to `:220`)
  and `NewGo` (`opencode.go:225`). No common line; the second to land
  re-reads both files.

### Readiness pass, fourth (2026-11-20): corrections

Corrections from an independent confirmation; no ruling changes. Phase 1b's
table now marks ST76's test red at the base (`NewStaticToken("   ")` keeps
the spaces, and `claude.go:213` and `kilo.go:205` test only `== ""`); the
other labels were re-checked and hold (ST71 and ST75 green, the rest red).
ST72's row names `chatcompat.New` apart from `providers.New`'s ten.
"Out of scope" is narrowed to `signin.VerifyKilo` by a dated annotation.
The MADR's "Corrections of 2026-11-20, fourth pass" adds rows ST53b, ST53c,
ST74e, ST74f and ST81, none with a test (each out of scope), and the final
count: 91 rows, 70 decided, 21 out of scope, 0 open.
