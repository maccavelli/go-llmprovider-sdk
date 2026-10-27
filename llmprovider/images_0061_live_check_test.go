package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// imagesLiveURL is the URL cases' image: a public JPEG on Google Cloud
// Storage, which answered HEAD with 200, image/jpeg and 394,671 bytes on
// 2026-10-26 (0061-PLAN Phase 7). The test holds its URL, never the image.
const imagesLiveURL = "https://storage.googleapis.com/generativeai-downloads/images/scones.jpg"

// imagesLivePrompt is every case's text.
const imagesLivePrompt = "Name the main colour of the image in one word."

// imagesLivePNG is a 24×24 grey PNG, built from bytes: xAI refuses an image
// under 8 pixels a side or 512 pixels in all, so the live cases do not send
// the offline tests' 1×1 (0061-PLAN deviation 20, as amended).
var imagesLivePNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x18, 0x00, 0x00, 0x00, 0x18,
	0x08, 0x00, 0x00, 0x00, 0x00, 0xc5, 0x1c, 0x62, 0x24, 0x00, 0x00, 0x00,
	0x13, 0x49, 0x44, 0x41, 0x54, 0x78, 0xda, 0x63, 0x68, 0xc0, 0x01, 0x18,
	0x46, 0x25, 0x46, 0x25, 0xc8, 0x97, 0x00, 0x00, 0x96, 0x17, 0x20, 0x10,
	0xb7, 0x58, 0x12, 0x31, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
	0xae, 0x42, 0x60, 0x82,
}

// The kinds of case: an image by its bytes, an image by URL, and a text
// request at EffortMax.
const (
	imageByData = "data"
	imageByURL  = "url"
	atMaxEffort = "max"
)

// imagesLive is one of 0061-PLAN Phase 7's live cases.
type imagesLive struct {
	id       string
	provider llmprovider.ProviderID
	route    opencode.Route
	model    string
	kind     string
	// dropped says the image by URL is dropped, with a WarningImageDropped
	// (0061-MADR D5: Ollama, OpenCode's google route).
	dropped bool
}

// image is the case's image.
func (c imagesLive) image() llmprovider.ImageItem {
	if c.kind == imageByURL {
		return llmprovider.ImageItem{Role: llmprovider.RoleUser, URL: imagesLiveURL}
	}
	return llmprovider.ImageItem{Role: llmprovider.RoleUser, MediaType: "image/png", Data: imagesLivePNG}
}

// request is the case's request: the prompt and the image, or the prompt at
// EffortMax.
func (c imagesLive) request() *llmprovider.Request {
	req := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: imagesLivePrompt}}}
	if c.kind == atMaxEffort {
		req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
	} else {
		req.Input = append(req.Input, c.image())
	}
	return req
}

// imagesLiveCases is a data, a URL and an EffortMax case for each provider
// and OpenCode route with a key variable, each on a model whose listing
// facts state image input (listed live 2026-10-26).
func imagesLiveCases() []imagesLive {
	zen, goGateway := llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo
	targets := []imagesLive{
		{id: "claude", provider: llmprovider.ProviderClaude, model: "claude-haiku-5-5"},
		{id: "openai", provider: llmprovider.ProviderOpenAI, model: "gpt-5-nano"},
		{id: "grok", provider: llmprovider.ProviderGrok, model: "grok-4.3"},
		{id: "gemini", provider: llmprovider.ProviderGemini, model: "gemini-3.1-flash-lite"},
		{id: "huggingface", provider: llmprovider.ProviderHuggingFace, model: "MiniMaxAI/MiniMax-M3.1-Flash"},
		{id: "together", provider: llmprovider.ProviderTogether, model: "nvidia/NVIDIA-Nemotron-3.5-Super-VL"},
		{id: "kilo", provider: llmprovider.ProviderKilo, model: "google/gemini-3.6-flash"},
		{id: "opencode-zen-messages", provider: zen, route: opencode.RouteMessages, model: "claude-haiku-5-5"},
		{id: "opencode-zen-responses", provider: zen, route: opencode.RouteResponses, model: "gpt-6-luna"},
		{id: "opencode-zen-chat", provider: zen, route: opencode.RouteChatCompletions, model: "glm-5.3-flash"},
		{id: "opencode-zen-google", provider: zen, route: opencode.RouteGoogle, model: "gemini-3.5-flash-lite", dropped: true},
		{id: "opencode-go-messages", provider: goGateway, route: opencode.RouteMessages, model: "claude-haiku-5-5"},
		{id: "opencode-go-responses", provider: goGateway, route: opencode.RouteResponses, model: "gpt-6-luna"},
		{id: "opencode-go-chat", provider: goGateway, route: opencode.RouteChatCompletions, model: "glm-5.3-flash"},
	}
	var out []imagesLive
	for _, t := range targets {
		for _, kind := range []string{imageByData, imageByURL, atMaxEffort} {
			c := t
			c.id, c.kind = t.id+"-"+kind, kind
			out = append(out, c)
		}
	}
	return out
}

