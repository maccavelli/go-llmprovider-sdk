---
date: 2026-11-24
subject: how qwen-code decides a request's output limit, uses the Responses API, streams a session and caches prompts, compared with this SDK, for 0088-MADR, 0089-MADR and the pending 0091 record
examines: "the read-only checkout qwen-code at 36fc21f5d8; this repository at 46f43d7; no live call"
---

<!-- markdownlint-disable MD013 MD024 -->

# qwen-code: Output Limits, the Responses API, Session Streaming and Caching, Recorded for 0088

This report records findings. It decides nothing. The decisions it bears on
belong to
[0088-MADR-default-output-limit-from-model-facts.md](../decisions/0088-MADR-default-output-limit-from-model-facts.md),
[0089-MADR-model-output-limit-discovery.md](../decisions/0089-MADR-model-output-limit-discovery.md)
and the pending 0091 record, adopt the Responses API per provider. It pairs
with
[0088-REPORT-output-limits-across-agents-and-vendors.md](0088-REPORT-output-limits-across-agents-and-vendors.md),
which surveys four other agents.

Reference report, 2026-11-24. Source: the read-only checkout `<gitrepos>/qwen-code` at `36fc21f5d8` (2026-11-22).
Read by two research agents and spot-checked by the coordinator (the constants at `tokenLimits.ts:17-18, 35, 51, 254`;
the model table `:350-388`; the clamp `:60-106`; the Responses cut handling `responses-converter.ts:390, 421-440, 527`;
the config-only wire choice `docs/users/configuration/model-providers.md:263`; the stream timeouts
`openaiContentGenerator/constants.ts:1, 33, 60`; the tool sort `tools/tool-registry.ts:305, 1059`; the event-bus limits
`<qwen-code>/packages/acp-bridge/src/eventBus.ts:197-243`). No live calls. Paths: `[C]` is
`<qwen-code>/packages/core/src`, `[CLI]` is `<qwen-code>/packages/cli/src`, `[BR]` is
`<qwen-code>/packages/acp-bridge/src`, `[D]` is `<qwen-code>/docs`.

## Answer to the question asked

The expectation was that qwen-code does not default to 8192 and has no hard-coded per-model heuristics, but detects
and optimises each model's output maximum dynamically. The code shows a mixed picture:

| Claim | What the code does |
| :--- | :--- |
| Does not default to 8192 | **True.** An unknown model gets 32,000 (`DEFAULT_OUTPUT_TOKEN_LIMIT`, `tokenLimits.ts:18`); 8,192 appears only as the `gemini-*` and `deepseek-chat` fallback rows. |
| No hard-coded per-model heuristics | **False.** A curated regex table sets output limits per family (`OUTPUT_PATTERNS`, `tokenLimits.ts:350-388`) and wins over the catalog; a second table sets input windows (`:257-343`). |
| Detects limits dynamically | **Partly.** Model facts come from models.dev (a bundled snapshot plus a cache refreshed at most every 24 h), not from each provider's `/models` listing (`providers/model-discovery.ts:35-60` reads only `id` and `created`). |
| Optimises the limit per request | **True.** Every main-turn request is sized to the room left in the window after the prompt (`clampOutputTokensToWindow`, `tokenLimits.ts:84-107`). |
| Learns from the service | **No for output limits.** It never parses "max_tokens: N > M" or "supports at most"; it learns only a window ceiling from context-overflow errors, used once for the compaction retry (`llm-chat.ts:5203-5260`). |

## How the output limit is decided (main turn)

1. **Ceiling**, the first that is set (`[C]/core/llm-chat.ts:3719-3730`):
   - the request's `config.maxOutputTokens`;
   - the user's `samplingParams.max_tokens`;
   - the env var `QWEN_CODE_MAX_OUTPUT_TOKENS`;
   - `defaultOutputCeiling(model)` (`tokenLimits.ts:496-506`): the model's output limit, clipped to `OUTPUT_TOKEN_CEILING`
     = 64,000, except Opus 4.6+ and `qwen4-max`, which keep their full limit.
2. **The model's output limit** (`tokenLimits.ts:412-448, 480-488`): the curated regex row, else the catalog's
   `output`, else 32,000.
