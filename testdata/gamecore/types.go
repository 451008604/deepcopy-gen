package gamecore

type Position struct {
	X float64
	Y float64
	Z float64
}

type Character struct {
	Name   string
	Level  int
	Pos    Position
	HP     int
	MP     int
}

type Inventory struct {
	Slots    []*Item
	Capacity int
}

type Item struct {
	Name   string
	Type   string
	Damage int
}
