package processors

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"strings"
)

type External struct {
	Prfx    string
	Nme     string
	Src     string
	Cmd     string
	History bool
}

func (e External) Name() string {
	return e.Nme
}

func (e External) Prefix() string {
	return e.Prfx
}

func (e *External) SetPrefix(val string) {
	e.Prfx = val
}

func (e External) Entries(term string) []Entry {
	entries := []Entry{}

	if e.Cmd == "" {
		return entries
	}

	if e.Prfx != "" && len(term) < 2 {
		return entries
	}

	if e.Prfx != "" {
		term = strings.TrimPrefix(term, e.Prfx)
	}

	if e.Src != "" {
		e.Src = strings.ReplaceAll(e.Src, "%TERM%", term)

		fields := strings.Fields(e.Src)

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
					Label:      txt,
					Sub:        e.Nme,
					Class:      e.Nme,
					Exec:       strings.ReplaceAll(e.Cmd, "%RESULT%", txt),
					Searchable: txt,
					History:    e.History,
				}

				entries = append(entries, e)
			}
		}

		return entries
	}

	fields := strings.Fields(e.Cmd)
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
		v.Class = e.Nme
		v.Sub = e.Nme
	}

	return entries
}
