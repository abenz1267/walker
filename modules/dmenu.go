package modules

import (
	"context"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Dmenu struct {
	Content     []string
	LabelColumn int
}

func (d Dmenu) Entries(ctx context.Context, term string) []Entry {
	if d.LabelColumn < 1 {
		d.LabelColumn = 1
	}

	entries := []Entry{}

	for _, v := range d.Content {
		entries = append(entries, Entry{
			Label: strings.Split(v, "\t")[d.LabelColumn-1],
			Sub:   "Dmenu",
			Exec:  v,
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
