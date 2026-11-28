---
status: accepted
date: 2026-11-27
decision-makers: repository owner
consulted: "the owner's instruction of 2026-11-27, relayed by the coordinator and quoted below; every record under docs/decisions and docs/reports at 69618a4, read only, swept for follow-ups (method below); this repository at 69618a4, read only, and a scratch clone of it at the same commit for the reproductions below"
informed: "the records each follow-up comes from (0081 to 0094); gobble-cli, whose consumer of the SDK sees any (b) row the owner rules to build"
---

<!-- markdownlint-disable MD013 MD024 -->

# Clear the SDK Follow-ups: Sort Every Unbuilt One, Build the Records-and-Tests Ones, Put the Behaviour and Live Ones to the Owner

Associated PLAN: [0095-PLAN-clear-the-sdk-follow-ups.md](0095-PLAN-clear-the-sdk-follow-ups.md)

*(2026-11-27, first draft. The tree is clean at `69618a4` on `main` (v1.34.0 plus the records commit); `python3 scripts/check_records.py --next` printed `0095` there. 17 rows are open: the 15 (b) rows, which wait for the owner's rulings, and the 2 (c) rows, which wait for the owner's word.)*

## Context and Problem Statement

The owner's instruction, quoted exactly: **"close out 0086 and clear the sdk follow-ups"**. The 0086 half is closed in [0086-PLAN-golangci-lint-v2-15-0.md](0086-PLAN-golangci-lint-v2-15-0.md)'s execution record (2026-11-27). This record is the other half.

Records 0081 to 0094 each left findings outside their rows as follow-ups, "recorded, not built". They are spread over twenty files, several were closed by later records without a pointer back, and some need the owner's ruling or a live measurement before anyone can build them. This record gathers each one, says which kind it is, and decides how each kind is cleared.

### How the follow-ups were found

A read-only sweep (`<scratch>/sdkfu/sweep.py`) listed each line of `docs/decisions/*.md` and `docs/reports/*.md` naming a follow-up, "not built" or "not implemented", an `FU` id, or an open item ("left open", "stays open", "open:" and the like): 618 lines in 199 files, 320 of them in records 0080 and later. Of those, the 138 naming a follow-up, "not built" or an `FU` id were each read, and the section of each that names a follow-up; the 182 matched only as "open" were scanned (139 after dropping row counts such as "0 open"), and each that could still be live (0080-PLAN's Q8, 0084-PLAN's B3, 0089-PLAN's DL65 docs step, 0091's OQ12, 0092's ST11) was found settled later in its own record (0080-PLAN:742, 0084-PLAN:1947, 0089-PLAN:1339 to :1348, 0091-MADR:843 to :853, 0092-MADR:704). The 102 non-"open" lines before 0080 were read too: each is a design choice recorded as not built (`Continuer`, 0004-MADR), work a later record did (0038-PLAN's follow-up, built in 0041-PLAN; 0046 and 0049 are themselves follow-up records), or a consumer's, so none is an open SDK follow-up. The coordinator's list of items noted during this program was searched for separately; each is a row below or is marked "not found".

### Kinds

* **(a)** docs, test, tool or record only, with no change to what the SDK does: built by the PLAN.
* **(b)** a change to production code: a reproduction of today's behaviour, the evidence, and three options; built only as the owner rules.
* **(c)** needs a live measurement: what, the key's name, the request count and the cost; run only on the owner's word.
* **(d)** already done, or no longer true: the evidence.

### Reproductions (2026-11-27, scratch clone of `69618a4`, never the tree)

`<scratch>/sdkfu/repro_run.py` clones the tree, copies one test file into a package, and runs it. Each is red today, on its assertion; each failure line is quoted in the row it reproduces.

| Id | Test (file in `<scratch>/sdkfu/repro/`) | Package | Exit |
| :--- | :--- | :--- | :--- |
| RP1 | `TestRepro_FU1_UnstoppedBlock` (`repro_fu1_messages_test.go`) | `internal/wire/messages` | 1 |
| RP2 | `TestRepro_0092_824_LastIndexFallback` (`repro_responses_test.go`) | `internal/wire/responses` | 1 |
| RP3 | `TestRepro_0091_1015_SummaryAndRaw` (`repro_responses_test.go`) | `internal/wire/responses` | 1 |
| RP4 | `TestRepro_0089_KiloRangeFullwidth` (`repro_kilo_fullwidth_test.go`) | `providers` | 1 |

## Decision Drivers

