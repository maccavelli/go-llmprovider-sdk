---
status: in-progress
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
* the forms the MADR could not establish are recorded live ~~by the owner~~,
  with no key ever shown to an agent (Phase 1). *(Annotated 2026-11-20:
  recorded by an agent with the keys in its process environment, by the
  owner's ruling "API keys from the environment only (Recommended)",
  relayed by the coordinator.)*

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
  *(Annotated 2026-11-20 by the owner's ruling, relayed by the
  coordinator: "API keys from the environment only (Recommended)". The
  keys are set in the environment of the shell the coordinator runs in
  (checked by name only on 2026-11-20: `ANTHROPIC_API_KEY`,
  `GEMINI_API_KEY`, `OPENAI_API_KEY`, `OPENCODE_API_KEY`, `XAI_API_KEY`).
  An agent or the coordinator starts the recording process, which inherits
  them; no person or agent displays, reads, copies, measures or compares a
  value, and none is written to a file.)*

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
*(Annotated 2026-11-20: `make api-check` lists incompatible changes only;
the check is the one the Rules' struck `make api-check` stop now names:
`tools/apidiff.py` from the phase's start commit to the tree.)*

### Out of scope

* Rewriting, dropping or resending anything (MADR D5).
* Routes with no recorded refusal (D3): the Chat Completions servers,
  OpenCode Go's Messages route, Gemini's Interactions, the ChatGPT plan
  session and `chatcompat`, unless Phase 1 records one, which is a stop.
  *(Annotated 2026-11-20 by the owner's rulings on N1 and N3, relayed by
  the coordinator: no stop; the kind applies on any route of a read
  service whose reply carries the provider's marker, OpenAI's sessions and
  OpenCode's other routes included, and OpenCode Zen's Messages route is
  recorded in Phase 1. `chatcompat` and the other services stay unread.)*
* A refusal inside a 200 event stream (`ClassifyStreamFailure`): none is
  recorded.
  *(Annotated 2026-11-20 by the owner's ruling on N4, relayed by the
  coordinator: "HTTP error replies only, with a guard test (Recommended)".
  The D3 rows apply in `ClassifyHTTPError` only; a failure in a 200 body or
  a stream keeps today's kind, though `setStreamKind` calls
  `classifyAPIError` with a status (MADR H2). The guard is
  `TestToolSchemaRefusal_StreamUnchanged_0084`.)*
* gobble-cli's changes (Phase 6 names them; gobble's record makes them).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **Fixtures only.** Every matcher and every test that feeds a provider's
  reply uses a recorded body: the 0062 fixtures (MADR F1 to F4, F7, F10) or
  Phase 1's. A test may alter a recorded body only to break its marker (the
  staleness guards) or to rename its tool (the redaction and index cases),
  and says so in its comment.
  *(Annotated 2026-11-20 by the owner's ruling on N7, relayed by the
  coordinator: "Named edits to a recorded body, each stated in a comment
  (Recommended)", the tool's position included. A test may also move the
  refused tool's position (Anthropic's `tools.0.` to `tools.1.`), add a
  second named tool or index to one reply, add a response header, serve
  the body at another status or wrapped as a page, wrap it as a stream
  event, or add a phrase from another table, each edit stated in the
  test's comment. A read route recorded `skipped` in Phase 1 uses its 0062
  fixture with the tool moved to index 1.)*
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
  * ~~Phase 1: a route that refused names no tool; a refusal's form differs
    from its 0062 fixture's (status, code, marker or where the name sits);
    a route that accepted a root string in 0062 refuses now; Grok's second
    model orders tag and name differently; a written record holds a value
    the redaction proof says it must not;~~
    *(Struck 2026-11-20 by the owner's ruling on N1, relayed by the
    coordinator: "No stop: apply the matchers as written (Recommended)".
    Instead, one by one: a refusal naming no tool is the kind with `Tool`
    empty when its route's marker is there, else `ErrInvalidRequest`, and
    its record joins the test that outcome names; a form unlike its 0062
    fixture is read by the matchers as written, recognised or not, and an
    unrecognised one stays `ErrInvalidRequest` and joins the fall-back
    tests; a route that accepted a root string in 0062 and refuses now is
    read the same way; Grok's second model in another order is the kind
    with `Tool` empty, its record in `Classify` and `Names`; ~~a record
    holding a value the redaction should have removed is the MADR's ST76,
    open as N9.~~ *(Struck 2026-11-20, fourth pass: N9 is answered; a hit
    is handled as the MADR's ST76 and ST84 say.)* New forms wait for a
    later amendment of the MADR.)*
  * an existing test, golden or fixture needs a change;
  * ~~`make api-check` lists anything but the two names;~~ *(Struck
    2026-11-20, the review's finding on ST36: `make api-check` runs apidiff
    with `-incompatible` (`scripts/check_api.py:85`) and so lists no
    addition, and against `v1.31.0` the listing also holds 0080's and the
    other records' additions. Instead: `make api-check` exits 0, and
    ~~`python3 -B tools/apidiff.py <the phase's start commit>` against the
    tree~~ lists exactly `ErrToolSchemaRefused` and `APIError.Tool`, both
    compatible; any other line is a stop.)* *(Corrected 2026-11-20, fourth
    pass: the tree may hold another record's uncommitted Go work, since
    Phase 2 has no clean-tree prerequisite. The comparison is two commits
    on a scratch clone. The clone is of the base, Phase 2's changed files
    are copied in and committed there, and `python3 -B tools/apidiff.py
    <base> <that scratch commit>` runs in the clone.)*
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
~~No agent runs the live test and no agent holds a key.~~ *(Struck
2026-11-20, fourth pass: by the owner's ruling "API keys from the
environment only (Recommended)", an agent runs it; the keys stay in its
process environment and no one reads them, Phase 1b's rules.)*
*(Annotated 2026-11-20: 57 cases with every key set, since the owner's
ruling on N3 adds the `zen-messages` route's six.)*

#### Phase 1a: the recorder (an agent)

Model: mainstream, medium (a recorder in the 0062 form, P13, P14, P24, with
a redaction proof).

*(Added 2026-11-20, the readiness pass. First step: re-ground, at this
phase's start commit, every line it cites (P13, P14, P18, P19, P24, P28,
P29) and record each in the execution record; a line whose meaning changed
is a stop. Exit criterion: no blocker, every row tested, plants caught;
the rows are the MADR's ST46 and ST47, with the tests and plants of this
PLAN's amendment of 2026-11-20.)*

Files: `llmprovider/toolschema_0084_record_test.go` (untagged),
`llmprovider/toolschema_0084_record_offline_test.go` (untagged),
`llmprovider/live_0084_toolschema_measure_test.go` (`//go:build
live_gateways`). *(Annotated 2026-11-20, the last edit: and
`llmprovider/toolschema_0084_scan_test.go` (untagged), the key-shape
scan.)*

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

   *(Added 2026-11-20 by the owner's ruling on N3, relayed by the
   coordinator: "By service and marker; record Zen Messages too
   (Recommended)". A tenth route, `zen-messages`: OpenCode Zen's Messages
   route (`opencode-zen`, `claude-sonnet-5`, routed to Messages by
   `routes_snapshot.json`), running `root-string` and O1 to O5 like the
   others, read under the `opencode` label (MADR ST66).)*
2. **The record**, one JSON file per case, written to the directory
   `LLMPROVIDER_LIVE_0084_OUT` names, which must be outside the module (the
   0062 check, P13). It holds: the case id, route, model, the tools' names
   and schemas sent, the input text, the status, the `*APIError`'s kind
   name, `Code` and `Message` (already redacted by the SDK, MADR C12), the
   error body, the outcome (`refused`, `accepted` or `other`), and for an
   accepted case its token counts and number of tool calls. It never holds
   a request header, the key, the model's text or a response header.

   *(Added 2026-11-20 by the owner's rulings on N8, relayed by the
   coordinator: "Mark it unreachable or skipped; earlier fixtures stand
   (Recommended)", and on the review's a1: "As accepted: the schema was not
   refused (Recommended)". The outcome, from what `Generate` returns:*

   | What `Generate` returns | Outcome |
   | :--- | :--- |
   | no error | `accepted` |
   | an `*APIError` with `Status` 0 of kind `ErrIncomplete` or beneath it (`ErrEmptyAnswer`): a 200 the SDK found cut short or empty | `accepted` (a1) |
   | an `*APIError` with `Status` 400 to 499, other than 401, 403, 404, 408, 409 and 429 | `refused` |
   | an `*APIError` with `Status` 404, 401, 403 or 429 | `unreachable` |
   | an error that is not an `*APIError`: a transport failure, the case's 90-second deadline, a refusal before the network | `unreachable`, with the fixed reason `transport`, never the error's text |
   | an `*APIError` with `Status` 408, 409 or 500 to 599, or `Status` 0 of any kind not beneath `ErrIncomplete` (a failed 200; never read, by the ruling on N4) | `other`: ~~the owner runs the case once more; Phase 1c records any case still `other` as `unreachable`~~ *(struck 2026-11-20, fourth pass: the recorder sends it once more itself and writes `unreachable` if it is still `other`; this table is replaced by the corrected one below)* |
   | the route's key variable unset | `skipped` |

   *A route with no case `refused` or `accepted` is `skipped` as a whole,
   and its 0062 fixtures stand, the tool moved to index 1 by a named edit
   (the ruling on N7). The writer never replaces a `refused` or `accepted`
   record with another outcome, and writes `skipped` only where the case
   has no record (the review's a8), so a full rerun keeps every good
   record; a one-case rerun replaces that case's `other`, `unreachable` or
   `skipped` record. `TestToolSchemaRecord_Outcome_0084` pins both.)*

   *(Corrected 2026-11-20, fourth pass, the delta check's finding on a2.
   The table above goes by status alone, so a key refused with HTTP 400
   (`ErrAuthFailure`, `api_error.go:554` to `:561`) or a 402
   (`ErrQuotaExhausted`, `:546`) would be recorded `refused`, and a 3xx had
   no outcome. It is replaced by this one, under the owner's ruling "Mark it
   unreachable or skipped; earlier fixtures stand (Recommended)" (MADR
   ST67), taking the first row that applies:*

   | What `Generate` returns | Outcome |
   | :--- | :--- |
   | the route's key variable unset, empty or all spaces | `skipped`, "key unset or blank" |
   | no error | `accepted` |
   | an `*APIError` with `Status` 0 of a kind beneath `ErrIncomplete` | `accepted` (a1) |
   | an `*APIError` with `Status` 400 to 499, other than 401, 403, 404, 408, 409 and 429, that `errors.Is` `ErrInvalidRequest` | `refused` |
   | an `*APIError` with `Status` 408, 409 or 500 to 599, or `Status` 0 of a kind not beneath `ErrIncomplete` | `other`; the recorder sends it once more after the pass and writes `unreachable` if it is still `other` |
   | any other `*APIError`: 300 to 399, 401, 403, 404, 429, or a 4xx of another kind (a refused key, a 402's exhausted quota, `ErrNotPermitted`, `x-should-retry: true`) | `unreachable` |
   | an error that is not an `*APIError` | `unreachable`, reason `transport`, never the error's text |

   *A `refused` record is therefore always of a kind beneath
   `ErrInvalidRequest` at the base. `TestToolSchemaRecord_Outcome_0084`
   asserts each row, the 402, the 3xx and Gemini's and xAI's 400 key
   refusals included.)*
3. **What is redacted in the body**, before it is cut to 4 KiB as 0062's
   is (P14): `internal/redact.String` (keys and tokens, bearer and basic
   credentials, key-value secrets, signed URLs); then, by the repository's
   fixture rules (P28): every `request_id` and `req_...` value to
   `req_REDACTED`, an account, user or organization id to its prefix and
   `REDACTED`, a UUID that is not a public object's to the nil UUID, an
   e-mail address to `user@example.com`. 0062 redacted request ids by hand
   after its run (P29); here the writer does it.
   *(Annotated 2026-11-20, the review's a3: the fixture rules apply to
   every text field of the record, `Message` and `Code` included, not the
   body alone, since `scripts/check_fixtures.py` reads every field and
   OpenAI's messages can name an organization.)*
4. **The redaction proof**, `TestToolSchemaRecord_Redacts_0084`, offline:
   it writes a record from a recorded refusal body (F4, Claude's) into
   which it plants a made-up `xai-` key of 48 characters, a made-up
   `sk-proj-` key, a `Bearer` value of 40 characters, `"api_key": "..."`, a
   made-up `req_` id, a UUID and an e-mail address, and asserts the written
   file holds none of the planted values and holds `req_REDACTED`. Seen red
   by a plant that skips the redaction (`tools/plant.py`), quoted; green
   on the tree.
   *(Annotated 2026-11-20, the review's a3 and finding on ST46: the proof
   also plants a made-up `org-` id, a UUID and an e-mail address in the
   record's `Message` and `Code`, as a named edit of F4, and asserts no
   field of the written file holds a planted value. A second plant
   redacts the body only; it must turn the proof red.)*
5. **The live test**, `TestLive_ToolSchema_0084`: skips without
   `LLMPROVIDER_LIVE_0084`; fails without `LLMPROVIDER_LIVE_0084_OUT` or
   with one inside the module; for each case, sends when its route's key
   variable is set (`ProviderEnvVars()`, P18, P19) and records `skipped`
   when it is not; builds each provider with `providers.New`,
   `WithModelProbes(false)` and `WithoutModelMetadata()`; one `Generate`
   per case, no `WithRetry`. It logs, per case, only `"%s: %s, status %d"`
   (the id, the outcome, the status).
   *(Annotated 2026-11-20, the review's a9: 0062's recorder passes an
   error to `t.Fatal` (`llmprovider/live_0062_refusals_measure_test.go:37`,
   `:48`, `:52`); this one does not. Every failure, skip and log call takes
   a fixed text and at most the case id, such as `t.Fatalf("%s: record not
   written", c.id)`; a transport failure writes the record's fixed reason
   `transport`. ~~Phase 1a's acceptance adds the check: `grep -nE
   't\.(Fatal|Error|Skip|Log)f?\((.*[^A-Za-z_])?err[^A-Za-z_]'` on the
   three files exits 1 (no hit). On 2026-11-20 it exited 0 on
   `live_0062_refusals_measure_test.go`, naming its seven `t.Fatal(err)`
   lines, and 1 on `live_0081_grant_test.go`.~~)* *(Struck 2026-11-20,
   fourth pass: the grep needs a `t.` call and a variable named `err`, and
   so misses 0062's `genErr` (`refusals_0062_record_test.go:415`), `fmt`
   and `log` printing, and the helpers that print an error themselves. It
   is replaced by `TestToolSchemaRecord_PrintCalls_0084`, below.)*
   *(Added later on 2026-11-20 by the owner's ruling "API keys from the
   environment only (Recommended)", relayed by the coordinator, and the
   MADR's ST67, ST70 and ST77 to ST80:*
   * *the recorder reads each route's key with `os.Getenv` of its
     `ProviderEnvVars()` name and hands it to `llmprovider.WithAPIKey`; a
     key unset, empty or all spaces makes the route's cases `skipped`,
     reason "key unset or blank", and the run goes on. The recorder's files
     use no `os.Args`, `flag.` or `os.ReadFile`: `grep -nE
     'os\.Args|flag\.|os\.ReadFile'` on them exits 1;*
   * *each case's one line, on standard output and in the test log, is
     `"<case> <route> <model>: status <n>, <outcome>"` and nothing else; no
     body, `Message`, `Code` or error text is printed or logged, before or
     after redaction;*
   * *after the pass over every case, the recorder sends each `other` case
     once more and writes `unreachable` for one still `other`, so one
     command records everything; the one-case command of Phase 1b step 4
     is then needed only to repeat a case on purpose;*
   * *a body and an error's text are redacted in memory before anything is
     written; the record is the only file the recorder writes.)*
   *(Added 2026-11-20, fourth pass, the delta check's findings on ST71,
   ST79, ST83 and ST85:*
   * *The recorder's files do not call or name `SkipIfTransient`,
     `skipIfTransient`, `LiveModel` or `liveModel`: the first prints an
     error with `%v` (`llmprovider/live_gateways_test.go:51`), the second
     prints a metadata error (`live_model_picker_test.go:54`). Each model is
     the case table's literal.*
   * *~~`TestToolSchemaRecord_PrintCalls_0084`~~ (superseded 2026-11-20,
     sixth records pass, by `TestToolSchemaRecord_CallAllowlist_0084`
     below) (offline) parses the three
     files with `go/parser`. It fails on any reference to those four
     names, and on any call that prints whose arguments hold anything but
     string literals and the allowlisted fields (the case's id, route and
     model, the record's status and outcome) or a constant of fixed text.
     Calls that print are every `t.Log`, `Logf`, `Error`, `Errorf`, `Fatal`,
     `Fatalf`, `Skip` and `Skipf`, every `fmt.Print*` and `fmt.Fprint*`,
     and every `log.` call. Its plants, on a scratch copy of the files:
     `t.Fatal(genErr)`, `fmt.Println(err)` and
     `llmprovider.SkipIfTransient(t, err)`, each of which must turn it red.*
   * *(Added 2026-11-20, fifth records pass. A parse of the three files
     cannot see a print inside a helper declared elsewhere: 0033's
     `getListing` passes an error to `t.Fatal` and `t.Fatalf`
     (`llmprovider/live_0033_measure_test.go:62`, `:69`). So
     `TestToolSchemaRecord_PrintCalls_0084` also fails on any call from the
     recorder's files to a function declared in a `_test.go` file of the
     module outside those three, found by parsing every such file, except
     this allowlist:*
     * *`llmprovider.LiveCtx` (`liveCtx`, `live_gateways_test.go:96` to
       `:99`): it calls `t.Helper` and returns `context.WithTimeout`; it
       prints nothing;*
     * *`outsideModule` (`refusals_0062_record_test.go:465`): it prints
       nothing. It returns an error naming the output directory, an
       environment value, so the recorder must not print that error, as
       the check's argument rule already requires.*

     *`wiretest.Encode` and `internal/redact`'s functions are declared in
     files that are not `_test.go` files, so this rule does not reach
     them. Neither prints: `Encode` writes to a buffer, and
     `internal/redact` has no print call. Two plants are added, each on a
     scratch copy and each to turn the test red: a call to `getListing`,
     and a call to 0062's `writeRefusalRecord`, a test helper not on the
     allowlist.)* *(Superseded 2026-11-20, sixth records pass: the
     list of allowed helpers from other files is withdrawn. The recorder
     declares its own deadline and its own outside-the-module check, and
     `TestToolSchemaRecord_CallAllowlist_0084` below allows no call into
     another test file.)*
   * *(Added 2026-11-20, sixth records pass, by the owner's ruling, relayed
     by the coordinator: "Switch to an allowlist: only named calls are
     allowed (Recommended)". This replaces the check above.*

     *The files checked are the code that runs during the recording. That
     is the untagged record file and the live file, and the scan, moved
     into a file of its own, `llmprovider/toolschema_0084_scan_test.go`
     (untagged), since it too runs with the keys in the environment. The
     offline proofs stay in `toolschema_0084_record_offline_test.go`, which
     never runs in the recording and is not checked. All four files are in
     package `llmprovider_test`.*

     *How callees are resolved: by type, with `go/types` and `go/importer`
     in source mode (`importer.ForCompiler(fset, "source", nil)`). Both are
     in the standard library, so no new module is needed. The module
     requires only `golang.org/x/term` and `authcontract`, and
     `scripts/check_deps.py` refuses any other, test builds included, so
     `golang.org/x/tools/go/packages` is not used. The three checked files
     are type-checked alone, as one package. Any type error fails the
     check, naming the file and line; a call into another test file is one
     such error, since that file is not loaded. A `testing.T` copied to
     another name keeps its type, and is checked as a `*testing.T`.*

     *Imports allowed in the checked files, each for a named need:*
     * *`context`: the run's and each case's deadline;*
     * *`errors`: `errors.Is` and `errors.AsType` for the outcome;*
     * *`net/http`: the status constants, and a transport that keeps a
       copy of the reply's body for the record;*
     * *`io`: `io.ReadAll`, `io.LimitReader` and `io.NopCloser` only, to
       read and restore that body;*
     * *`bytes`: `bytes.NewReader`, for the restored body;*
     * *`os`: `os.Getenv`, for the key and the run's variables, which
       includes the case table's live switch; `os.MkdirAll` and
       `os.WriteFile`, in the one write function; `os.ReadFile`, in the
       scan file only;*
     * *`path/filepath`: building and checking paths, and walking the
       scan's directory;*
     * *`strings`: trimming a key for blank, and matching;*
     * *`regexp`: the fixture-rule redaction of ids, UUIDs and e-mail
       addresses;*
     * *`unicode/utf8`: cutting a body at 4 KiB on a rune boundary;*
     * *`time`: durations;*
     * *`testing`: the tests themselves;*
     * *`internal/redact`: `redact.String` for the writer and
       `redact.Redact` for the scan;*
     * *`internal/wiretest`: `wiretest.Encode`, the record's encoding;*
     * *`llmprovider`, `llmprovider/providers`,
       `llmprovider/providers/opencode`: the requests, the providers and
       the route option.*

     *`fmt`, `log`, `log/slog`, `encoding/json`, `bufio` and every
     package not named above are refused.*

     *Calls allowed. Every call expression must resolve to one of these:*
     * *a function or method declared in the checked files, or a
       function literal in them;*
     * *one of these named functions: `context.Background`,
       `context.WithTimeout`; `errors.Is`, `errors.AsType`; `io.ReadAll`,
       `io.LimitReader`, `io.NopCloser`; `bytes.NewReader`; `os.Getenv`,
       `os.MkdirAll`, `os.WriteFile` (in the write function only),
       `os.ReadFile` (in the scan file only); `filepath.Join`,
       `filepath.Dir`, `filepath.Abs`, `filepath.Rel`, `filepath.Base`,
       `filepath.WalkDir`; `strings.TrimSpace`, `strings.HasPrefix`,
       `strings.Contains`, `strings.Cut`, `strings.Split`; `regexp.MustCompile`;
       `utf8.RuneStart`; `redact.String`, `redact.Redact`; `wiretest.Encode`;
       `llmprovider.WithAPIKey`, `WithModel`, `WithModelProbes`,
       `WithoutModelMetadata`, `WithHTTPClient`, `ProviderEnvVars`;
       `providers.New`; `opencode.WithRoute`;*
     * *one of these named methods: `(*regexp.Regexp).ReplaceAllString` and
       `MatchString`; `(llmprovider.Provider).Generate`;
       `(http.RoundTripper).RoundTrip`; `(io.Closer).Close` on a reply's
       body; a `context.CancelFunc` value; and `(*testing.T).Run`,
       `Helper`, `Logf`, `Fatal`, `Fatalf` and `Skip`;*
     * *a type conversion, which is not a call; and the builtins `len`,
       `append`, `make`, `copy`, `min` and `max`.*

     *Any other callee fails, the check printing the file, the line and
     the callee's name, never an argument.*

     *What the printing calls may take: `Logf`, `Fatal`, `Fatalf` and
     `Skip` take only constant strings and the allowlisted values. In the
     record and live files those are the case's id, route and model and
     the record's status and outcome (MADR ST79). In the scan file they are
     the scanned file's base name, a line number and a constant label. A
     file is written only by the record file's one write function, into
     the output directory.*

     *Builtins: `panic`, `print`, `println` and `recover` are refused. A
     programming error in the recorder fails the test with a constant
     message. Anything a library the recorder may call prints itself is
     caught by the output scan (MADR ST84, ST85).*

     *Plants, each on a scratch copy of the checked files, each to turn the
     check red:*
     * *`os.Stderr.WriteString(err.Error())`;*
     * *`io.WriteString(os.Stdout, s)`;*
     * *`println(err)`;*
     * *`panic(err)`;*
     * *`slog.Info("x", "err", err)`;*
     * *`tb := t; tb.Log(err)`;*
     * *a call to `getListing`;*
     * *an import of `fmt`.*

     *One guard: the checked files as specified pass.*

     *Rows: the MADR's ST71 and ST79, and ST77 and ST80 as far as printing
     goes. This check takes the place of `PrintCalls` in each of their
     tests.*

     *The check is complete by construction for the recorder's own files:
     a call not named above cannot pass it. Its stated limit is that a
     library the recorder is allowed to call may itself print; the output
     scan (MADR ST84, ST85) covers that.)*
   * *(Added 2026-11-20, the last edit, by the owner's ruling, relayed by
     the coordinator: "Apply the checker's fixes exactly; the build proves
     them (Recommended)". These five fixes amend the check above:*
     * *The argument rule, enforced. For `(*testing.T).Logf`, `Fatal`,
       `Fatalf` and `Skip`, for `(*testing.T).Run`'s name argument, and for
       a call through an injected function type (below), each argument
       must have a constant value (`types.Info.Types[e].Value != nil`) or be
       a selector of a named field. The named fields are the case's `id`,
       `route` and `model` and the record's `Status` and `Outcome`. In the
       scan file they are the hit's `file` (a base name), `line` and
       `label` (a constant). Plants added: `t.Fatal(err)` and
       `t.Run(err.Error(), f)`.*
     * *The write path. The check fails if `os.WriteFile` or `os.MkdirAll`
       is called anywhere but inside the record file's one write function.
       Inside it, the path argument of `os.WriteFile`, `os.MkdirAll` and
       `os.ReadFile` must be an identifier assigned there from
       `filepath.Join(dir, …)` alone, or `filepath.Dir` of that
       identifier, where `dir` is the function's output-directory
       parameter. The check enforces that much and no more: it does not
       follow where the caller's `dir` came from. The output scan (MADR
       ST84) remains the backstop.*
     * *Passing its own check. Also allowed: `os.ReadFile` inside the one
       write function, so it can read an existing record and keep a
       `refused` or `accepted` one (ST70), finding the outcome with
       `strings.Contains` and a missing file with `errors.Is(err,
       os.ErrNotExist)`. Also allowed: calls through a function type
       declared in the record file, the injected output and log function
       that `NoLeak` replaces, under the argument rule. And `io/fs`, for
       the scan's `filepath.WalkDir` callback, its `fs.DirEntry` type and
       the named method `(fs.DirEntry).IsDir` only. A generic call such as
       `errors.AsType[*llmprovider.APIError]` resolves through
       `types.Info.Instances` to `errors.AsType`.*
     * *The source importer and cgo. The check sets
       `build.Default.CgoEnabled = false` before `importer.ForCompiler`,
       so that loading `net/http` from source needs no C toolchain. The
       offline proofs file holds no `init` function and no package-level
       call expression, and the same test checks that file for those two
       things only.*
     * *The file lists, fixed in step 6, the Files line and the scan
       bullet, as dated there.*

     *The guard, checked by reading. These are the calls the recorder as
     specified makes, each on the list:*
     * *record file: `regexp.MustCompile` at package level,
       `(*regexp.Regexp).ReplaceAllString`, `redact.String`,
       `utf8.RuneStart`, `len`, `append`; in the outcome, `errors.AsType`
       and `errors.Is`; in the transport, `(http.RoundTripper).RoundTrip`
       on `http.DefaultTransport`, `io.ReadAll`, `io.LimitReader`,
       `(io.Closer).Close`, `io.NopCloser`, `bytes.NewReader`; in the case
       path, `providers.New`, `llmprovider.WithModel`, `WithModelProbes`,
       `WithoutModelMetadata`, `WithHTTPClient`, `WithAPIKey`,
       `opencode.WithRoute`, `(llmprovider.Provider).Generate`, and the
       injected output and log function; in the write function,
       `wiretest.Encode`, `filepath.Join`, `filepath.Dir`, `os.MkdirAll`,
       `os.ReadFile`, `errors.Is` with `os.ErrNotExist`, `strings.Contains`,
       `os.WriteFile`;*
     * *live file: `os.Getenv`, `(*testing.T).Skip` and `Fatal` with
       constants, `filepath.Abs`, `filepath.Rel`, `strings.HasPrefix` (the
       recorder's own outside-the-module check),
       `llmprovider.ProviderEnvVars`, `strings.TrimSpace`,
       `context.Background`, `context.WithTimeout`, the `context.CancelFunc`
       (`defer cancel()`), `(*testing.T).Run` with the case id,
       `(*testing.T).Helper`, `Logf` with the five fields, `Fatalf` with the
       case id, and the record file's own functions;*
     * *scan file: `os.Getenv`, `(*testing.T).Skip` with a constant,
       `filepath.WalkDir`, `(fs.DirEntry).IsDir`, `os.ReadFile`,
       `strings.Split`, `redact.Redact`, `filepath.Base`,
       `(*regexp.Regexp).MatchString`, `regexp.MustCompile` at package
       level, `(*testing.T).Logf` with the hit's `file`, `line` and `label`,
       and `Fatal` with a constant; and conversions (`string(…)`,
       `[]byte(…)`), which are not calls.*

     *None is off the list. The proof is the build: before any live run,
     Phase 1a shows the check passing on the recorder (the guard) and every
     plant failing.)*
   * *The writer encodes a record with `wiretest.Encode`, as 0062's does:
     indented JSON ~~with sorted keys~~ (struck 2026-11-20, fifth records
     pass: `encoding/json` writes a struct's fields in declaration order)
     (`internal/wiretest/wiretest.go:173`
     to `:183`). Every top-level key starts its own line at two spaces'
     indent. A nested value (the tools and their schemas) spans the lines
     after its key at a deeper indent. A string (the body, `Message`,
     `Code`) stays on one line.*
   * *The key-shape scan, `TestToolSchemaScan_0084`, sits in the ~~untagged
     file~~ scan file, `toolschema_0084_scan_test.go` (pointer added
     2026-11-20, the last edit) (MADR ST85).)*
6. ~~**Before the owner runs it,**~~ **Before the run** *(annotated
   2026-11-20: an agent runs it)*, the agent reads the ~~three files~~ four
   files, the scan file included (annotated 2026-11-20, the last edit), and
   confirms in the execution record that no log or failure call is given a
   key, a variable holding one, an error's text or a body; runs `go vet
   -tags live_gateways ./llmprovider/` and `go test -count=1 -run
   'ToolSchemaRecord' ./llmprovider/` (session host, no `-race`); and
   commits the ~~three files~~ four files (annotated 2026-11-20, the last
   edit: the scan file too) with `make pre-add-check` through `run-heavy`.

Acceptance: the offline test green, its plant caught; the live test
compiles; no key in any file.

#### Phase 1b: the owner's run

*(Added 2026-11-20, the readiness pass. First step: ~~the owner confirms~~
the agent confirms (fourth pass) the
checkout is at Phase 1a's commit (step 1 below), which re-grounds the
recorder this run uses. Exit criterion: no blocker, every row tested,
plants caught; for this run that means every case recorded or `skipped`,
and each observation handled as the MADR's ST53 to ST59 and ST66
(ST66 added, fourth pass) and the owner's
answers to N1 and N8 say, with no plant to run.)*

*(Added later on 2026-11-20 by the owner's ruling, relayed by the
coordinator: "API keys from the environment only (Recommended)". This run
is made by an agent or the coordinator, not the owner. The paragraph and
the numbered steps 1 to 6 below, struck through, are not to be run; the
run is this one.*

*Rules of the run, each a row of the MADR (ST77 to ST82):*

* *A key reaches the run only through the process environment, inherited
  from the shell where it is already set; never as an argument, a file, a
  log line, output or a record.*
* *Nobody displays, reads, copies, measures or compares a key's value. A
  key's presence is checked by its name only; the recorder alone tests a
  value for blank, and a blank or unset key makes its route `skipped`,
  reason "key unset or blank", with no stop and no question.*
* *The recorder writes nothing but its redacted records, and prints to
  standard output and the test log only ~~the case, route, model, HTTP status
  and outcome~~ the lines of MADR ST79's list as corrected (2026-11-20, fifth
  records pass). A raw body or raw error text never reaches output, a log or
  a record before redaction.*
* *The agent that runs it reads only the redacted records, after Phase
  1c's checks pass on them, and the run's output file, which holds only the
  allowlisted lines of MADR ST79's list as corrected (2026-11-20, fifth
  records pass).*
* *Nothing here needs a browser or device sign-in, a saved sign-in
  session, a provider account page or a person at a console (MADR ST82).*

*The steps, on this Mac, in the checkout at Phase 1a's commit, from the
shell whose environment holds the keys. It is one package without
`-race`, so it runs on the session host as the Rules allow; the records go
to a directory outside the module, and the output to `tmp/` (git-ignored):*

1. *Confirm the commit: `git log -1 --format=%h` prints Phase 1a's.*
2. *Run the recorder, capturing the exit status before reading anything:*

   ```sh
   mkdir -p <scratch>/tool-schema-0084
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT=<scratch>/tool-schema-0084 go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$' -count=1 -timeout 180m -v > tmp/0084-run.txt 2>&1
   echo "exit=$?"
   ```

   *`-timeout 180m` covers 57 cases and a second send of each at 90
   seconds apiece, the worst case. A run cut short is run again whole; the
   writer keeps every `refused` and `accepted` record (MADR ST70).*
3. *Hand Phase 1c the records' directory and `tmp/0084-run.txt`.)*

*(Corrected 2026-11-20, fourth pass, the delta check's findings on ST79,
ST84 and ST85. Step 2's command is replaced by the one below, which adds
`-trimpath` so that a panic's stack trace, such as the one `-timeout`
prints, names module paths and not this Mac's home directory. Step 3 is
replaced by the scans below, which the agent runs before it reads the
output file or any record.*

```sh
mkdir -p <scratch>/tool-schema-0084
LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT=<scratch>/tool-schema-0084 go test -trimpath -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$' -count=1 -timeout 180m -v > tmp/0084-run.txt 2>&1
echo "exit=$?"
```

*The output file may hold only these lines (MADR ST79):*

* *`go test -v`'s framing: `=== RUN`, `=== PAUSE`, `=== CONT` and
  `=== NAME`, each with a test or case name; `--- PASS:`, `--- FAIL:` and
  `--- SKIP:`, each with a name and a duration; `PASS` and `FAIL` alone;
  and `ok` or `FAIL` with the package path and a duration;*
* *the case line `t.Logf` writes, prefixed with its file name and line:
  `<case> <route> <model>: status <n>, <outcome>`;*
* *the recorder's fixed failure and skip texts, each with at most a case
  id.*

*The scans, each exit status captured before anything is read:*

```sh
LLMPROVIDER_LIVE_0084_SCAN=tmp/0084-run.txt go test ./llmprovider -run '^TestToolSchemaScan_0084$' -count=1
echo "exit=$?"
python3 -B scripts/check_fixtures.py tmp/0084-run.txt
echo "exit=$?"
python3 -B tools/denyscan.py tmp/0084-run.txt
echo "exit=$?"
LLMPROVIDER_LIVE_0084_SCAN=<scratch>/tool-schema-0084 go test ./llmprovider -run '^TestToolSchemaScan_0084$' -count=1
echo "exit=$?"
```

*A hit in the output file, or a line outside the list above, is the MADR's
ST84: the file is deleted unread, and the run is repeated once. A hit in a
record is ST76, handled in Phase 1c. Only after every scan exits 0 does
the agent read the output file and hand Phase 1c the records' directory.)*

~~The owner runs it, never an agent, at the commit that adds the recorder, in
a new terminal window that is closed afterwards. Each `read -rs NAME` line
waits, showing nothing, for one value: the owner types or pastes it and
presses Enter. Nothing is echoed; the value lives only in that shell's
environment and the `go test` process it starts, and is never written to a
file; the lines typed hold no key, so the shell's history holds none. The
commands work as written in zsh and in bash, as 0080's did (P23). Nothing
else is typed in that window: no `echo`, `printenv`, `env`, `set` or `set
-x`. A key the owner does not have is skipped: its route is recorded
`skipped`.~~

1. ~~Open the checkout and confirm the commit:~~

   ```sh
   cd <the go-llmprovider-sdk checkout>
   git log -1 --format=%h
   ```

2. ~~The keys, each at a hidden prompt:~~

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

3. ~~A directory for the records, outside the checkout:~~

   ```sh
   mkdir -p "$HOME/tool-schema-0084"
   ```

4. ~~The recorder, every case:~~

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$' -count=1 -v
   ```

   ~~One case again, if a line says `other` (a timeout or a 5xx):~~

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$/^grok-O1-root-union$' -count=1 -v
   ```

   *(Annotated 2026-11-20, the review's a8: the two commands above carry
   no `-timeout`, and Go's default of ten minutes can end a run of 57
   cases of up to 90 seconds each (`llmprovider/live_gateways_test.go:98`)
   part-way. They are replaced by these, which differ only by the
   `-timeout` flag; the lines above are not to be run:*

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$' -count=1 -timeout 100m -v
   ```

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$/^grok-O1-root-union$' -count=1 -timeout 5m -v
   ```

   *The one-case command is run, with the case's id in place of
   `grok-O1-root-union`, for each line that says `other`, and may be run
   for a line that says `unreachable`. A rerun, of one case or of all,
   never replaces a `refused` or `accepted` record (step 2 of Phase 1a, as
   amended), so it cannot lose a good record. A run that ends part-way is
   run again whole.)* *(Superseded later on 2026-11-20 by the run above:
   one command with `-timeout 180m`, the recorder sending each `other`
   case a second time itself.)*

5. ~~Remove the values from the shell, then close the window:~~

   ```sh
   unset XAI_API_KEY OPENAI_API_KEY ANTHROPIC_API_KEY GEMINI_API_KEY OPENCODE_API_KEY
   ```

6. ~~Tell the coordinator the run is done; the records stay in
   `$HOME/tool-schema-0084` for Phase 1c.~~

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

*(Annotated 2026-11-20 by the owner's ruling on N1: the third column's
"a different form is a stop" no longer holds. No observation changes SDK
code in this PLAN; each record feeds the test the MADR's ST53 to ST59 and
ST66 name.)*

Stop conditions (each a stop and a prompt to the owner before Phase 1c):

* ~~a route that refuses names no tool, or names it where its matcher does not
  read;~~ *(Struck 2026-11-20, N1: the kind with `Tool` empty when the
  route's marker is there; `ErrInvalidRequest` when it is not; the record
  joins `Classify` and `Names` (an empty `Tool` asserted) or the fall-back
  tests.)*
* ~~a refusal's form differs from the 0062 fixture's for its route: another
  status, code or marker;~~ *(Struck 2026-11-20, N1: read by the matchers
  as written; an unrecognised form stays `ErrInvalidRequest` and joins
  `TestToolSchemaRefusal_RecordedFallback_0084`; a later amendment may add
  it.)*
* ~~a route that accepted a root string in 0062 refuses now, or O4 is refused
  in a form unlike its route's O1;~~ *(Struck 2026-11-20, N1 and N3: read
  by its service's matchers as written; recognised, it joins `Classify`
  and `Names`; unrecognised, the fall-back tests.)*
* ~~O6 shows a different order;~~ *(Struck 2026-11-20, N1: the kind with
  `Tool` empty, the record in `Classify` and `Names`.)*
* ~~Phase 1c's checks find a value the redaction should have removed.~~
  *(Annotated 2026-11-20: the one stop a live observation can still
  trigger; the MADR's ST76 puts it to the owner as N9.)* *(Struck later on
  2026-11-20 by the owner's ruling on N9, relayed by the coordinator:
  "Treat that case as unrecorded, fix the recorder, rerun (Recommended)".
  What happens instead is in Phase 1c's note of the third amendment. A
  second leak of the same case after one fix and rerun is a deviation for
  the owner.)*

#### Phase 1c: the fixtures (an agent)

Model: mainstream, low.

*(Added 2026-11-20, the readiness pass. First step: re-ground, at this
phase's start commit, the fixture rule (P28) and `make fixtures-check`'s
baseline, and record them. Exit criterion: no blocker, every row tested,
plants caught; here every record copied, each sorted into the test it
feeds by the MADR's ST53 to ST59 and ST66 (ST66 added, fourth pass), and
`make fixtures-check` and
`tools/denyscan.py --changed` each exit 0.)*

1. Copy the records from `$HOME/tool-schema-0084` to
   `llmprovider/internal/wire/testdata/toolschema-0084/<wire>/`, unchanged.
2. `make fixtures-check` and `python3 -B tools/denyscan.py --changed`, each
   exit 0; ~~a hit is a stop,~~ never a baseline entry. *(Struck
   2026-11-20, fourth pass: a hit is handled as the MADR's ST76 says.)*
   *(Annotated 2026-11-20 by the owner's ruling on who runs the recording:
   the records come from the run's directory, `<scratch>/tool-schema-0084`,
   not `$HOME/tool-schema-0084`. Steps 1 and 2 run before the agent reads
   any record: the copy is made unread, and the two checks, which print a
   fingerprint and never a value, run on it first (MADR ST81). ~~A hit is the
   MADR's ST76, open as N9.~~ *(Struck 2026-11-20, fourth pass: N9 is
   answered; see ST76.)* Every case still `other` is copied as
   `unreachable`.)*
   *(Added later on 2026-11-20, the third amendment, by the owner's ruling
   on N9, relayed by the coordinator: "Treat that case as unrecorded, fix
   the recorder, rerun (Recommended)". This phase's rules for a hit (MADR
   ST76, ST83):*
   * *Nobody reads the leaked value. A hit is reported as the file, the
     field and the rule that matched, never the matched text.
     `scripts/check_fixtures.py` prints the path, line, column, kind and a
     fingerprint; `tools/denyscan.py` prints the file, line and kind. ~~Each
     record holds one top-level field per line, so the agent names the
     field by printing only the key of the reported line.~~ (Corrected
     2026-11-20, fourth pass: nested tools and schemas span several lines
     (`wiretest.Encode`). The agent names the field as the nearest line at
     or above the reported one that starts a top-level key, at exactly two
     spaces' indent, and prints only that key. `TestToolSchemaScan_0084`
     prints the file, the line and "key shape", never the text.)*
   * *Nothing of that case is committed, and it is `unreachable` for that
     run. Its file is removed, unread, from the run's directory, so the
     rerun writes it again (the drafter's reading under N9, in the MADR's
     third amendment).*
   * *In a further recorder commit, the recorder's redaction is extended
     to the missed value's shape. `TestToolSchemaRecord_NoLeak_0084` gains
     that shape and is seen red before the fix. An agent then reruns the
     recording with Phase 1b's command.*
   * *If the same case leaks again after one fix and one rerun, that is a
     deviation: stop, and put it to the owner. A third pass is never
     routine.)*
   *(Added 2026-11-20, fourth pass. Steps 1 and 2 copy before they check,
   so a leaked record can already sit in `testdata/toolschema-0084/`. On a
   hit, the agent removes, unread, each copied file a check reported,
   using the path the check printed. The copies are untracked and never
   staged. The key-shape scan, `TestToolSchemaScan_0084`, runs on the copies
   beside the two checks (MADR ST85). The checks are run again until they
   exit 0 on what remains.)*
3. The execution record gets one line per case (id, outcome, status, code,
   whether and where the tool is named), and the MADR a dated amendment of
   its provider table: each "not established" cell Phase 1 settles, with
   its fixture.
4. Commit the fixtures and the two records' edits.

Acceptance: every case recorded or `skipped`; no stop condition met, or
the owner's resolution recorded as a dated deviation.
*(Annotated 2026-11-20 by the owner's rulings on N1 and N8: a case still
`other` is copied as `unreachable`; a route with no `refused` or
`accepted` case is `skipped`, and its 0062 fixtures stand. ~~The one stop
left is a fixture check's hit (MADR ST76, N9).~~)* *(Struck 2026-11-20,
fourth pass: a hit is handled as ST76 and ST84 say; the one stop left is
the same case leaking again after one fix and rerun, a deviation for the
owner.)*

### Phase 2: the tests, the code and the docs, one commit

Model: mainstream, high (two names under v1, four send paths, and an index
that must be mapped through the list as sent after deferral).

*(Added 2026-11-20, the readiness pass. First step: step 1 below, the
re-read of the base, now also covering the lines the MADR's amendment of
2026-11-20 reads as H1 to H8. Exit criterion: no blocker, every row
tested, plants caught; the rows are the MADR's state table, and the tests
and plants are this PLAN's amendment of 2026-11-20, as the owner's answers
to N2 to N7 shape them.)*

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

   *(Annotated 2026-11-20, the owner's rulings relayed by the coordinator
   and the review; each row above stands except as said here:*
   * *`Classify`: "each recorded refusal ... Phase 1's" means each record
     a D3 matcher recognises (MADR ST53 to ST59, ST66); the others feed
     `TestToolSchemaRefusal_RecordedFallback_0084`. It also asserts that
     `errors.As` finds the `*APIError` and that its `Kind` is
     `ErrToolSchemaRefused`.*
   * *`Names`: where a read route is `skipped`, its 0062 fixture with the
     tool moved to index 1 by a named edit (N7). On OpenAI's routes the
     name comes from the message's `function '<name>'` only (N2). It also
     asserts `errors.As` and `Kind`, through `Generate` and `Stream`. Its
     routes include OpenCode Zen's Messages route when Phase 1 recorded it
     there in Anthropic's form (MADR ST66).*
   * *`NeverNamesUnsent`: ~~OpenAI's native `tool_search` entry~~ is struck
     from its list by the owner's ruling on ST10, relayed by the
     coordinator: "Name the caller's Search tool (Recommended)"; that case
     moves to `TestToolSchemaRefusal_SearchLoaded_0084`. It gains a request
     that sent no tools, and OpenAI's F2 with `param` naming index 1 while
     the message names an unsent tool: `Tool` empty (N2).*
   * *`Deferral`: Claude's index only, since OpenAI's routes no longer
     read an index (N2); on a Claude model in the Append form (before 4.5,
     `deferral.Of`, `llmprovider/internal/wire/deferral/deferral.go:95`), with
     a deferred tool not loaded ahead of the refused one, so the sent list
     and `Request.Tools` differ. On the native form they are the same list
     (`deferral.go:258`) and the plant would change nothing.*
   * *`SeveralNamed`: built in any case; from O2's record when a route
     names two tools, otherwise from F1 and F4 given a second name or index
     by a named edit (N7).*
   * *the tests the amendment of 2026-11-20 adds, with their plants.)*

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

   *(Annotated 2026-11-20 by the owner's rulings, relayed by the
   coordinator:*
   * *~~the unexported `param` read~~: not built (N2, "From the message's
     `function '<name>'` only (Recommended)"); OpenAI's routes name the
     tool from `Message` alone.*
   * *the D3 rows apply in `ClassifyHTTPError`'s call only, never in
     `setStreamKind`'s (N4); they sit after the overflow row, and the
     `x-should-retry` rule keeps its place after the table (N5, "Existing
     rules win (Recommended)"); each row keys on the service and the
     provider's marker, never on a route (N3).*
   * *the helper finds the `*APIError` through wrapping with `errors.As`
     and sets `Tool` on it, returning the error it was given (N6, "Name by
     string; find the error through wrapping (Recommended)"). It sets
     `Tool` when the names and indices the refusal gives resolve to
     exactly one name among the caller's tools the body held, duplicates of
     one name included. Those tools are the `tools` list as encoded plus
     the definitions inside each `tool_search_output` item of `input`
     (`InputLoading`, `llmprovider/internal/wire/responses/responses.go:183`),
     and the caller's Search tool whose native `tool_search` entry carries
     no name (the ruling on ST10); never the SDK's `use_tool` wrapper.*
   * *the helper also runs on OpenCode's Messages route
     (`opencode.go:334` and its stream), since N3 reads every route of a
     service.)*

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

   *(Annotated 2026-11-20 by the owner's rulings on N3 to N5: the
   architecture, the package docs and the changelog entry say that the kind
   is matched by service and marker on every route of Grok, OpenAI
   (API key, ChatGPT session, plan session), Anthropic and OpenCode, and
   name the routes each form was recorded on; that a failure inside a 200
   body or a stream is not classified; and that `x-should-retry: true` and
   a context overflow win over it. The changelog's quoted entry is
   reworded to match. The content is pinned by the MADR's ST73 and ST74.)*
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
*(Annotated 2026-11-20, the review's finding on ST36: `make api-check`
lists no addition, so the check is `make api-check` exiting 0 and
~~`python3 -B tools/apidiff.py <the phase's start commit>` against the tree~~
`python3 -B tools/apidiff.py <base> <scratch commit>` on a scratch clone,
as the Rules' note of the fourth pass says (corrected 2026-11-20),
listing exactly the two names, both compatible. `tools/apidiff.py v1.31.0`
also lists 0080's `signin.VerifyKilo` and whatever 0081, 0083 and 0085 add,
so it is a release note, not this phase's check.)*

Acceptance: every row as its table says; no existing test changed; the docs
say what the code does, the fall-back included.

### Phase 3: planted breaks and the gates

Model: mainstream, low.

*(Added 2026-11-20, the readiness pass. First step: re-ground, at this
phase's start commit, the lines each plant replaces, so every plant's old
text occurs once. Exit criterion: no blocker, every row tested, plants
caught; the plants are step 1's and those the amendment of 2026-11-20
adds.)*

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

*(Added 2026-11-20, the readiness pass. First step: re-ground the diff's
cited lines at this phase's start commit. Exit criterion: no blocker,
every row tested, plants caught; the reviewer checks each decided row of
the MADR's state table against a test and its plant.)*

Model: strongest, high. A fresh agent that did not write Phases 1 to 3,
briefed with the MADR, this PLAN and the diff from the base, not with the
implementer's conclusions.

1. It reads the diff whole and checks, with a quoted line for each finding:
   that `Tool` can only hold a sent caller tool's name, on every path,
   streams included; that no response text reaches `Tool` or a log; that a
   reworded refusal and every other error keep their kind; that the docs
   and the changelog say what the code does; that `make api-check`'s two
   names are all that is exported; that the recorder can print no key.
   *(Annotated 2026-11-20: "`make api-check`'s two names" reads as
   `tools/apidiff.py` from Phase 2's start commit; and the reviewer checks
   the docs and the changelog against the MADR's ST73 and ST74.)*
2. Each finding is fixed in a commit within this phase, its test seen red
   first, or put to the owner with its evidence. None is waved through.

Acceptance: the review's findings, and what was done with each, in the
execution record.

### Phase 5: the release (the owner's word)

*(Added 2026-11-20, the readiness pass. First step: re-ground the
changelog entry and the docs this record changed, at this phase's start
commit. Exit criterion: no blocker, every row tested, plants caught, as
recorded by Phases 2 to 4.)*

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
* ~~the owner's run~~ the recording run (Phase 1b), then `make fixtures-check` and `python3 -B
  tools/denyscan.py --changed` *(annotated 2026-11-20, fourth pass: an
  agent's run, by the owner's ruling "API keys from the environment only
  (Recommended)"; with Phase 1b's scans and `TestToolSchemaScan_0084`)*
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and at
  the stub, then `--expect pass`; `python3 -B tools/plant.py PLANTS.json`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=3 -timeout 45m ./llmprovider/... ./wizard/...`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`: the two names
  *(annotated 2026-11-20: `make api-check` exits 0, and `tools/apidiff.py`
  from Phase 2's start commit to its tree lists the two names, as Phase 2's
  Verification now says)*
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
  commit; 1b ~~when the owner chooses~~ after 1a's commit, run by an agent
  (annotated 2026-11-20, fourth pass); 1c after it; Phase 2 on 1c's commit,
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

### Approval (2026-11-17)

* **Approved 2026-11-17.** This PLAN was approved through its live step or
  release by the owner's ruling of 2026-11-17, relayed by the coordinator:
  "Accept all five and run each plan up to its live step or release
  (Recommended)". Its phases are committed and pushed by the coordinator as
  each passes. Tags and releases need the owner's word.

### Fifth records pass (2026-11-20)

* **Approved 2026-11-20** for one item only, by the owner's ruling,
  relayed by the coordinator: "One more records pass on this item only
  (Recommended)". This is the fifth records pass of the day. The owner's
  word was needed because the fourth pass did not pass its re-check. The
  item is `TestToolSchemaRecord_PrintCalls_0084`'s reach into helpers
  declared in other test files, with two one-line corrections (Phase 1b's
  run rules now point to ST79's list, and "sorted keys" is struck).

### Sixth records pass (2026-11-20)

* **Approved 2026-11-20** for the print check only, by the owner's
  ruling, relayed by the coordinator: "Switch to an allowlist: only named
  calls are allowed (Recommended)". The fifth pass's list of forbidden
  calls failed its re-check again. The checker found further ways to
  print: writes to `os.Stdout` and `os.Stderr`,
  `io.WriteString(os.Stdout, …)`, the builtins `print` and `println`,
  `panic(err)` writing into the run's output, any `slog` call, and a
  `testing.T` copied to another name. A list of forbidden calls is
  therefore not complete by construction. The check is now
  `TestToolSchemaRecord_CallAllowlist_0084`, in Phase 1a.

### The last edit (2026-11-20)

* **Approved 2026-11-20** by the owner's ruling, relayed by the
  coordinator: "Apply the checker's fixes exactly; the build proves them
  (Recommended)". The sixth pass's re-check found the allowlist design
  sound and all eight plants red. It also found five gaps in the
  specification, each with a named fix: the argument rule enforced, the
  write path, the recorder passing its own check, the source importer and
  cgo, and the file lists. The fixes are applied exactly, in Phase 1a,
  with no further review round. Before any live run, the recorder's build
  must show the check passing on the recorder (the guard) and every plant
  failing; that is the proof.

## Amendment of 2026-11-20: the test map, proposed tests, exit criteria and re-grounding

A readiness pass before Phase 1a, on the coordinator's brief, documents
only, read at `24756a6`. It pairs with the MADR's amendment of 2026-11-20,
which carries the state table (rows ST01 to ST65) and the open questions
N1 to N8. Nothing above is rewritten; each unbuilt phase gained a dated
note giving its first step (re-grounding) and its exit criterion.

### Where the phases stand

* Phase 0 is done (`a3002aa` placed the pair, `8f133db` accepted it).
  Phases 1a to 5 have not started: no `llmprovider/*0084*` file and no
  `testdata/toolschema-0084/` exist at `24756a6`.
* Phase 1a's prerequisite, 0080-PLAN Phase 3's code, is on `main`
  (`00757ed`). Its other prerequisite, a tree no other agent holds work
  in, waits for the four readiness passes' record edits to be committed.
* O1 to O6 are case shapes inside one ~~owner run~~ recording run (Phase
  1b; annotated 2026-11-20, fourth pass: an agent's), not steps.
* By the owner's rule that open cases are answered in one prompt before
  code, N1 to N8 are answered before Phase 1a starts. N1, N7 and N8 govern
  Phases 1a to 1c; N2 to N7 govern Phase 2.

### The test map

Each decided row of the MADR's state table, its test, the phase, and the
plant: the single production change that must turn the test red, run on a
scratch clone by `tools/plant.py`, never in the tree. "Phase 3" in the
plant column is a plant already in Phase 3 step 1.

| Row | Test | Phase | Plant |
| :--- | :--- | :--- | :--- |
| ST01, ST19 | `Classify`, `Names` | 2 | Grok's row removed (Phase 3) |
| ST02, ST04 | `Classify`, `Names` | 2 | OpenAI's row removed (Phase 3) |
| ST05 | `Classify`, `Names`, `Deferral` | 2 | Anthropic's row removed; the index mapped through `Request.Tools` (Phase 3) |
| ST06, ST18 | `Classify`, `Names` | 2 | the Google route's row removed (Phase 3) |
| ST07 | `OthersUnchanged` | 2 | a row loosened to any 400 of its code or type (Phase 3) |
| ST09 | `Names` | 2 | as ST01 |
| ST10 | `NeverNamesUnsent`, with a subcase added: a request that sent no tools | 2 | the name taken from the reply without the sent list (Phase 3) |
| ST11 | `NamesNothing` (added below) | 2 | the helper, finding no sent name, falls back to the last caller tool sent |
| ST12 | `SeveralNamed` (built if O2 records two names; else N7) | 2 | the helper takes the first of several names (added) |
| ST14 | `Reworded` | 2 | a row loosened (Phase 3) |
| ST15, ST16 | `Classify`, `Names` (its `Stream` half) | 2 | the helper called on `Generate`'s path only (added) |
| ST20, ST23, ST50 | `ProxyAltered` (added below; needs N7 (a)) | 2 | a row's status condition dropped |
| ST21 | `OthersUnchanged`; `KeyRowFirst` (added below) | 2 | Grok's D3 row moved above the key row |
| ST22 | `OthersUnchanged` | 2 | Grok's row loosened to any `invalid-argument` 400 (Phase 3) |
| ST24 | `Reworded`, its OpenAI case's code changed to `invalid_prompt` (added detail) | 2 | OpenAI's row keyed on the type `invalid_request_error` instead of its code |
| ST25, ST27 | `OthersUnchanged` (Claude's empty-input 400) and the existing overflow, 0064 and 0043 tests, unchanged | 2 | Anthropic's row keyed on its type alone |
| ST28, ST44 | `FieldsUnchanged` (added below) | 2 | the classification appends the tool's name to `Message` |
| ST29 | `Classify` | 2 | the kind a plain sentinel (Phase 3) |
| ST30 | `Classify`, with `errors.As` and `Kind` asserted (added) | 2 | the provider returns the refusal as `fmt.Errorf("%w: %w", ErrToolSchemaRefused, err)` with `Kind` left `ErrInvalidRequest` |
| ST31 | `Classify` | 2 | `Error()` prints `Tool` (Phase 3) |
| ST34 | `PassThrough` (added below) | 2 | the helper sets `Tool` on an `*APIError` of any kind |
| ST35 | `TestDescribe_ToolSchemaRefused_0084` | 2 | the kind missing from `errorClasses` (Phase 3) |
| ST36 | `make api-check`, `tools/apidiff.py` | 2 | a third exported name added: listed |
| ST37 | `Classify` | 2 | the D3 rows return `terminal` false |
| ST38 | `NoAdaptation` (added below) | 2 | the provider resends once without the named tool |
| ST39 | `LastSendKeepsName` | 2 | the name filled in the resend hook (Phase 3) |
| ST40 | `Concurrent` (added below), under `-race` in Phase 3 step 3 | 2 | the sent list kept in a provider field, not per request |
| ST41 | `RedactedNeighbour` | 2 | Grok's parse requires the word `tool` after the colon (added) |
| ST42 | `RedactedName` (added below) | 2 | as ST11 |
| ST43 | `NameBeyondBound` (added below) | 2 | the helper matches a sent name by prefix |
| ST45 | `NoLog` (added below) | 2 | the helper logs the error at `Debug` through the provider's logger |
| ST46 | `TestToolSchemaRecord_Redacts_0084` | 1a | the recorder's redaction skipped (Phase 3) |
| ST47 | `TestToolSchemaRecord_LogLine_0084` (added below) | 1a | the case's log line also prints the error's `Message` |
| ST48, ST49 | the existing `tools` tests and `TestValidate_UnmarshalableSchema`, unchanged | 2 | none: guards that hold at the base |
| ST56 | none: no SDK behaviour | – | – |
| ST63 to ST65 | Phase 2 step 1's re-read; `make changelog-check docs-check` | 2 | – |

### Decided rows that had no test: the tests added (dated addition, 2026-11-20)

Each joins Phase 2 step 3's table (or Phase 1a, as marked), in a
`_0084_test.go` file of its package, built from fixtures under the Rules,
red at the stub unless marked a guard, and its plant joins Phase 3 step 1.

| Test (package) | Row | Asserts | At the stub |
| :--- | :--- | :--- | :--- |
| `TestToolSchemaRefusal_NamesNothing_0084` (`providers`) | ST11 | F1 with its tool renamed to the empty string: the kind, `Tool` empty | red for the kind |
| `TestToolSchemaRefusal_ProxyAltered_0084` (`llmprovider`) | ST20, ST23, ST50 | F1, F2, F4 and F7 served at 401, 403, 429, 502 and as an HTML page at 400: each the kind it has at the base, never the new one. Only if N7 (a) | green (a guard) |
| `TestToolSchemaRefusal_KeyRowFirst_0084` (`llmprovider`) | ST21 | F1 with its tool renamed so the message names an API key: `ErrAuthFailure` | green (a guard) |
| `TestToolSchemaRefusal_FieldsUnchanged_0084` (`llmprovider`) | ST28, ST44 | for F1, F2, F4 and F7, `Status`, `Code`, `Message`, `RetryAfter`, `ResetAt`, `Limit` and `Reason` equal what the base classifies | green (a guard) |
| `TestNameRefusedTool_PassThrough_0084` (`internal/wire`) | ST34 | the helper returns a non-`*APIError`, `context.Canceled`, `context.DeadlineExceeded` and an `*APIError` of every other kind unchanged, `Tool` empty | red (the helper does not exist at the stub) |
| `TestToolSchemaRefusal_NoAdaptation_0084` (`providers`) | ST38 | per D3 route, through `Generate`: one request reaches the server, its `tools` as the base encodes them | green (a guard) |
| `TestToolSchemaRefusal_Concurrent_0084` (`providers`) | ST40 | two `Generate` calls at once on one provider, each with its own refused tool: each error names its own | red |
| `TestToolSchemaRefusal_RedactedName_0084` (`providers`) | ST42 | F1 with its tool renamed to a made-up key-shaped name that redaction removes (asserted absent from `Message`): the kind, `Tool` empty | red for the kind |
| `TestToolSchemaRefusal_NameBeyondBound_0084` (`providers`) | ST43 | F7 with its tool renamed to a 600-byte name, so `Message` is cut inside it: the kind, `Tool` empty | red for the kind |
| `TestToolSchemaRefusal_NoLog_0084` (`providers`) | ST45 | a provider given a recording `slog` handler logs no record that holds the refusal's text or `Tool` | green (a guard) |
| `TestToolSchemaRecord_LogLine_0084` (`llmprovider`, Phase 1a) | ST47 | the case's log line for a case whose body holds a made-up key is exactly `"<id>: <outcome>, status <n>"` | red at Phase 1a's base (the function does not exist) |

Details added to tests already listed: `NeverNamesUnsent` gains a request
that sent no tools; `Reworded`'s OpenAI case changes the code to
`invalid_prompt` (a content-policy code); `Classify` asserts `errors.As`
finds the `*APIError` and that its `Kind` is `ErrToolSchemaRefused`.
Plants added to Phase 3 step 1: one per row above marked "(added)" or
"added below", each with the plant the test map gives. Phase 3 step 1 had
no plant for `RedactedNeighbour` or `SeveralNamed`; the map gives each one.

### PLAN tests that map to no row

None. Two need a word, pending the owner:

* `Classify` asserts "each recorded refusal ... Phase 1's". A Phase 1
  refusal that is not a schema refusal (O5's, ST58) cannot be the kind;
  which test each Phase 1 record feeds is the MADR's ST53 to ST59, and N1
  decides the records no matcher recognises.
* `SeveralNamed` exists only if O2 records two names (ST55); N7 decides
  whether it is built from a constructed body otherwise.

### Re-grounding at `24756a6`

`tools/claims.py` passes 29 of 29 rows here at their pins. With `self` at
`24756a6` five rows fail, each a line that moved with the same text; every
code row (P1 to P15, P18 to P20, P24, P25) holds. The rows stay pinned:

| Row (here) | Pinned line at `b109cea` | Line at `24756a6` | Meaning |
| :--- | :--- | :--- | :--- |
| P16 | `docs/architecture.md:563` | `:566`, same text | unchanged |
| P21 | `docs/guides/api-standards.md:214` | `:215`, same text | unchanged; R24 now also lists `Reason` |
| P22 | `docs/guides/api-standards.md:217` | `:218`, same text | unchanged |
| P23 | `docs/decisions/0080-PLAN-verify-a-kilo-credential-through-its-profile.md:219` | `:207` and `:228`, same text | the form is unchanged; it now occurs twice |
| P28 | `AGENTS.md:278` | `:295`, same text | unchanged |

At gobble-cli's `HEAD` (`b1994a0`; pinned `cecc892`), P26 and P27 hold
their text on lines 398 and 16; the pin is still `v1.31.0`.

No code this PLAN changes has moved since `b109cea`: of the files under
`llmprovider/` it cites, only `providers/openai/` changed, by an added test
file (`held_session_0082_test.go`). Two facts the PLAN's steps assumed do
not hold as written, and are put to the owner rather than corrected here:
the `param` read cannot reach the helper (MADR H5, N2), and the D3 rows
cannot tell OpenCode's routes or OpenAI's modes apart (H3, H4, N3).

### The owner's live steps, in plain words

*(Superseded later on 2026-11-20 by the owner's ruling "API keys from the
environment only (Recommended)", relayed by the coordinator: the run is an
agent's or the coordinator's. What is left for the owner is in the second
amendment of 2026-11-20 below.)*

Still to run, in order, once the owner has answered N1 to N8 and an agent
has committed Phase 1a's recorder:

1. **Open a fresh terminal window in the checkout** and confirm it is at
   the recorder's commit. About a minute.
2. **Type or paste five API keys, each at a prompt that shows nothing**:
   xAI, OpenAI, Anthropic, Gemini and OpenCode (the OpenCode key needs Zen
   and Go access). A key the owner lacks is skipped, and its routes are
   recorded `skipped`. Two or three minutes. Nothing else is typed in that
   window that could show a key.
3. **Make a folder for the records outside the checkout.** Seconds.
4. **Run the recorder once.** 51 small calls (nine routes; at most 64
   output tokens each), one line per case showing only its id, outcome and
   status. Estimated at a few minutes; not measured. A case that says
   `other` (a timeout or a 5xx) is run again alone.
5. **Clear the keys from the shell and close the window.** Seconds.
6. **Tell the coordinator the run is done.** The records stay in the
   folder for Phase 1c.

In all, roughly 15 to 25 minutes of the owner's time, an estimate. What
each case's result decides is the MADR's ST53 to ST59; with N1 and N8
answered, no result needs a further ruling. Whether a refused request is
billed is not established for any provider.

## Second amendment of 2026-11-20: the rulings in the tests, and who runs the recording

A second records pass, the same day, at `24756a6`. It applies the owner's
rulings (N1 to N8, a1, ST10, and "API keys from the environment only
(Recommended)", each relayed by the coordinator and quoted in the MADR's
second amendment) and an independent review's findings. The changes above
are dated annotations and strike-throughs: the Rules' stop list and
`make api-check` stop, the Fixtures-only rule, the out-of-scope notes, the
Prerequisites' keys, Phase 1a's steps 1 to 5, Phase 1b's run (now an
agent's) and its stop list, Phase 1c's order, Phase 2's test table, code,
docs and Verification, Phase 4, and the Verification list.

### The test map, rows changed or added

Plants are single production changes, run by `tools/plant.py` on a
scratch clone; each joins Phase 3 step 1. Rows not listed keep the first
amendment's map.

| Row | Test | Phase | Plant |
| :--- | :--- | :--- | :--- |
| ST03 | `Names`; `NeverNamesUnsent`'s `param` subcase | 2 | the name taken from `param`'s index |
| ST05 | `Deferral`, on a Claude model in the Append form | 2 | the index mapped through `Request.Tools` |
| ST08 | `TestToolSchemaRefusal_SharedLabel_0084` | 2 | the helper not called on OpenCode's Messages route |
| ST10 | `TestToolSchemaRefusal_SearchLoaded_0084`; `NeverNamesUnsent` | 2 | the helper's list leaves out the definitions inside `tool_search_output` |
| ST12 | `SeveralNamed` | 2 | the helper takes the first of several names |
| ST13 | `TestToolSchemaRefusal_DuplicateNames_0084` | 2 | the helper requires exactly one entry, not one name |
| ST17 | `TestToolSchemaRefusal_StreamUnchanged_0084` | 2 | the D3 rows applied in `setStreamKind`'s calls too |
| ST20, ST23, ST50 | `ProxyAltered`, no longer conditional | 2 | a row's status condition dropped |
| ST26, ST32 | `TestToolSchemaRefusal_Precedence_0084` | 2 | the D3 rows placed before the overflow row; the `x-should-retry` rule skipping the new kind |
| ST28, ST44 | `FieldsUnchanged`, moved to `providers`, read through `Generate` and `Stream`, its expected fields those `ClassifyHTTPError` gives the same reply | 2 | the helper appends the tool's name to `Message` |
| ST30 | `Classify` and `Names`, each asserting `errors.As` and `Kind` | 2 | `ClassifyHTTPError` returns `fmt.Errorf("%w: %w", ErrToolSchemaRefused, e)` with `e.Kind` left `ErrInvalidRequest` |
| ST33 | `TestNameRefusedTool_Wrapped_0084` | 2 | the helper type-asserts `*APIError` instead of `errors.As` |
| ST36 | `make api-check` exit 0; ~~`tools/apidiff.py <the phase's start commit>`~~ `tools/apidiff.py <base> <scratch commit>` on a scratch clone (corrected 2026-11-20, fourth pass) | 2 | a third exported name: listed |
| ST37 | `Classify` | 2 | `ErrToolSchemaRefused`'s parent made `ErrRateLimited`, ~~so `Retryable()` is true~~ (replaces "terminal false", which no test could catch) *(corrected 2026-11-20, fourth pass: `Retryable()` stays false, since the D3 rows return a terminal error (`api_error.go:197` to `:199`). The plant is caught because `Unwrap` then also yields the 400's legacy `ErrInvalidRequest`, against `Classify`'s assertion that it yields the kind alone, and by an assertion added to `Classify`: `errors.Is(err, ErrRateLimited)` is false. A second plant sets `shouldRetry` for the kind, which makes `Retryable()` true (`:194` to `:196`), and is caught by `Classify`'s `Retryable()` assertion)* |
| ST40 | `Concurrent`, its server holding the first request until the second arrives | 2 | the sent list kept in a provider field |
| ST43 | `NameBeyondBound` | 2 | the helper, finding no sent name, falls back to the last caller tool sent (ST11's plant; replaces the prefix match, red only by a partial parse) |
| ST45 | `NoLog` | 2 | the helper logs the error at `Debug` |
| ST46 | `TestToolSchemaRecord_Redacts_0084`, widened | 1a | the record's `Message` and `Code` left unredacted, the body still redacted |
| ST47, ST79 | `TestToolSchemaRecord_LogLine_0084` (five fields); `TestToolSchemaRecord_NoLeak_0084` | 1a | the case line also prints `Message` |
| ST60 | `TestToolSchemaRefusal_RecordedFallback_0084` | 2 | a row loosened to its code or type alone (Phase 3) |
| ST67, ST68, ST70, ST78 | `TestToolSchemaRecord_Outcome_0084` | 1a | four plants: a 404 read as `refused`; a `Status`-0 `ErrIncomplete` read as `other`; the writer replacing a `refused` record; a blank key read as set |
| ST69 | `Names` | 2 | Anthropic's row removed (Phase 3) |
| ST71 | the `grep` check of Phase 1a step 5 | 1a | `t.Fatal(err)` in a scratch copy; seen 2026-11-20 on `live_0062_refusals_measure_test.go`, which it hit seven times |
| ST72 | `Names` (as ST09); `TestCheckDispatch_0040`, unchanged | 2 | none new |
| ST73, ST74 | `grep -c ErrToolSchemaRefused` on each file; `TestDescribe_ToolSchemaRefused_0084`; Phase 4's read | 2 | the entry removed on a scratch copy: exit 1 |
| ST77 | `NoLeak`; `grep -nE 'os\.Args\|flag\.\|os\.ReadFile'` on the recorder's files, exit 1 | 1a | the recorder logs its key variable's value |
| ST80 | `NoLeak` | 1a | the body written before redaction |
| ST81 | Phase 1c's order: the checks on the records and on `tmp/0084-run.txt` before any read | 1c | none: a process rule |
| ST82 | none: no such step | – | – |

### Tests added in this pass (dated addition, 2026-11-20)

| Test (package) | Row | Asserts | At the stub |
| :--- | :--- | :--- | :--- |
| `TestToolSchemaRefusal_SharedLabel_0084` (`providers`) | ST08 | OpenCode Zen's chat route and Go's Messages route each serving F3, and OpenAI in a ChatGPT session (built offline as the package's session tests build one) serving F2: the kind; `Tool` the sent name | red |
| `TestToolSchemaRefusal_SearchLoaded_0084` (`providers`) | ST10 | OpenAI's Responses with native deferral (a model of 5.4 or later): F2 renamed to a deferred tool loaded by a `tool_search_output`, then to the caller's Search tool: `Tool` is each name | red |
| `TestToolSchemaRefusal_DuplicateNames_0084` (`providers`) | ST13 | two sent tools named `record_city`, F1: `Tool` is `record_city` | red |
| `TestToolSchemaRefusal_StreamUnchanged_0084` (`llmprovider`) | ST17 | `ClassifyStreamFailureBody` given code `400` and F7's error object, and given F2's code and message: each the kind it has at the base | green (a guard) |
| `TestToolSchemaRefusal_Precedence_0084` (`llmprovider`) | ST26, ST32 | F1, F2, F4, F7 with `x-should-retry: true`: `ErrProviderUnavailable`, `Retryable()` true; F4 with an Anthropic overflow phrase added to its message: `ErrContextOverflow` | green (a guard) |
| `TestNameRefusedTool_Wrapped_0084` (`internal/wire`) | ST33 | the kind wrapped by `fmt.Errorf("%w")` and by `errors.Join`: `Tool` set on the `*APIError` found; the returned error is the one given | red |
| `TestToolSchemaRefusal_RecordedFallback_0084` (`llmprovider`) | ST60, ST53 to ST59, ST66 | each Phase 1 record no matcher recognises: `ErrInvalidRequest`, `Tool` empty. Not built when there is none, as the execution record then says | green (a guard) |
| `TestToolSchemaRecord_Outcome_0084` (`llmprovider`, Phase 1a) | ST67, ST68, ST70, ST78 | each row of Phase 1a step 2's outcome table; a key unset, `""` and `"   "`: `skipped`, "key unset or blank"; the writer keeps a `refused` or `accepted` record against every other outcome, and writes `skipped` only where none exists | red at Phase 1a's base |
| `TestToolSchemaRecord_NoLeak_0084` (`llmprovider`, Phase 1a) | ST77, ST79, ST80 | the recorder's case path, its output writer and log function replaced by buffers, a made-up key-shaped value set by `t.Setenv` under the route's variable, against an in-process transport replaying F2 with a made-up key-shaped string and a made-up `org-` id planted in its body and message: no record, output or log line holds the key or either planted value, and the line holds only the five allowlisted fields | red at Phase 1a's base |

Tests changed: `FieldsUnchanged` (now in `providers`), `Concurrent` (the
barrier), `Redacts` (every field), `LogLine` (five fields), `Deferral`
(the Append form), `Classify` and `Names` (`errors.As` and `Kind`),
`ProxyAltered` (unconditional), `NeverNamesUnsent` (the `param` subcase
added, the `tool_search` entry moved to `SearchLoaded`), `SeveralNamed`
(always built).

### The rows added or changed in this pass, checked once against the review's three questions

* **Missing cases.** One found and fixed while writing: the recorder's own
  second send of an `other` case doubles the worst-case time, so the
  timeout is 180 minutes, not the 100 first written. One more: a panic in
  the run would print its value to the output file, so that file passes
  `scripts/check_fixtures.py` and `tools/denyscan.py` before it is read
  (ST81). None else found in ST66 to ST82 or the changed rows.
* **Decided rows that a ruling does not quite decide.** Three readings, put
  to the coordinator in the hand-back rather than taken as settled: ST66
  (1), that Anthropic's marker joins the `opencode` service only when the
  run records Anthropic's body unchanged on Zen Messages (N3's "the
  provider's marker", with N1 and N8); ST67, that 408, 409, 5xx and other
  `Status`-0 failures are `other` until a second send (N8's "a single case
  still not recorded after its rerun"); and the recorder sending that
  second send itself, so the run is one command. ST73 and ST74's checks
  test presence; their content rests on Phase 4's read.
* **Tests that cannot pin their row.** `NoLeak` needs the recorder to take
  its output writer and log function as parameters: Phase 1a builds it
  so. `SharedLabel`'s ChatGPT case relies on the `openai` package building
  a session offline, as its tests already do. `RecordedFallback` may have
  no case, said in the execution record if so.

### What is left for the owner

Nothing in this record needs the owner at a browser, a device sign-in, a
saved session, an account page or a console; an agent runs the recording.
Left for the owner: ~~the answer to N9,~~ (answered, third amendment) and committing and pushing the
records and each phase's work, which the standing rule leaves to them.

## Third amendment of 2026-11-20: N9 answered

The third and last records pass of the day. It records the owner's ruling
on N9, "Treat that case as unrecorded, fix the recorder, rerun
(Recommended)", relayed by the coordinator. It also records the drafter's
readings and the a2 wording note, both in the MADR's third amendment.
Phase 1b's last stop is struck, and Phase 1c gains its rules for a hit.

### The test map, rows changed or added

| Row | Test | Phase | Plant |
| :--- | :--- | :--- | :--- |
| ST76 | `TestToolSchemaRecord_NoLeak_0084`, extended with the missed value's shape in the further recorder commit | 1a (again) | the extension itself, seen red at the recorder before the fix |
| ST83 | `test_fixtures_check_*` in `scripts/test_gates.py`, unchanged (they assert the value is not printed); `TestToolSchemaRecord_FieldPerLine_0084` (added) | 1a | the writer writes a record compact, on one line |

### The test added (dated addition, 2026-11-20)

| Test (package) | Row | Asserts | At the stub |
| :--- | :--- | :--- | :--- |
| `TestToolSchemaRecord_FieldPerLine_0084` (`llmprovider`, Phase 1a) | ST83 | ~~a written record holds each top-level field on its own line, its body included, so a reported line names one field~~ *(corrected 2026-11-20, fourth pass: every line of a written record lies under exactly one top-level key, the nearest line at or above it at two spaces' indent; the body, `Message` and `Code` are each one line; a nested tool or schema spans several lines under its key)* | red at Phase 1a's base |

### What is left for the owner

Nothing in this record needs the owner's hands, and no question is open.
Left for the owner: committing and pushing the records and each phase's
work, which the standing rule leaves to them.

## Fourth amendment of 2026-11-20: corrections from the delta check

Corrections only, after an independent delta check found seven items not
confirmed. Each item was checked at its cited lines and holds. The changes
above, each dated "fourth pass":

* the outcome table, corrected (a2);
* the stop texts and owner-run sentences, struck with pointers to ST76
  (a6);
* the Phase 1b and 1c exit criteria, which now cite ST66 (a5);
* Phase 1b's command, with `-trimpath`, its output allowlist and its scans
  (ST79, ST84, ST85);
* Phase 1a's forbidden helpers and `PrintCalls` test (ST71);
* the record's layout, restated (ST83);
* Phase 1c's removal of a leaked copy;
* ST36's comparison of two commits on a scratch clone;
* ST37's rationale.

A search of both files for a sentence telling the owner to run, type or
decide something in Phases 1a to 1c, or a stop a live observation can
reach, found these: the Goal's "recorded live by the owner", "No agent
runs the live test", Phase 1a step 6's "Before the owner runs it", Phase
1b's "the owner confirms", the outcome table's "the owner runs the case
once more", Phase 1c's "a hit is a stop" and "open as N9", the
Acceptance note's "one stop left", the Verification's "the owner's run",
the Order's "when the owner chooses", "one owner run", and "the answer to
N9". Each is now struck with a dated pointer. The one stop a live
observation can still reach is ST76's bound: the same case leaking again
after one fix and rerun.

### The test map, rows changed or added

| Row | Test | Phase | Plant |
| :--- | :--- | :--- | :--- |
| ST60 | `RecordedFallback`: each unrecognised `refused` record classifies, at the phase's tree, to the kind `ClassifyHTTPError` gives it at the base, with `Tool` empty | 2 | a row loosened to its code or type alone (Phase 3) |
| ST67 | `Outcome`, each row of the corrected table: Gemini's and xAI's 400 key refusals (`ErrAuthFailure`), a 402 (`ErrQuotaExhausted`), a 3xx, a 400 with `x-should-retry: true`: `unreachable` | 1a | the `errors.Is(err, ErrInvalidRequest)` condition dropped from the `refused` row |
| ST69 | `Names` and `Classify`: a read route with no recognised Phase 1 record (skipped, unreachable, root string accepted, or no record recognised) uses its 0062 fixture with the tool moved to index 1 | 2 | Anthropic's row removed (Phase 3) |
| ST71 | `TestToolSchemaRecord_PrintCalls_0084` (replaces the struck `grep`) | 1a | on a scratch copy: `t.Fatal(genErr)`; `fmt.Println(err)`; `llmprovider.SkipIfTransient(t, err)`. Each must turn it red |
| ST79 | `TestToolSchemaScan_0084` on the run's output file: each line matches the allowlist | 1b | a line of provider text in the output (in `TestToolSchemaScan_Plant_0084`) |
| ST84 | the four scans of Phase 1b on `tmp/0084-run.txt` | 1b | `TestToolSchemaScan_Plant_0084`'s output-file case |
| ST85 | `TestToolSchemaScan_0084`; `TestToolSchemaScan_Plant_0084` | 1a, 1b, 1c | the scan compares `redact.Redact`'s output with itself, so it never reports |

### Tests added in this pass (dated addition, 2026-11-20)

| Test (package) | Row | Asserts | At the stub |
| :--- | :--- | :--- | :--- |
| `TestToolSchemaRecord_PrintCalls_0084` (`llmprovider`, Phase 1a) | ST71, ST79 | the three recorder files, parsed with `go/parser`: no reference to `SkipIfTransient`, `skipIfTransient`, `LiveModel` or `liveModel`; no `t.Log`, `Logf`, `Error`, `Errorf`, `Fatal`, `Fatalf`, `Skip`, `Skipf`, `fmt.Print*`, `fmt.Fprint*` or `log.` call with an argument that is not a string literal, a constant of fixed text, or an allowlisted field (case id, route, model, status, outcome) | red at Phase 1a's base |
| `TestToolSchemaScan_0084` (`llmprovider`, untagged, Phase 1a) | ST79, ST84, ST85 | skips unless `LLMPROVIDER_LIVE_0084_SCAN` names a file or directory. For each line of each file, it reports `<file>:<line>: key shape` when `redact.Redact` changes the line (`internal/redact/redact.go:136`). For the output file it also reports `<file>:<line>: not allowlisted` for a line matching none of Phase 1b's allowed shapes. It never prints a line's text, and fails on any report | a scan, run in Phases 1b and 1c; it skips at the stub, and its own proof is the next test |
| `TestToolSchemaScan_Plant_0084` (`llmprovider`, Phase 1a) | ST85, ST84, ST79 | (1) records the writer makes from F1, F2, F4 and F7, and an output file of allowed lines: no report. (2) Copies holding, each on its own line: a made-up OpenAI `sk-proj-` key, Anthropic `sk-ant-api03-` key, xAI `xai-` key of 80 characters, Gemini `AIza` key, OpenCode `sk-` key of 64 characters, a `Bearer` value and an `"api_key": "…"` pair (the five providers' shapes are rows of `reToken`, `redact.go:90` to `:108`), plus an output line of provider text. Exactly those lines are reported, and no planted string appears in the test's captured output | red at Phase 1a's base |

The scan finds a key by the shapes the redactor knows, which are the
shapes the writer redacts. A key in a shape `internal/redact` does not
know would pass both. A scan for the exact values of the keys in the
environment is not used: the owner's ruling forbids anyone to compare a
key's value. The keys this run uses, xAI's, OpenAI's, Anthropic's,
Gemini's and OpenCode's, each have a shape row.
