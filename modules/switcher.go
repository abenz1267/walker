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

func (s *Switcher) General() *config.GeneralModule {
	return &s.general
}

func (s Switcher) Cleanup() {}

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

func (s *Switcher) Setup(cfg *config.Config) bool {
	s.general = cfg.Builtins.Switcher.GeneralModule

	s.cfg = cfg

	s.general.IsSetup = true

	return true
}

func (s *Switcher) SetupData(cfg *config.Config, ctx context.Context) {
	s.general.HasInitialSetup = true
}

func (s *Switcher) Refresh() {
	s.general.IsSetup = !s.general.Refresh
}