* **Nothing recorded is lost.** Each follow-up is a row here with its source `path:line`, so a later reader finds it in one place.
* **Measured, not assumed** (0091-MADR's driver): a behaviour no probe measured is not built on; a (b) option that rests on one says so.
* **Red first.** A (b) row the owner rules to build starts from its reproduction, already red on its assertion.
* **No release in scope.** The owner tags and releases; a built (b) row goes under `[Unreleased]`.

## Considered Options

* A. One record that sorts every follow-up into (a) to (d), builds (a), and puts (b) and (c) to the owner
* B. A record per follow-up, each in its owning record's numbering
* C. Leave the follow-ups where they are recorded

## Decision Outcome

Chosen option: "A", because the owner asked for the follow-ups to be cleared as one piece of work, and one table lets the owner rule on every (b) row in one prompt; option B repeats the review and the gates fifteen times, and option C leaves twenty files to read for the open work.

### D1. The rows

The state table below holds one row per follow-up. An (a) row is decided, with its check and its plant. A (d) row is decided, with its evidence. A (b) row is open until the owner picks one of its three options, which then becomes its outcome, with the reproduction as its red test. A (c) row is open until the owner's word, and its measurement's result then decides the (b) row it feeds, or is recorded where nothing is to change.

### D2. Owning records

This record does not edit the records the follow-ups come from. When a row is built or closed, the PLAN adds one dated line to its owning record's follow-up text pointing here, as the records rule allows (dated additions only).

### Consequences

* Good, because every unbuilt follow-up has one row, one kind and one next step.
* Good, because the (b) rows reach the owner with a red test already in hand.
* Bad, because the (b) rows wait on the owner, so the record stays `proposed` until they are ruled.
* Neutral, because no exported name changes under the (a) rows.

### Confirmation

* Each (a) row's check passes and its plant, where one applies, is caught on a scratch clone through `tools/plant.py`.
* Each (b) row the owner rules to build is red at the phase's base by its reproduction, green after, with its plants caught.
* `tools/records.py open` no longer lists this record once every row is decided and built or closed.

## State Table

### (a) Docs, test, tool or record only

| Row | Follow-up and source | Outcome | Check | Plant |
| :--- | :--- | :--- | :--- | :--- |
| FU16 | `docs/guides/migrating-from-mcplib.md:53`, the `APIError` row, names `Kind`, `Code`, `Reason`, `Retryable()`, `ResetAt` and `Limit` but not `Tool` (0084-MADR D2, D6; `llmprovider/api_error.go:188`) or `Observed` (0089-MADR DL78; `api_error.go:199`) | decided: the row gains both fields, each with its record | `make markdownlint links-check docs-check` exit 0; the row names both | none applies (prose) |
| FU17 | `Options.NoEndpointPrompt`'s doc (`wizard/configure.go:247` to `:254`) does not say that a run whose context ends during the check returns `wizard: endpoint check ended: ...` (`configure.go:837`); 0090-PLAN:460 | decided: one sentence added, naming 0090-MADR S78 | `make docs-check`; the behaviour is already pinned by `wizard/resolve_base_url_0090_test.go:285` | none applies (comment) |
| FU18 | `scripts/check_tables.py`'s test-variable check runs one way: a `*_TEST_*` variable `AGENTS.md` names after its test is gone is not reported (0090-PLAN:452) | decided: it is reported, `AGENTS.md names <VAR>, which no test file holds` | a new `scripts/test_gates.py` case, red at the base | the reverse check removed: the case fails |
| FU19 | Its `TEST_VAR` matches only a double-quoted literal; a backquoted one is not seen (0090-PLAN:455) | decided: a raw-string literal is matched too | a new `scripts/test_gates.py` case, red at the base | the pattern back to double quotes only: the case fails |
| FU20 | `tools/claims.py --changed` reads `0013-MADR`'s findings table as an evidence index and exits 1 (`the matched cell is not code`), though that record has no claims marker (0093-PLAN:726 to :728, :751) | decided: a record with no `<!-- claims: -->` marker has no evidence index; `--changed` passes over it, and a record named on the command line reports "no evidence index", exit 0 | a new `tools/test_tools.py` case, red at the base | the marker test removed: the case fails |
| FU21 | The 0084 recorder's left-overs (0084-PLAN:1809, :2012, :2016; `llmprovider/toolschema_0084_record_test.go`): UUID over-redaction (`:507`); the overlapping `request_id` and `req_` rules (`:514`, `:515`); the 4 MiB read limit (`:368`); symlinks under the output directory; a provider-build failure's text dropped; the live test's `each` literal never built offline | decided: (i) closed as records only: over-redaction errs safe, the two rules write the same `req_REDACTED`, no recorded body nears 4 MiB (the largest of the 57 files under `toolschema-0084/` is 1,545 bytes), the output directory is the developer's own; (ii) built: a provider-build failure's text kept, redacted; an untagged test that builds `each` and fails a literal that never calls `run` | (ii) two new tests in `llmprovider`, red at the base | (ii) each change reverted: its test fails |
| FU22 | The models.dev copyright line in `NOTICE` could not be verified against upstream (0089-PLAN:1383) | decided: one public read of the upstream licence file (no key, no provider call); the line corrected or confirmed, with the date read | the line matches the upstream text | none applies |

### (b) Production code: open until ruled

| Row | Follow-up and source | Today, reproduced | Evidence | Status |
| :--- | :--- | :--- | :--- | :--- |
| FU01 | 0092 FU1: on Messages, a block the stream never stopped is in `EventDone`'s response but never yielded as `EventItem` (0092-MADR:67, :212; M9) | RP1: `events ["text_delta:Hello" "done"], EventDone's text "Hello"; want ["text_delta:Hello" "item:MessageItem" "done"]` | `llmprovider/internal/wire/messages/messages.go:559` to `:566` folds the open block into the response at `message_stop` and yields nothing for it | open: OQ1 |
| FU02 | 0092: when the server's last call is dropped as `incomplete`, `lastIndex` falls back to the response's own last item, an earlier `{}` call (0092-MADR:824) | RP2: a whole `now {}` call, then an incomplete call: `Events err llmprovider: empty answer: max_output_tokens: responses: the answer has no content`, the same from `ReadStream` | `llmprovider/internal/wire/responses/responses.go:1231` to `:1253`: the last call's `call_id` is not in the response, so `fallback` (`:1232`) is returned at `:1252` | open: OQ2 |
| FU03 | 0091: a server that streams both summary and raw reasoning deltas for one item keeps only the summary in the final item (0091-MADR:1015; 0091-PLAN:187) | RP3: `reasoning deltas yielded ["S" "R"] (joined "SR"); final output ["reasoning:S/" "message:ok"]` | `responses.go:611` to `:627`: `reasoning_text` is read only when the summary holds none (ST10, ST63) | open: OQ3 |
| FU04 | 0089: Kilo's range form is read with ASCII `[`, `,` and `]` only; fullwidth `［`, `，`, `］` are not (0089-PLAN:1298) | RP4: `sent [200000], err llmprovider: invalid request: kilo HTTP 400 AI_APICallError: The max_tokens parameter is illegal.：限制数值范围［1，131072］; want [200000 131072]` | `llmprovider/internal/wire/limit_refusal.go:120` | open: OQ4 |
| FU05 | 0089: `wire.Send`'s uncounted branch (`sends--`) is no longer reached since ST88 (0089-PLAN:1142, :1151); production code, no behaviour change | no test reaches it: every resend at a stated limit is read over the stream (`StreamEventsRetry`, `llmprovider/internal/wire/sse.go:162`) | `llmprovider/internal/wire/retry.go:41` to `:43` | open: OQ5 |
| FU06 | 0089: `llmtest`'s W12 garbled check assumes a plain reply; a provider outside the module whose `Generate` reads over the stream would fail it (0089-PLAN:1149) | the module's ten harnesses all name a limit and stay plain, so no harness here shows it | `llmprovider/llmtest/llmtest.go:895` to `:911` wants one send and `ErrIncomplete` | open: OQ6 |
| FU07 | 0090: after the context check, the run's context can still end before `Notify` or `Confirm` (0090-PLAN:457) | a window between two calls; no test can hold it open without a hook | `wizard/configure.go:871` checks, `:878` to `:883` notify and ask | open: OQ7 |
| FU08 | 0085: a non-blank environment value is kept untrimmed (ST39, as ruled), so `"key\n"` reaches `Result.APIKey` and `New` refuses it for the control character, with no hint at the offer (0085-MADR:1366) | pinned today by `TestNonBlankEnvAndSavedKeyUntrimmed_0085` (`wizard/empty_key_routes_0085_test.go:65`) | 0085-MADR ST39 (D5, the owner's "offered and used as now, untrimmed") | open: OQ8 |
| FU09 | 0082: a held session that names no account (every Grok sign-in; a plan session whose ID token carries none) adopts a stored record of any account on the same issuer and client (0082-MADR:1012 to :1018, N-5 at :1176 to :1180) | pinned today by `TestSameSignInEdges_0082`, "no account on the held side": adopted (`llmprovider/auth/held_session_0082_test.go:456`) | 0082-MADR "Not fixed here; a future record" | open: OQ9 |
| FU10 | 0082 N-a: a client record whose JSON is valid but whose `client_id` is garbage (invalid UTF-8 decoded to U+FFFD) is used and saved again with no notice (0082-MADR:1596 to :1604) | no test pins it; the PLAN's first step for this row writes it red | 0082-PLAN:1480 to :1484 | open: OQ10 |
| FU11 | 0084: Gemini's refusal names the tool in single quotes (`function tool 'record_city'`), a form D3 does not read (0084-PLAN:2169, :2320; 0084-MADR:1334) | pinned today: `TestToolSchemaRefusal_RecordedFallback_0084` holds `interactions/gemini-O1-root-union.json` and `-O2-two-bad.json` as `ErrInvalidRequest`, `Tool` empty (`llmprovider/toolschema_refusal_0084_test.go:89`, `:90`) | the recorded bodies under `llmprovider/internal/wire/testdata/toolschema-0084/interactions/` | open: OQ11 |
| FU12 | 0084: OpenCode's Google route refuses a tool schema holding `additionalProperties` as a plain 400 naming no tool (0084-PLAN:2161 to :2166) | pinned today: the same test, `zen-google-O1`, `-O2` (`toolschema_refusal_0084_test.go:91`, `:92`) | `Unknown name "additionalProperties" at 'tools[0].function_declarations[1].parameters'` (0084-MADR:1326) | open: OQ12 |
| FU13 | 0084: `perDayQuota` is read only over the violations kept, at most 32; a per-day violation past the 32nd reads as `ErrRateLimited` (0084-PLAN:2832) | no recorded body comes close; no test holds more than 32 violations, so the PLAN's first step for this row writes one, red on its assertion | `llmprovider/api_error.go:869`, `:870` | open: OQ13 |
| FU14 | 0084: `io.ReadAll` over `io.LimitReader` grows by doubling, so an error body of up to 1 MiB can hold about twice that while it is read (0084-PLAN:2778) | bounded by the limit; not measured here: under (2) or (3) the PLAN's first step measures the allocation with a benchmark at the base | `llmprovider/api_error.go:322` | open: OQ14 |
| FU15 | 0084: Claude computes its deferral form twice (`llmprovider/providers/claude/claude.go:428`, `:457`), and `classifyAPIError` takes a positional `reply bool` (`api_error.go:606`) (0084-PLAN:2458); production code, no behaviour change | none: a refactor | as cited | open: OQ15 |

### (c) Live measurements: open until the owner's word

| Row | What | Key | Requests | Cost | Feeds |
| :--- | :--- | :--- | :--- | :--- | :--- |
| FU23 | One live test on the `gemini` provider (the Interactions API, `gemini-3.8-flash`), two streamed calls. (1) A plain prompt: whether `interaction.completed` carries `steps` (present or not, count, step types), whether it repeats the answer's text in them (so whether the 16 MiB event cap bounds the answer, 0091 ST72), and whether any `step.delta` of type text arrives for an index before its `step.start` (M11). (2) A prompt with one tool that takes no arguments, the call forced: whether any `arguments_delta` arrives for it, and whether it is non-empty (0088-MADR ST110), and whether the completed interaction carries the call step (M10). Allowlisted output only: event types in order, step types, field names, counts, booleans | `GEMINI_API_KEY`, by name only | 2 | two short calls, each capped at 1,024 output tokens with prompts of a few dozen tokens: at most about 2,200 tokens billed in all, at the model's rate, which no record states | FU2 (0092-MADR:213; M10, M11), ST72 (0091-MADR:1251), ST110 (0088-MADR:1644) |
| FU24 | Kilo's control run on `z-ai/glm-5.3-flashx`: one streamed request with `max_tokens` 131,072 sent. A `length` finish at 128,000 puts the cut in the model; one at 131,072 puts it in what Kilo fills when none is sent (0089-MADR:972) | `KILO_API_KEY`, by name only | 1 | up to 131,072 output tokens billed at the model's rate, which no record states | the wording of Kilo's cut in `kilo.go` and the changelog only; the interim stands either way |

### (d) Already done, or no longer true

| Row | Follow-up and source | Evidence |
| :--- | :--- | :--- |
| FU25 | `terminal_0092_test.go`'s failure message says "10 MiB of deltas" (0092-MADR:756) | Fixed in v1.34.0 (`8f88c9b`): the message, now at `llmprovider/internal/wire/responses/terminal_0092_test.go:511`, reads "15 MiB of reasoning and 10 MiB of text deltas". The guard at `:476` to `:478` holds `StreamAnswerLimit` in [25 MiB, 35 MiB). On a scratch clone (`tools/plant.py`, `<scratch>/sdkfu/plant_collect.json`) 3 of 3 plants caught: the limit at 48 MiB and at 16 MiB, each `terminal_0092_test.go:477: StreamAnswerLimit is 48 MiB` (16 MiB) `...need it in [25 MiB, 35 MiB)`; ST30's plant (`Collect` counting a message's text): `terminal_0092_test.go:511: 15 MiB of reasoning and 10 MiB of text deltas, then the terminal's message: 0 bytes of text (... over 32 MiB)`. Green in the tree, exit 0 |
| FU26 | The 0088-MADR `:1505` pointer (R5) | Done in v1.34.0: the annotation at 0088-MADR:1505 names `limit_refusal.go:74`; `git show v1.34.0:llmprovider/internal/wire/limit_refusal.go` line 74 is `ResponsesLimits = LimitWire{Minimum: 1}`; it landed in `223bb8d` and `8f88c9b` |
| FU27 | 0081-PLAN:2717's follow-ups and S78 | Built by 0090: P-20 as T12, P-16 as T13, S78; the stale comment closed as T16 (0090-MADR:24 to :27, :209, :210, :213) |
| FU28 | 0083-PLAN:1754's three follow-ups | Two done and the third dropped, 2026-11-22 (0083-PLAN:1782 to :1786) |
| FU29 | 0087: the R3 request's words; A-9's K4 follow-up; the `TestSecret_LateLineFeedDuringWait_0087` flake | The probe says "type one letter" (`llmprovider/live_raw_enter_0087_windows_test.go:159`); OR20 ruled "Leave it, recorded (Recommended)" (0087-MADR:234); the flake taken by 0087's A-9 and Phase 5 (0094-PLAN:299) |
| FU30 | 0082: the corrupt client record; the lossy store's failed first save; the flaky `TestVerifyIDToken_UnknownKidRefetchesOnce` | Fixed by R-9, option (a) (0082-MADR:1331); the limit no longer holds, F-4 (0082-MADR:1314 to :1320); fixed under 0083 ST64 (0083-MADR:1272) |
| FU31 | 0088-PLAN:958: `changelog-check` does not read the compare links | 0094's RS9 and RS31 (0094-MADR:152, :174) |
| FU32 | 0088 R4 and R5 | Ruled in 0088-MADR's nineteenth amendment (:1739 to :1762) |
| FU33 | 0084: `redactInput`'s 2 KiB clip untested | Struck: `TestClassify_16KiBTokensUnder1ms` catches it (0084-PLAN:2466 to :2469) |
| FU34 | 0088-PLAN:1050: 0091's records not annotated for R1 to R3 | Done in the twentieth entry (0088-PLAN:1050's own annotation) |
| FU35 | The Hugging Face reasoning detail of the 0091 report | No record, report or doc says where round 1's cut fell; 0091-PLAN:157 records that the docs claim none. Nothing is false. Knowing it would take a re-run of round 1 (a (c) the owner has not asked for) |

