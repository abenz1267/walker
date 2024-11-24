package windows

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules/windows/wlr"
	"github.com/abenz1267/walker/internal/util"
	"github.com/adrg/xdg"
	"github.com/neurlang/wayland/wl"
)

type Windows struct {
	general   config.GeneralModule
	entries   []util.Entry
	functions []func()
	icons     map[string]string
}

func (w Windows) General() *config.GeneralModule {
	return &w.general
}

func (w Windows) Cleanup() {
}

func (w *Windows) Setup(cfg *config.Config) bool {
	w.general = cfg.Builtins.Windows.GeneralModule

	return true
}

func (w *Windows) SetupData(cfg *config.Config, ctx context.Context) {
	if !wlr.IsRunning {
		go wlr.StartWM(nil, nil)
	}

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

				scanner := bufio.NewScanner(file)

				icon, class := "", ""

				for scanner.Scan() {
					if icon != "" && class != "" {
						w.icons[class] = icon
					}

					line := scanner.Text()

					if strings.HasPrefix(line, "StartupWMClass=") {
						class = strings.TrimSpace(strings.TrimPrefix(line, "StartupWMClass="))
						class = strings.ToLower(class)
						continue
					}

					if strings.HasPrefix(line, "Icon=") {
						icon = strings.TrimSpace(strings.TrimPrefix(line, "Icon="))
						continue
					}
				}
			}

			done[info.Name()] = struct{}{}

			return nil
		})
	}
}

func (w Windows) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	res := wlr.GetWindows()

	for _, v := range res {
		entries = append(entries, util.Entry{
			Label:           v.Title,
			Sub:             fmt.Sprintf("Windows: %s", v.AppId),
			Searchable:      v.AppId,
			Icon:            w.icons[v.AppId],
			Categories:      []string{"windows"},
			Class:           "windows",
			Matching:        util.Fuzzy,
			SpecialFunc:     w.SpecialFunc,
			SpecialFuncArgs: []interface{}{v.Toplevel.Id()},
		})
	}

	return entries
}

func (w *Windows) Refresh() {
}

func (w Windows) SpecialFunc(args ...interface{}) {
	id := args[0].(wl.ProxyId)

	wlr.Activate(id)
}
