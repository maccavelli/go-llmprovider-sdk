package main

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

// report is what -report prints: the measures 0048-MADR's table holds.
type report struct {
	Packages            []string          `json:"public_packages"`
	ExportedFuncs       int               `json:"exported_funcs_and_methods"`
	ExportedTypes       int               `json:"exported_types_public"`
	ReturnsInterface    []site            `json:"returns_module_interface"`
	CtxNotFirst         []site            `json:"ctx_not_first"`
	CtxInStruct         []site            `json:"ctx_in_struct"`
	MissingDoc          []site            `json:"exported_missing_doc"`
	DocNotName          []site            `json:"doc_not_starting_with_name"`
	Examples            map[string]int    `json:"examples_per_package"`
	PkgVars             map[string][]site `json:"package_vars_by_class"`
	Panics              []site            `json:"panics"`
	ErrorfNoW           []site            `json:"errorf_with_error_arg_no_w"`
	ConcurrencyNoted    []string          `json:"types_documenting_concurrency"`
	Deprecated          []site            `json:"deprecated"`
	TimeNow             map[string]int    `json:"time_now_calls_per_package"`
	HTTPNoCtx           []site            `json:"http_calls_without_ctx"`
	ReturnsError        int               `json:"exported_returning_error"`
	ReturnsErrorNamed   int               `json:"exported_returning_error_doc_names_errors"`
	ReturnsErrorUnnamed []site            `json:"exported_returning_error_doc_silent"`
}

// classMap is the class of a package-level map.
const classMap = "map"

// noCtxHTTP are the net/http functions that make a request with no context.
var noCtxHTTP = map[string]bool{"Get": true, "Post": true, "Head": true, "PostForm": true, "NewRequest": true}

// measure walks every file and returns the report.
func (m *module) measure() *report {
	r := &report{Examples: map[string]int{}, PkgVars: map[string][]site{}, TimeNow: map[string]int{}}
	packages := map[string]bool{}
	for _, fl := range m.files {
		if fl.test {
			if public(fl.dir) {
				r.Examples[fl.dir] += countExamples(fl.f)
			}
			continue
		}
		if public(fl.dir) {
			packages[fl.dir] = true
		}
		m.measureCalls(r, fl)
		for _, decl := range fl.f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if public(fl.dir) {
					m.measureFunc(r, fl, d)
				}
			case *ast.GenDecl:
				m.measureGen(r, fl, d)
			}
		}
	}
	for dir, n := range r.Examples {
		if n == 0 {
			delete(r.Examples, dir)
		}
	}
	for d := range packages {
		r.Packages = append(r.Packages, d)
	}
	sort.Strings(r.Packages)
	return r
}

func countExamples(f *ast.File) int {
	n := 0
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && strings.HasPrefix(fd.Name.Name, "Example") {
			n++
		}
	}
	return n
}

// isContext reports whether e is context.Context.
func isContext(e ast.Expr) bool {
	s, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := s.X.(*ast.Ident)
	return ok && x.Name == "context" && s.Sel.Name == "Context"
}

// pkgCall returns the package and name of a call such as fmt.Errorf.
func pkgCall(c *ast.CallExpr) (string, string, bool) {
	se, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	x, ok := se.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	return x.Name, se.Sel.Name, true
}

// measureCalls counts panics, clock reads, context-free requests and
// fmt.Errorf calls that drop an error, and contexts held in structs.
func (m *module) measureCalls(r *report, fl file) {
	ast.Inspect(fl.f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "panic" {
				r.Panics = append(r.Panics, site{m.pos(x.Pos()), fl.f.Name.Name})
			}
			pkg, name, ok := pkgCall(x)
			switch {
			case !ok:
			case pkg == "time" && name == "Now":
				r.TimeNow[fl.dir]++
			case pkg == "http" && noCtxHTTP[name]:
				r.HTTPNoCtx = append(r.HTTPNoCtx, site{m.pos(x.Pos()), "http." + name})
			case pkg == "fmt" && name == "Errorf" && errorfDropsError(x):
				r.ErrorfNoW = append(r.ErrorfNoW, site{m.pos(x.Pos()), fl.f.Name.Name})
			}
		case *ast.StructType:
			for _, fld := range x.Fields.List {
				if isContext(fld.Type) {
					r.CtxInStruct = append(r.CtxInStruct, site{m.pos(fld.Pos()), fl.f.Name.Name})
				}
			}
		}
		return true
	})
}

// errorfDropsError reports whether a fmt.Errorf call is given an error (an
// argument named err, or ending Err) and its format has no %w.
func errorfDropsError(c *ast.CallExpr) bool {
	if len(c.Args) < 2 {
		return false
	}
	lit, ok := c.Args[0].(*ast.BasicLit)
	if !ok || strings.Contains(lit.Value, "%w") {
		return false
	}
	for _, a := range c.Args[1:] {
		if id, ok := a.(*ast.Ident); ok && (id.Name == "err" || strings.HasSuffix(id.Name, "Err")) {
			return true
		}
	}
	return false
}

