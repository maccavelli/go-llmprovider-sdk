---
status: accepted
date: 2026-10-18
decision-makers: repository owner
consulted: CI runs 39177122292 (6aefe73, 2026-10-18) and 38569759839 (7f34e31, 2026-10-14); a read-only search of every test file at 6aefe73
informed: anyone who writes a test in this module
---

<!-- markdownlint-disable MD013 -->

# Make Tests Fail to Connect Without Reusing a Freed Port

## Context and Problem Statement

CI failed on `6aefe73`, the commit that `v1.15.0` was to be tagged on (run
39177122292, ubuntu-24.04). The code under test was not at fault: two
tests in two packages met on one loopback port.

* `TestGenerate_AnsweredOnceIsNotBoughtAgain`
  (`llmprovider/providers/together/reply_test.go:71-86`) wants a refused
  connection. It starts an `httptest` server, keeps its URL and closes it.
  The OS gave the freed port to the server of `TestWireGoldens` in
  `providers/openai`, which ran at the same time in the gates' self-test.
* Together's request reached that server: `a refused connection: 1
  attempt(s), want 3`.
* The openai golden recorded it: `requests[0].path: want "/responses", got
  "/chat/completions"`, `model: want "gpt-5.5", got "m"`.

CI failed on `7f34e31` (run 38569759839, 2026-10-14) the same way, through
the other door. `TestListenOpenAILoopback_UsesSecondPortWhenFirstBusy`
picks a port by binding `127.0.0.1:0` and closing it, then has the code
bind that port. Another package's server was given it first: `listen tcp4
127.0.0.1:40611: bind: address already in use`.

`go test ./...` runs packages in parallel, and the OS hands out a freed
ephemeral port again at once. A test that relies on such a port staying
unused can fail, and can make another package's test fail.

### What the search found (6aefe73)

| # | Pattern | Sites |
| :--- | :--- | :--- |
| P1 | A closed `httptest` server's URL dialled to provoke a failure | `auth/oauth_error_kinds_test.go:17-25`; `providers/together/reply_test.go:71-86`; `providers/ollama/listing_0026_test.go:22-31`; `catalog/steady_0041_test.go:187-205` and `:395-399`; `internal/wire/post_0028_test.go:39-48` |
| P2 | A port bound, closed, and dialled to provoke a failure | `wizard/configure_test.go:419-431` (`refusedURL`, two tests) |
| P3 | A port bound, closed, and handed to code that binds it | `auth/oauth_loopback_test.go:701-708` (`listenOnFreePort`: four tests); `auth/plan_login_0043_test.go:51-60` (`newPlanLogin`: nine tests) |
| P4 | A refused IPv6 dial on a port nobody holds | `auth/oauth_loopback_test.go:366-376` |

Already safe:

* The dials to `127.0.0.1:1`, a privileged port outside every ephemeral
  range.
* The listeners held open while the test asserts.
* `roundTripFunc` and `failingTransport` clients, which never dial.
* `TestBrowserListener_OpenAIRedirectsToLoopbackIP`, which binds the real
  ports 1455 and 1457. Both are below every ephemeral range, and the test
  skips when they are busy.

The Python under `tools/` and `scripts/` binds no port.

## Decision Drivers

* **CI is deterministic.** A test's result depends on its own package,
  never on another package's timing.
* **Each test keeps its meaning (R46).** A test of a refused connection
  still sees the error chain a refused dial gives the SDK: a `*url.Error`
  around a `*net.OpError`. A test of the port binding still binds real
  ports.
* **No production change.** Neither the wire, the API nor a golden
  changes.
* **One helper, not six copies.**

## Considered Options

* **A. A test-support package, `internal/nettest`.** Its client refuses at
  dial, and it hands out ports that no `:0` bind can be given.
* **B. Dial `127.0.0.1:1`** for P1, P2 and P4, as some tests already do;
  pick P3's ports as now.
* **C. A listener held open** that accepts each connection and closes it.
* **D. Run packages one at a time in CI** (`go test -p 1`).

## Decision Outcome

Chosen option: "A. A test-support package, `internal/nettest`", because
it removes both races at their cause, keeps every test's meaning, and
holds the rule in one place.

### D1. `nettest.Refusing()`, for a connection that must fail

`internal/nettest` gives an `*http.Client` whose `*http.Transport` dials
nothing. Its `DialContext` returns `&net.OpError{Op: "dial", Net: "tcp",
Err: syscall.ECONNREFUSED}`. `net/http` wraps that as it wraps a real
refusal, so the SDK sees a `*url.Error` around a `net.Error`. `WithRetry`
retries it, `transport.Unsendable` does not match it, and
`transport.IsAfterReply` does not mark it.

* P1 and P2 pass this client, through the option or field each already
  takes: `WithHTTPClient`, `OAuthSession.HTTPClient`, `Options.HTTPClient`
  or the wire test's client.
* Their URL becomes `nettest.RefusedURL`, `http://127.0.0.1:1`, which is
  never dialled. Even a client that lost the option could not reach
  another package's server there.

