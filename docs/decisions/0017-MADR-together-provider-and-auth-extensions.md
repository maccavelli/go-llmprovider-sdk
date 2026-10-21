---
status: accepted
date: 2026-09-30
decision-makers: go-llmprovider-sdk maintainers
consulted: the sources of pi, grok-build, codex, kilocode and opencode; the documentation of antigravity-cli and claude-code
informed: every consumer of github.com/maccavelli/go-llmprovider-sdk
---
# Add Together AI, Kilo Device Login and Command-Sourced Keys, from the Reference-Client Survey

## Context and Problem Statement

On 2026-09-30 the owner asked for:

* an assessment of how six coding agents authenticate, by browser/OAuth and
  by API key, and of whether this module's `llmprovider` can be improved from
  those facts;
* Together AI to be added as a provider.

The evidence is recorded in
[0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).
Its comparison table shows that most of what the reference clients do,
`llmprovider` already does. That is the result of the 0011 audit and the
0012 work. The findings fall into three groups:

* **Settled already.** Two apparent differences are deliberate recorded
  divergences:
  * the Grok session host ([0006-MADR-subscription-auth-for-llm-providers.md](0006-MADR-subscription-auth-for-llm-providers.md),
    decision 2);
  * the Kilo chat path ([0012-PLAN-gateway-conventions.md](0012-PLAN-gateway-conventions.md)).
* **Current work.** Four findings land in phases of
  [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md)
  and
  [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md)
  that have not run yet. They are recorded as amendments to those records,
  not here:
  * re-reading the store after a rejected refresh, and the cross-process
    lock, amend
    [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md);
  * the `id_token` details also amend 0016-MADR;
  * a context-overflow error kind amends
    [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md);
  * the usage facts amend 0015-PLAN S9.
* **New.** This record decides these.

The question: **how is Together AI added, and which of the survey's new
capabilities does the SDK take on?**

## Decision Drivers

* **The owner's request.** Together AI is added.
* **Identity.** A request names this module, never a reference client
  ([0012-MADR-conform-providers-to-reference-clients.md](0012-MADR-conform-providers-to-reference-clients.md)
  §1.4).
