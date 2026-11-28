---
status: complete
date: 2026-11-23
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
  and its tests. *(Annotated 2026-11-21, the owner's ruling "Accept the
  internal package (Recommended)": and `llmprovider/internal/toolschema`, the
  one place the four forms' markers are written, read by `api_error.go` and by
  this helper; see build pass 2. `api_error.go` also holds the error-body
  limit, `apiErrorBodyLimit`, raised to 1 MiB by the owner's ruling on ST43,
  MADR ST86.)*
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
   `skipped` record. `TestToolSchemaRecord_Outcome_0084` pins both.)* *(Annotated
   2026-11-21, build pass 4: the one-case rerun holds again; pass 3 had dropped
   the subtests it rests on, and pass 4 restores them.)* *(Annotated 2026-11-21,
   after the re-review: the command that runs it, second send included, is
   the corrected one at Phase 1b step 4, which names `(#01)?`.)*

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
   | an `*APIError` with `Status` 400 to 499, other than 401, 403, 404, 408, 409 and 429, ~~that `errors.Is` `ErrInvalidRequest`~~ *(struck 2026-11-21: `errors.Is` is true for a refused key's and a 402's error; the owner's answer, "Test the error's own kind (Recommended)", relayed by the coordinator)* whose `Kind` is nil (the status-mapped `ErrInvalidRequest`) or satisfies `errors.Is(apiErr.Kind, ErrInvalidRequest)` | `refused` |
   | an `*APIError` with `Status` 408, 409 or 500 to 599, or `Status` 0 of a kind not beneath `ErrIncomplete` | `other`; the recorder sends it once more after the pass and writes `unreachable` if it is still `other` |
   | any other `*APIError`: 300 to 399, 401, 403, 404, 429, or a 4xx of another kind (a refused key, a 402's exhausted quota, `ErrNotPermitted`, `x-should-retry: true`) | `unreachable` |
   | an error that is not an `*APIError` | `unreachable`, reason `transport`, never the error's text |

   *~~A `refused` record is therefore always of a kind beneath
   `ErrInvalidRequest` at the base.~~ *(Struck 2026-11-21: true of `Kind`, not of `errors.Is`. A `refused` record is always of a `Kind` that is nil or beneath `ErrInvalidRequest` at the base.)** `TestToolSchemaRecord_Outcome_0084`
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
   per case, no `WithRetry`. It logs, per case, only ~~`"%s: %s, status %d"`
   (the id, the outcome, the status)~~ *(superseded 2026-11-21: the five-field line of the later bullets)*.
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
     'os\.Args|flag\.|os\.ReadFile'` on them exits 1;* *(Annotated 2026-11-21, "Record all three readings (Recommended)": the `grep` for `os.Args|flag\.` exits 1 on all four recorder files, and the `grep` for `os\.ReadFile` exits 1 on the live file alone, since the write function and the scan file may read a file.)*
   * *each case's one line, on standard output and in the test log, is
     `"<case> <route> <model>: status <n>, <outcome>"` and nothing else; no
     body, `Message`, `Code` or error text is printed or logged, before or
     after redaction;*
   * *after the pass over every case, the recorder sends each `other` case
     once more and writes `unreachable` for one still `other`, so one
     command records everything; the one-case command of Phase 1b step 4
     is then needed only to repeat a case on purpose;* *(Annotated 2026-11-21,
     build pass 4: holds again, the subtests being restored.)* *(Annotated
     2026-11-21, after the re-review: the one-case command is the corrected
     one at Phase 1b step 4, `^<case>(#01)?$`.)*
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
       `Helper`, `Logf`, `Fatal`, `Fatalf` and `Skip`;* *(Annotated
       2026-11-21, "Record all three readings (Recommended)": `Helper`, `Logf`,
       `Fatal`, `Fatalf` and `Skip` resolve to `(*testing.common)`, where
       `testing` declares them; the check names them so, with the same
       meaning, and `Run` as `(*testing.T).Run`.)* *(Annotated 2026-11-21,
       build passes 3 and 4: `Errorf` joins the named methods, with the argument
       rule, for the second pass's fixed failure text; and a call through
       `caseRunner`, the injected runner of `recordToolCases`, is allowed with
       a `toolCase` variable and a function literal alone.)*
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

     *~~The check is complete by construction for the recorder's own files:
     a call not named above cannot pass it.~~ Its stated limit is that a
     library the recorder is allowed to call may itself print; the output
     scan (MADR ST84, ST85) covers that.)* *(Annotated 2026-11-21, build
     pass 3, the independent review's B1 to B3: the claim was false as built.
     The argument rule matched a field by its name alone, the one write
     function was matched by its name alone, and a reassignment of the path
     was not refused. Plants passed `CallAllowlist`; the fixes are in the
     build pass 3 entry below, and each plant joins `toolPlants`.)*
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

*(Corrected 2026-11-21, Phase 1b's run. The first scan command above passes
`LLMPROVIDER_LIVE_0084_SCAN=tmp/0084-run.txt`, a path relative to the
checkout, but `go test` runs in the package directory (`llmprovider/`), so
the scan failed with "the scan could not read its target" and read nothing.
Pass an absolute path, `"$PWD/tmp/0084-run.txt"` run from the checkout's
root, or `../tmp/0084-run.txt`. The records scan already used an absolute
path. Mechanical; no decision changes.)*

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

   *(Corrected 2026-11-21, after the re-review of build pass 4. The two
   one-case blocks above, with `-run '^TestLive_ToolSchema_0084$/^grok-O1-root-union$'`,
   run the first send of a case and skip its second: Go names the second
   subtest of a name `<case>#01`, and `-run` is applied to that name, so an
   anchored `^<case>$` does not match it (measured, build pass 4). They are
   not to be run. The one-case command is this one, with the case's id in
   place of `<case>`, and its worked example for `grok-O1-root-union`:)*

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$/^<case>(#01)?$' -count=1 -timeout 5m -v
   ```

   ```sh
   LLMPROVIDER_LIVE_0084=1 LLMPROVIDER_LIVE_0084_OUT="$HOME/tool-schema-0084" go test -tags live_gateways ./llmprovider -run '^TestLive_ToolSchema_0084$/^grok-O1-root-union(#01)?$' -count=1 -timeout 5m -v
   ```

   *(The pattern is the owner's one-case rerun made to reach the second
   send; it is the same command in every other respect, with the same
   `-timeout 5m`. A case that was not `other` has no `#01` subtest, and the
   pattern then runs the one.)*

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

### Phase 1a, build pass 1 (2026-11-21)

Stopped, by the stop condition "a decided row cannot be met as written". Start
state: branch `main`, HEAD `96594e4`, tree clean before the first write.

* **Re-grounding at `96594e4`, each held.**
  * P13: `llmprovider/refusals_0062_record_test.go:465`, `func
    outsideModule(`, held.
  * P14: `llmprovider/refusals_0062_record_test.go:61`, `const
    refusalBodyLimit = 4 << 10`, held.
  * P18: `llmprovider/provider.go:100`, `ProviderGrok:   "XAI_API_KEY",`,
    held.
  * P19: `llmprovider/provider.go:103`, `ProviderOpencodeZen:
    "OPENCODE_API_KEY",`, held.
  * P24: `llmprovider/live_0062_refusals_measure_test.go:28`, `func
    TestLive_Refusals_0062(`, held.
  * P28: moved from `AGENTS.md:278` to `:295`, same text, meaning unchanged
    (as the amendment of 2026-11-20 recorded).
  * P29: `docs/reports/0062-REPORT-thinking-refusals.md:31`, held.
* **What was built before the stop, and kept out of the tree.** The record
  file, the live file and the scan file, compiling (`go vet` with and
  without `-tags live_gateways`, exit 0, on a scratch clone), kept in a
  scratch folder, not in the tree. The offline proofs file, the plants and the
  gate were not written or run. On the scratch clone: the key-shape scan
  reports no line of the 62 files under `testdata/refusals-0062/`, and
  `go/types` with `importer.ForCompiler(fset, "source", nil)` and cgo off
  type-checks the three files with no type error in 7.4 seconds and
  resolves every callee (so the allowlist is buildable as specified). No
  live call was made; no `LLMPROVIDER_LIVE_*` variable was set.
* **The stop: ST67's `refused` condition cannot be met as written.** The
  corrected outcome table and ST67 say a `refused` record is an `*APIError`
  of status 400 to 499, bar 401, 403, 404, 408, 409 and 429, "that
  `errors.Is` `ErrInvalidRequest`", and that Gemini's and xAI's 400 key
  refusals, a 402 and a 400 with `x-should-retry: true` are `unreachable`.
  `APIError.Unwrap` appends the status's legacy sentinel for every
  terminal error whose kind differs (`llmprovider/api_error.go`, `Unwrap`),
  and the status-only mapping gives every 4xx but 401, 403 and 429
  `ErrInvalidRequest` (`statusSentinel`). So `errors.Is(err,
  ErrInvalidRequest)` is true for each of those errors. Measured on a
  scratch clone at `96594e4`, through `ClassifyHTTPError`: xAI 400 key
  refusal, `errors.Is(ErrInvalidRequest)` true and `ErrAuthFailure` true;
  Gemini 400 `API_KEY_INVALID`, both true; Claude 402, `ErrInvalidRequest`
  true and `ErrQuotaExhausted` true; a 302, `ErrInvalidRequest` true; only
  the 400 with `x-should-retry: true` is false (`ErrProviderUnavailable`).
  `TestToolSchemaRecord_Outcome_0084` is specified to assert the 402, the 3xx
  and the two 400 key refusals `unreachable`, which no build with the
  `errors.Is` condition passes. The sentence "a `refused` record is
  therefore always of a kind beneath `ErrInvalidRequest`" is true of
  `APIError.Kind`, not of `errors.Is`.
* **Resolutions, recommended first.** (1) Test the kind, not the unwrap
  chain: `refused` only when `apiErr.Kind` is nil (the status-mapped
  `ErrInvalidRequest`) or `errors.Is(apiErr.Kind, ErrInvalidRequest)`
  (`ErrContextOverflow`, a workspace kind and the like), with every other
  condition of ST67 as written; amend ST67 and the table above in the
  MADR and this PLAN, then resume. Cost: one amendment and one condition in
  one function. (2) Keep `errors.Is` as written and drop the 402 and the two
  key refusals from ST67's `unreachable` list and from the Outcome test:
  a bad key then records as `refused` and joins Phase 1c's fall-back
  tests, which contradicts the owner's ruling on N8 and ST68's route-level
  skip. Not recommended. Consequence of doing nothing: the Outcome test
  cannot be made to pass, and a recording with a refused xAI or Gemini key
  or an exhausted quota would write `refused` records that Phase 1c reads as
  schema refusals.
* **Also noted, not a stop.** ST77's check names `os.ReadFile` in the
  recorder's files with `grep` exiting 1, and the last edit allows
  `os.ReadFile` in the one write function (to keep a `refused` or
  `accepted` record, ST70) and in the scan file. Both cannot hold; the build
  reads the last edit's allowlist as the later and specific rule, runs the
  `grep` for `os.Args` and `flag.` on all four files, and for `os.ReadFile`
  on the live file alone, unless the pair is amended.
* **Deviation resolved, 2026-11-21.** The stop above was put to the owner,
  who answered, relayed by the coordinator: "Test the error's own kind (Recommended)". `refused` only when the
  `*APIError`'s status is 400 to 499 other than 401, 403, 404, 408, 409 and 429, and
  its `Kind` is nil (the status-mapped `ErrInvalidRequest`) or `errors.Is(apiErr.Kind,
  ErrInvalidRequest)`; every other ST67 condition is unchanged, and bad keys and a 402
  stay `unreachable`, per N8. Resolution (1) above is the one chosen. The owner also
  answered "Record all three readings (Recommended)": (1) ST77's `grep` runs for `os.Args` and `flag.` on all four
  recorder files and for `os.ReadFile` on the live file only; (2) the five-field line
  supersedes the first amendment's `LogLine` format; (3) the call allowlist names
  `t.Logf`, `Fatal`, `Fatalf`, `Skip` and `Helper` as `(*testing.common)` methods, with
  the same meaning. The corrections are annotated in place above and in the MADR's fifth
  amendment of 2026-11-21. One more test row: `Outcome` gains a plant that tests
  `errors.Is(err, ErrInvalidRequest)` in place of the kind, which must turn it red. No
  file is added to scope: the four recorder files and their places are as the Files line
  says. The recorder files kept out of the tree at the stop are not copied in until the
  coordinator releases the build.

### Phase 1a, build pass 2 (2026-11-21)

Built on `96594e4`, after the owner's answers recorded above, with the tree
holding only the two records' edits at the start. No live run: no
`LLMPROVIDER_LIVE_*` variable was set, and the tagged file was compiled and
vetted, never run.

* **Re-grounding.** Pass 1's table stands: P13, P14, P18, P19, P24 and P29
  held at their pinned lines, P28 at `AGENTS.md:295`, each with the same
  text. `tools/claims.py` passes 85 of 85 and 29 of 29.
* **What was built.** Four files in `llmprovider/`, all package
  `llmprovider_test`:
  * `toolschema_0084_record_test.go` (untagged): the case table (57 cases),
    `toolOutcome` as amended (status range, and `Kind` nil or beneath
    `ErrInvalidRequest`), the capture transport, `toolFill`,
    `recordToolCase`, the redaction (`redact.String`, then the fixture rules
    for request ids, account ids, workspace ids, UUIDs, e-mail addresses and
    home paths), and `writeToolRecord`, the one function that writes;
  * `live_0084_toolschema_measure_test.go` (`live_gateways`):
    `TestLive_ToolSchema_0084`, with the second send of each `other` case;
  * `toolschema_0084_scan_test.go` (untagged): `TestToolSchemaScan_0084`,
    which skips unless `LLMPROVIDER_LIVE_0084_SCAN` is set, and the scan;
  * `toolschema_0084_record_offline_test.go` (untagged): `Cases`,
    `InsideModule`, `Redacts`, `LogLine`, `Outcome`, `NoLeak`,
    `FieldPerLine`, `CallAllowlist` and `TestToolSchemaScan_Plant_0084`.
  The two locally-held function literals of pass 1 are declared functions
  (`toolBranch`, `toolCity`). `toolInsideModule` sits in the record file, not
  the live file, so that `InsideModule` tests it offline; the allowlist
  names the same calls either way.
* **Models, per route, none replaced.** Each route keeps its 0062 model:
  `grok` `grok-4.6`, `grok2` `grok-4.5`, `openai` `gpt-6-luna`,
  `zen-responses` `gpt-5.5`, `claude` and `zen-messages` `claude-sonnet-5`,
  `zen-google` and `gemini` `gemini-3.8-flash`, `zen-chat` `glm-5.3`,
  `go-messages` `qwen3.8-flash`. Whether each is still listed was not checked,
  since that needs a live call; Phase 1b is where an unlisted model shows.
* **Gate.**

  | Command | Exit | Decisive line |
  | :--- | :--- | :--- |
  | `go test -count=1 ./llmprovider/` | 0 | `ok` |
  | `go vet -tags live_gateways ./llmprovider/` | 0 | no output |
  | `go test -c -tags live_gateways -o <scratch>/x.test ./llmprovider/` | 0 | no output |
  | `run-heavy -- make pre-add-check FILES=<the four files>` | 0 | `4 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
  | `make records-check` | 0 | `182 records, 0 problem(s)` |
  | `make docs-check` | 0 | `0 violation(s) of R50,R51,R52` |
  | `make links-check` | 0 | `0 problem(s)` |
  | `make changelog-check` | 0 | `0 problem(s)` |
  | `make tables-check` | 0 | `0 problem(s)` |
  | `make disclosure-check` | 0 | `0 hit(s)` |
  | `make api-check` | 0 | `0 incompatible change(s)` |

  The first pre-add run failed on two lints in the offline file (`errorlint`
  on a type assertion, `gocognit` on `check`); `check` was split and the
  assertion made `errors.AsType`, and the second run passed. That is the one
  build pass of the phase's gate that failed.
* **Greps, as the amended ST77 reads them.** `grep -nE 'os\.Args|flag\.'` on
  the four recorder files exits 1 (no hit); `grep -nE 'os\.ReadFile'` on the
  live file exits 1; `grep -nE 'SkipIfTransient|skipIfTransient|LiveModel|liveModel'`
  on the four exits 1. Each `grep` exits 0 on a scratch copy of the live file
  with `os.Args`, `flag.Parse` and `os.ReadFile` appended.
* **Plants.** `python3 -B tools/plant.py` on a scratch clone, 15 plants,
  each caught (exit 0). Each failure line below is the test's:
  * redaction skipped: `Redacts`, `claude-root-string.json holds a planted
    value (52 bytes, starting "xai-")`;
  * `Message` and `Code` left unredacted, body still redacted: `Redacts`,
    `Message holds a planted value (16 bytes, starting "org-")`;
  * body written before redaction: `Redacts`, `Body holds a planted value
    (52 bytes, starting "xai-")`;
  * a 404 read as refused: `Outcome`, `404: refused, ""; want unreachable, ""`;
  * a `Status`-0 `ErrIncomplete` read as other: `Outcome`, `incomplete 200:
    other, ""; want accepted, ""`;
  * the writer replacing a refused record: `Outcome`, `claude-root-string:
    wrote unreachable over the record; it holds unreachable; want refused`;
  * a blank key read as set: `Outcome`, `blank key: false, , ""; want
    skipped, "key unset or blank"`;
  * `errors.Is(err, ErrInvalidRequest)` in place of the kind (the owner's
    amendment): `Outcome`, `xai 400 key refusal: refused, ""; want
    unreachable, ""`, and the same for Gemini's, the 402 and a not-permitted
    4xx;
  * the kind condition dropped from the refused row: `Outcome`, the same
    four rows and `400 x-should-retry: refused, ""; want unreachable, ""`;
  * the case line also prints `Message`: `LogLine`, `printed
    "claude-root-string claude claude-sonnet-5: status 400, refused: tools.0...`;
  * the case line also prints the key variable's value: `NoLeak`, `output
    holds a made-up value (starting "sk-p")`;
  * the writer writes a record compact: `FieldPerLine`, `1 lines, first "{
    \"case\": \"claude-O2-two-bad\", ...` (checked on a scratch copy; the
    `plant.py` run showed `Outcome`'s writer rows failing too, `it holds
    unknown; want skipped`);
  * the control tool not first: `Cases`, `grok-root-string: tools [{nop ...}
    ...]; want noop first`;
  * the scan compares `Redact` with itself: `Scan_Plant`, `records copy:
    reported map[] (ok true); want map[3 key shape:true ...]`;
  * the scan's allowlist inverted: `Scan_Plant`, `an output file of allowed
    lines: 16 hits, ok true; want none`.
  The first run of the plant for the writer replacing a refused record was
  caught only by a build failure (`declared and not used: old`), which proves
  nothing; its text was changed to `if _ = old; ...` and it was caught by the
  assertion above.
  The allowlist's own plants run inside `CallAllowlist` on in-memory copies;
  each report line it must hold, as logged: `os.Stderr.WriteString` ->
  `(*os.File).WriteString: call is not on the allowlist`; `io.WriteString`
  -> `io.WriteString: call is not on the allowlist`; `println(err)` ->
  `println: builtin is not allowed`; `panic(err)` -> `panic: builtin is not
  allowed`; `slog.Info` -> `import "log/slog" is not allowed`; `tb := t;
  tb.Log(err)` -> `(*testing.common).Log: call is not on the allowlist`;
  `getListing` -> `type error: undefined: getListing`; an import of `fmt` ->
  `import "fmt" is not allowed`; `t.Fatal(err)` ->
  `(*testing.common).Fatal: an argument is not a constant or an allowlisted
  field`; `t.Run(err.Error(), f)` -> `(*testing.T).Run: an argument is not a
  constant or an allowlisted field`; and, added: `os.WriteFile` elsewhere ->
  `outside the one write function`; `os.ReadFile` elsewhere -> `outside the
  write function and the scan file`; a write path not built from `dir` ->
  `path argument is not built from filepath.Join(dir, ...)`; a call through a
  function value -> `f: call through a function value`; the case line given
  `Message` -> `emitFunc: an argument is not a constant or an allowlisted
  field`. An `init` function and a package-level call in the offline file are
  caught by the same test. The guard, the three files as built, passes with
  no report, 60 or more calls inspected and two or more through `emitFunc`;
  the type check takes about 8 seconds.
* **Rows.** ST46: `Redacts` (plants 1 to 3). ST47, ST79: `LogLine`,
  `NoLeak` (plants `LogLine` and key variable). ST67, ~~ST68,~~ ST70, ST78 *(ST68 struck 2026-11-21, build pass 3)*:
  `Outcome` (its table includes the xAI and Gemini 400 key refusals, the 402,
  the 302 and `x-should-retry`; plants 4 to 9). ST71, ST79: `CallAllowlist`.
  ST77, ST80: `NoLeak`, and the amended `grep`s. ST83: `FieldPerLine`
  (the closing bracket or brace of a nested value, at two spaces' indent, is
  read as nested under the key above, since `wiretest.Encode` writes them there). ST85, ST84, ST79:
  `TestToolSchemaScan_Plant_0084` and the scan. On the 62 files under
  `testdata/refusals-0062/` the scan reports no line.
* **Not done, and why.** Phase 1b and later phases (not this phase). No
  model-listing check (needs a live call). `TestToolSchemaScan_0084` was not
  run on a real output file or records (none exist before 1b). The
  `make markdownlint` target does not lint the records; with `--no-globs`
  on the two records it finds MD004 only, as the committed records do.
* **Docs step.** Nothing in `README.md`, `docs/guides/`,
  `docs/architecture.md`, `docs/README.md` or `CHANGELOG.md` names a 0084
  recorder or is made false by test-only files; the plan's changelog and
  guide entries are Phase 2's. No docs row to fix.

### Phase 1a, build pass 3 (2026-11-21)

Opened on the owner's answer, relayed by the coordinator: "Third pass: fix all, records first (Recommended)".
An independent review of build pass 2 found three blockers and three rows
with no test that can fail. The reviewer's plants are kept in the session's
scratch folder (`review-0084-1a/`, `plants.py`, `plants2.py`, `plant3.py` and
their reports); none is in the tree. This entry records the findings and
what pass 3 builds; the code follows in a later step, after the records are
gated. A further stop is reported as before, and there is no fourth pass
without the owner.

* **B1, the argument rule matches a field by name alone**
  (`toolschema_0084_record_offline_test.go:721`,
  `toolNamedFields[se.Sel.Name]`). Plants that passed `CallAllowlist`, exit 0:
  P1, `emit(toolLineFormat, toolCase{id: r.Message}.id, ...)`; P1c,
  `t.Logf("%s", scanHit{label: root}.label)` in the scan file; P1d,
  `resp.Status` (an `http.Response`'s reason phrase) through an `emitFunc`.
  **Fix:** the argument must be a constant, or a selector whose
  `info.Selections` entry is a `types.FieldVal` of a named field whose
  receiver is `toolCase` or `toolRecord` (the record and live files) or
  `scanHit` (the scan file alone), declared in the package; and the selector's
  operand must itself be a variable or a field, not a composite literal or a
  call, so that `toolCase{id: r.Message}.id` is refused.
* **B2, the one write function is matched by name**
  (`toolschema_0084_record_offline_test.go:650` to `:653`, `:781`). P3, a
  method `func (toolW) writeToolRecord(dir string)` in the live file writing
  to `filepath.Join(dir, "..", "x")`, passed. **Fix:** the write function
  counts only when it has no receiver and is declared in the record file.
* **B3, a reassignment of the path is not refused**
  (`toolschema_0084_record_offline_test.go:669` to `:689`, `:803` to
  `:812`). P2, `path := filepath.Join(dir, ...)` followed by `if r.Case == ""
  { path = r.Message }`, passed. **Fix:** refuse any assignment to a path
  identifier whose right-hand side is not `filepath.Join(dir, ...)`, and any
  assignment to the `dir` parameter.
* **ST67's second send is untested.** The two-pass loop lives in the tagged
  live file, so no untagged test runs it; P11, which dropped it, left
  `-run _0084` at exit 0. **Fix:** the loop moves into an untagged function
  in the record file, `recordToolCases(cases, dir, base, emit)`, which the
  live test calls. New test `TestToolSchemaRecord_SecondSend_0084` (ST67):
  against replays, 503 then 503 gives `unreachable`, and 503 then 400 (a
  refusal) gives `refused`, and the file on disk agrees. The second pass
  fails with `t.Errorf` and the fixed text `toolTextNotWritten`, so that one
  failed write does not stop the other retries; `(*testing.common).Errorf`
  joins the allowlist and the argument rule, and the fixed text stays on the
  scan's allowlist.
* **ST46, "never the model's text", is untested.** Only the gate
  `rep.status/100 != 2` in `toolFill` keeps a 200 body out of a record; P4
  removed it and every 0084 test but the allowlist passed. **Fix:** new test
  `TestToolSchemaRecord_NoModelText_0084` (ST46): a `Status`-0
  `ErrIncomplete` arrives with a 200 body holding made-up model text; the
  record has no `Body` and neither it nor the written file holds any of the
  text. P4 becomes a plant.
* **ST68 is Phase 1c's rule, not Phase 1a's.** ST68 says a route with no
  `refused` or `accepted` case is `skipped` as a whole, and its 0062 fixtures
  stand. The recorder writes one record per case and cannot know a route's
  other cases; the rule is applied when Phase 1c sorts the records. It is
  struck from the test-map rows of Phase 1a and from pass 2's row list above,
  by annotation, and moves to Phase 1c: its check is Phase 1c's per-route line
  in the execution record, and `Names` and `Classify` in Phase 2 use the
  0062 fixture for such a route (ST69). The MADR's fifth-amendment table
  carries the same move in its sixth amendment.
* **Tests and plants added to Phase 1a's list.** In `toolPlants`, on
  in-memory copies, each to turn `CallAllowlist` red for the reason it
  plants: P1 (a fresh `toolCase{id: ...}` in an `emit` argument); P1c (a
  fresh `scanHit{label: ...}` in a `t.Logf` of the scan file); P1d
  (`resp.Status` through an `emitFunc`); P2 (the path reassigned after the
  `Join`); P3 (a method named `writeToolRecord` in the live file writing
  outside `dir`); and, for B2 and B3, an assignment to `dir`. By
  `tools/plant.py` on a scratch clone: P4 (the `rep.status/100 != 2` gate
  removed, caught by `NoModelText`) and P11 (the second pass dropped, caught
  by `SecondSend`).
* **Non-blocking items fixed in pass 3.** `writePathIdents` no longer panics
  on an unnamed first parameter; the checker's Go version comes from
  `go.mod`, not a literal; the second pass's write failure is a `t.Errorf`;
  the useless `t.Helper()` in the live subtest is removed.
* **Non-blocking items left as follow-ups, not fixed.** UUID over-redaction
  (the fixture rule redacts every UUID, a public object's included); the
  overlapping `request_id` and `req_` rules; the 4 MiB read limit of the
  capture transport; symlinks under the output directory; the text of a
  provider-build failure, which the recorder drops. The reviewer's P10, a
  range over a method value (an implicit call with no `CallExpr`), was not in
  the owner's list and is left open for the coordinator. *(Annotated 2026-11-21,
  build pass 3: not an open gap. The reviewer's own report has P10 caught,
  `CallAllowlist` exit 1 with `(*os.File).WriteString: call is not on the
  allowlist`; it joins `toolPlants` as a guard plant, by the coordinator's
  word.)*
* **Files.** The four files of pass 2 only; no file is added to scope. The
  record file's contents gain `recordToolCases`.

#### Outcome of build pass 3 (2026-11-21)

Built on `96594e4`, after the records above were gated. No live run, no
`LLMPROVIDER_LIVE_*` variable set, nothing staged or committed.

* **What changed.**
  * B1: the argument rule is by type: a constant, or a field read from a
    variable (a literal or a call is refused) that `types.Info.Selections`
    marks a `FieldVal` of `toolCase` (`id`, `route`, `model`), `toolRecord`
    (`Status`, `Outcome`) or `scanHit` (`file`, `line`, `label`), declared in
    the package; `scanHit` only in the scan file, the other two only outside
    it. The reviewer's fix named the receiver alone, which P1
    (`toolCase{id: r.Message}.id`) passes, so the operand rule is added.
  * B2: the write function is the record file's function with no receiver.
  * B3: any assignment to `dir`, and any to a path identifier from anything
    but `filepath.Join(dir, ...)`, is refused.
  * `recordToolCases(cases, dir, base, emit)` is in the record file; the live
    test is one call to it, and `t.Errorf(toolTextNotWritten, c.id)` for each
    case returned. `(*testing.common).Errorf` joins the allowlist and the
    argument rule.
  * New tests: `TestToolSchemaRecord_SecondSend_0084` (503 then 503
    unreachable; 503 then 400 refused; 400 once, one send; an unwritable
    directory returns the case) and `TestToolSchemaRecord_NoModelText_0084`
    (a cut-short 200 with model text has no body and no text in the file; an
    accepted answer has counts and no text).
  * Non-blocking: `writePathIdents` takes an unnamed first parameter; the Go
    version is read from `go.mod`; the second pass's failure is `t.Errorf`;
    the live subtest's `t.Helper()` is gone with the subtests.
* ~~**A deviation from the plan's step 1, for the owner to see.** The live test
  no longer runs each case as a subtest (`<route>-<shape>`), because the loop
  that sends each `other` case a second time had to move into the untagged
  `recordToolCases` to be tested, and a subtest needs the live test's `t`. A
  single case can no longer be selected with `-run
  'Live_ToolSchema_0084/<case>'`; a rerun is the whole run, which the writer's
  keep rule (ST70) makes harmless, and Phase 1b's one-case command is not
  needed. Every case line, record and outcome is as before.~~ *(Struck 2026-11-21,
  build pass 4, by the owner's answer "Fourth pass: B3 + keep subtests (Recommended)": the subtests are
  restored, one per case, through an injected runner; see the build pass 4
  entry below.)*
* **Gate, exit codes.** `go test -count=1 ./llmprovider/` 0 (`ok`);
  `go vet -tags live_gateways ./llmprovider/` 0; `go test -c -tags
  live_gateways -o <scratch>/x.test ./llmprovider/` 0; `run-heavy -- make
  pre-add-check FILES=<the four files>` 0 (`4 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck)`); the `make` targets and the
  record checks are in the hand-back. The ST77 `grep`s: `os\.Args|flag\.` on
  the four files exits 1, `os\.ReadFile` on the live file exits 1, and the
  forbidden helper names on the four exit 1.
* **Plants, in `toolPlants` (in memory, `CallAllowlist`),** each caught:
  P1, `emitFunc: an argument is not a constant or an allowlisted field`; P1c,
  `(*testing.common).Logf: an argument is not a constant or an allowlisted
  field` (scan file); P1d, `emitFunc: an argument is not a constant or an
  allowlisted field`; P2, `path reassigned from something but
  filepath.Join(dir, ...)`; the `dir` assignment, `assignment to dir`; P3,
  `os.WriteFile: outside the one write function`; P10 (a guard), `(*os.File).WriteString: call is
  not on the allowlist`; `t.Errorf(err)`, `(*testing.common).Errorf: an
  argument is not a constant or an allowlisted field`. All of pass 2's
  plants are still caught, and the guard passes.
* **Plants, by `tools/plant.py` on a scratch clone,** 17 caught, exit 0,
  the pass 2 set and: P4, the 200 body gate removed, `NoModelText`, `a cut-short
  200: accepted, 200, body of 101 bytes; want accepted, 200, no body`; P11,
  the second pass dropped, `SecondSend`, `503 then 503: 1 sends; want 2`.
* **The reviewer's own scripts, on a scratch copy of the pass 3 files,**
  each run as the reviewer wrote it, except where noted: `plants.py` exit 0
  (P1 and P2 and P3 fail `CallAllowlist`, exit 1; P1b fails `LogLine` and
  `NoLeak`; P4 fails `NoModelText` once its run pattern, which named the old
  tests alone, names it; P6 and P7 fail; P5, P8 and P9 exit 0 as the reviewer
  expected, being equivalent or informational); `plants2.py` exit 0 (P1, P1c,
  P1d, P3 and P10 fail `CallAllowlist`, exit 1; P11 fails `SecondSend`, exit
  1, its anchor moved from the live file to the record file, where the loop
  now is); `plant3.py` exit 0, printing `exit 1`, `NoLeak` failing on the
  body written before redaction.
* **Left as follow-ups, unchanged.** UUID over-redaction; the overlapping
  `request_id` and `req_` rules; the 4 MiB read limit; symlinks under the
  output directory; the text of a provider-build failure. A variable that
  holds a `toolCase` literal built with `id: r.Message` and is then read is not
  seen by the operand rule, which only refuses a literal or a call in the
  operand; the output scan (MADR ST84, ST85) covers it.

### Phase 1a, build pass 4 (2026-11-21)

Opened on the owner's answer, relayed by the coordinator: "Fourth pass: B3 + keep subtests (Recommended)".
The re-review of pass 3 found B1, B2, `SecondSend` and `NoModelText` closed,
and B3 and the subtest deviation not. The reviewer's `plants4.py` and its
report are in the session's scratch folder (`review-0084-1a/`); none is in
the tree. A further stop is reported as before, and there is no fifth pass.

* **B3, still open: the path is tracked by name, not by variable.** Three
  plants passed `CallAllowlist`, exit 0: N3, `{ var path = r.Message; _ =
  os.WriteFile(path, enc, 0o600) }`, a shadowing declaration; N4,
  `func(path string) { _ = os.WriteFile(path, enc, 0o600) }(r.Message)`, a
  function-literal parameter; N5, `p := &path; *p = r.Message`, a write
  through a pointer. **Fix:** record the `types.Object` that `path :=
  filepath.Join(dir, ...)` defines (`info.Defs`); in `pathArgOK` require
  `info.Uses[id]` to be that object, so a shadowing variable or a parameter
  of the same name is refused; refuse any other assignment to that object;
  and refuse `&` of it or of the `dir` parameter. Plants N3, N4 and N5 join
  `toolPlants`.
* **The subtest deviation of pass 3 is not accepted.** One subtest per case
  is restored. In the record file, `type caseRunner func(c toolCase, run
  func(emit emitFunc))` and `recordToolCases(cases, dir, base, each
  caseRunner)`, every send, the first and the second, made through `each`.
  In the live file, the runner calls `t.Run(c.id, ...)` and runs the case with
  `t.Logf`. In the offline proofs, `SecondSend` passes a runner that calls
  `run(toolBuffer(&buf))`. The checker allows a call through `caseRunner`
  only with a `toolCase` variable and a function literal. A one-case rerun,
  `-run '^TestLive_ToolSchema_0084$/^<case>$'`, again runs that one case and,
  if it was `other`, its second send. Plants: a `caseRunner` call given
  anything else, and a runner that drops the second send.
* **What the static check cannot follow, recorded as its limit.** The
  reviewer's N1, `x := toolCase{id: r.Message}; x.id`, and N2, `c.id =
  r.Message`, carry a value through a variable or a field, which the
  argument rule cannot follow. `TestToolSchemaRecord_LogLine_0084` and
  `TestToolSchemaRecord_NoLeak_0084` catch both at run time, since the line
  they pin is exact; both plants are run against those tests.
* **Records.** The one-case rerun mentions in step 2, in the run rules of
  step 5 and in the MADR's ST51 hold again, annotated, not struck; step 5's
  list of named methods gains `Errorf` and the `caseRunner` rule; pass 3's
  deviation entry is struck, each with a dated note.

#### Outcome of build pass 4 (2026-11-21)

Built on `96594e4`, after the records above. No live run, no
`LLMPROVIDER_LIVE_*` variable set, nothing staged or committed.

* **B3, by object.** `writeObjects` takes the `types.Object` that `path :=
  filepath.Join(dir, ...)` defines (`info.Defs`, now in the checker's `Info`)
  and the `dir` parameter's; `pathArgOK` requires `info.Uses[id]` to be that
  object, so a shadowing variable or a literal's parameter named `path` is
  another object and is refused; `writeAssigns` refuses any other assignment to
  either object, an increment, a range over either, and the address of either
  (`&`). Plants N3, N4 and N5, and one for `&dir`, are in `toolPlants` and
  caught: N3 and N4, `os.WriteFile: path argument is not built from
  filepath.Join(dir, ...)`; N5 and `&dir`, `address of dir or path taken`.
* **Subtests restored.** `type caseRunner func(c toolCase, run func(emit
  emitFunc))` and `recordToolCases(cases, dir, base, each caseRunner)` are in
  the record file, every send, the first and the second, made through `each`.
  The live test's runner is `t.Run(c.id, func(t *testing.T) { run(t.Logf) })`.
  The checker allows a call through `caseRunner` only with a `toolCase`
  variable and a function literal, and a call to a `func(emitFunc)` run
  function only with `t.Logf`. New test `TestToolSchemaRecord_Runner_0084`
  (one runner call per case, and one more per case that was `other`);
  `SecondSend` passes a buffer runner. Plants in `toolPlants`: a `caseRunner`
  call given a call as its case, and given a function value; a `run` given
  something but `t.Logf` (`run: the argument is not t.Logf`).
* **One finding on the one-case rerun, with its resolution.** The brief's
  command, `-run '^TestLive_ToolSchema_0084$/^<case>$'`, runs the first send
  of a case and not its second. Go names the second subtest of a name
  `<case>#01` and applies `-run` to that name, so an anchored `^<case>$` skips
  it. Measured on a scratch clone with a replayed 503 and the live test's
  runner: `^claude-root-string$` ran `claude-root-string` alone, `status 503,
  other`; `^claude-root-string(#01)?$` ran `claude-root-string` and
  `claude-root-string#01`, `status 503, other` then `status 503, unreachable`.
  The one-case command is therefore `-run
  '^TestLive_ToolSchema_0084$/^<case>(#01)?$'`, and the live test's comment
  says so. The runner is as the owner's answer asks; only the pattern differs.
* **The static check's limit, recorded.** The reviewer's N1, `x :=
  toolCase{id: r.Message}; x.id`, and N2, `c.id = r.Message`, pass
  `CallAllowlist` (exit 0), since the argument rule cannot follow a value
  through a variable or a field. `LogLine` and `NoLeak` catch both at run
  time: the plants N1 and N2 below, run by `tools/plant.py`, each fail
  `LogLine` (`printed "tools.0.custom.input_schema.type: Input should be
  'object' claude claude-sonnet-5: status 400, refused\n"; want
  "claude-root-string ...`). *(Extended 2026-11-21, after the re-review: the
  reviewer's R5, `c.id = r.Message` before the second send, is the same class
  as N1 and N2. `CallAllowlist` passes it, exit 0, and `Runner` and
  `SecondSend` catch it at run time, since the second line they pin is exact.)*
* **Gate, exit codes.** `go test -count=1 ./llmprovider/` 0 (`ok`); `go vet
  -tags live_gateways ./llmprovider/` 0; `go test -c -tags live_gateways -o
  <scratch>/x.test ./llmprovider/` 0; `run-heavy -- make pre-add-check
  FILES=<the four files>` 0 (`4 file(s) clean (gofmt, golangci-lint, go vet,
  go test, govulncheck)`). A first pre-add run failed on `gocyclo` in
  `call` and a `staticcheck` De Morgan finding; the variable-call case became
  `varCall`, and the second run passed. The ST77 `grep`s: `os\.Args|flag\.` on
  the four files exits 1, `os\.ReadFile` on the live file exits 1, the
  forbidden helper names on the four exit 1.
* **Plants by `tools/plant.py` on a scratch clone, 20 caught (exit 0):** pass
  3's set, and the second send made without the runner (`Runner`, `the runner
  ran [claude-root-string claude-O1-root-union]; want [... claude-O1-root-union
  claude-O1-root-union]`), N1 and N2 (`LogLine`). P11 now fails `Runner` and
  `SecondSend`.
* **The reviewer's scripts, on a scratch copy of the pass 4 files,** run as
  written except two anchors. `plants.py` and `plants2.py` are the copies of
  pass 3, whose P4 run pattern names `NoModelText` and whose P11 anchor is in
  the record file (the loop is there). Results: `plants.py` exit 0 (P1, P2, P3
  exit 1 on `CallAllowlist`; P4 exit 1 on `NoModelText`; P5, P8 and P9 exit 0
  as the reviewer expected); `plants2.py` exit 0; `plants4.py` exit 0,
  re-run on the final tree after the lint fix: P1, P1c, P1d, P3, P10, N3, N4
  and N5 exit 1 on `CallAllowlist`; P11' and P4' exit 1 on `_0084`; N1 and N2
  exit 0, the recorded limit.
* **Left as follow-ups.** As in pass 3: UUID over-redaction, the overlapping
  `request_id` and `req_` rules, the 4 MiB read limit, symlinks under the
  output directory, a provider-build failure's text, and the limit of the
  static check named above.
* **One more follow-up, recorded 2026-11-21, not a fix.** The live test's
  `each` literal never runs offline: no untagged test builds it. A literal
  that forgot to call `run` would send nothing, and `recordToolCases` would
  return no failure. Phase 1b's output would show it, as a run with no case
  lines.

### Phase 1b, the recording run (2026-11-21)

Run by the coordinator under the owner's ruling "API keys from the environment
only (Recommended)", at `a800e74`. The five key variables (`XAI_API_KEY`,
`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `OPENCODE_API_KEY`)
were checked present by name only; no value was read.

