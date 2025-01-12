package config

import (
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/abenz1267/walker/internal/util"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

//go:embed config.default.toml
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
	ActivationMode      ActivationMode `koanf:"activation_mode"`
	AsWindow            bool           `koanf:"as_window"`
	Bar                 Bar            `koanf:"bar"`
	Builtins            Builtins       `koanf:"builtins"`
	CloseWhenOpen       bool           `koanf:"close_when_open"`
	DisableClickToClose bool           `koanf:"disable_click_to_close"`
	Keys                Keys           `koanf:"keys"`
	Disabled            []string       `koanf:"disabled"`
	Events              Events         `koanf:"events"`
	ForceKeyboardFocus  bool           `koanf:"force_keyboard_focus"`
	HotreloadTheme      bool           `koanf:"hotreload_theme"`
	IgnoreMouse         bool           `koanf:"ignore_mouse"`
	AppLaunchPrefix     string         `koanf:"app_launch_prefix"`
	List                List           `koanf:"list"`
	Locale              string         `koanf:"locale"`
	Monitor             string         `koanf:"monitor"`
	Plugins             []Plugin       `koanf:"plugins"`
	Search              Search         `koanf:"search"`
	Terminal            string         `koanf:"terminal"`
	TerminalTitleFlag   string         `koanf:"terminal_title_flag"`
	Theme               string         `koanf:"theme"`
	ThemeBase           []string       `koanf:"theme_base"`
	Timeout             int            `koanf:"timeout"`

	Available []string `koanf:"-"`
	Hidden    []string `koanf:"-"`
	IsService bool     `koanf:"-"`
}

type Keys struct {
	AcceptTypeahead     []string            `koanf:"accept_typeahead"`
	ActivationModifiers ActivationModifiers `koanf:"activation_modifiers"`
	TriggerLabels       string              `koanf:"trigger_labels"`
	Ai                  AiKeys              `koanf:"ai"`
	Close               []string            `koanf:"close"`
	Next                []string            `koanf:"next"`
	Prev                []string            `koanf:"prev"`
	RemoveFromHistory   []string            `koanf:"remove_from_history"`
	ResumeQuery         []string            `koanf:"resume_query"`
	ToggleExactSearch   []string            `koanf:"toggle_exact_search"`
}

type ActivationModifiers struct {
	KeepOpen  string `koanf:"keep_open"`
	Alternate string `koanf:"alternate"`
}

type AiKeys struct {
	ClearSession     []string `koanf:"clear_session"`
	CopyLastResponse []string `koanf:"copy_last_response"`
	ResumeSession    []string `koanf:"resume_session"`
	RunLastResponse  []string `koanf:"run_last_response"`
}

type Events struct {
	OnLaunch      string `koanf:"on_launch"`
	OnSelection   string `koanf:"on_selection"`
	OnExit        string `koanf:"on_exit"`
	OnActivate    string `koanf:"on_activate"`
	OnQueryChange string `koanf:"on_query_change"`
}

type Bar struct {
	Entries []BarEntry `koanf:"entries"`
}

type BarEntry struct {
	Exec   string `koanf:"exec"`
	Icon   string `koanf:"icon"`
	Label  string `koanf:"label"`
	Module string `koanf:"module"`
}

type Builtins struct {
	Applications   Applications   `koanf:"applications"`
	AI             AI             `koanf:"ai"`
	Bookmarks      Bookmarks      `koanf:"bookmarks"`
	Calc           Calc           `koanf:"calc"`
	Clipboard      Clipboard      `koanf:"clipboard"`
	Commands       Commands       `koanf:"commands"`
	CustomCommands CustomCommands `koanf:"custom_commands"`
	Dmenu          Dmenu          `koanf:"dmenu"`
	Emojis         Emojis         `koanf:"emojis"`
	Finder         Finder         `koanf:"finder"`
	Runner         Runner         `koanf:"runner"`
	SSH            SSH            `koanf:"ssh"`
	Switcher       Switcher       `koanf:"switcher"`
	Symbols        Symbols        `koanf:"symbols"`
	Websearch      Websearch      `koanf:"websearch"`
	Windows        Windows        `koanf:"windows"`
	XdphPicker     XdphPicker     `koanf:"xdph_picker"`
	Translation    Translation    `koanf:"translation"`
}

