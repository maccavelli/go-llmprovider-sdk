---
status: accepted
date: 2026-10-12
decision-makers: repository owner
consulted: Anthropic's extended-thinking guide, read 2026-10-12; 0036-PLAN-cache-write-token-count.md's live payloads; reference clients (below)
informed: any caller that reads `Usage.ReasoningTokens` or `Usage.CachedTokens`
---

<!-- markdownlint-disable MD013 -->

# Read Anthropic's Reasoning Count, and a Fourth Cache-Read Spelling on Chat Completions

## Context and Problem Statement

0036-PLAN-cache-write-token-count.md's Phase 1 recorded each service's raw
usage. Two of those bodies show `Usage` counts the SDK does not decode. Both
were recorded there as "found, outside this PLAN".

### 1. Anthropic now reports a reasoning count

**What the SDK asserts.** 0015-MADR-canonical-sdk-api-and-module-layout.md's
amendment "what `Usage` counts" (2026-10-02) lists the services' conventions.
0015-PLAN-canonical-sdk-api-and-module-layout.md's Phase S9 states:
"Anthropic reports no reasoning count, so `ReasoningTokens` is zero, and a
test asserts it."

* The Messages decoder reads `input_tokens`, `output_tokens` and the cache
  fields, and nothing else (`internal/wire/messages/messages.go:468-481`).
* `internal/wirecase/wirecase.go`'s `messagesUsage` has no reasoning count,
  "Messages reports no reasoning count".

**What Anthropic documents.** The extended-thinking guide
(`platform.claude.com/docs/en/build-with-claude/extended-thinking`, read
2026-10-12) says:

> To track what a budget actually costs you, monitor the
> `usage.output_tokens_details.thinking_tokens` field in the response, which
> reports how many of the billed output tokens were internal reasoning. When
> streaming, this breakdown appears only on the final `message_delta` event.

So:

* `thinking_tokens` is a part of `output_tokens`, as `ReasoningTokens` is a
  part of `OutputTokens` (R7);
* a stream carries it on the last `message_delta` only, not on
  `message_start`.

**What was measured.** 0036-PLAN's Phase 1 bodies carry it, at zero, since
no request asked for reasoning:

* Claude called directly (T2):
  `"output_tokens":4,"output_tokens_details":{"thinking_tokens":0}`;
* OpenCode Zen's Messages route (T6): the same field, passed through.

No recorded body has a non-zero count. The API reference page was too large
to read in full, so the guide is the source here.

**What the reference clients do:**

| Client | Reads `thinking_tokens`? |
| :--- | :--- |
| **Pi** (`312184edb`) | yes, on a stream's `message_delta`, as `usage.reasoning`: "Anthropic reports reasoning tokens as a subset of output tokens" (`packages/ai/src/api/anthropic-messages.ts:786-790`), since 2026-06-25 (`d7868b099`) |
| **OpenCode** (`907b3bc518`) | no: "Extended thinking tokens are *not* broken out by Anthropic … so `reasoningTokens` stays `undefined`" (`packages/llm/src/protocols/anthropic-messages.ts:566-572`) |
| **OpenCode's Zen gateway** (same checkout, `packages/console`) | no: `reasoningTokens: undefined` (`routes/zen/util/provider/anthropic.ts:181`) |
| **grok-build** (`2bdd1d6a`) | no: its Messages stream reads no reasoning count |

OpenCode states the same belief this SDK recorded. Anthropic's guide now
contradicts it. Pi reads the field.

**The cost of leaving it.** `ReasoningTokens` is zero for every Claude
response, including one that reasoned at length. A caller cannot show how
much of the output was reasoning. Price is unaffected: Anthropic bills
reasoning as output, and `OutputTokens` already holds it.

### 2. A cache read spelled at the top level, on Chat Completions

The Chat Completions decoder reads the cache read from three spellings, in
Pi's order, as 0015-PLAN-canonical-sdk-api-and-module-layout.md's Phase S9
decoded them (`chatcompletions.go:831-871`):

1. `prompt_tokens_details.cached_tokens`;
2. DeepSeek's `prompt_cache_hit_tokens`;
3. a top-level `cached_tokens`.

**What was measured** on OpenCode Zen's Chat Completions route with
`deepseek-v4.1-flash` (0036-PLAN T8):