func (m *module) measureFunc(r *report, fl file, fd *ast.FuncDecl) {
	name, ok := exportedName(fd)
	if !ok {
		return
	}
	r.ExportedFuncs++
	doc := docText(fd.Doc)
	switch {
	case fd.Doc == nil:
		r.MissingDoc = append(r.MissingDoc, site{m.pos(fd.Pos()), name})
	case !strings.HasPrefix(strings.TrimSpace(doc), fd.Name.Name):
		r.DocNotName = append(r.DocNotName, site{m.pos(fd.Pos()), name})
	}
	if strings.Contains(doc, "Deprecated:") {
		r.Deprecated = append(r.Deprecated, site{m.pos(fd.Pos()), name})
	}
	for i, p := range fd.Type.Params.List {
		if isContext(p.Type) && i != 0 {
			r.CtxNotFirst = append(r.CtxNotFirst, site{m.pos(fd.Pos()), name})
		}
	}
	if returnsError(fd) {
		r.ReturnsError++
		if errDoc.MatchString(doc) {
			r.ReturnsErrorNamed++
		} else {
			r.ReturnsErrorUnnamed = append(r.ReturnsErrorUnnamed, site{m.pos(fd.Pos()), name})
		}
	}
	if fd.Recv == nil && fd.Type.Results != nil {
		for _, res := range fd.Type.Results.List {
			if n, ok := m.moduleInterface(fl.f.Name.Name, res.Type); ok {
				r.ReturnsInterface = append(r.ReturnsInterface, site{m.pos(fd.Pos()), name + " -> " + n})
			}
		}
	}
}

// moduleInterface reports whether e names one of the module's exported
// interfaces, and which.
func (m *module) moduleInterface(pkg string, e ast.Expr) (string, bool) {
	switch t := e.(type) {
	case *ast.Ident:
		return pkg + "." + t.Name, m.ifaces[pkg+"."+t.Name]
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name, m.ifaces[x.Name+"."+t.Sel.Name]
		}
	}
	return "", false
}

func (m *module) measureGen(r *report, fl file, g *ast.GenDecl) {
	for _, s := range g.Specs {
		switch sp := s.(type) {
		case *ast.TypeSpec:
			if !sp.Name.IsExported() || !public(fl.dir) {
				continue
			}
			r.ExportedTypes++
			doc := typeDoc(g, sp)
			if doc == nil {
				r.MissingDoc = append(r.MissingDoc, site{m.pos(sp.Pos()), fl.f.Name.Name + "." + sp.Name.Name})
				continue
			}
			t := strings.ToLower(doc.Text())
			if strings.Contains(t, "concurrent") || strings.Contains(t, "goroutine") {
				r.ConcurrencyNoted = append(r.ConcurrencyNoted, fl.dir+"."+sp.Name.Name)
			}
			if strings.Contains(doc.Text(), "Deprecated:") {
				r.Deprecated = append(r.Deprecated, site{m.pos(sp.Pos()), fl.f.Name.Name + "." + sp.Name.Name})
			}
		case *ast.ValueSpec:
			if g.Tok == token.VAR {
				m.measureVars(r, fl, sp)
			}
		}
	}
}

// measureVars classifies each package-level variable by what it holds.
func (m *module) measureVars(r *report, fl file, sp *ast.ValueSpec) {
	for i, nm := range sp.Names {
		if nm.Name == "_" {
			continue
		}
		var val ast.Expr
		if i < len(sp.Values) {
			val = sp.Values[i]
		}
		vis := "unexported"
		if nm.IsExported() && public(fl.dir) {
			vis = "exported"
		}
		key := vis + " " + varClass(sp.Type, val)
		r.PkgVars[key] = append(r.PkgVars[key], site{m.pos(nm.Pos()), fl.f.Name.Name + "." + nm.Name})
	}
}

// varClass names what a package-level variable of type typ, set to val,
// holds.
func varClass(typ, val ast.Expr) string {
	switch v := val.(type) {
	case *ast.CallExpr:
		if pkg, name, ok := pkgCall(v); ok {
			switch {
			case pkg == "errors" && name == "New", pkg == "fmt" && name == "Errorf":
				return "error sentinel"
			case pkg == "regexp" && strings.HasPrefix(name, "MustCompile"):
				return "compiled regexp"
			}
		}
		if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "make" && len(v.Args) > 0 {
			if _, isMap := v.Args[0].(*ast.MapType); isMap {
				return classMap
			}
		}
	case *ast.CompositeLit:
		switch v.Type.(type) {
		case *ast.MapType:
			return classMap
		case *ast.ArrayType:
			return "slice"
		}
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			if cl, ok := v.X.(*ast.CompositeLit); ok {
				if id, ok := cl.Type.(*ast.Ident); ok && strings.HasSuffix(strings.ToLower(id.Name), "error") {
					return "error sentinel"
				}
			}
		}
	case *ast.FuncLit, *ast.SelectorExpr, *ast.Ident:
		return "function variable (seam)"
	}
	if se, ok := typ.(*ast.SelectorExpr); ok {
		if x, ok := se.X.(*ast.Ident); ok && x.Name == "sync" {
			return "sync primitive"
		}
	}
	if _, ok := typ.(*ast.MapType); ok {
		return classMap
	}
	return "other"
}
