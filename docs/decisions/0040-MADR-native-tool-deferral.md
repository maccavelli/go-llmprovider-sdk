---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: vendor documentation read 2026-10-13 (below); the reference clients codex, grok-build, opencode and kilocode, and OpenCode's gateway source, each a local checkout (below); gobble-cli's records
informed: every caller of `Generate`, gobble-cli first
---

<!-- markdownlint-disable MD013 -->

# Let a Request Defer Tools Until a Tool Result Loads Them

## Context and Problem Statement

An agent offers many tools, but most turns need few of them. Every tool sent
costs context, and selection gets worse as the list grows. A harness that
keeps some tools back has to add them to `tools` when they are found, and
that change to the cached prefix breaks the prompt cache from that request
on. Anthropic and OpenAI each let a request send tools as *deferred*: kept
out of the model's context until a tool result loads them, with the cached
prefix left as it was. The SDK cannot ask for either.

### 1. What the SDK sends today

* **A tool is a name, a description, a schema and `Strict`**
  (`llmprovider/provider.go:17-27`). There is no way to mark a tool as
  deferred.
* **A tool result is text.** `FunctionCallOutputItem` has `CallID`, `Output`
  and `IsError` (`llmprovider/item.go:40-50`). It cannot carry a reference
  to another tool.
* **Every tool is sent on every request,** in the caller's order:
  * `wire.MessagesTools` (`internal/wire/tools.go:81`) for Claude and
    OpenCode's Messages route;
  * `wire.AddResponsesTools` (`:67`) for OpenAI, xAI and OpenCode's
    Responses route;
  * the Chat Completions encoder for Kilo, Together, Hugging Face, Ollama
    and OpenCode's chat route, and Gemini's own encoder.
* **Messages sends a tool result's content as one string**
  (`internal/wire/messages/messages.go:86`, `:170-172`).
* **Responses drops an output item it does not know** (`appendOutput`,
  `internal/wire/responses/responses.go:193-230`). A `tool_search_call`
  would be lost.
* **0039's cache breakpoint goes on the last tool** when there is no system
  prompt (`internal/wire/messages/cache.go:40`). Anthropic refuses
  `cache_control` on a deferred tool (§2). Deferral changes that rule.
* **Kilo's `reasoning_details` are already sent back** as they came
  (`internal/wire/chatcompletions/chatcompletions.go:58-61`;
  0021-MADR-harden-and-tune-after-the-v1-1-review.md W9).
  OpenRouter's protocol in §2 keeps its state there.
* **`Item` is sealed** (`llmprovider/item.go:14`), and new content is a new
  `Item` type (R9).

### 2. What the services offer

All of this was read on 2026-10-13. The quotes are short.

**Anthropic Messages** (platform.claude.com, "Tool search tool"):

* **A deferred tool.** A tool definition takes `"defer_loading": true`.
  "You still send every tool's full definition in the `tools` array on every
  request, including the deferred ones." "Tools with `defer_loading: true`
  load only when Claude discovers them through search."
* **The cache.** "Internally, the API excludes deferred tools from the
  system-prompt prefix. When Claude discovers a deferred tool through tool
  search, the API appends a `tool_reference` block inline in the
  conversation … The prefix is untouched, so prompt caching is preserved."
* **A search the caller runs.** "You can implement your own tool search
  logic … by returning `tool_reference` blocks from a custom tool":

  ```json
  {"type": "tool_result", "tool_use_id": "toolu_your_tool_id",
   "content": [{"type": "tool_reference", "tool_name": "discovered_tool_name"}]}
  ```

  "Every tool referenced must have a corresponding tool definition in the
  top-level `tools` parameter, normally with `defer_loading: true`."
* **A search the service runs.** `tool_search_tool_regex_20251119` and
  `tool_search_tool_bm25_20251119`, which return `server_tool_use` and
  `tool_search_tool_result` blocks. These must go back "unchanged" in the
  next request.
* **Refusals (400):**
  * "At least one tool must have defer_loading=false. All tools cannot be
    deferred."
  * "Tool reference 'unknown_tool' not found in available tools".
  * "A tool with `defer_loading: true` can't also carry `cache_control`".
* **Models.** Every current Claude model, from Haiku 4.5 and Sonnet 4.5 on.
  "Claude Opus 4.1 and earlier models don't support the tool search tool."
  The page names no beta header.
