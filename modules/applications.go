package modules

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules/windows/wlr"
	"github.com/abenz1267/walker/util"
	"github.com/adrg/xdg"
	"github.com/djherbis/times"
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
	a.openWindows = make(map[string]uint)

	return true
}

func (a *Applications) SetupData(_ *config.Config, ctx context.Context) {
	a.entries = parse(a.cache, a.actions, a.prioritizeNew, a.openWindows)

	if !a.wmRunning && a.isContextAware {
		go a.RunWm()
	}

	a.general.IsSetup = true
	a.general.HasInitialSetup = true
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
	a.general.IsSetup = !a.general.Refresh
}

func (a *Applications) Entries(ctx context.Context, _ string) []util.Entry {
	return a.entries
}

func parse(cache, actions, prioritizeNew bool, openWindows map[string]uint) []util.Entry {
	apps := []Application{}
	entries := []util.Entry{}

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

				defer file.Close()
				scanner := bufio.NewScanner(file)

				app := Application{
					Generic: util.Entry{
						Class:            ApplicationsName,
						History:          true,
						Matching:         matching,
						RecalculateScore: true,
					},
					Actions: []util.Entry{},
				}

				isAction := false

				for scanner.Scan() {
					line := scanner.Text()

					if strings.HasPrefix(line, "[Desktop Action") {
						if !actions {
							break
						}

						app.Actions = append(app.Actions, util.Entry{
							Sub:              app.Generic.Label,
							Path:             app.Generic.Path,
							Icon:             app.Generic.Icon,
							Terminal:         app.Generic.Terminal,
							Class:            ApplicationsName,
							Matching:         app.Generic.Matching,
							Categories:       app.Generic.Categories,
							History:          app.Generic.History,
							InitialClass:     app.Generic.InitialClass,
							OpenWindows:      app.Generic.OpenWindows,
							Prefer:           true,
							RecalculateScore: true,
						})

						isAction = true
					}

					if strings.HasPrefix(line, "NoDisplay=") {
						nodisplay := strings.TrimPrefix(line, "NoDisplay=") == "true"

						if nodisplay {
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
