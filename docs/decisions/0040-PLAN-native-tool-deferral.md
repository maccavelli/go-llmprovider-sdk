---
status: in-progress
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
  * `Tool.Deferred`, `Tool.Search` and ~~`FunctionCallOutputItem.Load`~~
    `ToolLoadItem` (0040-MADR A5) exist, the fields zero by default;
  * the four refusals of D1 fail before the network with `ErrInvalidRequest`,
    for every provider, through `llmtest` R23.
* **Each adapter sends its form** (D1's table):

  | Adapter, route | Form | When |
  | :--- | :--- | :--- |
  | `claude`, `opencode` Messages | D2 native | Claude 4.5 and later |
  | `openai` ChatGPT session | D3 native | every model |
  | `openai` key, `opencode` Responses | D3 native | `gpt-N.M` 5.4 and later |
  | `kilo` | ~~D4 OpenRouter protocol~~ D6 append on load (0040-MADR A2) | always |
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
| `llmprovider` | `Tool.Deferred`, `Tool.Search`, ~~`FunctionCallOutputItem.Load`~~ `ToolLoadItem` (A5) | 1 |
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
   * ~~`FunctionCallOutputItem.Load []string`.~~ `ToolLoadItem{CallID, Name}`
     (0040-MADR A5).
   * `validate` refuses, with `ErrInvalidRequest` and a message naming the
     tool or the name:
     * two `Search` tools;
     * a `Search` tool that is `Deferred`;
     * every tool `Deferred`;
     * a `Load` name that is not a `Deferred` tool in `Tools`;
     * `ForceTool` naming a deferred tool that no result in `Input` loads;
     * *(A5)* a `ToolLoadItem` with no earlier result for its call.
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
   * **`Loaded(input) []string`** returns the names the `ToolLoadItem`s bring
     in (A5), in first-load order, without repeats.
   * **`wire.RemapCallIDs`** gives each `ToolLoadItem` the id its call's
     result is sent with (A5), with a test that a remapped result keeps its
     load.
   * **`Tools(req, form) []llmprovider.Tool`** returns the tools a form
     sends:
     * all of them, for Messages native;
     * the non-deferred ones, for Responses native;
     * the non-deferred ones then the loaded ones, for `Append`;
     * every tool, for `Append` with `PreviousResponseID`.
3. **Change, Messages.**
   * `MessagesTools` sends `"defer_loading": true` on a deferred tool, when
     the form is native.
   * ~~`FromItems` sends a result with `Load` as a content list: one `text`
     block with `Output`, then one `tool_reference` per name.~~ *(0040-MADR
     A1.)* `FromItems` sends a result with `Load` as a `tool_result` whose
     content is one `tool_reference` per name, followed in the same user turn
     by a `text` block holding `Output`. This is the form Phase 0 measured.
   * `AddCache` puts the breakpoint on the last tool that is not deferred.
   * `claude` and `opencode`'s Messages route call `deferral.Form` and send
     its tools.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestForm_0040` | the D1 table, row by row, with the model rules at their edges: `claude-opus-4-1` is `Append`, `claude-haiku-4-5` `Native`, `gpt-5.3` `Append`, `gpt-5.4` `Native`, `minimax-m3` on Messages `Append` |
   | `TestLoaded_0040` | first-load order, without repeats, across several results |
   | `TestRemapCallIDs_ToolLoad_0040` | a `ToolLoadItem` follows its result's new id |
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
   * `Search` is an ordinary function, and a `ToolLoadItem` sends nothing
     (A5).
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

**Skipped (2026-10-13).** Phase 0 showed Kilo does not honour
`deferred_tools` (0040-MADR A2). Kilo takes Phase 4's append on load. The
steps below stay as written, for the record.

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
     owner's approval of the count: Claude, Zen Messages (A1's form, not yet
     measured through Zen), OpenAI, Zen Responses and Grok;
   * **a ChatGPT session path, which the owner runs** with
     `LLMPROVIDER_LIVE_CHATGPT=1` (0040-MADR A3). If it fails, the ChatGPT
     row takes D6 by amendment;
   * each path asserts the model called the loaded tool, and that the second
     and third requests read the first one's cache;
   * each check is seen failing on canned usages, offline, as 0039's were.
3. **Docs:**
   * `api-standards.md` R6;
   * `docs/architecture.md` (its deferral row, the importers' `Depends on`
     and the warning kind landed in Phase 4, deviation 1);
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
7. **gobble-cli.** Draft the issue: adopting `Deferred`, `Search` and
   `ToolLoadItem`
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

<!-- claims: self=3e81e3d -->

Checked against go-llmprovider-sdk `3e81e3d`, at that commit, by
`tools/claims.py`.

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

Each phase appends its record here: what it did, its FAIL and plant lines,
its gate summary, and a dated entry for every deviation.

### Phase 0: records and the live probe (2026-10-13)

The owner committed the records as `7abe504`, approved this PLAN's
execution, and approved the probe's 21 billed requests (Q5).

#### Deviations

* **Deviation 1 (2026-10-13): the claims checker's wiring.**
  * **Found.** Step 2 adopts 0041-PLAN Phase 0 step 2, which said the gate
    would run the checker, and that `make gate-selftest` would cover it.
    Research showed both wrong:
    * `gate-selftest` runs `scripts/test_gates.py`, the CI gates' self-test;
      `tools/` is tested by `make tools-test` (`Makefile:87-94`);
    * a record's claims are pinned to past revisions and to other
      checkouts. The gate's copy holds neither, and a checkout's path cannot
      be committed.
  * **Owner's decision:** "optimize claims.py for agentic use, it needs to
    be run both by agents and also human operators depending on the
    situation".
  * **Done:**
    * `tools/claims.py` is run by hand, never by the gate, which stays at 23
      checks;
    * each record names its revisions in a `<!-- claims: NAME=REV ... -->`
      marker, added to the four 0040 and 0041 records;
    * other repositories are found as checkouts beside this one, or beside
      its parent, so it runs with no arguments here;
    * `--repo` and `--rev` override, `--changed` checks the records `git
      status` lists, `--json` prints one object for an agent, and `--fixes`
      writes an `edit.py` spec moving each failing row to the one line its
      text is on;
    * it prints paths relative to the repository, never a machine path.
  * **Records.** 0041-PLAN Phase 0 step 2 is annotated. No MADR fact
    changes.

#### The claims checker (step 2)

* **Red:** `tools/test_tools.py` with the new `TestClaims`, on a scratch
  copy of `tools/` without `claims.py`:
  `ModuleNotFoundError: No module named 'claims'`.
* **Green:** `TestClaims`, 11 tests, ok; `make tools-test`, 40 tests, OK.
* **Plants** (`tools/plant.py`, 6 of 6 caught):

  | Plant | FAIL lines |
  | :--- | :--- |
  | the line compared off by one | `test_double_backtick_cell`, `test_json_output`, `test_other_repository_by_search_and_missing`, `test_passing_rows`, `test_pinned_revision_not_head` |
  | the file read at HEAD, not the record's revision | `test_json_output`, `test_passing_rows`, `test_pinned_revision_not_head`, `test_wrong_line_reports_where_found_and_writes_a_fix` |
  | a missing checkout passes | `test_other_repository_by_search_and_missing` |
  | a record with no rows is clean | `test_no_rows_is_a_finding` |
  | the fix spec names the old line | `test_wrong_line_reports_where_found_and_writes_a_fix` |
  | an escaped pipe kept escaped | `test_passing_rows` |
  | a marker quoted in prose is read | `test_marker_quoted_in_prose_is_not_the_marker` |

* **A fault the tool found in itself.** Its first run on the amended
  records failed 0041-PLAN with "claims marker: '...' is not NAME=REV". The
  annotation of 0041-PLAN Phase 0 quotes the marker's syntax in prose,
  before the real marker, and the tool took the first match. A new test,
  `test_marker_quoted_in_prose_is_not_the_marker`, failed the same way (red).
  The fix accepts a marker only alone on its line. The test passed, and its
  plant above was caught: 7 of 7.

* **On the records,** with no arguments: 0040-MADR 25 of 25, 0040-PLAN 31
  of 31, 0041-MADR 57 of 57, 0041-PLAN 39 of 39.

#### The probe (steps 3 to 7)

* **Keys,** checked for presence only: Anthropic, OpenAI, OpenCode, Kilo
  and xAI present. `LLMPROVIDER_LIVE_CHATGPT` absent: see deviation 3.
* **The file:** `llmprovider/live_0040_probe_test.go`, tagged
  `live_gateways`. It holds `TestLive_0040Probe`, `TestLive_0040ProbeFollowUp`
  and the offline `TestProbeChecks_0040`. It reuses 0039's
  `prompt0039`, with each path's own first line.
* **The checks, seen failing** (`tools/plant.py`, with `-tags
  live_gateways`). The first run left one plant uncaught: the "refused" case
  still reported a problem, from its missing tool call. The test was
  tightened so each case names the problem it must report, and all three
  plants were then caught:

  | Plant | FAIL line |
  | :--- | :--- |
  | a failed request is not a problem | `refused: problems ["step 1 called \"\", want the search \"tool_search\""], want one containing "step 1: HTTP 400: bad"` |
  | the loaded tool is not checked | `not loaded: problems [], want one containing "want the loaded tool"` |
  | the cache read is not checked | `no cache: problems [], want one containing "step 2 read nothing"` |

* **The run,** `LLMPROVIDER_LIVE_0040_PROBE=1`, 16 billed requests:

  | Path | Step 1 | Step 2 | Step 3 |
  | :--- | :--- | :--- | :--- |
  | Claude, `claude-haiku-5-5` | 200, `tool_search` | **400:** "Tool definitions/code execution functions cannot be mixed with other content" | not sent |
  | Zen Messages, `claude-haiku-5-5` | 200, `tool_search` | **400:** "Upstream request failed: [invalid_request_error] Tool definitions/code execution functions cannot be mixed with other content" | not sent |
  | OpenAI, `gpt-6-luna` | 200, `tool_search`, cached 0 | 200, `get_weather` `{"location":"Paris"}`, cached 4,498 | 200, answer, cached 4,675 |
  | Zen Responses, `gpt-6-luna` | 200, `tool_search`, cached 0 | 200, `get_weather`, cached 4,499 | 200, answer, cached 4,680 |
  | Kilo, `anthropic/claude-haiku-5.5` | 200, `tool_search`, cached 0 | 200, `get_weather`, cached 7,741 | 200, answer, cached 7,864 |
  | Grok, `grok-4.6` | 200, `tool_search`, cached 512 | 200, `use_tool` naming `get_weather`, `{"location":"Paris"}`, cached 5,248 | 200, answer, cached 5,376 |

* **The follow-up,** `LLMPROVIDER_LIVE_0040_FOLLOWUP=1`, 4 billed requests:

  | Path | Result |
  | :--- | :--- |
  | Claude, text beside the `tool_result` | 200 `tool_search`; 200 `get_weather` `{"location":"Paris"}`, cached 7,678; 200 `end_turn`, cached 7,836 |
  | Kilo, list the callable tools | 200: `read_file, tool_search, get_weather, lookup_stock` |

* **Billed in all:** 20 of the owner's 21.

#### Deviations, continued

* **Deviation 2 (2026-10-13): Anthropic refuses text beside
  `tool_reference` in one `tool_result`.** Foreseen by step 7. The
  follow-up measured the alternative step 7 names, the text as its own block
  after the `tool_result`, and it held the cache. **0040-MADR A1** records
  it, and Phase 2's step is annotated. No owner choice was needed: step 7
  fixed this outcome in advance.
* **Deviation 3 (2026-10-13): Kilo does not honour `deferred_tools`.**
  Foreseen by step 7. The first run could not tell, since its prompt sends
  the model to the search first, so one discriminating request was added
  within the approval. **0040-MADR A2** records it. Phase 6 is skipped, and
  the Goal's Kilo row is annotated.
* **Deviation 4 (2026-10-13): the ChatGPT path was not run.** Its
  credentials come from the Codex CLI's auth file, which only the owner opts
  into reading. Owner's choice: codex's source is the evidence now, and Phase
  7's live test adds a ChatGPT path the owner runs. **0040-MADR A3** records
  it, and Phase 7's step is annotated.

#### Checks (step 8)

* `make pre-add-check FILES="llmprovider/live_0040_probe_test.go"`: the
  first run found two `staticcheck` S1011 loops; collapsed to
  `append(input, output...)`, then clean. The probe's requests are unchanged
  by that edit, and `TestProbeChecks_0040` passed again with the tag.
* `make tools-test`: OK. `tools/claims.py --changed`: 25, 31, 57 and 39 of
  each record's rows pass.
* `make records-check`, `tools/linkcheck.py`, `tools/denyscan.py --changed`:
  clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 0 as `0af9ac7`.

### Phase 1: the contract (2026-10-13)

#### Deviations

* **Deviation 5 (2026-10-13): `FunctionCallOutputItem.Load []string` broke
  compatibility.**
  * **Found** by `make api-check` after the change:
    `api-check: incompatible: ./llmprovider.FunctionCallOutputItem: old is
    comparable, new is not`. A slice field makes the item incomparable, so
    `==` on items, and an item as a map key, stop compiling, and `==` on two
    `Item` interfaces holding one panics. R48 forbids it.
  * **Owner's decision:** a new item type, one per tool, over an opaque
    handle or a comma-separated string.
  * **Done:** `ToolLoadItem{CallID, Name string}`, after its call's result in
    `Input`, recorded as **0040-MADR A5**. The field was removed. `validate`
    gains a sixth refusal: a load with no earlier result for its call.
  * **Found while doing it:** `wire.RemapCallIDs` rewrites call ids for
    Anthropic and does not know the new item, so a remapped result would lose
    its load once Phase 2 attaches loads by id. **Phase 2 gains that step and
    its test.** Nothing sends a load before Phase 2, so Phase 1 is correct
    without it.
  * **Records:**
    * the MADR's D1, its refusals and its consequence on `Item` types are
      struck through and annotated;
    * this PLAN's Goal, exported additions, Phase 1, Phase 2, Phase 4 and
      Phase 7 steps are annotated;
    * `api-standards.md` R6 and `architecture.md` name `ToolLoadItem`.

#### Research

* `validate` (P3) runs before `Capabilities.Check`'s capability test, so every
  refusal reaches every provider before the network, whatever its tool
  support.
* `llmtest`'s R23 (P4, P5) runs in each provider's `TestConformance`.
* Every encoder's item switch skips an unknown type. `llmtest`'s R9 check of
  a response's output fails on one, which is right: no response holds a
  `ToolLoadItem`.

#### Tests, change and checks

* **Red,** with the fields added and no validation,
  `TestRequest_Deferral_0040`:

  ```text
  forcing an unloaded: <nil>; want ErrInvalidRequest containing "\"get_weather\" is deferred"
  two search tools: <nil>; want ErrInvalidRequest containing "\"read_file\" and \"tool_search\""
  a deferred search: <nil>; want ErrInvalidRequest containing "search tool \"tool_search\" is deferred"
  every tool deferred: <nil>; want ErrInvalidRequest containing "every tool is deferred"
  loading an unknown: <nil>; want ErrInvalidRequest containing "loads \"send_email\""
  loading a direct tool: <nil>; want ErrInvalidRequest containing "loads \"read_file\""
  ```

  Before that, the test did not compile: `unknown field Search in struct
  literal of type Tool`.
* **Change:**
  * `Tool.Deferred` and `Tool.Search` (`provider.go`);
  * `ToolLoadItem` and its kind (`item.go`, `constants.go`);
  * `validate` accepts the new item, and `validateDeferral` refuses six
    cases (`contract.go`);
  * `llmtest` R23 gains six cases.

  The expected text for the forced case was aligned to the message the code
  gives, "forces \"get_weather\", which is deferred and not loaded".
* **Green:** `TestRequest_Deferral_0040` passes, with eight accepted shapes
  and eight refused. Every provider's `TestConformance`, R23 included,
  passes.
* **Plants** (`tools/plant.py`, the contract test and Claude's
  `TestConformance`): 8 of 8 caught.
  * The first plant of "two search tools" deleted a block. That left a
    variable unused, so it caught nothing but a build failure. It was
    replaced by a compiling plant, `if false && search != ""`.

  | Plant | FAIL lines |
  | :--- | :--- |
  | a deferred search tool accepted | `a deferred search: <nil>; want ErrInvalidRequest containing …`; `R23 (invalid values): a deferred search tool returned <nil>`; `… sent 1 request(s); want none` |
  | two search tools accepted | `two search tools: <nil>; …`; `R23 …: two search tools returned <nil>`; `… sent 1 request(s)` |
  | every tool deferred accepted | `every tool deferred: <nil>; …`; `R23 …: every tool deferred returned <nil>`; `… sent 1 request(s)` |
  | a load of no deferred tool accepted | `loading an unknown: <nil>; …`; `loading a direct tool: <nil>; …`; `R23 …: a load of no deferred tool returned <nil>` |
  | a load with no earlier result accepted | `a load before its result: <nil>; …`; `a load with no result: <nil>; …`; `R23 …: a load with no earlier result returned <nil>`; `… sent 1 request(s)` |
  | a forced unloaded deferred tool accepted | `forcing an unloaded: <nil>; …`; `R23 …: a forced, unloaded deferred tool returned <nil>` |
  | `ToolLoadItem` refused as an unknown item | `deferred, loaded: llmprovider: invalid request: input 3 is llmprovider.ToolLoadItem; …`, and five more |

* **API:** `make api-check`, 0 incompatible changes against `v1.12.0`.
  `tools/apidiff.py v1.12.0` reports `Tool.Deferred`, `Tool.Search` and
  `ToolLoadItem`, each `added`, and nothing else.
* **Wire:** no encoder sends anything for the new fields or item yet, and
  every existing golden is unchanged (the gate's `g-wire-stable`).
* **Checks:**
  * `make pre-add-check` on the six Go files: clean;
  * `make records-check`: clean;
  * `tools/claims.py --changed`: every row passes;
  * **gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 1 as `12a921c`.

### Phase 2: the shared form, and Anthropic Messages (2026-10-13)

#### Research

* Only `FromItems` sets a Messages block's `Content`
  (`internal/wire/messages/messages.go`). One existing test reads it,
  `callid_0026_test.go:52`, which moved from `deref(b.Content)` to
  `b.Content.Text()`, with the same assertion (R46).
* No test counts the Messages encoder's allocations. A tool result's content
  still became a value type, `ResultContent`, like the message's `Content`,
  with `omitzero`, so no interface is encoded (0028-PLAN D16).
* `wire.Arena` grows past its size and keeps every pointer it gave
  (`internal/wire/arena.go`), so the loads' extra strings are safe.
* `MessagesTools` and `FromItems` each have two callers, Claude's body and
  OpenCode's messages route. Each keeps its old behaviour, and a sibling
  does the native form: `MessagesToolsNative` and `FromItemsLoading`. No
  existing test changes.

#### What was built

* **`llmprovider/internal/wire/deferral`:**
  * `Of(wire, model)` returns D1's form. `Wire` names `Messages`,
    `Responses`, `ChatGPT`, `Grok` and `Other`, in place of the four
    arguments step 2 sketched.
  * The Claude rule reuses `claudeVersionRE`, through a new
    `messages.ClaudeVersion`, at 4.5 or later.
  * The GPT rule is `gpt-N.M` at 5.4 or later. An unreadable id is
    `Append`.
  * `Loaded`, `SearchLoads` (for Phase 3), `Tools` and `Deferring`.
  * A request with nothing deferred gets its own `Tools` slice back.
* **Messages:**
  * `FromItemsLoading`: a loading result's `tool_result` holds one
    `tool_reference` per load, and its text follows the turn's results as a
    `text` block (A1). A result with no text adds no block.
  * `MessagesToolsNative` sends `defer_loading` on deferred tools.
  * `AddCache` marks the last tool that is not deferred.
* **`wire.RemapCallIDs`** gives a `ToolLoadItem` its result's id (A5).
* **`claude` and `opencode`'s messages route** take the form, the tools and
  the messages from it.

#### Tests, red and green

* **Red:**
  * `deferral`: `undefined: Wire`, then the package added;
  * `messages`: `undefined: FromItemsLoading`;
  * `wire`: `undefined: MessagesToolsNative`;
  * Claude, before the wiring:

    ```text
    deferral_0040_test.go:78: tools [read_file tool_search get_weather lookup_stock], deferred map[]; want all four, the last two deferred
    deferral_0040_test.go:81: cache marker on "lookup_stock", want tool_search, the last tool not deferred
    deferral_0040_test.go:100: tools [read_file tool_search get_weather lookup_stock], want read_file, tool_search, get_weather
    deferral_0040_test.go:103: cache marker on "lookup_stock", want the last tool sent, get_weather
    ```

  * OpenCode's test was written after its wiring, so it has no red run of
    its own. Its plant, the route forced to `Append`, is what shows it
    fails (below).
* **Green,** every one passing:
  * `TestForm_0040`, 25 rows;
  * `TestLoaded_0040`, `TestTools_0040` and `TestSearchLoads_0040`;
  * `TestFromItemsLoading_Load_0040`, `…_Unchanged_0040`, `…_EmptyOutput_0040`
    and `…_RemappedID_0040`;
  * `TestAddCache_Deferred_0040`;
  * `TestMessagesToolsNative_0040` and `TestRemapCallIDs_ToolLoad_0040`;
  * Claude's `TestDeferral_Native_0040`, `…_Append_0040` and `…_None_0040`;
  * OpenCode's `TestDeferral_MessagesRoute_0040`.

  The planned names `TestMessagesTools_Deferred_0040` and
  `TestFromItems_Load_0040` became the names above, after the siblings.
* **Every existing test** in `internal/wire/...`, `claude` and `opencode`
  passes, the G-wire goldens included.
* **Coverage:** `deferral` 97.0%; `make coverage-check`, 32 packages, 0
  problems.

#### Plants (`tools/plant.py`, 8 of 8 caught)

The first run left one plant uncaught: the load given its own call id
instead of its result's. For a single call the two ids are the same string,
so the test could not tell them apart. `TestRemapCallIDs_ToolLoad_0040` gained a
repeated id, where the second call is renamed. It then caught the plant:
`the load's id "fn_1", want the second result's "fn_1_2" (the first's is
"fn_1")`.

| Plant | FAIL lines |
| :--- | :--- |
| the Claude version edge moved to 4.6 | `Of(1, "claude-haiku-4-5") = 0, want 1`, and two more |
| a repeated load listed twice | `Loaded = [lookup_stock get_weather send_email get_weather], want [lookup_stock get_weather send_email]` |
| `defer_loading` sent when the form appends | `defer_loading sent on map[get_weather:true]` |
| the loading result's text dropped | `FromItemsLoading:` diff; Claude's `last turn:` diff; OpenCode's `want a tool_reference result and a text block` |
| the text sent before the turn's other results | `FromItemsLoading:` diff |
| the breakpoint on a deferred tool | `tool 1 (b): marked false; want only b marked`; `a deferred tool was marked`; `cache marker on "lookup_stock", want tool_search` |
| a load keeps its old call id | `the load's id "fn_1", want the second result's "fn_1_2"` |
| OpenCode's messages route always appends | `claude-haiku-5-5: deferred [] of 3 tools; want get_weather and lookup_stock of 4` |

#### Checks

* **`make pre-add-check`** on the 13 Go files. The first run found five
  issues, each fixed:
  * two `strconv.Atoi` errors not checked, now read by `gptVersion`, where
    an unreadable id gives `Append`;
  * an unchecked type assertion in `AddCache`;
  * two names differing only in case: the test helper `tools()` became
    `deferralTools()`, and `fromItems` became `encodeItems`.

  The plants were rerun on the fixed code, 8 of 8 caught, and the check was
  then clean.
* `make records-check` and `tools/claims.py --changed`: clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed, `g-wire-stable` included: no
  existing golden changed.

The owner committed Phase 2 as `338dedf`.

### Phase 3: OpenAI Responses (2026-10-13)

#### Research

* **`outputItem.Arguments` was a `string`.** A `tool_search_call` sends its
  `arguments` as a JSON object (MADR §3; codex's `ToolSearchCall`, E13), so
  a reply holding one would not have decoded at all. It is now a
  `json.RawMessage`, read as a string's content (every function call, as
  before) or as the object, compacted.
* **The decoder had no request.** `appendOutput` cannot know the Search
  tool's name. `responses.Events` is passed to `wire.StreamEvents` as a
  function value (`openai.go`, `grok.go`, `opencode.go`), so a variadic
  option would have changed its type.
  * A `responses.Decoder{Search}` value now carries the name, with
    `DecodeFor`, `ReadStream` and `Events` methods.
  * The package functions delegate with no name, so every other caller is
    untouched.
* **`InputItem` keeps its byte-for-byte order** (0028-MADR D-A6).
  * `Arguments` became a value type, `Arguments`, that writes a string or a
    JSON object, with `omitzero`.
  * `execution`, `status` and `tools` were added at their alphabetical
    places, each omitted when empty.
  * The existing byte tests (`typed_0028_test.go`, the goldens) pass
    unchanged.

#### Found while building

* **A forced choice of the Search tool cannot be native on Responses.**
  `tool_choice` names a function, and the search is a `tool_search` entry,
  with no name. `deferral.ForRequest` gives such a request `Append`, where
  the search is a function the choice can name. 0040-MADR is silent on
  forced choices, so this changes no decision. Each provider's package doc
  says so in Phase 7.
* **A hosted search** (`execution: "server"`) is the service's own, which
  D8 leaves out. It is dropped, as every unknown item is, even with a Search
  tool named.
* **Arguments that are not a JSON object** replay as `{}`, which the API
  takes, rather than a string it would refuse.

#### What was built

* **`responses`:**
  * `Decoder`;
  * `InputLoading(items, req)`, which sends a call to the Search tool as a
    client `tool_search_call` and its result as a `tool_search_output`
    holding the loaded definitions with `defer_loading`. The result's text
    is not sent;
  * `appendOutput` decodes a client `tool_search_call` as a
    `FunctionCallItem` naming the Search tool.
* **`wire.AddResponsesToolsNative`:** the Search tool as `{"type":
  "tool_search", "execution": "client", "description", "parameters"}`, with
  no name or `strict`.
* **`deferral`:** `ForRequest` and `SearchName`.
* **`openai`:** `deferForm` gives the form and the decoder once. The body,
  `Generate` and `Stream` all use it. A ChatGPT session takes the native
  form for every model (A3).
* **`opencode`'s responses route:** its body, its decoder and its stream.

#### Tests, red and green

* **Red:**
  * `responses`: `undefined: Decoder`, `undefined: InputLoading`;
  * `wire`: `undefined: AddResponsesToolsNative`;
  * `deferral`: `undefined: ForRequest`;
  * OpenAI, before the wiring, after two fixtures were given a text answer
    (an empty `output` is rightly refused as "the answer has no content"):

    ```text
    TestDeferral_KeyNative_0040: llmprovider: incomplete response: stop: responses: the answer has no content
    TestDeferral_KeyAppend_0040: tools [read_file tool_search get_weather lookup_stock]; want read_file, tool_search and the loaded get_weather
    TestDeferral_KeyStream_0040: llmprovider: incomplete response: stop: responses: the answer has no content
    TestDeferral_ChatGPT_0040: llmprovider: incomplete response: stop: responses: the answer has no content
    TestDeferral_ForcedSearch_0040: tools [read_file tool_search get_weather lookup_stock]; want the append form
    ```

    The search call was dropped, leaving no content, and the appending
    models sent every tool;
  * OpenCode's responses route: `the answer has no content`.
* **One test fixed while green:** `TestInputLoading_Unchanged_0040` first
  failed for flagging any `"tool_search"`. Plain `Input` rightly sends a
  call to a function of that name. It now checks that no
  `tool_search_call` or `tool_search_output` is sent, and that the load is
  dropped.
* **Green:**
  * `TestDecode_ToolSearchCall_0040`, with a hosted search added after the
    plant review below;
  * `TestEvents_ToolSearchCall_0040`;
  * `TestInputLoading_Replay_0040`, `…_Unchanged_0040` and
    `…_BadArguments_0040`;
  * `TestAddResponsesToolsNative_0040` and `TestForRequest_0040`;
  * OpenAI's `TestDeferral_KeyNative_0040`, `…_KeyAppend_0040`,
    `…_KeyStream_0040`, `…_ChatGPT_0040` and `…_ForcedSearch_0040`;
  * OpenCode's `TestDeferral_ResponsesRoute_0040`;
  * every other test in `./llmprovider/...`.

#### Plants (`tools/plant.py`, 10 of 10 caught)

Reviewing the plants before running them found one untested path: a hosted
search, which no test sent. `TestDecode_ToolSearchCall_0040` gained it. One
plant, "the search's arguments sent as a string", first left a variable
unused and failed only to build. It was rerun in a compiling form and
caught.

| Plant | FAIL lines |
| :--- | :--- |
| the search sent as a function | `search entry … type:function`; `search entry sent a name`; `search entry sent strict` |
| the call decoded under its wire name | `with a search tool: [{CallID:c1 Name:tool_search_call …`, and four more across the wire and both providers |
| a hosted search decoded | `a hosted search: [{CallID:c1 Name:tool_search …} …], want only …` |
| the stream ignores the search name | `the answer has no content`, in four stream tests |
| `Output` sent in `tool_search_output` | `InputLoading:` diff |
| the search's arguments sent as a string | `InputLoading:` diff; `bad arguments: … "arguments":"not json" …` |
| a deferred tool left at the top level | `tools [read_file tool_search get_weather lookup_stock]; want read_file and a tool_search entry`, and three more |
| a forced search kept native | `search forced on Responses: 2, want Append`; `tools [read_file tool_search]; want the append form` |
| a ChatGPT session ruled as a key | `Of(3, "gpt-5") = 0, want 2`; the ChatGPT provider test |
| OpenCode's responses route never native | `tools [read_file tool_search get_weather]; want read_file and a tool_search entry` |

#### Checks

* `make pre-add-check` on the 10 Go files. The first run found one
  unchecked type assertion in `AddResponsesToolsNative`, now returning when
  no tools were added. The plants were rerun on the fixed code, 10 of 10,
  and the check was then clean.
* `make records-check` and `tools/claims.py --changed`: clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed, `g-wire-stable` included: no
  existing golden changed.

### Phase 4: append on load, and the warning (2026-10-13)

#### Research

* **P23, P24.** `chatcompletions.Opts.Tools` goes through `toolList`
  unchanged, so the Chat Completions package does not change: each caller
  passes `deferral.Tools` as `Opts.Tools`.
* **P25, P26.** Gemini's Interactions encoder and OpenCode's Google route
  loop over `req.Tools`; both now loop over `deferral.Tools`.
* **P27.** The Chat Completions callers are Kilo, Together, Hugging Face,
  Ollama and OpenCode's chat route.
* **P28.** `Warning` is `Kind`, `Model` and `Detail`. Every provider sends
  through `wire.Post` (directly, or through `wire.Send`) or
  `wire.StreamEventsRetry` (through `StreamEvents` and `StreamEventsFor`).
  A field on `wire.Call` therefore reaches every response from the one place
  that builds the request, and no provider appends after decoding. A
  provider's own later warnings, such as the forced-choice ones on the
  messages route, still follow it.
* **A `ToolLoadItem` on an appending wire.** Each encoder's type switch has
  no case for it, so it sends nothing. The provider test now proves this on
  every wire, and five plants confirm it (below).

#### What was built

* **`llmprovider`:** `WarningDeferralEmulated` (`"deferral_emulated"`).
* **`deferral`:**
  * `Warning(form, req, model)`: for `Append` ("each loaded tool was
    appended", or "every deferred tool" with `PreviousResponseID`), for
    `Dispatch` ("use_tool"), and for `ResponsesNative` when a result other
    than the search's loaded a tool (it names them);
  * `Warnings`, the same as a slice for `wire.Call`.
* **`wire.Call.Warnings`:** `Post` appends them to a decoded `*Response` on
  success, and `StreamEventsRetry` appends them to the `EventDone`
  response. A rebuilt request's call is read again.
* **The providers:**
  * Gemini, Hugging Face, Kilo, Ollama and Together send
    `deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req))`
    and set `Call.Warnings`;
  * Claude and OpenAI set `Call.Warnings` from the form their body already
    used;
  * OpenCode has one `call.deferForm()` per route. The four bodies, the
    responses decoder and `wireCall` all use it.
* **Grok** is unchanged; its dispatch is Phase 5's.

#### Found while building

* **Test names.** The PLAN's `TestAppend_Order_0040` and
  `TestWarning_Emulated_0040` are the two checks of one table,
  `TestDeferral_PerProvider_0040`. It runs 13 provider and model rows, each
  by `Generate` and by `Stream`. `TestAppend_PreviousResponseID_0040` keeps
  its name. Its rows are OpenAI and Gemini (with `WithStore(true)`), the
  providers that take a continuation; Grok's is Phase 5's. Nothing decided
  changes.
* **A reply cannot read the request body**, because `wiretest` reads it
  first, and `wirecase` serves no stream. The stream half therefore runs
  against a second server, which serves each wire's smallest complete
  stream by path.
* **`docs/architecture.md`** has no row for `internal/wire/deferral`,
  added in Phase 2. Its `Response.Warnings` passage does not list the new
  kind either. This PLAN puts `architecture.md` in Phase 7, but AGENTS.md
  wants the layout updated in the same change (R43). ~~Not done in this
  phase; raised with the owner.~~ Done in this phase: deviation 1 below.
* **The package docs** (R12) and `api-standards.md` stay in Phase 7, as
  planned.

#### Deviations

1. **2026-10-13: `docs/architecture.md` in Phase 4, not Phase 7.**
   * **Found:** the package table had no row for `internal/wire/deferral`
     (Phase 2), no package's `Depends on` named it, and the
     `Response.Warnings` passage did not name the new kind. AGENTS.md wants
     the layout updated in the same change (R43).
   * **Decided by the owner:** fix it now. `docs/architecture.md` joins
     Phase 4's files.
   * **Done:**
     * a row for `internal/wire/deferral`;
     * the package added to the `Depends on` of the eight providers that
       import it (Grok does not yet) and of `internal/wire/responses`;
     * `WarningDeferralEmulated` named, with how it reaches a response.
   * **Left to Phase 7:** each wire package's list of exported names
     (`FromItemsLoading`, `InputLoading`, `Decoder`, `ClaudeVersion`, the
     native tool lists), with the package docs (R12). No MADR fact or
     decision changes.

#### Tests, red and green

* **Red, the mechanism.** Reproduced on a scratch clone of the working tree,
  with `warning.go`, `post.go`, `sse.go` and `deferral.go` put back to
  `5c9f46b` in the clone only:

  ```text
  warnings_0040_test.go:22:48: undefined: llmprovider.WarningDeferralEmulated
  warnings_0040_test.go:24:90: unknown field Warnings in struct literal of type Call
  warning_0040_test.go:33:12: undefined: Warning
  warning_0040_test.go:38:12: undefined: Warnings
  ```

* **Red, the providers**, run before they were wired:
  * The first run used `wirecase`'s replies for both halves, and every
    `Stream` failed: the stream ended before its end event, and OpenCode's
    Google route got a 404. That is the finding above.
  * With the stream server, the native rows passed (`claude-haiku-5-5`,
    `gpt-6-luna`, and OpenCode's `claude-sonnet-5` and `gpt-5.5`).
    `claude-opus-4-1` and `gpt-4.1-mini` already appended (Phases 2
    and 3), but had no warning:

    ```text
    claude/claude-opus-4-1: Generate: WarningDeferralEmulated false, want true (warnings [])
    openai/gpt-4.1-mini: Generate: WarningDeferralEmulated false, want true (warnings [])
    ```

  * Every other row sent every tool and no warning, by both halves:

    ```text
    kilo/openai/gpt-6-luna: Generate: tools [read_file tool_search get_weather lookup_stock], want [read_file tool_search get_weather]
    kilo/openai/gpt-6-luna: Generate: WarningDeferralEmulated false, want true (warnings [])
    ```

    The same pair failed for OpenCode's `glm-5.3` (chat route) and
    `gemini-3.8-flash` (Google route), and for Gemini, Hugging Face,
    Together and Ollama.
* **Tightened while green.**
  * After wiring, the request was strengthened: two loads, in the reverse
    of their declared order, and a deferred tool that is never loaded.
  * The append rows also check that the conversation (`messages`, `input`
    or `contents`) equals the one sent with the loads taken out.
  * `TestAppend_PreviousResponseID_0040` was added.
  * None of these was red first. Each is shown failing by the plants below.
* **Green:**
  * `TestCallWarnings_0040` (`wire`);
  * `TestWarning_0040` (`deferral`), now also checking `Warnings`;
  * `TestDeferral_PerProvider_0040`, `TestAppend_PreviousResponseID_0040`
    and `TestDeferral_NoWarningWithoutDeferral_0040` (`providers`);
  * every other test in `go test ./...`.

#### Plants (`tools/plant.py`, 20 of 20 caught)

Every plant compiles. Each was caught by a test assertion, not by a build
failure. `plant.py` prints a plant's first 12 FAIL lines, so a row names the
rows those lines show, not every row that failed.

| Plant | FAIL lines |
| :--- | :--- |
| load order reversed (`Loaded`) | `Generate: tools [read_file tool_search get_weather lookup_stock], want [read_file tool_search lookup_stock get_weather]`, on Claude 4.1, GPT-4.1 and OpenCode's chat and Google routes, by both halves |
| a deferred tool sent before its load (`Tools` returns every tool under `Append`) | `tools [… send_email], want [read_file tool_search lookup_stock get_weather]`, on the same rows |
| Together sends `req.Tools` | `together/openai/gpt-oss-120b: … tools [… send_email], want …`, by both halves |
| OpenCode's chat route sends `req.Tools` | `opencode-zen/glm-5.3: … tools [… send_email], want …` |
| OpenCode's Google route sends `req.Tools` | `opencode-zen/gemini-3.8-flash: … tools [… send_email], want …` |
| OpenCode's other routes ruled native | the two rows above: tools, and `WarningDeferralEmulated false, want true` |
| a continuation sends only `Input`'s loads | `openai/gpt-4.1-mini: tools [read_file tool_search], want [… send_email]`, and Gemini's; `TestTools_0040`'s continuation case |
| a continuation's warning says "appended" | `warning "… each loaded tool was appended …", want one containing "every deferred tool"`; `TestWarning_0040`'s "appended, continued" |
| the warning on Messages native | `claude/claude-haiku-5-5: WarningDeferralEmulated true, want false`, and OpenCode's `claude-sonnet-5`; `TestWarning_0040: Messages native: warned true` |
| the warning on every Responses native request | `openai/gpt-6-luna: … true, want false`, OpenCode's `gpt-5.5`, the continuation's `gpt-6-luna` row; `Responses, a search's loads only: warned true` |
| Kilo sets no warning | `kilo/openai/gpt-6-luna: … false, want true`, by both halves |
| OpenCode sets no warning | `glm-5.3` and `gemini-3.8-flash`: `… false, want true` |
| Claude sets no warning | `claude/claude-opus-4-1: … false, want true` |
| `Post` drops the call's warnings | `Generate: WarningDeferralEmulated false, want true` on six rows |
| a stream drops the call's warnings | `Stream: done … Warnings:[] …; want WarningDeferralEmulated true` on six rows |
| Chat Completions sends a load | `the loads sent something: […]` on Kilo, Hugging Face, Together, Ollama and OpenCode's chat route |
| Messages sends a load | `claude/claude-opus-4-1: the loads sent something` |
| Responses sends a load | `openai/gpt-4.1-mini: the loads sent something` |
| Interactions sends a load | `gemini/gemini-3.8-flash: the loads sent something` |
| generateContent sends a load | `opencode-zen/gemini-3.8-flash: the loads sent something` |

#### Checks

* `make pre-add-check` on the 15 Go files: clean.
* `make api-check`: `0 incompatible change(s)`. `tools/apidiff.py v1.12.0`
  lists four compatible additions: `Tool.Deferred`, `Tool.Search`,
  `ToolLoadItem` and `WarningDeferralEmulated`.
* `tools/claims.py` on both records: 25 of 25, and 31 of 31.
* `make records-check`, `tools/denyscan.py --changed` and
  `tools/linkcheck.py`: clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed, `g-wire-stable` included: no
  existing golden changed.