* **Strict mode** "builds from the full toolset, so `defer_loading` and
  strict mode compose".
* **Guidance.** "Keep your 3–5 most frequently used tools non-deferred."
  Tool search suits "10 or more tools", and selection "degrades once you
  exceed 30–50 available tools".

**OpenAI Responses** (developers.openai.com, "Tool search"):

* **Two modes.** Hosted, `{"type": "tool_search"}`. And run by the caller:
  a `tool_search` tool with `"execution": "client"`, a `description` and
  `parameters`, the caller's own search arguments.
* **Run by the caller.**
  * The model emits a `tool_search_call` with `execution: "client"`, a
    `call_id` and `arguments`, and the turn stops.
  * The caller answers with a `tool_search_output`:
    `{"type": "tool_search_output", "execution": "client", "call_id": …,
    "status": "completed", "tools": [ … ]}`.
  * The `tools` carry the loaded definitions, each with `"defer_loading":
    true` in the guide's examples. "Tools that were not listed as part of
    this array will not be available to the model."
  * In the guide's client example, the request's top-level `tools` holds
    only the `tool_search` entry. The deferred functions arrive through
    `tool_search_output`.
* **The cache.** "tool search is designed to preserve the model's cache."
  "All tools are loaded at the end of the model's context window. This
  holds true for both hosted tool search and client-executed tool search."
* **Models.** "only `gpt-5.4` and later models support `tool_search`."
* **Chat Completions** is not mentioned.
* **No rule that a tool stays loaded:** "Adding `tool_search` does not defer
  every function."

**OpenRouter** (openrouter.ai, "Tool Search Server Tool"), whose API Kilo's
gateway follows (Kilo's route is `/api/openrouter/chat/completions`, §3):

* **On Chat Completions, a versioned protocol.** A top-level
  `"deferred_tools": {"protocol": "v1", "profile": "portable",
  "validation": "runtime", "search": …}`, with deferred tools marked
  `"defer_loading": true`.
* **A search the caller runs.** `"search": {"type": "custom", "tool_id":
  …, "result_key": …}` names one of the request's non-deferred function
  tools. The caller "returns matching tool names under `result_key` in its
  result". The page gives no schema for that result.
* **Any model.** The portable profile "serves deferral with router-built
  search and call wrappers, on any model and provider".
* **The cache.** Discovery "does not disturb the tools already in the
  conversation, so prompt caching is preserved across a search". Each turn
  records what it loaded "inside encrypted reasoning" (`reasoning_details`).
* **The rules.** Send the same `tools`, `tool_choice` and `deferred_tools`
  on every turn. Return earlier assistant messages unchanged. `tool_choice`
  must be omitted, `"auto"`, or `{"type": "allowed_tools", …}`; any other
  value is a 400.
* **Anthropic models,** "on Anthropic-compatible endpoints that implement
  deferral", take plain `defer_loading` without the protocol.

**Other services.**

* **xAI** documents no tool search and no `defer_loading`. Its "deferred"
  completions are a different feature (asynchronous results).
* **Gemini, Together, Hugging Face and Ollama** document none.

### 3. What the reference clients and gateways do

**codex** (a local checkout at `823ea830c0`) runs its own search over
deferred tools on the Responses API, through the client mode:

* **The tool.** `ToolSpec::ToolSearch { execution, description, parameters }`,
  sent as `"type": "tool_search"` (`codex-rs/tools/src/tool_spec.rs:27-32`).
* **The items.** `ToolSearchCall { call_id, status, execution, arguments }`,
  where `arguments` is a JSON value, not a string. And
  `ToolSearchOutput { call_id, status, execution, tools }`
  (`codex-rs/protocol/src/models.rs:1094-1108`, `:867-873`).
* **The loaded tools** each get `defer_loading = Some(true)`
  (`codex-rs/tools/src/tool_search.rs:100`, `:105`).
* **No deferred tool is sent at the top level.** A tool is direct or
  deferred, never both. A model without search support gets the deferred
  tools as direct ones, from the first request
  (`codex-rs/core/src/tools/spec_plan.rs:267-275`).
* **Support is per model.** All 11 models in its catalog set
  `supports_search_tool: true` (`codex-rs/models-manager/models.json`),
  and its default is `false` (`models-manager/src/model_info.rs:137`).
  codex runs on the ChatGPT backend by default, so that backend takes the
  client mode on every model it serves.

**grok-build** (a local checkout at `2bdd1d6a`), xAI's own harness, defers
MCP tools behind two fixed tools:

* **`search_tool`** "discover[s] MCP tools via BM25 keyword search"
  (`crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:1`).
  Its description says it will "retrieve their input schemas" (`:218`), and
  each result carries the tool's `input_schema` (`:294`).
* **`use_tool`** "dispatch[es] to a discovered MCP tool"
  (`…/use_tool/mod.rs:1`, `:15`). Its arguments are `tool_name`, "Discovered
  MCP target name", and `tool_input`, an open object: "Inline remote
  arguments; use the discovered input schema" (`…/use_tool/input.rs:190-202`).
* **The tool list never changes,** so the cache holds on a service with no
  native deferral. A found tool reaches the model as text in a result, and
  runs through the one wrapper.

**opencode** (`907b3bc518`) and **kilocode** (`4a7c4e883f`), the clients,
have no deferral: no `defer_loading`, `tool_reference` or tool search
anywhere outside tests. They send every tool.

**OpenCode's gateway** (the same opencode checkout,
`packages/console/app/src/routes/zen/util/handler.ts`) forwards a request
body unchanged except for its `model` field (`:218-222`, and
`requestBody.ts`), and refuses to convert between formats: "Zen provider
format must match request format" (`:215`). Its Messages and Responses
routes therefore carry the native fields to the upstream as the caller sent
them.

