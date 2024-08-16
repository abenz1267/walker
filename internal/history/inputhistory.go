package history

import (
	"path/filepath"

	"github.com/abenz1267/walker/internal/util"
)

type InputHistoryItem struct {
	Term       string
	Identifier string
}

type InputHistory map[string][]InputHistoryItem

var inputhstry InputHistory

func SaveInputHistory(module string, input string, identifier string) {
	if inputhstry == nil {
		inputhstry = make(InputHistory)
	}

	if _, ok := inputhstry[module]; !ok {
		inputhstry[module] = []InputHistoryItem{}
	}

	n := InputHistoryItem{
		Term:       input,
		Identifier: identifier,
	}

	inputhstry[module] = append([]InputHistoryItem{n}, inputhstry[module]...)

	util.ToGob(&inputhstry, filepath.Join(util.CacheDir(), "inputhistory_0.7.6.gob"))
}

func GetInputHistory(module string) []InputHistoryItem {
	if inputhstry != nil {
		return inputhstry[module]
	}

	file := filepath.Join(util.CacheDir(), "inputhistory_0.7.6.gob")

	if inputhstry == nil {
		inputhstry = make(InputHistory)
	}

	_ = util.FromGob(file, &inputhstry)

	return inputhstry[module]
}
