package state

import (
	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/clipboard"
	"github.com/junegunn/fzf/src/algo"
)

type AppState struct {
	Clipboard           modules.Workable
	Dmenu               modules.Workable
	ExplicitConfig      string
	ExplicitModules     []string
	ExplicitPlaceholder string
	ExplicitStyle       string
	ForcePrint          bool
	HasUI               bool
	IsRunning           bool
	IsService           bool
	KeepSort            bool
	Password            bool
	Benchmark           bool
}

func Get() *AppState {
	algo.Init("default")

	return &AppState{
		IsService:      false,
		IsRunning:      false,
		HasUI:          false,
		ExplicitConfig: "config.json",
		ExplicitStyle:  "style.css",
	}
}

func (app *AppState) StartServiceableModules(cfg *config.Config) {
	app.Clipboard = &clipboard.Clipboard{}
}
