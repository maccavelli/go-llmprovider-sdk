package ambientcheck

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// planted reads ambient state each way the check must see (0020-MADR F25).
const planted = `package p

import (
	"log"
	"net/http"
	"os"
)

func f() {
	_ = http.DefaultClient
	_ = http.DefaultTransport
	_, _ = os.UserHomeDir()
	_ = os.ExpandEnv("$HOME")
	log.Printf("x")
	_ = os.Getenv("CONTROL")
}
`

// TestFileFindings_Planted (0020-MADR F25): each planted read is a finding.
func TestFileFindings_Planted(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "planted.go", planted, 0)
	if err != nil {
		t.Fatal(err)
	}
	findings := strings.Join(fileFindings(fset, "llmprovider/planted.go", file), "\n")
	for _, want := range []string{"http.DefaultClient", "http.DefaultTransport", "os.UserHomeDir", "os.ExpandEnv",
		"log.Printf", "os.Getenv"} {
		if !strings.Contains(findings, want) {
			t.Errorf("no finding for %s; findings:\n%s", want, findings)
		}
	}
}
