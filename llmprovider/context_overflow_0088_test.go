package llmprovider

import (
	"errors"
	"net/http"
	"testing"
)

// kiloWindowSeen is the seen label of Kilo's documented window form, an
// overflow form since 0088-MADR ST51.
const kiloWindowSeen = "Kilo's documented form (0088-MADR C97)"

// kiloWindowMessage is Kilo's documented window message (0088-MADR C97,
// kilocode's gateway API reference).
const kiloWindowMessage = "This request exceeds the model's context window of 200000 tokens. " +
	"Your request contains approximately 250000 tokens."

// TestClassify_KiloWindowForm_0088 (ST51): Kilo's documented window message,
// as an HTTP 400 and as the error inside a 200, is ErrContextOverflow,
// beneath ErrInvalidRequest, not retryable.
func TestClassify_KiloWindowForm_0088(t *testing.T) {
	assertOverflow(t, classifyBody(string(ProviderKilo), http.StatusBadRequest, `{"error":{"message":`+quote(kiloWindowMessage)+`}}`))
	err := ClassifyStreamFailureBody(string(ProviderKilo), "400", "", kiloWindowMessage,
		[]byte(`{"error":{"code":400,"message":`+quote(kiloWindowMessage)+`}}`))
	if !errors.Is(err, ErrContextOverflow) || !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("inside a 200: %v; want ErrContextOverflow, beneath ErrInvalidRequest", err)
	}
	// The form's sample in the guard tests' table (the ninth amendment's
	// planned change) is this message.
	if got := overflowSamples[kiloWindowSeen]; got != kiloWindowMessage {
		t.Errorf("overflowSamples[%q] = %q; want Kilo's documented message", kiloWindowSeen, got)
	}
}
