---
status: accepted
date: 2026-11-25
decision-makers: repository owner
consulted: "the owner's request of 2026-11-24, relayed by the coordinator, 'get agents working on 2 madrs, one that includes SDK H1'; the owner's answers of 2026-11-24 to OQ1 to OQ6, relayed by the coordinator, their labels quoted in the amendment; the upstream synthesis appended to 0091-REPORT (2026-11-24), idea H1, with the coordinator's check of `responses.go:833-834`; an independent review of the first draft (2026-11-24, not ready, relayed by the coordinator); this repository at 2f3c3a9, read only, and a scratch clone of it at the same commit for the measurements below (no file of the tree changed); gobble-cli at 97fa1f6, read only; the read-only upstream checkouts opencode at 4ce137b39e, goose at 36b29113a, pi at 1402973a9 and codex at 57e64682f8"
informed: the callers of `openai`, `grok` and OpenCode's responses route, whose decoder this record changes; `chatcompat` callers who opt in with `WithResponses()` (0091-MADR ST50), the path that reaches servers no record measured; gobble-cli, whose turn ends when a reply fails after it showed text
---

<!-- markdownlint-disable MD013 MD024 -->

# Fill a Completed Responses Stream From Its Terminal Output

Associated PLAN: [0092-PLAN-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-PLAN-fill-a-completed-responses-stream-from-its-terminal-output.md)

