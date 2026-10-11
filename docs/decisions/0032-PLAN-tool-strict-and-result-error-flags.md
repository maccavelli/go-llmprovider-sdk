---
status: in-progress
date: 2026-10-11
associated-madr: "0032-MADR-tool-strict-and-result-error-flags.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Per-Tool Strict Flag and a Tool Result's Error Flag

Associated MADR: [0032-MADR-tool-strict-and-result-error-flags.md](0032-MADR-tool-strict-and-result-error-flags.md)

## Goal

When it is done, every decision of the MADR holds, each proven red first,
and `v1.7.0` is released:

* **D1:** `llmprovider.Tool` has `Strict *bool`.
* **D2:** a set `Strict` is sent as `strict` on the Responses wire and as
  `function.strict` on the Chat Completions gateways Phase 1 shows
  accepting it. It is left out elsewhere, and each package doc says so.
* **D3:** `llmprovider.FunctionCallOutputItem` has `IsError bool`. It is
  sent as:
  * Messages' `is_error`;
  * generateContent's `{"error": …}`;
  * Interactions' error form if Phase 1 finds one.

  Elsewhere the output is sent unchanged, and each package doc says so.
* **D4:** the G-wire goldens are unchanged; new wirecase scenarios and
  `llmtest`'s Fidelity check pin both flags.

## Scope

### In scope

| Phase | Decisions | Files (production) |
| :--- | :--- | :--- |
| 0 | records | this PLAN, 0032-MADR, `docs/README.md` |
| 1 | measurement for D2, D3 | `llmprovider/live_0032_measure_test.go` (new, `live_gateways`) |
| 2 | D1, D2 | `llmprovider/provider.go` (`Tool`), `internal/wire/tools.go`, `internal/wire/chatcompletions`, the provider packages' docs |
| 3 | D3 | `llmprovider/item.go`, `internal/wire/messages`, `internal/wire/generatecontent`, `providers/gemini/interactions.go`, the provider packages' docs |
| 4 | D4 | `internal/wirecase`, `llmprovider/llmtest/llmtest.go` (Fidelity) |
| 5 | docs, release `v1.7.0` | `docs/architecture.md`, `README.md`, `docs/guides/api-standards.md` if a rule's text no longer holds |

### Out of scope

* Anthropic `strict`, unless Phase 1 shows a documented field: then an
  amendment.
* gobble-cli's own adoption: its 0012-MADR D3's opt-out, and its adapter
  passing `IsError`. Those are its records.
* The other gobble-cli items (model metadata, image input, prompt caching,
  tool deferral, timeouts).
* Push and tags, which are the owner's.

## Rules for every phase

The rules of
[0031-PLAN-native-streaming-on-the-responses-wire.md](0031-PLAN-native-streaming-on-the-responses-wire.md)
apply unchanged:

1. approval, naming billed requests;
2. red first, and plants only on a scratch copy;
3. the 22-check gate;
4. coverage floors;
5. benchmarks, where a phase names them;
6. live checks: keys for presence only, nothing logged but statuses, shapes
   and redacted errors;
7. deviations stop and prompt;
8. one commit per phase, the owner's or on a same-turn ask;
9. no identifiers.

## Implementation Steps

### Phase 0: records

1. 0032-MADR answered (Q1 to Q4) and marked accepted; this PLAN
   `proposed`.
2. `docs/README.md` indexes both, 69 records.
3. `make records-check` clean; the owner commits.

### Phase 1: measurement (live; billed: at most 16 requests of at most 64 output tokens)

