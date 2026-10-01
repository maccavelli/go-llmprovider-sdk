package responses

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's truncation_test.go and truncation_api_test.go
// (0015-PLAN S7b).

// TestDecodeResponses_IncompleteIsError: a Responses answer the service marked
// incomplete is an error naming the reason, never an empty success (MADR 0012
// §1.5).
func TestDecodeResponses_IncompleteIsError(t *testing.T) {
	body := `{"id":"r1","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},
		"output":[{"type":"reasoning","summary":[]}]}`
	res, err := Decode(strings.NewReader(body))
	if err == nil || !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), "max_output_tokens") {
		t.Fatalf("decode = %+v/%v, want an ErrInvalidRequest naming max_output_tokens", res, err)
	}
}

// TestIncomplete_Reason: the error is an *APIError of kind ErrIncomplete,
// carrying the service's reason (0015-MADR D7).
func TestIncomplete_Reason(t *testing.T) {
	_, err := Decode(strings.NewReader(
		`{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`))
	var incomplete *llmprovider.APIError
	if !errors.As(err, &incomplete) || !errors.Is(incomplete.Kind, llmprovider.ErrIncomplete) || incomplete.Reason != "max_output_tokens" {
		t.Fatalf("error = %#v, want an *APIError of kind ErrIncomplete with reason max_output_tokens", err)
	}
}
