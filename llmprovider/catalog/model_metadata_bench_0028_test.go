package catalog

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// 0028-PLAN Phase 6, step 4: the metadata decode's cost, before and after
// D-A1's streamed decode.

// generatedMetadata is a models.dev-shaped document of about 5 MB,
// deterministic (seed 28).
func generatedMetadata() string {
	r := rand.New(rand.NewPCG(28, 0))
	var b strings.Builder
	b.WriteString("{")
	for s, section := range []string{"opencode", "opencode-go", "huggingface", "togetherai", "other-a", "other-b"} {
		if s > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `%q:{"id":%q,"name":"Section %d","models":{`, section, section, s)
		for m := range 2500 {
			if m > 0 {
				b.WriteString(",")
			}
			id := fmt.Sprintf("model-%d-%05d", s, m)
			fmt.Fprintf(&b, `%q:{"id":%q,"name":"Model %d","family":"fam-%d","reasoning":%t,`+
				`"cost":{"input":%.2f,"output":%.2f},"limit":{"context":%d,"output":%d},`+
				`"release_date":"2026-%02d-%02d","status":"","provider":{"npm":"@ai-sdk/openai-compatible"},`+
				`"modalities":{"input":["text","image"],"output":["text"]},"reasoning_options":[{"type":"effort","values":["low","medium","high"]}]}`,
				id, id, m, r.IntN(40), r.IntN(2) == 0, r.Float64()*10, r.Float64()*30,
				1<<(14+r.IntN(5)), 1<<(12+r.IntN(4)), 1+r.IntN(12), 1+r.IntN(28))
		}
		b.WriteString("}}")
	}
	b.WriteString("}")
	return b.String()
}

// BenchmarkDecodeModelMetadata: the decode of a document of about 5 MB.
func BenchmarkDecodeModelMetadata(b *testing.B) {
	doc := generatedMetadata()
	b.SetBytes(int64(len(doc)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeModelMetadata(strings.NewReader(doc)); err != nil {
			b.Fatal(err)
		}
	}
}

// TestDecodeModelMetadata_MatchesUnmarshal (0028-PLAN D12): json/v2, with
// the three v1 behaviours it takes, decodes the generated document, and
// documents with mixed-case names, duplicate names, invalid UTF-8, nulls and
// each form of interleaved, exactly as json.Unmarshal does.
func TestDecodeModelMetadata_MatchesUnmarshal(t *testing.T) {
	for _, body := range []string{
		generatedMetadata(),
		smallMetadataDoc,
		`{"OpenCode":{"Models":{"a":{"Name":"A","LIMIT":{"Context":8}}}}}`,
		`{"opencode":{"models":{"a":{"name":"first"}}},"opencode":{"models":{"b":{"name":"second"}}}}`,
		"{\"opencode\":{\"models\":{\"a\":{\"name\":\"bad \xff utf-8\"}}}}",
		`{"opencode":{"models":{"a":{"interleaved":true},"b":{"interleaved":{"field":"reasoning_content"}},
			"c":{"interleaved":null,"cost":null,"reasoning":null,"reasoning_options":null,"limit":null,"provider":null},
			"e":{"Interleaved":{"field":"f"},"NAME":"upper"}}}}`,
	} {
		got, err := decodeModelMetadata(strings.NewReader(body))
		if err != nil {
			t.Fatalf("decodeModelMetadata(%.40q…): %v", body, err)
		}
		var raw struct {
			Zen      *modelMetadataSection `json:"opencode"`
			Go       *modelMetadataSection `json:"opencode-go"`
			HF       *modelMetadataSection `json:"huggingface"`
			Together *modelMetadataSection `json:"togetherai"`
		}
		if err := json.Unmarshal([]byte(body), &raw); err != nil {
			t.Fatalf("json.Unmarshal(%.40q…): %v", body, err)
		}
		want := modelMetadataDoc{}
		for key, s := range map[string]*modelMetadataSection{
			metadataKeyZen: raw.Zen, metadataKeyGo: raw.Go, metadataKeyHF: raw.HF, metadataKeyTogether: raw.Together,
		} {
			if s != nil && s.Models != nil {
				want[key] = s.Models
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%.40q…: json/v2 and json.Unmarshal differ", body)
		}
	}
}
