package modules

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type External struct {
	prefix            string
	ModuleName        string
	src               string
	cmd               string
	cmdAlt            string
	cachedOutput      []byte
	refresh           bool
	switcherExclusive bool
	recalculateScore  bool
	terminal          bool
}

func (e External) SwitcherExclusive() bool {
	return e.switcherExclusive
}

func (e *External) Setup(cfg *config.Config, module *config.Module) Workable {
	e.prefix = module.Prefix
	e.switcherExclusive = module.SwitcherExclusive
	e.src = os.ExpandEnv(module.Src)
	e.cmd = os.ExpandEnv(module.Cmd)
	e.cmdAlt = os.ExpandEnv(module.CmdAlt)
	e.terminal = module.Terminal

	if module.SrcOnce != "" {
		e.src = os.ExpandEnv(module.SrcOnce)
		e.cachedOutput = e.getSrcOutput(false, "")
	}

	e.refresh = module.SrcOnceRefresh

	return e
}

func (e *External) Refresh() {
	if e.refresh {
		e.cachedOutput = e.getSrcOutput(false, "")
	}
}

func (e External) Name() string {
	return e.ModuleName
}

func (e External) Prefix() string {
	return e.prefix
}

func (e External) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	if e.src == "" {
		return entries
	}

	if e.prefix != "" && len(term) == 1 {
		return entries
	}

	e.src = strings.ReplaceAll(e.src, "~", os.Getenv("HOME"))
	e.cmd = strings.ReplaceAll(e.cmd, "~", os.Getenv("HOME"))
	e.cmdAlt = strings.ReplaceAll(e.cmdAlt, "~", os.Getenv("HOME"))

	if e.prefix != "" {
		term = strings.TrimPrefix(term, e.prefix)
	}

	hasExplicitTerm := false
	hasExplicitResult := false
	hasExplicitResultAlt := false

	if strings.Contains(e.src, "%TERM%") {
		hasExplicitTerm = true
		e.src = strings.ReplaceAll(e.src, "%TERM%", term)
	}

	if strings.Contains(e.cmd, "%RESULT%") {
		hasExplicitResult = true
	}

	if strings.Contains(e.cmdAlt, "%RESULT%") {
		hasExplicitResultAlt = true
	}

	if e.cmd != "" {
		var out []byte

		if e.cachedOutput != nil {
			out = e.cachedOutput
		} else {
			out = e.getSrcOutput(hasExplicitTerm, term)
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))

		for scanner.Scan() {
			txt := scanner.Text()

			unescaped, err := url.QueryUnescape(txt)
			if err != nil {
				log.Println(err)
				continue
			}

			e := Entry{
				Label:    unescaped,
				Sub:      e.ModuleName,
				Class:    e.ModuleName,
				Terminal: e.terminal,
				Exec:     strings.ReplaceAll(e.cmd, "%RESULT%", txt),
				ExecAlt:  strings.ReplaceAll(e.cmdAlt, "%RESULT%", txt),
			}

			if !hasExplicitResult {
				e.Piped.Content = txt
				e.Piped.Type = "string"
			}

			if !hasExplicitResultAlt {
				e.PipedAlt.Content = txt
				e.PipedAlt.Type = "string"
			}

			entries = append(entries, e)
		}

		return entries
	}

	name, args := util.ParseShellCommand(e.src)

	cmd := exec.Command(name, args...)

	if !hasExplicitTerm {
		cmd.Stdin = strings.NewReader(term)
	}

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

	for k := range entries {
		entries[k].Class = e.ModuleName
	}

	return entries
}

func (e External) getSrcOutput(hasExplicitTerm bool, term string) []byte {
	name, args := util.ParseShellCommand(e.src)
	cmd := exec.Command(name, args...)

	if !hasExplicitTerm && term != "" {
		cmd.Stdin = strings.NewReader(term)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return nil
	}

	return out
}
