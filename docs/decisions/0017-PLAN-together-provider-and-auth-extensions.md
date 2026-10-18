---
status: complete
date: 2026-10-18
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

### Phase U5: the probe, streamed (accepted 2026-10-17, on request only)

From the MADR's amendment of 2026-10-16, option A. It runs only once that
amendment is accepted and the owner asks; its run needs the owner at a
browser.

1. **The report's correction.** In
   [0017-REPORT-openai-sign-in-for-agents.md](../reports/0017-REPORT-openai-sign-in-for-agents.md),
   annotate "has not been run" (its "Method"), and append a dated
   correction naming the 2026-10-02 run.
2. **Extend `llmprovider/live_openai_signin_probe_test.go`.** After the
   exchange, it:
   1. reads `GET https://api.openai.com/v1/models` with the token, and logs
      the top-level keys, the count, and how many entries have
      `visibility: "list"`. The rest uses the first `list` slug; with none,
      it uses `signinModel` and says so;
   2. sends two raw streamed requests to `/v1/responses`: OpenAI's
      documented minimum (`model`, `input`, `store: false`,
      `stream: true`), and the same with `max_output_tokens`. It logs each
      status, and the error's code and message;
   3. calls `Stream` through `openai.New`, with `WithAPIKey` and
      `WithStore(false)`, for one text and one top-level tool call;
   4. refreshes with the issued client id, the refresh token and
      `resource`, without `scope`, and logs whether new access and refresh
      tokens came back;
   5. revokes the latest refresh token at the discovery document's
      `revocation_endpoint`, with the issued client id, and logs the status
      and the body's length. A refresh with the revoked token follows, and
      is expected to be refused.

   U4's four generations stay, as the non-streamed baseline. As in U4, a
   failed registration or exchange fails the test; every other outcome is a
   logged finding.
3. **Logging stays as U4's:** shapes and outcomes, never a token or the
   issued client id.
4. **Red first.** The listing's and the revocation's parsing are helpers
   with an offline test beside the probe. Each is seen to fail on a planted
   break, by `tools/plant.py` on a scratch clone.
5. **Checks:** `go vet -tags live_gateways ./...`, `make pre-add-check` on
   the file, and `tools/gate.py`. No default changes, and no exported API
   is added (`tools/apidiff.py`).
6. **The run's findings** become the MADR's next amendment, which decides
   whether this becomes an OpenAI auth method. The report gets a pointer
   to it.

## Verification

| # | Criterion | Check | Phase |
|---|---|---|---|
| V1 | Together requests match D1, and no existing golden changes | unit tests; G-wire | U1 |
| V2 | Together works against the service | the live test, on request | U1 |
| V3 | `wizard` offers Together with an API key | descriptor test | U1 |
| V4 | Kilo device login handles every outcome, and never refreshes | fake-server tests | U2 |
| V5 | The command source is bounded, cached and redacted | unit tests | U3 |
| V6 | The OpenAI probe exists and is off by default | `go vet -tags live_gateways`; no default changed | U4 |
| V7 | The probe measures streamed requests, the listing, refresh and revocation, and is still off by default | the offline helper tests, seen to fail first; the run's log; `tools/apidiff.py` | U5 |

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
  *Annotated 2026-10-01:* 0015-PLAN, "Phase S7, commit 14", moved it to
  `providers/together`.
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

### Phase U3: keys from a command (2026-09-30)

* **Step 1.** `command_token.go`: `CommandToken` (`NewCommandToken(argv...)`).
  * **Running:** argv only, never a shell. `Timeout` defaults to 10 s, and a
    one-second `WaitDelay` reaps pipes. Output is capped at 8 KiB, trimmed,
    and cached for `TTL`, 5 minutes by default. Callers arriving together
    share one run.
  * **Errors:** empty output, a timeout, too much output or a non-zero exit
    is `ErrAuthFailure`, naming the command's base name. The output, and
    stderr, which is never read, never appear. An empty argv is
    `ErrInvalidRequest`.
  * **Formatting:** `String`, `GoString`, `LogValue` and `MarshalJSON` show
    the command's base name and the header only. Not the arguments, which
    may carry secrets, and not the token.
