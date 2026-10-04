package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestConfigureLLM_UnreachableEndpointAtEOFEnds (0020-MADR F6): an
// unreachable Ollama endpoint on an empty stdin ends with an error, instead
// of answering "try again" for ever.
func TestConfigureLLM_UnreachableEndpointAtEOFEnds(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	done := make(chan error, 1)
	go func() {
		_, err := ConfigureLLM(context.Background(), &TextPrompter{In: strings.NewReader(""), Out: io.Discard},
			Options{Providers: []llmprovider.ProviderID{llmprovider.ProviderOllama},
				Existing: Result{Provider: llmprovider.ProviderOllama, BaseURL: srv.URL}})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("ConfigureLLM = nil, want an error: input is exhausted")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ConfigureLLM still running 5 s after stdin ended")
	}
}

// yesPrompter answers every prompt with its default and every question yes.
type yesPrompter struct{ fakePrompter }

func (y *yesPrompter) Confirm(string, bool) (bool, error) { return true, nil }

func (y *yesPrompter) Input(_, def string) (string, error) { return def, nil }

// TestConfigureLLM_EndpointLoopHonoursContext (0020-MADR F6).
func TestConfigureLLM_EndpointLoopHonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	done := make(chan error, 1)
	go func() {
		_, err := ConfigureLLM(ctx, &yesPrompter{fakePrompter{t: t, selects: []int{0}}},
			Options{Providers: []llmprovider.ProviderID{llmprovider.ProviderOllama},
				Existing: Result{Provider: llmprovider.ProviderOllama, BaseURL: srv.URL}})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ConfigureLLM = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ConfigureLLM ignored its cancelled context")
	}
}

// TestTextPrompter_NotifyWhileInputWaits (0020-MADR F16): a Notify from one
// goroutine while another waits in Input is safe; run with -race.
func TestTextPrompter_NotifyWhileInputWaits(t *testing.T) {
	const rounds = 200
	p := &TextPrompter{In: strings.NewReader(strings.Repeat("\n", rounds)), Out: io.Discard}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range rounds {
			if _, err := p.Input("paste", ""); err != nil {
				t.Errorf("Input: %v", err)
				return
			}
		}
	}()
	for range rounds {
		p.Notify(LevelInfo, "Browser sign-in finished; press Enter to continue")
	}
	<-done
}

// TestTextPrompter_MaskedEntry (0020-MADR F17, F18): the raw-mode entry keeps
// whole runes, ignores escape sequences, erases one rune per Backspace, is
// trimmed, and leaves no \n behind a \r for the next prompt.
func TestTextPrompter_MaskedEntry(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"an arrow key", "abcdefgh\x1b[Dij\r", "abcdefghij"},
		{"an SS3 key", "abc\x1bOAdef\r", "abcdef"},
		{"UTF-8", "pässwörd\r", "pässwörd"},
		{"Backspace erases a rune", "pä\x7f\r", "p"},
		{"pasted with spaces", "  sk-key-1234  \r", "sk-key-1234"},
	} {
		p := &TextPrompter{In: strings.NewReader(c.in), Out: io.Discard}
		if got, err := p.readMasked("Key"); err != nil || got != c.want {
			t.Errorf("%s: readMasked = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	p := &TextPrompter{In: strings.NewReader("sk-abcdefgh\r\n3\n"), Out: io.Discard}
	if _, err := p.readMasked("Key"); err != nil {
		t.Fatal(err)
	}
	idx, err := p.Select("Pick", []Choice{{Label: "a"}, {Label: "b"}, {Label: "c"}}, 0)
	if err != nil || idx != 2 {
		t.Errorf("Select after a CRLF entry = %d, %v; want 2: the \\n must not answer it", idx, err)
	}
}

// chunkReader returns one chunk per Read, as a terminal returns each
// keystroke or paste.
type chunkReader struct{ chunks []string }

func (c *chunkReader) Read(b []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(b, c.chunks[0])
	c.chunks = c.chunks[1:]
	return n, nil
}

// TestTextPrompter_EnterAfterMaskedEntryIsKept (0020-MADR F17): Enter in raw
// mode sends only \r, so the next Enter, a blank answer, is kept for the
// next prompt; only a \n read with the \r, as in a paste, is dropped.
func TestTextPrompter_EnterAfterMaskedEntryIsKept(t *testing.T) {
	p := &TextPrompter{In: &chunkReader{chunks: []string{"sk-abcdefgh\r", "\n", "3\n"}}, Out: io.Discard}
	if _, err := p.readMasked("Key"); err != nil {
		t.Fatal(err)
	}
	choices := []Choice{{Label: "a"}, {Label: "b"}, {Label: "c"}}
	if idx, err := p.Select("Pick", choices, 1); err != nil || idx != 1 {
		t.Errorf("Select after Enter = %d, %v; want the default 1: the blank Enter was dropped", idx, err)
	}
}

// TestConfigureLLM_PastedKeyIsTrimmed (0020-MADR F18): an OpenAI key pasted
// with a leading space is an API key, not an access-only session.
func TestConfigureLLM_PastedKeyIsTrimmed(t *testing.T) {
	store := newMemoryTokenStore()
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 3, 0}, secrets: []string{" " + testKey}}
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store})
	if err != nil || res.Kind != CredAPIKey || res.APIKey != testKey || store.saves != 0 {
		t.Errorf("Kind %q, APIKey %q, saves %d, err %v; want the trimmed key as an API key", res.Kind, res.APIKey, store.saves, err)
	}
}