**Kilo's gateway** (the kilocode checkout,
`packages/kilo-gateway/src/server/routes.ts:525`) posts chat completions to
`/api/openrouter/chat/completions`. Whether it forwards `deferred_tools` is
not documented, and is measured before it is used (D4).

### 4. What the consumers need

**gobble-cli** (`origin/main` `108d9f0`) records three requests to this SDK
in its `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`
D13 (`:483-488`):

* "native deferral: Anthropic's `defer_loading` with tool references, and
  OpenAI's `tool_search` with `defer_loading`";
* cache control, which 0039-MADR-request-side-prompt-caching.md (`v1.12.0`)
  delivered;
* a per-tool strict flag, which
  0032-MADR-tool-strict-and-result-error-flags.md delivered (`Tool.Strict`).

Deferral is the last one open. gobble's design depends on it:

* **Two tiers** (its D7). About 13 tools are direct. The rest are deferred:
  `request_permissions`, `lsp`, `monitor`, `notebook`, the MCP resource
  tools, and every MCP server's tools.
* **Its own search** (D9, D11). Its `tool_search` ranks deferred tools with
  an in-house BM25. It returns names and loads them by family.
* **Loads on an event** (D9.3). A call refused for a missing permission
  loads the `permissions` family.
* **Loads are append-only and recorded** (D9.4). "Until the SDK carries
  native deferral (D13), each mid-session load breaks the prompt cache
  once" (`:452`). Its F10 (`:212-216`) records why.
* **It validates every call's arguments** against the tool's schema before
  it runs it (its V33, `tool/tool.go:245`).

So gobble needs the **caller-run** form, on every provider it uses: Claude,
OpenAI and ChatGPT, OpenCode, Kilo and Grok among them. It already has the
search. What it lacks is a way to load the results without changing the
prefix.

**prepare-commit-msg** offers one tool, forced. It is unaffected.

**The problem:** let a caller mark tools as deferred and load them from a
tool result, once, in the neutral contract. Each provider should then use
the most native form its service, gateway or own harness offers, and give
the same behaviour elsewhere. A caller that defers nothing must see no
change.

## Decision Drivers

* **The contract stays neutral.** A new feature is a field (R5), and new
  content would be a new `Item` type (R9). The fewer of those the better:
  each one is something every caller's `switch` meets.
* **No change unless asked.** A request with no deferred tool is sent as
  today, byte for byte (R45).
* **The cache is the point.** Native deferral exists to keep the prefix
  stable. Where a service, a gateway or the service's own harness has a
  form that keeps it, the SDK uses it.
* **As native as each provider allows.** Each adapter follows the form its
  vendor's documentation or reference client uses: Anthropic's for Claude,
  codex's for OpenAI and ChatGPT, OpenRouter's for Kilo, grok-build's for
  Grok.
* **One caller code path.** A caller writes the same request for every
  provider. Where a service has no native form, the SDK gives the nearest
  behaviour itself, and says so.
