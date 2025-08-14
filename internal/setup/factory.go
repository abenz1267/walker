package setup

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/data"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

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
				text.SetLabel(strings.TrimSpace(val.Text))
			}

			if subtext != nil {
				time, _ := time.Parse(time.RFC1123Z, val.Subtext)
				subtext.SetLabel(time.Format(config.LoadedConfig.Providers.Clipboard.TimeFormat))
			}

			if val.Type == pb.QueryResponse_FILE {
				text.SetVisible(false)
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
					// this leaks if the img is an svg
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
