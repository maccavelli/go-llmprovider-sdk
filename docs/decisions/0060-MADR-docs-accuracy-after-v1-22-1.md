---
status: proposed
date: 2026-10-24
decision-makers: repository owner
consulted: "the owner's request of 2026-10-24, \"ensure docs are up-to-date and accurate\"; a read of README.md, AGENTS.md, docs/architecture.md, docs/guides/, tools/README.md and the wizard's doc comments at 9ff2366; 0059-MADR-wizard-debugging-pass-findings.md and its PLAN; 0058-MADR-test-cache-isolation-for-reused-ports.md and its amendment of 2026-10-24; the Makefile, scripts/test_gates.py, scripts/coverage-floors.txt, .github/workflows/ and tools/"
informed: contributors, and consumers who read the README and docs/architecture.md
---

<!-- markdownlint-disable MD013 -->

# Bring the Docs Up to Date with v1.22.1's Wizard and the Local Gate

## Context and Problem Statement

`v1.22.1` (0059) changed how the wizard treats a typed key's shape, a
fresh session, a saved session, the paste prompt, the masked entry, saved
fallbacks, an ended context and a plain-http endpoint under
`NoEndpointPrompt`. 0059's PLAN updated each changed function's doc, the
changelog, and one paragraph of `docs/architecture.md`, "The wizard"
(the session's save). The rest of that section, and the README's wizard
section, still describe `v1.22.0`. 0058's amendment of 2026-10-24 added a
census option, a gate check and a test rule that no contributor-facing
doc names. And `docs/architecture.md`'s Tooling section has drifted from
the Makefile and the gate self-test on its own.

The owner asked, on 2026-10-24, that the docs be "up-to-date and
accurate". Which docs change, and what each says, is this record.

### Findings, at `9ff2366`

**A. The wizard's behaviour (0059's own Scope, partly done):**

* **A1. Wrong.** `docs/architecture.md` "What it accepts": "A typed key
  without its provider's published shape is warned about, not refused"
  (C1). Since 0059 S1 no shape warning comes before the listing: a typed
  key is named before it only when it is plainly another vendor's
  (`foreignKeyFor`, C2), and the shape is named only after the provider
  refuses a typed key without it (`notifyShape`, C3).
* **A2. Incomplete.** Plain http to a remote host is "used only once
  confirmed" (C4) and "asks first" (C5). Under `NoEndpointPrompt` it is
  used with a warning notice and no question (0059 W9, C6).
* **A3. Incomplete.** "An empty paste is asked again" (C7). After three
  empty answers each within 50 ms, the wizard says so and waits for the
  browser (0059 W5, C8).
* **A4. Incomplete.** The README: "the wizard offers to keep" an existing
  session (C9); `docs/architecture.md`: "A sign-in saves its session"
  (C10). A fresh session is saved once the listing has not refused it (0059
  W2, C11); a saved session that cannot be used is not offered, a notice
  says why, and the sign-in menu follows (0059 W6, C12).
* **A5. Incomplete.** The masked entry "ignores a lone Escape" (C13); it
  now follows an escape sequence split across reads (0059 W4, C14).
* **A6. Missing.** Neither doc says that a saved fallback the menu does
  not recommend is offered, marked "saved" (0059 W7, C15), or that a run
  whose context has ended asks nothing (0059 W8, C16).

