---
status: accepted
date: 2026-11-27
decision-makers: repository owner
consulted: "the owner's answers of 2026-11-23 to OQ1 to OQ8, relayed by the coordinator, their labels quoted in the amendment; the owner's question of 2026-11-23 and the option the owner chose, 'Research and adopt in v1.33.0', relayed by the coordinator; the coordinator's desk research of 2026-11-23 (vendor pages read that day, no live calls); the probe agent's seven rounds of live probes of 2026-11-23 (about 133 requests, store false, allowlisted facts only; rounds 3 to 6 on a local Ollama 0.40.17, round 7 on OpenAI and xAI); an independent review of the first draft (2026-11-23, NOT READY, findings F1 to F13), relayed by the coordinator; this repository at 247b745, read only, with the uncommitted 0088-PLAN Phase 1b work in the tree read where named; the read-only upstream checkouts codex at 57e64682f8 and kilocode at 58072bd696"
informed: every caller of the kilo, together, huggingface, ollama and chatcompat providers; the callers of openai, grok and OpenCode's responses route, whose decoder this record touches; gobble-cli first, which runs agent turns on Kilo
---

<!-- markdownlint-disable MD013 MD024 -->

# Adopt the Responses API Per Provider Where It Loses Nothing Chat Completions Gives, and Read a Cut a Service Reports as Complete

Associated PLAN: [0091-PLAN-adopt-the-responses-api-per-provider.md](0091-PLAN-adopt-the-responses-api-per-provider.md)

*(2026-11-23, the owner's answers: OQ1 to OQ8 answered; every row they reach is decided, and what OQ5's opt-in needs is decided as ST50 to ST62. One new question, OQ9, raised by OQ5's and OQ8's answers together, leaves ST63 open, so the record stays `proposed`. See "Amendment (2026-11-23): the owner's answers to OQ1 to OQ8" at the end.)* *(2026-11-23, second: the review of that amendment found ST55's premise false, OQ5's losses never put to the owner, and ST51 wrong about the Chat path's facts; ST55 is reopened as OQ10, OQ5's opt-in is put back as the confirming OQ11 (ST64), and ST51 and ST62 are corrected. Three rows are open. See "Amendment (2026-11-23, second)" at the end.)* *(2026-11-23, third: the owner answered OQ9 to OQ11, deciding ST55, ST63 and ST64; no row is open, and the record is `accepted`. See "Amendment (2026-11-23, third)" at the end.)* *(2026-11-24, fourth: PLAN Phase 0 placed the records and the REPORT in the tree and re-grounded the evidence index at `dd37a87`; no row changes. See "Amendment (2026-11-24, fourth)" at the end.)* *(2026-11-24, fifth: PLAN Phase 1 carried out ST1 to ST4 and ST28 in the four providers' package docs and `docs/architecture.md`, with twelve claims rows; no row changes. See "Amendment (2026-11-24, fifth)" at the end. After a review, pass 2 restated the reasons by their sources and the claims rows as M1 to M27: see "Annotation (2026-11-24, after the review of Phase 1)". Pass 3 (the owner's "Pass 3 with full rows") pinned every new doc line, as F1 to F41: see "Annotation (2026-11-24, after the third pass)".)* *(2026-11-25, seventh: 0092-MADR, built, makes a completed Responses stream take its terminal items and fail on an announced item never delivered; how ST55 meets that is put to the owner as OQ12, ST65 is open, and the record is `proposed`. See "Amendment (2026-11-25, seventh)" at the end.)* *(2026-11-25, eighth: OQ12 answered, "The cut rule wins (Recommended)"; ST65 is decided, no row is open, and the record is `accepted` again. See "Amendment (2026-11-25, eighth)" at the end.)* *(2026-11-26, ninth: PLAN Phase 2's re-ground found four cases no row decided; two owner answers and two coordinator calls decide them as ST66 to ST69, no row is open, and the record stays `accepted`. See "Amendment (2026-11-26, ninth)" at the end.)* *(2026-11-26, tenth: after the code review of PLAN Phase 2, the owner's "The refusal wins (Recommended)" and "Raise the bound" decide ST70 and ST71; no row is open. See "Amendment (2026-11-26, tenth)" at the end.)*

