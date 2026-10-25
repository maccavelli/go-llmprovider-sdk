package opencode

import (
	"context"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestEffortMax_ChatRoute_0061 (0061-MADR D3): on the chat route, whose
// reasoning_effort goes out only when the model's reasoning_options list it,
// max is sent where they list max, and otherwise the highest effort they
// list, warned.
func TestEffortMax_ChatRoute_0061(t *testing.T) {
	enableMetadata(t)
	for _, tc := range []struct {
		name, values, sent, warned string
	}{
		{"listing max", `"low","medium","high","max"`, "max", ""},
		{"listing to high", `"low","medium","high"`, "high", "high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta, _ := metadataServer(t, http.StatusOK, `{"opencode-go":{"models":{"glm-5.3-flash":{"id":"glm-5.3-flash",`+
				`"reasoning":true,"reasoning_options":[{"type":"effort","values":[`+tc.values+`]}]}}}}`)
			var body map[string]any
			srv := captureServer(t, &body, fxChat)
			metaURL := llmprovider.WithModelMetadataURL(meta.URL)
			// The document cached, as a request finds it once fetched.
			if _, err := catalog.LookupMetadataWith(context.Background(), llmprovider.ProviderOpencodeGo, metaURL,
				testCache(t)); err != nil {
				t.Fatalf("caching the document: %v", err)
			}
			p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "glm-5.3-flash", metaURL,
				WithRoute(RouteChatCompletions))...)
			req := text("hi")
			req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
			resp, err := p.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got, _ := body["reasoning_effort"].(string); got != tc.sent {
				t.Errorf("reasoning_effort = %q; want %q", got, tc.sent)
			}
			var warned []llmprovider.Warning
			for _, w := range resp.Warnings {
				if w.Kind == llmprovider.WarningEffortDegraded {
					warned = append(warned, w)
				}
			}
			switch {
			case tc.warned == "" && len(warned) != 0:
				t.Errorf("warnings %v; want none", warned)
			case tc.warned != "" && (len(warned) != 1 ||
				warned[0].Detail != "reasoning effort max is not offered for glm-5.3-flash; sent "+tc.warned):
				t.Errorf("warnings %v; want one naming %s", warned, tc.warned)
			}
		})
	}
}
