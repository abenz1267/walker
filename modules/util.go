package modules

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
)

type Workable interface {
	Entries(ctx context.Context, term string) []Entry
	Prefix() string
	Name() string
	SwitcherExclusive() bool
	Setup(cfg *config.Config, config *config.Module) Workable
	Refresh()
}

type MatchingType int

const (
	Fuzzy MatchingType = iota
	AlwaysTop
	AlwaysBottom
)

type Entry struct {
	Label            string       `json:"label,omitempty"`
	Sub              string       `json:"sub,omitempty"`
	Exec             string       `json:"exec,omitempty"`
	RawExec          []string     `json:"raw_exec,omitempty"`
	Terminal         bool         `json:"terminal,omitempty"`
	Piped            Piped        `json:"piped,omitempty"`
	Icon             string       `json:"icon,omitempty"`
	IconIsImage      bool         `json:"icon_is_image,omitempty"`
	DragDrop         bool         `json:"drag_drop,omitempty"`
	DragDropData     string       `json:"drag_drop_data,omitempty"`
	Image            string       `json:"image,omitempty"`
	HideText         bool         `json:"hide_text,omitempty"`
	Categories       []string     `json:"categories,omitempty"`
	Searchable       string       `json:"searchable,omitempty"`
	MatchFields      int          `json:"match_fields,omitempty"`
	Class            string       `json:"class,omitempty"`
	History          bool         `json:"history,omitempty"`
	Matching         MatchingType `json:"matching,omitempty"`
	RecalculateScore bool         `json:"recalculate_score,omitempty"`
	ScoreFinal       float64      `json:"score_final,omitempty"`
	ScoreFuzzy       float64      `json:"score_fuzzy,omitempty"`
	Used             int          `json:"-"`
	DaysSinceUsed    int          `json:"-"`
	SpecialLabel     string       `json:"special_label,omitempty"`
	LastUsed         time.Time    `json:"-"`
	InitialClass     string       `json:"initial_class,omitempty"`
	OpenWindows      uint         `json:"-"`
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

func readCache(name string, data any) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		return false
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	path := filepath.Join(cacheDir, fmt.Sprintf("%s.json", name))

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Panicln(err)
		}

		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Panicln(err)
		}

		return true
	}

	return false
}

func Find(modules []config.Module, name string) *config.Module {
	for _, v := range modules {
		if v.Name == name {
			return &v
		}
	}

	return nil
}
