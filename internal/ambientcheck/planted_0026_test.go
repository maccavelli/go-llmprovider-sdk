package ambientcheck

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// plantedDirs reads the user's directories and the environment the ways
// 0026-MADR F59 names.
const plantedDirs = `package p

import (
	"os"
	"os/user"
	"syscall"
)

func f() {
	_, _ = os.UserConfigDir()
	_, _ = os.UserCacheDir()
	_, _ = user.Current()
	_, _ = syscall.Getenv("CONTROL")
}
`

// TestFileFindings_PlantedDirs (0026-MADR F59): the user's config and cache
// directories, the current user, and syscall's environment are ambient reads.
func TestFileFindings_PlantedDirs(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "planted.go", plantedDirs, 0)
	if err != nil {
		t.Fatal(err)
	}
	findings := strings.Join(fileFindings(fset, "llmprovider/planted.go", file), "\n")
	for _, want := range []string{"os.UserConfigDir", "os.UserCacheDir", "user.Current", "syscall.Getenv"} {
		if !strings.Contains(findings, want) {
			t.Errorf("no finding for %s; findings:\n%s", want, findings)
		}
	}
}
