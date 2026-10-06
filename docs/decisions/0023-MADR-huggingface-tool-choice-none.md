---
status: accepted
date: 2026-10-06
decision-makers: repository owner
consulted: 0020-MADR-remediate-v1-debugging-pass-findings.md (F40, F43), 0015-MADR-canonical-sdk-api-and-module-layout.md (R10, R48), 0012-MADR-conform-providers-to-reference-clients.md (§1.6)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Keep Hugging Face's `ToolChoiceNone` by Sending No Tools

> **Status.** Accepted 2026-10-05 by the owner ("approved, proceed"), with
> the option proposed: "Send no tools for `ToolChoiceNone`, in the Hugging
> Face provider".

## Context and Problem Statement

`ToolChoiceNone` is documented as "forbids tool calls"
(`llmprovider/contract.go:100`). The Hugging Face provider sends it as the
Chat Completions value: the request's tools, and `"tool_choice":"none"`
(`providers/huggingface/huggingface.go:159-160`; its package comment's
Degradations, `:28-29`).

**Measured 2026-10-05.** The full live suite on `e90fad6` (`v1.2.0` plus the
0022 test change) had 117 passes, 19 skips, and 2 failures. Both failures
are `TestLive_HuggingFaceToolChoices/none` and its parent:

```text
Generate: llmprovider: invalid request: huggingface HTTP 400 tool_use_failed: Tool choice is none, but model called a tool
```

* It failed 3 runs of 3 on a scratch copy of `HEAD`. The same test passed in
  0021-PLAN's phase 1 and 2 live runs (2026-10-04), so the service changed,
  not the code.
* The router's `X-Inference-Provider` header names `groq` for
  `openai/gpt-oss-120b`. `tool_use_failed` is the upstream provider's
  refusal of a reply whose tool call it was told to forbid. The router
  passes it on as a 400.
