package modules

import (
	"errors"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Workable interface {
	Cleanup()
	Entries(term string) []util.Entry
	General() *config.GeneralModule
	Refresh()
	Setup() bool
	SetupData()
}

func Find(plugins []config.Plugin, name string) (config.Plugin, error) {
	for _, v := range plugins {
		if v.Name == name {
			return v, nil
		}
	}

	return config.Plugin{}, errors.New("plugin not found")
}
