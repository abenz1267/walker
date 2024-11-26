package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Bookmarks struct {
	config  *config.Bookmarks
	entries []util.Entry
}

func (bookmarks *Bookmarks) Cleanup() {
}

func (bookmarks *Bookmarks) Entries(ctx context.Context, term string) []util.Entry {
	if strings.HasPrefix(term, "a:") {
		return []util.Entry{
			{
				Label:    "Add Bookmark",
				Sub:      bookmarks.config.GeneralModule.Name,
				Matching: util.AlwaysTop,
			},
		}
	}

	return bookmarks.entries
}

func (bookmarks *Bookmarks) General() *config.GeneralModule {
	return &bookmarks.config.GeneralModule
}

func (bookmarks *Bookmarks) Refresh() {
	bookmarks.config.IsSetup = !bookmarks.config.Refresh
}

func (bookmarks *Bookmarks) Setup(cfg *config.Config) bool {
	bookmarks.config = &cfg.Builtins.Bookmarks

	return true
}

func (bookmarks *Bookmarks) SetupData(cfg *config.Config, ctx context.Context) {
	bookmarks.entries = []util.Entry{}

	for _, v := range cfg.Builtins.Bookmarks.Entries {
		bookmarks.entries = append(bookmarks.entries, util.Entry{
			Label:            v.Label,
			Sub:              v.Url,
			Categories:       v.Keywords,
			Icon:             cfg.Builtins.Bookmarks.GeneralModule.Icon,
			Exec:             fmt.Sprintf("xdg-open %s", v.Url),
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}
}
