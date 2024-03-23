package modules

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/config"
)

type SSH struct {
	prefix            string
	switcherExclusive bool
	entries           []Entry
}

func (s SSH) Entries(term string) []Entry {
	fields := strings.Fields(term)

	cmd := "ssh"

	for k, v := range s.entries {
		if len(fields) > 1 {
			cmd = fmt.Sprintf("ssh %s@%s", fields[1], v.Label)
		}

		s.entries[k].Exec = cmd
	}

	fmt.Println(s.entries)

	return s.entries
}

func (s SSH) Prefix() string {
	return s.prefix
}

func (s SSH) Name() string {
	return "ssh"
}

func (s SSH) SwitcherExclusive() bool {
	return s.switcherExclusive
}

func (SSH) Setup(cfg *config.Config) Workable {
	s := &SSH{}

	module := Find(cfg.Modules, s.Name())
	if module == nil {
		return nil
	}

	s.prefix = module.Prefix
	s.switcherExclusive = module.SwitcherExclusive

	home, err := os.UserHomeDir()
	if err != nil {
		log.Panicln(err)
	}

	hosts := filepath.Join(home, ".ssh", "known_hosts")

	file, err := os.Open(hosts)
	if err != nil {
		log.Panicln(err)
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	hs := make(map[string]struct{})

	for scanner.Scan() {
		host := strings.Fields(scanner.Text())[0]
		hs[host] = struct{}{}
	}

	entries := []Entry{}

	for k := range hs {
		entries = append(entries, Entry{
			Label:            k,
			Sub:              "SSH",
			Exec:             "ssh",
			MatchFields:      1,
			Searchable:       k,
			RawExec:          []string{},
			Terminal:         true,
			Categories:       []string{"ssh"},
			Class:            "ssh",
			Matching:         Fuzzy,
			RecalculateScore: true,
		})
	}

	s.entries = entries

	return s
}
