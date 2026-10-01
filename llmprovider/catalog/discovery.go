package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// chatgptModelsClientVersion is the client_version GET .../codex/models is
// sent when this module's build has no release version. The backend requires the
// parameter; 0.0.0 is accepted but hides models whose minimal_client_version
// is higher (gpt-6-sol and gpt-6-luna on 2026-09-27).
const chatgptModelsClientVersion = "0.0.0"

// chatgptVersionRE reads X.Y.Z from a module version: the only form the
// backend accepts ("v1.5.0", "1.5" and "(devel)" answer 400, as does anything
// over 32 characters; measured 2026-09-27). A pseudo-version names the
// release it precedes.
var chatgptVersionRE = regexp.MustCompile(`^v?(\d{1,9}\.\d{1,9}\.\d{1,9})(?:[-+].*)?$`)

// chatgptClientVersion is the client_version for this module at version v: its
// own release, never a Codex version string (MADR 0012 §4.3), else
// chatgptModelsClientVersion.
func chatgptClientVersion(v string) string {
	if m := chatgptVersionRE.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return chatgptModelsClientVersion
}

// Listing pagination (MADR 0007 §2): Gemini and Anthropic page their model
// lists, so each fetch requests the maximum page size and follows at most
// maxListingPages pages.
const (
	maxListingPages     = 10
	geminiListPageSize  = "1000"
	claudeListPageLimit = "1000"
)

// modelListingTimeout bounds one model listing, its metadata fetch included
// (MADR 0009 §2). List and every DiscoverModels listing
// apply it (MADR 0013 A5).
const modelListingTimeout = 10 * time.Second

// Catalog is the result of one model listing, viewed two ways.
type Catalog struct {
	// Recommended is what List returns: at most
	// MaxListed ids, curated against the static catalog.
	Recommended []string
	// Usable is every id the provider's usability filters admit, once each,
	// in listing order, uncapped. It equals Recommended when Live is false.
	Usable []string
	// Live reports whether Usable came from the provider's listing rather
	// than the static catalog.
	Live bool
	// Err is the listing failure that made Live false. It is nil when Live is
	// true, and when a listing that succeeded yielded no usable id (MADR 0013
	// C2).
	Err error
}

// List performs one model listing for id and returns both the curated
// recommendation and every usable id (MADR 0007 §1). src is the credential: a
// key is llmprovider.NewStaticToken(key). It curates the listing against the
// static catalog so configure UIs never show huge unusable lists (embeddings,
// TTS, Live, image, dated previews, …), spends no generation, and is bounded
// at 10 s.
//
// opts are llmprovider's common options, with WithProfile and
// WithKiloOrganization. A failed listing degrades to the static catalog with a
// nil error and Live false; only Ollama, an unknown provider, a missing
// source, a token failure or a refused option return an error.
func List(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, opts ...llmprovider.Option) (Catalog, error) {
	cfg, err := configFor(id, opts)
	if err != nil {
		return Catalog{}, err
	}
	providerName := string(id)

	ctx, cancel := context.WithTimeout(ctx, modelListingTimeout)
	defer cancel()
	if strings.EqualFold(providerName, llmprovider.ProviderOpenAI) && llmprovider.IsChatGPTSession(src) {
		return listChatGPTModels(ctx, src, cfg)
	}
	if src == nil {
		return Catalog{}, errors.New("model listing: TokenSource is required")
	}
	token, err := src.Token(ctx)
	if err != nil {
		return Catalog{}, fmt.Errorf("model listing: acquire token: %w", err)
	}
	return modelCatalogFor(ctx, providerName, token, cfg)
}

// catalogFrom applies the degrade-to-static contract every lister except
// Ollama has always had: a failed fetch, or one that yields no usable id,
// substitutes the static catalog.
func catalogFrom(usable []string, fetchErr error, static []string, curate func([]string) []string) Catalog {
	usable = uniqueIDs(usable)
	if fetchErr != nil || len(usable) == 0 {
		cat := staticCatalog(static)
		cat.Err = fetchErr
		return cat
	}
	recommended := curate(usable)
	if len(recommended) == 0 {
		return staticCatalog(static)
	}
	return Catalog{Recommended: recommended, Usable: usable, Live: true}
}

// uniqueIDs drops repeated ids, keeping the first of each (MADR 0013 A1).
func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// staticCatalog wraps a caller-owned copy of a static catalog.
func staticCatalog(static []string) Catalog {
	return Catalog{Recommended: static, Usable: slices.Clone(static), Live: false}
}

