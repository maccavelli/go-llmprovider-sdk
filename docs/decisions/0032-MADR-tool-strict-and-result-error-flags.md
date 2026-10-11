---
status: proposed
date: 2026-10-11
decision-makers: repository owner
consulted: gobble-cli 0012-MADR-tool-schemas-descriptions-and-loading.md (D3, D13), gobble-cli 0014-REPORT-llmprovider-sdk-integration.md (F5, F7), 0015-MADR-canonical-sdk-api-and-module-layout.md (D3)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 -->

# Carry a Per-Tool Strict Flag and a Tool Result's Error Flag

## Context and Problem Statement

gobble-cli, the SDK's agent consumer, needs two things `llmprovider.Tool`
and `llmprovider.FunctionCallOutputItem` cannot carry. These come from
gobble-cli's records, cited by repository and filename, which a link from
here cannot reach, and from a read-only survey of its code on 2026-10-10.

* **A per-tool strict flag.**
  * OpenAI's Responses API applies strict mode to a function tool unless
    the tool says `"strict": false`.
  * Strict mode makes every property required, so a model fills in
    arguments a schema leaves optional. gobble-cli measured `gpt-4.1-mini`
    inventing `limit: 50` for its `read` tool.
  * gobble-cli 0012-MADR D3, amended 2026-10-08 by its owner's choice:
    gobble opts out of strict mode for tools that have optional arguments,
    and keeps it for tools whose properties are all required. "It needs the
    SDK", so that decision is blocked until a release carries the flag. Its
    D13 records the request.
* **A tool result's error flag.** gobble-cli sets `IsError` on a failed
  tool's result and keeps it in its session files, but its adapter drops it
  (`llm/provider/provider.go`, `toRequest`). Anthropic's Messages API has a
  documented `is_error` on `tool_result`, which tells the model the call
  failed.

What the SDK sends today, read at `fa16e59`:

| Wire | Tool definition | Tool result |
| :--- | :--- | :--- |
| Responses (`wire.ResponsesTools`) | `type`, `name`, `description`, `parameters`; no `strict` | `function_call_output` with `call_id`, `output` |
| Chat Completions (`chatcompletions.toolList`) | `function.name`, `.description`, `.parameters`; no `strict` | a `tool` message with `tool_call_id`, `content` |
| Messages (`wire.MessagesTools`) | `name`, `description`, `input_schema` | `tool_result` with `tool_use_id`, `content`; no `is_error` |
| generateContent | `functionDeclarations` | `functionResponse` with `name` and `response: {"output": …}` |
| Interactions (`gemini`) | `function` tools | a `function_result` step with `call_id`, `name`, `result` |

The question: how should the SDK carry these two flags, and what should
each wire do with them?

## Decision Drivers

* **gobble-cli is blocked on the strict flag** (its 0012-MADR D3).
* **Nothing changes for a caller who sets neither.** Today's requests, and
  the G-wire goldens, stay byte for byte.
* **One contract, no vendor JSON** (0015-MADR D3;
  [api-standards.md](../guides/api-standards.md) R5, R9): a field on the
  existing types, not a pass-through of provider keys.
* **Capabilities stay honest** (R10–R12): a wire that cannot carry a flag
  says so in its package doc.
* **Measured, not assumed.** Whether xAI, the OpenCode gateways and the
  Chat Completions gateways accept `strict`, and whether the Gemini wires
  have any error form for a result, has not been measured here.

## Considered Options

* **Two fields:** `Tool.Strict *bool`, nil for the service's default; and
  `FunctionCallOutputItem.IsError bool`. Each wire sends what it has a
  field for.
* **`Tool.Strict bool`:** false would mean strict off, so every caller
  would lose OpenAI's default without asking.
* **A per-request provider option**, such as `openai.WithStrictTools(names
  …)`: options are fixed at construction, while strictness is per tool per
  request.
* **A raw pass-through**, `Tool.Extra map[string]any` merged into each
  definition: vendor JSON in the contract, and one caller's keys sent to
  every wire.

## Decision Outcome

Chosen option: "Two fields", because a nil pointer keeps every request
unchanged, the fields name the two meanings gobble-cli needs and no more,
and each wire maps them to its own documented field or says it has none.

### D1. `Tool.Strict *bool`

* `nil` (the zero value) sends nothing: the service's default applies, as
  today.
* `true` or `false` sends `"strict"` with that value on a wire that has the
  field.
* A caller sets it from a variable of its own (`&v`). Whether the package
  should export a pointer helper is Q1.

### D2. Where `Strict` is sent