* **No 400 in the middle of a session.** gobble switches models within a
  session. A model the SDK is not sure of falls back; it is never sent a
  field it may refuse.
* **The services' refusals are the SDK's to prevent.** Anthropic's three
  400s above can be known before the network, as `ErrInvalidRequest`.
* **The caller's search, not the service's.** gobble ranks its own tools,
  loads by family, and loads on events. A service-run search would replace
  all three.

## Considered Options

* **A. A caller-run search, in the contract:** `Tool.Deferred`, `Tool.Search`
  and `FunctionCallOutputItem.Load`, sent in each adapter's most native
  form, and emulated where there is none.
* **B. The services' own search:** Anthropic's regex or BM25 tool, OpenAI's
  hosted `tool_search`, OpenRouter's `openrouter:tool_search`, with new
  `Item` types for their blocks.
* **C. Pass-through fields per wire:** an `Extra` map on `Tool` and on a tool
  result, sent as given.
* **D. Nothing in the SDK:** the caller keeps appending tools, and the cache
  breaks once per load.

## Decision Outcome

Chosen option: **"A. A caller-run search, in the contract"**, because it is
the form gobble already built (its own search, families and event loads).
Anthropic, OpenAI and OpenRouter each have it natively, and grok-build shows
the form for Grok. It needs no new `Item` type, and every other provider can
give the same behaviour without it.

### D1. The contract

Three fields, each zero by default (Q1):

* **`Tool.Deferred bool`.** The tool is offered but kept out of the model's
  context until a tool result loads it.
* **`Tool.Search bool`.** The tool is the caller's search for deferred tools.
  At most one tool in a request has it. It is never deferred.
* **`FunctionCallOutputItem.Load []string`.** The names of the deferred tools
  this result loads. The result's `Output` is still the model's text, for
  example "loaded `lsp`".

A loaded tool stays loaded for the rest of the conversation. Every later
request carries the same `Tools`, deferred tools included, and the history
that loaded them.

**Refused before the network, with `ErrInvalidRequest`:**