* **A probe** on a scratch copy, three runs each, with `ToolChoiceNone` and
  one tool. It recorded only each outcome and the provider header:

  | Model | Request | Outcome |
  | :--- | :--- | :--- |
  | `openai/gpt-oss-120b` | tools and `"tool_choice":"none"` (today's) | 400 `tool_use_failed`, 3/3 |
  | `openai/gpt-oss-120b` | no `tools`, no `tool_choice` | a text reply, no call, 3/3 |
  | `openai/gpt-oss-20b` | tools and `"tool_choice":"none"` | 400 `tool_use_failed`, 3/3 |
  | `openai/gpt-oss-20b` | no `tools`, no `tool_choice` | not measured: `HTTP 402`, the account's monthly credits used up |

* **The account's credits ran out during the probe.** Every Hugging Face
  call now answers `HTTP 402 ... You have depleted your monthly included
  credits`, which `SkipIfTransient` skips, until the credits reset or are
  bought. Any further live measurement waits for that.

`required` and a named tool are unaffected: `required` passed in the same
runs, and a named tool is what the goldens pin
(`testdata/wire/huggingface/tool.json`, `thinking-tool.json`).

The repository has met this before. 0020-MADR F40 found that a service that
cannot take `tool_choice` keeps `ToolChoiceNone` only when no tools are
sent. `chatcompletions.Opts.NoToolChoice` does that for Ollama, always, and
for Kilo, when a model does not list `tool_choice`. But `NoToolChoice` also
stops sending `"required"` and a named tool, which Hugging Face honours. So
it is the wrong switch here.

How should the Hugging Face provider keep `ToolChoiceNone`?

## Decision Drivers

* R10: the contract says `ToolChoiceNone` forbids calls. A 400 for every
  `none` request on the router's default route is a broken capability, not
  a degradation.
* Keep what works: `"required"` and a named tool are sent and honoured.
* The smallest change to code that four other providers share
  (`internal/wire/chatcompletions`), where nothing else needs it today.
* R48: no exported API change.
* Evidence before a wire change: the fix is the form the probe measured, and
  the forms not measured are named as such.

## Considered Options

* Send no tools for `ToolChoiceNone`, in the Hugging Face provider
* A `chatcompletions.Opts` switch for "no tools under none", set by Hugging Face
* Set `NoToolChoice` for Hugging Face
* Keep the wire; retry once without tools after `tool_use_failed`
* Keep the wire; record `ToolChoiceNone` as unsupported on Hugging Face

## Decision Outcome

Chosen option: "Send no tools for `ToolChoiceNone`, in the Hugging Face
provider", because it is the form the probe measured to work. It keeps
`"required"` and a named tool as they are, and it touches one function in
one provider, leaving the shared wire as it is.

* **The rule:** in `huggingface.go`'s `body`, when `req.ToolChoice ==
  ToolChoiceNone`, `chatcompletions.Opts` gets no `Tools` and the
  auto choice. So the request has neither `tools` nor `tool_choice`, and the
  model cannot call one, as F40 does for Ollama.
* **Unchanged:**
  * `"required"` and a named tool send the tools and their `tool_choice`;
  * auto sends the tools with no `tool_choice`;
  * the goldens, which pin a named tool;
  * the `Capabilities`, `ForcedToolChoice: BestEffort` (0020-MADR F43);
  * every other provider, and `chatcompletions`.
* **A history with tool calls:** a request under `none` may carry earlier
  `FunctionCallItem`s and `FunctionCallOutputItem`s, sent as assistant
  `tool_calls` and `tool` messages, with no `tools` defined. Whether the
  router and Groq accept that is not measured. 0023-PLAN measures it live
  before the PLAN closes, and stops for a decision if it is refused.
* **Docs:** the package comment's Degradations line says `ToolChoiceNone` is
  kept by sending no tools, with this record and the measurement.

### Consequences

* Good, because `ToolChoiceNone` works on Hugging Face again, as measured on
  `gpt-oss-120b`.
* Good, because `"required"` and a named tool are untouched.
* Good, because the change is local: no shared wire, no golden, no API.
* Neutral, because it matches what F40 already does for Ollama and Kilo, in a
  provider that does send `tool_choice` otherwise.
* Bad, because the model no longer sees the tools' definitions under
  `none`. A reply that would have mentioned them may differ, though the
  contract forbids calls either way.
* Bad, because `gpt-oss-20b` and a history with tool calls are not measured
  yet, and cannot be until the account's credits return.

### Confirmation

* **Offline:** `TestGenerate_RequestFields`'s `none` row sends no `tools`
  and no `tool_choice`. It fails on `HEAD` and passes after. The other rows
  are unchanged.
* **Live,** when credits allow: `TestLive_HuggingFaceToolChoices` passes,
  `none` included; and a scratch run of `none` with a tool-call history
  answers without a 400.
* The repository gate is clean (0023-PLAN).

## Pros and Cons of the Options

### Send no tools for `ToolChoiceNone`, in the Hugging Face provider

* Good, because it is the measured fix.
* Good, because it changes one function in one provider.
* Neutral, because if a second provider needs it, it can move into
  `chatcompletions` then, with that provider's evidence.
* Bad, because the model loses the tool definitions as context under `none`.

### A `chatcompletions.Opts` switch for "no tools under none", set by Hugging Face

* Good, because the rule would live beside `NoToolChoice`, ready for another
  provider.
* Bad, because it adds a second knob to shared code for one caller, and the
  two knobs' interplay needs its own tests.
* Bad, because no other provider has shown the failure. OpenCode's chat
  route and Kilo passed `none` live in the same run, and Together's `none`
  was not run (its opt-in was unset).

### Set `NoToolChoice` for Hugging Face

* Good, because it is a one-word change, already tested for Ollama and Kilo.
* Bad, because it also stops sending `"required"` and a named tool, which
  Hugging Face honours: `required` passed live in the same runs. It would
  break the goldens and weaken `ForcedToolChoice`.

### Keep the wire; retry once without tools after `tool_use_failed`

* Good, because the model sees the tools whenever the upstream allows it.
* Bad, because every `none` request on the default route would cost two
  billed calls, on a router that meters every call (0012-MADR §1.6).
* Bad, because it adds a retry path keyed on one upstream's error code,
  which the router does not document.

### Keep the wire; record `ToolChoiceNone` as unsupported on Hugging Face

* Good, because it changes no request.
* Bad, because the contract has no capability for `none`, so it would
  either need an exported API change (R48) or be a note callers do not
  read.
* Bad, because a working fix is measured.

## More Information

* **Relationship:**
  * applies `0020-MADR-remediate-v1-debugging-pass-findings.md` F40's
    principle to a provider that does send `tool_choice`;
  * leaves 0020 F43's `ForcedToolChoice: BestEffort` as it is;
  * found by the full live run of 2026-10-05, after
    `0022-MADR-gemini-live-instructions-test.md`.
* **Plan:** [0023-PLAN-huggingface-tool-choice-none.md](0023-PLAN-huggingface-tool-choice-none.md).
* **The probe** was a scratch test on a scratch copy of `HEAD`, never in the
  tree. It logged outcomes, error kinds, the provider header, and, once the
  402 appeared, the error text. It logged no reply.

## Amendment 2026-10-06: the router's choice of upstream

Found by 0023-PLAN's phase 2; the owner chose the resolution on 2026-10-06
("Pin to Groq; follow-up turn").

* **Corrected fact:** the Context says the router's header names `groq` for
  `openai/gpt-oss-120b`. That was the router's choice on 2026-10-05. On
  2026-10-06 it sent the model to `cerebras`, which honours `none`: tools
  and `"tool_choice":"none"` gave text 3/3. Pinned with the `:groq` suffix,
  the same request still gave `400 tool_use_failed` 3/3.
* **So the decision stands:** the router may send a request to Groq at any
  time, and Groq refuses a call made under `none`.
* **Confirmation changes:** the live check pins its `none` subtests to Groq
  (`openai/gpt-oss-120b:groq`), so it tests the refusing upstream whatever
  the router prefers that day.
* **Measured with the fix,** on 2026-10-06, three runs each:
  * case (a), `none` after a call, on Groq: text 3/3;
  * case (b), `none` on `openai/gpt-oss-20b`, served by Groq: text 3/3.

## Amendment 2026-10-06: `none` after a tool turn on Groq

Found by 0023-PLAN's phase 2; the owner chose the resolution on 2026-10-06
("Record the Groq limit").

* **Corrected claim:** the Decision Outcome says that with no tools sent
  "the model cannot call one". On Groq that does not hold after a tool
  turn. `openai/gpt-oss-120b:groq`, with no tools, `none`, and a history of
  a `get_weather` call, its result and the user's "And what about Lyon?",
  got `400 tool_use_failed` in 2 runs of 2. Without the follow-up turn it
  gave text 3/3 (0023-PLAN step 2.3).
* **Flattening was measured, and does not hold.** With the history's call
  and result sent as plain text, Groq still refused: 1 run of 3 on
  `gpt-oss-120b`, and 3 of 3 on `gpt-oss-20b` (two `tool_use_failed`, one
  `output_parse_failed`).
* **So it is a service-side limit.** Asked again for what a tool answered,
  `gpt-oss` on Groq writes a call whatever the request holds, and Groq
  refuses it under `none`. No request form the provider controls avoids it.
* **Decided:**
  * the rule stands, for what it does fix: `none` with no call history, on
    Groq, fails without it and passes with it;
  * the package comment says that on Groq, `none` after a tool turn can
    still return `ErrInvalidRequest` with `tool_use_failed`;
  * no live subtest asserts the after-a-call case.
* **Not chosen:**
  * routing `none` requests away from Groq: it overrides the caller's
    routing, and was not measured;
  * leaving the record open.
