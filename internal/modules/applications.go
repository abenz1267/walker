package modules

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/history"
	"github.com/abenz1267/walker/internal/modules/windows/wlr"
	"github.com/abenz1267/walker/internal/util"
	"github.com/adrg/xdg"
	"github.com/djherbis/times"
	"github.com/fsnotify/fsnotify"
)

const ApplicationsName = "applications"

var fieldCodes = []string{"%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N", "%i", "%c", "%k", "%v", "%m"}

var TerminalApps = map[string]struct{}{}

type Applications struct {
	config      config.Applications
	mu          sync.Mutex
	entries     []util.Entry
	openWindows map[string]uint
	wmRunning   bool
	isWatching  bool
	Hstry       history.History
}

type Application struct {
	Generic util.Entry   `json:"generic,omitempty"`
	Actions []util.Entry `json:"actions,omitempty"`
}

func (a *Applications) General() *config.GeneralModule {
	return &a.config.GeneralModule
}

func (a *Applications) Cleanup() {}

func (a *Applications) Setup() bool {
	a.config = config.Cfg.Builtins.Applications

	a.openWindows = make(map[string]uint)

	return true
}

func (a *Applications) SetupData() {
	a.entries = a.parse()

	if config.Cfg.IsService {
		go a.Watch()
	}

	if !a.wmRunning && a.config.ContextAware {
		go a.RunWm()
	}

	a.config.IsSetup = true
	a.config.HasInitialSetup = true
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
				a.entries = a.parse()
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

			for k := range a.entries {
				if _, ok := a.openWindows[a.entries[k].InitialClass]; ok {
					a.entries[k].OpenWindows = a.openWindows[a.entries[k].InitialClass]
				} else {
					a.entries[k].OpenWindows = 0
				}
			}

			a.mu.Unlock()
		}
	}
}

func (a *Applications) Refresh() {
	if !a.isWatching {
		a.config.IsSetup = !a.config.Refresh
	}
}

func (a *Applications) Entries(term string) []util.Entry {
	if a.config.Actions.HideWithoutQuery && term == "" {
		entries := []util.Entry{}
		added := make(map[string]struct{})

		for _, entry := range a.entries {
			if entry.IsAction {
				for _, v := range a.Hstry {
					if _, ok := v[entry.Identifier()]; ok {
						if _, ok := added[entry.Identifier()]; ok {
							continue
						}

						entries = append(entries, entry)
						added[entry.Identifier()] = struct{}{}
						continue
					}
				}

				continue
			}

			entries = append(entries, entry)
		}

		return entries
	}

	return a.entries
}

