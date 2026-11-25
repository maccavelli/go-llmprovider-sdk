---
status: in-progress
date: 2026-11-25
associated-madr: "0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement: Fill a Completed Responses Stream From Its Terminal Output

Associated MADR: [0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md)

## Goal

After this PLAN, the Responses stream reader takes, at `response.completed` as at a cut, each item the terminal response carries that no `output_item.done` delivered, paired by id, then by `output_index` within one type, then in order within one type, and taking nothing of a type whose pairing is ambiguous ("Take nothing of that kind (Recommended)"). A `.done` that added nothing gives way to its terminal partner ("Use summary when done was empty (Recommended)"); a `.done` that added an item stands against a differing terminal copy ("The item's own message (Recommended)"). An announced message, call or client tool search that neither delivered fails the reply as a retryable broken stream, `item_not_finished` ("Fail as retryable stream (Recommended)"). A stream that took items writes one debug line through `WithLogger` ("Debug line via WithLogger (Recommended)"). It rides in v1.33.0, as its own phase between 0088-PLAN Phase 3's review and 0091-PLAN Phase 2 ("Own v1.33.0 phase (Recommended)").

The MADR is `accepted` (2026-11-24, the owner's answers to OQ1 to OQ6; 0 rows open). This PLAN is `proposed`. Phase 0's steps 1 to 4 place the records in the tree, which the bootstrap rule allows with no prior approval; its step 5 is the owner's approval of this PLAN. No phase after Phase 0 starts until the owner approves this PLAN and says `proceed`, `execute the plan` or `do phase N`.

*(2026-11-25, placed in the tree at `138d52f` (v1.33.0), `in-progress`. The owner's rulings of 2026-11-24, relayed by the coordinator, quoted as given: OQ1 "Fail as retryable stream (Recommended)", OQ2 "Use summary when done was empty (Recommended)", OQ5 "Take nothing of that kind (Recommended)", OQ6 "The item's own message (Recommended)", OQ3 "Debug line via WithLogger (Recommended)", OQ4 "Own v1.33.0 phase (Recommended)", and, for this PLAN, "Approve; follow the queue (Recommended)". The owner's "Yes, re-scope v1.33.0 (Recommended)" then moved this work from v1.33.0 to v1.34.0 ([0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md), seventh amendment, its 0092 note). The owner's build ruling of 2026-11-25, relayed by the coordinator, quoted exactly: "SDK: build 0092 (Recommended)". Read as follows; the steps keep their text. The Goal's "rides in v1.33.0": its own phase in v1.34.0. The Order rule: Phase 0's wait is met (`0c5e143` and `33e6f09` landed before `138d52f`); "after 0088-PLAN Phase 3's review" is met by v1.33.0's short release review, but 0088's R1 to R3 that Phase 1's step 1 reads are not ruled until v1.34.0's review, which the MADR puts to the owner as Q-B1 (its amendment of 2026-11-25), not resolved here. Phase 2 step 2: `tools/apidiff.py v1.33.0`, the newest tag, where it says `v1.32.0`. Phase 3: 0091-PLAN Phase 4's joint review is of v1.34.0, and the release is v1.34.0's. Rollout: v1.34.0. See the execution record's entry of 2026-11-25.)* *(2026-11-25, second: Q-B1 answered, quoted exactly: "Rule R1–R3 first". The Order rule reads: [0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md) Phase 6 (R1 by wire) first, then Phase 1 here, on Phase 6's commit, then 0091-PLAN Phase 2. Phase 1's step 1 reads 0088's rulings as 0088-MADR ST107 (R1), ST108 (R2) and ST109 (R3), all decided. None changes what ST10, ST11 or ST13 assume (the MADR's amendment of 2026-11-25, second), so that stop does not fire. See the execution record's entry of 2026-11-25, second.)*

## Scope

* Phase 0: `docs/decisions/0092-MADR-*`, `docs/decisions/0092-PLAN-*`, `docs/README.md` (two index rows and the count).
* Phase 1: `llmprovider/internal/wire/responses/responses.go` (the take, the pairing, the unfinished-item check, the internal `Logger` field, the docs of `ReadStream`, `Events`, `takeTerminal`, `end` and `streamEvent.Output`); new tests `llmprovider/internal/wire/responses/terminal_0092_test.go` and `llmprovider/providers/terminal_log_0092_test.go`; the decoder construction in `llmprovider/providers/openai/openai.go`, `grok/grok.go` and `opencode/opencode.go` (the logger only); `docs/architecture.md` (the stream paragraph); `CHANGELOG.md` (one **Fixed** entry); dated annotations or amendments in `0088-MADR` (ST104), `0091-MADR` (the fourth amendment's W7 note; how ST55's check treats an announced call never finished) and `0091-PLAN` (Phase 2's added cases: ST15's shapes, `chatcompat` setting the logger).
* Not in scope: the Messages and Interactions follow-ups (FU1, FU2); 0088's R1 to R3 (its Phase 3 review's); 0091's ST55 check itself (0091-PLAN Phase 2); any exported name.

