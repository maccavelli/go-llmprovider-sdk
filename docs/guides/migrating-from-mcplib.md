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
| `llmprovider.APIError` | `APIError` | Adds `Kind`, `Code`, `Reason` and `Retryable()`. `Type` (now `Code`) and `Terminal` go in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.APIError.Error` |  |  |
| `llmprovider.APIError.Message` |  |  |
| `llmprovider.APIError.Provider` |  |  |
| `llmprovider.APIError.RetryAfter` |  |  |
| `llmprovider.APIError.Status` |  |  |
| `llmprovider.APIError.Terminal` |  |  |
| `llmprovider.APIError.Type` |  |  |
| `llmprovider.APIError.Unwrap` |  |  |
| `llmprovider.ApplyOptions` |  |  |
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
| `llmprovider.Continuer` |  |  |
| `llmprovider.Continuer.Continue` |  |  |
| `llmprovider.DefaultGrokBaseURL` |  |  |
| `llmprovider.DefaultGrokOAuthClientID` |  |  |
| `llmprovider.DefaultGrokOAuthIssuer` |  |  |
| `llmprovider.DefaultOpenAIChatGPTBaseURL` |  |  |
| `llmprovider.DefaultOpenAIClientID` |  |  |
| `llmprovider.DefaultOpenAIIssuer` |  |  |
| `llmprovider.DefaultOpenAIPlatformBaseURL` |  |  |
| `llmprovider.DescriptorFor` |  |  |
| `llmprovider.Descriptors` |  |  |
| `llmprovider.ErrAuthFailure` |  |  |
| `llmprovider.ErrInvalidProvider` |  |  |
| `llmprovider.ErrInvalidRequest` |  |  |
| `llmprovider.ErrNotPermitted` |  |  |
| `llmprovider.ErrProviderUnavailable` |  |  |
| `llmprovider.ErrQuotaExhausted` |  |  |
| `llmprovider.ErrRateLimited` |  |  |
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
| `llmprovider.GenerateItemsWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateThinkingWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
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
| `llmprovider.IncompleteError` |  |  |
| `llmprovider.IncompleteError.Error` |  |  |
| `llmprovider.IncompleteError.Reason` |  |  |
| `llmprovider.IncompleteError.Unwrap` |  |  |
| `llmprovider.Item` |  |  |
| `llmprovider.ItemProvider` |  |  |
| `llmprovider.ItemProvider.GenerateItems` |  |  |
| `llmprovider.ItemThinkingProvider` |  |  |
| `llmprovider.ItemThinkingProvider.GenerateItemsThinking` |  |  |
| `llmprovider.ItemThinkingToolProvider` |  |  |
| `llmprovider.ItemThinkingToolProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.ItemToolProvider` |  |  |
| `llmprovider.ItemToolProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.KiloModelCapabilities` |  |  |
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
| `llmprovider.ListAvailableModels` |  |  |
| `llmprovider.ListAvailableModelsWithSource` |  |  |
| `llmprovider.ListModelCatalog` |  |  |
| `llmprovider.ListModelCatalogWithSource` |  |  |
| `llmprovider.LoginBrowserOAuth` |  |  |
| `llmprovider.LoginDeviceOAuth` |  |  |
| `llmprovider.MaxListedModels` |  |  |
| `llmprovider.MessageItem` |  |  |
| `llmprovider.MessageItem.Role` |  |  |
| `llmprovider.MessageItem.Text` |  |  |
| `llmprovider.ModelCatalog` |  |  |
| `llmprovider.ModelCatalog.Err` |  |  |
| `llmprovider.ModelCatalog.Live` |  |  |
| `llmprovider.ModelCatalog.Recommended` |  |  |
| `llmprovider.ModelCatalog.Usable` |  |  |
| `llmprovider.ModelDiscoverer` |  |  |
| `llmprovider.ModelDiscoverer.DiscoverModels` |  |  |
| `llmprovider.ModelLabel` |  |  |
| `llmprovider.ModelMatch` |  |  |
| `llmprovider.ModelMatch.ID` |  |  |
| `llmprovider.ModelMatch.Label` |  |  |
| `llmprovider.ModelMatch.Score` |  |  |
| `llmprovider.ModelProfile` |  |  |
| `llmprovider.ModelProfile.ReasoningEffort` |  |  |
| `llmprovider.NewClaude` | `claude.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderClaude, …)`; an OAuth session is refused with `ErrUnsupported` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewFileTokenStore` |  |  |
| `llmprovider.NewGemini` | `gemini.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderGemini, …)`. It takes no context; an OAuth session is refused with `ErrUnsupported` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewGrok` | `grok.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderGrok, …)`. A session, which only `NewProvider` took, is `WithTokenSource(src)` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewHuggingFace` | `huggingface.New(WithAPIKey(token), WithModel(model), …)` | Or `providers.New(ProviderHuggingFace, …)`; an empty token is refused with `ErrInvalidRequest` ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewKilo` | `kilo.New(WithAPIKey(key), WithModel(model), …)` | Or `providers.New(ProviderKilo, …)`. No key sends the anonymous token, as before ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.NewOllama` |  |  |
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
| `llmprovider.OllamaProvider` |  |  |
| `llmprovider.OllamaProvider.DiscoverModels` |  |  |
| `llmprovider.OllamaProvider.Generate` |  |  |
| `llmprovider.OllamaProvider.GenerateItems` |  |  |
| `llmprovider.OllamaProvider.GenerateItemsThinking` |  |  |
| `llmprovider.OllamaProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.OllamaProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.OllamaProvider.GenerateThinking` |  |  |
| `llmprovider.OllamaProvider.GenerateWithTool` |  |  |
| `llmprovider.OllamaProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.OllamaProvider.Name` |  |  |
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
| `llmprovider.ProfileCapable` |  |  |
| `llmprovider.ProfileUtility` |  |  |
| `llmprovider.Provider` | `Provider` | `ID`, `Capabilities` and `Generate(ctx, *Request)`. The old interface is `LegacyProvider` until [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.Provider.Generate` |  |  |
| `llmprovider.Provider.Name` |  |  |
| `llmprovider.ProviderClaude` |  |  |
| `llmprovider.ProviderConfig` |  |  |
| `llmprovider.ProviderConfig.BaseURL` |  |  |
| `llmprovider.ProviderConfig.ClientName` |  |  |
| `llmprovider.ProviderConfig.ClientVersion` |  |  |
| `llmprovider.ProviderConfig.HTTPClient` |  |  |
| `llmprovider.ProviderConfig.KiloCapabilities` |  |  |
| `llmprovider.ProviderConfig.KiloDataCollection` |  |  |
| `llmprovider.ProviderConfig.KiloOrganization` |  |  |
| `llmprovider.ProviderConfig.MaxTokens` |  |  |
| `llmprovider.ProviderConfig.ModelMetadataURL` |  |  |
| `llmprovider.ProviderConfig.ModelProfile` |  |  |
| `llmprovider.ProviderConfig.OpencodeRoute` | `opencode.WithRoute` | Removed with the old provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.ProviderConfig.ReasoningEffort` |  |  |
| `llmprovider.ProviderConfig.SessionID` |  |  |
| `llmprovider.ProviderConfig.Store` |  |  |
| `llmprovider.ProviderConfig.ThinkingBudget` |  |  |
| `llmprovider.ProviderDescriptor` |  |  |
| `llmprovider.ProviderDescriptor.AuthMethods` |  |  |
| `llmprovider.ProviderDescriptor.DefaultBaseURL` |  |  |
| `llmprovider.ProviderDescriptor.EnvVar` |  |  |
| `llmprovider.ProviderDescriptor.ID` |  |  |
| `llmprovider.ProviderDescriptor.IsLocal` |  |  |
| `llmprovider.ProviderDescriptor.Label` |  |  |
| `llmprovider.ProviderDescriptor.Notes` |  |  |
| `llmprovider.ProviderDescriptor.RequiresAPIKey` |  |  |
| `llmprovider.ProviderDescriptor.StaticModels` |  |  |
| `llmprovider.ProviderDescriptor.SupportsBaseURL` |  |  |
| `llmprovider.ProviderEnvVars` | `ProviderEnvVars()` | A function returning a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.ProviderGemini` |  |  |
| `llmprovider.ProviderGrok` |  |  |
| `llmprovider.ProviderHuggingFace` |  |  |
| `llmprovider.ProviderKilo` |  |  |
| `llmprovider.ProviderOllama` |  |  |
| `llmprovider.ProviderOpenAI` |  |  |
| `llmprovider.ProviderOpencodeGo` |  |  |
| `llmprovider.ProviderOpencodeZen` |  |  |
| `llmprovider.ProviderOption` | `Option` | `ProviderOption` is an alias of it until [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.RankClaudeModel` | `RankModel(ProviderClaude, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGeminiModel` | `RankModel(ProviderGemini, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankGrokModel` | `RankModel(ProviderGrok, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankHuggingFaceModel` | `RankModel(ProviderHuggingFace, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankKiloModel` | `RankModel(ProviderKilo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpenAIModel` | `RankModel(ProviderOpenAI, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RankOpencodeModel` | `RankModel(ProviderOpencodeZen` or `ProviderOpencodeGo, model)` | One function for every provider ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5); `catalog.Rank` from S7b. |
| `llmprovider.RateLimitError` |  |  |
| `llmprovider.RateLimitError.Error` |  |  |
| `llmprovider.RateLimitError.Message` |  |  |
| `llmprovider.RateLimitError.Provider` |  |  |
| `llmprovider.RateLimitError.RetryAfter` |  |  |
| `llmprovider.RateLimitError.Status` |  |  |
| `llmprovider.RateLimitError.Unwrap` |  |  |
| `llmprovider.ReasoningItem` |  |  |
| `llmprovider.ReasoningItem.Text` |  |  |
| `llmprovider.Response` | `Response` | Adds `Model` and `Usage`; `FinishReason` is the named type `FinishReason`. |
| `llmprovider.Response.FinishReason` |  |  |
| `llmprovider.Response.ID` |  |  |
| `llmprovider.Response.Items` |  |  |
| `llmprovider.Response.Output` |  |  |
| `llmprovider.Response.OutputText` |  |  |
| `llmprovider.RevokeOAuthSession` |  |  |
| `llmprovider.SearchModels` |  |  |
| `llmprovider.StaticClaude` | `StaticModels(ProviderClaude)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGemini` | `StaticModels(ProviderGemini)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticGrok` | `StaticModels(ProviderGrok)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticHuggingFace` | `StaticModels(ProviderHuggingFace)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticKilo` | `StaticModels(ProviderKilo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticModels` |  |  |
| `llmprovider.StaticOpenAI` | `StaticModels(ProviderOpenAI)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeGo` | `StaticModels(ProviderOpencodeGo)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticOpencodeZen` | `StaticModels(ProviderOpencodeZen)` | The variable is unexported; the function returns a copy ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S5, D9). |
| `llmprovider.StaticToken` |  |  |
| `llmprovider.StaticToken.Header` | `llmprovider.StaticToken.Header` | Empty now means the service's own header; it no longer defaults to `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Token` | `llmprovider.StaticToken.Token` | Returns a `TokenAPIKey`, with the `Header` set, if any; it returned `TokenBearer` and `Authorization` ([0016-MADR](../decisions/0016-MADR-provider-auth-and-support-baseline.md) A6). |
| `llmprovider.StaticToken.Value` |  |  |
| `llmprovider.ThinkingProvider` |  |  |
| `llmprovider.ThinkingProvider.GenerateThinking` |  |  |
| `llmprovider.ThinkingToolProvider` |  |  |
| `llmprovider.ThinkingToolProvider.GenerateWithToolThinking` |  |  |
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
| `llmprovider.ToolProvider` |  |  |
| `llmprovider.ToolProvider.GenerateWithTool` |  |  |
| `llmprovider.ValidateOAuthSession` |  |  |
| `llmprovider.ValidateOllamaURL` |  |  |
| `llmprovider.VendorCLISession` |  |  |
| `llmprovider.VendorCLISession.Path` |  |  |
| `llmprovider.VendorCLISession.Provider` |  |  |
| `llmprovider.VendorCLISession.Token` |  |  |
| `llmprovider.WithBaseURL` |  |  |
| `llmprovider.WithClientInfo` |  |  |
| `llmprovider.WithHTTPClient` |  |  |
| `llmprovider.WithKiloCapabilities` | `kilo.WithCapabilities` | The old option is removed in S8 ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.WithKiloDataCollection` | `kilo.WithDataCollection` | The old option is removed in S8 ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.WithKiloOrganization` | `kilo.WithOrganization` | The old option still scopes the old catalog functions' Kilo listing until S8b ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.WithMaxTokens` |  |  |
| `llmprovider.WithModelMetadataURL` | `llmprovider.WithModelMetadataURL` | A common option now: the new API takes it too ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithModelProfile` |  |  |
| `llmprovider.WithOpencodeRoute` | `opencode.WithRoute` | Scoped to both gateways ([0015-MADR](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md), amendment "the OpenCode family"). |
| `llmprovider.WithReasoningEffort` |  |  |
| `llmprovider.WithSessionID` |  |  |
| `llmprovider.WithStore` | `openai.WithStore`, `gemini.WithStore` or `grok.WithStore` | Scoped to its provider; the old option is removed in S8 ([0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S7). |
| `llmprovider.WithThinkingBudget` |  |  |

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
