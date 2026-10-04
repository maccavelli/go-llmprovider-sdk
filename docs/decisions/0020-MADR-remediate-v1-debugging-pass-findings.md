---
status: accepted
date: 2026-10-04
decision-makers: repository owner
consulted: 0015-MADR-canonical-sdk-api-and-module-layout.md (R7, R10, R23, R25, R27, R30–R31, R45, R48), 0016-MADR-provider-auth-and-support-baseline.md (D3, D4, A2), 0017-MADR-together-provider-and-auth-extensions.md (D1, D3)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Remediate the Defects Found by the v1 Debugging Pass

> **Accepted 2026-10-03.** The owner: "Follow recommendations and proceed with 0010". Each design question
> takes its recommended answer: Q1 (a), Q2 (a), Q3 (a), Q4 (a), Q5 (a),
> Q6 (a) and Q7 (a). The 0020 PLAN is written next; no fix starts
> before it is approved.

## Context and Problem Statement

`v1.0.0` is tagged, and every plan except 0002's later phases and 0010's
open phases is complete. On 2026-10-03 the owner asked for a debugging pass
over the whole module: "find bugs, find gaps. find missing wiring, and
incomplete features or functionality. report back with an madr for review."

The pass found 59 findings:

* 7 of high severity, among them one that can sign the user out (F1);
* 21 medium;
* 31 low, including documentation and tooling.

Some contradict decisions this repository already made. For example:

* 0016-MADR D4 promises a rotated token is never discarded. F1 discards the
  whole token family.
* 0017-MADR D3 promises a `CommandToken` reruns after a 401. F2 shows that
  only two of nine providers do so.
* 0017-MADR D1 says Together ranks by use case. F4 shows that it never
  does.

Others are gaps in wiring that the docs describe as working.

This record lists every finding with its evidence. It decides how to
remediate them, and asks the owner the seven design questions that the
fixes depend on.

### Method

* **Baseline**, on `9e8230a`:
  * `go test -race -count=1 ./...` passes;
  * `go vet ./...` passes for Linux, macOS and Windows (cgo off), with and
    without `-tags live_gateways`;
  * `GOOS=windows golangci-lint` reports 0 issues.
* **Seven read-only audits** ran in parallel, one per area:
  * the core contract;
  * `auth`;
  * `catalog`;
  * the four first-party providers, with the shared wire packages;
  * the five gateway providers, with the registry;
  * `wizard`;
  * internals, tooling and docs.
* **Reproduction.** Each audit copied the tracked files into the session's
  scratch space and proved its suspicions there, with `httptest` servers,
  scripted prompters or planted defects. Each then deleted its copy. The
  repository was not written to, and no live service was called.
* **Labels.** "Repro" below means a scratch test showed the failure, with
  its output quoted. "Read" means the finding comes from the code alone.
* **Checked again.** Before this record was written, the owner's agent
  re-read the code of F1, F2, F3, F4, F6, F8, F11 and F21 at `9e8230a`.
  The remaining findings rest on the audits' reproductions.
* **Line numbers** are at `9e8230a`.
* **Merged duplicates.** When several audits reported the same defect, it
  is listed once, with every source ID.

**Out of scope, already recorded:**

* the 0010 items:
  * D10, the Windows DACL;
  * D12, the reserved names;
  * the Windows lint and the Windows `dep-check`, which 0010-PLAN P6a
    already covers (T6 here);
* Together's `required` HTTP 500 on `gpt-oss-120b`, pinned by 0017;
* the `LookupMetadata` nil client, fixed in `9e8230a`.

### A. High: lost credentials, lost answers, features that never run