type XdphPicker struct {
	GeneralModule `koanf:",squash"`
}

type Bookmarks struct {
	GeneralModule `koanf:",squash"`
	Groups        []BookmarkGroup `koanf:"groups"`
	Entries       []BookmarkEntry `koanf:"entries"`
}

type BookmarkGroup struct {
	Label            string          `koanf:"label"`
	Prefix           string          `koanf:"prefix"`
	IgnoreUnprefixed bool            `koanf:"ignore_unprefixed"`
	Entries          []BookmarkEntry `koanf:"entries"`
}

type BookmarkEntry struct {
	Label    string   `koanf:"label"`
	Url      string   `koanf:"url"`
	Keywords []string `koanf:"keywords"`
}

type AI struct {
	GeneralModule `koanf:",squash"`
	Anthropic     Anthropic `koanf:"anthropic"`
}

type Anthropic struct {
	Prompts []AnthropicPrompt `koanf:"prompts"`
}

type AnthropicPrompt struct {
	Model            string  `koanf:"model"`
	MaxTokens        int     `koanf:"max_tokens"`
	Temperature      float64 `koanf:"temperature"`
	Label            string  `koanf:"label"`
	Prompt           string  `koanf:"prompt"`
	SingleModuleOnly bool    `koanf:"single_module_only"`
}

type Calc struct {
	GeneralModule `koanf:",squash"`
	RequireNumber bool `koanf:"require_number"`
}

type CustomCommands struct {
	GeneralModule `koanf:",squash"`
	Commands      []CustomCommand `koanf:"commands"`
}

type CustomCommand struct {
	Cmd               string   `koanf:"cmd"`
	CmdAlt            string   `koanf:"cmd_alt"`
	Env               []string `koanf:"env"`
	Name              string   `koanf:"name"`
	Path              string   `koanf:"path"`
	Terminal          bool     `koanf:"terminal"`
	TerminalTitleFlag string   `koanf:"terminal_title_flag"`
}

type GeneralModule struct {
	AutoSelect         bool        `koanf:"auto_select"`
	Blacklist          []Blacklist `koanf:"blacklist"`
	Delay              int         `koanf:"delay"`
	ExternalConfig     bool        `koanf:"external_config"`
	Hidden             bool        `koanf:"hidden"`
	History            bool        `koanf:"history"`
	Icon               string      `koanf:"icon"`
	KeepSort           bool        `koanf:"keep_sort"`
	MinChars           int         `koanf:"min_chars"`
	Name               string      `koanf:"name"`
	Placeholder        string      `koanf:"placeholder"`
	Prefix             string      `koanf:"prefix"`
	Refresh            bool        `koanf:"refresh"`
	ShowIconWhenSingle bool        `koanf:"show_icon_when_single"`
	ShowSubWhenSingle  bool        `koanf:"show_sub_when_single"`
	SwitcherOnly       bool        `koanf:"switcher_only"`
	Theme              string      `koanf:"theme"`
	ThemeBase          []string    `koanf:"theme_base"`
	Typeahead          bool        `koanf:"typeahead"`
	Weight             int         `koanf:"weight"`
	OnSelect           string      `koanf:"on_select"`
	OutputPlaceholder  string      `koanf:"output_placeholder"`

	// internal
	HasInitialSetup bool `koanf:"-"`
	IsSetup         bool `koanf:"-"`
}

type Blacklist struct {
	Regexp string `koanf:"regexp"`
	Label  bool   `koanf:"label"`
	Sub    bool   `koanf:"sub"`

	// internal
	Reg *regexp.Regexp `koanf:"-"`
}

type Finder struct {
	GeneralModule   `koanf:",squash"`
	UseFD           bool `koanf:"use_fd"`
	IgnoreGitIgnore bool `koanf:"ignore_gitignore"`
	Concurrency     int  `koanf:"concurrency"`
	EagerLoading    bool `koanf:"eager_loading"`
}

