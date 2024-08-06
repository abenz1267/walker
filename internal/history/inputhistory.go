package history

import (
	"path/filepath"

	"github.com/abenz1267/walker/internal/util"
)

type InputHistory map[string][]string

var inputhstry InputHistory

func SaveInputHistory(module string, input string) {
	if inputhstry == nil {
		inputhstry = make(InputHistory)
	}

	if _, ok := inputhstry[module]; !ok {
		inputhstry[module] = []string{}
	}

	inputhstry[module] = append([]string{input}, inputhstry[module]...)

	util.ToGob(&inputhstry, filepath.Join(util.CacheDir(), "inputhistory_0.3.8.gob"))
}

func GetInputHistory(module string) []string {
	if inputhstry != nil {
		return inputhstry[module]
	}

	file := filepath.Join(util.CacheDir(), "inputhistory_0.3.8.gob")

	if inputhstry == nil {
		inputhstry = make(InputHistory)
	}

	_ = util.FromGob(file, &inputhstry)

	return inputhstry[module]
}