### Not found

* **denyscan host names.** No record names this follow-up. The code reads this machine's account and host only (`tools/denyscan.py:37` to `:42`), so another fleet host's name is not scanned for; if the owner wants that, it is an (a) tool row for a later amendment.

Counts: 35 rows: 7 (a) decided, 15 (b) open, 2 (c) open, 11 (d) decided; 1 not found. The record is `proposed` until OQ1 to OQ15 are answered.

## Owner Questions

Each is put as three options, the recommended first. A ruling is quoted exactly and becomes its row's outcome and test.

| Question | (1) | (2) | (3) |
| :--- | :--- | :--- | :--- |
| OQ1 (FU01). A Messages block the stream never stopped | **Yield it** (Recommended): each block folded at `message_stop` is yielded as `EventItem` before `EventDone`, as every other wire does; a cut call stays dropped. RP1 is its red test | **Leave it, documented**: `docs/architecture.md` says Messages can end with an item in `EventDone` never yielded | **Refuse it**: an unstopped block on `end_turn` is `ErrIncomplete` |
| OQ2 (FU02). The server's last call dropped as incomplete | **Nothing else is cut** (Recommended): when the terminal's last call is not in the response, no item is last (-1), as ST37 already does for a call it has yielded; RP2 is its red test | **Leave the fallback, documented** in `lastIndex`'s comment and 0092 ST37 | **Keep the earlier call, with a warning** naming it as possibly cut |
| OQ3 (FU03). Summary and raw reasoning for one item | **Leave it, recorded** (Recommended): no probe has seen a server send both for one item (the driver "measured, not assumed"); the item keeps the summary | **Keep both**: the item's text is the summary, then the raw text, matching what was streamed | **Stream what is kept**: once a summary delta comes for an item, its raw deltas are not yielded |
| OQ4 (FU04). Kilo's range form in fullwidth | **Leave it, recorded** (Recommended): no reply has been recorded in that form; today's failure is the safe one, the refusal returned (DL95) | **Read both**: the pattern takes `［`/`[`, `，`/`,`, `］`/`]`; V1 to V6 still apply; RP4 is its red test | **Normalise first**: fullwidth punctuation folded to ASCII before every form is matched |
| OQ5 (FU05). `wire.Send`'s unreached `sends--` | **Remove it** (Recommended), with its doc sentence; no behaviour changes, since nothing reaches it | **Test it**: a plain send resent at a validated stated limit, reached by a test | **Leave it**, recorded as unreached |
| OQ6 (FU06). `llmtest` W12 and a provider that reads `Generate` over the stream | **Document it** (Recommended): `Harness.Garbled`'s doc says W12 assumes a plain reply; no check changes | **A harness field**: a provider that reads over the stream sets it, and W12 then wants that form's count, the one resend 0021-MADR D1 allows | **Detect the form** from the request W12 sees (`stream`, `Accept`) and want that form's count |
| OQ7 (FU07). The window after the wizard's context check | **Leave it, recorded** (Recommended): any check before a call leaves a window; the run's caller sees the ended context at the next read | **Check again** between `Notify` and `Confirm`: narrows the window, does not close it | **Pass the context to the prompter's calls**, so a call after the end returns at once: an exported `Prompter` change |
| OQ8 (FU08). An environment key ending in a line feed | **Say so at the offer** (Recommended): ST39 stands (kept untrimmed, as ruled), and the offer adds a notice when the value holds a control character | **Trim a kept value**, as a typed one is: reverses ST39's ruling | **Leave it**, as ruled |
| OQ9 (FU09). An account-less held session adopting another account's record | **Leave it, a stated limit** (Recommended): with no account id, nothing the service gives tells two accounts apart on one client | **Compare the ID token's subject** when both sides carry one: needs a measurement that Grok's ID token carries a stable `sub` (a (c) of one sign-in, the owner's) | **Refuse adoption when either side names no account**: breaks T10's "no account on one side" sibling rotation, a guard |
| OQ10 (FU10). A client record with a garbage `client_id` | **Treat it as unreadable** (Recommended): a `client_id` that is not valid printable text is handled as R-12 handles a record that cannot be decoded, a warning and a new registration | **Name the file** in the authorize error, so the person can delete it | **Leave it**, a stated limit |
| OQ11 (FU11). Gemini's single-quoted tool name | **Read it** (Recommended): `function tool '<name>'` names that tool, held to the tools the request sent (D6); the two recorded bodies move to the recognised list | **The kind only**: `ErrToolSchemaRefused`, `Tool` empty | **Leave it** as `ErrInvalidRequest` |
| OQ12 (FU12). OpenCode Google's `additionalProperties` refusal | **Leave it** (Recommended): the refusal is of a field the caller wrote, and it names no tool | **Read the index**: `function_declarations[N]` names the request's tool N, as Claude's `tools.N.` does | **Strip the field** on that route before sending |
| OQ13 (FU13). The per-day quota past the 32nd violation | **Read every violation** (Recommended): `perDayQuota` over all of them, only the stored list capped | **Raise the cap** | **Leave it**: no recorded body comes close |
| OQ14 (FU14). The error body's read buffer | **Leave it** (Recommended): bounded by 1 MiB | **Size the buffer** from `Content-Length`, up to the limit | **Read into a buffer of the limit** |
| OQ15 (FU15). Claude's deferral form computed twice; the positional `reply bool` | **Leave both** (Recommended): no behaviour, churn only | **Both**: the form computed once and passed; a named type for `reply` | **The form only** |

