package complex

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