## Rules for Every Phase

* **Order.** Phase 0 starts after the SDK commits now in flight land (0091-PLAN Phase 1 and the 0091 REPORT's addition), so the MADR's cited 0091 lines can be re-grounded at a commit. Phase 1 starts after 0088-PLAN Phase 3's review and before 0091-PLAN Phase 2 (ST27). All serially, in the same tree.
* **Red, then green.** A test marked red is run at the phase's base and must fail on its assertion, not on a build error; a pin marked green passes at the base. After the change, all pass.
* **Plants.** Each is one production change on a scratch clone, never the tree (`tools/plant.py`); it must turn its test red; the execution record quotes the failing line.
* **Heavy runs.** `python3 -B tools/gate.py` and `make pre-add-check` go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, or another host through `run-on-host`, one at a time per host. A single `go test ./llmprovider/internal/wire/responses` may stay on the session host.
* **Live runs.** None.
* **Staged only.** Each phase ends with its files staged; the owner commits.
* **Stop conditions:** an existing test or golden needs a change no row names; `apidiff` reports any name; a red test cannot be red at the base without a production change; a row's outcome cannot be met as written; 0088's Phase 3 ruling changes an outcome ST10, ST11 or ST13 assumes; a cited line moved so that a row's premise no longer holds; a third build pass of one phase.

## Implementation Steps

### Phase 0: records into the tree

1. **Wait** for the in-flight SDK commits (0091-PLAN Phase 1, the 0091 REPORT addition). Record the commit as this phase's base.
2. **Re-ground.** `python3 -B tools/claims.py <the MADR> --rev self=<base>`; move each row whose line moved (`--fixes`), by dated amendment with ids ending in "now", as 0091-MADR's fourth amendment does. E33 and E62 cite 0091-MADR lines that 0091-PLAN Phase 1 edits: find each text at the base and record its line, or, if a text is gone, what that changes in ST14 and ST26. Plant three claims false (`--plants`) and record the caught lines.
3. **The pair and the index.** Copy the MADR and this PLAN into `docs/decisions/`; add their rows to `docs/README.md`'s Records section and its count; `make records-check` (plant: the PLAN's index row removed on a scratch clone, caught); `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed`; `python3 -B tools/gate.py --markdown` through `run-heavy`.
4. In the tree, the MADR's index row reads `accepted` and this PLAN's `proposed`.
5. The owner approves this PLAN (`tools/records.py status` to `in-progress`).

Exit criterion: no blocker; claims pass at the base and three plants are caught; `make records-check` clean; staged.

### Phase 1: the take, the pairing and the unfinished-item check (ST1 to ST32)

