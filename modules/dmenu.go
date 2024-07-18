package modules

import (
	"context"
	"fmt"

	"github.com/abenz1267/walker/config"
)

type Dmenu struct {
	Content []string
}

func (d Dmenu) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	for _, v := range d.Content {
		entries = append(entries, Entry{
			Label: v,
			Sub:   "Dmenu",
			Exec:  fmt.Sprintf("echo '%s'", v),
		})
	}

	return entries
}

func (Dmenu) Prefix() string {
	return ""
}

func (Dmenu) Name() string {
	return "dmenu"
}

func (Dmenu) SwitcherExclusive() bool {
	return false
}

func (d Dmenu) Setup(cfg *config.Config, config *config.Module) Workable {
	return d
}

func (Dmenu) Refresh() {}
