package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/abenz1267/walker/internal/data"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
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
)

type LayoutBuilder struct {
	window          *gtk.Window
	box             *gtk.Box
	searchcontainer *gtk.Box
	input           *gtk.Entry
	scroll          *gtk.ScrolledWindow
	list            *gtk.ListView
	grid            *gtk.GridView
}

type ItemBuilder struct {
	box     *gtk.Box
	text    *gtk.Label
	subtext *gtk.Label
	image   *gtk.Image
}

func GTK() {
	homedir, _ = os.UserHomeDir()
	supportsLayerShell = gtk4layershell.IsSupported()

	app := gtk.NewApplication(name, gio.ApplicationHandlesCommandLine)

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
		setupInteractions(app)
		activate(app)
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
		searchcontainer: builder.GetObject("SearchContainer").Cast().(*gtk.Box),
		input:           builder.GetObject("Input").Cast().(*gtk.Entry),
		scroll:          builder.GetObject("Scroll").Cast().(*gtk.ScrolledWindow),
	}

	lb.window.SetName("window")
	lb.box.SetName("box")
	lb.searchcontainer.SetName("search-container")
	lb.input.SetName("input")
	lb.scroll.SetName("scroll")

	var isList bool

	selection = data.GetSelection()
	selection.SetAutoselect(true)
	selection.ConnectItemsChanged(func(pos, removed, added uint) {
		selection.SetSelected(0)
	})

	selection.ConnectSelectionChanged(func(pos, item uint) {
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
		lb.list.ConnectActivate(func(pos uint) {
			data.Activate(pos, currentBuilder.input.Text())
		})
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

func activate(app *gtk.Application) {
	initCSS()
	initShell(layoutBuilders["default"].window)

	app.AddWindow(layoutBuilders["default"].window)

	currentWindow = layoutBuilders["default"].window
	currentBuilder = layoutBuilders["default"]
	layoutBuilders["default"].input.Emit("changed")

	go data.StartListening()

	currentWindow.SetVisible(true)
}

func getFactory() *gtk.SignalListItemFactory {
	f := gtk.NewSignalListItemFactory()

	f.ConnectSetup(func(object *coreglib.Object) {
	})

	f.ConnectUnbind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		box := item.Child().(*gtk.Box)

		for box.FirstChild() != nil {
			box.Remove(box.FirstChild())
		}
	})

	f.ConnectBind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		resp := data.Items.At(int(item.Position()))
		val := resp.Item

		switch val.Provider {
		case "files":
			builder := gtk.NewBuilderFromString(itemBuilders["files"])

			box := builder.GetObject("ItemBox").Cast().(*gtk.Box)
			box.AddCSSClass(val.Provider)
			box.SetName("item-box")

			text := builder.GetObject("ItemText").Cast().(*gtk.Label)
			text.SetName("item-text")

			image := builder.GetObject("ItemImage").Cast().(*gtk.Image)
			image.SetName("item-image")

			if val.Fuzzyinfo != nil {
				if !text.Wrap() {
					if val.Fuzzyinfo.Start > int32(len(val.Text))/2 {
						text.SetEllipsize(1)
					} else {
						text.SetEllipsize(3)
					}
				}
			}

			if text != nil {
				text.SetLabel(strings.TrimPrefix(val.Text, homedir))
			}

			fileinfo := gio.NewFileForPath(val.Text)

			info, err := fileinfo.QueryInfo(context.Background(), "standard::icon", gio.FileQueryInfoNone)
			if err == nil {
				fi := info.Icon()
				image.SetFromGIcon(fi)
			}

			if image != nil && val.Icon != "" {
				if filepath.IsAbs(val.Icon) {
					image.SetFromFile(val.Icon)
				} else {
					image.SetFromIconName(val.Icon)
				}
			}

			item.SetChild(box)
		case "symbols":
			builder := gtk.NewBuilderFromString(itemBuilders["symbols"])

			box := builder.GetObject("ItemBox").Cast().(*gtk.Box)
			box.AddCSSClass(val.Provider)
			box.SetName("item-box")

			text := builder.GetObject("ItemText").Cast().(*gtk.Label)
			text.SetName("item-text")

			image := builder.GetObject("ItemImage").Cast().(*gtk.Label)
			image.SetName("item-image")
			image.SetLabel(val.Text)

			if val.Fuzzyinfo != nil {
				if !text.Wrap() {
					if val.Fuzzyinfo.Start > int32(len(val.Subtext))/2 {
						text.SetEllipsize(1)
					} else {
						text.SetEllipsize(3)
					}
				}
			}

			text.SetLabel(val.Subtext)

			item.SetChild(box)
		case "calc":
			builder := gtk.NewBuilderFromString(itemBuilders["calc"])

			box := builder.GetObject("ItemBox").Cast().(*gtk.Box)
			box.AddCSSClass(val.Provider)
			box.SetName("item-box")

			text := builder.GetObject("ItemText").Cast().(*gtk.Label)
			text.SetName("item-text")

			subtext := builder.GetObject("ItemSubtext").Cast().(*gtk.Label)
			subtext.SetName("item-subtext")

			if text != nil {
				text.SetLabel(val.Text)
			}

			if subtext != nil {
				if val.Subtext != "" {
					subtext.SetLabel(val.Subtext)
				} else {
					subtext.SetVisible(false)
				}
			}

			item.SetChild(box)
		case "clipboard":
			builder := gtk.NewBuilderFromString(itemBuilders["clipboard"])

			box := builder.GetObject("ItemBox").Cast().(*gtk.Box)
			box.AddCSSClass(val.Provider)
			box.SetName("item-box")

			text := builder.GetObject("ItemText").Cast().(*gtk.Label)
			text.SetName("item-text")

			subtext := builder.GetObject("ItemSubtext").Cast().(*gtk.Label)
			subtext.SetName("item-subtext")

			image := builder.GetObject("ItemImage").Cast().(*gtk.Image)
			image.SetName("item-image")

			if text != nil {
				text.SetLabel(val.Text)
			}

			if subtext != nil {
				if val.Subtext != "" {
					subtext.SetLabel(val.Subtext)
				} else {
					subtext.SetVisible(false)
				}
			}

			if val.Type == pb.QueryResponse_FILE {
				text.SetLabel(val.Mimetype)
				image.SetFromFile(val.Text)
			} else {
				image.SetVisible(false)
			}

			item.SetChild(box)
		default:
			builder := gtk.NewBuilderFromString(itemBuilders["default"])

			box := builder.GetObject("ItemBox").Cast().(*gtk.Box)
			box.AddCSSClass(val.Provider)
			box.SetName("item-box")

			text := builder.GetObject("ItemText").Cast().(*gtk.Label)
			text.SetName("item-text")

			subtext := builder.GetObject("ItemSubtext").Cast().(*gtk.Label)
			subtext.SetName("item-subtext")

			image := builder.GetObject("ItemImage").Cast().(*gtk.Image)
			image.SetName("item-image")

			if text != nil {
				text.SetLabel(val.Text)
			}

			if subtext != nil {
				if val.Subtext != "" {
					subtext.SetLabel(val.Subtext)
				} else {
					subtext.SetVisible(false)
				}
			}

			if image != nil && val.Icon != "" {
				if filepath.IsAbs(val.Icon) {
					image.SetFromFile(val.Icon)
				} else {
					image.SetFromIconName(val.Icon)
				}
			}

			item.SetChild(box)
		}
	})

	return f
}

func setupInteractions(app *gtk.Application) {
	controller := gtk.NewEventControllerKey()
	controller.SetPropagationPhase(gtk.PropagationPhase(1))
	controller.ConnectKeyPressed(func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Return:
			data.Activate(selection.Selected(), currentBuilder.input.Text())
			app.Quit()
			return true
		case gdk.KEY_Escape:
			app.Quit()
			return true
		case gdk.KEY_Down:
			selection.SetSelected(selection.Selected() + 1)
			return true
		case gdk.KEY_Up:
			selection.SetSelected(selection.Selected() - 1)
			return true
		default:
		}

		return false
	})

	layoutBuilders["default"].window.AddController(controller)
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
