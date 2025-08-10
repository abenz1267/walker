package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	aiModule "github.com/abenz1267/walker/internal/modules/ai"
	"github.com/abenz1267/walker/internal/modules/emojis"
	"github.com/abenz1267/walker/internal/modules/symbols"
	"github.com/abenz1267/walker/internal/modules/translation"
	"github.com/abenz1267/walker/internal/util"
	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

func setupModules() {
	setAvailables()
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

		if text != "" {
			elements.input.SetObjectProperty("placeholder-text", text)
		}
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

		if len(c.HistoryBlacklist) > 0 {
			for n, b := range c.HistoryBlacklist {
				available[k].General().HistoryBlacklist[n].Reg = regexp.MustCompile(b.Regexp)
			}
		}
	}
}

func setupLayouts(modules []modules.Workable) {
	base, _ := util.ThemeDir()
	dirs := []string{base}
	dirs = append(dirs, config.Cfg.ThemeLocation...)

	list := make(map[string]struct{})

	for _, v := range dirs {
		filepath.Walk(v, func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
				if name != "" {
					list[name] = struct{}{}
				}
			}

			return nil
		})
	}

	for k := range list {
		themes[k], layoutErr = config.GetLayout(k, nil)
	}

	for _, v := range modules {
		g := v.General()
		if v != nil && g.Theme != "" && g.Theme != config.Cfg.Theme {
			mergedLayouts[g.Name], layoutErr = config.GetLayout(g.Theme, g.ThemeBase)
		}
	}
}

func setAvailables() {
	res := []modules.Workable{
		&modules.Applications{Hstry: hstry},
		&modules.Bookmarks{},
		&aiModule.AI{},
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
		// &windows.Windows{},
		&translation.Translation{},
	}

	if os.Getenv("XDG_CURRENT_DESKTOP") == "Hyprland" {
		res = append(res, &modules.XdphPicker{})
		res = append(res, &modules.HyprlandKeybinds{})
	}

	loadPluginsFromDisk()

	for _, v := range config.Cfg.Plugins {
		e := &modules.Plugin{}
		e.Config = v

		res = append(res, e)
	}

	available = []modules.Workable{}
	config.Cfg.Hidden = []string{}

	for _, v := range res {
		if v == nil {
			continue
		}

		if ok := v.Setup(); ok {
			if v.General().Name == "" {
				log.Panicln("module has no name")
			}

			if v.General().EagerLoading {
				go v.SetupData()
			}

			if slices.Contains(config.Cfg.Disabled, v.General().Name) {
				continue
			}

			available = append(available, v)
			config.Cfg.Available = append(config.Cfg.Available, config.SwitcherAvailable{Name: v.General().Name, Icon: v.General().Icon})

			if v.General().Hidden {
				config.Cfg.Hidden = append(config.Cfg.Hidden, v.General().Name)
			}
		}
	}

	if appstate.Dmenu != nil {
		available = append(available, appstate.Dmenu)
		config.Cfg.Available = append(config.Cfg.Available, config.SwitcherAvailable{Name: appstate.Dmenu.General().Name, Icon: appstate.Dmenu.General().Icon})

		if appstate.Dmenu.Config.Hidden {
			config.Cfg.Hidden = append(config.Cfg.Hidden, appstate.Dmenu.General().Name)
		}
	}

	if appstate.IsService {
		if appstate.Clipboard != nil {
			available = append(available, appstate.Clipboard)
			config.Cfg.Available = append(config.Cfg.Available, config.SwitcherAvailable{Name: appstate.Clipboard.General().Name, Icon: appstate.Clipboard.General().Icon})

			if appstate.Clipboard.General().Hidden {
				config.Cfg.Hidden = append(config.Cfg.Hidden, appstate.Clipboard.General().Name)
			}
		}
	}

	// windows := findModule("windows", available)

	// if windows != nil || config.Cfg.Builtins.Applications.ContextAware {
	// 	go wlr.StartWM()
	// }
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

func setExplicits() error {
	explicits = []modules.Workable{}

	for _, v := range appstate.ExplicitModules {
		if slices.ContainsFunc(config.Cfg.Available, func(a config.SwitcherAvailable) bool {
			return a.Name == v
		}) {
			for k, m := range available {
				if m.General().Name == v {
					explicits = append(explicits, available[k])
				}
			}
		}
	}

	if len(explicits) == 0 {
		log.Println("Module(s) not found.")

		if !appstate.IsService {
			os.Exit(1)
		}

		return errors.New("expected module(s) not found")
	}

	return nil
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

	debouncedProcess = util.NewDebounce(time.Millisecond * time.Duration(singleModule.General().Delay))
}

func resetSingleModule() {
	t := 1

	if config.Cfg.Search.Delay > 0 {
		t = config.Cfg.Search.Delay
	}

	debouncedProcess = util.NewDebounce(time.Millisecond * time.Duration(t))
	singleModule = nil
}

func loadPluginsFromDisk() {
	dir, _ := util.ConfigDir()

	path := filepath.Join(dir, "plugins")
	if !util.FileExists(path) {
		return
	}

	locations := []string{path}
	locations = append(locations, config.Cfg.PluginLocation...)

	for _, v := range locations {
		filepath.Walk(v, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			executeWith := ""

			switch filepath.Ext(path) {
			case ".cjs", ".js":
				executeWith = appstate.JSRuntime
			case ".lua":
				executeWith = "lua"
			}

			// check if file is executable
			if executeWith == "" && info.Mode()&0111 == 0 {
				return nil
			}

			cmd := exec.Command(path, "info")
			if executeWith != "" {
				cmd = exec.Command(executeWith, path, "info")
			}

			out, err := cmd.CombinedOutput()
			if err != nil {
				slog.Error("plugins", "getinfo", err, "plugin", path, "out", string(out))
				return nil
			}

			defaults := koanf.New(".")

			err = defaults.Load(rawbytes.Provider(out), toml.Parser())
			if err != nil {
				slog.Error("plugins", "parse", err, "plugin", path)
				return nil
			}

			plugin := config.Plugin{}

			err = defaults.Unmarshal("", &plugin)
			if err != nil {
				slog.Error("plugins", "unmarshal", err, "plugin", path)
				return nil
			}

			plugin.Src = fmt.Sprintf("%s entries", path)

			if executeWith != "" {
				plugin.Src = fmt.Sprintf("%s %s entries", executeWith, path)
			}

			if plugin.SrcOnce == "yes" {
				plugin.SrcOnce = plugin.Src
			}

			config.Cfg.Plugins = append(config.Cfg.Plugins, plugin)

			return nil
		})
	}
}
