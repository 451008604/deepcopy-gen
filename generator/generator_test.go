package generator

import (
	"github.com/451008604/deepcopy-gen/scanner"
	"github.com/451008604/deepcopy-gen/types"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
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

func scanPackage(t *testing.T) types.PackageInfo {
	t.Helper()
	dir := testdataDir(t)
	packages, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	for _, pkg := range packages {
		if pkg.Name == "testdata" {
			return pkg
		}
	}
	t.Fatal("testdata package not found")
	return types.PackageInfo{}
}

func TestGenerate_Simple(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "package testdata")
	assertContains(t, code, "func (in *Point) DeepCopy() *Point")
	assertContains(t, code, "func (in *Person) DeepCopy() *Person")
	assertContains(t, code, "func (in *Config) DeepCopy() *Config")
	assertContains(t, code, "if in == nil")
	assertContains(t, code, "return nil")
	assertContains(t, code, "out := new(Point)")
	assertContains(t, code, "*out = *in")
	assertContains(t, code, "DO NOT EDIT")
}

func TestGenerate_SimpleNoHelperImport(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, `dc "github.com/451008604/deepcopy-gen/deepcopy"`)
}

func TestGenerate_Complex_PointerFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithPointer) DeepCopy() *WithPointer")
	assertContains(t, code, "out.Name = dc.CopyPtr(v, in.Name)")
	assertContains(t, code, "out.Value = dc.CopyPtr(v, in.Value)")
	assertContains(t, code, "out.Data = new([]byte)")
	assertContains(t, code, "out.Ref = dc.CopyDoublePtr(v, in.Ref)")
}

func TestGenerate_Complex_SliceFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithSlice) DeepCopy() *WithSlice")
	assertContains(t, code, "out.Names = dc.CopySlice(in.Names)")
	assertContains(t, code, "out.Numbers = dc.CopySlice(in.Numbers)")
	assertContains(t, code, "out.Matrix[i] = dc.CopySlice(in.Matrix[i])")
	assertContains(t, code, "out.Ptrs = dc.CopySlicePtr(v, in.Ptrs)")
}

func TestGenerate_Complex_MapFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithMap) DeepCopy() *WithMap")
	assertContains(t, code, "out.Labels = maps.Clone(in.Labels)")
	assertContains(t, code, "out.Scores = maps.Clone(in.Scores)")
	assertContains(t, code, "out.IntKeyMap = maps.Clone(in.IntKeyMap)")
}

func TestGenerate_Complex_MapWithPointerValues(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.PtrMap = dc.CopyMapPtr(v, in.PtrMap)")
}

func TestGenerate_Complex_MapWithSliceValues(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "cp = dc.CopySlice(val)")
	assertContains(t, code, "out.Nested[k] = cp")
}

func TestGenerate_Nested(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Employee) DeepCopy() *Employee")
	assertContains(t, code, "func (in *Department) DeepCopy() *Department")
	assertContains(t, code, "func (in *Node) DeepCopy() *Node")
	assertContains(t, code, "out.WorkAddr = in.WorkAddr.deepcopy(v)")
	assertContains(t, code, "out.Emails = dc.CopySlice(in.Emails)")
}

func TestGenerate_Empty(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Empty) DeepCopy() *Empty")
	assertContains(t, code, "func (in *Marker) DeepCopy() *Marker")
}

func TestGenerate_NilReceiver(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "if in == nil {\n\t\treturn nil\n\t}")
}

func TestGenerate_OutputPath(t *testing.T) {
	path := OutputPath("/some/dir")
	expected := filepath.Join("/some/dir", "deepcopy.dc.go")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestValidateGenerated_Valid(t *testing.T) {
	code := `package test

type Foo struct{ X int }

func (in *Foo) DeepCopy() *Foo {
	if in == nil {
		return nil
	}
	out := new(Foo)
	*out = *in
	return out
}
`
	if err := ValidateGenerated(code); err != nil {
		t.Errorf("expected valid code, got error: %v", err)
	}
}

func TestValidateGenerated_Invalid(t *testing.T) {
	code := `package test
func broken( {`
	if err := ValidateGenerated(code); err == nil {
		t.Error("expected error for invalid code")
	}
}

func TestGenerate_SliceOfPointers(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Ptrs = dc.CopySlicePtr(v, in.Ptrs)")
}

func TestGenerate_SliceOfSlices(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Matrix[i] = dc.CopySlice(in.Matrix[i])")
}

func TestGenerate_MixedStruct(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Mixed) DeepCopy() *Mixed")
	assertContains(t, code, "out.Name = dc.CopyPtr(v, in.Name)")
	assertContains(t, code, "out.Tags = dc.CopySlice(in.Tags)")
	assertContains(t, code, "out.Metadata = maps.Clone(in.Metadata)")
	assertContains(t, code, "out.Scores = dc.CopySlicePtr(v, in.Scores)")
}

