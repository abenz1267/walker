package modules

import (
	"context"

	"github.com/abenz1267/walker/config"
)

type Switcher struct {
	prefix string
	Procs  map[string][]Workable
}

func (s Switcher) SwitcherExclusive() bool {
	return false
}

func (s Switcher) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	for _, v := range s.Procs {
		for _, w := range v {
			if w.Name() == "switcher" {
				continue
			}

			e := Entry{
				Label:      w.Name(),
				Sub:        "switcher",
				Exec:       "",
				Categories: []string{"switcher"},
				Class:      "switcher",
				Matching:   Fuzzy,
			}

			entries = append(entries, e)
		}
	}

	return entries
}

func (s Switcher) Prefix() string {
	return s.prefix
}

func (s Switcher) Name() string {
	return "switcher"
}

func (s Switcher) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, s.Name())
	if module == nil {
		return nil
	}

	s.prefix = module.Prefix

	return s
}

func (s Switcher) Refresh() {}
