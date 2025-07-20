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
	"github.com/abenz1267/walker/internal/modules/clipboard"
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
	setWindowClasses        []string
)

func setupCommands() {
	commands = make(map[string]func() bool)
	commands["resethistory"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), history.HistoryName))
		hstry = history.Get()
		return true
	}
	commands["clearapplicationscache"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), "applications.gob"))
		return true
	}
	commands["clearclipboard"] = func() bool {
		m := findModule("clipboard", available)
		m.(*clipboard.Clipboard).Clear()

		return true
	}
	commands["cleartypeaheadcache"] = func() bool {
		os.Remove(filepath.Join(util.CacheDir(), "inputhistory_0.7.6.gob"))
		return true
	}
	commands["adjusttheme"] = func() bool {
		blockTimeout = true

		dir, root := util.ThemeDir()
		cssFile := filepath.Join(dir, fmt.Sprintf("%s.css", config.Cfg.Theme))

		if root {
			return false
		}

		cmd := exec.Command("sh", "-c", fmt.Sprintf("xdg-open %s", cssFile))
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Pgid:       0,
			Foreground: false,
		}

		cmd.Start()

		go func() {
			cmd.Wait()
		}()

		return false
	}
}

var lastQuery = ""

func setupInteractions(appstate *state.AppState) {
	go setupCommands()
	parseKeybinds()

	elements.input.Connect("changed", func() {
		if appstate.Hidebar {
			return
		}

		text := trimArgumentDelimiter(elements.input.Text())

		if lastQuery == text {
			return
		}

		executeEvent(config.EventQueryChange, "")
		lastQuery = text

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

	elements.appwin.AddCSSClass("activation")
	elements.grid.GrabFocus()

	activationEnabled = true
}

func disableAM(refocus bool) {
	if !config.Cfg.ActivationMode.Disabled && activationEnabled {
		activationEnabled = false

		glib.IdleAdd(func() {
			elements.appwin.RemoveCSSClass("activation")

			if refocus {
				elements.input.SetFocusable(true)
				elements.input.GrabFocus()
			}
		})
	}
}

func handleGlobalKeysReleased(val, code uint, state gdk.ModifierType) {
	switch val {
	case uint(labelTrigger):
		disableAM(true)
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
			if config.Cfg.ActivationMode.UseFKeys {
				index := slices.Index(fkeys, val)

				if index != -1 {
					isShift := modifier == gdk.ShiftMask
					selectActivationMode(isShift, true, uint(index))
					return true
				}
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

	entry := gioutil.ObjectValue[*util.Entry](common.items.Item(common.selection.Selected()))

	executeEvent(config.EventActivate, entry.Label)

	if !keepOpen && entry.Sub != "Walker" && entry.Sub != "switcher" && config.Cfg.IsService && entry.SpecialFunc == nil {
		go quit(true)
	}

	module := findModule(entry.Module, toUse, explicits)

	if entry.SpecialFunc != nil {
		args := []interface{}{}
		args = append(args, entry.SpecialFuncArgs...)
		args = append(args, elements.input.Text())

		switch module.General().Name {
		case config.Cfg.Builtins.AI.Name:
			elements.input.SetObjectProperty("placeholder-text", entry.Label)

			isAi = true

			glib.IdleAdd(func() {
				elements.input.SetText("")
				elements.scroll.SetVisible(false)
				elements.aiScroll.SetVisible(true)

				args = append(args, elements.aiList, common.aiItems, elements.spinner)

				go entry.SpecialFunc(args...)
			})
		case config.Cfg.Builtins.Translation.Name:
			entry.SpecialFunc(entry.Label)
			closeAfterActivation(keepOpen, selectNext)
		default:
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

	// check if desktop app is terminal
	if entry.Module == config.Cfg.Builtins.Finder.Name && !alt {
		forceTerminal = forceTerminalForFile(strings.TrimPrefix(entry.Exec, "xdg-open "))
	}

	if appstate.IsDmenu {
		handleDmenuResult(entry.Value)
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

	cmd := exec.Command("sh", "-c", util.WrapWithPrefix(config.Cfg.AppLaunchPrefix, toRun))

	if entry.Path != "" {
		cmd.Dir = entry.Path
	}

	if len(entry.Env) > 0 {
		cmd.Env = append(os.Environ(), entry.Env...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	setStdin(cmd, &entry.Piped)

	if alt {
		setStdin(cmd, &entry.PipedAlt)
	}

	identifier := entry.Identifier()

	mCfg := module.General()

	if mCfg.History {
		canSave := true

		if len(mCfg.HistoryBlacklist) > 0 {
			for _, b := range mCfg.HistoryBlacklist {
				if b.Match(entry) {
					canSave = false
					break
				}
			}
		}

		if canSave {
			hstry.Save(identifier, strings.TrimSpace(elements.input.Text()))
		}
	}

	if module != nil && module.General().Typeahead {
		history.SaveInputHistory(module.General().Name, elements.input.Text(), identifier)
	}

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	} else {
		go func() {
			cmd.Wait()
		}()
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

				if val, ok := mergedLayouts[singleModule.General().Name]; ok {
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
				appstate.DmenuResultChan <- result
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

	text := elements.input.Text()

	text = trimArgumentDelimiter(text)

	isEmpty := strings.TrimSpace(text) == ""

	if text == "" && config.Cfg.List.ShowInitialEntries && len(explicits) == 0 && !appstate.IsDmenu {
		setInitials()
		return
	}

	if ((!isEmpty && text != "") || appstate.IsDmenu) || (len(explicits) > 0 && config.Cfg.List.ShowInitialEntries) {
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
	entries := []*util.Entry{}

	defer func() {
		if !layout.Window.Box.Search.Spinner.Hide {
			glib.IdleAdd(func() {
				elements.spinner.SetVisible(false)
			})
		}
	}()

	p := toUse
	query := text
	queryHasPrefix := false
	prefixes := []string{}

	if singleModule == nil {
		for _, v := range p {
			prefix := v.General().Prefix

			if len(prefix) > 0 && strings.HasPrefix(text, prefix) {
				prefixes = append(prefixes, prefix)
				queryHasPrefix = true
			}
		}

		if queryHasPrefix {
			glib.IdleAdd(func() {
				for _, v := range prefixes {
					for _, class := range elements.prefixClasses[v] {
						elements.appwin.AddCSSClass(class)
						setWindowClasses = append(setWindowClasses, class)
					}
				}
			})
		}
	} else {
		p = []modules.Workable{singleModule}
	}

	setTypeahead(p)

	var wg sync.WaitGroup
	wg.Add(len(p))

	keepSort := false || appstate.KeepSort

	processedModulesKeepSort := []bool{}

	if len(p) == 1 {
		keepSort = p[0].General().KeepSort || appstate.KeepSort
		appstate.IsSingle = true
	}

	for k := range p {
		if p[k] == nil {
			wg.Done()
			continue
		}

		if len(p) > 1 {
			prefix := p[k].General().Prefix

			if len(appstate.ExplicitModules) == 0 {
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
			}

			if queryHasPrefix && prefix == "" {
				wg.Done()
				continue
			}

			if !queryHasPrefix && prefix != "" {
				wg.Done()
				continue
			}

			if queryHasPrefix && !strings.HasPrefix(text, prefix) {
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

			processedModulesKeepSort = append(processedModulesKeepSort, mCfg.KeepSort)

			if singleModule == nil && len(text) < mCfg.MinChars {
				return
			}

			text = strings.TrimSpace(strings.TrimPrefix(text, w.General().Prefix))
			toPush := []*util.Entry{}

			e := w.Entries(text)

			for k := range e {
				if evaluateEntry(text, e[k], mCfg) {
					toPush = append(toPush, e[k])
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

	populateList(text, keepSort, processedModulesKeepSort, entries)

	tahAcceptedIdentifier = ""
}

func populateList(text string, keepSort bool, processedModulesKeepSort []bool, entries []*util.Entry) {
	if len(processedModulesKeepSort) > 1 {
		if !keepSort || text != "" {
			sortEntries(entries, keepSort, false)
		}
	} else {
		if processedModulesKeepSort[0] != true {
			sortEntries(entries, keepSort, false)
		}
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
}

func evaluateEntry(text string, entry *util.Entry, cfg *config.GeneralModule) bool {
	if cfg.Blacklist.Contains(entry) {
		return false
	}

	if entry.SingleModuleOnly && singleModule == nil {
		return false
	}

	entry.Module = cfg.Name
	entry.Weight = cfg.Weight

	toMatch := text

	if entry.MatchFields > 0 {
		textFields := strings.Fields(text)

		if len(textFields) > 0 {
			toMatch = strings.Join(textFields[:1], " ")
		}
	}

	if entry.RecalculateScore {
		entry.ScoreFinal = 0
		entry.ScoreFuzzy = 0
		entry.MatchStartingPos = 0
	}

	if entry.ScoreFinal == 0 {
		switch entry.Matching {
		case util.AlwaysTopOnEmptySearch:
			if text != "" {
				entry.ScoreFinal = fuzzyScore(entry, toMatch, cfg.History)
			} else {
				entry.ScoreFinal = 1000
			}
		case util.Fuzzy, util.TopWhenFuzzyMatch:
			entry.ScoreFinal = fuzzyScore(entry, toMatch, cfg.History)

			if entry.Matching == util.TopWhenFuzzyMatch {
				if entry.ScoreFinal > 0 {
					entry.ScoreFinal = 10000
				}
			}
		case util.AlwaysTop:
			if entry.ScoreFinal == 0 {
				entry.ScoreFinal = 10000
			}
		case util.AlwaysBottom:
			if entry.ScoreFinal == 0 {
				entry.ScoreFinal = 1
			}
		default:
			entry.ScoreFinal = 0
		}
	}

	if toMatch == "" {
		if entry.ScoreFinal != 0 || config.Cfg.List.ShowInitialEntries {
			if entry.Prefix != "" && strings.HasPrefix(text, entry.Prefix) {
				return matchesVisibilityThreshold(text, cfg.Prefix, entry)
			} else {
				if entry.IgnoreUnprefixed {
					return false
				}

				return matchesVisibilityThreshold(text, cfg.Prefix, entry)
			}
		}
	} else {
		if entry.Prefix != "" && strings.HasPrefix(text, entry.Prefix) {
			return matchesVisibilityThreshold(text, cfg.Prefix, entry)
		} else {
			if entry.IgnoreUnprefixed {
				return false
			}

			return matchesVisibilityThreshold(text, cfg.Prefix, entry)
		}
	}

	return false
}

func matchesVisibilityThreshold(text, prefix string, entry *util.Entry) bool {
	if text == "" {
		return true
	}

	if prefix != "" && strings.HasPrefix(text, prefix) {
		if strings.TrimPrefix(text, prefix) == "" {
			return true
		}
	}

	if entry.ScoreFinal > float64(config.Cfg.List.VisibilityThreshold) {
		return true
	}

	return false
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
	entries := []*util.Entry{}

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

			entry.ScoreFinal = float64(usageModifier(entry))
			// fmt.Println(entry.ScoreFinal, entry.Label)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return
	}

	sortEntries(entries, false, true)

	glib.IdleAdd(func() {
		common.items.Splice(0, int(common.items.NItems()), entries...)
	})
}

func usageModifier(entry *util.Entry) int {
	base := 10

	if entry.Used > 0 {
		if entry.DaysSinceUsed > 0 {
			base -= entry.DaysSinceUsed
		}

		res := base * entry.Used

		if res < 1 {
			res = 1
		}

		return res
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
		glib.IdleAdd(func() {
			layout, _ = config.GetLayout(config.Cfg.Theme, config.Cfg.ThemeBase)
			setupLayout(config.Cfg.Theme, config.Cfg.ThemeBase)
		})

		resetSingleModule()
	}

	appstate.IsRunning = false
	appstate.IsSingle = false
	appstate.AutoSelect = false
	appstate.Hidebar = false

	historyIndex = 0

	for _, v := range toUse {
		go v.Cleanup()
	}

	disableAM(false)

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

		common.items.Splice(0, int(common.items.NItems()))
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

	moduleName := entry.Module

	remember := ""

	if textLength != 0 {
		var matchables []string

		if moduleName == config.Cfg.Builtins.Emojis.Name || moduleName == config.Cfg.Builtins.Symbols.Name {
			matchables = []string{entry.Searchable}
			matchables = append(matchables, entry.Categories...)
			remember = strings.Split(entry.Label, entry.Searchable)[0]
		} else {
			if !appstate.IsDmenu {
				matchables = []string{entry.Sub, entry.Searchable, entry.Searchable2}
				matchables = append(matchables, entry.Categories...)

				if entry.Output == "" {
					matchables = append([]string{entry.Label}, matchables...)
				}
			} else {
				matchables = []string{entry.Label}
			}
		}

		var pos *[]int

		splits := strings.Split(text, ";")

		totalScore := 0.0
		start := 0

		for _, text := range splits {
			matchScore := 0.0

			for k, t := range matchables {
				if t == "" {
					continue
				}

				var score float64

				if strings.HasPrefix(text, "'") {
					cleanText := strings.TrimPrefix(text, "'")

					score, _, start = util.ExactScore(cleanText, t)

					f := strings.Index(strings.ToLower(t), strings.ToLower(cleanText))

					if f != -1 {
						poss := []int{}

						for i := f; i < f+len(text); i++ {
							poss = append(poss, i)
						}

						pos = &poss
					}
				} else {
					score, pos, start = util.FuzzyScore(text, t)
				}

				if score < 1 {
					continue
				}

				m := (1 - modifier*float64(k))

				if m < 0.7 {
					m = 0.7
				}

				score = score * m

				if score > matchScore {
					if config.Cfg.List.DynamicSub && k > 1 {
						entry.MatchedSub = t
					}

					if len(splits) == 1 && layout.Window.Box.Scroll.List.MarkerColor != "" && len(t) < 1000 {
						res := ""

						if pos != nil {
							for k, v := range []rune(t) {
								if slices.Contains(*pos, k) {
									res = fmt.Sprintf("%s|MARKERSTART|%s|MARKEREND|", res, string(v))
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

					matchScore = score
				}
			}

			totalScore += matchScore
		}

		entry.ScoreFuzzy = totalScore

		if len(splits) == 1 {
			entry.MatchStartingPos = start
		}

		if entry.ScoreFuzzy == 0 {
			return 0
		}
	}

	usageScore := 0

	// so old `Used` values don't persist, in case of a change in history
	oldUsed := entry.Used
	entry.Used = 0

	if useHistory {
		for k, v := range hstry {
			if strings.HasPrefix(k, text) {
				if val, ok := v[entry.Identifier()]; ok {
					entry.Used = val.Used
					entry.DaysSinceUsed = val.DaysSinceUsed
					entry.LastUsed = val.LastUsed
				}
			}
		}

		usageScore = usageModifier(entry)

		if entry.Used == 0 {
			entry.Used = oldUsed
		}

		if textLength == 0 {
			return float64(usageScore)
		}
	}

	if entry.Used == 0 {
		entry.Used = oldUsed
	}

	if textLength == 0 {
		textLength = 1
	}

	tm := 1.0 / float64(textLength)

	score := float64(usageScore)*tm + float64(entry.ScoreFuzzy)/tm

	if appstate.IsDebug {
		fmt.Printf("Matching == label: %s sub: %s searchable: %s categories: %s score: %f usage: %d fuzzy: %f m: %f\n", entry.Label, entry.Sub, entry.Searchable, entry.Categories, score, usageScore, entry.ScoreFuzzy)
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

func executeOnSelect(entry *util.Entry) {
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

		go func() {
			cmd.Wait()
		}()
	}
}

func forceTerminalForFile(file string) bool {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("xdg-mime query default $(xdg-mime query filetype %s)", file))

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	cmd.Dir = homedir

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		log.Println(string(out))
		return false
	}

	desktopFile := strings.TrimSpace(string(out))

	_, ok := modules.TerminalApps[desktopFile]

	return ok
}
