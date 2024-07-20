package modules

import (
	"context"
	"strconv"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Dmenu struct {
	Content     []string
	Separator   string
	LabelColumn int
}

func (d Dmenu) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	for _, v := range d.Content {
		label := v

		if d.LabelColumn > 0 {
			split := strings.Split(v, d.Separator)

			if len(split) >= d.LabelColumn {
				label = split[d.LabelColumn-1]
			}
		}

		entries = append(entries, Entry{
			Label: label,
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

func (Dmenu) SwitcherOnly() bool {
	return false
}

func (d *Dmenu) Setup(cfg *config.Config) Workable {
	if d.Separator == "" {
		d.Separator = "\t"
	}

	s, err := strconv.Unquote(d.Separator)
	if err == nil {
		d.Separator = s
	}

	return d
}

func (d Dmenu) Placeholder() string {
	if d.Separator == "" {
		return "dmenu"
	}

	return d.Separator
}

func (Dmenu) Refresh() {}
