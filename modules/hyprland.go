package modules

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
)

type Hyprland struct {
	general config.GeneralModule
	windows map[string]uint
}

func (h Hyprland) Placeholder() string {
	if h.general.Placeholder == "" {
		return "hyprland"
	}

	return h.general.Placeholder
}

func (h Hyprland) SwitcherOnly() bool {
	return h.general.SwitcherOnly
}

func (h Hyprland) Setup(cfg *config.Config) Workable {
	b := &Hyprland{}

	pth, _ := exec.LookPath("hyprctl")
	if pth == "" {
		log.Println("Hyprland not found. Disabling module.")
		return nil
	}

	b.general.Prefix = cfg.Builtins.Hyprland.Prefix
	b.general.SwitcherOnly = cfg.Builtins.Hyprland.SwitcherOnly
	b.general.SpecialLabel = cfg.Builtins.Hyprland.SpecialLabel

	b.windows = make(map[string]uint)

	if cfg.IsService && cfg.Builtins.Hyprland.ContextAwareHistory {
		go b.monitorWindows()
	}

	return b
}

func (h Hyprland) Refresh() {}

func (h *Hyprland) monitorWindows() {
	for {
		clear(h.windows)

		cmd := exec.Command("hyprctl", "clients")
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(err)
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(out))

		for scanner.Scan() {
			text := scanner.Text()

			text = strings.TrimSpace(text)

			if strings.HasPrefix(text, "initialClass:") {
				text = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(text, "initialClass:")))
				h.windows[text] = h.windows[text] + 1
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (h *Hyprland) GetWindowAmount(class string) uint {
	if val, ok := h.windows[class]; ok {
		return val
	}

	return 0
}

func (Hyprland) Name() string {
	return "hyprland"
}

func (h Hyprland) Prefix() string {
	return h.general.Prefix
}

type window struct {
	title        string
	pid          string
	workspace    string
	initialTitle string
}

func (Hyprland) Entries(ctx context.Context, term string) []Entry {
	cmd := exec.Command("hyprctl", "clients")

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return nil
	}

	entries := []Entry{}

	scanner := bufio.NewScanner(bytes.NewReader(out))

	windows := []window{}

	for scanner.Scan() {
		text := scanner.Text()

		text = strings.TrimSpace(text)

		if strings.HasPrefix(text, "Window") {
			n := window{}

			windows = append(windows, n)
		}

		if strings.HasPrefix(text, "title:") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "title:"))
			windows[len(windows)-1].title = text
		}

		if strings.HasPrefix(text, "initialTitle:") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "initialTitle:"))
			windows[len(windows)-1].initialTitle = text
		}

		if strings.HasPrefix(text, "workspace:") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "workspace:"))
			fields := strings.Fields(text)
			windows[len(windows)-1].workspace = fields[0]
		}

		if strings.HasPrefix(text, "pid") {
			text = strings.TrimSpace(strings.TrimPrefix(text, "pid:"))
			windows[len(windows)-1].pid = text
		}
	}

	for _, v := range windows {
		if v.pid == "-1" {
			continue
		}

		n := Entry{
			Label:      v.title,
			Sub:        fmt.Sprintf("Hyprland (Workspace %s)", v.workspace),
			Exec:       fmt.Sprintf("hyprctl dispatch focuswindow pid:%s", v.pid),
			Categories: []string{"hyprland", "windows", fmt.Sprintf("workspace %s", v.workspace), fmt.Sprintf("ws %s", v.workspace), v.initialTitle},
			Class:      "hyprland",
			History:    false,
			Matching:   Fuzzy,
		}

		entries = append(entries, n)
	}

	return entries
}
