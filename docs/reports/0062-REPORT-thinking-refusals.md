---
date: 2026-10-25
subject: how each wire answers a thinking parameter it refuses, and a failure of another cause, for 0062-MADR D4 and D8
examines: "51 generations sent on 2026-10-25 by llmprovider/live_0062_refusals_measure_test.go through the SDK's own providers, each with one field rewritten; the records are llmprovider/internal/wire/testdata/refusals-0062/"
---

<!-- markdownlint-disable MD013 -->

# Thinking Refusals on Each Wire, Recorded for 0062

This report records findings. It decides nothing. The decisions it bears on
belong to
[0062-MADR-learn-thinking-parameters-from-refusals.md](../decisions/0062-MADR-learn-thinking-parameters-from-refusals.md),
whose amendment "Phase 1's evidence" reads them, and its PLAN's Phase 1
([0062-PLAN-learn-thinking-parameters-from-refusals.md](../decisions/0062-PLAN-learn-thinking-parameters-from-refusals.md))
recorded them. Every finding here is measured, from the records.

## How they were recorded

* **The recorder** builds each provider with `providers.New`, with
  `WithModelProbes(false)`, `WithoutModelMetadata()` and the default output
  limit, 8192, and sends `Generate` with the input "Reply with OK." and
  `Reasoning{Effort: EffortHigh}`. A transport rewrites the outgoing JSON
  body to carry the case's fields, setting some and dropping others, so the
  request leaves with the provider's own headers and credential and its reply
  reaches the `*APIError` as a matcher will see it.
* **Each record** holds the fields sent, the sends made, the status, the
  `*APIError`'s code and message, the raw error body redacted by
  `internal/redact` and cut to 4 KiB, whether the reply counts usage, and an
  accepted reply's reasoning tokens. Never a key or the model's text.
  Claude's five request ids and Together's one were redacted after the run
  (`req_REDACTED`, `REDACTED`), and `make fixtures-check` passes.
* **The outcome is the status's alone:** 404 `unreachable`; any other 4xx but
  401, 403, 408, 409 and 429 `refused`; anything else `other`. The tables keep
  it as recorded; the findings below read each message for its cause.
* **The run.** All 51 cases ran on 2026-10-25 with one send each. On the first
  run OpenCode's key had expired, and 15 of its 16 cases were answered 401
  `api_key_expired`; the owner renewed the key and those 16 reran alone. No
  case is `skipped` or 401.

| Outcome | Status | Cases |
| :--- | ---: | ---: |
| accepted | 200 | 19 |
| refused | 400 | 31 |
| unreachable | 404 | 1 |

## Findings

