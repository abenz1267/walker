package modules

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

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
	if strings.HasPrefix(h.config.Path, "~") {
		home, _ := os.UserHomeDir()
		h.config.Path = strings.ReplaceAll(h.config.Path, "~", home)
	}

	file, err := os.Open(h.config.Path)
	if err != nil {
		slog.Error("hyprland_keybinds", "error", err)
	}

	scanner := bufio.NewScanner(file)

	variables := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "$") {
			parts := strings.Split(line, "=")

			if len(parts) < 2 {
				continue
			}

			variables[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}

		if strings.HasPrefix(line, "bind") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 2 {
				continue
			}

			values := strings.Split(parts[1], ",")

			for k, v := range values {
				values[k] = strings.TrimSpace(v)
			}

			modifiers := strings.Split(values[0], " ")

			for _, v := range modifiers {
				if val, ok := variables[v]; ok {
					modifiers[0] = val
				}
			}

			values[0] = strings.Join(modifiers, " ")

			var label string

			if values[0] != "" {
				label = fmt.Sprintf("%s+%s", values[0], values[1])
			} else {
				label = values[1]
			}

			h.entries = append(h.entries, util.Entry{
				Label:            strings.Join(values[2:], " "),
				Sub:              label,
				Class:            "hyprland_keybinds",
				Exec:             fmt.Sprintf("hyprctl dispatch %s", strings.Join(values[2:], " ")),
				Matching:         util.Fuzzy,
				RecalculateScore: true,
			})
		}
	}

	h.config.IsSetup = true
	h.config.HasInitialSetup = true
}
