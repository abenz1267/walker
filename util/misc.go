package util

type MatchingType int

const (
	Fuzzy MatchingType = iota
	AlwaysTop
	AlwaysBottom
)
