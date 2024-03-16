package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Placeholder           string   `json:"placeholder,omitempty"`
	NotifyOnFail          bool     `json:"notify_on_fail,omitempty"`
	ShowInitialEntries    bool     `json:"show_initial_entries,omitempty"`
	ShellConfig           string   `json:"shell_config,omitempty"`
	Terminal              string   `json:"terminal,omitempty"`
	Orientation           string   `json:"orientation,omitempty"`
	Fullscreen            bool     `json:"fullscreen,omitempty"`
	Modules               []Module `json:"modules,omitempty"`
	External              []Module `json:"external,omitempty"`
	Icons                 Icons    `json:"icons,omitempty"`
	Align                 Align    `json:"align,omitempty"`
	List                  List     `json:"list,omitempty"`
	Search                Search   `json:"search,omitempty"`
	DisableActivationMode bool     `json:"disable_activation_mode,omitempty"`
}

type Module struct {
	Prefix  string `json:"prefix,omitempty"`
	Name    string `json:"name,omitempty"`
	Src     string `json:"src,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
	History bool   `json:"history,omitempty"`
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
	Height     int    `json:"height,omitempty"`
	Style      string `json:"style,omitempty"`
	AlwaysShow bool   `json:"always_show,omitempty"`
}

func Get() *Config {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	cfgDir = filepath.Join(cfgDir, "walker")
	cfgName := filepath.Join(cfgDir, "config.json")

	cfg := &Config{
		Terminal:              "",
		Fullscreen:            true,
		ShowInitialEntries:    false,
		ShellConfig:           "",
		Placeholder:           "Search...",
		DisableActivationMode: false,
		NotifyOnFail:          true,
		Icons: Icons{
			Hide:        false,
			Size:        32,
			ImageHeight: 100,
		},
		Search: Search{
			Delay:     0,
			HideIcons: false,
		},
		Align: Align{
			Horizontal: "center",
			Vertical:   "start",
			Width:      400,
			Margins: Margins{
				Top:    50,
				Bottom: 0,
				End:    0,
				Start:  0,
			},
		},
		Modules: []Module{
			{Name: "runner", Prefix: ""},
			{Name: "websearch", Prefix: "?"},
			{Name: "applications", Prefix: ""},
		},
		List: List{
			Height:     300,
			Style:      "dynamic",
			AlwaysShow: false,
		},
	}

	if _, err := os.Stat(cfgName); err == nil {
		file, err := os.Open(cfgName)
		if err != nil {
			log.Fatalln(err)
		}

		b, err := io.ReadAll(file)
		if err != nil {
			log.Fatalln(err)
		}

		err = json.Unmarshal(b, &cfg)
		if err != nil {
			log.Fatalln(err)
		}
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
