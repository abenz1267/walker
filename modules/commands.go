package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
)

type Commands struct {
	general config.GeneralModule
	entries []Entry
}

func (c Commands) IsSetup() bool {
	return c.general.IsSetup
}

func (c Commands) Placeholder() string {
	if c.general.Placeholder == "" {
		return "commands"
	}

	return c.general.Placeholder
}

func (c Commands) Entries(ctx context.Context, term string) []Entry {
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

func (c *Commands) Setup(cfg *config.Config) {
	c.general.Prefix = cfg.Builtins.Commands.Prefix
	c.general.SwitcherOnly = cfg.Builtins.Commands.SwitcherOnly
	c.general.SpecialLabel = cfg.Builtins.Commands.SpecialLabel
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
		c.entries = append(c.entries, Entry{
			Label:            v.label,
			Sub:              "Walker",
			Exec:             v.exec,
			RecalculateScore: true,
		})
	}

	c.general.IsSetup = true
}

func (c Commands) Refresh() {}