// ollamaImagesLive is Ollama's cases, on the model the owner names: the URL
// case's image is dropped, warned (0061-MADR D5, E9).
func ollamaImagesLive(model string) []imagesLive {
	var out []imagesLive
	for _, kind := range []string{imageByData, imageByURL, atMaxEffort} {
		out = append(out, imagesLive{id: "ollama-" + kind, provider: llmprovider.ProviderOllama, model: model, kind: kind,
			dropped: kind == imageByURL})
	}
	return out
}

// chatGPTImagesLive is the ChatGPT session's cases, its images sent by the
// provider (0061-MADR, amendment "Q11's live result"; 0061-PLAN deviation
// 21).
func chatGPTImagesLive() []imagesLive {
	var out []imagesLive
	for _, kind := range []string{imageByData, imageByURL} {
		out = append(out, imagesLive{id: "chatgpt-" + kind, provider: llmprovider.ProviderOpenAI, model: "gpt-6-astra",
			kind: kind})
	}
	return out
}

// imagesLiveResult is what a case records: the sends and the warnings, never
// the reply's text.
type imagesLiveResult struct {
	Sends    int
	Warnings []llmprovider.Warning
}

// kinds is the result's warnings' kinds, for a log line.
func (r imagesLiveResult) kinds() string {
	if len(r.Warnings) == 0 {
		return "none"
	}
	out := make([]string, len(r.Warnings))
	for i, w := range r.Warnings {
		out[i] = string(w.Kind)
	}
	return strings.Join(out, ", ")
}

// sentBodies records the bodies of the POSTs it sends on base.
type sentBodies struct {
	base   http.RoundTripper
	mu     sync.Mutex
	bodies [][]byte
}

func (b *sentBodies) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(raw))
		b.mu.Lock()
		b.bodies = append(b.bodies, raw)
		b.mu.Unlock()
	}
	return b.base.RoundTrip(req)
}

// sent is the recorded bodies.
func (b *sentBodies) sent() [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([][]byte(nil), b.bodies...)
}

// checkImagesLive sends c's request once through a provider built with opts,
// with no metadata, no probes and a cache of its own, on base, and reports
// the first check that fails: the request
// succeeds; an image case is one send whose body carries the image, the
// PNG's bytes or the URL, or, dropped, does not carry the URL and has one
// WarningImageDropped. It returns the sends and the warnings.
func checkImagesLive(ctx context.Context, c imagesLive, base http.RoundTripper,
	opts ...llmprovider.Option) (imagesLiveResult, error) {
	rec := &sentBodies{base: base}
	all := []llmprovider.Option{llmprovider.WithModel(c.model), llmprovider.WithModelProbes(false),
		llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec}),
		catalog.WithCache(new(catalog.Cache))}
	if c.route != "" {
		all = append(all, opencode.WithRoute(c.route))
	}
	p, err := providers.New(c.provider, append(all, opts...)...)
	if err != nil {
		return imagesLiveResult{}, fmt.Errorf("%s: %w", c.id, err)
	}
	resp, err := p.Generate(ctx, c.request())
	sent := rec.sent()
	r := imagesLiveResult{Sends: len(sent)}
	if err != nil {
		return r, fmt.Errorf("%s: %d send(s): %w", c.id, len(sent), err)
	}
	r.Warnings = resp.Warnings
	if c.kind == atMaxEffort {
		return r, nil
	}
	if len(sent) != 1 {
		return r, fmt.Errorf("%s: %d sends; want 1", c.id, len(sent))
	}
	body := string(sent[0])
	dropped := 0
	for _, w := range resp.Warnings {
		if w.Kind == llmprovider.WarningImageDropped {
			dropped++
		}
	}
	switch {
	case c.kind == imageByData && !strings.Contains(body, base64.StdEncoding.EncodeToString(imagesLivePNG)):
		return r, fmt.Errorf("%s: the body sent does not carry the PNG's bytes", c.id)
	case c.kind == imageByURL && c.dropped && (strings.Contains(body, imagesLiveURL) || dropped != 1):
		return r, fmt.Errorf("%s: the image by URL was not dropped with one warning (%d warning(s))", c.id, dropped)
	case c.kind == imageByURL && !c.dropped && !strings.Contains(body, imagesLiveURL):
		return r, fmt.Errorf("%s: the body sent does not carry the image's URL", c.id)
	}
	return r, nil
}

