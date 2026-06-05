package scanner

import (
	"github.com/451008604/deepcopy-gen/types"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine testdata path")
	}
	return filepath.Join(filepath.Dir(file), "..", "testdata")
}

func TestScanDir_Simple(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "simple")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.Name != "simple" {
		t.Errorf("expected package name 'simple', got %q", pkg.Name)
	}

	structNames := sortedStructNames(pkg.Structs)
	expected := []string{"Config", "Person", "Point"}
	if !sliceEqual(structNames, expected) {
		t.Errorf("expected structs %v, got %v", expected, structNames)
	}
}

func TestScanDir_Complex(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if len(pkg.Structs) != 5 {
		t.Fatalf("expected 5 structs, got %d", len(pkg.Structs))
	}
}

func TestScanDir_Nested(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "nested")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	structNames := sortedStructNames(pkg.Structs)
	expected := []string{"Address", "Department", "Employee", "Node"}
	if !sliceEqual(structNames, expected) {
		t.Errorf("expected structs %v, got %v", expected, structNames)
	}
}

func TestScanDir_Empty(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "empty")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if len(pkg.Structs) != 2 {
		t.Fatalf("expected 2 structs, got %d", len(pkg.Structs))
	}
	for _, s := range pkg.Structs {
		if len(s.Fields) != 0 {
			t.Errorf("struct %s should have 0 fields, got %d", s.Name, len(s.Fields))
		}
	}
}

func TestScanDir_AllPackages(t *testing.T) {
	dir := testdataDir(t)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	pkgNames := make([]string, len(packages))
	for i, p := range packages {
		pkgNames[i] = p.Name
	}
	sort.Strings(pkgNames)

	expected := []string{"complex", "crosspkg", "edgecase", "embedded", "empty", "external", "iface", "multifile", "nested", "selector", "selfref", "simple"}
	if !sliceEqual(pkgNames, expected) {
		t.Errorf("expected packages %v, got %v", expected, pkgNames)
	}
}

