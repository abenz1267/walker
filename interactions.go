package main

import (
	"log"
	"math/rand"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/processors"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
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
			Prfx: v.Prefix,
			Nme:  v.Name,
			Src:  v.Src,
			Cmd:  v.Cmd,
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

func handleKeys() KeyPressHandler {
	return func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Return:
			if modifier == gdk.ControlMask {
				activateItem(true)
			}
		case gdk.KEY_Escape:
			ui.app.Quit()
		case gdk.KEY_j:
			if modifier == gdk.ControlMask {
				items := ui.selection.NItems()

				if items == 0 {
					return true
				}

				current := ui.selection.Selected()

				if current+1 < items {
					ui.selection.SetSelected(current + 1)
				}
			}
		case gdk.KEY_k:
			if modifier == gdk.ControlMask {
				items := ui.selection.NItems()

				if items == 0 {
					return true
				}

				current := ui.selection.Selected()

				if current > 0 {
					ui.selection.SetSelected(current - 1)
				}
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

	if config.Terminal != "" {
		if entry.Terminal {
			f = append([]string{config.Terminal, "-e"}, f...)
		}
	} else {
		log.Println("terminal is not set")
		return
	}

	cmd := exec.Command(f[0], f[1:]...)

	if entry.Notifyable {
		if !keepOpen {
			ui.appwin.SetVisible(false)
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			notify, err := exec.LookPath("notify-send")
			if err != nil {
				log.Println(err)
			}

			if notify != "" {
				if config.NotifyOnFail {
					n := exec.Command("notify-send", "Walker", string(out), "--app-name=Walker")
					n.Start()
				}
			}
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

	for _, proc := range p {
		e := proc.Entries(text)

		for _, entry := range e {
			str := randomString(5)

			entries[str] = entry
		}
	}

	if hasPrefix {
		text = text[1:]
	}

	searchables := []string{}

	sm := make(map[string][]string)

	for k, entry := range entries {
		sm[entry.Searchable] = append(sm[entry.Searchable], k)
		searchables = append(searchables, entries[k].Searchable)
	}

	slices.Sort(searchables)

	if len(searchables) == 0 {
		return
	}

	j := 0
	for i := 1; i < len(searchables); i++ {
		if searchables[j] == searchables[i] {
			continue
		}
		j++
		searchables[j] = searchables[i]
	}
	result := searchables[:j+1]

	matches := fuzzy.RankFindFold(text, result)
	sort.Sort(matches)

	for _, v := range matches {
		for _, m := range sm[v.Target] {
			list = append(list, m)
		}
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

func signalHandler(signal os.Signal) {
	switch signal {
	case syscall.SIGTERM, syscall.SIGINT:
		os.Exit(0)
	case syscall.SIGUSR1:
		ui.appwin.SetVisible(true)
	default:
		log.Println(signal)
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
