---
status: accepted
date: 2026-10-09
decision-makers: repository owner
consulted: 0015-MADR-canonical-sdk-api-and-module-layout.md (R2, R48)
informed: consumers of go-llmprovider-sdk v1
---
# Adopt `go fix`'s modernizations, except one that changes behaviour

## Context and Problem Statement

The owner asked to run `go fix ./...` (Go 1.27.1) after `v1.0.0`. Its
`-diff` output, read before anything was applied, proposes five rewrites in
non-test code:

* `errors.As` with a declared target becomes `errors.AsType`, three times:
  `llmprovider/command_token.go`, `llmprovider/provider.go` and
  `llmprovider/retry.go`;
* a copy loop becomes `maps.Copy`, in `ProviderEnvVars`
  (`llmprovider/provider.go`);
* a search loop becomes `slices.Contains`, in `requireEndpoints`
  (`llmprovider/auth/oauth_loopback.go`);
* a backward loop becomes `range slices.Backward(parts)`, in `pathSegments`
  (`llmprovider/internal/kiloendpoint/kiloendpoint.go`).

The last one changes behaviour. `pathSegments` returns, in its named result
`api`, the index of the last `"api"` path segment, or -1. The original loop
leaves that index in `api` when it breaks. The rewrite ranges with `_, part`
and never assigns `api`, so it always returns 0.

On a scratch copy with `go fix ./...` applied, three tests fail:

* `TestResolve` in `kiloendpoint`;
* `TestRoute` in `kiloendpoint`, for example `Route("https://kilo.example.test/x/api/gateway?q=1")`
  returns `…/api/profile` where `…/x/api/profile` is wanted;
* `TestKilo_URLTokenSelectsBase` in `providers/kilo`.

So `go fix ./...` cannot be applied as it stands, and nothing stops the same
rewrite from being proposed again whenever someone runs it.

## Decision Drivers

* No change of behaviour, and none of the exported API: `v1.0.0` is released,
  and `make api-check` holds R48.
* The tree's tests must keep passing, unchanged.
* A later `go fix ./...` should be safe to run as it is.

## Considered Options

* "Apply the four safe rewrites, and rewrite the fifth's loop"
* "Apply the four safe rewrites only"
* "Do not apply `go fix`"

## Decision Outcome

Chosen option: "Apply the four safe rewrites, and rewrite the fifth's loop",
because it takes the modernizations that keep behaviour, and leaves
`pathSegments` in a form the modernizer does not rewrite, so a later
`go fix ./...` is safe. The owner chose it on 2026-10-02.

* `pathSegments` becomes a forward scan that keeps the last match: `api = -1`,
  then `api = i` for each `"api"` segment. It returns the same value for every
  input, and its comment names why the loop has this shape.
* Then `go fix ./...` is applied in full.

### Consequences

* Good, because the four rewrites are shorter and say what they do, with the
  same behaviour: `errors.AsType` finds the same error as `errors.As`,
  `maps.Copy` copies the same entries, and `slices.Contains` finds the same
  empty endpoint.
* Good, because a later `go fix ./...` has nothing left to propose here.
  Measured on a scratch copy: after the change, `go fix -diff ./...` is empty.
* Neutral, because no exported identifier changes, so `make api-check` stays
  clean against `v1.0.0`.
* Bad, because `pathSegments`' shape is held by a comment and its tests, not
  by the tool: a modernizer that learns a forward scan could propose a
  rewrite again. The tests caught this one.

### Confirmation

* `go test ./...` passes, with no test changed.
* `go fix -diff ./...` prints nothing afterwards.
* `make api-check` against `v1.0.0` reports no incompatible change.
* The gate passes.

*Annotated 2026-10-02:* all four are met; see the PLAN's execution record.

## Pros and Cons of the Options

### Apply the four safe rewrites, and rewrite the fifth's loop

* Good, because it keeps behaviour, and leaves `go fix ./...` safe to rerun.
* Bad, because it changes a loop for the tool's sake, and needs a comment to
  say so.

### Apply the four safe rewrites only

* Good, because `pathSegments` is untouched.
* Bad, because every later `go fix ./...` proposes the broken rewrite again,
  and only the tests would stop it.

### Do not apply `go fix`

* Good, because nothing changes.
* Bad, because the four rewrites are style only, and the trap stays for the
  next person who runs `go fix ./...`.

## More Information

* The plan: [0019-PLAN-go-fix-modernizations.md](0019-PLAN-go-fix-modernizations.md).
* The rewrite of a backward loop that drops its index looks like a
  modernizer defect. Reporting it upstream is not part of this decision.

## Amendment 2026-10-09: a second run, on Go 1.27.2

Asked by the owner ("run go fix ./... in this repo"), with the choices below
made by the owner the same day.

