package state

import (
	"os"
	"os/exec"
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/junegunn/fzf/src/algo"
)

type OldSizeData struct {
	BoxWidth      int
	BoxHeight     int
	ListWidth     int
	ListHeight    int
	ListMinWidth  int
	ListMinHeight int
	ListMaxWidth  int
	ListMaxHeight int
}

type AppState struct {
	ModulesStarted      bool
	ActiveItem          *int
	Hidebar             bool
	AutoSelect          bool
	Clipboard           modules.Workable
	ConfigError         error
	IsDebug             bool
	IsDmenu             bool
	Dmenu               *modules.Dmenu
	DmenuResultChan     chan string
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
	IsSingle            bool
	Labels              []string
	LabelsF             []string
	UsedLabels          []string
	InitialQuery        string
	LastQuery           string
	JSRuntime           string
	WidthOverwrite      int
	HeightOverwrite     int
	OldSizeData         *OldSizeData
}

func Get() *AppState {
	algo.Init("default")

	_, isDebug := os.LookupEnv("DEBUG")

	return &AppState{
		IsService:       false,
		IsDebug:         isDebug,
		IsRunning:       false,
		HasUI:           false,
		ExplicitConfig:  "config.json",
		JSRuntime:       getJsRuntime(),
		DmenuResultChan: make(chan string),
		OldSizeData:     nil,
	}
}

func getJsRuntime() string {
	possible := []string{"node", "bun", "deno"}

	for _, v := range possible {
		pth, _ := exec.LookPath(v)
		if pth != "" {
			return v
		}
	}

	return ""
}

func (app *AppState) StartServiceableModules() {
	config.Cfg.IsService = true

	app.Dmenu = &modules.Dmenu{
		DmenuShowChan: app.DmenuShowChan,
	}

	app.Dmenu.Setup()

	clipboard := &clipboard.Clipboard{}

	if !slices.Contains(config.Cfg.Disabled, clipboard.General().Name) && clipboard.Setup() {
		app.Clipboard = clipboard
		app.Clipboard.SetupData()
	}

	app.ModulesStarted = true
}