* more than one `Search` tool, or a `Search` tool that is `Deferred`;
* every tool `Deferred` (Anthropic's 400);
* a `Load` name that is not a `Deferred` tool in `Tools` (Anthropic's
  "Tool reference … not found");
* `ForceTool` naming a deferred tool that no result in `Input` has loaded.
  The model cannot see it.

`llmtest`'s refusal cases gain each of these.

**Each adapter's form** (Q3). The form is chosen per request, from the
adapter, the route and the model:

| Adapter, route | Form | Models |
| :--- | :--- | :--- |
| `claude` | D2, Anthropic native | Claude 4.5 and later |
| `opencode`, Messages route | D2, through Zen's pass-through | the same Claude models |
| `openai`, ChatGPT session | D3, OpenAI client search | every model, as codex |
| `openai`, API key | D3 | `gpt-N.M` at 5.4 or later |
| `opencode`, Responses route | D3, through Zen's pass-through | `gpt-N.M` at 5.4 or later |
| `kilo` | D4, OpenRouter's protocol | every model, once measured |
| `grok` | D5, grok-build's dispatch | every model |
| everything else | D6, append on load | — |

"Everything else" is Gemini, Together, Hugging Face, Ollama, OpenCode's
chat and Gemini routes, and any model a row above leaves out (OpenCode's
MiniMax models on its Messages route, Claude 3 and Claude 4.0 and 4.1, GPT
before 5.4, and o-series models).

### D2. Anthropic Messages: native

* **A deferred tool** is sent as today with `"defer_loading": true`.
* **The `Search` tool** is an ordinary tool. Messages needs no marker for it.
* **A result with `Load`** is sent as a `tool_result` whose `content` is a
  list: one `text` block with `Output`, then one
  `{"type": "tool_reference", "tool_name": …}` per name. Any result can
  load, so gobble's event load (its D9.3) is native here. A result with no
  `Load` is sent as today, its content a string.
* **0039's breakpoint** goes on the last tool that is not deferred. That is
  the last tool of the fixed prefix, since deferred tools are not in it.

### D3. OpenAI Responses: native

* **The `Search` tool** is sent as `{"type": "tool_search", "execution":
  "client", "description": …, "parameters": …}`, with its `Schema` as the
  parameters. Its name is not sent: the wire has none. `Strict` is not sent
  on it.
* **Deferred tools are not sent at the top level,** as in the guide and in
  codex.
* **A `tool_search_call`** is decoded as a `FunctionCallItem` whose `Name`
  is the `Search` tool's name, and whose `Arguments` is the call's
  `arguments` as JSON text. No new `Item` type is needed.
* **Replay.** A `FunctionCallItem` naming the `Search` tool is sent back as
  a `tool_search_call` (`execution: "client"`, `status: "completed"`,
  `arguments` as a JSON value).
* **Its result** is sent as a `tool_search_output` whose `tools` are the
  full definitions of the `Load` names, each with `"defer_loading": true`.
  `Output` is not sent: the item has no text field. The package doc says
  so (R12).
* **A `Load` on any other tool's result,** such as gobble's event load, has
  no native form on this wire. Those tools are appended to the top-level
  `tools`, as in D6 (Q4).

### D4. Kilo: OpenRouter's protocol, once measured

Used only if the probe (Confirmation) shows that Kilo's gateway forwards
`deferred_tools` and honours it. Until then, and if it does not, Kilo
appends on load (D6).

* **The request** carries `"deferred_tools": {"protocol": "v1", "profile":
  "portable", "validation": "runtime", "search": {"type": "custom",
  "tool_id": <the Search tool's name>, "result_key": …}}`. The same object is
  sent on every turn.
* **Deferred tools** are sent in `tools` with `"defer_loading": true`.
* **The `Search` tool's result** carries the `Load` names under
  `result_key`, with `Output` beside them. The exact form is what the probe
  shows works.
* **`reasoning_details`** are sent back as they came, as they already are.
* **A `Load` on any other tool's result,** or a forced `ToolChoice`, which
  the protocol refuses, falls back to D6 for that request.
* **No `Search` tool,** no protocol: D6.

### D5. Grok: grok-build's dispatch

xAI has no native form. grok-build, xAI's own harness, keeps the tool list
fixed with a wrapper, so the cache holds. The SDK does the same, and the
caller sees ordinary calls:

* **The tools sent** are the tools that are not deferred, then one wrapper,
  `use_tool`, as grok-build names it. Its arguments are `tool_name`, a
  string, and `tool_input`, an object. Its description says it runs a tool
  that a search found, with the input schema the search gave. A caller tool
  named `use_tool`, in a request with a deferred tool, is refused with
  `ErrInvalidRequest`.
* **A result with `Load`** is sent with `Output`, then the loaded tools'
  definitions (name, description, input schema) as JSON, as grok-build's
  search returns them.
* **A call to the wrapper** whose `tool_name` names a deferred tool in
  `Tools` is decoded as a `FunctionCallItem` with that `Name`, and
  `tool_input` as its `Arguments`. A call naming anything else is decoded
  as a call to `use_tool`, and the caller answers it as an unknown tool.
* **Replay.** A `FunctionCallItem` naming a deferred tool is sent back as a
  call to `use_tool`, with the same `tool_name` and `tool_input`.
* **The cost.** `tool_input` is an open object, so the service holds it to
  no schema. gobble validates every call's arguments itself (its V33).
* **With `PreviousResponseID`,** nothing changes: the tool list is fixed,
  and the SDK needs no history to send it.

### D6. Everything else: append on load

Where D2 to D5 do not apply (Q2):

* **The tools sent** are the tools that are not deferred, in the caller's
  order. After them come the deferred tools that some result in `Input`
  loads, in the order they were first loaded.
* **The `Search` tool** is an ordinary function, and `Load` sends nothing:
  the result's `Output` is sent as today.
* **The cost.** Each load changes the tools, so the cache breaks once from
  that request on. That is what gobble does today (its D9.4). Never
  reordering means no tool breaks it twice.
* **With `PreviousResponseID`** (Gemini), the SDK cannot see the earlier
  loads. Every deferred tool is then sent as an ordinary one, so no loaded
  tool goes missing.

### D7. One warning, no capability

* **`WarningDeferralEmulated`** goes on a response whose request had a
  deferred tool and was sent as in D5 or D6, or appended a load as D3 and D4
  allow. Its `Detail` names the form: dispatched through `use_tool`, or
  appended as loaded. It follows 0034-MADR D7: advisory, per response, with
  the model.
* **No new capability.** No form refuses a request, so a capability would
  gate nothing (R10). 0039-MADR D5 decided the same for `Cache`. Each
  provider's package doc lists its form, its routes and its models.

### D8. Out of scope

* **The services' own search** (option B): Anthropic's regex and BM25 tools,
  OpenAI's hosted `tool_search`, and OpenRouter's `openrouter:tool_search`
  and regex and BM25 search types. These would need new `Item` types for
  `server_tool_use`, `tool_search_tool_result` and the hosted items, which
  must be replayed unchanged. A later record can add them if a caller needs
  them.
* **OpenAI namespaces,** MCP toolsets (`mcp_toolset`, MCP servers on
  Responses), and Anthropic's computer and browser toolsets.
* **`input_examples`.** The SDK's `Tool` has none.
* **Dispatch for the other non-native adapters** (Q2 revisited). They append
  on load: their models are not tuned to a wrapper, and their calls keep
  each service's own schema handling.

### Consequences

* Good, because gobble can defer its MCP and occasional tools on Claude,
  ChatGPT, GPT-5.4 and later, and the OpenCode routes that serve them, and
  load them without breaking the cache. That was the point of its D13.
* Good, because Grok keeps the cache too, through the form xAI's own harness
  uses, and Kilo may, through OpenRouter's protocol.
* Good, because one request works on every provider. The warning says when
  a form was emulated.
* Good, because the contract gains three fields and a warning kind, and no
  `Item` type. A caller's `switch` is unchanged.
* Good, because Anthropic's three refusals are caught before the network.
* Neutral, because a request with no deferred tool is unchanged on every
  wire.
* Bad, because the adapters differ in what reaches the model of a search
  result. On Responses its text is not sent. On Grok the loaded schemas are
  sent as text, which costs tokens once per load.
* Bad, because on Responses and on Kilo a `Load` from a tool other than
  `Search` is appended, which costs the cache once. gobble's event load
  (its D9.3) pays that there.
* Bad, because on Grok a deferred tool's arguments are not held to its
  schema by the service. gobble's own validation covers it; another caller
  must do the same.
* Bad, because the SDK keeps per-model rules (the D1 table), which can go
  stale. A stale rule costs a lost saving, never a 400, since an unknown
  model falls back.
* Bad, because the conversation's history must carry the loads. A caller
  that trims the history and drops a loading result also drops the loaded
  tools.

### Confirmation

* **Offline:**
  * `TestRequest_Defer*`: each of D1's refusals, and `llmtest`'s cases for
    every provider.
  * Wire tests for D2 to D6: the deferred field, the `tool_reference`
    blocks, the `tool_search` tool and items, the `deferred_tools` object,
    the `use_tool` wrapper both ways, the appended order, and the
    `PreviousResponseID` cases.
  * A test of the D1 table: each adapter, route and model picks its form.
  * `TestAddCache` extended: the breakpoint skips deferred tools.
  * New wirecase scenarios, `defer-offer` (deferred tools and a search tool)
    and `defer-load` (a history with a search call and a loading result),
    with goldens on every provider. Every existing golden is unchanged.
  * `make api-check` shows only the additions: three fields and one
    `WarningKind`.
* **Live, billed, with the owner's approval, before the code is written**
  (Q5). On each path, three requests: the model calls the search, the result
  loads a tool, and the model calls it. The paths are Claude, OpenCode Zen's
  Messages route, OpenAI with an API key, OpenCode Zen's Responses route, a
  ChatGPT session, Kilo with `deferred_tools`, and Grok with the wrapper. The
  probe settles what the vendors leave unclear:
  * **A text block with `tool_reference` blocks** in one `tool_result`.
    Anthropic's example shows `tool_reference` blocks only.
  * **A stateless replay of `tool_search_call`** with `store: false`, and
    `arguments` as an object.
  * **Whether Zen's upstreams take these fields,** since Zen passes them
    through unchanged.
  * **Whether Kilo forwards `deferred_tools`,** and the form of a custom
    search's result under `result_key`.
  * **That Grok calls through the wrapper** with arguments that match the
    loaded schema.
  * **That the cache holds:** the second and third requests read the first
    request's prefix (`CachedTokens`).

  Each fact that fails changes this record before any code is written.
* **Live, after the code:** a gated test, as 0039's was, repeating the probe
  through the SDK.

## Pros and Cons of the Options

### A. A caller-run search, in the contract

* Good, because it is what gobble built, and what Anthropic, OpenAI and
  OpenRouter offer.
* Good, because it needs no new `Item` type: a search call is a function
  call on every wire.
* Good, because it has a cache-keeping form on every adapter gobble names
  except the four that append.
* Bad, because the forms differ per adapter, so the SDK keeps a table of
  them, with per-model rules.

### B. The services' own search

* Good, because the caller writes no search.
* Bad, because the services differ: Anthropic searches with regex or BM25,
  OpenAI over namespaces and MCP servers, OpenRouter with regex.
* Bad, because each needs new `Item` types for its blocks, replayed
  unchanged (R9).
* Bad, because it replaces gobble's ranking, families and event loads (its
  D9, D11).
