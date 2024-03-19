package modules

import (
	"net/url"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Websearch struct {
	prefix string
}

func (w Websearch) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, w.Name())
	if module == nil {
		return nil
	}

	w.prefix = module.Prefix

	return w
}

func (w Websearch) Prefix() string {
	return w.prefix
}

func (Websearch) Name() string {
	return "websearch"
}

func (w Websearch) Entries(term string) []Entry {
	entries := []Entry{}

	if term == "" {
		return entries
	}

	if w.prefix != "" && len(term) < 2 {
		return entries
	}

	term = strings.TrimPrefix(term, w.prefix)

	n := Entry{
		Label:    "Search with Google",
		Sub:      "Websearch",
		Exec:     "xdg-open https://www.google.com/search?q=" + url.QueryEscape(term),
		Class:    "websearch",
		Matching: AlwaysBottom,
	}

	entries = append(entries, n)

	return entries
}
