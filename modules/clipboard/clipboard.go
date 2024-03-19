package clipboard

import (
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
	prefix   string
	entries  []Entry
	file     string
	imgTypes map[string]string
	max      int
}

type Entry struct {
	Content string    `json:"content,omitempty"`
	Time    time.Time `json:"time,omitempty"`
	Hash    string    `json:"hash,omitempty"`
	IsImg   bool      `json:"is_img,omitempty"`
}

func (c Clipboard) Entries(term string) []modules.Entry {
	entries := []modules.Entry{}

	es := []Entry{}

	util.FromJson(c.file, &es)

	for _, v := range es {
		e := modules.Entry{
			Label:      v.Content,
			Sub:        "Text",
			Exec:       fmt.Sprintf("wl-copy %s", v.Content),
			Categories: []string{"clipboard"},
			Class:      "clipboard",
			Matching:   modules.Fuzzy,
			LastUsed:   v.Time,
		}

		if v.IsImg {
			e.Label = "Image"
			e.Image = v.Content
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
	c.file = filepath.Join(util.CacheDir(), "clipboard.json")
	c.max = cfg.Clipboard.MaxEntries

	c.imgTypes = make(map[string]string)
	c.imgTypes["image/png"] = "png"
	c.imgTypes["image/jpg"] = "jpg"
	c.imgTypes["image/jpeg"] = "jpeg"

	current := []Entry{}
	util.FromJson(c.file, &current)

	c.entries = clean(current, c.file)

	go c.watch()

	return c
}

func clean(entries []Entry, file string) []Entry {
	cleaned := []Entry{}

	for _, v := range entries {
		if !v.IsImg {
			cleaned = append(cleaned, v)
			continue
		}

		if _, err := os.Stat(v.Content); err == nil {
			cleaned = append(cleaned, v)
		}
	}

	util.ToJson(cleaned, file)

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
		log.Println(string(out))
		log.Panic(err)
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

		e := Entry{
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

		c.entries = append([]Entry{e}, c.entries...)

		if len(c.entries) >= c.max {
			c.entries = slices.Clone(c.entries[:c.max])
		}

		util.ToJson(c.entries, c.file)
	}
}
