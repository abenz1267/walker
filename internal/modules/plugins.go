package modules

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"log/slog"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Plugin struct {
	Config               config.Plugin
	cachedOutput         []byte
	entries              []*util.Entry
	hasExplicitResult    bool
	hasExplicitResultAlt bool
	hasExplicitTerm      bool
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
		e.Config.Parser = "raw"
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

	if strings.Contains(e.Config.Src, "%TERM%") {
		e.hasExplicitTerm = true
	}

	if strings.Contains(e.Config.Cmd, "%RESULT%") {
		e.hasExplicitResult = true
	}

	if strings.Contains(e.Config.CmdAlt, "%RESULT%") {
		e.hasExplicitResultAlt = true
	}

	if e.Config.SrcOnce != "" {
		e.cachedOutput = e.getSrcOutput(e.Config.SrcOnce, "")
		e.entries = e.parseOut(e.cachedOutput)
	}

	e.Config.IsSetup = true
	e.Config.HasInitialSetup = true
}

func (e Plugin) Entries(term string) []*util.Entry {
	if e.Config.Entries != nil {
		for k := range e.Config.Entries {
			e.Config.Entries[k].ScoreFinal = 0
			e.Config.Entries[k].ScoreFuzzy = 0
		}

		return e.Config.Entries
	}

	if e.Config.SrcOnce != "" && !e.Config.Output {
		return e.entries
	}

	entries := []*util.Entry{}

	if e.Config.Src == "" {
		return entries
	}

	src := e.Config.Src

	if e.hasExplicitTerm {
		src = strings.ReplaceAll(e.Config.Src, "%TERM%", term)
	}

	if e.Config.Output {
		out := e.cachedOutput

		var score float64
		var start int

		e.Config.Keywords = append(e.Config.Keywords, e.Config.Name)

		for _, v := range e.Config.Keywords {
			res, _, start := util.FuzzyScore(term, v)

			if res > score {
				score = res
				start = start
			}
		}

		result := string(out)

		if result == "" {
			result = e.Config.OutputPlaceholder
		}

		if result == "" {
			result = "Running command..."
		}

		entry := util.Entry{
			Label:            strings.TrimSpace(result),
			Exec:             strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result),
			ExecAlt:          strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result),
			Sub:              e.Config.Name,
			Output:           e.Config.Src,
			ScoreFinal:       score,
			MatchStartingPos: start,
			RecalculateScore: true,
			Categories:       e.Config.Keywords,
			Matching:         util.TopWhenFuzzyMatch,
			Icon:             e.Config.Icon,
		}

		if !e.hasExplicitResult {
			entry.Piped.String = result
			entry.Piped.Type = "string"
		}

		if !e.hasExplicitResultAlt {
			entry.PipedAlt.String = result
			entry.PipedAlt.Type = "string"
		}

		return []*util.Entry{&entry}
	}

	var out []byte

	if e.cachedOutput != nil {
		out = e.cachedOutput
	} else {
		out = e.getSrcOutput(src, term)
	}

	return e.parseOut(out)
}

func (e Plugin) parseRaw(out []byte) []*util.Entry {
	entries := []*util.Entry{}
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

		entry := util.Entry{
			Label:    label,
			Sub:      e.Config.Name,
			Class:    e.Config.Name,
			Terminal: e.Config.Terminal,
			Exec:     strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result),
			ExecAlt:  strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result),
			Matching: e.Config.Matching,
		}

		if !e.hasExplicitResult {
			entry.Piped.String = txt
			entry.Piped.Type = "string"
		}

		if !e.hasExplicitResultAlt {
			entry.PipedAlt.String = txt
			entry.PipedAlt.Type = "string"
		}

		entries = append(entries, &entry)
	}

	return entries
}

func (e Plugin) parseKv(out []byte) []*util.Entry {
	var entries []*util.Entry

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
			case pair[0] == "env":
				entry.Env = strings.Split(pair[1], ",")
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
			case pair[0] == "searchable2":
				entry.Searchable2 = pair[1]
			case pair[0] == "terminal_title_flag":
				entry.TerminalTitleFlag = pair[1]
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

		result := entry.Value

		if result == "" {
			result = entry.Label
		}

		if entry.Exec == "" && e.Config.Cmd != "" {
			entry.Exec = strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result)
		}

		if entry.ExecAlt == "" && e.Config.CmdAlt != "" {
			entry.ExecAlt = strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result)
		}

		if !e.hasExplicitResult {
			entry.Piped.String = result
			entry.Piped.Type = "string"
		}

		if !e.hasExplicitResultAlt {
			entry.PipedAlt.String = result
			entry.PipedAlt.Type = "string"
		}

		if entry.Label != "" {
			entries = append(entries, &entry)
		}
	}

	return entries
}

func (e Plugin) parseJson(out []byte) []*util.Entry {
	var entries []*util.Entry

	err := json.Unmarshal(out, &entries)
	if err != nil {
		log.Println(err)
		return nil
	}

	for k, v := range entries {
		result := v.Value

		if result == "" {
			result = v.Label
		}

		if v.Exec == "" && e.Config.Cmd != "" {
			entries[k].Exec = strings.ReplaceAll(e.Config.Cmd, "%RESULT%", result)
		}

		if v.ExecAlt == "" && e.Config.CmdAlt != "" {
			entries[k].ExecAlt = strings.ReplaceAll(e.Config.CmdAlt, "%RESULT%", result)
		}

		if !e.hasExplicitResult {
			entries[k].Piped.String = result
			entries[k].Piped.Type = "string"
		}

		if !e.hasExplicitResultAlt {
			entries[k].PipedAlt.String = result
			entries[k].PipedAlt.Type = "string"
		}
	}

	return entries
}

func (e Plugin) getSrcOutput(src, term string) []byte {
	cmd := exec.Command("sh", "-c", src)

	if !e.hasExplicitTerm && term != "" {
		cmd.Stdin = strings.NewReader(term)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("error", "plugins", err, "message", string(out))
		return nil
	}

	return out
}

func (e Plugin) parseOut(out []byte) []*util.Entry {
	entries := []*util.Entry{}

	if e.Config.Parser == "json" {
		entries = e.parseJson(out)
	} else if e.Config.Parser == "kv" {
		entries = e.parseKv(out)
	} else if e.Config.Parser == "raw" {
		entries = e.parseRaw(out)
	}

	for k := range entries {
		entries[k].Class = e.Config.Name
	}

	return entries
}
