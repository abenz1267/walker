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
	general config.GeneralModule
	entries []Entry
}

func (s SSH) IsSetup() bool {
	return s.general.IsSetup
}

func (SSH) KeepSort() bool {
	return false
}

func (s SSH) Placeholder() string {
	if s.general.Placeholder == "" {
		return "ssh"
	}

	return s.general.Placeholder
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
	return s.general.Prefix
}

func (s SSH) Name() string {
	return "ssh"
}

func (s SSH) SwitcherOnly() bool {
	return s.general.SwitcherOnly
}

func (s *SSH) Setup(cfg *config.Config) {
	s.general.Prefix = cfg.Builtins.SSH.Prefix
	s.general.SwitcherOnly = cfg.Builtins.SSH.SwitcherOnly
	s.general.SpecialLabel = cfg.Builtins.SSH.SpecialLabel
}

func (s *SSH) SetupData(cfg *config.Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panicln(err)
		return
	}

	hosts := filepath.Join(home, ".ssh", "known_hosts")
	if cfg.Builtins.SSH.HostFile != "" {
		hosts = cfg.Builtins.SSH.HostFile
	}

	if _, err := os.Stat(hosts); err != nil {
		log.Println("SSH host file not found, disabling ssh module")
		return
	}

	file, err := os.Open(hosts)
	if err != nil {
		log.Panicln(err)
		return
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

	s.general.IsSetup = true
}
