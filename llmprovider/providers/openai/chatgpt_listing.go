package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
)

// chatGPTListingTimeout bounds one ChatGPT listing, as catalog bounds every
// other (MADR 0013 A5).
const chatGPTListingTimeout = 10 * time.Second

// chatgptModelsClientVersion is the client_version a build with no release
// version sends (internal/chatgpt.ModelsClientVersion).
const chatgptModelsClientVersion = chatgpt.ModelsClientVersion

type chatGPTCatalogModel struct {
	Slug       string `json:"slug"`
	Visibility string `json:"visibility"`
	Priority   int    `json:"priority"`
	Supported  *bool  `json:"supported_in_api"`
}

// listChatGPT returns the live Codex catalog for a ChatGPT session, by
// priority. A failure is an error: no static OpenAI catalog stands in, because
// those models may not be available to the account (MADR 0008 D11).
func (p *provider) listChatGPT(ctx context.Context, token llmprovider.Token) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, chatGPTListingTimeout)
	defer cancel()
	// The request is the one catalog's ListRaw sends (0041-MADR D9).
	req, err := chatgpt.ModelsRequest(ctx, p.baseURL, p.userAgent, token, p.src)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model listing: chatgpt models: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Debug("llmprovider: openai: close listing body", "error", err)
		}
	}()
	// Classified like any answer, so a 401 can renew the token (0020-MADR F38).
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOpenAI), resp); err != nil {
		return nil, fmt.Errorf("model listing: %w", err)
	}

	// The catalog's listing cap, and a kind for a reply that cannot be used
	// (0021-MADR C3; 0026-MADR F34).
	raw, err := io.ReadAll(io.LimitReader(resp.Body, chatGPTListingLimit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: model listing: read chatgpt catalog: %w", llmprovider.ErrIncomplete, err)
	}
	if len(raw) > chatGPTListingLimit {
		return nil, fmt.Errorf("%w: model listing: the chatgpt catalog is larger than %d MiB", llmprovider.ErrIncomplete,
			chatGPTListingLimit>>20)
	}
	var payload struct {
		Models []chatGPTCatalogModel `json:"models"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: model listing: decode chatgpt catalog: %w", llmprovider.ErrIncomplete, err)
	}
	return rankChatGPTModels(payload.Models)
}

// chatGPTListingLimit bounds the ChatGPT listing, as catalog bounds one
// listing page (0021-MADR C3).
const chatGPTListingLimit = 8 << 20

// rankChatGPTModels keeps the listed models, by priority then listing order,
// once each.
func rankChatGPTModels(models []chatGPTCatalogModel) ([]string, error) {
	type ranked struct {
		slug     string
		priority int
		order    int
	}
	var listed []ranked
	for i, model := range models {
		if chatGPTCatalogModelListed(model) {
			listed = append(listed, ranked{slug: model.Slug, priority: model.Priority, order: i})
		}
	}
	slices.SortStableFunc(listed, func(a, b ranked) int {
		if a.priority != b.priority {
			return a.priority - b.priority
		}
		return a.order - b.order
	})
	out := make([]string, 0, len(listed))
	seen := make(map[string]struct{}, len(listed))
	for _, model := range listed {
		if _, duplicate := seen[model.slug]; duplicate {
			continue
		}
		seen[model.slug] = struct{}{}
		out = append(out, model.slug)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: model listing: chatgpt catalog listed no models", llmprovider.ErrIncomplete)
	}
	return out, nil
}

// chatGPTCatalogModelListed keeps a model with a slug, usable in the API, that
// the catalog lists.
func chatGPTCatalogModelListed(model chatGPTCatalogModel) bool {
	if strings.TrimSpace(model.Slug) == "" {
		return false
	}
	if model.Supported != nil && !*model.Supported {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(model.Visibility)) {
	case "", "list":
		return true
	default:
		return false
	}
}
