package main

import (
	"fmt"
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

func setupInteractions(ui *UI, entries map[string]processors.Entry, config *Config) {
	ps := make(map[string]Processor)
	ps["applications"] = processors.GetApplications()
	ps["runner"] = &processors.Runner{}
	ps["websearch"] = &processors.Websearch{}

	for _, v := range config.Processors {
		if _, ok := ps[v.Name]; !ok {
			fmt.Println(v.Name)
			delete(ps, v.Name)
		}
	}

	for _, v := range config.Processors {
		for m, n := range ps {
			if n.Name() == v.Name {
				ps[m].SetPrefix(v.Prefix)
			}
		}
	}

	procs := make(map[string][]Processor)

	for _, v := range ps {
		procs[v.Prefix()] = append(procs[v.Prefix()], v)
	}

	search := ui.search.Cast().(*gtk.Entry)

	keycontroller := gtk.NewEventControllerKey()
	keycontroller.ConnectKeyPressed(handleKeys(ui, entries, config))

	search.AddController(keycontroller)
	search.Connect("changed", process(procs, ui, entries))
	search.Connect("activate", activateItem(ui, entries, config, false))

	// sigchnl := make(chan os.Signal, 1)
	// signal.Notify(sigchnl)
	// go func() {
	// 	for {
	// 		s := <-sigchnl
	// 		signalHandler(appwin, s)
	// 	}
	// }()
}

func hideUI(ui *UI) {
	// ui.appwin.Cast().(*gtk.ApplicationWindow).SetVisible(false)
	// ui.search.Cast().(*gtk.Entry).SetText("")
	// ui.items.Splice(0, ui.items.NItems(), []string{})
	ui.app.Quit()
}

func handleKeys(ui *UI, entries map[string]processors.Entry, config *Config) KeyPressHandler {
	return func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Return:
			if modifier == gdk.ControlMask {
				activateItem(ui, entries, config, true)(ui.search.Cast().(*gtk.Entry))
			}
		case gdk.KEY_Escape:
			hideUI(ui)
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

func activateItem(ui *UI, entries map[string]processors.Entry, config *Config, keepOpen bool) func(search *gtk.Entry) {
	return func(search *gtk.Entry) {
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

		pth, err := exec.LookPath(f[0])
		if err != nil {
			log.Println("command not found")
			return
		}

		cmd := exec.Command(pth, f[1:]...)

		if entry.Notifyable {
			if !keepOpen {
				ui.appwin.SetVisible(false)
			}

			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println(err)

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
			hideUI(ui)
			return
		}

		ui.search.Cast().(*gtk.Entry).SetText("")
	}
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

func process(procs map[string][]Processor, ui *UI, entries map[string]processors.Entry) func(search *gtk.Entry) {
	return func(search *gtk.Entry) {
		clear(entries)

		view := ui.list.Cast().(*gtk.ListView)
		view.SetVisible(false)

		text := search.Text()
		if text == "" {
			ui.items.Splice(0, ui.items.NItems(), []string{})
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
			view.SetVisible(true)
		}
	}
}

func signalHandler(win *gtk.ApplicationWindow, signal os.Signal) {
	switch signal {
	case syscall.SIGTERM, syscall.SIGINT:
		os.Exit(0)
	case syscall.SIGUSR1:
		win.SetVisible(true)
	default:
	}
}
