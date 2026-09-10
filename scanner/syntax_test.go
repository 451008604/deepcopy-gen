package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/451008604/deepcopy-gen/types"
)

func TestScanDir_TypeSyntax(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package fixture
 type Alias = struct { Value *int }
 type Scalar int
 type ScalarAlias = Scalar
 type Defined Alias
 type Pointer *Defined
 type Generic[T any] struct { Value T; Next *Generic[T] }
 type Data struct {
 A Alias
 P Pointer
 F func(...int) (string,error)
 C <-chan int
 I interface { Read([]byte)(int,error) }
 X *(int)
 Y [0x10]*int
 Z [1_0]*int
 Generic[int]
 Éxported int
 _ *int
 }
`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	pkg := packages[0]
	if findStruct(pkg.Structs, "Alias") != nil || findStruct(pkg.Structs, "Pointer") != nil {
		t.Fatal("alias/pointer must not receive methods")
	}
	if findStruct(pkg.Structs, "Defined") == nil {
		t.Fatal("defined struct missing")
	}
	data := findStruct(pkg.Structs, "Data")
	if len(data.Fields) != 10 {
		t.Fatalf("blank field not skipped: %d", len(data.Fields))
	}
	for _, tc := range []struct {
		name     string
		category types.TypeCategory
		expr     string
	}{
		{"F", types.TypeFunc, "func(...int) (string, error)"},
		{"C", types.TypeChannel, "<-chan int"},
		{"I", types.TypeInterface, "interface{ Read([]byte) (int, error) }"},
		{"X", types.TypePointer, "*(int)"},
	} {
		f := findField(data.Fields, tc.name)
		if f.Category != tc.category || f.TypeExpr != tc.expr {
			t.Errorf("%s: category=%v expression=%q", tc.name, f.Category, f.TypeExpr)
		}
	}
	if findField(data.Fields, "Y").ArrayLen != 16 || findField(data.Fields, "Z").ArrayLen != 10 {
		t.Fatal("array literal length parsed incorrectly")
	}
	if !findField(data.Fields, "Éxported").IsExported {
		t.Fatal("Unicode exported field lost")
	}
	if f := findField(data.Fields, "Generic"); !f.IsEmbedded || !f.HasDeepCopy {
		t.Fatal("generic embedding lost")
	}
	if f := findField(data.Fields, "P"); f.Category != types.TypePointer || !f.ElemType.HasDeepCopy {
		t.Fatal("defined pointer shape lost")
	}
	g := findStruct(pkg.Structs, "Generic")
	if len(g.TypeParams) != 1 || g.TypeParams[0] != "T" || findField(g.Fields, "Value").Category != types.TypeParameter {
		t.Fatal("type parameter metadata lost")
	}
}

func TestScanDir_CyclesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "a.go", `package fixture; type A struct { Links [2]*B }`)
	writeGoFile(t, dir, "b.go", `package fixture; type B struct { Link struct { Back *A } }`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A", "B"} {
		if !findStruct(packages[0].Structs, name).IsSelfReferential {
			t.Errorf("%s cycle was not detected", name)
		}
	}
}

func TestScanDir_SkipsInactiveAndGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package fixture; type Active struct{ Value int }`)
	writeGoFile(t, dir, "inactive.go", "//go:build ignore\n\npackage fixture\ntype Broken struct { invalid syntax }\n")
	writeGoFile(t, dir, "deepcopy.dc.go", `invalid generated output`)
	hidden := filepath.Join(dir, ".ignored")
	if err := os.Mkdir(hidden, 0755); err != nil {
		t.Fatal(err)
	}
	writeGoFile(t, hidden, "broken.go", `invalid Go`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || len(packages[0].GoFiles) != 1 || len(packages[0].Structs) != 1 {
		t.Fatalf("unexpected scan result: %+v", packages)
	}
}

