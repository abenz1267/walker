package modules

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"unicode"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Calc struct {
	config  config.Calc
	hasClip bool
}

func (c *Calc) General() *config.GeneralModule {
	return &c.config.GeneralModule
}

func (c Calc) Cleanup() {}

func (c *Calc) Setup(cfg *config.Config) bool {
	pth, _ := exec.LookPath("qalc")
	if pth == "" {
		log.Println("Calc disabled: currently 'qalc' only.")
		return false
	}

	pthClip, _ := exec.LookPath("wl-copy")
	if pthClip != "" {
		c.hasClip = true
	}

	c.config = cfg.Builtins.Calc

	// to update exchange rates
	cmd := exec.Command("qalc", "-e", "1+1")
	cmd.Start()

	return true
}

func (c *Calc) SetupData(cfg *config.Config, ctx context.Context) {}

func (c Calc) Entries(ctx context.Context, term string) []util.Entry {
	if c.config.RequireNumber {
		hasNumber := false

		for _, c := range term {
			if unicode.IsDigit(c) {
				hasNumber = true
			}
		}

		if !hasNumber {
			return []util.Entry{}
		}
	}

	entries := []util.Entry{}

	cmd := exec.Command("qalc", "-t", term)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return entries
	}

	txt := string(out)

	if txt == "" {
		return entries
	}

	res := util.Entry{
		Label:    strings.TrimSpace(txt),
		Sub:      "Calc",
		Matching: util.AlwaysTop,
	}

	if c.hasClip {
		res.Exec = "wl-copy"
		res.Piped = util.Piped{
			Content: txt,
			Type:    "string",
		}
	}

	entries = append(entries, res)

	return entries
}

func (c *Calc) Refresh() {}