type chatGPTCatalogModel struct {
	Slug       string `json:"slug"`
	Visibility string `json:"visibility"`
	Priority   int    `json:"priority"`
	Supported  *bool  `json:"supported_in_api"`
}

// listChatGPTModels returns the live Codex catalog for a ChatGPT OAuth
// session. Unlike API-key providers, failure is not replaced by a static
// OpenAI catalog because those models may not be available to the account.
func listChatGPTModels(ctx context.Context, src llmprovider.TokenSource, cfg config) (Catalog, error) {
	token, err := src.Token(ctx)
	if err != nil {
		return Catalog{}, fmt.Errorf("model listing: acquire token: %w", err)
	}
	baseURL := llmprovider.DefaultOpenAIChatGPTBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	endpoint, err := url.Parse(baseURL + "/models")
	if err != nil {
		return Catalog{}, fmt.Errorf("model listing: parse chatgpt models URL: %w", err)
	}
	query := endpoint.Query()
	sdk, _ := transport.BuildVersions()
	query.Set("client_version", chatgptClientVersion(sdk))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return Catalog{}, fmt.Errorf("model listing: create chatgpt models request: %w", err)
	}
	cfg.setUserAgent(req)
	token.Apply(req, headerAuthorization, bearerScheme)
	req.Header.Set(llmprovider.ChatGPTOriginatorHeader, llmprovider.ChatGPTOriginatorValue)
	if accountID := llmprovider.ChatGPTSessionAccountID(src); accountID != "" {
		req.Header.Set(llmprovider.ChatGPTAccountHeader, accountID)
	}
	if llmprovider.ChatGPTSessionFedRAMP(src) {
		req.Header.Set(llmprovider.ChatGPTFedRAMPHeader, "true")
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return Catalog{}, fmt.Errorf("model listing: chatgpt models: %w", err)
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return Catalog{}, fmt.Errorf("model listing: chatgpt HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Models []chatGPTCatalogModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Catalog{}, fmt.Errorf("model listing: decode chatgpt catalog: %w", err)
	}

	type ranked struct {
		slug     string
		priority int
		order    int
	}
	var listed []ranked
	for i, model := range payload.Models {
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
		return Catalog{}, errors.New("model listing: chatgpt catalog listed no models")
	}
	return Catalog{Recommended: out, Usable: slices.Clone(out), Live: true}, nil
}

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

// modelCatalogFor dispatches one provider's fetch and curation. The caller
// owns the timeout. Every lister sends token as it describes itself
// (0016-MADR A6).
func modelCatalogFor(ctx context.Context, providerName string, token llmprovider.Token, cfg config) (Catalog, error) {
	switch p := strings.ToLower(providerName); p {
	case llmprovider.ProviderGemini:
		usable, err := fetchGeminiUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderGemini), curateGemini), nil
	case llmprovider.ProviderOpenAI:
		usable, err := fetchOpenAIUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderOpenAI), curateOpenAI), nil
	case llmprovider.ProviderClaude:
		usable, err := fetchClaudeUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderClaude), curateClaude), nil
	case llmprovider.ProviderGrok:
		usable, err := fetchGrokUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderGrok), curateGrok), nil
	case llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		return opencodeCatalog(ctx, p, token, cfg)
	case llmprovider.ProviderHuggingFace:
		meta := startModelMetadata(ctx, cfg)
		usable, err := fetchHuggingFaceUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderHuggingFace),
			metadataCurate(llmprovider.ProviderHuggingFace, cfg.ModelProfile, meta, curateHuggingFace)), nil
	case llmprovider.ProviderKilo:
		entries, err := fetchKiloCatalog(ctx, token, cfg)
		// Deliberate: a failed Kilo fetch degrades to the static catalog rather
		// than failing, like every other lister here. See catalogFrom.
		return catalogFrom(kiloUsable(entries), err, Static(llmprovider.ProviderKilo), kiloCurate(entries, cfg.ModelProfile)), nil
	case llmprovider.ProviderTogether:
		meta := startModelMetadata(ctx, cfg)
		usable, err := fetchTogetherUsable(ctx, token, cfg)
		return catalogFrom(usable, err, Static(llmprovider.ProviderTogether),
			metadataCurate(llmprovider.ProviderTogether, cfg.ModelProfile, meta, curateTogether)), nil
	case llmprovider.ProviderOllama:
		return ollamaCatalog(ctx, token, cfg)
	default:
		return Catalog{}, fmt.Errorf("unsupported provider for model listing: %s", providerName)
	}
}

