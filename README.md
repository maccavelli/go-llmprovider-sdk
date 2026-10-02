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
  Phase 4).
- Its API is the v1 API that
  [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
  decides, on the provider-auth baseline of
  [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md).
  [0015-PLAN](docs/decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
  has built it; its CI enforcement and close-out remain. The first release,
  `v1.0.0`, follows.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| know why the code is moving here from `mcplib`, and how | [0002-MADR](docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| know why the v1 API is shaped as it is | [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| change or add an exported API | [api-standards.md](docs/guides/api-standards.md) |
| add a provider, in this module or my own | [adding-a-provider.md](docs/guides/adding-a-provider.md) |
| move code from `mcplib`'s `llmprovider` or `wizard` | [migrating-from-mcplib.md](docs/guides/migrating-from-mcplib.md) |
| know how provider auth is built, and what it takes from `magic-cli-remote` | [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
