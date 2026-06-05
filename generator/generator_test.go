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

func scanPackage(t *testing.T, subDir string) types.PackageInfo {
	t.Helper()
	dir := filepath.Join(testdataDir(t), subDir)
	packages, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	return packages[0]
}

func TestGenerate_Simple(t *testing.T) {
	pkg := scanPackage(t, "simple")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "package simple")
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
	pkg := scanPackage(t, "simple")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(code, "github.com/451008604/deepcopy-gen/deepcopy") {
		t.Error("simple package should not import deepcopy helper")
	}
}

func TestGenerate_Complex_PointerFields(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithPointer) DeepCopy() *WithPointer")
	assertContains(t, code, "out.Name = dc.CopyPtr(in.Name)")
	assertContains(t, code, "out.Value = dc.CopyPtr(in.Value)")
	assertContains(t, code, "out.Data = dc.CopyPtr(in.Data)")
	assertContains(t, code, "out.Ref = dc.CopyDoublePtr(in.Ref)")
}

func TestGenerate_Complex_SliceFields(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithSlice) DeepCopy() *WithSlice")
	assertContains(t, code, "out.Names = dc.CopySlice(in.Names)")
	assertContains(t, code, "out.Numbers = dc.CopySlice(in.Numbers)")
	assertContains(t, code, "out.Matrix = dc.CopySliceSlice(in.Matrix)")
	assertContains(t, code, "out.Ptrs = dc.CopySlicePtr(in.Ptrs)")
}

func TestGenerate_Complex_MapFields(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithMap) DeepCopy() *WithMap")
	assertContains(t, code, "out.Labels = dc.CopyMap(in.Labels)")
	assertContains(t, code, "out.Scores = dc.CopyMap(in.Scores)")
	assertContains(t, code, "out.IntKeyMap = dc.CopyMap(in.IntKeyMap)")
}

func TestGenerate_Complex_MapWithPointerValues(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.PtrMap = dc.CopyMapPtr(in.PtrMap)")
}

func TestGenerate_Complex_MapWithSliceValues(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Nested = dc.CopyMapSlice(in.Nested)")
}

func TestGenerate_Nested(t *testing.T) {
	pkg := scanPackage(t, "nested")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Employee) DeepCopy() *Employee")
	assertContains(t, code, "func (in *Department) DeepCopy() *Department")
	assertContains(t, code, "func (in *Node) DeepCopy() *Node")

	assertContains(t, code, "out.WorkAddr = dc.CopyPtr(in.WorkAddr)")
	assertContains(t, code, "out.Emails = dc.CopySlice(in.Emails)")
}

func TestGenerate_Empty(t *testing.T) {
	pkg := scanPackage(t, "empty")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Empty) DeepCopy() *Empty")
	assertContains(t, code, "func (in *Marker) DeepCopy() *Marker")
}

func TestGenerate_NilReceiver(t *testing.T) {
	pkg := scanPackage(t, "simple")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "if in == nil {\n\t\treturn nil\n\t}")
}

func TestGenerate_OutputPath(t *testing.T) {
	path := OutputPath("/some/dir")
	expected := filepath.Join("/some/dir", "structinfo.go")
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
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Ptrs = dc.CopySlicePtr(in.Ptrs)")
}

func TestGenerate_SliceOfSlices(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Matrix = dc.CopySliceSlice(in.Matrix)")
}

func TestGenerate_MixedStruct(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Mixed) DeepCopy() *Mixed")
	assertContains(t, code, "out.Name = dc.CopyPtr(in.Name)")
	assertContains(t, code, "out.Tags = dc.CopySlice(in.Tags)")
	assertContains(t, code, "out.Metadata = dc.CopyMap(in.Metadata)")
	assertContains(t, code, "out.Scores = dc.CopySlicePtr(in.Scores)")
}

func TestGenerate_DepartmentSliceOfStructs(t *testing.T) {
	pkg := scanPackage(t, "nested")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Members = dc.CopySlice(in.Members)")
	assertContains(t, code, "out.Locations = dc.CopySlicePtr(in.Locations)")
}

func TestGenerate_NodeSelfReferential(t *testing.T) {
	pkg := scanPackage(t, "nested")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "visited")
	assertContains(t, code, "out.Children[i] = v.deepcopy(visited)")
	assertContains(t, code, "out.Parent = in.Parent.deepcopy(visited)")
}

