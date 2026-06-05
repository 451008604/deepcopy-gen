package crosspkg

import "github.com/451008604/deepcopy-gen/testdata/external"

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
