---
status: in-progress
date: 2026-10-03
associated-madr: "0020-MADR-remediate-v1-debugging-pass-findings.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Remediation of the v1 Debugging Pass

Associated MADR: [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)

## Goal

Every finding F1–F59 is fixed and proven, or is recorded here as not done
with its reason. The fixes follow the owner's answers of 2026-10-03: every
design question Q1–Q7 takes its recommended answer.

When it is done:

* the repository gate passes, with `go test -race ./...` in CI (F27);
* `api-check` reports only additive changes against `v1.0.0` (R48);
* the live checks under Verification pass with the owner's keys;
* every behaviour the docs promise holds, and the docs say what the code
  does.

## Scope

### In scope

The six phases of the MADR's Decision Outcome §1, in order. Each phase
touches one area and ends green, in its own commit: agents stage, the owner
commits.

| Phase | Area | Findings |
| :--- | :--- | :--- |
| 1 | credentials | F1, F12, F14, F2, F8, F44, F45, F47, F48 |
| 2 | answers | F3, F9, F10, F11, F39, F7, F24 |
| 3 | catalog | F4, F5, F15, F21, F28, F34, F35, F36, F37 |
| 4 | wizard | F6, F16, F17, F18, F19, F20, F49, F50, F51 |
| 5 | core hygiene | F13, F22, F23, F29, F30, F31, F32, F33, F38, F40, F41, F42, F43, F46, F52 |
| 6 | tooling and docs | F25, F26, F27, F53, F54, F55, F56, F57, F58, F59 |

### Out of scope

* 0010's open item, the Windows failing-first run of
  `TestMkdirAllAndFile_OnlyTheCurrentUser`. 0010-PLAN keeps it.
* Any exported API change that is not additive. A fix that seems to need
  one stops for a deviation.
* New providers, new features, and refactors beyond what a finding needs.
* Push and tags, which are the owner's.

## Rules for every phase

1. **Red first.** Each finding gets a test that fails on a scratch copy of
   the tree before its fix:
   * either the planted state of the finding,
   * or the unchanged code, when the test is new and the code is the
     defect.

   The FAIL line is recorded in the Execution record, and so is the PASS
   after the fix. Where the MADR quotes a scratch reproduction, that
   reproduction becomes the test.
2. **The gate before staging,** all clean:
   * `make pre-add-check`, vet for Linux, macOS and Windows,
     `go test -race -count=1 -cover ./...`, `go mod tidy -diff`;
   * `make lint` (host and Windows), `parity-check`, `dep-check`,
     `coverage-check`, `api-check` and `generate-check`;
   * markdownlint on the guides, G-wire stable, links, and the identifier
     scan.
3. **G-wire.** A change to what goes on the wire updates its goldens with
   `-update` in the same commit (R45), and the Execution record lists each
   changed golden and why.
4. **Deviations stop and prompt,** with evidence and resolutions. The
   chosen resolution is recorded here, and in the MADR when a decision
   changes, before work continues.
5. **Records cite records by full filename.** Nothing committed carries a
   hostname, account name or machine path.

## Implementation Steps

