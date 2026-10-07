---
status: accepted
date: 2026-10-06
decision-makers: repository owner
consulted: 0015-MADR-canonical-sdk-api-and-module-layout.md (R16, R23, R24, R25, R27, R29, R35, R40, R45, R48), 0016-MADR-provider-auth-and-support-baseline.md (D4, D8, A2), 0017-MADR-together-provider-and-auth-extensions.md (D2, D3), 0020-MADR-remediate-v1-debugging-pass-findings.md, 0021-MADR-harden-and-tune-after-the-v1-1-review.md
informed: consumers of go-llmprovider-sdk v1, among them prepare-commit-msg (on v1.2.1) and gobble-cli (on v1.1.1)
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Remediate the Defects Found by the v1.2 Debugging Pass

> **Accepted 2026-10-06.** Each design question takes its recommended
> answer: Q1 (a), Q2 (a), Q3 (a), Q4 (a), Q5 (A), Q6 (a) and Q7 (a).
> §2 records the owner's words. The 0026 PLAN is written next; no fix
> starts before it is approved.

## Context and Problem Statement

`v1.2.1` is tagged on `64b82d5`. The records it carries are complete:

* 0020, the v1 debugging pass;
* 0021, the hardening after the v1.1 review;
* 0022–0025.

On 2026-10-06 the owner asked for another pass over the whole module: "Run
a debugging pass across the codebase. Find bugs. Find gaps. Find missing or
incomplete wiring and functionality. Report findings to an comprehensive,
detailed, and up to date madr for review."

The pass found 64 findings. Research for Q5 added a 65th (F65):

* 2 of high severity:
  * F1 brings back 0020 F1's lost rotation, with two failed saves in place
    of one;
  * F2 sends one vendor's session token to another vendor.
* 14 medium;
* 49 low, including documentation, tooling and test-harness gaps.

Many are the edges of fixes that 0020 and 0021 made. For example:

* F1 is 0020 F1 with two failed saves.
* F5 is the half of 0021 W5 that never classifies the error code.
* F6 is 0020 F2's rerun, which never fires on Gemini's way of refusing a
  key.
* F8 is a defect that 0021 W10's own fix introduced.
* F14 is 0020 F56's class of broken mapping cell, in rows the parity gate
  still cannot see.

This record lists every finding with its evidence. It proposes how to
remediate them, and asks the owner the seven design questions that the
fixes depend on.

### Method

* **Baseline**, on `c523c65`, all exit 0:
  * `go test -race -count=1 ./...`: 27 packages ok;
  * `go test -shuffle=on -count=1 ./...`;
  * `CGO_ENABLED=0 go vet ./...` for Linux, macOS and Windows, with and
    without `-tags live_gateways`;
  * `make lint`: 0 issues for the host and for `GOOS=windows`;
  * `make vuln`: no vulnerabilities;
  * `records-check`: 55 records, 0 problems;
  * the six CI gates on a clean copy: coverage (27 packages), deps, parity
    (409 identifiers), records, api (against `v1.2.1`) and generate.
* **Six read-only audits ran in parallel,** one per area:
  * the core contract, with transport and redaction;
  * the wire encoders and decoders;
  * the ten providers and the registry;
  * `auth` and `ownerperm`;
  * `catalog` and `wizard`;
  * `llmtest`, tooling, CI and documentation.
* **Reproduction.** Each audit worked in its own shared clone of the
  repository, in the session's scratch space. There it proved its
  suspicions with `httptest` servers, scripted prompters, the token
  store's own test seams, or planted defects.
  * The repository was not written to.
  * No live service was called.
  * No real token store or home directory was touched.
* **Labels.** "Repro" below means a scratch test showed the failure, and its
  output is quoted. "Read" means the finding comes from the code alone, and
  says why it was not run.
* **Checked again.** Before this record was written, the owner's agent
  re-read the code of F1, F2, F4, F6, F7, F9, F14 and F15, and of the low
  findings F24 and F63, at `c523c65`. The remaining findings rest on the
  audits' reproductions.
* **Line numbers** are at `c523c65`.
* **Merged duplicates.** When two audits reported the same defect, it is
  listed once, with every source ID:
  * F5 is core C2 and wire W2;
  * F14 is docs D1 and tooling T1;
  * F28 is wire W11 and providers P8;
  * F51 is wizard Z8 and docs D4.
* **Source IDs** (C, W, P, A, K, Z, L, T, D) are the audits' own. The PLAN
  uses them to find each reproduction.

**Out of scope, already recorded:**

* 0002's later phases, which wait on the orchestrator move out of mcplib;
* Together's `required` HTTP 500 on `gpt-oss-120b`, pinned by 0017;
* the Groq refusal of a tool call after a tool turn under `none`, recorded
  by 0023 as a service limit;
* a refresh retried after the 15 s attempt timeout or a 5xx resends the same
  refresh token. That is how the vendors' own CLIs retry inside the issuer's
  grace window (the Grok CLI allows about 60 s). It is context for F3, not a
  finding.

### A. High: a credential lost or sent to the wrong vendor

| ID | Source | Where | Finding | Evidence | Prior record |
|---|---|---|---|---|---|
| F1 | A1 | `auth/oauth_session.go:189` (`s.spentRefresh = out.spent`), `:230-243` (`persistRotation`), `:691-702` (`loadRotated`) | **Two rotations in a row that fail to save lose the session.** <ul><li>The second failure overwrites `spentRefresh` with the second spent token, while the store still holds the first.</li><li>When saving works again, `persistRotation` sees a refresh token that is neither the recorded spent one nor the new one. It returns `errStoreMovedOn` and never writes.</li><li>The store keeps a spent token. The next refresh, here or in a new process, adopts it through `loadRotated` and sends it again.</li></ul> The issuer then revokes the whole family (0016 A2), and the user must sign in again. | Repro `TestZZ_TwoFailedSavesLoseRotation`: `step3 (resave) … spentRefresh="" store.Refresh="old-refresh"`, then `step4 … refresh_token_reused`, `sent: [old-refresh refresh-1 old-refresh]`. Code re-read: line 189 assigns unconditionally. | Incomplete 0020 F1, 0021 T10 |
| F2 | P1 | `providers/grok/grok.go:95-104`; `providers/openai/openai.go:116-122`; `openai/chatgpt_session.go:31-37` | **Grok and OpenAI accept another vendor's session, and send its token to their own API.** <ul><li>`grok.New` accepts any `TokenSource`, including a ChatGPT or Kilo `OAuthSession` and the Codex CLI's session.</li><li>`openai.New` picks its backend from the issuer alone. Any session that is not a ChatGPT session goes to the platform API as a bearer token. That includes a Grok session, and an OpenAI session with no issuer, which the refresh code itself treats as a ChatGPT one (`oauth_session.go:713`).</li></ul> R16 says `New` refuses a source the service does not accept, and `kilo.New` does (`kilo.go:141-150`). | Repro `TestZZRepro_GrokForeignSession`: `openai-chatgpt-oauth: New err=<nil> … Authorization sent to xAI endpoint = "Bearer CHATGPT-ACCESS"`; the Kilo session gives `"Bearer KILO-ACCESS"`. `TestZZRepro_OpenAIForeignSession`: `grok-oauth: … chatGPT=false … Authorization="Bearer XAI-ACCESS"`; `openai-no-issuer: chatGPT=false … "Bearer CHATGPT-LEGACY"`. Code re-read. | none. 0020 marked "credential refusals" sound, but checked only the providers that refuse sessions. |

