package modules

import (
	"strings"
	"sync"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Dmenu struct {
	Config        config.Dmenu
	Separator     string
	IconColum     int
	LabelColumn   int
	ValueColumn   int
	DmenuShowChan chan bool
	mut           sync.Mutex
	entries       []*util.Entry
}

func (d *Dmenu) General() *config.GeneralModule {
	return &d.Config.GeneralModule
}

func (d *Dmenu) Entries(term string) []*util.Entry {
	return d.entries
}

func (d *Dmenu) Setup() bool {
	d.Config = config.Cfg.Builtins.Dmenu
	d.Separator = util.TransformSeparator(d.Separator)
	d.Config.IsSetup = true
	d.Config.HasInitialSetup = true

	return true
}

func (d *Dmenu) Cleanup() {
	d.IconColum = 0
	d.LabelColumn = 0
	d.ValueColumn = 0
	d.entries = []*util.Entry{}
}

func (d *Dmenu) SetupData() {
}

func (d *Dmenu) Refresh() {
}

func (d *Dmenu) Append(in string) {
	d.mut.Lock()
	d.entries = append(d.entries, d.LineToEntry(in))
	d.mut.Unlock()
}

func (d *Dmenu) LineToEntry(in string) *util.Entry {
	label := in
	icon := ""
	value := in

	if strings.Contains(in, d.Separator) {
		split := strings.Split(in, d.Separator)
		label = split[0]
		value = label

		if d.IconColum > 0 {
			icon = split[d.IconColum]
		}

		if d.LabelColumn > 0 {
			label = split[d.LabelColumn]
		}

		if d.ValueColumn > 0 {
			value = split[d.ValueColumn]
		}
	}

	return &util.Entry{
		Label: label,
		Icon:  icon,
		Sub:   "Dmenu",
		Value: value,
	}
}
