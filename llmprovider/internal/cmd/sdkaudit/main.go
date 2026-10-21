// Command sdkaudit measures a Go module's exported API against the SDK
// practices that docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md
// weighed, and checks rules R50 to R52 of docs/guides/api-standards.md. It
// reads the source only, with go/ast, and changes nothing.
//
// A package is public when no element of its path is "internal". Directories
// named testdata, vendor, tools, scripts or docs, and hidden ones, are not
// read.
//
// Usage:
//
//	go run ./llmprovider/internal/cmd/sdkaudit [-check] [-rules LIST] [-codedoc DIRS] [ROOT]
//
// Without -check it prints the measures as JSON. With -check it prints each
// violation of the rules in LIST (default R50,R51,R52) as "path:line Rnn
// what", one per line, sorted:
//
//   - R50: each public package has an Example function, in a _test.go file,
//     with an "Output:" comment in its body; and each package in DIRS (default
//     llmprovider,llmprovider/providers,wizard) has an indented code block in
//     its package comment.
//   - R51: each exported type of a public package has a doc comment that
//     mentions "concurrent use".
//   - R52: each exported function or method of a public package that returns
//     an error has a doc comment that names an error or says when it fails;
//     a method of a standard interface (MarshalJSON, String, Error, …) is
//     exempt.
//
// ROOT defaults to the working directory.
//
// Exit codes: 0 the report printed, or no violation; 1 a violation; 2 a tool
// error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// writer writes lines, and keeps the first write error, as
// internal/cmd/catalogs's does.
type writer struct {
	w   io.Writer
	err error
}

func (l *writer) printf(format string, a ...any) {
	if l.err == nil {
		_, l.err = fmt.Fprintf(l.w, format, a...)
	}
}

// run is main with its arguments and its output, for the tests. A failed
// write is a tool error.
func run(args []string, stdout, stderr io.Writer) int {
	out, errOut := &writer{w: stdout}, &writer{w: stderr}
	code := runTo(args, out, errOut, stderr)
	if out.err != nil || errOut.err != nil {
		return 2
	}
	return code
}

// runTo is run, writing through out and errOut.
func runTo(args []string, out, errOut *writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("sdkaudit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "check the rules instead of printing the report")
	rules := fs.String("rules", "R50,R51,R52", "the rules -check checks")
	codedoc := fs.String("codedoc", "llmprovider,llmprovider/providers,wizard",
		"the packages whose comment must hold a code block (R50)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 1 {
		errOut.printf("sdkaudit: at most one ROOT\n")
		return 2
	}
	root := "."
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	}
	selected, err := parseRules(*rules)
	if err != nil {
		errOut.printf("sdkaudit: %v\n", err)
		return 2
	}
	m, err := load(root)
	if err != nil {
		errOut.printf("sdkaudit: %v\n", err)
		return 2
	}
	if !*check {
		doc, err := json.MarshalIndent(m.measure(), "", " ")
		if err != nil {
			errOut.printf("sdkaudit: %v\n", err)
			return 2
		}
		out.printf("%s\n", doc)
		return 0
	}
	found := m.check(selected, splitList(*codedoc))
	for _, v := range found {
		out.printf("%s\n", v)
	}
	out.printf("sdkaudit: %d violation(s) of %s\n", len(found), *rules)
	if len(found) > 0 {
		return 1
	}
	return 0
}

// parseRules is the set -rules names; an unknown rule is an error.
func parseRules(list string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, r := range splitList(list) {
		switch r {
		case "R50", "R51", "R52":
			out[r] = true
		default:
			return nil, fmt.Errorf("unknown rule %q: want R50, R51 or R52", r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("-rules names no rule")
	}
	return out, nil
}

// splitList is a comma-separated list's non-empty elements.
func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
