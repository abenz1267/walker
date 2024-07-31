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
	"github.com/abenz1267/walker/util"
)

const (
	GoogleURL     = "https://www.google.com/search?q=%TERM%"
	DuckDuckGoURL = "https://duckduckgo.com/?q=%TERM%"
	EcosiaURL     = "https://www.ecosia.org/search?q=%TERM%"
	YandexURL     = "https://yandex.com/search/?text=%TERM%"
)

type Websearch struct {
	general    config.GeneralModule
	engines    []string
	engineInfo map[string]EngineInfo
}

type EngineInfo struct {
	Label string
	URL   string
}

func (w *Websearch) General() *config.GeneralModule {
	return &w.general
}

func (w Websearch) Cleanup() {}

func (w *Websearch) Setup(cfg *config.Config) bool {
	w.engines = cfg.Builtins.Websearch.Engines
	w.general = cfg.Builtins.Websearch.GeneralModule

	return true
}

func (w *Websearch) SetupData(_ *config.Config, ctx context.Context) {
	slices.Reverse(w.engines)

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

	w.general.IsSetup = true
}

func (w *Websearch) Refresh() {
	w.general.IsSetup = !w.general.Refresh
}

func (w Websearch) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	path, _ := exec.LookPath("xdg-open")
	if path == "" {
		log.Println("xdg-open not found. Disabling websearch.")
		return nil
	}

	term = strings.TrimPrefix(term, w.general.Prefix)

	for k, v := range w.engines {
		if val, ok := w.engineInfo[strings.ToLower(v)]; ok {
			url := strings.ReplaceAll(val.URL, "%TERM%", url.QueryEscape(term))

			n := util.Entry{
				Label:      fmt.Sprintf("Search with %s", val.Label),
				Sub:        "Websearch",
				Exec:       fmt.Sprintf("xdg-open %s", url),
				Class:      "websearch",
				ScoreFinal: float64(k + 1),
			}

			entries = append(entries, n)
		}
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
