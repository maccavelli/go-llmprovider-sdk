# go-llmprovider-sdk documentation

[architecture.md](architecture.md) describes the system as it is. The records
below hold the decisions and their history. [guides/](guides/) holds the
documents a reader follows to do something.

## Records

137 records. Cite them by full filename.

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [LLM Provider Extraction Feasibility: mcplib/llmprovider to go-llmprovider-sdk](reports/0001-REPORT-llmprovider-extraction-feasibility.md) | observation |
| 0002 | MADR | [Migrate `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk as a standalone v1 module](decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) | accepted |
| 0002 | PLAN | [Implement the Migration of `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk](decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md) | complete |
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
| 0017 | REPORT | [OpenAI's Sign in with ChatGPT for Open-Source Agents: The Flow, Its Terms, and What `llmprovider` Lacks](reports/0017-REPORT-openai-sign-in-for-agents.md) | observation |
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
| 0031 | MADR | [Stream Natively on the OpenAI Responses Wire](decisions/0031-MADR-native-streaming-on-the-responses-wire.md) | accepted |
| 0031 | PLAN | [Implement Native Streaming on the OpenAI Responses Wire](decisions/0031-PLAN-native-streaming-on-the-responses-wire.md) | complete |
| 0031 | PLAN | [Implement Native Streaming on the Messages, Chat Completions, generateContent and Interactions Wires](decisions/0031-PLAN-native-streaming-on-the-other-wires.md) | complete |
| 0032 | MADR | [Carry a Per-Tool Strict Flag and a Tool Result's Error Flag](decisions/0032-MADR-tool-strict-and-result-error-flags.md) | accepted |
| 0032 | PLAN | [Implement a Per-Tool Strict Flag and a Tool Result's Error Flag](decisions/0032-PLAN-tool-strict-and-result-error-flags.md) | complete |
| 0033 | MADR | [Publish Model Metadata, and Bring the Curated Catalogs Up to Date](decisions/0033-MADR-public-model-metadata-and-current-catalogs.md) | accepted |
| 0033 | PLAN | [Refresh the Curated Model Catalogs](decisions/0033-PLAN-refresh-the-curated-catalogs.md) | complete |
| 0033 | PLAN | [Publish Model Metadata](decisions/0033-PLAN-public-model-metadata.md) | complete |
| 0034 | MADR | [Degrade a Forced Tool Choice Per Model, Where Anthropic Refuses It](decisions/0034-MADR-degrade-forced-tool-choice-per-model.md) | accepted |
| 0034 | PLAN | [Implement a Per-Model Degradation of Forced Tool Choice](decisions/0034-PLAN-degrade-forced-tool-choice-per-model.md) | complete |
| 0035 | MADR | [Keep the Maintainer Tools in `tools/`, Written in Python](decisions/0035-MADR-maintainer-tools-in-python.md) | accepted |
| 0035 | PLAN | [Implement the Maintainer Tools in `tools/`](decisions/0035-PLAN-maintainer-tools-in-python.md) | complete |
| 0036 | MADR | [Report Cache-Write Tokens in `Usage`, as a Part of the Input Count](decisions/0036-MADR-cache-write-token-count.md) | accepted |
| 0036 | PLAN | [Implement the Cache-Write Counts in `Usage`](decisions/0036-PLAN-cache-write-token-count.md) | complete |
| 0037 | MADR | [Read Anthropic's Reasoning Count, and a Fourth Cache-Read Spelling on Chat Completions](decisions/0037-MADR-reasoning-and-cache-read-spellings.md) | accepted |
| 0037 | PLAN | [Implement Anthropic's Reasoning Count and the Fourth Cache-Read Spelling](decisions/0037-PLAN-reasoning-and-cache-read-spellings.md) | complete |
| 0038 | MADR | [List the Full Curated Catalog by Default, and Six Only on Request](decisions/0038-MADR-full-curated-catalog-by-default.md) | accepted |
| 0038 | PLAN | [Implement the Full Curated Catalog by Default, and Six Only on Request](decisions/0038-PLAN-full-curated-catalog-by-default.md) | complete |
| 0039 | MADR | [Let a Request Ask for Prompt Caching](decisions/0039-MADR-request-side-prompt-caching.md) | accepted |
| 0039 | PLAN | [Implement Request-Side Prompt Caching](decisions/0039-PLAN-request-side-prompt-caching.md) | complete |
| 0040 | MADR | [Let a Request Defer Tools Until a Tool Result Loads Them](decisions/0040-MADR-native-tool-deferral.md) | accepted |
| 0040 | PLAN | [Implement Native Tool Deferral](decisions/0040-PLAN-native-tool-deferral.md) | complete |
| 0041 | MADR | [Order a Live Model Menu, and Decide Per-Model Behaviour, From Model Facts](decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md) | accepted |
| 0041 | PLAN | [Implement a Live Model Menu, and Per-Model Behaviour, From Model Facts](decisions/0041-PLAN-decide-per-model-behaviour-from-metadata.md) | complete |
| 0042 | MADR | [List Only the Ollama Models That Write Text, and Keep the Tool-less Ones Recommended](decisions/0042-MADR-ollama-models-that-write-text.md) | accepted |
| 0042 | PLAN | [Implement Only the Ollama Models That Write Text, and Keep the Tool-less Ones Recommended](decisions/0042-PLAN-ollama-models-that-write-text.md) | complete |
| 0043 | MADR | [Add OpenAI's Sign in with ChatGPT for Agents as an OpenAI Auth Method](decisions/0043-MADR-openai-chatgpt-plan-sign-in.md) | accepted |
| 0043 | PLAN | [Implement OpenAI's Sign in with ChatGPT for Agents as an OpenAI Auth Method](decisions/0043-PLAN-openai-chatgpt-plan-sign-in.md) | complete |
| 0044 | MADR | [Let a Caller See Each Retry That WithRetry Waits For](decisions/0044-MADR-observe-retries.md) | accepted |
| 0044 | PLAN | [Implement Letting a Caller See Each Retry That WithRetry Waits For](decisions/0044-PLAN-observe-retries.md) | complete |
| 0045 | MADR | [Make Tests Fail to Connect Without Reusing a Freed Port](decisions/0045-MADR-tests-never-reuse-a-freed-port.md) | accepted |
| 0045 | PLAN | [Implement Making Tests Fail to Connect Without Reusing a Freed Port](decisions/0045-PLAN-tests-never-reuse-a-freed-port.md) | complete |
| 0046 | MADR | [Close v1.15.0's Follow-ups: Answer the Issues, Tell the Consumers, Prove the Person-Gated Live Tests, and Keep the Docs Tables True](decisions/0046-MADR-v1-15-0-follow-ups.md) | accepted |
| 0046 | PLAN | [Implement Closing v1.15.0's Follow-ups](decisions/0046-PLAN-v1-15-0-follow-ups.md) | complete |
| 0047 | MADR | [Keep Five More Scratch Helpers: a Fixture Gate, Appends for `edit.py` and `plant.py`, Record Status, Record Lint, and Claims Plants](decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md) | accepted |
| 0047 | PLAN | [Implement Keeping Five More Scratch Helpers: a Fixture Gate, Appends for `edit.py` and `plant.py`, Record Status, Record Lint, and Claims Plants](decisions/0047-PLAN-keep-five-more-scratch-helpers-as-tools.md) | complete |
| 0048 | MADR | [Close the SDK's Gaps Against Go SDK Practice: Injectable Caches, Runnable Examples, Stated Concurrency and Errors, a Changelog, and Reviewed Exports](decisions/0048-MADR-adopt-go-sdk-best-practices.md) | accepted |
| 0048 | PLAN | [Implement Closing the SDK's Gaps Against Go SDK Practice](decisions/0048-PLAN-adopt-go-sdk-best-practices.md) | complete |
| 0049 | MADR | [Close v1.16.0's Follow-ups: Errors in Place of Two Panics, One Kind for Like Failures, the Learned Refusals in `catalog.Cache`, and the Tests 0048 Left Out](decisions/0049-MADR-v1-16-0-follow-ups.md) | accepted |
| 0049 | PLAN | [Implement Closing v1.16.0's Follow-ups](decisions/0049-PLAN-v1-16-0-follow-ups.md) | complete |
| 0050 | MADR | [Refuse a Typed-Nil Token Source Where It Enters, With `ErrInvalidRequest`](decisions/0050-MADR-refuse-typed-nil-token-sources.md) | accepted |
| 0050 | PLAN | [Implement Refusing a Typed-Nil Token Source Where It Enters](decisions/0050-PLAN-refuse-typed-nil-token-sources.md) | complete |
| 0051 | MADR | [Stream Through a Retry, Opt-In: Retry Before the First Event, Pass Through After](decisions/0051-MADR-opt-in-streaming-retry.md) | accepted |
| 0051 | PLAN | [Implement Streaming Through a Retry, Opt-In](decisions/0051-PLAN-opt-in-streaming-retry.md) | complete |
| 0052 | MADR | [Skip the Full CI for a Push or Pull Request That Changes Only Markdown, and Check It With a One-Job Docs Workflow](decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md) | accepted |
| 0052 | PLAN | [Implement Skipping the Full CI for Docs-Only Pushes, With a Docs Workflow](decisions/0052-PLAN-skip-full-ci-for-docs-only-pushes.md) | complete |
| 0053 | MADR | [Fix Six Defects in Streaming, Retry and the Catalog's Caches](decisions/0053-MADR-stream-retry-and-cache-defects.md) | accepted |
| 0053 | PLAN | [Implement Fixing Six Defects in Streaming, Retry and the Catalog's Caches](decisions/0053-PLAN-stream-retry-and-cache-defects.md) | complete |
| 0054 | REPORT | [Heuristics and Performance Opportunities, 2026-10-20](reports/0054-REPORT-heuristics-and-performance-opportunities.md) | observation |
| 0055 | MADR | [Require Go 1.27.3 for GO-2026-6809, Release It as v1.20.1, and Give v1.20.0 Its GitHub Release by Hand](decisions/0055-MADR-go-1-27-3-and-the-v1-20-0-release.md) | accepted |
| 0055 | PLAN | [Implement Go 1.27.3 as the Floor, Release v1.20.1, and Make v1.20.0's Release](decisions/0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md) | complete |
| 0056 | MADR | [Give the Wizard a Credential-Only Entry Point, Kilo's Device Sign-In as a Key, and Refusals Without the Provider's Text](decisions/0056-MADR-wizard-credential-only-entry-point.md) | accepted |
| 0056 | PLAN | [Implement the Wizard's Credential-Only Entry Point, Kilo's Device Sign-In as a Key, and Refusals Without the Provider's Text](decisions/0056-PLAN-wizard-credential-only-entry-point.md) | complete |
| 0057 | MADR | [Give the Wizard's Device Code to the Caller as Data, Let a Browser Sign-In Run Without a Paste Prompt, and End the Paste Prompt Once the Loopback Wins](decisions/0057-MADR-wizard-device-code-and-paste-prompt.md) | accepted |
| 0057 | PLAN | [Implement the Wizard's Device Code as Data, Browser Sign-In Without a Paste Prompt, and the End of the Paste Prompt Once the Loopback Wins](decisions/0057-PLAN-wizard-device-code-and-paste-prompt.md) | complete |
| 0058 | MADR | [Give Every Test That Points the Catalog's Cache at an httptest Server a Cache of Its Own](decisions/0058-MADR-test-cache-isolation-for-reused-ports.md) | accepted |
| 0058 | PLAN | [Implement a Cache of Its Own for Every Test That Points the Catalog's Cache at an httptest Server](decisions/0058-PLAN-test-cache-isolation-for-reused-ports.md) | complete |
| 0059 | MADR | [Remediate the Wizard's Debugging-Pass Findings, the Claude Key Warning First](decisions/0059-MADR-wizard-debugging-pass-findings.md) | accepted |
| 0059 | PLAN | [Implement the Remediation of the Wizard's Debugging-Pass Findings](decisions/0059-PLAN-wizard-debugging-pass-findings.md) | complete |
| 0060 | MADR | [Bring the Docs Up to Date with v1.22.1's Wizard and the Local Gate](decisions/0060-MADR-docs-accuracy-after-v1-22-1.md) | accepted |
| 0060 | PLAN | [Implement the Docs Update for v1.22.1's Wizard and the Local Gate](decisions/0060-PLAN-docs-accuracy-after-v1-22-1.md) | complete |
| 0061 | MADR | [Close gobble-cli's Consumer Gaps: Empty-Answer and Offline Errors, Max Effort, a Chat Completions Constructor, and Image Input](decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md) | accepted |
| 0061 | PLAN | [Implement gobble-cli's Consumer Gaps: Empty-Answer and Offline Errors, Max Effort, a Chat Completions Constructor, and Image Input](decisions/0061-PLAN-consumer-gaps-from-gobble-issue-9.md) | complete |
| 0062 | MADR | [Decide Thinking Parameters From Facts and Learned Refusals, Not Model Names](decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) | accepted |
| 0062 | PLAN | [Implement Thinking Parameters Decided From Facts and Learned Refusals](decisions/0062-PLAN-learn-thinking-parameters-from-refusals.md) | complete |
| 0062 | REPORT | [Thinking Refusals on Each Wire, Recorded for 0062](reports/0062-REPORT-thinking-refusals.md) | observation |
| 0063 | MADR | [Align docs-only CI and disclosure gates](decisions/0063-MADR-align-docs-ci-and-disclosure-gates.md) | accepted |
| 0063 | PLAN | [Implement aligned docs-only CI and disclosure gates](decisions/0063-PLAN-align-docs-ci-and-disclosure-gates.md) | complete |
| 0064 | MADR | [Send Anthropic's Workspace Header for Keys Not Scoped to a Workspace](decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md) | accepted |
| 0064 | PLAN | [Implement Anthropic's Workspace Header for Keys Not Scoped to a Workspace](decisions/0064-PLAN-anthropic-workspace-for-unscoped-keys.md) | complete |
| 0065 | MADR | [Keep the Tests Off the Network, and Hold Them There](decisions/0065-MADR-tests-reach-no-outside-host.md) | accepted |
| 0065 | PLAN | [Implement Keeping the Tests Off the Network](decisions/0065-PLAN-tests-reach-no-outside-host.md) | complete |

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
| know how native streaming is being added, starting with the Responses wire | [0031-MADR](decisions/0031-MADR-native-streaming-on-the-responses-wire.md) |
| opt a tool out of OpenAI's strict mode, or mark a tool result as failed | [0032-MADR](decisions/0032-MADR-tool-strict-and-result-error-flags.md) |
| read a model's context window, output limit, prices or modalities | [0033-MADR](decisions/0033-MADR-public-model-metadata-and-current-catalogs.md) |
| know why a model is or is not in a provider's menu | [0033-PLAN-refresh-the-curated-catalogs.md](decisions/0033-PLAN-refresh-the-curated-catalogs.md) |
| ask a provider to cache a prompt | [0039-MADR-request-side-prompt-caching.md](decisions/0039-MADR-request-side-prompt-caching.md) |
| offer many tools but load only the ones a search finds, keeping the cache | [0040-MADR-native-tool-deferral.md](decisions/0040-MADR-native-tool-deferral.md) |
| know how a provider's live model menu is ordered, what its caveats mean, or what it shows when a listing fails | [0041-MADR-decide-per-model-behaviour-from-metadata.md](decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md) |
| know why a model's thinking, effort or output limit comes from its facts | [0041-MADR-decide-per-model-behaviour-from-metadata.md](decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md) |
| get each provider's model listing exactly as the provider sent it, with none of the SDK's curation | [0041-MADR-decide-per-model-behaviour-from-metadata.md](decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md), D9 |
| know why Ollama's listing leaves out embedding models but keeps chat models that take no tools | [0042-MADR-ollama-models-that-write-text.md](decisions/0042-MADR-ollama-models-that-write-text.md) |
| sign a user in with their ChatGPT plan under my own app's name | [0043-MADR-openai-chatgpt-plan-sign-in.md](decisions/0043-MADR-openai-chatgpt-plan-sign-in.md) |
| tell my user that a request is waiting to be retried, and for how long | [0044-MADR-observe-retries.md](decisions/0044-MADR-observe-retries.md) |
| write a test that needs a refused connection or a free port | [0045-MADR-tests-never-reuse-a-freed-port.md](decisions/0045-MADR-tests-never-reuse-a-freed-port.md) |
| know what v1.15.0 left to do, and how each follow-up is closed | [0046-MADR-v1-15-0-follow-ups.md](decisions/0046-MADR-v1-15-0-follow-ups.md) |
| list every model a provider serves, or only the top six | [0038-MADR-full-curated-catalog-by-default.md](decisions/0038-MADR-full-curated-catalog-by-default.md) |
| know how GenerateToolCall forces a tool on the Claude models that refuse it | [0034-MADR](decisions/0034-MADR-degrade-forced-tool-choice-per-model.md) |
| run the local gate, the link check or the identifier scan myself | [0035-MADR](decisions/0035-MADR-maintainer-tools-in-python.md) |
| price a response's cache writes, or know what each `Usage` count includes | [0036-MADR](decisions/0036-MADR-cache-write-token-count.md) |
| know why a Claude response now reports reasoning tokens | [0037-MADR](decisions/0037-MADR-reasoning-and-cache-read-spellings.md) |
| know why provider auth builds on `mcplib` and not `magic-cli-remote`, and what it takes from each | [0016-MADR](decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| compare this module's auth with Codex, Grok, Kilo, OpenCode, pi, agy and Claude Code | [0017-REPORT](reports/0017-REPORT-reference-client-auth-survey.md) |
| know how OpenAI's Sign in with ChatGPT for agents works, what its terms require, and what this module lacks for it | [0017-REPORT-openai-sign-in-for-agents.md](reports/0017-REPORT-openai-sign-in-for-agents.md) |
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