3. **Window clamp** (`tokenLimits.ts:84-107`, applied at `llm-chat.ts:4398-4419`):
   `min(ceiling, max(floor, window − prompt − margin))`, where `margin = max(10,000, 5% of window)` and
   `floor = max(1, min(4,000, window − prompt))`.
   - The prompt count is the provider's reported usage when available. Otherwise it is a chars/4 estimate
     (`services/tokenEstimation.ts:101-129`). New content is multiplied by 1.5, and an estimate-based count gets a
     20,000 pad (`llm-chat.ts:942`).
   - Images count 1,600 tokens each (`compactionInputSlimming.ts:33`).
   - No tokenizer and no `countTokens` call exist anywhere in core.
4. **Wire adapters reconcile**: a user value on a known model becomes `min(user, model limit)`; on an unknown model it
   passes through (`provider/default.ts:384-433`, `anthropicContentGenerator.ts:968-1045`). If `samplingParams` is set
   at all, no default is injected (`default.ts:389`). DashScope adds one probed cap: `minimax-m2.5` 32,768
   (`dashscope.ts:57-59`).

**When the limit is sent.** The main turn always carries a limit (`llm-chat.ts:4407-4419`). Side queries may omit it
on the Responses and Gemini wires, which apply no ceiling of their own (`baseLlmClient.ts:238-290`). Anthropic always
sends `max_tokens`, because its wire requires it.

**Where model facts come from.**

- Order of use: curated tables, then models.dev, then the defaults (output 32,000, window 200,000). A user's
  `contextWindowSize` overrides all of them.
- The catalog: the embedded `models/generated/model-registry.json` (fetched 2026-10-03, 299 entries), or a newer cache
  refreshed in the background from `https://models.dev/api.json`, at most once a day, with ETag. The refresh is used
  by the next process.
- The refresh runs only when usage statistics are enabled (`config.ts:4400`).
- It can be switched off with `QWEN_CODE_MODELS_DEV=off`.
- Only eleven allowlisted catalog providers are kept (`model-catalog-refresh.ts:54-66`). Hand corrections override a
  few entries (`model-catalog.ts:253-270`).

## When a reply is cut at the limit

- The cut is mapped to `MAX_TOKENS` on every wire: OpenAI `length`, Anthropic `max_tokens`, Responses `incomplete`.
- **Escalate once.** With no user or env override, the turn is retried once at 64,000 (the partial reply is dropped),
  unless the first limit was already that high (`llm-chat.ts:4551-4564, 5662-5719`).
- **Then continue.** Up to three continuation turns follow, keeping the partial reply plus a recovery message and
  re-clamping each time (`:5726-5821`). There is no continuation if the cut turn holds a tool call.
- **Cut tool calls:**
  - On the Chat wire, unterminated JSON is treated as a `length` finish when usage corroborates it (completion tokens
    at least half the limit, `converter.ts:2234, 2265`), and the arguments are marked incomplete.
  - On Anthropic, the cut tool_use is discarded (`anthropicContentGenerator.ts:1666-1683`).
  - The scheduler rejects a cut Edit call with guidance to split the write (`coreToolScheduler.ts:3630-3700`).
- An explicit user or env limit disables escalation.

## Thinking budgets

- **Anthropic** (`anthropicContentGenerator.ts:1189-1291`):
  - Adaptive thinking on 4.6+ models, with no budget.
  - Below 4.6, an effort ladder: 16k, 32k, 64k, 96k, 128k.
  - The budget is clamped to `max_tokens − 1`, and thinking is dropped below 1,024.
- **DashScope/Qwen:** `thinking_budget` ladders. `qwen4-max`'s `max_tokens` covers thinking plus answer, which is why
  it is exempt from the 64k ceiling.
- **Responses:** `reasoning.effort` passes through with `summary: auto`, and no tie to `max_output_tokens`.

## Multiple agents

- **Subagent and team definitions have no output-limit field.** Their run config holds only time and turn limits, and
  a TODO reads "Consider adding max_tokens as a form of budgeting" (`agents/runtime/agent-types.ts:70-80`).
- **A subagent with its own `model:`** gets its own generator view, and its limit is decided by the same table and
  window clamp for that model (`subagents/subagent-manager.ts:1453-1520`). Otherwise it inherits the session's. The
  agent loop carries forward only an escalated limit (`agent-core.ts:1234`).
