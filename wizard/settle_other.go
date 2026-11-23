//go:build !unix && !windows

package wizard

import (
	"os"
	"time"
)

// pendingWithin is never true here: with no readiness check, a hidden entry
// does not wait (0087-MADR A7).
func pendingWithin(*os.File, time.Duration) (bool, error) { return false, nil }
