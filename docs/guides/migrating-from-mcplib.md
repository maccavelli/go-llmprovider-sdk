# Migrating from mcplib

How code that used `mcplib`'s `llmprovider` and `wizard` packages moves to this
module. The table maps every exported identifier of `mcplib` `v1.6.0` to its
equivalent here, or records its removal and why.

Every row has an "SDK equivalent": the identifier here, `removed`, or what
replaces it. "Unchanged" in the notes means the same name, type and behaviour.
`make parity-check` fails if an identifier has no row, if a row names one that
`mcplib` `v1.6.0` did not export, or if a cell is empty.

## Import paths

| `mcplib` | Here |
| :--- | :--- |
| `github.com/maccavelli/mcplib/llmprovider` | `github.com/maccavelli/go-llmprovider-sdk/llmprovider` |
| `github.com/maccavelli/mcplib/wizard` | `github.com/maccavelli/go-llmprovider-sdk/wizard` |
| `github.com/maccavelli/mcplib/logging` (`RedactString`, `MaskSecret`) | stays in `mcplib`; the copy here is internal and cannot be imported |

## Behaviour that has already changed

Recorded in [0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md):

- **Identity (§5).** Requests name `go-llmprovider-sdk` in `User-Agent`, the
  ChatGPT `originator` header and the Grok `referrer`.
- **Environment (§6).** `MCPLIB_*` variables are now `LLMPROVIDER_*`.
  Since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S10 the library reads none of them by itself:
  - pass `catalog.OptionsFromEnv()` for the metadata variables;
  - pass `auth.GrokFlowFromEnv()` for `GROK_OAUTH2_ISSUER` and
    `GROK_OAUTH2_CLIENT_ID`;
  - pass `os.Getenv` as the wizard's `Options.LookupEnv`.
- **Claude's key (0016-MADR D12).** It is read from `ANTHROPIC_API_KEY`
  only, no longer from `CLAUDE_API_KEY`.
- **Orchestration (sixth amendment).** The wizard has no orchestration option,
  and does not read `MCP_ORCHESTRATOR_OWNED`.
- **Credentials (§7).** Without `Options.TokenStore`, the wizard offers only
  the API key.
- **Go (fifth amendment).** The module requires Go 1.27.1.

## Identifier map

### `llmprovider`

