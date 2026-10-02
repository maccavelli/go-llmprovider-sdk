//go:build live_gateways

package auth

import (
	"context"
	"testing"
	"time"
)

// liveCtx bounds one live call, as llmprovider's live tests do.
func liveCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 90*time.Second)
}