| Run | Request 1 | Request 2 |
| :--- | :--- | :--- |
| Phase 1 | top-level `cache_read_input_tokens` 0, and **no** `prompt_tokens_details` | `cache_read_input_tokens` 8960 **and** `prompt_tokens_details.cached_tokens` 8960 |
| Phase 4 | `prompt_tokens_details.cached_tokens` 0 only | `prompt_tokens_details.cached_tokens` 8922 only |

* The spelling varies between requests to the same model and route, so it
  depends on the upstream that Zen picks.
* When both spellings came, they agreed, and the read was inside
  `prompt_tokens` (8,960 of 9,051).
* **No recorded body has a non-zero read under the top-level spelling
  alone.** The loss is possible, not observed.
* The Zen gateway's own code (OpenCode `907b3bc518`,
  `routes/zen/util/provider/openai-compatible.ts:367-379`) reads
  `cache_read_input_tokens` from an Anthropic-shaped upstream reply, and
  writes it out as `prompt_tokens_details.cached_tokens`. The top-level
  field that reached the SDK was passed through from some other upstream,
  not shaped by that function.
* No vendor document for Chat Completions names a top-level
  `cache_read_input_tokens`. It is Anthropic's Messages name, used outside
  Messages, as Kilo uses `cache_creation_input_tokens`
  (0036-MADR-cache-write-token-count.md, amendment 2026-10-12).

## Decision Drivers

* **R7:** each `Usage` count says what the service reported, and a total
  holds its parts.
* **No existing count changes meaning,** and the exported API does not
  change.
* **A documented spelling wins** over a gateway's own when both are
  present. That is the order 0015-PLAN's Phase S9 and 0036-MADR's
  amendment follow.
* **Records stay true:** a fact the SDK asserts and a vendor now contradicts
  is amended, not left standing.
* **Measured, not assumed:** each change is confirmed live before release.

## Considered Options

For the reasoning count:

* Read `output_tokens_details.thinking_tokens` as `ReasoningTokens`, in a
  body and on a stream's last `message_delta`
* Leave `ReasoningTokens` at zero for Messages
* Estimate it from the thinking blocks' text

For the cache read:

* Read a top-level `cache_read_input_tokens` as a fourth spelling, after
  the three
* Record the spelling, and change nothing until a read is lost
* Read it first, before `prompt_tokens_details.cached_tokens`

## Decision Outcome

Chosen options:

* "Read `output_tokens_details.thinking_tokens` as `ReasoningTokens`",
  because Anthropic documents it as the part of the billed output that was
  reasoning, and that is exactly what `ReasoningTokens` is;
* "Read a top-level `cache_read_input_tokens` as a fourth spelling, after
  the three", because it can only fill a read that would otherwise decode
  as zero, and never displaces a documented spelling (Q1, answered).

### D1. Messages: `ReasoningTokens` from `thinking_tokens`

* **The body:** `ReasoningTokens` is
  `usage.output_tokens_details.thinking_tokens`. `OutputTokens` stays
  `output_tokens`, which already includes it.
* **The stream:** the merge's list gains the reasoning count. A count that
  `message_delta` leaves out keeps `message_start`'s, as today. Anthropic
  sends it only on the last `message_delta`, so `message_start`'s is zero,
  and the delta's replaces it.
* **Absent** (an older model, or a gateway that drops it), it stays zero, as
  every `Usage` count does when it is not reported.
* **Every Messages route** gets it: Claude, and OpenCode Zen's and OpenCode
  Go's Messages routes, since they share the wire.
* **The invariant** `ReasoningTokens <= OutputTokens` (R7, `llmtest`)
  already holds for a part of the output.

### D2. Chat Completions: a fourth cache-read spelling

* The read order becomes:
  1. `prompt_tokens_details.cached_tokens`;
  2. `prompt_cache_hit_tokens`;
  3. a top-level `cached_tokens`;
  4. a top-level `cache_read_input_tokens`.
* **The first present wins,** as now. Zen's request 2 above, with both, keeps
  decoding from the details.
* **It is inside `prompt_tokens`,** as measured, so `InputTokens` does not
  change.

### D3. The records

* **0015-MADR's amendment "what `Usage` counts"** gains a dated note: its
  statement that Anthropic reports no reasoning count is superseded by this
  record's D1. The history is not rewritten.