* **Command.** The corrected step 2 command (`-trimpath`, `-timeout 180m`),
  records to `<scratch>/tool-schema-0084`, output to `tmp/0084-run.txt`. Exit
  0; `ok ... 274.571s`; 57 case lines; 175 output lines.
* **Deviation, mechanical, for the owner.** The first scan command's relative
  path failed (corrected beside the command, above). It was run again with the
  file's absolute path: `--- PASS: TestToolSchemaScan_0084`.
* **Scans, each exit 0, run before any line was read:** the key-shape scan on
  the output (`--- PASS`, checked with `-v`), `scripts/check_fixtures.py
  tmp/0084-run.txt`, `tools/denyscan.py tmp/0084-run.txt`, and the key-shape
  scan on the records directory (`--- PASS`).
* **Outcomes by route (case lines), 57 in all:** claude 200 accepted 2, 400
  refused 4; gemini 200 accepted 3, 400 refused 3; go-messages 200 accepted 6;
  grok 200 accepted 3, 400 refused 3; grok2 400 refused 3; openai 200 accepted
  4, 400 refused 2; zen-chat 200 accepted 6; zen-google 400 refused 3, 429
  unreachable 3; zen-messages status 0 unreachable 2, 200 accepted 4;
  zen-responses 200 accepted 4, 400 refused 2. No case was `other`, so no
  second send was made; none was `skipped`.