* **The 401 re-run.** A new exported interface, `InvalidatingSource`
  (`TokenSource` plus `Invalidate()`). `CommandToken` implements it by
  dropping its cache. `expireOpenAISession` and `expireGrokSession`, the two
  providers' existing 401-then-retry-once paths, now invalidate any such
  source, as they already expired an `OAuthSession`.
* **Step 2.** Today `NewProviderWithSource` takes a source for `openai` and
  `grok` only. The other providers get one with 0016 T3, in 0015-PLAN S7.
  Until then, a `CommandToken` serves those two, and listing for every
  provider (`ListModelCatalogWithSource`).
* **Step 3, tests** (`command_token_test.go`). The test binary acts as the
  key command (`TestHelperKeyCommand`, gated by an environment variable the
  tests set). The tests cover:
  * trimming, TTL reuse, and a rerun after the TTL;
  * a timeout, oversized output, a non-zero exit, empty output and a missing
    command, each an auth failure without the output;
  * eight concurrent callers, one run;
  * an OpenAI request refused with 401, then retried after one rerun;
  * redaction of the arguments and the token in every form.
* **A bug the tests found before commit.** `cappedBuffer` first embedded
  `bytes.Buffer`. `io.Copy`, which `exec` uses for the command's output,
  prefers the destination's `ReadFrom`, which bypassed the capping `Write`.
  The oversized test got `Token = <nil>`. The buffer is now a field, with
  the reason in a comment.
* **Red first** is by absence: `CommandToken` and `InvalidatingSource` are
  new. Seen to fail on deliberate breaks:

  | Break | Failure |
  |---|---|
  | a 401 does not invalidate | `Generate = "", llm: authentication failed: openai HTTP 401: expired key after 1 runs (sent [Bearer key-1]); want one rerun after the 401` |
  | the TTL ignored | `Token = "key-2", <nil>; want the cached key-1` |
  | no shared run | `Token = "key-2"`, `"key-3"`, `"key-4"`, … |
  | the cap raised to 8 MiB | `… wrote more than 8388608 bytes, want … "more than 8192 bytes"` |
  | a failure carrying the output | `the error carries the command's output: … OUTPUT-PLANTED-SECRET-7f3a` |
  | `String` showing the arguments | `%v = CommandToken{Command:/usr/local/bin/vault-key --secret=PLANTED-SECRET-7f3a …}` |

  The helper's output grows with the constant, so the cap break shows the
  cap's value, not an accepted overflow. The overflow path itself is the
  bug above, which the same test caught.
* **Lint.** Two findings were fixed: an error that is now wrapped with
  `%w`, and a private `token` method renamed `cachedToken` so it no longer
  differs from `Token` only by case.
* **Gate,** every step exit 0: `llmprovider` 90.1 %, `wizard` 83.4 %,
  `make lint` `0 issues.`, G-wire unchanged, and the deny list at 0 hits.

### Phase U4: OpenAI dynamic-registration probe (2026-10-02)

Requested by the owner ("proceed", after U4 was offered as the open step).
The probe follows 0017-REPORT P3 and pi's
`packages/ai/src/auth/oauth/openai-chatgpt.ts` at `312184edb`, read
locally: the public client `dynamic_agent_client`, an agent name hint, an
`ext_agent_host_id`, `resource https://api.openai.com/v1`, and the issued
client id in the callback. Its run waits for the owner at a browser.

* **Done, 2026-10-02 and 2026-10-03.**
  * **Step 1, the probe.** `llmprovider/live_openai_signin_probe_test.go`,
    tagged `live_gateways`, switched on by `LLMPROVIDER_LIVE_OPENAI_SIGNIN`.
    * It runs the flow on `127.0.0.1:1455`, trades the code with the issued
      client id, and makes four generations through `openai.New` with
      `WithAPIKey`: text, a tool call, `WithStore(false)`, and reasoning
      effort low.
    * It logs only shapes and outcomes: a length and a prefix of at most
      four characters, never a token or the client id.
    * `go vet -tags live_gateways` is clean, and the file passes
      `golangci-lint` with the live tag.
  * **Step 2.** No default changed, and no exported API was added.
  * **The run,** on the owner's request and sign-in: the test passed in
    47 s. Its findings:
    * registration succeeded, and every scope was granted;
    * every generation was refused, first for `store`, then for `stream`.
  * **Step 3.** 0017-MADR amendment "what the OpenAI sign-in probe found",
    proposed, holds the findings and options A–C for the owner.
  * **Found, not changed:** linting with `--build-tags live_gateways` reports
    a `goimports` finding in `llmprovider/live_gateways_test.go:36`, a file
    U4 does not touch. CI lints without the tag, so it has not shown there.
  * **V6 is met:** the probe exists, and is off by default.
