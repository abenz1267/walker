package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
)

type Commands struct {
	prefix            string
	switcherExclusive bool
	entries           []Entry
}

func (c Commands) Entries(ctx context.Context, term string) []Entry {
	return c.entries
}

func (c Commands) Prefix() string {
	return c.prefix
}

func (c Commands) Name() string {
	return "commands"
}

func (c Commands) SwitcherExclusive() bool {
	return c.switcherExclusive
}

func (cc Commands) Setup(cfg *config.Config, module *config.Module) Workable {
	c := &Commands{
		prefix:            module.Prefix,
		switcherExclusive: module.SwitcherExclusive,
		entries:           []Entry{},
	}

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

	return c
}

func (c Commands) Refresh() {}
