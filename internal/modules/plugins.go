package modules

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Plugin struct {
	Config       config.Plugin
	cachedOutput []byte
	entries      []util.Entry
}

func (e *Plugin) General() *config.GeneralModule {
	return &e.Config.GeneralModule
}

func (e *Plugin) Refresh() {
	e.Config.IsSetup = !e.Config.Refresh
}

func (e *Plugin) Setup() bool {
	e.Config.Separator = util.TransformSeparator(e.Config.Separator)

	if e.Config.Parser == "" {
		e.Config.Parser = "json"
	}

	if e.Config.KvSeparator == "" {
		e.Config.KvSeparator = ";"
	}

	return true
}

func (e Plugin) Cleanup() {}

func (e *Plugin) SetupData() {
	if e.Config.Entries != nil {
		for k := range e.Config.Entries {
			e.Config.Entries[k].Sub = e.Config.Name
			e.Config.Entries[k].RecalculateScore = e.Config.RecalculateScore
		}
	}

	if e.Config.SrcOnce != "" {
		e.Config.Src = e.Config.SrcOnce

		e.cachedOutput = e.getSrcOutput(e.Config.SrcOnce, false, "")

		if e.Config.Cmd == "" {
			if e.Config.Parser == "json" {
				e.entries = e.parseJson(e.cachedOutput)
			} else if e.Config.Parser == "kv" {
				e.entries = e.parseKv(e.cachedOutput)
			}

			for k := range e.entries {
				e.entries[k].Class = e.Config.Name
			}
		}
	}

	e.Config.IsSetup = true
	e.Config.HasInitialSetup = true
}

func (e Plugin) Entries(term string) []util.Entry {
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

	src := e.Config.Src

	if strings.Contains(e.Config.Src, "%TERM%") {
		hasExplicitTerm = true
		src = strings.ReplaceAll(e.Config.Src, "%TERM%", term)
	}

	hasExplicitResult := false
	hasExplicitResultAlt := false

	if strings.Contains(e.Config.Cmd, "%RESULT%") {
		hasExplicitResult = true
	}

	if strings.Contains(e.Config.CmdAlt, "%RESULT%") {
		hasExplicitResultAlt = true
	}

	if e.Config.Output {
		out := e.cachedOutput

		var score float64

		e.Config.Keywords = append(e.Config.Keywords, e.Config.Name)

		for _, v := range e.Config.Keywords {
			res, _ := util.FuzzyScore(term, v)

			if res > score {
				score = res
			}
		}

		result := string(out)

		if result == "" {
			result = e.Config.OutputPlaceholder
		}

		if result == "" {
			result = "Running command..."
		}

		e := util.Entry{
			Label:            strings.TrimSpace(result),
			Exec:             strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result),
			ExecAlt:          strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result),
			Sub:              e.Config.Name,
			Output:           e.Config.Src,
			ScoreFinal:       score,
			RecalculateScore: false,
			Categories:       e.Config.Keywords,
			Matching:         util.AlwaysTop,
			Icon:             e.Config.Icon,
		}

		if !hasExplicitResult {
			e.Piped.String = result
			e.Piped.Type = "string"
		}

		if !hasExplicitResultAlt {
			e.PipedAlt.String = result
			e.PipedAlt.Type = "string"
		}

		return []util.Entry{e}
	}

	if e.Config.Cmd == "" {
		if len(e.entries) > 0 && e.Config.SrcOnce != "" {
			return e.entries
		}

		cmd := exec.Command("sh", "-c", wrapWithPrefix(src))

		if !hasExplicitTerm {
			cmd.Stdin = strings.NewReader(term)
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(err)
			return entries
		}

		if e.Config.Parser == "json" {
			entries = e.parseJson(out)
		} else if e.Config.Parser == "kv" {
			entries = e.parseKv(out)
		}

		for k := range entries {
			entries[k].Class = e.Config.Name
		}
	}

	if e.Config.Cmd != "" {
		var out []byte

		if e.cachedOutput != nil {
			out = e.cachedOutput
		} else {
			out = e.getSrcOutput(src, hasExplicitTerm, term)
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

	return entries
}

func (e Plugin) parseKv(out []byte) []util.Entry {
	var entries []util.Entry

	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		pairs := strings.Split(scanner.Text(), e.Config.KvSeparator)

		entry := util.Entry{}

		for _, v := range pairs {
			pair := strings.Split(v, "=")
			switch {
			case pair[0] == "path":
				entry.Path = pair[1]
			case pair[0] == "score_final":
				score, err := strconv.ParseFloat(pair[1], 64)
				if err != nil {
					log.Println(err)
					continue
				}

				entry.ScoreFinal = score
			case pair[0] == "score_fuzzy":
				score, err := strconv.ParseFloat(pair[1], 64)
				if err != nil {
					log.Println(err)
					continue
				}

				entry.ScoreFuzzy = score
			case pair[0] == "recalculate_score":
				entry.RecalculateScore, _ = strconv.ParseBool(pair[1])
			case pair[0] == "label":
				entry.Label = pair[1]
			case pair[0] == "sub":
				entry.Sub = pair[1]
			case pair[0] == "exec":
				entry.Exec = pair[1]
			case pair[0] == "image":
				entry.Image = pair[1]
			case pair[0] == "icon":
				entry.Icon = pair[1]
			case pair[0] == "exec_alt":
				entry.ExecAlt = pair[1]
			case pair[0] == "class":
				entry.Class = pair[1]
			case pair[0] == "initial_class":
				entry.InitialClass = pair[1]
			case pair[0] == "matching":
				mt, err := strconv.Atoi(pair[1])
				if err != nil {
					log.Println(err)
					continue
				}

				entry.Matching = util.MatchingType(mt)
			case pair[0] == "match_fields":
				entry.MatchFields, _ = strconv.Atoi(pair[1])
			case pair[0] == "searchable":
				entry.Searchable = pair[1]
			case pair[0] == "categories":
				entry.Categories = strings.Split(pair[1], ",")
			case pair[0] == "terminal":
				entry.Terminal, _ = strconv.ParseBool(pair[1])
			case pair[0] == "prefer":
				entry.Prefer, _ = strconv.ParseBool(pair[1])
			case pair[0] == "drag_drop":
				entry.DragDrop, _ = strconv.ParseBool(pair[1])
			case pair[0] == "drag_drop_data":
				entry.DragDropData = pair[1]
			case pair[0] == "hide_text":
				entry.HideText, _ = strconv.ParseBool(pair[1])
			case pair[0] == "value":
				entry.Value = pair[1]
			}
		}

		if entry.Label != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

func (e Plugin) parseJson(out []byte) []util.Entry {
	var entries []util.Entry

	err := json.Unmarshal(out, &entries)
	if err != nil {
		log.Println(err)
		return nil
	}

	return entries
}

func (e Plugin) getSrcOutput(src string, hasExplicitTerm bool, term string) []byte {
	cmd := exec.Command("sh", "-c", wrapWithPrefix(src))

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
