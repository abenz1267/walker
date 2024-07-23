package modules

import (
	"context"
	"strconv"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Dmenu struct {
	isSetup     bool
	Content     []string
	Separator   string
	LabelColumn int
}

func (d Dmenu) IsSetup() bool {
	return d.isSetup
}

func (d Dmenu) KeepSort() bool {
	return false
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

func (d *Dmenu) Setup(cfg *config.Config) bool {
	if d.Separator == "" {
		d.Separator = "\t"
	}

	s, err := strconv.Unquote(d.Separator)
	if err == nil {
		d.Separator = s
	}

	d.isSetup = true

	return true
}

func (d *Dmenu) SetupData(cfg *config.Config) {}

func (Dmenu) Typeahead() bool {
	return false
}

func (Dmenu) History() bool {
	return false
}

func (d Dmenu) Placeholder() string {
	if d.Separator == "" {
		return "dmenu"
	}

	return d.Separator
}

func (Dmenu) Refresh() {}
