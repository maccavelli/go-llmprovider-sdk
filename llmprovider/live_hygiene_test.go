package llmprovider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// transientSentinels are the classes the live suite skips on (0013-MADR D5;
// liveTransient in live_gateways_test.go).
var transientSentinels = map[string]bool{
	"ErrRateLimited": true, "ErrProviderUnavailable": true, "ErrQuotaExhausted": true, "ErrNotPermitted": true,
}

// liveSkipMark precedes a skip that is narrow on purpose, with its reason.
const liveSkipMark = "live-skip:"

// TestLiveTests_TransientSkipsUseTheHelper (0027-MADR; 0013-MADR D5): a live
// test skips a transient failure through SkipIfTransient, which applies the
// suite's one rule. An if or case that names a transient class and skips is
// written by hand, and fails here unless a "// live-skip: <reason>" comment
// ends on the line before it, as for a test whose subject is that class.
func TestLiveTests_TransientSkipsUseTheHelper(t *testing.T) {
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "live_") && strings.HasSuffix(d.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no live_*_test.go files under llmprovider/")
	}
	var sites []string
	for _, path := range files {
		found, err := handWrittenSkips(path)
		if err != nil {
			t.Fatal(err)
		}
		sites = append(sites, found...)
	}
	if len(sites) > 0 {
		t.Errorf("%d transient skip(s) written by hand; call llmprovider.SkipIfTransient(t, err), the suite's one rule "+
			"(0013-MADR D5), or mark a skip that is narrow on purpose with \"// %s <reason>\" on the line before:\n\t%s",
			len(sites), liveSkipMark, strings.Join(sites, "\n\t"))
	}
}

// handWrittenSkips returns the file:line of each if statement or case clause
// in path whose condition names a transient sentinel and whose body skips,
// with no live-skip mark and reason on the line before it.
func handWrittenSkips(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// marked holds the last line of each comment that carries the mark and
	// a reason; the reason may run on over the comment's following lines.
	marked := map[int]bool{}
	for _, group := range f.Comments {
		for _, c := range group.List {
			reason, ok := strings.CutPrefix(strings.TrimSpace(strings.TrimPrefix(c.Text, "//")), liveSkipMark)
			if ok && strings.TrimSpace(reason) != "" {
				marked[fset.Position(group.End()).Line] = true
			}
		}
	}
	var sites []string
	check := func(at token.Pos, conds []ast.Expr, body []ast.Stmt) {
		line := fset.Position(at).Line
		if namesTransient(conds) && skips(body) && !marked[line-1] {
			sites = append(sites, fset.Position(at).String())
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt:
			check(s.Pos(), []ast.Expr{s.Cond}, s.Body.List)
		case *ast.CaseClause:
			check(s.Pos(), s.List, s.Body)
		}
		return true
	})
	return sites, nil
}

// namesTransient reports whether any of exprs mentions a transient sentinel.
func namesTransient(exprs []ast.Expr) bool {
	found := false
	for _, e := range exprs {
		ast.Inspect(e, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && transientSentinels[id.Name] {
				found = true
			}
			return !found
		})
	}
	return found
}

// skips reports whether stmts call a method Skip or Skipf.
func skips(stmts []ast.Stmt) bool {
	found := false
	for _, s := range stmts {
		ast.Inspect(s, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "Skip" || sel.Sel.Name == "Skipf") {
					found = true
				}
			}
			return !found
		})
	}
	return found
}
