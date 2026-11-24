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
