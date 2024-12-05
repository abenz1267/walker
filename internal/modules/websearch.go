package modules

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Websearch struct {
	config    config.Websearch
	threshold int
	prefixes  []string
}

type EngineInfo struct {
	Label string
	URL   string
}

func (w *Websearch) General() *config.GeneralModule {
	return &w.config.GeneralModule
}

func (w Websearch) Cleanup() {}

func (w *Websearch) Setup(cfg *config.Config) bool {
	w.config = cfg.Builtins.Websearch
	w.threshold = cfg.List.VisibilityThreshold

	return true
}

func (w *Websearch) SetupData(_ *config.Config) {
	w.config.IsSetup = true
	w.config.HasInitialSetup = true

	w.prefixes = []string{}

	for _, v := range w.config.Entries {
		w.prefixes = append(w.prefixes, v.Prefix)
	}
}

func (w *Websearch) Refresh() {
	w.config.IsSetup = !w.config.Refresh
}

func (w Websearch) Entries(term string) []util.Entry {
	entries := []util.Entry{}

	path, _ := exec.LookPath("xdg-open")
	if path == "" {
		log.Println("xdg-open not found. Disabling websearch.")
		return nil
	}

	term = strings.TrimPrefix(term, w.config.Prefix)

	prefix := ""

	for _, v := range w.prefixes {
		if strings.HasPrefix(term, v) {
			prefix = v
			break
		}
	}

	term = strings.TrimPrefix(term, prefix)

	for k, v := range w.config.Entries {
		if prefix != "" && v.Prefix != prefix {
			continue
		}

		url := strings.ReplaceAll(v.Url, "%TERM%", url.QueryEscape(term))

		score := float64(k + 1 + w.threshold)

		if prefix != "" {
			score = 1000000
		}

		n := util.Entry{
			Label:            fmt.Sprintf("Search with %s", v.Name),
			Sub:              "Websearch",
			Exec:             fmt.Sprintf("xdg-open %s", url),
			Class:            "websearch",
			ScoreFinal:       score,
			SingleModuleOnly: v.SwitcherOnly && prefix == "",
			Prefix:           prefix,
		}

		entries = append(entries, n)
	}

	if strings.ContainsAny(term, ".") && !strings.HasSuffix(term, ".") {
		_, err := url.ParseRequestURI(fmt.Sprintf("https://%s", term))
		if err == nil {
			entries = append(entries, util.Entry{
				Label:    fmt.Sprintf("Visit https://%s", term),
				Sub:      "Websearch",
				Exec:     "xdg-open https://" + term,
				Class:    "websearch",
				Matching: util.AlwaysTop,
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
