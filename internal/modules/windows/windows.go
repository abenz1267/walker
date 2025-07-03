package windows

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules/windows/wlr"
	"github.com/abenz1267/walker/internal/util"
	"github.com/adrg/xdg"
	"github.com/neurlang/wayland/wl"
)

type Windows struct {
	mutex     sync.Mutex
	general   config.GeneralModule
	entries   []util.Entry
	functions []func()
	icons     map[string]string
}

func (w *Windows) General() *config.GeneralModule {
	return &w.general
}

func (w *Windows) Cleanup() {
}

func (w *Windows) Setup() bool {
	w.general = config.Cfg.Builtins.Windows.GeneralModule

	return true
}

func (w *Windows) SetupData() {
	w.icons = make(map[string]string)
	w.GetIcons()

	w.general.IsSetup = true
	w.general.HasInitialSetup = true
}

func (w *Windows) GetIcons() {
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

				icon, class, isMainEntry, scanner := "", "", false, bufio.NewScanner(file)

				for scanner.Scan() {
					line := scanner.Text()

					if strings.Contains(line, "Desktop Entry") {
						isMainEntry = true
					}

					if isMainEntry {
						if strings.HasPrefix(line, "StartupWMClass=") && class == "" {
							class = strings.TrimSpace(strings.TrimPrefix(line, "StartupWMClass="))
							class = strings.ToLower(class)
							continue
						}

						if strings.HasPrefix(line, "Icon=") && icon == "" {
							icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
							continue
						}
					}
				}

				if icon != "" {
					w.mutex.Lock()

					w.icons[strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))] = icon

					if class != "" {
						w.icons[class] = icon
					}

					w.mutex.Unlock()
				}
			}

			done[info.Name()] = struct{}{}

			return nil
		})
	}
}

func (w *Windows) Entries(term string) []util.Entry {
	entries := []util.Entry{}

	res := wlr.GetWindows()

	for _, v := range res {
		entry := util.Entry{
			Label:           v.Title,
			Sub:             fmt.Sprintf("Windows: %s", v.AppId),
			Searchable:      v.AppId,
			Categories:      []string{"windows"},
			Class:           "windows",
			Matching:        util.Fuzzy,
			SpecialFunc:     w.SpecialFunc,
			SpecialFuncArgs: []interface{}{v.Toplevel.Id()},
		}

		w.mutex.Lock()
		entry.Icon = w.icons[v.AppId]
		w.mutex.Unlock()

		entries = append(entries, entry)
	}

	return entries
}

func (w *Windows) Refresh() {
}

func (w *Windows) SpecialFunc(args ...interface{}) {
	id := args[0].(wl.ProxyId)

	wlr.Activate(id)
}
