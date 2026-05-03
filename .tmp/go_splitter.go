package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"sort"
	"strings"
)

type MoveSpec struct {
	NewFile string   `json:"new_file"`
	Funcs   []string `json:"funcs"`
}

type FileSpec struct {
	Path  string     `json:"path"`
	Moves []MoveSpec `json:"moves"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: go_splitter <config.json>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var specs []FileSpec
	if err := json.Unmarshal(raw, &specs); err != nil {
		panic(err)
	}
	for _, spec := range specs {
		if err := splitFile(spec); err != nil {
			panic(fmt.Sprintf("%s: %v", spec.Path, err))
		}
	}
}

func splitFile(spec FileSpec) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, spec.Path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	declByKey := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		key := funcKey(fn)
		declByKey[key] = fn
		if _, exists := declByKey[fn.Name.Name]; !exists {
			declByKey[fn.Name.Name] = fn
		}
	}

	for _, move := range spec.Moves {
		moveSet := map[*ast.FuncDecl]bool{}
		var moved []ast.Decl
		for _, name := range move.Funcs {
			fn := declByKey[name]
			if fn == nil {
				return fmt.Errorf("function %s not found", name)
			}
			if moveSet[fn] {
				continue
			}
			moveSet[fn] = true
			moved = append(moved, fn)
		}

		out := &ast.File{
			Name:  ast.NewIdent(file.Name.Name),
			Decls: moved,
		}
		out.Imports = selectImports(file, moved)
		if err := writeFile(move.NewFile, out); err != nil {
			return err
		}

		var remaining []ast.Decl
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && moveSet[fn] {
				continue
			}
			remaining = append(remaining, decl)
		}
		file.Decls = remaining
		file.Imports = selectImports(file, remaining)
	}

	return writeFile(spec.Path, file)
}

func funcKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return receiverTypeName(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

func selectImports(src *ast.File, decls []ast.Decl) []*ast.ImportSpec {
	importMap := map[string]*ast.ImportSpec{}
	for _, imp := range src.Imports {
		name := importName(imp)
		if name != "" {
			importMap[name] = imp
		}
	}

	used := map[string]*ast.ImportSpec{}
	for _, decl := range decls {
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if imp := importMap[ident.Name]; imp != nil {
				used[imp.Path.Value] = imp
			}
			return true
		})
	}

	if len(used) == 0 {
		return nil
	}
	keys := make([]string, 0, len(used))
	for key := range used {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	imports := make([]*ast.ImportSpec, 0, len(keys))
	for _, key := range keys {
		imports = append(imports, used[key])
	}
	return imports
}

func importName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		if imp.Name.Name == "_" || imp.Name.Name == "." {
			return ""
		}
		return imp.Name.Name
	}
	importPath := strings.Trim(imp.Path.Value, `"`)
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") && len(base) > 1 && digitsOnly(base[1:]) {
		return path.Base(path.Dir(importPath))
	}
	return base
}

func writeFile(target string, file *ast.File) error {
	file.Comments = nil
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			d.Doc = nil
		case *ast.GenDecl:
			d.Doc = nil
			d.Specs = sanitizeSpecs(d.Specs)
		}
	}

	var decls []ast.Decl
	if len(file.Imports) > 0 {
		decls = append(decls, &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: func() []ast.Spec {
				specs := make([]ast.Spec, 0, len(file.Imports))
				for _, imp := range file.Imports {
					specs = append(specs, imp)
				}
				return specs
			}(),
		})
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			continue
		}
		decls = append(decls, decl)
	}
	file.Decls = decls

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), file); err != nil {
		return err
	}
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return os.WriteFile(target, out, 0644)
}

func sanitizeSpecs(specs []ast.Spec) []ast.Spec {
	for _, spec := range specs {
		switch s := spec.(type) {
		case *ast.ImportSpec:
			s.Doc = nil
			s.Comment = nil
		case *ast.TypeSpec:
			s.Doc = nil
			s.Comment = nil
		case *ast.ValueSpec:
			s.Doc = nil
			s.Comment = nil
		}
	}
	return specs
}

func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