func TestScanDir_NonExistent(t *testing.T) {
	_, err := ScanDir("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestScanDir_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "main.go", `package test
type Foo struct { X int }
`)
	writeGoFile(t, dir, "main_test.go", `package test
type Bar struct { Y int }
`)

	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	if len(packages[0].Structs) != 1 {
		t.Errorf("expected 1 struct (test file should be skipped), got %d", len(packages[0].Structs))
	}
	if packages[0].Structs[0].Name != "Foo" {
		t.Errorf("expected struct Foo, got %s", packages[0].Structs[0].Name)
	}
}

func TestFieldTypeResolution_Pointer(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	wp := findStruct(packages[0].Structs, "WithPointer")
	if wp == nil {
		t.Fatal("WithPointer struct not found")
	}

	tests := []struct {
		field    string
		category types.TypeCategory
		elemCat  types.TypeCategory
	}{
		{"Name", types.TypePointer, types.TypeBasic},
		{"Value", types.TypePointer, types.TypeBasic},
		{"Data", types.TypePointer, types.TypeSlice},
		{"Ref", types.TypePointer, types.TypePointer},
	}

	for _, tt := range tests {
		f := findField(wp.Fields, tt.field)
		if f == nil {
			t.Errorf("field %s not found", tt.field)
			continue
		}
		if f.Category != tt.category {
			t.Errorf("field %s: expected category %d, got %d", tt.field, tt.category, f.Category)
		}
		if f.ElemType == nil {
			t.Errorf("field %s: ElemType is nil", tt.field)
			continue
		}
		if f.ElemType.Category != tt.elemCat {
			t.Errorf("field %s: expected elem category %d, got %d", tt.field, tt.elemCat, f.ElemType.Category)
		}
	}
}

func TestFieldTypeResolution_Slice(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	ws := findStruct(packages[0].Structs, "WithSlice")
	if ws == nil {
		t.Fatal("WithSlice struct not found")
	}

	tests := []struct {
		field   string
		elemCat types.TypeCategory
	}{
		{"Names", types.TypeBasic},
		{"Numbers", types.TypeBasic},
		{"Matrix", types.TypeSlice},
		{"Ptrs", types.TypePointer},
	}

	for _, tt := range tests {
		f := findField(ws.Fields, tt.field)
		if f == nil {
			t.Errorf("field %s not found", tt.field)
			continue
		}
		if f.Category != types.TypeSlice {
			t.Errorf("field %s: expected TypeSlice, got %d", tt.field, f.Category)
		}
		if f.ElemCategory != tt.elemCat {
			t.Errorf("field %s: expected elem category %d, got %d", tt.field, tt.elemCat, f.ElemCategory)
		}
	}
}

func TestFieldTypeResolution_Map(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	wm := findStruct(packages[0].Structs, "WithMap")
	if wm == nil {
		t.Fatal("WithMap struct not found")
	}

	tests := []struct {
		field    string
		keyCat   types.TypeCategory
		valueCat types.TypeCategory
	}{
		{"Labels", types.TypeBasic, types.TypeBasic},
		{"Scores", types.TypeBasic, types.TypeBasic},
		{"Nested", types.TypeBasic, types.TypeSlice},
		{"PtrMap", types.TypeBasic, types.TypePointer},
		{"IntKeyMap", types.TypeBasic, types.TypeBasic},
	}

	for _, tt := range tests {
		f := findField(wm.Fields, tt.field)
		if f == nil {
			t.Errorf("field %s not found", tt.field)
			continue
		}
		if f.Category != types.TypeMap {
			t.Errorf("field %s: expected TypeMap, got %d", tt.field, f.Category)
		}
		if f.MapKeyType == nil || f.MapKeyType.Category != tt.keyCat {
			t.Errorf("field %s: key category mismatch", tt.field)
		}
		if f.MapValueType == nil || f.MapValueType.Category != tt.valueCat {
			t.Errorf("field %s: value category mismatch", tt.field)
		}
	}
}

func TestFieldTypeResolution_Array(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	wa := findStruct(packages[0].Structs, "WithArray")
	if wa == nil {
		t.Fatal("WithArray struct not found")
	}

	coords := findField(wa.Fields, "Coords")
	if coords == nil {
		t.Fatal("Coords field not found")
	}
	if coords.Category != types.TypeArray {
		t.Errorf("expected TypeArray, got %d", coords.Category)
	}
	if coords.ArrayLen != 3 {
		t.Errorf("expected array length 3, got %d", coords.ArrayLen)
	}
	if coords.ElemCategory != types.TypeBasic {
		t.Errorf("expected elem category TypeBasic, got %d", coords.ElemCategory)
	}

	matrix := findField(wa.Fields, "Matrix")
	if matrix == nil {
		t.Fatal("Matrix field not found")
	}
	if matrix.ArrayLen != 2 {
		t.Errorf("expected array length 2, got %d", matrix.ArrayLen)
	}
	if matrix.ElemCategory != types.TypeArray {
		t.Errorf("expected elem category TypeArray, got %d", matrix.ElemCategory)
	}
}

func TestFieldTypeResolution_BasicFields(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "simple")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	person := findStruct(packages[0].Structs, "Person")
	if person == nil {
		t.Fatal("Person struct not found")
	}

	for _, f := range person.Fields {
		if f.Category != types.TypeBasic {
			t.Errorf("field %s: expected TypeBasic, got %d", f.Name, f.Category)
		}
		if !f.IsExported {
			t.Errorf("field %s: expected IsExported=true", f.Name)
		}
	}
}

func TestFieldTypeResolution_NamedStruct(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "nested")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	emp := findStruct(packages[0].Structs, "Employee")
	if emp == nil {
		t.Fatal("Employee struct not found")
	}

	homeAddr := findField(emp.Fields, "HomeAddr")
	if homeAddr == nil {
		t.Fatal("HomeAddr field not found")
	}
	if homeAddr.Category != types.TypeStruct {
		t.Errorf("HomeAddr: expected TypeStruct, got %d", homeAddr.Category)
	}
	if homeAddr.TypeName != "Address" {
		t.Errorf("HomeAddr: expected TypeName 'Address', got %q", homeAddr.TypeName)
	}

	workAddr := findField(emp.Fields, "WorkAddr")
	if workAddr == nil {
		t.Fatal("WorkAddr field not found")
	}
	if workAddr.Category != types.TypePointer {
		t.Errorf("WorkAddr: expected TypePointer, got %d", workAddr.Category)
	}
	if workAddr.ElemType == nil || workAddr.ElemType.Category != types.TypeStruct {
		t.Error("WorkAddr: expected pointer to struct")
	}
}

func TestFieldIsExported(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package test
type Mixed struct {
	Exported   int
	unexported string
}
`)

	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	s := packages[0].Structs[0]
	exp := findField(s.Fields, "Exported")
	if exp == nil || !exp.IsExported {
		t.Error("Exported field should have IsExported=true")
	}
	unexp := findField(s.Fields, "unexported")
	if unexp == nil || unexp.IsExported {
		t.Error("unexported field should have IsExported=false")
	}
}