* Bad, because xAI and the other services have none to use.

### C. Pass-through fields per wire

* Good, because it is the smallest change in the SDK.
* Bad, because the caller writes one form per service and has to know which
  wire each provider uses. That is what R5 and the neutral contract exist
  to prevent.
* Bad, because the Responses items (`tool_search_call`, `tool_search_output`)
  and Grok's wrapper cannot be passed through a field.

### D. Nothing in the SDK

* Good, because there is no change.
* Bad, because gobble's cache breaks once per load on every provider, which
  is what its D13 asks the SDK to end.

## More Information

### Owner questions

**Answered on 2026-10-13:**

* **Q1. The names:** `Tool.Deferred`, `Tool.Search` and
  `FunctionCallOutputItem.Load` (D1). These are gobble's own words: its
  `deferred` exposure, its `tool_search`, and its loads.
* **Q2. The fallback:** append on load (D6), which is gobble's D9.4.
  * Revisited once grok-build's dispatch was found: append stays for every
    non-native adapter except Grok (D8).
* **Q3. The model rules.** The owner asked for the rules to be "optimized
  for, and compatible with grok, codex/chatgpt, opencode, kilo, and gobble
  (as close to natively as possible) not just anthropic", assessed across
  every adapter. The assessment is §2 and §3. The owner then chose:
  * the per-adapter table (D1);
  * grok-build's dispatch for Grok (D5);
  * OpenRouter's protocol for Kilo, after a probe (D4).