func TestGenerate_DepartmentSliceOfStructs(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Members[i] = *in.Members[i].deepcopy(v)")
	assertContains(t, code, "out.Locations = dc.CopySlicePtr(v, in.Locations)")
}

func TestGenerate_NodeSelfReferential(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Children = dc.CopySlicePtr(v, in.Children)")
	assertContains(t, code, "out.Parent = in.Parent.deepcopy(v)")
}

func TestGenerate_HelperImportPresent(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, `dc "github.com/451008604/deepcopy-gen/deepcopy"`)
}

func TestGenerate_Embedded(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithEmbedded) DeepCopy() *WithEmbedded")
	assertContains(t, code, "func (in *WithEmbeddedPointer) DeepCopy() *WithEmbeddedPointer")
	assertContains(t, code, "out.Base = in.Base.deepcopy(v)")
}

func TestGenerate_Selector(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Event) DeepCopy() *Event")
	assertContains(t, code, "out.UpdatedAt = dc.CopyPtr(v, in.UpdatedAt)")
}

func TestGenerate_InterfaceField(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithInterface) DeepCopy() *WithInterface")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_ChannelField(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *WithChannel) DeepCopy() *WithChannel")
}

func TestGenerate_ArrayOfPointers(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *ArrayOfPointers) DeepCopy() *ArrayOfPointers")
	assertContains(t, code, "for i := range in.Ptrs")
	assertContains(t, code, "out.Ptrs[i] = dc.CopyPtr(v, in.Ptrs[i])")
}

func TestGenerate_SliceOfMaps(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *SliceOfMaps) DeepCopy() *SliceOfMaps")
	assertContains(t, code, "out.Maps[i] = maps.Clone(in.Maps[i])")
}

func TestGenerate_MultiNameFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *MultiName) DeepCopy() *MultiName")
}

func TestGenerate_UnexportedStruct(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *withUnexported) DeepCopy() *withUnexported")
}

func TestAssemble_WithImports(t *testing.T) {
	g := &genState{
		pkgName: "test",
		imports: map[string]bool{"fmt": true, "time": true},
	}
	methods := []string{"func foo() {}\n"}
	result := g.assemble(methods)

	assertContains(t, result, "package test")
	assertContains(t, result, `"fmt"`)
	assertContains(t, result, `"time"`)
	assertContains(t, result, "func foo() {}")
}

func TestAssemble_WithHelperImport(t *testing.T) {
	g := &genState{
		pkgName:     "test",
		imports:     map[string]bool{},
		needsHelper: true,
	}
	methods := []string{"func foo() {}\n"}
	result := g.assemble(methods)

	assertContains(t, result, `dc "github.com/451008604/deepcopy-gen/deepcopy"`)
}

func TestGenDeepCopy_NoFieldsNeedingCopy(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Point) DeepCopy() *Point")
	assertContains(t, code, "*out = *in")
	assertContains(t, code, "return out")
}

func TestGenerate_AllEdgecaseStructs(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)

	expected := []string{
		"EdgeInterface", "WithChannel", "withUnexported",
		"MultiName", "ArrayOfPointers", "SliceOfMaps",
	}
	for _, name := range expected {
		assertContains(t, code, "func (in *"+name+") DeepCopy() *"+name)
	}
}

func TestGenerate_SelfRef_TreeNode(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *TreeNode) DeepCopy() *TreeNode")
	assertContains(t, code, "func (in *TreeNode) deepcopy(v dc.Visited) *TreeNode")
	assertContains(t, code, "out.Children = dc.CopySlicePtr(v, in.Children)")
	assertContains(t, code, "out.Parent = in.Parent.deepcopy(v)")
}

func TestGenerate_SelfRef_LinkedNode(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *LinkedNode) deepcopy(v dc.Visited) *LinkedNode")
	assertContains(t, code, "out.Next = in.Next.deepcopy(v)")
}

func TestGenerate_SelfRef_TreeWithMap(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *TreeWithMap) deepcopy(v dc.Visited) *TreeWithMap")
}

func TestGenerate_Iface_WithInterface(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithInterface) DeepCopy() *WithInterface")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_Iface_InterfaceSlice(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *InterfaceSlice) DeepCopy() *InterfaceSlice")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_Iface_InterfaceMap(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *InterfaceMap) DeepCopy() *InterfaceMap")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_Iface_NestedInterface(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *NestedInterface) DeepCopy() *NestedInterface")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_InterfaceMapValue(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *Node) DeepCopy() *Node")
	assertNotContains(t, code, "DeepCopyAny")
}

