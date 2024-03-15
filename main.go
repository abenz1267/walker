package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/processors"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const VERSION = "0.0.16"

type Config struct {
	Placeholder           string                 `json:"placeholder,omitempty"`
	NotifyOnFail          bool                   `json:"notify_on_fail,omitempty"`
	ShowInitialEntries    bool                   `json:"show_initial_entries,omitempty"`
	ShellConfig           string                 `json:"shell_config,omitempty"`
	Terminal              string                 `json:"terminal,omitempty"`
	Orientation           string                 `json:"orientation,omitempty"`
	Fullscreen            bool                   `json:"fullscreen,omitempty"`
	Processors            []processors.Processor `json:"processors,omitempty"`
	Icons                 Icons                  `json:"icons,omitempty"`
	Align                 Align                  `json:"align,omitempty"`
	List                  List                   `json:"list,omitempty"`
	Search                Search                 `json:"search,omitempty"`
	DisableActivationMode bool                   `json:"disable_activation_mode,omitempty"`
}

type Search struct {
	Delay     int  `json:"delay,omitempty"`
	HideIcons bool `json:"hide_icons,omitempty"`
}

type Icons struct {
	Hide bool `json:"hide,omitempty"`
	Size int  `json:"size,omitempty"`
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

type HistoryEntry struct {
	LastUsed      time.Time `json:"last_used,omitempty"`
	Used          int       `json:"used,omitempty"`
	daysSinceUsed int
}

var (
	now       time.Time
	measured  bool
	config    *Config
	ui        *UI
	entries   map[string]processors.Entry
	procs     map[string][]Processor
	history   map[string]HistoryEntry
	isService bool
	isRunning bool
)

func main() {
	if isRunning {
		return
	}

	withArgs := false

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			switch args[0] {
			case "--version":
				fmt.Println(VERSION)
				return
			case "--gapplication-service":
				isService = true
			case "--help", "-h", "--help-all":
				withArgs = true
			default:
				fmt.Printf("Unsupported option '%s'\n", args[0])
				return
			}
		}
	}

	now = time.Now()

	history = make(map[string]HistoryEntry)

	loadHistory()

	if !isService && !withArgs {
		tmp := os.TempDir()
		if _, err := os.Stat(filepath.Join(tmp, "walker.lock")); err == nil {
			log.Println("lockfile exists. exiting.")
			return
		}

		err := os.WriteFile(filepath.Join(tmp, "walker.lock"), []byte{}, 0o600)
		if err != nil {
			log.Fatalln(err)
		}
		defer os.Remove(filepath.Join(tmp, "walker.lock"))
	}

	app := gtk.NewApplication("dev.benz.walker", 0)
	app.Connect("activate", activate)

	app.Flags()

	if isService {
		app.Hold()
	}

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) {
	if isRunning {
		return
	}

	if isService {
		now = time.Now()
		isRunning = true
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	cfgDir = filepath.Join(cfgDir, "walker")
	cfgName := filepath.Join(cfgDir, "config.json")

	config = &Config{
		Terminal:              "",
		Fullscreen:            true,
		ShowInitialEntries:    false,
		ShellConfig:           "",
		Placeholder:           "Search...",
		DisableActivationMode: false,
		NotifyOnFail:          true,
		Icons: Icons{
			Hide: false,
			Size: 32,
		},
		Search: Search{
			Delay:     150,
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
		Processors: []processors.Processor{
			{Name: "runner", Prefix: "!"},
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

		err = json.Unmarshal(b, &config)
		if err != nil {
			log.Fatalln(err)
		}
	}

	go setTerminal()

	entries = make(map[string]processors.Entry)

	createUI(app)

	setupInteractions()

	ui.appwin.SetApplication(app)

	gtk4layershell.InitForWindow(&ui.appwin.Window)
	gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeExclusive)

	if !config.Fullscreen {
		gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerTop)
		gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
	} else {
		gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerOverlay)
		gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
		gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
		gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
		gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
		gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
	}

	ui.appwin.Show()
}

func setTerminal() {
	if config.Terminal != "" {
		path, _ := exec.LookPath(config.Terminal)

		if path != "" {
			config.Terminal = path
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
			config.Terminal = path
			break
		}
	}
}

func loadHistory() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		log.Fatalf("failed to get cache dir: %s", err)
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	path := filepath.Join(cacheDir, "history.json")

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Fatalln(err)
		}

		err = json.Unmarshal(b, &history)
		if err != nil {
			log.Fatalln(err)
		}

		for k, v := range history {
			today := time.Now()
			v.daysSinceUsed = int(today.Sub(v.LastUsed).Hours() / 24)

			history[k] = v
		}
	}
}