* **E1. Claude refuses each shape it does not take, by name.**
  `claude-sonnet-5` refuses `thinking.type` `"enabled"` and names the
  alternative ("Use \"thinking.type.adaptive\" and \"output_config.effort\"");
  `claude-sonnet-4-5-20250929` refuses `"adaptive"` ("adaptive thinking is not
  supported on this model"). Neither lists accepted values.
* **E2. An effort outside a service's vocabulary is refused with the
  vocabulary,** on Claude (`output_config.effort: Input should be …`), OpenAI
  and OpenCode's responses route (`invalid_value`, "Supported values are:
  …"), Gemini's Interactions ("Supported values: …"), Hugging Face
  (`reasoning_effort (bogus): Input should be …`) and Kilo ("Invalid option:
  expected one of …").
* **E3. An effort a model does not take, inside the vocabulary, is refused
  per model.**
  * OpenCode's responses route on `gpt-5.5`: `unsupported_value`, "'max' is
    not supported with the 'gpt-5.5' model. Supported values are: 'none',
    'low', 'medium', 'high', and 'xhigh'."
  * OpenAI on `gpt-4.1-mini`: `unsupported_parameter`, "'reasoning.effort' is
    not supported with this model": the field, not a value.
  * Grok on `grok-4.5` and `grok-4.3`: `max` is `invalid-argument`, "Invalid
    reasoning effort.", with no value and no accepted values. `xhigh` is
    accepted on `grok-4.5`.
  * Gemini on `gemini-3.8-flash`: `max` and `xhigh` are refused with the
    vocabulary minimal, low, medium, high.
  * Gemini on `gemini-2.5-flash-lite`: `medium` is refused with "Allowed
    values are: low, high"; `low` is refused as "The thinking budget 256 is
    invalid. Please choose a value between 512 and 24576.", naming a budget
    the request did not send.
* **E4. Some values a name rule avoids are accepted.**
  `qwen3.8-flash` on OpenCode Go's messages route accepts adaptive thinking
  (R1 sends it a budget); `minimax-m3` there accepts adaptive with
  `output_config.effort` `"high"` (R2 sends no effort); OpenCode Zen's
  google route accepts `thinkingLevel` `"max"` on `gemini-3.8-flash`, which
  Gemini's own Interactions refuses, and a `thinkingBudget` of 4096; Zen's
  chat route accepts `reasoning_effort` `"max"` on `glm-5.3` with no
  metadata.
* **E5. One service accepts a value outside its vocabulary.** Together
  accepts `reasoning_effort` `"bogus"` on `openai/gpt-oss-120b` and reasons
  (38 reasoning tokens), as Hugging Face's documentation allows for its
  providers (0062-MADR F4). Every other service sent a bogus effort refuses
  it.
* **E6. No refusal counts usage.** None of the 32 replies that are not 2xx
  carries a usage field. Whether a refused request is billed is not
  measured by this; no service's reply says.
* **E7. Gemini 2.x is gone from OpenCode's google route.**
  `gemini-2.5-flash` there is 400 `UNKNOWN`, "Upstream request failed: Model
  is unavailable." The status rule recorded it `refused`; its cause is the
  model.
* **E8. A refused value's parameter is not always in the message.**
  OpenAI's `invalid_value` and `unsupported_value`, and Kilo's "Invalid
  option", carry the parameter only in the body's `param`, which the
  `*APIError` does not keep (0062-MADR F6). The message holds the value sent,
  or a list made only of effort words.
* **E9. Failures of other causes look like refusals by status and code,
  and differ by message.**
  * An empty input is 400 on every provider and route but two: Grok
    accepts an empty
    `input` and answers; Gemini's Interactions answers it 404 `not_found`,
    "Model 'gemini-3.8-flash' not found", so a 404 is not always an
    unreachable model. OpenCode Go's messages route answers it 400 with the
    body `{"model":"qwen3.8-flash"}`, no code and no message.
  * A tool schema of `{"type":"string"}` is refused by Claude
    (`invalid_request_error`, the code of its thinking refusals), OpenAI and
    OpenCode's responses route (`invalid_function_parameters`), Grok
    (`invalid-argument`, the code of its effort refusal) and OpenCode's
    google route; Gemini, Hugging Face, Together, Kilo and OpenCode's chat
    and messages routes accept it.

## The records, by wire

The message is the `*APIError`'s, as a matcher reads it. An accepted case
shows its reasoning tokens; a gateway may not count them: `qwen3.8-flash`
reported 0 under both shapes, adaptive in its thinking case and a budget, the
SDK's default for it, in its tool case.

### Messages

Thinking cases:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `claude-sonnet-4-5-adaptive-high` | claude | `claude-sonnet-4-5-20250929` | `output_config.effort` "high"; `thinking` {"type": "adaptive"} | refused | 400 | `invalid_request_error` | adaptive thinking is not supported on this model |
| `claude-sonnet-5-adaptive-bogus` | claude | `claude-sonnet-5` | `output_config.effort` "bogus"; `thinking` {"type": "adaptive"} | refused | 400 | `invalid_request_error` | output_config.effort: Input should be 'low', 'medium', 'high', 'xhigh' or 'max' |
| `claude-sonnet-5-adaptive-max` | claude | `claude-sonnet-5` | `output_config.effort` "max"; `thinking` {"type": "adaptive"} | accepted (35 reasoning tokens) | 200 |  |  |
| `claude-sonnet-5-budget` | claude | `claude-sonnet-5` | `thinking` {"budget_tokens": 4096, "type": "enabled"}; `output_config` dropped | refused | 400 | `invalid_request_error` | "thinking.type.enabled" is not supported for this model. Use "thinking.type.adaptive" and "output_config.effort" to control thinking behavior. |
| `opencode-go-messages-minimax-m3-adaptive-high` | opencode-go/messages | `minimax-m3` | `output_config.effort` "high"; `thinking` {"type": "adaptive"} | accepted (10 reasoning tokens) | 200 |  |  |
| `opencode-go-messages-qwen3.8-flash-adaptive` | opencode-go/messages | `qwen3.8-flash` | `thinking` {"type": "adaptive"}; `output_config` dropped | accepted (0 reasoning tokens) | 200 |  |  |

Other causes:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `claude-empty-input` | claude | `claude-sonnet-5` | `messages` [] | refused | 400 | `invalid_request_error` | messages: at least one message is required |
| `claude-string-schema-tool` | claude | `claude-sonnet-5` | a tool, schema `{"type":"string"}` | refused | 400 | `invalid_request_error` | tools.0.custom.input_schema.type: Input should be 'object' |
| `opencode-go-messages-empty-input` | opencode-go/messages | `qwen3.8-flash` | `messages` [] | refused | 400 |  |  |
| `opencode-go-messages-string-schema-tool` | opencode-go/messages | `qwen3.8-flash` | a tool, schema `{"type":"string"}` | accepted (0 reasoning tokens) | 200 |  |  |

### Responses

Thinking cases:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `grok-4.3-max` | grok | `grok-4.3` | `reasoning.effort` "max" | refused | 400 | `invalid-argument` | Invalid reasoning effort. |
| `grok-4.5-max` | grok | `grok-4.5` | `reasoning.effort` "max" | refused | 400 | `invalid-argument` | Invalid reasoning effort. |
| `grok-4.5-xhigh` | grok | `grok-4.5` | `reasoning.effort` "xhigh" | accepted (24 reasoning tokens) | 200 |  |  |
| `openai-gpt-4.1-mini-low` | openai | `gpt-4.1-mini` | `reasoning.effort` "low" | refused | 400 | `unsupported_parameter` | Unsupported parameter: 'reasoning.effort' is not supported with this model. |
| `openai-gpt-6-luna-bogus` | openai | `gpt-6-luna` | `reasoning.effort` "bogus" | refused | 400 | `invalid_value` | Invalid value: 'bogus'. Supported values are: 'none', 'minimal', 'low', 'medium', 'high', 'xhigh', and 'max'. |
| `openai-gpt-6-luna-max` | openai | `gpt-6-luna` | `reasoning.effort` "max" | accepted (0 reasoning tokens) | 200 |  |  |
| `opencode-zen-responses-gpt-5.5-bogus` | opencode-zen/responses | `gpt-5.5` | `reasoning.effort` "bogus" | refused | 400 | `invalid_value` | Invalid value: 'bogus'. Supported values are: 'none', 'minimal', 'low', 'medium', 'high', 'xhigh', and 'max'. |
| `opencode-zen-responses-gpt-5.5-max` | opencode-zen/responses | `gpt-5.5` | `reasoning.effort` "max" | refused | 400 | `unsupported_value` | Unsupported value: 'max' is not supported with the 'gpt-5.5' model. Supported values are: 'none', 'low', 'medium', 'high', and 'xhigh'. |

Other causes:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `grok-empty-input` | grok | `grok-4.5` | `input` [] | accepted (77 reasoning tokens) | 200 |  |  |
| `grok-string-schema-tool` | grok | `grok-4.5` | a tool, schema `{"type":"string"}` | refused | 400 | `invalid-argument` | Failed to start sampling: [invalid_client_tool_schema] record_city: tool parameter root must be an object type (root schema is not an object type) |
| `openai-empty-input` | openai | `gpt-6-luna` | `input` [] | refused | 400 | `missing_required_parameter` | One of "input" or "previous_response_id" or 'prompt' or 'conversation' must be provided. |
| `openai-string-schema-tool` | openai | `gpt-6-luna` | a tool, schema `{"type":"string"}` | refused | 400 | `invalid_function_parameters` | Invalid schema for function 'record_city': schema must be a JSON Schema of 'type: "object"', got 'type: "string"'. |
| `opencode-zen-responses-empty-input` | opencode-zen/responses | `gpt-5.5` | `input` [] | refused | 400 | `missing_required_parameter` | One of "input" or "previous_response_id" or 'prompt' or 'conversation' must be provided. |
| `opencode-zen-responses-string-schema-tool` | opencode-zen/responses | `gpt-5.5` | a tool, schema `{"type":"string"}` | refused | 400 | `invalid_function_parameters` | Invalid schema for function 'record_city': schema must be a JSON Schema of 'type: "object"', got 'type: "string"'. |

### Interactions

Thinking cases:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `gemini-2.5-flash-lite-low` | gemini | `gemini-2.5-flash-lite` | `generation_config.thinking_level` "low" | refused | 400 | `invalid_request` | The thinking budget 256 is invalid. Please choose a value between 512 and 24576. |
| `gemini-2.5-flash-lite-medium` | gemini | `gemini-2.5-flash-lite` | `generation_config.thinking_level` "medium" | refused | 400 | `invalid_request` | 'medium' is not a supported thinking level for this model. Allowed values are: low, high. |
| `gemini-3.8-flash-bogus` | gemini | `gemini-3.8-flash` | `generation_config.thinking_level` "bogus" | refused | 400 | `invalid_request` | The value 'bogus' is not supported for 'generation_config.thinking_level'. Supported values: 'minimal', 'low', 'medium', 'high'. |
| `gemini-3.8-flash-max` | gemini | `gemini-3.8-flash` | `generation_config.thinking_level` "max" | refused | 400 | `invalid_request` | The value 'max' is not supported for 'generation_config.thinking_level'. Supported values: 'minimal', 'low', 'medium', 'high'. |
| `gemini-3.8-flash-xhigh` | gemini | `gemini-3.8-flash` | `generation_config.thinking_level` "xhigh" | refused | 400 | `invalid_request` | The value 'xhigh' is not supported for 'generation_config.thinking_level'. Supported values: 'minimal', 'low', 'medium', 'high'. |

Other causes:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `gemini-empty-input` | gemini | `gemini-3.8-flash` | `input` [] | unreachable | 404 | `not_found` | Model 'gemini-3.8-flash' not found. Please verify the model name against the supported list: <https://ai.google.dev/gemini-api/docs/models> |
| `gemini-string-schema-tool` | gemini | `gemini-3.8-flash` | a tool, schema `{"type":"string"}` | accepted (89 reasoning tokens) | 200 |  |  |

### generateContent

Thinking cases:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `opencode-zen-google-gemini-2.5-flash-level-low` | opencode-zen/google | `gemini-2.5-flash` | `generationConfig.thinkingConfig.thinkingLevel` "low"; `generationConfig.thinkingConfig.thinkingBudget` dropped | refused | 400 | `UNKNOWN` | Upstream request failed: Model is unavailable. |
| `opencode-zen-google-gemini-3.8-flash-budget` | opencode-zen/google | `gemini-3.8-flash` | `generationConfig.thinkingConfig.thinkingBudget` 4096; `generationConfig.thinkingConfig.thinkingLevel` dropped | accepted (63 reasoning tokens) | 200 |  |  |
| `opencode-zen-google-gemini-3.8-flash-level-max` | opencode-zen/google | `gemini-3.8-flash` | `generationConfig.thinkingConfig.thinkingLevel` "max"; `generationConfig.thinkingConfig.thinkingBudget` dropped | accepted (55 reasoning tokens) | 200 |  |  |

Other causes:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `opencode-zen-google-empty-input` | opencode-zen/google | `gemini-3.8-flash` | `contents` [] | refused | 400 | `INVALID_ARGUMENT` | Upstream request failed: [400] Unable to submit request because at least one contents field is required. Learn more: <https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/gemini> |
| `opencode-zen-google-string-schema-tool` | opencode-zen/google | `gemini-3.8-flash` | a tool, schema `{"type":"string"}` | refused | 400 | `INVALID_ARGUMENT` | Upstream request failed: [400] Unable to submit request because `record_city` functionDeclaration parameters schema should be of type OBJECT. Learn more: <https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/function-calling> |

### Chat Completions

Thinking cases:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `huggingface-deepseek-v4.1-flash-bogus` | huggingface | `deepseek-ai/DeepSeek-V4.1-Flash` | `reasoning_effort` "bogus" | refused | 400 | `400` | Invalid request: ["reasoning_effort (bogus): Input should be 'minimal', 'low', 'medium', 'high', 'xhigh', 'max' or 'none'"] |
| `huggingface-deepseek-v4.1-flash-max` | huggingface | `deepseek-ai/DeepSeek-V4.1-Flash` | `reasoning_effort` "max" | accepted (16 reasoning tokens) | 200 |  |  |
| `kilo-gpt-6-luna-bogus` | kilo | `openai/gpt-6-luna` | `reasoning.effort` "bogus" | refused | 400 | `invalid_request_error` | Invalid option: expected one of "none"\|"minimal"\|"low"\|"medium"\|"high"\|"xhigh"\|"max" |
| `kilo-gpt-6-luna-max` | kilo | `openai/gpt-6-luna` | `reasoning.effort` "max" | accepted (0 reasoning tokens) | 200 |  |  |
| `opencode-zen-chat-glm-5.3-max` | opencode-zen/chat_completions | `glm-5.3` | `reasoning_effort` "max" | accepted (72 reasoning tokens) | 200 |  |  |
| `together-gpt-oss-120b-bogus` | together | `openai/gpt-oss-120b` | `reasoning_effort` "bogus" | accepted (38 reasoning tokens) | 200 |  |  |
| `together-gpt-oss-120b-max` | together | `openai/gpt-oss-120b` | `reasoning_effort` "max" | accepted (27 reasoning tokens) | 200 |  |  |

Other causes:

| Case | Provider | Model | Sent | Outcome | Status | Code | Message, redacted |
| :--- | :--- | :--- | :--- | :--- | ---: | :--- | :--- |
| `huggingface-empty-input` | huggingface | `deepseek-ai/DeepSeek-V4.1-Flash` | `messages` [] | refused | 400 | `400` | Validation: The 'messages' field cannot be empty. At least one message is required. |
| `huggingface-string-schema-tool` | huggingface | `deepseek-ai/DeepSeek-V4.1-Flash` | a tool, schema `{"type":"string"}` | accepted (21 reasoning tokens) | 200 |  |  |
| `kilo-empty-input` | kilo | `openai/gpt-6-luna` | `messages` [] | refused | 400 | `invalid_request_error` | At least one message is required |
| `kilo-string-schema-tool` | kilo | `openai/gpt-6-luna` | a tool, schema `{"type":"string"}` | accepted (0 reasoning tokens) | 200 |  |  |
| `opencode-zen-chat-empty-input` | opencode-zen/chat_completions | `glm-5.3` | `messages` [] | refused | 400 | `invalid_request_error` | Upstream request failed: [400] Validation: The 'messages' field cannot be empty. At least one message is required. |
| `opencode-zen-chat-string-schema-tool` | opencode-zen/chat_completions | `glm-5.3` | a tool, schema `{"type":"string"}` | accepted (45 reasoning tokens) | 200 |  |  |
| `together-empty-input` | together | `openai/gpt-oss-120b` | `messages` [] | refused | 400 | `invalid_request_error` | messages: Validation error: messages cannot be empty [{"value": Array []}] |
| `together-string-schema-tool` | together | `openai/gpt-oss-120b` | a tool, schema `{"type":"string"}` | accepted (70 reasoning tokens) | 200 |  |  |
