package modules

import (
	"context"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Switcher struct {
	config    config.Switcher
	available []string
}

func (s *Switcher) General() *config.GeneralModule {
	return &s.config.GeneralModule
}

func (s Switcher) Cleanup() {}

func (s Switcher) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	for _, v := range s.available {
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
	s.config = cfg.Builtins.Switcher

	s.available = cfg.Available

	s.config.IsSetup = true

	return true
}

func (s *Switcher) SetupData(cfg *config.Config, ctx context.Context) {
	s.config.HasInitialSetup = true
}

func (s *Switcher) Refresh() {
	s.config.IsSetup = !s.config.Refresh
}
