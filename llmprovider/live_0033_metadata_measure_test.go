//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"
)

// 0033-PLAN-public-model-metadata.md Phase 1: the fields each source sets,
// for 0033-MADR's amended D1 and D2. It runs only with LLMPROVIDER_LIVE_0033
// set. Keys are read from their variables and sent only in request headers;
// only field names, shapes and public prices are logged. Nothing is billed:
// it reads the models.dev document, the listings, and a local Ollama.

// metadataSections0033 maps each provider to its models.dev section.
var metadataSections0033 = map[llmprovider.ProviderID]string{
	llmprovider.ProviderOpencodeZen: "opencode",
	llmprovider.ProviderOpencodeGo:  "opencode-go",
	llmprovider.ProviderHuggingFace: "huggingface",
	llmprovider.ProviderTogether:    "togetherai",
	llmprovider.ProviderOpenAI:      "openai",
	llmprovider.ProviderClaude:      "anthropic",
	llmprovider.ProviderGemini:      "google",
	llmprovider.ProviderGrok:        "xai",
	llmprovider.ProviderKilo:        "kilo",
}

// d1Keys0033 are the entry fields amended D1 reads, as dotted paths.
var d1Keys0033 = []string{
	"name", "family", "limit.context", "limit.input", "limit.output", "modalities.input", "modalities.output",
	"tool_call", "temperature", "structured_output", "reasoning", "reasoning_options",
	"cost.input", "cost.output", "cost.cache_read", "cost.cache_write", "cost.reasoning", "cost.tiers",
	"cost.context_over_200k", "release_date", "status",
}

// knownTopKeys0033 are the entry keys D1 reads or the SDK already knows.
var knownTopKeys0033 = []string{
	"id", "name", "family", "limit", "modalities", "tool_call", "temperature", "structured_output", "reasoning",
	"reasoning_options", "cost", "release_date", "status", "attachment", "last_updated", "knowledge",
	"open_weights", "interleaved", "provider",
}

// has0033 reports whether the dotted path is present, and not null, in entry.
func has0033(entry map[string]json.RawMessage, path string) bool {
	head, rest, nested := strings.Cut(path, ".")
	raw, ok := entry[head]
	if !ok || string(raw) == "null" {
		return false
	}
	if !nested {
		return true
	}
	var inner map[string]json.RawMessage
	if json.Unmarshal(raw, &inner) != nil {
		return false
	}
	return has0033(inner, rest)
}

// countLine renders counts as "k=n" pairs, sorted by key.
func countLine(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%d", k, counts[k])
	}
	return strings.Join(parts, " ")
}

