---
status: in-progress
date: 2026-11-24
associated-madr: "0091-MADR-adopt-the-responses-api-per-provider.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement: Adopt the Responses API Per Provider

Associated MADR: [0091-MADR-adopt-the-responses-api-per-provider.md](0091-MADR-adopt-the-responses-api-per-provider.md)

## Goal

After this PLAN, `kilo`, `together`, `huggingface` and `ollama` stay on Chat Completions, and each package doc states the 2026-11-23 measurement that keeps it there (the owner's answers "Chat stays (Recommended)", "Chat stays, every model (Recommended)", "Chat stays (Recommended)", "Chat stays (Recommended)"). A `chatcompat` provider built with the new `chatcompat.WithResponses()` sends `Generate` and `Stream` to `POST {base}/responses` and reads the reply with the shared Responses decoder, with every behaviour the MADR's ST50 to ST62 decide ("Opt-in WithResponses()"). The cut check (OQ6, OQ7) runs on `chatcompat`'s Responses path and on no other provider ("Yes, output tokens at limit (Recommended)"). On that path, reasoning a server sends only as plain text is read and shown, not sent back ("Read it, don't send it back (Recommended)"), and the package doc states every loss the opt-in carries ("Keep it, losses documented (Recommended)"). The evidence (desk research, probe rounds 1 to 7) is committed as `0091-REPORT-responses-api-across-providers.md`. All of it rides in v1.33.0.

*(2026-11-23, revision 1: after review. The open questions are now OQ9, OQ10 and OQ11 (the MADR's second amendment); Phase 0 takes their answers before the MADR is accepted; under OQ11 (2), Phase 2 is out of scope. Phase 2's step 4 follows the corrected ST51; its table adds ST13, ST16, ST18's `include`, ST19, ST23 and the OQ10 and OQ11 cases; ST27's and ST30's plants are listed.)* *(2026-11-23, revision 2: the OQ11 (3) guard test matches that option's one reading, the effort guard and the replay guard.)* *(2026-11-23, revision 3: the owner answered OQ9 "Read it, don't send it back (Recommended)", OQ10 "Yes, output tokens at limit (Recommended)" and OQ11 "Keep it, losses documented (Recommended)"; Phase 0 step 5 is met; Phase 2 takes those options' tests, and the OQ11 (3) case is dropped.)*

The MADR is `accepted` (2026-11-23, the owner's answers to OQ1 to OQ11; 0 rows open). This PLAN is `proposed`. Phase 0's steps 1 to 4 place the records in the tree, which the bootstrap rule allows with no prior approval; its step 5 is met; its step 6 is the owner's approval of this PLAN. No phase after Phase 0 starts until the owner approves this PLAN and says `proceed`, `execute the plan` or `do phase N`.

## Scope

* Phase 0: `docs/decisions/0091-MADR-*`, `docs/decisions/0091-PLAN-*`, `docs/reports/0091-REPORT-responses-api-across-providers.md`, `docs/README.md` (three index rows).
* Phase 1: the package docs of `llmprovider/providers/kilo/kilo.go`, `huggingface/huggingface.go`, `together/together.go`, `ollama/ollama.go` (comments only); `docs/architecture.md` and `README.md` where they name each provider's wire.
* Phase 2: `llmprovider/providers/chatcompat/` (the option, a Responses body function, the decoder and matcher wiring, the limit forms, tests, goldens, `ExampleWithResponses`, the package doc); `llmprovider/internal/wire/responses/responses.go` (ST10's read of `reasoning_text`, ST63; the limit field and the check of ST33 and ST34, set only by `chatcompat`, ST55); `CHANGELOG.md` (one **Added** entry); `docs/architecture.md`, `README.md`; the Responses package doc (C1); `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md` (one dated annotation at D4, ST31).
* Not in scope: any other provider's code; the cut check on any provider but `chatcompat`'s Responses path (ST14, ST15); the wizard (ST61); any Ollama move (OQ4's answer); 0088's R1 to R3 (its Phase 3 review's); 0089's limit rows on Grok (B3's note only).

## Rules for Every Phase

* **Order.** Phase 0 starts only after 0088-PLAN Phase 1b is committed, so the MADR's working-tree rows (W1now to W7now) can be re-grounded at a commit. Phase 2 starts only after 0088's phases and its Phase 3 review are done (ST30): `chatcompat`'s opt-in reads replies through the decoder they change. All serially, in the same tree.
* **Red, then green.** A test marked red is run at the phase's base and must fail on its assertion, not on a build error; a pin marked green passes at the base. After the change, all pass.
* **Plants.** Each is one production change on a scratch clone, never the tree (`tools/plant.py`); it must turn its test red; the execution record quotes the failing line.
* **Heavy runs.** `python3 -B tools/gate.py` and `make pre-add-check` go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, or another host through `run-on-host`, one at a time per host; never two heavy runs at once on one host. A single `go test ./llmprovider/providers/chatcompat` may stay on the session host.
* **Live runs.** None are needed under the answers, unless OQ9's answer names a probe; any live run is an agent's, with environment keys only, allowlisted fields only, under `tmp/live-0091/`.
* **Staged only.** Each phase ends with its files staged; the owner commits.
* **Stop conditions:** an existing test needs a change; `apidiff` reports anything but `chatcompat.WithResponses`; a red test cannot be red at the base without a production change; a row's outcome cannot be met as written; a cited line moved so that a row's premise no longer holds; a third build pass of one phase.

## Implementation Steps

### Phase 0: records into the tree

1. **Wait** for 0088-PLAN Phase 1b's commit. Record its hash as this phase's base.
2. **The REPORT.** Write `docs/reports/0091-REPORT-responses-api-across-providers.md` from the desk research and the probe hand-backs of rounds 1 to 7: the allowlisted facts only, each row of the MADR's probe table (PR1 to PR10) traceable to it; no key, no account id, no host name, no real path (`tools/denyscan.py --changed`, `make fixtures-check`, `make disclosure-check`).
3. **Re-ground.** At the base: re-run `python3 -B tools/claims.py` on the MADR with `self` set to the base; move any row whose line moved (`--fixes`); turn W1now to W7now into checked rows at the base, or record that the 0088 code they cite changed and what that does to B3 (evidence only; no row of this MADR rests on their line numbers). W2now, W6now and W7now are known to have moved with 0088 Phase 1b's later work in `responses.go`; their new lines are found here. Plant three claims false (`--plants`) and record the caught lines.
4. **The pair and the index.** Copy the MADR and this PLAN into `docs/decisions/`, the REPORT into `docs/reports/`; add the three rows to `docs/README.md`'s Records section and its count; `make records-check` (ST27's plant: the REPORT's index row removed on a scratch clone, caught); `python3 -B tools/gate.py --markdown` through `run-heavy`.
5. **OQ9 to OQ11.** Met on 2026-11-23: the owner's answers decide ST55, ST63, ST64 (and ST10, ST33, ST34) in the MADR's third amendment, and the MADR is `accepted`; in the tree, its index row reads `accepted`.
6. The owner approves this PLAN (`tools/records.py status` to `in-progress`).

Exit criterion: no blocker; the MADR `accepted` with no row open; claims pass and three plants are caught; `make records-check` clean.

### Phase 1: the docs of the providers that stay (ST1 to ST4, ST28)

1. **Re-ground** each paragraph named below at the phase's base.
2. **Claims first.** Add to the MADR's evidence (by dated amendment) one claims row per new doc sentence, with the text it will hold; at the base each fails (red); plant each false after the change.
3. **The docs.**
   * `huggingface.go`: the paragraph at lines 9 to 15 (C17, C18) restated: the router documents `/responses` as beta; on 2026-11-23 a bad key still answered HTTP 200 `failed` with `server_error`, a text cut answered `completed`, and no encrypted reasoning came back when asked; so Chat Completions stays (0091-MADR ST3).
   * `kilo.go` (C13, C14), `together.go` (C21), `ollama.go` (C23): one sentence each naming the 2026-11-23 measurement that keeps it on Chat (Kilo: an undocumented route, no encrypted reasoning, a call cut reported `completed`; Together: four models only; Ollama: an effort the model cannot take silently ignored, a text cut reported `completed`).
   * `docs/architecture.md`, `README.md`: where each provider's wire is named, the reason by record.
4. `make pre-add-check FILES="<the four .go files>"` through `run-heavy`; `make docs-check parity-check links-check`.

| Check | Rows | Base | Plant |
| :--- | :--- | :--- | :--- |
| a claims row per new doc sentence | ST1 to ST4, ST28 | red (the sentence absent) | the Hugging Face doc's old "undocumented" line kept: caught |
| the existing wire goldens of the four providers | ST1 to ST4 | green | none: no code changes |

Exit criterion: no blocker; every claims row passes and its plant is caught; the four packages' tests pass; staged.

### Phase 2: `chatcompat.WithResponses()` (ST5, ST10, ST33, ST34, ST50 to ST64)

1. **Re-ground** at the phase's base (after 0088's Phase 3 review): `chatcompat.go` (C94 to C97, C100, C101), `limit.go` (C98), the Responses decoder and refusal matcher (C1, C78, C80), `opencode.go`'s Responses limit forms (C99), and 0088's Phase 3 rulings on R1 to R3; record any moved line and any ruling that changes ST52's "0088's cut answer for an `incomplete` reply".
2. **The export, inert.** Add `WithResponses() llmprovider.Option`, built as `WithID` is (C94, C95), with its doc comment, and nothing that reads it, so the tests below compile at the base and fail on their assertions.
3. **Tests first**, in `llmprovider/providers/chatcompat/responses_0091_test.go`, over a fake server that answers `/responses` with the shapes rounds 3 and 7 recorded (redacted), and the G-wire goldens:

   | Test | Rows | Base | Plant (on the finished code) |
   | :--- | :--- | :--- | :--- |
   | `TestChatcompat_WithResponses_0091` | ST50 | red | the option ignored (`/chat/completions` hit); a 404 resent to `/chat/completions` |
   | `TestWireGoldens`, the `chatcompat` Responses cases (`Generate`, `Stream`, tools and an image, a `ToolLoadItem`) | ST51, ST57 | red (goldens absent) | `max_output_tokens` sent with no caller limit; `strict` dropped; `store` omitted; the native deferral form chosen |
   | `TestChatcompat_ResponsesDecode_0091`, plain and stream | ST52 | red | the reply labelled `responses` instead of the provider's id |
   | `TestChatcompat_ResponsesErrors_0091` | ST53 | red | a 200 `failed` decoded as an empty success |
   | `TestChatcompat_ResponsesRefusal_0091` | ST54 | red | the Chat matcher wired on the Responses path |
   | `TestChatcompat_ResponsesCut_0091`, plain and stream: a text reply at the limit finishes `length`, a reply under it `stop`, a call that does not parse at the limit dropped with `WarningToolCallCut`, and the streamed cut call never yielded | ST55, ST33, ST34 | red | the limit left unset for `chatcompat` (the reply at the limit finishes `stop`); the check moved after the held calls are released (the stream yields the cut call) |
   | `openai`'s existing decode tests, unchanged | ST33 | green | the limit field set for `openai`, so a reply at its limit finishes `length` |
   | `TestResponses_ReasoningText_0091`, plain and stream, with an OpenAI-shaped summary item decoding unchanged | ST10, ST63 | red | the decoder reading `summary[]` only; the `content` text preferred over a non-empty summary |
   | `TestChatcompat_ResponsesReasoningText_0091`: the text shown, and absent from the next request | ST63 | red | the text item replayed on the next turn |
   | `TestChatcompat_ResponsesFacts_0091` | ST51 (restated) | red | an image to a model whose facts take none sent (the refusal dropped); an effort outside the facts' list sent unchanged (the ladder bypassed) |
   | `TestChatcompat_ResponsesLimitRefusal_0091` | ST56 | red | the Chat Completions limit forms wired on the Responses path |
   | `TestChatcompat_ResponsesNoContinuation_0091`, with a `Generate` that names no limit read over the stream | ST58, ST23 | red (the stream half; the continuation half is green, a pin) | `Continuation` `BestEffort` on the Responses path; a plain request sent for a `Generate` with no limit named |
   | `TestChatcompat_ResponsesUsage_0091`: `reasoning_tokens` 0 with a reasoning item, as recorded by round 3, passed through as 0 | ST13 | red | a count estimated from the reasoning text |
   | `TestChatcompat_ResponsesDone_0091`: a stream ending `response.completed` then `[DONE]`, and one ending `response.incomplete` then `[DONE]` | ST16 | red | `[DONE]` decoded as JSON |
   | the request goldens' `store` and `include` | ST18 | red | `store` omitted; `include` omitted |
   | `TestChatcompat_ResponsesErrorShapes_0091`: the probe shapes (codes only) of a 401 with a top-level `error_type`, a 400 whose `error` is a string | ST19 | red | a 401 read as `ErrProviderUnavailable` |
   | `llmtest.Run` on a provider built `WithResponses()`, against a scripted Responses server | ST59 | red | `Continuation` declared `Supported` |
   | `TestChatcompat_ResponsesListingProbe_0091` | ST61 | red | the probe pinned to Chat |
   | ST62's claims rows, one per stated loss | ST62, ST64 | red (the sentences absent) | the ST53 sentence removed |

4. **The code.** `Generate` and `Stream` routed by the option (ST50). A Responses body function in the package (ST51, restated), from the same cached facts the Chat path reads (`catalog.CachedFacts`): the effort by 0062's ladder over `facts.Reasoning.Efforts`, learned through `responses.MatchRefusal` (ST54); an image refused before the network by facts (`wire.RefuseImages`), as on Chat; prior reasoning replayed only when it carries `encrypted_content`, with no replay under `facts.Reasoning.ReplayField`. The shared decoder labelled with the id, no `Search` or `Dispatch` (ST52), reading `reasoning_text` content when the summary holds no text and mapping its delta (ST10, ST63); `wire.ResponsesLimits` for the limit retry on that path (ST56); `deferral.Other` as on Chat (ST57); the limit sent passed to the decoder by `chatcompat` alone, and the check of ST33 and ST34 (ST55).
5. **The docs.** `chatcompat`'s package doc paragraph (ST62); `ExampleWithResponses` (ST60's call site); `CHANGELOG.md` **Added**: `chatcompat.WithResponses`; `docs/architecture.md`, `README.md`; the Responses package doc naming `chatcompat` as its fourth user; 0061-MADR D4's dated annotation (ST31); a claims row per new doc sentence, each planted false.
6. `go test ./llmprovider/providers/chatcompat` on the session host; `make pre-add-check FILES="<changed .go files>"` through `run-heavy`.

Exit criterion: no blocker; every test in the table red at the base (or green where marked) and green after; every plant caught, its failing line quoted; staged.

### Phase 3: gates

1. `python3 -B tools/gate.py` through `run-heavy` (one heavy run on the host at a time).
2. `make api-check`; `python3 -B tools/apidiff.py v1.32.0`: from this record, `chatcompat.WithResponses` and nothing else, beside the names `[Unreleased]` already carries (ST60); plant a second export on a scratch clone and see it reported.
3. `make coverage-check records-check changelog-check docs-check markdownlint links-check disclosure-check fixtures-check`; `python3 -B tools/claims.py` on the MADR with its plants.
4. `python3 -B tools/denyscan.py --changed`.

Exit criterion: every gate clean; the apidiff plant caught; staged.

### Phase 4: the joint review and v1.33.0

1. An independent review of Phases 1 to 3, together with the v1.33.0 work of 0088, 0089 and 0090, against the MADR's rows; a finding goes to the owner as a question, three options, the recommended first.
2. The owner's rulings recorded by dated amendment; a third build pass of a phase is a stop.
3. The release: `CHANGELOG.md`'s `[Unreleased]` becomes v1.33.0 in the release step 0088's, 0089's and 0090's records share (ST30's check, `make changelog-check`; its plant, this record's **Added** entry moved outside `[Unreleased]` on a scratch clone, caught); the tag and the push only at the owner's ask in the same turn.

Exit criterion: the review finds no blocker; the MADR and this PLAN describe what was built; this PLAN `complete` (`tools/records.py status`).

## Verification

* Every decided row of the MADR maps to one check above; the rows out of scope (ST12, ST20 to ST22, ST32, ST36 to ST46, ST48, ST49) and ST14 and ST15, which build nothing beyond `chatcompat`'s path, have none, as the MADR says; ST25, ST33 and ST34 are covered on `chatcompat` by `TestChatcompat_ResponsesCut_0091`.
* `python3 -B tools/apidiff.py v1.32.0` lists `chatcompat.WithResponses` from this record and nothing else.
* `tools/claims.py` passes the MADR's index, re-grounded at each phase's base, with its plants caught.

## Rollout and Rollback

* Rollout: v1.33.0. A caller who does not give `WithResponses()` sees no change; the four providers' wires do not change.
* Rollback before the tag: drop the phase's staged files. After the tag, removing `WithResponses` is an incompatible change that takes a new record and a major version (R48); turning it inert is the minor-version alternative that record would weigh.

## Execution Record

* **2026-11-23, approval (Phase 0 step 6).** The owner approved this PLAN, relayed by the coordinator, quoted as given: "Approve; follow the queue (Recommended)". It sets the order: Phases 0 and 1 run after 0088-PLAN Phase 1c and 0090's work are committed, one at a time in the tree; Phase 2 runs after 0088's Phase 3 review; each phase is its own commit, staged by the coordinator after a clean-clone gate (`tools/gate.py` through `run-heavy`). This draft stays `proposed` in scratch; placing it in the tree in Phase 0 sets it `in-progress` through `python3 -B tools/records.py status`. No phase has run; nothing in the SDK tree was changed.
* **2026-11-24, approval restated for Phase 0 (step 6).** The coordinator relayed the same approval for this phase, quoted as given: "Approve; follow the queue (Recommended)": Phases 0 and 1 after 0088 Phase 1c and 0090 are committed (both now are: `903c387` and `46f43d7`), one at a time in the tree; Phase 2 after 0088's Phase 3 review; each phase its own commit, staged by the coordinator after a clean-clone gate. This PLAN is `in-progress` from this entry (`python3 -B tools/records.py status ... in-progress --date 2026-11-24`); the MADR's date is moved to the same day for its fourth amendment, its status unchanged (`accepted`).
* **2026-11-24, Phase 0, pass 1.** Run in the tree, nothing staged (the coordinator stages).
  * **Step 1, the base.** `dd37a87` (the current HEAD; 0088-PLAN Phase 1b is `f4efc5e`, Phase 1c `903c387`, 0090 `46f43d7`, the qwen-code report `dd37a87`); the tree was clean (`git status --porcelain` empty) before the first write.
  * **Step 2, the REPORT.** `docs/reports/0091-REPORT-responses-api-across-providers.md`, from the desk research and the probe hand-backs of rounds 1 to 7: the research's sources and unknowns, the nine probes of round 1 and rounds 2 to 7 by service, and a table that maps each of the MADR's PR1 to PR10 to the rounds and services that hold it. Allowlisted facts only; the probe script is cited as `<scratch>/responses-probe/probe.py`; no key, account id, host name or real path. Its 17-row evidence index passes at `dd37a87` (`tools/claims.py`, exit 0).
  * **Step 3, the re-grounding.** `tools/claims.py` on the MADR with `--rev self=dd37a87`: 82 of 109 pass, exit 1; 27 rows moved (26 by line, C7 by text), none gone; recorded as the MADR's fourth amendment with the 27 rows under ids ending in "now" and the record's own marker (`self=247b745`) left, 109 of 109 passing. W1now to W7now are restated as rows checked at `dd37a87`: W1now, W3now, W4now and W5now on the same line; W2now, W6now and W7now moved with `streamState.end`; on a scratch copy with plain ids, moved rows in place, W1 to W7 appended and the marker at the base, 116 of 116 pass, exit 0. The cut rules R1 to R3 are still open for 0088's Phase 3 review; 0088's R4 and R5 reach no row here. Three plants (`--plants`), all caught: W3's line moved (`FAIL W3 ...finish.go:106: line 106 is 'return false'`); W1's text made false (`FAIL W1 ...responses.go:422: line 422 is 'if raw.Status == statusIncomplete {'`); C50's text on the kilocode checkout made false (`FAIL C50 kilocode:packages/kilo-gateway/src/provider.ts:1: line 1 is 'import { createOpenRouter } from "@openrouter/ai-sdk-provider"'`).
  * **Step 4, the pair and the index.** The MADR and this PLAN are in `docs/decisions/`, the REPORT in `docs/reports/`; `tools/records.py add` wrote the three rows and the count (193 records); `make records-check` is clean. **ST27's plant**, on a scratch clone (`tools/plant.py`): the REPORT's index row removed, caught, exit 1: `docs/README.md: says 193 records; its Records table has 192 rows` and `docs/reports/0091-REPORT-responses-api-across-providers.md: not indexed in docs/README.md`.
  * **Step 5, OQ9 to OQ11.** Met on 2026-11-23, as the PLAN states; the MADR's index row reads `accepted`.
  * **Gates, all run in the tree on this Mac (the session host), exit 0 each:** `make records-check markdownlint links-check docs-check fixtures-check disclosure-check` (193 records, 0 problems; markdownlint 0 issues; 1,489 relative links in 207 files, 0 problems; 0 violations of R50 to R52; 507 fixture files, 0 problems; 0 disclosure hits in 1,875 files); `tools/recordlint.py --changed` (the first run found one MD012 in the MADR, a doubled blank line before the fourth amendment, fixed in this pass; no new finding after); `tools/denyscan.py --changed` (0 hits in 4 files); `tools/claims.py` on the MADR (109 of 109 at its own marker) and on the REPORT (17 of 17); and `python3 -B tools/gate.py --markdown` through `run-heavy` (11 of 11 checks, 328 s, exit 0).
  * **Not done, and why.** Nothing outside Phase 0 (Phases 1 to 4 are separate commits and briefs); no code, no test, no golden and no live run were touched.
