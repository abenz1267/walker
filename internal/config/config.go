package config

import (
	"bytes"
	"embed"
	_ "embed"
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/abenz1267/walker/internal/util"
	"github.com/spf13/viper"
)

var noFoundErr viper.ConfigFileNotFoundError

//go:embed config.default.json
var defaultConfig []byte

//go:embed themes/*
var Themes embed.FS

type Config struct {
	ActivationMode ActivationMode `mapstructure:"activation_mode"`
	Builtins       Builtins       `mapstructure:"builtins"`
	Disabled       []string       `mapstructure:"disabled"`
	IgnoreMouse    bool           `mapstructure:"ignore_mouse"`
	List           List           `mapstructure:"list"`
	Plugins        []Plugin       `mapstructure:"plugins"`
	Search         Search         `mapstructure:"search"`
	Theme          string         `mapstructure:"theme"`
	ThemeBase      []string       `mapstructure:"theme_base"`
	Terminal       string         `mapstructure:"terminal"`

	Available []string `mapstructure:"-"`
	IsService bool     `mapstructure:"-"`
}

type Builtins struct {
	Applications   Applications   `mapstructure:"applications"`
	Calc           Calc           `mapstructure:"calc"`
	Clipboard      Clipboard      `mapstructure:"clipboard"`
	Commands       Commands       `mapstructure:"commands"`
	CustomCommands CustomCommands `mapstructure:"custom_commands"`
	Dmenu          Dmenu          `mapstructure:"dmenu"`
	Emojis         Emojis         `mapstructure:"emojis"`
	Finder         Finder         `mapstructure:"finder"`
	Runner         Runner         `mapstructure:"runner"`
	SSH            SSH            `mapstructure:"ssh"`
	Switcher       Switcher       `mapstructure:"switcher"`
	Websearch      Websearch      `mapstructure:"websearch"`
	Windows        Windows        `mapstructure:"windows"`
}

type Calc struct {
	GeneralModule `mapstructure:",squash"`
}

type CustomCommands struct {
	GeneralModule `mapstructure:",squash"`
	Commands      []CustomCommand `mapstructure:"commands"`
}

type CustomCommand struct {
	Cmd      string `mapstructure:"cmd"`
	CmdAlt   string `mapstructure:"cmd_alt"`
	Name     string `mapstructure:"name"`
	Terminal bool   `mapstructure:"terminal"`
}

type GeneralModule struct {
	Delay              int      `mapstructure:"delay"`
	EagerLoading       bool     `mapstructure:"eager_loading"`
	History            bool     `mapstructure:"history"`
	ShowIconWhenSingle bool     `mapstructure:"show_icon_when_single"`
	Icon               string   `mapstructure:"icon"`
	KeepSort           bool     `mapstructure:"keep_sort"`
	Name               string   `mapstructure:"name"`
	Placeholder        string   `mapstructure:"placeholder"`
	Prefix             string   `mapstructure:"prefix"`
	Refresh            bool     `mapstructure:"refresh"`
	SwitcherOnly       bool     `mapstructure:"switcher_only"`
	Theme              string   `mapstructure:"theme"`
	ThemeBase          []string `mapstructure:"theme_base"`
	Typeahead          bool     `mapstructure:"typeahead"`
	ShowSubWhenSingle  bool     `mapstructure:"show_sub_when_single"`

	// internal
	HasInitialSetup bool `mapstructure:"-"`
	IsSetup         bool `mapstructure:"-"`
}

type Finder struct {
	GeneralModule   `mapstructure:",squash"`
	IgnoreGitIgnore bool `mapstructure:"ignore_gitignore"`
	Concurrency     int  `mapstructure:"concurrency"`
}

type Commands struct {
	GeneralModule `mapstructure:",squash"`
}

type Switcher struct {
	GeneralModule `mapstructure:",squash"`
}

type Emojis struct {
	GeneralModule `mapstructure:",squash"`
}

type SSH struct {
	GeneralModule `mapstructure:",squash"`
	ConfigFile    string `mapstructure:"config_file"`
	HostFile      string `mapstructure:"host_file"`
}

type Websearch struct {
	GeneralModule `mapstructure:",squash"`
	Engines       []string `mapstructure:"engines"`
}

type Applications struct {
	GeneralModule `mapstructure:",squash"`
	Actions       bool `mapstructure:"actions"`
	Cache         bool `mapstructure:"cache"`
	PrioritizeNew bool `mapstructure:"prioritize_new"`
	ContextAware  bool `mapstructure:"context_aware"`
}

type Windows struct {
	GeneralModule `mapstructure:",squash"`
}

