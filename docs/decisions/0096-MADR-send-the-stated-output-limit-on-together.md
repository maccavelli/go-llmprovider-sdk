---
status: accepted
date: 2026-11-27
decision-makers: repository owner
consulted: "the owner's question and decision of 2026-11-27, relayed by the coordinator and quoted below; one live GET of Together's /v1/models on 2026-11-27, allowlisted output only; Together's public chat-completions API reference, read 2026-11-27; 0089-MADR-model-output-limit-discovery.md and docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md; this repository at 69618a4, read only"
informed: "every caller of the Together provider that names no output limit; gobble-cli, whose sessions on Together get the longer default"
---

<!-- markdownlint-disable MD013 MD024 -->

# Send the Model's Stated Output Limit on Together When the Caller Names None

Associated PLAN: [0096-PLAN-send-the-stated-output-limit-on-together.md](0096-PLAN-send-the-stated-output-limit-on-together.md)

Amends [0089-MADR-model-output-limit-discovery.md](0089-MADR-model-output-limit-discovery.md) DL47 for Together only. Every other wire keeps DL47 and DL48 as written.

*(2026-11-27, amendment 1, at the end of this record: the owner answered OQ1 to OQ4. The answer to OQ4 widens this record to Hugging Face and OpenCode's chat route; Ollama and chatcompat keep DL47 and DL48. The first of three measurements was refused with a code the SDK does not read (TG14), so the record stays `proposed`. The text above is unchanged.)*

*(2026-11-27, amendment 2, at the end of this record: the owner answered TG14 with "Re-run all three, streamed (Recommended)". The three streamed requests were each accepted. Every row is decided, so the record is `accepted`.)*

*(2026-11-27, amendment 4, at the end of this record: the owner ruled "Send nothing until the hold is known (Recommended)" on a gap the code review found. Hugging Face sends an SDK-chosen limit only when the cached listing states a live window (HF10). The record stays `accepted`.)*

## Context and Problem Statement

The owner asked, verbatim, on 2026-11-27:

> "does together not provide output token counts per model? research shows qwen4-max supports up to 131072 but it is capped at 32768 by default."

The coordinator proposed that the SDK send the model's stated output limit on Together by default, so a model gets its full output without the caller asking. The owner's decision, verbatim:

> "fold it into v1.35.0"

### What the SDK does today (69618a4)

* Together's wire is an optional-limit wire. 0089-MADR DL47 decided that on such a wire nothing is sent when the caller names no limit, "even where a fact is known", and the service's own default applies. The code: `r.Optional(...)` (E1) sends a caller's value only.
* The SDK already reads Together's stated limit. 0089-MADR DL9 reads `config.max_output_length` as service evidence (E2, the fixture E11); it reaches `catalog.Limits` but only cuts a `WithMaxTokens` value.
* Kilo is the precedent for an optional wire that sends: `LimitInput.Sends` is true for Messages and Kilo (E3, E5), after 0089-MADR DL65's measurement. The fact is held to the room the window leaves the estimated input (E14; 0088-MADR ST87).
* The Chat Completions refusal reader covers Together: OpenAI's "supports at most M" cap form and the split window form (E4, E12; 0089-MADR DL68).

### What Together states (live read, 2026-11-27)

One `GET https://api.together.ai/v1/models` with `TOGETHER_API_KEY` from the environment, by name; status 200; allowlisted output only.

| Measure | Value |
| :--- | :--- |
| entries | 274 |
| chat models (`type` `chat`) | 181 |
| chat models with `config.max_output_length` | 10, each below its `context_length` |
| `Qwen/Qwen4-Max` | `context_length` 1,000,000; no `config.max_output_length` |
| `Qwen/Qwen3.7-Max` | `context_length` 1,000,000; no `config.max_output_length` |

So Together states an output limit per model for 10 of 181 chat models, and not for Qwen4-Max. Under this record, Qwen4-Max gets a limit sent only where another source states one: the models.dev document a caller hands in (`catalog.WithMetadata`, 0093-MADR) or a caller's layer. Whether gobble-cli's document states 131,072 for Together's `Qwen/Qwen4-Max` is not checked here (OQ1).

### The 32,768 default

Not established. Together's API reference says of `max_tokens` only "The maximum number of tokens to generate.", with no default, no maximum, and no behaviour when it is omitted (read 2026-11-27). It documents `context_length_exceeded_behavior`, default `error`, whose `truncate` "overrides max_tokens with the maximum context length of the model". 0088's report found the same: "not documented" (E13). No record in this repository names 32,768 for Together. The 32,768 default and Qwen4-Max's 131,072 are the owner's research, unverified here.

## Decision Drivers

* The owner's decision: a Together model gets its stated output without the caller asking, in v1.35.0.
* 0089-MADR R57: no fixed number decides a limit the caller did not give; evidence does.
* No regression: a request that succeeds today still succeeds (0089-MADR's driver).
* No extra request: a request reads the caches only (0041-MADR D1).

## Considered Options

* A. Together sends the chosen output fact, as Kilo does (`Sends: true`), with the window hold and the existing refusal resend
* B. Keep DL47 on Together; callers state `WithMaxTokens` themselves
* C. Send the window's room (context length minus the estimated input) when nothing states an output limit

## Decision Outcome

Chosen option: "A. Together sends the chosen output fact, as Kilo does", because it is the owner's decision, reuses the rule, hold, resend and memo Kilo already runs (E3, E5, E14), and adds no request. Which evidence counts (OQ1), what happens with none (OQ2) and with a refusal the SDK cannot read (OQ3) are open below, so this record is `proposed`.

### Consequences

* Good, because a Together model with a stated limit gets it on every request with no caller value, and `OutputLimit` reports basis `fact` and its source.
* Good, because a refusal of the sent value in a recorded form is resent once at the stated cap, kept for the process, and later requests send the lower value (0089-MADR DL49, DL73).
* Neutral, because no request is added: the limit is read from the caches the request already reads.
* Bad, because replies may be longer and so cost more output tokens; whether Together holds credit against `max_tokens` is not documented.
* Bad, because a stated value the server refuses in a form the SDK does not read fails a request that succeeds today (OQ3).
* Bad, because DL47 stops being uniform across optional wires: Kilo and Together send, the others do not.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where the row changes behaviour, each with a plant caught on a scratch clone.
* `python3 -B tools/claims.py` on this record passes, each planted false row caught.
* An independent review of the build finds no blocker.

## State Table

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| TG1 | No caller value; Together's listing states `config.max_output_length` | decided *(2026-11-27: the owner's decision)* | that value sent as `max_tokens`, held to the window's room for the estimated input (0088-MADR ST87), basis `fact`, source the listing's | "fold it into v1.35.0" | `TestOutputLimit_Together_0096`, listing case; plant: `Sends` false on Together |
| TG2 | No caller value; only the handed-in models.dev document or a caller's layer states the output (Qwen4-Max's case) | open | OQ1 | | |
| TG3 | No caller value; no source states the output | open | OQ2 | | |
| TG4 | A caller's value | decided | unchanged: `Request.MaxOutputTokens` sent as given; `WithMaxTokens` cut to the stated output, never raised; basis `request` or `option` | 0088-MADR ST15 to ST18; 0089-MADR DL44 | `TestOutputLimit_Together_0096`, caller cases; plant: the option raised to the fact |
| TG5 | The stated value plus the estimated input exceeds `context_length` | decided | held to the room before sending (E14); a split-form refusal resent once at the room it states; the window not kept | 0088-MADR ST87; 0089-MADR DL35 | `TestOutputLimit_Together_0096`, window case; plant: the hold removed |
| TG6 | The sent fact refused in OpenAI's "supports at most M" form, validated | decided | resent once at M, basis `refusal`, outside 0062's bound; M kept in the process memo and reported (`WarningLimitLearned`) unless `WithoutLimitLearning` | 0089-MADR D4, DL49, DL68, DL73 | `TestLimitRefusal_Together_0096`, cap case; plant: the resend dropped |
| TG7 | A kept cap on later requests (does a refusal lower it?) | decided | yes: later requests on the same key send the kept cap; a response with more output than it drops it | 0089-MADR D5, DL38, DL39, DL76 | `TestLimitRefusal_Together_0096`, second request; plant: the memo not read |
| TG8 | The stated value refused in a form the SDK does not read (for example a server maximum below the stated value with new text) | open | OQ3 | | |
| TG9 | Streamed versus plain | decided | the same rule; a stream reads `StreamedOutput`, a plain call `Output`; the memo keyed by each | 0089-MADR DL76 | `TestOutputLimit_Together_0096`, stream case; plant: the stream reads `Output` |
| TG10 | Requests and billing | decided | no request added (cache reads only); the package doc and changelog say replies may be longer and bill more output | 0041-MADR D1 | `TestOutputLimit_Together_0096` counts one POST and no listing GET per request; plant: a listing read on the request |
| TG11 | Existing tests that pin Together sending none with a fact known | decided | `TestOutputLimit_Omitted_0089`'s Together case (E7) and `TestOutputLimit_Facts_0041`'s Together row (E8) change to expect the fact; `TestTogether_RequestShapes`' no-fact case (E9) stays a pin | this record | the changed cases; plant: `Sends` false on Together |
| TG12 | Docs | decided | `together.go`'s package doc (E6), `limit.go`'s comment, `LimitInput.Sends`'s comment (E3), `TestStreamEventsRetry_FirstPayloadError_0088`'s comment (E10), a **Changed** entry under `[Unreleased]` for v1.35.0; released changelog sections are not edited (0094-MADR) | D9 of 0089-MADR; 0094-MADR | `tools/claims.py` rows on the new sentences, planted false; `make changelog-check` |
| TG13 | The other optional wires | out of scope | OQ4 asks whether a later record should apply the same rule; nothing changes on them here | the owner's decision names Together | |

