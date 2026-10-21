---
status: complete
date: 2026-10-21
associated-madr: "0019-MADR-go-fix-modernizations.md"
---
# Implement `go fix`'s modernizations, except one that changes behaviour

Associated MADR: [0019-MADR-go-fix-modernizations.md](0019-MADR-go-fix-modernizations.md)

## Goal

`go fix ./...`'s four behaviour-preserving rewrites are in the tree.
`pathSegments` has a form `go fix` leaves alone. No behaviour or exported API
changes.

## Scope

* `llmprovider/internal/kiloendpoint/kiloendpoint.go`: the `pathSegments`
  loop.
* What `go fix ./...` then rewrites:
  * `llmprovider/command_token.go`;
  * `llmprovider/provider.go`;
  * `llmprovider/retry.go`;
  * `llmprovider/auth/oauth_loopback.go`.
* This pair, and `docs/README.md`'s index.

Out of scope: any test change, any exported identifier, and reporting the
modernizer's rewrite upstream.

## Implementation Steps

1. **`pathSegments`.** Replace the backward loop with a forward scan:
   `api = -1`, then `api = i` for each `"api"` segment, with a comment naming
   0019-MADR.
   * `TestResolve` and `TestRoute` must pass unchanged.
   * `go fix -diff ./llmprovider/internal/kiloendpoint/` must print nothing.
2. **`go fix ./...`.** Apply it. Then read the whole diff: only the four
   rewrites the MADR lists may appear, plus the imports they need (`maps`,
   `slices`).
3. **First-fail.** On a scratch copy, put the original backward loop back and
   apply `go fix`. `TestResolve`, `TestRoute` and
   `TestKilo_URLTokenSelectsBase` must fail, as they did before this plan.
   That shows the tests still guard the loop.
4. **Gate.** The full gate, `go fix -diff ./...` empty, and `make api-check`
   against `v1.0.0` clean.

## Verification

| Criterion | Check |
|---|---|
| Behaviour unchanged | `go test -race ./...` with no test changed; the G-wire goldens unchanged |
| `go fix` has nothing left | `go fix -diff ./...` prints nothing |
| No API change | `make api-check` against `v1.0.0` |
| The trap is guarded | step 3's first-fail |

## Rollout and Rollback

* One commit, which the owner makes and pushes. No tag: an internal change
  can wait for the next release.
* Rollback is `git revert` of that commit.

## Execution Record

### 2026-10-02

Approved by the owner ("proceed"). Staged for the owner to commit.

* **Step 1, `pathSegments`.**
  * It now sets `api = -1`, then `api = i` for each `"api"` segment, with a
    comment naming this record.
  * `kiloendpoint` and `providers/kilo` pass with no test changed.
  * `go fix -diff ./llmprovider/internal/kiloendpoint/` prints nothing.
* **Step 2, `go fix ./...`.** The whole diff was read. It changes four
  files, with only the four rewrites the MADR lists:
  * `requireEndpoints` uses `slices.Contains`;
  * `runTokenCommand`, `serverRetryAfter` and `retryable` use
    `errors.AsType`;
  * `ProviderEnvVars` uses `maps.Copy`;
  * the `maps` and `slices` imports they need.

  No test file changed.
* **Step 3, first-fail,** on a scratch copy with the original backward loop
  restored:
  * `go fix` rewrote it to `slices.Backward` again;
  * `TestResolve`, `TestRoute` and `TestKilo_URLTokenSelectsBase` failed.
* **Step 4:**
  * `go fix -diff ./...` prints nothing;
  * `make api-check`: `against v1.0.0, 0 incompatible change(s) outside
    llmprovider/x/`;
  * the gate passes, all 17 checks, G-wire unchanged;
  * coverage is `kiloendpoint` 100.0 %, `llmprovider` 98.2 % and `auth`
    85.3 %, as before.
* **Not done, as scoped:** no tag, and no report of the modernizer's
  rewrite upstream.

## Phase 2 (2026-10-09): a second run, on Go 1.27.2

Added by 0019-MADR's amendment "a second run, on Go 1.27.2". Runs after
0028-PLAN's Phase 9b is committed.

1. **The test first.** `TestIgnored_EventTypes`
   (`llmprovider/internal/wire/responses/ignored_test.go`): a payload whose
   type is a `response.` event not in `handledEvents` is skipped; one in
   `handledEvents`, one whose type is not a `response.` event, one with
   no type, and one whose type is not closed are decoded. It passes on
   `HEAD`; on a scratch copy with `ignored` returning `false` it fails, and
   with it returning `true` it fails.
2. **`go fix ./...` and `go fix -tags live_gateways ./...`.** Read the whole
   diff: only the three rewrites the amendment lists, and the `slices`
   import.
3. **Tidy `ignored`:** `_, rest, ok := bytes.Cut(payload, []byte(key))`, then
   `eventType, _, ok := bytes.Cut(rest, []byte{'"'})`; the logic as `go fix`
   left it. `go fix -diff ./...` and `go fix -diff -tags live_gateways ./...`
   print nothing.
