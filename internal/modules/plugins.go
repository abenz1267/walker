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
	Config       config.Plugin
	cachedOutput []byte
}

func (e *Plugin) General() *config.GeneralModule {
	return &e.Config.GeneralModule
}

func (e *Plugin) Refresh() {
	e.Config.IsSetup = !e.Config.Refresh
}

func (e *Plugin) Setup(cfg *config.Config) bool {
	e.Config.Separator = util.TrasformSeparator(e.Config.Separator)

	return true
}

func (e Plugin) Cleanup() {}

func (e *Plugin) SetupData(cfg *config.Config, ctx context.Context) {
	if e.Config.Entries != nil {
		for k := range e.Config.Entries {
			e.Config.Entries[k].Sub = e.Config.Name
			e.Config.Entries[k].RecalculateScore = e.Config.RecalculateScore
		}
	}

	if e.Config.SrcOnce != "" {
		e.Config.Src = e.Config.SrcOnce
		e.cachedOutput = e.getSrcOutput(false, "")
	}

	e.Config.IsSetup = true
	e.Config.HasInitialSetup = true
}

func (e Plugin) Entries(ctx context.Context, term string) []util.Entry {
	if e.Config.Entries != nil {
		for k := range e.Config.Entries {
			e.Config.Entries[k].ScoreFinal = 0
			e.Config.Entries[k].ScoreFuzzy = 0
		}

		return e.Config.Entries
	}

	entries := []util.Entry{}

	if e.Config.Src == "" {
		return entries
	}

	hasExplicitTerm := false
	hasExplicitResult := false
	hasExplicitResultAlt := false

	if strings.Contains(e.Config.Src, "%TERM%") {
		hasExplicitTerm = true
		e.Config.Src = strings.ReplaceAll(e.Config.Src, "%TERM%", term)
	}

	if strings.Contains(e.Config.Cmd, "%RESULT%") {
		hasExplicitResult = true
	}

	if strings.Contains(e.Config.CmdAlt, "%RESULT%") {
		hasExplicitResultAlt = true
	}

	if e.Config.Cmd != "" {
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

			result := txt
			label := unescaped

			if e.Config.ResultColumn > 0 || e.Config.LabelColumn > 0 {
				cols := strings.Split(txt, e.Config.Separator)

				if e.Config.ResultColumn > 0 {
					if len(cols) < e.Config.ResultColumn {
						continue
					}

					result = cols[e.Config.ResultColumn-1]
				}

				if e.Config.LabelColumn > 0 {
					if len(cols) < e.Config.LabelColumn {
						continue
					}

					label = cols[e.Config.LabelColumn-1]
				}
			}

			e := util.Entry{
				Label:    label,
				Sub:      e.Config.Name,
				Class:    e.Config.Name,
				Terminal: e.Config.Terminal,
				Exec:     strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result),
				ExecAlt:  strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result),
				Matching: e.Config.Matching,
			}

			if !hasExplicitResult {
				e.Piped.String = txt
				e.Piped.Type = "string"
			}

			if !hasExplicitResultAlt {
				e.PipedAlt.String = txt
				e.PipedAlt.Type = "string"
			}

			entries = append(entries, e)
		}

		return entries
	}

	cmd := exec.Command("sh", "-c", e.Config.Src)

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
		entries[k].Class = e.Config.Name
	}

	return entries
}

func (e Plugin) getSrcOutput(hasExplicitTerm bool, term string) []byte {
	cmd := exec.Command("sh", "-c", e.Config.Src)

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
