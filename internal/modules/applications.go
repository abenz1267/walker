package modules

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules/windows/wlr"
	"github.com/abenz1267/walker/internal/util"
	"github.com/adrg/xdg"
	"github.com/djherbis/times"
	"github.com/fsnotify/fsnotify"
)

const ApplicationsName = "applications"

type Applications struct {
	general        config.GeneralModule
	mu             sync.Mutex
	cache          bool
	actions        bool
	prioritizeNew  bool
	entries        []util.Entry
	isContextAware bool
	openWindows    map[string]uint
	wmRunning      bool
	isWatching     bool
	showGeneric    bool
}

type Application struct {
	Generic util.Entry   `json:"generic,omitempty"`
	Actions []util.Entry `json:"actions,omitempty"`
}

func (a *Applications) General() *config.GeneralModule {
	return &a.general
}

func (a *Applications) Cleanup() {}

func (a *Applications) Setup(cfg *config.Config) bool {
	a.general = cfg.Builtins.Applications.GeneralModule

	a.cache = cfg.Builtins.Applications.Cache
	a.actions = cfg.Builtins.Applications.Actions
	a.prioritizeNew = cfg.Builtins.Applications.PrioritizeNew
	a.isContextAware = cfg.Builtins.Applications.ContextAware
	a.showGeneric = cfg.Builtins.Applications.ShowGeneric
	a.openWindows = make(map[string]uint)

	return true
}

func (a *Applications) SetupData(cfg *config.Config, ctx context.Context) {
	a.entries = parse(a.cache, a.actions, a.prioritizeNew, a.openWindows, a.showGeneric)

	if cfg.IsService {
		go a.Watch()
	}

	if !a.wmRunning && a.isContextAware {
		go a.RunWm()
	}

	a.general.IsSetup = true
	a.general.HasInitialSetup = true
}

func (a *Applications) Watch() {
	a.isWatching = true

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panicln(err)
	}
	defer watcher.Close()

	for _, v := range xdg.ApplicationDirs {
		if util.FileExists(v) {
			err := watcher.Add(v)
			if err != nil {
				log.Panicln(err)
			}
		}
	}

	rc := make(chan struct{})
	go a.debounceParsing(500*time.Millisecond, rc)

	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}

				rc <- struct{}{}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	<-make(chan struct{})
}

func (a *Applications) debounceParsing(interval time.Duration, input chan struct{}) {
	shouldParse := false

	for {
		select {
		case <-input:
			shouldParse = true
		case <-time.After(interval):
			if shouldParse {
				a.entries = parse(a.cache, a.actions, a.prioritizeNew, a.openWindows, a.showGeneric)
				shouldParse = false
			}
		}
	}
}

func (a *Applications) RunWm() {
	addChan := make(chan string)
	deleteChan := make(chan string)

	a.wmRunning = true

	go wlr.StartWM(addChan, deleteChan)

	for {
		select {
		case appId := <-addChan:
			a.mu.Lock()
			val, ok := a.openWindows[appId]

			if ok {
				a.openWindows[appId] = val + 1
			} else {
				a.openWindows[appId] = 1
			}

			for k := range a.entries {
				if _, ok := a.openWindows[a.entries[k].InitialClass]; ok {
					a.entries[k].OpenWindows = a.openWindows[a.entries[k].InitialClass]
				}
			}

			a.mu.Unlock()
		case appId := <-deleteChan:
			a.mu.Lock()

			if val, ok := a.openWindows[appId]; ok {
				if val == 1 {
					delete(a.openWindows, appId)
				} else {
					a.openWindows[appId] = val - 1
				}
			}

			a.mu.Unlock()
		}
	}
}

func (a *Applications) Refresh() {
	if !a.isWatching {
		a.general.IsSetup = !a.general.Refresh
	}
}

func (a *Applications) Entries(ctx context.Context, term string) []util.Entry {
	return a.entries
}