* **Q4. A `Load` on Responses from a tool other than `Search`:** append it
  (D3), so that gobble's event load works on GPT.

**Accepted by the owner on 2026-10-13,** with Q1 to Q4 as answered.

**Open:**

* **Q5. The live probe** (Confirmation): seven paths of three requests, 21
  billed requests, before any code. The ChatGPT session uses the owner's
  subscription. 0040-PLAN-native-tool-deferral.md's Phase 0 asks for this
  approval before it runs anything.

### Sources

* Anthropic, "Tool search tool",
  `platform.claude.com/docs/en/agents-and-tools/tool-use/tool-search-tool`,
  read 2026-10-13.
* OpenAI, "Tool search",
  `developers.openai.com/api/docs/guides/tools-tool-search`, read 2026-10-13.
* OpenRouter, "Tool Search Server Tool",
  `openrouter.ai/docs/guides/features/server-tools/tool-search`, read
  2026-10-13.
* xAI: a search of its documentation on 2026-10-13 found no tool search and
  no `defer_loading`.
* codex `823ea830c0`, grok-build `2bdd1d6a`, opencode `907b3bc518` and
  kilocode `4a7c4e883f`, local checkouts, the files cited in §3.
* gobble-cli, `origin/main` `108d9f0`:
  `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`
  (F10 `:212-216`, D7 `:413-430`, D9 `:440-452`, D11 `:471`, D13 `:483-488`),
  and `docs/reports/0011-REPORT-native-tools-survey.md` §3, whose quotes of
  Anthropic's and OpenAI's pages (2026-10-08) agree with those above.

### Evidence index

Added on acceptance, 2026-10-13. Each row is checked, by the checker
0041-PLAN commits as `tools/claims.py`, against go-llmprovider-sdk
`3e81e3d`, codex `823ea830c0`, grok-build `2bdd1d6a`, opencode
`907b3bc518`, kilocode `4a7c4e883f` and gobble-cli `108d9f0`, each at that
commit. A range in the text above is anchored here at its first line.