1. **T1, strict.** `TestLive_0032Strict`, in
   `llmprovider/live_0032_measure_test.go` (`live_gateways`; runs only when
   `LLMPROVIDER_LIVE_0032` is set).
   * **Requests:** for each of `openai`, `grok`, OpenCode Go's `responses`
     route, `huggingface`, `kilo`, `together` and OpenCode Go's
     `chat_completions` route, one request offering a tool whose
     definition a recording transport rewrites to carry `"strict": false`
     (Responses) or `function.strict: false` (Chat Completions).
   * **Logged:** the status; any error, redacted; and, where the reply
     echoes the tools (Responses' `response.created` or JSON body), the
     `strict` value the service applied.
2. **T2, a result's error.** `TestLive_0032ResultError`, same file and
   switch.
   * **Requests:** for each of `claude` (`is_error: true`), Google's
     generateContent endpoint through the OpenCode `google` route's request
     (`response: {"error": …}`), and `gemini` (an Interactions
     `function_result` carrying `is_error: true`), one two-turn request: a
     call, then its failed result.
   * **Logged:** the status, any error (redacted), and whether the reply's
     text acknowledges the failure (a yes or no, never the text).
3. **Rule (D2):** a target answering 2xx with the field sent, and where it
   echoes, applying it, carries `Strict`. A 4xx naming the field leaves it
   unsent.
4. **Rule (D3):** a wire answering 2xx with its error form carries
   `IsError`. A 4xx leaves the output unchanged on that wire.
5. Gate; record T1 and T2; stage the measurement test.

### Phase 2: `Tool.Strict` (D1, D2)

1. **Tests, red first:**
   * `wire.ResponsesTools`: nil sends no `strict`; true and false send it.
   * `chatcompletions.toolList`: the same for `function.strict`, behind an
     `Opts` switch set only by the gateways Phase 1 accepted.
   * Each provider: a tool with `Strict` set reaches its wire as Phase 1
     decided, or is left out.
   * `TestGenerate_Unchanged`-style tests: a tool with nil `Strict` sends
     the same body as at `v1.6.0`.
2. **Change:** the field, with its doc; the encoders; the Chat Completions
   switch; and each provider's package doc where `Strict` is not sent.
3. **Plants:** an encoder that sends `strict` for a nil `Strict`; one that
   drops a set one.
4. Gate; stage.

### Phase 3: `FunctionCallOutputItem.IsError` (D3)

1. **Tests, red first:**
   * Messages `is_error`;
   * generateContent `{"error": …}`;
   * Interactions, as Phase 1 decided;
   * Responses and Chat Completions unchanged;
   * a result with `IsError` false sends today's body.
2. **Change:** the field, with its doc; the encoders; the package docs.
3. **Plants:** an encoder that marks every result an error; one that drops
   the flag.
4. Gate; stage.

### Phase 4: proof (D4)

1. **wirecase:** new scenarios with a `Strict: false` tool and an `IsError`
   result, recorded per wire. The existing goldens are unchanged, and the
   new goldens are generated once and then pinned.
2. **llmtest Fidelity:** with `Fidelity` set, the check also sends a
   `Strict: false` tool and an `IsError` result, and asserts each reaches
   the wire where the harness's provider documents it. It needs no new
   field: a provider declares nothing, so the check compares against what
   the provider's wire recorded. The exact form is settled in the phase,
   and recorded. *(Amended by deviation D2, 2026-10-11: the check takes
   two `Harness` fields, `SendsStrict` and `SendsIsError`.)*
3. **Plants:** a Fidelity check that cannot fail is caught by planting a
   provider that drops both flags.
4. Gate; stage.

### Phase 5: documentation and release

1. **`docs/architecture.md`:** the contract section names both fields and
   the wires that carry them.
2. **Release notes** in this PLAN, as `v1.7.0`; `README.md`'s Status.
3. **The live suites** on the release commit; a transient failure is rerun
   once alone.
4. **The owner** commits, pushes, and tags `v1.7.0`.
5. **After the tag:**
   * CI;
   * the proxy;
   * apidiff `v1.6.0` to `v1.7.0`: the two fields, and deviation D2's two
     `Harness` fields, as additions;
   * a scratch consumer, and a scratch copy of gobble-cli on `v1.7.0`.
6. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with both lines in the Execution Record, and
  every plant's FAIL line.
* **V2. The gate** clean at each phase; the existing goldens unchanged.
* **V3. Measurement:** T1 and T2 recorded before Phases 2 and 3 use them.
* **V4. API:** apidiff from `v1.6.0` shows the two fields, as additions
  only. *(Deviation D2 adds `llmtest.Harness`'s `SendsStrict` and
  `SendsIsError`, also additions.)*
* **V5. Live:** the live suites on the release commit.
* **V6. Records:** 0032-MADR amended for anything Phase 1 changes; the
  package docs name every wire that does not carry a flag.

## Rollout and Rollback

* **Rollout.** Five commits after the records, then `v1.7.0`. gobble-cli
  moves under its own record.
* **Rollback, before the tag:** each phase reverts alone, newest first.
* **Rollback, after the tag:** fix forward in `v1.7.x`. A caller that sets
  neither field is unaffected.

## Execution Record

### Phase 0: records (2026-10-11)

The owner answered Q1 to Q4 with each recommended option, and approved
execution ("follow recommendations, proceed"):

* Q1: no pointer helper;
* Q2: Chat Completions `strict` on each gateway that accepts it;
* Q3: an unmarked output where a wire has no error field;
* Q4: one record.

0032-MADR is `accepted`, this PLAN `in-progress`, and the index has 69
records.

### Phase 1: measurement (2026-10-11)

`llmprovider/live_0032_measure_test.go`, run with `LLMPROVIDER_LIVE_0032=1`
and every key variable set (checked for presence only). A recording
transport rewrites each provider's own request to carry the flag, and logs:

* the reply's status;
* any error, redacted;
* for Responses, the `strict` value the reply's echoed tools carry;
* for T2, whether the reply's text says the call failed (yes or no, never
  the text).

