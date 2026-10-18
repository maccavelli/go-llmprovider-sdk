---
status: proposed
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
| V4 | Additions only, exactly the MADR's | `make api-check`; `tools/apidiff.py v1.14.0` | 1 |
| V5 | Every new test failed first on a planted break | `tools/plant.py` table | 1 |
| V6 | The docs name the new API | review; `tools/linkcheck.py` | 2 |

## Rollout and Rollback

* Additive: a caller that never calls `WithRetryNotify` sees no change. The
  release is a minor version: the next one after approval. That is
  `v1.15.0` if it lands with 0043-PLAN's phases, which the owner decides.
* Rollback before the tag: revert the phase's commit. After the tag, the
  names stay (R48), and a fix ships as a patch.
