package processors

import (
	"net/url"
	"strings"
)

type Websearch struct {
	Prfx string
}

func (w *Websearch) SetPrefix(val string) {
	w.Prfx = val
}

func (w Websearch) Prefix() string {
	return w.Prfx
}

func (Websearch) Name() string {
	return "websearch"
}

func (w Websearch) Entries(term string) []Entry {
	entries := []Entry{}

	if w.Prfx != "" && len(term) < 2 {
		return entries
	}

	term = strings.TrimPrefix(term, w.Prfx)

	n := Entry{
		Label:      "Search with Google",
		Sub:        "Websearch",
		Img:        "",
		Exec:       "xdg-open https://www.google.com/search?q=" + url.QueryEscape(term),
		Searchable: term,
	}

	entries = append(entries, n)

	return entries
}
