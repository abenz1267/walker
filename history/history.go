package history

import (
	"path/filepath"
	"time"

	"github.com/abenz1267/walker/util"
)

type History map[string]Entry

type Entry struct {
	LastUsed      time.Time `json:"last_used,omitempty"`
	Used          int       `json:"used,omitempty"`
	DaysSinceUsed int
}

func (s History) Save(entry string) {
	h, ok := s[entry]
	if !ok {
		h = Entry{
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

	s[entry] = h

	util.ToJson(s, filepath.Join(util.CacheDir(), "history.json"))
}

func Get() History {
	file := filepath.Join(util.CacheDir(), "history.json")

	history := History{}
	_ = util.FromJson(file, &history)

	for k, v := range history {
		today := time.Now()
		v.DaysSinceUsed = int(today.Sub(v.LastUsed).Hours() / 24)

		history[k] = v
	}

	return history
}