func TestGenerate_CrossPkg_Player(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, `"github.com/451008604/deepcopy-gen/testdata/external"`)
	assertContains(t, code, "func (in *Player) DeepCopy() *Player")
	assertContains(t, code, "out.AccountInfo = *dc.CopyPtr(v, &in.AccountInfo)")
}

func TestGenerate_CrossPkg_PlayerWithExtra(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *PlayerWithExtra) DeepCopy() *PlayerWithExtra")
	assertContains(t, code, "out.AccountInfo = *dc.CopyPtr(v, &in.AccountInfo)")
	assertContains(t, code, "out.AccountExtra = *dc.CopyPtr(v, &in.AccountExtra)")
}

func TestGenerate_CrossPkg_MultiExternalEmbed(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *MultiExternalEmbed) DeepCopy() *MultiExternalEmbed")
	assertContains(t, code, "out.AccountInfo = *dc.CopyPtr(v, &in.AccountInfo)")
	assertContains(t, code, "out.GameItem = *dc.CopyPtr(v, &in.GameItem)")
	assertContains(t, code, "out.Character = *dc.CopyPtr(v, &in.Character)")
}

func TestGenerate_CrossPkg_SliceOfExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *SliceOfExternal) DeepCopy() *SliceOfExternal")
	assertContains(t, code, "out.Items[i] = *dc.CopyPtr(v, &in.Items[i])")
}

func TestGenerate_CrossPkg_SliceOfExternalPtr(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *SliceOfExternalPtr) DeepCopy() *SliceOfExternalPtr")
	assertContains(t, code, "out.Items = dc.CopySlicePtr(v, in.Items)")
}

func TestGenerate_CrossPkg_MapOfExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *MapOfExternal) DeepCopy() *MapOfExternal")
	assertContains(t, code, "out.Items = dc.CopyMapPtr(v, in.Items)")
}

func TestGenerate_CrossPkg_ExternalWithPointerToExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithPointerToExternal) DeepCopy() *ExternalWithPointerToExternal")
	assertContains(t, code, "out.Owner = dc.CopyPtr(v, in.Owner)")
	assertContains(t, code, "out.Items = dc.CopySlicePtr(v, in.Items)")
}

func TestGenerate_CrossPkg_ExternalWithSliceOfExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithSliceOfExternal) DeepCopy() *ExternalWithSliceOfExternal")
	assertContains(t, code, "out.Inventory[i] = *dc.CopyPtr(v, &in.Inventory[i])")
	assertContains(t, code, "out.History = dc.CopySlicePtr(v, in.History)")
}

func TestGenerate_CrossPkg_ExternalWithMapOfExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithMapOfExternal) DeepCopy() *ExternalWithMapOfExternal")
	assertContains(t, code, "out.Inventory[k] = *dc.CopyPtr(v, &val)")
	assertContains(t, code, "out.LastActive = dc.CopyMapPtr(v, in.LastActive)")
}

func TestGenerate_CrossPkg_NestedExternal(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *NestedExternal) DeepCopy() *NestedExternal")
	assertContains(t, code, "out.GameState = *dc.CopyPtr(v, &in.GameState)")
}

func TestGenerate_CrossPkg_ExternalWithLocalSlice(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithLocalSlice) DeepCopy() *ExternalWithLocalSlice")
	assertContains(t, code, "out.Friends[i] = *in.Friends[i].deepcopy(v)")
	assertContains(t, code, "out.Items[i] = *dc.CopyPtr(v, &in.Items[i])")
}

func TestGenerate_CrossPkg_AllStructs(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)

	expectedStructs := []string{
		"Player", "Team", "GameSession", "PlayerWithExtra",
		"PointerEmbed", "SliceOfExternal", "MapOfExternal", "SliceOfExternalPtr",
		"MultiExternalEmbed", "ExternalWithPointerToExternal",
		"ExternalWithSliceOfExternal", "ExternalWithMapOfExternal",
		"NestedExternal", "ExternalWithLocalSlice",
	}
	for _, name := range expectedStructs {
		assertContains(t, code, "func (in *"+name+") DeepCopy() *"+name)
	}
}

func TestGenerate_NamedTypes_PtrFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	assertValidGo(t, code)
	// Named basic alias pointer: shallow via dc.CopyPtr (no .deepcopy call).
	assertContains(t, code, "out.S = dc.CopyPtr(v, in.S)")
	// Named array/map pointer: direct .deepcopy(v) (same-package method).
	assertContains(t, code, "out.M = in.M.deepcopy(v)")
	assertContains(t, code, "out.I = in.I.deepcopy(v)")
}