| Id | Claim | Where | Matched |
| :--- | :--- | :--- | :--- |
| E1 | `Tool` | `llmprovider/provider.go:17` | `type Tool struct {` |
| E2 | `Tool.Strict` is the last field | `llmprovider/provider.go:26` | `Strict *bool` |
| E3 | `FunctionCallOutputItem` | `llmprovider/item.go:40` | `type FunctionCallOutputItem struct {` |
| E4 | `Item` is sealed | `llmprovider/item.go:14` | `type Item interface {` |
| E5 | Messages tools | `llmprovider/internal/wire/tools.go:81` | `func MessagesTools(tools []llmprovider.Tool) []map[string]any {` |
| E6 | Responses tools | `llmprovider/internal/wire/tools.go:67` | `func AddResponsesTools(body map[string]any, tools []llmprovider.Tool, choice llmprovider.ToolChoice) {` |
| E7 | a Messages block | `llmprovider/internal/wire/messages/messages.go:86` | `type Block struct {` |
| E8 | a tool result's content is one string | `llmprovider/internal/wire/messages/messages.go:171` | `appendBlock(wire.RoleUser, Block{Type: "tool_result", ToolUseID: arena.Keep(v.CallID), Content: arena.Keep(v.Output),` |
| E9 | Responses keeps only the output items it knows | `llmprovider/internal/wire/responses/responses.go:193` | `func appendOutput(r *llmprovider.Response, out outputItem) (refused bool) {` |
| E10 | 0039's breakpoint on the last tool | `llmprovider/internal/wire/messages/cache.go:40` | `tools[len(tools)-1][keyCacheControl] = CacheMarker(retention)` |
| E11 | Kilo's `reasoning_details` replay | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:58` | `ReplayReasoningDetails replays` |
| E12 | codex's `tool_search` tool | `codex:codex-rs/tools/src/tool_spec.rs:27` | `#[serde(rename = "tool_search")]` |
| E13 | codex's search call item | `codex:codex-rs/protocol/src/models.rs:1094` | `ToolSearchCall {` |
| E14 | codex's search output item | `codex:codex-rs/protocol/src/models.rs:867` | `ToolSearchOutput {` |
| E15 | codex marks loaded tools deferred | `codex:codex-rs/tools/src/tool_search.rs:100` | `tool.defer_loading = Some(true);` |
| E16 | codex defers only where the model supports search | `codex:codex-rs/core/src/tools/spec_plan.rs:267` | `exposures = if model_info.supports_search_tool` |
| E17 | codex's default is no search | `codex:codex-rs/models-manager/src/model_info.rs:137` | `supports_search_tool: false,` |
| E18 | grok-build's search tool | `grok-build:crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:1` | `` `search_tool` — discover MCP tools via BM25 keyword search. `` |
| E19 | its results carry input schemas | `grok-build:crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:294` | `"input_schema": r.input_schema,` |
| E20 | grok-build's dispatch tool | `grok-build:crates/codegen/xai-grok-tools/src/implementations/use_tool/mod.rs:15` | `pub const USE_TOOL_NAME: &str = "use_tool";` |
| E21 | the dispatch tool's open input | `grok-build:crates/codegen/xai-grok-tools/src/implementations/use_tool/input.rs:193` | `Inline remote arguments; use the discovered input schema` |
| E22 | Zen refuses to convert formats | `opencode:packages/console/app/src/routes/zen/util/handler.ts:215` | `Zen provider format must match request format` |
| E23 | Kilo's gateway posts to OpenRouter's path | `kilocode:packages/kilo-gateway/src/server/routes.ts:525` | `/api/openrouter/chat/completions` |
| E24 | gobble's request to the SDK | `gobble-cli:docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md:483` | `D13. A request to go-llmprovider-sdk` |
| E25 | gobble's cache break until D13 | `gobble-cli:docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md:452` | `Until the SDK carries native deferral (D13)` |

### Related records

* [0039-MADR-request-side-prompt-caching.md](0039-MADR-request-side-prompt-caching.md):
  the cache markers D2 must keep off deferred tools, and its D5, which
  chose no capability.
* [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md):
  `Warning`, whose D7 shape D7 here follows.
* [0032-MADR-tool-strict-and-result-error-flags.md](0032-MADR-tool-strict-and-result-error-flags.md):
  `Tool.Strict` and `IsError`, the last fields added to `Tool` and
  `FunctionCallOutputItem`.
* [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md):
  W9, Kilo's `reasoning_details` replay, which D4 relies on.
* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md):
  R5, R9, R10 and R12.
* gobble-cli `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`
  D13: the request this record answers.
