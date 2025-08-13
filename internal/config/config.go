package config

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Providers Providers `koanf:",squash"`
}

type Providers struct {
	Default  []string `koanf:"default" desc:"providers to query in default state" default:"desktopapplications,calc,runner"`
	Empty    []string `koanf:"empty" desc:"providers to query when query is empty" default:"desktopapplications"`
	Prefixes []Prefix `koanf:"prefixes" desc:"prefixes to target provider" default:""`
}

type Prefix struct {
	Prefix   string `koanf:"prefix" desc:"prefix" default:""`
	Provider string `koanf:"provider" desc:"provider" default:""`
}

var (
	LoadedConfig       *Config
	AvailableProviders = []Provider{}
)

func init() {
	loadProviderData()

	LoadedConfig = &Config{
		Providers: Providers{
			Default: []string{"desktopapplications", "calc", "runner"},
			Empty:   []string{"desktopapplications"},
			Prefixes: []Prefix{{
				Prefix:   "/",
				Provider: "files",
			}, {
				Prefix:   ".",
				Provider: "symbols",
			}},
		},
	}

	defaults := koanf.New(".")

	err := defaults.Load(structs.Provider(LoadedConfig, "koanf"), nil)
	if err != nil {
		panic(err)
	}
}

type Provider struct {
	Name       string
	NamePretty string
}

func loadProviderData() {
	cmd := exec.Command("elephant", "listproviders")

	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	for v := range bytes.Lines(out) {
		info := strings.TrimSpace(string(v))
		parts := strings.Split(info, ":")
		AvailableProviders = append(AvailableProviders, Provider{
			Name:       parts[1],
			NamePretty: parts[0],
		})
	}
}
