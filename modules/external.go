package modules

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
)

type External struct {
	prefix     string
	ModuleName string
	src        string
	cmd        string
}

func (e External) Setup(cfg *config.Config) Workable {
	module := find(cfg.External, e.Name())
	if module == nil {
		return nil
	}

	e.prefix = module.Prefix
	e.src = module.Src
	e.cmd = module.Cmd

	return e
}

func (e External) Name() string {
	return e.ModuleName
}

func (e External) Prefix() string {
	return e.prefix
}

func (e External) Entries(term string) []Entry {
	entries := []Entry{}

	if e.cmd == "" {
		return entries
	}

	if e.prefix != "" && len(term) == 1 {
		return entries
	}

	if e.prefix != "" {
		term = strings.TrimPrefix(term, e.prefix)
	}

	if e.src != "" {
		e.src = strings.ReplaceAll(e.src, "%TERM%", term)

		fields := strings.Fields(e.src)

		cmd := exec.Command(fields[0], fields[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(err)
			return entries
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))

		for scanner.Scan() {
			for scanner.Scan() {
				txt := scanner.Text()

				e := Entry{
					Label: txt,
					Sub:   e.ModuleName,
					Class: e.ModuleName,
					Exec:  strings.ReplaceAll(e.cmd, "%RESULT%", txt),
				}

				entries = append(entries, e)
			}
		}

		return entries
	}

	fields := strings.Fields(e.cmd)
	fields = append(fields, term)

	cmd := exec.Command(fields[0], fields[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return entries
	}

	err = json.Unmarshal(out, &entries)
	if err != nil {
		log.Println(err)
		return entries
	}

	for _, v := range entries {
		v.Class = e.ModuleName
		v.Sub = e.ModuleName
	}

	return entries
}
