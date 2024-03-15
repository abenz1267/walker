package processors

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

const ApplicationsName = "applications"

type Entry struct {
	Label           string    `json:"label,omitempty"`
	Sub             string    `json:"sub,omitempty"`
	Exec            string    `json:"exec,omitempty"`
	Terminal        bool      `json:"terminal,omitempty"`
	Icon            string    `json:"icon,omitempty"`
	Searchable      string    `json:"searchable,omitempty"`
	Categories      []string  `json:"categories,omitempty"`
	Notifyable      bool      `json:"notifyable,omitempty"`
	Class           string    `json:"class,omitempty"`
	History         bool      `json:"history,omitempty"`
	Identifier      string    `json:"-"`
	Used            int       `json:"-"`
	DaysSinceUsed   int       `json:"-"`
	LastUsed        time.Time `json:"-"`
	ScoreFuzzy      int       `json:"-"`
	ScoreFuzzyFinal float64   `json:"-"`
}

type Applications struct {
	Apps []Application `json:"apps,omitempty"`
	Prfx string        `json:"prfx,omitempty"`
}

type Application struct {
	Generic Entry   `json:"generic,omitempty"`
	Actions []Entry `json:"actions,omitempty"`
}

func GetApplications() *Applications {
	entries := parse()

	for k, v := range entries {
		entries[k].Generic.Searchable = fmt.Sprintf("%s %s", v.Generic.Sub, v.Generic.Label)

		for n, a := range v.Actions {
			entries[k].Actions[n].Searchable = fmt.Sprintf("%s %s", a.Sub, a.Label)
		}
	}

	return &Applications{
		Apps: entries,
	}
}

func (a Applications) Name() string {
	return ApplicationsName
}

func (a *Applications) SetPrefix(val string) {
	a.Prfx = val
}

func (a Applications) Prefix() string {
	return a.Prfx
}

func (a Applications) Entries(_ string) []Entry {
	entries := []Entry{}

	for _, v := range a.Apps {
		if len(v.Actions) > 0 {
			entries = append(entries, v.Actions...)

			continue
		}

		entries = append(entries, v.Generic)
	}

	return entries
}

func parse() []Application {
	apps := []Application{}

	ok := readCache(ApplicationsName, &apps)
	if ok {
		return apps
	}

	dirs := xdg.ApplicationDirs

	flags := []string{"%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N", "%i", "%c", "%k", "%v", "%m"}

	for _, d := range dirs {
		if _, err := os.Stat(d); err != nil {
			continue
		}

		filepath.Walk(d, func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() && filepath.Ext(path) == ".desktop" {
				file, err := os.Open(path)
				if err != nil {
					return err
				}

				defer file.Close()
				scanner := bufio.NewScanner(file)

				app := Application{
					Generic: Entry{
						Class:   ApplicationsName,
						History: true,
					},
					Actions: []Entry{},
				}

				isAction := false

				for scanner.Scan() {
					line := scanner.Text()

					if strings.HasPrefix(line, "[Desktop Action") {
						app.Actions = append(app.Actions, Entry{
							Sub:        app.Generic.Label,
							Icon:       app.Generic.Icon,
							Terminal:   app.Generic.Terminal,
							Class:      ApplicationsName,
							Categories: app.Generic.Categories,
							History:    app.Generic.History,
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
			}

			return nil
		})
	}

	writeCache(ApplicationsName, apps)

	return apps
}