| ID | Where | Finding | Evidence |
|---|---|---|---|
| F1 | `llmprovider/auth/oauth_session.go:498-545` (`reloadOrRefresh`, `loadRotated`) | **A failed save re-sends the spent refresh token, and the issuer revokes the whole token family.** After a rotation whose save failed, the store still holds the old refresh token. On the next 401 or expiry, `loadRotated` takes it for a sibling's rotation, because it compares only against the in-memory token and never against `spentRefresh`, and refreshes with it. This contradicts 0016-MADR D4 and A2. | Repro: a `FileTokenStore` holds `old-refresh` and its first save fails. Then `Token`, `Invalidate`, `Token` sent `[old-refresh old-refresh]`. Against an issuer that rejects reuse: `refresh_token_reused`, `ErrAuthFailure`, family revoked. `TestOAuthSession_SaveFailureKeepsRotation` misses it because its store's `Load` returns nil. (auth A1) |
| F2 | `providers/{claude,gemini,together,huggingface,kilo,opencode,ollama}` | **Only `openai` and `grok` re-run an `InvalidatingSource` after a 401.** The other seven return the 401. Every doc promises the rerun: the `InvalidatingSource` doc (`command_token.go:27-29`), README, `architecture.md`, `adding-a-provider.md` step 7, and 0017-MADR D3. | Repro, for each of the seven: `requests=1 invalidations=0 … authentication failed`. The only callers of `invalidate(` are `grok.go:158` and `openai.go:172`. (core C1, prov P5, gateways G1) |
| F3 | `internal/wire/messages/messages.go:88-143`; `generatecontent.go:107-163` | **Claude's `stop_reason` and the Google route's `finishReason` are never read.** A text answer cut at `max_tokens` returns as a success with an empty `FinishReason`. A `tool_use` cut at `max_tokens` is returned as a valid call with truncated arguments. Chat Completions refuses that case with `ErrIncomplete` (0012-MADR §1.5). README says a cut answer "sets `Response.FinishReason`". | Repro: Claude `{"stop_reason":"max_tokens",…"partial ans"}` gave `err=<nil> FinishReason=""`. `GenerateToolCall` on a cut `tool_use` gave `err=<nil> call={…}`. OpenCode's messages route gave `output=[FunctionCall {"subject":"fix: tru"}]`. (prov P2, gateways G3) |
| F4 | `llmprovider/catalog/model_metadata.go:278-296` (`decodeModelMetadata`) | **Together's ranking never runs.** The decoder keeps only the `opencode`, `opencode-go` and `huggingface` sections, while `modelMetadataKey(ProviderTogether)` asks for `togetherai`. Together therefore always falls back to its curated order, and a free model is recommended first. This contradicts 0017-MADR D1 and `WithProfile`'s doc. | Repro: a document with `togetherai` decodes to `[huggingface]`. End to end, `Recommended = [a/free b/paid]`. Re-read at `9e8230a`. (catalog K1) |
| F5 | `catalog/discovery.go:74-75`; `model_metadata.go:240-243, 306-312` | **A failed listing poisons the metadata cache for a minute.** The background metadata fetch shares the listing's context. When the listing fails, `List`'s `defer cancel()` aborts the fetch, and `context canceled` is cached as a metadata failure for `modelMetadataRetryAfter`. For that minute: <ul><li>every Hugging Face, OpenCode and Together listing ranks without metadata;</li><li>`LookupMetadata`, which the OpenCode provider uses for routing and reasoning effort at generation time, returns an error.</li></ul> A caller cancelling its own context does the same. | Repro: a Together listing answering 401, with a healthy 300 ms metadata server, is followed by `LookupMetadata … context canceled; NPM found=false`. (catalog K2) |
| F6 | `wizard/configure.go:238-258`; `text_prompter.go:216-251` | **The wizard loops forever on an unreachable Ollama endpoint when stdin is at EOF, and ignores `ctx`.** `Confirm` and `Input` swallow `io.EOF` and return their defaults, so "Try a different endpoint?" is answered yes for ever. | Repro: an empty stdin and an endpoint answering 404 gave 15,172 "cannot reach" warnings in the 500 ms after `ctx` was cancelled. Re-read at `9e8230a`. (wizard W1) |
| F7 | `internal/wire/messages/messages.go:52-116`; `llmprovider/item.go:51-53` | **Claude extended thinking cannot run a tool loop.** The thinking block's `signature` and any `redacted_thinking` block are dropped when a response is decoded, and `ReasoningItem` is never sent back. So the next turn's assistant `tool_use` message lacks the thinking block that Anthropic requires when thinking and tools are combined. Gemini has the equivalent (`FunctionCallItem.Signature`); Claude has nothing. | Repro: `signature:"SIG123"` was lost, and the replayed body carries no thinking block while `thinking` is enabled. **Not run live:** the 400 is Anthropic's documented rule, not a measured one. No live test sends Reasoning with tools. (prov P1) |

### B. Medium

