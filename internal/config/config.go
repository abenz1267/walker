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
	ActivationMode      ActivationMode `mapstructure:"activation_mode"`
	Bar                 Bar            `mapstructure:"bar"`
	Builtins            Builtins       `mapstructure:"builtins"`
	DisableClickToClose bool           `mapstructure:"disable_click_to_close"`
	Disabled            []string       `mapstructure:"disabled"`
	ForceKeyboardFocus  bool           `mapstructure:"force_keyboard_focus"`
	AsWindow            bool           `mapstructure:"as_window"`
	HotreloadTheme      bool           `mapstructure:"hotreload_theme"`
	IgnoreMouse         bool           `mapstructure:"ignore_mouse"`
	List                List           `mapstructure:"list"`
	Monitor             string         `mapstructure:"monitor"`
	Plugins             []Plugin       `mapstructure:"plugins"`
	Search              Search         `mapstructure:"search"`
	Terminal            string         `mapstructure:"terminal"`
	Theme               string         `mapstructure:"theme"`
	ThemeBase           []string       `mapstructure:"theme_base"`
	Timeout             int            `mapstructure:"timeout"`

	Available []string `mapstructure:"-"`
	IsService bool     `mapstructure:"-"`
}

type Bar struct {
	Entries []BarEntry `mapstructure:"entries"`
}

type BarEntry struct {
	Exec   string `mapstructure:"exec"`
	Icon   string `mapstructure:"icon"`
	Label  string `mapstructure:"label"`
	Module string `mapstructure:"module"`
}

type Builtins struct {
	Applications   Applications   `mapstructure:"applications"`
	AI             AI             `mapstructure:"ai"`
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
	Symbols        Symbols        `mapstructure:"symbols"`
	Websearch      Websearch      `mapstructure:"websearch"`
	Windows        Windows        `mapstructure:"windows"`
}

type AI struct {
	GeneralModule `mapstructure:",squash"`
	Anthropic     Anthropic `mapstructure:"anthropic"`
}

type Anthropic struct {
	Prompts []AnthropicPrompt `mapstructure:"prompts"`
}

type AnthropicPrompt struct {
	Model            string  `mapstructure:"model"`
	MaxTokens        int     `mapstructure:"max_tokens"`
	Temperature      float64 `mapstructure:"temperature"`
	Label            string  `mapstructure:"label"`
	Prompt           string  `mapstructure:"prompt"`
	SingleModuleOnly bool    `mapstructure:"single_module_only"`
}

type Calc struct {
	GeneralModule `mapstructure:",squash"`
	RequireNumber bool `mapstructure:"require_number"`
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
	AutoSelect         bool     `mapstructure:"auto_select"`
	Delay              int      `mapstructure:"delay"`
	EagerLoading       bool     `mapstructure:"eager_loading"`
	History            bool     `mapstructure:"history"`
	Icon               string   `mapstructure:"icon"`
	KeepSort           bool     `mapstructure:"keep_sort"`
	MinChars           int      `mapstructure:"min_chars"`
	Name               string   `mapstructure:"name"`
	Placeholder        string   `mapstructure:"placeholder"`
	Prefix             string   `mapstructure:"prefix"`
	Refresh            bool     `mapstructure:"refresh"`
	ShowIconWhenSingle bool     `mapstructure:"show_icon_when_single"`
	ShowSubWhenSingle  bool     `mapstructure:"show_sub_when_single"`
	SwitcherOnly       bool     `mapstructure:"switcher_only"`
	Theme              string   `mapstructure:"theme"`
	ThemeBase          []string `mapstructure:"theme_base"`
	Typeahead          bool     `mapstructure:"typeahead"`
	Weight             int      `mapstructure:"weight"`

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
	GeneralModule   `mapstructure:",squash"`
	Exec            string `mapstructure:"exec"`
	ExecAlt         string `mapstructure:"exec_alt"`
	ShowUnqualified bool   `mapstructure:"show_unqualified"`
}

type Symbols struct {
	GeneralModule `mapstructure:",squash"`
	AfterCopy     string `mapstructure:"after_copy"`
}

type SSH struct {
	GeneralModule `mapstructure:",squash"`
	ConfigFile    string `mapstructure:"config_file"`
	HostFile      string `mapstructure:"host_file"`
}

type Websearch struct {
	GeneralModule `mapstructure:",squash"`
	Entries       []WebsearchEntry `mapstructure:"entries"`
}

type WebsearchEntry struct {
	Name         string `mapstructure:"name"`
	Url          string `mapstructure:"url"`
	Prefix       string `mapstructure:"prefix"`
	SwitcherOnly bool   `mapstructure:"switcher_only"`
}

type Applications struct {
	GeneralModule `mapstructure:",squash"`
	Actions       ApplicationActions `mapstructure:"actions"`
	Cache         bool               `mapstructure:"cache"`
	ContextAware  bool               `mapstructure:"context_aware"`
	PrioritizeNew bool               `mapstructure:"prioritize_new"`
	ShowGeneric   bool               `mapstructure:"show_generic"`
}

type ApplicationActions struct {
	Enabled          bool `mapstructure:"enabled"`
	HideCategory     bool `mapstructure:"hide_category"`
	HideWithoutQuery bool `mapstructure:"hide_without_query"`
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
	GeneralModule   `mapstructure:",squash"`
	AvoidLineBreaks bool   `mapstructure:"avoid_line_breaks"`
	ImageHeight     int    `mapstructure:"image_height"`
	MaxEntries      int    `mapstructure:"max_entries"`
	Exec            string `mapstructure:"exec"`
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
	GeneralModule    `mapstructure:",squash"`
	Cmd              string            `mapstructure:"cmd"`
	CmdAlt           string            `mapstructure:"cmd_alt"`
	Entries          []util.Entry      `mapstructure:"entries"`
	LabelColumn      int               `mapstructure:"label_column"`
	Matching         util.MatchingType `mapstructure:"matching"`
	RecalculateScore bool              `mapstructure:"recalculate_score,omitempty" json:"recalculate_score,omitempty"`
	ResultColumn     int               `mapstructure:"result_column"`
	Separator        string            `mapstructure:"separator"`
	Src              string            `mapstructure:"src"`
	SrcOnce          string            `mapstructure:"src_once"`
	Terminal         bool              `mapstructure:"terminal"`
}

type Search struct {
	Delay       int    `mapstructure:"delay"`
	Placeholder string `mapstructure:"placeholder"`
}

type List struct {
	Cycle               bool   `mapstructure:"cycle"`
	KeyboardScrollStyle string `mapstructure:"keyboard_scroll_style"`
	MaxEntries          int    `mapstructure:"max_entries"`
	Placeholder         string `mapstructure:"placeholder"`
	ShowInitialEntries  bool   `mapstructure:"show_initial_entries"`
	SingleClick         bool   `mapstructure:"single_click"`
	VisibilityThreshold int    `mapstructure:"visibility_threshold"`
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

	viper.AutomaticEnv()

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

	for _, v := range t {
		path, _ := exec.LookPath(v)

		if path != "" {
			cfg.Terminal = path
			break
		}
	}
}
