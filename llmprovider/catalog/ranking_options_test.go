package catalog_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestListModels_HonoursRankingOptions is llmprovider's
// TestDiscoverModels_HonoursRankingOptions (MADR 0013 A4), with its profile
// halves back (0015-PLAN S8, commit 2): each open-catalog provider's ListModels
// returns exactly what the catalog recommends for the same profile and
// metadata URL, and the fixture separates the profiles. The environment's
// URL is never read: internal/ambientcheck holds that (0015-PLAN S10). Its
// fixtures and clock are the original's.
func TestListModels_HonoursRankingOptions(t *testing.T) {
	catalog.PinRankingNow(t, catalog.RefNow)
	catalog.EnableModelMetadata(t)

	kiloListing := `{"data":[` + strings.Join([]string{
		catalog.KiloRankEntry("a/flash-lite", "A Flash Lite", "0.0000001", "0.0000004", 20, true, ""),
		catalog.KiloRankEntry("b/flash", "B Flash", "0.0000003", "0.0000012", 10, true, `"terminalBench":{"overallScore":0.75}`),
		catalog.KiloRankEntry("c/pro", "C Pro", "0.000005", "0.000025", 40, true, `"terminalBench":{"overallScore":0.80}`),
		catalog.KiloRankEntry("d/mid", "D Mid", "0.000001", "0.000004", 60, true, `"preferredIndex":2`),
		catalog.KiloRankEntry("e/mini", "E Mini", "0.0000002", "0.0000008", 90, true, ""),
		catalog.KiloRankEntry("f/large", "F Large", "0.000002", "0.000008", 30, true, ""),
		catalog.KiloRankEntry("kilo-auto/efficient", "Auto Efficient", "-1", "-1", -1, true, `"preferredIndex":0`),
	}, ",") + `]}`
	zenIDs := []string{"glm-5.3-flash", "qwen3.8-flash", "kimi-k2.6", "gpt-6-luna", "hy3", "mimo-v2.6-flash", "deepseek-v4-pro"}
	zenMeta := `{` + catalog.MDSection("opencode",
		catalog.MDModel("glm-5.3-flash", "glm-flash", true, 0.15, 0.5, "2026-08-26"),
		catalog.MDModel("qwen3.8-flash", "qwen-flash", true, 0.2, 0.8, "2026-08-20"),
		catalog.MDModel("kimi-k2.6", "kimi", true, 0.6, 2.5, "2026-07-01"),
		catalog.MDModel("gpt-6-luna", "gpt-luna", true, 0.1, 0.5, "2026-09-22"),
		catalog.MDModel("hy3", "hy", true, 0.3, 1.2, "2026-08-01"),
		catalog.MDModel("mimo-v2.6-flash", "mimo", true, 0.14, 0.28, "2026-09-22"),
		catalog.MDModel("deepseek-v4-pro", "deepseek-pro", true, 2, 8, "2026-09-10"),
	) + `}`

	for _, tc := range []struct {
		provider      llmprovider.ProviderID
		model         string
		listing, meta string
	}{
		{llmprovider.ProviderKilo, "m", kiloListing, ""},
		{llmprovider.ProviderHuggingFace, "m", catalog.HFRankListing, catalog.HFRankMetadata},
		{llmprovider.ProviderOpencodeZen, "glm-5.3-flash", catalog.ZenStyleListing(zenIDs...), zenMeta},
	} {
		t.Run(string(tc.provider), func(t *testing.T) {
			listing := catalog.ServeBody(t, tc.listing)
			opts := []llmprovider.Option{llmprovider.WithBaseURL(listing.URL)}
			if tc.meta != "" {
				meta, _ := catalog.MetadataServer(t, http.StatusOK, tc.meta)
				opts = append(opts, llmprovider.WithModelMetadataURL(meta.URL))
			}
			key := llmprovider.NewStaticToken("k")
			utility, err := catalog.List(context.Background(), tc.provider, key, opts...)
			if err != nil {
				t.Fatalf("catalog.List: %v", err)
			}
			opts = append(opts, catalog.WithProfile(catalog.ProfileCapable))
			want, err := catalog.List(context.Background(), tc.provider, key, opts...)
			if err != nil {
				t.Fatalf("catalog.List: %v", err)
			}
			if slices.Equal(want.Recommended, utility.Recommended) {
				t.Fatalf("fixture does not separate the profiles: both %v", want.Recommended)
			}
			catalog.ResetModelMetadataCache()
			p, err := providers.New(tc.provider, append([]llmprovider.Option{llmprovider.WithAPIKey("k"),
				llmprovider.WithModel(tc.model)}, opts...)...)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			got, err := p.(llmprovider.ModelLister).ListModels(context.Background())
			if err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			if !slices.Equal(got, want.Recommended) {
				t.Errorf("ListModels = %v, want the catalog's %v", got, want.Recommended)
			}
		})
	}
}

// TestWithProfile_EveryProviderTakesIt: a shared option list holding
// catalog.WithProfile is valid for every built-in provider's New, and
// catalog.WithKiloOrganization only for kilo's.
func TestWithProfile_EveryProviderTakesIt(t *testing.T) {
	registry := providers.Default()
	for _, d := range registry.Descriptors() {
		opts := []llmprovider.Option{llmprovider.WithModel("m"), catalog.WithProfile(catalog.ProfileCapable)}
		if d.RequiresAPIKey {
			opts = append(opts, llmprovider.WithAPIKey("k"))
		}
		if _, err := registry.New(d.ID, opts...); err != nil {
			t.Errorf("%s: New with catalog.WithProfile: %v", d.ID, err)
		}
		_, err := registry.New(d.ID, append(opts, catalog.WithKiloOrganization("org"))...)
		if (err == nil) != (d.ID == llmprovider.ProviderKilo) {
			t.Errorf("%s: New with catalog.WithKiloOrganization: err = %v", d.ID, err)
		}
	}
}
