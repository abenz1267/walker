package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4-layer-shell/pkg/gtk4layershell"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed layout.xml
var layout string

//go:embed defaultstyle.css
var style string

type UI struct {
	app            *gtk.Application
	builder        *gtk.Builder
	scroll         *gtk.ScrolledWindow
	box            *gtk.Box
	appwin         *gtk.ApplicationWindow
	search         *gtk.Entry
	list           *gtk.ListView
	items          *gtk.StringList
	selection      *gtk.SingleSelection
	factory        *gtk.SignalListItemFactory
	prefixClasses  map[string][]string
	ListAlwaysShow bool
}

func createUI(app *gtk.Application) {
	if !gtk4layershell.IsSupported() {
		log.Fatalln("gtk-layer-shell not supported")
	}

	builder := gtk.NewBuilderFromString(layout, len(layout))

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	cfgDir = filepath.Join(cfgDir, "walker")

	cssFile := filepath.Join(cfgDir, "style.css")

	cssProvider := gtk.NewCSSProvider()
	if _, err := os.Stat(cssFile); err == nil {
		cssProvider.LoadFromPath(cssFile)
	} else {
		cssProvider.LoadFromData(style)
	}

	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), cssProvider, gtk.STYLE_PROVIDER_PRIORITY_USER)

	items := gtk.NewStringList([]string{})

	ui = &UI{
		app:            app,
		builder:        builder,
		scroll:         builder.GetObject("scroll").Cast().(*gtk.ScrolledWindow),
		box:            builder.GetObject("box").Cast().(*gtk.Box),
		appwin:         builder.GetObject("win").Cast().(*gtk.ApplicationWindow),
		search:         builder.GetObject("search").Cast().(*gtk.Entry),
		list:           builder.GetObject("list").Cast().(*gtk.ListView),
		items:          items,
		selection:      gtk.NewSingleSelection(items),
		factory:        gtk.NewSignalListItemFactory(),
		prefixClasses:  make(map[string][]string),
		ListAlwaysShow: config.List.AlwaysShow,
	}

	fc := gtk.NewEventControllerFocus()
	fc.Connect("enter", func() {
		if !measured {
			fmt.Printf("startup time: %s", time.Since(now))
			measured = true
		}
	})
	ui.search.AddController(fc)

	alignments := make(map[string]gtk.Align)
	alignments["fill"] = gtk.AlignFill
	alignments["start"] = gtk.AlignStart
	alignments["end"] = gtk.AlignEnd
	alignments["center"] = gtk.AlignCenter

	if config.Align.Width != 0 {
		ui.box.SetSizeRequest(config.Align.Width, -1)
	}

	if config.List.Height != 0 {
		ui.scroll.SetMaxContentHeight(config.List.Height)

		if config.List.Style == "fixed" {
			ui.list.SetSizeRequest(-1, config.List.Height)
		}
	}

	if config.Align.Horizontal != "" {
		ui.box.SetObjectProperty("halign", alignments[config.Align.Horizontal])
	}

	if config.Align.Vertical != "" {
		ui.box.SetObjectProperty("valign", alignments[config.Align.Vertical])
	}

	if config.Orientation == "horizontal" {
		ui.box.SetObjectProperty("orientation", gtk.OrientationHorizontal)
		ui.search.SetVAlign(gtk.AlignStart)

		// ui.list.SetObjectProperty("orientation", gtk.OrientationHorizontal)
	}

	if config.Placeholder != "" {
		ui.search.SetPlaceholderText(config.Placeholder)
	}

	ui.box.SetMarginBottom(config.Align.Margins.Bottom)
	ui.box.SetMarginTop(config.Align.Margins.Top)
	ui.box.SetMarginStart(config.Align.Margins.Start)
	ui.box.SetMarginEnd(config.Align.Margins.End)

	ui.selection.SetAutoselect(true)

	ui.factory.ConnectSetup(func(item *gtk.ListItem) {
		box := gtk.NewBox(gtk.OrientationHorizontal, 0)
		box.SetFocusable(true)
		item.SetChild(box)
	})

	ui.factory.ConnectBind(func(item *gtk.ListItem) {
		key := item.Item().Cast().(*gtk.StringObject).String()

		if item.Selected() {
			child := item.Child()
			if child != nil {
				box, ok := child.(*gtk.Box)
				if !ok {
					log.Fatalln("child is not a box")
				}

				box.GrabFocus()
				ui.appwin.SetCSSClasses([]string{entries[key].Class})
				ui.search.GrabFocusWithoutSelecting()
			}
		}

		if val, ok := entries[key]; ok {
			child := item.Child()

			if child != nil {
				box, ok := child.(*gtk.Box)
				if !ok {
					log.Fatalln("child is not a box")
				}

				if box.FirstChild() != nil {
					return
				}

				box.SetCSSClasses([]string{"item", val.Class})

				motion := gtk.NewEventControllerMotion()
				motion.ConnectEnter(func(_, _ float64) {
					ui.selection.SetSelected(item.Position())
				})

				click := gtk.NewGestureClick()
				click.ConnectPressed(func(m int, _, _ float64) {
					activateItem(false)
				})

				box.AddController(click)
				box.AddController(motion)

				wrapper := gtk.NewBox(gtk.OrientationVertical, 0)
				wrapper.SetCSSClasses([]string{"textwrapper"})

				if config.Icons.Hide || val.Icon != "" {
					icon := gtk.NewImageFromIconName(val.Icon)
					icon.SetIconSize(gtk.IconSizeLarge)
					icon.SetPixelSize(config.Icons.Size)
					icon.SetCSSClasses([]string{"icon"})
					box.Append(icon)
				}

				box.Append(wrapper)

				top := gtk.NewLabel(val.Label)
				top.SetHAlign(gtk.AlignStart)
				top.SetCSSClasses([]string{"label"})

				wrapper.Append(top)

				if val.Sub != "" {
					bottom := gtk.NewLabel(val.Sub)
					bottom.SetHAlign(gtk.AlignStart)
					bottom.SetCSSClasses([]string{"sub"})

					wrapper.Append(bottom)
				} else {
					wrapper.SetVAlign(gtk.AlignCenter)
				}
			}
		}
	})

	ui.list.SetModel(ui.selection)
	ui.list.SetFactory(&ui.factory.ListItemFactory)

	if !config.List.AlwaysShow {
		ui.list.SetVisible(false)
	}
}
