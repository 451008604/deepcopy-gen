package scanner

import (
	"github.com/451008604/deepcopy-gen/types"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ScanDir walks the given directory recursively, parses all .go files,
// and returns package-grouped struct information.
func ScanDir(dir string) ([]types.PackageInfo, error) {
	packages := make(map[string]*types.PackageInfo)

	type cachedFile struct {
		path   string
		pkgDir string
		file   *ast.File
	}
	var files []cachedFile

	// Pass 1: parse every file and build a package-level type table so that
	// field resolution can tell named basic aliases / anonymous composites
	// apart from named struct/slice/map/array types that carry a deepcopy
	// method.
	typeDefs := make(map[string]map[string]ast.Expr)
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".dc.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		pkgDir := filepath.Dir(path)
		if _, ok := packages[pkgDir]; !ok {
			packages[pkgDir] = &types.PackageInfo{Name: f.Name.Name, Dir: pkgDir}
		}
		td := typeDefs[pkgDir]
		if td == nil {
			td = make(map[string]ast.Expr)
			typeDefs[pkgDir] = td
		}
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					td[typeSpec.Name.Name] = typeSpec.Type
				}
			}
		}
		files = append(files, cachedFile{path: path, pkgDir: pkgDir, file: f})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	// Pass 2: extract structs (and named composite types), resolving field
	// types against the package type table.
	for _, cf := range files {
		pkg := packages[cf.pkgDir]
		pkg.GoFiles = append(pkg.GoFiles, cf.path)
		extracted := extractStructs(cf.file, cf.path, typeDefs[cf.pkgDir])
		detectSelfReferential(extracted)
		pkg.Structs = append(pkg.Structs, extracted...)
	}

	result := make([]types.PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		result = append(result, *pkg)
	}
	return result, nil
}

func extractStructs(f *ast.File, sourceFile string, typeDefs map[string]ast.Expr) []types.StructInfo {
	var structs []types.StructInfo

	importMap := make(map[string]string)
	for _, imp := range f.Imports {
		if imp.Name != nil {
			importMap[imp.Name.Name] = strings.Trim(imp.Path.Value, "\"")
		} else {
			parts := strings.Split(strings.Trim(imp.Path.Value, "\""), "/")
			if len(parts) > 0 {
				importMap[parts[len(parts)-1]] = strings.Trim(imp.Path.Value, "\"")
			}
		}
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				// Collect named composite types (slice/array/map) so the
				// generator emits a deepcopy method for them; fields
				// referencing the named type can then call `.deepcopy(v)`.
				// Named basic aliases and pointer/chan/interface aliases are
				// skipped (no method; shallow copy is correct or N/A).
				switch typeSpec.Type.(type) {
				case *ast.ArrayType, *ast.MapType:
					u := buildFieldInfo("", typeSpec.Type, importMap, typeDefs)
					structs = append(structs, types.StructInfo{
						Name:       typeSpec.Name.Name,
						Package:    f.Name.Name,
						SourceFile: sourceFile,
						Underlying: &u,
					})
				}
				continue
			}

			si := types.StructInfo{
				Name:       typeSpec.Name.Name,
				Package:    f.Name.Name,
				SourceFile: sourceFile,
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fi := parseField(field, importMap, typeDefs)
					si.Fields = append(si.Fields, fi...)
				}
			}

			structs = append(structs, si)
		}
	}

	return structs
}

func detectSelfReferential(structs []types.StructInfo) {
	nameToIdx := make(map[string]int, len(structs))
	for i, s := range structs {
		nameToIdx[s.Name] = i
	}

	for i := range structs {
		if isInCycle(structs, i, nameToIdx) {
			structs[i].IsSelfReferential = true
		}
	}
}

// isInCycle checks whether the struct at startIdx is part of a pointer cycle
// within the same package (e.g. A→B→A or A→A).
func isInCycle(structs []types.StructInfo, startIdx int, nameToIdx map[string]int) bool {
	visited := make(map[string]bool)
	return dfsCheckCycle(structs, startIdx, startIdx, nameToIdx, visited, false)
}

