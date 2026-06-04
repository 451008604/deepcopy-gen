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
}

// StructInfo describes a single Go struct type found during scanning.
type StructInfo struct {
	Name             string
	Package          string
	Fields           []FieldInfo
	SourceFile       string
	IsSelfReferential bool
}

// PackageInfo groups all structs found in a single Go package directory.
type PackageInfo struct {
	Name    string
	Dir     string
	Structs []StructInfo
	GoFiles []string
}
