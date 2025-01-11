package ui

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/history"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/state"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var (
	activationEnabled       bool
	amLabel                 string
	commands                map[string]func() bool
	singleModule            modules.Workable
	tahSuggestionIdentifier string
	tahAcceptedIdentifier   string
	isAi                    bool
	blockTimeout            bool
	mouseX                  float64
	mouseY                  float64
)

func setupCommands() {
	commands = make(map[string]func() bool)
	commands["resethistory"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), history.HistoryName))
		hstry = history.Get()
		return true
	}
	commands["clearapplicationscache"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), "applications.json"))
		return true
	}
	commands["clearclipboard"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), "clipboard.gob"))
		return true
	}
	commands["cleartypeaheadcache"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), "inputhistory_0.7.6.gob"))
		return true
	}
	commands["adjusttheme"] = func() bool {
		blockTimeout = true

		cssFile := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", config.Cfg.Theme))

		cmd := exec.Command("sh", "-c", wrapWithPrefix(fmt.Sprintf("xdg-open %s", cssFile)))
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Pgid:       0,
			Foreground: false,
		}

		cmd.Start()

		return false
	}
}

var lastQuery = ""

func setupInteractions(appstate *state.AppState) {
	go setupCommands()
	parseKeybinds()

	elements.input.Connect("changed", func() {
		text := elements.input.Text()

		text = trimArgumentDelimiter(text)

		if lastQuery != text {
			executeEvent(config.EventQueryChange, "")
			lastQuery = text
		}

		if elements.clear != nil {
			if text == "" {
				elements.clear.SetVisible(false)
			} else {
				elements.clear.SetVisible(true)
			}
		}

		debouncedProcess(process)

		return
	})

	globalKeyController := gtk.NewEventControllerKey()
	globalKeyController.ConnectKeyReleased(handleGlobalKeysReleased)
	globalKeyController.ConnectKeyPressed(handleGlobalKeysPressed)
	globalKeyController.SetPropagationPhase(gtk.PropagationPhase(1))

	elements.appwin.AddController(globalKeyController)

	if !config.Cfg.IgnoreMouse {
		motion := gtk.NewEventControllerMotion()

		motion.ConnectMotion(func(x, y float64) {
			if mouseX == 0 || mouseY == 0 {
				mouseX = x
				mouseY = y
				return
			}

			if x != mouseX || y != mouseY {
				if !elements.grid.CanTarget() {
					elements.grid.SetCanTarget(true)
				}
			}
		})

		elements.appwin.AddController(motion)
	}

	if !config.Cfg.IgnoreMouse && !config.Cfg.DisableClickToClose {
		gesture := gtk.NewGestureClick()
		gesture.SetPropagationPhase(gtk.PropagationPhase(3))
		gesture.Connect("pressed", func(gesture *gtk.GestureClick, n int) {
			if appstate.IsService {
				quit(false)
			} else {
				exit(false, false)
			}
		})

		elements.appwin.AddController(gesture)
	}
}

func selectNext() bool {
	items := common.selection.NItems()

	if items == 0 {
		return false
	}

	disableMouseGtk()

	current := common.selection.Selected()
	next := current + 1

	if next < items {
		common.selection.SetSelected(current + 1)
	}

	if next >= items && config.Cfg.List.Cycle {
		common.selection.SetSelected(0)
	}

	return true
}

func selectPrev() bool {
	items := common.selection.NItems()

	if items == 0 {
		return false
	}

	disableMouseGtk()

	current := common.selection.Selected()

	if current > 0 {
		common.selection.SetSelected(current - 1)
	}

	if current == 0 && config.Cfg.List.Cycle {
		common.selection.SetSelected(items - 1)
	}

	return true
}

var fkeys = []uint{65470, 65471, 65472, 65473, 65474, 65475, 65476, 65477}

func selectActivationMode(keepOpen bool, isFKey bool, target uint) {
	if target < common.selection.NItems() {
		common.selection.SetSelected(target)
	}

	if keepOpen {
		activateItem(true, false)
		return
	}

	activateItem(false, false)
}

func enableAM() {
	if isAi {
		return
	}

	c := elements.appwin.CSSClasses()
	c = append(c, "activation")

	elements.appwin.SetCSSClasses(c)
	elements.grid.GrabFocus()

	activationEnabled = true
}