| ID | Where | Finding | Evidence |
|---|---|---|---|
| F8 | `internal/redact/redact.go:31-50` | **Redaction misses this module's own key formats and every `*_token` field.** Nothing covers `sk-`, `sk-proj-`, `sk-ant-`, `xai-`, `tgp_` or `hf_`. `\btoken\b` cannot match `refresh_token` or `access_token`, because `_` is a word character. Also missed: form bodies (`code=`, `code_verifier=`), `Cookie:`, Kilo's `{url}:{secret}`, short bearers. `redact.String` is the only redaction applied to provider error bodies and OAuth token-endpoint errors. | Repro: each of those fake values came back unmasked. Re-read at `9e8230a`. (tooling T1) |
| F9 | `chatcompletions.go:195-199, 234-235`; `messages.go:100-105, 138-139`; `generatecontent.go:124-129`; `interactions.go:193-194`; `responses.go:67-68` | **Decode and empty-answer errors carry no kind, so `WithRetry` resends a request the service already answered and billed.** They break R25 and R27, and on OpenCode's messages route they name "claude" whatever the model. A valid reply over the 1 MiB `LimitReader` becomes an unkinded EOF error that is retried three times. | Repro: `{"choices":[]}` through `WithRetry(together)` sent 3 requests, `matches any kind = false`. The oversized reply gave 3 requests, then `unexpected EOF`. (core C3, prov P7, gateways G4) |
| F10 | `internal/wire/responses/responses.go:33-37` | **An empty `Role` is sent as `"role":""` on the Responses wire.** That covers OpenAI, the ChatGPT session, Grok and OpenCode's responses route. The contract says "An empty Role is the user's", and the other encoders map it to `user`. | Repro: `input=[{"content":"hi","role":""}]`. (core C5, prov P4, gateways G2) |
| F11 | every wire `Decode` | **`Response.Model` is never set,** and `FinishReason` is set only by Chat Completions. Each service reports both. R7 and `adding-a-provider.md:175` require them, and for `kilo-auto/*` the model is the only way to learn which model answered. | Repro: a reply carrying `"model":"served-model"` gave `Model=""` on every route. `llmtest` checks neither field. (prov P3, gateways G3) |
| F12 | `auth/tokenstore_file.go:217-223, 282-286` | **The cross-process refresh lock (0016 A2) can be held twice.** Taking over a stale lock is a check followed by a remove, so two waiters can both take it. Releasing the lock removes whatever lock file is present, possibly a successor's, and the heartbeat touches it too. | Repro: 8 goroutines against one stale lock gave 2 simultaneous holders in 1 run of 300. A stalled holder's release removed the new holder's file. (auth A2, A3) |
| F13 | `llmprovider/options.go:36-40`; `settings.go:213-215` | **`WithHTTPClient(nil)` is accepted by `New`, then panics in `Generate`.** `Settings.HTTPClient` says nil means the provider's default. `9e8230a` fixed the same defect in `catalog` only. | Repro: `New err=<nil>`, then `PANIC … nil pointer dereference`. (core C4) |
| F14 | `command_token.go:82-108`; `auth/oauth_session.go:110-162` | **A shared token fetch hands the first caller's cancellation to every waiter.** `CommandToken` and `OAuthSession` both run their single-flight fetch under the first caller's context, so a waiter whose own context is live gets `context.Canceled`. | Repro: both types gave `waiter (never cancelled) err = context canceled`. (core C2, auth A6) |
| F15 | `catalog/model_metadata.go:228-245` | **A slow failed metadata fetch overwrites a fresh document that another caller cached meanwhile.** The entry is copied before the fetch and written back on failure. | Repro: `load3 (after a good fetch was cached) err=… context canceled`. (catalog K3) |
| F16 | `wizard/auth.go:367-381`; `text_prompter.go:62-78` | **Data race in `TextPrompter`.** When the browser login wins, `pastePrompt.drain` calls `Notify` while the paste goroutine is still inside `Input`. The existing drain test uses a fake prompter, so it never runs `TextPrompter`. | Repro: `WARNING: DATA RACE`, `flushErr` against `printf`. (wizard W2) |
| F17 | `wizard/text_prompter.go:330-345` | **Raw-mode `Secret` corrupts what it reads.** <ul><li>It keeps escape sequences after ESC: an arrow key adds `[D`.</li><li>It turns non-ASCII into Latin-1 mojibake.</li><li>It ends on `\r` and leaves the `\n` behind, so the next prompt silently takes its default.</li></ul> | Repro through a pty: `CAPTURED="abcdefgh[Dij"`. A paste ending in CRLF made the next `Select` return its default. (wizard W3, W4) |
| F18 | `wizard/auth.go:272` | **A pasted OpenAI API key with a leading space is saved as a ChatGPT OAuth session.** The TTY path does not trim, and `HasPrefix(value, "sk-")` then fails. | Repro: `kind="oauth" apiKeySet=false saves=1`. (wizard W5) |
| F19 | `wizard/auth.go:85-258` | **"Keep the existing session" (0016-MADR A10) is offered only under browser sign-in.** <ul><li>Device-code logins (OpenAI, Grok, Kilo) and pasted tokens always log in again.</li><li>A Kilo session could never be kept anyway: `ValidateOAuthSession` rejects it because it has no refresh token.</li><li>The method menu ignores `Existing.Kind`.</li><li>`Existing.Organization` and `Existing.VendorAuthPath` are not used as defaults.</li></ul> | Repro: OpenAI device with a stored session gave `logins=1 confirms=[]`. Kilo with `org-2` gave `org=""`. (wizard W6) |
| F20 | `wizard/auth.go:110-186`; `auth/oauth_loopback.go:202-205, 673` | **The caller's HTTP client never reaches the sign-in flows or the sessions they create** (0016-MADR D8). <ul><li>The wizard leaves `OAuthFlowOptions.HTTPClient` unset.</li><li>A login stores its defaulted client on the session, so a provider's `UseHTTPClient` later does nothing.</li><li>`KiloProfile` does not get the base URL the user entered.</li></ul> | Repro: `loginHTTPClient=<nil>`; `session client is provider's: false`. (wizard W7, auth A5) |
| F21 | `providers/ollama/ollama.go:237-252`; `internal/transport/probe.go:31-40` | **Ollama's listing hides installed models that work.** Every probe starts at once with a 5 s timeout. Once any model answers, only the models that answered are listed, so a server that loads models one at a time loses the others. | Repro, against a server that answers one request at a time: `installed=[a b c d e] listed=[c:latest e:latest]`. That real Ollama queues cold loads past 5 s is not measured. (gateways G5) |
| F22 | `llmprovider/contract.go:214-218`; `item.go:14-16` | **Pointer and nil `Item`s pass validation and are then dropped.** The marker methods have value receivers, so `*MessageItem` is an `Item`. `validate` checks only the value types, and every encoder's type switch drops anything else silently. | Repro: `Input{&MessageItem{…}, nil}` passes `Check`, and the wire carries `"messages":null`. (core C6) |
| F23 | `llmprovider/api_error.go:228-260` | **Common vendor errors are classified by HTTP status alone.** Gemini's 400 `API_KEY_INVALID` and 400 `FAILED_PRECONDITION` (region), and Anthropic's 400 "credit balance is too low", come out as terminal `ErrInvalidRequest`. They should be `ErrAuthFailure`, `ErrNotPermitted` and `ErrQuotaExhausted`. | Repro through `ClassifyHTTPError` on the vendors' documented body shapes. **No live capture.** (prov P6) |
| F24 | `providers/openai/openai.go:197, 226-228`; `responses.go:29-133` | **OpenAI reasoning is wired halfway.** <ul><li>`reasoning.encrypted_content` is requested, then dropped when decoded, and never replayed.</li><li>No summary is requested, so every reasoning item decodes empty.</li><li>The empty items are kept, and `Stream` emits empty reasoning deltas.</li></ul> | Repro: `ReasoningItem{Text:""}`, and nothing is replayed. (prov P8) |
| F25 | `internal/ambientcheck/ambientcheck_test.go:18-23, 94-107` | **The R30–R31 check cannot see the defects it exists to catch.** It does not detect `http.DefaultClient`, `http.DefaultTransport`, `os.UserHomeDir`, `os.ExpandEnv` or the `log` package's global logger. Two real cases pass it: <ul><li>`catalog.ValidateOllamaURL` uses `http.DefaultClient` (`discovery.go:415`), against `transport.go:25`'s rule;</li><li>`wizard/import.go:33` reads the home directory directly, while 0015's "no ambient state" amendment says `wizard` reads the environment only through `Options.LookupEnv`.</li></ul> | Repro: a planted file using all of these was flagged only for its control `os.Getenv`. (tooling T2) |
| F26 | `.golangci.yml:91-105` | **A fleet-wide exclusion copied from other repositories turns off 12 linters for any file named `config.go`, `client.go`, `login.go` and others.** Here that is `catalog/config.go` today, and any future `client.go` or `login.go`. | Repro: the same unchecked `Body.Close` was reported in `discovery.go` and not in `config.go`. Without the rule, `config.go` lints clean today. (tooling T3) |
| F27 | `.github/workflows/ci.yml:23`; `Makefile:16-17` | **Nothing runs `-race`.** R44, `llmtest`'s concurrency check and `adding-a-provider.md:289` all depend on it. `go test -race ./...` passes today, so this is unenforced rather than broken. | Read: no `-race` in the Makefile, the scripts or CI. (tooling T5) |
| F28 | `llmprovider/settings.go:167`; `internal/transport/transport.go:30-41` | **Every `List` call, and every metadata lookup, without `WithHTTPClient` builds a new `http.Transport`.** Connections are never reused, and idle ones linger for 90 s. | Repro: 20 listings gave 60 goroutines with the default client, 3 with one shared client. (catalog K6) |

