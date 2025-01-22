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

var imgExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp"}

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

	exact := false

	if strings.HasPrefix(term, "'") {
		exact = true
		term = strings.TrimPrefix(term, "'")
	}

	if term == "" {
		scoremin = 0.

		if len(f.files) > 100 {
			toCheck = f.files[:100]
		}
	}

	for _, v := range toCheck {
		var score float64
		var pos *[]int

		if exact {
			score, _ = util.ExactScore(term, v)
			f := strings.Index(strings.ToLower(v), strings.ToLower(term))

			if f != -1 {
				poss := []int{}

				for i := f; i < f+len(term); i++ {
					poss = append(poss, i)
				}

				pos = &poss
			}
		} else {
			score, pos = util.FuzzyScore(term, v)
		}

		ddd := v
		label := strings.TrimPrefix(strings.TrimPrefix(v, f.homedir), "/")

		if f.config.UseFD {
			ddd = filepath.Join(f.homedir, v)
			label = v
		}

		hasExplicitResultAlt := false

		if strings.Contains(f.config.CmdAlt, "%RESULT%") {
			hasExplicitResultAlt = true
		}

		if score >= scoremin {
			entry := util.Entry{
				Label:            label,
				Sub:              "finder",
				Exec:             fmt.Sprintf("xdg-open %s", v),
				ExecAlt:          strings.ReplaceAll(f.config.CmdAlt, "%RESULT%", label),
				RecalculateScore: false,
				ScoreFinal:       score,
				DragDrop:         true,
				DragDropData:     ddd,
				Categories:       []string{"finder", "fzf"},
				Class:            "finder",
				Matching:         util.Fuzzy,
			}

			if !hasExplicitResultAlt {
				entry.PipedAlt.String = label
				entry.PipedAlt.Type = "string"
			}

			if f.config.PreviewImages {
				ext := filepath.Ext(v)

				if slices.Contains(imgExtensions, ext) {
					path := filepath.Join(f.homedir, v)
					entry.Image = path
				}
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

func (f *Finder) Setup() bool {
	f.config = config.Cfg.Builtins.Finder

	return true
}

func (f *Finder) SetupData() {
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
		fileWalker.IgnoreGitIgnore = config.Cfg.Builtins.Finder.IgnoreGitIgnore

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
