package modules

import (
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

func (c *Calc) Setup() bool {
	pth, _ := exec.LookPath("qalc")
	if pth == "" {
		log.Println("Calc disabled: currently 'qalc' only.")
		return false
	}

	pthClip, _ := exec.LookPath("wl-copy")
	if pthClip != "" {
		c.hasClip = true
	}

	c.config = config.Cfg.Builtins.Calc

	// to update exchange rates
	cmd := exec.Command("qalc", "-e", "1+1")
	cmd.Start()

	go func() {
		cmd.Wait()
	}()

	return true
}

func (c *Calc) SetupData() {}

func (c Calc) Entries(term string) []util.Entry {
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

	txt := strings.TrimSpace(string(out))

	if txt == "" {
		return entries
	}

	res := util.Entry{
		Label:    txt,
		Sub:      "Calc",
		Matching: util.AlwaysTop,
	}

	if c.hasClip {
		res.Exec = "wl-copy"
		res.Piped = util.Piped{
			String: txt,
			Type:   "string",
		}
	}

	entries = append(entries, res)

	return entries
}

func (c *Calc) Refresh() {}
