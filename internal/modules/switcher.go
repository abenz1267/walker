package modules

import (
	"slices"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Switcher struct {
	config config.Switcher
}

func (s *Switcher) General() *config.GeneralModule {
	return &s.config.GeneralModule
}

func (s Switcher) Cleanup() {}

func (s Switcher) Entries(term string) []*util.Entry {
	entries := []*util.Entry{}

	for _, v := range config.Cfg.Available {
		if v.Name == "switcher" || slices.Contains(config.Cfg.Hidden, v.Name) {
			continue
		}

		e := util.Entry{
			Label:      v.Name,
			Icon:       v.Icon,
			Sub:        "switcher",
			Exec:       "",
			Categories: []string{"switcher"},
			Class:      "switcher",
			Matching:   util.Fuzzy,
		}

		entries = append(entries, &e)
	}

	return entries
}

func (s *Switcher) Setup() bool {
	s.config = config.Cfg.Builtins.Switcher

	s.config.IsSetup = true

	return true
}

func (s *Switcher) SetupData() {
	s.config.HasInitialSetup = true
}

func (s *Switcher) Refresh() {
	s.config.IsSetup = !s.config.Refresh
}
