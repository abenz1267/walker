package modules

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Runner struct {
	ShellConfig       string
	prefix            string
	aliases           map[string]string
	switcherExclusive bool
}

func (r Runner) SwitcherExclusive() bool {
	return r.switcherExclusive
}

func (r Runner) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, r.Name())
	if module == nil {
		return nil
	}

	r.prefix = module.Prefix
	r.switcherExclusive = module.SwitcherExclusive

	return r
}

func (r Runner) Prefix() string {
	return r.prefix
}

func (Runner) Name() string {
	return "runner"
}

func (r Runner) Entries(term string) []Entry {
	entries := []Entry{}

	if term == "" {
		return entries
	}

	r.parseAliases()

	if r.prefix != "" && len(term) < 2 {
		return entries
	}

	if r.prefix != "" {
		term = strings.TrimPrefix(term, r.prefix)
	}

	fields := strings.Fields(term)

	alias, ok := r.aliases[fields[0]]
	if ok {
		term = strings.Replace(term, fields[0], alias, 1)
		fields = strings.Fields(term)
	}

	str, err := exec.LookPath(fields[0])
	if err != nil {
		return entries
	}

	n := Entry{
		Label:      str,
		Sub:        "Runner",
		Exec:       term,
		Notifyable: true,
		Class:      "runner",
		Matching:   AlwaysTop,
	}

	entries = append(entries, n)

	return entries
}

func (r *Runner) parseAliases() {
	if r.ShellConfig == "" {
		return
	}

	r.aliases = make(map[string]string)

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
			r.aliases[aliasFields[1]] = strings.TrimSuffix(strings.TrimPrefix(splits[1], "\""), "\"")
		}
	}
}
