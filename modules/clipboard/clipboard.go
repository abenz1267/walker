package clipboard

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/util"
)

const ClipboardName = "clipboard"

type Clipboard struct {
	prefix            string
	entries           []ClipboardItem
	file              string
	imgTypes          map[string]string
	max               int
	switcherExclusive bool
}

func (c Clipboard) SwitcherExclusive() bool {
	return c.switcherExclusive
}

type ClipboardItem struct {
	Content string    `json:"content,omitempty"`
	Time    time.Time `json:"time,omitempty"`
	Hash    string    `json:"hash,omitempty"`
	IsImg   bool      `json:"is_img,omitempty"`
}

func (c Clipboard) Entries(ctx context.Context, term string) []modules.Entry {
	entries := []modules.Entry{}

	es := []ClipboardItem{}

	util.FromGob(c.file, &es)

	for _, v := range es {
		e := modules.Entry{
			Label:      v.Content,
			Sub:        "Text",
			RawExec:    []string{"wl-copy", v.Content},
			Categories: []string{"clipboard"},
			Class:      "clipboard",
			Matching:   modules.Fuzzy,
			LastUsed:   v.Time,
		}

		if v.IsImg {
			e.Label = "Image"
			e.Image = v.Content
			e.RawExec = []string{}
			e.Exec = "wl-copy"
			e.Piped = modules.Piped{
				Content: v.Content,
				Type:    "file",
			}
			e.HideText = true
		}

		entries = append(entries, e)
	}

	return entries
}

func (c Clipboard) Prefix() string {
	return c.prefix
}

func (c Clipboard) Name() string {
	return ClipboardName
}

func (c Clipboard) Setup(cfg *config.Config) modules.Workable {
	pth, _ := exec.LookPath("wl-copy")
	if pth == "" {
		log.Println("currently wl-clipboard only.")
		return nil
	}

	pth, _ = exec.LookPath("wl-paste")
	if pth == "" {
		log.Println("currently wl-clipboard only.")
		return nil
	}

	module := modules.Find(cfg.Modules, c.Name())
	if module == nil {
		return nil
	}

	c.prefix = module.Prefix
	c.switcherExclusive = module.SwitcherExclusive
	c.file = filepath.Join(util.CacheDir(), "clipboard.gob")
	c.max = cfg.Clipboard.MaxEntries

	c.imgTypes = make(map[string]string)
	c.imgTypes["image/png"] = "png"
	c.imgTypes["image/jpg"] = "jpg"
	c.imgTypes["image/jpeg"] = "jpeg"

	current := []ClipboardItem{}
	util.FromGob(c.file, &current)

	c.entries = clean(current, c.file)

	go c.watch()

	return c
}

func clean(entries []ClipboardItem, file string) []ClipboardItem {
	cleaned := []ClipboardItem{}

	for _, v := range entries {
		if !v.IsImg {
			cleaned = append(cleaned, v)
			continue
		}

		if _, err := os.Stat(v.Content); err == nil {
			cleaned = append(cleaned, v)
		}
	}

	util.ToGob(&cleaned, file)

	return cleaned
}

func (c Clipboard) exists(hash string) bool {
	for _, v := range c.entries {
		if v.Hash == hash {
			return true
		}
	}

	return false
}

func getType() string {
	cmd := exec.Command("wl-paste", "--list-types")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		log.Panic(err)
	}

	fields := strings.Fields(string(out))

	return fields[0]
}

func getContent() (string, string) {
	cmd := exec.Command("wl-paste")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", ""
	}

	txt := strings.TrimSpace(string(out))
	hash := md5.Sum([]byte(txt))
	strg := hex.EncodeToString(hash[:])

	return txt, strg
}

func saveTmpImg(ext string) string {
	cmd := exec.Command("wl-paste")

	file := filepath.Join(util.TmpDir(), fmt.Sprintf("%d.%s", time.Now().Unix(), ext))

	outfile, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer outfile.Close()

	cmd.Stdout = outfile

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	cmd.Wait()

	return file
}

func (c *Clipboard) watch() {
	for {
		time.Sleep(500 * time.Millisecond)

		content, hash := getContent()

		if c.exists(hash) {
			continue
		}

		if len(content) < 2 {
			continue
		}

		mimetype := getType()

		e := ClipboardItem{
			Content: content,
			Time:    time.Now(),
			Hash:    hash,
			IsImg:   false,
		}

		if val, ok := c.imgTypes[mimetype]; ok {
			file := saveTmpImg(val)
			e.Content = file
			e.IsImg = true
		}

		c.entries = append([]ClipboardItem{e}, c.entries...)

		if len(c.entries) >= c.max {
			c.entries = slices.Clone(c.entries[:c.max])
		}

		util.ToGob(&c.entries, c.file)
	}
}
