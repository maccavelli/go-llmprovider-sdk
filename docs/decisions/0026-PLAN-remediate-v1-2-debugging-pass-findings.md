---
status: in-progress
date: 2026-10-07
associated-madr: "0026-MADR-remediate-v1-2-debugging-pass-findings.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Remediation of the v1.2 Debugging Pass

Associated MADR: [0026-MADR-remediate-v1-2-debugging-pass-findings.md](0026-MADR-remediate-v1-2-debugging-pass-findings.md)

## Goal

Every finding F1–F65 is fixed and proven, or is recorded here as not done,
with its reason. The fixes follow the owner's answers of 2026-10-06: Q1
(a), Q2 (a), Q3 (a), Q4 (a), Q5 (A), Q6 (a) and Q7 (a).

When it is done:

* the repository gate passes, `make gate-selftest` included;
* `api-check` reports only additive changes against `v1.2.1` (R48);
* the live checks under Verification pass with the owner's keys;
* every behaviour the docs promise holds, and the docs say what the code
  does;
* `v1.3.0` is tagged by the owner, with release notes naming the two
  behaviour changes: F2's refusal and F11's refusal.

## Scope

### In scope

The six phases of the MADR's Decision Outcome §1, in order, then a release
phase. Each phase touches one area and ends green, in its own commit. The
agent stages; the owner commits.

| Phase | Area | Findings |
| :--- | :--- | :--- |
| 0 | records | the MADR accepted, this PLAN approved |
| 1 | credentials | F1, F3, F13, F2, F40, F41, F42, F43, F44, F45 |
| 2 | errors and retry | F4, F5, F11, F12, F17, F18, F19, F20, F21, F22, F23, F24, F67 |
| 3 | answers on the wire | F7, F8, F25, F26, F27, F28, F29, F30, F31, F32, F65 |
| 4 | provider wiring | F6, F36, F37, F33, F34, F35, F38, F39 |
| 5 | wizard and catalog | F9, F10, F46, F47, F48, F49, F50, F51, F52, F53, F54 |
| 6 | harness, gates and docs | F14, F15, F16, F55, F56, F57, F58, F59, F60, F61, F62, F63, F64, F66 |
| 7 | release | `v1.3.0`, the owner's tag |

### Out of scope

* 0002's later phases, and the items the MADR lists as already recorded.
* Any exported API change that is not additive. A fix that seems to need
  one stops for a deviation.
* Mistral's 9-character call ids. Q5 (A)'s helper is built so that the rule
  can plug in, but no Mistral route is in scope.
* Consumers. prepare-commit-msg and gobble-cli move to `v1.3.0` under their
  own records.
* Push and tags, which are the owner's.

## Rules for every phase

1. **Red first.** Each finding gets a test that fails on a scratch copy of
   the tree before its fix:
   * either the planted state of the finding,
   * or the unchanged code, when the test is new and the code is the
     defect.

   The FAIL line is recorded in the Execution Record, and so is the PASS
   after the fix. Where the MADR quotes an audit's reproduction, that
   reproduction becomes the test, renamed to the package's convention.
