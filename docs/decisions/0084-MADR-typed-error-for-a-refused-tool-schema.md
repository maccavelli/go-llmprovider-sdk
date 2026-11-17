---
status: proposed
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
* **`make api-check`** lists exactly D7's two additions.
* **The owner's live steps** (the PLAN's Phase 1, O1 to O6) record the forms
  this record could not establish, with no key displayed.
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