type Commands struct {
	GeneralModule `koanf:",squash"`
}

type Switcher struct {
	GeneralModule `koanf:",squash"`
}

type Emojis struct {
	GeneralModule   `koanf:",squash"`
	Exec            string `koanf:"exec"`
	ExecAlt         string `koanf:"exec_alt"`
	ShowUnqualified bool   `koanf:"show_unqualified"`
}

type Symbols struct {
	GeneralModule `koanf:",squash"`
	AfterCopy     string `koanf:"after_copy"`
}

type SSH struct {
	GeneralModule `koanf:",squash"`
	ConfigFile    string `koanf:"config_file"`
	HostFile      string `koanf:"host_file"`
}

type Websearch struct {
	GeneralModule `koanf:",squash"`
	Entries       []WebsearchEntry `koanf:"entries"`
}

type WebsearchEntry struct {
	Name         string `koanf:"name"`
	Url          string `koanf:"url"`
	Prefix       string `koanf:"prefix"`
	SwitcherOnly bool   `koanf:"switcher_only"`
}

type Translation struct {
	GeneralModule `koanf:",squash"`
	Providers     []string `koanf:"providers"`
}

type Applications struct {
	GeneralModule `koanf:",squash"`
	Actions       ApplicationActions `koanf:"actions"`
	Cache         bool               `koanf:"cache"`
	ContextAware  bool               `koanf:"context_aware"`
	PrioritizeNew bool               `koanf:"prioritize_new"`
	ShowGeneric   bool               `koanf:"show_generic"`
}

type ApplicationActions struct {
	Enabled          bool `koanf:"enabled"`
	HideCategory     bool `koanf:"hide_category"`
	HideWithoutQuery bool `koanf:"hide_without_query"`
}

type Windows struct {
	GeneralModule `koanf:",squash"`
}

type ActivationMode struct {
	Disabled bool   `koanf:"disabled"`
	Labels   string `koanf:"labels"`
	UseFKeys bool   `koanf:"use_f_keys"`
}

type Clipboard struct {
	GeneralModule   `koanf:",squash"`
	AvoidLineBreaks bool   `koanf:"avoid_line_breaks"`
	ImageHeight     int    `koanf:"image_height"`
	MaxEntries      int    `koanf:"max_entries"`
	Exec            string `koanf:"exec"`
}

type Dmenu struct {
	GeneralModule `koanf:",squash"`
	Separator     string `koanf:"separator"`
	LabelColumn   int    `koanf:"label_column"`
}

type Runner struct {
	GeneralModule `koanf:",squash"`
	Excludes      []string `koanf:"excludes"`
	Includes      []string `koanf:"includes"`
	ShellConfig   string   `koanf:"shell_config"`
	GenericEntry  bool     `koanf:"generic_entry"`
}

type Plugin struct {
	GeneralModule    `koanf:",squash"`
	Cmd              string            `koanf:"cmd"`
	CmdAlt           string            `koanf:"cmd_alt"`
	Entries          []util.Entry      `koanf:"entries"`
	LabelColumn      int               `koanf:"label_column"`
	Matching         util.MatchingType `koanf:"matching"`
	RecalculateScore bool              `koanf:"recalculate_score,omitempty"`
	ResultColumn     int               `koanf:"result_column"`
	Separator        string            `koanf:"separator"`
	Src              string            `koanf:"src"`
	SrcOnce          string            `koanf:"src_once"`
	Terminal         bool              `koanf:"terminal"`
	Parser           string            `koanf:"parser"`
	KvSeparator      string            `koanf:"kv_separator"`
	Output           bool              `koanf:"output"`
	Keywords         []string          `koanf:"keywords"`
}

type Search struct {
	ArgumentDelimiter string `koanf:"argument_delimiter"`
	Delay             int    `koanf:"delay"`
	Placeholder       string `koanf:"placeholder"`
	ResumeLastQuery   bool   `koanf:"resume_last_query"`
}

