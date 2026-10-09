---
status: proposed
date: 2026-10-09
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
