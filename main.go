package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/abenz1267/walker/processors"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Config struct {
	Placeholder  string                 `json:"placeholder"`
	NotifyOnFail bool                   `json:"notify_on_fail"`
	KeepOpen     bool                   `json:"keep_open"`
	ShellConfig  string                 `json:"shell_config"`
	Terminal     string                 `json:"terminal"`
	Orientation  string                 `json:"orientation"`
	Fullscreen   bool                   `json:"fullscreen"`
	Processors   []processors.Processor `json:"processors"`
	Icons        Icons                  `json:"icons"`
	Align        Align                  `json:"align"`
	List         List                   `json:"list"`
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

func main() {
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

	config := &Config{
		Terminal:     "foot",
		Fullscreen:   true,
		KeepOpen:     false,
		ShellConfig:  "",
		Placeholder:  "Search...",
		NotifyOnFail: true,
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

	entries := make(map[string]processors.Entry)

	ui := getUI(app, entries, config)

	setupInteractions(ui, entries, config)

	appwin, ok := ui.appwin.Cast().(*gtk.ApplicationWindow)
	if !ok {
		log.Fatalln("Unable to load application window")
	}

	appwin.SetApplication(app)

	gtk4layershell.InitForWindow(&appwin.Window)
	gtk4layershell.SetKeyboardMode(&appwin.Window, gtk4layershell.LayerShellKeyboardModeExclusive)

	if !config.Fullscreen {
		gtk4layershell.SetLayer(&appwin.Window, gtk4layershell.LayerShellLayerTop)
		gtk4layershell.SetAnchor(&appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
	} else {
		gtk4layershell.SetLayer(&appwin.Window, gtk4layershell.LayerShellLayerOverlay)
		gtk4layershell.SetAnchor(&appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
		gtk4layershell.SetAnchor(&appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
		gtk4layershell.SetAnchor(&appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
		gtk4layershell.SetAnchor(&appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
		gtk4layershell.SetExclusiveZone(&appwin.Window, -1)
	}

	appwin.Show()

	if config.KeepOpen {
		appwin.SetVisible(false)
	}
}