Counts: 9 decided, 1 out of scope (TG13), 3 open (TG2, TG3, TG8); 13 rows.

## Owner Questions

Three options each, the recommended first.

| Question | Options (recommended first) | Cost of each | Rows |
| :--- | :--- | :--- | :--- |
| OQ1. Which sources count as "stated" on Together? | (1) **Any evidence 0089 ranks, as on Kilo**: Together's listing, a caller's layer, or the handed-in models.dev document. (2) **Together's listing and a caller's layer only**: the document is ignored for sending. (3) **Together's listing only.** | (1) fixes Qwen4-Max wherever the handed-in document states its limit; a wrong document value (registry errors such as output equal to the window, 0089 W12) is sent, held to the window's room. (2) Qwen4-Max is fixed only by a caller's layer (gobble-cli's models.json `maxTokens`); 10 of 181 chat models get a limit otherwise. (3) 10 of 181 chat models only; Qwen4-Max stays at the server's default. | TG2 |
| OQ2. Nothing states the output. | (1) **Send none**, as today; the server's default applies. (2) **Send the window's room** (`context_length` minus the estimated input), labelled `fallback`. (3) **A fixed fallback number**, labelled `fallback`. | (1) no change for 171 of 181 chat models unless OQ1 (1) or (2) supplies a fact. (2) near-window values (Qwen4-Max: about 1,000,000) that a server with a smaller output cap may refuse in an unread form; needs OQ3's answer. (3) a number with no source, against R57. | TG3 |
| OQ3. The sent fact is refused in a form the SDK does not read. | (1) **Measure before the build**: one live Together request on `Qwen/Qwen4-Max` (or the model OQ1 makes reachable) with `max_tokens` at the stated value and a one-word prompt; a refusal's text is recorded and, if new, added as a form on the Chat Completions wire; an acceptance confirms the value. (2) **Build without measuring**: such a refusal is returned as an error, as for any unread refusal. (3) **Resend once with no limit** on a 400 that names `max_tokens` after an SDK-chosen value, so the request becomes today's. | (1) one short billed request before the build, allowlisted output; delays the build by that run. (2) a request that succeeds today can fail until a later record reads the form. (3) one extra request on such refusals; matching by a word in the error text is a guess the validator (0089 D4) exists to avoid. | TG8 |
| OQ4. Should the same rule apply to Hugging Face, OpenCode's chat route, Ollama or chatcompat? (Kilo already sends, 0089 DL65.) | (1) **Not in this record**: each wire by its own record, on its own evidence. (2) **Also Hugging Face and OpenCode's chat route** in v1.35.0: hosted Chat Completions gateways whose facts come from the registry. (3) **Every optional wire.** | (1) none now; the question returns per wire. (2) widens this record; their limits come from the registry only, with the same OQ3 risk and no measurement. (3) reverses 0089 DL47 and DL48 entirely, including Ollama's unbounded default and caller servers (DL69). | TG13 |

