---
status: accepted
date: 2026-11-11
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b, unchanged at eefb7c25), its questions of §11, and the owner's answers of 2026-11-10 to them; the SDK audit behind it; this repository at 773b84c (v1.29.0 and one records commit); gobble-cli at d6c10a5 (= acpext/v0.4.0), re-checked on 2026-11-11 at origin/main 17d1605, at which every gobble-cli file this record cites is unchanged; its acpext module, its `0042-MADR-acpext-extension-contract-module.md`, and its proposed, untracked `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` and `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` and their PLANs, as revised on 2026-11-11; magic-cli-remote at 995d442b, re-checked on 2026-11-11 at origin/master eefb7c25, and its proposed, untracked `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` and `0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md` and their PLANs; acp-go-sdk2 v1.35.0 read from the module cache, and v1.36.0 read from its repository on 2026-11-11; the owner's answers of 2026-11-10 to the Magic drafts' O1 and O3; the owner's rulings of 2026-11-11 on D3's constant names and on the Go floor, and the owner's ruling of 2026-11-11, after acceptance, on a session-kind type and its labels, with gobble-cli's 0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md D8 and magic-cli-remote's 0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md D9 and D13 as revised that day"
informed: gobble-cli and magic-cli-remote, which consume the module; every caller of llmprovider/auth and wizard
---

<!-- markdownlint-disable MD013 MD024 -->

# Publish the Sign-In Vocabulary as a Nested, Standard-Library-Only Module, `authcontract`, Versioned on Its Own

## Context and Problem Statement

Three repositories describe one thing, a way to sign in other than an API
key, in three vocabularies. This SDK names methods by `AuthMethodID`
(C1, C2), and its `AuthMethod` carries a label and two booleans, with no JSON
tags (C3). gobble-cli has its own `--method` words and `--json` events, and its
`acpext` module a `Login` boolean. magic-cli-remote has three method types and
its own device-flow messages. Every hop translates by hand. The 0245 report
(magic-cli-remote `docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md`,
findings F1, F2, F3 and F18) records the cost: a fact such as "this method
completes by a pasted URL" is decided in five places, differently.

