package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type Workable interface {
	Entries(ctx context.Context, term string) []util.Entry
	Prefix() string
	Name() string
	Placeholder() string
	SwitcherOnly() bool
	IsSetup() bool
	Setup(cfg *config.Config) bool
	SetupData(cfg *config.Config)
	Refresh()
	KeepSort() bool
	Typeahead() bool
	History() bool
	Cleanup()
}

func readCache(name string, data any) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		return false
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	path := filepath.Join(cacheDir, fmt.Sprintf("%s.json", name))

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Panicln(err)
		}

		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Panicln(err)
		}

		return true
	}

	return false
}

func Find(plugins []config.Plugin, name string) (config.Plugin, error) {
	for _, v := range plugins {
		if v.Name == name {
			return v, nil
		}
	}

	return config.Plugin{}, errors.New("plugin not found")
}
