package util

import (
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func FuzzyScore(input, target string) float64 {
	chars := util.ToChars([]byte(target))
	res, _ := algo.FuzzyMatchV2(false, true, true, &chars, []rune(strings.ToLower(input)), true, nil)

	return float64(res.Score - res.Start)
}

func ExactScore(input, target string) float64 {
	input = strings.TrimPrefix(input, "'")

	chars := util.ToChars([]byte(target))
	res, _ := algo.ExactMatchNaive(false, true, true, &chars, []rune(strings.ToLower(input)), true, nil)

	return float64(res.Score - res.Start)
}
