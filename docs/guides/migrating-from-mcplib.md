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
| `llmprovider.ClaudeProvider` |  |  |
| `llmprovider.ClaudeProvider.DiscoverModels` |  |  |
| `llmprovider.ClaudeProvider.Generate` |  |  |
| `llmprovider.ClaudeProvider.GenerateItems` |  |  |
| `llmprovider.ClaudeProvider.GenerateItemsThinking` |  |  |
| `llmprovider.ClaudeProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.ClaudeProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.ClaudeProvider.GenerateThinking` |  |  |
| `llmprovider.ClaudeProvider.GenerateWithTool` |  |  |
| `llmprovider.ClaudeProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.ClaudeProvider.Name` |  |  |
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
| `llmprovider.GeminiProvider` |  |  |
| `llmprovider.GeminiProvider.Continue` |  |  |
| `llmprovider.GeminiProvider.DiscoverModels` |  |  |
| `llmprovider.GeminiProvider.Generate` |  |  |
| `llmprovider.GeminiProvider.GenerateItems` |  |  |
| `llmprovider.GeminiProvider.GenerateItemsThinking` |  |  |
| `llmprovider.GeminiProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.GeminiProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.GeminiProvider.GenerateThinking` |  |  |
| `llmprovider.GeminiProvider.GenerateWithTool` |  |  |
| `llmprovider.GeminiProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.GeminiProvider.Name` |  |  |
| `llmprovider.GenerateItemsWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateThinkingWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GenerateWithRetry` | `WithRetry(p, RetryPolicy{…})` | Middleware over any `Provider`; the function goes in [0015-PLAN](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md) S8. |
| `llmprovider.GrokProvider` |  |  |
| `llmprovider.GrokProvider.Continue` |  |  |
| `llmprovider.GrokProvider.DiscoverModels` |  |  |
| `llmprovider.GrokProvider.Generate` |  |  |
| `llmprovider.GrokProvider.GenerateItems` |  |  |
| `llmprovider.GrokProvider.GenerateItemsThinking` |  |  |
| `llmprovider.GrokProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.GrokProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.GrokProvider.GenerateThinking` |  |  |
| `llmprovider.GrokProvider.GenerateWithTool` |  |  |
| `llmprovider.GrokProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.GrokProvider.Name` |  |  |
| `llmprovider.HuggingFaceProvider` |  |  |
| `llmprovider.HuggingFaceProvider.DiscoverModels` |  |  |
| `llmprovider.HuggingFaceProvider.Generate` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateItems` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateItemsThinking` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateThinking` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateWithTool` |  |  |
| `llmprovider.HuggingFaceProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.HuggingFaceProvider.Name` |  |  |
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
| `llmprovider.KiloProvider` |  |  |
| `llmprovider.KiloProvider.DiscoverModels` |  |  |
| `llmprovider.KiloProvider.Generate` |  |  |
| `llmprovider.KiloProvider.GenerateItems` |  |  |
| `llmprovider.KiloProvider.GenerateItemsThinking` |  |  |
| `llmprovider.KiloProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.KiloProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.KiloProvider.GenerateThinking` |  |  |
| `llmprovider.KiloProvider.GenerateWithTool` |  |  |
| `llmprovider.KiloProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.KiloProvider.Name` |  |  |
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
| `llmprovider.NewClaude` |  |  |
| `llmprovider.NewFileTokenStore` |  |  |
| `llmprovider.NewGemini` |  |  |
| `llmprovider.NewGrok` |  |  |
| `llmprovider.NewHuggingFace` |  |  |
| `llmprovider.NewKilo` |  |  |
| `llmprovider.NewOllama` |  |  |
| `llmprovider.NewOpenAI` |  |  |
| `llmprovider.NewOpenAIWithSource` |  |  |
| `llmprovider.NewOpencode` |  |  |
| `llmprovider.NewProvider` |  |  |
| `llmprovider.NewProviderWithSource` |  |  |
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
| `llmprovider.OpenAIProvider` |  |  |
| `llmprovider.OpenAIProvider.Continue` |  |  |
| `llmprovider.OpenAIProvider.DiscoverModels` |  |  |
| `llmprovider.OpenAIProvider.Generate` |  |  |
| `llmprovider.OpenAIProvider.GenerateItems` |  |  |
| `llmprovider.OpenAIProvider.GenerateItemsThinking` |  |  |
| `llmprovider.OpenAIProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.OpenAIProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.OpenAIProvider.GenerateThinking` |  |  |
| `llmprovider.OpenAIProvider.GenerateWithTool` |  |  |
| `llmprovider.OpenAIProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.OpenAIProvider.Name` |  |  |
| `llmprovider.OpencodeProvider` |  |  |
| `llmprovider.OpencodeProvider.DiscoverModels` |  |  |
| `llmprovider.OpencodeProvider.Generate` |  |  |
| `llmprovider.OpencodeProvider.GenerateItems` |  |  |
| `llmprovider.OpencodeProvider.GenerateItemsThinking` |  |  |
| `llmprovider.OpencodeProvider.GenerateItemsWithTool` |  |  |
| `llmprovider.OpencodeProvider.GenerateItemsWithToolThinking` |  |  |
| `llmprovider.OpencodeProvider.GenerateThinking` |  |  |
| `llmprovider.OpencodeProvider.GenerateWithTool` |  |  |
| `llmprovider.OpencodeProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.OpencodeProvider.Name` |  |  |
| `llmprovider.OpencodeProvider.Route` |  |  |
| `llmprovider.OpencodeRoute` |  |  |
| `llmprovider.OpencodeRouteChatCompletions` |  |  |
| `llmprovider.OpencodeRouteGoogle` |  |  |
| `llmprovider.OpencodeRouteMessages` |  |  |
| `llmprovider.OpencodeRouteResponses` |  |  |
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
| `llmprovider.ProviderConfig.OpencodeRoute` |  |  |
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
| `llmprovider.StaticToken.Header` |  |  |
| `llmprovider.StaticToken.Token` |  |  |
| `llmprovider.StaticToken.Value` |  |  |
| `llmprovider.ThinkingProvider` |  |  |
| `llmprovider.ThinkingProvider.GenerateThinking` |  |  |
| `llmprovider.ThinkingToolProvider` |  |  |
| `llmprovider.ThinkingToolProvider.GenerateWithToolThinking` |  |  |
| `llmprovider.Token` |  |  |
| `llmprovider.Token.Expiry` |  |  |
| `llmprovider.Token.Header` |  |  |
| `llmprovider.Token.Type` |  |  |
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
| `llmprovider.WithKiloCapabilities` |  |  |
| `llmprovider.WithKiloDataCollection` |  |  |
| `llmprovider.WithKiloOrganization` |  |  |
| `llmprovider.WithMaxTokens` |  |  |
| `llmprovider.WithModelMetadataURL` |  |  |
| `llmprovider.WithModelProfile` |  |  |
| `llmprovider.WithOpencodeRoute` |  |  |
| `llmprovider.WithReasoningEffort` |  |  |
| `llmprovider.WithSessionID` |  |  |
| `llmprovider.WithStore` |  |  |
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