func parse(cache, actions, prioritizeNew bool, openWindows map[string]uint, showGeneric bool) []util.Entry {
	apps := []Application{}
	entries := []util.Entry{}
	desktop := os.Getenv("XDG_CURRENT_DESKTOP")

	if cache {
		ok := readCache(ApplicationsName, &entries)
		if ok {
			return entries
		}
	}

	dirs := xdg.ApplicationDirs

	flags := []string{"%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N", "%i", "%c", "%k", "%v", "%m"}

	done := make(map[string]struct{})

	for _, d := range dirs {
		if _, err := os.Stat(d); err != nil {
			continue
		}

		filepath.WalkDir(d, func(path string, info fs.DirEntry, err error) error {
			if _, ok := done[info.Name()]; ok {
				return nil
			}

			if !info.IsDir() && filepath.Ext(path) == ".desktop" {
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				defer file.Close()

				matching := util.Fuzzy

				if prioritizeNew {
					if info, err := times.Stat(path); err == nil {
						target := time.Now().Add(-time.Minute * 5)

						mod := info.BirthTime()
						if mod.After(target) {
							matching = util.AlwaysTopOnEmptySearch
						}
					}
				}

				scanner := bufio.NewScanner(file)

				fmt.Println(path)

				app := Application{
					Generic: util.Entry{
						Class:            ApplicationsName,
						History:          true,
						Matching:         matching,
						RecalculateScore: true,
						File:             path,
						Searchable:       path,
					},
					Actions: []util.Entry{},
				}

				isAction := false
				skip := false

				for scanner.Scan() {
					line := scanner.Text()

					if strings.HasPrefix(line, "[Desktop Entry") {
						isAction = false
						skip = false
						continue
					}

					if skip {
						continue
					}

					if strings.HasPrefix(line, "[Desktop Action") {
						if !actions {
							skip = true
						}

						app.Actions = append(app.Actions, util.Entry{})

						isAction = true
					}

					if strings.HasPrefix(line, "NoDisplay=") {
						nodisplay := strings.TrimPrefix(line, "NoDisplay=") == "true"

						if nodisplay {
							done[info.Name()] = struct{}{}
							return nil
						}

						continue
					}

					if strings.HasPrefix(line, "OnlyShowIn=") {
						onlyshowin := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "OnlyShowIn=")), ";")

						if slices.Contains(onlyshowin, desktop) {
							continue
						}

						done[info.Name()] = struct{}{}
						return nil
					}

					if strings.HasPrefix(line, "NotShowIn=") {
						notshowin := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "NotShowIn=")), ";")

						if slices.Contains(notshowin, desktop) {
							done[info.Name()] = struct{}{}
							return nil
						}

						continue
					}

					if !isAction {
						if strings.HasPrefix(line, "Name=") {
							app.Generic.Label = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
							continue
						}

						if strings.HasPrefix(line, "Path=") {
							app.Generic.Path = strings.TrimSpace(strings.TrimPrefix(line, "Path="))
							continue
						}

						if strings.HasPrefix(line, "Categories=") {
							cats := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Categories=")), ";")
							app.Generic.Categories = append(app.Generic.Categories, cats...)
							continue
						}

						if strings.HasPrefix(line, "Keywords=") {
							cats := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Keywords=")), ";")
							app.Generic.Categories = append(app.Generic.Categories, cats...)
							continue
						}

						if strings.HasPrefix(line, "GenericName=") {
							app.Generic.Sub = strings.TrimSpace(strings.TrimPrefix(line, "GenericName="))
							continue
						}

						if strings.HasPrefix(line, "Terminal=") {
							app.Generic.Terminal = strings.TrimSpace(strings.TrimPrefix(line, "Terminal=")) == "true"
							continue
						}

						if strings.HasPrefix(line, "StartupWMClass=") {
							app.Generic.InitialClass = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "StartupWMClass=")))

							if val, ok := openWindows[app.Generic.InitialClass]; ok {
								app.Generic.OpenWindows = val
							}

							continue
						}

						if strings.HasPrefix(line, "Icon=") {
							app.Generic.Icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
							continue
						}

						if strings.HasPrefix(line, "Exec=") {
							app.Generic.Exec = strings.TrimSpace(strings.TrimPrefix(line, "Exec="))

							for _, v := range flags {
								app.Generic.Exec = strings.ReplaceAll(app.Generic.Exec, v, "")
							}

							continue
						}
					} else {
						if strings.HasPrefix(line, "Exec=") {
							app.Actions[len(app.Actions)-1].Exec = strings.TrimSpace(strings.TrimPrefix(line, "Exec="))

							for _, v := range flags {
								app.Actions[len(app.Actions)-1].Exec = strings.ReplaceAll(app.Actions[len(app.Actions)-1].Exec, v, "")
							}
							continue
						}

						if strings.HasPrefix(line, "Name=") {
							app.Actions[len(app.Actions)-1].Label = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
							continue
						}
					}
				}

				for k := range app.Actions {
					sub := app.Generic.Label

					if showGeneric && app.Generic.Sub != "" {
						sub = fmt.Sprintf("%s (%s)", app.Generic.Label, app.Generic.Sub)
					}

					app.Actions[k].Sub = sub
					app.Actions[k].Path = app.Generic.Path
					app.Actions[k].Icon = app.Generic.Icon
					app.Actions[k].Terminal = app.Generic.Terminal
					app.Actions[k].Class = ApplicationsName
					app.Actions[k].Matching = app.Generic.Matching
					app.Actions[k].Categories = app.Generic.Categories
					app.Actions[k].History = app.Generic.History
					app.Actions[k].InitialClass = app.Generic.InitialClass
					app.Actions[k].OpenWindows = app.Generic.OpenWindows
					app.Actions[k].Prefer = true
					app.Actions[k].RecalculateScore = true
					app.Actions[k].File = path
					app.Actions[k].Searchable = path
				}

				apps = append(apps, app)

				done[info.Name()] = struct{}{}
			}

			return nil
		})
	}

	for _, v := range apps {
		entries = append(entries, v.Generic)
		entries = append(entries, v.Actions...)
	}

	if cache {
		util.ToJson(&entries, filepath.Join(util.CacheDir(), fmt.Sprintf("%s.json", ApplicationsName)))
	}

	return entries
}
