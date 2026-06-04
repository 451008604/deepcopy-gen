package embedded

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