func disableAM() {
	if !config.Cfg.ActivationMode.Disabled && activationEnabled {
		activationEnabled = false
		elements.input.SetFocusable(false)

		c := elements.appwin.CSSClasses()

		for k, v := range c {
			if v == "activation" {
				c = slices.Delete(c, k, k+1)
			}
		}

		elements.appwin.SetCSSClasses(c)
		elements.input.GrabFocus()
	}
}

func handleGlobalKeysReleased(val, code uint, state gdk.ModifierType) {
	switch val {
	case uint(labelTrigger):
		disableAM()
	}
}

func handleGlobalKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	timeoutReset()

	if val == uint(labelTrigger) && !config.Cfg.ActivationMode.Disabled {
		enableAM()
		return true
	} else {
		switch val {
		case gdk.KEY_F1, gdk.KEY_F2, gdk.KEY_F3, gdk.KEY_F4, gdk.KEY_F5, gdk.KEY_F6, gdk.KEY_F7, gdk.KEY_F8:
			index := slices.Index(fkeys, val)

			if index != -1 {
				isShift := modifier == gdk.ShiftMask
				selectActivationMode(isShift, true, uint(index))
				return true
			}
		default:
			if !config.Cfg.ActivationMode.Disabled && activationEnabled {
				uc := gdk.KeyvalToUnicode(gdk.KeyvalToLower(val))

				if uc != 0 {
					index := slices.Index(appstate.UsedLabels, string(rune(uc)))

					if index != -1 {
						keepOpen := modifier == (keepOpenModifier | labelModifier)

						selectActivationMode(keepOpen, false, uint(index))
						return true
					}
				}
			}

			if val == gdk.KEY_ISO_Left_Tab {
				val = gdk.KEY_Tab
			}

			hasBind := binds.execute(int(val), modifier)

			if hasBind {
				return true
			}

			hasFocus := false

			focused := elements.appwin.Window.Focus()
			widget, ok := focused.(*gtk.Text)

			if ok {
				_, ok := widget.Parent().(*gtk.Entry)
				if ok {
					hasFocus = true
				}
			}

			if !hasFocus {
				elements.input.GrabFocus()
				char := gdk.KeyvalToUnicode(val)
				elements.input.SetText(elements.input.Text() + string(rune(char)))
				elements.input.SetPosition(-1)
			}

			return false
		}
	}

	return false
}

var historyIndex = 0

func activateItem(keepOpen, alt bool) {
	selectNext := !activationEnabled && keepOpen

	if elements.grid.Model().NItems() == 0 {
		return
	}

	entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))

	executeEvent(config.EventActivate, entry.Label)

	if !keepOpen && entry.Sub != "Walker" && entry.Sub != "switcher" && config.Cfg.IsService && entry.SpecialFunc == nil {
		go quit(true)
	}

	module := findModule(entry.Module, toUse, explicits)

	if entry.SpecialFunc != nil {
		args := []interface{}{}
		args = append(args, entry.SpecialFuncArgs...)
		args = append(args, elements.input.Text())

		if module.General().Name == config.Cfg.Builtins.AI.Name {
			elements.input.SetObjectProperty("placeholder-text", entry.Label)

			isAi = true

			glib.IdleAdd(func() {
				elements.input.SetText("")
				elements.scroll.SetVisible(false)
				elements.aiScroll.SetVisible(true)

				args = append(args, elements.aiList, common.aiItems, elements.spinner)

				go entry.SpecialFunc(args...)
			})
		} else {
			entry.SpecialFunc(args...)
			closeAfterActivation(keepOpen, selectNext)
		}

		return
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
		shouldClose := commands[entry.Exec]()

		if shouldClose {
			closeAfterActivation(keepOpen, selectNext)
		}

		return
	}

	if entry.Sub == "switcher" {
		handleSwitcher(entry.Label)
		return
	}

	if entry.Terminal || forceTerminal {
		if config.Cfg.TerminalTitleFlag != "" || entry.TerminalTitleFlag != "" {
			flag := config.Cfg.TerminalTitleFlag

			if flag == "" {
				flag = entry.TerminalTitleFlag
			}

			toRun = fmt.Sprintf("%s %s -e %s", config.Cfg.Terminal, flag, toRun)
		} else {
			toRun = fmt.Sprintf("%s -e %s", config.Cfg.Terminal, toRun)
		}
	}

	input := elements.input.Text()

	if strings.Contains(input, config.Cfg.Search.ArgumentDelimiter) {
		split := strings.Split(input, config.Cfg.Search.ArgumentDelimiter)
		input = split[0]
		toRun = fmt.Sprintf("%s %s", toRun, split[1])
	}

	cmd := exec.Command("sh", "-c", wrapWithPrefix(toRun))

	if entry.Path != "" {
		cmd.Dir = entry.Path
	}

	if len(entry.Env) > 0 {
		cmd.Env = append(os.Environ(), entry.Env...)
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

	identifier := entry.Identifier()

	if entry.History {
		hstry.Save(identifier, strings.TrimSpace(elements.input.Text()))
	}

	if module != nil && module.General().Typeahead {
		history.SaveInputHistory(module.General().Name, elements.input.Text(), identifier)
	}

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}

	closeAfterActivation(keepOpen, selectNext)
}

