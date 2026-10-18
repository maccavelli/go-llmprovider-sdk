---
status: accepted
date: 2026-10-18
decision-makers: repository owner
consulted: issue 3 (gobble-cli's request, 2026-10-16); this module at 180f01e; `tools/apidiff.py v1.14.0` on two scratch clones (2026-10-18)
informed: every caller of `WithRetry`, gobble-cli first
---

<!-- markdownlint-disable MD013 -->

# Let a Caller See Each Retry That WithRetry Waits For

## Context and Problem Statement

`WithRetry` (0015-MADR-canonical-sdk-api-and-module-layout.md D8) retries
where its caller cannot see it. `RetryPolicy` has only `MaxAttempts`,
`BaseDelay` and `MaxDelay` (`llmprovider/retry.go:15-27`), and
`retrying.Generate` waits inside its own loop (`retry.go:101-137`). A caller
learns about the retries only when `Generate` returns, perhaps a minute
later, with the final error.

[Issue 3](https://github.com/maccavelli/go-llmprovider-sdk/issues/3), filed
for gobble-cli, asks for a way to see each retry without changing how
`WithRetry` decides. gobble-cli wraps every provider in `WithRetry(p,
RetryPolicy{MaxAttempts: 4, BaseDelay: 2 * time.Second, MaxDelay: 60 *
time.Second})`. If a 429 asks for a 50 s wait, its user sees nothing for 50
s. gobble cannot:

* say that a retry is pending, or why;
* report each attempt to an ACP client, as pi's `auto_retry_start` events
  do;
* tell a wait that will end from a request that has hung.

pi, opencode and Kilo all show their users this. The only way to get it
today is to drop `WithRetry` and rebuild its loop in the consumer: the
classification (`retryable`), the reading of `Retry-After` and the reset
headers, the single retry after a reply, the deadline check and the
jitter. That copy would drift from the SDK.

The issue sketches an optional func field on `RetryPolicy`, `OnRetry
func(ctx, RetryNotice)`, and leaves the shape to the SDK.

### What is measured

| # | Fact | Source |
| :--- | :--- | :--- |
| E1 | A field `OnRetry func(context.Context, int)` added to `RetryPolicy` builds, and `tools/apidiff.py v1.14.0` exits 1: `./llmprovider.RetryPolicy: old is comparable, new is not` | scratch clone of `180f01e`, 2026-10-18 |
| E2 | A field `Observer interface{ OnRetry(context.Context, int) }` builds, and `tools/apidiff.py v1.14.0` exits 0: `./llmprovider.RetryPolicy.Observer: added` | same |
| E3 | `WithRetry`'s result has no `Stream`, so `llmprovider.Stream` falls back to its `Generate` (`stream.go:45-50`; 0031-MADR D6) | code |
| E4 | `APIError.RetryAfter` is set from `Retry-After` / `retry-after-ms`, else from the rate-limit reset headers (`api_error.go:226`, `:245`); `RetryPolicy.wait` uses it when it is above zero (`retry.go`, `serverRetryAfter`) | code |

E1 is the same failure that
0021-PLAN-harden-and-tune-after-the-v1-1-review.md recorded on 2026-10-04,
and that 0043-MADR-openai-chatgpt-plan-sign-in.md A1 avoided: under R48, an
incomparable struct is an incompatible change.

## Decision Drivers

* **Additive only** (R48): the release stays a minor version, and
  `RetryPolicy` stays comparable (E1).
* **No change to how `WithRetry` decides:** the same retries, waits,
  jitter, errors and `Capabilities`, and the same wire (R45). A caller that
  asks for no notices gets exactly today's behaviour.
* **Middleware, not hooks inside providers** (R28): the notice comes from
  the wrapper, which is where the wait happens.
* **The notice says what a consumer shows:** the attempt, out of how many,
  how long, whether the service asked for that wait, and why. It comes with
  the request's `ctx`, so a caller can tell which request it belongs to.
* **Least new exported API** (`docs/guides/api-standards.md`).

## Considered Options

* **A. A second constructor,** `WithRetryNotify(p, policy, notify)`, which
  takes the callback as an argument.
* **B. An interface field on `RetryPolicy`,** `Notify RetryNotifier`.
* **C. A func field on `RetryPolicy`,** `OnRetry`, as the issue sketches.
* **D. A context value,** `ContextWithRetryNotify(ctx, fn)`, which the
  wrapper reads on each `Generate`.

## Decision Outcome

Chosen option: "A. A second constructor", because it keeps `RetryPolicy` a
comparable value of numbers (E1), adds two names, and puts the callback
where the wrapper is built. B passes the check (E2), but `==` on two
policies that hold a func adapter panics at run time. C needs a major
version. D hides control flow in a context.

### D1. The API

```go
// RetryNotice describes a retry that WithRetryNotify is about to wait for.
type RetryNotice struct {
	Provider       ProviderID    // the wrapped provider's ID
	Attempt        int           // the try that failed, the first being 1
	MaxAttempts    int           // the policy's, after its defaults
	Wait           time.Duration // the wait about to begin, jitter included
	ServerDirected bool          // the service asked for the wait (E4)
	Err            error         // the failure being retried
}

// WithRetryNotify is WithRetry, and calls notify before each wait.
func WithRetryNotify(p Provider, policy RetryPolicy, notify func(context.Context, RetryNotice)) Provider
```

* `WithRetry(p, policy)` is `WithRetryNotify(p, policy, nil)`: the same
  wrapper type, which keeps `ModelLister` when `p` has it (0020-MADR F32).
* `Err` is the inner provider's error, as `Generate` would return it if it
  stopped there: already classified and redacted (R25, R35).
* `ServerDirected` is true when the wait came from the error's
  `RetryAfter`, which covers both `Retry-After` and the reset headers (E4).

### D2. When notify is called

* Only when a wait is about to begin: after `retryable`, the single retry
  after a reply, `MaxAttempts`, `MaxDelay` and the deadline have all let
  the retry through. Each notice is followed by a wait and then another
  try, unless `ctx` ends during the wait.
* It is never called for a success, a terminal error, the last attempt, a
  wait longer than `MaxDelay`, a wait that would outlast `ctx`'s deadline,
  or a second failure after a reply. In each of those cases `Generate`
  returns, and its return is the end of the retry.
* It runs on the goroutine of `Generate`, with that call's `ctx`. The wait
  begins when it returns, so a notify that blocks delays the retry by as
  long. A panic in it is not recovered. The doc comment says it must not
  block.
* `ctx` ending during the wait is reported as today:
  `errors.Join(ctx.Err(), err)`. No second notice is sent.

### D3. Streaming

`WithRetry`'s result buffers a stream (E3), so `llmprovider.Stream` over
`WithRetryNotify` gets the notices before its first event. Nothing about
streaming changes. A retry that keeps a stream live after its first event
is out of scope, as the issue says.

### D4. Docs

`docs/guides/api-standards.md` R28 names `WithRetryNotify` next to
`WithRetry`. The README's middleware lines, `docs/architecture.md`'s
`WithRetry` lines and the release notes say what it adds. No layout
changes.

### Consequences

* Good, because a consumer can show "rate limited, retrying in 4 s
  (attempt 2 of 4)" without copying the retry loop.
* Good, because `RetryPolicy` stays a comparable value, and the release is
  a minor version.
* Good, because a caller that passes no notify, and every caller of
  `WithRetry`, sees no change.
* Neutral, because there are two constructors for one wrapper.
* Bad, because a notify that blocks delays the retry. This is documented,
  not guarded.

### Confirmation

* Tests with a fake provider:
  * a 429 with `RetryAfter`, then a success, sends one notice with every
    field, `ServerDirected` true;
  * backoff without `RetryAfter` sends `ServerDirected` false, and the
    `Wait` lies within the equal-jitter range;
  * no notice for a terminal error, the last attempt, a wait over
    `MaxDelay`, a wait past the deadline, or a second failure after a
    reply;
  * the `ctx` that notify gets is the caller's;
  * a notify that cancels `ctx` ends the call with no further try;
  * a nil notify behaves as `WithRetry`, whose existing tests pass
    unchanged;
  * `ModelLister` is kept;
  * notices arrive through `llmprovider.Stream`.
* Each test is first seen to fail on a planted break (`tools/plant.py`).
* `make api-check` reports additions only, and `tools/apidiff.py` lists
  exactly `RetryNotice`, its six fields and `WithRetryNotify`.

## Pros and Cons of the Options

### A. A second constructor

* Good, because `RetryPolicy` is unchanged and stays comparable.
* Good, because the callback is fixed when the wrapper is built, like
  everything else about it (R20).
* Bad, because a caller that wants notices changes its call from
  `WithRetry` to `WithRetryNotify`.

### B. An interface field on `RetryPolicy`

* Good, because it passes `api-check` (E2), and one constructor serves
  both uses.
* Bad, because the natural adapter is a func type, and `==` on two
  policies holding one panics at run time. A caller that compares policies
  today, as a map key or in a test, could break in a minor release.
* Bad, because a policy stops being plain data.

### C. A func field on `RetryPolicy`

* Good, because it is the issue's own sketch, and the most direct shape.
* Bad, because `RetryPolicy` becomes incomparable, which `api-check`
  refuses (E1). It would need `v2`.

### D. A context value

* Good, because notices are per request and pass through any stack of
  wrappers.
* Bad, because the behaviour hides in `ctx`, the compiler checks nothing,
  and a caller that forgets the value gets silence. It is also the only
  context-carried callback in the module.

## More Information

* The request: [issue 3](https://github.com/maccavelli/go-llmprovider-sdk/issues/3).
  Its "Not asked for here" (a streaming retry; a network-down sentinel)
  stays out of scope.
* The implementation: [0044-PLAN-observe-retries.md](0044-PLAN-observe-retries.md).
* Related: 0015-MADR D8 (`WithRetry`); 0021-MADR D1 and T6 (the single
  retry after a reply; the deadline); 0026-MADR F22 (jitter); 0031-MADR D6
  (the buffered stream).
