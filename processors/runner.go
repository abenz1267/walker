package processors

import (
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct {
	Prfx string
}

func (r *Runner) SetPrefix(val string) {
	r.Prfx = val
}

func (r Runner) Prefix() string {
	return r.Prfx
}

func (Runner) Name() string {
	return "runner"
}

func (r Runner) Entries(term string) []Entry {
	entries := []Entry{}

	if r.Prfx != "" && len(term) < 2 {
		return entries
	}

	if r.Prfx != "" {
		term = strings.TrimPrefix(term, r.Prfx)
	}

	fields := strings.Fields(term)
	str, err := exec.LookPath(fields[0])
	if err != nil {
		str = fmt.Sprintf("%s not found in $PATH", fields[0])
	}

	n := Entry{
		Label:      str,
		Sub:        "Runner",
		Img:        "",
		Exec:       term,
		Searchable: term,
	}

	entries = append(entries, n)

	return entries
}
