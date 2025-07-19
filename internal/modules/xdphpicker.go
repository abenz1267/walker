package modules

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

type XdphPicker struct {
	config  *config.XdphPicker
	entries []*util.Entry
}

func (x *XdphPicker) Cleanup() {
}

func (x *XdphPicker) Entries(term string) []*util.Entry {
	return x.entries
}

func (x *XdphPicker) General() *config.GeneralModule {
	return &x.config.GeneralModule
}

func (x *XdphPicker) Refresh() {
	x.config.IsSetup = !x.config.Refresh
}

func (x *XdphPicker) Setup() bool {
	x.config = &config.Cfg.Builtins.XdphPicker

	return true
}

type Window struct {
	ID    string
	Class string
	Title string
}

func (x *XdphPicker) SetupData() {
	x.entries = []*util.Entry{}

	windows := parseWindows(os.Getenv("XDPH_WINDOW_SHARING_LIST"))

	for _, w := range windows {
		x.entries = append(x.entries, &util.Entry{
			Label:            fmt.Sprintf("%s - %s", w.Title, w.Class),
			Sub:              "Window",
			Matching:         util.Fuzzy,
			SpecialFuncArgs:  []interface{}{fmt.Sprintf("[SELECTION]/window:%s", w.ID)},
			SpecialFunc:      x.SpecialFunc,
			RecalculateScore: true,
		})
	}

	monitors := gdk.DisplayManagerGet().DefaultDisplay().Monitors()

	for i := 0; i < int(monitors.NItems()); i++ {
		monitor := monitors.Item(uint(i)).Cast().(*gdk.Monitor)

		x.entries = append(x.entries, &util.Entry{
			Label:            monitor.Connector(),
			Sub:              "Screen",
			Matching:         util.Fuzzy,
			SpecialFuncArgs:  []interface{}{fmt.Sprintf("[SELECTION]/screen:%s", monitor.Connector())},
			SpecialFunc:      x.SpecialFunc,
			RecalculateScore: true,
		})
	}

	path, _ := exec.LookPath("slurp")

	if path != "" {
		x.entries = append(x.entries, &util.Entry{
			Label:            "Selection Region",
			Sub:              "Region",
			Matching:         util.Fuzzy,
			SpecialFuncArgs:  []interface{}{"region"},
			SpecialFunc:      x.SpecialFunc,
			RecalculateScore: true,
		})
	}
}

func (x *XdphPicker) SpecialFunc(args ...interface{}) {
	result := args[0].(string)

	if result == "region" {
		cmd := exec.Command("slurp", "-f", "%o@%x,%y,%w,%h")

		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("xdphpicker", "error", err, "output", string(out))
			return
		}

		fmt.Println(fmt.Sprintf("[SELECTION]/region:%s", string(out)))
	}

	fmt.Println(result)
}

func parseWindows(input string) []Window {
	// Split the string by [HE>] to separate each window entry
	entries := strings.Split(input, "[HE>]")
	var windows []Window

	for _, entry := range entries {
		if entry == "" {
			continue
		}

		// Split each entry by the markers
		parts := strings.Split(entry, "[HC>]")
		if len(parts) != 2 {
			continue
		}

		id := parts[0]
		remainingParts := strings.Split(parts[1], "[HT>]")
		if len(remainingParts) != 2 {
			continue
		}

		windows = append(windows, Window{
			ID:    id,
			Class: remainingParts[0],
			Title: remainingParts[1],
		})
	}

	return windows
}