func dfsCheckCycle(structs []types.StructInfo, currentIdx, startIdx int, nameToIdx map[string]int, visited map[string]bool, started bool) bool {
	currentName := structs[currentIdx].Name

	if started && currentIdx == startIdx {
		return true
	}

	if visited[currentName] {
		return false
	}
	visited[currentName] = true
	defer delete(visited, currentName)

	for _, f := range structs[currentIdx].Fields {
		for _, name := range collectPointerTargets(f) {
			if idx, ok := nameToIdx[name]; ok {
				if dfsCheckCycle(structs, idx, startIdx, nameToIdx, visited, true) {
					return true
				}
			}
		}
	}
	return false
}

// collectPointerTargets returns all struct names reachable through pointer fields
// within the same package (ignoring external package types).
func collectPointerTargets(f types.FieldInfo) []string {
	var result []string
	var walk func(fi types.FieldInfo)
	walk = func(fi types.FieldInfo) {
		switch fi.Category {
		case types.TypePointer:
			if fi.ElemType != nil {
				if fi.ElemType.Category == types.TypeStruct && fi.ElemType.TypeName != "" && fi.ElemType.PackageName == "" {
					result = append(result, fi.ElemType.TypeName)
				}
				walk(*fi.ElemType)
			}
		case types.TypeSlice:
			if fi.ElemType != nil {
				walk(*fi.ElemType)
			}
		case types.TypeMap:
			if fi.MapValueType != nil {
				walk(*fi.MapValueType)
			}
		}
	}
	walk(f)
	return result
}

func parseField(field *ast.Field, importMap map[string]string, typeDefs map[string]ast.Expr) []types.FieldInfo {
	if len(field.Names) == 0 {
		name := embeddedFieldName(field.Type)
		fi := buildFieldInfo(name, field.Type, importMap, typeDefs)
		fi.IsEmbedded = true
		fi.IsExported = isExportedTypeExpr(field.Type)
		return []types.FieldInfo{fi}
	}

	var fields []types.FieldInfo
	for _, name := range field.Names {
		fi := buildFieldInfo(name.Name, field.Type, importMap, typeDefs)
		fi.IsExported = len(name.Name) > 0 && unicode.IsUpper(rune(name.Name[0]))
		fields = append(fields, fi)
	}
	return fields
}

func buildFieldInfo(name string, expr ast.Expr, importMap map[string]string, typeDefs map[string]ast.Expr) types.FieldInfo {
	fi := types.FieldInfo{
		Name:     name,
		TypeExpr: typeExprString(expr),
	}
	resolveType(&fi, expr, importMap, typeDefs)
	return fi
}