### C. Low: correctness, hygiene, documentation and tooling

| ID | Where | Finding | Source |
|---|---|---|---|
| F29 | `options.go:42-47` | `WithMaxTokens(0)` and negative values go on the wire, against R23. Repro: `max_tokens:-1`. | core C7, prov P10 |
| F30 | `catalog/discovery.go:193, 256, 365, 431`; `openai.go:260`, `claude.go:136`, `gemini.go:142`, `grok.go:236` | A base URL with a trailing slash requests `//models`, `/v1//messages`. Some paths trim it and others do not. Repro. | catalog K7, prov P9 |
| F31 | `internal/transport/transport.go:121-128` | An unbounded `Retry-After` overflows the float-to-int conversion. On amd64 it becomes negative, and `WithRetry` ignores it. Repro under `GOARCH=amd64`. | core C8 |
| F32 | `llmprovider/retry.go:44-55` | `WithRetry` hides `ModelLister`, so the wizard loses listing. It also passes on `NativeStreaming` without having a `Stream` method. Repro (lister). | core C9, C10 |
| F33 | `api_error.go:180-181` | The exported `ClassifyHTTPError(provider, nil)` panics. Repro. | core C11 |
| F34 | `catalog/discovery.go:352-356` | Ollama's `Recommended` and `Usable` share one underlying array. Repro. | catalog K4 |
| F35 | `catalog/discovery.go:66-69` | `List("Kilo")` accepts the mixed case but refuses `WithProfile` for it. Repro. | catalog K5 |
| F36 | `catalog/models_catalog.go:431-440` | `sortByRankDesc` is not stable, though its comment says it is. Repro: `[c b a]` where `[c a b]` was expected. | catalog K8 |
| F37 | `catalog/model_metadata.go:264`; `discovery.go:413` | The metadata fetch and `ValidateOllamaURL` ignore `WithClientInfo` (0012-MADR §1.4). Repro. | catalog K9 |
| F38 | `providers/openai/chatgpt_listing.go:87-89` | A ChatGPT listing that is not 200 returns an unclassified error, and a 401 refreshes nothing. Repro. | prov P11 |
| F39 | `internal/wire/responses/responses.go:181-201` | A Responses stream `{"type":"error"}` event becomes a retryable "stream ended" error. Repro. Documented for OpenAI; not seen from the ChatGPT backend. | prov P12 |
| F40 | `ollama.go:187-196`; `kilo.go:281-288` | `ToolChoiceNone` still sends the tools when `tool_choice` cannot be sent; leaving the tools out would honour it. Repro. | gateways G6 |
| F41 | `providers/opencode/route.go:206-217` | The route heuristic's comment and its qwen rule contradict the table: unlisted qwen `-max` and Go models go to messages. Read. | gateways G7 |
| F42 | `internal/wire/chatcompletions/chatcompletions.go` | `Opts.Tool` and `Opts.ForceTool` are used only by tests. Four providers build `tools` and `tool_choice` with near-identical code. Read. | gateways G9 |
| F43 | `claude.go:232-233`; `grok`; `huggingface.go:105-106` | Capabilities claim more than is delivered (R10): <ul><li>Claude: `ForcedToolChoice: Supported`, but `auto` is sent when Reasoning is on;</li><li>Grok: `Reasoning: Supported`, but it is dropped outside its model menu;</li><li>Hugging Face: `ForcedToolChoice: Supported`, with only a named tool measured.</li></ul> Repro (Grok). | prov notes, gateways G10 |
| F44 | `auth/oauth_loopback.go:485-535, 588` | Any stray request to the callback path, from any method, aborts a browser login. A pasted callback URL with no `state` skips the state check. Repro. | auth A7, A8 |
| F45 | `auth/oauth_session.go:548-557`; `oauth_revoke.go:51-55` | A session with an empty `TokenURL` and a non-default issuer sends its refresh token to `auth.x.ai`. Revocation branches the same way. Logins always set `TokenURL`, so only hand-built or legacy sessions are exposed. Repro. | auth A9 |
| F46 | `auth/oauth_loopback.go:59, 221, 384-394` | Dead code: `listenFirstAvailable` and `oauthFlowConfig.notify`. The real 1455→1457 fallback (`listenOpenAILoopback`) has no test. Read. | auth A10 |
| F47 | `auth/oauth_device.go:200-202` | OpenAI's device `user_code` is not checked for control characters before printing; Grok's and Kilo's are. Read. | auth A11 |
| F48 | `auth/tokenstore.go:33-36`; `oauth_session.go:203-209` | A failed rotation save is logged only to `OAuthSession.Logger`, which nothing sets; 0016-MADR D4 says the `WithLogger` logger. Read. | auth A4 |
| F49 | `wizard/configure.go:145`; `auth.go:89, 210`; `model_select.go:57-91` | A consumer `Prompter` that returns an out-of-range index panics the wizard. Repro. | wizard W8 |
| F50 | `wizard/model_select.go:144, 157` | `Existing.Fallbacks` is never preselected. Repro. | wizard W9 |
| F51 | `wizard/auth.go:49-60` | Without a `TokenStore`, the vendor-CLI read-through and Grok's key paste are hidden too, though neither needs a store. Read. | wizard W10 |
| F52 | `openai.New`, `gemini.New` | An empty API key is accepted and sent; Claude, Together, Grok, Hugging Face and OpenCode refuse it. Read. | core, prov notes |
| F53 | `Makefile:34`; `ci.yml:30`; `go-precheck.sh:107` | golangci-lint version drift: CI pins `v2.13.1`, the Makefile hint says `@latest`, and the local binary is 2.14.0. Read. | tooling T7 |
| F54 | `scripts/check_api.py:31, 66-67` | The `llmprovider/x/` exemption misses apidiff's "package … removed" line. The path is dead until `x/` exists. Read; the core check was reproduced sound. | tooling T8 |
| F55 | `.github/workflows/ci.yml` | No `permissions:` block, so the job gets the default `GITHUB_TOKEN` scope. Read. | tooling T10 |
| F56 | `docs/guides/migrating-from-mcplib.md:314-320` | Seven rows name `RankModel(…)`, which does not exist (it is `catalog.Rank`), and one row's backticks are broken. `parity-check` passes because it checks only that the cell is not empty. Repro (`go doc`). | tooling T4 |
| F57 | `AGENTS.md:146-155`; `docs/guides/adding-a-provider.md:317-318` | The documented live-test command skips the live tests in `auth` and `catalog`, and the variable list omits `LLMPROVIDER_LIVE_OPENAI_SIGNIN`. The guide claims a `LLMPROVIDER_LIVE_<ID>` variable per provider; only Together has one. Read. | tooling T9 |
| F58 | `README.md:108-119`; `providers/together/together.go:9-11`; `catalog/models_catalog.go:713` | README's Ranking bullet: <ul><li>leaves out Together;</li><li>says "never" where `rankRecommended`'s fill step can add such models.</li></ul> Elsewhere: <ul><li>Together's package doc says its live tests "have not run yet";</li><li>the `kilo-auto/free` label says "★ Recommended", though the utility profile never recommends it.</li></ul> Read. | catalog K10, gateways G8 |
| F59 | `README.md`, "The configuration wizard"; `wizard/fake_prompter_test.go:88-98` | README files the search line under `Options.Discover`, but the search runs without it. The fake prompter accepts an unscripted `Input`, so extra or missing prompts go unnoticed in tests. Read. | wizard W11, W12 |

