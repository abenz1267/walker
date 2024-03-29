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
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed layout.xml
var layout string

//go:embed themes/style.default.css
var style []byte

var labels = []string{"j", "k", "l", ";", "a", "s", "d", "f"}

type ProcMap map[string][]modules.Workable

var (
	cfg        *config.Config
	ui         *UI
	procs      ProcMap
	singleProc modules.Workable
	hstry      history.History
	appstate   *state.AppState
)

type UI struct {
	app           *gtk.Application
	builder       *gtk.Builder
	scroll        *gtk.ScrolledWindow
	spinner       *gtk.Spinner
	searchwrapper *gtk.Box
	box           *gtk.Box
	appwin        *gtk.ApplicationWindow
	typeahead     *gtk.SearchEntry
	search        *gtk.SearchEntry
	list          *gtk.ListView
	items         *gioutil.ListModel[modules.Entry]
	selection     *gtk.SingleSelection
	prefixClasses map[string][]string
}

func Activate(state *state.AppState) func(app *gtk.Application) {
	appstate = state

	return func(app *gtk.Application) {
		appstate.Started = time.Now()

		if appstate.IsRunning {
			return
		}

		appstate.IsRunning = true

		if appstate.HasUI {
			if ui.items.NItems() > 0 {
				ui.selection.SetSelected(0)
			}

			ui.appwin.SetVisible(true)

			if !appstate.IsMeasured {
				fmt.Printf("startup time: %s\n", time.Since(appstate.Started))
				appstate.IsMeasured = true
			}

			ui.search.GrabFocus()
			process()

			return
		}

		cfg = config.Get()
		hstry = history.Get()

		setupUI(app)
		setupInteractions(appstate)

		ui.appwin.SetApplication(app)

		gtk4layershell.InitForWindow(&ui.appwin.Window)
		gtk4layershell.SetNamespace(&ui.appwin.Window, "walker")

		if cfg.ForceKeyboardFocus {
			gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeExclusive)
		} else {
			gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeOnDemand)
		}

		if !cfg.Fullscreen {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerTop)

			if cfg.Align.Anchors.Top {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
			}

			if cfg.Align.Anchors.Bottom {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
			}

			if cfg.Align.Anchors.Left {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
			}

			if cfg.Align.Anchors.Right {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
			}

			if cfg.Align.IgnoreExclusive {
				gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
			}
		} else {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerOverlay)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)

			if cfg.Align.IgnoreExclusive {
				gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
			}
		}

		ui.appwin.Show()
		appstate.HasUI = true
	}
}

func setupUI(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	builder := gtk.NewBuilderFromString(layout, len(layout))

	items := gioutil.NewListModel[modules.Entry]()

	ui = &UI{
		app:           app,
		builder:       builder,
		spinner:       builder.GetObject("spinner").Cast().(*gtk.Spinner),
		searchwrapper: builder.GetObject("searchwrapper").Cast().(*gtk.Box),
		typeahead:     builder.GetObject("typeahead").Cast().(*gtk.SearchEntry),
		scroll:        builder.GetObject("scroll").Cast().(*gtk.ScrolledWindow),
		box:           builder.GetObject("box").Cast().(*gtk.Box),
		appwin:        builder.GetObject("win").Cast().(*gtk.ApplicationWindow),
		search:        builder.GetObject("search").Cast().(*gtk.SearchEntry),
		list:          builder.GetObject("list").Cast().(*gtk.ListView),
		items:         items,
		selection:     gtk.NewSingleSelection(items.ListModel),
		prefixClasses: make(map[string][]string),
	}

	ui.list.SetSingleClickActivate(true)
	ui.list.ConnectActivate(func(pos uint) {
		activateItem(false, false)
	})

	if cfg.Search.MarginSpinner != 0 {
		ui.searchwrapper.SetSpacing(cfg.Search.MarginSpinner)
	}

	ui.spinner.SetSpinning(true)
	ui.typeahead.SetHExpand(true)
	ui.typeahead.SetFocusable(false)
	ui.typeahead.SetFocusOnClick(false)
	ui.typeahead.SetCanFocus(false)

	fc := gtk.NewEventControllerFocus()
	fc.Connect("enter", func() {
		if !appstate.IsMeasured {
			fmt.Printf("startup time: %s\n", time.Since(appstate.Started))
			appstate.IsMeasured = true
		}
	})

	ui.search.AddController(fc)
	ui.selection.SetAutoselect(true)

	factory := setupFactory()

	ui.list.SetModel(ui.selection)
	ui.list.SetFactory(&factory.ListItemFactory)

	setupUserStyle()
	handleListVisibility()

	ui.selection.ConnectItemsChanged(func(p, r, a uint) {
		handleListVisibility()
	})
}

