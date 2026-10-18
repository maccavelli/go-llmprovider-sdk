---
status: in-progress
date: 2026-10-18
associated-madr: "0044-MADR-observe-retries.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement Letting a Caller See Each Retry That WithRetry Waits For

Associated MADR: [0044-MADR-observe-retries.md](0044-MADR-observe-retries.md)

## Goal

A caller wraps a provider with `WithRetryNotify(p, policy, notify)`, and
gets one `RetryNotice` before each wait that `WithRetry` would take. Every
retry decision, wait, error and wire request is as before, and `WithRetry`
does not change.

## Scope

### In scope

* `llmprovider/retry.go`: `RetryNotice`, `WithRetryNotify`, and the call in
  `retrying.Generate` (D1, D2).
* Tests in `llmprovider/`, with a fake provider.
* Docs: api-standards R28, the README, `docs/architecture.md`, and the
  release notes (D4).

### Out of scope

* A retry that keeps a stream live after its first event; a sentinel for
  a network that is down (issue 3, "Not asked for here").
* Any change to `RetryPolicy`, `retryable`, `wait` or `Capabilities`.

## Preconditions and conventions

* **Approval.** The owner accepts the MADR and approves this PLAN.
* **Order.** This pair is independent of
  0043-PLAN-openai-chatgpt-plan-sign-in.md and touches none of its files.
  It may go before, between or after that PLAN's phases, as the owner
  chooses.
* **Gate.** Each phase ends with:
  * `tools/gate.py`;
  * `make pre-add-check` on its Go files;
  * `tools/apidiff.py v1.14.0`;
  * `tools/denyscan.py --changed`;
  * `make records-check`.

  It is then staged for the owner's `git commit --no-edit`.
* **Red first.** Every new test is first seen to fail on a planted break,
  by `tools/plant.py` on a scratch clone. The plant table goes in this
  record.
* **Stop conditions:** an existing retry test changes; `api-check` reports
  anything but additions; a wire golden changes.

## Implementation Steps

### Phase 1: the notice (D1, D2, D3)

1. Record `<base>`, `tools/gate.py` at `<base>`, and `llmprovider`'s
   coverage.
2. Add `RetryNotice` and `WithRetryNotify` to `llmprovider/retry.go`, with
   doc comments (R41).
   * `retrying` gains an unexported `notify` field.
   * `WithRetry` becomes `WithRetryNotify(p, policy, nil)`.
3. In `retrying.Generate`, after the deadline check and before the timer
   starts, call `notify` when it is set. Pass the call's `ctx` and a
   notice:
   * `Attempt` is the try that failed;
   * `MaxAttempts` is the policy's, after its defaults;
   * `Wait` is the wait;
   * `ServerDirected` is `serverRetryAfter(err) > 0`;
   * `Err` is `err`.
4. **Tests,** `llmprovider/retry_notify_0044_test.go`:
   * a `RetryAfter` 429 then a success sends one notice, every field
     checked;
   * backoff sends `ServerDirected` false, with `Wait` within the
     equal-jitter range;
   * no notice for a terminal error, the last attempt, a wait over
     `MaxDelay`, a wait past the deadline, or a second failure after a
     reply;
   * the `ctx` is the caller's (a context value);
   * a notify that cancels `ctx` ends the call with no further try;
   * a nil notify behaves as `WithRetry`;
   * `ModelLister` is kept;
   * notices arrive through `llmprovider.Stream`.

   Waits use millisecond delays, as the existing retry tests do.

### Phase 2: docs and the release

1. api-standards R28: `WithRetryNotify` next to `WithRetry`. The README's
   middleware lines, and `docs/architecture.md`'s `WithRetry` lines (R43).
2. Release notes, in the release's record: the two names, and that nothing
   changes for a caller who changes nothing.
3. Answer issue 3, and close it when the owner asks.

## Verification

| # | Criterion | Check | Phase |
| :--- | :--- | :--- | :--- |
| V1 | One notice before each wait, with the fields of D1 | Phase 1 tests | 1 |
| V2 | No notice where no wait follows (D2) | Phase 1 tests | 1 |
| V3 | `WithRetry` and a nil notify behave as before | existing retry tests unchanged; Phase 1 nil test | 1 |
| V4 | Additions only, exactly the MADR's (A1: `apidiff` lists the two names, not the fields) | `make api-check`; `tools/apidiff.py v1.14.0` | 1 |
| V5 | Every new test failed first on a planted break | `tools/plant.py` table | 1 |
| V6 | The docs name the new API | review; `tools/linkcheck.py` | 2 |

## Rollout and Rollback

* Additive: a caller that never calls `WithRetryNotify` sees no change. The
  release is a minor version: the next one after approval. That is
  `v1.15.0` if it lands with 0043-PLAN's phases, which the owner decides.
* Rollback before the tag: revert the phase's commit. After the tag, the
  names stay (R48), and a fix ships as a patch.

## Execution record