## Pros and Cons of the Options

### A. Send the chosen output fact

* Good, because it is the owner's decision and reuses Kilo's code path.
* Bad, because it adds a refusal path to requests that send none today.

### B. Keep DL47

* Good, because nothing changes and no request can newly fail.
* Bad, because it contradicts the owner's decision; a caller must name each model's limit.

### C. The window's room

* Good, because every Together model states `context_length`.
* Bad, because the window is not an output limit; it is the number 0089 W3 warns against reading as one.

## More Information

### Relation to other records

* 0089-MADR-model-output-limit-discovery.md: DL47 amended for Together; DL9, DL35, DL44, DL49, DL68, DL73 and DL76 apply unchanged.
* 0088-MADR-default-output-limit-from-model-facts.md: ST15 to ST18 and ST87 apply unchanged.
* 0095-MADR-clear-the-sdk-follow-ups.md: in progress in the same release; no row overlaps.

### What could not be established

* Together's default output when `max_tokens` is omitted, for any model.
* Qwen4-Max's own output limit on Together.
* The text Together sends when `max_tokens` exceeds a model's accepted maximum.

### Evidence index

<!-- claims: self=69618a4 -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1 | Together sends a caller's value only | `llmprovider/providers/together/limit.go:74` | `return r.Optional(req.MaxOutputTokens, p.maxTokens, p.maxGiven, func() (int, string) {` |
| E2 | Together's stated output is read | `llmprovider/catalog/listing_metadata.go:87` | `MaxOutputLength *int` |
| E3 | the routes that send a chosen limit | `llmprovider/internal/wire/limit_rule.go:26` | `// Messages wire, and Kilo (0089-MADR DL65, measured 2026-11-24; OQ1 (1)).` |
| E4 | the Chat Completions refusal forms | `llmprovider/internal/wire/limit_refusal.go:73` | `ChatCompletionsLimits = LimitWire{Forms: FormOpenAICap` |
| E5 | Kilo sends the chosen fact | `llmprovider/providers/kilo/kilo.go:552` | `Given: p.maxGiven, Sends: true,` |
| E6 | Together's package doc | `llmprovider/providers/together/together.go:79` | `// The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| E7 | the Omitted test's Together case | `llmprovider/providers/output_limit_0089_test.go:176` | `{"together", wirecase.ChatReply, byID(llmprovider.ProviderTogether), "openai/gpt-oss-120b"},` |
| E8 | the Facts test's Together row | `llmprovider/providers/facts_0041_test.go:84` | `{llmprovider.ProviderTogether, "openai/gpt-oss-120b", false},` |
| E9 | Together's own no-fact pin | `llmprovider/providers/together/together_test.go:85` | `if limit, sent := body["max_tokens"]; sent` |
| E10 | a comment that becomes false | `llmprovider/providers/default_limit_0088_test.go:257` | `// only (Together sends no limit the SDK chose, 0089-MADR DL47): a Stream` |
| E11 | the fixture's stated output | `llmprovider/catalog/testdata/ranking-2026-10-12/together.json:1` | `"max_output_length":40000` |
| E12 | OpenAI's cap form | `llmprovider/internal/wire/limit_refusal.go:118` | `supports at most (\d+) completion tokens, whereas you provided (\d+)` |
| E13 | Together documents no default | `docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md:468` | `not in its spec; 10 of 265 listing entries carry` |
| E14 | the fact is held to the room | `llmprovider/internal/wire/limit_rule.go:69` | `case in.Output > 0:` |

