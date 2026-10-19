---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: the owner's "Best practices for designing and building a Go SDK" (2026-10-19), which synthesizes Effective Go, Go Code Review Comments, Google's Go Style Guide and the module compatibility rules; docs/guides/api-standards.md; a source audit of b8ffd4b (method below)
informed: maintainers, agents, and the SDK's consumers (prepare-commit-msg, gobble-cli)
---

<!-- markdownlint-disable MD013 -->

# Close the SDK's Gaps Against Go SDK Practice: Injectable Caches, Runnable Examples, Stated Concurrency and Errors, a Changelog, and Reviewed Exports

## Context and Problem Statement

The owner gave a list of practices for designing and building a Go SDK, in ten
sections, and asked where this module falls short of them and what it must do
to comply. This record answers for this module. Section 9's second half, on a
self-update SDK, is go-selfupdate-lib's, and is left to a record there.

This module already has a normative standard:
[`docs/guides/api-standards.md`](../guides/api-standards.md), 48 rules, each
citing the decision that made it. AGENTS.md makes it binding: "a change that
breaks a rule is wrong". So each practice was weighed against the code and
against those rules, and falls in one of three classes: **met**, **a gap**
(the code or the docs fall short, and no rule chose that), or **a divergence**
(a rule chose otherwise, for a reason this record weighs).

### How the facts were found

Every claim is marked **[read]** (read in the source or a record at
`b8ffd4b`), **[probed]** (measured by a command or program, on a copy or the
tree, writing nothing to it) or **[derived]**.

* **The audit program.** A Go program, `sdkaudit`, parses every non-test `.go`
  file of the module with `go/ast` and reports, for the 15 packages outside
  `internal/`: exported functions and methods; those returning one of the
  module's interfaces; a `context.Context` parameter not first, or held in a
  struct; missing doc comments, and docs not starting with the name;
  `Example` functions; package-level variables by kind; `panic` calls;
  `fmt.Errorf` given an error without `%w`; exported types whose doc speaks
  of concurrency; `Deprecated:` notes; `time.Now` calls; `http` calls with no
  context; and exported functions returning `error` whose doc names an error
  or a failure.
* **The program was itself seen to fail first.** On a planted package with
  one of each defect, it counted each exactly once, nine kinds, plus one
  `Example`, one `time.Now` and the right class of each of three variables
  ("ALL OK") **[probed]**.
* **Other probes:** `go doc -short` on the three largest packages, each
  exported name searched for outside its package in this module and in both
  consumers; `go test -count=1 -cover ./...`; the GitHub releases API; the
  tags; and readings named below.

### What the audit found (b8ffd4b)

| Measure | Result |
| :--- | :--- |
| Public packages | 15 |
| Exported functions and methods | 199 |
| Exported types | 82 |
| A doc comment missing, or not starting with the name | 0 and 0 |
| `context.Context` not the first parameter | 0 |
| `context.Context` held in a struct | 1, internal (`transport.ReplyReader`) |
| `http` calls with no context | 0 |
| `fmt.Errorf` given an error without `%w` | 0 |
| Exported package-level variables | 10, all error sentinels |
| `panic` calls | 6 |
| Returning one of the module's interfaces | 13: every provider `New`, `providers.New`, `WithRetry`, `WithRetryNotify` |
| `Example` functions | **0** |
| `Deprecated:` notes | 0 |
| Exported types whose doc speaks of concurrency | **6 of 82** |
| Exported functions returning `error` whose doc names an error or a failure | **27 of 58** |
| Coverage of the public packages | 82.2 % (`llmtest`) to 97.8 % (`llmprovider`) |
| GitHub Releases | **0**, for 23 `v1.*` tags |
| `CHANGELOG` | **none** |

All **[probed]**.

### Met

1. **Context first, honoured, never stored.** No exported function takes a
   context other than first, and no `http` call is made without one
   **[probed]**. The one stored context is internal: `ReplyReader`, an
   `io.Reader` that lives for one request and cancels it on an idle body, as
   `http.Request` carries its own **[read]**. R40 is the rule.
2. **No exported mutable state.** All 10 exported package-level variables are
   error sentinels **[probed]**, as R29 requires (C14).
3. **Errors.** `*APIError` and the sentinel kinds, tested with `errors.Is`
   (R24, R25); no `fmt.Errorf` drops a wrapped error **[probed]**; messages
   redacted and bounded (R35).