func handleSwitcher(module string) {
	for _, m := range toUse {
		if m.General().Name == module {
			explicits = []modules.Workable{}
			explicits = append(explicits, m)

			glib.IdleAdd(func() {
				common.items.Splice(0, int(common.items.NItems()))
				elements.input.SetObjectProperty("placeholder-text", m.General().Placeholder)

				setupSingleModule()

				if val, ok := layouts[singleModule.General().Name]; ok {
					layout = val
					setupLayout(singleModule.General().Theme, singleModule.General().ThemeBase)
				}

				if elements.input.Text() != "" {
					elements.input.SetText("")
				} else {
					debouncedProcess(process)
				}

				elements.input.GrabFocus()
			})
		}
	}
}

func handleDmenuResult(result string) {
	if appstate.IsService {
		for _, v := range toUse {
			if v.General().Name == "dmenu" {
				v.(*modules.Dmenu).Reply(result)
			}
		}
	} else {
		if result != "CNCLD" {
			fmt.Println(result)
		}
	}
}

func setStdin(cmd *exec.Cmd, piped *util.Piped) {
	if piped.String != "" {
		switch piped.Type {
		case "bytes":
			cmd.Stdin = bytes.NewReader(piped.Bytes)
		case "string":
			cmd.Stdin = strings.NewReader(piped.String)
		case "file":
			b, err := os.ReadFile(piped.String)
			if err != nil {
				log.Panic(err)
			}

			r := bytes.NewReader(b)
			cmd.Stdin = r
		}
	}
}

func closeAfterActivation(keepOpen, next bool) {
	if !config.Cfg.IsService && !keepOpen {
		exit(true, false)
	}

	if !keepOpen && appstate.IsRunning {
		quit(true)
		return
	}

	if appstate.IsRunning {
		if !activationEnabled && !next {
			elements.input.SetText("")
		}

		if next {
			selectNext()
		}
	}
}

func disableMouseGtk() {
	mouseX = 0
	mouseY = 0
	elements.grid.SetCanTarget(false)
}

func process() {
	disableMouseGtk()

	if isAi {
		return
	}

	elements.typeahead.SetText("")

	text := strings.TrimSpace(elements.input.Text())

	text = trimArgumentDelimiter(text)

	if text == "" && config.Cfg.List.ShowInitialEntries && len(explicits) == 0 && !appstate.IsDmenu {
		setInitials()
		return
	}

	if (text != "" || appstate.IsDmenu) || (len(explicits) > 0 && config.Cfg.List.ShowInitialEntries) {
		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(true)
		}

		go processAsync(text)
	} else {
		common.items.Splice(0, int(common.items.NItems()))

		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}
	}
}

var timeoutTimer *time.Timer

func timeoutReset() {
	if config.Cfg.Timeout > 0 {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}

		timeoutTimer = time.AfterFunc(time.Duration(config.Cfg.Timeout)*time.Second, func() {
			if appstate.IsRunning {
				if appstate.Password {
					fmt.Print("")
				}

				if appstate.IsDmenu {
					handleDmenuResult("")
				}

				if !isAi && !blockTimeout {
					if appstate.IsService {
						glib.IdleAdd(quit)
					} else {
						glib.IdleAdd(exit)
					}
				}
			}
		})
	}
}

func handleTimeout() {
	if config.Cfg.Timeout > 0 {
		if appstate.Password {
			elements.password.Connect("changed", timeoutReset)
			return
		}

		elements.input.Connect("search-changed", timeoutReset)

		scrollController := gtk.NewEventControllerScroll(gtk.EventControllerScrollBothAxes)
		scrollController.Connect("scroll", timeoutReset)

		elements.scroll.AddController(scrollController)
	}
}

var mut sync.Mutex

