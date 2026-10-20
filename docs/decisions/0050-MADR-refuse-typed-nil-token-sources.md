---
status: accepted
date: 2026-10-19
decision-makers: repository owner
consulted: this module at 0fbaf90 (v1.17.0 is 1fa0d94); probes on a scratch copy of 0fbaf90 (method below); 0049-MADR-v1-16-0-follow-ups.md, which fixed the *StaticToken case and left the others unprobed
informed: anyone who builds a provider, lists models or looks a model up with a token source
---

<!-- markdownlint-disable MD013 -->

# Refuse a Typed-Nil Token Source Where It Enters, With `ErrInvalidRequest`

## Context and Problem Statement

A token source reaches this module in two ways:

1. **Through an option,** `WithTokenSource(src)` (or `WithAPIKey`, which
   builds a `*StaticToken`), read by `llmprovider.ResolveOptions`. Every
   built-in provider's `New` calls it first (C10 to C12), and so does
   `catalog`'s `configFor` for every catalog call that takes options (C13),
   `CachedFacts` among them.
2. **As an argument** of four catalog functions: `List` and `ListRaw`, which
   need one, and `LookupModel` and `CachedModel`, for which a nil source
   means no listing layer (C14, C15).

A **typed nil** is a nil pointer (or another nil value of a nilable kind)
held in a non-nil `TokenSource` interface: `src == nil` is false, so the
source is used, and its first method call dereferences nil.

[0049-MADR-v1-16-0-follow-ups.md](0049-MADR-v1-16-0-follow-ups.md) D1 made
`ResolveOptions` refuse one such value, a nil `*StaticToken`, with
`ErrInvalidRequest` (C8). Its PLAN left the rest out of scope: "Other
typed-nil token sources, which the MADR did not probe." This record probes
them and decides how every one is handled.

### How the facts were found (2026-10-19)

* **[probed]**: a test file added to a scratch copy of the tree at
  `0fbaf90`, never the tree, run with `go test -v`. Each call was wrapped to
  recover a panic and print its first line, or else the error's kind. Each
  exported pointer source other than `*StaticToken`, typed-nil
  (`*llmprovider.CommandToken`, `*auth.OAuthSession`,
  `*auth.VendorCLISession`), was given to every one of the ten built-in
  providers:
  * through `providers.New(id, WithTokenSource(src), …)`;
  * then, where `New` succeeded, through `Generate`, with a client that
    refuses every connection (`nettest.Refusing`), so the token is asked
    for and nothing leaves the machine;
  * and through `catalog.List(ctx, id, src, …)` with the same client.

  A second probe gave a typed-nil `*CommandToken` to `catalog.ListRaw`,
  `LookupModel` and `CachedModel`, for Claude.
* **[read]**: the code at the line the claims table pins.

### What was measured

* **F1. Every catalog entry point panics on a typed-nil source.**
  `catalog.List` panics, "runtime error: invalid memory address or nil
  pointer dereference", for all 30 pairs of the three sources and ten
  providers **[probed]**. `ListRaw`, `LookupModel` and `CachedModel` panic
  too **[probed]**. `List` and `ListRaw` share `listingToken`, which refuses
  only a nil interface (C1), then calls `Token` (C2); `LookupModel` and
  `CachedModel` check `src != nil` (C14, C15) and call `Token` themselves
  **[read]**.
* **F2. A typed-nil `*CommandToken` is accepted by every `New`, and its
  first `Generate` panics,** for all ten providers **[probed]**: its
  `Token` takes the receiver's lock first (C3) **[read]**.
* **F3. A typed-nil session panics inside `New` for three providers.**
  * A nil `*auth.OAuthSession`: OpenAI, Grok and Kilo panic in `New`; the
    other seven refuse it with `ErrUnsupported` **[probed]**.
  * A nil `*auth.VendorCLISession`: OpenAI and Grok panic in `New`; the
    other eight refuse it with `ErrUnsupported` **[probed]**.
  * OpenAI and Grok call the session's `Owner` (C4, C5), which takes its
    lock (C6); Kilo reads its `Provider` field (C7) **[read]**. Each does so
    after `ResolveOptions` (C10 to C12), so a refusal there comes first.
