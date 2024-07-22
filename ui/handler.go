package ui

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type Handler struct {
	receiver chan []modules.Entry
	entries  []modules.Entry
	ctx      context.Context
	keepSort bool
	mut      sync.Mutex
}

func (h *Handler) handle() {
	for {
		select {
		case entries := <-h.receiver:
			if len(entries) == 0 {
				continue
			}

			h.mut.Lock()
			h.entries = append(h.entries, entries...)

			if !appstate.KeepSort && !h.keepSort {
				sortEntries(h.entries)
			}

			if len(h.entries) > cfg.List.MaxEntries {
				h.entries = h.entries[:cfg.List.MaxEntries]
			}

			if len(h.entries) > 0 {
				glib.IdleAdd(func() {
					ui.items.Splice(0, int(ui.items.NItems()), h.entries...)
				})
			}

			h.mut.Unlock()
		case <-h.ctx.Done():
			return
		default:
		}
	}
}

func sortEntries(entries []modules.Entry) {
	slices.SortFunc(entries, func(a, b modules.Entry) int {
		if a.Matching == util.AlwaysTop && b.Matching != util.AlwaysTop {
			return -1
		}

		if b.Matching == util.AlwaysTop && a.Matching != util.AlwaysTop {
			return 1
		}

		if a.Matching == util.AlwaysBottom && b.Matching != util.AlwaysBottom {
			return 1
		}

		if b.Matching == util.AlwaysBottom && a.Matching != util.AlwaysBottom {
			return -1
		}

		if cfg.Builtins.Hyprland.ContextAwareHistory && cfg.IsService {
			if !a.LastUsed.IsZero() && !b.LastUsed.IsZero() {
				if a.OpenWindows > b.OpenWindows {
					return 1
				}

				if a.OpenWindows < b.OpenWindows {
					return -1
				}
			}
		}

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
