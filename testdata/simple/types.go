package simple

// Point is a simple struct with only value-type fields.
type Point struct {
	X int
	Y int
}

// Person has mixed basic-type fields.
type Person struct {
	Name    string
	Age     int
	Email   string
	Active  bool
	Score   float64
}

// Config uses various basic types.
type Config struct {
	Timeout  int64
	Verbose  bool
	MaxRetry int
	Label    string
	Rate     float32
}
