package modules

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
)

const (
	GoogleURL     = "https://www.google.com/search?q=%TERM%"
	DuckDuckGoURL = "https://duckduckgo.com/?t=h_&q=%TERM%"
	EcosiaURL     = "https://www.ecosia.org/search?q=%TERM%"
	YandexURL     = "https://yandex.com/search/?text=%TERM%"
)

type Websearch struct {
	prefix            string
	switcherExclusive bool
	specialLabel      string
	engines           []string
	engineInfo        map[string]EngineInfo
}

type EngineInfo struct {
	Label string
	URL   string
}

func (w Websearch) SwitcherExclusive() bool {
	return w.switcherExclusive
}

func (w Websearch) Setup(cfg *config.Config) Workable {
	module := Find(cfg.Modules, w.Name())
	if module == nil {
		return nil
	}

	w.engines = cfg.Websearch.Engines
	w.prefix = module.Prefix
	w.switcherExclusive = module.SwitcherExclusive
	w.specialLabel = module.SpecialLabel

	slices.Reverse(w.engines)

	if len(w.engines) == 0 {
		w.engines = []string{"google"}
	}

	w.engineInfo = make(map[string]EngineInfo)

	w.engineInfo["google"] = EngineInfo{
		Label: "Google",
		URL:   GoogleURL,
	}

	w.engineInfo["duckduckgo"] = EngineInfo{
		Label: "DuckDuckGo",
		URL:   DuckDuckGoURL,
	}

	w.engineInfo["ecosia"] = EngineInfo{
		Label: "Ecosia",
		URL:   EcosiaURL,
	}

	w.engineInfo["yandex"] = EngineInfo{
		Label: "Yandex",
		URL:   YandexURL,
	}

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

	for k, v := range w.engines {
		if val, ok := w.engineInfo[strings.ToLower(v)]; ok {
			url := strings.ReplaceAll(val.URL, "%TERM%", url.QueryEscape(term))

			n := Entry{
				Label:      fmt.Sprintf("Search with %s", val.Label),
				Sub:        "Websearch",
				Exec:       fmt.Sprintf("xdg-open %s", url),
				Class:      "websearch",
				ScoreFinal: float64(k + 1),
			}

			if len(w.engines) == 1 {
				n.SpecialLabel = w.specialLabel
			}

			entries = append(entries, n)
		}
	}

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
