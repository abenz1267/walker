package ui

import (
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/emojis"
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

	for _, v := range checkForLayout {
		if v != nil && v.General().Theme != "" && v.General().Theme != cfg.Theme {
			layouts[v.General().Name] = config.GetLayout(v.General().Theme, v.General().ThemeBase)
		}
	}

	setupSingleModule()
}

func setAvailables(cfg *config.Config) {
	res := []modules.Workable{
		&modules.Applications{},
		&modules.Runner{},
		&modules.Websearch{},
		&modules.Commands{},
		&modules.SSH{},
		&modules.Finder{},
		&modules.Switcher{},
		&emojis.Emojis{},
		&modules.CustomCommands{},
		&windows.Windows{},
	}

	if !appstate.IsService {
		res = append(res, &modules.Dmenu{})
	}

	for _, v := range cfg.Plugins {
		e := &modules.Plugin{}
		e.PluginCfg = v

		res = append(res, e)
	}

	available = []modules.Workable{}

	for _, v := range res {
		if v == nil {
			continue
		}

		if slices.Contains(cfg.Disabled, v.General().Name) {
			continue
		}

		if ok := v.Setup(cfg); ok {
			if v.General().Name == "" {
				log.Panicln("module has no name\n")
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

	glib.IdleAdd(func() {
		elements.input.SetObjectProperty("search-delay", singleModule.General().Delay)
	})
}

func resetSingleModule() {
	elements.input.SetObjectProperty("search-delay", cfg.Search.Delay)
	singleModule = nil
}