// geminiModelsPage is one page of Gemini's GET {base}/models.
type geminiModelsPage struct {
	Models []struct {
		Name                       string   `json:"name"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
	NextPageToken string `json:"nextPageToken"`
}

// fetchGeminiUsable returns the usable Gemini text models in listing order,
// following nextPageToken for at most maxListingPages pages (MADR 0007 §2).
// Any page failure, or running out of pages, fails the whole listing.
func fetchGeminiUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := "https://generativelanguage.googleapis.com/v1beta"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	var available []string
	pageToken := ""
	for page := 1; page <= maxListingPages; page++ {
		query := url.Values{"pageSize": {geminiListPageSize}}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		endpoint := baseURL + "/models?" + query.Encode()
		result, err := fetchGeminiPage(ctx, endpoint, token, cfg)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Models {
			id := strings.TrimPrefix(m.Name, "models/")
			if isUsableGeminiTextModel(id, m.SupportedGenerationMethods) {
				available = append(available, id)
			}
		}
		if result.NextPageToken == "" {
			return available, nil
		}
		pageToken = result.NextPageToken
	}
	return nil, fmt.Errorf("model listing: more than %d pages", maxListingPages)
}

// fetchGeminiPage performs one listing request. The key travels in the
// x-goog-api-key header, or the token's own, never the query string.
func fetchGeminiPage(ctx context.Context, endpoint string, token llmprovider.Token, cfg config) (geminiModelsPage, error) {
	var result geminiModelsPage
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return result, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, "x-goog-api-key", "")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return result, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("gemini: models endpoint returned HTTP %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func curateGemini(usable []string) []string {
	return curateFromCatalog(staticGemini, usable, func(s string) bool {
		return isUsableGeminiTextModel(s, []string{methodGenerateContent})
	}, rankGeminiModel)
}

// fetchOpenAIUsable returns the usable OpenAI chat models in listing order.
func fetchOpenAIUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := "https://api.openai.com/v1"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	header, value := tokenHeader(token, headerAuthorization, bearerScheme)
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, llmprovider.ProviderOpenAI)
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableOpenAIChatModel), nil
}

func curateOpenAI(usable []string) []string {
	return curateFromCatalog(staticOpenAI, usable, isUsableOpenAIChatModel, rankOpenAIModel)
}

// claudeModelsPage is one page of Anthropic's GET /v1/models.
type claudeModelsPage struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	HasMore bool   `json:"has_more"`
	LastID  string `json:"last_id"`
}

// fetchClaudeUsable returns the usable Claude text models in listing order,
// following has_more/last_id for at most maxListingPages pages (MADR 0007 §2).
// Any page failure, or running out of pages, fails the whole listing.
func fetchClaudeUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := "https://api.anthropic.com"
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}

	var available []string
	afterID := ""
	for page := 1; page <= maxListingPages; page++ {
		query := url.Values{"limit": {claudeListPageLimit}}
		if afterID != "" {
			query.Set("after_id", afterID)
		}
		result, err := fetchClaudePage(ctx, baseURL+"/v1/models?"+query.Encode(), token, cfg)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Data {
			if isUsableClaudeTextModel(m.ID) {
				available = append(available, m.ID)
			}
		}
		if !result.HasMore {
			return available, nil
		}
		if result.LastID == "" {
			return nil, errors.New("model listing: has_more without last_id")
		}
		afterID = result.LastID
	}
	return nil, fmt.Errorf("model listing: more than %d pages", maxListingPages)
}

// fetchClaudePage performs one listing request.
func fetchClaudePage(ctx context.Context, endpoint string, token llmprovider.Token, cfg config) (claudeModelsPage, error) {
	var result claudeModelsPage
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return result, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, "x-api-key", "")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return result, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		// Older keys / regional proxies may not support Models API.
		return result, fmt.Errorf("claude: models endpoint returned HTTP %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

func curateClaude(usable []string) []string {
	return curateFromCatalog(staticClaude, usable, isUsableClaudeTextModel, rankClaudeModel)
}

// ollamaCatalog lists every installed model. Ollama has no static catalog, so
// its errors are returned rather than degraded, and an empty install is a
// successful (Live) listing.
func ollamaCatalog(ctx context.Context, token llmprovider.Token, cfg config) (Catalog, error) {
	names, err := fetchOllamaNames(ctx, token, cfg)
	if err != nil {
		return Catalog{}, err
	}
	recommended := names
	if len(names) > MaxListed {
		recommended = slices.Clone(names[:MaxListed])
	}
	return Catalog{Recommended: recommended, Usable: names, Live: true}, nil
}

// fetchOllamaNames returns every installed model name in /api/tags order.
// Ollama takes no credential, so a token is sent only where it names its own
// Header (R16; 0016-MADR D2, A6).
func fetchOllamaNames(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := "http://localhost:11434"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	cfg.setUserAgent(req)
	if token.Header != "" {
		token.Apply(req, "", "")
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach Ollama at %s: %w", baseURL, err)
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// ValidateOllamaURL checks if an Ollama instance is reachable at the given URL
// by calling GET /api/version. Returns nil on success, error on failure.
func ValidateOllamaURL(ctx context.Context, baseURL string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/version", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	defaultConfig().setUserAgent(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach Ollama at %s: %w", baseURL, err)
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// fetchGrokUsable returns the usable Grok models in listing order.
func fetchGrokUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := "https://api.x.ai/v1"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	header, value := tokenHeader(token, headerAuthorization, bearerScheme)
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, llmprovider.ProviderGrok)
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableGrokModel), nil
}

func curateGrok(usable []string) []string {
	return curateFromCatalog(staticGrok, usable, isUsableGrokModel, rankGrokModel)
}

// fetchDataIDs performs GET endpoint and decodes a {"data":[{"id":…}]} body.
// The credential is sent as header: value when value is non-empty.
func fetchDataIDs(ctx context.Context, endpoint, header, value string, cfg config, provider string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if provider == serviceOpencode {
		req.Header.Set(opencodeSessionHeader, cfg.session) // 0013 B7
	}
	if value != "" {
		req.Header.Set(header, value)
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: models endpoint returned HTTP %d", provider, resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// filterIDs keeps the ids usable admits, preserving order.
func filterIDs(ids []string, usable func(string) bool) []string {
	var out []string
	for _, id := range ids {
		if usable(id) {
			out = append(out, id)
		}
	}
	return out
}

// opencodeCatalog lists one OpenCode gateway. An unknown gateway is an error,
// not a degradation. The OpenCode /models endpoint is PUBLIC — it answers 200
// with no credentials (verified 2026-08-28) — so the credential is sent only
// when there is one, and an empty key is not an error. The listing carries no
// routing or capability metadata (every entry reports owned_by "opencode"), so
// route selection cannot be derived from it; see providers/opencode.
func opencodeCatalog(ctx context.Context, gateway string, token llmprovider.Token, cfg config) (Catalog, error) {
	if _, err := opencodeBaseURL(gateway); err != nil {
		return Catalog{}, err
	}
	meta := startModelMetadata(ctx, cfg)
	usable, fetchErr := fetchOpencodeUsable(ctx, gateway, token, cfg)
	curate := func(usable []string) []string {
		return curateFromCatalog(staticOpencodeCatalog(gateway), usable, isUsableOpencodeModel, rankOpencodeModel)
	}
	return catalogFrom(usable, fetchErr, Static(gateway), metadataCurate(gateway, cfg.ModelProfile, meta, curate)), nil
}

// fetchOpencodeUsable returns the usable gateway models in listing order.
func fetchOpencodeUsable(ctx context.Context, gateway string, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL, err := opencodeBaseURL(gateway)
	if err != nil {
		return nil, err
	}
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	header, value := headerAuthorization, ""
	if token.Value != "" {
		header, value = tokenHeader(token, headerAuthorization, bearerScheme)
	}
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, "opencode")
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableOpencodeModel), nil
}

// onlyText reports whether a modality list is exactly ["text"].
func onlyText(mods []string) bool { return len(mods) == 1 && mods[0] == jsonKeyText }

// hasText reports whether a modality list includes "text". A model that also
// accepts images or files still serves a text prompt (MADR 0007 §1b).
func hasText(mods []string) bool { return slices.Contains(mods, jsonKeyText) }

// fetchHuggingFaceUsable returns the usable router models, fastest first: input
// must include text and output must be exactly text (MADR 0007 §1b), and at
// least one provider offering must be live. The endpoint is PUBLIC (200 with
// no credential, verified 2026-08-29), so the credential is optional.
//
// Unlike the other providers, ranking here uses measured figures rather than
// name heuristics: the listing reports throughput (tokens/sec) and
// first_token_latency_ms per provider offering. The sorted order is handed to
// curateFromCatalog with a nil rankFn, which preserves it.
func fetchHuggingFaceUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := huggingFaceBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if token.Value != "" {
		token.Apply(req, headerAuthorization, bearerScheme)
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("huggingface: models endpoint returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID           string `json:"id"`
			Architecture struct {
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Providers []struct {
				Status              string  `json:"status"`
				SupportsTools       bool    `json:"supports_tools"`
				Throughput          float64 `json:"throughput"`
				FirstTokenLatencyMs float64 `json:"first_token_latency_ms"`
			} `json:"providers"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	type scored struct {
		id   string
		tps  float64
		ttft float64
	}
	var ranked []scored
	for _, m := range result.Data {
		if !hasText(m.Architecture.InputModalities) || !onlyText(m.Architecture.OutputModalities) {
			continue
		}
		if !isUsableHuggingFaceModel(m.ID) {
			continue
		}
		best := scored{id: m.ID, ttft: math.MaxFloat64}
		live := false
		for _, pr := range m.Providers {
			if pr.Status != "live" {
				continue
			}
			live = true
			if pr.Throughput > best.tps {
				best.tps = pr.Throughput
			}
			if pr.FirstTokenLatencyMs > 0 && pr.FirstTokenLatencyMs < best.ttft {
				best.ttft = pr.FirstTokenLatencyMs
			}
		}
		if live {
			ranked = append(ranked, best)
		}
	}
	// Fastest first; ties broken by lowest time-to-first-token.
	slices.SortStableFunc(ranked, func(a, b scored) int {
		switch {
		case a.tps > b.tps:
			return -1
		case a.tps < b.tps:
			return 1
		case a.ttft < b.ttft:
			return -1
		case a.ttft > b.ttft:
			return 1
		}
		return 0
	})
	available := make([]string, 0, len(ranked))
	for _, r := range ranked {
		available = append(available, r.id)
	}
	return available, nil
}

// togetherListingLimit bounds Together's model listing, which also lists
// image, audio and embedding models.
const togetherListingLimit = 8 << 20

// fetchTogetherUsable returns Together's chat models in listing order.
// GET {base}/models answers with a bare JSON array, not an OpenAI
// {"data": [...]} envelope, and lists every model type; only "chat" models
// can serve a Chat Completions request (0017-REPORT, Together section).
func fetchTogetherUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := togetherBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, headerAuthorization, bearerScheme)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("together: models endpoint returned HTTP %d", resp.StatusCode)
	}

	var models []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, togetherListingLimit)).Decode(&models); err != nil {
		return nil, fmt.Errorf("together: decode models: %w", err)
	}
	usable := make([]string, 0, len(models))
	for _, m := range models {
		if m.Type == "chat" && m.ID != "" {
			usable = append(usable, m.ID)
		}
	}
	return usable, nil
}

