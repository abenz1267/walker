package modules

import (
	"context"
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Websearch struct {
	prefix            string
	switcherExclusive bool
	specialLabel      string
}

func (w Websearch) SwitcherExclusive() bool {
	return w.switcherExclusive
}

func (w Websearch) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, w.Name())
	if module == nil {
		return nil
	}

	w.prefix = module.Prefix
	w.switcherExclusive = module.SwitcherExclusive
	w.specialLabel = module.SpecialLabel

	return w
}

func (w Websearch) Prefix() string {
	return w.prefix
}

func (Websearch) Name() string {
	return "websearch"
}

func (w Websearch) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	if term == "" {
		return entries
	}

	if w.prefix != "" && len(term) < 2 {
		return entries
	}

	path, _ := exec.LookPath("xdg-open")
	if path == "" {
		log.Println("xdg-open not found. Disabling websearch.")
		return nil
	}

	term = strings.TrimPrefix(term, w.prefix)

	n := Entry{
		Label:        "Search with Google",
		Sub:          "Websearch",
		Exec:         "xdg-open https://www.google.com/search?q=" + url.QueryEscape(term),
		Class:        "websearch",
		Matching:     AlwaysBottom,
		SpecialLabel: w.specialLabel,
	}

	entries = append(entries, n)

	return entries
}
