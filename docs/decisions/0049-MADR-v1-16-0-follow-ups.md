---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: this module at c1eec68 and its v1.16.0 tag (14a5b63); probes on a scratch copy of c1eec68 (method below); prepare-commit-msg at 87f0e64; 0048-PLAN-adopt-go-sdk-best-practices.md's execution record, which found these
informed: gobble-cli and prepare-commit-msg; anyone who matches this module's errors with errors.Is
---

<!-- markdownlint-disable MD013 -->

# Close v1.16.0's Follow-ups: Errors in Place of Two Panics, One Kind for Like Failures, the Learned Refusals in `catalog.Cache`, and the Tests 0048 Left Out

## Context and Problem Statement

`v1.16.0` is released, tagged on `14a5b63`; CI passed on `main` and on the
tag, and its Release carries its changelog section.
[0048-PLAN-adopt-go-sdk-best-practices.md](0048-PLAN-adopt-go-sdk-best-practices.md)
is `complete`, and its record names five things it found and left out of its
scope, for a record of their own:

1. **Two panics** a caller can reach through the exported API.
2. **Uneven error kinds:** like failures on like paths match different
   sentinels, or none.
3. **The learned refusals:** the one in-memory state outside `catalog.Cache`,
   which R49 of `docs/guides/api-standards.md` names as its exception.
4. **Probe wiring untested** in four providers.
5. **Two sign-in errors untested** through `wizard.ConfigureLLM`, so its doc
   names only the device-code sign-in's.

Nothing yet decides any of them. Each is measured below.

### How the facts were found (2026-10-19)

* **[probed]**: a test file added to a scratch copy of the tree at `c1eec68`
  (never the tree), run with `go test -v`, printing what each call did; or a
  break planted with `tools/plant.py` on a scratch clone, where "NOT CAUGHT"
  is the finding; or the existing tests run as they are.
* **[read]**: the code, at the line the claims table pins.

### What was measured

* **F1. A typed-nil `*StaticToken` panics.**
  `llmprovider.ResolveOptions(ProviderClaude, []Option{WithTokenSource((*StaticToken)(nil))})`
  and `providers.New(ProviderClaude, WithTokenSource((*StaticToken)(nil)))`
  each panic, "runtime error: invalid memory address or nil pointer
  dereference" **[probed]**: `sendable` reads `st.Value` from the asserted
  `*StaticToken` without a nil check (C1) **[read]**. A nil interface,
  `WithTokenSource(nil)`, returns no error and no panic **[probed]**. Other
  typed-nil sources were not probed.
* **F2. A zero `DeviceLogin`'s `Wait` panics,** the same runtime error
  **[probed]**: its `poll` field is nil (C2), and `Wait` calls it (C3)
  **[read]**. A zero `DeviceLogin`'s `Cancel` does not panic **[probed]**.
  `DeviceLogin`'s doc says Wait gives "the session"; it does not say a zero
  value is unusable.
