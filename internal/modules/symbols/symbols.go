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
	entries []util.Entry
}

func (e *Symbols) General() *config.GeneralModule {
	return &e.config.GeneralModule
}

func (e Symbols) Cleanup() {}

func (e Symbols) Entries(term string) []util.Entry {
	return e.entries
}

func (e *Symbols) Setup() bool {
	e.config = config.Cfg.Builtins.Symbols

	return true
}

func (e *Symbols) SetupData() {
	scanner := bufio.NewScanner(strings.NewReader(list))

	entries := []util.Entry{}

	for scanner.Scan() {
		text := scanner.Text()

		fields := strings.Split(text, ";")

		symbol := fmt.Sprintf("'\\u%s'", fields[0])

		toUse, err := strconv.Unquote(symbol)
		if err != nil {
			continue
		}

		exec := fmt.Sprintf("wl-copy '%s'", toUse)

		if e.config.AfterCopy != "" {
			exec = fmt.Sprintf("wl-copy '%s' | %s", toUse, e.config.AfterCopy)
		}

		entries = append(entries, util.Entry{
			Label:            fmt.Sprintf("%s - %s", toUse, fields[1]),
			Sub:              "Symbols",
			Exec:             exec,
			Class:            "symbols",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	e.entries = entries

	e.config.IsSetup = true
	e.config.HasInitialSetup = true
}

func (e *Symbols) Refresh() {
	e.config.IsSetup = !e.config.Refresh
}
