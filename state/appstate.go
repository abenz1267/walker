package state

import (
	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/clipboard"
	"github.com/junegunn/fzf/src/algo"
)

type AppState struct {
	Clipboard           modules.Workable
	IsDmenu             bool
	Dmenu               *modules.Dmenu
	DmenuSeparator      string
	DmenuLabelColumn    int
	ExplicitConfig      string
	ExplicitModules     []string
	ExplicitPlaceholder string
	ExplicitTheme       string
	ForcePrint          bool
	HasUI               bool
	IsRunning           bool
	IsService           bool
	KeepSort            bool
	Password            bool
	Benchmark           bool
	IsSingle            bool
	Labels              []string
	LabelsF             []string
	UsedLabels          []string
	InitialQuery        string
}

func Get() *AppState {
	algo.Init("default")

	return &AppState{
		IsService:      false,
		IsRunning:      false,
		HasUI:          false,
		ExplicitConfig: "config.json",
	}
}

func (app *AppState) StartServiceableModules(cfg *config.Config) {
	app.Clipboard = &clipboard.Clipboard{}
	app.Dmenu = &modules.Dmenu{}
	app.Dmenu.Setup(cfg)
}
