package processors

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"
)

type External struct {
	Prfx string
	Nme  string
	Cmd  string
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

	return entries
}
