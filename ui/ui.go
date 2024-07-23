package ui

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/history"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed layout.xml
var layout string

//go:embed layout_password.xml
var layoutPassword string

//go:embed themes/style.default.css
var defaultStyle []byte

var (
	labels        = []string{"j", "k", "l", ";", "a", "s", "d", "f"}
	labelF        = []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8"}
	usedLabels    []string
	specialLabels = make(map[uint]uint)
)

var (
	cfg       *config.Config
	ui        *UI
	explicits []modules.Workable
	activated []modules.Workable
	hstry     history.History
	appstate  *state.AppState
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
	iconTheme     *gtk.IconTheme
	password      *gtk.PasswordEntry
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
			ui.appwin.SetVisible(true)

			for _, proc := range activated {
				proc.Refresh()
			}

			if len(appstate.ExplicitModules) > 0 {
				setExplicits()
			}

			if len(explicits) == 1 {
				ui.search.SetObjectProperty("placeholder-text", explicits[0].Placeholder())
			}

			if !appstate.IsMeasured && appstate.Dmenu == nil {
				fmt.Printf("startup time: %s\n", time.Since(appstate.Started))
				appstate.IsMeasured = true
			}

			ui.search.GrabFocus()
			process()

			return
		}

		cfg = config.Get(appstate.ExplicitConfig)
		cfg.IsService = appstate.IsService

		if appstate.ExplicitPlaceholder != "" {
			cfg.Search.Placeholder = appstate.ExplicitPlaceholder
		}

		hstry = history.Get()

		if appstate.Password {
			setupUIPassword(app)
		} else {
			setupUI(app)
			setupInteractions(appstate)
		}

		ui.appwin.SetApplication(app)

		gtk4layershell.InitForWindow(&ui.appwin.Window)
		gtk4layershell.SetNamespace(&ui.appwin.Window, "walker")

		if cfg.Search.ForceKeyboardFocus {
			gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeExclusive)
		} else {
			gtk4layershell.SetKeyboardMode(&ui.appwin.Window, gtk4layershell.LayerShellKeyboardModeOnDemand)
		}

		if !cfg.UI.Fullscreen {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerTop)

			if cfg.UI.Anchors.Top {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
			}

			if cfg.UI.Anchors.Bottom {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
			}

			if cfg.UI.Anchors.Left {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
			}

			if cfg.UI.Anchors.Right {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
			}

			if cfg.UI.IgnoreExclusive {
				gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
			}
		} else {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerOverlay)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
			gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)

			if cfg.UI.IgnoreExclusive {
				gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
			}
		}

		ui.appwin.SetVisible(true)
		appstate.HasUI = true
	}
}

func setupUIPassword(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	builder := gtk.NewBuilderFromString(layoutPassword)
	pw := builder.GetObject("password").Cast().(*gtk.PasswordEntry)

	controller := gtk.NewEventControllerKey()
	controller.ConnectKeyPressed(func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Escape:
			fmt.Print("")
			ui.appwin.Close()
			return true
		}

		return false
	})

	pw.AddController(controller)
	pw.Connect("activate", func() {
		fmt.Print(pw.Text())
		ui.appwin.Close()
	})

	ui = &UI{
		appwin:        builder.GetObject("win").Cast().(*gtk.ApplicationWindow),
		box:           builder.GetObject("box").Cast().(*gtk.Box),
		searchwrapper: builder.GetObject("searchwrapper").Cast().(*gtk.Box),
		password:      pw,
	}

	setupUserStylePassword()
}

func setupUI(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	usedLabels = labels
	if cfg.ActivationMode.UseFKeys {
		usedLabels = labelF
	}

	builder := gtk.NewBuilderFromString(layout)

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

	if cfg.UI.Icons.Theme != "" {
		ui.iconTheme = gtk.NewIconTheme()
		ui.iconTheme.SetThemeName(cfg.UI.Icons.Theme)
	} else {
		ui.iconTheme = gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault())
	}

	ui.list.SetSingleClickActivate(true)
	ui.list.ConnectActivate(func(pos uint) {
		activateItem(false, false, false)
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
		if !appstate.IsMeasured && appstate.Dmenu == nil {
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
		if ui.selection.NItems() > 0 {
			ui.selection.SetSelected(0)
		}

		handleListVisibility()
	})
}