func TestGenerate_InterfaceMapValue(t *testing.T) {
	pkg := scanPackage(t, "nested")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "out.Metadata = dc.DeepCopyAny(in.Metadata).(map[string]interface{})")
}

func TestGenerate_HelperImportPresent(t *testing.T) {
	pkg := scanPackage(t, "complex")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, `dc "github.com/451008604/deepcopy-gen/deepcopy"`)
}

func TestGenerate_Embedded(t *testing.T) {
	pkg := scanPackage(t, "embedded")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithEmbedded) DeepCopy() *WithEmbedded")
	assertContains(t, code, "func (in *WithEmbeddedPointer) DeepCopy() *WithEmbeddedPointer")
	assertContains(t, code, "out.Base = dc.CopyPtr(in.Base)")
}

func TestGenerate_Selector(t *testing.T) {
	pkg := scanPackage(t, "selector")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *Event) DeepCopy() *Event")
	assertContains(t, code, "out.UpdatedAt = dc.CopyPtr(in.UpdatedAt)")
}

func TestGenerate_InterfaceField(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithInterface) DeepCopy() *WithInterface")
	assertContains(t, code, "dc.DeepCopyAny(in.Data)")
}

func TestGenerate_ChannelField(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithChannel) DeepCopy() *WithChannel")
}

func TestGenerate_ArrayOfPointers(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *ArrayOfPointers) DeepCopy() *ArrayOfPointers")
	assertContains(t, code, "for i := range in.Ptrs")
	assertContains(t, code, "if in.Ptrs[i] != nil")
	assertContains(t, code, "out.Ptrs[i] = new(int)")
	assertContains(t, code, "*out.Ptrs[i] = *in.Ptrs[i]")
}

func TestGenerate_SliceOfMaps(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *SliceOfMaps) DeepCopy() *SliceOfMaps")
	assertContains(t, code, "out.Maps = dc.CopySliceMap[string, int](in.Maps)")
}

func TestGenerate_MultiNameFields(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *MultiName) DeepCopy() *MultiName")
}

func TestGenerate_UnexportedStruct(t *testing.T) {
	pkg := scanPackage(t, "edgecase")
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
	pkg := scanPackage(t, "simple")
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
	pkg := scanPackage(t, "edgecase")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)

	expected := []string{
		"WithInterface", "WithChannel", "withUnexported",
		"MultiName", "ArrayOfPointers", "SliceOfMaps",
	}
	for _, name := range expected {
		assertContains(t, code, "func (in *"+name+") DeepCopy() *"+name)
	}
}

func TestGenerate_SelfRef_TreeNode(t *testing.T) {
	pkg := scanPackage(t, "selfref")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *TreeNode) DeepCopy() *TreeNode")
	assertContains(t, code, "visited")
	assertContains(t, code, "func (in *TreeNode) deepcopy(visited map[any]any) *TreeNode")
	assertContains(t, code, "out.Children[i] = v.deepcopy(visited)")
	assertContains(t, code, "out.Parent = in.Parent.deepcopy(visited)")
}

func TestGenerate_SelfRef_LinkedNode(t *testing.T) {
	pkg := scanPackage(t, "selfref")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *LinkedNode) deepcopy(visited map[any]any) *LinkedNode")
	assertContains(t, code, "out.Next = in.Next.deepcopy(visited)")
}

func TestGenerate_SelfRef_TreeWithMap(t *testing.T) {
	pkg := scanPackage(t, "selfref")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *TreeWithMap) deepcopy(visited map[any]any) *TreeWithMap")
}

func TestGenerate_Iface_WithInterface(t *testing.T) {
	pkg := scanPackage(t, "iface")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, "func (in *WithInterface) DeepCopy() *WithInterface")
	assertContains(t, code, "out.Data = dc.DeepCopyAny(in.Data).(interface{})")
	assertContains(t, code, "out.Config = dc.DeepCopyAny(in.Config).(interface{})")
}

func TestGenerate_Iface_InterfaceSlice(t *testing.T) {
	pkg := scanPackage(t, "iface")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *InterfaceSlice) DeepCopy() *InterfaceSlice")
	assertContains(t, code, "out.Items = dc.DeepCopyAny(in.Items).([]interface{})")
}

func TestGenerate_Iface_InterfaceMap(t *testing.T) {
	pkg := scanPackage(t, "iface")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *InterfaceMap) DeepCopy() *InterfaceMap")
	assertContains(t, code, "out.Values = dc.DeepCopyAny(in.Values).(map[string]interface{})")
}