* **Responses** (`openai`, `grok`, OpenCode's `responses` route): the
  definition's `strict`. This is OpenAI's documented opt-out. xAI and the
  OpenCode route are gated on measurement, as 0031-MADR D1 was. One that
  refuses the field leaves it unsent and says so in its package doc.
* **Chat Completions:** `function.strict`, OpenAI's Chat Completions
  spelling. It is sent only on a gateway the measurement shows accepting it
  (Q2). Elsewhere it is left out and the package doc says so.
* **Messages, generateContent, Interactions:** not sent. Each package doc
  names it as a degradation (R12). If a measurement shows Anthropic
  accepting a documented `strict`, adding it is an amendment here.
* `Capabilities` gains no field. Strictness is a hint about the
  arguments' shape, and its absence never refuses a request.

### D3. `FunctionCallOutputItem.IsError bool`

* `false` (the zero value) sends what is sent today.
* `true`:
  * **Messages** (`claude`, OpenCode's `messages` route): `tool_result`'s
    `is_error: true`.
  * **generateContent** (OpenCode's `google` route): `functionResponse`'s
    `response` is `{"error": <output>}` in place of `{"output": <output>}`.
    This is Gemini's documented convention for a failed function, confirmed
    or corrected by measurement.
  * **Interactions** (`gemini`): the `function_result` step's error form,
    if the measurement finds one. Otherwise the output is sent unchanged
    and the package doc says so.
  * **Responses, Chat Completions:** no field exists. The output is sent
    unchanged (Q3), and each package doc says so.
* The output text is never rewritten. Whatever a caller wants the model to
  read about the failure, it writes in `Output`.

### D4. Proof

* **G-wire goldens:** the existing goldens are unchanged. New wirecase
  scenarios pin, on every wire, a tool with `Strict` set each way and a
  result with `IsError`.
* **`llmtest` Fidelity:** a harness that sets `Fidelity` also checks that a
  `Strict: false` tool and an `IsError` result reach the wire as that
  provider documents. This adds no `Harness` field; it uses what each
  harness already records.
* **Release:** `v1.7.0`. The two struct fields are additions, which apidiff
  reports as compatible.

### Owner questions

* **Q1 (D1). A pointer helper.** Offered:
  * export `llmprovider.Ptr[T any](v T) *T`, as other Go SDKs do, which
    adds one function to the API;
  * no helper: callers write `strict := false; tool.Strict = &strict`
    (recommended, since the API stays as small as the need).
* **Q2 (D2). Chat Completions.** Offered:
  * send `function.strict` on each gateway the measurement shows accepting
    it (recommended);
  * send it on the Responses wire only.
* **Q3 (D3). A result's error on a wire with no field.** Offered:
  * send the output unchanged (recommended);
  * prefix the output with a fixed marker, such as `"Error: "`, which
    rewrites the caller's text.
* **Q4. One record or two.** Offered:
  * one record for both fields (recommended): both are small tool-fidelity
    additions gobble-cli asked for together, released together;
  * a record each.

### Consequences

* Good, because gobble-cli can opt `read`, `bash` and `powershell` out of
  strict mode on OpenAI, unblocking its 0012-MADR D3.
* Good, because a failed tool call reaches Claude, and Gemini through
  OpenCode, as a failure.
* Good, because a caller who sets neither field sends byte-for-byte what it
  sends today.
* Neutral, because the release is a minor one: two struct fields.
* Bad, because a flag is honoured on some wires and not others. Each
  package doc names the gap.
* Bad, because the measurement is a handful of billed requests.

### Confirmation

* G-wire goldens unchanged; the new wirecase scenarios pass on every wire.
* Each new test is seen failing first.
* The live measurement is recorded before the wires that depend on it are
  built.
* apidiff from `v1.6.0` shows additions only.

## Pros and Cons of the Options

### Two fields

* Good, because nil and false keep every request unchanged.
* Good, because each field has one meaning, mapped per wire.
* Bad, because a pointer field is clumsier to set than a plain bool.

### `Tool.Strict bool`

* Good, because it is simple to set.
* Bad, because its zero value would turn OpenAI's strict mode off for every
  caller, a silent change of behaviour.

### A per-request provider option

* Bad, because options are construction-time, and strictness belongs to
  each tool of each request.
* Bad, because it would be OpenAI-only by shape.

### A raw pass-through

* Bad, because it puts vendor JSON in the contract (R5).
* Bad, because one caller's keys would reach wires that refuse them.

## More Information

* **Asked for by:** gobble-cli `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`
  D3 and D13, and gobble-cli
  `docs/reports/0014-REPORT-llmprovider-sdk-integration.md` F5 and F7, read
  in that repository.
* **Related:** [0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md),
  the first of the gobble-cli items. The others (model metadata, image
  input, prompt caching, tool deferral, timeouts) are their own records.
* The plan is
  [0032-PLAN-tool-strict-and-result-error-flags.md](0032-PLAN-tool-strict-and-result-error-flags.md).
