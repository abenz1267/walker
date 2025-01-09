package ui

import (
	"log/slog"
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

type keybinds map[int]map[gdk.ModifierType][]func() bool

var (
	binds   keybinds
	aibinds keybinds
)

var (
	modifiersInt = map[string]int{
		"lctrl":     gdk.KEY_Control_L,
		"rctrl":     gdk.KEY_Control_R,
		"lalt":      gdk.KEY_Alt_L,
		"ralt":      gdk.KEY_Alt_R,
		"lshift":    gdk.KEY_Shift_L,
		"rshift":    gdk.KEY_Shift_R,
		"shiftlock": gdk.KEY_Shift_Lock,
	}
	modifiers = map[string]gdk.ModifierType{
		"ctrl":   gdk.ControlMask,
		"lctrl":  gdk.ControlMask,
		"rctrl":  gdk.ControlMask,
		"alt":    gdk.AltMask,
		"lalt":   gdk.AltMask,
		"ralt":   gdk.AltMask,
		"lshift": gdk.ShiftMask,
		"rshift": gdk.ShiftMask,
		"shift":  gdk.ShiftMask,
	}
	specialKeys = map[string]int{
		"backspace": int(gdk.KEY_BackSpace),
		"tab":       int(gdk.KEY_Tab),
		"esc":       int(gdk.KEY_Escape),
		"enter":     int(gdk.KEY_Return),
		"down":      int(gdk.KEY_Down),
		"up":        int(gdk.KEY_Up),
		"left":      int(gdk.KEY_Left),
		"right":     int(gdk.KEY_Right),
	}

	labelTrigger        = gdk.KEY_Alt_L
	keepOpenModifier    = gdk.ShiftMask
	labelModifier       = gdk.AltMask
	activateAltModifier = gdk.AltMask
)

func parseKeybinds() {
	binds = make(keybinds)
	aibinds = make(keybinds)

	for _, v := range config.Cfg.Keys.AcceptTypeahead {
		binds.validate(v)
		binds.bind(binds, v, acceptTypeahead)
	}

	for _, v := range config.Cfg.Keys.Close {
		binds.validate(v)
		binds.bind(binds, v, quitKeybind)
	}

	for _, v := range config.Cfg.Keys.Next {
		binds.validate(v)
		binds.bind(binds, v, selectNext)
	}

	for _, v := range config.Cfg.Keys.Prev {
		binds.validate(v)
		binds.bind(binds, v, selectPrev)
	}

	for _, v := range config.Cfg.Keys.RemoveFromHistory {
		binds.validate(v)
		binds.bind(binds, v, deleteFromHistory)
	}

	for _, v := range config.Cfg.Keys.ResumeQuery {
		binds.validate(v)
		binds.bind(binds, v, resume)
	}

	for _, v := range config.Cfg.Keys.ToggleExactSearch {
		binds.validate(v)
		binds.bind(binds, v, toggleExactMatch)
	}

	binds.bind(binds, "enter", func() bool { return activate(false, false) })
	binds.bind(binds, strings.Join([]string{config.Cfg.Keys.ActivationModifiers.KeepOpen, "enter"}, " "), func() bool { return activate(true, false) })
	binds.bind(binds, strings.Join([]string{config.Cfg.Keys.ActivationModifiers.Alternate, "enter"}, " "), func() bool { return activate(false, true) })

	keepOpenModifier = modifiers[config.Cfg.Keys.ActivationModifiers.KeepOpen]
	activateAltModifier = modifiers[config.Cfg.Keys.ActivationModifiers.Alternate]

	binds.validateTriggerLabels(config.Cfg.Keys.TriggerLabels)
	labelTrigger = modifiersInt[strings.Fields(config.Cfg.Keys.TriggerLabels)[0]]
	labelModifier = modifiers[strings.Fields(config.Cfg.Keys.TriggerLabels)[0]]

	for _, v := range config.Cfg.Keys.Ai.ClearSession {
		binds.validate(v)
		binds.bind(aibinds, v, aiClearSession)
	}

	for _, v := range config.Cfg.Keys.Ai.CopyLastResponse {
		binds.validate(v)
		binds.bind(aibinds, v, aiCopyLast)
	}

	for _, v := range config.Cfg.Keys.Ai.ResumeSession {
		binds.validate(v)
		binds.bind(aibinds, v, aiResume)
	}

	for _, v := range config.Cfg.Keys.Ai.RunLastResponse {
		binds.validate(v)
		binds.bind(aibinds, v, aiExecuteLast)
	}
}

func (keybinds) bind(binds keybinds, val string, fn func() bool) {
	fields := strings.Fields(val)

	m := []gdk.ModifierType{}

	key := 0

	for _, v := range fields {
		if len(v) > 1 {
			if val, exists := modifiers[v]; exists {
				m = append(m, val)
			}

			if val, exists := specialKeys[v]; exists {
				key = val
			}
		} else {
			key = int(v[0])
		}
	}

	modifier := gdk.NoModifierMask

	switch len(m) {
	case 1:
		modifier = m[0]
	case 2:
		modifier = m[0] | m[1]
	case 3:
		modifier = m[0] | m[1] | m[2]
	}

	_, ok := binds[key]
	if !ok {
		binds[key] = make(map[gdk.ModifierType][]func() bool)
	}

	binds[key][modifier] = append(binds[key][modifier], fn)
}

func (keybinds) execute(key int, modifier gdk.ModifierType) bool {
	if isAi {
		fns, ok := aibinds[key][modifier]
		if ok {
			for _, fn := range fns {
				if fn() {
					return true
				}
			}
		}
	}

	if fns, ok := binds[key][modifier]; ok {
		for _, fn := range fns {
			if fn() {
				return true
			}
		}
	}

	return false
}

func (keybinds) validate(bind string) {
	fields := strings.Fields(bind)

	for _, v := range fields {
		if len(v) > 1 {
			_, existsMod := modifiers[v]
			_, existsSpecial := specialKeys[v]

			if !existsMod && !existsSpecial {
				slog.Error("keybinds", "bind", bind, "key", v)
			}
		}
	}
}

func (keybinds) validateTriggerLabels(bind string) {
	fields := strings.Fields(bind)
	_, exists := modifiersInt[fields[0]]

	if !exists || len(fields[0]) == 1 {
		slog.Error("keybinds", "invalid trigger_label keybind", bind)
	}
}

func toggleAM() bool {
	if config.Cfg.ActivationMode.Disabled {
		return false
	}

	if common.selection.NItems() != 0 {
		enableAM()

		return true
	}

	return false
}

func deleteFromHistory() bool {
	if singleModule != nil && singleModule.General().Name == config.Cfg.Builtins.Clipboard.Name {
		entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
		singleModule.(*clipboard.Clipboard).Delete(entry)
		debouncedProcess(process)
		return true
	}

	entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
	hstry.Delete(entry.Identifier())

	return true
}

func aiCopyLast() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.CopyLastResponse()

	return true
}