func processAsync(text string) {
	entries := []util.Entry{}

	defer func() {
		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}
	}()

	hasExplicit := len(explicits) > 0

	p := toUse

	query := text
	hasPrefix := false

	prefixes := []string{}

	for _, v := range p {
		prefix := v.General().Prefix

		if len(prefix) > 0 && strings.HasPrefix(text, prefix) {
			prefixes = append(prefixes, prefix)
			hasPrefix = true
		}
	}

	if !hasExplicit {
		if hasPrefix {
			glib.IdleAdd(func() {
				for _, v := range prefixes {
					elements.appwin.SetCSSClasses(elements.prefixClasses[v])
				}
			})
		}
	} else {
		p = explicits
	}

	setTypeahead(p)

	var wg sync.WaitGroup
	wg.Add(len(p))

	keepSort := false || appstate.KeepSort

	if len(p) == 1 {
		keepSort = p[0].General().KeepSort || appstate.KeepSort
		appstate.IsSingle = true
	}

	hasEntryPrefix := false

	for k := range p {
		if p[k] == nil {
			wg.Done()
			continue
		}

		if len(p) > 1 {
			prefix := p[k].General().Prefix

			if p[k].General().SwitcherOnly {
				if prefix == "" {
					wg.Done()
					continue
				}

				if !strings.HasPrefix(text, prefix) {
					wg.Done()
					continue
				}
			}

			if hasPrefix && prefix == "" {
				wg.Done()
				continue
			}

			if !hasPrefix && prefix != "" {
				wg.Done()
				continue
			}

			if hasPrefix && !strings.HasPrefix(text, prefix) {
				wg.Done()
				continue
			}
		}

		if !p[k].General().IsSetup {
			p[k].SetupData()
		}

		go func(wg *sync.WaitGroup, text string, w modules.Workable) {
			defer wg.Done()

			mCfg := w.General()

			if len(text) < mCfg.MinChars {
				return
			}

			text = strings.TrimPrefix(text, w.General().Prefix)

			e := w.Entries(text)

			toPush := []util.Entry{}
			g := w.General()

		outer:
			for k := range e {
				if len(mCfg.Blacklist) > 0 {
					for _, b := range mCfg.Blacklist {
						if !b.Label && !b.Sub {
							if b.Reg.MatchString(e[k].Label) {
								continue outer
							}

							if b.Reg.MatchString(e[k].Sub) {
								continue outer
							}
						}

						if b.Label {
							if b.Reg.MatchString(e[k].Label) {
								continue outer
							}
						}

						if b.Sub {
							if b.Reg.MatchString(e[k].Sub) {
								continue outer
							}
						}
					}
				}

				if e[k].SingleModuleOnly && singleModule == nil {
					continue
				}

				e[k].Module = g.Name
				e[k].Weight = g.Weight

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
					case util.AlwaysTopOnEmptySearch:
						if text != "" {
							e[k].ScoreFinal = fuzzyScore(&e[k], toMatch, g.History)
						} else {
							e[k].ScoreFinal = 1000
						}
					case util.Fuzzy:
						e[k].ScoreFinal = fuzzyScore(&e[k], toMatch, g.History)
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

				if toMatch == "" {
					if e[k].ScoreFinal != 0 {
						if e[k].Prefix != "" && strings.HasPrefix(text, e[k].Prefix) {
							hasEntryPrefix = true

							toPush = append(toPush, e[k])
						} else {
							if e[k].IgnoreUnprefixed {
								continue
							}

							toPush = append(toPush, e[k])
						}
					}
				} else {
					if e[k].ScoreFinal > float64(config.Cfg.List.VisibilityThreshold) {
						if e[k].Prefix != "" && strings.HasPrefix(text, e[k].Prefix) {
							hasEntryPrefix = true

							toPush = append(toPush, e[k])
						} else {
							if e[k].IgnoreUnprefixed {
								continue
							}

							toPush = append(toPush, e[k])
						}
					}
				}
			}

			mut.Lock()
			entries = append(entries, toPush...)
			mut.Unlock()
		}(&wg, text, p[k])
	}

	wg.Wait()

	if query != lastQuery {
		return
	}

	if hasEntryPrefix {
		finalEntries := []util.Entry{}

		for _, v := range entries {
			if v.Prefix != "" {
				finalEntries = append(finalEntries, v)
			}
		}

		entries = finalEntries
	}

	if !keepSort || text != "" {
		sortEntries(entries, keepSort)
	}

	if len(entries) > config.Cfg.List.MaxEntries {
		entries = entries[:config.Cfg.List.MaxEntries]
	}

	if appstate.IsDebug {
		for _, v := range entries {
			fmt.Printf("Entries == label: %s sub: %s score: %f\n", v.Label, v.Sub, v.ScoreFinal)
		}
	}

	glib.IdleAdd(func() {
		common.items.Splice(0, int(common.items.NItems()), entries...)

		if config.Cfg.IgnoreMouse && !elements.grid.CanTarget() {
			for _, v := range entries {
				if v.DragDrop {
					elements.grid.SetCanTarget(true)
					break
				}
			}
		} else if config.Cfg.IgnoreMouse {
			elements.grid.SetCanTarget(false)
		}

		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}
	})

	tahAcceptedIdentifier = ""
}