The owner accepted 0044-MADR (`d8efb45`) and approved this PLAN ("approve
the 0044 pair and proceed", 2026-10-18), after committing 0043-PLAN Phase
5 as `d247c22`.

### Phase 1: the notice (2026-10-18)

* **Deviation 2026-10-18: what `apidiff` lists.** Found by step 1's gate,
  after the code and tests:
  * `tools/apidiff.py v1.14.0` lists `./llmprovider.RetryNotice: added` and
    `./llmprovider.WithRetryNotify: added`. The MADR's Confirmation and V4
    expected the six fields too, and `apidiff` lists a new type once. Not
    pre-existing; the expectation in the record was wrong when written.
  * **Resolutions offered:** correct the record (recommended); also add a
    test that reflects over the fields.
  * **Decided:** correct the record. 0044-MADR A1 records it, and
    Confirmation and V4 are annotated. No file is added to the phase.
* **`<base>`** is `d247c22`. `llmprovider` coverage there: 97.7 %.
  `tools/gate.py` passed 23 of 23 on 0043-PLAN Phase 5's final tree, which
  differs from `<base>` only in two Markdown files.
* **Done.**
  * **Step 2.** `RetryNotice` (`Provider`, `Attempt`, `MaxAttempts`, `Wait`,
    `ServerDirected`, `Err`) and `WithRetryNotify`, in
    `llmprovider/retry.go`, each field with a doc comment.
    * `retrying` gains `notify`.
    * `WithRetry` is now `WithRetryNotify(p, policy, nil)`, and its doc
      comment names `WithRetryNotify`.
    * `WithRetryNotify`'s doc comment states D2: when notify is called and
      when not, the caller's goroutine and `ctx`, the wait beginning when
      notify returns, no recovery from a panic, and the order through
      `Stream`.
  * **Step 3.** In `retrying.Generate`, after the deadline check and before
    the timer, `notify` gets the call's `ctx` and the notice.
    `ServerDirected` is `serverRetryAfter(err) > 0`, so it covers
    `Retry-After` and the reset headers (E4).
  * **Step 4, tests,** `llmprovider/retry_notify_0044_test.go`, eight tests:
    * a `RetryAfter` 429 then a success: one notice, every field checked,
      and sent before the 40 ms wait ended;
    * backoff: two notices, not server-directed, each wait within the
      equal-jitter range, with `MaxAttempts` 3 from the default;
    * no notice where no wait follows: a terminal error, the last attempt,
      a wait over `MaxDelay`, a wait past the deadline, and a second
      failure after a reply (one notice, for the first);
    * the caller's `ctx`, read through a value; a notify that cancels it
      ends the call after one try, with an error that matches both
      `context.Canceled` and the failure;
    * a nil notify retries as `WithRetry`;
    * `ModelLister` kept, through `hygiene_test.go`'s `listingStub`;
    * through `Stream`, the notice before the first event.

    Three runs under `-race` are clean. The existing retry tests pass
    unchanged (V3).
* **Red first.** `tools/plant.py`, 15 plants on a scratch clone, all caught
  on the first run, each by an assertion. Before the run, the first test
  gained the check that the notice comes before the wait; without it, a
  notice after the wait would have been caught only by the 40 s slowdown
  of the context test. With it, the failure read `notified after
  117.512709ms; want before the 40ms wait`.

  | Plant | Result |
  | :--- | :--- |
  | notify is never called | caught |
  | notify comes before MaxDelay and the deadline | caught |
  | notify comes before the deadline check | caught |
  | notify comes on the last attempt | caught |
  | notify comes on a second failure after a reply | caught |
  | notify comes after the wait | caught |
  | the attempt is off by one | caught |
  | MaxAttempts is not sent | caught |
  | the wait is the base delay | caught |
  | ServerDirected is always false | caught |
  | ServerDirected is always true | caught |
  | the error is not sent | caught |
  | the provider is not sent | caught |
  | notify gets a fresh context | caught |
  | WithRetryNotify drops the lister | caught |
* **Gate.**
  * `make pre-add-check` on the two Go files: `2 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)`.
  * `tools/apidiff.py v1.14.0`: 0043's ten lines, and
    `./llmprovider.RetryNotice: added` and `./llmprovider.WithRetryNotify:
    added` (A1). `api-check` passes: `RetryPolicy` is unchanged.
  * `tools/gate.py`: 23 of 23. `llmprovider` coverage 97.7 %, unchanged.
  * `tools/denyscan.py --changed`: 0 hits. `make records-check`: 0
    problems.
* **Not done here:** the docs and the release notes (Phase 2).

### Phase 2: docs and the release (2026-10-18)

The owner committed Phase 1 as `3ac4c3a` and approved Phase 2 ("committed,
proceed").

* **No deviation.**
* **Step 1, the docs** (R43):
  * `docs/guides/api-standards.md` R28 names `WithRetryNotify` and
    `RetryNotice` next to `WithRetry`;
  * `README.md`: the `llmprovider` package row, a paragraph after
    `WithRetry`'s, and the `v1.15.0` status entry;
  * `docs/architecture.md`: "Around it" and a bullet after `WithRetry`'s.
  * No layout changed.
* **Step 2, the release notes.** `v1.15.0` carries both pairs, so its notes
  stay in one place:
  0043-PLAN-openai-chatgpt-plan-sign-in.md, Phase 5, step 4. They now say
  that the release carries this record, count 12 identifiers, list the two
  names under New API, and say that `WithRetry` retries as before.
* **Step 3, issue 3:** after the tag, and closed on the owner's ask, with
  0043-PLAN's step 5.
* **Gate.**
  * `tools/linkcheck.py`, markdownlint on the changed Markdown,
    `tools/denyscan.py --changed` and `make records-check`: clean.
  * No Go file changed in this phase.

