package modules

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type SSH struct {
	config  config.SSH
	entries []*util.Entry
}

func (s *SSH) General() *config.GeneralModule {
	return &s.config.GeneralModule
}

func (s SSH) Cleanup() {}

func (s *SSH) Refresh() {
	s.config.IsSetup = !s.config.Refresh
}

func (s SSH) Entries(term string) []*util.Entry {
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

func (s *SSH) Setup() bool {
	s.config = config.Cfg.Builtins.SSH

	return true
}

func (s *SSH) SetupData() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Panicln(err)
		return
	}

	sshCfg := filepath.Join(home, ".ssh", "config")
	if config.Cfg.Builtins.SSH.ConfigFile != "" {
		sshCfg = config.Cfg.Builtins.SSH.ConfigFile
	}

	hosts := filepath.Join(home, ".ssh", "known_hosts")
	if config.Cfg.Builtins.SSH.HostFile != "" {
		hosts = config.Cfg.Builtins.SSH.HostFile
	}

	s.entries = append(s.entries, getHostFileEntries(hosts)...)
	s.entries = append(s.entries, getConfigFileEntries(sshCfg)...)

	s.config.IsSetup = true
	s.config.HasInitialSetup = true
}

func getConfigFileEntries(sshCfg string) []*util.Entry {
	entries := []*util.Entry{}

	file, err := os.Open(sshCfg)
	if err != nil {
		return []*util.Entry{}
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "Host ") || strings.HasPrefix(text, "host ") {
			fields := strings.Fields(text)

			entries = append(entries, &util.Entry{
				Label:            fields[1],
				Sub:              "SSH Config",
				Exec:             fmt.Sprintf("ssh %s", fields[1]),
				Searchable:       fields[1],
				Terminal:         true,
				Categories:       []string{"ssh"},
				Class:            "ssh",
				Matching:         util.Fuzzy,
				RecalculateScore: true,
			})
		}
	}

	return entries
}

func getHostFileEntries(hosts string) []*util.Entry {
	file, err := os.Open(hosts)
	if err != nil {
		return []*util.Entry{}
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	hs := make(map[string]struct{})

	for scanner.Scan() {
		host := strings.Fields(scanner.Text())[0]
		hs[host] = struct{}{}
	}

	entries := []*util.Entry{}

	for k := range hs {
		entries = append(entries, &util.Entry{
			Label:            k,
			Sub:              "SSH Host",
			Exec:             "ssh",
			MatchFields:      1,
			Searchable:       k,
			Terminal:         true,
			Categories:       []string{"ssh"},
			Class:            "ssh",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	return entries
}
