package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/boyter/gocodewalker"
	"golang.org/x/exp/slog"
)

type Finder struct {
	config      config.Finder
	files       []string
	homedir     string
	hasList     bool
	MarkerColor string
}

func (f *Finder) General() *config.GeneralModule {
	return &f.config.GeneralModule
}

func (f *Finder) Cleanup() {}

func (f *Finder) Refresh() {
	f.config.IsSetup = false
}

func (f *Finder) Entries(term string) []util.Entry {
	for !f.hasList {
	}

	entries := []util.Entry{}

	scoremin := 50.0

	toCheck := f.files

	if term == "" {
		scoremin = 0.

		if len(f.files) > 100 {
			toCheck = f.files[:100]
		}
	}

	for _, v := range toCheck {
		score, pos := util.FuzzyScore(term, v)

		ddd := v
		label := strings.TrimPrefix(strings.TrimPrefix(v, f.homedir), "/")

		if f.config.UseFD {
			ddd = filepath.Join(f.homedir, v)
			label = v
		}

		if score >= scoremin {
			entry := util.Entry{
				Label:            label,
				Sub:              "finder",
				Exec:             fmt.Sprintf("xdg-open %s", v),
				RecalculateScore: false,
				ScoreFinal:       score,
				DragDrop:         true,
				DragDropData:     ddd,
				Categories:       []string{"finder", "fzf"},
				Class:            "finder",
				Matching:         util.Fuzzy,
			}

			res := ""

			if f.MarkerColor != "" {
				if pos != nil {
					for k, v := range label {
						if slices.Contains(*pos, k) {
							res = fmt.Sprintf("%s<span color=\"%s\">%s</span>", res, f.MarkerColor, string(v))
						} else {
							res = fmt.Sprintf("%s%s", res, string(v))
						}
					}
				}

				entry.MatchedLabel = res
			}

			entries = append(entries, entry)
		}
	}

	return entries
}

func (f *Finder) Setup(cfg *config.Config) bool {
	f.config = cfg.Builtins.Finder

	if cfg.Builtins.Finder.EagerLoading {
		go f.SetupData(cfg)
	}

	return true
}

func (f *Finder) SetupData(cfg *config.Config) {
	f.config.HasInitialSetup = true
	f.config.IsSetup = true
	f.files = []string{}
	f.hasList = false

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	f.homedir = homedir

	if f.config.UseFD {
		cmd := exec.Command("fd", "--ignore-vcs", "--type", "file")

		if f.config.IgnoreGitIgnore {
			cmd = exec.Command("fd", "--no-ignore-vcs", "--type", "file")
		}

		cmd.Dir = homedir

		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("finder", "error", err.Error())
		}

		scanner := bufio.NewScanner(bytes.NewReader(out))

		for scanner.Scan() {
			f.files = append(f.files, scanner.Text())
		}

		slices.Sort(f.files)

		f.hasList = true
	} else {
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
}