## Pros and Cons of the Options

### A. One record that sorts every follow-up into (a) to (d), builds (a), and puts (b) and (c) to the owner

* Good, because one table, one review and one prompt.
* Good, because the (b) rows carry a red reproduction from the start.
* Bad, because one record spans many subjects, so its PLAN has more phases than a single fix's.

### B. A record per follow-up

* Good, because each record is small.
* Bad, because fifteen reviews, fifteen gate runs and fifteen prompts for the owner.

### C. Leave the follow-ups where they are recorded

* Good, because nothing is written.
* Bad, because the owner asked for them cleared, and the open work stays spread over twenty files.

## More Information

* The 0086 half of the owner's instruction: [0086-PLAN-golangci-lint-v2-15-0.md](0086-PLAN-golangci-lint-v2-15-0.md), execution record of 2026-11-27.
* Scratch evidence: `<scratch>/sdkfu/` (`sweep.py` and its output, `repro/`, `repro_run.py`, `plant_collect.json`, `out/`).

## Amendment (2026-11-27, first): the owner's rulings

Additions only; nothing above is rewritten. The owner's rulings, relayed by the coordinator on 2026-11-27, quoted exactly with their labels:

| Question | Ruling, quoted | Option | Rows |
| :--- | :--- | :--- | :--- |
| OQ1 (FU01) | "Pass the block on as an item (Recommended)" | (1) | FU01 |
| OQ2 (FU02) | "Cut nothing else (Recommended)" | (1) | FU02 |
| OQ3 to OQ15 (FU03 to FU15) | "Accept all recommended (Recommended)" | (1) each; FU15 see below | FU03 to FU15 |
| FU23, FU24 | "Both" | run both | FU23, FU24 |

