package data

import (
	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Item struct {
	Text    string
	Subtext string
}

var Items *gioutil.ListModel[pb.QueryResponse]

func GetSelection() *gtk.SingleSelection {
	Items = gioutil.NewListModel[pb.QueryResponse]()

	selection := gtk.NewSingleSelection(Items.ListModel)
	selection.SetAutoselect(true)

	return selection
}