func TestGenerate_NamedTypes_ValueFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Named slice/map/array value fields now invoke the generated method
	// instead of relying on the shallow *out = *in.
	assertContains(t, code, "out.T = *in.T.deepcopy(v)")
	assertContains(t, code, "out.M = *in.M.deepcopy(v)")
	assertContains(t, code, "out.C = *in.C.deepcopy(v)")
}

func TestGenerate_NamedTypes_PtrToComposite(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Pointer-to-slice and pointer-to-map emit inline element-wise copies.
	assertContains(t, code, "out.Basic = new([]string)")
	assertContains(t, code, "(*out.Basic) = dc.CopySlice((*in.Basic))")
	assertContains(t, code, "(*out.Items) = dc.CopySlicePtr(v, (*in.Items))")
	assertContains(t, code, "out.M = new(map[string]int)")
	assertContains(t, code, "(*out.MI) = dc.CopyMapPtr(v, (*in.MI))")
}

func TestGenerate_NamedTypes_MapOfStruct(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Map of struct value emits an inline loop (values are not addressable).
	assertContains(t, code, "out.Same = make(map[string]Item, len(in.Same))")
	assertContains(t, code, "out.Same[k] = *val.deepcopy(v)")
	assertContains(t, code, "out.Cross = make(map[string]external.GameItem, len(in.Cross))")
	assertContains(t, code, "out.Cross[k] = *dc.CopyPtr(v, &val)")
}

func TestGenerate_NamedTypes_AnonStruct(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Anonymous struct: shallow-copy the value, then deep-copy ref fields.
	assertContains(t, code, "out.S = in.S")
	assertContains(t, code, "out.S.N = dc.CopyPtr(v, in.S.N)")
	assertContains(t, code, "out.S.Inner.P = dc.CopyPtr(v, in.S.Inner.P)")
}

func TestGenerate_NamedTypes_StructValueFields(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Same-pkg struct value field now calls .deepcopy (was shallow before).
	assertContains(t, code, "out.Home = *in.Home.deepcopy(v)")
	// Cross-pkg struct value field deep-copies via dc.CopyPtr.
	assertContains(t, code, "out.Owner = *dc.CopyPtr(v, &in.Owner)")
}

func TestGenerate_NamedTypes_MethodsForNamedComposites(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Named slice/array/map types get generated deepcopy methods.
	for _, name := range []string{"Tags", "Meta", "Matrix", "Inventory", "Coords"} {
		assertContains(t, code, "func (in *"+name+") deepcopy(v dc.Visited) *"+name)
	}
	// Named basic alias (Score) must NOT get a method.
	assertNotContains(t, code, "func (in *Score) deepcopy")
}

func TestGenerate_NamedTypes_AnonStructMapValue(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Empty anonymous struct map value (set pattern): shallow copy.
	assertContains(t, code, "out.Members = make(map[int32]struct{}, len(in.Members))")
	assertContains(t, code, "out.Members[k] = val")
	// Non-empty anonymous struct map value: temp var + field-wise deep copy.
	assertContains(t, code, "out.Tagged = make(map[string]struct{ N *int }, len(in.Tagged))")
	assertContains(t, code, "cp := val")
	assertContains(t, code, "cp.N = dc.CopyPtr(v, val.N)")
	assertContains(t, code, "out.Tagged[k] = cp")
}

func TestGenerate_NamedTypes_BasicAliasAndEmptyStructSlices(t *testing.T) {
	pkg := scanPackage(t)
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Named basic alias slice and empty struct slice: dc.CopySlice (no
	// .deepcopy call on method-less element types).
	assertContains(t, code, "out.Scores = dc.CopySlice(in.Scores)")
	assertContains(t, code, "out.Empty = dc.CopySlice(in.Empty)")
	assertNotContains(t, code, "in.Scores[i].deepcopy")
}

func assertValidGo(t *testing.T, code string) {
	t.Helper()
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", code, parser.AllErrors)
	if err != nil {
		t.Errorf("generated code is not valid Go:\n%s\nerror: %v", code, err)
	}
}

func assertContains(t *testing.T, code, substr string) {
	t.Helper()
	if !strings.Contains(code, substr) {
		t.Errorf("generated code does not contain %q\n\ngenerated:\n%s", substr, code)
	}
}

func assertNotContains(t *testing.T, code, substr string) {
	t.Helper()
	if strings.Contains(code, substr) {
		t.Errorf("generated code should not contain %q\n\ngenerated:\n%s", substr, code)
	}
}
