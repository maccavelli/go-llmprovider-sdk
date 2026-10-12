---
status: complete
date: 2026-10-12
associated-madr: "0037-MADR-reasoning-and-cache-read-spellings.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Anthropic's Reasoning Count and the Fourth Cache-Read Spelling

Associated MADR: [0037-MADR-reasoning-and-cache-read-spellings.md](0037-MADR-reasoning-and-cache-read-spellings.md)

## Goal

When it is done:

* the Messages wire decodes `usage.output_tokens_details.thinking_tokens` as
  `Usage.ReasoningTokens`, in a full response and in a stream, on every
  Messages route (0037-MADR D1);
* the Chat Completions wire reads a top-level `cache_read_input_tokens` as
  its fourth cache-read spelling, last in the order (D2);
* 0015-MADR's amendment "what `Usage` counts", the canned replies, the
  decoder comments and `docs/architecture.md` say so (D3);
* a live run shows a non-zero reasoning count from Claude, called directly
  and through OpenCode Zen, decoded exactly as sent;
* `v1.10.1` is released, with no change to the exported API.

## Scope

### In scope

* **Decoders:**
  * `llmprovider/internal/wire/messages/messages.go`: `usage`, `counts`,
    and the stream merge in `Events`;
  * `llmprovider/internal/wire/chatcompletions/chatcompletions.go`: `usage`,
    `counts`, and the comment's citation.
* **Tests:**
  * `usage_test.go` in `messages` and `chatcompletions`;
  * `llmprovider/internal/wirecase/wirecase.go`: the Messages reply, its
    expected `Usage`, and its comment;
  * a live measurement, `llmprovider/live_0037_reasoning_measure_test.go`.
* **Records and docs:**
  * a dated note in 0015-MADR's amendment "what `Usage` counts";
  * `docs/architecture.md`'s `Response.Usage` paragraph;
  * `README.md` Status, and this PLAN's release notes;
  * `docs/README.md`'s index.

### Out of scope

As 0037-MADR D4 decides:

* any change to how a request asks Claude to reason;
* OpenCode's and the Zen gateway's own decoders;
* spellings no measurement has shown;
* the exported API: no field or function is added.

## Rules for every phase

* **Red first,** with every FAIL line recorded.
* **Plants** through `tools/plant.py`, on a scratch clone, never in the
  tree.
* **Every script is Python;** a reusable one belongs in `tools/`.
* **A phase is staged only after:**
  * `make pre-add-check FILES="…"`;
  * `make records-check`;
  * `tools/denyscan.py --changed` and `tools/linkcheck.py`;
  * markdownlint on any changed non-record Markdown;
  * the gate, `tools/gate.py`, at 23 of 23.
* **Billed requests:** the count goes to the owner, and the owner approves,
  before any live run. Keys are checked for presence only.
* **A deviation stops the work:** evidence, options with the recommended
  one first, and a dated entry here, plus a MADR amendment where a fact or
  a decision changes.
* **The owner commits** each phase, pushes and tags.

## Implementation Steps

### Phase 0: records

1. 0037-MADR `accepted` (the owner, 2026-10-12). This PLAN, `proposed`.
2. **0015-MADR's amendment "what `Usage` counts"** gains, after its "Fact
   found" list, a dated note. It says that, on 2026-10-12, Anthropic's
   extended-thinking guide documents
   `usage.output_tokens_details.thinking_tokens` as the reasoning part of
   the billed output, and that 0037-MADR D1 supersedes the statement that
   Anthropic reports no reasoning count. The amendment's text is not
   changed.
3. `docs/README.md`: the rows and the count.
4. `make records-check`, `tools/linkcheck.py`, `tools/denyscan.py
   --changed`. Stage. The owner commits.

### Phase 1: the decoders