// TestConfigureLLM_KeepsAnySavedSession (0020-MADR F19): a saved session is
// offered first, whatever the sign-in method; a Kilo session, which has no
// refresh, can be kept with its organization.
func TestConfigureLLM_KeepsAnySavedSession(t *testing.T) {
	for _, c := range []struct {
		name     string
		provider llmprovider.ProviderID
		session  *auth.OAuthSession
		org      string
	}{
		{"OpenAI", llmprovider.ProviderOpenAI, testOAuthSession(llmprovider.ProviderOpenAI), ""},
		{"Kilo", llmprovider.ProviderKilo, &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: kiloWizardToken}, "org-2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			store := newMemoryTokenStore()
			store.sessions[c.provider] = c.session
			f := &fakePrompter{t: t, selects: []int{providerIdx(t, c.provider), 0}, confirms: []bool{true},
				inputs: []string{acceptDefault}, blankSearches: true}
			res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store,
				Existing: Result{Provider: c.provider, Kind: CredOAuth, Organization: c.org, Model: "m-1"}})
			if err != nil || res.Kind != CredOAuth || res.Organization != c.org || store.saves != 0 {
				t.Errorf("Kind %q, org %q, saves %d, err %v; want the saved session kept", res.Kind, res.Organization, store.saves, err)
			}
			if len(f.seenConfirm) == 0 || !strings.Contains(f.seenConfirm[0], "Keep the existing session") {
				t.Errorf("confirms = %q, want the keep question first", f.seenConfirm)
			}
		})
	}
}

// TestConfigureLLM_MethodMenuDefaultsFromExisting (0020-MADR F19).
func TestConfigureLLM_MethodMenuDefaultsFromExisting(t *testing.T) {
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 0}, secrets: []string{testKey}}
	_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
		Existing: Result{Provider: llmprovider.ProviderOpenAI, Kind: CredVendorCLI}})
	if len(f.seenSelectDefault) < 2 || f.seenSelectItems[1][f.seenSelectDefault[1]].Label != methodLabel(t, llmprovider.ProviderOpenAI, llmprovider.AuthImportVendorCLI) {
		t.Errorf("method menu defaults = %v, want the vendor CLI method the Result names", f.seenSelectDefault)
	}
}

// TestConfigureLLM_KiloOrganizationDefaultsFromExisting (0020-MADR F19).
func TestConfigureLLM_KiloOrganizationDefaultsFromExisting(t *testing.T) {
	stubKilo(t, auth.KiloAccount{HasPersonalAccount: true, SelectedOrganizationID: "org-2",
		Organizations: []auth.KiloOrganization{{ID: "org-1", Name: "Acme"}, {ID: "org-2", Name: "Beta"}}}, nil)
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderKilo), 1, 1, 0}}
	_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
		Existing: Result{Provider: llmprovider.ProviderKilo, Organization: "org-1"}})
	for i, title := range f.seenSelect {
		if title == "Use Kilo as:" && f.seenSelectItems[i][f.seenSelectDefault[i]].Label != "Acme" {
			t.Errorf("organization default = %q, want Acme, the one the Result names", f.seenSelectItems[i][f.seenSelectDefault[i]].Label)
		}
	}
}

