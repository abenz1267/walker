package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var barHasItems = false

func init() {
	os.MkdirAll(util.ThemeDir(), 0755)
	checkForDefaultCss()
}

func checkForDefaultCss() {
	file := filepath.Join(util.ThemeDir(), "default.css")
	os.Remove(file)

	var pybytes []byte

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Panicln(err)
	}

	pywal := filepath.Join(cacheDir, "wal", "colors-waybar.css")

	if util.FileExists(pywal) {
		var err error

		pybytes, err = os.ReadFile(pywal)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		var err error

		pybytes, err = config.Themes.ReadFile("themes/colors.css")
		if err != nil {
			log.Panicln(err)
		}
	}

	css, err := config.Themes.ReadFile("themes/default.css")
	if err != nil {
		log.Panicln(err)
	}

	if len(pybytes) > 0 {
		css = append(pybytes, css...)
	}

	err = os.WriteFile(file, css, 0o600)
	if err != nil {
		log.Panicln(err)
	}
}

func setupCss(theme string, base []string) {
	var css []byte

	hasCss := false

	if base != nil && len(base) > 0 {
		for _, v := range base {
			toPut := getCSS(v)

			if len(toPut) > 0 {
				hasCss = true

				css = append(css, '\n')
				css = append(css, toPut...)
			}
		}
	}

	toPut := getCSS(theme)

	if len(toPut) > 0 {
		hasCss = true
	}

	if !hasCss {
		var err error

		toPut, err = config.Themes.ReadFile("themes/default.css")
		if err != nil {
			log.Panicln(err)
		}
	}

	css = append(css, '\n')
	css = append(css, toPut...)

	common.cssProvider.LoadFromBytes(glib.NewBytes(css))
}

func getCSS(theme string) []byte {
	var css []byte

	file := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", theme))

	if _, err := os.Stat(file); err == nil {
		css, err = os.ReadFile(file)
		if err != nil {
			log.Panicln(err)
		}
	}

	return css
}

func setupTheme() {
	if layout == nil || elements == nil {
		return
	}

	if layout.AlignMap == nil {
		layout.InitUnitMaps()
	}

	if config.Cfg.IgnoreMouse {
		elements.grid.SetCanTarget(false)
	}

	if !appstate.Password {
		if layout.Window.Box.Scroll.List.Item.Icon.Theme != "" {
			elements.iconTheme = gtk.NewIconTheme()
			elements.iconTheme.SetThemeName(layout.Window.Box.Scroll.List.Item.Icon.Theme)
		} else {
			elements.iconTheme = gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault())
		}
	}

	setupWidgetStyle(&elements.appwin.Widget, &layout.Window.Widget, true)

	setupBarTheme()
	setupBoxTheme()

	if !appstate.Password {
		setupScrollTheme()
		setupAiScrollTheme()
		setupAiListTheme()
		setupListTheme()
	}

	setupBoxWidgetStyle(elements.search, &layout.Window.Box.Search.BoxWidget)

	if !appstate.Password {
		setupInputTheme()
		setupWidgetStyle(&elements.spinner.Widget, &layout.Window.Box.Search.Spinner.Widget, false)
	} else {
		setupPasswordTheme()
	}

	if !appstate.Password {
		elements.spinner.SetVisible(false)
	}
}

func setupListTheme() {
	setupWidgetStyle(&elements.grid.Widget, &layout.Window.Box.Scroll.List.Widget, false)

	elements.grid.SetOrientation(layout.OrientationMap[layout.Window.Box.Scroll.List.Orientation])

	if !layout.Window.Box.Scroll.List.Grid {
		elements.grid.SetMaxColumns(1)
	} else {
		elements.grid.SetMaxColumns(1000)
	}
}

func setupBoxTheme() {
	setupBoxWidgetStyle(elements.box, &layout.Window.Box.BoxWidget)

	if appstate.Password {
		return
	}

	first := elements.box.FirstChild()
	last := elements.box.LastChild()

	var scrolledIsFirst bool

	if first != nil && last != nil {
		_, scrolledIsFirst = first.(*gtk.ScrolledWindow)
	}

	if first != nil && last != nil {

		if layout.Window.Box.Revert {
			if !scrolledIsFirst {
				elements.box.ReorderChildAfter(last, first)
			}
		} else {
			if scrolledIsFirst {
				elements.box.ReorderChildAfter(first, last)
			}
		}

		return
	}

	if config.Cfg.List.Placeholder != "" {
		setupLabelWidgetStyle(elements.listPlaceholder, &layout.Window.Box.Scroll.List.Placeholder)
	}

	if layout.Window.Box.Revert {
		if layout.Window.Box.Bar.Position == "start" {
			elements.box.Append(elements.bar)
		}

		if config.Cfg.List.Placeholder != "" {
			elements.box.Append(elements.listPlaceholder)
		}

		elements.box.Append(elements.scroll)
		elements.box.Append(elements.aiScroll)

		if layout.Window.Box.Bar.Position == "between" {
			elements.box.Append(elements.bar)
		}

		elements.box.Append(elements.search)

		if layout.Window.Box.Bar.Position == "end" {
			elements.box.Append(elements.bar)
		}
	} else {
		if layout.Window.Box.Bar.Position == "start" {
			elements.box.Append(elements.bar)
		}

		elements.box.Append(elements.search)

		if layout.Window.Box.Bar.Position == "between" {
			elements.box.Append(elements.bar)
		}

		elements.box.Append(elements.scroll)
		elements.box.Append(elements.aiScroll)

		if config.Cfg.List.Placeholder != "" {
			elements.box.Append(elements.listPlaceholder)
		}

		if layout.Window.Box.Bar.Position == "end" {
			elements.box.Append(elements.bar)
		}
	}
}

