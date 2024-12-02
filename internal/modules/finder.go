package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/boyter/gocodewalker"
)

type Finder struct {
	config  config.Finder
	files   []string
	homedir string
	hasList bool
}

func (f *Finder) General() *config.GeneralModule {
	return &f.config.GeneralModule
}

func (f *Finder) Cleanup() {}

func (f *Finder) Refresh() {
	f.config.IsSetup = false
}

func (f *Finder) Entries(ctx context.Context, term string) []util.Entry {
	for !f.hasList {
	}

	entries := []util.Entry{}

	scoremin := 50.0

	if term == "" {
		scoremin = 0.
	}

	for _, v := range f.files {
		score := util.FuzzyScore(term, v)

		if score >= scoremin {
			entries = append(entries, util.Entry{
				Label:            strings.TrimPrefix(strings.TrimPrefix(v, f.homedir), "/"),
				Sub:              "finder",
				Exec:             fmt.Sprintf("xdg-open %s", v),
				RecalculateScore: false,
				ScoreFinal:       score,
				DragDrop:         true,
				DragDropData:     v,
				Categories:       []string{"finder", "fzf"},
				Class:            "finder",
				Matching:         util.Fuzzy,
			})
		}
	}

	return entries
}

func (f *Finder) Setup(cfg *config.Config) bool {
	f.config = cfg.Builtins.Finder

	if cfg.Builtins.Finder.EagerLoading {
		go f.SetupData(cfg, context.Background())
	}

	return true
}

func (f *Finder) SetupData(cfg *config.Config, ctx context.Context) {
	f.config.HasInitialSetup = true
	f.config.IsSetup = true

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	f.homedir = homedir

	fileListQueue := make(chan *gocodewalker.File)

	fileWalker := gocodewalker.NewFileWalker(homedir, fileListQueue)
	fileWalker.IgnoreGitIgnore = cfg.Builtins.Finder.IgnoreGitIgnore

	errorHandler := func(e error) bool {
		return true
	}

	fileWalker.SetConcurrency(f.config.Concurrency)
	fileWalker.SetErrorHandler(errorHandler)

	done := make(chan struct{})

	go func(done chan struct{}) {
		_ = fileWalker.Start()
		done <- struct{}{}
	}(done)

	go func(done chan struct{}) {
		for {
			select {
			case <-done:
				f.hasList = true
				return
			case file := <-fileListQueue:
				if file == nil {
					continue
				}

				f.files = append(f.files, file.Location)
			}
		}
	}(done)
}
