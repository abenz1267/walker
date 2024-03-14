package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/processors"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Config struct {
	Placeholder        string                 `json:"placeholder"`
	NotifyOnFail       bool                   `json:"notify_on_fail"`
	KeepOpen           bool                   `json:"keep_open"`
	ShowInitialEntries bool                   `json:"show_initial_entries"`
	ShellConfig        string                 `json:"shell_config"`
	Terminal           string                 `json:"terminal"`
	Orientation        string                 `json:"orientation"`
	Fullscreen         bool                   `json:"fullscreen"`
	Processors         []processors.Processor `json:"processors"`
	Icons              Icons                  `json:"icons"`
	Align              Align                  `json:"align"`
	List               List                   `json:"list"`
}

type Icons struct {
	Hide bool `json:"hide"`
	Size int  `json:"size"`
}

type Align struct {
	Horizontal string  `json:"horizontal"`
	Vertical   string  `json:"vertical"`
	Width      int     `json:"width"`
	Margins    Margins `json:"margins"`
}

type Margins struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	End    int `json:"end"`
	Start  int `json:"start"`
}

type List struct {
	Height     int    `json:"height"`
	Style      string `json:"style"`
	AlwaysShow bool   `json:"always_show"`
}

var (
	now      time.Time
	measured bool
	config   *Config
	ui       *UI
	entries  map[string]processors.Entry
	procs    map[string][]Processor
)

func main() {
	args := os.Args[1:]

	if len(os.Args) > 0 {
		switch args[0] {
		case "--version":
			fmt.Println("0.0.9-git")
			return
		case "--help", "-h":
			fmt.Println("see README.md at https://github.com/abenz1267/walker")
			return
		default:
			fmt.Printf("Unsupported option '%s'\n", args[0])
			return
	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			switch args[0] {
			case "--version":
				fmt.Println("0.0.9-git")
				return
			case "--help", "-h":
				fmt.Println("see README.md at https://github.com/abenz1267/walker")
				return
			default:
				fmt.Printf("Unsupported option '%s'\n", args[0])
				return
			}
		}
	}

	now = time.Now()

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

	app := gtk.NewApplication("dev.benz.walker", 0)
	app.Connect("activate", activate)

	app.Flags()

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	cfgDir = filepath.Join(cfgDir, "walker")
	cfgName := filepath.Join(cfgDir, "config.json")

	config = &Config{
		Terminal:           "",
		Fullscreen:         true,
		KeepOpen:           false,
		ShowInitialEntries: false,
		ShellConfig:        "",
		Placeholder:        "Search...",
		NotifyOnFail:       true,
		Icons: Icons{
			Hide: false,
			Size: 32,
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

	if config.KeepOpen {
		ui.appwin.SetVisible(false)
	}
}
