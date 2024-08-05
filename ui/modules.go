package ui

import (
	"slices"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/emojis"
	"github.com/abenz1267/walker/modules/windows"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func setupModules() {
	all := getModules()
	toUse = []modules.Workable{}
	available = []modules.Workable{}

	for k, v := range all {
		if v == nil {
			continue
		}

		if !v.General().IsSetup {
			if ok := all[k].Setup(cfg); ok {
				if slices.Contains(cfg.Disabled, v.General().Name) {
					continue
				}

				available = append(available, all[k])
				cfg.Available = append(cfg.Available, v.General().Name)
			}
		} else {
			if slices.Contains(cfg.Disabled, v.General().Name) {
				continue
			}

			available = append(available, all[k])
			cfg.Available = append(cfg.Available, v.General().Name)
		}
	}

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

	toCheck := toUse

	if appstate.IsService {
		toCheck = all
	}

	for _, v := range toCheck {
		if v != nil && v.General().Theme != "" && v.General().Theme != cfg.Theme {
			layouts[v.General().Name] = config.GetLayout(v.General().Theme, v.General().ThemeBase)
		}
	}

	setupSingleModule()
}

func getModules() []modules.Workable {
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
		appstate.Clipboard,
	}

	if appstate.Dmenu != nil {
		res = append(res, appstate.Dmenu)
	} else {
		res = append(res, &modules.Dmenu{})
	}

	for _, v := range cfg.Plugins {
		e := &modules.Plugin{}
		e.PluginCfg = v

		res = append(res, e)
	}

	return res
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

	toSetup := []string{}

	for _, v := range appstate.ExplicitModules {
		if slices.Contains(cfg.Available, v) {
			for k, m := range available {
				if m.General().Name == v {
					explicits = append(explicits, available[k])
				}
			}
		} else {
			toSetup = append(toSetup, v)
		}
	}

	modules := getModules()

	for k, v := range modules {
		if v != nil {
			if slices.Contains(toSetup, v.General().Name) {
				if !v.General().IsSetup {
					if ok := v.Setup(cfg); ok {
						explicits = append(explicits, modules[k])
					}
				}
			}
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
