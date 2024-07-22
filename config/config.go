package config

import (
	"bytes"
	_ "embed"
	"log"
	"os"
	"os/exec"

	"github.com/abenz1267/walker/util"
	"github.com/spf13/viper"
)

//go:embed config.default.json
var defaultConfig []byte

type Config struct {
	Terminal       string            `mapstructure:"terminal"`
	IgnoreMouse    bool              `mapstructure:"ignore_mouse"`
	SpecialLabels  map[string]string `mapstructure:"special_labels"`
	UI             UI                `mapstructure:"ui"`
	List           List              `mapstructure:"list"`
	Search         Search            `mapstructure:"search"`
	ActivationMode ActivationMode    `mapstructure:"activation_mode"`
	Disabled       []string          `mapstructure:"disabled"`
	Plugins        []Plugin          `mapstructure:"plugins"`
	Builtins       Builtins          `mapstructure:"builtins"`

	// internal
	IsService bool     `mapstructure:"-"`
	Enabled   []string `mapstructure:"-"`
}

type Builtins struct {
	Applications   Applications   `mapstructure:"applications"`
	Clipboard      Clipboard      `mapstructure:"clipboard"`
	Commands       Commands       `mapstructure:"commands"`
	CustomCommands CustomCommands `mapstructure:"custom_commands"`
	Emojis         Emojis         `mapstructure:"emojis"`
	Finder         Finder         `mapstructure:"finder"`
	Hyprland       Hyprland       `mapstructure:"hyprland"`
	Runner         Runner         `mapstructure:"runner"`
	SSH            SSH            `mapstructure:"ssh"`
	Switcher       Switcher       `mapstructure:"switcher"`
	Websearch      Websearch      `mapstructure:"websearch"`
}

type CustomCommands struct {
	GeneralModule `mapstructure:",squash"`
	Commands      []CustomCommand `mapstructure:"commands"`
}

type CustomCommand struct {
	Name     string `mapstructure:"name"`
	Cmd      string `mapstructure:"cmd"`
	CmdAlt   string `mapstructure:"cmd_alt"`
	Terminal bool   `mapstructure:"terminal"`
}

type GeneralModule struct {
	IsSetup      bool   `mapstructure:"-"`
	Placeholder  string `mapstructure:"placeholder"`
	Prefix       string `mapstructure:"prefix"`
	SpecialLabel string `mapstructure:"special_label"`
	SwitcherOnly bool   `mapstructure:"switcher_only"`
}

type Finder struct {
	GeneralModule `mapstructure:",squash"`
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
	HostFile      string `mapstructure:"host_file"`
	ConfigFile    string `mapstructure:"config_file"`
}

type Websearch struct {
	GeneralModule `mapstructure:",squash"`
	Engines       []string `mapstructure:"engines"`
}

type Hyprland struct {
	GeneralModule       `mapstructure:",squash"`
	ContextAwareHistory bool `mapstructure:"context_aware_history"`
}

type Applications struct {
	GeneralModule `mapstructure:",squash"`
	Cache         bool `mapstructure:"cache"`
	Actions       bool `mapstructure:"actions"`
}

type ActivationMode struct {
	UseAlt   bool `mapstructure:"use_alt"`
	Disabled bool `mapstructure:"disabled"`
	UseFKeys bool `mapstructure:"use_f_keys"`
}

type Clipboard struct {
	GeneralModule `mapstructure:",squash"`
	ImageHeight   int `mapstructure:"image_height"`
	MaxEntries    int `mapstructure:"max_entries"`
}

type Runner struct {
	GeneralModule `mapstructure:",squash"`
	ShellConfig   string   `mapstructure:"shell_config"`
	Excludes      []string `mapstructure:"excludes"`
	Includes      []string `mapstructure:"includes"`
}

