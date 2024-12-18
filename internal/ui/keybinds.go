package ui

import (
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
)

type keybinds map[string]map[string]func()

var binds keybinds

func parseKeybinds() {
	binds = make(map[string]map[string]func())

	specialKeys := []string{"ctrl", "alt", "shift", "backspace", "tab"}
}

func (keybinds) toggleAM() {
	if cfg.ActivationMode.Disabled {
		return
	}

	if common.selection.NItems() != 0 {
		enableAM()
	}
}

func (keybinds) clipboardDeleteEntry() {
	if singleModule != nil && singleModule.General().Name == cfg.Builtins.Clipboard.Name {
		entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
		singleModule.(*clipboard.Clipboard).Delete(entry)
		debouncedProcess(process)
	}
}

func (keybinds) deleteFromHistory() {
	entry := gioutil.ObjectValue[util.Entry](common.items.Item(common.selection.Selected()))
	hstry.Delete(entry.Identifier())
}

func (keybinds) aiCopyLast() {
	if !isAi {
		return
	}

	ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.CopyLastResponse()
}

func (keybinds) aiExecuteLast() {
	if !isAi {
		return
	}

	ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.RunLastMessageInTerminal()
	quit(true)
}

func (keybinds) toggleExactMatch() {
	text := elements.input.Text()

	if strings.HasPrefix(text, "'") {
		elements.input.SetText(strings.TrimPrefix(text, "'"))
	} else {
		elements.input.SetText("'" + text)
	}

	elements.input.SetPosition(-1)
}

func (keybinds) resume() {
	if appstate.IsService {
		elements.input.SetText(appstate.LastQuery)
		elements.input.SetPosition(-1)
		elements.input.GrabFocus()
	}
}

func (keybinds) aiResume() {
	if !isAi {
		return
	}

	ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	ai.ResumeLastMessages()
}

func (keybinds) aiClearSession() {
	if !isAi {
		return
	}

	ai := findModule(cfg.Builtins.AI.Name, toUse, explicits).(*modules.AI)
	elements.input.SetText("")
	ai.ClearCurrent()
}

func (keybinds) activateFunctionKeys(val uint) {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(false, true, uint(index))
	}
}

func (keybinds) activateKeepOpenFunctionKeys(val uint) {
	index := slices.Index(fkeys, val)

	if index != -1 {
		selectActivationMode(true, true, uint(index))
	}
}

func (keybinds) quit() {
	if appstate.IsDmenu {
		handleDmenuResult("")
	}

	if cfg.IsService {
		quit(false)
	} else {
		exit(false)
	}
}

func (keybinds) activate() {
}

func (keybinds) activateKeepOpen() {
}
