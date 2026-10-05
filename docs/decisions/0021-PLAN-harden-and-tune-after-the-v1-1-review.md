---
status: in-progress
date: 2026-10-05
associated-madr: "0021-MADR-harden-and-tune-after-the-v1-1-review.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Hardening and Tuning After the v1.1 Review

Associated MADR: [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md)

## Goal

Every finding of the MADR's Tables A–D is fixed and proven, or is recorded
here as not done with its reason. The fixes follow the owner's decisions
D1–D5 and the MADR's §2, as corrected by its amendment "facts corrected
while writing the PLAN" of 2026-10-04.

When it is done:

* the repository gate below passes at the end of every phase;
* `api-check` reports only additive changes against `v1.1.0` (R48);
* the live checks under Verification have run with the owner's keys and
  sessions, and their output is recorded here;
* the records the MADR amends (0009, 0010, 0012, 0020) carry their
  amendments, and the docs say what the code does.

## Scope

### In scope

The six phases of the MADR's §2, in order, and phase 7, added on 2026-10-04. Each phase touches one area,
ends with the gate green, and is staged as one change. Agents stage; the
owner commits.

| Phase | Area | Findings |
| :--- | :--- | :--- |
| 1 | replies and retries | W11, D1 (T5, W6), D2 (T12, T13), W2, T6, T7, T14 |
| 2 | answers | W1, W3, W4, W5, W10, D4 (W8), D5 `ReasoningItem.Format` (W7), W9 |
| 3 | credentials | D5 `TokenInvalidator` (T1), T2, T3, T4, T8, T9, T10, T11, T15 |
| 4 | catalog | C1, C2, C3, C4, C5, C6, D3 (C7, C8), C9, C11, Z11, C12, C13, C10, W13 |
| 5 | wizard and redaction | Z1, Z2, Z3, Z6, Z7, Z10 |
| 6 | tooling and tests | Z4, Z9, Z5, `nolintlint`, Z8, Z12, D5 harness fields (W12) |
| 7 | live drift (added 2026-10-04) | L1, L2 |

### Out of scope

* The MADR's §3 "Not adopted": a pid and host liveness check on the lock,
  typed request structs, faster search, the other 30 linters.
* `llmprovider.WithBaseURL` keeps accepting any string. Z7 is fixed in the
  wizard, where the URL is entered; refusing it in the library would break
  callers that pass such URLs today (R48).
* Sending a Gemini `functionCall.id` on the wire (W10 decodes it only). No
  live evidence shows the service needs it back, and the goldens stay as
  they are.
* Any exported API change that is not additive. A fix that seems to need
  one stops for a deviation.
* Push and tags, which are the owner's.

## Rules for every phase

1. **Red first.** Each finding gets a test that fails before its fix. The
   test is written into a scratch copy of the tree that still has the old
   code:

   ```bash
   SCRATCH="$TMPDIR/p21-red" && rm -rf "$SCRATCH" && mkdir -p "$SCRATCH"
   git ls-files -co --exclude-standard -z | rsync -a --from0 --files-from=- ./ "$SCRATCH"
   ```

   The FAIL line goes in the Execution record, then the PASS line from the
   tree after the fix. A change to a gate script is seen to fail on a
   planted breach in a scratch copy, never by dirtying the tree.
2. **The gate before staging.** Each command's output goes to a scratch
   file, and its exit status is captured before any filter. All must exit
   0:

   | # | Command |
   | :--- | :--- |
   | G1 | `make pre-add-check` |
   | G2 | `GOOS=darwin CGO_ENABLED=0 go vet ./...`, then the same with `GOOS=linux` and `GOOS=windows` |
   | G3 | `go vet -tags live_gateways ./...` |
   | G4 | `go test -race -count=1 -cover ./...` (from phase 6: with `-shuffle=on`) |
   | G5 | `go mod tidy -diff` |
   | G6 | `make lint` |
   | G7 | `make parity-check dep-check coverage-check api-check generate-check` (from phase 6: and `gate-selftest`) |
   | G8 | `markdownlint-cli2 --no-globs` on every changed `.md` file |
   | G9 | `go test -count=3 ./llmprovider/... -run TestWireGoldens` (goldens stable) |
   | G10 | every relative link in every changed `.md` file resolves |
   | G11 | the identifier scan of the changed files: the local account name, the hostname domain, machine paths |

3. **G-wire.** A change to what goes on the wire updates its goldens with
   `go test ./<pkg> -run TestWireGoldens -update` in the same change (R45).
   The Execution record lists each changed golden and why. A golden that
   changes when this PLAN says it does not is a deviation.
4. **Ranking goldens.** `TestListModelCatalog_Snapshot20260926`
   (`llmprovider/catalog/discovery_ranking_test.go:279`) changes only where
   a step says it does. Any other change to it, or to an existing ranking
   assertion, is a deviation.
5. **Deviations stop and prompt,** with evidence and resolutions, under the
   rules in `AGENTS.md`. The chosen resolution is recorded here, and in the
   MADR when a decision or an asserted fact changes, before work continues.
6. **Records** cite records by full filename. Nothing committed carries a
   hostname, account name or machine path.
7. **Staging.** At the end of a phase the agent stages the phase with
   `git add` and reports it; the owner commits with `git commit --no-edit`.

Paths below are from the repository root.

## Implementation Steps

### Phase 1: replies and retries

#### 1.1 The shared request path (W11, the mechanism of D1, W6)

* **Red:**
  * `TestGenerate_CutBodyIsUnavailable` in
    `llmprovider/providers/together/reply_test.go`. A 200 with
    `Content-Length: 100` that closes after 40 bytes must be
    `ErrProviderUnavailable`, and `transport.IsAfterReply` must be true.
    Today it fails with `ErrIncomplete` (W6).
  * `TestGenerate_OverLimitIsIncomplete`, same file. A 200 body of
    `wire.ReplyLimit+1` bytes must be `ErrIncomplete`, with a message
    containing `over 16 MiB`, and must not be after-reply. Today it fails
    on the message, because the cut is indistinguishable from a decode
    error.
  * `TestClaude_NilToolSchemaIsEmptyObject` in
    `llmprovider/providers/claude/request_test.go`. A `Tool{Name: "x"}`
    with a nil `Schema` sends `"input_schema":{"type":"object","properties":{}}`.
    Today it sends `null`.
* **Change:**
  * New `llmprovider/internal/transport/reply.go`, standard library only:
    * `const StreamIdleTimeout = 300 * time.Second`;
    * `var ErrIdleTimeout`, an error whose `Timeout()` is true, with the
      text `no data for 300s`;
    * `type afterReply struct{ err error }`, with `Error` and `Unwrap`,
      plus `func AfterReply(err error) error` and
      `func IsAfterReply(err error) bool`;
    * `type ReplyReader`. It wraps `resp.Body` with three jobs:
      * an idle timer, reset by every `Read` that returns `n > 0`. On
        expiry it calls the request's `context.CancelCauseFunc` with
        `ErrIdleTimeout`;
      * a byte limit. Past `limit` bytes it returns
        `ErrReplyTooLarge` (`reply over 16 MiB`). A limit of 0 means none;
      * the first read error that is not `io.EOF`, kept for `ReadErr()`.
        When the context was cancelled for idleness, it keeps
        `ErrIdleTimeout`, not `context.Canceled`.
  * New `llmprovider/internal/wire/post.go`:

    ```go
    type Call struct {
        Provider string                                  // label for ClassifyHTTPError and DecodeError
        Client   *http.Client
        Logger   *slog.Logger
        URL      string
        Body     any                                     // marshalled with json.Marshal
        Prepare  func(*http.Request, llmprovider.Token) // headers and the token
        Stream   bool                                    // no whole-body limit; the decoder bounds each event
    }

    func Post[T any](ctx context.Context, c Call, token llmprovider.Token,
        decode func(io.Reader) (T, error)) (T, error)
    ```

    `Post` runs these steps in order:
    1. Marshal the body. A failure is
       `llmprovider: <provider>: marshal request: %w`.
    2. Derive the request context with `context.WithCancelCause`.
    3. Build the POST, set `Content-Type: application/json`, and call
       `Prepare`.
    4. Call `Client.Do`. A send error is returned as is.
    5. Close the body with the debug log the providers use today.
    6. Pass a non-2xx reply to `llmprovider.ClassifyHTTPError(c.Provider,
       resp)`.
    7. Wrap the body in a `ReplyReader`: limit `ReplyLimit`, or 0 when
       `Stream` is set; idle `idleTimeout`, a package variable set to
       `transport.StreamIdleTimeout` that tests shorten.
    8. Decode, and pass a decode error to `DecodeError`.
  * `DecodeError(provider string, err, readErr error) error` in
    `llmprovider/internal/wire/decode_error.go` checks, in order:
    * nil is nil;
    * an error that already has a kind is returned as is;
    * `ErrReplyTooLarge` becomes `ErrIncomplete`, `<provider>: reply over
      16 MiB`;
    * when `readErr` is set, the error becomes `ErrProviderUnavailable`,
      `<provider>: read reply: <readErr>`, wrapped in
      `transport.AfterReply`;
    * anything else becomes `ErrIncomplete`.

    The `net.Error` branch goes. Every caller is updated:
    `grep -rn 'wire.DecodeError' llmprovider` lists them, and a caller
    without a `ReplyReader` passes `nil`.
  * `Reauth` becomes `func Reauth[T any](ctx context.Context, provider
    string, src llmprovider.TokenSource, send func(llmprovider.Token) (T,
    error)) (T, error)`:
    1. Fetch the token. A failure is
       `llmprovider: <provider>: acquire token: %w`.
    2. Call `send`.
    3. On a 401 `*APIError` from an `InvalidatingSource`, call
       `Invalidate()`, fetch a token again, and send once more.

    Step 3.1 adds the `TokenInvalidator` branch.
  * Move the nine `generateOnce` functions onto `Post` and the new
    `Reauth`. Each takes the token as a parameter and keeps its own URL,
    headers and decoder. The nine are in `openai`, `claude`, `gemini`,
    `grok`, `huggingface`, `kilo`, `ollama`, `opencode` and `together`.
    `openai.listChatGPT` (`openai.go:314`) moves to the new `Reauth`
    signature only.
  * Share the tool encoders:
    * new `llmprovider/internal/wire/tools.go` with `ToolSchema(any) any`,
      where nil becomes `{"type":"object","properties":{}}`;
    * `ResponsesTools([]llmprovider.Tool)` and
      `ResponsesToolChoice(llmprovider.ToolChoice)`, replacing the copies
      at `openai.go:210-226`, `grok.go:185-198` and
      `opencode.go:360-370`;
    * `MessagesTools([]llmprovider.Tool)`, replacing `claude.go:201-209`
      and `opencode.go:398-404`. Each keeps its own Messages tool_choice
      function;
    * `chatcompletions.toolList`, `gemini.go:225` and `opencode.go:447`
      call `ToolSchema`.
  * Delete the stale "1 MiB bounds a runaway reply." comments:
    `kilo.go:253`, `together.go:164`, `gemini.go:178`, `grok.go:259`,
    `huggingface.go:161`, `ollama.go:173`, and
    `together/reply_test.go:39`.
* **Characterisation:** `TestWireGoldens` passes unchanged in every
  provider package before and after the move. No golden case has a nil
  schema.
* **Done when** the three red tests pass, G9 shows no golden change, and
  `grep -rn 'func (p \*provider) generateOnce' llmprovider/providers`
  shows each body calling `wire.Post`.

#### 1.2 Timeouts (D2: T12, T13)

* **Red:**
  * `TestDefaultClient_Timeouts`
    (`llmprovider/internal/transport/client_test.go:14`) changes to the
    decided values:
    * `Timeout == 0`;
    * `ResponseHeaderTimeout == 300s`;
    * `ForceAttemptHTTP2`;
    * a non-nil `DialContext`;
    * the package `dialer` has `Timeout == 30s` and `KeepAlive == 30s`.

    It fails today on `Timeout` (330 s).
  * `TestPost_IdleLimit` in `llmprovider/internal/wire/post_test.go`. The
    server sends headers and one byte, then stalls; `idleTimeout` is 50
    ms, and the test's context allows 2 s. The error must be after-reply
    and wrap `transport.ErrIdleTimeout`. Today the same scenario through a
    provider ends with `context deadline exceeded` at 2 s.
  * `TestPost_IdleLimitCoversCallerClient`: the same, through a provider
    built with `llmprovider.WithHTTPClient(&http.Client{})`.
  * `TestAuthRequests_Bounded` in
    `llmprovider/auth/request_bound_test.go`. The issuer sends headers,
    then stalls the body. A refresh, a device poll, a token exchange and a
    revocation each return within `authRequestTimeout`, shortened to 100
    ms. This test is new with the change: it hangs on the changed client
    without the bound, which is the planted state.
* **Change:**
  * In `llmprovider/internal/transport/transport.go`, `DefaultClient`:
    * drops `Timeout`;
    * sets `DialContext: dialer.DialContext` with
      `dialer = &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 *
      time.Second}`;
    * sets `ForceAttemptHTTP2: true`;
    * keeps `ResponseHeaderTimeout: 300 * time.Second` and the other
      fields.

    Its doc comment states D2.
  * `llmprovider/auth` gets `authRequestTimeout = 30 * time.Second`. Every
    `.Do(` in `llmprovider/auth/*.go` runs under
    `context.WithTimeout(ctx, authRequestTimeout)`. The grep's list goes
    in the Execution record. The refresh is tightened to 15 s in step 3.4.
  * The listing bounds stay as they are:
    * `modelListingTimeout` 10 s (`llmprovider/catalog/discovery.go:33`);
    * `chatGPTListingTimeout` 10 s;
    * `metadataLookupTimeout` 5 s.
  * Docs: `README.md:92` and `docs/architecture.md:227` describe a 300 s
    idle limit, a 300 s wait for the first byte, and a 30 s connect
    timeout, with no total.
* **Records:** append "Amendment 2026-10-04: generation timeouts (0021
  D2)" to `0012-MADR-conform-providers-to-reference-clients.md`:
  * §1.3's 330 s total gives way to the 300 s idle limit;
  * the header timeout stays;
  * §4.1's retry is bounded by 0021 D1.

#### 1.3 The ChatGPT stream reader (W2)

* **Red**, in `llmprovider/internal/wire/responses/stream_limit_test.go`:
  * `TestReadStream_LongStreamCompletes`: 68,826
    `response.output_text.delta` events, about 17.8 MB, then
    `response.output_item.done` and `response.completed`. It must succeed
    with the item's text. It fails today with `ErrIncomplete … unexpected
    end of JSON input`.
  * `TestReadStream_OverlongEventIsIncomplete`: one `data:` line of 16 MiB
    + 1 byte must be `ErrIncomplete`. Today it is `ErrProviderUnavailable`
    (`bufio.ErrTooLong`).
  * `TestReadStream_EndWithoutCompletedIsAfterReply`: a stream that ends
    after `response.created` must be `ErrProviderUnavailable` and after-reply.
    It fails today on the mark.
* **Change** in `llmprovider/internal/wire/responses/responses.go`:
  * Replace the scanner (`:203-204`) with a `bufio.Reader` and
    `readLine(r, eventLimit)`, where `eventLimit = 16 << 20` bounds one
    line. There is no whole-stream limit; `Post` with `Stream: true`
    applies the idle limit.
  * Skip ignored events before decoding them. If the payload's first
    `"type":"` value ends in `.delta`, the event is skipped. No nested
    item type ends in `.delta`, and a payload with no match is decoded in
    full.
  * The "stream ended before response.completed" error (`:266`) is
    wrapped in `transport.AfterReply`.
  * Rename `streamLimit` to `eventLimit`, and change its doc to "bounds
    one event".
  * The ChatGPT branch of `openai.generateOnce` uses `Post` with
    `Stream: true`.
