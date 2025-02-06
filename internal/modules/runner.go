package modules

import (
	"bufio"
	"bytes"
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

func (r *Runner) Setup() bool {
	r.config = config.Cfg.Builtins.Runner

	return true
}

func (r *Runner) SetupData() {
	r.parseAliases()

	if len(config.Cfg.Builtins.Runner.Includes) > 0 {
		r.bins = config.Cfg.Builtins.Runner.Includes
	} else {
		r.getBins()
	}

	filtered := []string{}

	if len(config.Cfg.Builtins.Runner.Excludes) > 0 {
		for _, v := range r.bins {
			if !slices.Contains(config.Cfg.Builtins.Runner.Excludes, v) {
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

	toRun := term

	if len(fields) > 0 {
		bin := fields[0]

		if val, ok := r.aliases[bin]; ok {
			bin = val
		}

		toRun = fmt.Sprintf("%s %s", bin, strings.Join(fields[1:], " "))
	}

	if strings.HasPrefix(term, "'") {
		term = strings.TrimPrefix(term, "'")
	}

	if term != "" {
		if path, _ := exec.LookPath(fields[0]); path != "" {
			if r.config.GenericEntry {
				n := util.Entry{
					Label:            fmt.Sprintf("run: %s", term),
					Sub:              "Runner",
					Exec:             toRun,
					Class:            "runner",
					History:          true,
					RecalculateScore: true,
					MatchFields:      1,
					Matching:         util.Fuzzy,
					Terminal:         true,
				}

				entries = append(entries, n)
			}
		}
	}

	return entries
}

func (r *Runner) getBins() {
	path := os.Getenv("PATH")

	paths := strings.Split(path, ":")

	bins := []string{}

	if config.Cfg.Builtins.Runner.UseFD {
		args := []string{".", "--no-ignore-vcs", "--type", "executable", "--type", "symlink"}
		args = append(args, paths...)

		cmd := exec.Command("fd", args...)

		out, err := cmd.CombinedOutput()
		if err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))

			for scanner.Scan() {
				info, err := os.Stat(scanner.Text())
				if info == nil || err != nil {
					continue
				}

				if info.Mode()&0111 != 0 {
					bins = append(bins, filepath.Base(scanner.Text()))
				}
			}
		}
	} else {
		for _, p := range paths {
			filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
				if d != nil && d.IsDir() {
					return nil
				}

				info, err := os.Stat(path)
				if info == nil {
					return nil
				}

				if info.Mode()&0111 != 0 {
					bins = append(bins, filepath.Base(path))
				}

				return nil
			})
		}
	}

	for k := range r.aliases {
		bins = append(bins, k)
	}

	slices.Sort(bins)

	bins = slices.Compact(bins)

	r.bins = bins
}

func (r *Runner) parseAliases() {
	if r.config.ShellConfig == "" {
		return
	}

	r.aliases = make(map[string]string)

	r.parseAliasesFunc(r.config.ShellConfig)
}

func (r *Runner) parseAliasesFunc(src string) {
	homeDir, _ := os.UserHomeDir()

	file, err := os.Open(src)
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

			if strings.HasPrefix(splits[1], "\"") {
				r.aliases[alias] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "\""), "\"")
			} else if strings.HasPrefix(splits[1], "'") {
				r.aliases[alias] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "'"), "'")
			}
		}

		if strings.HasPrefix(text, "source") {
			file := strings.Split(text, " ")[1]
			file = strings.Replace(file, "~", homeDir, 1)

			r.parseAliasesFunc(file)
		}
	}
}
