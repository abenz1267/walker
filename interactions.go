package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/walker/processors"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

type KeyPressHandler func(uint, uint, gdk.ModifierType) bool

type Processor interface {
	Entries(term string) []processors.Entry
	Prefix() string
	SetPrefix(val string)
	Name() string
}

func setupInteractions() {
	internal := make(map[string]Processor)
	internal["applications"] = processors.GetApplications()
	internal["runner"] = &processors.Runner{ShellConfig: config.ShellConfig}
	internal["websearch"] = &processors.Websearch{}

	enabled := []Processor{}

	for _, v := range config.Processors {
		if val, ok := internal[v.Name]; ok {
			val.SetPrefix(v.Prefix)
			enabled = append(enabled, val)
			continue
		}

		enabled = append(enabled, &processors.External{
			Prfx:    v.Prefix,
			Nme:     v.Name,
			Src:     v.Src,
			Cmd:     v.Cmd,
			History: v.History,
		})
	}

	for _, v := range enabled {
		ui.prefixClasses[v.Prefix()] = append(ui.prefixClasses[v.Prefix()], v.Name())
	}

	procs = make(map[string][]Processor)

	for _, v := range enabled {
		procs[v.Prefix()] = append(procs[v.Prefix()], v)
	}

	keycontroller := gtk.NewEventControllerKey()
	keycontroller.ConnectKeyPressed(handleKeys())

	ui.search.AddController(keycontroller)
	ui.search.Connect("search-changed", process)
	ui.search.Connect("activate", func() { activateItem(false) })

	if config.ShowInitialEntries {
		setInitials()
	}
}

func selectNext() {
	items := ui.selection.NItems()

	if items == 0 {
		return
	}

	current := ui.selection.Selected()

	if current+1 < items {
		ui.selection.SetSelected(current + 1)
	}
}

func selectPrev() {
	items := ui.selection.NItems()

	if items == 0 {
		return
	}

	current := ui.selection.Selected()

	if current > 0 {
		ui.selection.SetSelected(current - 1)
	}
}

func handleKeys() KeyPressHandler {
	return func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Return:
			if modifier == gdk.ControlMask {
				activateItem(true)
			}
		case gdk.KEY_Escape:
			ui.app.Quit()
		case gdk.KEY_Down:
			selectNext()
		case gdk.KEY_Up:
			selectPrev()
		case gdk.KEY_Tab:
			selectNext()
		case gdk.KEY_ISO_Left_Tab:
			selectPrev()
		case gdk.KEY_j:
			if modifier == gdk.ControlMask {
				selectNext()
			}
		case gdk.KEY_k:
			if modifier == gdk.ControlMask {
				selectPrev()
			}
		default:
			return true
		}

		return true
	}
}