type Plugin struct {
	GeneralModule  `mapstructure:",squash"`
	Name           string            `mapstructure:"name"`
	SrcOnce        string            `mapstructure:"src_once"`
	SrcOnceRefresh bool              `mapstructure:"src_once_refresh"`
	Src            string            `mapstructure:"src"`
	Cmd            string            `mapstructure:"cmd"`
	CmdAlt         string            `mapstructure:"cmd_alt"`
	Terminal       bool              `mapstructure:"terminal"`
	KeepSort       bool              `mapstructure:"keep_sort"`
	Matching       util.MatchingType `mapstructure:"matching"`
}

type Search struct {
	Delay              int    `mapstructure:"delay"`
	Typeahead          bool   `mapstructure:"typeahead"`
	ForceKeyboardFocus bool   `mapstructure:"force_keyboard_focus"`
	Icons              bool   `mapstructure:"icons"`
	Spinner            bool   `mapstructure:"spinner"`
	History            bool   `mapstructure:"history"`
	MarginSpinner      int    `mapstructure:"margin_spinner"`
	Placeholder        string `mapstructure:"placeholder"`
}

type Icons struct {
	Hide      bool   `mapstructure:"hide"`
	Size      int    `mapstructure:"size"`
	ImageSize int    `mapstructure:"image_size"`
	Theme     string `mapstructure:"theme"`
}

type UI struct {
	Icons           Icons   `mapstructure:"icons"`
	Orientation     string  `mapstructure:"orientation"`
	Fullscreen      bool    `mapstructure:"fullscreen"`
	IgnoreExclusive bool    `mapstructure:"ignore_exclusive"`
	Horizontal      string  `mapstructure:"horizontal"`
	Vertical        string  `mapstructure:"vertical"`
	Width           int     `mapstructure:"width"`
	Height          int     `mapstructure:"height"`
	Margins         Margins `mapstructure:"margins"`
	Anchors         Anchors `mapstructure:"anchors"`
}

type Anchors struct {
	Top    bool `mapstructure:"top"`
	Left   bool `mapstructure:"left"`
	Right  bool `mapstructure:"right"`
	Bottom bool `mapstructure:"bottom"`
}

type Margins struct {
	Top    int `mapstructure:"top"`
	Bottom int `mapstructure:"bottom"`
	End    int `mapstructure:"end"`
	Start  int `mapstructure:"start"`
}

type List struct {
	AlwaysShow         bool   `mapstructure:"always_show"`
	Cycle              bool   `mapstructure:"cycle"`
	FixedHeight        bool   `mapstructure:"fixed_height"`
	Height             int    `mapstructure:"height"`
	HideSub            bool   `mapstructure:"hide_sub"`
	MarginTop          int    `mapstructure:"margin_top"`
	MaxEntries         int    `mapstructure:"max_entries"`
	ScrollbarPolicy    string `mapstructure:"scrollbar_policy"`
	ShowInitialEntries bool   `mapstructure:"show_initial_entries"`
	Width              int    `mapstructure:"width"`
}

func Get(config string) *Config {
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

	// ignore error.
	err = viper.ReadInConfig()
	if err != nil {
		viper.SetConfigType("json")
		err := viper.SafeWriteConfig()
		if err != nil {
			log.Println(err)
		}
	}

	cfg := &Config{}

	err = viper.Unmarshal(cfg)
	if err != nil {
		log.Panic(err)
	}

	go setTerminal(cfg)

	// defaults
	if cfg.List.MaxEntries == 0 {
		cfg.List.MaxEntries = 50
	}

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

	t := []string{
		"x-terminal-emulator",
		"mate-terminal",
		"gnome-terminal",
		"terminator",
		"xfce4-terminal",
		"urxvt",
		"rxvt",
		"termit",
		"Eterm",
		"aterm",
		"uxterm",
		"xterm",
		"roxterm",
		"termite",
		"lxterminal",
		"terminology",
		"st",
		"qterminal",
		"lilyterm",
		"tilix",
		"terminix",
		"konsole",
		"foot",
		"kitty",
		"guake",
		"tilda",
		"alacritty",
		"hyper",
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
