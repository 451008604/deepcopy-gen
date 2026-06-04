package nested

// Address is a value-type struct used as a nested field.
type Address struct {
	Street string
	City   string
	Zip    string
}

// Employee embeds Address and has pointer/slice fields.
type Employee struct {
	Name      string
	Age       int
	HomeAddr  Address
	WorkAddr  *Address
	Emails    []string
	Manager   *Employee
	Tags      map[string]string
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
