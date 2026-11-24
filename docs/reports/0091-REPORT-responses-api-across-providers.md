---
date: 2026-11-24
subject: which services answer the Responses API, how each one deviates from OpenAI's, and what seven rounds of live probes measured on them, for 0091-MADR's choice of a wire per provider
examines: "vendor and gateway pages read on 2026-11-23; the read-only checkouts kilocode at 58072bd696, opencode at 4ce137b39e and codex at 57e64682f8; this repository at 247b745 when the research was done and at dd37a87 where the index below is checked; about 133 live requests of 2026-11-23, seven rounds, with store false and allowlisted facts only"
---

<!-- markdownlint-disable MD013 MD024 -->

# The Responses API Across the SDK's Providers, Recorded for 0091

This report records findings. It decides nothing. The decisions it bears on
belong to
[0091-MADR-adopt-the-responses-api-per-provider.md](../decisions/0091-MADR-adopt-the-responses-api-per-provider.md),
whose probe table (PR1 to PR10) and desk-research paragraphs are read from it,
and whose implementation is
[0091-PLAN-adopt-the-responses-api-per-provider.md](../decisions/0091-PLAN-adopt-the-responses-api-per-provider.md).
The cut-at-the-limit measurements also bear on
[0088-MADR-default-output-limit-from-model-facts.md](../decisions/0088-MADR-default-output-limit-from-model-facts.md),
whose Phase 3 review rules on the rules the probes touch.

## How it was gathered

* **The desk research** (2026-11-23): one retrieval pass over vendor pages,
  gateway documentation and this repository's records, with no live call. A
  page was read through a fetch tool that relays the page through a small
  model, so a quotation below is the relayed reading, not a verified one. The
  coordinator saved the pass's report; nothing in it was run.
* **The live probes** (2026-11-23): seven rounds, about 133 requests in all,
  every request with `store` false. Rounds 1 and 2 ran against Kilo,
  OpenRouter, Together, the Hugging Face router and a local Ollama; rounds 3 to
  6 against a local Ollama 0.40.17 only; round 7 against OpenAI and xAI. Round
  1's script was `<scratch>/responses-probe/probe.py`; it is not committed.
  Its P6 body sends `include: ["reasoning.encrypted_content"]`
  (`<scratch>/responses-probe/probe.py:272`).
* **What was kept.** Only allowlisted facts: status codes, field names, enum
  values, event names, counts and booleans. No key, no account identifier, no
  host name, no response body and no model text was kept. The reasoning
  models of rounds 1 and 2 spent the small limits on reasoning, so round 1's P3
  and P4 never cut inside a tool call; round 2 closes that.
* **The checkouts' currency**: kilocode 58072bd696 (2026-11-07), opencode
  4ce137b39e (2026-11-07), codex 57e64682f8 (2026-11-07), qwen-code
  36fc21f5d8 (2026-11-22), goose 36b29113a (2026-11-20), pi 1402973a9
  (2026-10-18), grok-build 2bdd1d6a (2026-09-29), this repository 247b745
  (2026-11-23). `<gitrepos>` below names the directory that holds them.

## Desk research

### Summary

