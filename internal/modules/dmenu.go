package modules

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

var content []string

type Dmenu struct {
	Config             config.Dmenu
	initialSeparator   string
	initialLabelColumn int
	initialIconColumn  int
	initialValueColumn int
	IsService          bool
	DmenuShowChan      chan bool
	mut                sync.Mutex
}

func (d *Dmenu) General() *config.GeneralModule {
	return &d.Config.GeneralModule
}

func (d *Dmenu) Entries(term string) []*util.Entry {
	entries := make([]*util.Entry, 0, len(content))

	for _, v := range content {
		label := v
		icon := ""
		value := ""

		if strings.ContainsRune(label, '\x00') && strings.ContainsRune(label, '\x1f') {
			split := strings.Split(label, "\x00")
			label = split[0]
			value = label

			split = strings.Split(v, "\x1f")
			icon = split[1]
		} else {

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
		}

		entries = append(entries, &util.Entry{
			Label: label,
			Value: value,
			Sub:   "Dmenu",
			Icon:  icon,
		})
	}

	return entries
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
	content = []string{}
}

func (d *Dmenu) SetupData() {
	if config.Cfg.IsService {
		d.IsService = true
	}

	d.Config.IsSetup = true
	d.Config.HasInitialSetup = true

	if !d.IsService {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			content = append(content, scanner.Text())
		}
	}
}

func (d *Dmenu) Refresh() {
	d.Config.IsSetup = !d.Config.Refresh
}

func (d *Dmenu) Append(in string) {
	d.mut.Lock()
	content = append(content, in)
	d.mut.Unlock()
}
