---
status: accepted
date: 2026-10-02
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
