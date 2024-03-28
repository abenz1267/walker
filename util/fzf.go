package util

import (
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func FuzzyScore(input, target string) int {
	chars := util.ToChars([]byte(target))
	res, _ := algo.FuzzyMatchV2(false, true, true, &chars, []rune(input), true, nil)

	return res.Score
}
