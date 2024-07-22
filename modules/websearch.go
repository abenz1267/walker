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

func (Websearch) KeepSort() bool {
	return false
}

func (w Websearch) IsSetup() bool {
	return w.general.IsSetup
}

func (w Websearch) Placeholder() string {
	if w.general.Placeholder == "" {
		return "websearch"
	}

	return w.general.Placeholder
}

func (w Websearch) SwitcherOnly() bool {
	return w.general.SwitcherOnly
}

func (w *Websearch) Setup(cfg *config.Config) {
	w.engines = cfg.Builtins.Websearch.Engines
	w.general.Prefix = cfg.Builtins.Websearch.Prefix
	w.general.SwitcherOnly = cfg.Builtins.Websearch.SwitcherOnly
	w.general.SpecialLabel = cfg.Builtins.Websearch.SpecialLabel
}

func (w *Websearch) SetupData(_ *config.Config) {
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

	w.general.IsSetup = true
}

func (w Websearch) Refresh() {}

func (w Websearch) Prefix() string {
	return w.general.Prefix
}

func (Websearch) Name() string {
	return "websearch"
}

func (w Websearch) Entries(ctx context.Context, term string) []Entry {
	entries := []Entry{}

	path, _ := exec.LookPath("xdg-open")
	if path == "" {
		log.Println("xdg-open not found. Disabling websearch.")
		return nil
	}

	term = strings.TrimPrefix(term, w.general.Prefix)

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
				n.SpecialLabel = w.general.SpecialLabel
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
