package modules

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

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

	if strings.ContainsAny(term, ".") && !strings.HasSuffix(term, ".") {
		_, err := url.ParseRequestURI(fmt.Sprintf("https://%s", term))
		if err == nil {
			entries = append(entries, Entry{
				Label:    fmt.Sprintf("Visit https://%s", term),
				Sub:      "Websearch",
				Exec:     "xdg-open https://" + term,
				Class:    "websearch",
				Matching: AlwaysTop,
			})
		}
	}

	return entries
}

var httpClient = &http.Client{
	Timeout: time.Second * 1,
}

func ping(url string) bool {
	resp, err := httpClient.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
