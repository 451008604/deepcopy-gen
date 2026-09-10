package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/451008604/deepcopy-gen/scanner"
)

// 同时编译并执行生成结果，避免字符串断言掩盖类型错误和浅拷贝。
func testGeneratedPackage(t *testing.T, files map[string]string, tests string) {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	files["go.mod"] = fmt.Sprintf("module fixture\n\ngo 1.21\n\nrequire github.com/451008604/deepcopy-gen v0.0.0\nreplace github.com/451008604/deepcopy-gen => %q\n", root)
	for name, src := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	packages, err := scanner.ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		code, err := Generate(pkg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(OutputPath(pkg.Dir), []byte(code), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "syntax_test.go"), []byte("package fixture\nimport \"testing\"\n"+tests), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated package failed: %v\n%s", err, out)
	}
}

func TestSyntax_Expressions(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
const Size = 1 << 1
type Label string
type Reader interface { Read([]byte) (int, error) }
type Expressions struct {
 Tagged []struct { P *int ` + "`json:\"p,omitempty\"`" + ` }
 Callback *func(...int) (string, error)
 Receivers []<-chan int
 Senders map[string]chan<- int
 Readers []interface { Read([]byte) (int, error) }
 Embedded struct { Label; Reader }
 Paren *(struct { P *int })
 Array *[Size + 1]*int
 Hex [0x2]*int
 Blank struct { _ *int; Value *int }
}
`}, `func TestCopy(t *testing.T) {
 n := 3
 in := &Expressions{Array: &[Size+1]*int{&n}, Hex: [0x2]*int{&n}}
 in.Tagged = append(in.Tagged, struct { P *int `+"`json:\"p,omitempty\"`"+` }{&n})
 in.Paren = &struct { P *int }{&n}
 in.Blank.Value = &n
 ch := make(chan int)
 in.Receivers = []<-chan int{ch}; in.Senders = map[string]chan<- int{"x": ch}
 fn := func(v ...int) (string,error) { return "ok",nil }; in.Callback = &fn
 cp := in.DeepCopy()
 *cp.Tagged[0].P = 7; *cp.Paren.P = 8; *cp.Array[0] = 9; *cp.Hex[0] = 10; *cp.Blank.Value = 11
 if n != 3 { t.Fatal("mutable syntax fields share source storage") }
 if cp.Receivers[0] != ch || cp.Senders["x"] != ch { t.Fatal("channel identity changed") }
 if cp.Callback == in.Callback { t.Fatal("function pointer was not copied") }
 if s,_ := (*cp.Callback)(1); s != "ok" { t.Fatal("function changed") }
}`)
}