## Amendment 1 (2026-11-27): the owner's answers, two more wires, and the first measurement

Additions only; the sections above are unchanged.

### Owner rulings (2026-11-27)

Each answer is the owner's, relayed by the coordinator, its label quoted as given.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| OQ1 | "Anything the SDK ranks (Recommended)" | (1): any evidence 0089-MADR D1 ranks (a caller's layer, the service's listing, the handed-in models.dev document), chosen by rank, as on Kilo | TG2, HF1, OC1 |
| OQ2 | "Send nothing, as today (Recommended)" | (1): with no source stating the output, nothing is sent and the service's default applies | TG3, HF2, OC2 |
| OQ3 | "Measure first (Recommended)" | (1): one short live request per wire with `max_tokens` at the value the SDK would send, before the build | TG8, HF6, OC6 |
| OQ4 | "Also HF and OpenCode chat" | (2): Hugging Face and OpenCode's chat route (Zen and Go) join in v1.35.0; Ollama and chatcompat keep DL47 and DL48, as do OpenCode's responses and google routes, Gemini, Grok and OpenAI with a key | TG13 |

### What the two added wires state and do (69618a4)

* **Hugging Face.** The wire sends a caller's value only (E15, the same `r.Optional` as Together) and reads OpenAI's cap form and the split window form (E16; 0089-MADR DL68). Its listing states no output limit. A live read on 2026-11-27 (`GET https://router.huggingface.co/v1/models` with `HF_TOKEN`, status 200, field names and counts only) found 106 entries and 454 provider offerings. The entry fields are `architecture`, `created`, `id`, `object`, `owned_by` and `providers`. The offering fields are `context_length`, `first_token_latency_ms`, `is_free`, `is_model_author`, `pricing`, `provider`, `provider_name`, `quantization`, `status`, `supports_structured_output`, `supports_tools`, `throughput` and `uptime_last_day_percent`. The window is the suffixed provider's `context_length`, else the smallest among live providers (0089-MADR DL10). So the output comes from a caller's layer or the handed-in document only.
* **OpenCode's chat route.** Every route but messages sends a caller's value only (E17, E18). The chat route reads OpenAI's cap form and the split window form (E19). The listing carries ids only (0089-MADR DL11), so the output and the window come from a caller's layer or the handed-in document. The messages route already sends the chosen fact (E20). Both gateways have a chat route. On OpenCode Go, `qwen4-max` takes it by the `-max` rule (`internal/opencoderoute`, 0020-MADR F41).
* **The models.dev document** (one public read of `https://models.dev/api.json`, 2026-11-27, limits only): `togetherai` 27 of 27 models state `limit.output`, `huggingface` 125 of 127, `opencode` 147 of 147, `opencode-go` 44 of 44. Together's `Qwen/Qwen4-Max` states output 262,144 and context 1,000,000; `Qwen/Qwen3.7-Max` states 131,072 and 1,000,000; OpenCode Go's `qwen4-max` states 262,144 and 1,000,000. The owner's 131,072 for Qwen4-Max is not the document's value for that id.

### The measurements (OQ3 (1))

Each was planned as one request, `{"role":"user","content":"Reply with one word: ok"}` (7 tokens by `EstimateInput`), with `max_tokens` at the value the SDK would send: the document's output held to the window and to the room 0088-MADR ST87 leaves (window minus the estimate minus 4,096). The key was read by name only. Output was allowlisted: the status, the error's `type` and `code`, whether its message matches each of the SDK's seven forms, whether it names `max_tokens`, and on success the finish reason and token counts.

