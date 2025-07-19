package emojis

import (
	"bufio"
	"fmt"
	"strings"

	_ "embed"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

//go:embed list.csv
var list string

type Emojis struct {
	general         config.GeneralModule
	entries         []*util.Entry
	showUnqualified bool
	exec            string
	execAlt         string
}

func (e *Emojis) General() *config.GeneralModule {
	return &e.general
}

func (e Emojis) Cleanup() {}

func (e Emojis) Entries(term string) []*util.Entry {
	return e.entries
}

func (e *Emojis) Setup() bool {
	e.general = config.Cfg.Builtins.Emojis.GeneralModule
	e.showUnqualified = config.Cfg.Builtins.Emojis.ShowUnqualified
	e.exec = config.Cfg.Builtins.Emojis.Exec
	e.execAlt = config.Cfg.Builtins.Emojis.ExecAlt

	return true
}

func (e *Emojis) SetupData() {
	scanner := bufio.NewScanner(strings.NewReader(list))

	entries := []*util.Entry{}

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Group") {
			continue
		}

		fields := strings.Split(text, ",")

		if !e.showUnqualified && fields[3] == "unqualified" {
			continue
		}

		entries = append(entries, &util.Entry{
			Label:            fmt.Sprintf("%s %s", fields[4], fields[5]),
			Sub:              "Emojis",
			Exec:             e.exec,
			ExecAlt:          e.execAlt,
			Piped:            util.Piped{String: fields[4], Type: "string"},
			Searchable:       fields[5],
			Categories:       []string{fields[0], fields[1]},
			Class:            "emojis",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	e.entries = entries

	e.general.IsSetup = true
	e.general.HasInitialSetup = true
}

func (e *Emojis) Refresh() {
	e.general.IsSetup = !e.general.Refresh
}