type List struct {
	Cycle               bool   `koanf:"cycle"`
	DynamicSub          bool   `koanf:"dynamic_sub"`
	KeyboardScrollStyle string `koanf:"keyboard_scroll_style"`
	MaxEntries          int    `koanf:"max_entries"`
	Placeholder         string `koanf:"placeholder"`
	ShowInitialEntries  bool   `koanf:"show_initial_entries"`
	SingleClick         bool   `koanf:"single_click"`
	VisibilityThreshold int    `koanf:"visibility_threshold"`
}

var Cfg *Config

var (
	tomlFile = filepath.Join(util.ConfigDir(), "config.toml")
	jsonFile = filepath.Join(util.ConfigDir(), "config.json")
	yamlFile = filepath.Join(util.ConfigDir(), "config.yaml")
)

func init() {
	os.MkdirAll(util.ConfigDir(), 0755)

	if !util.FileExists(tomlFile) && !util.FileExists(jsonFile) && !util.FileExists(yamlFile) {
		err := os.WriteFile(tomlFile, defaultConfig, 0o600)
		if err != nil {
			slog.Error("Couldn't create config file", "err", err)
		}
	}
}

func Get(config string) error {
	defaults := koanf.New(".")
	err := defaults.Load(rawbytes.Provider(defaultConfig), toml.Parser())
	if err != nil {
		return err
	}

	usrCfg, usrCfgErr := parseConfigFile("config")

	defaults.Merge(usrCfg)

	b := defaults.Get("builtins")

	if builtins, ok := b.(map[string]interface{}); ok {
		for module, v := range builtins {
			if gm, ok := v.(map[string]interface{}); ok {
				for k, v := range gm {
					if k == "external_config" {
						if v == true {
							var cfgFile *koanf.Koanf

							cfgFile, usrCfgErr = parseConfigFile(module)
							if err == nil {
								defaults.MergeAt(cfgFile, fmt.Sprintf("builtins.%s", module))
							}
						}
					}
				}
			}
		}
	}

	parsed := &Config{}

	marshallErr := defaults.Unmarshal("", parsed)

	if marshallErr != nil || usrCfgErr != nil {
		defaults = koanf.New(".")
		_ = defaults.Load(rawbytes.Provider(defaultConfig), toml.Parser())
		_ = defaults.Unmarshal("", parsed)
	}

	Cfg = parsed

	setTerminal()

	if Cfg.Terminal == "" {
		return errors.New("Couldn't determine terminal, try setting terminal explicitly in config")
	}

	if marshallErr == nil {
		return usrCfgErr
	}

	return marshallErr
}

func setTerminal() {
	if Cfg.Terminal != "" {
		path, _ := exec.LookPath(Cfg.Terminal)

		if path != "" {
			Cfg.Terminal = path
		}

		return
	}

	envVars := []string{"TERM", "TERMINAL"}

	for _, v := range envVars {
		term, ok := os.LookupEnv(v)
		if ok {
			path, _ := exec.LookPath(term)

			if path != "" {
				Cfg.Terminal = path
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
		"ghostty",
	}

	for _, v := range t {
		path, _ := exec.LookPath(v)

		if path != "" {
			Cfg.Terminal = path
			break
		}
	}
}

func parseConfigFile(name string) (*koanf.Koanf, error) {
	tomlFile := filepath.Join(util.ConfigDir(), fmt.Sprintf("%s.toml", name))
	jsonFile := filepath.Join(util.ConfigDir(), fmt.Sprintf("%s.json", name))
	yamlFile := filepath.Join(util.ConfigDir(), fmt.Sprintf("%s.yaml", name))

	var usrCfgErr error

	config := koanf.New(".")

	if util.FileExists(tomlFile) {
		usrCfgErr = config.Load(file.Provider(tomlFile), toml.Parser())
	} else if util.FileExists(jsonFile) {
		usrCfgErr = config.Load(file.Provider(jsonFile), json.Parser())
	} else if util.FileExists(yamlFile) {
		usrCfgErr = config.Load(file.Provider(yamlFile), yaml.Parser())
	} else {
		return nil, errors.New("Couldn't find config file")
	}

	return config, usrCfgErr
}
