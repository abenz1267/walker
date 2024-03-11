package processors

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Label      string
	Sub        string
	Img        string
	Exec       string
	Terminal   bool
	Icon       string
	Searchable string
	Notifyable bool
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
	return "applications"
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

	ok := readCache("applications", &apps)
	if ok {
		return apps
	}

	dir := "/usr/share/applications/"

	flags := []string{"%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N", "%i", "%c", "%k", "%v", "%m"}

	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer file.Close()
			scanner := bufio.NewScanner(file)

			app := Application{
				Generic: Entry{},
				Actions: []Entry{},
			}

			isAction := false

			for scanner.Scan() {
				line := scanner.Text()

				if strings.HasPrefix(line, "[Desktop Action") {
					app.Actions = append(app.Actions, Entry{
						Sub:      app.Generic.Label,
						Icon:     app.Generic.Icon,
						Terminal: app.Generic.Terminal,
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
						app.Generic.Label = strings.TrimPrefix(line, "Name=")
						continue
					}

					if strings.HasPrefix(line, "GenericName=") {
						app.Generic.Sub = strings.TrimPrefix(line, "GenericName=")
						continue
					}

					if strings.HasPrefix(line, "Terminal=") {
						app.Generic.Terminal = strings.TrimPrefix(line, "Terminal=") == "true"
						continue
					}

					if strings.HasPrefix(line, "Icon=") {
						app.Generic.Icon = strings.TrimPrefix(line, "Icon=")
						continue
					}

					if strings.HasPrefix(line, "Exec=") {
						app.Generic.Exec = strings.TrimPrefix(line, "Exec=")

						for _, v := range flags {
							app.Generic.Exec = strings.ReplaceAll(app.Generic.Exec, v, "")
						}

						continue
					}
				} else {
					if strings.HasPrefix(line, "Exec=") {
						app.Actions[len(app.Actions)-1].Exec = strings.TrimPrefix(line, "Exec=")

						for _, v := range flags {
							app.Actions[len(app.Actions)-1].Exec = strings.ReplaceAll(app.Actions[len(app.Actions)-1].Exec, v, "")
						}
						continue
					}

					if strings.HasPrefix(line, "Name=") {
						app.Actions[len(app.Actions)-1].Label = strings.TrimPrefix(line, "Name=")
						continue
					}
				}
			}

			apps = append(apps, app)
		}

		return nil
	})

	writeCache("applications", apps)

	return apps
}
