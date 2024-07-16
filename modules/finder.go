package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/boyter/gocodewalker"
)

type Finder struct {
	prefix            string
	switcherExclusive bool
}

func (f Finder) Refresh() {}

func (f Finder) Entries(ctx context.Context, term string) []Entry {
	e := []Entry{}

	if len(term) < 2 {
		return e
	}

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

	for f := range fileListQueue {
		e = append(e, Entry{
			Label:        strings.TrimPrefix(strings.TrimPrefix(f.Location, homedir), "/"),
			Sub:          "fzf",
			Exec:         fmt.Sprintf("xdg-open %s", f.Location),
			DragDrop:     true,
			DragDropData: f.Location,
			Categories:   []string{"finder", "fzf"},
			Class:        "finder",
			Matching:     Fuzzy,
		})
	}

	return e
}

func (f Finder) Prefix() string {
	return f.prefix
}

func (f Finder) Name() string {
	return "finder"
}

func (f Finder) SwitcherExclusive() bool {
	return f.switcherExclusive
}

func (Finder) Setup(cfg *config.Config) Workable {
	f := &Finder{}

	module := Find(cfg.Modules, f.Name())
	if module == nil {
		return nil
	}

	f.prefix = module.Prefix
	f.switcherExclusive = module.SwitcherExclusive

	return f
}
