package translation

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

var providers = map[string]Provider{
	"googlefree": &GoogleFree{},
	"deeplfree":  &DeeplFree{},
}

type Provider interface {
	Name() string
	Translate(text, src, dest string) string
}

type Translation struct {
	config     config.Translation
	provider   Provider
	systemLang string
}

func (translation *Translation) Cleanup() {
}

func (translation *Translation) Entries(term string) []*util.Entry {
	entries := []*util.Entry{}

	src, dest := "auto", translation.systemLang

	splits := strings.Split(term, ">")

	if len(splits) == 2 {
		if len(splits[0]) == 2 {
			src = splits[0]
			term = splits[1]
		}

		if len(splits[1]) == 2 {
			dest = splits[1]
			term = splits[0]
		}
	}

	if len(splits) == 3 {
		if len(splits[0]) == 2 {
			src = splits[0]
		}

		if len(splits[2]) == 2 {
			dest = splits[2]
		}

		term = splits[1]
	}

	res := translation.provider.Translate(term, src, dest)

	if res == "" {
		return entries
	}

	entries = append(entries, &util.Entry{
		Label:            strings.TrimSpace(res),
		Sub:              "Translation",
		Exec:             "",
		Class:            "translation",
		Matching:         util.AlwaysTop,
		RecalculateScore: true,
		SpecialFunc:      translation.SpecialFunc,
	})

	return entries
}

func (translation *Translation) General() *config.GeneralModule {
	return &translation.config.GeneralModule
}

func (translation *Translation) Refresh() {
	translation.config.IsSetup = !translation.config.Refresh
}

func (translation *Translation) Setup() bool {
	translation.config = config.Cfg.Builtins.Translation
	translation.config.IsSetup = true

	if provider, ok := providers[translation.config.Provider]; ok {
		translation.provider = provider
	}

	if translation.provider == nil {
		return true
	}

	langFull := config.Cfg.Locale

	if langFull == "" {
		langFull = os.Getenv("LANG")

		lang_messages := os.Getenv("LC_MESSAGES")
		if lang_messages != "" {
			langFull = lang_messages
		}

		lang_all := os.Getenv("LC_ALL")
		if lang_all != "" {
			langFull = lang_all
		}

		langFull = strings.Split(langFull, ".")[0]
	}

	translation.systemLang = strings.Split(langFull, "_")[0]

	return true
}

func (translation *Translation) SetupData() {
}

func (translation *Translation) SpecialFunc(args ...interface{}) {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(args[0].(string))

	err := cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		cmd.Wait()
	}()
}
