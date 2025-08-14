package previews

import (
	"github.com/abenz1267/elephant/pkg/pb/pb"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type PreviewHandler interface {
	Handle(item *pb.QueryResponse_Item, preview *gtk.Box, builder *gtk.Builder)
}

var Previewers = make(map[string]PreviewHandler)

func Load() {
	Previewers["files"] = &FilesPreviewHandler{}
}
