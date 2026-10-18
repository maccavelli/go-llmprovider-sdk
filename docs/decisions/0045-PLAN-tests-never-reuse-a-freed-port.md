---
status: proposed
date: 2026-10-18
associated-madr: "0045-MADR-tests-never-reuse-a-freed-port.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement Making Tests Fail to Connect Without Reusing a Freed Port

Associated MADR: [0045-MADR-tests-never-reuse-a-freed-port.md](0045-MADR-tests-never-reuse-a-freed-port.md)

## Goal

No test dials a freed ephemeral port, and none hands one to code that
binds it. CI passes on `main` on all three platforms, and `v1.15.0` can be
tagged.

## Scope

### In scope

* `internal/nettest`: `Refusing`, `RefusedURL` and `Port`, with their own
  tests (D1, D2).
* The sites of the MADR's table: P1 (six tests), P2 (`refusedURL`, two
  tests), P3 (`listenOnFreePort`, `newPlanLogin`), P4 (one test).
* `docs/guides/api-standards.md` R2 and `docs/architecture.md`: the
  package's row (D3).

### Out of scope

* Any production file, wire golden or exported API.
* The tests that already avoid the race (the MADR's "Already safe").
* The live tests, which bind 1455 and 1457 only when switched on.

## Preconditions and conventions

* **Approval.** The owner accepts the MADR and approves this PLAN.
* **Gate.** Each phase ends with:
  * `tools/gate.py`;
  * `make pre-add-check` on its Go files;
  * `tools/apidiff.py v1.14.0`, unchanged from `6aefe73`;
  * `tools/denyscan.py --changed`;
  * `make records-check`.

  It is then staged for the owner's `git commit --no-edit`.
* **Red first.** `nettest`'s tests, and every changed test, are each seen
  to fail on a planted break by `tools/plant.py` on a scratch clone.
* **Stop conditions:**
  * a changed test can no longer assert what it asserted (R46);
  * a production file would have to change;
  * `apidiff` shows any change.

## Implementation Steps

### Phase 1: the package and the sites

1. **`internal/nettest`** (package doc, R41):
   * `RefusedURL = "http://127.0.0.1:1"`.
   * `Refusing() *http.Client`: an `*http.Transport` whose `DialContext`
     returns `&net.OpError{Op: "dial", Net: "tcp", Err:
     syscall.ECONNREFUSED}`.
   * `Port(t testing.TB) int`:
     * a random port in `[20000, 30000)` that binds on `127.0.0.1`, and on
       `[::1]` when the host has IPv6 loopback, then released;
     * never the same port twice in a process;
     * passes over a port that fails to bind;
     * fails the test after 100 tries.
   * **Tests:**
     * the error chain (`*url.Error`, `net.Error`,
       `syscall.ECONNREFUSED`), with no dial;
     * `Port`'s range, distinctness and bind;
     * a held port passed over.
2. **P1.** Each test passes `nettest.Refusing()` and uses
   `nettest.RefusedURL` (or keeps its server for the requests it answers):
   * `auth/oauth_error_kinds_test.go`: the session's `HTTPClient`;
   * `providers/together/reply_test.go`: the counter wraps `Refusing`'s
     transport;
   * `providers/ollama/listing_0026_test.go`: the unreachable case;
   * `catalog/steady_0041_test.go`: the second listing of
     `TestList_StaleOn503_0041/network`, and `TestList_OllamaOutsideD6_0041`.
     The listing cache's key has no client (`listingKeyFor`), so the stale
     listing is still found;
   * `internal/wire/post_0028_test.go`.
3. **P2.** `wizard/configure_test.go`: `refusedURL` gives way to
   `nettest.RefusedURL`, and the two tests pass `Options.HTTPClient:
   nettest.Refusing()`.
4. **P3, P4.** `listenOnFreePort`'s closed uses, `newPlanLogin` and
   `TestListenLoopbackBothFamilies_IPv4OnlyMissesIPv6Localhost` take their
   ports from `nettest.Port`. Where a listener is held open to make a port
   busy, it stays.
5. **Docs.** The R2 row and the architecture tree and table line for
   `internal/nettest`.
6. **Search.** No test file calls `.Close()` on a server or listener and
   then uses its address, or binds `:0` to choose a port for later.

### Phase 2: CI and the release

1. The owner commits and pushes. CI passes on all three platforms.
2. The owner tags `v1.15.0`. Then 0043-PLAN step 5 and 0044-PLAN Phase 2
   step 3 run: the proxy, `tools/consumer.py`, `tools/offline.py`, and
   issues 2 and 3.

## Verification

| # | Criterion | Check | Phase |
| :--- | :--- | :--- | :--- |
| V1 | `nettest` does what D1 and D2 say | its tests, red first | 1 |
| V2 | Every changed test asserts what it asserted | the plant table | 1 |
| V3 | None of P1–P4 is left | step 6's search | 1 |
| V4 | The changed packages pass together, many times | `go test -count=20 -race` over them at once | 1 |
| V5 | No production, API or golden change | `git diff --stat`; `tools/apidiff.py` | 1 |
| V6 | CI passes on all three platforms | the run on the commit | 2 |

## Rollout and Rollback

* Tests only: a caller sees no change, and no release carries this alone;
  it goes out with `v1.15.0`.
* Rollback: revert the commit; the races come back.
