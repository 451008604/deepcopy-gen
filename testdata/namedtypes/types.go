package namedtypes

import (
	"github.com/451008604/deepcopy-gen/testdata/external"
)

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