4. **Panics only on programmer error.** The six: `catalog.defaultConfig`
   (no options to refuse); `llmtest.Fake.Reply(nil)` (a script mistake, its
   doc says so); two in `opencode`'s embedded route snapshot (a build
   defect, a test pins it); `providers.Default` (a duplicate built-in); and
   `redact`'s anchor bound, whose anchors come from a fixed table indexed at
   package init (C16), and a test plants the overflow **[read]**.
5. **Every export documented, starting with its name** **[probed]**, which
   `revive` enforces (R41).
6. **Options.** Functional options with sensible defaults; the order rule is
   stated: "Options apply in order, so a later one wins" (C9); a foreign
   option is an error (R18) **[read]**.
7. **Versioning.** 23 semantic-version tags; `apidiff` against the latest
   `v1.*` tag fails an incompatible change (R48); the deprecation rule is
   R42 **[read]**. `tools/apidiff.py v1.15.0` finds no change since the last
   tag **[probed]**.
8. **Dependencies.** The standard library, `golang.org/x/term` and its
   `golang.org/x/sys`, held by `make dep-check` (R2) **[read]**.
9. **Concurrency of providers.** A provider is immutable and safe for
   concurrent use after `New` (R20, C15); every provider runs `llmtest`'s
   concurrent-use check under `-race` (C17). No cache holds its lock across
   a fetch: each unlocks, fetches, and locks again (C6, C7) **[read]**.
10. **Security of credentials.** Credentials are `TokenSource`s (R15);
    secret-bearing values redact themselves (R33); library code reads no
    environment variable except through `FromEnv` helpers (R30); logging goes
    only to a given `*slog.Logger` (R31); an API key is one option, the OAuth
    browser and device flows and the ChatGPT plan sign-in are one function
    each, and a vendor CLI's login is a `TokenSource` (`VendorCLISession`)
    **[read]**.
11. **The publishing checklist.** gofmt, vet, tests, lint and govulncheck in
    CI on three systems; a README with a quick start (C11) **[read]**.
12. **Testability.** `WithHTTPClient`, any `TokenSource`, `llmtest.Fake` for
    consumers, and coverage of 82–98 % of the public packages **[probed]**.

### Gaps

* **G1. Process-wide caches no caller can inject or reset.** The listing
  cache (C1), the model-metadata cache (C2) and the probe cache (C3) are
  package-level maps under package-level locks. No option gives a provider
  or a listing its own, and none clears them **[read]**. A consumer pays:
  prepare-commit-msg's tests had to give every run of every test its own key,
  because a repeated run read the listing the first run cached (C18). This
  is the practice "No global mutable state. Prefer explicit dependency
  injection", unmet; R29 forbids only *exported* mutable state, so no rule
  chose it.
* **G2. No runnable examples, and no happy path where pkg.go.dev shows it.**
  0 `Example` functions **[probed]**. The package comments of `llmprovider`
  (7 lines), `providers` (5), `wizard` (10) and `llmtest` (6) hold no code,
  nor do `auth`'s and `catalog`'s (26 each) **[probed]**. The README's quick
  start (C11) is the only one.
* **G3. Concurrency contracts unstated.** 6 of 82 exported types say whether
  they are safe for concurrent use **[probed]**. Two of the silent ones are
  interfaces a caller implements and a concurrent provider calls:
  `TokenSource` (C4) and `TokenStore` (C5); also `FileTokenStore` (C10),
  `catalog.Catalog` (C19), `StaticToken`, `VendorCLISession` and
  `wizard.TextPrompter` **[read]**. A caller writing a `TokenSource` cannot
  learn from its doc that `Token` is called from many goroutines **[derived]**.
* **G4. Errors undocumented per function.** 31 of 58 exported functions and
  methods returning `error` name no error and no failure in their doc
  **[probed]**: every provider's `New` (though `New` does fail early with
  `ErrInvalidRequest` naming the fix, C12), `GenerateText`,
  `LoginBrowserOAuth`, `LoginDeviceOAuth`, `StartDeviceOAuth`,
  `NewFileTokenStore`, `wizard.ConfigureLLM` and the `TextPrompter` methods
  among them **[probed]**. Some of the 31 are `MarshalJSON` methods, whose
  contract is the standard library's.