func setTypeahead(modules []modules.Workable) {
	if elements.input.Text() == "" {
		return
	}

	toSet := ""

	for _, v := range modules {
		if v.General().Typeahead {
			tah := history.GetInputHistory(v.General().Name)

			trimmed := strings.TrimSpace(elements.input.Text())

			if trimmed != "" {
				for _, v := range tah {
					if strings.HasPrefix(v.Term, trimmed) {
						toSet = v.Term
						tahSuggestionIdentifier = v.Identifier
					}
				}

				glib.IdleAdd(func() {
					if trimmed != toSet {
						elements.typeahead.SetText(toSet)
					}
				})
			}
		}
	}
}

func setInitials() {
	entries := []util.Entry{}

	proc := findModule("applications", toUse)

	if proc == nil {
		return
	}

	if !proc.General().IsSetup {
		proc.SetupData()
	}

	e := proc.Entries("")

	for _, entry := range e {
		entry.Module = proc.General().Name
		entry.MatchedLabel = ""
		entry.MatchedSub = ""
		entry.ScoreFinal = 0

		if proc.General().History {
			for _, v := range hstry {
				if val, ok := v[entry.Identifier()]; ok {
					if entry.LastUsed.IsZero() || val.LastUsed.After(entry.LastUsed) {
						entry.Used = val.Used
						entry.DaysSinceUsed = val.DaysSinceUsed
						entry.LastUsed = val.LastUsed
					}
				}
			}

			entry.ScoreFinal = float64(usageModifier(&entry))
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return
	}

	sortEntries(entries, false)

	glib.IdleAdd(func() {
		common.items.Splice(0, int(common.items.NItems()), entries...)
	})
}

func usageModifier(item *util.Entry) int {
	base := 10

	if item.Used > 0 {
		if item.DaysSinceUsed > 0 {
			base -= item.DaysSinceUsed
		}

		return base * item.Used
	}

	return 0
}

func quit(ignoreEvent bool) {
	if !ignoreEvent {
		executeEvent(config.EventExit, "")
	}

	if timeoutTimer != nil {
		timeoutTimer.Stop()
	}

	timeoutTimer = nil

	if singleModule != nil {
		if _, ok := layouts[singleModule.General().Name]; ok {
			glib.IdleAdd(func() {
				layout, _ = config.GetLayout(config.Cfg.Theme, config.Cfg.ThemeBase)
				setupLayout(config.Cfg.Theme, config.Cfg.ThemeBase)
			})
		}

		resetSingleModule()
	}

	appstate.IsRunning = false
	appstate.IsSingle = false
	appstate.AutoSelect = false

	historyIndex = 0

	for _, v := range toUse {
		go v.Cleanup()
	}

	disableAM()

	appstate.ExplicitModules = []string{}
	appstate.ExplicitPlaceholder = ""
	appstate.ExplicitTheme = ""
	appstate.IsDmenu = false

	explicits = []modules.Workable{}

	if appstate.IsService && elements.input.Text() != "" {
		appstate.LastQuery = elements.input.Text()
	}

	glib.IdleAdd(func() {
		if layout != nil {
			if !layout.Window.Box.Search.Spinner.Hide {
				elements.spinner.SetVisible(false)
			}
		}

		if !config.Cfg.Search.ResumeLastQuery {
			elements.input.SetText("")
			elements.input.SetObjectProperty("placeholder-text", config.Cfg.Search.Placeholder)
		} else {
			elements.input.SelectRegion(0, -1)
		}

		elements.appwin.SetVisible(false)

		elements.scroll.SetVisible(true)
		elements.aiScroll.SetVisible(false)
	})

	isAi = false
	blockTimeout = false

	common.app.Hold()
}

func exit(ignoreEvent bool, cancel bool) {
	code := 0

	if cancel {
		code = 2
	}

	if !ignoreEvent {
		executeEvent(config.EventExit, "")
	}

	elements.appwin.Close()

	os.Exit(code)
}

const modifier = 0.10

func fuzzyScore(entry *util.Entry, text string, useHistory bool) float64 {
	textLength := len(text)

	entry.MatchedLabel = ""
	entry.MatchedSub = ""

	if entry.Prefix != "" {
		if strings.HasPrefix(text, entry.Prefix) {
			text = strings.TrimPrefix(text, entry.Prefix)
		}
	}

	if textLength == 0 {
		return 1
	}

	var matchables []string

	if !appstate.IsDmenu {
		matchables = []string{entry.Label, entry.Sub, entry.Searchable, entry.Searchable2}
		matchables = append(matchables, entry.Categories...)
	} else {
		matchables = []string{entry.Label}
	}

	multiplier := 0

	var pos *[]int

	for k, t := range matchables {
		if t == "" {
			continue
		}

		remember := ""

		if k == 0 && singleModule != nil && singleModule.General().Name == config.Cfg.Builtins.Emojis.Name {
			remember = strings.Fields(t)[0]
			t = entry.Searchable
		}

		var score float64

		if strings.HasPrefix(text, "'") {
			cleanText := strings.TrimPrefix(text, "'")

			score, _ = util.ExactScore(cleanText, t)

			f := strings.Index(strings.ToLower(t), strings.ToLower(cleanText))

			if f != -1 {
				poss := []int{}

				for i := f; i < f+len(text); i++ {
					poss = append(poss, i)
				}

				pos = &poss
			}
		} else {
			score, pos = util.FuzzyScore(text, t)
		}

		if score < 2 {
			continue
		}

		if score > entry.ScoreFuzzy {
			multiplier = k

			if config.Cfg.List.DynamicSub && k > 1 {
				entry.MatchedSub = t
			}

			if layout.Window.Box.Scroll.List.MarkerColor != "" {
				res := ""

				if pos != nil {
					for k, v := range t {
						if slices.Contains(*pos, k) {
							res = fmt.Sprintf("%s<span color=\"%s\">%s</span>", res, layout.Window.Box.Scroll.List.MarkerColor, string(v))
						} else {
							res = fmt.Sprintf("%s%s", res, string(v))
						}
					}
				}

				if remember != "" {
					res = fmt.Sprintf("%s %s", remember, res)
				}

				if k == 0 {
					entry.MatchedLabel = res
				} else if k > 0 {
					entry.MatchedSub = res
				}
			}

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

	usageScore := 0

	if useHistory {
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

		usageScore = usageModifier(entry)
	}

	if textLength == 0 {
		textLength = 1
	}

	tm := 1.0 / float64(textLength)

	score := float64(usageScore)*tm + float64(entry.ScoreFuzzy)/tm

	if appstate.IsDebug {
		fmt.Printf("Matching == label: %s sub: %s searchable: %s categories: %s score: %f usage: %d fuzzy: %f m: %f\n", entry.Label, entry.Sub, entry.Searchable, entry.Categories, score, usageScore, entry.ScoreFuzzy, m)
	}

	return score
}

func wrapWithPrefix(text string) string {
	if config.Cfg.AppLaunchPrefix == "" {
		return text
	}

	return fmt.Sprintf("%s%s", config.Cfg.AppLaunchPrefix, text)
}

func trimArgumentDelimiter(text string) string {
	if strings.Contains(text, config.Cfg.Search.ArgumentDelimiter) {
		split := strings.Split(text, config.Cfg.Search.ArgumentDelimiter)

		text = split[0]

		if text == "" && len(split) > 1 {
			text = split[1]
		}
	}

	return text
}

func executeOnSelect(entry util.Entry) {
	if singleModule == nil || !appstate.IsRunning {
		return
	}

	if singleModule.General().OnSelect != "" {
		val := entry.Label

		if entry.Value != "" {
			val = entry.Value
		}

		toRun := singleModule.General().OnSelect

		explicit := false

		if strings.Contains(singleModule.General().OnSelect, "%RESULT%") {
			toRun = strings.ReplaceAll(singleModule.General().OnSelect, "%RESULT%", val)
			explicit = true
		}

		cmd := exec.Command("sh", "-c", toRun)

		if !explicit {
			setStdin(cmd, &util.Piped{String: val, Type: "string"})
		}

		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
	}
}
