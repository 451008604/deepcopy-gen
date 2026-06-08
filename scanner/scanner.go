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

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}

		pkgName := f.Name.Name
		pkgDir := filepath.Dir(path)
		key := pkgDir

		pkg, ok := packages[key]
		if !ok {
			pkg = &types.PackageInfo{
				Name: pkgName,
				Dir:  pkgDir,
			}
			packages[key] = pkg
		}

		pkg.GoFiles = append(pkg.GoFiles, path)
		extracted := extractStructs(f, path)
		detectSelfReferential(extracted)
		pkg.Structs = append(pkg.Structs, extracted...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]types.PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		result = append(result, *pkg)
	}
	return result, nil
}

func extractStructs(f *ast.File, sourceFile string) []types.StructInfo {
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
				continue
			}

			si := types.StructInfo{
				Name:       typeSpec.Name.Name,
				Package:    f.Name.Name,
				SourceFile: sourceFile,
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fi := parseField(field, importMap)
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

func parseField(field *ast.Field, importMap map[string]string) []types.FieldInfo {
	if len(field.Names) == 0 {
		name := embeddedFieldName(field.Type)
		fi := buildFieldInfo(name, field.Type, importMap)
		fi.IsEmbedded = true
		fi.IsExported = isExportedTypeExpr(field.Type)
		return []types.FieldInfo{fi}
	}

	var fields []types.FieldInfo
	for _, name := range field.Names {
		fi := buildFieldInfo(name.Name, field.Type, importMap)
		fi.IsExported = len(name.Name) > 0 && unicode.IsUpper(rune(name.Name[0]))
		fields = append(fields, fi)
	}
	return fields
}

func buildFieldInfo(name string, expr ast.Expr, importMap map[string]string) types.FieldInfo {
	fi := types.FieldInfo{
		Name:     name,
		TypeExpr: typeExprString(expr),
	}
	resolveType(&fi, expr, importMap)
	return fi
}

func resolveType(fi *types.FieldInfo, expr ast.Expr, importMap map[string]string) {
	if fi.TypeExpr == "" {
		fi.TypeExpr = typeExprString(expr)
	}

	switch t := expr.(type) {
	case *ast.StarExpr:
		fi.Category = types.TypePointer
		fi.ElemType = &types.FieldInfo{TypeExpr: typeExprString(t.X)}
		resolveType(fi.ElemType, t.X, importMap)
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
		resolveType(fi.ElemType, t.Elt, importMap)
		fi.ElemCategory = fi.ElemType.Category

	case *ast.MapType:
		fi.Category = types.TypeMap
		fi.MapKeyType = &types.FieldInfo{TypeExpr: typeExprString(t.Key)}
		resolveType(fi.MapKeyType, t.Key, importMap)
		fi.MapValueType = &types.FieldInfo{TypeExpr: typeExprString(t.Value)}
		resolveType(fi.MapValueType, t.Value, importMap)

	case *ast.StructType:
		fi.Category = types.TypeStruct

	case *ast.InterfaceType:
		fi.Category = types.TypeInterface

	case *ast.ChanType:
		fi.Category = types.TypeChannel

	case *ast.Ident:
		if isBuiltinType(t.Name) {
			fi.Category = types.TypeBasic
		} else {
			fi.Category = types.TypeStruct
			fi.TypeName = t.Name
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
		resolveType(fi.ElemType, t.Elt, importMap)
		fi.ElemCategory = fi.ElemType.Category
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
