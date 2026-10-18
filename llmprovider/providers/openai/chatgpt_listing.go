package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	models, err := p.fetchCatalog(req, "chatgpt")
	if err != nil {
		return nil, err
	}
	return rankChatGPTModels(models)
}

// listPlan returns a plan session's catalog: the "models" array of the
// platform's GET /models, the entries with visibility "list", in the
// listing's order, once each (0043-MADR D5, M6). A failure is an error, as
// listChatGPT's is.
func (p *provider) listPlan(ctx context.Context, token llmprovider.Token) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, chatGPTListingTimeout)
	defer cancel()
	// The request is the one catalog's ListRaw sends (0041-MADR D9).
	req, err := chatgpt.PlanModelsRequest(ctx, p.baseURL, p.userAgent, token)
	if err != nil {
		return nil, err
	}
	models, err := p.fetchCatalog(req, "plan")
	if err != nil {
		return nil, err
	}
	if models == nil {
		// The platform's "data" array is a key's listing, not the account's.
		return nil, fmt.Errorf(`%w: model listing: the plan catalog has no "models" array`, llmprovider.ErrIncomplete)
	}
	var out []string
	for _, model := range models {
		slug := strings.TrimSpace(model.Slug)
		if slug != "" && strings.EqualFold(strings.TrimSpace(model.Visibility), "list") && !slices.Contains(out, slug) {
			out = append(out, slug)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: model listing: the plan catalog listed no models", llmprovider.ErrIncomplete)
	}
	return out, nil
}

// fetchCatalog sends req and reads its "models" array, nil when the reply
// has none; what names the catalog in errors.
func (p *provider) fetchCatalog(req *http.Request, what string) ([]chatGPTCatalogModel, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model listing: %s models: %w", what, err)
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
		return nil, fmt.Errorf("%w: model listing: read %s catalog: %w", llmprovider.ErrIncomplete, what, err)
	}
	if len(raw) > chatGPTListingLimit {
		return nil, fmt.Errorf("%w: model listing: the %s catalog is larger than %d MiB", llmprovider.ErrIncomplete,
			what, chatGPTListingLimit>>20)
	}
	var payload struct {
		Models []chatGPTCatalogModel `json:"models"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: model listing: decode %s catalog: %w", llmprovider.ErrIncomplete, what, err)
	}
	return payload.Models, nil
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
