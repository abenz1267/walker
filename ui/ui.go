package ui

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/history"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/state"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed layout.xml
var layout string

//go:embed defaultstyle.css
var style string

var labels = []string{"j", "k", "l", ";", "a", "s", "d", "f"}

var (
	cfg      *config.Config
	ui       *UI
	entries  map[string]modules.Entry
	procs    map[string][]modules.Workable
	hstry    history.History
	appstate *state.AppState
)

type UI struct {
	app           *gtk.Application
	builder       *gtk.Builder
	scroll        *gtk.ScrolledWindow
	box           *gtk.Box
	appwin        *gtk.ApplicationWindow
	search        *gtk.SearchEntry
	list          *gtk.ListView
	items         *gtk.StringList
	selection     *gtk.SingleSelection
	factory       *gtk.SignalListItemFactory
	prefixClasses map[string][]string
}

func Activate(state *state.AppState) func(app *gtk.Application) {
	appstate = state

	return func(app *gtk.Application) {
		if appstate.IsRunning {
			return
		}

		if appstate.IsService {
			appstate.Started = time.Now()
			appstate.IsRunning = true
		}

		cfg = config.Get()
		hstry = history.Get()

		entries = make(map[string]modules.Entry)

		setupUI(app)

		setupInteractions()

		ui.appwin.SetApplication(app)

		gtk4layershell.InitForWindow(&ui.appwin.Window)
		gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeExclusive)

		if !cfg.Fullscreen {
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
}

func setupUI(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Fatalln("gtk-layer-shell not supported")
	}

	builder := gtk.NewBuilderFromString(layout, len(layout))

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	cfgDir = filepath.Join(cfgDir, "walker")

	cssFile := filepath.Join(cfgDir, "style.css")

	cssProvider := gtk.NewCSSProvider()
	if _, err := os.Stat(cssFile); err == nil {
		cssProvider.LoadFromPath(cssFile)
	} else {
		cssProvider.LoadFromData(style)
	}

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

	items := gtk.NewStringList([]string{""})

	ui = &UI{
		app:           app,
		builder:       builder,
		scroll:        builder.GetObject("scroll").Cast().(*gtk.ScrolledWindow),
		box:           builder.GetObject("box").Cast().(*gtk.Box),
		appwin:        builder.GetObject("win").Cast().(*gtk.ApplicationWindow),
		search:        builder.GetObject("search").Cast().(*gtk.SearchEntry),
		list:          builder.GetObject("list").Cast().(*gtk.ListView),
		items:         items,
		selection:     gtk.NewSingleSelection(items),
		factory:       gtk.NewSignalListItemFactory(),
		prefixClasses: make(map[string][]string),
	}

	fc := gtk.NewEventControllerFocus()
	fc.Connect("enter", func() {
		if !appstate.IsMeasured {
			fmt.Printf("startup time: %s\n", time.Since(appstate.Started))
			appstate.IsMeasured = true
		}
	})

	ui.search.AddController(fc)
	ui.search.SetObjectProperty("search-delay", cfg.Search.Delay)

	if cfg.Search.HideIcons {
		ui.search.FirstChild().(*gtk.Image).Hide()
		ui.search.LastChild().(*gtk.Image).Hide()
	}

	alignments := make(map[string]gtk.Align)
	alignments["fill"] = gtk.AlignFill
	alignments["start"] = gtk.AlignStart
	alignments["end"] = gtk.AlignEnd
	alignments["center"] = gtk.AlignCenter

	if cfg.Align.Width != 0 {
		ui.box.SetSizeRequest(cfg.Align.Width, -1)
	}

	if cfg.List.Height != 0 {
		ui.scroll.SetMaxContentHeight(cfg.List.Height)

		if cfg.List.FixedHeight {
			ui.list.SetSizeRequest(cfg.Align.Width, cfg.List.Height)
			ui.scroll.SetSizeRequest(cfg.Align.Width, cfg.List.Height)
		}
	}

	if cfg.Align.Horizontal != "" {
		ui.box.SetObjectProperty("halign", alignments[cfg.Align.Horizontal])
	}

	if cfg.Align.Vertical != "" {
		ui.box.SetObjectProperty("valign", alignments[cfg.Align.Vertical])
	}

	if cfg.Orientation == "horizontal" {
		ui.box.SetObjectProperty("orientation", gtk.OrientationHorizontal)
		ui.search.SetVAlign(gtk.AlignStart)
	}

	if cfg.Placeholder != "" {
		ui.search.SetObjectProperty("placeholder-text", cfg.Placeholder)
	}

	ui.box.SetMarginBottom(cfg.Align.Margins.Bottom)
	ui.box.SetMarginTop(cfg.Align.Margins.Top)
	ui.box.SetMarginStart(cfg.Align.Margins.Start)
	ui.box.SetMarginEnd(cfg.Align.Margins.End)

	ui.selection.SetAutoselect(true)

	ui.factory.ConnectSetup(func(item *gtk.ListItem) {
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		box.SetFocusable(true)
		item.SetChild(box)
	})

	ui.factory.ConnectBind(func(item *gtk.ListItem) {
		key := item.Item().Cast().(*gtk.StringObject).String()

		if item.Selected() {
			child := item.Child()
			if child != nil {
				box, ok := child.(*gtk.Box)
				if !ok {
					log.Fatalln("child is not a box")
				}

				if !activationEnabled {
					box.GrabFocus()
					ui.appwin.SetCSSClasses([]string{entries[key].Class})
					ui.search.GrabFocus()
				}
			}
		}

		if val, ok := entries[key]; ok {
			child := item.Child()

			if child != nil {
				box, ok := child.(*gtk.Box)
				if !ok {
					log.Fatalln("child is not a box")
				}

				if box.FirstChild() != nil {
					return
				}

				box.SetCSSClasses([]string{"item", val.Class})

				motion := gtk.NewEventControllerMotion()
				motion.ConnectEnter(func(_, _ float64) {
					ui.selection.SetSelected(item.Position())
				})

				click := gtk.NewGestureClick()
				click.ConnectPressed(func(m int, _, _ float64) {
					activateItem(false)
				})

				box.AddController(click)
				box.AddController(motion)

				wrapper := gtk.NewBox(gtk.OrientationVertical, 0)
				wrapper.SetCSSClasses([]string{"textwrapper"})
				wrapper.SetHExpand(true)

				if cfg.Icons.Hide || val.Icon != "" {
					if val.IconIsImage {
						image := gtk.NewPictureForFilename(val.Icon)
						image.SetMarginEnd(10)
						image.SetSizeRequest(0, 200)
						image.SetCanShrink(true)
						if val.HideText {
							image.SetHExpand(true)
						}
						box.Append(image)
					} else {
						icon := gtk.NewImageFromIconName(val.Icon)
						icon.SetIconSize(gtk.IconSizeLarge)
						icon.SetPixelSize(cfg.Icons.Size)
						icon.SetCSSClasses([]string{"icon"})
						box.Append(icon)
					}
				}

				if !val.HideText {
					box.Append(wrapper)
				}

				top := gtk.NewLabel(val.Label)
				top.SetMaxWidthChars(0)
				top.SetWrap(true)
				top.SetHAlign(gtk.AlignStart)
				top.SetCSSClasses([]string{"label"})

				wrapper.Append(top)

				if val.Sub != "" {
					bottom := gtk.NewLabel(val.Sub)
					bottom.SetMaxWidthChars(0)
					bottom.SetWrap(true)
					bottom.SetHAlign(gtk.AlignStart)
					bottom.SetCSSClasses([]string{"sub"})

					wrapper.Append(bottom)
				} else {
					wrapper.SetVAlign(gtk.AlignCenter)
				}

				if !cfg.DisableActivationMode {
					if item.Position()+1 <= uint(len(labels)) {
						l := gtk.NewLabel(labels[item.Position()])
						l.SetCSSClasses([]string{"activationlabel"})
						box.Append(l)
					}
				}
			}
		}
	})

	ui.list.SetModel(ui.selection)
	ui.list.SetFactory(&ui.factory.ListItemFactory)

	ui.selection.ConnectItemsChanged(func(p, r, a uint) {
		handleListVisibility()
	})
}

func handleListVisibility() {
	ui.list.SetVisible(false)
	ui.scroll.SetVisible(false)

	if cfg.List.AlwaysShow {
		ui.list.SetVisible(true)
		ui.scroll.SetVisible(true)
		return
	}

	if ui.items.NItems() != 0 {
		ui.list.SetVisible(true)
		ui.scroll.SetVisible(true)
		return
	}

	if ui.items.NItems() == 0 {
		if cfg.List.AlwaysShow {
			ui.list.SetVisible(true)
			ui.scroll.SetVisible(true)
		} else {
			ui.list.SetVisible(false)
			ui.scroll.SetVisible(false)
		}
	}
}
