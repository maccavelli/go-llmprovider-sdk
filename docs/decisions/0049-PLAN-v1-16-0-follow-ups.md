---
status: in-progress
date: 2026-10-19
associated-madr: "0049-MADR-v1-16-0-follow-ups.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Closing v1.16.0's Follow-ups

Associated MADR: [0049-MADR-v1-16-0-follow-ups.md](0049-MADR-v1-16-0-follow-ups.md)

## Goal

The MADR's D1 to D5, as the owner answered its questions (Q1 `v1.17.0`; Q2
all of D2a to D2d; Q3 the learned refusals in `catalog.Cache`):

* no exported call panics on a nil `*StaticToken` or a zero `DeviceLogin`;
* each pair of like failures matches one kind;
* the learned forced-choice refusals live in `catalog.Cache`, and the 0034
  tests pass at `-count=2`;
* the six untested paths of F5 and F6 are tested;
* `v1.17.0` is released, with no exported name added or changed.

## Scope

### In scope

* `llmprovider/settings.go`, `llmprovider/auth/oauth_device.go`,
  `llmprovider/auth/oauth_loopback.go`, `llmprovider/catalog/discovery.go`,
  `llmprovider/catalog/cache.go`, `llmprovider/internal/cacheref`,
  `llmprovider/internal/wire/messages`, `llmprovider/providers/claude`,
  `llmprovider/providers/opencode`, and the tests beside each;
* tests only in `llmprovider/providers/{gemini,claude,grok,ollama}` and
  `wizard`;
* the docs of every function whose errors change (R52), R49 in
  `docs/guides/api-standards.md`, `docs/architecture.md` and R2's table,
  `CHANGELOG.md`, the README's Status.

### Out of scope

* Any exported name added, changed or removed (R48); `apidiff` against
  `v1.16.0` reports nothing.
* Other typed-nil token sources, which the MADR did not probe.
* gobble-cli's and prepare-commit-msg's adoption of `v1.17.0`.

## Rules for every phase

* **Approval.** The owner accepts the MADR and approves this PLAN, whole or
  by phase; a phase starts after the owner has committed the last.
* **The gate, before staging:** `python3 -B tools/gate.py`; `make
  pre-add-check FILES="…"` on the phase's Go files; `tools/claims.py` on
  both records; `make records-check`; `tools/recordlint.py --changed`;
  `tools/denyscan.py --changed`. Each exit status is captured before any
  filter, and read before anything is written from it.
* **Red first.** Every new or changed test is seen to fail on a planted
  break, by `tools/plant.py` on a scratch clone, never in the tree; the
  phase's record holds each plant and its failure line.
* **The docs.** A function whose errors change has its doc changed in the
  same phase, naming only what a test shows (R52); `make docs-check` stays
  at 0.
* **The API and the wire.** `tools/apidiff.py v1.16.0` reports no change;
  G-wire's goldens do not change (R45); `make coverage-check` holds every
  floor (R47).
* **Stage, then the owner commits** with `git commit --no-edit`; push and
  tags on the owner's word.
* **Stop conditions:** a fact of the MADR or a claim below no longer holds;
  a step would add or change an exported name; a golden changes; a doc
  would state a behaviour no test shows. Stop, record a dated deviation
  here (and an amendment in the MADR when a decision or a fact changes),
  and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.

### Phase 1: errors in place of the panics (D1)

1. **`settings.sendable`** (C1): a `*StaticToken` that is nil is refused,
   `fmt.Errorf("%w: WithTokenSource: a nil *StaticToken", ErrInvalidRequest)`,
   before its value is read. `ResolveOptions`'s doc names it.
2. **`(*DeviceLogin).Wait`** (C2, C3): with no `poll`, it returns
   `fmt.Errorf("%w: oauth: a DeviceLogin that StartDeviceOAuth did not return", llmprovider.ErrInvalidRequest)`
   at once, before taking its lock. `Wait`'s doc and `DeviceLogin`'s say so.
3. **Tests** (`_0049$`): `ResolveOptions` and `providers.New(ProviderClaude, …)`
   with a typed-nil `*StaticToken` give `ErrInvalidRequest` and do not
   panic; a zero `DeviceLogin`'s `Wait` gives `ErrInvalidRequest`, twice,
   and its `Cancel` still returns.
