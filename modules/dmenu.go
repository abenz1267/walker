package modules

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

var (
	DmenuSocketAddrGet   = filepath.Join(util.TmpDir(), "walker-dmenu.sock")
	DmenuSocketAddrReply = filepath.Join(util.TmpDir(), "walker-dmenu-reply.sock")
)

type Dmenu struct {
	general            config.GeneralModule
	isSetup            bool
	Content            []string
	Separator          string
	LabelColumn        int
	initialSeparator   string
	initialLabelColumn int
	IsService          bool
}

func (d *Dmenu) General() *config.GeneralModule {
	return &d.general
}

func (d Dmenu) Entries(ctx context.Context, term string) []util.Entry {
	entries := []util.Entry{}

	for _, v := range d.Content {
		label := v

		if d.LabelColumn > 0 {
			split := strings.Split(v, d.Separator)

			if len(split) >= d.LabelColumn {
				label = split[d.LabelColumn-1]
			}
		}

		entries = append(entries, util.Entry{
			Label: label,
			Sub:   "Dmenu",
			Exec:  v,
		})
	}

	return entries
}

func (Dmenu) Reply(res string) {
	if !util.FileExists(DmenuSocketAddrReply) {
		return
	}

	conn, err := net.Dial("unix", DmenuSocketAddrReply)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(res))
	if err != nil {
		log.Println(err)
	}
}

func (d Dmenu) ListenForReply() {
	os.Remove(DmenuSocketAddrReply)

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: DmenuSocketAddrReply})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			log.Panic(err)
		}

		b := make([]byte, 1_048_576)
		i, err := conn.Read(b)
		if err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				log.Panic(err)
			}
			continue
		}

		fmt.Print(string(b[:i]))
		break
	}
}

func (d *Dmenu) Setup(cfg *config.Config) bool {
	d.general = cfg.Builtins.Dmenu.GeneralModule

	d.SetSeparator(cfg.Builtins.Dmenu.Separator)
	d.LabelColumn = cfg.Builtins.Dmenu.LabelColumn

	d.initialSeparator = d.Separator
	d.initialLabelColumn = d.LabelColumn

	d.general.SwitcherOnly = true

	return true
}

func (d *Dmenu) Cleanup() {
	d.Separator = d.initialSeparator
	d.LabelColumn = d.initialLabelColumn
}

func (d *Dmenu) SetSeparator(sep string) {
	if sep == "" {
		sep = "'\t'"
	}

	s, err := strconv.Unquote(sep)
	if err == nil {
		d.Separator = s
	}
}

func (d *Dmenu) StartListening() {
	os.Remove(DmenuSocketAddrGet)

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: DmenuSocketAddrGet})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			log.Panic(err)
		}

		b := make([]byte, 1_048_576)
		i, err := conn.Read(b)
		if err != nil {
			log.Println(err)
			continue
		}

		res := strings.Split(string(b[:i]), "\n")

		d.Content = []string{}

		for _, v := range res {
			if v != "" {
				d.Content = append(d.Content, v)
			}
		}
	}
}

func (d Dmenu) Send() {
	conn, err := net.Dial("unix", DmenuSocketAddrGet)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		msg := fmt.Sprintf("%s\n", scanner.Text())

		_, err = conn.Write([]byte(msg))
		if err != nil {
			log.Panic(err)
		}
	}
}

func (d *Dmenu) SetupData(cfg *config.Config, ctx context.Context) {
	if cfg.IsService {
		go d.StartListening()
		d.IsService = true
	}

	d.isSetup = true
	d.general.HasInitialSetup = true

	if !d.IsService {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			d.Content = append(d.Content, scanner.Text())
		}
	}
}

func (d *Dmenu) Refresh() {
	d.general.IsSetup = !d.general.Refresh
}
