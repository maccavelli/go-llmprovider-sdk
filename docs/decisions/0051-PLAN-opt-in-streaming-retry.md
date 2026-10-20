---
status: in-progress
date: 2026-10-20
associated-madr: "0051-MADR-opt-in-streaming-retry.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Streaming Through a Retry, Opt-In

Associated MADR: [0051-MADR-opt-in-streaming-retry.md](0051-MADR-opt-in-streaming-retry.md)

## Goal

The MADR's D1 to D6, as the owner answered its questions on 2026-10-20 (Q1:
a new `WithStreamRetry`; Q2: `v1.19.0`; Q3, as revised: at least one delta,
two preferred, liveness by holding the reply back, and a test in each
provider package keeping its scripts at two):

* `llmprovider.WithStreamRetry(p, policy, notify)` streams live over a
  provider that streams natively: a failure before the first event is
  retried as `Generate` retries it, through one shared decision; after the
  first event every pair passes through and a failure ends the stream;
* over a provider that does not stream, it is `WithRetryNotify`'s result;
* `WithRetry` and `WithRetryNotify` behave exactly as at `v1.18.0`;
* `llmtest.Run` checks every streaming provider through the new wrapper:
  retried before the first event, passed through after, live, and with the
  unwrapped stream's events; `StreamText` stays at least one delta, two
  preferred;
* every built-in `StreamText` script carries two deltas, and each provider
  package's own test holds it there;
* `v1.19.0` is released; issue 4 is answered, on the owner's word.

## Scope

### In scope

* `llmprovider/retry.go`: the shared decision, `WithStreamRetry` and its
  types, and the docs of `WithRetry`; its tests; `ExampleWithStreamRetry`
  in `llmprovider/example_test.go`.
* `llmprovider/llmtest/llmtest.go`: `StreamText`'s doc (two preferred), a
  `Logf` on its unexported reporter, and the new check; its tests.
* The twelve `StreamText` scripts of the built-in harnesses, in
  `llmprovider/providers/{openai,grok,claude,gemini,huggingface,kilo,together,ollama,opencode}`
  (`opencode` holds four, one per route), and one test in each of the nine
  packages holding its scripts to two deltas.
* `README.md`, `docs/architecture.md`, `docs/guides/api-standards.md`
  (R28), `docs/guides/adding-a-provider.md`, `CHANGELOG.md`.
* `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md`: a
  dated amendment that points here (D4).

### Out of scope

* Any change to `WithRetry`'s or `WithRetryNotify`'s behaviour, to
  `RetryPolicy` or to `RetryNotice`.
* A new `llmtest.Harness` field, or any other exported name in `llmtest`.
* gobble-cli's adoption: its own records (G1, G2).
* Posting on issue 4 without the owner's word.

### The exported name (R55)

| Name | Package | Call site that needs it |
| :--- | :--- | :--- |
| `WithStreamRetry` | `llmprovider` | gobble-cli `llm/provider/provider.go:197` (G1), which wraps every provider in `WithRetry` and shows each delta as it arrives |

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase; a phase
  starts after the owner has committed the last.
* **The gate, before staging:** `python3 -B tools/gate.py`; `make
  pre-add-check FILES="…"` on the phase's Go files; `tools/claims.py` on
  both records; `make records-check`; `tools/recordlint.py --changed`;
  `tools/denyscan.py --changed`; `tools/linkcheck.py`. Each exit status is
  captured before any filter, and read before anything is written from it.
* **Red first.** Every new test or check is seen to fail on a planted
  break, by `tools/plant.py` on a scratch clone, never in the tree; the
  phase's record holds each plant and its failure line.
* **The docs.** An API change updates the user docs in the same phase
  (R43); `make docs-check` and `make parity-check` stay at 0; a doc names
  only what a test shows (R52).
* **The API and the wire.** `tools/apidiff.py v1.18.0` reports
  `WithStreamRetry` as an addition and nothing else; G-wire's goldens do not
  change (R45); `make coverage-check` holds every floor (R47), `llmprovider`
  at its 95.9 %.
