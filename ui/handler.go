package ui

import (
	"context"
	"slices"
	"strings"

	"github.com/abenz1267/walker/modules"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type Handler struct {
	receiver chan []modules.Entry
	entries  []modules.Entry
	ctx      context.Context
}

func (h *Handler) handle() {
	for {
		select {
		case entries := <-h.receiver:
			if len(entries) == 0 {
				continue
			}

			h.entries = append(h.entries, entries...)

			sortEntries(h.entries)

			list := []string{}

			for _, v := range h.entries {
				list = append(list, v.Identifier)
			}

			if len(list) > 0 {
				glib.IdleAdd(func() {
					ui.items.Splice(0, ui.items.NItems(), list)
					ui.selection.SetSelected(0)
				})
			}
		case <-h.ctx.Done():
			return
		default:
		}
	}
}

func sortEntries(entries []modules.Entry) {
	slices.SortFunc(entries, func(a, b modules.Entry) int {
		if a.ScoreFinal == b.ScoreFinal {
			if !a.LastUsed.IsZero() && !b.LastUsed.IsZero() {
				return b.LastUsed.Compare(a.LastUsed)
			}

			return strings.Compare(a.Label, b.Label)
		}

		if a.ScoreFinal > b.ScoreFinal {
			return -1
		}

		if a.ScoreFinal < b.ScoreFinal {
			return 1
		}

		return 0
	})
}
