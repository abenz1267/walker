package ui

import (
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
)

func executeEvent(eventType config.EventType, label string) {
	if cfg == nil {
		return
	}

	go func() {
		cmd := exec.Command("sh", "-c")

		toRun := ""

		switch eventType {
		case config.EventLaunch:
			toRun = cfg.Events.OnLaunch
		case config.EventSelection:
			toRun = cfg.Events.OnSelection
		case config.EventExit:
			toRun = cfg.Events.OnExit
		case config.EventActivate:
			toRun = cfg.Events.OnActivate
		case config.EventQueryChange:
			toRun = cfg.Events.OnQueryChange
		}

		if toRun == "" {
			return
		}

		if label != "" {
			toRun = strings.ReplaceAll(toRun, "%LABEL%", label)
		}

		cmd.Args = append(cmd.Args, toRun)
		cmd.Start()
	}()
}
