package config

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/abenz1267/walker/util"
)

//go:embed config.default.json
var defaultConfig []byte

type Config struct {
	Terminal       string            `json:"terminal,omitempty"`
	IgnoreMouse    bool              `json:"ignore_mouse,omitempty"`
	SpecialLabels  map[string]string `json:"special_labels,omitempty"`
	UI             UI                `json:"ui,omitempty"`
	List           List              `json:"list,omitempty"`
	Search         Search            `json:"search,omitempty"`
	ActivationMode ActivationMode    `json:"activation_mode,omitempty"`
	Disabled       []string          `json:"disabled,omitempty"`
	Plugins        []Plugin          `json:"plugins,omitempty"`
	Builtins       Builtins          `json:"builtins,omitempty"`

	// internal
	IsService bool     `json:"-"`
	Enabled   []string `json:"-"`
}

type Builtins struct {
	Applications Applications `json:"applications,omitempty"`
	Clipboard    Clipboard    `json:"clipboard,omitempty"`
	Commands     Commands     `json:"commands,omitempty"`
	Emojis       Emojis       `json:"emojis,omitempty"`
	Finder       Finder       `json:"finder,omitempty"`
	Hyprland     Hyprland     `json:"hyprland,omitempty"`
	Runner       Runner       `json:"runner,omitempty"`
	SSH          SSH          `json:"ssh,omitempty"`
	Switcher     Switcher     `json:"switcher,omitempty"`
	Websearch    Websearch    `json:"websearch,omitempty"`
}

type GeneralModule struct {
	SpecialLabel string `json:"special_label,omitempty"`
	Prefix       string `json:"prefix,omitempty"`
	SwitcherOnly bool   `json:"switcher_only,omitempty"`
}

type Finder struct {
	GeneralModule
}

type Commands struct {
	GeneralModule
}

type Switcher struct {
	GeneralModule
}

type Emojis struct {
	GeneralModule
}

type SSH struct {
	GeneralModule
	HostFile string `json:"host_file,omitempty"`
}

type Websearch struct {
	GeneralModule
	Engines []string `json:"engines,omitempty"`
}

type Hyprland struct {
	GeneralModule
	ContextAwareHistory bool `json:"context_aware_history,omitempty"`
}

type Applications struct {
	GeneralModule
	Cache   bool `json:"cache,omitempty"`
	Actions bool `json:"actions,omitempty"`
}

type ActivationMode struct {
	UseAlt   bool `json:"use_alt,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
	UseFKeys bool `json:"use_f_keys,omitempty"`
}

type Clipboard struct {
	GeneralModule
	ImageHeight int `json:"image_height,omitempty"`
	MaxEntries  int `json:"max_entries,omitempty"`
}

type Runner struct {
	GeneralModule
	ShellConfig string `json:"shell_config,omitempty"`
	Excludes    []string
	Includes    []string
}

type Plugin struct {
	GeneralModule
	Name           string `json:"name,omitempty"`
	SrcOnce        string `json:"src_once,omitempty"`
	SrcOnceRefresh bool   `json:"src_once_refresh,omitempty"`
	Src            string `json:"src,omitempty"`
	Cmd            string `json:"cmd,omitempty"`
	CmdAlt         string `json:"cmd_alt,omitempty"`
	History        bool   `json:"history,omitempty"`
	Terminal       bool   `json:"terminal,omitempty"`
}

type Search struct {
	Delay              int    `json:"delay,omitempty"`
	Typeahead          bool   `json:"typeahead,omitempty"`
	ForceKeyboardFocus bool   `json:"force_keyboard_focus,omitempty"`
	Icons              bool   `json:"icons,omitempty"`
	Spinner            bool   `json:"spinner,omitempty"`
	History            bool   `json:"history,omitempty"`
	MarginSpinner      int    `json:"margin_spinner,omitempty"`
	Placeholder        string `json:"placeholder,omitempty"`
}

type Icons struct {
	Hide      bool   `json:"hide,omitempty"`
	Size      int    `json:"size,omitempty"`
	ImageSize int    `json:"image_size,omitempty"`
	Theme     string `json:"theme,omitempty"`
}

type UI struct {
	Icons           Icons   `json:"icons,omitempty"`
	Orientation     string  `json:"orientation,omitempty"`
	Fullscreen      bool    `json:"fullscreen,omitempty"`
	IgnoreExclusive bool    `json:"ignore_exclusive,omitempty"`
	Horizontal      string  `json:"horizontal,omitempty"`
	Vertical        string  `json:"vertical,omitempty"`
	Width           int     `json:"width,omitempty"`
	Height          int     `json:"height,omitempty"`
	Margins         Margins `json:"margins,omitempty"`
	Anchors         Anchors `json:"anchors,omitempty"`
}

type Anchors struct {
	Top    bool `json:"top,omitempty"`
	Left   bool `json:"left,omitempty"`
	Right  bool `json:"right,omitempty"`
	Bottom bool `json:"bottom,omitempty"`
}

type Margins struct {
	Top    int `json:"top,omitempty"`
	Bottom int `json:"bottom,omitempty"`
	End    int `json:"end,omitempty"`
	Start  int `json:"start,omitempty"`
}

type List struct {
	AlwaysShow         bool   `json:"always_show,omitempty"`
	FixedHeight        bool   `json:"fixed_height,omitempty"`
	Height             int    `json:"height,omitempty"`
	HideSub            bool   `json:"hide_sub,omitempty"`
	MarginTop          int    `json:"margin_top,omitempty"`
	MaxEntries         int    `json:"max_entries,omitempty"`
	ScrollbarPolicy    string `json:"scrollbar_policy,omitempty"`
	ShowInitialEntries bool   `json:"show_initial_entries,omitempty"`
	Width              int    `json:"width,omitempty"`
}

func Get(config string) *Config {
	file := filepath.Join(util.ConfigDir(), config)

	cfg := &Config{}
	ok := util.FromJson(file, cfg)

	if !ok {
		err := json.Unmarshal(defaultConfig, &cfg)
		if err != nil {
			log.Panicln(err)
		}

		util.ToJson(&cfg, file)
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