// TestLive_0033MetadataSources is Phase 1, steps 1 to 3.
func TestLive_0033MetadataSources(t *testing.T) {
	require0033(t)
	t.Run("models.dev", measureModelsDev0033)
	t.Run("claude", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
		for _, id := range catalog.Static(llmprovider.ProviderClaude)[:2] {
			_, body := getListing(t, "https://api.anthropic.com/v1/models/"+id, map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"})
			var m map[string]json.RawMessage
			if err := json.Unmarshal(body, &m); err != nil {
				t.Fatalf("claude %s: %v", id, err)
			}
			t.Logf("S claude %s: max_input_tokens %s, max_tokens %s, capabilities %s", id,
				compact(m["max_input_tokens"], 20), compact(m["max_tokens"], 20), compact(m["capabilities"], 1200))
		}
	})
	t.Run("gemini", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGemini])
		for _, id := range catalog.Static(llmprovider.ProviderGemini)[:2] {
			_, body := getListing(t, "https://generativelanguage.googleapis.com/v1beta/models/"+id, map[string]string{"x-goog-api-key": key})
			var m map[string]json.RawMessage
			if err := json.Unmarshal(body, &m); err != nil {
				t.Fatalf("gemini %s: %v", id, err)
			}
			t.Logf("S gemini %s: inputTokenLimit %s, outputTokenLimit %s, thinking %s", id,
				compact(m["inputTokenLimit"], 20), compact(m["outputTokenLimit"], 20), compact(m["thinking"], 20))
		}
	})
	t.Run("grok", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGrok])
		_, body := getListing(t, "https://api.x.ai/v1/models", map[string]string{"Authorization": "Bearer " + key})
		var r struct {
			Data []map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("grok listing: %v", err)
		}
		logListingEntries0033(t, "grok", r.Data, catalog.Static(llmprovider.ProviderGrok)[:2], func(m map[string]json.RawMessage) string {
			parts := []string{"context_length " + compact(m["context_length"], 20), "capabilities " + compact(m["capabilities"], 300),
				"long_context_threshold " + compact(m["long_context_threshold"], 20)}
			for _, k := range sortedKeys0033(m) {
				if strings.Contains(k, "price") {
					parts = append(parts, k+" "+compact(m[k], 40))
				}
			}
			return strings.Join(parts, ", ")
		})
	})
	t.Run("openai", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI])
		_, body := getListing(t, "https://api.openai.com/v1/models", map[string]string{"Authorization": "Bearer " + key})
		var r struct {
			Data []map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("openai listing: %v", err)
		}
		withDate := 0
		for _, m := range r.Data {
			if has0033(m, "shutdown_date") {
				withDate++
			}
		}
		t.Logf("S openai: %d models, %d with a shutdown_date", len(r.Data), withDate)
		logListingEntries0033(t, "openai", r.Data, catalog.Static(llmprovider.ProviderOpenAI)[:2], func(m map[string]json.RawMessage) string {
			return "shutdown_date " + compact(m["shutdown_date"], 40) + ", keys " + strings.Join(sortedKeys0033(m), " ")
		})
	})
	t.Run("kilo", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, "KILO_API_KEY")
		_, body := getListing(t, kiloendpoint.BaseURL+"/models", map[string]string{"Authorization": "Bearer " + key})
		var r struct {
			Data []map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("kilo listing: %v", err)
		}
		pricingKeys, minusOne := map[string]int{}, 0
		for _, m := range r.Data {
			var p map[string]json.RawMessage
			_ = json.Unmarshal(m["pricing"], &p)
			for k, v := range p {
				pricingKeys[k]++
				if string(v) == `"-1"` {
					minusOne++
				}
			}
		}
		t.Logf("S kilo: %d models; pricing keys %s; %d prices of \"-1\"", len(r.Data), countLine(pricingKeys), minusOne)
		logListingEntries0033(t, "kilo", r.Data, catalog.Static(llmprovider.ProviderKilo)[:2], func(m map[string]json.RawMessage) string {
			return fmt.Sprintf("context_length %s, top_provider %s, pricing %s, expiration_date %s", compact(m["context_length"], 20),
				compact(m["top_provider"], 200), compact(m["pricing"], 400), compact(m["expiration_date"], 40))
		})
	})
	t.Run("together", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderTogether])
		_, body := getListing(t, "https://api.together.ai/v1/models", map[string]string{"Authorization": "Bearer " + key})
		var r []map[string]json.RawMessage
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("together listing: %v", err)
		}
		logListingEntries0033(t, "together", r, catalog.Static(llmprovider.ProviderTogether)[:2], func(m map[string]json.RawMessage) string {
			return fmt.Sprintf("context_length %s, pricing %s", compact(m["context_length"], 20), compact(m["pricing"], 300))
		})
	})
	t.Run("ollama", measureOllama0033)
}

