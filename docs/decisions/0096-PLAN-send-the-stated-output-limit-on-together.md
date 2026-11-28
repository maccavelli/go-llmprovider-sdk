---
status: in-progress
date: 2026-11-27
associated-madr: "0096-MADR-send-the-stated-output-limit-on-together.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement: Send the Model's Stated Output Limit on Together

Associated MADR: [0096-MADR-send-the-stated-output-limit-on-together.md](0096-MADR-send-the-stated-output-limit-on-together.md)

*(2026-11-27, approved. The coordinator relayed that the owner's "fold it into v1.35.0" approves the build, and gave the go-ahead for Phase 2 at base `08d3b77`, where 0095's build is committed. This PLAN is `in-progress`. The repository's PLAN statuses have no `approved`; an approved PLAN being built carries `in-progress`, as 0094-PLAN did.)*

## Goal

After this PLAN, a Together request with no caller value sends the model's chosen output fact as `max_tokens`, held to the window's room, as Kilo does; a validated refusal is resent once at its stated cap and the cap kept; it ships in v1.35.0, by the owner's decision "fold it into v1.35.0".

The MADR has 3 open rows (TG2, TG3, TG8) and is `proposed`. No phase after Phase 0 starts until the owner answers OQ1 to OQ4, the MADR is `accepted`, this PLAN is approved, and the owner says to proceed. If OQ3 is answered (1), Phase 1 runs before Phase 2; otherwise Phase 1 is not run and that is recorded.

## Scope

* Phase 0: these two records; `docs/README.md`'s two index rows and its count, added once 0095's build has released the file.
* Phase 1 (only under OQ3 (1)): one live request; a scratch driver outside the tree; the result recorded in this PLAN and the MADR.
* Phase 2: `llmprovider/providers/together/limit.go` (`chooseLimit` builds a `wire.LimitInput` with `Sends: true`, the window and the estimate, as `kilo.go`'s `limitInput` does, and calls `r.Choose`); under OQ1 (2) or (3), the evidence filter that answer names; under OQ3 (1) with a new form, `llmprovider/internal/wire/limit_refusal.go`; the new tests; the changed cases TG11 names; the docs TG12 names.
* Not in scope: every other provider (TG13, OQ4); the exported API (no name is added); released changelog sections (0094-MADR).

## Rules for Every Phase

* **Re-ground first.** Each phase starts by re-reading the MADR's evidence rows E1 to E14 at its base with `python3 -B tools/claims.py docs/decisions/0096-MADR-send-the-stated-output-limit-on-together.md`; a moved line with changed text is a stop.
* **Red, then green.** A row that changes behaviour has a test red at the phase's base on its assertion. A pin passes at the base.
* **Plants.** Each is one production change on a scratch clone through `tools/plant.py`, caught only by its own row's test; the execution record quotes the failing line.
* **Heavy runs.** `go test ./...`, any `-race`, `make pre-add-check` and `python3 -B tools/gate.py` go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` or `scripts/run-on-host`, one at a time per host. `go test ./llmprovider/providers/together` without `-race` may stay on the session host.
* **Staged only.** The coordinator stages; the owner commits.
* **Stops:** an existing test or golden needing a change TG11 does not name; a row that cannot be met as written; the same step failing twice; a third build pass of one phase.

## Implementation Steps

### Phase 0. Records

1. Draft this pair (done, 2026-11-27); `scripts/check_records.py` and markdownlint on a copy of the index (0095's build holds `docs/README.md`).
2. Put OQ1 to OQ4 to the owner in one prompt; record each answer as a row, quoted exactly.

### Phase 1. Measure the refusal text (OQ3 (1) only)

1. One streamed request to Together's `/chat/completions` for the model OQ1 makes reachable (`Qwen/Qwen4-Max` if the handed-in document states it), `max_tokens` at the stated value, a one-word prompt, `TOGETHER_API_KEY` by name, one retry at most.
2. Allowlisted output: status code, finish reason, output token count, and on a refusal the error's `type`, `code` and whether its message matches each recorded form. The message text itself is recorded only if it matches no form and the owner allows it.
3. Record the result as a dated row in the MADR (TG8's outcome).

### Phase 2. Build

| Test | Rows | Case | Red at base | Plant |
| :--- | :--- | :--- | :--- | :--- |
| `TestOutputLimit_Together_0096` | TG1 | a listing stating 40,000, no caller value: `max_tokens` 40,000, basis `fact`, source the listing's | yes | `Sends` false on Together |
| `TestOutputLimit_Together_0096` | TG2 | as OQ1 answers: a document-only or layer-only fact sent, or not | per answer | the answer's source filter removed |
| `TestOutputLimit_Together_0096` | TG3 | as OQ2 answers: nothing stated | per answer | per answer |
| `TestOutputLimit_Together_0096` | TG4 | `Request.MaxOutputTokens` 77 sent as given; `WithMaxTokens(100000)` with 40,000 stated: 40,000, basis `option` | pin | the option raised to the fact |
| `TestOutputLimit_Together_0096` | TG5 | stated 40,000, `context_length` 32,768, a short input: held to the room | yes | the hold removed |
| `TestOutputLimit_Together_0096` | TG9 | a stream reads `StreamedOutput` | yes | the stream reads `Output` |
| `TestOutputLimit_Together_0096` | TG10 | one POST and no listing GET per request | pin | a listing read on the request |
| `TestLimitRefusal_Together_0096` | TG6 | the sent 40,000 refused "supports at most 32000 completion tokens, whereas you provided 40000": resent at 32,000, basis `refusal`, warning | yes | the resend dropped |
| `TestLimitRefusal_Together_0096` | TG7 | a second request on the same key sends 32,000 | yes | the memo not read |
| `TestLimitRefusal_Together_0096` | TG8 | as OQ3 answers | per answer | per answer |
| `TestOutputLimit_Omitted_0089`, `TestOutputLimit_Facts_0041` | TG11 | Together's case and row expect the fact | changed | `Sends` false on Together |
| `TestTogether_RequestShapes` | TG11 | no fact: none sent | pin | `Sends` true with a fixed value |

Docs (TG12): `together.go`'s package doc, `limit.go`'s comment, `LimitInput.Sends`'s comment, `TestStreamEventsRetry_FirstPayloadError_0088`'s comment, a **Changed** entry under `[Unreleased]`; each new sentence a claims row, planted false.

Exit: no blocker, every decided row tested, every plant caught.

## Verification

* `go test ./llmprovider/providers/together ./llmprovider/providers ./llmprovider/internal/wire` (single packages, session host), then `run-heavy -- make pre-add-check` and `run-heavy -- python3 -B tools/gate.py`.
* `python3 -B tools/claims.py` on the MADR, with its plants.
* `make changelog-check`, `make records-check`.
* An independent review (`code-reviewer`) of the build.

## Rollout and Rollback

Ships in v1.35.0 with 0095. A caller who wants today's behaviour on a model passes `WithMaxTokens` or `Request.MaxOutputTokens`. Rollback is reverting Phase 2's commit; the records stay with a dated note.

## Amendment 1 (2026-11-27): three wires, built after 0095

Additions only; the steps above are annotated here, not rewritten. The owner's answers are in the MADR's amendment 1: OQ1 "Anything the SDK ranks (Recommended)", OQ2 "Send nothing, as today (Recommended)", OQ3 "Measure first (Recommended)", OQ4 "Also HF and OpenCode chat".

* **Order.** Phase 2 starts only once 0095-PLAN-clear-the-sdk-follow-ups.md's build has landed and released its files: `CHANGELOG.md`, `docs/README.md` and `docs/architecture.md`, all shared with it. Phase 2 also needs TG14 answered, Phase 1's remaining measurements recorded, the MADR `accepted`, this PLAN approved, and the owner's word to proceed.
* **Phase 1 (now required, OQ3 (1)).** Together was measured once (MADR amendment 1: 400 `invalid_request_error` / `streaming_required` on a plain request), and Hugging Face and OpenCode were not run. What runs next is TG14's answer. Under (1), one streamed request per wire at the values the MADR lists. Under (2), Hugging Face and OpenCode streamed only. Under (3), as (1), and the caller-path resend is added to Phase 2. The driver is `<scratch>/together/measure_limit.py`, outside the tree; under (1) or (2) it gains a streamed mode, which counts the usage chunk's tokens and the finish reason.
* **Phase 2 scope, added.** `llmprovider/providers/huggingface/limit.go` (as Together's: `r.Choose` with `Sends: true`, the window (DL10) and the estimate). `llmprovider/providers/opencode/opencode.go`: in `routeCall`, the chat route leaves `Optional` for `Choose` with `Sends: true` and `Required: false`, so no fallback applies; the responses and google routes stay on `Optional`. The docs rows HF9 and OC9. The changed cases HF8 and OC8.

| Test | Rows | Case | Red at base | Plant |
| :--- | :--- | :--- | :--- | :--- |
| `TestOutputLimit_Together_0096` | TG2, TG3 | a document stating 262,144 with window 1,000,000: 262,144 sent, source the document's; nothing stated: none | TG2 yes; TG3 pin | registry evidence filtered out; the window's room sent |
| `TestOutputLimit_HuggingFace_0096` | HF1 to HF4, HF7 | document 384,000, providers 1,048,576 and 262,144, a 7-token input: 258,041; none; caller cases; stream reads `StreamedOutput`; one POST, no listing GET | HF1, HF4, HF7 stream yes; others pin | `Sends` false; the largest window read; the stream reads `Output` |
| `TestLimitRefusal_HuggingFace_0096` | HF5 | OpenAI's cap form: resent at M, kept, the second request sends M | yes | the resend dropped; the memo not read |
| `TestOutputLimit_OpenCodeChat_0096` | OC1 to OC4, OC7 | `glm-5.3` on Zen and `qwen4-max` on Go, document-stated: sent; none: none and no fallback; caller cases; window hold; stream | OC1, OC4, OC7 stream yes; others pin | the chat route left on `Optional`; the messages fallback on chat; the hold removed |
| `TestLimitRefusal_OpenCodeChat_0096` | OC5 | OpenAI's cap form on the chat route | yes | the resend dropped; the memo not read |
| `TestOutputLimit_Omitted_0089`, `TestOutputLimit_Facts_0041` | HF8, OC8, TG13 | the Hugging Face and OpenCode chat cases expect the fact; Ollama, chatcompat, Gemini, Grok, OpenAI, and OpenCode's responses and google routes stay none | changed / pin | `Sends` false on each; `Sends` true on Ollama |

Exit for Phase 2: no blocker, all 29 decided rows tested, every plant caught.

## Execution Record

*(2026-11-27, Phase 0 step 1: drafted. One live listing read for the MADR, allowlisted; check-records and markdownlint as the hand-back names.)*

*(2026-11-27, Phase 0 step 2: the owner answered OQ1 to OQ4, recorded in the MADR's amendment 1. Phase 1, pass 1: one Hugging Face listing read (200), one models.dev read, and one Together request (400 `streaming_required`). The Hugging Face and OpenCode requests were not sent: the brief said to stop at a refusal in a form the SDK does not read. TG14 is put to the owner.)*

*(2026-11-27, Phase 1, pass 2: the owner answered TG14 with "Re-run all three, streamed (Recommended)". The driver was `<scratch>/together/measure_stream.py`, with the stream body together.go, huggingface.go and opencode.go build. Three requests, one each: Together `Qwen/Qwen4-Max` at 262,144 returned 200, `stop`, 14 output tokens; Hugging Face `deepseek-ai/DeepSeek-V4.1-Flash` at 258,041 returned 200, `stop`, 29; OpenCode Zen `glm-5.3` at 131,072 returned 200, `stop`, 34. Each limit was accepted, and none of the requests produced an error. Phase 1 is complete. The MADR is `accepted`, with all 32 rows decided (amendment 2).)*

## Amendment 2 (2026-11-27): ready to build after 0095

Additions only. Phase 1 is complete. One test row is added for TG14, which is decided:

| Test | Rows | Case | Red at base | Plant |
| :--- | :--- | :--- | :--- | :--- |
| `TestOutputLimit_Together_0096` | TG14 | a plain `Generate` with an SDK-chosen limit (a document stating 262,144) is sent as a stream with `max_tokens` 262,144; one with the caller's `MaxOutputTokens` stays plain | stream case yes; caller case pin | `OverStream` bypassed on Together |

Phase 2's exit, replacing amendment 1's "all 29": no blocker, all 32 decided rows tested, every plant caught. This PLAN stays `proposed` until the owner approves it. Phase 2 starts once 0095-PLAN-clear-the-sdk-follow-ups.md's build has landed and released `CHANGELOG.md`, `docs/README.md` and `docs/architecture.md`, and the owner says to proceed.

## Execution Record, Phase 2

*(2026-11-27, Phase 2, pass 1, base `08d3b77`, stopped.)*

* **Re-ground.** `tools/claims.py` passes 30 of 30 of the MADR's rows at `08d3b77`.
* **Red at the base.** The tests were in place and the production code untouched. `go test ./llmprovider/providers -run '0096|TestOutputLimit_Omitted_0089|TestOutputLimit_Facts_0041'` failed on its assertions in 10 tests. Every changing row was red; the pins (caller cases, no request added) passed. The output is in `<scratch>/together/red.out`.
* **Change.**
  * `together/limit.go` and `huggingface/limit.go`: `chooseLimit` became `limitInput`, with `Sends: true`, the window and the estimate, chosen by `limit.Choose`; the call's `Refusal` became `cmp.Or(refused, …)`, as Kilo's.
  * `opencode.go`: the chat route chooses by `limitInput` with `Required` true on messages only, and reads `c.limits` on the chat route.
  * The targeted run is green.
* **Stop.** `run-heavy -- go test -count=1 ./llmprovider/providers/... ./llmprovider/internal/wire/... ./llmprovider/catalog/...` exited 1. `TestWireGoldens` in `llmprovider/providers/opencode` fails on 24 goldens: 12 in `testdata/wire/opencode-zen-chat/` and 12 in `testdata/wire/opencode-go-chat/`. Each shows one difference, `requests[0].body.max_tokens: want (absent), got 131072`. The golden harness hands in `extractDoc()`, whose document states 131,072 for those models, so OC1's outcome reaches the goldens. No row names them (OC8 names two tests only), so the build stopped before changing them, and before any plant, the docs, the changelog and the gates. Every other package in the run passed.

*(2026-11-27, deviation D-1, resolved by the coordinator: "Take your option (2)". The MADR's amendment 3 adds row OC8a: the 24 goldens are regenerated, and one new golden pins OC2. No other file joins the scope.)*

*(2026-11-27, Phase 2, pass 1, continued after D-1.)*

**The test names.** The PLAN's names, in amendments 1 and 2, map to the tests as built. The tests are split by row, and each covers every wire in one loop: Together, Hugging Face, and OpenCode's chat route on Zen and on Go.

| The PLAN's name | Rows | Built as |
| :--- | :--- | :--- |
| `TestOutputLimit_Together_0096`, listing case | TG1 | `TestOutputLimit_TogetherListing_0096` |
| `TestOutputLimit_Together_0096`, `_HuggingFace_0096`, `_OpenCodeChat_0096`, document and none cases | TG2, TG3, HF1, HF2, OC1, OC2 | `TestOutputLimit_Sends_0096` |
| the same tests, caller cases | TG4, HF3, OC3 | `TestOutputLimit_Caller_0096` |
| the same tests, window cases | TG5, OC4; HF4 | `TestOutputLimit_Window_0096`; `TestOutputLimit_HuggingFaceWindow_0096` |
| the same tests, stream cases | TG9, HF7, OC7 | `TestOutputLimit_Streamed_0096` |
| the same tests, request counts | TG10, HF7, OC7 | `TestOutputLimit_NoRequestAdded_0096` |
| `TestOutputLimit_Together_0096`, plain-Generate case | TG14 | `TestOutputLimit_OverStream_0096` |
| `TestLimitRefusal_Together_0096`, `_HuggingFace_0096`, `_OpenCodeChat_0096`, cap case | TG6, HF5, OC5 | `TestLimitRefusal_Resend_0096` |
| the same tests, second request | TG7, HF5, OC5 | `TestLimitRefusal_Kept_0096` |
| (OC8a) | OC2 at the wire | `TestWireGolden_ChatNoDocument_0096` |

The new tests are in `llmprovider/providers/output_limit_0096_test.go` and `llmprovider/providers/opencode/wire_nodoc_0096_test.go`.

**The goldens (OC8a).** `go test ./llmprovider/providers/opencode -run 'TestWireGoldens|TestWireGolden_ChatNoDocument_0096' -update` rewrote the 24 and wrote the new one. `python3 -B tools/goldendiff.py --rev 08d3b77` exited 1 with this summary: "opencode: changed 24, unchanged 74, new 1", "total: changed 24, unchanged 291, new 1", "goldendiff: 25 finding(s)". Every finding is explained:

* 24 are `opencode-{zen,go}-chat/<scenario>.json: requests/0/body/max_tokens: <absent> -> 131072`, one per changed golden, and nothing else. The tool allows only an 8192 changed or removed (0088 ST100), so an added limit is reported; OC8a decides it.
* One is `opencode-zen-chat-nodoc/text.json: new, and not an explicit-8192.json golden`. OC8a decides it.

No golden of any other provider changed.

**Plants.** `tools/plant.py` ran under `run-heavy`, on a scratch clone, with `-run '_0096$|TestOutputLimit_Omitted_0089|TestOutputLimit_Facts_0041|TestTogether_RequestShapes'` over `./llmprovider/providers`, `./llmprovider/providers/opencode` and `./llmprovider/providers/together`. All 26 plants were caught by their own rows' tests. One plant, OC7's request count, was first written as `catalog.LookupLimits`. It was not caught, because the gateways are not listing layers and the lookup sends no GET, so it was rewritten as a `catalog.List` call on the request path. That version was caught: "1 POST(s), 2 GET(s); want 1 and 0". The spec and output are in `<scratch>/together/plants_0096*.json` and `plants*.out`.

| Plant | Rows | Caught by |
| :--- | :--- | :--- |
| `Sends: false` (Together; Hugging Face); the chat route on `Optional` (OpenCode) | TG1, TG2, TG11; HF1, HF8; OC1, OC8 | `_Sends_0096`, `_TogetherListing_0096` and `_HuggingFaceWindow_0096`, with `TestOutputLimit_Omitted_0089` and `TestOutputLimit_Facts_0041` |
| `in.Output = cmp.Or(in.Output, 8192)` (Together; Hugging Face) | TG3; HF2 | `_Sends_0096`, its none case (and `TestTogether_RequestShapes`) |
| `Required: true` on chat | OC2, OC8a | `_Sends_0096`, `TestWireGolden_ChatNoDocument_0096` |
| `Given: false` (each wire) | TG4, HF3, OC3 | `_Caller_0096` |
| `Window: 0` (each wire) | TG5, HF4, OC4 | `_Window_0096`; `_HuggingFaceWindow_0096` |
| `out = limits.Output` (each wire) | TG9, HF7, OC7 | `_Streamed_0096` |
| a listing read on the request (each wire) | TG10, HF7, OC7 | `_NoRequestAdded_0096` |
| `Wire: wire.LimitWire{Minimum: 1}` (each wire) | TG6, HF5, OC5 | `_Resend_0096` (and `_Kept_0096`, which depends on it) |
| the memo key's `Stream: !stream` (each wire) | TG7, HF5, OC5 | `_Kept_0096` only |
| `if false && wire.OverStream(…)` | TG14 | `_OverStream_0096` |
| `Optional` sends the fact on every wire | TG13 | `TestOutputLimit_Omitted_0089`, `TestOutputLimit_Facts_0041` |

**Docs (TG12, HF9, OC9).**

* `llmprovider/providers/together/together.go:79` — fixed (package doc)
* `llmprovider/providers/together/limit.go:15` — fixed
* `llmprovider/providers/huggingface/huggingface.go:81` — fixed
* `llmprovider/providers/huggingface/limit.go:15` — fixed
* `llmprovider/providers/opencode/opencode.go:117` — fixed (package doc)
* `llmprovider/providers/opencode/opencode.go:601` — fixed (`routeCall`'s comment)
* `llmprovider/providers/opencode/opencode.go:1053` — fixed (`limitInput`'s comment)
* `llmprovider/internal/wire/limit_rule.go:25` — fixed (`LimitInput.Sends`)
* `llmprovider/providers/default_limit_0088_test.go:256` — fixed
* `llmprovider/providers/output_limit_0089_test.go:108` and `facts_0041_test.go:63` — fixed (the changed tests' comments)
* `README.md:258` — fixed
* `docs/guides/api-standards.md:129` — fixed
* `CHANGELOG.md:35` — a **Changed** entry under `[Unreleased]`

The search covered `README.md`, `CHANGELOG.md`, `docs/guides/`, `docs/architecture.md` and every `.go` file outside `docs/decisions`, for "every other wire sends none", "Kilo sends", "optional wire", "DL47" and `chooseLimit`. `docs/architecture.md` states no per-wire limit rule, so it is unchanged.

**Claims on the new sentences.** These rows are checked against a scratch clone's commit of this tree, with `tools/claims.py --rev self=<commit> --repo <clone>`, and planted false the same way. When the tree is committed, they are re-pointed.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| K1 | Together's package doc states the rule | `llmprovider/providers/together/together.go:85` | `// window leaves the request's input, as 0096-MADR decided (measured` |
| K2 | Together's package doc states the measurement | `llmprovider/providers/together/together.go:86` | `// 2026-11-27: Qwen/Qwen4-Max accepted 262,144 on a stream); with no fact,` |
| K3 | Hugging Face's package doc states the measurement | `llmprovider/providers/huggingface/huggingface.go:89` | `// accepted 258,041 on a stream); with no fact, none is sent, and the` |
| K4 | OpenCode's package doc states the chat route's rule | `llmprovider/providers/opencode/opencode.go:128` | `//     The chat route, which requires no limit, sends the model's chosen` |
| K5 | OpenCode's package doc: only responses and google send none | `llmprovider/providers/opencode/opencode.go:142` | `//     responses and google routes send none unless the caller gives one` |
| K6 | `LimitInput.Sends` names the three | `llmprovider/internal/wire/limit_rule.go:27` | `// Together, Hugging Face and OpenCode's chat route (0096-MADR, measured` |
| K7 | the changelog's entry | `CHANGELOG.md:35` | `and OpenCode's chat route (Zen and Go) now send` |
| K8 | the README names the wires that send | `README.md:258` | `it), Kilo, Together, Hugging Face and OpenCode's chat route send the` |
| K9 | the standards guide's R6 sentence | `docs/guides/api-standards.md:130` | `do the same (0096 TG1 to TG3, HF1, HF2, OC1, OC2, measured 2026-11-27);` |
| K10 | Hugging Face's `limitInput` | `llmprovider/providers/huggingface/limit.go:77` | `// Hugging Face sends the chosen fact, as 0096-MADR HF1 decided, and` |
| K11 | Together's `limitInput` | `llmprovider/providers/together/limit.go:76` | `// Together sends the chosen fact, as 0096-MADR TG1 and TG2 decided, and` |

*(2026-11-27, Phase 2, pass 1, finished after a session restart.)* The builder was cut off while `tools/gate.py` ran (its run ended with status 143, SIGTERM). The tree had not changed since 19:29, so what is recorded above held. Every gate was run again on the final tree:

* **Claims plants (K1 to K11).** Only four had been planted, and none was recorded. All 11 were planted false against a scratch clone's commit of the tree. All 11 were caught, for example "FAIL K6 llmprovider/internal/wire/limit_rule.go:27: line 27 is \"// Together, Hugging Face and OpenCode's chat route (0096-MADR, measured\"" for K6 with Together dropped.
* **Production plants.** The 26 are recorded above with the test each failed. The failing lines are in `<scratch>/together/plants.out` and `plants_oc7.out`, for example TG14: "output_limit_0096_test.go:273: no caller value: sent 262144, stream <nil>; want 262144 over a stream".
* **records-check and the 0097 records.** On the tree, `scripts/check_records.py` exits 1 with three problems, all about the 0097 records: "docs/README.md: says 203 records; there are 205" and "not indexed" for each 0097 file. Those files belong to another record. So the CI make set and `tools/gate.py` ran on a gate copy: a `git clone --shared` of the tree at `08d3b77`, with every changed and untracked file copied in except the two 0097 records. Its `git status` differs from the tree's only by those two lines.

| Gate | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...` | this Mac | 0 |
| `go test -count=1` of the 0096 tests in `providers`, `opencode`, `together`, `huggingface` | this Mac | 0 |
| `run-heavy -- make pre-add-check` | this Mac | 0 (1161 files clean, and `authcontract`) |
| `run-heavy -- go test -count=1 ./...` | this Mac | 0 (46 packages ok) |
| `run-on-host --need linux --race -- go test -race -count=1 ./...` | the Linux machine (not a fallback) | 0 (46 packages ok) |
| `run-heavy -- make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest`, on the gate copy | this Mac | 0 (records-check: 203 records, 0 problems; gate-selftest: 103 tests OK) |
| `run-heavy -- python3 -B tools/gate.py --src <gate copy>` | this Mac | 0 (30 of 30 passed) |
| `tools/goldendiff.py --rev 08d3b77` | this Mac | 1: 24 changed, 1 new, 25 findings, each explained under OC8a |
| `tools/apidiff.py v1.34.0` | this Mac | 0, no change (control: `v1.33.0 v1.34.0` lists the two names v1.34.0 added) |
| `tools/claims.py` on 0096-MADR and 0089-MADR, with the sibling checkouts | this Mac | 0 (30 of 30; 49 of 49) |
| `tools/claims.py` on this PLAN (K1 to K11), with its 11 plants | this Mac | 0 (11 of 11; 11 caught) |
| `tools/denyscan.py --changed` | this Mac | 0 (0 hits in 44 files) |
| the redact dry run (`github-disclosure.py redact`) | this Mac | 0 (would edit 0 files) |

Pass 1 met its exit: no blocker, every decided row tested, every plant caught. The code review of pass 1 came back ready, with one gap and three minor notes.

*(2026-11-27, deviation D-2, resolved by the owner, relayed by the coordinator: "Send nothing until the hold is known (Recommended)".)* The review found that on Hugging Face in a cold process, with the listing not cached, `limitInput` holds the fact only to the document's `context`. The models.dev output then goes out without HF4's hold, for example 384,000 for `deepseek-ai/DeepSeek-V4.1-Flash`. The MADR's amendment 4 records the ruling as HF10 and its consequences as HF8a. It also records the review's three minor notes as TG2a and TG12a, as the coordinator decided them. Files added to Phase 2's scope: `llmprovider/providers/generate_stream_0088_test.go` (its comment, TG12a). Pass 2 is Phase 2's second build pass.

## Execution Record, Phase 2, pass 2

*(2026-11-27, Phase 2, pass 2, base `08d3b77` with pass 1's tree.)*

* **Red first.** `TestOutputLimit_HuggingFaceCold_0096` was written, with the changed cases HF8a names, before the production change. Against pass 1's code, `go test ./llmprovider/providers -run '_0096$|TestOutputLimit_Omitted_0089|TestOutputLimit_Facts_0041'` failed on its assertions in three tests, and only those: "output_limit_0096_test.go:269: a cold process, 384,000 stated: sent 384000 (found true), OutputLimit {Tokens:384000 Sent:true Basis:fact Source:models.dev}, 1 POST(s); want none, basis none, 1 POST", and the Hugging Face cases of `TestOutputLimit_Omitted_0089` and `TestOutputLimit_Facts_0041`. The warmed 0096 cases passed. TG2a's `WithMetadata` case is a pin, which passed; its red is its plant.
* **Change.** In `huggingface/limit.go`, `Sends` became `liveWindow(limits.Window)`. This reports whether the window fact carries `listing:huggingface` evidence, that is, a live provider's window from the cached listing. No request is added. The warm case is unchanged: `TestOutputLimit_HuggingFaceWindow_0096` still sends `room(262144)`.
* **Tests.** The 0096 tests give each wire a `listing`. Hugging Face's states one live provider at 1,000,000, and the send cases cache it first (`warm0096`). `TestOutputLimit_NoRequestAdded_0096` counts Hugging Face both warm and cold. The cold count was added after the rebased HF7 plant (a listing read on the request) was caught only by `_HuggingFaceCold_0096`, not by its own row's test. With the cold count, its own test catches it: "huggingface, cold: 1 POST(s), 1 GET(s); want 1 and 0". `TestOutputLimit_Sends_0096` gained the Together `catalog.WithMetadata` case (TG2a) and a corrected comment.
* **Plants.** All ran under `run-heavy` through `tools/plant.py`, on scratch clones of the final tree. 28 were caught; the specs and outputs are `<scratch>/0096fin/plants_pass2_all.*` and `plants_pass1_rest.*`.
  * The unheld send (`Sends: true`; HF10, HF8a) was caught by `_HuggingFaceCold_0096`, `TestOutputLimit_Omitted_0089` and `TestOutputLimit_Facts_0041`.
  * The handed-in document's evidence filtered out (`if false && docOK` in `catalog/limits.go`; TG2a) was caught by "output_limit_0096_test.go:147: together, the handed-in document stating 262,144: sent 0 (found false) ... (TG2a)".
  * Pass 1's eight Hugging Face plants were rebased onto `Sends: liveWindow(limits.Window)` and were all caught by their own rows' tests.
  * Pass 1's 18 Together and OpenCode plants were run again and all caught.
* **Docs (HF10, TG12a).**
  * `llmprovider/providers/huggingface/huggingface.go:90` — fixed (package doc)
  * `llmprovider/providers/huggingface/limit.go:21` and `:78` — fixed
  * `llmprovider/internal/wire/limit_rule.go:28` — fixed (`LimitInput.Sends`)
  * `README.md:259` — fixed
  * `docs/guides/api-standards.md:131` — fixed
  * `CHANGELOG.md:42` — fixed (HF10), and `CHANGELOG.md:49` — fixed (it names the three measurements; Go's chat route was not measured)
  * `llmprovider/providers/generate_stream_0088_test.go:273` — fixed
  * `llmprovider/providers/output_limit_0096_test.go` `TestOutputLimit_Sends_0096`'s comment — fixed
  * The search covered the same paths and phrases as pass 1, and also "Hugging Face" near "send" or "limit" in every `.md` and non-test `.go` file outside the records. Nothing else states Hugging Face's rule.

**Claims on pass 2's sentences.** These are checked as K1 to K11 are, against a scratch clone's commit of the tree. K1 to K11 still hold on pass 2's tree, at the same lines.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| K12 | Hugging Face's package doc states HF10 | `llmprovider/providers/huggingface/huggingface.go:92` | `// never fetches the listing to learn it (0096-MADR HF10). The Response, and` |
| K13 | `limit.go`'s comment states HF10 | `llmprovider/providers/huggingface/limit.go:22` | `// cached listing states a live provider's window, none is sent, and the` |
| K14 | `limitInput`'s comment states HF10 | `llmprovider/providers/huggingface/limit.go:79` | `// provider's window, which holds it (HF4), and none before (HF10).` |
| K15 | the hold reads the listing's evidence | `llmprovider/providers/huggingface/limit.go:105` | `func liveWindow(window catalog.Fact) bool {` |
| K16 | `LimitInput.Sends` states HF10 | `llmprovider/internal/wire/limit_rule.go:28` | `// 2026-11-27); Hugging Face only once its cached listing states a live` |
| K17 | the changelog states HF10 | `CHANGELOG.md:44` | `that, as in a new process that has not listed models, it sends none. A` |
| K18 | the changelog names what was not measured | `CHANGELOG.md:51` | `chat route was not measured).` |
| K19 | the README states HF10 | `README.md:260` | `its cached listing states a live provider's window to hold it to), and` |
| K20 | the standards guide states HF10 | `docs/guides/api-standards.md:131` | `Hugging Face sends it only once its cached listing states a live` |
| K21 | the 0088 test's comment | `llmprovider/providers/generate_stream_0088_test.go:273` | `// whole response returned; on Together, with nothing stated and so nothing` |

**Gates, pass 2.** These ran on the final tree. The CI make set and `tools/gate.py` ran on a fresh gate copy without the 0097 records, built as in pass 1. That copy held this record before this table was added. Once the table was added, `scripts/check_records.py` (on the copy, with this record refreshed), `tools/recordlint.py` and `tools/claims.py` on this record were run again, each exiting 0.

| Gate | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`, `go vet` and the changed packages' tests (`providers`, `huggingface`, `together`, `opencode`) | this Mac | 0 |
| `run-heavy -- make pre-add-check` | this Mac | 0 (1161 files clean, and `authcontract`) |
| `run-heavy -- go test -count=1 ./...` | this Mac | 0 (46 packages ok) |
| `run-on-host --need linux --race -- go test -race -count=1 ./...` | the Linux machine (not a fallback) | 0 (46 packages ok) |
| `run-heavy --` the CI make set, as in pass 1, on the gate copy | this Mac | 0 (records-check: 203 records, 0 problems; gate-selftest: 103 tests OK) |
| `run-heavy -- python3 -B tools/gate.py --src <gate copy>` | this Mac | 0 (30 of 30 passed) |
| `run-heavy -- tools/plant.py`, three specs (28 plants) | this Mac | 0 (all caught) |
| `tools/goldendiff.py --rev 08d3b77` | this Mac | 1: 24 changed, 1 new; the 25 findings are the same as pass 1's |
| `tools/apidiff.py v1.34.0` | this Mac | 0, no change: no new name |
| `tools/claims.py` on 0096-MADR and 0089-MADR, with the sibling checkouts | this Mac | 0 (30 of 30; 49 of 49) |
| `tools/claims.py` on this PLAN (K1 to K21), and K12 to K21's 10 plants | this Mac | 0 (21 of 21; 10 caught) |
| `tools/recordlint.py` on both 0096 records | this Mac | 0 (no rule rose) |
| `tools/denyscan.py --changed` | this Mac | 0 (0 hits in 45 files) |
| the redact dry run | this Mac | 0 (would edit 0 files) |

Pass 2 met Phase 2's exit: no blocker, all 37 decided rows tested, every plant caught. Not done: the independent review of pass 2 is the coordinator's to run. Nothing is staged.
