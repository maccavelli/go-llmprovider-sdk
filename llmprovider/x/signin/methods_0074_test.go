package signin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// methodRow is one method of 0074-MADR D1's table of built-in values,
// spelled out here and not read from the descriptors it holds them to.
type methodRow struct {
	id, kind, completion, paste, first, yields, client, slot string
	refreshable, revocable                                   bool
}

var (
	rowAPIKey = methodRow{id: "api_key", kind: "api_key", completion: "value", yields: "api_key", client: "none"}
	rowImport = methodRow{id: "import_vendor_cli", kind: "vendor_import", completion: "local_file", yields: "read_through", client: "none"}
)

// builtinMethods are the explicit methods of the three descriptors that list
// any, with Grok's browser sign-in a paste (the live probe of 2026-11-13).
var builtinMethods = map[llmprovider.ProviderID][]methodRow{
	llmprovider.ProviderOpenAI: {
		rowAPIKey,
		{"browser_oauth", "browser", "browser_paste", "url_or_code", "", "oauth_session", "vendor_borrowed", "openai", true, true},
		{"device_code", "device_code", "device", "", "", "oauth_session", "vendor_borrowed", "openai", true, true},
		{"token_stdin", "token_paste", "value", "", "", "access_only", "none", "openai", false, true},
		rowImport,
		{"chatgpt_plan", "browser", "browser_paste", "url_or_code", "url_only", "oauth_session", "app_registered", "openai", true, true},
	},
	llmprovider.ProviderGrok: {
		rowAPIKey,
		{"browser_oauth", "browser", "browser_paste", "url_or_code", "", "oauth_session", "vendor_borrowed", "grok", true, true},
		{"device_code", "device_code", "device", "", "", "oauth_session", "vendor_borrowed", "grok", true, true},
		{"token_stdin", "token_paste", "value", "", "", "api_key", "none", "", false, false},
		rowImport,
	},
	llmprovider.ProviderKilo: {
		rowAPIKey,
		{"device_code", "device_code", "device", "", "", "device_key", "none", "kilo", false, false},
	},
}

// builtinDescriptors are the ten built-in descriptors and chatcompat's.
func builtinDescriptors() []llmprovider.Descriptor {
	return append(providers.Default().Descriptors(), chatcompat.Descriptor("local", "Local server"))
}

// TestMethods_0074: each built-in descriptor's methods, field for field;
// Claude's implicit key as one api_key method; Ollama, which requires no
// key, as none (0074-MADR D1, D2).
func TestMethods_0074(t *testing.T) {
	seen := 0
	for _, d := range builtinDescriptors() {
		got := Methods(d)
		rows, listed := builtinMethods[d.ID]
		switch {
		case listed:
			seen++
			if len(got) != len(rows) || len(got) != len(d.AuthMethods) {
				t.Fatalf("%s: %d methods; want %d", d.ID, len(got), len(rows))
			}
			for i, row := range rows {
				assertMethod(t, d.ID, got[i], row, d.AuthMethods[i])
			}
		case d.RequiresAPIKey:
			if len(got) != 1 {
				t.Fatalf("%s: %d methods; want its implicit api_key", d.ID, len(got))
			}
			assertMethod(t, d.ID, got[0], rowAPIKey, llmprovider.AuthMethod{Label: got[0].Label})
			if !strings.Contains(got[0].Label, "API key") {
				t.Errorf("%s: the implicit key's label is %q", d.ID, got[0].Label)
			}
		default:
			if len(got) != 0 {
				t.Errorf("%s requires no key and lists no method; Methods = %+v, want none", d.ID, got)
			}
		}
	}
	if seen != len(builtinMethods) {
		t.Errorf("%d of %d descriptors with methods were seen", seen, len(builtinMethods))
	}
	claude, ok := providers.Default().Descriptor(llmprovider.ProviderClaude)
	if !ok || len(Methods(claude)) != 1 || Methods(claude)[0].ID != "api_key" {
		t.Errorf("Claude's methods = %+v; want one implicit api_key", Methods(claude))
	}
	ollama, ok := providers.Default().Descriptor(llmprovider.ProviderOllama)
	if !ok || ollama.RequiresAPIKey || len(Methods(ollama)) != 0 {
		t.Errorf("Ollama's methods = %+v; want none", Methods(ollama))
	}
}

// assertMethod fails t unless m has row's facts, the schema version and the
// paste feature, and the label and detail of source.
func assertMethod(t *testing.T, id llmprovider.ProviderID, m authcontract.Method, row methodRow, source llmprovider.AuthMethod) {
	t.Helper()
	want := authcontract.Method{
		SchemaVersion: authcontract.SchemaVersion, ID: row.id, Kind: authcontract.Kind(row.kind), Label: source.Label,
		Detail: source.Detail, Completion: authcontract.Completion(row.completion), Paste: authcontract.Paste(row.paste),
		FirstPaste: authcontract.Paste(row.first), Yields: authcontract.Yields(row.yields),
		ClientIdentity: authcontract.ClientIdentity(row.client), Slot: row.slot,
		Refreshable: row.refreshable, Revocable: row.revocable,
	}
	if row.paste != "" {
		want.Features = []authcontract.Feature{authcontract.FeaturePaste}
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("%s %s:\n got  %+v\n want %+v", id, row.id, m, want)
	}
}

