package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Hyprland struct {
	prefix string
}

func (h Hyprland) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, h.Name())
	if module == nil {
		return nil
	}

	pth, _ := exec.LookPath("hyprctl")
	if pth == "" {
		log.Println("Hyprland not found. Disabling module.")
		return nil
	}

	h.prefix = module.Prefix

	return h
}

func (Hyprland) Name() string {
	return "hyprland"
}

func (h Hyprland) Prefix() string {
	return h.prefix
}

type window struct {
	title string
	pid   string
}

func (Hyprland) Entries(term string) []Entry {
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
			Label:             v.title,
			Sub:               "Hyprland",
			Exec:              fmt.Sprintf("hyprctl dispatch focuswindow pid:%s", v.pid),
			Categories:        []string{"hyprland", "windows"},
			Class:             "hyprland",
			Notifyable:        false,
			History:           false,
			Matching:          Fuzzy,
			MinScoreToInclude: 50,
		}

		entries = append(entries, n)
	}

	return entries
}
