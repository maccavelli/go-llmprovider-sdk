//go:build live_gateways

package llmprovider_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The public listings the guard reads without a key.
const (
	guardOpencodeZenModels = "https://opencode.ai/zen/v1/models"
	guardOpencodeGoModels  = "https://opencode.ai/zen/go/v1/models"
	guardHuggingFaceModels = "https://router.huggingface.co/v1/models"
	guardKiloModels        = "https://api.kilo.ai/api/gateway/models"
)

// listed is a provider's listing: each id it offers, with why it is stale,
// or "" when it is current.
type listed map[string]string

// guardListing is one provider's listing, as the guard reads it.
type guardListing struct {
	id   llmprovider.ProviderID
	read func(t *testing.T) listed
}

// readListing GETs u and decodes it into v. A 429 or a 5xx skips: the
// service did not answer for its catalog.
func readListing(t *testing.T, u string, header map[string]string, v any) {
	t.Helper()
	status, body := getListing(t, u, header)
	if status == http.StatusTooManyRequests || status >= 500 {
		t.Skipf("listing transient: HTTP %d", status)
	}
	if status != http.StatusOK {
		t.Fatalf("listing: HTTP %d", status)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("listing: %v", err)
	}
}

// staleWhen is why a listing's lifecycle field marks a model stale: its
// value, when it is set; else "".
func staleWhen(field string, raw json.RawMessage) string {
	if s := strings.TrimSpace(string(raw)); s != "" && s != "null" {
		return field + " " + s
	}
	return ""
}

// dataIDs reads an OpenAI-style {"data":[{"id":…}]} listing.
func dataIDs(t *testing.T, u string, header map[string]string) listed {
	t.Helper()
	var r struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	readListing(t, u, header, &r)
	out := listed{}
	for _, m := range r.Data {
		out[m.ID] = ""
	}
	return out
}

func bearer(key string) map[string]string { return map[string]string{"Authorization": "Bearer " + key} }

// guardListings are the listings of every provider with a static catalog.
var guardListings = []guardListing{
	{llmprovider.ProviderOpenAI, func(t *testing.T) listed {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI])
		var r struct {
			Data []struct {
				ID           string          `json:"id"`
				ShutdownDate json.RawMessage `json:"shutdown_date"`
			} `json:"data"`
		}
		readListing(t, "https://api.openai.com/v1/models", bearer(key), &r)
		out := listed{}
		for _, m := range r.Data {
			out[m.ID] = staleWhen("shutdown_date", m.ShutdownDate)
		}
		return out
	}},
	{llmprovider.ProviderClaude, func(t *testing.T) listed {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
		out, after := listed{}, ""
		for page := 0; page < 10; page++ {
			u := "https://api.anthropic.com/v1/models?limit=1000"
			if after != "" {
				u += "&after_id=" + url.QueryEscape(after)
			}
			var r struct {
				Data []struct {
					ID        string `json:"id"`
					Lifecycle string `json:"lifecycle"`
				} `json:"data"`
				HasMore bool   `json:"has_more"`
				LastID  string `json:"last_id"`
			}
			readListing(t, u, map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}, &r)
			for _, m := range r.Data {
				out[m.ID] = ""
				if m.Lifecycle != "" && m.Lifecycle != "active" {
					out[m.ID] = "lifecycle " + m.Lifecycle
				}
			}
			if !r.HasMore || r.LastID == "" {
				break
			}
			after = r.LastID
		}
		return out
	}},
	{llmprovider.ProviderGemini, func(t *testing.T) listed {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGemini])
		out, token := listed{}, ""
		for page := 0; page < 10; page++ {
			u := "https://generativelanguage.googleapis.com/v1beta/models?pageSize=1000"
			if token != "" {
				u += "&pageToken=" + url.QueryEscape(token)
			}
			var r struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
				NextPageToken string `json:"nextPageToken"`
			}
			readListing(t, u, map[string]string{"x-goog-api-key": key}, &r)
			for _, m := range r.Models {
				out[strings.TrimPrefix(m.Name, "models/")] = ""
			}
			if r.NextPageToken == "" {
				break
			}
			token = r.NextPageToken
		}
		return out
	}},
	{llmprovider.ProviderGrok, func(t *testing.T) listed {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGrok])
		var r struct {
			Data []struct {
				ID      string   `json:"id"`
				Aliases []string `json:"aliases"`
			} `json:"data"`
		}
		readListing(t, "https://api.x.ai/v1/models", bearer(key), &r)
		out := listed{}
		for _, m := range r.Data {
			out[m.ID] = ""
			for _, a := range m.Aliases {
				out[a] = ""
			}
		}
		return out
	}},
	{llmprovider.ProviderOpencodeZen, func(t *testing.T) listed { return dataIDs(t, guardOpencodeZenModels, nil) }},
	{llmprovider.ProviderOpencodeGo, func(t *testing.T) listed { return dataIDs(t, guardOpencodeGoModels, nil) }},
	{llmprovider.ProviderHuggingFace, func(t *testing.T) listed { return dataIDs(t, guardHuggingFaceModels, nil) }},
	{llmprovider.ProviderKilo, func(t *testing.T) listed {
		var r struct {
			Data []struct {
				ID             string          `json:"id"`
				ExpirationDate json.RawMessage `json:"expiration_date"`
			} `json:"data"`
		}
		readListing(t, guardKiloModels, nil, &r)
		out := listed{}
		for _, m := range r.Data {
			out[m.ID] = staleWhen("expiration_date", m.ExpirationDate)
		}
		return out
	}},
	{llmprovider.ProviderTogether, func(t *testing.T) listed {
		key := llmprovider.LiveTogetherKey(t)
		var r []struct {
			ID string `json:"id"`
		}
		readListing(t, "https://api.together.xyz/v1/models", bearer(key), &r)
		out := listed{}
		for _, m := range r {
			out[m.ID] = ""
		}
		return out
	}},
}

// TestLive_StaticCatalogsCurrent (0033-MADR D4) fails when a static catalog
// goes stale: an id its provider's listing no longer offers, or one the
// listing marks deprecated, scheduled for shutdown or expiring (Anthropic's
// lifecycle, OpenAI's shutdown_date, Kilo's expiration_date). It runs with
// the live suites before each release. Listings are not billed; a provider
// whose key is unset is skipped, and Together needs LLMPROVIDER_LIVE_TOGETHER.
//
// It cannot see what no listing reports. Anthropic's lifecycle lags its
// deprecations page, and Gemini lists a model it reroutes, so catalog's
// TestStatic_NoRetiredIDs_0033, a table taken from those pages, guards them
// (0033-MADR, "Amendment (2026-10-11): sources that lag").
func TestLive_StaticCatalogsCurrent(t *testing.T) {
	for _, g := range guardListings {
		t.Run(string(g.id), func(t *testing.T) {
			offered := g.read(t)
			t.Logf("%s: %d models listed", g.id, len(offered))
			for _, id := range catalog.Static(g.id) {
				why, ok := offered[id]
				switch {
				case !ok:
					t.Errorf("%s: static id %q is not in its provider's listing", g.id, id)
				case why != "":
					t.Errorf("%s: static id %q is listed as stale: %s", g.id, id, why)
				}
			}
		})
	}
}
