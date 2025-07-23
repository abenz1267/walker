package symbols

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	_ "embed"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

//go:embed UnicodeData.txt
var list string

type Symbols struct {
	config  config.Symbols
	entries []*util.Entry
	exec    string
	execAlt string
}

func (e *Symbols) General() *config.GeneralModule {
	return &e.config.GeneralModule
}

func (e Symbols) Cleanup() {}

func (e Symbols) Entries(term string) []*util.Entry {
	return e.entries
}

func (e *Symbols) Setup() bool {
	e.config = config.Cfg.Builtins.Symbols
	e.exec = config.Cfg.Builtins.Symbols.Exec
	e.execAlt = config.Cfg.Builtins.Symbols.ExecAlt

	return true
}

func (e *Symbols) SetupData() {
	scanner := bufio.NewScanner(strings.NewReader(list))

	entries := []*util.Entry{}

	explicitResult := strings.Contains(e.exec, "%RESULT%")
	explicitResultAlt := strings.Contains(e.execAlt, "%RESULT%")

	for scanner.Scan() {
		text := scanner.Text()

		fields := strings.Split(text, ";")

		symbol := fmt.Sprintf("'\\u%s'", fields[0])

		toUse, err := strconv.Unquote(symbol)
		if err != nil {
			continue
		}

		exec := e.exec
		execAlt := e.execAlt

		if explicitResult {
			exec = strings.ReplaceAll(exec, "%RESULT%", toUse)
		}

		if explicitResultAlt {
			execAlt = strings.ReplaceAll(execAlt, "%RESULT%", toUse)
		}

		entry := util.Entry{
			Label:            fmt.Sprintf("%s %s", toUse, fields[1]),
			Sub:              "Symbols",
			Exec:             exec,
			ExecAlt:          execAlt,
			Searchable:       fields[1],
			Class:            "symbols",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		}

		if !explicitResult {
			entry.Piped = util.Piped{String: toUse, Type: "string"}
		}

		if !explicitResultAlt {
			entry.PipedAlt = util.Piped{String: toUse, Type: "string"}
		}

		entries = append(entries, &entry)
	}

	e.entries = entries

	e.config.IsSetup = true
	e.config.HasInitialSetup = true
}

func (e *Symbols) Refresh() {
	e.config.IsSetup = !e.config.Refresh
}