func activateItem(keepOpen bool) {
	obj := ui.items.Item(ui.selection.Selected())
	str := obj.Cast().(*gtk.StringObject).String()

	entry := entries[str]
	f := strings.Fields(entry.Exec)

	if len(f) == 0 {
		return
	}

	if config.Terminal != "" {
		if entry.Terminal {
			f = append([]string{config.Terminal, "-e"}, f...)
		}
	} else {
		log.Println("terminal is not set")
		return
	}

	cmd := exec.Command(f[0])

	if len(f) > 1 {
		cmd = exec.Command(f[0], f[1:]...)
	}

	if entry.History {
		saveToHistory(entry.Searchable)
	}

	if entry.Notifyable {
		if !keepOpen {
			ui.appwin.SetVisible(false)
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			n := gio.NewNotification("Error running command...")
			n.SetBody(fmt.Sprintf("%s\n %s", cmd.String(), out))
			ui.app.SendNotification("Error", n)
		}
	} else {
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
	}

	if !keepOpen {
		ui.app.Quit()
		return
	}

	ui.search.SetText("")
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func process() {
	clear(entries)

	if !ui.ListAlwaysShow {
		ui.list.SetVisible(false)
	}

	ui.appwin.SetCSSClasses([]string{})

	text := strings.TrimSpace(ui.search.Text())
	if text == "" {
		ui.items.Splice(0, ui.items.NItems(), []string{})

		if config.ShowInitialEntries {
			setInitials()
		}

		return
	}

	list := []string{}

	prefix := text

	if len(prefix) > 1 {
		prefix = text[0:1]
	}

	hasPrefix := true

	p, ok := procs[prefix]
	if !ok {
		p = procs[""]
		hasPrefix = false
	}

	if hasPrefix {
		ui.appwin.SetCSSClasses(ui.prefixClasses[prefix])
	}

	entrySlice := []processors.Entry{}

	for _, proc := range p {
		e := proc.Entries(text)

		for _, entry := range e {
			str := randomString(5)

			if val, ok := history[entry.Searchable]; ok {
				entry.Used = val.Used
				entry.DaysSinceUsed = val.daysSinceUsed
			}

			entry.Identifier = str

			entries[str] = entry
			entrySlice = append(entrySlice, entry)
		}
	}

	if len(entries) == 0 {
		return
	}

	if hasPrefix {
		text = text[1:]
	}

	tm := 1.0 / float64(len(text))

	calcScore := func(text, target string) int {
		final := 100
		score := fuzzy.RankMatchFold(text, target)

		if score == -1 {
			return 0
		} else {
			score = score
			return final - score
		}
	}

	for k, v := range entrySlice {
		v.Categories = append(v.Categories, v.Label, v.Sub)

		for _, t := range v.Categories {
			if t == "" {
				continue
			}

			score := calcScore(text, t)
			if score > entrySlice[k].ScoreFuzzy {
				entrySlice[k].ScoreFuzzy = score
			}

			if score == 100 {
				break
			}
		}

		usageModifier := func(item processors.Entry) int {
			base := 10

			if item.Used > 0 {
				if item.DaysSinceUsed > 0 {
					base -= item.DaysSinceUsed
				}

				return base * item.Used
			}

			return 0
		}

		usageScore := usageModifier(v)

		entrySlice[k].ScoreFuzzyFinal = float64(usageScore)*tm + float64(entrySlice[k].ScoreFuzzy)/tm
	}

	slices.SortFunc(entrySlice, func(a, b processors.Entry) int {
		if a.ScoreFuzzyFinal == b.ScoreFuzzyFinal {
			return strings.Compare(a.Label, b.Label)
		}

		if a.ScoreFuzzyFinal > b.ScoreFuzzyFinal {
			return -1
		}

		if a.ScoreFuzzyFinal < b.ScoreFuzzyFinal {
			return 1
		}

		return 0
	})

	for _, v := range entrySlice {
		list = append(list, v.Identifier)
	}

	current := ui.items.NItems()

	if current == 0 {
		for _, str := range list {
			ui.items.Append(str)
		}
	} else {
		ui.items.Splice(0, current, list)
	}

	if ui.selection.NItems() > 0 {
		ui.selection.SetSelected(0)
		ui.list.SetVisible(true)
	}
}

func setInitials() {
	ui.list.SetVisible(true)

	ui.items.Splice(0, ui.items.NItems(), []string{})

	sorted := []processors.Entry{}

	for _, v := range procs {
		for _, proc := range v {
			e := proc.Entries("")

			for _, entry := range e {
				str := randomString(5)
				entry.Identifier = str

				sorted = append(sorted, entry)
				entries[str] = entry
			}
		}
	}

	slices.SortFunc(sorted, func(a, b processors.Entry) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})

	for _, v := range sorted {
		ui.items.Append(v.Identifier)
	}

	ui.selection.SetSelected(0)
}

func saveToHistory(searchterm string) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		return
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	h, ok := history[searchterm]
	if !ok {
		h = HistoryEntry{
			LastUsed: time.Now(),
			Used:     1,
		}
	} else {
		h.Used++

		if h.Used > 10 {
			h.Used = 10
		}

		h.LastUsed = time.Now()
	}

	history[searchterm] = h

	b, err := json.Marshal(history)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(filepath.Join(cacheDir, "history.json"), b, 0644)
	if err != nil {
		log.Println(err)
	}
}
