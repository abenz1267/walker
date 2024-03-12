package processors

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Runner struct {
	Prfx        string
	ShellConfig string
	Aliases     map[string]string
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

func (r *Runner) Entries(term string) []Entry {
	r.parseAliases()

	entries := []Entry{}

	if r.Prfx != "" && len(term) < 2 {
		return entries
	}

	if r.Prfx != "" {
		term = strings.TrimPrefix(term, r.Prfx)
	}

	fields := strings.Fields(term)

	alias, ok := r.Aliases[fields[0]]
	if ok {
		term = strings.Replace(term, fields[0], alias, 1)
		fields = strings.Fields(term)
	}

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
		Notifyable: true,
		Class:      "runner",
	}

	entries = append(entries, n)

	return entries
}

func (r *Runner) parseAliases() {
	if r.ShellConfig == "" {
		return
	}

	r.Aliases = make(map[string]string)

	file, err := os.Open(r.ShellConfig)
	if err != nil {
		log.Println(err)
		return
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "alias") {
			splits := strings.Split(text, "=")
			aliasFields := strings.Fields(splits[0])
			r.Aliases[aliasFields[1]] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "\""), "\"")
		}
	}
}