* **The five `unreachable` cases** were each run once more with the one-case
  command (`-run '^TestLive_ToolSchema_0084$/^<case>(#01)?$' -timeout 5m`), as
  step 4 allows; each output passed the three scans and gave the same line:
  `zen-google-root-string`, `zen-google-O4-nested-union`,
  `zen-google-O5-history-dropped` (429, unreachable); `zen-messages-root-string`,
  `zen-messages-O2-two-bad` (status 0, unreachable). The records scan passed
  again after the reruns; 57 records.
* **Models.** Each route kept its 0062 model (Phase 1a); none was unlisted.

### Phase 1c, the fixtures (2026-11-21)

By an agent, at `a800e74`, tree clean but for the git-ignored `tmp/`.

* **Re-grounding.** P28 holds, moved from `AGENTS.md:278` to `AGENTS.md:295`,
  same text (`an account, user or organization id becomes its prefix and
  `REDACTED``). `make fixtures-check` at `a800e74` before the copy: exit 0,
  `fixtures-check: 428 file(s), 134 hit(s), 0 problem(s)`.
* **Copy.** The 57 records were copied unread, unchanged, from
  `<scratch>/tool-schema-0084` to `llmprovider/internal/wire/testdata/toolschema-0084/<wire>/`
  (`chatcompletions` 6, `generatecontent` 6, `interactions` 6, `messages` 18,
  `responses` 21).
* **Checks before any record was read, each exit 0:** `make fixtures-check`
  (`485 file(s), 134 hit(s), 0 problem(s)`: 428 plus the 57, hits unchanged);
  `python3 -B tools/denyscan.py --changed` (`0 hit(s) in 57 file(s)`);
  `TestToolSchemaScan_0084` on the copies with `-v` (`--- PASS`, not
  `--- SKIP`). No hit, so no copy was removed (ST76, ST83).
* **One line per case.** Columns: id; outcome; status; code; whether and where
  the tool is named; the test it feeds (ST53 to ST59, ST66; ST68, ST69).
  "Names" is the matcher test `Names`; `Classify`, `RecordedFallback`,
  `OthersUnchanged` and `SeveralNamed` are the tests the rows name.

