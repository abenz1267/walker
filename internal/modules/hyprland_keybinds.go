package modules

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Bind struct {
	Modmask     int    `json:"modmask"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Dispatcher  string `json:"dispatcher"`
	Arg         string `json:"arg"`
	Submap      string `json:"submap"`
	Mouse       bool   `json:"mouse"`
}

type HyprlandKeybinds struct {
	config  config.HyprlandKeybinds
	entries []util.Entry
}

func (HyprlandKeybinds) Cleanup() {
}

func (h HyprlandKeybinds) Entries(term string) (_ []util.Entry) {
	return h.entries
}

func (h HyprlandKeybinds) General() (_ *config.GeneralModule) {
	return &h.config.GeneralModule
}

func (HyprlandKeybinds) Refresh() {
}

func (h *HyprlandKeybinds) Setup() (_ bool) {
	h.config = config.Cfg.Builtins.HyprlandKeybinds

	return true
}

func (h *HyprlandKeybinds) SetupData() {
	cmd := exec.Command("hyprctl", "-j", "binds")

	out, err := cmd.Output()
	if err != nil {
		slog.Error("error", "hyprland_keybinds", err)
		return
	}

	var binds []Bind

	err = json.Unmarshal(out, &binds)
	if err != nil {
		slog.Error("error", "hyprland_keybinds", err)
		return
	}

	var entries []util.Entry

	for _, v := range binds {
		label := v.Description

		if label == "" {
			label = fmt.Sprintf("%s %s", v.Dispatcher, v.Arg)
		}

		var sub string
		modmask := modMaskToString(v.Modmask)

		if modmask == "" {
			sub = fmt.Sprintf("%s", v.Key)
		} else {
			sub = fmt.Sprintf("%s+%s", modMaskToString(v.Modmask), v.Key)
		}

		if v.Submap != "" {
			sub = fmt.Sprintf("%s: %s", v.Submap, sub)
		}

		exec := fmt.Sprintf("hyprctl dispatch %s %s", v.Dispatcher, v.Arg)

		if v.Mouse {
			sub = fmt.Sprintf("%s (mouse)", sub)
			exec = ""
		}

		e := util.Entry{
			Label:            label,
			Exec:             exec,
			Sub:              sub,
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		}

		entries = append(entries, e)
	}

	h.entries = entries

	h.config.IsSetup = true
	h.config.HasInitialSetup = true
}

func modMaskToString(modmask int) string {
	var parts []string

	// Common modifier mask values
	const (
		ShiftMask   = 1 << 0 // 1
		LockMask    = 1 << 1 // 2
		ControlMask = 1 << 2 // 4
		Mod1Mask    = 1 << 3 // 8 (Alt)
		Mod2Mask    = 1 << 4 // 16
		Mod3Mask    = 1 << 5 // 32
		Mod4Mask    = 1 << 6 // 64 (Super/Windows key)
		Mod5Mask    = 1 << 7 // 128
	)

	if modmask&ShiftMask != 0 {
		parts = append(parts, "SHIFT")
	}
	if modmask&LockMask != 0 {
		parts = append(parts, "LOCK")
	}
	if modmask&ControlMask != 0 {
		parts = append(parts, "CONTROL")
	}
	if modmask&Mod1Mask != 0 {
		parts = append(parts, "ALT")
	}
	if modmask&Mod4Mask != 0 {
		parts = append(parts, "SUPER")
	}

	return strings.Join(parts, "+")
}
