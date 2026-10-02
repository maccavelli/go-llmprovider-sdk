package ambientcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// moduleRoot is this module's root, from this package's directory.
const moduleRoot = "../.."

// globalLogging names slog's package-level functions that write to, or
// change, the process-global logger.
var globalLogging = map[string]bool{
	"Debug": true, "Info": true, "Warn": true, "Error": true, "Log": true, "LogAttrs": true,
	"DebugContext": true, "InfoContext": true, "WarnContext": true, "ErrorContext": true,
	"Default": true, "SetDefault": true, "SetLogLoggerLevel": true,
}

// proxyHome is the one package allowed http.ProxyFromEnvironment: the default
// transport (0016-MADR D8).
const proxyHome = "llmprovider/internal/transport"

// TestNoAmbientState (0015-MADR D9): library code reads the environment only
// inside an exported function whose name ends in FromEnv, takes the proxy from
// the environment only in the default transport, and never uses the global
// logger.
func TestNoAmbientState(t *testing.T) {
	findings, err := scan(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Error(f)
	}
}

// scan walks root's non-test Go sources and returns each ambient read.
func scan(root string) ([]string, error) {
	var findings []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		findings = append(findings, fileFindings(fset, filepath.ToSlash(rel), file)...)
		return nil
	})
	return findings, err
}

// fileFindings is one file's ambient reads.
func fileFindings(fset *token.FileSet, rel string, file *ast.File) []string {
	names := importNames(file)
	var findings []string
	for _, decl := range file.Decls {
		allowedEnv := false
		if fn, ok := decl.(*ast.FuncDecl); ok {
			allowedEnv = fn.Name.IsExported() && strings.HasSuffix(fn.Name.Name, "FromEnv")
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			where := fset.Position(sel.Pos())
			at := rel + ":" + strconv.Itoa(where.Line)
			switch names[pkg.Name] {
			case "os":
				if (sel.Sel.Name == "Getenv" || sel.Sel.Name == "LookupEnv" || sel.Sel.Name == "Environ") && !allowedEnv {
					findings = append(findings, at+": os."+sel.Sel.Name+" outside an exported ...FromEnv function")
				}
			case "net/http":
				if sel.Sel.Name == "ProxyFromEnvironment" && !strings.HasPrefix(rel, proxyHome+"/") {
					findings = append(findings, at+": http.ProxyFromEnvironment outside "+proxyHome)
				}
			case "log/slog":
				if globalLogging[sel.Sel.Name] {
					findings = append(findings, at+": slog."+sel.Sel.Name+" uses the global logger")
				}
			}
			return true
		})
	}
	return findings
}

// importNames maps each name file uses for an import to the import's path.
func importNames(file *ast.File) map[string]string {
	names := map[string]string{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = path
	}
	return names
}