*(2026-11-23, revision 1, before any answer: the first draft was reviewed NOT READY. This revision states the cut check's reach as the evidence supports it (F2), decides where the check runs and how the limit sent reaches it, with its dependency on 0088's R3 (F4), lists every Chat-path feature Ollama's move must keep (F5), splits the cut question into which signal (OQ6) and where (OQ7) (F8), corrects Kilo's and Hugging Face's gate wording (F1) and the `include` fact (F3), adds Hugging Face's stream cut (F6), a gains column (F9), a reasoning-text replay question (OQ8, F10), conditional and plant wording (F11, F12), and plain-worded questions (F13). It folds in Ollama probe rounds 3 to 5. The first draft's rows keep their numbers; ST33 to ST46 are new.)* *(2026-11-23, revision 2, before any answer: after the delta review. Ollama's reasoning-replay gain is conditional on a round-trip probe, ST47 (N1); the gate rows ST22, ST35, ST36 and ST47 are decided as gates whose outcome the probe sets (N3); ST48 decides a cancelled, timed-out or hidden version read (N2); ST49 decides deferred tools on Ollama's Responses path; ST33 names the per-attempt limit the check reads; ST25's plants test how the cut is reported (F11). The forwarded results of the replay and thinking-refusal probes are not yet folded in.)* *(2026-11-23, revision 3, before any answer: Ollama probe round 6 (PR9). Ollama's server takes a reasoning item back even with its `encrypted_content` altered, so it does not validate it and no replay gain is claimed (ST47); and `/v1/responses` silently ignores an effort a model cannot take, where Chat refuses it in the form the SDK learns, so ST35's gate fails. Under the gates decided in revision 2, Ollama stays on Chat: OQ4's recommendation is now Chat, and under the recommended answers no provider changes its wire. It also folds in round 7 (PR10), which closes UM2: OpenAI and xAI report their cuts as `incomplete`, and xAI did not enforce `max_output_tokens` on a forced call, so the cut check would make such replies on `grok` finish `length` (OQ7 (2)'s cost).)* *(2026-11-23, revision 4, before any answer: after the second delta review. OQ4 (2)'s cost and ST35 are corrected: Ollama's listing reads each model's `thinking` capability from `/api/show` (C92), so only a model whose facts are not yet cached in the process has its effort dropped silently; the recommendation for OQ4 stays Chat, for the reasons given there. ST30 now agrees with B5 on what v1.33.0 waits for; the `grok` false positive is worded as a `length` finish with the calls kept; B3 notes that the xAI streamed text-cut defect is being fixed in 0088 Phase 1b; a stray blank line in the state table is gone.)*

## Context and Problem Statement

The owner asked: "are we leveraging the responses api and adopting it as fully as possible across all providers?", and chose, of the options the coordinator put, **"Research and adopt in v1.33.0"**: research first, then the adoption joins SDK v1.33.0. The research is done (the desk report and seven rounds of live probes, all of 2026-11-23). This record decides, provider by provider, which wire each SDK provider uses, and what the SDK does where a service's Responses endpoint deviates from OpenAI's. Where the owner must choose, the row is open and the question is put with three options, the recommended one first.

### Which wire each provider uses today, and why (read at 247b745)

| Provider | Wire today | Why, as recorded |
| :--- | :--- | :--- |
| `openai` (API key, ChatGPT backend, plan) | Responses (C29) | 0031-MADR: three providers speak the Responses wire (C40) |
| `grok` | Responses (C30) | as `openai` (C40) |
| `opencode` Zen and Go | per model: the model's `provider.npm`, `@ai-sdk/openai` routed to Responses, else the table (C27; 0012-MADR §3.1, C37) | the gateways do not translate; a wrong route is an opaque 500 |
| `kilo` | Chat Completions only (C13) | Kilo's `/responses` is undocumented and puts reasoning where the decoder does not read it; one route reaches the whole catalog (C14; 0004-MADR options 12 to 14, C31 to C34) |
| `huggingface` | Chat Completions only (C17) | `/responses` then undocumented, the docs said "chat completion tasks only", and a bad key answered HTTP 200 `status:"failed"` (C17; 0004-MADR option 8, C35, C36); "Do not 'fix' this omission without re-measuring" (C18) |
| `together` | Chat Completions only (C21) | the shared Chat primitive (0017-MADR, C39); no record discusses Together's Responses page |
| `ollama` | Chat Completions for generation, native `/api/tags` for listing (C23; 0005-MADR Q5, C38) | zero new wire logic; measured on a v0.31.1 server (C24) |
| `chatcompat` | Chat Completions (C26) | the package is defined as a provider for any server that speaks Chat Completions (0061-MADR D4) |
| `claude` | Messages | Anthropic's OpenAI-compatibility layer is Chat Completions only, has no `/responses`, ignores `strict`, returns no thinking, ignores `store` and prompt caching, and calls itself not "long-term or production-ready" (desk report, "Summary" 6 and "Anthropic compatibility layer") |
| `gemini` | Interactions (0014-MADR, C41) | Google documents no `/responses` on its OpenAI-compatible surface (desk report, "Summary" 6) |

The SDK has no OpenRouter provider. OpenRouter was probed because Kilo's own client is built on OpenRouter's (C49, C50); its results are read as evidence about the shapes such gateways send, not as an SDK provider's.

### What the Responses wire does in this SDK today (read at 247b745)

* One decoder serves `openai`, `grok` and OpenCode's responses route (C1); each of the three builds its own request body (C75, C76, C77).
* A reasoning item's text is read from `summary[]` only (C2); an item with no summary text and no `encrypted_content` is dropped (C3). The one reasoning delta mapped is `response.reasoning_summary_text.delta` (C4); `response.reasoning_text.delta` "was not seen and is not mapped" (C42), and every unmapped `response.*` event is skipped undecoded (C5).
* A reasoning item is replayed only when it carries encrypted content: "plain text reasoning cannot be replayed" (C6). On Chat Completions, Kilo replays `reasoning_details` (C15), and Together, Hugging Face and Ollama replay prior reasoning under the field the model's `interleaved.field` names (C22, C62, C63).
* A plain reply's `status:"failed"` is read before anything else and classified by its error object (C7, C8): a code that is an HTTP status is classified as that status (C9); any other code, such as `server_error`, is classified as an HTTP 500 unless the service's table or the error's type says otherwise (C65, C10), and a 500 is the retryable `ErrProviderUnavailable` (C66). A `failed` with no error object is retryable too (C79). No table row names a Hugging Face code.
* `[DONE]` is skipped (C11), and the reader stops at the terminal event.
* Usage's reasoning count is the service's `output_tokens_details.reasoning_tokens` (C12); a count the service does not report is zero (C45).
* A request with `PreviousResponseID` is refused before the network by a provider whose `Continuation` is `Unsupported` (C46).
* A plain `Generate` whose output limit the SDK chose or left out is read over the stream (C47; 0088-MADR ST88).
* On an optional-limit wire, nothing is sent when the caller names no limit (0089-MADR DL47, C43); Ollama with no limit is then unbounded (C44). 0089's limit-refusal forms are read on the Chat Completions wires, and not on Responses, whose text is not established (0089-MADR DL68, C48).
* **The decoder never sees the limit sent.** `Response.OutputLimit` is set after decoding, by the provider's limit rule (C67): on `Generate` after `wire.Send` returns (C68), on `Stream` by wrapping the decoded stream (C69). The decoder takes only per-request fields of its own (C80).
* Ollama's Chat path learns a refusal of thinking through the Chat Completions matcher (C70; `"<model>" does not support thinking`, C71), sends its effort by the model's facts with 0062's ladder, xhigh and max included (C72), sends no tool choice (C73) and no `strict` (C74), drops an image by URL with a warning and sends an image's bytes as a data-URL string (C81, C82), and probes each listed model with one generation (C83). The Responses wire's refusal matcher reads OpenAI's and xAI's forms only (C78).

### What the services' Responses endpoints are (desk research, 2026-11-23)

* **Kilo**: live but undocumented; the vendor's reference lists only `/chat/completions`, `/fim/completions`, `/systemone`, `/models`, `/providers`. kilocode, Kilo's own client, wraps OpenRouter's AI SDK provider (C50) and names an "OpenRouter-compatible endpoint (chat, responses, messages, models)" (C49). No source says the whole gateway is OpenRouter.
* **OpenRouter**: `POST /api/v1/responses`, labelled beta in a search of its docs, stateless only: `store:true` or a non-null `previous_response_id` is a 400.
* **Together**: `POST /v1/responses`, "partial support", for four models only (`zai-org/GLM-5.2`, `GLM-5.3`, `GLM-5.3-Flash`, `deepseek-ai/DeepSeek-V4.1-Flash-1017`); others "may return malformed output". `reasoning.summary` always empty; `encrypted_content` base64, "not encrypted"; images on two models; `input_file` 400; a `file_id`-only image silently dropped; `previous_response_id` an error; hosted tools and remote MCP 400.
* **Hugging Face router**: `POST /v1/responses`, a guide titled "Responses API (beta)"; "All Inference Providers chat completion models should be compatible".
* **Ollama**: documented, "Added in Ollama v0.13.3", "Only the non-stateful flavor is supported"; images not mentioned. codex gates the route on 0.13.4 (C51) and reaches Ollama on Responses only (its `ollama-chat` provider was removed).
* **Anthropic, Gemini**: no `/responses` (above).

### What the live probes measured (2026-11-23, allowlisted facts, `store:false`)

Round 1 ran P1 (plain), P2 (stream events), P3 (a cut in text, plain and stream), P4 (a cut while a tool is forced), P5 (parallel calls, `strict`), P6 (the reasoning item's keys, with `include: ["reasoning.encrypted_content"]` sent: the probe script's P6 body, `<scratch>/responses-probe/probe.py:272`), P7 (a bad key), P8 (a made-up `previous_response_id`), P9 (an image). Round 2 forced a cut inside a function call's arguments. Rounds 3 to 6 probed the local Ollama; round 7, OpenAI and xAI.

| Id | Service, model | Measured |
| :--- | :--- | :--- |
| PR1 | Kilo, `kilo-auto/free` | P1 200 `completed`; usage `input_tokens`, `cached_tokens`, `output_tokens`, `reasoning_tokens`, `total_tokens`, `cost`, `cost_details`, `is_byok`. P2 events include `reasoning_text.delta` and `.done`. P3 `incomplete`, `max_output_tokens`, the stream ending `response.incomplete`. P4 `incomplete` (cut in reasoning). P5 two calls, arguments parse, `strict` 200. P6 reasoning keys `content, format, id, status, summary, type`, summary empty, content `reasoning_text`, **no `encrypted_content` although it was asked for**. P7 401, `error.code` `INVALID_TOKEN`, top keys `error`, `error_type`. P8 400, `error` a string. P9 404. |
| PR2 | OpenRouter, `openai/gpt-oss-20b` | As Kilo's, the usage field set included; P2 and P3 streams then a literal `[DONE]` after the terminal event. P7 401 with a numeric code. P8 400 `invalid_prompt`. P9 404 (likely the model). |
| PR3 | Together, `zai-org/GLM-5.3-Flash` | P1 200, usage adds `cache_write_tokens`, no cost. P3, P4 `incomplete`, `max_output_tokens`. P5 two calls, `strict` 200. P6 `encrypted_content` present, summary empty, content `reasoning_text`. P7 401 `invalid_request_error` / `invalid_api_key`. P8 400 `previous_response_not_found`. P9 200 `incomplete`, `max_output_tokens`, the 64-token limit hit inside reasoning: whether the image was taken is not shown. |
| PR4 | Hugging Face, `openai/gpt-oss-20b` | P1 200, `reasoning_tokens` 0 although reasoning items exist. P3 plain and stream 200 **`completed`**, `incomplete_details` null, at `output_tokens` equal to the limit: **a cut is not reported**. P4 plain `completed` with no call; the stream ends **`response.failed`** (its error code not recorded). P6 **no `encrypted_content` although it was asked for**. P7 **HTTP 200 `status:"failed"`, `error.code` `server_error`**. P8 200 `completed`: the made-up id ignored. P9 200 `status:"failed"`, `server_error`. |
| PR5 | Ollama, local server 0.40.17, `llama3.2:latest` | P1 200 `completed`. P2 created ... completed. P3 200 **`completed`** at `output_tokens` equal to the limit: **a cut is not reported**. P5 two calls, `strict` 200. P8 200 `completed`: ignored. |
| PR6 | Round 2: a cut inside a call (forced tool, a 600-word argument) | Kilo (`kilo-auto/free`, effort none, limit 100), OpenRouter (`meta-llama/llama-3.1-8b-instruct`, 160), Hugging Face (`meta-llama/Llama-3.1-8B-Instruct`, 160): plain 200 **`completed`**, `incomplete_details` null, the call's item **`completed`**, its arguments **do not parse**, `output_tokens` **equal to the limit**. Kilo's and OpenRouter's streams: `function_call_arguments.delta`, `.done`, `output_item.done` (the call `completed`, not parsing), then **`response.completed`** (OpenRouter then `[DONE]`). Hugging Face's stream finished inside the limit (159 tokens, parse true). Together: the cut fell in reasoning at limits 160 and 400, and no model on its list is a non-reasoning one: unmeasured. |
| PR7 | Round 3, Ollama 0.40.17 | (a) `gpt-oss:20b`, `qwen3:8b`, effort low, `include` sent: 200 `completed`, items reasoning and message; reasoning keys `encrypted_content, id, summary, type` (no `content`), summary length 1, **`encrypted_content` present**, `reasoning_tokens` reported 0. (b) `gemma3:12b`, `gemma4:latest`, a 1x1 PNG: Responses 200 `completed`, Chat 200 `stop`; status only, whether the image was used not checked. (c) a cut inside a call on Responses not obtained (`llama3.2` honoured the forced tool once, finishing at 134 of 160). (d) Chat, forced tool, limit 160, `llama3.2`: 200, `finish_reason` `length`, 160 tokens, no call. (e) text at a limit of 16, `llama3.2`: **Responses `completed`, reason null, `output_tokens` 16; Chat `length`**. |
| PR8 | Rounds 4 and 5, Ollama 0.40.17 | `gpt-oss:20b`: **every request with tools fails on both wires** (HTTP 500 `api_error`, forced or `auto`); without tools, Responses 200. `qwen3:8b` (limit 400), `gemma4:latest` (300): the forced tool ignored; `completed`, reason null, **`output_tokens` equal to the limit** (a text cut reported as `completed`). `granite3-dense:8b` (300): the forced tool ignored; `completed`, 84 tokens. `gemma3:12b`: HTTP 400 `invalid_request_error` with tools. Of six models only `llama3.2` ever returned a call, intermittently and whole: **a cut inside a call on Ollama is unobserved**. Untried: `deepseek-r1:7b`, a granite `q5_K_M` variant. |
| PR9 | Round 6, Ollama 0.40.17 | Replay round trip (turn 2 sends turn 1's reasoning item and message back; `qwen3:8b`, `gpt-oss:20b`, no tools): **200 `completed` with the item as returned, and again with one character of its `encrypted_content` changed**. The server does not validate the item; whether it uses it is not shown. Thinking refused on `llama3.2` (no thinking): Responses with effort high or xhigh **200 `completed`, no error**, the effort silently ignored; Chat Completions `reasoning_effort` high or xhigh **400 `invalid_request_error`** (code null), the form the SDK's Ollama pattern in `chatcompletions.MatchRefusal` reads (C70, C71). |
| PR10 | Round 7, OpenAI (`gpt-4o-mini`) and xAI (`grok-4.20-0309-non-reasoning`) | OpenAI: a text cut, plain, `incomplete`, `max_output_tokens`, the message item `incomplete` with its text; streamed, every `.done` event, the item `incomplete`, then `response.incomplete`. A call cut (limit 160), plain, `incomplete`, the call item `incomplete`, arguments do not parse, 160 tokens; streamed, the arguments' `.done`, `output_item.done` (item `incomplete`), `response.incomplete`. xAI: a text cut, plain, top `incomplete`, `max_output_tokens`, the message item `completed`, with its text; streamed, text deltas then **`response.incomplete` with no `output_text.done`, `content_part.done` or `output_item.done`**, the final response's item `incomplete`. A forced call: **`max_output_tokens` not enforced** (1042 and 861 tokens against 160), `completed`, arguments parse; a cut inside a call not observed. |

The probe agent's finding from round 2, quoted: "on Kilo, OpenRouter and Hugging Face, a reply cut inside a function call's arguments is reported as success (`completed` response, `completed` item, `.done` events) with arguments that do not parse as a JSON object and output_tokens equal to the limit." In every cut the probes measured that the service reported as `completed`, in text or in a call, `output_tokens` equalled the limit sent.

### How this meets 0088's cut answer

0088-MADR's OQ11 answer, "Inside 0088, this tag", reports a reply the output limit cut one way on every wire: a response finishing `length` with its text and whole calls, the cut call dropped and named in `WarningToolCallCut`, or `ErrEmptyAnswer` with nothing usable (0088-MADR ST75 to ST84). Its PLAN Phase 1b is being built in the tree now, and three of the builder's code choices are open for that PLAN's Phase 3 review, which rules on each with the owner (W5): R1, the wire-general cut rule (a call whose arguments are not a JSON object, or a last call with `{}`, on a reply the limit cut); R2, Responses' cut call (an item `incomplete`, or added and never done); R3, `wire.HeldCalls`, which holds a streamed call that might be cut until the stream shows otherwise. This record does not decide R1, R2 or R3; where a row of this record depends on one, it says so and is conditional on that ruling. What the probes show about each is under B3. *(2026-11-25: R1 to R3 are now ruled, in [0088-MADR-default-output-limit-from-model-facts.md](0088-MADR-default-output-limit-from-model-facts.md) ST107 to ST110 (its fifteenth and sixteenth amendments): R2 and R3 are kept as built, and on the Responses wire R1 is kept as built. The text above is unchanged.)*

## Decision Drivers

* **The owner's direction**: adopt the Responses API as fully as the evidence allows, in v1.33.0.
* **No regression against today's Chat Completions path**: a provider that moves keeps how its errors are classified, sees the cuts it sees today, keeps image input where it is declared, keeps reasoning replay and refusal learning where they are done today, and keeps 0089's limit-refusal learning where it applies.
* **Measured, not assumed** (0031-MADR's driver): a deviation the probes measured is handled with a test; a behaviour nobody measured is not built on, and a decision that rests on one names its probe.
* **Documented routes over undocumented ones**: a route the vendor does not document is the one 0004-MADR declined twice (C31 to C36).
* **Per-model decisions from model facts, not ids** (0041-MADR D1, C52).
* **The smallest exported change** (docs/guides/api-standards.md R55, C53), and the wire does not change except where a record changes it (R45, C54).
* **One SDK report of a cut**, whatever the service calls it, where the evidence shows the SDK can tell.

## Considered Options

* A. Move every provider that has a Responses route
* B. Move each provider that passes three gates, and read a cut from evidence where a service was measured hiding it
* C. Keep every Chat Completions provider, and harden only the shared decoder
* D. Chat Completions by default, Responses as an opt-in per provider

A moves `kilo`, `together`, `huggingface`, `ollama`, and `chatcompat` where the server speaks Responses, and handles each measured deviation in the SDK. B's three gates are: the vendor documents the route; nothing is lost against today's Chat path; every measured deviation is handled (B1). C keeps every Chat provider where it is and changes only the decoder the Responses providers share. D adds one exported option per provider.

## Decision Outcome

Chosen option (proposed; the owner has not answered): "B. Move each provider that passes three gates, and read a cut from evidence where a service was measured hiding it", because it is the reading of "as fully as possible" that the evidence supports: it moves a provider where the vendor documents the route and the SDK loses nothing it has today, and keeps a provider on Chat Completions where the probes measured a loss. **On this evidence no provider passes all three gates.** Ollama, the one provider whose route the vendor documents and whose reference client uses it alone, fails G2: on `/v1/responses` an effort the model cannot take is silently ignored, where its Chat route refuses it and the SDK learns the refusal and warns (PR9, ST35); and the replay of its reasoning, the one gain it offered, is unproven, since the server takes the item back unvalidated (PR9, ST47). Under the recommended answers, then, no provider changes its wire in v1.33.0, the shared Responses decoder is not changed, and the record's v1.33.0 work is its docs and its REPORT; what a later move would take (the gates, the cut check, Ollama's Chat-path parity) is decided now, so that a move needs only an amendment with new measurements. The per-provider wires (OQ1 to OQ5), the cut check's signal (OQ6) and reach (OQ7), and reasoning-text replay (OQ8) are open; the record stays `proposed` until the owner answers.

### B1. The gates, per provider

A provider moves to the Responses wire only when all three gates hold. G2 compares with what the provider does today on Chat Completions, as its package doc states. The gains column lists what the Responses wire measurably gives over the provider's Chat path.

| Provider | G1: the vendor documents the route | G2: nothing lost against today's Chat path | G3: every measured deviation handled | Gains on Responses | Recommended (question) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `kilo` | no (desk report, Kilo) | no: no `encrypted_content` even when asked (PR1 P6), so reasoning is not replayed (C6) where Chat replays `reasoning_details` (C15); 0089's limit-refusal learning is read on Chat only (C48); image input unmeasured (P9 404) where Chat declares it Supported (C16). A cut inside a call is reported `completed` (PR6); how Kilo's Chat route reports the same cut is unmeasured (UM5) | with ST10, OQ6, OQ7 | none measured: typed reasoning items only with ST10; usage `cost` (not compared with Chat's) | Chat Completions (OQ1 (1)) |
| `together` | partly: "partial support", four models | no: other models "may return malformed output"; at most four models move, by an id table against 0041-MADR D1 (C52); 0089's "supports at most" refusal form is read on Together's Chat route, not on Responses (C48); images on two of the four; a `file_id`-only image silently dropped | a cut inside a call unmeasured (PR6) | `encrypted_content` returned (PR3 P6), so reasoning could be replayed on the four models; `cache_write_tokens` in usage; text and reasoning cuts reported `incomplete` as OpenAI does (PR3 P3, P4) | Chat Completions (OQ2 (1)) |
| `huggingface` | beta guide | no: a bad key and an image refusal are both HTTP 200 `failed` with `server_error` (PR4 P7, P9), which the SDK classifies as the retryable `ErrProviderUnavailable` (C65, C66) where an HTTP 401 is classified by its status (C58); a stream cut in reasoning ends `response.failed` (PR4 P4), retryable whatever its code (C65, C66, C79), so a cut would be resent and billed again (ST45); no `encrypted_content` even when asked (P6), so reasoning is not replayed where Chat replays it (C62). A text cut is reported `completed` (PR4 P3); Chat's report of it unmeasured | only with a message rule resting on one 2026-08-29 measurement (C36) | none measured | Chat Completions (OQ3 (1)) |
| `ollama` | yes, from 0.13.3; codex 0.13.4 (C51) | **no**: an effort the model cannot take is silently ignored on Responses, where Chat refuses it and the SDK learns it, sends again with no effort and warns (PR9; C70, C71; ST35); a text cut reported `completed` where Chat says `length` (PR7 (e), PR8), kept only by the cut check (ST41). Measured without loss: images answered on both wires (PR7 (b), status only); reasoning returned with `encrypted_content` (PR7 (a)) | with OQ6 (1) or (3) and OQ7 (1) or (2), except ST35, which no handling restores | none established: the reasoning item is taken back unvalidated, so its use, and any replay gain over Chat's interleaved field (C63), is unproven (PR9, ST47); a reasoning summary; the vendor's documented route and codex's only route | Chat Completions (OQ4 (1)) |
| `chatcompat` | not applicable: any server, none measured | not established for any server | not applicable | none measured | Chat Completions (OQ5 (1)) |
| `opencode` | per model (C27) | unchanged | not applicable unless OQ7 (2) | not applicable | per model, unchanged (ST6) |
| `openai`, `grok` | yes | unchanged | not applicable unless OQ7 (2) | not applicable | Responses, unchanged (ST7) |
| `claude`, `gemini` | no `/responses` | not applicable | not applicable | not applicable | native, unchanged (ST8, ST9) |

Under the recommended answers, no provider changes its wire, and the cut check (B2) has no provider to run on, so it is not built in v1.33.0. B2 and the rows marked "if Ollama moves" decide what a move under OQ1 to OQ5's other options would take.

### B2. A cut a service reports as complete (OQ6 and OQ7, recommended answers)

**What the evidence supports.** Services were measured reporting a cut as `completed` on Kilo, OpenRouter and Hugging Face (inside a call, PR6), and on Hugging Face and Ollama (in text, PR4 P3, PR5 P3, PR7 (e), PR8). Under the recommended answers Kilo and Hugging Face stay on Chat Completions, OpenRouter is not an SDK provider, and no probe shows `openai`, `grok` or OpenCode's responses route reporting a cut as `completed` (UM2). So the check's measured reach is a provider that moves and was measured hiding a cut: Ollama under OQ4 (2) or (3), Kilo or Hugging Face under their option (2); under the recommended answers, none. Reaching `openai`, `grok` and OpenCode is OQ7 (2); round 7 (PR10) found nothing there for it to catch: OpenAI and xAI report their cuts as `incomplete`, which 0088's path already reads, and xAI's unenforced limit on calls would make a `grok` reply holding whole calls past the limit finish `length` instead of `tool_calls`, its calls kept (they parse, so 0088's rule drops none).

**The signal (OQ6 (1)).** On a provider the check covers, a reply whose status is `completed` (plain, or `response.completed` on a stream) is read as a reply the output limit cut when its `output_tokens` is at or above the limit the request sent ("T2"). With no limit sent, nothing is read as cut. Every measured hidden cut had `output_tokens` equal to the limit (PR6, PR4 P3, PR5 P3, PR7 (e), PR8). Which call of such a reply is the cut one is then 0088's rule (a call whose arguments are not a JSON object, or, by R1 as built, a last call with `{}`). *(2026-11-25: R1 is ruled, as 0088-MADR ST107 to ST110. On Responses a last call with `{}` stays the cut call, so this sentence holds. The text is unchanged.)*

**Where it runs, and how the limit reaches it (ST33, ST34).** The check runs inside the Responses decoder, plain and streamed, before a held call is released and before the stream's last event. The limit sent reaches it as a per-request field of the internal `responses.Decoder` (C80), set by the provider from the value it writes to `max_output_tokens`; none written, none set. It cannot run in the provider's limit rule after decoding (C67 to C69): on a stream, `streamEnd` yields the held calls the response kept and then `EventDone` (W6), so by the time the rule sees `EventDone` a cut call has gone out. On a stream the check depends on 0088's R3 holding back a call whose arguments are not whole until the end (W3); that dependency is conditional on 0088's Phase 3 ruling (ST34).

Such a reply is then finished as 0088's cut answer finishes a reply the limit cut, with the reason `max_output_tokens`, under the rules 0088's Phase 3 review sets (ST25). A `failed` or `incomplete` reply is read as today.

### B3. Evidence for 0088's Phase 3 review (not a decision of this record)

The review rules on R1, R2 and R3 with the owner. What the probes and the tree show, for that review:

* **R2 alone misses every measured cut inside a call.** R2 names a call whose item is `incomplete`, or that a stream added and never finished. On Kilo, OpenRouter and Hugging Face the cut call's item was `completed` and its `.done` events arrived (PR6). R2 never fires on them.
* **R1 as built fires only on a reply already known to be cut.** `wire.CutAnswer`, where `wire.CutCall` (W4) applies, is called on the Responses wire only for `status:"incomplete"` with the reason `max_output_tokens` (W1); a `completed` reply goes through `completed()`, which returns the non-object call as a whole call with `FinishToolCalls` (W2). Its arguments test is the signal PR6 shows works, once a reply is known to be cut.
* **R3 delays the measured cut call and then yields it.** `HeldCalls.Hold` holds the non-object call; no later item releases it; at `response.completed`, `HeldCalls.Done` yields each held call the finished response kept (W3, W6), and the completed response kept it. The cut call reaches the caller just before `EventDone`. ST34 relies on R3's hold.
* **A cut in text that the service reports as `completed`** (PR4 P3, PR5 P3, PR7 (e), PR8: Hugging Face; Ollama on `llama3.2`, `qwen3:8b`, `gemma4`) is not a call, so no call rule sees it; only `output_tokens` against the limit sent does. Ollama's Chat route reports the same cut as `length` (PR7 (e)).
* **The SDK's own Responses services report cuts as `incomplete`** (PR10): OpenAI marks the cut item `incomplete` too, so R2 names OpenAI's cut call. xAI marks only the top status on a plain text cut (the item `completed`), and its streamed text cut ends `response.incomplete` with no `output_item.done`: the decoder builds a stream's output from `output_item.done` events and takes only the id, model and usage from the last event (W7), so that answer's text, already yielded as deltas, is not in the response, and the cut answer finds nothing usable (0088 ST79). The owner ruled on it, "Fix it now in Phase 1b (Recommended)", relayed by the coordinator on 2026-11-23: it is being fixed in 0088's Phase 1b, not left to the review.
* **xAI did not enforce `max_output_tokens` on a forced call** (PR10: 1042 and 861 tokens against 160), so a check that reads `output_tokens` at or above the limit would read those replies as cut: they would finish `length` instead of `tool_calls`, with their whole calls kept. It is also a fact for 0089's limit rows on Grok, which this record does not decide.
* **Hugging Face's stream cut in reasoning ends `response.failed`** (PR4 P4), which the SDK reads as a retryable failure, not a cut (C7, C8, C65, C66, C79).
* **Not established**: whether Kilo's and Hugging Face's Chat Completions routes report a cut inside a call as `finish_reason: "length"`. On Chat, 0088's cut answer runs on `length`; if these routes say `tool_calls` or `stop` there, R1 on Chat has the same blind spot. The gobble report shows Kilo's Chat route finishing `length` for a cut in reasoning (C55), and Ollama's Chat route says `length` at the limit (PR7 (d), (e)), neither inside a call.

### B4. What stays unmeasured, and what waits on it

| Item | Unmeasured | A decision waits on it? |
| :--- | :--- | :--- |
| UM1 | Ollama on `/v1/responses`: xhigh and max on a thinking model; an image actually used (the answer names the picture's colour). The thinking refusal and the replay round trip were measured in round 6 (PR9): ST35's gate failed, ST47's gain unproven | only under OQ4 (2): gates decided now, whose outcome the probe sets (ST22, ST36) |
| UM1b | A cut inside a call on Ollama: unobserved after six local models (PR7 (c), PR8); untried `deepseek-r1:7b` and a granite `q5_K_M` | **no**: the check reads `output_tokens` against the limit whatever the cut falls in, and every hidden call cut measured elsewhere had `output_tokens` equal to the limit (PR6); the PLAN tries the two untried models once and records the result (ST42) |
| UM2 | closed by round 7 (PR10): OpenAI and xAI report cuts as `incomplete`; xAI did not enforce the limit on a forced call; a cut inside a call not observed on xAI | none waits; OQ7 (2)'s costs now state it |
| UM3 | A cut inside a Together call (no non-reasoning model on its Responses list) | only under OQ2 (2) or (3) |
| UM4 | Hugging Face's stream cut inside a call (the stream finished inside the limit) | only under OQ3 (2) or (3) |
| UM5 | Kilo's and Hugging Face's Chat Completions finish for a cut inside a call | no decision of this record; evidence for 0088's review (B3) |
| UM6 | Whether a service takes a reasoning input item carrying `reasoning_text` content | only under OQ8 (2) |

The probes are the PLAN's first phase, live-tagged, switched on by `LLMPROVIDER_LIVE_0091` with a suffix per target as AGENTS.md has it for a record's own measurement (C56), run by an agent with environment keys only (Ollama's needs none), never a key value read and no raw output kept: each run writes only allowlisted fields (the response status, `incomplete_details.reason`, each item's type and status, whether each call's arguments parse as an object, `output_tokens` against the limit, the event type names, an error's status, code and type, and for UM1's image whether the answer holds the expected colour word) under `tmp/live-0091/`.

### B5. Release, API and docs

* **Release**: v1.33.0, the owner's ruling. Under the recommended answers its work is docs and the REPORT, with no code: the docs below, and the REPORT (ST27). Under an answer that moves a provider, it is built after 0088's phases and its Phase 3 review (B2 depends on its ruling), serially in the same tree, and the tag waits for it. Its entries join `[Unreleased]`.
* **API**: under the recommended answers, no exported name changes. Under OQ4 (2) or (3) Ollama's wire is chosen inside the package, and the cut check and its limit field are inside the internal Responses package. `python3 -B tools/apidiff.py v1.32.0` lists nothing from this record. An opt-in option (OQ1 (3), OQ3 (3), OQ5 (2)) adds one exported option per provider, a scoped option as `opencode.WithRoute` is (C28), with no outside call site named (R55's cost, C53).
* **Goldens**: a provider that moves gains G-wire goldens for its Responses request and its recorded reply shapes, for `Generate` and `Stream`; its Chat goldens stay for the path it keeps (R45, C54).
* **Docs**: the package doc of every provider whose wire or whose stated reason changes (under the recommended answers, `kilo`, `together`, `huggingface` and `ollama` each state the 2026-11-23 measurement that keeps it on Chat); `docs/architecture.md`; `README.md`'s provider table; the Responses package doc (C1); a **Changed** entry per provider that moves and for the cut check; and the Hugging Face package doc, whose "undocumented" and "chat completion tasks only" (C17) the desk research superseded, restated with the 2026-11-23 measurement whatever OQ3's answer.

### Consequences

* Good, because no provider moves onto a route where the probes measured an auth failure classified as a retryable outage, a cut reported as success, an effort dropped without a word, or reasoning that can no longer be replayed.
* Good, because each provider's package doc states, from 2026-11-23 measurements, why it is on its wire, and the REPORT keeps the facts, so a later move starts from them.
* Good, because what a move would take (the gates, the cut check and where it runs, Ollama's Chat-path parity) is decided now, so a provider that later passes needs only an amendment with its measurements.
* Neutral, because under the recommended answers no code, no API and no wire changes, and v1.33.0 does not wait on 0088's Phase 3 review for this record.
* Bad, because "adopt as fully as possible" yields, on this evidence, no provider's move: the owner asked for adoption, and the measurements show every candidate route losing something the Chat path gives today. The owner may weigh that loss differently (OQ1 to OQ5's other options).
* Bad, because under an answer that moves Ollama, the check reports as cut an answer that ended on exactly the last token of the caller's limit (`length` in place of `stop`), its effort refusal is lost (ST35), and the package carries two wires and its own Responses body builder.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes behaviour and green after, each with its plant caught on a scratch clone (`tools/plant.py`); a row that names no plant says why none can exist.
* `python3 -B tools/apidiff.py v1.32.0` lists no name from this record, under the recommended answers.
* The PLAN's probe phase (B4) is run and its allowlisted facts are committed as a REPORT of this record's number.
* `tools/claims.py` passes this record's evidence index, and claims planted false are caught.
* The independent review of the PLAN's last phase finds no blocker.

## Owner Questions

Eight questions, each with three options, the recommended one first. Each names its cost and the rows its answer decides. "Today's Chat path" is the provider's current Chat Completions behaviour; "the cut check" is OQ6's.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ1. Kilo: should it stay on Chat Completions or move to Kilo's `/responses` route? | (1) **Chat Completions stays.** (2) **Responses**, reading Kilo's reasoning from where it puts it (ST10) and with the cut check. (3) **Chat by default, Responses as an opt-in option.** | (1) nothing changes; the package doc's reason (C14) gains the 2026-11-23 measurement. (2) an undocumented route (0004-MADR declined it, C33, C34); no encrypted reasoning even when asked, so a reasoning model's next turn loses its reasoning (C6, C15); 0089's limit-refusal learning lost (C48); images unmeasured; every gobble-cli turn changes path; no model gained, as Kilo translates (C32). (3) one exported option, no outside call site (R55); every cost of (2) for a caller who opts in. | ST1 |
| OQ2. Together: should it stay on Chat Completions or use `/responses`? | (1) **Chat Completions stays, every model.** (2) **Responses for the four models Together lists**, Chat for the rest, by an id table. (3) **Responses for every model.** | (1) Together's encrypted reasoning on Responses is not used. (2) an id table against 0041-MADR D1 (C52), refreshed by hand; 0089's "supports at most" refusal form lost on those models (C48); a cut inside a call unmeasured (UM3); images on two of the four. (3) Together warns other models "may return malformed output"; every cost of (2). | ST2, ST20 |
| OQ3. Hugging Face: should it stay on Chat Completions or use the router's beta `/responses`? | (1) **Chat Completions stays**; the package doc is brought up to date. (2) **Responses**, with a rule that a `failed` reply whose message begins with an HTTP status is classified as that status (the form recorded on 2026-08-29, C36), plus the cut check. (3) **Chat by default, Responses as an opt-in.** | (1) nothing changes in code. (2) the rule rests on a message text last recorded on 2026-08-29; an image refusal has the same `server_error` and no measured message; a stream cut in reasoning ends `failed`, a retryable outage, so it is resent and billed again (ST45); no encrypted reasoning even when asked, so reasoning replay is lost (C62); beta. (3) one exported option (R55); every cost of (2) for a caller who opts in. | ST3, ST12, ST45 |
| OQ4. Ollama: should it move to `/v1/responses`, which Ollama documents and codex uses alone? Round 6 measured that on Responses an effort the model cannot take is silently ignored, where Chat refuses it and the SDK learns it and warns, and that the server takes a reasoning item back without checking it, so no replay gain is shown. | (1) **Chat Completions stays**: the gate decided in revision 2 failed (ST35). (2) **Responses anyway, with reasoning sent only to a model whose facts say it thinks**, on a server at or above 0.13.4 (read from `GET /api/version`, C57; ST48), Chat below or when unread; probe xhigh, max and an image first (ST22, ST36); needs OQ6 (1) or (3) and OQ7 (1) or (2). (3) **Responses only**, a server below 0.13.4 refused with `ErrUnsupported`. | (1) nothing changes in code; Ollama's documented route and codex's only route stay unused; no measured capability is lost, since the replay gain is unproven (ST47). Recommended still after the correction in (2)'s cost: the remaining gap is narrow, but Chat has none, and Responses offers no measured gain to set against it. (2) Ollama's listing reads each model's `thinking` capability from `/api/show` (C92), so a model the process has listed is gated correctly; a model whose facts are not yet cached (no listing run yet in the process, since a request never fetches, or a reply with no capability list) has an effort it cannot take dropped with no error and no warning, where Chat refuses and the SDK learns it on the first request; a new Responses body builder (C75 to C77), two wires, a version request, two sets of goldens; the cut check's false positive. (3) every server below 0.13.4 stops working; every model's unsupported effort dropped silently. | ST4, ST21, ST35; the gates ST22, ST36 apply under (2) |
| OQ5. chatcompat (any server the caller points at): should it stay Chat Completions? | (1) **Chat Completions stays**: the package is defined as a Chat Completions provider for any server (C26). (2) **An opt-in `chatcompat.WithResponses()`** for a server the caller knows speaks Responses. (3) **Detect at listing**: probe `{base}/responses` once per base URL and use it when it answers. | (1) a server that speaks Responses is still reached on Chat. (2) one exported option (R55); no server measured. (3) a request per base URL a local server may log or refuse; a server that answers but deviates (as four of five measured services do) is used unmeasured. | ST5 |
| OQ6. Some services answer `completed` when the output limit cut the reply. Which signal should make the SDK treat such a reply as cut? (It matters only if an answer to OQ1 to OQ5 moves a provider; under the recommended answers the check is not built.) | (1) **Output tokens at the limit**: a `completed` reply whose `output_tokens` reached the limit the request sent is a cut; 0088's rule then finds the cut call, if any. (2) **A broken call at the limit**: only a reply at the limit that holds a call whose arguments are not a JSON object is a cut. (3) **Both, and a broken call even with no limit sent**: (1), plus a call whose arguments are not a JSON object marks a reply cut when no limit was sent. | (1) catches every hidden cut measured, text and call (all at `output_tokens` equal to the limit); false positive: an answer ending naturally on exactly the limit's last token finishes `length` (text kept; under R1 as built a trailing `{}` call dropped); no limit sent, no check. (2) fewer false positives, but a hidden text cut stays `stop`, so a moved Ollama loses Chat's `length` (PR7 (e)). (3) also catches a cut by a service's own unannounced cap; false positive: a model that writes malformed arguments with no cut has its call dropped and named, where today the caller receives it. | ST14, ST15, ST41 |
| OQ7. Where should that check run? | (1) **Only on a provider that moves and was measured hiding a cut**: none under the recommended answers; Ollama under OQ4 (2) or (3); Kilo or Hugging Face under their option (2). (2) **Every provider on the Responses wire**, `openai`, `grok` and OpenCode's responses route included. (3) **Nowhere**. | (1) a provider later found hiding cuts needs an amendment. (2) round 7 found nothing for it to catch: OpenAI and xAI report cuts as `incomplete` (PR10), which 0088's path reads; and xAI did not enforce the limit on a forced call (1042 and 861 tokens against 160), so under OQ6 (1) every such whole `grok` call would finish `length` instead of `tool_calls`; each provider must pass its limit to the decoder. (3) a provider moved under another answer loses the cuts its service hides (Ollama's `length`, PR7 (e)). | ST14, ST15 |
| OQ8. When a service returns reasoning only as plain text (Kilo, Hugging Face), should the SDK send it back on the next turn? | (1) **No, as today**: only encrypted reasoning is replayed (C6); under the recommended answers no provider that moves needs it, since Ollama returns encrypted reasoning. (2) **Probe first**: on a provider that moves and returns plain-text reasoning, replay it as a reasoning item with `reasoning_text` content once a probe shows the service takes it (UM6). (3) **Replay it without a probe.** | (1) a provider moved later under OQ1 (2) or OQ3 (2) loses reasoning replay. (2) one probe per such service; an input form no SDK record has measured. (3) an unmeasured request form on every turn, possibly refused. | ST11 |

## State Table

"Decided" means decided by an existing record, by code no option changes, by every option a question offers that it applies under, or by the owner's ruling, named in the row. "Decided, if Ollama moves" applies under OQ4 (2) and (3). "Decided, conditional on 0088" reopens if 0088's Phase 3 ruling changes what it rests on. Each decided row names the test that pins it and the plant that must turn it red, from the PLAN to come, or says why no plant can exist.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST1 | Kilo's wire | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: Chat Completions stays | OQ1 | |
| ST2 | Together's wire | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: Chat Completions stays, every model | OQ2 | |
| ST3 | Hugging Face's wire | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: Chat Completions stays | OQ3 | |
| ST4 | Ollama's wire | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: Chat Completions stays (ST35's gate failed) | OQ4 | |
| ST5 | chatcompat's wire | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: Chat Completions stays | OQ5 | |
| ST6 | OpenCode Zen and Go | decided | per model by `provider.npm`, the table as fallback, `WithRoute` over both, as today | 0012-MADR §3.1 (C37); C27 | the existing route tests; plant: an `@ai-sdk/openai` model sent to the chat route |
| ST7 | `openai` and `grok` | decided | Responses, as today | 0031-MADR (C40) | the existing wire goldens; no plant can exist: no code changes |
| ST8 | `claude` | decided | Messages stays: Anthropic offers no Responses route, and its Chat layer drops thinking, `strict` and caching | desk report, "Anthropic compatibility layer" | the existing wire goldens; no plant can exist: no code changes |
| ST9 | `gemini` | decided | Interactions stays: no `/responses` documented | 0014-MADR (C41); desk report | the existing wire goldens; no plant can exist: no code changes |
| ST10 | A reasoning item whose text is in `content[]` as `reasoning_text`, `summary` empty (PR1, PR3, PR4 P6), and the `response.reasoning_text.delta` event (PR1 P2) | decided *(2026-11-23: restated in the amendment)* | built only in the phase of a provider that moves and was measured sending it: the shared decoder then reads `reasoning_text` content when the summary holds no text, and yields that delta as `EventReasoningDelta`. Its reach is every Responses provider, `openai`, `grok` and OpenCode included; none of them was seen sending it (C42), so it adds and changes nothing for them. Under the recommended answers it is not built: Ollama's reasoning is OpenAI-shaped (PR7 (a)) | the measurements; C2, C4, C42 | if built: `TestResponses_ReasoningText_0091`, plain and stream, with an OpenAI-shaped summary item decoding unchanged; plants: the decoder reading `summary[]` only; the `content` text preferred over a non-empty summary. Not built: no plant can exist |
| ST11 | Plain-text reasoning (no `encrypted_content`) on a moved provider's next turn | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: not replayed, as today (C6) | OQ8 | |
| ST12 | Hugging Face's HTTP 200 `failed` with `server_error` on a bad key or an image (PR4 P7, P9) | open *(2026-11-23: out of scope by the owner's answers; restated in the amendment)* | under OQ3 (1) out of scope; under OQ3 (2) the message-status rule | OQ3 | |
| ST13 | `reasoning_tokens` 0 while reasoning items exist (PR4 P1, PR7 (a)) | decided | passed through as reported, zero; no estimate | `Usage`'s doc: zero when the service reports none (C45) | `TestResponses_ReasoningTokensAsReported_0091`; plant: a count estimated from the reasoning text |
| ST14 | A `completed` reply whose `output_tokens` reached the limit sent, no call cut (PR4 P3, PR5 P3, PR7 (e), PR8) | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: a cut (OQ6 (1)), only on a provider that moves and was measured hiding one (OQ7 (1)): none under the recommended answers | OQ6, OQ7 | |
| ST15 | A `completed` reply holding a call whose arguments are not a JSON object (PR6) | open *(2026-11-23: decided by the owner's answers; restated in the amendment)* | recommended: a cut when `output_tokens` reached the limit sent (OQ6 (1)), only on a provider that moves and was measured hiding one (OQ7 (1)): none under the recommended answers; the treatment is 0088's | OQ6, OQ7 | |
| ST16 | `[DONE]` after the terminal event (PR2) | decided | no change: skipped, and the reader stops at the terminal event (C11) | C11 | `TestResponses_DoneAfterTerminal_0091`, one stream ending `response.completed` then `[DONE]`, one ending `response.incomplete` then `[DONE]`; plant: `[DONE]` decoded as JSON |
| ST17 | `previous_response_id` refused (PR1, PR2, PR3 P8) or silently ignored (PR4, PR5 P8) | decided | a provider that moves declares `Continuation` `Unsupported`, so a request naming one is refused before the network (C46), as OpenCode's is (C60) | the measurements; C46 | `TestResponses_NoContinuation_0091`, per moved provider; plant: `Continuation` `BestEffort`, so the request reaches the server |
| ST18 | `store` and `include` on a moved provider | decided | `store: false` and `include: ["reasoning.encrypted_content"]` on every request, as the probes sent them and as OpenCode's route sends `store` (C61); OpenRouter refuses `store:true` | the measurements; 0012-MADR §3.2 | the moved provider's request golden; plants: `store` omitted; `include` omitted |
| ST19 | Error shapes on a moved provider: Kilo's 401 `INVALID_TOKEN` with a top-level `error_type`, a 400 whose `error` is a string, Together's `invalid_request_error` / `invalid_api_key`, OpenRouter's numeric code | decided | classified by HTTP status, as on Chat today (C58); the body parser already reads `error_type` (C59) | C58, C59 | `TestResponses_ErrorShapes_0091`, the probe shapes as fixtures (codes only); plant: a 401 read as `ErrProviderUnavailable` |
| ST20 | Together's four-model list | open *(2026-11-23: out of scope by the owner's answers; restated in the amendment)* | moot under OQ2 (1) | OQ2 | |
| ST21 | An Ollama server without `/v1/responses` (below 0.13.3 by the docs; codex gates 0.13.4) | open *(2026-11-23: out of scope by the owner's answers; restated in the amendment)* | moot under OQ4 (1); under (2): Chat below 0.13.4 or when unread, by `GET /api/version` once per base URL per process (ST48); under (3): refused below 0.13.4 | OQ4 | |
| ST22 | Image input on `/v1/responses`: Kilo P9 404, Together P9 not shown, Hugging Face P9 `failed`; Ollama 200 on both wires, status only (PR7 (b)) | decided: gate, outcome by probe *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | under OQ4 (2): pass, the answer names the picture's colour on Responses as on Chat, and ST39 is built; fail, Ollama stays on Chat. Moot under OQ4 (1) | OQ4 (2); B1 G2 | `TestLive_ResponsesCut_0091`'s Ollama image case records the outcome; on pass, `TestOllama_ResponsesImages_0091` (ST39); plant: the gate read as passed on a recorded fail |
| ST23 | `Generate` with no limit named, on a moved provider | decided | read over the stream, as on every provider (C47) | 0088-MADR ST88 | the moved provider's `Generate` golden over the stream; plant: a plain request |
| ST24 | A limit refusal on a moved provider | decided | as 0089-MADR DL68: not read on Responses (C48); Ollama's Chat route is not among the wires that read one, and Ollama sends no limit by default (C43, C44), so nothing is lost | 0089-MADR DL68 | no plant can exist: no code changes |
| ST25 | How a reply read as cut by the check is reported | decided, conditional on 0088 *(2026-11-23: restated in the amendment)* | by 0088's cut answer, under the rules its Phase 3 review sets for R1 to R3: finishing `length` with its text and whole calls, the cut call dropped and named in `WarningToolCallCut`, or `ErrEmptyAnswer` with nothing usable; reopens if that ruling leaves a `completed` reply no path into `CutAnswer` | 0088-MADR OQ11 | `TestResponses_CompletedAtLimit_0091`'s report cases, plain and stream: a text reply at the limit, a reply with a whole call and a cut call, a reply with only a cut call; plants: the text reply returned finishing `stop`; the cut call kept in the output; the cut call dropped with no `WarningToolCallCut`; the only-a-cut-call reply returned as an empty response, not `ErrEmptyAnswer` |
| ST26 | The probes (B4) | decided *(2026-11-23: restated in the amendment)* | rounds 1 to 7 (2026-11-23) are the record's measurements, committed in the REPORT (ST27). The PLAN's live test re-runs only what an answer needs: UM1 and UM1b under OQ4 (2); UM3, UM4, UM6 under the answers B4 names; under the recommended answers, none. Allowlisted fields only, under `tmp/live-0091/` | the owner's ruling on live runs (agents, environment keys only); AGENTS.md (C56) | `TestLive_ResponsesCut_0091`, vetted by CI, run by the agent; its allowlist pinned by an offline twin; plant: a raw body written to the output |
| ST27 | The research and the probe facts | decided | committed as a REPORT of this record's number in the PLAN's first phase, so this record's PR ids resolve in the tree | the records rule | `make records-check`; plant: the REPORT unindexed |
| ST28 | Docs | decided *(2026-11-23: restated in the amendment)* | as B5; the Hugging Face doc restated whatever OQ3's answer | B5; R43 | `make docs-check parity-check`, and a claims row per changed doc line; plant: the old "undocumented" line kept |
| ST29 | The exported API | decided *(2026-11-23: restated in the amendment)* | nothing new under the recommended answers; one scoped option per opt-in answer | R55 (C53) | `make api-check`, `tools/apidiff.py v1.32.0`; plant: an option added on a scratch clone |
| ST30 | The release | decided *(2026-11-23: restated in the amendment)* | v1.33.0. Under the recommended answers, docs and the REPORT only, with no code, so v1.33.0 does not wait on 0088's Phase 3 review for this record; under an answer that moves a provider, built after 0088's phases and Phase 3 review, serially in the same tree, and the tag waits for it (B5). Entries under `[Unreleased]` | the owner's ruling "Research and adopt in v1.33.0" | `make changelog-check`; plant: an entry outside `[Unreleased]` |
| ST31 | 0004-MADR options 8 and 12, 0005-MADR Q5, and 0031-MADR E1, where a provider moves | decided *(2026-11-23: restated in the amendment)* | each gets a dated annotation pointing here, in the phase that moves the provider | the records rule (no silent rewrite) | a claims row per annotation, planted false |
| ST32 | UM5, Chat Completions' finish for a cut inside a call on Kilo and Hugging Face | out of scope | evidence for 0088's Phase 3 review (B3); this record runs no Chat probe on them | B3 | none: out of scope |
| ST33 | Where the cut check runs, and how the limit sent reaches it | decided *(2026-11-23: restated in the amendment)* | inside the Responses decoder, in the plain decode and in `streamEnd` before held calls are released and before `EventDone`; the limit sent is a per-request field of the internal `responses.Decoder` (C80), set by the provider from the value it writes to `max_output_tokens`; none written, none set, no check. That value is the attempt's own: the limit left in the built body after `DropLimit` removes one not sent (C84), as `LimitRetry.Sent` records it (C85), set again on every attempt. On Responses no limit refusal is read (0089-MADR DL68, C48), so today no resend changes it; a later record that adds one gets the per-attempt value without change here. Not in the limit rule after decoding (C67 to C69), which on a stream sees `EventDone` after the held calls went out (W6) | B2; C67 to C69, C80 | `TestResponses_CompletedAtLimit_0091`, plain and stream, a text cut and a call cut at the limit, a reply under the limit unchanged, and a request whose limit was dropped (no check); plants: the field left unset (no cut read); the check moved after `held.Done` (the stream yields the cut call); the field set from the configured limit rather than the body's |
| ST34 | A streamed call whose arguments are not whole, on a covered provider | decided, conditional on 0088 *(2026-11-23: restated in the amendment)* | held until the stream ends by 0088's R3 (`wire.HeldCalls`, W3), so the check at `streamEnd` drops it before it is yielded; if 0088's Phase 3 ruling withdraws or changes R3's hold, this row reopens and the PLAN holds such calls on the covered provider's stream itself | 0088-MADR tenth amendment, R3 (W5) | `TestResponses_CompletedAtLimit_0091`'s stream call case: the cut call never yielded; plant: the call yielded as its `output_item.done` arrives |
| ST35 | A refusal of thinking by an Ollama model that does not think, on `/v1/responses` | decided: gate failed (round 6) | measured (PR9): Responses answers 200 `completed` with the effort silently ignored, where Chat answers 400 in the form the SDK learns (C70, C71); so on Responses the refusal learning never fires and no `WarningReasoningLearned` or `WarningEffortDegraded` tells the caller the effort was dropped. By revision 2's gate rule, Ollama stays on Chat (OQ4 (1)). Under OQ4 (2), reasoning goes only to a model whose cached facts say it thinks (the listing reads the `thinking` capability from `/api/show`, C92, C93); a model whose facts say it does not is sent no effort, with a `WarningEffortDegraded` naming none; a model with no cached facts (a request never fetches them) is sent the effort as asked, and an effort it cannot take is dropped with no warning, the gap that remains. Under OQ4 (3), the drop stays silent for every model | PR9; revision 2's gate rule | under OQ4 (1), the Chat path's existing Ollama refusal tests (0061-MADR, amendment "Ollama's refusal of thinking, learned"); no plant can exist: no code changes. Under OQ4 (2), `TestOllama_ResponsesNoThinkingFacts_0091`; plant: an effort sent to a model with no thinking fact |
| ST36 | xhigh and max on `/v1/responses` (0062's ladder, C72) | decided: gate, outcome by probe *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | under OQ4 (2): pass, each is accepted by a thinking model and shows in its reasoning (round 6 shows a refusal is not sent, so an accepted effort cannot be told from an ignored one by status alone: the probe compares `output_tokens` at low and at max); fail, Ollama stays on Chat. Moot under OQ4 (1) | OQ4 (2); B1 G2 | the probe case records the outcome; on pass, `TestOllama_ResponsesEffortLadder_0091`; plant: max sent after a recorded refusal of it |
| ST37 | Tool choice on Ollama's Responses path | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | none sent; a named tool and `ToolChoiceRequired` offer the tools unforced; `ToolChoiceNone` sends no tools, as on Chat (C73) | C73 | the Responses request golden; plant: `tool_choice` sent |
| ST38 | `strict` on Ollama's Responses path | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | not sent, as on Chat (C74); P5's 200 shows it accepted, not honoured | C74; PR5 P5 | the Responses request golden; plant: `strict` sent |
| ST39 | Images on Ollama's Responses path | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | an image's bytes as `input_image` with a data URL; an image by URL dropped with `WarningImageDropped`, as on Chat (C81, C82) | C81, C82 | `TestOllama_ResponsesImages_0091`; plant: a URL image sent |
| ST40 | Ollama's Responses request body | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | built by a body function of the package's own, as each Responses provider has one (C75 to C77): `responses.Input`, ST18's `store` and `include`, the effort by ST36, `max_output_tokens` only when given (0089-MADR DL48), tools by ST37 and ST38; the limit sent passed to the decoder (ST33) | C75 to C77; 0089-MADR DL48 | the Responses request goldens, plain and stream; plant: `max_output_tokens` sent with no caller limit |
| ST41 | A text cut on Ollama: Chat says `length`, Responses says `completed` (PR7 (e), PR8) | open *(2026-11-23: out of scope by the owner's answers; restated in the amendment)* | moot under OQ4 (1); kept only by the cut check: OQ4 (2) or (3) needs OQ6 (1) or (3) and OQ7 (1) or (2) | OQ6, OQ7 | |
| ST42 | A cut inside a call on Ollama: unobserved after six models (PR7 (c), PR8) | decided *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | no gate waits on it: the check reads `output_tokens` against the limit sent wherever the cut falls (B2); the PLAN tries `deepseek-r1:7b` and a granite `q5_K_M` once, and records the result | B2; PR6 | `TestLive_ResponsesCut_0091`'s Ollama call case, recorded; no plant: a measurement, not a behaviour |
| ST43 | `gpt-oss:20b` fails every request with tools on both wires (PR8) | out of scope | a server or model defect, the same on Chat; G2 counts no difference | PR8 | none: out of scope |
| ST44 | Instructions on Ollama's Responses path | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | a leading system item, as the Chat path sends a leading system message | ollama.go's Degradations | the Responses request golden; plant: instructions dropped |
| ST45 | Hugging Face's stream cut in reasoning ends `response.failed` (PR4 P4) | open *(2026-11-23: out of scope by the owner's answers; restated in the amendment)* | under OQ3 (1) out of scope; under OQ3 (2) read as a retryable outage (C65, C66, C79), resent and billed again, unless a further rule is measured | OQ3 | |
| ST46 | Ollama's listing probe (one generation per listed model, C83) | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | runs through `Generate`, so on the wire ST4 and ST21 choose | C83 | the listing probe test on a fake server answering `/v1/responses`; plant: the probe sent to `/v1/chat/completions` on a server at or above 0.13.4 |
| ST47 | Ollama's encrypted reasoning item sent back on the next turn (round trip) | decided by round 6 | measured (PR9): the server answers 200 `completed` with the item as returned and with its `encrypted_content` altered by one character, so it does not validate the item, and a 200 shows neither that it uses the item nor that it ignores it. No replay gain is claimed for Ollama, and none counts toward OQ4; nor is a loss shown, so the item is no gate either way. Under OQ4 (2) or (3) the item is replayed as `responses.Input` replays it today (C6) | PR9 | under OQ4 (1), no plant can exist: no code changes. Under OQ4 (2) or (3), the Responses request golden carrying the item; plant: the item dropped from the replayed input |
| ST48 | Ollama's version read (`GET /api/version`, ST21) cancelled, timed out, or hidden | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | made with the request's context, bounded as the catalog's read is (5 s, C89). The request's context ended (a cancel or its deadline): the context's error is returned, nothing is cached and nothing is sent, as 0090-MADR has a cancelled probe stop rather than fall back (C90). The 5 s bound passed with the context alive, or the server unreachable: Chat for this request, nothing cached, so the next request reads again; at most one read in flight per base URL. An answer that is not 200, or a body with no parsable version (a proxy hiding it): Chat, cached for the process. A version below 0.13.4: Chat, cached; at or above: Responses, cached | ST21; 0090-MADR (C90) | `TestOllama_VersionRead_0091`, one case per outcome; plants: a cancel falling back to Chat and sending; a timeout cached as Chat; a hidden version read as Responses |
| ST49 | Deferred tools (`Tool.Deferred`) on Ollama's Responses path | decided, if Ollama moves *(2026-11-23: out of scope, OQ4's answer keeps Ollama on Chat; restated in the amendment)* | emulated as on Chat: `deferral.Other`, each loaded tool appended in load order, `WarningDeferralEmulated` (C86, C87); the Responses decoder is built with no `Search` and no `Dispatch` (C88, C91), so it reads no native search call. The native Responses form is OpenAI's from GPT-5.4 on, unmeasured on Ollama | C86, C87, C88; 0040-MADR D6 | the Responses request golden with a `ToolLoadItem`; plant: the native Responses deferral form chosen for Ollama |

Counts: 49 rows: 34 decided (ST6 to ST10, ST13, ST16 to ST19, ST22 to ST31, ST33 to ST40, ST42, ST44, ST46 to ST49; of them ST35 and ST47 decided by round 6's measurements, and ST22 and ST36 gates whose outcome the PLAN's probe sets under OQ4 (2)), 2 out of scope (ST32, ST43), 13 open (ST1 to ST5, ST11, ST12, ST14, ST15, ST20, ST21, ST41, ST45).

## Pros and Cons of the Options

### A. Move every provider that has a Responses route

* Good, because it is the literal reading of "as fully as possible": four more providers on the wire the SDK treats as canonical.
* Bad, because Hugging Face's auth failure, and its stream cut, become a retried `ErrProviderUnavailable` (PR4 P4, P7; C65, C66).
* Bad, because the hidden cuts on Kilo and Hugging Face need the cut check, which reads `output_tokens` only where a limit is sent, and Together, Hugging Face and Ollama send none by default (C43).
* Bad, because Kilo and Hugging Face lose reasoning replay (no `encrypted_content` even when asked; C6), and Kilo and Together lose 0089's refusal learning (C48).
* Bad, because Together's other models "may return malformed output", and Kilo's route is undocumented.
* Bad, because Ollama drops an effort its model cannot take without a word (PR9), where its Chat path learns and warns.

### B. Move each provider that passes three gates, and read a cut from evidence where a service was measured hiding it

* Good, because each move is justified by measurement, and each refusal to move names the measured loss.
* Good, because what a later move would take is decided now (the gates, the cut check, Ollama's Chat-path parity), so it needs only an amendment with its measurements.
* Neutral, because on this evidence the gates pass no provider, so in v1.33.0 B's code outcome is C's.
* Bad, because the decided rows for a move (ST33, ST34, ST37 to ST40, ST44, ST46, ST48, ST49) are record text with no build under the recommended answers, and a reader must see which rows apply only if a provider moves.

### C. Keep every Chat Completions provider, and harden only the shared decoder

* Good, because no provider's path changes.
* Neutral, because under the recommended answers it is B's outcome: no provider moves, and since round 6 no replay gain is forgone, as Ollama's is unproven (PR9, ST47).
* Bad, because it decides nothing about what a later move would take, so a provider that later passes the gates starts a new record; and Ollama's documented route and its reference client's only route stay unused.

### D. Chat Completions by default, Responses as an opt-in per provider

* Good, because a caller who wants the Responses wire can have it, and nobody else's path changes.
* Bad, because each option is an export with no outside call site (R55), and a caller who opts in meets every measured deviation of option A.
* Bad, because it is the cross-provider route option 0004-MADR option 14 warned against: Kilo's route would be a preference, OpenCode's a correctness requirement (C34).

## More Information

### Working-tree evidence (not checked by `tools/claims.py`)

The 0088-PLAN Phase 1b code and 0088-MADR's tenth amendment are uncommitted in the tree as this record is written. Their ids end in "now", as 0088-MADR's sixth and tenth amendments do, so that `tools/claims.py` does not read them against `self=247b745`; the PLAN's first step re-grounds them at its base.

| Id | Claim | Where (working tree, 2026-11-23) | Text |
| :--- | :--- | :--- | :--- |
| W1now | a plain reply is cut only when `incomplete` | `llmprovider/internal/wire/responses/responses.go:422` | `if raw.Status == statusIncomplete {` |
| W2now | a completed stream ends through `completed()`, not `CutAnswer` | `llmprovider/internal/wire/responses/responses.go:754` | `return streamEnd(result, event, false, refused, nil, &held, yield)` |
| W3now | the held calls the response kept are yielded at the end | `llmprovider/internal/wire/finish.go:104` | `kept := slices.ContainsFunc(resp.Output, func(item llmprovider.Item) bool { return item == call })` |
| W4now | `CutCall`: not a whole object, or a last `{}` | `llmprovider/internal/wire/finish.go:56` | `return !WholeObject(args) \|\| last && noArguments(args)` |
| W5now | R1 to R3 open for 0088's Phase 3 review | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1412` | `### Open for PLAN Phase 3's review (2026-11-23, tenth)` |
| W6now | `streamEnd` yields the held calls, then `EventDone` | `llmprovider/internal/wire/responses/responses.go:792` | `if held.Done(resp, yield) {` |
| W7now | the stream's last event gives only id, model and usage; the output comes from `output_item.done` | `llmprovider/internal/wire/responses/responses.go:778` | `result.Usage = event.Response.Usage.counts()` |

### Sources outside the repository

* The desk research (2026-11-23) and the probe agent's hand-backs for rounds 1 to 7, saved by the coordinator under `<scratch>/responses-research/`, to be committed as this record's REPORT (ST27). Vendor pages cited there: `https://kilo.ai/docs/gateway/api-reference`, `https://openrouter.ai/docs/api_reference/responses/overview.md`, `https://docs.together.ai/docs/inference/responses-api.md`, `https://huggingface.co/docs/inference-providers/guides/responses-api`, `https://docs.ollama.com/api/openai-compatibility`, `https://platform.claude.com/docs/en/api/openai-sdk`, `https://aistudio.google.com/docs/openai`.
* The probe script, `<scratch>/responses-probe/probe.py`, is not committed; its P6 body sends `include: ["reasoning.encrypted_content"]` (line 272). The PLAN's live test (ST26) replaces it.

### What could not be established

* Whether Kilo's whole gateway sits on OpenRouter (the desk report found no source saying so); the OpenRouter probes are read as the shapes such a gateway sends, not as Kilo's.
* Every item of B4.
* Whether Ollama's local models use an image they answer 200 to (PR7 (b), status only): ST22's gate item.
* The text of a Hugging Face `failed` reply's message after 2026-08-29, and the code of its stream's `response.failed` on a cut: the probes keep no message and recorded no code there.
* Whether Ollama's Chat route reports reasoning tokens: round 3 measured Responses only.

### Evidence index

<!-- claims: self=247b745 codex=57e64682f8 kilocode=58072bd696 -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | one Responses decoder, shared | `llmprovider/internal/wire/responses/responses.go:1` | `Responses wire, shared by the openai and` |
| C2 | reasoning text read from the summary only | `llmprovider/internal/wire/responses/responses.go:554` | `for _, s := range out.Summary {` |
| C3 | an item with no summary text and no encrypted content is dropped | `llmprovider/internal/wire/responses/responses.go:561` | `if sb.Len() > 0 \|\| out.EncryptedContent != "" {` |
| C4 | the one reasoning delta mapped | `llmprovider/internal/wire/responses/responses.go:756` | `"response.reasoning_summary_text.delta": llmprovider.EventReasoningDelta,` |
| C5 | unmapped `response.*` events are skipped | `llmprovider/internal/wire/responses/responses.go:777` | `return bytes.HasPrefix(t, []byte("response.")) && !handled[string(t)]` |
| C6 | plain text reasoning is not replayed | `llmprovider/internal/wire/responses/responses.go:169` | `this wire only (0021-MADR W7); plain text reasoning cannot be` |
| C7 | `failed` is read first | `llmprovider/internal/wire/responses/responses.go:407` | `case statusFailed:` |
| C8 | and classified by its error object | `llmprovider/internal/wire/responses/responses.go:409` | `return nil, failed(provider, raw.Error)` |
| C9 | a numeric code is classified as that status | `llmprovider/api_error.go:484` | `if status, err := strconv.Atoi(code); err == nil && status >= http.StatusBadRequest && status <= 599 {` |
| C10 | a `server_error` type is a retryable outage | `llmprovider/api_error.go:532` | `case "overloaded_error", "api_error", "server_error":` |
| C11 | `[DONE]` is skipped | `llmprovider/internal/wire/responses/responses.go:678` | `if string(payload) == "[DONE]" \|\| ignored(payload, handled) {` |
| C12 | reasoning tokens as the service reports them | `llmprovider/internal/wire/responses/responses.go:812` | `ReasoningTokens: u.OutputDetails.Reasoning, CachedTokens: u.InputDetails.Cached,` |
| C13 | Kilo uses Chat Completions only | `llmprovider/providers/kilo/kilo.go:13` | `Only POST {base}/chat/completions is used. Kilo also answers /responses and` |
| C14 | why | `llmprovider/providers/kilo/kilo.go:16` | `where the Responses decoder does not read it. One route reaches the whole` |
| C15 | Kilo replays `reasoning_details` on Chat | `llmprovider/providers/kilo/kilo.go:94` | `Its reasoning_details replay is unchanged.` |
| C16 | Kilo's images are Supported on Chat | `llmprovider/providers/kilo/kilo.go:26` | `Images are Supported: an ImageItem goes as a Chat Completions image_url` |
| C17 | Hugging Face's `/responses` deliberately not used | `llmprovider/providers/huggingface/huggingface.go:10` | `also exists, but it is deliberately NOT used: it is undocumented, the` |
| C18 | the stated reason | `llmprovider/providers/huggingface/huggingface.go:15` | `without re-measuring.` |
| C19 | the image path pinned live on Chat | `llmprovider/providers/huggingface/huggingface.go:28` | `and an https URL is not documented; a live test pins it (0061-PLAN Phase` |
| C20 | Hugging Face sends no limit by default | `llmprovider/providers/huggingface/huggingface.go:77` | `The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| C21 | Together uses Chat Completions only | `llmprovider/providers/together/together.go:9` | `Only POST {base}/chat/completions is used. The wire shapes are from` |
| C22 | Together replays reasoning under the interleaved field | `llmprovider/providers/together/together.go:75` | `Prior reasoning is replayed under the field the model's interleaved.field` |
| C23 | Ollama generates on Chat Completions | `llmprovider/providers/ollama/ollama.go:11` | `Generation uses POST {base}/v1/chat/completions. Listing stays on the native` |
| C24 | measured on v0.31.1 | `llmprovider/providers/ollama/ollama.go:63` | `The wire shapes were measured against a running v0.31.1 on the date` |
| C25 | Ollama's images are BestEffort on Chat | `llmprovider/providers/ollama/ollama.go:21` | `Images are BestEffort: an ImageItem's bytes go as a Chat Completions` |
| C26 | chatcompat is a Chat Completions provider | `llmprovider/providers/chatcompat/chatcompat.go:1` | `Package chatcompat is a provider for any server that speaks OpenAI's Chat` |
| C27 | OpenCode's route per model | `llmprovider/providers/opencode/opencode.go:15` | `The route for a request is WithRoute's, else the model's provider.npm in the` |
| C28 | the precedent for a scoped wire option | `llmprovider/providers/opencode/opencode.go:213` | `func WithRoute(route Route) llmprovider.Option {` |
| C29 | OpenAI is on Responses | `llmprovider/providers/openai/openai.go:1` | `Package openai is the OpenAI provider (0015-MADR D2): the Responses API at` |
| C30 | Grok is on Responses | `llmprovider/providers/grok/grok.go:1` | `Package grok is the xAI Grok provider (0015-MADR D2): the Responses API at` |
| C31 | 0004: Kilo's `/responses` reasoning is misplaced | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:536` | `` Kilo's `/responses` reasoning is in a non-standard place `` |
| C32 | 0004: Kilo translates, so one route reaches every model | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:454` | `Kilo translates wire formats; it does not route by model.` |
| C33 | 0004 chose Kilo on Chat | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:1331` | `` ### 12. Kilo on `/chat/completions` only (chosen) `` |
| C34 | 0004: a Kilo route option buys nothing | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:1362` | `Bad, because it buys nothing.` |
| C35 | 0004: Hugging Face's `/responses` hazardous | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:384` | `A Responses endpoint exists but is undocumented and hazardous.` |
| C36 | the 2026-08-29 failed reply's message | `docs/decisions/0004-MADR-add-gateway-llm-providers.md:390` | `"error":{"code":"server_error","message":"401` |
| C37 | 0012 §3.1: OpenCode's route from each model's `provider.npm` | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:376` | `` * The route for a Zen/Go model is derived from the per-model `provider.npm` in `` |
| C38 | 0005: Ollama on Chat Completions | `docs/decisions/0005-MADR-canonicalize-llm-provider-configuration.md:494` | `` OpenAI-compatible `/v1/chat/completions` for generation `` |
| C39 | 0017: Together on the Chat primitive | `docs/decisions/0017-MADR-together-provider-and-auth-extensions.md:104` | `` `finish_reason: "eos"` is a normal stop `` |
| C40 | 0031: three providers on Responses | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:44` | `**Three providers speak the Responses wire:**` |
| C41 | 0014: Gemini on Interactions | `docs/decisions/0014-MADR-gemini-wire-fidelity.md:9` | `# Complete Gemini's Move to the Interactions API` |
| C42 | 0031: the reasoning-text delta not mapped | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:259` | `` `response.reasoning_text.delta` was not seen and is not mapped. `` |
| C43 | 0089: nothing sent on an optional wire | `docs/decisions/0089-MADR-model-output-limit-discovery.md:524` | `\| DL47 \| No caller value, nothing known, optional field \|` |
| C44 | 0089: Ollama then unbounded | `docs/decisions/0089-MADR-model-output-limit-discovery.md:574` | `Ollama: unbounded; loses today's 8192 runaway guard` |
| C45 | a count not reported is zero | `llmprovider/contract.go:181` | `none (0015-MADR D3, D4)` |
| C46 | continuation refused before the network when Unsupported | `llmprovider/contract.go:250` | `{req.PreviousResponseID != "", c.Continuation, "continuation"},` |
| C47 | `Generate` over the stream when no limit is named | `llmprovider/internal/wire/overstream.go:20` | `func OverStream(req *llmprovider.Request, given bool) bool {` |
| C48 | 0089: refusal forms by wire, none on Responses | `docs/decisions/0089-MADR-model-output-limit-discovery.md:545` | `\| DL68 \| Which wires read which refusal form (new) \|` |
| C49 | kilocode names an OpenRouter-compatible endpoint | `kilocode:packages/kilo-gateway/src/api/constants.ts:65` | `OpenRouter-compatible endpoint (chat, responses, messages, models)` |
| C50 | kilocode wraps OpenRouter's provider | `kilocode:packages/kilo-gateway/src/provider.ts:1` | `@openrouter/ai-sdk-provider` |
| C51 | codex's Ollama floor | `codex:codex-rs/ollama/src/lib.rs:47` | `Version::new(0, 13, 4)` |
| C52 | per-model decisions from facts, not ids | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:208` | `Per-model decisions come from the model facts, not from ids (0041-MADR D1).` |
| C53 | R55 | `docs/guides/api-standards.md:394` | `**R55. A new export is justified.**` |
| C54 | R45 | `docs/guides/api-standards.md:332` | `**R45. The request on the wire does not change**` |
| C55 | Kilo's Chat route finished `length` in the gobble report | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:29` | `` * gobble-cli ran a large multi-file turn on Kilo's `kilo-auto/balanced` `` |
| C56 | a record's own live measurement | `AGENTS.md:265` | `A record's own measurement test is` |
| C57 | the SDK already reads Ollama's version endpoint | `llmprovider/catalog/discovery.go:1079` | `req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/version", http.NoBody)` |
| C58 | errors classified by HTTP status | `llmprovider/api_error.go:290` | `func ClassifyHTTPError(provider string, resp *http.Response) error {` |
| C59 | the parser reads a top-level `error_type` | `llmprovider/api_error.go:778` | `` ErrorType string          `json:"error_type"` `` |
| C60 | OpenCode refuses continuation: its gateway rejects the id | `llmprovider/providers/opencode/opencode.go:25` | `Unsupported: the gateway rejects previous_response_id with HTTP 400` |
| C61 | 0012 §3.2: OpenCode's Responses route sends `store: false` | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:397` | `` **Responses route:** sends `store: false` `` |
| C62 | Hugging Face replays reasoning under the interleaved field | `llmprovider/providers/huggingface/huggingface.go:82` | `Prior reasoning is replayed under the field the model's interleaved.field` |
| C63 | Ollama replays reasoning under the interleaved field | `llmprovider/providers/ollama/ollama.go:84` | `Prior reasoning is replayed under the field the model's interleaved.field` |
| C64 | the response carries the limit sent | `llmprovider/providers/huggingface/huggingface.go:81` | `event, carry the limit sent in OutputLimit.` |
| C65 | another code is classified as a 500 | `llmprovider/api_error.go:499` | `e.terminal, e.Kind = classifyAPIError(serviceOf(provider), http.StatusInternalServerError, env, nil, false)` |
| C66 | a 500 is a retryable outage | `llmprovider/api_error.go:636` | `case status >= http.StatusInternalServerError:` |
| C67 | `OutputLimit` is set by the limit rule after decoding | `llmprovider/internal/wire/limit_rule.go:237` | `resp.OutputLimit = r.Report()` |
| C68 | on `Generate`, after `wire.Send` returns | `llmprovider/providers/openai/openai.go:350` | `limit.Finish(resp, p.observe(req, false))` |
| C69 | on `Stream`, by wrapping the decoded stream | `llmprovider/providers/openai/openai.go:387` | `stream = limit.FinishStream(stream, p.observe(req, true))` |
| C70 | Ollama learns refusals through the Chat matcher | `llmprovider/providers/ollama/ollama.go:227` | `chatcompletions.MatchRefusal)` |
| C71 | the refusal of thinking it learns | `llmprovider/providers/ollama/ollama.go:46` | `` (`"<model>" does not support thinking`): that refusal is learned, with a `` |
| C72 | Ollama's effort ladder, xhigh and max included | `llmprovider/providers/ollama/ollama.go:40` | `The effort, xhigh and max included, is sent by the model's facts and` |
| C73 | Ollama sends no tool choice | `llmprovider/providers/ollama/ollama.go:36` | `No tool choice is sent: a named tool and ToolChoiceRequired offer the` |
| C74 | Ollama sends no `strict` | `llmprovider/providers/ollama/ollama.go:34` | `A tool's Strict is not sent: Ollama's acceptance of function.strict` |
| C75 | OpenAI builds its own Responses body | `llmprovider/providers/openai/openai.go:483` | `// body is the Responses request for req.` |
| C76 | Grok builds its own | `llmprovider/providers/grok/grok.go:369` | `// body is the Responses request for req, with its model's facts, at the` |
| C77 | OpenCode builds its own | `llmprovider/providers/opencode/opencode.go:756` | `func (p *provider) responsesBody(c call, learner *wire.Learner)` |
| C78 | the Responses refusal matcher, OpenAI's and xAI's forms | `llmprovider/internal/wire/responses/refusal.go:28` | `func MatchRefusal` |
| C79 | a `failed` with no error object is retryable | `llmprovider/internal/wire/responses/responses.go:640` | `return llmprovider.ClassifyStreamFailure(provider, "", "", "response.failed event received")` |
| C80 | the decoder's per-request fields | `llmprovider/internal/wire/responses/responses.go:349` | `type Decoder struct {` |
| C81 | Ollama drops an image by URL | `llmprovider/providers/ollama/ollama.go:29` | `An image by URL is not sent: Ollama marks image URLs unsupported, so it` |
| C82 | Ollama sends an image's bytes as a data-URL string | `llmprovider/providers/ollama/ollama.go:22` | `image_url part whose image_url is the data URL alone, a string, as Ollama` |
| C83 | Ollama's listing probes each model with one generation | `llmprovider/providers/ollama/ollama.go:55` | `ListModels returns the installed models, probing each listed model with one` |
| C84 | `DropLimit` removes a limit not sent | `llmprovider/internal/wire/limit_rule.go:159` | `func DropLimit(body map[string]any, l llmprovider.SentLimit, path ...string) {` |
| C85 | `LimitRetry.Sent` records the value the built body carries | `llmprovider/internal/wire/limit_rule.go:173` | `// Sent records the value the built body carries, after a thinking budget` |
| C86 | Ollama emulates deferred tools with `deferral.Other` | `llmprovider/providers/ollama/ollama.go:345` | `Tools:                deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)), ToolChoice: req.ToolChoice, NoToolChoice: true,` |
| C87 | no native form on Ollama | `llmprovider/providers/ollama/ollama.go:69` | `Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):` |
| C88 | the Responses decoder's native search field | `llmprovider/internal/wire/responses/responses.go:355` | `Search string` |
| C89 | the catalog's Ollama version read is bounded at 5 s | `llmprovider/catalog/discovery.go:1073` | `ctx, cancel := context.WithTimeout(ctx, 5*time.Second)` |
| C90 | 0090: a cancelled run does not ask about the endpoint | `docs/decisions/0090-MADR-a-cancelled-run-does-not-ask-about-the-endpoint.md:11` | `# A Cancelled Run Does Not Ask About the Endpoint` |
| C91 | the Responses decoder's dispatch field | `llmprovider/internal/wire/responses/responses.go:360` | `Dispatch map[string]bool` |
| C92 | Ollama's listing reads the `thinking` capability from `/api/show` | `llmprovider/catalog/listing_metadata.go:435` | `b.info.Reasoning.Supported = slices.Contains(caps, ollamaThinking)` |
| C93 | the listing's `/api/show` read | `llmprovider/catalog/listing_metadata.go:382` | `func fetchOllamaShow(ctx context.Context, token llmprovider.Token, cfg config, model string) (ModelInfo, error) {` |

## Amendment (2026-11-23): the owner's answers to OQ1 to OQ8, and what OQ5's opt-in needs

Additions only; the rows above keep their text, each answered row annotated where it stands.

### Owner rulings (2026-11-23)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ1 | "Chat stays (Recommended)" | (1): Kilo stays on Chat Completions | ST1 |
| OQ2 | "Chat stays, every model (Recommended)" | (1): Together stays on Chat Completions for every model | ST2, ST20 |
| OQ3 | "Chat stays (Recommended)" | (1): Hugging Face stays on Chat Completions; its package doc is brought up to date | ST3, ST12, ST45 |
| OQ4 | "Chat stays (Recommended)" | (1): Ollama stays on Chat Completions | ST4, ST21, ST22, ST36 to ST42, ST44, ST46, ST48, ST49 |
| OQ5 | "Opt-in WithResponses()" | (2), not the recommended one: a new exported `chatcompat.WithResponses()` for a server the caller knows speaks Responses; no server measured; R55's cost, one export with no outside call site, is accepted by this answer, whose option stated it | ST5, ST50 to ST62 |
| OQ6 | "Output tokens at the limit (Recommended)" | (1): a `completed` reply whose `output_tokens` reached the limit sent is a cut | ST14, ST15 |
| OQ7 | "Only on a moved, measured provider (Recommended)" | (1): the check runs only on a provider that moves and was measured hiding a cut | ST14, ST15, ST55 |
| OQ8 | "Probe first" | (2), not the recommended one: plain-text reasoning is replayed, as a reasoning item with `reasoning_text` content, only on a moved provider that returns it and only once a probe shows the service takes it | ST11; with OQ5's answer, raises OQ9 (ST63) |

The answers were checked against each other. OQ1 to OQ4 move no provider, so the only provider that moves is `chatcompat`, and only for a caller who gives `WithResponses()`. It was not measured hiding a cut, because no server reached through it was measured as such; so under OQ7's answer the check runs on no provider in v1.33.0, and OQ6's answer decides the signal for a later amendment that moves a measured provider. OQ8's answer has effect only through `chatcompat`, and names no server to probe: OQ9.

### Rows restated (2026-11-23)

| Row | Status now | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- |
| ST1, ST2, ST3, ST4 | decided | Kilo, Together (every model), Hugging Face and Ollama stay on Chat Completions; each package doc states the 2026-11-23 measurement that keeps it there (ST28) | OQ1 to OQ4's answers | the existing wire goldens of each provider; no plant can exist for the wire: no code changes. The docs' plants are ST28's |
| ST5 | decided | Chat Completions by default; Responses for a caller who gives `chatcompat.WithResponses()` (ST50 to ST62) | OQ5's answer | as ST50 to ST62 name |
| ST11 | decided | plain-text reasoning is replayed only on a moved provider that returns it, once a probe shows its service takes a reasoning item with `reasoning_text` content; on `chatcompat`, OQ9 | OQ8's answer | as OQ9's answer names; for no provider else is anything built |
| ST14, ST15 | decided | the signal is OQ6 (1); the check runs only on a moved, measured provider (OQ7 (1)): none in v1.33.0, `chatcompat` not being measured (ST55). ST33 and ST34 stand for the amendment that moves a measured provider | OQ6 and OQ7's answers | no test or plant in v1.33.0: nothing is built; ST33's and ST34's tests are that amendment's |
| ST10 | decided, unchanged | built only for a moved provider measured sending `reasoning_text`: none; whether `chatcompat` reads it is OQ9 | ST10; OQ9 | as OQ9's answer names |
| ST12, ST20, ST21, ST41, ST45 | out of scope | Hugging Face, Together and Ollama stay on Chat | OQ2 to OQ4's answers | none: out of scope |
| ST22, ST36 to ST40, ST42, ST44, ST46, ST48, ST49 | out of scope | rows for an Ollama move; Ollama stays on Chat. They stand as the record of what such a move would take | OQ4's answer | none: out of scope |
| ST25 | decided, conditional on 0088 | unchanged; reached by no provider in v1.33.0 (ST14, ST15) | 0088-MADR OQ11 | the amendment that moves a measured provider |
| ST26 | decided | the measurements are rounds 1 to 7, in the REPORT (ST27). The PLAN runs only what OQ9's answer names | ST27; OQ9 | as OQ9's answer names |
| ST28 | decided | docs: the package docs of `kilo`, `together`, `huggingface` and `ollama` state the measurement that keeps each on Chat; `chatcompat`'s as ST62; `docs/architecture.md`, `README.md`, the Responses package doc (C1) naming `chatcompat` as its fourth user; a **Changed** entry is not needed (no behaviour changes) and an **Added** entry names `WithResponses` *(2026-11-26: no longer so. ST10, ST67 and ST71 change `openai`, `grok` and OpenCode's responses route, so a **Changed** entry names them; see "Amendment (2026-11-26, tenth)")* | B5; R43 | `make docs-check parity-check changelog-check`, and a claims row per changed doc line; plant: the Hugging Face doc's old "undocumented" line kept |
| ST29 | decided | one export: `chatcompat.WithResponses` (ST60) | OQ5's answer; R55 | ST60's |
| ST30 | decided | v1.33.0, the owner's ruling. It now carries code (`chatcompat`'s opt-in), which reads replies through the Responses decoder 0088 Phase 1b changes, so its phase is built after 0088's phases and Phase 3 review, serially in the same tree, and the tag waits for it; entries under `[Unreleased]` | the owner's ruling "Research and adopt in v1.33.0"; OQ5's answer | `make changelog-check`; plant: an entry outside `[Unreleased]` |
| ST31 | decided | `chatcompat` moves for an opted-in caller, so 0061-MADR's D4 ("a provider for any server that speaks Chat Completions") gets a dated annotation pointing here; no other record's statement changes, as no other provider moves | the records rule | a claims row on the annotation, planted false |
| ST33, ST34 | decided, not built in v1.33.0 | they stand for the amendment that moves a measured provider | OQ7's answer | that amendment's |

### What `chatcompat.WithResponses()` needs (2026-11-23)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST50 | A `chatcompat` provider built with `WithResponses()` | decided | `Generate` and `Stream` send `POST {base}/responses`; the listing (`GET {base}/models`) and everything else unchanged. Without the option, Chat Completions as today. A server that answers 404 or refuses the route gets its error back, classified by status: there is no fallback to Chat, the option being for "a server the caller knows speaks Responses" | OQ5's answer, as its option worded it | `TestChatcompat_WithResponses_0091` (a fake server records each path, with and without the option); plants: the option ignored, so `/chat/completions` is hit; a 404 resent to `/chat/completions` |
| ST51 | Its request body | decided *(2026-11-23, second: corrected; restated in the second amendment)* | built by a body function of the package's own, as each Responses provider has one (C75 to C77): `model`; `responses.Input`; instructions as a leading system item, as the Chat path sends a leading system message; `store: false` and `include: ["reasoning.encrypted_content"]` (ST18); `reasoning: {effort}` as asked, the server's model having no facts here (C26's package doc); `max_output_tokens` only when the caller gives one (0089-MADR DL48); tools with `strict` as the caller sets it, as on Chat; a tool choice in the Responses form (`required`, `none`, a named function); an image as `input_image`, its bytes as a data URL or its https URL | C75 to C77; ST18; 0089-MADR DL48 | the G-wire goldens, `Generate` and `Stream`, one with tools and an image; plants: `max_output_tokens` sent with no caller limit; `strict` dropped; `store` omitted |
| ST52 | Its reply | decided | read by the shared Responses decoder (`DecodeFor`, `Events`), labelled with the provider's own id as the Chat path's is (C100); built with no `Search` and no `Dispatch`; 0088's cut answer for an `incomplete` reply, as for every Responses provider | C100; C1 | `TestChatcompat_ResponsesDecode_0091`, plain and stream, on the shapes rounds 3 and 7 recorded (Ollama's and OpenAI's); plant: the reply labelled `responses` instead of the id |
| ST53 | Errors | decided | an HTTP error by its status (C58), as on Chat; a 200 `failed` by its error object, retryable when its code is no HTTP status (C65, C66, C79), as on every Responses provider. Whether any server reached this way answers 200 `failed` is unmeasured; Hugging Face's router does (PR4) | C58, C65, C66, C79 | `TestChatcompat_ResponsesErrors_0091`: a 401 is `ErrAuthFailure`; a 200 `failed` with `server_error` is `ErrProviderUnavailable`; plant: the 200 `failed` decoded as an empty success |
| ST54 | A refusal of the effort sent | decided | learned through the Responses wire's matcher (C78), as the Chat path learns through the Chat matcher (C96); a refusal in another form is returned, not learned. A server that ignores an effort, as Ollama's `/v1/responses` does (PR9), drops it with no warning: a cost the opting-in caller accepts, stated in the package doc | C78, C96; PR9 | `TestChatcompat_ResponsesRefusal_0091`: OpenAI's `invalid_value` form learned and resent lower; an unknown form returned; plant: the Chat matcher wired on the Responses path |
| ST55 | A cut the server reports as `completed` | decided *(2026-11-23, second: reopened, its premise false; OQ10 in the second amendment)* | **the check does not run on `chatcompat`**: it moves, but no server reached through it was measured hiding a cut, and OQ7's answer runs the check only on a moved, measured provider. A server that hides a cut, as Ollama's and Hugging Face's `/responses` do (PR4 P3, PR5 P3, PR7 (e)), has it returned as `stop` or `tool_calls`, with a call that does not parse returned as it came. The package doc says so; a later amendment that measures a server may change it | OQ7's answer | `TestChatcompat_ResponsesNoCutCheck_0091`: a `completed` reply at the limit sent is returned finishing `stop`; plant: the decoder given the limit sent, so it finishes `length` |
| ST56 | A refusal of the output limit | decided | read in the Responses wire's forms, as OpenCode's responses route reads them (C99), where the Chat path reads the Chat Completions forms (C98); 0089-MADR DL68 reads no Responses form, so on this path the Chat path's "supports at most" learning is lost: a cost stated in the package doc | 0089-MADR DL68 (C48); C98, C99 | `TestChatcompat_ResponsesLimitRefusal_0091`: the Chat "supports at most" text on the Responses path is returned, not resent; plant: the Chat Completions forms wired on the Responses path |
| ST57 | Deferred tools | decided | emulated as on the Chat path: `deferral.Other`, each loaded tool appended in load order, `WarningDeferralEmulated` (C97) | C97; 0040-MADR D6 | the golden with a `ToolLoadItem`; plant: the native Responses deferral form chosen |
| ST58 | Continuation, `Generate` with no limit named | decided | `Continuation` stays `Unsupported`, so a `PreviousResponseID` is refused before the network (C46; ST17); `Generate` with no limit named is read over the stream (C47; ST23) | ST17, ST23 | `TestChatcompat_ResponsesNoContinuation_0091`; plant: `Continuation` `BestEffort` |
| ST59 | Capabilities, and the conformance suite | decided | unchanged by the option: tools, images and native streaming `Supported`, forced tool choice and reasoning `BestEffort`, continuation `Unsupported`; `llmtest.Run` passes with the option against a scripted Responses server (R44) | R44; C26 | `llmtest.Run` on a provider built `WithResponses()`; plant: `Continuation` declared `Supported` |
| ST60 | The exported API | decided | one export, `chatcompat.WithResponses() llmprovider.Option`, built as `WithID` is, a shared option (C94, C95), since a `chatcompat` provider's id is the caller's. R55: its call sites are the package's `ExampleWithResponses` and its tests; no call site outside the module is named, and the owner's answer accepts that cost, its option having stated it. `tools/apidiff.py v1.32.0` lists it and nothing else from this record | OQ5's answer; R55 (C53) | `make api-check`, `python3 -B tools/apidiff.py v1.32.0`; plant: a second export added on a scratch clone |
| ST61 | The wizard, and listing probes | decided | the wizard does not offer the option: OQ5's answer is for "a server the caller knows speaks Responses", and the wizard's local servers (C101) are found, not known; a listing probe under `WithModelProbes(true)` goes through `Generate`, so on the Responses path when the option is given | OQ5's answer; C101 | `TestChatcompat_ResponsesListingProbe_0091`: the probe hits `/responses` with the option and `/chat/completions` without; plant: the probe pinned to Chat; no plant for the wizard: nothing is added there |
| ST62 | `chatcompat`'s package doc | decided | a paragraph on the option: what it sends and reads (ST50 to ST52); that no server was measured; what the caller gives up against the Chat path: a hidden cut not read (ST55), an effort a server ignores dropped silently (ST54), the "supports at most" limit learning (ST56), no fallback to Chat (ST50); and what OQ9's answer decides for reasoning | R43; ST55, ST54, ST56, ST50 | `make docs-check`; a claims row per sentence; plant: the "no server was measured" sentence removed |
| ST63 | Plain-text reasoning on `chatcompat`'s Responses path: a server that puts its reasoning in `content[]` as `reasoning_text` | open *(2026-11-23, third: decided by OQ9's answer)* | OQ9 | OQ9 | |

### New question (2026-11-23)

OQ5's answer moves `chatcompat` for an opted-in caller, and OQ8's answer replays plain-text reasoning only after a probe shows the service takes it. A `chatcompat` provider can point at any server, so no probe names its service, and today's decoder drops reasoning that arrives only as `reasoning_text` (C3; ST10 builds the read only for a measured provider). The one server measured through this path's shapes, Ollama, sends OpenAI-shaped reasoning with `encrypted_content` (PR7 (a)), which the path replays as `responses.Input` does today (C6), so the question concerns other servers.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ9. On `chatcompat`'s Responses path, what happens to reasoning a server returns only as plain text? | (1) **Read it, do not send it back**: the shared decoder reads `reasoning_text` content when the summary holds no text (ST10, built for this path), so the caller sees the reasoning; it is not replayed, since no server is probed (OQ8's answer). (2) **Neither read nor send it back**: ST10 as decided; such reasoning is dropped. (3) **Read it and send it back without a probe**, as a reasoning item with `reasoning_text` content. | (1) ST10's change reaches `openai`, `grok` and OpenCode through the shared decoder; none was seen sending `reasoning_text` (C42), so it adds and changes nothing for them; a reasoning model's next turn on such a server lacks its reasoning, as on the Responses wire today. (2) such a server's reasoning never reaches the caller, where the Chat path reads it. (3) sets aside OQ8's "Probe first" for this provider; an input form no record has measured, on every turn. | ST63, ST10 |

Counts after this amendment: 63 rows: 44 decided (ST1 to ST11, ST13 to ST19, ST23 to ST31, ST33 to ST35, ST47, ST50 to ST62), 18 out of scope (ST12, ST20 to ST22, ST32, ST36 to ST46, ST48, ST49), 1 open (ST63).

### Evidence added (2026-11-23)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C94 | `chatcompat`'s `WithID` | `llmprovider/providers/chatcompat/chatcompat.go:144` | `func WithID(id llmprovider.ProviderID) llmprovider.Option {` |
| C95 | built as a shared option | `llmprovider/providers/chatcompat/chatcompat.go:145` | `return llmprovider.SharedOption("chatcompat.WithID", compatid.Value(id))` |
| C96 | the Chat path learns refusals through the Chat matcher | `llmprovider/providers/chatcompat/chatcompat.go:285` | `chatcompletions.MatchRefusal)` |
| C97 | the Chat path emulates deferred tools | `llmprovider/providers/chatcompat/chatcompat.go:340` | `Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...),` |
| C98 | the Chat path reads the Chat Completions limit forms | `llmprovider/providers/chatcompat/limit.go:54` | `r := &wire.LimitRetry{Wire: wire.ChatCompletionsLimits, Model: model,` |
| C99 | OpenCode's responses route reads the Responses forms | `llmprovider/providers/opencode/opencode.go:1012` | `return wire.ResponsesLimits` |
| C100 | the Chat path labels replies with the provider's id | `llmprovider/providers/chatcompat/chatcompat.go:273` | `chatcompletions.DecodeFor(string(p.id)), wire.Retries(limit.Check, effort.Check))` |
| C101 | the wizard offers the local servers it finds | `llmprovider/providers/chatcompat/chatcompat.go:32` | `The wizard's Options.LocalServers offers them, each as a provider of this` |

## Amendment (2026-11-23, second): ST55 reopened, OQ5's losses put to the owner, ST51 corrected

*(2026-11-23, revised after a delta check: OQ11 (3) now takes one reading, the effort guard plus the replay guard, with each guard's true reach and cost; "images refused by facts" is dropped from it, as ST51 decides that for every option; OQ10 (3)'s cost now says that a fact's source names its layer but not the listing's dialect, so the option needs new plumbing or an inference.)*

Additions only. A review of the first amendment and the PLAN found:

* **ST55's premise is false.** It said no server reached through `chatcompat` was measured hiding a cut. `chatcompat` reaches Ollama: its listing reads Ollama's `/api/show` dialect (C106), and Ollama's `/v1/responses` was measured reporting a text cut as `completed` where its Chat route says `length` (PR5 P3, PR7 (e), PR8). ST55 is open again, as OQ10. Its plant ("the decoder given the limit sent") could not exist unless ST33 is built; each OQ10 option below names a plant that fits it.
* **OQ5's losses were never put to the owner.** OQ5's option (2) stated "one exported option (R55); no server measured", not what an opted-in caller gives up against the Chat path. Those losses are ST53, ST54, ST56 and the reasoning replay below. OQ11 puts them to the owner as a confirming question; ST64 holds the answer, and ST50 to ST62 are built only under OQ11 (1) or (3).
* **ST51 described the Chat path wrongly.** `chatcompat`'s Chat path reads the model's cached facts (`catalog.CachedFacts`, C102), sends the effort by 0062's ladder over the facts' efforts (C104), replays prior reasoning under the facts' `ReplayField` (C105), and refuses an image before the network when the facts take none (`wire.RefuseImages`, C103). ST51 is restated below.
* **W2now, W6now and W7now** cite `responses.go` lines that 0088 Phase 1b has since moved. They are left to PLAN Phase 0's re-ground at that phase's commit; no row of this record rests on their line numbers.

### Rows (2026-11-23, second)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST51 (restated) | The Responses path's request, against what the Chat path does | decided | the Responses body function is the package's own (C75 to C77), from the same cached facts the Chat path reads (C102): the effort by 0062's ladder over `facts.Reasoning.Efforts`, learned through the Responses matcher (C104, ST54); an image refused before the network when the facts take none, as on Chat (C103); `store: false` and `include` (ST18); `max_output_tokens` only when given; tools with `strict` as the caller sets it; the Responses tool-choice form; instructions as a leading system item. Prior reasoning: an item carrying `encrypted_content` is replayed (C6); reasoning the Chat path would replay under the facts' `ReplayField` (C105) is not, unless OQ11 (3) or OQ9 says otherwise | C102 to C105; C6; ST18 | the goldens of ST51 in the first amendment, plus: an image to a model whose facts take none refused before the network (plant: the refusal dropped on the Responses path); an effort outside the facts' list stepped down with `WarningEffortDegraded` (plant: the ladder bypassed) |
| ST55 | A cut the server reports as `completed`, on `chatcompat`'s Responses path | open *(2026-11-23, third: decided by OQ10's answer)* | OQ10 | OQ10 | |
| ST62 (restated) | `chatcompat`'s package doc on the option | decided | as the first amendment's ST62, its list of what an opted-in caller gives up being: a 200 `failed` (such as a bad key, as Hugging Face's router answers) retried as an outage (ST53); an effort the model cannot take dropped with no warning where the server ignores it, as Ollama's does (ST54); the "supports at most" limit learning (ST56); prior reasoning replayed under a facts-named field on Chat but not here, unless it carries `encrypted_content` (ST51); a hidden cut, as OQ10's answer leaves it (ST55); no fallback to Chat (ST50); what OQ9's answer does with plain-text reasoning | ST50 to ST56; OQ9, OQ10 | `make docs-check`; a claims row per listed loss; plant: the ST53 sentence removed |
| ST64 | OQ5's opt-in, with its losses stated | open *(2026-11-23, third: decided by OQ11's answer)* | OQ11; under its option (2), ST5 reverts to "Chat stays" and ST50 to ST63 are out of scope | OQ11 | |

Counts after this amendment: 64 rows: 43 decided, 18 out of scope, 3 open (ST55, ST63, ST64).

### Open questions, as the owner reads them (2026-11-23, second)

Each stands alone. OQ9 is restated from the first amendment in these words; its options and costs are unchanged.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ9. You chose an opt-in `chatcompat.WithResponses()` and "Probe first" for replaying plain-text reasoning. A `chatcompat` provider can point at any server, so no probe can name its service, and today's Responses decoder drops reasoning a server sends only as plain text (`content[]` of type `reasoning_text`). On that path, what should happen to such reasoning? | (1) **Read it, do not send it back**: the caller sees it; it is not replayed. (2) **Neither read nor send it back**: it is dropped, as the decoder does today. (3) **Read it and send it back without a probe**. | (1) a small, additive change to the decoder `openai`, `grok` and OpenCode share; none of them was seen sending such reasoning, so nothing changes for them; the model's next turn on such a server lacks its reasoning. (2) the caller never sees that server's reasoning, where the Chat path shows it. (3) sets aside "Probe first" for this provider; a request form nobody has measured, on every turn, which a server may refuse. | ST63, ST10 |
| OQ10. With `WithResponses()`, `chatcompat` can reach Ollama, whose `/v1/responses` reports a reply cut at the output limit as `completed`, where its Chat route says `length`. Should the SDK read such a reply as cut on that path? | (1) **Yes, by output tokens at the limit**: a `completed` reply whose `output_tokens` reached the limit the request sent is read as cut, the check you chose in OQ6. (2) **No**: such a reply is returned as `stop` or `tool_calls`, and the package doc says so. (3) **Only when the cached listing shows Ollama's dialect**: the check runs for a model whose facts came from Ollama's `/api/show`, and not otherwise. | (1) an answer that ends naturally on exactly the last token of the limit is reported `length` instead of `stop`; the check's machinery (the limit passed to the decoder, ST33, and its reliance on 0088's held calls on a stream, ST34) is built, for this path only; test: a reply at the limit finishes `length`, plain and stream; plant: the limit left unset for `chatcompat`. (2) a hidden cut comes back as success, and a cut call whose arguments do not parse comes back as a call; test: the doc sentence's claims row and a reply at the limit finishing `stop`; plant: the sentence removed. (3) as (1) for Ollama; a server of another dialect that hides cuts is not caught; and new plumbing: a model's facts record the layer that set each field (`SourceOf`, "listing:<provider>", C107), not which dialect the listing read, so either the listing records its dialect (new code in `catalog`) or the check infers Ollama from a fact only Ollama's `/api/show` dialect sets, such as the thinking capability, which a non-thinking Ollama model may lack; plant: the dialect test removed, so a non-Ollama reply at the limit finishes `length`. | ST55 |
| OQ11. Your opt-in `chatcompat.WithResponses()` was chosen without these costs in front of you. A caller who opts in gives up, against the Chat path: a 200 `failed` reply (such as a bad key, as Hugging Face's router sends one) is treated as a temporary outage and retried; an effort the model cannot take is dropped with no warning where the server ignores it, as Ollama's does; the learning of a refused output limit ("supports at most") is lost; and prior reasoning the Chat path replays under the model's named field is not sent back unless it carries `encrypted_content`. No server was measured gaining anything on this path. Keep the opt-in? | (1) **Keep it, every loss stated in the package doc.** (2) **Revert: `chatcompat` stays on Chat Completions only.** (3) **Keep it with two guards**: send an effort only to a model whose cached facts say it thinks, and to any other model send none, with a warning; and send on Chat, not Responses, any turn whose prior reasoning the model's facts say to replay under a named field. | (1) one export with no outside caller (R55); the losses fall only on a caller who opts in, and the doc names each; the work of ST50 to ST62 and OQ10's answer. (2) nothing is built and no export is added; a caller with a server that speaks only Responses cannot use `chatcompat` for it. (3) as (1), plus: "cached facts" are a caller's layer, the cached listing, or models.dev's facts of the model's canonical id, and a request never fetches them (C102, C108, C109); the listing states a thinking capability only for Ollama's `/api/show` dialect (C106), so on LM Studio, vLLM or llama.cpp a model that is in neither models.dev nor a caller's layer is never sent reasoning, only the warning; and for any model whose facts name a replay field, every turn after the first carries reasoning to replay and goes back to Chat, so the opt-in does nothing past the first turn for such a model, and those turns reach the Chat route of a server the caller chose for Responses. | ST64, ST5, ST50 to ST63 |

**Why OQ11 recommends (1).** Every loss falls only on a caller who passes the option, the owner chose the option knowing no server was measured, and (1) states each loss where that caller reads. (3)'s effort guard sends no reasoning at all to most models on servers other than Ollama, and its replay guard returns every later turn of a replaying model to Chat, so it trades the losses for the opt-in's own purpose. (2) is the safe reading of the evidence, since no gain was measured, but it reverses a choice the owner made with a stated cost, on losses that the option's default (off) already keeps from everyone else.

### Evidence added (2026-11-23, second)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C102 | the Chat path reads the model's cached facts | `llmprovider/providers/chatcompat/chatcompat.go:314` | `info, _ := catalog.CachedFacts(ctx, p.id, cmp.Or(req.Model, p.model),` |
| C103 | it refuses an image by facts | `llmprovider/providers/chatcompat/chatcompat.go:343` | `Refusal: wire.RefuseImages(cmp.Or(req.Model, p.model), req.Input, facts.Has(catalog.FieldInput),` |
| C104 | it sends the effort by the ladder over the facts' efforts | `llmprovider/providers/chatcompat/chatcompat.go:375` | `effort, warnings := learner.Effort(llmprovider.Effort(p.effort(req)), facts.Reasoning.Efforts,` |
| C105 | it replays under the facts' `ReplayField` | `llmprovider/providers/chatcompat/chatcompat.go:391` | `return facts.Reasoning.ReplayField` |
| C106 | its listing reads Ollama's `/api/show` dialect | `llmprovider/providers/chatcompat/chatcompat.go:86` | `a recorded dialect's: Ollama's /api/show, for each model's window, tools,` |
| C107 | a fact's source names its layer, not a dialect | `llmprovider/catalog/model_info.go:233` | `// SourceOf names the layer that set f: "models.dev", "listing:<provider>", or` |
| C108 | the facts a request reads, never fetching: caller layers first | `llmprovider/providers/chatcompat/chatcompat.go:82` | `holds of its model, never fetching: a caller's catalog layers` |
| C109 | then models.dev by canonical id | `llmprovider/providers/chatcompat/chatcompat.go:84` | `then models.dev's facts of the model's canonical id, from the cached` |

## Amendment (2026-11-23, third): the owner's answers to OQ9 to OQ11

Additions only.

### Owner rulings (2026-11-23, third)

Relayed by the coordinator; labels quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ9 | "Read it, don't send it back (Recommended)" | (1): on `chatcompat`'s Responses path, reasoning a server sends only as `reasoning_text` content is read and shown, and not replayed | ST63, ST10 |
| OQ10 | "Yes, output tokens at limit (Recommended)" | (1): on `chatcompat`'s Responses path, a `completed` reply whose `output_tokens` reached the limit sent is read as cut (OQ6's check); ST33 and ST34 are built for that path | ST55, ST33, ST34, ST25 |
| OQ11 | "Keep it, losses documented (Recommended)" | (1): the opt-in stands, each loss stated in the package doc (ST62) | ST64 |

