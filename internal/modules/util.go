package modules

import (
	"errors"
	"fmt"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/spf13/viper"
)

type Workable interface {
	Cleanup()
	Entries(term string) []util.Entry
	General() *config.GeneralModule
	Refresh()
	Setup(cfg *config.Config) bool
	SetupData(cfg *config.Config)
}

func Find(plugins []config.Plugin, name string) (config.Plugin, error) {
	for _, v := range plugins {
		if v.Name == name {
			return v, nil
		}
	}

	return config.Plugin{}, errors.New("plugin not found")
}

func wrapWithPrefix(text string) string {
	if viper.GetString("app_launch_prefix") == "" {
		return text
	}

	return fmt.Sprintf("%s%s", viper.GetString("app_launch_prefix"), text)
}
