package state

import (
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/clipboard"
)

type AppState struct {
	Started    time.Time
	IsMeasured bool
	IsService  bool
	IsRunning  bool
	HasUI      bool
	Clipboard  modules.Workable
}

func Get() *AppState {
	return &AppState{
		Started:    time.Now(),
		IsService:  false,
		IsRunning:  false,
		IsMeasured: false,
		HasUI:      false,
	}
}

func (app *AppState) StartServiceableModules(cfg *config.Config) {
	app.Clipboard = clipboard.Clipboard{}.Setup(cfg)
}
