---
status: in-progress
date: 2026-10-06
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
| 2 | errors and retry | F4, F5, F11, F12, F17, F18, F19, F20, F21, F22, F23, F24 |
| 3 | answers on the wire | F7, F8, F25, F26, F27, F28, F29, F30, F31, F32, F65 |
| 4 | provider wiring | F6, F36, F37, F33, F34, F35, F38, F39 |
| 5 | wizard and catalog | F9, F10, F46, F47, F48, F49, F50, F51, F52, F53, F54 |
| 6 | harness, gates and docs | F14, F15, F16, F55, F56, F57, F58, F59, F60, F61, F62, F63, F64 |
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
| F13 | `TestFileTokenStore_TakeoverRaceOneHolder`, from `TestZZ_TakeoverRaceTwoHolders`, driven through the same seams; `TestFileTokenStore_LockReleasedWhenHolderDies`, a child process that takes the lock and is killed. The first fails today with `W3 err=<nil> holds=true`. | Q1 (a), third part: OS file locks. <ul><li>A new `llmprovider/internal/filelock` package locks an open file exclusively:<ul><li>`syscall.Flock` with `LOCK_EX|LOCK_NB` on Unix;</li><li>`LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` on Windows, through `//sys` lines and its own `//go:generate mkwinsyscall` output, as `ownerperm` has.</li></ul></li><li>A non-blocking attempt is retried with backoff, until the ctx or `lockWait` ends.</li><li>The lock file `<provider>.oslock` is never deleted, so no process can lock an unlinked file.</li><li>The OS releases a dead holder's lock, so staleness, takeover and the heartbeat go.</li><li>**Compatibility:** while it holds the OS lock, a process also creates the legacy `O_EXCL` lock file with its owner token, and removes it on unlock. A `v1.2.x` process sharing the store then still waits, as it did before.</li></ul> | `llmprovider/internal/filelock` (new), `auth/tokenstore_file.go`, their tests; `docs/guides/api-standards.md` R2 and `docs/architecture.md` gain the package; `generate-check` covers its generated file |
| F2 | `TestNew_RefusesForeignSession` in `grok` and `openai`: <ul><li>a ChatGPT, Kilo and Grok `OAuthSession`;</li><li>a Codex and a Grok `VendorCLISession`;</li></ul> each given to the provider that does not own it. Each must fail `New` with `ErrUnsupported`, and send nothing. They fail today with `New err=<nil>`. `TestIsChatGPTSession_NoIssuer`: an OpenAI session with no issuer is a ChatGPT session. | Q2 (a): <ul><li>`grok.New` refuses an `*auth.OAuthSession` or `*auth.VendorCLISession` whose `Provider` is not `grok`;</li><li>`openai.New` does the same for `openai`;</li><li>`OAuthSession.ChatGPT()` (or `isChatGPTSession`) counts an OpenAI session with an empty issuer, matching `refreshTokenURL`.</li></ul> The messages follow `kilo.New`'s. | `providers/grok`, `providers/openai`, `auth/oauth_session.go`, their tests |
| F40 | `TestVendorCLISession_SymlinkedAuthFile`: a Grok `auth.json` that is a symlink to a file outside its directory is read. It fails today with `path escapes from parent`. | Resolve the path with `filepath.EvalSymlinks` first, then open the resolved file. The `os.Root` stays for the resolved directory, as gosec wants (0006-PLAN). The error for a missing target says the link is broken, not "run grok login". | `auth/vendor_session.go` |
| F41 | `TestVendorCLISession_TornFileRetriedOnce`: a file that is truncated on the first read and whole on the second is read. It fails today with `unexpected end of JSON input … (kind auth)`. | A decode failure is retried once, after 50 ms. A second failure keeps today's error. | `auth/vendor_session.go` |
| F42 | Windows only, in CI: `TestFileTokenStore_SaveWhileOpenForRead`. A reader holds `<provider>.json` open while `Save` runs, and the save succeeds. It is read-only today on macOS, so its first failure is recorded from CI's `windows-2025` job on a scratch branch the owner pushes, or from the owner's Windows host. | Open token files for reading with `FILE_SHARE_DELETE`, through a Windows-only open helper beside `filelock`'s bindings (`CreateFile`), so a rename can replace a file a reader has open. | `auth/tokenstore_file.go`, a `_windows.go` helper |
| F43 | `TestRefreshErrors_HaveKinds`, from `TestZZ_RefreshErrorKinds`, plus the device "denied" and "expired" cases. Each error matches an `llmprovider` kind: <ul><li>a transport failure is `ErrProviderUnavailable`;</li><li>a 200 that does not decode is `ErrIncomplete`;</li><li>a lock file that cannot be created is `ErrProviderUnavailable`;</li><li>denied is `ErrAuthFailure`;</li><li>expired is `ErrAuthFailure`.</li></ul> They fail today with `kind NONE`. | Wrap each with its kind, keeping the message. | `auth/oauth_session.go`, `tokenstore_file.go`, `oauth_device.go`, `kilo_device.go` |
| F44 | `TestValidateOAuthSession_RefusesUnrefreshable`: a custom-issuer session with a refresh token and no token URL is refused. It passes `ValidateOAuthSession` today. | Refuse it there, with the same message `Token` gives, and correct the doc comment (0020 F45). | `auth/oauth_session.go` |
| F45 | `TestRevoke_LegacyGrokSession`: a Grok session with no issuer revokes against `auth.x.ai`'s discovery, or returns `ErrUnsupported` if that has no endpoint. It fails today with `unsupported protocol scheme ""`. | Revocation derives the issuer as `refreshTokenURL` does. | `auth/oauth_revoke.go` |

