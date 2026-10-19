package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func runCmd(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// TestReportCountsEachPlantedDefectOnce_0048: the planted module holds one
// of each defect the report counts (0048-PLAN Phase 1 step 3).
func TestReportCountsEachPlantedDefectOnce_0048(t *testing.T) {
	code, out, errOut := runCmd(t, "testdata/bad")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	var r report
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, out)
	}
	for name, got := range map[string]int{
		"returns_module_interface":                  len(r.ReturnsInterface),
		"ctx_not_first":                             len(r.CtxNotFirst),
		"ctx_in_struct":                             len(r.CtxInStruct),
		"exported_missing_doc":                      len(r.MissingDoc),
		"doc_not_starting_with_name":                len(r.DocNotName),
		"panics":                                    len(r.Panics),
		"errorf_with_error_arg_no_w":                len(r.ErrorfNoW),
		"deprecated":                                len(r.Deprecated),
		"http_calls_without_ctx":                    len(r.HTTPNoCtx),
		"exported_returning_error_doc_names_errors": r.ReturnsErrorNamed,
		"types_documenting_concurrency":             len(r.ConcurrencyNoted),
	} {
		if got != 1 {
			t.Errorf("%s = %d, want 1", name, got)
		}
	}
	if r.ReturnsError != 2 || len(r.ReturnsErrorUnnamed) != 1 {
		t.Errorf("returning error: %d, unnamed %d; want 2 and 1", r.ReturnsError, len(r.ReturnsErrorUnnamed))
	}
	if !reflect.DeepEqual(r.Examples, map[string]int{"pub": 1}) || !reflect.DeepEqual(r.TimeNow, map[string]int{"pub": 1}) {
		t.Errorf("examples %v, time.Now %v; want pub:1 each", r.Examples, r.TimeNow)
	}
	if !reflect.DeepEqual(r.Packages, []string{"pub"}) || r.ExportedTypes != 2 {
		t.Errorf("packages %v, exported types %d; want [pub] and 2", r.Packages, r.ExportedTypes)
	}
	classes := map[string][]string{}
	for class, sites := range r.PkgVars {
		for _, s := range sites {
			classes[class] = append(classes[class], s.What)
		}
		sort.Strings(classes[class])
	}
	want := map[string][]string{
		"exported error sentinel":             {"pub.ErrX"},
		"exported map":                        {"pub.Shared"},
		"unexported function variable (seam)": {"pub.now"},
		"unexported compiled regexp":          {"pub.pattern"},
		"unexported sync primitive":           {"pub.mu"},
		"unexported slice":                    {"pub.names"},
		"unexported other":                    {"pub.other"},
		"unexported error sentinel":           {"pub.errPlanted"},
	}
	if !reflect.DeepEqual(classes, want) {
		t.Errorf("variable classes\n got %v\nwant %v", classes, want)
	}
}

// TestCheckNamesEachViolationOnce_0048: -check names each of R50 to R52's
// planted violations once, and exits 1.
func TestCheckNamesEachViolationOnce_0048(t *testing.T) {
	code, out, _ := runCmd(t, "-check", "-codedoc", "pub", "testdata/bad")
	if code != 1 {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	want := []string{
		"pub/pub.go:15 R51 type Doer does not say whether it is safe for concurrent use",
		"pub/pub.go:57 R52 Quiet returns an error its doc does not name",
		"pub:1 R50 package pub has no Example function with an Output comment",
		"pub:1 R50 package pub's comment has no indented code block",
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	got, last := lines[:len(lines)-1], lines[len(lines)-1]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("violations\n got %q\nwant %q", got, want)
	}
	if last != "sdkaudit: 4 violation(s) of R50,R51,R52" {
		t.Errorf("summary %q", last)
	}
}

// TestCheckPassesTheGoodModule_0048: a module that meets the rules passes;
// an internal package's undocumented exports do not count.
func TestCheckPassesTheGoodModule_0048(t *testing.T) {
	code, out, _ := runCmd(t, "-check", "-codedoc", "pub", "testdata/good")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if !strings.Contains(out, "sdkaudit: 0 violation(s)") {
		t.Errorf("output %q", out)
	}
}

// TestCheckRunsOnlyTheRulesNamed_0048: -rules selects.
func TestCheckRunsOnlyTheRulesNamed_0048(t *testing.T) {
	for rule, want := range map[string]int{"R50": 2, "R51": 1, "R52": 1} {
		_, out, _ := runCmd(t, "-check", "-rules", rule, "-codedoc", "pub", "testdata/bad")
		if n := strings.Count(out, " "+rule+" "); n != want {
			t.Errorf("-rules %s: %d violation(s), want %d:\n%s", rule, n, want, out)
		}
		for _, other := range []string{"R50", "R51", "R52"} {
			if other != rule && strings.Contains(out, " "+other+" ") {
				t.Errorf("-rules %s reported %s:\n%s", rule, other, out)
			}
		}
	}
}

// TestToolErrorsExitTwo_0048: bad arguments and unreadable source are tool
// errors, exit 2. The unparsable file is written at run time, so no file of
// the tree fails gofmt.
func TestToolErrorsExitTwo_0048(t *testing.T) {
	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, "broken.go"), []byte("package pub\n\nfunc Broken( {\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"unknown rule":    {"-check", "-rules", "R99", "testdata/good"},
		"no rule":         {"-check", "-rules", ",", "testdata/good"},
		"two roots":       {"testdata/good", "testdata/bad"},
		"unknown flag":    {"-nope"},
		"missing root":    {"testdata/absent"},
		"unparsable file": {broken},
	} {
		if code, _, _ := runCmd(t, args...); code != 2 {
			t.Errorf("%s: exit %d, want 2", name, code)
		}
	}
}

// failingWriter refuses every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("refused") }

// TestFailedWriteExitsTwo_0048: output that cannot be written is a tool
// error, in both modes.
func TestFailedWriteExitsTwo_0048(t *testing.T) {
	for _, args := range [][]string{{"testdata/good"}, {"-check", "-codedoc", "pub", "testdata/good"}} {
		if code := run(args, failingWriter{}, failingWriter{}); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}
