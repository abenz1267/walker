package ui

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/history"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/modules/emojis"
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var (
	keys              map[uint]uint
	activationEnabled bool
	amKey             uint
	cmdAltModifier    gdk.ModifierType
	amModifier        gdk.ModifierType
	commands          map[string]func()
)

func setupCommands() {
	commands = make(map[string]func())
	commands["reloadconfig"] = func() {
		cfg = config.Get(appstate.ExplicitConfig)
		setupUserStyle()
		setupModules()
	}
	commands["resethistory"] = func() {
		os.Remove(filepath.Join(util.CacheDir(), "history.gob"))
		hstry = history.Get()
	}
	commands["clearapplicationscache"] = func() {
		os.Remove(filepath.Join(util.CacheDir(), "applications.json"))
	}
	commands["clearclipboard"] = func() {
		os.Remove(filepath.Join(util.CacheDir(), "clipboard.gob"))
	}
}

func getModules() []modules.Workable {
	res := []modules.Workable{
		&modules.Applications{},
		&modules.Runner{},
		&modules.Websearch{},
		&modules.Commands{},
		&modules.Hyprland{},
		&modules.SSH{},
		&modules.Finder{},
		&modules.Switcher{},
		&emojis.Emojis{},
		&modules.CustomCommands{},
		appstate.Clipboard,
	}

	if appstate.Dmenu != nil {
		res = append(res, appstate.Dmenu)
	} else {
		res = append(res, &modules.Dmenu{})
	}

	for _, v := range cfg.Plugins {
		e := &modules.Plugin{}
		e.General = v

		res = append(res, e)
	}

	return res
}