**Checked and found sound.** These came out clean:

* **Dependencies, registry, links:**
  * the registry: all ten providers, descriptors, credential refusals;
  * `dep-check`, `coverage-check` (every package measured), the G-wire goldens, `api-check`'s core logic;
  * the record index and its statuses, and every relative link and anchor.
* **Transport and wire:**
  * SSE parsing: CRLF, comments, multi-line `data:`, `[DONE]`;
  * body closing and context propagation everywhere;
  * Retry-After parsing apart from F31;
  * header sets per provider;
  * Gemini's thought-signature replay;
  * pagination for Gemini and Claude.
* **Auth:**
  * PKCE, state, nonce and `id_token` validation;
  * `FileTokenStore.Save`'s durability;
  * the redaction methods on `OAuthSession` and `Result`;
  * `DeviceLogin` synchronisation;
  * the read-through of `VendorCLISession`;
  * `CommandToken`'s output cap and the absence of a shell.
* **Catalog:** search escaping and determinism, and ranking ties, fill and price guards.

## Decision Drivers

* **v1 promises.** A behaviour the docs or an accepted record promise is a
  defect when it is missing, not a feature request: F1, F2, F3, F4, F11 and
  F19.
* **Credential safety first.** Signing a user out (F1), a double refresh
  (F12) and secrets in errors (F8) outrank everything else.
