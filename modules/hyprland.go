package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Hyprland struct {
	Prfx string
}

func (Hyprland) Name() string {
	return "hyprlandwindows"
}

func (h Hyprland) Prefix() string {
	return h.Prfx
}

func (h *Hyprland) SetPrefix(val string) {
	h.Prfx = val
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
			Label:      v.title,
			Sub:        "Hyprland",
			Exec:       fmt.Sprintf("hyprctl dispatch focuswindow pid:%s", v.pid),
			Categories: []string{"hyprland", "windows"},
			Class:      "hyprland",
			Notifyable: false,
			History:    false,
		}

		entries = append(entries, n)
	}

	return entries
}
