package util

import (
	"slices"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func FuzzyScore(input, target string) (float64, *[]int, int) {
	runes := []rune(input)
	chars := util.ToChars([]byte(target))
	res, pos := algo.FuzzyMatchV2(slices.ContainsFunc(runes, unicode.IsUpper), true, true, &chars, runes, true, nil)

	return float64(res.Score), pos, res.Start
}

func ExactScore(input, target string) (float64, *[]int, int) {
	runes := []rune(input)
	chars := util.ToChars([]byte(target))
	res, pos := algo.ExactMatchNaive(slices.ContainsFunc(runes, unicode.IsUpper), true, true, &chars, runes, true, nil)

	return float64(res.Score), pos, res.Start
}
