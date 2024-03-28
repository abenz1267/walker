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
	ShellConfig       string
	prefix            string
	aliases           map[string]string
	bins              []string
	switcherExclusive bool
}

func (r Runner) SwitcherExclusive() bool {
	return r.switcherExclusive
}

func (r Runner) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, r.Name())
	if module == nil {
		return nil
	}

	r.prefix = module.Prefix
	r.switcherExclusive = module.SwitcherExclusive

	r.parseAliases()

	if len(cfg.Runner.Includes) > 0 {
		r.bins = cfg.Runner.Includes
	} else {
		r.getBins()
	}

	filtered := []string{}

	if len(cfg.Runner.Excludes) > 0 {
		for _, v := range r.bins {
			if !slices.Contains(cfg.Runner.Excludes, v) {
				filtered = append(filtered, v)
			}
		}

		r.bins = filtered
	}

	return r
}

func (r Runner) Prefix() string {
	return r.prefix
}

func (Runner) Name() string {
	return "runner"
}

func (r Runner) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	if term == "" {
		return entries
	}

	if r.prefix != "" && len(term) < 2 {
		return entries
	}

	if r.prefix != "" {
		term = strings.TrimPrefix(term, r.prefix)
	}

	fields := strings.Fields(term)
	matchable := fields[0]

	for _, v := range r.bins {
		label := v

		if val, ok := r.aliases[v]; ok {
			label = val
		}

		n := Entry{
			Label:      label,
			Searchable: v,
			Sub:        "Runner",
			Exec:       fmt.Sprintf("%s %s", label, strings.Join(fields[1:], " ")),
			Class:      "runner",
			History:    true,
		}

		rank := util.FuzzyScore(matchable, v)
		n.ScoreFinal = float64(rank)

		if rank < 20 {
			continue
		}

		entries = append(entries, n)
	}

	slices.SortFunc(entries, func(a, b Entry) int {
		if a.ScoreFinal > b.ScoreFinal {
			return -1
		}

		if a.ScoreFinal < b.ScoreFinal {
			return 1
		}

		return 0
	})

	if len(entries) > 5 {
		return entries[0:5]
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
	if r.ShellConfig == "" {
		return
	}

	r.aliases = make(map[string]string)

	file, err := os.Open(r.ShellConfig)
	if err != nil {
		log.Println(err)
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "alias") {
			splits := strings.Split(text, "=")
			aliasFields := strings.Fields(splits[0])
			r.aliases[aliasFields[1]] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "\""), "\"")
		}
	}
}
