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
	Categories       []string     `json:"categories,omitempty"`
	Class            string       `json:"class,omitempty"`
	DragDrop         bool         `json:"drag_drop,omitempty"`
	DragDropData     string       `json:"drag_drop_data,omitempty"`
	Exec             string       `json:"exec,omitempty"`
	ExecAlt          string       `json:"exec_alt,omitempty"`
	HideText         bool         `json:"hide_text,omitempty"`
	Icon             string       `json:"icon,omitempty"`
	IconIsImage      bool         `json:"icon_is_image,omitempty"`
	Image            string       `json:"image,omitempty"`
	InitialClass     string       `json:"initial_class,omitempty"`
	Label            string       `json:"label,omitempty"`
	MatchFields      int          `json:"match_fields,omitempty"`
	Matching         MatchingType `json:"matching,omitempty"`
	Path             string       `json:"path,omitempty"`
	RecalculateScore bool         `json:"recalculate_score,omitempty"`
	ScoreFinal       float64      `json:"score_final,omitempty"`
	ScoreFuzzy       float64      `json:"score_fuzzy,omitempty"`
	Searchable       string       `json:"searchable,omitempty"`
	SpecialLabel     string       `json:"special_label,omitempty"`
	Sub              string       `json:"sub,omitempty"`
	Terminal         bool         `json:"terminal,omitempty"`

	// internal
	DaysSinceUsed int       `json:"-"`
	History       bool      `json:"-"`
	LastUsed      time.Time `json:"-"`
	Module        string    `json:"-"`
	OpenWindows   uint      `json:"-"`
	Piped         Piped     `json:"-"`
	PipedAlt      Piped     `json:"-"`
	Used          int       `json:"-"`
}

func (e Entry) Identifier() string {
	str := fmt.Sprintf("%s %s %s %s", e.Label, e.Sub, e.Searchable, strings.Join(e.Categories, " "))

	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

type Piped struct {
	Content string `json:"content,omitempty"`
	Type    string `json:"type,omitempty"`
}