- **Workflow budgets** are cumulative soft caps on total output tokens, not per-request limits:
  `QWEN_CODE_MAX_TOKENS_PER_WORKFLOW` (`workflow-budget.ts`), and a `+500k` turn directive capped at 100M
  (`turn-budget.ts:33`).
- **The managed-agent server** accepts a per-session `generation_config.max_tokens`, inherited by child sessions, and
  forwards it to core (README `:1071-1090`).
- **The ACP bridge** has no output-limit logic.
- **Compaction thresholds** follow the window, not the output limit (`chatCompressionService.ts:268-304`).

## The Responses API

- **Supported, chosen by configuration only:** `wireApi: 'responses'` per model on an `openai` provider
  (`models/types.ts:73-82`). "No endpoint detection or automatic fallback occurs when an API request fails"
  (`model-providers.md:263`). It's a config error on anthropic, gemini, vertex or qwen-oauth providers.
- **Request** (`openaiResponsesContentGenerator/responses-pipeline.ts:413-606`):
  - It is a hand-rolled fetch to `${baseUrl}/v1/responses`.
  - It always streams, with `store: false` and a session `prompt_cache_key`.
  - It sends `reasoning` with `summary: auto` and `include: ["reasoning.encrypted_content"]`.
  - `max_output_tokens` is sent only when set.
  - `previous_response_id` is declared but never set: the full history is resent each turn.
- **Stream decode** (`responses-converter.ts`):
  - Text, refusal and reasoning deltas are emitted.
  - Call arguments are buffered, and calls are emitted at `output_item.done`. A missing `added` or delta event falls
    back to the done item, as a "non-compliant proxy" may do (`:305-325`).
  - `response.completed` always maps to STOP (`:390-404`).
  - `response.incomplete` maps by `incomplete_details.reason`, defaulting to `max_output_tokens`, to `MAX_TOKENS`, then
    the escalate and continue flow.
  - `response.failed` and `error` events throw. Only `server_error` maps to 500 and `rate_limit_exceeded` to 429.
- **Reasoning replay:**
  - A reasoning item with `encrypted_content` is replayed as a reasoning input item next turn.
  - Without it, the summary is replayed as plain assistant text.
  - On a 400 naming an over-long replayed id or `invalid_encrypted_content`, it retries once with those items
    downgraded (`responses-pipeline.ts:300-360`).
- **Known deviations it documents:**
  - Endpoints that stream `reasoning_text` without `encrypted_content` cannot have reasoning replayed
    (`model-providers.md:505`).
  - OCI's gpt-oss Responses route rejects function tools (`:420-424`).
  - Nothing describes a service that reports a cut as `completed`.

## Compared with the SDK (records 0088, 0089, 0091)

| Topic | qwen-code | go-llmprovider-sdk |
| :--- | :--- | :--- |
| Unknown model | sends 32,000 | Messages wire: the route's smallest current limit, else refuses with a fix message (0089 DL51); optional wires send nothing (the service's default) |
| Source of limits | curated regex table first, then models.dev | provider listings and models.dev, with evidence per fact; no hand table (0089); overrides via models.json |
| Per-request sizing | window − prompt − max(10k, 5%), floor 4,000; chars/4, images 1,600 tokens | window − estimate − 4,096; content counted as pi does, images 4,800 characters (0088 ST87, Phase 1c) |
| Global ceiling | 64,000 (except Opus 4.6+, qwen4-max) | none: the model's stated limit (0088 ST99) |
| Learning from refusals | none for output; window ceiling used once | resend once at a stated cap, memo per process, pass-back across starts (0089 D4–D6) |
| Cut reply | escalate once to 64k, then up to 3 continuation turns (agent-level) | returned as a response finishing `length` with its usable part; cut call dropped with `WarningToolCallCut` (0088 Phase 1b); continuation is the caller's choice |
| Responses: which providers | any OpenAI-type provider, by per-model config | openai, grok, OpenCode per model; the others stay on Chat on measured evidence; chatcompat opt-in (0091) |
| Responses: `completed` | always STOP | the same today; 0091 adds an output-tokens-at-limit check on chatcompat's Responses path, because probes found services hiding cuts behind `completed` |
| Reasoning replay | encrypted when present, else summary as text; retry once on `invalid_encrypted_content` | encrypted only; plain-text reasoning read but not replayed on chatcompat (0091 OQ9) |

