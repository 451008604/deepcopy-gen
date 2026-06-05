package crosspkg

import (
	"github.com/451008604/deepcopy-gen/testdata/external"
	"github.com/451008604/deepcopy-gen/testdata/gamecore"
)

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

type NestedExternal struct {
	external.GameState
	MainCharacter gamecore.Character
}

type ExternalWithLocalSlice struct {
	Friends []Player
	Items   []external.GameItem
}