| Case | Outcome | Status | Code | Tool named? Where | Feeds |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `claude-O1-root-union` | refused | 400 | invalid_request_error | yes, by index: `tools.1.` (`record_city`) | ST54 (1): recognised; joins `Classify`, `Names` |
| `claude-O2-two-bad` | refused | 400 | invalid_request_error | yes, by index: `tools.1.` (`record_city`) | ST55 (1): one tool named, `Tool` is it, `SeveralNamed` not built; one named by index; joins `Names` |
| `claude-O3-typed-branches` | refused | 400 | invalid_request_error | yes, by index: `tools.1.` (`record_city`) | ST56: evidence only (refused); no test |
| `claude-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `claude-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `claude-root-string` | refused | 400 | invalid_request_error | yes, by index: `tools.1.` (`record_city`) | ST53 (1): the 0062 form again; joins `Classify`, `Names` |
| `gemini-O1-root-union` | refused | 400 | invalid_request | yes, `'record_city'` in the message | ST54 (3): form not read by D3; `RecordedFallback` |
| `gemini-O2-two-bad` | refused | 400 | invalid_request | yes, `'record_city'` in the message | ST55 (4): not recognised; `RecordedFallback` |
| `gemini-O3-typed-branches` | refused | 400 | invalid_request | yes, `'record_city'` in the message | ST56: evidence only (refused); no test |
| `gemini-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `gemini-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `gemini-root-string` | accepted | 200 | - | - | ST53 (3)/D3 not read: accepted, recorded, no change |
| `go-messages-O1-root-union` | accepted | 200 | - | - | ST54 (2): accepted, no change |
| `go-messages-O2-two-bad` | accepted | 200 | (200, `empty_answer`: ST67 accepted) | - | ST55 (5): nothing |
| `go-messages-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `go-messages-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `go-messages-O5-history-dropped` | accepted | 200 | (200, `empty_answer`: ST67 accepted) | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `go-messages-root-string` | accepted | 200 | (200, `empty_answer`: ST67 accepted) | - | ST53 (3)/D3 not read: accepted, recorded, no change |
| `grok-O1-root-union` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST54 (1): recognised; joins `Classify`, `Names` |
| `grok-O2-two-bad` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST55 (1): one tool named, `Tool` is it, `SeveralNamed` not built; one named; joins `Names` |
| `grok-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `grok-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `grok-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `grok-root-string` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST53 (1): the 0062 form again; joins `Classify`, `Names` |
| `grok2-O1-root-union` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST54 (1): recognised; joins `Classify`, `Names`; ST59 (1) |
| `grok2-O2-two-bad` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST55 (1): one tool named, `Tool` is it, `SeveralNamed` not built; one named; joins `Names` |
| `grok2-root-string` | refused | 400 | invalid-argument | yes, `record_city` after the tag, before `:` | ST53 (1): the 0062 form again; joins `Classify`, `Names`; ST59 (1) for O6 |
| `openai-O1-root-union` | accepted | 200 | - | - | ST54 (2): accepted, no change |
| `openai-O2-two-bad` | refused | 400 | invalid_function_parameters | yes, `function 'record_country'` in the message | ST55 (1): one tool named, `Tool` is it, `SeveralNamed` not built; only one tool is bad (`record_country`), so one named is expected; joins `Names` |
| `openai-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `openai-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `openai-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `openai-root-string` | refused | 400 | invalid_function_parameters | yes, `function 'record_city'` in the message | ST53 (1): the 0062 form again; joins `Classify`, `Names` |
| `zen-chat-O1-root-union` | accepted | 200 | - | - | ST54 (2): accepted, no change |
| `zen-chat-O2-two-bad` | accepted | 200 | - | - | ST55 (5): nothing |
| `zen-chat-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `zen-chat-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `zen-chat-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `zen-chat-root-string` | accepted | 200 | - | - | ST53 (3)/D3 not read: accepted, recorded, no change |
| `zen-google-O1-root-union` | refused | 400 | INVALID_ARGUMENT | no: an unknown field at `function_declarations[1]`, by index, no name | ST54 (3): form not read by D3; `RecordedFallback` |
| `zen-google-O2-two-bad` | refused | 400 | INVALID_ARGUMENT | no: an unknown field at `function_declarations[1]`, by index, no name | ST55 (4): not recognised; `RecordedFallback` |
| `zen-google-O3-typed-branches` | refused | 400 | INVALID_ARGUMENT | no: an unknown field at `function_declarations[1]`, by index, no name | ST56: evidence only (refused); no test |
| `zen-google-O4-nested-union` | unreachable | 429 | RESOURCE_EXHAUSTED | - | ST57 (4): nothing |
| `zen-google-O5-history-dropped` | unreachable | 429 | RESOURCE_EXHAUSTED | - | ST58 (4): nothing |
| `zen-google-root-string` | unreachable | 429 | RESOURCE_EXHAUSTED | - | ST53 (4): the 0062 fixture stands at index 1 (ST69) |
| `zen-messages-O1-root-union` | accepted | 200 | - | - | ST54 (2): accepted, no change; ST66 (4) |
| `zen-messages-O2-two-bad` | unreachable | 0 | - | - | ST55 (5): nothing |
| `zen-messages-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `zen-messages-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `zen-messages-O5-history-dropped` | accepted | 200 | (200, `empty_answer`: ST67 accepted) | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `zen-messages-root-string` | unreachable | 0 | - | - | ST66 (5) (status 0 twice): nothing recorded; Anthropic's marker does not join `opencode` on this evidence |
| `zen-responses-O1-root-union` | accepted | 200 | - | - | ST54 (2): accepted, no change |
| `zen-responses-O2-two-bad` | refused | 400 | invalid_function_parameters | yes, `function 'record_country'` in the message | ST55 (1): one tool named, `Tool` is it, `SeveralNamed` not built; only one tool is bad, as openai; joins `Names` |
| `zen-responses-O3-typed-branches` | accepted | 200 | - | - | ST56: evidence only (accepted); no test |
| `zen-responses-O4-nested-union` | accepted | 200 | - | - | ST57 (1): accepted, no change |
| `zen-responses-O5-history-dropped` | accepted | 200 | - | - | ST58 (1): accepted; gobble's drop is safe on this route |
| `zen-responses-root-string` | refused | 400 | invalid_function_parameters | yes, `function 'record_city'` in the message | ST53 (1): the 0062 form again; joins `Classify`, `Names` |

* **ST68.** Every route has a `refused` or an `accepted` case, so none is
  `skipped` and each keeps its 0062 fixtures for the cases it did not record
  (ST69): `zen-google` `root-string` (429) and `zen-messages` `root-string`
  (status 0) use the 0062 fixture with the tool moved to index 1.
* **ST69, the read routes' fixture sources for `Names` and `Classify`.**
  Recognised Phase 1 records: grok (`root-string`, `O1`, `O2`), grok2
  (`root-string`, `O1`, `O2`), openai and zen-responses (`root-string`, `O2`),
  claude (`root-string`, `O1`, `O2`). `zen-google` has no recognised Phase 1
  record: its three refusals are in a form D3 does not read, so it uses its 0062
  fixture.
* **Unrecognised refusals (`RecordedFallback`, ST54 (3), ST55 (4)):**
  `gemini-O1-root-union`, `gemini-O2-two-bad`, `zen-google-O1-root-union`,
  `zen-google-O2-two-bad`.
* **ST56 (O3), evidence only, each route's outcome:** accepted (200) on xAI
  (`grok-O3`), OpenAI, Zen Responses, Zen chat, Go Messages and Zen
  Messages; refused (400) on Gemini (Interactions) and Anthropic; refused (400)
  on Zen Google for another reason (below).
* **ST58 (O5):** every route that returned a result accepted a history calling
  a tool no longer offered (`OthersUnchanged` gets no record: none was refused).
* **ST66, Zen Messages:** its `root-string` and `O2` are `unreachable`
  (status 0, each twice), so no refusal was recorded on it; (5) applies: it
  stays `ErrInvalidRequest`, Anthropic's marker does not join `opencode` on this
  evidence. Its `O1`, `O3`, `O4` and `O5` were accepted (`(4)`).
* **Observations the rows do not predict, for the owner (follow-ups, not
  fixes):** (1) `zen-google` refuses `O1`, `O2` and `O3` for the recorder's own
  `additionalProperties: false`, an unknown field to the Gemini function
  declaration on that route, not for the union; so `O1` there shows nothing
  about the owner's shape, `root-string` and `O4` and `O5` are `unreachable`
  (429), and a tool schema holding `additionalProperties` is refused on that
  route as a plain 400 with no tool name. (2) `openai` and `zen-responses`
  accept the owner's shape (`O1`), and their `O2` has only one bad tool
  (`record_country`), so it cannot show two named. (3) `gemini` refuses
  `O1` to `O3` in a form naming `record_city` in single quotes, which D3 does not
  read. (4) Three `go-messages` and one `zen-messages` 200s are `empty_answer`,
  recorded `accepted` by ST67.
* **Checks after the records were written, each exit 0:** `make
  fixtures-check` (485 files, 134 hits, 0 problems), `python3 -B tools/denyscan.py
  --changed` (0 hits in 59 files), `make records-check` (182 records),
  `make links-check` (0 problems), `make tables-check` (0 problems),
  `tools/claims.py` on both records (85 of 85, 29 of 29),
  `tools/recordlint.py --changed` (no rule rose). Left unstaged for the owner.
* **Docs step.** README.md, docs/README.md, docs/architecture.md, docs/guides/,
  CHANGELOG.md and AGENTS.md name no testdata index or fixtures guide that this
  phase makes false.

### Phase 2, build pass 1 (2026-11-21)

Built by an agent at `9e84a88` (Phase 1c's commit), on `main`, tree clean at
the start. Nothing is staged or committed; no live run was made, no
`LLMPROVIDER_LIVE_*` variable was set, no key was read. The MADR is unchanged.

* **Step 1, the base re-read.** Every line the PLAN and the MADR's H1 to H8
  cite holds its text; the lines that moved are `docs/architecture.md`'s
  P16, now `:569` (`:566` at `24756a6`, `:563` at `b109cea`), and none other.
  P1 `api_error.go:538`, P2 `:103`, P3 `:690`; P4 `deferral.go:253`; P5 and
  P6 `tools.go:85`, `:101`; P7 and P8 `grok.go:252`, `:297`; P9 `openai.go:317`
  (its stream `:333`); P10 `claude.go:283`; P11 and P12 `opencode.go:322`,
  `:765` (the Messages send `:334`, H3); P15 `llmtest.go:125`; P17
  `CHANGELOG.md:12`; P20 `describe.go:15`; P21 and P22
  `api-standards.md:215`, `:218`; P25 `retry.go:20`. H1 `api_error.go:309`,
  `:314`; H2 `:441` and `generatecontent.go:307`; H3 `serviceOf` `:498`,
  `:500`; H4 `openai.go:317`, `:333`; H5 `api_error.go:259`, `:275` (the
  envelope is still not kept on the error); H6 `contract.go:271`; H7
  `reauth.go:50`, `:57`, `learn.go:300`, `responses/refusal.go:49`; H8
  `post.go:115`; C13 `:559`; C20 `messages/learn.go:80`. `git diff --stat
  24756a6 HEAD` over `llmprovider/`, `wizard/`, `docs/architecture.md`,
  `docs/guides/` and `CHANGELOG.md` shows no change to a Go file this phase
  changes: the changes are 0081's and 0083's wizard files, `signin.go`, the
  Phase 1 recorder and the two docs. No meaning changed, so no stop. Relation
  to 0080 and the others: 0080's **Added** and **Changed** entries are in
  `CHANGELOG.md` `[Unreleased]`, and this phase adds its own **Added** beside
  them; 0081, 0083, 0085 and 0087 hold no file in common with this phase
  (their PLANs cite `api_error.go`, `claude.go` and `opencode.go` as evidence
  only; 0087's names `wizard/describe.go` as not changed). 0080's gates ran the
  suite on the Windows laptop by `run-on-host` and `pre-add-check` and `-race`
  by `run-heavy` on the session host; this phase's ran on this Mac by
  `run-heavy`.
* **Step 2, the stub**, on a scratch clone of `9e84a88`, committed there and
  never pushed: `ErrToolSchemaRefused` as D1 says and `Tool string` on
  `APIError`, in `api_error.go` alone; nothing classified, nothing named. The
  tests were copied in, and `tools/redgreen.py --root <scratch> --rev <stub>`
  ran each package once. **Red at the stub, each by its assertion** (the first
  failing line is given; every one fails on the kind, and the ones marked
  both also on `Tool`): `Classify` (`llmprovider`, `:141`, "want
  ErrToolSchemaRefused, and so ErrInvalidRequest", then `Kind`, `Unwrap` and
  the text, each for the 14 records); `Names` (`:283`, the kind, and `Tool ""`;
  want the name; both); `NeverNamesUnsent` (`:347`, the kind only, 22 lines,
  none on `Tool`: the empty `Tool` holds at the stub, as the PLAN says);
  `Deferral` (`:367`, both); `LastSendKeepsName` (`:412`, both, for xAI, and
  for Anthropic's seventh send through Generate and Stream, which at the stub
  still reached the schema refusal after seven sends, so the sequence is
  built right); `RedactedNeighbour` (`:466`, both); `RedactedName` (`:486`,
  the kind only); `NameBeyondBound` (`:506`, the kind only); `SeveralNamed`
  (`:538`, the kind only; built, since F1 and F4 take a second name by a named
  edit); `NamesNothing` (`:558`, the kind only); `DuplicateNames` (`:574`,
  both); `SearchLoaded` (`:612`, both); `SharedLabel` (`:652`, both);
  `Concurrent` (`:879`, both); `TestDescribe_ToolSchemaRefused_0084`
  (`describe_0084_test.go:18`, "invalid request, HTTP 400", want "tool schema
  refused, HTTP 400"). **Red by a build failure, as the PLAN says of the
  helper:** `TestNameRefusedTool_PassThrough_0084`, `_Wrapped_0084`,
  `_Forms_0084` and `TestNameRefusedStream_0084` (`undefined: Held`,
  `undefined: NameRefusedTool`), and, added in this pass, `TestListed_0084`
  (`deferral`), `TestSearchLoaded_0084` (`responses`) and `TestRead_0084` and
  `TestReferences_0084` (`internal/toolschema`, no such package at the stub).
  **Green at the stub, as guards (`--expect pass`, 11 of 11):** `Reworded`,
  `RecordedFallback`, `OthersUnchanged`, `KeyRowFirst`, `ProxyAltered`,
  `StreamUnchanged`, `Precedence`, and `Kinds` (the sentinel as D1 declares it)
  in `llmprovider`; `FieldsUnchanged`, `NoAdaptation` and `NoLog` in
  `providers`. At the base, `9e84a88`, every one of the 23 red tests does not
  build, in each of the seven packages (`undefined: ErrToolSchemaRefused`,
  `undefined: Held`).
* **Step 3, the tests**, each in a `_0084_test.go` file of its package, read
  from the 0062 fixtures and Phase 1's, by Phase 1c's sort (13 recognised
  records and the 0062 Google record feed `Classify` and `Names`; the four
  unrecognised feed `RecordedFallback`; the three evidence-only records are
  listed so that a new refusal cannot go untested): `llmprovider/
  toolschema_refusal_0084_test.go` (9 tests), `llmprovider/providers/
  toolschema_refusal_0084_test.go` (16), `llmprovider/internal/wire/
  refusedtool_0084_test.go` (4), `wizard/describe_0084_test.go` (1), and three
  small files for the three helpers this phase adds: `internal/wire/deferral/
  listed_0084_test.go`, `internal/wire/responses/searchloaded_0084_test.go`
  and `internal/toolschema/toolschema_0084_test.go`. Each named edit of a
  recorded body is stated in its test's comment.
* **Step 5, the code.** `llmprovider/api_error.go`: `ErrToolSchemaRefused`
  beside `ErrContextOverflow`, `APIError.Tool` with D2's doc, the rows of D3
  after the key, workspace and overflow rows (`classifyInvalid`, below), for
  an HTTP error reply only (`classifyAPIError` takes `reply bool`; the two
  stream call sites pass false). `llmprovider/internal/wire/refusedtool.go`:
  `Held`, `NameRefusedTool` (finds the `*APIError` by `errors.AsType`, sets
  `Tool` only when every name and index the message gives resolves to one
  name the body held, returns the error it was given) and `NameRefusedStream`.
  `deferral.Listed` and `responses.SearchLoaded` give the helper the list as
  sent after deferral. The four providers compute the tools list once per
  request (`deferral.Tools`, threaded through `call` and `body`; OpenCode holds
  it in its per-request `call`), pass it to the encoder and to the helper, in
  `Generate` and `Stream` (OpenCode on every route, Messages included).
  `wizard/describe.go`: the kind before `ErrInvalidRequest`.
* **Deviations and decisions for the coordinator and the owner** (each with
  its evidence; none is worked around):
  1. **ST43's second half cannot be met as written.** The row says a name
     beyond the 512-byte bound "or a body over 64 KiB" gives "the kind;
     `Tool` empty". A body over 64 KiB is read only to 64 KiB
     (`api_error.go`, `io.LimitReader`), so its JSON is cut and does not
     parse; `parseAPIErrorBody` then returns no type, and no D3 row can match
     (each needs a code or type). A probe on the scratch clone: a Grok body of
     70 KiB with the tag first gives `Kind` `ErrInvalidRequest` and an empty
     `Code`. The first half is tested (`NameBeyondBound`); the second has no
     test, since the only test that would pass asserts a different outcome
     than the row. Resolutions: (a) amend ST43 to say a body over 64 KiB keeps
     `ErrInvalidRequest` as today, and pin that with a guard (recommended; no
     code); (b) read the marker from the raw text when the JSON does not parse
     (new code and a new matcher, a change to D3). Doing nothing leaves the row
     false.
  2. **`LastSendKeepsName` as the PLAN words it cannot reach the last send.**
     Grok's ladder is five efforts and then one send with none, so at most six
     sends, and the hook is consulted after each; a name filled in the hook
     would still pass the PLAN's two-send case. The test keeps the PLAN's
     Grok case and adds the last allowed send on Anthropic, where five effort
     refusals, each listing fewer efforts, and one shape refusal make the
     seventh send, `wire.MaxSends`, refused for its schema, through Generate
     and Stream; it asserts seven sends.
  3. **A new internal package, `llmprovider/internal/toolschema`.** The PLAN
     puts the markers in `classifyAPIError` and the naming in `internal/wire`;
     `llmprovider` cannot import `wire`, and `wire` can read only `APIError`'s
     fields, so the four forms' markers would be written twice. One package
     holds `Form`, `Read` and `References`, imported by both, so they cannot
     disagree. It is not exported API (`apidiff` lists the two names only).
     It costs a row in `docs/architecture.md`'s and the standards guide's
     package tables, added, and `tables-check` passes. Resolution if refused:
     duplicate the markers in both packages, with a test that they agree.
  4. **`classifyInvalid`.** Adding the D3 rows to `classifyAPIError` took its
     cyclomatic complexity to 42 against a limit of 40 (`gocyclo`, pre-add).
     The overflow row, the new row and the status-only default moved into
     `classifyInvalid`, called from the table's `default`, in the same order;
     the lint stays as it is. `errcheck` found a blank type assertion in
     `SearchLoaded`; fixed. The first `pre-add-check` failed on those two and
     the second passed, so no step failed twice.
  5. **`NoLog` is a guard**, as the PLAN says (green at the stub): it asserts
     the refusal is returned and no log record holds the tool or the text, not
     the kind. **`Kinds`, `Wrapped`/`Forms`/`NamedStream`, `Listed`,
     `SearchLoaded`, `Read` and `References`, and `Names`' ST72 case are not
     in the PLAN's table**: each pins a helper or a row (D1, ST72) the table
     reaches only through other tests.
  6. **Follow-ups outside the record's cases, not done:** Gemini's refusal
     (single-quoted name), Zen Google's refusal of `additionalProperties`
     before the union, OpenAI's and Zen Responses' O2 holding one bad tool, and
     Zen Messages' having no refusal recorded stay as Phase 1c left them; no
     matcher and no row was added for any.
* **Checks of the tests themselves, on scratch copies, never the tree.** Eight
  single breaks, not Phase 3's plants (which are not run in this phase), each
  caught: Grok's tag changed in `toolschema` (`TestRead_0084`,
  `TestReferences_0084`); `Listed` not blanking the wrapper (`TestListed_0084`);
  `SearchLoaded` finding nothing (`TestSearchLoaded_0084`); a type assertion in
  place of `errors.As` (`Wrapped`); `Tool` set on an error of any kind
  (`PassThrough`); the rows applied in `setStreamKind`'s calls
  (`StreamUnchanged`); the rows before the overflow row (`Precedence`); Claude's
  index mapped through `Request.Tools` (`Deferral`, `NeverNamesUnsent`).
* **The gate**, on this Mac, each status taken before any filter:
  `go test -count=1` on each changed package, exit 0 (`llmprovider`,
  `internal/toolschema`, `internal/wire`, `internal/wire/deferral`,
  `internal/wire/responses`, `providers`, `grok`, `openai`, `claude`,
  `opencode`, `wizard`: each `ok`); `run-heavy -- go test -count=1 ./...`,
  exit 0, 44 packages `ok`, 30 s, run after the last code change; PLAN step 7
  on a scratch clone of the phase's tree, exit 0, the eight packages `ok`, no
  existing test changed (no existing test file is in the porcelain);
  `run-heavy -- make pre-add-check FILES="<the 18 changed .go files>"`, exit
  0, "go-precheck: 18 file(s) clean (gofmt, golangci-lint, go vet, go test,
  govulncheck)"; `make records-check` 0, `docs-check` 0 (`sdkaudit: 0
  violation(s) of R50,R51,R52`), `links-check` 0, `changelog-check` 0,
  `tables-check` 0 (49 packages, 0 problems), `disclosure-check` 0 (0 hits),
  `fixtures-check` 0 (485 files, 134 hits, 0 problems), `markdownlint` 0;
  `make api-check` 0 ("0 incompatible change(s) outside llmprovider/x/": it
  lists incompatible changes only, as the MADR's D7 annotation says, so it
  lists neither name); `tools/apidiff.py 9e84a88 <scratch commit>` on a scratch
  clone with this phase's files copied in, exit 0, "Compatible changes:
  `llmprovider.APIError.Tool`: added; `llmprovider.ErrToolSchemaRefused`:
  added", and nothing else (ST36); `python3 -B tools/claims.py` on both
  records, exit 0 ("85 of 85 pass", "29 of 29 pass"); `tools/recordlint.py
  --changed`, exit 0 ("no rule rose"); `tools/denyscan.py --changed`, exit 0
  ("0 hit(s) in 23 file(s)"). The doc gates were run again after this entry
  was written, with the same results. No `-race` run: this phase's PLAN names
  none.
* **Docs step (D8, ST73, ST74).** `docs/guides/api-standards.md` R24 (`Tool`)
  and R25 (the kind), both citing 0084 D1 and D2, and its package table;
  `docs/architecture.md`'s error paragraph (the kind, the routes each form was
  recorded on, N3 to N5, the fall-backs), its "Errors" bullet, and its package
  rows for `llmprovider`, `internal/wire`, `responses`, `deferral` and the new
  `internal/toolschema`; `CHANGELOG.md` `[Unreleased]` **Added**; `README.md`'s
  "Errors" bullet; the package docs of `llmprovider`, `grok`, `openai`,
  `claude` and `opencode`, and the doc comments of `ErrToolSchemaRefused` and
  `APIError.Tool`; `wizard/describe.go`'s `errorClasses`. ST73's and ST74's
  checks: `grep -c ErrToolSchemaRefused` on each of ten files (the standards
  guide, the architecture, the changelog, the README, `provider.go`, the four
  providers' files and `describe.go`), each exit 0 with a count of 1 or 2,
  and, on a scratch copy of `describe.go` with the name replaced, a count of
  0 and exit 1. Phase 4 reads the text against ST73 and ST74.
* **Not done.** The MADR's provider table and state table are not amended
  (nothing asserted there changed; deviation 1 is put to the owner first).
  Phase 3's plants, `-race`, `make lint` over the whole module, `make vuln`,
  `go vet -tags live_gateways ./...` and CI's set are Phase 3's. No commit.

### Phase 2, build pass 2 (2026-11-21)

Opened on the owner's rulings, relayed by the coordinator, and an independent
review of pass 1 that found no blocker and the gaps below. Records first (this
entry's opening, the MADR's eighth amendment and the annotations it names),
then code, with no stop between. A third pass would need the owner.

* **The owner's rulings.**
  1. **ST43.** "Raise the body limit", then "we need to cap that body read at
     128KiB", then "change this to the recommended 1 MiB". The last stands;
     **128 KiB is superseded**. `apiErrorBodyLimit` (`api_error.go:106`) is
     `1 << 20`, shared by every error path; the cost is up to 1 MiB held while
     each error is classified. ST43's body half now reads as the MADR's eighth
     amendment says, and ST86 is the new row for the limit itself. Deviation 1
     of pass 1 is closed by it: resolution (b) of that entry, in a form that
     reads no raw text.
  2. **The new package.** "Accept the internal package (Recommended)": the
     deviation below.
  3. **The test and the refactor.** "Accept both as recorded (Recommended)":
     the two deviations below.
* **Deviation, dated 2026-11-21: `llmprovider/internal/toolschema` is added to
  Phase 2's scope** (pass 1, deviation 3). `In scope` lists `internal/wire` for
  the helper and `api_error.go` for the rows; neither can hold the markers once
  (`llmprovider` cannot import `wire`; `wire` reads only `APIError`'s fields).
  The package holds `Form`, `Read` and `References`, read by both. It is
  internal: `make api-check` and `tools/apidiff.py` list the two names only.
  Its rows are in the architecture's and the standards guide's package tables.
* **Deviation, dated 2026-11-21: `LastSendKeepsName` gains the Anthropic
  seventh-send case** (pass 1, deviation 2). The PLAN's Grok case cannot reach
  the last allowed send (five rungs and one send with none: six sends, the hook
  consulted after each), so it is kept, and a second case sends seven times on
  Anthropic and names the error of the seventh, through Generate and Stream.
* **Deviation, dated 2026-11-21: `classifyInvalid`** (pass 1, deviation 4).
  Adding the D3 rows took `classifyAPIError` to cyclomatic complexity 42 against
  the lint's 40. The overflow row, the D3 rows and the status-only default are
  `classifyInvalid`, called from the table's `default`, in the same order. The
  lint is unchanged.
* **The review's findings, and what pass 2 does.**
  * **(a) "By service" (N3, ST07) was unpinned.** Reading each marker form on
    every service left every test green. A guard, `TestToolSchemaRefusal_
    OtherServices_0084` (`llmprovider`, green at the stub), serves F1, F2, F4
    and F7 unedited but for the label: each under `gemini`, `kilo`,
    `huggingface`, `ollama`, `together` and a `chatcompat` id; Grok's form under
    `openai`, `claude` and `opencode`; OpenAI's under `grok` and `claude`;
    Google's under `openai`, `grok`, `claude` and `gemini`; Anthropic's under
    `grok`, `openai`, `gemini` and `kilo`. Each asserts today's kind. Plants:
    the four reviewer's survivors (a form read on every service).
  * **(b) D3's "400" was pinned against removal, not widening.** `ProxyAltered`
    gains 404 and 422; plant: the status condition widened to any status that
    maps to `ErrInvalidRequest`.
  * **(c) ST73's docs.** The package docs of `llmprovider`, `grok`, `openai`,
    `claude` and `opencode` say what N3 to N5 decided: a failure inside a 200
    body or a stream is not classified, and `x-should-retry: true` and an
    overflow win. In the changelog, the architecture and the sentinel's doc,
    "stays `ErrInvalidRequest`" for a failure inside a 200 or a stream reads
    "keeps today's kind", as ST17 decides. The architecture's `toolschema` row
    names `Refs` and the `Form` constants.
  * **(d)** `toolschema.go`'s claim that classification and naming "cannot
    disagree" is softened: they share the markers but read different inputs
    (the helper reads `apiErr.Code` and the redacted, bounded `Message`), so it
    can only fall to an empty `Tool`, the safe direction.
* **The limit's tests** (ST43, ST86), red at the stub unless marked a guard:
  `TestClassify_ErrorBodyLimit_0084` (`llmprovider`: an overflow-typed 400 of
  200 KiB is `ErrContextOverflow`; one of exactly 1 MiB parses; one cut by the
  read keeps `ErrInvalidRequest`, a guard in that half); `TestToolSchemaRefusal_
  BodyLimit_0084` (`llmprovider`: a 70 KiB refusal gets the kind; one of
  exactly 1 MiB, the cap, gets the kind; one whose JSON the read cuts keeps
  today's kind, a guard; one over 1 MiB whose object closes early, followed by
  whitespace, gets the kind); `TestToolSchemaRefusal_BigBody_0084`
  (`providers`: through Grok's Generate and Stream, a 70 KiB refusal names its
  tool, and one whose name lies beyond `Message`'s 512 bytes has the kind and
  no `Tool`). Each body is a recorded one with a named edit: padding added in a
  field after the error, or whitespace after the object. The 64 KiB limit's
  other statements, searched for in tests, docs and records (`64 << 10`,
  "64 KiB", `apiErrorBodyLimit`): `api_error.go`'s two comments are updated;
  0012-MADR, 0012-PLAN and 0021-MADR's Z1 are annotated; the file-size limits
  of the token and client stores (0016, 0082 and their tests, `auth`, the
  wizard) are another limit and are not touched; `htmlErrorPage` and its
  benchmark keep 64 KiB as their input, with a dated note in the test's
  comment that the read limit is now larger.
* **Follow-ups, not done now** (seen in review): Claude computes its deferral
  form twice, in `call`'s warnings and in `body` (`claude.go:379`, `:463`);
  `classifyAPIError`'s positional `reply bool` may be better as a named type or
  a field of the envelope. ~~Also, from the re-review of this pass:
  `redactInput`'s 2 KiB clip (0021-MADR Z1) is now the only bound on redaction
  cost for a message of up to 1 MiB, and no test pins it; removing it survives
  every test, since the output is unchanged and only the cost changes. A
  benchmark guard or an allocation assertion would cover it, in a later
  record.~~ *(Struck 2026-11-21, Phase 4: the claim is wrong, and the follow-up
  is dropped. The reviewer's plant removing the clip is caught by
  `TestClassify_16KiBTokensUnder1ms` (`api_error_cut_0028_test.go:18`), which
  asserts that `redactInput` keeps at most 2 KiB.)*
* **Correction, dated 2026-11-21, after the re-review:** the MADR's ST86 row
  said an overflow-typed 400 of 200 KiB is `ErrContextOverflow`, but
  `TestClassify_ErrorBodyLimit_0084` checks an OpenAI quota 429 at 200 KiB; the
  row now says that, `ErrQuotaExhausted`, in place.

#### Outcome of build pass 2 (2026-11-21)

Built on `9e84a88`, after the records above. No live run, no
`LLMPROVIDER_LIVE_*` variable, nothing staged or committed.

* **The limit.** `apiErrorBodyLimit` is `1 << 20`; its comment and
  `apiErrorRedactLimit`'s say so. The search for the old limit, in tests, docs
  and records, found: the two comments (updated); 0012-MADR, 0012-PLAN and
  0021-MADR's Z1 (annotated, dated); `htmlErrorPage` (a dated note: 64 KiB is
  the benchmark's input); the file-size limits of `auth`, the wizard and
  0016, 0081 and 0082's records (another limit, left alone); and one test that
  stated the limit without saying so, `TestLookupModel_Bounds_0033`, whose
  "100 KiB error body" had been cut by the 64 KiB read and now parses. **An
  existing test changed**, by the owner's ruling and not by choice: the
  original bound, at most 512 bytes of error text, is kept for a 2 MiB body,
  which the new read cuts, and a 100 KiB body, now parsed, is held to the
  `APIError`'s 512-byte `Message` and 1 KiB of text. A scratch copy with
  `apiErrorMessageLimit` at 4096 turns the test red (2,123 bytes). The full
  suite found no other test that depended on the old limit. *(Confirmed
  2026-11-21, Phase 4: the owner's answer "Confirm the change (Recommended)",
  relayed by the coordinator; the change is covered by the 1 MiB ruling.)*
* **The new tests.** `TestToolSchemaRefusal_OtherServices_0084` (a guard),
  `TestToolSchemaRefusal_BodyLimit_0084`, `TestClassify_ErrorBodyLimit_0084`
  (`llmprovider`, in `toolschema_services_0084_test.go`),
  `TestToolSchemaRefusal_BigBody_0084` (`providers`), and 404 and 422 in
  `ProxyAltered`. **At the stub:** `OtherServices` and `ProxyAltered` green
  (12 of 12 guards pass); `BodyLimit` red by its assertion ("70 KiB: Kind
  llmprovider: invalid request; want ErrToolSchemaRefused"),
  `ErrorBodyLimit` red ("200 KiB: llmprovider: rate limited: openai HTTP 429",
  want quota exhausted: the old read cut it), `BigBody` red ("70 KiB through
  Generate: Tool \"\"; want \"record_city\"").
* **Plants, on scratch copies of this tree, never the tree: all 13 caught.**
  Pass 2's own: the limit back at 64 KiB (`BodyLimit`); at the superseded
  128 KiB (`BodyLimit`, the 1 MiB cases); at 2 MiB (`BodyLimit`, the
  one-byte-over cut case; `ErrorBodyLimit`); the helper falling back to the
  last tool (`BigBody`, the name past the bound); an OpenAI, Grok, Google, or
  Anthropic form read on every service, and Anthropic's on every service but
  OpenCode (`OtherServices`, each under `gemini`, `kilo` and the others); the
  row's status widened to any status that maps to `ErrInvalidRequest`, to 404
  only and to 422 only (`ProxyAltered`). **The reviewer's `plant_runner.py`,
  unchanged, with its `spec.json` and `spec2.json`, against a scratch copy of
  this tree: 25 of 25 caught**, including the four survivors of pass 1 (the
  OpenAI, Grok and Google forms on every service, and the 400 widened) and the
  fifth, Anthropic's form on every service but OpenCode (`spec2.json`).
* **The review's other findings.** (c): the package docs of `llmprovider`,
  `grok`, `openai`, `claude` and `opencode` now say that a failure inside a
  200 body or a stream is not classified and keeps the kind it had, and that
  `x-should-retry: true` and an overflow win; the changelog, the architecture
  and the sentinel's doc read "keeps today's kind" for such a failure, and the
  changelog and the architecture state the 1 MiB read; the architecture's
  `toolschema` row names `Refs` and the `Form` constants. (d): `toolschema.go`'s
  comment now says the two share markers and read different inputs, so the
  helper can only fall to an empty `Tool`.
* **The gate**, on this Mac, each status before any filter: `go test -count=1`
  on each changed package, exit 0 (the eleven of pass 1, and `catalog`);
  `run-heavy -- go test -count=1 ./...`, exit 0, 44 packages `ok`, 29 s, after
  the last code change (an earlier run of this pass failed once, on
  `TestLookupModel_Bounds_0033`, above); `run-heavy -- make pre-add-check
  FILES="<the 22 changed .go files>"`, exit 0, "go-precheck: 22 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `make api-check`, exit
  0 ("0 incompatible change(s)"); `tools/apidiff.py 9e84a88 <scratch commit>`,
  exit 0, "Compatible changes: `APIError.Tool`: added; `ErrToolSchemaRefused`:
  added", nothing else. The `make` doc gates, claims, recordlint and denyscan
  are in the hand-back, run after this entry was written.

### Phase 3, the plants and the gates (2026-11-21)

Run by an agent at `6602dcd` (Phase 2's commit), on `main`, tree clean at the
start. Nothing is staged or committed; no live run was made, no
`LLMPROVIDER_LIVE_*` variable was set, no key was read. The MADR is unchanged.
Steps 0 to 7 ran; the base red's exit 1 is the dated deviation below.

* **Step 0, the re-grounding.** The PLAN's nine plants, the amendment of
  2026-11-20's added plants, and the plants Phase 2's passes ran (including the
  independent reviewer's `spec.json`, `spec2.json` and `spec3.json`, read, not
  changed) make 70 single-edit plants, plus one `apidiff` plant. Each plant's
  old text occurs exactly once at `6602dcd`; the line each replaces is in the
  table. The plants are in `<scratch>/specs/` as one `tools/plant.py` spec per
  test selection (38 specs), built by `<scratch>/make_plants.py`, which stops
  if any old text does not occur once. A plant with two edits in the reviewer's
  runner (ST40) is one contiguous span here.
* **Step 1, the plants,** `python3 -B tools/plant.py` over each spec on a scratch
  clone (never the tree), each named test run alone. Every build was checked:
  no plant was caught by a build failure. Each failure line is the test's, the
  first assertion line printed.

| Plant | Replaces | Catching test(s), as run | First failure line |
| :--- | :--- | :--- | :--- |
| row removed: Grok | `llmprovider/api_error.go:668` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:141: grok-root-string/grok: llmprovider: invalid request: grok HTTP 400 invalid-argument: Failed to start sampling: [i` |
| row removed: OpenAI | `llmprovider/api_error.go:671` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:141: openai-root-string/openai: llmprovider: invalid request: openai HTTP 400 invalid_function_parameters: Invalid sch` |
| row removed: Anthropic | `llmprovider/api_error.go:672` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:141: claude-root-string/claude: llmprovider: invalid request: claude HTTP 400 invalid_request_error: tools.1.custom.in` |
| row removed: Google route | `llmprovider/api_error.go:674` | `TestToolSchemaRefusal_Classify_0084`, `TestToolSchemaRefusal_Names_0084` | `toolschema_refusal_0084_test.go:141: opencode-zen-google-string-schema-tool/opencode: llmprovider: invalid request: opencode HTTP 400 INVALID_ARGUMENT` |
| ST30: ClassifyHTTPError wraps the kind with Kind left ErrInvalidRequest | `llmprovider/api_error.go:355` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:145: grok-root-string/grok: Kind llmprovider: invalid request, Status 400; want ErrToolSchemaRefused, 400` |
| loosened: Grok to any invalid-argument 400 (ST22, ST60) | `llmprovider/internal/toolschema/toolschema.go:78` | `TestToolSchemaRefusal_Reworded_0084`, `TestToolSchemaRefusal_OthersUnchanged_0084` | `toolschema_refusal_0084_test.go:207: grok: the tag renamed (grok): llmprovider: tool schema refused: grok HTTP 400 invalid-argument: Failed to start s` |
| loosened: OpenAI keyed on the type invalid_request_error (ST24) | `llmprovider/internal/toolschema/toolschema.go:76` | `TestToolSchemaRefusal_Reworded_0084`, `TestToolSchemaRefusal_OthersUnchanged_0084` | `toolschema_refusal_0084_test.go:207: openai: the code changed to invalid_prompt (openai): llmprovider: tool schema refused: openai HTTP 400 invalid_pr` |
| loosened: Anthropic keyed on its type alone (ST25, ST27) | `llmprovider/internal/toolschema/toolschema.go:82` | `TestToolSchemaRefusal_Reworded_0084`, `TestToolSchemaRefusal_OthersUnchanged_0084` | `toolschema_refusal_0084_test.go:207: claude: the tools.N. path renamed (claude): llmprovider: tool schema refused: claude HTTP 400 invalid_request_err` |
| loosened: Google to its status alone (ST60) | `llmprovider/internal/toolschema/toolschema.go:80` | `TestToolSchemaRefusal_Reworded_0084`, `TestToolSchemaRefusal_RecordedFallback_0084`, `TestToolSchemaRefusal_OthersUnchanged_0084` | `toolschema_refusal_0084_test.go:207: zen google: the phrase changed (opencode): llmprovider: tool schema refused: opencode HTTP 400 INVALID_ARGUMENT:` |
| name taken from the reply's text without the sent list | `llmprovider/internal/wire/refusedtool.go:64` | `TestToolSchemaRefusal_NeverNamesUnsent_0084` | `toolschema_refusal_0084_test.go:347: grok: a name the request did not send through Generate: Tool "ghost_tool"; want ""` |
| index mapped through Request.Tools (Claude Generate) | `llmprovider/providers/claude/claude.go:305` | `TestToolSchemaRefusal_Deferral_0084` | `toolschema_refusal_0084_test.go:367: claude: Tool "deferred_tool"; want "record_city"` |
| index mapped through Request.Tools (Claude Stream) | `llmprovider/providers/claude/claude.go:354` | `TestToolSchemaRefusal_Deferral_0084` | `toolschema_refusal_0084_test.go:383: claude-opus-4-1 through Stream: Tool "deferred_tool"; want "record_city"` |
| ST39: Claude name filled in the resend hook only | `llmprovider/providers/claude/claude.go:302` | `TestToolSchemaRefusal_LastSendKeepsName_0084` | `toolschema_refusal_0084_test.go:445: claude on the last send: Tool ""; want "record_city"` |
| ST29: kind not beneath ErrInvalidRequest | `llmprovider/api_error.go:68` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:155: grok-root-string/grok: Unwrap [llmprovider: tool schema refused llmprovider: invalid request]; want the kind alon` |
| Error() prints Tool (ST31) | `llmprovider/api_error.go:200` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:167: grok-root-string/grok: Error() changed from "llmprovider: tool schema refused: grok HTTP 400 invalid-argument: Fa` |
| ST37: shouldRetry set for the kind | `llmprovider/api_error.go:314` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:149: grok-root-string/grok: retryable true, or a rate limit or unavailability; want a terminal invalid request` |
| ST37: parent made ErrRateLimited | `llmprovider/api_error.go:68` | `TestToolSchemaRefusal_Classify_0084` | `toolschema_refusal_0084_test.go:149: grok-root-string/grok: retryable false, or a rate limit or unavailability; want a terminal invalid request` |
| kind missing from errorClasses (ST35) | `wizard/describe.go:24` | `TestDescribe_ToolSchemaRefused_0084` | `describe_0084_test.go:18: describe(ErrToolSchemaRefused, HTTP 400) = "invalid request, HTTP 400"; want "tool schema refused, HTTP 400"` |
| ST12: helper takes the first of several names | `llmprovider/internal/wire/refusedtool.go:64` | `TestToolSchemaRefusal_SeveralNamed_0084` | `toolschema_refusal_0084_test.go:538: grok-root-string through Generate: Tool "record_city"; want ""` |
| ST13: helper requires exactly one entry, not one name | `llmprovider/internal/wire/refusedtool.go:30` | `TestToolSchemaRefusal_DuplicateNames_0084` | `toolschema_refusal_0084_test.go:574: grok through Generate: Tool ""; want "record_city"` |
| ST11/ST42/ST43: helper falls back to the last caller tool when it finds none | `llmprovider/internal/wire/refusedtool.go:77` | `TestToolSchemaRefusal_NameBeyondBound_0084` | `toolschema_refusal_0084_test.go:508: Generate: Tool "tool_nnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn` |
| ST28/ST44: helper appends the tool's name to Message | `llmprovider/internal/wire/refusedtool.go:53` | `TestToolSchemaRefusal_FieldsUnchanged_0084` | `toolschema_refusal_0084_test.go:693: grok-string-schema-tool through Generate: llmprovider: tool schema refused: grok HTTP 400 invalid-argument: Faile` |
| ST10: helper ignores Inside (tool_search_output definitions) | `llmprovider/internal/wire/refusedtool.go:30` | `TestToolSchemaRefusal_SearchLoaded_0084` | `toolschema_refusal_0084_test.go:612: openai: the tool the search loaded: Tool ""; want "loaded_tool"` |
| ST08: OpenCode Messages route Generate not named | `llmprovider/providers/opencode/opencode.go:358` | `TestToolSchemaRefusal_SharedLabel_0084` | `toolschema_refusal_0084_test.go:652: go messages route through Generate: Tool ""; want "record_city"` |
| ST45: Grok logs the refusal at Debug | `llmprovider/providers/grok/grok.go:273` | `TestToolSchemaRefusal_NoLog_0084` | `toolschema_refusal_0084_test.go:820: grok-string-schema-tool: a log record holds the refusal or the tool: "llmprovider: grok: refused error=llmprovide` |
| ST21: D3 row placed first in classifyAPIError (above key row) | `llmprovider/api_error.go:582` | `TestToolSchemaRefusal_KeyRowFirst_0084` | `toolschema_refusal_0084_test.go:334: the tag with an API key named: llmprovider: tool schema refused: grok HTTP 400 invalid-argument: Failed to start` |
| D3: row's status widened from 400 to any 4xx mapped to ErrInvalidRequest (404, 422...) | `llmprovider/api_error.go:652` | `TestToolSchemaRefusal_ProxyAltered_0084` | `toolschema_refusal_0084_test.go:367: grok-string-schema-tool (grok) served as 404: llmprovider: tool schema refused: grok HTTP 404 invalid-argument: F` |
| ST20/ST23/ST50: row's status condition dropped | `llmprovider/api_error.go:652` | `TestToolSchemaRefusal_ProxyAltered_0084` | `toolschema_refusal_0084_test.go:367: grok-string-schema-tool (grok) served as 401: llmprovider: tool schema refused: grok HTTP 401 invalid-argument: F` |
| ST16: OpenAI Stream not named | `llmprovider/providers/openai/openai.go:369` | `TestToolSchemaRefusal_Names_0084` | `toolschema_refusal_0084_test.go:284: openai-root-string through Stream: Tool ""; want "record_city"` |
| ST16: Claude Stream not named | `llmprovider/providers/claude/claude.go:354` | `TestToolSchemaRefusal_Names_0084` | `toolschema_refusal_0084_test.go:284: claude-root-string through Stream: Tool ""; want "record_city"` |
| ST16: Grok Stream not named | `llmprovider/providers/grok/grok.go:323` | `TestToolSchemaRefusal_Names_0084` | `toolschema_refusal_0084_test.go:284: grok-root-string through Stream: Tool ""; want "record_city"` |
| ST10: Grok Stream Listed from Request.Tools (Dispatch: deferred and use_tool) | `llmprovider/providers/grok/grok.go:323` | `TestToolSchemaRefusal_NeverNamesUnsent_0084` | `toolschema_refusal_0084_test.go:347: grok: a deferred tool that was not sent through Stream: Tool "deferred_tool"; want ""` |
| ST10: OpenAI Listed from Request.Tools | `llmprovider/providers/openai/openai.go:345` | `TestToolSchemaRefusal_NeverNamesUnsent_0084`, `TestToolSchemaRefusal_SearchLoaded_0084` | `toolschema_refusal_0084_test.go:347: openai: a deferred tool that was not sent through Generate: Tool "deferred_tool"; want ""` |
| ST10: OpenCode Listed from Request.Tools | `llmprovider/providers/opencode/opencode.go:470` | `TestToolSchemaRefusal_SearchLoaded_0084` | `toolschema_refusal_0084_test.go:612: zen-responses: a deferred tool not loaded: Tool "unloaded_tool"; want ""` |
| ST08: OpenCode non-Messages Stream not named | `llmprovider/providers/opencode/opencode.go:421` | `TestToolSchemaRefusal_Names_0084`, `TestToolSchemaRefusal_SharedLabel_0084` | `toolschema_refusal_0084_test.go:284: zen-responses-root-string through Stream: Tool ""; want "record_city"` |
| ST17: D3 rows applied in setStreamKind's status call | `llmprovider/api_error.go:479` | `TestToolSchemaRefusal_StreamUnchanged_0084` | `toolschema_refusal_0084_test.go:418: opencode-zen-google-string-schema-tool as a stream failure with code 400 (opencode-zen/google): llmprovider: tool` |
| ST26: D3 row placed before the overflow row | `llmprovider/api_error.go:640` | `TestToolSchemaRefusal_Precedence_0084` | `toolschema_refusal_0084_test.go:449: a refusal that also says the prompt is too long: llmprovider: tool schema refused: claude HTTP 400 invalid_reques` |
| ST32: x-should-retry rule skips the new kind | `llmprovider/api_error.go:351` | `TestToolSchemaRefusal_Precedence_0084` | `toolschema_refusal_0084_test.go:437: grok-string-schema-tool (grok) with x-should-retry: true: llmprovider: tool schema refused: grok HTTP 400 invalid` |
| ST33: helper type-asserts *APIError instead of errors.As | `llmprovider/internal/wire/refusedtool.go:49` | `TestNameRefusedTool_Wrapped_0084` | `refusedtool_0084_test.go:96: Join: Tool ""; want record_city, set on the *APIError found` |
| ST34: helper sets Tool on an *APIError of any kind | `llmprovider/internal/wire/refusedtool.go:50` | `TestNameRefusedTool_PassThrough_0084` | `refusedtool_0084_test.go:73: kind llmprovider: invalid request: Tool set to "record_city" on an error that is not the kind` |
| ST41: Grok parse requires the word tool after the colon | `llmprovider/internal/toolschema/toolschema.go:110` | `TestToolSchemaRefusal_RedactedNeighbour_0084` | `toolschema_refusal_0084_test.go:466: through Generate: Tool ""; want "lookup_api_key"` |
| ST86: limit back to 64 KiB | `llmprovider/api_error.go:113` | `TestToolSchemaRefusal_BodyLimit_0084`, `TestClassify_ErrorBodyLimit_0084` | `toolschema_services_0084_test.go:92: grok-string-schema-tool (grok), 70 KiB: Kind llmprovider: invalid request; want ErrToolSchemaRefused` |
| ST86: limit one byte short of 1 MiB | `llmprovider/api_error.go:113` | `TestToolSchemaRefusal_BodyLimit_0084`, `TestClassify_ErrorBodyLimit_0084`, `TestToolSchemaRefusal_BigBody_0084` | `toolschema_services_0084_test.go:92: grok-string-schema-tool (grok), exactly 1 MiB, the cap: Kind llmprovider: invalid request; want ErrToolSchemaRefu` |
| ST86: limit 2 MiB | `llmprovider/api_error.go:113` | `TestToolSchemaRefusal_BodyLimit_0084`, `TestClassify_ErrorBodyLimit_0084` | `toolschema_services_0084_test.go:94: grok-string-schema-tool (grok), one byte over the cap, the object cut: Kind llmprovider: tool schema refused; wan` |
| ST86: no limit (whole body read) | `llmprovider/api_error.go:294` | `TestToolSchemaRefusal_BodyLimit_0084`, `TestClassify_ErrorBodyLimit_0084` | `toolschema_services_0084_test.go:94: grok-string-schema-tool (grok), one byte over the cap, the object cut: Kind llmprovider: tool schema refused; wan` |
| ST43: helper falls back to the last tool (BigBody name past the bound) | `llmprovider/internal/wire/refusedtool.go:77` | `TestToolSchemaRefusal_BigBody_0084` | `toolschema_bigbody_0084_test.go:44: the name past the 512 bytes Message keeps through Generate: Tool "record_city"; want ""` |
| N3/ST07: OpenAI form read on every service (Kilo, HF, Together, Gemini...) | `llmprovider/api_error.go:671` | `TestToolSchemaRefusal_OtherServices_0084` | `toolschema_services_0084_test.go:42: openai's form under gemini: llmprovider: tool schema refused: gemini HTTP 400 invalid_function_parameters: Invali` |
| N3/ST07: Grok form read on every service | `llmprovider/api_error.go:668` | `TestToolSchemaRefusal_OtherServices_0084` | `toolschema_services_0084_test.go:42: grok's form under gemini: llmprovider: tool schema refused: gemini HTTP 400 invalid-argument: Failed to start sam` |
| N3/ST07: Google form read on every service | `llmprovider/api_error.go:674` | `TestToolSchemaRefusal_OtherServices_0084` | `toolschema_services_0084_test.go:42: google's form under gemini: llmprovider: tool schema refused: gemini HTTP 400 INVALID_ARGUMENT: Upstream request` |
| N3: Anthropic form read on every service but OpenCode | `llmprovider/api_error.go:672` | `TestToolSchemaRefusal_OtherServices_0084` | `toolschema_services_0084_test.go:42: anthropic's form under gemini: llmprovider: tool schema refused: gemini HTTP 400 invalid_request_error: tools.0.c` |
| N3: Anthropic form read on every service | `llmprovider/api_error.go:672` | `TestToolSchemaRefusal_OtherServices_0084` | `toolschema_services_0084_test.go:42: anthropic's form under gemini: llmprovider: tool schema refused: gemini HTTP 400 invalid_request_error: tools.0.c` |
| recorder: redaction skipped (ST46) | `llmprovider/toolschema_0084_record_test.go:529` | `TestToolSchemaRecord_Redacts_0084` | `toolschema_0084_record_offline_test.go:224: claude-root-string.json holds a planted value (52 bytes, starting "xai-")` |
| recorder: Message and Code left unredacted, body still redacted (ST46) | `llmprovider/toolschema_0084_record_test.go:436` | `TestToolSchemaRecord_Redacts_0084` | `toolschema_0084_record_offline_test.go:224: claude-root-string.json holds a planted value (36 bytes, starting "3f2b")` |
| recorder: body written before redaction (ST80) | `llmprovider/toolschema_0084_record_test.go:439` | `TestToolSchemaRecord_Redacts_0084` | `toolschema_0084_record_offline_test.go:224: claude-root-string.json holds a planted value (52 bytes, starting "xai-")` |
| recorder: a 404 read as refused (ST67) | `llmprovider/toolschema_0084_record_test.go:344` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:345: 404: refused, ""; want unreachable, ""` |
| recorder: a Status-0 ErrIncomplete read as other (ST67) | `llmprovider/toolschema_0084_record_test.go:337` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:345: incomplete 200: other, ""; want accepted, ""` |
| recorder: the writer replacing a refused record (ST70) | `llmprovider/toolschema_0084_record_test.go:569` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:417: claude-root-string: wrote unreachable over the record; it holds unreachable; want refused` |
| recorder: a blank key read as set (ST78) | `llmprovider/toolschema_0084_record_test.go:453` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:368: blank key: false, , ""; want skipped, "key unset or blank"` |
| recorder: errors.Is(err, ErrInvalidRequest) in place of the kind (ST67) | `llmprovider/toolschema_0084_record_test.go:345` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:345: xai 400 key refusal: refused, ""; want unreachable, ""` |
| recorder: the kind condition dropped from the refused row (ST67) | `llmprovider/toolschema_0084_record_test.go:344` | `TestToolSchemaRecord_Outcome_0084` | `toolschema_0084_record_offline_test.go:345: xai 400 key refusal: refused, ""; want unreachable, ""` |
| recorder: the case line also prints Message (ST47) | `llmprovider/toolschema_0084_record_test.go:465` | `TestToolSchemaRecord_LogLine_0084` | `toolschema_0084_record_offline_test.go:263: printed "claude-root-string claude claude-sonnet-5: status 400, refused tools.0.custom.input_schema.type:` |
| recorder: the case line also prints the key variable's value (ST77) | `llmprovider/toolschema_0084_record_test.go:465` | `TestToolSchemaRecord_NoLeak_0084` | `toolschema_0084_record_offline_test.go:456: output holds a made-up value (starting "sk-p")` |
| recorder: the writer writes a record compact (ST83) | `llmprovider/toolschema_0084_record_test.go:565` | `TestToolSchemaRecord_FieldPerLine_0084` | `toolschema_0084_record_offline_test.go:484: 1 lines, first "{  \"case\": \"claude-O2-two-bad\",  \"route\": \"claude\",  \"wire\": \"messages\",  \"pr` |
| recorder: println of the message (ST71, PrintCalls) | `llmprovider/toolschema_0084_record_test.go:465` | `TestToolSchemaRecord_CallAllowlist_0084` | `toolschema_0084_record_offline_test.go:1330: the checked files break the allowlist:` |
| recorder: emit given the message (ST71, PrintCalls) | `llmprovider/toolschema_0084_record_test.go:465` | `TestToolSchemaRecord_CallAllowlist_0084` | `toolschema_0084_record_offline_test.go:1330: the checked files break the allowlist:` |
| scan compares Redact with itself (ST85) | `llmprovider/toolschema_0084_scan_test.go:82` | `TestToolSchemaScan_Plant_0084` | `toolschema_0084_record_offline_test.go:1461: records copy: reported map[] (ok true); want map[3 key shape:true 4 key shape:true 5 key shape:true 6 key` |
| scan's allowlist inverted (ST85) | `llmprovider/toolschema_0084_scan_test.go:85` | `TestToolSchemaScan_Plant_0084` | `toolschema_0084_record_offline_test.go:1430: an output file of allowed lines: 16 hits, ok true; want none` |
| ST40: the sent list kept in a package-level variable (Grok) | `llmprovider/providers/grok/grok.go:249` | `TestToolSchemaRefusal_Concurrent_0084` | `toolschema_refusal_0084_test.go:880: the second call: Tool ""; want "tool_two"` |
| ST38: the provider resends once after naming the tool (Grok) | `llmprovider/providers/grok/grok.go:261` | `TestToolSchemaRefusal_NoAdaptation_0084` | `toolschema_refusal_0084_test.go:740: grok-root-string: 2 requests reached the server; want one, no resend` |

