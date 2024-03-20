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
	Placeholder           string    `json:"placeholder,omitempty"`
	NotifyOnFail          bool      `json:"notify_on_fail,omitempty"`
	ShowInitialEntries    bool      `json:"show_initial_entries,omitempty"`
	ShellConfig           string    `json:"shell_config,omitempty"`
	Terminal              string    `json:"terminal,omitempty"`
	Orientation           string    `json:"orientation,omitempty"`
	Fullscreen            bool      `json:"fullscreen,omitempty"`
	Modules               []Module  `json:"modules,omitempty"`
	External              []Module  `json:"external,omitempty"`
	Icons                 Icons     `json:"icons,omitempty"`
	Align                 Align     `json:"align,omitempty"`
	List                  List      `json:"list,omitempty"`
	Search                Search    `json:"search,omitempty"`
	DisableActivationMode bool      `json:"disable_activation_mode,omitempty"`
	Clipboard             Clipboard `json:"clipboard,omitempty"`
}

type Clipboard struct {
	ImageHeight int `json:"image_height,omitempty"`
	MaxEntries  int `json:"max_entries,omitempty"`
}

type Module struct {
	Prefix            string `json:"prefix,omitempty"`
	Name              string `json:"name,omitempty"`
	Src               string `json:"src,omitempty"`
	Cmd               string `json:"cmd,omitempty"`
	Transform         bool   `json:"transform,omitempty"`
	History           bool   `json:"history,omitempty"`
	SwitcherExclusive bool   `json:"switcher_exclusive,omitempty"`
}

type Search struct {
	Delay     int  `json:"delay,omitempty"`
	HideIcons bool `json:"hide_icons,omitempty"`
}

type Icons struct {
	Hide        bool `json:"hide,omitempty"`
	Size        int  `json:"size,omitempty"`
	ImageHeight int  `json:"image_height,omitempty"`
}

type Align struct {
	Horizontal string  `json:"horizontal,omitempty"`
	Vertical   string  `json:"vertical,omitempty"`
	Width      int     `json:"width,omitempty"`
	Margins    Margins `json:"margins,omitempty"`
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
	AlwaysShow  bool `json:"always_show,omitempty"`
	FixedHeight bool `json:"fixed_height,omitempty"`
}

func Get() *Config {
	file := filepath.Join(util.ConfigDir(), "config.json")

	cfg := &Config{}
	ok := util.FromJson(file, cfg)

	if !ok {
		err := json.Unmarshal(config, &cfg)
		if err != nil {
			log.Fatalln(err)
		}

		util.ToJson(&cfg, file)
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
