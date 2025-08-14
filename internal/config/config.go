package config

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	CloseWhenOpen bool      `koanf:"close_when_open" desc:"closes walker when it's already opened" default:"true"`
	Providers     Providers `koanf:",squash"`
	Keybinds      Keybinds  `koanf:",squash"`
}

type Keybinds struct {
	Close    string `koanf:"close" desc:"close walker" default:"escape"`
	Next     string `koanf:"next" desc:"select next in list" default:"Down"`
	Previous string `koanf:"previous" desc:"select previous in list" default:"Up"`
}

type Providers struct {
	Default             []string            `koanf:"default" desc:"providers to query in default state" default:"desktopapplications,calc,runner"`
	Empty               []string            `koanf:"empty" desc:"providers to query when query is empty" default:"desktopapplications"`
	Prefixes            []Prefix            `koanf:"prefixes" desc:"prefixes to target provider" default:""`
	Calc                Calc                `koanf:",squash"`
	Clipboard           Clipboard           `koanf:",squash"`
	DesktopApplications DesktopApplications `koanf:",squash"`
	Files               Files               `koanf:",squash"`
	Runner              Runner              `koanf:",squash"`
	Symbols             Symbols             `koanf:",squash"`
}

type Calc struct {
	Copy   string `koanf:"copy" desc:"keybind to copy result" default:"enter"`
	Save   string `koanf:"save" desc:"keybind to save result to history" default:"ctrl s"`
	Delete string `koanf:"delete" desc:"keybind to remove from history" default:"ctrl d"`
}

type DesktopApplications struct {
	Start string `koanf:"start" desc:"keybind for activation" default:"enter"`
}

type Runner struct {
	Start string `koanf:"start" desc:"keybind for activation" default:"enter"`
}

type Symbols struct {
	Copy string `koanf:"start" desc:"keybind for activation" default:"enter"`
}

type Files struct {
	Open     string `koanf:"open" desc:"keybind for opening the file" default:"enter"`
	OpenDir  string `koanf:"open_dir" desc:"keybind for opening parent directory" default:"alt enter"`
	CopyPath string `koanf:"copy_path" desc:"keybind to copy the path" default:"ctrl shift C"`
	CopyFile string `koanf:"copy_file" desc:"keybind to copy file" default:"ctrl c"`
}

type Prefix struct {
	Prefix   string `koanf:"prefix" desc:"prefix" default:""`
	Provider string `koanf:"provider" desc:"provider" default:""`
}

type Clipboard struct {
	TimeFormat string `koanf:"time_format" desc:"format in which the time shoulb be displayed" default:"01.02. - 15:04"`
	Copy       string `koanf:"copy" desc:"keybind to copy result" default:"enter"`
	Delete     string `koanf:"delete" desc:"keybind to remove from history" default:"ctrl d"`
}

var (
	LoadedConfig       *Config
	AvailableProviders = []Provider{}
)

func Load() {
	loadProviderData()

	LoadedConfig = &Config{
		CloseWhenOpen: true,
		Keybinds: Keybinds{
			Close:    "esc",
			Next:     "down",
			Previous: "up",
		},
		Providers: Providers{
			Default: []string{"desktopapplications", "calc", "runner"},
			Empty:   []string{"desktopapplications"},
			Prefixes: []Prefix{
				{
					Prefix:   "/",
					Provider: "files",
				},
				{
					Prefix:   ".",
					Provider: "symbols",
				},
				{
					Prefix:   "=",
					Provider: "calc",
				},
				{
					Prefix:   ":",
					Provider: "clipboard",
				},
			},
			Clipboard: Clipboard{
				TimeFormat: "01.02. - 15:04",
				Copy:       "enter",
				Delete:     "ctrl d",
			},
			Calc: Calc{
				Copy:   "enter",
				Save:   "ctrl s",
				Delete: "ctrl d",
			},
			DesktopApplications: DesktopApplications{
				Start: "enter",
			},
			Files: Files{
				Open:     "enter",
				OpenDir:  "ctrl enter",
				CopyPath: "ctrl shift C",
				CopyFile: "ctrl c",
			},
			Runner: Runner{
				Start: "enter",
			},
			Symbols: Symbols{
				Copy: "enter",
			},
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
