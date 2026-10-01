package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const togetherBaseURL = "https://api.together.ai/v1"

// TogetherProvider implements Provider against Together AI's OpenAI Chat
// Completions API (docs/decisions/0017-MADR-together-provider-and-auth-extensions.md
// D1). The wire shapes are from Together's API reference, read 2026-09-30;
// live_together_test.go confirms them against the service.
//
// The thinking path sends reasoning {"enabled": true}, which Together's
// toggleable models read, and reasoning_effort only when an effort was set
// with WithReasoningEffort: Together accepts it for some models only
// (openai/gpt-oss-*, DeepSeek-V4-Pro). The plain path sends neither, so the
// model's own default applies. finish_reason "eos" is a normal stop.
//
// TogetherProvider does NOT implement Continuer: Chat Completions is
// stateless.
type TogetherProvider struct {
	apiKey          string
	model           string
	baseURL         string
	client          *http.Client
	maxTokens       int
	reasoningEffort string
	modelProfile    ModelProfile
	metadataURL     string
	// identity names the client on every request (MADR 0012 §1.4).
	identity clientIdentity
}

// NewTogether creates a Together AI client.
func NewTogether(apiKey, model string, opts ...ProviderOption) (*TogetherProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("together api key is required")
	}
	cfg := ApplyOptions(opts)
	baseURL := togetherBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	return &TogetherProvider{
		apiKey:          apiKey,
		model:           model,
		baseURL:         baseURL,
		client:          cfg.HTTPClient,
		identity:        identityOf(cfg),
		maxTokens:       cfg.MaxTokens,
		reasoningEffort: cfg.ReasoningEffort,
		modelProfile:    cfg.ModelProfile,
		metadataURL:     cfg.ModelMetadataURL,
	}, nil
}

// Name returns the provider's canonical identifier "together".
func (p *TogetherProvider) Name() string { return ProviderTogether }

// Generate sends a prompt and returns the generated text.
func (p *TogetherProvider) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := p.GenerateItems(ctx, MessageItem{Role: jsonRoleUser, Text: prompt})
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

// GenerateThinking runs Generate with reasoning enabled.
func (p *TogetherProvider) GenerateThinking(ctx context.Context, prompt string) (string, error) {
	resp, err := p.GenerateItemsThinking(ctx, MessageItem{Role: jsonRoleUser, Text: prompt})
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

// GenerateWithTool forces a tool call and returns its JSON arguments.
func (p *TogetherProvider) GenerateWithTool(ctx context.Context, prompt string, tool Tool) (string, error) {
	resp, err := p.GenerateItemsWithTool(ctx, tool, MessageItem{Role: jsonRoleUser, Text: prompt})
	if err != nil {
		return "", err
	}
	return firstFunctionCallArgs(resp, ProviderTogether)
}

// GenerateWithToolThinking runs GenerateWithTool with reasoning enabled.
func (p *TogetherProvider) GenerateWithToolThinking(ctx context.Context, prompt string, tool Tool) (string, error) {
	resp, err := p.GenerateItemsWithToolThinking(ctx, tool, MessageItem{Role: jsonRoleUser, Text: prompt})
	if err != nil {
		return "", err
	}
	return firstFunctionCallArgs(resp, ProviderTogether)
}

// GenerateItems sends items and returns typed output items.
func (p *TogetherProvider) GenerateItems(ctx context.Context, input ...Item) (*Response, error) {
	return p.doGenerateItems(ctx, input, nil, false)
}

// GenerateItemsWithTool sends items with a forced tool call.
func (p *TogetherProvider) GenerateItemsWithTool(ctx context.Context, tool Tool, input ...Item) (*Response, error) {
	return p.doGenerateItems(ctx, input, &tool, false)
}

// GenerateItemsThinking sends items with reasoning enabled.
func (p *TogetherProvider) GenerateItemsThinking(ctx context.Context, input ...Item) (*Response, error) {
	return p.doGenerateItems(ctx, input, nil, true)
}

// GenerateItemsWithToolThinking sends items with both tool calling and reasoning.
func (p *TogetherProvider) GenerateItemsWithToolThinking(ctx context.Context, tool Tool, input ...Item) (*Response, error) {
	return p.doGenerateItems(ctx, input, &tool, true)
}

func (p *TogetherProvider) doGenerateItems(ctx context.Context, input []Item, tool *Tool, thinking bool) (*Response, error) {
	opts := ChatCompletionsOpts{Tool: tool, ForceTool: tool != nil}
	if thinking {
		opts.Reasoning = map[string]any{jsonKeyEnabled: true}
		opts.ReasoningEffort = p.reasoningEffort
	}
	reqBody, err := json.Marshal(ChatCompletionsBody(p.model, p.maxTokens, input, opts))
	if err != nil {
		return nil, fmt.Errorf("together: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	p.identity.setUserAgent(req)
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	// Bound the body before the status check, so error bodies are bounded too.
	limitedBody := io.LimitReader(resp.Body, 1<<20)
	if err := ClassifyHTTPError(ProviderTogether, resp); err != nil {
		return nil, err
	}
	return DecodeChatCompletionsResponse(limitedBody)
}

// DiscoverModels returns the curated chat models, falling back to the static
// catalog. It spends no generation on probes: every call is metered
// (0016-MADR D9).
func (p *TogetherProvider) DiscoverModels(ctx context.Context) ([]string, error) {
	listed, err := listTogetherModels(ctx, p.apiKey, p.identity.apply(ProviderConfig{
		HTTPClient:       p.client,
		BaseURL:          p.baseURL,
		ModelProfile:     p.modelProfile,
		ModelMetadataURL: p.metadataURL,
	}))
	if err != nil || len(listed) == 0 {
		listed = StaticModels(ProviderTogether)
	}
	return listed, nil
}
