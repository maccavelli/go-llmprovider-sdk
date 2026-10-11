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
   and recorded.
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
   * apidiff `v1.6.0` to `v1.7.0`: the two fields, as additions;
   * a scratch consumer, and a scratch copy of gobble-cli on `v1.7.0`.
6. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with both lines in the Execution Record, and
  every plant's FAIL line.
* **V2. The gate** clean at each phase; the existing goldens unchanged.
* **V3. Measurement:** T1 and T2 recorded before Phases 2 and 3 use them.
* **V4. API:** apidiff from `v1.6.0` shows the two fields, as additions
  only.
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
