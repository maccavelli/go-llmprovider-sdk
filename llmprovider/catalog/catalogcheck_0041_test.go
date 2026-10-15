package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCatalogCheck0041_Sixes prints, for tools/catalogcheck.py, each open
// catalog's six under each profile, from the snapshot
// LLMPROVIDER_SNAPSHOT_DIR names (0041-MADR D7). A provider whose listing
// the snapshot lacks is printed as missing. Each line reads
// "SIX<TAB>provider<TAB>profile<TAB>id,id,…". It is skipped without the
// variable, and never fails on what it prints.
func TestCatalogCheck0041_Sixes(t *testing.T) {
	dir := os.Getenv("LLMPROVIDER_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("LLMPROVIDER_SNAPSHOT_DIR unset: tools/catalogcheck.py sets it")
	}
	providers := make([]string, 0, len(snapshotFile))
	for p := range snapshotFile {
		providers = append(providers, string(p))
	}
	slices.Sort(providers)
	profiles := []struct {
		name    string
		profile Profile
	}{{"utility", ProfileUtility}, {"capable", ProfileCapable}, {"agent", ProfileAgent}}
	for _, name := range providers {
		for provider, file := range snapshotFile {
			if string(provider) != name {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, file)); err != nil {
				fmt.Printf("SIX\t%s\t-\tmissing\n", provider)
				continue
			}
			for _, p := range profiles {
				six := replaySnapshot(t, dir, provider, p.profile, WithLimit(MaxListed)).Recommended
				fmt.Printf("SIX\t%s\t%s\t%s\n", provider, p.name, strings.Join(six, ","))
			}
		}
	}
}
