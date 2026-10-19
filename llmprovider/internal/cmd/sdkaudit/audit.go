package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// skipDirs are the directories load does not read.
var skipDirs = map[string]bool{"testdata": true, "vendor": true, "tools": true, "scripts": true, "docs": true}

// errDoc matches a doc comment that names an error or says when a function
// fails (R52).
var errDoc = regexp.MustCompile(`\bErr[A-Z]\w*|\berrors?\b|\bfail|\brefus|\bAPIError\b`)

// exemptMethods are the methods of standard interfaces, whose contract is
// the interface's: R52 does not ask their docs to name errors.
var exemptMethods = map[string]bool{
	"MarshalJSON": true, "UnmarshalJSON": true, "MarshalText": true, "UnmarshalText": true,
	"Error": true, "String": true, "GoString": true, "LogValue": true, "Format": true,
}

// file is one parsed Go file.
type file struct {
	dir  string // the package directory, relative to the root, "." for the root
	f    *ast.File
	test bool
}

// module is a parsed module tree.
type module struct {
	root   string
	fset   *token.FileSet
	files  []file
	ifaces map[string]bool // "pkg.Name" of each exported interface
}

// site is one place in the source and what is there.
type site struct {
	Pos  string `json:"pos"`
	What string `json:"what"`
}

// load parses every Go file under root, test files included.
func load(root string) (*module, error) {
	m := &module{root: root, fset: token.NewFileSet(), ifaces: map[string]bool{}}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(m.fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		m.files = append(m.files, file{dir: filepath.ToSlash(rel), f: f, test: strings.HasSuffix(path, "_test.go")})
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, fl := range m.files {
		if !fl.test {
			m.collectInterfaces(fl)
		}
	}
	return m, nil
}

func (m *module) collectInterfaces(fl file) {
	for _, decl := range fl.f.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.TYPE {
			continue
		}
		for _, s := range g.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, isIface := ts.Type.(*ast.InterfaceType); isIface && ts.Name.IsExported() {
				m.ifaces[fl.f.Name.Name+"."+ts.Name.Name] = true
			}
		}
	}
}

// public reports whether the package in dir is outside every internal/.
func public(dir string) bool {
	for _, e := range strings.Split(dir, "/") {
		if e == "internal" {
			return false
		}
	}
	return true
}

// pos renders p as a path relative to the root and a line.
func (m *module) pos(p token.Pos) string {
	pp := m.fset.Position(p)
	rel, err := filepath.Rel(m.root, pp.Filename)
	if err != nil {
		rel = pp.Filename
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(rel), pp.Line)
}

// exportedName is a function's or method's name as the doc reads it
// ("Type.Method"), and whether it is part of the exported API: an exported
// function, or an exported method of an exported type.
func exportedName(fd *ast.FuncDecl) (string, bool) {
	if !fd.Name.IsExported() {
		return "", false
	}
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name, true
	}
	t := fd.Recv.List[0].Type
	if st, ok := t.(*ast.StarExpr); ok {
		t = st.X
	}
	if ix, ok := t.(*ast.IndexExpr); ok {
		t = ix.X
	}
	id, ok := t.(*ast.Ident)
	if !ok || !id.IsExported() {
		return "", false
	}
	return id.Name + "." + fd.Name.Name, true
}

// returnsError reports whether one of fd's results is error.
func returnsError(fd *ast.FuncDecl) bool {
	if fd.Type.Results == nil {
		return false
	}
	for _, r := range fd.Type.Results.List {
		if id, ok := r.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}
	return false
}

// typeDoc is a type's doc comment: its spec's, else its declaration's.
func typeDoc(g *ast.GenDecl, ts *ast.TypeSpec) *ast.CommentGroup {
	if ts.Doc != nil {
		return ts.Doc
	}
	return g.Doc
}

// docText is a comment group's text, "" for none.
func docText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return cg.Text()
}

// hasCodeBlock reports whether a package comment holds an indented line.
func hasCodeBlock(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		for _, line := range strings.Split(c.Text, "\n") {
			if strings.HasPrefix(line, "//\t") {
				return true
			}
		}
	}
	return false
}

// hasOutput reports whether a comment inside fd's body is an Output comment.
func hasOutput(f *ast.File, fd *ast.FuncDecl) bool {
	if fd.Body == nil {
		return false
	}
	for _, cg := range f.Comments {
		if cg.Pos() < fd.Body.Lbrace || cg.End() > fd.Body.Rbrace {
			continue
		}
		t := strings.TrimSpace(cg.Text())
		if strings.HasPrefix(t, "Output:") || strings.HasPrefix(t, "Unordered output:") {
			return true
		}
	}
	return false
}

// check returns each violation of the selected rules, sorted.
func (m *module) check(rules map[string]bool, codedoc []string) []string {
	var out []string
	if rules["R50"] {
		out = append(out, m.checkExamples(codedoc)...)
	}
	for _, fl := range m.files {
		if fl.test || !public(fl.dir) {
			continue
		}
		for _, decl := range fl.f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if rules["R51"] {
					out = append(out, m.checkTypes(d)...)
				}
			case *ast.FuncDecl:
				if rules["R52"] {
					out = append(out, m.checkErrors(d)...)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func (m *module) checkExamples(codedoc []string) []string {
	packages, examples := map[string]bool{}, map[string]bool{}
	docs := map[string]bool{}
	for _, fl := range m.files {
		if !public(fl.dir) {
			continue
		}
		if !fl.test {
			packages[fl.dir] = true
			if hasCodeBlock(fl.f.Doc) {
				docs[fl.dir] = true
			}
			continue
		}
		for _, decl := range fl.f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && fd.Recv == nil && strings.HasPrefix(fd.Name.Name, "Example") && hasOutput(fl.f, fd) {
				examples[fl.dir] = true
			}
		}
	}
	var out []string
	for dir := range packages {
		if !examples[dir] {
			out = append(out, fmt.Sprintf("%s:1 R50 package %s has no Example function with an Output comment", dir, dir))
		}
	}
	for _, dir := range codedoc {
		if packages[dir] && !docs[dir] {
			out = append(out, fmt.Sprintf("%s:1 R50 package %s's comment has no indented code block", dir, dir))
		}
	}
	return out
}

func (m *module) checkTypes(g *ast.GenDecl) []string {
	if g.Tok != token.TYPE {
		return nil
	}
	var out []string
	for _, s := range g.Specs {
		ts, ok := s.(*ast.TypeSpec)
		if !ok || !ts.Name.IsExported() {
			continue
		}
		if !strings.Contains(strings.ToLower(docText(typeDoc(g, ts))), "concurrent use") {
			out = append(out, fmt.Sprintf("%s R51 type %s does not say whether it is safe for concurrent use",
				m.pos(ts.Pos()), ts.Name.Name))
		}
	}
	return out
}

func (m *module) checkErrors(fd *ast.FuncDecl) []string {
	name, ok := exportedName(fd)
	if !ok || !returnsError(fd) || exemptMethods[fd.Name.Name] {
		return nil
	}
	if errDoc.MatchString(docText(fd.Doc)) {
		return nil
	}
	return []string{fmt.Sprintf("%s R52 %s returns an error its doc does not name", m.pos(fd.Pos()), name)}
}
