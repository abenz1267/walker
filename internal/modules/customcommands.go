package modules

import (
	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type CustomCommands struct {
	config  config.CustomCommands
	entries []*util.Entry
}

func (c *CustomCommands) General() *config.GeneralModule {
	return &c.config.GeneralModule
}

func (c CustomCommands) Cleanup() {}

func (c CustomCommands) Entries(term string) (_ []*util.Entry) {
	return c.entries
}

func (c *CustomCommands) Setup() bool {
	c.config = config.Cfg.Builtins.CustomCommands

	return true
}

func (c *CustomCommands) SetupData() {
	c.entries = []*util.Entry{}

	for _, v := range config.Cfg.Builtins.CustomCommands.Commands {
		c.entries = append(c.entries, &util.Entry{
			Label:             v.Name,
			Sub:               "Commands",
			Icon:              v.Icon,
			Exec:              v.Cmd,
			ExecAlt:           v.CmdAlt,
			Terminal:          v.Terminal,
			Matching:          util.Fuzzy,
			Path:              v.Path,
			TerminalTitleFlag: v.TerminalTitleFlag,
			Env:               v.Env,
			RecalculateScore:  true,
		})
	}

	c.config.IsSetup = true
	c.config.HasInitialSetup = true
}

func (c *CustomCommands) Refresh() {
	if c.config.HasInitialSetup {
		c.config.IsSetup = !c.config.Refresh
	}
}
