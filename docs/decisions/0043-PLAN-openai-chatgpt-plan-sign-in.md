---
status: proposed
date: 2026-10-18
associated-madr: "0043-MADR-openai-chatgpt-plan-sign-in.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement OpenAI's Sign in with ChatGPT for Agents as an OpenAI Auth Method

Associated MADR: [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)

## Goal

A consumer signs a user in with their ChatGPT plan under the consumer's own
app name, and uses the `openai` provider with that session: generate,
stream, tools, listing, refresh and sign-out. No existing session, key,
stored record or wire golden changes behaviour.

## Scope

### In scope

* `llmprovider/auth`: the session's fields and kind (D1), refresh and
  revocation (D2), the store record (D3), `LoginChatGPTPlan` (D4).
* `llmprovider/providers/openai`: the plan request shape, listing and
  `ChatGPTUsageURL` (D5, D6); `llmprovider/internal/chatgpt`'s session
  check; `catalog.ListRaw` for a plan session.
* `llmprovider/api_error.go`: the usage-limit code (D6).
* `wizard`: the method, `Options.AgentName`, the notice (D7).
* Docs: `auth`'s package doc and the README (D8), the API standards'
  rows, `docs/architecture.md`, the release notes.
* The live test that replaces the probe (D9).

### Out of scope

* Retiring the Codex-id logins (D7).
* A device flow; OpenAI offers none.
* Native `tool_search`, continuation, or any unsupported field for plan
  sessions.
* Any change to an existing G-wire golden.

## 0. Preconditions and conventions

* **Approval.** The owner accepts the MADR and approves this PLAN, phase by
  phase or whole.
* **Gate.** Each phase ends on `tools/gate.py` (23 checks), `make
  pre-add-check` on its Go files, `tools/apidiff.py` against `v1.14.0`,
  `tools/denyscan.py --changed` and `make records-check`, and is staged for
  the owner's `git commit --no-edit`.
* **Red first.** Every new test is seen to fail on a planted break, by
  `tools/plant.py` on a scratch clone, and the plant table goes in this
  record.
* **Stop conditions.**
  * An existing G-wire golden changes.
  * An existing session's refresh, revocation or backend changes.
  * A live run contradicts M1–M9.
  * A step needs a module outside the dependency rule.

## Implementation Steps

### Phase 0: baseline and names (records only)

1. Record `<base>`, `tools/gate.py` at `<base>`, and the coverage of
   `auth`, `providers/openai` and `wizard`.
2. Check every exported name in MADR D1, D4, D5, D6 and D7 against
   `docs/guides/api-standards.md`. A rule a name breaks becomes a deviation
   and a MADR amendment before Phase 1.
3. Re-read the discovery document and compare it with M9.

### Phase 1: the session (D1, D2, D3)

1. `OAuthSession` gains `Resource`, `Scopes`, `IDToken` and `HostID`;
   `PlanUsage()`; `ChatGPT()` false for a plan session. `String`,
   `GoString`, `MarshalJSON` and `LogValue` hide `IDToken`.
2. The refresh of a plan session: a form to `TokenURL` with `resource` and
   no `scope`. The lock, re-read and rotation are unchanged.
3. `RevokeOAuthSession` for a plan session: an RFC 7009 form to the
   discovered `revocation_endpoint`, with the issued client id.
4. `fileRecord` gains the four fields, `omitempty`. `ValidateOAuthSession`
   requires `TokenURL` for a plan session.
5. **Tests** (fake issuer): each field round-trips the file store; an
   older record (no new fields) loads as before; `PlanUsage` and
   `ChatGPT()` for each kind; the refresh form's fields and the absence of
   `scope`; the revocation form and endpoint; the R33 formatting test with
   a planted ID token; a Codex-id session's refresh and revocation
   byte-for-byte unchanged.

### Phase 2: the login (D4)

1. `ChatGPTPlanOptions` and `LoginChatGPTPlan`, reusing the loopback
   listener, PKCE, the callback handler, discovery and the ID-token
   verifier.