* **Dependencies.** The standard library only in `llmprovider`
  ([0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
  D2).
* **One migration.** Work done now in `llmprovider` is ported by 0015-PLAN
  S7 anyway, so new code must be cheap to port.
* **Measured, not assumed.** A wire shape taken from another client's
  source, or from documentation, is confirmed by a live-tagged test before
  a release depends on it.

## Considered Options

For Together, the one choice with real alternatives:

* A. A dedicated `together` provider now, in `llmprovider`, on the shared
  Chat Completions code, ported by 0015-PLAN S7 like the others.
* B. A dedicated provider after S7, written directly as
  `llmprovider/providers/together` on the new contract.
* C. No dedicated provider: a generic Chat Completions provider configured
  by base URL and key, which would also serve Together.

## Decision Outcome

Chosen option: "A. A dedicated `together` provider now", because the owner
asked for Together now. The code is small: Hugging Face's provider, the
closest template, is 194 lines over the shared primitive. Porting it in S7
costs one more package among nine. B makes Together wait for S7. C loses
what the dedicated providers give: a curated listing, a descriptor for
`wizard`, and typed quirks. C remains a candidate for its own record.

### D1. Together AI

* **Identity.**
  * Provider id `together`.
  * Base URL `https://api.together.ai/v1`.
  * Key from `WithAPIKey`, sent as `Authorization: Bearer`.
  * The environment variable named in the catalog is `TOGETHER_API_KEY`
    (0017-REPORT, Together section; models.dev key `togetherai`).
* **Wire.** `POST {base}/chat/completions` through the shared Chat
  Completions code, with `max_tokens`.
  * A forced tool uses the object form of `tool_choice`, as the other
    Chat Completions providers do.
  * **The thinking path** sends `reasoning: {"enabled": true}`, and
    `reasoning_effort` only when the caller set an effort.
  * **The plain path** sends neither, so the model's default applies.

  pi sends `reasoning.enabled` to every toggleable model, and
  `reasoning_effort` only to models known to accept it (0017-REPORT P2). A
  live-tagged test confirms both shapes on one toggleable model and on
  `openai/gpt-oss-120b` before release.
* **Responses.** The existing decoder already reads `reasoning` and
  `reasoning_content`. `finish_reason: "eos"` is a normal stop; only
  `length` is truncation.
* **Listing.** `GET {base}/models` returns a bare array. Entries with
  `type: "chat"` are kept, then curated by the open-catalog ranking with
  the models.dev key `togetherai`.
  * There is **no generation probe**: every call is metered
    ([0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) D9).
  * The static fallback is a short list of tool-capable chat models, taken
    from models.dev when the provider is written.
* **Capabilities.** Tools, forced tool choice and reasoning are
  `BestEffort`, because support is per model on Together. There is no
  continuation: Chat Completions is stateless.
* **Descriptor.** An API-key method only. Together has no OAuth in any
  surveyed client.
* **Sequence.** It lands in `llmprovider` now, with its own G-wire case (new
  golden files, not a regeneration of existing ones). 0015-PLAN S7 ports it
  after `ollama`.

### D2. Kilo device login

* `auth` gains a Kilo device login:
  * `POST {base}/api/device-auth/codes` starts it;
  * `GET {base}/api/device-auth/codes/{code}` polls every 3 s until 200
    (approved, with `token`), 403 (denied) or 410 (expired)
    (0017-REPORT K1).
* It returns the handle of 0016-MADR D6.
* **The token has no refresh and no known expiry.** It is stored in the
  `TokenStore` and applied like an API key. It is never refreshed. A 401 is
  `ErrAuthFailure`, telling the caller to log in again.
* After login, `/api/profile` offers the organizations, for Kilo's
  organization option.
* The requests name this module; nothing presents as Kilo's own client.

### D3. Keys from a command

* `auth` gains a `TokenSource` that runs a caller-supplied command and uses
  its standard output as the token. This is the pattern of Claude Code's
  `apiKeyHelper`, pi's `!command` keys and Grok's
  `GROK_AUTH_PROVIDER_COMMAND` (0017-REPORT A2, P6, G section).
* **How it runs:**
  * The command is an argv. It never runs through a shell.
  * Each run has a timeout (default 10 s), and its output is capped
    (8 KiB, as 0016-MADR R3).
  * The result is cached for a TTL (default 5 minutes, as `apiKeyHelper`).
  * A 401 from the service runs the command once more.
* **Safety:**
  * The value redacts itself (0016-MADR D5).
  * A failure's message names the command, and never its output.
* **Ambient state.** Library code never finds a command on its own: the
  caller names it. So 0015-MADR D9 holds.

### D4. OpenAI sign-in without Codex's client id: probe, then decide

* pi's source shows a public-client flow with **dynamic client
  registration**, whose token is used on `api.openai.com/v1`: scope
  `chatgpt.tokens.use.direct`, resource `https://api.openai.com/v1`
  (0017-REPORT P3).
* If it works for any third-party agent, it removes 0016-MADR's recorded
  risk of borrowing Codex's client id.
* **The evidence is one day old and comes from one client.** No OpenAI
  document was read. *(Since read: see the amendment of 2026-10-16.)*
* **Decided:** a live-tagged probe, run only on the owner's request, records:
  * whether registration succeeds under this module's name;
  * which scopes are granted;
  * which Responses fields the token is refused.

  A later amendment to this record decides whether it becomes an OpenAI
  auth method. Nothing ships before that.

### D5. Session-affinity headers on Kilo: not now

* Kilo's client adds `X-Session-Id` and `x-session-affinity` beside the
  task id (0017-REPORT K2).
* Sending them would change every Kilo G-wire golden, and no benefit is
  measured.
* Revisit when a measurement shows cache or routing gains.

### D6. Not adopted, with the reason recorded

* **Anthropic Pro/Max and GitHub Copilot subscription logins.** Both
  surveyed implementations present as the vendor's own client:
  * pi sends `claude-cli` as its user agent and a required Claude Code
    system prompt;
  * pi sends Copilot's editor headers.

  That breaks 0012-MADR §1.4. OpenCode dropped its Anthropic flow.
  0016-MADR D10 stands.
* **Grok's chat proxy and `X-XAI-Token-Auth`.** 0006-MADR decision 2 stands.
  The marker identifies the official CLI, and the proxy refused a
  third-party session with 426.
* **Kilo's `/api/openrouter` path.** The 0011/0012 choice of `/api/gateway`
  stands.
* **OS keyring storage** (Codex, `agy`, Claude Code). It needs a module
  outside the dependency rule, or running platform tools. Not now.
* **OpenRouter, Kimi and Meta logins.** They belong to providers this
  module does not have. Each is a new provider, with its own record.

### Consequences

* Good, because Together is available without waiting for the
  restructure, on code that S7 already knows how to port.
* Good, because D2 gives Kilo users a login that needs no copied key, and
  D3 covers short-lived and vaulted keys in any provider without a new
  dependency.
* Good, because D4 tests the one finding that could remove a standing
  risk, before any code depends on it.
* Neutral, because D5 and D6 change nothing, but now carry their evidence.
* Bad, because S7 has one more provider to port. The G-wire suite grows by
  one case, and the parity map by no rows, since Together was never in
  `mcplib`.
* Bad, because D1's reasoning shapes rest on documentation and on pi's
  source until the live test runs. A model that rejects `reasoning` would
  fail on the thinking path until corrected.
* Bad, because D3 runs a caller's command inside the library's process.
  The caller owns what it runs; the library bounds how long it runs, and
  what it reads.

### Confirmation

* `together` passes G-wire (its own case) and the ported tests. S7 then
  moves it into `llmprovider/providers/together` with its `llmtest`
  harness.
* A live-tagged Together test (`LLMPROVIDER_LIVE_TOGETHER`) confirms:
  * text, a forced tool, and both thinking shapes of D1;
  * the listing's bare array.
* Kilo device login is tested against a fake server for approved, denied,
  expired and slow cases. A live test on the owner's request confirms it
  against the service.
* The command source is tested for a timeout, oversized output, a failing
  command, the TTL cache, the 401 re-run and redaction. Each test is first
  seen to fail.

*Annotated 2026-10-03, at 0017-PLAN's close-out: all four are met.* The
live Together and Kilo device-login checks ran that day, with the owner's
key and approval.

## Pros and Cons of the Options

### A. A dedicated `together` provider now

* Good, because it is available now, and it is typed, curated and offered
  by `wizard`.
* Bad, because it is ported once more in S7.

### B. A dedicated provider after S7

* Good, because it is written once, on the final contract.
* Bad, because the owner's request waits for the restructure.

### C. A generic Chat Completions provider

* Good, because one provider serves every compatible service.
* Bad, because it has no curated listing, no descriptor and no per-service
  quirks. Each caller would rediscover Together's reasoning fields.

## More Information

* Evidence:
  [0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).
* Implementation:
  [0017-PLAN-together-provider-and-auth-extensions.md](0017-PLAN-together-provider-and-auth-extensions.md).
* **Findings that amend current work,** each proposed in its own record:
  * 0016-MADR, "Amendment 2026-09-30": re-read the store after a rejected
    refresh; reopen D3's cross-process lock; the details of `id_token`
    checking.
  * 0015-MADR, "Amendment 2026-09-30": `ErrContextOverflow`.
  * 0015-PLAN: the S9 usage facts, and S7's order with `together`.

## Owner's decisions (2026-09-30)

The owner answered on 2026-09-30: "1. accept d1, proceed with d1-d5. 2. both as recommended. 3. add it."

* **D1–D5 are accepted as written.**
  * D1: Together lands now, in place (0017-PLAN U1).
  * D2 and D3 run after 0015-PLAN S4.
  * D4 is a probe that runs only when the owner asks.
  * D5 stays deferred.
* **The survey's amendments to current work:**
  * 0016-MADR A1 and A3 are accepted, and A2 as option (b);
  * 0015-MADR's `ErrContextOverflow` is accepted.

## Amendment 2026-10-03: what the OpenAI sign-in probe found

Status: **accepted** 2026-10-03, from 0017-PLAN U4. D4 said the probe's
output decides whether this flow becomes an OpenAI auth method. The owner
chose option B ("adopt option b").

### Fact found

`TestLive_OpenAISigninProbe` (`llmprovider/live_openai_signin_probe_test.go`)
ran on 2026-10-02 with the owner signing in. It follows pi's flow (0017-REPORT
P3) with agent name hint `go-llmprovider-sdk` and a per-run
`ext_agent_host_id`.

