---
status: complete
date: 2026-10-02
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
