---
status: in-progress
date: 2026-10-13
associated-madr: "0039-MADR-request-side-prompt-caching.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Request-Side Prompt Caching

Associated MADR: [0039-MADR-request-side-prompt-caching.md](0039-MADR-request-side-prompt-caching.md)

## Goal

When it is done:

* **The field.** `llmprovider.Request` has a `Cache` field of type
  `CacheRetention`, `CacheShort` or `CacheLong`. Its zero value asks for
  nothing, and an unknown value is refused before the network (D1).
* **Claude.** A cached request on the Messages wire, for Claude and for
  OpenCode's Claude routes, carries the top-level `cache_control` and one
  breakpoint, on the system prompt or else the last tool. Both have one TTL:
  `5m` for short, `1h` for long (D2).
* **Routing.** OpenAI with an API key, xAI, Together and OpenCode's
  Responses route send `prompt_cache_key`, the provider's session id, when
  `Cache` is set (D3, D4).
* **Kilo.** Kilo sends OpenRouter's top-level `cache_control` for a model
  whose id starts `anthropic/` (D4).
* **No change otherwise.** Hugging Face, Ollama and Gemini send nothing for
  `Cache`, and every package doc says what its provider sends (D4, D5).
* **The wire.** A request without `Cache` is the same on the wire, byte for
  byte, and every existing G-wire golden is unchanged.
* **A billed live run** shows the second of two cached requests reading
  what the first wrote, on each measured path.
* **`v1.12.0` is released.** `tools/apidiff.py` reports only the additions:
  the type, its two constants and the field.

## Scope

### In scope

* **The contract:**
  * `llmprovider/contract.go`: the type, its constants, the field, and
    `validate`;
  * `llmprovider/llmtest/llmtest.go`: an R23 case for an unknown retention.
* **The wire:**
  * `llmprovider/internal/wire/messages`: one helper that adds D2's
    markers;
  * `llmprovider/internal/wire`: one helper that adds `prompt_cache_key`.
* **The providers:**
  * `claude`;
  * `opencode`: its Messages and Responses bodies;
  * `openai`: the API-key body;
  * `grok` and `together`: each keeps its session id, and its body adds the
    key;
  * `kilo`: its body adds the top-level field for an `anthropic/` model;
  * the package docs of all nine, as R12 asks.
* **The tests:**
  * `llmprovider/internal/wirecase`: two scenarios, `cache-short` and
    `cache-long`, which add new goldens for every case;
  * a unit test per helper;
  * a live measurement, `llmprovider/live_0039_prompt_cache_measure_test.go`.
* **The docs:**
  * `docs/guides/api-standards.md` R6;
  * `docs/architecture.md`, at the `Request` fields and a paragraph on
    prompt caching;
  * `README.md`, its Status and a feature bullet;
  * `docs/README.md`.

### Out of scope

As 0039-MADR D6 decides:

* a `CacheNone` value;
* breakpoints placed by the caller;
* cache warming;
* Gemini's `cachedContents`;
* OpenAI's `prompt_cache_breakpoint`, `prompt_cache_options` and `prewarm`;
* a per-request session id;
* OpenCode's Chat Completions and generateContent routes, which send
  nothing for `Cache`.

## Rules for every phase

* **Research first.** Each phase starts by reading the code it changes, at
  the commit it starts from. A fact that differs from this PLAN or the MADR
  is a deviation.
* **Red first,** with every FAIL line recorded.
* **Plants** go through `tools/plant.py`, on a scratch clone, never in the
  tree. Each plant's FAIL line is recorded.
* **Every script is Python.** A reusable one belongs in `tools/`. Specs and
  scripts are written with the Write tool, and exact edits go through
  `tools/edit.py`.
* **The goldens:**
  * an existing golden never changes;
  * a new golden is generated alone, with `-update -run` and its scenario,
    and its content is recorded here.
* **A phase is staged only after:**
  * `make pre-add-check FILES="…"`;
  * `make records-check`;
  * `tools/denyscan.py --changed` and `tools/linkcheck.py`;
  * markdownlint on any changed non-record Markdown;
  * the gate, `tools/gate.py`, at 23 of 23.
* **Billed requests:** the count goes to the owner, and the owner approves,
  before any live run. Keys are checked for presence only.
