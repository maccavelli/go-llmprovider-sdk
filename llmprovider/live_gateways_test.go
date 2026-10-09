//go:build live_gateways

// Opt-in live tests against the real gateways. Excluded from the default build.
//
//	go test -tags live_gateways ./llmprovider/ -run Live -v
//
// Kilo tests REQUIRE KILO_API_KEY: since 2026-09-26 Kilo answers a placeholder
// bearer with 401, even on free models (0009 PLAN deviation). The free-model
// tests are rate-limited upstream, so rate limits, outages, quota and
// not-permitted refusals SKIP rather than fail; a 400 FAILS, since a wire
// regression answers with one (skipIfTransient; 0013 D5, MADR 0012 §1.1). These
// assert wire-format correctness, not gateway availability. The 0009 reasoning
// gate (TestLive_KiloReasoningShapes) uses a paid model and treats a 400 as
// DRIFT.
//
// OpenCode tests REQUIRE OPENCODE_API_KEY (0004-PLAN deviation D3): a bogus key is
// answered 401. Without a key, the opencode provider sends the gateway's "public" token
// (MADR 0012 §1.7). The generation tests use paid OpenCode Go models: Zen's
// free tier refuses clients other than OpenCode (403 FreeTierError, typed as
// ErrNotPermitted; measured 2026-09-26/27; MADR 0013 D1).
//
// The Hugging Face test REQUIRES HF_TOKEN: HF reports is_free:false for all
// provider offerings, so no credential-free path exists (verified 2026-08-29).
//
// The shape assertions below are the point of this suite. A test that only
// checked "a response came back" would still pass after a gateway renamed a
// field, while the code silently degraded. Each failure is a DRIFT REPORT, not
// necessarily a bug: it means a wire shape changed after the date recorded in
// the relevant wireShapesProbedOn* constant.
package llmprovider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// skipIfTransient converts upstream rate limiting, outages and account-state
// refusals into a skip: these assert wire shapes, not uptime or quota. A 400
// (ErrInvalidRequest) is a failure, since a wire regression answers with one
// (MADR 0012 revision 2, 0013 D5).
func skipIfTransient(t *testing.T, err error) {
	t.Helper()
	if liveTransient(err) {
		t.Skipf("gateway transient (rate limit / outage / quota / not permitted): %v", err)
	}
}

// liveTransient reports whether err is a class the live suite skips on.
func liveTransient(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrProviderUnavailable) ||
		errors.Is(err, ErrQuotaExhausted) || errors.Is(err, ErrNotPermitted)
}

// kiloKey returns a real Kilo credential or skips: Kilo rejects a placeholder
// bearer with 401 (0009 PLAN deviation, 2026-09-26).
func kiloKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("KILO_API_KEY")
	if key == "" {
		t.Skip("KILO_API_KEY unset: Kilo returns 401 for a placeholder key, even on free models")
	}
	return key
}

// opencodeKey returns a real OpenCode credential or skips. See 0004-PLAN deviation D3:
// OpenCode rejects a bogus bearer even for models that are free without one.
func opencodeKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("OPENCODE_API_KEY")
	if key == "" {
		t.Skip("OPENCODE_API_KEY unset: OpenCode returns 401 for a bogus key even on " +
			"free models, and the opencode provider always sends the key it is given (0004-PLAN deviation D3)")
	}
	return key
}

// The gateways' public listing bases, as catalog's listing uses them. These
// raw probes keep their own copies (0015-PLAN S8, commit 2).
const (
	liveOpencodeZenBaseURL = "https://opencode.ai/zen/v1"
	liveOpencodeGoBaseURL  = "https://opencode.ai/zen/go/v1"
	liveHuggingFaceBaseURL = "https://router.huggingface.co/v1"
)

func liveCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 90*time.Second)
}