**FU15, a discrepancy, recorded.** This record's OQ15 option (1) is "Leave both". The prompt the owner read named the recommendation as "compute the deferral form once". The coordinator's direction, given with the rulings: build option (3), "The form only", since that is what the owner read; the positional `reply bool` stays as it is.

The coordinator's further terms for the build, recorded as given: every behaviour fix gets a red-first test and a plant; every exported API change shows in `apidiff`, and one not expected is a stop; each live call is tried at most twice (one attempt and one retry), a failure or an unexpected reply is recorded, and a measurement that changes a decided row's premise (such as the Interactions 16 MiB cap) is a stop.

### The rows, decided

| Row | Status | Outcome | Test (red at `69618a4` on its assertion unless noted), and its plant |
| :--- | :--- | :--- | :--- |
| FU01 | decided | At `message_stop`, each block the stream never stopped whose item the response kept is yielded as `EventItem`, after the held calls and before `EventDone`; a cut call the response dropped is not yielded | `TestEvents_UnstoppedBlock_0095` (`internal/wire/messages`): RP1's case, red; a cut `tool_use` left open on `max_tokens`, not yielded (green, a guard); plant: the new yield removed |
| FU02 | decided | When the terminal output's last call has a `call_id` the response does not hold, as when it was dropped as `incomplete`, no item is last (-1); an earlier whole `{}` call is kept | `TestTerminalFill_LastCallDropped_0095` (`internal/wire/responses`), RP2's case; plant: the fallback restored there |
| FU03, FU04, FU07, FU09, FU12, FU14 | decided | as their option (1): left as they are, each a stated limit recorded here; no code | none: records only |
| FU05 | decided | `wire.Send`'s uncounted branch, and `uncounted` if nothing else uses it, and their doc sentence are removed; no behaviour changes | the package tests stay green; a coverage run at the base shows the branch unreached; no plant (nothing to catch) |
| FU06 | decided | `Harness.Garbled`'s doc says W12 assumes a plain reply, so a provider whose `Generate` reads over the stream is not checked by it; no check changes | `make docs-check` |
| FU08 | decided | ST39 stands; when the environment value offered holds a control character, the wizard first notifies, naming the variable and never its value, that kept as it is it will be refused | `TestEnvKeyControlNotice_0095` (`wizard`): the line-feed value, red; a clean value, no notice (green, a guard); plant: the notice removed |
| FU10 | decided | `FileTokenStore.LoadClient` reports a record whose `client_id` is not printable text (a replacement character, a control character) as undecodable, by a new marker in `internal/recorderr` that `signinflow.Undecodable` reads; the sign-in replaces it, as for a record that cannot be decoded | `TestLoadClient_ImplausibleClientID_0095` (`llmprovider/auth`), red; `TestUndecodable_Implausible_0095` (`internal/signinflow`), red from its plant, the marker being new; plants: the check removed; `Undecodable` not reading the marker |
| FU11 | decided | A Gemini 400 whose code is `invalid_request` and whose message holds `Invalid schema in function tool '` is `ErrToolSchemaRefused` on `gemini`, and the `gemini` provider names the one tool it holds, on `Generate` and `Stream` | the two recorded bodies move to `recognised0084`, so `TestToolSchemaRefusal_Classify_0084` is red at the base; `TestRead_Gemini_0095` (`internal/toolschema`); `TestGeminiNamesRefusedTool_0095` (`providers`); plants: the form not read for `gemini`; `Generate` not naming; `Stream` not naming |
| FU13 | decided | `perDayQuota` is read over every violation of the kept details; only the stored `Quotas` list stays capped at 32 | `TestPerDayQuota_PastTheCap_0095` (`llmprovider`): 33 violations, only the 33rd per-day, is `ErrQuotaExhausted`, red; plant: the read capped again |
| FU15 | decided | option (3): Claude's `call` takes the form that `tools` computed and uses it for its warnings; no behaviour changes | the 0040 deferral warning tests stay green; plant: `call` given another form for its warnings, caught by them |
| FU23, FU24 | decided: run | as the (c) table; their results go in a later amendment | the live tests themselves |