Billed: 14 requests, of the 16 budgeted.

**T1, `strict: false`** (`TestLive_0032Strict`, 03:11 UTC). A tool with an
optional property, at most 64 output tokens:

| Target | Model | Status | Echoed `strict` |
| :--- | :--- | :--- | :--- |
| `openai` | `gpt-4.1-mini` | 200 | `false` |
| `grok` | `grok-4.6` | 200 | `false` |
| OpenCode Go `responses` | `gpt-6-luna` | 200 | `false` |
| `huggingface` | `deepseek-ai/DeepSeek-V4-Flash-0731` | 500 "Internal Server Error" | none (D1) |
| `kilo` | `deepseek/deepseek-v4.1-flash` | 200 | none (no echo) |
| `together` | `deepseek-ai/DeepSeek-V4.1-Flash` | 200 | none (no echo) |
| OpenCode Go `chat_completions` | `glm-5.3-flash` | 200 | none (no echo) |

**T1c, Hugging Face's control** (`TestLive_0032StrictHuggingFaceControl`; D1):

| Run | Model | With `strict: false` | Without it |
| :--- | :--- | :--- | :--- |
| 03:12 UTC | `deepseek-ai/DeepSeek-V4-Flash-0731` | 500 | 500 |
| 03:13 UTC (`LLMPROVIDER_LIVE_0032_HF_MODEL`) | `zai-org/GLM-5.3-Flash` | 200 | 200 |

**T2, a failed result** (`TestLive_0032ResultError`, 03:11 UTC). A call, then
its result "the city registry is offline" in the wire's error form:

| Target | Error form | Model | Status | Reply says it failed |
| :--- | :--- | :--- | :--- | :--- |
| `claude` | `tool_result.is_error: true` | `claude-haiku-4-5` | 200 | yes |
| Google's endpoint (OpenCode `google` route's request) | `functionResponse.response: {"error": …}` | `gemini-3.7-flash`, effort `low`, 1,024 tokens | 200 | yes |
| `gemini` | `function_result.is_error: true` | `gemini-3.7-flash`, effort `low`, 1,024 tokens | 200 | yes |

The "yes" is not proof the flag was read, since the output text also says
the call failed.

**Outcome:**

* **By rule 3:**
  * `Strict` is sent on the Responses wire (three targets) and on the four
    Chat Completions gateways.
  * `ollama` was not a T1 target, so its `function.strict` stays unsent.
    Measuring it would be an amendment.
* **By rule 4:** `IsError` is carried on Messages, generateContent and
  Interactions, in the forms above.

**Found, out of this plan's scope:** a tool request to Hugging Face's
default model, `deepseek-ai/DeepSeek-V4-Flash-0731`, answers 500 with or
without `strict`. Existing users of `huggingface` with tools on that model
are affected. It is not fixed here.

### Deviation D1 (2026-10-11): Hugging Face's default model fails every tool request

* **Found** in step 1. Hugging Face answered T1's `strict` request with a
  500, and so did T1c's control without the field. Its acceptance of
  `function.strict` could not be measured on that model, and rule 3 covers
  only a 2xx or a 4xx naming the field.
* **Options put to the owner:**
  * re-measure the pair on the next model in Hugging Face's static catalog,
    2 billed requests (recommended);
  * leave `strict` unsent on Hugging Face;
  * send it on the other gateways' evidence.
* **Decision.** The owner chose the re-measure. On `zai-org/GLM-5.3-Flash`
  both requests answered 200, so Hugging Face carries `function.strict`.
  The control test takes `LLMPROVIDER_LIVE_0032_HF_MODEL` to name the
  model. No decision of 0032-MADR changes.

### Phase 2: `Tool.Strict` (2026-10-11)

**Change.**

* `llmprovider/provider.go`: `Tool` gains `Strict *bool`. Its doc says nil
  leaves the service's default, and that a set value is sent where the wire
  has the field.
* `internal/wire/tools.go`: `ResponsesTools` adds `"strict"` when `Strict`
  is set.
* `internal/wire/chatcompletions`:
  * `Opts` gains `SendStrict`, and `chatToolFunction` gains `Strict *bool`
    (`strict,omitempty`, after `parameters`, so an unset one leaves the
    JSON byte for byte);
  * `toolList(tools, sendStrict)` copies a tool's `Strict` only when
    `sendStrict` is set.
* `huggingface`, `kilo`, `together` and OpenCode's `chat_completions` route
  set `SendStrict`. `ollama` does not: it was not measured.