1. **Red first:**

   | Test | Input | Want |
   | :--- | :--- | :--- |
   | `messages.TestDecode_Reasoning` (new) | a body with `"output_tokens":40,"output_tokens_details":{"thinking_tokens":30}` and the cache counts of `TestDecode_CacheWrites` | `{InputTokens:120 OutputTokens:40 ReasoningTokens:30 CachedTokens:100 CacheWriteTokens:5 CacheWrite1hTokens:2}` |
   | `messages.TestDecode_Reasoning`, a second body | `output_tokens_details` absent | `ReasoningTokens: 0` |
   | `messages.TestEvents_Reasoning` (new) | a stream whose `message_start` has `output_tokens` 1 and no `output_tokens_details`, and whose last `message_delta` has `"output_tokens":40,"output_tokens_details":{"thinking_tokens":30}`, as Anthropic's guide says | the `EventDone` `Response`'s `ReasoningTokens` is 30 and `OutputTokens` 40 |
   | `messages.TestEvents_Reasoning`, a second stream | `message_delta` with `output_tokens` 40 only | `ReasoningTokens: 0` |
   | `chatcompletions.TestDecode_Usage`, a new row | `{"prompt_tokens":120,"completion_tokens":40,"cache_read_input_tokens":100}` | `CachedTokens: 100` |
   | `chatcompletions.TestDecode_Usage`, a second row | `prompt_tokens_details.cached_tokens` 100 and a top-level `cache_read_input_tokens` 90 | `CachedTokens: 100`: the details win |
   | `chatcompletions.TestDecode_Usage`, a third row | `cached_tokens` 80 at the top level and `cache_read_input_tokens` 90 | `CachedTokens: 80`: the third spelling wins over the fourth |

