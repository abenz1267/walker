package modules

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Plugin struct {
	PluginCfg    config.Plugin
	isSetup      bool
	cachedOutput []byte
}

func (e *Plugin) General() *config.GeneralModule {
	return &e.PluginCfg.GeneralModule
}

func (e *Plugin) Refresh() {
	e.PluginCfg.IsSetup = !e.PluginCfg.Refresh
}

func (e *Plugin) Setup(cfg *config.Config) bool {
	return true
}

func (e Plugin) Cleanup() {}

func (e *Plugin) SetupData(cfg *config.Config, ctx context.Context) {
	if e.PluginCfg.Entries != nil {
		for k := range e.PluginCfg.Entries {
			e.PluginCfg.Entries[k].Sub = e.PluginCfg.Name
			e.PluginCfg.Entries[k].RecalculateScore = e.PluginCfg.RecalculateScore
		}
	}

	if e.PluginCfg.SrcOnce != "" {
		e.PluginCfg.Src = e.PluginCfg.SrcOnce
		e.cachedOutput = e.getSrcOutput(false, "")
	}

	e.isSetup = true
	e.PluginCfg.HasInitialSetup = true
}

func (e Plugin) Entries(ctx context.Context, term string) []util.Entry {
	if e.PluginCfg.Entries != nil {
		for k := range e.PluginCfg.Entries {
			e.PluginCfg.Entries[k].ScoreFinal = 0
			e.PluginCfg.Entries[k].ScoreFuzzy = 0
		}

		return e.PluginCfg.Entries
	}

	entries := []util.Entry{}

	if e.PluginCfg.Src == "" {
		return entries
	}

	hasExplicitTerm := false
	hasExplicitResult := false
	hasExplicitResultAlt := false

	if strings.Contains(e.PluginCfg.Src, "%TERM%") {
		hasExplicitTerm = true
		e.PluginCfg.Src = strings.ReplaceAll(e.PluginCfg.Src, "%TERM%", term)
	}

	if strings.Contains(e.PluginCfg.Cmd, "%RESULT%") {
		hasExplicitResult = true
	}

	if strings.Contains(e.PluginCfg.CmdAlt, "%RESULT%") {
		hasExplicitResultAlt = true
	}

	if e.PluginCfg.Cmd != "" {
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

			e := util.Entry{
				Label:    unescaped,
				Sub:      e.PluginCfg.Name,
				Class:    e.PluginCfg.Name,
				Terminal: e.PluginCfg.Terminal,
				Exec:     strings.ReplaceAll(e.PluginCfg.Cmd, "%RESULT%", txt),
				ExecAlt:  strings.ReplaceAll(e.PluginCfg.CmdAlt, "%RESULT%", txt),
				Matching: e.PluginCfg.Matching,
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

	cmd := exec.Command("sh", "-c", e.PluginCfg.Src)

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
		entries[k].Class = e.PluginCfg.Name
	}

	return entries
}

func (e Plugin) getSrcOutput(hasExplicitTerm bool, term string) []byte {
	cmd := exec.Command("sh", "-c", e.PluginCfg.Src)

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