## Points worth a look (observations, not decisions)

- **Truncation recovery at the agent level:** escalate once, then continue up to three times, and refuse cut Edit
  calls with a "split the write" message. This is a gobble-side behaviour; relevant to gobble 0065/0066.
- **Retrying once with replayed reasoning downgraded** on `invalid_encrypted_content` or an over-long id: the SDK has
  no such retry.
- **Corroborating a Chat-wire cut call by usage** (completion tokens at least half the limit): close to 0091's
  output-tokens-at-the-limit signal.
- **A margin of `max(10k, 5% of window)` versus the SDK's 4,096:** qwen-code's is larger, because its chars/4 estimate
  under-counts CJK and tool-heavy prompts (`tokenLimits.ts:80-82`).
- **qwen-code would miss the hidden cuts our probes measured** (Hugging Face, Ollama, and Kilo's cut call reported as
  `completed`), since `completed` always maps to STOP.

## Session streaming

### The path, layer by layer

Each layer is an async generator unless stated.

1. **Wire readers:**
   - OpenAI chat and Anthropic use their vendor SDK streams, wrapped in `withStreamGuards` (`[C]/core/stream-guards.ts`).
   - Responses has its own fetch and SSE reader, with the guards re-implemented inline
     (`responses-pipeline.ts:937-1176`).
   - Gemini has no idle or lifetime guard. The lifetime cap is documented as not implemented there
     (`[C]/core/contentGenerator.ts:103-109`).
   - Each wire converts its chunks into one Gemini-shaped response type, dropping empty chunks.
2. **Chat:** `LlmChat.sendMessageStream` (`[C]/core/llm-chat.ts:4638-4760`) yields CHUNK, RETRY, COMPRESSED and
   MODEL_FALLBACK events.
3. **Turn:** `Turn.run` (`[C]/core/turn.ts:820`) maps them to typed events: content, thought, tool-call preparation,
   tool-call request, finished, retry.
4. **Agent loop:**
   - Tool calls are collected during the model round, and the whole batch runs after it ends
     (`[C]/core/agent-loop.ts:327-339, 408-466`).
   - Starting tools mid-stream is "decided: not implemented" (`[D]/design/2026-11-11-streaming-tool-dispatch.md:5`).
5. **Consumers:**
   - The Ink TUI.
   - OpenTUI.
   - Headless and SDK, as stream-JSON over stdio.
   - The ACP child: each content or thought event becomes an ACP `session/update` over NDJSON stdio, then the bridge's
     event bus, then re-emission as SSE (`GET /session/:id/events`) or over the `/acp` WebSocket or SSE JSON-RPC.
   - The managed-agent server: deltas are committed as append-only journal events split at 3,072 bytes
     (`[CLI]/serve/hosted-text-deltas.ts:9-12`), then batched by the Java control plane.

### Backpressure and buffering

- **Bounded at every hop:**

  | Hop | Bound |
  | :--- | :--- |
  | Bridge event bus, per subscriber | 256 frames and 2 MiB; warns at 75%, evicts with `client_evicted` on overflow (`eventBus.ts:197-198`) |
  | Bridge event bus, per session | 64 subscribers (`eventBus.ts:243`) |
  | Replay ring | 8,000 events and 8 MiB (`eventBus.ts:200-222`) |
  | NDJSON to the ACP child | 64 MiB frames, 256 queued messages, a 5 s saturation grace (`[BR]/ndJsonStream.ts:52, 70-76`) |
  | SSE writer | awaits `drain`; reaped after 60 s stalled (`[CLI]/serve/sse-events.ts:726-1002`) |

- **Coalescing happens only at the edges.** Every inner layer publishes one frame per delta.
  - The Ink TUI flushes every 60 ms and joins consecutive text.
  - The web shell batches every 16 ms and renders at most every 50 ms.
  - The managed server batches at 75 ms, 64 events or 64 KiB.
- **Deltas are dropped in two places only:** the SDK's WebSocket queue (drop-oldest at 256) and subscriber eviction.

### Cancellation

- Each request gets a child AbortController, made that way to avoid SDK listener leaks (`pipeline.ts:632-635`).
- The guards' `finally` aborts the request and closes the iterator (`stream-guards.ts:288-295`).
- A parent abort wins over a retryable timeout.
- On abort, the delivered parts go into history and a partial assistant record is staged for the session file
  (`llm-chat.ts:7653-7691`).
