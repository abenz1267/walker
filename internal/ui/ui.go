package ui

import (
	"context"
	_ "embed"
	"fmt"
	"html"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/history"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/state"
	"github.com/abenz1267/walker/internal/util"
	"github.com/davidbyttow/govips/v2/vips"
	ls "github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/fsnotify/fsnotify"
)

var (
	elements          *Elements
	startupTheme      string
	layout            *config.UI
	layouts           map[string]*config.UI
	common            *Common
	explicits         []modules.Workable
	toUse             []modules.Workable
	available         []modules.Workable
	hstry             history.History
	appstate          *state.AppState
	thumbnails        map[string][]byte
	thumbnailsMutex   sync.Mutex
	debouncedProcess  func(f func())
	debouncedOnSelect func(f func())
	cfgErr            error
	layoutErr         error
)

type Common struct {
	items       *gioutil.ListModel[util.Entry]
	aiItems     *gioutil.ListModel[modules.AnthropicMessage]
	selection   *gtk.SingleSelection
	factory     *gtk.SignalListItemFactory
	aiFactory   *gtk.SignalListItemFactory
	cssProvider *gtk.CSSProvider
	app         *gtk.Application
}

type Elements struct {
	scroll          *gtk.ScrolledWindow
	overlay         *gtk.Overlay
	spinner         *gtk.Spinner
	search          *gtk.Box
	bar             *gtk.Box
	cfgErr          *gtk.Label
	layoutErr       *gtk.Label
	box             *gtk.Box
	appwin          *gtk.ApplicationWindow
	aiScroll        *gtk.ScrolledWindow
	typeahead       *gtk.Entry
	input           *gtk.Entry
	clear           *gtk.Image
	grid            *gtk.GridView
	aiList          *gtk.ListView
	prefixClasses   map[string][]string
	iconTheme       *gtk.IconTheme
	password        *gtk.PasswordEntry
	listPlaceholder *gtk.Label
}