| Wire | Model | `max_tokens` | Request | Status | Error `type` / `code` | Forms matched | Accepted | Output tokens |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Together (`TOGETHER_API_KEY`) | `Qwen/Qwen4-Max` | 262,144 (the document's; window 1,000,000) | plain (`stream` false) | 400 | `invalid_request_error` / `streaming_required` | none of seven; the message names no `max_tokens` and no number | no | none |
| Hugging Face (`HF_TOKEN`) | `deepseek-ai/DeepSeek-V4.1-Flash` | 258,041 planned (the document's 384,000, held to the smallest live window 262,144, then the room) | not run | | | | | |
| OpenCode Zen chat (`OPENCODE_API_KEY`) | `glm-5.3` | 131,072 planned (the document's; window 1,000,000) | not run | | | | | |

The brief said to stop at a refusal in a form the SDK does not read, so the Hugging Face and OpenCode requests were not sent. The Together request was sent plain, but a plain `Generate` whose limit the SDK chose is read over the stream (E21; 0088-MADR ST88). So this request is not the one the build would send. It also does not show whether `streaming_required` depends on the size of `max_tokens` or whether the model always requires a stream: there was one request and no control.

### Rows added and decided (amendment 1)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| TG2 | No caller value; only the handed-in document or a caller's layer states the output | decided *(2026-11-27: the owner's answer)* | that value sent, chosen by 0089 D1's rank, held to the window and the room, basis `fact`, source its evidence's | OQ1 "Anything the SDK ranks (Recommended)" | `TestOutputLimit_Together_0096`, document case; plant: registry evidence filtered out |
| TG3 | No caller value; no source states the output | decided *(2026-11-27: the owner's answer)* | nothing sent, basis `none`; the service's default applies | OQ2 "Send nothing, as today (Recommended)" | `TestTogether_RequestShapes` (pin) and `TestOutputLimit_Together_0096`, none case; plant: the window's room sent |
| TG8 | A refusal in a form the SDK does not read | decided *(2026-11-27: the owner's answer)*, its measurement open as TG14 | measured before the build, one request per wire; a new form is recorded and put to the owner | OQ3 "Measure first (Recommended)" | Phase 1 of the PLAN |
| TG13 | The other optional wires (was out of scope) | decided *(2026-11-27: the owner's answer)* | Hugging Face and OpenCode's chat route follow this record (rows HF and OC); Ollama, chatcompat, OpenCode's responses and google routes, Gemini, Grok and OpenAI with a key keep DL47 and DL48 | OQ4 "Also HF and OpenCode chat" | `TestOutputLimit_Omitted_0089`'s remaining optional cases stay pins; plant: `Sends` true on Ollama |
| TG14 | Together, `Qwen/Qwen4-Max`, a plain request with `max_tokens` 262,144: 400 `invalid_request_error` / `streaming_required` | open | the three options below | | |
| HF1 | Hugging Face, no caller value, the document or a caller's layer states the output (its listing never does) | decided | sent, by rank, held to the window (DL10) and the room, basis `fact` | OQ1 "Anything the SDK ranks (Recommended)" | `TestOutputLimit_HuggingFace_0096`, document case; plant: `Sends` false on Hugging Face |
| HF2 | Hugging Face, nothing states the output | decided | nothing sent | OQ2 "Send nothing, as today (Recommended)" | same test, none case; plant: the window's room sent |
| HF3 | Hugging Face, a caller's value | decided | unchanged (0088 ST15 to ST18) | 0089 DL44 | same test, caller cases; plant: the option raised |
| HF4 | Hugging Face, live providers stating different windows | decided | held to the suffixed provider's window, else the smallest live one's, then the room | 0089 DL10; 0088 ST87 | same test, window case (384,000 stated, windows 1,048,576 and 262,144: 262,144 less the estimate and 4,096); plant: the largest window read |
| HF5 | Hugging Face, the sent fact refused in OpenAI's cap form | decided | resent once at M, M kept and reported; later requests send M | 0089 D4, D5, DL49, DL68, DL73 | `TestLimitRefusal_HuggingFace_0096`; plants: the resend dropped; the memo not read |
| HF6 | Hugging Face, a refusal in an unread form | open: awaits its measurement | as TG14's answer decides how to measure | OQ3 "Measure first (Recommended)" | Phase 1 |
| HF7 | Hugging Face, streamed versus plain; requests and billing | decided | as TG9 and TG10 | 0089 DL76; 0041 D1 | `TestOutputLimit_HuggingFace_0096`, stream case and request count; plants as TG9, TG10 |
| HF8 | Hugging Face, existing tests | decided | `TestOutputLimit_Omitted_0089`'s case (E22) and `TestOutputLimit_Facts_0041`'s row (E23) change to expect the fact | this record | the changed cases; plant: `Sends` false on Hugging Face |
| HF9 | Hugging Face, docs | decided | its package doc (E24) and `limit.go`'s comment | 0089 D9 | claims rows, planted false |
| OC1 | OpenCode chat route (Zen and Go), no caller value, the document or a caller's layer states the output (the listing never does) | decided | sent, by rank, held to the window and the room, basis `fact`; no fallback and no refusal before sending: the route is not required (0089 DL51 and DL77 stay on messages) | OQ1 "Anything the SDK ranks (Recommended)" | `TestOutputLimit_OpenCodeChat_0096`, document case on each gateway; plant: the chat route left on `Optional` |
| OC2 | OpenCode chat route, nothing states the output | decided | nothing sent | OQ2 "Send nothing, as today (Recommended)" | same test, none case; plant: the messages route's fallback sent on chat |
| OC3 | OpenCode chat route, a caller's value | decided | unchanged | 0089 DL44 | same test, caller cases; plant: the option raised |
| OC4 | OpenCode chat route, the window | decided | the document's or a layer's window, else none (the fact sent unheld) | 0088 ST87 | same test, window case; plant: the hold removed |
| OC5 | OpenCode chat route, the sent fact refused in OpenAI's cap form | decided | resent once at M, kept, later requests send M | 0089 D4, D5, DL68 | `TestLimitRefusal_OpenCodeChat_0096`; plants: the resend dropped; the memo not read |
| OC6 | OpenCode chat route, a refusal in an unread form | open: awaits its measurement | as TG14's answer decides how to measure | OQ3 "Measure first (Recommended)" | Phase 1 |
| OC7 | OpenCode chat route, streamed versus plain; requests and billing | decided | as TG9 and TG10 | 0089 DL76; 0041 D1 | `TestOutputLimit_OpenCodeChat_0096`, stream case and request count; plants as TG9, TG10 |
| OC8 | OpenCode, existing tests | decided | `TestOutputLimit_Omitted_0089`'s chat case (E25) and `TestOutputLimit_Facts_0041`'s `glm-5.3` row (E26) change to expect the fact; its responses and google cases stay pins | this record | the changed cases; plant: the chat route left on `Optional` |
| OC9 | OpenCode, docs | decided | the package doc's sentence that the chat route sends none (E27) | 0089 D9 | claims rows, planted false |

### TG14: three options, recommended first

| Option | What happens | Cost |
| :--- | :--- | :--- |
| (1) **Measure again, streamed, as the SDK sends it** | one streamed request per wire (Together `Qwen/Qwen4-Max` at 262,144; Hugging Face and OpenCode as planned), same allowlist; the build waits for them | three short billed requests; Together is measured a second time, which needs the owner's word |
| (2) **Build on the over-stream rule, Together not remeasured** | `streaming_required` is recorded as reached only by a plain request, which the SDK sends only with a caller's value (unchanged); Hugging Face and OpenCode measured streamed | Together's acceptance of 262,144 on a stream stays unmeasured; an unread refusal there fails a request that succeeds today |
| (3) **Read `streaming_required` as a refusal and resend over the stream** | a plain request carrying a caller's value that draws this code is resent once as a stream | widens scope to the caller's path, which no ruling touched; a new rule on one observation |

Counts (amendment 1): TG1 to TG14, HF1 to HF9 and OC1 to OC9 are 32 rows: 29 decided, 1 open (TG14), 2 awaiting measurement (HF6, OC6), none out of scope.

### Evidence added (amendment 1)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E15 | Hugging Face sends a caller's value only | `llmprovider/providers/huggingface/limit.go:74` | `return r.Optional(req.MaxOutputTokens, p.maxTokens, p.maxGiven, func() (int, string) {` |
| E16 | Hugging Face reads the Chat Completions forms | `llmprovider/providers/huggingface/limit.go:54` | `r := &wire.LimitRetry{Wire: wire.ChatCompletionsLimits, Model: model,` |
| E17 | OpenCode: every route but messages is optional | `llmprovider/providers/opencode/opencode.go:599` | `if c.route != RouteMessages {` |
| E18 | OpenCode's optional routes send a caller's value only | `llmprovider/providers/opencode/opencode.go:600` | `optional = c.limit.Optional(c.req.MaxOutputTokens, p.maxTokens, p.maxGiven, func() (int, string) {` |
| E19 | OpenCode's chat route reads the Chat Completions forms | `llmprovider/providers/opencode/opencode.go:1000` | `return wire.ChatCompletionsLimits` |
| E20 | OpenCode's messages route sends | `llmprovider/providers/opencode/opencode.go:1043` | `Given: p.maxGiven, Sends: true,` |
| E21 | a plain Generate the SDK chose the limit for is streamed | `llmprovider/internal/wire/overstream.go:20` | `func OverStream(req *llmprovider.Request, given bool) bool {` |
| E22 | the Omitted test's Hugging Face case | `llmprovider/providers/output_limit_0089_test.go:177` | `{"huggingface", wirecase.ChatReply, byID(llmprovider.ProviderHuggingFace), "deepseek-ai/DeepSeek-V4.1-Flash"},` |
| E23 | the Facts test's Hugging Face row | `llmprovider/providers/facts_0041_test.go:83` | `{llmprovider.ProviderHuggingFace, "deepseek-ai/DeepSeek-V4.1-Flash", false},` |
| E24 | Hugging Face's package doc | `llmprovider/providers/huggingface/huggingface.go:81` | `// The output limit is optional on this wire (0089-MADR D6, DL47, DL48): a` |
| E25 | the Omitted test's OpenCode chat case | `llmprovider/providers/output_limit_0089_test.go:182` | `{"opencode-zen chat", wirecase.ChatReply, zenOn(opencode.RouteChatCompletions), "glm-5.3"},` |
| E26 | the Facts test's OpenCode chat row | `llmprovider/providers/facts_0041_test.go:80` | `{llmprovider.ProviderOpencodeZen, "glm-5.3", false},` |
| E27 | OpenCode's sentence on the optional routes | `llmprovider/providers/opencode/opencode.go:138` | `//     (DL47). The Response, and a stream's done event, carry the limit sent` |

## Amendment 2 (2026-11-27): TG14 answered, three streamed measurements

Additions only; the sections above are unchanged.

### Owner ruling (2026-11-27)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| TG14 | "Re-run all three, streamed (Recommended)" | (1): one streamed request per wire, the shape the SDK sends when it chooses the limit, same allowlist | TG14; with the results, TG8, HF6, OC6 |

### The request shape

A plain `Generate` whose limit the SDK chose is read over the stream (E21). Each wire's stream body is the same (E28 to E30), with no reasoning or tools asked for: `{"model", "messages": [{"role": "user", "content": "Reply with one word: ok"}], "max_tokens", "stream": true, "stream_options": {"include_usage": true}}`, with `Accept: text/event-stream` and the key as a bearer. Each `max_tokens` is the value the SDK would send: the document's output, held to the window and to the room 0088-MADR ST87 leaves (E14). Each wire got one attempt, with no retry, and the key was read by name only. The output was allowlisted: the status, any error's `type` and `code`, the kinds of events seen and their counts, the finish reason, and the usage counts.

### The measurements (streamed)

| Wire | Model | `max_tokens` | Status | Error | Limit accepted | Event kinds seen | Finish | Output tokens (reasoning) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Together (`TOGETHER_API_KEY`) | `Qwen/Qwen4-Max` | 262,144 | 200 | none | yes | role 1, reasoning 4, content 1, finish 1, a chunk with no choices 1, usage 1, `[DONE]` | `stop` | 14 (11) |
| Hugging Face (`HF_TOKEN`) | `deepseek-ai/DeepSeek-V4.1-Flash` | 258,041 (384,000 stated, held to the smallest live window 262,144, then the room) | 200; `x-inference-provider` `openrouter` | none | yes | role 11, reasoning 8, content 1, finish 2, usage 1, `[DONE]` | `stop` | 29 (26) |
| OpenCode Zen chat (`OPENCODE_API_KEY`) | `glm-5.3` | 131,072 | 200 | none | yes | role 1, `reasoning_content` 10, content 1, finish 1, usage 1, `[DONE]` | `stop` | 34 (31) |

What this shows: each service accepted the limit the SDK would send, on a stream, with no refusal and no in-stream error. It does not show what each service does at a value above its own maximum. A stop at a short reply says nothing about the cap itself. A refusal in a form the SDK does not read is returned as an error, as on every wire today (0089-MADR D4). Hugging Face's router served the request through `openrouter`, a provider not in the model's listing entry. The plain request's `streaming_required` (amendment 1) applies only to a plain request, which the SDK sends only with a caller's own value, so that path is unchanged.

### Rows decided (amendment 2)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| TG14 | Together's `streaming_required` on a plain request | decided *(2026-11-27: the owner's answer and the measurement)* | not reached by an SDK-chosen limit, which is streamed (E21); a caller's own value on a plain request is unchanged; no new form | "Re-run all three, streamed (Recommended)" | `TestOutputLimit_Together_0096`'s plain-Generate case: an SDK-chosen limit goes over the stream; plant: `OverStream` bypassed on Together |
| TG8 | A refusal in a form the SDK does not read (the measurement) | decided *(2026-11-27: measured)* | Together accepted 262,144 streamed; no new form; an unread refusal is returned as today | OQ3 "Measure first (Recommended)"; this measurement | Phase 1's record; the unit tests as amendment 1 names them |
| HF6 | Hugging Face, a refusal in an unread form | decided *(2026-11-27: measured)* | 258,041 accepted streamed; no new form; an unread refusal is returned as today | as TG8 | as TG8 |
| OC6 | OpenCode chat route, a refusal in an unread form | decided *(2026-11-27: measured)* | 131,072 accepted streamed on Zen; no new form; Go's chat route reads the same forms (E19) and was not measured | as TG8 | as TG8 |

Counts (amendment 2): 32 rows, 32 decided, 0 open, 0 awaiting. The record is `accepted`.

### Evidence added (amendment 2)

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E28 | Together's stream body | `llmprovider/providers/together/together.go:298` | `body["stream_options"] = map[string]any{"include_usage": true}` |
| E29 | Hugging Face's stream body | `llmprovider/providers/huggingface/huggingface.go:296` | `body["stream_options"] = map[string]any{"include_usage": true}` |
| E30 | OpenCode's chat-route stream body | `llmprovider/providers/opencode/opencode.go:636` | `body["stream_options"] = map[string]any{"include_usage": true}` |

## Amendment 3 (2026-11-27): OpenCode's chat goldens

Additions only; OC8 above is unchanged and this row adds to it. Phase 2's first pass stopped because 24 OpenCode chat goldens changed, and no row named them (0096-PLAN, "Execution Record, Phase 2"). The coordinator decided it as a test-artifact matter that follows from OC1: "Take your option (2)".

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| OC8a | OpenCode's chat-route G-wire goldens, whose harness hands in the extract fixture (`extractDoc()`, which states 131,072 for `glm-5.3` and `kimi-k3`) | decided *(2026-11-27: the coordinator's call)* | the 12 goldens in `testdata/wire/opencode-zen-chat/` and the 12 in `opencode-go-chat/` are regenerated with `-update`, each changed only by `requests[0].body.max_tokens` 131,072 added (`tools/goldendiff.py --rev 08d3b77`); a new golden, `testdata/wire/opencode-zen-chat-nodoc/text.json`, pins OC2 at the wire: Zen's chat route with no document sends no `max_tokens` | "Take your option (2)" | `TestWireGoldens`, `TestWireGolden_ChatNoDocument_0096`; plant: the messages route's fallback rule on chat (`Required: true`), which the new golden catches |

Counts (amendment 3): 33 rows, 33 decided, 0 open.

## Amendment 4 (2026-11-27): Hugging Face sends only a held limit

Additions only; HF1, HF8, TG2 and TG12 above are unchanged, and the rows below add to them. The code review of Phase 2's first pass came back ready with notes. One note was a gap: on Hugging Face in a cold process, where the listing is not cached, `limitInput` holds the fact only to the document's `context`. So the models.dev output goes out without the live providers' window hold that HF4 decides, for example 384,000 for `deepseek-ai/DeepSeek-V4.1-Flash`, whose smallest live window is 262,144. The owner ruled on it, relayed by the coordinator.

### Owner ruling (2026-11-27)

| Question | The owner's answer | Rows it decides |
| :--- | :--- | :--- |
| The Hugging Face cold-process gap | "Send nothing until the hold is known (Recommended)" | HF10, HF8a |

The rule: on Hugging Face, an SDK-chosen limit is sent only when the cached listing states a live provider's window, so that HF4's hold applies. The window fact must carry `listing:huggingface` evidence. Otherwise none is sent, as before this record, and no request is added to learn the window. A caller's value is unchanged (HF3): `WithMaxTokens` is still cut to the stated output.

### Rows added and decided (amendment 4)

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| HF10 | Hugging Face, no caller value, an output stated (a caller's layer or the document) but the listing not cached, so no live provider's window is known (a cold process) | decided *(2026-11-27: the owner's ruling)* | nothing sent, basis `none`; no request added; once the listing is cached, HF1 and HF4 apply (held to 258,041 in the measured case) | "Send nothing until the hold is known (Recommended)" | `TestOutputLimit_HuggingFaceCold_0096`, red at pass 1's tree; `TestOutputLimit_NoRequestAdded_0096`'s cold case (one POST, no GET); plant: the unheld send (`Sends: true` whatever the window's evidence) |
| HF8a | Hugging Face, the existing tests that hand in no listing | decided *(2026-11-27: follows from HF10)* | `TestOutputLimit_Omitted_0089`'s and `TestOutputLimit_Facts_0041`'s Hugging Face cases, which hand in no listing, expect none again, as at the base; the 0096 tests' Hugging Face send cases (`_Sends_0096`, `_Streamed_0096`, `_NoRequestAdded_0096`, `_Resend_0096`, `_Kept_0096`) cache a listing stating a live window of 1,000,000 first, so HF1, HF5 and HF7 stay tested; `_NoRequestAdded_0096` counts the requests after that listing read, and once more in a cold process | "Send nothing until the hold is known (Recommended)" | the changed cases; plant: the unheld send, caught by `_HuggingFaceCold_0096`, `TestOutputLimit_Omitted_0089` and `TestOutputLimit_Facts_0041` |
| TG2a | Together, the output stated by the models.dev document handed to `New` with `catalog.WithMetadata` (not a layer) | decided *(2026-11-27: the review's note, the coordinator's call)* | sent, source `models.dev`, as TG2 | TG2 | `TestOutputLimit_Sends_0096`, a `WithMetadata` case; plant: registry evidence filtered out |
| TG12a | Docs the review found false or stale | decided *(2026-11-27: the review's notes, the coordinator's call)* | the changelog entry names the three measurements (Together, Hugging Face, OpenCode Zen's chat route; Go's chat route was not measured, OC6) and states HF10; `TestGenerateOverStream_ErrorIn200_0088`'s comment reads "on Together, with nothing stated and so nothing sent"; `TestOutputLimit_Sends_0096`'s comment names both a layer and the document; Hugging Face's package doc, `limit.go`'s comment, `LimitInput.Sends`, the README and the standards guide state HF10 | 0089 D9 | claims rows on the new sentences, planted false |

Counts (amendment 4): 37 rows, 37 decided, 0 open.
