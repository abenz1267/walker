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

func (s *SSH) Refresh() {
	s.general.IsSetup = false
}

func (s SSH) Entries(ctx context.Context, term string) []Entry {
	fields := strings.Fields(term)

	cmd := "ssh"

	for k, v := range s.entries {
		if v.Sub == "SSH Host" {
			if len(fields) > 1 {
				cmd = fmt.Sprintf("ssh %s@%s", fields[1], v.Label)
			}

			s.entries[k].Exec = cmd
		}
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

	sshCfg := filepath.Join(home, ".ssh", "config")
	if cfg.Builtins.SSH.ConfigFile != "" {
		sshCfg = cfg.Builtins.SSH.ConfigFile
	}

	hosts := filepath.Join(home, ".ssh", "known_hosts")
	if cfg.Builtins.SSH.HostFile != "" {
		hosts = cfg.Builtins.SSH.HostFile
	}

	s.entries = append(s.entries, getHostFileEntries(hosts)...)
	s.entries = append(s.entries, getConfigFileEntries(sshCfg)...)

	s.general.IsSetup = true
}

func getConfigFileEntries(sshCfg string) []Entry {
	entries := []Entry{}

	file, err := os.Open(sshCfg)
	if err != nil {
		return []Entry{}
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Host ") || strings.HasPrefix(text, "host ") {
			fields := strings.Fields(text)

			entries = append(entries, Entry{
				Label:            fields[1],
				Sub:              "SSH Config",
				Exec:             fmt.Sprintf("ssh %s", fields[1]),
				Searchable:       fields[1],
				Terminal:         true,
				Categories:       []string{"ssh"},
				Class:            "ssh",
				Matching:         Fuzzy,
				RecalculateScore: true,
			})
		}
	}

	return entries
}

func getHostFileEntries(hosts string) []Entry {
	file, err := os.Open(hosts)
	if err != nil {
		return []Entry{}
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
			Sub:              "SSH Host",
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

	return entries
}
