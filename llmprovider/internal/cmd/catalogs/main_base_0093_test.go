package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// askedFetcher0093 is fakeFetcher, counting each time the document is asked
// of it.
type askedFetcher0093 struct {
	*fakeFetcher
	asked int
}

func (f *askedFetcher0093) Metadata(ctx context.Context) (catalog.Metadata, error) {
	f.asked++
	return f.fakeFetcher.Metadata(ctx)
}

// TestCatalogs_RequiresDoc_0093 (MH26; D6): with no -doc the command is a
// usage error, exit 2, naming -doc, and asks for no document.
func TestCatalogs_RequiresDoc_0093(t *testing.T) {
	f := &askedFetcher0093{fakeFetcher: &fakeFetcher{
		listed: map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-haiku-5-5"}}}}
	var out, errOut bytes.Buffer
	rc := run(nil, &out, &errOut, map[llmprovider.ProviderID]string{llmprovider.ProviderClaude: testKey}, f)
	if rc != 2 {
		t.Errorf("run with no -doc = %d; want 2", rc)
	}
	if !strings.Contains(errOut.String(), "-doc") {
		t.Errorf("stderr %q does not name -doc", errOut.String())
	}
	if f.asked != 0 {
		t.Errorf("the document was asked for %d time(s); want none", f.asked)
	}
}
