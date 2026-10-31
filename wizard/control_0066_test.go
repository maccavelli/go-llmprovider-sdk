package wizard

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// 0066-PLAN Phase 1 (V1; Q3 (a)): no control character a server sends
// reaches the terminal, a menu or the Result.

// compatID0066 is the Chat Completions provider these tests register.
const compatID0066 llmprovider.ProviderID = "lmstudio"

// compatRegistry0066 is the registry chatcompat's package doc shows.
func compatRegistry0066(t *testing.T) *llmprovider.Registry {
	t.Helper()
	r := llmprovider.NewRegistry()
	if err := r.Register(chatcompat.Descriptor(compatID0066, "LM Studio"),
		func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return chatcompat.New(append(opts, chatcompat.WithID(compatID0066))...)
		}); err != nil {
		t.Fatal(err)
	}
	return r
}

// hasControl0066 reports a C0 control but tab, DEL, or a C1 control.
func hasControl0066(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f })
}

// TestTextPrompter_ListedIDEscapes_0066: the MADR's probe J. A server lists
// an id holding an OSC title, a BEL and a screen clear beside a plain one;
// no ESC or BEL reaches the terminal, and row 1 is the plain id.
func TestTextPrompter_ListedIDEscapes_0066(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"evil\u001b]0;owned\u0007\u001b[2Jmodel","object":"model"},`+
			`{"id":"plain-model","object":"model"}]}`)
	}))
	t.Cleanup(srv.Close)
	var out bytes.Buffer
	p := &TextPrompter{In: strings.NewReader("1\n" + srv.URL + "/v1\n\n1\n"), Out: &out}
	res, err := ConfigureLLM(context.Background(), p, Options{Registry: compatRegistry0066(t), Discover: true,
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}})
	if err != nil {
		t.Fatal(err)
	}
	if n, bel := bytes.Count(out.Bytes(), []byte{0x1b}), bytes.Count(out.Bytes(), []byte{0x07}); n != 0 || bel != 0 {
		t.Errorf("%d ESC and %d BEL bytes reached the terminal; want none", n, bel)
	}
	if res.Model != "plain-model" {
		t.Errorf("Model = %q; want plain-model", res.Model)
	}
}

// TestTextPrompter_StripsEveryArgument_0066: a control character in any
// text a TextPrompter prints, a title, a label, a detail, a question, a
// prompt or a default, does not reach Out; a default accepted is returned as
// it was given.
func TestTextPrompter_StripsEveryArgument_0066(t *testing.T) {
	const bad = "\x1b[2J\u009b"
	var out bytes.Buffer
	p := &TextPrompter{In: strings.NewReader("1\n1\n\n\nsecret-value\n"), Out: &out}
	if _, err := p.Select("title"+bad, []Choice{{Label: "label" + bad, Detail: "detail" + bad}}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.MultiSelect("multi"+bad, []Choice{{Label: "label" + bad, Detail: "detail" + bad}}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Confirm("question"+bad, true); err != nil {
		t.Fatal(err)
	}
	got, err := p.Input("prompt"+bad, "default"+bad)
	if err != nil {
		t.Fatal(err)
	}
	if got != "default"+bad {
		t.Errorf("Input returned %q; want the default as it was given", got)
	}
	if _, err := p.Secret("secret" + bad); err != nil {
		t.Fatal(err)
	}
	if hasControl0066(strings.ReplaceAll(strings.ReplaceAll(out.String(), "\n", ""), "\r", "")) {
		t.Errorf("Out holds a control character: %q", out.String())
	}
}

// TestKiloOrg_LabelStripped_0066: a Kilo organization's name and role reach
// the menu without their control characters; the id chosen is kept as it
// came.
func TestKiloOrg_LabelStripped_0066(t *testing.T) {
	stubKilo(t, auth.KiloAccount{HasPersonalAccount: true, Organizations: []auth.KiloOrganization{
		{ID: "org-1", Name: "Ev\x1b]0;x\x07il", Role: "ad\u009bmin"}}}, nil)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderKilo), 1, 1, 0},
		inputs: []string{acceptDefault}, blankSearches: true})
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore()})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.Index(f.seenSelect, "Use Kilo as:")
	if i < 0 {
		t.Fatalf("selects %q; want the organization menu", f.seenSelect)
	}
	for _, c := range f.seenSelectItems[i] {
		if hasControl0066(c.Label) || hasControl0066(c.Detail) {
			t.Errorf("menu row %q / %q holds a control character", c.Label, c.Detail)
		}
	}
	if res.Organization != "org-1" {
		t.Errorf("Organization = %q; want org-1", res.Organization)
	}
}

// TestProviderCatalog_ControlIDsDropped_0066: a provider's own ListModels,
// which catalog does not see, gives the wizard its clean ids only.
func TestProviderCatalog_ControlIDsDropped_0066(t *testing.T) {
	var built []*thirdParty
	reg := thirdPartyRegistry(t, thirdParty{models: []string{"acme-a", "ev\x1b[2Jil", "acme-b", "c1\u009bx"}}, true, &built)
	f := newFake(t, fakePrompter{secrets: []string{"acme-key"}})
	_, cat, err := ResolveCredential(context.Background(), f, Options{Registry: reg, Discover: true}, thirdPartyID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"acme-a", "acme-b"}
	if !slices.Equal(cat.Usable, want) || !slices.Equal(cat.Recommended, want) {
		t.Errorf("Usable %q, Recommended %q; want %q", cat.Usable, cat.Recommended, want)
	}
}