* **A deviation stops the work:** evidence, options with the recommended
  one first, and a dated entry here, plus a MADR amendment where a fact or
  a decision changes.
* **The owner commits** each phase, pushes and tags. If a phase is still
  staged, the next one stays unstaged.

## Implementation Steps

### Phase 0: records

1. 0039-MADR is `accepted` (the owner, 2026-10-13). This PLAN is
   `in-progress`, approved by the owner on 2026-10-13.
2. `docs/README.md`: the PLAN's row, the status and the count.
3. `make records-check`, `tools/linkcheck.py` and `tools/denyscan.py
   --changed`. Stage. The owner commits.

### Phase 1: the field

1. **Research.**
   * Read `contract.go`'s `Request` and `validate`, and how `ToolChoice`, a
     string type with constants, is validated.
   * Read `llmtest`'s `checkInvalidValues` (`llmtest.go:380-410`).
   * Read R6 and R23 in `docs/guides/api-standards.md`.
2. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestRequest_CacheValues` (`llmprovider/cache_0039_test.go`) | `""`, `CacheShort` and `CacheLong` pass `Capabilities.Check`. `"forever"` fails with `ErrInvalidRequest`, naming the value |
   | `llmtest` R23, a new case, "an unknown cache retention" | every provider refuses `Cache: "forever"` with `ErrInvalidRequest`, and sends no request |

   Each fails before the change: the first does not compile (the field is
   added with no validation), and `llmtest` reports a sent request.
3. **Change.**
   * `type CacheRetention string`, `CacheShort = "short"` and
     `CacheLong = "long"`, with docs citing 0039-MADR D1.
   * `Request.Cache`, with its doc: "the zero value asks for nothing".
   * `validate` refuses any other value.
   * R6 in `api-standards.md` lists `Cache`, citing 0039.
4. **Plants:**
   * the validation removed;
   * `"long"` refused;
   * `llmtest`'s new case dropped from its table, shown by a plant on one
     provider's validation path.
5. **Green:** `go test ./...`. `make api-check` shows the additions. The
   gate. Stage. The owner commits.

No provider sends anything for `Cache` yet. Every golden is unchanged.

### Phase 2: the Messages wire (Claude, OpenCode's Claude routes)

1. **Research.**
   * Read `claude.go:273-298` and `opencode.go:507-525`, the two Messages
     bodies.
   * Read `wire.MessagesTools`, `wire.SystemPrompt` and the wirecase
     scenario table (`wirecase.go:244-293`).
   * Confirm both bodies end with the system and tools set, so that one
     helper can run last.
2. **The helper,** `messages.AddCache(body map[string]any,
   retention llmprovider.CacheRetention)`. With a non-zero retention:
   * `ttl` is `"5m"` for short and `"1h"` for long, and the marker is
     `{"type":"ephemeral","ttl":ttl}`;
   * `body["cache_control"]` is the marker;
   * a string `body["system"]` becomes
     `[{"type":"text","text":…,"cache_control":marker}]`;
   * else the last of `body["tools"]` gets `"cache_control": marker`;
   * else nothing more.

   The zero value returns at once, so the body is untouched.
3. **The callers:** `claude`'s `body` and `opencode`'s `messagesBody` call
   it last, with `req.Cache`.
4. **The scenarios,** added to `wirecase`'s table:
   * `cache-short` and `cache-long` send `Instructions: "Be brief."`, the
     weather tool and `user(Prompt)`, with `Cache` set.
   * Every case gets the two new goldens. Messages cases show the markers.
     Every other case shows what it sends today.
   * Phase 3 regenerates the Responses, Together and Kilo goldens, each
     diff recorded.
5. **Red first:**

   | Test (file) | Want |
   | :--- | :--- |
   | `TestAddCache` (`internal/wire/messages/cache_0039_test.go`) | for system and tools, tools alone, and neither, with each retention: the top-level marker, one breakpoint where D2 puts it, one TTL throughout, at most two markers; the zero value leaves the body equal to its copy |
   | `TestAddCache_SystemFormOnlyWhenAsked` | a body without `Cache` keeps `system` a string |
   | the new goldens, for `claude` and both OpenCode Messages cases | recorded from the change; their content in this PLAN |

6. **Plants:**
   * the system kept a string when caching;
   * the TTL `"5m"` on the breakpoint under `CacheLong`;
   * a marker on every tool;
   * the zero value adding the top-level field. The existing goldens catch
     this one too.
7. **Green:**
   * `go test ./...`;
   * `g-wire-stable` in the gate: no existing golden changed;
   * the `claude` and `opencode` package docs say what `Cache` sends.

   The gate. Stage. The owner commits.

### Phase 3: routing keys and Kilo

1. **Research.**
   * Read `openai.go`'s API-key body, `opencode.go:482-505`
     (`responsesBody`), and `grok.go:198` and `together.go:185` (the
     bodies).
   * Read how each provider resolves `st.SessionID()`. Grok and Together
     keep no session field today.
   * Read `kilo.go:278-310` and `chatcompletions.Body`.
2. **The helper,** `wire.AddPromptCacheKey(body, retention, session)`:
   `prompt_cache_key` is the session, when the retention is not zero and the
   session is not empty.
3. **The callers:**
   * `openai`'s API-key body, and `opencode`'s `responsesBody`;
   * `grok`'s body, with a new `session` field from `st.SessionID()`;
   * `together`'s body, the same way.
   * The ChatGPT body is unchanged: it already sends the key.
4. **Kilo.** When `req.Cache` is set and the model starts `anthropic/`, the
   body gets `cache_control: {"type":"ephemeral","ttl":…}`, through
   `messages`' marker function so the TTL rule is one function.
5. **Red first:**

   | Test (file) | Want |
   | :--- | :--- |
   | `TestAddPromptCacheKey` (`internal/wire/cache_0039_test.go`) | the key for each retention, none for the zero value or an empty session |
   | `TestBody_PromptCacheKey` in `openai`, `grok`, `together` and `opencode` (Responses route) | the key equals the session with `Cache` set, and is absent without it |
   | `TestKilo_CacheControlForClaude` (`providers/kilo/cache_0039_test.go`) | `anthropic/claude-haiku-5.5` gets the top-level marker with the TTL; `openai/gpt-6-luna` gets none; no `Cache`, none |
   | the regenerated `cache-short` and `cache-long` goldens of `openai`, `grok`, `together`, OpenCode's Responses cases and `kilo` | diffs recorded: one `prompt_cache_key` line, or Kilo's unchanged when its case's model is not `anthropic/` |

6. **Plants:**
   * the key sent without `Cache`;
   * Grok's session not stored, so the key is empty;
   * Kilo marking every model;
   * the ChatGPT body sending the key twice.
7. **Docs:**
   * the package docs of `openai`, `grok`, `together` and `kilo`;
   * `gemini`, `huggingface` and `ollama` say that `Cache` sends nothing,
     and why;
   * `docs/architecture.md`, at the `Request` fields and a "Prompt caching"
     paragraph listing each provider's form;
   * `README.md`'s feature bullet.
8. **Green:** `go test ./...`, with `g-wire-stable`: only the new goldens
   changed. The gate. Stage. The owner commits.

### Phase 4: live confirmation and release

1. **The measurement,** `live_0039_prompt_cache_measure_test.go`
   (`live_gateways`). It runs only when `LLMPROVIDER_LIVE_0039=1` is set,
   since every request is billed.
   * **The prompt:** each path sends the same request twice, back to back:
     a fixed system prompt of about 3,000 tokens (above every minimum the
     MADR lists), a short question and `MaxOutputTokens` 16. Probes are off.
   * **The paths, and the counts they log:**

     | Path | Model | Retention | Requests |
     | :--- | :--- | :--- | ---: |
     | Claude, directly | `claude-haiku-5-5` | short, then long | 4 |
     | OpenCode Zen | `claude-haiku-5-5` | short | 2 |
     | Kilo | `anthropic/claude-haiku-5.5` | short, then long | 4 |
     | OpenAI, API key | `gpt-6-luna` | short | 2 |
     | xAI | `grok-4.6` | short | 2 |
     | Together | the first of `staticTogether` | short | 2 |

     That is 16 billed requests, which go to the owner for approval before
     the run.
   * **What each path logs:** the first and second requests'
     `CachedTokens`, `CacheWriteTokens` and `CacheWrite1hTokens`.
   * **What the test requires:**
     * a non-zero `CachedTokens` on the second request of every Claude
       path, and of OpenAI's;
     * a non-zero `CacheWrite1hTokens` on Claude's first long request;
     * no 400 on any path.
   * **What it only logs:** xAI and Together's reads are logged, not
     required. Their caching is best-effort routing, and Together reports
     none.
   * **Kilo's hour:** Kilo sends no lifetime split (0036), so its first
     long request is logged, not held to a non-zero hour.
2. **A refusal is a deviation:** Kilo or Zen refusing the field, or a key
   refused.
3. **`README.md` Status:** `v1.12.0`, and the release notes in this PLAN.
4. **The checks before the owner pushes:**
   * `make lint vuln parity-check dep-check coverage-check api-check
     generate-check records-check gate-selftest`;
   * the gate.

   The disclosure guard runs when the owner asks.
5. **The owner** commits, pushes, and tags `v1.12.0`.
6. **After the tag:**
   * CI on `main` and on the tag;
   * the proxy reports `v1.12.0`;
   * `tools/apidiff.py v1.11.0 v1.12.0` reports only the additions;
   * `tools/consumer.py v1.12.0` and `tools/offline.py v1.12.0` pass.
7. **This PLAN** becomes `complete`, and `docs/README.md` says so.

## Verification

| | Acceptance criterion | Shown by |
| :--- | :--- | :--- |
| V1 | an unknown retention is refused before the network, by every provider | `TestRequest_CacheValues`; `llmtest` R23 |
| V2 | no request without `Cache` changes on the wire | `g-wire-stable`: every existing golden unchanged |
| V3 | a cached Messages request carries two markers at most, one TTL, at D2's places | `TestAddCache`; the `claude` and OpenCode goldens |
| V4 | the routing key and Kilo's field are sent only as D3 and D4 say | Phase 3's tests and goldens |
| V5 | live, a second request reads what the first wrote, and Claude's long write is an hour's | Phase 4's measurement |
| V6 | API: only the type, its two constants and the field are added | `tools/apidiff.py v1.11.0 v1.12.0` |
| V7 | every new assertion has failed on a plant | each phase's plant table |

## Rollout and Rollback

* **Rollout:**
  * `v1.12.0`, a minor release with additions only. No caller's requests
    change until it sets `Cache`.
  * gobble-cli adopts it in its own repository, setting `Cache` on its
    conversation requests. Its prompt-cache warming is then unblocked.
* **Rollback:**
  * A caller stops setting `Cache`.
  * A provider whose service refuses a marker can stop sending it in a
    patch release, and the field stays.
  * No exported identifier is removed, so R48 holds.

## Execution Record

### Phase 0: records (2026-10-13)

* **The commit:** the owner committed the MADR, this PLAN and the index as
  `2883e90`.
* **The approval:** the owner approved the PLAN ("committed, accepted,
  approved to proceed"), so it is `in-progress`.

### Phase 1: the field (2026-10-13)

**Research** confirmed the PLAN's facts:

* `Effort` is the pattern for a string type with constants: a `valid()`
  method, checked in `validate`. `ToolChoice` and `Reasoning` are checked
  at the end of `validate` (`contract.go`).
* `llmtest`'s `checkInvalidValues` builds each case from `textRequest()`,
  and requires `ErrInvalidRequest` and no request sent.

**Step A:** the type, its two constants and the field were added with no
validation, so the tests compiled.

**Red first:**

```text
--- FAIL: TestRequest_CacheValues
    cache_0039_test.go:23: Cache "forever": <nil>; want ErrInvalidRequest naming the value
