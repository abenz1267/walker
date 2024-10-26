package ui

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/history"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/state"
	"github.com/abenz1267/walker/internal/util"
	ls "github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var (
	cfg          *config.Config
	elements     *Elements
	startupTheme string
	layout       *config.UI
	layouts      map[string]*config.UI
	common       *Common
	explicits    []modules.Workable
	toUse        []modules.Workable
	available    []modules.Workable
	hstry        history.History
	appstate     *state.AppState
)

type Common struct {
	items       *gioutil.ListModel[util.Entry]
	selection   *gtk.SingleSelection
	factory     *gtk.SignalListItemFactory
	cssProvider *gtk.CSSProvider
	app         *gtk.Application
}

type Elements struct {
	scroll        *gtk.ScrolledWindow
	overlay       *gtk.Overlay
	spinner       *gtk.Spinner
	search        *gtk.Box
	prompt        *gtk.Label
	box           *gtk.Box
	appwin        *gtk.ApplicationWindow
	typeahead     *gtk.SearchEntry
	input         *gtk.SearchEntry
	grid          *gtk.GridView
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

		layouts = make(map[string]*config.UI)

		hstry = history.Get()
		cfg = config.Get(appstate.ExplicitConfig)

		theme := cfg.Theme
		themeBase := cfg.ThemeBase

		if appstate.ExplicitTheme != "" {
			theme = appstate.ExplicitTheme
			themeBase = nil
		}

		layout = config.GetLayout(theme, themeBase)

		appstate.Labels = strings.Split(cfg.ActivationMode.Labels, "")
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
			cssProvider := gtk.NewCSSProvider()
			gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

			common = &Common{
				cssProvider: cssProvider,
			}

			elements = setupElementsPassword(app)

			setupLayerShell()
		} else {
			setupCommon(app)

			elements = setupElements(app)

			setupLayerShell()

			setupModules()

			afterUI()

			setupInteractions(appstate)
		}

		if singleModule == nil {
			setupLayout(theme, themeBase)
		} else {
			g := singleModule.General()

			if val, ok := layouts[g.Name]; ok {
				layout = val

				theme := g.Theme
				themeBase := g.ThemeBase

				if appstate.ExplicitTheme != "" {
					theme = appstate.ExplicitTheme
					themeBase = nil
				}

				setupLayout(theme, themeBase)
			} else {
				setupLayout(theme, themeBase)
			}
		}

		elements.appwin.SetVisible(true)

		if appstate.Password {
			elements.password.GrabFocus()
		} else {
			elements.input.GrabFocus()
		}

		appstate.HasUI = true
		appstate.IsRunning = true

		if appstate.Benchmark {
			fmt.Println("Visible (first ui)", time.Now().UnixMilli())
		}
	}
}

func setupElementsPassword(app *gtk.Application) *Elements {
	pw := gtk.NewPasswordEntry()

	controller := gtk.NewEventControllerKey()
	controller.ConnectKeyPressed(func(val uint, code uint, modifier gdk.ModifierType) bool {
		switch val {
		case gdk.KEY_Escape:
			elements.appwin.Close()
			return true
		}

		return false
	})

	pw.AddController(controller)
	pw.Connect("activate", func() {
		fmt.Print(pw.Text())
		elements.appwin.Close()
	})

	appwin := gtk.NewApplicationWindow(app)
	appwin.SetApplication(app)

	search := gtk.NewBox(gtk.OrientationVertical, 0)
	search.Append(pw)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(search)

	appwin.SetChild(box)

	ui := &Elements{
		appwin:   appwin,
		box:      box,
		search:   search,
		password: pw,
	}

	return ui
}

func setupCommon(app *gtk.Application) {
	items := gioutil.NewListModel[util.Entry]()
	selection := gtk.NewSingleSelection(items.ListModel)
	selection.SetAutoselect(true)

	factory := setupFactory()

	cssProvider := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

	common = &Common{
		items:       items,
		selection:   selection,
		factory:     factory,
		cssProvider: cssProvider,
		app:         app,
	}
}

func setupElements(app *gtk.Application) *Elements {
	spinner := gtk.NewSpinner()
	search := gtk.NewBox(gtk.OrientationHorizontal, 0)
	typeahead := gtk.NewSearchEntry()
	typeahead.SetCanFocus(false)
	typeahead.SetCanTarget(false)

	prompt := gtk.NewLabel("")

	scroll := gtk.NewScrolledWindow()

	scroll.SetName("scroll")
	scroll.SetPropagateNaturalWidth(true)
	scroll.SetPropagateNaturalHeight(true)

	box := gtk.NewBox(gtk.OrientationVertical, 0)

	appwin := gtk.NewApplicationWindow(app)
	appwin.SetApplication(app)

	input := gtk.NewSearchEntry()

	grid := gtk.NewGridView(common.selection, &common.factory.ListItemFactory)
	scroll.SetChild(grid)

	overlay := gtk.NewOverlay()

	overlay.SetChild(typeahead)
	overlay.AddOverlay(input)

	appwin.SetChild(box)

	ui := &Elements{
		overlay:       overlay,
		spinner:       spinner,
		search:        search,
		prompt:        prompt,
		typeahead:     typeahead,
		scroll:        scroll,
		box:           box,
		appwin:        appwin,
		input:         input,
		grid:          grid,
		prefixClasses: make(map[string][]string),
	}

	if cfg.List.SingleClick {
		ui.grid.SetSingleClickActivate(true)
	}

	ui.grid.ConnectActivate(func(pos uint) {
		activateItem(false, false, false)
	})

	ui.spinner.SetSpinning(true)

	ui.input.SetObjectProperty("search-delay", cfg.Search.Delay)

	if cfg.Search.Placeholder != "" {
		ui.input.SetObjectProperty("placeholder-text", cfg.Search.Placeholder)
	}

	return ui
}