### D2. `nettest.Port(t)`, for a port the code will bind

`Port` returns a loopback port in `[20000, 30000)`:

* That range is below the ephemeral ranges of Linux (`32768–60999`),
  macOS and Windows (`49152–65535`), so no `:0` bind in any package is
  given one of its ports.
* `Port` picks at random, checks that the port binds on `127.0.0.1` (and
  on `[::1]` when the host has IPv6 loopback), and releases it. A port
  that fails to bind, such as one that Windows reserves for Hyper-V, is
  passed over.
* A port it has handed out is never handed out again in the process, so
  parallel tests of one package do not share one.

P3's helpers and P4 take their ports from it. The tests still bind real
ports and assert what they assert now.

### D3. Scope

* Tests only, plus the new package and its own tests. No production file,
  wire golden or exported API changes.
* `docs/guides/api-standards.md` R2 and `docs/architecture.md` gain the
  package's row.
* The package has test files of its own and holds the 80 % floor on its
  own tests, so `scripts/coverage-floors.txt` does not change.

### Consequences

* Good, because neither race can happen: no test dials a freed ephemeral
  port, and none hands one to code that binds it.
* Good, because the refused-connection tests no longer depend on how fast
  an OS refuses.
* Neutral, because a refused connection is now produced in the test's
  transport, not by the OS. What the SDK receives is the same, and the
  live tests still meet real networks.
* Bad, because a new internal package needs its own row and its own tests.

### Confirmation

* Every changed test passes, and fails on a planted break of what it
  asserts (`tools/plant.py`), as before.
* `nettest`'s own tests: `Refusing` gives a `*url.Error` around a
  `net.Error` that matches `syscall.ECONNREFUSED`, with no dial; `Port`
  gives distinct ports in its range that bind, and passes over one that is
  held.
* A search of the test files at the phase's end finds none of P1–P4.
* A stress run of the changed packages, together and in parallel, many
  times, passes.
* CI passes on all three platforms.

## Pros and Cons of the Options

### A. A test-support package

* Good, because both causes go, with one rule.
* Good, because the error chain is the one a refused dial gives.
* Bad, because a package is added.

### B. Dial `127.0.0.1:1`

* Good, because the repository already does it, and the OS refuses for
  real.
* Bad, because P3, which binds ports, is not fixed.
* Bad, because it depends on nothing listening on port 1.

### C. A listener that accepts and closes

* Good, because the port is never freed.
* Bad, because a reset after the connection opens is not a refusal: what
  P1's tests assert (a failure before the request was sent) changes.

### D. One package at a time in CI

* Good, because nothing in the code changes.
* Bad, because it hides the race rather than removing it, slows CI, and
  leaves `go test ./...` on a developer's machine as flaky as before.

## More Information

* The failures:
  [run 39177122292](https://github.com/maccavelli/go-llmprovider-sdk/actions/runs/39177122292)
  and
  [run 38569759839](https://github.com/maccavelli/go-llmprovider-sdk/actions/runs/38569759839).
* The implementation:
  [0045-PLAN-tests-never-reuse-a-freed-port.md](0045-PLAN-tests-never-reuse-a-freed-port.md).
* `v1.15.0` (0043-PLAN-openai-chatgpt-plan-sign-in.md and
  0044-PLAN-observe-retries.md) is tagged once CI passes on this record's
  commit.