* **Registration succeeds** under this module's name. The callback carried
  an issued client id of 31 characters, beginning `oaia`.
* **Every scope asked for was granted:** `openid profile email
  offline_access resource.invoke chatgpt.tokens.use.direct`.
* **The token response** holds a 2105-character access token that is a JWT
  (`eyJh…`), a refresh token, and an `id_token`. It expires in 3600 s.
* **`api.openai.com/v1/responses` refuses this module's requests with the
  token**, HTTP 400:

  | Request | Refusal |
  |---|---|
  | text, as `openai.New` sends it | `Store must be set to false` |
  | a tool call | `Store must be set to false` |
  | text with `openai.WithStore(false)` | `Stream must be set to true` |
  | text with reasoning effort low | `Store must be set to false` |

  So the token is accepted only on unstored, streamed requests. pi always
  streams, and omits the fields it refuses. This module's `openai` provider
  has no native streaming: `NativeStreaming` is `Unsupported`, and
  `Generate` sends a non-streamed request.
* **Not measured:** whether a request with both `store: false` and
  `stream: true` is then accepted, and whether tool calls work there. The
  probe cannot send a streamed request through the provider.
* **No revocation endpoint is known** for the issued client, so the probe's
  session was left to expire. Each run registers another client.

### Considered options

* **A. Adopt it, with native streaming in `openai`.** The `openai` provider
  streams the Responses API, sending `store: false` for this token. `auth`
  gains the dynamic-registration flow, which refreshes with the issued
  client id, and `wizard` offers it. This removes the borrowed Codex client
  id, the risk 0016-MADR records, for API use. It needs native streaming
  (0015-MADR D4 allows it with no API change), a new flow, and its own MADR
  and PLAN.