### Phase 2: errors and retry

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F4 | `TestPost_StalledErrorBodyEndsAtIdleLimit`, from `TestZZReproStalledErrorBody`: a 503 whose body stalls returns within the idle limit, with no caller deadline. It fails today, returning only at the 3 s deadline. A 200 that stalls keeps today's behaviour. | `Post` wraps `resp.Body` in the idle-limited `ReplyReader` before `ClassifyHTTPError`. `ClassifyHTTPError`'s own bound stays. | `internal/wire/post.go`, its test |
| F5 | `TestChatCompletions_ErrorIn200Classified`, from `TestZZReproGatewayErrorIn200` and `TestReproGatewayErrorCodes`: a numeric `error.code` gives the status's kind. <ul><li>400 overflow is `ErrContextOverflow`;</li><li>401 is `ErrAuthFailure`;</li><li>402 is `ErrQuotaExhausted`;</li><li>403 is `ErrNotPermitted`;</li><li>429 is `ErrRateLimited`.</li></ul> `APIError.Provider` is the provider's label. A Kilo or OpenCode type gives its row's kind. The server is hit once for terminal kinds. These fail today. | A numeric code from 400 to 599 is classified through `classifyAPIError(serviceOf(provider), code, env, nil)`, overflow check included. A non-numeric code keeps `ClassifyStreamFailure`. The decoder receives the provider label from its caller. | `api_error.go`, `internal/wire/chatcompletions`, the providers that call it |
| F11 | `TestResolveOptions_RefusesUnsendable` and `TestWithRetry_UnsendableIsTerminal`, from `TestZZReproDeterministicURLErrorRetried`: <ul><li>a base URL with no scheme or host, or with an invalid escape, fails `New` with `ErrInvalidRequest`;</li><li>a key, client info or session id with a control character fails `New` the same way;</li><li>a source token with a newline fails on the first attempt, with `ErrInvalidRequest`, and is not retried.</li></ul> They fail today with `attempts=4 kind=NO KIND`. | Q7 (a): <ul><li>`ResolveOptions` parses `WithBaseURL` (scheme and host required), and refuses control characters in `WithAPIKey`, `WithClientInfo` and `WithSessionID`;</li><li>`retryable` treats a `*url.Error` from parsing, an invalid header value, or an unsupported scheme as terminal, and `Post` gives it `ErrInvalidRequest`.</li></ul> 0021-PLAN kept `WithBaseURL` permissive (R48); Q7 (a) reverses that, so the release notes say so. | `settings.go`, `retry.go`, `internal/wire/post.go`, their tests |
| F12 | *Live first* (Q6 a): capture a real Gemini 429 with the owner's key, in a live-tagged test that records the body shape and headers. Then `TestClassify_GeminiRetryInfo`: the captured body gives `RetryAfter` from `retryDelay`, and a per-day `QuotaFailure` gives `ErrQuotaExhausted`. It fails today with `RetryAfter=0s retryable=true`. | Decode `details[].retryDelay` (a protobuf Duration string) into `RetryAfter` when no header is present. Classify a `QuotaFailure` violation whose quota id names a day as `ErrQuotaExhausted`. If the capture differs from Google's documented shape, that is a deviation. | `api_error.go`, a live test |
| F17 | `TestRedact_PrefixedKeys`: `openai_api_key=`, `"x_api_key"`, `app_secret:`, `"apiSecret"`, `db_password=` and `"subscription_key"` are redacted. They fail today. | `reKV` matches a key name at the end of a snake_case or camelCase identifier, not only after a word boundary. | `internal/redact` |
| F18 | `TestAPIError_ReasonIsBounded`: a `Reason` with ESC and BEL, 2 KiB long, reaches `Error()` stripped and capped. It fails today with `ESC=true BEL=true len=2093`. The token endpoint's error body is stripped too. | `StripControl` and the code bound on `Reason`, where it is set; `StripControl` on the token endpoint's message. | `api_error.go`, `internal/wire/finish.go`, `responses.go`, `auth/oauth_session.go` |
| F19 | `TestNew_RefusesInvalidDefaultReasoning`: `WithReasoning` with an unknown effort or a negative budget fails `New`. It fails today with `New err=<nil>`. | `ResolveOptions` applies the `Request` validation to `WithReasoning`. | `settings.go` |
| F20 | `TestValidate_UnmarshalableSchema`: a `Tool.Schema` holding a channel, or an invalid `json.RawMessage`, is `ErrInvalidRequest` before any request. It fails today with `kind=NO KIND`. | `validate` marshals each schema once. `Post`'s marshal failure is `ErrInvalidRequest`. | `contract.go`, `internal/wire/post.go` |
| F21 | `TestRedact_KeepsOrdinaryWords`: `"Invalid bearer token"`, `"Basic authentication is not supported"` and `"The token provided is invalid"` come back unchanged, while `Bearer sk-…` and `Basic dXNlcjpwYXNz` are redacted. It fails today. | `reAuth` requires a credential-shaped value after the scheme: base64 or token characters, at least 8 long, not an English word followed by a space. | `internal/redact` |
| F22 | `TestRetry_ServerWaitNeverExceedsMaxDelay`, from `TestZZReproServerWaitOverMaxDelay`: over 20 runs, no wait exceeds `MaxDelay`. It fails today at 338 ms over 100 ms. | Cap the jittered wait at `MaxDelay`. | `retry.go` |
| F23 | `TestAPIError_RetryableDoesNotMatchInvalidRequest`: a retryable 408, or 409 on openai and claude, does not match `ErrInvalidRequest`. It fails today with `isInvalidRequest=true`. | `Unwrap`'s legacy sentinel is omitted when the error is `Retryable()`. 0012 §7's compatibility rule is kept for terminal errors, and its amendment records the exception. | `api_error.go`, `0012-MADR` amendment |
| F24 | `TestPost_Accepts2xx`: a 201 decodes as a success. It fails today with `invalid request … HTTP 201`. | `ClassifyHTTPError` passes 200–299. Its doc and `Post`'s agree. | `api_error.go` |

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

### Phase 7: release (owner, then agent)

1. **The owner** commits and pushes Phase 6. CI is green on Linux, macOS and
   Windows.
2. **Release notes**, in this PLAN and the README's release paragraph:
   * the behaviour changes: F2's refusal of a foreign session, F11's
     refusal of unsendable inputs at `New`, and F7's `ErrIncomplete`;
   * the lock change of F13: `<provider>.oslock`, and the legacy file kept
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
  * Phase 1's lock change is safe to revert: the legacy lock file kept
    alongside the OS lock means a reverted process waits correctly on a
    store a `v1.3.0` process holds.
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