### Phase 1: credentials

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F1 | `TestOAuthSession_FailedSaveNeverReusesSpentRefresh`: a store whose first `Save` fails and whose `Load` returns the pre-rotation session; then `Token`, `Invalidate`, `Token`. It asserts the refresh tokens sent are `[old, rotated]`, and fails today with `[old old]`. A second case uses an issuer that rejects reuse, and asserts the second `Token` succeeds. | `reloadOrRefresh` takes the session's `spentRefresh`. `loadRotated` treats a stored refresh equal to it as not newer. While `spentRefresh` is set, the pending save is retried under the lock before any refresh. | `auth/oauth_session.go`, its test |
| F12 | `TestFileTokenStore_StaleLockHasOneTaker`: deterministic, through a seam that pauses between seeing the lock stale and taking it. Two takers, and at most one holds. `TestFileTokenStore_UnlockKeepsSuccessorsLock`: a holder that outlived `staleAfter` unlocks after a successor took over, and the successor's file survives. Both fail today. | Q4 (a): the lock file holds an owner token, pid and random bytes. A stale lock is taken by renaming it to a unique name; only the process whose rename succeeds goes on to the `O_EXCL` create. Unlock and the heartbeat act only while the file still holds their token. | `auth/tokenstore_file.go`, its test |
| F14 | `TestCommandToken_WaiterOutlivesLeaderCancel` and `TestOAuthSession_WaiterOutlivesLeaderCancel`: the leader's ctx is cancelled mid-fetch, and a waiter whose ctx is live gets a token. Both fail today with `context canceled`. | A waiter whose shared result is a context error, while its own ctx is live, runs the fetch itself. The leader's cancellation stays the leader's. | `llmprovider/command_token.go`, `auth/oauth_session.go`, their tests |
| F2 | `llmtest` gains a 401 check. A provider built with an `InvalidatingSource`, against a server that refuses the first token, must invalidate once and resend once. The seven providers that lack it fail, as the MADR's reproductions show. | One shared helper in `llmprovider/internal/wire`: on a 401 `*APIError` with an `InvalidatingSource`, invalidate, fetch a fresh token, rebuild the body and send once more. All nine providers use it; openai's and grok's own copies go. A Kilo device session has nothing to refresh: its `Token` after `Invalidate` is `ErrAuthFailure`, and that is what is returned (0017-MADR D2). | `internal/wire`, all provider packages, `llmtest` |
| F8 | `redact` table tests for every value the MADR lists. Each fails today. | New patterns: <ul><li>`sk-` with the `ant-`, `proj-` and `or-` variants, `xai-`, `tgp_` and `hf_`;</li><li>key names ending `_token` or `-token`, and `code`, `code_verifier`, `device_code`, `key` and `cookie`, in JSON and form bodies;</li><li>`Cookie:`;</li><li>Kilo's `{url}:{secret}`.</li></ul> Clean text keeps no allocation. | `internal/redact` |
| F44 | Three callback tests, each failing today: <ul><li>a request with no state or a wrong state does not end the login, and a later right one completes it;</li><li>`DELETE` is refused;</li><li>a pasted URL without `state` is refused.</li></ul> | Q7 (a): <ul><li>a mismatch gets 400 and the listener keeps waiting;</li><li>only GET, and OPTIONS for Grok's preflight, are accepted;</li><li>a pasted URL must carry a matching state, while a bare code still skips the check.</li></ul> | `auth/oauth_loopback.go`, its test |
| F45 | A session with an empty `TokenURL` and a non-default issuer fails to refresh, and sends nothing to `auth.x.ai`. Revocation takes the same rule. Both fail today. | The fallback to xAI's URL applies only to `DefaultGrokOAuthIssuer`; any other issuer is an error. | `auth/oauth_session.go`, `auth/oauth_revoke.go` |
| F47 | An OpenAI device `user_code` with a control character is refused. It fails today. | Apply the check Grok and Kilo already use. | `auth/oauth_device.go` |
| F48 | A provider built with `WithLogger` and a session whose save fails logs through that logger. It fails today. | `(*OAuthSession).UseLogger(*slog.Logger)`, which fills a nil `Logger` and is additive, called where providers call `UseHTTPClient`. | `auth/oauth_session.go`, `openai`, `grok` |