func Activate(state *state.AppState) func(app *gtk.Application) {
	appstate = state
	thumbnails = make(map[string][]byte)

	go setupThumbnails()

	return func(app *gtk.Application) {
		if appstate.HasUI {
			reopen()
			return
		}

		layouts = make(map[string]*config.UI)

		hstry = history.Get()

		if appstate.IsService {
			cfgErr = appstate.ConfigError
		} else {
			cfgErr = config.Get(appstate.ExplicitConfig)
		}

		t := 1

		if config.Cfg.Search.Delay > 0 {
			t = config.Cfg.Search.Delay
		}

		debouncedProcess = util.NewDebounce(time.Millisecond * time.Duration(t))
		debouncedOnSelect = util.NewDebounce(time.Millisecond * 5)

		theme := config.Cfg.Theme
		themeBase := config.Cfg.ThemeBase

		if appstate.ExplicitTheme != "" {
			theme = appstate.ExplicitTheme
			themeBase = nil
		}

		layout, layoutErr = config.GetLayout(theme, themeBase)

		appstate.Labels = strings.Split(config.Cfg.ActivationMode.Labels, "")
		appstate.LabelsF = []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8"}
		appstate.UsedLabels = appstate.Labels

		if config.Cfg.ActivationMode.UseFKeys {
			appstate.UsedLabels = appstate.LabelsF
		}

		config.Cfg.IsService = appstate.IsService

		if appstate.Dmenu == nil {
			if appstate.DmenuSeparator != "" {
				config.Cfg.Builtins.Dmenu.Separator = appstate.DmenuSeparator
			}

			if appstate.DmenuLabelColumn != 0 {
				config.Cfg.Builtins.Dmenu.LabelColumn = appstate.DmenuLabelColumn
			}
		}

		if appstate.ExplicitPlaceholder != "" {
			config.Cfg.Search.Placeholder = appstate.ExplicitPlaceholder
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
			timeoutReset()
		} else {
			elements.input.GrabFocus()
		}

		appstate.HasUI = true
		appstate.IsRunning = true

		handleTimeout()

		if config.Cfg.IsService && config.Cfg.HotreloadTheme {
			go watchTheme()
		}

		if appstate.Benchmark {
			fmt.Println("Visible (first ui)", time.Now().UnixMilli())
		}

		executeEvent(config.EventLaunch, "")

		if !appstate.Password {
			debouncedProcess(process)
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

	if appstate.ExplicitPlaceholder != "" {
		pw.SetObjectProperty("placeholder-text", appstate.ExplicitPlaceholder)
	}

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
	aiItems := gioutil.NewListModel[modules.AnthropicMessage]()

	selection := gtk.NewSingleSelection(items.ListModel)
	selection.SetAutoselect(true)

	selection.ConnectSelectionChanged(func(pos, item uint) {
		executeEvent(config.EventSelection, "")

		if singleModule != nil {
			valObj := common.items.Item(common.selection.Selected())
			entry := gioutil.ObjectValue[util.Entry](valObj)

			debouncedOnSelect(func() {
				executeOnSelect(entry)
			})
		}

		elements.grid.ScrollTo(common.selection.Selected(), gtk.ListScrollNone, nil)
	})

	factory := setupFactory()
	aiFactory := setupAiFactory()

	cssProvider := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

	common = &Common{
		items:       items,
		aiItems:     aiItems,
		selection:   selection,
		factory:     factory,
		aiFactory:   aiFactory,
		cssProvider: cssProvider,
		app:         app,
	}
}

func setupElements(app *gtk.Application) *Elements {
	spinner := gtk.NewSpinner()
	spinner.SetName("spinner")

	search := gtk.NewBox(gtk.OrientationHorizontal, 0)
	typeahead := gtk.NewEntry()
	typeahead.SetCanFocus(false)
	typeahead.SetCanTarget(false)

	scroll := gtk.NewScrolledWindow()

	scroll.SetName("scroll")
	scroll.SetPropagateNaturalWidth(true)
	scroll.SetPropagateNaturalHeight(true)

	box := gtk.NewBox(gtk.OrientationVertical, 0)

	appwin := gtk.NewApplicationWindow(app)
	appwin.SetApplication(app)

	input := gtk.NewEntry()
	input.SetCanFocus(true)
	input.SetCanTarget(true)
	input.SetFocusable(true)

	grid := gtk.NewGridView(common.selection, &common.factory.ListItemFactory)
	scroll.SetChild(grid)

	overlay := gtk.NewOverlay()

	overlay.SetChild(typeahead)
	overlay.AddOverlay(input)

	appwin.SetChild(box)

	bar := gtk.NewBox(gtk.OrientationVertical, 0)

	var listPlaceholder *gtk.Label

	if config.Cfg.List.Placeholder != "" {
		listPlaceholder = gtk.NewLabel(config.Cfg.List.Placeholder)
		listPlaceholder.SetVisible(false)
	}

	aiScroll := gtk.NewScrolledWindow()
	scroll.SetPropagateNaturalWidth(true)
	scroll.SetPropagateNaturalHeight(true)

	aiList := gtk.NewListView(gtk.NewNoSelection(common.aiItems.ListModel), &common.aiFactory.ListItemFactory)

	aiScroll.SetChild(aiList)

	ui := &Elements{
		listPlaceholder: listPlaceholder,
		bar:             bar,
		overlay:         overlay,
		spinner:         spinner,
		search:          search,
		typeahead:       typeahead,
		scroll:          scroll,
		box:             box,
		appwin:          appwin,
		input:           input,
		grid:            grid,
		aiScroll:        aiScroll,
		aiList:          aiList,
		prefixClasses:   make(map[string][]string),
	}

	if cfgErr != nil {
		label := gtk.NewLabel(fmt.Sprintf("Error loading config:\n\n%s", cfgErr.Error()))
		label.SetName("cfgerr")
		label.SetHAlign(gtk.AlignFill)
		label.SetXAlign(0.0)
		label.SetHExpand(true)
		label.SetHExpandSet(true)
		ui.cfgErr = label
	}

	if layoutErr != nil {
		label := gtk.NewLabel(fmt.Sprintf("Error loading layout:\n\n%s", layoutErr.Error()))
		label.SetName("cfgerr")
		label.SetHAlign(gtk.AlignFill)
		label.SetXAlign(0.0)
		label.SetHExpand(true)
		label.SetHExpandSet(true)
		ui.layoutErr = label
	}

	if config.Cfg.List.SingleClick {
		ui.grid.SetSingleClickActivate(true)
	}

	ui.grid.ConnectActivate(func(pos uint) {
		activateItem(false, false)
	})

	ui.spinner.SetSpinning(true)

	if config.Cfg.Search.Placeholder != "" {
		ui.input.SetObjectProperty("placeholder-text", config.Cfg.Search.Placeholder)
	}

	return ui
}

func setupAiFactory() *gtk.SignalListItemFactory {
	factory := gtk.NewSignalListItemFactory()

	factory.ConnectBind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		item.SetSelectable(false)
		item.SetFocusable(false)
		item.SetActivatable(false)

		valObj := common.aiItems.Item(item.Position())
		val := gioutil.ObjectValue[modules.AnthropicMessage](valObj)

		content := val.Content
		label := gtk.NewLabel(content)
		label.SetSelectable(true)

		if val.Role == "user" {
			label.SetText(fmt.Sprintf(">> %s", content))
			label.SetCSSClasses([]string{"aiItem", "user"})
		} else {
			label.SetCSSClasses([]string{"aiItem", "assistant"})
		}

		setupLabelWidgetStyle(label, &layout.Window.Box.AiScroll.List.Item)

		item.SetChild(label)
	})

	return factory
}