**F1.** The trigger is any save failure that lasts across one refresh or
one 401: a full disk, a permissions change, or F42's Windows sharing
violation. 0020 F1 fixed the single failure; this is the double one, and
the result is the same: the user is signed out. The audit reproduced it
with the store's own seams.

**Fix:** record the refresh token the store is known to hold. Have
`loadRotated` and `persistRotation` compare against that, not against the
last spent token. The minimal form keeps the oldest spent token while one
is set. Q1 asks which.

**F2.** The audit rated this medium, because the trigger is a caller's
mistake:
* a token-store key mixed up between providers;
* a registry configured with the wrong source.

This record rates it high. The consequence is a credential sent to a third
party, and then a refresh against the wrong issuer when that party answers
401. 0020's driver "credential safety first" puts it with F1.

**Fix:** refuse, with `ErrUnsupported`:
* in `grok.New`, an `*auth.OAuthSession` or `*auth.VendorCLISession` whose
  `Provider` is not `grok`;
* in `openai.New`, the same for `openai`.

Also treat an OpenAI session with no issuer as a ChatGPT session. Q2 asks
whether to refuse or only warn.

### B. Medium

| ID | Source | Where | Finding | Evidence | Prior record |
|---|---|---|---|---|---|
| F3 | A2 | `auth/oauth_session.go:182`, `:203`, `:144-146` | **A refresh cancelled by the caller loses the rotation.** <ul><li>The refresh runs under the caller's context.</li><li>Issuers commit a rotation before they answer. A `Generate` deadline or a Ctrl-C during a slow refresh drops the reply, and the session and store keep the spent token.</li><li>Under 0020 F14's "abandoned" rule, a waiter sharing that refresh re-runs it at once with the same spent token.</li></ul> | Repro `TestZZ_CanceledRefreshLosesRotation` and `TestZZ_AbandonedRefreshWaiterResendsSpent`: `sent: [old-refresh old-refresh]; family revoked: true`. | none; F14's fix makes the resend immediate |
| F4 | C1 | `api_error.go:199-203`; `internal/wire/post.go:73-80`; `internal/transport/transport.go:43-55` | **A stalled error body hangs the call.** <ul><li>`ClassifyHTTPError` reads a non-200 body with `io.ReadAll`, before `Post` installs the idle-limited `ReplyReader`.</li><li>Since 0021 D2 removed `DefaultClient`'s 330 s total timeout, a server that sends error headers and then stalls blocks `Generate` until the caller's deadline, or for ever without one.</li><li>`WithRetry` cannot help: the attempt never returns.</li></ul> | Repro `TestZZReproStalledErrorBody` (idle limit 100 ms, caller deadline 3 s): `status 200: returned after 100ms`, `status 503: returned after 3s … (error body unreadable: context deadline exceeded)`. Code re-read. | Incomplete 0021 D2, T13 |
| F5 | C2, W2 | `api_error.go:246-266`; `internal/wire/chatcompletions/chatcompletions.go:248-259`, `:339-346` | **An error inside a 200 reply is misclassified and mislabelled.** <ul><li>OpenRouter-style gateways (Kilo, Hugging Face, Together, OpenCode's chat route) put the HTTP status in `error.code`. W5 passes it to `ClassifyStreamFailure` as a type string, which uses status 500.</li><li>So a 400 overflow, 401, 402 credits, 403 moderation and 429 all become retryable `ErrProviderUnavailable`, and `WithRetry` resends each up to `MaxAttempts` times.</li><li>`APIError.Provider` is `"chat completions"`, against its doc ("names the provider, or gateway/route"). So the Kilo and OpenCode quota and region rows never apply.</li></ul> | Repro `TestZZReproGatewayErrorIn200` (Kilo, `WithRetry` 3): `serverHits=3 overflow=false quota=false auth=false unavailable=true … provider unavailable: chat completions stream 400: …maximum context length is 8192 tokens`. `TestReproGatewayErrorCodes`: 402 gives `quota=false … retryable=true`; 429 gives `rateLimited=false`; `FreeUsageLimitError` gives `quota=false`. | Incomplete 0021 W5 |
| F6 | P2 | `internal/wire/reauth.go:28-30` | **The rerun after a refused key never happens on Gemini.** <ul><li>`Reauth` reruns only on HTTP 401.</li><li>Gemini refuses a key with HTTP 400 `API_KEY_INVALID`, measured live in the 0020 F23 amendment. That reply is classified `ErrAuthFailure` but is never rerun.</li><li>So a `CommandToken`, Gemini's only renewable credential, is never run again. 0017 D3 and the README promise it is.</li></ul> The `llmtest` re-auth check passes because it sends a 401. | Repro `TestZZRepro_GeminiReauthOnInvalidKey`: `requests=1 invalidations=0 authFailure=true err=… gemini HTTP 400 API_KEY_INVALID`. Code re-read. | Incomplete 0020 F2 |
| F7 | W1 | `chatcompletions.go:261-305`; `wire/messages/messages.go:144-185`; `wire/generatecontent/generatecontent.go:140-180` | **A cut answer that holds only reasoning is a success with no text.** <ul><li>On Chat Completions, Messages and generateContent, a reply with reasoning, no content and a `length` finish keeps the reasoning item as output.</li><li>`Generate` succeeds, and `GenerateText` returns `("", nil)`.</li><li>Responses and Interactions return `ErrIncomplete` for the same event.</li></ul> The README says a truncated answer is `ErrIncomplete`, and excepts only a cut text answer. | Repro `TestReproReasoningOnlyCut`: `chat: ok finish=length text= output=[{"Text":"thinking about it",…}]`, the same for messages and generatecontent; `responses: err=… incomplete=true reason=max_output_tokens`. Code re-read (chat). | none |
| F8 | W3 | `generatecontent.go:186-191`; `messages.go:99-111` | **Gemini call ids from 0021 W10 break replay on the Messages wire.** <ul><li>generateContent ids are `name#index`. The `#` fails Anthropic's tool id pattern `^[a-zA-Z0-9_-]+$`.</li><li>The index restarts with every reply, so `get_weather#0` repeats across turns.</li><li>A conversation moves wire through OpenCode's per-request `Model` (0021 W7), or a caller's provider fallback. The Messages encoder sends the ids as they are.</li></ul> Before W10 the id was the bare function name, which Anthropic accepts. | Repro `TestReproCrossWireCallID`: `"id":"get_weather#0"`, `"tool_use_id":"get_weather#0"`; `TestReproGeminiIDsAcrossTurns`: both turns `CallID:get_weather#0`. Anthropic's refusal is its documented rule; not run live. | Introduced by 0021 W10 |
| F9 | Z1 | `wizard/model_select.go:36-50` | **The wizard loops for ever on a listing that recommends nothing.** <ul><li>When a live listing recommends nothing (0021 C13) and the `Prompter` accepts defaults, as a `--yes` or headless consumer does, `selectModel` asks again without end.</li><li>It takes no `ctx`, so cancellation does nothing, against R40.</li><li>`TextPrompter` stops only through its end-of-input rule.</li></ul> | Repro `TestZZ_SelectModelIgnoresCtx` (Kilo listing of only `kilo-auto/small` and `kilo-auto/balanced`, ctx 300 ms): `ctx expired 1.7s ago; ConfigureLLM still running: 92784295 Input calls, 92784293 notices`. Code re-read: no ctx parameter, `continue` on an empty `Recommended`. | Incomplete 0020 F6; the path came with 0021 C13 |
| F10 | Z2 | `wizard/auth.go:298-327` (`keepExistingOAuth`); `catalog/discovery.go:104` | **A kept session refreshes outside the caller's HTTP client.** <ul><li>`keepExistingOAuth` never gives a kept session `Options.HTTPClient`.</li><li>`catalog.List` never hands its `WithHTTPClient` to a session source.</li><li>So when the listing refreshes an expired kept session, the token request bypasses the caller's proxy, TLS roots or test transport. The listing itself does not.</li></ul> | Repro `TestZZ_KeptSessionRefreshBypassesClient` (Grok session kept and expired, Discover on): `Options.HTTPClient saw: [api.x.ai/v1/models]`, `token endpoint reached by another client: 1 request(s)`. | Incomplete 0020 F20 (0016 D8) |
| F11 | C3 | `retry.go:177-179`; `internal/wire/post.go:57-66`; `settings.go:199-208` | **Failures that can never succeed are retried, with no kind.** Any `*url.Error` counts as a failure to send, including ones raised before anything is sent: <ul><li>a base URL with no scheme (`localhost:11434`, a common Ollama typo);</li><li>`%zz` in the base URL;</li><li>a key with a trailing newline;</li><li>a CR or LF in `WithClientInfo`.</li></ul> Each comes back with no kind (R25), after `MaxAttempts` tries that never reach the server. R23 says an invalid value is `ErrInvalidRequest` from `New` or `Generate`. | Repro `TestZZReproDeterministicURLErrorRetried`: four cases give `attempts=4 serverHits=0 kind=NO KIND`, with `first path segment in URL cannot contain colon` and `invalid header field value for "Authorization"`. | Refines 0020 Q2 and F9 |
| F12 | C4 | `api_error.go:376-398`, `:216` | **Gemini's retry delay is ignored.** <ul><li>Gemini sends no `Retry-After` header. Its 429 body carries `google.rpc.RetryInfo.retryDelay`, for example `"52s"`. That is never read, so `RetryAfter` is 0.</li><li>`WithRetry` then backs off 1 s and 2 s, which cannot succeed, and a scheduler loses the delay it was given.</li><li>A per-day quota is treated as a retryable rate limit.</li></ul> | Repro `TestZZReproGeminiRetryInfo`: `RetryAfter=0s retryable=true … gemini HTTP 429 RESOURCE_EXHAUSTED`. The body shape is Google's documented one, with no live capture. | none |
| F13 | A3 | `auth/tokenstore_file.go:256-258`, `:325-345` | **A stale-lock takeover can still give two holders.** <ul><li>Waiter W1 judges a dead holder's lock stale, then pauses. W2 takes it over legitimately.</li><li>W1 renames W2's live lock aside. W3 creates a new lock in the gap, so W1's `os.Link` put-back fails with `ErrExist`, and W1 deletes W2's lock.</li><li>W2 and W3 now both refresh with one refresh token: what 0016 A2 exists to prevent.</li></ul> Narrow trigger: a dead holder, two waiters, and a third contender in that gap. | Repro `TestZZ_TakeoverRaceTwoHolders`, driven through the `lockBeforeTakeover` and `lockRename` seams: `W3 err=<nil> holds=true; W2 token still on disk: false`. | Incomplete 0020 F12 (Q4 a) |
| F14 | D1, T1 | `docs/guides/migrating-from-mcplib.md:338-342`, `:344-346`; `scripts/check_parity_map.py:157` | **Eight mapping rows name a function that does not exist, and the parity gate passes them.** <ul><li>`llmprovider.StaticClaude`, `StaticGemini`, `StaticGrok`, `StaticHuggingFace`, `StaticKilo`, `StaticOpenAI`, `StaticOpencodeGo` and `StaticOpencodeZen` map to `StaticModels(ProviderX)`. The function is `catalog.Static(llmprovider.ProviderX)`.</li><li>The gate resolves a bare name if it is any exported name, field or method anywhere, so it matches the field `Descriptor.StaticModels`.</li></ul> | `grep -rn 'func StaticModels'`: nothing (re-run by the owner's agent). A strict re-resolution flags exactly these 8 rows; the gate prints `0 problem(s)`. | Incomplete 0020 F56, 0021 Z9 |
| F15 | T2 | `scripts/go-precheck.sh:44` | **The precheck passes a commit that only deletes a Go file.** <ul><li>The `[ -f "$f" ]` filter drops a deleted file, so the script checks nothing and exits 0, even when the deletion breaks the package.</li><li>The machine-wide agent commit gate runs this script.</li></ul> | Repro: with `llmprovider/capabilities.go` removed, `go build` fails `undefined: Capabilities`, while `GO_PRECHECK_SKIP_VULN=1 ./scripts/go-precheck.sh llmprovider/capabilities.go` prints `no Go files to check.`, rc=0. Code re-read. | none |
| F16 | T3 | `scripts/test_gates.py:88`, `:121` | **The gate self-test does not exercise two gates' main checks.** <ul><li>dep-check's planted module is also caught by the `go.mod` check, so its per-package `check()` is never needed.</li><li>parity's `TBD` cell has no spans, so `unresolved()` is never used.</li></ul> Either check can be disabled and `make gate-selftest` stays green. | Repro: with `check()` returning `[]`, `test_dep_check_test_only_module` is OK, yet only `check()` catches a non-wizard import of `golang.org/x/term`. With `unresolved()` disabled, `test_parity_check_placeholder_cell` is OK, while a planted `catalog.SearchEverything` passes the broken gate and fails the real one. | Incomplete 0021 Z9 |

### C. Low: correctness, hygiene, documentation and tooling

**Core, errors and retry.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F17 | C5 | `internal/redact/redact.go:45` | `reKV` puts `\b` before each key, so a snake_case or camelCase prefix hides it: `openai_api_key=`, `"x_api_key"`, `app_secret:`, `"apiSecret"`, `db_password=`, `"subscription_key"` come back unredacted. Incomplete 0020 F8 and 0021 Z2. | Repro `TestZZReproRedaction` |
| F18 | C6, and core's note on auth | `api_error.go:123-126`; `wire/finish.go:30-32`; `wire/responses/responses.go:129-134`; `auth/oauth_session.go:96-97`, `:558` | `APIError.Reason`, the service's raw finish or incomplete reason, is neither stripped of control characters nor bounded, and reaches `Error()`. The token endpoint's error body is redacted but not stripped. Incomplete 0021 Z3 (R35). | Repro `TestZZReproReasonControlChars`: `ESC=true BEL=true len=2093`; the auth half is read |
| F19 | C7 | `settings.go:116-124`, `:199-208` | `WithReasoning` is never validated: `New` accepts an unknown effort or a negative budget and sends it, while the same value on a `Request` is refused (R23). | Repro `TestZZReproInvalidDefaultReasoning`: `New err=<nil>`, body `"reasoning_effort":"extreme"` |
| F20 | C8 | `wire/post.go:51-54`; `contract.go:209-213` | A `Tool.Schema` that cannot be marshalled gives an error with no kind, and `validate` never checks it (R23, R25). | Repro `TestZZReproUnmarshalableSchema`: `kind=NO KIND … unsupported type: chan int` |
| F21 | C9 | `redact.go:36` | `reAuth` redacts ordinary words after "bearer", "basic" and "token": `"Invalid bearer token"` becomes `"Invalid [REDACTED]"`. Diagnostics are lost (0021 Z2). | Repro `TestZZReproRedaction` |
| F22 | C10 | `retry.go:144-147`, doc `:23-26`, `:51-53` | A wait the server asked for gets its jitter after the `<= MaxDelay` check, so a wait can exceed `MaxDelay`, against "MaxDelay caps each wait" (0021 T7). | Repro `TestZZReproServerWaitOverMaxDelay`: `MaxDelay=100ms; longest wait over 20 runs=338ms` |
| F23 | C11 | `api_error.go:133-139`; `provider.go:28-30` | A retryable 408, or a 409 on openai and claude (0021 T14), also unwraps to the legacy `ErrInvalidRequest`, whose doc says retrying "can never succeed". | Repro `TestZZReproDualKind`: `openai 409: Retryable=true isInvalidRequest=true isUnavailable=true` |
| F24 | C12 | `wire/post.go:45-48` vs `api_error.go:196` | `Post`'s doc says it decodes "a 2xx reply", but only 200 passes `ClassifyHTTPError`. A 201 is a terminal `ErrInvalidRequest`. | Repro `TestZZRepro201`; code re-read |

**Wire.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F25 | W4 | `generatecontent.go:110-135` | `promptFeedback.blockReason` is not decoded, so a prompt Gemini blocks is `ErrIncomplete` with `Reason ""` (0021 W3). | Repro `TestReproGeminiEmptyReasons` |
| F26 | W5 | `generatecontent.go:215-216` | `MALFORMED_FUNCTION_CALL` and `UNEXPECTED_TOOL_CALL` map to `stop`, so such an empty answer reports `Reason "stop"` (0021 W3). | Repro `TestReproGeminiEmptyReasons` |
| F27 | W6 | `internal/wire/tools.go:15-20` | A typed-nil schema (a nil map or nil `json.RawMessage`) goes out as `input_schema: null` or `parameters: null`, which Anthropic refuses (0021 W11). | Repro `TestReproTypedNilSchema` |
| F28 | W11, P8 | `wire/responses/responses.go:75-100` | A non-streamed 200 with `"status":"failed"` becomes terminal `ErrIncomplete` "no content", and the vendor's error is lost. The same failure in a stream is retryable, with its message. Affects Grok, OpenAI with a key, and OpenCode's responses route. | Repro `TestReproResponsesFailedAsymmetry`, `TestZZRepro_GrokStatusFailed`. Whether a synchronous call returns this body was not captured. |
| F29 | W7 | `chatcompletions.go:226-231` | Chat Completions drops `message.refusal`: a refusal is "no content" with `Reason stop`, where Responses keeps the text with `content_filter` (0021 W4). | Repro `TestReproChatRefusalField` |
| F30 | W8 | `chatcompletions.go:228` | A reply whose `content` is an array of parts fails to decode (`Content string`). Which gateways send it was not captured. | Repro `TestReproChatContentParts` |
| F31 | W9 | `responses.go:174-179` | Empty function arguments stay `""` on Responses (both paths), where every other decoder gives `"{}"` (0021 W1). | Repro `TestReproEmptyArgsPerWire` |
| F32 | W10 | `chatcompletions.go:76-147` | Reasoning not followed by an assistant message is held over a user turn and attached, signature included, to the next turn's assistant message. F7's reasoning-only replies make this reachable. | Repro `TestReproChatPendingReasoningAcrossTurns` |
| F65 | Q5 research | `generatecontent.go:86`, `:97` (`Contents`, from `:43`) | generateContent never sends a call's id back: `functionCall` and `functionResponse` carry only the name, so calls to one function in a turn pair by name and position alone. <ul><li>The reference client pi sends `id` on both for Gemini 3 and later, and for `claude-` and `gpt-oss-` models on Google's API (`packages/ai/src/api/google-shared.ts:165-171`, `requiresToolCallId`).</li><li>OpenCode's google route serves `gemini-3.x` models (the static catalog has `gemini-3.5-flash-lite` and `gemini-3.8-flash`).</li><li>0021-PLAN excluded it: "No live evidence shows the service needs it back".</li></ul> | Read, with reference-client evidence. Needs a live two-call round trip on a `gemini-3.x` model through OpenCode's google route. |

**Providers.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F33 | P3 | `catalog/model_metadata.go:140-150`, from `opencode.go:504` | OpenCode's metadata lookup at generation time ignores `WithClientInfo`, and sends the module's own User-Agent (0020 F37). | Repro `TestZZRepro_OpencodeMetadataUserAgent` |
| F34 | P4 | `providers/openai/chatgpt_listing.go:89-95`, `:128` | The ChatGPT listing reads its body with no limit (a 40 MiB body decodes), and its decode and empty-catalog errors have no kind (0021 C3, R25). | Repro `TestZZRepro_ChatGPTListingKinds` |
| F35 | P5 | `providers/ollama/ollama.go:204-209`, through `catalog/discovery.go:399-416` | Ollama's `ListModels` errors (401, 503, unreachable) have no kind. | Repro `TestZZRepro_OllamaListingKinds` |
| F36 | P6 | every `ListModels` through `catalog.List` | A listing that gets 401 does not rerun the credential, and silently returns the static catalog with a nil error. Only the ChatGPT listing reruns (F38 of 0020). | Repro `TestZZRepro_TogetherListing401`: `err=<nil>; listing requests=1 invalidations=0` |
| F37 | P7 | `openai.go:156-157`, `grok.go:132-133`; `auth/oauth_session.go:386-410` | The first provider built on a session fixes its client and logger. `New` always passes a default client and a discard logger, and `UseHTTPClient`/`UseLogger` set them only when unset, so a later `New(…, WithHTTPClient, WithLogger)` changes nothing (0020 F20, F48). | Repro `TestZZRepro_SessionSharingFirstProviderWins` |
| F38 | P9 | `providers/gemini/gemini.go:106-110`, `interactions.go:100-113` | Gemini declares Reasoning `Supported`, but for `gemini-2.5-flash-lite`, in the static catalog, low and medium effort send no thinking setting. Google documents that model's default as thinking off. Same pattern as 0020 F43. | Repro `TestZZRepro_GeminiFlashLiteReasoning`; the model's default is documented, not measured |
| F39 | P10 | `ollama.go:21-22`; `kilo.go:30-31`; `grok.go:23-25`; `identification.go:22-23` | Four doc comments contradict the code or the records: <ul><li>Ollama's `ToolChoiceNone` sends no tools (F40);</li><li>Kilo's doc omits that case;</li><li>Grok's says neither instructions form was measured, but 0021 L2 measured both;</li><li>`WithSessionID` omits the ChatGPT backend.</li></ul> | Read; `TestOllama_ToolChoiceNoneSendsNoTools` |

**Auth.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F40 | A4 | `auth/vendor_session.go:92-104` | `os.Root` refuses a symlinked `auth.json` that points outside its directory. The Grok CLI writes through one (dotfile and `GROK_HOME` overlays). The error advises `grok login`, which cannot fix it. | Repro `TestZZ_VendorSymlinkedAuthFile`: `path escapes from parent` |
| F41 | A5 | `vendor_session.go:63-65` | Codex rewrites `auth.json` in place (truncate, then write). A read in that window is a terminal `ErrAuthFailure` with login advice, where one re-read would succeed. | Repro, torn file: `unexpected end of JSON input … (kind auth)` |
| F42 | A6 | `auth/tokenstore_file.go:195`, `:71`, `:99` | Windows only: Go's `os.Open` does not share delete access, so `Save`'s rename fails while another process reads the file. The resave recovers it, but each such failure is one of the two F1 needs. | Read; Windows cannot run here |
| F43 | A7 | `oauth_session.go:459`, `:472-481`; `tokenstore_file.go:244`; `oauth_device.go:347-351`; `kilo_device.go:106-108` | Errors with no kind (R25): a refresh transport failure, a 200 refresh reply that does not decode, a lock file that cannot be created, and the device "denied" and "expired" errors. | Repro `TestZZ_RefreshErrorKinds`; the device half is read |
| F44 | A8 | `oauth_session.go:68-72` vs `:704-720` | `ValidateOAuthSession`'s doc says the token URL may be empty "because the refresh derives it from the issuer". Since 0020 F45 that holds for the two built-in issuers only, so a custom-issuer session without one validates and then can never refresh. | Repro `TestZZ_ValidateAcceptsUnrefreshable` |
| F45 | A9 | `auth/oauth_revoke.go:58-59`, `:104-106` | An older Grok session with no issuer refreshes against `auth.x.ai`, but cannot be revoked: discovery runs on an empty issuer, with a kindless error, not `ErrUnsupported` (gap in 0020 F45). | Repro `TestZZ_RevokeLegacyGrokSession` |

**Catalog and wizard.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F46 | Z3 | `wizard/model_select.go:90-95` | `selectRecommended` makes `Existing.Model` the default without checking `Existing.Provider`. 0007 §4.3 says only for its own provider. | Repro `TestZZ_ExistingModelOfOtherProviderIsDefault` |
| F47 | Z4 | `wizard/configure.go:417` | A ChatGPT session's `Recommended` is the whole Codex listing, uncapped, against "at most MaxListed". | Repro `TestZZ_ChatGPTRecommendedUncapped`: 14 models, 15 menu rows |
| F48 | Z5 | `wizard/auth.go:431-447` | A pasted ChatGPT access token is saved with no `AccountID` or `FedRAMP`, though `checkAccessToken` decodes the claim, so requests carry no account id. The effect on a multi-workspace account was not measured. | Repro `TestZZ_PastedAccessTokenDropsAccount` |
| F49 | Z6 | `wizard/text_prompter.go:337-340`, `:350-357` | `Secret` never surfaces a failed write, against the type's own comment; the failure appears at the next prompt. | Repro `TestZZ_SecretIgnoresWriteError` |
| F50 | Z7 | `text_prompter.go:405-408`, `:453-459` | In raw mode a read error in mid-entry returns the partial entry, and `Secret` keeps it: a hang-up mid-paste yields a truncated key. | Repro `TestZZ_ReadMaskedPartialOnError`: `"sk-partial-k"` |
| F51 | Z8, D4 | `wizard/configure.go:81-84` | `Options.Profile`'s doc omits Together, which is ranked (0020 F4); `catalog.WithProfile` and the README include it (0020 F58). | Read; `go doc ./wizard Options` |
| F52 | K1 | `catalog/discovery.go:444-448`; `wizard/configure.go:266` | `ValidateOllamaURL` and `ValidateOllamaURLWith` do not trim a trailing slash, so they request `//api/version`; the listing trims (0020 F30). | Repro `TestZZ_ValidateOllamaTrailingSlash` on a router that does not clean paths; not run on a real Ollama |
| F53 | K2 | `catalog/model_metadata.go:91-98`, `:154-156` | `Metadata.ReasoningEfforts` returns the process-wide cache's own slice; a caller that edits it poisons later lookups (R29). The in-tree caller only reads. | Repro `TestZZ_ReasoningEffortsAliasesCache` |
| F54 | K3 | `catalog/discovery.go:63-65` | `Catalog.Usable` is documented as "in listing order", but Kilo's is cheapest first and Hugging Face's fastest first (0007's table). | Read |

**Test harness, tooling and documentation.**

| ID | Source | Where | Finding | Evidence |
|---|---|---|---|---|
| F55 | L1 | `llmprovider/llmtest/llmtest.go:268` | With `ForcedToolChoice` Unsupported and no `Harness.ToolCall`, the loop skips the R11 refusal check, which needs no handler. No built-in is affected. | Repro: a scratch provider that sends a forced choice it declares Unsupported gets 2 R11 failures with `ToolCall` set, and none without |
| F56 | L2 | `providers/opencode/llmtest_test.go:14`, `:23` | OpenCode runs the conformance suite on its chat and messages routes only. Since Gemini moved to Interactions, the generateContent wire runs through `llmtest` in no provider. | Repro: zen-google and go-responses harnesses both pass when added |
| F57 | L3 | `llmtest/llmtest.go` | Obligations with no harness check: <ul><li>`ListModels` (identity, cancellation, token, classification);</li><li>R16's default header and `TokenBearer`;</li><li>`TokenInvalidator` (0021 D5);</li><li>403 to `ErrNotPermitted`, quota, 413 to overflow;</li><li>`RetryAfter`;</li><li>an HTTP failure being an `*APIError` (R24);</li><li>R35 redaction, R18 foreign option, R27 prefix.</li></ul> F6 is one result of this. | Read |
| F58 | L4 | `llmtest/fake.go:38`, `:103` | `Fake.Reply(nil)` is accepted, then panics in `Generate`. `Requests()` says copies, but `Tool.Schema` is shared. | Read |
| F59 | T4 | `internal/ambientcheck/ambientcheck_test.go:35` | The ambient-state check misses `os.UserConfigDir`, `os.UserCacheDir`, `os/user.Current` and `syscall.Getenv` (0020 F25). None is used today. | Repro: all three planted, `go test ./internal/ambientcheck/` ok |
| F60 | T5 | `Makefile` `vuln`; `scripts/go-precheck.sh:64`, `:142`; `ci.yml` | govulncheck drifts: CI pins `v1.8.0`; `make vuln` and the precheck run any installed binary; the hint says `@latest` (0020 F53's class). | Read |
| F61 | T6 | `scripts/check_records.py:40` | records-check and `--next` scan only `docs/decisions` and `docs/reports`, while AGENTS.md says "anywhere under `docs/`". | Read |
| F62 | D2 | `docs/guides/api-standards.md:38-39`; `docs/architecture.md` package table | R2's import table lets `internal/wire` and `internal/wire/<format>` import only `llmprovider`, but both import `internal/transport` (since `3a584df`); no gate checks it. architecture.md's "Depends on" column is stale for six providers, `catalog` and `wire/responses`. | `go list` import dump |
| F63 | D3 | `README.md:190` | "The current release is `v1.0.0`"; the tags reach `v1.2.1`. | `git tag --list`; re-read |
| F64 | D5 | `docs/architecture.md:520-523` | The G-wire section says 16 cases and 100 golden files; there are 18 cases and 114 files. | `find … -name '*.json' \| wc -l` |

**Checked and found sound.** These came out clean, and the PLAN need not
touch them:

* **Core:**
  * `validate` (0020 F22 holds);
  * `WithRetry`'s deadline check, its join on cancel, the single retry after
    a reply, a `Retry-After` over `MaxDelay` returning at once, and
    overflow clamping;
  * `ListModels` passes through `WithRetry`, and `NativeStreaming` is
    stripped (0020 F32);
  * the stream fallback;
  * `For` and scoped options;
  * every `Settings` accessor has a reader;
  * the tokens hide their secret everywhere;
  * `CommandToken`'s single flight and output cap;
  * the transport timeouts of 0021 D2, `ParseRetryAfter`, and
    `ReplyReader`.
* **Wire:**
  * tool choice on every wire;
  * the Responses stream: event limit, long lines, CRLF, `[DONE]`, mid-stream
    errors;
  * reasoning replay per `Format`;
  * thought signatures;
  * empty roles;
  * `Model` and `Usage` decoded on every wire;
  * `kiloendpoint`;
  * the goldens, except the one F8 questions.
* **Providers:**
  * every option reaches every provider that should honour it;
  * every provider wraps generation in `Reauth`;
  * the per-route headers and R16 overrides;
  * the ChatGPT headers;
  * OpenCode's routing order;
  * the Capabilities declared by 0020 F43 and 0023;
  * the registry and descriptors;
  * the refusal of sessions by the providers that take keys only.
* **Auth:**
  * 64 goroutines over 4 sessions sharing one store, with invalidations,
    3 runs under `-race`, sent no token twice;
  * 0020 F1's single failure, 0021 T2, T3, T4, T8 and T10 hold;
  * the device flows;
  * the browser login's state, method and `id_token` checks;
  * the store's write path;
  * Unix `ownerperm`.
* **Catalog and wizard:**
  * `List`'s degrade-to-static contract and C13 rule;
  * pagination bounds;
  * body caps;
  * the metadata cache (single flight, failure memory, detached fetch), which
    survived a 40-goroutine stress race-free;
  * ranking against 0009 and 0021 D3;
  * `Search`;
  * `OptionsFromEnv`'s opt-in;
  * the wizard's environment reads, all through `LookupEnv`;
  * the auth flows and their 0020 fixes;
  * `TextPrompter`'s end-of-input rule and raw-mode keys.
* **Harness, tooling and docs:**
  * every built-in id passes `llmtest.Run` with every optional check on;
  * the harness catches what it claims;
  * coverage, generate, api and records gates;
  * CI matches AGENTS.md;
  * the documented examples compile;
  * every package-qualified name in the docs resolves.

## Decision Drivers

* **Credential safety first.** A user signed out (F1, F3, F13) or a token
  sent to the wrong vendor (F2) outranks everything else, as it did in
  0020.
* **Promises in the records hold.** A behaviour a record or the README
  promises is a defect when it is missing:
  * the rerun (F6);
  * truncation is `ErrIncomplete` (F7);
  * an error's provider label and kind (F5, F11);
  * cancellation (F9);
  * the caller's client (F10, F37).
* **No re-billing and no futile retries.** An error that cannot succeed
  must not be resent: F5, F11 and F12.
* **Fixes do not create the next pass.** F8 came from a fix. Each fix lands
  with a check that fails without it, on the path that broke, not only the
  case at hand.
* **The gates guard what they claim.** A gate that passes a broken input
  (F14, F15, F16) is worse than none, because it is trusted.
* **API compatibility (R48).** Fixes stay inside v1: additive exported API
  only, checked by `api-check`. A wire change follows R45, with its goldens
  in the same commit. F2's refusal is a behaviour change, so Q2 asks about
  it.

## Considered Options

* **A. Remediate in a 0026 PLAN, by severity, after the owner answers seven
  design questions.**
* **B. Fold each finding into the record that owns its area** (0016, 0017,
  0020, 0021).
* **C. Fix the high and medium findings now; record the low ones as
  accepted.**
* **D. Defer everything to a v1.3 planning cycle.**

## Decision Outcome

Chosen option: **"A. Remediate in a 0026 PLAN, by severity, after the
owner answers seven design questions"**. One record keeps the 65 findings,
their evidence and their fixes together, as 0013 and 0020 did. Seven fixes
depend on choices that only the owner can make. The rest have one obvious
fix, which the PLAN carries. The resulting release is `v1.3.0`, because F2
and F11 refuse inputs that `v1.2.1` accepted.

### 1. Phases for the PLAN, once this record is accepted

1. **Credentials:** F1, F3, F13, F2, then F40–F45.
2. **Errors and retry:** F4, F5, F11, F12, then F17–F24.
3. **Answers on the wire:** F7, F8, then F25–F32 and F65 (after its live
   check).
4. **Provider wiring:** F6, F36, F37, then F33–F35, F38 and F39.
5. **Wizard and catalog:** F9, F10, then F46–F54.
6. **Harness, gates and docs:** F14, F15, F16, then F55–F64. F57's checks
   land with the fixes they would have caught, F6's first.

Each phase ends green, and is releasable alone.

### 2. Design questions

**Answered 2026-10-06,** six of seven. The owner: "Q1 file locks. Q2 yes
Q3 yes Q4 yes Q5 assess this evaluate the codebase, analyze options,
provide me with 3 choices. Q6 yes Q7 yes. give me the options. make
recommendations for all questions."

| Question | Answer | Recorded as |
| :--- | :--- | :--- |
| Q1 | "file locks" | (a): the store's known refresh token, the detached refresh, and OS locks for F13. "File locks" names (a)'s lock half. The other two halves have no alternative in (a), and are read as included. |
| Q2 | "yes" | (a): refuse a foreign session at `New` |
| Q3 | "yes" | (a): rerun on `ErrAuthFailure`'s kind, listings included |
| Q4 | "yes" | (a): `ErrIncomplete` for a reasoning-only cut answer |
| Q5 | "Q5 A, write the plan" (2026-10-06, after the three choices below) | (A): remap ids in the encoder |
| Q6 | "yes" | (a): capture a live Gemini 429 first |
| Q7 | "yes" | (a): refuse unsendable inputs at `New`, and classify them in `WithRetry` |

Every question is answered, each with its recommended choice, and this
record is `accepted`. The 0026 PLAN is written next; no fix starts before
it is approved.

**Q1. Which refresh token the store holds (F1, F3, F13).**

* (a) **Recommended.** Three changes:
  * The session records the refresh token the store is known to hold.
    `loadRotated` and `persistRotation` compare against that, so any number
    of failed saves is recovered.
  * The refresh runs under `context.WithoutCancel(ctx)`, within the
    existing 15 s bound. Each caller and waiter stops waiting on its own
    context, and the leader still adopts and saves a rotation it was sent
    (F3). 0021 C1 did the same for the metadata fetch.
  * For F13, the store takes the refresh lock with OS locks: `flock` on
    Unix, and `LockFileEx` through the `mkwinsyscall` bindings that 0010
    added. That is 0020 Q4 (b), now that the bindings exist.
* (b) The minimal F1 fix: keep the oldest spent token while one is set. F3
  as in (a). F13: serialise takeovers with an `O_EXCL` marker file, and
  re-check the owner token before renaming.
* (c) As (b), with F13 recorded as an accepted narrow risk.

**Q2. A foreign session given to Grok or OpenAI (F2).**

* (a) **Recommended.** `New` refuses it with `ErrUnsupported`, as `kilo.New`
  does. An OpenAI session with no issuer is a ChatGPT session.
  * This is a behaviour change: a misconfiguration that ran before now
    fails at `New`. That is the point, and the release notes say so.
* (b) Accept it, and log a warning through `WithLogger`.

**Q3. What reruns the credential (F6, F36).**

* (a) **Recommended.** `Reauth` reruns when the error matches
  `ErrAuthFailure`, whatever the status, and the catalog listings go through
  it too. That covers Gemini's 400, any vendor that refuses a key with 400
  or 403, and a listing's 401.
  * An `ErrNotPermitted` 403 also matches `ErrAuthFailure` through the
    legacy sentinel, so the test uses the kind, not `errors.Is`. Otherwise
    an entitlement refusal would rerun a command for nothing.
* (b) Add Gemini's `API_KEY_INVALID` 400 as a second trigger, and leave the
  listings as they are.

**Q4. A cut answer that holds only reasoning (F7).**

* (a) **Recommended.** `ErrIncomplete` with `Reason "length"` on Chat
  Completions, Messages and generateContent, as Responses and Interactions
  already do and the README promises. A `llmtest` check pins it on every
  provider.
* (b) Keep it a success, and document that `GenerateText` may return `""`
  with a `length` finish.

**Q5. Call ids that cross wires (F8).** The owner asked for an assessment
of the codebase and three choices. The first draft's two options are
replaced by the analysis below.

*Where ids come from, and where they go back,* at `c523c65`:

| Wire | Id decoded from | Sent back as | Rule the service applies |
| :--- | :--- | :--- | :--- |
| Responses | `call_id` (`responses.go:148`, `:176`) | `call_id` (`:59`, `:66`) | none documented |
| Messages | `tool_use.id` (`messages.go:130`, `:174`) | `id`, `tool_use_id` (`:102`, `:109`) | `^[a-zA-Z0-9_-]+$`, at most 64 (Anthropic's documented pattern; pi's `anthropic-messages.ts:1215-1217`) |
| Chat Completions | `tool_calls[].id` (`chatcompletions.go:295`) | `id`, `tool_call_id` (`:99`, `:124`) | the gateway's upstream; OpenCode pads Mistral's to 9 alphanumerics |
| Interactions | the step's `id` (`interactions.go:137`, `:184`) | `id`, `call_id` (`:82`, `:89`) | none documented |
| generateContent | `functionCall.id`, else `name#index` (`generatecontent.go:169`, `:186-191`) | nothing: name only (F65) | pi sends ids for Gemini 3 and later |

*Facts that decide it:*

* **A foreign id reaches the Messages encoder three ways:**
  * OpenCode's per-request `Model` switches route on the same `Input`
    (0021 W7);
  * a caller falls back from one provider to another with the same
    history;
  * a caller builds the history by hand.

  So the failure is in the encoder, whatever made the id.
* **`name#index` repeats across turns,** because the index restarts with
  every reply. A remap keyed only by id would merge two different calls.
  Any remap must pair each result with the latest call before it.
* **Both reference clients normalise at the encoder:**
  * OpenCode scrubs `[^a-zA-Z0-9_-]` to `_` for Claude
    (`packages/opencode/src/provider/transform.ts:224-226`), and pads ids to
    9 alphanumerics for Mistral;
  * pi remaps a call and its result together, for the target API, when the
    model changes (`packages/ai/src/api/transform-messages.ts:60-67`,
    `:136-140`). It caps them at 64 for Anthropic and Google.
* **pi also makes Gemini ids unique at the source:** `name_<time>_<counter>`
  when the service sends no id, or a duplicate
  (`packages/ai/src/api/google-generative-ai.ts:195-201`).
* **The decoded ids are pinned by goldens** (`opencode-zen-google`'s
  `items.json`, 0021-PLAN's W10 deviation). Changing them is an R45 change.

*The three choices:*

* **(A) Remap at the encoder.** **Recommended.**
  * A shared helper in `internal/wire` returns a copy of `Input` in which
    each call id that breaks the target's rule, or repeats an earlier call
    in the same request, gets a valid and unique id. That is
    `[^a-zA-Z0-9_-]` replaced with `_`, at most 64 characters, and `_2`,
    `_3`… on a collision. Each result takes the id of the latest call
    before it with its original id.
  * `messages.FromItems` uses it with Anthropic's rule.
  * The caller's items and the decoded ids are unchanged, and no golden
    moves.
  * Good, because it fixes every source of a bad id (Gemini's, a gateway's,
    a caller's), on the path that broke. That is what both reference
    clients do.
  * Good, because it is one helper and one call site, and a later rule
    plugs in: Mistral's 9 alphanumerics, or Interactions' if a live check
    shows one.
  * Bad, because a caller's own history still holds `get_weather#0` in two
    turns. A caller that keys results by CallID across a whole conversation
    can still collide. 0021 W10 fixed only one reply, and no failure has
    been seen from this.
  * Bad, because the service sees ids that differ from the caller's, which
    a log reader must know.
* **(B) Make Gemini's ids valid and unique at the source.**
  * generateContent's synthetic id becomes `<name>_<index>_<8 random hex>`,
    through a seam so tests stay deterministic. `callName` recovers the
    name from the `names` map, or from the suffix.
  * Good, because CallIDs are unique across a conversation and valid on
    every known wire.
  * Bad, because it fixes only Gemini's synthetic ids. A service-issued or
    caller-built id outside Anthropic's pattern still breaks the Messages
    wire.
  * Bad, because decoded results change: the goldens move under R45, and a
    caller matching `name#index` sees a different value.
* **(C) Both, and echo ids to Gemini (F65).**
  * (A) and (B) together. generateContent also sends `id` in
    `functionCall` and `functionResponse` when a call carries a
    service-issued id, as pi does for Gemini 3, after F65's live check.
  * Good, because it is complete: every id valid, unique, and returned to
    the service that issued it.
  * Bad, because it is the largest: (B)'s golden churn, plus a wire change
    that 0021-PLAN excluded and that needs live evidence first.

**Recommendation: (A).** It fixes F8 where the failure happens, for every
source of an id, and moves no decoded value or golden. (B)'s extra benefit,
ids unique across a caller's whole history, answers no failure seen.

F65 stands as its own finding whatever is chosen. Its live check runs in
Phase 3:
* if Gemini 3 needs ids back, F65's fix follows;
* if it also needs them unique across turns, (B) follows, under its own
  amendment.

**Q6. Gemini's retry delay (F12).**

* (a) **Recommended.** Capture a real Gemini 429 with the owner's key first,
  as 0020 Q6 (a) did for invalid keys. Then:
  * decode `RetryInfo.retryDelay` into `RetryAfter` when no header is
    present;
  * classify a per-day `QuotaFailure` as `ErrQuotaExhausted`.
* (b) Implement from Google's documented shape now, with no capture.

**Q7. Inputs that can never be sent (F11).**

* (a) **Recommended.** Both:
  * `ResolveOptions` parses the base URL (scheme and host), and refuses
    control characters in the key, client info and session id, with
    `ErrInvalidRequest` from `New`;
  * `WithRetry` treats a `*url.Error` from parsing, or an invalid header, as
    terminal with that kind. A source token with a newline then fails on
    the first attempt.

  New behaviour: a malformed base URL fails at `New`, not at the first call.
* (b) Only the classification in `WithRetry`. `New` still accepts the
  input.

### 3. Recommended without a question

* **F4.** `Post` wraps the body in the idle-limited reader before
  `ClassifyHTTPError`.
* **F5.** A numeric `error.code` from 400 to 599 is classified as that HTTP
  status through `classifyAPIError`, overflow check included. The decoder
  receives the provider label.
* **F9.** `selectModel` and `selectFallbacks` take `ctx`, and check it on
  every pass. With nothing recommended, the current model or Other is
  offered, never an endless re-prompt.
* **F10, F37.**
  * `keepExistingOAuth` and `catalog.List` hand the caller's client to a
    session.
  * `UseHTTPClient` and `UseLogger` take a non-default value even when one
    is already set.
* **F14.**
  * The eight cells name `catalog.Static(llmprovider.ProviderX)`.
  * The parity gate resolves a bare call-shaped name only against
    package-level functions and types.
  * The self-test plants a bare name that does not exist.
* **F15.** A deleted `.go` path still adds its package directory, or the
  precheck falls back to `./...`.
* **F16.** The self-test gains a non-wizard `x/term` import, and an
  unexported name in a mapping cell.
* **F38.**
  * A live check of `gemini-2.5-flash-lite` with low effort, as 0020 F43
    did for Grok.
  * Then either `BestEffort`, or a thinking budget sent for 2.5-series
    models.

### Consequences

* Good, because the two ways of losing or leaking a credential close, and
  F1 stays closed: its test covers any number of failed saves.
* Good, because errors carry their real kind and provider, so `WithRetry`
  stops buying futile retries (F5, F11, F12), and callers can act on
  overflow, quota and auth.
* Good, because the gates catch what they claim (F14–F16), and the harness
  gains the checks whose absence let F6 through (F57).
* Neutral, because F8 and Q4 change wire bodies and decoded results, and
  the goldens move with them under R45.
* Bad, because F2 and F11 refuse inputs that `v1.2.1` accepted, so the
  release is `v1.3.0`, with a note.
* Bad, because three items need the owner live:
  * the Gemini 429 capture (Q6);
  * the `gemini-2.5-flash-lite` reasoning check (F38);
  * the Windows lock behaviour under Q1 (a).
* Bad, because it is a large PLAN again: 65 findings in six phases.

### Confirmation

* **Failing first.** Each fixed finding has a test that fails on a scratch
  copy before the fix, and passes after. For each high and medium finding,
  the audit's reproduction above becomes that test. The PLAN records both
  outputs.
* **The gate is clean:**
  * `make pre-add-check`, `make lint`, and `parity-check`, `dep-check`,
    `coverage-check`, `api-check` (additive only), `generate-check`,
    `records-check` and `gate-selftest`;
  * `go test -race ./...` and `-shuffle=on`;
  * markdownlint and links.
* **Live checks, with the owner's keys:** the Gemini 429 (Q6), Gemini
  flash-lite reasoning (F38), and a `CommandToken` rerun against Gemini's
  real invalid-key reply (F6).

## Pros and Cons of the Options

### A. Remediate in a 0026 PLAN, by severity, after seven design questions

* Good, because the evidence, decisions and fixes stay in one place, as
  they did for 0013 and 0020.
* Good, because the credential findings land first, and each phase is
  releasable on its own.
* Bad, because one PLAN touches every package again. The phases keep each
  commit to one area.

### B. Fold each finding into the record that owns its area

* Good, because each fix sits beside the decision it repairs.
* Bad, because shared fixes would be decided several times, across four
  records:
  * F5's classification serves three wires;
  * Q3's rerun trigger serves every provider and listing.

### C. Fix the high and medium findings now; record the low ones as accepted

* Good, because it is smaller.
* Bad, because several low findings break documented behaviour or safety:
  * F17 and F18, redaction and control characters;
  * F27, a schema a vendor refuses;
  * F43, kindless errors;
  * F49 and F50, a truncated pasted key.
* Bad, because F57's missing harness checks are what let F6 through.

### D. Defer everything to a v1.3 planning cycle

* Bad, because F1 can sign users out and F2 can send a token to the wrong
  vendor. Both are in a tagged release that two consumers use.

## More Information

### Relationship to other records

* [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
  is the previous debugging pass, and this record follows its shape. Its
  F1, F2, F6, F8, F12, F20, F25, F30, F37, F43, F45, F48, F53, F56 and F58
  are each extended by a finding here.
* [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md):
  * D2, T7, T10, T13 and T14;
  * W1, W3, W4, W5, W10 and W11;
  * C3, C13, Z2, Z3 and Z9.

  Each has an edge recorded here. F8 is a defect the W10 fix introduced.
* [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md):
  D4, D8 and A2 are repaired by F1, F3, F10, F13 and F37.
* [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md):
  D3 is repaired by F6 and F36.
* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md):
  R16, R23, R24, R25, R27, R29, R35 and R40 are restored by F2, F11, F19,
  F20, F43, F53, F18 and F9.
* [0010-MADR-windows-stdio-oauth-tokenstore-ci.md](0010-MADR-windows-stdio-oauth-tokenstore-ci.md):
  its `mkwinsyscall` bindings are what Q1 (a) would reuse, and F42 is its
  Windows half.

### Consumers

* prepare-commit-msg requires `v1.2.1`. Its users with a ChatGPT or Grok
  sign-in are exposed to F1 and F3. Its fallback loop already moves past an
  empty answer, so F7 costs it a retry on the next model, not a bad commit
  message.
* gobble-cli requires `v1.1.1`. It is not re-audited here.

### Evidence

* **Where the scratch tests ran.** The six audits worked in the session's
  scratch space, in one shared clone each, as `zz_repro_*_test.go` files.
  The scratch space is not kept, so the quoted output is the evidence. The
  PLAN re-creates each reproduction as its failing-first test.
* **No live calls were made in this pass.** Three findings rest on vendors'
  documented behaviour, and say so:
  * F8: Anthropic's id pattern;
  * F12: Google's `RetryInfo`;
  * F38: Gemini flash-lite's default.