// TestConfigureLLM_VendorPathDefaultsFromExisting (0020-MADR F19).
func TestConfigureLLM_VendorPathDefaultsFromExisting(t *testing.T) {
	path := writeGrokCLILogin(t, t.TempDir())
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 4, 0}, confirms: []bool{true}}
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
		Existing: Result{Provider: llmprovider.ProviderGrok, Kind: CredVendorCLI, VendorAuthPath: path}})
	if err != nil || res.VendorAuthPath != path {
		t.Errorf("VendorAuthPath = %q, %v; want the saved %q", res.VendorAuthPath, err, path)
	}
}

// TestConfigureLLM_LoginUsesTheCallersClient (0020-MADR F20).
func TestConfigureLLM_LoginUsesTheCallersClient(t *testing.T) {
	client := &http.Client{}
	var got *http.Client
	original := loginDeviceOAuth
	t.Cleanup(func() { loginDeviceOAuth = original })
	loginDeviceOAuth = func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		got = opts.HTTPClient
		return testOAuthSession(provider), nil
	}
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 2, 0}}
	_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), HTTPClient: client})
	if got != client {
		t.Error("the device login did not use Options.HTTPClient")
	}
}

// TestConfigureLLM_KiloProfileGetsTheEndpoint (0020-MADR F20): the Kilo
// profile is read from the endpoint the user entered.
func TestConfigureLLM_KiloProfileGetsTheEndpoint(t *testing.T) {
	stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
	var base string
	kiloProfile = func(_ context.Context, _ string, opts ...llmprovider.Option) (auth.KiloAccount, error) {
		st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)
		if err != nil {
			t.Errorf("profile options: %v", err)
		}
		base = st.BaseURL()
		return auth.KiloAccount{HasPersonalAccount: true}, nil
	}
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderKilo), 1, 0}, inputs: []string{"https://kilo.example/api/gateway"}}
	_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore()})
	if base != "https://kilo.example/api/gateway" {
		t.Errorf("profile base URL = %q, want the entered endpoint", base)
	}
}

// badIndex returns an index out of range for the first Select.
type badIndex struct{ fakePrompter }

func (b *badIndex) Select(string, []Choice, int) (int, error) { return -1, nil }

// TestConfigureLLM_OutOfRangeSelectIsAnError (0020-MADR F49).
func TestConfigureLLM_OutOfRangeSelectIsAnError(t *testing.T) {
	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ConfigureLLM panicked: %v", v)
		}
	}()
	if _, err := ConfigureLLM(context.Background(), &badIndex{fakePrompter{t: t}}, Options{}); err == nil {
		t.Error("ConfigureLLM = nil, want an error for index -1")
	}
}

// preselectRecorder records MultiSelect's preselection.
type preselectRecorder struct {
	fakePrompter
	preselected [][]int
}

func (r *preselectRecorder) MultiSelect(title string, choices []Choice, pre []int) ([]int, error) {
	r.preselected = append(r.preselected, pre)
	return r.fakePrompter.MultiSelect(title, choices, pre)
}

// TestConfigureLLM_PreselectsExistingFallbacks (0020-MADR F50).
func TestConfigureLLM_PreselectsExistingFallbacks(t *testing.T) {
	srv := zenServer(t, http.StatusOK, zenListing([]string{"m-one", "m-two", "m-three"}))
	o := zenOptions()
	o.NeedFallbacks = true
	o.Existing = Result{Provider: llmprovider.ProviderOpencodeZen, Model: "m-one", Fallbacks: []string{"m-three"}}
	r := &preselectRecorder{fakePrompter: fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderOpencodeZen), 0},
		inputs: []string{srv.URL, "", ""}, secrets: []string{"zen-key"}, multiSelects: [][]int{nil}}}
	_, _ = ConfigureLLM(context.Background(), r, o)
	if len(r.preselected) == 0 || len(r.preselected[0]) != 1 {
		t.Fatalf("preselected = %v, want the saved fallback", r.preselected)
	}
}