* **F3. Like failures, unlike kinds:**
  * **F3a.** A denied sign-in: the device flow's `access_denied` (Grok) and
    Kilo's 403 match `ErrAuthFailure` (C4); the browser flow's
    `access_denied` is plain (C5). The tests 0048 left assert each:
    `TestGrokDevice_DeniedAndExpiredAreAuthFailures`,
    `TestKiloDevice_DeniedAndExpiredAreAuthFailures` and
    `TestLoginBrowserOAuth_PlainFailures_0048/IdP_access_denied` pass
    **[probed]**.
  * **F3b.** An expired device code: the service's `expired_token` matches
    `ErrAuthFailure` (C4's neighbour); the code passing its own expiry is a
    plain error (C6) **[read]**.
  * **F3c.** An unreachable Ollama, through a client that refuses the
    connection: `catalog.List(ProviderOllama, …)` gives an error matching
    `ErrProviderUnavailable` (C7); `catalog.ValidateOllamaURLWith` gives a
    plain one, "could not reach Ollama at …" (C9) **[probed]**. A status
    other than 200: the listing classifies it, as an `*APIError` (C8); the
    check gives a plain "ollama returned HTTP N" (C10) **[read]**.
  * **F3d.** A provider a call does not serve: `StartDeviceOAuth` and
    `LoginBrowserOAuth` give a plain "oauth: provider \"claude\" is not
    supported" (C11); `Registry.New` of an unknown id, `ErrInvalidProvider`;
    `catalog.List` of one, `ErrUnsupported` **[probed]**.
* **F4. The learned refusals are process-wide.** The models a provider
  learned refuse a forced tool choice are one package-level memo, at most
  256 (C12; 0034-MADR-degrade-forced-tool-choice-per-model.md D3), which no
  option replaces; R49 names it as the exception (C13). The tests of it
  each use a model name of their own (C17), as prepare-commit-msg's used a
  key per run (C16). That holds for one run only: `go test -count=2 -run
  _0034` fails 10 tests, seven in `providers/claude`
  (`TestGenerate_LearnsARefusal_0034` first, "first call sent tool_choice
  [none] … want tool, then none") and three in `providers/opencode`, on
  their second run, because the first run's refusals are still learned;
  with `-count=1` both packages pass **[probed]**. CI runs each test once,
  so it does not see this.
* **F5. Probe wiring untested in four providers.** Each of Gemini, Claude,
  Grok and Ollama passes `p.cache.ProbeStore()` to `wire.ProbeHealth`, as
  OpenAI does; only OpenAI's has a test (0048-PLAN Phase 2, "Not
  covered"). Planting `nil` in its place, in each of the four, is NOT
  CAUGHT by `go test ./llmprovider/... ./wizard/...` **[probed]**.
* **F6. Two sign-in errors untested through `ConfigureLLM`.** The browser
  and the ChatGPT plan sign-ins return their error as it is (C14, C15), as
  the device-code one does; only the device-code one has a test, so
  `ConfigureLLM`'s doc names only it (C18). Planting `fmt.Errorf("%v",
  loginErr)` at either return is NOT CAUGHT **[probed]**.

### Code claims

<!-- claims: self=c1eec68 prepare-commit-msg=87f0e64 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `sendable` reads a `*StaticToken`'s value unchecked | `llmprovider/settings.go:235` | `if st, ok := s.tokens.(*StaticToken); ok && hasControl(st.Value) {` |
| C2 | A `DeviceLogin`'s poll is a field | `llmprovider/auth/oauth_device.go:91` | `poll func(context.Context) (*OAuthSession, error)` |
| C3 | `Wait` calls it | `llmprovider/auth/oauth_device.go:156` | `session, err := d.poll(pollCtx)` |
| C4 | The device flow's denial is `ErrAuthFailure` | `llmprovider/auth/oauth_device.go:360` | `return nil, fmt.Errorf("%w: oauth: Grok device authorization denied", llmprovider.ErrAuthFailure)` |
| C5 | The browser flow's is plain | `llmprovider/auth/oauth_loopback.go:583` | `return fmt.Errorf("oauth: authorization failed: %s", code)` |
| C6 | The code's own expiry is plain | `llmprovider/auth/oauth_device.go:431` | `var errDeviceCodeExpired = errors.New("oauth: device code expired")` |
| C7 | The Ollama listing's unreachable server | `llmprovider/catalog/discovery.go:874` | `return nil, fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL, err)` |
| C8 | The listing's status, classified | `llmprovider/catalog/discovery.go:879` | `return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)` |
| C9 | The check's unreachable server, plain | `llmprovider/catalog/discovery.go:929` | `return fmt.Errorf("could not reach Ollama at %s: %w", baseURL, err)` |
| C10 | The check's status, plain | `llmprovider/catalog/discovery.go:934` | `return fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)` |
| C11 | A login's unsupported provider, plain | `llmprovider/auth/oauth_loopback.go:214` | `return oauthFlowConfig{}, fmt.Errorf("oauth: provider %q is not supported", provider)` |
| C12 | The learned refusals, a package variable | `llmprovider/internal/wire/messages/learn.go:19` | `var learned = struct {` |
| C13 | R49's exception | `docs/guides/api-standards.md:305` | `builds; without it, the process's cache is used. One memory stays` |
| C14 | The wizard's browser sign-in returns its error as it is | `wizard/auth.go:194` | `return resolvedCredential{}, loginErr` |
| C15 | The plan sign-in, likewise | `wizard/auth.go:245` | `return resolvedCredential{}, loginErr` |
| C16 | prepare-commit-msg's key per run | `prepare-commit-msg:internal/ui/setup_test.go:66` | `return fmt.Sprintf("test-key-%s-%d", t.Name(), testKeys.Add(1))` |
| C17 | A memo test's model of its own | `llmprovider/providers/claude/forced_learn_0034_test.go:78` | `p := build(t, apiKey(srv.URL, "claude-future-learns")...)` |
| C18 | `ConfigureLLM`'s doc names the device-code sign-in only | `wizard/configure.go:151` | `// errors.Is matches it; so is a failed device-code sign-in's, as` |

## Decision Drivers

* The `v1` promise (R48): no exported name changes or goes; this record
  adds none.
* An exported call fails with an error, never a panic, on a value a caller
  can build (R23: an invalid value is `ErrInvalidRequest`).
* Like failures match like kinds, so one `errors.Is` check serves every
  path a caller takes (R25).
* R49: what is kept in memory between calls is the caller's to own.
* A test is unused until it has been seen to fail on a planted break; every
  doc names only what a test shows (R52).

## Considered Options

* **A. One record, five parts:** errors for the panics, one kind for each
  pair of like failures, the learned refusals in `catalog.Cache`, the tests,
  and a release.
* **B. The panics and the tests only,** leaving the kinds and the memo.
* **C. A record for each part.**
* **D. Change nothing,** and document each finding where it is.

## Decision Outcome

Chosen option: "A", because each part is small, none adds or changes an
exported name, all were found by one record and close its gaps together, and
one release carries every changed `errors.Is` answer under one changelog
entry.

### D1. Errors in place of the two panics (F1, F2)

* `ResolveOptions` refuses a nil `*StaticToken` with an error matching
  `ErrInvalidRequest`, naming `WithTokenSource`, as it refuses a key that
  holds a control character; every `New` therefore refuses it too.
* `(*DeviceLogin).Wait` on a `DeviceLogin` that `StartDeviceOAuth` did not
  return gives an error matching `ErrInvalidRequest`, at once.
* Each doc says so, and a test shows each, red first.

### D2. One kind for like failures (F3)

Each takes its counterpart's form: the sentinel's text, then the message
as it is today, as `fmt.Errorf("%w: …")` gives it.

* **D2a.** The browser flow's `access_denied` matches `ErrAuthFailure`, as
  the device flow's does, the workspace-without-Codex case included.
  Another error the identity provider sends stays plain, as on the device
  flow.
* **D2b.** A device code that passes its own expiry matches
  `ErrAuthFailure`, as the service's `expired_token` does.
* **D2c.** `ValidateOllamaURL` and `ValidateOllamaURLWith` classify as the
  Ollama listing does: an unreachable server matches
  `ErrProviderUnavailable`, and a status other than 200 is an `*APIError`
  from `ClassifyHTTPError`.
* **D2d.** A login asked for a provider it does not serve
  (`StartDeviceOAuth`, `LoginDeviceOAuth`, `LoginBrowserOAuth`) matches
  `ErrUnsupported`, as `catalog.List` does.
* 0048's docs of those functions, and the tests that assert "plain" for
  them, change with them; the changelog lists each under Changed.

### D3. The learned refusals in `catalog.Cache` (F4)

* `catalog.Cache` holds the learned refusals beside its listings, metadata
  and probes: a provider built `WithCache(c)` learns into and reads from
  `c`; without it, the process's memo, as today. The bound, 256, is per
  memo.
* The 0034 tests each give their provider a `new(catalog.Cache)`, and
  `go test -count=2` of them passes.
* R49 loses its exception.

### D4. The tests 0048 left out (F5, F6)

* Gemini, Claude, Grok and Ollama each get the test OpenAI has: a provider
  built `WithCache(&a)` probes into `a`, a second listing through it sends
  no probe, and another cache probes again.
* `ConfigureLLM` gets a test of a failed browser sign-in and of a failed
  ChatGPT plan sign-in, each returned as the login gave it; its doc then
  names "a failed sign-in's" error, of every method.
* Each is seen to catch its planted break.

### D5. The release

* A minor release, `v1.17.0`, because `errors.Is` answers change (D2) and a
  caller may branch on them; no exported name changes, so `apidiff` against
  `v1.16.0` reports nothing. Its changelog section lists D1 under Fixed, D2
  and D3 under Changed. The owner tags it (Q1).

### Consequences

* Good, because no exported call panics on a value a caller can build
  through the API that was probed.
* Good, because one `errors.Is` check serves a denial, an expiry, an
  unreachable Ollama or an unsupported provider on every path.
* Good, because a test, or a process with two configurations, can keep its
  learned refusals apart, and the module's own tests pass at any `-count`.
* Good, because the probe and sign-in paths 0048 named as untested are
  tested.
* Bad, because a caller that relied on one of D2's errors being plain sees
  it match a sentinel; the changelog names each.
* Neutral, because a model's refusal is a fact of the model, so sharing it
  was not wrong; it is now shared only as widely as the caller's cache.

### Confirmation

* The probes of F1 and F2, run again, print errors matching
  `ErrInvalidRequest` and no panic.
* The tests 0048 left for F3, changed with D2, pass; each D2 change is seen
  to fail its test when planted back.
* `go test -count=2 -run _0034 ./llmprovider/...` passes.
* F5's and F6's six plants are caught.
* `sdkaudit -check` reports 0; `tools/apidiff.py v1.16.0` reports nothing;
  `make changelog-check` passes with the `v1.17.0` section.

## Pros and Cons of the Options

### A. One record, five parts

* Good, because it closes everything 0048 found, in one release.
* Bad, because one record carries five unrelated fixes; each is a decision
  of its own below, and the PLAN can phase them apart.

### B. The panics and the tests only

* Good, because it changes no `errors.Is` answer, so a patch release
  suffices.
* Bad, because a caller keeps needing two checks for one failure, and the
  0034 tests keep failing at `-count=2`.

### C. A record for each part

* Good, because each decision stands alone.
* Bad, because five records and five releases for small changes found
  together.

### D. Change nothing

* Good, because no caller sees a change.
* Bad, because two panics stay reachable, and R49 keeps an exception the
  module's own tests trip over.

### D1's options

* **Refuse with `ErrInvalidRequest` (chosen):** Good, because R23 says an
  invalid value is that error, from `New`.
* **Treat a typed nil as no credential:** Good, because a keyless provider
  would build. Bad, because a keyed provider then fails later and further
  from the mistake, and a nil source is not the same as none.

### D2's options

* **Align the plain paths to the sentinel ones (chosen):** Good, because
  the sentinel paths are the older contract (0028-MADR, 0034-MADR), and a
  sentinel can be matched where a message cannot.
* **Align the sentinel paths to plain:** Bad, because it takes a working
  `errors.Is` answer away from callers who use it.
* **Document the difference:** Good, because nothing changes. Bad, because
  a caller still needs to know which path it took.

### D3's options

* **The memo in `catalog.Cache` (chosen):** Good, because R49 then holds
  without exception, and `WithCache` already reaches every provider.
* **An exported reset, such as `catalog.ForgetRefusals()`:** Good, because
  it is small. Bad, because it adds an export that only tests want, and a
  reset of shared state still races between parallel tests.
* **Leave it, and give each test a model of its own:** Bad, because the
  tests already do, and still fail at `-count=2`.

## Owner questions

* **Q1. The version:** `v1.17.0` (recommended, for D2's changed `errors.Is`
  answers), or `v1.16.1`.
* **Q2. D2's reach:** all four of D2a to D2d (recommended), or a subset.
* **Q3. D3:** the memo in `catalog.Cache` (recommended), or leave it and
  record the `-count=2` limit.

The owner answered (2026-10-19): Q1 `v1.17.0`; Q2 all four; Q3 the memo in
`catalog.Cache`.

## More Information

* Found by
  [0048-PLAN-adopt-go-sdk-best-practices.md](0048-PLAN-adopt-go-sdk-best-practices.md):
  Phase 2's "Not covered" (F5), Phase 3's deviation (F1 to F3, F6), Phase
  6's R49 finding (F4).
* The memo is
  [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md)'s
  D3; the caches are
  [0048-MADR-adopt-go-sdk-best-practices.md](0048-MADR-adopt-go-sdk-best-practices.md)'s
  D1.
* The probes' scripts and outputs were scratch files, not kept; each finding
  above gives what they printed.
* The PLAN, `0049-PLAN-v1-16-0-follow-ups.md`, follows once this record is
  reviewed.