func findModule(name string, modules ...[]modules.Workable) modules.Workable {
	for _, v := range modules {
		for _, w := range v {
			if w != nil && w.Name() == name {
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
				if m.Name() == v {
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
			if slices.Contains(toSetup, v.Name()) {
				if !v.IsSetup() {
					if ok := v.Setup(cfg); ok {
						explicits = append(explicits, modules[k])
					}
				}
			}
		}
	}
}

func setupModules() {
	all := getModules()
	toUse = []modules.Workable{}
	available = []modules.Workable{}

	for k, v := range all {
		if v == nil || slices.Contains(cfg.Disabled, v.Name()) {
			continue
		}

		if !v.IsSetup() {
			if ok := all[k].Setup(cfg); ok {
				available = append(available, all[k])
				cfg.Available = append(cfg.Available, v.Name())
			}
		} else {
			available = append(available, all[k])
			cfg.Available = append(cfg.Available, v.Name())
		}
	}

	if len(appstate.ExplicitModules) > 0 {
		setExplicits()
	}

	clear(ui.prefixClasses)

	for _, v := range available {
		ui.prefixClasses[v.Prefix()] = append(ui.prefixClasses[v.Prefix()], v.Name())
	}

	if len(explicits) > 0 {
		toUse = explicits
	} else {
		toUse = available
	}

	if len(toUse) == 1 {
		text := toUse[0].Placeholder()
		if appstate.ExplicitPlaceholder != "" {
			text = appstate.ExplicitPlaceholder
		}

		ui.search.SetObjectProperty("placeholder-text", text)
	}
}

func setupInteractions(appstate *state.AppState) {
	setupCommands()
	createActivationKeys()

	setupModules()

	keycontroller := gtk.NewEventControllerKey()
	keycontroller.ConnectKeyPressed(handleSearchKeysPressed)

	ui.search.AddController(keycontroller)
	ui.search.Connect("search-changed", process)
	ui.search.Connect("activate", func() {
		if appstate.ForcePrint && ui.list.Model().NItems() == 0 {
			if appstate.IsDmenu {
				handleDmenuResult(ui.search.Text())
			}

			closeAfterActivation(false, false)
			return
		}

		activateItem(false, false, false)
	})

	listkc := gtk.NewEventControllerKey()
	listkc.ConnectKeyPressed(handleListKeysPressed)

	ui.list.AddController(listkc)

	amKey = gdk.KEY_Control_L
	amModifier = gdk.ControlMask

	cmdAltModifier = gdk.AltMask

	if cfg.ActivationMode.UseAlt {
		amKey = gdk.KEY_Alt_L
		amModifier = gdk.AltMask
		cmdAltModifier = gdk.ControlMask
	}
}

func selectNext() {
	items := ui.selection.NItems()

	if items == 0 {
		return
	}

	current := ui.selection.Selected()
	next := current + 1

	if next < items {
		ui.selection.SetSelected(current + 1)
		ui.list.ScrollTo(ui.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}

	if next >= items && cfg.List.Cycle {
		ui.selection.SetSelected(0)
		ui.list.ScrollTo(ui.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}
}

func selectPrev() {
	items := ui.selection.NItems()

	if items == 0 {
		return
	}

	current := ui.selection.Selected()

	if current > 0 {
		ui.selection.SetSelected(current - 1)
		ui.list.ScrollTo(ui.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}

	if current == 0 && cfg.List.Cycle {
		ui.selection.SetSelected(items - 1)
		ui.list.ScrollTo(ui.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}
}

func selectActivationMode(val uint, keepOpen bool) {
	var target uint

	if k, ok := specialLabels[val]; ok {
		target = k
	} else {
		if n, ok := keys[val]; ok {
			target = n
		}
	}

	if target < ui.selection.NItems() {
		ui.selection.SetSelected(target)
	}

	if keepOpen {
		activateItem(true, false, false)
		return
	}

	activateItem(false, false, false)
}

func enableAM() {
	c := ui.appwin.CSSClasses()
	c = append(c, "activation")

	ui.appwin.SetCSSClasses(c)
	ui.list.GrabFocus()

	activationEnabled = true
}

func disabledAM() {
	if !cfg.ActivationMode.Disabled && activationEnabled {
		activationEnabled = false
		ui.search.SetFocusable(false)

		c := ui.appwin.CSSClasses()

		for k, v := range c {
			if v == "activation" {
				c = slices.Delete(c, k, k+1)
			}
		}

		ui.appwin.SetCSSClasses(c)
		ui.search.GrabFocus()
	}
}

func handleListKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	if !cfg.ActivationMode.Disabled && ui.selection.NItems() != 0 {
		if val == amKey {
			enableAM()
			return true
		}
	}

	switch val {
	case gdk.KEY_Shift_L:
		return true
	case gdk.KEY_Return:
		if modifier == gdk.ShiftMask {
			activateItem(true, true, false)
		} else {
			return false
		}
	case gdk.KEY_Escape:
		if activationEnabled {
			disabledAM()
		} else {
			if appstate.IsDmenu {
				handleDmenuResult("")
			}

			if cfg.IsService {
				quit()
			} else {
				exit()
			}
		}
	case gdk.KEY_Tab:
		if ui.typeahead.Text() != "" {
			ui.search.SetText(ui.typeahead.Text())
			ui.search.SetPosition(-1)
		} else {
			selectNext()
		}
	case gdk.KEY_ISO_Left_Tab:
		selectPrev()
	case gdk.KEY_J, gdk.KEY_K, gdk.KEY_L, gdk.KEY_colon, gdk.KEY_A, gdk.KEY_S, gdk.KEY_D, gdk.KEY_F:
		fallthrough
	case gdk.KEY_j, gdk.KEY_k, gdk.KEY_l, gdk.KEY_semicolon, gdk.KEY_a, gdk.KEY_s, gdk.KEY_d, gdk.KEY_f:
		if !cfg.ActivationMode.Disabled && activationEnabled {
			if modifier == gdk.ShiftMask {
				selectActivationMode(val, true)
			} else {
				selectActivationMode(val, false)
			}
		} else {
			ui.search.GrabFocus()
		}
	default:
		uni := strings.ToLower(string(gdk.KeyvalToUnicode(val)))
		check := gdk.UnicodeToKeyval(uint32(uni[0]))

		if _, ok := specialLabels[check]; ok {
			if modifier == gdk.ShiftMask {
				selectActivationMode(check, true)
			} else {
				selectActivationMode(check, false)
			}
		} else {
			if !activationEnabled {
				ui.search.GrabFocus()
				return false
			}
		}
	}

	return true
}

var historyIndex = 0

func handleSearchKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	if !cfg.ActivationMode.Disabled && ui.selection.NItems() != 0 && !cfg.ActivationMode.UseFKeys {
		if val == amKey {
			enableAM()
			return true
		}
	}

	switch val {
	case gdk.KEY_Return:
		isShift := modifier == gdk.ShiftMask
		isAlt := modifier == cmdAltModifier

		isAltShift := modifier == (gdk.ShiftMask | cmdAltModifier)

		if isAltShift {
			isShift = true
			isAlt = true
		}

		if appstate.ForcePrint && ui.list.Model().NItems() == 0 {
			if appstate.IsDmenu {
				handleDmenuResult(ui.search.Text())
			}

			closeAfterActivation(isShift, false)
			break
		}

		activateItem(isShift, isShift, isAlt)
	case gdk.KEY_Escape:
		if appstate.IsDmenu {
			handleDmenuResult("")
		}

		if cfg.IsService {
			quit()
		} else {
			exit()
		}
	case gdk.KEY_Tab:
		if ui.typeahead.Text() != "" {
			ui.search.SetText(ui.typeahead.Text())
			ui.search.SetPosition(-1)
		} else {
			selectNext()
		}
	case gdk.KEY_Down:
		selectNext()
	case gdk.KEY_Up:
		if ui.selection.Selected() == 0 || ui.items.NItems() == 0 {
			if len(toUse) != 1 {
				selectPrev()
				break
			}

			if len(explicits) != 0 && len(explicits) != 1 {
				selectPrev()
				break
			}

			var inputhstry []string

			if len(explicits) == 1 {
				inputhstry = history.GetInputHistory(explicits[0].Name())
			} else {
				inputhstry = history.GetInputHistory(toUse[0].Name())
			}

			if len(inputhstry) > 0 {
				historyIndex++

				if historyIndex == len(inputhstry) {
					historyIndex = 0
				}

				glib.IdleAdd(func() {
					ui.search.SetText(inputhstry[historyIndex])
					ui.search.SetPosition(-1)
				})
			}
		} else {
			selectPrev()
		}
	case gdk.KEY_ISO_Left_Tab:
		selectPrev()
	case gdk.KEY_j:
		if cfg.ActivationMode.Disabled {
			selectNext()
		}
	case gdk.KEY_k:
		if cfg.ActivationMode.Disabled {
			selectPrev()
		}
	case gdk.KEY_F1, gdk.KEY_F2, gdk.KEY_F3, gdk.KEY_F4, gdk.KEY_F5, gdk.KEY_F6, gdk.KEY_F7, gdk.KEY_F8:
		if modifier == gdk.ShiftMask {
			selectActivationMode(val, true)
		} else {
			selectActivationMode(val, false)
		}
	default:
		if modifier == amModifier {
			return true
		}

		return false
	}

	return true
}

func activateItem(keepOpen, selectNext, alt bool) {
	if ui.list.Model().NItems() == 0 {
		return
	}

	entry := gioutil.ObjectValue[util.Entry](ui.items.Item(ui.selection.Selected()))

	if !keepOpen && entry.Sub != "switcher" && cfg.IsService {
		go quit()
	}

	toRun := entry.Exec

	forceTerminal := false

	if alt {
		if entry.ExecAlt != "" {
			toRun = entry.ExecAlt
		} else {
			forceTerminal = true
		}
	}

	if appstate.IsDmenu {
		handleDmenuResult(toRun)
		closeAfterActivation(keepOpen, selectNext)
		return
	}

	if entry.Sub == "Walker" {
		commands[entry.Exec]()
		closeAfterActivation(keepOpen, selectNext)
		return
	}

	if entry.Sub == "switcher" {
		for _, m := range toUse {
			if m.Name() == entry.Label {
				explicits = []modules.Workable{}
				explicits = append(explicits, m)

				ui.items.Splice(0, int(ui.items.NItems()))
				ui.search.SetObjectProperty("placeholder-text", m.Placeholder())
				ui.search.SetText("")
				ui.search.GrabFocus()
				return
			}
		}
	}

	if cfg.Terminal != "" {
		if entry.Terminal || forceTerminal {
			toRun = fmt.Sprintf("%s -e %s", cfg.Terminal, toRun)
		}
	} else {
		log.Println("terminal is not set")
		return
	}

	cmd := exec.Command("sh", "-c", toRun)

	if entry.Path != "" {
		cmd.Dir = entry.Path
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:    true,
		Pgid:       0,
		Foreground: false,
	}

	setStdin(cmd, &entry.Piped)

	if alt {
		setStdin(cmd, &entry.PipedAlt)
	}

	if entry.History {
		hstry.Save(entry.Identifier(), strings.TrimSpace(ui.search.Text()))
	}

	module := findModule(entry.Module, toUse, explicits)

	if module != nil && (module.History() || module.Typeahead()) {
		history.SaveInputHistory(module.Name(), ui.search.Text())
	}

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}

	closeAfterActivation(keepOpen, selectNext)
}

func handleDmenuResult(result string) {
	if appstate.IsService {
		for _, v := range toUse {
			if v.Name() == "dmenu" {
				v.(*modules.Dmenu).Reply(result)
			}
		}
	} else {
		fmt.Print(result)
	}
}

func setStdin(cmd *exec.Cmd, piped *util.Piped) {
	if piped.Content != "" {
		switch piped.Type {
		case "string":
			cmd.Stdin = strings.NewReader(piped.Content)
		case "file":
			b, err := os.ReadFile(piped.Content)
			if err != nil {
				log.Panic(err)
			}

			r := bytes.NewReader(b)
			cmd.Stdin = r
		}
	}
}

func closeAfterActivation(keepOpen, next bool) {
	if !cfg.IsService {
		exit()
	}

	if !keepOpen && appstate.IsRunning {
		quit()
		return
	}

	if appstate.IsRunning {
		if !activationEnabled && !next {
			ui.search.SetText("")
		}

		if next {
			selectNext()
		}
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var cancel context.CancelFunc

func process() {
	if cancel != nil {
		cancel()
	}

	ui.typeahead.SetText("")

	if cfg.IgnoreMouse {
		ui.list.SetCanTarget(false)
	}

	ui.spinner.SetCSSClasses([]string{"visible"})

	if !appstate.IsRunning {
		return
	}

	text := strings.TrimSpace(ui.search.Text())

	if text == "" && cfg.List.ShowInitialEntries && len(explicits) == 0 && !appstate.IsDmenu {
		setInitials()
		return
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	if (ui.search.Text() != "" || appstate.IsDmenu) || (len(explicits) > 0 && cfg.List.ShowInitialEntries) {
		go processAsync(ctx, text)
	} else {
		ui.items.Splice(0, int(ui.items.NItems()))
		ui.spinner.SetCSSClasses([]string{})
	}
}

var handlerPool = sync.Pool{
	New: func() any {
		return new(Handler)
	},
}

func processAsync(ctx context.Context, text string) {
	handler := handlerPool.Get().(*Handler)
	defer func() {
		handlerPool.Put(handler)
		ui.spinner.SetCSSClasses([]string{})
		cancel()
	}()

	hasExplicit := len(explicits) > 0

	handler.ctx = ctx
	handler.entries = []util.Entry{}

	glib.IdleAdd(func() {
		ui.items.Splice(0, int(ui.items.NItems()))
	})

	p := toUse

	hasPrefix := false

	prefixes := []string{}

	for _, v := range p {
		prefix := v.Prefix()

		if len(prefix) == 1 {
			if strings.HasPrefix(text, prefix) {
				prefixes = append(prefixes, prefix)
				hasPrefix = true
			}
		}

		if len(prefix) > 1 {
			if strings.HasPrefix(text, fmt.Sprintf("%s ", prefix)) {
				prefixes = append(prefixes, prefix)
				hasPrefix = true
			}
		}
	}

	if !hasExplicit {
		if hasPrefix {
			glib.IdleAdd(func() {
				for _, v := range prefixes {
					ui.appwin.SetCSSClasses(ui.prefixClasses[v])
				}
			})
		}
	} else {
		p = explicits
	}

	setTypeahead(p)

	handler.receiver = make(chan []util.Entry)
	go handler.handle()

	var hyprland *modules.Hyprland

	if cfg.Builtins.Hyprland.ContextAwareHistory && cfg.IsService {
		for _, v := range p {
			if v.Name() == "hyprland" {
				hyprland = v.(*modules.Hyprland)
				break
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(p))

	if len(p) == 1 {
		handler.keepSort = p[0].KeepSort()
		appstate.IsSingle = true
	}

	for k := range p {
		if p[k] == nil {
			continue
		}

		if !hasExplicit {
			if p[k].SwitcherOnly() {
				continue
			}

			prefix := p[k].Prefix()

			if hasPrefix && prefix == "" {
				continue
			}

			if !hasPrefix && prefix != "" {
				continue
			}

			if len(prefix) > 1 {
				prefix = fmt.Sprintf("%s ", prefix)
			}

			if hasPrefix && !strings.HasPrefix(text, prefix) {
				continue
			}

			text = strings.TrimPrefix(text, prefix)
		}

		if !p[k].IsSetup() {
			p[k].SetupData(cfg)
		}

		go func(ctx context.Context, wg *sync.WaitGroup, text string, w modules.Workable) {
			defer wg.Done()

			e := w.Entries(ctx, text)

			toPush := []util.Entry{}

			for k := range e {
				e[k].Module = w.Name()

				if e[k].DragDrop && !ui.list.CanTarget() {
					ui.list.SetCanTarget(true)
				}

				toMatch := text

				if e[k].MatchFields > 0 {
					textFields := strings.Fields(text)

					if len(textFields) > 0 {
						toMatch = strings.Join(textFields[:1], " ")
					}
				}

				if e[k].RecalculateScore {
					e[k].ScoreFinal = 0
					e[k].ScoreFuzzy = 0
				}

				if e[k].ScoreFinal == 0 {
					switch e[k].Matching {
					case util.Fuzzy:
						e[k].ScoreFinal = fuzzyScore(e[k], toMatch, hyprland)
					case util.AlwaysTop:
						if e[k].ScoreFinal == 0 {
							e[k].ScoreFinal = 1000
						}
					case util.AlwaysBottom:
						if e[k].ScoreFinal == 0 {
							e[k].ScoreFinal = 1
						}
					default:
						e[k].ScoreFinal = 0
					}
				}

				if e[k].ScoreFinal != 0 {
					toPush = append(toPush, e[k])
				}
			}

			handler.receiver <- toPush
		}(ctx, &wg, text, p[k])
	}

	wg.Wait()
}

func setTypeahead(modules []modules.Workable) {
	if ui.search.Text() == "" {
		return
	}

	toSet := ""

	for _, v := range modules {
		if v.Typeahead() {
			tah := history.GetInputHistory(v.Name())

			trimmed := strings.TrimSpace(ui.search.Text())

			if trimmed != "" {
				for _, v := range tah {
					if strings.HasPrefix(v, trimmed) {
						toSet = v
					}
				}

				glib.IdleAdd(func() {
					if trimmed != toSet {
						ui.typeahead.SetText(toSet)
					}
				})
			}
		}
	}
}

func setInitials() {
	if len(appstate.ExplicitModules) > 0 {
		return
	}

	entrySlice := []util.Entry{}

	var hyprland *modules.Hyprland

	if cfg.Builtins.Hyprland.ContextAwareHistory && cfg.IsService {
		for _, proc := range toUse {
			if proc.Name() == "hyprland" {
				if proc.IsSetup() {
					hyprland = proc.(*modules.Hyprland)
				} else {
					ok := proc.Setup(cfg)

					if ok {
						hyprland = proc.(*modules.Hyprland)
					}
				}

				break
			}
		}
	}

	for _, proc := range toUse {
		if proc.Name() != "applications" {
			continue
		}

		if !proc.IsSetup() {
			proc.SetupData(cfg)
		}

		e := proc.Entries(nil, "")

		for _, entry := range e {
			for _, v := range hstry {
				if val, ok := v[entry.Identifier()]; ok {
					if entry.LastUsed.IsZero() || val.LastUsed.After(entry.LastUsed) {
						entry.Used = val.Used
						entry.DaysSinceUsed = val.DaysSinceUsed
						entry.LastUsed = val.LastUsed
					}
				}
			}

			entry.ScoreFinal = float64(usageModifier(entry))

			if cfg.Builtins.Hyprland.ContextAwareHistory && cfg.IsService && hyprland != nil {
				entry.OpenWindows = hyprland.GetWindowAmount(entry.InitialClass)
			}

			entrySlice = append(entrySlice, entry)
		}
	}

	if len(entrySlice) == 0 {
		return
	}

	sortEntries(entrySlice)

	ui.items.Splice(0, int(ui.items.NItems()), entrySlice...)
	ui.list.ScrollTo(0, gtk.ListScrollNone, nil)

	ui.spinner.SetCSSClasses([]string{})
}

func usageModifier(item util.Entry) int {
	base := 10

	if item.Used > 0 {
		if item.DaysSinceUsed > 0 {
			base -= item.DaysSinceUsed
		}

		return base * item.Used
	}

	return 0
}

func quit() {
	appstate.IsRunning = false
	appstate.IsSingle = false
	historyIndex = 0

	disabledAM()

	appstate.ExplicitModules = []string{}
	appstate.ExplicitPlaceholder = ""
	appstate.IsDmenu = false

	explicits = []modules.Workable{}

	glib.IdleAdd(func() {
		ui.search.SetText("")
		ui.search.SetObjectProperty("placeholder-text", cfg.Search.Placeholder)
		ui.appwin.SetVisible(false)
	})

	ui.app.Hold()
}

func exit() {
	ui.appwin.Close()
	os.Exit(0)
}

func createActivationKeys() {
	keys = make(map[uint]uint)
	keys[gdk.KEY_j] = 0
	keys[gdk.KEY_k] = 1
	keys[gdk.KEY_l] = 2
	keys[gdk.KEY_semicolon] = 3
	keys[gdk.KEY_a] = 4
	keys[gdk.KEY_s] = 5
	keys[gdk.KEY_d] = 6
	keys[gdk.KEY_f] = 7
	keys[gdk.KEY_J] = 0
	keys[gdk.KEY_K] = 1
	keys[gdk.KEY_L] = 2
	keys[gdk.KEY_colon] = 3
	keys[gdk.KEY_A] = 4
	keys[gdk.KEY_S] = 5
	keys[gdk.KEY_D] = 6
	keys[gdk.KEY_F] = 7
	keys[gdk.KEY_F1] = 0
	keys[gdk.KEY_F2] = 1
	keys[gdk.KEY_F3] = 2
	keys[gdk.KEY_F4] = 3
	keys[gdk.KEY_F5] = 4
	keys[gdk.KEY_F6] = 5
	keys[gdk.KEY_F7] = 6
	keys[gdk.KEY_F8] = 7
}

const modifier = 0.10

func fuzzyScore(entry util.Entry, text string, hyprland *modules.Hyprland) float64 {
	textLength := len(text)

	if textLength == 0 {
		return 1
	}

	var matchables []string

	if !appstate.IsDmenu {
		matchables = []string{entry.Label, entry.Sub, entry.Searchable}
		matchables = append(matchables, entry.Categories...)
	} else {
		matchables = []string{entry.Label}
	}

	multiplier := 0

	if hyprland != nil {
		entry.OpenWindows = hyprland.GetWindowAmount(entry.InitialClass)
	}

	for k, t := range matchables {

		if t == "" {
			continue
		}

		score := util.FuzzyScore(text, t)

		if score == 0 {
			continue
		}

		if score > entry.ScoreFuzzy {
			multiplier = k
			entry.ScoreFuzzy = score
		}
	}

	if entry.ScoreFuzzy == 0 {
		return 0
	}

	m := (1 - modifier*float64(multiplier))

	if m < 0.7 {
		m = 0.7
	}

	entry.ScoreFuzzy = entry.ScoreFuzzy * m

	for k, v := range hstry {
		if strings.HasPrefix(k, text) {
			if val, ok := v[entry.Identifier()]; ok {
				if entry.LastUsed.IsZero() || val.LastUsed.After(entry.LastUsed) {
					entry.Used = val.Used
					entry.DaysSinceUsed = val.DaysSinceUsed
					entry.LastUsed = val.LastUsed
				}
			}
		}
	}

	usageScore := usageModifier(entry)

	if textLength == 0 {
		textLength = 1
	}

	tm := 1.0 / float64(textLength)

	return float64(usageScore)*tm + float64(entry.ScoreFuzzy)/tm
}