func resolveType(fi *types.FieldInfo, expr ast.Expr, importMap map[string]string, typeDefs map[string]ast.Expr) {
	if fi.TypeExpr == "" {
		fi.TypeExpr = typeExprString(expr)
	}

	switch t := expr.(type) {
	case *ast.StarExpr:
		fi.Category = types.TypePointer
		fi.ElemType = &types.FieldInfo{TypeExpr: typeExprString(t.X)}
		resolveType(fi.ElemType, t.X, importMap, typeDefs)
		fi.ElemCategory = fi.ElemType.Category

	case *ast.ArrayType:
		if t.Len == nil {
			fi.Category = types.TypeSlice
		} else {
			fi.Category = types.TypeArray
			if lit, ok := t.Len.(*ast.BasicLit); ok && lit.Kind == token.INT {
				n := 0
				for _, c := range lit.Value {
					n = n*10 + int(c-'0')
				}
				fi.ArrayLen = n
			}
		}
		fi.ElemType = &types.FieldInfo{TypeExpr: typeExprString(t.Elt)}
		resolveType(fi.ElemType, t.Elt, importMap, typeDefs)
		fi.ElemCategory = fi.ElemType.Category

	case *ast.MapType:
		fi.Category = types.TypeMap
		fi.MapKeyType = &types.FieldInfo{TypeExpr: typeExprString(t.Key)}
		resolveType(fi.MapKeyType, t.Key, importMap, typeDefs)
		fi.MapValueType = &types.FieldInfo{TypeExpr: typeExprString(t.Value)}
		resolveType(fi.MapValueType, t.Value, importMap, typeDefs)

	case *ast.StructType:
		// Anonymous struct field type (e.g. `S struct{ N *int }`). It cannot
		// carry a method, so the generator emits an inline field-wise copy.
		fi.Category = types.TypeStruct
		fi.HasDeepCopy = false
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fi.AnonFields = append(fi.AnonFields, parseField(field, importMap, typeDefs)...)
			}
		}

	case *ast.InterfaceType:
		fi.Category = types.TypeInterface

	case *ast.ChanType:
		fi.Category = types.TypeChannel

	case *ast.Ident:
		if isBuiltinType(t.Name) {
			fi.Category = types.TypeBasic
		} else {
			// Same-package named type. Keep Category=TypeStruct so the
			// generator dispatches uniformly; record whether the named type
			// has a generated deepcopy method.
			fi.Category = types.TypeStruct
			fi.TypeName = t.Name
			fi.HasDeepCopy = namedTypeHasDeepCopy(t.Name, typeDefs)
		}

	case *ast.SelectorExpr:
		fi.Category = types.TypeStruct
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			fi.PackageName = pkgIdent.Name
			if fullPath, ok := importMap[pkgIdent.Name]; ok {
				fi.PackageName = fullPath
			}
		}
		fi.TypeName = t.Sel.Name

	case *ast.Ellipsis:
		fi.Category = types.TypeSlice
		fi.ElemType = &types.FieldInfo{}
		resolveType(fi.ElemType, t.Elt, importMap, typeDefs)
		fi.ElemCategory = fi.ElemType.Category
	}
}

// namedTypeHasDeepCopy reports whether a same-package named type has a
// generated deepcopy method. Structs and named slice/array/map types get
// methods; named basic aliases (and aliases to pointer/chan/interface) do
// not. Aliases to other named types are resolved transitively.
func namedTypeHasDeepCopy(name string, typeDefs map[string]ast.Expr) bool {
	expr, ok := typeDefs[name]
	if !ok {
		// Not a package-level type decl; assume a same-package struct, which
		// always gets a method. Safe default.
		return true
	}
	switch e := expr.(type) {
	case *ast.StructType, *ast.ArrayType, *ast.MapType:
		return true
	case *ast.Ident:
		if isBuiltinType(e.Name) {
			return false
		}
		return namedTypeHasDeepCopy(e.Name, typeDefs)
	default:
		return false
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "bool", "byte", "rune", "error", "string", "any",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64",
		"complex64", "complex128":
		return true
	}
	return false
}

func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[" + typeExprString(t.Len) + "]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.StructType:
		return structTypeString(t)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.BasicLit:
		return t.Value
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.ChanType:
		return "chan " + typeExprString(t.Value)
	default:
		return "unknown"
	}
}

// structTypeString renders an anonymous struct type as a literal so it can be
// used in generated make/new expressions, e.g. struct{} or struct{ N *int }.
func structTypeString(t *ast.StructType) string {
	if t.Fields == nil || len(t.Fields.List) == 0 {
		return "struct{}"
	}
	var b strings.Builder
	b.WriteString("struct{")
	for i, field := range t.Fields.List {
		if i > 0 {
			b.WriteString("; ")
		}
		ft := typeExprString(field.Type)
		if len(field.Names) == 0 {
			// embedded field
			b.WriteString(ft)
		} else {
			for j, n := range field.Names {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(n.Name)
			}
			b.WriteString(" " + ft)
		}
	}
	b.WriteString("}")
	return b.String()
}

func embeddedFieldName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return embeddedFieldName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

func isExportedTypeExpr(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return len(t.Name) > 0 && unicode.IsUpper(rune(t.Name[0]))
	case *ast.StarExpr:
		return isExportedTypeExpr(t.X)
	case *ast.SelectorExpr:
		return len(t.Sel.Name) > 0 && unicode.IsUpper(rune(t.Sel.Name[0]))
	default:
		return false
	}
}