### Phase 2: answers

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F3 | Messages fixtures: <ul><li>`stop_reason` `max_tokens` with text gives `FinishLength`;</li><li>with a `tool_use` it gives `ErrIncomplete`;</li><li>`end_turn` gives `FinishStop`, and `tool_use` gives `FinishToolCalls`.</li></ul> Google's `finishReason` follows the same pattern. All fail today. | Decode `stop_reason` and `finishReason` and map them to `FinishReason`. A cut tool call is `ErrIncomplete`, as Chat Completions does (0012-MADR §1.5). | `internal/wire/messages`, `generatecontent`, `providers/gemini` |
| F9 | <ul><li>An empty or undecodable 200 through `WithRetry` is sent once and is `ErrIncomplete`.</li><li>A reply over 1 MiB that is valid JSON is decoded.</li><li>A transport error is still retried.</li></ul> These fail today. | Q2 (a): <ul><li>decode and empty-answer errors are `ErrIncomplete`, named by provider (R27);</li><li>the reply cap is the stream limit, 16 MiB, and above it the error is `ErrIncomplete`;</li><li>`WithRetry` retries a kindless error only when it is a `*url.Error`.</li></ul> | every wire `Decode`, providers' read limits, `llmprovider/retry.go` |
| F10 | An empty `Role` on the Responses wire goes out as `user`, with an `llmtest` check for it. It fails today. | Map `""` to `user` in `responses.Input`. Update the goldens if any changes. | `internal/wire/responses`, `llmtest` |
| F11 | Each wire's fixture with `model` and a stop status gives `Response.Model` and `FinishReason`, with an `llmtest` check. They fail today. | Decode `model` and the status in every wire. A call-only reply gives `FinishToolCalls`. | every wire, `llmtest` |
| F39 | A Responses stream `{"type":"error","code":"insufficient_quota"}` gives `ErrQuotaExhausted`. Today it gives "stream ended". | `case "error"` calls `ClassifyStreamFailure`. | `internal/wire/responses` |
| F7 | A Messages fixture: thinking with `signature`, plus `tool_use`, then replay with a tool result. The replayed assistant message starts with the thinking block, signature included. A `redacted_thinking` block replays its data. It fails today. | Q1 (a): `ReasoningItem` gains `Signature` and `Encrypted`, an additive change. Messages decodes both and replays them, ahead of `tool_use`. | `llmprovider/item.go`, `internal/wire/messages` |
| F24 | <ul><li>A Responses reasoning item with `encrypted_content` decodes into `Encrypted` and is replayed in the input.</li><li>`reasoning.summary` is `auto`.</li><li>An empty reasoning item is not kept, so no empty reasoning delta is streamed.</li></ul> These fail today. | Responses decodes and replays the item. If the live replay shows the item's `id` is needed too, `ReasoningItem` gains `ID`, also additive; this is measured first. | `internal/wire/responses`, `providers/openai` |

### Phase 3: catalog

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F4 | A Together listing with a `togetherai` metadata section ranks by profile, so a free model is not first. It fails today with `[a/free b/paid]`. | Decode the `togetherai` section. | `catalog/model_metadata.go` |
| F5 | A failed listing followed by `LookupMetadata` finds the healthy document. It fails today with `context canceled`. | A context error is never cached as a metadata failure. | `catalog/model_metadata.go` |
| F15 | A failed fetch that finishes after another fetch cached a document leaves that document cached. It fails today. | On failure, re-read the entry under the lock, and keep a newer document. | `catalog/model_metadata.go` |
| F21 | A server that answers one probe at a time keeps every installed model that did not fail. Today it gives `[c e]`. | Q3 (a): only a probe that failed, or answered wrongly, drops a model. A timeout keeps it. | `internal/transport/probe.go`, `providers/ollama` |
| F28 | 20 listings without `WithHTTPClient` leave the goroutine count flat. Today it grows to 60. | `transport.DefaultClient` returns one shared client, built once. | `internal/transport/transport.go` |
| F34 | Writing to Ollama's `Recommended` leaves `Usable` alone. It fails today. | Clone. | `catalog/discovery.go` |
| F35 | `List("Kilo", WithProfile(…))` works. It fails today. | Lowercase the id once, at the top of `List`. | `catalog/discovery.go` |
| F36 | `sortByRankDesc` keeps the listing order among equal ranks, `[c a b]`. It fails today. | `slices.SortStableFunc`. | `catalog/models_catalog.go` |
| F37 | The metadata fetch sends the `WithClientInfo` User-Agent. It fails today. | `cfg.setUserAgent`. `ValidateOllamaURL` keeps its signature and uses the shared transport client. A new `ValidateOllamaURLWith(ctx, url, opts...)` takes options: the MADR's amendment of 2026-10-03. | `catalog/model_metadata.go`, `catalog/discovery.go` |

