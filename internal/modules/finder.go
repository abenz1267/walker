package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/boyter/gocodewalker"
)

type Finder struct {
	mutex       sync.Mutex
	general     config.GeneralModule
	concurrency int
	entries     []util.Entry
}

func (f *Finder) General() *config.GeneralModule {
	return &f.general
}

func (f *Finder) Cleanup() {}

func (f *Finder) Refresh() {
	f.general.IsSetup = !f.general.Refresh
}

func (f *Finder) Entries(ctx context.Context, term string) []util.Entry {
	return f.entries
}

func (f *Finder) Setup(cfg *config.Config) bool {
	f.general = cfg.Builtins.Finder.GeneralModule
	f.concurrency = cfg.Builtins.Finder.Concurrency

	if cfg.Builtins.Finder.EagerLoading {
		go f.SetupData(cfg, context.Background())
	}

	return true
}

func (f *Finder) SetupData(cfg *config.Config, ctx context.Context) {
	f.entries = []util.Entry{}

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	fileListQueue := make(chan *gocodewalker.File)

	fileWalker := gocodewalker.NewFileWalker(homedir, fileListQueue)
	fileWalker.IgnoreGitIgnore = cfg.Builtins.Finder.IgnoreGitIgnore

	errorHandler := func(e error) bool {
		return true
	}

	fileWalker.SetConcurrency(f.concurrency)

	fileWalker.SetErrorHandler(errorHandler)

	done := make(chan bool)

	go func(isWalking chan bool) {
		err := fileWalker.Start()
		if err == nil {
			isWalking <- false
		}
	}(done)

	for {
		select {
		case <-done:
			fileWalker.Terminate()
			f.general.IsSetup = true
			f.general.HasInitialSetup = true
			return
		case <-ctx.Done():
			fileWalker.Terminate()
			f.general.IsSetup = true
			f.general.HasInitialSetup = true
			return
		case file := <-fileListQueue:
			if file == nil {
				continue
			}

			f.entries = append(f.entries, util.Entry{
				Label:            strings.TrimPrefix(strings.TrimPrefix(file.Location, homedir), "/"),
				Sub:              "finder",
				Exec:             fmt.Sprintf("xdg-open %s", file.Location),
				RecalculateScore: true,
				DragDrop:         true,
				DragDropData:     file.Location,
				Categories:       []string{"finder", "fzf"},
				Class:            "finder",
				Matching:         util.Fuzzy,
			})
		}
	}
}
