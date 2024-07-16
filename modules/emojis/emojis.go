package emojis

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
)

//go:embed list.csv
var list string

type Emojis struct {
	entries           []modules.Entry
	prefix            string
	switcherExclusive bool
}

func (e Emojis) Entries(ctx context.Context, term string) []modules.Entry {
	return e.entries
}

func (e Emojis) Prefix() string {
	return e.prefix
}

func (Emojis) Name() string {
	return "emojis"
}

func (e Emojis) SwitcherExclusive() bool {
	return e.switcherExclusive
}

func (e Emojis) Setup(cfg *config.Config) modules.Workable {
	module := modules.Find(cfg.Modules, e.Name())
	if module == nil {
		return nil
	}

	e.prefix = module.Prefix
	e.switcherExclusive = module.SwitcherExclusive

	scanner := bufio.NewScanner(strings.NewReader(list))

	entries := []modules.Entry{}

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Group") {
			continue
		}

		fields := strings.Split(text, ",")

		entries = append(entries, modules.Entry{
			Label:            fmt.Sprintf("%s %s", fields[4], fields[5]),
			Sub:              "Emojis",
			Exec:             fmt.Sprintf("wl-copy %s", fields[4]),
			Searchable:       fields[5],
			Categories:       []string{fields[0], fields[1]},
			Class:            "emojis",
			Matching:         modules.Fuzzy,
			RecalculateScore: true,
		})
	}

	e.entries = entries

	return e
}

func (Emojis) Refresh() {}
