---
status: complete
date: 2026-10-20
associated-madr: "0050-MADR-refuse-typed-nil-token-sources.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Refusing a Typed-Nil Token Source Where It Enters

Associated MADR: [0050-MADR-refuse-typed-nil-token-sources.md](0050-MADR-refuse-typed-nil-token-sources.md)

## Goal

The MADR's D1 to D6, as the owner answered its questions (Q2: a nil
interface in `List` and `ListRaw` matches `ErrInvalidRequest` too; Q1:
`v1.18.0`):

* `internal/nilsource.Is` decides a typed nil, for every nilable kind;
* `ResolveOptions` refuses a typed-nil source, so every `New`, and every
  catalog call's options, do;
* `List` and `ListRaw` refuse a typed-nil source and a nil interface alike,
  with `ErrInvalidRequest`; `LookupModel` reports a typed nil and merges the
  other layers; `CachedModel` treats it as no source;
* the MADR's probe, run again, prints no panic;
* `v1.18.0` is released, with no exported name added or changed.

## Scope

### In scope

* A new internal package, `llmprovider/internal/nilsource`, with its tests.
* `llmprovider/settings.go` (`sendable`, `WithTokenSource`'s and
  `ResolveOptions`'s docs), and its tests.
* `llmprovider/catalog`: `listingToken` (`discovery.go`), `LookupModel` and
  `CachedModel` (`lookup.go`), the docs of `List`, `ListRaw`, `LookupModel`
  and `CachedModel`, and their tests.
* Tests beside each built-in provider's `New`, in one table, in
  `llmprovider/providers`.
* R2's table in `docs/guides/api-standards.md`, `docs/architecture.md`,
  `CHANGELOG.md`, the README's Status.

### Out of scope

* Nil-safe methods on the module's sources (the MADR's option B).
* Any exported name added, changed or removed (R48); `apidiff` against
  `v1.17.0` reports nothing.

## Rules for every phase

* **Approval.** The owner accepts the MADR and approves this PLAN, whole or
  by phase; a phase starts after the owner has committed the last.
* **The gate, before staging:** `python3 -B tools/gate.py`; `make
  pre-add-check FILES="…"` on the phase's Go files; `tools/claims.py` on
  both records; `make records-check`; `tools/recordlint.py --changed`;
  `tools/denyscan.py --changed`. Each exit status is captured before any
  filter, and read before anything is written from it.
* **Red first.** Every new test is seen to fail on a planted break, by
  `tools/plant.py` on a scratch clone, never in the tree; the phase's
  record holds each plant and its failure line.
* **The docs.** A function whose errors change has its doc changed in the
  same phase, naming only what a test shows (R52); `make docs-check` stays
  at 0.
* **The API and the wire.** `tools/apidiff.py v1.17.0` reports no change;
  G-wire's goldens do not change (R45); `make coverage-check` holds every
  floor (R47), the new package's at 80 %.
* **Stage, then the owner commits** with `git commit --no-edit`; push and
  tags on the owner's word; the disclosure guard runs before the owner
  pushes.
* **Stop conditions:** a fact of the MADR or a claim below no longer holds;
  a step would add or change an exported name; a golden changes; a doc
  would state a behaviour no test shows. Stop, record a dated deviation
  here (and an amendment in the MADR when a decision or a fact changes),
  and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.

### Phase 1: the test, and `ResolveOptions` (D1 to D3)

1. **`llmprovider/internal/nilsource`** (D1, D2): `doc.go`'s package
   comment, and `Is(v any) bool`: false for a nil interface; for a value of
   kind `Pointer`, `Map`, `Slice`, `Func`, `Chan` or `Interface`,
   `reflect.ValueOf(v).IsNil()`; false for any other kind. The standard
   library alone.
2. **Its tests,** `nilsource_0050_test.go`: a nil interface; a nil and a
   non-nil value of each of the six kinds, behind an `any`; an `int`, a
   string and a struct, never nil. Coverage 100 %.
