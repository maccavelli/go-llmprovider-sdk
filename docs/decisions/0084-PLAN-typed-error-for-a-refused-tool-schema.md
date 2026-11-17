---
status: proposed
date: 2026-11-17
associated-madr: "0084-MADR-typed-error-for-a-refused-tool-schema.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Refused Tool Schema as Its Own Kind, Naming the Tool: ErrToolSchemaRefused and APIError.Tool

Associated MADR: [0084-MADR-typed-error-for-a-refused-tool-schema.md](0084-MADR-typed-error-for-a-refused-tool-schema.md)

## Goal

After this PLAN, in the release the owner names on 2026-11-17's terms (MADR
D9):

* a refusal of a tool's schema on Grok, OpenAI's Responses API, OpenCode
  Zen's Responses and Google routes, and Anthropic's Messages API is an
  `*APIError` matching `llmprovider.ErrToolSchemaRefused`, and through it
  `ErrInvalidRequest` (MADR D1, D3);
* its `APIError.Tool` names the refused tool when the service named exactly
  one tool the request sent, after deferral, and is empty otherwise (D2,
  D4);
* a refusal a provider has reworded stays a plain `ErrInvalidRequest`, and
  every other error keeps its kind (D3);
* the SDK sends every tool definition exactly as before: no rewrite, drop,
  resend or warning (D5);
* `Error()` leads with "llmprovider: tool schema refused" and never prints
  the tool (D6);
* the docs say all of it, in the same commit as the code (D8);
* the forms the MADR could not establish are recorded live by the owner,
  with no key ever shown to an agent (Phase 1).

The MADR's choices are the owner's answers of 2026-11-17 (the MADR's
"Decided"). The MADR and this PLAN are `proposed` until the owner accepts
the MADR and approves this PLAN, whole or by phase. Nothing below Phase 0
starts before that.

## Prerequisites

* 0080-PLAN Phase 3's code is committed on `main`, and no other agent holds
  staged or uncommitted work in the tree. Phase 1a starts on that commit.
  Phase 2 starts on Phase 1c's commit, which adds only the recorder and the
  fixtures to it; that commit is called **the base** below.
* The owner has keys for xAI, OpenAI, Anthropic, Gemini and OpenCode in a
  store of their own; no agent holds or reads any of them, and none is
  written to a file.

## Relation to 0080 and the other drafts

* **Order.** This PLAN runs after 0080-PLAN Phase 3 is committed and before
  0080-PLAN's release phase, so that all four records the owner answered on
  2026-11-17 can ship together on the owner's word (MADR D9). Its gates run
  one at a time with 0080's and the other three drafts' gates, never two at
  once on one host.
* **Files this PLAN shares with the others.** 0080's Phase 3 changed
  `CHANGELOG.md` `[Unreleased]` (an **Added** and a **Changed** entry),
  `docs/architecture.md` and `docs/guides/api-standards.md` (the package
  table only, at `b109cea`'s lines 44 and 64 to 65, read on the index on
  2026-11-17). This PLAN adds its own entries beside theirs and edits none
  of theirs. It touches none of 0080's Go files (read on the index on
  2026-11-17: 0080 changes no file under `llmprovider/providers/` but
  `kilo/`, no file under `llmprovider/internal/wire/`, and not
  `api_error.go` or `wizard/describe.go`).
* **What Phase 2 re-reads at the base,** recording each line in the
  execution record before any edit, since this PLAN's claims below are at
  `b109cea`:
  * `llmprovider/api_error.go`: `APIError` (P2), `classifyAPIError` (P1),
    `parseAPIErrorBody` (P3), and the sentinels' block;
  * `llmprovider/internal/wire/retry.go` (P25), `post.go`, `sse.go`'s
    `StreamEventsRetry`, `tools.go` (P5, P6) and `deferral.Tools` (P4);
  * each send path: Grok (P7, P8), OpenAI (P9 and its stream), Claude
    (P10 and its stream), OpenCode's non-Messages routes (P11) and its
    Google declarations (P12);
  * `wizard/describe.go`'s `errorClasses` (P20);
  * `docs/guides/api-standards.md` R24 and R25 (P21, P22),
    `docs/architecture.md`'s error paragraphs (P16 and the "Errors" bullet),
    `CHANGELOG.md` `[Unreleased]` as 0080 left it (P17);
  * the other three drafts' PLANs, if placed, for any file they share with
    this one;
  * 0080-PLAN's execution record, for which hosts its gates used.
  A line that moved is recorded with its new number; a line whose meaning
  changed is a stop.

## Scope

### In scope

* `llmprovider/api_error.go`: the sentinel, the field and its doc, the
  classification rows of MADR D3, an unexported read of an OpenAI-style
  error's `param` (C15), `Error()` unchanged in form (D6).