func TestTypeExprString(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "complex")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	mixed := findStruct(packages[0].Structs, "Mixed")
	if mixed == nil {
		t.Fatal("Mixed struct not found")
	}

	tests := map[string]string{
		"ID":       "int",
		"Name":     "*string",
		"Tags":     "[]string",
		"Metadata": "map[string]string",
		"Active":   "bool",
		"Scores":   "[]*int",
	}

	for name, expected := range tests {
		f := findField(mixed.Fields, name)
		if f == nil {
			t.Errorf("field %s not found", name)
			continue
		}
		if f.TypeExpr != expected {
			t.Errorf("field %s: expected TypeExpr %q, got %q", name, expected, f.TypeExpr)
		}
	}
}

func BenchmarkScanDir_Simple(b *testing.B) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "testdata", "simple")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanDir(dir)
	}
}

func BenchmarkScanDir_Complex(b *testing.B) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "testdata", "complex")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanDir(dir)
	}
}

func BenchmarkScanDir_All(b *testing.B) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "testdata")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanDir(dir)
	}
}

func TestScanDir_Embedded(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "embedded")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	we := findStruct(pkg.Structs, "WithEmbedded")
	if we == nil {
		t.Fatal("WithEmbedded not found")
	}

	base := findField(we.Fields, "Base")
	if base == nil {
		t.Fatal("embedded Base field not found")
	}
	if !base.IsEmbedded {
		t.Error("Base should be IsEmbedded=true")
	}
	if !base.IsExported {
		t.Error("Base should be IsExported=true")
	}
	if base.Category != types.TypeStruct {
		t.Errorf("Base: expected TypeStruct, got %d", base.Category)
	}

	ts := findField(we.Fields, "Timestamped")
	if ts == nil {
		t.Fatal("embedded Timestamped field not found")
	}
	if !ts.IsEmbedded {
		t.Error("Timestamped should be IsEmbedded=true")
	}

	wep := findStruct(pkg.Structs, "WithEmbeddedPointer")
	if wep == nil {
		t.Fatal("WithEmbeddedPointer not found")
	}
	embBase := findField(wep.Fields, "Base")
	if embBase == nil {
		t.Fatal("embedded *Base field not found")
	}
	if !embBase.IsEmbedded {
		t.Error("*Base should be IsEmbedded=true")
	}
	if embBase.Category != types.TypePointer {
		t.Errorf("*Base: expected TypePointer, got %d", embBase.Category)
	}
}