--- FAIL: TestConformance/R23-invalid-values (claude; the same for every provider)
    llmtest.go:405: R23 (invalid values): an unknown cache retention returned <nil>; want an error matching ErrInvalidRequest
    llmtest.go:408: R23 (invalid values): an unknown cache retention sent 1 request(s); want none
```

**Change:**

* **`llmprovider/contract.go`:**
  * `type CacheRetention string`, with `CacheShort` (`"short"`) and
    `CacheLong` (`"long"`), and `valid()`;
  * `Request.Cache`;
  * `validate` refuses an unknown value: `unknown cache retention %q`,
    matching `ErrInvalidRequest`.
* **`llmprovider/llmtest/llmtest.go`:** R23's table gains "an unknown cache
  retention", so every provider's conformance run checks it.
* **`docs/guides/api-standards.md`:** R6 lists `Cache`, citing 0039 D1.

No provider sends anything for `Cache` yet.

**Green:** `go test ./...`, all 30 packages `ok`.

**Plants** (`tools/plant.py`, 2 of 2 caught):

| Plant | FAIL line |
| :--- | :--- |
| the validation removed | `TestRequest_CacheValues`: `Cache "forever": <nil>; want ErrInvalidRequest naming the value`; `TestConformance/R23-invalid-values` for `claude` and `ollama`: `an unknown cache retention returned <nil>` and `sent 1 request(s); want none` |
| `CacheLong` refused | `TestRequest_CacheValues`: `Cache "long": … unknown cache retention "long"; want it accepted` |

**Done differently from step 4.** The third plant, "`llmtest`'s new case
dropped", was not run. The first plant already shows the new `llmtest` case
failing, through two providers' conformance runs, when validation is
missing. That was the third plant's purpose.

**Checks:**

* `make pre-add-check` on the three Go files: clean.
* `tools/apidiff.py v1.11.0`, compatible changes only:
  `./llmprovider.CacheLong`, `./llmprovider.CacheRetention`,
  `./llmprovider.CacheShort` and `./llmprovider.Request.Cache`, each
  `added`.

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 1 as `c2f2f60`.

### Phase 2: the Messages wire (2026-10-13)

**Research** confirmed the PLAN's facts:

* Both Messages bodies set `system` as a string and `tools` from
  `wire.MessagesTools`, and return. So one helper can run last:
  `claude.go:273-298`, and `opencode.go:507-525`, where `Instructions`
  reach `system` as a leading system item (`:361-362`).
* The scenario table is a plain list (`wirecase.go:255-293`). A new entry
  adds one golden per case and leaves the others alone.

**Step A:** `messages.CacheMarker` and `messages.AddCache` were added as
no-ops, so the tests compiled.

**Red first:**

```text
--- FAIL: TestAddCache
    cache_0039_test.go:87: short, system and tools: markers at []; want them at ["" ".system[0]"]
    cache_0039_test.go:87: short, tools alone: markers at []; want them at ["" ".tools[1]"]
    cache_0039_test.go:87: short, neither: markers at []; want them at [""]
    (the same three for long)