* `llmprovider/internal/wire/`: one unexported helper that names the
  refused tool from a classified error and the list a request sent (D4),
  and its tests.
* `llmprovider/providers/{grok,openai,claude,opencode}/`: each send path
  passes its sent list to the helper where it returns an error, for
  `Generate` and `Stream`; each package doc says which refusals it reads.
* `wizard/describe.go`: `errorClasses` gains the kind before
  `ErrInvalidRequest`.
* Phase 1's recorder: `llmprovider/live_0084_toolschema_measure_test.go`
  (tag `live_gateways`), `llmprovider/toolschema_0084_record_test.go`
  (untagged: the case table and the record writer), and its offline test;
  the fixtures, under `llmprovider/internal/wire/testdata/toolschema-0084/`.
* Docs (D8): `docs/guides/api-standards.md` R24 and R25,
  `docs/architecture.md`, `CHANGELOG.md` `[Unreleased]`, the package docs
  above; a dated amendment of the MADR's provider table after Phase 1.

### Exported names (R55)

| Name | Kind | Call site |
| :--- | :--- | :--- |
| `llmprovider.ErrToolSchemaRefused` | `var ... error` | gobble-cli's `llm/provider` adapter, where it tests an `*APIError` before its signature match (MADR G5, D10) |
| `llmprovider.APIError.Tool` | `string` field | the same adapter, to name the tool it drops (D10) |

Nothing else is exported. `make api-check` lists exactly these two
additions; any other line it prints is a stop.

### Out of scope

* Rewriting, dropping or resending anything (MADR D5).
* Routes with no recorded refusal (D3): the Chat Completions servers,
  OpenCode Go's Messages route, Gemini's Interactions, the ChatGPT plan
  session and `chatcompat`, unless Phase 1 records one, which is a stop.
* A refusal inside a 200 event stream (`ClassifyStreamFailure`): none is
  recorded.
* gobble-cli's changes (Phase 6 names them; gobble's record makes them).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **Fixtures only.** Every matcher and every test that feeds a provider's
  reply uses a recorded body: the 0062 fixtures (MADR F1 to F4, F7, F10) or
  Phase 1's. A test may alter a recorded body only to break its marker (the
  staleness guards) or to rename its tool (the redaction and index cases),
  and says so in its comment.
* **Red, then green.** Each new test is seen red twice, on scratch clones,
  with `python3 -B tools/redgreen.py RED.json --expect fail --rev <rev>`:
  at the base (a build failure, quoted, since the names do not exist) and
  at **the stub** (the base plus the two exported names declared and
  nothing classified, a scratch commit never pushed), where each fails by
  its assertion, quoted. A guard, a test that must hold at the base, is
  green at the stub with `--expect pass`. Every test is green on the phase's
  tree. Each is seen to fail on a planted break with `python3 -B
  tools/plant.py PLANTS.json`, on a scratch clone, never in the tree. A
  break no test catches is a deviation.
* **Heavy runs queue.** Every `-race` run, every multi-package or
  multi-round run and the gate run through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` on
  the session host, or through the skill's `scripts/run-on-host` on the
  host `~/.agents/skills/available-dev-hosts/scripts/pick-host` prints. A
  multi-round `-race` run carries an explicit `-timeout`. No `-race` run
  goes to a WSL distro. A single-package `go test` without `-race` may run
  on the session host.
* **Pre-add.** `make pre-add-check FILES="..."` on every changed Go file,
  through `run-heavy`, since it lints and tests the packages.
* **No outside host in a test** (0065-MADR): replies come from `httptest`
  or a `RoundTripper` in the test; the packages' `localonly` guard stays on.
* **No credential in a record, a test or a log.** The live recorder reads
  each key from its variable and hands it to the provider; no `t.Log`,
  `t.Logf`, `t.Error`, `t.Errorf`, `t.Fatal`, `t.Fatalf` or `t.Skip` call
  is given a key, a variable holding one, an error's text or a body.
* **Exit statuses** are captured before any filter, into a scratch file.
* **Stop conditions,** each a stop and a prompt to the owner, with the
  evidence and resolutions that leave the tree correct:
  * Phase 1: a route that refused names no tool; a refusal's form differs
    from its 0062 fixture's (status, code, marker or where the name sits);
    a route that accepted a root string in 0062 refuses now; Grok's second
    model orders tag and name differently; a written record holds a value
    the redaction proof says it must not;
  * an existing test, golden or fixture needs a change;
  * `make api-check` lists anything but the two names;
  * a matcher needs a field `*APIError` does not keep and this PLAN does
    not add (as 0062-MADR D4 says of `param`);
  * a test cannot be red at the stub without changing code that is not a
    test.

## Implementation Steps

### Phase 0: records

Model: lightweight, low.

1. The coordinator places this pair after 0080's Phase 3 commit, as the
   drafting's `PLACEMENT.md` says: the number from `python3 -B
   scripts/check_records.py --next`, written by `place.py`, the index rows
   by `python3 -B tools/records.py add`, committed under the bootstrap
   exception (records only).
2. When the owner accepts the record and approves this PLAN,
   `tools/records.py status` sets the MADR `accepted` and this PLAN
   `in-progress`, each with its index row.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --markdown`, then stage the records and `docs/README.md`
   and commit (`git commit --no-edit`).

