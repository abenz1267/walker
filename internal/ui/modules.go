package ui

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/emojis"
	"github.com/abenz1267/walker/internal/modules/symbols"
	"github.com/abenz1267/walker/internal/modules/windows"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func setupModules() {
	setAvailables(cfg)
	toUse = []modules.Workable{}

	if len(appstate.ExplicitModules) > 0 {
		setExplicits()
	}

	clear(elements.prefixClasses)

	for _, v := range available {
		elements.prefixClasses[v.General().Prefix] = append(elements.prefixClasses[v.General().Prefix], v.General().Name)
	}

	if len(explicits) > 0 {
		toUse = explicits
	} else {
		toUse = available
	}

	if len(toUse) == 1 {
		text := toUse[0].General().Placeholder
		if appstate.ExplicitPlaceholder != "" {
			text = appstate.ExplicitPlaceholder
		}

		elements.input.SetObjectProperty("placeholder-text", text)
	}

	checkForLayout := toUse

	if appstate.IsService {
		checkForLayout = available
	}

	if len(toUse) == 1 {
		setupLayouts(checkForLayout)
	} else {
		go setupLayouts(checkForLayout)
	}

	prepareBlacklists()
	setupSingleModule()
}

func prepareBlacklists() {
	for k, v := range available {
		c := v.General()

		if len(c.Blacklist) > 0 {
			for n, b := range c.Blacklist {
				available[k].General().Blacklist[n].Reg = regexp.MustCompile(b.Regexp)
			}
		}
	}
}

func setupLayouts(modules []modules.Workable) {
	for _, v := range modules {
		g := v.General()
		if v != nil && g.Theme != "" && g.Theme != cfg.Theme {
			layouts[g.Name] = config.GetLayout(g.Theme, g.ThemeBase)
		}
	}
}

func setAvailables(cfg *config.Config) {
	res := []modules.Workable{
		&modules.Applications{Hstry: hstry},
		&modules.Bookmarks{},
		&modules.AI{},
		&modules.Runner{},
		&modules.Websearch{},
		&modules.Calc{},
		&modules.Commands{},
		&modules.SSH{},
		&modules.Finder{MarkerColor: layout.Window.Box.Scroll.List.MarkerColor},
		&modules.Switcher{},
		&emojis.Emojis{},
		&symbols.Symbols{},
		&modules.CustomCommands{},
		&windows.Windows{},
	}

	if os.Getenv("XDG_CURRENT_DESKTOP") == "Hyprland" {
		res = append(res, &modules.XdphPicker{})
	}

	if !appstate.IsService {
		res = append(res, &modules.Dmenu{})
	}

	for _, v := range cfg.Plugins {
		e := &modules.Plugin{}
		e.Config = v

		res = append(res, e)
	}

	available = []modules.Workable{}

	for _, v := range res {
		if v == nil {
			continue
		}

		if ok := v.Setup(cfg); ok {
			if v.General().Name == "" {
				log.Panicln("module has no name\n")
			}

			if slices.Contains(cfg.Disabled, v.General().Name) {
				continue
			}

			available = append(available, v)
			cfg.Available = append(cfg.Available, v.General().Name)
		}
	}

	if appstate.IsService {
		if appstate.Dmenu != nil {
			available = append(available, appstate.Dmenu)
			cfg.Available = append(cfg.Available, appstate.Dmenu.General().Name)
		}

		if appstate.Clipboard != nil {
			available = append(available, appstate.Clipboard)
			cfg.Available = append(cfg.Available, appstate.Clipboard.General().Name)
		}
	}
}

func findModule(name string, modules ...[]modules.Workable) modules.Workable {
	for _, v := range modules {
		for _, w := range v {
			if w != nil && w.General().Name == name {
				return w
			}
		}
	}

	return nil
}

func setExplicits() {
	explicits = []modules.Workable{}

	for _, v := range appstate.ExplicitModules {
		if slices.Contains(cfg.Available, v) {
			for k, m := range available {
				if m.General().Name == v {
					explicits = append(explicits, available[k])
				}
			}
		}
	}

	if len(explicits) == 0 {
		fmt.Printf("Module(s) not found\n.")

		if !appstate.IsService {
			os.Exit(1)
		}
	}
}

func setupSingleModule() {
	if len(explicits) != 1 && len(toUse) != 1 {
		return
	}

	if len(explicits) == 1 {
		singleModule = explicits[0]
	} else {
		singleModule = toUse[0]
	}

	appstate.AutoSelectOld = appstate.AutoSelect

	if !appstate.AutoSelect {
		appstate.AutoSelect = singleModule.General().AutoSelect
	}

	glib.IdleAdd(func() {
		elements.input.SetObjectProperty("search-delay", singleModule.General().Delay)
	})
}

func resetSingleModule() {
	elements.input.SetObjectProperty("search-delay", cfg.Search.Delay)
	singleModule = nil
}
