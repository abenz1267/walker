package setup

import (
	"os"
	"slices"

	_ "embed"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/data"
	"github.com/abenz1267/walker/internal/setup/previews"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed layout_default.xml
var layoutDefault string

//go:embed item_default.xml
var itemDefault string

//go:embed item_clipboard.xml
var itemClipboard string

//go:embed item_calc.xml
var itemCalc string

//go:embed item_files.xml
var itemFiles string

//go:embed item_symbols.xml
var itemSymbols string

//go:embed preview_default.xml
var previewDefault string

//go:embed style_default.css
var styleDefault []byte

var (
	name               = "dev.benz.walker"
	supportsLayerShell = false
	itemBuilders       = make(map[string]string)
	layoutBuilders     = make(map[string]LayoutBuilder)
	currentWindow      *gtk.Window
	currentBuilder     LayoutBuilder
	selection          *gtk.SingleSelection
	homedir            string
	hasUI              = false
	app                *gtk.Application
	isService          = false
	isRunning          = false
)

type LayoutBuilder struct {
	window          *gtk.Window
	box             *gtk.Box
	boxwrapper      *gtk.Box
	preview         *gtk.Box
	searchcontainer *gtk.Box
	input           *gtk.Entry
	scroll          *gtk.ScrolledWindow
	list            *gtk.ListView
	grid            *gtk.GridView
	placeholder     *gtk.Label
}

func GTK() {
	homedir, _ = os.UserHomeDir()
	supportsLayerShell = gtk4layershell.IsSupported()

	app = gtk.NewApplication(name, gio.ApplicationHandlesCommandLine)

	isService = slices.Contains(os.Args, "--gapplication-service")

	app.ConnectHandleLocalOptions(func(dict *glib.VariantDict) int {
		return -1
	})

	app.ConnectCommandLine(func(cmd *gio.ApplicationCommandLine) int {
		app.Activate()

		cmd.Done()

		return 0
	})

	app.ConnectActivate(func() {
		setupBuilders()
		activate()
	})

	app.Hold()

	os.Exit(app.Run(os.Args))
}

func setupBuilders() {
	// default
	builder := gtk.NewBuilderFromString(layoutDefault)

	lb := LayoutBuilder{
		window:          builder.GetObject("Window").Cast().(*gtk.Window),
		box:             builder.GetObject("Box").Cast().(*gtk.Box),
		boxwrapper:      builder.GetObject("BoxWrapper").Cast().(*gtk.Box),
		preview:         builder.GetObject("Preview").Cast().(*gtk.Box),
		searchcontainer: builder.GetObject("SearchContainer").Cast().(*gtk.Box),
		input:           builder.GetObject("Input").Cast().(*gtk.Entry),
		scroll:          builder.GetObject("Scroll").Cast().(*gtk.ScrolledWindow),
		placeholder:     builder.GetObject("Placeholder").Cast().(*gtk.Label),
	}

	lb.window.SetName("window")
	lb.box.SetName("box")
	lb.boxwrapper.SetName("box-wrapper")
	lb.preview.SetName("preview")
	lb.searchcontainer.SetName("search-container")
	lb.input.SetName("input")
	lb.scroll.SetName("scroll")
	lb.placeholder.SetName("placeholder")

	var isList bool

	selection = data.GetSelection()
	selection.SetAutoselect(true)
	selection.ConnectItemsChanged(func(pos, removed, added uint) {
		if data.Items.Len() > 0 {
			currentBuilder.placeholder.SetVisible(false)
			currentBuilder.scroll.SetVisible(true)
		} else {
			currentBuilder.placeholder.SetVisible(true)
			currentBuilder.scroll.SetVisible(false)
		}

		selection.SetSelected(0)

		if currentBuilder.preview != nil {
			for item := range data.Items.All() {
				if v, ok := previews.Previewers[item.Item.Provider]; ok {
					v.Handle(item.Item, currentBuilder.preview, gtk.NewBuilderFromString(previewDefault))
				} else {
					currentBuilder.preview.SetVisible(false)
				}

				break
			}
		}
	})

	selection.ConnectSelectionChanged(func(pos, item uint) {
		if currentBuilder.preview != nil {
			i := data.Items.At(int(selection.Selected()))

			if v, ok := previews.Previewers[i.Item.Provider]; ok {
				v.Handle(i.Item, currentBuilder.preview, gtk.NewBuilderFromString(previewDefault))
			}
		}

		if currentBuilder.grid != nil {
			currentBuilder.grid.ScrollTo(selection.Selected(), gtk.ListScrollNone, nil)
		} else {
			currentBuilder.list.ScrollTo(selection.Selected(), gtk.ListScrollNone, nil)
		}
	})

	lb.list, isList = builder.GetObject("List").Cast().(*gtk.ListView)
	if !isList {
		lb.grid = builder.GetObject("List").Cast().(*gtk.GridView)
		lb.grid.SetName("list")
		lb.grid.SetSingleClickActivate(true)
		lb.grid.SetModel(selection)
		lb.grid.SetCanTarget(false)
		lb.grid.SetCanFocus(false)
	} else {
		lb.list.SetName("list")
		lb.list.SetModel(selection)
		lb.list.SetFactory(&getFactory().ListItemFactory)
		lb.list.SetSingleClickActivate(true)
		lb.list.SetCanTarget(false)
		lb.list.SetCanFocus(false)
	}

	lb.input.ConnectChanged(func() {
		data.InputChanged(lb.input)
	})

	layoutBuilders["default"] = lb
	itemBuilders["default"] = itemDefault
	itemBuilders["clipboard"] = itemClipboard
	itemBuilders["calc"] = itemCalc
	itemBuilders["files"] = itemFiles
	itemBuilders["symbols"] = itemSymbols
}

func activate() {
	if hasUI {
		if config.LoadedConfig.CloseWhenOpen && isRunning {
			quit()
			return
		}

		currentWindow.SetVisible(true)
		isRunning = true
		return
	}

	config.Load()
	previews.Load()
	data.Init()

	initCSS()
	initShell(layoutBuilders["default"].window)

	app.AddWindow(layoutBuilders["default"].window)

	currentWindow = layoutBuilders["default"].window
	currentBuilder = layoutBuilders["default"]
	layoutBuilders["default"].input.Emit("changed")

	go data.StartListening()

	setupBinds()
	setupKeyEvents(app, currentBuilder.window)

	currentWindow.SetVisible(true)
	hasUI = true
	isRunning = true
}

func initShell(window *gtk.Window) {
	if !supportsLayerShell {
		return
	}

	gtk4layershell.InitForWindow(window)
	gtk4layershell.SetNamespace(window, "walker")

	gtk4layershell.SetAnchor(window, gtk4layershell.LayerShellEdgeTop, true)
	gtk4layershell.SetAnchor(window, gtk4layershell.LayerShellEdgeBottom, true)
	gtk4layershell.SetAnchor(window, gtk4layershell.LayerShellEdgeLeft, true)
	gtk4layershell.SetAnchor(window, gtk4layershell.LayerShellEdgeRight, true)

	gtk4layershell.SetLayer(window, gtk4layershell.LayerShellLayerOverlay)
	gtk4layershell.SetExclusiveZone(window, -1)

	gtk4layershell.SetKeyboardMode(window, gtk4layershell.LayerShellKeyboardModeOnDemand)
}

func initCSS() {
	provider := gtk.NewCSSProvider()
	provider.LoadFromBytes(glib.NewBytes(styleDefault))

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), provider, gtk.STYLE_PROVIDER_PRIORITY_USER)
}
