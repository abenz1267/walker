package state

import (
	"os"
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/junegunn/fzf/src/algo"
)

type AppState struct {
	ActiveItem          *int
	Hidebar             bool
	AutoSelect          bool
	Clipboard           modules.Workable
	ConfigError         error
	IsDebug             bool
	IsDmenu             bool
	Dmenu               *modules.Dmenu
	DmenuSeparator      string
	DmenuLabelColumn    int
	DmenuIconColumn     int
	DmenuValueColumn    int
	ExplicitConfig      string
	ExplicitModules     []string
	DmenuShowChan       chan bool
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
	LastQuery           string
}

func Get() *AppState {
	algo.Init("default")

	_, isDebug := os.LookupEnv("DEBUG")

	return &AppState{
		IsService:      false,
		IsDebug:        isDebug,
		IsRunning:      false,
		HasUI:          false,
		ExplicitConfig: "config.json",
		DmenuShowChan:  make(chan bool, 1),
	}
}

func (app *AppState) StartServiceableModules() {
	config.Cfg.IsService = true

	app.Dmenu = &modules.Dmenu{
		DmenuShowChan: app.DmenuShowChan,
	}

	clipboard := &clipboard.Clipboard{}

	app.Dmenu.Setup()

	if !slices.Contains(config.Cfg.Disabled, clipboard.General().Name) && clipboard.Setup() {
		app.Clipboard = clipboard
		app.Clipboard.SetupData()
	}

	if !slices.Contains(config.Cfg.Disabled, app.Dmenu.General().Name) {
		app.Dmenu.SetupData()
	}
}
