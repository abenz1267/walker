package util

import (
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func FuzzyScore(input, target string) (float64, *[]int, int) {
	chars := util.ToChars([]byte(target))
	res, pos := algo.FuzzyMatchV2(false, true, true, &chars, []rune(strings.ToLower(input)), true, nil)

	return float64(res.Score), pos, res.Start
}

func ExactScore(input, target string) (float64, *[]int, int) {
	chars := util.ToChars([]byte(target))
	res, pos := algo.ExactMatchNaive(false, true, true, &chars, []rune(strings.ToLower(input)), true, nil)

	return float64(res.Score), pos, res.Start
}
