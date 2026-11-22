---
date: 2026-11-22
subject: how four reference agents, six services, two metadata catalogues and the common client libraries choose a request's output limit, for 0088-MADR's amendment of 2026-11-22 and gobble-cli's 0065-MADR
examines: "the read-only checkouts kilocode at 58072bd6, opencode at 4ce137b3, codex at 57e64682 and grok-build at 2bdd1d6a; this repository at 18e8e72; gobble-cli at 0589454; vendor and catalogue pages read on 2026-11-22, the ones a decision rests on fetched again by the synthesis step itself"
---

<!-- markdownlint-disable MD013 MD024 -->

# Output Limits Across Agents and Vendors, Recorded for 0088

This report records findings. It decides nothing. The decisions it bears on
belong to
[0088-MADR-default-output-limit-from-model-facts.md](../decisions/0088-MADR-default-output-limit-from-model-facts.md),
whose amendment of 2026-11-22 reads them, and to gobble-cli's
`docs/decisions/0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`,
which states what gobble owns and what it leaves to this SDK.

## How it was gathered

* **Three retrieval passes** (2026-11-22): one over kilocode and opencode,
  one over codex and grok-build, one over vendor and catalogue pages. Each
  returned facts with sources and no design. The coordinator re-read the key
  code claims of the first two against the checkouts.
* **The synthesis step** re-read every code line this report cites at the
  revisions above (the evidence index below is checked by
  `tools/claims.py`), and fetched again each vendor page a decision rests
  on. A web page was read through a fetch tool that relays the page through
  a small model; a quote below is marked **verified** when the synthesis
  step's own fetch returned the same sentence, **relayed** when only the
  retrieval pass saw it, and **secondary** when the source is not the
  vendor's own.
* Nothing was run: no binary, no test of an upstream tree, no live call.

## The sources

### kilocode (Kilo's own client; an older opencode engine with Kilo changes)

What it does:

* The default is the model's catalogue limit **capped** at 32,000, or 32,000
  when the catalogue says 0 (E1, E2). An environment variable,
  `KILO_EXPERIMENTAL_OUTPUT_TOKEN_MAX`, moves the ceiling.
* For Claude on first-party routes, with no explicit thinking budget, it asks
  for the model's **whole** limit, because "the shared 32k default can be
  used up by thinking before any text or tool call is written" (E3; 0088
  C61, C62).
* It fits the limit to the window the prompt leaves: room = window − max(the
  provider's reported count, its own estimate × 1.3) − 2,048, floor 1,024;
  when the input alone overflows it sends the unreduced value so the service
  refuses and compaction runs (E4; 0088 C63).
* Its gateway listing, when an entry has no output cap, invents one: 20 % of
  the window (E7).
* On a `length` finish it **warns only**, with a separate sentence for an
  answer cut while reasoning: "The model hit its output limit while reasoning
  and produced no actionable output. Try disabling reasoning or increasing
  the output limit." (E5, E6). No continuation.
* Small calls (titles, branch names, commit messages) take min(limit,
  32,000); compaction sizes its chunks for a 2,048-token reply.

Evaluation: primary source, current, and the client of the gateway the
reported failure came through. It shows that a shipping agent takes the
default from the catalogue, caps it, fits it to the window, and treats Claude's
thinking as the case that needs the whole limit. It does not show why 32,000:
no rationale text exists in the tree (R1 §7). Its 20 %-of-window invention
for gateway entries is a heuristic with no stated source, and is not what this
SDK's Kilo listing does (0088 C33). Its docs quote a Claude window refusal,
"input length and max tokens exceed context limit" (E8), without saying which
service sends it.

### opencode (the new engine)

What it does:

* The default **is** the catalogue limit, with 32,000 only when the catalogue
  has none, and never above 256,000: "No reply needs more, however much the
  model allows" (E9 to E11).
* It fits that value to the window on every primary turn: room = ⌊window ×
  0.99⌋ − the provider-measured input − ⌈1.15 × the estimate of unmeasured
  input⌉, floor 1,024 (0088 C70; R1 §1).
* The limit is set per call kind: primary turns and compaction only; titles
  and other generation calls send none, "because their reasoning is hard to
  budget" (E12), so the wire's own default applies (32,000 on Anthropic's,
  which requires one).
* An explicit thinking budget is held to half the output limit on every wire
  that takes a budget (E13).
* On a `length` finish it **continues automatically** with a synthetic user
  message, "Your last response hit the output token limit (stop reason:
  length). Do not apologize, recap, or repeat yourself. Break the remaining
  work into smaller pieces." (E14), and fails after three `length` turns in a
  row (E15). The limit is not raised on the retry. A cut tool call fails with
  the reason "length", not "malformed".
* The catalogue is models.dev, refreshed every five minutes over a bundled
  snapshot; a user's `limit` merges over it field by field.

Evaluation: primary source, current, the most complete design of the four.
It shows the catalogue limit as the default (the choice 0088 made), a dynamic
window fit from measured plus estimated input, per-call-kind profiles,
reasoning held below the limit, and continuation. It does not show measured
outcomes: the fitting commit has no body (R1 §1), and the 256,000 ceiling's
only rationale is the comment quoted.

### codex

What it does: its Responses request has **no output-limit field** (E16); a
commit that read `max_output_tokens` from configuration was reverted the same
day with no reason given (R2 A1). The only output headroom is a flat 5 % of
the window kept from inputs (E17). An incomplete answer, `max_output_tokens`
included, is a non-retryable error that fails the turn (R2 A5).

