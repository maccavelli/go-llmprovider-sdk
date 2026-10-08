package llmprovider

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestClassify_16KiBTokensUnder1ms (0028-MADR D-A10): classification redacts
// only what it can keep, at most 2 KiB of the message, whatever its length.
// The time is measured by BenchmarkClassify_16KiBTokens; this asserts the
// bound that gives it.
func TestClassify_16KiBTokensUnder1ms(t *testing.T) {
	msg := "This request used too many tokens. " + strings.Repeat("detail ", 2300)
	if got := len(redactInput(msg)); got > 2048 {
		t.Fatalf("redactInput kept %d bytes of a %d-byte message; want at most 2048", got, len(msg))
	}
}

// TestRedactInput_NoSecretFragmentAtTheCut: a key cut in two at the bound
// must not leave a fragment too short for redaction to recognise. Twenty long
// keys come first, so redaction shrinks the message and the text at the cut
// falls inside the 512 bytes kept.
func TestRedactInput_NoSecretFragmentAtTheCut(t *testing.T) {
	var b strings.Builder
	for range 20 {
		b.WriteString(" sk-proj-" + strings.Repeat("A", 92))
	}
	b.WriteString(strings.Repeat("x", 2036-b.Len()-1) + " ")
	straddling := "sk-proj-QWERTYUIOPASDFGHJKLZXCVBNMQWERTYUIOPASDF"
	b.WriteString(straddling + " and more text after it")
	if start := strings.Index(b.String(), straddling); start != 2036 {
		t.Fatalf("the straddling key starts at %d; want 2036", start)
	}
	raw, err := json.Marshal(map[string]any{"error": map[string]any{"message": b.String(), "type": "invalid_request_error"}})
	if err != nil {
		t.Fatal(err)
	}
	err = ClassifyHTTPError("together", &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader(string(raw)))})
	if strings.Contains(err.Error(), "QWER") || strings.Contains(err.Error(), "sk-proj-Q") {
		t.Fatalf("a fragment of the straddling key was kept: %v", err)
	}
}
