package modules

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Runner struct {
	config  config.Runner
	aliases map[string]string
	bins    []string
}

func (r Runner) Cleanup() {}

func (r *Runner) General() *config.GeneralModule {
	return &r.config.GeneralModule
}

func (r *Runner) Setup(cfg *config.Config) bool {
	r.config = cfg.Builtins.Runner

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

	r.config.IsSetup = true
	r.config.HasInitialSetup = true
}

func (r *Runner) Refresh() {
	r.config.IsSetup = !r.config.Refresh
}

func (r Runner) Entries(term string) []util.Entry {
	entries := []util.Entry{}

	fields := strings.Fields(term)

	for _, v := range r.bins {
		bin := v

		if val, ok := r.aliases[v]; ok {
			bin = val
		}

		exec := term

		if len(fields) > 0 {
			exec = fmt.Sprintf("%s %s", bin, strings.Join(fields[1:], " "))
		}

		n := util.Entry{
			Label:            bin,
			Searchable:       v,
			Sub:              "Runner",
			Exec:             exec,
			Class:            "runner",
			History:          true,
			RecalculateScore: true,
			MatchFields:      1,
			Matching:         util.Fuzzy,
		}

		entries = append(entries, n)
	}

	exec := term

	if len(fields) > 0 {
		bin := fields[0]

		if val, ok := r.aliases[bin]; ok {
			bin = val
		}

		exec = fmt.Sprintf("%s %s", bin, strings.Join(fields[1:], " "))
	}

	if r.config.GenericEntry {
		n := util.Entry{
			Label:            fmt.Sprintf("run: %s", term),
			Sub:              "Runner",
			Exec:             exec,
			Class:            "runner",
			History:          true,
			RecalculateScore: true,
			MatchFields:      1,
			Matching:         util.Fuzzy,
			Terminal:         true,
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
	if r.config.ShellConfig == "" {
		return
	}

	r.aliases = make(map[string]string)

	file, err := os.Open(r.config.ShellConfig)
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