Evaluation: primary source. It shows that a first-party client on a
first-party service can leave the limit to the service. It does not
generalise: it speaks one wire to one vendor, the vendor's own default is not
documented on the pages read (Gaps), and Messages requires the field.

### grok-build (xAI's Grok CLI)

What it does:

* The **agent turn sends a limit**: `max_output_tokens` is the model's
  `max_completion_tokens` from its configuration (E18), which comes from the
  server's model catalogue, a per-model user setting, or a global fallback
  that never clamps (E19). This corrects 0088's line "grok-build: an agent
  turn sends no output limit" (0088 C73, C74), which read only the side
  tasks.
* Side tasks set small fixed limits: a session summary 100 (E21), image
  description 4,096, web search 8,192, prompt suggestions 64 plus a 256-token
  reasoning reserve.
* A response header, `x-grok-max-completion-tokens`, updates a model's limit
  at run time, up or down (E22, E23).
* `LengthPolicy`'s default runs a cut answer's tool calls when all their
  arguments are complete, and fails a cut text answer: "`Length` is usually
  context exhaustion (the output budget is the window minus the prompt), so
  retrying cannot succeed." (E20). A continuation ("salvage") with a reminder
  message exists, off unless a remote budget or an environment variable turns
  it on, with a cap of 2 by default (E24).

Evaluation: primary source, a monorepo sync of 2026-09-29. It shows the
model's limit taken from server metadata, a server able to push a new limit,
and salvage of complete tool calls. The header is its own proxy's; whether
xAI's public API sends it was not established.

### Anthropic (Messages)

* `max_tokens` is required; different models take different maxima (R3 §2,
  relayed). Current models with a 1M window can generate up to 128k per
  request (WS4, verified).
* "Thinking tokens are a subset of your `max_tokens` parameter, are billed as
  output tokens, and count toward rate limits." (WS4, verified).
* **Rate limits.** "OTPM rate limits are evaluated in real time as output
  tokens are produced, counting only the actual tokens generated. The
  `max_tokens` parameter does not factor into OTPM rate limit calculations, so
  there is no rate limit downside to setting a higher `max_tokens` value."
  (WS1, verified). This contradicts 0088's statement that Anthropic counts
  `max_tokens` at a request's start (0088, "What a raised limit costs").
* **The window.** "If the input alone already exceeds the model's context
  window, the API returns a 400 `invalid_request_error` ("prompt is too
  long") on every model. On Claude 4.5 models and newer, if input tokens plus
  `max_tokens` exceeds the context window size, the API accepts the request.
  If generation then reaches the context window limit, it stops with
  `stop_reason: "model_context_window_exceeded"`. On earlier models, the API
  returns a validation error instead." A beta header,
  `model-context-window-exceeded-2025-08-26`, opts earlier models in (WS4,
  verified). The page does not quote the earlier models' error text, and
  neither does the stop-reasons page, which says "On Sonnet 4.5 (deprecated)
  and newer models, the API returns this value without a beta header. For
  earlier models, add the `model-context-window-exceeded-2025-08-26` beta
  header to enable it." (WS6, verified; the page was read to 100,000 of its
  125,028 characters). No local checkout holds the text either: a search of
  every checkout under the reference directory found only Kilo's paraphrase
  (E8).
* The token counting API is the vendor's advice for estimating a request
  before sending it (WS4, verified).
* Long non-streaming requests: the vendor advises streaming for large
  `max_tokens`; its SDKs refuse a non-streaming request expected to exceed ten
  minutes (R3 §2, relayed; the threshold constant was not found).
* The Models API lists `max_tokens` and `max_input_tokens` per model (R3 §2,
  relayed; this SDK reads it, 0088 C105).

Evaluation: primary, current, and the strongest vendor evidence of the set.
It settles the rate-limit cost and the window behaviour by model generation.

### OpenAI (Responses and Chat Completions)

* "OpenAI recommends reserving at least 25,000 tokens for reasoning and
  outputs when you start experimenting with these models." The cap covers
  "reasoning tokens, visible output tokens, and non-visible formatting
  tokens"; at the cap the response has "`status` of `incomplete`" and
  `incomplete_details.reason` `max_output_tokens`, which "can happen before
  any visible text is produced" (WS2, verified).
* Chat Completions' `max_tokens` is deprecated for `max_completion_tokens`,
  which includes reasoning (R3 §1, relayed).
* `/v1/models` states no window and no output limit (R3 §1, relayed).

Evaluation: primary. It gives the one numeric reservation a vendor publishes
(25,000), above 0088's `Generate` ceiling of 16,384. It does not state the
default when the field is omitted, nor the refusal for a value above a
model's maximum (Gaps).

### Google Gemini

* `maxOutputTokens`: "The default value varies by model, see the
  `Model.output_token_limit` attribute"; the limit includes thought tokens;
  the model resource lists `inputTokenLimit` and `outputTokenLimit` (R3 §3,
  relayed). Forum reports (secondary) show a small limit spent wholly on
  thoughts, the same failure 0088 records.

Evaluation: primary for the field definitions; the thinking-budget ranges
and per-model maxima were not retrieved (Gaps).

### xAI

* Responses `max_output_tokens`: "Max number of tokens that can be generated
  in a response." "Only applies to visible output tokens (i.e. does not apply
  to tokens used for reasoning or function calls)." "Defaults to 128,000 when
  unset; set a larger value to allow longer generations." (WS3, verified
  through the fetch tool's relay of the vendor's API reference). Reasoning
  cannot be turned off on its reasoning models (R3 §4, relayed).

