package wizard

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-MADR H10b: the wizard names the researched foreign keys, and warns,
// without refusing, when a typed key does not have its provider's confirmed
// shape (0028-PLAN D3). The values are built to a shape; none is a key.

// shaped is n characters cycled from charset.
func shaped(charset string, n int) string {
	return strings.Repeat(charset, n/len(charset)+1)[:n]
}

const (
	shapeAlnum = "aB3dE5gH7jK9mN1pQ"
	shapeB64   = "aB3-dE5_gH7jK9mN1pQ"
	shapeHex   = "0a1b2c3d4e5f6789"
)

// TestForeignKey_Researched: each researched prefix, and an OpenCode key, is
// named when pasted as an OpenAI credential; OpenAI's own keys are not.
func TestForeignKey_Researched(t *testing.T) {
	for _, c := range []struct{ value, want string }{
		{"sk-or-v1-" + shaped(shapeHex, 64), "an OpenRouter"},
		{"AQ.Ab" + shaped(shapeB64, 50), "a Google Gemini"},
		{"api_org_" + shaped(shapeAlnum, 34), "a Hugging Face"},
		{"sk-" + shaped(shapeAlnum, 64), "an OpenCode"},
		{"sk-ant-api03-" + shaped(shapeB64, 93) + "AA", "an Anthropic"},
		{"sk-proj-" + shaped(shapeB64, 74) + "T3BlbkFJ" + shaped(shapeB64, 74), ""},
		{"sk-" + shaped(shapeAlnum, 20) + "T3BlbkFJ" + shaped(shapeAlnum, 20), ""},
	} {
		if got := foreignKeyVendor(c.value); got != c.want {
			t.Errorf("foreignKeyVendor(%.12s…) = %q; want %q", c.value, got, c.want)
		}
	}
}

// TestProviderShapeWarning: a typed key without its provider's shape gets a
// warning naming the shape, never the key, and the run goes on with it; a
// well-shaped key, and any key for a provider with no confirmed shape, gets
// none.
func TestProviderShapeWarning(t *testing.T) {
	malformed := "not-a-key-" + shaped(shapeAlnum, 20)
	for _, c := range []struct {
		id        llmprovider.ProviderID
		wellShape string
	}{
		{llmprovider.ProviderOpenAI, "sk-proj-" + shaped(shapeB64, 74) + "T3BlbkFJ" + shaped(shapeB64, 74)},
		{llmprovider.ProviderClaude, "sk-ant-api03-" + shaped(shapeB64, 93) + "AA"},
		{llmprovider.ProviderGemini, "AIza" + shaped(shapeB64, 35)},
		{llmprovider.ProviderGemini, "AQ.Ab" + shaped(shapeB64, 50)},
		{llmprovider.ProviderTogether, "tgp_v1_" + shaped(shapeB64, 43)},
		{llmprovider.ProviderHuggingFace, "hf_" + shaped(shapeAlnum, 34)},
		{llmprovider.ProviderOpencodeZen, "sk-" + shaped(shapeAlnum, 64)},
		{llmprovider.ProviderOpencodeGo, "sk-" + shaped(shapeAlnum, 64)},
		{llmprovider.ProviderGrok, "xai-" + shaped(shapeAlnum, 80)},
	} {
		d := llmprovider.Descriptor{ID: c.id, Label: string(c.id), RequiresAPIKey: true}
		f := newFake(t, fakePrompter{secrets: []string{malformed, c.wellShape}})
		for i, key := range []string{malformed, c.wellShape} {
			before := len(f.seenNotify)
			got, from, err := resolveAPIKey(f, d, Options{})
			if err != nil || got != key || from != keyTyped {
				t.Fatalf("%s: resolveAPIKey = %q, %v, %v; want the typed key", c.id, got, from, err)
			}
			warned := f.seenNotify[before:]
			if i == 0 && (len(warned) != 1 || !strings.Contains(warned[0], "does not look like")) {
				t.Errorf("%s: malformed key: notices %v; want one warning", c.id, warned)
			}
			if i == 1 && len(warned) != 0 {
				t.Errorf("%s: well-shaped key: notices %v; want none", c.id, warned)
			}
			for _, n := range warned {
				if strings.Contains(n, key) {
					t.Errorf("%s: a notice shows the key: %q", c.id, n)
				}
			}
		}
	}
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderKilo, llmprovider.ProviderOllama} {
		d := llmprovider.Descriptor{ID: id, Label: string(id), RequiresAPIKey: true}
		f := newFake(t, fakePrompter{secrets: []string{malformed}})
		if _, _, err := resolveAPIKey(f, d, Options{}); err != nil || len(f.seenNotify) != 0 {
			t.Errorf("%s: err %v, notices %v; want no shape check", id, err, f.seenNotify)
		}
	}
}
