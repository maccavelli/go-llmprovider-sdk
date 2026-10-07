---
status: proposed
date: 2026-10-07
decision-makers: repository owner
consulted: 0013-MADR-remediate-debugging-pass-findings.md (D5), 0026-MADR-remediate-v1-2-debugging-pass-findings.md (F12, amendments "Q6 is (b)" and "Gemini's Interactions API wraps its errors in an array")
informed: maintainers who run the live suites
---

<!-- markdownlint-disable MD013 -->

# Skip Live Tests by One Rule, and Measure Gemini's 429 on the Path the SDK Sends

## Context and Problem Statement

The release gate of
[0026-PLAN](0026-PLAN-remediate-v1-2-debugging-pass-findings.md) (its
deviations D11 and D12) ran the live suites for the first time since
`v1.2`, and found two gaps in the live tests. Neither is a defect in the
released code; each lets a live run say the wrong thing about it.

**1. Live tests skip transient failures by more than one rule.**
0013-MADR D5, as implemented by `skipIfTransient`
(`llmprovider/live_gateways_test.go`, exported to the external test package
as `llmprovider.SkipIfTransient`), sets the suite's rule: a rate limit, an
outage, an exhausted quota or a not-permitted refusal is the service's
state and skips; a 400 is a wire regression and fails.
`TestLiveTransient_SkipsOnlyTransientClasses` pins the helper. Ten skips in
nine live tests are written by hand instead:

| Site | Skips on | Narrower than the rule |
| :--- | :--- | :--- |
| `live_together_test.go:63` `TestLive_TogetherWire` | `ErrRateLimited` | yes |
| `live_together_test.go:99` `TestLive_TogetherToolChoices` | `ErrRateLimited` | yes |
| `live_claude_test.go:40`, `:63` | `ErrRateLimited` | yes |
| `live_opencode_test.go:155`, `:303` | `ErrRateLimited` | yes |
| `live_gemini_test.go:242-245` `TestLive_GeminiThinkingShapes` | `ErrProviderUnavailable`, `ErrRateLimited` | yes |
| `live_kilo_test.go:84` `TestLive_KiloReasoningShapes` | `ErrRateLimited`, `ErrProviderUnavailable` | yes |
| `live_together_test.go:129` `TestLive_TogetherRequiredFailsOnGptOss` | `ErrRateLimited` | on purpose: it asserts `ErrProviderUnavailable` |
| `live_kilo_test.go:110` `TestLive_KiloDataCollectionDenied` | `ErrRateLimited`, `ErrProviderUnavailable` | on purpose: it asserts `ErrNotPermitted` |

The cost was seen on 2026-10-07: `TestLive_TogetherWire` failed on
`together HTTP 503 service_unavailable`, classified correctly as
`ErrProviderUnavailable`, and passed when rerun alone. A live failure
that is the service's state costs a rerun and an investigation, and
teaches the reader to discount live failures. Nothing stops the next test
from writing its own skip.