- **A partial tool call is never released.** The OpenAI finish chunk carrying it is held, and open Anthropic
  `tool_use` blocks are not yielded.
- The ACP `cancel`, an HTTP disconnect mid-prompt and `/quit` all abort the active request.

### Errors mid-stream

| Situation | Handling |
| :--- | :--- |
| Before the first event | HTTP retry, 7 attempts, 1.5 s to 30 s backoff (`[C]/utils/retry.ts:100-103`) |
| Transport reset before any content | replayed up to 2 times; a RETRY event clears pending tool calls |
| Text already delivered | up to 3 continuation turns asking the model to resume; none if a tool call was delivered |
| Repeated output | overlap stripped (`mergeDeliveredPrefix`) |
| Stall | idle watchdog 240 s and lifetime cap 900 s, monotonic, counting only upstream waiting (`constants.ts:33, 60`; `stream-guards.ts:196-205`); settable by config or env |
| Stream anomaly (no finish reason, no text, leaked protocol tags, malformed tool call) | typed errors with their own retry budgets |

### Sessions

- **On disk:** an append-only, tree-structured JSONL file per session (`uuid`, `parentUuid`), at
  `<home>/.qwen/projects/<cwd>/chats/<id>.jsonl`, written on a serial queue (`[C]/services/chatRecordingService.ts`).
- **The main chat is recorded once per reply, not per delta.** Background subagents keep a per-delta `.stream`
  sidecar, flushed every 64 KiB or 100 ms.
- **Resume:** walks the tree. Abandoned branches stay in the file after a rewind.
- **Remote clients resume by event id:**
  - Per-session ids increase by one from 1. Clients send `Last-Event-ID`, and the server sends `retry: 3000` and a
    15 s heartbeat.
  - A gap in the ring sends an id-less `state_resync_required`, and the client reloads the session.
  - `replay_complete` marks where replay ends and live events begin.
  - WebSocket clients have no resume.
  - Prompts carry an idempotency key, remembered for 128 keys per session.
- **Concurrency:**
  - One ACP child per workspace serves its sessions, each with its own event bus and FIFO prompt queue.
  - A subagent has its own chat, and its events are re-emitted as tagged ACP updates.

## Caching and the hot path

### Prompt caching

**Anthropic** (`[C]/core/anthropicContentGenerator/converter.ts`) uses up to 4 `cache_control` breakpoints:

- the system prompt (split into a stable prefix and the rest when they differ);
- the last tool definition;
- the last stable block of the last user message, skipping a volatile tail.

The TTL is 5 minutes by default, or 1 hour on request. Beta headers are derived from the body.

**OpenAI and Responses:**

- `prompt_cache_key` = `qwen-code:<session>[:agent]`, sent to api.openai.com only on the chat wire, and always on
  Responses.
- Up to 3 explicit breakpoints at stable boundaries on Responses, and on gpt-5.6+ chat (`prompt-caching.ts`;
  `[D]/design/responses-stable-prefix-cache.md`).

**DashScope:** a cache header, plus markers on the system message and the last stable message.

**Gemini:** implicit caching only.

**Keeping the request prefix byte-identical:**

- Tools are sorted by name whatever order they registered in (`tool-registry.ts:305, 1059`).
- Request-only content (a memory catalog, re-attached images) goes in a volatile tail after the cache boundary and is
  never stored in history.
- Thinking blocks are normalised so a turn serialises the same wherever it sits.
- Compaction sends the main session's own tools, system prompt and history, so it reuses the cache.

**Measured** from `cache_read_input_tokens`/`cache_creation_input_tokens` (Anthropic) and
`prompt_tokens_details.cached_tokens` (OpenAI). Normalised to one count, and exported to OpenTelemetry.

### Other caches

| Cache | Detail |
| :--- | :--- |
| HTTP | shared undici dispatchers per proxy, keep-alive 60 s; an early HEAD warms TCP and TLS ("100–200 ms") |
| File reads | per session, keyed by device and inode, storing mtime and size; gates Edit and short-circuits repeat reads |
| Bounded LRU/TTL caches | web fetch (15 min, 32 entries), transcript index, YAML front matter, ripgrep ignores, path validation |
| Token estimates | not memoised |
| DNS | none found |

### The per-delta and per-turn hot path