type ActivationMode struct {
	Disabled bool   `mapstructure:"disabled"`
	Labels   string `mapstructure:"labels"`
	UseAlt   bool   `mapstructure:"use_alt"`
	UseFKeys bool   `mapstructure:"use_f_keys"`
}

type Clipboard struct {
	GeneralModule `mapstructure:",squash"`
	ImageHeight   int `mapstructure:"image_height"`
	MaxEntries    int `mapstructure:"max_entries"`
}

type Dmenu struct {
	GeneralModule `mapstructure:",squash"`
	Separator     string `mapstructure:"separator"`
	LabelColumn   int    `mapstructure:"label_column"`
}

type Runner struct {
	GeneralModule `mapstructure:",squash"`
	Excludes      []string `mapstructure:"excludes"`
	Includes      []string `mapstructure:"includes"`
	ShellConfig   string   `mapstructure:"shell_config"`
	GenericEntry  bool     `mapstructure:"generic_entry"`
}

type Plugin struct {
	GeneralModule  `mapstructure:",squash"`
	Cmd            string            `mapstructure:"cmd"`
	CmdAlt         string            `mapstructure:"cmd_alt"`
	Matching       util.MatchingType `mapstructure:"matching"`
	Src            string            `mapstructure:"src"`
	SrcOnce        string            `mapstructure:"src_once"`
	SrcOnceRefresh bool              `mapstructure:"src_once_refresh"`
	Entries        []util.Entry      `mapstructure:"entries"`
	Terminal       bool              `mapstructure:"terminal"`
}

type Search struct {
	Delay       int    `mapstructure:"delay"`
	Placeholder string `mapstructure:"placeholder"`
}

type List struct {
	Cycle              bool `mapstructure:"cycle"`
	MaxEntries         int  `mapstructure:"max_entries"`
	ShowInitialEntries bool `mapstructure:"show_initial_entries"`
	SingleClick        bool `mapstructure:"single_click"`
}

func Get(config string) *Config {
	os.MkdirAll(util.ThemeDir(), 0755)

	defs := viper.New()
	defs.SetConfigType("json")

	err := defs.ReadConfig(bytes.NewBuffer(defaultConfig))
	if err != nil {
		log.Panicln(err)
	}

	for k, v := range defs.AllSettings() {
		viper.SetDefault(k, v)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(util.ConfigDir())

	err = viper.ReadInConfig()
	if err != nil {
		dErr := os.MkdirAll(util.ConfigDir(), 0755)
		if dErr != nil {
			log.Panicln(dErr)
		}

		if errors.As(err, &noFoundErr) {
			ft := "json"

			et := os.Getenv("WALKER_CONFIG_TYPE")

			if et != "" {
				ft = et
			}

			viper.SetConfigType(ft)
			wErr := viper.SafeWriteConfig()
			if wErr != nil {
				log.Println(wErr)
			}
		} else {
			log.Panicln(err)
		}
	}

	cfg := &Config{}

	err = viper.Unmarshal(cfg)
	if err != nil {
		log.Panic(err)
	}

	go setTerminal(cfg)

	return cfg
}

func setTerminal(cfg *Config) {
	if cfg.Terminal != "" {
		path, _ := exec.LookPath(cfg.Terminal)

		if path != "" {
			cfg.Terminal = path
		}

		return
	}

	envVars := []string{"TERM", "TERMINAL"}

	for _, v := range envVars {
		term, ok := os.LookupEnv(v)
		if ok {
			path, _ := exec.LookPath(term)

			if path != "" {
				cfg.Terminal = path
				return
			}
		}
	}

	t := []string{
		"Eterm",
		"alacritty",
		"aterm",
		"foot",
		"gnome-terminal",
		"guake",
		"hyper",
		"kitty",
		"konsole",
		"lilyterm",
		"lxterminal",
		"mate-terminal",
		"qterminal",
		"roxterm",
		"rxvt",
		"st",
		"terminator",
		"terminix",
		"terminology",
		"termit",
		"termite",
		"tilda",
		"tilix",
		"urxvt",
		"uxterm",
		"wezterm",
		"x-terminal-emulator",
		"xfce4-terminal",
		"xterm",
	}

	term, ok := os.LookupEnv("TERM")
	if ok {
		t = append([]string{term}, t...)
	}

	terminal, ok := os.LookupEnv("TERMINAL")
	if ok {
		t = append([]string{terminal}, t...)
	}

	for _, v := range t {
		path, _ := exec.LookPath(v)

		if path != "" {
			cfg.Terminal = path
			break
		}
	}
}