func aiExecuteLast() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.RunLastMessageInTerminal()
	quit(true)

	return true
}

func toggleExactMatch() bool {
	text := elements.input.Text()

	if strings.HasPrefix(text, "'") {
		elements.input.SetText(strings.TrimPrefix(text, "'"))
	} else {
		elements.input.SetText("'" + text)
	}

	elements.input.SetPosition(-1)

	return true
}

func resume() bool {
	if appstate.LastQuery != "" {
		elements.input.SetText(appstate.LastQuery)
		elements.input.SetPosition(-1)
		elements.input.GrabFocus()
	}

	return true
}

func aiResume() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.ResumeLastMessages()

	return true
}

func aiClearSession() bool {
	if !isAi {
		return false
	}

	ai := findModule(config.Cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	elements.input.SetText("")
	ai.ClearCurrent()

	return true
}

func activateFunctionKeys(val uint) bool {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(false, true, uint(index))
		return true
	}

	return false
}

func activateKeepOpenFunctionKeys(val uint) bool {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(true, true, uint(index))
		return true
	}

	return false
}

func quitKeybind() bool {
	if appstate.IsDmenu {
		handleDmenuResult("CNCLD")
	}

	if config.Cfg.IsService {
		quit(false)
		return true
	} else {
		exit(false, true)
		return true
	}
}

func acceptTypeahead() bool {
	if elements.typeahead.Text() != "" {
		tahAcceptedIdentifier = tahSuggestionIdentifier
		tahSuggestionIdentifier = ""

		elements.input.SetText(elements.typeahead.Text())
		elements.input.SetPosition(-1)

		return true
	}

	return false
}

func activate(keepOpen bool, isAlt bool) bool {
	if appstate.ForcePrint && elements.grid.Model().NItems() == 0 {
		if appstate.IsDmenu {
			handleDmenuResult(elements.input.Text())
		}

		closeAfterActivation(keepOpen, false)
		return true
	}

	activateItem(keepOpen, isAlt)
	return true
}
