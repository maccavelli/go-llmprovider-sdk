---
status: in-progress
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

## Execution record

The owner committed the pair as `943211e`, accepted the MADR and approved
this PLAN ("committed, approved, proceed", 2026-10-18).

### Phase 1: the package and the sites (2026-10-18)

* **No deviation.**
* **Step 1, `internal/nettest`** (`nettest.go`, `nettest_test.go`):
  * `RefusedURL`; `Refusing()`, whose `DialContext` returns a dial
    `*net.OpError` around `syscall.ECONNREFUSED`;
  * `Port(t)`: a random port in `[20000, 30000)` that binds on `127.0.0.1`,
    and on `[::1]` where the host has it, both at once. It is never handed
    out twice in a process, a port that fails to bind is passed over, and
    it fails its test after 100 tries.
  * Five tests:
    * the refusal's chain, with a listening server never reached;
    * 50 distinct ports in the range that bind;
    * a held port and a given port passed over;
    * a port held on `[::1]` only passed over;
    * the failure after 100 tries.

    Coverage 97.0 %; only the branches for a probe listener that fails to
    close are not run. `make pre-add-check` first refused two `_ =
    l.Close()` in `nettest.go` (errcheck). Now a port whose probe does not
    close cleanly is not bindable, and IPv6 loopback counts only if its
    probe closes. The 8 `nettest` plants were run again after that change,
    and all were caught.
* **Steps 2-4, the sites:**

  | Site | Now |
  | :--- | :--- |
  | `auth/oauth_error_kinds_test.go` (P1) | the session's `TokenURL` is `RefusedURL`, its `HTTPClient` `Refusing()` |
  | `providers/together/reply_test.go` (P1) | the counter wraps `Refusing()`'s transport; the base URL is `RefusedURL` |
  | `providers/ollama/listing_0026_test.go` (P1) | the unreachable case is `RefusedURL` with `Refusing()` |
  | `catalog/steady_0041_test.go`, `/network` (P1) | the server stays open; the second listing passes `WithHTTPClient(Refusing())`, and the stale listing is still found, since the cache's key has no client |
  | `catalog/steady_0041_test.go`, Ollama down (P1) | the same, with `WithHTTPClient(Refusing())` |
  | `internal/wire/post_0028_test.go` (P1) | `postCall(RefusedURL)` with `Client` `Refusing()` |
  | `wizard/configure_test.go` (P2) | `refusedURL` is removed; two tests use `RefusedURL` and `Options.HTTPClient: Refusing()` |
  | `auth/oauth_loopback_test.go` (P3) | the second port of `UsesSecondPortWhenFirstBusy`, the ports of `LocalhostDials`, `AcceptsIPv6Loopback` and `OpenAICompletesCallbackAndExchange` come from `Port` |
  | `auth/oauth_loopback_test.go` (P4) | `IPv4OnlyMissesIPv6Localhost` takes its port from `Port` and holds it on `tcp4` |
  | `auth/plan_login_0043_test.go` (P3) | `newPlanLogin`'s two ports come from `Port` |

  `goimports` added the import and removed the unused ones. The listeners
  that are held open to make a port busy stay.
* **Step 5, docs:** `docs/guides/api-standards.md` R2 and
  `docs/architecture.md` (the tree and the table) gain `internal/nettest`.
* **Step 6, the search** (`grep` over every `_test.go`):
  * no `httptest` server or listener is closed and then dialled;
  * no `:0` bind is closed to choose a port.

  Six `srv.Close()` calls that are not deferred remain: in gemini, claude,
  grok, openai, `catalog/raw_0041_test.go` and
  `internal/wire/stream_0031_test.go`. Each closes its server after the
  last request to it, which the MADR's search found too, and two were read
  again.
* **Red first.** `tools/plant.py`, on scratch clones:
  * **The changed tests, 10 plants,** each breaking what a test asserts.
    All were caught. One, "both families bind any port but the one asked",
    first failed by not building (an unused `portText`); it was rewritten
    to keep the build, and caught alone.

    | Plant | Caught by |
    | :--- | :--- |
    | a refresh's send failure has no kind | `TestOAuthRefresh_EveryFailureHasAKind/transport_failure` |
    | a send failure is not retried | `TestGenerate_AnsweredOnceIsNotBoughtAgain` (`1 attempt(s), want 3`) |
    | an unreachable Ollama has no kind | `TestOllama_ListModelsKinds`, `TestList_OllamaOutsideD6_0041` |
    | a network failure serves no stale listing | `TestList_StaleOn503_0041/network` |
    | a failure before writing is marked after the reply | `TestPost_FailureBeforeWriteIsNot`, `TestGenerate_AnsweredOnceIsNotBoughtAgain` |
    | the wizard accepts an endpoint it cannot reach | both wizard tests (`1 scripted confirms left unused`) |
    | the OpenAI loopback tries only its first port | `TestListenOpenAILoopback_UsesSecondPortWhenFirstBusy` |
    | both families bind any port but the one asked | `…_LocalhostDials`, `…_AcceptsIPv6Loopback` (second run) |
    | the IPv4-only listener takes IPv6 too | `…_IPv4OnlyMissesIPv6Localhost` |
    | the OpenAI redirect names localhost | `TestLoginBrowserOAuth_OpenAICompletesCallbackAndExchange`, `TestLoginChatGPTPlan_FirstSignIn_0043` |

    The refused requests' errors read `dial tcp: connection refused`, as a
    refusal from the OS does.
  * **`nettest`, 8 plants,** all caught:
    * `Refusing` dials for real;
    * the refusal is not `ECONNREFUSED`;
    * it is not a dial error;
    * `Port` picks from the ephemeral range;
    * hands a port out twice;
    * does not check the bind;
    * ignores IPv6;
    * does not fail its test.

    The test for a port held on IPv6 only was added before the plants for
    this reason: without it, "ignores IPv6" would have passed.
* **V4, the stress run:** `go test -count=20 -race` over the seven changed
  packages at once: every package passed (catalog in 843 s, auth in 146 s). It started before `TestPort_PassesOverIPv6Held` and the errcheck change to `nettest.go`; `nettest` passed again after both, alone and in the gate.
* **V5:** `git diff --stat` touches test files, the new package, three
  Markdown files and the records; no production file or golden.
* **Gate.**
  * `make pre-add-check` on the ten Go files: `10 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)`, after the errcheck change; `tools/gate.py`: 23 of 23, `gate-selftest` among them.
  * `tools/apidiff.py v1.14.0`: unchanged from `6aefe73` (12 lines).
  * `tools/denyscan.py --changed`: 0 hits. `make records-check`: 0
    problems.
* **Note:** `go install golang.org/x/tools/cmd/goimports@latest` ran once
  to make sure `goimports` was present in the home directory's `go/bin`;
  it writes nothing in the tree.

