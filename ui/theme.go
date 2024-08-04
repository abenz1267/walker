package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func setupCss(theme string) {
	var css []byte

	file := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", theme))

	if _, err := os.Stat(file); err == nil {
		css, err = os.ReadFile(file)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		switch cfg.Theme {
		case "kanagawa":
			css, err = config.Themes.ReadFile("themes/kanagawa.css")
			if err != nil {
				log.Panicln(err)
			}

			createThemeFile(css)
		case "catppuccin":
			css, err = config.Themes.ReadFile("themes/catppuccin.css")
			if err != nil {
				log.Panicln(err)
			}

			createThemeFile(css)
		default:
			log.Printf("css file for theme '%s' not found\n", cfg.Theme)
			os.Exit(1)
		}
	}

	common.cssProvider.LoadFromBytes(glib.NewBytes(css))
}

func setupTheme(theme string) {
	if layout == nil || elements == nil {
		return
	}

	if layout.AlignMap == nil {
		layout.InitUnitMaps()
	}

	if layout.Window != nil {
		setupWidgetStyle(&elements.appwin.Widget, &layout.Window.Widget, true)

		if layout.Window.Box != nil {
			setupBoxTheme()

			if !appstate.Password && layout.Window.Box.Scroll != nil {
				setupScrollTheme()

				if layout.Window.Box.Scroll.List != nil {
					setupListTheme()

					if layout.Window.Box.Scroll.List.Item != nil {
						if layout.Window.Box.Scroll.List.Item.Icon != nil {
							if layout.Window.Box.Scroll.List.Item.Icon.Theme != nil && *layout.Window.Box.Scroll.List.Item.Icon.Theme != "" {
								elements.iconTheme = gtk.NewIconTheme()
								elements.iconTheme.SetThemeName(*layout.Window.Box.Scroll.List.Item.Icon.Theme)
							} else {
								elements.iconTheme = gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault())
							}
						}
					}

				}
			}

			if layout.Window.Box.Search != nil {
				setupBoxWidgetStyle(elements.search, &layout.Window.Box.Search.BoxWidget)

				if !appstate.Password {
					if layout.Window.Box.Search.Input != nil {
						setupInputTheme()
					}

					if layout.Window.Box.Search.Spinner != nil {
						setupWidgetStyle(&elements.spinner.Widget, &layout.Window.Box.Search.Spinner.Widget, false)
					}
				} else {
					if layout.Window.Box.Search.Input != nil {
						setupPasswordTheme()
					}
				}
			}
		}
	}

	if !appstate.Password {
		elements.spinner.SetVisible(false)
	}
}