| `mcplib` identifier | SDK equivalent | Notes |
| :--- | :--- | :--- |
| `llmprovider.APIError` | `APIError` | The only structured error: it absorbs `RateLimitError` and `IncompleteError`, and adds `Kind`, `Code`, `Reason` and `Retryable()` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7). |
| `llmprovider.APIError.Error` | `APIError.Error` | Reads `<kind>: <provider> HTTP <status> <code> (retry-after <d>): <reason>: <message>`, leaving out what is unset. The kind's text starts `llmprovider:` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7). |
| `llmprovider.APIError.Message` | `APIError.Message` | Unchanged: redacted and bounded to 512 bytes. An error body that cannot be read is named in it ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S10). |
| `llmprovider.APIError.Provider` | `APIError.Provider` | Unchanged: the provider, or `gateway/route` for a gateway. A `string`, not a `ProviderID`. |
| `llmprovider.APIError.RetryAfter` | `APIError.RetryAfter` | Unchanged. |
| `llmprovider.APIError.Status` | `APIError.Status` | Unchanged: 0 for a failure inside a 200 event stream. |
| `llmprovider.APIError.Terminal` | `APIError.Retryable()` | Unexported in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8; `Retryable()` reads it. |
| `llmprovider.APIError.Type` | `APIError.Code` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8; `Code` carries the same value. |
| `llmprovider.APIError.Unwrap` | `APIError.Unwrap` | Returns the `Kind` and the pre-0012 status sentinel, as before. `ErrIncomplete` and `ErrContextOverflow` sit beneath `ErrInvalidRequest` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7 and its amendments). |
| `llmprovider.ApplyOptions` | `ResolveOptions(id, opts)` | A provider's `New` resolves its options to read-only `Settings`. Unexported in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 (R19). |
| `llmprovider.AuthAPIKey` | `AuthAPIKey` | Unchanged, with its value. |
| `llmprovider.AuthBrowserOAuth` | `AuthBrowserOAuth` | Unchanged, with its value. |
| `llmprovider.AuthDeviceCode` | `AuthDeviceCode` | Unchanged, with its value. |
| `llmprovider.AuthImportVendorCLI` | `AuthImportVendorCLI` | Unchanged, with its value. |
| `llmprovider.AuthMethod` | `AuthMethod` | Unchanged. A `Descriptor`'s `AuthMethods` list them; each provider package declares its own descriptor. |
| `llmprovider.AuthMethod.Detail` | `AuthMethod.Detail` | Unchanged. |
| `llmprovider.AuthMethod.HeadlessOK` | `AuthMethod.HeadlessOK` | Unchanged. |
| `llmprovider.AuthMethod.ID` | `AuthMethod.ID` | Unchanged. |
| `llmprovider.AuthMethod.Interactive` | `AuthMethod.Interactive` | Unchanged. |
| `llmprovider.AuthMethod.Label` | `AuthMethod.Label` | Unchanged. |
| `llmprovider.AuthMethodID` | `AuthMethodID` | Unchanged. |
| `llmprovider.AuthTokenStdin` | `AuthTokenStdin` | Unchanged, with its value. |
| `llmprovider.ClaudeProvider` | the `llmprovider.Provider` that `claude.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.ClaudeProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` |  |
| `llmprovider.ClaudeProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.ClaudeProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.ClaudeProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithThinkingBudget` and `WithReasoningEffort` are `Reasoning`'s `Budget` and `Effort`. |
| `llmprovider.ClaudeProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.ClaudeProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` | The forced tool is sent as `auto` while thinking, as before. |
| `llmprovider.ClaudeProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.ClaudeProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.ClaudeProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.ClaudeProvider.Name` | `ID()` | It returns `ProviderClaude` as a `ProviderID`. |
| `llmprovider.Continuer` | `Request.PreviousResponseID` | `Capabilities.Continuation` says whether a provider supports it. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.Continuer.Continue` | `Provider.Generate` with `Request.PreviousResponseID` | As `Continuer`. |
| `llmprovider.DefaultGrokBaseURL` | `grok.BaseURL` | In `providers/grok` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultGrokOAuthClientID` | `auth.DefaultGrokOAuthClientID` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultGrokOAuthIssuer` | `auth.DefaultGrokOAuthIssuer` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultOpenAIChatGPTBaseURL` | `openai.ChatGPTBaseURL` | In `providers/openai` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultOpenAIClientID` | `auth.DefaultOpenAIClientID` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultOpenAIIssuer` | `auth.DefaultOpenAIIssuer` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DefaultOpenAIPlatformBaseURL` | `openai.PlatformBaseURL` | In `providers/openai` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.DescriptorFor` | `Registry.Descriptor(id)` | On `providers.Default()`, or the caller's `Registry` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.Descriptors` | `Registry.Descriptors()` | In menu order; each provider package declares its own descriptor ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrAuthFailure` | `ErrAuthFailure` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrInvalidProvider` | `ErrInvalidProvider` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrInvalidRequest` | `ErrInvalidRequest` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrNotPermitted` | `ErrNotPermitted` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrProviderUnavailable` | `ErrProviderUnavailable` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrQuotaExhausted` | `ErrQuotaExhausted` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrRateLimited` | `ErrRateLimited` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.FileTokenStore` | `auth.FileTokenStore` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.FileTokenStore.Delete` | `auth.FileTokenStore.Delete` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.FileTokenStore.Dir` | `auth.FileTokenStore.Dir` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.FileTokenStore.Load` | `auth.FileTokenStore.Load` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.FileTokenStore.Save` | `auth.FileTokenStore.Save` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.FunctionCallItem` | `FunctionCallItem` | Unchanged. `GenerateToolCall` returns one. |
| `llmprovider.FunctionCallItem.Arguments` | `FunctionCallItem.Arguments` | Unchanged. |
| `llmprovider.FunctionCallItem.CallID` | `FunctionCallItem.CallID` | Unchanged. |
| `llmprovider.FunctionCallItem.Name` | `FunctionCallItem.Name` | Unchanged. |
| `llmprovider.FunctionCallItem.Signature` | `FunctionCallItem.Signature` | Unchanged. |
| `llmprovider.FunctionCallOutputItem` | `FunctionCallOutputItem` | Unchanged. |
| `llmprovider.FunctionCallOutputItem.CallID` | `FunctionCallOutputItem.CallID` | Unchanged. |
| `llmprovider.FunctionCallOutputItem.Output` | `FunctionCallOutputItem.Output` | Unchanged. |
| `llmprovider.GeminiProvider` | the `llmprovider.Provider` that `gemini.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.GeminiProvider.Continue` | `Generate` with `Request.PreviousResponseID` | Needs `gemini.WithStore(true)`; without it continuation is `Unsupported`, and refused with `ErrUnsupported` where it was `ErrInvalidRequest` (0015-MADR D4). |
| `llmprovider.GeminiProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` |  |
| `llmprovider.GeminiProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.GeminiProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.GeminiProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` is `Reasoning.Effort`; the Interactions API has no budget. |
| `llmprovider.GeminiProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.GeminiProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.GeminiProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.GeminiProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.GeminiProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.GeminiProvider.Name` | `ID()` | It returns `ProviderGemini` as a `ProviderID`. |
| `llmprovider.GenerateItemsWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateThinkingWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`, with `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GrokProvider` | the `llmprovider.Provider` that `grok.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.GrokProvider.Continue` | `Generate` with `Request.PreviousResponseID` |  |
| `llmprovider.GrokProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` |  |
| `llmprovider.GrokProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.GrokProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.GrokProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` is `Reasoning.Effort`, clamped to the model's menu as before. |
| `llmprovider.GrokProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.GrokProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.GrokProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.GrokProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.GrokProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.GrokProvider.Name` | `ID()` | It returns `ProviderGrok` as a `ProviderID`. |
| `llmprovider.HuggingFaceProvider` | the `llmprovider.Provider` that `huggingface.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.HuggingFaceProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` | No profile can be chosen on the new API until S8b; the wizard is unaffected. |
| `llmprovider.HuggingFaceProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.HuggingFaceProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.HuggingFaceProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` is `Reasoning.Effort`. |
| `llmprovider.HuggingFaceProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.HuggingFaceProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.HuggingFaceProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.HuggingFaceProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.HuggingFaceProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.HuggingFaceProvider.Name` | `ID()` | It returns `ProviderHuggingFace` as a `ProviderID`. |
| `llmprovider.IncompleteError` | `*APIError` of kind `ErrIncomplete` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7). It still matches `ErrInvalidRequest`. |
| `llmprovider.IncompleteError.Error` | `APIError.Error` | Reads `llmprovider: incomplete response: <reason>`. |
| `llmprovider.IncompleteError.Reason` | `APIError.Reason` | As `IncompleteError`. |
| `llmprovider.IncompleteError.Unwrap` | `APIError.Unwrap` | `ErrIncomplete` and `ErrInvalidRequest`, as before. |
| `llmprovider.Item` | `Item` | Still sealed. Code that switches on it has a `default` case (R9). |
| `llmprovider.ItemProvider` | `Provider` | Every provider takes items. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ItemProvider.GenerateItems` | `Provider.Generate` with `Request.Input` | As `ItemProvider`. |
| `llmprovider.ItemThinkingProvider` | `Provider` | Reasoning is `Request.Reasoning`, or `WithReasoning` at construction; `Capabilities.Reasoning` says whether it is supported. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ItemThinkingProvider.GenerateItemsThinking` | `Provider.Generate` with `Request.Input` and `Request.Reasoning` | As `ItemThinkingProvider`. |
| `llmprovider.ItemThinkingToolProvider` | `Provider` | As `ItemThinkingProvider` and `ItemToolProvider`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ItemThinkingToolProvider.GenerateItemsWithToolThinking` | `Provider.Generate` with `Request.Input`, `Request.Tools` and `Request.Reasoning` | As `ItemThinkingToolProvider`. |
| `llmprovider.ItemToolProvider` | `Provider` | Tools are `Request.Tools` and `ToolChoice`, or `GenerateToolCall`; `Capabilities.Tools` says whether they are supported. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ItemToolProvider.GenerateItemsWithTool` | `Provider.Generate` with `Request.Input` and `Request.Tools` | As `ItemToolProvider`. |
| `llmprovider.KiloModelCapabilities` | `catalog.KiloModelCapabilities` |  |
| `llmprovider.KiloProvider` | the `llmprovider.Provider` that `kilo.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.KiloProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` | No profile can be chosen on the new API until S8b; the wizard is unaffected. |
| `llmprovider.KiloProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.KiloProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.KiloProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` is `Reasoning.Effort`. |
| `llmprovider.KiloProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` | The choice is sent only when the model accepts `tool_choice`, as before. |
| `llmprovider.KiloProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.KiloProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.KiloProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.KiloProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.KiloProvider.Name` | `ID()` | It returns `ProviderKilo` as a `ProviderID`. |
| `llmprovider.ListAvailableModels` | `catalog.List(ctx, id, NewStaticToken(key), opts...)` | Its `Recommended`; an error gives none ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ListAvailableModelsWithSource` | `catalog.List(ctx, id, src, opts...)` | Its `Recommended`; an error gives none. |
| `llmprovider.ListModelCatalog` | `catalog.List` | With `NewStaticToken(key)` for a key ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ListModelCatalogWithSource` | `catalog.List` |  |
| `llmprovider.LoginBrowserOAuth` | `auth.LoginBrowserOAuth` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.LoginDeviceOAuth` | `auth.LoginDeviceOAuth` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.MaxListedModels` | `catalog.MaxListed` |  |
| `llmprovider.MessageItem` | `MessageItem` | Its `Role` is the named type `Role` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). |
| `llmprovider.MessageItem.Role` | `MessageItem.Role` | A `Role` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). A role other than `RoleUser`, `RoleAssistant`, `RoleSystem` or empty is refused with `ErrInvalidRequest`. |
| `llmprovider.MessageItem.Text` | `MessageItem.Text` | Unchanged. |
| `llmprovider.ModelCatalog` | `catalog.Catalog` |  |
| `llmprovider.ModelCatalog.Err` | `Catalog.Err` |  |
| `llmprovider.ModelCatalog.Live` | `Catalog.Live` |  |
| `llmprovider.ModelCatalog.Recommended` | `Catalog.Recommended` |  |
| `llmprovider.ModelCatalog.Usable` | `Catalog.Usable` |  |
| `llmprovider.ModelDiscoverer` | `ModelLister` | Or `catalog.List`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ModelDiscoverer.DiscoverModels` | `ModelLister.ListModels` | As `ModelDiscoverer`. |
| `llmprovider.ModelLabel` | `catalog.Label` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). |
| `llmprovider.ModelMatch` | `catalog.Match` |  |
| `llmprovider.ModelMatch.ID` | `Match.ID` |  |
| `llmprovider.ModelMatch.Label` | `Match.Label` |  |
| `llmprovider.ModelMatch.Score` | `Match.Score` |  |
| `llmprovider.ModelProfile` | `catalog.Profile` | Passed to a listing or a provider with `catalog.WithProfile`. |
| `llmprovider.ModelProfile.ReasoningEffort` | `Profile.ReasoningEffort` |  |
| `llmprovider.NewClaude` | `claude.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderClaude, …)`; an OAuth session is refused with `ErrUnsupported` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewFileTokenStore` | `auth.NewFileTokenStore` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.NewGemini` | `gemini.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderGemini, …)`. It takes no context; an OAuth session is refused with `ErrUnsupported` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewGrok` | `grok.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderGrok, …)`. A session, which only `NewProvider` took, is `WithTokenSource(src)` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewHuggingFace` | `huggingface.New(WithAPIKey(token), WithModel(model), …)` | Or `providers.New(ProviderHuggingFace, …)`; an empty token is refused with `ErrInvalidRequest` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewKilo` | `kilo.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderKilo, …)`. No key sends the anonymous token, as before ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewOllama` | `ollama.New(WithModel(model), …)` | Or `providers.New(ProviderOllama, …)`. It needs no key, as before; `WithBaseURL` names the instance ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewOpenAI` | `openai.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderOpenAI, …)` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewOpenAIWithSource` | `openai.New(WithTokenSource(src), WithModel(model), …)` | A ChatGPT session selects the ChatGPT backend ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewOpencode` | `opencode.NewZen(…)` or `opencode.NewGo(…)`, with `WithAPIKey(key)` and `WithModel(model)` | Or `providers.New(ProviderOpencodeZen` or `ProviderOpencodeGo, …)`; one constructor per gateway ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). No key sends the public token, as before. |
| `llmprovider.NewProvider` | `providers.New(id, opts...)` | Over `providers.Default()`, a `Registry`; removed at the start of [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7. |
| `llmprovider.NewProviderWithSource` | `providers.New(id, WithTokenSource(src), …)` | Removed at the start of [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7; a credential is an option (D5). |
| `llmprovider.NewStaticToken` | `NewStaticToken` | Its token is a `TokenAPIKey`, sent in the service's own header unless `Header` names another ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6); see `StaticToken.Token`. |
| `llmprovider.OAuthFlowOptions` | `auth.OAuthFlowOptions` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.ClientID` | `auth.OAuthFlowOptions.ClientID` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.HTTPClient` | `auth.OAuthFlowOptions.HTTPClient` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.InputCode` | `auth.OAuthFlowOptions.InputCode` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.Issuer` | `auth.OAuthFlowOptions.Issuer` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.NotifyDevice` | `auth.OAuthFlowOptions.NotifyDevice` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthFlowOptions.OpenURL` | `auth.OAuthFlowOptions.OpenURL` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession` | `auth.OAuthSession` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Access` | `auth.OAuthSession.Access` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.AccountID` | `auth.OAuthSession.AccountID` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.ChatGPT` | `auth.OAuthSession.ChatGPT` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.ClientID` | `auth.OAuthSession.ClientID` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Expiry` | `auth.OAuthSession.Expiry` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.FedRAMP` | `auth.OAuthSession.FedRAMP` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.HTTPClient` | `auth.OAuthSession.HTTPClient` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Issuer` | `auth.OAuthSession.Issuer` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Provider` | `auth.OAuthSession.Provider` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). Its JSON is unchanged. In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Refresh` | `auth.OAuthSession.Refresh` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Store` | `auth.OAuthSession.Store` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.Token` | `auth.OAuthSession.Token` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OAuthSession.TokenURL` | `auth.OAuthSession.TokenURL` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.OllamaProvider` | the `llmprovider.Provider` that `ollama.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.OllamaProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` | It still probes by default; `WithModelProbes(false)` turns that off. |
| `llmprovider.OllamaProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.OllamaProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.OllamaProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` is `Reasoning.Effort`; `EffortXHigh` is still sent as `max`. |
| `llmprovider.OllamaProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` | The tools are offered, never forced, as before. |
| `llmprovider.OllamaProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools` and `Reasoning` |  |
| `llmprovider.OllamaProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.OllamaProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. The tool is still offered, not forced. |
| `llmprovider.OllamaProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.OllamaProvider.Name` | `ID()` | It returns `ProviderOllama` as a `ProviderID`. |
| `llmprovider.OpenAIProvider` | the `llmprovider.Provider` that `openai.New` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.OpenAIProvider.Continue` | `Generate` with `Request.PreviousResponseID` | A ChatGPT session refuses it with `ErrUnsupported` (0015-MADR D4). |
| `llmprovider.OpenAIProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` |  |
| `llmprovider.OpenAIProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.OpenAIProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.OpenAIProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` |  |
| `llmprovider.OpenAIProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.OpenAIProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.OpenAIProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.OpenAIProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.OpenAIProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.OpenAIProvider.Name` | `ID()` | It returns `ProviderOpenAI` as a `ProviderID`. |
| `llmprovider.OpencodeProvider` | the `llmprovider.Provider` that `opencode.NewZen` or `NewGo` returns | The type is unexported ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.OpencodeProvider.DiscoverModels` | `ListModels`, through `llmprovider.ModelLister` | No profile can be chosen on the new API until S8b; the wizard is unaffected. |
| `llmprovider.OpencodeProvider.Generate` | `llmprovider.GenerateText` |  |
| `llmprovider.OpencodeProvider.GenerateItems` | `Generate` with `Request.Input` |  |
| `llmprovider.OpencodeProvider.GenerateItemsThinking` | `Generate` with `Request.Reasoning` | `WithReasoningEffort` and `WithThinkingBudget` are `Reasoning`'s `Effort` and `Budget`. |
| `llmprovider.OpencodeProvider.GenerateItemsWithTool` | `Generate` with `Request.Tools` and `ForceTool` |  |
| `llmprovider.OpencodeProvider.GenerateItemsWithToolThinking` | `Generate` with `Tools`, `ForceTool` and `Reasoning` |  |
| `llmprovider.OpencodeProvider.GenerateThinking` | `llmprovider.GenerateText` with `Request.Reasoning` |  |
| `llmprovider.OpencodeProvider.GenerateWithTool` | `llmprovider.GenerateToolCall` | It returns the call; its `Arguments` are the old result. |
| `llmprovider.OpencodeProvider.GenerateWithToolThinking` | `llmprovider.GenerateToolCall` with `Request.Reasoning` |  |
| `llmprovider.OpencodeProvider.Name` | `ID()` | It returns the gateway's `ProviderID`. |
| `llmprovider.OpencodeProvider.Route` | none | The route is chosen per request, by model; `opencode.WithRoute` pins one ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.OpencodeRoute` | `opencode.Route` |  |
| `llmprovider.OpencodeRouteChatCompletions` | `opencode.RouteChatCompletions` |  |
| `llmprovider.OpencodeRouteGoogle` | `opencode.RouteGoogle` |  |
| `llmprovider.OpencodeRouteMessages` | `opencode.RouteMessages` |  |
| `llmprovider.OpencodeRouteResponses` | `opencode.RouteResponses` |  |
| `llmprovider.ProfileCapable` | `catalog.ProfileCapable` |  |
| `llmprovider.ProfileUtility` | `catalog.ProfileUtility` |  |
| `llmprovider.Provider` | `Provider` | `ID`, `Capabilities` and `Generate(ctx, *Request)`. The old interface, `LegacyProvider` after the move, is removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.Provider.Generate` | `GenerateText(ctx, p, req)` | Or `Provider.Generate` for the whole `*Response`. |
| `llmprovider.Provider.Name` | `Provider.ID` | A `ProviderID`. |
| `llmprovider.ProviderClaude` | `ProviderClaude` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderConfig` | `Settings` | Unexported in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 (R19); a provider reads its resolved `Settings`. |
| `llmprovider.ProviderConfig.BaseURL` | `Settings.BaseURL` | Set with `WithBaseURL`. |
| `llmprovider.ProviderConfig.ClientName` | `Settings.ClientName` | Set with `WithClientInfo`. |
| `llmprovider.ProviderConfig.ClientVersion` | `Settings.UserAgent` | Set with `WithClientInfo`. |
| `llmprovider.ProviderConfig.HTTPClient` | `Settings.HTTPClient` | Set with `WithHTTPClient`. |
| `llmprovider.ProviderConfig.KiloCapabilities` | `kilo.WithCapabilities` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ProviderConfig.KiloDataCollection` | `kilo.WithDataCollection` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ProviderConfig.KiloOrganization` | `kilo.WithOrganization`, `catalog.WithKiloOrganization` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, commit 2. |
| `llmprovider.ProviderConfig.MaxTokens` | `Settings.MaxTokens` | Set with `WithMaxTokens`. |
| `llmprovider.ProviderConfig.ModelMetadataURL` | `Settings.ModelMetadataURL` | Set with `WithModelMetadataURL`. |
| `llmprovider.ProviderConfig.ModelProfile` | `catalog.WithProfile` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, commit 2. |
| `llmprovider.ProviderConfig.OpencodeRoute` | `opencode.WithRoute` | Removed with the old provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.ProviderConfig.ReasoningEffort` | `Reasoning.Effort` | Through `WithReasoning` or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ProviderConfig.SessionID` | `Settings.SessionID` | Set with `WithSessionID`. |
| `llmprovider.ProviderConfig.Store` | `openai.WithStore`, `gemini.WithStore` or `grok.WithStore` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ProviderConfig.ThinkingBudget` | `Reasoning.Budget` | Through `WithReasoning` or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ProviderDescriptor` | `llmprovider.Descriptor` | The same fields; `ID` is a `ProviderID`. |
| `llmprovider.ProviderDescriptor.AuthMethods` | `Descriptor.AuthMethods` |  |
| `llmprovider.ProviderDescriptor.DefaultBaseURL` | `Descriptor.DefaultBaseURL` |  |
| `llmprovider.ProviderDescriptor.EnvVar` | `Descriptor.EnvVar` |  |
| `llmprovider.ProviderDescriptor.ID` | `Descriptor.ID` | A `ProviderID`. |
| `llmprovider.ProviderDescriptor.IsLocal` | `Descriptor.IsLocal` |  |
| `llmprovider.ProviderDescriptor.Label` | `Descriptor.Label` |  |
| `llmprovider.ProviderDescriptor.Notes` | `Descriptor.Notes` |  |
| `llmprovider.ProviderDescriptor.RequiresAPIKey` | `Descriptor.RequiresAPIKey` |  |
| `llmprovider.ProviderDescriptor.StaticModels` | `Descriptor.StaticModels` |  |
| `llmprovider.ProviderDescriptor.SupportsBaseURL` | `Descriptor.SupportsBaseURL` |  |
| `llmprovider.ProviderEnvVars` | `ProviderEnvVars()` | A function returning a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9), keyed by `ProviderID` since S8. |
| `llmprovider.ProviderGemini` | `ProviderGemini` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderGrok` | `ProviderGrok` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderHuggingFace` | `ProviderHuggingFace` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderKilo` | `ProviderKilo` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderOllama` | `ProviderOllama` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderOpenAI` | `ProviderOpenAI` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderOpencodeGo` | `ProviderOpencodeGo` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderOpencodeZen` | `ProviderOpencodeZen` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The value is unchanged. |
| `llmprovider.ProviderOption` | `Option` | The alias is removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.RankClaudeModel` | `catalog.Rank(ProviderClaude, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGeminiModel` | `catalog.Rank(ProviderGemini, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGrokModel` | `catalog.Rank(ProviderGrok, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankHuggingFaceModel` | `catalog.Rank(ProviderHuggingFace, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankKiloModel` | `catalog.Rank(ProviderKilo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpenAIModel` | `catalog.Rank(ProviderOpenAI, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpencodeModel` | `catalog.Rank(ProviderOpencodeZen, model)` or `catalog.Rank(ProviderOpencodeGo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RateLimitError` | `*APIError` of kind `ErrRateLimited` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7). A 429 and a stream rate limit are both an `*APIError`. |
| `llmprovider.RateLimitError.Error` | `APIError.Error` | Reads `llmprovider: rate limited: <provider> HTTP 429 (retry-after <d>): <message>`. |
| `llmprovider.RateLimitError.Message` | `APIError.Message` | As `RateLimitError`. |
| `llmprovider.RateLimitError.Provider` | `APIError.Provider` | As `RateLimitError`. |
| `llmprovider.RateLimitError.RetryAfter` | `APIError.RetryAfter` | As `RateLimitError`. |
| `llmprovider.RateLimitError.Status` | `APIError.Status` | 0 for a stream rate limit. |
| `llmprovider.RateLimitError.Unwrap` | `APIError.Unwrap` | Matches `ErrRateLimited`, as before. |
| `llmprovider.ReasoningItem` | `ReasoningItem` | Unchanged. |
| `llmprovider.ReasoningItem.Text` | `ReasoningItem.Text` | Unchanged. |
| `llmprovider.Response` | `Response` | Adds `Model` and `Usage`; `FinishReason` is the named type `FinishReason`. |
| `llmprovider.Response.FinishReason` | `Response.FinishReason` | The named type `FinishReason` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). The Chat Completions providers set it from the service's `finish_reason`, as before; compare it with `FinishStop`, `FinishLength`, `FinishToolCalls` or `FinishContentFilter`. |
| `llmprovider.Response.ID` | `Response.ID` | Unchanged. Continue from it with `Request.PreviousResponseID`. |
| `llmprovider.Response.Items` | `Response.Items` | Unchanged. |
| `llmprovider.Response.Output` | `Response.Output` | Unchanged. |
| `llmprovider.Response.OutputText` | `Response.OutputText` | Unchanged. `GenerateText` returns it. |
| `llmprovider.RevokeOAuthSession` | `auth.RevokeOAuthSession` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.SearchModels` | `catalog.Search` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). |
| `llmprovider.StaticClaude` | `StaticModels(ProviderClaude)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGemini` | `StaticModels(ProviderGemini)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGrok` | `StaticModels(ProviderGrok)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticHuggingFace` | `StaticModels(ProviderHuggingFace)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticKilo` | `StaticModels(ProviderKilo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticModels` | `catalog.Static` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). |
| `llmprovider.StaticOpenAI` | `StaticModels(ProviderOpenAI)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeGo` | `StaticModels(ProviderOpencodeGo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeZen` | `StaticModels(ProviderOpencodeZen)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticToken` | `StaticToken` | `String`, `GoString`, `LogValue` and `MarshalJSON` never show `Value` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D5). |
| `llmprovider.StaticToken.Header` | `llmprovider.StaticToken.Header` | Empty now means the service's own header; it no longer defaults to `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Token` | `llmprovider.StaticToken.Token` | Returns a `TokenAPIKey`, with the `Header` set, if any; it returned `TokenBearer` and `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Value` | `StaticToken.Value` | Unchanged. |
| `llmprovider.ThinkingProvider` | `Provider` | Reasoning is `Request.Reasoning`, or `WithReasoning` at construction. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ThinkingProvider.GenerateThinking` | `GenerateText` with `Request.Reasoning` | As `ThinkingProvider`. |
| `llmprovider.ThinkingToolProvider` | `Provider` | As `ThinkingProvider` and `ToolProvider`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ThinkingToolProvider.GenerateWithToolThinking` | `GenerateToolCall` with `Request.Reasoning` | As `ThinkingToolProvider`. |
| `llmprovider.Token` | `Token` | Adds `Apply`, which a provider calls to send it (R16), and `String`, `GoString`, `LogValue` and `MarshalJSON`, which never show `Value` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D2, D5, A6). |
| `llmprovider.Token.Expiry` | `Token.Expiry` | Unchanged. |
| `llmprovider.Token.Header` | `llmprovider.Token.Header` | A non-empty `Header` overrides the service's header (R16, [0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.Token.Type` | `llmprovider.Token.Type` | With an overriding `Header`, `TokenBearer` is sent `Bearer`-prefixed and anything else bare ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.Token.Value` | `Token.Value` | Unchanged. |
| `llmprovider.TokenAPIKey` | `TokenAPIKey` | Key sources return it. In an overriding `Header` it is sent bare ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.TokenBearer` | `TokenBearer` | Sessions return it. In an overriding `Header` it is sent as `Bearer <value>` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.TokenSource` | `TokenSource` | Unchanged. |
| `llmprovider.TokenSource.Token` | `TokenSource.Token` | Unchanged. |
| `llmprovider.TokenStore` | `auth.TokenStore` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.TokenStore.Delete` | `auth.TokenStore.Delete` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). An implementation outside this module changes its signature. In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.TokenStore.Load` | `auth.TokenStore.Load` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). An implementation outside this module changes its signature. In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.TokenStore.Save` | `auth.TokenStore.Save` | Its provider parameter is a `ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). An implementation outside this module changes its signature. In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.TokenType` | `TokenType` | Unchanged. |
| `llmprovider.Tool` | `Tool` | Unchanged. It goes in `Request.Tools`. |
| `llmprovider.Tool.Description` | `Tool.Description` | Unchanged. |
| `llmprovider.Tool.Name` | `Tool.Name` | Unchanged. |
| `llmprovider.Tool.Schema` | `Tool.Schema` | Unchanged. |
| `llmprovider.ToolProvider` | `Provider` | Tools are `Request.Tools` and `ToolChoice`, or `GenerateToolCall`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ToolProvider.GenerateWithTool` | `GenerateToolCall(ctx, p, req)` | As `ToolProvider`. |
| `llmprovider.ValidateOAuthSession` | `auth.ValidateOAuthSession` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.ValidateOllamaURL` | `catalog.ValidateOllamaURL` |  |
| `llmprovider.VendorCLISession` | `auth.VendorCLISession` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.VendorCLISession.Path` | `auth.VendorCLISession.Path` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.VendorCLISession.Provider` | `auth.VendorCLISession.Provider` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.VendorCLISession.Token` | `auth.VendorCLISession.Token` | In `auth` since [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c. |
| `llmprovider.WithBaseURL` | `WithBaseURL` | Returns an `Option`, common to every provider ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5); `For(id, …)` scopes it to one. |
| `llmprovider.WithClientInfo` | `WithClientInfo` | Returns an `Option`, common to every provider ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5); `For(id, …)` scopes it to one. An empty name keeps `go-llmprovider-sdk` ([0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) §5). |
| `llmprovider.WithHTTPClient` | `WithHTTPClient` | Returns an `Option`, common to every provider ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5); `For(id, …)` scopes it to one. |
| `llmprovider.WithKiloCapabilities` | `kilo.WithCapabilities` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithKiloDataCollection` | `kilo.WithDataCollection` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithKiloOrganization` | `kilo.WithOrganization`, and `catalog.WithKiloOrganization` for a listing | `kilo`'s `ListModels` passes its organization on ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.WithMaxTokens` | `WithMaxTokens` | Returns an `Option`, common to every provider ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5); `For(id, …)` scopes it to one. |
| `llmprovider.WithModelMetadataURL` | `llmprovider.WithModelMetadataURL` | A common option now: the new API takes it too ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithModelProfile` | `catalog.WithProfile` | Every built-in provider's `New` takes it; the open catalogs rank with it. |
| `llmprovider.WithOpencodeRoute` | `opencode.WithRoute` | Scoped to both gateways ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithReasoningEffort` | `WithReasoning(&Reasoning{Effort: …})` | Or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithSessionID` | `WithSessionID` | Returns an `Option`, common to every provider ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5); `For(id, …)` scopes it to one. |
| `llmprovider.WithStore` | `openai.WithStore`, `gemini.WithStore` or `grok.WithStore` | Scoped to its provider. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithThinkingBudget` | `WithReasoning(&Reasoning{Budget: …})` | Or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |

