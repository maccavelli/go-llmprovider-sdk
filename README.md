# go-llmprovider-sdk

A Go library for calling LLM providers and authenticating to them: API keys,
subscription OAuth (browser and device-code login, refresh, revocation), a
token store, model discovery, and an interactive configuration wizard. It
depends on the standard library and `golang.org/x/term` only, and ships no
binary.

Module: `github.com/maccavelli/go-llmprovider-sdk`

**Documentation:** [docs/](docs/README.md)

## Status

There is no release yet. The module requires Go 1.27.1.

- The provider and wizard code was imported, with its history, from
  `mcplib` `v1.6.0` and re-homed here
  ([0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md),
  Phase 4). It builds and its tests pass. Its API is still `mcplib`'s,
  apart from the removed orchestration option, until the v1 API lands.
- The v1 API is decided by
  [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
  (accepted). The provider-auth baseline is proposed in
  [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md).
  The first release, `v1.0.0`, follows them.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| know why the code is moving here from `mcplib`, and how | [0002-MADR](docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| know what the v1 API will look like | [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| know how provider auth is built, and what it takes from `magic-cli-remote` | [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |
