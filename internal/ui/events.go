package ui

import (
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
)

func executeEvent(eventType config.EventType, label string) {
	if config.Cfg == nil {
		return
	}

	go func() {
		cmd := exec.Command("sh", "-c")

		toRun := ""

		switch eventType {
		case config.EventLaunch:
			toRun = config.Cfg.Events.OnLaunch
		case config.EventSelection:
			toRun = config.Cfg.Events.OnSelection
		case config.EventExit:
			toRun = config.Cfg.Events.OnExit
		case config.EventActivate:
			toRun = config.Cfg.Events.OnActivate
		case config.EventQueryChange:
			toRun = config.Cfg.Events.OnQueryChange
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
