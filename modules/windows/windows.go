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

func (w Windows) Cleanup() {
}

func (w Windows) History() bool {
	return w.general.History
}

func (w Windows) Typeahead() bool {
	return w.general.Typeahead
}

func (Windows) KeepSort() bool {
	return false
}

func (w Windows) IsSetup() bool {
	return w.general.IsSetup
}

func (w Windows) Placeholder() string {
	if w.general.Placeholder == "" {
		return "Windows"
	}

	return w.general.Placeholder
}

func (w Windows) SwitcherOnly() bool {
	return w.general.SwitcherOnly
}

func (w *Windows) Setup(cfg *config.Config) bool {
	return true
}

func (w *Windows) SetupData(cfg *config.Config) {
	go wlr.StartWM()

	w.general.IsSetup = true
}

func (Windows) Name() string {
	return "windows"
}

func (w Windows) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	res := wlr.GetWindows()

	for _, v := range res {
		entries = append(entries, util.Entry{
			Label:          v.Title,
			Sub:            fmt.Sprintf("Windows: %s", v.AppId),
			UseSpecialFunc: true,
			Args:           []interface{}{v.Toplevel.Id()},
			Searchable:     v.AppId,
			Categories:     []string{"windows"},
			Class:          "windows",
			Matching:       util.Fuzzy,
		})
	}

	return entries
}

func (w Windows) Prefix() string {
	return w.general.Prefix
}

func (w *Windows) Refresh() {
	// w.general.IsSetup = false
}

func (w Windows) SpecialFunc(args ...interface{}) {
	if len(args) == 0 {
		return
	}

	index := args[0].(wl.ProxyId)

	wlr.Activate(index)
}
