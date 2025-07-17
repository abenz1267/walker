package clipboard

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/history"
	"github.com/abenz1267/walker/internal/util"
)

const ClipboardName = "clipboard"

type Clipboard struct {
	general         config.GeneralModule
	items           []ClipboardItem
	entries         []util.Entry
	file            string
	imgTypes        map[string]string
	max             int
	exec            string
	avoidLineBreaks bool
	isWatching      bool
}

type ClipboardItem struct {
	Content string    `json:"content,omitempty"`
	Time    time.Time `json:"time,omitempty"`
	Hash    string    `json:"hash,omitempty"`
	IsImg   bool      `json:"is_img,omitempty"`
}

func (c *Clipboard) General() *config.GeneralModule {
	return &c.general
}

func (c *Clipboard) Refresh() {
	c.general.IsSetup = !c.general.Refresh
}

func (c Clipboard) Cleanup() {}

func (c Clipboard) Entries(term string) []util.Entry {
	for k, v := range c.entries {
		for _, vv := range c.items {
			if v.HashIdent == vv.Hash {
				c.entries[k].LastUsed = vv.Time
			}
		}
	}

	return c.entries
}

func (c *Clipboard) Setup() bool {
	pth, _ := exec.LookPath("wl-copy")
	if pth == "" {
		log.Println("Clipboard disabled: currently wl-clipboard only.")
		return false
	}

	c.general = config.Cfg.Builtins.Clipboard.GeneralModule

	c.file = filepath.Join(util.CacheDir(), "clipboard.gob")
	c.max = config.Cfg.Builtins.Clipboard.MaxEntries
	c.exec = config.Cfg.Builtins.Clipboard.Exec
	c.avoidLineBreaks = config.Cfg.Builtins.Clipboard.AvoidLineBreaks

	c.imgTypes = make(map[string]string)
	c.imgTypes["image/png"] = "png"
	c.imgTypes["image/jpg"] = "jpg"
	c.imgTypes["image/jpeg"] = "jpeg"

	return true
}

func (c *Clipboard) SetupData() {
	current := []ClipboardItem{}
	_ = util.FromGob(c.file, &current)

	c.items = clean(current, c.file)

	for _, v := range c.items {
		c.entries = append(c.entries, itemToEntry(v, c.exec, c.avoidLineBreaks))
	}

	go c.watch()

	c.general.IsSetup = true
	c.general.HasInitialSetup = true
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
	for _, v := range c.items {
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
	cmd := exec.Command("wl-paste", "-n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", ""
	}

	txt := string(out)
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
	if c.isWatching {
		return
	}

	c.isWatching = true

	pid := os.Getpid()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("wl-paste --watch kill -USR1 %d", pid))

	_ = cmd.Run()
}

func (c *Clipboard) Update() {
	cmd := exec.Command("wl-paste")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Nothing is copied") {
			return
		}

		slog.Error("clipboard", "error", err)

		return
	}

	content := string(out)

	hash := md5.Sum([]byte(content))
	strgHash := hex.EncodeToString(hash[:])

	exists := c.exists(strgHash)

	if exists && !config.Cfg.Builtins.Clipboard.AlwaysPutNewOnTop {
		return
	}

	if len(content) < 2 {
		return
	}

	if exists && c.items[0].Hash != strgHash {
		for k, v := range c.items {
			if v.Hash == strgHash {
				c.items[k].Time = time.Now()
			}
		}

		for k, v := range c.entries {
			if v.HashIdent == strgHash {
				c.entries[k].LastUsed = time.Now()
			}
		}

		slices.SortFunc(c.items, func(a, b ClipboardItem) int {
			if a.Time.After(b.Time) {
				return -1
			}

			if a.Time.Before(b.Time) {
				return 1
			}

			return 0
		})

		slices.SortFunc(c.entries, func(a, b util.Entry) int {
			if a.LastUsed.After(b.LastUsed) {
				return -1
			}

			if a.LastUsed.Before(b.LastUsed) {
				return 1
			}

			return 0
		})
	} else if !exists {
		mimetype := getType()

		e := ClipboardItem{
			Content: content,
			Time:    time.Now(),
			Hash:    strgHash,
			IsImg:   false,
		}

		if val, ok := c.imgTypes[mimetype]; ok {
			file := saveTmpImg(val)
			e.Content = file
			e.IsImg = true
		}

		c.entries = append([]util.Entry{itemToEntry(e, c.exec, c.avoidLineBreaks)}, c.entries...)
		c.items = append([]ClipboardItem{e}, c.items...)
	}

	hstry := history.Get()

	toSpareEntries := []util.Entry{}
	toSpareItems := []ClipboardItem{}

	for _, v := range c.entries {
		if hstry.Has(v.Identifier()) {
			toSpareEntries = append(toSpareEntries, v)

			for _, vv := range c.items {
				if v.HashIdent == vv.Hash {
					toSpareItems = append(toSpareItems, vv)
				}
			}
		}
	}

	if len(c.items) >= c.max {
		c.items = slices.Clone(c.items[:c.max])

		for _, v := range toSpareItems {
			if !slices.ContainsFunc(c.items, func(item ClipboardItem) bool {
				if item.Hash == v.Hash {
					return true
				}

				return false
			}) {
				c.items = append(c.items, v)
			}
		}
	}

	if len(c.entries) >= c.max {
		c.entries = slices.Clone(c.entries[:c.max])

		for _, v := range toSpareEntries {
			if !slices.ContainsFunc(c.entries, func(item util.Entry) bool {
				if item.HashIdent == v.HashIdent {
					return true
				}

				return false
			}) {
				c.entries = append(c.entries, v)
			}
		}
	}

	util.ToGob(&c.items, c.file)
}

func itemToEntry(item ClipboardItem, exec string, avoidLineBreaks bool) util.Entry {
	label := strings.TrimSpace(item.Content)

	if avoidLineBreaks {
		label = strings.ReplaceAll(label, "\n", " ")
	}

	entry := util.Entry{
		Label:            label,
		Sub:              "Text",
		Exec:             exec,
		Piped:            util.Piped{String: item.Content, Type: "string"},
		Categories:       []string{"clipboard"},
		Class:            "clipboard",
		Matching:         util.Fuzzy,
		LastUsed:         item.Time,
		RecalculateScore: true,
		HashIdent:        item.Hash,
	}

	if item.IsImg {
		entry.Label = "Image"
		entry.Image = item.Content
		entry.Exec = exec
		entry.Piped = util.Piped{
			String: item.Content,
			Type:   "file",
		}
		entry.HideText = true
		entry.DragDrop = true
		entry.DragDropData = item.Content
	}

	return entry
}

func (c *Clipboard) Delete(entry util.Entry) {
	content := entry.Piped.String

	c.entries = []util.Entry{}

	for k, v := range c.items {
		if v.Content == content {
			max := k + 1

			if max > len(c.items) {
				max = len(c.items)
			}

			c.items = slices.Delete(c.items, k, max)

			continue
		}
	}

	for _, v := range c.items {
		c.entries = append(c.entries, itemToEntry(v, c.exec, c.avoidLineBreaks))
	}

	util.ToGob(&c.items, c.file)
}

func (c *Clipboard) Clear() {
	c.items = []ClipboardItem{}
	c.entries = []util.Entry{}

	util.ToGob(&c.items, c.file)
}