The answers were checked against each other: OQ10 (1) is OQ6's signal on the one moved provider that reaches a server measured hiding cuts, as OQ7's answer allows; OQ9 (1) reads reasoning and leaves replay to a later probe, as OQ8's "Probe first" asks; OQ11 (1) keeps ST50 to ST62 as decided.

### Rows (2026-11-23, third)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST55 | A `completed` reply on `chatcompat`'s Responses path whose `output_tokens` reached the limit sent | decided | read as cut, and finished by 0088's cut answer (ST25): finishing `length` with its text and whole calls, a call whose arguments do not parse dropped and named; the limit sent reaches the decoder as ST33 decides, per attempt; on a stream the check relies on 0088's held calls (ST34); with no limit sent, no check | OQ10 "Yes, output tokens at limit (Recommended)" | `TestChatcompat_ResponsesCut_0091`, plain and stream: a text reply at the limit finishes `length`, a reply under it `stop`, a call that does not parse at the limit dropped with `WarningToolCallCut`; plant: the limit left unset for `chatcompat`, so the reply at the limit finishes `stop` |
| ST33 (built) | Where the check runs | decided | as ST33 states, built for `chatcompat`'s Responses path only: the limit sent is a per-request field of the internal decoder, set by `chatcompat`; `openai`, `grok` and OpenCode leave it unset, so their replies read as today | OQ10's answer; OQ7's answer | ST33's test and plants, run on `chatcompat`; plus `openai`'s existing decode tests unchanged (plant: the field set for `openai`, so a reply at its limit finishes `length`) |
| ST34 (built) | A streamed call whose arguments are not whole | decided, conditional on 0088 | as ST34 states, on `chatcompat`'s Responses path | OQ10's answer; 0088's Phase 3 ruling on R3 | ST34's test, run on `chatcompat` |
| ST63 | Reasoning a server sends only as `reasoning_text` content, on `chatcompat`'s Responses path | decided | read: the shared decoder takes `reasoning_text` content when the summary holds no text, and the stream yields `response.reasoning_text.delta` as `EventReasoningDelta` (ST10, built); the `ReasoningItem` carries the text and no encrypted content, so it is not replayed (C6) | OQ9 "Read it, don't send it back (Recommended)" | `TestResponses_ReasoningText_0091`, plain and stream, with an OpenAI-shaped summary item decoding unchanged; `TestChatcompat_ResponsesReasoningText_0091`: the text shown, and absent from the next request; plants: the decoder reading `summary[]` only; the `content` text preferred over a non-empty summary; the text item replayed on the next turn |
| ST10 (built) | The decoder's read of `reasoning_text` | decided | built, for the path ST63 decides; its reach into `openai`, `grok` and OpenCode adds and changes nothing, none being seen sending it (C42) | OQ9's answer | ST63's |
| ST64 | OQ5's opt-in, with its losses stated | decided | kept; ST50 to ST62 stand, ST62 naming each loss | OQ11 "Keep it, losses documented (Recommended)" | ST62's claims rows; plant: the ST53 sentence removed |

