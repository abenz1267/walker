package ui

import (
	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/history"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/state"
)

var (
	cfg      *config.Config
	ui       *UI
	entries  map[string]modules.Entry
	procs    map[string][]modules.Workable
	hstry    history.History
	appstate *state.AppState
)
