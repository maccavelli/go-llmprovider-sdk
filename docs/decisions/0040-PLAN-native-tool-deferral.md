---
status: proposed
date: 2026-10-13
associated-madr: "0040-MADR-native-tool-deferral.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Native Tool Deferral

Associated MADR: [0040-MADR-native-tool-deferral.md](0040-MADR-native-tool-deferral.md)

Every code location a step names has an id, P*n*, in this PLAN's evidence
index, checked by the claims checker. A step that needs a location the index
does not hold adds it, and checks it, before the step runs. The MADR's own
claims are its E*n* rows.

## Goal

When it is done:

* **The contract** (D1):
  * `Tool.Deferred`, `Tool.Search` and `FunctionCallOutputItem.Load` exist,
    each zero by default;
  * the four refusals of D1 fail before the network with `ErrInvalidRequest`,
    for every provider, through `llmtest` R23.
* **Each adapter sends its form** (D1's table):

  | Adapter, route | Form | When |
  | :--- | :--- | :--- |
  | `claude`, `opencode` Messages | D2 native | Claude 4.5 and later |
  | `openai` ChatGPT session | D3 native | every model |
  | `openai` key, `opencode` Responses | D3 native | `gpt-N.M` 5.4 and later |
  | `kilo` | D4 OpenRouter protocol | once Phase 0 shows Kilo forwards it |
  | `grok` | D5 dispatch | every model |
  | the rest, and every model left out above | D6 append on load | always |

* **A request with no deferred tool is unchanged** on every wire: every
  existing G-wire golden is byte for byte the same.
* **0039's breakpoint** never lands on a deferred tool (D2).
* **`WarningDeferralEmulated`** marks every response sent as in D5 or D6, or
  that appended a load (D7).
* **Proven live:**
  * before the code, by the probe of 0040-MADR's Confirmation, Q5,
    approved by the owner;
  * after it, by a gated test through the SDK.
* **Released** as a minor version. `tools/apidiff.py` reports exactly the
  additions below.

## Scope

### In scope

* **`llmprovider`:**
  * `Tool` and `FunctionCallOutputItem`'s new fields;
  * `validate`;
  * `WarningDeferralEmulated`;
  * `llmtest` R23's new cases.
* **`llmprovider/internal/wire`:**
  * a new module-internal package, `deferral`, that resolves a request's
    form, its tool list and its loads, once, for every encoder;
  * the Messages, Responses, Chat Completions and Gemini encoders and
    decoders named in each phase;
  * `messages.AddCache`.
* **Every provider that encodes tools:**
  * `claude`, `opencode` (all four routes) and `openai`;
  * `grok`, `kilo`, `together`, `huggingface`, `ollama` and `gemini`;
  * each one's package doc (R12).
* **`llmprovider/internal/wirecase`:** two scenarios, `defer-offer` and
  `defer-load`, with new goldens for every case.
* **The live tests:** a probe, `llmprovider/live_0040_probe_test.go`
  (Phase 0), and a measurement, `llmprovider/live_0040_deferral_test.go`
  (Phase 7).
* **Docs:**
  * `docs/guides/api-standards.md` R6, at `Tools`;
  * `docs/architecture.md`;
  * `README.md`;
  * `docs/README.md`.

### Out of scope

As 0040-MADR D8 decides:

* the services' own search tools (Anthropic's regex and BM25, OpenAI's
  hosted `tool_search`, OpenRouter's `openrouter:tool_search`);
* OpenAI namespaces, MCP toolsets, and Anthropic's computer and browser
  toolsets;
* `input_examples`.

gobble-cli's adoption is its own repository's record. Phase 7 drafts the
issue, and opens it only on the owner's ask.

### The exported additions

`tools/apidiff.py` must report these, all `added`, and nothing else:

| Package | Addition | Phase |
| :--- | :--- | :--- |
| `llmprovider` | `Tool.Deferred`, `Tool.Search`, `FunctionCallOutputItem.Load` | 1 |
| `llmprovider` | `WarningDeferralEmulated` | 4 |

## Rules for every phase

* **Research first.** Each phase starts by reading the code it changes, at
  the commit it starts from, and re-running the claims check on both
  records. A fact that differs from this PLAN or the MADR is a deviation.
* **Red first.** Each new test is run before the change, and its FAIL lines
  are recorded here.
* **Plants.** Every new assertion is shown failing by `tools/plant.py` on a
  scratch clone, never in the tree. Each plant's FAIL line is recorded.
* **Python.** Every script is Python, and a reusable one goes in `tools/`.
  Specs and scripts are written with the Write tool. Exact edits go through
  `tools/edit.py`.
* **The goldens.** An existing golden never changes. A new golden is
  generated alone, with `-update -run` and its scenario, and its content is
  recorded here.
* **Before a phase is staged:**
  * `make pre-add-check FILES="…"`;
  * `make records-check`;
  * `tools/denyscan.py --changed` and `tools/linkcheck.py`;
  * the claims check on both records;
  * the gate, `tools/gate.py`.
* **Billed requests** run only after the owner approves their count. Keys
  are checked for presence only, never printed.
* **A deviation stops the work.** Give the evidence, the options with the
  recommended one first, and a dated entry here, plus a MADR amendment where
  a fact or a decision changes.
* **The owner commits** each phase, pushes and tags. While one phase is
  staged, the next stays unstaged.
* **Ordering with 0041-PLAN.** Both plans change `internal/wire/messages`:
  0041's Phase 1 its thinking, this PLAN's Phase 2 its tools and tool
  results. They are run one at a time, in the order the owner chooses. The
  second starts by re-reading the files the first changed.

## Implementation Steps

### Phase 0: records and the live probe

1. **The records.** 0040-MADR was accepted on 2026-10-13. This PLAN becomes
   `in-progress` on the owner's approval. `docs/README.md` gets its status.
2. **The claims check.** If 0041-PLAN's Phase 0 has not landed, this phase
   commits `tools/claims.py`, its tests and its gate check, exactly as
   0041-PLAN-decide-per-model-behaviour-from-metadata.md Phase 0 step 2
   specifies. That phase then reuses it. Either way the check runs on both
   0040 records before this phase is staged.
3. **Ask for Q5.** The owner approves the probe's 21 billed requests, or
   fewer paths. Nothing runs before that. The ChatGPT path also needs the
   owner's signed-in session, as `live_chatgpt_session_test.go` already
   uses (P30).
4. **The probe,** `llmprovider/live_0040_probe_test.go`:
   * It is tagged `live_gateways`, and gated by
     `LLMPROVIDER_LIVE_0040_PROBE=1` and each path's key.
   * It builds each request by hand, with `net/http`, because the SDK cannot
     send these fields yet.
   * It logs each response's status, stop reason, the tool the model called,
     and the decoded usage. It never logs a key, a header or a prompt.
   * Each path sends a system prompt long enough to be cached, built as 0039's
     measurement built one (P31), with its own first line, so that no path
     reads another's cache.
5. **The paths,** each with three requests: the model calls the search; the
   result loads one tool; the model calls that tool.

   | Path | Model | Fields sent | Settles |
   | :--- | :--- | :--- | :--- |
   | Claude | `claude-haiku-5-5` | `defer_loading`; a `tool_result` of one `text` block and one `tool_reference` | the mixed content (MADR §2 leaves it open); the cache read on requests 2 and 3 |
   | Zen Messages | `claude-haiku-5-5` | the same, through Zen | that Zen's upstream takes it |
   | OpenAI key | `gpt-6-luna` | a client `tool_search`; deferred tools off the top level; a `tool_search_output` | the stateless replay of `tool_search_call` with `store: false` and object `arguments`; the cache read |
   | Zen Responses | `gpt-6-luna` | the same, through Zen | that Zen's upstream takes it |
   | ChatGPT session | its default model | the same, on the Codex backend | that the backend takes it, as codex does (MADR E12–E17) |
   | Kilo | `anthropic/claude-haiku-5.5` | `deferred_tools` v1, custom search | that Kilo forwards it; the exact form of a result under `result_key` |
   | Grok | `grok-4.6` | the `use_tool` wrapper; the loaded schema as text in the result | that Grok calls through the wrapper with arguments that match the schema |

6. **Record** each path's outcome here, as a table of status, stop reason,
   call, and cached tokens.
7. **What each failure changes, before any code:**
   * **Claude refuses text beside `tool_reference`:** D2 sends the text as a
     separate `text` block in the same user turn. That is a MADR amendment.
   * **OpenAI refuses the stateless replay:** D3 needs `store: true` with
     `previous_response_id`, or falls back to D6 on that wire. The owner
     chooses.
   * **A Zen route refuses:** that route falls back to D6. That is a D1-table
     amendment.
   * **Kilo does not forward `deferred_tools`:** D4 is not built, and Kilo
     takes D6, as D4 already provides. Phase 6 is then skipped and recorded.
   * **Grok does not use the wrapper reliably:** the owner chooses between
     D6 for Grok and keeping D5.
   * **The cache does not hold on a native path:** that path is recorded,
     and the owner chooses whether it stays native.
8. **Green.** `go vet -tags live_gateways ./llmprovider/`, the gate. Stage.
   The owner commits.

### Phase 1: the contract (D1)

1. **Research.**
   * Read `Tool` (P1), `FunctionCallOutputItem` (P2) and `validate` (P3).
   * Read `llmtest`'s R23 table and runner (P4, P5).
   * Read R6 and R9 in `docs/guides/api-standards.md`.
2. **Change.**
   * `Tool.Deferred bool` and `Tool.Search bool`, each documented against
     D1.
   * `FunctionCallOutputItem.Load []string`.
   * `validate` refuses, with `ErrInvalidRequest` and a message naming the
     tool or the name:
     * two `Search` tools;
     * a `Search` tool that is `Deferred`;
     * every tool `Deferred`;
     * a `Load` name that is not a `Deferred` tool in `Tools`;
     * `ForceTool` naming a deferred tool that no result in `Input` loads.
   * R6 in `api-standards.md` names the fields, citing 0040.
3. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestRequest_Deferral_0040` (`llmprovider/deferral_0040_test.go`) | each refusal above fails with `ErrInvalidRequest`; a valid deferred request, a request with only a `Search` tool loaded, and a request with no deferral all pass |
   | `llmtest` R23, five new cases | every provider refuses each before the network |

4. **Plants:** each of the five checks removed in turn.
5. **Green.** `go test ./...`. `make api-check` shows the three fields. The
   gate. Stage. The owner commits.

No wire sends anything for the fields yet. Every golden is unchanged.

### Phase 2: the shared form, and Anthropic Messages (D1, D2)

1. **Research.**
   * Read the Messages tools (P6), `Block` (P7), the tool result (P8) and
     `AddCache` (P9).
   * Read the two Messages bodies (P10, P11).
   * Read the Claude version rule, which the model rule reuses (P12).
2. **`internal/wire/deferral`,** one package every encoder calls:
   * **`Form(provider, route, model, session) Form`** returns `Native`,
     `OpenRouter`, `Dispatch` or `Append`, from the MADR's D1 table. The
     Claude rule is `claudeVersionRE` at 4.5 or later. The GPT rule parses
     `gpt-N.M` at 5.4 or later. Anything unparsed is `Append`.
   * **`Loaded(input) []string`** returns the names `Load` brings in, in
     first-load order, without repeats.
   * **`Tools(req, form) []llmprovider.Tool`** returns the tools a form
     sends:
     * all of them, for Messages native;
     * the non-deferred ones, for Responses native;
     * the non-deferred ones then the loaded ones, for `Append`;
     * every tool, for `Append` with `PreviousResponseID`.
3. **Change, Messages.**
   * `MessagesTools` sends `"defer_loading": true` on a deferred tool, when
     the form is native.
   * `FromItems` sends a result with `Load` as a content list: one `text`
     block with `Output`, then one `tool_reference` per name. This is the
     form Phase 0 measured; a Phase 0 amendment governs here.
   * `AddCache` puts the breakpoint on the last tool that is not deferred.
   * `claude` and `opencode`'s Messages route call `deferral.Form` and send
     its tools.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestForm_0040` | the D1 table, row by row, with the model rules at their edges: `claude-opus-4-1` is `Append`, `claude-haiku-4-5` `Native`, `gpt-5.3` `Append`, `gpt-5.4` `Native`, `minimax-m3` on Messages `Append` |
   | `TestLoaded_0040` | first-load order, without repeats, across several results |
   | `TestMessagesTools_Deferred_0040` | `defer_loading` on deferred tools only, and nowhere when the form is `Append` |
   | `TestFromItems_Load_0040` | a result with `Load` is a text block then `tool_reference` blocks, byte for byte; one without is a string, as today |
   | `TestAddCache_Deferred_0040` | with no system prompt and the last tool deferred, the marker is on the last non-deferred tool |
   | the existing goldens | unchanged |

5. **Plants:**
   * the model rule's edge moved;
   * `defer_loading` sent under `Append`;
   * the text block dropped from a loading result;
   * the breakpoint on a deferred tool.
6. **Green.** The gate. Stage. The owner commits.

### Phase 3: OpenAI Responses (D3)

1. **Research.**
   * Read `ResponsesTools` and `AddResponsesTools` (P13, P14).
   * Read `InputItem` and `Input` (P15, P16).
   * Read `appendOutput` (P17), which both the body and the stream use
     (P18).
   * Read the three Responses bodies (P19, P20, P21) and the ChatGPT session
     check (P22).
2. **Change.**
   * **The tools.** The `Search` tool is sent as `{"type": "tool_search",
     "execution": "client", "description", "parameters"}`. Deferred tools
     are left out of the top level when the form is native.
   * **Decoding.** `appendOutput` decodes a `tool_search_call` with
     `execution: "client"` as a `FunctionCallItem`. Its `Name` is the
     `Search` tool's name, passed in by the provider, and its `Arguments` are
     the call's `arguments` as JSON text. Without a `Search` tool in the
     request, the item is dropped, as today.
   * **Replay.** `InputItem` gains `Execution`, `Status`, `Tools` and an
     object `arguments`. Each is omitted when empty, and placed where 0028's
     field order puts it (P15), so an existing body is byte for byte the
     same. A `FunctionCallItem` naming the `Search` tool is sent as
     `tool_search_call`. Its result is sent as `tool_search_output`, whose
     `tools` are the loaded definitions, each with `defer_loading: true`.
     `Output` is not sent.
   * **A load from any other result** appends its tools to the top level,
     as D6 does, and sets the warning (Phase 4).
   * The ChatGPT session takes the native form for every model.
3. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestResponsesTools_Search_0040` | the `tool_search` entry; deferred tools absent at the top level |
   | `TestAppendOutput_ToolSearchCall_0040` | body and stream both give one `FunctionCallItem` named for the `Search` tool, with the arguments' JSON; dropped with no `Search` tool |
   | `TestInput_ToolSearchReplay_0040` | the call and its output, byte for byte, with the loaded definitions |
   | `TestInput_Unchanged_0040` | every existing item kind encodes as before |
   | `TestResponses_LoadFromOtherTool_0040` | the tools are appended at the top level |
   | the existing goldens | unchanged |

4. **Plants:**
   * the `Search` tool sent as a function;
   * the call decoded under its wire name;
   * `Output` sent in `tool_search_output`;
   * a deferred tool left at the top level.
5. **Green.** The gate. Stage. The owner commits.

### Phase 4: append on load, and the warning (D6, D7)

1. **Research.**
   * Read `chatcompletions.Opts` and `toolList` (P23, P24).
   * Read Gemini's and OpenCode's Google route's tool encoders (P25, P26).
   * Read the Chat Completions callers (P27).
   * Read `Warning` (P28).
2. **Change.**
   * Every encoder under `Append` sends `deferral.Tools`.
   * `Search` is an ordinary function, and `Load` sends nothing.
   * With `PreviousResponseID`, every deferred tool is sent.
   * `WarningDeferralEmulated` is added. A response gets it, with its model
     and a `Detail` naming the form, when the request had a deferred tool
     and its form was `Dispatch` or `Append`, or it appended a load under
     D3 or D4.
3. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestAppend_Order_0040` (per wire) | non-deferred tools in the caller's order, then the loaded ones in first-load order |
   | `TestAppend_PreviousResponseID_0040` | every deferred tool sent |
   | `TestWarning_Emulated_0040` | on Gemini, Together, Hugging Face, Ollama, OpenCode's chat route, and `claude-opus-4-1` on Claude; none on a native path |
   | the existing goldens | unchanged |

4. **Plants:**
   * load order reversed;
   * a deferred tool sent before its load;
   * the warning on a native path;
   * no warning on an emulated one.
5. **Green.** The gate. `make api-check` shows the warning kind. Stage. The
   owner commits.

### Phase 5: Grok dispatch (D5)

1. **Research.** Read the Grok body (P20) and the Responses decoder (P17).
2. **Change.**
   * Grok's tools are the non-deferred ones and `use_tool`, with
     `tool_name` and `tool_input`.
   * A result with `Load` carries `Output`, then the loaded definitions as
     JSON.
   * A `use_tool` call naming a deferred tool is decoded as that tool's
     `FunctionCallItem`, and is replayed as a `use_tool` call.
   * A caller tool named `use_tool`, beside a deferred tool, is refused with
     `ErrInvalidRequest`.
3. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestGrokDispatch_Tools_0040` | the tool list, fixed across loads |
   | `TestGrokDispatch_Decode_0040` | a `use_tool` call becomes the named tool's call; one naming an unknown tool stays a `use_tool` call |
   | `TestGrokDispatch_Replay_0040` | byte for byte |
   | `TestGrokDispatch_NameClash_0040` | the refusal |

4. **Plants:**
   * the list changed on a load;
   * the decode skipped;
   * the replay sent under the tool's own name.
5. **Green.** The gate. Stage. The owner commits.

### Phase 6: Kilo's OpenRouter protocol (D4), if Phase 0 allows

1. **Skipped, and recorded,** if Phase 0 showed Kilo does not forward
   `deferred_tools`.
2. **Research.**
   * Read Kilo's body (P29).
   * Read the `reasoning_details` replay (MADR E11), which the protocol's
     state needs.
3. **Change.**
   * Kilo sends `deferred_tools` (protocol `v1`, profile `portable`,
     validation `runtime`, and a custom search naming the `Search` tool) on
     every turn.
   * Deferred tools are sent with `defer_loading`.
   * A result of the `Search` tool carries its loads under `result_key`, in
     the form Phase 0 measured.
   * A forced choice, a load from another tool, or no `Search` tool takes
     `Append`.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestKiloDeferred_Body_0040` | the object, identical on every turn |
   | `TestKiloDeferred_Result_0040` | the measured form |
   | `TestKiloDeferred_Fallbacks_0040` | each of the three falls to `Append`, with the warning |

5. **Plants:** the object changed between turns; a forced choice sent with
   the protocol.
6. **Green.** The gate. Stage. The owner commits.

### Phase 7: goldens, live measurement, docs and release

1. **The wire scenarios,** in `internal/wirecase`:
   * **`defer-offer`:** two direct tools, a `Search` tool and two deferred
     tools;
   * **`defer-load`:** a history of a search call and its result loading one
     tool.

   New goldens are generated for every case, one scenario at a time, and
   their content is recorded here. Every existing golden is unchanged, three
   runs.
2. **The live measurement,** `llmprovider/live_0040_deferral_test.go`:
   * gated by `LLMPROVIDER_LIVE_0040=1`;
   * the Phase 0 paths that stayed native, through the SDK, with the
     owner's approval of the count;
   * each path asserts the model called the loaded tool, and that the second
     and third requests read the first one's cache;
   * each check is seen failing on canned usages, offline, as 0039's were.
3. **Docs:**
   * `api-standards.md` R6;
   * `docs/architecture.md`;
   * `README.md`;
   * each provider's package doc: its form, its models, and what it does not
     send (R12).
4. **Checks:**
   * `make lint vuln parity-check dep-check coverage-check api-check
     generate-check records-check gate-selftest`;
   * `tools/apidiff.py` against the last tag shows exactly "The exported
     additions";
   * the gate.
5. **Release notes,** recorded here. The owner tags.
6. **After the tag:**
   * CI on `main` and the tag;
   * the module proxy;
   * `tools/consumer.py`.
7. **gobble-cli.** Draft the issue: adopting `Deferred`, `Search` and `Load`
   for its D7 tiers and D9 loads, and amending its 0012-MADR D13. Open it
   only on the owner's ask.

## Verification

| 0040-MADR | Proven by |
| :--- | :--- |
| §2 facts the vendors leave open | the Phase 0 probe, recorded |
| D1 contract and refusals | `TestRequest_Deferral_0040`, `llmtest` R23 |
| D1 table and model rules | `TestForm_0040` |
| D2 | the Phase 2 tests, `defer-*` goldens |
| D3 | the Phase 3 tests, `defer-*` goldens |
| D4 | Phase 6's tests, or its recorded skip |
| D5 | the Phase 5 tests |
| D6, D7 | the Phase 4 tests |
| "a request with no deferral is unchanged" | every existing golden, three runs per phase |
| the cache holds | Phase 0, and the Phase 7 live test |
| every E*n*, P*n* | the claims check |
| API | `tools/apidiff.py` |

## Rollout and Rollback

* **Rollout.** One minor version, after Phase 7. A caller that sets no new
  field sees no change.
* **Rollback.** Each phase is its own commit. Native paths fall back by
  model rule: changing a rule to return `Append` turns a path's native form
  off without a contract change. A field cannot be removed once released
  (R48). Before release, reverting Phase 1 reverts the contract, and the
  later phases must be reverted first.

## Evidence index

Checked against go-llmprovider-sdk `3e81e3d`, at that commit.

| Id | Location | Where | Matched |
| :--- | :--- | :--- | :--- |
| P1 | `Tool` | `llmprovider/provider.go:17` | `type Tool struct {` |
| P2 | `FunctionCallOutputItem` | `llmprovider/item.go:40` | `type FunctionCallOutputItem struct {` |
| P3 | `validate` | `llmprovider/contract.go:241` | `func (req *Request) validate() error {` |
| P4 | `llmtest` runs R23 | `llmprovider/llmtest/llmtest.go:183` | `r.Run("R23-invalid-values", func(r reporter) { checkInvalidValues(r, h) })` |
| P5 | R23's cases | `llmprovider/llmtest/llmtest.go:380` | `func checkInvalidValues(r reporter, h Harness) {` |
| P6 | Messages tools | `llmprovider/internal/wire/tools.go:81` | `func MessagesTools(tools []llmprovider.Tool) []map[string]any {` |
| P7 | a Messages block | `llmprovider/internal/wire/messages/messages.go:86` | `type Block struct {` |
| P8 | a Messages tool result | `llmprovider/internal/wire/messages/messages.go:170` | `case llmprovider.FunctionCallOutputItem:` |
| P9 | the cache breakpoint on the last tool | `llmprovider/internal/wire/messages/cache.go:40` | `tools[len(tools)-1][keyCacheControl] = CacheMarker(retention)` |
| P10 | Claude's tools | `llmprovider/providers/claude/claude.go:303` | `body["tools"] = wire.MessagesTools(req.Tools)` |
| P11 | OpenCode's Messages tools | `llmprovider/providers/opencode/opencode.go:527` | `body[jsonKeyTools] = wire.MessagesTools(tools)` |
| P12 | the Claude version rule | `llmprovider/internal/wire/messages/messages.go:396` | `var claudeVersionRE = regexp.MustCompile` |
| P13 | Responses tools | `llmprovider/internal/wire/tools.go:47` | `func ResponsesTools(tools []llmprovider.Tool) []map[string]any {` |
| P14 | Responses tools and choice | `llmprovider/internal/wire/tools.go:67` | `func AddResponsesTools(body map[string]any` |
| P15 | `InputItem`, in 0028's field order | `llmprovider/internal/wire/responses/responses.go:37` | `type InputItem struct {` |
| P16 | `Input` | `llmprovider/internal/wire/responses/responses.go:60` | `func Input(items []llmprovider.Item) []InputItem {` |
| P17 | `appendOutput` | `llmprovider/internal/wire/responses/responses.go:193` | `func appendOutput(r *llmprovider.Response, out outputItem) (refused bool) {` |
| P18 | the stream decodes through it too | `llmprovider/internal/wire/responses/responses.go:337` | `refused = appendOutput(result, event.Item) \|\| refused` |
| P19 | OpenAI's tools | `llmprovider/providers/openai/openai.go:271` | `wire.AddResponsesTools(body, req.Tools, req.ToolChoice)` |
| P20 | Grok's tools | `llmprovider/providers/grok/grok.go:232` | `wire.AddResponsesTools(body, req.Tools, req.ToolChoice)` |
| P21 | OpenCode's Responses tools | `llmprovider/providers/opencode/opencode.go:497` | `wire.AddResponsesTools(body, c.req.Tools, c.req.ToolChoice)` |
| P22 | the ChatGPT session check | `llmprovider/providers/openai/chatgpt_session.go:31` | `func isChatGPTSession(src llmprovider.TokenSource) bool {` |
| P23 | the Chat Completions options | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:37` | `type Opts struct {` |
| P24 | the Chat Completions tool list | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:320` | `func toolList(tools []llmprovider.Tool, sendStrict bool) []chatTool {` |
| P25 | Gemini's tools | `llmprovider/providers/gemini/gemini.go:249` | `body[jsonKeyTools] = tools` |
| P26 | OpenCode's Google tools | `llmprovider/providers/opencode/opencode.go:553` | `body[jsonKeyTools] = []map[string]any{{"functionDeclarations": decls}}` |
| P27 | Together's tools | `llmprovider/providers/together/together.go:206` | `o := chatcompletions.Opts{Tools: req.Tools, ToolChoice: req.ToolChoice, SendStrict: true}` |
| P28 | `Warning` | `llmprovider/warning.go:22` | `type Warning struct {` |
| P29 | Kilo's tools | `llmprovider/providers/kilo/kilo.go:303` | `Tools:           req.Tools,` |
| P30 | the ChatGPT session live test | `llmprovider/live_chatgpt_session_test.go:1` | `//go:build live_gateways` |
| P31 | 0039's cacheable prompt | `llmprovider/live_0039_prompt_cache_measure_test.go:1` | `//go:build live_gateways` |

## Execution Record

No phase has run. Each phase appends its record here: what it did, its FAIL
and plant lines, its gate summary, and a dated entry for every deviation.