The owner decided on 2026-11-10 to tie the phone's contract to this SDK, and
how (the answer to the report's Q8): a small module, standard library only, nested in this
repository with its own tags, whose types the SDK, gobble and Magic all use.
Magic embeds them in its own envelope. This record decides that module: its
name and path, its types, its JSON Schema, its golden vectors, the rules by
which it may change, and the gates that hold those rules.

The engine that produces these types (`llmprovider/x/signin`) is
[0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)'s.
The ChatGPT plan's consent is
[0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)'s.
This record holds data and rules only: no flow, no HTTP, no provider.

### What this repository has at `773b84c`

The claims below are pinned by the evidence index at the end of this
section, and `tools/claims.py` checks each against the revision.

* **One module.** `go.mod` names `github.com/maccavelli/go-llmprovider-sdk`
  (C4), and it is the only `go.mod` in the tree (read with `find`, 2026-11-10).
  No `llmprovider/x/` package and no `authcontract/` directory exist.
* **A second module needs a record.** The standards guide's R1: "A second
  module needs its own MADR and a dependency that justifies it" (C5).
  0015-MADR D2: "**No second module.** A module is split out only when a
  dependency justifies one, and that takes its own MADR" (C6). AGENTS.md
  allows the standard library and `golang.org/x/term` only, and "Any other
  module needs a MADR in this repository" (C7).
* **Every gate is written for one module.**
  * `make api-check` compares the root module, `apidiff -m` at the module
    path, with the latest tag matching `^v1\.(\d+)\.(\d+)$` (C8, C9). A
    nested module's packages are not in that comparison, and a tag
    `authcontract/v0.1.0` is not a release to it.
  * `make dep-check` passes any import under the module path's prefix (C10),
    which a nested module's path shares, but fails any `go.mod` requirement
    outside `golang.org/x/term` and `golang.org/x/sys` (C11).
  * `make tables-check` and `make coverage-check` read `go list ./...` and
    `go test ./...` from the root (C12, C13). Go's `./...` pattern stops at a
    directory holding its own `go.mod`, so neither sees a nested module.
  * `scripts/go-precheck.sh` runs `go vet` and `go test -race` on the
    directories of the files it is given, from the root (C14, C15). Go
    refuses a directory inside another module from there, so a nested
    module's file would fail the pre-add check for a reason that is not its
    own. The exact message is not reproduced here (unverified; Phase 1a's
    planted file shows it).
  * CI runs on pushes to `main`, on pull requests, weekly on a schedule,
    and, of tags, only on `v*` (C35, C36, C37, C16); its release job runs
    only for `refs/tags/v` (C17). It tests `./...` from the root (C18).
    `make changelog-check` reads `v*` tags only (C19).
  * Two gates do reach a nested module. `make docs-check` walks every
    directory that is not hidden or named `testdata`, `vendor`, `tools`,
    `scripts` or `docs` (C20), so R50 to R52 would apply. `make
    generate-check` reads every `//go:generate` line in every tracked Go file
    and runs it in that file's directory (C21), so a generated schema with an
    `-output` flag would be checked.
* **A precedent in a sibling repository.** gobble-cli publishes `acpext` as
  a nested module tagged `acpext/vX.Y.Z`, standard library only, required by
  its root at a tag with no `replace` and no committed `go.work`, with a
  dependency gate of its own (gobble-cli
  `docs/decisions/0042-MADR-acpext-extension-contract-module.md`, D1, D5, D6
  and A1).

### What the three vocabularies disagree on

* **Field names drift inside this SDK alone.** `OAuthFlowOptions.NotifyDevice`
  takes `(verificationURL, userCode string)` (C22); `auth.DeviceLogin` has
  `UserCode` and `VerificationURI` (C23, C24); `wizard.DeviceCode` has `URL`
  and `Code` (C25).
* **Error kinds disagree across repositories.** This SDK's sentinels are
  `ErrRateLimited`, `ErrProviderUnavailable`, `ErrAuthFailure`,
  `ErrInvalidRequest` (C26 to C29), `ErrNotPermitted`, `ErrUnsupported`
  (C30, C31) and the rest. gobble's ACP error kinds use hyphens, such as
  `rate-limited` (C32), and its `--json` sign-in failures use a second,
  shorter list with words of its own, such as `empty` (C33, C34). An offline
  sign-in is `other` in gobble's `--json` and `offline` in a turn (the
  report's F3, confirmed by C34: the list has no `offline`).

### Evidence index

<!-- claims: self=773b84c gobble-cli=17d1605 magic-cli-remote=eefb7c25 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The method ids are a named string | `llmprovider/descriptor.go:8` | `type AuthMethodID string` |
| C2 | A mechanism and a product side by side | `llmprovider/descriptor.go:24` | `AuthChatGPTPlan AuthMethodID = "chatgpt_plan"` |
| C3 | `AuthMethod` has no JSON tags | `llmprovider/descriptor.go:33` | `Interactive bool` |
| C4 | One module path | `go.mod:1` | `module github.com/maccavelli/go-llmprovider-sdk` |
| C5 | R1 | `docs/guides/api-standards.md:28` | `A second module needs its own MADR and a dependency that justifies it.` |
| C6 | 0015-MADR D2 | `docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md:126` | `**No second module.** A module is split out only when a dependency` |
| C7 | AGENTS.md's dependency rule | `AGENTS.md:33` | `Any other module needs a MADR in this repository` |
| C8 | api-check's tag pattern | `scripts/check_api.py:32` | `RELEASE = re.compile(r"^v1\.(\d+)\.(\d+)$")` |
| C9 | api-check compares the root module | `scripts/check_api.py:65` | `run(["go", "run", APIDIFF, "-m", "-w", str(new), MODULE], ROOT)` |
| C10 | dep-check passes the module's prefix | `scripts/check_deps.py:76` | `if dep in standard or dep == MODULE or dep.startswith(MODULE + "/"):` |
| C11 | dep-check's allowed requirements | `scripts/check_deps.py:37` | `REQUIRES = {"golang.org/x/term", "golang.org/x/sys"}` |
| C12 | tables-check lists the root's packages | `scripts/check_tables.py:64` | `proc = subprocess.run(["go", "list", "-f", "{{.ImportPath}}\|{{join .Imports \",\"}}", "./..."],` |
| C13 | coverage-check tests the root's packages | `scripts/check_coverage.py:69` | `for line in run("go", "test", "-count=1", "-cover", "./...").splitlines():` |
| C14 | The pre-add check vets from the root | `scripts/go-precheck.sh:134` | `if ! vet_out="$(go vet "${pkgs[@]}" 2>&1)"; then` |
| C15 | and tests from the root | `scripts/go-precheck.sh:139` | `if ! test_out="$(go test -race "${pkgs[@]}" 2>&1)"; then` |
| C16 | CI runs on `v*` tags | `.github/workflows/ci.yml:5` | `tags: ['v*']` |
| C17 | The release job's tags | `.github/workflows/ci.yml:73` | `if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')` |
| C18 | CI tests the root module | `.github/workflows/ci.yml:46` | `- run: go test -shuffle=on ./...` |
| C19 | changelog-check reads `v*` tags | `scripts/check_changelog.py:86` | `tags = [t for t in git("tag", "--list", "v*").split() if VERSION.match(t)]` |
| C20 | docs-check walks every directory | `llmprovider/internal/cmd/sdkaudit/audit.go:53` | `err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {` |
| C21 | generate-check reads every tracked Go file | `scripts/check_generated.py:44` | `files = subprocess.run(["git", "ls-files", "-co", "--exclude-standard", "*.go"], cwd=ROOT,` |
| C22 | `NotifyDevice`'s argument names | `llmprovider/auth/oauth_loopback.go:46` | `NotifyDevice func(verificationURL, userCode string)` |
| C23 | `DeviceLogin.UserCode` | `llmprovider/auth/oauth_device.go:85` | `UserCode string` |
| C24 | `DeviceLogin.VerificationURI` | `llmprovider/auth/oauth_device.go:88` | `VerificationURI string` |
| C25 | `wizard.DeviceCode.URL` | `wizard/device.go:19` | `URL string` |
| C26 | `ErrRateLimited` | `llmprovider/provider.go:65` | `ErrRateLimited         = errors.New("llmprovider: rate limited")` |
| C27 | `ErrProviderUnavailable` | `llmprovider/provider.go:66` | `ErrProviderUnavailable = errors.New("llmprovider: provider unavailable")` |
| C28 | `ErrAuthFailure` | `llmprovider/provider.go:67` | `ErrAuthFailure         = errors.New("llmprovider: authentication failed")` |
| C29 | `ErrInvalidRequest` | `llmprovider/provider.go:70` | `ErrInvalidRequest = errors.New("llmprovider: invalid request")` |
| C30 | `ErrNotPermitted` | `llmprovider/api_error.go:39` | `ErrNotPermitted = errors.New("llmprovider: not permitted")` |
| C31 | `ErrUnsupported` wraps the standard library's | `llmprovider/api_error.go:76` | `ErrUnsupported error = &sentinelError{msg: "llmprovider: unsupported", parent: errors.ErrUnsupported}` |
| C32 | gobble's ACP kinds use hyphens | `gobble-cli:llm/kind.go:31` | `{ErrRateLimited, "rate-limited"},` |
| C33 | gobble's sign-in failure kinds | `gobble-cli:llm/provider/list.go:113` | `{llm.ErrRateLimited, "rate-limited"},` |
| C34 | and a word of their own | `gobble-cli:llm/provider/list.go:119` | `{errNothingListed, "empty"},` |
| C35 | CI runs on pushes to `main` | `.github/workflows/ci.yml:4` | `branches: [main]` |
| C36 | and on pull requests | `.github/workflows/ci.yml:11` | `pull_request:` |
| C37 | and weekly | `.github/workflows/ci.yml:16` | `- cron: '17 6 * * 1'` |
| C38 | gobble's empty reply is `empty_reply`, with an underscore | `gobble-cli:llm/kind.go:33` | `{ErrEmptyReply, "empty_reply"},` |
| C39 | plant.py's default command runs from the clone root | `tools/plant.py:45` | `args = ["go", "test", "-count=1"]` |
| C40 | acpext's `ErrorKind`, an open enum of fourteen words | `gobble-cli:acpext/types.go:341` | `type ErrorKind string` |
| C41 | gobble's archtest refuses the SDK's prefix outside `llm/provider` | `gobble-cli:internal/archtest/check.go:308` | `if pathUnder(e.to, "github.com/maccavelli/go-llmprovider-sdk") && !under(rel, "llm/provider") {` |
| C42 | Magic's boundary test reads every import with the SDK's prefix | `magic-cli-remote:internal/provider/grok/sdk_boundary_test.go:72` | `if !strings.HasPrefix(ip, sdkModule) {` |
| C43 | Magic's `go.mod` requires this SDK, committed | `magic-cli-remote:go.mod:12` | `github.com/maccavelli/go-llmprovider-sdk v1.29.0` |
| C44 | gate-selftest's workflow contract asserts only the Markdown gates run in `ci.yml` | `scripts/test_gates.py:619` | `self.assertEqual([g for g in MARKDOWN_GATES if g not in ci], [], "ci.yml does not run these Markdown gates")` |

## Decision Drivers

* **The owner's decisions of 2026-11-10.** The 0245 report's §11 (its
  `:1168-1237`) puts twenty-two questions, each with two to four options, and
  records no answer. The owner answered them on 2026-11-10, choosing an
  option and stating the answer in words. The answers are recorded in the
  records that act on them, each quoting what it needs: this record and
  0074-MADR here; magic-cli-remote
  `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  (Context); gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` and
  `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` (Decision
  Drivers). Where two of them quote one answer, the wording below agrees with
  theirs but for case and the clauses each leaves out. Q1's words, and Q15's
  sentence "The SDK maps its sentinels to it", appear in no other record, so
  nothing outside this repository cross-checks them.
  * **Q8, the contract module** (option (c)). "Stdlib-only, nested in the SDK
    repo with its own tags. SDK, gobble and Magic all use its types. Magic
    embeds them in its own envelope." (magic-cli-remote's 0246-MADR quotes
    it with a further clause, "superseding Magic 0185-MADR in part".)
  * **Q1, the engine starts outside v1** (option (a)). "The contract at v0, the engine
    under `llmprovider/x/signin` (outside v1 api-check), promoted into
    `llmprovider/auth` in a minor release once gobble and Magic have run it."
  * **Q9, Dart models** (option (a)). "Generated from the contract's JSON Schema, with a
    generator and a drift gate." Magic generates them; the schema must suit
    that: flat objects, string enums, no `oneOf`.
  * **Q15, one error vocabulary** (option (a)). "The contract's `ErrorKind`
    everywhere. The SDK maps its sentinels to it, gobble retires
    `FailureKind` in `--json`, and acpext/llm `ErrorKind` aligns with it."
    "acpext/llm" names two vocabularies, not a package: there is no package
    `acpext/llm`. They are acpext's `ErrorKind`, fourteen open-enum words
    (gobble-cli `acpext/types.go`, C40), and gobble's `llm.Kind`, which
    returns the same fourteen (C32). gobble-cli's 0051-MADR quotes the clause
    as "gobble retires `FailureKind` in `--json`, and acpext/llm `ErrorKind`
    aligns".
  * **Q7, ACP elicitation now** (option (b), gobble's, against the report's
    recommendation): gobble's native sign-in runs inside ACP `authenticate`
    through request-scoped `elicitation/create`. The steps must therefore map
    onto ACP elicitation requests as well as onto gobble's `--json` and
    Magic's envelope.
  * **Q10 and Q11:** the method's client identity is shown on the phone, and
    a pasted callback is a secret: redacted, forwarded once, never stored.
* **The owner's answers of 2026-11-10 to two questions the Magic draft put to
  this module** (magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`,
  O1 and O3; the options as put are under More Information):
  * **O1, a method whose kind is known only at start** (kilo's and opencode's
    OAuth entries): "the SDK contract gains an `oauth` kind and an `at_start`
    completion (or equivalent) **before `authcontract/v0.1.0` is tagged**. The
    phone shows 'a sign-in whose type is learned on start'."
  * **O3, the command of a host-only method** (opencode's `command` and
    `external` entries): "a typed `command` field on the contract's `Method`,
    before `authcontract/v0.1.0`. The phone shows it with a copy button."
* **The owner's standing constraints of 2026-11-10.** The reusable bulk of
  auth goes in this SDK, "idempotent, reusable, canonical and flexible". The
  SDK stays multi-purpose: "no gobble-specific features or baggage". Magic
  stays a relay. The daemon and the app change in lockstep, so the phone is
  not the reason for the evolution rules; gobble and other SDK consumers,
  which upgrade on their own schedules, are.
* **This repository's rules.** A second module needs a record (C5, C6, C7);
  every new export is justified (R55); every exported type says whether it
  is safe for concurrent use and every error-returning function names its
  kinds (R51, R52); every public package has an example (R50); a new package
  holds 80 % coverage (R47); a secret-bearing value redacts itself (R33).
* **No baggage in either direction** (the report's §5.3.2). The module holds
  no transport, no session id, no admission or resume, no host availability
  reason, no product or app setting name, and no agent CLI's wire format.

## Considered Options

* **A. Mirror.** Each repository keeps its own vocabulary and maps to the
  others by hand; Magic writes its Dart models by hand (the report's §5.2
  (a)).
* **B. Import the full SDK.** Magic and gobble use `llmprovider/auth` types on
  their wires directly (the report's §5.2 (b)).
* **C. A nested contract module at the repository root, `authcontract/`,**
  standard library only, tagged `authcontract/vX.Y.Z` from `v0.1.0` (the
  report's §5.2 (c)).
* **D. The same module under `llmprovider/`,** at
  `llmprovider/authcontract/`, tagged `llmprovider/authcontract/vX.Y.Z`.
* **E. A standard-library-only package inside the root module,** such as
  `llmprovider/authwire`, released with the SDK under its v1 promise.
* **F. A repository of its own** (the report's Q1 (c)).

## Decision Outcome

Chosen option: "C. A nested contract module at the repository root,
`authcontract/`", because it gives the three repositories one vocabulary
with no dependency weight, lets the contract change on its own cadence and
under a promise stricter than Go's v0 rule, and follows a layout gobble-cli
already runs (`acpext`). The owner chose it on 2026-11-10 (Q8, with Q1, Q9
and Q15 as above), and the same day answered O1 and O3, which D2 carries
before the first tag. This record is `proposed`: it awaits the owner's review
of this draft.

On 2026-11-11 the owner ruled on two points of this draft:
the `Failure*` names of `ErrorKind`'s constants are kept, as a stated
exception to R21 (D3); and the Go floor that D1 follows is raised by a record
of its own,
[0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md),
approved the same day. The record stays `proposed`: the owner has answered
questions, not accepted it.

The owner accepted this record on 2026-11-11, and approved its PLAN the
same day.

Later on 2026-11-11 the owner amended the accepted record: the contract
gains a session-kind type, `SessionKind`, with a human label for each of its
four values, before `authcontract/v0.1.0` is tagged. "Amendment 2026-11-11:
a session-kind type and its labels", at the end of this record, holds it;
D2, D4 and D5 each carry a dated pointer to it, and their text is otherwise
not rewritten. The record stays `accepted`.

### D1. The module

* **Path and name.** The directory `authcontract/` at the repository root;
  the module path `github.com/maccavelli/go-llmprovider-sdk/authcontract`;
  one public package, `authcontract`. The name says what it is, a contract
  for authentication, and matches the report's working name.
* **`go.mod`.** `module github.com/maccavelli/go-llmprovider-sdk/authcontract`,
  a `go` line at R54's floor as the root's `go.mod` states it when the
  module is created, so one toolchain builds both modules, and no `require`
  line, so no `go.sum`. That floor is `go 1.27.3` at `773b84c`. On
  2026-11-11 govulncheck v1.8.0 reports GO-2026-6859 (`net/http`, fixed in
  Go 1.27.4) against this repository built with Go 1.27.3, and reports
  nothing with Go 1.27.4; the floor rises only by a record of its own, as
  `0055-MADR-go-1-27-3-and-the-v1-20-0-release.md` raised it to 1.27.3, and
  the module's line follows it. That record is
  [0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md),
  which the owner approved on 2026-11-11. It raises the root's floor to
  `go 1.27.4` before any of 0076's code, so the module's line is
  `go 1.27.4` once 0077's PLAN has run. *(2026-11-11: the record named; the
  bullet had none to name.)*
* **Imports.** The standard library only, in the package and in its tests. Its
  internal packages (D4's generator and validator) are standard library only
  too.
* **Tags.** `authcontract/vX.Y.Z`. The first is `authcontract/v0.1.0`. The
  module stays at v0 until gobble-cli and magic-cli-remote have each shipped
  a release that uses it, and `llmprovider/x/signin` has been promoted
  (0074-MADR D9); then a record moves it to `v1.0.0` (the owner's Q1).
* **No `replace`, no committed `go.work`.** The root module requires the
  contract at a published tag, from 0074's first import on. `go.work` and
  `go.work.sum` are added to `.gitignore`, which at `773b84c` names neither
  (its eighteen lines, read 2026-11-11), so a local workspace never reaches a
  commit, as gobble-cli `0042-MADR-acpext-extension-contract-module.md` D5 decided for `acpext`.

### D2. The types

These are the module's exported names. Each is a value type, safe for
concurrent use while no goroutine changes it (R51). Every JSON name is
`snake_case`; every list and map field is `omitempty`.

```go
// Package authcontract is the shared vocabulary of signing in without an API
// key: method facts, steps, answers and failure kinds, as data. No flow, no
// HTTP, no provider. Standard library only.
package authcontract

// SchemaVersion counts the incompatible changes of this module's JSON. The
// evolution rules forbid one within a major version, so it is 1 for v0 and v1.
const SchemaVersion = 1

type Feature string        // FeatureConsent "consent", FeatureForm "form", FeaturePaste "paste"
type Kind string           // KindAPIKey "api_key", KindTokenPaste "token_paste", KindDeviceCode "device_code",
                           // KindBrowser "browser", KindOAuth "oauth", KindVendorImport "vendor_import",
                           // KindHostOnly "host_only"
type Completion string     // CompletionValue "value", CompletionDevice "device", CompletionBrowserPaste "browser_paste",
                           // CompletionBrowserLoopback "browser_loopback", CompletionAtStart "at_start",
                           // CompletionLocalFile "local_file", CompletionHostCommand "host_command"
type Paste string          // PasteNone "", PasteURLOrCode "url_or_code", PasteURLOnly "url_only", PasteCode "code"
type Yields string         // YieldsAPIKey "api_key", YieldsOAuthSession "oauth_session", YieldsAccessOnly "access_only",
                           // YieldsDeviceKey "device_key", YieldsReadThrough "read_through", YieldsUnknown "unknown"
type ClientIdentity string // ClientNone "none", ClientVendorBorrowed "vendor_borrowed", ClientAppRegistered "app_registered",
                           // ClientUnknown "unknown"
type StepKind string       // StepOpenURL "open_url", StepShowCode "show_code", StepPaste "paste", StepConfirm "confirm",
                           // StepForm "form", StepWaiting "waiting", StepDone "done", StepFailed "failed"
type LinkRel string        // LinkTerms "terms", LinkUsage "usage", LinkNotice "notice"
type FieldType string      // FieldText "text", FieldSecret "secret", FieldSelect "select"
type ErrorKind string      // D3

// Method is one way to sign in, as facts a client reads before it starts.
type Method struct {
    SchemaVersion  int                        `json:"schema_version"`
    ID             string                     `json:"id"` // the provider's method id, e.g. "chatgpt_plan"
    Kind           Kind                       `json:"kind"`
    Label          string                     `json:"label"`
    Detail         string                     `json:"detail,omitempty"`
    Completion     Completion                 `json:"completion"`
    Paste          Paste                      `json:"paste,omitempty"`       // what a paste may be
    FirstPaste     Paste                      `json:"first_paste,omitempty"` // a first sign-in's, when it differs
    Yields         Yields                     `json:"yields"`
    ClientIdentity ClientIdentity             `json:"client_identity"`
    Refreshable    bool                       `json:"refreshable"`
    Revocable      bool                       `json:"revocable"`
    Slot           string                     `json:"slot,omitempty"` // methods that share one stored credential
    Command        string                     `json:"command,omitempty"` // host_command: what to run on the host; shown, never run
    Features       []Feature                  `json:"features,omitempty"`
    Consents       []Consent                  `json:"consents,omitempty"`
    Links          []Link                     `json:"links,omitempty"`
    Meta           map[string]json.RawMessage `json:"_meta,omitempty"`
}

// Consent is a yes or no a method's terms require, under a stable id such as
// "chatgpt_plan.background_use". Text is the default wording; an app may show its own.
type Consent struct {
    ID       string `json:"id"`
    Required bool   `json:"required"`
    Text     string `json:"text,omitempty"`
    Links    []Link `json:"links,omitempty"`
}

type Link struct {
    Rel   LinkRel `json:"rel"`
    URL   string  `json:"url"`
    Title string  `json:"title,omitempty"`
}

type Field struct {
    Key      string    `json:"key"`
    Label    string    `json:"label"`
    Type     FieldType `json:"type"`
    Options  []string  `json:"options,omitempty"`
    Required bool      `json:"required,omitempty"`
    Default  string    `json:"default,omitempty"`
}

// Step is one thing a sign-in shows or asks. Seq rises by one per step; the
// last is done or failed.
type Step struct {
    SchemaVersion int                        `json:"schema_version"`
    Seq           int                        `json:"seq"`
    Kind          StepKind                   `json:"kind"`
    URL           string                     `json:"url,omitempty"`
    Code          string                     `json:"code,omitempty"`
    ExpiresAt     time.Time                  `json:"expires_at,omitzero"`
    Accepts       Paste                      `json:"accepts,omitempty"`
    Secret        bool                       `json:"secret,omitempty"` // the answer this step asks for is a secret
    ConfirmID     string                     `json:"confirm_id,omitempty"`
    Text          string                     `json:"text,omitempty"`
    Default       *bool                      `json:"default,omitempty"`
    Links         []Link                     `json:"links,omitempty"`
    Fields        []Field                    `json:"fields,omitempty"`
    Yields        Yields                     `json:"yields,omitempty"`
    Error         *Failure                   `json:"error,omitempty"`
    Meta          map[string]json.RawMessage `json:"_meta,omitempty"`
}

// Answer returns what a paste, confirm or form step asked for. Secret is the
// answered step's secret, set by the reader that holds that step; nil means
// not known. String, GoString and LogValue mask Paste and every Fields value
// unless Secret is a non-nil false; MarshalJSON does not mask, since the
// answer must reach the sign-in.
type Answer struct {
    SchemaVersion int                        `json:"schema_version"`
    Seq           int                        `json:"seq"`
    Secret        *bool                      `json:"secret,omitempty"`
    Paste         string                     `json:"paste,omitempty"`
    Confirm       *bool                      `json:"confirm,omitempty"`
    Fields        map[string]string          `json:"fields,omitempty"`
    Meta          map[string]json.RawMessage `json:"_meta,omitempty"`
}

type Failure struct {
    Kind      ErrorKind `json:"kind"`
    Message   string    `json:"message"`
    Retryable bool      `json:"retryable"`
}
```

* **Why these step kinds** (the report's §5.3.1). A device sign-in is
  `show_code`, then `waiting`. A pasteable browser sign-in is `open_url` with
  `accepts`, then `paste`. The ChatGPT plan is a `confirm`, then `open_url`,
  then `paste`; a re-sign-in whose session already holds the consent's
  answer starts at `open_url` (0075-MADR D4). A form of named inputs is `form`. A host-only
  method has no steps: its `Completion` says so.
* **A method whose mechanism is learned at start (O1):** `kind` `oauth`,
  `completion` `at_start`, and no `paste` or `first_paste`. Its describer
  does not know what it yields, whose client it signs in as, or whether the
  result refreshes or can be revoked, and states none of these: `yields` is
  `unknown`, `client_identity` is `unknown`, and `refreshable` and `revocable`
  are false, which a reader of an `at_start` method takes as "not known before
  the sign-in", never as "no". The four stay required members, so no reader
  meets an absent one. What the sign-in yielded is its `done` step's `yields`.
  `unknown` is valid only on an `at_start` method; a method whose completion
  is known states its facts. Its step sequence
  says what it is: after any `form` step that collects the inputs it starts
  with, the first step is `show_code` (a device sign-in, then `waiting`), or
  `open_url` (a browser sign-in: with `accepts`, a paste completes it and a
  `paste` step follows; without, only the host's loopback does, and the
  sequence goes to `waiting`), or `failed`. A client lists such a method as
  "a sign-in whose type is learned on start" and drives it as it would any
  other, rendering each step as it arrives. No built-in descriptor of this
  SDK uses either value: each of its methods knows its mechanism. They exist
  for a consumer that describes another program's methods, as Magic's
  mappers describe kilo's and opencode's OAuth entries, whose catalogs name
  no mechanism.
* **A host-only method's command (O3):** `kind` `host_only`, `completion`
  `host_command`, and `command`, the command a person runs on the host, such
  as one of opencode's `command` or `external` entries. It is display text: a
  relay or a client shows it, with a copy button on the phone, and never runs
  it. It holds no key material (rule 5 of D6). This SDK writes none, since it
  runs no host command (0074-MADR D11).
* **How a step travels is its consumer's.** This module names no transport,
  ACP construct included (Q7 is gobble's; the owner's no-baggage constraint).
  It carries only what any transport needs from the step itself: which steps
  take an answer (`paste`, `confirm` and `form`; every other kind is shown,
  never answered, 0074-MADR D3), and whether that answer is a secret
  (`secret`, below). A transport may have no way of its own to mark an input
  secret; ACP's stable form schema has none (its string formats are `email`,
  `uri`, `date` and `date-time`, acp-go-sdk2 v1.35.0 `types_gen.go:18110-18129`,
  read 2026-11-10 in the module cache; at v1.36.0 the same lines are
  `types_gen.go:18291-18310`, read 2026-11-11), which is why the contract
  carries secrecy itself. The consumers' mappings onto ACP elicitation are recorded
  under More Information, "How the consumers carry the steps".
* **Secrecy is a step's field, and redaction keys on it (K4, Q11).**
  `Step.secret` is true when the answer the step asks for is a secret: every
  `paste` step `x/signin` emits (a pasted callback carries an authorization
  code), and every `form` step with a `secret` field. `Answer.secret` is the
  answered step's, set by the reader that holds the step for that `seq` (the
  engine, gobble's relay, Magic's daemon), never taken on trust from the
  sender; nil means not known. `String`, `GoString` and `LogValue` are
  value-receiver methods, so an `Answer` and a `*Answer` format alike. They
  print `Paste` and every `Fields` value as a mask unless `secret` is a
  non-nil false, so an answer whose step is unknown is masked. `Confirm`,
  `Seq`, `secret` and the field keys are shown. `MarshalJSON` is the
  default: the answer must reach the sign-in through Magic's envelope and
  gobble's input.
* **What is not here.** The provider id (a consumer knows which provider it
  asked), a session or sign-in id, admission, resume, cancel messages, host
  reasons such as Magic's `host_oauth`, app setting names such as gobble's,
  and any agent CLI's format (§5.3.2 of the report; the owner's no-baggage
  constraint). They live in each consumer's envelope.
* **A session's kind** *(added 2026-11-11 by "Amendment 2026-11-11: a
  session-kind type and its labels", below)*: `type SessionKind string`,
  with `SessionChatGPT` "chatgpt", `SessionChatGPTPlan` "chatgpt_plan",
  `SessionGrok` "grok" and `SessionKiloDevice` "kilo_device", an open enum
  with a label for each value in the schema. No struct above gains a member
  of it.

### D3. One error vocabulary (Q15)

`ErrorKind` is a `Failure`'s kind and the one word every repository uses for
a failure's class. Its values, `snake_case`, with the SDK sentinel each maps
from (0074-MADR D8 builds `ErrorKindOf`):

| Value | Meaning | From |
| :--- | :--- | :--- |
| `canceled` | the caller or the user stopped it | `context.Canceled` |
| `timeout` | a deadline passed | `context.DeadlineExceeded` |
| `denied` | the user or the identity provider refused the sign-in | 0074-MADR D8's denied sentinel, beneath `ErrAuthFailure` |
| `expired` | a code or a sign-in window expired | 0074-MADR D8's expired sentinel, beneath `ErrAuthFailure` |
| `auth` | a credential was refused | `ErrAuthFailure` |
| `quota` | quota, credit or balance exhausted | `ErrQuotaExhausted` |
| `rate_limited` | throttled | `ErrRateLimited` |
| `not_permitted` | the account may not do this | `ErrNotPermitted` |
| `workspace` | a workspace is required or not found | `ErrWorkspaceRequired`, `ErrWorkspaceNotFound` |
| `context_overflow` | input longer than the model's window | `ErrContextOverflow` |
| `empty_answer` | an answer with nothing usable | `ErrEmptyAnswer` |
| `incomplete` | a response cut short | `ErrIncomplete` |
| `invalid_request` | a request the service refuses as malformed | `ErrInvalidRequest` |
| `invalid_provider` | an unknown or unsafe provider id | `ErrInvalidProvider` |
| `offline` | no network | `ErrOffline` |
| `unavailable` | the service failed | `ErrProviderUnavailable` |
| `unsupported` | a capability the provider lacks | `ErrUnsupported`, `errors.ErrUnsupported` |
| `other` | none of the above | anything else |

* **Names.** Each constant is `Failure` and its value in camel case:
  `FailureCanceled`, `FailureRateLimited`, … `FailureOther`. R21 asks for a
  short prefix naming the constant's type; the prefix here names the type
  that carries the kind, `Failure`, not the kind's own type, `ErrorKind`.
  gobble-cli's 0051-MADR already uses these names. *Ruled by the owner,
  2026-11-11: the names are kept, as a stated exception to R21. No constant
  is renamed.*
* **It covers turns too,** because Q15 says "everywhere": gobble's ACP kinds
  (C32, and acpext's `ErrorKind`, C40) are this list less `denied`,
  `expired`, `invalid_request` and `invalid_provider`, with three words
  hyphenated (`rate-limited`, `context-overflow`, `not-permitted`) and
  `empty_reply` for `empty_answer` (C38). gobble aligns them in its own
  record, gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D6 (proposed): `llm.Kind` and acpext's `ErrorKind` take these spellings,
  and acpext's `ErrorKind` holds all eighteen at `acpext/v0.5.0`. Its PLAN's
  P3 step 4 copies this table into an acpext test, since acpext imports no
  SDK module, so a word added here reaches acpext only by gobble's next
  record. This module only names the words.
* **Order.** A consumer that maps an error tests the most specific first, in
  the table's order, as gobble's own maps do (C32 to C34).

### D4. The JSON Schema, generated and committed

* **Generator.** `authcontract/internal/schemagen`, standard library only,
  reads the types by `reflect` and the constants of each enum type by
  `go/parser` over the package's source, and writes
  `authcontract/authcontract.schema.json` (JSON Schema draft 2020-12). The
  package carries `//go:generate go run ./internal/schemagen -output authcontract.schema.json`,
  so `make generate-check` holds the file to the types (C21).
* **Shape, for Dart generation (Q9).**
  * One object per Go struct, in `$defs`, referenced with `$ref`. No `oneOf`,
    `anyOf`, `allOf`, `not` or conditional keywords anywhere: a step is one
    flat object whose `kind` says which fields matter.
  * Each enum type is a `$defs` entry `{"type": "string", "enum": [...]}`
    listing every constant of its type, with the annotation
    `"x-open-enum": true`. A producer emits only listed values. A reader
    keeps an unknown one (D5). How a consumer's generator renders an open
    enum is that consumer's to build and gate (Q9); Magic's renders each as a
    Dart extension type over `String` with its known values and `isKnown`
    (magic-cli-remote
    `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
    D9).
  * `required` lists the fields without `omitempty` or `omitzero`. Objects
    allow additional properties, so a newer producer's field does not break
    an older reader's validation.
  * **When a member is on the wire, stated in the schema** (Q9: a generator
    reads the schema, never the Go tags). An optional member is one of two
    kinds, and the schema says which:
    * a **pointer** member (`Step.default`, `Step.error`, `Answer.secret`,
      `Answer.confirm`) carries the annotation `"x-nullable": true`. It is
      absent when nil and present whenever it is set, `false` included: an
      `Answer` whose `confirm` is `false` is the "no" to a consent, and a
      writer that drops it loses the answer;
    * every other optional member (`omitempty` or `omitzero` on a value type)
      carries no annotation, and is absent when it holds its type's zero
      value: `Step.secret` false, an empty string, an empty list or map, a
      zero `expires_at`.

    A generator therefore writes an annotated member whenever it is not null,
    and an unannotated optional member only when it is not its zero value.
    A required member is always written.
  * `_meta` is an object of any values; `expires_at` is a string with
    `"format": "date-time"`, written by Go's `time.Time` encoding (RFC 3339,
    fractional seconds only when non-zero). A consumer that must re-encode a
    vector byte for byte keeps the string as it arrived rather than
    re-formatting a parsed time.
  * `Step.secret` is a boolean, absent meaning false. `Answer.secret` is an
    optional boolean, absent meaning not known; a Dart reader maps it to a
    nullable `bool` and masks when it is not `false`, as the Go type does.
  * **Keywords.** The schema uses exactly these: `$schema`, `$id`, `$defs`,
    `$ref`, `type`, `properties`, `required`, `enum`, `items`,
    `additionalProperties` and `format` (`date-time` only), and the two
    annotations `x-open-enum` and `x-nullable`. It emits no `description` or
    `title`. A change to this list is a change to D4, so a consumer's
    keyword check can be exact. *(2026-11-11: "Amendment 2026-11-11: a
    session-kind type and its labels", below, adds a third annotation,
    `x-enum-labels`, carried by `SessionKind`'s entry only.)*
* **The schema states `SchemaVersion`** as its `$id`'s last segment
  (`.../authcontract/schema/v1`), read from the constant by the generator. It
  does not state the module's version: no constant holds it (it is a git
  tag), and writing it would make the committed schema, and D7's comparison
  of it, change at every tag.

### D5. Golden vectors

* **Where.** `authcontract/testdata/vectors/`, one JSON file per case, named
  for its type and case (`step-open-url-paste.json`, `answer-paste.json`,
  `method-device-code.json`, `failure-denied.json`, and so on): at least one
  per step kind, per method kind, per answer kind, and per failure kind;
  `method-oauth-at-start.json` (O1: `yields` and `client_identity`
  `unknown`, `refreshable` and `revocable` false); `method-host-only-command.json` (O3);
  a `paste` step with `secret` true and no `expires_at`; answers with
  `secret` true, false and absent; and an answer whose `confirm` is `false`
  (`answer-confirm-false.json`), which a writer must not drop (D4).
  *(2026-11-11: "Amendment 2026-11-11: a session-kind type and its labels",
  below, adds one vector per `SessionKind` value and a forward vector with
  an unknown one.)*
* **The module's tests:**
  * every vector decodes into its type, and re-encodes to the same bytes
    after `json.Compact`;
  * every vector but those under `testdata/vectors/forward/` validates
    against the committed schema, by a small validator for exactly the
    keywords D4 lists (`internal/schemacheck`, standard library only);
  * `forward/` holds a newer producer's output. Each decodes without error
    and drops an unknown member. A vector with an unknown enum value keeps
    it, and fails strict validation, because strict validation checks enum
    membership, which the test also asserts, so the two rules stay apart. A
    vector whose only novelty is an unknown member validates, since objects
    allow additional properties (D4), and the test asserts that too.
* **For consumers.** gobble-cli and magic-cli-remote decode and re-encode the
  same files in their own tests, read from the module's tagged source, and
  Magic's Dart tests decode them too (Q9). Their records say how. The vectors
  are written in Go's encoding, which escapes `<`, `>` and `&` in strings, so
  a Go consumer can compare bytes after `json.Compact`, and a consumer in
  another language compares JSON values, every member present or absent as in
  the vector (magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  Confirmation 2).

### D6. The evolution rules

Written into the package doc and into `authcontract/README.md`:

1. **Additive only within a major version, v0 included.** No rename, no
   removal, no change of a field's type or JSON name, and no new required
   field. A new field is optional, and its absence means the old behaviour.
2. **Enums are open.** A reader keeps an unknown value and renders it
   generically: an unknown step kind shows its `text` and offers cancel; an
   unknown method kind or completion is listed and not driven. A producer
   emits only the values of the schema of its version. `oauth` and
   `at_start` are known values, not unknown ones: a method that carries them
   is driven, and its first steps say its mechanism (D2).
3. **`schema_version` and features.** Every `Method`, `Step` and `Answer`
   carries `schema_version`. A reader refuses an object whose
   `schema_version` is higher than its own `SchemaVersion`. A `Method` names
   in `features` the additive capabilities its sign-in uses (`consent`,
   `form`, `paste`), so a client that cannot render one says so before it
   starts.
4. **`_meta` is the extension namespace,** keyed by owner (`gobble`, `magic`,
   and so on). Nothing in `_meta` is required to complete a sign-in. This SDK
   writes none.
5. **Secrets.** A `Method` and a `Step` carry no key material (R34); a
   `Method`'s `command` is display text, never run by a relay or a client. A
   step marks the answer it asks for as a secret with `secret`. An answer to
   such a step, and an answer whose step a reader does not hold, is a secret:
   never logged, never stored by a relay (Q11). A reader sets
   `Answer.secret` from the step it holds, never from the sender's copy.
6. **A break** needs a record, a new major version and a new
   `SchemaVersion`.

### D7. A stricter gate for the module: `make contract-check`

`scripts/check_contract.py`, Python on the standard library as every gate
(0035-MADR), run by CI and by `tools/gate.py`, fails when:

* **The module leaves the standard library:** `go -C authcontract list -deps
  -test ./...` names a package that is neither standard nor under
  `github.com/maccavelli/go-llmprovider-sdk/authcontract`, or `go -C
  authcontract mod edit -json` lists a requirement.
* **Its tests fail:** `go -C authcontract vet ./...`, `go -C authcontract
  test -race -count=1 ./...`.
* **Its coverage falls** below 80 % for any package with statements (R47;
  `coverage-check` cannot see the module, C13), the schema generator's
  `package main` included: it is measured by its own test, which generates
  into a temporary file and compares it with the committed schema. No package
  is exempt.
* **Its API breaks.** apidiff, at `scripts/check_api.py`'s pin, compares the
  module with its latest `authcontract/v*` tag, and **any** incompatible
  change fails, even at v0, where Go's own rules would allow it. This is
  stricter than `make api-check`, which tolerates changes under
  `llmprovider/x/` (R48).
* **Its JSON breaks.** The committed schema is compared with the schema at
  that tag: a `$defs` entry, a property or an enum value removed, a type
  changed, or a property newly required, fails. This catches a changed JSON
  name, which apidiff may not report (whether apidiff reports a changed
  struct tag is unverified; the schema comparison does not depend on it).
* **Before the first tag** there is nothing to compare; it says so and
  passes, as `api-check` did before `v1.0.0`.

### D8. The repository's other gates learn the module

* **R1 and 0015-MADR D2 are amended:** a second module is allowed when it is
  a wire contract that must version apart from the SDK, with no dependency of
  its own. `authcontract` is the one such module.
* **`scripts/go-precheck.sh`** runs `go vet`, `go test -race`, `golangci-lint`
  and `go mod tidy -diff` in each nested module that holds a given file,
  with `go -C`, and leaves the root's run as it is. It learns this before
  the module's first Go file exists, so no `authcontract` file is ever
  staged that the pre-add check could not vet (AGENTS.md, "Pre-add checks":
  a file that fails is not committed).
* **`make lint`, `make vuln` and `make test`** run in the module too.
* **CI** runs on `authcontract/v*` tags as well as `v*`, tests the module on
  all three operating systems with `-race` on Linux, runs its `go mod tidy
  -diff` and govulncheck on Linux beside the root's, and runs
  `contract-check` with the other gates. The release job stays for `v*` tags
  only: a contract tag gets no GitHub Release, and R53's six newest are the
  SDK's.
* **`make changelog-check`** also holds `authcontract/CHANGELOG.md` to the
  `authcontract/v*` tags.
* **`scripts/test_gates.py`'s workflow contract** places `contract-check` in
  `CODE_GATES`, and gains an assertion that `ci.yml` runs every gate in
  `CODE_GATES`. Today it asserts that only of the Markdown gates (C44), so a
  code gate dropped from `ci.yml` would pass it unseen.
* **`docs/architecture.md`** describes the module in prose. Its package table
  and R2's table stay the root module's, because `tables-check` reads `go
  list ./...` from the root (C12); 0074-MADR extends that check when the
  root first imports the module.

### Consequences

* Good, because one vocabulary replaces the three repositories' own, and a fact such as "this
  method completes by a pasted URL" is decided once, in the SDK, and read by
  every consumer (F1, F18).
* Good, because the module has no dependency: gobble, acpext if it ever
  chooses, and Magic can import it without pulling HTTP clients, flows or
  `golang.org/x/term`.
* Good, because the contract's rules are written and gated: apidiff and the
  schema comparison refuse a break even at v0.
* Good, because the schema is designed for Dart generation, which
  magic-cli-remote
  `docs/decisions/0236-MADR-re-pin-codex-to-0-167-0-and-generate-more-of-its-contract.md`
  could not do from codex's schema (its option C).
* Neutral, because the phone does not need the evolution rules: its daemon
  and app ship together. gobble and other consumers do.
* Neutral, because `oauth`, `at_start` and `command` describe methods of
  programs this SDK does not drive; no SDK descriptor sets them, so they are
  held by vectors and by their consumer's tests, not by the SDK's own.
* Good, because a step says when its answer is a secret, so the engine,
  gobble's relay, Magic's daemon and the phone redact the same answers, and
  an answer whose step is unknown is masked.
* Bad, because every gate that assumed one module has to learn a second
  (D8), and a contract change now needs a tag before the SDK, gobble or
  Magic can use it.
* Bad, because a wrong field is hard to remove: the rules allow no removal
  even at v0. A deprecated field stays until a major version.
* Bad, because two error vocabularies exist until gobble and Magic adopt
  this one (Q15's work is theirs).

### Confirmation

* `authcontract/go.mod` names no requirement; `make contract-check` passes,
  and was seen to fail on each planted break the PLAN names, on a scratch
  clone.
* `make generate-check` passes, and fails on a hand edit of the schema.
* The vectors' tests pass: every forward vector is accepted by decoding;
  one with an unknown enum value is rejected by strict validation; one whose
  only novelty is an unknown member validates.
* The schema's pointer members, and only they, carry `x-nullable`, and
  `answer-confirm-false.json` re-encodes with its `"confirm":false`.
* `make docs-check` passes for the module (R50 to R52 reach it, C20).
* `tools/claims.py` passes on this record.
* The tag `authcontract/v0.1.0` exists on the owner's word, and
  `go list -m github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0`
  resolves through the module proxy.

## Pros and Cons of the Options

### A. Mirror

* Good, because each repository's wire moves only when it decides, and no new
  dependency or tag exists.
* Bad, because it keeps the duplication the owner wants gone: every SDK
  change is mapped by hand in two repositories, and the Dart models stay
  hand-written.
* Bad, because the drift the report found (F1, F2, F3, F18) continues.

### B. Import the full SDK

* Good, because there is one set of types and direct access to the engine.
* Bad, because Magic's daemon, which runs no SDK sign-in (0245 report §2.5),
  would carry flows and HTTP clients to read a few structs.
* Bad, because the wire would follow the SDK's v1 API, whose rule is Go's
  compatibility, not a JSON promise: a field's JSON name is not covered.
* Bad, because it pulls the SDK's flow code toward 0016-MADR D10's line.

### C. A nested module at the root, `authcontract/`

* Good, because it has no dependency, and its own cadence and promise.
* Good, because gobble-cli's `acpext` is a working precedent, with the same
  tag scheme and the same "no replace, no go.work" rule.
* Good, because a root-level directory is plainly not a package of the root
  module, so no `./llmprovider/...` pattern silently skips it.
* Neutral, because its tags sort apart from the SDK's in `git tag`.
* Bad, because the gates need extending (D8), and the release order gains a
  step: the contract's tag before the SDK minor that uses it.

### D. Under `llmprovider/`

* Good, because it sits beside the code that produces it.
* Bad, because `go test ./llmprovider/...` and every other pattern over
  `llmprovider/` would silently stop at it, a trap for every existing gate and
  habit.
* Bad, because 0015-MADR D2 gives `llmprovider/*` to packages of the one
  module; a module there reads as one of them.
* Bad, because its tag, `llmprovider/authcontract/v0.1.0`, is longer for no
  gain.

### E. A package inside the root module

* Good, because no second module, no second tag and no gate changes.
* Good, because magic-cli-remote's `go.mod` already requires this SDK, at
  `v1.29.0` since its `912b9aa1` and `v1.28.0` before (C43), so the
  requirement would not be new to it.
* Bad, because the contract would be under the SDK's v1 promise from its
  first commit, with no v0 period to correct a field (against Q1).
* Bad, because every contract change would wait for an SDK release, and
  every SDK release would move the contract's version for consumers that
  only read the types.
* Bad, because it is not what the owner chose (Q8: "nested in the SDK repo
  with its own tags").

### F. A repository of its own

* Good, because it is a plain module with plain tags.
* Bad, because the SDK, which produces the contract, would need a second
  repository's release for every fact it adds: two repositories, two sets of
  records, two CI configurations, for a package of types.
* Bad, because the owner chose against it (Q1 (c) was not chosen).

## More Information

### Records this decision amends, applied on acceptance

These notes are listed here and applied by the PLAN's Phase 0, step 3. No
existing record is edited before this record is accepted.

* **`0015-MADR-canonical-sdk-api-and-module-layout.md`, D2,** whose bullet
  reads "**No second module.** A module is split out only when a dependency
  justifies one, and that takes its own MADR" (C6). An amendment adds: "A
  second module is also allowed for a wire contract that must version apart
  from the SDK and has no dependency of its own: `authcontract/` (0073-MADR
  D1, D8)."
* **The standards guide's R1** (C5), the same, citing 0015 D2 and this
  record. R2's table is unchanged here (D8).
* **AGENTS.md "Dependencies"** (C7) gains a sentence: the nested module
  `authcontract` requires nothing, and the root may require it at a tag
  (0074-MADR adds the requirement with its first import).

### Records in other repositories this bears on

None is changed by this repository. Each is the named repository's to amend.

* magic-cli-remote `docs/decisions/0185-MADR-migrating-to-go-llmprovider-sdk-is-out-of-scope.md`:
  superseded in part, for auth types (the report's M1), by magic-cli-remote's
  proposed 0246-MADR, D1.
* magic-cli-remote `docs/reports/0202-REPORT-go-llmprovider-sdk-provider-auth-adoption-assessment.md`:
  its objection that "the phone's contract would then change on the SDK's
  release cadence" is answered by D1's own tags and D6's rules.
* gobble-cli `docs/decisions/0042-MADR-acpext-extension-contract-module.md`:
  unchanged. The owner answered gobble's QC on 2026-11-10 as recommended:
  acpext publishes the provider list only, with facts from this contract,
  and stays standard library only (gobble-cli
  `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D7).
* gobble-cli `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  (proposed, untracked in gobble-cli on 2026-11-11): D4, the mapping of
  these steps onto ACP elicitation (below); D6, D3's words in `llm.Kind` and
  in acpext's `ErrorKind` and `RetryCause` at `acpext/v0.5.0`; and D1, which
  amends gobble's archtest rule 4. That rule refuses any import under this
  SDK's module path outside `llm/provider` (C41), and the contract's path
  shares the prefix, so gobble cannot import the module in `acpserver` or
  `internal/cli` until D1 lands.
* magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  (proposed, untracked in magic-cli-remote on 2026-11-11):
  embeds these types in Magic's envelope, builds kilo's and opencode's
  methods with `oauth`, `at_start` and `command`, generates its Dart models
  from the schema (its D9), and advertises form-only elicitation (its O4,
  answered by the owner on 2026-11-10). Its D1 amends Magic's boundary test,
  `TestSDKIsImportedOnlyByGrok`, which fails any import with this SDK's
  module path as its prefix outside Magic's grok package (C42), the
  contract's included; neither the 0245 report nor this record's first
  draft named that gate.
* magic-cli-remote
  `docs/decisions/0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md`
  (proposed, untracked): replays a pasted callback to an agent's own
  loopback listener where the agent has no paste channel. It relies on D2's
  `Step.secret` on every `paste` step and on `Answer`'s masking, and on D6
  rule 5, so that the paste is never logged or kept. It needs nothing more
  of this module: the replay, its checks and its port admission are Magic's.

### How the consumers carry the steps (not part of this decision)

Recorded here so a reader of the contract can see where its steps go; each
mapping is its repository's decision, and this module holds either way.

* **gobble over ACP** (gobble-cli
  `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D4, the owner's Q7). gobble chooses per client capability, because
  acp-go-sdk2 v1.35.0 refuses to send a form to a client without
  `elicitation.form` and a URL elicitation to a client without
  `elicitation.url`, as a `*CapabilityError`, before sending
  (`types_gen.go:25493-25516`, read 2026-11-10 in the module cache; at
  v1.36.0 the same function is at `types_gen.go:25802-25825`, read
  2026-11-11):
  * with `elicitation.url`, `open_url` and `show_code` travel in URL mode,
    the code and its expiry in the message;
  * with `elicitation.form` only, `open_url` and `show_code` each travel as a
    form with no fields whose message holds the URL, the code and the expiry;
  * either way, an `open_url` or `show_code` elicitation is acknowledged, not
    answered: its `accept` builds no `Answer` and the sign-in runs on, and its
    `decline` or `cancel` cancels the sign-in. An `open_url` that accepts a
    paste is followed by the `paste` step as a form of its own, one required
    string, and that form's `accept` is the `Answer` for the `paste` step's
    `seq`, as this module's answer rule requires (0074-MADR D3);
  * `confirm` and `form` travel as forms (one boolean, the fields);
    `waiting` is not sent; `done` and `failed` are the `authenticate` call's
    result;
  * every elicitation also carries the step itself, the `Step` of this
    module, in `_meta.gobble.step`, so a client that reads the contract
    renders the step natively, and learns from `secret` that the answer is a
    secret, whatever mode carried it.
* **Magic** (magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  D8). The owner answered its O4 on 2026-11-10: form only. Magic advertises
  `elicitation.form` and not `elicitation.url`, renders each step from
  `_meta.gobble.step`, acknowledges an `open_url` or `show_code` form with an
  empty `accept` once it has sent the step to the phone, and answers `paste`,
  `confirm` and `form` with the phone's `Answer`.

### Questions answered by the owner (2026-11-10)

The Magic draft put these two to this module; the owner answered both on
2026-11-10, O1 as recommended and O3 against the recommendation. The options
are kept as they were put (magic-cli-remote
`docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`):

* **O1. "Decided at start" in the contract.** (a) *recommended, chosen:* add
  `KindOAuth "oauth"` and `CompletionAtStart "at_start"` before
  `authcontract/v0.1.0`; (b) Magic lists such methods as `host_only` until
  the agents declare their mechanism, which hides kilo's and opencode's
  device sign-ins; (c) Magic lists them as `device_code` and fails a browser
  one at start, which states a fact Magic does not have.
* **O3. The host command.** (a) *recommended:* `Method.detail` holds it now,
  and a contract field is asked for later; (b) *chosen:* a `command` field
  before `v0.1.0`; (c) show "Finish on the host" with no command.

### Where the report and the audit differ from the code

* **Gates the report does not name.** The report's §5.3.3 names `make
  api-check` as the gate that misses a nested module. Seven more assume one
  module (C10 to C19), and one, `scripts/go-precheck.sh`, would fail a
  nested module's file outright (C14, C15). D7 and D8 handle them.
* **The illustrative types.** The report's `Field` has no JSON tags and its
  `ErrorKind` list has no `auth`, `quota` or turn kinds. Q15's "everywhere"
  needs them (D3).
* **Of the report's questions, four answers went against its
  recommendation** (Q6, Q7, Q14 and Q19), not three. Q17, first answered
  against it, was asked again and answered as recommended ("wire them after
  all"). None of them is this record's to carry, but Q7 is why a consumer
  maps the steps onto ACP elicitation.
* **An earlier draft of D2 mapped `open_url` and `show_code` onto URL mode,
  and said no step needs a construct form mode lacks.** Both were wrong. A
  client that advertises form only cannot receive URL mode
  (`types_gen.go:25493-25516`), and Magic is such a client; form mode has no
  secret format (`types_gen.go:18110-18129`). D2 now names no transport and
  carries `secret` on the step; gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D4
  holds the mapping, summarised above.

### Revisions after the reviews of 2026-11-10

Two independent reviews of this draft and its consumers' drafts found
defects; each fix below was checked against the code first.

* D2: `yields` `unknown` and `client_identity` `unknown` added, so an
  `at_start` method (O1) states no fact its describer lacks; the ACP mapping
  moved here from D2, and an `open_url` or `show_code` elicitation made an
  acknowledgement, not an answer, matching 0074-MADR D3's answer rule.
* D4: the `x-nullable` annotation, so a generator reading the schema alone
  writes `confirm: false`; `required` excludes `omitzero`; the keyword list
  made exact; `x-module-version` dropped, since no constant holds the
  module's version; Magic's open-enum mapping stated as Magic's record has it.
* D5: forward vectors split by what strict validation can refuse; vectors for
  a `paste` with no `expires_at` and a `confirm` of `false`.
* D7: the dependency check admits the module's own packages; the generator's
  coverage is measured, not exempted.
* D8 and the PLAN: the pre-add check learns the nested module before the
  module's first Go file.

### Revisions of 2026-11-11

A review against the code and the sibling repositories as they stood on
2026-11-11. The chosen option and D1 to D7 are unchanged; D8 gains two gate
details, below. Every other fix states a fact as it is.

* 2026-11-11, Decision Drivers: the owner's answers were attributed to the
  report's §11, "each answered by the owner choosing among three options".
  §11 holds the questions and their options, two to four each, and no
  answer. The bullet now says where the answers are recorded, gives each
  chosen option, and says which quoted words no other record carries.
* 2026-11-11, Decision Drivers, Q15: "acpext/llm `ErrorKind`" is the owner's
  shorthand for acpext's `ErrorKind` (C40) and gobble's `llm.Kind` (C32);
  there is no package `acpext/llm`. Said so where it is quoted.
* 2026-11-11, D1: `go.work` and `go.work.sum` were said to be ignored by
  git. `.gitignore` names neither at `773b84c`; the PLAN's Phase 1a adds
  them. D1 now says they are added.
* 2026-11-11, D3: "R21's prefix rule" did not hold, since `Failure` names
  the type that carries the kind, not `ErrorKind`. The bullet now says so.
  "gobble aligns them in its own record" now names that record, gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D6, and its PLAN's copy of D3's table into an acpext test.
* 2026-11-11, D8: CI also runs the module's `go mod tidy -diff` and
  govulncheck, which its root-only steps do not reach. The workflow
  contract gains an assertion that `ci.yml` runs every code gate: it checks
  only the Markdown gates today (C44), so the PLAN's planted removal of
  `contract-check` from `ci.yml` would not have failed it.
* 2026-11-11, option E: Magic's requirement on this SDK is committed in its
  `go.mod` (C43), not only in a working tree.
* 2026-11-11, More Information: the gate in each consumer that refuses the
  contract's path today, gobble's archtest rule 4 (C41) and Magic's
  `TestSDKIsImportedOnlyByGrok` (C42), each amended by that consumer's
  record; magic-cli-remote's 0247-MADR, which relies on D2 and D6 rule 5;
  and 0185's superseding record named.
* 2026-11-11, the claims marker: gobble-cli is read at `17d1605`, its
  origin/main on 2026-11-11, at which every gobble-cli file this record
  cites is as at `d6c10a5`; magic-cli-remote is read at `eefb7c25`, its
  origin/master. The `consulted` line says so.
* 2026-11-11, D1: the module's `go` line named `go 1.27.3`. It now follows
  R54's floor as the root states it when the module is created. govulncheck
  v1.8.0, run on 2026-11-11, reports GO-2026-6859 (`net/http`, "HTTP/1
  server crash due to MaxBytesReader header race", fixed in Go 1.27.4)
  against this repository with Go 1.27.3, reachable from
  `auth.oauthCallbackHandler` and `localonly.Start`, and nothing with Go
  1.27.4. magic-cli-remote moved to Go 1.27.4 for it (its ae13109a).
* 2026-11-11, D2, "Why these step kinds": the ChatGPT plan was said to be
  always `confirm`, `open_url`, `paste`. 0075-MADR D4 shows no `confirm` on a
  re-sign-in whose session already holds the answer; the sentence says so.

The owner's rulings and approvals of 2026-11-11, written in after the review
above. No decision of D1 to D8 changes.

* 2026-11-11, D3: the `Failure*` names of `ErrorKind`'s eighteen constants
  are kept. The owner ruled them a stated exception to R21, and no constant
  is renamed. The PLAN said "The constant names follow R21"; it now names
  the exception.
* 2026-11-11, D1: "the floor rises only by a record of its own" named no
  record. It is
  [0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md),
  accepted the same day, which raises the root's floor to `go 1.27.4` for
  GO-2026-6859 before any of 0076's code. The module's `go` line is
  therefore `go 1.27.4`, and the PLAN's Phase 1a says so.
* 2026-11-11, the PLAN's order across repositories: 0077's PLAN is its first
  step, before 0076's `v1.30.0`. The owner also ruled that gobble-cli
  `v0.6.0` is the release that must pin the same acp-go-sdk2 tag as
  magic-cli-remote; the PLAN's step 3 notes it.
* 2026-11-11, the acp-go-sdk2 citations: the owner approved moving
  gobble-cli and magic-cli-remote together from acp-go-sdk2 v1.35.0 to
  v1.36.0, before gobble's `v0.6.0`, by a record of each repository that was
  still being written that day. This record's two citations hold at v1.35.0
  as written. At v1.36.0 the text of both is unchanged and its lines have
  moved: `type StringFormat` and its four constants are at
  `types_gen.go:18291-18310` (the type at `:18291`), and
  `(*CreateElicitationRequest).gates` is at `types_gen.go:25802-25825`,
  each read on 2026-11-11 with `git show v1.36.0:types_gen.go` in
  acp-go-sdk2's repository. D2 and "How the consumers carry the steps" now
  give both; "Where the report and the audit differ" cites the v1.35.0
  lines, as it did.

The owner's decision of 2026-11-12, recorded in gobble-cli, written in
after. No decision of D1 to D8 changes, and the two bullets above are not
rewritten.

* 2026-11-12, gobble-cli's release numbers: the owner decided that
  gobble-cli releases `v0.6.0` now, from its main as it stands, without the
  sign-in redesign on this contract (gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  and `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`), and
  that the redesign, which these records named `v0.6.0`, is released as
  `v0.7.0` (gobble-cli
  `docs/decisions/0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`,
  D1 and D2). The two bullets above that name gobble's `v0.6.0` stay true of
  that release: it is cut from a main whose `go.mod` pins acp-go-sdk2
  `v1.36.0`, as magic-cli-remote's does, and both moved to that tag before
  it (0055-MADR F8). gobble's release on the contract is `v0.7.0`, which must
  meet the same lockstep when it is cut (its D2). The PLAN's "Order across
  repositories", step 3, carries the same note.

### Not verified

* Go's exact message for `go vet ./authcontract` run from the root. The
  PLAN's Phase 1a records it from the planted file (corrected 2026-11-11;
  it said Phase 2).
* Whether apidiff reports a changed struct tag. D7's schema comparison holds
  the JSON names either way.
* Whether Magic's Dart generator, a repository script that does not exist
  yet, honours `x-open-enum` and `x-nullable` as D4 states them. Magic's
  record decides its generator and its drift gate (Q9), and its PLAN's tests
  hold it.

### Related

* PLAN: [0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md).
* The engine that produces these types:
  [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md).
* The plan's consent:
  [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md).
* The defects that land first:
  [0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md).

## Amendment 2026-11-11: a session-kind type and its labels

The owner's ruling of 2026-11-11, relayed by the cross-repository
coordinator, made after this record's acceptance and before
`authcontract/v0.1.0` is tagged. It adds to D2, D4 and D5, each of which
carries a dated pointer here. It changes no decision of D1, D3, D6, D7 or
D8, and nothing above is rewritten.

### The ruling

The contract gains a session-kind type with exactly the four values of the
SDK's `auth.SessionKind`
([0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
D5), `chatgpt`, `chatgpt_plan`, `grok` and `kilo_device`, and a human label
for each, before `authcontract/v0.1.0` is tagged. magic-cli-remote
generates its phone's Dart models from `authcontract.schema.json` alone
(Q9; magic-cli-remote
`docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
D9) and shows a stored session's kind by its label (its D13, which cites
this amendment), so the values and the labels must both be in the
generated schema, not only in Go.

### The type (D2)

```go
// SessionKind is the kind of a stored sign-in session, with the strings of
// the SDK's auth.SessionKind. It is a value, safe for concurrent use while
// no goroutine changes it.
type SessionKind string // SessionChatGPT "chatgpt", SessionChatGPTPlan "chatgpt_plan",
                        // SessionGrok "grok", SessionKiloDevice "kilo_device"
```

* **Data only,** like D2's other enums: a named string and four constants,
  in the standard library's terms (D1). No enum of the module has a method
  or a label lookup, so this one has neither. Each constant's name is R21's
  short prefix, `Session`, and its value in camel case, which are also
  `auth`'s names, so a reader of either package reads the same words.
* **Open, like the others** (D4; D6 rule 2). Its `$defs` entry carries
  `"x-open-enum": true`. A reader keeps a value it does not know and shows
  it as it came, with no label; a producer emits only listed values.
* **No struct gains a member.** The type is named by its consumers'
  envelopes: gobble-cli's `auth status --json` member `sessionKind`, whose
  values these are (gobble-cli
  `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D8), and
  magic-cli-remote's auth block, whose daemon holds the kind as this type
  and whose phone shows its label (its 0246-MADR D13). D2's "What is not
  here" holds: a session's kind is not a session id, and no provider id is
  added.
* **The labels.** The wording is this amendment's; the owner may change any
  label before the tag, by a dated note here.

  | Value | Label |
  | :--- | :--- |
  | `chatgpt` | ChatGPT |
  | `chatgpt_plan` | ChatGPT plan |
  | `grok` | Grok |
  | `kilo_device` | Kilo |

  *(2026-11-11, later: the owner chose these short names. The amendment as
  first written proposed "ChatGPT sign-in", "ChatGPT plan for this app",
  "xAI sign-in" and "Kilo device sign-in", each following the label the
  SDK's descriptors give the method that makes such a session. A short name
  reads cleanly where the phone shows it, after "Stored sign-in:".)* A label
  is display text: a reader never matches on it, and a later tag may reword
  it. A value is never renamed (D6 rule 1).

### How the labels reach the schema (D4)

* **A third annotation, `x-enum-labels`,** on an enum's `$defs` entry: an
  object whose members are exactly that entry's `enum` values, each with
  its label, a non-empty string. At `v0.1.0` only `SessionKind`'s entry
  carries it; an enum with no labels carries none.
* **D4's keyword list gains it.** The schema uses exactly `$schema`, `$id`,
  `$defs`, `$ref`, `type`, `properties`, `required`, `enum`, `items`,
  `additionalProperties` and `format` (`date-time` only), and the three
  annotations `x-open-enum`, `x-nullable` and `x-enum-labels`. It still
  emits no `description` or `title`, and no `oneOf`. D4 says a change to the
  list is a change to D4; this is that change, made in the open, so a
  consumer's exact keyword check (magic-cli-remote's 0246-PLAN P1) gains
  the one annotation and nothing else.

```json
"SessionKind": {
  "type": "string",
  "enum": ["chatgpt", "chatgpt_plan", "grok", "kilo_device"],
  "x-open-enum": true,
  "x-enum-labels": {
    "chatgpt": "ChatGPT",
    "chatgpt_plan": "ChatGPT plan",
    "grok": "Grok",
    "kilo_device": "Kilo"
  }
}
```

* **Why this form.** The schema gives no other enum per-value text, so it
  has no form of its own to reuse. The nearest are its two `x-`
  annotations, and this one is a third of the same kind: ignored by
  validation, read by a generator. The alternatives each fail a constraint:
  * JSON Schema's own way to title each value, a `oneOf` of `{"const",
    "title"}` entries, is refused by D4 twice (no `oneOf`, no `title`), and
    asks a Dart generator for a union;
  * a `title` or `description` on the entry gives one string for the type,
    not one per value, and D4 refuses both;
  * an array in `enum`'s order, the form of some generators' `x-enumNames`,
    goes wrong silently when a value is inserted, where an object keyed by
    value is checked member by member;
  * labels held in Go only would not reach the phone, whose models come
    from the schema alone.
* **Where the generator gets them.** `internal/schemagen` holds one table,
  value to label, for `SessionKind`, and writes it as the entry's
  `x-enum-labels`. The package `authcontract` gains no exported name for
  the labels: R55 asks each exported name for the call site that needs it,
  and no Go consumer shows a label (gobble writes the value; Magic's label
  is the phone's, read from the schema). An unexported table in
  `authcontract` would be read by nothing in its own package, and a
  constant's comment is documentation, not data. The generator's table is
  the smallest form that puts the labels in the schema, and the module's
  tests hold it to the type's constants (0073-PLAN Phase 3).
* **The entry is written though no struct refers to it,** as D4 already
  gives every enum type the generator finds by `go/parser` an entry. A
  consumer's generator reaches it by its name.
* **`internal/schemacheck`** accepts `x-enum-labels` and ignores it for
  validation, as it does `x-open-enum` and `x-nullable`.
* **D7 is unchanged.** Its schema comparison already fails on an enum value
  removed. A reworded label is not a break. A value with no label is
  refused by the module's own test.

### The vectors (D5)

* `session-kind-chatgpt.json`, `session-kind-chatgpt-plan.json`,
  `session-kind-grok.json` and `session-kind-kilo-device.json`, each the
  bare JSON string of its value: each decodes into `SessionKind`,
  re-encodes to the same bytes, and validates against the schema's
  `SessionKind` entry.
* `forward/session-kind-unknown.json`, the string `"a_future_session_kind"`:
  it decodes with the value kept, re-encodes unchanged, and fails strict
  validation, as D5's other unknown enum values do. As a bare string it
  has no `_meta.example` entry, which D5's other forward vectors carry.

### The root module's copy (0074)

`auth.SessionKind` carries the same four strings, and this amendment changes
nothing of its v1 shape (0074-MADR D5). Once the root requires the contract
(0074-PLAN Phase 4), a test in `llmprovider/x/signin`, the package that
imports both `auth` and `authcontract`, pins the two value sets equal:
`TestSessionKindsMatchContract_0074`. It holds this amendment's table of
values, pairs each `auth` constant with the `authcontract` constant of the
same name, asserts each pair's strings equal the row's value, and asserts
that the `SessionKind` constants `go/parser` finds in `llmprovider/auth`
are exactly the table's. The contract's own test holds the contract's
constants to the same table (0073-PLAN Phase 1b). This is how D3's table is
held in acpext: a copy of the table in each test, so that a value added on
one side fails until a record adds it on the other. 0074-PLAN Phase 4 adds
the test, with a dated note.

### Against D1, D2 and D4 as accepted

No conflict was found. D1: the type is standard library only and adds no
requirement. D2: it is one more open enum of named-string form, and no
member, field name or JSON name of D2 changes. D4: its shape rules hold (no
`oneOf`, no `title` or `description`, string enums with `x-open-enum`); its
keyword list gains one annotation, which D4 itself provides for ("a change
to this list is a change to D4").

### Not verified

* Whether Magic's Dart generator, which does not exist yet, renders
  `x-enum-labels` as stated here. Magic's record decides its generator
  (Q9); its 0246-PLAN P1 reads the tagged schema's keyword set and compares
  it with D4's list, which now holds this annotation.
