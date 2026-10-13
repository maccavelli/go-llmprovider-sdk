//go:build live_gateways

package llmprovider_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0039-PLAN Phase 4's measurement. TestLive_0039PromptCache runs only with
// LLMPROVIDER_LIVE_0039 set, and each path only with its key; Together also
// needs LLMPROVIDER_LIVE_TOGETHER, as its other live tests do. Each path sends
// the same request twice, with Request.Cache set, and logs each reply's
// decoded cache counts. It logs no prompt, header or key. Billed: two
// requests per path, eight paths, about 3,000 input and at most 16 output
// tokens each.

// prompt0039 is one path's fixed system prompt: about 3,000 tokens of
// numbered plain text, above every caching minimum 0039-MADR lists. It begins
// with the path's own line, so that no path reads another's cache entry
// (0039-PLAN Phase 4, deviation 1).
func prompt0039(path string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Measurement path: %s.\n", path)
	b.WriteString("You are a terse assistant. The reference list below is context only.\n")
	for i := 1; i <= 150; i++ {
		fmt.Fprintf(&b, "Entry %03d: the catalogue record for item %d lists alpha, bravo, charlie, delta, echo, foxtrot and golf.\n", i, i)
	}
	return b.String()
}

// path0039 is one row of 0039-PLAN Phase 4's table.
type path0039 struct {
	name      string
	id        llmprovider.ProviderID
	key       func(t *testing.T) string
	model     string
	retention llmprovider.CacheRetention
	// readRequired says the second request must read what the first wrote;
	// hourRequired says the first must report a one-hour write.
	readRequired, hourRequired bool
}

// problems lists what a path's two usages break: a required read missing
// from the second, or a required one-hour write missing from the first.
func (pt path0039) problems(usages [2]llmprovider.Usage) []string {
	var out []string
	if pt.readRequired && usages[1].CachedTokens == 0 {
		out = append(out, fmt.Sprintf("%s: the second request read nothing from the cache: %+v", pt.name, usages[1]))
	}
	if pt.hourRequired && usages[0].CacheWrite1hTokens == 0 {
		out = append(out, fmt.Sprintf("%s: the first long request reported no one-hour write: %+v", pt.name, usages[0]))
	}
	return out
}

// TestPromptCacheChecks_0039 proves the measurement's checks on canned
// usages, with no key and no network.
func TestPromptCacheChecks_0039(t *testing.T) {
	wrote := llmprovider.Usage{InputTokens: 7097, CacheWriteTokens: 7093, CacheWrite1hTokens: 7093}
	read := llmprovider.Usage{InputTokens: 7097, CachedTokens: 7093}
	long := path0039{name: "long", readRequired: true, hourRequired: true}
	for _, c := range []struct {
		name   string
		path   path0039
		usages [2]llmprovider.Usage
		want   string
	}{
		{"a written hour, then a read", long, [2]llmprovider.Usage{wrote, read}, ""},
		{"no read", long, [2]llmprovider.Usage{wrote, wrote}, "read nothing"},
		{"a read where the hour was due", long, [2]llmprovider.Usage{read, read}, "no one-hour write"},
		{"nothing required", path0039{name: "grok"}, [2]llmprovider.Usage{}, ""},
	} {
		got := strings.Join(c.path.problems(c.usages), "; ")
		switch {
		case c.want == "" && got != "":
			t.Errorf("%s: problems %q, want none", c.name, got)
		case c.want != "" && !strings.Contains(got, c.want):
			t.Errorf("%s: problems %q, want one containing %q", c.name, got, c.want)
		}
	}
}

func envKey0039(id llmprovider.ProviderID) func(t *testing.T) string {
	return func(t *testing.T) string { return llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id]) }
}

// TestLive_0039PromptCache is 0039-PLAN Phase 4's measurement.
func TestLive_0039PromptCache(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0039") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0039 to run 0039-PLAN Phase 4's measurement")
	}
	short, long := llmprovider.CacheShort, llmprovider.CacheLong
	claude, zen, kilo := llmprovider.ProviderClaude, llmprovider.ProviderOpencodeZen, llmprovider.ProviderKilo
	paths := []path0039{
		{"claude short", claude, envKey0039(claude), "claude-haiku-5-5", short, true, false},
		{"claude long", claude, envKey0039(claude), "claude-haiku-5-5", long, true, true},
		{"zen claude short", zen, llmprovider.LiveOpencodeKey, "claude-haiku-5-5", short, true, false},
		{"kilo claude short", kilo, llmprovider.LiveKiloKey, "anthropic/claude-haiku-5.5", short, true, false},
		{"kilo claude long", kilo, llmprovider.LiveKiloKey, "anthropic/claude-haiku-5.5", long, true, false},
		{"openai short", llmprovider.ProviderOpenAI, envKey0039(llmprovider.ProviderOpenAI), "gpt-6-luna", short, true, false},
		{"grok short", llmprovider.ProviderGrok, envKey0039(llmprovider.ProviderGrok), "grok-4.6", short, false, false},
		{"together short", llmprovider.ProviderTogether, llmprovider.LiveTogetherKey,
			catalog.Static(llmprovider.ProviderTogether)[0], short, false, false},
	}
	for _, pt := range paths {
		t.Run(pt.name, func(t *testing.T) {
			prompt := prompt0039(pt.name)
			key := pt.key(t)
			rec := &postRecorder{}
			p, err := providers.New(pt.id, llmprovider.WithAPIKey(key), llmprovider.WithModel(pt.model),
				llmprovider.WithModelProbes(false), llmprovider.WithMaxTokens(16),
				llmprovider.WithSessionID("live-0039-"+strings.ReplaceAll(pt.name, " ", "-")),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
			if err != nil {
				t.Fatal(err)
			}
			var usages [2]llmprovider.Usage
			for n := range 2 {
				if n == 1 {
					time.Sleep(3 * time.Second)
				}
				req := &llmprovider.Request{Instructions: prompt, Cache: pt.retention,
					Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Reply with the word ok."}}}
				ctx, cancel := llmprovider.LiveCtx(t)
				resp, genErr := p.Generate(ctx, req)
				cancel()
				if genErr != nil || resp == nil {
					llmprovider.SkipIfTransient(t, genErr)
					path, body, _ := rec.last()
					t.Fatalf("%s request %d (%s): %v (reply at %s: %.300s)", pt.name, n+1, pt.model, genErr, path, body)
				}
				usages[n] = resp.Usage
				t.Logf("%s request %d: model %s, retention %s, usage %+v", pt.name, n+1, pt.model, pt.retention, resp.Usage)
			}
			for _, problem := range pt.problems(usages) {
				t.Error(problem)
			}
		})
	}
}
