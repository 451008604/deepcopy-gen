package edgecase

type WithInterface struct {
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