**Per delta:**

- Each chunk is converted, empty chunks are filtered, and one finish chunk is held to merge a later usage chunk.
  Usage-only chunks are not retained: an earlier version grew without bound, crashing at 515 MiB; after the fix,
  memory stayed at 101–131 MiB over 13.57M chunks.
- Tool-call arguments are scanned incrementally, tracking nesting depth and string state, and parsed once when the
  depth returns to 0 (`streamingToolCallParser.ts:293-316`).
- The Responses reader splits each network chunk on newlines and parses each frame, with no buffer cap seen
  (`responses-pipeline.ts:1070-1071`).

**Per turn:**

- History is passed without a deep clone, because a clone could exhaust V8 memory on long sessions
  (`llm-chat.ts:3028-3050`).
- Tool sets are compared, and a version is bumped only on change.
- Compaction gates use the last reported prompt count plus an estimate of the new content.

**Budgets with numbers:**

- Daemon benchmark ceilings: cold start p99 10 s, session create p99 5 s, tree RSS 1.5 GB.
- First-output latency gates (paired-median confidence bounds).
- Scrolling at 45 FPS or more.
- Most benchmarks use fake providers; no live time-to-first-token figure is checked in.

### Compared with the SDK: prompt caching

The SDK at `46f43d7`, read by a third research agent; the coordinator checked the breakpoint placement
(`llmprovider/internal/wire/messages/cache.go:24-54`), the idle timeout (`llmprovider/internal/transport/reply.go:14`)
and the `prompt_cache_key` rule (`llmprovider/internal/wire/cache.go:8`). Paths below are in the SDK.

| Topic | qwen-code | SDK |
| :--- | :--- | :--- |
| Who asks for caching | always on | opt-in per request: `Request.Cache` = `CacheShort` (5m) or `CacheLong` (1h); the zero value asks for nothing (`llmprovider/contract.go:106-125`; 0039-MADR D1) |
| Anthropic breakpoints | up to 4: system (split into a stable prefix and the rest), last tool, last stable user block | at most 2: the top-level automatic `cache_control`, plus one at the end of the fixed prefix (the system prompt, or with none the last non-deferred tool) (`messages/cache.go:24-54`; 0039-MADR D2, D6). No breakpoint on a message block, none placed by the caller |
| Where Anthropic caching applies | Anthropic wire | Claude, OpenCode's messages route, and Kilo for `anthropic/` models (OpenRouter-style top-level `cache_control`, `providers/kilo/kilo.go:439-443`) |
| `prompt_cache_key` | per session, always (`qwen-code:<session>[:agent]`) | the provider's session id, only when `Request.Cache` is set (`wire/cache.go:8-15`): OpenAI with a key, Grok, OpenCode's responses route, Together; the ChatGPT backend always sends it |
| Explicit OpenAI/Responses breakpoints, `prompt_cache_retention` | yes (Responses; gpt-5.6+ chat) | not sent; out of scope (0039-MADR D3, D6) |
| DashScope cache header | yes | none (the SDK has no DashScope provider) |
| Gemini | implicit only | implicit only; `cachedContents` out of scope (`providers/gemini/gemini.go:61-62`) |
| Byte-stable prefix | tools sorted by name; request-only content in a volatile tail; thinking normalised | tools in the caller's order, no sorting; no volatile-tail mechanism; the thinking effort can change between sends after a refusal (0062-MADR). Body keys are sorted by `encoding/json` and messages are typed structs with a fixed field order (0028-MADR A6). No test pins whole-request byte stability across turns |
| Measurement | cache read and write normalised to one count; exported to OpenTelemetry | `Usage.CachedTokens`, `CacheWriteTokens` and `CacheWrite1hTokens`, read per wire (`messages.go:750-767`, `responses.go:922-939`, `chatcompletions.go:945-989`, `generatecontent.go:498-511`; 0036, 0037); the shares feed `catalog.ObservedWorkload` for model ranking; no hit-rate or cost computation |
| Live evidence | none checked in | 0039-PLAN Phase 4: Claude, Zen-Claude and Kilo-Claude each wrote then read the cache |

`internal/cacheref` is not prompt caching: it carries the catalog's process caches (listings, probes, learned refusals,
thinking settings, output caps) from a provider's options to its readers (`llmprovider/internal/cacheref/cacheref.go`).