func setupUserStylePassword() {
	cssFile := filepath.Join(util.ConfigDir(), appstate.ExplicitStyle)

	cssProvider := gtk.NewCSSProvider()
	if _, err := os.Stat(cssFile); err == nil {
		cssProvider.LoadFromPath(cssFile)
	} else {
		cssProvider.LoadFromString(string(defaultStyle))

		err := os.WriteFile(cssFile, defaultStyle, 0o600)
		if err != nil {
			log.Panicln(err)
		}
	}

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

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

	width := -1
	if cfg.UI.Width != 0 {
		width = cfg.UI.Width
	}

	height := -1
	if cfg.UI.Height != 0 {
		height = cfg.UI.Height
	}

	ui.box.SetSizeRequest(width, height)

	if cfg.List.Width != 0 {
		ui.password.SetSizeRequest(cfg.List.Width, -1)
	}

	if cfg.UI.Horizontal != "" {
		ui.box.SetObjectProperty("halign", alignments[cfg.UI.Horizontal])
	}

	if cfg.UI.Vertical != "" {
		ui.box.SetObjectProperty("valign", alignments[cfg.UI.Vertical])
	}

	if cfg.UI.Orientation == "horizontal" {
		ui.box.SetObjectProperty("orientation", gtk.OrientationHorizontal)
	}

	ui.box.SetMarginBottom(cfg.UI.Margins.Bottom)
	ui.box.SetMarginTop(cfg.UI.Margins.Top)
	ui.box.SetMarginStart(cfg.UI.Margins.Start)
	ui.box.SetMarginEnd(cfg.UI.Margins.End)
}

func setupUserStyle() {
	cssFile := filepath.Join(util.ConfigDir(), appstate.ExplicitStyle)

	cssProvider := gtk.NewCSSProvider()
	if _, err := os.Stat(cssFile); err == nil {
		cssProvider.LoadFromPath(cssFile)
	} else {
		cssProvider.LoadFromString(string(defaultStyle))

		err := os.WriteFile(cssFile, defaultStyle, 0o600)
		if err != nil {
			log.Panicln(err)
		}
	}

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)
	ui.search.SetObjectProperty("search-delay", cfg.Search.Delay)

	if cfg.List.MarginTop != 0 {
		ui.list.SetMarginTop(cfg.List.MarginTop)
	}

	if !cfg.Search.Spinner {
		ui.spinner.SetVisible(false)
	}

	if !cfg.Search.Icons {
		ui.search.FirstChild().(*gtk.Image).SetVisible(false)
		ui.search.LastChild().(*gtk.Image).SetVisible(false)
		ui.typeahead.FirstChild().(*gtk.Image).SetVisible(false)
		ui.typeahead.LastChild().(*gtk.Image).SetVisible(false)
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

	if cfg.List.ScrollbarPolicy != "" {
		ui.scroll.SetPolicy(gtk.PolicyNever, policies[cfg.List.ScrollbarPolicy])
	}

	width := -1
	if cfg.UI.Width != 0 {
		width = cfg.UI.Width
	}

	height := -1
	if cfg.UI.Height != 0 {
		height = cfg.UI.Height
	}

	ui.box.SetSizeRequest(width, height)

	if cfg.List.Height != 0 {
		ui.scroll.SetMaxContentHeight(cfg.List.Height)

		if cfg.List.FixedHeight {
			ui.list.SetSizeRequest(cfg.UI.Width, cfg.List.Height)
			ui.scroll.SetSizeRequest(cfg.UI.Width, cfg.List.Height)
		}
	}

	if cfg.UI.Horizontal != "" {
		ui.box.SetObjectProperty("halign", alignments[cfg.UI.Horizontal])
	}

	if cfg.UI.Vertical != "" {
		ui.box.SetObjectProperty("valign", alignments[cfg.UI.Vertical])
	}

	if cfg.UI.Orientation == "horizontal" {
		ui.box.SetObjectProperty("orientation", gtk.OrientationHorizontal)
		ui.search.SetVAlign(gtk.AlignCenter)
		ui.typeahead.SetVAlign(gtk.AlignCenter)
		ui.search.SetHExpand(false)
		ui.typeahead.SetHExpand(false)
		ui.list.SetOrientation(gtk.OrientationHorizontal)
		ui.scroll.SetPolicy(policies[cfg.List.ScrollbarPolicy], gtk.PolicyNever)
	}

	ui.scroll.SetMaxContentWidth(cfg.List.Width)
	ui.scroll.SetMaxContentHeight(cfg.List.Height)
	ui.scroll.SetMinContentHeight(cfg.List.Height)
	ui.scroll.SetMinContentWidth(cfg.List.Width)

	if cfg.Search.Placeholder != "" {
		ui.search.SetObjectProperty("placeholder-text", cfg.Search.Placeholder)
	}

	ui.box.SetMarginBottom(cfg.UI.Margins.Bottom)
	ui.box.SetMarginTop(cfg.UI.Margins.Top)
	ui.box.SetMarginStart(cfg.UI.Margins.Start)
	ui.box.SetMarginEnd(cfg.UI.Margins.End)
}