* **Deviation, 2026-10-03: the `goimports` finding.**
  * **Found** while linting the probe with `--build-tags live_gateways`.
    `llmprovider/live_gateways_test.go:36` puts
    `github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint`
    in the standard-library import group.
    * Linting the whole module with the tag shows it is the only finding.
    * CI's `make lint` runs without the tag, so it never showed there.
    * The file is outside U4's scope.
  * **Decided.** The owner chose to fix it ("fix the finding"). The import
    moves to the module group beside `internal/transport`, and nothing else
    changes.
  * **Done.** The import is in the module group. `golangci-lint run -c
    .golangci.yml --build-tags live_gateways ./...` reports `0 issues.`, and
    `go vet -tags live_gateways` is clean. The failing run above is the
    check's first failure.
* **What still keeps this PLAN open** (the MADR's Confirmation):
  * the Together live test (V2), which has not run: it is billed, and needs
    `TOGETHER_API_KEY` and the owner's request;
  * a live Kilo device login, "on the owner's request". No live test of it
    exists yet; only the fake-server tests of U2 do.

### Confirmation items: the live runs (2026-10-03, started)

The owner asked for all three open items ("Together live test, CI lints with
the live tag, Kilo device login live test").

* **Kilo device login.** `TestLive_KiloDeviceLogin` is added to
  `llmprovider/live_oauth_login_test.go`, behind `LLMPROVIDER_LIVE_DEVICE_LOGIN`
  like the Grok one. It signs in with `auth.LoginDeviceOAuth(ctx,
  ProviderKilo, …)`, checks the session holds no refresh token and no expiry
  (U2: the approved token never refreshes), reads `auth.KiloProfile`, and
  generates once on a free model. It logs no token and no e-mail address.
* **Together.** `TOGETHER_API_KEY` is not set in this session, so the run
  waits for it.
* **Deviation, 2026-10-03: a Kilo device-login session is refused by
  `kilo.New`.**
  * **Found** in `TestLive_KiloDeviceLogin`, run with the owner approving the
    code. The login succeeded: no refresh token and no expiry, and the
    profile read back with 0 organizations and a personal account. Then
    `kilo.New` failed: `unsupported: kilo takes an API key or a Kilo
    device-login token, not an OAuth session (0016-MADR D10)`.
  * **Cause.** `auth`'s Kilo device login returns an
    `*auth.OAuthSession{Provider: kilo}` (`llmprovider/auth/kilo_device.go`),
    and `kilo.New` refuses every `*auth.OAuthSession`.
    * D2 says the token "is stored in the `TokenStore` and applied like an
      API key", and the error message itself allows "a Kilo device-login
      token".
    * `TestNew_RefusesAnOAuthSession` checks only an OpenAI session.
    * So a consumer following `wizard`'s `CredOAuth` result cannot build
      Kilo.
  * **Decided.** The owner chose "accept a Kilo session". `kilo.New` accepts
    an `*auth.OAuthSession` whose `Provider` is `kilo`, sent in
    `Authorization: Bearer` as a key is. It still refuses another provider's
    session and a vendor CLI login. A 401 must stay `ErrAuthFailure` (D2:
    log in again).
  * **Tests:** a unit test of the accepted session, its header, and its 401,
    seen to fail first; then the live login, rerun.
* **Deviation, 2026-10-03: Together refuses `"required"` on
  `gpt-oss-120b`.**
  * **Found** on the first run of the Together live tests (V2). Text, the
    forced tool, both thinking shapes, `none` and the listing passed, with
    172 chat models. `TestLive_TogetherToolChoices/required` failed with
    `HTTP 500 server_error`, 4 runs out of 4.
  * **Measured,** with `"required"` and with a named tool:
    * `openai/gpt-oss-120b`: `"required"` gets HTTP 500; a named tool
      calls.
    * `deepseek-ai/DeepSeek-V4.1-Flash`, `zai-org/GLM-5.3`,
      `moonshotai/Kimi-K3` and `zai-org/GLM-5.3-Flash`: both call.
  * **Decided.** The owner chose "measured model, pin the 500".
    * `required` runs on `DeepSeek-V4.1-Flash`.
    * A subtest pins `gpt-oss-120b`'s 500.
    * Together's package doc names the degradation. Forced tool choice
      was already `BestEffort`.
  * **Recorded, not changed:** the 500 is classified
    `ErrProviderUnavailable`, which `WithRetry` retries, though it never
    succeeds.
* **Done, 2026-10-03.**
  * **Kilo:** `kilo.New` accepts an `*auth.OAuthSession` whose `Provider` is
    `kilo`. It refuses another provider's session and a vendor CLI login,
    each with its own message.
    * `TestNew_AcceptsAKiloDeviceSession` (`llmprovider/providers/kilo/device_session_test.go`)
      was red first: `New with a Kilo device-login session: llmprovider:
      unsupported: kilo takes an API key or a Kilo device-login token, not
      an OAuth session`. Now it passes: `Authorization: Bearer <token>`, and
      a 401 is `ErrAuthFailure`.
    * **First-fail of the refusal:** with the provider check disabled on a
      scratch copy, `TestNew_RefusesAnOAuthSession` fails: `oauth: err =
      <nil>, want ErrUnsupported`.
    * The package doc says which sessions are accepted.
  * **Together:**
    * `required` runs on `deepseek-ai/DeepSeek-V4.1-Flash`.
    * `TestLive_TogetherRequiredFailsOnGptOss` pins the 500.
    * The package doc gives the measured models.
    * **Live:** every Together test passes, `TestLive_TogetherWire` (4
      subtests), `TestLive_TogetherToolChoices`,
      `TestLive_TogetherRequiredFailsOnGptOss` and
      `TestLive_TogetherListing` (172 chat models). The two changed tests
      passed 3 more runs of 3. **V2 is met.**
    * **First-fails, live,** on scratch copies:
      * `required` sent as `auto` fails the pin: `Generate = <nil>, want
        the HTTP 500`;
      * `required` sent as `none` fails `ToolChoices/required`: `a call came
        back: false, want true`.
  * ~~**The Kilo live login after the fix:** its run timed out waiting for the
    approval (`device-code wait: context deadline exceeded`). It waits for
    the owner, so V4's live check is still open.~~ *Annotated 2026-10-03:*
    rerun with the owner approving ("ready"), and it passes; see below.
  * **The gate:** all 17 checks pass, with `make lint` now tagged.

### Close-out (2026-10-03)

* **`TestLive_KiloDeviceLogin` passes** (15 s), with the owner approving the
  code.
  * The session has an access token, no refresh token and no expiry.
  * `KiloProfile` read 0 organizations and a personal account.
  * The fixed `kilo.New` generated on `kilo-auto/free` with the session.
  * Nothing is revoked: Kilo has none.
* **The MADR's Confirmation, each item:**
  * `together` passes G-wire and the ported tests, U1, and S7 moved it with
    its `llmtest` harness;
  * the live Together tests pass today: text, a forced tool, both thinking
    shapes and the listing;
  * Kilo device login passes the fake-server tests of U2, and today the
    live login;
  * the command source's tests were each seen to fail first, U3.
* **Verification:** V1–V6 are met. V2 and V4's live check were met today;
  V6 by U4's probe.
* **U4's decision** is the MADR's amendment of 2026-10-03, option B.
* This PLAN is `complete`. *(Reopened 2026-10-17 for U5, below.)*

### Phase U5: the probe, streamed (2026-10-17)

The owner accepted the MADR's amendment of 2026-10-16, option A, and asked
for U5 ("proceed").

* **Step 1, the report's correction.** The report's "has not been run" is
  annotated, and "Correction (2026-10-17)" is appended to it.
* **Step 2, the probe.** `llmprovider/live_openai_signin_probe_test.go`
  gains, after the exchange: the listing, two raw streamed requests,
  streamed text and a streamed tool call through `openai.New` with
  `WithStore(false)`, a refresh, a revocation and a refresh after it. The
  post-sign-in steps have their own three minutes. U4's four generations
  stay, and now use the listing's first model rather than `signinModel`,
  so both sets ask the same model.
  * The revocation sends `token_type_hint=refresh_token` beside the token
    and the issued client id. OpenAI's pages name only the endpoint; the
    hint is RFC 7009's, and optional.
* **Step 3, logging.** Shapes and outcomes only. The run's log holds no
  token and no issued client id, which `probeShape` describes by length
  and a prefix of four characters.
* **Step 4, red first.** The parsing is in
  `llmprovider/signin_probe_0017_test.go`, untagged, with
  `TestSigninProbeHelpers_0017`. `tools/plant.py` ran eight plants on a
  scratch clone:

  | Plant | First run | Second run |
  | :--- | :--- | :--- |
  | the listing counts hidden models as listed | caught | caught |
  | the listing accepts the platform's `data` array | **not caught** | caught |
  | the revocation endpoint may be plain http | caught | caught |
  | an OAuth refusal loses its description | caught | caught |
  | a refusal's param is dropped | caught | caught |
  | a failed stream reads as completed | caught | caught |
  | text deltas are not counted | caught | caught |
  | a stream with no terminal event reads as accepted | caught | caught |

  The miss: with the `"models"` check bypassed, decoding the absent array
  still errs, with a vaguer message. The run's finding is that message, so
  the test now asserts it names `no "models" array`; the second run's
  failure read `the listing's "models" is not an array of models:
  unexpected end of JSON input, want an error naming no "models" array`.
* **Step 5, checks.**
  * `make pre-add-check` on both files: `2 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)`.
  * `go vet -tags live_gateways ./llmprovider/`: clean.
  * `tools/apidiff.py HEAD`: no output, exit 0.
  * `tools/gate.py`: 20 of 23. `api-check`, `generate-check` and
    `gate-selftest` failed on `lookup proxy.golang.org: no such host`
    while fetching their pinned tools. Once the name resolved, each was
    run again in the tree: `api-check` "0 incompatible change(s)" against
    `v1.14.0`, `generate-check` "2 generated file(s), 0 problem(s)", and
    `gate-selftest` 14 tests, OK.
* **The run,** 2026-10-17, with the owner signing in: the test passed in
  203.6 s. Its findings, as logged:

  | Step | Finding |
  | :--- | :--- |
  | registration | succeeded; issued client id of 32 characters, starting `oaia` |
  | scopes | all six granted, `chatgpt.tokens.use.direct` among them |
  | token response | a 2106-character JWT access token, a refresh token and an `id_token`; `expires_in` 3600 s |
  | listing | HTTP 200; top-level keys `["models"]`; 9 models, 7 with `visibility: "list"`; the first `gpt-6.1-sol` |
  | raw streamed, the documented minimum | accepted, completed, after 2 text deltas |
  | raw streamed, with `max_output_tokens` | accepted, completed, after 2 text deltas |
  | text (`Generate`) | REFUSED, HTTP 400: `Store must be set to false` |
  | tool call (`Generate`) | REFUSED, HTTP 400: `Store must be set to false` |
  | text with `WithStore(false)` (`Generate`) | REFUSED, HTTP 400: `Stream must be set to true` |
  | text with reasoning effort low (`Generate`) | REFUSED, HTTP 400: `Store must be set to false` |
  | streamed text with `WithStore(false)` | accepted, text `"ALPHA"` after 2 text deltas |
  | streamed tool call with `WithStore(false)` | accepted, `get_weather({"city":"Paris"})` |
  | refresh with the issued client id | accepted; rotated; the same six scopes; 3600 s |
  | revocation | `https://auth.openai.com/api/accounts/oauth/revoke`: HTTP 200, an empty body |
  | refresh after revocation | REFUSED, HTTP 400, `invalid_grant`: "The refresh token has been invalidated. A fresh sign-in is required." |

  The run registered one more client under the owner's account. Its
  refresh token is revoked; the registration itself stays listed in the
  account's ChatGPT settings.
* **Step 6.** The MADR's amendment of 2026-10-17, "what the streamed probe
  found", holds these findings and the adoption options, `proposed`, for
  the owner. The report points to it.
* **V7 is met:** the probe measures streamed requests, the listing,
  refresh and revocation; the helpers' test was seen to fail on each
  plant; the probe is still off by default; no exported API changed.
* **Not done here:** nothing ships from U5. ~~This PLAN stays
  `in-progress` until the owner decides the amendment.~~ The owner
  accepted it on 2026-10-18, option A; the design and its
  implementation are
  [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)
  and its PLAN. This PLAN is `complete`.
