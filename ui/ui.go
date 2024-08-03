package ui

import (
	"embed"
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
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed themes/*
var themes embed.FS

var (
	cfg       *config.Config
	ui        *UI
	explicits []modules.Workable
	toUse     []modules.Workable
	available []modules.Workable
	hstry     history.History
	appstate  *state.AppState
)

type UI struct {
	app           *gtk.Application
	scroll        *gtk.ScrolledWindow
	overlay       *gtk.Overlay
	spinner       *gtk.Spinner
	search        *gtk.Box
	box           *gtk.Box
	appwin        *gtk.ApplicationWindow
	typeahead     *gtk.SearchEntry
	input         *gtk.SearchEntry
	list          *gtk.ListView
	items         *gioutil.ListModel[util.Entry]
	selection     *gtk.SingleSelection
	prefixClasses map[string][]string
	iconTheme     *gtk.IconTheme
	password      *gtk.PasswordEntry
}

func Activate(state *state.AppState) func(app *gtk.Application) {
	appstate = state

	return func(app *gtk.Application) {
		if appstate.HasUI {
			reopen()
			return
		}

		hstry = history.Get()
		cfg = config.Get(appstate.ExplicitConfig, appstate.ExplicitTheme)

		appstate.Labels = []string{"j", "k", "l", ";", "a", "s", "d", "f"}
		appstate.LabelsF = []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8"}
		appstate.UsedLabels = appstate.Labels

		if cfg.ActivationMode.UseFKeys {
			appstate.UsedLabels = appstate.LabelsF
		}

		cfg.IsService = appstate.IsService

		if appstate.Dmenu == nil {
			if appstate.DmenuSeparator != "" {
				cfg.Builtins.Dmenu.Separator = appstate.DmenuSeparator
			}

			if appstate.DmenuLabelColumn != 0 {
				cfg.Builtins.Dmenu.LabelColumn = appstate.DmenuLabelColumn
			}
		}

		if appstate.ExplicitPlaceholder != "" {
			cfg.Search.Placeholder = appstate.ExplicitPlaceholder
		}

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

		if cfg.UI != nil && cfg.UI.Fullscreen != nil && !*cfg.UI.Fullscreen {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerTop)

			if cfg.UI.Anchors.Top != nil && *cfg.UI.Anchors.Top {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
			}

			if cfg.UI.Anchors.Bottom != nil && *cfg.UI.Anchors.Bottom {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
			}

			if cfg.UI.Anchors.Left != nil && *cfg.UI.Anchors.Left {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
			}

			if cfg.UI.Anchors.Right != nil && *cfg.UI.Anchors.Right {
				gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
			}

			if cfg.UI.IgnoreExclusive != nil && *cfg.UI.IgnoreExclusive {
				gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
			}
		} else {
			gtk4layershell.SetLayer(&ui.appwin.Window, gtk4layershell.LayerShellLayerOverlay)

			if cfg.UI != nil {
				if cfg.UI.Anchors != nil {
					if cfg.UI.Anchors.Top != nil && *cfg.UI.Anchors.Top {
						gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeTop, true)
					}

					if cfg.UI.Anchors.Bottom != nil && *cfg.UI.Anchors.Bottom {
						gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeBottom, true)
					}

					if cfg.UI.Anchors.Left != nil && *cfg.UI.Anchors.Left {
						gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeLeft, true)
					}

					if cfg.UI.Anchors.Right != nil && *cfg.UI.Anchors.Right {
						gtk4layershell.SetAnchor(&ui.appwin.Window, gtk4layershell.LayerShellEdgeRight, true)
					}

				}

				if cfg.UI.IgnoreExclusive != nil && *cfg.UI.IgnoreExclusive {
					gtk4layershell.SetExclusiveZone(&ui.appwin.Window, -1)
				}
			}
		}

		ui.appwin.SetVisible(true)

		if appstate.Password {
			ui.password.GrabFocus()
		} else {
			ui.input.GrabFocus()
		}

		appstate.HasUI = true
		appstate.IsRunning = true

		if appstate.Benchmark {
			fmt.Println("Visible (first ui)", time.Now().UnixNano())
		}
	}
}

func setupUIPassword(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	pw := gtk.NewPasswordEntry()

	controller := gtk.NewEventControllerKey()
	controller.ConnectKeyPressed(func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Escape:
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

	appwin := gtk.NewApplicationWindow(app)

	search := gtk.NewBox(gtk.OrientationVertical, 0)
	search.Append(pw)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(search)

	appwin.SetChild(box)

	ui = &UI{
		appwin:   appwin,
		box:      box,
		search:   search,
		password: pw,
	}

	setupTheme()
}

func setupUI(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	items := gioutil.NewListModel[util.Entry]()
	spinner := gtk.NewSpinner()
	search := gtk.NewBox(gtk.OrientationHorizontal, 0)
	typeahead := gtk.NewSearchEntry()
	typeahead.SetCanFocus(false)
	typeahead.SetCanTarget(false)

	scroll := gtk.NewScrolledWindow()

	scroll.SetName("scroll")
	scroll.SetPropagateNaturalWidth(true)
	scroll.SetPropagateNaturalHeight(true)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	appwin := gtk.NewApplicationWindow(app)
	input := gtk.NewSearchEntry()
	selection := gtk.NewSingleSelection(items.ListModel)
	factory := setupFactory()

	list := gtk.NewListView(selection, &factory.ListItemFactory)
	scroll.SetChild(list)

	overlay := gtk.NewOverlay()

	overlay.SetChild(typeahead)
	overlay.AddOverlay(input)

	appwin.SetChild(box)

	ui = &UI{
		overlay:       overlay,
		app:           app,
		spinner:       spinner,
		search:        search,
		typeahead:     typeahead,
		scroll:        scroll,
		box:           box,
		appwin:        appwin,
		input:         input,
		items:         items,
		list:          list,
		selection:     selection,
		prefixClasses: make(map[string][]string),
	}

	if cfg.List.SingleClick {
		ui.list.SetSingleClickActivate(true)
	}

	ui.list.ConnectActivate(func(pos uint) {
		activateItem(false, false, false)
	})

	ui.spinner.SetSpinning(true)

	ui.selection.SetAutoselect(true)

	ui.input.SetObjectProperty("search-delay", cfg.Search.Delay)

	if cfg.Search.Placeholder != "" {
		ui.input.SetObjectProperty("placeholder-text", cfg.Search.Placeholder)
	}

	setupTheme()
	handleListVisibility()

	if appstate.InitialQuery != "" {
		ui.input.SetText(appstate.InitialQuery)
		glib.IdleAdd(func() {
			ui.input.SetPosition(-1)
		})
	}

	ui.selection.ConnectItemsChanged(func(p, r, a uint) {
		if ui.selection.NItems() > 0 {
			ui.selection.SetSelected(0)
			ui.list.ScrollTo(0, gtk.ListScrollNone, nil)
		}

		handleListVisibility()
	})
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
		val := gioutil.ObjectValue[util.Entry](valObj)
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

		var icon *gtk.Image

		if val.Image != "" {
			icon = gtk.NewImageFromFile(val.Image)
		}

		if (cfg.UI.Window.Box.Scroll.List.Item.Icon.Hide == nil || !*cfg.UI.Window.Box.Scroll.List.Item.Icon.Hide) && val.Icon != "" {
			if filepath.IsAbs(val.Icon) {
				icon = gtk.NewImageFromFile(val.Icon)
			} else {
				i := ui.iconTheme.LookupIcon(val.Icon, []string{}, cfg.UI.IconSizeIntMap[*cfg.UI.Window.Box.Scroll.List.Item.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

				icon = gtk.NewImageFromPaintable(i)
			}
		}

		label := gtk.NewLabel(val.Label)
		sub := gtk.NewLabel(val.Sub)

		var activationLabel *gtk.Label

		if !cfg.ActivationMode.Disabled {
			if item.Position()+1 <= uint(len(appstate.Labels)) {
				activationLabel = gtk.NewLabel(appstate.UsedLabels[item.Position()])
			}
		}

		text := gtk.NewBox(gtk.OrientationVertical, 0)

		setupBoxWidgetStyle(box, &cfg.UI.Window.Box.Scroll.List.Item.BoxWidget)

		if cfg.UI.Window.Box.Scroll.List.Item.Revert != nil && *cfg.UI.Window.Box.Scroll.List.Item.Revert {
			if activationLabel != nil {
				box.Append(activationLabel)
			}

			if text != nil {
				box.Append(text)
			}

			if icon != nil {
				box.Append(icon)
			}
		} else {
			if icon != nil {
				box.Append(icon)
			}

			if text != nil {
				box.Append(text)
			}

			if activationLabel != nil {
				box.Append(activationLabel)
			}
		}

		setupBoxWidgetStyle(text, &cfg.UI.Window.Box.Scroll.List.Item.Text.BoxWidget)

		if cfg.UI.Window.Box.Scroll.List.Item.Text.Revert != nil && *cfg.UI.Window.Box.Scroll.List.Item.Text.Revert {
			if sub != nil && val.Sub != "" {
				if !appstate.IsSingle || (singleModule != nil && singleModule.General().ShowSubWhenSingle) {
					text.Append(sub)
				}
			}

			if label != nil {
				text.Append(label)
			}
		} else {
			if label != nil {
				text.Append(label)
			}

			if sub != nil && val.Sub != "" {
				if !appstate.IsSingle || (singleModule != nil && singleModule.General().ShowSubWhenSingle) {
					text.Append(sub)
				}
			}
		}

		if label != nil {
			setupLabelWidgetStyle(label, cfg.UI.Window.Box.Scroll.List.Item.Text.Label)
		}

		if sub != nil {
			setupLabelWidgetStyle(sub, cfg.UI.Window.Box.Scroll.List.Item.Text.Sub)
		}

		if activationLabel != nil {
			setupLabelWidgetStyle(activationLabel, cfg.UI.Window.Box.Scroll.List.Item.ActivationLabel)
		}

		if icon != nil {
			setupIconWidgetStyle(icon, cfg.UI.Window.Box.Scroll.List.Item.Icon)
		}
	})

	return factory
}

func setupIconWidgetStyle(icon *gtk.Image, style *config.ImageWidget) {
	setupWidgetStyle(&icon.Widget, &style.Widget, false)

	if style.IconSize != nil {
		icon.SetIconSize(cfg.UI.IconSizeMap[*style.IconSize])
	}

	if style.PixelSize != nil {
		icon.SetPixelSize(*style.PixelSize)
	}

	if style.CssClasses != nil && len(*style.CssClasses) > 0 {
		icon.SetCSSClasses(*style.CssClasses)
	}

	if style.Name != nil {
		icon.SetName(*style.Name)
	}
}

func setupLabelWidgetStyle(label *gtk.Label, style *config.LabelWidget) {
	setupWidgetStyle(&label.Widget, &style.Widget, false)

	label.SetWrap(true)

	if style.Justify != nil {
		label.SetJustify(cfg.UI.JustifyMap[*style.Justify])
	}

	if style.XAlign != nil {
		label.SetXAlign(*style.XAlign)
	}

	if style.YAlign != nil {
		label.SetYAlign(*style.YAlign)
	}
}

func handleListVisibility() {
	show := ui.items.NItems() != 0

	if cfg.UI != nil {
		if cfg.UI.Window != nil {
			if cfg.UI.Window.Box != nil {
				if cfg.UI.Window.Box.Scroll != nil {
					if cfg.UI.Window.Box.Scroll.List != nil {
						if cfg.UI.Window.Box.Scroll.List.AlwaysShow != nil && *cfg.UI.Window.Box.Scroll.List.AlwaysShow {
							show = true
						}
					}
				}
			}
		}
	}

	ui.list.SetVisible(show)
	ui.scroll.SetVisible(show)
}

func reopen() {
	if appstate.IsRunning {
		return
	}

	appstate.IsRunning = true

	ui.appwin.SetVisible(true)

	if appstate.Benchmark {
		fmt.Println("Visible (re-open)", time.Now().UnixNano())
	}

	go func() {
		for _, proc := range toUse {
			proc.Refresh()
		}
	}()

	if len(appstate.ExplicitModules) > 0 {
		setExplicits()
		toUse = explicits
	} else {
		toUse = available
	}

	if len(toUse) == 1 {
		text := toUse[0].General().Placeholder

		if appstate.ExplicitPlaceholder != "" {
			text = appstate.ExplicitPlaceholder
		}

		ui.input.SetObjectProperty("placeholder-text", text)
	}

	if appstate.InitialQuery != "" {
		ui.input.SetText(appstate.InitialQuery)
		glib.IdleAdd(func() {
			ui.input.SetPosition(-1)
		})
	}

	setupSingleModule()

	ui.input.GrabFocus()

	process()
}

func createThemeFile(data []byte) {
	err := os.WriteFile(filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", cfg.Theme)), data, 0o600)
	if err != nil {
		log.Panicln(err)
	}
}
