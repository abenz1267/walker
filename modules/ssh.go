package modules

import (
	"bufio"
	"context"
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

func (s SSH) Refresh() {}

func (s SSH) Entries(ctx context.Context, term string) []Entry {
	fields := strings.Fields(term)

	cmd := "ssh"

	for k, v := range s.entries {
		if len(fields) > 1 {
			cmd = fmt.Sprintf("ssh %s@%s", fields[1], v.Label)
		}

		s.entries[k].Exec = cmd
	}

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

func (SSH) Setup(cfg *config.Config, module *config.Module) Workable {
	s := &SSH{}

	s.prefix = module.Prefix
	s.switcherExclusive = module.SwitcherExclusive

	home, err := os.UserHomeDir()
	if err != nil {
		log.Panicln(err)
	}

	hosts := filepath.Join(home, ".ssh", "known_hosts")
	if cfg.SSHHostFile != "" {
		hosts = cfg.SSHHostFile
	}

	if _, err := os.Stat(hosts); err != nil {
		log.Println("SSH host file not found, disabling ssh module")
		return nil
	}

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