func TestScanDir_RejectsAmbiguousInput(t *testing.T) {
	for name, source := range map[string]string{
		"cycle":               `package fixture; type A B; type B A`,
		"dot import":          `package fixture; import . "time"; type A struct{ Time }`,
		"cgo":                 `package fixture; import "C"; type A struct{ V C.int }`,
		"generic alias":       `package fixture; type Alias[T any] = []T`,
		"external definition": `package fixture; import "time"; type Local time.Time`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeGoFile(t, dir, "types.go", source)
			if _, err := ScanDir(dir); err == nil {
				t.Fatal("expected actionable scan error")
			}
		})
	}
	t.Run("mixed packages", func(t *testing.T) {
		dir := t.TempDir()
		writeGoFile(t, dir, "a.go", `package one; type A struct{}`)
		writeGoFile(t, dir, "b.go", `package two; type B struct{}`)
		if _, err := ScanDir(dir); err == nil {
			t.Fatal("mixed packages accepted")
		}
	})
}

func TestScanDir_GenericDefinitions(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package fixture
 type Pair[K comparable,V any] struct{ Key K; Value V }
 type Box[T comparable] struct {
  Value T
  Pointer *T
  Slice []T
  Array [2](T)
  Map map[string]T
  Function func(...T) T
  Channel chan<- T
  Interface interface{ Accept(T) }
  Other Pair[string,T]
 }
 type Derived Box[int]
 type Alias = Derived
 type AliasHolder struct { Value Alias }
 type Dict[K comparable,V any] map[K][]V
 type LocalDict Dict[string,int]
 type Odd[in,out,dc any] struct { Values []in; Value out; dc dc }
 type Pointer[T any] *T
 type PointerHolder struct { Value Pointer[*Derived] }
`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	pkg := packages[0]
	derived := findStruct(pkg.Structs, "Derived")
	if derived == nil || len(derived.TypeParams) != 0 {
		t.Fatal("concrete generic definition missing")
	}
	for _, tc := range []struct{ name, expr string }{
		{"Value", "int"}, {"Pointer", "*int"}, {"Slice", "[]int"}, {"Array", "[2](int)"}, {"Map", "map[string]int"}, {"Function", "func(...int) int"}, {"Channel", "chan<- int"}, {"Interface", "interface{ Accept(int) }"}, {"Other", "Pair[string, int]"},
	} {
		f := findField(derived.Fields, tc.name)
		if f == nil || f.TypeExpr != tc.expr {
			t.Errorf("%s: want %q, got %+v", tc.name, tc.expr, f)
		}
	}
	if dict := findStruct(pkg.Structs, "LocalDict"); dict == nil || dict.Underlying.MapKeyType.TypeExpr != "string" || dict.Underlying.MapValueType.TypeExpr != "[]int" {
		t.Fatal("generic named map not instantiated")
	}
	odd := findStruct(pkg.Structs, "Odd")
	if len(odd.TypeParams) != 3 || odd.TypeParams[0] == "in" || findField(odd.Fields, "dc").Name != "dc" {
		t.Fatal("receiver type parameter conflicts were not handled")
	}
	p := findField(findStruct(pkg.Structs, "PointerHolder").Fields, "Value")
	if p.Category != types.TypePointer || p.ElemType.Category != types.TypePointer || !p.ElemType.ElemType.HasDeepCopy {
		t.Fatal("instantiated named pointer shape lost")
	}
	if !findField(findStruct(pkg.Structs, "AliasHolder").Fields, "Value").HasDeepCopy {
		t.Fatal("alias does not reuse base method")
	}
}

func TestScanDir_ImportAliasesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "a.go", `package fixture
 import shared "time"
 type A struct { Values []shared.Time }
 `)
	writeGoFile(t, dir, "b.go", `package fixture
 import (shared "net"; dc "time"; _ "fmt")
 type B struct { Values []shared.IP; Time *dc.Time }
 `)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	pkg := packages[0]
	a := findField(findStruct(pkg.Structs, "A").Fields, "Values")
	b := findField(findStruct(pkg.Structs, "B").Fields, "Values")
	if a.TypeExpr == b.TypeExpr || a.ElemType.PackageName != "time" || b.ElemType.PackageName != "net" {
		t.Fatal("import names collided")
	}
	if a.Imports["shared"] != "time" || b.Imports["shared1"] != "net" {
		t.Fatal("canonical imports missing")
	}
	when := findField(findStruct(pkg.Structs, "B").Fields, "Time")
	if when.TypeExpr != "*shared.Time" {
		t.Fatalf("one import path has multiple aliases: %s", when.TypeExpr)
	}
}
