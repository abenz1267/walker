package emojis

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

//go:embed list.csv
var list string

type Emojis struct {
	general config.GeneralModule
	entries []util.Entry
}

func (e *Emojis) General() *config.GeneralModule {
	return &e.general
}

func (e Emojis) Cleanup() {}

func (e Emojis) Entries(ctx context.Context, term string) []util.Entry {
	return e.entries
}

func (e *Emojis) Setup(cfg *config.Config) bool {
	e.general = cfg.Builtins.Emojis.GeneralModule

	return true
}

func (e *Emojis) SetupData(cfg *config.Config, ctx context.Context) {
	scanner := bufio.NewScanner(strings.NewReader(list))

	entries := []util.Entry{}

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Group") {
			continue
		}

		fields := strings.Split(text, ",")

		entries = append(entries, util.Entry{
			Label:            fmt.Sprintf("%s %s", fields[4], fields[5]),
			Sub:              "Emojis",
			Exec:             fmt.Sprintf("wl-copy %s", fields[4]),
			Searchable:       fields[5],
			Categories:       []string{fields[0], fields[1]},
			Class:            "emojis",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	e.entries = entries

	e.general.IsSetup = true
}

func (e *Emojis) Refresh() {
	e.general.IsSetup = !e.general.Refresh
}
