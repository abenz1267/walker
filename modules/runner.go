package modules

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type Runner struct {
	general      config.GeneralModule
	shellConfig  string
	genericEntry bool
	aliases      map[string]string
	bins         []string
}

func (r Runner) History() bool {
	return r.general.History
}

func (r Runner) Typeahead() bool {
	return r.general.Typeahead
}

func (Runner) KeepSort() bool {
	return false
}

func (r Runner) IsSetup() bool {
	return r.general.IsSetup
}

func (r Runner) Placeholder() string {
	if r.general.Placeholder == "" {
		return "runner"
	}

	return r.general.Placeholder
}

func (r Runner) SwitcherOnly() bool {
	return r.general.SwitcherOnly
}

func (r *Runner) Setup(cfg *config.Config) bool {
	r.general = cfg.Builtins.Runner.GeneralModule
	r.shellConfig = cfg.Builtins.Runner.ShellConfig
	r.genericEntry = cfg.Builtins.Runner.GenericEntry

	return true
}

func (r *Runner) SetupData(cfg *config.Config) {
	r.parseAliases()

	if len(cfg.Builtins.Runner.Includes) > 0 {
		r.bins = cfg.Builtins.Runner.Includes
	} else {
		r.getBins()
	}

	filtered := []string{}

	if len(cfg.Builtins.Runner.Excludes) > 0 {
		for _, v := range r.bins {
			if !slices.Contains(cfg.Builtins.Runner.Excludes, v) {
				filtered = append(filtered, v)
			}
		}

		r.bins = filtered
	}

	r.general.IsSetup = true
}

func (r Runner) Refresh() {}

func (r Runner) Prefix() string {
	return r.general.Prefix
}

func (Runner) Name() string {
	return "runner"
}

func (r Runner) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	fields := strings.Fields(term)

	if len(fields) == 0 {
		return entries
	}

	for _, v := range r.bins {
		label := v

		if val, ok := r.aliases[v]; ok {
			label = val
		}

		n := util.Entry{
			Label:            label,
			Searchable:       v,
			Sub:              "Runner",
			Exec:             fmt.Sprintf("%s %s", label, strings.Join(fields[1:], " ")),
			Class:            "runner",
			History:          true,
			RecalculateScore: true,
			MatchFields:      1,
			Matching:         util.Fuzzy,
		}

		entries = append(entries, n)
	}

	bin := fields[0]

	if val, ok := r.aliases[bin]; ok {
		bin = val
	}

	if r.genericEntry {
		n := util.Entry{
			Label:            fmt.Sprintf("run: %s", term),
			Sub:              "Runner",
			Exec:             fmt.Sprintf("%s %s", bin, strings.Join(fields[1:], " ")),
			Class:            "runner",
			History:          true,
			RecalculateScore: true,
			MatchFields:      1,
			Matching:         util.Fuzzy,
		}

		entries = append(entries, n)
	}

	return entries
}

func (r *Runner) getBins() {
	path := os.Getenv("PATH")

	paths := strings.Split(path, ":")

	bins := []string{}

	for _, p := range paths {
		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if d != nil && d.IsDir() {
				return nil
			}

			exec, _ := exec.LookPath(filepath.Base(path))
			if exec == "" {
				return nil
			}

			bins = append(bins, filepath.Base(path))

			return nil
		})
	}

	for k := range r.aliases {
		bins = append(bins, k)
	}

	slices.Sort(bins)

	j := 0
	for i := 1; i < len(bins); i++ {
		if bins[j] == bins[i] {
			continue
		}
		j++
		bins[j] = bins[i]
	}

	bins = bins[:j+1]

	r.bins = bins
}

func (r *Runner) parseAliases() {
	if r.shellConfig == "" {
		return
	}

	r.aliases = make(map[string]string)

	file, err := os.Open(r.shellConfig)
	if err != nil {
		log.Println(err)
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "alias") {
			splits := strings.SplitN(text, "=", 2)
			alias := strings.TrimPrefix(splits[0], "alias ")

			r.aliases[alias] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "\""), "\"")
		}
	}
}
