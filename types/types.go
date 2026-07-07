package types

// TypeCategory classifies the kind of a struct field's Go type.
type TypeCategory int

const (
	TypeBasic     TypeCategory = iota // bool, int, string, float64, etc.
	TypePointer                       // *T
	TypeSlice                         // []T
	TypeArray                         // [N]T
	TypeMap                           // map[K]V
	TypeStruct                        // named struct value type
	TypeInterface                     // interface{}
	TypeChannel                       // chan T
)

// FieldInfo describes a single field within a struct declaration.
type FieldInfo struct {
	Name         string
	TypeExpr     string
	Category     TypeCategory
	IsExported   bool
	IsEmbedded   bool
	ElemType     *FieldInfo
	ArrayLen     int
	ElemCategory TypeCategory
	MapKeyType   *FieldInfo
	MapValueType *FieldInfo
	PackageName  string
	TypeName     string

	// HasDeepCopy indicates that a same-package named type referenced by this
	// field has a generated deepcopy method (struct/slice/map/array). Named
	// basic aliases and anonymous structs have HasDeepCopy=false.
	HasDeepCopy bool
	// AnnonFields holds the parsed fields of an anonymous struct field type
	// (ast.StructType used directly as a field type). Nil for named types.
	AnonFields []FieldInfo
}

// StructInfo describes a single Go type found during scanning. For ordinary
// struct types, Fields is populated. For named non-struct types (e.g. a named
// slice type `type heap []T`), Underlying describes the underlying type so the
// generator can emit a suitable deepcopy method.
type StructInfo struct {
	Name              string
	Package           string
	Fields            []FieldInfo
	SourceFile        string
	IsSelfReferential bool
	Underlying        *FieldInfo
}

// PackageInfo groups all structs found in a single Go package directory.
type PackageInfo struct {
	Name    string
	Dir     string
	Structs []StructInfo
	GoFiles []string
}