1. **Kilo gateway**: live but undocumented. Its API reference lists only
   `/chat/completions`, `/fim/completions`, `/systemone`, `/models` and
   `/providers` (<https://kilo.ai/docs/gateway/api-reference>). A 200 on
   `/responses` for `kilo-auto/free` was measured earlier, in this
   repository's 0004-MADR (E3).
2. **OpenRouter**: beta, `POST https://openrouter.ai/api/v1/responses`
   ("Responses API Beta", "may have breaking changes", read in a search of the
   docs; the overview page fetched directly shows no beta text). Stateless
   only.
3. **Together AI**: `POST https://api.together.ai/v1/responses`, "partial
   support", no beta or GA label
   (<https://docs.together.ai/docs/inference/responses-api.md>).
4. **Hugging Face router**: beta, `POST https://router.huggingface.co/v1/responses`,
   a guide titled "Responses API (beta)"
   (<https://huggingface.co/docs/inference-providers/guides/responses-api>).
5. **Ollama**: yes, stateless only, "Added in Ollama v0.13.3"
   (<https://docs.ollama.com/api/openai-compatibility>); codex gates the route
   on 0.13.4 or above (E7).
6. **Anthropic**: no; its OpenAI-compatibility layer is Chat Completions only
   (<https://platform.claude.com/docs/en/api/openai-sdk>). **Gemini**: no
   `/responses` documented (<https://aistudio.google.com/docs/openai>).
7. **OpenCode Zen and Go**: per model, from the route table
   (`@ai-sdk/openai` is Responses; 0012-MADR §3.1, E6). Zen: 77 models, 30
   Responses, 16 Messages, 7 Google, 24 Chat Completions. Go: 33 models, 6
   Responses, 3 Messages, 24 Chat Completions.

### Kilo against OpenRouter

Partly confirmed. Kilo's docs say System One "forwards each request to
OpenRouter using the platform credential". kilocode wraps OpenRouter's AI SDK
provider (E2) and names an "OpenRouter-compatible endpoint (chat, responses,
messages, models)" (E1). A legacy `/api/openrouter/` route exists and is an
alias of the gateway (E17). No source says the whole gateway is OpenRouter.

### Per service

**Kilo**

* Reasoning: the item's text is in `content[]` with type `reasoning_text` and
  `summary` empty, `format` "unknown"; OpenAI's is in `summary[]` (E4).
* kilocode's Responses sanitiser strips item ids and `item_reference` entries
  when `store` is false, and keeps `encrypted_content` (E5).
* The same model answers on Chat Completions, Responses and Messages (Kilo
  translates); one model was tried.

**OpenRouter**

* Stream events (its OpenAPI schema): `response.created`, `.in_progress`,
  `.completed`, `.incomplete`, `.failed`, `output_item.*`, `content_part.*`,
  `output_text.*`, `function_call_arguments.*`, `reasoning_text.*`,
  `reasoning_summary_*`, plus MCP, web-search, image and code-interpreter
  events; its basic-usage page lists a shorter set ending `response.done`.
* Tools: function calling, parallel calls included; `tools[].strict`,
  `parallel_tool_calls`; web search with citations.
* Reasoning: `reasoning` with `enabled`, `max_tokens`, `effort`, `summary`;
  `include` takes `reasoning.encrypted_content`.
* Usage: `input_tokens`, `input_tokens_details.cached_tokens`, `output_tokens`,
  `output_tokens_details.reasoning_tokens`, `total_tokens`, `cost`,
  `cost_details`, `is_byok`.
* `max_output_tokens` is an integer or null. Images are `input_image`
  (`image_url` or `file_id`).
* `store:true` or a non-null `previous_response_id` is a 400. A replayed
  assistant message needs `id` and `status`.
* Pages: <https://openrouter.ai/docs/api_reference/responses/overview.md>,
  <https://openrouter.ai/docs/api_reference/responses/basic-usage.md>,
  <https://openrouter.ai/docs/api/api-reference/responses/create-responses.md>.

**Together**

* Models: only `zai-org/GLM-5.2`, `GLM-5.3`, `GLM-5.3-Flash` and
  `deepseek-ai/DeepSeek-V4.1-Flash-1017`; other models "may return malformed
  output".
* Streaming, with per-item added, delta and done events, reasoning and
  function-call argument deltas included.
* Tools: function, custom, `shell`, `local_shell`, `apply_patch`;
  `parallel_tool_calls:false` is honoured. Hosted tools and remote MCP are a
  400. `web_search` is stripped; forcing it is a 400.
* Reasoning: `effort` from none to xhigh; `reasoning.summary` always empty;
  `encrypted_content` is base64 and "not encrypted".
* Usage: `input_tokens`, `output_tokens`, `total_tokens`, `cached_tokens`,
  `reasoning_tokens`.
* Images only on GLM-5.3 Flash and DeepSeek V4.1 Flash; `input_file` is a 400;
  a `file_id`-only image is silently dropped.
* `store:true` is accepted and nothing is stored; `previous_response_id` and
  `conversation` are errors; retrieve, delete, cancel, input_items,
  input_tokens and compact are 404; there is no `background`. Error `type` and
  `code` differ from OpenAI's.

**Hugging Face**

* Models: "All Inference Providers chat completion models should be
  compatible"; a suffix `:provider`, `:fastest` or `:cheapest` picks the
  backend. Streaming yes; function tools and remote MCP shown; `effort` low,
  medium or high on some open models; `input_image` with `image_url`;
  structured output through `response_format`.
* An earlier measurement of this repository: a bad key gave HTTP 200 with
  `status:"failed"`, and the docs then said "chat completion tasks only" (E8).

**Ollama**

* Streaming, tools, `reasoning.effort`, reasoning summaries, the `think`
  extension and `max_output_tokens` are listed. Unsupported:
  `previous_response_id`, `conversation`, `truncation`; "Only the non-stateful
  flavor is supported." On Ollama's cloud: no custom or freeform tool-call
  replay and no built-in web search through `/v1/responses`. Images are not
  mentioned. codex reaches Ollama on Responses only (its `ollama-chat`
  provider was removed).

**Anthropic's compatibility layer**

`strict` is ignored, thinking is not returned, `reasoning_effort` and `store`
are ignored, there is no prompt caching, and the page says it is not "long-term
or production-ready".

**OpenCode Zen and Go**

* Streaming on `/responses` was measured on Go; Zen's key was refused with a
  403, and Zen rests on Go (0031-MADR).
* A wrong-route request is an HTTP 500 (0004-MADR); `previous_response_id` is a
  400 "referenced response not found or expired"; the opencode client sends
  `store:false` (0012-MADR §3.2).
* Zen's Responses models: the gpt-5 family, gpt-6-astra, gpt-6-luna,
  gpt-6-sol, grok-4.5, grok-4.6, grok-4.7, grok-build-0.1, muse-spark-1.2 and
  muse-spark-1.3. Go's: gpt-5.6-luna, gpt-6-luna, grok-4.6, grok-4.7 and
  muse-spark-1.2 and 1.3 (contributor builds).

### Why each SDK provider is on its wire today (this repository's records)

* **kilo, Chat Completions**: Kilo's only documented route; its
  `supported_parameters` describe Chat; its `/responses` reasoning sits where
  the shared decoder, which reads `summary[]` only, would drop it; two of its
  three routes are undocumented; Kilo translates, so one route reaches every
  model (0004-MADR options 12 to 14, E9, E10).
* **huggingface, Chat Completions**: Responses undocumented, the docs said
  "chat completion tasks only", and a bad key gave HTTP 200 with
  `status:"failed"`, which the status-code classifier would turn into a silent
  empty success (0004-MADR option 8, E8). The Hugging Face guide now documents
  the route as beta; no record of this repository re-decided it before 0091.
* **together, Chat Completions**: the shared primitive and `eos` a normal stop
  (0017-MADR, E11). No record discussed Together's Responses page.
* **ollama, Chat Completions**: "OpenAI-compatible `/v1/chat/completions` for
  generation; keep the native `/api/tags` for listing" (0005-MADR, E12).
* **opencode-zen and opencode-go**: the route by each model's `provider.npm`,
  the embedded table as the fallback (0012-MADR §3.1, E6).
* **openai and grok, Responses**: 0031-MADR (E13); streaming Responses first
  (0031-MADR D8); every reachable wire measured 2026-10-10.
* **Output limit per wire**: Chat `max_tokens`, Responses `max_output_tokens`
  (0088-MADR). Truncation: `status:"incomplete"`, and `finish_reason:"length"`
  on a tool call, become `IncompleteError` (0012-MADR §1.5).

### Other agents' checkouts

opencode has both a Chat Completions route and a `/responses` route for
OpenRouter (E14, `store:false` and `include` of encrypted reasoning on the
latter) and sends Together on Chat Completions with `strict:false` (E15). pi
sends Hugging Face, Together and OpenRouter through `openai-completions`
(`<gitrepos>/pi/packages/ai/src/providers/*.ts`, at 1402973a9). codex reaches
Ollama on Responses only, from 0.13.4. goose, qwen-code and grok-build hold
only generic Responses code, with nothing per service.

## The live probes

### The probe set

Round 1 ran nine probes on each service: **P1** a plain request; **P2** a
streamed one, its event names; **P3** a cut in text at a small limit, plain
and streamed; **P4** a cut while a tool is forced; **P5** parallel calls and
`strict`; **P6** the reasoning item's keys, with
`include: ["reasoning.encrypted_content"]` sent; **P7** a bad key; **P8** a
made-up `previous_response_id`; **P9** an image.

### Round 1 (66 requests)

**Kilo** (`kilo-auto/free`)

* P1 200 `completed`; usage `input_tokens`, `input_tokens_details.cached_tokens`,
  `output_tokens`, `output_tokens_details.reasoning_tokens`, `total_tokens`,
  `cost`, `cost_details.upstream_inference_{cost,input_cost,output_cost}`,
  `is_byok`.
* P2 events: created, in_progress, output_item.added, content_part.added,
  reasoning_text.delta, reasoning_text.done, content_part.done,
  output_item.done, output_text.delta, output_text.done, completed.
* P3 plain and streamed `incomplete`, reason `max_output_tokens`; the stream
  ends `response.incomplete`. P4 `incomplete`, `max_output_tokens`, no
  function_call (the cut fell in reasoning).
* P5 two calls, `completed`, arguments parse; `strict` 200.
* P6 reasoning keys `content`, `format`, `id`, `status`, `summary`, `type`;
  `summary` empty; content `reasoning_text`; no `encrypted_content`.
* P7 401, `error.code` `INVALID_TOKEN`, top-level keys `error` and
  `error_type`. P8 400, `error` a string. P9 404.

**OpenRouter** (`openai/gpt-oss-20b`; the `:free` variant answered 404 on
every call)

* P1 200 `completed`, Kilo's usage field set. P2 the same events in another
  order, then a literal `[DONE]`.
* P3 and P4 `incomplete`, `max_output_tokens`; the stream ends
  `response.incomplete` then `[DONE]`.
* P5 one call (model-dependent), `strict` 200. P6 as Kilo's.
* P7 401 with a numeric code, top-level `error` only. P8 400 `invalid_prompt`.
  P9 404 (likely the model).

**Together** (`zai-org/GLM-5.3-Flash`)

* P1 200; usage adds `input_tokens_details.cache_write_tokens`; no cost. P2 as
  Kilo's.
* P3 and P4 `incomplete`, `max_output_tokens`. P5 two calls, `strict` 200.
* P6 reasoning keys include `encrypted_content`, present; `summary` empty;
  content `reasoning_text`.
* P7 401 `invalid_request_error` and `invalid_api_key`. P8 400
  `invalid_request_error` and `previous_response_not_found`.
* P9 200 `status:"incomplete"`; round 2 decoded its reason as
  `max_output_tokens`, the 64-token limit hit inside reasoning, so the image
  error was not what was measured.

**Hugging Face router** (`openai/gpt-oss-20b`)

* P1 200; `reasoning_tokens` 0 although reasoning items exist. P2 as Kilo's.
* P3 plain and streamed 200 `completed`, `incomplete_details` null, at
  `output_tokens` equal to the limit: a cut is not reported.
* P4 plain `completed` with no call; the stream ends `response.failed` (its
  error code was not recorded).
* P5 one call, `strict` 200. P6 no `encrypted_content`.
* P7 **HTTP 200 `status:"failed"`**, `error.code` `server_error`. P8 200
  `completed` (the made-up id ignored). P9 200 `status:"failed"`,
  `server_error`.

**Ollama** (local 0.40.17, `llama3.2:latest`)

* P1 200 `completed`, one message item. P2 events from created to completed.
* P3 200 `completed` at `output_tokens` equal to the limit: a cut is not
  reported. P5 two calls, `strict` 200. P8 200 `completed` (ignored).
* P4, P6, P7 and P9 were not run.

### Round 2: a cut inside a function call (about 15 requests)

A forced tool with a 600-word argument, at a limit too small for it.

* **Kilo** (`kilo-auto/free`, effort none, limit 100): plain 200
  **`completed`**, reason null, the function_call `completed`, its arguments
  **do not parse**, `output_tokens` 100, the limit. Streamed:
  `function_call_arguments.delta`, `.done`, `output_item.done` (the call
  `completed`, not parsing), and a terminal **`response.completed`**. At limit
  160 the model finished inside the limit, which is not a cut.
* **OpenRouter** (`meta-llama/llama-3.1-8b-instruct`, limit 160): plain 200
  **`completed`**, the call `completed`, arguments do not parse,
  `output_tokens` 160. Streamed: delta, done, `output_item.done` (`completed`,
  not parsing), `response.completed`, `[DONE]`.
* **Hugging Face** (`meta-llama/Llama-3.1-8B-Instruct`, limit 160): plain 200
  **`completed`**, the call `completed`, arguments do not parse,
  `output_tokens` 160. The stream finished inside the limit (159 tokens,
  arguments parse).
* **Together** (`zai-org/GLM-5.3-Flash`, effort none, limits 160 and 400):
  `incomplete`, `max_output_tokens`, the cut in reasoning (effort none did not
  stop reasoning). Together lists no non-reasoning model on its Responses
  route, so a cut inside a Together call is unmeasured.
* **Ollama** (`llama3.2`): a forced `tool_choice` was not honoured (a text
  message came back); unmeasured.

The finding of the round, as the probe agent put it: on Kilo, OpenRouter and
Hugging Face, a reply cut inside a function call's arguments is reported as
success (`completed` response, `completed` item, `.done` events) with
arguments that do not parse as a JSON object and `output_tokens` equal to the
limit.

### Round 3: a local Ollama 0.40.17 (about 18 requests)

* **(a) Reasoning** (`gpt-oss:20b` and `qwen3:8b`, effort low, `include`
  sent): 200 `completed`, items reasoning and message; the reasoning item's
  keys `encrypted_content`, `id`, `summary`, `type` (no `content`); `summary`
  of length 1; `encrypted_content` present; `reasoning_tokens` reported 0
  (qwen3 used all 800 output tokens).
* **(b) Images** (`gemma3:12b`, `gemma4:latest`, a 1x1 PNG): Responses 200
  `completed`; Chat Completions 200 `stop`. Status only; whether the image was
  used was not checked.
* **(c) A cut inside a call on Responses**: not obtained on `llama3.2`; the
  forced tool was honoured once and finished at 134 tokens of 160; otherwise a
  text message cut at the limit with `completed` and reason null.
* **(d) Chat Completions**, forced tool, limit 160, `llama3.2`: 200,
  `finish_reason` `length`, `completion_tokens` 160, no tool call.
* **(e) Text at a limit of 16**, `llama3.2`: Responses `completed`, reason
  null, `output_tokens` 16; **Chat Completions `length`**. Chat signals the
  cut; Responses does not.

### Rounds 4 and 5: a local Ollama 0.40.17, a cut inside a call (about 14 requests)

* `gpt-oss:20b`: every request with tools fails, on Responses and on Chat
  Completions alike (HTTP 500 `api_error`, forced or `auto`); without tools,
  Responses answers 200 `completed`. A server or model defect with tools, not
  a Responses one.
* `qwen3:8b` (limit 400) and `gemma4:latest` (limit 300): the forced tool was
  ignored; reasoning and a message; `completed`, reason null, `output_tokens`
  equal to the limit (a text cut reported as `completed`).
* `granite3-dense:8b` (limit 300): the forced tool was ignored; a message;
  `completed`, 84 tokens.
* `gemma3:12b`: HTTP 400 `invalid_request_error` with tools (the model takes
  none there).
* A cut inside a call on local Ollama remains **unobserved**: of six models
  only `llama3.2` ever returned a call, intermittently and whole. Untried:
  `deepseek-r1:7b` and the granite `q5_K_M` variant.

### Round 6: a local Ollama 0.40.17, replay and the refusal of thinking (10 requests)

* **Replay round trip** (turn 2 sends turn 1's reasoning item and message
  back; `qwen3:8b` and `gpt-oss:20b`, no tools): 200 `completed`, items
  reasoning and message, both as returned and with one character of
  `encrypted_content` changed. The server does not validate
  `encrypted_content`; whether it uses it at all is not shown, so a replay
  gain is unproven.
* **Thinking refused** on `llama3.2` (a model without thinking): Responses
  with effort high or xhigh answered **200 `completed`, no error**; the effort
  is silently ignored, and no refusal pattern matches. Chat Completions
  `reasoning_effort` high or xhigh answered **400 `invalid_request_error`**
  (type; code null), the form this repository's Ollama pattern in
  `chatcompletions.MatchRefusal` reads (E16), not the Hugging Face or Kilo
  patterns.

### Round 7: OpenAI and xAI, the Responses providers (8 POSTs and 2 listing GETs)

* **OpenAI** `gpt-4o-mini`. A text cut, plain: `incomplete`,
  `max_output_tokens`, the message item `incomplete`, with its output_text.
  Streamed: created, in_progress, output_item.added, content_part.added,
  output_text.delta, output_text.done, content_part.done, output_item.done
  (item `incomplete`), `response.incomplete`. A call cut (limit 160), plain:
  `incomplete`, the call item `incomplete`, arguments do not parse, 160
  tokens; streamed: arguments delta, arguments `.done`, `output_item.done`
  (item `incomplete`, not parsing), `response.incomplete`.
* **xAI** `grok-4.20-0309-non-reasoning`. A text cut, plain: top-level
  `incomplete`, `max_output_tokens`, but the **message item `completed`**,
  with its output_text. Streamed: created, in_progress, output_item.added,
  content_part.added, output_text.delta, then **`response.incomplete` with no
  `output_text.done`, `content_part.done` or `output_item.done`**; the final
  response's message item `incomplete`. A forced tool call: xAI did **not
  enforce `max_output_tokens`** (1042 and 861 tokens against 160),
  `completed`, arguments parse; a cut inside a call was not observed on xAI.

## Where each row of 0091-MADR's probe table is read

| MADR row | Service and model | Read in this report |
| :--- | :--- | :--- |
| PR1 | Kilo, `kilo-auto/free` | Round 1, Kilo (P1 to P9); round 2, Kilo |
| PR2 | OpenRouter, `openai/gpt-oss-20b` | Round 1, OpenRouter (P1 to P9); round 2, OpenRouter |
| PR3 | Together, `zai-org/GLM-5.3-Flash` | Round 1, Together (P1 to P9, P9's reason decoded); round 2, Together |
| PR4 | Hugging Face, `openai/gpt-oss-20b` | Round 1, Hugging Face (P1, P3, P4, P6 to P9) |
| PR5 | Ollama, local 0.40.17, `llama3.2:latest` | Round 1, Ollama (P1, P2, P3, P5, P8) |
| PR6 | a cut inside a call, round 2 | Round 2, all five services |
| PR7 | Round 3, Ollama | Round 3, (a) to (e) |
| PR8 | Rounds 4 and 5, Ollama | Rounds 4 and 5 |
| PR9 | Round 6, Ollama | Round 6, both bullets |
| PR10 | Round 7, OpenAI and xAI | Round 7, both bullets |

## What this report does not establish

* **Kilo's `/responses`**: tools, parallel calls, `strict`, streaming, cut at
  the limit, usage, `max_output_tokens`, images, `previous_response_id`, and
  any vendor statement of support were not read in the docs; the probes cover
  some of them. Whether the whole gateway sits on OpenRouter is not shown.
* **OpenRouter**: the overview page's exact beta text; the `status` enum and
  the `incomplete_details.reason` values; the effort and summary enums (its
  OpenAPI document was read to character 300,000 of 467,695).
* **Together**: cut at the limit inside a call, `max_output_tokens`, `strict`,
  the support label and its start date.
* **Hugging Face**: parallel calls and `strict` in the docs, summaries,
  encrypted reasoning, usage, `max_output_tokens`, `store`, the start date; the
  code of a stream's `response.failed` on a cut; the text of a `failed` reply's
  message after 2026-08-29.
* **Ollama**: event names in the docs, parallel calls and `strict` in the docs,
  images (round 3 shows a status only), usage; a cut inside a call (six models
  tried); whether Chat Completions reports reasoning tokens; the difference
  between v0.13.3 (the docs) and 0.13.4 (codex's floor).
* **Zen and Go**: vendor docs for tools, reasoning, usage, images and the cut
  at the limit on the Responses route.
* **Start dates**: none was found for any service's Responses route.

## Evidence index

The rows are checked by `python3 -B tools/claims.py
docs/reports/0091-REPORT-responses-api-across-providers.md` at the revisions
in the marker.

<!-- claims: self=dd37a87 kilocode=58072bd696 opencode=4ce137b39e codex=57e64682f8 -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1 | kilocode names an OpenRouter-compatible endpoint | `kilocode:packages/kilo-gateway/src/api/constants.ts:65` | `OpenRouter-compatible endpoint (chat, responses, messages, models)` |
| E2 | kilocode wraps OpenRouter's provider | `kilocode:packages/kilo-gateway/src/provider.ts:1` | `@openrouter/ai-sdk-provider` |
| E3 | Kilo's gateway translates, and `/responses` answers on it | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:454` | `**Kilo translates wire formats; it does not route by model.**` |
| E4 | Kilo's `/responses` reasoning is in a non-standard place | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:536` | `` **Kilo's `/responses` reasoning is in a non-standard place `` |
| E5 | kilocode's Responses sanitiser keeps `encrypted_content` and drops item ids | `kilocode:packages/kilo-gateway/test/responses.test.ts:5` | `strips item ids when storage is disabled` |
| E6 | OpenCode's route is derived from each model's `provider.npm` | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:382` | `` * The route for a Zen/Go model is derived from the per-model `provider.npm` in `` |
| E7 | codex's Ollama floor | `codex:codex-rs/ollama/src/lib.rs:47` | `Version::new(0, 13, 4)` |
| E8 | Hugging Face's `/responses` was found undocumented and hazardous | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:384` | `**A Responses endpoint exists but is undocumented and hazardous.**` |
| E9 | 0004 chose Kilo on Chat Completions only | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:1331` | `` ### 12. Kilo on `/chat/completions` only (chosen) `` |
| E10 | Kilo's package doc states its route | `llmprovider/providers/kilo/kilo.go:13` | `Only POST {base}/chat/completions is used. Kilo also answers /responses and` |
| E11 | 0017: `eos` is a normal stop on Together | `docs/decisions/0017-MADR-together-provider-and-auth-extensions.md:104` | `` `finish_reason: "eos"` is a normal stop `` |
| E12 | 0005: Ollama on Chat Completions | `docs/decisions/0005-MADR-canonicalize-llm-provider-configuration.md:494` | `` OpenAI-compatible `/v1/chat/completions` for generation `` |
| E13 | 0031: three providers speak the Responses wire | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:44` | `**Three providers speak the Responses wire:**` |
| E14 | opencode's OpenRouter Responses route | `opencode:packages/ai/src/providers/openrouter.ts:245` | `endpoint: Endpoint.path("/responses", { baseURL }),` |
| E15 | opencode's Together route sends no `strict` | `opencode:packages/ai/src/providers/togetherai.ts:44` | `capabilities: { tools: { strict: false } },` |
| E16 | this repository's Ollama pattern is the Chat matcher's | `llmprovider/providers/ollama/ollama.go:227` | `chatcompletions.MatchRefusal)` |
| E17 | the legacy OpenRouter-named route is an alias of Kilo's gateway | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:430` | `is an alias retained for the` |

## Addition of 2026-11-24: Upstream agents: Responses and streaming (codex, opencode, pi, goose), and ideas for the SDK and gobble

*(2026-11-24, appended.)* This section records findings. It decides nothing. An idea becomes a record only when the
owner says so. Sources:

* the read-only checkouts codex `57e64682f8`, opencode `4ce137b39e`, pi `1402973a9` and goose `36b29113a`, read
  through four retrieval hand-backs, with the spot checks listed at the end;
* this repository at `dd37a87`;
* gobble-cli at `97fa1f6`.

Path conventions:

* codex paths are under `<codex>/codex-rs/`.
* opencode: `[A]` is `<opencode>/packages/ai/src` and `[C]` is `<opencode>/packages/core/src`.
* pi paths are under `<pi>/packages/`, and `shared.ts` is `ai/src/api/openai-responses-shared.ts`.
* goose: `RS` is `<goose>/crates/oi-kit-provider-types/src/formats/openai_responses.rs`, and `PT`, `PR`, `AG` and
  `GS` are that crate root, `oi-kit-providers/src`, `oi-kit-agent/src` and `goose/src`.
* SDK paths are from this repository's root, with `wire/` for `llmprovider/internal/wire/` and `transport/` for
  `llmprovider/internal/transport/`.
* gobble paths are under `<gobble>/`.
* The qwen-code comparison is [0088-REPORT-qwen-code-output-limits-responses-streaming-and-caching.md](0088-REPORT-qwen-code-output-limits-responses-streaming-and-caching.md),
  and this section does not repeat it.

### How each does it

| Topic | codex | opencode | pi | goose | SDK | gobble |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Responses selection | the only wire; `wire_api="chat"` is a config error (`model-provider-info/src/lib.rs:110-144`) | OpenAI defaults to Responses; per-host tables (`[C]/aisdk-native.ts:59-136`) | fixed per model from models.dev (`ai/scripts/generate-models.ts`) | model-id regex plus base path (`PT/formats/openai.rs:1914-1920`; `PR/openai.rs:470-489`) | per provider: openai, grok, OpenCode per model; the rest stay on Chat on measured evidence (0091-MADR) | the SDK's (only `llm/provider` imports it) |
| Transports: WebSocket, compaction | WebSocket preferred on OpenAI, falls back to HTTP for the session; in-band `compaction_trigger` (`core/src/client.rs:1332-1340`; `core/src/compact_remote_v2_attempt.rs:228`) | WebSocket by default on OpenAI, rotated at 55 min; `/compact` and `compaction_trigger` (`[A]/protocols/utils/responses-compaction.ts:107-135`) | HTTP through the vendor SDKs; WebSocket only to the Codex backend; compaction on the client | HTTP SSE only; no compaction endpoint | HTTP SSE only; HTTP `previous_response_id` where `Continuation` is Supported (`llmprovider/providers/openai/openai.go:279`); WebSocket out of scope (0012-MADR "Out of scope") | local compaction (`compaction/`) |
| Decode: missing `.done`, ids, terminal | terminal required (`codex-api/src/sse/responses.rs:616-621`); nothing rebuilt from deltas; an unparsable item is dropped | one terminal guard for every wire; `.done` authoritative; pending calls recovered from `completed.output`; ids minted (`[A]/route/client.ts:307-331`; `[A]/protocols/open-responses.ts:1364-1394`) | terminal required; an orphan `.done` gets a slot; unfinished-call guard (`shared.ts:556-558, 785-801`) | terminal required; the final `output` replaces the done items (`RS:1300-1302, 1396-1400`) | terminal required on every wire, and an event cut by EOF is dropped (`wire/sse.go:86-91`); the terminal `output` is read only on a cut (`wire/responses/responses.go:833-834`) | n/a (the SDK's) |
| Cut and failed replies | `incomplete` → `ResponseIncomplete`, not retried; `ContentFilter` typed (`codex-api/src/sse/responses.rs:461-492`) | length, content-filter, or "unknown", which is retried (`[A]/protocols/open-responses.ts:912-922`; `[C]/session/runner/step.ts:171-179`) | only `max_output_tokens` → length; any other reason is an error (`shared.ts:804-834`) | `output_token_limit_reached`; calls not `completed` dropped (`RS:1307-1341`) | a cut → `FinishLength` with its usable part; the cut call dropped with `WarningToolCallCut`; any other reason → `ErrIncomplete` (`wire/finish.go:26-46`; `wire/responses/responses.go:741-745`) | a cut → `StopMaxTokens` with a notice, no continuation (`agent/agent.go:366-367`; 0065-MADR Q2 (1)) |
| Reasoning replay and rejection | encrypted replayed; no handler for a rejection | same provider and model only; no handler (`[C]/session/runner/to-llm-message.ts:142-174`) | same model; Azure's terminal backfill (`shared.ts:559-576`); no handler | same model; one retry with reasoning stripped (`PR/openai.rs:374-388`) | same wire format only (`wire/wire.go:42-44`); no rejection handling (searched `llmprovider/` for `could not be verified` and `invalid_encrypted`) | same provider and model (`llm/provider/replay.go:27-60`); one resend without thinking on a 400 naming a signature or encrypted content (`replay.go:118-140`; `provider.go:352-359`) |
| Tool-call assembly and release | from `output_item.done` only, dispatched at once (`core/src/stream_events_utils.rs:359-405`) | raw text accumulated, partial JSON parsed lazily; tools start mid-stream (`[A]/protocols/utils/tool-stream.ts:63-99`; `[C]/session/runner/step.ts:119-131`) | partial JSON re-parsed on every delta (`shared.ts:680-696`) | Responses argument deltas ignored; malformed arguments become `{}` (`RS:980-989, 1343-1350`) | text accumulated in a builder; a call is released when its item ends, and held while a cut could drop it (`wire/finish.go:59-111`) | calls collected; run after the stream ends (`agent/agent.go:738-744, 342-347`) |
| Timeouts | idle 300 s per provider; no total; WebSocket connect 15 s (`model-provider-info/src/lib.rs:73-78`) | header 300 s, chunk 900 s, set per request (`[A]/route/transport/http.ts:97-137`) | undici body and header timeouts 300 s, configurable (`coding-agent/src/core/http-dispatcher.ts:4-14`) | connect 30 s, read 600 s; Ollama 120 s per line (`PR/api_client.rs:19-20`) | connect 30 s, header 300 s, idle 300 s fixed, no total (`transport/transport.go:28, 80-88`; `transport/reply.go:14`) | the SDK's; offline wait 5 min (`llm/llm.go:42`) |
| Retries before and after the first event | transport (not 429) plus sampling, which also covers failures after output; history rebuilt (`core/src/responses_retry.rs:58-177`) | re-run before output; after output a synthetic "Continue" prompt (`[C]/session/runner/step.ts:211-215, 272-280`) | none in the provider by default; the agent reruns the whole turn and keeps the failed attempt out of later requests (`coding-agent/src/core/agent-session.ts:1241-1257`) | before the first item only; after it the turn ends and the partial is kept (`GS/agents/reply_parts.rs:322-381`) | before the first event only (`llmprovider/retry.go:234-296`); an after-reply failure resent once (`retry.go:175-181`) | the SDK's, 3 retries from 2 s to 60 s (`llm/llm.go:42`; `llm/provider/provider.go:254-261`); after text is shown the turn ends with the error (`agent/agent.go:514-517`) |
| Cancellation | a biased select drops the stream; WebSocket `response.interrupt` (`codex-api/src/endpoint/responses_websocket.rs:824-913`) | fiber interrupt; final events uninterruptible (`[C]/session/runner/step.ts:134-153`) | one AbortController; the aborted message is kept, not replayed | select drop; partials saved; synthetic tool results (`AG/machine.rs:196-321`) | the request's context comes from the caller's; stopping the range closes the body (`wire/post.go:100-118`) | the partial reply kept and marked failed; calls not run are answered (`agent/agent.go:337-340`; `acpserver/choice.go:240-256`) |
| Backpressure and batching | bounded at 1600 per stage; adaptive TUI chunking (`tui/src/streaming/chunking.rs:86-116`) | deltas batched per 100 ms; WebSocket queue unbounded (`[C]/session/runner/publish-llm-event.ts:169-203`) | unbounded push queue with awaited emits; renders 16 ms apart | unbounded mpsc; a Markdown buffer | a pull iterator: `yield` runs inside the read loop (`wire/sse.go:171-179`) | ACP chunks at most every 50 ms or 4 KiB (`acpserver/coalesce.go:11-14`) |
| Caching and prefix stability | `prompt_cache_key` plus a TTL; content-hash ids; a frozen synthetic-id namespace (`core/src/client.rs:1191-1224`) | 4 breakpoints placed automatically; effort changes as `configuration_update` items (`[A]/cache-policy.ts:215-247`) | 3 or 4 Anthropic breakpoints; a placeholder tool; 24 h retention; miss detector and warmer (`coding-agent/src/core/cache-warmer.ts:12-76`) | caching rules declared per provider and model; tools sorted; a volatile `turn_context` message; prefix-invariance tests with seeded regressions (`PT/cache_semantics.rs`; `<goose>/crates/oi-kit-provider-types/tests/prefix_invariance.rs:389-428`) | opt-in; at most 2 Anthropic markers; the cache key only when asked; no retention field; the caller's tool order; no byte test across turns (`wire/messages/cache.go:24-54`; `wire/cache.go:8-15`) | always asks for caching; tools in config order with loads appended (`agent/agent.go:241-282`); a prefix test on resume (`acpserver/resume_test.go:197`) |
| Error taxonomy | typed, with one terminal-or-retryable match (`protocol/src/error.rs:430-492`) | ordered rules plus about 35 regexes; delivery state on transport errors (`[A]/schema/errors.ts:107-120`) | message regexes (`ai/src/utils/retry.ts:7-130`; `ai/src/utils/overflow.ts:37-175`) | 14 variants, each with a telemetry type (`PT/errors.rs:8-85`) | kinds read from status, code and type; `x-should-retry` honoured (`llmprovider/provider.go:79-84`; `llmprovider/api_error.go:33-112, 358`) | the SDK's kinds in the auth contract's words (`llm/kind.go`) |
| Telemetry | OTEL ttft, sse_event, retry and WebSocket continuation metrics (`otel/src/metrics/names.rs:15-47`) | `first_output_ms` span; a WebSocket event counter | diagnostics per message; no stream-health metrics | `gen_ai` spans; TTFT (`GS/agents/reply_parts.rs:146-162`) | `RetryNotice` (0044-MADR); usage fields; nothing per stream | TTFT under `--stats` only (`internal/cli/render/status.go:43-44`); `telemetry` is a placeholder (`telemetry/doc.go`) |
| Tests | SSE unit tests; no stream benchmarks | recorded cassettes per provider, WebSocket included (`packages/http-recorder`) | terminal-event and partial-JSON tests | a 12-case stream-failure matrix; prefix invariance | goldens, fake servers, stream-error tests per wire (`stream_error_0072_test.go`); stream benchmarks for Responses, Chat and generateContent, none for Messages | fake-server wire tests (`llm/provider/wire_sse_test.go`); ACP golden transcripts (`acpclient/acptest`) |

### Ideas for go-llmprovider-sdk

#### Where the SDK already does as well or better (no idea needed)

* **Terminal-event guard on every wire.** One shared reader requires each wire's last event: `response.completed`,
  `message_stop`, `[DONE]`, a finishReason and `interaction.completed`. An event the stream ends inside is dropped
  (`wire/sse.go:86-91`; 0069-MADR). This matches opencode's `requireTerminalEvent`.
* **`x-should-retry`** is honoured (`llmprovider/api_error.go:358`; 0021-MADR T14).
* **Server-delay cap.** A wait longer than `MaxDelay` returns the error at once (`llmprovider/retry.go:27-29, 310-313`),
  as pi does (`ai/src/utils/provider-retry.ts:37-51`). opencode instead takes Retry-After as a floor, capped at 15 min.
* **Per-event decode on Responses is one pass.** A byte scan reads the event type, unused events are skipped, and a
  delta is decoded into a two-field struct (`wire/responses/responses.go:711-722, 900-918`). codex and goose decode
  into `Value`, then decode again.
* **Backpressure.** The pull iterator stops the read when the consumer is slow. The SDK has none of the unbounded
  queues codex, goose and pi carry.
* **Cut calls.** Arguments are accumulated once and checked whole (`wire/finish.go:136-141`), never re-parsed per delta.

#### Harden

* **H1. Fill a `completed` Responses stream from its terminal `output`.**
  * *Upstream:* opencode recovers pending calls from `response.completed.output` (`[A]/protocols/open-responses.ts:1364-1394`,
    spot-checked: "Some compatible providers omit output_item.done"). goose takes the final `output` (`RS:1300-1302`).
    pi gives an orphan `.done` a slot (`shared.ts:556-558`).
  * *SDK today:* `takeTerminal` runs only on a cut (`wire/responses/responses.go:833-834`). A `completed` stream with no
    `output_item.done` reaches `completed()` with no output and fails `ErrEmptyAnswer`, after its text deltas were
    already yielded (`:440-445`).
  * *Change:* on `completed` as on a cut, add each terminal item no `.done` delivered. Missing ids are matched by
    position, as `delivered` already does (`:789-794`). Held calls are released as today.
  * *Record:* amend 0088 (the review of R2 and R3 rules on the same code). *Cost:* S; risk low.
  * *Priority:* **P1**, because 0091 Phase 2's `chatcompat.WithResponses` sends to arbitrary servers, the shape qwen-code
    calls a "non-compliant proxy".
* **H2. The idle limit should time the service, not the consumer.**
  * *Upstream:* qwen-code counts only upstream waiting (0088-REPORT on qwen-code).
  * *SDK today:* the timer resets on body reads only (`transport/reply.go:105-112`), and `yield` runs inside the read
    loop (`wire/sse.go:171-179`). A consumer that holds an event for 300 s gets `ErrIdleTimeout` from a healthy
    stream. This is not documented.
  * *Change:* stop the timer while `yield` runs, and restart it after.
  * *Record:* amend 0021-MADR D2. *Cost:* S; risk low.
  * *Priority:* **P2**, a silent failure mode for any caller that does work per event.
* **H3. A typed refusal of replayed reasoning.**
  * *Upstream:* goose resends once with the reasoning items stripped, and excludes that error from its generic retries
    (`PR/openai.rs:374-388`, spot-checked; `PT/retry.rs:94-107`). qwen-code downgrades the items once. codex, opencode
    and pi have no handler.
  * *SDK today:* none (searched `llmprovider/` for `could not be verified` and `invalid_encrypted`). Replay is scoped
    by wire format alone (`wire/wire.go:42-44`), so OpenAI reasoning goes to another model, or to Grok, unfiltered.
    gobble covers this for itself with a text match (`llm/provider/replay.go:118-140`).
  * *Change:* classify the four vendors' refusals as `ErrReplayRefused`, a child of `ErrInvalidRequest`. Optionally,
    resend once without reasoning items inside `MaxSends`, before any event (`wire/sse.go:155-165`).
  * *Record:* new, amending 0021-MADR W7 and 0020 F24. *Cost:* M; risk: matching vendor wording, so it needs recorded
    fixtures.
  * *Priority:* **P2**: gobble is covered, and any other caller fails hard.
* **H4. Delivery state on errors.**
  * *Upstream:* opencode tags transport errors `not-sent`, `rejected`, `ambiguous` or `accepted`, and decides retry or
    continue from that (`[A]/schema/errors.ts:107-120`, spot-checked).
  * *SDK today:* it already knows the state internally:
    * the `WroteRequest` trace (`wire/post.go:139`);
    * `AfterReply` (`:168`; `transport/reply.go:30-53`);
    * `AfterEvent` (`transport/reply.go:55-77`);
    * `ErrOffline` (`wire/post.go:170`).

    But `internal/transport` cannot be imported, so a caller sees only kinds. gobble rebuilds the state as its own
    `shown` flag (`agent/agent.go:514-517`).
  * *Change:* export `llmprovider.DeliveryOf(err)`, returning not sent, sent with no reply, or streaming (after the
    first event).
  * *Record:* new, citing 0021-MADR D1, 0028-MADR D-H1 and 0088-MADR ST88a. *Cost:* S to M (one export, apidiff).
  * *Priority:* **P2**. It also states what 0091-MADR ST45 says about billing: a `failed` HTTP 200 reply is resent.
* **H5. A Chat stream that ends after `finish_reason` with no `[DONE]`.**
  * *Upstream:* opencode drops `[DONE]` in framing and ends on the finish event (`[A]/route/framing.ts:51-62`).
  * *SDK today:* it is `ErrProviderUnavailable` marked after-reply, so it is resent once and may be billed twice
    (`wire/chatcompletions/chatcompletions.go:663`; `wire/sse.go:90-91`).
  * *Change:* none until a service is measured doing it. Then accept a stream that has `finish_reason` and usage.
  * *Record:* amend 0021-MADR D1. *Cost:* S. *Priority:* **P3** (unmeasured).

#### Improve

* **I1. Settings for stream idle and response-header limits.**
  * *Upstream:* opencode sets header and chunk timeouts per request or model, 300 s and 900 s, with "DeepSeek asked
    15 min". codex sets `stream_idle_timeout_ms` per provider. pi offers a choice of 30 s to 5 min.
  * *SDK today:* idle is fixed at 300 s and settable only by tests (`wire/post.go:20-28`); the header limit is fixed at
    300 s (`transport/transport.go:82`); there is no lifetime cap, by design (the caller's context,
    `transport/transport.go:36-40`).
  * *Change:* a provider option for the idle and header limits. The lifetime stays with the caller's context, and the
    docs say so.
  * *Record:* amend 0021-MADR D2. *Cost:* S; risk low.
  * *Priority:* **P2**: a model that reasons silently for more than 300 s has no setting.
* **I2. Gaps in the error taxonomy.**
  * *Upstream:* codex has `Timeout`, `ContentFilter` and `ModelNotFound`; goose has `Refusal{category}` and
    `EndpointNotFound`; opencode has `Timeout` and `PayloadTooLarge`.
  * *SDK today:*
    * the idle timeout is unexported (`transport/reply.go:24-25`) and surfaces as `ErrProviderUnavailable`;
    * a content filter is `ErrEmptyAnswer` or `ErrIncomplete` with a `Reason` string;
    * there is no not-found-model kind (searched `llmprovider/api_error.go`).
  * *Change:* add `ErrTimeout` (a child of `ErrProviderUnavailable`) and `ErrContentFiltered` (a child of
    `ErrIncomplete`). Measure a not-found model first.
  * *Record:* amend 0015-MADR D7. *Cost:* S each. *Priority:* **P3**.
* **I3. Retry-After as an absolute instant for every form.**
  * *Upstream:* codex parses Retry-After once into an `Instant` (`<codex>/codex-rs/http-client/src/retry_after.rs:11-42`,
    spot-checked).
  * *SDK today:* `ResetAt` is kept only for an HTTP-date; delta-seconds and `retry-after-ms` name no instant
    (`transport/transport.go:249-272`; 0072-MADR D1). A caller that reschedules later restarts the clock.
  * *Change:* also stamp the reception time plus the delay.
  * *Record:* amend 0072-MADR D1. *Cost:* S.
  * *Priority:* **P3**: `WithRetry` waits at once, and gobble maps the error at once (`llm/provider/provider.go:305, 410`).
* **I4. OpenAI cache retention under `CacheLong`.**
  * *Upstream:* pi sends `prompt_cache_retention:"24h"` (`ai/src/api/openai-responses.ts:117-124`), codex sends
    `prompt_cache_options.ttl` (`core/src/client.rs:1307-1310`), and opencode sends both.
  * *SDK today:* not sent (0039-MADR D3, D6).
  * *Change:* map `CacheLong` to the retention field where a model's facts allow it.
  * *Record:* amend 0039-MADR D3. *Cost:* S; risk: per-model support. *Priority:* **P2**.
* **I5. An Anthropic breakpoint inside the conversation.**
  * *Upstream:* goose adds a second anchor about 20 blocks behind the tail for lookback (`PT/cache_semantics.rs:55-107`).
    pi, opencode and qwen-code mark the last stable user block.
  * *SDK today:* the top-level automatic marker plus one at the end of the fixed prefix (`wire/messages/cache.go:24-54`).
    0039-MADR D6 rejected breakpoints placed by the caller.
  * *Change:* one SDK-placed anchor, only when more blocks than Anthropic's lookback were added since the last marker.
  * *Record:* amend 0039-MADR D2. *Cost:* M.
  * *Priority:* **P3**: measure misses first (gobble's G4 would show them).

#### Optimise

* **O1. A test that the request prefix stays byte-stable across turns, with seeded regressions.**
  * *Upstream:* goose checks consecutive requests: an append-only prefix for implicit caches, and stable bytes up to
    the last breakpoint for explicit ones. Its seeded regressions must fail (`<goose>/crates/oi-kit-provider-types/tests/prefix_invariance.rs:389-428`,
    spot-checked).
  * *SDK today:* no such test (0088-REPORT on qwen-code, gap 5).
  * *Change:* per wire, encode turns N and N+1 of one conversation and assert the prefix relation. Plants: a reordered
    tool, a moved marker, an effort change mid-history (0062-MADR).
  * *Record:* new, touching 0039. *Cost:* S; tests only. *Priority:* **P1**: cheap, and it guards the main cost lever.
* **O2. Benchmarks for every stream path before any decode change.**
  * *SDK today:* benchmarks exist for Responses `ReadStream`, Chat `reasoning_details` and generateContent text parts.
    There is none for `Events` with deltas and none for Messages, which decodes every delta into the whole
    `streamEvent` (`wire/messages/messages.go:476-479`).
  * *Change:* `BenchmarkEvents_*` per wire, at 10k deltas (allocs and B/op). Only then consider a fast path for
    Messages deltas, like the Responses one.
  * *Record:* amend 0028-MADR (benchmarks). *Cost:* S. *Priority:* **P3**.

#### Modernise

* **M1. A Responses WebSocket transport with safe continuation (a candidate).**
  * *Upstream:*
    * codex prefers it on OpenAI, and sends a delta only when an exhaustive comparison of the request's fields matches;
      otherwise it sends everything (`core/src/client.rs:429-500, 1675-1736`);
    * codex, opencode and pi resend the full input on `previous_response_not_found` or
      `websocket_connection_limit_reached` (codex: `codex-api/src/endpoint/responses_websocket.rs:733-750`; opencode:
      `[A]/protocols/open-responses-continuation.ts:161-183`; pi: its Codex provider in `ai/src/api`, `:346-353`);
    * all three fall back to HTTP for the session.
  * *SDK today:* absent. Out of scope in 0012-MADR ("The Codex WebSocket transport and its `previous_response_id`
    chaining"). `go.mod` has no WebSocket package.
  * *Change:* an opt-in transport for OpenAI only. Continuation is checkpointed after the terminal event, any rejection
    resends in full, the socket is rotated before 60 min, and HTTP takes over for the session after failures.
  * *Record:* new. *Cost:* L; risk high: a new dependency or a hand-written RFC 6455 client, and state across
    iterators.
  * *Priority:* **P3**: the gains (upload size, time to first token on long sessions) are unmeasured here. Measure
    first.
* **M2. Recorded stream cassettes.**
  * *Upstream:* opencode records cassettes per provider, WebSocket included (`packages/http-recorder`). goose has a
    12-case failure matrix (`<goose>/crates/oi-kit-provider-types/tests/responses_stream_failures.rs`).
  * *SDK today:* fake servers, goldens and recorded refusal fixtures (`wire/testdata/refusals-0061`, `-0062`), but no
    recorded stream transcripts. 0091-PLAN Phase 2 hand-writes the shapes from rounds 3 and 7.
  * *Change:* a `tools/` recorder that keeps allowlisted, redacted SSE transcripts from live runs, and a replay
    harness per wire.
  * *Record:* new, under 0065-MADR (tests reach no outside host). *Cost:* M. *Priority:* **P2**: every decode idea
    above needs real shapes.
* **M3. Effort changes without invalidating the cache.**
  * *Upstream:* pi and opencode send a `configuration_update` item on GPT-6-class models (`shared.ts:133-143`).
  * *SDK today:* an effort learned from a refusal changes the request's top-level effort (0062-MADR).
  * *Change:* none until a provider in the SDK documents it.
  * *Record:* amend 0062. *Cost:* S. *Priority:* **P3**.

Decided no: encoding the body once for reuse across resends, as codex does (`<codex>/codex-rs/http-client/src/request.rs:12-39`).
The SDK rebuilds each resend from what the retry learned (`wire/retry.go:24-27`), so only the after-reply resend would
reuse identical bytes.

### Ideas for gobble-cli (the harness)

#### Where gobble already does as well or better

* **Orphaned-call repair:** "No result provided", Pi's words (`llm/provider/replay.go:10, 27-45`).
* **Model-scoped replay, with one resend after a refusal of a signature or of encrypted reasoning**
  (`replay.go:27-60, 118-140`).
* **Cancellation persistence.** The partial reply is kept and marked failed, so it is not replayed; calls not run get
  results (`agent/agent.go:337-340, 1028-1036`; `acpserver/choice.go:240-256`). This matches goose's
  `finalize_cancellation` and pi.
* **Delta batching** at the ACP edge, at most every 50 ms or 4 KiB (`acpserver/coalesce.go:11-14`). The CLI reads
  through `acpclient`.
* **A streak limit after length cuts is moot.** The owner ruled "Stop with a notice (Recommended)" (0065-MADR, fifth
  note: "there is no streak").

#### Ideas

* **G1. Keep a reply cut by a failure after it showed text.**
  * *Upstream:*
    * pi keeps the failed attempt, keeps it out of later requests, and reruns the turn (`coding-agent/src/core/agent-session.ts:1241-1257`);
    * opencode continues with "Continue from where you left off" (`[C]/session/runner/step.ts:272-280`);
    * goose keeps the partial and ends the turn (`AG/inference.rs:433-456`).
  * *gobble today:* the turn ends with the error (`agent/agent.go:514-517, 571-575`). No `agent.Message` is emitted, so
    the session file never holds the shown text (`acpserver/agent.go:1048-1050`), and a reload loses what the user
    saw.
  * *Change:* write the partial as a failed message, not replayed, using the SDK's H4 to tell a transport cut from a
    refusal.
  * *Open question:* whether to continue on a transport cut (pi or opencode style). 0065-MADR Q2 answered this for
    length cuts only.
  * *Record:* new, citing 0031-MADR D3. *Cost:* S; continuing is M. *Priority:* **P1** for keeping the partial.
* **G2. Retry budget per turn.**
  * *gobble today:*
    * the SDK's 3 retries apply to each call (`llm/llm.go:42`);
    * the offline wait is 5 min (`agent/agent.go:533-547`);
    * 2 more runs follow an empty reply (`:439, 548-567`);
    * a turn may make up to 100 calls (`:24`);
    * nothing bounds the turn's total retry time.
  * *Upstream:* opencode has a separate 1 h budget for unreachable hosts and caps timeout retries at 3
    (`[C]/model-retry.ts:43-86`).
  * *Change:* a total wait per turn, with idle timeouts counted separately.
  * *Record:* new. *Cost:* S. *Priority:* **P3**.
* **G3. Run a reply's tool calls in parallel.**
  * *Upstream:* codex uses a read-write gate per tool (`core/src/tools/parallel.rs:183-300`); pi runs calls in parallel
    unless a tool is sequential (`agent/src/agent-loop.ts:512-525`); opencode starts a tool while the stream continues.
  * *gobble today:* calls run one after another (`agent/agent.go:761-764`).
  * *Change:* run read-only tools concurrently, bounded, with results kept in call order and permission questions
    asked one at a time.
  * *Record:* new. *Cost:* M; risk: ordering of tool output in ACP. *Priority:* **P2** (latency on read-heavy turns).
* **G4. Report cache misses, then consider warming.**
  * *Upstream:* pi's detector counts missed tokens as the lesser of the previous and current prompts, less the cache
    read, with a 1,024-token floor and the dollar cost of the miss. Its warmer sends a one-token request only when the
    expected saving is at least $0.05 (`coding-agent/src/core/cache-stats.ts`; `cache-warmer.ts:12-76`).
  * *gobble today:* it shows reads and writes (0057-MADR); warming was deferred until the catalog priced it (0021-MADR
    D7, quoted in 0057).
  * *Change:* a miss notice first.
  * *Record:* amend 0057. *Cost:* S; warming is M. *Priority:* **P2**.
* **G5. Test the prefix across consecutive turns, and measure what the tool list costs.**
  * *gobble today:*
    * `TestResumeKeepsPrefix` covers a resume only (`acpserver/resume_test.go:197`);
    * MCP tools that connect late "join a later prompt" (`acpserver/mcp.go:171`), in the server's own order
      (`mcpclient/manager.go:76-81`), which changes the tools at the head of the prefix mid-session.
  * *Change:* goose-style checks on consecutive calls per wire, with seeded plants (relocated context, reordered
    tools). Log a tool-list change.
  * *Record:* amend 0037-MADR (its prefix amendment). *Cost:* S. *Priority:* **P1**.
* **G6. Second overflow recovery.**
  * *gobble today:* it compacts once and resends once (`agent/agent.go:518-531`).
  * *Upstream:* opencode shrinks at 85, 70, 50 and 35% (`[C]/session/compaction.ts:602-613`); goose allows 2
    compactions per turn.
  * *Change:* one more, harder compaction before failing.
  * *Record:* amend 0005-PLAN F5-2. *Cost:* S. *Priority:* **P3**.
* **G7. Subagent limits.**
  * *gobble today:* not applicable, since gobble has no subagent tool (searched `tool/` for "subagent").
  * *For later:* if one is added, codex's 6 threads at depth 1 and goose's 5 concurrent subagents are the references,
    as is sharing the parent's cache key (`<codex>/codex-rs/core/src/client.rs:720-732`). *Priority:* **P3**.
* **G8. Stream-health telemetry.**
  * *Upstream:* codex records ttft, SSE event, retry and continuation metrics (`otel/src/metrics/names.rs:15-47`);
    opencode records `first_output_ms`.
  * *gobble today:* `telemetry` is a placeholder, and TTFT shows only under `--stats`.
  * *Change:* one debug record per call: TTFT, longest gap between events, event count, retries and their causes,
    delivery state, cache reads, writes and misses. OTEL later.
  * *Record:* joins 0065-MADR D11 and 0066-MADR D8. *Cost:* S. *Priority:* **P2**.
* **G9. Adaptive catch-up.**
  * *Upstream:* codex has a smooth mode and a catch-up mode with hysteresis on queue depth and age
    (`tui/src/streaming/chunking.rs:86-116`).
  * *Change:* only if a backlog behind the 50 ms coalescer is measured. *Priority:* **P3**.

### What not to copy

* **Error typing by message regex.**
  * *Who:* pi (`ai/src/utils/retry.ts:7-130`; `ai/src/utils/overflow.ts:37-175`), opencode's ~35 overflow patterns
    (`[A]/provider-error.ts:16-70`), and codex's "try again in" regex.
  * *Why not:* wording drifts. The SDK reads status, code and type against recorded markers (0084-MADR).
  * gobble's `signatureMarks` (`llm/provider/replay.go:122`) should go once H3 exists.
* **Partial JSON re-parsed on every delta, O(n²) per call** (pi, `shared.ts:680-685`). The SDK accumulates and parses
  once.
* **Unbounded queues.**
  * *Who:* codex's WebSocket inbound channel (`codex-api/src/endpoint/responses_websocket.rs:72-129`), goose's event
    mpsc (`AG/operation.rs:273-291`), pi's `EventStream`, and opencode's WebSocket queue.
  * *Why not:* the SDK's pull iterator already gives backpressure.
* **Silent drops.** codex drops an item that will not parse and ignores unknown `error` events
  (`codex-api/src/sse/responses.rs:379-385`). goose turns malformed arguments into `{}` (`RS:980-989`). The SDK fails
  or names the drop (`WarningToolCallCut`).
* **Retrying unknown failures, or retrying after output.** opencode retries `UnknownProvider` deliberately
  (`[A]/schema/errors.ts:207-217`), and codex's sampling retry covers stream failures after output. Both risk double
  billing. The SDK resends an after-reply failure once and never after an event (`llmprovider/retry.go:175-181`).
* **Routing by model-name regex** (goose, `PT/formats/openai.rs:1914-1920`). The SDK routes by catalog facts
  (0041-MADR D1).
* **Passing unknown request parameters through verbatim** (goose, `RS:906-930`). The SDK has a typed `Request`.
* **Per-event immutable state spreads** (opencode, `[A]/protocols/utils/lifecycle.ts:17-22`). They allocate on every
  delta.
* **A canned "continue" prompt by default.** The owner ruled against it for length cuts (0065-MADR Q2 (1)). For
  transport cuts it is G1's question.

### Suggested order

| Rank | SDK idea | Joins | gobble idea | Joins |
| :--- | :--- | :--- | :--- | :--- |
| 1 | H1 fill `completed` from its terminal output | 0088 Phase 3 (R2, R3 rule on the same code), then 0091 Phase 2 (ST52, chatcompat's path) | G1 keep a partial reply | new record next to 0065 Q2 and 0031 D3 |
| 2 | O1 prefix byte-stability test with plants | new record touching 0039; before I4 and I5 | G5 consecutive-turn prefix test, MCP late join measured | 0037 prefix amendment |
| 3 | H2 idle times the service; I1 idle and header settings | one new record amending 0021 D2 | G8 stream-health log | 0065 D11, 0066 D8 |
| 4 | H4 delivery state exported | new record; informs 0091 ST45 (out of scope today) and gobble G1 | G4 cache-miss notices | 0057 |
| 5 | H3 typed replay refusal | new record amending 0021 W7; gobble drops `signatureMarks` | G3 parallel read-only tools | new record |

None of these changes 0088's R1 (the cut rule for every wire), R4 (`ChooseLimit`'s fallback cap) or R5 (the Responses
minimum of 16). 0091 Phase 2 already plans `reasoning_text` deltas (ST10, ST63), which the table's reasoning row
leaves out. gobble 0065 and 0066 decide output limits and cut notices, and none of the ideas above repeats them.

### How this section was checked

Four read-only spot checks were made in the upstream checkouts: opencode's
recovery from `completed.output` (`open-responses.ts:1364-1394`) and its
delivery states (`errors.ts:107-120`); goose's stripped-reasoning retry
(`openai.rs:374-388`) and its seeded prefix regressions
(`prefix_invariance.rs:389-428`); and codex's `RetryAfter(Instant)` and its
WebSocket error codes. Every SDK and gobble claim was read at `dd37a87` and
`97fa1f6`; the SDK facts cited from the qwen-code report still hold, since the
only commit since `46f43d7` is docs. The coordinator then verified H1 at
`llmprovider/internal/wire/responses/responses.go:833-834`, H2 at
`llmprovider/internal/transport/reply.go:110-112`, and G1 at gobble
`agent/agent.go:514-517`.

Could not verify: whether any Chat server omits `[DONE]` (H5); whether
go-tui-lib's stream renderer throttles; whether MCP servers keep their tool
order across restarts; the WebSocket gains (M1); and whether gobble's
shown-but-failed text is lost on reload. G1's claim rests on reading
`agent.go` and `acpserver/agent.go`; it was not run.
