# go-llmprovider-sdk documentation

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [LLM provider extraction feasibility](reports/0001-REPORT-llmprovider-extraction-feasibility.md) | observation |
| 0002 | MADR | [Migrate `llmprovider` and `wizard` from mcplib](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) | accepted |
| 0002 | PLAN | [Implement the migration from mcplib](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md) | in-progress |
| 0015 | REPORT | [SDK API surface assessment](reports/0015-REPORT-sdk-api-surface-assessment.md) | observation |
| 0015 | MADR | [Canonical, modular v1 API](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) | accepted |
| 0015 | PLAN | [Implement the canonical v1 API](decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) | in-progress |
| 0016 | MADR | [Provider auth and support baseline: `mcplib` code, `magic-cli-remote` credential hygiene](decisions/0016-MADR-provider-auth-and-support-baseline.md) | proposed |
| 0016 | PLAN | [Implement the provider auth and support baseline](decisions/0016-PLAN-provider-auth-and-support-baseline.md) | proposed |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| contribute: checks, records and commit rules | [AGENTS.md](../AGENTS.md) |
| know whether `mcplib/llmprovider` can move here, and what it would take | [0001-REPORT](reports/0001-REPORT-llmprovider-extraction-feasibility.md) |
| see what the migration decides: identity, versions, records, open work | [0002-MADR](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| see the migration steps and which repository each happens in | [0002-PLAN](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md) |
| know what runs before an agent commit here, and why | [0002-MADR, "Amendment 2026-09-29"](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md#amendment-2026-09-29-pre-add-gate-and-agent-pointers) |
| know why the SDK's API differs from `mcplib`'s, and what it will be | [0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| see how the imported API measured against the SDK requirements | [0015-REPORT](reports/0015-REPORT-sdk-api-surface-assessment.md) |
| know why provider auth builds on `mcplib` and not `magic-cli-remote`, and what it takes from each | [0016-MADR](decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| find where an `mcplib` record number ends up here | [0002-MADR, "Records move here and are renumbered locally"](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md#10-records-move-here-and-are-renumbered-locally) |
