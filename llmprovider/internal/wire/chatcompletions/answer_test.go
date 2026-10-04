package chatcompletions

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_ReportsModel (0020-MADR F11): the model that answered; for
// kilo-auto/* it is the only way to learn which one did.
func TestDecode_ReportsModel(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"id":"c","model":"served-model","choices":[
		{"finish_reason":"stop","message":{"role":"assistant","content":"hi"}}]}`))
	if err != nil || res.Model != "served-model" {
		t.Errorf("Decode = %+v, %v; want model served-model", res, err)
	}
}

// TestDecode_EmptyIsIncomplete (0020-MADR F9): an answer with nothing in it
// has a kind, so WithRetry does not buy it again.
func TestDecode_EmptyIsIncomplete(t *testing.T) {
	for _, body := range []string{`{"choices":[]}`, `{"choices":[{"finish_reason":"stop","message":{"content":""}}]}`} {
		if _, err := Decode(strings.NewReader(body)); !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("Decode(%s) = %v, want ErrIncomplete", body, err)
		}
	}
}