func setupFactory() *gtk.SignalListItemFactory {
	factory := gtk.NewSignalListItemFactory()

	factory.ConnectSetup(func(object *coreglib.Object) {
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		box.SetFocusable(true)

		overlay := gtk.NewOverlay()
		overlay.SetChild(box)

		item := object.Cast().(*gtk.ListItem)
		item.SetChild(overlay)
	})

	factory.ConnectUnbind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)
		overlay := item.Child().(*gtk.Overlay)
		box := overlay.Child().(*gtk.Box)

		for box.FirstChild() != nil {
			box.Remove(box.FirstChild())
		}
	})

	factory.ConnectTeardown(func(object *coreglib.Object) {
	})

	factory.ConnectBind(func(object *coreglib.Object) {
		item := object.Cast().(*gtk.ListItem)

		valObj := common.items.Item(item.Position())
		val := gioutil.ObjectValue[util.Entry](valObj)

		overlay := item.Child().(*gtk.Overlay)
		box := overlay.Child().(*gtk.Box)

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
			hash := util.GetMD5Hash(val.Image)

			b, ok := thumbnails[hash]

			if !ok {
				b = createThumbnail(val.Image)
			}

			t, _ := gdk.NewTextureFromBytes(glib.NewBytes(b))
			icon = gtk.NewImageFromPaintable(t)
		}

		if !layout.Window.Box.Scroll.List.Item.Icon.Hide {
			if singleModule == nil || singleModule.General().ShowIconWhenSingle {
				ii := val.Icon

				if ii == "" {
					ii = findModule(val.Module, toUse).General().Icon
				}

				if ii != "" {
					if ii == "file" {
						fileinfo := gio.NewFileForPath(val.DragDropData)

						info, err := fileinfo.QueryInfo(context.Background(), "standard::icon", gio.FileQueryInfoNone)
						if err == nil {
							fi := info.Icon()
							icon = gtk.NewImageFromGIcon(fi)
						}
					} else {
						if filepath.IsAbs(ii) {
							icon = gtk.NewImageFromFile(ii)
						} else {
							i := elements.iconTheme.LookupIcon(ii, []string{}, layout.IconSizeIntMap[layout.Window.Box.Scroll.List.Item.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

							icon = gtk.NewImageFromPaintable(i)
						}
					}
				}
			}
		}

		labelTxt := html.EscapeString(val.Label)

		if appstate.IsDmenu {
			labelTxt = val.Label
		}

		label := gtk.NewLabel(labelTxt)
		label.SetUseMarkup(true)

		if val.Output != "" {
			go func() {
				run := val.Output

				text := elements.input.Text()
				text = strings.TrimPrefix(text, "'")

				module := findModule(val.Module, toUse)

				if module.General().Prefix != "" {
					text = strings.TrimPrefix(text, module.General().Prefix)
				}

				if strings.Contains(run, "%TERM%") {
					run = strings.ReplaceAll(run, "%TERM%", text)
				}

				run = trimArgumentDelimiter(run)

				cmd := exec.Command("sh", "-c", run)

				out, err := cmd.CombinedOutput()
				if err != nil {
					log.Println(err)
				}

				glib.IdleAdd(func() {
					label.SetText(strings.TrimSpace(string(out)))
				})
			}()
		}

		if val.MatchedLabel != "" {
			val.MatchedLabel = strings.ReplaceAll(val.MatchedLabel, "&", "&amp;")
			label.SetMarkup(val.MatchedLabel)
		}

		sub := gtk.NewLabel(html.EscapeString(val.Sub))
		sub.SetUseMarkup(true)

		if val.MatchedSub != "" {
			val.MatchedSub = strings.ReplaceAll(val.MatchedSub, "&", "&amp;")
			sub.SetMarkup(val.MatchedSub)
		}

		var activationLabel *gtk.Label

		if !config.Cfg.ActivationMode.Disabled {
			if item.Position()+1 <= uint(len(appstate.Labels)) {
				aml := appstate.UsedLabels[item.Position()]

				if !config.Cfg.ActivationMode.UseFKeys && !layout.Window.Box.Scroll.List.Item.ActivationLabel.HideModifier {
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

	if !style.Wrap {
		label.SetEllipsize(3)
	}

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
	timeoutReset()

	if appstate.IsRunning {
		if config.Cfg.CloseWhenOpen {
			if appstate.IsService {
				quit(false)
			} else {
				exit(false, false)
			}
		}

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

	executeEvent(config.EventLaunch, "")
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

	handleTimeout()

	if appstate.InitialQuery != "" {
		glib.IdleAdd(func() {
			elements.input.SetText(appstate.InitialQuery)
			elements.input.SetPosition(-1)
			elements.input.GrabFocus()
		})

		return
	}

	glib.IdleAdd(func() {
		elements.input.GrabFocus()
		process()
	})
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
			if elements.listPlaceholder != nil && elements.listPlaceholder.Visible() {
				elements.listPlaceholder.SetVisible(false)
			}

			common.selection.SetSelected(0)

			if common.items.NItems() == 1 {
				entry := gioutil.ObjectValue[util.Entry](common.items.Item(0))
				module := findModule(entry.Module, toUse)

				if module.General().AutoSelect || appstate.AutoSelect {
					activateItem(false, false)
				}
			}

			elements.grid.ScrollTo(0, gtk.ListScrollNone, nil)

			if singleModule != nil {
				entry := gioutil.ObjectValue[util.Entry](common.items.Item(0))

				debouncedOnSelect(func() {
					executeOnSelect(entry)
				})
			}
		} else {
			if config.Cfg.List.Placeholder != "" && elements.input.Text() != "" {
				elements.listPlaceholder.SetVisible(true)
			}
		}

		handleListVisibility()
	})
}

func setupLayerShell() {
	if config.Cfg.AsWindow {
		return
	}

	if !ls.IsSupported() {
		log.Panicln("gtk-layer-shell not supported")
	}

	ls.InitForWindow(&elements.appwin.Window)
	ls.SetNamespace(&elements.appwin.Window, "walker")

	if config.Cfg.Monitor != "" {
		monitors := gdk.DisplayManagerGet().DefaultDisplay().Monitors()

		for i := 0; i < int(monitors.NItems()); i++ {
			monitor := monitors.Item(uint(i)).Cast().(*gdk.Monitor)

			if monitor.Connector() == config.Cfg.Monitor {
				ls.SetMonitor(&elements.appwin.Window, monitor)
			}
		}
	}

	if !config.Cfg.ForceKeyboardFocus {
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
	if config.Cfg.AsWindow {
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
	setupTheme()

	if layoutErr != nil {
		theme = "default"
		base = []string{}
	}

	setupCss(theme, base)
	setupLayerShellAnchors()
}

func watchTheme() {
	themes := filepath.Join(util.ThemeDir())

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panicln(err)
	}

	err = watcher.Add(themes)
	if err != nil {
		slog.Error("watcher", "add", err)
		return
	}

	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}

				glib.IdleAdd(func() {
					setupLayout(config.Cfg.Theme, config.Cfg.ThemeBase)
				})
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	defer watcher.Close()

	<-make(chan struct{})
}

func createThumbnail(file string) []byte {
	image, err := vips.NewImageFromFile(file)
	if err != nil {
		slog.Error("thumbnail", "error", err)
	}

	err = image.Thumbnail(300, 300, vips.InterestingNone)
	if err != nil {
		slog.Error("thumbnail", "error", err)
	}

	ep := vips.NewDefaultJPEGExportParams()

	b, _, _ := image.Export(ep)
	if err != nil {
		slog.Error("thumbnail", "error", err)
	}

	hash := util.GetMD5Hash(file)

	err = os.WriteFile(filepath.Join(util.ThumbnailsDir(), hash), b, 0o600)
	if err != nil {
		slog.Error("thumbnail", "error", err)
		return b
	}

	thumbnailsMutex.Lock()
	thumbnails[hash] = b
	thumbnailsMutex.Unlock()

	return b
}

func setupThumbnails() {
	filepath.WalkDir(util.ThumbnailsDir(), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			slog.Error("thumbnail", "error", err)
			return nil
		}

		thumbnailsMutex.Lock()
		thumbnails[d.Name()] = b
		thumbnailsMutex.Unlock()

		return nil
	})
}