* **`internal/wirecase`'s** Messages reply gains a reasoning count, and its
  comment changes to match.
* **`docs/architecture.md`'s `Response.Usage` paragraph** names both
  spellings.
* **The Chat Completions decoder's comment** cites "0017-REPORT P7" for the
  read spellings. P7 there is Pi's error handling. The comment cites
  0015-PLAN's Phase S9 instead.

### D4. Out of scope

* **Asking Claude to reason** is already `Request.Reasoning`. This record
  only reads the count.
* **Zen's and OpenCode's own decoders,** which still assume no reasoning
  count, belong to those projects.
* **Other gateways' spellings** that have not been measured.

### Consequences

* Good, because a Claude response that reasoned reports how much of its
  output was reasoning, as OpenAI, Gemini and xAI responses already do.
* Good, because a record that a vendor now contradicts is corrected, with
  its history kept.
* Good, because a Zen read under the top-level spelling no longer decodes as
  zero.
* Neutral, because the exported API does not change. A caller that relied
  on `ReasoningTokens` being zero for Claude sees real counts.
* Neutral, because pricing is unchanged: reasoning is billed as output, and
  `OutputTokens` already holds it.
* Bad, because D2 rests on two bodies from one route, and no non-zero read
  under that spelling alone has been seen. It may never fire.

### Confirmation

* **Red first:** a Messages body and a stream with `thinking_tokens`, and a
  Chat Completions body with only a top-level `cache_read_input_tokens`,
  each fail before the decoder reads them. Every plant's FAIL lines are
  recorded.
* **`wirecase`:** every Messages provider and route decodes the canned
  reasoning count.
* **The goldens** are unchanged, and apidiff against `v1.10.0` reports no
  change.
* **Live:** Claude directly, and OpenCode Zen's Messages route, each asked
  to reason. The decoded `ReasoningTokens` equals
  `output_tokens_details.thinking_tokens`, it is above zero, and it is no
  larger than `OutputTokens`. The PLAN states the billed count first.
* **D2** is checked on canned bodies. It is checked live only if a Zen reply
  happens to carry the top-level spelling. That reply cannot be forced.

## Pros and Cons of the Options

### Reading `thinking_tokens`

* Good, because it is Anthropic's documented field, and the measured bodies
  carry it.
* Good, because Pi reads it the same way.
* Neutral, because on a stream it arrives only at the end. A caller reading
  usage mid-stream sees zero until then, as it does for the output count.

### Leaving it at zero

* Good, because it costs nothing.
* Bad, because it reports zero for reasoning Anthropic says happened, and it
  leaves a record that a vendor contradicts.

### Estimating from the thinking text

* Bad, because the visible thinking is summarized, so its length does not
  measure the billed reasoning.
* Bad, because it is a guess where a reported count exists.

### A fourth spelling, last

* Good, because it can only fill a zero.
* Good, because the documented spelling still wins when both are present.
* Bad, because it is one more spelling to keep, for a loss not yet seen.

### Record only

* Good, because nothing changes without a lost read to show.
* Bad, because a read under that spelling alone would decode as zero, with
  nothing to say so.

### Reading it first

* Bad, because an undocumented field would displace OpenAI's documented one.

## More Information

### Owner questions

Answered by the owner on 2026-10-12.

* **Q1 (D2). The fourth spelling.** Offered:
  * **read it, last** (recommended). One field and one test row. It cannot
    change a read any documented spelling gives;
  * **record only.** No decoder change until a non-zero read under the
    top-level spelling alone is seen.

  **Answer: read it, last.** D2 stands as written.

### Related

* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md),
  amendment "what `Usage` counts": the fact D1 supersedes.
* [0036-MADR-cache-write-token-count.md](0036-MADR-cache-write-token-count.md)
  and
  [0036-PLAN-cache-write-token-count.md](0036-PLAN-cache-write-token-count.md):
  the measurement that found both, and the Kilo spelling decided the same
  way.
* [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md),
  Phase S9: the three read spellings, in Pi's order, and the test that
  asserts Anthropic's reasoning count is zero.
* The plan: [0037-PLAN-reasoning-and-cache-read-spellings.md](0037-PLAN-reasoning-and-cache-read-spellings.md).
  The owner accepted this record on 2026-10-12, after answering Q1.