3. **`settings.sendable`** (C1): `if nilsource.Is(s.tokens)` refuses with
   `fmt.Errorf("%w: WithTokenSource: a nil %T", ErrInvalidRequest,
   s.tokens)`, before any check that reads the source; 0049's `*StaticToken`
   nil check (C1) goes, its case covered; the control-character check of a
   `*StaticToken`'s value stays.
4. **Docs:** `WithTokenSource` (C2): a nil source is none; a typed-nil one
   is refused by every `New`, with an error matching `ErrInvalidRequest`.
   `ResolveOptions` (C3): "a typed-nil TokenSource given to
   `WithTokenSource`" in place of "a nil *StaticToken given to
   WithTokenSource".
5. **Tests** (`_0050$`):
   * in `llmprovider`: `ResolveOptions` with a typed-nil `*StaticToken`,
     `*CommandToken`, and a caller's own source of a func type, each
     `ErrInvalidRequest`, naming the type, and no panic; a nil interface,
     no source and no error;
   * in `llmprovider/providers`: a table of every built-in provider's `New`
     (through `providers.New`) with each typed-nil source,
     `*llmprovider.StaticToken`, `*llmprovider.CommandToken`,
     `*auth.OAuthSession`, `*auth.VendorCLISession` and the func type, each
     `ErrInvalidRequest`, none a panic.

   0049's `TestResolveOptions_NilStaticToken_0049` keeps passing: it asserts
   the kind alone.
6. **Tables:** R2's table gains `llmprovider/internal/nilsource` ("a
   typed-nil `TokenSource`'s test, for `llmprovider` and `catalog`" | "the
   standard library"), and `llmprovider`'s row (C4) adds it to what it may
   import; `docs/architecture.md`'s tree and table likewise, and
   `llmprovider`'s row (C6). `make tables-check`.
7. **Red first:** `Is` that ignores one kind (`Func`), and one that answers
   false for a nil pointer; `sendable` without its check; each caught by its
   test, the provider table by a recovered panic reported as a failure.

### Phase 2: the catalog (D4)

1. **`listingToken`** (C7, C8): a nil interface and a typed nil both refused
   with errors matching `llmprovider.ErrInvalidRequest`: the first keeps
   its message after the sentinel's text, `"%w: model listing: TokenSource
   is required"`; the second, `"%w: model listing: a nil %T is not a
   TokenSource"`. `List` and `ListRaw` return each, as today they return
   the plain one.
2. **`LookupModel`** (C9): a typed-nil source gives no listing layer, and
   appends `fmt.Errorf("model metadata: listing %s: %w: a nil %T is not a
   TokenSource", id, llmprovider.ErrInvalidRequest, src)` to its joined
   errors; the caller's layers and the document are merged as today. A nil
   interface, as today: no listing layer, no error.
3. **`CachedModel`** (C10): a typed-nil source is no source: no listing
   read, no fetch started.
4. **Docs:** `List`'s (C11) names a missing or typed-nil source as an error
   matching `llmprovider.ErrInvalidRequest`; `ListRaw`'s (C12) likewise;
   `LookupModel`'s (C13) that a typed-nil source is reported as
   `ErrInvalidRequest` beside the other layers; `CachedModel`'s (C14) that
   it is no source.
5. **Tests** (`_0050$`, in `catalog_test`): `List` and `ListRaw`, for a nil
   interface and for each typed-nil source, `ErrInvalidRequest` and no
   panic; `LookupModel` with a typed-nil source and a caller's layer that
   knows the model: the layer's fields, and an error matching
   `ErrInvalidRequest`; with a nil interface: the layer's fields and no
   error; `CachedModel` with a typed-nil source and that layer: the layer's
   fields, known, no panic.
6. **The MADR's probe,** run again unchanged on a scratch copy: no panic in
   any of its 30 pairs, nor in the catalog probe; each line recorded.
7. **Red first:** each catalog check removed in turn, and the nil
   interface's error left plain; each caught.
