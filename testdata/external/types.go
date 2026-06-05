package external

type AccountInfo struct {
	ID       int64
	Username string
	Email    string
	Level    int
	Score    float64
}

type AccountExtra struct {
	Avatar   string
	Bio      string
	Tags     []string
	Metadata map[string]string
}

type GameItem struct {
	Name     string
	Count    int
	Owner    *AccountInfo
}

type GameState struct {
	Items      []GameItem
	Inventory  map[string]*GameItem
	LastActive *AccountInfo
}
