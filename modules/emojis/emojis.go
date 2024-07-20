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
	general config.GeneralModule
	entries []modules.Entry
}

func (e Emojis) IsSetup() bool {
	return e.general.IsSetup
}

func (e Emojis) Entries(ctx context.Context, term string) []modules.Entry {
	return e.entries
}

func (e Emojis) Prefix() string {
	return e.general.Prefix
}

func (Emojis) Name() string {
	return "emojis"
}

func (e Emojis) SwitcherOnly() bool {
	return e.general.SwitcherOnly
}

func (e *Emojis) Setup(cfg *config.Config) {
	e.general.Prefix = cfg.Builtins.Emojis.Prefix
	e.general.SwitcherOnly = cfg.Builtins.Emojis.SwitcherOnly
	e.general.SpecialLabel = cfg.Builtins.Emojis.SpecialLabel
}

func (e *Emojis) SetupData(cfg *config.Config) {
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

	e.general.IsSetup = true
}

func (e Emojis) Placeholder() string {
	if e.general.Placeholder == "" {
		return "emojis"
	}

	return e.general.Placeholder
}

func (Emojis) Refresh() {}
