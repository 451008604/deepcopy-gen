package testdata

import (
	"github.com/451008604/deepcopy-gen/testdata/external"
	"github.com/451008604/deepcopy-gen/testdata/gamecore"
	"net"
	"sync"
	"time"
)

// simple
// Point is a simple struct with only value-type fields.
type Point struct {
	X int
	Y int
}

// Person has mixed basic-type fields.
type Person struct {
	Name   string
	Age    int
	Email  string
	Active bool
	Score  float64
}

// Config uses various basic types.
type Config struct {
	Timeout  int64
	Verbose  bool
	MaxRetry int
	Label    string
	Rate     float32
}

// complex
// WithPointer has pointer fields that need nil-safe deep copy.
type WithPointer struct {
	Name  *string
	Value *int
	Data  *[]byte
	Ref   **float64
}

// WithSlice has slice fields of various element types.
type WithSlice struct {
	Names   []string
	Numbers []int
	Matrix  [][]float64
	Ptrs    []*int
}

// WithMap has map fields with different key/value types.
type WithMap struct {
	Labels    map[string]string
	Scores    map[string]int
	Nested    map[string][]int
	PtrMap    map[string]*bool
	IntKeyMap map[int]string
	MailIDs   map[int32]struct{}
}

// WithArray has fixed-size array fields.
type WithArray struct {
	Coords [3]float64
	Matrix [2][2]int
	Tags   [5]string
}

// Mixed has a combination of field types.
type Mixed struct {
	ID       int
	Name     *string
	Tags     []string
	Metadata map[string]string
	Active   bool
	Scores   []*int
}

// nested
// Address is a value-type struct used as a nested field.
type Address struct {
	Street string
	City   string
	Zip    string
}

// Employee embeds Address and has pointer/slice fields.
type Employee struct {
	Name     string
	Age      int
	HomeAddr Address
	WorkAddr *Address
	Emails   []string
	Manager  *Employee
	Tags     map[string]string
}

// Department has nested slices and maps of structs.
type Department struct {
	Name      string
	Head      *Employee
	Members   []Employee
	Locations []*Address
	Budget    map[string]float64
}

// Node is a self-referential struct.
type Node struct {
	Value    int
	Children []*Node
	Parent   *Node
	Metadata map[string]interface{}
}

// empty
// Empty is a struct with no fields.
type Empty struct{}

// Marker is another empty struct used as a tag type.
type Marker struct{}

// embedded
type Base struct {
	ID int
}

type Timestamped struct {
	CreatedAt string
	UpdatedAt string
}

type WithEmbedded struct {
	Base
	Timestamped
	Name string
}

type WithEmbeddedPointer struct {
	*Base
	Name string
}

// selector
type Event struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

// edgecase
type EdgeInterface struct {
	Data interface{}
	Name string
}

type WithChannel struct {
	Ch   chan int
	Name string
}

type withUnexported struct {
	name string
	Age  int
}

type MultiName struct {
	X, Y, Z int
}

type ArrayOfPointers struct {
	Ptrs [3]*int
}

type SliceOfMaps struct {
	Maps []map[string]int
}

// selfref
type TreeNode struct {
	Value    int
	Children []*TreeNode
	Parent   *TreeNode
}

type GraphNode struct {
	Name      string
	Neighbors []*GraphNode
	Metadata  map[string]interface{}
}

type LinkedNode struct {
	Value int
	Next  *LinkedNode
}

type TreeWithMap struct {
	Label    string
	Children map[string]*TreeWithMap
}

// iface
type WithInterface struct {
	Name   string
	Data   interface{}
	Config interface{}
}

type InterfaceSlice struct {
	Items []interface{}
}

type InterfaceMap struct {
	Values map[string]interface{}
}

type NestedInterface struct {
	Meta map[string]interface{}
	Tags []interface{}
}

// crosspkg
type Player struct {
	external.AccountInfo
	Nickname string
	Level    int
}

type Team struct {
	Name    string
	Players []Player
	Captain *Player
}

type GameSession struct {
	Player  *Player
	Team    *Team
	Score   int
	History []map[string]int
}

type PlayerWithExtra struct {
	external.AccountInfo
	external.AccountExtra
	Nickname string
}

type PointerEmbed struct {
	*external.AccountInfo
	Name string
}

type SliceOfExternal struct {
	Items []external.AccountInfo
}