4. **Red first:** each check removed in turn; the test fails (a recovered
   panic is reported as a failure, not a crash).

### Phase 2: one kind for like failures (D2)

1. **D2a,** `callbackError` (C5): each `access_denied` branch, the Codex one
   included, wraps `llmprovider.ErrAuthFailure` as `"%w: oauth: …"`; the
   other codes stay plain.
2. **D2b,** `errDeviceCodeExpired` (C6) becomes
   `fmt.Errorf("%w: oauth: device code expired", llmprovider.ErrAuthFailure)`;
   its three uses are unchanged.
3. **D2c,** `checkOllamaURL` (C9, C10): an unreachable server as the listing
   gives it (C7), and a status other than 200 through
   `llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)`
   (C8).
4. **D2d,** `resolveOAuthFlowConfig` (C11): `"%w: oauth: provider %q is not
   supported"` with `llmprovider.ErrUnsupported`.
5. **Tests:** 0048's tests that assert "plain" for these (the browser's IdP
   `access_denied` and unsupported provider, the device's unsupported
   provider, `LoginDeviceOAuth`'s Start error, `ValidateOllamaURLWith`'s
   refused connection and HTTP 503) assert the new kind instead, and its
   message still holds today's text; new `_0049` tests: a Grok device code
   past its own expiry matches `ErrAuthFailure`; the browser's Codex
   `access_denied` matches it; `ValidateOllamaURL`, without options, as
   `ValidateOllamaURLWith`.
6. **Docs:** `StartDeviceOAuth`, `Wait`, `LoginBrowserOAuth`,
   `ValidateOllamaURL` and `ValidateOllamaURLWith` name the new kinds.
7. **Red first:** each of the four wraps removed in turn; its test fails.
8. **`CHANGELOG.md`**, `[Unreleased]`: D2 under Changed, each function and
   the kind it now matches; D1 under Fixed.

### Phase 3: the learned refusals in `catalog.Cache` (D3)

1. **`internal/wire/messages`** (C12): the memo becomes a type, `Refusals`,
   with its mutex, order and set, the bound per memo; `NewRefusals()`; the
   process's `processRefusals`; a nil `*Refusals` is the process's.
   `Policy` gains its `*Refusals`; `NewPolicy` (C13) takes it;
   `refusesForcedChoice` (C14) and `RefusalRetry.Check` read and learn
   through the request's policy.
2. **`internal/cacheref`** (C15): `Value` gains `Refusals
   *messages.Refusals` and a nil-safe accessor; `catalog.Cache` (C16) makes
   one with the others.
3. **Claude and OpenCode** (C17, C18): `New` passes
   `cacheref.From(st.Values()).Refusals()` to `messages.NewPolicy`.
4. **Tests:** the 0034 tests that learn give each provider a
   `catalog.WithCache(new(catalog.Cache))`; `forgetRefusals` resets the
   process's; new `_0049` tests: two caches do not share a learned refusal,
   and no cache is the process's. `go test -count=2 -run _0034
   ./llmprovider/...` passes, where the MADR's F4 measured 10 failures.
5. **Docs:** R49 loses its exception, and names the refusals beside the
   listings, metadata and probes; `catalog.Cache`'s and `WithCache`'s docs
   likewise; `docs/architecture.md` (`cacheref`'s row, its imports) and
   R2's row for `internal/cacheref` (`internal/wire/messages`); `make
   tables-check`.
6. **Red first:** a provider's `New` that drops the cache's refusals; a
   `Policy` that learns into the process's memo whatever it was given.
7. **`CHANGELOG.md`**, `[Unreleased]`: D3 under Changed.

### Phase 4: the tests 0048 left out (D4)

1. **F5:** in each of `providers/{gemini,claude,grok,ollama}`, a test as
   `providers/openai/probe_cache_0048_test.go` is: a provider built
   `WithCache(&a)` with probes on probes each listed model once; a second
   listing through `a` sends none; through `b`, and through the process's,
   it probes again.
2. **F6:** in `wizard`, a failed browser sign-in and a failed ChatGPT plan
   sign-in through `ConfigureLLM`, each returned as the login gave it
   (`errors.Is` on the login's error), as
   `TestConfigureLLM_DeviceLoginErrorBare_0048` is for the device code.
   `ConfigureLLM`'s doc (C19) then names a failed sign-in's error of every
   method.
3. **Red first:** the MADR's six plants (F5, F6), each now caught.

### Phase 5: release `v1.17.0` (D5)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.17.0] - <date>`, the date
   that of the commit the tag will name; an empty `[Unreleased]` above it;
   the comparison links. The README's Status names `v1.17.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.16.0`, no change;
   the disclosure guard over the outgoing commits, run by the agent before
   the owner pushes.
