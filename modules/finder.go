package modules

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/config"
)

type Finder struct {
	prefix            string
	switcherExclusive bool
}

func (f Finder) Entries(ctx context.Context, term string) []Entry {
	e := []Entry{}

	if len(term) < 2 {
		return e
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Panic(err)
	}

	fd := exec.Command("fd")
	fd.Dir = homedir

	fzf := exec.Command("fzf", "-f", term)

	r, w := io.Pipe()
	fd.Stdout = w
	fzf.Stdin = r

	var result strings.Builder
	fzf.Stdout = &result

	fd.Start()
	fzf.Start()
	fd.Wait()
	w.Close()
	fzf.Wait()

	scanner := bufio.NewScanner(strings.NewReader(result.String()))

	counter := 0
	for scanner.Scan() {
		v := scanner.Text()

		full := filepath.Join(homedir, v)

		e = append(e, Entry{
			Label:            v,
			Sub:              "fzf",
			Exec:             fmt.Sprintf("xdg-open %s", full),
			DragDrop:         true,
			DragDropData:     full,
			Categories:       []string{"finder", "fzf"},
			Class:            "finder",
			Matching:         Fuzzy,
			RecalculateScore: false,
			ScoreFinal:       float64(100 - counter),
		})

		counter++
	}

	return e
}

func (f Finder) Prefix() string {
	return f.prefix
}

func (f Finder) Name() string {
	return "finder"
}

func (f Finder) SwitcherExclusive() bool {
	return f.switcherExclusive
}

func (Finder) Setup(cfg *config.Config) Workable {
	fd, _ := exec.LookPath("fd")
	fzf, _ := exec.LookPath("fzf")

	if fd == "" || fzf == "" {
		log.Println("fd or fzf not found. Disabling finder.")
		return nil
	}

	f := &Finder{}

	module := Find(cfg.Modules, f.Name())
	if module == nil {
		return nil
	}

	f.prefix = module.Prefix
	f.switcherExclusive = module.SwitcherExclusive

	return f
}
