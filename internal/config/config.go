package config

import (
	"bytes"
	"embed"
	_ "embed"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"regexp"

	"github.com/abenz1267/walker/internal/util"
	"github.com/spf13/viper"
)

var notFoundErr viper.ConfigFileNotFoundError

//go:embed config.default.json
var defaultConfig []byte

//go:embed themes/*
var Themes embed.FS

type EventType int

const (
	EventLaunch EventType = iota
	EventSelection
	EventExit
	EventActivate
	EventQueryChange
)

type Config struct {
	ActivationMode      ActivationMode `mapstructure:"activation_mode"`
	AsWindow            bool           `mapstructure:"as_window"`
	Bar                 Bar            `mapstructure:"bar"`
	Builtins            Builtins       `mapstructure:"builtins"`
	CloseWhenOpen       bool           `mapstructure:"close_when_open"`
	DisableClickToClose bool           `mapstructure:"disable_click_to_close"`
	Disabled            []string       `mapstructure:"disabled"`
	Events              Events         `mapstructure:"events"`
	ForceKeyboardFocus  bool           `mapstructure:"force_keyboard_focus"`
	HotreloadTheme      bool           `mapstructure:"hotreload_theme"`
	IgnoreMouse         bool           `mapstructure:"ignore_mouse"`
	List                List           `mapstructure:"list"`
	Locale              string         `mapstructure:"locale"`
	Monitor             string         `mapstructure:"monitor"`
	Plugins             []Plugin       `mapstructure:"plugins"`
	Search              Search         `mapstructure:"search"`
	Terminal            string         `mapstructure:"terminal"`
	Theme               string         `mapstructure:"theme"`
	ThemeBase           []string       `mapstructure:"theme_base"`
	Timeout             int            `mapstructure:"timeout"`
	UseUWSM             bool           `mapstructure:"use_uwsm"`

	Available []string `mapstructure:"-"`
	IsService bool     `mapstructure:"-"`
}

type Events struct {
	OnLaunch      string `mapstructure:"on_launch"`
	OnSelection   string `mapstructure:"on_selection"`
	OnExit        string `mapstructure:"on_exit"`
	OnActivate    string `mapstructure:"on_activate"`
	OnQueryChange string `mapstructure:"on_query_change"`
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
	Bookmarks      Bookmarks      `mapstructure:"bookmarks"`
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

type Bookmarks struct {
	GeneralModule `mapstructure:",squash"`
	Groups        []BookmarkGroup `mapstructure:"groups"`
	Entries       []BookmarkEntry `mapstructure:"entries"`
}

type BookmarkGroup struct {
	Label            string          `mapstructure:"label"`
	Prefix           string          `mapstructure:"prefix"`
	IgnoreUnprefixed bool            `mapstructure:"ignore_unprefixed"`
	Entries          []BookmarkEntry `mapstructure:"entries"`
}

type BookmarkEntry struct {
	Label    string   `mapstructure:"label"`
	Url      string   `mapstructure:"url"`
	Keywords []string `mapstructure:"keywords"`
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
	AutoSelect         bool        `mapstructure:"auto_select"`
	Blacklist          []Blacklist `mapstructure:"blacklist"`
	Delay              int         `mapstructure:"delay"`
	History            bool        `mapstructure:"history"`
	Icon               string      `mapstructure:"icon"`
	KeepSort           bool        `mapstructure:"keep_sort"`
	MinChars           int         `mapstructure:"min_chars"`
	Name               string      `mapstructure:"name"`
	Placeholder        string      `mapstructure:"placeholder"`
	Prefix             string      `mapstructure:"prefix"`
	Refresh            bool        `mapstructure:"refresh"`
	ShowIconWhenSingle bool        `mapstructure:"show_icon_when_single"`
	ShowSubWhenSingle  bool        `mapstructure:"show_sub_when_single"`
	SwitcherOnly       bool        `mapstructure:"switcher_only"`
	Theme              string      `mapstructure:"theme"`
	ThemeBase          []string    `mapstructure:"theme_base"`
	Typeahead          bool        `mapstructure:"typeahead"`
	Weight             int         `mapstructure:"weight"`

	// internal
	HasInitialSetup bool `mapstructure:"-"`
	IsSetup         bool `mapstructure:"-"`
}

type Blacklist struct {
	Regexp string `mapstructure:"regexp"`
	Label  bool   `mapstructure:"label"`
	Sub    bool   `mapstructure:"sub"`

	// internal
	Reg *regexp.Regexp `mapstructure:"-"`
}

type Finder struct {
	GeneralModule   `mapstructure:",squash"`
	UseFD           bool `mapstructure:"use_fd"`
	IgnoreGitIgnore bool `mapstructure:"ignore_gitignore"`
	Concurrency     int  `mapstructure:"concurrency"`
	EagerLoading    bool `mapstructure:"eager_loading"`
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
	Parser           string            `mapstructure:"parser"`
	KvSeparator      string            `mapstructure:"kv_separator"`
	Output           bool              `mapstructure:"output"`
	Keywords         []string          `mapstructure:"keywords"`
}

type Search struct {
	Delay           int    `mapstructure:"delay"`
	Placeholder     string `mapstructure:"placeholder"`
	ResumeLastQuery bool   `mapstructure:"resume_last_query"`
}

type List struct {
	Cycle               bool   `mapstructure:"cycle"`
	DynamicSub          bool   `mapstructure:"dynamic_sub"`
	KeyboardScrollStyle string `mapstructure:"keyboard_scroll_style"`
	MaxEntries          int    `mapstructure:"max_entries"`
	Placeholder         string `mapstructure:"placeholder"`
	ShowInitialEntries  bool   `mapstructure:"show_initial_entries"`
	SingleClick         bool   `mapstructure:"single_click"`
	VisibilityThreshold int    `mapstructure:"visibility_threshold"`
}

func Get(config string) (*Config, error) {
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

		if errors.As(err, &notFoundErr) {
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
		slog.Error("config", "error", err)
		return nil, err
	}

	go setTerminal(cfg)

	return cfg, nil
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
