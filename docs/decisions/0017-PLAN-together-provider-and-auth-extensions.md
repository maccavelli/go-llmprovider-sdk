---
status: in-progress
date: 2026-09-30
associated-madr: "0017-MADR-together-provider-and-auth-extensions.md"
decision-makers: go-llmprovider-sdk maintainers
---
# Implement Together AI, Kilo Device Login and Command-Sourced Keys

Associated MADR: [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md)

## Goal

1. `together` is a provider in `llmprovider`, covered by G-wire and a live
   test, and offered by `wizard` (MADR D1).
2. Kilo has a device login (D2), and any provider can take a key from a
   command (D3), both built on the handle and redaction of
   [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md)
   T2.
3. The OpenAI dynamic-registration probe exists and runs only when the owner
   asks (D4).

## Scope

### In scope

* `llmprovider/**` and `wizard/**` for the steps below, and their tests.
* `llmprovider/testdata/wire/together/**`: new golden files only.
* `docs/architecture.md`, `docs/README.md`, `AGENTS.md` (the new live-test
  variable), and this pair.

### Out of scope

* The MADR's D5 and D6.
* Any change to an existing G-wire golden.
* A generic Chat Completions provider (MADR option C).
* Anything 0015-PLAN or 0016-PLAN does. The amendments that the survey
  proposes to them are theirs.

## 0. Preconditions and conventions

* **Order.**
  * U1 needs only U0. It lands before 0015-PLAN S4.
  * U2 and U3 need 0016-PLAN T2 (the device handle, and redaction), so they
    run after 0015-PLAN S4.
  * U4 runs only on the owner's request.
* **Gate.** Every phase ends on 0015-PLAN's §0 gate, including G-wire, then
  commits with `git commit --no-edit`.
* **Red first.** Every new test is first seen to fail on a scratch copy, and
  the failure is quoted in the execution record.
* **Stop conditions.**
  * An existing G-wire golden changes.
  * A live test contradicts D1's wire shapes.
  * A step needs a module outside the dependency rule.

## Implementation Steps

### Phase U0: accept (docs only)

1. The owner decides the MADR's D1–D5, and the proposed amendments to 0015
   and 0016.
2. On acceptance:
   * set the MADR `accepted` and this PLAN `in-progress`;
   * update `docs/README.md`;
   * commit the records only (bootstrap exception).

### Phase U1: Together AI, in place

1. **Identity.** Add `ProviderTogether = "together"` (`constants.go`), and
   the environment-variable entry `TOGETHER_API_KEY`.
2. **Provider.** Add `together.go` on the shared Chat Completions code,
   following `huggingface.go`:
   * `NewTogether`, and the eight generation methods;
   * `max_tokens`;
   * the thinking path sends `reasoning: {"enabled": true}`, plus
     `reasoning_effort` only when an effort was set;
   * the plain path sends neither;
   * `DiscoverModels` has no probe.
3. **Construction.** Wire `together` into `NewProvider`, into the static
   catalog (a short list of tool-capable chat models from models.dev,
   dated), and into the metadata key `togetherai` (`model_metadata.go`).
4. **Listing.** Add a bare-array decoder for `GET {base}/models`. It keeps
   `type: "chat"`, curates with the open-catalog ranking, and bounds the body
   as the other listings do.
5. **Descriptor.** Add an API-key-only descriptor, so `wizard` offers
   Together, and `TestDescriptors_CoverEveryRegisteredProvider` stays green.
6. **Tests:**
   * request bodies for each path, including that the plain path has no
     `reasoning` fields;
   * `eos` decodes as a normal stop;
   * the listing drops non-chat entries, and falls back to the static list
     on error;
   * the descriptor.
7. **G-wire.** Add a `together` case to `TestWireGoldens`: seven scenarios,
   `continuation` skipped. Record its goldens with `-update`, scoped to that
   case. The gate proves that no existing golden changed.
8. **Live.** Add `live_together_test.go` (`live_gateways` tag, switched on
   by `LLMPROVIDER_LIVE_TOGETHER` and `TOGETHER_API_KEY`). It covers text, a
   forced tool, both thinking shapes, on one toggleable model and on
   `openai/gpt-oss-120b`, and the listing. It runs on the owner's request,
   and its output goes in the execution record.