* **B. Not now; keep the probe.** Record the findings and leave the probe
  off by default. Revisit when `openai` streams natively for other reasons,
  or OpenAI accepts non-streamed requests.
* **C. Reject it.** Remove the probe, and keep the Codex flow as the only
  OpenAI subscription path.

### Decided

Option B, "Not now; keep the probe":

* The flow does not become an OpenAI auth method now. The Codex flow stays
  the only OpenAI subscription path, with 0016-MADR's recorded risk of a
  borrowed client id.
* `TestLive_OpenAISigninProbe` stays, off by default, as the measurement to
  repeat.
* **Revisit when** `openai` streams natively for another reason, or OpenAI
  accepts a non-streamed request with this token. The probe then measures
  `store: false` with `stream: true`, which it cannot send today.

### Rejected

* **A, adopt it now,** because it needs native streaming in `openai`, a new
  sign-in flow and a wizard method, a MADR and PLAN of their own, for a path
  no consumer has asked for.
* **C, reject it,** because the probe costs nothing while off, and the
  borrowed-client-id risk is still open.

## Amendment 2026-10-16: the revisit trigger has fired, and OpenAI's documents are read

Status: **accepted** 2026-10-17; proposed 2026-10-16. The owner chose
option A ("proceed"). It answers
[go-llmprovider-sdk issue 2](https://github.com/maccavelli/go-llmprovider-sdk/issues/2),
whose text is [0017-REPORT-openai-sign-in-for-agents.md](../reports/0017-REPORT-openai-sign-in-for-agents.md). The
amendment of 2026-10-03 chose option B, "Not now; keep the probe", and named
when to revisit it.

### Facts found

* **The trigger of 2026-10-03 has fired.** `openai` has streamed natively
  since `eab8fce` (2026-10-10, first tagged `v1.10.0`;
  [0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md)).
  With an API key, `Stream` sends `stream: true`
  (`llmprovider/providers/openai/openai.go:394`), and `WithStore(false)`
  sends `store: false` (`openai.go:328-334`). The provider can now send the
  pair the probe could not.
  * It still sends `max_output_tokens` on every key request
    (`openai.go:327`), a field OpenAI lists as unsupported for this token.
* **A consumer has asked.** On 2026-10-14 gobble-cli's owner asked whether
  this flow keeps a ChatGPT subscription login within OpenAI's policy, and
  said to adopt it if so (the report, "Purpose"). The 2026-10-03 reason for
  rejecting option A, "a path no consumer has asked for", no longer holds.
* **OpenAI's documents are read.** The report read them in full on
  2026-10-14 (its F1–F3). On 2026-10-16 four pages under
  `https://developers.openai.com/siwc/token-sharing-open-source/` were read
  again (`sign-in`, `profiles-and-sessions`, `preview-limitations`,
  `models-and-inference`). Each still says what the report quotes:
  * "Set `store: false` and `stream: true`." Fifteen fields are to be
    omitted, `max_output_tokens` and `temperature` among them, and
    "Omit `previous_response_id` over HTTP".
  * "Group function/custom tools in namespaces or supply them through
    `additional_tools` input items." Responses `tool_search` is
    unsupported. No page shows a namespace's JSON.
  * `agent_name_hint` is "your app's actual name, used consistently across
    installations".
  * Sign-out revokes at "the `revocation_endpoint` from
    `https://auth.openai.com/.well-known/openid-configuration`".
  * `GET /v1/models` returns "a `models` array", with `slug`,
    `display_name` and `visibility`.
  * No page shows a date.

  This replaces D4's "No OpenAI document was read", and the 2026-10-03
  amendment's "No revocation endpoint is known".
* **A correction.** The report, and issue 2, say this module's probe "has
  not been run" (the report, "Method"). It ran on 2026-10-02, with the owner
  signing in (the amendment of 2026-10-03, above).
* **Still unmeasured:**
  * whether a request with `store: false` and `stream: true` is accepted;
  * whether a top-level function tool is accepted, as pi sends it, or only
    a namespace;
  * whether an unsupported field, such as `max_output_tokens`, is refused
    or ignored;
  * the model listing's shape with this token;
  * a refresh with the issued client id, and a revocation.
* **What adoption would change,** each checked at `v1.14.0`:
  * `auth.OAuthSession` has no `resource`, scopes, ID token or host id
    (`llmprovider/auth/tokenstore.go:21`), which refresh and re-sign-in
    need (the report, F1 and F6);
  * the agent name has to come from the consumer, since the terms (§1) and
    the `sign-in` page ask for the app's own name;
  * `openai` would gain a third request shape: the platform URL, with
    `store: false` and `stream: true`, without `max_output_tokens` or
    `previous_response_id`, without native `tool_search`, and with no
    billed model probes;
  * there is no device flow, so only the browser login applies.

### Considered options

* **A. Probe again with streamed requests, then decide.** 0017-PLAN U5
  extends the probe to measure the five unmeasured items, and to revoke
  what it signs in. The next amendment decides adoption with the results.
  If it adopts, the design takes a new MADR at the next free number,
  since it is a new auth method, session record, request shape and wizard
  method.
  * Good, because adoption is decided on measured acceptance, as D4 asked,
    and the 2026-10-03 amendment named this measurement.
  * Good, because a run now ends by revoking its session, which U4's could
    not.
  * Bad, because it needs the owner at a browser once, and registers one
    more client under the account.
  * Bad, because the consumer waits for two records, not one.
* **B. Adopt now; design in a new MADR, and measure in its first phase.**
  * Good, because the consumer's answer comes one record sooner.
  * Bad, because it decides before the one refusal that blocked the path
    in 2026-10-02's run is measured again.
* **C. Keep "not now".**
  * Bad, because both of its conditions have since changed: the trigger
    fired, and a consumer asked.
* **D. Reject it, and remove the probe.**
  * Bad, because it keeps the borrowed Codex client id (0016-MADR's
    recorded risk) as the only OpenAI subscription path, against a
    consumer's request.

### Decided

Option A, "Probe again with streamed requests, then decide":

* 0017-PLAN U5 runs on the owner's request, after this amendment is
  accepted.
* Nothing ships from it: no default changes and no exported API is added.
* Issue 2 stays open until the amendment that decides adoption. A comment
  correcting "has not been run" is posted only on the owner's ask.

## Amendment 2026-10-17: what the streamed probe found, and adoption

Status: **accepted** 2026-10-18; proposed 2026-10-17. The owner chose
option A ("go with recommendations, proceed"). It is the amendment that
D4, and the amendment of 2026-10-16, leave to decide whether this flow
becomes an OpenAI auth method. Its evidence is 0017-PLAN U5's run of
2026-10-17, with the owner signing in.

### Facts found

* **The token works on streamed, unstored requests, through today's
  provider.** `openai.New` with the access token as `WithAPIKey`, and
  `WithStore(false)`, streamed a text reply (`"ALPHA"`) and a forced
  top-level tool call (`get_weather({"city":"Paris"})`). So pi's form,
  tools at the top level, is accepted; no namespace is needed.
* **`Generate` is still refused,** with the same messages as on
  2026-10-02: `Store must be set to false`, then `Stream must be set to
  true`. A request is accepted only when it streams.
* **`max_output_tokens` is not refused.** A raw request with OpenAI's
  documented minimum completed, and so did the same with
  `max_output_tokens: 64`. OpenAI's page says to omit it; whether the
  service applies it or ignores it is unmeasured, since the reply fitted.
  The provider's streamed request sends it too, and was accepted.
* **The listing is not the platform's.** `GET /v1/models` answered HTTP
  200 with one top-level key, `models`: 9 models, 7 with
  `visibility: "list"`, the first `gpt-6.1-sol`. The `openai` provider's
  key listing reads the platform's `data` array, so it would not read
  this one.
* **Refresh and revocation work as documented.** A refresh with the issued
  client id and no `scope` returned a rotated refresh token and the same six
  scopes. Revocation at the discovery document's `revocation_endpoint`,
  `https://auth.openai.com/api/accounts/oauth/revoke`, answered HTTP 200
  with an empty body, and a refresh afterwards was refused `invalid_grant`.
  Every run can now end by revoking its session.
* **Registration** succeeded again, and every scope was granted. The
  issued client id was 32 characters this time, and 31 on 2026-10-02, so
  its length is not fixed.
* **Not measured:**
  * the other fourteen fields OpenAI lists as unsupported;
  * the `subscription_sharing_*` errors, among them the usage limit that
    can fail a stream after it has begun;
  * native `tool_search` (0040-MADR D3), which OpenAI lists as
    unsupported, and `previous_response_id`;
  * the two models not listed for a picker.

### What adoption takes

From these findings and the amendment of 2026-10-16:

* **A login.** The browser flow with `dynamic_agent_client`, the agent name
  taken from the consumer (the terms, §1), a host id kept for the host's
  life, and the issued client id kept per account. There is no device
  flow.
* **A session record** that keeps what a refresh and a re-sign-in need:
  the issued client id, `resource`, the scopes, the ID token and the host
  id. `auth.OAuthSession` has none of the last four.
* **A request shape in `openai`.** The platform URL, `store: false`, and
  always streamed, `Generate` reading the stream as a ChatGPT session's
  does today. Continuation is Unsupported. Deferred tools are appended,
  never native. No billed model probes, and the listing read from
  `models`. The provider picks its backend by the kind of credential
  (`llmprovider/providers/openai/openai.go:5-8`), and pi tells this token
  from a key only by a missing `sk-` prefix (the report, F5), so the
  session has to say what it is.
* **Errors:** `subscription_sharing_*` codes mapped to the existing
  classes, and the usage URL exposed for a consumer's "Manage usage".
* **The wizard** offers the login beside the Codex one.
* **The terms in a consumer.** Consent before background use, and "connected
  application only", are the consumer's to meet; the SDK can carry the
  consent as an option and the usage URL as a value, not meet them alone.

### Considered options

* **A. Adopt it, designed in a new MADR.** The flow becomes an OpenAI auth
  method. Its design, the items above, takes a new MADR and PLAN at the
  next free number, since it adds a login, a session record, a request
  shape and a wizard method. The Codex-id login stays beside it, as pi
  keeps both; whether it is ever retired is that MADR's question.
  * Good, because it gives consumers an OpenAI subscription login under
    their own name, which removes 0016-MADR's borrowed-client-id risk for
    those who use it.
  * Good, because it is now measured: text, a tool call, refresh and
    revocation all work with the provider's streamed request.
  * Good, because a consumer asked for it.
  * Bad, because OpenAI's pages carry no dates and call the limits
    "Preview limitations", so the shape may move under it.
  * Bad, because it is a large addition: a second OpenAI login, a session
    kind and a third request shape to keep working.
* **B. Document the token only.** No login: a caller that obtains a token
  itself passes it as `WithAPIKey` and calls `Stream` with
  `WithStore(false)`.
  * Good, because it costs one paragraph.
  * Bad, because `Generate` is refused, the listing and refresh do not
    work, and each consumer writes the login itself.
* **C. Not now; keep the probe.**
  * Bad, because nothing it waited for is still missing: streaming works,
    and a consumer asked.
* **D. Reject it.**
  * Bad, because the flow works, and it is the only OpenAI subscription
    login that does not borrow a vendor's client id.

### Decided

Option A, "Adopt it, designed in a new MADR":

* The design goes to a new MADR and PLAN, at the number
  `scripts/check_records.py --next` gives when they are written (0043
  today). Nothing ships before that PLAN is approved. *(Written
  2026-10-18: [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md).)*
* The probe stays, off by default, as the measurement to repeat.
* Issue 2 is answered with this decision once it is accepted. Whether it
  closes then, or when the method ships, is the owner's choice.

## Amendment 2026-10-21: the device token can be kept as a key

D2 says the Kilo device login's token "is stored in the `TokenStore` and
applied like an API key". It stays so by default.
[0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)
D2 adds one exception: with the wizard's `Options.KiloDeviceKey`, the token
is returned as the `Result`'s API key, nothing is stored, and no
organization is chosen. The `auth` package's login is unchanged: only what
the wizard does with its token differs.