**2. The only live check of F12 measures an endpoint the SDK does not
call.** `TestLive_GeminiRateLimitShape` (`live_gemini_429_test.go`, kept
opt-in by 0026-MADR's amendment "Q6 is (b)") bursts
`models/…:generateContent` with a raw HTTP client, then classifies the body
it captured with `ClassifyHTTPError("gemini", …)`. Gemini's `Generate`
calls only `POST /v1beta/interactions`, which sends its errors in a
one-element array (0026-MADR amendment of 2026-10-07; fixed in `v1.3.2`).
So the test:

* cannot see the Interactions API's envelope, and would have passed
  through D12, the defect that left F12 without effect in `v1.3.0` and
  `v1.3.1`;
* checks the classifier on a body the provider never receives, not the
  error a caller of `Generate` gets;
* decodes the body as a bare object, so pointed at Interactions its own
  shape check would report the array as DRIFT.

Reaching a 429 at all is unresolved: in 0026-PLAN's D4 the owner's key
answered two bursts of 20 one-token requests with 200.

## Decision Drivers

* A live failure should mean the SDK, or a change in the service's wire;
  the service's transient state skips (0013-MADR D5).
* One rule, in one helper, that a new live test cannot quietly bypass.
* A test whose subject is a transient class keeps its own skip, and says
  why.
* A live test measures the request the SDK sends and the error a caller
  receives, not a neighbouring endpoint.
* Every new check is seen failing first on a deliberately broken copy.
* No change to the shipped code, so no release.
* A live 429 costs money or a free-tier key; the run stays opt-in.

## Considered Options

* **A.** Route every live skip through `SkipIfTransient`, with a CI check
  that refuses a hand-written transient skip unless it is marked with its
  reason; and move the 429 test onto `Generate`'s own path.
* **B.** Fix the two observed sites only: `TestLive_TogetherWire`'s skip,
  and the 429 test's endpoint.
* **C.** Retry instead of skipping: give live providers `WithRetry`, so a
  transient failure is retried before it fails.
* **D.** Change nothing; record that a live failure needs a rerun before it
  is believed.

## Decision Outcome

Chosen option: "A", because it makes the suite's rule (0013-MADR D5) hold
in every live test and keeps it holding, at the price of one source-reading
test, and because a check of F12 that cannot see the Interactions API's
envelope is not a check of what callers get.

1. **Transient skips.**
   * The eight narrower sites call `llmprovider.SkipIfTransient(t, err)`.
   * The two tests whose subject is a transient class keep their own skip,
     preceded by a `// live-skip: <reason>` comment.
   * A new test, `TestLiveTests_TransientSkipsUseTheHelper`, with no build
     tag so CI runs it, parses every `live_*_test.go` under `llmprovider/`.
     It fails on an `if` or `case` whose condition names `ErrRateLimited`,
     `ErrProviderUnavailable`, `ErrQuotaExhausted` or `ErrNotPermitted` and
     whose body calls `Skip` or `Skipf`, unless a `// live-skip:` comment
     with a reason directly precedes it.
2. **Gemini's 429 on the Interactions path.**
   * `TestLive_GeminiRateLimitShape` sends its burst through
     `gemini.New(…).Generate`, with `WithMaxTokens(1)`, and a recording
     transport that keeps the first 429's header and body.
   * It checks the error `Generate` returned: an `*APIError` whose
     `RetryAfter` is the body's `retryDelay` when no `Retry-After` header
     came, and which is `ErrQuotaExhausted` exactly when a violation names a
     per-day quota.
   * Its DRIFT checks read the body through the same one-element-array rule
     as the SDK, and report the envelope it found: bare or wrapped.
   * Those checks move to a function in an untagged test file, unit-tested
     on a fixture of Google's documented 429 wrapped in the array, so CI
     runs them and they can be seen failing without a live 429.
3. **Q1, below,** decides whether the plan attempts a live 429.

### Q1. How the plan attempts a live 429

* **(a) A burst size the owner sets** *(recommended)*.
  `LLMPROVIDER_LIVE_GEMINI_429_BURST` overrides the default of 20. Before
  any burst, one request is sent and its usage reported, so the owner sees
  the cost of one request and picks the burst knowing it. No 429 is a skip,
  recorded with the burst size.
* **(b) No attempt.** The test is corrected and stays opt-in; F12 remains
  checked against Google's documented shape only (0026-MADR, "Q6 is (b)").
* **(c) A free-tier key.** Its lower limit makes a 429 likely within a
  small burst; it needs the owner to provide one, as D4 found.

### Consequences

* Good, because a live failure from a rate limit, an outage, a quota or a
  not-permitted refusal skips in every test, and a 400 still fails.
* Good, because a new hand-written transient skip fails CI, naming its file
  and line.
* Good, because the 429 test checks the error `Generate` returns, through
  the envelope `Generate` receives.
* Neutral, because `SkipIfTransient` skips more at the eight sites than they
  skipped before: a quota or not-permitted refusal now skips, where it
  failed. That is the suite's rule; a test that wants one of those classes
  to fail must say so with `// live-skip:`.
* Neutral, because no shipped code changes, so there is nothing to tag.
* Bad, because a test reads source files, and a skip written in a shape it
  does not parse (a helper of the test's own) passes it unnoticed.
* Bad, because a live 429 is still not guaranteed; under Q1 (a) or (b), F12
  can remain unmeasured on a real reply.

### Confirmation

* `TestLiveTests_TransientSkipsUseTheHelper` fails on today's tree, naming
  the eight sites, and passes after them. On a scratch copy it fails on a
  planted hand-written skip, and on a `// live-skip:` comment with no
  reason.
* The 429 checker's unit test fails on a scratch copy with `soleObject`
  (`llmprovider/api_error.go`) planted out.
* `go vet -tags live_gateways` is clean on every platform, and the
  converted live tests pass, or skip by the rule, in one live run.

## Pros and Cons of the Options

### A. One helper, a CI check, and the 429 test on `Generate`'s path

* Good, because it fixes every narrower site, not only the one that failed.
* Good, because the check keeps the rule from drifting again.
* Good, because the 429 test's shape and classification checks run in CI.
* Bad, because the source-reading check is a heuristic over Go syntax.

### B. The two observed sites only

* Good, because it is the smallest change.
* Bad, because seven narrower sites stay, and fail the same way when their
  service has an outage.
* Bad, because nothing stops the next one.

### C. Retry instead of skip

* Good, because a brief outage passes rather than skips.
* Bad, because it changes what the tests measure: `WithRetry` is a caller's
  option, and the live tests check the provider without it.
* Bad, because a long outage still fails, and makes the run slower first.
* Bad, because tests that assert a transient class would need it turned
  off.

### D. Nothing

* Good, because it costs nothing now.
* Bad, because each live run's failures need rerunning before they are
  believed, as on 2026-10-07.
* Bad, because the 429 test stays blind to the envelope that hid D12.

## More Information

### Relationship to other records

* **0013-MADR-remediate-debugging-pass-findings.md** D5 set the rule this
  record enforces. This record does not change it.
* **0026-MADR-remediate-v1-2-debugging-pass-findings.md** owns F12 and
  D12. Its PLAN is complete, and records that fixes after `v1.3.2` go
  forward under a new record; this is that record. Both findings were
  found in that PLAN's live run (its D11).

### Left out

* **Gemini's `llmtest` harness `Error`** still writes a bare object where
  the Interactions API sends the array. The checks that use it classify by
  status, not body, so changing it would add no check that can fail. 0026
  moved its `AuthFailure`, which does read the body, to the array.
* **The subscription and sign-in suites** (`LLMPROVIDER_LIVE_CHATGPT`,
  `…_GROK_CLI`, `…_BROWSER_LOGIN`, `…_DEVICE_LOGIN`, `…_OPENAI_SIGNIN`)
  were not run by 0026's gate. Running them needs the owner present, and
  is not a gap in the tests.