* **No re-billing.** An answer the service has given must not be bought
  again (F9).
* **API compatibility (R48).** Fixes stay inside v1: additive exported API
  only, checked by `api-check`. A wire change follows R45, with goldens
  updated in the same commit.
* **Each fix proven.** Every fix is shown by a test that fails first on a
  scratch copy, as this repository requires.
* **One place per concern.** F2, F9 and F42 come from providers each
  carrying their own copy of shared logic. The fix should leave one copy.

## Considered Options

* Remediate in a 0020 PLAN, ordered by severity, after the owner answers
  seven design questions
* Fold each finding into the record that owns its area (0015, 0016, 0017,
  0012)
* Fix only the high findings now; record the rest as accepted
* Defer everything to a v1.1 planning cycle

## Decision Outcome

Chosen option: "Remediate in a 0020 PLAN, ordered by severity, after the
owner answers seven design questions". One record keeps the 59 findings,
their evidence and their fixes together. The findings cross every area,
which would scatter them across five records. Seven fixes depend on
choices that only the owner can make. The rest have one obvious fix, which
the PLAN carries.

### 1. Phases for the PLAN, once this record is accepted

1. **Credentials:** F1, F12, F14, F2, then F8, F44, F45, F47 and F48. F2
   becomes one shared helper used by every provider, with Kilo's device
   session excepted under 0017-MADR D2: a Kilo session cannot refresh, so
   its 401 stays `ErrAuthFailure`.
2. **Answers:** F3, F9, F10, F11 and F39, then F7 and F24 per Q1. F11 adds
   `llmtest` checks for `Model` and `FinishReason`.
3. **Catalog:** F4, F5, F15 and F21 (per Q3), then F28 and F34–F37.
4. **Wizard:** F6, F16–F20, F49–F51.
5. **Core hygiene:** F13, F22, F29–F33, F38, F40–F43, F46, F52, and F23
   per Q6.
6. **Tooling and documentation:** F25 (per Q5), F26, F27, F53–F59.

### 2. Design questions

**Q1. Replaying reasoning (F7, F24).**

* (a) **Recommended.** Add opaque replay fields to `ReasoningItem`:
  `Signature` for Claude's thinking signature, and `Encrypted` for
  Claude's redacted thinking and OpenAI's `encrypted_content`.
  * Messages and Responses decode them and replay them in place.
  * Additive under R48.
  * A live Claude thinking-plus-tools round trip confirms it.
* (b) Refuse Reasoning with tools on Claude (`ErrUnsupported`), declare it
  `BestEffort`, and stop requesting `encrypted_content`.
* (c) Defer.

**Q2. The kind of a decode or empty-answer error (F9).**

* (a) **Recommended.** `ErrIncomplete`, which is terminal and so never
  retried.
  * The 1 MiB reply cap is raised to the stream limit (16 MiB).
  * Above the cap, a reply is `ErrIncomplete` too.
  * `WithRetry` retries a kindless error only when it comes from the
    transport (`*url.Error`).
* (b) `ErrProviderUnavailable`, retried, as today: it re-bills.

**Q3. Ollama probes (F21).**

