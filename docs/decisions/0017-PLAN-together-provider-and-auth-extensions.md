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

* **2026-09-30, U1 step 4.** The step says the listing "bounds the body as
  the other listings do". They do not: `fetchDataIDs`,
  `fetchHuggingFaceUsable` and the other listers decode the body unbounded.
  Together's listing is bounded at 8 MiB (`togetherListingLimit`), because
  it also lists image, audio and embedding models. That is a
  correction of the step's premise, not of what it asks for. The other
  listers are unchanged; bounding them is outside this plan.

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

### Phase U1: Together AI, in place (2026-09-30)

* **Steps 1–5.**
  * `ProviderTogether = "together"` and `TOGETHER_API_KEY` in
    `ProviderEnvVars`.
  * `together.go`, on the shared Chat Completions code.
  * `NewProvider`.
  * `StaticTogether`: six models from models.dev's `togetherai` entry on
    2026-09-30 (tool-calling, not deprecated, one per vendor).
  * The metadata key `togetherai`.
  * `fetchTogetherUsable`: a bare-array decoder keeping `type: "chat"`,
    curated by the open-catalog ranking, with no probe.
  * An API-key descriptor.
* **The thinking path** sets the shared builder's existing `Reasoning`
  option to `{"enabled": true}`, and `ReasoningEffort` only from
  `WithReasoningEffort`. The plain path sets neither.
* **Step 6, tests** (`together_test.go`):
  * request shapes for six combinations of effort, thinking and tool;
  * `eos` as a normal stop;
  * the listing's filter and order, no generation from it, and the static
    fallback for a non-array body and for a listing with no chat model;
  * a key is required;
  * `together` added to `TestNewProvider` and `TestStaticModels`.
* **Seen to fail,** each break in its own scratch copy, never this tree:

  | Break | Failure |
  |---|---|
  | The plain path sends reasoning | `reasoning = map[enabled:true], want it absent` |
  | An effort sent unasked | `reasoning_effort = medium, want it absent` |
  | The listing keeps non-chat models | `DiscoverModels = [zai-org/GLM-5.3 openai/gpt-oss-120b BAAI/bge-large-en-v1.5 black-forest-labs/FLUX.2 new/Chat-Model], want [...]`, and `DiscoverModels = [x], <nil>; want the static catalog` |
  | `eos` treated as truncation | `GenerateItems = <nil>, llm: invalid request: response incomplete: length; want text "done", finish "eos", no error` |
  | An empty key accepted | `NewTogether with no key: want an error` |
  | The listing sends a probe | `listing sent a generation to "/chat/completions"` |
  | The key under another scheme | `path "/chat/completions", Authorization "Token tg-key"` |

* **Step 7, G-wire.** A `together` case, with its six goldens recorded by
  `go test -run 'TestWireGoldens/together/' -update`. G-wire now has 100
  files. `git status` showed only the new `testdata/wire/together/`
  directory; no existing golden changed. The goldens show:
  * the thinking body with `reasoning: {"enabled": true}` and no
    `reasoning_effort`;
  * the plain body with neither.

  The proxy test of 0016 T1 covers `together` too, since it walks the wire
  cases.
* **Step 8, live.** `live_together_test.go` (`live_gateways`,
  `LLMPROVIDER_LIVE_TOGETHER=1` with `TOGETHER_API_KEY`) covers text, a
  forced tool, the toggle model's thinking, `gpt-oss-120b` with effort
  `high`, and the listing. It vets clean.
  * **Not run.** It needs the owner's key and request, and it is billed.
    Until it runs, D1's reasoning shapes rest on Together's reference and
    pi's source, as the MADR's Consequences say.
* **Step 9, docs.** `docs/architecture.md` has ten provider ids and the G-wire
  counts, and `AGENTS.md` names the live variable.
* **Step 10.** 0015-PLAN S7's order already names `together`, from the
  accepted amendment. S10's environment helper will read `TOGETHER_API_KEY`
  from `ProviderEnvVars`, which already has it.
* **Gate,** every step exit 0:
  * `make pre-add-check`;
  * `go vet` for darwin, linux and windows (`CGO_ENABLED=0`), and with
    `-tags live_gateways`;
  * `go test -race -count=1 -cover ./...`: `internal/redact` 100.0 %,
    `internal/wiretest` 95.2 %, `llmprovider` 90.2 %, `wizard` 83.4 %;
  * `go mod tidy -diff`, empty;
  * `make lint`, `0 issues.`;
  * `make parity-check`;
  * markdownlint;
  * G-wire three times over;
  * G-links;
  * the deny list over every changed file: 0 hits.

