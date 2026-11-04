---
status: in-progress
date: 2026-11-04
associated-madr: "0069-MADR-discard-an-sse-event-the-stream-ends-inside.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Discarding a Server-Sent Event the Stream Ends Inside

Associated MADR: [0069-MADR-discard-an-sse-event-the-stream-ends-inside.md](0069-MADR-discard-an-sse-event-the-stream-ends-inside.md)

## Goal

The MADR's D1 to D3:

* at a clean end of the body, `wire.ReadEvents` does not dispatch pending
  data. With data pending, the early-end error names "inside an event",
  with the early end's kind and mark (D1);
* the three tests that pin today's dispatch change as D2 names, each seen
  red at `8203326` and on its plant;
* `ReadEvents`' doc, `docs/architecture.md`, `CHANGELOG.md` and the two
  amended records say so (D3);
* no exported name added or changed, no request changed, released as the
  owner answers Q1.

## Scope

### In scope

* **Phase 1, the reader (D1, D2):** `llmprovider/internal/wire/sse.go`,
  `llmprovider/internal/wire/sse_0031b_test.go`,
  `llmprovider/internal/wire/sse_0053_test.go` and
  `llmprovider/internal/wire/responses/wire_test.go`.
* **Phase 2, docs (D3):** `docs/architecture.md` and `CHANGELOG.md`.
* **Phase 3, the live pin (D2):** the live streaming tests, run once. No file
  in the tree changes.
* **Phase 4, the release (Q1):** the README's Status if it names a version,
  `CHANGELOG.md`'s section, and the tag, which is the owner's.
* **Records:** this PLAN's execution record; the MADR's amendments if a fact
  changes; the dated amendments to 0031-MADR E2 and 0053-MADR D3.

### Out of scope

* Any exported name added, changed or removed (R48, R55):
  `tools/apidiff.py v1.26.0` reports nothing.
* Any request's content (R45): G-wire's goldens do not change.
* A line ended by a lone CR, which the specification allows and `readLine`
  does not split on. No server was seen to send one, and it is a separate
  rule.
* Each wire's own `Events` test of the rule: the rule is the shared
  reader's, tested there and through the Responses decoder.