Evaluation: primary, and the one vendor whose limit excludes reasoning. It
means the reported failure, reasoning spending the whole limit, cannot happen
on Grok's Responses wire, and that a limit this SDK sends there, 8192 today or
0088's raised value, is below the service's own default for visible output.
The Chat Completions field text and per-model maxima were not retrieved.

### OpenRouter (an OpenAI-compatible gateway; reachable here only through chatcompat)

* `max_tokens`' "maximum value is the context length minus the prompt
  length"; for reasoning models it "covers reasoning tokens and visible output
  together on most providers"; an omitted parameter is omitted upstream so the
  provider's default applies (R3 §5, relayed).
* The listing states `context_length` and `top_provider.max_completion_tokens`
  (R3 §5, relayed; kilocode's gateway reads the same field, E7).
* Effort is a share of the output limit: about 95 % at xhigh, 80 % high, 50 %
  medium, 20 % low, 10 % minimal; an Anthropic budget is
  max(min(limit × ratio, 128,000), 1,024) (R3 §5, relayed).
* Credit is reserved up front from `max_tokens` (or a fixed cap when unset),
  and a request whose estimate does not fit is refused with 402 before it
  reaches a provider (R3 §5, relayed).

Evaluation: primary docs relayed, not re-fetched: no decision here rests on
them alone. They show the only documented refusal that depends on the
limit's size, and a reasoning share expressed as a fraction of the limit.

### Kilo gateway

* `max_tokens` is optional; `GET /models` documents `context_length` and
  prices, not an output cap (R3 §6, relayed), although the live listing this
  SDK reads carries `top_provider.max_completion_tokens` (0088 C33, C77).
  Its window refusal and 402 forms are 0088's C95 to C97.

Evaluation: primary docs that lag the listing's real shape. No clamping and
no too-large-limit refusal form is documented.

### models.dev

* `limit.context`, `limit.input` and `limit.output` per model; in a host's
  file `limit` is override-only (R3 §7, relayed). opencode and kilocode refresh
  it every 5 and 60 minutes; this SDK reads it as a document layer (0041-MADR).

Evaluation: the de facto open catalogue and the source of most stated
limits here. Its values are a host's or a lab's statement, not a measurement;
for a gateway alias or a local server they can overstate what the route takes
(0088 ST50, OQ7).

### LiteLLM

* The cost map separates `max_input_tokens` and `max_output_tokens`;
  `max_tokens` is the legacy field (R3 §7, relayed). Its Anthropic transform
  defaults `max_tokens` to the model's `max_output_tokens`, then a constant,
  and holds a thinking budget below `max_tokens` or drops thinking when the
  limit is below the minimum budget (R3 §7, relayed; the constant's value was
  not retrieved).

Evaluation: widely used, but read only in part (a 100k-character slice of
a 535k file); the clamping helper's body was not seen.

### Client libraries and other agents (each relayed or secondary)

| Library | Default when the caller names none | Source quality |
| :--- | :--- | :--- |
| Anthropic SDK | none; `max_tokens` required | relayed |
| OpenAI SDKs | none; the field is optional | relayed, not re-read in source |
| Vercel AI SDK | a per-model table, 4096 for an unknown Anthropic model, now with a warning (PR 17691) | relayed; a stale table capped one model at 32k against a real 128k (secondary) |
| LangChain.js | a model-prefix table, 4096 fallback; one entry contradicts its own comment | relayed source |
| LangChain Python | the model profile's `max_output_tokens`, 4096 fallback | relayed source |
| Roo Code | min(model max, 20 % of the window), 8192 for Anthropic hybrids, 16,384 for budget models | relayed paraphrase |
| Aider | sends none itself | relayed |
| TanStack AI, Continue, Cline | the model's ceiling / not documented / not retrieved | secondary / relayed / none |

Evaluation: the libraries agree on one thing, a per-model value with a
fallback for an unknown model, and disagree on the fallback (4096 to 32,000).
Their tables go stale (the Vercel and LangChain cases), which is the argument
for reading the limit from live metadata rather than a shipped table.

## Established practices, ranked by how well they are evidenced

1. **Take the default from the model's own stated limit, not a global
   constant.** opencode, kilocode, grok-build, LiteLLM, LangChain, Roo and
   Vercel all do; Anthropic and Gemini publish the limit per model in their
   listings. Evidence: strong (four primary agents, two vendor listings).
2. **Thinking counts inside the limit; leave the answer room.** Anthropic,
   OpenAI, Gemini and OpenRouter say so on their own pages; xAI is the
   exception (visible output only). Remedies differ: kilocode asks for
   Claude's whole limit; opencode and LiteLLM hold a budget below the limit;
   OpenRouter states effort as a share of it; OpenAI recommends reserving
   25,000. Evidence: strong for the fact, mixed for the remedy.
3. **Cap the default.** kilocode 32,000; opencode 256,000; Anthropic's
   wire default in opencode 32,000. Evidence: moderate (two agents, no stated
   measurement; the numbers differ eightfold).
4. **Fit the limit to the room the prompt leaves.** opencode and kilocode
   both do, from measured plus estimated input with a safety margin and a
   floor of 1,024; OpenRouter documents the arithmetic as its maximum;
   Anthropic 4.5+ makes it unnecessary there (it stops at the window).
   Evidence: moderate to strong.
5. **Set limits per call kind.** opencode (primary, compaction, none for
   titles), kilocode (small calls), grok-build (summary 100, suggestions 64
   plus a reasoning reserve). Evidence: moderate; the values disagree.
6. **Report a cut answer distinctly, and say when it was cut while
   reasoning.** kilocode's two warnings; grok-build's `had_reasoning`;
   OpenAI's incomplete-before-visible-text. Evidence: moderate.
7. **Run the complete tool calls of a cut answer; fail only the cut one.**
   grok-build by default; opencode fails the cut call with reason "length".
   Evidence: two agents.
8. **Continue automatically after a cut answer, with a bounded streak.**
   opencode (on), grok-build (off by default), kilocode and codex (no).
   Evidence: weak to mixed; prefill continuation is refused on Claude 4.6+
   (R3 §2, relayed), so a continuation must be a new user message.
9. **Let the service push or correct the limit.** grok-build's header only.
   Evidence: one agent, its own proxy.

## What no source settles

* Why 32,000 (kilocode, opencode's fallback) or 256,000: no rationale text
  anywhere.
* What OpenAI, Kilo and the OpenAI-compatible hosts default to when no limit
  is sent, and how OpenAI refuses a value above a model's maximum for current
  models (only a legacy, secondary form was found).
* The exact text of Claude's validation error for input plus `max_tokens`
  past the window on models before 4.5 (Anthropic's page names the error
  class only; Kilo's docs quote a form without attribution, E8).
* Whether a refused request is billed (no vendor page read says either way).
* Whether a continuation after a cut answer improves outcomes: no source
  measures it.
* Gemini's thinking-budget ranges and per-model maxima; xAI's Chat
  Completions field; per-model maxima for OpenAI and xAI.
* Whether xAI's public API sends grok-build's limit header.

## Evidence index

The code rows are checked by `python3 -B tools/claims.py
docs/reports/0088-REPORT-output-limits-across-agents-and-vendors.md` at the
revisions in the marker.

<!-- claims: self=18e8e72 gobble-cli=0589454 kilocode=58072bd6 opencode=4ce137b3 codex=57e64682 grok-build=2bdd1d6a pi=1402973a -->

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1 | kilocode's ceiling | `kilocode:packages/opencode/src/provider/transform.ts:22` | `export const OUTPUT_TOKEN_MAX = 32_000` |
| E2 | the catalogue limit, capped, or the ceiling when 0 | `kilocode:packages/opencode/src/provider/transform.ts:1796` | `return Math.min(model.limit.output, outputTokenMax) \|\| outputTokenMax` |
| E3 | Claude asks for its whole limit | `kilocode:packages/opencode/src/kilocode/session/llm.ts:71` | `return Math.max(base, input.model.limit.output)` |
| E4 | the window fit's safety margin | `kilocode:packages/opencode/src/kilocode/session/llm.ts:9` | `const SAFETY = 2048` |
| E5 | the cut-answer warning | `kilocode:packages/opencode/src/kilocode/session/processor.ts:45` | `The model hit its output limit, so this response may be incomplete.` |
| E6 | the cut-while-reasoning warning | `kilocode:packages/opencode/src/kilocode/session/processor.ts:47` | `produced no actionable output. Try disabling reasoning or increasing the output limit.` |
| E7 | a gateway entry with no cap gets 20 % of the window | `kilocode:packages/kilo-gateway/src/api/models.ts:302` | `Math.ceil(model.context_length * 0.2)` |
| E8 | a Claude window refusal quoted in Kilo's docs | `kilocode:packages/kilo-docs/pages/code-with-ai/agents/model-selection.md:112` | `input length and max tokens exceed context limit` |
| E9 | opencode's fallback | `opencode:packages/core/src/session/model-request.ts:54` | `const OUTPUT_TOKEN_FALLBACK = 32_000` |
| E10 | opencode's ceiling | `opencode:packages/core/src/session/model-request.ts:56` | `const OUTPUT_TOKEN_MAX = 256_000` |
| E11 | the catalogue limit is the default | `opencode:packages/core/src/session/model-request.ts:96` | `const requested = Math.min(limit.output > 0 ? limit.output : OUTPUT_TOKEN_FALLBACK, OUTPUT_TOKEN_MAX)` |
| E12 | titles send no limit | `opencode:packages/core/src/session/model-request.ts:284` | `because their reasoning is hard to budget.` |
| E13 | a thinking budget held to half the limit | `opencode:packages/ai/src/protocols/shared.ts:169` | `export const fitThinkingBudget = (budget: number, maxTokens: number \| undefined, minimum = 1) =>` |
| E14 | the continuation message | `opencode:packages/core/src/session/runner/llm.ts:41` | `Your last response hit the output token limit (stop reason: length).` |
| E15 | the continuation streak's end | `opencode:packages/core/src/session/runner/llm.ts:363` | `Response still truncated after two output token limit continuations` |
| E16 | codex's request type | `codex:codex-rs/codex-api/src/common.rs:287` | `pub struct ResponsesApiRequest {` |
| E17 | codex keeps 5 % of the window | `codex:codex-rs/protocol/src/openai_models.rs:414` | `95` |
| E18 | grok-build's agent turn sends the model's limit | `grok-build:crates/codegen/xai-chat-state/src/actor/request_builder.rs:86` | `max_output_tokens: self.state.sampling_config.max_completion_tokens,` |
| E19 | a global value is a fallback, not a clamp | `grok-build:crates/codegen/xai-grok-shell/src/agent/config.rs:3089` | `if let Some(v) = models.max_completion_tokens {` |
| E20 | why a cut text answer is not retried | `grok-build:crates/codegen/xai-grok-sampling-types/src/conversation.rs:565` | `` `Length` is usually context exhaustion `` |
| E21 | a session summary's limit | `grok-build:crates/codegen/xai-grok-shell/src/session/helpers/session_summary.rs:230` | `.with_max_output_tokens(100)` |
| E22 | the server can push a limit | `grok-build:crates/codegen/xai-grok-sampler/src/client.rs:218` | `.get("x-grok-max-completion-tokens")` |
| E23 | and it replaces the model's | `grok-build:crates/codegen/xai-grok-shell/src/session/acp_session_impl/session_setup.rs:550` | `Model max_completion_tokens changed via response header` |
| E24 | salvage's default streak | `grok-build:crates/codegen/xai-grok-shell/src/session/acp_session_impl/length_salvage.rs:8` | `const DEFAULT_LENGTH_CONTINUE_BUDGET: u32 = 2;` |
| E25 | this SDK's default | `llmprovider/settings.go:88` | `return settings{cfg: providerConfig{MaxTokens: 8192}}` |
| E26 | a cut answer with a tool call is an error on Chat Completions | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:533` | `if finish == llmprovider.FinishLength && len(msg.ToolCalls) > 0 {` |
| E27 | and on Messages | `llmprovider/internal/wire/messages/messages.go:349` | `if finish == llmprovider.FinishLength && hasCall {` |
| E28 | every incomplete Responses answer is an error | `llmprovider/internal/wire/responses/responses.go:413` | `return nil, incomplete(raw.IncompleteDetails.Reason)` |
| E29 | a caller's layer is a provider option | `llmprovider/catalog/lookup.go:34` | `func WithModelLayer(l Layer) llmprovider.Option {` |
| E30 | gobble's agent turn names no limit | `gobble-cli:agent/agent.go:685` | `req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.specs, Thinking: t.a.cfg.Thinking, Cache: t.a.cfg.Cache,` |
| E31 | gobble's provider carries no model document | `gobble-cli:llm/provider/provider.go:215` | `llmprovider.WithoutModelMetadata(),` |

### Web sources

| Id | Page | Read | Status |
| :--- | :--- | :--- | :--- |
| WS1 | Anthropic, rate limits: `https://platform.claude.com/docs/en/api/rate-limits` | 2026-11-22 | verified: the OTPM sentence quoted above |
| WS2 | OpenAI, reasoning guide: `https://developers.openai.com/api/docs/guides/reasoning` | 2026-11-22 | verified: the 25,000 reservation, the cap's scope, the incomplete status |
| WS3 | xAI, API reference: `https://docs.x.ai/docs/api-reference` | 2026-11-22 | verified through the fetch tool's relay: the three `max_output_tokens` sentences |
| WS4 | Anthropic, context windows: `https://platform.claude.com/docs/en/build-with-claude/context-windows` | 2026-11-22 | verified: the overflow behaviour by model generation, thinking inside `max_tokens`, 128k output, the token counting API |
| WS6 | Anthropic, stop reasons: `https://platform.claude.com/docs/en/build-with-claude/handling-stop-reasons` | 2026-11-22 | verified: the opt-in header's name and the models it is for; no error text for the earlier models in the part read |
| WS5 | every other page in R3 (OpenAI Chat reference and models, Gemini, OpenRouter, Kilo gateway, models.dev, LiteLLM, the libraries) | 2026-11-22 | relayed or secondary, as marked in each section; no decision rests on one alone |

## Round 2 (2026-11-22): what evidence can set the limit, and where every fixed number comes from

Added after the owner's reply on 0088's OQ9 (quoted in the MADR's second
amendment): the owner wants the limit set from discovered evidence, not from
picked numbers. Four more retrieval passes: R4 (pi's source, and goose's
upstream source relayed by a fetch tool), R5 (the history and stated reason
of every fixed number in opencode, kilocode, grok-build and codex), R6 (vendor
pages, OpenAPI specs and vendor SDK source, fetched verbatim with curl), R7
(what this SDK's catalogue already gathers). The synthesis step re-read the
code rows below at their revisions, fetched Gemini's and OpenAI's pages again
itself (WS7, WS8), and read the other vendor copies it relies on in the
verbatim files R6 saved. Like round 1, this section decides nothing.

### The sources of round 2

**pi** (the harness gobble ports, read at 1402973a). The default is the
model's catalogue limit, fitted to the context: catalogue limit, less a local
estimate of the input (3.5 characters per token, anchored on the last
reported usage), less 4,096 (E32, E33). No fixed ceiling exists. Summaries
send no limit since commit 6fcb48670, "remove fixed output limits from summary
requests": the old caps (80 % and 50 % of `reserveTokens`, 4,096 for branch
summaries) failed with "generation hit the token cap" when reasoning used
most of them, and "`compaction.reserveTokens` only controls when compaction
starts" (E34, E35). No refusal is parsed for a limit; a `length` stop below
the intended limit gets one compact-and-retry. Evaluation: primary and recent;
the summary fix is the very failure gobble's summary cap has today. Its 4,096
safety margin and 3.5 characters per token have no stated measurement.

**goose** (upstream, relayed by a fetch tool, line numbers approximate). It
sends no limit on OpenAI Chat Completions and Responses when neither the
user nor its models.dev record states one, sends 4,096 on Anthropic when
unknown, takes models.dev's output limit only when it is below the window, and
fits nothing to the remaining context. Its open issue 13369 records the cost
of copying the full catalogue limit into every request, the compaction
request included: strict endpoints refuse input plus the full reservation.
Evaluation: relayed, not verified; it supports sending nothing when the limit
is unknown on an optional field, and warns against a full reservation with no
window fit.

**The history of every fixed number** (R5). opencode's 32,000 ceiling came
from commit `469f667774` (2025-07-10), "set max output token limit to
32_000", with no body and no review text; opencode later raised its ceiling to
256,000 with the comment "No reply needs more, however much the model allows"
and kept 32,000 only as the fallback for an unknown limit, again with no
reason. kilocode inherited 32,000 and keeps it, except that it gives Claude
its whole limit (its changeset: Claude "responses ending early ... during long
reasoning"). The fitting margins (opencode's 1.15 estimate error and 1 %
headroom, kilocode's 1.3 factor and 2,048 safety, pi's 4,096) each carry a
comment and no measurement. The only measured numbers in the four agents are
opencode's compaction reserve ("Single steps grew by up to ~22k tokens at p99
in real sessions", E36) and its summary precompute lead. grok-build's 100,
4,096, 8,192, 64 and 256 carry no reason; its server pushes a limit in a
response header. codex once sent a per-model table's limit for a few hours,
then reverted it and deleted the table, with no stated reason. Evaluation:
every output limit in the four agents is either the model's own stated limit
or a constant with no recorded reason.

**The vendors** (R6, verbatim copies). What a client can learn, per service:

| Service | When no limit is sent | Listing states the output limit | What else the service documents |
| :--- | :--- | :--- | :--- |
| Anthropic | the field is required | yes, `max_tokens` (every model in the SDK's fixture) | the window as `max_input_tokens`; 4.5 and later stop at the window rather than refuse; a free, rate-limited token count; its SDKs refuse a plain request whose `max_tokens` implies more than ten minutes at 128,000 tokens an hour, 21,333 tokens (WS10), and cap Opus 4 and 4.1 plain requests at 8,192 |
| Gemini | "The default value varies by model, see the `Model.output_token_limit` attribute" (WS7, generateContent) | yes, `outputTokenLimit` | Gemini 4 caps the limit to the room left in the window itself; per-hop caps of 65,536 (streamed) and 32,768 (plain, some models); a token count endpoint |
| xAI | 128,000 of visible output; reasoning not counted (round 1 WS3; both Chat and Responses schemas) | no | the window on its legacy listing |
| OpenAI | not documented | no | "Your rate limit is calculated as the maximum of `max_tokens` and the estimated number of tokens" (WS8); Responses' minimum is 16 (WS11); an exact input-token count endpoint |
| OpenRouter | "omits it upstream ... so the provider applies its own default" (WS12) | `top_provider.max_completion_tokens`, nullable | a set limit routes only to providers that support that length; credit is held from the limit |
| Kilo | not documented | no in its docs; yes in its live listing (C33 of the MADR) | a window refusal that states the window and an approximate request size |
| Together | not documented | not in its spec; 10 of 265 listing entries carry `config.max_output_length` | a `context_length_exceeded_behavior` switch whose `truncate` "overrides max_tokens with the maximum context length of the model" (WS13): a cap for a limit above the window, not a fit of input plus limit |
| Ollama | `num_predict` "Default: -1, infinite generation" (WS9) | no | the allocated window from `/api/ps`, the trained one from `/api/show` |
| Hugging Face router | not documented | no | a window per serving provider |

Refusals that state their numbers: OpenAI's window refusal states the window
and the split ("29 in the messages, 20000 in the completion"; the SDK already
parses it, E40, and discards the numbers); Anthropic's limit refusal states
the requested and maximum limits and the model (secondary: quoted from real
responses in public issues); Gemini's states the accepted range (secondary).
Streaming for long outputs is advised by Anthropic, OpenAI (its background
guide) and xAI.

**This SDK's catalogue** (R7). Three listings state an output limit (Claude,
Gemini, Kilo); models.dev states one for 765 of 775 models in the 2026-10-12
fixture, equal to the Claude listing on every paired model, different from the
Gemini listing on 3 of 39, and copied from Kilo's listing on all 383 (not
independent). Fields fetched but unread: Together's `config.max_output_length`,
Hugging Face's per-provider `context_length`, the ChatGPT backend's
`context_window`. Claude's and Gemini's windows arrive only as
`MaxInputTokens`, so "the window" has no single accessor. Refusal numbers are
parsed for one boolean and dropped (E40). 0062's learner (a memo per cache,
256 models, a warning, an option to pass it back) is a complete template for a
learned fact. No success-response header is read.

### Every fixed number, with its provenance

| Number | Where | Provenance |
| :--- | :--- | :--- |
| 8192 | this SDK's default (E25) | none: inherited from mcplib; no record gives a reason |
| 32,000 | 0088 OQ2's streamed ceiling; kilocode's ceiling; opencode's unknown-limit fallback | none: opencode `469f667774`, no body |
| 256,000 | opencode's ceiling | a comment only ("No reply needs more") |
| 16,384 | 0088 OQ2's plain-request ceiling | the SDK's own arithmetic on its 300 s header wait at an assumed rate; not measured |
| half the window | 0088 OQ3's window rule | none: 0088's own choice |
| 300 s | this SDK's reply-header wait (E38) | none found: carried from the transport's extraction |
| 21,333 | Anthropic SDKs' plain-request guard | vendor SDK source: 3,600 s × N / 128,000 > 600 s (WS10) |
| 8,192 (Opus 4, 4.1) | Anthropic SDKs' plain-request caps | vendor SDK source (WS10) |
| 128,000 | xAI's default | vendor schema (round 1 WS3) |
| 16 | OpenAI Responses' minimum | vendor spec (WS11) |
| 4096 | this SDK's default thinking budget (E39) | cited to 0062-MADR only; not measured |
| 1024 | this SDK's low-effort thinking budget | measured: 1023 refused with HTTP 400 (2026-09-27) |
| 4,096; 3.5 chars per token | pi's safety margin and estimate | comments only |
| 1.15; 1 %; 1,024 | opencode's estimate error, headroom and floor | comments only |
| 16,000 or 15 % | opencode's compaction reserve | measured: p99 step growth of about 22,000 (E36) |
| 16,384; 20,000 | pi's (and gobble's) compaction reserve and kept recent tokens | none stated (E35) |
| 128 | gobble's title limit | none stated |
| 80 % and 50 % of the reserve | gobble's summary caps (pi's former ones) | none; pi removed them for the failure they cause (E34) |
| 65,536; 32,768 | Gemini's per-call output caps (a streamed hop; some plain calls), with a continuation token beyond | Gemini's pages (R6, thinking and long-decode); the service's own |
| 16 MiB | this SDK's whole-reply bound (`ReplyLimit`) and per-event bound (`EventLimit`) | this SDK's choice against an unbounded body; not measured |
| none (a zero margin) | a resend at the room a refusal states | the refusal's own numbers; OpenRouter's and Kilo's counts are "about" and "approximately", so such a resend can be refused again by a few tokens |

Added after record review 2 (2026-11-22, third): the window fitting that
Anthropic (4.5 and later) and Gemini 4 do is documented for their own APIs
only; through Kilo, OpenRouter or chatcompat it was not established, and
OpenRouter documents its maximum as "the context length minus the prompt
length", refusing above it.

### Practices, ranked again with round 2

1. **The model's own stated limit is the default.** pi, opencode, kilocode
   (for Claude), grok-build, goose (when below the window), LiteLLM and
   LangChain; Anthropic and Gemini publish it per model. Strong.
2. **Long outputs are streamed.** Anthropic (its SDKs enforce it), OpenAI and
   xAI say so. Strong for the fact.
3. **Send nothing where the service's own default is the model's maximum or a
   documented, generous value.** Gemini, xAI and Ollama document it; codex
   (OpenAI's own client) and goose do it; OpenAI charges a sent limit against
   the rate limit. Moderate to strong.
4. **Fit the limit to the room the input leaves, from the last reported
   usage plus an estimate of what was added**, or let the service fit it
   (Anthropic 4.5 and later, Gemini 4). pi, opencode, kilocode; their margins
   are unmeasured. Moderate.
5. **Summaries and small calls send no fixed limit.** pi (with a stated
   failure), opencode's titles. Moderate.
6. **Read the numbers a refusal states.** No agent does; the vendors' texts
   carry them (some only from secondary sources). Weak evidence, open ground.
7. **A fixed ceiling.** No source gives a reason for any. Weakest.

### Round 2 evidence

The code rows are checked by the claims marker above.

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E32 | pi's default is the catalogue limit, fitted to the context | `pi:packages/ai/src/api/simple-options.ts:46` | `maxTokens: clampMaxTokensToContext(model, context, options?.maxTokens ?? model.maxTokens),` |
| E33 | pi's safety margin | `pi:packages/ai/src/api/simple-options.ts:15` | `const CONTEXT_SAFETY_TOKENS = 4096;` |
| E34 | pi's summaries send no fixed limit | `pi:packages/coding-agent/CHANGELOG.md:15` | `Summaries now use the model's output limit, and` |
| E35 | pi's compaction reserve | `pi:packages/coding-agent/src/core/compaction/compaction.ts:128` | `reserveTokens: 16384,` |
| E36 | opencode's measured reserve | `opencode:packages/core/src/session/compaction.ts:133` | `Single steps grew by up to ~22k tokens at p99 in real sessions.` |
| E37 | opencode: OpenAI counts a sent limit | `opencode:packages/core/src/plugin/provider/openai.ts:324` | `OpenAI counts one against rate limits.` |
| E38 | this SDK's reply-header wait | `llmprovider/internal/transport/transport.go:82` | `ResponseHeaderTimeout: 300 * time.Second,` |
| E39 | this SDK's default thinking budget | `llmprovider/internal/wire/wire.go:53` | `const DefaultThinkingBudget = 4096` |
| E40 | the split this SDK parses and discards | `llmprovider/context_overflow.go:64` | `in the messages, (\d+) in the completion` |

### Round 2 web sources

| Id | Page | Read | Status |
| :--- | :--- | :--- | :--- |
| WS7 | Gemini, generateContent reference: `https://aistudio.google.com/api-reference/generate-content` | 2026-11-22 | verified by the synthesis step: the `maxOutputTokens` default sentence |
| WS8 | OpenAI, rate limits: `https://developers.openai.com/api/docs/guides/rate-limits` | 2026-11-22 | verified by the synthesis step: the limit counts toward the rate limit |
| WS9 | Ollama, Modelfile reference (`docs/modelfile.mdx`) | 2026-11-22 | verbatim copy read: `num_predict` default -1 |
| WS10 | Anthropic Python SDK `src/anthropic/_base_client.py` and `_constants.py` at 8baaef1c4d | 2026-11-22 | verbatim copy read: the guard's formula and the per-model plain caps |
| WS11 | OpenAI OpenAPI spec (`openai-openapi` master) | 2026-11-22 | verbatim copy read: Responses `max_output_tokens` `minimum: 16`; no default |
| WS12 | OpenRouter, parameters and provider selection | 2026-11-22 | verbatim copy read: omitted parameters go upstream absent; a limit steers routing |
| WS13 | Together OpenAPI spec | 2026-11-22 | verbatim copy read: `context_length_exceeded_behavior` |

## Round 3 (2026-11-22): how other projects and services let a caller discover a model's limits

Added for
[0089-MADR-model-output-limit-discovery.md](../decisions/0089-MADR-model-output-limit-discovery.md),
after the owner's reply on 0088's OQ9. It records findings and decides
nothing. Two retrieval passes read client libraries, gateways and servers
(their source at a named commit, or their published schema); the synthesis
step re-read at their source the three claims 0089 rests on most (WS14, WS16,
WS17). The passes fetched public, unauthenticated catalogues and documents
only: Vercel's AI gateway `/v1/models`, Portkey's pricing configuration and
OpenRouter's `openapi.json`; no key, sign-in or inference request.

### What services state

* **State the output limit in a listing:** Anthropic's Models API
  (`max_tokens`, "Maximum value for the `max_tokens` parameter when using this
  model"; `max_input_tokens`; both nullable; one model by id or alias,
  WS14, WS15), Gemini (`outputTokenLimit`, `inputTokenLimit`), OpenRouter
  (`top_provider.max_completion_tokens`, and per endpoint
  `/models/{author}/{slug}/endpoints`, each nullable, WS16), Groq's one-model
  read (an example only), Kilo's gateway (its OpenRouter-shaped
  `top_provider.max_completion_tokens`, read by this SDK), LiteLLM's proxy and
  Vercel's gateway (from their own maps).
* **State a window but no output limit:** xAI, Together (`context_length`;
  `config.max_output_length` on 10 of 265 models in this SDK's fixture),
  Hugging Face (per provider), Mistral (an SDK default of 32,768 when the
  server omits it), Fireworks, vLLM (`max_model_len`), llama.cpp (`n_ctx` per
  slot, `n_ctx_train`), LM Studio (`max_context_length`, and the loaded
  instance's), TGI (`max_total_tokens`), Ollama (`/api/show` trained,
  `/api/ps` served, WS17).
* **State nothing:** OpenAI, Cerebras, Bedrock, Azure's model list.
* **Refusals that state the cap:** Anthropic's `max_tokens: N > M, which is
  the maximum allowed number of output tokens for <model>`; OpenAI's "supports
  at most M completion tokens"; Gemini's "supported range is from 1
  (inclusive) to M (exclusive)"; TGI's `` `max_new_tokens` must be <= M ``.
  Only TGI's is read from source; the others are forum copies.
* **No service** found sends a model's limit in a response header.

### What client projects do with an unknown model

* A conservative constant: LangChain (4,096, silent), Continue (4,096),
  Cline's SDK (4,096); Vercel for a non-Claude id (4,096, with a warning).
* A family constant: LiteLLM for Claude (128,000, changed from 4,096 in a
  recent release), Vercel for a `claude-` id (128,000, warned), llm-anthropic
  (64,000, "the lowest output limit of any current model, so requests are
  never rejected"), Qwen Code (32,000, then one resend at 64,000 or the
  model's limit after a cut answer).
* Unknown kept as unknown, with a number still sent: Pydantic AI (`None`, and
  16,384 sent),
  Cline's extension (`source: "unknown"`, -1), Roo for non-Anthropic formats
  (the field omitted).
* Live first-party discovery: llm-anthropic (`llm anthropic refresh` reads the
  Models API and caches it on disk); Hermes for the context length only.
* Learning from refusals: Hermes parses the stated cap and window from about
  ten forms; its open issue #150519 records an output-cap refusal read as a
  window, persisted, and blocking every later start until removed by hand
  (WS18); PI-Desktop's issue #2156 asks to clamp to a stated ceiling once.
* Known wrong values: LiteLLM's map gave Claude Sonnet 4 an output limit of
  1,000,000 (litellm#14876); SWE-agent and Roo Code read a window or a legacy
  field as the output limit (`max_tokens: 128000 > 64000`); aider and Roo set
  a local server's output limit to its window; Continue's hand table states
  64,000 for models the others state at 128,000.
* No project reconciles two sources at run time: each has one winning source
  and override layers.

### Round 3 web sources

| Id | Page | Read | Status |
| :--- | :--- | :--- | :--- |
| WS14 | Anthropic Python SDK `src/anthropic/types/model_info.py` (main) | 2026-11-22 | verified by the synthesis step: `max_tokens` and `max_input_tokens`, their descriptions |
| WS15 | Anthropic Python SDK `src/anthropic/resources/models.py` (main) | 2026-11-22 | verbatim copy read: `retrieve`, "Model identifier or alias" |
| WS16 | OpenRouter `openapi.json` | 2026-11-22 | verbatim copy read by the synthesis step: `PublicEndpoint` and `TopProviderInfo` `max_completion_tokens` |
| WS17 | Ollama `api/types.go` (main) | 2026-11-22 | verbatim copy read by the synthesis step: `ProcessModelResponse.ContextLength` |
| WS18 | hermes-agent issue #150519 | 2026-11-22 | verified by the synthesis step through the public issue API |
| WS19 | LiteLLM, LangChain, Vercel AI SDK, Pydantic AI, llm-anthropic, aider, Continue, Cline, Roo Code, Qwen Code, models.dev, Portkey: source at the commits named in round 3's notes | 2026-11-22 | verbatim copies read; relayed |