* Every provider's package doc names, under Degradations, where `Strict` is
  sent and where it is not: Responses `strict`; Chat Completions
  `function.strict`; not on Messages, Interactions, Ollama, or OpenCode's
  `messages` and `google` routes.
* The existing `TestToolList_*` comparison (`typed_0028_test.go`) now calls
  `toolList(tools, false)`. Its tools set no `Strict`, so what it compares
  is unchanged.

**Red** (step 1), on the tree before the change: every new test failed to
build.

```text
llmprovider/internal/wire/tools_0032_test.go:19:74: unknown field Strict in struct literal of type llmprovider.Tool
llmprovider/internal/wire/chatcompletions/strict_0032_test.go:24:94: unknown field SendStrict in struct literal of type Opts
llmprovider/providers/{claude,gemini,grok,huggingface,kilo,…}/strict_0032_test.go:23:56: unknown field Strict in struct literal of type llmprovider.Tool
```

**Green, under `-race`:**

* `TestResponsesTools_Strict`: nil sends no key; `false` and `true` send
  theirs.
* `TestBody_Strict`: set and sent; set and not sent; nil and sent.
* `TestStrict_0032` in each provider:
  * `openai`, `grok`, `huggingface`, `kilo` and `together` send
    `"strict":false`;
  * `ollama`, `claude` and `gemini` send no `strict`;
  * `opencode` sends it on `responses` and `chat_completions`, and not on
    `messages` or `google`;
  * a nil `Strict` sends no `strict` anywhere.
* `go test -count=1 ./...` passes; `golangci-lint` reports `0 issues.`; the
  G-wire goldens are unchanged.

**Plants** (step 3), on a scratch clone. The last two prove the "not sent"
guards, which pass on the tree by design.

| Plant | Test | Result |
| :--- | :--- | :--- |
| `ResponsesTools` sends `strict` for a nil `Strict` | `TestResponsesTools_Strict` | `tools_0032_test.go:25: nil: tools [{… "strict":false …}]; want ""`; FAIL |
| `ResponsesTools` drops a set `Strict` | `openai` `TestStrict_0032` | `strict_0032_test.go:35: false: sent tools [{… no strict …}]; want "\"strict\":false"`; FAIL |
| `MessagesTools` sends `strict` | `claude` `TestStrict_0032` | `nil: sent tools [{… "strict":false}]; want ""` and `false: … "strict":true …`; FAIL |
| `ollama` sets `SendStrict` | `ollama` `TestStrict_0032` | `false: sent tools [{"function":{… "strict":false} …}]; want ""`; FAIL |

**API:** apidiff from `v1.6.0` (the tree, without `-incompatible`):

```text
Compatible changes:
- ./llmprovider.Tool.Strict: added
```

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

### Phase 3: `FunctionCallOutputItem.IsError` (2026-10-11)

**Change.**

* `llmprovider/item.go`: `FunctionCallOutputItem` gains `IsError bool`. Its
  doc names the wires that carry it, and says `Output` is never rewritten.
* `internal/wire/messages`: `Block` gains `IsError` (`is_error,omitempty`,
  in the order `json.Marshal` writes it), and a `tool_result` carries it.
* `internal/wire/generatecontent`: `FunctionOutput` is now `Error` or
  `Output`, both `*string` with `omitempty`, exactly one set. A failed
  result sends `{"error": …}`, and any other sends `{"output": …}` as
  before.
* `providers/gemini`: a failed `function_result` step carries
  `is_error: true`.
* Responses and Chat Completions are unchanged.
* `wire.RemapCallIDs` copies the whole item, so the flag survives a
  remapped call id.
* Every provider's package doc names, under Degradations, where `IsError`
  is sent and where `Output` is sent as it is.

**Red** (step 1), on the tree before the change:

```text
llmprovider/internal/wire/{messages,generatecontent,responses,chatcompletions}/iserror_0032_test.go:16:75: unknown field IsError in struct literal of type llmprovider.FunctionCallOutputItem
```

**Green, under `-race`:** `TestIsError_0032` passes in each place it runs.

* `messages`: a failed result carries `"is_error":true`; one not failed has
  no `is_error`.
* `generatecontent`: `"response":{"error":"offline"}`, and otherwise
  `"response":{"output":"offline"}`.
* `gemini`: a failed `function_result` carries `"is_error":true`.
* `responses` and `chatcompletions`: a failed result encodes exactly as one
  not failed.

`go test -count=1 ./...` passes; `golangci-lint` reports `0 issues.`; the
G-wire goldens are unchanged.

**Corrections to the new tests while they were made to build** (neither
loosens an assertion):