8. **`CHANGELOG.md`,** `[Unreleased]`: under Fixed, a typed-nil token source
   is refused with `ErrInvalidRequest` by every `New`, `List`, `ListRaw`
   and `LookupModel` (D3, D4), where it panicked; under Changed, a nil
   source in `List` and `ListRaw` matches `ErrInvalidRequest` (Q2).

### Phase 3: release `v1.18.0` (D6)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.18.0] - <date>`, the date
   that of the commit the tag will name; an empty `[Unreleased]` above it;
   the comparison links. The README's Status names `v1.18.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.17.0`, no change;
   the disclosure guard over the outgoing commits, run by the agent before
   the owner pushes.
3. **The owner tags `v1.18.0`** and pushes it; CI's `release` job makes the
   Release; its body equals `check_changelog.py --section v1.18.0 --repo
   maccavelli/go-llmprovider-sdk`.
4. **After the tag:** `tools/consumer.py v1.18.0`; `tools/offline.py
   v1.18.0`.
5. **Close:** the MADR's Confirmation items, each with its evidence; this
   PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `nilsource.Is` answers each nilable kind, nil and not | 1 | its tests, 100 % |
| V2 | Every built-in `New` refuses each typed-nil source with `ErrInvalidRequest`, never a panic | 1 | the provider table |
| V3 | `List` and `ListRaw` refuse a nil and a typed-nil source with `ErrInvalidRequest` | 2 | the catalog tests |
| V4 | `LookupModel` reports a typed nil and merges the rest; `CachedModel` treats it as none | 2 | the catalog tests |
| V5 | The MADR's probe prints no panic | 2 | the probe, run again |
| V6 | No exported name changes; the docs name what the tests show | all | `tools/apidiff.py v1.17.0`; `make docs-check` |
| V7 | `v1.18.0`'s Release body is its section | 3 | `gh release view`; `--section` |
| V8 | Every new test failed first | 1, 2 | the execution record |

## Rollout and Rollback

* **Rollout.** Three phase commits on `main`, staged by the agent and
  committed by the owner; then the owner's tag.
* **Rollback.** Each phase reverts on its own, newest first. Phase 2's
  nil-interface kind is the one change a caller sees besides an error in
  place of a panic; reverting Phase 2 restores the plain error and the
  panics there.

## Code claims

<!-- claims: self=0fbaf90 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | 0049's check, by type | `llmprovider/settings.go:239` | `if st == nil {` |
| C2 | `WithTokenSource`'s doc | `llmprovider/settings.go:109` | `// WithTokenSource sets where a provider gets its credential (0015-MADR D5,` |
| C3 | `ResolveOptions`'s doc names 0049's case | `llmprovider/settings.go:173` | `// scoped to another provider that no For scopes away, a malformed For, and a` |
| C4 | R2's `llmprovider` row | `docs/guides/api-standards.md:33` | `` the standard library, `internal/redact`, `internal/transport` `` |
| C5 | R2's `catalog` row | `docs/guides/api-standards.md:35` | `` `internal/chatgpt`, `internal/cacheref` `` |
| C6 | The architecture's `llmprovider` row | `docs/architecture.md:96` | `` `StaticToken`, `CommandToken`) `` |
| C7 | `listingToken` refuses a nil interface | `llmprovider/catalog/discovery.go:242` | `if src == nil {` |
| C8 | With a plain error | `llmprovider/catalog/discovery.go:243` | `return llmprovider.Token{}, errors.New("model listing: TokenSource is required")` |
| C9 | `LookupModel`'s listing layer | `llmprovider/catalog/lookup.go:84` | `if src != nil && listingLayers[id] {` |
| C10 | `CachedModel`'s | `llmprovider/catalog/lookup.go:112` | `if src != nil && listingLayers[id] {` |
| C11 | `List`'s doc of its errors | `llmprovider/catalog/discovery.go:196` | `// stale. Only Ollama, an unknown provider, a missing source, a token` |
| C12 | `ListRaw`'s doc | `llmprovider/catalog/raw.go:54` | `// ListRaw lists id's models exactly as the provider sends them (0041-MADR` |
| C13 | `LookupModel`'s doc | `llmprovider/catalog/lookup.go:53` | `// LookupModel returns what is known of model, merged field by field from its` |
| C14 | `CachedModel`'s doc | `llmprovider/catalog/lookup.go:93` | `// CachedModel is LookupModel without the wait: what the caches and the` |

## Execution record

### Phase 0: records (2026-10-19)

* **No deviation.** The owner committed the MADR and this PLAN as
  `b92cf29`, with the MADR's questions answered (Q2 align a nil interface
  in `List` and `ListRaw` to `ErrInvalidRequest`; Q1 `v1.18.0`), and said
  "committed, proceed".
