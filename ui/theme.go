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

func setupCss() {
	var css []byte

	if appstate.ExplicitTheme != "" {
		cfg.Theme = appstate.ExplicitTheme
	}

	file := filepath.Join(util.ThemeDir(), fmt.Sprintf("%s.css", cfg.Theme))

	if _, err := os.Stat(file); err == nil {
		css, err = os.ReadFile(file)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		switch cfg.Theme {
		case "kanagawa":
			css, err = themes.ReadFile("themes/kanagawa.css")
			if err != nil {
				log.Panicln(err)
			}

			createThemeFile(css)
		case "catppuccin":
			css, err = themes.ReadFile("themes/catppuccin.css")
			if err != nil {
				log.Panicln(err)
			}

			createThemeFile(css)
		default:
			log.Printf("css file for theme '%s' not found\n", cfg.Theme)
			os.Exit(1)
		}
	}

	cssProvider := gtk.NewCSSProvider()
	cssProvider.LoadFromBytes(glib.NewBytes(css))

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)
}

func setupTheme() {
	if cfg.UI == nil {
		return
	}

	cfg.UI.InitUnitMaps()
	setupCss()

	if cfg.UI == nil {
		return
	}

	if cfg.UI.Window != nil {
		setupWidgetStyle(&ui.appwin.Widget, &cfg.UI.Window.Widget, true)

		if cfg.UI.Window.Box != nil {
			setupBoxTheme()

			if !appstate.Password && cfg.UI.Window.Box.Scroll != nil {
				setupScrollTheme()

				if cfg.UI.Window.Box.Scroll.List != nil {
					setupListTheme()

					if cfg.UI.Window.Box.Scroll.List.Item != nil {
						if cfg.UI.Window.Box.Scroll.List.Item.Icon != nil {
							if cfg.UI.Window.Box.Scroll.List.Item.Icon.Theme != nil && *cfg.UI.Window.Box.Scroll.List.Item.Icon.Theme != "" {
								ui.iconTheme = gtk.NewIconTheme()
								ui.iconTheme.SetThemeName(*cfg.UI.Window.Box.Scroll.List.Item.Icon.Theme)
							} else {
								ui.iconTheme = gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault())
							}
						}
					}

				}
			}

			if cfg.UI.Window.Box.Search != nil {
				setupBoxWidgetStyle(ui.search, &cfg.UI.Window.Box.Search.BoxWidget)

				if !appstate.Password {
					if cfg.UI.Window.Box.Search.Input != nil {
						setupInputTheme()
					}

					if cfg.UI.Window.Box.Search.Spinner != nil {
						setupWidgetStyle(&ui.spinner.Widget, &cfg.UI.Window.Box.Search.Spinner.Widget, false)
					}
				} else {
					if cfg.UI.Window.Box.Search.Input != nil {
						setupPasswordTheme()
					}
				}
			}
		}
	}

	if !appstate.Password {
		ui.spinner.SetVisible(false)
	}
}

func setupListTheme() {
	setupWidgetStyle(&ui.list.Widget, &cfg.UI.Window.Box.Scroll.List.Widget, false)

	if cfg.UI.Window.Box.Scroll.List.Orientation != nil {
		ui.list.SetOrientation(cfg.UI.OrientationMap[*cfg.UI.Window.Box.Scroll.List.Orientation])
	}
}

func setupBoxTheme() {
	setupBoxWidgetStyle(ui.box, &cfg.UI.Window.Box.BoxWidget)

	if appstate.Password {
		return
	}

	if cfg.UI.Window.Box.Revert != nil && *cfg.UI.Window.Box.Revert {
		ui.box.Append(ui.scroll)
		ui.box.Append(ui.search)
	} else {
		ui.box.Append(ui.search)
		ui.box.Append(ui.scroll)
	}
}

func setupScrollTheme() {
	vScrollbarPolicy := gtk.PolicyAutomatic
	hScrollbarPolicy := gtk.PolicyAutomatic

	setupWidgetStyle(&ui.scroll.Widget, &cfg.UI.Window.Box.Scroll.Widget, false)

	if cfg.UI.Window.Box.Scroll.VScrollbarPolicy != nil {
		vScrollbarPolicy = cfg.UI.ScrollPolicyMap[*cfg.UI.Window.Box.Scroll.VScrollbarPolicy]
	}

	if cfg.UI.Window.Box.Scroll.HScrollbarPolicy != nil {
		hScrollbarPolicy = cfg.UI.ScrollPolicyMap[*cfg.UI.Window.Box.Scroll.HScrollbarPolicy]
	}

	ui.scroll.SetOverlayScrolling(cfg.UI.Window.Box.Scroll.OverlayScrolling != nil && *cfg.UI.Window.Box.Scroll.OverlayScrolling)
	ui.scroll.SetPolicy(vScrollbarPolicy, hScrollbarPolicy)

	if cfg.UI.Window.Box.Scroll.List.MaxWidth != nil {
		ui.scroll.SetMaxContentWidth(*cfg.UI.Window.Box.Scroll.List.MaxWidth)
	}

	if cfg.UI.Window.Box.Scroll.List.MinWidth != nil {
		ui.scroll.SetMinContentWidth(*cfg.UI.Window.Box.Scroll.List.MinWidth)
	}

	if cfg.UI.Window.Box.Scroll.List.MaxHeight != nil {
		ui.scroll.SetMaxContentHeight(*cfg.UI.Window.Box.Scroll.List.MaxHeight)
	}

	if cfg.UI.Window.Box.Scroll.List.MinHeight != nil {
		ui.scroll.SetMinContentHeight(*cfg.UI.Window.Box.Scroll.List.MinHeight)
	}
}

func setupPasswordTheme() {
	setupWidgetStyle(&ui.password.Widget, &cfg.UI.Window.Box.Search.Input.Widget, false)
	ui.password.SetName("password")
}

func setupInputTheme() {
	if cfg.UI.Window.Box.Search.Revert != nil && *cfg.UI.Window.Box.Search.Revert {
		ui.search.Append(ui.spinner)
		ui.search.Append(ui.overlay)
	} else {
		ui.search.Append(ui.overlay)
		ui.search.Append(ui.spinner)
	}

	setupWidgetStyle(&ui.input.Widget, &cfg.UI.Window.Box.Search.Input.Widget, false)
	setupWidgetStyle(&ui.typeahead.Widget, &cfg.UI.Window.Box.Search.Input.Widget, false)

	ui.typeahead.SetName("typeahead")

	if cfg.UI.Window.Box.Search.Input != nil {
		if cfg.UI.Window.Box.Search.Input.Icons != nil && !*cfg.UI.Window.Box.Search.Input.Icons {
			ui.input.FirstChild().(*gtk.Image).SetVisible(false)
			ui.input.LastChild().(*gtk.Image).SetVisible(false)
			ui.typeahead.FirstChild().(*gtk.Image).SetVisible(false)
			ui.typeahead.LastChild().(*gtk.Image).SetVisible(false)
		}
	}
}

func setupBoxWidgetStyle(box *gtk.Box, style *config.BoxWidget) {
	if style == nil {
		return
	}

	if style.Orientation != nil {
		box.SetOrientation(cfg.UI.OrientationMap[*style.Orientation])
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
		widget.SetHAlign(cfg.UI.AlignMap[*style.HAlign])
	}

	if style.HExpand != nil {
		widget.SetHExpand(*style.HExpand)
	}

	if style.VAlign != nil {
		widget.SetVAlign(cfg.UI.AlignMap[*style.VAlign])
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
