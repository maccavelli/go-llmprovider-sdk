# Architecture

How `go-llmprovider-sdk` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

A Git repository for the Go module `github.com/maccavelli/go-llmprovider-sdk`:
a library for LLM provider access and provider authentication. It has no
binary.

The Go code in the tree is `mcplib` `v1.6.0`'s `llmprovider` and `wizard`
packages and its redaction files, byte for byte. The module requires Go
1.27.1, `golang.org/x/term` and, indirectly, `golang.org/x/sys`. The code
still imports three `mcplib` packages that the module does not require, so
`llmprovider` and `wizard` do not build; `internal/redact` builds and passes
its tests.

## Tree

```text
README.md                   repository entry; links here
AGENTS.md                   rules for agents: records, checks, commits
go.mod, go.sum              the module and its two requirements
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration
.markdownlint-cli2.jsonc    Markdown lint configuration
.github/workflows/ci.yml    CI
scripts/go-precheck.sh      the pre-add check
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
llmprovider/                provider adapters, OAuth, token store, discovery
wizard/                     interactive provider configuration
internal/redact/            secret redaction
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  numbered observations
  mcplib-import/            records imported from mcplib, not yet renumbered
```

## Go code

| Directory | Package | Non-test files | Test files | Imports from `mcplib` |
| :--- | :--- | :--- | :--- | :--- |
| `llmprovider/` | `llmprovider` | 43 | 107, of which 23 are `live_gateways` only | `logging` |
| `wizard/` | `wizard` | 6 | 11 | the root package, `llmprovider`, `logging` |
| `internal/redact/` | `logging` | 2 | 2 | none |

- `llmprovider` holds nine providers over five wire formats (OpenAI
  Responses, Anthropic Messages, Gemini Interactions, Chat Completions, and
  Gemini `generateContent` for OpenCode's Google route), the
  browser and device-code OAuth flows, token refresh and revocation,
  `FileTokenStore`, read-through vendor-CLI sessions, and model listing and
  ranking.
- `wizard` runs the configuration flow through a `Prompter` and returns a
  `Result`; it does not write configuration.
- `internal/redact` is a copy of `mcplib`'s `logging` redaction and still
  declares `package logging`.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `pre-add-check`, `help`.
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files,
  `golangci-lint run -c .golangci.yml ./...`, `go vet` and `go test` on their
  packages, and `govulncheck ./...`. `make pre-add-check` runs it, and so
  does the machine-wide agent gate before an agent `git commit` that stages
  Go files. Its `golangci-lint`, `go vet` and `go test` steps fail with
  `no required module provides package github.com/maccavelli/mcplib…`.
- **`make tidy`** must not be run yet: `go mod tidy` would add `mcplib` as a
  requirement.
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`: `go test`; on Linux also `go vet`,
  `gofmt`, `go mod tidy -diff`, `make lint` and
  `go vet -tags live_gateways`. It fails at `go test` on the `mcplib`
  imports.

## What is not here

- **The re-homing of imports and identity**, after which every package
  builds and `go mod tidy -diff` is clean:
  [0002-PLAN](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md)
  Phase 4.
- **The records in `mcplib-import/`** in their final numbered places:
  0002-PLAN Phase 6.
- **`guides/`:** created with its first guide by
  [0015-PLAN](decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
  Phase S1.
- **The v1 package layout** (`llmprovider/auth`, `llmprovider/catalog`,
  `llmprovider/providers/…`, `llmprovider/llmtest`):
  [0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md),
  proposed.
- **A licence file.**
