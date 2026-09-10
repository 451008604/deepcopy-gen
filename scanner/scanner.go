package scanner

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/451008604/deepcopy-gen/types"
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
	typeDefs := make(map[string]map[string]*ast.TypeSpec)
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != dir && (strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".dc.go") {
			return nil
		}
		match, matchErr := build.Default.MatchFile(filepath.Dir(path), info.Name())
		if matchErr != nil {
			return matchErr
		}
		if !match {
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
		if packages[pkgDir].Name != f.Name.Name {
			return fmt.Errorf("%s: conflicting packages %s and %s", path, packages[pkgDir].Name, f.Name.Name)
		}
		td := typeDefs[pkgDir]
		if td == nil {
			td = make(map[string]*ast.TypeSpec)
			typeDefs[pkgDir] = td
		}
		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					td[typeSpec.Name.Name] = typeSpec
				}
			}
		}
		files = append(files, cachedFile{path: path, pkgDir: pkgDir, file: f})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	// 统一每个包的导入名称，防止不同文件的同名别名或 dc 别名冲突。
	packageImports := make(map[string]map[string]string)
	for _, cf := range files {
		imports := packageImports[cf.pkgDir]
		if imports == nil {
			imports = make(map[string]string)
			packageImports[cf.pkgDir] = imports
		}
		rename := make(map[string]string)
		for _, imp := range cf.file.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			local := filepath.Base(path)
			if imp.Name != nil {
				local = imp.Name.Name
			}
			if local == "_" {
				continue
			}
			if local == "." || path == "C" {
				return nil, fmt.Errorf("%s: dot imports and cgo types are not supported", cf.path)
			}
			canonical := ""
			for name, existing := range imports {
				if existing == path {
					canonical = name
					break
				}
			}
			if canonical == "" {
				canonical = local
				for n := 1; imports[canonical] != "" || typeDefs[cf.pkgDir][canonical] != nil || slices.Contains([]string{"dc", "in", "out", "v", "i", "k", "val", "cp", "cached"}, canonical); n++ {
					canonical = fmt.Sprintf("%s%d", local, n)
				}
				imports[canonical] = path
			}
			rename[local] = canonical
		}
		for _, decl := range cf.file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			// 类型与数组长度常量中的包限定名也必须保持一致。
			ast.Inspect(gd, func(node ast.Node) bool {
				if sel, ok := node.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok && rename[ident.Name] != "" {
						ident.Name = rename[ident.Name]
					}
				}
				return true
			})
		}
	}
	for _, cf := range files {
		pkg := packages[cf.pkgDir]
		pkg.GoFiles = append(pkg.GoFiles, cf.path)
		extracted, err := extractStructs(cf.file, cf.path, typeDefs[cf.pkgDir], packageImports[cf.pkgDir])
		if err != nil {
			return nil, err
		}
		pkg.Structs = append(pkg.Structs, extracted...)
	}
	for _, pkg := range packages {
		detectSelfReferential(pkg.Structs)
	}

	result := make([]types.PackageInfo, 0, len(packages))
	for _, pkg := range packages {
		result = append(result, *pkg)
	}
	slices.SortFunc(result, func(a, b types.PackageInfo) int { return strings.Compare(a.Dir, b.Dir) })
	return result, nil
}

func extractStructs(f *ast.File, sourceFile string, typeDefs map[string]*ast.TypeSpec, importMap map[string]string) ([]types.StructInfo, error) {
	var structs []types.StructInfo
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			// 别名复用原类型的方法；匿名别名只能在字段处内联复制。
			if ts.Assign.IsValid() && ts.TypeParams != nil {
				return nil, fmt.Errorf("%s: generic aliases are outside the Go 1.21 target", sourceFile)
			}
			if ts.Assign.IsValid() || ts.Name.Name == "_" {
				continue
			}
			defs := maps.Clone(typeDefs)
			var params []string
			renameParams := make(map[string]ast.Expr)
			if ts.TypeParams != nil {
				for _, field := range ts.TypeParams.List {
					for _, name := range field.Names {
						param := name.Name
						if slices.Contains([]string{"dc", "in", "out", "v", "i", "k", "val", "cp", "cached", "_"}, param) || importMap[param] != "" {
							param = fmt.Sprintf("DCType%d", len(params))
							renameParams[name.Name] = ast.NewIdent(param)
						}
						params = append(params, param)
						defs[param] = &ast.TypeSpec{Name: ast.NewIdent(param)}
					}
				}
			}
			sourceType := ts.Type
			if len(renameParams) > 0 {
				sourceType = substituteType(sourceType, renameParams)
			}
			expr, err := underlyingType(sourceType, defs, make(map[string]bool))
			if err != nil {
				return nil, fmt.Errorf("%s: %s: %w", sourceFile, ts.Name.Name, err)
			}
			si := types.StructInfo{Name: ts.Name.Name, TypeParams: params, Package: f.Name.Name, SourceFile: sourceFile}
			switch t := expr.(type) {
			case *ast.StructType:
				for _, field := range t.Fields.List {
					si.Fields = append(si.Fields, parseField(field, importMap, defs)...)
				}
			case *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr:
				return nil, fmt.Errorf("%s: %s: cannot resolve underlying type %s; use an alias or a local structural definition", sourceFile, ts.Name.Name, typeExprString(expr))
			case *ast.ArrayType, *ast.MapType:
				u := buildFieldInfo("", expr, importMap, defs)
				si.Underlying = &u
			default:
				continue
			}
			structs = append(structs, si)
		}
	}
	return structs, nil
}