func (a *Applications) parse() []util.Entry {
	apps := []Application{}
	entries := []util.Entry{}
	desktop := os.Getenv("XDG_CURRENT_DESKTOP")

	langFull := config.Cfg.Locale

	if langFull == "" {
		langFull = os.Getenv("LANG")

		lang_messages := os.Getenv("LC_MESSAGES")
		if lang_messages != "" {
			langFull = lang_messages
		}

		lang_all := os.Getenv("LC_ALL")
		if lang_all != "" {
			langFull = lang_all
		}

		langFull = strings.Split(langFull, ".")[0]
	}

	langSingle := strings.Split(langFull, "_")[0]

	nameFull := fmt.Sprintf("Name[%s]=", langFull)
	nameSingle := fmt.Sprintf("Name[%s]=", langSingle)
	commentFull := fmt.Sprintf("Comment[%s]=", langFull)
	commentSingle := fmt.Sprintf("Comment[%s]=", langSingle)
	genericNameFull := fmt.Sprintf("GenericName[%s]=", langFull)
	genericNameSingle := fmt.Sprintf("GenericName[%s]=", langSingle)
	keywordsFull := fmt.Sprintf("Keywords[%s]=", langFull)
	keywordsSingle := fmt.Sprintf("Keywords[%s]=", langSingle)

	if a.config.Cache {
		ok := util.FromGob(filepath.Join(util.CacheDir(), fmt.Sprintf("%s.gob", ApplicationsName)), &entries)
		if ok {
			return entries
		}
	}

	dirs := xdg.ApplicationDirs

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

				if a.config.PrioritizeNew {
					if info, err := times.Stat(path); err == nil {
						if info.HasBirthTime() {
							target := time.Now().Add(-time.Minute * 5)
							bt := info.BirthTime()

							if bt.After(target) {
								matching = util.AlwaysTopOnEmptySearch
							}
						}
					}
				}

				scanner := bufio.NewScanner(file)

				app := Application{
					Generic: util.Entry{
						Class:            ApplicationsName,
						History:          a.config.History,
						Matching:         matching,
						RecalculateScore: true,
						File:             path,
						Searchable:       path,
					},
					Actions: []util.Entry{},
				}

				isAction := false
				skip := false
				keywords := []string{}
				name := ""
				localizedNameSingle := ""
				localizedNameFull := ""

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
						if !a.config.Actions.Enabled {
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
							name = strings.TrimSpace(strings.TrimPrefix(line, "Name="))

							continue
						}

						if strings.HasPrefix(line, nameSingle) {
							localizedNameSingle = strings.TrimSpace(strings.TrimPrefix(line, nameSingle))

							continue
						}

						if strings.HasPrefix(line, nameFull) {
							localizedNameFull = strings.TrimSpace(strings.TrimPrefix(line, nameFull))

							continue
						}

						if strings.HasPrefix(line, "Comment=") {
							app.Generic.Searchable2 = strings.TrimSpace(strings.TrimPrefix(line, "Comment="))
							continue
						}

						if strings.HasPrefix(line, commentSingle) {
							app.Generic.Searchable2 = strings.TrimSpace(strings.TrimPrefix(line, commentSingle))
							continue
						}

						if strings.HasPrefix(line, commentFull) {
							app.Generic.Searchable2 = strings.TrimSpace(strings.TrimPrefix(line, commentFull))
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
							keywords = strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Keywords=")), ";")
							continue
						}

						if strings.HasPrefix(line, keywordsSingle) {
							keywords = strings.Split(strings.TrimSpace(strings.TrimPrefix(line, keywordsSingle)), ";")
							continue
						}

						if strings.HasPrefix(line, keywordsFull) {
							keywords = strings.Split(strings.TrimSpace(strings.TrimPrefix(line, keywordsFull)), ";")
							continue
						}

						if strings.HasPrefix(line, "GenericName=") {
							app.Generic.Sub = strings.TrimSpace(strings.TrimPrefix(line, "GenericName="))
							continue
						}

						if strings.HasPrefix(line, genericNameSingle) {
							app.Generic.Sub = strings.TrimSpace(strings.TrimPrefix(line, genericNameSingle))
							continue
						}

						if strings.HasPrefix(line, genericNameFull) {
							app.Generic.Sub = strings.TrimSpace(strings.TrimPrefix(line, genericNameFull))
							continue
						}

						if strings.HasPrefix(line, "Terminal=") {
							app.Generic.Terminal = strings.TrimSpace(strings.TrimPrefix(line, "Terminal=")) == "true"

							if app.Generic.Terminal {
								TerminalApps[filepath.Base(path)] = struct{}{}
							}

							continue
						}

						if strings.HasPrefix(line, "StartupWMClass=") {
							app.Generic.InitialClass = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "StartupWMClass=")))

							if val, ok := a.openWindows[app.Generic.InitialClass]; ok {
								app.Generic.OpenWindows = val
							}

							continue
						}

						if strings.HasPrefix(line, "Icon=") {
							app.Generic.Icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
							continue
						}

						if strings.HasPrefix(line, "Exec=") {
							parsed, err := parseExec(strings.TrimPrefix(line, "Exec="))
							if err != nil {
								slog.Error("applications", "error", err)
								continue
							}

							app.Generic.Exec = strings.TrimSpace(strings.Join(parsed, " "))

							continue
						}
					} else {
						if strings.HasPrefix(line, "Exec=") {
							parsed, err := parseExec(strings.TrimPrefix(line, "Exec="))
							if err != nil {
								slog.Error("applications", "error", err)
								continue
							}

							app.Actions[len(app.Actions)-1].Exec = strings.TrimSpace(strings.Join(parsed, " "))

							continue
						}

						if strings.HasPrefix(line, "Name=") {
							app.Actions[len(app.Actions)-1].Label = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
							continue
						}

						if strings.HasPrefix(line, nameSingle) {
							app.Actions[len(app.Actions)-1].Label = strings.TrimSpace(strings.TrimPrefix(line, nameSingle))
							continue
						}

						if strings.HasPrefix(line, nameFull) {
							app.Actions[len(app.Actions)-1].Label = strings.TrimSpace(strings.TrimPrefix(line, nameFull))
							continue
						}
					}
				}

				app.Generic.Label = name

				if localizedNameSingle != "" {
					app.Generic.Label = localizedNameSingle
				}

				if localizedNameFull != "" {
					app.Generic.Label = localizedNameFull
				}

				for k := range app.Actions {
					sub := app.Generic.Label

					if a.config.ShowGeneric && app.Generic.Sub != "" && !a.config.Actions.HideCategory {
						sub = fmt.Sprintf("%s (%s)", app.Generic.Label, app.Generic.Sub)
					}

					app.Actions[k].Sub = sub
					app.Actions[k].Path = app.Generic.Path
					app.Actions[k].Icon = app.Generic.Icon
					app.Actions[k].Terminal = app.Generic.Terminal
					app.Actions[k].Class = ApplicationsName
					app.Actions[k].Matching = app.Generic.Matching
					app.Actions[k].Categories = app.Generic.Categories
					app.Actions[k].Categories = append(app.Actions[k].Categories, keywords...)
					app.Actions[k].History = app.Generic.History
					app.Actions[k].InitialClass = app.Generic.InitialClass
					app.Actions[k].OpenWindows = app.Generic.OpenWindows
					app.Actions[k].Prefer = true
					app.Actions[k].RecalculateScore = true
					app.Actions[k].File = path
					app.Actions[k].Searchable = path
					app.Actions[k].Searchable2 = app.Generic.Searchable2
					app.Actions[k].IsAction = true
				}

				app.Generic.Categories = append(app.Generic.Categories, keywords...)

				apps = append(apps, app)

				done[info.Name()] = struct{}{}
			}

			return nil
		})
	}

	for _, v := range apps {
		if a.config.ShowGeneric || !a.config.Actions.Enabled || len(v.Actions) == 0 {
			entries = append(entries, v.Generic)
		}

		if a.config.Actions.Enabled {
			entries = append(entries, v.Actions...)
		}
	}

	if a.config.Cache {
		util.ToGob(&entries, filepath.Join(util.CacheDir(), fmt.Sprintf("%s.gob", ApplicationsName)))
	}

	return entries
}