func setupBarTheme() {
	if len(config.Cfg.Bar.Entries) == 0 {
		return
	}

	if layout.Window.Box.Bar.Orientation == "horizontal" {
		elements.bar.SetOrientation(gtk.OrientationHorizontal)
	}

	setupBoxWidgetStyle(elements.bar, &layout.Window.Box.Bar.BoxWidget)

	if !barHasItems {
		for _, v := range config.Cfg.Bar.Entries {
			box := gtk.NewBox(gtk.OrientationHorizontal, 0)
			box.SetCSSClasses([]string{"barentry"})

			setupWidgetStyle(&box.Widget, &layout.Window.Box.Bar.Entry.Widget, false)

			controller := gtk.NewGestureClick()
			controller.SetPropagationPhase(gtk.PropagationPhase(1))
			controller.Connect("pressed", func(gesture *gtk.GestureClick, n int) {
				if v.Module == "" && v.Exec != "" {
					cmd := exec.Command("sh", "-c", wrapWithPrefix(v.Exec))

					cmd.SysProcAttr = &syscall.SysProcAttr{
						Setpgid:    true,
						Pgid:       0,
						Foreground: false,
					}

					err := cmd.Start()
					if err != nil {
						log.Println(err)
					}

					closeAfterActivation(false, false)
				} else {
					handleSwitcher(v.Module)
				}
			})

			box.AddController(controller)

			if v.Icon != "" {
				var icon *gtk.Image

				i := elements.iconTheme.LookupIcon(v.Icon, []string{}, layout.IconSizeIntMap[layout.Window.Box.Bar.Entry.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

				icon = gtk.NewImageFromPaintable(i)

				setupIconWidgetStyle(icon, &layout.Window.Box.Bar.Entry.Icon)

				box.Append(icon)
			}

			if v.Label != "" {
				label := gtk.NewLabel(v.Label)
				setupLabelWidgetStyle(label, &layout.Window.Box.Bar.Entry.Label)

				box.Append(label)
			}

			elements.bar.Append(box)
		}

		barHasItems = true
	}
}

func setupAiListTheme() {
	setupWidgetStyle(&elements.aiList.Widget, &layout.Window.Box.AiScroll.List.Widget, false)
}

func setupAiScrollTheme() {
	vScrollbarPolicy := gtk.PolicyAutomatic
	hScrollbarPolicy := gtk.PolicyAutomatic

	setupWidgetStyle(&elements.aiScroll.Widget, &layout.Window.Box.AiScroll.Widget, false)

	elements.aiScroll.Widget.SetVisible(false)

	vScrollbarPolicy = layout.ScrollPolicyMap[layout.Window.Box.AiScroll.VScrollbarPolicy]

	hScrollbarPolicy = layout.ScrollPolicyMap[layout.Window.Box.AiScroll.HScrollbarPolicy]

	elements.aiScroll.SetOverlayScrolling(layout.Window.Box.AiScroll.OverlayScrolling)
	elements.aiScroll.SetPolicy(hScrollbarPolicy, vScrollbarPolicy)
}

func setupScrollTheme() {
	vScrollbarPolicy := gtk.PolicyAutomatic
	hScrollbarPolicy := gtk.PolicyAutomatic

	setupWidgetStyle(&elements.scroll.Widget, &layout.Window.Box.Scroll.Widget, false)

	vScrollbarPolicy = layout.ScrollPolicyMap[layout.Window.Box.Scroll.VScrollbarPolicy]
	hScrollbarPolicy = layout.ScrollPolicyMap[layout.Window.Box.Scroll.HScrollbarPolicy]

	elements.scroll.SetOverlayScrolling(layout.Window.Box.Scroll.OverlayScrolling)
	elements.scroll.SetPolicy(hScrollbarPolicy, vScrollbarPolicy)

	elements.scroll.SetMinContentWidth(layout.Window.Box.Scroll.List.MinWidth)
	elements.scroll.SetMaxContentWidth(layout.Window.Box.Scroll.List.MaxWidth)

	if layout.Window.Box.Scroll.List.MinHeight == 0 {
		elements.scroll.SetMinContentHeight(layout.Window.Box.Scroll.List.MinHeight)
		elements.scroll.SetMaxContentHeight(layout.Window.Box.Scroll.List.MaxHeight)
	} else {
		elements.scroll.SetMaxContentHeight(layout.Window.Box.Scroll.List.MaxHeight)
		elements.scroll.SetMinContentHeight(layout.Window.Box.Scroll.List.MinHeight)
	}
}

func setupPasswordTheme() {
	setupWidgetStyle(&elements.password.Widget, &layout.Window.Box.Search.Input.Widget, false)
	elements.password.SetName("password")
}

func setupInputTheme() {
	first := elements.search.FirstChild()
	last := elements.search.LastChild()

	var spinnerIsFirst bool

	if first != nil && last != nil {
		_, spinnerIsFirst = first.(*gtk.Spinner)
	}

	if first != nil && last != nil {
		if layout.Window.Box.Search.Revert {
			if !spinnerIsFirst {
				elements.box.ReorderChildAfter(last, first)
			}
		} else {
			if spinnerIsFirst {
				elements.box.ReorderChildAfter(first, last)
			}
		}
	} else {
		if layout.Window.Box.Search.Revert {
			elements.search.Append(elements.spinner)
			elements.search.Append(elements.overlay)
		} else {
			elements.search.Append(elements.overlay)
			elements.search.Append(elements.spinner)
		}
	}

	if layout.Window.Box.Search.Prompt.Text != "" && layout.Window.Box.Search.Prompt.Icon == "" {
		prompt := gtk.NewLabel(layout.Window.Box.Search.Prompt.Text)
		setupLabelWidgetStyle(prompt, &layout.Window.Box.Search.Prompt.LabelWidget)
		elements.search.Prepend(prompt)
	}

	if layout.Window.Box.Search.Prompt.Icon != "" && layout.Window.Box.Search.Prompt.Text == "" {
		var icon *gtk.Image
		if filepath.IsAbs(layout.Window.Box.Search.Prompt.Icon) {
			icon = gtk.NewImageFromFile(layout.Window.Box.Search.Prompt.Icon)
		} else {
			i := elements.iconTheme.LookupIcon(layout.Window.Box.Search.Prompt.Icon, []string{}, layout.IconSizeIntMap[layout.Window.Box.Scroll.List.Item.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

			icon = gtk.NewImageFromPaintable(i)
		}

		setupIconWidgetStyle(icon, &layout.Window.Box.Search.Prompt.ImageWidget)

		firstChild := elements.search.FirstChild()

		if _, ok := firstChild.(*gtk.Image); !ok {
			elements.search.Prepend(icon)
		}
	}

	setupWidgetStyle(&elements.input.Widget, &layout.Window.Box.Search.Input.Widget, false)
	setupWidgetStyle(&elements.typeahead.Widget, &layout.Window.Box.Search.Input.Widget, false)

	if layout.Window.Box.Search.Clear.Icon != "" {
		var icon *gtk.Image

		if filepath.IsAbs(layout.Window.Box.Search.Clear.Icon) {
			icon = gtk.NewImageFromFile(layout.Window.Box.Search.Clear.Icon)
		} else {
			i := elements.iconTheme.LookupIcon(layout.Window.Box.Search.Clear.Icon, []string{}, layout.IconSizeIntMap[layout.Window.Box.Scroll.List.Item.Icon.IconSize], 1, gtk.GetLocaleDirection(), 0)

			icon = gtk.NewImageFromPaintable(i)
		}

		setupIconWidgetStyle(icon, &layout.Window.Box.Search.Clear)

		icon.SetVisible(false)

		controller := gtk.NewGestureClick()
		controller.ConnectPressed(func(press int, x, y float64) {
			elements.input.SetText("")
			elements.typeahead.SetText("")
		})

		icon.AddController(controller)

		elements.clear = icon

		lastChild := elements.search.LastChild()

		if _, ok := lastChild.(*gtk.Image); !ok {
			elements.search.Append(elements.clear)
		}
	}

	elements.typeahead.SetName("typeahead")
}

func setupBoxWidgetStyle(box *gtk.Box, style *config.BoxWidget) {
	if style == nil {
		return
	}

	box.SetOrientation(layout.OrientationMap[style.Orientation])
	box.SetSpacing(style.Spacing)

	setupWidgetStyle(&box.Widget, &style.Widget, false)
}

func setupWidgetStyle(
	widget *gtk.Widget,
	style *config.Widget,
	isAppWin bool,
) {
	if style == nil {
		return
	}

	if !isAppWin {
		if style.Hide {
			widget.SetVisible(false)
			return
		}

		widget.SetVisible(true)
	}

	widget.SetHExpandSet(true)
	widget.SetVExpandSet(true)

	if style.CssClasses != nil && len(style.CssClasses) > 0 {
		widget.SetCSSClasses(style.CssClasses)
	}

	widget.SetName(style.Name)
	widget.SetHAlign(layout.AlignMap[style.HAlign])
	widget.SetHExpand(style.HExpand)
	widget.SetVAlign(layout.AlignMap[style.VAlign])
	widget.SetVExpand(style.VExpand)
	widget.SetMarginBottom(style.Margins.Bottom)
	widget.SetMarginTop(style.Margins.Top)
	widget.SetMarginStart(style.Margins.Start)
	widget.SetMarginEnd(style.Margins.End)
	widget.SetSizeRequest(style.Width, style.Height)
}
