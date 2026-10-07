---
status: proposed
date: 2026-10-07
associated-madr: "0027-MADR-live-test-skips-and-gemini-429-path.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement One Skip Rule for the Live Tests, and Gemini's 429 Check on the Interactions Path

Associated MADR: [0027-MADR-live-test-skips-and-gemini-429-path.md](0027-MADR-live-test-skips-and-gemini-429-path.md)

## Goal

When it is done:

* every live test skips a transient failure through `SkipIfTransient`,
  except the two whose subject is a transient class, which say why;
* CI fails on a new hand-written transient skip;
* `TestLive_GeminiRateLimitShape` checks the error `Gemini.Generate`
  returns for a 429 from the Interactions API, and its checks run in CI on
  a fixture;
* the attempt at a live 429 is made, or not, as Q1 decides, and its result
  recorded.

No shipped code changes; there is no release.

## Scope

### In scope

* `llmprovider/live_together_test.go`, `live_claude_test.go`,
  `live_opencode_test.go`, `live_gemini_test.go`, `live_kilo_test.go`: the
  ten hand-written skips (0027-MADR, Context, table).
* `llmprovider/live_hygiene_test.go` (new, no build tag):
  `TestLiveTests_TransientSkipsUseTheHelper`.
* `llmprovider/live_gemini_429_test.go`: the burst through `Generate`.
* `llmprovider/gemini429_check_test.go` (new, no build tag): the 429
  checker and its unit test.
* `docs/README.md`: this pair's rows.

### Out of scope

* Any file outside `_test.go` and the records: the SDK does not change.
* Gemini's `llmtest` harness `Error` (0027-MADR, More Information).
* The subscription and sign-in live suites.

## Implementation Steps

Rules for every phase:

1. **Red first.** A new check is seen failing, on today's tree where it
   can, or on a scratch copy with a deliberate break; the FAIL line goes in
   the Execution Record.
2. **The gate** at the end of each phase, as 0026-PLAN rule 3 ran it:
   `make pre-add-check`; `CGO_ENABLED=0 go vet` for darwin, linux and
   windows, with and without `-tags live_gateways`; `go test -race -cover`
   and `-shuffle=on`; `go mod tidy -diff`; `make lint parity-check
   dep-check coverage-check api-check generate-check records-check
   gate-selftest`; markdownlint on the lint scope; links; the identifier
   scan.
3. **Live runs** use the owner's keys from the environment, checked for
   presence only, never printed.
4. **Deviations stop and prompt,** with evidence and resolutions, and are
   recorded here, and in the MADR when a decision or asserted fact changes.
5. **The owner commits.** One commit per phase; the agent stages.

### Phase 0: records

1. This pair, and its two rows in `docs/README.md` (59 records).
2. `make records-check` clean.

### Phase 1: one skip rule

1. **The check, red on today's tree.** Add
   `TestLiveTests_TransientSkipsUseTheHelper` in
   `llmprovider/live_hygiene_test.go`:
   * walk `llmprovider/` for `live_*_test.go`, and parse each with
     `go/parser`, comments included;
   * for each `if` statement, and each `case` clause of a `switch`, whose
     condition mentions an identifier `ErrRateLimited`,
     `ErrProviderUnavailable`, `ErrQuotaExhausted` or `ErrNotPermitted`, and
     whose body calls a method `Skip` or `Skipf`, report `file:line` unless
     a comment `// live-skip: <reason>`, with a non-empty reason, ends on
     the line before it;
   * the failure names every site, and the rule (0013-MADR D5).

   Run it: it fails, naming the eight narrower sites and the two
   purposeful ones.
2. **The eight narrower sites** call `llmprovider.SkipIfTransient(t, err)`
   in place of their own skip. In `TestLive_GeminiThinkingShapes` the two
   `case`s go, and the `switch` keeps its `err != nil` and output cases.
3. **The two purposeful sites** keep their skips, each preceded by
   `// live-skip:` and its reason:
   * `TestLive_TogetherRequiredFailsOnGptOss`: it asserts the 500 that
     `ErrProviderUnavailable` carries;
   * `TestLive_KiloDataCollectionDenied`: it asserts `ErrNotPermitted`.
