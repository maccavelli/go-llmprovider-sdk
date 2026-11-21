package llmprovider_test

// 0084-PLAN Phase 1a's key-shape scan (0084-MADR ST79, ST84, ST85). It runs
// beside the recorder, in the same environment, so the call allowlist checks
// it too (TestToolSchemaRecord_CallAllowlist_0084): it prints a hit as the
// file's base name, a line number and a constant label, never a line's text.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
)

// The labels a hit carries.
const (
	scanLabelKey  = "key shape"
	scanLabelLine = "not allowlisted"
)

// scanHit is one finding: where, and which kind. It holds no text.
type scanHit struct {
	file  string
	line  int
	label string
}

// scanAllowed are the shapes a line of the run's output file may take
// (0084-MADR ST79): go test -v's framing, the case line t.Logf writes with
// its file name and line, and the recorder's fixed failure and skip texts,
// each with at most a case id.
var scanAllowed = []*regexp.Regexp{
	regexp.MustCompile(`^=== (RUN|PAUSE|CONT|NAME)\s+\S+$`),
	regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP): \S+ \([0-9.]+s\)$`),
	regexp.MustCompile(`^(PASS|FAIL)$`),
	regexp.MustCompile(`^(ok|FAIL)\s+\S+\s+[0-9.]+s( \(cached\))?$`),
	regexp.MustCompile(`^\s+[A-Za-z0-9_]+\.go:[0-9]+: \S+ \S+ \S+: status [0-9]+, (accepted|refused|other|unreachable|skipped)$`),
	regexp.MustCompile(`^\s+[A-Za-z0-9_]+\.go:[0-9]+: set LLMPROVIDER_LIVE_0084 to run 0084-PLAN Phase 1's recorder$`),
	regexp.MustCompile(`^\s+[A-Za-z0-9_]+\.go:[0-9]+: LLMPROVIDER_LIVE_0084_OUT unset: name a directory outside the module for the records$`),
	regexp.MustCompile(`^\s+[A-Za-z0-9_]+\.go:[0-9]+: LLMPROVIDER_LIVE_0084_OUT is inside the module: name a directory outside it$`),
	regexp.MustCompile(`^\s+[A-Za-z0-9_]+\.go:[0-9]+: \S+: record not written$`),
}

// scanLineAllowed reports whether a line of the output file takes one of the
// allowed shapes.
func scanLineAllowed(line string) bool {
	for _, re := range scanAllowed {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// scanToolPath scans root: a file is the run's output file, whose lines must
// also take an allowed shape; a directory is a set of records, scanned for key
// shapes alone. A line is a key shape when redact.Redact changes it. ok is
// false when the walk or a read failed, which the caller fails on with a
// fixed text.
func scanToolPath(root string) (hits []scanHit, ok bool) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		output := path == root
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == len(lines)-1 && line == "" {
				continue
			}
			if string(redact.Redact([]byte(line))) != line {
				hits = append(hits, scanHit{file: filepath.Base(path), line: i + 1, label: scanLabelKey})
			}
			if output && !scanLineAllowed(line) {
				hits = append(hits, scanHit{file: filepath.Base(path), line: i + 1, label: scanLabelLine})
			}
		}
		return nil
	})
	return hits, err == nil
}

// reportToolScan prints each hit as "<file>:<line>: <label>".
func reportToolScan(emit emitFunc, hits []scanHit) {
	for _, h := range hits {
		emit("%s:%d: %s", h.file, h.line, h.label)
	}
}

// TestToolSchemaScan_0084 scans the file or directory
// LLMPROVIDER_LIVE_0084_SCAN names, and fails on any hit. It skips when the
// variable is unset, so the default run does not scan; Phase 1b and 1c run it
// on the run's output file and on the records. Its proof is
// TestToolSchemaScan_Plant_0084.
func TestToolSchemaScan_0084(t *testing.T) {
	root := os.Getenv("LLMPROVIDER_LIVE_0084_SCAN")
	if root == "" {
		t.Skip("set LLMPROVIDER_LIVE_0084_SCAN to a file or directory to scan")
	}
	hits, ok := scanToolPath(root)
	if !ok {
		t.Fatal("the scan could not read its target")
	}
	reportToolScan(t.Logf, hits)
	if len(hits) > 0 {
		t.Fatal("a key shape or a line outside the allowlist was found")
	}
}
