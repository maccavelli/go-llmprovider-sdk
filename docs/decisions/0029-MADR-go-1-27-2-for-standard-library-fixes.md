---
status: accepted
date: 2026-10-08
decision-makers: repository owner
consulted: 0002-MADR-migrate-llmprovider-from-mcplib.md (amendment 2026-09-29 (fifth)), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (Z5)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 -->

# Require Go 1.27.2, for the Standard Library's Security Fixes

## Context and Problem Statement

`go.mod` says `go 1.27.1`, set by
[0002-MADR](0002-MADR-migrate-llmprovider-from-mcplib.md)'s fifth amendment.
CI installs that version (`.github/workflows/ci.yml`, `setup-go` with
`go-version-file: go.mod`) and runs govulncheck `v1.8.0` over the module
([0021-MADR](0021-MADR-harden-and-tune-after-the-v1-1-review.md) Z5).

On 2026-10-08 CI failed on `main` at `52709e4`: the Linux job's govulncheck
step, and only that step; the Windows and macOS jobs passed. Run on this
host, from the tree, govulncheck `v1.8.0` gives:

| Toolchain | Exit | Findings |
| :--- | :--- | :--- |
| `GOTOOLCHAIN=go1.27.1` | 3 | 10 called, 1 imported but not called, 2 in required modules but not called; every one in the standard library, every one "Fixed in: …@go1.27.2" |
| `GOTOOLCHAIN=go1.27.2` | 0 | "No vulnerabilities found." |

The ten called:

| ID | Package | Summary (govulncheck's) |
| :--- | :--- | :--- |
| GO-2026-6617 | `net/http/internal/http2` | HTTP/2 server crash due to HPACK encoder race |
| GO-2026-6613 | `net/http` | HTTP/1 server connection desynchronization after 2xx CONNECT response |
| GO-2026-6612 | `net/http/internal/http2` | Double flow control refund on HTTP/2 server streams |
| GO-2026-6611 | `net/http/internal/http2` | Excessive CPU consumption from repeated initial window changes |
| GO-2026-6610 | `net/http/internal/http2` | HTTP/2 transport accepts malformed framing-related headers |
| GO-2026-6608 | `net/textproto` | Memory limit bypass when parsing MIME headers |
| GO-2026-6607 | `crypto/tls` | Reject malformed ECH outer extension references |
| GO-2026-6605 | `net/http` | HTTP/1 client connection desynchronization after CONNECT rejection |
| GO-2026-6604 | `os` (Windows) | `Root.Mkdir(All)` can follow junctions out of the root |
| GO-2026-6603 | `net/http/internal/http2` | HTTP/2 server memory exhaustion due to Trailer headers |

Not called: GO-2026-6609 (`net/http`, Range header size), GO-2026-6600 and
GO-2026-6599 (`html/template`). `golang.org/x/sys v0.47.0` and
`golang.org/x/term v0.43.0` have no findings.

The owner's Go environment already selects `go1.27.2`
(`GOTOOLCHAIN=go1.27.2` in the user's Go env file), so local runs pass while
CI, reading `go.mod`, does not. `go.mod` is tidy at both versions
(`go mod tidy -diff` exits 0).

The SDK is a library: its `go` directive is the minimum a consumer's module
must declare, not the toolchain a consumer builds with. Every SDK request
goes through `net/http` and `crypto/tls`, so a binary built with 1.27.1 is
exposed whatever the directive says. The directive does decide whether the
SDK's own CI passes, and raising it makes `go get` raise a consumer's
directive to 1.27.2, so a consumer that honours `go`'s toolchain selection
builds with the fixed library.

## Decision Drivers

* CI on `main` is red, and every later commit stays red until CI's Go moves.
* The vulnerabilities are in code every request runs: HTTP/2 and HTTP/1
  transports, TLS.
* AGENTS.md's dependency rule: `go.mod` changes with the change that needs
  it, and `go mod tidy -diff` is clean at every commit.
* A raised directive is a requirement imposed on consumers.

## Considered Options

* Raise the `go` directive to `1.27.2`.
* Keep `go 1.27.1` and add `toolchain go1.27.2`.
* Keep `go.mod`; pin CI's `setup-go` to 1.27.2 directly.
* Keep everything; exclude the findings from govulncheck.

## Decision Outcome

Chosen option: "Raise the `go` directive to `1.27.2`", because it is the one
option where `go.mod`, CI and every consumer's minimum agree on a toolchain
with the fixes, with no second source of truth for CI's version.

It supersedes the directive of
[0002-MADR](0002-MADR-migrate-llmprovider-from-mcplib.md)'s fifth amendment,
which gains a pointer here. AGENTS.md, README.md and
[architecture.md](../architecture.md) say "Go 1.27.2". It ships with 0028 in
`v1.4.0` (the owner, Q1).

### Consequences

* Good, because CI's govulncheck step passes with no exclusion, and CI and
  the owner's host run the same toolchain.
* Good, because a consumer that runs `go get` on the release gets
  `go 1.27.2`, so its own `go` command selects a toolchain with the fixes.
* Bad, because a consumer at `go 1.27.1` must raise its directive to take
  the release; with `GOTOOLCHAIN=auto`, the default, `go` downloads 1.27.2
  for it.
* Neutral, because `go.sum` and the requirements do not change.

### Confirmation

* `go.mod` says `go 1.27.2` and has no `toolchain` line; `go mod tidy -diff`
  exits 0 under `GOTOOLCHAIN=go1.27.2`.
* `GOTOOLCHAIN=go1.27.1 go build ./...` fails, naming the requirement.
* `make vuln` exits 0, and CI's Linux job passes its govulncheck step.
* `rg -n "1\.27\.1"` finds no current-state statement outside
  `docs/decisions/` and `docs/reports/`.

## Pros and Cons of the Options

### Raise the `go` directive to `1.27.2`

* Good, because one line sets CI's Go, the module's minimum, and consumers'.
* Bad, because consumers must be at 1.27.2.

### Keep `go 1.27.1` and add `toolchain go1.27.2`

* Good, because consumers' minimum stays 1.27.1.
* Bad, because a `toolchain` line in a dependency is ignored by consumers,
  so it protects only this repository's own builds.
* Bad, because CI's Go would rest on how `setup-go`'s `go-version-file`
  treats a `toolchain` line, which was not checked for the pinned `v7.0.0`.

### Keep `go.mod`; pin CI's `setup-go` to 1.27.2 directly

* Good, because nothing changes for consumers.
* Bad, because `go.mod` and CI disagree, the drift this record fixes, and
  consumers keep no signal that 1.27.1 is unsafe.

### Keep everything; exclude the findings from govulncheck

* Bad, because it hides ten called vulnerabilities to turn CI green.
  Not a fix.

## More Information

* Measured 2026-10-08 on the owner's host, govulncheck `v1.8.0` (CI's pin),
  through `go run`, as `make vuln` runs it; outputs kept in the session's
  scratch directory, not committed.
* CI run: the Linux `validate` job on `52709e4`, step "govulncheck
  (0021-MADR Z5)", failed; the Windows and macOS jobs passed.
* [0029-PLAN-go-1-27-2-for-standard-library-fixes.md](0029-PLAN-go-1-27-2-for-standard-library-fixes.md)
  carries the change.
