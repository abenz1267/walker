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

	explicitResult := strings.Contains(e.exec, "%RESULT%")
	explicitResultAlt := strings.Contains(e.execAlt, "%RESULT%")

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Group") {
			continue
		}

		fields := strings.Split(text, ",")

		if !e.showUnqualified && fields[3] == "unqualified" {
			continue
		}

		exec := e.exec
		execAlt := e.execAlt

		if explicitResult {
			exec = strings.ReplaceAll(exec, "%RESULT%", fields[4])
		}

		if explicitResultAlt {
			execAlt = strings.ReplaceAll(execAlt, "%RESULT%", fields[4])
		}

		entry := util.Entry{
			Label:            fmt.Sprintf("%s %s", fields[4], fields[5]),
			Sub:              "Emojis",
			Exec:             exec,
			ExecAlt:          execAlt,
			Searchable:       fields[5],
			Categories:       []string{fields[0], fields[1]},
			Class:            "emojis",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		}

		if !explicitResult {
			entry.Piped = util.Piped{String: fields[4], Type: "string"}
		}

		if !explicitResultAlt {
			entry.PipedAlt = util.Piped{String: fields[4], Type: "string"}
		}

		entries = append(entries, &entry)
	}

	e.entries = entries

	e.general.IsSetup = true
	e.general.HasInitialSetup = true
}

func (e *Emojis) Refresh() {
	e.general.IsSetup = !e.general.Refresh
}