* The helpers were first named `failed` and `encoded`. `failed` clashes
  with the Responses package's own function, and `asJSON` with
  `messages_test.go`'s, so they became `turnWithResult`, and `asJSON`, or
  `resultJSON` in `messages`.
* An unused `strings` import was removed from the two "unchanged" tests.

**Plants** (step 3), on a scratch clone, each against `TestIsError_0032`:

| Plant | Package | Result |
| :--- | :--- | :--- |
| every result marked an error | `messages` | `iserror_0032_test.go:38: a result not failed: […"is_error":true…]`; FAIL |
| the flag dropped | `generatecontent` | `iserror_0032_test.go:35: a failed result: […"output"…]; want "response":{"error":"offline"}`; FAIL |
| the flag dropped | `gemini` | `iserror_0032_test.go:35: a failed result: […no is_error…]`; FAIL |
| a failed output rewritten (`"Error: "` prefix) | `responses` | `iserror_0032_test.go:33: a failed result […"output":"Error: offline"…] differs from one not failed`; FAIL |

**API:** apidiff from `v1.6.0` (the tree):

```text
Compatible changes:
- ./llmprovider.FunctionCallOutputItem.IsError: added
- ./llmprovider.Tool.Strict: added
```

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

### Deviation D2 (2026-10-11): the Fidelity check needs to know which flags a provider sends

* **Found** at step 2. Step 2 says the check "needs no new field", and
  0032-MADR D4 says it "adds no `Harness` field". Step 3 requires the check
  to fail for a planted provider that drops both flags.
* **Why both cannot hold:** dropping both flags is `ollama`'s documented
  behaviour, and each wire drops one of the two.
  `llmprovider/llmtest/llmtest.go` gives the check no way to tell them
  apart:
  * `Harness` does not name the wire;
  * `Capabilities` (`llmprovider/capabilities.go:33-44`) has no entry for
    either flag.

  The error is in this pair as written, not in existing code.
* **Options put to the owner:**
  * two optional `Harness` fields, `SendsStrict` and `SendsIsError`
    (recommended): two additions, up to three requests a check, and each
    provider's harness sets them;
  * `llmtest` infers the wire from the request's path, with no new field.
    It cannot tell `ollama` from the other Chat Completions gateways.

  A third route was not offered: keeping the field-less check and dropping
  step 3's plant would redraw scope.
* **Decision.** The owner chose the two fields. 0032-MADR records it as
  "Amendment (2026-10-11): two `Harness` fields". Step 2 is annotated.
* **Files added to the phase:**
  * each provider's `llmtest_test.go`, which sets the fields;
  * `llmprovider/llmtest/llmtest_test.go`, whose reference provider gains
    the flags and the flaws that break them;
  * `docs/guides/adding-a-provider.md` and `docs/architecture.md`'s
    `llmtest` bullet, which describe the fields. Phase 5's `architecture.md`
    step still covers the contract section.

### Phase 4: proof (2026-10-11)

**Step 1, wirecase.** `llmprovider/internal/wirecase` gains a `flags`
scenario. `FlagTools` offers `get_weather` with `Strict: true` and
`get_time` with `Strict: false`. `FlagItems` replays two calls, the first
result with `IsError` set and the second without. The literals it shares
with `Items` became the constants `firstCall`, `weatherTool` and
`parisArguments`, and the schema the function `citySchema`, for
`goconst`; the values are unchanged.

* **Red:** 18 cases, every case in the nine provider packages, failed with
  `wirecase.go:312: wiretest: no golden file testdata/wire/claude/flags.json; record one with -update`
  (and the same per case). No other subtest failed.
* **Green:** the 18 goldens were recorded once with `-update -run
  'TestWireGoldens/.*/flags$'`. `TestWireGoldens` passes at `-count=3`, and
  no existing golden changed.
* **What the goldens pin**, read from each one, as each package doc says:

| Wire | Cases | `Strict` | `IsError` |
| :--- | :--- | :--- | :--- |
| Responses | `openai`, `openai-stateless`, `chatgpt`, `grok`, `grok-stateless`, `opencode-{zen,go}-responses` | `tools[i].strict` `true`, then `false` | not sent |
| Chat Completions | `huggingface`, `kilo`, `together`, `opencode-{zen,go}-chat` | `tools[i].function.strict` `true`, then `false` | not sent |
| Chat Completions | `ollama` | not sent | not sent |
| Messages | `claude`, `opencode-{zen,go}-messages` | not sent | `is_error: true` on the failed `tool_result` only |
| Interactions | `gemini` | not sent | `is_error: true` on the failed `function_result` only |
| generateContent | `opencode-zen-google` | not sent | `response.error` for the failed result, `response.output` for the other |

Every `Output` is sent as written.

* **Plants** on a scratch clone, each against `TestWireGoldens`:

| Plant | Failing cases | First difference |
| :--- | :--- | :--- |
| Messages drops `is_error` | `claude`, `opencode-zen-messages`, `opencode-go-messages` | `requests[0].body.messages[2].content[0].is_error: want true, got (absent)` |
| generateContent drops the error form | `opencode-zen-google` | `…functionResponse.response.error: want "weather service offline", got (absent)` |
| Interactions drops `is_error` | `gemini` | `requests[0].body.input[5].is_error: want true, got (absent)` |
| Responses drops `strict` | the 7 Responses cases | `requests[0].body.tools[0].strict: want true, got (absent)` |
| Chat Completions drops `strict` | the 5 gateway chat cases | `requests[0].body.tools[0].function.strict: want true, got (absent)` |
| `ollama` sends `strict` | `ollama` | `requests[0].body.tools[0].function.strict: want (absent), got true` |

**Step 2, llmtest Fidelity**, in the form deviation D2 chose.

* `llmtest.Harness` gains `SendsStrict` and `SendsIsError`.
* `checkFidelity` builds its request with `fidelityRequest(strict,
  isError)` and records what each `Generate` sent. `checkFlags` then
  checks the flags:
  * with `Fidelity` set, a request whose result failed must carry
    `"llmtest-output-7f3a"`, quoted, so its `Output` reached the wire
    unchanged (0032-MADR D3);
  * with either field set, the plain request is sent again and must send
    the same body. A body that changes between identical requests is
    reported, rather than taken for a flag reaching the wire;
  * `SendsIsError`: the failed request's body must differ from the plain
    one;
  * `SendsStrict`: a request with `Strict: false` must differ from the
    plain one.
* **Requests per check:** two with neither field set, three with
  `SendsIsError` alone, four with `SendsStrict`. That is one more than the
  "up to three" put to the owner in deviation D2: the repeat of the plain
  request was added while building, so that a body that differs between
  identical requests cannot pass for a flag. Every request goes to the
  harness's local server, so none is billed.
* **The harnesses:**
  * `SendsStrict`: `openai`, `grok`, `huggingface`, `kilo`, `together`,
    and OpenCode's `chat_completions` and `responses` routes;
  * `SendsIsError`: `claude`, `gemini`, and OpenCode's `messages` and
    `google` routes;
  * neither: `ollama`.
* **The reference provider** in `llmtest_test.go` sends `strict` and
  `is_error` when set. It gains three flaws:
  * `dropFlags`;
  * `rewritesFailure`, which prefixes `"Error: "`;
  * `noisyBody`, which adds a counter.

  Its harness sets both fields. `TestRun_FlagChecksAreOptional` shows that a
  harness setting neither field passes with `dropFlags` and `noisyBody`.
* **Red,** in two steps:
  * the tests alone did not build:
    `llmtest_test.go:296:3: unknown field SendsStrict in struct literal of type Harness`;
  * with the fields and no check, exactly the four new cases failed:

    ```text
    --- FAIL: TestRun_NamesTheBrokenRule/drops_a_tool's_Strict
        llmtest_test.go:395: failures []; want one starting "W12-fidelity: W12 (fidelity, 0032-MADR D4): the harness sets SendsStrict"
    --- FAIL: TestRun_NamesTheBrokenRule/drops_a_result's_IsError
        llmtest_test.go:395: failures []; want one starting "W12-fidelity: W12 (fidelity, 0032-MADR D4): the harness sets SendsIsError"
    --- FAIL: TestRun_NamesTheBrokenRule/rewrites_a_failed_result
        llmtest_test.go:395: failures []; want one starting "W12-fidelity: W12 (fidelity, 0032-MADR D3): a failed result's Output"
    --- FAIL: TestRun_NamesTheBrokenRule/sends_a_different_body_each_time
        llmtest_test.go:395: failures []; want one starting "W12-fidelity: W12 (fidelity, 0032-MADR D4): the same request sent twice"
    ```

* **Green, under `-race`:**
  * `llmtest` passes;
  * every provider's `TestConformance` passes, first with no field set (the
    unchanged-`Output` check alone), then with the fields set;
  * `go test -race -count=1 ./...` passes.

**Step 3, plants** on a scratch clone, each against the providers'
`TestConformance`. Every package named failed, and none passed:

| Plant | Failing harnesses | Message |
| :--- | :--- | :--- |
| Messages drops `is_error` | `claude`; OpenCode `go-messages` | `W12 (fidelity, 0032-MADR D4): the harness sets SendsIsError, but a failed result reached the wire as one not failed: /messages {…` |
| generateContent drops the error form | OpenCode `zen-google` | the same, on `…:generateContent` |
| Interactions drops `is_error` | `gemini` `stored` and `unstored` | the same, on `/interactions` |
| Responses drops `strict` | `openai` (`api-key`, `session`, `chatgpt`), `grok`, OpenCode `go-responses` | `W12 (fidelity, 0032-MADR D4): the harness sets SendsStrict, but a tool with Strict false reached the wire as one with Strict unset: /responses {…` |
| Chat Completions drops `strict` | `huggingface`, `kilo`, `together`, OpenCode `zen-chat` | the same, on `/chat/completions` |
| Chat Completions rewrites a failed output | `ollama` (no field set), `kilo` | `W12 (fidelity, 0032-MADR D3): a failed result's Output, "llmtest-output-7f3a", did not reach the wire unchanged: /v1/chat/completions {…` |

**Documentation**, added to the phase by deviation D2:

* `docs/guides/adding-a-provider.md` tells a new provider when to set the
  fields;
* `docs/architecture.md`'s `llmtest` bullet names them.

**Fixes in the phase**, none of which loosens an assertion:

* `goconst` flagged `call_1`, then `get_weather`. Each became a constant,
  as above.
* My first constant script asserted the wrong count and stopped before
  writing. Its second version would also have rewritten the new constant's
  own literal. It was rewritten to replace the uses first; the file was
  never left wrong.

**Scratch tooling:** the gate's `g_links.py` and `denyscan.py` had been
pruned from the scratchpad again. Each was restored byte-for-byte from
this session's transcript, the version used through Phase 3. Each was
shown failing on a planted input before the gate ran:

* `a.md:1: missing.md`, rc 1;
* `a machine home path`, rc 1.

**API:** apidiff from `v1.6.0` (the tree):

```text
Compatible changes:
- ./llmprovider.FunctionCallOutputItem.IsError: added
- ./llmprovider.Tool.Strict: added
- ./llmprovider/llmtest.Harness.SendsIsError: added
- ./llmprovider/llmtest.Harness.SendsStrict: added
```

**Gate:** all 22 checks pass on a scratch copy with the phase's files
staged:

* `g-links`: 0 problems in 700 relative links;
* `deny-scan`: 0 hits in 33 files;
* records check: 69 records, 0 problems.

### Phase 5: documentation and release (2026-10-11)

**Steps 1 and 2**, on `9446adc` (Phase 4):

* **`docs/architecture.md`:** the contract section gains a bullet naming
  both flags and the wires that send each. Phase 4 had already added the
  `llmtest` bullet's fields. "What is not here" is unchanged: it names
  nothing these flags address.
* **`README.md`:** Status names `v1.7.0`, with a `v1.7` bullet.
* **Package docs:** changed in Phases 2 and 3. **`api-standards.md`:**
  unchanged; no rule's text changes.

**Step 3, the live suites,** on the release commit `8b3841d`.

* **Run:** 13:01 to 13:05 UTC. Every provider's key variable was set
  (checked for presence only), with `LLMPROVIDER_LIVE_TOGETHER=1` and a
  local Ollama server answering. The suites that need a person, a
  subscription or their own switch stayed off, `LLMPROVIDER_LIVE_0032`
  among them.
* **Command:** `go test -tags live_gateways -count=1 -v ./llmprovider/...
  -run Live`.
* **Result:** exit 0.
  * Top level: 76 passed, 21 skipped, 0 failed.
  * Subtests: 79 passed, 5 skipped, 0 failed.

  The requests were not counted: the suites log tests, not requests. No
  test failed, so nothing was rerun.

The streaming tests, as in `v1.6.0`:

| Test | Model | Result |
| :--- | :--- | :--- |
| `TestLive_OpenAIStreams` | `gpt-4.1-mini` | 9 text deltas, then `EventDone` |
| `TestLive_GrokStreams` | `grok-4.6` | 9 text deltas, then `EventDone` |
| `TestLive_OpencodeStreams/opencode-go` (responses) | `gpt-6-luna` | 2 text deltas, then `EventDone` |
| `TestLive_ClaudeStreams` | `claude-haiku-4-5` | 5 text deltas, then `EventDone` |
| `TestLive_OpencodeMessagesStreams/opencode-go` | `qwen3.8-flash` | 4 text deltas, then `EventDone` |
| `TestLive_HuggingFaceStreams` | `deepseek-ai/DeepSeek-V4-Flash-0731` | 3 text deltas, then `EventDone` |
| `TestLive_KiloStreams` | `deepseek/deepseek-v4.1-flash` | 2 text deltas, then `EventDone` |
| `TestLive_TogetherStreams` | `deepseek-ai/DeepSeek-V4.1-Flash` | 5 text deltas, then `EventDone` |
| `TestLive_OpencodeChatStreams/opencode-go` | `glm-5.3-flash` | 17 text deltas, then `EventDone` |
| `TestLive_OllamaStreams` | `llama3.2:latest` | 10 text deltas, then `EventDone` |
| `TestLive_OpencodeGoogleStreams/google-endpoint` | `gemini-3.7-flash` | 1 text delta, then `EventDone` |
| `TestLive_GeminiStreams` (stored) | `gemini-3.7-flash` | 1 text delta, then `EventDone` with an interaction id |