### Phase U2: Kilo device login (2026-09-30)

* **Step 1.** `kilo_device.go`: `StartDeviceOAuth(ctx, ProviderKilo, opts)`
  runs Kilo's flow on the 0016 D6 handle.
  * **Start:** `POST {origin}/api/device-auth/codes`, where the origin is
    `https://api.kilo.ai`, or `opts.Issuer`. A 429 is `ErrRateLimited`. An
    unsafe verification URL, or a code with a control character or `/`, is
    refused.
  * **Poll:** `GET …/codes/{code}` (path-escaped) every 3 s, until:
    * 202, pending;
    * 403, denied;
    * 410, expired;
    * 200, approved, with a token of more than 10 characters.

  The expiry is `expiresIn`, else 10 minutes. `Cancel` works as for the
  other device flows. Kilo is kept out of the OAuth flow configuration, so
  `LoginBrowserOAuth` still refuses it.
* **Step 2.** The approved token becomes an `OAuthSession` with no refresh
  token and no expiry, so `Token` returns it and never refreshes. The wizard
  saves it to the `TokenStore`.
  * **Choice recorded: the `Result` kind.** The wizard returns it as
    `CredAPIKey`, the token in `APIKey`. That is D2's "applied like an API
    key", and every consumer's existing Kilo path works unchanged. 0016 T4
    (D11) will revisit what a `Result` carries for a stored credential.
  * **A later 401** is Kilo's existing `ErrAuthFailure`. The provider takes
    the token as a key, so its message cannot say "log in again". A
    source-aware message belongs with 0016 T3, which gives every provider a
    `TokenSource`.
* **Step 3.** `KiloProfile(ctx, token, opts...)` reads
  `{origin}{prefix}/api/profile`, routed as the gateway's endpoints are.
  It returns the email, the organizations, the selected organization and
  whether a personal account exists. When the account has organizations,
  the wizard offers them (and "Personal account" when there is one), with
  the account's selected organization as the default. The choice goes to
  the new `Result.Organization`, and into the wizard's listing through
  `WithKiloOrganization`. An unreadable profile warns, and keeps the
  personal account.
* **Step 4.** Kilo's descriptor gains two methods: "Kilo API key" and "Sign
  in with Kilo (device code)". As before, only the key is offered without a
  `TokenStore`.
  * `TestDescriptors_NoOAuthOnOtherProviders` pinned "only OpenAI and Grok
    offer a device code". It now also allows Kilo's device code, the one
    addition D2 makes, and still fails any other provider offering OAuth.
* **Step 5, tests:**
  * `llmprovider/kilo_device_test.go`:
    * approved after two pending polls, with three 3 s waits, a session with
      no refresh and no expiry, and the token hidden when formatted;
    * denied, expired, an unexpected poll status, a 429 start, and an unsafe
      verification URL;
    * `Cancel` within one interval;
    * a stored login making no request when its token is read;
    * `KiloProfile`.
  * `wizard/kilo_device_test.go`: a personal-only account; choosing an
    organization; choosing the personal account; and an unreadable profile.
* **Red first,** against a clean clone of `86817bb`, with the flow tests,
  which use only the old API. Exit 1: `oauth: provider "kilo" is not
  supported` for the approval, each outcome, and the 429 start. The profile,
  cancel and stored-login tests name identifiers the old code lacks, and are
  proven by deliberate breaks.
* **Seen to fail on deliberate breaks:**

  | Break | Failure |
  |---|---|
  | polls every 1 s | `sleeps [1s 1s 1s] over 3 polls, want three 3 s waits` |
  | a denial read as pending | `Wait = oauth: device code expired, want an error naming "denied"` |
  | the verification URL not checked | `start accepted a plain-http, non-loopback verification URL` |
  | the session given an expiry | `session = OAuthSession{… Expiry:2024-11-13T22:13:29Z …}` |
  | the chosen organization dropped | `Result kind "api_key" key-set true organization ""; want … "org-1"` |
  | the login not saved | `stored session OAuthSession(nil), want the token with no refresh` |

  The first denial break did not compile, a duplicate `case`. The break
  runner reported it as invalid, and it was rewritten.
* **Gate,** every step exit 0: `llmprovider` 90.0 %, `wizard` 83.4 % (both
  above their `P7` floors), `make lint` `0 issues.`, G-wire unchanged, and
  the deny list at 0 hits.
