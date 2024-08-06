package state

import (
	"context"
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
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
	cfg.IsService = true

	app.Dmenu = &modules.Dmenu{}

	clipboard := &clipboard.Clipboard{}
	hasClipboard := clipboard.Setup(cfg)

	app.Dmenu.Setup(cfg)

	if !slices.Contains(cfg.Disabled, clipboard.General().Name) && hasClipboard {
		app.Clipboard = clipboard
		app.Clipboard.SetupData(cfg, context.Background())
	}

	if !slices.Contains(cfg.Disabled, app.Dmenu.General().Name) {
		app.Dmenu.SetupData(cfg, context.Background())
	}
}