2. The first sign-in's and the re-sign-in's authorize parameters; the
   issued client id from the callback; a mismatched client id refused; the
   direct scope required; `AgentName` required; the pasted-callback
   fallback.
3. **Tests** (fake issuer): both authorize URLs, parameter by parameter;
   `localhost` never used; the exchange with the issued id; a bad
   signature, audience or nonce; a grant without the direct scope; an
   empty `AgentName`; a callback with no client id or another one; the
   generated host id's form.

### Phase 3: the provider (D5, D6)

1. `internal/chatgpt.IsSession` false for a plan session (it follows
   `ChatGPT()`); `providers/openai` detects `PlanUsage()` and uses the
   platform URL, `store: false`, `stream: true`, no `max_output_tokens`,
   Continuation `Unsupported`, deferred tools appended, `prompt_cache_key`
   and `include` as with `WithStore(false)`.
2. `ListModels` and `catalog.ListRaw` read the `models` array; no probes.
3. `subscription_sharing_usage_limit_exceeded` joins `openAIQuotaTypes`;
   `openai.ChatGPTUsageURL`.
4. **Tests:** new G-wire goldens for the plan shape (text, tool call,
   reasoning, deferred tools); every existing golden unchanged; the
   listing from a `models` body and the refusal of a `data` body; no probe
   sent; the four `subscription_sharing_*` codes classified, in a reply
   and mid-stream; the capabilities.

### Phase 4: the wizard (D7)

1. `AuthChatGPTPlan` in `llmprovider/descriptor.go` and the OpenAI
   descriptor; `wizard.Options.AgentName`; the method offered only with
   it and a store; the one-time notice; save and logout through the
   store.
2. **Tests:** offered and not offered; the notice shown once; the session
   saved under `openai`, replacing a Codex-id one; logout revokes it.

### Phase 5: docs, the live test, and the release

1. `auth`'s package doc and the README: the method, and what stays the
   consumer's (D8). `docs/guides/api-standards.md` rows and
   `docs/architecture.md` for any layout change (R43).
2. `llmprovider/live_chatgpt_plan_test.go`, switched on by
   `LLMPROVIDER_LIVE_CHATGPT_PLAN`: sign in, `Generate`, `Stream`, a tool
   call, `ListModels`, a forced refresh, revoke, and a refresh after it
   refused. `live_openai_signin_probe_test.go` and its helpers are removed;
   `AGENTS.md`'s live-test list names the new variable.
3. The live run, with the owner signing in.
4. Release notes for `v1.15.0`, the downgrade cost of D3 among them. The
   owner commits, pushes and tags; after the tag: CI, the proxy,
   `tools/consumer.py` and `tools/offline.py`; issue 2 is answered and
   closed on the owner's ask.

## Verification

| # | Criterion | Check | Phase |
| :--- | :--- | :--- | :--- |
| V1 | A plan session stores, refreshes and revokes as M7 and M8 measured | fake-issuer tests; the live test | 1, 5 |
| V2 | Every existing session, record and golden behaves as before | the unchanged-behaviour tests; G-wire | 1, 3 |
| V3 | The login follows OpenAI's flow and refuses what D4 refuses | fake-issuer tests | 2 |
| V4 | The provider's plan requests match D5, and its errors D6 | goldens; classification tests | 3 |
| V5 | The wizard offers the method only with an app name and a store | wizard tests | 4 |
| V6 | The API changes are additions only, and exactly the MADR's | `make api-check`; `tools/apidiff.py v1.14.0` | every |
| V7 | The live test passes with the owner signing in | the run's log | 5 |
| V8 | Every new test failed first on a planted break | `tools/plant.py` tables | every |

## Rollout and Rollback

* Additive: a consumer that never calls `LoginChatGPTPlan` sees no change.
  The release is `v1.15.0`.
* Rollback before the tag: revert the phase's commit. After it: a patch
  release that stops offering the method in the wizard; stored plan
  records then fail to refresh, and the user signs in with the Codex-id
  login (D3).