* **The existing retry tests are the drift check.** Every test of
  `WithRetry` and `WithRetryNotify` at `b5583cf` passes unchanged.
* **Stage, then the owner commits** with `git commit --no-edit`; push and
  tags on the owner's word; the disclosure guard runs before the owner
  pushes.
* **Stop conditions:** a fact of the MADR or a claim below no longer holds;
  a step would add or change an exported name other than `WithStreamRetry`;
  an existing retry test would need to change; a golden changes; a doc
  would state a behaviour no test shows. Stop, record a dated deviation
  here (and an amendment in the MADR when a decision or a fact changes),
  and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.
2. **0031-MADR's amendment** (C13), inserted before its `### Consequences`:
   "Amendment (2026-10-20): live text through a retry, opt-in". It says
   that D6 stands for `WithRetry` and `WithRetryNotify`, that its last
   bullet (C8) is narrowed by
   [0051-MADR-opt-in-streaming-retry.md](0051-MADR-opt-in-streaming-retry.md)
   for a caller of `WithStreamRetry`, and that D6's text is not rewritten.

### Phase 1: `WithStreamRetry` (D1 to D4; D5's unit tests; D6's example)

1. **The shared decision** (C3 to C5): `retrying.Generate`'s body after a
   failed attempt moves, unchanged, into one unexported method, such as
   `(r *retrying) after(ctx, attempt, err, &retriedAfterReply) error`,
   which waits and returns nil to try again, or returns the error to give
   the caller: the `retryable` test; the once-only rule of `AfterReply`;
   `MaxAttempts` with "failed after %d attempts"; `policy.wait`'s refusal;
   the deadline rule; `notify`; the timer, and `errors.Join(ctx.Err(),
   err)` when `ctx` ends. `Generate` calls it. Every existing retry test
   passes unchanged.
2. **`WithStreamRetry`** (D1, D3): over a `Streamer`, a `*streamRetrying`
   (embedding `*retrying`, holding the inner `Streamer`), or a
   `*streamRetryingLister` when the inner provider lists models; over any
   other provider, `WithRetryNotify(p, policy, notify)`. `Capabilities` is
   the inner provider's, unchanged. `Generate` and `ListModels` are the
   embedded ones.
3. **Its `Stream`** (D2): per attempt, range over the inner stream; while
   no event has been yielded, a pair with an error ends the attempt and goes
   to step 1's method, with one `retriedAfterReply` per `Stream` call; once
   one has, every pair is yielded as it comes, and an error is yielded
   unchanged and ends the stream. The error that ends the stream before the
   first event is yielded once, with a zero `Event`. A `yield` that returns
   false returns at once.
4. **Docs:** `WithStreamRetry`'s (D2, D3, D6: the trade-off; R51 safe for
   concurrent use as `p` is; R52 the errors, those of `WithRetry` before
   the first event and the inner stream's after it); `WithRetry`'s last
   paragraph (C1) names `WithStreamRetry` for a caller who wants text as it
   is generated. `WithRetryNotify`'s (C11) stays true and is unchanged.
5. **Tests,** `llmprovider/retry_stream_0051_test.go`, `_0051$`, under
   `-race`, over a scripted inner provider that is a `Streamer`: each call
   to `Stream` plays the next attempt's script (events, an error, or both)
   and counts the calls.
   * **live:** the script yields one text delta, then waits until the test
     has received it (a channel, with a two-second limit that fails the
     test rather than hanging), then a second delta and `EventDone`; the
     deltas joined equal `EventDone`'s text;
   * a 503 (`ErrProviderUnavailable`) first, then success: two calls, one
     notice, the success's events only;
   * one delta, then a failure: the delta, the error unchanged, no
     `EventDone`, one call, no notice;
   * against `Generate`, case by case: 503 every time, `MaxAttempts` calls
     and "failed after"; an `AfterReply` failure every time, two calls; an
     `APIError` with a `RetryAfter` over `MaxDelay`, one call, the error at
     once; a wait past `ctx`'s deadline, one call; `ctx` cancelled during a
     wait, an error matching `context.Canceled` and the failure; a
     not-retryable failure (`ErrAuthFailure`), one call;
   * a caller that stops after the first event: the inner stream's yield
     returned false, and no second call;
   * `Capabilities().NativeStreaming` is the inner provider's; a
     `ModelLister` inner gives a `ModelLister` result;
   * over `llmtest.Fake`, which is not a `Streamer`: the result is not one,
     reports `NativeStreaming` `Unsupported`, and `Stream` over it retries
     as `WithRetryNotify` does.