### Compared with the SDK: streaming and the hot path

| Topic | qwen-code | SDK |
| :--- | :--- | :--- |
| Delivery | async generators through five layers, then an event bus, SSE or WebSocket | one pull iterator, `iter.Seq2[Event, error]`, with four event types (`llmprovider/stream.go:13-40`) |
| Backpressure | bounded queues per hop, eviction of slow subscribers | none needed: `yield` runs inside the read loop, so a slow consumer stops the read (`wire/sse.go:171-178`). A consumer stalled over 300 s trips the idle timer (derived from `transport/reply.go:110-112`, not documented) |
| Bounds | 64 MiB NDJSON frames, 8,000-event ring; Responses reader uncapped | 16 MiB per event (`EventLimit`), 4,096 stream indexes, 16 MiB for an assembled `Generate` answer; the stream as a whole is unbounded (`wire/sse.go:16-19`) |
| Idle and lifetime | idle 240 s, lifetime 900 s | idle 300 s (`ErrIdleTimeout`); no lifetime cap, the caller sets a context deadline (`transport/transport.go:36-40`) |
| Retry before the first event | transport replay up to 2 | `StreamEventsRetry` within `MaxSends` (7), for effort, thinking-shape and forced-choice refusals; network retry is the caller's `WithStreamRetry` (3 attempts, 1–30 s, honours Retry-After) (0051-MADR) |
| After output began | up to 3 "please resume" continuation turns | never resent; continuation is the caller's choice (0034-MADR D3) |
| Cancellation | child AbortController per request | the request context derives from the caller's; stopping the range or cancelling closes the body, with no goroutine left (tests `wire/stream_0031_test.go:229-282`) |
| Cut tool calls | held and dropped per wire | held back (`wire.HeldCalls`) and dropped with `WarningToolCallCut` (0088 Phase 1b) |
| Per-event cost | incremental argument parser; one held finish chunk | `json.Unmarshal` into a typed struct per event; the reader reuses its buffer and copies only lines longer than 64 KiB (`wire/sse.go:94-115`); text in a `strings.Builder` |
| Benchmarks | daemon and first-output budgets; fake providers | request encoders: typed versus map, 100 messages 12.3 µs and 4 allocations versus 54 µs and 1,008 (0028-MADR); stream benchmarks for 4 MiB Responses deltas and reasoning details; no benchmark of the Messages stream path |
| Connections | undici dispatchers per proxy, keep-alive 60 s, a warming HEAD | one shared `http.Transport` per configuration: 16 idle per host, idle 90 s, HTTP/2 with a 30 s health ping, TLS 10 s, response header 300 s (`transport/transport.go:28, 80-124`) |

### Gaps in the SDK against qwen-code (facts, not decisions)

1. No breakpoint on the conversation (the 3rd and 4th Anthropic slots), and no caller-placed breakpoint (0039-MADR D6
   rejected it).
2. No sorting of tools, and no volatile-tail boundary for request-only content, so a caller's reordering or
   per-request content breaks the cache.
3. Caching is off unless the request asks; `prompt_cache_key` follows the same rule (except the ChatGPT backend).
4. No explicit OpenAI/Responses breakpoints or `prompt_cache_retention`.
5. No test that the request bytes stay stable across turns.
6. No hit-rate reporting beyond the `Usage` fields.
7. No lifetime cap on a stream and no idle setting for callers (the 300 s is fixed).
8. No benchmark of the Messages stream decoder's allocations.

Each of these is a candidate for its own record, if wanted. Agent-level features (continuation turns, session files,
an event bus for remote clients) belong to gobble and magic-cli-remote, not the SDK.

## Unanswered

- Streaming: the Gemini SDK's own timeout and abort handling, agent-team stream multiplexing, and live TTFT numbers
  (none checked in).
- Stale qwen-code docs against its code:
  - the idle-timeout doc says 120,000 ms, the code 240,000;
  - the large-frame doc says "no cap", the code 64 MiB;
  - a comment still says `tmp/<project_id>` for session files.
- No code reads `/models` for limits or persists learned output limits.
- No per-subagent output budget exists beyond the TODO.
- The escalation design doc is partly stale against the code (`[D]/design/adaptive-output-token-escalation/...:12`
  says "64K floor"; the code escalates to the 64,000 ceiling).
