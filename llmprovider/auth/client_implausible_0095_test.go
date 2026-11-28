package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLoadClient_ImplausibleClientID_0095 (0095-MADR FU10; 0082-MADR N-a): a
// client record that decodes but names a client id that is not printable
// text, such as bytes that were not UTF-8, is an error that names the file
// and not the id, and no record; a plausible one, or none, loads as before.
func TestLoadClient_ImplausibleClientID_0095(t *testing.T) {
	for _, tc := range []struct {
		name string
		file []byte
		want ClientRecord
		bad  bool
	}{
		{"bytes that are not UTF-8", []byte("{\"client_id\":\"app_\xff\xfe\"}"), ClientRecord{}, true},
		{"a replacement character written out", []byte(`{"client_id":"app_�"}`), ClientRecord{}, true},
		{"a control character", []byte(`{"client_id":"app_\u0001x"}`), ClientRecord{}, true},
		{"a plausible id", []byte(`{"client_id":"app_Ab12-x","host_id":"h1"}`), ClientRecord{ClientID: "app_Ab12-x", HostID: "h1"}, false},
		{"a host id alone", []byte(`{"host_id":"h1"}`), ClientRecord{HostID: "h1"}, false},
	} {
		dir := t.TempDir()
		fs := &FileTokenStore{Dir: dir}
		path := filepath.Join(dir, "openai.client")
		if err := os.WriteFile(path, tc.file, 0o600); err != nil {
			t.Fatal(err)
		}
		rec, err := fs.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
		if rec != tc.want || (err != nil) != tc.bad {
			t.Errorf("%s: record %+v, err %v; want %+v, an error: %t", tc.name, rec, err, tc.want, tc.bad)
			continue
		}
		if tc.bad && (!strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "app_")) {
			t.Errorf("%s: error %q; want the file named and no part of the id", tc.name, err)
		}
	}
}
