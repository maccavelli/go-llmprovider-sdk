package catalog

import (
	"strings"
	"testing"
)

// 0028-MADR D-A1: the document is decoded as it streams in, under its limit.
// Its fetch tests went with the fetch (0093-MADR D12, MH30).

// TestDecodeModelMetadata_StreamedLimit: a body one byte past the limit is
// refused, naming it; a valid body decodes; trailing data is still refused,
// as json.Unmarshal refused it.
func TestDecodeModelMetadata_StreamedLimit(t *testing.T) {
	const head, tail = `{"opencode":{"models":{}},"pad":"`, `"}`
	body := head + strings.Repeat("x", metadataLimit+1-len(head)-len(tail)) + tail
	if len(body) != metadataLimit+1 {
		t.Fatalf("body is %d bytes; want %d", len(body), metadataLimit+1)
	}
	if _, err := decodeModelMetadata(strings.NewReader(body)); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Errorf("a body past the limit: err = %v; want one naming 32 MiB", err)
	}
	exact := head + strings.Repeat("x", metadataLimit-len(head)-len(tail)) + tail
	if _, err := decodeModelMetadata(strings.NewReader(exact)); err != nil {
		t.Errorf("a body of exactly the limit: %v; want it decoded", err)
	}
	doc, err := decodeModelMetadata(strings.NewReader(smallMetadataDoc))
	if err != nil || doc.all()[metadataKeyZen] == nil {
		t.Errorf("a valid body: %v, %v; want it decoded", doc, err)
	}
	if _, err := decodeModelMetadata(strings.NewReader(smallMetadataDoc + ` {}`)); err == nil {
		t.Error("trailing data: no error; want one")
	}
}