--- FAIL: TestAddCache_SystemFormOnlyWhenAsked
    cache_0039_test.go:110: a cached body's system is still a string: Be brief.
```

**Change:**

* **`llmprovider/internal/wire/messages/cache.go`:**
  * `CacheMarker(retention)` is `{"type":"ephemeral","ttl":…}`: `"5m"` for
    `CacheShort`, `"1h"` for `CacheLong`.
  * `AddCache(body, retention)` returns at once for the zero value.
    Otherwise it sets the top-level `cache_control`, and either turns
    `system` into one text block carrying the marker, or marks the last
    tool.
* **The callers:** `claude`'s `body` and `opencode`'s `messagesBody` call it
  last.
* **`wirecase`:** the `cache-short` and `cache-long` scenarios send
  `cacheRequest(retention)`: `Instructions: "Be brief."`, the weather tool
  and the question.
* **The package docs** of `claude` and `opencode` say what `Cache` sends.

**The goldens.** Before any golden was written, `TestWireGoldens` failed
only on the 38 new `cache-*` files, and no existing scenario failed. The new
goldens were then written alone, package by package:

```text
go test ./llmprovider/providers/<p>/ -run 'TestWireGoldens/.*/cache-(short|long)$' -update
```

No existing golden changed. Their cache fields:

| Cases | `cache_control` | `prompt_cache_key` |
| :--- | ---: | ---: |
| `claude`, `claude-refuses-forced`, `opencode-zen-messages`, `opencode-go-messages` | 2: top level and `system[0]`, `5m` or `1h` | 0 |
| `chatgpt` | 0 | 1, as before 0039 |
| every other case | 0 | 0, until Phase 3 |

`claude/cache-long.json`'s body carries `"cache_control":{"ttl":"1h","type":"ephemeral"}`
and `"system":[{"cache_control":{"ttl":"1h","type":"ephemeral"},"text":"Be brief.","type":"text"}]`.
Its tool carries no marker.

**Green:** `go test ./...`, all 30 packages `ok`.

**Plants** (`tools/plant.py`, 4 of 4 caught):

| Plant | FAIL line |
| :--- | :--- |
| the system kept a string when caching | `TestAddCache`: `short, system and tools: markers at []`; `TestWireGoldens/claude/cache-short`: `requests[0].body.system: want [{"cache_control":…}], got "Be brief."` |
| the breakpoint's TTL `5m` under `CacheLong` | `TestAddCache`: `the marker at ".system[0]" is map[ttl:5m type:ephemeral]; want map[ttl:1h …]`; `TestWireGoldens/claude/cache-long`: `system[0].cache_control.ttl: want "1h", got "5m"` |
| a marker on every tool | `TestAddCache`: `tools alone: markers at [ .tools[0] .tools[1]]; want them at ["" ".tools[1]"]` |
| the zero value adding the markers | `TestAddCache_SystemFormOnlyWhenAsked`: `AddCache with no retention changed the body`; the existing goldens: `TestWireGoldens/claude/text`: `requests[0].body.cache_control: want (absent), got {"ttl":"5m",…}` |

The last plant shows V2 holding: an existing golden fails as soon as a
request without `Cache` changes.

**Checks:** `make pre-add-check` on the five Go files: clean.

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 2 as `5aa7ef5`.

### Phase 3: routing keys and Kilo (2026-10-13)

**Research** confirmed the PLAN's facts:

* **The bodies that take the key:**
  * `openai`'s body adds `prompt_cache_key` on the ChatGPT branch only;
  * `opencode`'s `responsesBody`, `grok`'s body (Responses) and
    `together`'s body (Chat Completions, through `chatcompletions.Body`)
    send none.
* **Sessions.** Grok and Together resolve `st.SessionID()` only for their
  listing options. `openai` and `opencode` keep `p.session`.
* **Kilo's wire case** uses `anthropic/claude-sonnet-5`
  (`providers/kilo/wire_test.go:25`). So its `cache-*` goldens show the
  top-level field. The PLAN allowed for either.

**Step A:** `wire.AddPromptCacheKey` was added as a no-op, so the tests
compiled.

**Red first:**

```text
--- FAIL: TestAddPromptCacheKey
    cache_0039_test.go:25: retention "short", session "s-1": prompt_cache_key = <nil>; want s-1
    cache_0039_test.go:25: retention "long", session "s-1": prompt_cache_key = <nil>; want s-1
