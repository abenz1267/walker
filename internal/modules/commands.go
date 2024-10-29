package modules

import (
	"context"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Commands struct {
	general config.GeneralModule
	entries []util.Entry
}

func (c *Commands) General() *config.GeneralModule {
	return &c.general
}

func (c Commands) Cleanup() {}

func (c Commands) Entries(ctx context.Context, term string) []util.Entry {
	return c.entries
}

func (c *Commands) Setup(cfg *config.Config) bool {
	c.general = cfg.Builtins.Commands.GeneralModule

	return true
}

func (c *Commands) SetupData(cfg *config.Config, ctx context.Context) {
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
	c.general.HasInitialSetup = true
}

func (c *Commands) Refresh() {
	c.general.IsSetup = !c.general.Refresh
}
