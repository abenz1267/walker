package modules

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Plugin struct {
	General      config.Plugin
	cachedOutput []byte
}

func (e Plugin) SwitcherOnly() bool {
	return e.General.SwitcherOnly
}

func (e *Plugin) Setup(cfg *config.Config) Workable {
	if e.General.SrcOnce != "" {
		e.General.Src = e.General.SrcOnce
		e.cachedOutput = e.getSrcOutput(false, "")
	}

	return e
}

func (e *Plugin) Refresh() {
	if e.General.SrcOnceRefresh {
		e.cachedOutput = e.getSrcOutput(false, "")
	}
}

func (e Plugin) Name() string {
	return e.General.Name
}

func (e Plugin) Prefix() string {
	return e.General.Prefix
}

func (e Plugin) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	if e.General.Src == "" {
		return entries
	}

	hasExplicitTerm := false
	hasExplicitResult := false
	hasExplicitResultAlt := false

	if strings.Contains(e.General.Src, "%TERM%") {
		hasExplicitTerm = true
		e.General.Src = strings.ReplaceAll(e.General.Src, "%TERM%", term)
	}

	if strings.Contains(e.General.Cmd, "%RESULT%") {
		hasExplicitResult = true
	}

	if strings.Contains(e.General.CmdAlt, "%RESULT%") {
		hasExplicitResultAlt = true
	}

	if e.General.Cmd != "" {
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
				Sub:      e.General.Name,
				Class:    e.General.Name,
				Terminal: e.General.Terminal,
				Exec:     strings.ReplaceAll(e.General.Cmd, "%RESULT%", txt),
				ExecAlt:  strings.ReplaceAll(e.General.CmdAlt, "%RESULT%", txt),
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

	cmd := exec.Command("sh", "-c", e.General.Src)

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
		entries[k].Class = e.General.Name
	}

	return entries
}

func (e Plugin) getSrcOutput(hasExplicitTerm bool, term string) []byte {
	cmd := exec.Command("sh", "-c", e.General.Src)

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
