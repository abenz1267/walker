package ui

import (
	"bytes"
	"context"
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
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var (
	keys              map[uint]uint
	activationEnabled bool
	amKey             uint
	amModifier        gdk.ModifierType
	commands          map[string]func()
	tah               []string
)

func setupCommands() {
	commands = make(map[string]func())
	commands["reloadconfig"] = func() {
		cfg = config.Get()
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
	commands["cleartypeaheadcache"] = func() {
		os.Remove(filepath.Join(util.CacheDir(), "typeahead.gob"))
		tah = []string{}
	}
}

func setupModules() {
	util.FromGob(filepath.Join(util.CacheDir(), "typeahead.gob"), &tah)

	internals := []modules.Workable{
		modules.Applications{},
		modules.Runner{ShellConfig: cfg.ShellConfig},
		modules.Websearch{},
		modules.Commands{},
		modules.Hyprland{},
		modules.SSH{},
		modules.Finder{},
		appstate.Clipboard,
	}

	for _, v := range cfg.External {
		e := modules.External{
			ModuleName: v.Name,
		}

		internals = append(internals, e)
	}

	clear(procs)

	procs = make(map[string][]modules.Workable)

	for _, v := range internals {
		if v == nil {
			continue
		}

		if v.Name() == "switcher" {
			continue
		}

		w := v.Setup(cfg)
		if w != nil {
			procs[w.Prefix()] = append(procs[w.Prefix()], w)
		}
	}

	// setup switcher individually
	switcher := modules.Switcher{Procs: procs}

	s := switcher.Setup(cfg)
	if s != nil {
		procs[s.Prefix()] = append(procs[s.Prefix()], s)
	}

	clear(ui.prefixClasses)

	for _, v := range procs {
		for _, vv := range v {
			ui.prefixClasses[vv.Prefix()] = append(ui.prefixClasses[vv.Prefix()], vv.Name())
		}
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
	ui.search.Connect("activate", func() { activateItem(false, false) })

	listkc := gtk.NewEventControllerKey()
	listkc.ConnectKeyPressed(handleListKeysPressed)

	ui.list.AddController(listkc)

	amKey = gdk.KEY_Control_L
	amModifier = gdk.ControlMask

	if cfg.ActivationMode.UseAlt {
		amKey = gdk.KEY_Alt_L
		amModifier = gdk.AltMask
	}
}

func selectNext() {
	items := ui.selection.NItems()

	if items == 0 {
		return
	}

	current := ui.selection.Selected()

	if current+1 < items {
		ui.selection.SetSelected(current + 1)
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
	}
}

func selectActivationMode(val uint, keepOpen bool) {
	if k, ok := specialLabels[val]; ok {
		ui.selection.SetSelected(k)
	} else {
		ui.selection.SetSelected(keys[val])
	}

	if keepOpen {
		activateItem(true, false)
		return
	}

	activateItem(false, false)
}

func disabledAM() {
	if !cfg.ActivationMode.Disabled && activationEnabled {
		activationEnabled = false

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

func toggleForceTerminal() {
	forceTerminal = !forceTerminal

	c := ui.appwin.CSSClasses()

	if !forceTerminal {
		for k, v := range c {
			if v == "forceterminal" {
				c = slices.Delete(c, k, k+1)
			}
		}
	} else {
		c = append(c, "forceterminal")
	}

	ui.appwin.SetCSSClasses(c)
}

func handleListKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	if !cfg.ActivationMode.Disabled && ui.selection.NItems() != 0 {
		if val == amKey {
			c := ui.appwin.CSSClasses()
			c = append(c, "activation")
			ui.appwin.SetCSSClasses(c)
			ui.list.GrabFocus()
			activationEnabled = true

			return true
		}
	}

	switch val {
	case gdk.KEY_Super_L:
		toggleForceTerminal()
	case gdk.KEY_Shift_L:
		return true
	case gdk.KEY_Return:
		if modifier == gdk.ShiftMask {
			activateItem(true, true)
		} else {
			return false
		}
	case gdk.KEY_Escape:
		if activationEnabled {
			disabledAM()
		} else {
			if singleProc != nil {
				disableSingleProc()
			} else {
				quit()
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
		if _, ok := specialLabels[val]; ok {
			if modifier == gdk.ShiftMask {
				selectActivationMode(val, true)
			} else {
				selectActivationMode(val, false)
			}
		} else {
			ui.search.GrabFocus()
			return false
		}
	}

	return true
}

var forceTerminal bool

func handleSearchKeysPressed(val uint, code uint, modifier gdk.ModifierType) bool {
	if !cfg.ActivationMode.Disabled && ui.selection.NItems() != 0 && !cfg.ActivationMode.UseFKeys {
		if val == amKey {
			c := ui.appwin.CSSClasses()
			c = append(c, "activation")
			ui.appwin.SetCSSClasses(c)
			ui.list.GrabFocus()
			activationEnabled = true

			return true
		}
	}

	switch val {
	case gdk.KEY_Super_L:
		toggleForceTerminal()
	case gdk.KEY_Return:
		if modifier == gdk.ShiftMask {
			activateItem(true, true)
		}
	case gdk.KEY_Escape:
		if singleProc != nil {
			disableSingleProc()
		} else {
			quit()
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
		selectPrev()
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

func disableSingleProc() {
	if singleProc != nil {
		singleProc = nil
		ui.search.SetObjectProperty("placeholder-text", cfg.Placeholder)

		if ui.search.Text() != "" {
			ui.search.SetText("")
		} else {
			if cfg.ShowInitialEntries {
				process()
			}
		}
	}
}

func activateItem(keepOpen, selectNext bool) {
	if ui.list.Model().NItems() == 0 {
		return
	}

	entry := ui.items.Item(int(ui.selection.Selected()))

	if entry.Sub == "Walker" {
		commands[entry.Exec]()
		closeAfterActivation(keepOpen, selectNext)
		return
	}

	if entry.Sub == "switcher" {
		for _, v := range procs {
			for _, w := range v {
				if w.Name() == entry.Label {
					singleProc = w
					ui.items.Splice(0, ui.items.NItems())
					ui.search.SetObjectProperty("placeholder-text", w.Name())
					ui.search.SetText("")
					ui.search.GrabFocus()
					return
				}
			}
		}
	}

	f := strings.Fields(entry.Exec)

	if len(entry.RawExec) > 0 {
		f = entry.RawExec
	}

	if len(f) == 0 {
		return
	}

	if cfg.Terminal != "" {
		if entry.Terminal || forceTerminal {
			f = append([]string{cfg.Terminal, "-e"}, f...)
		}
	} else {
		log.Println("terminal is not set")
		return
	}

	cmd := exec.Command(f[0])
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:     true,
		Foreground: false,
	}

	if entry.Piped.Content != "" {
		if entry.Piped.Type == "file" {
			b, err := os.ReadFile(entry.Piped.Content)
			if err != nil {
				log.Panic(err)
			}

			r := bytes.NewReader(b)
			cmd.Stdin = r
		}
	}

	if len(f) > 1 {
		cmd = exec.Command(f[0], f[1:]...)
	}

	if entry.History {
		hstry.Save(entry.Identifier(), strings.TrimSpace(ui.search.Text()))
	}

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}

	closeAfterActivation(keepOpen, selectNext)
}

func closeAfterActivation(keepOpen, sn bool) {
	if cfg.EnableTypeahead {
		tah = append([]string{ui.search.Text()}, tah...)
		util.ToGob(&tah, filepath.Join(util.CacheDir(), "typeahead.gob"))
	}

	if !keepOpen {
		quit()
		return
	}

	if !activationEnabled && !sn {
		ui.search.SetText("")
	}

	if sn {
		selectNext()
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var cancel context.CancelFunc

func process() {
	ui.spinner.SetCSSClasses([]string{"visible"})

	if cfg.EnableTypeahead {
		ui.typeahead.SetText("")

		if strings.TrimSpace(ui.search.Text()) != "" {
			for _, v := range tah {
				if strings.HasPrefix(v, ui.search.Text()) {
					ui.typeahead.SetText(v)
				}
			}
		}
	}

	if !appstate.IsRunning {
		return
	}

	if cancel != nil {
		cancel()
	}

	text := strings.TrimSpace(ui.search.Text())
	if text == "" && cfg.ShowInitialEntries && singleProc == nil {
		setInitials()
		return
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	if ui.search.Text() != "" || singleProc != nil {
		go processAsync(ctx)
	} else {
		ui.items.Splice(0, ui.items.NItems())
		ui.spinner.SetCSSClasses([]string{})
	}
}

var handlerPool = sync.Pool{
	New: func() any {
		return new(Handler)
	},
}

func processAsync(ctx context.Context) {
	handler := handlerPool.Get().(*Handler)
	defer func() {
		handlerPool.Put(handler)
		ui.spinner.SetCSSClasses([]string{})
		cancel()
	}()

	handler.ctx = ctx
	handler.entries = []modules.Entry{}

	text := strings.TrimSpace(ui.search.Text())

	glib.IdleAdd(func() {
		ui.items.Splice(0, ui.items.NItems())
		ui.appwin.SetCSSClasses([]string{})
	})

	prefix := text

	if len(prefix) > 1 {
		prefix = prefix[0:1]
	}

	p := []modules.Workable{}

	if singleProc == nil {
		hasPrefix := true

		var ok bool
		p, ok = procs[prefix]
		if !ok {
			p = procs[""]
			hasPrefix = false
		}

		if hasPrefix {
			glib.IdleAdd(func() {
				ui.appwin.SetCSSClasses(ui.prefixClasses[prefix])
			})

			text = strings.TrimPrefix(text, prefix)
		}
	} else {
		p = []modules.Workable{singleProc}

		if singleProc != nil {
			glib.IdleAdd(func() {
				ui.appwin.SetCSSClasses(ui.prefixClasses[singleProc.Prefix()])
			})
		}
	}

	handler.receiver = make(chan []modules.Entry)
	go handler.handle()

	var wg sync.WaitGroup
	wg.Add(len(p))

	for _, proc := range p {
		if proc.SwitcherExclusive() {
			if singleProc == nil || singleProc.Name() != proc.Name() {
				wg.Done()
				handler.receiver <- []modules.Entry{}
				continue
			}
		}

		go func(ctx context.Context, wg *sync.WaitGroup, text string, w modules.Workable) {
			defer wg.Done()

			e := w.Entries(ctx, text)

			toPush := []modules.Entry{}

			for k := range e {
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
					case modules.Fuzzy:
						e[k].ScoreFinal = fuzzyScore(e[k], toMatch)
					case modules.AlwaysTop:
						if e[k].ScoreFinal == 0 {
							e[k].ScoreFinal = 1000
						}
					case modules.AlwaysBottom:
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
		}(ctx, &wg, text, proc)
	}

	wg.Wait()
}

func setInitials() {
	entrySlice := []modules.Entry{}

	for _, v := range procs {
		for _, proc := range v {
			if proc.Name() != "applications" {
				continue
			}

			e := proc.Entries(nil, "")

			for _, entry := range e {
				for _, v := range hstry {
					if val, ok := v[entry.Identifier()]; ok {
						entry.Used = val.Used
						entry.DaysSinceUsed = val.DaysSinceUsed
						entry.LastUsed = val.LastUsed
					}
				}

				entry.ScoreFinal = float64(usageModifier(entry))

				entrySlice = append(entrySlice, entry)
			}
		}
	}

	if len(entrySlice) == 0 {
		return
	}

	sortEntries(entrySlice)

	ui.items.Splice(0, ui.items.NItems(), entrySlice...)

	ui.selection.SetSelected(0)

	ui.spinner.SetCSSClasses([]string{})
}

func usageModifier(item modules.Entry) int {
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
	if appstate.IsService {
		if !cfg.ActivationMode.Disabled && activationEnabled {
			activationEnabled = false

			c := ui.appwin.CSSClasses()

			for k, v := range c {
				if v == "activation" {
					c = slices.Delete(c, k, k+1)
				}
			}

			ui.appwin.SetCSSClasses(c)
			ui.search.GrabFocus()
		}

		singleProc = nil

		ui.appwin.SetVisible(false)

		go func() {
			glib.IdleAdd(func() {
				ui.search.SetText("")
				ui.search.SetObjectProperty("placeholder-text", cfg.Placeholder)

				appstate.IsRunning = false
				appstate.IsMeasured = false
			})
		}()

		ui.app.Hold()
	} else {
		ui.appwin.Close()
	}
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

func fuzzyScore(entry modules.Entry, text string) float64 {
	textLength := len(text)

	if textLength == 0 {
		return 1
	}

	matchables := []string{entry.Label, entry.Sub, entry.Searchable}
	matchables = append(matchables, entry.Categories...)

	multiplier := 0

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

	if p, ok := hstry[text]; ok {
		if val, ok := p[entry.Identifier()]; ok {
			entry.Used = val.Used
			entry.DaysSinceUsed = val.DaysSinceUsed
			entry.LastUsed = val.LastUsed
		}
	}

	usageScore := usageModifier(entry)

	if textLength == 0 {
		textLength = 1
	}

	tm := 1.0 / float64(textLength)

	return float64(usageScore)*tm + float64(entry.ScoreFuzzy)/tm
}