2. **`internal/wirecase`, red first:**
   * `MessagesReply`'s usage gains `"output_tokens_details":{"thinking_tokens":30}`,
     so it reports the same 30 reasoning tokens as the other wires' replies.
   * `messagesUsage` gains `ReasoningTokens: 30`.
   * The comment "Messages reports no reasoning count" becomes the rule: each
     reply reports 30 of its 40 output tokens as reasoning, and Messages
     reports it as `thinking_tokens`.
   * Every Messages provider and route (Claude; OpenCode Zen's and OpenCode
     Go's Messages routes) fails `Usage = …, want the canned reply's …`
     until the decoder reads it.
3. **Change:**
   * **Messages:** `usage` gains

     ```go
     OutputDetails struct {
     	Thinking int `json:"thinking_tokens"`
     } `json:"output_tokens_details"`
     ```

     `counts` sets `ReasoningTokens: u.OutputDetails.Thinking`. The stream
     merge's list gains `{&m.Usage.OutputDetails.Thinking,
     &ev.Usage.OutputDetails.Thinking}`, so `message_delta`'s count replaces
     `message_start`'s zero.
   * **The comment on `usage`** drops "the API reports no reasoning count",
     and says that `thinking_tokens` is the reasoning part of
     `output_tokens` (0037-MADR D1).
   * **Chat Completions:** `usage` gains

     ```go
     CacheReadTop *int `json:"cache_read_input_tokens"`
     ```

     last in `counts`' read list. The comment names the four spellings and
     their order, and cites 0015-PLAN-canonical-sdk-api-and-module-layout.md's
     Phase S9 instead of "0017-REPORT P7".
4. **`docs/architecture.md`:** the `Response.Usage` paragraph says
   Anthropic's reasoning count comes from `thinking_tokens`, and names the
   fourth read spelling.
5. **Plants,** each failing a named test:
   * Messages `counts` leaving out `ReasoningTokens`;
   * Messages reading `thinking_tokens` from the top level of `usage`;
   * the stream merge without the reasoning count;
   * the stream merge adding `message_delta`'s count to `message_start`'s,
     with a `message_start` that carries 30;
   * Chat Completions leaving `cache_read_input_tokens` out of the list;
   * Chat Completions reading it before `prompt_tokens_details.cached_tokens`;
   * Chat Completions reading it before the top-level `cached_tokens`.
6. **The goldens:** `TestWireGoldens` passes three times unchanged, through
   the gate's `g-wire-stable`. They keep no usage.
7. Gate; stage. The owner commits.

### Phase 2: live confirmation and release

1. **`llmprovider/live_0037_reasoning_measure_test.go`**
   (`//go:build live_gateways`, package `llmprovider_test`):
   * **`TestLive_0037ReasoningCount`,** skipped unless
     `LLMPROVIDER_LIVE_0037` is set, and per target unless its key is set.
   * **Targets, 2:**
     * R1 `claude`, `claude-haiku-5-5`;
     * R2 `opencode-zen`, `claude-haiku-5-5`, then `claude-sonnet-5-5`,
       through the live model picker.
   * **Two calls per target:** `Generate`, a full response, and
     `llmprovider.Stream`, which streams natively on Messages. Both send
     the same prompt: a short arithmetic task worded to need reasoning,
     with `Reasoning{Effort: EffortHigh}` and `WithMaxTokens(2048)`.
   * **A recording transport** keeps each reply's body. For a stream, the
     test reads the last `message_delta`'s `usage` from the event stream.
   * **It asserts:**
     * the decoded `ReasoningTokens` equals the raw `thinking_tokens`, and
       `OutputTokens` equals `output_tokens`;
     * `ReasoningTokens <= OutputTokens`;
     * at least one of each target's two calls reports `thinking_tokens`
       above zero. Adaptive thinking may skip an easy task, so a target
       that reports zero on both is a deviation.
   * **It logs** the usage object and the decoded `Usage`, never a prompt,
     header or key.
   * **`TestReasoningChecks_0037`,** in the same file, runs the helpers on
     canned bodies and a canned stream, with no key and no network. It is
     red first, and `tools/plant.py` breaks each helper in turn.
2. **Cost, for the owner before the run:** 4 billed requests, each a short
   prompt and at most 2,048 output tokens. That is at most 8,192 output
   tokens in all, about US$0.004 at Claude Haiku 5.5's US$0.50 per million
   (models.dev, 2026-10-12), and more if R2 falls back to
   `claude-sonnet-5-5`.
3. **Run it,** after the owner's approval:

   ```text
   LLMPROVIDER_LIVE_0037=1 go test -tags live_gateways -count=1 -v \
     -run TestLive_0037ReasoningCount ./llmprovider/
   ```

4. **What decides a deviation:**
   * `thinking_tokens` found somewhere other than `output_tokens_details`;
   * a stream that carries it on `message_start`, or on a `message_delta`
     that is not the last, contrary to the guide. The merge would still
     decode it, but the MADR's fact would need amending;
   * `thinking_tokens` larger than `output_tokens`;
   * both calls of a target reporting zero.
5. **Docs:** `README.md` Status names `v1.10.1`. This PLAN gains the release
   notes.
6. Gate; stage. The owner commits, runs the disclosure guard, pushes, and
   tags `v1.10.1`.
7. **After the tag:**
   * CI on `main` and on the tag;
   * the proxy: `v1.10.1`, Go 1.27.2, on the tagged commit;
   * `tools/apidiff.py v1.10.0 v1.10.1`: no change;
   * `tools/consumer.py v1.10.1`: the consumer, built for six platforms;
   * `tools/offline.py v1.10.1`: still passes.
8. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with every plant's FAIL lines.
* **V2. The gate,** 23 of 23, at each phase. The goldens are unchanged.
* **V3. Canned:** every Messages provider and route decodes the canned
  reasoning count, through `wirecase`.
* **V4. Live:** Claude, directly and through Zen, reports a non-zero
  `thinking_tokens`, in a full response and in a stream. The decoded count
  equals it.
* **V5. API:** apidiff `v1.10.0` to `v1.10.1` reports no change.

## Rollout and Rollback

* **Rollout:** the records, then Phase 1, then Phase 2 and `v1.10.1`, each
  a commit the owner makes.
* **Compatibility:** no exported identifier changes. A Claude response that
  reasoned now reports `ReasoningTokens` above zero, where it reported zero.
  `OutputTokens` is unchanged, so a caller's price is unchanged.
* **Rollback:** Phase 1 reverts alone, and `ReasoningTokens` reads zero on
  Messages again. After the tag, a fault is fixed forward in a patch
  release.

## Execution Record

### Phase 0: records (2026-10-12)

* 0037-MADR was accepted, and this PLAN written; the owner committed both
  (`41a18fe`), then directed "committed, proceed".
* **0015-MADR's amendment "what `Usage` counts"** gains a dated note after
  its "Fact found" list. The statement it supersedes is in 0015-PLAN's
  Phase S9, "Anthropic reports no reasoning count", and the note says so.
  The amendment's own text is unchanged.
* `make records-check`, `tools/linkcheck.py` and `tools/denyscan.py
  --changed`: clean.

Committed by the owner as `1c1a43f`.

### Phase 1: the decoders (2026-10-12)

**Change:**

* **Messages** (`internal/wire/messages/messages.go`): `usage` gains
  `OutputDetails.Thinking`, from `output_tokens_details.thinking_tokens`.
  `counts` sets `ReasoningTokens` from it, and the stream merge's list gains
  it. The comment no longer says the API reports no reasoning count.
* **Chat Completions:** `usage` gains `CacheReadTop`, from a top-level
  `cache_read_input_tokens`, last in the read list. The comment names the
  four spellings and their order, and cites 0015-PLAN's Phase S9 instead of
  "0017-REPORT P7".
* **`internal/wirecase`:** `MessagesReply` reports `thinking_tokens` 30.
  `messagesUsage` gains `ReasoningTokens: 30`, and the comment states the
  rule.
* **`docs/architecture.md`:** the `Response.Usage` paragraph names
  `thinking_tokens` and the four read spellings.

**Tests,** as step 1 lists them:

* `messages.TestDecode_Reasoning`, with and without `output_tokens_details`;
* `messages.TestEvents_Reasoning`, three streams: the count on the last
  `message_delta` only, not reported, and on both events. The third is the
  case the "adding" plant needs.
* `chatcompletions.TestDecode_Usage`, three rows: the fourth spelling
  alone, the details before it, and the top-level `cached_tokens` before it.

**Red first:**

* `TestDecode_Reasoning`: `with thinking_tokens: Usage = {… ReasoningTokens:0
  …}, want {… ReasoningTokens:30 …}`.
* `TestEvents_Reasoning`: `on the last message_delta: Usage = {…
  ReasoningTokens:0 …}, want ReasoningTokens 30 and OutputTokens 40`, and
  the same for `on both events`.
* `chatcompletions.TestDecode_Usage`: `cache_read_input_tokens: Usage =
  {… CachedTokens:0 …}, want {… CachedTokens:100 …}`. The two order rows
  passed before the change, since nothing read the field. They are there
  for the plants.
* With the canned reply, `TestWireGoldens` failed on every Messages route,
  and on no other: `claude`, `claude-refuses-forced`,
  `opencode-zen-messages` and `opencode-go-messages`, with `wirecase.go:329:
  Usage = {… ReasoningTokens:0 …}, want the canned reply's {…
  ReasoningTokens:30 …}`.

**Green:** `go test ./...`, all 29 packages.

**Plants,** through `tools/plant.py`:

| Plant | Caught by | First FAIL line |
| :--- | :--- | :--- |
| Messages `counts` leaving out `ReasoningTokens` | `TestDecode_Reasoning`, `TestEvents_Reasoning`, `TestWireGoldens/claude/…`, `…/claude-refuses-forced/items` | `with thinking_tokens: Usage = {… ReasoningTokens:0 …}` |
| Messages not reading `output_tokens_details` | the same | the same |
| the stream merge without the reasoning count | `TestEvents_Reasoning` | `on the last message_delta: Usage = {… ReasoningTokens:0 …}` |
| the stream merge adding `message_delta`'s count to `message_start`'s | `TestEvents_EqualsDecode` and its subtests, and `TestEvents_Reasoning` | `on both events: Usage = {… OutputTokens:41 ReasoningTokens:60 …}, want ReasoningTokens 30 and OutputTokens 40` |
| Chat Completions leaving `cache_read_input_tokens` out | `TestDecode_Usage` | `cache_read_input_tokens: Usage = {… CachedTokens:0 …}` |
| Chat Completions reading it before the details | `TestDecode_Usage` | `the details before cache_read_input_tokens: Usage = {… CachedTokens:90 …}` |
| Chat Completions reading it before the top-level `cached_tokens` | `TestDecode_Usage` | `cached_tokens before cache_read_input_tokens: Usage = {… CachedTokens:90 …}` |

Step 5 named the second plant "reading `thinking_tokens` from the top level
of `usage`". It ran as the `output_tokens_details` field not being read, the
one-edit form of the same fault. `plant.py` printed only the head of the
fourth plant's output, so it was rerun against `TestEvents_Reasoning` alone,
which gave the line above.

**The goldens:** the gate's `g-wire-stable` ran `TestWireGoldens` three
times, unchanged.

**Checks:** `make pre-add-check` on the five Go files, clean (gofmt,
golangci-lint, go vet, go test, govulncheck); markdownlint on
`docs/architecture.md`, 0 issues.

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 9.6.

Committed by the owner as `3d7a360`.

### Phase 2: live confirmation and release (2026-10-12)

**Step 1, the test** (`llmprovider/live_0037_reasoning_measure_test.go`):

* `parseReasoning0037` reads a Messages reply, either a JSON body or an
  event stream. For a stream it takes the last `message_delta`'s counts,
  and notes whether `message_start`, or a `message_delta` before the last,
  carried `thinking_tokens`.
* `problems` reports `thinking_tokens` above `output_tokens`, and a count
  sent anywhere but the last `message_delta`. `decodedProblems` reports a
  decoded `ReasoningTokens` or `OutputTokens` that differs from the reply,
  and `ReasoningTokens` above `OutputTokens`.
* `TestLive_0037ReasoningCount` is switched on by `LLMPROVIDER_LIVE_0037`.
  It has the two targets and two calls step 1 gives, at
  `Reasoning{Effort: EffortHigh}` and `WithMaxTokens(2048)`. A target
  whose two calls both report zero fails.
* `TestReasoningChecks_0037` runs the checks on 7 canned replies.

**Red first:**

* `undefined: parseReasoning0037`;
* then, with the checks stubbed to report nothing, five canned replies
  passed: `problems "", want one containing "above output_tokens"`,
  `"message_start"`, `"not the last"`, `"decoded ReasoningTokens 0, raw
  thinking_tokens 30"` and `"decoded OutputTokens 1, raw output_tokens 40"`.

**Green:** 7 of 7. `go vet -tags live_gateways ./llmprovider/` is clean.
`gofmt` rewrapped two closures in the target table; nothing else moved.

**Plants,** through `tools/plant.py`:

| Plant | FAIL line |
| :--- | :--- |
| the above-the-output check dropped | `problems "decoded ReasoningTokens 50 above OutputTokens 40", want one containing "above output_tokens"` |
| the `message_start` check dropped | `problems "", want one containing "message_start"` |
| the not-the-last check dropped | `problems "", want one containing "not the last"` |
| the decoded reasoning comparison dropped | `problems "", want one containing "decoded ReasoningTokens 0, raw thinking_tokens 30"` |
| the decoded output comparison dropped | `problems "decoded ReasoningTokens 30 above OutputTokens 1", want one containing "decoded OutputTokens 1, raw output_tokens 40"` |

**Steps 2 and 3, the live run.** The owner approved 4 billed requests.
`ANTHROPIC_API_KEY` and `OPENCODE_API_KEY` were set, checked for presence
only. The run exited 0, and both targets passed:

| Target | Call | Raw usage, the parts that bear on reasoning | Decoded |
| :--- | :--- | :--- | :--- |
| R1 claude `claude-haiku-5-5` | generate | `"output_tokens":253,"output_tokens_details":{"thinking_tokens":153}` | `OutputTokens:253 ReasoningTokens:153` |
| R1 claude | stream | `message_start` `"output_tokens":3`, no `output_tokens_details`; `message_delta` `"output_tokens":239,"output_tokens_details":{"thinking_tokens":105}` | `OutputTokens:239 ReasoningTokens:105` |
| R2 opencode-zen `claude-haiku-5-5` | generate | `"output_tokens":313,"output_tokens_details":{"thinking_tokens":172}` | `OutputTokens:313 ReasoningTokens:172` |
| R2 opencode-zen | stream | `message_start` `"output_tokens":3`, no `output_tokens_details`; `message_delta` `"output_tokens":328,"output_tokens_details":{"thinking_tokens":177}` | `OutputTokens:328 ReasoningTokens:177` |

* **Anthropic's guide holds:** in both streams, `message_start` had no
  `output_tokens_details`, and the one `message_delta`, the last, carried
  it.
* **OpenCode Zen** passes the field through unchanged, in a body and in a
  stream.
* **No deviation:** each count was found where D1 says, no larger than the
  output, and above zero on every call.

**Step 5, docs:** `README.md` Status names `v1.10.1`, with a bullet.

**Checks:** `make pre-add-check` on the test file, clean (gofmt,
golangci-lint, go vet, go test, govulncheck); markdownlint on `README.md`,
0 issues; `tools/linkcheck.py`, 0 problems.

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 10.0.

#### Release notes: `v1.10.1`

`v1.10.1` reads two `Usage` counts that the SDK missed (0037-MADR). The
exported API does not change.

**Claude's reasoning count.** On every Messages route (Claude, and OpenCode
Zen's and OpenCode Go's Messages routes), `Usage.ReasoningTokens` is now
`usage.output_tokens_details.thinking_tokens`: the part of the billed output
that was reasoning, as Anthropic's extended-thinking guide documents it.