Counts after this amendment: 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Amendment (2026-11-24, fourth): the REPORT in the tree, and the rows re-grounded at `dd37a87`

Additions only; no row's decision changes. This is PLAN Phase 0, steps 2 and 3. The base is `dd37a87`, the commit after 0088-PLAN Phase 1b (`f4efc5e`), Phase 1c (`903c387`) and 0090-PLAN (`46f43d7`).

### The REPORT (ST27)

The desk research and the probe hand-backs of rounds 1 to 7 are committed as [0091-REPORT-responses-api-across-providers.md](../reports/0091-REPORT-responses-api-across-providers.md), which replaces the "Sources outside the repository" paragraph's scratch folders as the place this record's evidence lives. Its section "Where each row of 0091-MADR's probe table is read" maps PR1 to PR10 to the rounds and services that hold them; every measurement the table states is in it, in allowlisted form (status codes, field names, enum values, event names, counts, booleans). Its own evidence index, 17 rows, passes at `dd37a87` (`python3 -B tools/claims.py docs/reports/0091-REPORT-responses-api-across-providers.md`, exit 0). The probe script stays uncommitted, cited there as `<scratch>/responses-probe/probe.py`.

### Re-grounding at `dd37a87` (2026-11-24)

`python3 -B tools/claims.py docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md --rev self=dd37a87`: 82 of 109 pass, exit 1. With the record's own marker (`self=247b745`) all 109 pass, unchanged. The 27 rows that did not pass at `dd37a87` moved and none is gone: 26 by line, and C7 by text, its line unchanged (0088-PLAN Phase 1b, `f4efc5e`, rewrote the plain decode's `switch raw.Status` as a tagless `switch`, so the claim's `case statusFailed:` reads `case raw.Status == statusFailed:`, the status still read first, as its comment says). C66's text is on two lines at `dd37a87` (279 and 645); 645 is the one, by the nine-line shift its neighbours C65 and C9 took. The other 25 are the tool's `--fixes` (`<scratch>/p0091/fixes.json`).

