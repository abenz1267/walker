package history

import (
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/internal/util"
)

type (
	Prefix string
	Hash   string
)

type HistoryMap map[string]map[string]*HistoryEntry

type History HistoryMap

type HistoryEntry struct {
	LastUsed      time.Time `json:"last_used,omitempty"`
	Used          int       `json:"used,omitempty"`
	DaysSinceUsed int       `json:"-"`
}

const HistoryName = "history_0.8.14.gob"

func (s *History) Has(hash string) bool {
	for _, v := range *s {
		for h := range v {
			if h == hash {
				return true
			}
		}
	}

	return false
}

func (s *History) Delete(hash string) bool {
	deleted := false

	for _, v := range *s {
		for h := range v {
			if h == hash {
				delete(v, h)
				deleted = true
			}
		}
	}

	util.ToGob(&s, filepath.Join(util.CacheDir(), HistoryName))

	return deleted
}

func (s History) Save(hash string, prefix string) {
	p, ok := s[prefix]
	if !ok {
		p = make(map[string]*HistoryEntry)
		s[prefix] = p
	}

	h, ok := p[hash]
	if !ok {
		h = &HistoryEntry{
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

	util.ToGob(&s, filepath.Join(util.CacheDir(), HistoryName))
}

func Get() History {
	file := filepath.Join(util.CacheDir(), HistoryName)

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
