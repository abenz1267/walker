package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
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
	Categories       []string     `mapstructure:"categories,omitempty"`
	Class            string       `mapstructure:"class,omitempty"`
	DragDrop         bool         `mapstructure:"drag_drop,omitempty"`
	DragDropData     string       `mapstructure:"drag_drop_data,omitempty"`
	Exec             string       `mapstructure:"exec,omitempty"`
	ExecAlt          string       `mapstructure:"exec_alt,omitempty"`
	HideText         bool         `mapstructure:"hide_text,omitempty"`
	Icon             string       `mapstructure:"icon,omitempty"`
	IconIsImage      bool         `mapstructure:"icon_is_image,omitempty"`
	Image            string       `mapstructure:"image,omitempty"`
	InitialClass     string       `mapstructure:"initial_class,omitempty"`
	Label            string       `mapstructure:"label,omitempty"`
	MatchFields      int          `mapstructure:"match_fields,omitempty"`
	Matching         MatchingType `mapstructure:"matching,omitempty"`
	Path             string       `mapstructure:"path,omitempty"`
	RecalculateScore bool         `mapstructure:"recalculate_score,omitempty"`
	ScoreFinal       float64      `mapstructure:"score_final,omitempty"`
	ScoreFuzzy       float64      `mapstructure:"score_fuzzy,omitempty"`
	Searchable       string       `mapstructure:"searchable,omitempty"`
	SpecialLabel     string       `mapstructure:"special_label,omitempty"`
	Sub              string       `mapstructure:"sub,omitempty"`
	Terminal         bool         `mapstructure:"terminal,omitempty"`

	// internal
	DaysSinceUsed int       `mapstructure:"-"`
	History       bool      `mapstructure:"-"`
	LastUsed      time.Time `mapstructure:"-"`
	Module        string    `mapstructure:"-"`
	OpenWindows   uint      `mapstructure:"-"`
	Piped         Piped     `mapstructure:"-"`
	PipedAlt      Piped     `mapstructure:"-"`
	Used          int       `mapstructure:"-"`
}

func (e Entry) Identifier() string {
	str := fmt.Sprintf("%s %s %s %s", e.Label, e.Sub, e.Searchable, strings.Join(e.Categories, " "))

	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

type Piped struct {
	Content string `mapstructure:"content,omitempty"`
	Type    string `mapstructure:"type,omitempty"`
}
