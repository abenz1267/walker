package state

import (
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/clipboard"
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
	IsMeasured          bool
	IsRunning           bool
	IsService           bool
	KeepSort            bool
	Password            bool
	Started             time.Time
}

func Get() *AppState {
	return &AppState{
		Started:        time.Now(),
		IsService:      false,
		IsRunning:      false,
		IsMeasured:     false,
		HasUI:          false,
		ExplicitConfig: "config.json",
		ExplicitStyle:  "style.css",
	}
}

func (app *AppState) StartServiceableModules(cfg *config.Config) {
	app.Clipboard = &clipboard.Clipboard{}
}
