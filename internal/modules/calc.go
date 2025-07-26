package modules

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"unicode"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Calc struct {
	config     config.Calc
	hasClip    bool
	history    []calchist
	hiddenhist []calchist
}

type calchist struct {
	in  string
	res string
}

func (c *Calc) General() *config.GeneralModule {
	return &c.config.GeneralModule
}

func (c *Calc) Cleanup() {
	if len(c.hiddenhist) > 0 {
		c.history = append([]calchist{c.hiddenhist[0]}, c.history...)

		if len(c.history) > 10 {
			c.history = c.history[10:]
		}

		c.hiddenhist = []calchist{}
	}
}

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
	c.history = []calchist{}
	c.hiddenhist = []calchist{}

	// to update exchange rates
	cmd := exec.Command("qalc", "-e", "1+1")
	cmd.Start()

	go func() {
		cmd.Wait()
	}()

	return true
}

func (c *Calc) SetupData() {}

func (c *Calc) Entries(term string) []*util.Entry {
	entries := []*util.Entry{}

	useHistory := strings.HasPrefix(term, ">")
	term = strings.TrimPrefix(term, ">")

	if c.config.RequireNumber {
		hasNumber := false

		for _, c := range term {
			if unicode.IsDigit(c) {
				hasNumber = true
			}
		}

		if !hasNumber {
			return []*util.Entry{}
		}
	}

	if term == "" && len(c.history) == 0 {
		return entries
	}

	if term != "" {
		if useHistory {
			term = fmt.Sprintf("%s%s", c.history[0].res, term)
		}

		cmd := exec.Command("qalc", "-t", term)
		out, err := cmd.CombinedOutput()
		if err != nil && len(c.history) == 0 {
			return []*util.Entry{}
		}

		txt := strings.TrimSpace(string(out))

		if txt == "" && len(c.history) == 0 {
			return []*util.Entry{}
		}

		if txt != "" {
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

			entries = append(entries, &res)

			c.hiddenhist = append([]calchist{{
				in:  term,
				res: txt,
			}}, c.hiddenhist...)
		}
	}

	for k, v := range c.history {
		res := util.Entry{
			Label:            fmt.Sprintf("%d. %s = %s", k+1, v.in, v.res),
			Sub:              "Calc",
			ScoreFinal:       100 - float64(k),
			Matching:         util.Fuzzy,
			RecalculateScore: false,
		}

		if c.hasClip {
			res.Exec = "wl-copy"
			res.Piped = util.Piped{
				String: v.res,
				Type:   "string",
			}
		}

		entries = append(entries, &res)
	}

	return entries
}

func (c *Calc) Refresh() {}