4. **Gate.** The full gate; `go vet -tags live_gateways` on the three
   platforms; the G-wire goldens unchanged; `make api-check` clean.
5. **Record; stage** as one change, separate from 0028's.

| Criterion | Check |
|---|---|
| Behaviour unchanged | every test passes with none changed but the one added; goldens unchanged |
| `go fix` has nothing left | step 3's two `-diff` runs print nothing |
| The tidy is guarded | step 1's two plants fail `TestIgnored_EventTypes` |

### Phase 2's execution (2026-10-09)

* **Before it.** 0028-PLAN's Phase 9b committed by the owner as `f378147`,
  and this phase's records as `c26b57b`.
* **Approval.** "i committed, proceed", 2026-10-09. Staged for the owner to
  commit.

**Step 1, the test first.** `TestIgnored_EventTypes` checks three skipped
`response.` events (a text delta, `response.in_progress`, a reasoning
delta) and eleven decoded payloads: each of the five `handledEvents`, an
`error` event, no type, a type not closed, an empty payload, and one whose
first `"type"` is a nested item's. It passed on `HEAD`'s `ignored`. Plants on
scratch copies (`gf_plants.py`), run before `go fix` and again after the
tidy, with the same result each time:

| Plant | Tests that failed |
| :--- | :--- |
| `ignored` always `false` | `TestIgnored_EventTypes` only, the gap the amendment names |
| `ignored` always `true` | `TestIgnored_EventTypes` and seven decode and stream tests |
| the `response.` prefix check dropped | `TestIgnored_EventTypes`, `TestReadStream_ErrorEventIsClassified` |

**Step 2, `go fix ./...` and `go fix -tags live_gateways ./...`,** both exit
0. The whole diff was read. It changes three files, with only the three
rewrites the amendment lists:

* `startMetadataFetch` uses `metadataFetching.Go`;
* `ignored` uses `bytes.Cut`, with the names `after`, `rest := after`,
  `before0` and `ok0`;
* the live-tagged `contains` uses `slices.Contains`, with its import.

**Step 3, the tidy.** `ignored` now reads
`_, rest, ok := bytes.Cut(payload, []byte(key))`, then
`eventType, _, ok := bytes.Cut(rest, []byte{'"'})`, with the logic `go fix`
left. `go fix -diff` prints nothing for darwin, linux (`CGO_ENABLED=0`) and
windows, with and without `-tags live_gateways`.