func TestScanDir_Selector(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "selector")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	event := findStruct(packages[0].Structs, "Event")
	if event == nil {
		t.Fatal("Event struct not found")
	}

	createdAt := findField(event.Fields, "CreatedAt")
	if createdAt == nil {
		t.Fatal("CreatedAt field not found")
	}
	if createdAt.Category != types.TypeStruct {
		t.Errorf("CreatedAt: expected TypeStruct, got %d", createdAt.Category)
	}
	if createdAt.PackageName != "time" {
		t.Errorf("CreatedAt: expected PackageName 'time', got %q", createdAt.PackageName)
	}
	if createdAt.TypeName != "Time" {
		t.Errorf("CreatedAt: expected TypeName 'Time', got %q", createdAt.TypeName)
	}
	if createdAt.TypeExpr != "time.Time" {
		t.Errorf("CreatedAt: expected TypeExpr 'time.Time', got %q", createdAt.TypeExpr)
	}

	updatedAt := findField(event.Fields, "UpdatedAt")
	if updatedAt == nil {
		t.Fatal("UpdatedAt field not found")
	}
	if updatedAt.Category != types.TypePointer {
		t.Errorf("UpdatedAt: expected TypePointer, got %d", updatedAt.Category)
	}
	if updatedAt.ElemType == nil || updatedAt.ElemType.Category != types.TypeStruct {
		t.Error("UpdatedAt: expected pointer to struct")
	}
	if updatedAt.ElemType != nil && updatedAt.ElemType.PackageName != "time" {
		t.Errorf("UpdatedAt elem: expected PackageName 'time', got %q", updatedAt.ElemType.PackageName)
	}
}

func TestScanDir_Edgecase(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "edgecase")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]

	wi := findStruct(pkg.Structs, "WithInterface")
	if wi == nil {
		t.Fatal("WithInterface not found")
	}
	data := findField(wi.Fields, "Data")
	if data == nil {
		t.Fatal("Data field not found")
	}
	if data.Category != types.TypeInterface {
		t.Errorf("Data: expected TypeInterface, got %d", data.Category)
	}

	wc := findStruct(pkg.Structs, "WithChannel")
	if wc == nil {
		t.Fatal("WithChannel not found")
	}
	ch := findField(wc.Fields, "Ch")
	if ch == nil {
		t.Fatal("Ch field not found")
	}
	if ch.Category != types.TypeChannel {
		t.Errorf("Ch: expected TypeChannel, got %d", ch.Category)
	}

	wu := findStruct(pkg.Structs, "withUnexported")
	if wu == nil {
		t.Fatal("withUnexported not found")
	}
	name := findField(wu.Fields, "name")
	if name == nil {
		t.Fatal("name field not found")
	}
	if name.IsExported {
		t.Error("name should be IsExported=false")
	}

	mn := findStruct(pkg.Structs, "MultiName")
	if mn == nil {
		t.Fatal("MultiName not found")
	}
	if len(mn.Fields) != 3 {
		t.Errorf("MultiName: expected 3 fields, got %d", len(mn.Fields))
	}
	for _, fname := range []string{"X", "Y", "Z"} {
		f := findField(mn.Fields, fname)
		if f == nil {
			t.Errorf("MultiName: field %s not found", fname)
		}
	}

	aop := findStruct(pkg.Structs, "ArrayOfPointers")
	if aop == nil {
		t.Fatal("ArrayOfPointers not found")
	}
	ptrs := findField(aop.Fields, "Ptrs")
	if ptrs == nil {
		t.Fatal("Ptrs field not found")
	}
	if ptrs.Category != types.TypeArray {
		t.Errorf("Ptrs: expected TypeArray, got %d", ptrs.Category)
	}
	if ptrs.ElemCategory != types.TypePointer {
		t.Errorf("Ptrs: expected elem TypePointer, got %d", ptrs.ElemCategory)
	}

	som := findStruct(pkg.Structs, "SliceOfMaps")
	if som == nil {
		t.Fatal("SliceOfMaps not found")
	}
	maps := findField(som.Fields, "Maps")
	if maps == nil {
		t.Fatal("Maps field not found")
	}
	if maps.Category != types.TypeSlice {
		t.Errorf("Maps: expected TypeSlice, got %d", maps.Category)
	}
	if maps.ElemCategory != types.TypeMap {
		t.Errorf("Maps: expected elem TypeMap, got %d", maps.ElemCategory)
	}
}

