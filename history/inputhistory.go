package history

import (
	"path/filepath"
	"slices"

	"github.com/abenz1267/walker/util"
)

type InputHistory []string

func (h InputHistory) SaveToInputHistory(input string) {
	i := slices.Index(h, input)

	if i != -1 {
		h = append(h[:i], h[i+1:]...)
	}

	h = append([]string{input}, h...)

	if len(h) > 50 {
		h = h[:50]
	}

	util.ToGob(&h, filepath.Join(util.CacheDir(), "inputhistory.gob"))
}

func GetInputHistory() InputHistory {
	file := filepath.Join(util.CacheDir(), "inputhistory.gob")

	h := InputHistory{}
	_ = util.FromGob(file, &h)

	return h
}