* `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`,
  both index rows with them; "records-check: 0 problem(s)".

### Phase 1: the test, and `ResolveOptions` (2026-10-19)

* **No deviation.** Phase 0 was committed by the owner as `9e46609`. Step
  1's package comment is in `nilsource.go`, its one source file, as
  `internal/listlimit` keeps its own, not in a separate `doc.go`.
* **Steps 1 and 2,** `llmprovider/internal/nilsource`: `Is(v any) bool`,
  false for a nil interface, `reflect.ValueOf(v).IsNil()` for the six
  nilable kinds, false otherwise, calling no method of `v`; the standard
  library alone. `TestIs_0050`: a nil interface, a nil and a non-nil value
  of each kind behind an `any`, and an `int`, a string and a struct; 100 %.
* **Step 3,** `settings.sendable`: `if nilsource.Is(s.tokens)` refuses with
  "%w: WithTokenSource: a nil %T" and `ErrInvalidRequest`, before the
  control-character check of a `*StaticToken`'s value, which stays. 0049's
  `*StaticToken` nil check is gone, its case covered.
* **Step 4, the docs:** `WithTokenSource` (a nil source is none; a typed-nil
  one is refused by every `New`) and `ResolveOptions` ("a typed-nil
  TokenSource given to WithTokenSource").
* **Step 5, the tests,** under `-race`:
  `TestResolveOptions_TypedNilSources_0050` (a typed-nil `*StaticToken`,
  `*CommandToken` and a caller's own func type, each `ErrInvalidRequest`
  naming its type; a nil interface, no source and no error) and
  `TestNew_TypedNilSources_0050` (every built-in provider's `New` with each
  of five typed-nil sources, `*StaticToken`, `*CommandToken`,
  `*auth.OAuthSession`, `*auth.VendorCLISession` and the func type: 50
  calls, each `ErrInvalidRequest`, none a panic). 0049's
  `TestResolveOptions_NilStaticToken_0049` passes: it asserts the kind
  alone.
* **Step 6, the tables:** R2's `nilsource` row and `llmprovider`'s import,
  and its citation; `docs/architecture.md`'s tree, row and `llmprovider`'s
  imports. `make tables-check`: "38 packages, 0 problem(s)".
* **Step 7, red first,** three plants, all caught:
  * `Is` ignoring a nil func: "typed_nil_0050_test.go:35: a nil
    llmprovider_test.tokenFunc0050: ResolveOptions = <nil>; want
    ErrInvalidRequest naming its type", and the provider table at ":44";
  * `Is` answering false for a nil pointer: "nil_token_0049_test.go:19:
    ResolveOptions panicked: runtime error: invalid memory address or nil
    pointer dereference; want an error": the control-character check reads
    the value, so the refusal must come first, as it does;
  * `sendable` without its check: the same panics, reported as failures.
* **The checks:** `make docs-check`, 0; `tools/apidiff.py v1.17.0`, no output,
  exit 0; `go test -count=1 ./...`, ok; `make pre-add-check` on the five Go
  files, "5 file(s) clean"; `tools/gate.py` on the final tree, 27 of 27
  passed.

### Phase 2: the catalog (2026-10-20)

* **No deviation.** Phase 1 was committed by the owner as `d1e089b`. The
  steps name no table, but the MADR's D2 (its C17) gives `catalog`'s rows
  the import, and this PLAN's C5 is that row: R2's `catalog` row and
  `docs/architecture.md`'s add `internal/nilsource`. `make tables-check`:
  "38 packages, 0 problem(s)".
* **Step 1,** `listingToken`: a nil interface gives "%w: model listing:
  TokenSource is required", a typed nil "%w: model listing: a nil %T is not
  a TokenSource", each with `llmprovider.ErrInvalidRequest`, before any
  method of the source is called.
* **Step 2,** `LookupModel`: a typed nil gives no listing layer and appends
  "model metadata: listing %s: %w: a nil %T is not a TokenSource", for any
  provider, listing layer or not; the caller's layers and the document are
  merged as before. A nil interface is unchanged.
* **Step 3,** `CachedModel`: `src != nil && !nilsource.Is(src)` guards the
  listing read and the fetch.
* **Step 4, the docs:** `List` and `ListRaw` name a missing and a typed-nil
  source as `ErrInvalidRequest`; `LookupModel`, a typed nil reported so,
  beside the other layers; `CachedModel`, a typed nil as no source. `make
  docs-check`: "sdkaudit: 0 violation(s) of R50,R51,R52".
* **Step 5, the tests,** under `-race`, in `catalog_test`:
  `TestList_TypedNilSources_0050` (every built-in provider, `List` and
  `ListRaw`, a nil source and five typed-nil ones: 120 calls, each
  `ErrInvalidRequest` with its message, none a panic) and
  `TestLookupModel_TypedNilSources_0050` (Claude, a caller's layer that
  knows the model: `LookupModel` gives the layer's context window, with no
  error for a nil source and `ErrInvalidRequest` naming the type for each
  typed nil; `CachedModel` the layer's, known; none a panic).
* **Step 6, the MADR's probe,** unchanged, on a copy of the working tree: 30
  lines, each "New=ErrInvalidRequest | Generate=- | List=ErrInvalidRequest",
  0 "PANIC"; the catalog probe "ListRaw=ErrInvalidRequest",
  "LookupModel=ErrInvalidRequest", "CachedModel=ok". Before, every line
  held a panic.
* **Step 7, red first,** four plants, all caught:
  * `listingToken` without its typed-nil check: "typed_nil_0050_test.go:55:
    gemini, *llmprovider.StaticToken: List panicked: runtime error: invalid
    memory address or nil pointer dereference; want no panic";
  * the nil interface's error left plain: "typed_nil_0050_test.go:80:
    gemini, <nil>: List = model listing: TokenSource is required; want
    ErrInvalidRequest";
  * `LookupModel` without its check: ":55: *llmprovider.StaticToken:
    LookupModel panicked", and ":113: … LookupModel err = <nil>; want
    ErrInvalidRequest naming its type";
  * `CachedModel` without its check: ":55: *llmprovider.StaticToken:
    CachedModel panicked", for each of the five.
* **Step 8, `CHANGELOG.md`,** `[Unreleased]`: Changed, a missing source in
  `List` and `ListRaw` matches `ErrInvalidRequest` (Q2); Fixed, a typed-nil
  source refused by every `New`, `List`, `ListRaw` and `LookupModel`, and no
  source to `CachedModel`. `make changelog-check`: "25 release section(s), 0
  problem(s)".
* **The checks:** `tools/apidiff.py v1.17.0`, no output, exit 0; `make
  coverage-check`, "37 packages, 0 problem(s)", `catalog` at 95.5 %; `make
  pre-add-check` on the four Go files, "4 file(s) clean"; `tools/gate.py`
  on the final tree, 27 of 27 passed.

### Phase 3: release `v1.18.0` (2026-10-20)

* **No deviation so far.** Phase 2 was committed by the owner as `6724d45`.
* **Step 1.** `CHANGELOG.md`: `[Unreleased]` became `[v1.18.0] -
  2026-10-20`, with Changed (Q2) and Fixed (D3, D4); an empty
  `[Unreleased]` above it; the comparison links gain `v1.18.0`.
  "changelog-check: 26 release section(s), 0 problem(s)". The date must be
  the commit's that the tag names (R53). The README's Status names
  `v1.18.0`. `tools/linkcheck.py`, 0 problem(s); markdownlint on both
  files, no finding (its 189 are all in
  `docs/reports/0011-REPORT-provider-source-compatibility-audit.md`, which
  the configuration's globs add).
* **`--section v1.18.0`** in the tree exits 1, "v1.18.0's section links
  docs/decisions/0050-MADR-refuse-typed-nil-token-sources.md, which v1.18.0
  lacks": it reads each link at the tag, and there is no tag yet. In a
  local clone of `6724d45`, which holds the MADR, with this `CHANGELOG.md`
  and a local `v1.18.0` tag, it exits 0 and prints the 14 lines of Changed
  and Fixed, the MADR's link made absolute at `blob/v1.18.0`. It runs again
  against the Release at step 3.
* **Step 2, before the push:** `tools/gate.py`, 27 of 27 passed;
  `tools/apidiff.py v1.17.0`, no output, exit 0: no exported name changed;
  the disclosure guard over the outgoing commits, once this one is
  committed, before the owner pushes.
* **Step 2, the guard:** the owner committed step 1 as `d19f7a3`; the
  disclosure guard over the 5 outgoing commits (`b92cf29..d19f7a3`), run by
  the agent before the push: exit 0, nothing reported. The owner pushed
  `main`; CI run 39433940716: `validate` on ubuntu-24.04, macos-15 and
  windows-2025, success; `release` skipped, as on a branch.
* **Step 3,** the owner tagged `v1.18.0` on `d19f7a3` (commit date
  2026-10-20, the section's) and pushed it; run 39435526622: `validate` on
  the three systems and `release`, success. The
  [Release](https://github.com/maccavelli/go-llmprovider-sdk/releases/tag/v1.18.0),
  not a draft or a prerelease, Latest; `gh release view v1.18.0 --json
  body` equals `check_changelog.py --section v1.18.0 --repo
  maccavelli/go-llmprovider-sdk`, 661 characters each, the latter exit 0
  now that the tag holds the MADR.
* **Step 4:** `tools/consumer.py v1.18.0`, "consumer: 15 packages linked",
  "every step passed"; `tools/offline.py v1.18.0`, "offline: ok", 0
  requests, its two plants caught, "every step passed".
* **Step 5, the MADR's Confirmation:**
  * the probe of F1 to F3, run again unchanged on a scratch copy of
    `v1.18.0`: 30 of 30 lines "New=ErrInvalidRequest | Generate=- |
    List=ErrInvalidRequest", 0 "PANIC"; "ListRaw=ErrInvalidRequest",
    "LookupModel=ErrInvalidRequest", "CachedModel=ok", D4's answers;
  * D5's tests pass, and each failed on its planted break (Phases 1 and 2:
    seven plants, all caught);
  * at `v1.18.0`, `scripts/check_tables.py`, "38 packages, 0 problem(s)";
    `sdkaudit -check`, "0 violation(s) of R50,R51,R52"; `tools/apidiff.py
    v1.17.0`, no change (step 2 of this phase).
* **The Verification table:** V1 Phase 1 (100 %); V2 Phase 1's provider
  table; V3 and V4 Phase 2's catalog tests; V5 Phase 2's probe and step 5's;
  V6 apidiff and `docs-check` at every phase; V7 step 3; V8 each phase's
  plants.
* **Not done, and why:** nil-safe methods on the module's sources, the
  MADR's option B, out of scope; gobble-cli's and prepare-commit-msg's
  adoption of `v1.18.0`, theirs to make.
* **Status:** this PLAN `complete`, by `tools/records.py status`.
