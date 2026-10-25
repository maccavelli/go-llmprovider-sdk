package catalog

import (
	"fmt"
	"testing"
)

// benchIDs0061 is a listing of a thousand ids, shaped as gateways and local
// servers name models: an organisation, a family, a version, a size and a
// quantisation (0061-PLAN Phase 4b, S5).
func benchIDs0061() []string {
	orgs := []string{"meta-llama", "qwen", "google", "mistralai", "deepseek-ai", "anthropic", "openai", "zai", "moonshot", "ibm"}
	families := []string{"llama", "qwen", "gemma", "mistral", "deepseek", "claude-sonnet", "gpt", "glm", "kimi", "granite"}
	sizes := []string{"1b", "3b", "7b", "8b", "12b", "14b", "32b", "70b", "120b", "235b"}
	quants := []string{"", "-instruct", "-q4_k_m", "-awq", "-fp8", "-gguf", "-instruct-q5_k_m", "-mlx", "-bf16", "-int4"}
	ids := make([]string, 0, 1000)
	for i := range 1000 {
		ids = append(ids, fmt.Sprintf("%s/%s-%d.%d-%s%s", orgs[i%10], families[(i/10)%10], 2+i%3, i%7, sizes[(i/100)%10], quants[(i/7)%10]))
	}
	return ids
}

// benchQueries0061 are a word, two words, a size, a typo and a query that
// matches nothing.
var benchQueries0061 = []string{"sonnet", "qwen 32b", "qwen3:8b", "gema", "zzzz"}

// BenchmarkSearch_1000_0061 (0061-PLAN Phase 4b, S5): Search over a thousand
// ids, which indexes them on each call. Its figures are in the PLAN's record,
// before the engine and after.
func BenchmarkSearch_1000_0061(b *testing.B) {
	ids := benchIDs0061()
	for _, q := range benchQueries0061 {
		b.Run(q, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = Search("", ids, q)
			}
		})
	}
}

// BenchmarkCatalogSearch_1000_0061 (0061-PLAN Phase 4b, S5): Catalog.Search
// over the same thousand, on the index built once, as List builds it; an
// index built per query shows in its allocations.
func BenchmarkCatalogSearch_1000_0061(b *testing.B) {
	cat := Catalog{Usable: benchIDs0061()}
	cat.index = cat.newIndex()
	for _, q := range benchQueries0061 {
		b.Run(q, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = cat.Search(q)
			}
		})
	}
}