func setupUserStyle() {
	cssFile := filepath.Join(util.ConfigDir(), "style.css")

	cssProvider := gtk.NewCSSProvider()
	if _, err := os.Stat(cssFile); err == nil {
		cssProvider.LoadFromPath(cssFile)
	} else {
		cssProvider.LoadFromData(string(style))

		err := os.WriteFile(cssFile, style, 0o600)
		if err != nil {
			log.Panicln(err)
		}
	}

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)
	ui.search.SetObjectProperty("search-delay", cfg.Search.Delay)

	if cfg.IgnoreMouse {
		ui.list.SetCanTarget(false)
	}

	if cfg.List.MarginTop != 0 {
		ui.list.SetMarginTop(cfg.List.MarginTop)
	}

	if cfg.Search.HideSpinner {
		ui.spinner.SetVisible(false)
	}

	if cfg.Search.HideIcons {
		ui.search.FirstChild().(*gtk.Image).Hide()
		ui.search.LastChild().(*gtk.Image).Hide()
		ui.typeahead.FirstChild().(*gtk.Image).Hide()
		ui.typeahead.LastChild().(*gtk.Image).Hide()
	}

	alignments := make(map[string]gtk.Align)
	alignments["fill"] = gtk.AlignFill
	alignments["start"] = gtk.AlignStart
	alignments["end"] = gtk.AlignEnd
	alignments["center"] = gtk.AlignCenter

	policies := make(map[string]gtk.PolicyType)
	policies["never"] = gtk.PolicyNever
	policies["always"] = gtk.PolicyAlways
	policies["automatic"] = gtk.PolicyAutomatic
	policies["external"] = gtk.PolicyExternal

	ui.scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	if cfg.ScrollbarPolicy != "" {
		ui.scroll.SetPolicy(gtk.PolicyNever, policies[cfg.ScrollbarPolicy])
	}

	width := -1
	if cfg.Align.Width != 0 {
		width = cfg.Align.Width
	}

	height := -1
	if cfg.Align.Height != 0 {
		height = cfg.Align.Height
	}

	ui.box.SetSizeRequest(width, height)

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
		ui.search.SetVAlign(gtk.AlignCenter)
		ui.typeahead.SetVAlign(gtk.AlignCenter)
		ui.search.SetHExpand(false)
		ui.typeahead.SetHExpand(false)
		ui.list.SetOrientation(gtk.OrientationHorizontal)
		ui.scroll.SetPolicy(policies[cfg.ScrollbarPolicy], gtk.PolicyNever)
	}

	ui.scroll.SetMaxContentWidth(cfg.List.Width)
	ui.scroll.SetMaxContentHeight(cfg.List.Height)

	if cfg.Placeholder != "" {
		ui.search.SetObjectProperty("placeholder-text", cfg.Placeholder)
	}

	ui.box.SetMarginBottom(cfg.Align.Margins.Bottom)
	ui.box.SetMarginTop(cfg.Align.Margins.Top)
	ui.box.SetMarginStart(cfg.Align.Margins.Start)
	ui.box.SetMarginEnd(cfg.Align.Margins.End)
}

func setupFactory() *gtk.SignalListItemFactory {
	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(func(item *gtk.ListItem) {
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		item.SetChild(box)
	})

	factory.ConnectBind(func(item *gtk.ListItem) {
		val := ui.items.Item(int(item.Position()))
		child := item.Child()

		if child == nil {
			return
		}

		box, ok := child.(*gtk.Box)
		if !ok {
			log.Panicln("child is not a box")
		}

		if box.FirstChild() != nil {
			return
		}

		// if val.DragDrop {
		// 	dd := gtk.NewDragSource()
		// 	dd.ConnectPrepare(func(_, _ float64) *gdk.ContentProvider {
		// 		file := gio.NewFileForPath(val.DragDropData)
		//
		// 		b := glib.NewBytes([]byte(fmt.Sprintf("%s\n", file.URI())))
		//
		// 		cp := gdk.NewContentProviderForBytes("text/uri-list", b)
		//
		// 		return cp
		// 	})
		//
		// 	dd.ConnectDragBegin(func(_ gdk.Dragger) {
		// 		closeAfterActivation(false, false)
		// 	})
		//
		// 	box.AddController(dd)
		// }

		box.SetCSSClasses([]string{"item", val.Class})

		// if !cfg.IgnoreMouse {
		// click := gtk.NewGestureClick()
		//
		// if val.DragDrop {
		// 	click.ConnectReleased(func(m int, _, _ float64) {
		// 		activateItem(false, false)
		// 	})
		// }
		//
		// box.AddController(click)
		// }

		wrapper := gtk.NewBox(gtk.OrientationVertical, 0)
		wrapper.SetCSSClasses([]string{"textwrapper"})
		wrapper.SetHExpand(true)

		if val.Image != "" {
			f := gio.NewFileForPath(val.Image)

			texture, err := gdk.NewTextureFromFile(f)
			if err != nil {
				log.Println(err)
			}

			image := gtk.NewImageFromPaintable(texture)
			image.SetHExpand(true)
			image.SetSizeRequest(-1, cfg.Clipboard.ImageHeight)
			box.Append(image)
		}

		if !cfg.Icons.Hide && val.Icon != "" {
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

		if cfg.Orientation != "horizontal" {
			top.SetWrap(true)
		}

		top.SetHAlign(gtk.AlignStart)
		top.SetCSSClasses([]string{"label"})

		wrapper.Append(top)

		if val.Sub != "" && !cfg.List.HideSub {
			bottom := gtk.NewLabel(val.Sub)
			bottom.SetMaxWidthChars(0)

			if cfg.Orientation != "horizontal" {
				bottom.SetWrap(true)
			}

			bottom.SetHAlign(gtk.AlignStart)
			bottom.SetCSSClasses([]string{"sub"})

			wrapper.Append(bottom)
		} else {
			wrapper.SetVAlign(gtk.AlignCenter)
		}

		if !cfg.ActivationMode.Disabled {
			if item.Position()+1 <= uint(len(labels)) {
				l := gtk.NewLabel(labels[item.Position()])
				l.SetCSSClasses([]string{"activationlabel"})
				box.Append(l)
			}
		}
	})

	return factory
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
