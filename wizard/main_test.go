package wizard

import (
	"os"
	"testing"
)

// TestMain keeps unit tests off the network: the models metadata fetch (MADR
// 0009 §2) is off unless a test re-enables it with t.Setenv and a fixture URL.
func TestMain(m *testing.M) {
	if err := os.Setenv("LLMPROVIDER_DISABLE_MODELS_METADATA", "1"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
