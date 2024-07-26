package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
	"github.com/boyter/gocodewalker"
)

type Finder struct {
	general config.GeneralModule
	entries []Entry
}

func (f Finder) History() bool {
	return f.general.History
}

func (f Finder) Typeahead() bool {
	return f.general.Typeahead
}

func (Finder) KeepSort() bool {
	return false
}

func (f Finder) IsSetup() bool {
	return f.general.IsSetup
}

func (f Finder) Placeholder() string {
	if f.general.Placeholder == "" {
		return "finder"
	}

	return f.general.Placeholder
}

func (f Finder) Refresh() {
	f.general.IsSetup = false
}

func (f Finder) Entries(ctx context.Context, term string) []Entry {
	return f.entries
}

func (f Finder) Prefix() string {
	return f.general.Prefix
}

func (f Finder) Name() string {
	return "finder"
}

func (f Finder) SwitcherOnly() bool {
	return f.general.SwitcherOnly
}

func (f *Finder) Setup(cfg *config.Config) bool {
	f.general = cfg.Builtins.Finder.GeneralModule

	return true
}

func (f *Finder) SetupData(cfg *config.Config) {
	f.entries = []Entry{}

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	fileListQueue := make(chan *gocodewalker.File)

	fileWalker := gocodewalker.NewFileWalker(homedir, fileListQueue)

	errorHandler := func(e error) bool {
		log.Println(e)
		return true
	}

	fileWalker.SetErrorHandler(errorHandler)

	go fileWalker.Start()

	for file := range fileListQueue {
		f.entries = append(f.entries, Entry{
			Label:            strings.TrimPrefix(strings.TrimPrefix(file.Location, homedir), "/"),
			Sub:              "fzf",
			Exec:             fmt.Sprintf("xdg-open %s", file.Location),
			RecalculateScore: true,
			DragDrop:         true,
			DragDropData:     file.Location,
			Categories:       []string{"finder", "fzf"},
			Class:            "finder",
			Matching:         util.Fuzzy,
		})
	}

	f.general.IsSetup = true
}
