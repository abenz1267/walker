package modules

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

var (
	DmenuSocketAddrGet   = filepath.Join(util.TmpDir(), "walker-dmenu.sock")
	DmenuSocketAddrReply = filepath.Join(util.TmpDir(), "walker-dmenu-reply.sock")
)

type Dmenu struct {
	Config             config.Dmenu
	Content            []string
	initialSeparator   string
	initialLabelColumn int
	initialIconColumn  int
	initialValueColumn int
	IsService          bool
}

func (d *Dmenu) General() *config.GeneralModule {
	return &d.Config.GeneralModule
}

func (d Dmenu) Entries(term string) []util.Entry {
	entries := []util.Entry{}

	for _, v := range d.Content {
		label := v
		icon := ""
		value := ""

		// if strings.Contains(label, "\\0icon\\x1f") {
		// 	split := strings.Split(label, "\\0icon\\x1f")
		// 	label = split[0]
		// 	icon = split[1]
		// }

		split := strings.Split(v, d.Config.Separator)

		if len(split) > 1 {
			if d.Config.Icon > 0 {
				icon = split[d.Config.Icon-1]
			}

			if d.Config.Label > 0 {
				label = split[d.Config.Label-1]
			} else {
				label = split[0]
			}

			if d.Config.Value > 0 {
				value = split[d.Config.Value-1]
			} else {
				value = label
			}
		} else {
			value = v
		}

		entries = append(entries, util.Entry{
			Label: label,
			Value: value,
			Sub:   "Dmenu",
			Icon:  icon,
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

func (d Dmenu) ListenForReply() bool {
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

		res := string(b[:i])

		if res != "CNCLD" {
			fmt.Println(res)
		} else {
			return true
		}

		break
	}

	return false
}

func (d *Dmenu) Setup() bool {
	d.Config = config.Cfg.Builtins.Dmenu

	d.Config.Separator = util.TransformSeparator(d.Config.Separator)

	d.initialSeparator = d.Config.Separator
	d.initialLabelColumn = d.Config.Label

	d.Config.SwitcherOnly = true

	return true
}

func (d *Dmenu) Cleanup() {
	d.Config.Separator = d.initialSeparator
	d.Config.Label = d.initialLabelColumn
	d.Config.Icon = d.initialIconColumn
	d.Config.Value = d.initialValueColumn
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

func (d *Dmenu) SetupData() {
	if config.Cfg.IsService {
		go d.StartListening()
		d.IsService = true
	}

	d.Config.IsSetup = true
	d.Config.HasInitialSetup = true

	if !d.IsService {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			d.Content = append(d.Content, scanner.Text())
		}
	}
}

func (d *Dmenu) Refresh() {
	d.Config.IsSetup = !d.Config.Refresh
}
