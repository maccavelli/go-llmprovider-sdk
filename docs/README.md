# go-llmprovider-sdk documentation

[architecture.md](architecture.md) describes the system as it is. The records
below hold the decisions and their history. [guides/](guides/) holds the
documents a reader follows to do something.

## Records

64 records. Cite them by full filename.

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [LLM Provider Extraction Feasibility: mcplib/llmprovider to go-llmprovider-sdk](reports/0001-REPORT-llmprovider-extraction-feasibility.md) | observation |
| 0002 | MADR | [Migrate `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk as a standalone v1 module](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) | accepted |
| 0002 | PLAN | [Implement the Migration of `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md) | in-progress |
| 0003 | MADR | [Adopt a Responses-API-Shaped Canonical Contract Across All `llmprovider` Providers, Including a New Grok Provider](decisions/0003-MADR-add-grok-xai-llm-provider.md) | accepted |
| 0003 | PLAN | [Implementation Plan: Responses-API Canonical Contract + Grok Provider](decisions/0003-PLAN-add-grok-xai-llm-provider.md) | complete |
| 0004 | MADR | [Add OpenCode Zen/Go, Hugging Face and Kilo Gateway Providers on a Shared Chat Completions Primitive](decisions/0004-MADR-add-gateway-llm-providers.md) | accepted |
| 0004 | PLAN | [Implementation Plan: Gateway LLM Providers on a Shared Chat Completions Primitive](decisions/0004-PLAN-add-gateway-llm-providers.md) | complete |
| 0005 | MADR | [Canonicalize LLM Provider Configuration — Descriptors, Flow and Prompting — in `mcplib`, Renderer-Agnostic](decisions/0005-MADR-canonicalize-llm-provider-configuration.md) | accepted |
| 0005 | PLAN | [Implementation Plan: Canonicalize LLM Provider Configuration in `mcplib`](decisions/0005-PLAN-canonicalize-llm-provider-configuration.md) | complete |
| 0006 | MADR | [Support Browser, API-Key, and Headless Subscription Authentication for OpenAI and xAI Grok in `mcplib`](decisions/0006-MADR-subscription-auth-for-llm-providers.md) | accepted |
| 0006 | PLAN | [Implement Browser, API-Key, and Headless Subscription Authentication for OpenAI and xAI Grok](decisions/0006-PLAN-subscription-auth-for-llm-providers.md) | complete |
| 0007 | MADR | [Search Live Provider Catalogs for Primary and Fallback Model Selection](decisions/0007-MADR-live-catalog-model-search.md) | accepted |
| 0007 | PLAN | [Implement Search Live Provider Catalogs for Primary and Fallback Model Selection](decisions/0007-PLAN-live-catalog-model-search.md) | complete |
| 0008 | MADR | [Repair OAuth loopback, paste-code fallback, and Windows session wiring so ChatGPT and xAI browser login actually persist](decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md) | accepted |
| 0008 | PLAN | [PLAN 0008 — Repair OAuth loopback, paste-code, and Windows session wiring](decisions/0008-PLAN-repair-oauth-loopback-and-session-wiring.md) | complete |
| 0009 | MADR | [Rank Recommended Models by Use Case from Live Catalog Metadata](decisions/0009-MADR-use-case-aware-default-model-ranking.md) | accepted |
| 0009 | PLAN | [Implement Rank Recommended Models by Use Case from Live Catalog Metadata](decisions/0009-PLAN-use-case-aware-default-model-ranking.md) | complete |
| 0010 | MADR | [Close Windows stdio shutdown misclassification, finish the mcplib remainder of 0008, harden FileTokenStore, and make CI prove it on macOS, Linux, and Windows](decisions/0010-MADR-windows-stdio-oauth-tokenstore-ci.md) | accepted |
| 0010 | PLAN | [PLAN 0010 — Windows stdio shutdown, 0008 mcplib remainder, FileTokenStore, CI parity](decisions/0010-PLAN-windows-stdio-oauth-tokenstore-ci.md) | complete |
| 0011 | REPORT | [Provider Source Compatibility Audit: Kilo, OpenCode, Grok and Codex](reports/0011-REPORT-provider-source-compatibility-audit.md) | observation |
| 0012 | MADR | [Conform `llmprovider` to the Reference Clients of Kilo, OpenCode, Grok and Codex](decisions/0012-MADR-conform-providers-to-reference-clients.md) | accepted |
| 0012 | PLAN | [Implement 0012 §4 — The ChatGPT Backend](decisions/0012-PLAN-chatgpt-backend.md) | complete |
| 0012 | PLAN | [Implement 0012 §3 — Gateway Conventions (OpenCode and Kilo)](decisions/0012-PLAN-gateway-conventions.md) | complete |
| 0012 | PLAN | [Implement 0012 §6 — Grok](decisions/0012-PLAN-grok.md) | complete |
| 0012 | PLAN | [Implement 0012 §2 — Canonical Item Fidelity](decisions/0012-PLAN-item-fidelity.md) | complete |
| 0012 | PLAN | [Implement 0012 §5 — OAuth Session Hygiene](decisions/0012-PLAN-oauth-hygiene.md) | complete |
| 0012 | PLAN | [Implement 0012 §1 — Shared Transport](decisions/0012-PLAN-shared-transport.md) | complete |
| 0013 | MADR | [Remediate the Defects Found by the Post-0010 Debugging Pass](decisions/0013-MADR-remediate-debugging-pass-findings.md) | accepted |
| 0013 | PLAN | [Implement Remediation of the Post-0010 Debugging-Pass Defects](decisions/0013-PLAN-remediate-debugging-pass-findings.md) | complete |
| 0014 | MADR | [Complete Gemini's Move to the Interactions API, and Fix the `generateContent` Wire That Stays](decisions/0014-MADR-gemini-wire-fidelity.md) | accepted |
| 0014 | PLAN | [Implement 0014 — Gemini on the Interactions API, and the `generateContent` Fixes](decisions/0014-PLAN-gemini-wire-fidelity.md) | complete |
| 0015 | REPORT | [SDK API Surface Assessment: The Imported `mcplib` API Against the SDK Requirements](reports/0015-REPORT-sdk-api-surface-assessment.md) | observation |
| 0015 | MADR | [Define a Canonical, Modular v1 API for go-llmprovider-sdk Before the First Release](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) | accepted |
| 0015 | PLAN | [Implement the Canonical, Modular v1 API for go-llmprovider-sdk](decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) | complete |
| 0016 | MADR | [Build Provider Support and Authentication from mcplib's `llmprovider`, and Adopt magic-cli-remote's Credential Hygiene](decisions/0016-MADR-provider-auth-and-support-baseline.md) | accepted |
| 0016 | PLAN | [Implement the Provider Auth and Support Baseline](decisions/0016-PLAN-provider-auth-and-support-baseline.md) | complete |
| 0017 | REPORT | [Reference-Client Survey: How Six Coding Agents Authenticate, Compared with `llmprovider`](reports/0017-REPORT-reference-client-auth-survey.md) | observation |
| 0017 | MADR | [Add Together AI, Kilo Device Login and Command-Sourced Keys, from the Reference-Client Survey](decisions/0017-MADR-together-provider-and-auth-extensions.md) | accepted |
| 0017 | PLAN | [Implement Together AI, Kilo Device Login and Command-Sourced Keys](decisions/0017-PLAN-together-provider-and-auth-extensions.md) | complete |
| 0018 | MADR | [The module is released under the Apache License 2.0](decisions/0018-MADR-apache-2-license.md) | accepted |
| 0018 | PLAN | [Implement the Apache License 2.0 for go-llmprovider-sdk](decisions/0018-PLAN-apache-2-license.md) | complete |
| 0019 | MADR | [Adopt `go fix`'s modernizations, except one that changes behaviour](decisions/0019-MADR-go-fix-modernizations.md) | accepted |
| 0019 | PLAN | [Implement `go fix`'s modernizations, except one that changes behaviour](decisions/0019-PLAN-go-fix-modernizations.md) | complete |
| 0020 | MADR | [Remediate the Defects Found by the v1 Debugging Pass](decisions/0020-MADR-remediate-v1-debugging-pass-findings.md) | accepted |
| 0020 | PLAN | [Implement the Remediation of the v1 Debugging Pass](decisions/0020-PLAN-remediate-v1-debugging-pass-findings.md) | complete |
| 0021 | MADR | [Harden and Tune the SDK After the v1.1 Review](decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md) | accepted |
| 0021 | PLAN | [Implement the Hardening and Tuning After the v1.1 Review](decisions/0021-PLAN-harden-and-tune-after-the-v1-1-review.md) | complete |
| 0022 | MADR | [Pin Gemini's Live Instructions Check with a Neutral Instruction and a Baseline](decisions/0022-MADR-gemini-live-instructions-test.md) | accepted |
| 0022 | PLAN | [Implement the Paired Gemini Live Instructions Check](decisions/0022-PLAN-gemini-live-instructions-test.md) | complete |
| 0023 | MADR | [Keep Hugging Face's `ToolChoiceNone` by Sending No Tools](decisions/0023-MADR-huggingface-tool-choice-none.md) | accepted |
| 0023 | PLAN | [Implement Hugging Face's `ToolChoiceNone` by Sending No Tools](decisions/0023-PLAN-huggingface-tool-choice-none.md) | complete |
| 0024 | MADR | [Pair OpenCode's Live System-Message Check with a Neutral Instruction and a Baseline](decisions/0024-MADR-opencode-live-system-message-test.md) | accepted |
| 0024 | PLAN | [Implement the Paired OpenCode Live System-Message Check](decisions/0024-PLAN-opencode-live-system-message-test.md) | complete |
| 0025 | MADR | [Check the Decision Records Mechanically, and Correct the Two Stale PLAN Statuses](decisions/0025-MADR-records-consistency-check.md) | accepted |
| 0025 | PLAN | [Implement the Records Consistency Check, and Correct the Two Stale PLAN Statuses](decisions/0025-PLAN-records-consistency-check.md) | complete |
| 0026 | MADR | [Remediate the Defects Found by the v1.2 Debugging Pass](decisions/0026-MADR-remediate-v1-2-debugging-pass-findings.md) | accepted |
| 0026 | PLAN | [Implement the Remediation of the v1.2 Debugging Pass](decisions/0026-PLAN-remediate-v1-2-debugging-pass-findings.md) | complete |
| 0027 | MADR | [Skip Live Tests by One Rule, and Measure Gemini's 429 on the Path the SDK Sends](decisions/0027-MADR-live-test-skips-and-gemini-429-path.md) | accepted |
| 0027 | PLAN | [Implement One Skip Rule for the Live Tests, and Gemini's 429 Check on the Interactions Path](decisions/0027-PLAN-live-test-skips-and-gemini-429-path.md) | complete |
| 0028 | MADR | [Correct Six Error and Retry Heuristics, and Remove Five Measured Costs](decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md) | accepted |
| 0028 | PLAN | [Implement the Six Heuristic Corrections and the Five Measured Cost Reductions](decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md) | complete |
| 0029 | MADR | [Require Go 1.27.2, for the Standard Library's Security Fixes](decisions/0029-MADR-go-1-27-2-for-standard-library-fixes.md) | accepted |
| 0029 | PLAN | [Implement Go 1.27.2 as the Module's Requirement](decisions/0029-PLAN-go-1-27-2-for-standard-library-fixes.md) | complete |
| 0030 | REPORT | [The v1 SDK API Surfaces and the Protocols the Built-in Providers Speak](reports/0030-REPORT-v1-api-and-protocol-surfaces.md) | observation |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| contribute: checks, records and commit rules | [AGENTS.md](../AGENTS.md) |
| know the licence | [0018-MADR](decisions/0018-MADR-apache-2-license.md), [LICENSE](../LICENSE) |
| run the live tests against real services | [AGENTS.md, "Live tests"](../AGENTS.md#live-tests) |
| add a provider, in this module or my own | [guides/adding-a-provider.md](guides/adding-a-provider.md) |
| know how the gateways and the first direct providers were added | [0004-MADR](decisions/0004-MADR-add-gateway-llm-providers.md) (gateways), [0003-MADR](decisions/0003-MADR-add-grok-xai-llm-provider.md) (a direct provider) |
| know how each provider matches its vendor's reference client | [0012-MADR](decisions/0012-MADR-conform-providers-to-reference-clients.md) |
| understand ChatGPT and Grok sign-in (OAuth) | [0006-MADR](decisions/0006-MADR-subscription-auth-for-llm-providers.md), [0008-MADR](decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md) |
| understand model listing, search and ranking | [0007-MADR](decisions/0007-MADR-live-catalog-model-search.md), [0009-MADR](decisions/0009-MADR-use-case-aware-default-model-ranking.md) |
| understand the configuration wizard | [0005-MADR](decisions/0005-MADR-canonicalize-llm-provider-configuration.md) |
| know whether `mcplib/llmprovider` can move here, and what it would take | [0001-REPORT](reports/0001-REPORT-llmprovider-extraction-feasibility.md) |
| see what the migration decides: identity, versions, records, open work | [0002-MADR](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| see the migration steps and which repository each happens in | [0002-PLAN](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md) |
| know what runs before an agent commit here, and why | [0002-MADR, "Amendment 2026-09-29"](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md#amendment-2026-09-29-pre-add-gate-and-agent-pointers) |
| know why the SDK's API differs from `mcplib`'s | [0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| know the rules a new or changed API must follow | [guides/api-standards.md](guides/api-standards.md) |
| move code that used `mcplib`'s `llmprovider` or `wizard` to this module | [guides/migrating-from-mcplib.md](guides/migrating-from-mcplib.md) |
| see how the imported API measured against the SDK requirements | [0015-REPORT](reports/0015-REPORT-sdk-api-surface-assessment.md) |
| see the v1 generation API and which wire protocols each provider speaks | [0030-REPORT](reports/0030-REPORT-v1-api-and-protocol-surfaces.md) |
| know why provider auth builds on `mcplib` and not `magic-cli-remote`, and what it takes from each | [0016-MADR](decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| compare this module's auth with Codex, Grok, Kilo, OpenCode, pi, agy and Claude Code | [0017-REPORT](reports/0017-REPORT-reference-client-auth-survey.md) |
| know how Together AI, Kilo device login and command-sourced keys are being added | [0017-MADR](decisions/0017-MADR-together-provider-and-auth-extensions.md) |
| find where an `mcplib` record number ends up here | [the table below](#migrated-from-mcplib) |

## Migrated from mcplib

Records 0003–0014 came from `mcplib` at `4e1f9a5` under
[0002-MADR](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) §10, renumbered once.
`mcplib` numbered two records 0009 and two 0010; here each has its own number.
Records that stayed in `mcplib` (0002, 0005, 0006, 0007 and
`0012-PLAN-circuit-breaker-test.md`) are cited here by repository and name.

| `mcplib` path | Here |
| :--- | :--- |
| `docs/0001-MADR-add-grok-xai-llm-provider.md` | [0003-MADR-add-grok-xai-llm-provider.md](decisions/0003-MADR-add-grok-xai-llm-provider.md) |
| `docs/0001-PLAN-add-grok-xai-llm-provider.md` | [0003-PLAN-add-grok-xai-llm-provider.md](decisions/0003-PLAN-add-grok-xai-llm-provider.md) |
| `docs/0003-MADR-add-gateway-llm-providers.md` | [0004-MADR-add-gateway-llm-providers.md](decisions/0004-MADR-add-gateway-llm-providers.md) |
| `docs/0003-PLAN-add-gateway-llm-providers.md` | [0004-PLAN-add-gateway-llm-providers.md](decisions/0004-PLAN-add-gateway-llm-providers.md) |
| `docs/0004-MADR-canonicalize-llm-provider-configuration.md` | [0005-MADR-canonicalize-llm-provider-configuration.md](decisions/0005-MADR-canonicalize-llm-provider-configuration.md) |
| `docs/0004-PLAN-canonicalize-llm-provider-configuration.md` | [0005-PLAN-canonicalize-llm-provider-configuration.md](decisions/0005-PLAN-canonicalize-llm-provider-configuration.md) |
| `docs/0008-MADR-subscription-auth-for-llm-providers.md` | [0006-MADR-subscription-auth-for-llm-providers.md](decisions/0006-MADR-subscription-auth-for-llm-providers.md) |
| `docs/0008-PLAN-subscription-auth-for-llm-providers.md` | [0006-PLAN-subscription-auth-for-llm-providers.md](decisions/0006-PLAN-subscription-auth-for-llm-providers.md) |
| `docs/0009-MADR-live-catalog-model-search.md` | [0007-MADR-live-catalog-model-search.md](decisions/0007-MADR-live-catalog-model-search.md) |
| `docs/0009-PLAN-live-catalog-model-search.md` | [0007-PLAN-live-catalog-model-search.md](decisions/0007-PLAN-live-catalog-model-search.md) |
| `docs/decisions/0009-MADR-repair-oauth-loopback-and-session-wiring.md` | [0008-MADR-repair-oauth-loopback-and-session-wiring.md](decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md) |
| `docs/decisions/0009-PLAN-repair-oauth-loopback-and-session-wiring.md` | [0008-PLAN-repair-oauth-loopback-and-session-wiring.md](decisions/0008-PLAN-repair-oauth-loopback-and-session-wiring.md) |
| `docs/0010-MADR-use-case-aware-default-model-ranking.md` | [0009-MADR-use-case-aware-default-model-ranking.md](decisions/0009-MADR-use-case-aware-default-model-ranking.md) |
| `docs/0010-PLAN-use-case-aware-default-model-ranking.md` | [0009-PLAN-use-case-aware-default-model-ranking.md](decisions/0009-PLAN-use-case-aware-default-model-ranking.md) |
| `docs/decisions/0010-MADR-windows-stdio-oauth-tokenstore-ci.md (copied)` | [0010-MADR-windows-stdio-oauth-tokenstore-ci.md](decisions/0010-MADR-windows-stdio-oauth-tokenstore-ci.md) |
| `docs/decisions/0010-PLAN-windows-stdio-oauth-tokenstore-ci.md (copied)` | [0010-PLAN-windows-stdio-oauth-tokenstore-ci.md](decisions/0010-PLAN-windows-stdio-oauth-tokenstore-ci.md) |
| `docs/0011-REPORT-provider-source-compatibility-audit.md` | [0011-REPORT-provider-source-compatibility-audit.md](reports/0011-REPORT-provider-source-compatibility-audit.md) |
| `docs/0012-MADR-conform-providers-to-reference-clients.md` | [0012-MADR-conform-providers-to-reference-clients.md](decisions/0012-MADR-conform-providers-to-reference-clients.md) |
| `docs/0012-PLAN-chatgpt-backend.md` | [0012-PLAN-chatgpt-backend.md](decisions/0012-PLAN-chatgpt-backend.md) |
| `docs/0012-PLAN-gateway-conventions.md` | [0012-PLAN-gateway-conventions.md](decisions/0012-PLAN-gateway-conventions.md) |
| `docs/0012-PLAN-grok.md` | [0012-PLAN-grok.md](decisions/0012-PLAN-grok.md) |
| `docs/0012-PLAN-item-fidelity.md` | [0012-PLAN-item-fidelity.md](decisions/0012-PLAN-item-fidelity.md) |
| `docs/0012-PLAN-oauth-hygiene.md` | [0012-PLAN-oauth-hygiene.md](decisions/0012-PLAN-oauth-hygiene.md) |
| `docs/0012-PLAN-shared-transport.md` | [0012-PLAN-shared-transport.md](decisions/0012-PLAN-shared-transport.md) |
| `docs/0013-MADR-remediate-debugging-pass-findings.md` | [0013-MADR-remediate-debugging-pass-findings.md](decisions/0013-MADR-remediate-debugging-pass-findings.md) |
| `docs/0013-PLAN-remediate-debugging-pass-findings.md` | [0013-PLAN-remediate-debugging-pass-findings.md](decisions/0013-PLAN-remediate-debugging-pass-findings.md) |
| `docs/decisions/0014-MADR-gemini-wire-fidelity.md` | [0014-MADR-gemini-wire-fidelity.md](decisions/0014-MADR-gemini-wire-fidelity.md) |
| `docs/decisions/0014-PLAN-gemini-wire-fidelity.md` | [0014-PLAN-gemini-wire-fidelity.md](decisions/0014-PLAN-gemini-wire-fidelity.md) |