*(2026-11-24, the owner's answers: OQ1 to OQ6 answered, each with its recommended option; every open row is decided (ST6 (f), ST7, ST11, ST12, ST20, ST27), 0 rows open, and the record is `accepted`. See "Amendment (2026-11-24): the owner's answers to OQ1 to OQ6" at the end. The rows above keep their text.)*

*(2026-11-24, revision 1, before any answer: after the independent review. OQ1's costs are corrected (M3 succeeds today; a server fault made non-retryable), and its options now include a retryable failure, recommended. D3's "preview" claim is dropped for `Stream`'s own words. ST19 cites 0021-MADR D1 and 0012-MADR §4.1. D2's index pass requires one item type. New rows decide terminal-only items `appendOutput` drops (ST28), which types OQ1 tracks, the per-event limit (ST29), `Collect`'s bound (ST30), usage (ST31) and deltas (ST32); a `.done` that delivered less than the terminal moves into OQ2. ST6's last plant gets a case that can catch it; ST9 asserts the order outright; every row conditional on 0088 names the ruling it assumes and the test that flips. Two choices the first draft made are the owner's: ambiguous pairing (OQ5) and a `.done` that differs from the terminal (OQ6). ST14 no longer rules on 0091's check: that part goes to a 0091 amendment. ST26 documents OQ1's error. E19 cited the synthesized stream's contract and now cites the native one.)* *(2026-11-24, revision 2, before any answer: after a delta check. OQ1 (1)'s cost no longer promises a retry where none happens: after an event nothing is resent, so (1) and (2) differ mostly in the error's kind, and the recommendation is re-weighed on that. OQ5's case is defined after passes 1 and 2, as D2's pass 3 now says, with its cost on the cut path. OQ6's claim about opencode and pi cites their code. ST28's plant names one production change.)*

*(2026-11-25, placed in the tree at `138d52f` (v1.33.0) and built next. The owner's rulings of 2026-11-24, relayed by the coordinator, quoted as given: OQ1 "Fail as retryable stream (Recommended)", OQ2 "Use summary when done was empty (Recommended)", OQ5 "Take nothing of that kind (Recommended)", OQ6 "The item's own message (Recommended)", OQ3 "Debug line via WithLogger (Recommended)", OQ4 "Own v1.33.0 phase (Recommended)", and, for the PLAN, "Approve; follow the queue (Recommended)". The owner's "Yes, re-scope v1.33.0 (Recommended)" then moved this record from v1.33.0 to v1.34.0 ([0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md), seventh amendment, its 0092 note): where this record says v1.33.0, read v1.34.0. The owner's build ruling of 2026-11-25, relayed by the coordinator, quoted exactly: "SDK: build 0092 (Recommended)". No row's decision changes; the evidence index is re-grounded at `138d52f`, and one question about the build order is put to the owner, not resolved. See "Amendment (2026-11-25): placed for v1.34.0, built next, re-grounded at `138d52f`" at the end.)* *(2026-11-25, second: Q-B1 answered, "Rule R1–R3 first". 0088's R1 to R3 are ruled, as 0088-MADR ST107 to ST109. ST10, ST11 and ST13 now rest on those decided rows, and no outcome here changes. The order is 0088-PLAN Phase 6 first, then this record's PLAN Phase 1. See "Amendment (2026-11-25, second)".)*

## Context and Problem Statement

The upstream synthesis appended to [0091-REPORT-responses-api-across-providers.md](../reports/0091-REPORT-responses-api-across-providers.md) ranks H1 first among the SDK's hardening ideas: "Fill a `completed` Responses stream from its terminal `output`". The owner asked for a record that includes it. This record decides what the Responses stream reader does with the items a terminal `response.completed` carries that no `response.output_item.done` delivered, and every case next to that one.

### What the reader does today (read at 2f3c3a9)

* The stream builds its answer from `response.output_item.done` events (E7, E20). Text deltas are yielded as they arrive and are not kept (E8). Each `.done` item goes through `appendOutput`, which keeps a message with text, a call, a client tool search when the request has a Search tool, and a reasoning item with text or an encrypted part, and drops everything else: an empty message, empty reasoning, a hosted search, and any other type such as `custom_tool_call` (E75, E76, E87).
* At the last event, `end` takes the terminal response's own items only on a cut (E1, E2): `takeTerminal`, added by 0088-MADR ST104 for xAI's cut stream that sends no `.done` events (E31). On `response.completed` it calls `completed` (E3), which fails an answer with no output as `ErrEmptyAnswer` (E4). The id, model and usage come from the terminal event on both paths (E78).
* A terminal item counts as delivered when an `output_item.done` carried its id, or, when the terminal item has no id, by its place among the delivered items (E5, E6). Each delivered item's id is recorded, an empty one included (E28).
* The reader tracks the items a stream announced (`output_item.added`) only for calls (E9); at a cut, each call announced and never finished is named as the cut call (0088 R2 as built, E81).
* The plain decode reads the reply's `output` array whole (E12); it has no `.done` events and no gap.
* The ChatGPT backend's recorded streams send every `.done` event and a terminal `output` that is empty (E14, E15, E16, E17). xAI's completed streams send `output_item.done` (E18); its cut streams do not (E31).

### What that does on a server that omits `.done` (measured 2026-11-24 on a scratch clone at 2f3c3a9)

A scratch test (`<scratch>/m0092/sdk/llmprovider/internal/wire/responses/measure_0092_test.go`, not committed) fed hand-written streams to `Events` and `ReadStream`. Output quoted from `<scratch>/m0092/measure-responses.txt`.

| Id | Stream | Today |
| :--- | :--- | :--- |
| M1 | text deltas, no `.done`, the message only in the terminal `output` | `Events=["text_delta:Hello"] err=llmprovider: empty answer: stop: responses: the answer has no content`; `ReadStream` the same error |
| M2 | the message's `.done`, a call only in the terminal `output` | `done:finish=stop:output=[{Role:assistant Text:Hello}]`: the call dropped, no warning |
| M3 | the message's `.done`, then a call announced (`added`), never done, absent from an empty terminal `output` | succeeds: `done:finish=stop`, the call dropped, no warning |
| M4 | an encrypted reasoning item only in the terminal `output` | dropped; `output=[{Role:assistant Text:Hello}]` |
| M5 | a cut (`response.incomplete`, `max_output_tokens`) whose message `.done` had no id, the terminal's message an id | **duplicated**: `item:...Text:Hello`, `item:...Text:Hello`, `done:finish=length:output=[{... Hello} {... Hello}]` |
| M5b | M5's items on `response.completed` | one message (the terminal is not read) |
| M7 | a reasoning `.done` with no summary and no `encrypted_content`, the terminal's same item with it (pi's Azure case, E27) | the reasoning item dropped |
| M8 | the terminal's message text differs from its `.done`, same id | the `.done` text kept |
| M6 | the plain decode of a completed reply holding a message and a call | both returned, `FinishReason:tool_calls` |

So a non-compliant server costs today a hard failure after the caller showed text (M1), a tool call dropped while the reply reports `stop` (M2, M3), and encrypted reasoning that is never replayed (M4, M7). On the cut path, ST104's matching duplicates text when the two sides disagree on ids (M5).

### Who reaches this reader

`openai` (`Events`, and `ReadStream` for the ChatGPT backend's plain call, E21, E22), `grok` (E23), OpenCode's responses route (E24), and, from 0091-PLAN Phase 2, `chatcompat` with `WithResponses()` (0091-MADR ST52), which sends to any server the caller names. No service the SDK reaches today was measured omitting `.done` on a completed stream (E14 to E18); the case is the "non-compliant proxy" qwen-code guards against for the converse gap (E38) and the "compatible providers" opencode recovers from (E34).

### What the upstream agents do (read-only checkouts)

| Agent | On `response.completed` | Where |
| :--- | :--- | :--- |
| opencode | recovers calls it saw announced from the terminal `output`, "Some compatible providers omit output_item.done even after completing the response"; an item with no id takes the id open at its output index | E34, E35, E36, E37 |
| goose | the terminal `output`, when not empty, replaces the done items | E39, E40 |
| pi | builds from events; an orphan `.done` gets a slot; backfills `encrypted_content` from the terminal for Azure | E25, E26, E27 |
| codex | reads no terminal `output` (its `ResponseCompleted` has none); requires the terminal event | E29, E30 |

### Other wires (read at 2f3c3a9; M9 to M11 measured on the scratch clone)

* **Messages**: `message_stop` carries no content (E41), and a block the stream never stopped is folded into the answer there (E42). No terminal snapshot exists to fill from. M9: a text block with no `content_block_stop`, on `end_turn`, ends `["text_delta:Hello" "done:stop:[{Role:assistant Text:Hello}]"]`: the text is in the response, but no `EventItem` was yielded for it, where the architecture says each wire's decoder "yields each item as `EventItem`" (E19). A different gap, recorded as FU1.
* **Chat Completions**: `[DONE]` carries no content; the items are built from deltas at `[DONE]` (E43, E44). No gap.
* **generateContent**: each chunk's parts are added before the chunk carrying `finishReason` finishes the answer (E45, E46). No gap.
* **Interactions**: `interaction.completed` is read for its status, usage, id and model; its `steps` are not (E47, E48). M10: a step only in the terminal interaction ends `err=llmprovider: empty answer: gemini interaction`; M11: a text delta with no `step.start` is neither yielded nor kept (E49), the same error. The shape is H1's, but the wire reaches only Google's service, whose terminal interaction was not measured carrying steps. Recorded as FU2.

## Decision Drivers

* **One answer, whatever the server omits**: what `EventDone` carries is what the wire's `Decode` returns for the same answer (E50), and the plain decode already reads the terminal `output` (M6).
* **No silent loss**: a call or a reasoning item the server sent is not dropped without a word (the synthesis's "What not to copy: silent drops").
* **No duplicates, and no change for compliant servers**: an item `.done` delivered is not added again; the ChatGPT goldens and xAI's shapes decode as today.
* **The stream contract**: `EventItem`s come in `Response.Output`'s order (E51), only `EventDone` carries a `Response` (E64), and what was yielded is never taken back.
* **The smallest change**: no exported name under the recommended answers (R55); no per-delta cost (0028-MADR's one-pass decode).
* **The reach that motivates it**: `chatcompat`'s opt-in sends to servers nobody measured (0091-MADR ST52).

## Considered Options

* A. Take the terminal's undelivered items on `response.completed` as on a cut
* B. Rebuild the answer from its deltas
* C. Let the terminal `output` replace the streamed items
* D. Leave it, and document it

A extends ST104's take to the completed path, with a matching rule that does not duplicate an item that has an id or an index. B accumulates text and argument deltas, as pi and opencode build their answers. C is goose's. D is codex's.

## Decision Outcome

Chosen option (proposed): "A. Take the terminal's undelivered items on `response.completed` as on a cut", because it fixes M1, M2 and M4 with the code ST104 already runs on a cut, keeps every compliant stream unchanged, adds no per-delta cost and no export, and gives the stream and the plain decode one answer. Six questions are open (OQ1 to OQ6); the record stays `proposed` until the owner answers. *(2026-11-24: the owner answered all six with the recommended options; option A is chosen, and the record is `accepted`. See the amendment.)*

### D1. The take

At the stream's terminal event, `response.completed` or `response.incomplete` for the output limit, the reader adds each item the terminal response carries that no `output_item.done` delivered (D2), before any decision on how to finish: `completed` for a completed stream, `CutAnswer` for a cut, and whatever check 0091-PLAN Phase 2 adds between them. Taken items go through the same `appendOutput` a `.done` item goes through, so each is kept or dropped as its `.done` would be (ST28), and a refusal, a client tool search and Grok's dispatch wrapper decode as they do from a `.done`. Taken items are added after the delivered ones, in the terminal's order, and yielded as `EventItem`s after the held calls and before `EventDone`, with no delta of their own (ST32).

### D2. What counts as delivered

The `.done` events and the terminal's items are paired in three passes, each over what the earlier ones left unpaired:

1. by id, where both carry one;
2. by index: a `.done` event whose `output_index` is the item's place in the terminal `output`, when both are of one item type;
3. in order within each item type, the k-th unpaired `.done` message with the k-th unpaired terminal message, and so for each type; except OQ5's case: after passes 1 and 2, k ≥ 1 unpaired `.done`s of a type and more than k unpaired terminal items of that type, which OQ5 decides.

A terminal item left unpaired is taken. Every recorded `.done` event carries `output_index` (E15); pairing by it is opencode's (E37). The rule replaces ST104's on the cut path too, so M5 no longer duplicates: a dated annotation on 0088-MADR ST104 points here. A `.done` that delivered less than its terminal partner is OQ2's.

### D3. An item `.done` delivered, and one it did not

For an item no `.done` delivered, the terminal's item is the answer. No delta text is kept or compared: `Stream`'s contract is that only `EventDone` carries a `Response` (E64), and today a `.done` whose text differs from its deltas already yields the `.done`'s text in the response the same way. For an item `.done` delivered whose terminal copy differs (M8), which wins is OQ6.

### Consequences

* Good, because a non-compliant server's completed stream returns the answer its plain reply would (M1, M2, M4), and `chatcompat`'s opt-in is not exposed to it.
* Good, because the cut path stops duplicating an item whose ids disagree (M5).
* Good, because gobble-cli's turn no longer ends with an error after it showed text on such a stream (E53); fewer of gobble's G1 cases arise. gobble needs no change.
* Neutral, because no service the SDK reaches today was measured omitting `.done` on a completed stream, so no provider's measured behaviour changes.
* Bad, because a stream whose items arrive partly by `.done` and partly from the terminal returns them in arrival order, then terminal order, which may differ from the server's order (ST9).
* Bad, because one more field, `output_index`, is decoded per `.done` event (not per delta).
* Bad, under OQ1 (1) or (2), because M3, a success today, becomes an error after the caller showed text.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes behaviour, each plant caught on a scratch clone (`tools/plant.py`).
* `python3 -B tools/apidiff.py` against the last tag lists no name from this record under the recommended answers.
* `tools/claims.py` passes this record's evidence index; its planted claims are caught.
* The ChatGPT goldens (`TestEvents_Goldens`) and 0088's xAI shapes (`TestCutAnswer_XAIShapes_0088`) pass unchanged.

## Owner Questions

Each stands alone, three options, the recommended one first.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ1. A server says it finished, but an item it announced (a message or a tool call) never arrived, neither on its own nor in the server's final summary. Today the item is lost without a word: a reply with text and a lost tool call reports success, `stop`. What should the SDK do? | (1) **Fail it as a broken stream that may be retried**, as a stream that stops before its last event is failed today. (2) **Fail it as a reply that could not be read whole, not retried.** (3) **As today**: drop the item silently. | (1) the error is `ErrProviderUnavailable`, marked as after the reply (E68, E69): a server fault, said as one. A resend happens only where nothing was yet yielded (a reply whose only item is the lost call) or on the ChatGPT backend's plain call under `WithRetry` (E22), at most once (E94), and may be billed again. After any text or item was shown nothing is resent: `WithStreamRetry` stops retrying at the first event (E91), and a `Generate` read over the stream marks a later failure as after an event (E92), which is never resent (E93). So M3, a success with `stop` today, becomes an error after its text was shown, and in gobble that ends the turn with the error (E53). (2) the error is `ErrIncomplete` naming the item, which matches `ErrInvalidRequest` and is never retried (E54, E65), though the fault is the server's and may pass; after shown text it behaves as (1); with nothing shown it is not resent where (1) would be; `ErrIncomplete`'s doc gains the case. (1) and (2) differ mostly in the error's kind; (1) is recommended because the kind names the server's fault, and its one resend falls only where nothing was shown. (3) a tool call the model asked for is lost and the reply says `stop`; the caller is never told. Under (1) and (2) the reader tracks announced messages, calls, and client tool searches when the request has a Search tool: the types `appendOutput` keeps (E75, E76). It does not track reasoning items, whose loss only affects the next turn's replay, or hosted and unknown types, which `appendOutput` drops anyway. | ST11 |
| OQ2. The server's per-item message delivered less than its final summary holds for the same item: an empty message whose text is only in the summary, or a reasoning item whose encrypted part (needed to send it back next turn) is only in the summary. pi records Azure OpenAI doing the second (E27). Today the fuller copy is ignored (M7). What should the SDK do? | (1) **Use the summary's copy when the per-item message added nothing**: an empty message, or a reasoning item with neither text nor an encrypted part, is treated as not delivered, and the summary's copy is taken. (2) **Ignore the summary's copy, as today.** (3) **Also fill in a kept item**: as (1), and a reasoning item that was kept with text but no encrypted part gets the summary's encrypted part in the final response. | (1) nothing yielded earlier changes, since the empty item was never yielded; the taken item lands after the items already delivered (ST9), so a reasoning item can follow the message, an order no service was measured taking back on replay. (2) a server that does it loses the reasoning on the next turn, and an empty message's text is lost. (3) as (1), plus: the reasoning item yielded earlier lacks the encrypted part while the final response's copy has it, so the two differ, against the stream's contract (E51). | ST12 |
| OQ3. When the SDK fills items from the server's final summary, should anything report it? | (1) **A debug line through `WithLogger`**: one line per stream that took items, with counts by type and the provider, no content. (2) **A new warning kind** on the response. (3) **Nothing.** | (1) an internal `Logger` field on `responses.Decoder`, set by `openai`, `grok`, OpenCode and, in 0091 Phase 2, `chatcompat` (E56, E57); silent for a caller with no logger (E58); no export. (2) one export with no outside call site (R55); a warning on an answer given as asked, where a Warning means "answered, but not exactly as asked" (E59); gobble could show it. (3) a non-compliant server goes unnoticed, and gobble's planned stream-health record has nothing to report. | ST20 |
| OQ4. When should this be built? | (1) **Its own phase in v1.33.0, after 0088's Phase 3 review and before 0091-PLAN Phase 2.** (2) **Inside 0091-PLAN Phase 2.** (3) **After v1.33.0.** | (1) one more phase and commit, gated like the others; v1.33.0's tag waits for it; 0091 Phase 2 builds on the reader this changes, so its tests see the take. (2) 0091's PLAN is amended to carry another record's rows; one larger phase and review. (3) `chatcompat`'s opt-in ships without it: on a server that omits the per-item messages it fails after showing text (M1) or drops calls (M2, M3). | ST27 |
| OQ5. Some per-item messages carry neither an id nor a position the SDK can match, so it cannot always tell which final-summary item one was about. The case: after matching by id and by position, k (one or more) unmatched per-item messages of one kind remain, and more than k unmatched summary items of that kind. What should the SDK do? | (1) **Take nothing of that kind from the summary.** (2) **Pair them in order**, the first unmatched message with the first unmatched summary item, and take the rest. (3) **Take every unmatched summary item of that kind.** | (1) never a duplicate; on a completed stream a summary-only item of that kind may be lost, as every one is today; on a cut stream, where 0088's take pairs by position today (E6), an item that rule would take is no longer taken. (2) right when the missing messages are the last ones; when an earlier one is missing, one item comes twice and another is lost; on a cut stream it is today's position rule, kept to one kind. (3) each of the k items whose message came comes twice. No recorded server sends such messages: every one carries a position (E15). | ST6 |
| OQ6. A per-item message and the final summary disagree about the same item (same id, different text) (M8). Which should the SDK keep? | (1) **The per-item message**, as today. (2) **The summary**, in the final response only. (3) **The summary, holding every item back until the end.** | (1) the item already yielded and the final response agree. opencode and pi take a call's finished arguments over the deltas before them (E88, E89), opencode recovers from the summary only calls still pending (E36), and pi fills an encrypted part from the summary only where the item had none (E90); goose does the opposite (E39). (2) the item yielded earlier and the final response differ, against the stream's contract (E51). (3) as goose: nothing is yielded before the end on any stream, compliant ones included, which undoes 0031-MADR D2's items-as-they-end for every Responses caller. | ST7 |

## State Table

"Decided" means decided by this record's outcome, by an existing record, or by code no option changes. "Conditional on 0088" names the ruling of 0088-PLAN Phase 3 it assumes, as built at 2f3c3a9, and the test that flips if that ruling changes. Tests are the PLAN's to come; each names its plant.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | A completed stream whose message only the terminal `output` carries (M1) | decided | taken: `EventItem` with the message, then `EventDone`, finishing `stop`, from `Events` and `ReadStream` alike; a refusal part finishes `content_filter` | D1 | `TestTerminalFill_Text_0092` (M1's shape, both readers, and a refusal case); plant: the take guarded by `cut` again, so M1 is `ErrEmptyAnswer` |
| ST2 | A completed stream whose call only the terminal carries (M2), including a client `tool_search_call` with a Search tool and Grok's dispatch wrapper | decided | taken through `appendOutput`, finishing `tool_calls`; the search call and the wrapper decode as from a `.done` | D1 | `TestTerminalFill_Call_0092`, three cases; plant: calls skipped by the take, so M2 finishes `stop` |
| ST3 | A completed stream whose reasoning item only the terminal carries (M4) | decided | taken as a `ReasoningItem` with its summary text and encrypted part, `Format` Responses, so it is replayed (0021-MADR W7) | D1 | `TestTerminalFill_Reasoning_0092`, the next request's input carrying the item; plant: reasoning skipped by the take |
| ST4 | A completed stream with no `.done` at all, a reasoning item, a message and a call | decided | all three taken in the terminal's order, yielded in that order, before `EventDone` | D1 | `TestTerminalFill_All_0092`; plant: taken items yielded in reverse |
| ST5 | A compliant stream: every item by `.done`, the terminal's `output` the same items (OpenAI) or empty (the ChatGPT backend, E14) | decided | nothing taken; no duplicate; the goldens decode as today | D2 | `TestEvents_Goldens` unchanged, and `TestTerminalFill_NoDuplicate_0092` with a full terminal `output`; plant: nothing ever paired, so each item comes twice |
| ST6 | Pairing: (a) a `.done` with no id and the terminal's item with one (M5); (b) a `.done` skipped for item 0 and given, with no id, for item 1; (c) as (b) with no `output_index`, item 0 a reasoning item, item 1 a message; (d) terminal `[message A with no id, message B with id b]`, a `.done` for B by id, no `output_index`; (e) a `.done` message with no id whose `output_index` is 0, where the terminal's item 0 is a reasoning item and item 1 the message; (f) the ambiguous case OQ5 names | decided, except (f) *(2026-11-24: (f) decided by OQ5's answer; restated in the amendment)* | (a) one message, completed and cut; (b) item 0 taken, item 1 not; (c) the message paired, the reasoning item taken; (d) B paired by id, A taken; (e) the index pass does not pair across types, the message pairs in order, the reasoning item is taken; (f) OQ5 | D2; opencode (E37); OQ5 | `TestTerminalFill_Match_0092`, cases (a) to (e), completed and cut; plants: today's rule ((a) duplicates); id-only pairing ((b) duplicates item 1); order pairing across types ((c) drops the reasoning item); order pairing over every `.done`, not only the unpaired ones ((d) pairs the B `.done` with A, so A is lost); index pairing across types ((e) drops the reasoning item) |
| ST7 | The terminal's item differs from the `.done` item of the same id (M8) | open *(2026-11-24: decided by OQ6's answer; restated in the amendment)* | recommended: OQ6 (1), the `.done` item stands | OQ6 | |
| ST8 | The deltas yielded differ from the terminal's item, for an item no `.done` delivered | decided | the terminal's item is the response's; deltas are not kept or compared, as for a `.done` today | D3; E64 | `TestTerminalFill_Deltas_0092`, deltas `Hel`, terminal `Hello`, response `Hello`; plant: the text rebuilt from the deltas |
| ST9 | Items arrive partly by `.done`, partly from the terminal | decided | `Response.Output` is the delivered items in arrival order followed by the taken items in the terminal's order; the `EventItem`s come in that order | D1; E51 | `TestTerminalFill_Order_0092`: `Response.Output` equals delivered ++ taken, and the `EventItem`s equal `Response.Output`; plant: taken items inserted at their terminal index in `Response.Output` |
| ST10 | Held calls on a completed stream that also takes items | decided, conditional on 0088 R3 *(2026-11-25: rests on 0088-MADR ST109, R3 as built, decided)* | assumes R3 as built: a call whose arguments are `{}` or not a whole object is held until a later item or the end (E83), and released by `HeldCalls.Done` (E10, E13). Held calls are released first, then taken items (E11); a taken call is not held, being yielded at the end; a taken `{}` or non-object call on a completed reply is kept, as a delivered one is. If R3 is withdrawn, the held call is yielded at its `.done` and only the release step changes | D1; 0088 R3 | `TestTerminalFill_Held_0092`: a held `{}` call, then a taken call; it and 0088's `TestCutAnswer_HeldCalls_0088` (E86) flip if R3 changes; plant: taken items yielded before held calls |
| ST11 | An announced item delivered neither by `.done` nor by the terminal, on a completed stream (M3) | open, conditional on 0088 R2 *(2026-11-24: decided by OQ1's answer; restated in the amendment)* | recommended: OQ1 (1). Assumes R2 as built, which names such a call as the cut call on a cut (E81); OQ1 decides the completed path only. If R2 changes, the cut path's counterpart changes, and `TestCutAnswer_ResponsesCutItem_0088`'s stream case (E85) flips | OQ1; 0088 R2 | |
| ST12 | A `.done` that delivered less than its terminal partner (an empty message; reasoning without its encrypted part, M7) | open *(2026-11-24: decided by OQ2's answer; restated in the amendment)* | recommended: OQ2 (1) | OQ2 | |
| ST13 | A cut stream (`response.incomplete`, `max_output_tokens`) | decided, conditional on 0088 R1 and R2 *(2026-11-25: rests on 0088-MADR ST107, R1 as built on Responses, and ST108, R2 as built, both decided)* | as ST104, with D2's pairing. Assumes R1 as built (a call whose arguments are not a whole object, or the last call with `{}`, is the cut call, E80) and R2 as built (E81); taken calls meet those rules as delivered ones do. If either ruling changes, `TestCutAnswer_XAIShapes_0088`'s terminal-call case (E84) and `TestTerminalFill_Match_0092`'s cut cases flip | D2; 0088 ST104, R1, R2 | `TestCutAnswer_XAIShapes_0088` unchanged; `TestTerminalFill_Match_0092`'s cut cases; plants: as ST6 |
| ST14 | The order of the take against any finish decision, 0091's ST55 check included | decided | the take runs before the reply is finished, so whatever decides how it finishes sees the taken items. How 0091's ST55 check treats an announced call never finished is not ruled here: ST26 routes it to a 0091-MADR amendment | D1 | `TestTerminalFill_Order_0092`'s cut case: a taken call meets the cut rules; plant: the take moved after `CutAnswer` |
| ST15 | `chatcompat` with `WithResponses()` (0091 ST52) | decided | reached through the shared reader; no `chatcompat` code | D1; 0091 ST52 | at decoder level by ST1 and ST2; at provider level by a case 0091-PLAN Phase 2's `TestChatcompat_ResponsesDecode_0091` gains (M1's and M2's shapes), through the 0091-PLAN amendment ST26 names; plant: as ST1 |
| ST16 | The plain decode (M6) | decided | unchanged: it reads `output` whole (E12) | E12 | the existing decode tests, plus M6's shape in `TestTerminalFill_Plain_0092`; no plant: no code changes |
| ST17 | `ReadStream`: the ChatGPT backend's plain call, and `Generate` read over the stream (0088 ST88) | decided | as `Events`: one reader (E20) | E20, E22 | ST1's test runs both readers; plant: as ST1 |
| ST18 | `response.failed`, an `error` event, `response.incomplete` for another reason | decided | unchanged: an error; nothing taken | 0012-MADR §1.1, §1.5; 0020-MADR F39 | existing stream error tests; no plant: no code changes |
| ST19 | A stream that ends before its terminal event | decided | unchanged: `ErrProviderUnavailable`, after the reply, asked again at most once (E68, E69); no terminal, nothing taken | 0021-MADR D1 (E66); 0012-MADR §4.1 (E67) | `TestReadStream_EndWithoutCompletedIsAfterReply` unchanged; no plant: no code changes |
| ST20 | A report that items were taken | open *(2026-11-24: decided by OQ3's answer; restated in the amendment)* | recommended: OQ3 (1) | OQ3 | |
| ST21 | Messages | out of scope | no terminal snapshot (E41); the unstopped block's missing `EventItem` (M9) is FU1 | E41, E42 | none: out of scope |
| ST22 | Chat Completions | out of scope | no terminal snapshot; items built from deltas at `[DONE]` (E43, E44) | E43, E44 | none: out of scope |
| ST23 | generateContent | out of scope | no terminal snapshot; parts added as they arrive (E45, E46) | E45, E46 | none: out of scope |
| ST24 | Interactions | out of scope | H1's shape (M10, M11), on a wire that reaches only Google's service, unmeasured there; FU2 | E47, E48, E49 | none: out of scope |
| ST25 | The exported API | decided | nothing new under OQ1 (1) or (2) and OQ3 (1); OQ3 (2) would add one `WarningKind` | R55 | `make api-check`, `tools/apidiff.py`; plant: an export added on a scratch clone |
| ST26 | Docs and records | decided | `responses.go`'s docs of `ReadStream`, `Events`, `takeTerminal`, `end` and `streamEvent.Output`; `docs/architecture.md`'s stream paragraph (E50); a **Fixed** entry under `[Unreleased]`; OQ1's error documented where its answer puts it: under (1) in `Events`' and `ReadStream`'s docs and the changelog, under (2) also in `ErrIncomplete`'s doc (E54); dated annotations on 0088-MADR ST104 (D2) and on 0091-MADR's fourth amendment, whose W7 note says a completed stream does not take (E62); a 0091-MADR amendment for how ST55's check treats an announced call never finished, and a 0091-PLAN amendment adding ST15's case to Phase 2 | the records rule; R43 | `make docs-check changelog-check records-check`, a claims row per changed doc line; plant: the `takeTerminal` doc left saying "on a cut" |
| ST27 | The release and the build order | open *(2026-11-24: decided by OQ4's answer; restated in the amendment)* | recommended: OQ4 (1) | OQ4 | |
| ST28 | A terminal-only item `appendOutput` drops: an empty message, an empty reasoning item, a hosted `tool_search_call`, a client one with no Search tool, a `custom_tool_call` or any other type | decided | dropped, as its `.done` would be (E75, E76, E87); not counted as a loss, and not tracked by OQ1 | D1 | `TestTerminalFill_Dropped_0092`, one case per kind, each finishing as if the item were absent; plant: the take maps an item of an unknown type to a `FunctionCallItem`, so the `custom_tool_call` case returns a call |
| ST29 | A terminal event over `EventLimit` (16 MiB, E71) | decided | unchanged: `ErrIncomplete` "stream event over 16 MiB" (E70); nothing taken | 0021-MADR W2 | `TestReadStream_OverlongEventIsIncomplete` unchanged; no plant: no code changes |
| ST30 | `Collect`'s bound on a `Generate` read over the stream (0088 ST88b) | decided | unchanged: `Collect` counts delta text and the arguments of yielded calls (E73, E74) against `ReplyLimit` (E72); a taken message's text is not counted again, being bounded by its terminal event's `EventLimit`, the same 16 MiB | 0088 ST88b | `TestTerminalFill_Collect_0092`: 10 MiB of text deltas, then a terminal message of the same 10 MiB, returns the answer; plant: `Collect` counting a message item's text, so the answer is `ErrIncomplete` |
| ST31 | Usage, id and model | decided | as today: from the terminal event on both paths (E78) | E78 | `TestTerminalFill_Text_0092` asserts M1's usage (3 in, 5 out) and `TestReadStream_Usage` unchanged; plant: usage read only on a cut |
| ST32 | Deltas | decided | as today: text and reasoning-summary deltas yielded as they arrive (E8, E79); a taken item gets no delta of its own | D1 | `TestTerminalFill_All_0092` asserts no delta after the first taken `EventItem`; plant: a text delta yielded for a taken message |

Counts: 32 rows: 23 decided (ST6's case (f) open under OQ5), 4 out of scope, 5 open (ST7, ST11, ST12, ST20, ST27). Rows conditional on 0088's Phase 3 ruling: ST10 (R3), ST11 (R2), ST13 (R1, R2).

## Pros and Cons of the Options

### A. Take the terminal's undelivered items on `response.completed` as on a cut

* Good, because it is ST104's code on one more path, and the plain decode's answer on the stream.
* Good, because no per-delta work is added, and compliant streams take nothing.
* Bad, because a mixed stream's order is arrival, then terminal (ST9).

### B. Rebuild the answer from its deltas

* Good, because it also covers a server that sends neither `.done` nor a terminal `output`.
* Bad, because the reader would decode and keep every argument delta it skips today (0031-MADR Q4), and an argument rebuilt from deltas is unvalidated.
* Bad, because it is a second source for the same item, and either can be wrong.

### C. Let the terminal `output` replace the streamed items

* Good, because it is goose's simple rule.
* Bad, because items already yielded as `EventItem`s would differ from `EventDone`'s, against the contract (E51), unless every item is held to the end (OQ6 (3)).
* Bad, because the ChatGPT backend's empty terminal `output` (E14) shows the terminal is not always whole; goose replaces only when it is not empty.

### D. Leave it, and document it

* Good, because nothing changes.
* Bad, because M1 to M4 stay: a failure after shown text, a call dropped under `stop`, reasoning lost; and M5 duplicates on the cut path.

## More Information

### Follow-ups (recorded, not built)

* **FU1.** Messages: a block the stream never stopped is in `EventDone`'s response but never yielded as `EventItem` (M9; E42, E19).
* **FU2.** Interactions: whether Google's terminal interaction carries `steps`, and a text delta with no `step.start` (M10, M11; E47 to E49). Measure first.

### Relation to other records

* **0088-MADR** ST104 (the take on a cut) is extended to the completed path; its matching is replaced by D2 (dated annotation, ST26). R1 to R3 stay with 0088-PLAN Phase 3's review (E60, E32, E61); ST10, ST11 and ST13 name what they assume.
* **0091-MADR** ST52 and ST55: `chatcompat`'s opt-in reads through this reader (ST15); the take precedes ST55's check (ST14, E33), and how that check treats an announced call never finished goes to a 0091 amendment (ST26). 0091-PLAN Phase 2 starts after 0088's Phase 3 review (E63); OQ4 places this record's phase.
* **0031-MADR** D2 and D4: the stream's items and its `EventDone`; what was yielded before an error is not a response (E64).
* **gobble-cli**: no dependency either way. H1 removes one way a reply fails after it showed text (E53); gobble's G1 record (its 0070, drafted alongside) handles the rest. Under OQ1 (1) or (2), M3 becomes such a failure. gobble reruns an empty answer only for `ErrEmptyAnswer` (E55), which no OQ1 option returns.

### What could not be established

* Whether any server the SDK reaches omits `.done` on a completed stream: none of the recorded or measured ones does (E14 to E18).
* Whether Azure OpenAI still sends `encrypted_content` only in the terminal (pi's comment, E27; OQ2).
* Whether a service takes back a reasoning item placed after the message on replay (OQ2 (1) and (3)).

### Evidence index

<!-- claims: self=2f3c3a9 opencode=4ce137b39e goose=36b29113a pi=1402973a9 codex=57e64682f8 gobble-cli=97fa1f6 -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1 | the terminal's items are taken only on a cut | `llmprovider/internal/wire/responses/responses.go:833` | `if cut {` |
| E2 | the take | `llmprovider/internal/wire/responses/responses.go:834` | `taken = s.takeTerminal(event.Response.Output)` |
| E3 | a completed stream finishes through `completed` | `llmprovider/internal/wire/responses/responses.go:840` | `resp, err = completed(result, s.refused)` |
| E4 | an answer with no output is `ErrEmptyAnswer` | `llmprovider/internal/wire/responses/responses.go:443` | `return nil, wire.EmptyAnswer("responses", r.FinishReason)` |
| E5 | what counts as delivered | `llmprovider/internal/wire/responses/responses.go:789` | `func (s *streamState) delivered(item outputItem, index int) bool {` |
| E6 | an item with no id, by position | `llmprovider/internal/wire/responses/responses.go:793` | `return index < len(s.done)` |
| E7 | each `.done` adds its item | `llmprovider/internal/wire/responses/responses.go:738` | `return s.itemDone(event.Item, yield), nil` |
| E8 | a delta is yielded, not kept | `llmprovider/internal/wire/responses/responses.go:721` | `return !yield(llmprovider.Event{Type: deltaType, Text: delta.Delta}), nil` |
| E9 | only calls are tracked when announced | `llmprovider/internal/wire/responses/responses.go:734` | `if event.Item.Type == itemTypeFunctionCall \|\| event.Item.Type == itemTypeToolSearchCall {` |
| E10 | held calls are released at the end | `llmprovider/internal/wire/responses/responses.go:845` | `if !s.held.Done(resp, yield) {` |
| E11 | taken items are yielded after them | `llmprovider/internal/wire/responses/responses.go:849` | `if slices.Contains(resp.Output, item) && !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}) {` |
| E12 | the plain decode reads `output` whole | `llmprovider/internal/wire/responses/responses.go:419` | `for _, out := range raw.Output {` |
| E13 | `HeldCalls.Done` | `llmprovider/internal/wire/finish.go:102` | `func (h *HeldCalls) Done(resp *llmprovider.Response, yield func(llmprovider.Event) bool) bool {` |
| E14 | the ChatGPT backend's terminal `output` is empty (tool golden) | `llmprovider/providers/openai/testdata/chatgpt-tool.sse:38` | `"moderation":null,"output":[],"parallel_tool_calls"` |
| E15 | its `.done` events carry `output_index` | `llmprovider/providers/openai/testdata/chatgpt-tool.sse:35` | `"output_index":0,"sequence_number":11}` |
| E16 | empty in the text golden too | `llmprovider/providers/openai/testdata/chatgpt-text.sse:26` | `"moderation":null,"output":[],"parallel_tool_calls"` |
| E17 | and in the reasoning golden | `llmprovider/providers/openai/testdata/chatgpt-reasoning.sse:26` | `"moderation":null,"output":[],"parallel_tool_calls"` |
| E18 | xAI's completed stream sends `output_item.done` | `docs/decisions/0031-PLAN-native-streaming-on-the-responses-wire.md:355` | `` reasoning_summary_part.done`, `output_item.done`, `content_part.added `` |
| E19 | each native decoder yields each item as `EventItem` | `docs/architecture.md:476` | `It yields each item as` |
| E20 | `ReadStream` and `Events` share one reader | `llmprovider/internal/wire/responses/responses.go:371` | `err := readEvents(provider, body, handledEvents, d, func(ev llmprovider.Event) bool {` |
| E21 | `openai` builds the reader | `llmprovider/providers/openai/openai.go:465` | `return deferral.Append, responses.Decoder{}` |
| E22 | the ChatGPT backend's plain call is read over the stream | `llmprovider/providers/openai/openai.go:587` | `return decoder.ReadStream(string(llmprovider.ProviderOpenAI), r)` |
| E23 | `grok` streams through it | `llmprovider/providers/grok/grok.go:338` | `decoder.Events, wire.Retries(limit.Check, effort.Check))` |
| E24 | OpenCode's responses route builds it | `llmprovider/providers/opencode/opencode.go:753` | `return responses.Decoder{Search: deferral.SearchName(c.req.Tools)}` |
| E25 | pi: an orphan `.done` gets a slot | `pi:packages/ai/src/api/openai-responses-shared.ts:557` | `return outputSlots.get(outputIndex) ?? createSlot(outputIndex, item);` |
| E26 | pi backfills from the terminal | `pi:packages/ai/src/api/openai-responses-shared.ts:581` | `backfillReasoningSignatures(response.output ?? []);` |
| E27 | pi: Azure's encrypted reasoning only in the terminal | `pi:packages/ai/src/api/openai-responses-shared.ts:559` | `// Azure OpenAI can omit reasoning.encrypted_content from response.output_item.done` |
| E28 | a delivered item's id is recorded, empty or not | `llmprovider/internal/wire/responses/responses.go:774` | `s.done = append(s.done, item.ID)` |
| E29 | codex's `ResponseCompleted` | `codex:codex-rs/codex-api/src/sse/responses.rs:111` | `struct ResponseCompleted {` |
| E30 | codex requires the terminal event | `codex:codex-rs/codex-api/src/sse/responses.rs:618` | `"stream closed before response.completed".into(),` |
| E31 | ST104, the take on a cut | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1450` | `on a cut the stream reader also takes each item the terminal` |
| E32 | R2, open for 0088's Phase 3 | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1417` | `**R2, Responses' cut call.**` |
| E33 | 0091 ST55, the check on `chatcompat` | `docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md:568` | `read as cut, and finished by 0088's cut answer (ST25)` |
| E34 | opencode: compatible providers omit `.done` | `opencode:packages/ai/src/protocols/open-responses.ts:1384` | `// Some compatible providers omit output_item.done even after completing the response.` |
| E35 | opencode reads the terminal `output` by index | `opencode:packages/ai/src/protocols/open-responses.ts:1369` | `resolveItem(state, item, index)` |
| E36 | opencode recovers a call it saw announced | `opencode:packages/ai/src/protocols/open-responses.ts:1378` | `current.tools[item.id] !== undefined` |
| E37 | opencode: an item with no id takes its output index's | `opencode:packages/ai/src/protocols/open-responses.ts:994` | `(index === undefined ? undefined : state.outputItems[index])` |
| E38 | qwen-code guards a non-compliant proxy | `docs/reports/0088-REPORT-qwen-code-output-limits-responses-streaming-and-caching.md:134` | `back to the done item, as a "non-compliant proxy" may do` |
| E39 | goose: the terminal `output` replaces the done items | `goose:crates/oi-kit-provider-types/src/formats/openai_responses.rs:1301` | `output_items = response.output;` |
| E40 | only when it is not empty | `goose:crates/oi-kit-provider-types/src/formats/openai_responses.rs:1300` | `if !response.output.is_empty() {` |
| E41 | Messages: `message_stop` finishes from the blocks | `llmprovider/internal/wire/messages/messages.go:540` | `resp, err := m.finish(texts)` |
| E42 | an unstopped block is folded in | `llmprovider/internal/wire/messages/messages.go:561` | `if m.Content[i].open {` |
| E43 | Chat: `[DONE]` finishes | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:664` | `if string(payload) == "[DONE]" {` |
| E44 | from the deltas kept | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:674` | `msg.Content, msg.Refusal = chatContent(s.content.String()), s.refusal.String()` |
| E45 | generateContent: each part kept | `llmprovider/internal/wire/generatecontent/generatecontent.go:327` | `parts.add(part)` |
| E46 | then the finish | `llmprovider/internal/wire/generatecontent/generatecontent.go:337` | `raw.Candidates[0].Content.Parts = parts.all()` |
| E47 | Interactions: the terminal gives status and usage | `llmprovider/providers/gemini/interactions.go:352` | `s.raw.Status, s.raw.Usage = final.Status, final.Usage` |
| E48 | the steps come from the stream's own | `llmprovider/providers/gemini/interactions.go:359` | `for i := range s.raw.Steps {` |
| E49 | a text delta is yielded only for a started `model_output` step | `llmprovider/providers/gemini/interactions.go:332` | `if step.Type == interactionStepModelOutput {` |
| E50 | the architecture's promise | `docs/architecture.md:477` | `` `EventItem`, and `EventDone` carries the `Response` that wire's `Decode` `` |
| E51 | the items are the `EventItem`s in order | `llmprovider/internal/wire/responses/events_0031_test.go:43` | `// one EventDone comes last, its Response is ReadStream's, its items are the` |
| E53 | gobble ends the turn after shown text | `gobble-cli:agent/agent.go:514` | `case err != nil && r != nil && r.shown:` |
| E54 | `ErrIncomplete`: a reply not read whole | `llmprovider/api_error.go:80` | `// filter stopped, a reply that could not be read whole, or one without` |
| E55 | gobble reruns only `ErrEmptyAnswer` | `gobble-cli:llm/provider/provider.go:762` | `errors.Is(ae.Kind, llmprovider.ErrEmptyAnswer) && (ae.Reason == "" \|\| ae.Reason == string(llmprovider.FinishStop))` |
| E56 | `WithLogger` | `llmprovider/settings.go:125` | `func WithLogger(logger *slog.Logger) Option {` |
| E57 | the wire already logs at debug | `llmprovider/internal/wire/post.go:115` | `s.logger.Debug("llmprovider: "+s.provider+": close response body", "error", err)` |
| E58 | no logger, nothing written | `llmprovider/settings.go:333` | `// Logger is the logger from WithLogger, or one that discards everything.` |
| E59 | what a Warning is | `llmprovider/warning.go:54` | `// Warning is a non-fatal note on a Response: the request was answered, but` |
| E60 | R1, open for 0088's Phase 3 | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1416` | `**R1, the wire-general cut rule.**` |
| E61 | R3, open for 0088's Phase 3 | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1418` | `**R3, streamed calls held back.**` |
| E62 | 0091's note: a completed stream does not take | `docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md:640` | `which a completed stream does not` |
| E63 | 0091 Phase 2 waits for 0088's Phase 3 | `docs/decisions/0091-PLAN-adopt-the-responses-api-per-provider.md:31` | `Phase 2 starts only after 0088's phases and its Phase 3 review are done (ST30)` |
| E64 | only `EventDone` carries a `Response` | `llmprovider/stream.go:46` | `// stream. Only EventDone carries a Response, and it comes last: the deltas` |
| E65 | `ErrIncomplete` matches `ErrInvalidRequest`, not retried | `llmprovider/api_error.go:84` | `// to ST81). Retrying it unchanged cannot help, so it also matches` |
| E66 | 0021-MADR D1 | `docs/decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md:168` | `#### D1. A failure while reading a 200 reply is retried once` |
| E67 | 0012-MADR §4.1 | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:452` | `**4.1 Streaming.**` |
| E68 | an early end is `ErrProviderUnavailable` after the reply | `llmprovider/internal/wire/sse.go:91` | `transport.AfterReply(fmt.Errorf("stream ended before %s", done)))` |
| E69 | asked again at most once | `llmprovider/internal/wire/sse.go:38` | `// WithRetry asks again at most once (0021-MADR D1). So is dispatch's error,` |
| E70 | an event over the limit is `ErrIncomplete` | `llmprovider/internal/wire/sse.go:57` | `tooLarge := fmt.Errorf("%w: %s: stream event over %d MiB", llmprovider.ErrIncomplete, provider, EventLimit>>20)` |
| E71 | `EventLimit` | `llmprovider/internal/wire/sse.go:19` | `const EventLimit = 16 << 20` |
| E72 | `ReplyLimit` | `llmprovider/internal/wire/decode_error.go:13` | `const ReplyLimit = 16 << 20` |
| E73 | `Collect` counts delta text | `llmprovider/internal/wire/overstream.go:55` | `size += len(ev.Text)` |
| E74 | and a yielded call's arguments | `llmprovider/internal/wire/overstream.go:58` | `size += len(call.Arguments)` |
| E75 | an empty message is not kept | `llmprovider/internal/wire/responses/responses.go:545` | `if text := sb.String(); text != "" {` |
| E76 | a hosted search, or one with no Search tool, is dropped | `llmprovider/internal/wire/responses/responses.go:568` | `if d.Search == "" \|\| out.Execution != executionClientValue {` |
| E78 | usage from the terminal event | `llmprovider/internal/wire/responses/responses.go:829` | `result.Usage = event.Response.Usage.counts()` |
| E79 | the reasoning-summary delta | `llmprovider/internal/wire/responses/responses.go:881` | `"response.reasoning_summary_text.delta": llmprovider.EventReasoningDelta,` |
| E80 | R1 as built | `llmprovider/internal/wire/finish.go:56` | `return !WholeObject(args) \|\| last && noArguments(args)` |
| E81 | R2 as built: an announced, unfinished call named on a cut | `llmprovider/internal/wire/responses/responses.go:836` | `result.Warnings = append(result.Warnings, wire.ToolCallCut(result.Model, o.Name, o.CallID))` |
| E83 | R3 as built: the hold | `llmprovider/internal/wire/finish.go:74` | `if !ok \|\| !CutCall(call.Arguments, true) {` |
| E84 | the xAI shapes test | `llmprovider/providers/cut_answer_0088_test.go:682` | `func TestCutAnswer_XAIShapes_0088(t *testing.T) {` |
| E85 | R2's test | `llmprovider/providers/cut_answer_0088_test.go:618` | `func TestCutAnswer_ResponsesCutItem_0088(t *testing.T) {` |
| E86 | R3's test | `llmprovider/providers/cut_answer_0088_test.go:633` | `func TestCutAnswer_HeldCalls_0088(t *testing.T) {` |
| E87 | `appendOutput` keeps reasoning only with text or an encrypted part | `llmprovider/internal/wire/responses/responses.go:585` | `if sb.Len() > 0 \|\| out.EncryptedContent != "" {` |
| E88 | opencode: a call's finished arguments replace the deltas | `opencode:packages/ai/src/protocols/open-responses.ts:1209` | `if (final !== undefined && !final.startsWith(tool.input))` |
| E89 | pi: the same | `pi:packages/ai/src/api/openai-responses-shared.ts:690` | `slot.block.partialJson = event.arguments;` |
| E90 | pi keeps an encrypted part the item already has | `pi:packages/ai/src/api/openai-responses-shared.ts:570` | `if (storedItem.encrypted_content) continue;` |
| E91 | `WithStreamRetry` stops retrying at the first event | `llmprovider/retry.go:221` | `//   - After the first event (a text or reasoning delta, or an item) every` |
| E92 | `Collect` marks a failure after an event | `llmprovider/internal/wire/overstream.go:44` | `return nil, transport.AfterEvent(err)` |
| E93 | which is never resent | `llmprovider/retry.go:342` | `if transport.IsAfterEvent(err) {` |
| E94 | an after-reply failure is resent once per call | `llmprovider/retry.go:176` | `// Once per Generate or Stream call (0021-MADR D1).` |

## Amendment (2026-11-24): the owner's answers to OQ1 to OQ6

Additions only; the rows above keep their text, each answered row annotated where it stands.

### Owner rulings (2026-11-24)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ1 | "Fail as retryable stream (Recommended)" | (1): an announced item never delivered fails the reply as a broken stream, retryable | ST11 |
| OQ2 | "Use summary when done was empty (Recommended)" | (1): a `.done` that added nothing is not counted as delivered; the terminal's copy is taken | ST12 |
| OQ3 | "Debug line via WithLogger (Recommended)" | (1): one debug line through the provider's logger per stream that took items | ST20 |
| OQ4 | "Own v1.33.0 phase (Recommended)" | (1): its own phase in v1.33.0, after 0088's Phase 3 review and before 0091-PLAN Phase 2 | ST27 |
| OQ5 | "Take nothing of that kind (Recommended)" | (1): in the ambiguous case, nothing of that type is taken | ST6 (f) |
| OQ6 | "The item's own message (Recommended)" | (1): the `.done` item stands against a differing terminal copy | ST7 |

The answers were checked against each other. OQ2 (1) and OQ5 (1) both act on pairing: an empty `.done` is paired by D2's passes as any `.done` is, so it keeps its place in the order pass, and only its partner's status changes (taken, not delivered). OQ1 (1) and OQ2 (1) meet on an announced message whose `.done` came empty: it was delivered by `.done`, so OQ1's check does not fire, and the terminal's copy is taken by OQ2; with no terminal copy, nothing is lost that `appendOutput` would have kept. OQ3 (1) adds no export, so ST25 stands: nothing new in the API.

### Rows decided (2026-11-24)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST6 (f) | After passes 1 and 2, k ≥ 1 unpaired `.done`s of a type and more than k unpaired terminal items of that type | decided | nothing of that type is taken from the terminal, on a completed and on a cut stream; the other types are paired and taken as D2 says. On a cut stream this narrows 0088's position rule (E6) for that type only | OQ5 "Take nothing of that kind (Recommended)" | `TestTerminalFill_Match_0092`'s case (f): terminal `[message A, message B]`, neither with an id, one `.done` message with B's text and no id or index; completed and cut, `Response.Output` holds the one `.done` message and no other message; plant: order pairing in that case (the `.done` pairs with A, B is taken, so B's text comes twice) |
| ST7 | The terminal's item differs from the `.done` item of the same id (M8) | decided | the `.done` item stands; the terminal neither replaces nor removes a delivered item; an empty terminal `output` removes nothing | OQ6 "The item's own message (Recommended)" | `TestTerminalFill_DoneWins_0092`, M8's shape keeps `Hello`, `EventItem` and `EventDone`'s item equal; plant: the terminal's item replacing the delivered one in `Response.Output` |
| ST11 | An announced item (`output_item.added`) of a tracked type delivered neither by `.done` nor by the terminal, on a completed stream (M3) | decided, conditional on 0088 R2 *(2026-11-25: rests on 0088-MADR ST108, R2 as built, decided)* | the reply fails with `ErrProviderUnavailable`, marked after the reply as an early end is (E68, E69), its text `item_not_finished` with the item's type and id, each stripped and bounded as a service's text is (E95); no response. Tracked types: `message`, `function_call`, and `tool_search_call` when the request has a Search tool; not `reasoning`, hosted or unknown types. A resend follows only as the retry rules allow: at most once, and never after an event (E91 to E94). On a cut stream R2 as built still names an announced call never finished as the cut call (E81); an announced message never finished on a cut is left as today. If R2 changes, `TestCutAnswer_ResponsesCutItem_0088`'s stream case (E85) flips | OQ1 "Fail as retryable stream (Recommended)"; 0088 R2 | `TestTerminalFill_Unfinished_0092`: M3 is the error, `errors.Is(err, ErrProviderUnavailable)`, after the reply, its text holding `item_not_finished` and the call's id; an announced message never delivered, the same; an announced reasoning item never delivered, a success as today; a `Generate`, read over the stream, of a reply whose only item is the lost call, under `WithRetry` over a fake server, sent twice; M3 under `WithStreamRetry`, sent once. Plants: the check removed (M3 finishes `stop`); the error built as `ErrIncomplete` (the kind assertion fails); reasoning tracked (the reasoning case fails) |
| ST12 | A `.done` that added nothing to the output, while its terminal partner has content: an empty message `.done` with the text in the terminal; a reasoning `.done` with neither text nor encrypted part, the encrypted part in the terminal (M7) | decided | the `.done` is paired as D2 says, but its partner is taken, not counted as delivered: added after the delivered items (ST9) and yielded before `EventDone`. A `.done` that added an item stands: a reasoning item kept with text but no encrypted part is not filled in | OQ2 "Use summary when done was empty (Recommended)" | `TestTerminalFill_EmptyDone_0092`: M7's reasoning item taken with its encrypted part, after the message; the empty message's text taken; a reasoning item kept with text and no encrypted part left as delivered. Plants: an empty `.done` counted as delivered (M7 drops the reasoning item); the kept item filled in place (the third case gains an encrypted part) |
| ST20 | A report that items were taken | decided | `responses.Decoder` gains an internal `Logger` field; `openai`, `grok` and OpenCode's responses route set it from the provider's logger (E56; `chatcompat` in 0091-PLAN Phase 2). A stream that took one or more items writes one `Debug` line: the provider, the terminal event's type, and the count taken of each type. No text, no id, no arguments, no encrypted content. With no logger nothing is written (E58). No export | OQ3 "Debug line via WithLogger (Recommended)" | `TestTerminalFill_Log_0092`, with a capturing `slog` handler: one line for M1, none for a compliant stream, no attribute holding `Hello` or an id; `TestTerminalFill_ProviderLogger_0092`: `openai`, `grok` and OpenCode's responses route, each given `WithLogger`, log the line for M1's shape. Plants: a line written for a compliant stream; the message text logged; `grok` leaving the field unset |
| ST27 | The release and the build order | decided *(2026-11-25: v1.33.0 read as v1.34.0, by the owner's "Yes, re-scope v1.33.0 (Recommended)"; v1.33.0 is tagged without it; see the amendment of 2026-11-25)* | its own phase in v1.33.0, after 0088-PLAN Phase 3's review and before 0091-PLAN Phase 2, serially in the same tree; a **Fixed** entry under `[Unreleased]`; v1.33.0's tag waits for it | OQ4 "Own v1.33.0 phase (Recommended)" | `make changelog-check`; plant: the entry outside `[Unreleased]` |

ST25 stands: no export (OQ3 (1)). ST26's OQ1 clause resolves to (1): the error is documented in `Events`' and `ReadStream`'s docs and in the changelog; `ErrIncomplete`'s doc does not change.

Counts after this amendment: 32 rows: 28 decided, 4 out of scope (ST21 to ST24), 0 open. The record is `accepted`.

### Evidence added (2026-11-24)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E95 | a service's text is stripped and bounded | `llmprovider/internal/wire/responses/responses.go:468` | `reason = redact.Field(reason)` |

## Amendment (2026-11-25): placed for v1.34.0, built next, re-grounded at `138d52f`

Additions only; no row's decision changes. The record and its PLAN are placed in the tree as the PLAN's Phase 0, at `138d52f`, the v1.33.0 release commit. The marker above stays `self=2f3c3a9`, where every row of the evidence index still passes; the rows below whose ids end in "now", as 0091-MADR's fourth amendment does, are this amendment's lines at `138d52f` (and, for gobble-cli, at `073fd1f`), which `tools/claims.py` does not read against the marker. Each was checked with a plain id against those commits (the PLAN's execution record, Phase 0).

### Owner rulings (quoted as given)

| Ruling | Given | Effect here |
| :--- | :--- | :--- |
| OQ1 "Fail as retryable stream (Recommended)", OQ2 "Use summary when done was empty (Recommended)", OQ5 "Take nothing of that kind (Recommended)", OQ6 "The item's own message (Recommended)", OQ3 "Debug line via WithLogger (Recommended)", OQ4 "Own v1.33.0 phase (Recommended)" | 2026-11-24 | the amendment of 2026-11-24 above; unchanged |
| "Approve; follow the queue (Recommended)" | 2026-11-24 | the PLAN, approved |
| "Yes, re-scope v1.33.0 (Recommended)" | 2026-11-24 | this record moves from v1.33.0 to v1.34.0 (E98now): OQ4's "Own v1.33.0 phase" is read as its own phase in v1.34.0 |
| "SDK: build 0092 (Recommended)" | 2026-11-25 | the PLAN's phases run next, from `138d52f` |

### What the move changes

* **ST27** (annotated in place). Its own phase, now in v1.34.0; v1.33.0 is tagged at `138d52f` without it (E101now), so "v1.33.0's tag waits for it" no longer applies. "After 0088-PLAN Phase 3's review" is met: that review became v1.33.0's short release review, done before the tag (E97now). "Before 0091-PLAN Phase 2" stands: that phase moved to v1.34.0 too, and has not run (E99now). The **Fixed** entry goes under `[Unreleased]` (E102now), above `[v1.33.0]`.
* **ST25.** "The last tag" is now v1.33.0.
* **ST10, ST11, ST13.** The 0088 rulings they assume (R3; R2; R1 and R2) are no longer 0088-PLAN Phase 3's: R1 to R5 ship as built and stay open for v1.34.0's review (E96now, E97now), and 0091-PLAN Phase 2 starts after they are ruled (E100now). Each row states its outcome against R1 to R3 as built, and the code they read (`finish.go`, `responses.go`) has no commit between `2f3c3a9` and `138d52f`, so each premise holds at `138d52f`. Whether Phase 1 builds before those rulings is Q-B1 below.

### Re-ground at `138d52f`

`python3 -B tools/claims.py` on this record with `--rev self=138d52f`: 82 of 92 rows pass; 10 fail, each found on one other line with its text unchanged. At the record's own marker, 92 of 92 pass. No commit between `2f3c3a9` and `138d52f` touches `responses.go`, `finish.go`, `overstream.go`, `sse.go`, `decode_error.go`, `post.go`, `stream.go`, `events_0031_test.go`, `messages.go`, `chatcompletions.go`, `generatecontent.go`, `interactions.go` or the ChatGPT goldens: the 0088 Phase 1b and 1c changes to the Responses wire (`f4efc5e`) precede `2f3c3a9`, so `takeTerminal` is still taken only under `if cut {` at `responses.go:833-834` (E1, E2 pass), and M1 to M11, measured at `2f3c3a9` on those files, stand. The commits in between (0089's caller-limit refusal report, `9e86748`; 0093's catalog change, `be6828e`; the release, `138d52f`) move the lines below.

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| E19now | each native decoder yields each item as `EventItem` | from `docs/architecture.md:476` | `docs/architecture.md:472` | `It yields each item as` |
| E21now | `openai` builds the reader | from `openai.go:465` | `llmprovider/providers/openai/openai.go:468` | `return deferral.Append, responses.Decoder{}` |
| E22now | the ChatGPT backend's plain call is read over the stream | from `openai.go:587` | `llmprovider/providers/openai/openai.go:590` | `return decoder.ReadStream(string(llmprovider.ProviderOpenAI), r)` |
| E23now | `grok` streams through it | from `grok.go:338` | `llmprovider/providers/grok/grok.go:341` | `decoder.Events, wire.Retries(limit.Check, effort.Check))` |
| E24now | OpenCode's responses route builds it, with a Search tool | from `opencode.go:753` | `llmprovider/providers/opencode/opencode.go:751` | `return responses.Decoder{Search: deferral.SearchName(c.req.Tools)}` |
| E24bnow | and without one: `responsesDecoder` has two returns, as at `2f3c3a9`; ST20's `Logger` is set on both | new | `llmprovider/providers/opencode/opencode.go:753` | `return responses.Decoder{}` |
| E50now | the architecture's promise | from `docs/architecture.md:477` | `docs/architecture.md:473` | `` `EventItem`, and `EventDone` carries the `Response` that wire's `Decode` `` |
| E59now | what a Warning is | from `warning.go:54` | `llmprovider/warning.go:57` | `// Warning is a non-fatal note on a Response: the request was answered, but` |
| E91now | `WithStreamRetry` stops retrying at the first event | from `retry.go:221` | `llmprovider/retry.go:271` | `//   - After the first event (a text or reasoning delta, or an item) every` |
| E93now | which is never resent | from `retry.go:342` | `llmprovider/retry.go:408` | `if transport.IsAfterEvent(err) {` |
| E94now | an after-reply failure is resent once per call | from `retry.go:176` | `llmprovider/retry.go:226` | `// Once per Generate or Stream call (0021-MADR D1).` |
| E53now | gobble ends the turn after shown text, at gobble-cli `073fd1f` (its text changed; the fact holds) | from `agent/agent.go:514` at `97fa1f6` | `gobble-cli:agent/agent.go:560` | `case err != nil && (r.shown \|\| r.started && accidental(err)):` |
| E55now | gobble reruns only `ErrEmptyAnswer`, at gobble-cli `073fd1f` | from `llm/provider/provider.go:762` at `97fa1f6` | `gobble-cli:llm/provider/provider.go:811` | `errors.Is(ae.Kind, llmprovider.ErrEmptyAnswer) && (ae.Reason == "" \|\| ae.Reason == string(llmprovider.FinishStop))` |

### Facts that no longer hold, or are new

* **E32, E60, E61** ("R1/R2/R3, open for 0088's Phase 3"): the lines hold, but R1 to R3 are now open for v1.34.0's review (E96now, E97now).
* **E63** ("0091 Phase 2 waits for 0088's Phase 3"): the line holds, but Phase 2 moved to v1.34.0 and waits for R1 to R3 to be ruled there (E99now, E100now).
* **OQ4 and ST27** ("its own phase in v1.33.0"): v1.34.0, as above.
* **New since the drafts, 0089-MADR DL78:** a provider sets a cap it kept on the `*APIError` that ends a failed call (E103now). ST11's error is built as an early end's is, so it carries whatever DL78 gives an early end's error on that provider; `openai` and `grok` keep no cap on this wire (E104now, E105now). No row of this record changes.
* **The number.** `python3 -B scripts/check_records.py --next` prints `0094` at `138d52f` (the highest record, 0093, plus one). 0092 was reserved for this record before 0093 was numbered (E106now), so the pair takes 0092: the number was never used, so nothing is reused.

### Put to the owner, not resolved here

**Q-B1. Phase 1 runs before 0088's R1 to R3 are ruled.** The PLAN, as approved, ran Phase 1 after 0088-PLAN Phase 3 had ruled R1 to R3, and its step 1 stops if a ruling changes what ST10, ST11 or ST13 assume. The re-scope moved those rulings to v1.34.0's review (E96now), so under "SDK: build 0092 (Recommended)" Phase 1 runs with R1 to R3 unruled, against the code as built. No row is open: each of the three states its outcome against R1 to R3 as built, and names the test that flips.

| Option | Cost |
| :--- | :--- |
| (1) **Build now against R1 to R3 as built (recommended).** Phase 1's step 1 records that R1 to R3 are unruled at its base; when v1.34.0's review rules them, a changed ruling is a dated amendment here, with the tests ST10, ST11 and ST13 name re-pinned. | If the review changes R1, R2 or R3, the take's order against held calls (ST10), the cut path's unfinished call (ST11) or the cut rules a taken call meets (ST13) are reworked after this phase has landed, as one more phase; v1.34.0's review reads both records together. |
| (2) Put R1 to R3 to the owner now, then build. | Three rulings, and a 0088-MADR amendment with any code they bring and its tests, before Phase 1 starts; the build ruling waits for them. |
| (3) Hold Phase 1 for v1.34.0's review, which rules R1 to R3. | 0092 waits for a review that has no date; 0091-PLAN Phase 2 waits for the same review, so the order between the two must be ruled again. |

Doing nothing builds under (1) by default, with no record that the owner chose it. *(2026-11-25: answered. The owner's answer, relayed by the coordinator, quoted exactly: "Rule R1–R3 first", option (2). See "Amendment (2026-11-25, second)".)*

### Evidence added (2026-11-25)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E96now | R1 to R5 ship as built and stay open for v1.34.0's review | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1513` | `and stays open for v1.34.0's review, where the reviewer rules on each with the owner.` |
| E97now | 0088-PLAN Phase 3 became a short release review; its R1 to R3 step moved | `docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md:927` | `Step 4 (R1 to R3) moves to v1.34.0 with R4 and R5.` |
| E98now | 0092's move to v1.34.0 | `docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md:929` | `They target v1.34.0, where their drafts say v1.33.0` |
| E99now | 0091-PLAN Phase 2 moved to v1.34.0, and 0092 with it | `docs/decisions/0091-PLAN-adopt-the-responses-api-per-provider.md:163` | `0092's records, which add cases to Phase 2, also target v1.34.0.` |
| E100now | the opt-in ships after R1 to R3 are ruled | `docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md:812` | `ship in v1.34.0, after 0088's R1 to R3 are ruled there.` |
| E101now | v1.33.0 is released | `CHANGELOG.md:20` | `## [v1.33.0] - 2026-11-24` |
| E102now | `[Unreleased]` is open above it | `CHANGELOG.md:18` | `## [Unreleased]` |
| E103now | DL78: the kept cap on the error that ends a call | `llmprovider/internal/wire/limit_rule.go:258` | `func (r *LimitRetry) Finish(resp *llmprovider.Response, err error, observe func(output int)) error {` |
| E104now | `openai` keeps no cap on this wire | `llmprovider/providers/openai/openai.go:116` | `// stated cap, so none is kept, and the error's APIError.Observed stays` |
| E105now | nor does `grok` | `llmprovider/providers/grok/grok.go:100` | `// stated cap, so none is kept, and the error's APIError.Observed stays` |
| E106now | 0092 was reserved for this record | `docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md:15` | `0092 is reserved by` |

Counts after this amendment: unchanged, 32 rows: 28 decided, 4 out of scope (ST21 to ST24), 0 open. The record is `accepted`.

## Amendment (2026-11-25, second): Q-B1 answered; ST10, ST11 and ST13 rest on 0088's decided rows

Additions only; no row's decision changes. Q-B1, ST10, ST11 and ST13 are annotated where they stand.

### Owner rulings (2026-11-25, second)

Relayed by the coordinator; labels quoted exactly.

| Question | The owner's answer | Option taken | Effect here |
| :--- | :--- | :--- | :--- |
| Q-B1 | "Rule R1–R3 first" | (2): R1 to R3 are ruled before Phase 1 | the order below |
| 0088's R1 | "Keep, except where calls arrive whole (Recommended)" | 0088-MADR ST107 (E109now): the last-`{}` cut rule only on the wires that stream arguments in pieces (Messages, Responses, Chat Completions); on generateContent and Interactions a last `{}` call is whole | ST13 rests on it; no change on Responses |
| 0088's R2 | "Keep as built (Recommended)" | 0088-MADR ST108 (E110now) | ST11 and ST13 rest on it; no change |
| 0088's R3 | "Keep as built (Recommended)" | 0088-MADR ST109 (E111now) | ST10 rests on it; no change |

### The order

1. [0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md) Phase 6 (E112now), R1 by wire (0088-MADR ST107 and ST110). It waits for 0088-MADR's OQ19, the streamed Interaction, put to the owner with R1's ruling, and for the owner's word.
2. Then this record's PLAN Phase 1, on Phase 6's commit.
3. Then 0091-PLAN Phase 2, as ST27 says.

So Phase 1 waits for OQ19 through Phase 6, although no row here depends on ST110 (ST24 puts Interactions out of this record's scope). *(2026-11-25: OQ19 answered, "Decide by what arrived". ST110 is decided and 0088-MADR is `accepted`; Phase 6 now waits only for the owner's word.)*

### ST10, ST11 and ST13, re-pointed

| Row | Rested on | Now rests on | What changes |
| :--- | :--- | :--- | :--- |
| ST10 | 0088 R3 as built (E61, E83) | 0088-MADR ST109, decided | nothing. `TestCutAnswer_HeldCalls_0088` (E86) pins a decided row: it no longer flips on a ruling still to come |
| ST11 | 0088 R2 as built (E32, E81) | 0088-MADR ST108, decided | nothing. `TestCutAnswer_ResponsesCutItem_0088` (E85) pins a decided row |
| ST13 | 0088 R1 and R2 as built (E60, E80, E81) | 0088-MADR ST107 on the Responses wire, and ST108, both decided | nothing on Responses. `TestCutAnswer_XAIShapes_0088` (E84) and `TestTerminalFill_Match_0092`'s cut cases stand as planned |

### Whether R1's change touches a row here: measured, it does not

* ST107 changes the last-`{}` rule only on generateContent and an Interaction's plain reply, and 0088's OQ19 decides the streamed Interaction. Responses is a wire that streams arguments in pieces: ST107 keeps R1 as built there. A cut reply's last call with `{}`, and a call whose arguments are not a JSON object, stay the cut call.
* The Responses wire finishes a cut through `wire.CutAnswer` on both paths, plain (E107now) and streamed (E108now). 0088-PLAN Phase 6's step 3 passes "in pieces" from both. `HeldCalls` (E13, E83) runs on the Responses stream unchanged (ST109).
* Measured on a scratch clone at `4fc5211`, with the existing `cutRun` fixtures (`<scratch>/r1/measure-r1.txt`): a cut Responses reply of text, `lookup {"x":1}` and a last `now` call with no arguments returns `lookup` alone, with one `tool_call_cut` warning, by `Generate` and by `Stream`. That is ST107's outcome for Responses, so Phase 6 changes nothing this record reads. At the tree, `TestCutAnswer_ResponsesCutItem_0088`, `TestCutAnswer_HeldCalls_0088` and `TestCutAnswer_XAIShapes_0088` pass (`go test -count=1`, this Mac, exit 0).
* **What moves.** Phase 6 edits `finish.go`, so E13 (`:102`), E80 (`:56`) and E83 (`:74`) may move by line; this record's PLAN Phase 1 step 1 re-grounds them, as it already lists. E31, E32, E60 and E61 hold their text on their lines: 0088's fifteenth amendment annotates without adding a line above them. E96now's text is also unchanged on its line, and R1 to R3 are now ruled there.
* **Still open in 0088, and not this record's:** R4 and R5, for v1.34.0's review. Neither is on a row here.

### Evidence added (2026-11-25, second)

The ids end in "now", as the first amendment of 2026-11-25 does. Each was checked with a plain id against a scratch clone: at `4fc5211` for the code, and at that clone's commit of the tree's 0088 records for the rest. Each was also planted false there (this record's PLAN, the execution entry of 2026-11-25, second).

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E107now | the Responses wire's plain cut through `CutAnswer` | `llmprovider/internal/wire/responses/responses.go:423` | `return wire.CutAnswer("responses", result, reasonOutputLimit)` |
| E108now | and its stream's | `llmprovider/internal/wire/responses/responses.go:838` | `resp, err = wire.CutAnswer("responses", result, reasonOutputLimit)` |
| E109now | 0088's ST107, R1 ruled | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1590` | `\| ST107 (new, 2026-11-25; R1) \|` |
| E110now | 0088's ST108, R2 ruled | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1591` | `\| ST108 (new, 2026-11-25; R2) \|` |
| E111now | 0088's ST109, R3 ruled | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1592` | `\| ST109 (new, 2026-11-25; R3) \|` |
| E112now | 0088-PLAN Phase 6, R1's change | `docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md:971` | `### Phase 6 (v1.34.0): R1 by wire (ST107, ST110)` |

Counts after this amendment: unchanged, 32 rows: 28 decided, 4 out of scope (ST21 to ST24), 0 open. The record is `accepted`.
