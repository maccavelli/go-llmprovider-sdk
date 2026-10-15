package catalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOllamaTagInfo_0042: an /api/tags entry's capabilities say only what
// the model writes, as /api/show's do; its tools, input and reasoning come
// from /api/show alone (0042-MADR D1).
func TestOllamaTagInfo_0042(t *testing.T) {
	for _, tc := range []struct {
		capabilities []string
		output       []Modality
		known        bool
	}{
		{[]string{"completion", "tools", "insert"}, []Modality{ModalityText}, true},
		{[]string{"embedding"}, nil, true},
		{[]string{"image"}, []Modality{ModalityImage}, true},
		{[]string{"completion", "image"}, []Modality{ModalityText, ModalityImage}, true},
		{[]string{"decision"}, nil, true},
		{nil, nil, false},
		{[]string{}, nil, false},
	} {
		info := ollamaTagInfo(ollamaTag{Name: "m", Capabilities: tc.capabilities})
		if info.ID != "m" || info.Has(FieldOutput) != tc.known || !slices.Equal(info.Output, tc.output) {
			t.Errorf("%q: %s Output %v, known %t; want m, %v, known %t", tc.capabilities, info.ID, info.Output,
				info.Has(FieldOutput), tc.output, tc.known)
		}
		if tc.known && info.SourceOf(FieldOutput) != "listing:ollama" {
			t.Errorf("%q: Output's source %q; want listing:ollama", tc.capabilities, info.SourceOf(FieldOutput))
		}
		for _, f := range []Field{FieldTools, FieldInput, FieldReasoning} {
			if info.Has(f) {
				t.Errorf("%q: field %d set from /api/tags", tc.capabilities, f)
			}
		}
	}
}

// TestFetchOllamaTags_0042: /api/tags decodes in order, with each entry's
// capabilities where Ollama lists them (v0.30.0 on), and nil where not.
func TestFetchOllamaTags_0042(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"models":[{"name":"b:1","capabilities":["completion","tools"]},{"name":"a:1"}]}`)
	}))
	t.Cleanup(srv.Close)
	tags, err := fetchOllamaTags(context.Background(), llmprovider.Token{}, testConfig(t, llmprovider.WithBaseURL(srv.URL)))
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0].Name != "b:1" || !slices.Equal(tags[0].Capabilities, []string{"completion", "tools"}) ||
		tags[1].Name != "a:1" || tags[1].Capabilities != nil {
		t.Errorf("tags %+v; want b:1 with [completion tools], then a:1 with none", tags)
	}
}