func TestGenerate_Iface_NestedInterface(t *testing.T) {
	pkg := scanPackage(t, "iface")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *NestedInterface) DeepCopy() *NestedInterface")
	assertContains(t, code, "out.Meta = dc.DeepCopyAny(in.Meta).(map[string]interface{})")
	assertContains(t, code, "out.Tags = dc.DeepCopyAny(in.Tags).([]interface{})")
}

func TestGenerate_CrossPkg_Player(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertValidGo(t, code)
	assertContains(t, code, `"github.com/451008604/deepcopy-gen/testdata/external"`)
	assertContains(t, code, "func (in *Player) DeepCopy() *Player")
	assertContains(t, code, "out.AccountInfo = dc.DeepCopyAny(in.AccountInfo).(external.AccountInfo)")
}

func TestGenerate_CrossPkg_PlayerWithExtra(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *PlayerWithExtra) DeepCopy() *PlayerWithExtra")
	assertContains(t, code, "out.AccountInfo = dc.DeepCopyAny(in.AccountInfo).(external.AccountInfo)")
	assertContains(t, code, "out.AccountExtra = dc.DeepCopyAny(in.AccountExtra).(external.AccountExtra)")
}

func TestGenerate_CrossPkg_MultiExternalEmbed(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, `"github.com/451008604/deepcopy-gen/testdata/gamecore"`)
	assertContains(t, code, "func (in *MultiExternalEmbed) DeepCopy() *MultiExternalEmbed")
	assertContains(t, code, "out.AccountInfo = dc.DeepCopyAny(in.AccountInfo).(external.AccountInfo)")
	assertContains(t, code, "out.GameItem = dc.DeepCopyAny(in.GameItem).(external.GameItem)")
	assertContains(t, code, "out.Character = dc.DeepCopyAny(in.Character).(gamecore.Character)")
}

func TestGenerate_CrossPkg_SliceOfExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *SliceOfExternal) DeepCopy() *SliceOfExternal")
	assertContains(t, code, "out.Items = dc.CopySlice(in.Items)")
}

func TestGenerate_CrossPkg_SliceOfExternalPtr(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *SliceOfExternalPtr) DeepCopy() *SliceOfExternalPtr")
	assertContains(t, code, "out.Items = dc.CopySlicePtr(in.Items)")
}

func TestGenerate_CrossPkg_MapOfExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *MapOfExternal) DeepCopy() *MapOfExternal")
	assertContains(t, code, "out.Items = dc.CopyMapPtr(in.Items)")
}

func TestGenerate_CrossPkg_ExternalWithPointerToExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithPointerToExternal) DeepCopy() *ExternalWithPointerToExternal")
	assertContains(t, code, "out.Owner = dc.CopyPtr(in.Owner)")
	assertContains(t, code, "out.Items = dc.CopySlicePtr(in.Items)")
}

func TestGenerate_CrossPkg_ExternalWithSliceOfExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithSliceOfExternal) DeepCopy() *ExternalWithSliceOfExternal")
	assertContains(t, code, "out.Inventory = dc.CopySlice(in.Inventory)")
	assertContains(t, code, "out.History = dc.CopySlicePtr(in.History)")
}

func TestGenerate_CrossPkg_ExternalWithMapOfExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithMapOfExternal) DeepCopy() *ExternalWithMapOfExternal")
	assertContains(t, code, "out.Inventory = dc.CopyMap(in.Inventory)")
	assertContains(t, code, "out.LastActive = dc.CopyMapPtr(in.LastActive)")
}

func TestGenerate_CrossPkg_NestedExternal(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *NestedExternal) DeepCopy() *NestedExternal")
	assertContains(t, code, "out.GameState = dc.DeepCopyAny(in.GameState).(external.GameState)")
}

func TestGenerate_CrossPkg_ExternalWithLocalSlice(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
	code, err := Generate(pkg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "func (in *ExternalWithLocalSlice) DeepCopy() *ExternalWithLocalSlice")
	assertContains(t, code, "out.Friends = dc.CopySlice(in.Friends)")
	assertContains(t, code, "out.Items = dc.CopySlice(in.Items)")
}

func TestGenerate_CrossPkg_AllStructs(t *testing.T) {
	pkg := scanPackage(t, "crosspkg")
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