### `wizard`

| `mcplib` identifier | SDK equivalent | Notes |
| :--- | :--- | :--- |
| `wizard.Choice` | `Choice` | Unchanged. |
| `wizard.Choice.Detail` | `Choice.Detail` | Unchanged. |
| `wizard.Choice.Label` | `Choice.Label` | Unchanged. |
| `wizard.ConfigureLLM` | `ConfigureLLM` | The same flow. A session goes to `Options.TokenStore` and never into the `Result` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D11, A7). The menu comes from `Options.Registry`, and the environment is read only through `Options.LookupEnv` ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D9). |
| `wizard.CredAPIKey` | `CredAPIKey` | Unchanged, with its value. |
| `wizard.CredNone` | `CredNone` | Unchanged, with its value. |
| `wizard.CredOAuth` | `CredOAuth` | The `Result` holds no token; load the session from `Options.TokenStore`. A Kilo device login is one (A7). |
| `wizard.CredVendorCLI` | `CredVendorCLI` | Unchanged, with its value. Build `auth.VendorCLISession` from `Result.VendorAuthPath`. |
| `wizard.CredentialKind` | `CredentialKind` | Unchanged. |
| `wizard.ErrOrchestrated` | removed | Orchestration stays in `mcplib`; a caller checks its own state before calling `ConfigureLLM` ([0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md), sixth amendment). |
| `wizard.Level` | `Level` | Unchanged. |
| `wizard.Level.String` | `Level.String` | Unchanged. |
| `wizard.LevelError` | `LevelError` | Unchanged. |
| `wizard.LevelInfo` | `LevelInfo` | Unchanged. |
| `wizard.LevelWarn` | `LevelWarn` | Unchanged. |
| `wizard.NewTextPrompter` | `NewTextPrompter` | Unchanged. |
| `wizard.Options` | `Options` | Adds `Registry` and `ProviderOptions` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, S10); `Orchestrated` is removed. A nil `LookupEnv` reads nothing. |
| `wizard.Options.AllowEnv` | `Options.AllowEnv` | Reads through `Options.LookupEnv` only. |
| `wizard.Options.Discover` | `Options.Discover` | Unchanged. |
| `wizard.Options.DiscoverLimit` | `Options.DiscoverLimit` | Unchanged. |
| `wizard.Options.Existing` | `Options.Existing` | A `Result`; its `Provider` is a `ProviderID`. A kept `CredOAuth` session is read from `Options.TokenStore` (A10). |
| `wizard.Options.HTTPClient` | `Options.HTTPClient` | Unchanged. It carries the sign-ins, every listing, and `Logout`'s revocation. |
| `wizard.Options.LookupEnv` | `Options.LookupEnv` | Nil reads nothing; pass `os.Getenv` for the process environment ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "no ambient state, in detail"). |
| `wizard.Options.NeedFallbacks` | `Options.NeedFallbacks` | Unchanged. |
| `wizard.Options.OpenURL` | `Options.OpenURL` | Unchanged. |
| `wizard.Options.Orchestrated` | removed | As `ErrOrchestrated` ([0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md), sixth amendment). |
| `wizard.Options.Profile` | `Options.Profile` | A `catalog.Profile` (`catalog.ProfileUtility`, `catalog.ProfileCapable`); it was an `llmprovider.ModelProfile`. |
| `wizard.Options.Providers` | `Options.Providers` | A `[]llmprovider.ProviderID` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8, D6). |
| `wizard.Options.TokenStore` | `Options.TokenStore` | Holds every session the wizard signs in, and is read when one is kept (A10). |
| `wizard.Prompter` | `Prompter` | Unchanged. |
| `wizard.Prompter.Confirm` | `Prompter.Confirm` | Unchanged. |
| `wizard.Prompter.Input` | `Prompter.Input` | Unchanged. |
| `wizard.Prompter.MultiSelect` | `Prompter.MultiSelect` | Unchanged. |
| `wizard.Prompter.Notify` | `Prompter.Notify` | Unchanged. |
| `wizard.Prompter.Secret` | `Prompter.Secret` | Unchanged. |
| `wizard.Prompter.Select` | `Prompter.Select` | Unchanged. |
| `wizard.Result` | `Result` | `String`, `GoString` and `LogValue` redact `APIKey`; JSON keeps it ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D5, A9). |
| `wizard.Result.APIKey` | `Result.APIKey` | Set for `CredAPIKey` only; a Kilo device login no longer sets it (A7). |
| `wizard.Result.AccessToken` | removed | The session is in `Options.TokenStore`, its only copy ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) D11, A7; [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `wizard.Result.AccountID` | `Result.AccountID` | Unchanged. It describes the `CredOAuth` session whose tokens are in `Options.TokenStore` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.BaseURL` | `Result.BaseURL` | Unchanged. |
| `wizard.Result.ClientID` | `Result.ClientID` | Unchanged. It describes the `CredOAuth` session whose tokens are in `Options.TokenStore` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.Fallbacks` | `Result.Fallbacks` | Unchanged. |
| `wizard.Result.FedRAMP` | `Result.FedRAMP` | Unchanged. It describes the `CredOAuth` session whose tokens are in `Options.TokenStore` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.Issuer` | `Result.Issuer` | Unchanged. It describes the `CredOAuth` session whose tokens are in `Options.TokenStore` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.Kind` | `Result.Kind` | Unchanged. A Kilo device login is `CredOAuth` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.Model` | `Result.Model` | Unchanged. |
| `wizard.Result.Provider` | `Result.Provider` | Typed `ProviderID` in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D6). |
| `wizard.Result.RefreshToken` | removed | As `AccessToken`. |
| `wizard.Result.TokenExpiry` | `Result.TokenExpiry` | Unchanged. It describes the `CredOAuth` session whose tokens are in `Options.TokenStore` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A7). |
| `wizard.Result.VendorAuthPath` | `Result.VendorAuthPath` | Unchanged. Build `auth.VendorCLISession` from it ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8c). |
| `wizard.TextPrompter` | `TextPrompter` | Unchanged. |
| `wizard.TextPrompter.Confirm` | `TextPrompter.Confirm` | Unchanged. |
| `wizard.TextPrompter.In` | `TextPrompter.In` | Unchanged. |
| `wizard.TextPrompter.Input` | `TextPrompter.Input` | Unchanged. |
| `wizard.TextPrompter.MultiSelect` | `TextPrompter.MultiSelect` | Unchanged. |
| `wizard.TextPrompter.Notify` | `TextPrompter.Notify` | Unchanged. |
| `wizard.TextPrompter.Out` | `TextPrompter.Out` | Unchanged. |
| `wizard.TextPrompter.Secret` | `TextPrompter.Secret` | Unchanged. |
| `wizard.TextPrompter.Select` | `TextPrompter.Select` | Unchanged. |
