package wizard

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDescribe_ToolSchemaRefused_0084: a refusal of a tool's schema is worded
// by its own class, before "invalid request", which it also matches (0084-MADR
// D8, ST35). The words never carry the tool's name or the service's text.
func TestDescribe_ToolSchemaRefused_0084(t *testing.T) {
	refused := &llmprovider.APIError{Provider: "grok", Status: http.StatusBadRequest, Kind: llmprovider.ErrToolSchemaRefused,
		Message: maskedEcho0056, Tool: "record_city"}
	if got, want := describe(fmt.Errorf("send: %w", refused)), "tool schema refused, HTTP 400"; got != want {
		t.Errorf("describe(ErrToolSchemaRefused, HTTP 400) = %q; want %q", got, want)
	}
	// It is an invalid request too, and a plain one is still worded as one.
	plain := &llmprovider.APIError{Provider: "grok", Status: http.StatusBadRequest, Kind: llmprovider.ErrInvalidRequest,
		Message: maskedEcho0056}
	if got, want := describe(plain), "invalid request, HTTP 400"; got != want {
		t.Errorf("describe(ErrInvalidRequest, HTTP 400) = %q; want %q", got, want)
	}
	// The bare sentinel, with no status.
	if got, want := describe(fmt.Errorf("send: %w", llmprovider.ErrToolSchemaRefused)), "tool schema refused"; got != want {
		t.Errorf("describe(ErrToolSchemaRefused) = %q; want %q", got, want)
	}
}