### Phase 4: wizard

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F6 | <ul><li>An unreachable Ollama endpoint, on an empty stdin, returns an error.</li><li>A cancelled ctx returns `ctx.Err()`.</li></ul> Both fail today: the loop never ends. | <ul><li>The endpoint loop checks `ctx`.</li><li>`TextPrompter` remembers reaching EOF. The first read at EOF still answers the default, which scripts rely on; a read after that returns `io.EOF`.</li></ul> | `wizard/configure.go`, `wizard/text_prompter.go` |
| F16 | The drain test, on a `TextPrompter` over a pipe, passes under `-race`. Today it reports a data race. | A mutex around `TextPrompter`'s writer state. | `wizard/text_prompter.go` |
| F17 | Raw-mode `Secret`, tested through its byte-reading loop with the terminal faked: <ul><li>an arrow key adds nothing;</li><li>UTF-8 survives;</li><li>Backspace removes a rune;</li><li>`\r\n` leaves no `\n` for the next prompt.</li></ul> All fail today. | Decode UTF-8, swallow CSI and SS3 sequences, and drop a `\n` that follows `\r`. Print `\r\n` while raw. | `wizard/text_prompter.go` |
| F18 | A pasted `" sk-…"` is an API key. Today it is saved as an OAuth session. | `TrimSpace` the secret on both paths. | `wizard/auth.go`, `wizard/text_prompter.go` |
| F19 | <ul><li>A stored session is offered "keep" under every OAuth method.</li><li>The method menu defaults from `Existing.Kind`.</li><li>A Kilo session can be kept.</li><li>The organisation and the vendor path default from `Existing`.</li></ul> All fail today. | The keep check runs before the method menu. A Kilo session is validated by Kilo's own rule (access present, provider `kilo`). Menu defaults come from `Existing`. | `wizard/auth.go` |
| F20 | <ul><li>A login uses `Options.HTTPClient`.</li><li>A fresh session takes the provider's client.</li><li>`KiloProfile` gets the entered base URL.</li></ul> All fail today. | <ul><li>Flows get `o.HTTPClient`.</li><li>A login stores only a client the caller gave.</li><li>`kiloProfile` gets `WithBaseURL(res.BaseURL)` and `o.ProviderOptions`.</li></ul> | `wizard/auth.go`, `auth/oauth_loopback.go` |
| F49 | A `Prompter` returning -1 gives an error, not a panic. It fails today. | One bounds-checking helper for every `Select` result. | `wizard` |
| F50 | `Existing.Fallbacks` is preselected. It fails today. | Pass their indices. | `wizard/model_select.go` |
| F51 | Without a `TokenStore`, the vendor-CLI and Grok key-paste methods are offered. It fails today. | Filter on "needs a store". | `wizard/auth.go` |
| F25 (wizard half) | The vendor-CLI default path reads `HOME`/`USERPROFILE` through `Options.LookupEnv`. With no `LookupEnv`, it asks the caller to pass one. It fails today. | Q5 (a). The consequence goes in the README: a consumer that wants the default path passes `os.Getenv`. | `wizard/import.go` |

### Phase 5: core hygiene

