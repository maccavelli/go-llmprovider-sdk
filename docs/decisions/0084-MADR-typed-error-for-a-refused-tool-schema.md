---
status: accepted
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's request of 2026-11-17 and answers to Q1 to Q7 of the same day, relayed by the coordinator (each label quoted below); the owner's paste of a gobble-cli failure of 2026-11-16 (quoted below); the coordinator's read-only diagnosis of 2026-11-16, checked here line by line; this repository at b109cea; gobble-cli at cecc892; mcp-server-magictools at 368ac6a; the codex checkout at 57e64682f8, the opencode checkout at 4ce137b39e and the kilocode checkout at 58072bd696; xAI, OpenAI, Anthropic and Google documentation fetched 2026-11-17"
informed: gobble-cli, which sends every MCP tool's schema through this module and is the first caller of the error this record adds; every caller that matches errors.Is(err, llmprovider.ErrInvalidRequest)
---

<!-- markdownlint-disable MD013 MD024 -->

# Classify a Provider's Refusal of a Tool's Schema as Its Own Kind, Naming the Tool It Refused, So a Host Can Drop That Tool Instead of Failing Every Turn

## Context and Problem Statement

On 2026-11-16 the owner ran gobble-cli with an MCP server attached and the
model `grok-4.6`. Every prompt failed with (the owner's paste, P1):

```text
Error: llm: api status 400: llmprovider: invalid request: grok HTTP 400 invalid-argument: Failed to start sampling:
  [invalid_client_tool_schema] mcp__magictools__execute_pipeline: tool parameter root must be an object type (root
  schema is an anyOf/oneOf union with a non-object branch)
```

### The mechanism, read line by line

* **The schema.** The MCP server's `execute_pipeline` input schema is an
  object (`"type": "object"`, `properties`, `additionalProperties: false`)
  with a root-level `anyOf` of two branches, `{"required": ["target",
  "intent"]}` and `{"required": ["session_id"]}`, neither of which declares
  a `type` (M1, M2). It is valid JSON Schema.
* **gobble passes it on.** gobble's `inputSchema` adds `"type": "object"`
  and `"properties": {}` only when they are missing, and otherwise sends the
  schema as the server gave it (G1). It sends every direct tool on every
  model call (G2), so one schema a provider refuses fails every turn (G3).
* **This SDK sends it as given.** `Tool.Schema` is `any` (C1). Every wire
  sends it through `wire.ToolSchema`, which replaces only a nil schema with
  an empty object (C2); the Responses, Messages, Chat Completions, Gemini
  Interactions and OpenCode generateContent encoders all call it (C3 to C6,
  C26), Grok's among them, after deferral (C7). The only schema the SDK
  rewrites is a copy it builds for structured outputs when a Claude model
  refuses a forced tool; the tool definition in `tools` stays exactly the
  caller's (C8; R9). Before the network the SDK checks only that a schema
  marshals (C9).
* **xAI refuses it.** xAI documents that "The root of a `parameters`
  schema must be an object", that a root `anyOf`/`oneOf` "also works when
  every branch is itself an object", and that a tool whose root is "an
  `anyOf`/`oneOf` with a non-object branch ... is rejected with a `400`
  error that names the tool" (W1).
* **The SDK classifies the 400 as an invalid request.** `ClassifyHTTPError`
  keeps the service's code in `Code` and its message, redacted and bounded
  to 512 bytes, in `Message` (C10 to C12). xAI's `invalid-argument` 400 is
  `ErrAuthFailure` only when its message names an API key (C13); any other
  400 falls to the status-only mapping, `ErrInvalidRequest` (C14).

### What a caller can read today

A run on an archive of `b109cea` (X1) fed `ClassifyHTTPError` the owner's
message in xAI's recorded body shape (F1):

| Field | Value |
| :--- | :--- |
| `errors.Is(err, ErrInvalidRequest)` | true; `Retryable()` false |
| `Status` | 400 |
| `Code` | `invalid-argument`: xAI's code for every invalid argument, a refused key (C13) and a refused effort (F9) included |
| `Message` | `Failed to start sampling: [invalid_client_tool_schema] mcp__magictools__execute_pipeline: tool parameter root must be an object type (...)` |
| a tool's name, a parameter, a typed reason | none: nothing parses them. `parseAPIErrorBody` reads no `param` (C15), and 0062 recorded that "No field holds an error's `param`" (R12) |

So the tool's name is in the text, and nothing typed says "a tool's schema
was refused" or which tool. Two side findings of the same run (X1):

* **Redaction keeps the name and can eat the word after it.** For tools
  named `lookup_api_key`, `get_token`, `rotate_secret`, `store_password`,
  `fetch_key` and `signature`, the message became
  `... [invalid_client_tool_schema] lookup_api_key: [REDACTED] parameter root ...`:
  inferred from the rule's pattern, the key-value rule reads `name: value`
  (C16). The names themselves survived in all nine Grok cases and both
  OpenAI cases tried.
* **The kind text is the only class a person sees.** `Error()` starts with
  the kind's text, "llmprovider: invalid request" (C17).

### What each provider refuses, and how it says so

Sources: **fixture** is a recorded live response in this repository
(0062-PLAN Phase 1, 2026-10-25; each case sent one tool whose schema was
`{"type":"string"}`, a non-object root; F1 to F14, R13); **paste** is P1;
**documentation** is a page fetched 2026-11-17 (W1 to W7). Every cell with
no source is "not established".

| Provider and route | Refused (source) | Status | Code / type | Tool named? Where | Root `anyOf` with an untyped branch (the owner's shape) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Grok, Responses (`grok-4.5`) | root `{"type":"string"}` (fixture F1) | 400 | `invalid-argument`, message tag `[invalid_client_tool_schema]` | yes: the name after the tag, before `:` (F1) | refused, 400, same code and tag, named the same way (paste P1, `grok-4.6`); documented (W1) |
| Grok, documented limits | root not an object or union of objects; "max 350 tools per request" (documentation W1) | 400 (W1) | not documented | "names the tool" (W1) | as above |
| OpenAI, Responses (`gpt-6-luna`) | root `{"type":"string"}` (fixture F2) | 400 | code `invalid_function_parameters`, type `invalid_request_error` | yes, twice: `param` `tools[0].parameters` (the index, F10), and the message `Invalid schema for function '<name>'` (F2) | not established. Strict mode documents "the root level object ... must be an object, and not use `anyOf`" (W2); Responses "will fall back to non-strict" when a schema cannot be made strict (W3) |
| OpenCode Zen, Responses route (`gpt-5.5`) | root `{"type":"string"}` (fixture F3) | 400 | as OpenAI | as OpenAI (F3) | not established |
| Anthropic, Messages (`claude-sonnet-5`) | root `{"type":"string"}` (fixture F4) | 400 | type `invalid_request_error` | by index only: `tools.0.custom.input_schema.type: Input should be 'object'` (F4) | not established. The documentation says only that `input_schema` is "A JSON Schema object" (W4), and that error values "can expand, and new `type` values can appear" (W5) |
| OpenCode Go, Messages route (`qwen3.8-flash`) | nothing: root string accepted, 200 (fixture F5) | – | – | – | not established |
| Gemini, Interactions (`gemini-3.8-flash`) | nothing: root string accepted, 200 (fixture F6) | – | – | – | not established. Documentation: `parametersJsonSchema` "must describe an object" (W6); very complex tools are refused with "`400 INVALID_ARGUMENT`" (W7) |
| OpenCode Zen, Google generateContent route (`gemini-3.8-flash`) | root `{"type":"string"}` (fixture F7) | 400 | status `INVALID_ARGUMENT` | yes, in backticks in free text, behind the gateway's `Upstream request failed: [400]` (F7) | not established |
| Chat Completions: Hugging Face, Kilo (`openai/gpt-6-luna`), Ollama, OpenCode Zen chat, Together | nothing: root string accepted, 200 on each (fixtures F8, F11 to F14) | – | – | – | not established |
| OpenAI ChatGPT plan session; `chatcompat` servers | not established | not established | not established | not established | not established |

Four facts follow from the table:

* **Refusal depends on the route, not the model family.** `gpt-6-luna`
  refused a root string on OpenAI's Responses API (F2) and accepted it
  through Kilo's Chat Completions (F11); `gemini-3.8-flash` accepted it on
  Gemini's Interactions (F6) and refused it through OpenCode Zen's
  generateContent route (F7).
* **Where a provider refuses, three of four forms name the tool**;
  Anthropic gives the tool's position in the list it received (F4).
* **No provider's response is a typed tool-schema refusal.** OpenAI's code
  `invalid_function_parameters` is the nearest (F2); xAI's tag is inside
  free text under a code shared with refused keys and efforts (F1, C13, F9).
* **Nothing establishes** what any provider says about a union below the
  root, whether it reports one bad tool or all of them, or whether xAI's tag
  and word order are stable.

### How the upstream agents handle it (read-only checkouts, current with upstream as of 2026-11-16 by the owner's statement)

None of the three matches a schema refusal in an error: `git grep` for
`invalid_function_parameters`, `Invalid schema for function` and
`invalid_client_tool_schema` finds nothing in any of them (U0). All three
rewrite schemas before sending instead:

| Agent | Rule | For | Where |
| :--- | :--- | :--- | :--- |
| opencode | a root with no `type` gets `"type": "object"`; a root `not: {type: null}` is dropped; a root `anyOf` holding a bare `{type: object}` branch is dropped. A root that has a `type` is returned unchanged | every provider | U1 |
| opencode | Gemini: `required` filtered to declared properties; draft-04 boolean exclusive bounds made numeric; draft-07 tuples made `prefixItems`; recursive `$ref` loops cut unless through an optional property or possibly-empty `items` | the Gemini protocols, and any model whose id matches `gemini` | U2, U3 |
| opencode | Moonshot: an `enum` with no `type` gets one; `$ref` siblings dropped; tuple `items` and `prefixItems` made one `items` (`anyOf` when several); `unevaluatedItems` dropped | Moonshot, and model ids matching `kimi` | U4 |
| opencode | no sanitizer: the AI SDK route converts schemas itself | AI SDK providers | U5 |
| kilocode | regex `pattern`s with lookaround dropped, and `patternProperties` keys with lookaround; a schema changed under `contains`/`if`/`not`/`oneOf`, or that is dynamic or holds a `$ref` there, replaced by `{type: object, additionalProperties: true}` | every provider: a session's tools (U16) | U6 |
| kilocode | OpenAI: boolean schemas become `{type: string}`; `const` becomes `enum`; a missing `type` is inferred from its keywords; an object gets `properties: {}`, an array `items: {type: string}`; unknown keywords dropped; `anyOf`/`oneOf`/`allOf`, `$ref` and definitions kept ("Mirrors Codex's Rust JSON schema compatibility lowering") | `@ai-sdk/openai` and `@ai-sdk/azure` | U7 |
| kilocode | Moonshot: `$ref` siblings dropped; tuple `items` made their first | Moonshot and model ids holding `kimi` | U8 |
| kilocode | Gemini: enum values made strings, a numeric enum's type made string; a type array made an `anyOf` plus `nullable`; `required` filtered to properties; array `items` defaulted; `properties`/`required` dropped from non-objects | Google, and model ids holding `gemini` | U9 |
| kilocode | in its newer `packages/llm` protocols: a root `anyOf` flattened, its branches' properties merged into one object with `additionalProperties: false`; `null` branches removed | its OpenAI Chat and Responses lowering; which runtime path sends through it was not established | U10 |
| codex | an MCP schema with no `properties`, or `null`, gets `properties: {}` | MCP tools | U11 |
| codex | the same lowering kilocode mirrors (U7), applied recursively, `anyOf` branches included; unreachable definitions pruned | every tool it parses | U12 |
| codex | a schema over 5,000 bytes is compacted: descriptions stripped, definitions dropped, deep objects collapsed, compositions pruned, while over budget | every tool parsed through its `parse_tool_input_schema`, MCP tools included | U13 |
| codex | a dynamic tool whose schema does not parse is refused at registration naming the tool, or skipped with a logged error | dynamic tools | U14, U15 |

Two inferences, not measured:

* **opencode would send the owner's schema unchanged** (it has a `type`,
  U1), and so, inferred, would meet xAI's refusal. kilocode's OpenAI path
  and codex would give each `{"required": [...]}` branch `"type":
  "object"` and `"properties": {}` (U7, U12), which, inferred, is what xAI's
  rule asks of a branch.
* **That rewrite does not change what the schema accepts.** Under a root
  that already says `"type": "object"`, a branch `{"required": [...]}` and
  the same branch with `"type": "object"` accept the same instances, since
  a root's keywords and its `anyOf` apply together (inferred from JSON
  Schema's semantics). So not every normalisation of a root union is lossy;
  dropping the union, as kilocode's `packages/llm` does (U10), is. Whether
  xAI accepts the typed branches is not established (the PLAN's step O3).

### The owner's request and what was already decided

The owner's request of 2026-11-17, in their words: "analyze, research,
assess and evaluate these: - Whether to draft SDK records for the three
other items on its next release: interfaces in place of the concrete
session-type checks, a wizard option to skip the callback listener, and the
typed "tool schema refused" error. and draft the detailed, facts-grounded
and complete madr docs for review."

Decided before this record, and not reopened here:

* **The normalisation is gobble's.** The owner chose, for gobble, "A gobble
  record, drafted after P6 commits" (relayed by the coordinator): gobble
  normalises root-level unions at load, for every provider. The
  coordinator's reasoning was that "the rewrite is lossy and so the host's
  decision"; the second inference above narrows that premise for one
  rewrite and is put to the owner as a finding, not a change of the ruling.
* **This item was queued for a record and a shared release.**
  [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D11 ships one minor release after `v1.31.0` "shared with the typed "tool
  schema refused" error's record" (R14).

### The owner's answers of 2026-11-17

The owner answered the draft's seven questions the same day, each as the
draft recommended, relayed by the coordinator. Each answer is recorded
under "Decided" below, with the question as it was put and the options not
chosen, and each decision section says what was decided. The record stays
`proposed` until the owner accepts it with its PLAN.

### Evidence index

Each row of the first table is a line read at the commit the marker below
names for its repository (mark: **read**); `tools/claims.py` checks each.
The second table holds the evidence that is not a line: the owner's paste,
pages fetched on 2026-11-17, and runs made for this record.

<!-- claims: self=b109cea gobble-cli=cecc892 mcp-server-magictools=368ac6a opencode=4ce137b39e kilocode=58072bd696 codex=57e64682f8 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| M1 | The tool's schema is an object | `mcp-server-magictools:internal/handler/inventory.go:290` | `"type": "object",` |
| M2 | with a root `anyOf` of untyped branches | `mcp-server-magictools:internal/handler/inventory.go:323` | `"anyOf": [` |
| G1 | gobble adds `type` and `properties` only when missing | `gobble-cli:mcpclient/tool.go:71` | `if m["type"] == nil {` |
| G2 | Each model call sends the turn's tools | `gobble-cli:agent/agent.go:671` | `req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.specs,` |
| G3 | A stream error ends the call | `gobble-cli:agent/agent.go:691` | `return r, err` |
| G4 | gobble copies `Code` and `Message` onto its own error | `gobble-cli:llm/provider/provider.go:764` | `Code: ae.Code, Message: ae.Message, Retryable: &retryable, Limit: limitOf(ae)}` |
| G5 | gobble already resends once on a refusal it matches by text | `gobble-cli:llm/provider/provider.go:367` | `if canRetry && !sent && signatureRefused(mapError(err)) {` |
| G6 | by substrings of the error's text | `gobble-cli:llm/provider/replay.go:123` | `var signatureMarks = []string{"signature", "encrypted content", "encrypted_content"}` |
| G7 | The schema goes to the SDK as gobble holds it | `gobble-cli:llm/provider/provider.go:597` | `out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, Schema: s.InputSchema,` |
| G8 | gobble's pin | `gobble-cli:go.mod:17` | `github.com/maccavelli/go-llmprovider-sdk v1.31.0` |
| C1 | `Tool.Schema` | `llmprovider/provider.go:42` | `Schema      any` |
| C2 | Only a nil schema is replaced | `llmprovider/internal/wire/tools.go:25` | `return map[string]any{KeyType: "object", "properties": map[string]any{}}` |
| C3 | Responses tools | `llmprovider/internal/wire/tools.go:54` | `keyParameters:  ToolSchema(tool.Schema),` |
| C4 | Messages tools | `llmprovider/internal/wire/tools.go:116` | `"input_schema": ToolSchema(tool.Schema)}` |
| C5 | Chat Completions tools | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:396` | `Parameters: wire.ToolSchema(tool.Schema)}}` |
| C6 | Gemini Interactions tools | `llmprovider/providers/gemini/gemini.go:345` | `jsonKeyParameters:  wire.ToolSchema(tool.Schema),` |
| C7 | Grok sends Responses tools, after deferral | `llmprovider/providers/grok/grok.go:367` | `wire.AddResponsesTools(body, deferral.Tools(req, form), req.ToolChoice)` |
| C8 | The one schema rewrite: a structured-outputs copy | `llmprovider/internal/wire/messages/toolchoice.go:80` | `"schema": closedSchema(schema)}` |
| C9 | Before the network, a schema must only marshal | `llmprovider/contract.go:271` | `if _, err := json.Marshal(tool.Schema); err != nil {` |
| C10 | `APIError.Code` | `llmprovider/api_error.go:112` | `Code string` |
| C11 | `APIError.Message` | `llmprovider/api_error.go:114` | `Message string` |
| C12 | The message is redacted and bounded | `llmprovider/api_error.go:272` | `Message:    boundMessage(redact.String(redact.StripControl(redactInput(message)))),` |
| C13 | xAI's `invalid-argument` is a refused key only when it names one | `llmprovider/api_error.go:559` | `case service == string(ProviderGrok) && status == http.StatusBadRequest && has("invalid-argument") &&` |
| C14 | Otherwise the status decides | `llmprovider/api_error.go:598` | `return true, statusSentinel(status)` |
| C15 | The parsed error object has no `param` | `llmprovider/api_error.go:763` | `add(jsonString(inner.Code), inner.Type, inner.Status)` |
| C16 | The redaction rule that eats the word after `..._key:` | `internal/redact/redact.go:49` | `reKV = regexp.MustCompile(` |
| C17 | `Error()` leads with the kind | `llmprovider/api_error.go:163` | `parts := []string{e.kind().Error()}` |
| C18 | `ErrContextOverflow`, a child kind of `ErrInvalidRequest` | `llmprovider/api_error.go:43` | `ErrContextOverflow error = &sentinelError{msg: "llmprovider: context window exceeded", parent: ErrInvalidRequest}` |
| C19 | classified from a per-vendor message table | `llmprovider/context_overflow.go:32` | `var contextOverflowMessages = []struct {` |
| C20 | The forced-choice refusal matcher, by status, type and text | `llmprovider/internal/wire/messages/learn.go:80` | `apiErr.Code == "invalid_request_error" && strings.Contains(apiErr.Message, "tool_choice")` |
| C21 | A send's retry hook may replace the error | `llmprovider/internal/wire/retry.go:22` | `type Retry func(err error) (resend bool, replaced error)` |
| C22 | A non-2xx is classified in `Post` | `llmprovider/internal/wire/post.go:181` | `if err := llmprovider.ClassifyHTTPError(c.Provider, resp); err != nil {` |
| C23 | Warnings on a response | `llmprovider/warning.go:45` | `type Warning struct {` |
| C24 | The wizard lists every kind, child first | `wizard/describe.go:15` | `var errorClasses = []error{` |
| C25 | `APIError.Limit`, the last field added | `llmprovider/api_error.go:131` | `Limit *Limit` |
| C26 | OpenCode's generateContent declarations | `llmprovider/providers/opencode/opencode.go:765` | `jsonKeyParameters: wire.ToolSchema(tool.Schema)}` |
| C27 | The retry hook is not consulted after the last send | `llmprovider/internal/wire/retry.go:20` | `nor once MaxSends sends have been made` |
| C28 | Grok wraps deferred tools in its own `use_tool` | `llmprovider/providers/grok/grok.go:64` | `// Deferred tools (Tool.Deferred) go through grok-build's use_tool wrapper` |
| F1 | Grok's refusal, recorded | `llmprovider/internal/wire/testdata/refusals-0062/responses/grok-string-schema-tool.json:11` | `[invalid_client_tool_schema] record_city: tool parameter root must be an object type` |
| F2 | OpenAI's refusal, recorded | `llmprovider/internal/wire/testdata/refusals-0062/responses/openai-string-schema-tool.json:10` | `"code": "invalid_function_parameters",` |
| F3 | OpenCode Zen's Responses route, recorded | `llmprovider/internal/wire/testdata/refusals-0062/responses/opencode-zen-responses-string-schema-tool.json:11` | `"code": "invalid_function_parameters",` |
| F4 | Anthropic's refusal, recorded | `llmprovider/internal/wire/testdata/refusals-0062/messages/claude-string-schema-tool.json:11` | `"message": "tools.0.custom.input_schema.type: Input should be 'object'",` |
| F5 | OpenCode Go's Messages route accepted | `llmprovider/internal/wire/testdata/refusals-0062/messages/opencode-go-messages-string-schema-tool.json:12` | `"outcome": "accepted"` |
| F6 | Gemini's Interactions accepted | `llmprovider/internal/wire/testdata/refusals-0062/interactions/gemini-string-schema-tool.json:12` | `"outcome": "accepted"` |
| F7 | OpenCode Zen's Google route, recorded | `llmprovider/internal/wire/testdata/refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json:12` | `` Unable to submit request because `record_city` functionDeclaration parameters schema should be of type OBJECT `` |
| F8 | Hugging Face accepted | `llmprovider/internal/wire/testdata/refusals-0062/chatcompletions/huggingface-string-schema-tool.json:12` | `"outcome": "accepted"` |
| F9 | xAI uses `invalid-argument` for a refused effort too | `llmprovider/providers/learn_0062_test.go:28` | `` refusedGrok        = `{"code":"invalid-argument","error":"Invalid reasoning effort."}` `` |
| F10 | OpenAI's `param` names the index | `llmprovider/internal/wire/testdata/refusals-0062/responses/openai-string-schema-tool.json:12` | `\"param\": \"tools[0].parameters\"` |
| F11 | Kilo (`openai/gpt-6-luna`) accepted | `llmprovider/internal/wire/testdata/refusals-0062/chatcompletions/kilo-string-schema-tool.json:11` | `"outcome": "accepted"` |
| F12 | Ollama accepted | `llmprovider/internal/wire/testdata/refusals-0062/chatcompletions/ollama-string-schema-tool.json:11` | `"outcome": "accepted"` |
| F13 | OpenCode Zen's chat route accepted | `llmprovider/internal/wire/testdata/refusals-0062/chatcompletions/opencode-zen-chat-string-schema-tool.json:13` | `"outcome": "accepted"` |
| F14 | Together accepted | `llmprovider/internal/wire/testdata/refusals-0062/chatcompletions/together-string-schema-tool.json:12` | `"outcome": "accepted"` |
| R1 | R24: `*APIError` is the only structured error | `docs/guides/api-standards.md:214` | `` - **R24. `*APIError` is the only structured error.** `` |
| R2 | R25: a new failure mode maps to an existing kind first | `docs/guides/api-standards.md:221` | `A new failure mode maps to an existing kind before it` |
| R3 | R4: experimental API only under `llmprovider/x/` | `docs/guides/api-standards.md:77` | `` - **R4. Experimental API lives under `llmprovider/x/`** and nowhere else. `` |
| R4 | R45: the wire changes only by a record | `docs/guides/api-standards.md:305` | `- **R45. The request on the wire does not change** unless a record changes it.` |
| R5 | R48: the v1 promise | `docs/guides/api-standards.md:317` | `` - **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| R6 | R55: each export with its call site | `docs/guides/api-standards.md:367` | `- **R55. A new export is justified.** A PLAN that adds an exported name` |
| R7 | 0015-MADR D7: one error type, kinds by sentinel | `docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md:211` | `### D7. One error type, classified by kind` |
| R8 | 0015-MADR's overflow amendment: a child kind, a per-vendor table, because the caller's right response differs | `docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md:470` | `because a caller's right response to overflow is to` |
| R9 | 0034-MADR D8's amendment: the tool definition stays the caller's | `docs/decisions/0034-MADR-degrade-forced-tool-choice-per-model.md:351` | `` The tool definition in `tools` stays `` |
| R10 | 0034-MADR D3: a match on text only decides a resend | `docs/decisions/0034-MADR-degrade-forced-tool-choice-per-model.md:208` | `The match is narrower than a typed class, because the API gives no other` |
| R11 | 0064-MADR D3: two child sentinels of `ErrInvalidRequest` from a type and a fixed phrase | `docs/decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md:159` | `` Two sentinels, each a child of `ErrInvalidRequest` `` |
| R12 | 0062-MADR F6: no field holds `param` | `docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md:88` | `` No field holds an error's `param`. `` |
| R13 | The fixtures' method and table | `docs/reports/0062-REPORT-thinking-refusals.md:204` | `` `grok-string-schema-tool` `` |
| R14 | 0080-MADR D11: the shared release | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:393` | `shared with the typed "tool schema refused" error's record` |
| R15 | 0062-MADR D4: one matcher per wire, tested on every recorded refusal and every other 400 | `docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md:194` | `**Each wire package has one matcher.**` |
| R16 | 0040-MADR D7: one advisory warning | `docs/decisions/0040-MADR-native-tool-deferral.md:439` | `### D7. One warning, no capability` |
| R17 | 0072-MADR D2: one nil-able field for what the service stated | `docs/decisions/0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md:324` | `` ### D2. `APIError.Limit *Limit` `` |
| U1 | opencode's object root, every provider | `opencode:packages/ai/src/protocols/utils/tool-schema.ts:73` | `if (schema.type !== undefined) return schema` |
| U2 | opencode's sanitizer choice | `opencode:packages/ai/src/protocols/utils/tool-schema.ts:86` | `const modelCompatibility = (schema: JsonSchema, model: LanguageModel): JsonSchema => {` |
| U3 | opencode's Gemini rules | `opencode:packages/ai/src/protocols/utils/gemini-json-schema.ts:28` | `if (key === "required" && properties && Array.isArray(value))` |
| U4 | opencode's Moonshot rules | `opencode:packages/ai/src/protocols/utils/tool-schema.ts:27` | `if (typeof schema.$ref === "string") return { $ref: schema.$ref }` |
| U5 | opencode's AI SDK route has none | `opencode:packages/core/src/aisdk.ts:384` | `sanitizer: "none",` |
| U6 | kilocode's lookaround and fallback rule | `kilocode:packages/opencode/src/kilocode/session/tool-schema.ts:136` | `const schema = fallback ? { type: "object" as const, additionalProperties: true } : result.value` |
| U7 | kilocode's OpenAI lowering | `kilocode:packages/opencode/src/provider/transform.ts:1921` | `model.api.npm === "@ai-sdk/azure"` |
| U8 | kilocode's Moonshot rule | `kilocode:packages/opencode/src/provider/transform.ts:1926` | `model.providerID === "moonshotai"` |
| U9 | kilocode's Gemini rules | `kilocode:packages/opencode/src/provider/transform.ts:1945` | `model.providerID === "google"` |
| U10 | kilocode's root-union flattening | `kilocode:packages/llm/src/protocols/utils/tool-schema.ts:48` | `const openAI = (schema: JsonSchema): JsonSchema => {` |
| U11 | codex adds `properties` to an MCP schema | `codex:codex-rs/tools/src/mcp_tool.rs:48` | `if let serde_json::Value::Object(obj) = &mut serialized_input_schema` |
| U12 | codex's lowering | `codex:codex-rs/tools/src/json_schema.rs:91` | `fn sanitize_json_schema(value: &mut JsonValue) {` |
| U13 | codex's 5,000-byte compaction | `codex:codex-rs/tools/src/json_schema/compaction.rs:15` | `pub(crate) const DEFAULT_COMPACT_TOOL_SCHEMA_BYTES: usize = 5_000;` |
| U14 | codex refuses a dynamic tool's schema at registration | `codex:codex-rs/app-server/src/request_processors/thread_processor.rs:439` | `"dynamic tool input schema is not supported for {name}: {err}"` |
| U15 | or skips it with an error log | `codex:codex-rs/core/src/tools/spec_plan.rs:1639` | `"Failed to convert dynamic tool {:?} to OpenAI tool",` |
| U16 | kilocode applies U6 to a session's tools | `kilocode:packages/opencode/src/session/llm.ts:167` | `KiloToolSchema.sanitize(base.tools)` |

| Id | Claim | Source | Text or result | Mark |
| :--- | :--- | :--- | :--- | :--- |
| P1 | The failure | the owner's paste, 2026-11-16 | quoted above | the owner's |
| U0 | No upstream agent matches a schema refusal | `git grep` in all three checkouts, 2026-11-17 | no hit | run |
| X1 | What `ClassifyHTTPError` makes of each recorded refusal and of the owner's message | a scratch test on an archive of `b109cea`, nothing in the tree | quoted in "What a caller can read today" | run |
| W1 | xAI: root rules, the 400 that names the tool, 350 tools | <https://docs.x.ai/docs/guides/function-calling>, "Parameter Schema" | quoted above | fetched |
| W2 | OpenAI: strict root is an object, not `anyOf` | <https://developers.openai.com/api/docs/guides/structured-outputs>, "Root objects must not be `anyOf` and must be an object" | "Note that the root level object of a schema must be an object, and not use `anyOf`." | fetched |
| W3 | OpenAI: Responses falls back to non-strict | <https://developers.openai.com/api/docs/guides/function-calling> | "Responses requests will attempt to normalize your schema into strict mode when possible, and will fall back to non-strict, best-effort function calling if the schema cannot be made compatible with strict mode." | fetched |
| W4 | Anthropic: `input_schema` | <https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools> | "A JSON Schema object defining the expected parameters for the tool." | fetched |
| W5 | Anthropic: error values can expand; do not string-match | <https://platform.claude.com/docs/en/api/errors>, "Error shapes", "SDK error types" | "the values within these objects can expand, and new `type` values can appear"; "Catch the SDK's typed classes rather than string-matching error messages" | fetched |
| W6 | Google: `parametersJsonSchema` describes an object | <https://docs.cloud.google.com/gemini-enterprise-agent-platform/reference/rest/Shared.Types/FunctionDeclaration> | "The schema must describe an object where the properties are the parameters to the function." | fetched |
| W7 | Google: complex tools refused with 400 | <https://aistudio.google.com/docs/function-calling> | "the API rejects the request with a `400 INVALID_ARGUMENT` error." | fetched |

Each page was fetched on 2026-11-17 at the URL given. OpenAI's
structured-outputs page was reached through a 301 from
`platform.openai.com`, Anthropic's tool page through a 301 from
`docs.anthropic.com` (it now lives at the "define tools" URL above), and
Google's function-calling page through a 301 from `ai.google.dev`.

## Decision Drivers

* **A host can recover from what it cannot foresee.** gobble's
  normalisation (the owner's ruling) covers root unions; it cannot cover a
  rule nobody has met yet, such as a union below the root or another
  provider's limit, and one refused schema fails every turn (G2, G3).
* **What is typed is reliable.** A field set from free text is only as good
  as the parse; a host must never drop a tool the provider did not name.
* **The v1 promise** (R5): what is added is kept; R25 asks that a new kind
  be earned (R2), and R24 that `*APIError` stay the only structured error
  (R1).
* **The request on the wire is the caller's** (R4): the SDK has never
  rewritten a tool definition in `tools` (R9).
* **Provider messages change** (W5): each parse costs a recorded fixture, a
  test against every other recorded 400, and a maintenance path.
* **The least new API, with a call site** (R6).
* **Precedent:** `ErrContextOverflow` (R8, C18, C19) and the workspace
  sentinels (R11) were added for the same reason, a different right
  response; 0034 and 0062 match provider text internally to choose a resend
  (R10, R15, C20).

## Considered Options

* **A. A typed refusal (the item as named).** A kind sentinel
  `llmprovider.ErrToolSchemaRefused`, message `"llmprovider: tool schema
  refused"`, a child of `ErrInvalidRequest` like `ErrContextOverflow` (C18),
  terminal; and one new field, `APIError.Tool string`, the name of the tool
  the service refused, exactly as the request sent it, set only when the
  service's refusal identifies one tool the request sent. Recognised per
  route from recorded refusals (D3's table); every other 400 is unchanged.
* **B. Per-provider adaptation inside the SDK.** Before sending, each
  provider rewrites what its service is known to refuse (for Grok: give each
  root-union branch `"type": "object"`; for others, the upstream rules
  above), and puts a `WarningToolSchemaAdapted` on the response naming the
  tool, as 0034 and 0040 warn (C23, R16).
* **C. Declared provider limits.** An exported table or function, such as
  `llmprovider.ToolSchemaRules(provider ProviderID, model string)
  []SchemaRule` or `CheckToolSchema(provider, model string, schema any)
  error`, that a host calls to validate or normalise before sending; the
  SDK owns the knowledge, the host the rewrite.
* **D. General structured error detail.** `APIError.Param string`, the
  service's `param` (OpenAI's `tools[0].parameters`), alongside `Code`, for
  every 4xx; no tool-schema concept; hosts match on it.
* **E. Do nothing.** Hosts read `Code` and `Message`, as gobble already
  does for thinking signatures (G5, G6).
* **F. A refusal-and-drop inside the SDK.** On a recognised refusal the
  provider drops the named tool, resends once per refusal through the
  existing resend hook (C21), and warns, as 0062 learns and resends (R15).

Two narrower choices sit across A:

* **Where the name comes from:** parsed from the message (Grok, OpenAI,
  OpenCode's Google route), from `param` (OpenAI), or from an index mapped
  onto the list the provider sent (Anthropic, OpenAI).
* **Where it is exported:** under the v1 promise in `llmprovider`, or as
  an experimental helper in `llmprovider/x/` (R3).

## Decision Outcome

Chosen option: **"A", narrowed to the refusals this repository has
recorded, with the name filled only from the tools the provider sent; not
B, C, D or F; E stays every caller's fallback**, because it gives a host a
reliable, typed signal for the one failure that fails every turn, and its
parse is no more brittle than the overflow table this SDK already keeps
(C19), while B, C and F would each make the SDK decide or predict something
the owner has placed with the host, or that the provider table cannot
support. The owner decided each choice below on 2026-11-17 (Q1 to Q7, each
as the draft recommended; "Decided" below). The record stays `proposed`
until the owner accepts it with its PLAN.

### D1. The kind (Q1, Q2, decided)

* `var ErrToolSchemaRefused error = &sentinelError{msg: "llmprovider: tool
  schema refused", parent: ErrInvalidRequest}`, in `llmprovider`, under the
  v1 promise.
* Terminal, as every `ErrInvalidRequest` is: retrying the same request
  cannot succeed. `Retryable()` is false; `errors.Is(err,
  ErrInvalidRequest)` stays true, so every existing check holds.
* R25 is earned the way `ErrContextOverflow` earned it (R8): a host's right
  response differs from any other invalid request's. It sends the same
  conversation without one tool. The standards guide's R25 gains the kind.

### D2. The tool (Q1, Q4, decided)

* `APIError.Tool string`, in `llmprovider`, under the v1 promise: "Tool is
  the name of the tool whose schema the service refused, exactly as the
  request sent it; empty when the service named none, named several, named
  one the request did not send, or the kind is not `ErrToolSchemaRefused`.
  `Error` does not print it."
* It is set only to the name of a caller's tool that the request's body
  held, after deferral (C7, C28): no text from the response reaches it. A
  parse that misreads the message yields no name, never a wrong one; a
  deferred tool that was not sent, and the SDK's own `use_tool` wrapper,
  are never named.
* When the service names no tool, or several, the kind is still
  `ErrToolSchemaRefused` and `Tool` is empty: a host knows the class and
  never drops a tool on a guess.

### D3. Which refusals, and how each is read (Q3, decided)

Every route with a recorded refusal (F1 to F4, F7, and P1's form). The kind
is set where `classifyAPIError` sets the others (C13, R11); the name is
filled where the provider returns the error (D4).

| Route | Kind when | Name from | Brittleness |
| :--- | :--- | :--- | :--- |
| Grok, Responses | 400, code `invalid-argument`, message contains `[invalid_client_tool_schema]` | the token after the tag, up to `:`, matched against the sent names | medium: a bracketed machine tag xAI documents as naming the tool (W1), but neither the tag nor the word order is documented |
| OpenAI, Responses; OpenCode Zen Responses route | 400, code `invalid_function_parameters` | `param` `tools[N].parameters`, N the index into the sent list; else the message's `function '<name>'` | low: a stable-looking code and a structured `param`; needs `param` read from the body (C15), kept unexported |
| Anthropic, Messages | 400, type `invalid_request_error`, message begins `tools.N.` and contains `input_schema` | N, the index into the sent list | high, and the more likely to go stale (the owner's note on Q3): a validation path in free text, and Anthropic advises against string-matching (W5); the index needs the list as sent, after deferral |
| OpenCode Zen, Google generateContent route | 400, status `INVALID_ARGUMENT`, message contains `functionDeclaration parameters schema` | the backticked name, matched against the sent names | high, and the more likely to go stale (the owner's note on Q3): free text, wrapped by the gateway |

* **When a provider rewords a refusal, the error stays a plain invalid
  request.** A message that no longer matches falls back to
  `ErrInvalidRequest` with `Tool` empty, exactly as today. The PLAN holds
  that with a test per route, from the recorded fixture with its marker
  changed, so a fall-back is a known state, not an accident.
* **Not read, so unchanged** (`ErrInvalidRequest`): the Chat Completions
  servers, OpenCode Go's Messages route and Gemini's Interactions, which
  accepted a root string (F5, F6, F8, F11 to F14), and the ChatGPT plan
  session and `chatcompat`, not recorded. A refusal recorded later is added
  by amendment, with its fixture, as the overflow table grows (C19).
* *(Annotated 2026-11-20 by the owner's rulings relayed by the
  coordinator, recorded in the second amendment of 2026-11-20: on OpenAI's
  routes the name comes from the message's `function '<name>'` only, never
  `param` (N2); each row applies on every route of its service whose reply
  carries the marker, so OpenAI's ChatGPT and plan sessions and OpenCode's
  chat and Messages routes are read where they send a read form, and
  OpenCode Zen's Messages route is recorded (N3); the rows apply to HTTP
  error replies only (N4); `x-should-retry: true` and the overflow row win
  over them (N5).)*

### D4. Where the name is filled

* Where each provider returns a send's error: after `wire.Send` returns,
  and where a stream yields its error before any event. The provider holds
  the list it encoded, in the order and form sent after deferral (C7).
  *(Refined 2026-11-17, while drafting the PLAN: the draft also offered the
  send's resend hook (C21), which is not consulted once the last allowed
  send has been made (C27), so a refusal on that send would carry no name.
  The PLAN tests that case.)*
* `ClassifyHTTPError` never sees the request, so it sets the kind and
  leaves `Tool` empty. `Message`, `Code`, `Status` and `Retryable()` are
  unchanged.
* *(Annotated 2026-11-20 by the owner's rulings relayed by the
  coordinator, recorded in the second amendment of 2026-11-20: the
  provider finds the `*APIError` through wrapping (N6); "the list it
  encoded" includes the tool definitions inside a `tool_search_output`
  item of `input` and the caller's Search tool behind a native
  `tool_search` entry (N2, and the ruling on ST10); several entries of one
  name count as one name (N6).)*

### D5. The SDK adapts nothing and resends nothing (Q5, decided)

No rewrite of a tool definition (R4, R9), no drop, no resend, no warning.
The host decides, per the owner's gobble ruling. A later record may revisit
a provider-specific, semantics-preserving rewrite once the PLAN's step O3
shows whether xAI accepts typed branches.

### D6. The error's text (Q6, decided)

`Error()` reads "llmprovider: tool schema refused: <provider> HTTP 400
<code>: <message>", the kind's text in place of "llmprovider: invalid
request" (C17); the rest as today. `Tool` is not printed, as `Limit` and
`ResetAt` are not (C25).

### D7. The exports and their call sites (R55)

* `llmprovider.ErrToolSchemaRefused`; `llmprovider.APIError.Tool`. Nothing
  else is exported; the per-route matchers stay internal, as 0062's are
  (R15). `make api-check` lists exactly these two additions.
  *(Annotated 2026-11-20: `make api-check` lists incompatible changes
  only (`scripts/check_api.py:85`), so it lists neither; the check is
  `make api-check` exiting 0 and `tools/apidiff.py` from the PLAN's Phase 2
  start commit listing exactly these two.)*
* Call site: gobble-cli's `llm/provider` adapter, where it maps an
  `*APIError` (G4) and where it already resends once on a matched refusal
  (G5), by a gobble-cli record of its own.

### D8. Docs

In the same commit as the code: the standards guide's R24 (the field) and
R25 (the kind); `docs/architecture.md`'s error paragraphs; `CHANGELOG.md`
`[Unreleased]`; the package docs of `llmprovider` (the kind and the field),
`grok`, `openai`, `claude` and `opencode` (which refusals each reads); and
the wizard's `errorClasses` gains the kind before `ErrInvalidRequest`
(C24), as 0064 did for its two.

### D9. The release (Q7, decided)

The next release, together with 0080's and the other two SDK records the
owner answered on 2026-11-17, on the owner's word. This record's PLAN must
finish and pass its gates before the tag; if it is not ready when the owner
gives the word, it moves to the release after. The tag and the GitHub
Release are each the owner's word.

### D10. gobble afterwards

By a gobble-cli record of its own, after the pin bump (G8): on
`ErrToolSchemaRefused` with `Tool` set, before any event, drop that tool
for the rest of the session on that provider and model, send once more, and
tell the user which tool was left out and why; with `Tool` empty, end the
turn naming the class. Bounded: at most one drop per direct tool per turn.
The kind is tested before gobble's existing signature match (G5, G6). It
sits behind gobble's own load-time normalisation, which the owner has
already chosen, and catches what that normalisation does not know.

### Consequences

* Good, because the owner's failure becomes recoverable in gobble without
  a text parse there: the kind says what failed, the name says what to
  drop.
* Good, because the name is never a guess: only a sent tool's name is
  reported.
* Good, because every existing `ErrInvalidRequest` check, gobble's
  included, matches as before.
* Good, because the wire is unchanged (the standards guide's R45; R4).
* Neutral, because only five routes, in four forms, are read in the first
  release; the rest stay `ErrInvalidRequest` until a refusal is recorded.
* Bad, because each matcher reads provider text that can change without
  notice (W5); a change makes a refusal fall back to `ErrInvalidRequest`,
  which the staleness tests hold as a known state but cannot detect live.
* Bad, because two exports enter the v1 promise for a failure whose forms
  are known on five routes.
* Bad, because `Error()`'s leading text changes for these refusals, which a
  caller matching the text "invalid request" would notice.
* Bad, because a provider that reports one bad tool per response costs a
  host one round trip per bad tool (not established which providers do;
  the PLAN's step O2).

### Confirmation

The PLAN
([0084-PLAN-typed-error-for-a-refused-tool-schema.md](0084-PLAN-typed-error-for-a-refused-tool-schema.md))
holds the steps. In short:

* **Tests from recorded responses only**, each seen failing first at the
  base (`tools/redgreen.py --expect fail`) and each proven by a planted
  break on a scratch clone (`tools/plant.py`): every row of D3 gives the
  kind and the right `Tool`; a name in the provider's text that the request
  did not send, or did not send after deferral, is never returned; a
  reworded refusal stays `ErrInvalidRequest`, per route; every other
  recorded 400 keeps its kind; a refusal on the last allowed send keeps its
  name; `llmtest.Run` passes for every provider.
* **`make api-check`** lists exactly D7's two additions. *(Annotated
  2026-11-20: as D7's annotation of that date says.)*
* **The owner's live steps** (the PLAN's Phase 1, O1 to O6) record the forms
  this record could not establish, with no key displayed. *(Annotated
  2026-11-20: an agent runs them, with the keys in its process environment,
  by the owner's ruling "API keys from the environment only
  (Recommended)", relayed by the coordinator; second amendment, ST77 to
  ST85.)*
* **An independent review** of the change, and the gates through the
  session's heavy-run queue.

## Pros and Cons of the Options

### A. A typed refusal

* Good, because a host gets `errors.Is` and a reliable name for every
  recorded refusal, Anthropic's index included, which only the SDK can map
  (it alone knows the list as sent after deferral, C7).
* Good, because it follows `ErrContextOverflow` (R8) and the workspace
  sentinels (R11) in form and justification.
* Neutral, because its reach is the routes recorded: five today, in four
  forms.
* Bad, because the parses are text, brittle in proportion to D3's last
  column, and a fall-back when a provider rewords.
* Bad, because it adds two names under the v1 promise.

### B. Per-provider adaptation inside the SDK

* Good, because the request succeeds the first time, with no round trip.
* Good, because the one rewrite the owner's case needs, typing root-union
  branches under an object root, is semantics-preserving (inferred).
* Bad, because it changes the wire (R4) and the tool definition, which the
  SDK has never done (R9), and the owner placed normalisation with the
  host.
* Bad, because the table shows refusal varies by route, not provider
  (F2/F11, F6/F7), so a per-provider rule set is either too wide or wrong.
* Bad, because the upstream rules (U1 to U16) are each other projects'
  guesses, most not measured here; and it fixes only rules known in
  advance, so a host still needs A or E for the rest.

### C. Declared provider limits

* Good, because a host could check before sending, at load.
* Bad, because the limits are mostly not established: no provider documents
  a complete subset (W1 to W7), and the fixtures show acceptance by route.
  A table would promise what nobody measured.
* Bad, because it is the largest new surface of the six, and every entry is
  a maintenance promise under v1.

### D. General structured error detail

* Good, because it is generic: no tool concept in the SDK.
* Bad, because only OpenAI sends `param` (F10); Grok, Anthropic and Google
  do not (F1, F4, F7), so the owner's case gets nothing from it.
* Bad, because OpenAI's `param` is an index into what was sent, which a host
  cannot map after deferral.

### E. Do nothing

* Good, because it costs nothing, and gobble can match Grok's text today:
  `Message` holds `[invalid_client_tool_schema] <name>:` (X1, G4).
* Bad, because every host re-learns each provider's text, cannot map
  Anthropic's index (C7), and sees the redacted message, not the body (C12).
* Bad, because a text match on `err.Error()` is what gobble's signature
  check already does (G6); a tool named `verify_signature` with a refused
  schema would match that check first (inferred).

### F. A refusal-and-drop inside the SDK

* Good, because every host gets recovery with no code.
* Bad, because dropping a tool changes what the model can do, which is the
  host's decision and its user's to be told; a warning on a response the
  host may not read is not telling the user.
* Bad, because history may hold calls to the dropped tool, and a forced
  choice or the deferral search may name it; whether providers accept that
  is not established (the PLAN's step O5).

## Decided

The draft's seven questions, in the order they were put, with the owner's
answer to each, relayed by the coordinator on 2026-11-17. Each label is
quoted exactly; what was asked is beside it in this record's words.

* **Q1 (D1, D2). A sentinel, a field, or both?** Asked: a new kind and the
  tool's name; the kind only, with the name left in `Message`; or the name
  only, on a plain invalid request.
  *Decided 2026-11-17: **"A typed error kind plus the tool's name
  (Recommended)"**. Not chosen: the kind only; the name only.*
* **Q2 (D1, D2, D7). Under the v1 promise, or experimental first?** Asked:
  both names in `llmprovider` under v1; an experimental helper under
  `llmprovider/x/`, nothing under v1; or nothing exported.
  *Decided 2026-11-17: **"Stable, in `llmprovider` (Recommended)"**. Not
  chosen: an experimental helper under `x/`; nothing exported.*
* **Q3 (D3). Which routes in the first release?** Asked: every route with
  a recorded refusal, that is Grok, OpenAI's Responses API, OpenCode Zen's
  Responses route, Anthropic and OpenCode Zen's Google route, with the
  Google-route and Anthropic matchers named as the most likely to go stale;
  Grok only; or Grok and OpenAI only.
  *Decided 2026-11-17: **"Every route with a recorded refusal
  (Recommended)"**. Not chosen: Grok only; Grok and OpenAI only.*
* **Q4 (D2). When the service names no tool, or several?** Asked: the new
  kind with `Tool` empty; a plain invalid request unless one sent tool is
  named; or the first name of several.
  *Decided 2026-11-17: **"The new kind, with the tool left empty
  (Recommended)"**. Not chosen: a plain invalid request; the first name.*
* **Q5 (D5). Does the SDK ever adapt a schema itself?** Asked: not in this
  record, revisited after the live probe of typed branches; one
  semantics-preserving rewrite for Grok after that probe; or per-provider
  adaptation as the upstream agents do.
  *Decided 2026-11-17: **"No; revisit after the live typed-branch probe
  (Recommended)"**. Not chosen: the one Grok rewrite; per-provider
  adaptation.*
* **Q6 (D6). What the error's text shows.** Asked: the new kind's text
  leading, the tool's name not printed; the same with the name printed; or
  the old "invalid request" text kept.
  *Decided 2026-11-17: **"\"llmprovider: tool schema refused\", the tool
  name not printed (Recommended)"**. Not chosen: the name printed too; the
  old text kept.*
* **Q7 (D9). The release.** Asked of the four SDK drafts together: all four
  in the next release with 0080, on the owner's word, each record's PLAN
  finished and its gates passed before the tag, and one not ready when the
  owner gives the word moving to the release after; or other orders.
  *Decided 2026-11-17: **"All four in the next release with 0080, on my
  word (Recommended)"**. The options not chosen were put to the owner by
  the coordinator across the four drafts and are not recorded here.*

* **Ruling, 2026-11-17: accepted.** Accepted 2026-11-17 by the owner's ruling,
  relayed by the coordinator: "Accept all five and run each plan up to its
  live step or release (Recommended)". The option's description, as put with
  it by the coordinator (the coordinator's words, not the owner's own): "Each
  is marked accepted and built in dependency order, one at a time in the SDK
  tree, each with its independent review; 0081 and 0084 stop at the probes
  only you can run; nothing is tagged without your word. They are committed
  and pushed under your standing approval as each phase passes."

## More Information

### The gobble record to come, and how this sits with it

The owner's gobble ruling (normalise root unions at load, for every
provider) and this record are complementary, not redundant:

* gobble's normalisation prevents the one known refusal before it is sent,
  on every provider; this record lets gobble recover from refusals nobody
  has catalogued, on the routes recorded.
* Neither changes the other's choice. One finding is offered to the gobble
  record: typing each root-union branch as an object, under a root that is
  already an object, is semantics-preserving (inferred), so that rewrite
  need not be treated as lossy; dropping the union is. The PLAN's step O3
  measures whether xAI accepts it.
* gobble's existing text match for thinking signatures (G5, G6) would also
  catch a schema refusal whose message holds "signature"; the gobble record
  tests the kind first (D10).

### Related records

* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
  D7 and its overflow amendment (R7, R8): the kinds and how a new one is
  earned.
* [0064-MADR-anthropic-workspace-for-unscoped-keys.md](0064-MADR-anthropic-workspace-for-unscoped-keys.md)
  D3 (R11): child sentinels from a type and a fixed phrase.
* [0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md](0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md)
  D2 (R17): the last field added to `APIError`.
* [0032-MADR-tool-strict-and-result-error-flags.md](0032-MADR-tool-strict-and-result-error-flags.md):
  `Tool.Strict`, sent where the wire has it.
* [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md)
  D3, D7 and D8 (R9, R10): degrade and warn; match text only to resend;
  the tool definition stays the caller's.
* [0040-MADR-native-tool-deferral.md](0040-MADR-native-tool-deferral.md)
  D7 (R16): an advisory warning, no capability.
* [0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md)
  D4 to D6 and F6 (R12, R15), and
  [0062-REPORT-thinking-refusals.md](../reports/0062-REPORT-thinking-refusals.md)
  (R13), whose fixtures are this record's evidence.
* [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D11 (R14): the shared release.

### Found beyond the question asked

* xAI's `invalid-argument` covers a refused key, a refused effort and a
  refused schema (C13, F9, F1); only the message tells them apart.
* Redaction rewrites the word after a tool name ending in `key`, `token`,
  `secret`, `password` or `signature` and a colon (X1, C16). The name
  survives; a matcher that expects the word "tool" after it would not.
* `gpt-6-luna` and `gemini-3.8-flash` each accepted on one route what they
  refused on another (F2, F6, F7, F11).

### Not verified

* Every "not established" cell of the provider table, and the PLAN's steps
  O1 to O6.
* That xAI's tag and word order are stable: one model's fixture (F1) and
  the owner's paste (P1) only.
* Whether a refused request is billed: not established for any provider.
* The ChatGPT plan session's and `chatcompat` servers' forms: not recorded.
* Which runtime path of kilocode sends through `packages/llm` (U10).

## Amendment of 2026-11-20: the state table, before any code

A readiness pass, on the coordinator's brief, before Phase 1a. It changes no
decision, no owner answer and the status: the record stays `accepted`, as
the owner ruled. It adds the state table the owner now asks of every record
that changes behaviour: every case the change can meet, each `decided` (with
the place that decides it and the test that pins it), `out of scope` (with
the reason), `open` (the owner must choose; nothing below chooses for them),
or `open: awaits live evidence` (the owner's run in the PLAN's Phase 1b
settles it). Under the `madr-and-plan-writing` skill a record with an open
row would be `proposed`; the status is the owner's to change, and is not
changed here. Read at `24756a6` on `main`, read-only.

### What the PLAN's live steps are, and which are done

* **O1 to O6 are not steps.** They are six of the case shapes that one
  owner run records (the PLAN's Phase 1a step 1: `root-string`, then O1
  root union, O2 two bad tools, O3 typed branches, O4 nested union, O5 a
  history calling a tool no longer offered, O6 Grok's second model). The
  steps are the PLAN's Phase 1a (an agent writes the recorder), 1b (the
  owner runs it once), 1c (an agent turns the records into fixtures), then
  Phase 2 (tests, code, docs).
* **Done:** Phase 0 only: the pair placed (`a3002aa`) and accepted
  (`8f133db`). The prerequisite of Phase 1a, 0080-PLAN Phase 3's code, is on
  `main` (`00757ed`).
* **Not done:** Phase 1a (no `llmprovider/*0084*` file and no
  `testdata/toolschema-0084/` at `24756a6`), 1b, 1c and every later phase.

### Facts read at `24756a6` for this table

Each is a line read at `24756a6`; none is an evidence-index row, so
`tools/claims.py` does not check them.

* **H1.** A reply whose `x-should-retry` header is `true` turns any kind
  beneath `ErrInvalidRequest` into `ErrProviderUnavailable`, retryable
  (`llmprovider/api_error.go:306` to `:317`). Neither the 0062 fixtures nor
  Phase 1's records keep a response header (PLAN Phase 1a step 2).
* **H2.** A failure inside a 200 stream whose code is an HTTP status from
  400 to 599 is classified through `classifyAPIError` with that status
  (`api_error.go:441` to `:444`); the generateContent stream reader passes
  Google's numeric code there (`llmprovider/internal/wire/generatecontent/generatecontent.go:307`).
  So a D3 row in `classifyAPIError` also meets an in-stream failure on
  OpenCode Zen's Google route.
* **H3.** OpenCode's `Generate` sends every route under the label
  `opencode` (`llmprovider/providers/opencode/opencode.go:322`, `:334`), and
  `serviceOf` reduces any `opencode…/<route>` label to `opencode`
  (`api_error.go:498` to `:504`): `classifyAPIError` cannot tell OpenCode's
  routes apart.
* **H4.** OpenAI's API-key, ChatGPT-session and plan-session modes all send
  under `openai` (`llmprovider/providers/openai/openai.go:317`, `:333`).
* **H5.** The PLAN puts the naming helper in `llmprovider/internal/wire`,
  which can read only `APIError`'s exported fields; the envelope
  `ClassifyHTTPError` parses is not kept on the error (`api_error.go:259` to
  `:277`). The PLAN's `param`, "kept on the envelope, never exported"
  (Phase 2 step 5), cannot reach that helper as written.
* **H6.** `validate` refuses a tool with no name and a schema that does not
  marshal, not two tools of one name (`llmprovider/contract.go:265` to
  `:274`).
* **H7.** `Reauth` wraps an error (`errors.Join`) only after a refused
  credential (`llmprovider/internal/wire/reauth.go:27` to `:52`); a
  learner's check returns the error as it came unless its matcher read it
  (`learn.go:300` to `:324`); Grok's matcher reads only two effort messages
  under `invalid-argument` (`responses/refusal.go:49` to `:55`). So today
  a schema refusal reaches the provider unwrapped.
* **H8.** No log call on the send path takes an error's text:
  `post.go:115` logs a failed body close only.

### The state table

`Test` names the PLAN's test and phase, a test this amendment proposes
(marked *proposed*; the PLAN's amendment of 2026-11-20 gives each its
plant), or the existing test that must stay green unchanged.

#### Providers and routes

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST01 | Grok Responses: 400, `invalid-argument`, `[invalid_client_tool_schema]` | decided | kind `ErrToolSchemaRefused`; `Tool` is the sent name after the tag, up to `:` | D3 row 1; Q3 "Every route with a recorded refusal (Recommended)" | `Classify`, `Names` (Phase 2) |
| ST02 | OpenAI Responses (API key): 400, `invalid_function_parameters` | decided | the kind | D3 row 2; Q3 | `Classify`, `Names` (Phase 2) |
| ST03 | Where OpenAI's and OpenCode Zen Responses' `Tool` comes from: `param`'s index, or the message's `function '<name>'` | open (N2) | today nothing names it; the PLAN's `param` cannot reach its helper (H5) | – | depends on N2 |
| ST04 | OpenCode Zen Responses route: as ST02 | decided | the kind | D3 row 2; Q3 | `Classify`, `Names` (Phase 2) |
| ST05 | Anthropic Messages: 400, `invalid_request_error`, message begins `tools.N.` and holds `input_schema` | decided | the kind; `Tool` is entry N of the list as sent, after deferral | D3 row 3; D4; Q3 | `Classify`, `Names`, `Deferral` (Phase 2) |
| ST06 | OpenCode Zen Google route, behind the gateway's `Upstream request failed: [400]`: `INVALID_ARGUMENT` and `functionDeclaration parameters schema` | decided | the kind; `Tool` is the backticked sent name | D3 row 4; Q3 | `Classify`, `Names` (Phase 2) |
| ST07 | Left out, own service label: Gemini Interactions, Kilo, Hugging Face, Ollama, Together, `chatcompat` | decided | unchanged: `ErrInvalidRequest`; reason: each accepted a root string (F6, F8, F11, F12, F14) or is not recorded | D3 "Not read, so unchanged" | `OthersUnchanged` (Phase 2) |
| ST08 | Left out but sharing a read route's label: OpenAI's ChatGPT session and plan session (`openai`, H4); OpenCode's chat routes and Go Messages route (`opencode`, H3) | open (N3) | today: a D3 row placed in `classifyAPIError` matches them too, though D3 says "not read" | – | depends on N3 |

#### What the refusal says

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST09 | Names exactly one tool the body held | decided | `Tool` is its name, exactly as sent | D2; Q1 "A typed error kind plus the tool's name (Recommended)" | `Names` (Phase 2) |
| ST10 | Names a schema field only, or a name the request did not send, a deferred tool not sent, Grok's `use_tool`, an index past the list, a native `tool_search` entry, or a request that sent no tools | decided | the kind; `Tool` empty | D2; Q4 "The new kind, with the tool left empty (Recommended)" | `NeverNamesUnsent` (Phase 2), with a no-tools subcase *proposed* |
| ST11 | The route's marker, and no tool named at all | decided | the kind; `Tool` empty | D2; Q4 | *proposed* `NamesNothing` (Phase 2) |
| ST12 | Several tools named in one reply | decided | the kind; `Tool` empty | D2; Q4 | `SeveralNamed` if O2 records it; otherwise open (N7) |
| ST13 | Two sent tools share one name, and the refusal names it | open (N6) | today no check refuses duplicates (H6); the PLAN's helper wants "exactly one entry" | – | depends on N6 |
| ST14 | A reworded refusal: the route's marker gone | decided | `ErrInvalidRequest`; `Tool` empty | D3 "When a provider rewords a refusal" | `Reworded` (Phase 2) |

#### How the refusal arrives

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST15 | HTTP 400, `Generate` | decided | kind from `ClassifyHTTPError`; `Tool` set after `wire.Send` returns | D3; D4 | `Classify`, `Names` (Phase 2) |
| ST16 | HTTP 400, `Stream`, before any event | decided | the same, set where the stream yields its error | D4 | `Names`, `Stream` half (Phase 2) |
| ST17 | A 200 with an error body, or an error event inside a 200 stream, before or after other events | open (N4) | out of scope in the PLAN, yet H2 lets the Google-route row match there | – | depends on N4 |
| ST18 | A gateway in front of the provider: OpenCode Zen's Google route | decided | read through the gateway's wrapper | D3 row 4 | `Classify`, `Names` (Phase 2) |
| ST19 | A caller's proxy (`WithBaseURL`) passing status and body through | decided | as the provider itself: the rows read status, code and message, not the host | D3 | `Names` (its in-process server is such a proxy) |
| ST20 | A proxy that changes the status (502, 200) or replaces the body (HTML) | decided | no D3 row matches (each needs a 400 and its marker); the status's kind as today | D3 | *proposed* `ProxyAltered` (Phase 2; needs N7 (a)) |

#### Errors that look like it

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST21 | Grok 400 `invalid-argument` naming an API key, even with the tag | decided | `ErrAuthFailure` | PLAN Phase 2 step 5 ("after the Grok key row"); C13 | `OthersUnchanged`; *proposed* `KeyRowFirst` (Phase 2) |
| ST22 | Grok's refused effort | decided | `ErrInvalidRequest`; 0062's learner resends as today | D3; 0062-MADR D4 (R15) | `OthersUnchanged` (Phase 2) |
| ST23 | 401, 403, 402, 429, 5xx carrying a refusal's body | decided | the status's kind as today | D3 (each row needs a 400) | *proposed* `ProxyAltered` (needs N7 (a)) |
| ST24 | Content policy: OpenAI `invalid_prompt`, others' own codes | decided | `ErrInvalidRequest` | D3 | `Reworded`'s OpenAI case, its code changed to `invalid_prompt` (*proposed* detail) |
| ST25 | Context overflow alone | decided | `ErrContextOverflow` | 0015-MADR (R8); D3 | `OthersUnchanged` and existing overflow tests, unchanged |
| ST26 | A body that matches both a D3 row and the overflow table | open (N5) | the PLAN places the rows only "after the Grok key row" | – | depends on N5 |
| ST27 | Anthropic's workspace refusals; OpenCode's `ModelError`; plan-session codes | decided | their kinds as today | 0064-MADR D3 (R11); D3 | existing 0064 and 0043 tests, unchanged (PLAN Phase 2 step 7) |

#### The error's fields, wrapping and callers

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST28 | The provider gives no detail beyond its text | decided | `Status`, `Code`, `Message`, `RetryAfter`, `ResetAt`, `Limit`, `Reason` as today; `Tool` empty unless named | D4; D2 | *proposed* `FieldsUnchanged` (Phase 2) |
| ST29 | `errors.Is` | decided | true for the kind and `ErrInvalidRequest`; `Unwrap` yields the kind only | D1; Q1 | `Classify` (Phase 2) |
| ST30 | `errors.As`, and a caller that compares `Kind ==` | decided | `errors.As` finds the `*APIError`; its `Kind` is the new sentinel, so `Kind == ErrInvalidRequest` is now false; R25 says to test by `errors.Is`; gobble maps kinds by `errors.Is` through a list that holds no `ErrInvalidRequest`, so its mapping is unchanged (read at its `HEAD`) | D1; R1, R2 | `Classify`, with an `errors.As` assertion *proposed* |
| ST31 | `Error()` | decided | begins `llmprovider: tool schema refused:`; `Tool` not printed; text matching "invalid request" no longer matches these | D6; Q6 | `Classify` (Phase 2) |
| ST32 | `x-should-retry: true` on such a 400 | open (N5) | today H1 makes it `ErrProviderUnavailable`, retryable | – | depends on N5 |
| ST33 | The kind arriving wrapped (`errors.Join`, `%w`) | open (N6) | unreachable today (H7); the PLAN's helper takes "a classified error" | – | depends on N6 |
| ST34 | Any other error reaching the helper: another kind, a cancelled or timed-out context, a non-`*APIError` | decided | returned unchanged | D2 ("empty when ... the kind is not `ErrToolSchemaRefused`"); PLAN Phase 2 step 5 | *proposed* `PassThrough` (Phase 2) |
| ST35 | The wizard's `describe` | decided | names "tool schema refused, HTTP 400" before "invalid request" | D8 | `TestDescribe_ToolSchemaRefused_0084` (Phase 2) |
| ST36 | Exports | decided | exactly `ErrToolSchemaRefused` and `APIError.Tool` | D7; Q2 "Stable, in `llmprovider` (Recommended)" | `make api-check`, `tools/apidiff.py` (Phase 2) |

#### Retries

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST37 | A caller's retry (`Retryable()`, `WithRetry`) | decided | not retried: `Retryable()` false | D1 | `Classify` (Phase 2) |
| ST38 | An SDK resend, drop or rewrite on the kind | decided | none: one request reaches the server, its tools as the caller gave them | D5; Q5 "No; revisit after the live typed-branch probe (Recommended)" | *proposed* `NoAdaptation` (Phase 2) |
| ST39 | A learner resend, then the schema refusal on the last allowed send | decided | `Tool` named | D4 (refinement of 2026-11-17) | `LastSendKeepsName` (Phase 2) |
| ST40 | Concurrent requests on one provider, each refused for its own tool | decided | each error names its own request's tool | PLAN Phase 2 step 5 (the list "computed once" per request) | *proposed* `Concurrent` (Phase 2, under `-race` in Phase 3) |

#### Logging and redaction

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST41 | A tool name whose next word redaction removes | decided | `Tool` named | D2; X1 | `RedactedNeighbour` (Phase 2) |
| ST42 | A tool name that redaction itself removes | decided | the kind; `Tool` empty | D2 ("A parse that misreads the message yields no name") | *proposed* `RedactedName` (Phase 2) |
| ST43 | A name beyond `Message`'s 512-byte bound, or a body over 64 KiB | decided | the kind; `Tool` empty | D2; C12 | *proposed* `NameBeyondBound` (Phase 2) |
| ST44 | Schema or request body in the error | decided | none: `Tool` holds only a sent name; `Message` redacted and bounded as today | D2; D4; C12 | `FieldsUnchanged` (*proposed*); Phase 4 review |
| ST45 | A log of the refusal | decided | the SDK logs nothing for it (H8) | D5 ("no warning"); PLAN Phase 4 step 1 | *proposed* `NoLog` (Phase 2) |
| ST46 | The recorder's files | decided | no key, header or model text; bodies redacted, request ids `req_REDACTED` | PLAN Phase 1a steps 2 to 4 | `TestToolSchemaRecord_Redacts_0084` (Phase 1a) |
| ST47 | The recorder's log lines | decided | only id, outcome and status | PLAN Phase 1a step 5 | *proposed* `TestToolSchemaRecord_LogLine_0084` (Phase 1a) |

#### Other states

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST48 | A nil schema | decided | sent as an empty object (C2), as today | D5 | existing `tools` tests, unchanged |
| ST49 | A schema that does not marshal | decided | `ErrInvalidRequest` before the network (C9), not the kind | D3 (a service's refusal only) | `TestValidate_UnmarshalableSchema`, unchanged |
| ST50 | A body that is not JSON | decided | no D3 row matches; today's kind | D3 | *proposed* `ProxyAltered` (needs N7 (a)) |
| ST51 | Interrupted mid-write | out of scope | nothing is written at run time; a recorder file cut short fails Phase 1c's fixture read, and the PLAN's one-case rerun applies | – | – |
| ST52 | Platforms | out of scope | no path differs by platform; the owner's commands are for zsh and bash on this Mac (PLAN Phase 1b) | – | – |

#### The owner's live run (Phase 1b)

Each row is settled by what the named case records; the outcomes below
assume N1 is ruled, so that no observation needs a further ruling.

| Id | Case | State | Each possible observation, and what it means for the code | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST53 | `root-string` on the five read routes | open: awaits live evidence (Phase 1b, case `<route>-root-string`) | (1) the 0062 form again: its record joins `Classify` and `Names`, no code change; (2) refused in another status, code, marker or name position, or (3) accepted now: N1 decides (today a stop) | PLAN Phase 1b table | `Classify`, `Names`, `Reworded` |
| ST54 | O1, the owner's shape, on every route | open: awaits live evidence (Phase 1b, `<route>-O1-root-union`) | (1) refused in the route's `root-string` form: joins `Classify` and `Names`; (2) accepted: recorded, not read, no change; (3) refused in another form, or on a route not read: N1 decides | PLAN Phase 1b table | `Classify`, `Names` |
| ST55 | O2, two bad tools at indices 1 and 2 | open: awaits live evidence (Phase 1b, `<route>-O2-two-bad`) | (1) one tool named: `Tool` is that one (ST09), `SeveralNamed` not built, N7 decides its test, and a host pays a round trip per bad tool; (2) both named: `Tool` empty (ST12), `SeveralNamed` built from the record; (3) marker but neither named: kind, `Tool` empty (ST11); (4) another form, such as Anthropic listing two `tools.N.` paths in one message: N1 decides | D2; Q4; PLAN Phase 1b table | `Names`, `SeveralNamed` |
| ST56 | O3, typed branches | decided | whatever is recorded, the SDK rewrites nothing; the record is evidence for gobble's normalisation and for revisiting D5 | D5; Q5 | none in this record (no SDK behaviour) |
| ST57 | O4, a union below the root | open: awaits live evidence (Phase 1b, `<route>-O4-nested-union`) | (1) accepted: no change; (2) refused in the route's O1 form: joins `Classify` and `Names`; (3) another form: N1 decides | PLAN Phase 1b table | `Classify`, `Names` |
| ST58 | O5, history calling a tool no longer offered | open: awaits live evidence (Phase 1b, `<route>-O5-history-dropped`) | (1) accepted: no SDK change; gobble's drop is safe on that route; (2) refused without a D3 marker: its record joins `OthersUnchanged` (keeps its kind); (3) refused with a D3 marker: N1 decides | D10; PLAN Phase 1b table | `OthersUnchanged` |
| ST59 | O6, Grok's second model (`grok2`) | open: awaits live evidence (Phase 1b, `grok2-*`) | (1) tag, then name, then `:`: joins `Classify` and `Names`; (2) another order: the kind with `Tool` empty (ST10), and N1 decides whether that is accepted or a stop | D2; PLAN Phase 1b table | `Classify`, `Names` |
| ST60 | A live observation unlike what the PLAN expects (ST53 to ST59's last branches) | open (N1) | the PLAN stops for a ruling per case | – | – |
| ST61 | A route whose key is set but every case comes back `other` twice (no subscription, no credit, a refused key) | open (N8) | the PLAN re-runs `other` once and says nothing more | – | – |
| ST62 | A decided row with no recorded body to test it (ST12 when O2 names one tool; ST20, ST23, ST50) | open (N7) | the PLAN lets a test alter a recorded body only to break its marker or rename its tool | – | – |

#### Sibling records

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST63 | 0080: the shared release and docs | decided | this record's entries sit beside 0080's in `CHANGELOG.md`, `docs/architecture.md` and the standards guide; Phase 1a's prerequisite is met (`00757ed`) | D9; Q7 "All four in the next release with 0080, on my word (Recommended)"; PLAN "Relation to 0080" | Phase 2 step 1's re-read; `make changelog-check docs-check` |
| ST64 | 0081, 0083, 0085 | decided | no Go file in common at `24756a6`: 0081's PLAN cites `api_error.go:225` as evidence only, and 0085's records cite `claude.go:214` and `opencode.go:238` as evidence only; the shared files are `CHANGELOG.md` `[Unreleased]` and `docs/architecture.md`, different sections; gates one at a time | PLAN "Relation to 0080" ("the other three drafts' PLANs ... for any file they share") | Phase 2 step 1's re-read |
| ST65 | 0082's code (`04433ae`, `5f2c2ab`, `1545172`, `fd986c4`, `8051b00`) | decided | no overlap: since `b109cea` no file this record cites under `llmprovider/` or `wizard/describe.go` changed but an added test file (`providers/openai/held_session_0082_test.go`) | re-grounding below | Phase 2 step 1's re-read |

Counts: 65 rows; 47 decided, 2 out of scope, 10 open (ST03, ST08, ST13,
ST17, ST26, ST32, ST33, ST60, ST61, ST62), 6 open: awaits live evidence
(ST53, ST54, ST55, ST57, ST58, ST59). ST12's outcome is decided and it is
counted decided; only its test, when O2 names one tool, waits on N7.

### Open questions for the owner (N1 to N8)

None is answered here. Each names the rows it settles. The coordinator puts
them to the owner in one prompt.

* **N1 (ST53 to ST60).** When the live run records something unlike what
  the PLAN expects, a stop and a ruling per case, or a rule now?
* **N2 (ST03).** How OpenAI's routes name the tool, given that `param`
  cannot reach a helper in `internal/wire` (H5).
* **N3 (ST08).** Whether the kind applies to routes D3 does not read that
  share a read route's service label (H3, H4).
* **N4 (ST17).** Whether a refusal in a 200 body or inside a stream is
  classified, given H2.
* **N5 (ST26, ST32).** Precedence: the overflow row, and `x-should-retry:
  true` (H1).
* **N6 (ST13, ST33).** Two tools of one name; the kind arriving wrapped.
* **N7 (ST62).** Whether a test may build a body no provider has sent.
* **N8 (ST61).** A route whose key is set but whose every case fails.

### Re-grounding at `24756a6`

`tools/claims.py` passes every row at its pin: 85 of 85 here, 29 of 29 in
the PLAN. With `self` at `24756a6` it fails seven rows here, each a line
that moved with the same text; every code row (C1 to C28, F1 to F14, P1 to
P15, P18 to P20, P24, P25) holds at `24756a6`. The rows stay pinned, as
`tools/README.md` says pins stand; the moves are:

| Row (here) | Pinned line at `b109cea` | Line at `24756a6` | Meaning |
| :--- | :--- | :--- | :--- |
| R1 (R24) | `docs/guides/api-standards.md:214` | `:215`, same text | unchanged; R24 now also lists `Reason` |
| R2 (R25) | `docs/guides/api-standards.md:221` | `:222`, same text | unchanged |
| R3 (R4) | `docs/guides/api-standards.md:77` | `:78`, same text | unchanged |
| R4 (R45) | `docs/guides/api-standards.md:305` | `:306`, same text | unchanged |
| R5 (R48) | `docs/guides/api-standards.md:317` | `:318`, same text | unchanged |
| R6 (R55) | `docs/guides/api-standards.md:367` | `:368`, same text | unchanged |
| R14 (0080 D11) | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:393` | `:516`, same text | unchanged |

gobble-cli has moved from `cecc892` to `b1994a0`: G1 to G5, G7 and G8 hold
their text on other lines (G8, the pin `v1.31.0`, now on line 16); G6
holds; M1 and M2 hold at mcp-server-magictools' `HEAD`. D10 and the PLAN's
Phase 6 name `cecc892`; gobble's own record re-reads at its start.

## Second amendment of 2026-11-20: the owner's rulings and the independent review

A second records pass, the same day, read at `24756a6`. An independent
review of the first amendment found cases it had missed and showed false
its claim that answering N1 and N8 would leave nothing to rule on. The
owner then ruled on every open question. Both are applied here. Nothing
above is rewritten: D3, D4, D7 and the Confirmation carry dated
annotations, and the rows below replace the first amendment's rows of the
same id. The status stays `accepted`.

### The review's findings, checked at the cited lines

| Finding | Holds? | What was checked |
| :--- | :--- | :--- |
| a1, a 200 the SDK turns into an error | yes | `responses.go:438`, `generatecontent.go:240` and `:243`, `messages.go:347` and `:350` return an `*APIError` with `Status` 0 of kind `ErrIncomplete` or `ErrEmptyAnswer`; 0062's `refusalOutcome` maps `Status` 0 to `other` (`llmprovider/refusals_0062_record_test.go:372` to `:389`) |
| a2, statuses to outcomes | yes | the PLAN's step 1a.2 names three outcomes; 0062 maps 404 to `unreachable` and 401, 403, 408, 409, 429 and a transport failure to `other`. The owner's ruling on N8 lists 401, 403, 429 and a transport failure as `unreachable` too; the ruling's list is followed where it differs from 0062's |
| a3, `Message` and `Code` not redacted | yes | `scripts/check_fixtures.py:10` and `:11` read every field of a fixture |
| a4, tools loaded inside `input` | yes | `InputLoading` sends them in a `tool_search_output` item (`responses.go:183` to `:189`); `withLoaded` keeps them out of `tools` (`deferral.go:345` to `:361`) |
| a5, OpenCode Zen's Messages route | yes | `routes_snapshot.json` routes Zen's `claude-*` models to Messages; `Generate` sends under `opencode` (`opencode.go:334`). Whether Zen passes Anthropic's error body through unchanged is not shown: Zen's server source is not in the opencode checkout (`4ce137b39e`), whose client tests show Zen prefixing some upstream errors with "Upstream request failed:" (`packages/ai/test/provider-error.test.ts:125`, `:311`), as F7 shows on Zen's Google route. ST66 records it |
| a6, the PLAN's stop lists | yes | the PLAN's Rules and Phase 1b each list stops the first amendment mapped to outcomes |
| a7, a skipped route's index | yes | F4 is `tools.0.`, F10 `tools[0]`; the Rules allowed no change of position |
| a8, a partial run | yes | no `-timeout` on the commands; 90 seconds a case (`llmprovider/live_gateways_test.go:96` to `:98`); 0062's writer replaces a file (`refusals_0062_record_test.go:459`) |
| a9, error text in failure calls | yes, at other lines | `t.Fatal(err)` is at `live_0062_refusals_measure_test.go:37`, `:48`, `:52`, `:67`, `:70`, `:101`, `:115`; line 415 of the record file is the `Generate` call |
| a `chatcompat` id beginning "opencode" | yes | `chatcompat` refuses only built-in ids; `serviceOf` maps any "opencode…" label to OpenCode (`api_error.go:500`) |
| a caller tool named `use_tool` beside Grok's wrapper | no, as stated | the wrapper is sent only in the Dispatch form (`deferral.go:118` to `:130`), and in that form `CheckDispatch` refuses a caller tool named `use_tool` before the network (`deferral.go:142` to `:146`). Without deferral a caller's `use_tool` is an ordinary tool. The owner's rule is recorded anyway (ST72) |
| ST10, the native `tool_search` entry | yes | the entry carries the caller's Search tool's description and schema (`tools.go:85` to `:98`) |
| ST45, "logs nothing" | yes | D5's "no warning" is a `Warning` on a response, not a log |
| ST36, the API check | yes | `scripts/check_api.py:85` runs apidiff with `-incompatible`; `:41` compares the whole `llmprovider` subtree, `x/` included |
| ST37, the plant | yes | `Retryable()` is false for any child of `ErrInvalidRequest` whatever `terminal` says (`api_error.go:193` to `:205`) |
| ST30, ST28, ST44, ST46 | yes | `Classify` calls `ClassifyHTTPError` only; `FieldsUnchanged` as placed never runs the helper; the redaction proof planted into the body only |
| ST05 | yes | on the native form `deferral.Tools` returns `req.Tools` (`deferral.go:257` to `:258`) |
| ST40, ST43 | yes (the reviewer's inference, accepted) | the plants as written are red only by timing, or only with a parse that returns part of a name |
| docs and changelog content | yes | no row covered D8's docs or the changelog's text |

### The owner's rulings of 2026-11-20

Each label is quoted exactly as the owner's answer, relayed by the
coordinator; what each decides is in the rows below.

* **N1** (with the review's a6): "No stop: apply the matchers as written
  (Recommended)".
* **N2** (with a4): "From the message's `function '<name>'` only
  (Recommended)".
* **N3** (with a5): "By service and marker; record Zen Messages too
  (Recommended)".
* **N4**: "HTTP error replies only, with a guard test (Recommended)".
* **N5**: "Existing rules win (Recommended)".
* **N6**: "Name by string; find the error through wrapping
  (Recommended)".
* **N7** (with a7): "Named edits to a recorded body, each stated in a
  comment (Recommended)".
* **N8** (with a2): "Mark it unreachable or skipped; earlier fixtures
  stand (Recommended)".
* **a1**: "As accepted: the schema was not refused (Recommended)".
* **ST10**: "Name the caller's Search tool (Recommended)".
* **Who runs the recording**: "API keys from the environment only
  (Recommended)". It replaces the rule that only the owner runs a live
  step: Phase 1b is run by an agent or the coordinator, with the keys
  already in the process environment.

### Rows changed

| Id | Was | Now | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST03 | open | decided | on every OpenAI-form route (OpenAI's API key and sessions, OpenCode's Responses and chat routes), `Tool` is the sent name in `Message`'s `function '<name>'`; never taken from `param` | N2 | `Names`; `NeverNamesUnsent`'s subcase: `param` names index 1, the message an unsent tool, `Tool` empty |
| ST05 | decided | decided, test fixed | as before | D3 row 3; D4 | `Deferral` on a Claude model in the Append form (before 4.5), so the sent list and `Request.Tools` differ |
| ST08 | open | decided | the kind on any route of the `openai` or `opencode` service whose reply carries a read marker; the docs name the routes each form was recorded on; Anthropic's marker joins `opencode` only as ST66 (1) says | N3 | `TestToolSchemaRefusal_SharedLabel_0084` |
| ST10 | decided | decided, corrected | as before, except: when the message names the caller's Search tool, whose native `tool_search` entry carries no name, `Tool` is that name; a tool loaded inside a `tool_search_output` item is named when the message names it | the ruling on ST10; N2; D2 | `NeverNamesUnsent` (the rest); `TestToolSchemaRefusal_SearchLoaded_0084` |
| ST12 | decided | decided, test settled | as before | D2; Q4; N7 | `SeveralNamed`, always built: from O2's record, or from F1 and F4 given a second name or index |
| ST13 | open | decided | `Tool` is the name when every entry it can mean shares it | N6 | `TestToolSchemaRefusal_DuplicateNames_0084` |
| ST17 | open | decided | a failure in a 200 body or in a stream keeps today's kind; `Tool` empty | N4 | `TestToolSchemaRefusal_StreamUnchanged_0084` (a guard) |
| ST20, ST23, ST50 | decided, test waiting on N7 | decided | as before | D3; N7 | `ProxyAltered` |
| ST26 | open | decided | `ErrContextOverflow` | N5 | `TestToolSchemaRefusal_Precedence_0084` |
| ST28, ST44 | decided | decided, test moved | as before, read through `Generate` and `Stream`, so the helper runs | D4; D2 | `FieldsUnchanged`, in `providers` |
| ST30 | decided | decided, plant moved | as before | D1; R1, R2 | `Classify` and `Names` each assert `errors.As` and `Kind` |
| ST32 | open | decided | `ErrProviderUnavailable`, `Retryable()` true, `Tool` empty | N5 | `Precedence` |
| ST33 | open | decided | the helper finds the `*APIError` through wrapping, sets `Tool`, and returns the error it was given | N6 | `TestNameRefusedTool_Wrapped_0084` |
| ST36 | decided | decided, check fixed | `make api-check` exits 0; `tools/apidiff.py` from Phase 2's start commit lists exactly the two names, compatible | D7; Q2 | those two commands |
| ST37 | decided | decided, plant fixed | not retried: `Retryable()` false, except as ST32 says | D1; N5 | `Classify` |
| ST40 | decided | decided, test fixed | as before | PLAN Phase 2 step 5 | `Concurrent`, its server holding the first request until the second arrives |
| ST43 | decided | decided, plant fixed | as before | D2; C12 | `NameBeyondBound` |
| ST45 | decided | decided, ground corrected | no log record holds the refusal's text or `Tool`; it rests on H8 and the PLAN's Phase 4 step 1 ("no response text reaches ... a log"), not on D5 | H8; PLAN Phase 4 | `NoLog` |
| ST46 | decided | decided, widened | no field of a record holds a key, an account id, a UUID or an e-mail address: body, `Message` and `Code` alike | PLAN Phase 1a steps 2 to 4; the review's a3 | `TestToolSchemaRecord_Redacts_0084`, planting into `Message` and `Code` too |
| ST47 | decided | decided, widened | each output line holds only the case, route, model, HTTP status and outcome | the ruling on who runs the recording | `TestToolSchemaRecord_LogLine_0084` |
| ST60 | open | decided | no stop: every record is read by the matchers as written; recognised, it feeds `Classify` and `Names`; unrecognised, `ErrInvalidRequest` and `TestToolSchemaRefusal_RecordedFallback_0084`; a new form waits for a later amendment | N1 | `RecordedFallback` |
| ST61 | open | decided | as ST67 and ST68 | N8 | `TestToolSchemaRecord_Outcome_0084` |
| ST62 | open | decided | named edits allowed, the tool's position included, each stated in the test's comment | N7 | the tests that use them |

### Rows added

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST66 | OpenCode Zen's Messages route (`claude-sonnet-5`) under the `opencode` label | open: awaits live evidence (the run, `zen-messages-*`) | see "Each live observation" below | N3; N1; N8 | `Classify`, `Names` or `RecordedFallback` |
| ST67 | The recorder's outcome for each reply | decided | no error, or a `Status`-0 `ErrIncomplete` or `ErrEmptyAnswer`: `accepted`; 400 to 499 but 401, 403, 404, 408, 409, 429: `refused`; 404, 401, 403, 429, or any error that is not an `*APIError`: `unreachable`; 408, 409, 5xx, or another `Status`-0 failure: `other`, sent once more by the recorder, then `unreachable` if still so; a key unset or blank: `skipped` | N8; a1; N4 | `TestToolSchemaRecord_Outcome_0084` |
| ST68 | A route with no case `refused` or `accepted` | decided | `skipped` as a whole; its 0062 fixtures stand | N8 | `Outcome` |
| ST69 | A read route `skipped` | decided | `Classify` and `Names` use its 0062 fixture, the tool moved to index 1 by a named edit | N7; N8 | `Names` |
| ST70 | A rerun, and a run cut short | decided | the writer never replaces a `refused` or `accepted` record, and writes `skipped` only where none exists; the command carries `-timeout 180m`, which covers every case sent twice; a run cut short is run again whole | the review's a8 | `Outcome` (the writer's rule) |
| ST71 | The recorder's failure, skip and log calls | decided | a fixed text and at most the case id; never an error's text | the review's a9; the PLAN's Rules | the PLAN's `grep` check, seen to hit `live_0062_refusals_measure_test.go` |
| ST72 | A caller tool named `use_tool` | decided | without deferral, an ordinary sent tool, named by string; beside Grok's wrapper, refused before the network | N6; `CheckDispatch` | `Names` (as ST09); `TestCheckDispatch_0040`, unchanged |
| ST73 | D8's docs: the standards guide's R24 and R25, `docs/architecture.md`, the package docs (`llmprovider/provider.go`, `grok.go`, `openai.go`, `claude.go`, `opencode.go`), the wizard's `errorClasses` | decided | each names the kind and the field and says what N3 to N5 decided | D8; N3 to N5 | `grep -c ErrToolSchemaRefused` on each file, each exit 0, seen to exit 1 on a scratch copy with the entry removed; `TestDescribe_ToolSchemaRefused_0084`; Phase 4's read against this table |
| ST74 | The changelog's **Added** entry | decided | names the kind, the field, "matched by service and marker", the routes each form was recorded on, the fall-backs (a reworded refusal, a failure in a 200 or a stream, `x-should-retry`, an overflow) and the new leading text | D8; N3 to N5 | `grep -c ErrToolSchemaRefused CHANGELOG.md`, exit 0; `make changelog-check`; Phase 4's read |
| ST75 | A `chatcompat` id beginning "opencode" | out of scope | the caller's own name; its errors already take OpenCode's rows by `serviceOf` today, and this record changes nothing about that | – | – |
| ST76 | Phase 1c's fixture check or identifier scan finds a value the redaction should have removed | open (N9) | ~~the PLAN stops: "a hit is a stop, never a baseline entry"~~ (struck 2026-11-20, fourth pass: decided in the third amendment; see ST76 there) | – | – |
| ST77 | How a key reaches the run | decided | only through the process environment, read by the recorder by its `ProviderEnvVars()` name and passed to the provider; never an argument, a file, a log, output or a record | the ruling on who runs the recording | `TestToolSchemaRecord_NoLeak_0084`; the PLAN's `grep` for `os.Args`, `flag.` and `os.ReadFile` in the recorder's files, exit 1 |
| ST78 | A key unset or blank | decided | the route is `skipped`, reason "key unset or blank"; the run goes on; nobody is asked | N8; the ruling on who runs the recording | `Outcome` (unset, empty and spaces) |
| ST79 | What the run prints | decided | standard output and the test log carry only the case, route, model, HTTP status and outcome | the ruling on who runs the recording | `NoLeak`; `LogLine` |
| ST80 | A key-shaped string or an `org-` id in a provider's reply | decided | neither appears in any output, log or record; a raw body or raw error text is redacted before it is written and never printed | the ruling on who runs the recording; the review's a3 | `NoLeak`, planting both into a reply's body and message |
| ST81 | What the running agent reads | decided | the redacted records and the allowlisted summary only, after `make fixtures-check` and `tools/denyscan.py` pass on the records, and `scripts/check_fixtures.py` and `tools/denyscan.py` on the run's output file ~~(which would also catch a panic's printed value)~~ (struck 2026-11-20, fourth pass: neither looks for a key; see ST85) | the ruling on who runs the recording | those checks, in Phase 1c's order |
| ST82 | A step needing a browser or device sign-in, a saved sign-in session, a provider account page or a person at a console | decided | none in this record: every route uses an API key from the environment; the ChatGPT and plan sessions are tested offline only (`SharedLabel`) | the ruling on who runs the recording | none: no such step |

### Each live observation, and what it means for the code

With the rulings above, each observation of the run maps to an outcome.
"Recognised" means one of D3's matchers, as N3 places them, reads it.

* **ST53, `root-string` on a read route.** (1) The 0062 form, or another
  form recognised: its record joins `Classify` and `Names`. (2) A form not
  recognised: `ErrInvalidRequest`; `RecordedFallback`. (3) Accepted: no
  change; the 0062 fixture stands for the matcher. (4) `unreachable` or
  `skipped`: the 0062 fixture stands, at index 1 (ST69).
* **ST54, O1 on any route; ST57, O4.** (1) Recognised: `Classify` and
  `Names`. (2) Accepted: no change. (3) Not recognised:
  `RecordedFallback`. (4) `unreachable` or `skipped`: nothing recorded,
  nothing tested from it.
* **ST55, O2.** (1) One tool named: `Tool` is it; `SeveralNamed` from a
  named edit. (2) Two named, by name or index: `Tool` empty;
  `SeveralNamed` from the record. (3) The marker, no name: `Tool` empty.
  (4) Not recognised: `RecordedFallback`. (5) Accepted, `unreachable` or
  `skipped`: nothing.
* **ST58, O5.** (1) Accepted: no SDK change; gobble's drop is safe there.
  (2) Refused without a marker: `OthersUnchanged`. (3) Refused with a
  marker: the kind with `Tool` empty, since only `noop` was sent;
  `Classify`. (4) `unreachable` or `skipped`: nothing.
* **ST59, O6.** (1) Tag, name, colon: `Classify` and `Names`. (2) The tag
  in another order: the kind with `Tool` empty; `Classify` and `Names`.
  (3) No tag: `RecordedFallback`. (4) Accepted, `unreachable` or
  `skipped`: nothing.
* **ST66, Zen Messages.** (1) Anthropic's body unchanged (type
  `invalid_request_error`, message beginning `tools.N.` and holding
  `input_schema`): Anthropic's marker joins the `opencode` service's rows,
  so the kind on every OpenCode route that sends it, `Tool` the entry N of
  the sent list; the record joins `Classify` and `Names`. (2) Anthropic's
  text behind a Zen prefix, or in Zen's own envelope: not recognised;
  `ErrInvalidRequest`; `RecordedFallback`; a later amendment may read it.
  (3) A marker the `opencode` service already reads (OpenAI's code, the
  Google phrase): that row applies. (4) Accepted: no change. (5)
  `unreachable` or `skipped`: Anthropic's marker does not join `opencode`;
  a Zen Messages refusal stays `ErrInvalidRequest`.
* **Any case: a 200 the SDK finds cut short or empty** is `accepted`
  (ST67).

### Open question for the owner

* **N9 (ST76).** A redaction hit found by Phase 1c's checks in a written
  record. Today the PLAN stops.

### Counts after both amendments

82 rows: 71 decided, 3 out of scope, 1 open (ST76), 7 open: awaits live
evidence (ST53, ST54, ST55, ST57, ST58, ST59, ST66).

With these rulings every observation of the recording run maps to an
outcome with no further ruling, except one: a value the redaction should
have removed, found by Phase 1c's checks in a written record (ST76), stops
the build until the owner answers N9.

## Third amendment of 2026-11-20: N9 answered, and the readings recorded

The third and last records pass of the day, read at `24756a6`. Nothing
else changes.

### The owner's ruling on N9

* **N9** (ST76): "Treat that case as unrecorded, fix the recorder, rerun
  (Recommended)", the owner's answer, relayed by the coordinator.

### Readings of the drafter, not rulings

The coordinator accepted these as consistent with the rulings they follow;
they are the drafter's readings, not owner rulings:

* **Under "By service and marker; record Zen Messages too
  (Recommended)":** ST66 (1). Anthropic's marker joins the `opencode` rows
  only if the run records Anthropic's body unchanged on Zen Messages.
* **Under "Mark it unreachable or skipped; earlier fixtures stand
  (Recommended)":** ST67. 408, 409, 5xx and other `Status`-0 failures are
  `other`, then `unreachable` after the recorder's own second send, so the
  run is one command.
* **Under "Treat that case as unrecorded, fix the recorder, rerun
  (Recommended)":** ST76's rerun. Before the rerun, the leaked case's file
  is removed, unread, from the run's directory outside the tree. Without
  that, the writer, which never replaces a `refused` or `accepted` record
  (ST70), would keep the leaked file.

**On a2.** The question put for N8 described 0062's recorder as recording
401, 403, 429 and a transport failure as `unreachable`. That was the
coordinator's wording error: 0062 records them as `other`
(`llmprovider/refusals_0062_record_test.go:372` to `:389`). What is built
is the ruling's own list, `unreachable` (ST67).

### Rows changed and added

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST76 | Phase 1c's fixture check or identifier scan finds, in a written record, a value the redaction should have removed | decided (was open) | nothing of that case is committed; it is `unreachable` for that run; the recorder's redaction and `TestToolSchemaRecord_NoLeak_0084` are extended to the missed value in a further recorder commit; an agent reruns the recording. If the same case leaks again after one fix and one rerun, that is a deviation: the build stops and goes to the owner. A third pass is never routine | N9 | `NoLeak`, extended with the missed value's shape, seen red at the recorder before the fix |
| ST83 | How a leak is reported | decided | the check names the file, the field and the rule that matched, never the matched text. `scripts/check_fixtures.py` prints the path, line, column, kind and a fingerprint, never the value. `tools/denyscan.py` prints the file, line and kind. ~~Each record holds one top-level field per line, so the line names the field, and the agent prints only that line's key.~~ (Corrected 2026-11-20, fourth pass: see ST83 in the fourth amendment.) Nobody reads the leaked value | N9 | the existing `test_fixtures_check_*` cases in `scripts/test_gates.py` (`assertFixtureCaught` asserts the value is not printed); `TestToolSchemaRecord_FieldPerLine_0084` (added, Phase 1a) |

### Counts after all three amendments

83 rows: 73 decided, 3 out of scope, 0 open, 7 open: awaits live evidence
(ST53, ST54, ST55, ST57, ST58, ST59, ST66), each with every observation
mapped.

Every observation of the recording run now maps to an outcome with no
further ruling. One stop is left by design: the same case leaking again
after one fix and rerun is a deviation for the owner (ST76).

## Fourth amendment of 2026-11-20: corrections from the delta check

Corrections only. An independent delta check confirmed most findings
resolved and left seven items not confirmed. Each was checked at its cited
lines and holds. No owner ruling is added. The changes follow the rulings
already recorded.

### Rows changed

| Id | Now | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- |
| ST36 | decided, check corrected | the two commits are compared on a scratch clone: `tools/apidiff.py <base> <scratch commit>`, where the scratch commit holds Phase 2's changed files on the base. Comparing with the working tree would also list another record's uncommitted Go work, since Phase 2 has no clean-tree prerequisite | D7 | that command and `make api-check` exit 0 |
| ST37 | decided, rationale corrected | the parent-`ErrRateLimited` plant leaves `Retryable()` false, because the D3 rows return a terminal error (`api_error.go:197` to `:199`). It is caught because `Unwrap` then also yields the legacy `ErrInvalidRequest`, and by `errors.Is(err, ErrRateLimited)` being asserted false. A second plant, `shouldRetry` set for the kind, makes `Retryable()` true and is caught | D1 | `Classify` |
| ST60 | decided, test corrected | an unrecognised `refused` record classifies at the phase's tree to the kind it has at the base, with `Tool` empty. That kind is always beneath `ErrInvalidRequest`, by ST67 | N1 | `RecordedFallback` |
| ST67 | decided, corrected | `refused` only for an `*APIError` of status 400 to 499, other than 401, 403, 404, 408, 409 and 429, that `errors.Is` `ErrInvalidRequest`. Any other `*APIError`, and any 3xx, is `unreachable`: Gemini's and xAI's 400 key refusals (`ErrAuthFailure`, `api_error.go:554` to `:561`), a 402 (`ErrQuotaExhausted`, `:546`), and a 400 with `x-should-retry: true`. The other rows are as before | N8 ("Mark it unreachable or skipped; earlier fixtures stand (Recommended)"); a1 | `Outcome` |
| ST69 | decided, widened | any read route with no recognised Phase 1 record (skipped, unreachable, root string accepted, or nothing recognised) feeds `Names` and `Classify` from its 0062 fixture, the tool moved to index 1 | N7; N8 | `Names` |
| ST71 | decided, check replaced | the recorder's files neither call nor name `SkipIfTransient`, `skipIfTransient`, `LiveModel` or `liveModel`. No call that prints takes anything but literals, constants of fixed text and the allowlisted fields. Calls that print are `t.Log`, `Logf`, `Error`, `Errorf`, `Fatal`, `Fatalf`, `Skip`, `Skipf`, `fmt.Print*`, `fmt.Fprint*` and `log.` | the PLAN's Rules; the review's a9 | `TestToolSchemaRecord_PrintCalls_0084` |
| ST79 | decided, made to agree with the command | the output file holds only `go test -v`'s framing lines, the `t.Logf` case line and the recorder's fixed texts. The framing lines are `=== RUN`, `=== PAUSE`, `=== CONT`, `=== NAME`, `--- PASS:`, `--- FAIL:`, `--- SKIP:`, `PASS`, `FAIL`, and `ok` or `FAIL` with the package | the ruling on who runs the recording | `TestToolSchemaScan_0084` on the output; `PrintCalls` |
| ST81 | decided, corrected | as before, with the key-shape scan (ST85) beside the two checks; the claim that those checks would catch a panic's value is struck | the ruling on who runs the recording | Phase 1b's four scans; Phase 1c's |
| ST83 | decided, restated | every line of a record lies under exactly one top-level key, the nearest line at or above it at two spaces' indent. The body, `Message` and `Code` are each one line; nested tools and schemas span several (`internal/wiretest/wiretest.go:173` to `:183`). The agent prints only that key. A copy of a leaked record already placed in `testdata/` is removed unread, by the path the check printed | N9 | `TestToolSchemaRecord_FieldPerLine_0084`, restated; `test_fixtures_check_*` |

### Rows added

| Id | Case | State | Outcome a test can check | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ST84 | A hit in the run's output file (`tmp/0084-run.txt`), such as a panic's stack trace, or a line outside ST79's list | decided | the agent that ran Phase 1b scans the file before reading it (`TestToolSchemaScan_0084`, `scripts/check_fixtures.py`, `tools/denyscan.py`). On a hit, the file is deleted unread and the run repeated once. The records are judged by their own scans. A second hit after the repeat is a deviation for the owner. The command's `-trimpath` keeps this Mac's paths out of a stack trace | N9, applied to the output; the ruling on who runs the recording | `TestToolSchemaScan_Plant_0084` (its output-file case) |
| ST85 | A key-shaped value in a record or in the output | decided | found by `TestToolSchemaScan_0084`, which reports a line when `redact.Redact` changes it, printing only the file and line. It runs on the output file in Phase 1b and on the records in Phases 1b and 1c. It sees the shapes `internal/redact` knows, and each of the five providers' keys used here has one (`redact.go:90` to `:108`). A scan for the exact values of the keys in the environment is not used: the ruling forbids anyone to compare a key value | the ruling on who runs the recording; N9 | `TestToolSchemaScan_Plant_0084`, planting each provider's key shape, a `Bearer` value and an `api_key` pair |

### Counts after all four amendments

85 rows: 75 decided, 3 out of scope, 0 open, 7 open: awaits live evidence
(ST53, ST54, ST55, ST57, ST58, ST59, ST66), each with every observation
mapped.

Every observation of the recording run maps to an outcome with no further
ruling. One stop is left by design, ST76's and ST84's bound: the same leak
found again after one fix and rerun goes to the owner as a deviation.