4. **Green, and the plants.** The check passes. On a scratch copy:
   * a hand-written `if errors.Is(err, llmprovider.ErrQuotaExhausted) {
     t.Skip() }` in one live file fails it, naming that line;
   * a `// live-skip:` comment with no reason fails it;
   * a `case errors.Is(err, llmprovider.ErrRateLimited): t.Skipf(…)` fails
     it.
5. **Live run** of the converted tests, with
   `LLMPROVIDER_LIVE_TOGETHER=1`: `TestLive_TogetherWire`,
   `TestLive_TogetherToolChoices`, `TestLive_TogetherRequiredFailsOnGptOss`,
   the Claude, OpenCode reasoning and messages-thinking tests,
   `TestLive_GeminiThinkingShapes`, `TestLive_KiloReasoningShapes` and
   `TestLive_KiloDataCollectionDenied`. Each passes, or skips with a
   transient class in its message.
6. The gate; the Execution Record; stage.

### Phase 2: Gemini's 429 on the Interactions path

1. **The checker, red on a scratch copy.** In
   `llmprovider/gemini429_check_test.go` (package `llmprovider_test`, no
   build tag), `checkGemini429(r reporter, header http.Header, body []byte,
   err error)` checks:
   * DRIFT: the body is a Gemini error, bare or in a one-element array, with
     `error.status` `RESOURCE_EXHAUSTED` and a `google.rpc.RetryInfo`
     `retryDelay`; it reports which envelope it found;
   * `err` is an `*APIError`; with no `Retry-After` header, its
     `RetryAfter` is the `retryDelay`;
   * `err` is `ErrQuotaExhausted` exactly when a `QuotaFailure` violation's
     `quotaId` names a per-day quota.

   `reporter` is an interface with `Errorf` and `Logf`, which `*testing.T`
   satisfies. `TestCheckGemini429` feeds it Google's documented 429, per
   minute and per day, wrapped in the array. The fixture restates the one
   `api_error_gemini_0026_test.go` builds, which is in package
   `llmprovider` and so out of reach. It passes the body with the error
   `ClassifyHTTPError("gemini", …)` gives for it, and a recording reporter
   that must see no error. Seen
   failing: on a scratch copy with `soleObject` planted out (the per-day
   case reports `ErrQuotaExhausted = false` and `RetryAfter = 0s`).
2. **The live test.** `TestLive_GeminiRateLimitShape`:
   * builds `gemini.New(WithAPIKey(key), WithModel(model),
     WithMaxTokens(1), WithHTTPClient(&http.Client{Transport: rec}))`,
     where `rec` keeps the first 429's header and body;
   * sends the burst as concurrent `Generate` calls, stopping at the first
     429, and passes that call's error to `checkGemini429`;
   * keeps `LLMPROVIDER_LIVE_GEMINI_429` and `…_MODEL`; adds
     `LLMPROVIDER_LIVE_GEMINI_429_BURST` (default 20) under Q1 (a);
   * a status other than 200 or 429 still fails, and no 429 still skips,
     naming the burst size.
3. **One request first.** Under Q1 (a), a single `Generate` on the chosen
   model, its `Usage` logged, before the burst; the owner sets the burst
   from it.
4. **The live attempt,** as Q1 decides. Its result goes in the Execution
   Record: the burst, the 429's envelope, the checks' outcome, or the skip.
   A DRIFT failure, or a check failure, is a deviation.
5. The gate; the Execution Record; stage.

## Verification

* **V1. Red, then green,** with the FAIL lines, for the skip check (on
  today's tree and three plants) and the 429 checker (on a planted copy).
* **V2. The gate** is clean at the end of each phase.
* **V3. Live:** the converted tests pass or skip by the rule in one run;
  the 429 attempt is recorded as Q1 decides.
* **V4. Scope:** `git diff --stat` touches only `_test.go` files and the
  records; `api-check` reports no change.

## Rollout and Rollback

* **Rollout.** Three commits, Phases 0–2, committed and pushed by the
  owner. No tag: the module does not change.
* **Rollback.** Each phase reverts alone. Reverting Phase 1 restores the
  hand-written skips and removes the check together; reverting Phase 2
  restores the `generateContent` burst.

## Execution Record

Not started. The MADR is `proposed`, with Q1 open.
