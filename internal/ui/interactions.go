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
	amKey                   uint
	cmdAltModifier          gdk.ModifierType
	amModifier              gdk.ModifierType
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

		cssFile := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", cfg.Theme))

		cmd := exec.Command("sh", "-c", wrapWithUWSM(fmt.Sprintf("xdg-open %s", cssFile)))
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

	keycontroller := gtk.NewEventControllerKey()
	keycontroller.SetPropagationPhase(gtk.PropagationPhase(1))

	elements.input.AddController(keycontroller)
	elements.input.Connect("search-changed", func() {
		text := elements.input.Text()

		if lastQuery != text {
			executeEvent(config.EventQueryChange, "")
			lastQuery = text
		}

		process()
	})

	amKey = gdk.KEY_Control_L
	amModifier = gdk.ControlMask
	amLabel = "Ctrl + "

	cmdAltModifier = gdk.AltMask

	if cfg.ActivationMode.UseAlt {
		amKey = gdk.KEY_Alt_L
		amModifier = gdk.AltMask
		amLabel = "Alt + "
		cmdAltModifier = gdk.ControlMask
	}

	globalKeyReleasedController := gtk.NewEventControllerKey()
	globalKeyReleasedController.ConnectKeyReleased(handleGlobalKeysReleased)
	globalKeyReleasedController.SetPropagationPhase(gtk.PropagationPhase(1))

	globalKeyController := gtk.NewEventControllerKey()
	globalKeyController.ConnectKeyPressed(handleGlobalKeysPressed)
	globalKeyController.SetPropagationPhase(gtk.PropagationPhase(1))

	elements.appwin.AddController(globalKeyController)
	elements.appwin.AddController(globalKeyReleasedController)

	if !cfg.IgnoreMouse {
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

	if !cfg.IgnoreMouse && !cfg.DisableClickToClose {
		gesture := gtk.NewGestureClick()
		gesture.SetPropagationPhase(gtk.PropagationPhase(3))
		gesture.Connect("pressed", func(gesture *gtk.GestureClick, n int) {
			if appstate.IsService {
				quit(false)
			} else {
				exit(false)
			}
		})

		elements.appwin.AddController(gesture)
	}
}