6. **`ExampleWithStreamRetry`** (R50), in `llmprovider/example_test.go`:
   over `httptest` or a scripted `Streamer` defined in the example file
   (`llmtest.Fake` has no `Stream`), a 503 then two deltas, printing each
   delta as it arrives, with an `// Output:` comment.
7. **The user docs** (R43): README's `llmprovider` row (C14) and its retry
   paragraph (C15); `docs/architecture.md`'s step 5 (C16), its list entries
   (C17) and the "not yet" entry (C9), which goes; R28 (C12).
   `CHANGELOG.md` `[Unreleased]`, Added: `WithStreamRetry`.
8. **Red first,** each caught:
   * `Stream` retrying after the first event (the started test dropped):
     the one-delta-then-failure case sees two calls;
   * `Stream` not retrying before it: the 503-then-success case fails;
   * the shared method without the once-only rule: the `AfterReply` case
     here, and the existing `Generate` test of 0021-MADR D1;
   * `Stream` collecting every event before yielding any: the live case
     fails at its limit;
   * `Capabilities` dropping `NativeStreaming`: its case fails;
   * the non-`Streamer` branch returning a `*streamRetrying`: its case
     fails.

### Phase 2: `llmtest` (D5, Q3)

1. **`StreamText`'s doc** (C10): its text in at least one text delta, as
   today; two or more preferred, since one cannot show that deltas are not
   merged. D7's check (C18) is unchanged. `llmtest`'s unexported `reporter`
   gains `Logf` (`t.Logf`; its own tests' recorder keeps the lines).
2. **The new check,** "D7-stream-retry", run beside D7's (C19) when the
   provider is a `Streamer` and both fields are set; each provider built as
   `build` builds it, and wrapped in `WithStreamRetry` with `BaseDelay`
   1 ms, `MaxDelay` 10 ms and a counting `notify`:
   * **retry before the first event:** a server whose first reply is the
     harness's `Error` with 503 and whose later ones are `StreamText`: two
     requests, one notice, one `EventDone`, last, the text deltas joined
     equal its message text, its `Response` equal to `Generate`'s on
     `Text`;
   * **pass through after it:** `StreamFailure`: one request, no notice,
     one error, last, no `EventDone`;
   * **live** (M5): `StreamText` is split into parts, each ending in a
     blank line. The unwrapped provider is served one part at a time,
     50 ms apart, until its first text delta, which gives `k`, the part
     after which it arrived. The wrapped provider is then served parts 0
     to `k` at once, and the rest only once the caller holds a text delta,
     or after a limit of 2 s; one that has none by then fails, with
     "D7-stream-retry (0051-MADR D5): no text delta before the reply's
     part %d of %d was sent; the stream is not live";
   * **the same events:** the wrapped run's events, types and texts in
     order, equal the unwrapped run's;
   * **not checkable,** when no part follows the first delta, or there is
     no delta before the last part: a `Logf` line saying why, and no
     error; **one delta:** a `Logf` line that two would show more;
   * every reply body closed, as D7 checks.
3. **The twelve scripts,** each `StreamText` carrying its text in two text
   deltas, joined equal to the text `Text` writes: `openai`, `grok`,
   `claude`, `gemini`, `huggingface`, `kilo`, `together`, `ollama`, and
   `opencode`'s Responses, Messages, Chat Completions and generateContent
   scripts; each script's comment says so.
4. **The nine packages' tests,** `TestStreamTextDeltas_0051` in each
   provider package's `llmtest_test.go`: its own `StreamText` scripts (all
   four for `opencode`, each route's), served to its provider, give at
   least two text deltas through `Stream`, unwrapped, joined equal to the
   text `Text` writes.
5. **Docs:** `docs/architecture.md`'s stream harness entry (C20): one delta
   at least, two preferred, and the check through `WithStreamRetry`;
   `docs/guides/adding-a-provider.md`'s capabilities section (C21): a
   `Streamer`'s harness sets `StreamText` and `StreamFailure`, `StreamText`
   in parts that end in blank lines, with two deltas. `CHANGELOG.md`
   `[Unreleased]`, Changed: the new check, which a third-party streaming
   provider can fail (D6).
6. **Red first,** each caught:
   * `WithStreamRetry`'s `Stream` collecting the inner stream before
     yielding: the live case fails in every streaming provider's suite,
     Gemini's included;
   * `WithStreamRetry`'s `Stream` dropping a text delta after the first:
     the event comparison fails in every suite, where the scripts carry
     two;
   * `Stream` retrying after the first event (Phase 1's first plant,
     again): the `StreamFailure` case sees two requests;
   * `Stream` not retrying before it: the 503 case fails;
   * one built-in script left with one delta: that package's
     `TestStreamTextDeltas_0051` fails;
   * `llmtest`'s own tests, over a fake streaming provider: a one-part
     script gives the not-checkable line and no error, and a one-delta
     script the preference line, each planted the other way (an error for
     the first, no line for the second) and caught.

### Phase 3: release `v1.19.0` (D6)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.19.0] - <date>`, the
   date that of the commit the tag will name; an empty `[Unreleased]` above
   it; the comparison links. The README's Status names `v1.19.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.18.0`, the one
   addition; the disclosure guard over the outgoing commits, run by the
   agent before the owner pushes.
3. **The owner tags `v1.19.0`** and pushes it; CI's `release` job makes the
   Release; its body equals `check_changelog.py --section v1.19.0 --repo
   maccavelli/go-llmprovider-sdk`.
4. **After the tag:** `tools/consumer.py v1.19.0`; `tools/offline.py
   v1.19.0`.
5. **Issue 4:** the agent drafts a reply naming `v1.19.0`, the function,
   and each acceptance item with the test that shows it; it is posted, and
   the issue closed, only on the owner's word.
6. **Close:** the MADR's Confirmation items, each with its evidence; this
   PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `WithRetry` and `WithRetryNotify` are unchanged | 1 | the existing retry tests, unchanged |
| V2 | Deltas reach the caller live, joined equal to `EventDone`'s text; two or more on every built-in wire | 1, 2 | the live unit test; `llmtest`'s live case and event comparison on every wire |
| V3 | A failure before the first event is retried under the policy, with its notices | 1, 2 | the unit tests; the `llmtest` check |
| V4 | A failure after the first event ends the stream, one request, no `EventDone` | 1, 2 | the unit test; the `llmtest` check |
| V5 | Over a provider that does not stream, the result is `WithRetryNotify`'s | 1 | its unit test; R10 |
| V6 | `apidiff` lists `WithStreamRetry` only; the docs name what the tests show | all | `tools/apidiff.py v1.18.0`; `make docs-check`, `parity-check` |
| V7 | Every built-in `StreamText` carries two deltas, and a one-delta harness still passes | 2 | the nine `TestStreamTextDeltas_0051`; `llmtest`'s own tests |
| V8 | `v1.19.0`'s Release body is its section | 3 | `gh release view`; `--section` |
| V9 | Every new test and check failed first | 1, 2 | the execution record |

## Rollout and Rollback

* **Rollout.** Four commits on `main` (the records, then a commit per
  phase), staged by the agent and committed by the owner; then the
  owner's tag.
* **Rollback.** Each phase reverts on its own, newest first. Reverting
  Phase 2 removes the new check and restores the scripts; Phase 1
  then reverts cleanly, `WithStreamRetry` with it. After `v1.19.0` is
  tagged, `WithStreamRetry` is under R48 and is not removed.

## Code claims

<!-- claims: self=b5583cf -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `WithRetry`'s doc: no native stream | `llmprovider/retry.go:81` | `// The result does not stream natively, even when p does; Stream falls back to` |
| C3 | `Generate`'s loop | `llmprovider/retry.go:139` | `func (r *retrying) Generate(ctx context.Context, req *Request) (*Response, error) {` |
| C4 | Once-only after a reply began | `llmprovider/retry.go:149` | `if transport.IsAfterReply(err) {` |
| C5 | A notice only once a wait is certain | `llmprovider/retry.go:167` | `if r.notify != nil {` |
| C8 | 0031's D6: exclusive | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:195` | `` Live deltas and `WithRetry` are therefore exclusive `` |
| C9 | The architecture's "not yet" entry | `docs/architecture.md:928` | `` Live text through `WithRetry`. `` |
| C10 | The stream harness's contract | `llmprovider/llmtest/llmtest.go:100` | `// reply, carrying the response Text writes, its text in at least one` |
| C11 | `WithRetryNotify`'s doc: notices before the first event | `llmprovider/retry.go:101` | `// Through llmprovider.Stream the notices come before the first event, since` |
| C12 | R28 names the retry wrappers | `docs/guides/api-standards.md:210` | `` error kind; `WithRetryNotify` retries the same `` |
| C13 | 0031-MADR's amendments end before its consequences | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:496` | `### Consequences` |
| C14 | README's `llmprovider` row | `README.md:26` | `` `Registry`, `WithRetry` and `WithRetryNotify`, `Stream` `` |
| C15 | README's retry paragraph | `README.md:101` | `` `WithRetryNotify(p, policy, notify)` retries the same, and calls `notify` `` |
| C16 | The architecture's step 5 | `docs/architecture.md:177` | `` 5. **Around it.** `WithRetry` wraps a provider and retries by the error's `` |
| C17 | The architecture's `WithRetryNotify` entry | `docs/architecture.md:446` | `` - `WithRetryNotify(p, policy, notify)` is the same wrapper, which calls `` |
| C18 | D7's delta check | `llmprovider/llmtest/llmtest.go:1222` | `deltas.String() != texts.String() {` |
| C19 | D7's check is switched on here | `llmprovider/llmtest/llmtest.go:213` | `if h.StreamText != nil && h.StreamFailure != nil {` |
| C20 | The architecture's stream harness entry | `docs/architecture.md:470` | `` be `ErrNotPermitted` (Kilo, OpenCode, OpenAI). `StreamText` and `` |
| C21 | The provider guide's streaming rule | `docs/guides/adding-a-provider.md:135` | `` - Declare `NativeStreaming` `Supported` only if you implement `Streamer`. `` |

## Execution record

### Phase 0: records (2026-10-20)

* **No deviation.** The owner committed the MADR and this PLAN, with Q3
  as revised, as `ea4fddb`.
* **Step 1,** `tools/records.py status`: the MADR `accepted`, this PLAN
  `in-progress`, each with its index row; "records-check: 0 problem(s)".
* **Step 2,** 0031-MADR's "Amendment (2026-10-20): live text through a
  retry, opt-in", before its `### Consequences` (C13): D6 stands for
  `WithRetry` and `WithRetryNotify`, its last bullet (C8) holds for those
  two and no longer for the module, and D6's text is not rewritten.
* **The checks:** `tools/claims.py`, 14 of 14 and 18 of 18; `make
  records-check`, 0 problems; `tools/recordlint.py --changed`, no rule rose;
  `tools/linkcheck.py`, 0 problems; `tools/gate.py`, 27 of 27 passed.