func setupFactory() *gtk.SignalListItemFactory {
	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		item.SetChild(box)
		box.SetFocusable(true)
	})

	factory.ConnectBind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		valObj := ui.items.Item(item.Position())
		val := gioutil.ObjectValue[modules.Entry](valObj)
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

		if val.DragDrop {
			dd := gtk.NewDragSource()
			dd.ConnectPrepare(func(_, _ float64) *gdk.ContentProvider {
				file := gio.NewFileForPath(val.DragDropData)

				b := glib.NewBytes([]byte(fmt.Sprintf("%s\n", file.URI())))

				cp := gdk.NewContentProviderForBytes("text/uri-list", b)

				return cp
			})

			dd.ConnectDragBegin(func(_ gdk.Dragger) {
				ui.appwin.SetVisible(false)
			})

			dd.ConnectDragEnd(func(_ gdk.Dragger, _ bool) {
				closeAfterActivation(false, false)
			})

			box.AddController(dd)
		}

		box.SetCSSClasses([]string{"item", val.Class})

		wrapper := gtk.NewBox(gtk.OrientationVertical, 0)
		wrapper.SetCSSClasses([]string{"textwrapper"})
		wrapper.SetHExpand(true)

		if val.Image != "" {
			image := gtk.NewImageFromFile(val.Image)
			image.SetHExpand(true)
			image.SetSizeRequest(-1, cfg.Builtins.Clipboard.ImageHeight)
			box.Append(image)
		}

		if !cfg.UI.Icons.Hide && val.Icon != "" {
			if val.IconIsImage {
				image := gtk.NewImageFromFile(val.Icon)
				image.SetMarginEnd(10)
				image.SetSizeRequest(cfg.UI.Icons.ImageSize, cfg.UI.Icons.ImageSize)
				box.Append(image)
			} else {
				i := ui.iconTheme.LookupIcon(val.Icon, []string{}, cfg.UI.Icons.Size, 1, gtk.GetLocaleDirection(), 0)
				icon := gtk.NewImageFromPaintable(i)
				icon.SetIconSize(gtk.IconSizeLarge)
				icon.SetPixelSize(cfg.UI.Icons.Size)
				icon.SetCSSClasses([]string{"icon"})
				box.Append(icon)
			}
		}

		if !val.HideText {
			box.Append(wrapper)
		}

		top := gtk.NewLabel(val.Label)
		top.SetMaxWidthChars(0)

		if cfg.UI.Orientation != "horizontal" {
			top.SetWrap(true)
		}

		top.SetHAlign(gtk.AlignStart)
		top.SetCSSClasses([]string{"label"})

		wrapper.Append(top)

		if val.Sub != "" && !cfg.List.HideSub && appstate.Dmenu == nil {
			bottom := gtk.NewLabel(val.Sub)
			bottom.SetMaxWidthChars(0)

			if cfg.UI.Orientation != "horizontal" {
				bottom.SetWrap(true)
			}

			bottom.SetHAlign(gtk.AlignStart)
			bottom.SetCSSClasses([]string{"sub"})

			wrapper.Append(bottom)
		} else {
			wrapper.SetVAlign(gtk.AlignCenter)
		}

		if !cfg.ActivationMode.Disabled {
			if l, ok := cfg.SpecialLabels[fmt.Sprintf("%s;%s", strings.ToLower(val.Label), strings.ToLower(val.Sub))]; ok {
				val.SpecialLabel = l
			}

			if !cfg.ActivationMode.UseFKeys && val.SpecialLabel != "" {
				l := gtk.NewLabel(val.SpecialLabel)
				l.SetCSSClasses([]string{"activationlabel"})
				box.Append(l)

				k := gdk.UnicodeToKeyval(uint32(val.SpecialLabel[0]))
				specialLabels[k] = item.Position()

				return
			}

			if item.Position()+1 <= uint(len(labels)) {
				l := gtk.NewLabel(usedLabels[item.Position()])
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