// TestConfigureLLM_NoStoreStillOffersStorelessMethods (0020-MADR F51).
func TestConfigureLLM_NoStoreStillOffersStorelessMethods(t *testing.T) {
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 0}, secrets: []string{"xai-key-0123456789"}}
	_, _ = ConfigureLLM(context.Background(), f, Options{})
	if len(f.seenSelectItems) < 2 {
		t.Fatalf("selects = %q, want a method menu", f.seenSelect)
	}
	got := labels(f.seenSelectItems[1])
	for _, want := range []llmprovider.AuthMethodID{llmprovider.AuthTokenStdin, llmprovider.AuthImportVendorCLI} {
		if !slices.Contains(got, methodLabel(t, llmprovider.ProviderGrok, want)) {
			t.Errorf("without a TokenStore the menu is %q; want %q, which needs no store", got, want)
		}
	}
}

// TestVendorAuthPath_HomeFromLookupEnv (0020-MADR F25, Q5 a): the default path
// is under the HOME the caller's LookupEnv gives; with none, it says so.
func TestVendorAuthPath_HomeFromLookupEnv(t *testing.T) {
	got, err := vendorAuthPath(llmprovider.ProviderOpenAI, Options{LookupEnv: envOf(map[string]string{"HOME": "/h", "USERPROFILE": "/h"})})
	if err != nil || got != filepath.Join("/h", ".codex", "auth.json") {
		t.Errorf("vendorAuthPath = %q, %v; want it under the given HOME", got, err)
	}
	if _, err := vendorAuthPath(llmprovider.ProviderOpenAI, Options{}); err == nil {
		t.Error("vendorAuthPath with no LookupEnv = nil error; want one naming LookupEnv")
	}
}

// TestTextPrompter_MultiSelectShowsThePreselection (0020-MADR F50; 0020-PLAN
// deviation 2026-10-03): the preselected rows are marked, a blank line keeps
// them and the prompt says so, and 0 clears them. With none, nothing changes.
func TestTextPrompter_MultiSelectShowsThePreselection(t *testing.T) {
	choices := []Choice{{Label: "a"}, {Label: "b", Detail: "d"}, {Label: "c"}}
	var out strings.Builder
	p := &TextPrompter{In: strings.NewReader("\n0\n"), Out: &out}
	if got, err := p.MultiSelect("Fallbacks", choices, []int{0, 2}); err != nil || !slices.Equal(got, []int{0, 2}) {
		t.Errorf("blank = %v, %v; want the preselection [0 2]", got, err)
	}
	shown := out.String()
	for _, want := range []string{"1) a (selected)", "3) c (selected)", "blank keeps 1,3; 0 for none"} {
		if !strings.Contains(shown, want) {
			t.Errorf("output lacks %q:\n%s", want, shown)
		}
	}
	if strings.Contains(shown, "b — d (selected)") || strings.Contains(shown, "blank for none") {
		t.Errorf("output marks b, or says blank is none:\n%s", shown)
	}
	if got, err := p.MultiSelect("Fallbacks", choices, []int{0, 2}); err != nil || len(got) != 0 {
		t.Errorf("0 = %v, %v; want no fallbacks", got, err)
	}
	var plain strings.Builder
	q := &TextPrompter{In: strings.NewReader("\n"), Out: &plain}
	if got, err := q.MultiSelect("Fallbacks", choices, nil); err != nil || got != nil ||
		!strings.Contains(plain.String(), "blank for none") || strings.Contains(plain.String(), "selected") {
		t.Errorf("no preselection = %v, %v; want nil and the old prompt:\n%s", got, err, plain.String())
	}
}

// methodLabel is the label of a provider's sign-in method.
func methodLabel(t *testing.T, provider llmprovider.ProviderID, id llmprovider.AuthMethodID) string {
	t.Helper()
	for _, d := range (Options{}).registry().Descriptors() {
		if d.ID != provider {
			continue
		}
		for _, m := range d.AuthMethods {
			if m.ID == id {
				return m.Label
			}
		}
	}
	t.Fatalf("%s has no method %q", provider, id)
	return ""
}