3. **The owner tags `v1.17.0`** and pushes it; CI's `release` job makes the
   Release; its body equals `check_changelog.py --section v1.17.0 --repo
   maccavelli/go-llmprovider-sdk`.
4. **After the tag:** `tools/consumer.py v1.17.0`; `tools/offline.py
   v1.17.0`.
5. **Close:** the MADR's Confirmation items, each with its evidence; this
   PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | A typed-nil `*StaticToken` and a zero `DeviceLogin` give `ErrInvalidRequest`, never a panic | 1 | the `_0049` tests; the MADR's probe again |
| V2 | Each of D2a to D2d matches its kind, and keeps today's text | 2 | the changed and new tests |
| V3 | Two caches do not share a learned refusal; the 0034 tests pass at `-count=2` | 3 | `go test -count=2 -run _0034 ./llmprovider/...` |
| V4 | The MADR's six plants are caught | 4 | `tools/plant.py` |
| V5 | No exported name changes; the docs name what the tests show | all | `tools/apidiff.py v1.16.0`; `make docs-check` |
| V6 | `v1.17.0`'s Release body is its section | 5 | `gh release view`; `--section` |
| V7 | Every new or changed test failed first | 1–4 | the execution record |

## Rollout and Rollback

* **Rollout.** Five phase commits on `main`, staged by the agent and
  committed by the owner; then the owner's tag.
* **Rollback.** Each phase reverts on its own, newest first. D2's kinds are
  the one change a caller sees; reverting Phase 2 restores the plain
  errors. Phase 3 reverts with its test changes.

## Code claims

