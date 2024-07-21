package modules

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
	"github.com/adrg/xdg"
)

const ApplicationsName = "applications"

type Applications struct {
	general config.GeneralModule
	cache   bool
	actions bool
}

type Application struct {
	Generic Entry   `json:"generic,omitempty"`
	Actions []Entry `json:"actions,omitempty"`
}

func (a Applications) Placeholder() string {
	if a.general.Placeholder == "" {
		return "applications"
	}

	return a.general.Placeholder
}

func (Applications) KeepSort() bool {
	return false
}

func (a Applications) IsSetup() bool {
	return a.general.IsSetup
}

func (a Applications) SwitcherOnly() bool {
	return a.general.SwitcherOnly
}

func (a *Applications) Setup(cfg *config.Config) {
	a.general.Prefix = cfg.Builtins.Applications.Prefix
	a.general.SwitcherOnly = cfg.Builtins.Applications.SwitcherOnly
	a.general.SpecialLabel = cfg.Builtins.Applications.SpecialLabel

	a.cache = cfg.Builtins.Applications.Cache
	a.actions = cfg.Builtins.Applications.Actions

	a.general.IsSetup = true
}

func (a Applications) SetupData(_ *config.Config) {}

func (a Applications) Refresh() {}

func (a Applications) Name() string {
	return ApplicationsName
}

func (a Applications) Prefix() string {
	return a.general.Prefix
}

func (a Applications) Entries(ctx context.Context, _ string) []Entry {
	return parse(a.cache, a.actions)
}

func parse(cache bool, actions bool) []Entry {
	apps := []Application{}
	entries := []Entry{}

	if cache {
		ok := readCache(ApplicationsName, &entries)
		if ok {
			return entries
		}
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
						Class:            ApplicationsName,
						History:          true,
						Matching:         Fuzzy,
						RecalculateScore: true,
					},
					Actions: []Entry{},
				}

				isAction := false

				for scanner.Scan() {
					line := scanner.Text()

					if strings.HasPrefix(line, "[Desktop Action") {
						app.Actions = append(app.Actions, Entry{
							Sub:              app.Generic.Label,
							Icon:             app.Generic.Icon,
							Terminal:         app.Generic.Terminal,
							Class:            ApplicationsName,
							Matching:         app.Generic.Matching,
							Categories:       app.Generic.Categories,
							History:          app.Generic.History,
							InitialClass:     app.Generic.InitialClass,
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

				if !actions {
					app.Actions = []Entry{}
				}

				apps = append(apps, app)
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