**Step 4, the gate** (`p26_gate.py`, a scratch copy of the tree). Every
check exits 0 on the first run:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `508 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `coverage-check` | `wire/responses: 98.5%` (97.8 % before); `catalog: 92.6%`; `llmprovider: 98.2%`; `28 packages, 0 problem(s)` |
| goldens (`g-wire-stable`) | 0; no golden changed |
| `api-check` | `against v1.3.2, 0 incompatible change(s) outside llmprovider/x/` |
| `records-check` | `63 records, 0 problem(s)` |
| links | `0 problem(s), 628 relative link(s) in 73 file(s)` |
| vet (darwin, linux, windows; with and without `live_gateways`), race, shuffle, lint, tidy, parity, dep, generate, gate self-test, markdownlint, deny scan | 0 each |

No test changed but the one added.

* **Not done, as scoped:** no tag; the live-tagged `contains` was vetted and
  linted, not run, since live tests call real services.

## Phase 3 (2026-10-20): a third run, on Go 1.27.3

Added by 0019-MADR's amendment "a third run, on Go 1.27.3". Proposed: it
runs on the owner's approval, after 0052's staged fix is committed.

1. **The test first.** In `llmprovider/llmtest`, a test that `Run`'s stream
   relay passes on the headers the harness's `StreamText` set (a custom
   header and `Content-Type`). It passes on `HEAD`. On a scratch copy whose
   relay copies no header it fails, and that failure is recorded.
2. **`go fix ./...` and `go fix -tags live_gateways ./...`.** Read the whole
   diff: only the rewrites the amendment lists, in its 12 files.
3. **The tidies:**
   * `messages/cache.go`: one import block;
   * `band_0041_test.go`: `ptrBool` deleted;
   * `sdkaudit`: `strings.SplitSeq` at `audit.go:108`, `audit.go:185` and
     `main.go:141`.

   `go fix -diff` prints nothing for darwin, linux (`CGO_ENABLED=0`) and
   windows, with and without `-tags live_gateways`.
4. **Gate.** `tools/gate.py`; `make pre-add-check` on the changed Go files;
   the G-wire goldens unchanged; `tools/apidiff.py` against the latest tag,
   no change.
5. **Plants,** on scratch copies, of each production rewrite broken. Each
   fails a test, and the record names it.
6. **Record; stage** as one change, apart from 0052's, 0055's and 0056's.

| Criterion | Check |
|---|---|
| Behaviour unchanged | every test passes with none changed but the one added; goldens unchanged |
| `go fix` has nothing left | step 3's six `-diff` runs print nothing |
| Every production rewrite is guarded | step 5's plants each fail a test |

### Phase 3's execution (2026-10-20)

* **Approval.** "Proceed", 2026-10-20.
* **Not yet as the phase is written.** 0052's fix was still staged, not
  committed, when the work began. The work stayed in the working tree, and
  is staged only once the owner has committed 0052's fix, so the two stay
  apart. No decision changes.
* **The toolchain.** `go version go1.27.3 darwin/arm64`: the owner's Go
  environment already names 1.27.3.

**Step 1, the test first.**
`TestStreamStaged_RelaysHeaders_0019`
(`llmprovider/llmtest/stream_relay_0019_test.go`, new):

* `StreamText` sets `Content-Type: text/event-stream` and
  `X-Llmtest-Probe: kept`;
* a `headerProbe` around the reference streaming provider keeps the reply's
  headers;
* `streamStaged` runs it.

It passed on `HEAD`'s loop. Plants on a scratch clone (`tools/plant.py`):

| Plant | Failure line |
| :--- | :--- |
| the relay copies no header | "the relayed reply's headers are map[Content-Type:[text/plain; charset=utf-8] Date:[…]]; want StreamText's X-Llmtest-Probe and Content-Type" |
| the relay copies `Content-Type` only | "the relayed reply's headers are map[Content-Type:[text/event-stream] Date:[…]]; want …" |

**Step 2, `go fix ./...` and `go fix -tags live_gateways ./...`.** Each
exited 1, with "fix: applied 3 of 4 fixes; 2 files updated" for `sdkaudit`,
and changed 12 files: the amendment's 9, and the 3 live-tagged tests. Each
file was compared with the scratch clone whose whole diff had been read:
10 were identical, and the 4 that differ are the step 3 tidies' files.

**Step 3, the tidies,** by `tools/edit.py`, each asserted:

* `sdkaudit`: `strings.SplitSeq` in `public` and `hasCodeBlock`
  (`audit.go`) and `splitList` (`main.go`);
* `messages/cache.go`: one import block;
* `band_0041_test.go`: `ptrBool` deleted.

After them, every changed file equals the scratch clone's, which had passed
every check: 14 files, 40 lines in and 40 out. `go fix -diff` exits 0 with
no output for darwin, linux and windows (`CGO_ENABLED=0`), with and without
`-tags live_gateways`.

**Step 5, plants of each production rewrite,** on a scratch clone of the
tree, each import kept in use:

| Plant | Tests that failed |
| :--- | :--- |
| `cacheref.From` forward (`slices.Clone`) | `TestFrom_LastValueWins_0048` |
| the cache marker on the first tool | `TestAddCache`, `TestAddCache_Deferred_0040`, `TestDeferral_Native_0040`, `TestDeferral_Append_0040` |
| a deferred tool not skipped | `TestAddCache_Deferred_0040`, `TestDeferral_Native_0040` |
| `streamDetail.value` without its plain fields | `TestEvents_EqualsDecode`, `TestReasoningDetailsReplay_0053` |
| the relay's headers not copied | `TestStreamStaged_RelaysHeaders_0019` |
| `public` not seeing `internal/` | `TestCheckPassesTheGoodModule_0048` |
| `hasCodeBlock` seeing no line | `TestCheckPassesTheGoodModule_0048` ("exit 1, want 0") |
| `splitList` splitting nothing | `TestReportCountsEachPlantedDefectOnce_0048`, `TestCheckNamesEachViolationOnce_0048`, `TestCheckPassesTheGoodModule_0048` |

A first plant of `hasCodeBlock`, splitting on `\r`, was not caught. It
broke nothing: a `//` comment's `Text` is one line, so either split gives
it whole. The plant was replaced by one that sees no line, which is caught.
Before the phase, the amendment's own plants first dropped three calls
outright, and failed to build on the unused import. Every plant here keeps
the import in use, so that a test, not the compiler, catches it.

**Step 4, the gate,** each exit status captured before any filter:

```text
$ make pre-add-check FILES="<the 14 changed Go files> llmprovider/llmtest/stream_relay_0019_test.go"
go-precheck: 15 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck).   (exit 0)
$ python3 -B tools/apidiff.py v1.20.0
(no output)                                                           (exit 0)
$ python3 -B tools/gate.py
started at load 9.5
gate: 27 of 27 passed                                                 (exit 0)
```

The gate's 27 lines all read `rc=0`, `g-wire-stable` among them, so the
goldens are unchanged. Its govulncheck passes because this host runs Go
1.27.3; CI, on the `go 1.27.2` that `go.mod` names, still fails it until
`0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md`'s Phase 1 lands.

**Step 6.** Recorded here. The 15 Go files and this pair are staged as one
change once 0052's fix is committed.

### Completion of Phase 3 (2026-10-21)

* 0052's fix was committed first, at the owner's ask ("Commit then
  proceed"), as `e68be51`. This phase is then staged on its own: the 15 Go
  files and this pair.
* Every criterion of the phase's table is met: every test passes, with only
  `TestStreamStaged_RelaysHeaders_0019` added and the goldens unchanged;
  `go fix -diff` prints nothing on the six runs; each of the eight
  production plants fails a test. This PLAN is `complete`.
