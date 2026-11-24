//go:build live_gateways

package llmprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/kiloendpoint"
)

// liveOmittedModels is the pinned model (0089-PLAN, its entry "DL65's
// measurement test written", 2026-11-24): the smallest-limit Kilo model whose
// long stream can finish. anthropic/claude-haiku-4.5 lists 64,000 (as
// anthropic/claude-sonnet-4.5 does). anthropic/claude-opus-4.1 lists less
// (32,000) and is not used: its long stream died on Kilo's gateway timeout, an
// in-stream 503, before any finish. The next model if haiku's run is
// undecided or errors is z-ai/glm-5.3-flashx (131,072), named in
// LLMPROVIDER_LIVE_0089_MODELS.
var liveOmittedModels = []string{"anthropic/claude-haiku-4.5"}

// liveOmittedPerModel is each model's deadline; the test needs it for every
// model, plus a minute, inside go test's own deadline.
const liveOmittedPerModel = 20 * time.Minute

// liveOmittedStated reads the model's stated output limit, the listing's
// top_provider.max_completion_tokens, from Kilo's model listing.
func liveOmittedStated(ctx context.Context, key, model string) (int, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kiloendpoint.BaseURL+"/models", http.NoBody)
	if err != nil {
		return 0, false
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer closeResponseBody(resp)
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	var listing struct {
		Data []struct {
			ID          string `json:"id"`
			TopProvider struct {
				Max int `json:"max_completion_tokens"`
			} `json:"top_provider"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&listing) != nil {
		return 0, false
	}
	for _, m := range listing.Data {
		if m.ID == model && m.TopProvider.Max > 0 {
			return m.TopProvider.Max, true
		}
	}
	return 0, false
}

// TestLive_OmittedLimit_0089 is 0089-PLAN's "OQ1 (1)'s measurement", and
// decides 0089-MADR DL65: does Kilo hand out less than a model's stated output
// limit when NO limit is sent? It runs only with LLMPROVIDER_LIVE_0089 set and
// KILO_API_KEY in the environment (read by name, never printed). Per model
// (the pinned list, or LLMPROVIDER_LIVE_0089_MODELS, a comma list) it streams
// one request that asks for more output than the stated limit and sends no
// limit field, with a 20-minute deadline each (liveOmittedPerModel). The test
// fails at once, with a clear text, when go test's own deadline (-timeout)
// leaves less than that for every model in the list, so run it with a
// -timeout of 20 minutes per model plus 10: 30m for the pinned model.
//
// Only a "length" finish decides: output tokens below the stated limit mean
// Kilo keeps sending the discovered limit; equal to or above it mean Kilo
// omits the field as the other optional wires do. The run cannot tell whether
// Kilo or the upstream model fills less, or whether the model's real cap is
// below its listing: it has no control run with the stated limit sent. A
// "stop", or a "length" without a token count, is logged as "undecided" and
// does not fail. A non-200 status, an in-stream error or a transport failure
// fails, with a fixed text, and stops the run.
//
// It logs and writes (to ../tmp/live-0089/results.txt, git-ignored) only the
// allowlisted line: model, status, finish reason, output and reasoning token
// counts, the stated limit, the verdict word. Billed: at most the stated limit
// in output tokens, per model.
func TestLive_OmittedLimit_0089(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0089") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0089 to run 0089-PLAN's DL65 measurement (it bills up to the stated limit in output tokens)")
	}
	key := kiloKey(t)
	models := liveOmittedModels
	if env := strings.TrimSpace(os.Getenv("LLMPROVIDER_LIVE_0089_MODELS")); env != "" {
		models = nil
		for _, m := range strings.Split(env, ",") {
			if m = strings.TrimSpace(m); m != "" {
				models = append(models, m)
			}
		}
	}
	if dl, ok := t.Deadline(); ok {
		need := time.Duration(len(models))*liveOmittedPerModel + time.Minute
		if left := time.Until(dl); left < need {
			t.Fatalf("go test's deadline leaves %s, and %d model(s) need %s (%s each): run with -timeout %s or fewer models",
				left.Round(time.Second), len(models), need, liveOmittedPerModel,
				time.Duration(len(models))*liveOmittedPerModel+10*time.Minute)
		}
	}
	out := filepath.Join("..", "tmp", "live-0089")
	if err := os.MkdirAll(out, 0o750); err != nil {
		t.Fatal("the output directory could not be made")
	}
	for _, model := range models {
		ctx, cancel := context.WithTimeout(context.Background(), liveOmittedPerModel)
		stated, ok := liveOmittedStated(ctx, key, model)
		if !ok {
			cancel()
			t.Fatalf("%s: no stated output limit in Kilo's listing, or the listing could not be read", model)
		}
		res, err := streamOmitted(ctx, http.DefaultClient, kiloendpoint.BaseURL, key, model)
		cancel()
		if err != nil {
			t.Fatalf("%s: %v", model, err)
		}
		verdict := omittedVerdict(res.Finish, res.Out, stated)
		line := omittedLine(model, stated, res, verdict)
		t.Log(line)
		f, ferr := os.OpenFile(filepath.Join(out, "results.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if ferr != nil {
			t.Fatal("the results file could not be opened")
		}
		_, werr := f.WriteString(time.Now().UTC().Format("2006-01-02") + " " + line + "\n")
		if cerr := f.Close(); werr != nil || cerr != nil {
			t.Fatal("the results file could not be written")
		}
		if res.Status != http.StatusOK {
			t.Fatalf("%s: HTTP status %d, want 200", model, res.Status)
		}
		if res.StreamErrType != "" {
			t.Fatalf("%s: the stream carried an error of type %q, code %d", model, res.StreamErrType, res.StreamErrCode)
		}
		switch verdict {
		case "below":
			t.Logf("%s: DECIDED, a length finish at %d output tokens, below the stated %d: Kilo keeps sending the discovered limit", model, res.Out, stated)
		case "equal", "above":
			t.Logf("%s: DECIDED, a length finish at %d output tokens, at or above the stated %d: Kilo omits the field as the other optional wires do", model, res.Out, stated)
		default:
			t.Logf("%s: %s", model, omittedUndecided(res.Finish, res.Out, stated))
		}
	}
}
