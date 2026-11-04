package auth

import (
	"testing"
	"time"
)

// TestLockWait_OutlastsLongestRefresh_0068 (0068-MADR D3): a refresh holds
// the store's lock across every attempt and the backoff between them, so a
// waiter that gives up sooner fails beside a holder still within its bounds.
// Not parallel: other tests shorten oauthRefreshAttemptTimeout.
func TestLockWait_OutlastsLongestRefresh_0068(t *testing.T) {
	hold := time.Duration(oauthRefreshAttempts) * oauthRefreshAttemptTimeout
	for attempt := 1; attempt < oauthRefreshAttempts; attempt++ {
		hold += oauthRefreshBackoff << (attempt - 1)
	}
	if lockWait <= hold {
		t.Errorf("lockWait = %v; want more than the longest refresh hold, %v", lockWait, hold)
	}
}
