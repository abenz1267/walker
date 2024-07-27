package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type Switcher struct {
	general config.GeneralModule
	cfg     *config.Config
}

func (s Switcher) Cleanup() {}

func (s Switcher) History() bool {
	return s.general.History
}

func (s Switcher) Typeahead() bool {
	return s.general.Typeahead
}

func (Switcher) KeepSort() bool {
	return false
}

func (s Switcher) Placeholder() string {
	if s.general.Placeholder == "" {
		return "switcher"
	}

	return s.general.Placeholder
}

func (s Switcher) SwitcherOnly() bool {
	return false
}

func (s Switcher) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	for _, v := range s.cfg.Available {
		if v == "switcher" {
			continue
		}

		e := util.Entry{
			Label:      v,
			Sub:        "switcher",
			Exec:       "",
			Categories: []string{"switcher"},
			Class:      "switcher",
			Matching:   util.Fuzzy,
		}

		entries = append(entries, e)
	}

	return entries
}

func (s Switcher) Prefix() string {
	return s.general.Prefix
}

func (s Switcher) Name() string {
	return "switcher"
}

func (s *Switcher) Setup(cfg *config.Config) bool {
	s.general = cfg.Builtins.Switcher.GeneralModule
	s.cfg = cfg

	s.general.IsSetup = true

	return true
}

func (s *Switcher) SetupData(cfg *config.Config) {}

func (s Switcher) IsSetup() bool {
	return s.general.IsSetup
}

func (s Switcher) Refresh() {}
