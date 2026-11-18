//go:build !live_gateways

package auth

import (
	"os"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/localonly"
)

// TestMain runs the package's tests behind localonly's guard: a test that
// sends a request off the machine through net/http's proxy settings fails the
// run, naming the host (0065-MADR D2; 0082-PLAN Phase 1, the second review's
// B-1: two planted breaks of the guard's own fixtures once sent refreshes to
// OpenAI's real token endpoint). The live tests, built with live_gateways,
// reach real services on purpose and run without it.
func TestMain(m *testing.M) { os.Exit(localonly.Main(m)) }