func setupFactory() *gtk.SignalListItemFactory {
	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		overlay := gtk.NewOverlay()
		item.SetChild(overlay)
	})

	factory.ConnectBind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		valObj := common.items.Item(item.Position())
		val := gioutil.ObjectValue[util.Entry](valObj)
		child := item.Child()

		if child == nil {
			return
		}

		overlay, ok := child.(*gtk.Overlay)
		if !ok {
			log.Panicln("child is not a box")
		}

		if overlay.FirstChild() != nil {
			return
		}

		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		box.SetFocusable(true)
		overlay.SetChild(box)

		if val.DragDrop {
			dd := gtk.NewDragSource()
			dd.ConnectPrepare(func(_, _ float64) *gdk.ContentProvider {
				file := gio.NewFileForPath(val.DragDropData)

				b := glib.NewBytes([]byte(fmt.Sprintf("%s\n", file.URI())))

				cp := gdk.NewContentProviderForBytes("text/uri-list", b)

				return cp
			})

			dd.ConnectDragBegin(func(_ gdk.Dragger) {
				elements.appwin.SetVisible(false)
			})

			dd.ConnectDragEnd(func(_ gdk.Dragger, _ bool) {
				closeAfterActivation(false, false)
			})

			box.AddController(dd)
		}

		boxClasses := []string{"item", val.Class}

		if appstate.ActiveItem != nil && *appstate.ActiveItem >= 0 {
			if item.Position() == uint(*appstate.ActiveItem) {
				boxClasses = append(boxClasses, "active")
			}
		} else if appstate.ActiveItem != nil {
			if item.Position() == common.selection.NItems()-1 {
				boxClasses = append(boxClasses, "active")
			}
		}

		box.SetCSSClasses(boxClasses)

		var icon *gtk.Image

		if val.Image != "" {
			icon = gtk.NewImageFromFile(val.Image)
		}

		if !layout.Window.Box.Scroll.List.Item.Icon.Hide {
			if singleModule == nil || singleModule.General().ShowIconWhenSingle {
				ii := val.Icon

				if ii == "" {
					ii = findModule(val.Module, toUse).General().Icon
				}

				if ii != "" {
					if filepath.IsAbs(ii) {
						icon = gtk.NewImageFromFile(ii)
					} else {
						i := elements.iconTheme.LookupIcon(ii, []string{}, layout.IconSizeIntMap[layout.Window.Box.Scroll.List.Item.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

						icon = gtk.NewImageFromPaintable(i)
					}
				}
			}
		}

		label := gtk.NewLabel(val.Label)
		sub := gtk.NewLabel(val.Sub)

		var activationLabel *gtk.Label

		if !cfg.ActivationMode.Disabled {
			if item.Position()+1 <= uint(len(appstate.Labels)) {
				aml := appstate.UsedLabels[item.Position()]

				if !cfg.ActivationMode.UseFKeys && !layout.Window.Box.Scroll.List.Item.ActivationLabel.HideModifier {
					aml = fmt.Sprintf("%s%s", amLabel, aml)
				}

				activationLabel = gtk.NewLabel(aml)
			}
		}

		text := gtk.NewBox(gtk.OrientationVertical, 0)

		setupBoxWidgetStyle(box, &layout.Window.Box.Scroll.List.Item.BoxWidget)

		if layout.Window.Box.Scroll.List.Item.Revert {
			if activationLabel != nil {
				if layout.Window.Box.Scroll.List.Item.ActivationLabel.Overlay {
					overlay.AddOverlay(activationLabel)
				} else {
					box.Append(activationLabel)
				}
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
				if layout.Window.Box.Scroll.List.Item.ActivationLabel.Overlay {
					overlay.AddOverlay(activationLabel)
				} else {
					box.Append(activationLabel)
				}
			}
		}

		setupBoxWidgetStyle(text, &layout.Window.Box.Scroll.List.Item.Text.BoxWidget)

		if layout.Window.Box.Scroll.List.Item.Text.Revert {
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
			setupLabelWidgetStyle(label, &layout.Window.Box.Scroll.List.Item.Text.Label)
		}

		if sub != nil {
			setupLabelWidgetStyle(sub, &layout.Window.Box.Scroll.List.Item.Text.Sub)
		}

		if activationLabel != nil {
			setupLabelWidgetStyle(activationLabel, &layout.Window.Box.Scroll.List.Item.ActivationLabel.LabelWidget)
			activationLabel.SetWrap(false)
		}

		if icon != nil {
			setupIconWidgetStyle(icon, &layout.Window.Box.Scroll.List.Item.Icon)
		}
	})

	return factory
}

