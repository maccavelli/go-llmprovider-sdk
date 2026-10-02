# Migrating from mcplib

How code that used `mcplib`'s `llmprovider` and `wizard` packages moves to this
module. The table maps every exported identifier of `mcplib` `v1.6.0` to its
equivalent here, or records its removal and why.

The table is being filled in: the "SDK equivalent" column gains a value as each
phase of
[0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) lands.
An empty cell means "not mapped yet", not "unchanged". `make parity-check`
fails if an identifier has no row; from 0015-PLAN S11 it also fails on an empty
cell.

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
| `llmprovider.APIError.Error` |  |  |
| `llmprovider.APIError.Message` |  |  |
| `llmprovider.APIError.Provider` |  |  |
| `llmprovider.APIError.RetryAfter` |  |  |
| `llmprovider.APIError.Status` |  |  |
| `llmprovider.APIError.Terminal` | `APIError.Retryable()` | Unexported in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8; `Retryable()` reads it. |
| `llmprovider.APIError.Type` | `APIError.Code` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8; `Code` carries the same value. |
| `llmprovider.APIError.Unwrap` |  |  |
| `llmprovider.ApplyOptions` | `ResolveOptions(id, opts)` | A provider's `New` resolves its options to read-only `Settings`. Unexported in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 (R19). |
| `llmprovider.AuthAPIKey` |  |  |
| `llmprovider.AuthBrowserOAuth` |  |  |
| `llmprovider.AuthDeviceCode` |  |  |
| `llmprovider.AuthImportVendorCLI` |  |  |
| `llmprovider.AuthMethod` |  |  |
| `llmprovider.AuthMethod.Detail` |  |  |
| `llmprovider.AuthMethod.HeadlessOK` |  |  |
| `llmprovider.AuthMethod.ID` |  |  |
| `llmprovider.AuthMethod.Interactive` |  |  |
| `llmprovider.AuthMethod.Label` |  |  |
| `llmprovider.AuthMethodID` |  |  |
| `llmprovider.AuthTokenStdin` |  |  |
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
| `llmprovider.DefaultGrokBaseURL` |  |  |
| `llmprovider.DefaultGrokOAuthClientID` |  |  |
| `llmprovider.DefaultGrokOAuthIssuer` |  |  |
| `llmprovider.DefaultOpenAIChatGPTBaseURL` |  |  |
| `llmprovider.DefaultOpenAIClientID` |  |  |
| `llmprovider.DefaultOpenAIIssuer` |  |  |
| `llmprovider.DefaultOpenAIPlatformBaseURL` |  |  |
| `llmprovider.DescriptorFor` | `Registry.Descriptor(id)` | On `providers.Default()`, or the caller's `Registry` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.Descriptors` | `Registry.Descriptors()` | In menu order; each provider package declares its own descriptor ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrAuthFailure` | `ErrAuthFailure` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrInvalidProvider` | `ErrInvalidProvider` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrInvalidRequest` | `ErrInvalidRequest` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrNotPermitted` | `ErrNotPermitted` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrProviderUnavailable` | `ErrProviderUnavailable` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrQuotaExhausted` | `ErrQuotaExhausted` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.ErrRateLimited` | `ErrRateLimited` | Its message reads `llmprovider:`, no longer `llm:` (R27, [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.FileTokenStore` |  |  |
| `llmprovider.FileTokenStore.Delete` |  |  |
| `llmprovider.FileTokenStore.Dir` |  |  |
| `llmprovider.FileTokenStore.Load` |  |  |
| `llmprovider.FileTokenStore.Save` |  |  |
| `llmprovider.FunctionCallItem` |  |  |
| `llmprovider.FunctionCallItem.Arguments` |  |  |
| `llmprovider.FunctionCallItem.CallID` |  |  |
| `llmprovider.FunctionCallItem.Name` |  |  |
| `llmprovider.FunctionCallItem.Signature` |  |  |
| `llmprovider.FunctionCallOutputItem` |  |  |
| `llmprovider.FunctionCallOutputItem.CallID` |  |  |
| `llmprovider.FunctionCallOutputItem.Output` |  |  |
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
| `llmprovider.Item` |  |  |
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
| `llmprovider.LoginBrowserOAuth` |  |  |
| `llmprovider.LoginDeviceOAuth` |  |  |
| `llmprovider.MaxListedModels` | `catalog.MaxListed` |  |
| `llmprovider.MessageItem` |  |  |
| `llmprovider.MessageItem.Role` |  |  |
| `llmprovider.MessageItem.Text` |  |  |
| `llmprovider.ModelCatalog` | `catalog.Catalog` |  |
| `llmprovider.ModelCatalog.Err` | `Catalog.Err` |  |
| `llmprovider.ModelCatalog.Live` | `Catalog.Live` |  |
| `llmprovider.ModelCatalog.Recommended` | `Catalog.Recommended` |  |
| `llmprovider.ModelCatalog.Usable` | `Catalog.Usable` |  |
| `llmprovider.ModelDiscoverer` | `ModelLister` | Or `catalog.List`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ModelDiscoverer.DiscoverModels` | `ModelLister.ListModels` | As `ModelDiscoverer`. |
| `llmprovider.ModelLabel` | `catalog.Label` |  |
| `llmprovider.ModelMatch` | `catalog.Match` |  |
| `llmprovider.ModelMatch.ID` | `Match.ID` |  |
| `llmprovider.ModelMatch.Label` | `Match.Label` |  |
| `llmprovider.ModelMatch.Score` | `Match.Score` |  |
| `llmprovider.ModelProfile` | `catalog.Profile` | Passed to a listing or a provider with `catalog.WithProfile`. |
| `llmprovider.ModelProfile.ReasoningEffort` | `Profile.ReasoningEffort` |  |
| `llmprovider.NewClaude` | `claude.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderClaude, …)`; an OAuth session is refused with `ErrUnsupported` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewFileTokenStore` |  |  |
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
| `llmprovider.NewStaticToken` |  |  |
| `llmprovider.OAuthFlowOptions` |  |  |
| `llmprovider.OAuthFlowOptions.ClientID` |  |  |
| `llmprovider.OAuthFlowOptions.HTTPClient` |  |  |
| `llmprovider.OAuthFlowOptions.InputCode` |  |  |
| `llmprovider.OAuthFlowOptions.Issuer` |  |  |
| `llmprovider.OAuthFlowOptions.NotifyDevice` |  |  |
| `llmprovider.OAuthFlowOptions.OpenURL` |  |  |
| `llmprovider.OAuthSession` |  |  |
| `llmprovider.OAuthSession.Access` |  |  |
| `llmprovider.OAuthSession.AccountID` |  |  |
| `llmprovider.OAuthSession.ChatGPT` |  |  |
| `llmprovider.OAuthSession.ClientID` |  |  |
| `llmprovider.OAuthSession.Expiry` |  |  |
| `llmprovider.OAuthSession.FedRAMP` |  |  |
| `llmprovider.OAuthSession.HTTPClient` |  |  |
| `llmprovider.OAuthSession.Issuer` |  |  |
| `llmprovider.OAuthSession.Provider` |  |  |
| `llmprovider.OAuthSession.Refresh` |  |  |
| `llmprovider.OAuthSession.Store` |  |  |
| `llmprovider.OAuthSession.Token` |  |  |
| `llmprovider.OAuthSession.TokenURL` |  |  |
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
| `llmprovider.ProviderClaude` |  |  |
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
| `llmprovider.ProviderEnvVars` | `ProviderEnvVars()` | A function returning a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.ProviderGemini` |  |  |
| `llmprovider.ProviderGrok` |  |  |
| `llmprovider.ProviderHuggingFace` |  |  |
| `llmprovider.ProviderKilo` |  |  |
| `llmprovider.ProviderOllama` |  |  |
| `llmprovider.ProviderOpenAI` |  |  |
| `llmprovider.ProviderOpencodeGo` |  |  |
| `llmprovider.ProviderOpencodeZen` |  |  |
| `llmprovider.ProviderOption` | `Option` | The alias is removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.RankClaudeModel` | `RankModel(ProviderClaude, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGeminiModel` | `RankModel(ProviderGemini, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGrokModel` | `RankModel(ProviderGrok, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankHuggingFaceModel` | `RankModel(ProviderHuggingFace, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankKiloModel` | `RankModel(ProviderKilo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpenAIModel` | `RankModel(ProviderOpenAI, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpencodeModel` | `RankModel(ProviderOpencodeZen` or `ProviderOpencodeGo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RateLimitError` | `*APIError` of kind `ErrRateLimited` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8 ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7). A 429 and a stream rate limit are both an `*APIError`. |
| `llmprovider.RateLimitError.Error` | `APIError.Error` | Reads `llmprovider: rate limited: <provider> HTTP 429 (retry-after <d>): <message>`. |
| `llmprovider.RateLimitError.Message` | `APIError.Message` | As `RateLimitError`. |
| `llmprovider.RateLimitError.Provider` | `APIError.Provider` | As `RateLimitError`. |
| `llmprovider.RateLimitError.RetryAfter` | `APIError.RetryAfter` | As `RateLimitError`. |
| `llmprovider.RateLimitError.Status` | `APIError.Status` | 0 for a stream rate limit. |
| `llmprovider.RateLimitError.Unwrap` | `APIError.Unwrap` | Matches `ErrRateLimited`, as before. |
| `llmprovider.ReasoningItem` |  |  |
| `llmprovider.ReasoningItem.Text` |  |  |
| `llmprovider.Response` | `Response` | Adds `Model` and `Usage`; `FinishReason` is the named type `FinishReason`. |
| `llmprovider.Response.FinishReason` |  |  |
| `llmprovider.Response.ID` |  |  |
| `llmprovider.Response.Items` |  |  |
| `llmprovider.Response.Output` |  |  |
| `llmprovider.Response.OutputText` |  |  |
| `llmprovider.RevokeOAuthSession` |  |  |
| `llmprovider.SearchModels` | `catalog.Search` |  |
| `llmprovider.StaticClaude` | `StaticModels(ProviderClaude)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGemini` | `StaticModels(ProviderGemini)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGrok` | `StaticModels(ProviderGrok)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticHuggingFace` | `StaticModels(ProviderHuggingFace)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticKilo` | `StaticModels(ProviderKilo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticModels` | `catalog.Static` |  |
| `llmprovider.StaticOpenAI` | `StaticModels(ProviderOpenAI)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeGo` | `StaticModels(ProviderOpencodeGo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeZen` | `StaticModels(ProviderOpencodeZen)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticToken` |  |  |
| `llmprovider.StaticToken.Header` | `llmprovider.StaticToken.Header` | Empty now means the service's own header; it no longer defaults to `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Token` | `llmprovider.StaticToken.Token` | Returns a `TokenAPIKey`, with the `Header` set, if any; it returned `TokenBearer` and `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Value` |  |  |
| `llmprovider.ThinkingProvider` | `Provider` | Reasoning is `Request.Reasoning`, or `WithReasoning` at construction. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ThinkingProvider.GenerateThinking` | `GenerateText` with `Request.Reasoning` | As `ThinkingProvider`. |
| `llmprovider.ThinkingToolProvider` | `Provider` | As `ThinkingProvider` and `ToolProvider`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ThinkingToolProvider.GenerateWithToolThinking` | `GenerateToolCall` with `Request.Reasoning` | As `ThinkingToolProvider`. |
| `llmprovider.Token` |  |  |
| `llmprovider.Token.Expiry` |  |  |
| `llmprovider.Token.Header` | `llmprovider.Token.Header` | A non-empty `Header` overrides the service's header (R16, [0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.Token.Type` | `llmprovider.Token.Type` | With an overriding `Header`, `TokenBearer` is sent `Bearer`-prefixed and anything else bare ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.Token.Value` |  |  |
| `llmprovider.TokenAPIKey` |  |  |
| `llmprovider.TokenBearer` |  |  |
| `llmprovider.TokenSource` |  |  |
| `llmprovider.TokenSource.Token` |  |  |
| `llmprovider.TokenStore` |  |  |
| `llmprovider.TokenStore.Delete` |  |  |
| `llmprovider.TokenStore.Load` |  |  |
| `llmprovider.TokenStore.Save` |  |  |
| `llmprovider.TokenType` |  |  |
| `llmprovider.Tool` |  |  |
| `llmprovider.Tool.Description` |  |  |
| `llmprovider.Tool.Name` |  |  |
| `llmprovider.Tool.Schema` |  |  |
| `llmprovider.ToolProvider` | `Provider` | Tools are `Request.Tools` and `ToolChoice`, or `GenerateToolCall`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.ToolProvider.GenerateWithTool` | `GenerateToolCall(ctx, p, req)` | As `ToolProvider`. |
| `llmprovider.ValidateOAuthSession` |  |  |
| `llmprovider.ValidateOllamaURL` | `catalog.ValidateOllamaURL` |  |
| `llmprovider.VendorCLISession` |  |  |
| `llmprovider.VendorCLISession.Path` |  |  |
| `llmprovider.VendorCLISession.Provider` |  |  |
| `llmprovider.VendorCLISession.Token` |  |  |
| `llmprovider.WithBaseURL` |  |  |
| `llmprovider.WithClientInfo` |  |  |
| `llmprovider.WithHTTPClient` |  |  |
| `llmprovider.WithKiloCapabilities` | `kilo.WithCapabilities` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithKiloDataCollection` | `kilo.WithDataCollection` | Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithKiloOrganization` | `kilo.WithOrganization`, and `catalog.WithKiloOrganization` for a listing | `kilo`'s `ListModels` passes its organization on ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8). |
| `llmprovider.WithMaxTokens` |  |  |
| `llmprovider.WithModelMetadataURL` | `llmprovider.WithModelMetadataURL` | A common option now: the new API takes it too ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithModelProfile` | `catalog.WithProfile` | Every built-in provider's `New` takes it; the open catalogs rank with it. |
| `llmprovider.WithOpencodeRoute` | `opencode.WithRoute` | Scoped to both gateways ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithReasoningEffort` | `WithReasoning(&Reasoning{Effort: …})` | Or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithSessionID` |  |  |
| `llmprovider.WithStore` | `openai.WithStore`, `gemini.WithStore` or `grok.WithStore` | Scoped to its provider. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.WithThinkingBudget` | `WithReasoning(&Reasoning{Budget: …})` | Or `Request.Reasoning`. Removed in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |

### `wizard`

| `mcplib` identifier | SDK equivalent | Notes |
| :--- | :--- | :--- |
| `wizard.Choice` |  |  |
| `wizard.Choice.Detail` |  |  |
| `wizard.Choice.Label` |  |  |
| `wizard.ConfigureLLM` |  |  |
| `wizard.CredAPIKey` |  |  |
| `wizard.CredNone` |  |  |
| `wizard.CredOAuth` |  |  |
| `wizard.CredVendorCLI` |  |  |
| `wizard.CredentialKind` |  |  |
| `wizard.ErrOrchestrated` | removed | Orchestration stays in `mcplib`; a caller checks its own state before calling `ConfigureLLM` ([0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md), sixth amendment). |
| `wizard.Level` |  |  |
| `wizard.Level.String` |  |  |
| `wizard.LevelError` |  |  |
| `wizard.LevelInfo` |  |  |
| `wizard.LevelWarn` |  |  |
| `wizard.NewTextPrompter` |  |  |
| `wizard.Options` |  |  |
| `wizard.Options.AllowEnv` |  |  |
| `wizard.Options.Discover` |  |  |
| `wizard.Options.DiscoverLimit` |  |  |
| `wizard.Options.Existing` |  |  |
| `wizard.Options.HTTPClient` |  |  |
| `wizard.Options.LookupEnv` |  |  |
| `wizard.Options.NeedFallbacks` |  |  |
| `wizard.Options.OpenURL` |  |  |
| `wizard.Options.Orchestrated` | removed | As `ErrOrchestrated` ([0002-MADR](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md), sixth amendment). |
| `wizard.Options.Profile` |  |  |
| `wizard.Options.Providers` |  |  |
| `wizard.Options.TokenStore` |  |  |
| `wizard.Prompter` |  |  |
| `wizard.Prompter.Confirm` |  |  |
| `wizard.Prompter.Input` |  |  |
| `wizard.Prompter.MultiSelect` |  |  |
| `wizard.Prompter.Notify` |  |  |
| `wizard.Prompter.Secret` |  |  |
| `wizard.Prompter.Select` |  |  |
| `wizard.Result` |  |  |
| `wizard.Result.APIKey` |  |  |
| `wizard.Result.AccessToken` |  |  |
| `wizard.Result.AccountID` |  |  |
| `wizard.Result.BaseURL` |  |  |
| `wizard.Result.ClientID` |  |  |
| `wizard.Result.Fallbacks` |  |  |
| `wizard.Result.FedRAMP` |  |  |
| `wizard.Result.Issuer` |  |  |
| `wizard.Result.Kind` |  |  |
| `wizard.Result.Model` |  |  |
| `wizard.Result.Provider` |  |  |
| `wizard.Result.RefreshToken` |  |  |
| `wizard.Result.TokenExpiry` |  |  |
| `wizard.Result.VendorAuthPath` |  |  |
| `wizard.TextPrompter` |  |  |
| `wizard.TextPrompter.Confirm` |  |  |
| `wizard.TextPrompter.In` |  |  |
| `wizard.TextPrompter.Input` |  |  |
| `wizard.TextPrompter.MultiSelect` |  |  |
| `wizard.TextPrompter.Notify` |  |  |
| `wizard.TextPrompter.Out` |  |  |
| `wizard.TextPrompter.Secret` |  |  |
| `wizard.TextPrompter.Select` |  |  |