func TestSyntax_AliasesAndDefinitions(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
type Base struct { Data []int }
type Alias = Base
type StructAlias = struct { Data []int }
type SliceAlias = []map[string]*int
type PointerAlias = *Base
type Defined Base
type MoreDefined Defined
type DefinedSlice SliceAlias
type DefinedPointer *Base
type Aliases struct {
 A Alias; B StructAlias; C SliceAlias; D PointerAlias
 E Defined; F MoreDefined; G DefinedSlice; H DefinedPointer
}
`}, `func TestCopy(t *testing.T) {
 n:=1
 in := &Aliases{A:Base{[]int{1}}, B:StructAlias{[]int{1}}, C:SliceAlias{{"a":&n}}, D:&Base{[]int{1}}, E:Defined{[]int{1}}, F:MoreDefined{[]int{1}}, G:DefinedSlice{{"a":&n}}, H:&Base{[]int{1}}}
 cp:=in.DeepCopy()
 cp.A.Data[0]=9; cp.B.Data[0]=9; *cp.C[0]["a"]=9; cp.D.Data[0]=9
 cp.E.Data[0]=9; cp.F.Data[0]=9; *cp.G[0]["a"]=9; cp.H.Data[0]=9
 if n!=1 || in.A.Data[0]!=1 || in.B.Data[0]!=1 || in.D.Data[0]!=1 || in.E.Data[0]!=1 || in.F.Data[0]!=1 || in.H.Data[0]!=1 {t.Fatal("alias or defined type copied shallowly")}
}`)
}

func TestSyntax_Generics(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
type Box[T any] struct { Value T; Values []T; Next *Box[T] }
type Pair[K comparable, V any] struct { Items map[K][]V }
type List[T any] []T
type Node struct { Data []int }
type GenericPointer[T any] *T
type GenericHolder struct { Box[*Node]; Pair Pair[string,int]; List List[int]; P GenericPointer[*Node] }
`}, `func TestCopy(t *testing.T) {
 in:=&GenericHolder{Box:Box[*Node]{Value:&Node{[]int{1}}}, Pair:Pair[string,int]{map[string][]int{"a":{1}}},List:List[int]{1}}
 in.Box.Next=&in.Box
 in.P=&in.Box.Value
 cp:=in.DeepCopy()
 (*cp.P).Data[0]=9; cp.Value.Data[0]=9; cp.Pair.Items["a"][0]=9; cp.List[0]=9
 if in.Value.Data[0]!=1 || in.Pair.Items["a"][0]!=1 || in.List[0]!=1 {t.Fatal("generic copy leaked")}
 if cp.Box.Next==&in.Box || cp.Box.Next.Next!=cp.Box.Next {t.Fatal("generic cycle lost")}
}`)
}

func TestSyntax_RecursiveCollections(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
type Item struct { Data []int }
type NamedSlice [][]*Item
type NamedMap map[string]map[string][]*Item
type NamedArray [2][]*Item
type Containers struct {
 Slices [][]*Item
 Maps map[string]map[string][]*Item
 Arrays [2][2]*Item
 Anon []struct { Data []int }
 Ptr *[2][]*Item
 Deep ***Item
 NamedSlice NamedSlice
 NamedMap NamedMap
 NamedArray NamedArray
}
`}, `func TestCopy(t *testing.T) {
 item:=&Item{[]int{1}}; p:=&item; pp:=&p
 in:=&Containers{Slices:[][]*Item{{item}},Maps:map[string]map[string][]*Item{"a":{"b":{item}}},Arrays:[2][2]*Item{{item}},Ptr:&[2][]*Item{{item}},Deep:pp,NamedSlice:NamedSlice{{item}},NamedMap:NamedMap{"a":{"b":{item}}},NamedArray:NamedArray{{item}}}
 in.Anon=append(in.Anon,struct{Data []int}{[]int{1}})
 cp:=in.DeepCopy()
 items:=[]*Item{cp.Slices[0][0],cp.Maps["a"]["b"][0],cp.Arrays[0][0],(*cp.Ptr)[0][0],**cp.Deep,cp.NamedSlice[0][0],cp.NamedMap["a"]["b"][0],cp.NamedArray[0][0]}
 for _, v:=range items {v.Data[0]=9}
 cp.Anon[0].Data[0]=9
 if item.Data[0]!=1 || in.Anon[0].Data[0]!=1 {t.Fatal("nested mutable data leaked")}
}`)
}

func TestSyntax_ImportAliases(t *testing.T) {
	testGeneratedPackage(t, map[string]string{
		"types.go": `package fixture
import (same "fixture/one"; dc "time"; helper "github.com/451008604/deepcopy-gen/deepcopy")
type First struct { Values map[same.Key][]*same.Value; Times *[]dc.Time; Seen []struct { Value *int; Cache helper.Visited } }
`,
		"more.go": `package fixture
import same "fixture/two"
type Second struct { Values []same.Value }
`,
		"one/types.go": `package one
type Key string
type Value struct { Data []int }
`,
		"two/types.go": `package two
type Value struct { Data []int }
`,
	}, `func TestNil(t *testing.T) { if (*First)(nil).DeepCopy()!=nil || (*Second)(nil).DeepCopy()!=nil {t.Fatal("nil receiver")} }`)
}

func TestSyntax_NilMapValues(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
type Item struct { Value int }
type Holder struct { Pointers map[string]*Item; Slices map[string][]int }
`}, `func TestCopy(t *testing.T) {
 in:=&Holder{Pointers:map[string]*Item{"nil":nil},Slices:map[string][]int{"nil":nil,"empty":{}}}
 cp:=in.DeepCopy()
 if p,ok:=cp.Pointers["nil"]; !ok || p!=nil {t.Fatal("nil pointer map entry lost")}
 if s,ok:=cp.Slices["nil"]; !ok || s!=nil {t.Fatal("nil slice map entry lost")}
 if cp.Slices["empty"]==nil {t.Fatal("empty slice changed to nil")}
}`)
}

