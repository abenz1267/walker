package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type CustomCommands struct {
	general config.GeneralModule
	entries []util.Entry
}

func (c *CustomCommands) General() *config.GeneralModule {
	return &c.general
}

func (c CustomCommands) Cleanup() {}

func (c CustomCommands) Entries(ctx context.Context, term string) (_ []util.Entry) {
	return c.entries
}

func (c *CustomCommands) Setup(cfg *config.Config) bool {
	c.general = cfg.Builtins.CustomCommands.GeneralModule

	return true
}

func (c *CustomCommands) SetupData(cfg *config.Config, ctx context.Context) {
	c.entries = []util.Entry{}

	for _, v := range cfg.Builtins.CustomCommands.Commands {
		c.entries = append(c.entries, util.Entry{
			Label:            v.Name,
			Sub:              "Commands",
			Exec:             v.Cmd,
			ExecAlt:          v.CmdAlt,
			Terminal:         v.Terminal,
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	c.general.IsSetup = true
}

func (c *CustomCommands) Refresh() {
	c.general.IsSetup = !c.general.Refresh
}