9. **Docs.** `docs/architecture.md` lists the provider, and `AGENTS.md`
   names the live variable.
10. **Hand-off to 0015.** Its S7 order gains `together` after `ollama` (the
    0015-PLAN amendment). Its S10 environment helper gains
    `TOGETHER_API_KEY`.

### Phase U2: Kilo device login

1. In the auth code (`llmprovider`, until 0015-PLAN S7b moves it), add a
   Kilo device login on the 0016 T2 device handle:
   * start with `POST {base}/api/device-auth/codes`;
   * poll `GET …/codes/{code}` every 3 s;
   * map 202, 403, 410 and 200, and a 429 at the start;
   * honour the handle's cancel and the code's `expiresIn`.
2. Save the token through `TokenStore` as a credential with no refresh,
   applied like an API key. A 401 later is `ErrAuthFailure` asking for a new
   login.
3. After login, read `/api/profile` for the organizations, and let the
   caller choose one.
4. **Wizard.** Kilo's descriptor gains the device method. It is offered only
   with a `TokenStore`, like the other stored methods.
5. **Tests** against a fake server:
   * approved;
   * denied;
   * expired;
   * slow approval;
   * a 429 at the start;
   * cancel;
   * the saved credential never refreshing;
   * redaction of the token in every formatted value.

### Phase U3: keys from a command

1. Add the command `TokenSource` of MADR D3:
   * argv only;
   * a timeout (default 10 s) and an 8 KiB output cap;
   * a TTL cache (default 5 minutes);
   * trimmed output;
   * one re-run after a 401 reported by the provider.
2. Any provider takes it through `WithTokenSource` (0016 T3).
3. **Tests:**
   * a timeout;
   * oversized output;
   * a non-zero exit (the message names the command, never its output);
   * an empty output;
   * TTL reuse;
   * the 401 re-run;
   * concurrent callers running the command once;
   * redaction.

### Phase U4: OpenAI dynamic-registration probe (on request only)

1. Add `live_openai_signin_probe_test.go`, tagged `live_gateways` and
   switched on by `LLMPROVIDER_LIVE_OPENAI_SIGNIN`. It runs the browser flow
   of 0017-REPORT P3 under this module's agent name, and records:
   * the issued client id's form;
   * the granted scopes;
   * one text and one tool call on `api.openai.com/v1/responses`;
   * which request fields are refused.
2. It changes no default and adds no exported API.
3. Its output becomes an amendment to the MADR, which decides whether it
   becomes an auth method.

## Verification

| # | Criterion | Check | Phase |
|---|---|---|---|
| V1 | Together requests match D1, and no existing golden changes | unit tests; G-wire | U1 |
| V2 | Together works against the service | the live test, on request | U1 |
| V3 | `wizard` offers Together with an API key | descriptor test | U1 |
| V4 | Kilo device login handles every outcome, and never refreshes | fake-server tests | U2 |
| V5 | The command source is bounded, cached and redacted | unit tests | U3 |
| V6 | The OpenAI probe exists and is off by default | `go vet -tags live_gateways`; no default changed | U4 |

## Rollout and Rollback

* No consumer imports the module before `v1.0.0-rc.1`, so each phase is
  reverted with its commit.
* `together` is additive: removing it leaves the other providers and their
  goldens unchanged.
* U2 and U3 add auth methods a caller must choose. Nothing existing changes
  behaviour.

## Deviation Log

None.

## Execution Record

### Phase U0: accept (2026-09-30)

* The owner answered on 2026-09-30: "1. accept d1, proceed with d1-d5. 2. both as recommended. 3. add it."
* The MADR is `accepted`, D1–D5 as written:
  * D4 is a probe that runs only on the owner's request;
  * D5 stays deferred.
* This PLAN is `in-progress`.
* **The same answer decided the survey's amendments:**
  * 0016-MADR A1 and A3, and A2 as option (b);
  * 0015-MADR's `ErrContextOverflow`.
* The standards guide's R25 names the new kind.
* The commit holds records and the guide only.
