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
var config []byte

type Config struct {
	Placeholder        string            `json:"placeholder,omitempty"`
	EnableTypeahead    bool              `json:"enable_typeahead,omitempty"`
	ShowInitialEntries bool              `json:"show_initial_entries,omitempty"`
	ForceKeyboardFocus bool              `json:"force_keyboard_focus,omitempty"`
	SSHHostFile        string            `json:"ssh_host_file,omitempty"`
	ShellConfig        string            `json:"shell_config,omitempty"`
	Terminal           string            `json:"terminal,omitempty"`
	Orientation        string            `json:"orientation,omitempty"`
	Fullscreen         bool              `json:"fullscreen,omitempty"`
	Modules            []Module          `json:"modules,omitempty"`
	External           []Module          `json:"external,omitempty"`
	Icons              Icons             `json:"icons,omitempty"`
	Align              Align             `json:"align,omitempty"`
	List               List              `json:"list,omitempty"`
	Search             Search            `json:"search,omitempty"`
	Clipboard          Clipboard         `json:"clipboard,omitempty"`
	Runner             Runner            `json:"runner,omitempty"`
	ActivationMode     ActivationMode    `json:"activation_mode,omitempty"`
	ScrollbarPolicy    string            `json:"scrollbar_policy,omitempty"`
	IgnoreMouse        bool              `json:"ignore_mouse,omitempty"`
	Hyprland           Hyprland          `json:"hyprland,omitempty"`
	SpecialLabels      map[string]string `json:"special_labels,omitempty"`
	IsService          bool              `json:"-"`
}

type Hyprland struct {
	ContextAwareHistory bool `json:"context_aware_history,omitempty"`
}

type ActivationMode struct {
	UseAlt   bool `json:"use_alt,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
	UseFKeys bool `json:"use_f_keys,omitempty"`
}

type Clipboard struct {
	ImageHeight int `json:"image_height,omitempty"`
	MaxEntries  int `json:"max_entries,omitempty"`
}

type Runner struct {
	Excludes []string
	Includes []string
}

type Module struct {
	Prefix            string `json:"prefix,omitempty"`
	Name              string `json:"name,omitempty"`
	Src               string `json:"src,omitempty"`
	Cmd               string `json:"cmd,omitempty"`
	SpecialLabel      string `json:"special_label,omitempty"`
	Transform         bool   `json:"transform,omitempty"`
	History           bool   `json:"history,omitempty"`
	SwitcherExclusive bool   `json:"switcher_exclusive,omitempty"`
}

type Search struct {
	Delay         int  `json:"delay,omitempty"`
	HideIcons     bool `json:"hide_icons,omitempty"`
	MarginSpinner int  `json:"margin_spinner,omitempty"`
	HideSpinner   bool `json:"hide_spinner,omitempty"`
}

type Icons struct {
	Hide      bool   `json:"hide,omitempty"`
	Size      int    `json:"size,omitempty"`
	ImageSize int    `json:"image_size,omitempty"`
	Theme     string `json:"theme,omitempty"`
}

type Align struct {
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
	MarginTop   int  `json:"margin_top,omitempty"`
	Height      int  `json:"height,omitempty"`
	Width       int  `json:"width,omitempty"`
	AlwaysShow  bool `json:"always_show,omitempty"`
	FixedHeight bool `json:"fixed_height,omitempty"`
	HideSub     bool `json:"hide_sub,omitempty"`
}

func Get() *Config {
	file := filepath.Join(util.ConfigDir(), "config.json")

	cfg := &Config{}
	ok := util.FromJson(file, cfg)

	if !ok {
		err := json.Unmarshal(config, &cfg)
		if err != nil {
			log.Panicln(err)
		}

		util.ToJson(&cfg, file)
	}

	go setTerminal(cfg)

	if len(cfg.Modules) == 0 {
		log.Println("no modules configured")
		os.Exit(1)
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