func TestSyntax_CyclesAndSharedPointers(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
import "fixture/other"
type Node struct { Name string; Children []*Node; ByName map[string]*Node; Array [2]*Node; Indirect **Node }
type Shared struct { A, B *int; M1, M2 *map[string][]int; S1, S2 *[]int }
type Cross struct { Items []*other.Node }
`, "other/types.go": `package other
type Node struct { Next *Node; Items []*Node; Value int }
`}, `func TestCycles(t *testing.T) {
 root:=&Node{Name:"root"};child:=&Node{Name:"child"}
 root.Children=[]*Node{root,child,child,nil};root.ByName=map[string]*Node{"root":root,"child":child,"nil":nil};root.Array=[2]*Node{root,child};root.Indirect=&root;child.Children=[]*Node{root}
 cp:=root.DeepCopy()
 if cp==root || cp.Children[0]!=cp || cp.Children[1]!=cp.Children[2] || cp.Children[1].Children[0]!=cp || cp.ByName["root"]!=cp || cp.ByName["child"]!=cp.Children[1] || cp.Array[0]!=cp || *cp.Indirect!=cp {t.Fatal("cycle or shared node identity lost")}
 if len(cp.ByName)!=3 || cp.Children[3]!=nil {t.Fatal("nil collection values changed")}
 if root.DeepCopyWithVisited(nil)==nil {t.Fatal("nil visited map rejected")}
}
func TestShared(t *testing.T) {
 n:=1;m:=map[string][]int{"a":{1}};s:=[]int{1}
 in:=&Shared{&n,&n,&m,&m,&s,&s};cp:=in.DeepCopy()
 if cp.A!=cp.B || cp.A==&n || cp.M1!=cp.M2 || cp.M1==&m || cp.S1!=cp.S2 || cp.S1==&s {t.Fatal("shared pointer identity lost")}
 (*cp.M1)["a"][0]=9;(*cp.S1)[0]=9
 if m["a"][0]!=1 || s[0]!=1 {t.Fatal("pointer to container shares original")}
}
`)
}

func TestSyntax_NamedLeavesAndEmptyContainers(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
 type Number int
 type Pointer *Number
 type Numbers []Number
 type NumberArray [2]Number
 type NumberMap map[string]Number
 type PointerSlice []Pointer
 type LeafContainers struct {
  S Numbers; A NumberArray; M NumberMap
  P *Numbers; PA *NumberArray; PM *NumberMap
  PS PointerSlice; F []func() int; C map[string]<-chan int
  Empty *map[string][]int
 }
`}, `func TestLeaves(t *testing.T) {
 n:=Number(1);s:=Numbers{};a:=NumberArray{1,2};m:=NumberMap{};var nilMap map[string][]int
 in:=&LeafContainers{S:s,A:a,M:m,P:&s,PA:&a,PM:&m,PS:PointerSlice{&n,nil},F:[]func()int{func()int{return 7},nil},C:map[string]<-chan int{"nil":nil},Empty:&nilMap}
 cp:=in.DeepCopy()
 if cp.S==nil || cp.M==nil || cp.P==nil || *cp.P==nil || cp.PM==nil || *cp.PM==nil || cp.Empty==nil || *cp.Empty!=nil {t.Fatal("nil/empty distinctions lost")}
 *cp.PS[0]=9;cp.PA[0]=9
 if n!=1 || a[0]!=1 || cp.F[0]()!=7 || cp.F[1]!=nil || len(cp.C)!=1 {t.Fatal("leaf copy invalid")}
 var zero LeafContainers
 z:=zero.DeepCopy()
 if z.S!=nil || z.M!=nil || z.P!=nil || z.PS!=nil {t.Fatal("zero value changed")}
}`)
}

