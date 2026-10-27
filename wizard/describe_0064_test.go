package wizard

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDescribe_0064: Anthropic's workspace refusals are worded by their own
// class, not as an invalid request (0064-MADR D3; 0064-PLAN Phase 2).
func TestDescribe_0064(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   error
		want   string
	}{
		{http.StatusBadRequest, llmprovider.ErrWorkspaceRequired, "workspace required, HTTP 400"},
		{http.StatusBadRequest, llmprovider.ErrWorkspaceNotFound, "workspace not found, HTTP 400"},
		{http.StatusNotFound, llmprovider.ErrWorkspaceNotFound, "workspace not found, HTTP 404"},
	} {
		err := fmt.Errorf("list: %w", &llmprovider.APIError{Provider: "claude", Status: tc.status, Kind: tc.kind, Message: maskedEcho0056})
		if got := describe(err); got != tc.want {
			t.Errorf("describe(%v, HTTP %d) = %q; want %q", tc.kind, tc.status, got, tc.want)
		}
	}
}
