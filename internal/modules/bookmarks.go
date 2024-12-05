package modules

import (
	"fmt"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Bookmarks struct {
	config   *config.Bookmarks
	entries  []util.Entry
	prefixes []string
}

func (bookmarks *Bookmarks) Cleanup() {
}

func (bookmarks *Bookmarks) Entries(term string) []util.Entry {
	hasPrefix := false

	for _, v := range bookmarks.prefixes {
		if strings.HasPrefix(term, v) {
			hasPrefix = true
			break
		}
	}

	if hasPrefix {
		entries := []util.Entry{}

		for _, v := range bookmarks.entries {
			if v.Prefix != "" && strings.HasPrefix(term, v.Prefix) {
				entries = append(entries, v)
			}
		}

		return entries
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

func (bookmarks *Bookmarks) SetupData(cfg *config.Config) {
	bookmarks.entries = []util.Entry{}
	bookmarks.prefixes = []string{}

	for _, v := range cfg.Builtins.Bookmarks.Entries {
		bookmarks.entries = append(bookmarks.entries, util.Entry{
			Label:            v.Label,
			Sub:              v.Url,
			Categories:       v.Keywords,
			Icon:             cfg.Builtins.Bookmarks.GeneralModule.Icon,
			Exec:             fmt.Sprintf("xdg-open '%s'", v.Url),
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})
	}

	for _, v := range cfg.Builtins.Bookmarks.Groups {
		if v.Prefix != "" {
			bookmarks.prefixes = append(bookmarks.prefixes, v.Prefix)
		}

		for _, entry := range v.Entries {
			bookmarks.entries = append(bookmarks.entries, util.Entry{
				Label:            entry.Label,
				Sub:              fmt.Sprintf("%s: %s", v.Label, entry.Url),
				Categories:       entry.Keywords,
				Icon:             cfg.Builtins.Bookmarks.GeneralModule.Icon,
				Exec:             fmt.Sprintf("xdg-open '%s'", entry.Url),
				Matching:         util.Fuzzy,
				RecalculateScore: true,
				Prefix:           v.Prefix,
				IgnoreUnprefixed: v.IgnoreUnprefixed,
			})
		}
	}
}