func setupIconWidgetStyle(icon *gtk.Image, style *config.ImageWidget) {
	setupWidgetStyle(&icon.Widget, &style.Widget, false)

	icon.SetIconSize(layout.IconSizeMap[style.IconSize])

	icon.SetPixelSize(style.PixelSize)

	if style.CssClasses != nil && len(style.CssClasses) > 0 {
		icon.SetCSSClasses(style.CssClasses)
	}

	icon.SetName(style.Name)
}

func setupLabelWidgetStyle(label *gtk.Label, style *config.LabelWidget) {
	setupWidgetStyle(&label.Widget, &style.Widget, false)

	label.SetWrap(style.Wrap)
	label.SetJustify(layout.JustifyMap[style.Justify])
	label.SetXAlign(style.XAlign)
	label.SetYAlign(style.YAlign)
}

func handleListVisibility() {
	show := common.items.NItems() != 0

	if layout.Window.Box.Scroll.List.AlwaysShow {
		show = layout.Window.Box.Scroll.List.AlwaysShow
	}

	elements.grid.SetVisible(show)
	elements.scroll.SetVisible(show)
}

func reopen() {
	if appstate.IsRunning {
		return
	}

	appstate.IsRunning = true

	go func() {
		for _, proc := range toUse {
			if proc.General().HasInitialSetup {
				proc.Refresh()
			}
		}
	}()

	if len(appstate.ExplicitModules) > 0 {
		setExplicits()
		toUse = explicits
	} else {
		toUse = available
	}

	setupSingleModule()

	if singleModule != nil {
		if val, ok := layouts[singleModule.General().Name]; ok {
			layout = val

			theme := singleModule.General().Theme
			themeBase := singleModule.General().ThemeBase

			if appstate.ExplicitTheme != "" {
				theme = appstate.ExplicitTheme
				themeBase = nil
			}

			setupLayout(theme, themeBase)
		}
	}

	elements.appwin.SetVisible(true)

	if appstate.Benchmark {
		fmt.Println("Visible (re-open)", time.Now().UnixMilli())
	}

	if len(toUse) == 1 {
		text := toUse[0].General().Placeholder

		if appstate.ExplicitPlaceholder != "" {
			text = appstate.ExplicitPlaceholder
		}

		elements.input.SetObjectProperty("placeholder-text", text)
	}

	if appstate.InitialQuery != "" {
		elements.input.SetText(appstate.InitialQuery)
		glib.IdleAdd(func() {
			elements.input.SetPosition(-1)
		})
	}

	elements.input.GrabFocus()

	process()
}

func createThemeFile(data []byte) {
	err := os.WriteFile(filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", cfg.Theme)), data, 0o600)
	if err != nil {
		log.Panicln(err)
	}
}

func afterUI() {
	handleListVisibility()

	if appstate.InitialQuery != "" {
		elements.input.SetText(appstate.InitialQuery)
		glib.IdleAdd(func() {
			elements.input.SetPosition(-1)
		})
	}

	common.selection.ConnectItemsChanged(func(p, r, a uint) {
		if common.selection.NItems() > 0 {
			common.selection.SetSelected(0)
			elements.grid.ScrollTo(0, gtk.ListScrollNone, nil)
		}

		handleListVisibility()
	})
}

func setupLayerShell() {
	if cfg.AsWindow {
		return
	}

	if !ls.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	ls.InitForWindow(&elements.appwin.Window)
	ls.SetNamespace(&elements.appwin.Window, "walker")

	if !cfg.ForceKeyboardFocus {
		ls.SetKeyboardMode(&elements.appwin.Window, ls.LayerShellKeyboardModeOnDemand)
	} else {
		ls.SetKeyboardMode(&elements.appwin.Window, ls.LayerShellKeyboardModeExclusive)
	}

	if layout != nil {
		if layout.IgnoreExclusive {
			ls.SetExclusiveZone(&elements.appwin.Window, -1)
		}

		if !layout.Fullscreen {
			ls.SetLayer(&elements.appwin.Window, ls.LayerShellLayerTop)
		} else {
			ls.SetLayer(&elements.appwin.Window, ls.LayerShellLayerOverlay)
		}
	}
}

func setupLayerShellAnchors() {
	if cfg.AsWindow {
		return
	}

	if layout != nil {
		ls.SetAnchor(&elements.appwin.Window, ls.LayerShellEdgeTop, layout.Anchors.Top)
		ls.SetAnchor(&elements.appwin.Window, ls.LayerShellEdgeBottom, layout.Anchors.Bottom)
		ls.SetAnchor(&elements.appwin.Window, ls.LayerShellEdgeLeft, layout.Anchors.Left)
		ls.SetAnchor(&elements.appwin.Window, ls.LayerShellEdgeRight, layout.Anchors.Right)
	}
}

func setupLayout(theme string, base []string) {
	setupTheme(theme)
	setupCss(theme, base)
	setupLayerShellAnchors()
}