// curateTogether is the fallback curation when the metadata document is
// unavailable: the static catalog's order, then the listing's.
func curateTogether(usable []string) []string {
	return curateFromCatalog(staticTogether, usable, nil, nil)
}

// curateHuggingFace passes a nil rankFn, which preserves the metadata order.
func curateHuggingFace(usable []string) []string {
	return curateFromCatalog(staticHuggingFace, usable, isUsableHuggingFaceModel, nil)
}

// kiloCatalogEntry is the subset of Kilo's OpenRouter-shaped catalog entry this
// package reads. Shared by listKiloModels, KiloModelCapabilities and the
// ranker (MADR 0009 §2).
type kiloCatalogEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Created      int64  `json:"created"`
	Architecture struct {
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
	Pricing struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
	ContextLength         int                `json:"context_length"`
	ExpirationDate        string             `json:"expiration_date"`
	PreferredIndex        *int               `json:"preferredIndex"`
	TerminalBench         *kiloTerminalBench `json:"terminalBench"`
	SupportedParameters   []string           `json:"supported_parameters"`
	MayTrainOnYourPrompts bool               `json:"mayTrainOnYourPrompts"`
}

// kiloTerminalBench is the benchmark block Kilo publishes for some models.
type kiloTerminalBench struct {
	OverallScore *float64 `json:"overallScore"`
}