**B. The local gate and its test rule (0058's amendment):**

* **B1.** The architecture's account of `tools/gate.py` lists its checks
  (C17) without `test-repeat`, every package run twice in one process
  (C18).
* **B2.** `AGENTS.md`'s list of maintainer tools (C19) has no entry for
  `tools/cachecensus.py`, added by 0058's Phase 1 and given `--any-url`
  by its Phase 6.
* **B3.** 0058's D1 and D6, that a test which lists or probes through the
  catalog gives its calls a cache of its own (`testCache(t)` in the wizard,
  C20; each provider package's `build`), are in the records and the
  helpers, and in no doc a test's author reads.

**C. Drift in `docs/architecture.md`'s Tooling section:**

* **C-1.** The list of `make` targets (C21) lacks `tables-check`,
  `fixtures-check`, `changelog-check` and `docs-check`, which the Makefile
  declares (C22).
* **C-2.** `gate-selftest` is said to plant one breach per gate for six
  gates (C23). `scripts/test_gates.py` plants breaches of ten: those six,
  `tables-check`, `fixtures-check`, `docs-check` and `changelog-check`.

**Checked and accurate:** the README's Status (`v1.22.1`); the
five-Release rule where the README, `CHANGELOG.md` and
`docs/architecture.md` state it; `CHANGELOG.md`; the coverage floors
(`wizard` 86.4 %, as `scripts/coverage-floors.txt`); CI's `make` lines;
`tools/README.md`'s rows; the `Prompter` and `InputerContext` docs; the
doc comments 0059's phases changed.

**Not docs to change:** the records. They are history; several link a
GitHub Release that R53 has since deleted (`v1.10.2` to `v1.19.0`), whose
tag and changelog section remain.

## Decision Drivers

* A doc states what the code does, and nothing the code no longer does.
* Every behaviour a doc states is one a test shows (0059's rule).
* A rule a contributor must follow is written where a contributor reads.
* No code, test or behaviour changes; no release.

## Considered Options

* **O1.** Fix A, B and C in the docs that hold them.
* **O2.** Fix A and B only, the drift of 0059 and 0058; leave C.
* **O3.** Fix A only, 0059's leftover.

## Decision Outcome

Chosen option: *proposed* **O1**, because each of A, B and C is a
statement a reader would act on that is now wrong or missing, the fixes
are text only, and C is two lists checked mechanically against their
sources. Awaits Q1 to Q3.

### D1. The wizard (A)

* `docs/architecture.md`, "The wizard": "One copy of a session", "The
  paste prompt", "What it accepts", "A refused key", "`TextPrompter`", and
  a sentence each for saved fallbacks and an ended context, say what
  `v1.22.1` does, citing 0059's findings.
* `README.md`, "The configuration wizard": the session bullet and the
  base-URL bullet likewise; a saved fallback in the search bullet.

### D2. The gate and its rule (B)

* `docs/architecture.md`'s account of `tools/gate.py` names
  `test-repeat`.
* `AGENTS.md`'s tool list gains `tools/cachecensus.py [--any-url]`.
* Under Q2, the test rule is written in `AGENTS.md` beside the tool list,
  and in `docs/guides/adding-a-provider.md`'s tests.

### D3. The Tooling section (C)

* The `make` target list matches the Makefile's `.PHONY`.
* The `gate-selftest` bullet names the ten gates it plants.

### D4. Nothing else

No Go file, test, CI file, `Makefile` or record of another number
changes. No `CHANGELOG.md` entry: the change is to docs, not to the
module.

### Consequences

* Good, because the architecture no longer tells a reader of a shape
  warning that `v1.22.1` removed.
* Good, because the next test author meets the cache rule before the
  census or `test-repeat` does.
* Neutral, because the records' links to deleted Releases stay dead (Q3).
* Bad, because two long docs grow by a few sentences each.

### Confirmation

* Each finding of A, B and C is resolved in its doc, and the PLAN names,
  for each wizard sentence changed, the test that shows it.
* `make docs-check`, `tools/linkcheck.py`, `tools/denyscan.py --changed`,
  markdownlint through `python3 -B tools/gate.py`, and `make
  records-check` pass.

## Pros and Cons of the Options

### O1. A, B and C

* Good, because every inaccuracy found is fixed in one pass.
* Bad, because C was not caused by 0059 or 0058, so the pass is wider than
  the release.

### O2. A and B

* Good, because it stays with what the recent work changed.
* Bad, because C's two lists stay wrong, and a reader trusts them.

### O3. A only

* Good, because it is the smallest.
* Bad, because B3's rule stays where no contributor reads it.

## Owner questions

* **Q1 (scope).** (a) O1 (recommended); (b) O2; (c) O3.
* **Q2 (B3, where the cache rule is written).** (a) `AGENTS.md`, beside
  the tool list, and `docs/guides/adding-a-provider.md`'s tests
  (recommended); (b) `AGENTS.md` only; (c) not written: the helpers and the
  census hold it.
* **Q3 (the records' links to deleted Releases).** (a) Leave them, as
  each earlier deletion did (recommended); (b) annotate each in a later
  record of its own; this record would not.

## More Information

### Code claims

<!-- claims: self=9ff2366 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The architecture's shape warning | `docs/architecture.md:854` | `its provider's published shape is warned about, not refused.` |
| C2 | Another vendor's key | `wizard/auth.go:549` | `func foreignKeyFor(provider llmprovider.ProviderID, value string) string {` |
| C3 | The shape, after a refusal | `wizard/auth.go:605` | `func notifyShape(p Prompter, d llmprovider.Descriptor, key string) {` |
| C4 | Plain http, confirmed | `docs/architecture.md:851` | `is used only once confirmed` |
| C5 | Plain http, asked | `README.md:289` | `` Plain `http` to a remote host asks first. `` |
| C6 | Not asked under NoEndpointPrompt | `wizard/base_url.go:39` | `if !ask {` |
| C7 | An empty paste | `docs/architecture.md:836` | `An empty paste is asked again` |
| C8 | Three quick empties | `wizard/auth.go:783` | `pasteQuickEmpties = 3` |
| C9 | The README's keep | `README.md:270` | `the wizard offers to keep` |
| C10 | The architecture's save | `docs/architecture.md:815` | `A sign-in saves its session to` |
| C11 | Saved after the listing | `wizard/configure.go:368` | `if err := credential.save(ctx, d); err != nil {` |
| C12 | An unusable session not offered | `wizard/auth.go:447` | `session cannot be used (%v); sign in again` |
| C13 | A lone Escape | `docs/architecture.md:866` | `ignores a lone Escape` |
| C14 | Escapes across reads | `wizard/text_prompter.go:526` | `func (s escapeState) next(c rune) (escapeState, bool) {` |
| C15 | A saved fallback, marked | `wizard/model_select.go:25` | `savedFallbackDetail  = "saved"` |
| C16 | An ended context first | `wizard/configure.go:197` | `if err := ctx.Err(); err != nil {` |
| C17 | The gate's account | `docs/architecture.md:961` | `the link check, the identifier scan, and` |
| C18 | test-repeat | `tools/gate.py:93` | `Check("test-repeat", command(["go", "test", "-count=2", "./..."])),` |
| C19 | AGENTS' tool list | `AGENTS.md:170` | `` `tools/consumer.py` and `tools/offline.py` serve `` |
| C20 | The wizard's test cache | `wizard/fake_prompter_test.go:20` | `func testCache(t *testing.T) llmprovider.Option {` |
| C21 | The make targets | `docs/architecture.md:886` | `` `api-check`, `generate-check`, `records-check`, `gate-selftest`, `` |
| C22 | The Makefile's targets | `Makefile:14` | `tables-check fixtures-check changelog-check docs-check gate-selftest` |
| C23 | gate-selftest's six | `docs/architecture.md:938` | `generate-check, parity-check, coverage-check, api-check, records-check),` |

### Related

* [0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md):
  the behaviour of A.
* [0058-MADR-test-cache-isolation-for-reused-ports.md](0058-MADR-test-cache-isolation-for-reused-ports.md):
  B's census, gate check and rule.
* [0035-MADR-maintainer-tools-in-python.md](0035-MADR-maintainer-tools-in-python.md):
  `tools/`.
