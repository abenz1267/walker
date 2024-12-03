package clipboard

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

const ClipboardName = "clipboard"

var ClipboardSocketAddrUpdate = filepath.Join(util.TmpDir(), "walker-clipboard-update.sock")

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

func (c Clipboard) Entries(ctx context.Context, term string) []util.Entry {
	return c.entries
}

func (c *Clipboard) Setup(cfg *config.Config) bool {
	pth, _ := exec.LookPath("wl-copy")
	if pth == "" {
		log.Println("Clipboard disabled: currently wl-clipboard only.")
		return false
	}

	c.general = cfg.Builtins.Clipboard.GeneralModule

	c.file = filepath.Join(util.CacheDir(), "clipboard.gob")
	c.max = cfg.Builtins.Clipboard.MaxEntries
	c.exec = cfg.Builtins.Clipboard.Exec
	c.avoidLineBreaks = cfg.Builtins.Clipboard.AvoidLineBreaks

	c.imgTypes = make(map[string]string)
	c.imgTypes["image/png"] = "png"
	c.imgTypes["image/jpg"] = "jpg"
	c.imgTypes["image/jpeg"] = "jpeg"

	return true
}

func (c *Clipboard) SetupData(cfg *config.Config, ctx context.Context) {
	current := []ClipboardItem{}
	_ = util.FromGob(c.file, &current)

	go c.watch()

	c.items = clean(current, c.file)

	for _, v := range c.items {
		c.entries = append(c.entries, itemToEntry(v, c.exec, c.avoidLineBreaks))
	}

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

	os.Remove(ClipboardSocketAddrUpdate)

	go func() {
		time.Sleep(time.Second * 5)

		cmd := exec.Command("sh", "-c", "wl-paste --watch walker --update-clipboard")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid:    true,
			Pgid:       0,
			Foreground: false,
		}

		_ = cmd.Run()
	}()

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: ClipboardSocketAddrUpdate})
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			slog.Error("clipboard", "error", "accept", err)
		}

		b := make([]byte, 104_857_600)

		i, err := conn.Read(b)
		if err != nil {
			if err.Error() == "EOF" {
				continue
			} else {
				log.Panic(err)
			}
			continue
		}

		content := string(b[:i])

		hash := md5.Sum([]byte(content))
		strgHash := hex.EncodeToString(hash[:])

		if c.exists(strgHash) {
			continue
		}

		if len(content) < 2 {
			continue
		}

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

		if len(c.items) >= c.max {
			c.items = slices.Clone(c.items[:c.max])
		}

		if len(c.entries) >= c.max {
			c.entries = slices.Clone(c.entries[:c.max])
		}

		util.ToGob(&c.items, c.file)
	}
}

func Update(content []byte) {
	if !util.FileExists(ClipboardSocketAddrUpdate) {
		return
	}

	conn, err := net.Dial("unix", ClipboardSocketAddrUpdate)
	if err != nil {
		slog.Error("clipboard", "error", "socket", err)
		return
	}

	_, err = conn.Write(content)
	if err != nil {
		slog.Error("clipboard", "error", "write", err)
	}
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
	}

	return entry
}

func (c *Clipboard) Delete(entry util.Entry) {
	content := entry.Piped.String

	c.entries = []util.Entry{}

	for k, v := range c.items {
		if v.Content == content {
			c.items = slices.Delete(c.items, k, k+1)
			continue
		}
	}

	for _, v := range c.items {
		c.entries = append(c.entries, itemToEntry(v, c.exec, c.avoidLineBreaks))
	}

	util.ToGob(&c.items, c.file)
}