func selectNext() {
	items := common.selection.NItems()

	if items == 0 {
		return
	}

	current := common.selection.Selected()
	next := current + 1

	if next < items {
		common.selection.SetSelected(current + 1)
		elements.grid.ScrollTo(common.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}

	if next >= items && cfg.List.Cycle {
		common.selection.SetSelected(0)
		elements.grid.ScrollTo(common.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}
}

func selectPrev() {
	items := common.selection.NItems()

	if items == 0 {
		return
	}

	current := common.selection.Selected()

	if current > 0 {
		common.selection.SetSelected(current - 1)
		elements.grid.ScrollTo(common.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}

	if current == 0 && cfg.List.Cycle {
		common.selection.SetSelected(items - 1)
		elements.grid.ScrollTo(common.selection.Selected(), gtk.ListScrollNone, nil)
		return
	}
}

var fkeys = []uint{65470, 65471, 65472, 65473, 65474, 65475, 65476, 65477}

func selectActivationMode(keepOpen bool, isFKey bool, target uint) {
	if target < common.selection.NItems() {
		common.selection.SetSelected(target)
	}

	if keepOpen {
		activateItem(true, false, false)
		return
	}

	activateItem(false, false, false)
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
	if !cfg.ActivationMode.Disabled && activationEnabled {
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
	case amKey:
		disableAM()
	}
}

func handleGlobalKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	timeoutReset()

	switch val {
	case amKey:
		if cfg.ActivationMode.Disabled {
			return false
		}

		if common.selection.NItems() != 0 {
			if val == amKey {
				enableAM()
				return true
			}
		}
	case gdk.KEY_BackSpace:
		if modifier == gdk.ShiftMask {
			if singleModule == nil {

				entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
				hstry.Delete(entry.Identifier())
			} else {
				switch singleModule.General().Name {
				case cfg.Builtins.Clipboard.GeneralModule.Name:

					entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
					singleModule.(*clipboard.Clipboard).Delete(entry)
					process()
				}
			}

			return true
		}
	case gdk.KEY_c:
		if modifier == gdk.ControlMask {
			if isAi {
				ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
				ai.CopyLastResponse()
			}
		}
	case gdk.KEY_e:
		if modifier == gdk.ControlMask {
			if isAi {
				ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
				ai.RunLastMessageInTerminal()
				quit(true)
			}
		}
	case gdk.KEY_m:
		if modifier == gdk.ControlMask {
			if strings.HasPrefix(elements.input.Text(), "'") {
				elements.input.SetText(strings.TrimPrefix(elements.input.Text(), "'"))
			} else {
				elements.input.SetText("'" + elements.input.Text())
			}

			elements.input.SetPosition(-1)
		}
	case gdk.KEY_r:
		if modifier == gdk.ControlMask {
			if isAi {
				ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
				ai.ResumeLastMessages()
			} else {
				if appstate.IsService {
					elements.input.SetText(appstate.LastQuery)
					elements.input.SetPosition(-1)
					elements.input.GrabFocus()
				}
			}
		}
	case gdk.KEY_x:
		if modifier == gdk.ControlMask {
			if isAi {
				ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
				elements.input.SetText("")
				ai.ClearCurrent()
			}
		}
	case gdk.KEY_Escape:
		if appstate.IsDmenu {
			handleDmenuResult("")
		}

		if cfg.IsService {
			quit(false)
			return true
		} else {
			exit(false)
			return true
		}
	case gdk.KEY_F1, gdk.KEY_F2, gdk.KEY_F3, gdk.KEY_F4, gdk.KEY_F5, gdk.KEY_F6, gdk.KEY_F7, gdk.KEY_F8:
		index := slices.Index(fkeys, val)

		if index != -1 {
			isShift := modifier == gdk.ShiftMask
			selectActivationMode(isShift, true, uint(index))
			return true
		}
	case gdk.KEY_Return:
		isShift := modifier == gdk.ShiftMask
		isAlt := modifier == cmdAltModifier

		isAltShift := modifier == (gdk.ShiftMask | cmdAltModifier)

		if isAltShift {
			isShift = true
			isAlt = true
		}

		if appstate.ForcePrint && elements.grid.Model().NItems() == 0 {
			if appstate.IsDmenu {
				handleDmenuResult(elements.input.Text())
			}

			closeAfterActivation(isShift, false)
			return true
		}

		activateItem(isShift, isShift, isAlt)
		return true
	case gdk.KEY_Tab:
		if elements.typeahead.Text() != "" {
			tahAcceptedIdentifier = tahSuggestionIdentifier
			tahSuggestionIdentifier = ""

			elements.input.SetText(elements.typeahead.Text())
			elements.input.SetPosition(-1)

			return true
		} else {
			selectNext()

			return true
		}
	case gdk.KEY_Down:
		if layout.Window.Box.Scroll.List.Grid {
			return false
		}

		selectNext()
		return true
	case gdk.KEY_Up:
		if layout.Window.Box.Scroll.List.Grid {
			return false
		}

		if common.selection.Selected() == 0 || common.items.NItems() == 0 {
			if len(toUse) != 1 {
				selectPrev()
				return true
			}

			if len(explicits) != 0 && len(explicits) != 1 {
				selectPrev()
				return true
			}

			var inputhstry []history.InputHistoryItem

			if len(explicits) == 1 {
				inputhstry = history.GetInputHistory(explicits[0].General().Name)
			} else {
				inputhstry = history.GetInputHistory(toUse[0].General().Name)
			}

			if len(inputhstry) > 0 {
				historyIndex++

				if historyIndex == len(inputhstry) {
					historyIndex = 0
				}

				glib.IdleAdd(func() {
					elements.input.SetText(inputhstry[historyIndex].Term)
					elements.input.SetPosition(-1)
				})
			}
		} else {
			selectPrev()
			return true
		}
	case gdk.KEY_ISO_Left_Tab:
		selectPrev()
		return true
	default:
		if (cfg.List.KeyboardScrollStyle == "vim" && val == gdk.KEY_j) || val == gdk.KEY_n {
			if cfg.ActivationMode.Disabled {
				if modifier == gdk.ControlMask {
					selectNext()
					return true
				}
			}
		} else if (cfg.List.KeyboardScrollStyle == "vim" && val == gdk.KEY_k) || val == gdk.KEY_p {
			if cfg.ActivationMode.Disabled {
				if modifier == gdk.ControlMask {
					selectPrev()
					return true
				}
			}
		}

		if !cfg.ActivationMode.Disabled && activationEnabled {
			uc := gdk.KeyvalToUnicode(gdk.KeyvalToLower(val))

			if uc != 0 {
				index := slices.Index(appstate.UsedLabels, string(uc))

				if index != -1 {
					isAmShift := modifier == (gdk.ShiftMask | amModifier)

					selectActivationMode(isAmShift, false, uint(index))
					return true
				} else {
					return false
				}
			}
		}

		elements.input.GrabFocus()
		return false
	}

	return false
}

var historyIndex = 0

func activateItem(keepOpen, selectNext, alt bool) {
	if elements.grid.Model().NItems() == 0 {
		return
	}

	entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))

	executeEvent(config.EventActivate, entry.Label)

	if !keepOpen && entry.Sub != "Walker" && entry.Sub != "switcher" && cfg.IsService && entry.SpecialFunc == nil {
		go quit(true)
	}

	module := findModule(entry.Module, toUse, explicits)

	if entry.SpecialFunc != nil {
		args := []interface{}{}
		args = append(args, entry.SpecialFuncArgs...)
		args = append(args, elements.input.Text())

		if module.General().Name == cfg.Builtins.AI.Name {
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

	if cfg.Terminal != "" {
		if entry.Terminal || forceTerminal {
			toRun = fmt.Sprintf("%s --title %s -e %s", cfg.Terminal, "WalkerRunner", toRun)
		}
	} else {
		log.Println("terminal is not set")
		return
	}

	cmd := exec.Command("sh", "-c", wrapWithUWSM(toRun))

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

	identifier := entry.Identifier()

	if entry.History {
		hstry.Save(identifier, strings.TrimSpace(elements.input.Text()))
	}

	if module != nil && (module.General().History || module.General().Typeahead) {
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

			common.items.Splice(0, int(common.items.NItems()))
			elements.input.SetObjectProperty("placeholder-text", m.General().Placeholder)
			setupSingleModule()

			if val, ok := layouts[singleModule.General().Name]; ok {
				glib.IdleAdd(func() {
					layout = val
					setupLayout(singleModule.General().Theme, singleModule.General().ThemeBase)
				})
			}

			if elements.input.Text() != "" {
				elements.input.SetText("")
			} else {
				process()
			}

			elements.input.GrabFocus()
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
		fmt.Print(result)
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
	if !cfg.IsService && !keepOpen {
		exit(true)
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

var cancel context.CancelFunc

func process() {
	mouseX = 0
	mouseY = 0
	elements.grid.SetCanTarget(false)

	if isAi {
		return
	}

	if cfg.List.Placeholder != "" {
		elements.listPlaceholder.SetVisible(false)
	}

	if cancel != nil {
		cancel()
	}

	elements.typeahead.SetText("")

	text := strings.TrimSpace(elements.input.Text())

	if text == "" && cfg.List.ShowInitialEntries && len(explicits) == 0 && !appstate.IsDmenu {
		setInitials()
		return
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	if (elements.input.Text() != "" || appstate.IsDmenu) || (len(explicits) > 0 && cfg.List.ShowInitialEntries) {
		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(true)
		}

		go processAsync(ctx, text)
	} else {
		common.items.Splice(0, int(common.items.NItems()))

		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}
	}
}

var timeoutTimer *time.Timer

func timeoutReset() {
	if cfg.Timeout > 0 {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}

		timeoutTimer = time.AfterFunc(time.Duration(cfg.Timeout)*time.Second, func() {
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
	if cfg.Timeout > 0 {
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

func processAsync(ctx context.Context, text string) {
	entries := []util.Entry{}

	defer func() {
		cancel()

		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}
	}()

	hasExplicit := len(explicits) > 0

	p := toUse

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

			text = strings.TrimPrefix(text, prefix)
		}

		if !p[k].General().IsSetup {
			p[k].SetupData(cfg, ctx)
		}

		go func(ctx context.Context, wg *sync.WaitGroup, text string, w modules.Workable) {
			defer wg.Done()

			mCfg := w.General()

			if len(text) < mCfg.MinChars {
				return
			}

			e := w.Entries(ctx, text)

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
							e[k].ScoreFinal = fuzzyScore(e[k], toMatch)
						} else {
							e[k].ScoreFinal = 1000
						}
					case util.Fuzzy:
						e[k].ScoreFinal = fuzzyScore(e[k], toMatch)
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
					if e[k].ScoreFinal > float64(cfg.List.VisibilityThreshold) {
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
		}(ctx, &wg, text, p[k])
	}

	wg.Wait()

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

	if len(entries) > cfg.List.MaxEntries {
		entries = entries[:cfg.List.MaxEntries]
	}

	if appstate.IsDebug {
		for _, v := range entries {
			fmt.Printf("Entries == label: %s sub: %s score: %f\n", v.Label, v.Sub, v.ScoreFinal)
		}
	}

	glib.IdleAdd(func() {
		if cfg.List.Placeholder != "" && len(entries) == 0 {
			elements.listPlaceholder.SetVisible(true)
		}

		common.items.Splice(0, int(common.items.NItems()), entries...)

		if cfg.IgnoreMouse && !elements.grid.CanTarget() {
			for _, v := range entries {
				if v.DragDrop {
					elements.grid.SetCanTarget(true)
					break
				}
			}
		} else if cfg.IgnoreMouse {
			elements.grid.SetCanTarget(false)
		}
	})

	tahAcceptedIdentifier = ""

	glib.IdleAdd(func() {
		if !layout.Window.Box.Search.Spinner.Hide {
			elements.spinner.SetVisible(false)
		}

		if common.items.NItems() == 1 && appstate.AutoSelect {
			common.selection.SelectItem(0, true)
			activateItem(false, false, false)
		}
	})
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
		proc.SetupData(cfg, nil)
	}

	e := proc.Entries(nil, "")

	for _, entry := range e {
		entry.Module = proc.General().Name

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

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return
	}

	sortEntries(entries, false)

	common.items.Splice(0, int(common.items.NItems()), entries...)
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
				layout = config.GetLayout(cfg.Theme, cfg.ThemeBase)
				setupLayout(cfg.Theme, cfg.ThemeBase)
			})
		}

		resetSingleModule()
	}

	appstate.IsRunning = false
	appstate.IsSingle = false
	appstate.AutoSelect = appstate.AutoSelectOld

	historyIndex = 0

	for _, v := range toUse {
		go v.Cleanup()
	}

	disableAM()

	appstate.ExplicitModules = []string{}
	appstate.ExplicitPlaceholder = ""
	appstate.ExplicitTheme = ""
	appstate.IsDmenu = false
	thumbnails = make(map[string][]byte)

	explicits = []modules.Workable{}

	if appstate.IsService {
		appstate.LastQuery = elements.input.Text()
	}

	glib.IdleAdd(func() {
		if layout != nil {
			if !layout.Window.Box.Search.Spinner.Hide {
				elements.spinner.SetVisible(false)
			}
		}

		if !cfg.Search.ResumeLastQuery {
			elements.input.SetText("")
			elements.input.SetObjectProperty("placeholder-text", cfg.Search.Placeholder)
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

func exit(ignoreEvent bool) {
	if !ignoreEvent {
		executeEvent(config.EventExit, "")
	}

	elements.appwin.Close()
	os.Exit(0)
}

const modifier = 0.10

func fuzzyScore(entry util.Entry, text string) float64 {
	textLength := len(text)

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
		matchables = []string{entry.Label, entry.Sub, entry.Searchable}
		matchables = append(matchables, entry.Categories...)
	} else {
		matchables = []string{entry.Label}
	}

	multiplier := 0

	for k, t := range matchables {

		if t == "" {
			continue
		}

		var score float64

		if strings.HasPrefix(text, "'") {
			score = util.ExactScore(text, t)
		} else {
			score = util.FuzzyScore(text, t)
		}

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

	score := float64(usageScore)*tm + float64(entry.ScoreFuzzy)/tm

	if appstate.IsDebug {
		fmt.Printf("Matching == label: %s sub: %s searchable: %s categories: %s score: %f usage: %d fuzzy: %f m: %f\n", entry.Label, entry.Sub, entry.Searchable, entry.Categories, score, usageScore, entry.ScoreFuzzy, m)
	}

	return score
}

func wrapWithUWSM(text string) string {
	if !cfg.UseUWSM {
		return text
	}

	return fmt.Sprintf("uwsm app -- %s", text)
}
