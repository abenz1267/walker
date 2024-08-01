package windows

import (
	"context"
	"fmt"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules/windows/wlr"
	"github.com/abenz1267/walker/util"
	"github.com/neurlang/wayland/wl"
)

type Windows struct {
	general   config.GeneralModule
	entries   []util.Entry
	functions []func()
}

func (w Windows) General() *config.GeneralModule {
	return &w.general
}

func (w Windows) Cleanup() {
}

func (w *Windows) Setup(cfg *config.Config) bool {
	w.general = cfg.Builtins.Windows.GeneralModule

	return true
}

func (w *Windows) SetupData(cfg *config.Config, ctx context.Context) {
	if !wlr.IsRunning {
		go wlr.StartWM(nil, nil)
	}

	w.general.IsSetup = true
}

func (w Windows) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	res := wlr.GetWindows()

	for _, v := range res {
		entries = append(entries, util.Entry{
			Label:           v.Title,
			Sub:             fmt.Sprintf("Windows: %s", v.AppId),
			Searchable:      v.AppId,
			Categories:      []string{"windows"},
			Class:           "windows",
			Matching:        util.Fuzzy,
			SpecialFunc:     w.SpecialFunc,
			SpecialFuncArgs: []interface{}{v.Toplevel.Id()},
		})
	}

	return entries
}

func (w *Windows) Refresh() {
}

func (w Windows) SpecialFunc(args ...interface{}) {
	if len(args) == 0 {
		return
	}

	id := args[0].(wl.ProxyId)

	wlr.Activate(id)
}