* **F4. 0049's check is by type,** `*StaticToken` alone (C8), so it reaches
  no other source of this module, nor a caller's own type **[read]**.
* **F5. A nil source differs by entry point.** `List` and `ListRaw`
  refuse a nil interface with a plain error, "model listing: TokenSource is
  required" (C1), which `List`'s doc names "a missing source" without a
  kind (C16); `LookupModel` and `CachedModel` treat it as no listing layer,
  as their docs say **[read]**.
* **F6. 0048's rule** "Panics only on programmer error" lists six
  deliberate panics, none of them a token source (C9) **[read]**. R23 of
  `docs/guides/api-standards.md` says an invalid value is
  `ErrInvalidRequest` from `New` or `Generate`.

### Code claims

<!-- claims: self=0fbaf90 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The listing refuses only a nil interface | `llmprovider/catalog/discovery.go:242` | `if src == nil {` |
| C2 | It then asks the source | `llmprovider/catalog/discovery.go:252` | `token, err := src.Token(ctx)` |
| C3 | `CommandToken.Token` locks its receiver first | `llmprovider/command_token.go:100` | `c.mu.Lock()` |
| C4 | OpenAI's `New` asks a session's owner | `llmprovider/providers/openai/openai.go:205` | `if owner := s.Owner(); owner != llmprovider.ProviderOpenAI {` |
| C5 | Grok's, likewise | `llmprovider/providers/grok/grok.go:160` | `if owner := s.Owner(); owner != llmprovider.ProviderGrok {` |
| C6 | `Owner` locks its receiver | `llmprovider/auth/oauth_session.go:324` | `s.mu.Lock()` |
| C7 | Kilo's `New` reads a session's provider | `llmprovider/providers/kilo/kilo.go:188` | `if s.Provider != llmprovider.ProviderKilo {` |
| C8 | 0049's check is by type | `llmprovider/settings.go:239` | `if st == nil {` |
| C9 | 0048's deliberate panics | `docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md:90` | `**Panics only on programmer error.**` |
| C10 | OpenAI's `New` resolves its options first | `llmprovider/providers/openai/openai.go:189` | `st, err := llmprovider.ResolveOptions(llmprovider.ProviderOpenAI, opts)` |
| C11 | Grok's, likewise | `llmprovider/providers/grok/grok.go:145` | `st, err := llmprovider.ResolveOptions(llmprovider.ProviderGrok, opts)` |
| C12 | Kilo's, likewise | `llmprovider/providers/kilo/kilo.go:173` | `st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)` |
| C13 | The catalog resolves its options through it | `llmprovider/catalog/config.go:163` | `st, err := llmprovider.ResolveOptions(id, opts)` |
| C14 | `LookupModel`'s listing layer, when src is not nil | `llmprovider/catalog/lookup.go:84` | `if src != nil && listingLayers[id] {` |
| C15 | `CachedModel`'s, likewise | `llmprovider/catalog/lookup.go:112` | `if src != nil && listingLayers[id] {` |
| C16 | `List`'s doc names a missing source | `llmprovider/catalog/discovery.go:196` | `// stale. Only Ollama, an unknown provider, a missing source, a token` |
| C17 | R2 lets `llmprovider` import two internal packages | `docs/guides/api-standards.md:33` | `` the standard library, `internal/redact`, `internal/transport` `` |

## Decision Drivers