// sortedKeys0033 is m's keys, sorted.
func sortedKeys0033(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// logListingEntries0033 logs describe for each id's entry in a listing whose
// entries carry "id", or that the listing lacks it.
func logListingEntries0033(t *testing.T, name string, entries []map[string]json.RawMessage, ids []string,
	describe func(map[string]json.RawMessage) string) {
	t.Helper()
	byID := map[string]map[string]json.RawMessage{}
	for _, m := range entries {
		var id string
		if json.Unmarshal(m["id"], &id) == nil {
			byID[id] = m
		}
	}
	for _, id := range ids {
		if m, ok := byID[id]; ok {
			t.Logf("S %s %s: %s", name, id, describe(m))
		} else {
			t.Logf("S %s %s: NOT LISTED", name, id)
		}
	}
}

// measureModelsDev0033 is step 1: the nine sections, every static id's D1
// fields, and the shapes across the sections.
func measureModelsDev0033(t *testing.T) {
	status, body := getListing(t, "https://models.opencode.ai/api.json", nil)
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("models.dev: status %d: %v", status, err)
	}
	t.Logf("S models.dev: status %d, %d bytes, %d sections", status, len(body), len(doc))

	sections := map[string]map[string]map[string]json.RawMessage{}
	total, start := 0, time.Now()
	for _, key := range metadataSections0033 {
		raw, ok := doc[key]
		if !ok {
			t.Logf("S models.dev section %s: ABSENT", key)
			continue
		}
		var sec struct {
			Models map[string]map[string]json.RawMessage `json:"models"`
		}
		if err := json.Unmarshal(raw, &sec); err != nil {
			t.Fatalf("models.dev section %s: %v", key, err)
		}
		sections[key] = sec.Models
		total += len(raw)
	}
	t.Logf("S models.dev: the nine sections are %d bytes, decoded generically in %s", total, time.Since(start).Round(time.Millisecond))

	keys := make([]string, 0, len(sections))
	for k := range sections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	dateForm := regexp.MustCompile(`^"\d{4}-\d{2}(-\d{2})?"$`)
	for _, key := range keys {
		types, statuses, dates, unknown := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
		tiers, over200k := 0, 0
		for _, entry := range sections[key] {
			var opts []struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(entry["reasoning_options"], &opts) == nil {
				for _, o := range opts {
					types[o.Type]++
				}
			}
			statuses[compact(entry["status"], 20)]++
			switch raw := entry["release_date"]; {
			case raw == nil:
				dates["absent"]++
			case dateForm.Match(raw) && len(raw) == 9:
				dates["YYYY-MM"]++
			case dateForm.Match(raw):
				dates["YYYY-MM-DD"]++
			default:
				dates["other"]++
			}
			if has0033(entry, "cost.tiers") {
				tiers++
			}
			if has0033(entry, "cost.context_over_200k") {
				over200k++
			}
			for k := range entry {
				if !slices.Contains(knownTopKeys0033, k) {
					unknown[k]++
				}
			}
		}
		t.Logf("S models.dev section %s: %d models; reasoning_options %s; status %s; release_date %s; tiers %d; context_over_200k %d; other keys %s",
			key, len(sections[key]), countLine(types), countLine(statuses), countLine(dates), tiers, over200k, countLine(unknown))
	}

	providerIDs := make([]llmprovider.ProviderID, 0, len(metadataSections0033))
	for p := range metadataSections0033 {
		providerIDs = append(providerIDs, p)
	}
	slices.Sort(providerIDs)
	for _, p := range providerIDs {
		key := metadataSections0033[p]
		for _, id := range catalog.Static(p) {
			entry, ok := sections[key][id]
			if !ok {
				t.Logf("S models.dev %s %s: NOT IN SECTION %s", p, id, key)
				continue
			}
			var set, absent []string
			for _, path := range d1Keys0033 {
				if has0033(entry, path) {
					set = append(set, path)
				} else {
					absent = append(absent, path)
				}
			}
			t.Logf("S models.dev %s %s: absent %s (set %d of %d)", p, id, strings.Join(absent, " "), len(set), len(d1Keys0033))
		}
	}
}

// measureOllama0033 is step 3: one local model's /api/show.
func measureOllama0033(t *testing.T) {
	_, body := getListing(t, "http://localhost:11434/api/tags", nil)
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &tags); err != nil || len(tags.Models) == 0 {
		t.Skipf("no local Ollama model: %v", err)
	}
	name := tags.Models[0].Name
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:11434/api/show",
		bytes.NewReader([]byte(fmt.Sprintf(`{"model":%q}`, name))))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ollama /api/show: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		t.Fatal(err)
	}
	var show struct {
		ModelInfo    map[string]json.RawMessage `json:"model_info"`
		Capabilities []string                   `json:"capabilities"`
		Details      map[string]json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(raw, &show); err != nil {
		t.Fatalf("ollama /api/show: status %d: %v", resp.StatusCode, err)
	}
	var arch string
	_ = json.Unmarshal(show.ModelInfo["general.architecture"], &arch)
	t.Logf("S ollama %s: status %d, general.architecture %q, %s.context_length %s, capabilities %v, details.context_length %s",
		name, resp.StatusCode, arch, arch, compact(show.ModelInfo[arch+".context_length"], 20), show.Capabilities,
		compact(show.Details["context_length"], 20))
}
