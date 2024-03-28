package history

import (
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/util"
)

type (
	Prefix string
	Hash   string
)

type HistoryMap map[string]map[string]HistoryEntry

type History HistoryMap

type HistoryEntry struct {
	LastUsed      time.Time `json:"last_used,omitempty"`
	Used          int       `json:"used,omitempty"`
	DaysSinceUsed int       `json:"-"`
}

func (s History) Save(hash string, prefix string) {
	p, ok := s[prefix]
	if !ok {
		p = make(map[string]HistoryEntry)
		s[prefix] = p
	}

	h, ok := p[hash]
	if !ok {
		h = HistoryEntry{
			LastUsed: time.Now(),
			Used:     1,
		}
	} else {
		h.Used++

		if h.Used > 10 {
			h.Used = 10
		}

		h.LastUsed = time.Now()
	}

	p[hash] = h

	util.ToGob(&s, filepath.Join(util.CacheDir(), "history.gob"))
}

func Get() History {
	file := filepath.Join(util.CacheDir(), "history.gob")

	history := History{}
	_ = util.FromGob(file, &history)

	for _, v := range history {
		for _, vv := range v {
			today := time.Now()
			vv.DaysSinceUsed = int(today.Sub(vv.LastUsed).Hours() / 24)
		}
	}

	return history
}