--- FAIL: TestCache_PerProvider
    --- FAIL: …/openai/gpt-6-luna: Cache true: prompt_cache_key = <nil> (sent false); want sent true, as "session-0039"
    --- FAIL: …/grok/grok-4.6: the same
    --- FAIL: …/together/openai/gpt-oss-120b: the same
    --- FAIL: …/opencode-zen/gpt-5.5: the same
    --- FAIL: …/kilo/anthropic/claude-haiku-5.5: Cache true: top-level cache_control sent false; want true
```

Seven rows passed before the change, as they must:

* the Messages routes, already done in Phase 2;
* OpenCode's chat route, Kilo's `openai/` model, Gemini, Hugging Face and
  Ollama, which send nothing.

**Change:**

* **`llmprovider/internal/wire/cache.go`:** `AddPromptCacheKey` sets
  `prompt_cache_key` to the session when the retention and the session are
  both non-empty.
* **The providers:**
  * `openai`'s API-key branch and `opencode`'s `responsesBody` call it;
  * `grok` and `together` gain `session` (from `st.SessionID()`) and call it
    last;
  * `kilo` adds `cache_control: messages.CacheMarker(req.Cache)` when
    `Cache` is set and the model starts `anthropic/`.
* **Docs:**
  * the package docs of `openai`, `grok`, `together` and `kilo` say what
    `Cache` sends;
  * those of `gemini`, `huggingface` and `ollama` say it sends nothing, and
    why;
  * `docs/architecture.md`'s contract section lists each wire's form;
  * `README.md` has a "Prompt caching" bullet.

**The goldens.** With the change, `TestWireGoldens` failed on exactly the 16
expected `cache-*` goldens, and on nothing else. The only differences:

```text
14 × requests[0].body.prompt_cache_key: want (absent), got "wire-session"
     (openai, openai-stateless, grok, grok-stateless, together,
      opencode-zen-responses, opencode-go-responses; short and long)
 1 × requests[0].body.cache_control: want (absent), got {"ttl":"5m","type":"ephemeral"}   (kilo/cache-short)
 1 × requests[0].body.cache_control: want (absent), got {"ttl":"1h","type":"ephemeral"}   (kilo/cache-long)
