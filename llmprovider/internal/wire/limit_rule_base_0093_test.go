package wire

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLimitRule_MessageNamesWithMetadata_0093 (MH19; D9, Q2): a request on a
// wire that requires a limit, with no source, is refused with an error
// matching ErrInvalidRequest whose message names catalog.WithMetadata, beside
// the fixes it named before (0089-MADR DL51).
func TestLimitRule_MessageNamesWithMetadata_0093(t *testing.T) {
	_, err := ChooseLimit(LimitInput{Sends: true, Required: true, Provider: "claude", Model: "claude-x"})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("ChooseLimit error = %v; want one matching ErrInvalidRequest", err)
	}
	for _, want := range []string{"catalog.WithMetadata", "WithMaxTokens", "catalog.WithModelLayer", "Request.MaxOutputTokens",
		"maxTokens in models.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %s: %v", want, err)
		}
	}
}
