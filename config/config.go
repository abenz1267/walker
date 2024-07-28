package config

import (
	"bytes"
	_ "embed"
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/abenz1267/walker/util"
	"github.com/spf13/viper"
)

var noFoundErr viper.ConfigFileNotFoundError

//go:embed config.default.json
var defaultConfig []byte

type Config struct {
	ActivationMode ActivationMode    `mapstructure:"activation_mode"`
	Builtins       Builtins          `mapstructure:"builtins"`
	Disabled       []string          `mapstructure:"disabled"`
	IgnoreMouse    bool              `mapstructure:"ignore_mouse"`
	List           List              `mapstructure:"list"`
	Plugins        []Plugin          `mapstructure:"plugins"`
	Search         Search            `mapstructure:"search"`
	SpecialLabels  map[string]string `mapstructure:"special_labels"`
	Terminal       string            `mapstructure:"terminal"`
	UI             UI                `mapstructure:"ui"`

	// internal
	Available []string `mapstructure:"-"`
	IsService bool     `mapstructure:"-"`
}

type Builtins struct {
	Applications   Applications   `mapstructure:"applications"`
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
	IsSetup      bool   `mapstructure:"-"`
	History      bool   `mapstructure:"history"`
	Placeholder  string `mapstructure:"placeholder"`
	Prefix       string `mapstructure:"prefix"`
	SpecialLabel string `mapstructure:"special_label"`
	SwitcherOnly bool   `mapstructure:"switcher_only"`
	Typeahead    bool   `mapstructure:"typeahead"`
}

type Finder struct {
	GeneralModule   `mapstructure:",squash"`
	IgnoreGitIgnore bool `mapstructure:"ignore_gitignore"`
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

type ActivationMode struct {
	Disabled bool `mapstructure:"disabled"`
	UseAlt   bool `mapstructure:"use_alt"`
	UseFKeys bool `mapstructure:"use_f_keys"`
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
	KeepSort       bool              `mapstructure:"keep_sort"`
	Matching       util.MatchingType `mapstructure:"matching"`
	Name           string            `mapstructure:"name"`
	Src            string            `mapstructure:"src"`
	SrcOnce        string            `mapstructure:"src_once"`
	SrcOnceRefresh bool              `mapstructure:"src_once_refresh"`
	Entries        []util.Entry      `mapstructure:"entries"`
	Terminal       bool              `mapstructure:"terminal"`
}

type Search struct {
	Delay              int    `mapstructure:"delay"`
	ForceKeyboardFocus bool   `mapstructure:"force_keyboard_focus"`
	Icons              bool   `mapstructure:"icons"`
	MarginSpinner      int    `mapstructure:"margin_spinner"`
	Placeholder        string `mapstructure:"placeholder"`
	Spinner            bool   `mapstructure:"spinner"`
}

type Icons struct {
	Hide             bool   `mapstructure:"hide"`
	ImageSize        int    `mapstructure:"image_size"`
	Size             int    `mapstructure:"size"`
	SizeSingleModule int    `mapstructure:"size_single_module"`
	Theme            string `mapstructure:"theme"`
}

type UI struct {
	Anchors         Anchors `mapstructure:"anchors"`
	Fullscreen      bool    `mapstructure:"fullscreen"`
	Height          int     `mapstructure:"height"`
	Horizontal      string  `mapstructure:"horizontal"`
	Icons           Icons   `mapstructure:"icons"`
	IgnoreExclusive bool    `mapstructure:"ignore_exclusive"`
	Margins         Margins `mapstructure:"margins"`
	Orientation     string  `mapstructure:"orientation"`
	Vertical        string  `mapstructure:"vertical"`
	Width           int     `mapstructure:"width"`
}

type Anchors struct {
	Bottom bool `mapstructure:"bottom"`
	Left   bool `mapstructure:"left"`
	Right  bool `mapstructure:"right"`
	Top    bool `mapstructure:"top"`
}

type Margins struct {
	Bottom int `mapstructure:"bottom"`
	End    int `mapstructure:"end"`
	Start  int `mapstructure:"start"`
	Top    int `mapstructure:"top"`
}

type List struct {
	AlwaysShow          bool   `mapstructure:"always_show"`
	Cycle               bool   `mapstructure:"cycle"`
	FixedHeight         bool   `mapstructure:"fixed_height"`
	Height              int    `mapstructure:"height"`
	HideSub             bool   `mapstructure:"hide_sub"`
	ShowSubSingleModule bool   `mapstructure:"show_sub_single_module"`
	MarginTop           int    `mapstructure:"margin_top"`
	MaxEntries          int    `mapstructure:"max_entries"`
	ScrollbarPolicy     string `mapstructure:"scrollbar_policy"`
	ShowInitialEntries  bool   `mapstructure:"show_initial_entries"`
	Width               int    `mapstructure:"width"`
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

	ft := "json"

	et := os.Getenv("WALKER_CONFIG_TYPE")

	if et != "" {
		ft = et
	}

	err = viper.ReadInConfig()
	if err != nil {
		dErr := os.MkdirAll(util.ConfigDir(), 0755)
		if dErr != nil {
			log.Panicln(dErr)
		}

		if errors.As(err, &noFoundErr) {
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