// kiloPriceRank parses Kilo's string pricing into a sortable value. A negative
// or unparseable price means "variable" (the kilo-auto tiers report "-1") and
// sorts last rather than first.
func kiloPriceRank(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return math.MaxFloat64
	}
	return v
}

// fetchKiloCatalog performs the shared GET {base}/models, or an organization's
// listing (resolveKiloEndpoints). The public endpoint answers with no
// credential (verified 2026-08-29), so the token may be empty.
func fetchKiloCatalog(ctx context.Context, token llmprovider.Token, cfg config) ([]kiloCatalogEntry, error) {
	endpoints := kiloendpoint.Resolve(cfg.BaseURL, token.Value, cfg.KiloOrganization)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoints.Models, http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if token.Value != "" {
		token.Apply(req, headerAuthorization, bearerScheme)
	}
	if endpoints.Org != "" {
		req.Header.Set(kiloendpoint.OrganizationHeader, endpoints.Org)
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kilo: models endpoint returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Data []kiloCatalogEntry `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// The Kilo catalog's curation. Two documented traps are handled in kiloUsable:
//
//  1. pricing.completion is a STRING and is "-1" for the variable-priced
//     kilo-auto/{frontier,balanced,efficient} tiers. A naive ascending sort would
//     rank the most expensive tiers as cheaper than free, so kiloPriceRank sorts
//     negative and unparseable prices LAST.
//  2. Kilo model ids may legitimately END in ":free" (tencent/hy3:free), so
//     nothing is stripped at the colon. splitHuggingFaceModelPolicy must not be
//     used here.
//
// Models flagged mayTrainOnYourPrompts are excluded. That is a POLICY decision,
// not a capability filter — see isUsableKiloModel's comment.

// kiloUsable returns the usable Kilo models, cheapest first: input must include
// text and output must be exactly text (MADR 0007 §1b), tools must be supported,
// and the training policy applies.
func kiloUsable(entries []kiloCatalogEntry) []string {
	type priced struct {
		id    string
		price float64
	}
	var ranked []priced
	for _, m := range entries {
		if !hasText(m.Architecture.InputModalities) || !onlyText(m.Architecture.OutputModalities) {
			continue
		}
		if m.MayTrainOnYourPrompts { // POLICY — see isUsableKiloModel
			continue
		}
		if !slices.Contains(m.SupportedParameters, jsonKeyTools) {
			continue
		}
		if !isUsableKiloModel(m.ID) {
			continue
		}
		ranked = append(ranked, priced{id: m.ID, price: kiloPriceRank(m.Pricing.Completion)})
	}
	// Cheapest first; "-1" and unparseable prices sort last via kiloPriceRank.
	slices.SortStableFunc(ranked, func(a, b priced) int {
		switch {
		case a.price < b.price:
			return -1
		case a.price > b.price:
			return 1
		}
		return 0
	})
	available := make([]string, 0, len(ranked))
	for _, r := range ranked {
		available = append(available, r.id)
	}
	return available
}

// curateKilo passes a nil rankFn, which preserves the price ordering.
func curateKilo(usable []string) []string {
	return curateFromCatalog(staticKilo, usable, isUsableKiloModel, nil)
}

// kiloCurate ranks Kilo's usable models from the listing's own metadata
// (MADR 0009 §2). When fewer than MaxListed are eligible, the rest come
// from curateKilo's order, then the usable list.
func kiloCurate(entries []kiloCatalogEntry, profile Profile) func([]string) []string {
	return func(usable []string) []string {
		byID := make(map[string]kiloCatalogEntry, len(entries))
		for _, e := range entries {
			byID[e.ID] = e
		}
		now := rankingNow()
		cands := make([]rankCandidate, 0, len(usable))
		for _, id := range usable {
			e, ok := byID[id]
			if !ok {
				e.ID = id
			}
			cands = append(cands, kiloCandidate(e, now))
		}
		return rankRecommended(profile, llmprovider.ProviderKilo, cands, append(curateKilo(usable), usable...))
	}
}

// KiloModelCapabilities returns the supported_parameters published for one Kilo
// model, for use with kilo.WithCapabilities. The catalog endpoint is public, so
// apiKey may be empty. An empty list is a valid answer; an error means the
// catalog was unreachable or the model is absent from it.
func KiloModelCapabilities(ctx context.Context, apiKey, model string, opts ...llmprovider.Option) ([]string, error) {
	cfg, err := configFor(llmprovider.ProviderKilo, opts)
	if err != nil {
		return nil, err
	}
	entries, err := fetchKiloCatalog(ctx, llmprovider.Token{Value: apiKey}, cfg)
	if err != nil {
		return nil, err
	}
	for _, m := range entries {
		if m.ID == model {
			return m.SupportedParameters, nil
		}
	}
	return nil, fmt.Errorf("%w: kilo model %q not in catalog", llmprovider.ErrInvalidRequest, model)
}