* (a) **Recommended.** Drop only a model whose probe failed or answered
  wrongly. A model whose probe timed out stays listed.
* (b) Probe one model at a time.
* (c) A longer timeout.

**Q4. The refresh lock (F12).**

* (a) **Recommended.** Write an owner token (pid and random bytes) into the
  lock file.
  * A stale lock is taken over by renaming it to a unique name; only the
    process whose rename succeeds proceeds.
  * Unlock and the heartbeat check the token before acting.
  * Standard library on every OS.
* (b) OS locks: `flock`, and `LockFileEx` through the `mkwinsyscall`
  bindings 0010 introduces.

**Q5. The home directory and ambient state (F25).**

* (a) **Recommended.** The wizard gets `HOME`/`USERPROFILE` through
  `Options.LookupEnv`.
  * `ValidateOllamaURL` takes options, so it can use the transport client
    and `WithClientInfo`; this also fixes F37's second half.
  * `ambientcheck` learns the five missing selectors.
* (b) Amend 0015's "no ambient state" amendment to allow
  `os.UserHomeDir`, and extend `ambientcheck` for the rest.

**Q6. Vendor error rows (F23).**

* (a) **Recommended.** Capture the real bodies live first, then add the
  rows. A request with an invalid key costs nothing, and the owner's keys
  are available.
* (b) Add the rows now from the vendors' documented shapes.

**Q7. Stray callback requests (F44).**

* (a) **Recommended.** A request whose state does not match gets 400, and
  the listener keeps waiting. Only GET is accepted, and OPTIONS for Grok's
  preflight. A pasted URL must carry a matching state; a bare code still
  skips the check.
* (b) Add only the method check.

### 3. Routed elsewhere

* T6 (Windows lint, Windows `dep-check`) is 0010-PLAN P6a, proposed.

### Consequences

* Good:
  * The documented behaviour becomes true again: 401 reruns, Together
    ranking, cut-answer reporting, keeping an existing session.
  * Signing a user out after a failed save stops.
* Good, because the shared helpers (401 rerun, tool encoding,
  stop-reason mapping) remove whole classes of drift between providers.
* Good:
  * `llmtest` gains checks for 401 rerun, empty role, pointer items,
    `Model` and `FinishReason`.
  * A third-party provider is then held to the same rules.
* Neutral, because F10's wire change updates goldens under R45, and F3 and
  F11 change what callers receive. Both bring the code in line with what
  the docs already say.
* Bad, because this is a large PLAN: 59 findings in six phases.
* Bad, because three items need the owner live: Claude thinking with tools
  (Q1), the vendor error bodies (Q6), and the Windows lock behaviour under
  0010.

### Confirmation

* **Failing first.** Each fixed finding has a test that fails first on a
  scratch copy and passes after the fix. The PLAN records both outputs. For
  the medium and high findings, the scratch reproduction above becomes
  that test.
* **The gate is clean:**
  * `make pre-add-check`, `make lint`, `parity-check`, `dep-check`,
    `coverage-check`, `api-check` (additive changes only), markdownlint and
    links;
  * `go test -race ./...`, which F27 then makes part of CI.
* **Live checks, with the owner's keys:**
  * a Claude thinking-and-tools round trip (Q1);
  * a `CommandToken` 401 rerun against one gateway (F2);
  * a Together listing ranked by metadata (F4);
  * the vendor error captures (Q6).

## Pros and Cons of the Options

### Remediate in a 0020 PLAN, ordered by severity, after seven design questions

* Good, because the evidence, the decisions and the fixes stay in one
  place, as 0013 did for the previous pass.
* Good, because the high findings land first, and each phase is releasable
  on its own.
* Bad, because one PLAN touches every package. The phases keep each commit
  to one area.

### Fold each finding into the record that owns its area

* Good, because each fix sits beside the decision it repairs.
* Bad, because the evidence is split across five records, and shared fixes
  (F2's helper, F9's error kind) would be decided five times.

### Fix only the high findings now; record the rest as accepted

* Good, because it is faster.
* Bad, because several medium findings break documented behaviour or
  safety: F8 redaction, F12 locking, F13 a panic, F19 keep-existing. They
  are not acceptable as recorded debt.

### Defer everything to a v1.1 planning cycle

* Bad, because F1 can sign users out and F2 breaks a decided feature.
  Leaving them in a tagged v1 contradicts the records.

## More Information

### Relationship to other records

* [0013-MADR-remediate-debugging-pass-findings.md](0013-MADR-remediate-debugging-pass-findings.md)
  is the previous debugging pass, and this record follows its shape.
* [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
  D4, A2 and A10 are repaired by F1, F12 and F19.
* [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md)
  D1 and D3 are repaired by F4 and F2.
* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md):
  * R7, R10, R23, R25 and R27 are restored by F11, F43, F29 and F9;
  * F25 is its "no ambient state" amendment.
* [0012-MADR-conform-providers-to-reference-clients.md](0012-MADR-conform-providers-to-reference-clients.md)
  §1.4 and §1.5 are extended by F37 and F3.