type MapOfExternal struct {
	Items map[string]*external.GameItem
}

type SliceOfExternalPtr struct {
	Items []*external.AccountInfo
}

type MultiExternalEmbed struct {
	external.AccountInfo
	external.GameItem
	gamecore.Character
	Nickname string
}

type ExternalWithPointerToExternal struct {
	Owner *external.AccountInfo
	Items []*external.GameItem
}

type ExternalWithSliceOfExternal struct {
	Inventory []external.GameItem
	History   []*external.AccountInfo
}

type ExternalWithMapOfExternal struct {
	Inventory  map[string]external.GameItem
	LastActive map[string]*external.AccountInfo
}

type compensationHeap []external.GameState
type NestedExternal struct {
	external.GameState
	MainCharacter gamecore.Character
	heap          *compensationHeap
	heap2         compensationHeap
	compensationHeap
	*net.TCPListener
}

type ExternalWithLocalSlice struct {
	Friends []Player
	Items   []external.GameItem
}

// namedtypes
// This package exercises field/type patterns that previously produced either
// compile errors or silent shallow copies. Each group is annotated with the
// bug it guards against.

// --- Tier 1: named non-struct types as POINTER fields (were compile errors:
//     generator emitted .deepcopy(v) on types with no method). ---

// Score is a named basic alias; *Score must shallow-copy (no method needed).
type Score int

// Matrix is a named array of basics; *Matrix must call the generated method.
type Matrix [3]int

// Inventory is a named map of basics; *Inventory must call the generated method.
type Inventory map[string]int

type NamedPtrFields struct {
	S *Score     // named basic alias pointer -> dc.CopyPtr (shallow, correct)
	M *Matrix    // named array pointer -> .deepcopy(v)
	I *Inventory // named map pointer -> .deepcopy(v)
}

// --- Tier 2: named composite VALUE fields (were silent shallow copies because
//     the generator emitted nothing for TypeStruct value fields). ---

// Tags is a named slice.
type Tags []string

// Meta is a named map.
type Meta map[string]int

// Coords is a named array of pointers.
type Coords [3]*int

type NamedValueFields struct {
	T Tags   // named slice value -> .deepcopy(v)
	M Meta   // named map value -> .deepcopy(v)
	C Coords // named array value -> .deepcopy(v)
}

// --- Tier 2: pointer to ANONYMOUS slice/map (were shallow via dc.CopyPtr). ---

type PtrToComposite struct {
	Basic *[]string                      // *[]basic
	Items *[]*external.GameItem          // []*external.X
	M     *map[string]int                // *map[K]basic
	MI    *map[string]*external.GameItem // *map[K]*external.X
}

// --- Tier 2: map of STRUCT value (were shallow: map values are not
//     addressable so dc.CopyMap's deepcopy assertion failed). ---

type Item struct {
	Tags []string
}

type MapOfStruct struct {
	Same  map[string]Item              // same-pkg struct value
	Cross map[string]external.GameItem // cross-pkg struct value
}

// --- Tier 2: anonymous struct field (was shallow: no method possible). ---

type AnonHolder struct {
	S struct {
		N     *int
		Inner struct {
			P *string
		}
	}
}

// --- Tier 2: same-pkg struct VALUE field (was shallow: default branch). ---

type Addr struct {
	Tags []string
}

type HasStructValue struct {
	Home Addr
}

// --- Cross-pkg struct VALUE field (was shallow). ---

type HasCrossValue struct {
	Owner external.AccountInfo
}

// --- Anonymous struct as MAP value: empty struct (set pattern) and
// non-empty anonymous struct (temp-var field-wise copy). ---

type Sets struct {
	Members map[int32]struct{}          // empty struct set
	Tagged  map[string]struct{ N *int } // non-empty anonymous struct value
}

// --- Slices of named basic alias and empty struct (were compile errors via
//     .deepcopy on method-less types). ---

type Aliases struct {
	Scores []Score    // named basic alias slice
	Empty  []struct{} // empty struct slice
}

type ConcurrentMapShared[K comparable, V any] struct {
	items        map[K]V
	sync.RWMutex // Read Write mutex, guards access to internal map.
}
type ConcurrentMap[K comparable, V any] struct {
	shards   []*ConcurrentMapShared[K, V]
	sharding func(key K) uint32
}