* **Plants whose catching test needed a note.**
  * *Row removed (Grok, OpenAI, Anthropic):* `plant.py` prints twelve lines, and
    `Classify` fills them. Run again with `Names` alone (`<scratch>/specs/sub1.json`,
    exit 0, 4 of 4 caught), each fails `TestToolSchemaRefusal_Names_0084`:
    `grok-root-string through Generate: Tool ""; want "record_city"` (and the
    same for the OpenAI and Anthropic rows, and for ST30's wrapped kind).
  * *ST11, ST42, ST43, "the helper, finding no sent name, falls back to the last
    caller tool".* The reviewer's variant, which falls back only when no name
    was read, is **not** caught by `NamesNothing` or `RedactedName` (each run alone,
    exit 0: `NOT CAUGHT`), because in those cases the helper has already
    returned `""` from inside its loop on a name it cannot resolve, so the fallback
    after the loops never runs; it is caught by `NameBeyondBound` (and `BigBody`).
    The plant as the PLAN words it, the fallback applied whenever the helper
    resolves no name, is caught by each of the three: `NamesNothing`
    (`grok-string-schema-tool through Generate: Tool "record_city"; want ""`),
    `RedactedName` (`Generate: Tool "key_aaaaaaaaaaaa"; want ""`) and
    `NameBeyondBound` (`Generate: Tool "tool_nnnn..."; want ""`), runs `sub4`,
    `sub5` and `sub6`, each exit 0. Recorded as a fact for Phase 4: the reviewer's variant
    survives two of the three tests the map names.
  * *ST10, OpenCode's list from `Request.Tools`:* caught by `SearchLoaded`
    (`zen-responses: a deferred tool not loaded: Tool "unloaded_tool"; want ""`),
    not by `Deferral`, `NeverNamesUnsent` or `SharedLabel`, which the map does not
    name for this provider.
  * *ST17, `setStreamKind`'s second call (status 500, `reply` true):* **not caught,
    and cannot be**: the call's status is `http.StatusInternalServerError`, and
    the row needs status 400, so `reply` true changes nothing. It is an
    equivalent mutant. The first call (the status `code` carries) is caught by
    `StreamUnchanged`. No row is unpinned by it.
  * *ST03, the name taken from `param`'s index:* no plant was written. The helper
    never receives `param` (MADR H5, N2), so no one-line production change reads
    it; the test `NeverNamesUnsent`'s `param` subcase holds the row.
  * *ST36, a third exported name:* a plant adding `func ThirdName0084() {}` to
    `api_error.go`, run with `command` set to a check that `tools/apidiff.py
    9e84a88` lists exactly two additions (exit 0 on the tree: the two names;
    caught, exit 1, with the third).
* **Step 2, red and green.**
  * `python3 -B tools/redgreen.py <scratch>/RED_base.json --expect fail --rev 9e84a88`:
    **exit 1**, `redgreen: 0 of 26 fail as expected`; the nine test files copied
    in, every package `did not build`, first lines
    `llmprovider/toolschema_refusal_0084_test.go:140:23: undefined: ErrToolSchemaRefused`
    and `apiErr.Tool undefined (type *APIError has no field or method Tool)`.
    This is the PLAN's "a build failure, quoted, since the names do not exist"
    (the PLAN's Rules for every phase), but the tool counts a build failure as a
    finding and exits 1, so the command's exit is not 0.
  * `python3 -B tools/redgreen.py <scratch>/RED_tree.json --expect pass`, exit 0,
    `26 of 26 pass as expected`; with the 12 guards added (`ALL.json`), exit 0,
    `38 of 38 pass as expected`.
* **Deviation, dated 2026-11-21 (the coordinator's resolution, nothing waived).**
  The base run `python3 -B tools/redgreen.py <scratch>/RED_base.json --expect
  fail --rev 9e84a88` exits 1 by build failure: `redgreen: 0 of 26 fail as
  expected`, each package "did not build", first lines
  `llmprovider/toolschema_refusal_0084_test.go:140:23: undefined:
  ErrToolSchemaRefused` and `apiErr.Tool undefined (type *APIError has no
  field or method Tool)`. The PLAN anticipates this ("a build failure, quoted,
  since the names do not exist"); the assertion-level red is Phase 2's stub
  runs, build passes 1 and 2 above, where every test was red by its assertion.
  So the exit 1 is the expected red, and the acceptance's "every command exit
  0" does not apply to this one base run. No test is loosened and no code
  changes.
* **Step 3.** `run-heavy -- go test -race -count=3 -timeout 45m ./llmprovider/...
  ./wizard/...`: exit 0, 36 packages `ok`, no `FAIL`; the last line
  `ok  github.com/maccavelli/go-llmprovider-sdk/wizard  67.022s`; run-heavy
  `status=0 elapsed=147s`.
* **Step 4.** `run-heavy -- python3 -B tools/gate.py --timeout 5400`: exit 0,
  "gate: 30 of 30 passed", run-heavy `status=0 elapsed=649s`.
* **Step 5.** `run-heavy -- make lint`: exit 0, "0 issues."; `run-heavy -- make
  vuln`: exit 0, "No vulnerabilities found."; `go vet -tags live_gateways
  ./...`: exit 0, no output.
* **Step 6,** each target run alone, exit taken first: `parity-check` 0
  ("409 identifiers ... 0 problem(s)"); `dep-check` 0 ("46 packages, 0
  problem(s)"); `coverage-check` 0 ("45 packages, 0 problem(s)"); `api-check` 0
  ("against v1.31.0, 0 incompatible change(s) outside llmprovider/x/");
  `contract-check` 0 ("0 problem(s)"); `generate-check` 0 ("3 generated file(s),
  0 problem(s)"); `records-check` 0 ("182 records, 0 problem(s)");
  `tables-check` 0 ("49 packages, 0 problem(s)"); `fixtures-check` 0 ("485
  file(s), 134 hit(s), 0 problem(s)"); `changelog-check` 0 ("45 release
  section(s), 0 problem(s)"); `docs-check` 0 ("sdkaudit: 0 violation(s) of
  R50,R51,R52"); `markdownlint` 0 ("0 issues"); `links-check` 0 ("0
  problem(s), 1428 relative link(s)"); `disclosure-check` 0 ("0 hit(s) in 1773
  file(s)"); `gate-selftest` 0 ("OK"); `make tools-test` 0 ("OK"). No docs gap
  was found by any gate.
* **Step 7.** `python3 -B tools/claims.py` on the MADR and this PLAN: exit 0,
  "85 of 85 pass" and "29 of 29 pass"; `tools/denyscan.py --changed`: exit 0,
  "0 hit(s) in 1 file(s)"; `tools/recordlint.py --changed`: exit 0, "no rule
  rose".
* **Findings kept for Phase 4.** (1) The reviewer's weaker fallback variant
  (fall back to the last tool only when no name was read) survives
  `NamesNothing` and `RedactedName`, each run alone; only `NameBeyondBound`
  and `BigBody` catch it. The PLAN's wording is caught by all three. (2)
  OpenCode's list from `Request.Tools` is caught only by `SearchLoaded`. (3)
  Two equivalent mutants: `setStreamKind`'s second call given `reply` true
  (status 500; the row needs 400), and ST03's name from `param` (the helper
  never receives `param`, so no plant exists). (4) `tools/plant.py` prints at
  most twelve failure lines, so a second named test can be hidden by a
  noisy first one; the row-removed plants were rerun with `Names` alone to
  show it.

### Phase 4, the independent review and its fixes (2026-11-21)

At `b4ad41f`, Phases 2 and 3 committed, tree clean. The reviewer's files are in
the session's scratch folder (`p4-0084/`); none is in the tree. Records first
(the MADR's ninth amendment, this entry), then code, each fix's test seen red
before the fix. The owner's answers, relayed by the coordinator: B1, "Set-based
check + cap the lists (Recommended)"; `TestLookupModel_Bounds_0033`, "Confirm
the change (Recommended)".

* **B1, the blocker: a quadratic duplicate check, and lists with no cap.**
  `api_error.go:785` (`!env.hasType(v)`, reached from `:829`) scans the list
  it builds for each reason, so at the 1 MiB read a body of distinct
  `details[].reason` values takes about 3.15 s to classify (22.6 ms at 60 KiB);
  `api_limit.go:154` appends `Quotas` with no cap (8,134 entries from 1 MiB).
  ST86's stated cost covered memory only. **Done:** a set in
  `parseAPIErrorBody`; at most 32 `details` entries and at most 32 quota
  violations read; MADR ST87, with ST86 annotated. **Test, red first:**
  `TestClassify_ManyDetails_0084`: a 1 MiB Google-shaped body of distinct
  reasons classifies in under 100 ms and a body of distinct violations keeps at
  most 32 `Quota` entries, and the entries kept on the error are asserted
  apart (so removing a cap shows). **Plants:** the scanned list restored; each
  cap removed.
* **Fix 2, ST11, ST42 and ST43 unpinned.** `refusedtool.go`'s `return found` is
  tested for a message that gives no name only with the name emptied in its
  slot; the reviewer's plants A1, A2 and A4 (a marker seen with no name read
  returns the last tool, or the only tool, or the last tool on OpenAI's form)
  survived every 0084 test. **Done:** the reviewer's draft is
  `TestToolSchemaRefusal_NamesNothingNoSlot_0084` among the providers' 0084
  tests (F2 with `function 'record_city'` replaced by `a function`, F7 with
  the backticked name cut, on openai, zen-responses and zen-google, through
  Generate and Stream, with two tools sent and with one; `Tool` empty), and a
  matching case in `TestNameRefusedTool_Forms_0084`. A1 to A4 are run again.
* **Fix 3, ST10 on OpenCode's other routes.** B2, the held list taken from
  `req.Tools` on the routes other than Responses, was caught by no test.
  **Done:** a zen-google case with an unsent deferred tool in
  `NeverNamesUnsent`.
* **Fix 4, two docs nits.** `opencode.go`'s package doc said the Google form is
  read "on the google route", but the code reads it on every OpenCode route (as
  the MADR decides); the doc now matches. The changelog gains a **Changed**
  entry: an error reply is read to 1 MiB for every provider, so other kinds can
  change too (ST86's own example, a 200 KiB quota 429 is now
  `ErrQuotaExhausted`), and the two caps of B1.
* **Fix 5, records.** (D1) The execution record's claim in build pass 2 that
  removing `redactInput`'s 2 KiB clip "survives every test" was wrong: the
  reviewer's plant was caught by `TestClassify_16KiBTokensUnder1ms`
  (`api_error_cut_0028_test.go:18`). Struck, dated, and the follow-up it raised
  dropped. The owner's confirmation of `TestLookupModel_Bounds_0033` is
  recorded where build pass 2 names that change.
* **A follow-up, not a fix:** `io.ReadAll` over `io.LimitReader` grows its
  buffer by doubling, so reading an error body up to 1 MiB can hold about twice
  that while it is read; a read into a buffer sized from `Content-Length`, up to
  the limit, would avoid it. It is bounded by the limit, and is for a later
  record.

#### Outcome of Phase 4's fixes (2026-11-21)

Built on `b4ad41f`, after the records above. No live run, nothing staged or
committed.

* **B1.** `TestClassify_ManyDetails_0084` (`llmprovider`) was **red first** on
  the unfixed code: "a 1023820-byte body of distinct reasons took 745 ms; want
  under 100 ms", "kept 32340 types; want at most 32 reasons and the status",
  and 8,134 quotas kept, in the error and in the envelope. It is green after
  the fix: `parseAPIErrorBody` tells a new type through a set, reads at most
  `apiErrorDetailLimit` (32) `details` entries and, in all, at most
  `apiErrorQuotaLimit` (32) quota violations. The test also pins one detail
  carrying all the violations (the entry cap does not bound it) and an error
  with three reasons and three violations keeping all of them. ~~The bound is
  ten times as long under the race detector (`race_0084_test.go` and
  `norace_0084_test.go`, one constant); `-race -count=3` on the test, exit
  0.~~ *(Struck 2026-11-21, after the re-check, see "The re-check's two
  fixes" below: the race detector slows this parse about twentyfold, not
  tenfold, and the two files are removed.)*
  **Plants** (`tools/plant.py` and the reviewer's runner, on scratch copies):
  the code before the fix, the scan restored and the details cap removed
  together: caught (3.15 s; the time and the types); the details cap removed:
  caught; the quota cap removed: caught (the one-detail body); each cap at
  zero: caught. **One survivor, an equivalent mutant:** the scanned list
  restored *alone*, with the 32-entry cap kept. The list it scans then holds at
  most 33 types, so the cost and the output are unchanged and no test can see
  it; the set matters only if the cap is removed, which the plants above catch.
* **The re-check's two fixes (2026-11-21), tests only, no production change.**
  (1) **E1:** "at most 32 quota violations in all" was unpinned: the plant that
  caps per detail (`apiErrorQuotaLimit` in place of `max(0,
  apiErrorQuotaLimit-len(env.quotas))`, up to 1,024 kept) survived
  `TestClassify_ManyDetails_0084`. The reviewer's case is in that test: two
  details of 40 violations each keep at most 32 in all. It fails under E1
  ("kept 64 quotas") and under E3, the quota cap removed ("kept 80"). With the
  reviewer's `caps.json` and `caps_inall.json` (the latter pointed at this
  test's name), by `tools/plant.py`: **E1, E2 and E3 caught; E4, the scanned
  list restored with the caps kept, stays the recorded equivalent mutant.**
  (2) **The time bound, replaced.** The race detector slows this parse about
  twentyfold (18 ms to 390 ms), not tenfold, so the absolute bound, ten times
  longer under `-race`, had only about threefold headroom, and no plant was
  caught by the time alone (the count assertions catch E2 and E3). The test
  now takes the best of five classifications of a body of distinct reasons at
  about 1 MiB over the best of five at 64 KiB, in the same run, and requires
  the ratio under 64: about 16 as built, in both modes; the pre-fix code (the
  scanned list, no caps, on a scratch copy) gives 194, red ("took 747 ms, and
  one of 64 KiB 3.85 ms: a ratio of 194"). The count assertions are kept. The
  two race files are removed, nothing else using them. `-race -count=5` on the
  test through `run-heavy`, exit 0; `-count=10` without `-race`, exit 0. **A
  follow-up, not a fix:** `perDayQuota` could be computed over every violation
  of the kept details, with only the stored list capped; today a per-day
  violation past the 32nd would read as `ErrRateLimited`, and no recorded body
  comes close.
* **Fix 2, ST11, ST42, ST43.** `TestToolSchemaRefusal_NamesNothingNoSlot_0084`
  (the reviewer's draft, placed with the providers' 0084 tests) and four cases
  of `TestNameRefusedTool_Forms_0084` (OpenAI's and Google's, two tools and
  one). These tests are additions that pass on the unchanged code; their red is
  their plants, the reviewer's A1 to A4 run unchanged from `specs/`: against
  the providers' package, 4 of 4 caught; against `internal/wire`, 4 of 4
  caught; against the renamed test alone, 4 of 4 caught; against the `Forms`
  cases alone, 4 of 4 caught. (`helper_proposed.json` names the draft's own
  test and so matches nothing now; it was run again with the new names.)
* **Fix 3, ST10.** A zen-google case with an unsent deferred tool in
  `NeverNamesUnsent`: the reviewer's B1 and B2 (`opencode_providers.json`), 2
  of 2 caught; B2 was caught by no test before. The reviewer's `extra.json`
  (C1 to C3) and `clip.json` (D1) are caught too, 4 of 4.
* **Fix 4, docs.** `opencode.go`'s package doc says both forms are read on
  every route of the gateway, by marker, and that the google route was recorded
  sending the second; the changelog's **Added** says the same, and its
  1 MiB sentence moved to a new **Changed** entry that names the example (a
  200 KiB OpenAI quota 429 is now `ErrQuotaExhausted`) and the two caps
  (`Limit.Quotas` holds at most 32 entries).
* **Fix 5, records.** The build pass 2 claim about `redactInput` is struck, and
  its follow-up dropped; the owner's confirmation of
  `TestLookupModel_Bounds_0033` is recorded in build pass 2's entry; the
  MADR's ninth amendment holds ST87 and the annotated ST86. The `io.ReadAll`
  note is a follow-up above, not a fix.
* **The gate**, on this Mac, each status before any filter: `run-heavy -- go
  test -count=1 ./...`, exit 0, 44 packages `ok`, 30 s; `run-heavy -- make
  pre-add-check FILES="<the 7 changed .go files>"`, exit 0, "go-precheck: 7
  file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)" *(run
  again after the re-check's fixes, on the 5 .go files that remain: exit 0,
  "go-precheck: 5 file(s) clean", with `go test -count=1 ./llmprovider/` and
  `run-heavy -- go test -count=1 ./...`, 44 packages `ok`, exit 0)*;
  `go vet -tags live_gateways ./...`, exit 0; `tools/apidiff.py 9e84a88
  <scratch commit>` (a scratch clone of `b4ad41f` with this phase's files
  copied in), exit 0, exactly `APIError.Tool` and `ErrToolSchemaRefused`, both
  compatible additions. The `make` doc gates, claims, recordlint and denyscan
  are in the hand-back, run after this entry was written.

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
| `TestToolSchemaRecord_LogLine_0084` (`llmprovider`, Phase 1a) | ST47 | the case's log line for a case whose body holds a made-up key is exactly ~~`"<id>: <outcome>, status <n>"`~~ *(superseded 2026-11-21, "Record all three readings (Recommended)": the five-field line `<case> <route> <model>: status <n>, <outcome>`)* | red at Phase 1a's base (the function does not exist) |

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
| ST67, ~~ST68,~~ ST70, ST78 | `TestToolSchemaRecord_Outcome_0084` | 1a | four plants: a 404 read as `refused`; a `Status`-0 `ErrIncomplete` read as `other`; the writer replacing a `refused` record; a blank key read as set *(ST68 struck 2026-11-21, build pass 3: it is Phase 1c's rule, see the entry below)* |
| ST69 | `Names` | 2 | Anthropic's row removed (Phase 3) |
| ST71 | the `grep` check of Phase 1a step 5 | 1a | `t.Fatal(err)` in a scratch copy; seen 2026-11-20 on `live_0062_refusals_measure_test.go`, which it hit seven times |
| ST72 | `Names` (as ST09); `TestCheckDispatch_0040`, unchanged | 2 | none new |
| ST73, ST74 | `grep -c ErrToolSchemaRefused` on each file; `TestDescribe_ToolSchemaRefused_0084`; Phase 4's read | 2 | the entry removed on a scratch copy: exit 1 |
| ST77 | `NoLeak`; `grep -nE 'os\.Args\|flag\.\|os\.ReadFile'` on the recorder's files, exit 1 *(restated 2026-11-21: `os.Args\|flag\.` on all four files; `os\.ReadFile` on the live file alone)* | 1a | the recorder logs its key variable's value |
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
| `TestToolSchemaRecord_Outcome_0084` (`llmprovider`, Phase 1a) | ST67, ~~ST68,~~ ST70, ST78 *(ST68 struck 2026-11-21, build pass 3: Phase 1c's rule)* | each row of Phase 1a step 2's outcome table; a key unset, `""` and `"   "`: `skipped`, "key unset or blank"; the writer keeps a `refused` or `accepted` record against every other outcome, and writes `skipped` only where none exists | red at Phase 1a's base |
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

### The release preparation (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This record's entries are under `Added` and `Changed`. `docs/guides/migrating-from-mcplib.md`'s `APIError.Unwrap` row now lists `ErrToolSchemaRefused` beneath `ErrInvalidRequest`.

### The release completed (2026-11-23)

Tag `v1.32.0` (annotated, the owner's) on `f86933c`, pushed 2026-11-23. Tag CI run 45195957872: completed, success. `python3 -B tools/consumer.py v1.32.0`: exit 0, "consumer: 17 packages linked", "consumer: every step passed". `python3 -B tools/offline.py v1.32.0`: exit 0, "caught: no caller layer", "offline: every step passed". The GitHub Release `v1.32.0` was published 2026-11-23T14:46:16Z by the release workflow, not a draft, not a prerelease, its body the `v1.32.0` changelog section (verified by the coordinator, 2026-11-23). Release phase complete. This PLAN's status is set to `complete` (`tools/records.py status`, 2026-11-23). 0088 and 0089 are not in this release.

*(2026-11-27: taken up by [0095-MADR-clear-the-sdk-follow-ups.md](0095-MADR-clear-the-sdk-follow-ups.md): Gemini's single-quoted name is its FU11, read; Zen Google's `additionalProperties` refusal is FU12, left; the per-day quota past the 32nd is FU13, read; the read buffer is FU14, left; Claude's deferral form is FU15, computed once, the positional `reply bool` left; the recorder's left-overs are FU21, four closed as records only and two stopped in pass 1 for the owner.)*

*(2026-11-27, later: of FU21's two stopped items, the live runner's move is built, on the owner's "Build both"; keeping a provider-build failure's text needs an outcome ST67 does not have, so it is stopped again, for the owner; 0095-MADR's third amendment.)*
