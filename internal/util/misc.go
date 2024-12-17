package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type MatchingType int

const (
	Fuzzy MatchingType = iota
	AlwaysTop
	AlwaysBottom
	AlwaysTopOnEmptySearch
)

type Entry struct {
	Categories       []string     `mapstructure:"categories,omitempty" json:"categories,omitempty"`
	Class            string       `mapstructure:"class,omitempty" json:"class,omitempty"`
	DragDrop         bool         `mapstructure:"drag_drop,omitempty" json:"drag_drop,omitempty"`
	DragDropData     string       `mapstructure:"drag_drop_data,omitempty" json:"drag_drop_data,omitempty"`
	Exec             string       `mapstructure:"exec,omitempty" json:"exec,omitempty"`
	ExecAlt          string       `mapstructure:"exec_alt,omitempty" json:"exec_alt,omitempty"`
	HideText         bool         `mapstructure:"hide_text,omitempty" json:"hide_text,omitempty"`
	Icon             string       `mapstructure:"icon,omitempty" json:"icon,omitempty"`
	Image            string       `mapstructure:"image,omitempty" json:"image,omitempty"`
	InitialClass     string       `mapstructure:"initial_class,omitempty" json:"initial_class,omitempty"`
	Label            string       `mapstructure:"label,omitempty" json:"label,omitempty"`
	MatchedLabel     string       `mapstructure:"matched_label,omitempty" json:"matched_label,omitempty"`
	MatchFields      int          `mapstructure:"match_fields,omitempty" json:"match_fields,omitempty"`
	Matching         MatchingType `mapstructure:"matching,omitempty" json:"matching,omitempty"`
	Path             string       `mapstructure:"path,omitempty" json:"path,omitempty"`
	RecalculateScore bool         `mapstructure:"recalculate_score,omitempty" json:"recalculate_score,omitempty"`
	ScoreFinal       float64      `mapstructure:"score_final,omitempty" json:"score_final,omitempty"`
	ScoreFuzzy       float64      `mapstructure:"score_fuzzy,omitempty" json:"score_fuzzy,omitempty"`
	Searchable       string       `mapstructure:"searchable,omitempty" json:"searchable,omitempty"`
	Searchable2      string       `mapstructure:"searchable2,omitempty" json:"searchable2,omitempty"`
	Sub              string       `mapstructure:"sub,omitempty" json:"sub,omitempty"`
	MatchedSub       string       `mapstructure:"matched_sub,omitempty" json:"matched_sub,omitempty"`
	Terminal         bool         `mapstructure:"terminal,omitempty" json:"terminal,omitempty"`
	Prefer           bool         `mapstructure:"prefer,omitempty" json:"prefer,omitempty"`

	// internal
	DaysSinceUsed    int                       `mapstructure:"-"`
	File             string                    `mapstructure:"-"`
	History          bool                      `mapstructure:"-"`
	IgnoreUnprefixed bool                      `mapstructure:"-"`
	IsAction         bool                      `mapstructure:"-"`
	LastUsed         time.Time                 `mapstructure:"-"`
	Module           string                    `mapstructure:"-"`
	OpenWindows      uint                      `mapstructure:"-"`
	Piped            Piped                     `mapstructure:"-"`
	PipedAlt         Piped                     `mapstructure:"-"`
	Prefix           string                    `mapstructure:"-"`
	SingleModuleOnly bool                      `mapstructure:"-"`
	SpecialFunc      func(args ...interface{}) `mapstructure:"-"`
	SpecialFuncArgs  []interface{}             `mapstructure:"-"`
	Used             int                       `mapstructure:"-"`
	Weight           int                       `mapstructure:"-"`
}

func (e Entry) Identifier() string {
	str := fmt.Sprintf("%s %s %s %s", e.Label, e.Sub, e.Searchable, strings.Join(e.Categories, " "))

	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

type Piped struct {
	Bytes  []byte `mapstructure:"bytes,omitempty"`
	String string `mapstructure:"content,omitempty"`
	Type   string `mapstructure:"type,omitempty"`
}

func TransformSeparator(sep string) string {
	if sep == "" {
		sep = "'\t'"
	}

	s, err := strconv.Unquote(sep)
	if err != nil {
		return sep
	}

	return s
}