* Kilo's capture, which needs the account's settings changed.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`, or `--markdown` for a records-only phase;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on the MADR;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything
  is written from it. Files are staged only when the gate passes (`&&`).
* **Red first.** Every changed test is seen to fail at `8203326` by
  `tools/redgreen.py SPEC.json --expect fail --rev 8203326`, then to fail on
  each planted return of the dispatch, by `tools/plant.py`. Both run on
  scratch clones, never in the tree. The phase's record holds each failure
  line.
* **The docs.** `ReadEvents`' doc changes in the phase that changes it,
  naming only what a test shows (R52). `make docs-check` stays at 0.
* **The existing tests are the drift check.** Every other test at
  `8203326` passes unchanged.
* **Other platforms.** The change is byte parsing, with no build tag, file,
  path, process or network code, so it does not run on the Linux and
  Windows dev hosts as 0068-PLAN's networking phases did. CI's three
  platforms run it before the tag.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a fact of the MADR, or one of its claims, no longer holds;
  * a test other than D2's three would need to change;
  * a step would add or change an exported name, or change a golden;
  * Phase 3 finds a live stream that fails with D1's error.

  On any of these, stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. This PLAN, the MADR and their rows in `docs/README.md`, committed as
   records only.
2. Once the owner approves this PLAN, `tools/records.py status` sets this
   PLAN `in-progress`.
3. A dated amendment, without rewriting the decision, in:
   * 0031-MADR-native-streaming-on-the-responses-wire.md E2: the shared
     framing discards an event the stream ends inside (0069-MADR D1);
   * 0053-MADR-stream-retry-and-cache-defects.md D3: the reader dispatches
     nothing at the stream end, so the mark covers a dispatch error at a
     blank line only, and its test's "at the stream end" case goes
     (0069-MADR D2).

### Phase 1: the reader (D1, D2)

1. **`ReadEvents`,** after its loop: with `data` pending, return
   `ErrProviderUnavailable`, wrapping `transport.AfterReply` of "stream ended
   inside an event, before " and `done`. With none pending, return today's
   early-end error. The end-of-body dispatch (C1) goes.
2. **The doc:** `ReadEvents`' doc gains: "An event the stream ends inside,
   with no blank line after it, is not dispatched, as the server-sent events
   specification requires: the stream ended before done (0069-MADR D1)."
3. **Tests:**
   * `TestReadEvents_LastEventWithoutBlankLine` (`sse_0031b_test.go`): its
     comment cites 0069-MADR D2. A table of bodies:
     * `data: one\n\ndata: end`;
     * `data: one\n\ndata: end\n`;
     * `data: one\r\n\r\ndata: end\r\n`;
     * `data: one\n\ndata: end\n: a comment\n`.

     Each must give events `["one"]` and an error that is
     `ErrProviderUnavailable`, after the reply, containing "stream ended
     inside an event, before the end event". The body `data: one\n\nevent:
     end\n` must give `["one"]` and an error containing "stream ended before
     the end event" and not "inside";
   * `TestReadEvents_DispatchErrorMarked_0053` (`sse_0053_test.go`): the
     "at the stream end" entry is removed. A comment cites 0069-MADR D2;
     *(Deviation 1, 2026-11-04: the case is inverted, not removed. At the
     stream end a failing dispatch is never called, and the error is the
     early end's: `ErrProviderUnavailable`, after the reply, with no
     `*APIError`.)*
   * `TestReadStream` (`responses/wire_test.go`): the case "data split over
     lines, and a final event with no blank line" becomes "data split over
     lines", its body ending `\n\n`, still wanting `r2`. A new case, "a final
     event with no blank line is not read", is the same body without the
     `\n\n`, and wants `ErrProviderUnavailable` and no response.
4. **Red:** a `tools/redgreen.py` spec naming the three tests and the new
   case, `--expect fail --rev 8203326`.
5. **Plants,** by `tools/plant.py`:
   * the end-of-body dispatch restored before the new return: each changed
     test fails;
   * the pending case's text made "stream ended before": the reader's test
     fails.
6. **The drift check:** `go test ./...` passes, and the gate.

### Phase 2: docs (D3)

1. `docs/architecture.md`: the `internal/wire` row's `ReadEvents` gains
   "which does not dispatch an event the stream ends inside (0069-MADR
   D1)".
2. `CHANGELOG.md`, `[Unreleased]`, Fixed: "An event the stream ends inside,
   with no blank line after it, is no longer read. It was, against the
   server-sent events specification. The stream now ends with
   `ErrProviderUnavailable`, naming the event it lacked, as a stream that
   ends early does. A stream cut short on a connection's close is now
   `ErrProviderUnavailable`, retried once, where it was a decode error with no
   kind. Every server measured ends its events as the specification requires
   (0069-MADR)." *(2026-11-04, in execution: "retried once" overstated it, since
   a retry happens only under `WithRetry` or `WithStreamRetry`. The entry reads
   "which `WithRetry` and `WithStreamRetry` ask again at most once, where it
   was a decode error with no kind, never retried".)*
3. `make docs-check`, `make changelog-check`.

### Phase 3: the live pin (D2)

No file in the tree changes. Each test sends one billed request.

1. `go test -tags live_gateways ./llmprovider/ -run 'TestLive_.*Streams' -count=1 -v`,
   at the commit of Phases 1 and 2, its output in the session's scratch
   directory.
2. **The rule:** a stream that fails with "stream ended inside an event"
   stops the work. A skip for a transient failure or an unfunded account,
   such as Kilo's, is recorded with its reason.

### Phase 4: the release (Q1)

1. As the owner answers Q1: `CHANGELOG.md`'s `[Unreleased]` becomes the
   release's section, dated by the tag's commit; the README's Status names
   the version if it names one.
2. Before the tag: CI on `main` passes; `make changelog-check`; the
   disclosure guard over the outgoing commits.
3. The tag and its push are the owner's. After it: CI on the tag; the
   GitHub Release's notes equal the section; `tools/consumer.py` and
   `tools/offline.py` on the tag.

## Verification

* Phase 1's red and plant output, each failure line, in this record.
* `go test ./...` and `python3 -B tools/gate.py` passing at each code phase.
* `tools/apidiff.py v1.26.0` empty; `make docs-check` at 0.
* Phase 3's live output: every stream passes or skips with a recorded
  reason, and none fails with D1's error.
* The MADR's claims pass at their revision.

## Rollout and Rollback

* **Rollout:** a patch or minor release, as Q1 says. Callers take it by
  upgrading the module; nothing is configured.
* **Rollback:** revert Phase 1's commit. The reader dispatches the pending
  event again, and the three tests return to their `8203326` form. No data,
  file or request format depends on the change.
* **If a server is found that leaves off the blank line:** the caller's
  error names it ("stream ended inside an event"). The remedy is a new
  record weighing that server, not a special case in the reader (MADR
  option C).

## Execution record

### Phase 0 (2026-11-04)

* **Step 1:** the MADR, this PLAN and their rows, committed as `0ddf5e5`.
  Before that commit:
  * the MADR's claims passed, 18 of 18;
  * three planted false claims were each caught by `tools/claims.py
    --plants`: C1 at the wrong line, C3 with text the code lacks, and C5
    naming a case that is not there;
  * the Markdown gate passed, 11 of 11, on its second run. On the first,
    `gate-selftest` exited 2: its generate-check fixture's `go run
    …mkwinsyscall@v0.47.0` failed with "net/http: TLS handshake timeout"
    fetching from the module proxy, with the host's load at 44.9 before
    the gate started. Nothing was staged until the second run passed.
* **Step 2:** the owner approved this PLAN on 2026-11-04 and answered Q1,
  `v1.26.1`, which the MADR's questions now record. `tools/records.py
  status` set this PLAN `in-progress`.
* **Step 3:** the dated amendments, appended without rewriting:
  * 0031-MADR-native-streaming-on-the-responses-wire.md, "Amendment
    2026-11-04: E2, an event the stream ends inside is not dispatched";
  * 0053-MADR-stream-retry-and-cache-defects.md, "Amendment 2026-11-04: D3,
    no dispatch at the stream end".

### Deviation 1 (2026-11-04): the 0053 test's case is inverted, not removed

* **Found:** steps 4 and 5 require every changed test to be red at
  `8203326` and to fail on each plant. MADR D2 only removed
  `TestReadEvents_DispatchErrorMarked_0053`'s "at the stream end" case, so
  the test asserted nothing that could fail:
  * `tools/redgreen.py --expect fail --rev 8203326`: `BAD pass
    TestReadEvents_DispatchErrorMarked_0053`, while the other two were red;
  * `tools/plant.py`, the end-of-body dispatch restored, that test alone:
    `NOT CAUGHT (exit 0)`.

  The fault was the PLAN's: it listed this test with the ones that gain an
  assertion. It was not pre-existing.
* **Decided by the owner, 2026-11-04, from three resolutions:** invert the
  case (chosen); keep the removal and limit steps 4 and 5 to the other two
  tests; move the case into `TestReadEvents_LastEventWithoutBlankLine`.
* **The change:** the case stays, and requires that, at the stream end, the
  failing dispatch is never called. The error must be the early end's,
  `ErrProviderUnavailable` after the reply, with no `*APIError` in its
  chain. The MADR's D2 gains an amendment; the 0053-MADR amendment of this
  date gains a dated note. No file is added to the phase.

### Phase 1 (2026-11-04)

* **Steps 1 and 2:** `ReadEvents` returns, with data pending at the end of
  the body, `ErrProviderUnavailable` wrapping `transport.AfterReply` of
  "stream ended inside an event, before " and `done`. The end-of-body
  dispatch is gone. Its doc gains the step's sentence.
* **Step 3:** the three tests, as the step says and as Deviation 1 amends it.
* **Step 4, red at `8203326`,** by `tools/redgreen.py --expect fail --rev
  8203326`: "3 of 3 fail as expected". The failure lines:
  * `sse_0031b_test.go:55: no line ending: events ["one" "end"] (<nil>);
    want one, then the early end inside an event`, and the same for "one
    LF", "one CR LF" and "a comment after the data";
  * `sse_0053_test.go:33: at the stream end: dispatched true, llmprovider:
    provider unavailable: server_error; want no dispatch, and the early end
    marked after the reply`;
  * `wire_test.go:112: ReadStream = &{ID:r2 …}, <nil>; want no response and
    the early-end error`.

  The "no data pending" row of the reader's test passes at `8203326`, as it
  should: that early end is unchanged.
* **Step 5, plants,** by `tools/plant.py`, both caught:
  * the end-of-body dispatch restored: `--- FAIL:
    TestReadEvents_LastEventWithoutBlankLine`, `--- FAIL:
    TestReadEvents_DispatchErrorMarked_0053`, `--- FAIL: TestReadStream`
    (its case "a final event with no blank line is not read");
  * the pending case given "stream ended before": `--- FAIL:
    TestReadEvents_LastEventWithoutBlankLine`, each of its four rows
    reading `(llmprovider: provider unavailable: p: stream ended before the
    end event); want one, then the early end inside an event`.
* **Step 6:** `go test ./llmprovider/internal/wire/...` passes.
* **The gate, before staging:**
  * `make pre-add-check` on the four Go files: "4 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)";
  * `python3 -B tools/gate.py`: "29 of 29 passed", among them
    `test-race-cover`, `test-shuffle`, `test-repeat`, `lint`, `api-check`,
    `docs-check` and `g-wire-stable`;
  * `tools/apidiff.py v1.26.0`: no output, exit 0;
  * `tools/claims.py` on the MADR: 18 of 18 pass; `tools/recordlint.py
    --changed`: no rule rose.

  Every other test at `8203326` passes unchanged: the gate's test runs are
  the whole module's.
* Committed by the owner as `6401b7c`.

### Phase 2 (2026-11-04)

* **Step 1:** `docs/architecture.md`'s `internal/wire` row: `ReadEvents`
  "does not dispatch an event the stream ends inside (0069-MADR D1)".
* **Step 2:** `CHANGELOG.md`'s `[Unreleased]` gains a Fixed entry, worded as
  the step's annotation says.
* **Step 3:** `make changelog-check`: "38 release section(s), 0 problem(s)";
  `make docs-check`: "0 violation(s) of R50,R51,R52".
* Committed by the owner as `e416033`.

### Phase 3 (2026-11-04)

* **Step 1,** at `e416033`: `go test -tags live_gateways ./llmprovider/ -run
  'TestLive_.*Streams' -count=1 -v`, exit 0, "ok … 54.436s". Each stream
  that ran passed with at least one text delta and then `EventDone`:
  * Responses: OpenAI `gpt-4.1-mini`, 9 deltas; xAI `grok-4.6`, 9; OpenCode
    Zen and Go `gpt-6-luna`, 2 each;
  * Messages: Anthropic `claude-haiku-5-5`, 2; OpenCode Zen and Go
    `qwen3.8-flash`, 3 each;
  * Chat Completions: Hugging Face `deepseek-ai/DeepSeek-V4-Flash-0731`, 1;
    Together `zai-org/GLM-5.3-Flash`, 2; OpenCode Zen `glm-5.3-flash`, 5,
    and Go, 7; a local Ollama `llama3.2:latest`, 2;
  * generateContent, Google's endpoint: `gemini-3.5-flash-lite`, 2;
  * Interactions: Gemini `gemini-3.5-flash-lite`, 2, with an interaction id.
* **Step 2, the rule:** no stream failed with "stream ended inside an event"
  (no line of the output holds it). Two skipped, each on a failure the live
  tests treat as transient:
  * `TestLive_KiloStreams`: "quota exhausted: kilo HTTP 402
    usage_limit_exceeded: Add credits to continue, or switch to a free
    model";
  * `TestLive_OpencodeGoogleStreams/opencode-zen`: "rate limited:
    opencode-zen/google HTTP 429 UNKNOWN: Upstream request failed: Endpoint
    is unavailable". It was run once more, alone, and skipped the same way.
    The same route streamed, ending with a blank line, in the MADR's
    captures earlier the same day.
* **Kilo's free models,** tried at the owner's request in this phase, with
  the scratch capture program and no file in the tree changed. Kilo's
  public list marks eleven models `isFree`. Each was tried in the list's
  order with the account's key (`nvidia/nemotron-3.5-lightning:free` in the
  MADR's captures, the other ten here). Each was tried again with no key,
  when the provider sends Kilo's `anonymous` token (`kilo.go:203`, which
  reads no environment variable). All twenty-two requests were refused with
  HTTP 400 `data_collection_required`. The list marks all of them
  `mayTrainOnYourPrompts: true`. Enabling data collection is a setting of
  the owner's Kilo account and was not changed. The MADR gains an amendment,
  since it called the refusal the account's setting alone.
* Committed by the owner as `8141e26`.

### Phase 4 (2026-11-04)

* **Step 1:** `CHANGELOG.md`'s `[Unreleased]` becomes `[v1.26.1] -
  2026-11-04`, under a new empty `[Unreleased]`; the compare links gain
  `v1.26.1`. The README's Status reads "The current release is `v1.26.1`".
  `make changelog-check`: "39 release section(s), 0 problem(s)".
  `scripts/check_changelog.py --section v1.26.1` gives the section's one
  Fixed entry, the notes the Release is to carry.
* **Steps 2 and 3:** the push, CI on `main`, the tag and the checks after
  it, to follow in this record.