// TestImagesLive_Twin_0061 (0061-PLAN Phase 7; deviation 18): checkImagesLive,
// the live cases' checks, against a local server answering each wire's canned
// reply: it passes for an image by data and by URL on each wire, the drops
// included, and the EffortMax case; it fails where a drop is wanted and not
// made, where none is wanted and one is made, and where the service refuses.
func TestImagesLive_Twin_0061(t *testing.T) {
	ctx := context.Background()
	cases := map[string]imagesLive{}
	for _, c := range append(imagesLiveCases(), ollamaImagesLive("gemma3:12b")...) {
		cases[c.id] = c
	}
	for _, id := range []string{"claude-data", "claude-url", "claude-max", "openai-url", "grok-data", "gemini-url",
		"huggingface-data", "together-url", "kilo-data", "opencode-zen-messages-url", "opencode-zen-responses-data",
		"opencode-zen-chat-url", "opencode-zen-google-data", "opencode-zen-google-url", "opencode-go-chat-data",
		"ollama-data", "ollama-url", "ollama-max"} {
		c, ok := cases[id]
		if !ok {
			t.Fatalf("no case %q", id)
		}
		srv := twinServer(t)
		r, err := checkImagesLive(ctx, c, http.DefaultTransport, llmprovider.WithAPIKey("k"), llmprovider.WithBaseURL(srv.URL))
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		if c.kind != atMaxEffort && r.Sends != 1 {
			t.Errorf("%s: %d sends; want 1", id, r.Sends)
		}
	}
	wrong := cases["claude-url"]
	wrong.dropped = true
	if _, err := checkImagesLive(ctx, wrong, http.DefaultTransport, llmprovider.WithAPIKey("k"),
		llmprovider.WithBaseURL(twinServer(t).URL)); err == nil || !strings.Contains(err.Error(), "was not dropped") {
		t.Errorf("a drop wanted and not made: %v; want the check to fail", err)
	}
	wrong = cases["ollama-url"]
	wrong.dropped = false
	if _, err := checkImagesLive(ctx, wrong, http.DefaultTransport,
		llmprovider.WithBaseURL(twinServer(t).URL)); err == nil {
		t.Error("a drop made where none is wanted: nil; want the check to fail")
	}
	refusing := twinServer(t, `{"type":"error","error":{"type":"invalid_request_error","message":"Could not process image"}}`)
	if _, err := checkImagesLive(ctx, cases["claude-data"], http.DefaultTransport, llmprovider.WithAPIKey("k"),
		llmprovider.WithBaseURL(refusing.URL)); err == nil {
		t.Error("a refused image: nil; want the check to fail")
	}
}

// TestImagesLivePNG_0061 (0061-PLAN deviation 20): the live cases' image is a
// PNG of 24×24, 576 pixels, the least square xAI takes.
func TestImagesLivePNG_0061(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(imagesLivePNG))
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds(); got != image.Rect(0, 0, 24, 24) {
		t.Errorf("imagesLivePNG's bounds = %v; want 24×24", got)
	}
}