The ids end in "now", as the sixth amendment of 0088-MADR's do, so that `tools/claims.py` does not read them against this record's `self=247b745`. On a scratch copy of this record with plain ids, each moved row replaced in place, the seven rows W1 to W7 below appended and the marker set to `self=dd37a87`, 116 of 116 rows pass, exit 0. The 27 moved rows:

| Id | Claim | Moved | Where | Text |
| :--- | :--- | :--- | :--- | :--- |
| C2now | reasoning text read from the summary only | line | `llmprovider/internal/wire/responses/responses.go:578` | `for _, s := range out.Summary {` |
| C3now | an item with no summary text and no encrypted content is dropped | line | `llmprovider/internal/wire/responses/responses.go:585` | `if sb.Len() > 0 \|\| out.EncryptedContent != "" {` |
| C4now | the one reasoning delta mapped | line | `llmprovider/internal/wire/responses/responses.go:881` | `"response.reasoning_summary_text.delta": llmprovider.EventReasoningDelta,` |
| C5now | unmapped `response.*` events are skipped | line | `llmprovider/internal/wire/responses/responses.go:902` | `return bytes.HasPrefix(t, []byte("response.")) && !handled[string(t)]` |
| C7now | `failed` is read first | text | `llmprovider/internal/wire/responses/responses.go:407` | `case raw.Status == statusFailed:` |
| C9now | a numeric code is classified as that status | line | `llmprovider/api_error.go:493` | `if status, err := strconv.Atoi(code); err == nil && status >= http.StatusBadRequest && status <= 599 {` |
| C10now | a `server_error` type is a retryable outage | line | `llmprovider/api_error.go:541` | `case "overloaded_error", "api_error", "server_error":` |
| C11now | `[DONE]` is skipped | line | `llmprovider/internal/wire/responses/responses.go:711` | `if string(payload) == "[DONE]" \|\| ignored(payload, handled) {` |
| C12now | reasoning tokens as the service reports them | line | `llmprovider/internal/wire/responses/responses.go:937` | `ReasoningTokens: u.OutputDetails.Reasoning, CachedTokens: u.InputDetails.Cached,` |
| C15now | Kilo replays `reasoning_details` on Chat | line | `llmprovider/providers/kilo/kilo.go:96` | `Its reasoning_details replay is unchanged.` |
| C28now | the precedent for a scoped wire option | line | `llmprovider/providers/opencode/opencode.go:217` | `func WithRoute(route Route) llmprovider.Option {` |
| C37now | 0012 §3.1: OpenCode's route from each model's `provider.npm` | line | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:382` | `` * The route for a Zen/Go model is derived from the per-model `provider.npm` in `` |
| C45now | a count not reported is zero | line | `llmprovider/contract.go:184` | `none (0015-MADR D3, D4)` |
| C46now | continuation refused before the network when Unsupported | line | `llmprovider/contract.go:254` | `{req.PreviousResponseID != "", c.Continuation, "continuation"},` |
| C53now | R55 | line | `docs/guides/api-standards.md:405` | `**R55. A new export is justified.**` |
| C54now | R45 | line | `docs/guides/api-standards.md:343` | `**R45. The request on the wire does not change**` |
| C58now | errors classified by HTTP status | line | `llmprovider/api_error.go:299` | `func ClassifyHTTPError(provider string, resp *http.Response) error {` |
| C59now | the parser reads a top-level `error_type` | line | `llmprovider/api_error.go:787` | `` ErrorType string          `json:"error_type"` `` |
| C61now | 0012 §3.2: OpenCode's Responses route sends `store: false` | line | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:403` | `` **Responses route:** sends `store: false` `` |
| C65now | another code is classified as a 500 | line | `llmprovider/api_error.go:508` | `e.terminal, e.Kind = classifyAPIError(serviceOf(provider), http.StatusInternalServerError, env, nil, false)` |
| C66now | a 500 is a retryable outage | line | `llmprovider/api_error.go:645` | `case status >= http.StatusInternalServerError:` |
| C67now | `OutputLimit` is set by the limit rule after decoding | line | `llmprovider/internal/wire/limit_rule.go:246` | `resp.OutputLimit = r.Report()` |
| C77now | OpenCode builds its own | line | `llmprovider/providers/opencode/opencode.go:760` | `func (p *provider) responsesBody(c call, learner *wire.Learner)` |
| C79now | a `failed` with no error object is retryable | line | `llmprovider/internal/wire/responses/responses.go:667` | `return llmprovider.ClassifyStreamFailure(provider, "", "", "response.failed event received")` |
| C84now | `DropLimit` removes a limit not sent | line | `llmprovider/internal/wire/limit_rule.go:168` | `func DropLimit(body map[string]any, l llmprovider.SentLimit, path ...string) {` |
| C85now | `LimitRetry.Sent` records the value the built body carries | line | `llmprovider/internal/wire/limit_rule.go:182` | `// Sent records the value the built body carries, after a thinking budget` |
| C99now | OpenCode's responses route reads the Responses forms | line | `llmprovider/providers/opencode/opencode.go:1016` | `return wire.ResponsesLimits` |

None of the 27 moved rows changes what it is cited for: each text is the same line, found again; C7's new text is the same test, written as 0088 left it.

### The working-tree rows at the base (2026-11-24)

W1now to W7now cited uncommitted 0088 Phase 1b code. At `dd37a87` it is committed, and each row is checked in the scratch copy above (as W1 to W7, planted false there, the three plants below). Rows restated:

| Id | Claim | Moved | Where (at `dd37a87`) | Text |
| :--- | :--- | :--- | :--- | :--- |
| W1now | a plain reply is cut only when `incomplete` | same line | `llmprovider/internal/wire/responses/responses.go:422` | `if raw.Status == statusIncomplete {` |
| W2now | a completed stream ends through `end` with `cut` false, not `CutAnswer` | moved and reworded | `llmprovider/internal/wire/responses/responses.go:747` | `return s.end(event, false, yield)` |
| W3now | the held calls the response kept are yielded at the end | same line | `llmprovider/internal/wire/finish.go:104` | `kept := slices.ContainsFunc(resp.Output, func(item llmprovider.Item) bool { return item == call })` |
| W4now | `CutCall`: not a whole object, or a last `{}` | same line | `llmprovider/internal/wire/finish.go:56` | `return !WholeObject(args) \|\| last && noArguments(args)` |
| W5now | R1 to R3 open for 0088's Phase 3 review | same line | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1412` | `### Open for PLAN Phase 3's review (2026-11-23, tenth)` |
| W6now | `end` yields the held calls, then `EventDone` | moved and reworded | `llmprovider/internal/wire/responses/responses.go:845` | `if !s.held.Done(resp, yield) {` |
| W7now | the stream's last event gives only id, model and usage; the output comes from `output_item.done` | moved | `llmprovider/internal/wire/responses/responses.go:829` | `result.Usage = event.Response.Usage.counts()` |

What changed, as evidence only; no row of this record rests on these line numbers:

* **W2now and W6now.** The stream's end is a method, `streamState.end(event, cut, yield)` (`responses.go:821`), called with `cut` true for `response.incomplete` (`responses.go:745`) and false for `response.completed` (`responses.go:747`): the completed path reaches `completed(result, s.refused)` (`responses.go:840`), never `CutAnswer`, as W2now said of `streamEnd`; the held calls are released by `s.held.Done(resp, yield)` after that and before `EventDone` (W6now). ST33's check "inside the decoder, in `streamEnd` before held calls are released" is read as `end`, before line 845.
* **W7now.** The stream's last event still gives only id, model and usage (line 829), and a cut answer takes the terminal response's undelivered items (`takeTerminal`, 0088-MADR ST104), which a completed stream does not. *(2026-11-25: no longer so. A completed stream takes them too, and fails as `item_not_finished` on an announced item delivered neither way ([0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md) D1, ST11, built in 0092-PLAN Phase 1). How ST55's check meets that is OQ12, in the amendment of 2026-11-25 at the end. The text is unchanged.)*
* **W1now, W3now, W4now, W5now.** Unchanged lines. W5now's heading is still the tenth amendment's; R1 to R3 are still open for 0088's Phase 3 review, as B3 states. 0088-MADR's twelfth and thirteenth amendments add R4 (the route's fallback against a stated window, `limit_rule.go`) and R5 (`ResponsesLimits.Minimum` of 16, which no route on the Responses wire reaches with a limit the SDK chose). Neither touches a row here: `chatcompat`'s Responses path sends no limit the SDK chose (ST51), so R4 and R5 do not reach it; B3 is not changed.

ST33 and ST34 stand as decided: the held-call hold they rely on is in the tree at `wire.HeldCalls` (W3now, W6now), and ST34 stays conditional on 0088's Phase 3 ruling on R3.

### Plants (2026-11-24)

`python3 -B tools/claims.py --plants <scratch>/p0091/plants_claims.json`, on the scratch copy, exit 0, all three caught:

* W3's line moved off its text (`finish.go:104` to `:106`): `FAIL W3 llmprovider/internal/wire/finish.go:106: line 106 is 'return false'`.
* W1's text made false (`statusIncomplete` to `statusCompleted`): `FAIL W1 llmprovider/internal/wire/responses/responses.go:422: line 422 is 'if raw.Status == statusIncomplete {'`.
* C50's text on the kilocode checkout made false (`@openrouter/ai-sdk-provider` to `...-v2`): `FAIL C50 kilocode:packages/kilo-gateway/src/provider.ts:1: line 1 is 'import { createOpenRouter } from "@openrouter/ai-sdk-provider"'`.

Counts after this amendment: unchanged, 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Amendment (2026-11-24, fifth): PLAN Phase 1, the claims rows of the docs of the four providers that stay

Additions only; no row's decision changes. This is PLAN Phase 1 (ST1 to ST4, ST28): the owner's answers to OQ1 to OQ4 ("Chat stays (Recommended)", "Chat stays, every model (Recommended)", "Chat stays (Recommended)", "Chat stays (Recommended)") are carried out as comment-only changes to the four providers' package docs, and as the two places `docs/architecture.md` names each provider's wire. The base is `2f3c3a9`.

### Re-grounding at `2f3c3a9` (2026-11-24)

`python3 -B tools/claims.py docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md --rev self=2f3c3a9`: 82 of 109 pass, the same 27 rows that moved by `dd37a87` and are restated in the fourth amendment; nothing moved since. Of the rows that cite the four files, C13, C14, C16 to C25, C62, C63, C64 and C70 to C74, C81 to C83, C86 and C87 pass at `2f3c3a9`; C15 is the one that moved (`kilo.go:94` to `:96`, already C15now). Each paragraph Phase 1 changes is the one its row cites, so the re-ground holds.

### The new doc sentences, as claims rows (2026-11-24)

One row per new sentence, or per new line of `docs/architecture.md`, with the text it holds, each on one line of its file. Their ids end in "now" so that `tools/claims.py` does not read them against this record's `self=247b745`: the lines are those of the working tree of Phase 1, and the phase's commit re-pins them. The facts the sentences state are PR1 (Kilo: P1, P6, PR6), ~~PR3 (Together's page)~~ *(2026-11-24, after review: struck, Together's reason is a read of its docs, not PR3; see "Annotation (2026-11-24, after the review of Phase 1)" at the end)*, PR4 (Hugging Face: P3, P6, P7) and PR5, PR7 (e), PR8, PR9 (Ollama), each in the REPORT.

| Id | Claim | Where (working tree, 2026-11-24) | Text |
| :--- | :--- | :--- | :--- |
| N1now | `kilo.go`: the 2026-11-23 findings that keep Kilo on Chat (ST1) | `llmprovider/providers/kilo/kilo.go:20` | `found its /responses undocumented,` |
| N2now | `together.go`: four models only keep Together on Chat (ST2) | `llmprovider/providers/together/together.go:15` | `lists four models only (zai-org/GLM-5.2,` |
| N3now | `huggingface.go`: the router documents `/responses` as beta (ST3) | `llmprovider/providers/huggingface/huggingface.go:10` | `the router's guide documents it as beta` |
| N4now | `huggingface.go`: a bad key still answers 200 `failed` `server_error` (ST3) | `llmprovider/providers/huggingface/huggingface.go:12` | `still answers HTTP 200 with status:"failed" and the code server_error,` |
| N5now | `huggingface.go`: a text cut answers `completed`, no encrypted reasoning (ST3) | `llmprovider/providers/huggingface/huggingface.go:15` | `"completed" with no incomplete_details; and no encrypted reasoning comes` |
| N6now | `ollama.go`: an effort a model cannot take is silently ignored on `/v1/responses` (ST4) | `llmprovider/providers/ollama/ollama.go:16` | `found it silently ignoring an effort a model cannot take, where this route` |
| N7now | `ollama.go`: a text cut reported `completed` on `/v1/responses` (ST4) | `llmprovider/providers/ollama/ollama.go:18` | `at the output limit as completed, where this route says length.` |
| N8now | `architecture.md`: the four Chat providers stay on measurements (ST1 to ST4) | `docs/architecture.md:690` | `The four Chat Completions providers stay there on measurements of 2026-11-23` |
| N9now | `architecture.md`: Kilo's line states why | `docs/architecture.md:64` | `Kilo Gateway: Chat Completions, kept on measurement (0091-MADR ST1)` |
| N10now | `architecture.md`: Hugging Face's line states why | `docs/architecture.md:65` | `Hugging Face Inference Providers: Chat Completions, kept on measurement (0091-MADR ST3)` |
| N11now | `architecture.md`: Together's line states why | `docs/architecture.md:66` | `Together AI: Chat Completions, kept on measurement (0091-MADR ST2)` |
| N12now | `architecture.md`: Ollama's line states why | `docs/architecture.md:67` | `a local Ollama: Chat Completions, kept on measurement (0091-MADR ST4)` |

`README.md` names no provider's wire for these four (its provider table lists package paths; the wire lines are in `docs/architecture.md`), so it gets no row and no change.

### Checked (2026-11-24)

Scripts: `<scratch>/p1-0091/p1.py`, which writes the 12 rows with plain ids and a marker into a scratch record.

* **Red at the base.** The rows' texts were written first, before any doc changed. At `2f3c3a9`, with each row at line 1 of its file, all 12 fail on their assertion, and none finds its text on any line of its file (`found=[]`): for example `FAIL N1 llmprovider/providers/kilo/kilo.go:1: line 1 is '// Package kilo is the Kilo Gateway provider (0015-MADR D2), the inference API'`.
* **Green after the change.** On a scratch clone holding the working tree, committed there, with each row's line found (one line each), 12 of 12 pass, exit 0.
* **Plants.** `tools/claims.py --plants` on that record, each row's text made false (the word "(false)" appended to it): all 12 caught, for example `FAIL N4 llmprovider/providers/huggingface/huggingface.go:12: line 12 is '// still answers HTTP 200 with status:"failed" and the code server_error,'`. ST28's plant, on a scratch clone: the Hugging Face paragraph put back as it read at the base (the "undocumented" line, "chat completion tasks only", "measured 2026-08-29") and committed there: exit 1, N3, N4 and N5 fail, for example `N3 llmprovider/providers/huggingface/huggingface.go:10: line 10 is '// also exists, but it is deliberately NOT used: it is undocumented, the'`.

Counts after this amendment: unchanged, 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Annotation (2026-11-24, after the review of Phase 1)

Additions only; no row's decision changes. An independent review of Phase 1 (pass 1) returned FIX REQUIRED, docs only. This is PLAN Phase 1, pass 2. What it found, and what the record now says:

* **The reasons were stated as one kind of evidence; they are two.** The fifth amendment above and the pass-1 docs said the four providers stay on Chat "on measurements". That is wrong for Together, and imprecise for Kilo. Together's reason is a read of its `/responses` page (the REPORT's desk research, "Per service", Together); its probes (PR3, rounds 1 and 2) reported cuts as `incomplete` and returned `encrypted_content`, so no probe supports keeping it on Chat, and the amendment's "PR3 (Together's page)" is struck above. Kilo's "undocumented" is a docs read too; its probes found no encrypted reasoning when asked (PR1 P6, round 1) and a call cut inside its arguments reported `completed` (PR6, round 2, `kilo-auto/free`); its text cut was reported `incomplete` (PR1 P3), so it is not cited. The package docs and `docs/architecture.md` now name each reason's source, docs or probe, and the round.
* **Hugging Face's cut.** Pass 1 said "a text reply cut at the output limit answers `completed`". Round 1's P3 (`openai/gpt-oss-20b`) did answer `completed` with `incomplete_details` null at `output_tokens` equal to the limit, but where in the reply the cut fell is not recorded, and the measured hidden cut on that router is round 2's, inside a tool call on `meta-llama/Llama-3.1-8B-Instruct` (plain: `completed`, call `completed`, arguments not parseable, 160 tokens, the limit). The docs now say "a reply cut at the output limit", citing round 1 (P3) and, inside a tool call, round 2. The rows do not claim a text cut.
* **Together's two quoted phrases** ("may return malformed output", "partial") came through a fetch tool that relays a page through a small model (the REPORT, "How it was gathered"); the doc no longer quotes them and says they were not checked against the page.
* **Coverage.** Three sentences had no row (`kilo.go`'s first, `together.go`'s first, `ollama.go`'s first), and each multi-line sentence was pinned on one line only. Rows M1 to M27 below pin every sentence and each fact of the multi-line ones, and the architecture bullet's facts and link. *(2026-11-24, third pass: this claim was not true, eight facts were unpinned; M1now to M27now are superseded by the 41 rows F1now to F41now, one per new doc line. See "Annotation (2026-11-24, after the third pass)" at the end.)*
* **C15now moved again.** The fourth amendment's C15now (`kilo.go:96`) has its text at `llmprovider/providers/kilo/kilo.go:102` in the tree after pass 2 (`:101` after pass 1): Phase 1's own paragraph above it (five comment lines and a blank). The pin is the fourth amendment's, at `dd37a87`, and stands; the move is recorded here, as the phase's commit re-grounds it.

The table below replaces the fifth amendment's table of N1now to N12now (superseded, left as written above; ids M1now to M27now do not clash with it). Same rule: ids end in "now", so `tools/claims.py` does not read them against `self=247b745`; the lines are those of the working tree of Phase 1, pass 2.

| Id | Claim | Where (working tree) | Text |
| :--- | :--- | :--- | :--- |
| M1now | kilo.go: Chat Completions stays the only route (ST1) | `llmprovider/providers/kilo/kilo.go:19` | `Chat Completions stays the only route (0091-MADR ST1).` |
| M2now | kilo.go: a docs read found /responses undocumented | `llmprovider/providers/kilo/kilo.go:20` | `docs on 2026-11-23 found its /responses undocumented.` |
| M3now | kilo.go: probe, round 1: no encrypted reasoning when asked | `llmprovider/providers/kilo/kilo.go:21` | `day (kilo-auto/free) found it returning no encrypted reasoning when asked` |
| M4now | kilo.go: probe, round 2: a tool call cut inside its arguments reported completed | `llmprovider/providers/kilo/kilo.go:22` | `reporting a tool call cut inside its arguments as a` |
| M5now | kilo.go: that reply's arguments do not parse (round 2) | `llmprovider/providers/kilo/kilo.go:23` | `completed reply, with arguments that do not parse (round 2).` |
| M6now | together.go: Chat Completions stays for every model (ST2) | `llmprovider/providers/together/together.go:14` | `Chat Completions stays for every model (0091-MADR ST2).` |
| M7now | together.go: a docs read found four models only | `llmprovider/providers/together/together.go:15` | `found it lists four models only` |
| M8now | together.go: the two words relayed by the fetch tool, marked unchecked | `llmprovider/providers/together/together.go:18` | `page reported that other models may return malformed output and that` |
| M9now | huggingface.go: the router's guide documents /responses as beta | `llmprovider/providers/huggingface/huggingface.go:10` | `the router's guide documents it as beta` |
| M10now | huggingface.go: deliberately not used, by 0091-MADR ST3, on probes | `llmprovider/providers/huggingface/huggingface.go:11` | `deliberately NOT used (0091-MADR ST3). Probes of 2026-11-23 found that a` |
| M11now | huggingface.go: round 1, P7: a bad key answers 200 failed server_error | `llmprovider/providers/huggingface/huggingface.go:12` | `bad key still answers HTTP 200 with status:"failed" and the code` |
| M12now | huggingface.go: rounds 1 and 2: a cut reply answers completed | `llmprovider/providers/huggingface/huggingface.go:15` | `answers "completed" with no incomplete_details, in round 1 (P3) and,` |
| M13now | huggingface.go: round 1, P6: no encrypted reasoning when asked | `llmprovider/providers/huggingface/huggingface.go:17` | `back when it is asked for (round 1, P6).` |
| M14now | ollama.go: Chat Completions stays for generation (ST4) | `llmprovider/providers/ollama/ollama.go:14` | `Chat Completions stays for generation (0091-MADR ST4).` |
| M15now | ollama.go: round 6: an effort a model cannot take silently ignored | `llmprovider/providers/ollama/ollama.go:16` | `found it silently ignoring an effort a model cannot take (round 6), where` |
| M16now | ollama.go: rounds 1 and 3: a text cut reported completed | `llmprovider/providers/ollama/ollama.go:18` | `reply cut at the output limit as completed (rounds 1 and 3), where this` |
| M17now | architecture.md: Kilo's tree line states its sources | `docs/architecture.md:64` | `Kilo Gateway: Chat Completions, kept on its docs and probes (0091-MADR ST1)` |
| M18now | architecture.md: Hugging Face's tree line states its sources | `docs/architecture.md:65` | `Hugging Face Inference Providers: Chat Completions, kept on its docs and probes (0091-MADR ST3)` |
| M19now | architecture.md: Together's tree line states its source | `docs/architecture.md:66` | `Together AI: Chat Completions, kept on its docs (0091-MADR ST2)` |
| M20now | architecture.md: Ollama's tree line states its source | `docs/architecture.md:67` | `a local Ollama: Chat Completions, kept on probes (0091-MADR ST4)` |
| M21now | architecture.md: the four stay for stated reasons, docs and probes | `docs/architecture.md:690` | `The four Chat Completions providers stay there for stated reasons, not by` |
| M22now | architecture.md: Together's docs fact, and Kilo's round-1 probe | `docs/architecture.md:693` | `models only. Probes: Kilo's returned no encrypted reasoning when asked (round` |
| M23now | architecture.md: Kilo's round-2 call cut | `docs/architecture.md:694` | `1) and reported a tool call cut inside its arguments as` |
| M24now | architecture.md: Hugging Face's bad key and cut, rounds 1 and 2 | `docs/architecture.md:695` | `2); the Hugging Face router answered a bad key with HTTP 200` |
| M25now | architecture.md: Ollama's ignored effort (round 6) and text cut | `docs/architecture.md:697` | `effort a model cannot take (round 6) and reported a text reply cut at the` |
| M26now | architecture.md: Together's probes found no such loss | `docs/architecture.md:698` | `Together's probes found no such loss` |
| M27now | architecture.md: the link to the REPORT | `docs/architecture.md:701` | `[0091-REPORT-responses-api-across-providers.md](reports/0091-REPORT-responses-api-across-providers.md)` |

### Checked (2026-11-24, pass 2)

Scripts: `<scratch>/p1-0091/p1.py` and `rows.py`; the rows are written with plain ids M1 to M27 and a marker into a scratch record.

* **Red at the base.** At `2f3c3a9`, each row at line 1 of its file: all 27 fail on their assertion and none finds its text on any line (`found=[]`); the new and changed rows' texts were written before the docs were changed again.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 27 of 27 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 27 of 27 caught, for example `FAIL M4 llmprovider/providers/kilo/kilo.go:22: line 22 is '// for (round 1), and reporting a tool call cut inside its arguments as a'`. ST28's plant, the Hugging Face paragraph put back as it read at the base, committed on a scratch clone: exit 1, M9 to M13 fail, for example `M9 llmprovider/providers/huggingface/huggingface.go:10: line 10 is '// also exists, but it is deliberately NOT used: it is undocumented, the'`.

Counts after this annotation: unchanged, 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Annotation (2026-11-24, after the third pass)

Additions only; no row's decision changes. The delta review passed pass 2 with two findings, and the owner ruled a third pass of Phase 1, quoted exactly: "Pass 3 with full rows" (recorded in the PLAN as a dated deviation).

* **Together's encrypted reasoning.** `docs/architecture.md` said Together's "reasoning" came back "encrypted". The probes (PR3 P6, round 1) found `encrypted_content` returned; Together's page, as relayed, calls that field base64 and "not encrypted" (the REPORT's desk research, Together). The bullet now says only that its probes returned `encrypted_content`.
* **The pass-2 annotation's coverage claim was not true.** It said every fact was pinned; eight were not (`architecture.md` lines 691 to 692, 696 and 699, `huggingface.go` 13 to 14 and 16, `ollama.go` 15 and 17, `together.go` 19). Pass 3 pins every new doc line, whole: 41 rows, F1now to F41now, one per new line of the four package docs (5 in `kilo.go`, 6 in `together.go`, 8 in `huggingface.go`, 6 in `ollama.go`) and of `docs/architecture.md` (4 tree lines, 12 bullet lines). They replace the pass-2 table's M1now to M27now (superseded, left as written above; the ids do not clash). The pre-existing lines of each paragraph are not rows (C13, C14, C17, C18, C21 and C23 hold them at their pins).
* **Same rule as before:** ids end in "now", so `tools/claims.py` does not read them against `self=247b745`; the lines are those of the working tree of Phase 1, pass 3, and the phase's commit re-pins them. A line whose text holds a backtick is cited in a double-backtick span.

| Id | Claim | Where (working tree) | Text |
| :--- | :--- | :--- | :--- |
| F1now | kilo.go: Chat stays the only route (ST1); the docs read begins | `llmprovider/providers/kilo/kilo.go:19` | `Chat Completions stays the only route (0091-MADR ST1). A read of Kilo's` |
| F2now | kilo.go: the docs read found /responses undocumented | `llmprovider/providers/kilo/kilo.go:20` | `docs on 2026-11-23 found its /responses undocumented. Probes of the same` |
| F3now | kilo.go: probe, round 1: no encrypted reasoning when asked | `llmprovider/providers/kilo/kilo.go:21` | `day (kilo-auto/free) found it returning no encrypted reasoning when asked` |
| F4now | kilo.go: probe, round 2: a tool call cut inside its arguments reported completed | `llmprovider/providers/kilo/kilo.go:22` | `for (round 1), and reporting a tool call cut inside its arguments as a` |
| F5now | kilo.go: round 2: that reply's arguments do not parse | `llmprovider/providers/kilo/kilo.go:23` | `completed reply, with arguments that do not parse (round 2).` |
| F6now | together.go: Chat stays for every model (ST2); the docs read begins | `llmprovider/providers/together/together.go:14` | `Chat Completions stays for every model (0091-MADR ST2). A read of` |
| F7now | together.go: the docs read found four models only | `llmprovider/providers/together/together.go:15` | `Together's /responses page on 2026-11-23 found it lists four models only` |
| F8now | together.go: the models, first line | `llmprovider/providers/together/together.go:16` | `(zai-org/GLM-5.2, GLM-5.3, GLM-5.3-Flash and` |
| F9now | together.go: the last model; the relay caveat begins | `llmprovider/providers/together/together.go:17` | `deepseek-ai/DeepSeek-V4.1-Flash-1017). The fetch tool that relayed the` |
| F10now | together.go: the relayed reading: malformed output, partial support | `llmprovider/providers/together/together.go:18` | `page reported that other models may return malformed output and that` |
| F11now | together.go: the caveat: the words were not checked | `llmprovider/providers/together/together.go:19` | `support is partial; those two words were not checked against the page.` |
| F12now | huggingface.go: the router's guide documents /responses as beta | `llmprovider/providers/huggingface/huggingface.go:10` | `also exists and the router's guide documents it as beta, but it is` |
| F13now | huggingface.go: deliberately not used (ST3); the probes begin | `llmprovider/providers/huggingface/huggingface.go:11` | `deliberately NOT used (0091-MADR ST3). Probes of 2026-11-23 found that a` |
| F14now | huggingface.go: round 1, P7: a bad key answers 200 failed | `llmprovider/providers/huggingface/huggingface.go:12` | `bad key still answers HTTP 200 with status:"failed" and the code` |
| F15now | huggingface.go: round 1, P7: server_error, read as a retryable outage | `llmprovider/providers/huggingface/huggingface.go:13` | `server_error (round 1, P7), which the SDK reads as a retryable outage, not` |
| F16now | huggingface.go: not as an authentication failure; the cut begins | `llmprovider/providers/huggingface/huggingface.go:14` | `as an authentication failure; that a reply cut at the output limit` |
| F17now | huggingface.go: rounds 1 and 2: a cut reply answers completed | `llmprovider/providers/huggingface/huggingface.go:15` | `answers "completed" with no incomplete_details, in round 1 (P3) and,` |
| F18now | huggingface.go: round 2: inside a tool call; encrypted reasoning begins | `llmprovider/providers/huggingface/huggingface.go:16` | `inside a tool call, in round 2; and that no encrypted reasoning comes` |
| F19now | huggingface.go: round 1, P6: none when asked | `llmprovider/providers/huggingface/huggingface.go:17` | `back when it is asked for (round 1, P6). Do not "fix" this omission` |
| F20now | ollama.go: Chat stays for generation (ST4); Ollama documents a route | `llmprovider/providers/ollama/ollama.go:14` | `Chat Completions stays for generation (0091-MADR ST4). Ollama documents a` |
| F21now | ollama.go: the route's version, the probes' version | `llmprovider/providers/ollama/ollama.go:15` | `/v1/responses route (since v0.13.3), and probes of 2026-11-23 on v0.40.17` |
| F22now | ollama.go: round 6: an effort a model cannot take silently ignored | `llmprovider/providers/ollama/ollama.go:16` | `found it silently ignoring an effort a model cannot take (round 6), where` |
| F23now | ollama.go: Chat refuses it and the SDK learns the refusal | `llmprovider/providers/ollama/ollama.go:17` | `this route refuses it and the SDK learns the refusal, and reporting a text` |
| F24now | ollama.go: rounds 1 and 3: a text cut reported completed | `llmprovider/providers/ollama/ollama.go:18` | `reply cut at the output limit as completed (rounds 1 and 3), where this` |
| F25now | ollama.go: Chat says length | `llmprovider/providers/ollama/ollama.go:19` | `route says length.` |
| F26now | architecture.md: Kilo's tree line | `docs/architecture.md:64` | `Kilo Gateway: Chat Completions, kept on its docs and probes (0091-MADR ST1)` |
| F27now | architecture.md: Hugging Face's tree line | `docs/architecture.md:65` | `Hugging Face Inference Providers: Chat Completions, kept on its docs and probes (0091-MADR ST3)` |
| F28now | architecture.md: Together's tree line | `docs/architecture.md:66` | `Together AI: Chat Completions, kept on its docs (0091-MADR ST2)` |
| F29now | architecture.md: Ollama's tree line | `docs/architecture.md:67` | `a local Ollama: Chat Completions, kept on probes (0091-MADR ST4)` |
| F30now | architecture.md: the bullet: stated reasons | `docs/architecture.md:690` | `The four Chat Completions providers stay there for stated reasons, not by` |
| F31now | architecture.md: the bullet: a docs read and probes of 2026-11-23 | `docs/architecture.md:691` | `default (0091-MADR ST1 to ST4), from a docs read and probes of 2026-11-23.` |
| F32now | architecture.md: the bullet: Kilo undocumented, Together's page lists four | `docs/architecture.md:692` | `` Docs: Kilo's `/responses` is undocumented, and Together's page lists four `` |
| F33now | architecture.md: the bullet: four models only; Kilo's round-1 probe | `docs/architecture.md:693` | `models only. Probes: Kilo's returned no encrypted reasoning when asked (round` |
| F34now | architecture.md: the bullet: Kilo's round-2 call cut | `docs/architecture.md:694` | `` 1) and reported a tool call cut inside its arguments as `completed` (round `` |
| F35now | architecture.md: the bullet: Hugging Face's bad key | `docs/architecture.md:695` | `` 2); the Hugging Face router answered a bad key with HTTP 200 `failed` and a `` |
| F36now | architecture.md: the bullet: Hugging Face's cut, rounds 1 and 2; Ollama begins | `docs/architecture.md:696` | `` reply cut at the limit as `completed` (rounds 1 and 2); Ollama's ignored an `` |
| F37now | architecture.md: the bullet: Ollama's ignored effort (round 6), a text cut | `docs/architecture.md:697` | `effort a model cannot take (round 6) and reported a text reply cut at the` |
| F38now | architecture.md: the bullet: rounds 1 and 3; Together's probes found no loss | `docs/architecture.md:698` | `` limit as `completed` (rounds 1 and 3). Together's probes found no such loss `` |
| F39now | architecture.md: the bullet: Together's cuts incomplete, encrypted_content returned | `docs/architecture.md:699` | `` (its cuts came back `incomplete` and it returned `encrypted_content`), so `` |
| F40now | architecture.md: the bullet: Together's reason is the docs alone | `docs/architecture.md:700` | `its reason is the docs alone. The facts are in` |
| F41now | architecture.md: the bullet: the link to the REPORT | `docs/architecture.md:701` | `[0091-REPORT-responses-api-across-providers.md](reports/0091-REPORT-responses-api-across-providers.md).` |

### Checked (2026-11-24, pass 3)

Scripts: `<scratch>/p1-0091/p1.py` and `rows.py`; the rows are written with plain ids F1 to F41 and a marker into a scratch record.

* **Red at the base.** At `2f3c3a9`, each row at line 1 of its file: all 41 fail on their assertion and none finds its text on any line (`found=[]`): the text did not exist at the base.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 41 of 41 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 41 of 41 caught, for example (F39 pins the corrected parenthesis on Together's probes, `docs/architecture.md:699`) `FAIL F24 llmprovider/providers/ollama/ollama.go:18: line 18 is '// reply cut at the output limit as completed (rounds 1 and 3), where this'`. ST28's plant, the Hugging Face paragraph put back as it read at the base, committed on a scratch clone: exit 1, F12 to F19 fail, for example `F12 llmprovider/providers/huggingface/huggingface.go:10: line 10 is '// also exists, but it is deliberately NOT used: it is undocumented, the'`.

Counts after this annotation: unchanged, 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Amendment (2026-11-24, sixth): the release is split, v1.33.0 and v1.34.0

Annotation only; no row's decision changes but ST30's release. The owner's ruling of 2026-11-24, relayed by the coordinator, quoted as given: "Yes, re-scope v1.33.0 (Recommended)".

* **ST30 (annotated 2026-11-24).** v1.33.0 carries this record's REPORT and the docs of the four providers that stay (PLAN Phases 0 and 1), with no code. `chatcompat.WithResponses()` and the rows only it builds (ST5, ST10, ST13, ST16, ST18, ST19, ST23, ST33, ST34, ST50 to ST64 as PLAN Phase 2 lists them) ship in v1.34.0, after 0088's R1 to R3 are ruled there. Where this record says v1.33.0 of that opt-in, read v1.34.0. *(2026-11-25: R1 to R3 are now ruled, as 0088-MADR ST107 to ST110. R1's change is built in 0088-PLAN Phase 6, then 0092-PLAN Phase 1, then PLAN Phase 2 here. The text is unchanged.)*

Counts after this amendment: unchanged, 64 rows: 46 decided, 18 out of scope, 0 open. The record is `accepted`.

## Amendment (2026-11-25, seventh): 0092 fills a completed stream; ST55 meets its check (OQ12)

Additions only. [0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md), built in its PLAN's Phase 1 on `5f57f02`, changes the shared Responses reader that this record's PLAN Phase 2 builds on. Its ST26 routes one question here, as this record's own, and its ST14 leaves the order of ST55's check to it.

* **What 0092 built.** At the terminal event, completed or cut, the reader takes each item the terminal response carries that no `output_item.done` delivered, before any decision on how to finish (0092-MADR D1, ST14). On a completed stream it then fails a reply that announced a message, a call or a client tool search and delivered it neither way, as `ErrProviderUnavailable` after the reply, naming `item_not_finished` (0092-MADR ST11, OQ1). On a cut it names such a call as the cut call, R2 as built (0088-MADR ST108). The fourth amendment's W7now note is annotated where it stands.
* **ST15 and ST20 of 0092 reach PLAN Phase 2:** `TestChatcompat_ResponsesDecode_0091` gains M1's and M2's shapes, and `chatcompat` sets the decoder's `Logger` from its settings. The PLAN's amendment of 2026-11-25 records both.

### Put to the owner: OQ12

**OQ12. On `chatcompat`'s Responses stream, a `completed` reply whose `output_tokens` reached the limit sent (ST55 reads it as cut) also announced a call that neither an `output_item.done` nor the terminal response delivered. Which rule wins?** Today neither applies: `chatcompat` has no Responses path yet. Under 0092 alone, the completed stream fails as `item_not_finished`; under ST55 alone, the reply is a cut, and R2 names the call as the cut call.

| Option | Cost |
| :--- | :--- |
| (1) **The cut wins (recommended).** ST55's check decides first; a reply it reads as cut takes the cut path, where R2 names the unfinished call with `WarningToolCallCut` and the reply finishes `length` with what is usable. 0092's `item_not_finished` applies only to a completed reply ST55 does not read as cut. | A call a server lost for another reason, at exactly the limit, is reported as cut. The caller keeps the text it was shown, as ST55 intends for a hidden cut. ST33's decoder field is read before the completed path's check. |
| (2) **The broken stream wins.** 0092's check runs first, as it does for `openai`, `grok` and OpenCode; ST55 reads only a reply with no announced item left. | A hidden cut that stopped inside a call fails as a retryable broken stream after text was shown. A resend at the same limit is likely cut again. ST55's stream case loses the call cut case it was ruled for. |
| (3) **Measure first.** PLAN Phase 2 builds ST55 without this case. 0092's check stands until a server reached through `chatcompat` is measured cutting inside a call on a `completed` stream; that measurement's amendment rules it. | (2)'s behaviour, unmeasured, until a probe. The probe is a live run on the owner's word. |

Doing nothing leaves PLAN Phase 2 to choose an order the owner never ruled. *(2026-11-25: answered, "The cut rule wins (Recommended)", option (1). See "Amendment (2026-11-25, eighth)".)*

### Rows (2026-11-25, seventh)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST65 | `chatcompat`'s Responses stream: a `completed` reply at the limit sent (ST55) that announced a call delivered neither by `output_item.done` nor by the terminal response | open | recommended: OQ12 (1) | OQ12; 0092-MADR ST11; 0088-MADR ST108 | *(2026-11-25: decided by OQ12's answer; its test is PLAN Phase 2's, restated in the amendment of 2026-11-25, eighth)* |

Counts after this amendment: 65 rows: 46 decided, 18 out of scope, 1 open (ST65). The record is `proposed` until OQ12 is answered; PLAN Phase 2 waits for it.

## Amendment (2026-11-25, eighth): OQ12 answered, ST65 decided

Additions only. The owner's answer, relayed by the coordinator, quoted exactly: "The cut rule wins (Recommended)".

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ12 | "The cut rule wins (Recommended)" | (1): ST55's check decides first | ST65 |

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST65 | `chatcompat`'s Responses stream: a `completed` reply at the limit sent (ST55) that announced a call delivered neither by `output_item.done` nor by the terminal response | decided | the cut: that call is the cut call, dropped and named in `WarningToolCallCut`, and the reply finishes `length` (ST55; 0088-MADR ST108). A `completed` reply under the limit sent, or with none sent, keeps 0092-MADR ST11's `item_not_finished`, `ErrProviderUnavailable` after the reply. ST55's check runs before the completed path's check | OQ12 "The cut rule wins (Recommended)" | PLAN Phase 2: `TestChatcompat_ResponsesCut_0091`, stream, gains both cases: at the limit, `length` with one `WarningToolCallCut` naming the call; under it, `ErrProviderUnavailable` naming `item_not_finished`. Red at Phase 2's base; plant: 0092's check run before ST55's, so the reply at the limit fails |

Counts after this amendment: 65 rows: 47 decided, 18 out of scope, 0 open. The record is `accepted`.

## Amendment (2026-11-26, ninth): four cases from PLAN Phase 2's re-ground

Additions only. PLAN Phase 2's re-ground at `dc21fe2` (its execution entry of 2026-11-26) found four cases no row decided, D1 to D4, and stopped before any code. The coordinator relayed two owner answers, quoted exactly, and decided two cases as the coordinator's own calls, marked so below.

| Case | Answer | By | Rows it decides |
| :--- | :--- | :--- | :--- |
| D1, `kilo.go:15-16` says Kilo's `/responses` "puts reasoning where the Responses decoder does not read it", which ST10 and ST63 make false | option (1): Phase 2 restates the clause, comment only, to what is true after ST10 | the coordinator's call | ST66 |
| D2, `TestEvents_ReasoningDelta` (`events_0031_test.go:90-106`) asserts that `response.reasoning_text.delta` yields nothing, which ST63 reverses | "Pass it on everywhere (Recommended)" | the owner | ST67 |
| D3, the PLAN's pin "`openai`'s existing decode tests, unchanged" has a plant no existing test can catch: no test of `openai`, `grok` or OpenCode's responses route has a completed reply that reports `output_tokens` | option (1): a new green pin on the three | the coordinator's call | ST68 |
| D4, ST14 and ST55 say "a `completed` reply", and the plain decode reads a reply with no `status` as completed | "Check every reply treated as completed (Recommended)" | the owner | ST69 |

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST66 | `kilo.go`'s package doc, its lines 15 and 16, once ST10 is built | decided | restated, comment only, with the paragraph's line count kept: Kilo's `/responses` sends its reasoning as plain text, which the Responses decoder reads but cannot replay, with no encrypted content (PR1 P6) | D1, the coordinator's call | a claims row per changed line, red at `dc21fe2` (the text absent) and planted false |
| ST67 | `response.reasoning_text.delta` on every Responses provider, and the existing test that says it yields nothing | decided | yielded as `EventReasoningDelta` by the shared reader, on `openai`, `grok`, OpenCode's responses route and `chatcompat`'s opt-in, as ST63 states; `TestEvents_ReasoningDelta`'s expectation changes to include it; 0031-MADR D2's "is not mapped" gets a dated annotation pointing here | D2: the owner, "Pass it on everywhere (Recommended)" | `TestEvents_ReasoningDelta`, changed: red at `dc21fe2` (the delta yields nothing); plant: the delta unmapped |
| ST68 | A `completed` reply at the limit sent on `openai`, `grok` and OpenCode's responses route, which leave the decoder's limit unset (ST33 (built)) | decided | read as today: finishing `stop`, or `tool_calls` with a call, never `length`. It replaces the PLAN's pin "`openai`'s existing decode tests, unchanged", whose plant no existing test could catch | D3, the coordinator's call | `TestResponses_CompletedAtLimitUnchecked_0091` in `llmprovider/providers/`, each provider plain and stream: green at `dc21fe2`; plants: the decoder's limit set by `openai`, by `grok`, by OpenCode, each caught by its own cases |
| ST69 | A plain reply on `chatcompat`'s Responses path whose `status` is absent, or a value other than `completed`, `failed` and `incomplete`, at the limit sent | decided | the check covers every reply the decoder finishes as completed, so such a reply whose `output_tokens` reached the limit sent is read as cut and finishes `length`, as ST55's. A reply with an unknown status at the limit reads `length` too: a cost the owner's answer accepts | D4: the owner, "Check every reply treated as completed (Recommended)" | `TestChatcompat_ResponsesCut_0091`, plain: a reply with no `status` and one with `status` `in_progress`, each at the limit, finish `length`; plant: the check gated on `status` `completed` |

Counts after this amendment: 69 rows: 51 decided, 18 out of scope, 0 open. The record is `accepted`.

### Claims rows of PLAN Phase 2's doc lines (2026-11-26)

One row per new doc line of Phase 2 (PLAN Phase 2 step 5, as Phase 1's third pass pinned its lines): the package docs of `chatcompat` (ST62, ST64), `kilo.go` (ST66) and the Responses package, `CHANGELOG.md`, `docs/architecture.md`, `README.md`, and the dated annotations on 0061-MADR D4 (ST31) and 0031-MADR D2 (ST67). A table row of `docs/architecture.md` is pinned by its longest new clause. Ids end in "now", so `tools/claims.py` does not read them against this record's `self=247b745`; the lines are the working tree's of Phase 2, pass 1, and the phase's commit re-pins them. Scripts: `<scratch>/b0091p2/claims_rows.py`.

* **Red at the base.** At `dc21fe2`, each row at line 1 of its file: all 106 fail on their assertion, and none finds its text on any line of its file.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 106 of 106 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 106 of 106 caught, for example `FAIL H1 llmprovider/providers/kilo/kilo.go:15: line 15 is "// but those routes are undocumented, and Kilo's /responses sends reasoning"`.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| H1now | new doc line | `llmprovider/providers/kilo/kilo.go:15` | `but those routes are undocumented, and Kilo's /responses sends reasoning` |
| H2now | new doc line | `llmprovider/providers/kilo/kilo.go:16` | `as text alone, which is read but never replayed. One route reaches the whole` |
| H3now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:45` | `WithResponses is for a server the caller knows speaks OpenAI's Responses` |
| H4now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:46` | `API: Generate and Stream then send POST {base}/responses, a listing probe` |
| H5now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:47` | `among them, and read the reply with the decoder the openai and grok` |
| H6now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:48` | `providers share; the listing and the capabilities are unchanged` |
| H7now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:49` | `(0091-MADR ST50 to ST52, ST59). No server was measured through this path.` |
| H8now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:50` | `The request carries store false and asks for encrypted reasoning; its` |
| H9now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:51` | `effort, image refusal and output limit follow the same facts as on Chat` |
| H10now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:52` | `Completions. A server that refuses the route gets its error back: there` |
| H11now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:53` | `is no fallback to Chat Completions. Against Chat Completions, a caller` |
| H12now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:54` | `who gives it loses: a 200 reply with status failed, as Hugging Face's` |
| H13now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:55` | `router sends for a bad key, is read as a retryable outage (ST53); an` |
| H14now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:56` | `effort a model cannot take is dropped with no warning by a server that` |
| H15now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:57` | `ignores it, as Ollama's /v1/responses does (ST54); a refused limit's` |
| H16now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:58` | `"supports at most" cap is not learned (ST56); and prior reasoning goes` |
| H17now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:59` | `back only when it carries encrypted content, never under a facts-named` |
| H18now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:60` | `field (ST51). Reasoning a server sends only as reasoning_text is shown,` |
| H19now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:61` | `not sent back (ST63). A reply finished as completed whose output_tokens` |
| H20now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:62` | `reached the limit sent is read as cut, finishing length, as Ollama's` |
| H21now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:63` | `/v1/responses hides a cut that way; one that ends on exactly the limit's` |
| H22now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:64` | `last token is reported cut too (ST55, ST69).` |
| H23now | new doc line | `llmprovider/internal/wire/responses/responses.go:2` | `grok providers and OpenCode's responses route (0015-MADR D2), and by a` |
| H24now | new doc line | `llmprovider/internal/wire/responses/responses.go:3` | `chatcompat provider built with chatcompat.WithResponses (0091-MADR ST52).` |
| H25now | new doc line | `llmprovider/internal/wire/responses/responses.go:44` | `contentReasoningText is a reasoning item's content part that carries` |
| H26now | new doc line | `llmprovider/internal/wire/responses/responses.go:45` | `its text, where a server sends no summary (0091-MADR ST10).` |
| H27now | new doc line | `llmprovider/internal/wire/responses/responses.go:377` | `Limit is the output limit the request's attempt sent, set only by` |
| H28now | new doc line | `llmprovider/internal/wire/responses/responses.go:378` | `chatcompat's Responses path (0091-MADR ST33, ST55): a reply the` |
| H29now | new doc line | `llmprovider/internal/wire/responses/responses.go:379` | `decoder would finish as completed, its status completed, absent or` |
| H30now | new doc line | `llmprovider/internal/wire/responses/responses.go:380` | `any other but failed and incomplete (ST69), whose output_tokens` |
| H31now | new doc line | `llmprovider/internal/wire/responses/responses.go:381` | `reached Limit is read as one the limit cut, as a reply incomplete for` |
| H32now | new doc line | `llmprovider/internal/wire/responses/responses.go:382` | `max_output_tokens is. On a stream it is read before the held calls` |
| H33now | new doc line | `llmprovider/internal/wire/responses/responses.go:383` | `are released and before an announced item's check (ST34, ST65). Zero,` |
| H34now | new doc line | `llmprovider/internal/wire/responses/responses.go:384` | `as openai, grok and OpenCode leave it, reads no reply as cut (ST68).` |
| H35now | new doc line | `llmprovider/internal/wire/responses/responses.go:388` | `atLimit reports a reply whose output_tokens, u's, reached d.Limit, so` |
| H36now | new doc line | `llmprovider/internal/wire/responses/responses.go:389` | `that the limit cut it however its service finished it (0091-MADR ST55).` |
| H37now | new doc line | `llmprovider/internal/wire/responses/responses.go:452` | `A call's arguments stream in pieces (0088-MADR ST107). A reply` |
| H38now | new doc line | `llmprovider/internal/wire/responses/responses.go:453` | `at the limit sent is cut whatever its status said (0091-MADR` |
| H39now | new doc line | `llmprovider/internal/wire/responses/responses.go:454` | `ST55, ST69).` |
| H40now | new doc line | `llmprovider/internal/wire/responses/responses.go:615` | `A server that sends its reasoning as text puts it in content, as` |
| H41now | new doc line | `llmprovider/internal/wire/responses/responses.go:616` | `reasoning_text: it is read when the summary holds none (0091-MADR` |
| H42now | new doc line | `llmprovider/internal/wire/responses/responses.go:617` | `ST10, ST63). It has no encrypted content, so it is not replayed.` |
| H43now | new doc line | `llmprovider/internal/wire/responses/responses.go:743` | `delta (0031-PLAN Phase 1) and a reasoning text delta (0091-MADR ST63) as` |
| H44now | new doc line | `llmprovider/internal/wire/responses/responses.go:744` | `EventReasoningDelta, each` |
| H45now | new doc line | `llmprovider/internal/wire/responses/responses.go:1135` | `A completed stream at the limit sent is a cut, decided before the` |
| H46now | new doc line | `llmprovider/internal/wire/responses/responses.go:1136` | `announced items' check (0091-MADR ST55, ST65).` |
| H47now | new doc line | `llmprovider/internal/wire/responses/responses.go:1183` | `each. The reasoning summary delta is the one 0031-PLAN Phase 1 measured;` |
| H48now | new doc line | `llmprovider/internal/wire/responses/responses.go:1184` | `the reasoning text delta, which probes of 2026-11-23 saw, is passed on on` |
| H49now | new doc line | `llmprovider/internal/wire/responses/responses.go:1185` | `every provider (0091-MADR ST63, ST67). A` |
| H50now | new doc line | `llmprovider/providers/chatcompat/responses.go:16` | `The Responses path, for a server the caller knows speaks OpenAI's` |
| H51now | new doc line | `llmprovider/providers/chatcompat/responses.go:17` | `Responses API (0091-MADR ST5, ST50 to ST65, OQ5 "Opt-in WithResponses()").` |
| H52now | new doc line | `llmprovider/providers/chatcompat/responses.go:19` | `responsesValue is WithResponses' value among the options.` |
| H53now | new doc line | `llmprovider/providers/chatcompat/responses.go:22` | `WithResponses sends Generate and Stream to POST {base}/responses, OpenAI's` |
| H54now | new doc line | `llmprovider/providers/chatcompat/responses.go:23` | `Responses API, in place of /chat/completions, for a server the caller` |
| H55now | new doc line | `llmprovider/providers/chatcompat/responses.go:24` | `knows speaks it (0091-MADR ST50). The listing and everything else are` |
| H56now | new doc line | `llmprovider/providers/chatcompat/responses.go:25` | `unchanged, and there is no fallback: a server that refuses the route` |
| H57now | new doc line | `llmprovider/providers/chatcompat/responses.go:26` | `answers with its error, classified by its status. It is a shared option` |
| H58now | new doc line | `llmprovider/providers/chatcompat/responses.go:27` | `(llmprovider.SharedOption), as WithID is, so another provider's New takes` |
| H59now | new doc line | `llmprovider/providers/chatcompat/responses.go:28` | `it and ignores it. The package doc names what a caller gives up on this` |
| H60now | new doc line | `llmprovider/providers/chatcompat/responses.go:29` | `path against Chat Completions.` |
| H61now | new doc line | `llmprovider/providers/chatcompat/responses.go:34` | `responsesOn reports whether values carry WithResponses.` |
| H62now | new doc line | `llmprovider/providers/chatcompat/responses.go:42` | `decode is a plain reply's decoder: the Chat Completions wire's, or the` |
| H63now | new doc line | `llmprovider/providers/chatcompat/responses.go:43` | `Responses wire's with WithResponses, labelled with the provider's id` |
| H64now | new doc line | `llmprovider/providers/chatcompat/responses.go:44` | `(0091-MADR ST52).` |
| H65now | new doc line | `llmprovider/providers/chatcompat/responses.go:54` | `events is a stream's reader: the Chat Completions wire's, or the Responses` |
| H66now | new doc line | `llmprovider/providers/chatcompat/responses.go:55` | `wire's with WithResponses.` |
| H67now | new doc line | `CHANGELOG.md:30` | `` `chatcompat.WithResponses()`: a `chatcompat` provider built with it sends `` |
| H68now | new doc line | `CHANGELOG.md:31` | `` `Generate` and `Stream` to the server's `POST {base}/responses`, OpenAI's `` |
| H69now | new doc line | `CHANGELOG.md:32` | `` Responses API, and reads the reply with the decoder `openai` and `grok` `` |
| H70now | new doc line | `CHANGELOG.md:33` | `share, for a server the caller knows speaks it; there is no fallback to` |
| H71now | new doc line | `CHANGELOG.md:34` | `` Chat Completions. A reply finished as completed whose `output_tokens` `` |
| H72now | new doc line | `CHANGELOG.md:35` | `` reached the limit sent is reported cut, finishing `FinishLength`. `` |
| H73now | new doc line | `CHANGELOG.md:36` | `` Reasoning a server sends only as `reasoning_text` content is read, on `` |
| H74now | new doc line | `CHANGELOG.md:37` | `` every Responses provider, and its `response.reasoning_text.delta` `` |
| H75now | new doc line | `CHANGELOG.md:38` | `` yielded as `EventReasoningDelta`; it is not sent back. The package doc `` |
| H76now | new doc line | `CHANGELOG.md:39` | `names what the option gives up against Chat Completions` |
| H77now | new doc line | `CHANGELOG.md:40` | `([0091-MADR](docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md)` |
| H78now | new doc line | `CHANGELOG.md:41` | `ST50 to ST69).` |
| H79now | new doc line | `docs/architecture.md:68` | `llmprovider/providers/chatcompat/ any Chat Completions server, under the caller's id: not in Default(); Responses with WithResponses (0091-MADR ST5)` |
| H80now | new doc line | `docs/architecture.md:173` | `` `WithResponses`, which sends `Generate` and `Stream` to `{base}/responses` and reads the reply with the shared Responses decoder (0091-MADR ST50 to ST52) `` |
| H81now | new doc line | `docs/architecture.md:175` | `` a reasoning item whose summary holds no text is read from its `reasoning_text` content (0091-MADR ST10, ST63) `` |
| H82now | new doc line | `docs/architecture.md:470` | `` `EventReasoningDelta`: Responses' reasoning summary and reasoning text `` |
| H83now | new doc line | `docs/architecture.md:471` | `(0091-MADR ST67), Messages'` |
| H84now | new doc line | `docs/architecture.md:486` | `` not run the `item_not_finished` check (ST33). On `chatcompat`'s `` |
| H85now | new doc line | `docs/architecture.md:487` | `` `WithResponses` path, a reply finished as completed whose `output_tokens` `` |
| H86now | new doc line | `docs/architecture.md:488` | `reached the limit sent is a cut, decided before that check (0091-MADR` |
| H87now | new doc line | `docs/architecture.md:489` | `ST55, ST65).` |
| H88now | new doc line | `docs/architecture.md:701` | `` **Five wire formats:** OpenAI Responses (`openai`, `grok`, and `chatcompat` `` |
| H89now | new doc line | `docs/architecture.md:702` | `` with `WithResponses`, 0091-MADR ST5), Anthropic Messages `` |
| H90now | new doc line | `README.md:131` | `of its arguments arrived (0088-MADR ST107, ST110, OQ21). On a` |
| H91now | new doc line | `README.md:132` | `` `chatcompat` provider built with `chatcompat.WithResponses()`, which sends `` |
| H92now | new doc line | `README.md:133` | `` `Generate` and `Stream` to the server's `/responses`, a reply the server `` |
| H93now | new doc line | `README.md:134` | `` finishes as completed is reported cut too when its `output_tokens` `` |
| H94now | new doc line | `README.md:135` | `` reached the limit sent, as Ollama's `/v1/responses` hides a cut `` |
| H95now | new doc line | `README.md:136` | `` (0091-MADR ST55). An empty answer is `ErrEmptyAnswer` too, with `` |
| H96now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:238` | `` (2026-11-26: a caller who gives `chatcompat.WithResponses()` has `` |
| H97now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:239` | `` `Generate` and `Stream` sent to the server's `/responses`, read by `` |
| H98now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:240` | `` `internal/wire/responses`, by `` |
| H99now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:241` | `[0091-MADR-adopt-the-responses-api-per-provider.md](0091-MADR-adopt-the-responses-api-per-provider.md)` |
| H100now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:242` | `ST5 and ST50 (OQ5 "Opt-in WithResponses()"). Without it, the package` |
| H101now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:243` | `speaks Chat Completions as above. The text above is unchanged.)*` |
| H102now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:260` | `(2026-11-26: since` |
| H103now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:261` | `[0091-MADR-adopt-the-responses-api-per-provider.md](0091-MADR-adopt-the-responses-api-per-provider.md)` |
| H104now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:262` | `` ST63 and ST67, it is mapped to `EventReasoningDelta` on every Responses `` |
| H105now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:263` | `provider, after probes of 2026-11-23 saw Kilo, Together and Hugging Face` |
| H106now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:264` | `send it. The text above is unchanged.)*` |

### Note (2026-11-26): ST16's and ST59's tests and plants, the coordinator's calls

Additions only; no row's outcome changes. PLAN Phase 2, pass 1, found neither row's plant catchable as written (D5, D6). Both are test design, decided by the coordinator, not by the owner:

* **ST16 (D5, option (2)).** Its outcome is protected by a pair of guards: the reader stopping at the terminal event (`end` reports a stop, on which `wire.ReadEvents` returns, `llmprovider/internal/wire/sse.go:64-66`) and the `[DONE]` skip. Either alone keeps it, so the plant "`[DONE]` decoded as JSON" alone is not caught, nor is the reader alone reading on. ST16's plant is now one plant removing both guards. Its test, `TestChatcompat_ResponsesDone_0091`, reads the whole stream and wants nothing after the done event, no error either.
* **ST59 (D6, option (1)).** `llmtest.Run` checks only that a declared capability is honoured, so it passes a provider declaring `Continuation` `Supported` against a server that answers. ST59's test, `TestConformance_Responses_0091`, now also wants `Capabilities()` identical with and without `WithResponses()`. Its plant "`Continuation` declared `Supported`" is read as the option declaring it, since a change to the declaration both paths share leaves them equal: the Responses path's `Capabilities()` reporting `Supported`.

## Amendment (2026-11-26, tenth): the code review of PLAN Phase 2

Additions only. The code review of PLAN Phase 2 passed with notes; the coordinator relayed two owner rulings, quoted exactly, and the review's fixes. Built as Phase 2's pass 2.

| Question | The owner's answer | Rows it decides |
| :--- | :--- | :--- |
| A reply on `chatcompat`'s Responses path that is both a refusal and at the output limit: `content_filter` or `length`? | "The refusal wins (Recommended)" | ST70 |
| Since ST67, `reasoning_text` deltas count toward the bound on an answer read over the stream (`wire.Collect`), on `openai`, `grok` and OpenCode's responses route too: keep the bound, or raise it? | "Raise the bound" | ST71 |

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST70 | A reply finished as completed that holds a refusal and whose `output_tokens` reached the limit sent (ST55), plain or streamed | decided | it finishes `content_filter`, as a refusal does: the refusal is read before the limit check, so `atLimit` does not make it a cut. A refusal under the limit, or with no limit sent, finishes `content_filter` as before | the owner, "The refusal wins (Recommended)" | `TestChatcompat_ResponsesRefusalAtLimit_0091`, plain and stream: red at Phase 2's pass-1 tree (`length`); plant: the old order, the limit check before the refusal |
| ST71 | The bound on an answer read over the stream (`wire.Collect`): text, reasoning and call-argument bytes | decided | raised from `ReplyLimit`, 16 MiB, to its own `wire.StreamAnswerLimit`, 32 MiB; reasoning still counted, so a runaway stream still stops; a plain reply's body keeps `ReplyLimit`. The value is the coordinator's call under the owner's ruling, from this evidence: the largest output limits in the providers' recorded facts are 524,288 tokens on OpenCode (`space-bunny`, `space-bunny-free`), 500,000 on xAI (`grok-4.5` to `grok-4.7`) and 272,000 on OpenAI (`gpt-5-pro`), in `llmprovider/catalog/testdata/extract-2026-10-12/api.json`; the old 16 MiB allowed 32 bytes per output token at 524,288 tokens, with reasoning counted once; since ST67, a server that streams both a summary and the raw text of its reasoning has the reasoning counted twice, so the same 32 bytes per token over reasoning plus answer needs twice the bound | the owner, "Raise the bound"; the value, the coordinator's call | `TestCollect_StreamAnswerLimit_0091`: 20 MiB of reasoning deltas, between the old bound and the new, is returned; 33 MiB is `ErrIncomplete`; red at Phase 2's pass-1 tree (the 20 MiB reply fails); plant: the old value, 16 MiB |

Review fixes, no row's outcome changing: `CHANGELOG.md` gains a **Changed** entry naming `openai`, `grok` and OpenCode's responses route (ST28's annotation above), and its **Added** entry is cut to a reference; `chatcompat`'s package doc scopes its Chat-only lines to the Chat path; `responses.go`'s reasoning comment says what is sent back. A follow-up, not built: a server that streams both summary and raw deltas for one item keeps only the summary in the final item (ST10), though both were streamed.

Counts after this amendment: 71 rows: 53 decided, 18 out of scope, 0 open. The record is `accepted`.

### Owner ruling (2026-11-26): ST71's reach

Building ST71 broke `TestGenerateOverStream_ReplyLimit_0088` (0088-MADR ST88b), which holds the streamed answer to a plain reply's 16 MiB on Claude's Messages wire: `wire.Collect` serves every wire. Put to the owner through the coordinator, three options; the answer, quoted exactly: "Every provider, one bound (Recommended)".

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST71 (restated) | The bound on an answer read over the stream, every provider's `Generate` that names no limit | decided | one bound, 32 MiB (`wire.StreamAnswerLimit`), for every streamed `Generate` on every wire, reasoning counted; a plain reply's body keeps `ReplyLimit`, 16 MiB, so the two are separate constants. 0088-MADR ST88b gets a dated note: its parity with a plain reply ends *(2026-11-27: the joint review found that a Responses reply cannot reach 32 MiB: its terminal `response.completed` repeats the whole `output` and is one event, capped by `EventLimit`, 16 MiB. Responses replies stay capped at about 16 MiB by the terminal event; the other wires get 32 MiB. See the eleventh amendment, ST72. The text before this note is unchanged.)* | the owner, "Every provider, one bound (Recommended)"; the value, the coordinator's call | ST71's `TestCollect_StreamAnswerLimit_0091`; `TestGenerateOverStream_ReplyLimit_0088`, changed: 33 MiB over Claude's stream is still `ErrIncomplete`, and 17 MiB now returns; plant: the old value, 16 MiB, caught by both |

### Claims rows of PLAN Phase 2's doc lines, pass 2 (2026-11-26)

One row per new doc line of Phase 2 after its second pass, the same files as pass 1's annotation plus `chatcompat/limit.go`'s and `wire/overstream.go`'s doc comments, the rest of `chatcompat.go`'s doc comments, and 0088-MADR ST88b's note. They supersede H1now to H106now, which pinned lines pass 2 moved or changed (left as written above; the ids do not clash). Ids end in "now"; the phase's commit re-pins them. Scripts: `<scratch>/b0091p2/claims_rows.py J`.

* **Red at the base.** At `dc21fe2`, each row at line 1 of its file: all 155 fail on their assertion, and none finds its text on any line of its file.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 155 of 155 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 155 of 155 caught, for example `FAIL J1 llmprovider/providers/kilo/kilo.go:15: line 15 is "// but those routes are undocumented, and Kilo's /responses sends reasoning"`.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| J1now | new doc line | `llmprovider/providers/kilo/kilo.go:15` | `but those routes are undocumented, and Kilo's /responses sends reasoning` |
| J2now | new doc line | `llmprovider/providers/kilo/kilo.go:16` | `as text alone, which is read but never replayed. One route reaches the whole` |
| J3now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:45` | `WithResponses is for a server the caller knows speaks OpenAI's Responses` |
| J4now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:46` | `API: Generate and Stream then send POST {base}/responses, a listing probe` |
| J5now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:47` | `among them, and read the reply with the decoder the openai and grok` |
| J6now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:48` | `providers share; the listing and the capabilities are unchanged` |
| J7now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:49` | `(0091-MADR ST50 to ST52, ST59). No server was measured through this path.` |
| J8now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:50` | `The request carries store false and asks for encrypted reasoning; its` |
| J9now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:51` | `effort, image refusal and output limit follow the same facts as on Chat` |
| J10now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:52` | `Completions. A server that refuses the route gets its error back: there` |
| J11now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:53` | `is no fallback to Chat Completions. Against Chat Completions, a caller` |
| J12now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:54` | `who gives it loses: a 200 reply with status failed, as Hugging Face's` |
| J13now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:55` | `router sends for a bad key, is read as a retryable outage (ST53); an` |
| J14now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:56` | `effort a model cannot take is dropped with no warning by a server that` |
| J15now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:57` | `ignores it, as Ollama's /v1/responses does (ST54); a refused limit's` |
| J16now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:58` | `"supports at most" cap is not learned (ST56); and prior reasoning goes` |
| J17now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:59` | `back only when it carries encrypted content, never under a facts-named` |
| J18now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:60` | `field (ST51). Reasoning a server sends only as reasoning_text is shown,` |
| J19now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:61` | `not sent back (ST63). A reply finished as completed whose output_tokens` |
| J20now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:62` | `reached the limit sent is read as cut, finishing length, as Ollama's` |
| J21now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:63` | `/v1/responses hides a cut that way; one that ends on exactly the limit's` |
| J22now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:64` | `last token is reported cut too (ST55, ST69).` |
| J23now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:65` | `A refusal finishes content_filter, at the limit or under it (ST70).` |
| J24now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:70` | `is Unsupported: Chat Completions is stateless, and with WithResponses no` |
| J25now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:71` | `previous_response_id is sent either. Native streaming is Supported: Stream` |
| J26now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:72` | `asks for an event stream, on Chat Completions with its usage, and yields` |
| J27now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:73` | `its events as they arrive (0031-MADR E1, E5).` |
| J28now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:75` | `Images are Supported: on Chat Completions an ImageItem goes as an image_url` |
| J29now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:77` | `before it, whose text becomes a text part; with WithResponses, as an` |
| J30now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:78` | `input_image part. The server decides what it takes. A model whose facts` |
| J31now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:79` | `state its input without image is refused before the network, with` |
| J32now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:82` | `Degradations, as on Chat Completions; WithResponses' are in its paragraph` |
| J33now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:86` | `- Reasoning takes an effort, sent as reasoning_effort (medium with none),` |
| J34now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:87` | `or with WithResponses as reasoning.effort; a Budget is not sent.` |
| J35now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:90` | `recorded for the Chat Completions wire, or with WithResponses for the` |
| J36now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:91` | `Responses wire, is remembered for the rest of` |
| J37now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:98` | `"none", the Chat Completions values, which the Responses API takes` |
| J38now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:205` | `responses is WithResponses': Generate and Stream go to /responses` |
| J39now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:206` | `(0091-MADR ST50; responses.go).` |
| J40now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:294` | `Generate sends req to the server's chat completions, or with` |
| J41now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:295` | `WithResponses to its responses (0091-MADR ST50).` |
| J42now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:319` | `(0062-MADR D2 to D6), read by the Chat Completions wire's matcher, or the` |
| J43now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:320` | `Responses wire's with WithResponses (0091-MADR ST54).` |
| J44now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:329` | `Stream streams req from the Chat Completions API, or with WithResponses` |
| J45now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:330` | `from the Responses API, as it is generated (0031-MADR E1, E5; 0091-MADR` |
| J46now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:331` | `ST50): the request asks for an event stream, on Chat Completions with its` |
| J47now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:332` | `usage, and each event is yielded as it arrives. A 401 before the first event` |
| J48now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:362` | `with its usage when stream is set (0031-MADR E5); or, with WithResponses,` |
| J49now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:363` | `the Responses request (0091-MADR ST50, ST51).` |
| J50now | new doc line | `llmprovider/providers/chatcompat/limit.go:51` | `limitRetry is req's output limit across its sends. With WithResponses a` |
| J51now | new doc line | `llmprovider/providers/chatcompat/limit.go:52` | `refusal is read in the Responses wire's forms, none of which states a` |
| J52now | new doc line | `llmprovider/providers/chatcompat/limit.go:53` | `number, so the "supports at most" learning is lost (0089-MADR DL68;` |
| J53now | new doc line | `llmprovider/providers/chatcompat/limit.go:54` | `0091-MADR ST56).` |
| J54now | new doc line | `llmprovider/internal/wire/overstream.go:24` | `StreamAnswerLimit bounds an answer Collect reads, on every wire: its text,` |
| J55now | new doc line | `llmprovider/internal/wire/overstream.go:25` | `reasoning and call-argument bytes, reasoning counted so that a runaway` |
| J56now | new doc line | `llmprovider/internal/wire/overstream.go:26` | `stream stops. It is twice ReplyLimit, which a plain reply's body keeps` |
| J57now | new doc line | `llmprovider/internal/wire/overstream.go:27` | `(0091-MADR ST71, "Raise the bound", "Every provider, one bound` |
| J58now | new doc line | `llmprovider/internal/wire/overstream.go:28` | `(Recommended)"): ReplyLimit's 16 MiB allowed 32 bytes for each of the` |
| J59now | new doc line | `llmprovider/internal/wire/overstream.go:29` | `524,288 output tokens the largest recorded output limit states, and a` |
| J60now | new doc line | `llmprovider/internal/wire/overstream.go:30` | `server that streams both a summary and the raw text of its reasoning has` |
| J61now | new doc line | `llmprovider/internal/wire/overstream.go:31` | `that reasoning counted twice.` |
| J62now | new doc line | `llmprovider/internal/wire/overstream.go:45` | `bounded by StreamAnswerLimit, on top of each event's EventLimit: past` |
| J63now | new doc line | `docs/decisions/0088-MADR-default-output-limit-from-model-facts.md:1106` | `` bounded as a plain reply is, 16 MiB (`ReplyLimit`), on top of the per-event 16 MiB (`EventLimit`), as `ErrIncomplete` *(2026-11-26: its parity with a plain repl `` |
| J64now | new doc line | `llmprovider/internal/wire/responses/responses.go:2` | `grok providers and OpenCode's responses route (0015-MADR D2), and by a` |
| J65now | new doc line | `llmprovider/internal/wire/responses/responses.go:3` | `chatcompat provider built with chatcompat.WithResponses (0091-MADR ST52).` |
| J66now | new doc line | `llmprovider/internal/wire/responses/responses.go:44` | `contentReasoningText is a reasoning item's content part that carries` |
| J67now | new doc line | `llmprovider/internal/wire/responses/responses.go:45` | `its text, where a server sends no summary (0091-MADR ST10).` |
| J68now | new doc line | `llmprovider/internal/wire/responses/responses.go:377` | `Limit is the output limit the request's attempt sent, set only by` |
| J69now | new doc line | `llmprovider/internal/wire/responses/responses.go:378` | `chatcompat's Responses path (0091-MADR ST33, ST55): a reply the` |
| J70now | new doc line | `llmprovider/internal/wire/responses/responses.go:379` | `decoder would finish as completed, its status completed, absent or` |
| J71now | new doc line | `llmprovider/internal/wire/responses/responses.go:380` | `any other but failed and incomplete (ST69), whose output_tokens` |
| J72now | new doc line | `llmprovider/internal/wire/responses/responses.go:381` | `reached Limit is read as one the limit cut, as a reply incomplete for` |
| J73now | new doc line | `llmprovider/internal/wire/responses/responses.go:382` | `max_output_tokens is. On a stream it is read before the held calls` |
| J74now | new doc line | `llmprovider/internal/wire/responses/responses.go:383` | `are released and before an announced item's check (ST34, ST65). Zero,` |
| J75now | new doc line | `llmprovider/internal/wire/responses/responses.go:384` | `as openai, grok and OpenCode leave it, reads no reply as cut (ST68).` |
| J76now | new doc line | `llmprovider/internal/wire/responses/responses.go:388` | `atLimit reports a reply whose output_tokens, u's, reached d.Limit, so` |
| J77now | new doc line | `llmprovider/internal/wire/responses/responses.go:389` | `that the limit cut it however its service finished it (0091-MADR ST55).` |
| J78now | new doc line | `llmprovider/internal/wire/responses/responses.go:452` | `A call's arguments stream in pieces (0088-MADR ST107). A reply` |
| J79now | new doc line | `llmprovider/internal/wire/responses/responses.go:453` | `at the limit sent is cut whatever its status said (0091-MADR` |
| J80now | new doc line | `llmprovider/internal/wire/responses/responses.go:454` | `ST55, ST69), unless it is a refusal, which is read first (ST70).` |
| J81now | new doc line | `llmprovider/internal/wire/responses/responses.go:615` | `A server that sends its reasoning as text puts it in content, as` |
| J82now | new doc line | `llmprovider/internal/wire/responses/responses.go:616` | `reasoning_text: it is read when the summary holds none (0091-MADR` |
| J83now | new doc line | `llmprovider/internal/wire/responses/responses.go:617` | `ST10, ST63). Its text is never sent back; only encrypted content is.` |
| J84now | new doc line | `llmprovider/internal/wire/responses/responses.go:743` | `delta (0031-PLAN Phase 1) and a reasoning text delta (0091-MADR ST63) as` |
| J85now | new doc line | `llmprovider/internal/wire/responses/responses.go:744` | `EventReasoningDelta, each` |
| J86now | new doc line | `llmprovider/internal/wire/responses/responses.go:1135` | `A completed stream at the limit sent is a cut, decided before the` |
| J87now | new doc line | `llmprovider/internal/wire/responses/responses.go:1136` | `announced items' check (0091-MADR ST55, ST65); a refusal is read` |
| J88now | new doc line | `llmprovider/internal/wire/responses/responses.go:1137` | `first, and is not one (ST70).` |
| J89now | new doc line | `llmprovider/internal/wire/responses/responses.go:1184` | `each. The reasoning summary delta is the one 0031-PLAN Phase 1 measured;` |
| J90now | new doc line | `llmprovider/internal/wire/responses/responses.go:1185` | `the reasoning text delta, which probes of 2026-11-23 saw, is passed on on` |
| J91now | new doc line | `llmprovider/internal/wire/responses/responses.go:1186` | `every provider (0091-MADR ST63, ST67). A` |
| J92now | new doc line | `llmprovider/providers/chatcompat/responses.go:16` | `The Responses path, for a server the caller knows speaks OpenAI's` |
| J93now | new doc line | `llmprovider/providers/chatcompat/responses.go:17` | `Responses API (0091-MADR ST5, ST50 to ST65, OQ5 "Opt-in WithResponses()").` |
| J94now | new doc line | `llmprovider/providers/chatcompat/responses.go:19` | `responsesValue is WithResponses' value among the options.` |
| J95now | new doc line | `llmprovider/providers/chatcompat/responses.go:22` | `WithResponses sends Generate and Stream to POST {base}/responses, OpenAI's` |
| J96now | new doc line | `llmprovider/providers/chatcompat/responses.go:23` | `Responses API, in place of /chat/completions, for a server the caller` |
| J97now | new doc line | `llmprovider/providers/chatcompat/responses.go:24` | `knows speaks it (0091-MADR ST50). The listing and everything else are` |
| J98now | new doc line | `llmprovider/providers/chatcompat/responses.go:25` | `unchanged, and there is no fallback: a server that refuses the route` |
| J99now | new doc line | `llmprovider/providers/chatcompat/responses.go:26` | `answers with its error, classified by its status. It is a shared option` |
| J100now | new doc line | `llmprovider/providers/chatcompat/responses.go:27` | `(llmprovider.SharedOption), as WithID is, so another provider's New takes` |
| J101now | new doc line | `llmprovider/providers/chatcompat/responses.go:28` | `it and ignores it. The package doc names what a caller gives up on this` |
| J102now | new doc line | `llmprovider/providers/chatcompat/responses.go:29` | `path against Chat Completions.` |
| J103now | new doc line | `llmprovider/providers/chatcompat/responses.go:34` | `responsesOn reports whether values carry WithResponses.` |
| J104now | new doc line | `llmprovider/providers/chatcompat/responses.go:42` | `decode is a plain reply's decoder: the Chat Completions wire's, or the` |
| J105now | new doc line | `llmprovider/providers/chatcompat/responses.go:43` | `Responses wire's with WithResponses, labelled with the provider's id` |
| J106now | new doc line | `llmprovider/providers/chatcompat/responses.go:44` | `(0091-MADR ST52).` |
| J107now | new doc line | `llmprovider/providers/chatcompat/responses.go:54` | `events is a stream's reader: the Chat Completions wire's, or the Responses` |
| J108now | new doc line | `llmprovider/providers/chatcompat/responses.go:55` | `wire's with WithResponses.` |
| J109now | new doc line | `CHANGELOG.md:30` | `` `chatcompat.WithResponses()`: `Generate` and `Stream` go to the server's `` |
| J110now | new doc line | `CHANGELOG.md:31` | `` `POST {base}/responses`, for a server the caller knows speaks OpenAI's `` |
| J111now | new doc line | `CHANGELOG.md:32` | `Responses API. What it sends, reads and gives up is in the package doc` |
| J112now | new doc line | `CHANGELOG.md:33` | `([0091-MADR](docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md)` |
| J113now | new doc line | `CHANGELOG.md:34` | `ST50 to ST71).` |
| J114now | new doc line | `CHANGELOG.md:38` | `` `openai`, `grok` and OpenCode's responses route: a stream's raw `` |
| J115now | new doc line | `CHANGELOG.md:39` | `` reasoning deltas, `response.reasoning_text.delta`, are now yielded as `` |
| J116now | new doc line | `CHANGELOG.md:40` | `` `EventReasoningDelta`, beside the summary's; a reasoning item whose `` |
| J117now | new doc line | `CHANGELOG.md:41` | `` summary is empty is now kept, its text read from its `reasoning_text` `` |
| J118now | new doc line | `CHANGELOG.md:42` | `content, and is never sent back` |
| J119now | new doc line | `CHANGELOG.md:43` | `([0091-MADR](docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md)` |
| J120now | new doc line | `CHANGELOG.md:44` | `ST10, ST67).` |
| J121now | new doc line | `CHANGELOG.md:45` | `` Every provider, on every wire: an answer `Generate` reads over the `` |
| J122now | new doc line | `CHANGELOG.md:46` | `stream, when it names no output limit, is now bounded at 32 MiB of` |
| J123now | new doc line | `CHANGELOG.md:47` | `text, reasoning and call arguments, up from 16 MiB, since reasoning may` |
| J124now | new doc line | `CHANGELOG.md:48` | `` now be counted twice; past it, `ErrIncomplete` as before. A plain `` |
| J125now | new doc line | `CHANGELOG.md:49` | `reply's body keeps 16 MiB` |
| J126now | new doc line | `CHANGELOG.md:50` | `([0091-MADR](docs/decisions/0091-MADR-adopt-the-responses-api-per-provider.md)` |
| J127now | new doc line | `docs/architecture.md:68` | `llmprovider/providers/chatcompat/ any Chat Completions server, under the caller's id: not in Default(); Responses with WithResponses (0091-MADR ST5)` |
| J128now | new doc line | `docs/architecture.md:173` | `` `WithResponses`, which sends `Generate` and `Stream` to `{base}/responses` and reads the reply with the shared Responses decoder (0091-MADR ST50 to ST52) `` |
| J129now | new doc line | `docs/architecture.md:174` | `0091-MADR ST71)` |
| J130now | new doc line | `docs/architecture.md:175` | `` a reasoning item whose summary holds no text is read from its `reasoning_text` content (0091-MADR ST10, ST63) `` |
| J131now | new doc line | `docs/architecture.md:470` | `` `EventReasoningDelta`: Responses' reasoning summary and reasoning text `` |
| J132now | new doc line | `docs/architecture.md:471` | `(0091-MADR ST67), Messages'` |
| J133now | new doc line | `docs/architecture.md:486` | `` not run the `item_not_finished` check (ST33). On `chatcompat`'s `` |
| J134now | new doc line | `docs/architecture.md:487` | `` `WithResponses` path, a reply finished as completed whose `output_tokens` `` |
| J135now | new doc line | `docs/architecture.md:488` | `reached the limit sent is a cut, decided before that check (0091-MADR` |
| J136now | new doc line | `docs/architecture.md:489` | `ST55, ST65).` |
| J137now | new doc line | `docs/architecture.md:701` | `` **Five wire formats:** OpenAI Responses (`openai`, `grok`, and `chatcompat` `` |
| J138now | new doc line | `docs/architecture.md:702` | `` with `WithResponses`, 0091-MADR ST5), Anthropic Messages `` |
| J139now | new doc line | `README.md:131` | `of its arguments arrived (0088-MADR ST107, ST110, OQ21). On a` |
| J140now | new doc line | `README.md:132` | `` `chatcompat` provider built with `chatcompat.WithResponses()`, which sends `` |
| J141now | new doc line | `README.md:133` | `` `Generate` and `Stream` to the server's `/responses`, a reply the server `` |
| J142now | new doc line | `README.md:134` | `` finishes as completed is reported cut too when its `output_tokens` `` |
| J143now | new doc line | `README.md:135` | `` reached the limit sent, as Ollama's `/v1/responses` hides a cut `` |
| J144now | new doc line | `README.md:136` | `` (0091-MADR ST55). An empty answer is `ErrEmptyAnswer` too, with `` |
| J145now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:238` | `` (2026-11-26: a caller who gives `chatcompat.WithResponses()` has `` |
| J146now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:239` | `` `Generate` and `Stream` sent to the server's `/responses`, read by `` |
| J147now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:240` | `` `internal/wire/responses`, by `` |
| J148now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:241` | `[0091-MADR-adopt-the-responses-api-per-provider.md](0091-MADR-adopt-the-responses-api-per-provider.md)` |
| J149now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:242` | `ST5 and ST50 (OQ5 "Opt-in WithResponses()"). Without it, the package` |
| J150now | new doc line | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:243` | `speaks Chat Completions as above. The text above is unchanged.)*` |
| J151now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:260` | `(2026-11-26: since` |
| J152now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:261` | `[0091-MADR-adopt-the-responses-api-per-provider.md](0091-MADR-adopt-the-responses-api-per-provider.md)` |
| J153now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:262` | `` ST63 and ST67, it is mapped to `EventReasoningDelta` on every Responses `` |
| J154now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:263` | `provider, after probes of 2026-11-23 saw Kilo, Together and Hugging Face` |
| J155now | new doc line | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:264` | `send it. The text above is unchanged.)*` |

## Amendment (2026-11-27, eleventh): the joint pre-release review of v1.34.0

Additions only. PLAN Phase 4's joint review (its step 1) read the v1.34.0 work of this record, 0088, 0092, 0093 and 0094 at `223bb8d`. The coordinator relayed its findings and the owner's rulings; each label is quoted exactly. This amendment records the rulings that reach this record (step 2). The rulings that reach other records are in [0088-MADR-default-output-limit-from-model-facts.md](0088-MADR-default-output-limit-from-model-facts.md)'s twentieth amendment, [0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md](0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md)'s tenth amendment and [0093-MADR-take-model-metadata-from-the-caller.md](0093-MADR-take-model-metadata-from-the-caller.md)'s amendment of 2026-11-27. The PLAN's amendment of 2026-11-27 builds every row of all four in its Phase 4.

### Re-ground at `223bb8d` (2026-11-27)

* `wire.EventLimit` is 16 MiB at `llmprovider/internal/wire/sse.go:19`. `ReadEvents` returns `ErrIncomplete` "stream event over 16 MiB" for a longer line (`:58` to `:61`) or event (`:73` to `:74`).
* `wire.StreamAnswerLimit` is `2 * ReplyLimit`, 32 MiB, at `llmprovider/internal/wire/overstream.go:32`. Its doc says "on every wire" (`:24`).
* `CHANGELOG.md:45` to `:51` is the **Changed** bullet that names 32 MiB "on every wire". `:52` is a blank line inside the same **Changed** list.
* `streamState.end` is at `llmprovider/internal/wire/responses/responses.go:1124`. It reads the refusal first (`:1138`, ST70). It names each open call on a cut (`:1141` to `:1145`), and runs 0092's announced-item check on the completed path (`:1147` to `:1149`).

### Owner rulings (2026-11-27, eleventh)

| Finding | The owner's answer | Rows it decides |
| :--- | :--- | :--- |
| The 32 MiB `StreamAnswerLimit` cannot be reached on the Responses wire: the terminal `response.completed` repeats the whole `output` and is capped by `EventLimit`, 16 MiB (`sse.go:19`, `overstream.go:32`) | "Correct the record and changelog (Recommended)" | ST71 (annotated in place), ST72 |
| The review's follow-ups and its three edge cases | "Everything before release" | ST73; JR1 is put to the owner as ST74 |

### What the terminal event does to the bound (measured 2026-11-27, scratch clone at `223bb8d`)

`<scratch>/v1340prep/repro_v1340_test.go`, `TestRepro_R1`, sends two messages of 9 MiB each, read through `wire.Collect`. Each message arrives by text deltas and an `output_item.done` under 16 MiB. The stream ends in a `response.completed` whose `output` holds both, about 18 MiB. Observed: `0 bytes, err=llmprovider: incomplete response: p: stream event over 16 MiB`. An item whose own `output_item.done` is over 16 MiB fails earlier, at that event. So a Responses reply over about 16 MiB fails, whatever `Collect`'s bound. A server whose terminal `output` is empty, as the ChatGPT backend's is (0092-MADR ST5), reaches `Collect`'s 32 MiB only when every item's `.done` is under 16 MiB.

### Rows (2026-11-27, eleventh)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST72 | An answer read over a Responses stream (`openai`, `grok`, OpenCode's responses route, `chatcompat` with `WithResponses`) over `EventLimit`, 16 MiB, whose terminal event repeats it in `output` | decided | `ErrIncomplete` "stream event over 16 MiB" at the terminal event, and no response, as 0092-MADR ST29 states for one event; `Collect`'s 32 MiB is not reached. Responses replies stay capped at about 16 MiB by the terminal event; the other wires get 32 MiB (ST71). No code changes. Docs: the bullet at `CHANGELOG.md:45` restated under `[Unreleased]`; `overstream.go:24`'s "on every wire"; `docs/architecture.md:174`'s `wire` row | "Correct the record and changelog (Recommended)" | `TestCollect_ResponsesTerminalOverEventLimit_0091` (new, `llmprovider/internal/wire/responses/`), `TestRepro_R1`'s stream: `ErrIncomplete` naming "stream event over 16 MiB", no response. A green pin at `223bb8d`; its red is its plant: `EventLimit` raised to 32 MiB (`sse.go:19`), so the 18 MiB answer is returned |
| ST73 | `CHANGELOG.md:52`: a blank line inside `[Unreleased]`'s **Changed** list, which splits it in two | decided | removed before the date step. 0094-MADR freezes a released section, so after the tag it could not be removed | "Everything before release" | a one-off check, `<scratch>/v1340prep/changelog_lists.py`: no blank line between two items of one list under `[Unreleased]`. Red at `223bb8d` (`CHANGELOG.md:52`); plant: the blank line put back, on a copy, caught; `make changelog-check markdownlint` after |
| ST74 | JR1 (the review's E1): on `chatcompat`'s Responses stream, a reply finished as completed at the limit sent (ST55), holding a refusal (ST70), that announced a call delivered neither by `output_item.done` nor by the terminal response | open *(2026-11-27: decided by JR1's answer, option (1); restated in the twelfth amendment)* | JR1 below; recommended (1) | JR1; ST65; ST70; 0092-MADR ST11 | under (1): `TestChatcompat_ResponsesRefusalAtLimit_0091`, stream, gains the case: `content_filter`, one `WarningToolCallCut` naming the call, no error. Red at `223bb8d`. Plant: the completed path's check run before the refusal's branch (`item_not_finished` again) |

### Put to the owner: JR1 (the review's E1)

**JR1. A refusal at the output limit, with a call announced and never delivered: which rule decides?** ST70 says the refusal wins over the cut, so `end` does not read the reply as cut (`responses.go:1138`). ST65 says the cut wins over 0092's check. Nothing decides the case where all three occur. Today the completed path runs 0092's check (`:1147` to `:1149`), and the reply fails as `ErrProviderUnavailable` after the reply, naming `item_not_finished`. The refusal's text was already yielded as an `EventItem`.

Reproduction, `TestRepro_E1` (scratch clone at `223bb8d`, `Decoder{Limit: 5}`, usage 5 out): `err=llmprovider: provider unavailable: p: item_not_finished: function_call fc_1`, with one `MessageItem` yielded first. The controls: the same stream with text in place of the refusal finishes `length` with one `WarningToolCallCut` naming `lookup` and `call_1` (ST65); the refusal with no call finishes `content_filter` (ST70); the refusal and lost call under the limit fail `item_not_finished` (0092 ST11).

| Option | Cost |
| :--- | :--- |
| (1) **The refusal finishes the reply; the lost call is named (recommended).** The reply finishes `content_filter`. Each call announced and never delivered is dropped and named in a `WarningToolCallCut`, as on a cut. There is no error. This keeps ST70 (the refusal decides the finish) and ST65 (at the limit, a lost call is the limit's). | `end` gains one branch: refused and at the limit. A message announced and lost there is not reported, as on a cut today. |
| (2) **The limit wins when a call is lost.** The reply is read as cut: `length`, with the warning, as ST65. ST70 gives way in this one case. | A refusal is reported as a cut. A caller may send the refused request again with a higher limit. |
| (3) **Keep it as built, and document it.** The reply fails `item_not_finished`, retryable at most once, after the refusal was shown. | `WithRetry` sends the request again, and a refusal is likely again. The caller sees an outage where the server refused. |

Doing nothing ships (3) without a ruling.

Counts after this amendment: 74 rows: 55 decided, 18 out of scope, 1 open (ST74). The record is `proposed` until JR1 is answered. PLAN Phase 4 builds ST72 and ST73 now. It builds ST74 once JR1 is answered.

## Amendment (2026-11-27, twelfth): JR1 answered

Additions only. The coordinator relayed the owner's answer, quoted exactly.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| JR1 | "Refusal, plus a 'call cut' warning (Recommended)" | (1) | ST74 |

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST74 (decided 2026-11-27) | On `chatcompat`'s Responses stream, a reply finished as completed at the limit sent (ST55), holding a refusal (ST70), that announced a call delivered neither by `output_item.done` nor by the terminal response | decided | the reply finishes `content_filter`; each call announced and never delivered is dropped and named in a `WarningToolCallCut`; there is no error. Under the limit, 0092's `item_not_finished` stands (ST65) | JR1 "Refusal, plus a 'call cut' warning (Recommended)" | `TestChatcompat_ResponsesRefusalAtLimit_0091`, stream, gains the case: `content_filter`, one `WarningToolCallCut` naming the call, no error. Red at `223bb8d` (`item_not_finished`). Plant: the refused branch removed, so the completed path's check runs |

**ST72, a recorded unknown.** Whether Gemini's Interactions stream repeats its steps in `interaction.completed`, and so whether one event caps its answer below 32 MiB, is not measured. The SDK's reader takes no steps from that event (`llmprovider/providers/gemini/interactions.go:318`), and its test fixtures carry none. It is not measured now, at the coordinator's word; ST72's "the other wires get 32 MiB" is read for Interactions with this unknown.

Counts after this amendment: 74 rows: 56 decided, 18 out of scope, 0 open. The record is `accepted`.

### Claims rows of PLAN Phase 4's doc-comment lines (2026-11-27)

One row per new Go doc-comment line of PLAN Phase 4's step 4a, in `wire/overstream.go`, `wire/finish.go`, `wire/responses/responses.go`, `chatcompat/chatcompat.go`, `llmprovider/registry.go` and `wizard/configure.go` (ST72, ST74; 0088-MADR ST115, ST116; 0092-MADR ST37, ST38; 0093-MADR MH62). Ids end in "now"; the phase's commit re-pins them. Scripts: `<scratch>/v1340prep/claims_rows.py`.

* **Red at the base.** At `223bb8d`, each row at line 1 of its file: all 52 fail on their assertion, and none finds its text on any line of its file.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 52 of 52 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 52 of 52 caught.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| K1now | new doc line | `llmprovider/internal/wire/overstream.go:31` | `that reasoning counted twice. A Responses answer stays at about 16 MiB:` |
| K2now | new doc line | `llmprovider/internal/wire/overstream.go:32` | `its terminal event repeats the whole answer, and one event is bounded by` |
| K3now | new doc line | `llmprovider/internal/wire/overstream.go:33` | `EventLimit, so a longer one is ErrIncomplete at that event (ST72).` |
| K4now | new doc line | `llmprovider/internal/wire/finish.go:35` | `CutAnswerLast is CutAnswer with the reply's last item, where the limit` |
| K5now | new doc line | `llmprovider/internal/wire/finish.go:36` | `fell, at last in r.Output, or -1 when no item there is last: a Responses` |
| K6now | new doc line | `llmprovider/internal/wire/finish.go:37` | `stream that took items from its terminal response reads it in the server's` |
| K7now | new doc line | `llmprovider/internal/wire/finish.go:38` | `order, which r.Output's need not be (0092-MADR ST37).` |
| K8now | new doc line | `llmprovider/internal/wire/responses/responses.go:383` | `are released and before an announced item's check (ST34, ST65). A` |
| K9now | new doc line | `llmprovider/internal/wire/responses/responses.go:384` | `refusal is read first and finishes content_filter (ST70); a call` |
| K10now | new doc line | `llmprovider/internal/wire/responses/responses.go:385` | `announced and never delivered is then named in a WarningToolCallCut` |
| K11now | new doc line | `llmprovider/internal/wire/responses/responses.go:386` | `(ST74). Zero, as openai, grok and OpenCode leave it, reads no reply as` |
| K12now | new doc line | `llmprovider/internal/wire/responses/responses.go:833` | `announcedKeys are the announced items' keys, by id, call id and` |
| K13now | new doc line | `llmprovider/internal/wire/responses/responses.go:834` | `output_index with type, so a repeat is found without a scan (ST34,` |
| K14now | new doc line | `llmprovider/internal/wire/responses/responses.go:878` | `announcement is the same item, recorded once: the same id (ST34); else,` |
| K15now | new doc line | `llmprovider/internal/wire/responses/responses.go:879` | `with no id, the same call id; else, with neither, the same output_index` |
| K16now | new doc line | `llmprovider/internal/wire/responses/responses.go:880` | `and type (ST38). One with none of the three is recorded each time. Once` |
| K17now | new doc line | `llmprovider/internal/wire/responses/responses.go:881` | `the announced items pass wire.MaxStreamIndex, where ST11's check is off` |
| K18now | new doc line | `llmprovider/internal/wire/responses/responses.go:882` | `(ST33), no more are recorded, so each announcement costs constant work.` |
| K19now | new doc line | `llmprovider/internal/wire/responses/responses.go:912` | `announceKey is the key a repeat of item, announced at index, is found by:` |
| K20now | new doc line | `llmprovider/internal/wire/responses/responses.go:913` | `its id; else its call id; else its output_index and type; "" for none` |
| K21now | new doc line | `llmprovider/internal/wire/responses/responses.go:914` | `(0092-MADR ST34, ST38).` |
| K22now | new doc line | `llmprovider/internal/wire/responses/responses.go:927` | `closes reports whether item, finished by an output_item.done or taken` |
| K23now | new doc line | `llmprovider/internal/wire/responses/responses.go:928` | `from the terminal response, is the open call o: the same call id, else` |
| K24now | new doc line | `llmprovider/internal/wire/responses/responses.go:929` | `the same id, each compared only when item carries it (0088-MADR ST115,` |
| K25now | new doc line | `llmprovider/internal/wire/responses/responses.go:930` | `ST116). A message's or a reasoning item's .done, which carries no call id,` |
| K26now | new doc line | `llmprovider/internal/wire/responses/responses.go:931` | `closes no call announced without one.` |
| K27now | new doc line | `llmprovider/internal/wire/responses/responses.go:1158` | `and is finished by CutAnswer, the reply's last item read in the server's` |
| K28now | new doc line | `llmprovider/internal/wire/responses/responses.go:1159` | `order (0092-MADR ST37); a refusal at the limit sent names them too, and` |
| K29now | new doc line | `llmprovider/internal/wire/responses/responses.go:1160` | `finishes content_filter (0091-MADR ST74); a completed one fails when an` |
| K30now | new doc line | `llmprovider/internal/wire/responses/responses.go:1161` | `announced item was never delivered (ST11), and is otherwise finished by` |
| K31now | new doc line | `llmprovider/internal/wire/responses/responses.go:1162` | `completed; then the held calls and the taken items the response kept are` |
| K32now | new doc line | `llmprovider/internal/wire/responses/responses.go:1163` | `yielded, and EventDone (0088-MADR ST77, ST80). Its results are a stream` |
| K33now | new doc line | `llmprovider/internal/wire/responses/responses.go:1188` | `The refusal finishes the reply; a call announced and never` |
| K34now | new doc line | `llmprovider/internal/wire/responses/responses.go:1189` | `delivered is the limit's, named (0091-MADR ST74).` |
| K35now | new doc line | `llmprovider/internal/wire/responses/responses.go:1213` | `nameOpen names each call added and never done in a WarningToolCallCut` |
| K36now | new doc line | `llmprovider/internal/wire/responses/responses.go:1214` | `on result (0088-MADR ST77, ST108).` |
| K37now | new doc line | `llmprovider/internal/wire/responses/responses.go:1221` | `lastIndex is the place in s.result.Output of the reply's last item, where` |
| K38now | new doc line | `llmprovider/internal/wire/responses/responses.go:1222` | `the limit fell, in the server's order (0092-MADR ST37): the terminal` |
| K39now | new doc line | `llmprovider/internal/wire/responses/responses.go:1223` | `output's last item. When it is a call, it is the response's call with its` |
| K40now | new doc line | `llmprovider/internal/wire/responses/responses.go:1224` | `call id; when it is not a call, no item is last, -1. With no terminal` |
| K41now | new doc line | `llmprovider/internal/wire/responses/responses.go:1225` | `output, past the bound (ST33), or a last call whose call id the response` |
| K42now | new doc line | `llmprovider/internal/wire/responses/responses.go:1226` | `does not hold, it is the response's own last item, as before.` |
| K43now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:65` | `A refusal finishes content_filter, at the limit or under it (ST70); at the` |
| K44now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:66` | `limit, a call the stream announced and never delivered is named in a` |
| K45now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:67` | `WarningToolCallCut (ST74).` |
| K46now | new doc line | `llmprovider/registry.go:34` | `StaticModels is the curated fallback catalog, or empty: model ids` |
| K47now | new doc line | `llmprovider/registry.go:35` | `only, the wizard's built-in catalog, which carries no model facts` |
| K48now | new doc line | `llmprovider/registry.go:36` | `since 0093-MADR.` |
| K49now | new doc line | `wizard/configure.go:109` | `path and the built-in catalog alike (the descriptor's curated model` |
| K50now | new doc line | `wizard/configure.go:110` | `ids, which carry no facts since 0093-MADR). Zero recommends the whole curated` |
| K51now | new doc line | `wizard/configure.go:1111` | `from the check that follows, so a warning about a built-in catalog (the` |
| K52now | new doc line | `wizard/configure.go:1112` | `descriptor's curated model ids, with no facts since 0093-MADR) would` |

### Note (2026-11-27): ST72 qualified

Additions only, from the code review of [0091-PLAN-adopt-the-responses-api-per-provider.md](0091-PLAN-adopt-the-responses-api-per-provider.md)'s Phase 4 (its pass 2). ST72's "Responses replies stay capped at about 16 MiB by the terminal event" holds when the terminal event repeats the answer, as OpenAI's does. A server whose terminal `output` is empty, as the ChatGPT backend's is (0092-MADR ST5), reaches `StreamAnswerLimit`, 32 MiB, when no single item's event passes 16 MiB. `CHANGELOG.md`, `overstream.go`'s doc and `docs/architecture.md`'s `wire` row say so. The changelog's `WithResponses` entry now cites ST50 to ST72 and ST74, with a clause for ST74.

### Claims rows of PLAN Phase 4's doc-comment lines, pass 2 (2026-11-27)

One row per new Go doc-comment line after step 4a's second pass, the same files as pass 1's rows. They supersede K1now to K52now, which pinned lines pass 2 moved or changed (left as written above; the ids do not clash). Ids end in "now"; the phase's commit re-pins them. Scripts: `<scratch>/v1340prep/claims_rows.py L`.

* **Red at the base.** At `223bb8d`, each row at line 1 of its file: all 59 fail on their assertion, and none finds its text on any line of its file.
* **Green after the change.** On a scratch clone holding the working tree, committed there: 59 of 59 pass, exit 0.
* **Plants.** `tools/claims.py --plants`, each row's text made false: 59 of 59 caught.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| L1now | new doc line | `llmprovider/internal/wire/overstream.go:31` | `that reasoning counted twice. A Responses answer stays at about 16 MiB` |
| L2now | new doc line | `llmprovider/internal/wire/overstream.go:32` | `when its terminal event repeats the whole answer, as OpenAI's does: one` |
| L3now | new doc line | `llmprovider/internal/wire/overstream.go:33` | `event is bounded by EventLimit, so a longer one is ErrIncomplete at that` |
| L4now | new doc line | `llmprovider/internal/wire/overstream.go:34` | `event (ST72). A terminal with an empty output, as the ChatGPT backend` |
| L5now | new doc line | `llmprovider/internal/wire/overstream.go:35` | `sends, leaves this bound to decide.` |
| L6now | new doc line | `llmprovider/internal/wire/finish.go:35` | `CutAnswerLast is CutAnswer with the reply's last item, where the limit` |
| L7now | new doc line | `llmprovider/internal/wire/finish.go:36` | `fell, at last in r.Output, or -1 when no item there is last: a Responses` |
| L8now | new doc line | `llmprovider/internal/wire/finish.go:37` | `stream that took items from its terminal response reads it in the server's` |
| L9now | new doc line | `llmprovider/internal/wire/finish.go:38` | `order, which r.Output's need not be (0092-MADR ST37).` |
| L10now | new doc line | `llmprovider/internal/wire/finish.go:95` | `Holds reports whether item is a call held now, not yet yielded.` |
| L11now | new doc line | `llmprovider/internal/wire/responses/responses.go:383` | `are released and before an announced item's check (ST34, ST65). A` |
| L12now | new doc line | `llmprovider/internal/wire/responses/responses.go:384` | `refusal is read first and finishes content_filter (ST70); a call` |
| L13now | new doc line | `llmprovider/internal/wire/responses/responses.go:385` | `announced and never delivered is then named in a WarningToolCallCut` |
| L14now | new doc line | `llmprovider/internal/wire/responses/responses.go:386` | `(ST74). Zero, as openai, grok and OpenCode leave it, reads no reply as` |
| L15now | new doc line | `llmprovider/internal/wire/responses/responses.go:833` | `announcedKeys are the announced items' keys, by id, call id and` |
| L16now | new doc line | `llmprovider/internal/wire/responses/responses.go:834` | `output_index with type, so a repeat is found without a scan (ST34,` |
| L17now | new doc line | `llmprovider/internal/wire/responses/responses.go:878` | `announcement is the same item, recorded once: the same id (ST34); else,` |
| L18now | new doc line | `llmprovider/internal/wire/responses/responses.go:879` | `with no id, the same call id; else, with neither, the same output_index` |
| L19now | new doc line | `llmprovider/internal/wire/responses/responses.go:880` | `and type (ST38). One with none of the three is recorded each time. Once` |
| L20now | new doc line | `llmprovider/internal/wire/responses/responses.go:881` | `the announced items pass wire.MaxStreamIndex, where ST11's check is off` |
| L21now | new doc line | `llmprovider/internal/wire/responses/responses.go:882` | `(ST33), no more are recorded, so each announcement costs constant work.` |
| L22now | new doc line | `llmprovider/internal/wire/responses/responses.go:912` | `announceKey is the key a repeat of item, announced at index, is found by:` |
| L23now | new doc line | `llmprovider/internal/wire/responses/responses.go:913` | `its id; else its call id; else its output_index and type; "" for none` |
| L24now | new doc line | `llmprovider/internal/wire/responses/responses.go:914` | `(0092-MADR ST34, ST38).` |
| L25now | new doc line | `llmprovider/internal/wire/responses/responses.go:927` | `closes reports whether item, finished by an output_item.done or taken` |
| L26now | new doc line | `llmprovider/internal/wire/responses/responses.go:928` | `from the terminal response, is the open call o: by call id when both` |
| L27now | new doc line | `llmprovider/internal/wire/responses/responses.go:929` | `carry one; else by id, when item carries one (0088-MADR ST115, ST116). Two` |
| L28now | new doc line | `llmprovider/internal/wire/responses/responses.go:930` | `different call ids never match through an id. A message's or a reasoning` |
| L29now | new doc line | `llmprovider/internal/wire/responses/responses.go:931` | `item's .done, which carries no call id, closes no call announced without` |
| L30now | new doc line | `llmprovider/internal/wire/responses/responses.go:932` | `one unless their ids match.` |
| L31now | new doc line | `llmprovider/internal/wire/responses/responses.go:1159` | `and is finished by CutAnswer, the reply's last item read in the server's` |
| L32now | new doc line | `llmprovider/internal/wire/responses/responses.go:1160` | `order (0092-MADR ST37); a refusal at the limit sent names them too, and` |
| L33now | new doc line | `llmprovider/internal/wire/responses/responses.go:1161` | `finishes content_filter (0091-MADR ST74); a completed one fails when an` |
| L34now | new doc line | `llmprovider/internal/wire/responses/responses.go:1162` | `announced item was never delivered (ST11), and is otherwise finished by` |
| L35now | new doc line | `llmprovider/internal/wire/responses/responses.go:1163` | `completed; then the held calls and the taken items the response kept are` |
| L36now | new doc line | `llmprovider/internal/wire/responses/responses.go:1164` | `yielded, and EventDone (0088-MADR ST77, ST80). Its results are a stream` |
| L37now | new doc line | `llmprovider/internal/wire/responses/responses.go:1189` | `The refusal finishes the reply; a call announced and never` |
| L38now | new doc line | `llmprovider/internal/wire/responses/responses.go:1190` | `delivered is the limit's, named (0091-MADR ST74).` |
| L39now | new doc line | `llmprovider/internal/wire/responses/responses.go:1214` | `nameOpen names each call added and never done in a WarningToolCallCut` |
| L40now | new doc line | `llmprovider/internal/wire/responses/responses.go:1215` | `on result (0088-MADR ST77, ST108).` |
| L41now | new doc line | `llmprovider/internal/wire/responses/responses.go:1222` | `lastIndex is the place in s.result.Output of the reply's last item, where` |
| L42now | new doc line | `llmprovider/internal/wire/responses/responses.go:1223` | `the limit fell, in the server's order (0092-MADR ST37): the terminal` |
| L43now | new doc line | `llmprovider/internal/wire/responses/responses.go:1224` | `output's last item. When it is a call, it is the response's call with its` |
| L44now | new doc line | `llmprovider/internal/wire/responses/responses.go:1225` | `call id; when it is not a call, no item is last, -1. With no terminal` |
| L45now | new doc line | `llmprovider/internal/wire/responses/responses.go:1226` | `output, past the bound (ST33), or a last call whose call id the response` |
| L46now | new doc line | `llmprovider/internal/wire/responses/responses.go:1227` | `does not hold, it is the response's own last item, as before. The cut` |
| L47now | new doc line | `llmprovider/internal/wire/responses/responses.go:1228` | `applies only to a call not yet yielded, one still held or one taken from` |
| L48now | new doc line | `llmprovider/internal/wire/responses/responses.go:1229` | `the terminal: a call a later .done released was yielded, so it is kept,` |
| L49now | new doc line | `llmprovider/internal/wire/responses/responses.go:1230` | `and no item is last, -1 (ST9; 0088-MADR ST109).` |
| L50now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:65` | `A refusal finishes content_filter, at the limit or under it (ST70); at the` |
| L51now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:66` | `limit, a call the stream announced and never delivered is named in a` |
| L52now | new doc line | `llmprovider/providers/chatcompat/chatcompat.go:67` | `WarningToolCallCut (ST74).` |
| L53now | new doc line | `llmprovider/registry.go:34` | `StaticModels is the curated fallback catalog, or empty: model ids` |
| L54now | new doc line | `llmprovider/registry.go:35` | `only, the wizard's built-in catalog, which carries no model facts` |
| L55now | new doc line | `llmprovider/registry.go:36` | `since 0093-MADR.` |
| L56now | new doc line | `wizard/configure.go:109` | `path and the built-in catalog alike (the descriptor's curated model` |
| L57now | new doc line | `wizard/configure.go:110` | `ids, which carry no facts since 0093-MADR). Zero recommends the whole curated` |
| L58now | new doc line | `wizard/configure.go:1111` | `from the check that follows, so a warning about a built-in catalog (the` |
| L59now | new doc line | `wizard/configure.go:1112` | `descriptor's curated model ids, with no facts since 0093-MADR) would` |

*(2026-11-27: taken up by [0095-MADR-clear-the-sdk-follow-ups.md](0095-MADR-clear-the-sdk-follow-ups.md). The follow-up on summary and raw reasoning deltas for one item is its FU03, left as it is by the owner's ruling. ST72's recorded unknown is measured in FU23: `interaction.completed` carries no steps, so `StreamAnswerLimit` bounds an Interactions answer, as ST72 read it.)*
