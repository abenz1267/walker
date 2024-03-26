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
	prefix            string
	switcherExclusive bool
}

func (h Hyprland) HandleWorkspace(number int) {
	current := h.getWindows()

	exists := make(map[string]bool)

	for _, v := range current {
		exists[v.title] = true
	}

	done := make(chan bool)

	go func(done chan bool) {
		time.Sleep(1 * time.Second)

		n := h.getWindows()

		fmt.Println(exists)

		for _, v := range n {
			if _, ok := exists[v.title]; !ok {
				fmt.Println("switching")
				cmd := exec.Command("hyprctl", "dispatch", "movetoworkspacesilent", fmt.Sprintf("%d,title:%s", number, v.title))

				err := cmd.Run()
				if err != nil {
					log.Println(err)
				}
			}
		}

		done <- true
	}(done)

	<-done
}

func (h Hyprland) SwitcherExclusive() bool {
	return h.switcherExclusive
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
	h.switcherExclusive = module.SwitcherExclusive

	return h
}

func (Hyprland) Name() string {
	return "hyprland"
}

func (h Hyprland) Prefix() string {
	return h.prefix
}

type window struct {
	title        string
	pid          string
	workspace    string
	initialTitle string
}

func (Hyprland) getWindows() []window {
	cmd := exec.Command("hyprctl", "clients")

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Panicln(err)
	}

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

	return windows
}

func (h Hyprland) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	for _, v := range h.getWindows() {
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