Counts after this amendment: 35 rows, 35 decided, 0 open; 1 not found. The record is `accepted`.

## Amendment (2026-11-27, second): the live results, and two rows stopped

Additions only. The two live tests ran once each, on the owner's "Both", with `LLMPROVIDER_LIVE_0095` set and the keys read from the environment by name only (`llmprovider/live_0095_interactions_test.go`, `llmprovider/live_0095_kilo_test.go`). Output allowlisted: event types and counts, step and delta types, field names, counts, booleans; no text, no key.

### FU23, Gemini's Interactions stream (`gemini-3.8-flash`, two streamed calls, one attempt each)

| Call | Status, SDK result | Events, in order | Step and delta types | `interaction.completed`'s interaction | Counts |
| :--- | :--- | :--- | :--- | :--- | :--- |
| plain prompt | 200, no error, one `MessageItem` | `interaction.created`, `interaction.status_update`, then two steps, each `step.start`, `step.delta`, `step.stop`, then `interaction.completed`, then one data line the measurement does not decode as JSON | steps `thought`, `model_output`; deltas `text`, `thought_signature` | fields `created`, `model`, `object`, `service_tier`, `status`, `updated`, `usage`: no `steps` | text deltas before their `step.start`: 0 |
| one tool taking no arguments, forced | 200, no error, one `FunctionCallItem` | as above, the second step with no `step.delta` | steps `thought`, `function_call`; deltas `thought_signature` only | the same fields, no `steps` | `arguments_delta`: 0 |