* [0010-MADR-windows-stdio-oauth-tokenstore-ci.md](0010-MADR-windows-stdio-oauth-tokenstore-ci.md)
  keeps T6, D10 and D12. Q4 (b) would reuse its Windows bindings.

### Evidence

* **Where the scratch tests ran.** The seven audits worked in
  `scratchpad/audit/<area>/` and deleted it afterwards, so their quoted
  output is the evidence. The PLAN re-creates each reproduction as its
  failing-first test.
* **No live calls were made in this pass.** F7, F21 and F23 rest on
  documented vendor behaviour, and say so.

## Amendment 2026-10-03: Q5 (a) without breaking `ValidateOllamaURL`

Found while writing
[0020-PLAN-remediate-v1-debugging-pass-findings.md](0020-PLAN-remediate-v1-debugging-pass-findings.md).

* **The conflict.** Q5 (a) says "`ValidateOllamaURL` takes options".
  * Adding a parameter, even a variadic one, to an exported function is an
    incompatible change under R48.
  * `api-check` reports exactly that:
    `ValidateOllamaURL: changed from func(context.Context, string) error to
    func(context.Context, string, ...int) error` (planted, in the debugging
    pass).
* **As implemented:**
  * `ValidateOllamaURL` keeps its signature, and uses the shared transport
    client instead of `http.DefaultClient`.
  * A new `ValidateOllamaURLWith(ctx, url, opts...)` takes options: the
    client and `WithClientInfo`. The wizard calls it.
  * The change is additive, so Q5's intent holds within R48.
* Nothing else in Q5 changes.

## Amendment 2026-10-03: F7 measured live, and two live checks open

Found running 0020-PLAN's phase 2 live checks.

* **F7's premise does not hold on `claude-haiku-4-5`.**
  * The record said that replaying a thinking-and-tools turn without the
    thinking block's signature is refused with HTTP 400. It said this from
    Anthropic's documentation, unmeasured.
  * Live, `HEAD`'s Messages wire, which drops the signature, completed the
    round trip; the second turn answered.
  * The fix still stands. It keeps the signed thinking and sends it back
    ahead of the tool call, as Anthropic documents for thinking with tools,
    and the live round trip passes with it (`1 signed reasoning item`).
  * But F7 is not a tool loop that cannot run. Its severity was overstated:
    it is a lost reasoning continuity, not a failure.
* **Open live checks:**
  * **OpenAI's `Model`.** The OpenAI API key answered `429
    credit_balance_exhausted`, so it could not be checked. It waits for the
    owner to add credits.
  * **The ChatGPT reasoning replay (F24).** It needs the owner's ChatGPT
    session. The replay sends `encrypted_content` without the item's `id`,
    which is believed to be Codex's practice with `store: false` but is
    unconfirmed, and the live check will settle it.

## Amendment 2026-10-03: F50 shows the preselection in `TextPrompter`

Found running 0020-PLAN's phase 4.

* **Found.** F50's fix preselects the saved fallbacks. `TextPrompter`,
  given a preselection, said "blank for none" but kept it on a blank line,
  marked no row, and had no input for none.
* **Decision, the owner's:** with a preselection, `TextPrompter.MultiSelect`
  marks those rows `(selected)`, says `blank keeps …; 0 for none`, and
  takes `0` as none. Without one it is unchanged.
* **Consequence.** A change to what `TextPrompter` prints and accepts, and
  only when given a preselection. The wizard passes one only from
  `Existing.Fallbacks`. The `Prompter` interface does not change.

## Amendment 2026-10-03: facts corrected by phase 5

Found running 0020-PLAN's phase 5. No decision changes.

* **F52.** OpenCode does not refuse an empty key. Kilo and the OpenCode
  gateways send their anonymous key for one, by design. Only OpenAI and
  Gemini accepted an empty key and sent it.
* **F23.** The live capture, 2026-10-03:
  * Gemini answers an invalid key with HTTP 400 `INVALID_ARGUMENT`, the
    key failure only in `details[].reason: "API_KEY_INVALID"`. It is now
    `ErrAuthFailure`.
  * Anthropic answers an invalid key with HTTP 401, already
    `ErrAuthFailure`.
  * The region and credit bodies the finding names were not captured, and
    stay open.
* **F30.** A trailing slash doubled the path on five providers (Gemini,
  OpenAI, Claude, Grok, Ollama), eleven requests in all, not seven.

## Amendment 2026-10-04: the ChatGPT reasoning replay, measured

Found by 0020-PLAN's live check of 2026-10-04 ("the ChatGPT reasoning replay
(F24)"). It settles the open question of the amendment "F7 measured live,
and two live checks open". No decision changes.

* **The replay needs no `id`.**
  * With `store: false`, a reasoning item replayed with its
    `encrypted_content` alone is accepted. The follow-up answered from it.
  * Tampered content is refused with `invalid_encrypted_content`, so the
    backend reads the item.
  * `ReasoningItem` gains no `ID` field.
* **A forced tool call returns no reasoning item** from the ChatGPT backend,
  on every listed model. A text turn does, with summary text and
  `encrypted_content`. That is the service's behaviour, not a defect here.
