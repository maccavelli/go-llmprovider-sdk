package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0033-PLAN-refresh-the-curated-catalogs.md Phase 8: the snapshot the
// maintainer command recorded (catalogs -snapshot) is replayed offline, and
// each ranked static list equals the latest snapshot's utility six
// (0033-MADR's amendment "the second refresh, and what the command reads",
// D4c as refined).

// snapshotFile names each ranked list's listing in a recorded snapshot, as
// internal/cmd/catalogs's snapshotFiles does; the five are its rankedLists.
var snapshotFile = map[llmprovider.ProviderID]string{
	llmprovider.ProviderOpencodeZen: "zen.json", llmprovider.ProviderOpencodeGo: "go.json",
	llmprovider.ProviderKilo: "kilo.json", llmprovider.ProviderHuggingFace: "hf.json",
	llmprovider.ProviderTogether: "together.json",
}

// latestSnapshot is the newest recorded snapshot directory: the last
// testdata/ranking-YYYY-MM-DD in name order that holds a snapshot.json.
func latestSnapshot(t *testing.T) string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join("testdata", "ranking-*", "snapshot.json"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no recorded snapshot under testdata (err %v)", err)
	}
	slices.Sort(dirs)
	return filepath.Dir(dirs[len(dirs)-1])
}

// replaySnapshot lists provider from the snapshot in dir, with the ranking
// clock pinned to the snapshot's recorded_at.
func replaySnapshot(t *testing.T, dir string, provider llmprovider.ProviderID, profile Profile) Catalog {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RecordedAt string `json:"recorded_at"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339, manifest.RecordedAt)
	if err != nil {
		t.Fatalf("recorded_at %q: %v", manifest.RecordedAt, err)
	}
	pinRankingNow(t, at)
	enableModelMetadata(t)
	serve := func(name string) string {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return serveBody(t, string(body)).URL
	}
	return listCatalog(context.Background(), t, provider, llmprovider.WithBaseURL(serve(snapshotFile[provider])),
		llmprovider.WithModelMetadataURL(serve("metadata.json")), WithProfile(profile))
}

// TestListModelCatalog_Snapshot20261012 replays the 2026-10-12 snapshot and
// requires each ranked list's sixes exactly. A rule change must update this
// test and the records together, as for the 2026-09-26 snapshot.
func TestListModelCatalog_Snapshot20261012(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for _, tc := range []struct {
		provider         llmprovider.ProviderID
		usable           int
		utility, capable []string
	}{
		{llmprovider.ProviderOpencodeZen, 79,
			[]string{"claude-haiku-5-5", opencodeDeepSeekV41Flash, "qwen3.8-flash", "glm-5.3-flash", opencodeDeepSeekV4Flash,
				"gemini-3.5-flash-lite"},
			[]string{"mistral-large-4", "gpt-6.1-sol", "claude-sonnet-5-5", "claude-opus-5-5", "gpt-6-sol", "grok-4.7"}},
		{llmprovider.ProviderOpencodeGo, 34,
			[]string{"claude-haiku-5-5", "mimo-v2.6-flash", "qwen3.8-flash", "glm-5.3-flash", "gpt-6-luna", "space-bunny"},
			[]string{"space-bunny", "mimo-v2.6-pro", "gpt-6-luna", "grok-4.7", "glm-5.3", "grok-4.6"}},
		{llmprovider.ProviderKilo, 285,
			[]string{"deepseek/deepseek-v4.1-flash", "z-ai/glm-5.3-flash", "google/gemini-3.8-flash", "google/gemini-3.6-flash",
				"meta/muse-spark-1.2", "openai/gpt-6.1-sol"},
			[]string{"openai/gpt-6-astra", "anthropic/claude-fable-5.1", "openai/gpt-5.6-sol", "deepseek/deepseek-v4.1-flash",
				"google/gemini-3.8-flash", "x-ai/grok-4.6"}},
		{llmprovider.ProviderHuggingFace, 119,
			[]string{"deepseek-ai/DeepSeek-V4-Flash-0731", "zai-org/GLM-5.3-Flash", "deepseek-ai/DeepSeek-V4.1-Flash",
				"thinkingmachines/Inkling-Small", "stepfun-ai/Step-3.7-Flash", "stepfun-ai/Step-3.5-Flash"},
			[]string{"zai-org/GLM-5.3", "Qwen/Qwen3.8-27B", "Qwen/Qwen3.8-2.4T-A95B", "deepseek-ai/DeepSeek-V4-Pro-0813",
				"moonshotai/Kimi-K3", "thinkingmachines/Inkling"}},
		{llmprovider.ProviderTogether, 172,
			[]string{"zai-org/GLM-5.3-Flash", "deepseek-ai/DeepSeek-V4-Flash-0731", "deepseek-ai/DeepSeek-V4.1-Flash",
				"MiniMaxAI/MiniMax-M3", "Qwen/Qwen3.6-Plus", "openai/gpt-oss-120b"},
			[]string{"zai-org/GLM-5.3", "deepseek-ai/DeepSeek-V4-Pro-0813", "moonshotai/Kimi-K3", "thinkingmachines/Inkling",
				"zai-org/GLM-5.2", "MiniMaxAI/MiniMax-M3"}},
	} {
		for _, p := range []struct {
			profile Profile
			want    []string
		}{{ProfileUtility, tc.utility}, {ProfileCapable, tc.capable}} {
			cat := replaySnapshot(t, dir, tc.provider, p.profile)
			if !slices.Equal(cat.Recommended, p.want) {
				t.Errorf("%s profile %d:\n got  %v\n want %v", tc.provider, p.profile, cat.Recommended, p.want)
			}
			if len(cat.Usable) != tc.usable {
				t.Errorf("%s: %d usable, want %d", tc.provider, len(cat.Usable), tc.usable)
			}
		}
	}
}

// TestStatic_EqualsLatestSnapshot: each ranked static list is the latest
// snapshot's utility six, in its order, so a list changes only with a new
// snapshot, and a hand edit fails here.
func TestStatic_EqualsLatestSnapshot(t *testing.T) {
	dir := latestSnapshot(t)
	for provider := range snapshotFile {
		six := replaySnapshot(t, dir, provider, ProfileUtility).Recommended
		if static := Static(provider); !slices.Equal(static, six) {
			t.Errorf("%s: static list\n %v\nwant the utility six of %s\n %v", provider, static, dir, six)
		}
	}
}

// TestReplaySnapshot_PinsTheClock: a replay ranks at the snapshot's
// recorded_at, so a snapshot gives the same sixes on any later day.
func TestReplaySnapshot_PinsTheClock(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	replaySnapshot(t, dir, llmprovider.ProviderOpencodeGo, ProfileUtility)
	if want := time.Date(2026, 10, 12, 23, 54, 43, 0, time.UTC); !rankingNow().Equal(want) {
		t.Errorf("the ranking clock is %v during the replay, want recorded_at %v", rankingNow(), want)
	}
}
