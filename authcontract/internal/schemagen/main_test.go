package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The package's directory and its committed schema, from this one.
const (
	pkgDir    = "../.."
	committed = "../../authcontract.schema.json"
)

// generateTo runs the generator on the package into a temporary file and
// returns what it wrote.
func generateTo(t *testing.T) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "authcontract.schema.json")
	if err := run([]string{"-dir", pkgDir, "-output", out}, io.Discard); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestSchemagenMatchesCommitted_0073: run writes the bytes of the committed
// authcontract.schema.json, so the schema is the types' (MADR D4); and with
// no -output it writes the same bytes to stdout.
func TestSchemagenMatchesCommitted_0073(t *testing.T) {
	want, err := os.ReadFile(committed)
	if err != nil {
		t.Fatal(err)
	}
	got := generateTo(t)
	if !bytes.Equal(got, want) {
		t.Errorf("run wrote %d bytes that differ from the committed %s (%d bytes); run go generate\n%s",
			len(got), committed, len(want), firstDifference(got, want))
	}
	var stdout bytes.Buffer
	if err := run([]string{"-dir", pkgDir}, &stdout); err != nil {
		t.Fatalf("run to stdout: %v", err)
	}
	if !bytes.Equal(stdout.Bytes(), want) {
		t.Error("run with no -output wrote other bytes than the committed schema")
	}
}

// firstDifference shows the first line at which got and want differ.
func firstDifference(got, want []byte) string {
	g, w := strings.Split(string(got), "\n"), strings.Split(string(want), "\n")
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return fmt.Sprintf("line %d: generated %q, committed %q", i+1, g[i], w[i])
		}
	}
	return fmt.Sprintf("generated %d lines, committed %d", len(g), len(w))
}

// TestSchemagenRefuses_0073: the generator refuses what it cannot write
// faithfully, rather than writing a partial schema.
func TestSchemagenRefuses_0073(t *testing.T) {
	write := func(t *testing.T, src string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	cases := []struct {
		name, src string
		args      []string
		want      string
	}{
		{name: "an unknown flag", args: []string{"-planted"}, want: "flag provided but not defined"},
		{name: "an argument", args: []string{"extra"}, want: `unexpected argument "extra"`},
		{name: "no Go file", src: "", want: "no Go file"},
		{name: "a struct not in the list", src: "package p\n\ntype Extra struct{ A string }\n",
			want: "struct type Extra is not in schemagen's list"},
		{name: "an int type", src: "package p\n\ntype Count int\n", want: "only string types are enums"},
		{name: "a func type", src: "package p\n\ntype F func()\n", want: "not a struct or a string type"},
		{name: "a constant not a string literal", src: "package p\n\ntype K string\n\nconst A K = \"a\" + \"b\"\n",
			want: "A's value is not a string literal"},
		{name: "a constant with no value", src: "package p\n\ntype K string\n\nconst (\n\tA K = \"a\"\n\tB\n)\n",
			want: "B has no value of its own"},
		{name: "a constant typed by a conversion", src: "package p\n\ntype K string\n\nconst A = K(\"a\")\n",
			want: "A is a conversion to K"},
		{name: "a constant of another type", src: "package p\n\nconst A Other = \"a\"\n",
			want: "constants of Other, which is not a string type of the package"},
		{name: "a parse error", src: "package p\n\ntype\n", want: "expected"},
		{name: "a struct in the list missing from the source", src: "package p\n\ntype K string\n",
			want: "struct type Method is in schemagen's list, not in the source"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := c.args
			if args == nil {
				dir := t.TempDir()
				if c.src != "" {
					dir = write(t, c.src)
				}
				args = []string{"-dir", dir}
			}
			err := run(args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("run(%q) = %v, want an error holding %q", args, err, c.want)
			}
		})
	}
	if err := run([]string{"-dir", pkgDir, "-output", filepath.Join(t.TempDir(), "no", "such", "dir", "x.json")}, io.Discard); err == nil {
		t.Error("run into a directory that does not exist: no error")
	}
}