// 只展开声明外层，字段中的递归引用仍通过生成的方法处理。
func underlyingType(expr ast.Expr, defs map[string]*ast.TypeSpec, seen map[string]bool) (ast.Expr, error) {
	switch t := expr.(type) {
	case *ast.ParenExpr:
		return underlyingType(t.X, defs, seen)
	case *ast.IndexExpr, *ast.IndexListExpr:
		var base ast.Expr
		var args []ast.Expr
		if idx, ok := expr.(*ast.IndexExpr); ok {
			base = idx.X
			args = []ast.Expr{idx.Index}
		} else {
			idx := expr.(*ast.IndexListExpr)
			base = idx.X
			args = idx.Indices
		}
		ident, ok := base.(*ast.Ident)
		if !ok {
			return expr, nil
		}
		def := defs[ident.Name]
		if def == nil || def.TypeParams == nil {
			return expr, nil
		}
		if seen[ident.Name] {
			return nil, fmt.Errorf("cyclic generic definition involving %s", ident.Name)
		}
		bindings := make(map[string]ast.Expr)
		index := 0
		for _, field := range def.TypeParams.List {
			for _, param := range field.Names {
				if index >= len(args) {
					return nil, fmt.Errorf("missing type arguments for %s", ident.Name)
				}
				bindings[param.Name] = args[index]
				index++
			}
		}
		if index != len(args) {
			return nil, fmt.Errorf("too many type arguments for %s", ident.Name)
		}
		seen[ident.Name] = true
		defer delete(seen, ident.Name)
		return underlyingType(substituteType(def.Type, bindings), defs, seen)

	case *ast.Ident:
		if def := defs[t.Name]; def != nil && def.Type != nil {
			if seen[t.Name] {
				return nil, fmt.Errorf("cyclic type definition involving %s", t.Name)
			}
			seen[t.Name] = true
			defer delete(seen, t.Name)
			return underlyingType(def.Type, defs, seen)
		}
	}
	return expr, nil
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

	fields := structs[currentIdx].Fields
	if u := structs[currentIdx].Underlying; u != nil {
		fields = append(slices.Clone(fields), *u)
	}
	for _, f := range fields {
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
		case types.TypeSlice, types.TypeArray:
			if fi.ElemType != nil {
				walk(*fi.ElemType)
			}
		case types.TypeStruct:
			for _, field := range fi.AnonFields {
				walk(field)
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

func parseField(field *ast.Field, importMap map[string]string, typeDefs map[string]*ast.TypeSpec) []types.FieldInfo {
	if len(field.Names) == 0 {
		name := embeddedFieldName(field.Type)
		fi := buildFieldInfo(name, field.Type, importMap, typeDefs)
		fi.IsEmbedded = true
		fi.IsExported = isExportedTypeExpr(field.Type)
		return []types.FieldInfo{fi}
	}

	var fields []types.FieldInfo
	for _, name := range field.Names {
		if name.Name == "_" {
			continue
		}
		fi := buildFieldInfo(name.Name, field.Type, importMap, typeDefs)
		fi.IsExported = len(name.Name) > 0 && ast.IsExported(name.Name)
		fields = append(fields, fi)
	}
	return fields
}

func buildFieldInfo(name string, expr ast.Expr, importMap map[string]string, typeDefs map[string]*ast.TypeSpec) types.FieldInfo {
	fi := types.FieldInfo{
		Name:     name,
		TypeExpr: typeExprString(expr), Imports: importMap,
	}
	resolveType(&fi, expr, importMap, typeDefs)
	return fi
}

func resolveType(fi *types.FieldInfo, expr ast.Expr, importMap map[string]string, typeDefs map[string]*ast.TypeSpec) {
	fi.Imports = importMap
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
				n, _ := strconv.ParseInt(strings.ReplaceAll(lit.Value, "_", ""), 0, 64)
				fi.ArrayLen = int(n)
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

	case *ast.ParenExpr:
		resolveType(fi, t.X, importMap, typeDefs)
	case *ast.FuncType:
		fi.Category = types.TypeFunc
	case *ast.IndexExpr, *ast.IndexListExpr:
		var base ast.Expr
		if idx, ok := expr.(*ast.IndexExpr); ok {
			base = idx.X
		} else {
			base = expr.(*ast.IndexListExpr).X
		}
		if ident, ok := base.(*ast.Ident); ok && !namedTypeHasDeepCopy(ident.Name, typeDefs) {
			underlying, err := underlyingType(expr, typeDefs, make(map[string]bool))
			if err == nil && underlying != expr {
				resolveType(fi, underlying, importMap, typeDefs)
				return
			}
		}
		resolveType(fi, base, importMap, typeDefs)
	case *ast.Ident:
		if def := typeDefs[t.Name]; def != nil {
			if def.Type == nil {
				fi.Category = types.TypeParameter
				return
			}
			if !def.Assign.IsValid() && namedTypeHasDeepCopy(t.Name, typeDefs) {
				fi.Category = types.TypeStruct
				fi.TypeName = t.Name
				fi.HasDeepCopy = true
				return
			}
			// 不可声明方法的命名指针及别名按底层形状复制。
			remaining := maps.Clone(typeDefs)
			delete(remaining, t.Name)
			resolveType(fi, def.Type, importMap, remaining)
		} else if t.Name == "any" || t.Name == "error" {
			fi.Category = types.TypeInterface
		} else if isBuiltinType(t.Name) {
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
		resolveType(fi.ElemType, t.Elt, importMap, typeDefs)
		fi.ElemCategory = fi.ElemType.Category
	}
}

// 只有可声明方法的本地结构体、切片、数组和 map 定义才生成复制方法。
func namedTypeHasDeepCopy(name string, typeDefs map[string]*ast.TypeSpec) bool {
	def := typeDefs[name]
	if def == nil || def.Type == nil || def.Assign.IsValid() {
		return false
	}
	expr, err := underlyingType(def.Type, typeDefs, make(map[string]bool))
	if err != nil {
		return false
	}
	switch expr.(type) {
	case *ast.StructType, *ast.ArrayType, *ast.MapType:
		return true
	}
	return false
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
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), expr); err != nil {
		return ""
	}
	return b.String()
}

func embeddedFieldName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return embeddedFieldName(t.X)
	case *ast.IndexExpr:
		return embeddedFieldName(t.X)
	case *ast.IndexListExpr:
		return embeddedFieldName(t.X)
	case *ast.ParenExpr:
		return embeddedFieldName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

func isExportedTypeExpr(expr ast.Expr) bool {
	return ast.IsExported(embeddedFieldName(expr))
}

// 替换类型位置的参数，不改写同名字段、方法或导入限定名。
func substituteType(expr ast.Expr, args map[string]ast.Expr) ast.Expr {
	cloned, err := parser.ParseExpr(typeExprString(expr))
	if err != nil {
		return expr
	}
	var replace func(ast.Expr) ast.Expr
	var fields func(*ast.FieldList)
	fields = func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, field := range list.List {
			field.Type = replace(field.Type)
		}
	}
	replace = func(expr ast.Expr) ast.Expr {
		switch t := expr.(type) {
		case *ast.Ident:
			if arg := args[t.Name]; arg != nil {
				return arg
			}
		case *ast.StarExpr:
			t.X = replace(t.X)
		case *ast.ParenExpr:
			t.X = replace(t.X)
		case *ast.ArrayType:
			t.Elt = replace(t.Elt)
		case *ast.MapType:
			t.Key = replace(t.Key)
			t.Value = replace(t.Value)
		case *ast.ChanType:
			t.Value = replace(t.Value)
		case *ast.StructType:
			fields(t.Fields)
		case *ast.InterfaceType:
			fields(t.Methods)
		case *ast.FuncType:
			fields(t.Params)
			fields(t.Results)
		case *ast.Ellipsis:
			t.Elt = replace(t.Elt)
		case *ast.IndexExpr:
			t.X = replace(t.X)
			t.Index = replace(t.Index)
		case *ast.IndexListExpr:
			t.X = replace(t.X)
			for i, arg := range t.Indices {
				t.Indices[i] = replace(arg)
			}
		case *ast.UnaryExpr:
			t.X = replace(t.X)
		case *ast.BinaryExpr:
			t.X = replace(t.X)
			t.Y = replace(t.Y)
		}
		return expr
	}
	return replace(cloned)
}
