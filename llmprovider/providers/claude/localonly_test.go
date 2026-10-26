//go:build !live_gateways

package claude

import (
	"os"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/localonly"
)

// TestMain runs the package's tests behind localonly's guard: a test that
// sends a request off the machine fails the run, naming the host (0065-MADR
// D2). The live tests, built with live_gateways, reach real services on
// purpose and run without it.
func TestMain(m *testing.M) { os.Exit(localonly.Main(m)) }