// getJSON fetches a public gateway listing with NO Authorization header.
func getJSON(t *testing.T, url string, into any) int {
	t.Helper()
	ctx, cancel := liveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp, err := transport.DefaultClient().Do(req)
	if err != nil {
		t.Skipf("gateway unreachable: %v", err)
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode
	}
	if into != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(into); err != nil {
			t.Fatalf("decode %s: %v", url, err)
		}
	}
	return resp.StatusCode
}

// TestLive_KiloReasoningSpelling pins the dual-name decoder: Kilo emits
// message.reasoning, OpenCode emits message.reasoning_content.
func TestLive_KiloReasoningSpelling(t *testing.T) {
	ctx, cancel := liveCtx(t)
	defer cancel()
	// The request is written out, as the other raw probes here are: the Chat
	// Completions encoder is in internal/wire, which imports this package.
	body := map[string]any{
		"model":      liveModel(t, ProviderKilo, kiloFreeCollecting...),
		"messages":   []map[string]any{{"role": "user", "content": "Say ALPHA only"}},
		"max_tokens": 400,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", kiloendpoint.BaseURL+"/chat/completions", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := transport.DefaultClient().Do(req)
	if err != nil {
		t.Skipf("gateway unreachable: %v", err)
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		t.Skipf("gateway returned HTTP %d (free tier limits)", resp.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message map[string]json.RawMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded.Choices) == 0 {
		t.Skip("no choices returned")
	}
	msg := decoded.Choices[0].Message
	if _, hasReasoning := msg["reasoning"]; !hasReasoning {
		// Literal, not a constant: jsonKeyReasoningContent was dropped in 0004-PLAN
		// deviation D1 as unused, and reintroducing it for a build-tagged file
		// only would re-create the D2 `unused` problem.
		if _, hasContent := msg["reasoning_content"]; hasContent {
			t.Errorf("DRIFT (probed %s): Kilo now emits %q, not %q. The decoder handles "+
				"both, but 0004-MADR's field-name table is stale",
				wireShapesProbedOnKilo, "reasoning_content", "reasoning")
		}
		// Neither present is acceptable: not every model reasons.
	}
}

// TestLive_KiloSupportedParameters pins the metadata capability gating relies on.
func TestLive_KiloSupportedParameters(t *testing.T) {
	var listing struct {
		Data []struct {
			SupportedParameters []string `json:"supported_parameters"`
		} `json:"data"`
	}
	if code := getJSON(t, kiloendpoint.BaseURL+"/models", &listing); code != http.StatusOK {
		t.Skipf("models endpoint returned HTTP %d", code)
	}
	if len(listing.Data) == 0 {
		t.Fatalf("DRIFT (probed %s): empty catalog", wireShapesProbedOnKilo)
	}
	var withParams, withTools int
	for _, m := range listing.Data {
		if len(m.SupportedParameters) > 0 {
			withParams++
		}
		if contains(m.SupportedParameters, "tools") {
			withTools++
		}
	}
	if withParams == 0 {
		t.Errorf("DRIFT (probed %s): no model publishes supported_parameters; "+
			"capability gating has nothing to gate on", wireShapesProbedOnKilo)
	}
	if withTools == 0 {
		t.Errorf("DRIFT (probed %s): no model lists %q in supported_parameters",
			wireShapesProbedOnKilo, "tools")
	}
}

func contains(hay []string, needle string) bool {
	return slices.Contains(hay, needle)
}

// TestLive_HuggingFaceMetadataFields pins the four fields metadata-driven
// curation depends on. Deliberately not universal: re-probed 2026-08-29, only
// 253 of 317 offerings carry all four, so asserting every offering would be
// flaky by construction.
func TestLive_HuggingFaceMetadataFields(t *testing.T) {
	var listing struct {
		Data []struct {
			ID           string `json:"id"`
			Architecture struct {
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Providers []map[string]json.RawMessage `json:"providers"`
		} `json:"data"`
	}
	if code := getJSON(t, liveHuggingFaceBaseURL+"/models", &listing); code != http.StatusOK {
		t.Skipf("models endpoint returned HTTP %d", code)
	}
	if len(listing.Data) == 0 {
		t.Fatalf("DRIFT (probed %s): empty catalog", wireShapesProbedOnHuggingFace)
	}
	for _, m := range listing.Data {
		if len(m.Architecture.OutputModalities) == 0 {
			t.Errorf("DRIFT (probed %s): %q has no architecture.output_modalities; "+
				"modality filtering would silently pass everything",
				wireShapesProbedOnHuggingFace, m.ID)
			break
		}
	}
	var ok int
	for _, m := range listing.Data {
		for _, pr := range m.Providers {
			_, a := pr["throughput"]
			_, b := pr["first_token_latency_ms"]
			_, c := pr["supports_tools"]
			if a && b && c {
				ok++
			}
		}
	}
	if ok == 0 {
		t.Errorf("DRIFT (probed %s): no offering publishes throughput + "+
			"first_token_latency_ms + supports_tools; metadata ranking is dead",
			wireShapesProbedOnHuggingFace)
	}
}

// TestLive_ListingsNeedNoCredential pins that all four catalogs are public.
// Discovery works before a key is configured, which the wizards rely on.
func TestLive_ListingsNeedNoCredential(t *testing.T) {
	for name, url := range map[ProviderID]string{
		ProviderOpencodeZen: liveOpencodeZenBaseURL + "/models",
		ProviderOpencodeGo:  liveOpencodeGoBaseURL + "/models",
		ProviderHuggingFace: liveHuggingFaceBaseURL + "/models",
		ProviderKilo:        kiloendpoint.BaseURL + "/models",
	} {
		t.Run(string(name), func(t *testing.T) {
			if code := getJSON(t, url, nil); code != http.StatusOK {
				t.Errorf("DRIFT: %s returned HTTP %d with no credential; discovery "+
					"before key configuration would break", url, code)
			}
		})
	}
}

// TestLive_OpencodeKeyHeaderPerRoute pins MADR 0007 §1c against the live Zen
// server with a bogus key, which spends nothing: a route that does not read the
// header answers "Missing API key.", and the route's own header reaches key
// validation ("Invalid API key."). Measured 2026-09-26.
func TestLive_OpencodeKeyHeaderPerRoute(t *testing.T) {
	const bogus = "sk-bogus-000"
	routes := []struct{ name, path, body, right string }{
		{"messages", "/messages", `{"model":"claude-haiku-4-5","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`, "x-api-key"},
		{"google", "/models/gemini-3.7-flash:generateContent", `{"contents":[{"parts":[{"text":"hi"}]}]}`, "x-goog-api-key"},
	}
	for _, rt := range routes {
		t.Run(rt.name, func(t *testing.T) {
			for _, tc := range []struct{ header, value, want string }{
				{"Authorization", "Bearer " + bogus, "Missing API key."},
				{rt.right, bogus, "Invalid API key."},
			} {
				status, body := postLive(t, liveOpencodeZenBaseURL+rt.path, rt.body, tc.header, tc.value)
				if status != http.StatusUnauthorized {
					t.Skipf("%s via %s returned HTTP %d, not 401", rt.name, tc.header, status)
				}
				if !strings.Contains(body, tc.want) {
					t.Errorf("DRIFT: %s with the key in %s: want %q, got %s", rt.name, tc.header, tc.want, body)
				}
			}
		})
	}
}

// postLive sends a JSON POST with one key header and returns the status and a
// bounded body. A transport failure skips, like the suite's other requests.
func postLive(t *testing.T, url, body, header, value string) (int, string) {
	t.Helper()
	ctx, cancel := liveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(header, value)
	resp, err := transport.DefaultClient().Do(req)
	if err != nil {
		t.Skipf("gateway unreachable: %v", err)
	}
	defer closeResponseBody(resp)
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(raw)
}