// parseExec converts an XDG desktop file Exec entry into a slice of strings
// suitable for exec.Command. It handles field codes and proper escaping according
// to the XDG Desktop Entry specification.
// See: https://specifications.freedesktop.org/desktop-entry-spec/latest/ar01s07.html
func parseExec(execLine string) ([]string, error) {
	if execLine == "" {
		return nil, errors.New("empty exec line")
	}

	var (
		parts   []string
		current strings.Builder
		inQuote bool
		escaped bool
	)

	// Helper to append current token and reset builder
	appendCurrent := func() {
		if current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
	}

	// Process each rune in the exec line
	for _, r := range execLine {
		switch {
		case escaped:
			// Handle escaped character
			current.WriteRune(r)
			escaped = false

		case r == '\\':
			escaped = true

		case r == '"':
			inQuote = !inQuote

		case unicode.IsSpace(r) && !inQuote:
			// Space outside quotes marks token boundary
			appendCurrent()

		default:
			current.WriteRune(r)
		}
	}

	// Append final token if any
	appendCurrent()

	// Remove field codes
	for k, v := range parts {
		if len(v) == 2 && slices.Contains(fieldCodes, v) {
			until := k + 1

			if until > len(parts) {
				until = len(parts)
			}

			parts = slices.Delete(parts, k, until)
		}
	}

	if len(parts) == 0 {
		return nil, errors.New("no command found after parsing")
	}

	return parts, nil
}