func TestSyntax_ExternalCycles(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
import "fixture/other"
type Holder struct { A,B *other.Node; Nodes []*other.Node; Values map[string]other.Node }
func NewHolder() *Holder {
 node:=&other.Node{Value:1};node.Next=node
 return &Holder{A:node,B:node,Nodes:[]*other.Node{node},Values:map[string]other.Node{"a":{Value:1},"b":{Value:2}}}
}
`, "other/types.go": `package other
type Node struct { Next *Node; Value int }
`}, `func TestCopy(t *testing.T) {
 in:=NewHolder();cp:=in.DeepCopy()
 if cp.A==in.A || cp.A!=cp.B || cp.A!=cp.Nodes[0] || cp.A.Next!=cp.A {t.Fatal("external cycle or shared reference lost")}
 if cp.Values["a"].Value!=1 || cp.Values["b"].Value!=2 {t.Fatal("map range variable reuse corrupted copies")}
}`)
}

func TestSyntax_GenericDefinitionsAndParameterNames(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
 type Box[T any] struct { T T; Values []T }
 type Derived Box[int]
 type Dict[K comparable,V any] map[K][]V
 type DefinedDict Dict[string,int]
 type Odd[in, out, dc any] struct { Values []in; Item out; dc dc }
 type Data struct { A Derived; B DefinedDict; C Odd[int,int,int] }
`}, `func TestCopy(t *testing.T) {
 in:=&Data{A:Derived{T:1,Values:[]int{1}},B:DefinedDict{"a":{1}},C:Odd[int,int,int]{Values:[]int{1},Item:1,dc:2}}
 cp:=in.DeepCopy();cp.A.Values[0]=9;cp.B["a"][0]=9;cp.C.Values[0]=9
 if in.A.Values[0]!=1 || in.B["a"][0]!=1 || in.C.Values[0]!=1 || cp.A.T!=1 || cp.C.dc!=2 {t.Fatal("generic definition or parameter renaming failed")}
}`)
}

func TestSyntax_OpaqueValues(t *testing.T) {
	testGeneratedPackage(t, map[string]string{"types.go": `package fixture
import "unsafe"
type Box[T any] struct { Value T }
type Key struct { Value int }
type Opaque struct { Value any; Error error; Fn func() int; Ch chan int; Raw unsafe.Pointer; Generic Box[[]int]; Keys map[*Key]int }
`}, `func TestContract(t *testing.T) {
 values:=[]int{1};key:=&Key{Value:1}
 in:=&Opaque{Value:values,Fn:func()int{return 7},Ch:make(chan int),Generic:Box[[]int]{Value:values},Keys:map[*Key]int{key:1}}
 cp:=in.DeepCopy()
 if cp.Ch!=in.Ch || cp.Fn()!=7 || cp.Error!=nil || cp.Raw!=nil || cp.Keys[key]!=1 {t.Fatal("opaque value or key identity changed")}
 cp.Value.([]int)[0]=9
 if values[0]!=9 || cp.Generic.Value[0]!=9 {t.Fatal("opaque values must retain their documented assignment semantics")}
 delete(cp.Keys,key)
 if len(in.Keys)!=1 {t.Fatal("map container shares source")}
}`)
}