Acceptance: `make records-check links-check` pass; `python3 -B
tools/claims.py` passes on both records; no source changed. Docs: the
records are the docs.

### Phase 1: the owner's live steps (O1 to O6)

Billed: one generation per case, at most 64 output tokens; 51 cases with
every key set (eight routes of six cases, and `grok2`'s three; not
established whether a refused request is billed).
No agent runs the live test and no agent holds a key.

#### Phase 1a: the recorder (an agent)

Model: mainstream, medium (a recorder in the 0062 form, P13, P14, P24, with
a redaction proof).

Files: `llmprovider/toolschema_0084_record_test.go` (untagged),
`llmprovider/toolschema_0084_record_offline_test.go` (untagged),
`llmprovider/live_0084_toolschema_measure_test.go` (`//go:build
live_gateways`).

1. **The case table**, in the untagged file, each case a subtest named
   `<route>-<shape>`, the shape carrying its step where it has one
   (`grok-root-string`, `grok-O1-root-union`, `claude-O2-two-bad`). Every
   route runs `root-string` and O1 to O5; `grok2` runs O6's three. Routes,
   each with the model its 0062 fixture used unless the agent finds it no
   longer listed (then the nearest listed model, named in the execution
   record): `grok` (`grok-4.6`, the owner's),
   `grok2` (`grok-4.5`, F1's), `openai` (`gpt-6-luna`), `zen-responses`
   (`gpt-5.5`), `claude` (`claude-sonnet-5`), `zen-google`
   (`gemini-3.8-flash`); and, observed but not read (D3), `gemini`
   (`gemini-3.8-flash`), `zen-chat` (`glm-5.3`), `go-messages`
   (`qwen3.8-flash`). Every case offers a valid control tool `noop` first
   and the case's tool second, so a refused tool sits at index 1, never 0.
   Shapes:
   * `root-string`: `record_city`, schema `{"type":"string"}`, 0062's case
     again;
   * **O1** `O1-root-union`: `record_city`, the owner's shape: an object with
     `city` and `country`, `additionalProperties: false`, and a root `anyOf`
     of `{"required": ["city"]}` and `{"required": ["country"]}`;
   * **O2** `O2-two-bad`: `record_city` (root union) and `record_country`
     (root string), both after `noop`;
   * **O3** `O3-typed-branches`: the O1 schema with each branch given
     `"type": "object"` and `"properties": {}`;
   * **O4** `O4-nested-union`: an object whose property `target` is an
     `anyOf` of a string and an object;
   * **O5** `O5-history-dropped`: `noop` only offered, the input holding a
     prior call of `record_city` and its result;
   * **O6**: `grok2` runs `root-string`, `O1-root-union` and `O2-two-bad`
     on a second Grok model.
2. **The record**, one JSON file per case, written to the directory
   `LLMPROVIDER_LIVE_0084_OUT` names, which must be outside the module (the
   0062 check, P13). It holds: the case id, route, model, the tools' names
   and schemas sent, the input text, the status, the `*APIError`'s kind
   name, `Code` and `Message` (already redacted by the SDK, MADR C12), the
   error body, the outcome (`refused`, `accepted` or `other`), and for an
   accepted case its token counts and number of tool calls. It never holds
   a request header, the key, the model's text or a response header.
3. **What is redacted in the body**, before it is cut to 4 KiB as 0062's
   is (P14): `internal/redact.String` (keys and tokens, bearer and basic
   credentials, key-value secrets, signed URLs); then, by the repository's
   fixture rules (P28): every `request_id` and `req_...` value to
   `req_REDACTED`, an account, user or organization id to its prefix and
   `REDACTED`, a UUID that is not a public object's to the nil UUID, an
   e-mail address to `user@example.com`. 0062 redacted request ids by hand
   after its run (P29); here the writer does it.
4. **The redaction proof**, `TestToolSchemaRecord_Redacts_0084`, offline:
   it writes a record from a recorded refusal body (F4, Claude's) into
   which it plants a made-up `xai-` key of 48 characters, a made-up
   `sk-proj-` key, a `Bearer` value of 40 characters, `"api_key": "..."`, a
   made-up `req_` id, a UUID and an e-mail address, and asserts the written
   file holds none of the planted values and holds `req_REDACTED`. Seen red
   by a plant that skips the redaction (`tools/plant.py`), quoted; green
   on the tree.
5. **The live test**, `TestLive_ToolSchema_0084`: skips without
   `LLMPROVIDER_LIVE_0084`; fails without `LLMPROVIDER_LIVE_0084_OUT` or
   with one inside the module; for each case, sends when its route's key
   variable is set (`ProviderEnvVars()`, P18, P19) and records `skipped`
   when it is not; builds each provider with `providers.New`,
   `WithModelProbes(false)` and `WithoutModelMetadata()`; one `Generate`
   per case, no `WithRetry`. It logs, per case, only `"%s: %s, status %d"`
   (the id, the outcome, the status).
6. **Before the owner runs it,** the agent reads the three files and
   confirms in the execution record that no log or failure call is given a
   key, a variable holding one, an error's text or a body; runs `go vet
   -tags live_gateways ./llmprovider/` and `go test -count=1 -run
   'ToolSchemaRecord' ./llmprovider/` (session host, no `-race`); and
   commits the three files with `make pre-add-check` through `run-heavy`.

Acceptance: the offline test green, its plant caught; the live test
compiles; no key in any file.

#### Phase 1b: the owner's run

The owner runs it, never an agent, at the commit that adds the recorder, in
a new terminal window that is closed afterwards. Each `read -rs NAME` line
waits, showing nothing, for one value: the owner types or pastes it and
presses Enter. Nothing is echoed; the value lives only in that shell's
environment and the `go test` process it starts, and is never written to a
file; the lines typed hold no key, so the shell's history holds none. The
commands work as written in zsh and in bash, as 0080's did (P23). Nothing
else is typed in that window: no `echo`, `printenv`, `env`, `set` or `set
-x`. A key the owner does not have is skipped: its route is recorded
`skipped`.

1. Open the checkout and confirm the commit:

   ```sh
   cd <the go-llmprovider-sdk checkout>
   git log -1 --format=%h
   ```

2. The keys, each at a hidden prompt:

   ```sh
   read -rs XAI_API_KEY
   export XAI_API_KEY
   read -rs OPENAI_API_KEY
   export OPENAI_API_KEY
   read -rs ANTHROPIC_API_KEY
   export ANTHROPIC_API_KEY
   read -rs GEMINI_API_KEY
   export GEMINI_API_KEY
   read -rs OPENCODE_API_KEY
   export OPENCODE_API_KEY
   ```

3. A directory for the records, outside the checkout:

   ```sh
   mkdir -p "$HOME/tool-schema-0084"
   ```

4. The recorder, every case:

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$' -count=1 -v
   ```

   One case again, if a line says `other` (a timeout or a 5xx):

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$/^grok-O1-root-union$' -count=1 -v
   ```

5. Remove the values from the shell, then close the window:

   ```sh
   unset XAI_API_KEY OPENAI_API_KEY ANTHROPIC_API_KEY GEMINI_API_KEY OPENCODE_API_KEY
   ```

6. Tell the coordinator the run is done; the records stay in
   `$HOME/tool-schema-0084` for Phase 1c.

What each step decides:

| Step | Decides | No SDK code changes when |
| :--- | :--- | :--- |
| `root-string` | that each 0062 form still holds; the matchers' fixtures stand | always: a different form is a stop |
| O1 | the form of the owner's shape on each route; its record joins the route's fixtures in Phase 2 | the route accepts it (recorded, not read) |
| O2 | whether a route names one bad tool per reply or several; so whether the "several named" test is built (MADR D2) and what a host's retry costs | always |
| O3 | whether xAI and the others accept typed branches: evidence for gobble's normalisation and for revisiting MADR D5 (Q5's answer) | always: this PLAN rewrites nothing |
| O4 | whether a union below the root is refused, and in what form | it is accepted, or refused in its route's O1 form |
| O5 | whether a history holding a call of a tool no longer offered is accepted: the safety of gobble's drop (MADR D10) | always: gobble's record acts on it |
| O6 | that Grok's tag and name keep their order on a second model | the order holds |

Stop conditions (each a stop and a prompt to the owner before Phase 1c):

* a route that refuses names no tool, or names it where its matcher does not
  read;
* a refusal's form differs from the 0062 fixture's for its route: another
  status, code or marker;
* a route that accepted a root string in 0062 refuses now, or O4 is refused
  in a form unlike its route's O1;
* O6 shows a different order;
* Phase 1c's checks find a value the redaction should have removed.

#### Phase 1c: the fixtures (an agent)

Model: mainstream, low.

1. Copy the records from `$HOME/tool-schema-0084` to
   `llmprovider/internal/wire/testdata/toolschema-0084/<wire>/`, unchanged.
2. `make fixtures-check` and `python3 -B tools/denyscan.py --changed`, each
   exit 0; a hit is a stop, never a baseline entry.
3. The execution record gets one line per case (id, outcome, status, code,
   whether and where the tool is named), and the MADR a dated amendment of
   its provider table: each "not established" cell Phase 1 settles, with
   its fixture.
4. Commit the fixtures and the two records' edits.

Acceptance: every case recorded or `skipped`; no stop condition met, or
the owner's resolution recorded as a dated deviation.

### Phase 2: the tests, the code and the docs, one commit

Model: mainstream, high (two names under v1, four send paths, and an index
that must be mapped through the list as sent after deferral).

Files: `llmprovider/api_error.go`; `llmprovider/internal/wire/` (the helper
and its test); `llmprovider/providers/{grok,openai,claude,opencode}/`;
`wizard/describe.go`; the tests below; the docs of step 6.

1. **Re-read the base** (Relation to 0080) and record the lines.
2. **The stub**, on a scratch clone only: the base plus
   `ErrToolSchemaRefused` declared as D1 says and `Tool string` on
   `APIError`, nothing classified and nothing named.
3. **The tests.** Each built from fixtures (Rules), each in a `_0084_test.go`
   file of its package.

   | Test | Asserts | At the stub |
   | :--- | :--- | :--- |
   | `TestToolSchemaRefusal_Classify_0084` (`llmprovider`) | each recorded refusal of D3's routes, 0062's and Phase 1's, through `ClassifyHTTPError`: `errors.Is` the kind and `ErrInvalidRequest`, `Retryable()` false, `Tool` empty, `Error()` beginning with `llmprovider: tool schema refused:` and not holding the tool's name twice | red |
   | `TestToolSchemaRefusal_Names_0084` (`providers`) | each route, through `Generate` and `Stream`, against an in-process server replaying the fixture: `Tool` is the refused tool, which sits at index 1 | red |
   | `TestToolSchemaRefusal_NeverNamesUnsent_0084` (`providers`) | the kind, with `Tool` empty, when the reply names a tool the request did not send, a deferred tool that was not sent, Grok's `use_tool` wrapper, an index past the sent list, or OpenAI's native `tool_search` entry | red for the kind, green for the empty `Tool` (both asserted) |
   | `TestToolSchemaRefusal_Deferral_0084` (`providers`) | with deferred tools, Claude's and OpenAI's indices map through the list as sent (P4 to P6), not through `Request.Tools` | red |
   | `TestToolSchemaRefusal_Reworded_0084` (`llmprovider`, one subtest per route) | the staleness guard: each fixture with its marker changed (Grok's tag renamed; OpenAI's code changed; Anthropic's `tools.N.` path or `input_schema` renamed; the Google route's phrase changed) is `ErrInvalidRequest`, not the kind, with `Tool` empty | green (a guard) |
   | `TestToolSchemaRefusal_OthersUnchanged_0084` (`llmprovider`) | every other recorded 400 under `testdata/refusals-0062/` (efforts, empty input) keeps its kind; Grok's refused key stays `ErrAuthFailure` | green (a guard) |
   | `TestToolSchemaRefusal_LastSendKeepsName_0084` (`providers`) | Grok refuses the effort sent (its 0062 fixture), the request is resent, and the resend is refused for its schema: `Tool` is named, though the resend hook is not consulted after it (MADR D4, C27) | red |
   | `TestToolSchemaRefusal_RedactedNeighbour_0084` (`providers`) | F1 with its tool renamed `lookup_api_key`, whose next word redaction removes (MADR X1): `Tool` is `lookup_api_key` | red |
   | `TestToolSchemaRefusal_SeveralNamed_0084` (`providers`) | built only if O2 records a reply naming two tools: the kind, with `Tool` empty; otherwise not built, and the execution record says why | red, if built |
   | `TestDescribe_ToolSchemaRefused_0084` (`wizard`) | `describe` names the class, "tool schema refused, HTTP 400", before "invalid request" | red |

4. **Red.** `tools/redgreen.py` at the base (build failures) and at the stub
   (assertions), each failure quoted; the guards `--expect pass` at the
   stub.
5. **The code:**
   * `api_error.go`: the sentinel (D1) beside `ErrContextOverflow`; the
     field with D2's doc; the unexported `param` read in
     `parseAPIErrorBody` (P3), kept on the envelope, never exported; the
     rows of D3 in `classifyAPIError` (P1), each for its service only and
     after the Grok key row (MADR C13), so a refused key stays
     `ErrAuthFailure`.
   * `internal/wire`: one unexported helper taking a classified error and
     the list of tools the body held, returning the error with `Tool` set
     when the refusal names, by name or index, exactly one entry of that
     list that is a caller's tool, and the error unchanged otherwise. The
     list each provider passes is the one its encoder used, computed once,
     so the two cannot differ.
   * Grok (P7, P8), OpenAI (P9 and its stream), Claude (P10 and its
     stream), OpenCode's non-Messages routes (P11, P12): the helper where a
     send's error is returned and where a stream yields its error before
     any event.
   * `wizard/describe.go` (P20): the kind before `ErrInvalidRequest`.
6. **Docs, in the same commit** (MADR D8):
   * `docs/guides/api-standards.md`: R24 lists `Tool` among the fields
     (P21); R25 lists `ErrToolSchemaRefused` beneath `ErrInvalidRequest`
     (P22); both cite 0084-MADR D1 and D2;
   * `docs/architecture.md` (P16): the kind, which routes it is read for,
     that `Tool` is only a sent tool's name, and that a reworded refusal
     falls back to `ErrInvalidRequest`; its "Errors" bullet names `Tool`;
   * `CHANGELOG.md` `[Unreleased]` (P17), **Added**:
     "`llmprovider.ErrToolSchemaRefused`, beneath `ErrInvalidRequest`, for a
     service's refusal of a tool's schema, read on Grok, OpenAI's Responses
     API, OpenCode Zen's Responses and Google routes and Anthropic's
     Messages API; and `APIError.Tool`, the refused tool's name when the
     service named exactly one tool the request sent, else empty. Such an
     error's text now begins "llmprovider: tool schema refused". A refusal
     in another form stays `ErrInvalidRequest` (0084-MADR)";
   * package docs: `llmprovider` (the kind and the field), and `grok`,
     `openai`, `claude`, `opencode`, each naming the refusal it reads;
   * the `llmprovider` package's example, if R50's check asks for one for
     the kind (`go run ./llmprovider/internal/cmd/sdkaudit -check`).
7. **Before staging,** on a scratch clone of the phase's tree: `go test
   -count=1 ./llmprovider/ ./llmprovider/internal/wire/ ./wizard/` and each
   provider package (session host, no `-race`); every existing test passes
   unchanged, or the phase stops.

Verification: the tests green; `~/.agents/skills/available-dev-hosts/scripts/run-heavy --
make pre-add-check FILES="<each changed Go file>"`; `make api-check` lists
exactly `ErrToolSchemaRefused` and `APIError.Tool` as compatible additions;
`python3 -B tools/apidiff.py v1.31.0` lists the same two; `go run
./llmprovider/internal/cmd/sdkaudit -check`; `make docs-check tables-check
changelog-check links-check records-check markdownlint`.

Acceptance: every row as its table says; no existing test changed; the docs
say what the code does, the fall-back included.

### Phase 3: planted breaks and the gates

Model: mainstream, low.

1. `python3 -B tools/plant.py PLANTS.json` on a scratch clone, each plant
   caught, each failure quoted:
   * each route's row removed from `classifyAPIError`: caught by `Classify`
     and `Names` for that route;
   * a row loosened to any `invalid-argument` 400: caught by
     `OthersUnchanged` (a refused effort) and `Reworded`;
   * the name taken from the reply's text without the sent list: caught by
     `NeverNamesUnsent`;
   * the index mapped through `Request.Tools`: caught by `Deferral`;
   * the name filled in the resend hook instead: caught by
     `LastSendKeepsName`;
   * the kind made a plain sentinel, not beneath `ErrInvalidRequest`:
     caught by `Classify`;
   * `Error()` printing `Tool`: caught by `Classify`;
   * the kind missing from `errorClasses`: caught by
     `TestDescribe_ToolSchemaRefused_0084`;
   * the recorder's redaction skipped: caught by
     `TestToolSchemaRecord_Redacts_0084`.
2. `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and
   `--expect pass` on the tree, quoted.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=3 -timeout 45m ./llmprovider/... ./wizard/...`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
5. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
6. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
7. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/denyscan.py --changed`; `python3 -B
   tools/recordlint.py --changed`.

Acceptance: every plant caught; every command exit 0, each status taken
before any filter. Docs: none beyond Phase 2's; a gate that finds a docs gap
is a deviation.

### Phase 4: independent review

Model: strongest, high. A fresh agent that did not write Phases 1 to 3,
briefed with the MADR, this PLAN and the diff from the base, not with the
implementer's conclusions.

1. It reads the diff whole and checks, with a quoted line for each finding:
   that `Tool` can only hold a sent caller tool's name, on every path,
   streams included; that no response text reaches `Tool` or a log; that a
   reworded refusal and every other error keep their kind; that the docs
   and the changelog say what the code does; that `make api-check`'s two
   names are all that is exported; that the recorder can print no key.
2. Each finding is fixed in a commit within this phase, its test seen red
   first, or put to the owner with its evidence. None is waved through.

Acceptance: the review's findings, and what was done with each, in the
execution record.

### Phase 5: the release (the owner's word)

Model: mainstream, medium. No step runs under a standing approval: the tag
and the GitHub Release each wait for the owner's word in the same turn.

1. With the owner's word, the release that carries 0080 and the other
   records ready by then (MADR D9) is cut by 0080-PLAN's release phase;
   this PLAN adds nothing but its changelog entry, already under
   `[Unreleased]`. If this PLAN is not finished, its gates not passed, when
   the owner gives the word, it moves to the release after, by the same
   steps.
2. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`, quoted.
3. `tools/records.py status` sets this PLAN `complete` once the tag's CI
   and step 2 are recorded.

Acceptance: the tag's CI passes; the section and the Release name the kind.

### Phase 6: gobble, after the release (gobble-cli's own record)

Not executed under this PLAN; recorded so the hand-off is exact. At
gobble-cli `cecc892` (P27), by a gobble-cli record:

* `go.mod`: the SDK pin from `v1.31.0` to the release.
* `llm/provider`: on `ErrToolSchemaRefused` with `Tool` set, before any
  event, tested before the existing signature match (P26): drop that tool
  for the rest of the session on that provider and model, resend once, and
  tell the user which tool was left out and why; with `Tool` empty, end the
  turn naming the class; at most one drop per direct tool per turn (MADR
  D10).
* Its load-time normalisation of root unions (the owner's earlier gobble
  ruling) stays first; O3's record says whether typed branches suffice.
* O5's record says whether the drop must also rewrite a history that calls
  the dropped tool.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`.

* `go test -count=1 -run 'ToolSchemaRecord' ./llmprovider/`; `go vet -tags
  live_gateways ./llmprovider/`
* the owner's run (Phase 1b), then `make fixtures-check` and `python3 -B
  tools/denyscan.py --changed`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and at
  the stub, then `--expect pass`; `python3 -B tools/plant.py PLANTS.json`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=3 -timeout 45m ./llmprovider/... ./wizard/...`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`: the two names
* `go run ./llmprovider/internal/cmd/sdkaudit -check`
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN
* Phase 4's review, recorded

## Rollout and Rollback

* **Order.** Phase 0 when placed; Phase 1a on approval, on 0080 Phase 3's
  commit; 1b when the owner chooses; 1c after it; Phase 2 on 1c's commit,
  the base; Phases 3 and 4
  after it; Phase 5 with the shared release, on the owner's word; Phase 6
  in gobble-cli after the tag.
* **Rollout.** A caller sees nothing new unless a provider refuses a tool's
  schema on a read route: then `errors.Is(err, ErrInvalidRequest)` still
  holds, `errors.Is(err, ErrToolSchemaRefused)` now does, `Tool` may name
  the tool, and the error's text begins "llmprovider: tool schema refused"
  where it began "llmprovider: invalid request". The wire is unchanged.
* **Rollback before the tag:** revert Phase 2's commit; Phase 1's recorder
  and fixtures may stay (they change no behaviour); the records stay, with
  a dated note.
* **Rollback after the tag:** the names are under v1 and stay; a patch
  release may stop classifying a route (its errors fall back to
  `ErrInvalidRequest`, as a reworded refusal does), by a dated amendment of
  the MADR. Nothing persistent changes shape.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, at the
revisions the marker names; Phase 2 re-reads each at the base.

<!-- claims: self=b109cea gobble-cli=cecc892 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The classification table | `llmprovider/api_error.go:538` | `func classifyAPIError(` |
| P2 | `APIError` | `llmprovider/api_error.go:103` | `type APIError struct {` |
| P3 | The error body's parser | `llmprovider/api_error.go:690` | `func parseAPIErrorBody(` |
| P4 | The tools a request sends, after deferral | `llmprovider/internal/wire/deferral/deferral.go:253` | `func Tools(` |
| P5 | Responses tools with native deferral | `llmprovider/internal/wire/tools.go:85` | `func AddResponsesToolsNative(` |
| P6 | Messages tools with native deferral | `llmprovider/internal/wire/tools.go:101` | `func MessagesToolsNative(` |
| P7 | Grok's send | `llmprovider/providers/grok/grok.go:252` | `return wire.Send(ctx, string(llmprovider.ProviderGrok)` |
| P8 | Grok's stream | `llmprovider/providers/grok/grok.go:297` | `wire.StreamEventsRetry(ctx, string(llmprovider.ProviderGrok)` |
| P9 | OpenAI's send | `llmprovider/providers/openai/openai.go:317` | `return wire.Send(ctx, string(llmprovider.ProviderOpenAI)` |
| P10 | Claude's send | `llmprovider/providers/claude/claude.go:283` | `resp, err := wire.Send(ctx, string(llmprovider.ProviderClaude)` |
| P11 | OpenCode's non-Messages routes' send | `llmprovider/providers/opencode/opencode.go:322` | `return wire.Send(ctx, "opencode", p.src, build, p.decode(c), effort.Check)` |
| P12 | OpenCode's Google declarations | `llmprovider/providers/opencode/opencode.go:765` | `jsonKeyParameters: wire.ToolSchema(tool.Schema)}` |
| P13 | 0062's recorder keeps records outside the module | `llmprovider/refusals_0062_record_test.go:465` | `func outsideModule(` |
| P14 | and cuts a body to 4 KiB | `llmprovider/refusals_0062_record_test.go:61` | `const refusalBodyLimit` |
| P15 | `llmtest.Run` checks status-to-kind | `llmprovider/llmtest/llmtest.go:125` | `status-to-kind classification` |
| P16 | The architecture's error paragraph | `docs/architecture.md:563` | `` - `APIError` carries a `Kind`, one of the sentinels. `` |
| P17 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| P18 | xAI's key variable | `llmprovider/provider.go:100` | `ProviderGrok:   "XAI_API_KEY",` |
| P19 | OpenCode's key variable | `llmprovider/provider.go:103` | `ProviderOpencodeZen: "OPENCODE_API_KEY",` |
| P20 | The wizard's kinds | `wizard/describe.go:15` | `var errorClasses = []error{` |
| P21 | R24 | `docs/guides/api-standards.md:214` | `` - **R24. `*APIError` is the only structured error.** `` |
| P22 | R25 | `docs/guides/api-standards.md:217` | `` - **R25. Test errors by kind with `errors.Is`.** `` |
| P23 | 0080's owner commands, the form copied | `docs/decisions/0080-PLAN-verify-a-kilo-credential-through-its-profile.md:219` | `read -rs KILO_API_KEY` |
| P24 | 0062's live recorder | `llmprovider/live_0062_refusals_measure_test.go:28` | `func TestLive_Refusals_0062(t *testing.T) {` |
| P25 | The resend hook is not consulted after the last send | `llmprovider/internal/wire/retry.go:20` | `nor once MaxSends sends have been made` |
| P26 | gobble's signature match | `gobble-cli:llm/provider/provider.go:367` | `if canRetry && !sent && signatureRefused(mapError(err)) {` |
| P27 | gobble's pin | `gobble-cli:go.mod:17` | `github.com/maccavelli/go-llmprovider-sdk v1.31.0` |
| P28 | The fixture redaction rule | `AGENTS.md:278` | `` an account, user or organization id becomes its prefix and `REDACTED` `` |
| P29 | 0062 redacted request ids by hand | `docs/reports/0062-REPORT-thinking-refusals.md:31` | `Claude's five request ids and Together's one were redacted after the run` |

## Execution record

### Drafting (2026-11-17)

* **Provenance.** Drafted by a strongest-tier model at high effort, on the
  coordinator's brief, after the owner answered the MADR's Q1 to Q7 (each
  as recommended, relayed by the coordinator); from read-only
  investigation; nothing implemented, nothing written to any repository's
  tree or index. The MADR and this PLAN are `proposed`, with `0084` for the
  number, which the placing step takes from `scripts/check_records.py
  --next`.
* **Verified, and how.**
  * Read at `b109cea` (`git show` and a `git archive` copy in a scratch
    folder): the error model, the wire encoders and send paths, the 0062
    recorder and fixtures, the wizard's kinds, the standards guide, the
    architecture and the changelog. Read on the index of the tree as it
    stood on 2026-11-17 (`git diff --cached`): 0080's staged changes to the
    standards guide (the package table only), the changelog and the
    architecture, and the list of files 0080 stages.
  * The MADR's evidence index (85 rows) and this PLAN's code claims (29
    rows) checked with `tools/claims.py` from a scratch copy, against the
    revisions their markers name, each repository given by `--repo`.
  * A scratch run on an archive of `b109cea` (the MADR's X1): what
    `ClassifyHTTPError` makes of each recorded refusal, and of tool names
    that redaction touches.
  * `tools/claims.py --plants`: three false rows planted in each record (a
    wrong line in this repository, a wrong text or line in another), each
    caught.
  * Placement rehearsed on a scratch clone at `b109cea` (number `0081`
    there): `place.py` wrote both files; `tools/records.py add` wrote the
    two rows and the count went from 168 to 170; `scripts/check_records.py`,
    `scripts/check_links.py` and `scripts/check_disclosure.py` exit 0;
    `tools/recordlint.py` exit 0 on both, after it had caught one MD038 in
    this PLAN, which was fixed; `tools/claims.py` 85 of 85 and 29 of 29. A
    planted wrong `0084` count made `place.py` refuse before writing either
    file.
* **Not verified.** Any live form beyond the 0062 fixtures and the owner's
  paste (Phase 1 exists for them); the base, which does not exist yet; that
  the 0062 models are still listed; that `sdkaudit -check` asks for an
  example for a new sentinel; any `-race` run, any gate, any host but the
  session's.
