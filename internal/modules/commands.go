package modules

import (
	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Commands struct {
	config  config.Commands
	entries []*util.Entry
}

func (c *Commands) General() *config.GeneralModule {
	return &c.config.GeneralModule
}

func (c Commands) Cleanup() {}

func (c Commands) Entries(term string) []*util.Entry {
	return c.entries
}

func (c *Commands) Setup() bool {
	c.config = config.Cfg.Builtins.Commands

	return true
}

func (c *Commands) SetupData() {
	entries := []struct {
		label string
		exec  string
	}{
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
		{
			label: "Adjust Theme",
			exec:  "adjusttheme",
		},
	}

	for _, v := range entries {
		c.entries = append(c.entries, &util.Entry{
			Label:            v.label,
			Sub:              "Walker",
			Exec:             v.exec,
			RecalculateScore: true,
		})
	}

	c.config.IsSetup = true
	c.config.HasInitialSetup = true
}

func (c *Commands) Refresh() {
	c.config.IsSetup = !c.config.Refresh
}