* No exported call panics on a value a caller can build through the API
  (0049-MADR's driver; R23).
* One test of nilness serves every source: this module's four types, and a
  caller's own type of any nilable kind, so a new source type cannot reopen
  the gap.
* No exported name is added or changed (R48, R55).
* Each entry point keeps its contract for a nil interface, which callers
  rely on; a typed nil is handled as near to it as the contract allows.
* The check sits where a source enters, so no method and no `New` needs one
  of its own.

## Considered Options

* **A. Refuse a typed-nil source where it enters:** in `ResolveOptions`,
  for every `New` and every catalog call's options; and in the four catalog
  functions that take a source as an argument. Any `TokenSource` whose
  dynamic value is nil, of any type.
* **B. Make each of this module's sources nil-safe:** `Token`, `Owner` and
  their kin return an error on a nil receiver, and each `New` checks before
  reading a session's field.
* **C. Both A and B.**
* **D. Document a typed-nil source as a programmer error, and let it
  panic.**

## Decision Outcome

Chosen option: "A", because a test of nilness at the two places a source
enters covers every source and every path after it, a caller's own type
included, with no new export; B covers only this module's types and needs a
check in each method and each `New`, and D keeps panics that 0049 already
judged errors for `*StaticToken`.

### D1. What counts as a typed nil

* A `TokenSource` that is not a nil interface, whose dynamic value is of a
  nilable kind (a pointer, map, slice, func, chan or interface) and is nil:
  `reflect.ValueOf(src)`'s `Kind` is one of those and its `IsNil` is true.
* A nil interface is not a typed nil, and keeps each entry point's present
  meaning (D3, D4).
* No method of the source is called to decide it.

### D2. Where the test lives

* An internal package, `llmprovider/internal/nilsource`, with one function,
  `Is(v any) bool`, by D1, and its own tests. It imports the standard
  library alone.
* `llmprovider` and `catalog` import it. R2's table gains its row, and the
  rows of `llmprovider` and `catalog` add it to what each may import (C17);
  `docs/architecture.md` likewise.

### D3. Through an option: `ResolveOptions`

* A typed-nil source given with `WithTokenSource` is refused with an error
  matching `ErrInvalidRequest`:
  `fmt.Errorf("%w: WithTokenSource: a nil %T", ErrInvalidRequest, src)`,
  for example `llmprovider: invalid request: WithTokenSource: a nil
  *llmprovider.CommandToken`.
* It replaces 0049's `*StaticToken` check (C8), whose case it includes; that
  case's message becomes `… a nil *llmprovider.StaticToken`, where it read
  `a nil *StaticToken`. Its kind is unchanged, and 0049's test asserts the
  kind alone.
* So every `New` refuses it before it reads the source (C10 to C12), and so
  does every catalog call whose options carry it (C13), `CachedFacts`
  included, which reports nothing known, as it does for any option
  `ResolveOptions` refuses.
* `WithTokenSource(nil)` is still no source.

### D4. As an argument: the catalog

* **`List` and `ListRaw`** (through `listingToken`): a typed-nil source is
  refused with an error matching `ErrInvalidRequest`,
  `fmt.Errorf("%w: model listing: a nil %T is not a TokenSource",
  llmprovider.ErrInvalidRequest, src)`. By the owner's Q2, a nil interface
  matches `ErrInvalidRequest` too: its error keeps its message, "model
  listing: TokenSource is required", after the sentinel's text, so like
  failures match like kinds, as 0049 D2 decided for its paths.
* **`LookupModel`:** a typed-nil source gives no listing layer, as a nil one
  does, and is reported in the joined error, as a source that fails is ("A
  source that fails is reported in the error, beside what the other layers
  set"), matching `ErrInvalidRequest`; the other layers are merged as
  today.
* **`CachedModel`,** which returns no error: a typed-nil source gives no
  listing layer, as a nil one does, and starts no listing fetch.

### D5. The docs and the tests

* **Docs** (R52): `WithTokenSource` and `ResolveOptions` name the refusal;
  `List`, `ListRaw` and `LookupModel` name `ErrInvalidRequest` for a
  typed-nil source; `CachedModel` says a typed-nil source is no source.
* **Tests,** each seen to fail first on a planted break (the owner's rule):
  * `nilsource.Is` over each nilable kind, nil and not, and over non-nilable
    values;
  * `ResolveOptions`, and `New` of every built-in provider, with a typed-nil
    `*StaticToken`, `*CommandToken`, `*auth.OAuthSession`,
    `*auth.VendorCLISession` and a caller's own source of a func type: each
    `ErrInvalidRequest`, none a panic;
  * `List`, `ListRaw`, `LookupModel` and `CachedModel` with each typed-nil
    source, by D4, none a panic;
  * the probe of F1 to F3, run again unchanged, prints no panic.

### D6. The release

* `v1.18.0`, a minor release, by the owner's Q1 and Q2: a nil interface's
  error in `List` and `ListRaw` now matches `ErrInvalidRequest`, an
  `errors.Is` answer that changes, as 0049 released its D2 in a minor
  version. The changelog lists D3 and D4's typed-nil refusals under Fixed,
  and the nil interface's new kind under Changed. No exported name changes.

### Consequences

* Good, because no built-in provider and no catalog entry point panics on a
  typed-nil source, of this module's types or a caller's.
* Good, because one tested function, in two call sites, decides it.
* Neutral, because detecting a nil behind an interface needs `reflect`, in
  the standard library; it runs once per `New` and per catalog call, not per
  request.
* Neutral, because 0049's message for a nil `*StaticToken` gains the
  package's name.
* Bad, because a source that is a nil value of a type whose methods were
  written to work on a nil receiver is refused; no source of this module is
  one, and such a source is unusual.
* Bad, because a caller who calls a nil source's methods directly, outside
  the module, still panics; B would cover that, for this module's types
  only.

### Confirmation

* The probe of F1 to F3, run again unchanged on the result: `ErrInvalidRequest`
  for every `New` and every `List`, D4's answers for the lookups, and no
  panic.
* D5's tests pass, and each was seen to fail on its planted break.
* `make tables-check` passes with D2's rows; `sdkaudit -check` reports 0;
  `tools/apidiff.py v1.17.0` reports nothing.

## Pros and Cons of the Options

### A. Refuse a typed-nil source where it enters

* Good, because it covers every source type and every path after it.
* Good, because it adds no export, and its check is in one package.
* Bad, because it uses `reflect` for a nil check, and adds an internal
  package.

### B. Make each source nil-safe

* Good, because no `reflect`.
* Bad, because a caller's own source is not covered, and each of this
  module's sources, and each `New` that reads one, needs a check of its own.

### C. Both

* Good, because a source is safe even used directly, outside the module.
* Bad, because the second half adds checks that A already makes unreachable
  through the module's calls.

### D. Document it as a programmer error

* Good, because nothing changes.
* Bad, because 0049 made the `*StaticToken` case an error, and the rest
  would stay panics, unlike it.

### D2's options

* **An internal package, `internal/nilsource` (chosen):** Good, because one
  function, tested once, serves both call sites. Bad, because R2's table
  changes for a five-line function.
* **The function twice, unexported, in `llmprovider` and `catalog`:** Good,
  because R2 does not change. Bad, because two copies can drift, and each
  needs its own tests.
* **In `internal/transport`, which both import:** Good, because no new
  package. Bad, because nilness is not transport, the package R2 gives the
  default client, identity headers, `Retry-After` and the listing probe.

### D4's options for the lookups

* **No listing layer, and in `LookupModel` a reported error (chosen):**
  Good, because it keeps their contract that the source is optional, and
  reports the mistake where the function can.
* **Refuse as `List` does:** Bad, because `LookupModel` would then fail a
  lookup that has its other layers, and `CachedModel` has no error to give.

## Owner questions

* **Q1. The version:** `v1.17.1` or `v1.18.0`, which follows from Q2.
* **Q2. A nil interface in `List` and `ListRaw`:** keep its plain error
  (recommended for a patch: no `errors.Is` answer changes), or make it match
  `ErrInvalidRequest` too, as the typed nil does (alike failures, alike
  kinds, as 0049 D2 did; a minor release).

The owner answered (2026-10-19): Q2 align it, so a nil interface in `List`
and `ListRaw` matches `ErrInvalidRequest`; Q1 `v1.18.0`, which follows.

## More Information

* Left open by
  [0049-PLAN-v1-16-0-follow-ups.md](0049-PLAN-v1-16-0-follow-ups.md)'s
  scope, "Other typed-nil token sources, which the MADR did not probe".
* The probes' files were scratch files, not kept; each finding above gives
  what they printed.
* The PLAN, `0050-PLAN-refuse-typed-nil-token-sources.md`, follows once
  this record is reviewed.
