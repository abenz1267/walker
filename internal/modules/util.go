package modules

import (
	"errors"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

var EntryChan chan *util.Entry

func init() {
	EntryChan = make(chan *util.Entry)
}

type Workable interface {
	Cleanup()
	Entries(term string) []*util.Entry
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