<!-- claims: self=c1eec68 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `sendable` reads a `*StaticToken`'s value | `llmprovider/settings.go:235` | `if st, ok := s.tokens.(*StaticToken); ok && hasControl(st.Value) {` |
| C2 | `Wait` | `llmprovider/auth/oauth_device.go:134` | `func (d *DeviceLogin) Wait(ctx context.Context) (*OAuthSession, error) {` |
| C3 | `Wait` calls `poll` | `llmprovider/auth/oauth_device.go:156` | `session, err := d.poll(pollCtx)` |
| C4 | `callbackError` | `llmprovider/auth/oauth_loopback.go:562` | `func callbackError(query url.Values) error {` |
| C5 | Its plain `access_denied` | `llmprovider/auth/oauth_loopback.go:583` | `return fmt.Errorf("oauth: authorization failed: %s", code)` |
| C6 | The code's own expiry | `llmprovider/auth/oauth_device.go:431` | `var errDeviceCodeExpired = errors.New("oauth: device code expired")` |
| C7 | The listing's unreachable Ollama | `llmprovider/catalog/discovery.go:874` | `return nil, fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL, err)` |
| C8 | The listing's status | `llmprovider/catalog/discovery.go:879` | `return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)` |
| C9 | The check's unreachable Ollama | `llmprovider/catalog/discovery.go:929` | `return fmt.Errorf("could not reach Ollama at %s: %w", baseURL, err)` |
| C10 | The check's status | `llmprovider/catalog/discovery.go:934` | `return fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)` |
| C11 | A login's unsupported provider | `llmprovider/auth/oauth_loopback.go:214` | `return oauthFlowConfig{}, fmt.Errorf("oauth: provider %q is not supported", provider)` |
| C12 | The memo | `llmprovider/internal/wire/messages/learn.go:19` | `var learned = struct {` |
| C13 | `NewPolicy` | `llmprovider/internal/wire/messages/forcing.go:39` | `func NewPolicy(fallback llmprovider.ForcedChoiceFallback, overrides map[string]llmprovider.Support) Policy {` |
| C14 | The memo's reader | `llmprovider/internal/wire/messages/toolchoice.go:31` | `func refusesForcedChoice(provider, model string) bool {` |
| C15 | `cacheref.Value` | `llmprovider/internal/cacheref/cacheref.go:13` | `type Value struct {` |
| C16 | `catalog.Cache` makes its value | `llmprovider/catalog/cache.go:28` | `c.ref = &cacheref.Value{Catalog: newStore(), Probes: transport.NewProbeStore()}` |
| C17 | Claude's policy | `llmprovider/providers/claude/claude.go:181` | `forcing:   messages.NewPolicy(st.ForcedChoiceFallback(), st.ModelForcedChoices()),` |
| C18 | OpenCode's policy | `llmprovider/providers/opencode/opencode.go:219` | `forcing:     messages.NewPolicy(st.ForcedChoiceFallback(), st.ModelForcedChoices()),` |
| C19 | `ConfigureLLM`'s doc | `wizard/configure.go:151` | `// errors.Is matches it; so is a failed device-code sign-in's, as` |

## Execution record

### Phase 0: records (2026-10-19)

* **No deviation.** The owner committed the MADR and this PLAN as
  `941e547`, with the MADR's questions answered (Q1 `v1.17.0`; Q2 all of
  D2a to D2d; Q3 the learned refusals in `catalog.Cache`), and said
  "committed, proceed".
* `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`,
  both index rows with them; "records-check: 0 problem(s)".

### Phase 1: errors in place of the panics (2026-10-19)

* **No deviation.** Phase 0 was committed by the owner as `0979446`.
* **Step 1,** `settings.sendable`: a nil `*StaticToken` is refused, "%w:
  WithTokenSource: a nil *StaticToken" with `ErrInvalidRequest`, before its
  value is read; `ResolveOptions`'s doc names it.
* **Step 2,** `(*DeviceLogin).Wait`: with no `poll`, "%w: oauth: a
  DeviceLogin that StartDeviceOAuth did not return" with
  `llmprovider.ErrInvalidRequest`, at once, before the lock (`poll` is set
  once, by `StartDeviceOAuth`). `Wait`'s doc says so; `DeviceLogin`'s says
  only the one `StartDeviceOAuth` returns can be waited on.
* **Step 3, the tests,** under `-race`:
  `TestResolveOptions_NilStaticToken_0049` (`ResolveOptions` and
  `providers.New(ProviderClaude, …)` with a typed-nil `*StaticToken` give
  `ErrInvalidRequest`; a nil `TokenSource` is still no source) and
  `TestDeviceLogin_ZeroWaitIsAnError_0049` (two `Wait`s on a zero
  `DeviceLogin`, each `ErrInvalidRequest`; `Cancel` returns; a third `Wait`
  likewise). Each recovers a panic and fails with it.
* **Step 4, red first,** three plants, all caught: the nil check removed
  ("nil_token_0049_test.go:19: ResolveOptions panicked: runtime error:
  invalid memory address or nil pointer dereference; want an error"); the
  zero check removed ("device_login_zero_0049_test.go:21: Wait 1 on a zero
  DeviceLogin panicked: …"); the refusal without its sentinel
  ("nil_token_0049_test.go:37: … want ErrInvalidRequest").
* **The checks:** `make docs-check`, "sdkaudit: 0 violation(s) of
  R50,R51,R52"; `tools/apidiff.py v1.16.0`, no output, exit 0; `make
  pre-add-check` on the four Go files, "4 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck)".
* **The gate,** first run: 26 of 27. `gate-selftest`'s
  `test_docs_check_concurrency_sentence_removed` plants a break by finding
  `DeviceLogin`'s two concurrency lines as 0048 wrote them, and step 2's
  sentence, wrapped onto the second, changed it ("found 0 times"). The
  sentence was moved to a line of its own, leaving those two lines as they
  were; the case then passed, and `tools/gate.py` on the final tree, 27 of
  27 passed.