Read against the rows:

* **0092 FU2, M10.** Google's terminal interaction carries no `steps`, so a step only in the terminal does not arise on this service; 0092-MADR ST24 stays out of scope, and nothing is built.
* **0092 FU2, M11.** No text delta came before its `step.start` (0 in the run); nothing is built.
* **0091 ST72.** `interaction.completed` repeats no step, so no one event carries the answer, and the 16 MiB event cap does not bound an Interactions answer: `StreamAnswerLimit`, 32 MiB, does, which is how ST72 already read this wire with the unknown. Its premise holds; no row changes.
* **0088 ST110.** For a call that takes no arguments, Google sent no `arguments_delta`. That is the case 0088-MADR's "What is unmeasured" names: under ST110 (a), such a call that ends a cut streamed Interaction is dropped and named in a `WarningToolCallCut`, the direction that record calls safe. ST110's decided outcome is unchanged; the case is now measured, not assumed. A call that is not last, or a reply that was not cut, is unaffected (this run's call was returned).

### FU24, Kilo's control run

One attempt: the service answered HTTP 401, with no stream. Not retried, as the coordinator's terms say. The measurement is not made; the interim wording of 0089-MADR:972 stands. A rerun needs a Kilo key the service accepts, on the owner's word.

### FU20 and FU21, stopped in pass 1

* **FU20.** Built as decided ("a record named on the command line reports 'no evidence index', exit 0"), the existing case `TestClaims.test_no_revision_names_the_fix` (`tools/test_tools.py:1245`) fails: it requires a record with rows and no marker to exit 1 naming `--rev self=REV`. No row names that case. The change was taken back; the tool is as at `69618a4`. Put to the owner.
* **FU21 (ii).** Moving the live test's runner into the untagged file trips an existing plant of `TestToolSchemaRecord_CallAllowlist_0084` ("a run given something but t.Logf": `"run(t.Logf)" does not occur once in live_0084_toolschema_measure_test.go`), and keeping a provider-build failure's text needs a record outcome the recorder does not have. No row names either. Both changes were taken back; the recorder is as at `69618a4`. FU21 (i), the records-only closures, stands. Put to the owner.

Counts after this amendment: unchanged, 35 rows, 35 decided; FU20 and FU21 (ii) decided and not built, awaiting the owner.

### Note (2026-11-27): two tests named for FU10 and FU11, found at the gates

Additions only. FU11's row also governs `TestRead_0084` (`llmprovider/internal/toolschema/toolschema_0084_test.go:47`), which pinned Gemini's recorded refusal as `None`, the fact OQ11's ruling reverses: it now expects `Gemini`, and OpenCode Zen Google's record stays `None` (FU12). FU10's row also gains `TestImplausibleShowsAndUnwrapsItsCause_0095` (`internal/recorderr`), which `coverage-check`'s 80 % floor requires of the new marker. Each is caught by its plant; the PLAN's execution record of 2026-11-27 holds both. The live tests read `LLMPROVIDER_LIVE_0095`.

## Amendment (2026-11-27, third): the owner's rulings on FU20, FU21 (ii) and FU24

Additions only. The owner's rulings, relayed by the coordinator, quoted exactly:

| Row | Ruling, quoted | Outcome | Test, and its plant |
| :--- | :--- | :--- | :--- |
| FU20 | "Only for --changed runs (Recommended)" | under `--changed`, a changed record with no claims marker keeps no evidence index and is passed over, not a finding, unless it is also named or `--rev` is given; a record named on the command line is checked as before, so `test_no_revision_names_the_fix` stands | `TestClaims.test_changed_record_with_no_marker_is_passed_over` (`tools/test_tools.py`), red at `69618a4` (`AssertionError: 1 != 0 : MISSING E1 demo.go:3: no revision for 'self' ...`); plant: the new branch off, caught on the same line |
| FU21 (ii) | "Build both" | (1) the live runner moves to the untagged `liveCaseRunner` in `llmprovider/toolschema_0084_record_test.go`, and the allowlist plant "a run given something but t.Logf" anchors there; (2) a provider-build failure's text kept: **stopped**, see below | (1) `TestToolSchemaRecord_LiveRunner_0095`, red from its plant (the runner is new): `toolschema_record_runner_0095_test.go:33: the live runner made 0 sends and wrote 0 records; want 2 of each`; `TestToolSchemaRecord_CallAllowlist_0084` green with the anchor moved |
| FU24 | "I'll fix the key; rerun", then "kilo key is fixed, rerun FU24" | one rerun, exactly as recorded | see below |

The coordinator accepted the `TestRead_0084` change of the second amendment's note, as following from the owner's FU11 ruling, with its own plant.

**FU21 (ii), part (2), stopped by the coordinator's own condition.** Keeping a provider-build failure's text needs an outcome the recorder lacks. The recorder's outcomes are decided by 0084-MADR ST67 (`accepted`, `refused`, `unreachable`, `other`, `skipped`), and each record file holds one of them: a new one changes a recorded file format, which the coordinator's terms make a stop. Not built. Options: (1) **leave it, recorded** (Recommended): a build failure is the recorder's own fault, already failing the case with a fixed text, and the 0084 recording run is finished; (2) a new record of its own decides a sixth outcome, amending ST67, and the recorder writes the redacted text in `message`; (3) keep the text outside the records, in a separate file of the run's output directory, also a format decision.

**FU24, rerun once.** The service answered HTTP 401 again, with no stream (`status=401 finish="" out=-1 reasoning=-1`). Not retried. No measurement, so 0089-MADR:972's premise is unchanged; FU24 stays open, waiting on a Kilo key the service accepts. Whether the shell this run used saw the updated key cannot be checked here without reading it, which the rules forbid.

Counts after this amendment: 35 rows, 35 decided; FU21 (ii) part (2) not built (stopped), FU24 open on the owner's key.

## Amendment (2026-11-27, fourth): FU21 (ii) part (2) and FU24

Additions only. The owner's rulings, relayed by the coordinator, quoted exactly:

| Row | Ruling, quoted | Outcome |
| :--- | :--- | :--- |
| FU21 (ii), part (2) | "Leave it, recorded (Recommended)" | a provider-build failure's text is not kept: a stated limit of the 0084 recorder, whose outcomes stay ST67's five; the runner's move, part (1), stands |
| FU24 | "I'll run it myself" | the owner runs `TestLive_KiloControl_0095` in this session, with their own environment's `KILO_API_KEY`; its one result line, `0095 FU24 RESULT ...`, is recorded here when the coordinator relays it. The line now also gives `limit_accepted` and `stream_code`, and the run ends with `0095 FU24 OK: ...` or `0095 FU24 FAILED: ...` |

Counts after this amendment: 35 rows, 35 decided; FU24 open, waiting on the owner's run.

## Amendment (2026-11-27, fifth): FU24 measured, and closed

Additions only. On the owner's "I'll run it myself", the coordinator ran `TestLive_KiloControl_0095` once in this session, with the owner's new Kilo key passed from the owner's key file into the test's environment; the key was never printed. Allowlisted output, as relayed, verbatim:

```text
0095 FU24 RESULT model=z-ai/glm-5.3-flashx sent=131072 status=200 limit_accepted=true finish="length" out=131072 reasoning=836 stream_error="" stream_code=0
0095 FU24 OK: a "length" finish at 131072 output tokens with 131072 sent
--- PASS: TestLive_KiloControl_0095 (403.31s)
```

Before it, with the old key, the SDK's own `kilo` provider also got an HTTP 401 classified as an auth failure (the coordinator's run): the earlier 401s of FU24 were the key refused, not the shape of the control run's request.

**What it means**, by the test's own doc and 0089-MADR:972: a `length` finish at 131,072 with 131,072 sent puts the earlier cut, at 128,000 with no limit sent (DL65's measurement), in what Kilo fills when no limit is sent, not in the model: the model produced the whole 131,072 when asked for it.

**Checked against the decided rows.** DL65's outcome, "Kilo keeps sending the discovered limit", was decided to hold "on every reading" (0089-MADR's review note on the cause), and 0089-MADR:972 says "Either way the interim stands, so the run decides only whether the sentences may name the cause". The measurement confirms DL65's premise rather than changing it: with no limit sent, Kilo hands out less than the stated limit, so sending it is right. No row changes. The docs that state the measurement alone ("measured 2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated") stay true; that they may now also name the cause is recorded, and left as they are.

FU24 is decided and done. Counts after this amendment: 35 rows, 35 decided, 0 open; 1 not found.