func TestScanDir_Multifile(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "multifile")
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.Name != "multifile" {
		t.Errorf("expected package name 'multifile', got %q", pkg.Name)
	}
	if len(pkg.GoFiles) != 2 {
		t.Errorf("expected 2 Go files, got %d", len(pkg.GoFiles))
	}
	if len(pkg.Structs) != 2 {
		t.Errorf("expected 2 structs, got %d", len(pkg.Structs))
	}

	foo := findStruct(pkg.Structs, "Foo")
	bar := findStruct(pkg.Structs, "Bar")
	if foo == nil || bar == nil {
		t.Error("expected both Foo and Bar structs")
	}
}

func TestScanDir_InvalidGoFile(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "bad.go", `package bad
type Foo struct { this is not valid go }
`)
	_, err := ScanDir(dir)
	if err == nil {
		t.Fatal("expected error for invalid Go file")
	}
}

func TestScanDir_SkipsNonGoFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "good.go", `package test
type Foo struct { X int }
`)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)

	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	if len(packages[0].Structs) != 1 {
		t.Errorf("expected 1 struct, got %d", len(packages[0].Structs))
	}
}

func TestTypeExprString_AllBranches(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package test
type AllTypes struct {
	A chan int
	B interface{}
	C *[]string
	D map[int]bool
}
`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	s := packages[0].Structs[0]

	ch := findField(s.Fields, "A")
	if ch == nil || !strings.Contains(ch.TypeExpr, "chan") {
		t.Errorf("chan field: expected TypeExpr containing 'chan', got %q", ch.TypeExpr)
	}

	iface := findField(s.Fields, "B")
	if iface == nil || iface.TypeExpr != "interface{}" {
		t.Errorf("interface field: expected 'interface{}', got %q", iface.TypeExpr)
	}

	ptrSlice := findField(s.Fields, "C")
	if ptrSlice == nil || ptrSlice.TypeExpr != "*[]string" {
		t.Errorf("pointer-to-slice field: expected '*[]string', got %q", ptrSlice.TypeExpr)
	}

	m := findField(s.Fields, "D")
	if m == nil || m.TypeExpr != "map[int]bool" {
		t.Errorf("map field: expected 'map[int]bool', got %q", m.TypeExpr)
	}
}

func TestIsExportedTypeExpr(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package test
import "time"
type Mixed struct {
	Exported   int
	unexported string
	PtrExport  *time.Time
	ptrUnexp   *int
}
`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	s := packages[0].Structs[0]

	tests := []struct {
		name     string
		exported bool
	}{
		{"Exported", true},
		{"unexported", false},
		{"PtrExport", true},
		{"ptrUnexp", false},
	}

	for _, tt := range tests {
		f := findField(s.Fields, tt.name)
		if f == nil {
			t.Errorf("field %s not found", tt.name)
			continue
		}
		if f.IsExported != tt.exported {
			t.Errorf("field %s: expected IsExported=%v, got %v", tt.name, tt.exported, f.IsExported)
		}
	}
}

func TestEmbeddedFieldName_Selector(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package test
import "time"
type Event struct {
	time.Time
	Name string
}
`)
	packages, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}

	s := packages[0].Structs[0]
	embedded := findField(s.Fields, "Time")
	if embedded == nil {
		t.Fatal("embedded time.Time field not found (expected name 'Time')")
	}
	if !embedded.IsEmbedded {
		t.Error("expected IsEmbedded=true")
	}
	if embedded.PackageName != "time" {
		t.Errorf("expected PackageName 'time', got %q", embedded.PackageName)
	}
}

func findStruct(structs []types.StructInfo, name string) *types.StructInfo {
	for i := range structs {
		if structs[i].Name == name {
			return &structs[i]
		}
	}
	return nil
}

func findField(fields []types.FieldInfo, name string) *types.FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func sortedStructNames(structs []types.StructInfo) []string {
	names := make([]string, len(structs))
	for i, s := range structs {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func writeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
