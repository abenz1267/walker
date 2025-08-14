package setup

import (
	"log/slog"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

const (
	ActionClose          = "%CLOSE%"
	ActionSelectNext     = "%NEXT%"
	ActionSelectPrevious = "%PREVIOUS%"

	AfterClose   = "%CLOSE%"
	AfterNothing = "%NOTHING%"
	AfterReload  = "%RELOAD%"

	ActionCalcCopy   = "copy"
	ActionCalcDelete = "delete"
	ActionCalcSave   = "save"

	ActionClipboardCopy   = "copy"
	ActionClipboardDelete = "remove"

	ActionDesktopapplicationsStart = ""

	ActionFilesCopy     = "copyfile"
	ActionFilesCopyPath = "copypath"
	ActionFilesOpen     = "open"
	ActionFilesOpenDir  = "opendir"

	ActionRunnerStart = ""

	ActionSymbolsCopy = ""
)

type Action struct {
	action string
	after  string
}

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
		"kpenter":   int(gdk.KEY_KP_Enter),
		"enter":     int(gdk.KEY_Return),
		"down":      int(gdk.KEY_Down),
		"up":        int(gdk.KEY_Up),
		"left":      int(gdk.KEY_Left),
		"right":     int(gdk.KEY_Right),
	}
)

var (
	Binds         = make(map[int]map[gdk.ModifierType]Action)
	ProviderBinds = make(map[string]map[int]map[gdk.ModifierType]Action)
)

func setupBinds() {
	parseBind(config.LoadedConfig.Keybinds.Close, ActionClose, AfterClose, "")
	parseBind(config.LoadedConfig.Keybinds.Next, ActionSelectNext, AfterNothing, "")
	parseBind(config.LoadedConfig.Keybinds.Previous, ActionSelectPrevious, AfterNothing, "")

	parseBind(config.LoadedConfig.Providers.Clipboard.Copy, ActionClipboardCopy, AfterClose, "clipboard")
	parseBind(config.LoadedConfig.Providers.Clipboard.Delete, ActionClipboardDelete, AfterReload, "clipboard")

	parseBind(config.LoadedConfig.Providers.Calc.Copy, ActionCalcCopy, AfterClose, "calc")
	parseBind(config.LoadedConfig.Providers.Calc.Save, ActionCalcSave, AfterReload, "calc")
	parseBind(config.LoadedConfig.Providers.Calc.Delete, ActionCalcDelete, AfterReload, "calc")

	parseBind(config.LoadedConfig.Providers.DesktopApplications.Start, ActionDesktopapplicationsStart, AfterClose, "desktopapplications")

	parseBind(config.LoadedConfig.Providers.Files.CopyFile, ActionFilesCopy, AfterClose, "files")
	parseBind(config.LoadedConfig.Providers.Files.CopyPath, ActionFilesCopyPath, AfterClose, "files")
	parseBind(config.LoadedConfig.Providers.Files.Open, ActionFilesOpen, AfterClose, "files")
	parseBind(config.LoadedConfig.Providers.Files.OpenDir, ActionFilesOpenDir, AfterClose, "files")

	parseBind(config.LoadedConfig.Providers.Runner.Start, ActionRunnerStart, AfterClose, "runner")

	parseBind(config.LoadedConfig.Providers.Symbols.Copy, ActionSymbolsCopy, AfterClose, "symbols")
}

func validateBind(bind string) bool {
	fields := strings.Fields(bind)

	ok := true

	for _, v := range fields {
		if len(v) > 1 {
			_, existsMod := modifiers[v]
			_, existsSpecial := specialKeys[v]

			if !existsMod && !existsSpecial {
				slog.Error("keybinds", "invalid", bind, "key", v)
				ok = false
			}
		}
	}

	return ok
}

func parseBind(bind string, action string, after string, provider string) {
	if ok := validateBind(bind); !ok {
		return
	}

	fields := strings.Fields(bind)

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

	a := Action{
		action: action,
		after:  after,
	}

	if provider == "" {
		if _, exists := Binds[key]; !exists {
			Binds[key] = make(map[gdk.ModifierType]Action)
			Binds[key][modifier] = a
		} else {
			Binds[key][modifier] = a
		}
	} else {
		if _, exists := ProviderBinds[provider]; !exists {
			ProviderBinds[provider] = make(map[int]map[gdk.ModifierType]Action)
			ProviderBinds[provider][key] = make(map[gdk.ModifierType]Action)
			ProviderBinds[provider][key][modifier] = a
		} else {
			if _, exists := ProviderBinds[provider][key]; !exists {
				ProviderBinds[provider][key] = make(map[gdk.ModifierType]Action)
			}

			ProviderBinds[provider][key][modifier] = a
		}
	}
}