* **Found.** `go fix -diff ./...` under Go 1.27.2, run read-only for darwin,
  linux (`CGO_ENABLED=0`) and windows, with and without `-tags
  live_gateways`, proposes the same three rewrites on every platform; each
  keeps behaviour:
  * `llmprovider/live_gateways_test.go`: the `contains` loop becomes
    `slices.Contains` (live-tagged; this record's first run did not use the
    tag);
  * `llmprovider/catalog/model_metadata.go`: `metadataFetching.Add(1)` and a
    `go` statement with `defer metadataFetching.Done()` become
    `metadataFetching.Go(…)`;
  * `llmprovider/internal/wire/responses/responses.go`, `ignored`: two
    `bytes.Index` lookups become `bytes.Cut`, leaving the names
    `rest := after`, `before0` and `ok0`.
* **Decided** ("Apply, then tidy"; "9b commit first"): all three are
  applied, after 0028-PLAN's Phase 9b is committed, as a change of their
  own; `ignored`'s names are then tidied by hand to the plain `bytes.Cut`
  form, with no change of logic, and `go fix -diff ./...` stays empty.
* **Found while planning:** no test notices `ignored` reporting an event it
  should skip as one to decode (a scratch copy with `return false` passes
  every test), since decoding it is correct, only slower. A direct test of
  `ignored` is added before the rewrite, so the hand tidy is guarded.
* The first run's decision stands: the modernizer's rewrites are adopted
  when they keep behaviour.

## Amendment 2026-10-20: a third run, on Go 1.27.3

Asked by the owner ("Run go fix ./..."). Proposed; the choices below are the
owner's to make.

* **Found.** `go fix -diff ./...` under Go 1.27.3, run read-only at
  `5c60593` for darwin, linux (`CGO_ENABLED=0`) and windows, with and
  without `-tags live_gateways`, proposes the same rewrites on every
  platform, in 9 files, and 3 more with the tag:
  * **`mapsloop`,** a copy loop to `maps.Copy`, four times: in
    `chatcompletions.go` (`streamDetail.value`), `llmtest/stream_retry.go`,
    and two `auth` tests;
  * **`slicesbackward`,** a backward index loop to `range
    slices.Backward(…)`, twice: `cacheref.From` and `messages/cache.go`'s
    cache marker. Neither returns or keeps the index, so the first run's
    trap (`pathSegments`) is not here; the tests confirm it;
  * **`embedlit`,** `ModelCost{ModelRates: ModelRates{…}}` to
    `ModelCost{Input: …}`, in two `catalog` tests. Go 1.27 allows a
    promoted field as a key in a struct literal (the spec, "[Go 1.27] for
    a (possibly promoted) field");
  * **`newexpr`** and **`inline`,** `ptrBool(v)` to `new(v)`, in
    `band_0041_test.go`;
  * **`rangeint`,** `for i := 0; i < n; i++` with `i` unused, to `for range
    n`, in three live-tagged tests.
* **Found: three things `go fix` leaves wrong.** On a scratch clone with
  both `go fix` runs applied, vet passes for every platform and tag,
  `go test -race ./...` passes in 36 packages, and `make lint` exits 2:
  * `messages/cache.go` gets a second `import` declaration, which
    goimports rejects;
  * `ptrBool` is left with `//go:fix inline` and no caller, which `unused`
    rejects;
  * `sdkaudit` gets none of its four suggested fixes. `slicescontains` and
    `stringsseq` both claim `audit.go:108`'s loop, so `go fix` reports "1 of
    4 fixes skipped" on every run, exits 1, and changes nothing there. The
    other three are `stringsseq`, at `audit.go:185` and `main.go:141`.

  With these tidied by hand, on the same clone, every check passes:
  * the import merged into one block;
  * `ptrBool` deleted;
  * `strings.SplitSeq` at the three `sdkaudit` loops, which leaves
    `slicescontains` nothing to claim.

  The results: `go fix -diff ./...` prints nothing, with and without the
  tag; `make lint` reports 0 issues for the host and for Windows; `go test
  -race ./...` passes in all 36 packages; and `make docs-check` reports 0
  violations. That is 14 files, 40 lines in and 40 out.
* **Found: one rewrite no test guards.** Each production rewrite was
  planted back broken on the scratch clone, with its import kept in use. Six
  were caught by a failing test:
  * `cacheref.From` iterating forward: `TestFrom_LastValueWins_0048`;
  * the cache marker on the first tool: `TestAddCache` and the 0040 tests;
  * a deferred tool not skipped: `TestAddCache_Deferred_0040`;
  * `streamDetail.value` dropping its plain fields: `TestEvents_EqualsDecode`
    and `TestReasoningDetailsReplay_0053`;
  * `sdkaudit`'s two splits: its 0048 tests.

  One was not. `llmtest`'s stream relay (`stream_retry.go`) not copying the
  recorded headers at all passes every `llmtest` test. The rewrite keeps
  behaviour (`maps.Copy` is that loop), but nothing would see the line
  broken.
* **Proposed**, as the second run did ("Apply, then tidy"):
  * apply every rewrite, with the three hand tidies, as one change;
  * first add a test that `llmtest`'s relayed stream carries the headers
    `StreamText` set, seen failing on a scratch copy without the copy;
  * this record's rule stands: the modernizer's rewrites are adopted when
    they keep behaviour.