2. **A fix is proven on the path that broke** (the MADR's driver "fixes do
   not create the next pass"). Where a finding came from an earlier fix
   (F8 from 0021 W10, F9 from 0021 C13), the test also covers the case that
   earlier fix was for, so neither regresses.
3. **The gate before staging,** all clean:
   * `make pre-add-check`;
   * `CGO_ENABLED=0 go vet ./...` for Linux, macOS and Windows, with and
     without `-tags live_gateways`;
   * `go test -race -count=1 ./...`, and `go test -shuffle=on ./...`;
   * `go mod tidy -diff`;
   * `make lint` (host and Windows);
   * `make parity-check dep-check coverage-check api-check generate-check
     records-check gate-selftest`;
   * markdownlint on the files the repository lints, links, and the
     identifier scan.
4. **G-wire.** A change to what goes on the wire updates its goldens with
   `-update` in the same commit (R45). The Execution Record lists each
   changed golden and why.
5. **Live checks wait for the owner.** A step marked *live* is run with the
   owner's key, through the live-tagged tests, and its output is recorded.
   A fix that depends on one does not start before it.
6. **Deviations stop and prompt,** with evidence and resolutions. The
   chosen resolution is recorded here, and in the MADR when a decision or
   asserted fact changes, before work continues.
7. **Records cite records by full filename.** Nothing committed carries a
   hostname, an account name or a machine path.

## Implementation Steps

### Phase 0: records

1. The MADR is `accepted` (2026-10-06). The owner reviews this PLAN and
   approves it, and it becomes `in-progress`.
2. `docs/README.md` indexes both records with their statuses.
3. The owner commits the records.

### Phase 1: credentials

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F1 | `TestOAuthSession_RepeatedFailedSavesKeepRotation`, from the audit's `TestZZ_TwoFailedSavesLoseRotation`: <ul><li>a store whose `Save` fails twice, then works;</li><li>a refresh, then an `InvalidateToken`, then a second refresh, then a resave, then a third refresh, against an issuer that rejects a reused token.</li></ul> It asserts every refresh token sent is distinct and that the store ends on the newest. It fails today with `[old-refresh refresh-1 old-refresh]`. A table runs it with 1, 2 and 3 failed saves; the 1-save case is 0020 F1's own test, kept. | Q1 (a), first part: <ul><li>the session records `storedRefresh`, the refresh token the store is known to hold, set by `Load` and by every successful `Save`;</li><li>`loadRotated` adopts a stored session only when its refresh token differs from both `storedRefresh` and the session's own;</li><li>`persistRotation` writes only while the store still holds `storedRefresh`, or nothing;</li><li>`spentRefresh` is removed if nothing else reads it.</li></ul> | `auth/oauth_session.go`, its test |
| F3 | `TestOAuthSession_CallerCancelKeepsRotation` and `TestOAuthSession_AbandonedWaiterDoesNotResendSpent`, from the audit's two tests. The caller's ctx ends after the issuer rotated, and before the reply is read. The next `Token` sends the rotated refresh token, and the store holds it. Both fail today with `sent: [old-refresh old-refresh]`. | Q1 (a), second part: <ul><li>the refresh runs under `context.WithoutCancel(ctx)`, bounded by the existing 15 s per attempt;</li><li>it runs as the in-flight future, and the leader and every waiter wait on it or on their own ctx, whichever ends first;</li><li>a refresh that completes after its callers have gone still adopts and saves its result;</li><li>0020 F14's rule, that a waiter re-runs a fetch its leader abandoned, applies to `CommandToken` only, because the session no longer abandons.</li></ul> | `auth/oauth_session.go`, its test |
| F13 | ~~`TestFileTokenStore_TakeoverRaceOneHolder`, from `TestZZ_TakeoverRaceTwoHolders`, driven through the same seams;~~ *(Deviation D2: the seams go with the takeover; the race test is replaced by a mutual-exclusion stress test.)* `TestFileTokenStore_LockReleasedWhenHolderDies`, a child process that takes the lock and is killed. The first fails today with `W3 err=<nil> holds=true`. | Q1 (a), third part: OS file locks. <ul><li>A new `llmprovider/internal/filelock` package locks an open file exclusively:<ul><li>`syscall.Flock` with `LOCK_EX|LOCK_NB` on Unix;</li><li>`LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` on Windows, through `//sys` lines and its own `//go:generate mkwinsyscall` output, as `ownerperm` has.</li></ul></li><li>A non-blocking attempt is retried with backoff, until the ctx or `lockWait` ends.</li><li>The lock file `<provider>.oslock` is never deleted, so no process can lock an unlinked file.</li><li>The OS releases a dead holder's lock, so staleness, takeover and the heartbeat go.</li><li>~~**Compatibility:** while it holds the OS lock, a process also creates the legacy `O_EXCL` lock file with its owner token, and removes it on unlock. A `v1.2.x` process sharing the store then still waits, as it did before.~~ *(Deviation D2: no legacy file; the old lock goes entirely.)*</li></ul> | `llmprovider/internal/filelock` (new), `auth/tokenstore_file.go`, their tests; `docs/guides/api-standards.md` R2 and `docs/architecture.md` gain the package; `generate-check` covers its generated file |
| F2 | `TestNew_RefusesForeignSession` in `grok` and `openai`: <ul><li>a ChatGPT, Kilo and Grok `OAuthSession`;</li><li>a Codex and a Grok `VendorCLISession`;</li></ul> each given to the provider that does not own it. Each must fail `New` with `ErrUnsupported`, and send nothing. They fail today with `New err=<nil>`. `TestIsChatGPTSession_NoIssuer`: an OpenAI session with no issuer is a ChatGPT session. | Q2 (a): <ul><li>`grok.New` refuses an `*auth.OAuthSession` or `*auth.VendorCLISession` whose `Provider` is not `grok`;</li><li>`openai.New` does the same for `openai`;</li><li>`OAuthSession.ChatGPT()` (or `isChatGPTSession`) counts an OpenAI session with an empty issuer, matching `refreshTokenURL`.</li></ul> The messages follow `kilo.New`'s. | `providers/grok`, `providers/openai`, `auth/oauth_session.go`, their tests |
| F40 | `TestVendorCLISession_SymlinkedAuthFile`: a Grok `auth.json` that is a symlink to a file outside its directory is read. It fails today with `path escapes from parent`. | Resolve the path with `filepath.EvalSymlinks` first, then open the resolved file. The `os.Root` stays for the resolved directory, as gosec wants (0006-PLAN). The error for a missing target says the link is broken, not "run grok login". | `auth/vendor_session.go` |
| F41 | `TestVendorCLISession_TornFileRetriedOnce`: a file that is truncated on the first read and whole on the second is read. It fails today with `unexpected end of JSON input … (kind auth)`. | A decode failure is retried once, after 50 ms. A second failure keeps today's error. | `auth/vendor_session.go` |
| F42 | Windows only, in CI: `TestFileTokenStore_SaveWhileOpenForRead`. A reader holds `<provider>.json` open while `Save` runs, and the save succeeds. It is read-only today on macOS, so its first failure is recorded from CI's `windows-2025` job on a scratch branch the owner pushes, or from the owner's Windows host. | Open token files for reading with `FILE_SHARE_DELETE`, through a Windows-only open helper beside `filelock`'s bindings (`CreateFile`), ~~so a rename can replace a file a reader has open~~. *(Deviation D3: that alone does not let `os.Rename` replace the file; `Save` also renames through `os.Root`, a POSIX-semantics rename on Windows.)* | `auth/tokenstore_file.go`, a `_windows.go` helper |
| F43 | `TestRefreshErrors_HaveKinds`, from `TestZZ_RefreshErrorKinds`, plus the device "denied" and "expired" cases. Each error matches an `llmprovider` kind: <ul><li>a transport failure is `ErrProviderUnavailable`;</li><li>a 200 that does not decode is `ErrIncomplete`;</li><li>a lock file that cannot be created is `ErrProviderUnavailable`;</li><li>denied is `ErrAuthFailure`;</li><li>expired is `ErrAuthFailure`.</li></ul> They fail today with `kind NONE`. | Wrap each with its kind, keeping the message. | `auth/oauth_session.go`, `tokenstore_file.go`, `oauth_device.go`, `kilo_device.go` |
| F44 | `TestValidateOAuthSession_RefusesUnrefreshable`: a custom-issuer session with a refresh token and no token URL is refused. It passes `ValidateOAuthSession` today. | Refuse it there, with the same message `Token` gives, and correct the doc comment (0020 F45). | `auth/oauth_session.go` |
| F45 | `TestRevoke_LegacyGrokSession`: a Grok session with no issuer revokes against `auth.x.ai`'s discovery, or returns `ErrUnsupported` if that has no endpoint. It fails today with `unsupported protocol scheme ""`. | Revocation derives the issuer as `refreshTokenURL` does. | `auth/oauth_revoke.go` |

### Phase 2: errors and retry

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F4 | `TestPost_StalledErrorBodyEndsAtIdleLimit`, from `TestZZReproStalledErrorBody`: a 503 whose body stalls returns within the idle limit, with no caller deadline. It fails today, returning only at the 3 s deadline. A 200 that stalls keeps today's behaviour. | `Post` wraps `resp.Body` in the idle-limited `ReplyReader` before `ClassifyHTTPError`. `ClassifyHTTPError`'s own bound stays. | `internal/wire/post.go`, its test |
| F5 | `TestChatCompletions_ErrorIn200Classified`, from `TestZZReproGatewayErrorIn200` and `TestReproGatewayErrorCodes`: a numeric `error.code` gives the status's kind. <ul><li>400 overflow is `ErrContextOverflow`;</li><li>401 is `ErrAuthFailure`;</li><li>402 is `ErrQuotaExhausted`;</li><li>403 is `ErrNotPermitted`;</li><li>429 is `ErrRateLimited`.</li></ul> `APIError.Provider` is the provider's label. A Kilo or OpenCode type gives its row's kind. The server is hit once for terminal kinds. These fail today. | A numeric code from 400 to 599 is classified through `classifyAPIError(serviceOf(provider), code, env, nil)`, overflow check included. A non-numeric code keeps `ClassifyStreamFailure`. The decoder receives the provider label from its caller. | `api_error.go`, `internal/wire/chatcompletions`, the providers that call it |
| F11 | `TestResolveOptions_RefusesUnsendable` and `TestWithRetry_UnsendableIsTerminal`, from `TestZZReproDeterministicURLErrorRetried`: <ul><li>a base URL with no scheme or host, or with an invalid escape, fails `New` with `ErrInvalidRequest`;</li><li>a key, client info or session id with a control character fails `New` the same way;</li><li>a source token with a newline fails on the first attempt, with `ErrInvalidRequest`, and is not retried.</li></ul> They fail today with `attempts=4 kind=NO KIND`. | Q7 (a): <ul><li>`ResolveOptions` parses `WithBaseURL` (scheme and host required), and refuses control characters in `WithAPIKey`, `WithClientInfo` and `WithSessionID`;</li><li>`retryable` treats a `*url.Error` from parsing, an invalid header value, or an unsupported scheme as terminal, and `Post` gives it `ErrInvalidRequest`.</li></ul> 0021-PLAN kept `WithBaseURL` permissive (R48); Q7 (a) reverses that, so the release notes say so. | `settings.go`, `retry.go`, `internal/wire/post.go`, their tests |
| F12 | *Live first* (Q6 a): capture a real Gemini 429 with the owner's key, in a live-tagged test that records the body shape and headers. Then `TestClassify_GeminiRetryInfo`: the captured body gives `RetryAfter` from `retryDelay`, and a per-day `QuotaFailure` gives `ErrQuotaExhausted`. It fails today with `RetryAfter=0s retryable=true`. | Decode `details[].retryDelay` (a protobuf Duration string) into `RetryAfter` when no header is present. Classify a `QuotaFailure` violation whose quota id names a day as `ErrQuotaExhausted`. If the capture differs from Google's documented shape, that is a deviation. *Amended by deviation D4: no capture could be made, and the owner took Q6 (b); the fixture follows Google's documented shape, and the live test stays, opt-in.* | `api_error.go`, a live test |
| F17 | `TestRedact_PrefixedKeys`: `openai_api_key=`, `"x_api_key"`, `app_secret:`, `"apiSecret"`, `db_password=` and `"subscription_key"` are redacted. They fail today. | `reKV` matches a key name at the end of a snake_case or camelCase identifier, not only after a word boundary. | `internal/redact` |
| F18 | `TestAPIError_ReasonIsBounded`: a `Reason` with ESC and BEL, 2 KiB long, reaches `Error()` stripped and capped. It fails today with `ESC=true BEL=true len=2093`. The token endpoint's error body is stripped too. | `StripControl` and the code bound on `Reason`, where it is set; `StripControl` on the token endpoint's message. | `api_error.go`, `internal/wire/finish.go`, `responses.go`, `auth/oauth_session.go` |
| F19 | `TestNew_RefusesInvalidDefaultReasoning`: `WithReasoning` with an unknown effort or a negative budget fails `New`. It fails today with `New err=<nil>`. | `ResolveOptions` applies the `Request` validation to `WithReasoning`. | `settings.go` |
| F20 | `TestValidate_UnmarshalableSchema`: a `Tool.Schema` holding a channel, or an invalid `json.RawMessage`, is `ErrInvalidRequest` before any request. It fails today with `kind=NO KIND`. | `validate` marshals each schema once. `Post`'s marshal failure is `ErrInvalidRequest`. | `contract.go`, `internal/wire/post.go` |
| F21 | `TestRedact_KeepsOrdinaryWords`: `"Invalid bearer token"`, `"Basic authentication is not supported"` and `"The token provided is invalid"` come back unchanged, while `Bearer sk-…` and `Basic dXNlcjpwYXNz` are redacted. It fails today. | `reAuth` requires a credential-shaped value after the scheme: base64 or token characters, at least 8 long, not an English word followed by a space. | `internal/redact` |
| F22 | `TestRetry_ServerWaitNeverExceedsMaxDelay`, from `TestZZReproServerWaitOverMaxDelay`: over 20 runs, no wait exceeds `MaxDelay`. It fails today at 338 ms over 100 ms. | Cap the jittered wait at `MaxDelay`. | `retry.go` |
| F23 | `TestAPIError_RetryableDoesNotMatchInvalidRequest`: a retryable 408, or 409 on openai and claude, does not match `ErrInvalidRequest`. It fails today with `isInvalidRequest=true`. | `Unwrap`'s legacy sentinel is omitted when the error is `Retryable()`. 0012 §7's compatibility rule is kept for terminal errors, and its amendment records the exception. | `api_error.go`, `0012-MADR` amendment |
| F24 | `TestPost_Accepts2xx`: a 201 decodes as a success. It fails today with `invalid request … HTTP 201`. | `ClassifyHTTPError` passes 200–299. Its doc and `Post`'s agree. | `api_error.go` |
| F67 | `TestAPIError_ShouldRetryIsNotInvalidRequest` (added by deviation D5): a 400 with `x-should-retry: true`, on openai, claude and kilo, is retryable, of kind `ErrProviderUnavailable`, and does not match `ErrInvalidRequest`. A 401 with the header keeps `ErrAuthFailure`; a 400 without it stays `ErrInvalidRequest`. It fails today with `invalid request … retryable=true`. | In `ClassifyHTTPError`, `x-should-retry: true` on a kind that matches `ErrInvalidRequest` sets the kind to `ErrProviderUnavailable`. 0012's amendment of 2026-10-07 and the migration guide say so. | `api_error.go`, `0012-MADR` amendment, `docs/guides/migrating-from-mcplib.md` |

### Phase 3: answers on the wire

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F7 | `TestDecode_ReasoningOnlyCutIsIncomplete` on Chat Completions, Messages and generateContent, from `TestReproReasoningOnlyCut`. Reasoning, no text or call, and a length finish is `ErrIncomplete` with `Reason "length"`. It fails today with `ok finish=length text=`. A cut text answer keeps its text with `FinishLength`, as the README says. | Q4 (a): with a length finish and neither a message nor a call, each decoder returns `wire.EmptyAnswer(where, FinishLength)`. `llmtest` gains a check that a reasoning-only cut answer is `ErrIncomplete` on every provider. | the three wires, `llmtest` |
| F8 | `TestFromItems_CallIDsFitAnthropicRule`, from `TestReproCrossWireCallID` and `TestReproGeminiIDsAcrossTurns`. A Messages request built from Gemini items has these properties: <ul><li>every `tool_use.id` matches `^[a-zA-Z0-9_-]+$` and is at most 64 long;</li><li>ids repeated across turns (`get_weather#0` twice) become distinct;</li><li>each `tool_result` pairs with the call before it.</li></ul> It fails today with `"id":"get_weather#0"` twice. Rule 2: two calls to one function in one reply still decode as distinct CallIDs (0021 W10's own test, kept). | Q5 (A): a shared helper in `internal/wire` returns a remapped copy of the items. <ul><li>Each call id that breaks the target's rule, or repeats an earlier call in the request, gets a valid unique id: `[^a-zA-Z0-9_-]` becomes `_`, cut to 64, then `_2`, `_3`… on a collision.</li><li>Each result takes the id of the latest call before it with its original id.</li><li>`messages.FromItems` uses it with Anthropic's rule.</li></ul> The caller's items and the decoded ids do not change. No golden moves. | `internal/wire`, `internal/wire/messages`, their tests |
| F25 | `TestGenerateContent_BlockReason`: `{"promptFeedback":{"blockReason":"SAFETY"}}` gives `ErrIncomplete` with `Reason "SAFETY"`. It fails today with `reason=`. | Decode `promptFeedback.blockReason` into `Reason`. | `internal/wire/generatecontent` |
| F26 | `TestGenerateContent_ToolCallFailureReasons`: an empty answer with `MALFORMED_FUNCTION_CALL` keeps that reason. It fails today with `reason=stop`. | Keep the two values as sent (0021 W3's rule for unknown values). | `generatecontent.go` |
| F27 | `TestToolSchema_TypedNil`: a nil `map[string]any` and a nil `json.RawMessage` send an empty object schema. They fail today with `null`. | `ToolSchema` treats a typed nil as nil. | `internal/wire/tools.go` |
| F28 | `TestResponses_FailedStatusClassified`, from `TestReproResponsesFailedAsymmetry`: a non-streamed `"status":"failed"` with an `error` is classified through `ClassifyStreamFailure`, as in the stream, keeping its message. It fails today with `incomplete=true reason=stop`. | `Decode` checks the status first, as `ReadStream` does. | `internal/wire/responses` |
| F29 | `TestChatCompletions_Refusal`: `message.refusal` gives its text with `FinishContentFilter`, as on the Responses wire. It fails today with `reason=stop`. | Decode `refusal`. | `chatcompletions.go` |
| F30 | `TestChatCompletions_ContentParts`: `content` given as an array of text parts decodes to their joined text. It fails today with `cannot unmarshal array`. | Accept a string or an array of parts. | `chatcompletions.go` |
| F31 | `TestResponses_EmptyArguments`: `"arguments":""` gives `"{}"`, on both `Decode` and `ReadStream`. It fails today. | Normalise as the other decoders do (0021 W1). | `responses.go` |
| F32 | `TestChatCompletions_ReasoningNotCarriedAcrossTurns`, from `TestReproChatPendingReasoningAcrossTurns`: reasoning not followed by an assistant message is not attached to a later turn. It fails today with `turn-1 reasoning` on turn 2. | Drop pending reasoning at a user turn. | `chatcompletions.go` |
| F65 | *Live first*: a live-tagged two-call round trip on a `gemini-3.x` model through OpenCode's google route. Two calls to one function, then both results. It records whether the service accepts results paired by name alone, and whether it sent `functionCall.id`. | Only if the service refuses or mis-pairs: `Contents` sends `id` in `functionCall` and `functionResponse` when a call carries a service-issued id, as pi does for Gemini 3. Otherwise F65 is recorded as measured, and closed. If the service needs ids unique across turns, Q5 (B) follows under a MADR amendment. | `generatecontent.go`, a live test |

### Phase 4: provider wiring

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F6 | `TestReauth_RerunsOnAuthFailureKind`, from `TestZZRepro_GeminiReauthOnInvalidKey`. A `CommandToken` on Gemini, against a server answering 400 `API_KEY_INVALID` then 200, reruns once and succeeds. It fails today with `requests=1 invalidations=0`. A 403 of kind `ErrNotPermitted` does not rerun. | Q3 (a): `Reauth` reruns when `apiErr.Kind` (with `kind()`'s status fallback) is `ErrAuthFailure`, whatever the status. It tests the kind, not `errors.Is`, because a not-permitted 403 also matches `ErrAuthFailure` through the legacy sentinel. `llmtest` gains a re-auth variant with the vendor's own refusal: `Harness.AuthFailure`, a status and a body, defaulting to 401. Gemini's harness sets it. | `internal/wire/reauth.go`, `llmtest`, `providers/gemini` |
| F36 | `TestList_RerunsCredentialOn401`, from `TestZZRepro_TogetherListing401`: a listing that gets 401 with an invalidating source reruns once, then lists live. It fails today with `invalidations=0` and the static catalog. | Q3 (a): the catalog listers fetch through `Reauth`'s rule. A listing that still fails after the rerun keeps the degrade-to-static contract, with `Catalog.Err` set. | `catalog/discovery.go` |
| F37 | `TestSession_LaterProviderOptionsApply`, from `TestZZRepro_SessionSharingFirstProviderWins`: a second `New` on the same session with `WithHTTPClient` and `WithLogger` makes the session use them. It fails today. | `New` passes the caller's client and logger only when the caller gave them. `UseHTTPClient` and `UseLogger` replace a default value, but never a caller's, and the doc says so. | `providers/openai`, `providers/grok`, `auth/oauth_session.go` |
| F33 | `TestOpencode_MetadataLookupUsesClientInfo`: the generation-time metadata request carries the caller's `WithClientInfo` User-Agent. It fails today with the module's own. | `LookupMetadata` takes the provider's settings (an internal variant, or options passed through). `catalog`'s exported API stays additive. | `catalog/model_metadata.go`, `providers/opencode` |
| F34 | `TestChatGPTListing_BoundedAndKinded`, from `TestZZRepro_ChatGPTListingKinds`: a 40 MiB body is refused by the listing cap; an undecodable body is `ErrIncomplete`; an empty catalog is `ErrIncomplete`. They fail today. | Apply catalog's listing cap (0021 C3) and kinds. | `providers/openai/chatgpt_listing.go` |
| F35 | `TestOllama_ListModelsKinds`, from `TestZZRepro_OllamaListingKinds`: <ul><li>503 is `ErrProviderUnavailable`;</li><li>401 is `ErrAuthFailure`;</li><li>unreachable is `ErrProviderUnavailable`.</li></ul> They fail today with `NO KIND`. | Classify through `ClassifyHTTPError`, and wrap transport errors. | `catalog/discovery.go`, `providers/ollama` |
| F38 | *Live first*: `gemini-2.5-flash-lite` with low effort, through `TestLive_…` with the owner's key, recording whether thinking happens. | If it does not think: either Gemini declares Reasoning `BestEffort`, as 0020 F43 did for Grok, or the provider sends a thinking budget for 2.5-series models. The live result decides which, and it is recorded. | `providers/gemini` |
| F39 | none, documentation: four doc comments corrected (Ollama, Kilo, Grok, `WithSessionID`). Checked by reading against the code and 0021 L2. | Correct the comments. | the four files |

### Phase 5: wizard and catalog

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F9 | `TestConfigure_EmptyRecommendedHonoursCtx`, from `TestZZ_SelectModelIgnoresCtx`: a prompter that accepts defaults, a listing that recommends nothing, and a ctx of 300 ms. `ConfigureLLM` returns `context.DeadlineExceeded` promptly. It fails today, still running after 1.7 s. Rule 2: 0021 C13's search path still opens (`TestConfigure_EmptyRecommendedOpensSearch`, kept). | `selectModel` and `selectFallbacks` take `ctx` and check it each pass. With nothing recommended, the menu offers the current model, if any, and Other, never an endless re-prompt. | `wizard/model_select.go`, `configure.go` |
| F10 | `TestConfigure_KeptSessionUsesCallersClient`, from `TestZZ_KeptSessionRefreshBypassesClient`: the kept session's refresh goes through `Options.HTTPClient`. It fails today. | `keepExistingOAuth` calls `session.UseHTTPClient(o.HTTPClient)` beside `session.Store`. `catalog.List` hands `cfg.HTTPClient` to a source that has `UseHTTPClient`. | `wizard/auth.go`, `catalog/discovery.go` |
| F46 | `TestSelectRecommended_ExistingModelOnlyForItsProvider`: a saved Together model is not the Hugging Face default. It fails today with `index=2`. | Compare `Existing.Provider` (0007 §4.3). | `wizard/model_select.go` |
| F47 | `TestChatGPTCatalog_Capped`: 14 listed models give at most `MaxListed` recommended. It fails today with 14. | Cap as `providerCatalog` does. | `wizard/configure.go` |
| F48 | `TestPasteAccessToken_KeepsAccount`: a pasted JWT carrying `chatgpt_account_id` saves `AccountID` and `FedRAMP`. It fails today with `""`. | Take them from the decoded payload, as logins do from the id_token. | `wizard/auth.go` |
| F49 | `TestTextPrompter_SecretSurfacesWriteError`: a failed write is returned by `Secret`. It fails today with `err=<nil>`. | Apply `flushErr`, and return a read error with the value. | `wizard/text_prompter.go` |
| F50 | `TestTextPrompter_SecretRefusesPartialEntry`: a read error mid-entry gives an error and no value. It fails today with `"sk-partial-k"`. | `readMasked` returns `""` and the error on a mid-entry failure. | `text_prompter.go` |
| F51 | none, documentation: `Options.Profile` names Together. | Correct the comment. | `wizard/configure.go` |
| F52 | `TestValidateOllamaURL_TrailingSlash`, from `TestZZ_ValidateOllamaTrailingSlash`: a URL ending in `/` requests `/api/version`. It fails today with `//api/version`. | Trim the trailing slash, as `ResolveOptions` does (0020 F30). | `catalog/discovery.go` |
| F53 | `TestMetadata_ReasoningEffortsIsACopy`, from `TestZZ_ReasoningEffortsAliasesCache`: editing the returned slice does not change a later lookup. It fails today with `[POISONED medium high]`. | Return a clone (R29). | `catalog/model_metadata.go` |
| F54 | none, documentation: `Catalog.Usable` and `Search`'s tie order say "in the order each lister gives", naming Kilo's price order and Hugging Face's speed order. | Correct the comments. | `catalog/discovery.go`, `model_matcher.go` |

### Phase 6: harness, gates and docs

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F14 | `test_parity_check_bare_missing_function` in `scripts/test_gates.py`: a cell naming `StaticModels(ProviderClaude)` fails the gate. It passes today. | The eight cells name `catalog.Static(llmprovider.ProviderX)`. `unresolved()` resolves a bare call-shaped span only against package-level functions and types. | `docs/guides/migrating-from-mcplib.md`, `scripts/check_parity_map.py`, `scripts/test_gates.py` |
| F15 | A precheck self-test: `go-precheck.sh` given only a deleted `.go` path that breaks its package exits non-zero. It exits 0 today with `no Go files to check.` | A named `.go` path that no longer exists adds its directory to the packages to check, or the script falls back to `./...` when the directory has no Go files left. | `scripts/go-precheck.sh`, a test in `scripts/` |
| F16 | `test_dep_check_allowed_module_outside_wizard` (a non-test `llmprovider` file importing `golang.org/x/term`) and `test_parity_check_unexported_name` (a cell naming `catalog.SearchEverything`). Each is seen to pass when its gate's main check is disabled on a scratch copy, as today, and to fail with the check in place. | Add both tests. | `scripts/test_gates.py` |
| F55 | `TestRun_ForcedChoiceRefusalWithoutToolCall`: a provider that declares forced choice Unsupported and sends it anyway fails R11 even with no `Harness.ToolCall`. It passes today. | Run the refusal check before the `ToolCall` guard. | `llmtest/llmtest.go` |
| F56 | none: coverage. OpenCode's google and responses routes gain `llmtest.Run`, so the generateContent wire is checked again. | Add the two harnesses. | `providers/opencode/llmtest_test.go` |
| F57 | Each new harness check is first seen failing on the self-test's broken reference provider: <ul><li>R24, an HTTP failure that is not an `*APIError`;</li><li>403 to `ErrNotPermitted`;</li><li>`RetryAfter` on a 429;</li><li>`ListModels` identity and cancellation, where a lister exists;</li><li>R16's default header.</li></ul> The re-auth variant lands with F6, and the reasoning-only check with F7. | Add the checks. Every built-in must pass them; a built-in that fails is a deviation. | `llmtest/llmtest.go`, `llmtest_test.go` |
| F58 | `TestFake_ReplyNilRefused` and `TestFake_RequestsCopiesSchema`. They fail today. | `Reply(nil)` panics at the call with a clear message, or records `ErrIncomplete`; `Requests` deep-copies `Tool.Schema` by marshalling. | `llmtest/fake.go` |
| F59 | The ambient check is seen to flag a planted `os.UserConfigDir`, `os.UserCacheDir`, `user.Current` and `syscall.Getenv`. It does not today. | Add the four selectors. | `internal/ambientcheck` |
| F60 | none: drift. `make vuln` and the precheck run govulncheck at CI's pinned version, through `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0`, as `check_api.py` runs apidiff. | Pin it in one place. | `Makefile`, `scripts/go-precheck.sh` |
| F61 | A records self-test: a record placed under `docs/guides/` is reported, and counted by `--next`. It passes today. | Scan every `NNNN-KIND-*.md` under `docs/`, and report one outside `decisions/` and `reports/`. | `scripts/check_records.py`, `scripts/test_gates.py` |
| F62 | none: docs. The R2 table admits `internal/transport` for `internal/wire` and `wire/responses`, and the `<format>` row drops "more than one provider". `docs/architecture.md`'s "Depends on" column is regenerated from `go list`. | Correct both, and gain `filelock` from F13. | `docs/guides/api-standards.md`, `docs/architecture.md` |
| F63 | none: docs. README names the current release at release time, in Phase 7. | Correct it. | `README.md` |
| F64 | none: docs. architecture.md's G-wire counts are 18 cases and 114 files, or whatever Phase 3 leaves. | Correct it. | `docs/architecture.md` |
| F66 | `test_copy_tree_drops_staged_deletion` in `scripts/test_gates.py`. In a temporary clone with a tracked file's deletion staged, `copy_tree` makes a copy that still holds the file. It fails today: Phase 1's gate met it with `tokenstore_lock_owner_test.go`. | `copy_tree` removes every path in `git diff --name-only --diff-filter=D HEAD`, staged or not, as well as `git ls-files -d`. *(Added 2026-10-07, by the owner's "F66 add them"; MADR amendment of 2026-10-07.)* | `scripts/test_gates.py` |

### Phase 7: release (owner, then agent)

1. **The owner** commits and pushes Phase 6. CI is green on Linux, macOS and
   Windows.
2. **Release notes**, in this PLAN and the README's release paragraph:
   * the behaviour changes: F2's refusal of a foreign session, F11's
     refusal of unsendable inputs at `New`, and F7's `ErrIncomplete`;
   * the lock change of F13: `<provider>.oslock` ~~, and the legacy file kept~~
     *(Deviation D2: no legacy file)*
     for `v1.2.x` processes.
3. **The owner** tags `v1.3.0` on that commit and pushes it.
4. **The agent** checks:
   * CI on the tag;
   * `api-check` against `v1.2.1`, additive only;
   * the module proxy resolves `v1.3.0`;
   * a scratch consumer that requires `v1.3.0` builds for Linux, Windows
     and macOS.

## Verification

* **V1. Red, then green.** Every finding with a test has its FAIL line
  before the fix and its PASS after, in the Execution Record. Every
  documentation finding has its corrected text cited.
* **V2. The gate,** rule 3, is clean at the end of every phase.
* **V3. Credentials:**
  * F1's test passes for 1, 2 and 3 failed saves;
  * F3's for a cancelled leader and an abandoned waiter;
  * F13's for the takeover race and a killed holder, on macOS locally and
    on Linux and Windows in CI;
  * the 64-goroutine shared-store stress from the audit, kept as a test,
    sends no refresh token twice under `-race`.
* **V4. Live checks, with the owner's keys:**
  * the Gemini 429 capture (F12, Q6);
  * the Gemini 3 call-id round trip through OpenCode's google route (F65);
  * `gemini-2.5-flash-lite` reasoning (F38);
  * a `CommandToken` rerun against Gemini's real invalid-key reply (F6);
  * a two-process lock check on the owner's Windows host, if CI's cannot
    run two processes (F13).
* **V5. API.** `api-check` against `v1.2.1` reports only additive changes.
* **V6. Docs.** The parity gate, with F14's fix, passes 409 identifiers
  with no bare-name match. Every relative link resolves.
* **V7. Identifiers.** Nothing committed carries a hostname, an account
  name or a machine path.

## Rollout and Rollback

* **Rollout.**
  * Seven commits after the records, one per phase; the owner commits and
    pushes.
  * `v1.3.0` is tagged after Phase 6.
  * Consumers move under their own records.
* **Rollback.**
  * Before the tag, each phase reverts alone, newest first, with one
    exception: Phase 6's F57 checks assume Phases 3 and 4. Reverting
    either of those takes F57's matching check with it.
  * ~~Phase 1's lock change is safe to revert: the legacy lock file kept
    alongside the OS lock means a reverted process waits correctly on a
    store a `v1.3.0` process holds.~~ *(Deviation D2.)* Phase 1 reverts
    whole: a `v1.2.x` and a `v1.3.0` process sharing one store do not
    exclude each other, so a revert is a release, never a mix.
  * After the tag, fix forward in `v1.3.x`. A consumer that meets a
    defect pins `v1.2.1` again; nothing on disk needs changing, because the
    token file format does not change.

## Execution Record

The audits behind the MADR ran on scratch clones; nothing was committed
from them.

### Phase 0: records (2026-10-06)

* **Approval.** The owner answered Q1–Q7 ("Q1 file locks. Q2 yes Q3 yes Q4
  yes … Q6 yes Q7 yes", then "Q5 A, write the plan"), and approved this
  PLAN: "Proceed".
* The MADR is `accepted`, and this PLAN `in-progress`.
* `docs/README.md` indexes both, with those statuses.
* The records are staged for the owner's commit, alone. Phase 1's code is
  staged only after that commit (AGENTS.md, the bootstrap exception).

### Deviation D1 (2026-10-06): F2 judges a session with no `Provider` by its issuer

* **Found.** Phase 1's F2 step refuses a session "whose `Provider` is not"
  the provider's.
  * With that rule, 8 existing openai tests fail with `openai takes an API
    key, a ChatGPT sign-in or the Codex CLI's login, not a "" session`. For
    example: `chatgpt_test.go:292`, `chatgpt_listing_test.go:42` and
    `chatgpt_default_host_test.go:22`.
  * Their ChatGPT sessions carry the OpenAI issuer and no `Provider`, as do
    `openai/helpers_test.go:121`, `wire_test.go:44`, `listing_test.go:63`
    and the harness at `openai/llmtest_test.go:78`.
  * Every session the SDK builds sets `Provider`:
    * the browser login (`auth/oauth_loopback.go:658`);
    * the device logins;
    * the Kilo login;
    * the token store (`tokenstore_file.go:82`);
    * the wizard's pasted token.

    Only a hand-built session omits it, and a consumer may build one.
* **Decision.** The owner chose option 1, "identify by provider, falling
  back to issuer":
  * a session whose `Provider` is set, and is another provider's, is
    refused;
  * a session with no `Provider` is judged by its issuer: openai accepts
    only OpenAI's issuer, and grok only xAI's;
  * a session with neither is refused.

  Every case the audit reproduced stays refused, and no existing test
  changes. The MADR gains an amendment refining Q2.
* **Added to the phase's scope.** `(*auth.OAuthSession).Owner()` reads
  `Provider` and `Issuer` under the session's lock. A refresh's `adopt`
  rewrites both, so a provider must not read the fields directly. The
  method is additive under R48.

### Deviation D2 (2026-10-06): F13 is the OS lock alone, with no legacy lock file

* **Found.** F13's step said two things that cannot both hold:
  * "staleness, takeover and the heartbeat go";
  * "while it holds the OS lock, a process also creates the legacy `O_EXCL`
    lock file … A `v1.2.x` process sharing the store then still waits".

  A `v1.2.x` waiter judges the legacy file stale when its mtime is 30 s
  old (`lockStaleAfter`, `tokenstore_file.go:29`). A refresh can outlast
  that: three 15 s attempts, plus backoff. So without the heartbeat, a
  `v1.2.x` process takes over mid-refresh. Without takeover, a `v1.2.x`
  process that died holding the legacy file blocks every `v1.3.0` refresh
  for 40 s.
* **Options put to the owner:**
  1. keep the legacy protocol on the legacy file for one release;
  2. the OS lock alone;
  3. the legacy file without the heartbeat or takeover.
* **Decision.** The owner chose option 2: "This is essentially greenfield.
  Only prepare-commit-msg uses it. Wouldn't option 2 be the cleanest?"
  * Checked first: of the consumers on disk, only prepare-commit-msg uses
    `FileTokenStore` (`internal/config/config.go:106`), in its own
    directory. gobble-cli does not use it.
  * So a `v1.2.x` and a `v1.3.0` process share a store only while one
    hook binary replaces another, and a hook run lasts seconds.
* **What changes.**
  * `LockRefresh` takes the OS lock on `<provider>.oslock` alone.
  * The legacy `.lock` file, its owner token, staleness, takeover and
    heartbeat are removed, with their seams (`lockBeforeTakeover`,
    `lockRead`, `lockRename`) and the timing fields `staleAfter` and
    `heartbeat`.
  * The tests of the removed mechanism go:
    * `tokenstore_lock_owner_test.go`'s five tests;
    * `TestFileTokenStore_RefreshLock_StaleTakenOver`.

    `TestFileTokenStore_LockReleasedWhenHolderDies` and a mutual-exclusion
    stress test replace them.
  * The rollback note and the release notes are corrected above.
  * The MADR gains an amendment refining Q1.

### Phase 1: credentials (2026-10-06)

* **Approval.** "Proceed", 2026-10-06, with D1 ("Option 1") and D2 ("Wouldn't
  option 2 be the cleanest?") chosen on the way.
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code, except where a plant on a scratch copy is named.

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F1 | `TestOAuthSession_RepeatedFailedSavesKeepRotation` (1, 2, 3 failed saves; 2, then a resave) | `two_failed_saves` and `three_failed_saves`: `refresh 3: … refresh_token_reused; want access-3`. The resave case, on a scratch copy with the old overwrite rule planted: `after the resave the store holds "old-refresh", want "refresh-2"`. The one-save case passes before and after: 0020 F1 holds. | The session's `storedRefresh` is the refresh token the store holds. `Load` and every successful save set it, and an adopted stored session sets it too. A failed save sets it only when nothing is pending, so it stays the store's token. `loadRotated` and `persistRotation` compare against it. `spentRefresh` and `state.spent` are gone. |
| F3 | `TestOAuthSession_CallerCancelKeepsRotation`, `TestOAuthSession_AbandonedWaiterDoesNotResendSpent` | `Token after the cancelled one = "", … refresh_token_reused`; `refresh tokens sent = [old-refresh old-refresh]`. | `Token` starts the refresh in `complete`, a goroutine under `context.WithoutCancel(ctx)` within the 15 s attempt bound. Every caller and waiter waits on the future or on its own ctx (`waitToken`). A refresh is adopted and saved when it finishes, whoever is still waiting. `tokenFuture.abandoned` and the re-run are gone. A Token whose ctx is already done starts nothing. |
| F13 | `TestFileTokenStore_LockReleasedWhenHolderDies` (a child process takes the lock and is killed), `TestFileTokenStore_RefreshLockExcludes`, `TestOAuthSession_SharedStoreStress` (V3), and in the new package `TestTryLock_ExcludesAnotherHandle`, `TestTryLock_ClosedFileFails`, `TestTryLock_ReleasedWhenHolderDies` | `LockRefresh after the holder died = … another process is refreshing the openai session after 2.03s`. On a scratch copy with `TryLock` planted out: `at most 16 holders at once, want 1`, and the stress test's goroutines get `refresh_token_reused`. | `llmprovider/internal/filelock`: `flock` on Unix; `LockFileEx` on Windows, through `//sys` lines generated by `mkwinsyscall` into `zsyscall_windows.go`. `LockRefresh` holds it on `<provider>.oslock`, never deleted. The legacy lock, its owner token, staleness, takeover, heartbeat, seams and timing fields are removed (D2). A failure to open or lock the file is `ErrProviderUnavailable` (F43). |
| F2 | `TestNew_RefusesAForeignSession` (grok, openai), `TestNew_AcceptsItsOwnSessionWithoutProvider` (grok), `TestIsChatGPTSession_NoIssuer`, `TestOAuthSession_Owner`, `TestOAuthSession_ChatGPTWithoutIssuer` | grok and openai: every foreign session `New … <nil>; want ErrUnsupported`; `no issuer: isChatGPTSession = false, want true`. | D1: `(*OAuthSession).Owner()`, then `grok.New` and `openai.New` refuse a session another provider owns, and a `VendorCLISession` for another CLI, with `ErrUnsupported`. `ChatGPT()` counts an OpenAI session with no issuer. |
| F40 | `TestVendorCLISession_SymlinkedAuthFile`, `TestVendorCLISession_BrokenSymlink` | `read the Grok CLI login: openat auth.json: path escapes from parent; run grok to refresh it, or grok login` for both. | `readVendorAuthFile` resolves symlinks, then opens the target inside its own directory with `os.Root`. A dangling link is `errVendorBrokenLink`, reported without the login advice. |
| F41 | `TestVendorCLISession_TornFileRetriedOnce` | `decode: unexpected end of JSON input; … after 1 reads; want the whole file on the second read`. | A file that does not parse is read again once, after `vendorRetryDelay` (50 ms), through the `vendorReadFile` seam. |
| F42 | `TestFileTokenStore_SaveWhileOpenForRead` (Windows only) | Run on the owner's Windows host (D3): before F42, `FileTokenStore rename: rename .tok-N.json openai.json: Access is denied.` | `readBounded` opens through `openShared`: `os.Open` on Unix, and `CreateFile` with `FILE_SHARE_DELETE` on Windows. `Save` renames through `renameInDir`, an `os.Root` rename: POSIX semantics on Windows, `renameat` on Unix (D3). |
| F43 | `TestOAuthRefresh_EveryFailureHasAKind`, `TestGrokDevice_DeniedAndExpiredAreAuthFailures`, `TestKiloDevice_DeniedAndExpiredAreAuthFailures` | `oauth: refresh request: … connection refused, want ErrProviderUnavailable`; `decode refresh response: unexpected end of JSON input, want ErrIncomplete`; `refresh response missing access token, want ErrIncomplete`; Grok and Kilo `… denied` and `… expired`, `want ErrAuthFailure`. | The refresh's transport and read failures are `ErrProviderUnavailable`. An oversized, undecodable or token-less 200 is `ErrIncomplete`. Grok's `access_denied` and `expired_token`, and Kilo's 403 and 410, are `ErrAuthFailure`. The lock's failures are kinded in F13. |
| F44 | `TestValidateOAuthSession_RejectsFixture/custom_issuer_without_token_URL` (and an older OpenAI session without an issuer, valid) | `ValidateOAuthSession() = <nil>, want valid=false`. | `ValidateOAuthSession` asks `refreshTokenURL` whether the session can refresh, and its doc comment says so. |
| F45 | `TestRevokeOAuthSession_LegacyGrokSession` | `hosts [ 127.0.0.1:…] … want discovery at auth.x.ai`: discovery ran on an empty issuer. | `grokRevokeRequest` uses xAI's issuer for a Grok session with none, as `refreshTokenURL` does. |

* **Tests changed, not new:**
  * `TestOAuthSession_WaiterOutlivesLeaderCancel` (0020 F14) relied on the
    leader's cancel aborting its request. Under F3 the refresh is
    detached. The test keeps its assertion (the waiter gets the fresh
    token) and now also requires a single issuer call.
  * `TestOAuthSession_PendingResaveDoesNotBlock` sets `storedRefresh`, not
    `spentRefresh`.
  * `TestFileTokenStore_RefreshLock_HeldTimesOutRetryable` waits its whole
    `wait`, with no stale window.
  * The stalled-reply refresh test checks that the lock can be taken, not
    that a `.lock` file is gone.
* **Tests removed with the old lock (D2):**
  * `TestFileTokenStore_RefreshLock_StaleTakenOver`;
  * the five tests of `tokenstore_lock_owner_test.go`:
    * `StaleLockHasOneTaker`;
    * `UnlockKeepsSuccessorsLock`;
    * `SkewedHeartbeatKeepsLock`;
    * `HeartbeatSurvivesReadError`;
    * `TakeoverRenameRetried`.
* **The gate's self-test.** `test_generate_check_no_output` stripped
  `-output` from `ownerperm`'s directive only. With `filelock`'s second
  generated file, the gate counted 1, and the self-test failed with
  `generate-check exit 0, want 1: … 1 generated file(s)`. It now strips
  every package's directive.
* **Docs.**
  * `docs/architecture.md` gains `filelock`:
    * the tree;
    * the package table;
    * `auth`'s imports;
    * the layering;
    * `generate-check`'s list.

    Its `FileTokenStore` paragraph describes the OS lock.
  * `docs/guides/api-standards.md` R2 gains the package and `auth`'s import.

**Proofs** (a scratch copy; `p26_phase1_plants.py`). The control passed;
each plant failed:

| Planted | Failed |
| :--- | :--- |
| a failed save always overwrites `storedRefresh` (the old rule) | `RepeatedFailedSavesKeepRotation`: two and three failed saves, and the resave case |
| the refresh runs under the caller's ctx | `CallerCancelKeepsRotation`, `AbandonedWaiterDoesNotResendSpent`: `context canceled` |
| `LockRefresh` takes no lock | `RefreshLockExcludes`: 16 holders at once; `SharedStoreStress`: `refresh_token_reused` |

**The gate,** on a scratch copy with the phase staged and committed there
with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `413 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems |
| `coverage-check` | 28 packages, 0 problems; `auth` 87.5 %, `filelock` 90.0 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)`; `Owner` is the one addition |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 602 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 39 files |

**Found, and outside this phase (not done):**

* **`scripts/test_gates.py`'s `copy_tree`** clones `HEAD` and lays the
  working tree over it, but drops only files deleted from the working
  tree, not a deletion already staged. So with a staged deletion,
  `make gate-selftest` builds a copy that still holds the deleted file, and
  fails. It failed here on `tokenstore_lock_owner_test.go`. CI runs on a
  commit, so it never sees this.
  * It is a new tooling finding.
  * This phase's gate ran on a copy where the staged state was committed
    with plumbing.
  * The fix waits for an amendment adding it to Phase 6, and the owner's
    approval.
* ~~**F42's red** waits for a Windows run, as the PLAN's F42 row says.~~
  *(Done 2026-10-07; deviation D3.)*

### Deviation D3 (2026-10-07): F42 also needs an `os.Root` rename

* **Found,** on the owner's Windows host over SSH, with test binaries built
  here for windows/amd64. Phase 1's F42 fix was the reader alone: token
  files opened with `FILE_SHARE_DELETE`. With it, the test still failed,
  as before the fix: `FileTokenStore rename: rename .tok-N.json
  openai.json: Access is denied.` The PLAN's premise was wrong:
  `os.Rename` (`windows.Rename`) is refused while any reader holds the
  target, even one that shares delete access.
* **Measured,** each in its own scratch build:

  | Reader | Rename | Result |
  | :--- | :--- | :--- |
  | `os.Open` | `os.Rename` | Access is denied |
  | `FILE_SHARE_DELETE` | `os.Rename` | Access is denied |
  | `FILE_SHARE_DELETE` | `os.Root.Rename` | pass |
  | `os.Open` | `os.Root.Rename` | `renameat … Access is denied` |

  * `os.Root.Rename` is Go's `Renameat`
    (`internal/syscall/windows/at_windows.go:363-430`). It renames with
    `FILE_RENAME_POSIX_SEMANTICS | FILE_RENAME_REPLACE_IF_EXISTS`, and falls
    back to a plain replace on a filesystem without POSIX semantics, such
    as FAT.
  * Go's own toolchain retries renames on Windows for 2 s instead
    (`cmd/internal/robustio`).
* **Options put to the owner:**
  1. both halves;
  2. a 2 s rename retry;
  3. both of those;
  4. record F42 as not fixed.
* **Decision.** The owner chose option 1:
  * readers keep `FILE_SHARE_DELETE`;
  * `Save`'s rename is `renameInDir`, an `os.Root` rename;
  * the two tests that reset the `tokenStoreRename` seam to `os.Rename`
    (`oauth_rotation_test.go`, `tokenstore_durable_test.go`) now restore
    the previous value.
* **Proved on the owner's Windows host:**

  | Build | `TestFileTokenStore_SaveWhileOpenForRead` |
  | :--- | :--- |
  | before F42 (both reverted) | FAIL: `rename .tok-N.json openai.json: Access is denied.` |
  | the reader reverted to `os.Open` | FAIL: `renameat .tok-N.json openai.json: Access is denied.` |
  | the rename reverted to `os.Rename` | FAIL: `rename .tok-N.json openai.json: Access is denied.` |
  | fixed | PASS |

  * The fixed build's whole `auth` suite passes on Windows, with 0
    failures. That covers the OS lock, the shared-store stress test and the
    rotation tests.
  * `filelock`'s suite passes on `LockFileEx`.
* **Limit, recorded:** a consumer that reads the token file itself with
  plain `os.Open` still blocks a save on Windows. The SDK cannot prevent
  that.
* The MADR gains an amendment, since F42's fix is a decision it describes.
* **The gate, rerun after D3,** on a scratch copy with the phase committed
  there with plumbing: every check rc=0.
  * `make pre-add-check`: `413 file(s) clean`;
  * `coverage-check`: 28 packages, 0 problems, `auth` 87.4 %;
  * `api-check`: 0 incompatible changes;
  * `generate-check`: 2 generated files;
  * `gate-selftest`: OK;
  * links: 602 in 67 files;
  * identifier scan: 0 hits in 39 files.

### Deviation D4 (2026-10-07): F12's capture needs a free-tier key

* **Found,** running F12's live capture,
  `TestLive_GeminiRateLimitShape` (`llmprovider/live_gemini_429_test.go`).
  It is switched on by its own variable, `LLMPROVIDER_LIVE_GEMINI_429`,
  since it exhausts a rate limit on purpose. It sends up to 20 one-token
  requests at once and stops at the first 429.
  * To `gemini-2.5-pro`, every request was answered `HTTP 404`: the model
    "is no longer available to new users".
  * To `gemini-pro-latest`, an alias that follows the current pro model,
    every request was answered 200. The key's limit is above the burst, as
    a paid tier's is, so the PLAN's step could not produce a 429 cheaply.
  * The test now fails, rather than skips, when a request is answered with
    a status other than 200 or 429: the model is unusable, not unlimited.
* **Options put to the owner:**
  1. run the capture once with a free-tier key, which reaches 429 on a pro
     model within a few requests, at no cost;
  2. a larger burst on the paid key, up to 300 requests;
  3. implement from Google's documented shape, which reverses Q6 (a) and
     needs a MADR amendment.
* **Decision.** The owner chose option 1. Q6 (a) stands: the capture is
  still live, with a key of the owner's. The owner runs the test; the
  fixture is taken from its logged body, with any project number replaced
  by a placeholder. The F12 fix and `TestClassify_GeminiRetryInfo` follow
  the capture.
* **Revisited, the same day.** The owner's exported `GEMINI_API_KEY`, the
  only key offered, was run through the test: 20 requests, all 200, and a
  skip ("no 429 after 20 requests"). It is the same key as before, not a
  free-tier one, so option 1 could not be met.
  * **Options put to the owner again:** a free-tier key under a second
    variable; a burst of up to 300 on this key; implement from Google's
    documented shape.
  * **Decision.** The owner chose the last ("Option 3"). Q6 is now (b), and
    the MADR's amendment "Q6 is (b), F12 from Google's documented shape"
    records it. `TestClassify_GeminiRetryInfo`'s fixture follows
    `google/rpc/error_details.proto`; the live test stays, opt-in, to check
    it against a real reply.

### Deviation D5 (2026-10-07): F67 joins Phase 2

* **Found** while fixing F23. A probe classified a 400 sent with
  `x-should-retry: true` on openai, claude and kilo; each gave
  `invalid request: <provider> HTTP 400: m retryable=true invalid=true`.
  The kind itself is `ErrInvalidRequest`, so F23's fix, which leaves out a
  retryable error's legacy sentinel, does not reach it. The probe was a
  throwaway test file, deleted once run.
* **Options put to the owner:**
  1. add it to Phase 2 as F67, and make such an error
     `ErrProviderUnavailable`, as a retryable 408 or 409 is;
  2. ignore `x-should-retry: true` on an invalid-request status, against
     the OpenAI and Anthropic SDKs;
  3. record it for a later phase.
* **Decision.** The owner chose option 1 ("Decision 2, a"). The MADR's
  amendment of 2026-10-07 adds F67, and Phase 2's table gains its row.

### Phase 2: errors and retry (2026-10-07)

* **Before it.** Phase 1 was committed as `ecee48a` and pushed on the
  owner's word ("Push to main"). Its CI run, 37578319181, passed on
  Ubuntu, Windows and macOS.
* **Approval.** "Proceed", 2026-10-07, with D4 ("Option a", then "Option
  3") and D5 ("Decision 2, a") chosen on the way.
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code. Test names differ from the table above where one
  test became several.

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F4 | `TestPost_StalledErrorBodyEndsAtIdleLimit` | `a stalled 503 body took 5.002397417s; want about the idle limit, 100 ms`: it ended only at the test's own 5 s deadline. | `Post` wraps `resp.Body` in the idle-limited `ReplyReader` before `ClassifyHTTPError` reads it. |
| F5 | `TestDecodeFor_ErrorIn200Classified` (400 overflow, 401, 402, 403 on Kilo, 429, a 503 inside a choice, OpenCode's `FreeUsageLimitError`); `TestKilo_ErrorIn200IsNotRetried` | `kilo failed after 3 attempts: llmprovider: provider unavailable: chat completions stream 400`, and each case had the wrong kind. | A numeric `error.code` from 400 to 599 is classified by `classifyAPIError` for the provider's service. `chatcompletions.DecodeFor(provider)` passes the label; Kilo, Together, Hugging Face, Ollama and OpenCode use it. |
| F11 | `TestResolveOptions_RefusesUnsendable`, `TestRetryable_UnsendableIsTerminal`, `TestPost_UnsendableIsInvalidRequest` | All 10 refused `ResolveOptions` cases returned `<nil>`, and three unsendable failures were `retryable = true`. | `settings.sendable()`: a base URL must parse with a scheme and a host, and a static key, the client info and the session id must hold no control character. `transport.Unsendable` recognises a `*url.Error` no retry can fix; `retryable` treats it as terminal, and `Post` returns it as `ErrInvalidRequest`. |
| F12 | `TestClassify_GeminiRetryInfo` (per minute, fractional, per day); `TestClassify_GeminiRetryInfoEdges` | `per minute: RetryAfter=0s … retryable=true`; `per day: RetryAfter=0s kind ok=false retryable=true`. The edges test passes before and after; it guards a header winning over the body, a bare 429, malformed delays, and another service. | D4, Q6 (b): the envelope reads Google's `details[]` `@type`, `retryDelay` and `violations[].quotaId`. `protoDuration` parses the delay, which sets `RetryAfter` when no header did; a Gemini 429 naming a `PerDay` quota is terminal `ErrQuotaExhausted`. `TestLive_GeminiRateLimitShape` stays, opt-in. |
| F17 | `TestRedact_PrefixedKeys` | `openai_api_key`, `"x_api_key"`, `app_secret`, `"apiSecret"`, `db_password` and `"subscription_key"`: `the value is not redacted`. | `reKV` allows a snake, kebab or camelCase prefix before each secret name, and takes any prefixed `…_key`. `max_tokens` and `prompt_token_count` stay unredacted. |
| F18 | `TestEmptyAnswer_ReasonIsBounded`, `TestIncomplete_ReasonIsBounded`, `TestOAuthHTTPStatusError_StripsControl`; `TestField`, added after the gate's first run (below) | `ESC=true BEL=true and is 2125 bytes`; `… 2087 bytes`; the token endpoint's error held `\x1b]0;owned\a\x1b[2J`. | `redact.Field` strips control characters and bounds to `redact.FieldLimit`, 128 bytes; `boundCode`, `wire.EmptyAnswer` and `responses.incomplete` use it. `oauthHTTPStatusError` strips the status and the body before redacting. |
| F19 | `TestResolveOptions_RefusesUnsendable` (an unknown effort, a negative budget) | Among the 10 `<nil>` results above. | `(*Reasoning).check()`, shared by `Request.validate` and `settings.sendable`. |
| F20 | `TestValidate_UnmarshalableSchema`, `TestPost_UnmarshalableBodyIsInvalidRequest` | `a channel`, `invalid raw JSON`, `an unsupported value`: `Check = <nil>`; `marshal request: json: unsupported type: chan int; want ErrInvalidRequest`. | `validate` marshals each tool's schema; `Post`'s marshal failure is `ErrInvalidRequest`. |
| F21 | `TestRedact_KeepsOrdinaryWords` | `"Invalid bearer token"` became `"Invalid [REDACTED]"`, and the same for the three other sentences. | `redactAuth`: after an `Authorization` label the value is always redacted; without one, only a credential-shaped value is (a digit, token punctuation, a capital after the first letter, or 20 characters or more; a final full stop does not count). |
| F22 | `TestRetry_ServerWaitNeverExceedsMaxDelay` | `asked 90ms: longest wait over 50 runs = 326.792386ms`; `asked 100ms: … 349.10718ms`. | The jittered server wait is capped at `MaxDelay`. |
| F23 | `TestAPIError_RetryableDoesNotMatchInvalidRequest` | openai 408, openai 409, claude 409: `Retryable=true and matches ErrInvalidRequest`. | `Unwrap` leaves out the legacy status sentinel when the error is `Retryable()`. 0012-MADR gains its amendment of 2026-10-07. |
| F24 | `TestPost_Accepts2xx` | `a 201: "", llmprovider: invalid request: p HTTP 201`. | `ClassifyHTTPError` returns nil for any 2xx. |
| F67 | `TestAPIError_ShouldRetryIsNotInvalidRequest` | openai, claude, kilo: `invalid request: <provider> HTTP 400: m; want retryable ErrProviderUnavailable`. | `x-should-retry: true` on a kind that matches `ErrInvalidRequest` sets the kind to `ErrProviderUnavailable`. |

* **F21, a note on the fix.** The table above says the credential is "at
  least 8 long". Applied after an `Authorization` label, that would have
  broken the pinned case `Authorization: Bearer abc12`
  (`redact_providers_test.go`). The existing minimums, 4 after `bearer` or
  `basic` and 8 after `token`, are kept, and the shape test is added. This
  redacts more than the table's words ask, never less.
* **Tests changed, not new:**
  * `TestClassifyHTTPError_Table` checked that every row still matches its
    pre-0012 sentinel. Under F23 that holds for a terminal error; a
    retryable one must not match `ErrInvalidRequest`. Its 408 row is the
    case.
  * `TestClassifyHTTPError_ShouldRetryTrue` (0021 T14) wanted "a retryable
    `ErrInvalidRequest`". Under F67 it wants a retryable
    `ErrProviderUnavailable`, and still checks that `WithRetry` retries it.
    0021 says only that the header makes a reply retryable.
* **Renamed for lint:** `chatcompletions.decode` is `decodeAs`, since
  `revive`'s `confusing-naming` refuses a name that differs from `Decode`
  only by case.
* **Docs.**
  * 0012-MADR: the amendment of 2026-10-07 (F23, F67).
  * `docs/guides/migrating-from-mcplib.md`: the `APIError.Unwrap` row (F23,
    F67) and the `APIError.RetryAfter` row (F12).

**Proofs** of the changed tests (scratch copies; `plant_f23.py`,
`plant_f67.py`). Each plant failed, and the restored code passed:

| Planted | Failed |
| :--- | :--- |
| `Unwrap` as before F23 | `TestClassifyHTTPError_Table`: `huggingface HTTP 408: timeout is retryable and matches llmprovider: invalid request` |
| `Unwrap` with no legacy sentinel at all | `TestClassifyHTTPError_Table`: the OpenCode 401 rows `no longer match the pre-0012 sentinel … authentication failed` |
| the classification before F67 | `TestClassifyHTTPError_ShouldRetryTrue` and `TestAPIError_ShouldRetryIsNotInvalidRequest` |
| `redact.Field` cutting inside a rune (`plant_field.py`) | `TestField`: `Field across a rune: 128 bytes, valid UTF-8 false; want 127 bytes, valid` |
| `redact.Field` not stripping control characters | `TestField`: `Field(16 bytes) = "\x1b]0;owned\aSAFETY"; want "]0;ownedSAFETY"` |

**The gate's first run failed** `coverage-check` and `gate-selftest`, with
one cause: `internal/redact is 90.3%, below its 100.0% floor`. `Field` was
tested only from the wire and `llmprovider` packages, and a package's
floor counts its own tests. `TestField` (`field_0026_test.go`) covers it,
the plants above prove it, and the package is at 100.0 % again. The floor
is unchanged.

**The gate, rerun,** on a scratch copy with the phase staged and committed
there with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `428 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems |
| `coverage-check` | 28 packages, 0 problems; `internal/redact` 100.0 %, `llmprovider` 97.7 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)` |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 605 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 37 files |

* After the gate, this record gained its gate results and its `TestField`
  entries. Records, links and the identifier scan were rerun on the final
  tree: 0 problems each. The records sit outside markdownlint's scope;
  against `HEAD` they gain only MD004 (`*` list markers, the records'
  style) and nothing else.

**Not done in this phase:**

* **F12 is checked against Google's documented shape, not a real reply**
  (D4). `TestLive_GeminiRateLimitShape` checks it whenever it is run with a
  key that reaches its limit.
* **A live run of the other Phase 2 changes** was not part of this phase:
  each is covered by the unit tests above, and Phase 7's release gate runs
  the live suites.

### Deviation D6 (2026-10-07): F65 is measured on Google's generateContent

* **Found,** running F65's live test,
  `TestLive_OpencodeGoogleTwoCallRoundTrip`. It picked `gemini-3.8-flash`
  on OpenCode Zen's google route, and Zen refused the first call:
  `quota exhausted: opencode-zen/google HTTP 402 UNKNOWN: Upstream request
  failed: Insufficient account funds`. models.dev lists no Gemini model on
  OpenCode Go. Zen's google route is the only way the SDK reaches the
  generateContent wire, so the step as written could not run.
* **Options put to the owner:**
  1. measure on Google's generateContent API, with `GEMINI_API_KEY`, through
     the same encoder and decoder;
  2. fund the Zen account, and run the test as written;
  3. record F65 as unmeasured.
* **Decision.** The owner chose option 1 ("D1: A"). The test builds the
  OpenCode provider with the google route pinned, Google's base URL
  (`https://generativelanguage.googleapis.com/v1beta`) and the Gemini key.
  The request is the route's own: `/models/{model}:generateContent`, the
  key in `x-goog-api-key`, the body from `generatecontent.Contents`.
  * **Not measured:** OpenCode's proxy in front of Google, and so whether
    it passes ids through.
  * The MADR gains an amendment, since F65's evidence names the route.

### Deviation D7 (2026-10-07): F28 wires three providers to `responses.DecodeFor`

* **Found** while fixing F28. A failed status is classified by the
  provider's error vocabulary, as the stream's is: OpenAI's
  `insufficient_quota` is `ErrQuotaExhausted` only for the service
  `openai`. Under `Decode`'s label, "responses", it would be retryable.
  F28's row names only `internal/wire/responses`.
* **Options put to the owner:**
  1. Grok, OpenAI with a key, and OpenCode's responses route decode with
     `responses.DecodeFor(<their label>)`, as F5 did for Chat Completions;
  2. the providers keep `Decode`, with the generic classification.
* **Decision.** The owner chose option 1 ("D2: A"). F28's files gain
  `providers/grok/grok.go`, `providers/openai/openai.go` and
  `providers/opencode/opencode.go`.

### Phase 3: answers on the wire (2026-10-07)

* **Before it.** Phase 2 was committed as `faf06a8` and pushed on the
  owner's word ("push to main"). Its CI run, 37637638536, passed on Ubuntu,
  macOS and Windows.
* **Approval.** "proceed", 2026-10-07, in the same message as the push,
  with D6 ("D1: A") and D7 ("D2: A") chosen on the way.
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code (`p26_phase3_red.out`). For F28, `DecodeFor` was first
  added as a plain wrapper over `Decode`, so that the test compiled and
  failed on behaviour.

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F7 | `TestDecode_ReasoningOnlyCutIsIncomplete` in `chatcompletions`, `messages` and `generatecontent`; `llmtest`'s `F7-reasoning-cut` check in every provider's harness | chat and messages: `Output:[{Text:thinking about it …}] FinishReason:length`, a success; generatecontent: `err = <nil>; want an ErrIncomplete *APIError` | With a `length` finish and no text or call, each decoder returns `wire.EmptyAnswer(where, FinishLength)`. A cut text answer keeps its text. `llmtest.Harness.ReasoningCut` switches on the check; every built-in provider sets it. |
| F8 | `TestFromItems_CallIDsFitAnthropicRule`; `TestRemapCallIDs_Edges` | `tool_use id "get_weather#0" breaks Anthropic's rule`, `"get_weather#1"`, `"get_weather#0"` again, and the 100-character id; `want 5 distinct` | Q5 (A): `wire.RemapCallIDs` with `wire.AnthropicCallIDs`. Refused characters become `_`, the id is cut to 64, and `_2`, `_3`… tell collisions apart. Each result takes its call's new id. `messages.FromItems` applies it. The caller's items are not changed, and no golden moved. |
| F25 | `TestGenerateContent_BlockReason` | `Reason = ""; want SAFETY` | `promptFeedback.blockReason` is the empty answer's `Reason`. |
| F26 | `TestGenerateContent_ToolCallFailureReasons` | `Reason = "stop"; want MALFORMED_FUNCTION_CALL`, the same for `UNEXPECTED_TOOL_CALL` | Both are removed from `finishReasons`, so they are kept as sent (0021 W3). |
| F27 | `TestToolSchema_TypedNil` | `nil map: null`; `nil RawMessage: null` | `ToolSchema` treats a typed nil, and a `RawMessage` of `null` or nothing, as no schema. |
| F28 | `TestResponses_FailedStatusClassified`; `TestResponsesProviders_FailedStatusUsesTheirVocabulary` (D7) | Every case: `incomplete response: stop: responses: the answer has no content`. Through the providers: `openai quota: provider unavailable: responses stream insufficient_quota`, and no provider named. | `decodeAs` reads the status first. `failed` classifies a failed response, streamed or not. `responses.DecodeFor(label)`, which Grok, OpenAI with a key, and OpenCode's responses route now use (D7). |
| F29 | `TestChatCompletions_Refusal` | `incomplete response: stop: chat completions: the answer has no content` | `message.refusal` is the answer's text, with `FinishContentFilter`, when there is no content. |
| F30 | `TestChatCompletions_ContentParts` | `cannot unmarshal array into … content of type string` | `chatContent` takes a string, null, or an array of parts whose text parts are joined; anything else is a decode error. |
| F31 | `TestResponses_EmptyArguments` | `Decode: arguments ""`; `ReadStream: arguments ""` | `appendOutput` gives empty arguments as `{}`, on both paths. |
| F32 | `TestChatCompletions_ReasoningNotCarriedAcrossTurns` | `reasoning_content = "turn-1 reasoningturn-2 reasoning"`, and turn 1's `reasoning_details` on turn 2 | Pending reasoning, and its details, are dropped at a user turn. |
| F65 | `TestLive_OpencodeGoogleTwoCallRoundTrip` (live, D6) | Not a red test: a measurement. | None needed. See below. |

* **F65, measured** (`p26_f65_live.out`). Three runs of `google-direct`,
  on `gemini-3.8-flash` through the google route's own encoder and decoder,
  against Google's generateContent API:

  | Run | Call ids Gemini sent | Turn 2's reply |
  | :--- | :--- | :--- |
  | 1 | `call_510369`, `call_510370` | `Oslo=OSLO-7731; Rome=ROME-4412` |
  | 2 | `call_366915`, `call_366916` | `Oslo=OSLO-7731; Rome=ROME-4412` |
  | 3 | `call_366514`, `call_366515` | `Oslo=OSLO-7731; Rome=ROME-4412` |

  Gemini 3 sends `functionCall.id`, and accepts two results to one function
  sent by name and position, pairing them correctly. Under the PLAN's F65
  row, F65 is recorded as measured, and closed, with no code change. The
  `zen` subtest skipped each time: `HTTP 402 … Insufficient account funds`.
* **Docs.**
  * `docs/architecture.md`: the package table gains `RemapCallIDs`,
    `responses.DecodeFor`, `chatcompletions.DecodeFor` and `redact.Field`.
    It also gains imports the code already had: `internal/redact` for
    `internal/wire` and `responses` (Phase 2's F18), and
    `internal/transport` for `responses`. `llmtest`'s list of optional
    checks gains `ReasoningCut`, and "Answers" covers the reasoning cut and
    call ids.
  * `docs/guides/adding-a-provider.md`: the optional checks gain
    `ReasoningCut`.
  * `README.md`: an answer cut while reasoning is `ErrIncomplete`.

**Proofs** (a scratch copy; `plant_f7.py`). With F7 reverted in the three
decoders, the `F7-reasoning-cut` check failed for Claude, Hugging Face,
Kilo, Ollama, Together and both OpenCode harnesses, and the three decoder
tests failed. In `llmtest`'s own tests, the reference provider's
`keepReasoningCut` flaw fails the check, as each W12 flaw fails its own.

**The gate,** on a scratch copy with the phase staged and committed there
with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `438 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems |
| `coverage-check` | 28 packages, 0 problems; `internal/wire` 96.0 %, `chatcompletions` 93.1 %, `generatecontent` 94.2 %, `messages` 96.7 %, `responses` 97.8 %, `llmtest` 95.9 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)`; the new exported field, `llmtest.Harness.ReasoningCut`, is compatible |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 605 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 34 files |

**Not done in this phase:**

* **F65 through OpenCode's proxy.** The `zen` subtest stays; it measures
  the route through Zen once the account is funded (D6).
* **A live run of the other Phase 3 changes** was not part of this phase.
  Each is covered by the unit and conformance tests above; Phase 7's
  release gate runs the live suites.

### Deviation D8 (2026-10-07): F37 adds two `Settings` methods

* **Found** while fixing F37. Its fix has `New` pass the client and logger
  to a shared session "only when the caller gave them", and no provider can
  tell. `Settings.HTTPClient()` returns `transport.DefaultClient()`'s
  client as it returns a caller's, and that function makes a fresh client
  on each call. `Settings.Logger()` returns a new discard logger. F37's row
  names `providers/openai`, `providers/grok` and `auth/oauth_session.go`
  only.
* **Options put to the owner:**
  1. `llmprovider.Settings` reports whether `WithHTTPClient` and
     `WithLogger` were given;
  2. `internal/transport` remembers the default clients it makes, and the
     logger is compared with `slog.DiscardHandler`;
  3. record F37 as accepted, and document that the first provider fixes a
     session's client and logger.
* **Decision.** The owner chose option 1 ("go with option a").
  `(*Settings).HTTPClientGiven` and `(*Settings).LoggerGiven` are added;
  OpenAI and Grok give a session the client and logger only when they
  report true. F37's files gain `llmprovider/settings.go`. The MADR gains
  an amendment, since the change adds to the exported API.
* **Revisited, the same day.** Built that way, the pre-existing
  `TestProviderClient_SharedWithListingAndRefresh` (0016-PLAN T1 step 3)
  failed in OpenAI and Grok: `POST /oauth/token did not go through the
  provider's client (it carried [POST /responses GET /models])`. 0016-MADR
  D8 has a provider built without `WithHTTPClient` refresh its session
  through its own default client, and a provider that passes no default
  breaks that.
  * **Options put to the owner:** a default client passed as a default,
    which a caller's replaces; amend 0016 D8; record F37's client half as
    accepted.
  * **Decision.** The owner chose the first ("option 1"). The provider
    gives a session its default client with a new
    `(*auth.OAuthSession).UseDefaultHTTPClient`, which sets it only when the
    session has none, and marks it a default. `UseHTTPClient`, called with a
    caller's client, replaces a default, never another caller's. A client
    set on the session's field by the caller counts as the caller's. The
    logger keeps the rule above: a discarding default is never shared.
    F37's files gain nothing more.

### Phase 4: provider wiring (2026-10-07)

* **Before it.** Phase 3 was committed as `8a428de` and pushed on the
  owner's word ("commit it to main, then push"). Its CI run, 37642295233,
  passed on Ubuntu, Windows and macOS.
* **Approval.** "proceed", 2026-10-07, with D8 ("option a", then "option
  1") chosen on the way.
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code (`p26_phase4_red.out`). For `UseDefaultHTTPClient`,
  the method was first added with `UseHTTPClient`'s behaviour, so the test
  compiled and failed on behaviour.

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F6 | `TestReauth_RerunsOnAuthFailureKind`; `llmtest`'s R16 check with `Harness.AuthFailure`, which Gemini's harness sets to its 400 `API_KEY_INVALID`; `TestRun_AuthFailureIsTheServicesRefusal` | `gemini 400 API_KEY_INVALID` and `403 auth failure`: `sends = 1, invalidations = 0; want 2 and 1`; `401 quota exhausted`: `sends = 2, invalidations = 1; want 1 and 0` | Q3 (a): `credentialRefused` reruns on an `*APIError` whose `Kind` is `ErrAuthFailure`, compared, not matched, whatever its status; one with no kind, on a 401. A 401 of kind `ErrQuotaExhausted`, such as OpenCode's `CreditsError`, is no longer rerun: a new key does not restore credits. |
| F36 | `TestList_RerunsCredentialOn401` (renewed; still refused; a static key) | `Live=false requests=1 invalidations=0 Err=together: models endpoint returned HTTP 401` | `List` reruns once by the same rule, when the source can be invalidated; a listing still refused degrades to the static catalog, with the refusal in `Catalog.Err`. Every lister's non-200 reply is now `ClassifyHTTPError`'s, so the rule can see it. `catalog` keeps its own copy of the rule, since it does not import `internal/wire`. |
| F37 | `TestSession_LaterProviderOptionsApply` (OpenAI, and Grok's twin); `TestOAuthSession_CallersClientReplacesDefault` | `client is the caller's false, logger false; want both`; `a caller's client did not replace the provider's default` | D8: `Settings.HTTPClientGiven` and `LoggerGiven`. A provider gives a session a caller's client through `UseHTTPClient`, and its default through `UseDefaultHTTPClient`, which a caller's replaces; only a caller's logger is shared. 0016-PLAN T1's shared-client test passes in both providers. |
| F33 | `TestOpencode_MetadataLookupUsesClientInfo` | `metadata User-Agent = "go-llmprovider-sdk/(devel) (darwin; arm64) go-llmprovider-sdk/(devel)"` | `catalog.LookupMetadataWith(ctx, id, opts...)`, additive; OpenCode passes its listing options, which carry the caller's. The unused `metadataURL` field is removed. |
| F34 | `TestChatGPTListing_BoundedAndKinded` | `40 MiB: … listed no models`, a 40 MiB body decoded; `undecodable: … unexpected EOF` with no kind; `lists nothing: …` with no kind | The listing is read under `chatGPTListingLimit`, catalog's 8 MiB (0021 C3); an oversized, undecodable or empty listing is `ErrIncomplete`. |
| F35 | `TestOllama_ListModelsKinds` | `503: ollama returned HTTP 503`; `401: ollama returned HTTP 401`; `unreachable: could not reach Ollama …`, none with a kind | Ollama's listing is classified by `ClassifyHTTPError`; an unreachable server is `ErrProviderUnavailable`. |
| F38 | `TestLive_GeminiFlashLiteEffortThinks` (live); `TestGemini_Capabilities` | `Capabilities = {… Reasoning:Supported …}` | Measured below. Gemini's Reasoning is `BestEffort`, with the degradation in the package doc (R12). |
| F39 | none, documentation | Read against the code and 0021 L2. | Ollama's and Kilo's `ToolChoiceNone` sends no tools (0020 F40); Grok's instructions were measured in both forms (0021 L2's amendment); `WithSessionID` names the ChatGPT backend's `session-id`. |

* **F38, measured** twice, with `GEMINI_API_KEY`, through the provider:

  | Run | low | medium | high |
  | :--- | :--- | :--- | :--- |
  | 1 | 0 reasoning tokens, no summary | 0, no summary | 1350, a summary |
  | 2 | 0, no summary | 0, no summary | 910, a summary |

  At low and medium, `thinkingLevel` sends nothing for this model (it
  refuses medium and fails on low), and the model does not think. The
  PLAN's second fix, a thinking budget, is not open: the Interactions API
  has none (the package doc; MADR 0014). So Reasoning is `BestEffort`, as
  0020 F43 made Grok's.
* **Tests changed, not new:**
  * `TestListModels_ChatGPTFailureIsAnAPIError`: its fixture answered every
    request 403, the refresh included. A 403 is `ErrAuthFailure` on OpenAI,
    so F6 refreshes once. The fixture now answers the refresh, and the test
    also checks one refresh and two listings.
  * `wizard`'s `TestConfigureLLM_StaticCatalogNotice`: the notice's cause
    is the classified error, `llmprovider: provider unavailable: opencode
    HTTP 500`. It is still shown once.
  * `TestGemini_Capabilities` wants `BestEffort` (F38).
* **Docs.**
  * `docs/architecture.md`: "Sessions are `auth`'s" covers the default and
    caller's clients; the catalog row gains `LookupMetadataWith`.
  * Doc comments: `List`, `Reauth`, `UseHTTPClient`,
    `UseDefaultHTTPClient`, `UseLogger`, the new `Settings` methods,
    Gemini's package doc, and F39's four.

**Proofs** (scratch copies):

| Planted | Failed |
| :--- | :--- |
| `Reauth` back to 401 only (`plant_f6.py`) | Gemini's `R16-reauth`, stored and unstored: `0 invalidation(s) and 1 request(s)`; three `TestReauth_RerunsOnAuthFailureKind` cases |
| `List` without its rerun (`plant_f36.py`) | `TestList_RerunsCredentialOn401`, both cases: `requests=1 invalidations=0` |
| the default client passed as a caller's (`plant_f37.py`) | `TestSession_LaterProviderOptionsApply` in OpenAI and Grok: `client is the caller's false` |
| the reference provider renewing only after a 401, against a 400 refusal | `TestRun_AuthFailureIsTheServicesRefusal`'s own assertion, in the tree |

The first `List` plant, the 401-only rule, passed, since F36's test sends a
401; the second removes the rerun.

**The gate,** on a scratch copy with the phase staged and committed there
with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `447 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems |
| `coverage-check` | 28 packages, 0 problems; `llmprovider` 98.2 %, `auth` 87.4 %, `catalog` 92.1 %, `internal/wire` 96.1 %, `llmtest` 95.9 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)`; the additions are `Settings.HTTPClientGiven`, `Settings.LoggerGiven`, `OAuthSession.UseDefaultHTTPClient`, `catalog.LookupMetadataWith` and `llmtest.Harness.AuthFailure` |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 605 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 32 files |

**Not done in this phase:** a live run of the changes other than F38's
measurement. Each is covered by the unit and conformance tests above;
Phase 7's release gate runs the live suites.

**Found, and outside this phase (not done):** `docs/guides/api-standards.md`
R2's package table lists `internal/wire`'s imports as `llmprovider` alone,
and `catalog`'s as including `internal/transport`; the code differs, as it
did before this phase. The guide is normative and changes only through its
decision, so it is left for Phase 6's docs work. *(Corrected in Phase 6:
`catalog` does import `internal/transport`, as `go list` shows, so R2's
`catalog` row was right. Only the `internal/wire` rows differed.)*

### Phase 5: wizard and catalog (2026-10-07)

* **Before it.** Phase 4 was committed as `1477638` and pushed by the
  owner. Its CI run, 37649475401, passed.
* **Approval.** "comitted and pushed. proceed.", 2026-10-07.
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code (`p26_phase5_red.out`).

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F9 | `TestConfigure_EmptyRecommendedHonoursCtx` (defaults on an empty recommendation; a search that never matches); `TestConfigure_EmptyRecommendedSearchStillWorks` | Both cases: `ctx expired 1.7s ago; ConfigureLLM still running after 154614735` and `2056285 prompter calls` | `selectModel` and `selectFallbacks` take `ctx`, and check it each round. A blank search on an empty recommendation shows the notice, then the menu of the current model, if any, and Other. |
| F10 | `TestConfigure_KeptSessionUsesCallersClient`; `TestList_GivesASessionItsClient` | `the kept session does not refresh through Options.HTTPClient`; `the source was not given the listing's client` | `keepExistingOAuth` calls `session.UseHTTPClient(o.HTTPClient)`. `List` gives a source that takes one its client, through `UseDefaultHTTPClient`. |
| F46 | `TestSelectRecommended_ExistingModelOnlyForItsProvider` | `another provider's model: default index = 2; want 0` | The saved model is the default only when `Existing.Provider` is the provider. |
| F47 | `TestChatGPTCatalog_Capped` | `Recommended has 14 models, Usable 14; want at most 6 recommended` | `chatGPTCatalog` recommends the first `MaxListed`, as `providerCatalog` does; search covers all. |
| F48 | `TestPasteAccessToken_KeepsAccount` (nested claims; a top-level account; none) | `AccountID "", FedRAMP false; want "acct-123", true`, and `"acct-456"` | `accessTokenAccount` reads `chatgpt_account_id`, top level or under `https://api.openai.com/auth`, and `chatgpt_account_is_fedramp`, the claims a login reads in its id_token. |
| F49 | `TestTextPrompter_SecretSurfacesWriteError` | `Secret = "sk-key-1234", <nil>; want the write error and no value` | `Secret` returns `flushErr`'s failure; a value read with a failing reader is not kept; the first end of input still answers, as `Input`'s does. |
| F50 | `TestTextPrompter_SecretRefusesPartialEntry` (a hang-up; end of input) | `readMasked = "sk-partial-k", read: input/output error`, and `…, EOF` | `endMasked` returns no value: an entry not ended with Enter is never kept. Its unused `entered` parameter is gone. |
| F51 | none, documentation | Read against `catalog.WithProfile` and the README. | `Options.Profile` names Together. |
| F52 | `TestValidateOllamaURL_TrailingSlash` | `requested "//api/version"`, and `"///api/version"` | `checkOllamaURL` trims trailing slashes, as `ResolveOptions` does for the listing. |
| F53 | `TestMetadata_ReasoningEffortsIsACopy` | `a later lookup = ["POISONED" "medium" "high"]` | `reasoningEfforts` returns a clone (R29). |
| F54 | none, documentation | Read against `kiloCurate` (cheapest first) and `fetchHuggingFaceUsable` (fastest first). | `Catalog.Usable` and `Search` say the order each lister gives. |

* **Two notes on the fixes, against the table above:**
  * **F9.** The row's test wants a defaults-only run to end in
    `context.DeadlineExceeded`. With the row's own menu fix, that run ends
    first, in milliseconds, with `wizard: no model entered`. The ctx check
    is still needed: a search that never matches, answered by its default,
    Search again, loops without it. The test therefore has the two cases:
    the defaults run returns promptly, with an error, and the search run
    returns `DeadlineExceeded` promptly. 0021 C13's
    `TestConfigure_EmptyRecommendedOpensSearch` is kept, and passes; its
    blank first search now reaches its model through Other, so
    `TestConfigure_EmptyRecommendedSearchStillWorks` checks that a typed
    query still searches the listing.
  * **F10.** The row says `List` hands its client to "a source that has
    `UseHTTPClient`". D8, decided in Phase 4, made `UseHTTPClient` the
    caller's and added `UseDefaultHTTPClient`. A provider calls `List` with
    its own client, which may be its default: through `UseHTTPClient` it
    would be marked a caller's, and a later caller's client could not
    replace it, against F37. `List` uses `UseDefaultHTTPClient`, which
    sets a client only on a session with none. The wizard's half,
    `keepExistingOAuth`, passes `Options.HTTPClient` as the caller's.
* **Docs.** The README's "goes straight to search" now says what a blank
  search offers; `Options.Profile`, `Catalog.Usable` and `Search` are
  corrected (F51, F54).

**The gate,** on a scratch copy with the phase staged and committed there
with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `456 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems |
| `coverage-check` | 28 packages, 0 problems; `wizard` 89.2 %, `catalog` 92.5 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)` |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 605 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 19 files |

**Not done in this phase:** a live run. Each change is covered by the unit
tests above; the wizard's TTY path (`readMasked` in raw mode) is tested
through its reader, not a terminal. Phase 7's release gate runs the live
suites.

### Deviation D9 (2026-10-07): a failed renewal keeps the refusal's `*APIError`

* **Found** with F57's R24 check, which wants every HTTP failure to be an
  `*APIError`. Every built-in passes, except Grok's and OpenAI's session
  harnesses: `HTTP 401 returned *fmt.wrapError (llmprovider: grok: acquire
  token: oauth: no refresh token: llmprovider: authentication failed)`. A
  401 makes `Reauth` renew the session; the harness session has no refresh
  token, so the renewal fails, and `Reauth` returns that failure alone. The
  401's status, kind and message are lost. A session that cannot renew,
  such as Kilo's device login, meets the same path. F57's PLAN row says a
  built-in that fails a new check is a deviation.
* **Options put to the owner:** keep both errors; exempt the path from the
  check; give the harness sessions a refresh.
* **Decision.** The owner chose to keep both ("Keep both errors"). When the
  renewal fails, `Reauth` returns `errors.Join` of the refusal's error and
  the renewal's. F57's files gain `llmprovider/internal/wire/reauth.go`.

### Phase 6: harness, gates and docs (2026-10-07)

* **Before it.** Phase 5 was committed as `97bb439` and pushed by the
  owner. Its CI run, 37667118751, passed.
* **Approval.** "proceed", 2026-10-07, with D9 ("Keep both errors").
* **Red first.** Each test below was written first, and its FAIL is from
  the unchanged code (`p26_phase6_red.out`).

| ID | Test | FAIL before the fix | Fix |
| :--- | :--- | :--- | :--- |
| F66 | `CopyTreeTest.test_copy_tree_drops_staged_deletion` | `the copy still holds llmprovider/catalog/model_matcher.go, whose deletion is staged` | `copy_tree` (now given its source) also removes `git diff --name-only --diff-filter=D HEAD`. |
| F14 | `test_parity_check_bare_missing_function` | `parity-check exit 0, want 1` | `unresolved()` resolves a bare call only against package-level names. The eight `Static…` rows name `catalog.Static(llmprovider.ProviderX)`. |
| F16 | `test_dep_check_allowed_module_outside_wizard`, `test_parity_check_unexported_name` | Not red: they pass against the real gates. Seen to fail with each gate's main check disabled (below). | Both tests. |
| F61 | `test_records_check_record_outside_its_directories` | `records-check exit 0, want 1` | `stray_records()` reports a record under `docs/` outside `decisions/` and `reports/`; `--next` counts every record under `docs/`. |
| F15 | `PrecheckTest.test_precheck_deleted_file_checks_its_package` | `go-precheck exit 0, want 1: go-precheck: no Go files to check.` | A deleted `.go` path adds its directory to the packages checked, or `./...` when no Go file is left there. |
| F60 | none: drift. Seen to fail with the pin removed (below). | — | `make vuln` and the precheck run `go run golang.org/x/vuln/cmd/govulncheck@<CI's pin>`, the pin read from `.github/workflows/ci.yml`, its one place. |
| F55 | `TestRun_ForcedChoiceRefusalWithoutToolCall` | `ToolCall set false: failures []; want R11 for the forced tool choice` | The refusal check runs whatever `ToolCall` is; only an honoured forced choice needs it. |
| F57 | `TestRun_NamesTheBrokenRule`'s six new rows; `TestReauth_FailedRenewalKeepsTheRefusal` (D9) | Each new check fails on its flaw (below); `Reauth = llmprovider: p: acquire token: oauth: no refresh token; want the refusal's *APIError` | R24 and `RetryAfter` in the classification check; `checkNotPermitted` (`Harness.NotPermitted`, set by Kilo, OpenCode and OpenAI); `checkTokenHeader` (R16); `checkListing` (R44, R40). D9's `errors.Join` in `Reauth`. |
| F58 | `TestFake_ReplyNilRefused`, `TestFake_RequestsCopiesSchema` | `Reply(nil) returned; want it to panic at the call`; `recorded schema = map[… type:CHANGED]` | `Reply(nil)` panics, naming itself; `cloneRequest` copies each schema through JSON. |
| F56 | none: coverage. Seen to catch a reverted fix (below). | — | OpenCode's harness runs four routes from a per-route reply table: chat, messages, google and responses. |
| F59 | `TestFileFindings_PlantedDirs` | `no finding for os.UserConfigDir`, `os.UserCacheDir`, `user.Current`, `syscall.Getenv` | The four selectors; `syscall.Getenv` is allowed in a `…FromEnv` function, as `os.Getenv` is. |
| F62 | none: docs. Checked by `go list` (`p6_f62_deps.py`, rerun to 0 differences). | — | R2: `internal/wire` and `<format>` may import `internal/transport` and `internal/redact`, and `<format>` is "one wire format". `architecture.md`'s "Depends on" column is regenerated for six providers and `catalog`. |
| F64 | none: docs. | — | 18 cases and 114 golden files, counted from `testdata/wire`. |
| F63 | not done here | — | Its row says the README names the current release "at release time, in Phase 7". |

* **Notes on the fixes:**
  * **F14.** Under the row's rule, eight more cells fail: each
    `ProviderX.Name` row named `ID()`, a bare method call. They now name
    `Provider.ID()`, which resolves as a member. The gate then reports 0
    problems over 409 rows.
  * **F57.** A token with no `Header` goes in the service's own header,
    which may be none: Ollama's takes no credential and sends nothing. So
    that case is not a check every provider must pass; the named-header
    case, bare and `TokenBearer`, is. A Harness with `NoReauth` skips it,
    as the reauth check does, since the check's source would change the
    provider's mode. The reference provider gained `ListModels`, `failure`
    (an `*APIError` with `RetryAfter`, and `ErrNotPermitted` for a
    `not_permitted` 403), and six flaws.
  * **F60.** The row says "pin it in one place". CI already pins
    `govulncheck@v1.8.0`; the Makefile and the precheck read that pin, so
    the workflow is the one place. `make vuln` ran it: `No vulnerabilities
    found.`
  * **F62.** Phase 1 added `filelock` to R2's table with no amendment to
    0015-MADR; this follows that, and R2's citation names 0026 F62. Phase
    4's note that `catalog`'s row was wrong was itself wrong, and is
    annotated there.
* **Docs.** `architecture.md`'s and `adding-a-provider.md`'s `llmtest`
  sections list `AuthFailure`, `NotPermitted` and the field-less checks;
  `llmtest.Run`'s doc lists them; AGENTS.md's pre-add paragraph says what
  the precheck now does; `check_records.py`'s docstring.

**Proofs** (scratch copies):

| Planted | Failed |
| :--- | :--- |
| `dep-check`'s per-package `check()` returning no problems (`plant_f16.py`) | `test_dep_check_allowed_module_outside_wizard`: `dep-check exit 0, want 1` |
| parity's `unresolved()` returning nothing (`plant_f16.py`) | `test_parity_check_unexported_name`: `parity-check exit 0, want 1` |
| CI's govulncheck pin removed (`plant_f60.py`) | `make vuln`: exit 2; the precheck: exit 2, `govulncheck: no pin in .github/workflows/ci.yml` |
| generateContent's F7 fix reverted (`plant_f56.py`) | `TestConformance/zen-google/F7-reasoning-cut` |
| the reference provider's flaws, in the tree | `notAPIError` → R24; `noRetryAfter` → R24 (RetryAfter); `notPermittedAsAuth` → R25-not-permitted; `ignoresTokenHeader` → R16-token-header; `listingNoIdentity` → R44; `listingIgnoresCancel` → R40 |

**The gate,** on a scratch copy with the phase staged and committed there
with plumbing (`p26_gate.py`); every check rc=0:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `459 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)`, govulncheck now through `go run …@v1.8.0` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -count=1 -cover ./...`; `go test -shuffle=on ./...` | 27 packages ok, the 28th having no tests |
| `go mod tidy -diff` | clean |
| `make lint` (host and Windows) | 0 issues |
| `parity-check`, `dep-check` | 0 problems; `G-parity: 409 identifiers, 409 rows, 409 with an SDK equivalent, 0 problem(s)` |
| `coverage-check` | 28 packages, 0 problems; `llmtest` 96.0 %, `internal/wire` 96.8 % |
| `api-check` | `against v1.2.1, 0 incompatible change(s)` |
| `generate-check` | `2 generated file(s), 0 problem(s)` |
| `records-check` | `57 records, 0 problem(s)` |
| `gate-selftest` | `Ran 14 tests`, OK |
| markdownlint (the lint scope), G-wire stable, links | 0 problems; 605 relative links in 67 files |
| identifier scan of the changed files | 0 hits in 22 files |

**Not done in this phase:** F63, which its row puts in Phase 7; and a live
run, which Phase 7's release gate does.