1. **Re-ground** at the phase's base (after 0088-PLAN Phase 3's review): `responses.go` (E1 to E12, E20, E28, E75, E76, E78, E79, E81, E87, E95), `finish.go` (E13, E80, E83), `overstream.go` (E73, E74, E92), `retry.go` (E91, E93, E94), `sse.go` (E68 to E71), the three providers' decoder lines (E21 to E24), and 0088's Phase 3 rulings on R1 to R3. If a ruling changes the outcome ST10, ST11 or ST13 assumes, stop.
2. **The field, inert.** Add `Logger *slog.Logger` to `responses.Decoder`, with its doc comment, and nothing that reads or sets it, so the tests below compile at the base and fail on their assertions.
3. **Tests first**, in `terminal_0092_test.go` (the reader, over hand-written streams in M1 to M8's shapes) and `terminal_log_0092_test.go` (the three providers over fake servers):

   | Test | Rows | Base | Plant (on the finished code) |
   | :--- | :--- | :--- | :--- |
   | `TestTerminalFill_Text_0092`: M1's shape by `Events` and `ReadStream`, finishing `stop`, usage 3 in and 5 out; a refusal part finishing `content_filter` | ST1, ST17, ST31 | red | the take guarded by `cut` again (M1 is `ErrEmptyAnswer`); usage read only on a cut |
   | `TestTerminalFill_Call_0092`: a call, a client tool search with a Search tool, Grok's dispatch wrapper, each only in the terminal | ST2 | red | calls skipped by the take (M2 finishes `stop`) |
   | `TestTerminalFill_Reasoning_0092`: M4's reasoning item taken, and carried in the next request's input | ST3 | red | reasoning skipped by the take |
   | `TestTerminalFill_All_0092`: no `.done` at all, reasoning, message and call taken in the terminal's order; no delta after the first taken `EventItem` | ST4, ST32 | red | taken items yielded in reverse; a text delta yielded for a taken message |
   | `TestTerminalFill_NoDuplicate_0092`: every item by `.done`, the terminal `output` the same items; and `TestEvents_Goldens` | ST5 | green (pins) | nothing ever paired (each item twice) |
   | `TestTerminalFill_Match_0092`, cases (a) to (f), completed and cut | ST6, ST13 | red (each case on at least one path) | today's rule ((a) duplicates); id-only pairing ((b) duplicates item 1); order pairing across types ((c)); order pairing over every `.done` ((d) loses A); index pairing across types ((e)); order pairing in OQ5's case ((f) duplicates B's text) |
   | `TestTerminalFill_DoneWins_0092`: M8 keeps `Hello`; the `EventItem` and `EventDone`'s item equal | ST7 | green (pin) | the terminal's item replacing the delivered one |
   | `TestTerminalFill_Deltas_0092`: deltas `Hel`, terminal `Hello`, response `Hello` | ST8 | red | the text rebuilt from the deltas |
   | `TestTerminalFill_Order_0092`: `Response.Output` equals delivered ++ taken, the `EventItem`s equal `Response.Output`; a cut case where a taken call meets the cut rules | ST9, ST14 | red (the completed case; the cut case is a green pin) | taken items inserted at their terminal index; the take moved after `CutAnswer` |
   | `TestTerminalFill_Held_0092`: a held `{}` call, then a taken call | ST10 | red | taken items yielded before held calls |
   | `TestTerminalFill_Unfinished_0092`: M3, an announced message, and an announced reasoning item (a success); a `Generate` read over the stream under `WithRetry`, sent twice; M3 under `WithStreamRetry`, sent once | ST11 | red (the M3 and message cases; the reasoning case is a green pin) | the check removed; the error built as `ErrIncomplete`; reasoning tracked |
   | `TestTerminalFill_EmptyDone_0092`: M7, an empty message `.done`, a reasoning item kept with text and no encrypted part | ST12 | red (the first two; the third is a green pin) | an empty `.done` counted as delivered; the kept item filled in place |
   | `TestTerminalFill_Plain_0092`: M6's shape by the plain decode | ST16 | green (pin) | none: no code changes |
   | `TestTerminalFill_Dropped_0092`: an empty message, an empty reasoning item, a hosted tool search, a client one with no Search tool, a `custom_tool_call`, only in the terminal | ST28 | green (pins) | the take maps an item of an unknown type to a `FunctionCallItem` |
   | `TestTerminalFill_Collect_0092`: 10 MiB of text deltas, then a terminal message of the same text, through `wire.Collect` | ST30 | red | `Collect` counting a message item's text |
   | `TestTerminalFill_Log_0092`: one line for M1, none for a compliant stream, no text or id in it | ST20 | red | a line for a compliant stream; the text logged |
   | `TestTerminalFill_ProviderLogger_0092`: `openai`, `grok`, OpenCode's responses route, each with `WithLogger`, log M1's line | ST20 | red | `grok` leaving the field unset |
   | existing: `TestCutAnswer_XAIShapes_0088`, `TestCutAnswer_ResponsesCutItem_0088`, `TestCutAnswer_HeldCalls_0088`, `TestReadStream_EndWithoutCompletedIsAfterReply`, `TestReadStream_OverlongEventIsIncomplete`, `TestReadStream_Usage`, the stream error tests | ST13, ST18, ST19, ST29, ST31 | green (pins, unchanged) | none beyond the rows above |

4. **The code.** `streamEvent` reads `output_index`. `streamState` records each `.done` as (id, index, type, whether it added an item) and tracks announced messages, calls and client tool searches. At the terminal event, before the finish decision, D2's three passes pair the `.done`s with the terminal's items (OQ5's case taking nothing of that type); an unpaired item, or the partner of a `.done` that added nothing, is taken through `appendOutput`. On a completed stream, a tracked announced item delivered neither way fails the reply with `ErrProviderUnavailable` marked after the reply, `item_not_finished` with its type and id through `redact.Field`. The cut path keeps 0088's rules. A stream that took items writes one `Debug` line through `Decoder.Logger`; `openai`, `grok` and `opencode` set the field from `Settings.Logger()`.
5. **The docs.** `responses.go`'s docs of `ReadStream`, `Events` (naming `item_not_finished` and its kind), `takeTerminal`, `end` and `streamEvent.Output`; `docs/architecture.md`'s stream paragraph; `CHANGELOG.md` **Fixed**: a completed Responses stream takes the terminal's undelivered items; a cut stream no longer duplicates an item whose ids disagree; an announced item never delivered is `ErrProviderUnavailable` naming `item_not_finished`. Dated annotations: 0088-MADR ST104 (D2's pairing); 0091-MADR's fourth amendment W7 note (a completed stream now takes). A 0091-MADR amendment: how ST55's check treats an announced call never finished, put to the owner there as 0091's own question. A 0091-PLAN amendment: Phase 2's `TestChatcompat_ResponsesDecode_0091` gains M1's and M2's shapes (ST15), and `chatcompat` sets `Decoder.Logger` (ST20). A claims row per new doc sentence in the MADR's evidence (dated amendment), red at the base and planted false after.
6. `go test ./llmprovider/internal/wire/responses ./llmprovider/providers` on the session host; `make pre-add-check FILES="<changed .go files>"` through `run-heavy`.

Exit criterion: no blocker; every test in the table red at the base (or green where marked) and green after; every plant caught, its failing line quoted; every doc row `fixed`; staged.

### Phase 2: gates

1. `python3 -B tools/gate.py` through `run-heavy` (one heavy run on the host at a time).
2. `make api-check`; `python3 -B tools/apidiff.py v1.32.0`: nothing from this record beside the names `[Unreleased]` already carries (ST25); plant an export on a scratch clone and see it reported.
3. `make coverage-check records-check changelog-check docs-check markdownlint links-check disclosure-check fixtures-check`; ST27's plant (the **Fixed** entry outside `[Unreleased]` on a scratch clone) caught by `make changelog-check`.
4. `python3 -B tools/claims.py` on the MADR with its plants; `python3 -B tools/denyscan.py --changed`.

Exit criterion: every gate clean; the apidiff and changelog plants caught; staged.

### Phase 3: the joint review and v1.33.0

1. An independent review of Phases 1 and 2, in 0091-PLAN Phase 4's joint review of v1.33.0, against the MADR's rows; a finding goes to the owner as a question, three options, the recommended first.
2. The owner's rulings recorded by dated amendment; a third build pass of a phase is a stop.
3. The release is 0091-PLAN Phase 4's; the tag and the push only at the owner's ask in the same turn.

Exit criterion: the review finds no blocker; the MADR and this PLAN describe what was built; this PLAN `complete` (`tools/records.py status`).

## Verification

* Every decided row of the MADR maps to one check above; ST21 to ST24 are out of scope and have none; ST25 and ST27 are checked in Phase 2; ST26 is step 5.
* `python3 -B tools/apidiff.py v1.32.0` lists nothing from this record.
* `tools/claims.py` passes the MADR's index, re-grounded at each phase's base, with its plants caught.

## Rollout and Rollback

* Rollout: v1.33.0. A compliant stream sees no change; a stream that omits `.done` returns its answer; a stream that announces an item and never delivers it fails as a retryable broken stream where it succeeded silently before.
* Rollback before the tag: drop the phase's staged files. After the tag, reverting the take is a behaviour change that takes a new record; no export is involved.

## Execution Record

No phase has run. Nothing in the SDK tree was changed.

* **2026-11-24, approval (Phase 0 step 5).** The owner approved this PLAN, relayed by the coordinator, quoted as given: "Approve; follow the queue (Recommended)". The order it sets:
  * Phase 0 runs after the staged 0091-REPORT commit and 0091-PLAN Phase 1's commit land.
  * Phase 1 runs after 0088-PLAN Phase 3's review.
  * Each phase is its own commit, staged by the coordinator after a clean-clone gate and a code review.

  This draft stays `proposed` in scratch. Placing it in the tree in Phase 0 sets it `in-progress` through `python3 -B tools/records.py status`. No phase has run, and nothing in the SDK tree was changed.
* **2026-11-25, Phase 0, pass 1.** Base `138d52f` (v1.33.0), `git status --porcelain` empty at the start; records only, nothing staged (the coordinator stages). Authority: "SDK: build 0092 (Recommended)", quoted in the dated line under the Goal. Scripts and outputs: `<scratch>/m0092/`.
  * **Step 1.** The wait is met: `0c5e143` (the 0091 REPORT addition) and `33e6f09` (0091-PLAN Phase 1) precede the base.
  * **Step 3, the pair and the index.** The MADR and this PLAN copied from `<scratch>/m0092/` byte for byte (`cmp`: identical); `python3 -B tools/records.py add` wrote both index rows and the count (195 to 197 records). `python3 -B scripts/check_records.py --next` printed `0094`, not `0092`: it prints the highest number plus one, and 0093 was numbered after 0092 was reserved for this record (the MADR's E106now); the pair keeps 0092.
  * **Steps 4 and 5, the statuses.** `tools/records.py status`: the MADR `accepted` and this PLAN `in-progress`, each dated 2026-11-25; `records-check: 0 problem(s)` after each.
  * **Step 2, the re-ground.** In the MADR's amendment of 2026-11-25. `tools/claims.py` on the MADR at `--rev self=138d52f`: 82 of 92 pass; the 10 that fail are line moves with their text unchanged, plus gobble-cli's E53 (text changed, fact holds) and E55 at `073fd1f`: 12 rows restated as "now" rows. At the record's own marker: 92 of 92. The "now" rows and the new evidence (24 rows) checked with plain ids at `self=138d52f gobble-cli=073fd1f`: 24 of 24 pass; three planted false, all caught, for example `FAIL P14 docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1513: line 1513 is "* **R1 to R5** ...`. The drafts' ten claims plants, run against the placed MADR: 10 of 10 caught, for example `FAIL E2 llmprovider/internal/wire/responses/responses.go:835: line 835 is 'for _, o := range s.open {'`. `responses.go` has no commit after `2f3c3a9`; no row's premise moved except the release and the 0088 rulings, put to the owner as the MADR's Q-B1.
  * **Checks, this Mac.** `make records-check docs-check links-check markdownlint`, exit 0 (197 records, 0 problems; 0 violations of R50 to R52; 1,532 links in 211 files, 0 problems; 0 issues); `python3 -B tools/recordlint.py --changed`, exit 0 (no rule rose); `python3 -B tools/denyscan.py --changed`, exit 0 (0 hits in 3 files); `tools/claims.py` on the MADR, exit 0 (92 of 92); `python3 ~/.global-git-hooks/github-disclosure.py redact` (a dry run), exit 0, 0 files.
  * **`python3 -B tools/gate.py --markdown` through `run-heavy`, exit 1: 10 of 11 checks pass; `gate-selftest` fails, not on this change.** `scripts/test_gates.py:336`, `test_api_check_accepts_the_0093_removals`: `AssertionError: 'accepted (0093-MADR D10): ./llmprovider/catalog.OptionsFromEnv: removed' not found in 'api-check: against v1.33.0, 0 incompatible change(s) outside llmprovider/x/ and not accepted by a record\n'`. The test expects `api-check` to compare against `v1.32.0`; with `v1.33.0` tagged, the newest tag no longer holds `OptionsFromEnv`, so the acceptance line it asserts cannot print. It depends on the tag, not on any file this phase changed (Markdown under `docs/` only). Recorded as a finding for the owner; nothing changed for it.
  * **Not done, and why.** Phase 0's exit criterion "no blocker" waits on the owner: Q-B1 (the MADR's amendment of 2026-11-25) and the `gate-selftest` finding above. Nothing outside Phase 0: no code, test, golden or changelog was touched.
* **2026-11-25, second: Q-B1 answered (records only).** Base `4fc5211`; the tree held this pair and `docs/README.md` as Phase 0 left them, kept. Nothing staged.
  * **The ruling,** relayed by the coordinator, quoted exactly: "Rule R1–R3 first". With it, 0088's R1 to R3: R1 "Keep, except where calls arrive whole (Recommended)", R2 "Keep as built (Recommended)", R3 "Keep as built (Recommended)". These are 0088-MADR ST107 to ST109; R1's streamed-Interaction case is 0088-MADR ST110, open under 0088's OQ19.
  * **The order:** 0088-PLAN Phase 6, then this PLAN's Phase 1, then 0091-PLAN Phase 2. Phase 1 waits for 0088's OQ19 through Phase 6. *(2026-11-25: OQ19 answered, "Decide by what arrived". ST110 is decided.)*
  * **Re-pointed:** ST10 to ST109, ST11 to ST108, ST13 to ST107 and ST108 (the MADR's amendment of 2026-11-25, second). No outcome changes.
  * **Measured** on a scratch clone at `4fc5211`: a cut Responses reply's last `{}` call is dropped and named by `Generate` and `Stream`, which is ST107's Responses outcome. At the tree, `TestCutAnswer_ResponsesCutItem_0088`, `TestCutAnswer_HeldCalls_0088` and `TestCutAnswer_XAIShapes_0088` pass (exit 0).
  * **Evidence:** E107now to E112now were checked with plain ids in a scratch record. The base was a scratch clone at `4fc5211` with the tree's 0088 pair committed on it (`8c97b68`, scratch only). `tools/claims.py` exited 0, 6 of 6 pass. Three rows were planted false (a line moved by one): E107, E109 and E112. Each was caught, exit 1; for example, `FAIL E112 docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md:972: line 972 is ''; text found at [971]`.
  * **Still open for Phase 0's exit:** the `gate-selftest` finding above. `4fc5211` ("pin api-check base to fix post-tag gate failure") touches that gate; it was not re-run here.