* **Benchmark:** add `BenchmarkReadStream_4MiB`, over a stream built from
  `providers/openai/testdata/chatgpt-text.sse`'s events repeated to 4 MiB.
  It runs before and after the change on this host, with `-benchmem
  -count=5`. ~~Done when allocs/op after is at most a tenth of before.~~
  Both are recorded. *(Deviation 2026-10-04: the tenth applies to
  `BenchmarkReadStream_Deltas4MiB`, a 4 MiB long-answer stream of text
  deltas; the fixture benchmark's numbers are recorded as information.)*
* **Done when** the three red tests and the three `.sse` fixture tests in
  `providers/openai/chatgpt_test.go` pass.

#### 1.4 Retries (D1, T5, T6, T7, T14)

* **Red:**
  * `TestWithRetry_AfterReplyOnce` in
    `llmprovider/with_retry_test.go`. With `MaxAttempts: 5`, an
    after-reply failure is sent twice in all and then returned. It fails
    today: the failure is sent 5 times.
  * `TestWithRetry_TooLargeNotRetried`: `ErrReplyTooLarge` through
    `DecodeError` is sent once.
  * `TestWithRetry_WaitPastDeadlineReturnsAtOnce`: a 429 with
    `RetryAfter: 5s` under a 1 s deadline returns within 50 ms, keeping
    the `*APIError` and its `RetryAfter`. Today it waits 1 s and returns a
    bare `context deadline exceeded`.
  * `TestWithRetry_CancelledWaitKeepsError`: a cancel during the wait
    returns an error that matches both `context.Canceled` and the last
    `*APIError`.
  * `TestRetryPolicy_ServerWaitJitter`: 1,000 waits for `Retry-After: 1s`
    lie in `[1s, 1.25s)` and are not all equal. Today they are all 1 s.
  * `TestRetryPolicy_BackoffEqualJitterAfterCap`: at the cap, 1,000 waits
    lie in `[MaxDelay/2, MaxDelay]` and are not all equal. Today every one
    is `MaxDelay`.
  * `TestClassifyHTTPError_ShouldRetryTrue` in
    `llmprovider/retry_policy_test.go`. A 400 with `x-should-retry: true`
    is `Retryable()`, and its kind stays `ErrInvalidRequest`.
  * `TestClassifyHTTPError_Conflict`. A 409 from `openai` or `claude` is
    `ErrProviderUnavailable` and retryable; a 409 from `grok` stays
    terminal `ErrInvalidRequest`.
* **Change:**
  * `retryable` (`llmprovider/retry.go:127-144`):
    * an after-reply error is retryable when this `Generate` has not yet
      retried one. `(*retrying).Generate` keeps that flag. The retry
      counts against `MaxAttempts` and happens at most once, whatever the
      budget;
    * the kindless default (`:140-142`) becomes `errors.As(err,
      new(*url.Error))`, which is a send failure (0020 F9's wording).
  * `(*retrying).Generate` (`:80-104`), before the sleep: if
    `ctx.Deadline()` is sooner than the wait, return the last error at
    once. When the context ends during the wait, return
    `errors.Join(ctx.Err(), err)`.
  * `(RetryPolicy).wait` (`:108-121`):
    * a server-directed wait is `server + rand.N(max(server/10,
      250ms))`, allowed when `server <= MaxDelay`, as today;
    * backoff is `d := min(BaseDelay<<(attempt-1), MaxDelay)`, then
      `d/2 + rand.N(d/2+1)`, keeping today's guards for sub-nanosecond
      and negative values.
  * `ClassifyHTTPError` (`llmprovider/api_error.go:213-216`):
    `x-should-retry: true` sets a new unexported `shouldRetry`, which
    `Retryable()` checks first. `false` keeps setting `terminal`.
  * `classifyAPIError` (`:294-306`): a 409 from the `openai` or `claude`
    service is `false, ErrProviderUnavailable`.
  * The existing assertions that pin the old waits are updated to the new
    bounds and listed in the Execution record:
    `TestRetryPolicy_WaitBackoffAndCap` (`with_retry_test.go:142`) and
    `TestWithRetry_HonoursRetryAfter` (`:71`). D1 decided this behaviour
    change; it is not a loosened check.
  * `WithRetry`'s doc comment states the once-after-a-reply rule, the
    deadline rule and the jitter.
* **Records:** append "Amendment 2026-10-04: a failure after a 200 is
  retried once (0021 D1)" to
  `0020-MADR-remediate-v1-debugging-pass-findings.md`, amending Q2 (a).

#### Phase 1 live check

V3.1 under Verification.

### Phase 2: answers

#### 2.1 Tool arguments (W1)

* **Red:**
  * `TestDecode_ToolInputKeepsPrecision` in
    `llmprovider/internal/wire/messages/answer_test.go`. `tool_use.input`
    `{"z":1,"id":12345678901234567890}` gives `Arguments` byte-equal to
    `{"z":1,"id":12345678901234567890}`. Today it gives
    `{"id":12345678901234567000,"z":1}`.
  * `TestDecode_ToolInputMissingIsEmptyObject`, same file. A `tool_use`
    with no `input` gives `"{}"`; today it gives `""`.
  * `TestDecode_FunctionArgsKeepPrecision` in
    `llmprovider/internal/wire/generatecontent/answer_test.go`.
  * `TestDecode_ArgumentsAsObject` in
    `llmprovider/internal/wire/chatcompletions/answer_test.go`.
    `"arguments":{"city":"Paris"}` gives `{"city":"Paris"}`. Today the
    whole reply fails to decode.
  * `TestToolArguments_PassesValidJSONThrough` in
    `llmprovider/internal/wire/wire_test.go`. `{"id":12345678901234567890}`
    encodes byte-equal.
* **Change:**
  * `messages.go:128`: `Input json.RawMessage`.
  * `generatecontent.go:116-119`: `Args json.RawMessage`.
  * Both use a new `wire.CompactArguments(json.RawMessage) (string,
    error)`: empty or `null` gives `"{}"`, and anything else goes through
    `json.Compact`, the same rule as `gemini/interactions.go:179-190`,
    which then uses the helper too.
  * `chatcompletions.go:211` becomes a `chatArguments` type whose
    `UnmarshalJSON` accepts a JSON string as is, an object or array
    compacted, and `null` as `"{}"`.
  * `wire.ToolArguments` returns `any`:
    * a valid JSON object comes back as `json.RawMessage` (compacted);
    * empty gives `map[string]any{}`;
    * anything else keeps today's `{"arguments": s}` wrap.
* **Goldens:** expected unchanged, since `wirecase`'s arguments are a
  single key. Any change is listed.

#### 2.2 Finish reasons (W3)

* **Red:**
  * Messages: `stop_reason: model_context_window_exceeded` with a
    `tool_use` is `ErrIncomplete` with `Reason: "length"`. Today it is a
    valid call.
  * Messages: an unknown `stop_reason` `"new_reason"` is kept verbatim.
    Today it is `""`.
  * generateContent: `OTHER` is kept as `"OTHER"`. Today it is `""`.
  * Chat Completions: a reply with `tool_calls` and `finish_reason: stop`
    gives `FinishToolCalls`.
  * Chat Completions: `"role":"Assistant"` decodes as
    `llmprovider.RoleAssistant`.
  * An empty answer keeps its reason. Each case is an `*APIError` with
    `Kind` `ErrIncomplete` and a non-empty `Reason`:
    * Gemini `SAFETY` with no parts;
    * Claude `refusal` with no content;
    * Chat `content_filter` with no content.
  * Each case is a row in a new `TestFinish_*` table in each wire's
    `answer_test.go`.
* **Change:**
  * New `llmprovider/internal/wire/finish.go`:
    * `Finish(raw string, table map[string]llmprovider.FinishReason,
      hasCall bool) llmprovider.FinishReason`: the table value, else
      `FinishReason(raw)`; then `stop` with a call becomes `tool_calls`;
    * `EmptyAnswer(provider string, reason llmprovider.FinishReason)
      error`: `&llmprovider.APIError{Provider: provider, Kind:
      ErrIncomplete, Reason: string(reason), Message: "the answer has no
      content"}`.
  * Messages (`messages.go:31-39, 139`): the table gains
    `model_context_window_exceeded: FinishLength`.
  * generateContent (`generatecontent.go:183-197, 133`): it keeps every
    entry of today's table, including `FINISH_REASON_UNSPECIFIED: ""`.
  * Chat Completions (`chatcompletions.go:226`): it uses `Finish` with an
    empty table, so values are verbatim.
  * The no-content errors use `EmptyAnswer`: `messages.go:137, 182`,
    `generatecontent.go:131`, `chatcompletions.go:222, 258` and
    `gemini/interactions.go:193`.
  * `chatcompletions.go:243-247` always emits `llmprovider.RoleAssistant`.

#### 2.3 Responses empty output and refusals (W4)

* **Red** (`llmprovider/internal/wire/responses/answer_test.go`):
  * `{"status":"completed","output":[{"type":"reasoning","summary":[]}]}`
    is `ErrIncomplete`, from both `Decode` and `ReadStream`. Today both
    succeed with nothing.
  * A message with content `[{"type":"refusal","refusal":"I can't help
    with that."}]` gives a `MessageItem` with that text and
    `FinishContentFilter`. Today it is an empty success.
* **Change:**
  * `outputItem.Content` (`responses.go:120-123`) gains `Refusal string
    \`json:"refusal"\``.
  * `appendOutput` (`:138-147`) keeps a `refusal` part as text and
    reports it.
  * `Decode` (`:71-96`) and the `response.completed` case (`:233-242`)
    return `wire.EmptyAnswer` when the output is empty. They set
    `FinishContentFilter` when a refusal was kept.

#### 2.4 Gateway errors in a 200 (W5)

* **Red** (`chatcompletions/answer_test.go`):
  * `{"error":{"message":"Upstream overloaded","code":502}}` is
    `ErrProviderUnavailable` with that message.
  * A choice with `finish_reason: "error"` and `"error":{"message":"x"}`
    is classified the same way.

  Today both give `ErrIncomplete` "no choices" and "no usable content".
* **Change:** the reply struct (`chatcompletions.go:197-217`) gains a
  top-level and a per-choice `Error *struct{ Message string; Code
  json.RawMessage; Type string }`. Either one, or `finish_reason:
  "error"`, goes to `llmprovider.ClassifyStreamFailure(provider, code,
  type, message)`. `code` is the raw value with any quotes trimmed.

#### 2.5 Gemini call ids (W10)

* **Red** (`generatecontent/answer_test.go`):
  * two `weather` calls with ids `fc_1` and `fc_2` decode as CallIDs
    `fc_1` and `fc_2`;
  * without ids they decode as `weather#0` and `weather#1`.

  Today both give `weather`.
* **Change:**
  * `FunctionCall` (`generatecontent.go:116-119`) gains `ID string`.
  * `CallID` is the `ID`, or `name#index`, where the index counts the
    reply's function calls from 0.
  * The `functionResponse` name lookup (`:89-92`) falls back to the
    CallID with any `#<digits>` suffix removed.
  * Nothing new is sent. ~~So the goldens stay unchanged.~~ *(Deviation
    2026-10-04: the goldens also record the decoded result, and
    `opencode-zen-google/items.json` changes there.)*

#### 2.6 `ReasoningItem.Format` (D5, W7)

* **Red:**
  * `TestInput_SkipsForeignReasoning` in `responses/wire_test.go`: a
    `ReasoningItem{Encrypted: "e", Format: "messages"}` sends no
    reasoning item.
  * `TestMessages_SkipsForeignReasoning` in `messages/messages_test.go`:
    a `ReasoningItem{Signature: "s", Text: "t", Format: "responses"}` is
    encoded as a text-only item, with no signature.
  * Decoders set `Format`: each wire's reasoning fixture decodes with
    `Format` `responses`, `messages`, `generatecontent` or
    `chatcompletions`.
* **Change:**
  * `llmprovider/item.go`: `ReasoningItem` gains `Format string`. Its doc
    names the four values and the replay rule.
  * `llmprovider/internal/wire/wire.go` gains `FormatResponses`,
    `FormatMessages`, `FormatGenerateContent` and
    `FormatChatCompletions`.
  * Each decoder sets the field:
    * `responses.go:164`;
    * `messages.go:157, 159`;
    * `generatecontent.go:148`;
    * `chatcompletions.go:240`;
    * `gemini/interactions.go:166`, as `generatecontent`.
  * Encoders replay `Signature` and `Encrypted` only when `Format` is
    their own or empty, at `responses.go:42-51` and `messages.go:83-94`.
    A foreign item is treated as text-only.
  * The existing tests that compare decoded `ReasoningItem` values gain
    the field, listed in the Execution record.
  * *(Deviation 2026-10-04:)* every golden whose decoded result holds a
    `ReasoningItem` gains `Format` there, each listed.
  * `docs/architecture.md` gains the field's row.

#### 2.7 Encrypted reasoning when stateless (D4, W8)

* **Red:**
  * `TestBody_StatelessAsksForEncryptedReasoning` in
    `llmprovider/providers/openai/request_test.go`. With `WithStore(false)`
    the body has `include: ["reasoning.encrypted_content"]`; with no
    `WithStore`, and with `WithStore(true)`, it has none.
  * The same test in `llmprovider/providers/grok/request_test.go`.
  * `TestResponsesBody_AsksForEncryptedReasoning` in
    `llmprovider/providers/opencode/request_test.go`. Every
    responses-route body has the `include`, and `reasoning.summary:
    "auto"` whenever `reasoning` is sent.
* **Change:**
  * `openai.go` (`:206-207`) and `grok.go` (`:182-183`) set the `include`
    when `p.store` is non-nil and false.
  * `opencode.go` `responsesBody` (`:358, 378`) always sets the `include`,
    and adds `"summary": "auto"` to `reasoning`.
* **Goldens:**
  * new cases: `Case{Name: "openai-stateless"}` with `WithStore(false)` in
    `providers/openai/wire_test.go`, and `grok-stateless` in
    `providers/grok/wire_test.go`;
  * changed goldens: `providers/opencode/testdata/wire/opencode-zen-responses/*.json`
    and `opencode-go-responses/*.json`, every scenario except `listing`;
  * the existing `openai/` and `grok/` cases do not change (MADR
    amendment, item 3).

#### 2.8 Kilo `reasoning_details` (W9): measured first

* **2.8a, probe.** A scratch live test, not committed, runs on a scratch
  copy with `KILO_API_KEY`, checked for presence only. It sends one
  reasoning prompt with a tool to an Anthropic thinking model and to a
  Gemini 3 model from the live Kilo listing. It records only counts: the
  number of `reasoning_details` entries per reply, and each entry's
  `type` and `format`. No content or token is printed.
* **2.8b, only if 2.8a shows the field:**
  * **Red:** `TestDecode_ReasoningDetailsKept` and
    `TestBody_ReplaysReasoningDetails` in `chatcompletions/`, on a fixture
    shaped like the probe's entries.
  * **Change:**
    * each entry decodes to a `ReasoningItem` with `Text` (the entry's
      `text` or `summary`), `Encrypted` (the raw entry, compacted) and
      `Format: chatcompletions`;
    * a new `Opts.ReplayReasoningDetails`, set by `kilo` only, replays
      those raw entries in order as the assistant message's
      `reasoning_details`;
    * `Format` keeps them off the other wires;
    * the doc at `chatcompletions.go:183-195` changes with them.
* **If 2.8a shows no field** on either model, W9 is recorded as not done,
  with the probe's counts.
* **If the shape differs** from OpenRouter's documented one, stop and
  prompt.

#### Phase 2 live check

V3.2 under Verification.

### Phase 3: credentials

#### 3.1 Token-aware invalidation (D5, T1)

* **Red:**
  * `TestReauth_LateRefusalKeepsFreshToken` in
    `llmprovider/internal/wire/reauth_test.go`. Eight concurrent sends on
    one token all get a 401; the source is refreshed exactly once.
  * `TestOAuthSession_InvalidateTokenIgnoresStale` in
    `llmprovider/auth/oauth_session_invalidate_test.go`.
  * `TestCommandToken_InvalidateTokenIgnoresStale` in
    `llmprovider/command_token_test.go`.
  * `TestClaude_ConcurrentRefusalsRunCommandOnce` in
    `llmprovider/providers/claude/`. Through `claude.New` with a
    `CommandToken`, 4 concurrent 401s on the first token give 2 command
    runs. Today they give 4.
* **Change:**
  * `llmprovider/command_token.go`, next to `InvalidatingSource`
    (`:30-33`): `type TokenInvalidator interface {
    InvalidateToken(Token) }`, documented as a no-op unless the refused
    token is still the current one.
  * `(*CommandToken).InvalidateToken`: clears only when `c.value ==
    t.Value`.
  * `(*auth.OAuthSession).InvalidateToken`: expires only when `s.Access ==
    t.Value`.
  * `wire.Reauth`: on a 401 it calls `InvalidateToken(token)` when the
    source has it, else `Invalidate()`.
  * `docs/architecture.md` gains the rows (`:140, 263, 311`).

#### 3.2 Early refresh and short-lived tokens (T2, T3)

* **Red** (`llmprovider/auth/oauth_refresh_test.go`):
  * `TestOAuthSession_FailedEarlyRefreshKeepsToken`. The token is valid
    for 4 more minutes and the issuer answers 503. `Token` returns the
    current token with no error. A second call within 10 s sends nothing.
    Today it gives an empty token, an error, and 3 attempts on every call.
  * `TestOAuthSession_AuthFailureStillFails`. An issuer 401 still returns
    `ErrAuthFailure` and no token.
  * `TestOAuthSession_ShortLivedTokenNotRefreshedEachCall`. The JWT has
    `iat = now` and `exp = now + 240s`; 10 `Token` calls make 0 refreshes.
    Today they make 10.
* **Change** in `llmprovider/auth/oauth_session.go`:
  * an unexported clock, ~~`now func() time.Time`~~ (nil means `time.Now`),
    used everywhere the file calls `time.Now` or `time.Until`. *(Deviation
    2026-10-04: the field is `now sessionClock`, an interface with `Now()
    time.Time`, since a func field makes `OAuthSession` incomparable and
    fails `api-check`.)*
  * `refreshMargin()` is `min(oauthRefreshSkew, lifetime/2)`. The
    lifetime is `exp − iat` from the access JWT, or `Expiry − issued`
    when `issued` is set. `issued` is a new unexported field, set when a
    refresh or exchange succeeds. Otherwise the margin is
    `oauthRefreshSkew`;
  * `currentToken` (`:241-253`) uses it;
  * in `Token`, when a refresh fails with an error other than
    `ErrAuthFailure` and more than `oauthEarlyRefreshFloor` (30 s)
    remains:
    * set `nextRefresh = now + 10s + rand.N(20s)`;
    * log at Warn;
    * return the current token.

    Before `nextRefresh`, with more than 30 s left, `Token` returns the
    current token without refreshing.

#### 3.3 The refresh decode (T4)

* **Red** (`llmprovider/auth/oauth_rotation_test.go`). Each case must keep
  and save the rotated refresh token; today each loses it:
  * a 200 with `"expires_in":"3600"`;
  * `"expires_in":3600.0`;
  * `"expires_in":"abc"` with both tokens present, which takes the 3600 s
    default.

  A 2 MiB 200 body must be an error, after reading at most
  `oauthResponseLimit` + 1 bytes.
* **Change:**
  * `oauthRefreshResponse.ExpiresIn` (`oauth_session.go:91-95`) and
    `oauthTokenResponse.ExpiresIn` (`oauth_loopback.go:81`) become
    `oauthSeconds` (`oauth_device.go:30-51`);
  * the refresh body is read through `io.LimitReader(resp.Body,
    oauthResponseLimit+1)`, and anything longer is an error;
  * when the full decode fails, a second decode into `{access_token,
    refresh_token}` keeps both tokens if both are present.

#### 3.4 The refresh lock (T8, T9, T10)

* **Red:**
  * `TestOAuthSession_RefreshAttemptBounded`. The issuer stalls the body.
    Each attempt ends at `oauthRefreshAttemptTimeout`, shortened to 50
    ms. There are at most 3 attempts, and the lock is released.
  * `TestFileTokenStore_SkewedHeartbeatKeepsLock` in
    `llmprovider/auth/tokenstore_lock_owner_test.go`. A holder keeps
    touching the lock with an mtime 40 s in the past, advancing every 20
    ms. A waiter with `staleAfter` 200 ms and `wait` 300 ms times out and
    does not take over. Today it takes over at once.
  * `TestFileTokenStore_HeartbeatSurvivesReadError`. With the `lockRead`
    seam failing once, the heartbeat keeps advancing the mtime.
  * `TestFileTokenStore_TakeoverRenameRetried`. With the `lockRename`
    seam failing once, `LockRefresh` still succeeds.
  * `TestOAuthSession_PendingResaveDoesNotBlock` in
    `llmprovider/auth/oauth_spent_refresh_test.go`. Another holder keeps
    the lock, `spentRefresh` is set, and the token is valid. Five `Token`
    calls finish in under 100 ms in all, with at most one lock attempt.
    Today each waits `wait`.
* **Change:**
  * `oauthRefreshAttemptTimeout = 15 * time.Second` wraps each attempt in
    `refreshOAuthSessionOnce` (`:347`), replacing step 1.2's 30 s for the
    refresh.
  * In `llmprovider/auth/tokenstore_file.go`:
    * `lockWait` becomes 40 s, which is more than `lockStaleAfter` plus
      `lockHeartbeat` (35 s);
    * `LockRefresh` (`:220-258`) records each new mtime it sees with the
      waiter's monotonic `time.Now()`. A lock is stale when its mtime has
      not changed for `staleAfter` by that clock;
    * a takeover error `continue`s the loop instead of returning
      (`:241-243`). The deadline error wraps the last takeover error;
    * after the first failed attempt, `LockRefresh` returns at once if
      `ctx` is already done. Its doc states that a done context tries
      once;
    * `holdLock` (`:327-352`) skips a tick on a read error, and stops only
      on a token mismatch or on unlock;
    * new seams `lockRead = readLockToken` and `lockRename = os.Rename`.
  * In `oauth_session.go`, `persistRotation` (`:192-205`):
    * runs at most every 30 s (`nextResave`);
    * calls `LockRefresh` with an already-cancelled context, so it tries
      once without waiting.

#### 3.5 Device logins (T11)

* **Red**, with `fakeOAuthClock` (`llmprovider/auth/oauth_device_test.go:198`):
  * Kilo: a 502, then a 200, signs in.
  * OpenAI: a dropped connection, then a 200, signs in.
  * Grok: a 503 with an HTML body, then a 200, signs in.
  * The poll that ends at expiry returns "device code expired", wrapping
    the last transient error.

  Today the first three fail on the first transient answer.
* **Change:** the three loops (`oauth_device.go:206-250, 289-331`,
  `kilo_device.go:78-118`) treat a transport error, a 429 and any 5xx as
  transient:
  * the backoff starts at the poll interval and doubles up to 60 s,
    through `sleepBeforeDeadline`;
  * it resets after a normal answer.

  RFC 8628 §3.5.

#### 3.6 The token directory (T15)

* **Red** (`llmprovider/internal/ownerperm/ownerperm_unix_test.go`):
  * a directory at 0o777 is 0o755 after `MkdirAll`; today it stays
    0o777;
  * a directory the user does not own, through the `ownerOf` seam, is an
    error;
  * a symlink at the path is an error.
* **Change** in `ownerperm_unix.go`. After `os.MkdirAll(dir, 0o700)`:
  * `os.Lstat` must show a directory;
  * the owner must be `os.Getuid()`, read through a new seam `ownerOf =
    func(os.FileInfo) (uid int, ok bool)`;
  * a group or other write bit is cleared with `Chmod(mode &^ 0o022)`.

  The doc comment drops "An existing directory keeps its mode". Windows
  is unchanged. The package floor stays at 100%.
* **Records:** append "Amendment 2026-10-04: the token directory on Unix
  (0021 T15)" to `0010-MADR-windows-stdio-oauth-tokenstore-ci.md`,
  extending D10.

### Phase 4: catalog

#### 4.1 The metadata cache (C1, C2)

* **Red** (`llmprovider/catalog/metadata_request_test.go`):
  * `TestLookupMetadata_CallerDeadlineNotCached`. A lookup under a 1 ms
    deadline fails. The next lookup, with time, gets the document. Today
    it gets the cached failure for a minute.
  * `TestLookupMetadata_OneFetchForConcurrentLookups`. 20 concurrent cold
    lookups make 1 request. Today they make 20.
  * `TestLookupMetadata_StaleServedWhileRefreshing`. After the TTL, by
    the clock seam, 10 lookups return the stale document at once and make
    1 background request.
* **Change** in `llmprovider/catalog/model_metadata.go`:
  * an inflight map, `url → *metadataFetch{done chan struct{}; doc; err}`,
    guarded by `modelMetadataMu`;
  * the fetch runs in a goroutine under
    `context.WithTimeout(context.WithoutCancel(ctx),
    metadataFetchTimeout)`, with `metadataFetchTimeout = 10 *
    time.Second`;
  * each waiter selects on `done` and on its own context, which keeps its
    5 s `metadataLookupTimeout`. *(Deviation 2026-10-05: three existing
    tests change with this step; see the Execution record.)*
  * past the TTL, a cached document is returned at once, and a background
    fetch starts if none is in flight;
  * every failure of the detached fetch is cached for
    `modelMetadataRetryAfter`, and a waiter's own context ending is never
    cached (MADR amendment, item 7). The `context.Canceled` special case
    (`:246-252`) goes;
  * a clock seam, `metadataNow = time.Now`.

#### 4.2 Read limits (C3)

* **Red:** a 33 MiB metadata document, and a 9 MiB page from each lister,
  each give an error naming the limit. Today they decode without limit.
* **Change:**
  * `decodeLimited(r io.Reader, limit int64, v any, what string) error`
    reads `limit+1` bytes;
  * `metadataLimit = 32 << 20` applies at `model_metadata.go:301`;
  * `listingPageLimit = 8 << 20` applies at `discovery.go:244, 338, 392,
    492, 607, 785`, and replaces `togetherListingLimit` (`:665, 696`).

#### 4.3 OpenAI's filter (C4)

* **Red** (`models_catalog_test.go`):
  * `gpt-4o-mini-tts`, `gpt-4o-transcribe`, `gpt-4o-search-preview`,
    `o3-deep-research` and `gpt-5-search-api` are not usable;
  * `gpt-6-sol`, `gpt-4.1-mini` and `o3` still are.
* **Change:** `isUsableOpenAIChatModel` (`:329-350`) also denies an id
  containing the token sequences `tts`, `transcribe`, `search`,
  `deep-research` or `realtime`. Tokens are the id split on `-`, `.`,
  `_` and `/`, matched by a new `hasTokens(id string, seq ...string)
  bool`. *(Deviation 2026-10-05: `gpt-6-sol` is not
  usable on `HEAD`, since `openaiAllowPrefixes` has no `gpt-6`. The step
  also adds `gpt-6` to that list, ranked like `gpt-5`.)*

#### 4.4 Name rules match tokens (C5)

* **Red:**
  * `rankOpencodeModel("gemini-3.1-pro")` includes the −300 `-pro`
    penalty;
  * `rankOpencodeModel("minimax-m3")` gets no `mini` bonus;
  * the same two in `rankKiloModel`, which has the same `mini` match at
    `:658`.
* **Change:** the name switches in `rankOpencodeModel` (`:272-299`) and
  `rankKiloModel` (`:650-665`) test tokens with `hasTokens`, not
  substrings.

#### 4.5 Generations parsed numerically (C6)

* **Red:**
  * `gemini-3.8-flash` > `gemini-2.5-flash`;
  * `gemini-3.10-flash` > `gemini-3.9-flash` > `gemini-3.1-flash`;
  * `grok-4.7` > `grok-4.6`, and `grok-5` > `grok-4.7`;
  * `claude-opus-5` > `claude-opus-4-8`.

  Each is a comparison of the provider's `rank…Model`.
* **Change:** every id that today's tables match keeps its score. Only
  the ids they miss, or match by accident, change.
  * `parseGeneration(id string) (major, minor int, ok bool)` reads the
    first `N`, `N.M` or `N-M` version after the family name, as integers.
    The order key is `ord = 100*major + minor`.
  * Today's hint tables stay, and a hint that matches still wins, with
    two changes:
    * a version hint matches as a whole token, so `"3.1"` no longer
      matches `gemini-3.10-flash`;
    * the family catch-alls, the prefixes `grok-4`, `grok-3` and
      `gemini-3-`, act only as anchors. So `grok-4.1` still scores 40,
      while `grok-4.7` is above every anchor.

    The tables are `rankGeminiModel` `:460-473`, `rankClaudeModel`
    `:517-526` and `rankGrokModel` `:558-568`.
  * Each hint is an anchor at its generation's `ord`:
    * Gemini: 307→100, 306→90, 305→80, 301 and 300→40, 205→30;
    * Grok: 406→60, 405→50, `grok-4`→40 at 400, `grok-3`→30 at 300;
    * Claude: `sonnet-5`→40 at 500, `opus-4-8`→35 at 408, `sonnet-4-6`→35
      at 406, `4-5` / `4.5`→40 at 405, `haiku-4`→40 at 400, `3-5`→10 at
      305.
  * An id that no hint matches takes the value of the highest anchor at
    or below its `ord`. *(Deviation 2026-10-05: capped at one less than the
    lowest anchor above its `ord`, since Claude's anchors do not rise with
    the generation.)*
  * An `ord` above every anchor gets `newest + 10 + min(40, steps)`:
    * `steps = 20·(major − newestMajor) + minor`, when the majors differ;
    * `steps = minor − newestMinor`, when they are equal.

    So `gemini-3.8-flash` is 111, `gemini-3.10-flash` 113, `grok-4.7` 71
    and `grok-5` 90.
  * An `ord` below every anchor keeps today's default: 0, or −2000 for a
    shut-down Gemini generation. The shut-down rule becomes
    `geminiShutDown = []string{"2.0", "1.5"}`.
  * `claude-opus-5` takes the 500 anchor (40), so it scores 140 against
    `claude-opus-4-8`'s 135.
* **Guard:** the existing `TestRankGemini…`, `TestRankGrok…` and
  `TestRankClaude…` assertions and the snapshot golden stay unchanged
  (Rule 4). An id that no hint matched before and now takes an anchor,
  such as `claude-opus-4-6`, is listed in the Execution record.

#### 4.6 The fill and the diversity group (D3: C7, C8)

* **Red:**
  * `TestRankRecommended_FillSkipsRequestFailures` in
    `model_ranking_test.go`. A sparse Go set fills without
    `-contributor` or the region-gated DeepSeek ids. Today it fills with
    them.
  * `TestOpencodeGoFallback_AppliesItemsSevenAndEight` in
    `discovery_ranking_test.go`. The metadata-failure fallback curation
    excludes them.
  * `TestRankGroup_FamilyLeadingLetters`. `gpt-sol`, `gpt-luna` and
    `gpt-astra` are one group, and a prefixed id keeps its vendor group.
* **Change:**
  * `excludedAtRequest(provider ProviderID, id string) bool` holds items 7
    and 8, today inline in `eligible` at `model_ranking.go:115, 118`;
  * `eligible`, the fill (`:230-237`, next to `utilityExcluded`) and Go's
    fallback closure (`discovery.go:519-529`) all use it;
  * `rankGroup` (`:243-252`): an unprefixed id groups by the `^[a-z]+` of
    its lower-cased family, then by the family, then by the id.
* **Snapshot golden:** `TestListModelCatalog_Snapshot20260926` changes in
  the Zen capable six only: `gpt-6-astra` gives way to
  `claude-fable-5-1`. The new order is recorded. Any other change is a
  deviation.
* **Records and docs:**
  * append "Amendment 2026-10-04: the fill and the diversity group (0021
    D3)" to `0009-MADR-use-case-aware-default-model-ranking.md`. It covers
    the §4 fill, the §4 group, and §3 item 9 kept on the owner's
    condition;
  * update README's Ranking bullet.

#### 4.7 Metadata guards (C9)

* **Red** (`ranking_edge_test.go`):
  * a negative cost, a NaN or +Inf cost, and a 1e308 + 1e308 cost each
    rank as unknown;
  * a future `release_date` gives an unknown age;
  * `kiloPriceRank("NaN")` is `math.MaxFloat64`;
  * the comparator is a strict weak order over a set with each kind,
    checked pairwise.
* **Change:**
  * `metadataCandidate` (`model_metadata.go:345-353`) marks the cost
    unknown unless it is finite and ≥ 0 after the sum, and the age
    unknown when the date is after `metadataNow()`;
  * `kiloPriceRank` (`discovery.go:750-756`) returns `kiloPrice`'s value,
    or `math.MaxFloat64`.

#### 4.8 Search (C11, Z11)

* **Red** (`model_matcher_test.go`):
  * `"fast"` matches no OpenAI model by its bracketed annotation;
  * `"gpt41"` finds `gpt-4.1`;
  * `"o3"` returns only the ids with an `o3` token or prefix, not
    subsequences;
  * among equal scores the listing order is kept;
  * `"kilo-auto/*"` ranks an id match before a label-only match;
  * `wizard`: a query with no match offers "Use `<q>` as the model id",
    which returns it.
* **Change** in `llmprovider/catalog/model_matcher.go`:
  * the label tier (`:124`) matches the display name only: the label
    before its first `[`, trimmed;
  * a new tier, `scoreCompactSubstring = 3500`, compares the query and
    the id with `-`, `.`, `_`, `/` and spaces removed;
  * the subsequence tier runs only when no id matched a stronger tier,
    and only for queries of 3 or more characters;
  * ties break by input index (`:56-64`);
  * glob matches score 2 for an id match and 1 for a label-only match,
    then sort by input index;
  * `Search`'s doc (`:36-39`) states the new order.

  In `wizard/model_select.go:45-49`, a search with no match offers a
  `Select` with "Use `<q>` as the model id" and "Search again".

#### 4.9 Helpers and labels (C12)

* **Red** (`model_label_test.go`, `catalog_state_test.go`):
  * `Rank("Gemini", id) == Rank("gemini", id)`;
  * `Rank(ProviderTogether, staticTogether[0]) > Rank(ProviderTogether,
    staticTogether[1])`;
  * every id in every provider's `Static` has a label, or is in
    `unlabelledStaticIDs`;
  * every `modelLabels` key is in some `Static` list.
* **Change:**
  * `Rank` (`models_catalog.go:183-201`) lower-cases the provider, and
    ranks Together by static position (`len − index`), the order
    `curateTogether` keeps;
  * the missing labels are added, or listed with a reason;
  * orphan labels are removed.

#### 4.10 Empty curation (C13) and D3's condition

* **Red:**
  * `TestCatalogFrom_EmptyCurationKeepsLiveListing` in
    `discovery_catalog_test.go`. A Kilo listing of only `kilo-auto/small`
    and `kilo-auto/balanced`, under the utility profile, gives
    `Live=true`, `Usable` equal to the listing, an empty `Recommended`,
    and `Err=nil`. Today it gives `Live=false` and the static six.
  * `TestKiloAuto_ExcludedButChoosable` in `kilo_catalog_test.go`, for
    D3's condition. Under the utility profile, a full snapshot listing
    and the two-tier listing each have no `kilo-auto/*` in
    `Recommended`, every tier in `Usable`, and each tier found by
    `Search(id)` and by `Search("kilo-auto/*")`.
  * `TestConfigure_EmptyRecommendedOpensSearch` in `wizard/`. A live
    catalog with an empty `Recommended` and a non-empty `Usable` goes
    straight to search over `Usable`, and a tier can be picked. Today the
    wizard swaps in the static catalog (`configure.go:377-378`).
* **Change:**
  * `catalogFrom` (`discovery.go:98-101`) returns `Catalog{Usable: usable,
    Live: true}` when curation is empty;
  * the `Catalog` doc (`:36-51`) says `Recommended` may be empty on a
    live listing;
  * `wizard/configure.go:377` keeps a live catalog with a non-empty
    `Usable`;
  * `configure.go:174` asks for a manual id only when `Usable` is empty
    too;
  * `wizard/model_select.go:43` skips the recommended menu when it is
    empty, and re-prompts the search with a notice that no model meets
    the profile.

  This is the MADR amendment, item 5.

#### 4.11 The OpenCode route table (C10, W13)

* **Red:** `TestRouteHeuristic_AgreesWithEveryRow` in
  `llmprovider/providers/opencode/metadata_route_test.go`. For every
  `(gateway, model, npm)` in the snapshot, `heuristicRoute(model,
  gateway) == routeForNPM(npm)`. It is a new gate: it is seen to fail
  with one planted disagreeing row in a scratch copy, and then passes on
  the tree.
* **Change:**
  * `git mv llmprovider/providers/opencode/testdata/opencode-routes.json
    llmprovider/providers/opencode/routes_snapshot.json`;
  * `//go:embed routes_snapshot.json`;
  * `routeTable` (`route.go:76-203`) is built in `init` with `routeForNPM`,
    replacing the hand table and its `//nolint:goconst` (`:75`);
  * `TestOpencodeRouteTable_MatchesMetadataSnapshot` (`:149-180`) reads
    the embedded file.

#### Phase 4 check

No live check is needed: the snapshot fixtures cover the ranking. A live
Kilo and OpenCode listing runs as a smoke test, if the keys are present,
and its counts are recorded.

### Phase 5: wizard and redaction

#### 5.1 Redaction cost (Z1)

* **Benchmarks first**, before the change, recorded with `-benchmem
  -count=5`:
  * `BenchmarkRedact_Clean4KiB` in `internal/redact/redact_test.go`;
  * `BenchmarkClassifyHTTPError_64KiBHTML` in
    `llmprovider/api_error_test.go`.
* **Change:**
  * `ClassifyHTTPError` and `ClassifyStreamFailure` cut the message to
    `apiErrorRedactLimit = 16 << 10`, on a rune boundary, before
    `redact.String`. `boundMessage` still bounds it at 512 bytes
    (`api_error.go:59`);
  * in `internal/redact/redact.go:70-98`, each regex is gated on a
    case-insensitive literal prefilter of its anchors (for example
    `bearer`, `basic`, `token`, `authorization`, `eyJ`, `sk-`, `xai-`,
    `-----BEGIN`), instead of `Match()`.
* **Done when** both benchmarks are at least 10× faster than before, with
  the numbers recorded, and every redact test is unchanged and green.

#### 5.2 Redaction coverage (Z2)

* **Red**, rows in `internal/redact/redact_test.go`.

  Must redact:
  * `{"accessToken":"abcd1234efgh5678"}`, and the same for
    `refreshToken`, `idToken` and `sessionToken`;
  * a PEM private key, from BEGIN to END inclusive;
  * `Cookie: a=secret1234; b=secret5678`, both values;
  * `password="two words here"`, the whole quoted value;
  * `grant_type=refresh_token&refresh_token=abc%2Fdef12345`;
  * `signature=abcdef123456`;
  * `"private_key":"…"`.

  Must keep:
  * `{"code":"context_length_exceeded"}`;
  * `'code': 'model_not_found'`;
  * `token: 128000`.

  A row that passes today stays as a guard and is noted.
* **Property test:** `TestRedact_PlantedSecrets`, with seeded
  `rand.New(rand.NewPCG(1, 2))`. It runs 1,000 cases, each planting one
  secret of each class in random JSON, form, header and prose contexts,
  and asserts the secret is absent from the output.
* **Change:**
  * `reKV` (`:38`) gains the camelCase token keys, `apiKey`,
    `clientSecret`, `private[_-]?key` and `signature`;
  * the value runs to the closing quote when it is quoted, and the
    cookie rule repeats per pair;
  * the PEM rule spans BEGIN to END;
  * *(Deviation 2026-10-05: the Google key rule loses its trailing `\b`, so
    a key ending in `-` or `_` is redacted.)*
  * `reKVLong` (`:43`) and the bare `token` key skip values that are
    plain numbers or snake_case words (`^[a-z]+(_[a-z]+)+$`);
  * the package stays at 100% coverage.

#### 5.3 Control characters (Z3)

* **Red:**
  * `ClassifyHTTPError` on
    `{"error":{"type":"x\u001b[31m","message":"\u001b]0;owned\u0007\u001b[2Jhello"}}`
    gives `Error()`, `Code` and `Message` with no byte below 0x20 except
    `\t`, no 0x7f and no U+0080–U+009F. `Code` is at most 128 bytes;
  * the OAuth callback's `error` and `code` with ESC are stripped;
  * `TextPrompter.Notify` with an ESC argument writes none.

  Each fails today.
* **Change:**
  * `internal/redact/control.go` gains `StripControl(string) string`;
  * it applies to `Message` and `Code` in `ClassifyHTTPError`
    (`api_error.go:198-204`) and `ClassifyStreamFailure` (`:236`), with
    `apiErrorCodeLimit = 128`;
  * it applies to the callback's `error` (`oauth_loopback.go:551`) and
    `code` (`:508, 591`);
  * `TextPrompter.Notify` (`wizard/text_prompter.go:150-160`) applies it
    to the formatted text.

#### 5.4 Masked entry (Z6)

* **Red** (`wizard/text_prompter_test.go`, `TextPrompter` over a reader):
  * a 2,000-character paste writes at most 2 redraws;
  * the output has no `\033[K`;
  * a lone ESC followed by `a` keeps the `a`;
  * Ctrl-U clears the entry;
  * Ctrl-C returns an error that matches `context.Canceled`.

  Each fails today.
* **Change** in `readMasked` (`:379-424`):
  * redraw only when the reader has nothing buffered;
  * pad with spaces to the previous width, then return the cursor, in
    place of `\033[K`;
  * `skipEscape` (`:440-460`) ignores a lone ESC when nothing is
    buffered;
  * Ctrl-U (0x15) clears;
  * Ctrl-C returns `fmt.Errorf("wizard: cancelled: %w",
    context.Canceled)`.
* **Windows:** V3.5.

#### 5.5 Base URLs (Z7)

* **Red** (`wizard/configure_test.go`):
  * `api.example.com`, with no scheme, is refused and asked again;
  * `https://u:p@host`, `https://host/?q=1` and `https://host/#f` are
    refused;
  * `http://remote.example` asks for confirmation, and a "no" asks again;
  * `http://localhost:11434` is accepted without asking;
  * the same rules apply to the Kilo base URL (`wizard/auth.go:240`).
* **Change:** a new `validateBaseURL(p Prompter, raw string) (string,
  error)` in `wizard/`:
  * `url.Parse`;
  * the scheme must be `http` or `https`, with a host;
  * userinfo, a query and a fragment are refused;
  * `http` to a host that is not loopback asks to confirm.

  It is called from `resolveBaseURL` (`configure.go:254-260`) for remote
  descriptors, and from the Kilo path.

#### 5.6 Pasted credentials (Z10)

* **Red** (`wizard/auth_test.go`):
  * as an OpenAI credential, `xai-…` is refused with "this looks like an
    xAI key";
  * `sk-ant-…` is refused as an Anthropic key, checked before `sk-`;
  * a non-JWT string is refused;
  * a JWT with a past `exp` is refused;
  * a JWT with a future `exp` is saved as today.
* **Change** in `resolveTokenStdin` (`wizard/auth.go:339-357`):
  * vendor prefixes, in order: `sk-ant-`, then `xai-`, `tgp_`, `hf_` and
    `AIza`;
  * the JWT shape check: three base64url segments, with a header that is
    JSON with `alg`;
  * `exp`, when present, must be in the future.

  The session is still stored with no `Expiry`, so
  `ValidateOAuthSession`'s rule (`oauth_session.go:64-67`) is unchanged
  (MADR amendment, item 8).

### Phase 6: tooling and tests

#### 6.1 Dependencies (Z4)

* **Planted breach**, in a scratch copy: a `_test.go` importing
  `example.com/evil`, with a matching `go.mod` require. dep-check must
  fail. Today it passes.
* **Change** in `scripts/check_deps.py`:
  * read `go mod edit -json`, where each `Require` path must be in
    `{golang.org/x/term, golang.org/x/sys}`;
  * repeat the import check with `go list -test -deps`, stripping the `
    [pkg.test]` suffix.

#### 6.2 Gates that can pass while broken (Z9)

* **Planted breaches**, in a scratch copy:
  * a `//go:generate … -output=zsyscall_windows.go` directive must make
    generate-check check that file, and a directive with no output must
    fail with "0 generated files";
  * a mapping cell `TBD` must fail parity-check.
* **Change:**
  * `scripts/check_generated.py:38-42` parses `-output=X` and `-output
    X`, and fails when it checks no file;
  * `scripts/check_parity_map.py` `check` requires each cell to have a
    backticked span that resolves to a Go name or an SDK package path, or
    to be exactly one of the markers the guide already uses: `none`,
    `removed`, `Here` (MADR amendment, item 12);
  * new `scripts/test_gates.py` (`unittest`, standard library). It copies
    the tree to a temp dir, plants one breach per gate (dep-check,
    generate-check, parity-check, coverage-check, api-check), and asserts
    each gate exits non-zero, then 0 on the clean copy;
  * new `make gate-selftest`, which runs `python3 -B -m unittest
    scripts/test_gates.py`. `.gitignore` already ignores `__pycache__/`
    and `*.py[cod]`.

#### 6.3 CI, precheck, floors and `nolintlint` (Z5)

* **Change:**
  * `.github/workflows/ci.yml`:
    * `go test -shuffle=on ./...` and `go test -race -shuffle=on ./...`;
    * a govulncheck step, `go run golang.org/x/vuln/cmd/govulncheck@<pin>
      ./...`, pinned to the newest release at execution, which is
      recorded;
    * `make gate-selftest` with the other gates;
    * `timeout-minutes: 30` on the job;
    * `concurrency: {group: ci-${{ github.ref }}, cancel-in-progress:
      ${{ github.event_name == 'pull_request' }}}`;
    * `schedule: [{cron: '17 6 * * 1'}]` for a weekly run.
  * `scripts/go-precheck.sh:119` adds `-race`, and a `go mod tidy -diff`
    step follows.
  * `scripts/coverage-floors.txt`: each listed floor becomes the lower of
    the local and the CI Linux measurement, minus 2 points, rounded down
    to 0.1. The measurements are recorded. A floor already at 100.0 stays.
  * `.golangci.yml` enables `nolintlint` (`require-specific: true`,
    `allow-unused: false`). Each directive it reports is removed:
    `models_catalog.go:675` is expected, and `:33` is checked;
    `route.go:75` is gone with step 4.11.
* **Check:** the CI change is seen working on the branch's first push.
  The govulncheck step is seen to fail on a scratch copy pinned to a
  module version with a known advisory.

#### 6.4 The test fake (Z8)

* **Change** in `wizard/fake_prompter_test.go`:
  * add `newFake(t *testing.T, f fakePrompter) *fakePrompter`, which
    registers `t.Cleanup` to fail on any scripted answer left over,
    unless the new `allowLeftover` field is set with a comment saying
    why;
  * convert the 90 `fakePrompter{` literals in 14 files with a
    scratchpad script;
  * fix the stale scripts the check finds, expected to be 9, each listed
    with its correction.
* **Check:** a planted extra answer in one test fails it.

#### 6.5 Timing tests (Z12)

* **Change:**
  * the waiter join is made observable by an unexported `onJoin func()`
    hook on `CommandToken` (`command_token.go:86-116`) and on
    `OAuthSession` (the future at `auth/helpers.go:58-66`), called when a
    waiter joins the in-flight fetch; *(Deviation 2026-10-05: on
    `OAuthSession` the hook is `onJoin joinHook`, an interface, as the
    phase 3 clock is.)*
  * `TestCommandToken_WaiterOutlivesLeaderCancel`
    (`command_token_leader_test.go:37`) and
    `TestOAuthSession_WaiterOutlivesLeaderCancel`
    (`oauth_session_leader_test.go:58`) wait on it, replacing the 50 ms
    sleeps;
  * the `CommandToken` tests call `t.Parallel()`.

  `testing/synctest` is not used, because `CommandToken` runs a real
  process.
* **Check:** with the hook call removed in a scratch copy, both tests
  fail by timeout. They also pass `-count=50 -race`.

#### 6.6 The conformance harness (D5, W12)

* **Change:** `llmprovider/llmtest/llmtest.go` `Harness` (`:27-48`) gains
  optional fields. Each switches on one check:
  * `Fidelity bool`: the request carries `Model:
    "llmtest-model-7f3a"`, `Instructions: "llmtest-instructions-7f3a"`,
    and a `FunctionCallOutputItem{Output: "llmtest-output-7f3a"}`. The
    raw body must contain all three;
  * `Garbled func(w, r)`: an undecodable 200 must be `ErrIncomplete`,
    sent once through `WithRetry`;
  * `Truncated func(w, r)`: a cut answer with a partial call must be
    `ErrIncomplete` with `Reason` `length`; *(Deviation 2026-10-05: with
    the `Reason` the harness names in `TruncatedReason`, since `Reason` is
    the service's own.)*
  * `StrictTools bool`: a `ToolCall` reply must have `FinishToolCalls`
    and non-empty, valid JSON `Arguments`.

  Each provider's `llmtest_test.go` sets all four.
* **Check:** each check is seen to fail on a scratch copy with its fault
  planted in one provider: a dropped `Instructions`, a `DecodeError`
  bypass, a missing length check, and an `Arguments ""`. A harness that
  sets none still passes.
* `docs/architecture.md` gains the field rows.

#### 6.7 Close-out docs

* `AGENTS.md`'s CI list gains `gate-selftest`.
* `docs/architecture.md`, `README.md` and the guides say what the code
  does. A guide's changes are checked by G8 and G10.
* `docs/README.md`: this PLAN becomes `complete` when every condition of
  the Goal holds.

### Phase 7: live drift (added 2026-10-04)

Added by the owner on 2026-10-04 ("added as extra phase, proceed"), after
phase 1's live suite found two failures that also fail on `6be2d80` (the
Execution record, phase 1, V3.3). The MADR's amendment "two live failures
added as phase 7" records them as L1 and L2. L3, from phase 2's live suite,
was added on 2026-10-05 (the MADR's amendment "L3 added to phase 7").

#### 7.1 OpenCode's metadata route test follows the live metadata (L1)

* **Red:** `TestLive_OpencodeRoutesFromMetadata`
  (`llmprovider/live_opencode_test.go:175`) fails today. It reports
  `request paths = [/zen/go/v1/messages], want one /chat/completions`,
  because the live document now gives Go's `qwen3.8-max` the npm
  `@ai-sdk/anthropic`.
* **Change:** the test reads the live models.opencode.ai document. It
  asserts that the request path is the route that `qwen3.8-max`'s npm on
  `opencode-go` maps to through `routeForNPM`'s rule, and logs the npm it
  found. The rule's mapping is `@ai-sdk/openai` → `/responses`,
  `@ai-sdk/anthropic` → `/messages`, `@ai-sdk/google` → the Google route,
  and anything else, or none, → `/chat/completions`. The provider's routing
  is unchanged: it already follows the metadata.
* **Done when:**
  * the test passes live;
  * a planted wrong expectation, on a scratch copy, fails it;
  * the comment says what the test pins now.

#### 7.2 Grok's instructions: measured first (L2)

* **7.2a, probe.** A scratch live test runs on a scratch copy, with
  `XAI_API_KEY` checked for presence only. It sends the test's request to
  `grok-4.6` and to the newest Grok model in the live listing, three times
  each, in two forms:
  * (a) today's, the instructions as a leading system message in `input`;
  * (b) the instructions in the top-level `instructions` field.

  It records only whether each reply contains `OMEGA`.
* **7.2b, if (b) is obeyed and (a) is not:**
  * **Red:** `TestBody_InstructionsField` in
    `llmprovider/providers/grok/request_test.go`. A request's
    `Instructions` go in the top-level `instructions` field, not in
    `input`.
  * **Change:** `grok.go`'s `body` sends `instructions`.
  * The `grok` goldens with instructions change, each listed.
  * The live test passes.
* **If both forms are obeyed, or the results disagree between runs,** stop
  and prompt with the counts.
* **If neither form is obeyed,** the service ignores the instruction in
  both forms. Stop and prompt with the counts: changing the test's prompt
  to pass would loosen it.
* **Run 2026-10-05:** both forms obeyed a neutral instruction, and neither
  the override one. Resolved by the deviation "step 7.2a finds both forms
  obeyed": 7.2b is not done, and the live test is a pair.

#### 7.3 OpenCode's 403 is a refusal, not a bad key (L3)

* **Red,** on a scratch copy of `HEAD`:
  * `TestClassifyHTTPError_OpencodeForbidden` in
    `llmprovider/api_error_kind_test.go`. For `opencode-go/responses` and
    `opencode-zen/chat_completions`, a 403 with each of these bodies is
    `ErrNotPermitted`, not `ErrAuthFailure`, and terminal:
    * `{"type":"error","error":{"type":"error","message":"Your organization does not have access to this model"}}`;
    * a plain-text body;
    * an empty body.

    Today each is `ErrAuthFailure`.
  * Guards, which pass today and must still pass:
    * an OpenCode 401 `{"error":{"message":"Invalid API key."}}` is
      `ErrAuthFailure`;
    * an OpenCode 403 `RegionError` is `ErrNotPermitted`;
    * `TestClassifyHTTPStatus`'s 403 row, for a provider that is not
      OpenCode, stays `ErrAuthFailure`.
* **Change:** in `classifyAPIError`'s `ErrNotPermitted` case
  (`api_error.go:301-306`), the row `service == serviceOpencode && status ==
  http.StatusForbidden`, beside Kilo's.
* **Records:** append "Amendment 2026-10-05: OpenCode's 403 (0021 L3)" to
  `0012-MADR-conform-providers-to-reference-clients.md`, adding the row to
  §1.1's table.
* **Done when** the red test passes, the guards pass, and the gate is clean.

#### Phase 7 live check

7.1 and 7.2 are live checks themselves. For 7.3, the five tests L3 broke
(`TestLive_OpencodeResponses`, `TestLive_OpencodeResponsesStoreFalse`,
`TestLive_OpencodeRouteStillEnforced`, and the go-responses rows of
`TestLive_OpencodeToolRoundTrip` and `TestLive_OpencodeToolChoices`) run
once more and pass. The account's access has returned, so a live 403 cannot
be forced; the unit rows pin the classification.

## Verification

**V1. Red and green.** Each finding in the Execution record shows its
FAIL line and then its PASS line. A gate change shows its planted breach.

**V2. The gate** (G1–G11) is clean at the end of each phase, with
`api-check` additive only against `v1.1.0`.

**V3. Live checks,** with the owner's keys and sessions. Each key is
checked for presence only, and no token or content is printed.

* **V3.1 (phase 1):** a ChatGPT-session stream through the new path
  (`LLMPROVIDER_LIVE_CHATGPT=1`). The prompt asks for long reasoning, at
  high effort. The duration and finish are recorded. If it finishes under
  330 s, the record says it did not exercise the old limit; a run past
  330 s cannot be forced.
* **V3.2 (phase 2):**
  * OpenAI with `WithStore(false)` (`OPENAI_API_KEY`): a reasoning text
    turn returns encrypted reasoning, and a replay is accepted. As a
    control, tampered content is refused;
  * the same on an OpenCode responses-route model (`OPENCODE_API_KEY`);
  * the step 2.8a probe (`KILO_API_KEY`).
* **V3.3 (phases 1 and 2):** the existing live suite,
  `go test -tags live_gateways ./llmprovider/... -run Live`, with the keys
  present. The pass, skip and fail counts are recorded.
* **V3.4 (phase 4):** the optional live listing smoke test.
* **V3.5 (phase 5):** masked entry on the owner's Windows machine, in
  Windows Terminal and in the legacy console. The owner runs a scratch
  program, not committed, that calls `TextPrompter.Secret`, and checks a
  paste, Backspace, Ctrl-U, a lone ESC and Ctrl-C. The results are
  recorded.

**V4. Benchmarks.** The before and after numbers of steps 1.3 and 5.1
are recorded, and meet their targets.

**V5. Docs.** Every changed doc passes G8 and G10.

## Rollout and Rollback

* **Rollout.**
  * Six phase changes on `main`, staged by the agent and committed by the
    owner.
  * The new APIs are additive (R48): `ReasoningItem.Format`,
    `TokenInvalidator`, the harness fields. Behaviour changes in timeouts,
    retries, `Search` and `Rank` are documented. The release that carries
    them is `v1.2.0`, tagged when the owner asks.
* **Rollback.**
  * Each phase reverts on its own, newest first.
  * Phase 2 changes goldens and adds golden cases; a revert takes both
    back.
  * Phase 1 changes `DefaultClient`. Reverting it restores the 330 s total
    and drops the auth request bounds with it.
  * Phase 4's route snapshot move reverts with `git mv` back.

## Execution record

*Approved 2026-10-04 by the owner: "proceed".*

### Deviation 2026-10-04: step 1.3's benchmark target

* **Found:** after the change, `BenchmarkReadStream_4MiB`, the fixture
  repeated to 4 MiB, went from 41,620 to 7,120 allocs/op. That is a sixth,
  not the tenth the step asked for.
  * The fixture is mostly events the reader must decode:
    `response.created` carries the whole response object, and the input
    repeats it about 450 times, where a real stream sends it once.
  * The MADR's W2 evidence measured a long answer, which is mostly text
    deltas.
  * The cause is the PLAN's choice of input. The MADR's facts stand.
* **Resolution, chosen by the owner** ("Long-answer stream"):
  * the tenth applies to a new `BenchmarkReadStream_Deltas4MiB`, a 4 MiB
    stream of text deltas, then the done and completed events;
  * the fixture benchmark stays, and its numbers are recorded as
    information.
* **Files added to the phase:** none; both benchmarks are in
  `llmprovider/internal/wire/responses/stream_limit_test.go`.

### Phase 1: replies and retries (2026-10-04)

**Red runs.** The red tests ran on a scratch copy of `HEAD` (`6be2d80`) with
the new tests added. Where a test needs a symbol that `HEAD` lacks
(`transport.IsAfterReply`, `ErrIdleTimeout`, `wire.SetIdleTimeout`,
`dialer`), the scratch copy ran the same scenario asserted by kind, count and
message. The FAIL lines:

* **1.1, W6 and D1**, `TestGenerate_CutBodyIsUnavailable`:
  `a cut body: llmprovider: incomplete response: together: unexpected EOF;
  want ErrProviderUnavailable after the reply` and `a cut body was sent 1
  time(s), want 2: retried once`.
* **1.1, W6**, `TestGenerate_OverLimitIsIncomplete`: `a reply over the
  limit: llmprovider: incomplete response: together: unexpected EOF; want
  ErrIncomplete, over 16 MiB`.
* **1.1, W11**, `TestClaude_NilToolSchemaIsEmptyObject`: `input_schema =
  <nil>, want map[...]{"properties":map[...]{}, "type":"object"}`.
* **1.2, D2**:
  * `TestGenerate_IdleBodyIsUnavailable`: `an idle body: together: read
    reply: context deadline exceeded; want the idle limit's error`, after
    2.00 s;
  * `TestDefaultClient_Timeouts`: `ResponseHeaderTimeout/Timeout =
    5m0s/5m30s, want 5m0s/0s`;
  * `TestResolveOptions_DefaultTimeout`: `default timeout: got 5m30s want
    none`.
* **1.3, W2**:
  * `TestReadStream_LongStreamCompletes`: `a 21198605-byte stream: ... p:
    decode stream event: unexpected end of JSON input; want it read
    whole`;
  * `TestReadStream_OverlongEventIsIncomplete`: `an event over 16 MiB: p:
    decode stream event: unexpected end of JSON input; want ErrIncomplete`.
* **1.4**:
  * `TestWithRetry_AfterReplyOnce`: `calls = 5, err = llmprovider: stub
    failed after 5 attempts: ...; want 2 calls: retried once`;
  * `TestWithRetry_WaitPastDeadlineReturnsAtOnce`: `err = context deadline
    exceeded, calls = 1; want the 429 with its RetryAfter`, after 1.00 s;
  * `TestWithRetry_CancelledWaitKeepsError`: `err = context canceled; want
    context.Canceled and the 503`;
  * `TestRetryPolicy_ServerWaitJitter`: `every wait was the same: no
    jitter`;
  * `TestRetryPolicy_BackoffEqualJitterAfterCap`: `every wait at the cap
    was the same`;
  * `TestClassifyHTTPError_ShouldRetryTrue`: `err = llmprovider: invalid
    request: openai HTTP 400; want a retryable ErrInvalidRequest`;
  * `TestClassifyHTTPError_Conflict`: `openai 409: ... retryable false;
    want true`, and the same for `claude`;
  * `TestWithRetry_KindlessRetriedOnlyFromTheNetwork/a_bare_network_read_failure`:
    `2 calls, want 1`;
  * `TestRetryPolicy_WaitBackoffAndCap`: `wait(1) = 107.229215ms, true;
    want between 50ms and 100ms`.

**Planted breaches,** on a scratch copy of the tree:

* `TestAuthRequests_Bounded`, with `doBounded`'s timeout planted out: every
  subtest failed, for example `a stalled body took 5.001575541s, want each
  request ended at 100 ms`.
* `TestReadStream_EndWithoutCompletedIsAfterReply`, with the early end's
  mark planted out: `an early end: llmprovider: provider unavailable: p:
  stream ended before response.completed; want ErrProviderUnavailable after
  the reply`.

**Guards that pass on `HEAD` too,** recorded as guards:

* `TestWithRetry_TooLargeNotRetried`: `HEAD` already did not retry
  `ErrIncomplete`.
* `TestReadStream_EndWithoutCompletedIsAfterReply`'s kind check: the mark
  itself is new, and its planted breach is above.

**PASS:** every test above passes on the tree, and so does the whole suite.

**What was built:**

* **1.1:**
  * `llmprovider/internal/transport/reply.go`: `StreamIdleTimeout`,
    `ErrIdleTimeout`, `ErrReplyTooLarge`, `AfterReply`, `IsAfterReply`,
    `ReplyReader`;
  * `llmprovider/internal/wire/post.go`: `Call`, `Post`, and
    `SetIdleTimeout` for tests;
  * `DecodeError(provider, err, readErr)`;
  * `Reauth(ctx, provider, src, send(token))`;
  * `llmprovider/internal/wire/tools.go`: `ToolSchema`, `ResponsesTools`,
    `AddResponsesTools`, `MessagesTools`.

  The nine `generateOnce` functions are each one `wire.Post` call now, and
  `listChatGPT` takes its token from `Reauth`.
* **1.2:** `DefaultClient` has no total timeout, a 30 s dialer and
  `ForceAttemptHTTP2`. `llmprovider/auth/request_bound.go` adds
  `doBounded`, which bounds each auth request at `authRequestTimeout` (30 s)
  until its body is closed. These are the nine `.Do(` call sites the
  grep listed, all moved to `doBounded`:
  * `kilo_device.go:130, 181`;
  * `oauth_device.go:422, 436`;
  * `oauth_idtoken.go:208`;
  * `oauth_loopback.go:309, 622`;
  * `oauth_revoke.go:65`;
  * `oauth_session.go:356`.
* **1.3:** `ReadStream` reads line by line with a 16 MiB limit per event.
  It skips every event it does not use before decoding it, not only
  deltas: a first `"type"` that starts `response.` and is not handled. It
  copies a line only when the line is longer than its 64 KiB buffer.
* **1.4:** the once-after-a-reply rule, the deadline rule, both jitters,
  `x-should-retry: true`, and a retryable 409 on `openai` and `claude`.

**Implementation notes, within the steps:**

* `DecodeError` checks the read failure before an existing kind, not
  after as step 1.1 lists. A read failure is always the cause of the decode
  error that follows it, and the stream reader's own read error is kinded.
  The outcome is what the step asks for.
* The Responses tool helper is `AddResponsesTools`, adding the tools and the
  choice together, where the step named `ResponsesTools` and
  `ResponsesToolChoice`.
* A failure to fetch the ChatGPT listing's token now reads
  `llmprovider: openai: acquire token: …`, where it read `model listing:
  acquire token: …`. No test or caller matched the old text.
* Grok's five Responses tool constants went with its copy of the list.

**Existing assertions changed to the decided behaviour** (D1, D2, T5, T7):

* `TestResolveOptions_DefaultTimeout` (`llmprovider/options_test.go`):
  330 s total became "no total, 300 s first byte, a bounded dialer".
* `TestDefaultClient_Timeouts`: the same, and the dialer.
* `TestWithRetry_KindlessRetriedOnlyFromTheNetwork`:
  * the `net.OpError` row now expects one call;
  * a `*url.Error` row expects two.
* `TestRetryPolicy_WaitBackoffAndCap`: a wait is between half and all of
  its backoff, and at the cap between half the cap and the cap.
* `TestDecodeError` and `TestReauth`: the new signatures, and the read
  failure and over-limit cases.

`TestWithRetry_HonoursRetryAfter` is unchanged. A wait still lasts at least
what the service asked.

**Goldens:** none changed. G9 passed three times in a row, and no golden
case has a nil schema.

**Benchmarks** (step 1.3, with the deviation above), `-benchmem -count=5`
on this host:

| Benchmark | Before (`HEAD`) | After |
| :--- | :--- | :--- |
| `BenchmarkReadStream_Deltas4MiB` | 18.8 ms, 20.2 MB, 83,923 allocs/op | 1.35 ms, 66.8 KB, 13 allocs/op |
| `BenchmarkReadStream_4MiB` (information) | 18.9 ms, 15.0 MB, 41,621 allocs/op | 7.3 ms, 844 KB, 7,120 allocs/op |

The target, a tenth of the allocations on the long-answer stream, is met.

**Records and docs:**

* `0012-MADR-conform-providers-to-reference-clients.md` gains "Amendment
  2026-10-04: generation timeouts (0021 D2)", and its date is now
  2026-10-04;
* `0020-MADR-remediate-v1-debugging-pass-findings.md` gains "Amendment
  2026-10-04: a failure after a 200 is retried once (0021 D1)";
* `README.md` (Errors, Transport) and `docs/architecture.md` (Transport,
  `WithRetry`, and the rows for `internal/wire` and `internal/transport`)
  describe the new behaviour.

**Gate,** all exit 0:

* G1 `make pre-add-check`, G2 vet for darwin, linux and windows, G3 the
  live-tagged vet;
* G4 `go test -race -count=1 -cover ./...`: 26 packages ok;
* G5 `go mod tidy -diff`, G6 `make lint`: 0 issues on the host and for
  Windows;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: 27 packages, 0 problems;
  * coverage-check: 27 packages, 0 problems. `internal/transport` is at
    90.3% and `internal/wire` at 96.5%, after `reply_test.go`,
    `post_test.go` and `tools_test.go` were added;
  * api-check: "against v1.1.0, 0 incompatible change(s)";
  * generate-check: 1 file, 0 problems;
* G8 markdownlint: 0 issues;
* G9 goldens stable;
* G10 links: 0 problems;
* G11 identifier scan: 0 hits in 49 files.

**Live checks:**

* **V3.1,** on a scratch copy of the tree, with the owner's ChatGPT session
  borrowed read-only from the Codex CLI and no token printed:
  `gpt-6-astra`, a long proof prompt at high effort.
  * The result was finish `stop` after 58 s, 2 items (1 reasoning), an
    answer of 6,979 characters, and 2,677 output tokens (340 reasoning).
  * The stream went through `wire.Post` with the idle limit, and passed.
  * It finished well under 330 s, so it did not exercise the old total
    limit. A run past 330 s cannot be forced.
* **V3.3,** the existing live suite on the tree, with all eight keys
  present (checked for presence only):

  * The suite: 111 passed, 19 skipped (their opt-in variables unset), 2
    failed, in `llmprovider` only. Every other package was ok.
  * **Both failures predate this change.** On a scratch copy of `HEAD`
    (`6be2d80`), the same two tests failed in the same way, twice each.
  * Neither failure touches what phase 1 changed. They are outside this
    PLAN and wait for the owner's decision (Rule 5):
    * **`TestLive_OpencodeRoutesFromMetadata`:** `request paths =
      [/zen/go/v1/messages], want one /chat/completions`.
      * The test assumes Go's `qwen3.8-max` has no `provider.npm`.
      * The live models.opencode.ai document of 2026-10-04 lists it as
        `@ai-sdk/anthropic`.
      * So `/messages` is what the metadata now says, and the provider
        follows it. The test's assumption is stale.
    * **`TestLive_GrokInstructions`:** `GenerateText = "Hello!", <nil>;
      want the instructions obeyed`. `grok-4.6` did not follow a leading
      system message that said to reply only "OMEGA".
      * Whether xAI now wants the top-level `instructions` field, or the
        model changed, is not measured.

### Deviation 2026-10-04: steps 2.5 and 2.6 change the goldens' decoded results

* **Found:**
  * A golden file records the decoded result as well as the request
    (`internal/wirecase/wirecase.go`, `Run`).
  * Step 2.5's red run failed G9 on `opencode-zen-google/items.json`:
    `result.output[2].item.CallID: want "get_weather", got
    "get_weather#0"`. That is W10's decided fix, but step 2.5 had said
    "the goldens stay unchanged".
  * Step 2.6's `Format` field will likewise appear in the decoded results
    of the goldens that hold a `ReasoningItem`, about 19 files across all
    nine provider packages.
  * No request changes.
* **Resolution, chosen by the owner** ("Regenerate and list them"):
  * regenerate those goldens with `-update`;
  * list each golden and the changed field in phase 2's record.
* **Files added to the phase:** the regenerated golden files, listed in
  the record.

### Phase 2: answers (2026-10-04)

**Red runs.** The red tests ran on a scratch copy of `HEAD` (`3a584df`,
phase 1's commit) with the new tests added. The FAIL lines:

* **2.1, W1:**
  * `TestDecode_ToolInputKeepsPrecision`: `Arguments =
    {"id":12345678901234567000,"z":1}, want the input exactly`;
  * `TestDecode_ToolInputMissingIsEmptyObject`: `Arguments = "", want {}`;
  * `TestDecode_FunctionArgsKeepPrecision`: `Arguments:{"id":12345678901234567000,"z":1}`;
  * `TestDecode_ArgumentsAsObject`: `json: cannot unmarshal object into Go
    struct field .choices.0.message.tool_calls.0.function.arguments of type
    string`;
  * `TestToolArguments_PassesValidJSONThrough`: `ToolArguments =
    {"id":12345678901234567000,"z":1}; want {"z":1,"id":12345678901234567890}`.
* **2.2, W3:**
  * `TestFinish_Messages`, four failures:
    * `model_context_window_exceeded with a call: <nil>; want ErrIncomplete,
      reason length`;
    * `an unknown stop_reason: ... ; want it kept`;
    * `end_turn with a call: ... stop ...; want tool_calls`;
    * `an empty refusal: llmprovider: incomplete response: messages: the
      answer has no content; want ErrIncomplete, reason content_filter`;
  * `TestFinish_GenerateContent`: `OTHER: ... ; want it kept`, and `SAFETY
    with no parts: ...; want ErrIncomplete, reason content_filter`;
  * `TestFinish_Chat`: `stop with a call: ... stop ...; want tool_calls`,
    `role Assistant: [{Role:Assistant Text:hi}]; want RoleAssistant`, and
    `an empty content_filter answer: ... no usable content; want
    ErrIncomplete, reason content_filter`.
* **2.3, W4:**
  * `TestDecode_EmptyOutputIsIncomplete`: `Decode = &{... Output:[]
    FinishReason:stop ...}, <nil>; want ErrIncomplete`, and the same from
    `ReadStream`;
  * `TestDecode_RefusalIsKept`: `Output:[] FinishReason:stop`, from both.
* **2.4, W5,** `TestDecode_GatewayErrorIn200`:
  * `top level: ... the answer has no choices; want ErrProviderUnavailable
    with the message`;
  * `in the choice: ... the answer has no usable content`;
  * `finish_reason error with no envelope: ... no usable content`.
* **2.5, W10,** `TestDecode_GeminiCallIDs`:
  * `ids: calls = [{CallID:weather ...} {CallID:weather ...}], want ids
    [fc_1 fc_2]`;
  * `no ids: ... want ids [weather#0 weather#1]`;
  * `a result for weather#1: [...{"name":"weather#1"...}]; want it sent as
    weather`.
* **2.7, D4:**
  * `TestBody_StatelessAsksForEncryptedReasoning` (openai and grok):
    `store false: include = <nil>, want [reasoning.encrypted_content]`;
  * `TestResponsesBody_AsksForEncryptedReasoning`: `include = <nil>` for
    both requests, and `thinking: reasoning = map[effort:medium], want
    summary auto`.
* **2.8b, W9,** `TestDecode_ReasoningDetailsKept`, through a scratch
  variant without the `Format` check, since `HEAD` has no such field:
  `reasoning = [{Text:Check the weather. Signature: Encrypted:}]; want one
  item, the entry kept`.

**Planted breaches,** on a scratch copy of the tree, for the tests that
need `ReasoningItem.Format`, which `HEAD` lacks:

* `TestInput_SkipsForeignReasoning`, with `wire.Replays` planted to `true`:
  `Format "messages": input [map[encrypted_content:e ...]]; want sent false`.
* `TestMessages_SkipsForeignReasoning`, with the same plant: both foreign
  items were sent, as a `thinking` block and as `redacted_thinking`.
* `TestBody_ReplaysReasoningDetails`, with `ReplayReasoningDetails` planted
  out of `Body`: `assistant message = map[content: role:assistant
  tool_calls:[...]]; want the entry replayed`.

**PASS:** every test above passes on the tree, and so does the whole suite.

**What was built:**

* **2.1:** `wire.CompactArguments`. `ToolArguments` now returns `any`, with
  a JSON object passed through as `json.RawMessage`. The Messages and
  generateContent decoders read raw arguments, and Chat Completions'
  `chatArguments` accepts a string or an object. Gemini Interactions uses
  the shared helper.
* **2.2:**
  * `wire.Finish` and `wire.EmptyAnswer`;
  * Messages maps `model_context_window_exceeded` to `length`;
  * an unknown value is kept;
  * a call promotes `stop` to `tool_calls` on every wire;
  * Chat always emits `RoleAssistant`.
* **2.3:** the Responses wire's `completed`, shared by `Decode` and
  `ReadStream`. A `refusal` part is the answer's text, with
  `FinishContentFilter`.
* **2.4:** a top-level or choice-level `error`, or `finish_reason:
  "error"`, goes to `ClassifyStreamFailure`.
* **2.5:** `functionCall.id` is decoded, and `callID` / `callName` handle
  `name#index`. Nothing new is sent.
* **2.6:** `ReasoningItem.Format`, the `wire.Format*` constants, and
  `wire.Replays`. Every decoder sets it, and the Responses and Messages
  encoders replay opaque reasoning only from their own wire.
* **2.7:**
  * `include: ["reasoning.encrypted_content"]` with `WithStore(false)` on
    `openai` and `grok`, and on every OpenCode responses-route request;
  * `reasoning.summary: "auto"` on that route.
* **2.8:** see 2.8a and 2.8b below.

**Step 2.8a, the probe,** live on 2026-10-04, counts only:

* `anthropic/claude-sonnet-5.5`: a tool call, no reasoning text, and 0
  `reasoning_details`.
* `google/gemini-3.8-flash`: a tool call, reasoning text, and 1
  `reasoning_details` entry: `type=reasoning.text`,
  `format=google-gemini-v1`, keys `format, index, text, type`. That is
  OpenRouter's documented shape.

**Step 2.8b, so done:**

* each entry is a `ReasoningItem` with the entry itself, compacted, in
  `Encrypted`, and `Format: "chatcompletions"`;
* when entries are present, the plain `reasoning` string is not decoded as
  well, so the trace has one source;
* `Opts.ReplayReasoningDetails`, set by `kilo` only, puts them back on the
  assistant message they precede.

**Implementation notes, within the steps:**

* An empty answer's `APIError` names its wire in `Message`, such as
  `messages: the answer has no content`, and leaves `Provider` empty, since
  a decoder does not know the provider.
* A Chat Completions error in a 200 is classified under the label `chat
  completions`.
* The tests are in new files, `decode_0021_test.go` in each wire,
  `format_test.go`, `reasoning_details_test.go` and `stateless_test.go`,
  where the steps named each wire's `answer_test.go` or `request_test.go`.

**Existing assertions changed to the decided behaviour** (W1, W4, D5, W9):

* `TestToolArguments` compares the encoded JSON, since the result is
  `any`. Every row keeps its meaning.
* `messages.TestDecode`: a call with no input has `Arguments "{}"`.
* `TestDecode_Usage`, `TestReadStream_Usage`, and two `TestReadStream`
  subtests: their fixtures completed with no output, the empty success W4
  forbids. Each now carries one answer item.
* The decoded `ReasoningItem` expectations gain `Format`:
  * `TestThinking_SignatureIsReplayed` and `messages.TestDecode`;
  * `TestReasoning_EncryptedIsKeptAndReplayed` and `responses.TestDecode`;
  * `TestGeminiInteractions_Response`.
* `TestDecodeChatCompletions_ReasoningFieldNames`: the row
  "reasoning_details only ...: not decoded" becomes "decoded". W9
  reverses that rule after its live check.

**Goldens** (G-wire, with the deviation above):

* **Decoded results only**, 19 files, each gaining `"Format"` on its
  reasoning item (6 `chatcompletions`, 3 `generatecontent`, 3 `messages`,
  7 `responses`):
  * `claude/items`;
  * `gemini/items` and `gemini/continuation`;
  * `grok/items` and `grok/continuation`;
  * `huggingface/items`;
  * `kilo/items`;
  * `ollama/items`;
  * `openai/items`, `openai/continuation` and `chatgpt/items`;
  * `opencode-{go,zen}-{chat,messages,responses}/items` and
    `opencode-zen-google/items`;
  * `together/items`.

  `opencode-zen-google/items` also has CallID `get_weather` →
  `get_weather#0` (W10).
* **Requests, D4:** `opencode-zen-responses` and `opencode-go-responses`,
  every scenario except `listing` (10 files). Each gains `include`, and the
  two `thinking*` scenarios of each gain `reasoning.summary`.
* **New cases, D4:** `openai-stateless` and `grok-stateless`, 7 files each.
  Each differs from its base case only by `"store": false` and the
  `include`, where a request is sent.
* G9 passed three times in a row after the update.

**Docs:**

* `docs/architecture.md`: the `Item` bullet (arguments, `Format`, D4, W9),
  and a new "Answers" bullet;
* `README.md`: an empty answer is `ErrIncomplete` with its reason.

**Gate,** all exit 0:

* G1, G2 and G3;
* G4: 26 packages ok;
* G5, and G6 with 0 issues;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: "against v1.1.0, 0 incompatible change(s)". The new field
    is additive;
  * generate-check: 0 problems;
* G8: 0 issues;
* G9: stable;
* G10: 0 problems;
* G11: 0 hits.

**Live checks:**

* **V3.2, D4,** on a scratch copy of the tree, keys checked for presence
  only:
  * **OpenAI `gpt-6-luna` with `WithStore(false)`:**
    * the `include` was sent;
    * 1 encrypted reasoning item came back, with `Format: responses`;
    * tampered content was refused: `HTTP 400 invalid_encrypted_content`;
    * the replay was accepted, with finish `stop` and the right answer.
  * **OpenCode Go `gpt-5.6-luna` and `grok-4.6`:** the same on each: the
    `include` sent, 1 encrypted item, tampered content refused with HTTP
    400, the replay accepted with the right answer.
  * **`gpt-6-luna` on Go,** the PLAN's first pick, answered `HTTP 403 ...
    Your organization does not have access to this model`. It does so on
    `HEAD` too, 5 of 5 tries, though it answered in phase 1's run earlier
    the same day. So the account lost access; the `include` did not cause
    it.
  * **Zen** answered `HTTP 402 ... Insufficient account funds`, and was
    skipped.
* **V3.2, W9:** the probe above.
* **V3.3,** the existing live suite on the tree:

  * The suite: 101 passed, 19 skipped, 12 failed (subtests included), in
    `llmprovider` only.
  * Every failing test was then run twice on a scratch copy of `HEAD`
    (`3a584df`) and twice on the tree. Both behaved the same, so phase 2
    caused none of the failures:
    * **L1 and L2** (phase 7), as in phase 1:
      `TestLive_OpencodeRoutesFromMetadata` and `TestLive_GrokInstructions`.
    * **New since phase 1's run, and failing on `HEAD` too:** OpenCode Go
      now answers `HTTP 403 ... Your organization does not have access to
      this model` for `gpt-6-luna`. That is the first pick of 5 tests:
      * `TestLive_OpencodeResponses`;
      * `TestLive_OpencodeResponsesStoreFalse`;
      * `TestLive_OpencodeRouteStillEnforced`;
      * `TestLive_OpencodeToolRoundTrip/go-responses`;
      * `TestLive_OpencodeToolChoices/go-responses/{required,none}`.

      The same account still reaches `gpt-5.6-luna` and `grok-4.6` on the
      same route (V3.2). This is outside the PLAN, and waits for the
      owner's decision (Rule 5).
    * **Flaky, not reproduced:** `TestLive_GeminiReplaysRealCall`, which
      got `HTTP 400 ... Request contains an invalid argument` once, and
      `TestLive_OpencodeSystemMessage`, where the reply "Hello" did not
      follow the system instruction. Each passed 2 of 2 on `HEAD` and 2 of
      2 on the tree.

### Deviation 2026-10-04: step 3.2's clock field breaks `api-check`

* **Found:** G7's `api-check` failed with `incompatible:
  ./llmprovider/auth.OAuthSession: old is comparable, new is not` (1
  incompatible change against `v1.1.0`).
  * The cause is step 3.2's clock as written, the field `now func()
    time.Time`. A field of function type makes the struct incomparable.
  * The other new fields (`issued`, `nextRefresh`, `nextResave`) are
    `time.Time`, which is comparable.
  * Not pre-existing: it comes from the PLAN's wording of the step. The
    MADR does not name the field.
* **Resolution, chosen by the owner** ("Comparable clock field"):
  * the clock stays a per-session unexported field, `now sessionClock`,
    where `sessionClock` is `interface{ Now() time.Time }` and nil means
    `time.Now`;
  * `TestOAuthSession_RefreshesAgainAfterBackoff` sets it to a
    `stepClock` that the test moves by hand.
* **Files added to the phase:** none.

### Phase 3: credentials (2026-10-04)

**Red runs.** The red tests ran on a scratch copy of `HEAD` (`f8c36ec`,
phase 2's commit) with the new tests added. Three test files had their
tree-only tests cut for that run, because those need seams `HEAD` lacks;
those tests were seen to fail under planted breaches instead (below). The
FAIL lines:

* **3.1, T1:**
  * `TestReauth_LateRefusalKeepsFreshToken`: `refreshes = 8, want 1: a late
    401 discarded a fresh token`;
  * `TestClaude_ConcurrentRefusalsRunCommandOnce`: `the key command ran 5
    times, want 2: a late 401 discarded a fresh key`.
* **3.2, T2 and T3:**
  * `TestOAuthSession_FailedEarlyRefreshKeepsToken`: `call 1: Token = "",
    llmprovider: provider unavailable: oauth: refresh failed: 503 Service
    Unavailable: ; want the current token`;
  * `TestOAuthSession_ShortLivedTokenNotRefreshedEachCall`: `10 calls made
    10 refreshes, want 0`;
  * `TestOAuthSession_AuthFailureStillFails` passes on `HEAD`, as a guard
    should; see its plant below.
* **3.3, T4:**
  * `TestOAuthRefresh_TolerantDecodeKeepsRotation`, all three cases. `a
    string` and `unreadable`: `oauth: decode refresh response: json: cannot
    unmarshal string into Go struct field oauthRefreshResponse.expires_in of
    type int64 (refresh "rt-old")`; `a float`: `cannot unmarshal number
    3600.0 ...`;
  * `TestOAuthRefresh_LargeReplyIsBounded`: `Token = 2097152 bytes, <nil>;
    want an error for the oversized reply`.
* **3.4, T9 and T10:**
  * `TestFileTokenStore_SkewedHeartbeatKeepsLock`: `the waiter took over a
    lock its holder kept touching`;
  * `TestOAuthSession_PendingResaveDoesNotBlock`: `5 calls took 1.906s with
    5 lock attempts; want under 100 ms and at most 1`.
* **3.5, T11:**
  * `TestKiloDevice_TransientPollKeepsPolling`: `oauth: Kilo device poll
    failed: 502 Bad Gateway after 1 polls; want the session after 2`;
  * `TestKiloDevice_ExpiryWrapsLastTransientError`: `Wait = oauth: Kilo
    device poll failed: 502 Bad Gateway, want the expiry, naming the last
    502`;
  * `TestOpenAIDevice_DroppedConnectionKeepsPolling`: `oauth: request: Post
    ".../api/accounts/deviceauth/token": EOF after 1 polls`;
  * `TestGrokDevice_ServerErrorKeepsPolling`: `oauth: decode Grok device
    error: invalid character '<' looking for beginning of value after 1
    polls`.
* **3.6, T15:**
  * `TestMkdirAll_TightensExisting`: `dir mode = -rwxrwxrwx (<nil>), want
    0755`;
  * `TestMkdirAll_RefusesSymlink`: `MkdirAll accepted a symlink`.

**Planted breaches,** on a scratch copy of the tree:

* `TestCommandToken_InvalidateTokenIgnoresStale`, with `InvalidateToken`
  planted to clear on any token: `tokens "key-1", "key-2", "key-3" after 3
  runs; want key-1, key-2, key-2 after 2 runs`.
* `TestOAuthSession_InvalidateTokenIgnoresStale`, with the same plant:
  `after a stale refusal: Token() = "", oauth: no refresh token: ...; want
  the current token`.
* `TestOAuthSession_RefreshesAgainAfterBackoff`, with
  `oauthEarlyRefreshRetry` planted to 24 h: `31 s later: Token = "a-old",
  <nil> after 3 requests; want the current token after 6`. Run again after
  the deviation's clock change, with the same failure.
* `TestOAuthSession_AuthFailureStillFails`, with the `ErrAuthFailure`
  exclusion planted out: `Token = "a-old", <nil>; want no token and
  ErrAuthFailure`.
* `TestOAuthSession_RefreshAttemptBounded`, with the attempt timeout
  planted to a plain cancel: `oauth: read refresh response: context deadline
  exceeded after 5.005s and 1 attempts; want each attempt ended at 50 ms`.
* `TestFileTokenStore_HeartbeatSurvivesReadError`, with a read error
  planted to stop the heartbeat: `the heartbeat stopped touching the lock
  after one failed read`.
* `TestFileTokenStore_TakeoverRenameRetried`, with a takeover error planted
  to return: `LockRefresh = planted rename failure, want the takeover tried
  again`.
* `TestMkdirAll_RefusesForeignOwner`, with the owner check planted out: `MkdirAll
  accepted a directory the user does not own`, in both subtests.
* `TestStatOwner`, with `statOwner` planted to add 1 to the uid: it
  reported the uid plus one, not the user's.

**PASS:** every test above passes on the tree, and so does the whole
suite. The lock, re-save, heartbeat, takeover, bound, refresh, device and
invalidation tests passed three runs in a row.

**What was built:**

* **3.1:**
  * `llmprovider.TokenInvalidator`;
  * `(*CommandToken).InvalidateToken` and
    `(*auth.OAuthSession).InvalidateToken`;
  * `wire.Reauth` calls `InvalidateToken(token)` when the source has it,
    else `Invalidate()`.
* **3.2:**
  * the session clock (see the deviation);
  * `issued`, set by a refresh and by a login exchange;
  * `refreshMargin`, from `jwtTimes` (exp and iat; `jwtExpiry` now uses
    it) or from `Expiry − issued`;
  * `heldToken` and `nextRefresh`, with the constants
    `oauthEarlyRefreshFloor` (30 s), `oauthEarlyRefreshRetry` (10 s) and
    `oauthEarlyRefreshJitter` (20 s);
  * the failure logged at Warn by `logEarlyRefreshFailure`.
* **3.3:**
  * `oauthRefreshResponse.ExpiresIn` and `oauthTokenResponse.ExpiresIn` are
    `oauthSeconds`;
  * the body is read through `io.LimitReader(oauthResponseLimit+1)`, and a
    longer one is an error;
  * `decodeRefreshResponse` falls back to the two tokens.
* **3.4:**
  * `oauthRefreshAttemptTimeout` (15 s) wraps each attempt, which sends with
    `client.Do`, no longer `doBounded`;
  * `lockWait` is 40 s;
  * `LockRefresh` judges staleness by the mtime it has seen unchanged, on
    its own monotonic clock;
  * a takeover error goes back round the loop, and the deadline error names
    the last one;
  * a done context returns after the first failed attempt;
  * `holdLock` skips a tick on a read error;
  * the seams `lockRead` and `lockRename`;
  * `persistRotation` tries the lock with a cancelled context, at most
    every 30 s (`nextResave`, `oauthResaveInterval`).
* **3.5:** `devicePacer`, shared by the OpenAI, Grok and Kilo loops, with
  `transientStatus`, `deviceMaxBackoff` (60 s) and the sentinel
  `errDeviceCodeExpired`.
* **3.6:**
  * `ownerperm.MkdirAll` checks `Lstat`, the owner through the seam
    `ownerOf` (default `statOwner`), and clears group and other write;
  * the doc comments of `MkdirAll`, the package and `NewFileTokenStore`
    say so;
  * `0010-MADR-windows-stdio-oauth-tokenstore-ci.md` gained "Amendment
    2026-10-04: the token directory on Unix (0021 T15)".

**Implementation notes, within the steps:**

* A refresh that failed because the caller's own context ended does not
  keep the token or set `nextRefresh`, so a caller that gives up does not
  delay the next refresh.
* `nextResave` is set when a retried save runs. A save that fails straight
  after a refresh is still retried on the next call, as before;
  `TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer` relies on that.
* `tokenExpiry` takes `oauthSeconds` and reads it through
  `durationFromSeconds`, so NaN, infinite and non-positive values take the
  3600 s default.
* A failed takeover waits the usual pause before the next try, rather than
  looping at once.
* `ownerperm.MkdirAll` reports an `Lstat` failure as "not a directory",
  with the cause joined, which keeps the package at its 100% floor.
* The Claude test runs its own key command,
  `TestHelperClaudeKeyCommand`, since the `llmprovider` package's helper is
  in its internal tests.
* The two concurrency tests release their 401s one at a time, so `HEAD`'s
  counts are fixed: 8 refreshes and 5 command runs. The MADR's evidence, 4
  of each, came from free-running requests.
* `docs/guides/adding-a-provider.md` step 7 now names `InvalidateToken`,
  alongside the `docs/architecture.md` rows the step named.

**Existing tests changed to the decided behaviour** (T8, T9, T11):

* `TestAuthRequests_Bounded`: the refresh row is bounded by
  `oauthRefreshAttemptTimeout`, shortened to 100 ms with
  `authRequestTimeout`, since step 3.4 replaces step 1.2's bound for the
  refresh.
* `lockStore`'s `staleAfter` is 100 ms instead of 10 s, and
  `TestFileTokenStore_StaleLockHasOneTaker` gives the first waiter a 20 ms
  heartbeat. A waiter now watches a lock for `staleAfter` before taking it
  over, so the first change lets the takeover happen within `wait`. The
  second keeps the second waiter from judging the first's fresh lock
  stale. Both assertions are unchanged.
* `TestFileTokenStore_RefreshLock_StaleTakenOver` sets `staleAfter` to 100
  ms; with the default it would now wait 30 s.
* `TestKiloDevice_Outcomes`, the row "unexpected poll status": 500 becomes
  400, since a 5xx is now transient.

**Goldens:** none changed; nothing on the wire changed.

**Docs:**

* `docs/architecture.md`: the reply step, the `ownerperm` row, "After a
  401", device logins, refresh, the lock and `CommandToken`;
* `docs/guides/adding-a-provider.md`, step 7.

**Gate,** all exit 0, after the deviation's fix:

* G1, G2 and G3;
* G4: 26 packages ok; `auth` 87.2%, `ownerperm` 100.0%, `internal/wire`
  93.3%;
* G5, and G6 with 0 issues on the host and with `GOOS=windows`;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: 27 packages, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: "against v1.1.0, 0 incompatible change(s)". Before the fix
    it reported the one in the deviation above;
  * generate-check: 0 problems;
* G8: 0 issues in the guides, `architecture.md` and the READMEs; the two
  records show only MD004, their `*` markers, which the repository's
  config excludes for records;
* G9: stable;
* G10: 0 problems;
* G11: 0 hits in 28 files.

**Live checks:** the PLAN has none for phase 3.

### Deviation 2026-10-05: step 4.1 and three existing metadata tests

* **Found:** with step 4.1 built, three existing tests fail. None of the
  failures exists on `HEAD` (`6fbd2e6`).
  * `TestLoadModelMetadata_StaleOnRefreshFailure`
    (`catalog/metadata_request_test.go:45`): `requests = 1, want 2`. Past
    the TTL the stale document now returns at once and the refresh runs in
    the background, so the test counted before the refresh had finished.
  * `TestLoadModelMetadata_LateFailureKeepsNewerDocument`
    (`catalog/remediation_test.go:84`, 0020-MADR F15): the test hangs until
    the package's timeout. It needs two fetches of one URL at once, the
    first failing after the second cached a document. Under C2's one fetch
    per URL the second lookup joins the first, so F15's race cannot occur.
  * `TestChatReasoningEffort_LookupHasDeadline`
    (`providers/opencode/wire_shapes_test.go:356`, 0013-MADR A6):
    `metadata lookup had 10s left, want at most 5s`. The test reads the
    outgoing request's deadline, which is now the detached fetch's
    `metadataFetchTimeout` (10 s). Each lookup still waits at most 5 s.
* **Resolutions, chosen by the owner:**
  * the stale-refresh test waits for the background fetch before it counts
    (`waitMetadataFetches`); its assertion is unchanged. This needed no
    choice;
  * **F15** ("Restate as single-flight"): the test keeps its name and
    citation, and checks that a second lookup joins the held fetch, with 1
    request, and that a document already cached is kept when that fetch
    fails;
  * **A6** ("Keep 10 s; retarget test"): the fetch keeps the PLAN's 10 s;
    the opencode test checks that the request carries a deadline of at most
    `metadataFetchTimeout`. `TestLookupMetadata_CallerDeadlineNotCached`
    covers the lookup giving up at its own deadline while the fetch goes
    on.
* **MADR:** amended with "F15 and A6 under the shared fetch".
* **Files added to the phase:** `catalog/remediation_test.go`,
  `providers/opencode/wire_shapes_test.go`, and
  `catalog/model_metadata_test.go` (the wait helper).

### Deviation 2026-10-05: step 4.3 and the gpt-6 family

* **Found:** step 4.3's red test expects `gpt-6-sol` to be "still" usable.
  On `HEAD` (`6fbd2e6`) it is not: `openaiAllowPrefixes`
  (`catalog/models_catalog.go:170-173`) is `gpt-4`, `gpt-5`,
  `gpt-3.5-turbo`, `o1`, `o3`, `o4` and `chatgpt-4o`. So OpenAI's live
  listing drops every `gpt-6-*` id, and `rankOpenAIModel` has no `gpt-6`
  case. The PLAN's step was wrong about the starting state; the MADR's C4
  did not mention it.
* **Resolution, chosen by the owner** ("Allow gpt-6 too"):
  * `gpt-6` joins `openaiAllowPrefixes`;
  * `rankOpenAIModel` scores it as it scores `gpt-5` (+150);
  * the red test stands as written. The ranking snapshot, which covers only
    Kilo, Zen and Go, is not affected.
* **MADR:** amended with "the gpt-6 family on OpenAI's listing".
* **Files added to the phase:** none beyond step 4.3's.

### Deviation 2026-10-05: step 4.5's anchors on Claude

* **Found** while writing step 4.5, before any code: Claude's anchors do not
  rise with the generation. They are 305→10, 400→40 (`haiku-4`), 405→40,
  406→35 (`sonnet-4-6`), 408→35 (`opus-4-8`) and 500→40 (`sonnet-5`).
  "The highest anchor at or below its `ord`" would therefore give:
  * `claude-opus-4-1` (401): 100 + 40 = 140, above `claude-opus-4-8`'s 135
    (today 100);
  * `claude-sonnet-4-20250514` (400): 150 + 40 = 190, above
    `claude-sonnet-4-6`'s 185 (today 150);
  * `claude-opus-4-6` (406): 135, a tie with `claude-opus-4-8` (today
    100).

  Gemini's and Grok's anchors rise, so the rule is sound for them.
* **Resolution, chosen by the owner** ("Cap below newer anchors"): for all
  three providers, an id between anchors takes the highest anchor at or
  below its `ord`, capped at one less than the lowest anchor above it. So
  `claude-opus-4-1` is 134, `claude-sonnet-4-20250514` 184 and
  `claude-opus-4-6` 134. The step's required scores (`gemini-3.8-flash`
  111, `gemini-3.10-flash` 113, `grok-4.7` 71, `grok-5` 90,
  `claude-opus-5` 140) stand.
* **MADR:** amended with "C6's anchors are capped".
* **Files added to the phase:** none.

### Phase 4: catalog (2026-10-05)

**Red runs.** The red tests ran on a scratch copy of `HEAD` (`6fbd2e6`,
phase 3's commit) with the new tests added. Two test files were adapted for
that run: `metadata_request_test.go` without the stale-refresh test and the
background wait, which need `metadataNow` and `waitMetadataFetches`, and
`model_matcher_test.go` with the new score constants as literals. The FAIL
lines:

* **4.1, C1 and C2:**
  * `TestLookupMetadata_CallerDeadlineNotCached`: `the next lookup = map[],
    model metadata: Get ...: context deadline exceeded; want the document,
    not the first caller's deadline`;
  * `TestLookupMetadata_OneFetchForConcurrentLookups`: `20 concurrent
    lookups made 20 requests, want 1`.
* **4.2, C3:** `TestListing_PageLimit`, all seven listers: `a 9 MiB page:
  err = <nil>, want an error naming the 8 MiB limit` (Together's: `together:
  decode models: unexpected EOF`); `TestModelMetadata_DocumentLimit`: `a 33
  MiB document: err = <nil>`.
* **4.3, C4,** `TestIsUsableOpenAIChatModel_DeniesNonChatTokens`: the five
  ids `... is usable, want it denied`, and `gpt-6-sol is not usable, want it
  allowed` (the deviation above).
* **4.4, C5,** `TestRankNames_MatchTokens`: `opencode
  rank("gemini-3.1-pro") = 170, want -300`; `opencode rank("minimax-m3") =
  170, want 0`; the same two in `kilo`, 100 against -200 and 0.
* **4.5, C6,** `TestRankGenerations`:
  * `gemini-3.8-flash = 180, not above gemini-2.5-flash = 210`;
  * `gemini-3.9-flash = 180, not above gemini-3.1-flash = 220`;
  * `grok-4.7 = 40, not above grok-4.6 = 60`; `grok-5 = 0, not above
    grok-4.7 = 40`;
  * `claude-opus-5 = 100, not above claude-opus-4-8 = 135`;
  * and seven exact values, such as `rank("gemini-3.10-flash") = 220, want
    293` and `rank("claude-opus-4-1") = 100, want 134`.
* **4.6, D3:**
  * `TestRankRecommended_FillSkipsRequestFailures`: `ranked =
    [glm-5.3-flash muse-spark-1.3-contributor deepseek-v4-flash hy3
    deepseek-v4-pro kimi-k2.6]`;
  * `TestOpencodeGoFallback_AppliesItemsSevenAndEight`: `ranked =
    [glm-5.3-flash deepseek-v4-flash muse-spark-1.3-contributor kimi-k2.6]`;
  * `TestRankGroup_FamilyLeadingLetters`: `rankGroup("gpt-6-sol",
    "gpt-sol") = "gpt-sol", want gpt`, and the same for the other two;
  * `TestListModelCatalog_Snapshot20260926`: `opencode-zen profile 1`, the
    planned change.
* **4.7, C9:**
  * `TestMetadataCandidate_GuardsCostAndAge`: `negative: cost -0.5 is known,
    want unknown`, the same for `nan`, `inf` and `overflow`, and `a future
    release date gives age -61 days, want unknown`;
  * `TestKiloPriceRank_NaNIsUnknown`: `kiloPriceRank("NaN") = NaN, want
    math.MaxFloat64`, and `+Inf` for `"Infinity"`;
  * `TestRankOrders_StrictWeakOrder` passes on `HEAD`, as a guard: Go's
    `cmp.Compare` orders NaN consistently. See its plant below.
* **4.8, C11 and Z11:**
  * `TestSearch_AnnotationNotSearched`: `"fast" = [gpt-4.1 o4-mini
    gpt-4o-mini gpt-4.1-mini], want no match`;
  * `TestSearch_CompactQuery`: `"gpt41" = [... gpt-4.1 ... 1035 ...], want
    gpt-4.1 at 3500`;
  * `TestSearch_ShortQueryNoSubsequence`: `ids = [o3 o3-mini openai/o3-pro
    gpt-4o-2024-08-13]`;
  * `TestSearch_TiesKeepListingOrder`: `ids = [mid-flash zeta-flash
    alpha-flash]`;
  * `TestSearch_GlobIDBeforeLabel`: `ids = [vendor/mirror kilo-auto/small]`,
    `scores = [1 1], want [2 1]`;
  * `TestSearchModels_LabelSubstringTier`: `balanced speed = [... Score:3000],
    want no match: it is in the annotation`;
  * `wizard TestConfigureLLM_SearchNoMatchUsesTheQuery`: `unexpected
    Input("Search models (blank for recommended)")`.
* **4.9, C12:**
  * `TestRank_ProviderCaseAndTogether`: `Rank("Gemini") = 0, Rank("gemini")
    = 280`, and `Together ranks its first static id 0, not above its second,
    0`;
  * `TestStaticIDs_Labelled`: `grok: static id "grok-4.5" has no label`;
  * `TestLabels_NoOrphans`: six orphans, `claude-3-5-haiku-latest`, the four
    `kilo-auto/*` tiers and `openai/gpt-oss-20b`.
* **4.10, C13:**
  * `TestCatalogFrom_EmptyCurationKeepsLiveListing`: the static six, not the
    live listing;
  * `TestKiloAuto_ExcludedButChoosable/tiers_only`: `Usable has tiers [],
    want the listing's` (the full listing passes on `HEAD`);
  * `wizard TestConfigure_EmptyRecommendedOpensSearch`: `Model =
    "deepseek/deepseek-v4.1-flash", want kilo-auto/small`, with the notice
    `live model listing for Kilo Gateway is unavailable`.

**Planted breaches,** on a scratch copy of the tree:

* `TestLookupMetadata_StaleServedWhileRefreshing`, with the stale return
  planted out: `10 lookups past the TTL took 302ms; want the stale document
  at once`.
* `TestLoadModelMetadata_LateFailureKeepsNewerDocument` (restated), with
  single-flight planted out: `requests = 3, want 2`. Under that plant the
  test ran 300 s before it failed; on the tree it takes milliseconds.
* `TestChatReasoningEffort_LookupHasDeadline` (retargeted), with the fetch's
  timeout planted out: `metadata fetch ran without a deadline`.
* `TestRankOrders_StrictWeakOrder`, with the blend comparison planted to
  `-1`: `utility: future is not equal to itself`, and the antisymmetry
  failures.
* `TestRouteHeuristic_AgreesWithEveryRow`, the new gate (4.11), with one
  disagreeing row planted in `routes_snapshot.json`: `opencode gpt-planted
  (npm "@ai-sdk/anthropic"): heuristic route "responses", want "messages"`.
* `TestBuildRouteTable_RefusesABadSnapshot`, with the unknown-section panic
  planted to `continue`: `unknown_section: buildRouteTable accepted it`.

**PASS:** every test above passes on the tree, and so does the whole suite.
The metadata, search and snapshot tests passed three runs in a row under
`-race`.

**What was built:**

* **4.1:** `metadataFetch` and `modelMetadataFetches`, the clock
  `metadataNow`, `metadataFetchTimeout` (10 s), the counter
  `metadataFetching`, and `startMetadataFetch`. The `context.Canceled`
  special case is gone.
* **4.2:** `decodeLimited`, `listingPageLimit` (8 MiB, in place of
  `togetherListingLimit`) at the seven list decodes, and `metadataLimit`
  (32 MiB).
* **4.3:** `hasTokens` and `openaiDenyTokens`; `gpt-6` in
  `openaiAllowPrefixes` and in `rankOpenAIModel` (deviation).
* **4.4:** the OpenCode and Kilo name switches use `hasTokens`.
* **4.5:**
  * `parseGeneration`, `versionNumber`, `hasRun`, `generationAnchor` and
    `generationScore`;
  * the anchor tables `geminiAnchors`, `claudeAnchors` and `grokAnchors`,
    with the cap (deviation);
  * `geminiShutDown`.
* **4.6:**
  * `excludedAtRequest`, used by `eligible`, the fill and Go's fallback
    curation;
  * `rankGroup` with `familyLetters`;
  * the snapshot golden's Zen capable six;
  * "Amendment 2026-10-05: the fill and the diversity group (0021 D3)" in
    `0009-MADR-use-case-aware-default-model-ranking.md`;
  * README's Ranking bullet.
* **4.7:** `metadataCandidate`'s guards, and `kiloPriceRank` through
  `kiloPrice`.
* **4.8:**
  * `scoreCompactSubstring` (3500), `scoreGlobID` (2), `scoreGlobLabel` (1),
    `minSubsequenceQuery` (3);
  * `displayName`, `compactModelText` and `sortByScore`;
  * `Search`'s doc;
  * the wizard's `useQueryLabel` menu on a search with no match.
* **4.9:**
  * `Rank` lower-cases the provider and ranks Together by static position;
  * a label for `grok-4.5` ("previous flagship");
  * the six orphan labels removed;
  * `unlabelledStaticIDs`, with one reason for the 26 ids of the
    metadata-ranked fallback sixes.
* **4.10:**
  * `catalogFrom` keeps a live listing whose curation is empty;
  * the `Catalog` doc;
  * `configure.go` keeps a live catalog with a non-empty `Usable` and asks
    for a manual id only when `Usable` is empty too;
  * `selectModel` re-prompts with `noRecommendedNotice` on a blank query.
* **4.11:**
  * `git mv` of `testdata/opencode-routes.json` to `routes_snapshot.json`;
  * `//go:embed`;
  * `buildRouteTable` in `init`, in place of the hand table and its
    `//nolint:goconst`;
  * `TestOpencodeRouteTable_MatchesMetadataSnapshot` reads the embedded file.

**Implementation notes, within the steps:**

* `hasTokens` uses the matcher's existing `modelTokens`, which also splits
  on `:`, `~` and spaces.
* Gemini's `3.0 || 3.1 || gemini-3-` case is one anchor, at 3.0. With 3.0
  and 3.1 both at 40, the cap would have put a `gemini-3-` id at 39, where
  today's table gives it 40.
* `parseGeneration` reads a one-digit major and a one- or two-digit minor.
  A two-digit major read the test id `claude-sonnet-45` as generation 45
  (240).
* The label tiers, the substring one and the token one, read the display
  name only. With the token tier still on the whole label, `"fast"` would
  match OpenAI's annotations.
* The no-match menu defaults to Search again. The wizard still prints the
  no-match notice first.
* `metadataCandidate` compares a release date with its `now` argument, the
  ranking clock the age is computed from, rather than `metadataNow()`.
* A malformed or unknown-section `routes_snapshot.json` panics at `init`:
  the file is embedded, so that is a build defect.
* The 0009 amendment is dated 2026-10-05, the day it was written; the step
  named it 2026-10-04.

**Existing assertions changed to the decided behaviour:**

* With the deviations above: `TestLoadModelMetadata_StaleOnRefreshFailure`,
  `TestLoadModelMetadata_LateFailureKeepsNewerDocument` and
  `TestChatReasoningEffort_LookupHasDeadline`.
* `TestListModelCatalog_Snapshot20260926`: the Zen capable six is
  `claude-opus-5-5, gpt-6-sol, gpt-6-luna, grok-4.7, muse-spark-1.3,
  claude-fable-5-1`, where `gpt-6-astra` was fifth. No other six changed.
* `TestMetadataCandidate_Fields`: the group of family `glm-flash` is `glm`.
* The lock-step search tests:
  * `TestSearchModels_GlobKeepsInputOrder` checks `scoreGlobID`;
  * `TestSearchModels_SubstringOutranksSubsequence` no longer expects the
    subsequence `fireworks/llama-ash`;
  * `TestSearchModels_TieBreak` expects input order;
  * `TestSearchModels_LabelSubstringTier` labels a model of its own, since
    every curated display name restates its id.
* `wizard`: `TestConfigureLLM_SearchNoMatchReturnsToSearch` picks Search
  again on the new menu. `TestConfigureLLM_FallbackSearch` and
  `TestConfigureLLM_FallbackSearchLoops` expect `claude-sonnet-5` before
  `claude-opus-5`, their listing's order.

**Generations, 4.5's guard.** Of the 99 `gemini-`, `claude-` and `grok-`
ids in the catalog package's sources, tests and testdata, these score
differently on the tree (before → after). The existing `TestRankGemini…`,
`TestRankGrok…` and `TestRankClaude…` assertions are unchanged.

* Claude:
  * no hint matched before, and an anchor now applies: `claude-fable-4` 90 →
    124; `claude-fable-5` 90 → 130; `claude-fable-5-1` and `5.1` 90 → 141;
    `claude-opus-4-1`, `4-6`, `4-7` and the dotted forms 100 → 134;
    `claude-opus-5` 100 → 140; `claude-opus-5-5` and `5.5` 100 → 155;
    `claude-sonnet-4` and `claude-sonnet-4-20250514` 150 → 184;
  * a hint now matches by token: `claude-opus-4.8` 100 → 135,
    `claude-sonnet-4.6` 150 → 185.
* Gemini: `gemini-3.8-flash` 180 → 291, `gemini-3.9-flash` 180 → 292,
  `gemini-3.10-flash` 220 → 293.
* Grok: `grok-4.7` 40 → 71, `grok-5` 0 → 90, and the five `grok-4.20…` ids
  40 → 84.

**Goldens:** no wire golden changed.

**Docs:**

* `docs/architecture.md`: "A listing" (empty curation, the metadata cache
  and the read limits), the `catalog.Search` bullet, and the `opencode` row;
* `README.md`: the Ranking bullet (D3, the background refresh, empty
  recommendations) and the wizard's search bullet;
* `0009-MADR-use-case-aware-default-model-ranking.md`: the amendment.

**Gate,** all exit 0:

* G1, G2 and G3;
* G4: 26 packages ok; `catalog` 92.9%, `opencode` 99.1%, `wizard` 88.0%;
* G5, and G6 with 0 issues on the host and with `GOOS=windows`, after two
  findings in the new code (a third `"tts"` literal, and `len(token) == 0`)
  were fixed;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: 27 packages, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: "against v1.1.0, 0 incompatible change(s)";
  * generate-check: 0 problems;
* G8: 0 issues in the READMEs, `architecture.md` and the guides;
* G9: stable;
* G10: 0 problems;
* G11: 0 hits in 32 files.

**Phase 4 check, live,** on a scratch copy of the tree, keys checked for
presence only, counts only:

* Kilo: live, 290 usable, 6 recommended under each profile, no
  `kilo-auto/*` recommended;
* OpenCode Zen: live, 41 usable, 6 recommended under each profile;
* OpenCode Go: live, 33 usable, 6 recommended under each profile;
* `TestLive_ModelMetadataDocument` passes, so the live document is under
  the 32 MiB limit; `TestLive_GrokListingTextOnly` passes;
  `TestLive_TogetherListing` skipped (`LLMPROVIDER_LIVE_TOGETHER` unset).
* Observed: xAI lists `grok-4.20-0309-*` beside `grok-4.3` to `grok-4.7`.
  Read numerically, 4.20 scores 84, above 4.7's 71, though its date suggests
  it came before 4.3. That follows C6 as decided (minor versions are
  integers, as `gemini-3.10` needs). Grok's static-first curation still
  leads with `grok-4.6` and `grok-4.5`; the backfill now ends with
  `grok-4.7` where it ended with `grok-4.3`.

### Deviation 2026-10-05: step 5.2 and the Google key boundary

* **Found:** `TestRedact_PlantedSecrets`, the step's own property test,
  failed on 15 of its 1,000 seeded cases, every one a Google API key ending
  in `-`, such as case 44's. The rule `\bAIza[0-9A-Za-z_-]{35}\b` ends in
  `\b`, which needs a word character on one side, so a key whose last
  character is `-` or `_`, followed by a space or a quote, is not matched
  and leaks whole. It leaks on `HEAD` (`1ba08e2`) too; step 5.2's change list
  did not name the rule. Every other planted class was redacted in all 1,000
  cases.
* **Resolution, chosen by the owner** ("Fix it in 5.2"):
  * the rule becomes `\bAIza[0-9A-Za-z_-]{35}`, with no trailing `\b`;
  * `TestRedact_Coverage0021` gains a row for a key ending in `-`;
  * the property test stands as written.
* **MADR:** amended with "Z2 also covers the Google key boundary".
* **Files added to the phase:** none beyond step 5.2's.

### Phase 5: wizard and redaction (2026-10-05)

**Benchmarks, step 5.1** (V4), `-benchmem -count=5` on this host. Before:
the two new benchmarks on a scratch copy of `HEAD` (`1ba08e2`, phase 4's
commit). After: the tree at the end of the phase.

| Benchmark | Before (ns/op, B/op, allocs/op) | After | Speed-up |
| :--- | :--- | :--- | :--- |
| `BenchmarkRedact_Clean4KiB` | 1,615,141–1,650,466, 23–167 B, 0 | 61,117–61,751, 0 B, 0 | about 26× |
| `BenchmarkClassifyHTTPError_64KiBHTML` | 27,354,055–27,932,291, about 339 KB, 40–46 | 881,134–895,346, about 307 KB, 45 | about 31× |

Both meet the 10× target. Every redact test that existed before the phase
is unchanged and passes.

**Red runs.** The red tests ran on a scratch copy of `HEAD` (`1ba08e2`) with
the new tests added. The FAIL lines:

* **5.2, Z2,** `TestRedact_Coverage0021`:
  * `{"accessToken":"abcd1234efgh5678"}`, and the same for `refreshToken`,
    `idToken` and `sessionToken`: `"abcd1234efgh5678" is not redacted`;
  * the PEM block: `= "key [REDACTED]\nMIIEvQ..."`, its body kept;
  * `Cookie: a=secret1234; b=secret5678`: `= "Cookie: [REDACTED]
    b=secret5678"`;
  * `password="two words here"`: unchanged;
  * `signature=abcdef123456` and `"private_key":"..."`: unchanged;
  * the kept rows: `{"code":"[REDACTED]"}`, `'code': '[REDACTED]'` and
    `token: [REDACTED]`, each `a diagnostic must be kept`;
  * `grant_type=refresh_token&refresh_token=abc%2Fdef12345` passes on
    `HEAD`, and stays as a guard;
  * `TestRedact_PlantedSecrets`: `case 0: "vbARC1r8DKc2SVuLJchr6NZ9"
    survived`.
* **5.3, Z3:**
  * `TestClassify_StripsControlCharacters`: each of `Error()`, `Code` and
    `Message`, from `ClassifyHTTPError` and from `ClassifyStreamFailure`,
    `holds a control character`, and `Code is 300 bytes, want at most 128`;
  * `TestCallback_StripsControlCharacters`: `callbackError = oauth:
    authorization failed: access\x1b[2J_denied`, and `parseOAuthInput =
    "ab\x1b[2Jcd"`;
  * `TestTextPrompter_NotifyStripsControlCharacters`: `Notify wrote
    "warning: listing failed: \x1b]0;owned\a\x1b[2Jbad\n"`.
* **5.4, Z6:**
  * `TestReadMasked_Paste`: `a paste drew the entry 2000 times, want at
    most 2`, and `the redraw erases the line with \033[K`;
  * `TestReadMasked_TypedRedrawPads`: `output
    "\r\x1b[KKey: •\r\x1b[KKey: ••\r\x1b[KKey: •\r\n"`;
  * `TestReadMasked_Keys`: `a lone ESC: readMasked = "x"`, `Ctrl-U:
    readMasked = "abcdef"`, and `Ctrl-C: err = wizard: cancelled, want it to
    match context.Canceled`.
* **5.5, Z7,** `TestResolveBaseURL_Validates`: `resolveBaseURL =
  "api.example.com"`, `"https://u:p@host"`, `"https://host/?q=1"`,
  `"https://host/#f"` and `"http://remote.example"`, each taken as entered;
  and `0 confirmations asked; want ... 1 asked`. The loopback and blank
  rows pass on `HEAD`, as guards.
* **5.6, Z10,** `TestResolveTokenStdin_ChecksTheCredential`: the `xai-`
  key, a non-JWT and an expired JWT were each saved (`<nil> with 1 saves`),
  and the `sk-ant-` key was taken as an API key (`<nil> with 0 saves`). The
  current-JWT row passes on `HEAD`, as a guard.

**PASS:** every test above passes on the tree, and so does the whole suite.

**What was built:**

* **5.1:**
  * `apiErrorRedactLimit` (16 KiB) and `cutMessage`, in both classifiers;
  * in `internal/redact`, the table `passes`, each pass gated on its
    lower-case literal `anchors` by `containsAny`, over a copy lower-cased
    by `asciiLower` into a 4 KiB stack buffer.
* **5.2:**
  * `reKV` gains the camelCase token keys, `private[_-]?key` and
    `signature`;
  * a quoted value runs to its closing quote;
  * `reCookie` and `reCookiePair` redact each pair's value;
  * the PEM rule spans BEGIN to END, or to the end of the base64 body;
  * `reKVLong` and the bare `token` key keep values matching
    `reDiagnostic`;
  * the Google key rule loses its trailing `\b` (deviation above).
* **5.3:** `internal/redact/control.go`'s `StripControl`, applied in both
  classifiers, through `boundCode` (`apiErrorCodeLimit`, 128) for `Code`, to
  the callback's `error` and `code`, and to `TextPrompter.Notify`.
* **5.4:** in `readMasked`, a `redraw` that pads to the previous width; the
  redraw runs only when nothing is buffered; Ctrl-U clears, and Ctrl-C wraps
  `context.Canceled`. `skipEscape` ignores a lone ESC.
* **5.5:** `wizard/base_url.go`, with `validateBaseURL`, `errBaseURLRefused`
  and `loopbackHost`, called from `resolveBaseURL` for every remote
  descriptor.
* **5.6:** in `resolveTokenStdin`, for OpenAI: `foreignKeyPrefixes` and
  `foreignKeyVendor`, then `checkAccessToken`.

**Implementation notes, within the steps:**

* `StripControl` turns a newline or carriage return into a space, rather
  than removing it, so the words on either side stay apart. The step's rule
  ("no byte below 0x20 except `\t`") holds.
* The prefilter reads the text as it was before any pass: a replacement only
  removes text and adds `[REDACTED]`, so it never adds an anchor.
* The cookie value is redacted pair by pair, so `Cookie: a=[REDACTED];
  b=[REDACTED]` keeps the names; `cookie` left `reKV`'s keys.
* A pool for the lower-cased copy was tried first; `errcheck` rejects its
  type assertion in every form, so a 4 KiB stack buffer serves instead.
  Longer text allocates once.
* `checkAccessToken` checks the signature segment's alphabet only: the
  fixture `eyJhbGciOiJub25lIn0.e30.x` has a one-character signature, which
  is base64url but does not decode.
* `validateBaseURL` is called once, from `resolveBaseURL`. The Kilo path
  receives the URL that call returned, so it needs no second call.
* The confirmation for plain `http` to a remote host defaults to no.

**Existing assertions changed to the decided behaviour** (Z10):
`TestConfigureLLM_TokenStdinStillAcceptsChatGPTToken` pastes a JWT, built
by the new `testJWT`, in place of `pasted-chatgpt-access`, which the paste
check now refuses as not a JWT. Its assertions are unchanged.

**Goldens:** none changed.

**Docs:**

* `docs/architecture.md`: the `internal/redact` row, the Errors bullet, and
  the wizard's "What it accepts" and `TextPrompter` bullets;
* `README.md`: the wizard's hidden-entry keys, base URLs and pasted
  credentials.

**Gate,** all exit 0, after four lint findings in the new code were fixed
(the pool's type assertion, `\d` in `reDiagnostic`, and two misplaced
imports):

* G1, G2 and G3;
* G4: 26 packages ok; `internal/redact` 100.0%, `llmprovider` 97.9%,
  `auth` 87.2%, `wizard` 88.4%;
* G5, and G6 with 0 issues on the host and with `GOOS=windows`;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: 27 packages, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: "against v1.1.0, 0 incompatible change(s)";
  * generate-check: 0 problems;
* G8: 0 issues; G9: stable; G10: 0 problems; G11: 0 hits in 20 files.

**V3.5, masked entry on Windows: pending, the owner's.** A scratch program,
not committed, calls `TextPrompter.Secret` for the five checks: a paste of
about 2,000 characters, Backspace, Ctrl-U, a lone ESC, and Ctrl-C. It
prints only lengths and errors, and builds with `GOOS=windows`. The owner
runs it in Windows Terminal and in the legacy console; the results go here.

### L3 investigation (2026-10-05)

Read-only, before the owner's decision; scratch copies only.

* **Probe:** on OpenCode Go, `gpt-6-luna`, `gpt-5.6-luna` and `grok-4.6`
  each answered `Reply with only OK` with no error. The 403 of 2026-10-04 has
  cleared on OpenCode's side.
* **The five tests,** run twice on a scratch copy of the tree: all passed
  both times, every subtest included.
* **The code:** see the MADR's amendment "L3 added to phase 7". The picker
  cannot see a per-account entitlement, and an untyped OpenCode 403 is
  `ErrAuthFailure` today.
* **Owner's choice,** from three options: "Fix the classification", added as
  step 7.3.

### Deviation 2026-10-05: step 6.5's hook on OAuthSession

* **Found** while building step 6.5: the step names the hook `onJoin
  func()` on `OAuthSession` too. A func field makes `OAuthSession`
  incomparable, which `api-check` reports as an incompatible change, as the
  phase 3 deviation "step 3.2's clock field breaks `api-check`" found for
  the clock.
* **Resolution,** following the owner's phase 3 choice ("Comparable clock
  field") for the same case: on `OAuthSession` the hook is `onJoin
  joinHook`, where `joinHook` is `interface{ joined() }`. `CommandToken`,
  already incomparable through its `now func()`, takes `onJoin func()` as
  written. `api-check` reports 0 incompatible changes.
* **Not asked again:** the owner decided this case in phase 3; this entry
  records that the decision was applied.
* **Files added to the phase:** none.

### Deviation 2026-10-05: step 6.6's checks find two gaps

* **Found:** with the four checks set on all nine providers' harnesses:
  * **`StrictTools`,** on Claude, Hugging Face, Kilo, Ollama, OpenCode and
    Together: `a call reply finished ""; want "tool_calls"`.
    * The harnesses' `ToolCall` replies carry no finish field.
    * `wire.Finish` (`internal/wire/finish.go:14-23`) promotes only `"stop"`
      to `tool_calls`, so a call reply with no finish reason has
      `FinishReason ""`, though R7 has every answer carry one.
    * The same on `HEAD` (`267c204`).
  * **`Truncated`,** on OpenAI, Grok and Gemini: `Reason
    "max_output_tokens"` (Responses) and `"incomplete"` (Interactions,
    measured at `max_output_tokens`), where Messages, Chat Completions and
    generateContent give `"length"`.
    * `APIError.Reason` is documented as "the service's reason for a
      response it cut short, such as `max_output_tokens`".
    * So the step's "`Reason` `length`" does not hold for every wire as
      documented.
  * **ChatGPT's stream harness** failed `Garbled` and `Truncated` because of
    its fixtures:
    * a body with no events is a stream that ended early, which D1 retries
      once by design;
    * the test helper `stream` turned the cut reply into a completed one.

    The fixtures are corrected (a garbled event, and a
    `response.incomplete` event). This needed no choice.
* **Resolutions, chosen by the owner:**
  * **"Fix wire.Finish":** a reply that carries a call and has no finish
    reason finishes `tool_calls`, on every wire. The harness fixtures stay
    as they are. Any golden that changes is listed in the record;
  * **"Keep the contract":** `Reason` stays the service's own, with no
    behaviour change. A new optional `Harness` field, `TruncatedReason`,
    names the reason the harness's wire gives, and the check requires it
    exactly (`length` when the field is empty).
* **MADR:** amended with "the empty finish reason and the cut answer's
  reason".
* **Files added to the phase:** `llmprovider/internal/wire/finish.go` and
  its test, and any golden that changes.

### Deviation 2026-10-05: step 6.6 finds W14 in ReadStream

* **Found:** with the ChatGPT stream harness's fixtures corrected, its
  `Truncated` check failed with `R7 (Response invariants): Generate returned
  both a response and the error llmprovider: incomplete response:
  max_output_tokens`.
  * `ReadStream` (`internal/wire/responses/responses.go:287-289` and
    `:308-310`) returns `result, err` from a dispatch that failed. So a
    `response.incomplete`, `response.failed` or `error` event, or one that
    does not decode, gives the partial response with the error.
  * The non-stream `Decode` returns `nil, err`.
  * Present since `940fee0`, before `v1.1.0`; not from this PLAN.
* **Resolution, chosen by the owner** ("Fix it in phase 6"):
  * a failed dispatch returns `nil` with its error, at both call sites;
  * a red-first test, `TestReadStream_FailureReturnsNoResponse` in
    `internal/wire/responses`, covers each of the four failing events;
  * the ChatGPT harness keeps its `Truncated` check.
* **MADR:** amended with "W14 found by the conformance checks".
* **Files added to the phase:** `llmprovider/internal/wire/responses/
  responses.go` and the new test.

### Phase 6: tooling and tests (2026-10-05)

**6.1, dep-check (Z4).**
* **The breach,** on a scratch copy: `internal/redact/zz_evil_test.go`
  imports `example.com/evil`, and `go.mod` requires it, replaced by a local
  module so it runs offline.
  * `HEAD`'s `check_deps.py`: exit 0, "27 packages, 0 problem(s)" on both
    builds.
  * The new one: exit 1, with `go.mod requires example.com/evil` and
    `internal/redact depends on example.com/evil` for the host and Windows.
* **Built:**
  * `check_requires` reads `go mod edit -json` against `REQUIRES`;
  * `check` reads `go list` again with `-test`, and strips a test build's
    ` [pkg.test]` and `.test`.
  * A test build's `Deps` entries carry a space, so they are joined with
    commas rather than spaces.

**6.2, generate-check, parity-check and gate-selftest (Z9).**
* **Built:**
  * `check_generated.py` reads `-output X` and `-output=X` (`output_at`),
    and fails when it checks no file;
  * `check_parity_map.py` requires each cell to hold a span that resolves
    (`resolves`, `sdk_packages`) or to be one of `MARKERS`. Before the
    change, every one of the 409 cells passed the new rule (five are
    markers: one `none`, four `removed`), so nothing in the guide changed;
  * `scripts/test_gates.py`, and `make gate-selftest`.
* **The self-test,** 7 tests in about 50 s. Its breaches:
  * dep-check: the module only a test imports;
  * generate-check: a hand edit behind `-output=X`, and a directive without
    `-output`;
  * parity-check: a `TBD` cell;
  * coverage-check: wizard's floor raised to 99.9;
  * api-check: `const MaxListed` made a `var`.

  Each fails with exit 1 and its message, and every gate passes on a clean
  copy.
* **The self-test seen to fail:** on a scratch clone with `HEAD`'s
  `check_deps.py`, `check_generated.py` and `check_parity_map.py` put back,
  4 of its 7 tests fail, each `exit 0, want 1`: the dep, both generate and
  the parity breaches.

**6.3, CI, precheck, floors and nolintlint (Z5).**
* **`ci.yml`:**
  * `go test -shuffle=on`, and `-race -shuffle=on` on Linux;
  * `govulncheck` pinned at `v1.8.0`, the newest release on 2026-10-05
    (`go list -m -versions golang.org/x/vuln`);
  * `gate-selftest` with the other gates;
  * `timeout-minutes: 30`, the `concurrency` block, and the weekly
    `schedule`.
* **`go-precheck.sh`:** `go test -race`, then `go mod tidy -diff`.
* **Floors,** with the local and CI Linux measurements (CI run 37330520202,
  on `3f18dd7`, which the tree matches in code), equal on both:

  | Package | Measured | Floor before | Floor after |
  | :--- | :--- | :--- | :--- |
  | `llmprovider` | 97.9 | 89.2 | 95.9 |
  | `wizard` | 88.4 | 83.4 | 86.4 |
  | `llmprovider/internal/wirecase` | 91.5 | 80.0 | 89.5 |
  | `internal/redact`, `llmprovider/internal/ownerperm` | 100.0 | 100.0 | 100.0 |

  `test_gates.py` finds wizard's floor by pattern, not by its value.
* **`nolintlint`** (`require-specific`, `allow-unused: false`) reported 5
  directives, all removed:
  * `models_catalog.go`'s `//nolint:goconst` on `modelLabels`, as the step
    expected (now line 806);
  * four `//nolint:gosec` in `auth`'s tests: `oauth_login_verify_test.go:34`
    and `oauth_loopback_test.go:224, 305, 438`.

  The `:35` directive the step said to check is still used, and stays.
* **govulncheck seen to fail,** on a scratch copy: clean, "No
  vulnerabilities found", exit 0. With a package calling
  `golang.org/x/text/language.Parse` at `v0.3.5`: `GO-2021-0113 ... Your
  code is affected`, exit 1 (`go run` passes on the tool's failure as 1).
* **Pending:** the CI change is seen working on the owner's next push.

**6.4, the fake (Z8).**
* **Built:**
  * `newFake` and `allowLeftover` in `fake_prompter_test.go`;
  * 94 literals in 14 files converted by `p6_fake.py` (the step counted 90,
    before phases 4 and 5 added four), including the three wrapper
    prompters, which now embed `*fakePrompter`;
  * no test needed `allowLeftover`.
* **The check found 10 tests (12 subtests) with stale scripts,** each
  corrected:
  * `TestConfigureLLM_TokenStdinClassification`: the model-menu answer now
    only for the API-key rows; a ChatGPT session types its model, and the
    fixture row stops at the refusal;
  * `TestConfigureLLM_KeepsAnySavedSession`: the model-menu answer only for
    Kilo; OpenAI's kept session types its model;
  * `TestConfigureLLM_ChatGPTListingFailurePromptsForModel` and
    `TestConfigureLLM_KeepRefusesStubSession`: a sign-in answer that keeping
    the session skips, and the stub test's model answer, removed;
  * `TestConfigureLLM_LoginUsesTheCallersClient`: the model is typed, not
    picked from a menu;
  * `TestConfigureLLM_KiloProfileGetsTheEndpoint`: `blankSearches`, so the
    flow reaches the menu it scripts;
  * `TestConfigureLLM_LocalProviderSkipsKey` and
    `TestConfigureLLM_NoModelsAndNoneEnteredErrors`: the endpoint is a
    refused loopback port (`refusedURL`), not `localhost:11434`, so the
    endpoint's Confirm is asked on every host; on this one an Ollama
    answered, and the Confirm was left;
  * `TestConfigureLLM_KiloOrganizationDefaultsFromExisting` **was hollow:**
    its flow ended at the endpoint prompt, so the organization menu it
    checks was never shown. It now answers the endpoint, reaches the menu,
    requires it shown, and passes: the default is Acme.
* **Seen to fail:** an extra input planted in
  `TestConfigureLLM_SearchNoMatchUsesTheQuery`: `fakePrompter: 1 scripted
  inputs left unused`.

**6.5, the timing tests (Z12).**
* **Built:**
  * `CommandToken.onJoin func()`;
  * `OAuthSession.onJoin joinHook` (the deviation above);
  * both called as a waiter joins;
  * the two waiter tests wait on the join, not 50 ms;
  * the `CommandToken` tests call `t.Parallel()`: the six in
    `command_token_test.go` and the leader test.
  * `TestHelperKeyCommand` recognises its run by the `--` that
    `helperArgv` passes, so `helperArgv` no longer calls `t.Setenv`, which a
    parallel test may not.
* **Seen to fail:** with both hook calls removed, `the waiter never joined
  the leader's run` and `... refresh`, each after 5 s.
* **`-count=50 -race`:** both pass.

**6.6, the harness (D5, W12).**
* **Built:**
  * `Harness.Fidelity`, `Garbled`, `Truncated`, `TruncatedReason` (the
    deviation above) and `StrictTools`;
  * the constants `FidelityModel`, `FidelityInstructions` and
    `FidelityOutput`;
  * four checks, run only when set;
  * all nine providers' harnesses set all four;
  * `llmtest`'s reference provider passes them, and has a flaw for each.
* **What the checks found,** each a deviation above:
  * the empty finish reason, fixed in `wire.Finish`, with `TestFinish`;
  * the cut answer's `Reason`, pinned per wire;
  * W14 in `ReadStream`, fixed: `finished`, with
    `TestReadStream_FailureReturnsNoResponse`. That test failed on `HEAD` in
    all 8 cases, each `ReadStream = &{... Output:[{... Text:partial}] ...},
    <error>`.
  * No golden changed.
* **Seen to fail,** each fault planted in one provider on a scratch copy:
  * Claude without `Instructions`: `W12 (fidelity, 0021-MADR D5): the
    request's Instructions, "llmtest-instructions-7f3a", never reached the
    wire`;
  * Kilo bypassing `DecodeError`: `Generate returned ... kilo failed after
    3 attempts ... want an error matching ErrIncomplete`, and `WithRetry
    sent 3 request(s); want 1`;
  * Chat Completions without the length check, on Kilo and Hugging Face:
    `W12 (a cut answer, 0021-MADR W3): Generate returned <nil>`;
  * Messages with `Arguments ""`, on Claude: `W12 (strict tools, 0021-MADR
    W1): the call to "llmtest_tool" has arguments ""`.

  In `llmtest`'s own tests, each of the reference provider's four flaws
  fails its check by name, and `TestRun_W12ChecksAreOptional` shows a
  harness that sets none passes.

**6.7, close-out docs.**
* `AGENTS.md` names `gate-selftest`.
* `docs/architecture.md`:
  * the script list;
  * the `make` targets, `nolintlint`, dep-check, generate-check, G-parity,
    `gate-selftest`, the floors, the precheck and CI;
  * the `llmtest` passage.
* `docs/guides/adding-a-provider.md` §8 names the four checks.
* **The PLAN stays `in-progress`:** phase 7 is not yet run, so the Goal does
  not yet hold.

**Gate,** all exit 0 (from this phase, G4 with `-shuffle=on`, and G7 with
`gate-selftest`):

* G1, G2 and G3;
* G4:
  * 26 packages ok;
  * `go test -race -shuffle=on ./...` exit 0;
  * `llmprovider` 97.9%, `internal/wire` 94.2%, `responses` 97.7%,
    `llmtest` 96.0%, `wizard` 88.5%;
* G5, and G6 with 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: 0 incompatible changes;
  * generate-check: 0 problems;
  * gate-selftest: 7 tests OK in 63 s;
* G8: 0 issues; G9: stable; G10: 0 problems; G11: 0 hits in 53 files.

### Deviation 2026-10-05: step 7.2a finds both forms obeyed

* **Found:** see the MADR's amendment "L2 is the test's prompt, not the
  wire" for the counts.
  * The test's override instruction: 0/3 in both forms on `grok-4.6` and
    `grok-4.7`, and 1/3 and 0/3 even in the user's message.
  * A neutral instruction: 3/3 in both forms on both models, and 0/3
    without it.
  * The step's "if both forms are obeyed" branch: stop and prompt.
* **Resolution, chosen by the owner** ("Keep wire; paired test"):
  * 7.2b is not done: `grok.go` and the goldens are unchanged;
  * `TestLive_GrokInstructions` is a pair: with `Instructions` "Begin every
    reply with the word OMEGA." the reply contains `OMEGA`; the same
    request without them has no `OMEGA`;
  * seen to fail on a scratch copy with `body` dropping `Instructions`.
* **Not changed:** `live_gemini_test.go:61` uses the same override wording
  in a system message. It is outside L2 and was not measured here.
* **Files added to the phase:** none; `llmprovider/live_grok_test.go` was
  already 7.2's.

### Phase 7: live drift (2026-10-05)

**Phase 6's pending CI check.** CI run 37339373303, on `7d7ead9`: success
on `ubuntu-24.04`, `windows-2025` and `macos-15`. Step 6.3's change is seen
working.

**7.1, OpenCode's metadata route test (L1).**
* **Built:** `TestLive_OpencodeRoutesFromMetadata` reads
  models.opencode.ai's `api.json` itself (`liveNPM`). It maps
  `qwen3.8-max`'s `provider.npm` on `opencode-go` through its own table,
  rather than the provider's: `@ai-sdk/openai` → `/responses`,
  `@ai-sdk/anthropic` → `/messages`, `@ai-sdk/google` → `:generateContent`,
  anything else → `/chat/completions`. It skips when the document cannot
  be read or the model is not listed, and logs the npm. The comment says
  what it pins.
* **Live, on a tree copy, twice:** both pass, with `qwen3.8-max on
  opencode-go: provider.npm "@ai-sdk/anthropic", so /messages`.
* **Seen to fail:** on a scratch copy with the table's
  `"@ai-sdk/anthropic"` row planted as `/chat/completions`: `request paths
  = [/zen/go/v1/messages], want one ending /chat/completions (provider.npm
  "@ai-sdk/anthropic")`, exit 1.

**7.2, Grok's instructions (L2).**
* **7.2a, the probe:** see the deviation "step 7.2a finds both forms
  obeyed", and the MADR's amendment "L2 is the test's prompt, not the wire"
  for the counts.
  * A scratch test, never in the tree: form (b) is made by a transport that
    moves the leading system message to `instructions`, and counts each
    move (`moved=1` on every (b) run).
  * `grok-4.7` was named by hand: the listing's first entry is `grok-4.6`,
    since the listing is ordered by curated rank, not by release.
* **7.2b, not done,** by the owner's choice. `grok.go` and the goldens are
  unchanged.
* **Built:** `TestLive_GrokInstructions` has two subtests, `with` and
  `without`. Each asks `grok-4.6` to `Say hello.`; `with` carries
  `Instructions` "Begin every reply with the word OMEGA.".
* **Live, on a tree copy, twice:** both pass, each subtest included.
* **Seen to fail:** on a scratch copy whose `body` drops `Instructions`
  (`if false && req.Instructions != ""`): `with` fails with `contains OMEGA
  = false, want true`, exit 1, and `without` passes.

**7.3, OpenCode's 403 (L3).**
* **Red,** on a scratch copy of `HEAD` (`7d7ead9`) with the new test:
  `TestClassifyHTTPError_OpencodeForbidden` fails in 6 of its 10 subtests.
  Those are the untyped JSON, plain-text and empty 403s, on both
  `opencode-go/responses` and `opencode-zen/chat_completions`. Each fails
  with `llmprovider: authentication failed: opencode-go/responses HTTP 403:
  ... (kind llmprovider: authentication failed, terminal true); want kind
  llmprovider: not permitted, terminal`.
* **Guards,** passing on `HEAD` and on the tree:
  * the typed `RegionError` 403 and the 401 `Invalid API key.` subtests;
  * `TestClassifyHTTPError_Table`, with its `403 otherwise` row on Claude;
  * `TestClassifyHTTPStatus`'s `403` row, on `gw/route`.
* **Built:**
  * `classifyAPIError`'s `ErrNotPermitted` case takes `service ==
    serviceOpencode && (status == http.StatusForbidden || has(...))`;
  * the test checks `Kind`, since the error also unwraps to the 403's
    pre-0012 sentinel, `ErrAuthFailure` (0012-MADR §7), as a typed OpenCode
    403 and Kilo's 403 already do;
  * `wire.Reauth` retries only a 401, so a 403 never caused a re-sign-in,
    and still does not.
* **Green:** 62 subtests of the classification and `APIError` tests pass,
  0 fail.
* **Records:**
  * `0012-MADR-conform-providers-to-reference-clients.md` gains "Amendment
    2026-10-05: OpenCode's 403 (0021 L3)", with the row;
  * its date is 2026-10-05.

**Phase 7 live check,** on a tree copy, once: `TestLive_OpencodeResponses`,
`TestLive_OpencodeResponsesStoreFalse`,
`TestLive_OpencodeRouteStillEnforced`, `TestLive_OpencodeToolRoundTrip`
(`go-chat`, `go-messages`, `go-responses`), `TestLive_OpencodeToolChoices`
(six subtests) and `TestLive_OpencodeRoutesFromMetadata`. All pass, none
skip. The picker chose `gpt-6-luna`.

**The PLAN stays `in-progress`.** Phase 7 completes the PLAN's steps, but
the Goal's live checks are not all run: V3.5, masked entry on Windows, is
the owner's and pending.

**Gate,** all exit 0 (G4 with `-shuffle=on`, and G7 with `gate-selftest`,
as from phase 6):

* G1, G2 and G3: `make pre-add-check`, "396 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck)"; `go vet` for darwin,
  linux, windows and `live_gateways`;
* G4:
  * 26 packages ok with `-race -cover`;
  * `go test -race -shuffle=on ./...` exit 0;
  * `llmprovider` 97.9%, `wizard` 88.5%;
* G5: `go mod tidy -diff` clean;
* G6: 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.1.0`, 0 incompatible changes;
  * generate-check: 1 generated file, 0 problems;
  * gate-selftest: 7 tests OK in 58 s;
* G8: 0 issues on the repository's markdownlint scope, which
  `.markdownlint-cli2.jsonc` defines without the records;
* G9: stable; G10: 0 problems; G11: 0 hits in 7 files.