* A stream takes it from the last `message_delta`, where Anthropic sends it.
* `OutputTokens` is unchanged and still includes it, so a price computed
  from `OutputTokens` is unchanged.
* A Claude response that reasoned reported 0 before. It now reports the
  count, measured at 105 to 177 tokens on Claude Haiku 5.5 at high effort.

**A fourth cache-read spelling.** Chat Completions reads a top-level
`cache_read_input_tokens` as `CachedTokens`, when none of
`prompt_tokens_details.cached_tokens`, `prompt_cache_hit_tokens` or a
top-level `cached_tokens` is present. OpenCode Zen passes it through from
some upstreams.

**Upgrading:** no code change is needed. A caller that assumed
`ReasoningTokens` is always 0 for Claude sees real counts.

**Step 6, the release.**

* The disclosure guard passed over the four outgoing commits, `41a18fe` to
  `47920f2`.
* The owner pushed `main` and tagged `v1.10.1`, annotated (`7281e02`), on
  `47920f2`.

**Step 7, after the tag:**

| Check | Result |
| :--- | :--- |
| CI on `main` | run `38310723576`, push, success |
| CI on the tag | run `38311390239`, push, success |
| The proxy | `"Version": "v1.10.1"`, `"GoVersion": "1.27.2"`, `Ref refs/tags/v1.10.1`, hash `47920f2…` |
| apidiff `v1.10.0` to `v1.10.1` | `tools/apidiff.py`, exit 0, no output: no change to the exported API |
| `tools/consumer.py v1.10.1` | 15 packages linked, built for six platforms, every step passed |
| `tools/offline.py v1.10.1` | passed: 19 fields, `requests=0`, six platforms, both its plants caught |

**Step 8:** this PLAN is `complete`, and `docs/README.md` says so.

#### Verification, confirmed

| | Result |
| :--- | :--- |
| V1. Red, then green | each phase's record gives its red runs and every plant's FAIL lines |
| V2. The gate | 23 of 23 at each phase; the goldens unchanged |
| V3. Canned | every Messages provider and route decodes the canned reasoning count, through `wirecase` |
| V4. Live | Claude, directly and through Zen, reported 105 to 177 reasoning tokens, in bodies and streams, decoded exactly |
| V5. API | apidiff `v1.10.0` to `v1.10.1`: no change |

#### Deviations, in all

None changed a decision or a stated fact. Two plants in Phases 1 and 2 ran
in a one-edit form of the plan's wording, and `plant.py` cut one plant's
output short so it was rerun alone. Each phase's record says so.

#### Not done, and why

* **D2 has not fired live.** No reply under the top-level
  `cache_read_input_tokens` alone has been seen, as 0037-MADR's consequences
  state. It is checked on canned bodies only.
* **OpenCode's and the Zen gateway's own decoders** still assume no
  reasoning count from Anthropic (0037-MADR D4). That is those projects'
  work.
* **Requests that ask for caching** remain the next gobble-cli item
  (0036-MADR D4).