// TestMethods_StartFactsAreTheDescriptors_0074: the facts Start holds for
// each pair it drives are the descriptor's method, as Methods describes it.
func TestMethods_StartFactsAreTheDescriptors_0074(t *testing.T) {
	for p := range driven {
		d, ok := providers.Default().Descriptor(p.provider)
		if !ok {
			t.Fatalf("no descriptor for %s", p.provider)
		}
		i := slices.IndexFunc(d.AuthMethods, func(m llmprovider.AuthMethod) bool { return m.ID == p.method })
		if i < 0 {
			t.Fatalf("%s lists no %s", p.provider, p.method)
		}
		got, _ := methodFor(p.provider, p.method)
		if want := Methods(d)[i]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s %s:\n Start has  %+v\n descriptor %+v", p.provider, p.method, got, want)
		}
	}
}

// TestMethodsUseContractValues_0074: every string fact of every built-in
// descriptor is one of the contract's constants of its type (0074-MADR D1).
func TestMethodsUseContractValues_0074(t *testing.T) {
	kindsOK := []string{string(authcontract.KindAPIKey), string(authcontract.KindTokenPaste), string(authcontract.KindDeviceCode),
		string(authcontract.KindBrowser), string(authcontract.KindOAuth), string(authcontract.KindVendorImport), string(authcontract.KindHostOnly)}
	completions := []string{string(authcontract.CompletionValue), string(authcontract.CompletionDevice),
		string(authcontract.CompletionBrowserPaste), string(authcontract.CompletionBrowserLoopback), string(authcontract.CompletionAtStart),
		string(authcontract.CompletionLocalFile), string(authcontract.CompletionHostCommand)}
	pastes := []string{string(authcontract.PasteNone), string(authcontract.PasteURLOrCode), string(authcontract.PasteURLOnly), string(authcontract.PasteCode)}
	yields := []string{string(authcontract.YieldsAPIKey), string(authcontract.YieldsOAuthSession), string(authcontract.YieldsAccessOnly),
		string(authcontract.YieldsDeviceKey), string(authcontract.YieldsReadThrough), string(authcontract.YieldsUnknown)}
	clients := []string{string(authcontract.ClientNone), string(authcontract.ClientVendorBorrowed),
		string(authcontract.ClientAppRegistered), string(authcontract.ClientUnknown)}
	check := func(d llmprovider.ProviderID, m llmprovider.AuthMethodID, field, value string, allowed []string) {
		if !slices.Contains(allowed, value) {
			t.Errorf("%s %s: %s = %q; want one of the contract's %v", d, m, field, value, allowed)
		}
	}
	for _, d := range builtinDescriptors() {
		for _, m := range d.AuthMethods {
			check(d.ID, m.ID, "Kind", m.Kind, kindsOK)
			check(d.ID, m.ID, "Completion", m.Completion, completions)
			check(d.ID, m.ID, "Paste", m.Paste, pastes)
			check(d.ID, m.ID, "FirstPaste", m.FirstPaste, pastes)
			check(d.ID, m.ID, "Yields", m.Yields, yields)
			check(d.ID, m.ID, "ClientIdentity", m.ClientIdentity, clients)
		}
	}
}

// TestSessionKindsMatchContract_0074: in x/signin, which imports both auth
// and authcontract, a table copied from 0073-MADR's amendment of 2026-11-11
// pairs each auth.SessionKind constant with the authcontract constant of the
// same name, and each pair's strings equal the row's value; the SessionKind
// constants that go/parser finds in auth's non-test files are exactly the
// table's auth column (0074-MADR D5). The contract's side is held to the
// same table by 0073-PLAN's TestSessionKindValues_0073.
func TestSessionKindsMatchContract_0074(t *testing.T) {
	table := []struct {
		name, value string
		auth        auth.SessionKind
		contract    authcontract.SessionKind
	}{
		{"SessionChatGPT", "chatgpt", auth.SessionChatGPT, authcontract.SessionChatGPT},
		{"SessionChatGPTPlan", "chatgpt_plan", auth.SessionChatGPTPlan, authcontract.SessionChatGPTPlan},
		{"SessionGrok", "grok", auth.SessionGrok, authcontract.SessionGrok},
		{"SessionKiloDevice", "kilo_device", auth.SessionKiloDevice, authcontract.SessionKiloDevice},
	}
	var names []string
	for _, row := range table {
		names = append(names, row.name)
		if string(row.auth) != row.value || string(row.contract) != row.value {
			t.Errorf("%s: auth %q, contract %q; want both %q", row.name, row.auth, row.contract, row.value)
		}
	}
	found := authSessionKindNames(t)
	slices.Sort(names)
	if !slices.Equal(found, names) {
		t.Errorf("auth's SessionKind constants are %v; want exactly the table's %v", found, names)
	}
}

// authSessionKindNames lists the constants of type SessionKind that
// llmprovider/auth's non-test files declare, sorted.
func authSessionKindNames(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "auth")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if ident, ok := value.Type.(*ast.Ident); ok && ident.Name == "SessionKind" {
					for _, name := range value.Names {
						names = append(names, name.Name)
					}
				}
			}
		}
	}
	slices.Sort(names)
	return names
}
