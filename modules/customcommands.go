package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type CustomCommands struct {
	general config.GeneralModule
	entries []Entry
}

func (c CustomCommands) Entries(ctx context.Context, term string) (_ []Entry) {
	return c.entries
}

func (c CustomCommands) Prefix() (_ string) {
	return c.general.Prefix
}

func (CustomCommands) Name() (_ string) {
	return "custom_commands"
}

func (c CustomCommands) Placeholder() (_ string) {
	if c.general.Placeholder == "" {
		return "Commands"
	}

	return c.general.Placeholder
}

func (c CustomCommands) SwitcherOnly() (_ bool) {
	return c.general.SwitcherOnly
}

func (c CustomCommands) IsSetup() (_ bool) {
	return c.general.IsSetup
}

func (c *CustomCommands) Setup(cfg *config.Config) {
	c.general = cfg.Builtins.CustomCommands.GeneralModule
}

func (c *CustomCommands) SetupData(cfg *config.Config) {
	c.entries = []Entry{}

	for _, v := range cfg.Builtins.CustomCommands.Commands {
		c.entries = append(c.entries, Entry{
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

func (CustomCommands) Refresh() {}

func (CustomCommands) KeepSort() bool {
	return false
}
