//go:build unix

package llmprovider

import (
	"errors"
	"os"
	"path/filepath"
)

// syncDir fsyncs a directory, so a rename into it survives a crash.
func syncDir(dir string) (err error) {
	d, err := os.Open(filepath.Clean(dir))
	if err != nil {
		return err
	}
	return errors.Join(d.Sync(), d.Close())
}