```

Those 16 were regenerated alone, package by package, with `-update -run
'TestWireGoldens/.*/cache-(short|long)$'`. No other golden changed.

**Green:** `go test ./...`, all 30 packages `ok`.

**Plants** (`tools/plant.py`, 4 of 4 caught):

| Plant | FAIL line |
| :--- | :--- |
| the key sent without `Cache` | `TestAddPromptCacheKey`: `retention "", session "s-1": prompt_cache_key = s-1; want <nil>`; `TestCache_PerProvider`: `Cache false: prompt_cache_key = session-0039 (sent true); want sent false` |
| Grok's session not stored | `TestCache_PerProvider`: `Cache true: prompt_cache_key = <nil> (sent false)`; the goldens: `requests[0].body.prompt_cache_key: want "wire-session", got (absent)` |
| Kilo marking every model | `TestCache_PerProvider`: `Cache true: top-level cache_control sent true; want false` (`openai/gpt-6-luna`) |
| OpenCode's Messages route also sending the key | `TestCache_PerProvider`: `Cache true: prompt_cache_key = session-0039 (sent true); want sent false`; the goldens: `prompt_cache_key: want (absent), got "wire-session"` |

**Done differently from the steps' wording.** No decision changed:

* **One table test, not four.** The four per-package body tests are one
  table test, `TestCache_PerProvider` (`llmprovider/providers/cache_0039_test.go`).
  It covers every built-in provider, including the five that must send
  nothing. Kilo's two models are in it, so it is also
  `TestKilo_CacheControlForClaude`.
* **The fourth plant was replaced.** The PLAN's fourth plant, "the ChatGPT
  body sending the key twice", cannot fail: a body is a map, and the key
  already holds the same session. It was replaced by OpenCode's Messages
  route also sending the key, which can fail.

**Checks:**

* `make pre-add-check` on the 11 Go files: clean.
* markdownlint on `README.md` and `docs/architecture.md`: no issue.

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 3 as `a4169b3`.

### Phase 4: live confirmation and release (2026-10-13)

#### Deviations

**Deviation 1 (2026-10-13): the long paths read the short paths' entries.**

* **Found:** the first run sent all 16 approved requests. In it, `claude
  long`'s first request read `CachedTokens: 7082` and wrote nothing, so
  `CacheWrite1hTokens` was 0 and the test failed. It ran about five seconds
  after `claude short`, with the same system prompt and question, so it
  read the 5-minute entry `claude short` had just written. `kilo claude
  long` read the same way.
* **What the code did:** it sent `"ttl":"1h"`, as the goldens show, but
  the measurement never exercised a fresh one-hour write. The fault is the
  measurement's: the paths shared one prefix.
* **Decision (the owner):** each path's system prompt begins with its own
  line, so no path can read another's entry. Only the two long paths run
  again, four more billed requests. The first run stays recorded below.

#### The measurement

`llmprovider/live_0039_prompt_cache_measure_test.go`
(`TestLive_0039PromptCache`) ran with `LLMPROVIDER_LIVE_0039=1`, and with
`LLMPROVIDER_LIVE_TOGETHER=1` for the first run, which Together needs.

* **Keys:** all were checked for presence only.
* **Logs:** each request logs its decoded `Usage`, and no prompt, header or
  key.
* **The prompt:** each path sends one system prompt of about 3,000 words,
  7,086 Claude tokens. From the rerun on, each path's prompt begins with its
  own line (deviation 1).

**Run 1:** 16 billed requests, 7 of 8 paths passed.

| Path | Request 1 | Request 2 |
| :--- | :--- | :--- |
| Claude, short | write 7,082 | read 7,082 |
| Claude, long | read 7,082, no write (deviation 1) | read 7,082 |
| Zen Claude, short | write 7,082 | read 7,082 |
| Kilo Claude, short | write 7,082 | read 7,082 |
| Kilo Claude, long | read 7,082 (deviation 1) | read 7,082 |
| OpenAI `gpt-6-luna`, short | write 4,377 | read 4,377 |
| xAI `grok-4.6`, short | read 512 | read 512 |
| Together `zai-org/GLM-5.3-Flash`, short | none | read 4,544 |

```text
claude long: the first long request reported no one-hour write: {InputTokens:7086 … CachedTokens:7082 CacheWriteTokens:0 CacheWrite1hTokens:0}
```

**Run 2, after deviation 1:** the two long paths, each with its own
prompt. 4 billed requests, 2 of 2 passed.

| Path | Request 1 | Request 2 |
| :--- | :--- | :--- |
| Claude, long | write 7,093, all of it `CacheWrite1hTokens` | read 7,093 |
| Kilo Claude, long | write 7,096; Kilo sends no lifetime split, as 0036 found | read 7,096 |

**What it shows:**

* **Each path behaves as D2 to D4 expect.**
  * Every Claude path writes on the first request and reads on the second.
  * Claude's long write is an hour's.
  * OpenAI, on GPT-6, writes and reads.
  * Together reads on the second request, routed by the key.
  * xAI reads a 512-token prefix on both requests.
* **No path refused a field:**
  * Kilo took OpenRouter's top-level `cache_control`;
  * Zen took Anthropic's markers;
  * OpenAI, xAI and Together took `prompt_cache_key`.
* **Billed in all:** 20 requests, 16 approved and the 4 the deviation's
  decision added.

**The live checks, seen failing:**

* The one-hour check failed for real in run 1, as above.
* Both checks were moved into `path0039.problems`, and
  `TestPromptCacheChecks_0039` tests that function on canned usages, with
  no key and no network.
* Plants (`tools/plant.py`, with the `live_gateways` tag, 2 of 2 caught):

  | Plant | FAIL line |
  | :--- | :--- |
  | the read check never firing | `no read: problems "", want one containing "read nothing"` |
  | the hour check never firing | `a read where the hour was due: problems "", want one containing "no one-hour write"` |

**Step 3:** `README.md` Status names `v1.12.0`, with a bullet.

**Step 4, the checks:**

* `make pre-add-check` on the live test: clean.
* `make lint vuln parity-check dep-check coverage-check api-check
  generate-check records-check gate-selftest` exited 0:
  * lint: 0 issues, on the host and on Windows;
  * vuln: no vulnerabilities;
  * dep-check: 0 problems;
  * coverage-check: 31 packages, 0 problems;
  * api-check: 0 incompatible changes against `v1.11.0`;
  * generate-check and records-check: 0 problems.
* **Gate:** `tools/gate.py`, 23 of 23 passed.

#### Release notes: `v1.12.0`

`v1.12.0` lets a request ask for prompt caching (0039-MADR). The exported
API adds four identifiers and changes none. A request that does not set the
new field is sent exactly as before.

**New API:**

* `llmprovider.CacheRetention`, with `CacheShort` and `CacheLong`.
* `Request.Cache`. The zero value asks for nothing. Any other value is
  refused before the network with `ErrInvalidRequest`.

**What each provider sends:**

| Provider | With `Cache` set |
| :--- | :--- |
| `claude`, OpenCode's Claude routes | Anthropic's top-level `cache_control`, plus one breakpoint on the system prompt (sent as one text block) or else the last tool, with `ttl` `5m` (`CacheShort`) or `1h` (`CacheLong`) |
| `openai` (API key), `grok`, `together`, OpenCode's Responses route | `prompt_cache_key`, the provider's session id. These services cache on their own; the key routes a conversation to one cache |
| `kilo`, an `anthropic/` model | OpenRouter's top-level `cache_control`, with the same TTLs |
| `gemini`, `huggingface`, `ollama`, OpenCode's chat and Gemini routes | nothing |

**What it saves, measured:** a request repeating a 7,086-token prompt read
7,082 of those tokens from the cache. Anthropic charges a read at 0.1 times
the input price. `Usage.CachedTokens`, `CacheWriteTokens` and
`CacheWrite1hTokens` report what each request read and wrote.

**For gobble-cli:**

* Set `Cache: llmprovider.CacheShort`, or `CacheLong` across idle turns, on
  each conversation request.
* Set `WithSessionID` per session, so the routing key follows the
  conversation.

**Upgrading:** no change is needed. Caching is opt-in.
