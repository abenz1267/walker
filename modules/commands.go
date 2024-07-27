package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type Commands struct {
	general config.GeneralModule
	entries []util.Entry
}

func (c Commands) History() bool {
	return c.general.History
}

func (c Commands) Typeahead() bool {
	return c.general.Typeahead
}

func (c Commands) IsSetup() bool {
	return c.general.IsSetup
}

func (Commands) KeepSort() bool {
	return false
}

func (c Commands) Placeholder() string {
	if c.general.Placeholder == "" {
		return "commands"
	}

	return c.general.Placeholder
}

func (c Commands) Entries(ctx context.Context, term string) []util.Entry {
	return c.entries
}

func (c Commands) Prefix() string {
	return c.general.Prefix
}

func (c Commands) Name() string {
	return "commands"
}

func (c Commands) SwitcherOnly() bool {
	return c.general.SwitcherOnly
}

func (c *Commands) Setup(cfg *config.Config) bool {
	c.general = cfg.Builtins.Commands.GeneralModule

	return true
}

func (c *Commands) SetupData(cfg *config.Config) {
	entries := []struct {
		label string
		exec  string
	}{
		{
			label: "Reload Config",
			exec:  "reloadconfig",
		},
		{
			label: "Reset History",
			exec:  "resethistory",
		},
		{
			label: "Clear Clipboard",
			exec:  "clearclipboard",
		},
		{
			label: "Clear Applications Cache",
			exec:  "clearapplicationscache",
		},
		{
			label: "Clear Typeahead Cache",
			exec:  "cleartypeaheadcache",
		},
	}

	for _, v := range entries {
		c.entries = append(c.entries, util.Entry{
			Label:            v.label,
			Sub:              "Walker",
			Exec:             v.exec,
			RecalculateScore: true,
		})
	}

	c.general.IsSetup = true
}

func (c Commands) Refresh() {}