**The 21 top-level skips:**

* **Off by their own switch, 20:**
  * the 0028, 0031 and 0032 measurements (`LLMPROVIDER_LIVE_0028`,
    `LLMPROVIDER_LIVE_0031`, `LLMPROVIDER_LIVE_0031B`,
    `LLMPROVIDER_LIVE_0032`), 9;
  * the ChatGPT and Codex CLI subscription, 4;
  * Gemini's rate-limit drain, 1;
  * the Grok CLI, 1;
  * the browser and device sign-ins and the OpenAI sign-in probe, 5.
* **No active model:** `TestLive_OpencodeInterleavedReasoningReplay`, "no
  active opencode-go model among [kimi-k2.6]", as in `v1.6.0`.

Against `v1.6.0`'s 19, this adds the three 0032 measurement tests. It
loses `TestLive_KiloReasoningSpelling`, which passed this time.

**The 5 subtest skips** are all OpenCode Zen, each under 0027-MADR's
transient rule. They are the same five, for the same account reasons, as
in `v1.6.0`:

* `TestLive_OpencodeGoogleTwoCallRoundTrip/zen`: 402 "Insufficient account
  funds";
* the four `opencode-zen` stream subtests:
  * 403 "Model access is disabled" on `responses`;
  * 402 "Insufficient account funds" on `messages`, `chat_completions` and
    `google`.

Every check that ran against a service passed on the release commit.

#### Release notes: `v1.7.0`

`v1.7.0` adds a per-tool strict flag and a tool result's error flag
(0032-MADR). Every change to the exported API is an addition.

**`Tool.Strict *bool`.** Nil sends nothing, as before. Set, it opts the
tool in or out of the service's strict schema mode:

| Wire | Providers | Sent as |
| :--- | :--- | :--- |
| Responses | `openai`, `grok`, OpenCode's `responses` route | `strict` |
| Chat Completions | `huggingface`, `kilo`, `together`, OpenCode's `chat_completions` route | `function.strict` |
| Chat Completions | `ollama` | not sent |
| Messages, Interactions, generateContent | `claude`, `gemini`, OpenCode's `messages` and `google` routes | not sent: no such field |

**`FunctionCallOutputItem.IsError bool`.** It marks a failed call's result:

| Wire | Providers | Sent as |
| :--- | :--- | :--- |
| Messages | `claude`, OpenCode's `messages` route | `tool_result`'s `is_error` |
| Interactions | `gemini` | the `function_result` step's `is_error` |
| generateContent | OpenCode's `google` route | `response: {"error": …}` in place of `{"output": …}` |
| Responses, Chat Completions | the rest | not sent: no such field |

* `Output` is never rewritten. A caller who wants the model to read why
  the call failed writes it in `Output`.
* Each provider's package doc names, under Degradations, where each flag is
  not sent.
* Each form was measured live before it was built (Phase 1 above):
  * `strict` on `openai`, `grok` and OpenCode Go's `responses` route;
  * `function.strict` on `huggingface` (on `zai-org/GLM-5.3-Flash`; its
    default model fails every tool request), `kilo`, `together` and
    OpenCode Go's `chat_completions` route;
  * `is_error` on `claude` and `gemini`;
  * the `error` response on Google's endpoint, with the `google` route's
    request.

  OpenCode Zen's routes, and OpenCode's `messages` route, stand on those
  measurements of the same wire. The Chat Completions gateways do not echo
  their tools, so whether one applies `function.strict` is unmeasured.

**`llmtest`:**

* `Harness` gains `SendsStrict` and `SendsIsError`. With `Fidelity` set,
  each that is set requires its flag to change the request, and the same
  request sent twice must send the same body.
* With `Fidelity` set, a failed result's `Output` must reach the wire
  unchanged, whether or not either field is set.
* A harness that sets neither field otherwise passes as before.

**Unchanged:** a caller that sets neither field sends byte-for-byte what it
sent in `v1.6.0`. The existing wire goldens are unchanged.

**Upgrading:**

* No code change is needed.
* To opt a tool out of OpenAI's strict mode, write
  `strict := false; tool.Strict = &strict`.
* To report a failed call, set `IsError: true` on its result.
* A third-party provider whose wire carries either flag can set the
  matching `Harness` field.