func setupListTheme() {
	setupWidgetStyle(&elements.list.Widget, &layout.Window.Box.Scroll.List.Widget, false)

	if layout.Window.Box.Scroll.List.Orientation != nil {
		elements.list.SetOrientation(layout.OrientationMap[*layout.Window.Box.Scroll.List.Orientation])
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

		if layout.Window.Box.Revert != nil && *layout.Window.Box.Revert {
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

	if layout.Window.Box.Revert != nil && *layout.Window.Box.Revert {
		elements.box.Append(elements.scroll)
		elements.box.Append(elements.search)
	} else {
		elements.box.Append(elements.search)
		elements.box.Append(elements.scroll)
	}
}

func setupScrollTheme() {
	vScrollbarPolicy := gtk.PolicyAutomatic
	hScrollbarPolicy := gtk.PolicyAutomatic

	setupWidgetStyle(&elements.scroll.Widget, &layout.Window.Box.Scroll.Widget, false)

	if layout.Window.Box.Scroll.VScrollbarPolicy != nil {
		vScrollbarPolicy = layout.ScrollPolicyMap[*layout.Window.Box.Scroll.VScrollbarPolicy]
	}

	if layout.Window.Box.Scroll.HScrollbarPolicy != nil {
		hScrollbarPolicy = layout.ScrollPolicyMap[*layout.Window.Box.Scroll.HScrollbarPolicy]
	}

	elements.scroll.SetOverlayScrolling(layout.Window.Box.Scroll.OverlayScrolling != nil && *layout.Window.Box.Scroll.OverlayScrolling)
	elements.scroll.SetPolicy(vScrollbarPolicy, hScrollbarPolicy)

	if layout.Window.Box.Scroll.List.MaxWidth != nil {
		elements.scroll.SetMaxContentWidth(*layout.Window.Box.Scroll.List.MaxWidth)
	}

	if layout.Window.Box.Scroll.List.MinWidth != nil {
		elements.scroll.SetMinContentWidth(*layout.Window.Box.Scroll.List.MinWidth)
	}

	if layout.Window.Box.Scroll.List.MaxHeight != nil {
		elements.scroll.SetMaxContentHeight(*layout.Window.Box.Scroll.List.MaxHeight)
	}

	if layout.Window.Box.Scroll.List.MinHeight != nil {
		elements.scroll.SetMinContentHeight(*layout.Window.Box.Scroll.List.MinHeight)
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
		if layout.Window.Box.Search.Revert != nil && *layout.Window.Box.Search.Revert {
			if !spinnerIsFirst {
				elements.box.ReorderChildAfter(last, first)
			}
		} else {
			if spinnerIsFirst {
				elements.box.ReorderChildAfter(first, last)
			}
		}
	} else {
		if layout.Window.Box.Search.Revert != nil && *layout.Window.Box.Search.Revert {
			elements.search.Append(elements.spinner)
			elements.search.Append(elements.overlay)
		} else {
			elements.search.Append(elements.overlay)
			elements.search.Append(elements.spinner)
		}
	}

	setupWidgetStyle(&elements.input.Widget, &layout.Window.Box.Search.Input.Widget, false)
	setupWidgetStyle(&elements.typeahead.Widget, &layout.Window.Box.Search.Input.Widget, false)

	elements.typeahead.SetName("typeahead")

	if layout.Window.Box.Search.Input != nil {
		show := layout.Window.Box.Search.Input.Icons != nil && *layout.Window.Box.Search.Input.Icons

		elements.input.FirstChild().(*gtk.Image).SetVisible(show)
		elements.input.LastChild().(*gtk.Image).SetVisible(show)
		elements.typeahead.FirstChild().(*gtk.Image).SetVisible(show)
		elements.typeahead.LastChild().(*gtk.Image).SetVisible(show)
	}
}

func setupBoxWidgetStyle(box *gtk.Box, style *config.BoxWidget) {
	if style == nil {
		return
	}

	if style.Orientation != nil {
		box.SetOrientation(layout.OrientationMap[*style.Orientation])
	}

	if style.Spacing != nil {
		box.SetSpacing(*style.Spacing)
	}

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
		if style.Hide != nil && *style.Hide {
			widget.SetVisible(false)
			return
		}

		widget.SetVisible(true)
	}

	widget.SetHExpandSet(true)
	widget.SetVExpandSet(true)

	if style.CssClasses != nil && len(*style.CssClasses) > 0 {
		widget.SetCSSClasses(*style.CssClasses)
	}

	if style.Name != nil {
		widget.SetName(*style.Name)
	}

	if style.HAlign != nil {
		widget.SetHAlign(layout.AlignMap[*style.HAlign])
	}

	if style.HExpand != nil {
		widget.SetHExpand(*style.HExpand)
	}

	if style.VAlign != nil {
		widget.SetVAlign(layout.AlignMap[*style.VAlign])
	}

	if style.VExpand != nil {
		widget.SetVExpand(*style.VExpand)
	}

	if style.Margins != nil {
		if style.Margins.Bottom != nil {
			widget.SetMarginBottom(*style.Margins.Bottom)
		}

		if style.Margins.Top != nil {
			widget.SetMarginTop(*style.Margins.Top)
		}

		if style.Margins.Start != nil {
			widget.SetMarginStart(*style.Margins.Start)
		}

		if style.Margins.End != nil {
			widget.SetMarginEnd(*style.Margins.End)
		}
	}

	height := -1
	width := -1

	if style.Width != nil {
		width = *style.Width
	}

	if style.Height != nil {
		height = *style.Height
	}

	widget.SetSizeRequest(width, height)
}