* **G5. No CHANGELOG and no GitHub Releases.** The history is the README's
  Status section and a "release notes" step in 17 PLANs (for example
  `v1.15.0`'s in 0043-PLAN) **[probed]**. A consumer reading the repository's
  Releases page sees none of 23 tags.
* **G6. A Go floor of the newest patch, tested on one version.** `go.mod`
  says `go 1.27.2` (C8), which makes 1.27.2 the least Go any consumer may
  build with; CI reads the same file and tests that one version (C13)
  **[read]**. Whether the module builds with an older supported Go has not
  been measured.

  The two supported majors' newest patches, from Go's release history
  ([go.dev/doc/devel/release](https://go.dev/doc/devel/release)) **[read]**: go1.26.9 and go1.27.2, both
  of 2026-10-08, each "includes security fixes to the `go` command, and the
  `crypto/tls`, `html/template`, `net/http`, `net/textproto`, and `os`
  packages"; go1.26.8, of 2026-09-01, lists none. Every provider here speaks
  `net/http` over `crypto/tls` **[derived]**.
* **G7. Exports no caller names.** Of the exported names that `go doc -short`
  lists, 2 of 90 in `llmprovider`, 17 of 49 in `catalog` and 2 of 20 in `auth`
  are referenced by name nowhere outside their package, in this module or
  either consumer **[probed]**. Many are result or field types a caller
  receives without naming (`catalog.MenuEntry`, `catalog.Caveat`), so this
  does not show them needless; and under R48 none can be removed in `v1`.
  It shows that nothing reviews a new export before it is promised.

### Divergences, chosen by this module's rules

* **X1. Constructors return an interface.** "Accept interfaces, return
  concrete types" is not followed by `New` (R14, C20) or the middleware
  (R28, C21). The SDK's purpose is one contract over many providers: a
  `Registry` holds factories of one signature, and a wrapper returns what it
  wraps. The standard library does the same where the abstraction is the
  product (`sha256.New` returns `hash.Hash`) **[derived]**.
* **X2. A required credential is an option.** "Required values go as regular
  parameters" is not followed: `WithAPIKey` or `WithTokenSource` (R15). One
  constructor shape serves a keyless provider (Ollama) and every kind of
  source, and `New` refuses a missing credential at once (C12) **[read]**.
* **X3. Configuration structs are unexported** (R19, C22). The practice
  prefers an exported config struct when configuration is serialized; this
  module serializes none of its own. The consumers own their configuration,
  and `wizard.Result` is exported for them to keep **[read]**.

## Decision Drivers

* The `v1` compatibility promise (R48): every change here is additive.
* A practice is adopted unless a rule chose otherwise for a reason that still
  holds; then the reason is written down.
* A rule is held by a check where one can be written, as R41 is by `revive`.
* What a consumer meets first: pkg.go.dev, the Releases page, a type's doc.

## Considered Options

* **A. Close G1–G7 with additive changes, make each a rule with a check
  where it can be, and record X1–X3's reasons in the standard.**
* **B. Documentation only:** G2–G5, leaving the caches and the Go floor.
* **C. A `v2` that also reverses X1–X3.**
* **D. Change nothing.**

## Decision Outcome

Chosen option: "A", because each gap can close without breaking a `v1`
caller, the divergences are deliberate and need only their reasons stated,
and a rule with a check keeps the module from drifting back.

### D1. Caches a caller can own (G1)

* A `catalog.Cache` type holds what the three caches hold today; its zero
  value is ready to use and safe for concurrent use.
* An option gives a provider and a listing their own:
  `catalog.WithCache(*Cache)`, accepted by `catalog.List`, `catalog.ListRaw`
  and every provider's `New` (scoped as R17 says).
* Without it, the process-wide cache is used, as today, so no caller's
  behaviour changes.
* A `Registry`'s providers can share one: an additive
  `providers.DefaultWith(opts ...llmprovider.Option)` passes the options,
  `WithCache` among them, to every built-in factory. `Default()` keeps its
  signature, which R48 holds (Q2).
* A test gives each case a `new(catalog.Cache)`; nothing needs a per-run key.

### D2. Examples and a happy path on pkg.go.dev (G2)

* Each public package has at least one `Example` function that runs offline
  (`llmtest.Fake`, `httptest`) and has an `// Output:` check, so `go test`
  runs it.
* The package comments of `llmprovider`, `providers` and `wizard` show the
  simplest call, as the README's quick start does.

### D3. Stated concurrency (G3)

* Every exported type's doc says whether it is safe for concurrent use; an
  interface a caller implements says what it must be. `TokenSource`:
  "`Token` may be called from many goroutines at once". `TokenStore`:
  likewise, and whether `FileTokenStore` serializes within a process.

### D4. Stated errors (G4)

* Every exported function or method returning `error` names in its doc the
  kinds it can return (by sentinel, or `*APIError`), or says when it fails;
  a `MarshalJSON`, `UnmarshalJSON` or other method of a standard interface is
  exempt.

### D5. A changelog and Releases (G5)

* `CHANGELOG.md`, in the [Keep a Changelog](https://keepachangelog.com/)
  form, from `v1.0.0`, built once from the README's Status section and the
  PLANs' release notes, then kept with each release.
* Each tag gets a GitHub Release whose body is its changelog section, made
  by a CI job on a `v*` tag; a gate fails the tag when the section is missing
  (Q3). The 23 past tags get theirs once, from the built changelog.
  *(A2, 2026-10-19: only the six tags before `v1.16.0` get one.)*
* The README's Status keeps the current release, and links the changelog.

### D6. ~~A Go 1.26.9 floor, this repository on the newest~~ The Go floor stays `go 1.27.2`, with its reasons (G6; A1)

* *(A1, 2026-10-19: the floor stays `go 1.27.2`, CI its one version, and no
  `toolchain` line. `api-standards.md` states the two reasons: the module
  uses `encoding/json/v2`, standard library only from Go 1.27.0, and 1.27.2
  carries the 2026-10-08 security fixes. The bullets below are struck.)*
* ~~`go.mod` says `go 1.26.9`: the older supported major's newest patch, the
  widest floor that still carries the 2026-10-08 security fixes to
  `net/http`, `crypto/tls`, `net/textproto` and `os` (Q1).~~
* ~~`go.mod` also says `toolchain go1.27.2`, so this repository's own builds,
  tests and CI use the newest patch; a consumer needs only the floor.~~
* ~~CI tests both: the floor job with Go 1.26.9 and `GOTOOLCHAIN=local`, so the
  toolchain line does not raise it, and the newest with 1.27.2.~~
* ~~That the module builds and passes its tests on Go 1.26.9 is measured
  first, on a scratch clone; a failure stops the work for the owner.~~
* ~~AGENTS.md's "Requires Go 1.27.2", the README's Status and
  `api-standards.md` follow.~~

### D7. A review of new exports (G7)

* A PLAN that adds an exported name says, for each, the call site that needs
  it. `make api-check` already lists every addition; the PLAN's record holds
  the justification.

### D8. The standard and its checks

* `api-standards.md` gains a rule for each of D1–D4, D6 and D7, and its
  Construction section states X1–X3's reasons.
* **The audit program is kept (Q4)** as `llmprovider/internal/cmd/sdkaudit`,
  beside the maintainer command `internal/cmd/catalogs`: a package of the
  module, with its R2 table row, tests at R47's 80 % with a planted input,
  vet and lint; run with `go run`. It is Go, where AGENTS.md asks for Python
  tools, because `go/ast` parses Go reliably and a regex does not; AGENTS.md
  and `tools/README.md` name it.
* A check holds each rule it can: `sdkaudit`, run as a gate, for the examples,
  the concurrency docs and the error docs; a gate that the latest tag has a
  changelog section.

### Consequences

* Good, because a test or a long-running process can own its cache, and a
  consumer's tests stop needing per-run keys.
* Good, because pkg.go.dev shows working code for every public package, and
  the Releases page shows every release.
* Good, because a caller implementing `TokenSource` or `TokenStore` learns its
  contract from the doc.
* Neutral, because X1–X3 stay as they are, now with written reasons.
* ~~Good, because a consumer on Go 1.26.9 can build the SDK, and still has the
  October security fixes.~~ *(A1)*
* Bad, because there is more API (`catalog.Cache`, `WithCache`,
  `providers.DefaultWith`), more gates, ~~a second Go version in CI,~~ and a
  Go maintainer command to keep *(A1)*.

### Confirmation

* `sdkaudit` on the result: `Example` functions in
  every public package; every exported type's doc states its concurrency;
  every error-returning export's doc names its errors, but the exempt ones.
* `tools/apidiff.py` shows only additions.
* prepare-commit-msg's listing tests pass with a `catalog.Cache` per test and
  no per-run key.
* A tag without a changelog section fails CI, and a tag with one gets its
  Release.
* ~~CI green on Go 1.26.9 (`GOTOOLCHAIN=local`) and on 1.27.2.~~ *(A1)*

## Pros and Cons of the Options

### A. Close the gaps, additively, with rules and checks

* Good, because no `v1` caller breaks.
* Good, because the checks keep the module where it lands.
* Bad, because it is the most work of the options.

### B. Documentation only

* Good, because it is cheap and breaks nothing.
* Bad, because G1 is a design gap a consumer has already paid for (C18), and
  G6 limits every consumer's Go.

### C. A `v2` that also reverses X1–X3

* Good, because it would follow every practice to the letter.
* Bad, because every consumer would migrate, for divergences whose reasons
  hold.

### D. Change nothing

* Good, because nothing changes.
* Bad, because the gaps stay, and new code repeats them.

## Owner questions

Answered 2026-10-19; each answer is in the decisions above.

* **Q1. G6's floor.** **The owner: `go 1.26.9`, with `toolchain go1.27.2`**
  (D6). The owner first kept `go 1.27.2` for its security fixes, then chose
  `go 1.26.8`; Go's release history showed that 1.26.8 lacks the 2026-10-08
  security fixes to `net/http` and `crypto/tls`, which 1.26.9 carries, and
  the owner took 1.26.9. *Then, with A1's fact, the owner kept `go 1.27.2`.*
* **Q2. D1's shape.** **The owner: one `catalog.Cache`, and the default
  registry can take it.** Since `Default()`'s signature is under R48, that is
  an additive `providers.DefaultWith` (D1).
* **Q3. D5's Releases.** **The owner: made by CI on each tag,** from the
  changelog, with a gate for the section (D5).
* **Q4. The audit program.** **The owner: keep it as a Go tool,** at
  `llmprovider/internal/cmd/sdkaudit`, beside `internal/cmd/catalogs`, rather
  than under `tools/`, where a Go package would join the module without the
  precedent (D8).

## More Information

<!-- claims: self=b8ffd4b prepare-commit-msg=60f1de1 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The listing cache is a package-level map | `llmprovider/catalog/listing_cache.go:65` | `listingCache = map[listingKey]listingEntry{}` |
| C2 | The model-metadata cache is a package-level map | `llmprovider/catalog/model_metadata.go:400` | `modelMetadataCache = map[string]modelMetadataCacheEntry{}` |
| C3 | The probe cache is a package-level map | `llmprovider/internal/transport/probe.go:118` | `probeCache = map[ProbeKey]probeCacheEntry{}` |
| C4 | `TokenSource`'s declaration | `llmprovider/token.go:92` | `type TokenSource interface {` |
| C5 | `TokenStore`'s declaration | `llmprovider/auth/tokenstore.go:169` | `type TokenStore interface {` |
| C6 | The listing cache unlocks before it fetches | `llmprovider/catalog/listing_cache.go:184` | `listingMu.Unlock()` |
| C7 | … and fetches after | `llmprovider/catalog/listing_cache.go:195` | `fetchListingModel(fetchCtx, id, src, model, cfg)` |
| C8 | The Go floor | `go.mod:3` | `go 1.27.2` |
| C9 | The option order rule | `llmprovider/settings.go:17` | `Options apply in order, so a later one wins;` |
| C10 | `FileTokenStore`'s declaration | `llmprovider/auth/tokenstore_file.go:31` | `type FileTokenStore struct` |
| C11 | The README's quick start | `README.md:37` | `## Quick start` |
| C12 | `New` refuses a missing credential | `llmprovider/providers/claude/claude.go:159` | `claude needs WithAPIKey or WithTokenSource` |
| C13 | CI's one Go version, from `go.mod` | `.github/workflows/ci.yml:36` | `go-version-file: go.mod` |
| C14 | R29 | `docs/guides/api-standards.md:203` | `**R29. Nothing mutable is exported at package level.**` |
| C15 | R20 | `docs/guides/api-standards.md:161` | `` **R20. After `New`, a provider is immutable and safe for concurrent use.** `` |
| C16 | `redact`'s anchors come from a fixed table, at init | `internal/redact/redact.go:459` | `var anchors, passAnchors = indexAnchors(passes)` |
| C17 | `llmtest` checks concurrent use under `-race` | `llmprovider/llmtest/llmtest.go:132` | `concurrent use (R20), which the race detector checks;` |
| C18 | prepare-commit-msg's tests needed a key per run | `prepare-commit-msg:docs/decisions/0015-PLAN-adopt-go-llmprovider-sdk-v1-15-0.md:262` | `### Deviation 2 (2026-10-19): a key per run, not per test name` |
| C19 | `catalog.Catalog`'s declaration | `llmprovider/catalog/discovery.go:117` | `type Catalog struct` |
| C20 | R14's constructor returns `Provider` | `docs/guides/api-standards.md:134` | `New(opts ...llmprovider.Option) (llmprovider.Provider, error)` |
| C21 | `WithRetry` returns `Provider` | `llmprovider/retry.go:86` | `func WithRetry(p Provider, policy RetryPolicy) Provider {` |
| C22 | R19 | `docs/guides/api-standards.md:159` | `**R19. Configuration structs are unexported.**` |

* The audit program and its planted check were run from a scratch directory,
  not committed; Q4 asks whether to keep them.
* Related: [0015-MADR](0015-MADR-canonical-sdk-api-and-module-layout.md)
  (the API's shape, X1–X3), [0035-MADR](0035-MADR-maintainer-tools-in-python.md)
  and [0047-MADR](0047-MADR-keep-five-more-scratch-helpers-as-tools.md) (the
  tools), [0041-MADR](0041-MADR-decide-per-model-behaviour-from-metadata.md)
  (the listing cache, D6).

### A1 (2026-10-19): the floor stays `go 1.27.2`, because the module needs Go 1.27

* **Found** while grounding 0048-PLAN: the module imports
  `encoding/json/v2` and `encoding/json/jsontext` (C23, C24), which Go's
  release notes list among Go 1.27's new packages (go.dev/doc/go1.27)
  **[read]**; before 1.27 they were behind the `jsonv2` experiment. So a
  `go 1.26.9` floor cannot build `catalog` or `internal/wire/messages`
  **[derived]**. No other Go 1.27 API, and no generic method, is used: each
  of the release notes' new names was searched for in the module's Go files
  **[probed]**.
* **The uses are decisions:** 0028-MADR's amendments of 2026-10-08 (decoding
  models.dev with json/v2 took 4.71 MB against 19.24 MB, 75 % less) and of
  2026-10-09 (the Messages wire's typed encoder, 13.7 µs and 4 allocations
  against 54.3 µs and 1,008) **[read]**.
* **Weighed:** a build-tag split keeping both encoders, or dropping json/v2.
  Go 1.26 leaves support when Go 1.28 ships, due about February 2027 on Go's
  six-month cadence **[derived]**, and every consumer is on 1.27.2 already
  **[read]**.
* **The owner chose (2026-10-19):** keep `go 1.27.2`. D6 is struck and
  rewritten; the title drops "a Go 1.26.9 Floor"; G6 is closed by the stated
  reasons, not by a lower floor.

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C23 | `catalog` imports json/v2 | `llmprovider/catalog/model_metadata.go:7` | `jsonv2 "encoding/json/v2"` |
| C24 | The Messages wire imports jsontext | `llmprovider/internal/wire/messages/messages.go:8` | `"encoding/json/jsontext"` |

### A2 (2026-10-19): Releases for the six tags before `v1.16.0`, not all 23

* **The owner chose (2026-10-19),** at 0048-PLAN Phase 7 step 5, with each
  of the 23 notes written and scanned: a GitHub Release for the six tags
  before `v1.16.0` only, `v1.10.2`, `v1.11.0`, `v1.12.0`, `v1.13.0`,
  `v1.14.0` and `v1.15.0`. D5's "The 23 past tags get theirs once" is
  narrowed to those six.
* **The other 17,** `v1.0.0` to `v1.10.1`, get no Release. Their sections
  stay in `CHANGELOG.md`, which `make changelog-check` still holds to every
  tag, and the tags stay as they are; a Release can be made for any of them
  later from its section, as the six were.