| ID | Red test (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F13 | `WithHTTPClient(nil)` gives the default client and no panic. It fails today. | `ResolveOptions` keeps the default for a nil client. | `llmprovider/options.go` |
| F22 | Pointer and nil items are `ErrInvalidRequest` from `Check`. Today they pass. | `validate` accepts only the four value types. | `llmprovider/contract.go` |
| F23 | Q6 (a): the owner's live capture first, with an invalid Gemini key and an invalid Anthropic key, which cost nothing. Then tests on the captured bodies give `ErrAuthFailure`. The region and credit bodies cannot be produced on demand, so they wait for a capture and are recorded as not done. | Vendor rows in `classifyAPIError`, for the captured bodies only. | `llmprovider/api_error.go` |
| F29 | `WithMaxTokens(0)` and `WithMaxTokens(-1)` are `ErrInvalidRequest` from `New`. Today they pass. | Validated in `ResolveOptions` (R23). | `llmprovider/options.go` |
| F30 | A trailing-slash `WithBaseURL` requests `/models` and `/v1/messages` on every provider and listing. Today it fails on seven. | Trim it once, in `ResolveOptions`. | `llmprovider/settings.go` |
| F31 | `Retry-After: 99999999999` is a long positive wait on amd64. Run with `GOARCH=amd64`. Today it is negative. | Clamp before converting. | `internal/transport/transport.go` |
| F32 | A wrapped lister is still a `ModelLister`, and the wrapper reports `NativeStreaming: Unsupported`. Both fail today. | Two wrapper types, chosen by whether the inner provider lists models. | `llmprovider/retry.go` |
| F33 | `ClassifyHTTPError(p, nil)` returns `ErrProviderUnavailable`. Today it panics. | Guard nil. | `llmprovider/api_error.go` |
| F38 | A ChatGPT listing's 401 is an `*APIError` and refreshes once. It fails today. | `ClassifyHTTPError`, and the F2 helper. | `providers/openai/chatgpt_listing.go` |
| F40 | `ToolChoiceNone` sends no tools where `tool_choice` cannot be sent. It fails today. | Leave the tools out. | `providers/ollama`, `providers/kilo` |
| F41 | An unlisted qwen model with no metadata routes as the table does, `-max` and Go's to chat. It fails today. | Correct the rule and its comment. | `providers/opencode/route.go` |
| F42 | The four chat providers' tools bodies, through one helper, match their current goldens. A characterisation test, which passes before and after. | A `chatcompletions` tools helper that takes `ToolChoice`. The dead `Opts.Tool` and `ForceTool` go. | `internal/wire/chatcompletions`, four providers |
| F43 | `Capabilities()` tables. | Claude `ForcedToolChoice`, Grok `Reasoning` and Hugging Face `ForcedToolChoice` become `BestEffort`, with their degradations documented. | three providers |
| F46 | A test of `listenOpenAILoopback`'s fallback from 1455 to 1457. New; it passes on the unchanged code and fails when the fallback is planted out. | Delete `listenFirstAvailable` and `oauthFlowConfig.notify`. | `auth/oauth_loopback.go` |
| F52 | `openai.New` and `gemini.New` refuse an empty key. Today they accept it. | Refuse it, as the other providers do. | `providers/openai`, `providers/gemini` |

### Phase 6: tooling and docs

| ID | Check (fails first) | Fix | Files |
| :--- | :--- | :--- | :--- |
| F25 | `ambientcheck` flags a planted `http.DefaultClient`, `http.DefaultTransport`, `os.UserHomeDir`, `os.ExpandEnv` and `log.Printf`. It misses them today. Once phases 3 and 4 have removed the real uses, the tree passes. | Five more selectors. | `internal/ambientcheck` |
| F26 | A planted unchecked error in `catalog/config.go` is reported. It is not today. | Delete the copied fleet exclusion. `config.go` lints clean without it, as measured. | `.golangci.yml` |
| F27 | CI's Linux job runs `go test -race ./...`. | One step. | `.github/workflows/ci.yml` |
| F53 | — | The Makefile install hint pins `v2.13.1`, as CI does. | `Makefile` |
| F54 | `check_api.py` on a planted `llmprovider/x/` package removal reports nothing breaking. It does today. | Exempt the `package …/llmprovider/x/` line. | `scripts/check_api.py` |
| F55 | — | `permissions: contents: read`. | `.github/workflows/ci.yml` |
| F56 | `parity-check` resolves every backticked Go name in the equivalent column with `go doc`. A planted `RankModel` fails it. | Correct the seven rows to `catalog.Rank`. Add the resolution to the script. | `docs/guides/migrating-from-mcplib.md`, `scripts/check_parity_map.py` |
| F57 | — | <ul><li>The live command becomes `./llmprovider/...`;</li><li>add `LLMPROVIDER_LIVE_OPENAI_SIGNIN`;</li><li>correct the guide's per-provider variable claim.</li></ul> | `AGENTS.md`, `docs/guides/adding-a-provider.md` |
| F58 | — | <ul><li>README's Ranking bullet names Together and states the fill rule.</li><li>Together's package doc gives the live results.</li><li>The `kilo-auto/free` label drops "★ Recommended".</li></ul> | `README.md`, `providers/together`, `catalog` |
| F59 | The fake prompter refuses an unscripted `Input`, unless a test opts in to blank searches. Tests that leaned on the lenient fake are updated, and each update is listed. | Move README's search sentence out of the `Discover` bullet. | `README.md`, `wizard/fake_prompter_test.go` |

## Verification

**V1. Red and green.** Each finding in the Execution record shows its FAIL
line, then its PASS line.

**V2. Gate.** The repository gate is clean at the end of each phase, with
`api-check` additive only.

**V3. Live checks,** with the owner's keys. On 2026-10-03 all eight
provider keys are present: `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`,
`OPENAI_API_KEY`, `XAI_API_KEY`, `TOGETHER_API_KEY`, `KILO_API_KEY`,
`OPENCODE_API_KEY` and `HF_TOKEN` (checked for presence only).

* **Phase 1:** a `CommandToken` whose first run prints a bad Together key
  and whose second prints the real one. One 401, one rerun, and an answer.
* **Phase 2:**
  * Claude thinking with a tool round trip;
  * `Response.Model` reported live by Claude, Gemini, OpenAI and one
    gateway;
  * a ChatGPT-session reasoning replay, which needs the owner's session.
* **Phase 3:** a live Together listing ranked by metadata.
* **Phase 5:** the invalid-key captures for Gemini and Anthropic (Q6).

**V4. Docs.** Every doc changed in phase 6 passes markdownlint and the link
check.

## Rollout and Rollback

* **Rollout.**
  * Six phase commits on `main`; the owner commits and pushes.
  * The phases are additive under R48, so the release that carries them is
    `v1.1.0`, tagged when the owner asks.
  * `prepare-commit-msg` picks it up with its own dependency bump.
* **Rollback.**
  * Each phase reverts on its own, newest first.
  * Phase 2 and Phase 5 change wire bodies and the goldens with them. A
    revert takes both back.

## Execution record

*Approved 2026-10-03 by the owner: "plan is approved, proceed."*

### Phase 1: credentials (2026-10-03)

* **F1.** `TestOAuthSession_FailedSaveNeverReusesSpentRefresh`.
  * Red, on the unchanged code: `second Token = "", llmprovider: authentication failed: oauth: refresh failed: 400 Bad Request: {"error":"invalid_grant","error_description":"refresh_token_reused"}`, and `refresh tokens sent = [old-refresh old-refresh], want [old-refresh refresh-1]`.
  * Fix: `oauthSessionState.spent` carries the session's `spentRefresh` into the refresh; `loadRotated` never adopts a stored refresh token equal to it.
  * Green, and `go test -race ./llmprovider/auth` passes.
* **F12.** `TestFileTokenStore_StaleLockHasOneTaker` and `TestFileTokenStore_UnlockKeepsSuccessorsLock`.
  * A seam, `lockBeforeTakeover`, pauses a waiter between judging the lock stale and taking it.
  * Red, on `HEAD`'s lock logic with only the seam added: `the second waiter also holds the lock: two holders`; `the stalled holder's unlock removed its successor's lock: … no such file or directory`.
  * Fix (Q4 a):
    * the lock file holds an owner token (pid, `crypto/rand` text, time);
    * a stale lock is taken over through `removeLockIfHeld`, which moves the file to a unique name, removes it only if it still holds the token judged stale, and otherwise puts it back (a hard link, so it never overwrites a newer lock);
    * the heartbeat touches only a lock it still holds;
    * unlock uses `removeLockIfHeld` with its own token.
  * Green, and `-race` passes.
  * **Test correction.** The first version of the one-taker test set `staleAfter` (200 ms) below the waiter's wait (300 ms) with heartbeats off. The first holder's fresh lock then went legitimately stale during the wait, and the test failed on the fixed code. `staleAfter` is now 10 s. The red was re-run with this timing, on `HEAD`'s logic, and fails as above.
* **F14.** `TestCommandToken_WaiterOutlivesLeaderCancel` and `TestOAuthSession_WaiterOutlivesLeaderCancel`.
  * Red, on the unchanged code: `waiter Token = "", context canceled; want a token: only the leader was cancelled`, from both.
  * Fix: each package's `tokenFuture` gains `abandoned`, set when the run failed and the leader's own ctx had ended. A waiter whose ctx is live runs the fetch again itself. A command that times out on its own is still shared.
  * Green. `-race` on `./llmprovider` and `./llmprovider/auth` passes.
  * **Test hang.** The first OAuth test's handler blocked on `r.Context().Done()`, which never fired, so cleanup hung on `srv.Close()`. The stuck test process was stopped. The handler now also waits on a `release` channel that cleanup closes first. The red was re-shown on `HEAD` with that test: `oauth_session_leader_test.go:63: waiter Token = "", context canceled`.
* **F2.** A new `llmtest` check, `R16-reauth`.
  * The check's token source hands out a refused token until it is invalidated. Its tokens name their own header (`X-Llmtest-Credential`), so every provider sends them, Ollama included (R16).
  * Red, on the unchanged providers: Claude, Gemini (both variants), Hugging Face, Kilo, Ollama, OpenCode (`zen-chat`, `go-messages`) and Together each report `R16 (a refused token is renewed once; 0017-MADR D3): after a 401, 0 invalidation(s) and 1 request(s)`. Grok and OpenAI pass.
  * **Check design, before the fix:**
    * A first version used an untyped token, and Ollama sends no credential without a named header. Naming the header covers Ollama.
    * OpenAI's ChatGPT variant gets its mode from the session, so the check's own source would build an API-key provider. `Harness.NoReauth`, a new field and additive, records why the check does not apply there. `TestOpenAI_OAuth401RetriesOnceAfterRefresh` already covers that session's 401.
  * Fix: `wire.Reauth` (`llmprovider/internal/wire/reauth.go`). Every provider's `Generate` now calls `generateOnce` through it, and OpenAI's and Grok's own `invalidate` copies are gone. A Kilo device session's renewal answers `ErrAuthFailure` (0017-MADR D2), which is returned.
  * `TestReauth` covers the helper's six paths; the package is at 100.0%.
  * `llmtest`'s reference provider now sends its token and reruns once. A `noReauth` flaw shows the check catching it (`TestRun_NamesTheBrokenRule`), and `TestRun_NoReauthSkipsTheCheck` shows the skip.
  * Green: every provider package passes, G-wire included. `llmtest` passes under `-race`, at 95.3%. Together's red was re-shown on `HEAD`'s `together.go`.
* **F8.** `TestRedact_ProviderKeysAndTokenFields` (20 cases) and `TestRedact_KeepsDiagnostics`.
  * Red, on the unchanged patterns: all 20 cases fail. The unsigned-JWT sample first began with "token", which the existing `Token` pattern caught, so it was reworded to test the JWT pattern itself, and then failed too. `KeepsDiagnostics` passes before and after.
  * Fix:
    * `sk-` (covering `sk-ant-` and `sk-or-`), `xai-`, `tgp_` and `hf_`;
    * a JWT whose signature is empty;
    * every name ending in `token`, plus `cookie`, `code_verifier` and `device_code`;
    * escaped quotes, and `&` ends a value;
    * the bare names `code` and `key` from 8 characters only (`reKVLong`), so `error code: 1102` stays readable;
    * Kilo's `{url}:{secret}` (`reKiloToken`);
    * `Bearer` and `Basic` from 4 characters.
  * **Test catch.** The first Kilo pattern allowed `/` in the secret, and `TestRedact_KeepsDiagnostics` caught it redacting `http://localhost:11434/v1/…`. The secret's class no longer has `/`.
  * Green: `internal/redact` at 100.0%, and `go test ./...` passes.
* **F44.** `TestOAuthCallback_StrayRequestKeepsWaiting`, `TestOAuthCallback_RefusesOtherMethods` and `TestParseOAuthInput_PastedURLNeedsState`.
  * Red, on the unchanged code:
    * `GET http://127.0.0.1/callback ended the login …; a stray request must not`;
    * `DELETE = 200, want 405` (POST and PUT the same), and `a refused method ended the login with … code:"abc"`;
    * `a pasted URL without state = "attacker-code", <nil>`.
  * Fix (Q7 a):
    * a state mismatch gets 400 and the login keeps waiting;
    * any method but GET and OPTIONS gets 405, with `Allow: GET, OPTIONS`;
    * `parseOAuthInput` requires the matching state on a pasted URL, while a bare code still skips the check.
  * **Tests of the replaced behaviour.** This replaces 0008-MADR D5's state-mismatch branch.
    * `TestOAuthCallback_RejectsStateMismatch` pinned the old behaviour and is removed; `TestOAuthCallback_StrayRequestKeepsWaiting` replaces it.
    * `TestOAuthCallback_OnlyTheExactSuffix` keeps its purpose, that another suffix is a mismatch, and now asserts 400 with the login still waiting.
    * A first attempt to remove the old test did not land: the edit script skips a pair whose replacement is already in the file, and an empty replacement always is. The failing run showed it, and it was removed by hand.
  * Green: `go test -race ./llmprovider/auth` passes.
* **F45.** `TestOAuthSession_UnknownIssuerNeverRefreshesAtXAI` and `TestRevokeOAuthSession_OnlyItsOwnIssuer`.
  * Both use a transport that records each request and dials nothing.
  * Red, on the unchanged code. It is worse than the MADR said:
    * an OpenAI session from a staging issuer sent its refresh token to xAI three times (`sent [POST auth.x.ai/oauth2/token ×3]`), once per refresh attempt, and a custom-issuer Grok session the same;
    * revoking a Kilo session started xAI-style discovery (`GET /.well-known/openid-configuration`);
    * a staging OpenAI session's revocation went to discovery, not to `/oauth/revoke`.
  * Fix:
    * `refreshTokenURL` returns an error. It derives a URL only for OpenAI's and xAI's issuers, or for a session with no issuer whose provider is OpenAI or Grok; anything else is `ErrAuthFailure`, sign in again, before any request.
    * Revocation dispatches by provider: OpenAI's JSON request, to its own issuer when it has one; Grok's discovery; any other provider `errors.ErrUnsupported`, with nothing sent.
  * **Fixtures corrected.** `TestRevokeOAuthSession_GrokUsesDiscovery` and `TestRevokeOAuthSession_GrokWithoutEndpoint` built sessions with no provider, which logins never do (`oauth_loopback.go:670`). The first then failed. The second passed through the new refusal branch rather than the path it tests. Both now set `Provider: ProviderGrok`.
  * `TestRefreshTokenURL` follows the new signature, and its comment no longer claims any other issuer falls back to Grok.
  * Green: `auth` under `-race`, `wizard` and every provider pass.
* **F47.** `TestOpenAIDevice_RefusesControlCharactersInUserCode`.
  * Red, on the unchanged code: `LoginDeviceOAuth = oauth: device-code wait: context deadline exceeded, want an invalid user_code error`, and `the user_code with a terminal escape was shown to the user`.
  * The test bounds the login with a 3 s deadline. Its first version had none, and on the unchanged code the login showed the code and polled until `go test`'s timeout.
  * Fix: `plainUserCode` (ASCII letters, digits, dashes) is Grok's check moved into a helper. Grok and OpenAI both use it.
  * Green, and `auth` passes under `-race`.
* **F48.** `TestNew_SessionLogsThroughWithLogger` in `providers/openai` and `providers/grok`, and `TestOAuthSession_UseLogger`.
  * Red, on the unchanged providers: `the session does not log through the provider's WithLogger logger`, from both. Re-shown on `HEAD`'s `grok.go`.
  * Fix: `(*OAuthSession).UseLogger`, which is additive and fills only a nil `Logger`. OpenAI's and Grok's `New` call it through `shareLogger`, next to `shareHTTPClient`.
  * Green.

* **Gate.** The first run failed only on `make lint` and the pre-add check, for
  the same reason: `unparam` on `removeLockIfHeld`'s unused `bool` result
  (F12). It now returns only an error. The rerun passes all 17 checks, `make
  generate-check` passes, and the precheck reports `330 file(s) clean`.
* **Live (V3).** `TestScratch_CommandTokenRerunsAfterLive401`, on a scratch
  copy: a `CommandToken` whose first run prints a refused key and whose
  second prints `TOGETHER_API_KEY`, which is never printed. Together answered
  401, the command ran again, and the reply came back: `command runs: 2;
  answer: "ALPHA"; err: <nil>`, `PASS (1.14s)`.
* **Records.**
  * `docs/guides/adding-a-provider.md` step 7 and `docs/architecture.md`
    name `wire.Reauth` and the `llmtest` check.
  * [0008-MADR-repair-oauth-loopback-and-session-wiring.md](0008-MADR-repair-oauth-loopback-and-session-wiring.md)
    records that F44 replaces D5's state-mismatch branch.
* **Additive API (R48).** `(*auth.OAuthSession).UseLogger` and
  `llmtest.Harness.NoReauth`. `api-check` passes.
